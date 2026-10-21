package shareaccessrules

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// AllowActionOpts is the body of POST /shares/{id}/action {"allow_access":...}
// as sent by Python create_access_rule. A zero Optional is omitted; Present
// keeps empty and false values; Null sends JSON null.
type AllowActionOpts struct {
	AccessType     request.Optional[string]          `json:"access_type,omitzero"`
	AccessTo       request.Optional[string]          `json:"access_to,omitzero"`
	AccessLevel    request.Optional[string]          `json:"access_level,omitzero"`
	Metadata       request.Optional[json.RawMessage] `json:"metadata,omitzero"`
	LockVisibility request.Optional[bool]            `json:"lock_visibility,omitzero"`
	LockDeletion   request.Optional[bool]            `json:"lock_deletion,omitzero"`
	LockReason     request.Optional[string]          `json:"lock_reason,omitzero"`
}

type AllowActionOption = request.Option[AllowActionOpts]

func WithAllowActionOptions(opts AllowActionOpts) AllowActionOption { return request.WithOptions(opts) }
func WithAllowActionField(key string, value any) AllowActionOption {
	return request.WithField[AllowActionOpts](key, value)
}

// AllowAccessAction grants access through the share action route directly,
// with no microversion floor and no parent lookup, like Python's
// create_access_rule. The share must be given by ID. The request is sent once
// and never replayed; HTTP 200 is the only accepted status and the "access"
// object is returned with all of its fields.
func (a *API) AllowAccessAction(ctx context.Context, shareID string, opts AllowActionOpts, options ...AllowActionOption) (*resource.RawResource, error) {
	const op, kind = "AllowAccessAction", "share_access_rules"
	if err := a.actionPreflight(ctx, shareID); err != nil {
		return nil, request.Wrap(op, kind, err)
	}
	config, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, false)
	}
	var body map[string]any
	if err == nil {
		body, err = actionBody("allow_access", config.Options, config.Fields)
	}
	if err != nil {
		return nil, request.Wrap(op, kind, err)
	}
	response, err := rest.DoJSON(ctx, a.client, http.MethodPost, a.client.ServiceURL("shares", url.PathEscape(shareID), "action"), body, nil, 200)
	if err != nil {
		return nil, request.Wrap(op, kind, err)
	}
	value, err := rest.Decode(response, "access", func(r *resource.RawResource) *resource.Metadata { return &r.Metadata })
	return value, request.Wrap(op, kind, err)
}

// DenyActionOpts selects the deny_access options. Unrestrict is sent only when set.
type DenyActionOpts struct {
	IgnoreMissing *bool
	Unrestrict    *bool
}
type DenyActionOption = request.Option[DenyActionOpts]

func WithDenyActionUnrestrict(value bool) DenyActionOption {
	return func(c *request.Config[DenyActionOpts]) error { c.Options.Unrestrict = &value; return nil }
}

// WithDenyActionIgnoreMissing selects Python's ignore_missing; the default is true.
func WithDenyActionIgnoreMissing(value bool) DenyActionOption {
	return func(c *request.Config[DenyActionOpts]) error { c.Options.IgnoreMissing = &value; return nil }
}

// DenyActionResult is the actual HTTP response of the accepted action, the
// equivalent of the raw response Python returns.
type DenyActionResult struct {
	Body       json.RawMessage
	Header     http.Header
	StatusCode int
}

// DenyAccessAction POSTs {"deny_access":{"access_id":...}} to the share action
// route with no lookup, matching Python delete_access_rule. HTTP 200, 202 and
// 204 succeed. A 404 returns nil, nil unless WithDenyActionIgnoreMissing(false)
// is given; other errors are returned. Never replayed. Success only means the
// request was accepted, not that the rule is gone.
func (a *API) DenyAccessAction(ctx context.Context, shareID, accessID string, options ...DenyActionOption) (*DenyActionResult, error) {
	const op, kind = "DenyAccessAction", "share_access_rules"
	if err := a.actionPreflight(ctx, shareID); err != nil {
		return nil, request.Wrap(op, kind, err)
	}
	if err := resource.ID(accessID).Validate(); err != nil {
		return nil, request.Wrap(op, kind, err)
	}
	config, err := request.Apply(DenyActionOpts{}, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, false, false, false)
	}
	if err != nil {
		return nil, request.Wrap(op, kind, err)
	}
	inner := map[string]any{"access_id": accessID}
	if config.Options.Unrestrict != nil {
		inner["unrestrict"] = *config.Options.Unrestrict
	}
	response, err := rest.DoJSON(ctx, a.client, http.MethodPost, a.client.ServiceURL("shares", url.PathEscape(shareID), "action"), map[string]any{"deny_access": inner}, nil, 200, 202, 204)
	if err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) && (config.Options.IgnoreMissing == nil || *config.Options.IgnoreMissing) {
			return nil, nil
		}
		return nil, request.Wrap(op, kind, err)
	}
	return &DenyActionResult{Body: append(json.RawMessage(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}, nil
}

func (a *API) actionPreflight(ctx context.Context, shareID string) error {
	if ctx == nil {
		return fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if a == nil || a.client == nil || a.client.ProviderClient == nil {
		return invalid("a service client is required")
	}
	if err := resource.ID(shareID).Validate(); err != nil {
		return err
	}
	return ctx.Err()
}

func actionBody[T any](envelope string, options T, fields map[string]json.RawMessage) (map[string]any, error) {
	encoded, err := json.Marshal(options)
	if err != nil {
		return nil, err
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &members); err != nil {
		return nil, err
	}
	inner := make(map[string]any, len(members))
	for key, value := range members {
		inner[key] = value
	}
	return request.MergeFieldsFor(map[string]any{envelope: inner}, fields, options)
}
