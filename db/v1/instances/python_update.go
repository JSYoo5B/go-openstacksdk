package instances

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/masakari"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// UpdateAttributes holds the Body attributes the pinned Python Instance
// declares, which update_instance may commit. Every field is a
// request.Optional: omitted by default, JSON null with request.Null, and exact
// values (including empty strings) with request.Present. Structured values are
// supplied as JSON. The SDK does not validate them; Trove owns the update
// schema. The instance ID is never part of the body.
type UpdateAttributes struct {
	Name      request.Optional[string]          `json:"name,omitzero"`
	Flavor    request.Optional[json.RawMessage] `json:"flavor,omitzero"`
	Volume    request.Optional[json.RawMessage] `json:"volume,omitzero"`
	Datastore request.Optional[json.RawMessage] `json:"datastore,omitzero"`
	Status    request.Optional[string]          `json:"status,omitzero"`
	Region    request.Optional[string]          `json:"region,omitzero"`
	Hostname  request.Optional[string]          `json:"hostname,omitzero"`
	Links     request.Optional[json.RawMessage] `json:"links,omitzero"`
	CreatedAt request.Optional[string]          `json:"created,omitzero"`
	UpdatedAt request.Optional[string]          `json:"updated,omitzero"`
}

type UpdateAttributeOption = request.Option[UpdateAttributes]

func WithUpdateAttributeField(key string, value any) UpdateAttributeOption {
	return request.WithField[UpdateAttributes](key, value)
}

func WithUpdateAttributeHeader(key, value string) UpdateAttributeOption {
	return request.WithHeader[UpdateAttributes](key, value)
}

// UpdateReceipt is the accepted HTTP response of UpdateAttributes. Trove
// normally acknowledges with 202 and no body, so Body is usually empty and is
// never decoded into an Instance; read the instance again to observe state.
type UpdateReceipt struct {
	StatusCode int
	Header     http.Header
	Body       json.RawMessage
}

// UpdateAttributes sends PUT /instances/{id} with {"instance": <attributes>}
// and accepts 200, 202 or 204. Only supplied attributes are sent. A request
// without any attribute is rejected before HTTP, whereas Python skips the PUT
// when nothing is dirty. A name is resolved through the instance list once;
// an ID reference needs no lookup. The call never replays on failure, and a
// canceled context or an error after the request was written does not prove
// that Trove did not apply the change.
func (a *API) UpdateAttributes(ctx context.Context, ref resource.Ref, attrs UpdateAttributes, options ...UpdateAttributeOption) (*UpdateReceipt, error) {
	const op = "UpdateAttributes"
	config, err := request.Apply(attrs, append([]UpdateAttributeOption(nil), options...)...)
	if err != nil {
		return nil, request.Wrap(op, "instances", err)
	}
	if err := request.ValidateCapabilities(config, true, false, true); err != nil {
		return nil, request.Wrap(op, "instances", err)
	}
	if err := masakari.Headers(config.Headers); err != nil {
		return nil, request.Wrap(op, "instances", err)
	}
	encoded, err := json.Marshal(config.Options)
	if err != nil {
		return nil, request.Wrap(op, "instances", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, request.Wrap(op, "instances", err)
	}
	body := make(map[string]any, len(fields))
	for key, value := range fields {
		body[key] = value
	}
	if _, protected := config.Fields["id"]; protected {
		return nil, request.Wrap(op, "instances", resourceInvalid("extension field \"id\" is owned by the SDK"))
	}
	if body, err = request.MergeFieldsFor(body, config.Fields, config.Options); err != nil {
		return nil, request.Wrap(op, "instances", err)
	}
	if len(body) == 0 {
		return nil, request.Wrap(op, "instances", resourceInvalid("update requires at least one attribute"))
	}
	id, err := a.Resources.ResolveID(ctx, ref)
	if err != nil {
		return nil, request.Wrap(op, "instances", err)
	}
	payload, err := json.Marshal(map[string]any{"instance": body})
	if err != nil {
		return nil, request.Wrap(op, "instances", err)
	}
	response, err := rest.DoJSON(ctx, a.client, http.MethodPut, a.client.ServiceURL("instances", url.PathEscape(id)), json.RawMessage(payload), config.Headers, 200, 202, 204)
	if err != nil {
		return nil, request.Wrap(op, "instances", err)
	}
	return &UpdateReceipt{StatusCode: response.StatusCode, Header: response.Header.Clone(), Body: append(json.RawMessage(nil), response.Body...)}, nil
}

func resourceInvalid(message string) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, message)
}
