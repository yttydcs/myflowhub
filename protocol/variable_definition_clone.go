package protocol

// CloneVariableDefinition copies a previously validated, bounded definition.
// Returned definitions must not share mutable fields with a session's cache.
func CloneVariableDefinition(definition DataSchemaDefinition) DataSchemaDefinition {
	definition.Precision = cloneVariablePointer(definition.Precision)
	definition.Minimum = cloneVariablePointer(definition.Minimum)
	definition.Maximum = cloneVariablePointer(definition.Maximum)
	definition.MultipleOf = cloneVariablePointer(definition.MultipleOf)
	definition.MinLength = cloneVariablePointer(definition.MinLength)
	definition.MaxLength = cloneVariablePointer(definition.MaxLength)
	definition.MinItems = cloneVariablePointer(definition.MinItems)
	definition.MaxItems = cloneVariablePointer(definition.MaxItems)
	definition.Enum = append([]string(nil), definition.Enum...)
	definition.Properties = append([]DataSchemaProperty(nil), definition.Properties...)
	for i := range definition.Properties {
		if definition.Properties[i].Schema != nil {
			child := CloneVariableDefinition(*definition.Properties[i].Schema)
			definition.Properties[i].Schema = &child
		}
	}
	if definition.Items != nil {
		child := CloneVariableDefinition(*definition.Items)
		definition.Items = &child
	}
	if definition.AdditionalProperties != nil {
		child := CloneVariableDefinition(*definition.AdditionalProperties)
		definition.AdditionalProperties = &child
	}
	return definition
}

func cloneVariablePointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
