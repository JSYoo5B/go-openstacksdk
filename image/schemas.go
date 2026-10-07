package image

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
)

// GetImagesSchema reads the fixed image collection schema once.
func (s *Service) GetImagesSchema(ctx context.Context, options ...GetSchemaOption) (*Schema, error) {
	return s.getSchema(ctx, "GetImagesSchema", "schemas/images", options)
}

// GetImageSchema reads the fixed single-image schema once.
func (s *Service) GetImageSchema(ctx context.Context, options ...GetSchemaOption) (*Schema, error) {
	return s.getSchema(ctx, "GetImageSchema", "schemas/image", options)
}

// GetMembersSchema reads the fixed image member collection schema once.
func (s *Service) GetMembersSchema(ctx context.Context, options ...GetSchemaOption) (*Schema, error) {
	return s.getSchema(ctx, "GetMembersSchema", "schemas/members", options)
}

// GetMemberSchema reads the fixed single image member schema once.
func (s *Service) GetMemberSchema(ctx context.Context, options ...GetSchemaOption) (*Schema, error) {
	return s.getSchema(ctx, "GetMemberSchema", "schemas/member", options)
}

// GetTasksSchema reads the fixed task collection schema once.
func (s *Service) GetTasksSchema(ctx context.Context, options ...GetSchemaOption) (*Schema, error) {
	return s.getSchema(ctx, "GetTasksSchema", "schemas/tasks", options)
}

// GetTaskSchema reads the fixed single-task schema once.
func (s *Service) GetTaskSchema(ctx context.Context, options ...GetSchemaOption) (*Schema, error) {
	return s.getSchema(ctx, "GetTaskSchema", "schemas/task", options)
}

// GetMetadefNamespaceSchema reads the metadata definition namespace schema.
func (s *Service) GetMetadefNamespaceSchema(ctx context.Context, options ...GetSchemaOption) (*Schema, error) {
	return s.getSchema(ctx, "GetMetadefNamespaceSchema", "schemas/metadefs/namespace", options)
}

// GetMetadefNamespacesSchema reads the metadata definition namespaces schema.
func (s *Service) GetMetadefNamespacesSchema(ctx context.Context, options ...GetSchemaOption) (*Schema, error) {
	return s.getSchema(ctx, "GetMetadefNamespacesSchema", "schemas/metadefs/namespaces", options)
}

// GetMetadefResourceTypeSchema reads the resource type association schema.
func (s *Service) GetMetadefResourceTypeSchema(ctx context.Context, options ...GetSchemaOption) (*Schema, error) {
	return s.getSchema(ctx, "GetMetadefResourceTypeSchema", "schemas/metadefs/resource_type", options)
}

// GetMetadefResourceTypesSchema reads the resource type associations schema.
func (s *Service) GetMetadefResourceTypesSchema(ctx context.Context, options ...GetSchemaOption) (*Schema, error) {
	return s.getSchema(ctx, "GetMetadefResourceTypesSchema", "schemas/metadefs/resource_types", options)
}

// GetMetadefObjectSchema reads the metadata definition object schema.
func (s *Service) GetMetadefObjectSchema(ctx context.Context, options ...GetSchemaOption) (*Schema, error) {
	return s.getSchema(ctx, "GetMetadefObjectSchema", "schemas/metadefs/object", options)
}

// GetMetadefObjectsSchema reads the metadata definition objects schema.
func (s *Service) GetMetadefObjectsSchema(ctx context.Context, options ...GetSchemaOption) (*Schema, error) {
	return s.getSchema(ctx, "GetMetadefObjectsSchema", "schemas/metadefs/objects", options)
}

// GetMetadefPropertySchema reads the metadata definition property schema.
func (s *Service) GetMetadefPropertySchema(ctx context.Context, options ...GetSchemaOption) (*Schema, error) {
	return s.getSchema(ctx, "GetMetadefPropertySchema", "schemas/metadefs/property", options)
}

// GetMetadefPropertiesSchema reads the metadata definition properties schema.
func (s *Service) GetMetadefPropertiesSchema(ctx context.Context, options ...GetSchemaOption) (*Schema, error) {
	return s.getSchema(ctx, "GetMetadefPropertiesSchema", "schemas/metadefs/properties", options)
}

// GetMetadefTagSchema reads the metadata definition tag schema.
func (s *Service) GetMetadefTagSchema(ctx context.Context, options ...GetSchemaOption) (*Schema, error) {
	return s.getSchema(ctx, "GetMetadefTagSchema", "schemas/metadefs/tag", options)
}

// GetMetadefTagsSchema reads the metadata definition tags schema.
func (s *Service) GetMetadefTagsSchema(ctx context.Context, options ...GetSchemaOption) (*Schema, error) {
	return s.getSchema(ctx, "GetMetadefTagsSchema", "schemas/metadefs/tags", options)
}

func (s *Service) getSchema(ctx context.Context, operation, path string, options []GetSchemaOption) (*Schema, error) {
	prepared, err := s.prepareSchema(ctx, options)
	if err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	response, err := rest.DoJSON(ctx, prepared.client, http.MethodGet, prepared.base+path, nil, nil, http.StatusOK)
	if err != nil {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	var value Schema
	err = json.Unmarshal(response.Body, &value)
	if err == nil {
		if err = ctx.Err(); err != nil && !errors.Is(err, context.Cause(ctx)) {
			err = errors.Join(err, context.Cause(ctx))
		}
	}
	if err != nil {
		return nil, wrapImageMutationError(ctx, operation, response.Fail(err))
	}
	value.Header = response.Header.Clone()
	value.StatusCode = response.StatusCode
	return &value, nil
}
