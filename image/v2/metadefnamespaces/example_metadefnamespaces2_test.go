package metadefnamespaces_test

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces"
)

func createNestedNamespaceMetadefnamespaces2(ctx context.Context, service *image.Service, namespace string) (*metadefnamespaces.Namespace, error) {
	description := "A compute shape"
	prefix, target := "hw:", "metadata"
	properties := map[string]metadefnamespaces.PropertyDefinition{
		"mode": {
			Type: "string", Title: "",
			Attributes: map[string]json.RawMessage{
				"enum":    json.RawMessage(`["small","large"]`),
				"default": json.RawMessage(`"small"`),
			},
		},
	}
	objects := []metadefnamespaces.ObjectDefinition{{
		Name: "shape", Description: &description,
		Properties: map[string]metadefnamespaces.PropertyDefinition{
			"vcpus": {
				Type: "integer", Title: "VCPUs",
				Attributes: map[string]json.RawMessage{"minimum": json.RawMessage(`0`)},
			},
		},
		Required: []string{"vcpus"},
	}}
	tags := []metadefnamespaces.TagDefinition{{Name: "example"}}
	associations := []metadefnamespaces.ResourceTypeAssociationDefinition{{
		Name: "OS::Nova::Server", Prefix: &prefix, PropertiesTarget: &target,
	}}
	return service.API.MetadefNamespaces.Create(ctx, namespace,
		metadefnamespaces.WithCreateOpts(metadefnamespaces.CreateOpts{}),
		metadefnamespaces.WithCreateProperties(properties),
		metadefnamespaces.WithCreateObjects(objects),
		metadefnamespaces.WithCreateTags(tags),
		metadefnamespaces.WithCreateResourceTypeAssociations(associations))
}
