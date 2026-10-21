package ironicrest

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// PatchOperation is one RFC 6902 operation. A nil Value is omitted; use
// json.RawMessage("null") to send an explicit JSON null.
type PatchOperation struct {
	Op    string          `json:"op"`
	Path  string          `json:"path"`
	From  *string         `json:"from,omitempty"`
	Value json.RawMessage `json:"value,omitempty"`
}

// ValidatePatch rejects operations the service cannot interpret before HTTP.
func ValidatePatch(ops []PatchOperation) error {
	if len(ops) == 0 {
		return invalid("a patch needs at least one operation")
	}
	for i, op := range ops {
		if !strings.HasPrefix(op.Path, "/") || len(op.Path) < 2 {
			return invalid("patch operation %d: path %q must be a JSON pointer below the root", i, op.Path)
		}
		switch op.Op {
		case "add", "replace", "test":
			if len(op.Value) == 0 || !json.Valid(op.Value) {
				return invalid("patch operation %d: %s needs a valid JSON value", i, op.Op)
			}
		case "remove":
			if len(op.Value) != 0 {
				return invalid("patch operation %d: remove must not carry a value", i)
			}
		case "move", "copy":
			if op.From == nil || !strings.HasPrefix(*op.From, "/") {
				return invalid("patch operation %d: %s needs a from pointer", i, op.Op)
			}
		default:
			return invalid("patch operation %d: unsupported op %q", i, op.Op)
		}
	}
	return nil
}

// AddOps builds one add operation per field, ordered by path, which is how the
// Python resource commit expresses a set of attribute changes.
func AddOps(fields map[string]json.RawMessage) []PatchOperation {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	ops := make([]PatchOperation, 0, len(keys))
	for _, k := range keys {
		ops = append(ops, PatchOperation{Op: "add", Path: "/" + escapePointer(k), Value: append(json.RawMessage(nil), fields[k]...)})
	}
	return ops
}

func escapePointer(s string) string {
	return strings.NewReplacer("~", "~0", "/", "~1").Replace(s)
}

// MarshalFlat encodes core members and flattened extras. An extra may not
// replace a core member.
func MarshalFlat(core map[string]any, extra map[string]json.RawMessage) ([]byte, error) {
	out := make(map[string]any, len(core)+len(extra))
	for k, v := range core {
		out[k] = v
	}
	for k, v := range extra {
		if _, exists := out[k]; exists {
			return nil, invalid("extra member %q conflicts with a typed member", k)
		}
		out[k] = v
	}
	return json.Marshal(out)
}

// SplitExtra decodes an object and returns the members not in known.
func SplitExtra(data []byte, known ...string) (map[string]json.RawMessage, error) {
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, err
	}
	for _, k := range known {
		delete(all, k)
	}
	if len(all) == 0 {
		return nil, nil
	}
	return all, nil
}

// BuildBody marshals typed options and merges extension fields, refusing any
// extension that names a typed member.
func BuildBody[O any](opts O, fields map[string]json.RawMessage) (json.RawMessage, error) {
	encoded, err := json.Marshal(opts)
	if err != nil {
		return nil, invalid("request options: %v", err)
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &members); err != nil {
		return nil, err
	}
	body := make(map[string]any, len(members))
	for k, v := range members {
		body[k] = v
	}
	merged, err := request.MergeFieldsFor(body, fields, opts)
	if err != nil {
		return nil, err
	}
	return json.Marshal(merged)
}

// Resource is the shared route engine for one Ironic-style collection.
type Resource[T any] struct {
	Client     *gophercloud.ServiceClient
	Kind       string // singular label used in errors, for example "runbook"
	Collection string // collection route segment, for example "runbooks"
	ListKey    string // key of the list array in a list response
	Decode     func(json.RawMessage, *Response) (*T, error)
	Identify   func(*T) (uuid, name string)
}

func (r Resource[T]) itemURL(id string, segments ...string) string {
	parts := append([]string{r.Collection, url.PathEscape(id)}, segments...)
	return r.Client.ServiceURL(parts...)
}

// Create sends POST /{collection}; only 201 succeeds.
func (r Resource[T]) Create(ctx context.Context, body json.RawMessage, headers map[string]string) (*T, error) {
	resp, err := Send(ctx, r.Client, http.MethodPost, r.Client.ServiceURL(r.Collection), body, headers, http.StatusCreated)
	if err != nil {
		return nil, err
	}
	return Decode(resp, r.Decode)
}

