package clipboard

import "github.com/yttydcs/myflowhub/protocol"

// These definitions travel with the variables; consumers need no Clipboard
// schema registration. Domain Validate methods retain the relational checks.
func configDefinition() protocol.DataSchemaDefinition {
	peer := protocol.DataSchemaDefinition{Type: "object", Properties: []protocol.DataSchemaProperty{
		variableProperty("node_id", true, protocol.DataSchemaDefinition{Type: "string", Format: "node-id"}),
		variableProperty("receive", true, protocol.DataSchemaDefinition{Type: "boolean"}),
	}}
	maxPeers := protocol.MaxItems
	return protocol.DataSchemaDefinition{ID: SchemaConfigV1, Type: "object", Title: "Clipboard configuration", Properties: []protocol.DataSchemaProperty{
		variableProperty("version", true, variableInteger(1, 1)),
		variableProperty("revision", true, variableInteger(1, 1<<53-1)),
		variableProperty("enabled", true, protocol.DataSchemaDefinition{Type: "boolean"}),
		variableProperty("max_inline_bytes", true, variableInteger(1, MaxTextBytes)),
		variableProperty("auto_watch", true, protocol.DataSchemaDefinition{Type: "boolean"}),
		variableProperty("auto_apply", true, protocol.DataSchemaDefinition{Type: "boolean"}),
		variableProperty("history_retention", true, protocol.DataSchemaDefinition{Type: "string", Enum: []string{"body", "metadata", "none"}}),
		variableProperty("history_limit", true, variableInteger(1, MaxHistoryEntries)),
		variableProperty("history_max_bytes", true, variableInteger(1, MaxHistoryBytes)),
		variableProperty("history_ttl_ms", true, variableInteger(1, float64(MaxHistoryTTLMS))),
		variableProperty("peers", true, protocol.DataSchemaDefinition{Type: "array", Items: &peer, MaxItems: &maxPeers}),
	}}
}

func statusDefinition() protocol.DataSchemaDefinition {
	properties := []protocol.DataSchemaProperty{
		variableProperty("version", true, variableInteger(1, 1)),
		variableProperty("config_revision", true, variableInteger(1, 1<<53-1)),
		variableProperty("enabled", true, protocol.DataSchemaDefinition{Type: "boolean"}),
		variableProperty("auto_watch", true, protocol.DataSchemaDefinition{Type: "boolean"}),
		variableProperty("auto_apply", true, protocol.DataSchemaDefinition{Type: "boolean"}),
		variableProperty("peer_count", true, variableInteger(0, protocol.MaxItems)),
		variableProperty("pending_count", true, variableInteger(0, MaxPendingEvents)),
		variableProperty("pending_size_bytes", false, variableInteger(0, MaxTextBytes)),
		variableProperty("history_count", true, variableInteger(0, MaxHistoryEntries)),
		variableProperty("history_body_bytes", true, variableInteger(0, MaxHistoryBytes)),
		variableProperty("last_size_bytes", false, variableInteger(0, 1<<53-1)),
		variableProperty("observed_at_unix_ms", true, variableInteger(1, 1<<53-1)),
	}
	for _, field := range []struct {
		name string
		max  int
	}{
		{"pending_event_id", 32}, {"pending_hash_prefix", 12}, {"last_action", 128},
		{"last_event_id", 32}, {"last_hash_prefix", 12}, {"last_error", 1024},
	} {
		properties = append(properties, variableProperty(field.name, false, protocol.DataSchemaDefinition{Type: "string", MaxLength: &field.max}))
	}
	return protocol.DataSchemaDefinition{ID: SchemaStatusV1, Type: "object", Title: "Clipboard status", Properties: properties}
}

func variableProperty(name string, required bool, definition protocol.DataSchemaDefinition) protocol.DataSchemaProperty {
	return protocol.DataSchemaProperty{Name: name, Required: required, Schema: &definition}
}

func variableInteger(minimum, maximum float64) protocol.DataSchemaDefinition {
	return protocol.DataSchemaDefinition{Type: "integer", Minimum: &minimum, Maximum: &maximum}
}
