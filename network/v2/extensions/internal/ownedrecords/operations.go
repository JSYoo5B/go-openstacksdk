package ownedrecords

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

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// DeleteResult exposes the actual bodyless DELETE receipt, including an
// ignored clean 404. Python returns None; the receipt is a Go convenience.
type DeleteResult struct {
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	IgnoredMissing bool
}

// Get implements get_*(value, fields=...): GET {path}/{id} with canonical
// field names mapped to wire names. A 404 is an error, as in Python.
func Get[O ~func(*GetOpts) error](ctx context.Context, t Target, id string, options []O) (*Record, error) {
	const op = "Get"
	engine, err := t.Engine(ctx, t.Spec.Profile)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	policy, err := Apply(ctx, engine, GetOpts{}, CopyGet, slices.Clone(options))
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	prepared := neutronfind.Options{Microversion: policy.Microversion, Headers: policy.Headers}
	if len(policy.Fields) != 0 {
		fields := slices.Clone(policy.Fields)
		for i, name := range fields {
			if !utf8.ValidString(name) {
				return nil, t.Wrap(ctx, op, Invalid("fields must be UTF-8"))
			}
			for _, field := range t.Spec.Profile.Body {
				if name == field.Name {
					fields[i] = field.Wire
					break
				}
			}
		}
		raw, err := json.Marshal(fields)
		if err != nil {
			return nil, t.Wrap(ctx, op, err)
		}
		prepared.Query = map[string]json.RawMessage{"fields": raw}
		prepared.Order = []string{"fields"}
	}
	value, err := engine.Get(ctx, id, prepared)
	return value, t.Wrap(ctx, op, err)
}

// List implements the Python list generator lazily: declared server query,
// local Body filters, unknown names dropped, *_links and marker paging.
func List[O ~func(*ListOpts) error](ctx context.Context, t Target, options []O) iter.Seq2[*Record, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*Record, error) bool) {
		const op = "List"
		engine, err := t.Engine(ctx, t.Spec.Profile)
		if err != nil {
			yield(nil, t.Wrap(ctx, op, err))
			return
		}
		policy, err := Apply(ctx, engine, ListOpts{}, CopyList, snapshot)
		if err != nil {
			yield(nil, t.Wrap(ctx, op, err))
			return
		}
		for value, err := range engine.List(ctx, listPolicy(policy)) {
			if !yield(value, t.Wrap(ctx, op, err)) || err != nil {
				return
			}
		}
	}
}

// Find implements find_*: GET by ID, then after a clean 400, 403 or 404 a
// complete list matched on ID or name. Missing is nil unless IgnoreMissing
// is false.
func Find[O ~func(*FindOpts) error](ctx context.Context, t Target, nameOrID string, options []O) (*Record, error) {
	const op = "Find"
	engine, err := t.Engine(ctx, t.Spec.Profile)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	policy, err := Apply(ctx, engine, FindOpts{}, CopyFind, slices.Clone(options))
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	value, err := engine.Find(ctx, nameOrID, findPolicy(policy))
	return value, t.Wrap(ctx, op, err)
}

// attrs maps Python **attrs to wire Body keys (body) and canonical names
// (seed). Unknown attributes stay in the body, as the pinned Resources set
// _allow_unknown_attrs_in_body.
func (s *Spec) attrs(policy WriteOpts, update bool) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	owned := CopyWrite(policy)
	body, seed := make(map[string]json.RawMessage), make(map[string]json.RawMessage)
	for _, key := range owned.order {
		raw := owned.Attrs[key]
		if raw == nil {
			raw = json.RawMessage("null")
		}
		if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
			return nil, nil, Invalid("nonblank UTF-8 attribute key is required")
		}
		for _, character := range key {
			if unicode.IsControl(character) {
				return nil, nil, Invalid("attribute key contains controls")
			}
		}
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, Invalid("attribute %q requires complete UTF-8 JSON", key)
		}
		switch key {
		case "base_path", "resource_type", "cls", "connection", "_synchronized", "self", "__conflicting_attrs", "location", "requires_id", "create_opts", "create_method", "commit_method", "has_body", "allow_empty_commit", "resource_key", "resource_request_key", "resource_response_key", "_connection", "_body", "_header", "_uri", "_computed":
			return nil, nil, Invalid("%q is an SDK-owned constructor/route control", key)
		case "microversion":
			// Resource.new consumes it; create/commit use the selected default.
			continue
		}
		if update && (key == "id" || key == s.Bound) {
			return nil, nil, Invalid("%q collides with the update identity", key)
		}
		mapped := false
		for _, field := range s.Profile.Body {
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

func transport(v WriteOpts) neutronfind.Options {
	return neutronfind.Options{Microversion: copyVersion(v.Microversion), Headers: CopyWrite(v).Headers}
}

// Create implements create_*(**attrs): POST {path} with {singular: attrs}.
func Create[O ~func(*WriteOpts) error](ctx context.Context, t Target, options []O) (*Record, error) {
	const op = "Create"
	engine, err := t.Engine(ctx, t.Spec.Profile)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	policy, err := Apply(ctx, engine, WriteOpts{}, CopyWrite, slices.Clone(options))
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	body, seed, err := t.Spec.attrs(policy, false)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	value, err := engine.Mutate(ctx, http.MethodPost, "", body, seed, transport(policy))
	return value, t.Wrap(ctx, op, err)
}

// Update implements update_*(value, **attrs): PUT {path}/{id} with only the
// given attributes. No attributes returns the seed view without HTTP.
func Update[O ~func(*WriteOpts) error](ctx context.Context, t Target, id string, options []O) (*Record, error) {
	const op = "Update"
	engine, err := t.Engine(ctx, t.Spec.Profile)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	policy, err := Apply(ctx, engine, WriteOpts{}, CopyWrite, slices.Clone(options))
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	body, seed, err := t.Spec.attrs(policy, true)
	if err == nil {
		err = ValidateID("ID", id)
	}
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	seed["id"], _ = json.Marshal(id)
	var value *Record
	if len(body) == 0 {
		value, err = engine.ProjectSeed(ctx, seed, transport(policy))
	} else {
		value, err = engine.Mutate(ctx, http.MethodPut, id, body, seed, transport(policy))
	}
	return value, t.Wrap(ctx, op, err)
}

// Delete implements delete_*(value, ignore_missing=True). Only a clean 404
// is ignored, and only while IgnoreMissing is nil or true.
func Delete[O ~func(*DeleteOpts) error](ctx context.Context, t Target, id string, options []O) (*DeleteResult, error) {
	const op = "Delete"
	engine, err := t.Engine(ctx, t.Spec.Profile)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	policy, err := Apply(ctx, engine, DeleteOpts{}, CopyDelete, slices.Clone(options))
	if err == nil {
		err = ValidateID("ID", id)
	}
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	value, err := engine.Mutate(ctx, http.MethodDelete, id, nil, nil, neutronfind.Options{Microversion: policy.Microversion, Headers: policy.Headers})
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
	return result, t.Wrap(ctx, op, err)
}
