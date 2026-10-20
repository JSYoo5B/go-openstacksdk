// Package serverexternalevents owns Nova's inter-service external event API.
package serverexternalevents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/responsejson"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }

// Event is a concrete JSON attributes object. The SDK selects declared fields;
// unknown attributes are discarded, while known request values remain raw.
type Event map[string]any
type EventRecord = neutronfind.Record

type CreateOpts struct{ Microversion *string }
type CreateOption = request.Option[CreateOpts]

func WithCreateHeader(key, value string) CreateOption {
	return request.WithHeader[CreateOpts](key, value)
}
func WithCreateMicroversion(value string) CreateOption {
	return func(c *request.Config[CreateOpts]) error { x := value; c.Options.Microversion = &x; return nil }
}

func eventProfile() neutronfind.Profile {
	return neutronfind.Profile{
		Kind: "server external events", Path: "os-server-external-events", ServiceRole: "compute", IDKey: "id", NameKey: "name",
		SourceFindPolicy: true, SourceStatusPolicy: true, StopFaultRetryCallbacks: true,
		Body: []neutronfind.Descriptor{
			{Name: "name", Wire: "name", Missing: "null"}, {Name: "server_uuid", Wire: "server_uuid", Missing: "null"}, {Name: "status", Wire: "status", Missing: "null"}, {Name: "tag", Wire: "tag", Missing: "null"},
			{Name: "code", Wire: "code", Missing: "null", Kind: neutronfind.Integer}, {Name: "id", Wire: "id", Missing: "null"},
		}}
}
func eventInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}
func ownCreate(c *request.Config[CreateOpts]) {
	cloudread.OwnReadConfig(c)
	if c.Options.Microversion != nil {
		x := *c.Options.Microversion
		c.Options.Microversion = &x
	}
}

// CreateRecords posts a nonempty batch immediately, then projects response
// rows lazily. A malformed later row yields an error retaining the whole HTTP
// receipt after earlier rows. Response rows never inherit submitted fields.
func (a *API) CreateRecords(ctx context.Context, events []Event, options ...CreateOption) (iter.Seq2[*EventRecord, error], error) {
	fail := func(err error) error {
		return request.Wrap("CreateRecords", "server external events", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, fail(err)
	}
	if a == nil {
		return nil, fail(eventInvalid("event API is required"))
	}
	selected := a.client
	var observed error
	selectedGuard := func(callCtx context.Context) error {
		if observed == nil && a.client != selected {
			observed = eventInvalid("selected event API changed")
		}
		return cloudread.ContextError(callCtx, observed)
	}
	engine, err := neutronfind.New(ctx, selected, eventProfile(), selectedGuard)
	if err != nil {
		return nil, fail(err)
	}
	if len(events) == 0 {
		return nil, fail(eventInvalid("events must be a nonempty list of objects"))
	}
	body := make([]map[string]json.RawMessage, len(events))
	for i, event := range events {
		if event == nil {
			return nil, fail(eventInvalid("event %d must be an object", i))
		}
		for _, key := range []string{"connection", "_synchronized", "self", "cls"} {
			if _, exists := event[key]; exists {
				return nil, fail(eventInvalid("event %d %q collides with Source constructor", i, key))
			}
		}
		body[i] = map[string]json.RawMessage{}
		for _, field := range eventProfile().Body {
			if value, exists := event[field.Wire]; exists {
				raw, cause := json.Marshal(value)
				if cause != nil {
					return nil, fail(cause)
				}
				body[i][field.Wire] = bytes.Clone(raw)
			}
		}
		if _, cause := engine.ProjectSeed(ctx, body[i], neutronfind.Options{}); cause != nil {
			return nil, fail(cause)
		}
	}
	cfg, err := cloudread.ApplyReadOptions(ctx, CreateOpts{}, options, ownCreate, engine.Check)
	if err == nil {
		err = request.ValidateCapabilities(cfg, false, false, true)
	}
	if err != nil {
		return nil, fail(err)
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fail(err)
	}
	response, err := engine.PostCollection(ctx, map[string]json.RawMessage{"events": encoded}, neutronfind.Options{Headers: cfg.Headers, Microversion: cfg.Options.Microversion})
	if err != nil {
		return nil, fail(err)
	}
	normalized, err := responsejson.Normalize(response.Body, response.Header)
	if err != nil {
		return nil, fail(response.Fail(err))
	}
	var envelope map[string]json.RawMessage
	if err = json.Unmarshal(normalized, &envelope); err != nil || envelope == nil {
		return nil, fail(response.Fail(errors.Join(eventInvalid("response must be an events object"), err)))
	}
	raw, present := envelope["events"]
	if !present {
		return nil, fail(response.Fail(eventInvalid("response events key is required")))
	}
	return func(yield func(*EventRecord, error) bool) {
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			yield(nil, fail(response.Fail(errors.Join(eventInvalid("events response must be an array"), err))))
			return
		}
		for _, row := range rows {
			if err := engine.Check(ctx); err != nil {
				yield(nil, fail(response.Fail(err)))
				return
			}
			value, err := engine.ProjectExistingRow(ctx, row, response, neutronfind.ExistingRowPolicy{Bulk: true})
			if err != nil {
				yield(nil, fail(response.Fail(err)))
				return
			}
			if !yield(value, nil) {
				return
			}
		}
	}, nil
}
