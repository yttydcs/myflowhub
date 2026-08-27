package notification

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/command"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

type Config struct {
	Node *node.Node
	Now  func() time.Time
}

type Controller struct {
	node   *node.Node
	now    func() time.Time
	events *resource.Stream
}

func Register(config Config) (*Controller, error) {
	if config.Node == nil {
		return nil, errors.New("notification node is required")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	owner := config.Node.ID()
	events, err := resource.NewStream(resource.Descriptor{
		ID: protocol.ResourceID{Owner: owner, Name: protocol.BuiltinNotificationEvents}, Kind: resource.KindStream,
		ContentType: "application/json", Schema: protocol.SchemaNotificationEventV1, Permission: "notification.read", MaxValueBytes: protocol.DefaultMaxPayload,
	})
	if err != nil {
		return nil, err
	}
	value := &Controller{node: config.Node, now: config.Now, events: events}
	publish, err := resource.NewCommand(resource.Descriptor{
		ID: protocol.ResourceID{Owner: owner, Name: protocol.BuiltinNotificationPublish}, Kind: resource.KindCommand,
		ContentType: "application/json", Schema: protocol.SchemaNotificationPublishV1, Permission: "notification.publish", MaxValueBytes: protocol.DefaultMaxPayload,
	}, value.publish)
	if err != nil {
		return nil, err
	}
	registered := make([]protocol.ResourceID, 0, 2)
	for _, current := range []resource.Resource{events, publish} {
		if err := config.Node.Registry().Register(current); err != nil {
			for index := len(registered) - 1; index >= 0; index-- {
				_ = config.Node.Registry().Remove(registered[index])
			}
			return nil, fmt.Errorf("register notification resource %s: %w", current.Descriptor().ID.Name, err)
		}
		registered = append(registered, current.Descriptor().ID)
	}
	return value, nil
}

func (c *Controller) publish(ctx context.Context, input []byte) ([]byte, error) {
	delegation, ok := command.DelegationFromContext(ctx)
	if !ok {
		return nil, errors.New("notification publish requires an authenticated command context")
	}
	source, ok := delegation.Subject()
	if !ok {
		return nil, errors.New("notification publish subject is unavailable")
	}
	var request protocol.NotificationPublishV1
	if err := protocol.DecodeJSONPayload(input, protocol.DefaultMaxPayload, &request); err != nil {
		return nil, err
	}
	id, err := protocol.NewMessageID()
	if err != nil {
		return nil, err
	}
	event := protocol.NotificationEventV1{
		Version: 1, EventID: id.String(), Channel: request.Channel, SourceNodeID: strconv.FormatUint(uint64(source), 10),
		CreatedAtUnixMS: c.now().UTC().UnixMilli(), ContentType: request.ContentType, Body: append([]byte(nil), request.Body...), Attributes: cloneAttributes(request.Attributes),
	}
	payload, err := protocol.EncodeJSONPayload(&event, protocol.DefaultMaxPayload)
	if err != nil {
		return nil, err
	}
	if _, err := c.events.Publish(payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func cloneAttributes(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
