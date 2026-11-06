package nodes

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/senlin"
	"github.com/JSYoo5B/go-openstacksdk/request"
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

// AdoptOption is one option of Adopt: WithAdoptOptions, the WithAdopt field
// options, request.Header, request.Field or request.Custom over
// *request.Config[AdoptOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type AdoptOption = request.WriteCall

var adoptCalls = senlin.WriteCalls[AdoptOpts]()

// AdoptPreviewOption is one option of AdoptPreview: WithAdoptPreviewOptions,
// the WithAdoptPreview field options, request.Header or request.Custom over
// *request.Config[AdoptPreviewOpts]. Preview never took extension fields, so
// request.Field and the other shared calls fail with resource.ErrInvalidOption
// before HTTP.
type AdoptPreviewOption = request.WriteCall

var adoptPreviewCalls = senlin.HeaderCalls[AdoptPreviewOpts]()

func WithAdoptOptions(value AdoptOpts) AdoptOption { return request.Custom(senlin.Snapshot(value)) }
func WithAdoptName(value string) AdoptOption {
	return request.Custom(func(config *request.Config[AdoptOpts]) error {
		config.Options.Name = request.Present(value)
		return nil
	})
}
func WithAdoptNameNull() AdoptOption {
	return request.Custom(func(config *request.Config[AdoptOpts]) error {
		config.Options.Name = request.Null[string]()
		return nil
	})
}
func WithAdoptRole(value string) AdoptOption {
	return request.Custom(func(config *request.Config[AdoptOpts]) error {
		config.Options.Role = request.Present(value)
		return nil
	})
}
func WithAdoptRoleNull() AdoptOption {
	return request.Custom(func(config *request.Config[AdoptOpts]) error {
		config.Options.Role = request.Null[string]()
		return nil
	})
}
func WithAdoptSnapshot(value bool) AdoptOption {
	return request.Custom(func(config *request.Config[AdoptOpts]) error {
		config.Options.Snapshot = request.Present(value)
		return nil
	})
}
func WithAdoptSnapshotNull() AdoptOption {
	return request.Custom(func(config *request.Config[AdoptOpts]) error {
		config.Options.Snapshot = request.Null[bool]()
		return nil
	})
}
func WithAdoptMetadata(value any) AdoptOption {
	return request.Custom(withJSON(value, func(opts *AdoptOpts, raw json.RawMessage) { opts.Metadata = raw }))
}
func WithAdoptOverrides(value any) AdoptOption {
	return request.Custom(withJSON(value, func(opts *AdoptOpts, raw json.RawMessage) { opts.Overrides = raw }))
}

func WithAdoptPreviewOptions(value AdoptPreviewOpts) AdoptPreviewOption {
	return request.Custom(senlin.Snapshot(value))
}
func WithAdoptPreviewOverrides(value any) AdoptPreviewOption {
	return request.Custom(withJSON(value, func(opts *AdoptPreviewOpts, raw json.RawMessage) { opts.Overrides = raw }))
}
func WithAdoptPreviewSnapshot(value bool) AdoptPreviewOption {
	return request.Custom(func(config *request.Config[AdoptPreviewOpts]) error {
		config.Options.Snapshot = request.Present(value)
		return nil
	})
}
func WithAdoptPreviewSnapshotNull() AdoptPreviewOption {
	return request.Custom(func(config *request.Config[AdoptPreviewOpts]) error {
		config.Options.Snapshot = request.Null[bool]()
		return nil
	})
}
