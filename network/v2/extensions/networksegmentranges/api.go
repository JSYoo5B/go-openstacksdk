package networksegmentranges

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// API shares the selected authenticated Neutron client without taking
// ownership. Every route is /network_segment_ranges on that client.
type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

func wrap(ctx context.Context, op string, err error) error {
	return request.Wrap(op, kind, cloudread.ContextError(ctx, err))
}

func (a *API) engine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, invalid("configured network segment range API is required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, profile, func(context.Context) error {
		if a.client != selected {
			return invalid("selected network segment range client changed")
		}
		return nil
	})
}

func validID(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return invalid("nonblank UTF-8 literal network segment range ID is required")
	}
	for _, char := range id {
		if unicode.IsControl(char) {
			return invalid("network segment range ID contains controls")
		}
	}
	return nil
}

// mutationAttrs splits ordered Source attrs into the wire Body and the
// canonical seed that Resource keeps when the response omits a member.
func mutationAttrs(policy neutronfind.Options, update bool) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	owned := neutronfind.CloneOptions(policy)
	body, seed := make(map[string]json.RawMessage), make(map[string]json.RawMessage)
	for _, key := range owned.Order {
		raw := owned.Query[key]
		if raw == nil {
			raw = json.RawMessage("null")
		}
		if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
			return nil, nil, invalid("nonblank UTF-8 attribute key is required")
		}
		for _, char := range key {
			if unicode.IsControl(char) {
				return nil, nil, invalid("attribute key contains controls")
			}
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, invalid("attribute %q requires complete UTF-8 JSON", key)
		}
		switch key {
		case "base_path", "resource_type", "cls", "connection", "_synchronized", "self", "__conflicting_attrs", "location", "requires_id", "create_method", "commit_method", "has_body", "allow_empty_commit", "resource_key", "resource_request_key", "resource_response_key", "_connection", "_body", "_header", "_uri", "_computed":
			return nil, nil, invalid("%q is an SDK-owned constructor or route control", key)
		case "microversion":
			// Resource.new consumes it; create/commit keep the selected default.
			continue
		}
		if update && key == "id" {
			return nil, nil, invalid("id collides with the update identity")
		}
		mapped := false
		for _, field := range profile.Body {
			if key == field.Name || key == field.Wire {
				body[field.Wire] = bytes.Clone(raw)
				seed[field.Name] = bytes.Clone(raw)
				mapped = true
			}
		}
		if !mapped {
			body[key] = bytes.Clone(raw)
			seed[key] = bytes.Clone(raw)
		}
	}
	return body, seed, nil
}

func transportPolicy(policy neutronfind.Options) neutronfind.Options {
	owned := neutronfind.CloneOptions(policy)
	// Attributes belong to the Body. They never become URL query values.
	owned.Query, owned.Order = nil, nil
	return owned
}

