package notification_test

import (
	"context"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub/feature/notification"
	"github.com/yttydcs/myflowhub/protocol"
	"github.com/yttydcs/myflowhub/runtime/auth"
	"github.com/yttydcs/myflowhub/runtime/node"
	"github.com/yttydcs/myflowhub/runtime/resource"
)

func TestPublishCreatesOwnedStreamEvent(t *testing.T) {
	identity, _ := auth.GenerateIdentity(1)
	trust := auth.NewTrustStore()
	_ = trust.Add(1, identity.PublicKey)
	runtime, err := node.New(context.Background(), node.Config{Identity: identity, Trust: trust})
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	if _, err := notification.Register(notification.Config{Node: runtime, Now: func() time.Time { return time.UnixMilli(1234) }}); err != nil {
		t.Fatal(err)
	}
	eventsValue, _ := runtime.Registry().Resolve(protocol.ResourceID{Owner: 1, Name: protocol.BuiltinNotificationEvents})
	events := make(chan resource.StreamEvent, 1)
	cancel, _ := eventsValue.(*resource.Stream).Watch(func(event resource.StreamEvent) { events <- event })
	defer cancel()
	request := protocol.NotificationPublishV1{Version: 1, Channel: "system", ContentType: "text/plain", Body: []byte("hello")}
	payload, _ := protocol.EncodeJSONPayload(&request, protocol.DefaultMaxPayload)
	output, err := runtime.Invoke(context.Background(), protocol.ResourceID{Owner: 1, Name: protocol.BuiltinNotificationPublish}, payload)
	if err != nil {
		t.Fatal(err)
	}
	var response protocol.NotificationEventV1
	if err := protocol.DecodeJSONPayload(output, protocol.DefaultMaxPayload, &response); err != nil {
		t.Fatal(err)
	}
	if response.SourceNodeID != "1" || response.CreatedAtUnixMS != 1234 || string(response.Body) != "hello" {
		t.Fatalf("unexpected event: %#v", response)
	}
	select {
	case event := <-events:
		if string(event.Value) != string(output) {
			t.Fatal("command result and stream event differ")
		}
	case <-time.After(time.Second):
		t.Fatal("notification event was not published")
	}
}
