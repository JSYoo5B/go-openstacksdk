package routers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/gophercloud/gophercloud/v2"
)

// CreateCloudRecord sends only the translated create_router cloud body. Source
// request values are uncoerced; descriptor casts apply to the returned view.
// The caller uses network.Service.CreateRouterRecord for defaults and AZ gates.
// Existing native Create/CreateWithAttributes and their status policy are intact.
func (a *API) CreateCloudRecord(ctx context.Context, attrs map[string]json.RawMessage) (*RouterReadRecord, error) {
	const op = "CreateCloudRecord"
	if a == nil {
		return nil, routerReadWrap(ctx, op, routerActionInvalid("configured routers API required"))
	}
	profile := routerReadProfile
	profile.SourceStatusPolicy = true
	selected := a.client
	engine, err := neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if a.client != selected {
			return routerActionInvalid("selected routers create client changed")
		}
		return nil
	})
	if err != nil {
		return nil, routerReadWrap(ctx, op, err)
	}
	body, seed := make(map[string]json.RawMessage, len(attrs)), make(map[string]json.RawMessage, len(attrs))
	if _, present := attrs["admin_state_up"]; !present {
		return nil, routerReadWrap(ctx, op, routerActionInvalid("translated cloud body requires admin_state_up by presence"))
	}
	for key, raw := range attrs {
		switch key {
		case "admin_state_up", "name", "project_id", "external_gateway_info", "availability_zone_hints":
		default:
			return nil, routerReadWrap(ctx, op, routerActionInvalid("%q is not a translated cloud router attribute", key))
		}
		if raw == nil {
			raw = json.RawMessage("null")
		}
		if _, err := cloudfilter.PythonJSONTruthy(raw); err != nil {
			return nil, routerReadWrap(ctx, op, routerActionInvalid("attribute %q requires complete UTF-8 JSON: %v", key, err))
		}
		body[key] = bytes.Clone(raw)
		if key == "admin_state_up" {
			key = "is_admin_state_up"
		}
		seed[key] = bytes.Clone(raw)
	}
	value, err := engine.Mutate(ctx, http.MethodPost, "", body, seed, neutronfind.Options{})
	if value == nil {
		original := err
		var auth *gophercloud.ErrUnableToReauthenticate
		if errors.As(err, &auth) && auth != nil {
			original = auth.ErrOriginal
		}
		var native gophercloud.ErrUnexpectedResponseCode
		if errors.As(original, &native) && native.Method == http.MethodPost {
			value = &neutronfind.Record{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
		}
	}
	return routerReadRecord(value), routerReadWrap(ctx, op, errors.Join(err, engine.Check(ctx)))
}