// Create POSTs only the provided attributes under network_segment_range.
// Neutron owns required-field validation; nothing is synthesized.
func (a *API) Create(ctx context.Context, options ...CreateOption) (*Record, error) {
	const op = "Create"
	engine, err := a.engine(ctx)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	callbacks := make([]func(*CreateOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyOptions(ctx, engine, CreateOpts{}, copyCreate, callbacks)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	prepared := createPolicy(policy)
	body, seed, err := mutationAttrs(prepared, false)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	value, err := engine.Mutate(ctx, http.MethodPost, "", body, seed, transportPolicy(prepared))
	return record(value), wrap(ctx, op, err)
}

// Get fetches one literal ID. Nonempty fields become repeated fields query
// parameters; the member is never searched by name.
func (a *API) Get(ctx context.Context, id string, options ...GetOption) (*Record, error) {
	const op = "Get"
	engine, err := a.engine(ctx)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	callbacks := make([]func(*GetOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyOptions(ctx, engine, GetOpts{}, copyGet, callbacks)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	prepared, err := getPolicy(policy)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	value, err := engine.Get(ctx, id, prepared)
	return record(value), wrap(ctx, op, err)
}

// Update PUTs only the provided attributes for a literal ID. No attribute
// returns the Source seed view without an HTTP request.
func (a *API) Update(ctx context.Context, id string, options ...UpdateOption) (*Record, error) {
	const op = "Update"
	engine, err := a.engine(ctx)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	callbacks := make([]func(*UpdateOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyOptions(ctx, engine, UpdateOpts{}, copyUpdate, callbacks)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	prepared := updatePolicy(policy)
	body, seed, err := mutationAttrs(prepared, true)
	if err == nil {
		err = validID(id)
	}
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	seed["id"], _ = json.Marshal(id)
	var value *neutronfind.Record
	if len(body) == 0 {
		value, err = engine.ProjectSeed(ctx, seed, transportPolicy(prepared))
	} else {
		value, err = engine.Mutate(ctx, http.MethodPut, id, body, seed, transportPolicy(prepared))
	}
	return record(value), wrap(ctx, op, err)
}

// Delete sends a bodyless DELETE. IgnoreMissing defaults to true and only a
// clean actual 404 is suppressed.
func (a *API) Delete(ctx context.Context, id string, options ...DeleteOption) (*DeleteResult, error) {
	const op = "Delete"
	engine, err := a.engine(ctx)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	callbacks := make([]func(*DeleteOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyOptions(ctx, engine, DeleteOpts{}, copyDelete, callbacks)
	if err == nil {
		err = validID(id)
	}
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	value, err := engine.Mutate(ctx, http.MethodDelete, id, nil, nil, deletePolicy(policy))
	var result *DeleteResult
	if value != nil {
		result = &DeleteResult{Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode}
	}
	if err != nil && (policy.IgnoreMissing == nil || *policy.IgnoreMissing) && resource.IgnoreMissing(err) == nil {
		var native gophercloud.ErrUnexpectedResponseCode
		if errors.As(err, &native) && native.Actual == http.StatusNotFound {
			result = &DeleteResult{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual, IgnoredMissing: true}
			err = nil
		}
	}
	return result, wrap(ctx, op, err)
}

// List lazily yields Source-shaped records with declared server query
// parameters, local Body filters, raw-row caps and complete owned pagination
// (limit/marker, network_segment_ranges_links, HTTP Link and next).
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Record, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*Record, error) bool) {
		const op = "List"
		engine, err := a.engine(ctx)
		if err != nil {
			yield(nil, wrap(ctx, op, err))
			return
		}
		callbacks := make([]func(*ListOpts) error, len(snapshot))
		for i, apply := range snapshot {
			callbacks[i] = apply
		}
		policy, err := applyOptions(ctx, engine, ListOpts{}, copyList, callbacks)
		if err != nil {
			yield(nil, wrap(ctx, op, err))
			return
		}
		for value, err := range engine.List(ctx, listPolicy(policy)) {
			if !yield(record(value), wrap(ctx, op, err)) {
				return
			}
		}
	}
}

// All collects a successful List and discards partial rows on failure.
func (a *API) All(ctx context.Context, options ...ListOption) ([]*Record, error) {
	var values []*Record
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// Find GETs the literal identity first. Only a clean 400/403/404 falls back to
// a complete list with the name query hint and exact ID or name matching.
// Default ignore-missing returns nil, nil after a clean search.
func (a *API) Find(ctx context.Context, nameOrID string, options ...FindOption) (*Record, error) {
	const op = "Find"
	engine, err := a.engine(ctx)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	callbacks := make([]func(*FindOpts) error, len(options))
	for i, apply := range slices.Clone(options) {
		callbacks[i] = apply
	}
	policy, err := applyOptions(ctx, engine, FindOpts{}, copyFind, callbacks)
	if err != nil {
		return nil, wrap(ctx, op, err)
	}
	value, err := engine.Find(ctx, nameOrID, findPolicy(policy))
	return record(value), wrap(ctx, op, err)
}
