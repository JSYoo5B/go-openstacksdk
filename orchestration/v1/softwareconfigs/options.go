package softwareconfigs

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/internal/swcommon"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// CreateOpts holds the declared Python SoftwareConfig body attributes. A zero
// Optional is omitted; request.Present sends the value, including empty
// strings and false-like JSON; request.Null sends an explicit JSON null.
type CreateOpts struct {
	Name    request.Optional[string]          `json:"name,omitzero"`
	Group   request.Optional[string]          `json:"group,omitzero"`
	Config  request.Optional[json.RawMessage] `json:"config,omitzero"`
	Inputs  request.Optional[json.RawMessage] `json:"inputs,omitzero"`
	Outputs request.Optional[json.RawMessage] `json:"outputs,omitzero"`
	Options request.Optional[json.RawMessage] `json:"options,omitzero"`
}

// ListOpts holds Heat's list controls. Python's default query mapping accepts
// the same limit and marker.
type ListOpts struct {
	Limit  *int    `json:"limit,omitempty"`
	Marker *string `json:"marker,omitempty"`
}

type CreateOption = request.Option[CreateOpts]
type ListOption = request.Option[ListOpts]

// WithCreateOptions snapshots a complete option struct.
func WithCreateOptions(opts CreateOpts) CreateOption { return swcommon.Snapshot(opts) }
func WithListOptions(opts ListOpts) ListOption       { return swcommon.Snapshot(opts) }

// WithCreateField adds an additional body member. Declared members and
// response-only members (id, creation_time) are protected.
func WithCreateField(key string, value any) CreateOption {
	return request.WithField[CreateOpts](key, value)
}
func WithCreateHeader(key, value string) CreateOption {
	return request.WithHeader[CreateOpts](key, value)
}
func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

func WithCreateName(value string) CreateOption {
	return func(c *request.Config[CreateOpts]) error { c.Options.Name = request.Present(value); return nil }
}
func WithCreateGroup(value string) CreateOption {
	return func(c *request.Config[CreateOpts]) error { c.Options.Group = request.Present(value); return nil }
}

// WithCreateConfig sends any JSON-encodable value, such as a script string or
// a mapping. The value is encoded immediately.
func WithCreateConfig(value any) CreateOption {
	return rawOption(value, func(c *request.Config[CreateOpts], v json.RawMessage) { c.Options.Config = request.Present(v) })
}
func WithCreateInputs(value any) CreateOption {
	return rawOption(value, func(c *request.Config[CreateOpts], v json.RawMessage) { c.Options.Inputs = request.Present(v) })
}
func WithCreateOutputs(value any) CreateOption {
	return rawOption(value, func(c *request.Config[CreateOpts], v json.RawMessage) { c.Options.Outputs = request.Present(v) })
}

// WithCreateConfigOptions sets the tool-specific "options" member.
func WithCreateConfigOptions(value any) CreateOption {
	return rawOption(value, func(c *request.Config[CreateOpts], v json.RawMessage) { c.Options.Options = request.Present(v) })
}

func WithListLimit(limit int) ListOption {
	return func(c *request.Config[ListOpts]) error { c.Options.Limit = &limit; return nil }
}
func WithListMarker(marker string) ListOption {
	return func(c *request.Config[ListOpts]) error { c.Options.Marker = &marker; return nil }
}

func rawOption(value any, set func(*request.Config[CreateOpts], json.RawMessage)) CreateOption {
	encoded, err := swcommon.RawJSON("software config member", value)
	return func(c *request.Config[CreateOpts]) error {
		if err != nil {
			return err
		}
		set(c, append(json.RawMessage(nil), encoded...))
		return nil
	}
}
