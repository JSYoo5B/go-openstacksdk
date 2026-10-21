package hosts

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/masakari"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// UpdateAttributes is the Resource-style attribute set of update_host. Each
// field is a request.Optional: omitted by default, JSON null with
// request.Null, and exact false/empty values with request.Present.
// ControlAttributes is arbitrary JSON because the controller owns its schema;
// the SDK does not validate it.
type UpdateAttributes struct {
	Name              request.Optional[string]          `json:"name,omitzero"`
	Type              request.Optional[string]          `json:"type,omitzero"`
	ControlAttributes request.Optional[json.RawMessage] `json:"control_attributes,omitzero"`
	OnMaintenance     request.Optional[bool]            `json:"on_maintenance,omitzero"`
	Reserved          request.Optional[bool]            `json:"reserved,omitzero"`
}

type AttributeOption = request.Option[UpdateAttributes]

func WithAttributeField(key string, value any) AttributeOption {
	return request.WithField[UpdateAttributes](key, value)
}

func WithAttributeHeader(key, value string) AttributeOption {
	return request.WithHeader[UpdateAttributes](key, value)
}

// UpdateAttributes PUTs {"host": <attributes>} to the resolved host UUID in
// the fixed segment and requires 200. Only supplied attributes are sent. A
// request without any attribute is rejected before HTTP, whereas Python skips
// the PUT when nothing is dirty.
func (s *SegmentScope) UpdateAttributes(ctx context.Context, ref resource.Ref, attrs UpdateAttributes, options ...AttributeOption) (*Host, error) {
	const op = "UpdateAttributes"
	if err := s.validate(ctx); err != nil {
		return nil, request.Wrap(op, "hosts", err)
	}
	config, err := request.Apply(attrs, append([]AttributeOption(nil), options...)...)
	if err != nil {
		return nil, request.Wrap(op, "hosts", err)
	}
	body, err := masakari.Body(config, "host", "id", "uuid", "segment", "segment_id", "failover_segment_id", "failover_segment", "created_at", "updated_at", "deleted", "deleted_at")
	if err != nil {
		return nil, request.Wrap(op, "hosts", err)
	}
	id, err := s.Resources.ResolveID(ctx, ref)
	if err != nil {
		return nil, request.Wrap(op, "hosts", err)
	}
	response, err := rest.DoJSON(ctx, s.client, "PUT", s.client.ServiceURL(s.spec.Path, url.PathEscape(id)), body, config.Headers, 200)
	if err != nil {
		return nil, request.Wrap(op, "hosts", err)
	}
	value, err := rest.Decode(response, "host", s.spec.Metadata)
	return value, request.Wrap(op, "hosts", err)
}
