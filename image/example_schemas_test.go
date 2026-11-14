package image_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func getAllSchemas(ctx context.Context, service *image.Service) (map[string]*image.Schema, error) {
	options := []image.GetSchemaOption{
		image.WithGetSchemaOpts(image.GetSchemaOpts{
			Headers: map[string]string{"X-Request-Source": "example"},
		}),
		request.Header("X-Schema-Request", "discovery"),
		image.WithGetSchemaHeaders(map[string]string{"X-Client-Workflow": "schemas"}),
	}
	getters := []struct {
		path string
		get  func(context.Context, ...image.GetSchemaOption) (*image.Schema, error)
	}{
		{"schemas/images", service.GetImagesSchema},
		{"schemas/image", service.GetImageSchema},
		{"schemas/members", service.GetMembersSchema},
		{"schemas/member", service.GetMemberSchema},
		{"schemas/tasks", service.GetTasksSchema},
		{"schemas/task", service.GetTaskSchema},
		{"schemas/metadefs/namespace", service.GetMetadefNamespaceSchema},
		{"schemas/metadefs/namespaces", service.GetMetadefNamespacesSchema},
		{"schemas/metadefs/resource_type", service.GetMetadefResourceTypeSchema},
		{"schemas/metadefs/resource_types", service.GetMetadefResourceTypesSchema},
		{"schemas/metadefs/object", service.GetMetadefObjectSchema},
		{"schemas/metadefs/objects", service.GetMetadefObjectsSchema},
		{"schemas/metadefs/property", service.GetMetadefPropertySchema},
		{"schemas/metadefs/properties", service.GetMetadefPropertiesSchema},
		{"schemas/metadefs/tag", service.GetMetadefTagSchema},
		{"schemas/metadefs/tags", service.GetMetadefTagsSchema},
	}
	results := make(map[string]*image.Schema, len(getters))
	for _, getter := range getters {
		schema, err := getter.get(ctx, options...)
		if err != nil {
			return results, err
		}
		results[getter.path] = schema
	}
	return results, nil
}
