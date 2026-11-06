package segments

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/masakari"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Attributes is the Resource-style attribute set of create_segment and
// update_segment. Every field is a request.Optional: the zero value is omitted,
// request.Null sends JSON null (Python sends description=None as null) and
// request.Present keeps false, zero and empty strings. Unlike CreateOpts there
// are no required fields and no client-side emptiness checks; the controller
// validates the payload.
type Attributes struct {
	Name           request.Optional[string] `json:"name,omitzero"`
	Description    request.Optional[string] `json:"description,omitzero"`
	RecoveryMethod request.Optional[string] `json:"recovery_method,omitzero"`
	ServiceType    request.Optional[string] `json:"service_type,omitzero"`
	Enabled        request.Optional[bool]   `json:"enabled,omitzero"`
}

// AttributeOption is one option of CreateAttributes and UpdateAttributes:
// request.Field (an extension member that is not a concrete attribute or a
// protected identity/timestamp field), request.Header or request.Custom over
// *request.Config[Attributes]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type AttributeOption = request.WriteCall

var attributeCalls = masakari.WriteCalls[Attributes]()

var attributeProtected = []string{"id", "uuid", "created_at", "updated_at", "deleted_at", "deleted"}

func (a *API) attributeBody(ctx context.Context, attrs Attributes, options []AttributeOption) (request.Config[Attributes], json.RawMessage, error) {
	if err := masakari.Require(ctx, a.client, 0); err != nil {
		return request.Config[Attributes]{}, nil, err
	}
	config, err := calls.ApplyConfig(attrs, attributeCalls, options...)
	if err != nil {
		return config, nil, err
	}
	if config.Options.Enabled.IsSet() {
		if err := masakari.Require(ctx, a.client, 2); err != nil {
			return config, nil, err
		}
	}
	body, err := masakari.Body(config, "segment", attributeProtected...)
	return config, body, err
}

// CreateAttributes POSTs {"segment": <attributes>} and requires 202. Only the
// supplied attributes are sent, so explicit null and omission stay distinct.
func (a *API) CreateAttributes(ctx context.Context, attrs Attributes, options ...AttributeOption) (*Segment, error) {
	const op = "CreateAttributes"
	config, body, err := a.attributeBody(ctx, attrs, append([]AttributeOption(nil), options...))
	if err != nil {
		return nil, request.Wrap(op, "segments", err)
	}
	response, err := rest.DoJSON(ctx, a.client, "POST", a.client.ServiceURL("segments"), body, config.Headers, 202)
	if err != nil {
		return nil, request.Wrap(op, "segments", err)
	}
	value, err := rest.Decode(response, "segment", func(s *Segment) *resource.Metadata { return &s.Metadata })
	return value, request.Wrap(op, "segments", err)
}

// UpdateAttributes PUTs {"segment": <attributes>} to the resolved UUID and
// requires 200. A name is resolved once through the list, as in Update.
func (a *API) UpdateAttributes(ctx context.Context, ref resource.Ref, attrs Attributes, options ...AttributeOption) (*Segment, error) {
	const op = "UpdateAttributes"
	config, body, err := a.attributeBody(ctx, attrs, append([]AttributeOption(nil), options...))
	if err != nil {
		return nil, request.Wrap(op, "segments", err)
	}
	id, err := a.Resources.ResolveID(ctx, ref)
	if err != nil {
		return nil, request.Wrap(op, "segments", err)
	}
	response, err := rest.DoJSON(ctx, a.client, "PUT", a.client.ServiceURL("segments", url.PathEscape(id)), body, config.Headers, 200)
	if err != nil {
		return nil, request.Wrap(op, "segments", err)
	}
	value, err := rest.Decode(response, "segment", func(s *Segment) *resource.Metadata { return &s.Metadata })
	return value, request.Wrap(op, "segments", err)
}
