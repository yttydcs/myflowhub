package protocol

import (
	"errors"
	"fmt"
)

const (
	SchemaNotificationEventV1   = "mfh.notification.event.v1"
	SchemaNotificationPublishV1 = "mfh.notification.publish.v1"
	MaxNotificationBodyBytes    = 256 << 10
	BuiltinNotificationEvents   = "notifications/events"
	BuiltinNotificationPublish  = "notifications/publish"
)

type NotificationEventV1 struct {
	Version         int               `json:"version"`
	EventID         string            `json:"event_id"`
	Channel         string            `json:"channel"`
	SourceNodeID    string            `json:"source_node_id"`
	CreatedAtUnixMS int64             `json:"created_at_unix_ms"`
	ContentType     string            `json:"content_type"`
	Body            []byte            `json:"body"`
	Attributes      map[string]string `json:"attributes,omitempty"`
}

func (e NotificationEventV1) Validate() error {
	if err := validateVersion(e.Version); err != nil {
		return err
	}
	if err := validateHexID("event_id", e.EventID, 16); err != nil {
		return err
	}
	if err := validateText("channel", e.Channel, MaxIdentifierBytes, true); err != nil {
		return err
	}
	if err := validateNodeIDText("source_node_id", e.SourceNodeID); err != nil {
		return err
	}
	if e.CreatedAtUnixMS <= 0 {
		return errors.New("created_at_unix_ms must be positive")
	}
	if err := validateText("content_type", e.ContentType, MaxContentTypeBytes, true); err != nil {
		return err
	}
	if len(e.Body) > MaxNotificationBodyBytes {
		return fmt.Errorf("notification body exceeds %d bytes", MaxNotificationBodyBytes)
	}
	return validateAttributes(e.Attributes)
}

type NotificationPublishV1 struct {
	Version     int               `json:"version"`
	Channel     string            `json:"channel"`
	ContentType string            `json:"content_type"`
	Body        []byte            `json:"body"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

func (p NotificationPublishV1) Validate() error {
	if err := validateVersion(p.Version); err != nil {
		return err
	}
	if err := validateText("channel", p.Channel, MaxIdentifierBytes, true); err != nil {
		return err
	}
	if err := validateText("content_type", p.ContentType, MaxContentTypeBytes, true); err != nil {
		return err
	}
	if len(p.Body) > MaxNotificationBodyBytes {
		return fmt.Errorf("notification body exceeds %d bytes", MaxNotificationBodyBytes)
	}
	return validateAttributes(p.Attributes)
}
