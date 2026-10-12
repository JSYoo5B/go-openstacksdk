package ownedrecord

import (
	"bytes"
	"context"
	"encoding/json"
	"iter"
	"net/http"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Create implements a Python create_* proxy call: POST {collection} with
// {resource_key: attrs}. Only the given attributes are sent.
func Create[O ~func(*WriteOpts) error](ctx context.Context, b Binding, parent string, options []O) (*Record, error) {
	const op = "Create"
	engine, err := b.engine(ctx, parent)
	if err != nil {
		return nil, b.Wrap(ctx, op, err)
	}
	policy, err := Apply(ctx, engine.Check, WriteOpts{}, CopyWrite, slices.Clone(options))
	if err != nil {
		return nil, b.Wrap(ctx, op, err)
	}
	body, seed, err := b.Spec.attrs(policy, b.Spec.ParentReserved())
	if err != nil {
		return nil, b.Wrap(ctx, op, err)
	}
	value, err := engine.Mutate(ctx, http.MethodPost, "", body, seed, writeTransport(policy))
	return b.record(value, parent), b.Wrap(ctx, op, err)
}

// Get implements a Python get_* proxy call (Resource.fetch through
// _get_with_fields): GET {collection}/{id} with fields mapped to wire names.
func Get[O ~func(*GetOpts) error](ctx context.Context, b Binding, parent, id string, options []O) (*Record, error) {
	const op = "Get"
	engine, err := b.engine(ctx, parent)
	if err != nil {
		return nil, b.Wrap(ctx, op, err)
	}
	policy, err := Apply(ctx, engine.Check, GetOpts{}, CopyGet, slices.Clone(options))
	if err != nil {
		return nil, b.Wrap(ctx, op, err)
	}
	prepared, err := b.Spec.getOptions(policy)
	if err != nil {
		return nil, b.Wrap(ctx, op, err)
	}
	value, err := engine.Get(ctx, id, prepared)
	return b.record(value, parent), b.Wrap(ctx, op, err)
}

// Update implements a Python update_* proxy call (Resource.commit with PUT)
// for a literal ID. No dirty attribute returns the seeded view without HTTP.
func Update[O ~func(*WriteOpts) error](ctx context.Context, b Binding, parent, id string, options []O) (*Record, error) {
	const op = "Update"
	engine, err := b.engine(ctx, parent)
	if err != nil {
		return nil, b.Wrap(ctx, op, err)
	}
	policy, err := Apply(ctx, engine.Check, WriteOpts{}, CopyWrite, slices.Clone(options))
	if err != nil {
		return nil, b.Wrap(ctx, op, err)
	}
	reserved := append([]string{"id"}, b.Spec.UpdateReserved...)
	body, seed, err := b.Spec.attrs(policy, append(reserved, b.Spec.ParentReserved()...))
	if err == nil {
		err = ValidateSegment("ID", id)
	}
	if err != nil {
		return nil, b.Wrap(ctx, op, err)
	}
	seed["id"], _ = json.Marshal(id)
	var value *neutronfind.Record
	if len(body) == 0 {
		value, err = engine.ProjectSeed(ctx, seed, writeTransport(policy))
	} else {
		value, err = engine.Mutate(ctx, http.MethodPut, id, body, seed, writeTransport(policy))
	}
	return b.record(value, parent), b.Wrap(ctx, op, err)
}

// Delete implements a Python delete_* proxy call. A clean 404 is nil unless
// IgnoreMissing is false; every other failure is returned.
func Delete[O ~func(*DeleteOpts) error](ctx context.Context, b Binding, parent, id string, options []O) error {
	const op = "Delete"
	engine, err := b.engine(ctx, parent)
	if err != nil {
		return b.Wrap(ctx, op, err)
	}
	policy, err := Apply(ctx, engine.Check, DeleteOpts{}, CopyDelete, slices.Clone(options))
	if err == nil {
		err = ValidateSegment("ID", id)
	}
	if err != nil {
		return b.Wrap(ctx, op, err)
	}
	_, err = engine.Mutate(ctx, http.MethodDelete, id, nil, nil, neutronfind.Options{Headers: policy.Headers, Microversion: policy.Microversion})
	if policy.IgnoreMissing == nil || *policy.IgnoreMissing {
		err = resource.IgnoreMissing(err)
	}
	return b.Wrap(ctx, op, err)
}

// List implements a Python list proxy call as a lazy iterator. Every
// iteration captures the client and sends the requests again.
func List[O ~func(*ListOpts) error](ctx context.Context, b Binding, parent string, options []O) iter.Seq2[*Record, error] {
	owned := slices.Clone(options)
	return func(yield func(*Record, error) bool) {
		const op = "List"
		engine, err := b.engine(ctx, parent)
		if err != nil {
			yield(nil, b.Wrap(ctx, op, err))
			return
		}
		policy, err := Apply(ctx, engine.Check, ListOpts{}, CopyList, owned)
		if err == nil {
			err = b.Spec.rejectReserved(policy.order, b.Spec.ParentReserved())
		}
		if err != nil {
			yield(nil, b.Wrap(ctx, op, err))
			return
		}
		for value, err := range engine.List(ctx, listOptions(policy)) {
			if !yield(b.record(value, parent), b.Wrap(ctx, op, err)) || err != nil {
				return
			}
		}
	}
}

// Find implements a Python find_* proxy call: GET by ID first and, after a
// clean 400, 403 or 404 only, the complete list matched by exact ID or name.
// A parented class forwards the parent URI attribute as Resource.find does.
func Find[O ~func(*FindOpts) error](ctx context.Context, b Binding, parent, nameOrID string, options []O) (*Record, error) {
	const op = "Find"
	engine, err := b.engine(ctx, parent)
	if err != nil {
		return nil, b.Wrap(ctx, op, err)
	}
	policy, err := Apply(ctx, engine.Check, FindOpts{}, CopyFind, slices.Clone(options))
	if err == nil {
		err = b.Spec.rejectReserved(policy.order, b.Spec.ParentReserved())
	}
	if err != nil {
		return nil, b.Wrap(ctx, op, err)
	}
	prepared := findOptions(policy)
	if b.Spec.Parent != "" {
		raw, _ := json.Marshal(parent)
		if prepared.Query == nil {
			prepared.Query = make(map[string]json.RawMessage)
		}
		prepared.Query[b.Spec.Parent] = raw
		prepared.Order = append(slices.Clone(prepared.Order), b.Spec.Parent)
	}
	value, err := engine.Find(ctx, nameOrID, prepared)
	return b.record(value, parent), b.Wrap(ctx, op, err)
}

func writeTransport(v WriteOpts) neutronfind.Options {
	// Attributes belong to the body. They never become URL query values.
	return neutronfind.Options{Headers: v.Headers, Microversion: v.Microversion}
}

func (s *Spec) rejectReserved(keys []string, reserved []string) error {
	for _, key := range keys {
		if slices.Contains(reserved, key) {
			return Invalid("%q duplicates a bound proxy argument", key)
		}
	}
	return nil
}

// attrs maps Python **attrs to the request body (wire names) and the
// Resource seed (attribute names). Unknown names stay in both.
func (s *Spec) attrs(v WriteOpts, reserved []string) (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	owned := CopyWrite(v)
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
			return nil, nil, Invalid("%q is an SDK-owned constructor or route control", key)
		case "microversion":
			// Resource.new consumes this constructor value; create and commit
			// use the selected client version rather than a Body member.
			continue
		}
		if slices.Contains(reserved, key) {
			return nil, nil, Invalid("%q duplicates a bound proxy argument", key)
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

// getOptions maps fields from Python attribute names to wire names, as
// _get_with_fields does; nil and empty fields send no query.
func (s *Spec) getOptions(v GetOpts) (neutronfind.Options, error) {
	prepared := neutronfind.Options{Microversion: v.Microversion, Headers: v.Headers}
	if len(v.Fields) == 0 {
		return prepared, nil
	}
	fields := slices.Clone(v.Fields)
	for i, name := range fields {
		if !utf8.ValidString(name) {
			return prepared, Invalid("fields must be UTF-8")
		}
		for _, field := range s.Profile.Body {
			if name == field.Name {
				fields[i] = field.Wire
				break
			}
		}
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		return prepared, err
	}
	prepared.Query = map[string]json.RawMessage{"fields": raw}
	prepared.Order = []string{"fields"}
	return prepared, nil
}