// Get sends GET /{collection}/{id}[?fields=csv]; only 200 succeeds.
func (r Resource[T]) Get(ctx context.Context, id string, fields []string, headers map[string]string) (*T, error) {
	if err := Preflight(ctx, r.Client); err != nil {
		return nil, err
	}
	if err := ValidateID(r.Kind, id); err != nil {
		return nil, err
	}
	csv, err := Fields(fields)
	if err != nil {
		return nil, err
	}
	target := r.itemURL(id)
	if len(fields) > 0 {
		target += "?" + url.Values{"fields": {csv}}.Encode()
	}
	resp, err := Send(ctx, r.Client, http.MethodGet, target, nil, headers, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return Decode(resp, r.Decode)
}

// Patch sends PATCH /{collection}/{id}; only 200 succeeds.
func (r Resource[T]) Patch(ctx context.Context, id string, ops []PatchOperation, headers map[string]string) (*T, error) {
	if err := Preflight(ctx, r.Client); err != nil {
		return nil, err
	}
	if err := ValidateID(r.Kind, id); err != nil {
		return nil, err
	}
	if err := ValidatePatch(ops); err != nil {
		return nil, err
	}
	resp, err := Send(ctx, r.Client, http.MethodPatch, r.itemURL(id), ops, headers, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return Decode(resp, r.Decode)
}

// Delete sends DELETE /{collection}/{id}; only 204 succeeds. A 404 is a
// success with deleted=false when ignoreMissing is set.
func (r Resource[T]) Delete(ctx context.Context, id string, ignoreMissing bool, headers map[string]string) (bool, error) {
	if err := Preflight(ctx, r.Client); err != nil {
		return false, err
	}
	if err := ValidateID(r.Kind, id); err != nil {
		return false, err
	}
	_, err := Send(ctx, r.Client, http.MethodDelete, r.itemURL(id), nil, headers, http.StatusNoContent)
	if err != nil {
		if ignoreMissing && IsNotFound(err) {
			return false, nil
		}
		if IsNotFound(err) {
			return false, &resource.NotFoundError{Resource: r.Kind, Reference: id, Cause: err}
		}
		return false, err
	}
	return true, nil
}

// List walks the collection and decodes every row.
func (r Resource[T]) List(ctx context.Context, cfg ListConfig) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		for row, err := range List(ctx, r.Client, r.Collection, r.ListKey, cfg) {
			if err != nil {
				yield(nil, err)
				return
			}
			value, err := Decode(&Response{Body: row.Raw, Header: row.Page.Header, Status: row.Page.Status}, r.Decode)
			if err != nil {
				yield(nil, err)
				return
			}
			if !yield(value, nil) {
				return
			}
		}
	}
}

// Find reads GET /{collection}/{ref} first. A 400, 403 or 404 falls back to one
// complete list whose rows are matched by exact uuid or name.
func (r Resource[T]) Find(ctx context.Context, ref string, ignoreMissing, detail bool, headers map[string]string) (*T, error) {
	if err := Preflight(ctx, r.Client); err != nil {
		return nil, err
	}
	if err := ValidateID(r.Kind, ref); err != nil {
		return nil, err
	}
	value, err := r.Get(ctx, ref, nil, headers)
	if err == nil {
		return value, nil
	}
	if !gophercloud.ResponseCodeIs(err, http.StatusNotFound) && !gophercloud.ResponseCodeIs(err, http.StatusBadRequest) && !gophercloud.ResponseCodeIs(err, http.StatusForbidden) {
		return nil, err
	}
	cfg := ListConfig{Query: url.Values{}, Headers: headers, Paginated: true}
	if detail {
		cfg.Query.Set("detail", "true")
	}
	var found *T
	var ids []string
	for row, err := range r.List(ctx, cfg) {
		if err != nil {
			return nil, err
		}
		id, name := r.Identify(row)
		if id == ref || name == ref {
			ids = append(ids, id)
			if found == nil {
				found = row
			}
		}
	}
	switch {
	case len(ids) > 1:
		return nil, &resource.AmbiguousError{Resource: r.Kind, Name: ref, IDs: ids}
	case found != nil:
		return found, nil
	case ignoreMissing:
		return nil, nil
	}
	return nil, &resource.NotFoundError{Resource: r.Kind, Reference: ref, Cause: err}
}

// TraitURL returns the route below one resource.
func (r Resource[T]) SubURL(id string, segments ...string) string {
	escaped := make([]string, len(segments))
	for i, s := range segments {
		escaped[i] = url.PathEscape(s)
	}
	return r.itemURL(id, escaped...)
}

// CheckQuery rejects extension queries that would override a typed control.
func CheckQuery(extra url.Values, reserved ...string) error {
	for key := range extra {
		if slices.Contains(reserved, key) {
			return fmt.Errorf("%w: query extension %q is a typed list control", resource.ErrInvalidOption, key)
		}
	}
	return nil
}
