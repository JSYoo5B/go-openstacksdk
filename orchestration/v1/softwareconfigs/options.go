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

// CreateOption is one option of Create: WithCreateOptions, the WithCreate*
// member setters, request.Field (an additional body member; declared members
// and response-only members such as id and creation_time are protected),
// request.Header or request.Custom over *request.Config[CreateOpts]. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

// ListOption is one option of List and All: WithListOptions, request.Limit,
// request.Marker, request.Query with a string or []string value (it cannot
// replace limit or marker) or request.Custom over *request.Config[ListOpts].
// Other shared calls fail with resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

var (
	createCalls = swcommon.WriteCalls[CreateOpts]()
	listCalls   = swcommon.ListCalls(func(o *ListOpts) **int { return &o.Limit }, func(o *ListOpts) **string { return &o.Marker })
)

// WithCreateOptions snapshots a complete option struct.
func WithCreateOptions(opts CreateOpts) CreateOption { return request.Custom(swcommon.Snapshot(opts)) }
func WithListOptions(opts ListOpts) ListOption       { return request.Custom(swcommon.Snapshot(opts)) }

func WithCreateName(value string) CreateOption {
	return request.Custom(func(c *request.Config[CreateOpts]) error { c.Options.Name = request.Present(value); return nil })
}
func WithCreateGroup(value string) CreateOption {
	return request.Custom(func(c *request.Config[CreateOpts]) error { c.Options.Group = request.Present(value); return nil })
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

func rawOption(value any, set func(*request.Config[CreateOpts], json.RawMessage)) CreateOption {
	encoded, err := swcommon.RawJSON("software config member", value)
	return request.Custom(func(c *request.Config[CreateOpts]) error {
		if err != nil {
			return err
		}
		set(c, append(json.RawMessage(nil), encoded...))
		return nil
	})
}
