package metadefnamespaces

import (
	"encoding/json"
	"slices"
	"unicode/utf8"
)

// PropertyDefinition is a property schema submitted within a namespace.
// Type must be nonempty. Title is always encoded, including an empty title.
// Attributes are flattened JSON keywords; the server owns their semantics.
type PropertyDefinition struct {
	Type        string
	Title       string
	Description *string
	Attributes  map[string]json.RawMessage
}

// ObjectDefinition is a named object submitted within a namespace.
type ObjectDefinition struct {
	Name        string
	Description *string
	Properties  map[string]PropertyDefinition
	Required    []string
}

// TagDefinition is a literal tag name submitted within a namespace.
type TagDefinition struct{ Name string }

// ResourceTypeAssociationDefinition associates a literal resource type with
// the namespace. Optional prefix and target values are JSON data.
type ResourceTypeAssociationDefinition struct {
	Name             string
	Prefix           *string
	PropertiesTarget *string
}

// WithCreateProperties snapshots and replaces nested property definitions.
// A nil map omits properties; a nonnil empty map sends an empty JSON object.
func WithCreateProperties(value map[string]PropertyDefinition) CreateOption {
	owned := copyPropertyDefinitions(value)
	return func(config *CreateOpts) error { config.Properties = copyPropertyDefinitions(owned); return nil }
}

// WithCreateObjects snapshots and replaces nested object definitions.
func WithCreateObjects(value []ObjectDefinition) CreateOption {
	owned := copyObjectDefinitions(value)
	return func(config *CreateOpts) error { config.Objects = copyObjectDefinitions(owned); return nil }
}

// WithCreateTags snapshots and replaces nested tags without deduplication.
func WithCreateTags(value []TagDefinition) CreateOption {
	owned := slices.Clone(value)
	return func(config *CreateOpts) error { config.Tags = slices.Clone(owned); return nil }
}

// WithCreateResourceTypeAssociations snapshots and replaces associations.
func WithCreateResourceTypeAssociations(value []ResourceTypeAssociationDefinition) CreateOption {
	owned := copyAssociationDefinitions(value)
	return func(config *CreateOpts) error {
		config.ResourceTypeAssociations = copyAssociationDefinitions(owned)
		return nil
	}
}

func copyPropertyDefinitions(value map[string]PropertyDefinition) map[string]PropertyDefinition {
	if value == nil {
		return nil
	}
	owned := make(map[string]PropertyDefinition, len(value))
	for key, definition := range value {
		definition.Description = copyPointer(definition.Description)
		if definition.Attributes != nil {
			attributes := make(map[string]json.RawMessage, len(definition.Attributes))
			for key, raw := range definition.Attributes {
				attributes[key] = slices.Clone(raw)
			}
			definition.Attributes = attributes
		}
		owned[key] = definition
	}
	return owned
}
func copyObjectDefinitions(value []ObjectDefinition) []ObjectDefinition {
	if value == nil {
		return nil
	}
	owned := make([]ObjectDefinition, len(value))
	for index, definition := range value {
		definition.Description = copyPointer(definition.Description)
		definition.Properties = copyPropertyDefinitions(definition.Properties)
		definition.Required = slices.Clone(definition.Required)
		owned[index] = definition
	}
	return owned
}
func copyAssociationDefinitions(value []ResourceTypeAssociationDefinition) []ResourceTypeAssociationDefinition {
	if value == nil {
		return nil
	}
	owned := make([]ResourceTypeAssociationDefinition, len(value))
	for index, definition := range value {
		definition.Prefix = copyPointer(definition.Prefix)
		definition.PropertiesTarget = copyPointer(definition.PropertiesTarget)
		owned[index] = definition
	}
	return owned
}
func definitionText(value string) error {
	if !utf8.ValidString(value) {
		return invalid("nested definition text must be valid UTF-8")
	}
	return nil
}
func validatePropertyDefinitions(values map[string]PropertyDefinition) error {
	for key, value := range values {
		if err := definitionText(key); err != nil {
			return err
		}
		if value.Type == "" {
			return invalid("nested property type is required")
		}
		for _, text := range []string{value.Type, value.Title} {
			if err := definitionText(text); err != nil {
				return err
			}
		}
		if value.Description != nil {
			if err := definitionText(*value.Description); err != nil {
				return err
			}
		}
		for key, raw := range value.Attributes {
			if !utf8.ValidString(key) || !utf8.Valid(raw) || !json.Valid(raw) {
				return invalid("nested attribute %q must be valid UTF-8 JSON", key)
			}
			switch key {
			case "name", "type", "title", "description", "self", "schema", "created_at", "updated_at", "namespace_name":
				return invalid("nested attribute %q is owned by the SDK", key)
			}
		}
	}
	return nil
}
func validateDefinitions(value CreateOpts) error {
	if err := validatePropertyDefinitions(value.Properties); err != nil {
		return err
	}
	for _, definition := range value.Objects {
		if err := definitionText(definition.Name); err != nil {
			return err
		}
		if definition.Description != nil {
			if err := definitionText(*definition.Description); err != nil {
				return err
			}
		}
		if err := validatePropertyDefinitions(definition.Properties); err != nil {
			return err
		}
		for _, name := range definition.Required {
			if err := definitionText(name); err != nil {
				return err
			}
		}
	}
	for _, definition := range value.Tags {
		if err := definitionText(definition.Name); err != nil {
			return err
		}
	}
	for _, definition := range value.ResourceTypeAssociations {
		if err := definitionText(definition.Name); err != nil {
			return err
		}
		for _, text := range []*string{definition.Prefix, definition.PropertiesTarget} {
			if text != nil {
				if err := definitionText(*text); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
func propertyDefinitionsBody(values map[string]PropertyDefinition) map[string]any {
	body := make(map[string]any, len(values))
	for name, value := range values {
		property := map[string]any{"type": value.Type, "title": value.Title}
		if value.Description != nil {
			property["description"] = *value.Description
		}
		for key, raw := range value.Attributes {
			property[key] = raw
		}
		body[name] = property
	}
	return body
}
func createBody(namespace string, value CreateOpts) map[string]any {
	body := scalarBody(namespace, value.DisplayName, value.Description, value.Visibility, value.Owner, value.Protected)
	if value.Properties != nil {
		body["properties"] = propertyDefinitionsBody(value.Properties)
	}
	if value.Objects != nil {
		objects := make([]map[string]any, 0, len(value.Objects))
		for _, value := range value.Objects {
			object := map[string]any{"name": value.Name}
			if value.Description != nil {
				object["description"] = *value.Description
			}
			if value.Properties != nil {
				object["properties"] = propertyDefinitionsBody(value.Properties)
			}
			if value.Required != nil {
				object["required"] = value.Required
			}
			objects = append(objects, object)
		}
		body["objects"] = objects
	}
	if value.Tags != nil {
		tags := make([]map[string]string, 0, len(value.Tags))
		for _, value := range value.Tags {
			tags = append(tags, map[string]string{"name": value.Name})
		}
		body["tags"] = tags
	}
	if value.ResourceTypeAssociations != nil {
		associations := make([]map[string]any, 0, len(value.ResourceTypeAssociations))
		for _, value := range value.ResourceTypeAssociations {
			association := map[string]any{"name": value.Name}
			if value.Prefix != nil {
				association["prefix"] = *value.Prefix
			}
			if value.PropertiesTarget != nil {
				association["properties_target"] = *value.PropertiesTarget
			}
			associations = append(associations, association)
		}
		body["resource_type_associations"] = associations
	}
	return body
}
