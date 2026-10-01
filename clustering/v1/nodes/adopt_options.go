package nodes

import (
	"encoding/json"

	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
)

// AdoptOpts identifies a physical resource in the body rather than the URL.
// Unspecified attributes stay omitted. Explicit null values remain distinct;
// the deployment validates their field and profile-specific semantics.
type AdoptOpts struct {
	Identity  string                   `json:"identity"`
	Type      string                   `json:"type"`
	Name      request.Optional[string] `json:"name,omitzero"`
	Role      request.Optional[string] `json:"role,omitzero"`
	Snapshot  request.Optional[bool]   `json:"snapshot,omitzero"`
	Metadata  json.RawMessage          `json:"metadata,omitempty"`
	Overrides json.RawMessage          `json:"overrides,omitempty"`
}

// AdoptPreviewOpts contains exactly the four attributes forwarded by pinned
// Python preview. Omitted Overrides and Snapshot are serialized as null.
type AdoptPreviewOpts struct {
	Identity  string                 `json:"identity"`
	Type      string                 `json:"type"`
	Overrides json.RawMessage        `json:"overrides"`
	Snapshot  request.Optional[bool] `json:"snapshot"`
}

type AdoptOption = request.Option[AdoptOpts]
type AdoptPreviewOption = request.Option[AdoptPreviewOpts]

func WithAdoptOptions(value AdoptOpts) AdoptOption { return senlin.Snapshot(value) }
func WithAdoptName(value string) AdoptOption {
	return func(config *request.Config[AdoptOpts]) error {
		config.Options.Name = request.Present(value)
		return nil
	}
}
func WithAdoptNameNull() AdoptOption {
	return func(config *request.Config[AdoptOpts]) error {
		config.Options.Name = request.Null[string]()
		return nil
	}
}
func WithAdoptRole(value string) AdoptOption {
	return func(config *request.Config[AdoptOpts]) error {
		config.Options.Role = request.Present(value)
		return nil
	}
}
func WithAdoptRoleNull() AdoptOption {
	return func(config *request.Config[AdoptOpts]) error {
		config.Options.Role = request.Null[string]()
		return nil
	}
}
func WithAdoptSnapshot(value bool) AdoptOption {
	return func(config *request.Config[AdoptOpts]) error {
		config.Options.Snapshot = request.Present(value)
		return nil
	}
}
func WithAdoptSnapshotNull() AdoptOption {
	return func(config *request.Config[AdoptOpts]) error {
		config.Options.Snapshot = request.Null[bool]()
		return nil
	}
}
func WithAdoptMetadata(value any) AdoptOption {
	return withJSON(value, func(opts *AdoptOpts, raw json.RawMessage) { opts.Metadata = raw })
}
func WithAdoptOverrides(value any) AdoptOption {
	return withJSON(value, func(opts *AdoptOpts, raw json.RawMessage) { opts.Overrides = raw })
}
func WithAdoptField(key string, value any) AdoptOption {
	return request.WithField[AdoptOpts](key, value)
}
func WithAdoptHeader(key, value string) AdoptOption { return request.WithHeader[AdoptOpts](key, value) }

func WithAdoptPreviewOptions(value AdoptPreviewOpts) AdoptPreviewOption {
	return senlin.Snapshot(value)
}
func WithAdoptPreviewOverrides(value any) AdoptPreviewOption {
	return withJSON(value, func(opts *AdoptPreviewOpts, raw json.RawMessage) { opts.Overrides = raw })
}
func WithAdoptPreviewSnapshot(value bool) AdoptPreviewOption {
	return func(config *request.Config[AdoptPreviewOpts]) error {
		config.Options.Snapshot = request.Present(value)
		return nil
	}
}
func WithAdoptPreviewSnapshotNull() AdoptPreviewOption {
	return func(config *request.Config[AdoptPreviewOpts]) error {
		config.Options.Snapshot = request.Null[bool]()
		return nil
	}
}
func WithAdoptPreviewHeader(key, value string) AdoptPreviewOption {
	return request.WithHeader[AdoptPreviewOpts](key, value)
}
