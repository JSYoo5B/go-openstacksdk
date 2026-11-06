package softwaredeployments

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/internal/swcommon"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// CreateOpts holds the declared Python SoftwareDeployment body attributes. A
// zero Optional is omitted; request.Present sends the value, including empty
// strings and empty objects; request.Null sends an explicit JSON null.
type CreateOpts struct {
	Action             request.Optional[string]          `json:"action,omitzero"`
	ConfigID           request.Optional[string]          `json:"config_id,omitzero"`
	InputValues        request.Optional[json.RawMessage] `json:"input_values,omitzero"`
	OutputValues       request.Optional[json.RawMessage] `json:"output_values,omitzero"`
	ServerID           request.Optional[string]          `json:"server_id,omitzero"`
	StackUserProjectID request.Optional[string]          `json:"stack_user_project_id,omitzero"`
	Status             request.Optional[string]          `json:"status,omitzero"`
	StatusReason       request.Optional[string]          `json:"status_reason,omitzero"`
}

// UpdateOpts has the same members as CreateOpts. Only the members that are
// set are sent, matching Python's dirty-attribute commit.
type UpdateOpts CreateOpts

// ListOpts holds Heat's list controls. Python's default query mapping accepts
// limit and marker; ServerID is the Heat API's documented server filter.
type ListOpts struct {
	Limit    *int    `json:"limit,omitempty"`
	Marker   *string `json:"marker,omitempty"`
	ServerID *string `json:"server_id,omitempty"`
}

// CreateOption is one option of Create: WithCreateOptions, the WithCreate*
// member setters, request.Field (an additional body member; declared members
// and response-only members such as id, creation_time and updated_time are
// protected), request.Header or request.Custom over
// *request.Config[CreateOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

// UpdateOption is one option of Update: WithUpdateOptions, the WithUpdate*
// member setters, request.Field, request.Header or request.Custom over
// *request.Config[UpdateOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type UpdateOption = request.WriteCall

// ListOption is one option of List and All: WithListOptions,
// WithListServerID, request.Limit, request.Marker, request.Query with a string
// or []string value (it cannot replace limit, marker or server_id) or
// request.Custom over *request.Config[ListOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

var (
	createCalls = swcommon.WriteCalls[CreateOpts]()
	updateCalls = swcommon.WriteCalls[UpdateOpts]()
	listCalls   = swcommon.ListCalls(func(o *ListOpts) **int { return &o.Limit }, func(o *ListOpts) **string { return &o.Marker })
)

func WithCreateOptions(opts CreateOpts) CreateOption { return request.Custom(swcommon.Snapshot(opts)) }
func WithUpdateOptions(opts UpdateOpts) UpdateOption { return request.Custom(swcommon.Snapshot(opts)) }
func WithListOptions(opts ListOpts) ListOption       { return request.Custom(swcommon.Snapshot(opts)) }
func WithListServerID(serverID string) ListOption {
	return request.Custom(func(c *request.Config[ListOpts]) error { c.Options.ServerID = &serverID; return nil })
}

func WithCreateAction(v string) CreateOption {
	return request.Custom(func(c *request.Config[CreateOpts]) error { c.Options.Action = request.Present(v); return nil })
}
func WithCreateConfigID(v string) CreateOption {
	return request.Custom(func(c *request.Config[CreateOpts]) error { c.Options.ConfigID = request.Present(v); return nil })
}
func WithCreateServerID(v string) CreateOption {
	return request.Custom(func(c *request.Config[CreateOpts]) error { c.Options.ServerID = request.Present(v); return nil })
}
func WithCreateStackUserProjectID(v string) CreateOption {
	return request.Custom(func(c *request.Config[CreateOpts]) error {
		c.Options.StackUserProjectID = request.Present(v)
		return nil
	})
}
func WithCreateStatus(v string) CreateOption {
	return request.Custom(func(c *request.Config[CreateOpts]) error { c.Options.Status = request.Present(v); return nil })
}
func WithCreateStatusReason(v string) CreateOption {
	return request.Custom(func(c *request.Config[CreateOpts]) error { c.Options.StatusReason = request.Present(v); return nil })
}

// WithCreateInputValues and WithCreateOutputValues encode any JSON-encodable
// value immediately; Heat expects an object.
func WithCreateInputValues(value any) CreateOption {
	encoded, err := swcommon.RawJSON("input_values", value)
	return request.Custom(func(c *request.Config[CreateOpts]) error {
		if err != nil {
			return err
		}
		c.Options.InputValues = request.Present(json.RawMessage(append([]byte(nil), encoded...)))
		return nil
	})
}
func WithCreateOutputValues(value any) CreateOption {
	encoded, err := swcommon.RawJSON("output_values", value)
	return request.Custom(func(c *request.Config[CreateOpts]) error {
		if err != nil {
			return err
		}
		c.Options.OutputValues = request.Present(json.RawMessage(append([]byte(nil), encoded...)))
		return nil
	})
}

func WithUpdateAction(v string) UpdateOption {
	return request.Custom(func(c *request.Config[UpdateOpts]) error { c.Options.Action = request.Present(v); return nil })
}
func WithUpdateConfigID(v string) UpdateOption {
	return request.Custom(func(c *request.Config[UpdateOpts]) error { c.Options.ConfigID = request.Present(v); return nil })
}
func WithUpdateStatus(v string) UpdateOption {
	return request.Custom(func(c *request.Config[UpdateOpts]) error { c.Options.Status = request.Present(v); return nil })
}
func WithUpdateStatusReason(v string) UpdateOption {
	return request.Custom(func(c *request.Config[UpdateOpts]) error { c.Options.StatusReason = request.Present(v); return nil })
}
func WithUpdateInputValues(value any) UpdateOption {
	encoded, err := swcommon.RawJSON("input_values", value)
	return request.Custom(func(c *request.Config[UpdateOpts]) error {
		if err != nil {
			return err
		}
		c.Options.InputValues = request.Present(json.RawMessage(append([]byte(nil), encoded...)))
		return nil
	})
}
func WithUpdateOutputValues(value any) UpdateOption {
	encoded, err := swcommon.RawJSON("output_values", value)
	return request.Custom(func(c *request.Config[UpdateOpts]) error {
		if err != nil {
			return err
		}
		c.Options.OutputValues = request.Present(json.RawMessage(append([]byte(nil), encoded...)))
		return nil
	})
}
