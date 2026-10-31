package containers

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// CreateContainerOpts owns ordinary headers and custom metadata for a PUT.
type CreateContainerOpts struct {
	Headers  map[string]string
	Metadata map[string]string
}

// CreateContainerOption is one option of CreateContainer: the
// WithCreateContainer* options, request.Header (validated and merged like a
// one-entry WithCreateContainerHeaders) or request.Custom over
// *CreateContainerOpts. Other shared calls, including request.Headers (the
// WithCreateContainerHeaders map merges instead of replacing), fail with
// resource.ErrInvalidOption before HTTP.
type CreateContainerOption = request.WriteCall

var createContainerCalls = calls.Spec[CreateContainerOpts]{Header: calls.MergeHeader[CreateContainerOpts](WithCreateContainerHeaders)}

// DeleteContainerOpts owns ordinary headers and an optional missing policy.
// A nil IgnoreMissing defaults to true.
type DeleteContainerOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}

// DeleteContainerOption is one option of DeleteContainer: the
// WithDeleteContainer* options, request.Header (validated and merged like a
// one-entry WithDeleteContainerHeaders), request.IgnoreMissing or
// request.Custom over *DeleteContainerOpts. Other shared calls, including
// request.Headers (the WithDeleteContainerHeaders map merges instead of
// replacing), fail with resource.ErrInvalidOption before HTTP.
type DeleteContainerOption = request.DeleteCall

var deleteContainerCalls = calls.Spec[DeleteContainerOpts]{
	Header:        calls.MergeHeader[DeleteContainerOpts](WithDeleteContainerHeaders),
	IgnoreMissing: calls.Pointer(func(cfg *DeleteContainerOpts) **bool { return &cfg.IgnoreMissing }),
}

func WithCreateContainerOpts(value CreateContainerOpts) CreateContainerOption {
	snapshot := cloneCreateContainerOpts(value)
	return request.Custom(func(cfg *CreateContainerOpts) error { *cfg = cloneCreateContainerOpts(snapshot); return nil })
}
func WithCreateContainerHeaders(values map[string]string) CreateContainerOption {
	snapshot := cloneMetadataHeaders(values)
	return request.Custom(func(cfg *CreateContainerOpts) error { return mergeMetadataHeaders(cfg.Headers, snapshot) })
}

// WithCreateContainerMetadata replaces the complete custom metadata input.
func WithCreateContainerMetadata(values map[string]string) CreateContainerOption {
	snapshot := cloneMetadataHeaders(values)
	return request.Custom(func(cfg *CreateContainerOpts) error { cfg.Metadata = cloneMetadataHeaders(snapshot); return nil })
}
func WithDeleteContainerOpts(value DeleteContainerOpts) DeleteContainerOption {
	snapshot := cloneDeleteContainerOpts(value)
	return request.Custom(func(cfg *DeleteContainerOpts) error { *cfg = cloneDeleteContainerOpts(snapshot); return nil })
}
func WithDeleteContainerHeaders(values map[string]string) DeleteContainerOption {
	snapshot := cloneMetadataHeaders(values)
	return request.Custom(func(cfg *DeleteContainerOpts) error { return mergeMetadataHeaders(cfg.Headers, snapshot) })
}

func cloneCreateContainerOpts(value CreateContainerOpts) CreateContainerOpts {
	value.Headers = cloneMetadataHeaders(value.Headers)
	value.Metadata = cloneMetadataHeaders(value.Metadata)
	return value
}
func cloneDeleteContainerOpts(value DeleteContainerOpts) DeleteContainerOpts {
	value.Headers = cloneMetadataHeaders(value.Headers)
	if value.IgnoreMissing != nil {
		owned := *value.IgnoreMissing
		value.IgnoreMissing = &owned
	}
	return value
}
func applyCreateContainerOptions(options []CreateContainerOption) (CreateContainerOpts, error) {
	cfg := cloneCreateContainerOpts(CreateContainerOpts{})
	for _, option := range calls.Translate(options, createContainerCalls) {
		if option == nil {
			return cfg, metadataInvalid("nil create container option")
		}
		callback := cloneCreateContainerOpts(cfg)
		err := option(&callback)
		cfg = cloneCreateContainerOpts(callback)
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
func applyDeleteContainerOptions(options []DeleteContainerOption) (DeleteContainerOpts, error) {
	cfg := cloneDeleteContainerOpts(DeleteContainerOpts{})
	for _, option := range calls.Translate(options, deleteContainerCalls) {
		if option == nil {
			return cfg, metadataInvalid("nil delete container option")
		}
		callback := cloneDeleteContainerOpts(cfg)
		err := option(&callback)
		cfg = cloneDeleteContainerOpts(callback)
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
