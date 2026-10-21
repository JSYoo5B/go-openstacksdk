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

type CreateOption = request.Option[CreateOpts]
type UpdateOption = request.Option[UpdateOpts]
type ListOption = request.Option[ListOpts]

func WithCreateOptions(opts CreateOpts) CreateOption { return swcommon.Snapshot(opts) }
func WithUpdateOptions(opts UpdateOpts) UpdateOption { return swcommon.Snapshot(opts) }
func WithListOptions(opts ListOpts) ListOption       { return swcommon.Snapshot(opts) }

// WithCreateField and WithUpdateField add additional body members. Declared
// members and response-only members (id, creation_time, updated_time) are
// protected.
func WithCreateField(key string, value any) CreateOption {
	return request.WithField[CreateOpts](key, value)
}
func WithUpdateField(key string, value any) UpdateOption {
	return request.WithField[UpdateOpts](key, value)
}
func WithCreateHeader(key, value string) CreateOption {
	return request.WithHeader[CreateOpts](key, value)
}
func WithUpdateHeader(key, value string) UpdateOption {
	return request.WithHeader[UpdateOpts](key, value)
}
func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

func WithListLimit(limit int) ListOption {
	return func(c *request.Config[ListOpts]) error { c.Options.Limit = &limit; return nil }
}
func WithListMarker(marker string) ListOption {
	return func(c *request.Config[ListOpts]) error { c.Options.Marker = &marker; return nil }
}
func WithListServerID(serverID string) ListOption {
	return func(c *request.Config[ListOpts]) error { c.Options.ServerID = &serverID; return nil }
}

func WithCreateAction(v string) CreateOption {
	return func(c *request.Config[CreateOpts]) error { c.Options.Action = request.Present(v); return nil }
}
func WithCreateConfigID(v string) CreateOption {
	return func(c *request.Config[CreateOpts]) error { c.Options.ConfigID = request.Present(v); return nil }
}
func WithCreateServerID(v string) CreateOption {
	return func(c *request.Config[CreateOpts]) error { c.Options.ServerID = request.Present(v); return nil }
}
func WithCreateStackUserProjectID(v string) CreateOption {
	return func(c *request.Config[CreateOpts]) error {
		c.Options.StackUserProjectID = request.Present(v)
		return nil
	}
}
func WithCreateStatus(v string) CreateOption {
	return func(c *request.Config[CreateOpts]) error { c.Options.Status = request.Present(v); return nil }
}
func WithCreateStatusReason(v string) CreateOption {
	return func(c *request.Config[CreateOpts]) error { c.Options.StatusReason = request.Present(v); return nil }
}

// WithCreateInputValues and WithCreateOutputValues encode any JSON-encodable
// value immediately; Heat expects an object.
func WithCreateInputValues(value any) CreateOption {
	encoded, err := swcommon.RawJSON("input_values", value)
	return func(c *request.Config[CreateOpts]) error {
		if err != nil {
			return err
		}
		c.Options.InputValues = request.Present(json.RawMessage(append([]byte(nil), encoded...)))
		return nil
	}
}
func WithCreateOutputValues(value any) CreateOption {
	encoded, err := swcommon.RawJSON("output_values", value)
	return func(c *request.Config[CreateOpts]) error {
		if err != nil {
			return err
		}
		c.Options.OutputValues = request.Present(json.RawMessage(append([]byte(nil), encoded...)))
		return nil
	}
}

func WithUpdateAction(v string) UpdateOption {
	return func(c *request.Config[UpdateOpts]) error { c.Options.Action = request.Present(v); return nil }
}
func WithUpdateConfigID(v string) UpdateOption {
	return func(c *request.Config[UpdateOpts]) error { c.Options.ConfigID = request.Present(v); return nil }
}
func WithUpdateStatus(v string) UpdateOption {
	return func(c *request.Config[UpdateOpts]) error { c.Options.Status = request.Present(v); return nil }
}
func WithUpdateStatusReason(v string) UpdateOption {
	return func(c *request.Config[UpdateOpts]) error { c.Options.StatusReason = request.Present(v); return nil }
}
func WithUpdateInputValues(value any) UpdateOption {
	encoded, err := swcommon.RawJSON("input_values", value)
	return func(c *request.Config[UpdateOpts]) error {
		if err != nil {
			return err
		}
		c.Options.InputValues = request.Present(json.RawMessage(append([]byte(nil), encoded...)))
		return nil
	}
}
func WithUpdateOutputValues(value any) UpdateOption {
	encoded, err := swcommon.RawJSON("output_values", value)
	return func(c *request.Config[UpdateOpts]) error {
		if err != nil {
			return err
		}
		c.Options.OutputValues = request.Present(json.RawMessage(append([]byte(nil), encoded...)))
		return nil
	}
}
