// Package ironic holds the shared SDK-owned transport for Bare Metal resources
// whose routes the pinned Gophercloud release does not bind. Service packages own
// models, option types, field names and documentation; this helper owns request
// framing, explicit success codes, JSON Patch validation and paging.
package ironic

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Spec describes one Ironic resource collection.
type Spec[T any] struct {
	Client *gophercloud.ServiceClient
	// Kind is the resource name used in wrapped errors.
	Kind string
	// Path is the collection path below the endpoint, for example
	// "volume/connectors". PluralKey is the list envelope key.
	Path, PluralKey string
	// DetailRoute selects GET <Path>/detail for detailed lists. Otherwise the
	// detailed list is GET <Path>?detail=True.
	DetailRoute bool
	UUID, Name  func(*T) string
	Metadata    func(*T) *resource.Metadata
	// PatchProtected lists top-level JSON Patch targets the server owns.
	PatchProtected []string
}

// Core is the shared implementation behind one concrete API.
type Core[T any] struct {
	spec       Spec[T]
	list, full rest.CollectionSpec[T]
	Resources  *resource.Collection[T]
}

// New performs no HTTP.
func New[T any](spec Spec[T]) *Core[T] {
	base := rest.CollectionSpec[T]{
		Client: spec.Client, Path: spec.Path, Kind: spec.Kind, PluralKey: spec.PluralKey,
		ID: spec.UUID, Name: spec.Name, Metadata: spec.Metadata,
		Paging: rest.PagePolicy[T]{SourceLinkPolicy: true, IgnoreFalseyNext: true, MarkerFallback: true, MarkerOnShortPage: true,
			Marker: func(value *T) (string, error) {
				if value == nil || strings.TrimSpace(spec.UUID(value)) == "" {
					return "", fmt.Errorf("%w: pagination row lacks uuid", resource.ErrInvalidOption)
				}
				return spec.UUID(value), nil
			}},
	}
	c := &Core[T]{spec: spec, list: base, full: base}
	if spec.DetailRoute {
		c.full.Path = spec.Path + "/detail"
	}
	c.Resources = resource.NewCollection(resource.Adapter[T]{
		Kind: spec.Kind, IdentityFind: true, ValidateID: validateRoute,
		Get:    func(ctx context.Context, id string) (*T, error) { return c.Get(ctx, id, nil) },
		Delete: c.Delete, ID: spec.UUID, Name: spec.Name,
		IterateControlled: func(ctx context.Context, query url.Values, control resource.ListControl) iter.Seq2[*T, error] {
			return c.iterate(ctx, true, query, rest.ListControl{MaxItems: control.MaxItems, SinglePage: control.SinglePage})
		},
		IterateIdentity: func(ctx context.Context, query url.Values, details bool) iter.Seq2[*T, error] {
			return c.iterate(ctx, details, query, rest.ListControl{})
		},
	})
	return c
}

func validateRoute(id string) error { return resource.ID(id).Validate() }

func (c *Core[T]) wrap(op string, err error) error { return request.Wrap(op, c.spec.Kind, err) }

func (c *Core[T]) decode(op string, response *rest.Response) (*T, error) {
	value, err := rest.Decode(response, "", c.spec.Metadata)
	return value, c.wrap(op, err)
}

// Create sends the already validated body and accepts only 201.
func (c *Core[T]) Create(ctx context.Context, body map[string]any, headers map[string]string) (*T, error) {
	response, err := rest.DoJSON(ctx, c.spec.Client, http.MethodPost, c.spec.Client.ServiceURL(strings.Split(c.spec.Path, "/")...), body, headers, http.StatusCreated)
	if err != nil {
		return nil, c.wrap("Create", err)
	}
	return c.decode("Create", response)
}

// Get reads one resource and accepts only 200. Fields are wire names.
func (c *Core[T]) Get(ctx context.Context, id string, fields []string) (*T, error) {
	if err := validateRoute(id); err != nil {
		return nil, c.wrap("Get", err)
	}
	endpoint := c.member(id)
	if fields != nil {
		joined, err := joinFields(fields)
		if err != nil {
			return nil, c.wrap("Get", err)
		}
		endpoint += "?" + url.Values{"fields": {joined}}.Encode()
	}
	response, err := rest.DoJSON(ctx, c.spec.Client, http.MethodGet, endpoint, nil, nil, http.StatusOK)
	if err != nil {
		return nil, c.wrap("Get", err)
	}
	return c.decode("Get", response)
}

// Update sends a JSON Patch and accepts only 200.
func (c *Core[T]) Update(ctx context.Context, id string, operations []PatchOperation) (*T, error) {
	if err := validateRoute(id); err != nil {
		return nil, c.wrap("Update", err)
	}
	if err := ValidatePatch(operations, c.spec.PatchProtected); err != nil {
		return nil, c.wrap("Update", err)
	}
	response, err := rest.DoJSON(ctx, c.spec.Client, http.MethodPatch, c.member(id), operations, nil, http.StatusOK)
	if err != nil {
		return nil, c.wrap("Update", err)
	}
	return c.decode("Update", response)
}

// Delete accepts only 204; a 404 stays an error.
func (c *Core[T]) Delete(ctx context.Context, id string) error {
	if err := validateRoute(id); err != nil {
		return c.wrap("Delete", err)
	}
	_, err := rest.DoJSON(ctx, c.spec.Client, http.MethodDelete, c.member(id), nil, nil, http.StatusNoContent)
	return c.wrap("Delete", err)
}

func (c *Core[T]) member(id string) string {
	return c.spec.Client.ServiceURL(append(strings.Split(c.spec.Path, "/"), url.PathEscape(id))...)
}

// List streams one summary or detailed collection. Wrapped errors name op.
func (c *Core[T]) List(ctx context.Context, op string, detail bool, query url.Values) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		if detail && query.Has("fields") {
			yield(nil, c.wrap(op, fmt.Errorf("%w: fields cannot be combined with a detailed list", resource.ErrInvalidOption)))
			return
		}
		for value, err := range c.iterate(ctx, detail, query, rest.ListControl{}) {
			if err != nil {
				yield(nil, c.wrap(op, err))
				return
			}
			if !yield(value, nil) {
				return
			}
		}
	}
}

func (c *Core[T]) iterate(ctx context.Context, detail bool, query url.Values, control rest.ListControl) iter.Seq2[*T, error] {
	spec := c.list
	query = cloneQuery(query)
	if detail {
		if c.spec.DetailRoute {
			spec = c.full
		} else {
			query.Set("detail", "True")
		}
	}
	return rest.ListWithControl(ctx, spec, query, control)
}

func cloneQuery(query url.Values) url.Values {
	out := make(url.Values, len(query)+1)
	for key, values := range query {
		out[key] = append([]string(nil), values...)
	}
	return out
}

func joinFields(fields []string) (string, error) {
	if len(fields) == 0 {
		return "", fmt.Errorf("%w: fields must not be empty", resource.ErrInvalidOption)
	}
	for _, field := range fields {
		if strings.TrimSpace(field) == "" || strings.Contains(field, ",") {
			return "", fmt.Errorf("%w: field names must be nonempty and comma-free", resource.ErrInvalidOption)
		}
	}
	return strings.Join(fields, ","), nil
}

// JoinFields validates and joins a fields selection for a query value.
func JoinFields(fields []string) (string, error) { return joinFields(fields) }

// ListQuery is the concrete list filter set shared by Ironic collections.
type ListQuery struct {
	Fields                  []string
	Limit                   int
	Marker                  string
	SortKey, SortDir        string
	Node, Address           string
	AllowNode, AllowAddress bool
}

// Values converts typed filters into a query. Zero values are omitted, like
// Python's falsy query values; a negative limit is invalid.
func (q ListQuery) Values() (url.Values, error) {
	values := url.Values{}
	if q.Limit < 0 {
		return nil, fmt.Errorf("%w: limit must not be negative", resource.ErrInvalidOption)
	}
	if q.Limit > 0 {
		values.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.Fields != nil {
		joined, err := joinFields(q.Fields)
		if err != nil {
			return nil, err
		}
		values.Set("fields", joined)
	}
	for key, value := range map[string]string{"marker": q.Marker, "sort_key": q.SortKey, "sort_dir": q.SortDir, "node": q.Node, "address": q.Address} {
		if value != "" {
			values.Set(key, value)
		}
	}
	if q.Node != "" && !q.AllowNode || q.Address != "" && !q.AllowAddress {
		return nil, fmt.Errorf("%w: filter is not defined for this resource", resource.ErrInvalidOption)
	}
	return values, nil
}

// CoreQueryKeys are the keys concrete filters own; extension queries cannot use them.
var CoreQueryKeys = []string{"fields", "limit", "marker", "sort_key", "sort_dir", "node", "address", "detail"}

// MergeQuery adds extension queries and rejects keys owned by typed filters.
func MergeQuery(values, extra url.Values) (url.Values, error) {
	for key, list := range extra {
		for _, core := range CoreQueryKeys {
			if key == core {
				return nil, fmt.Errorf("%w: query %q is a core filter", resource.ErrInvalidOption, key)
			}
		}
		values[key] = append([]string(nil), list...)
	}
	return values, nil
}

// BuildBody merges extension fields at the body root. Keys listed in reserved
// are owned by typed inputs or the server and cannot be supplied as extensions.
func BuildBody(core map[string]any, fields map[string]json.RawMessage, reserved ...string) (map[string]any, error) {
	body := make(map[string]any, len(core)+len(fields))
	for key, value := range core {
		body[key] = value
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		for _, name := range reserved {
			if key == name {
				return nil, fmt.Errorf("%w: extension %q is a core or protected field", resource.ErrInvalidOption, key)
			}
		}
		body[key] = append(json.RawMessage(nil), fields[key]...)
	}
	return body, nil
}

// PatchOperation is one RFC 6902 operation accepted by Ironic. Add and
// Replace always send "value", including an explicit nil as JSON null; Remove
// sends no value unless Value is set.
type PatchOperation struct {
	Op    string
	Path  string
	Value any
}

func Add(path string, value any) PatchOperation     { return PatchOperation{"add", path, value} }
func Replace(path string, value any) PatchOperation { return PatchOperation{"replace", path, value} }
func Remove(path string) PatchOperation             { return PatchOperation{Op: "remove", Path: path} }

func (p PatchOperation) MarshalJSON() ([]byte, error) {
	if p.Op == "remove" && p.Value == nil {
		return json.Marshal(struct {
			Op   string `json:"op"`
			Path string `json:"path"`
		}{p.Op, p.Path})
	}
	return json.Marshal(struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value any    `json:"value"`
	}{p.Op, p.Path, p.Value})
}

// FieldPatches reproduces Python update_*(**attrs): one "add" per top-level
// attribute, sorted by path. Python derives the same list from a diff against an
// empty remote state.
func FieldPatches(fields map[string]any) []PatchOperation {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]PatchOperation, 0, len(keys))
	for _, key := range keys {
		out = append(out, Add("/"+key, fields[key]))
	}
	return out
}

// ValidatePatch rejects empty patches, unknown operations, malformed paths and
// server-owned targets before HTTP.
func ValidatePatch(operations []PatchOperation, protected []string) error {
	if len(operations) == 0 {
		return fmt.Errorf("%w: at least one patch operation is required", resource.ErrInvalidOption)
	}
	for i, op := range operations {
		switch op.Op {
		case "add", "replace", "remove":
		default:
			return fmt.Errorf("%w: patch operation %d has unsupported op %q", resource.ErrInvalidOption, i, op.Op)
		}
		if len(op.Path) < 2 || op.Path[0] != '/' {
			return fmt.Errorf("%w: patch operation %d needs a path starting with /", resource.ErrInvalidOption, i)
		}
		first := strings.SplitN(op.Path[1:], "/", 2)[0]
		for _, name := range protected {
			if first == name {
				return fmt.Errorf("%w: patch operation %d targets server-owned %q", resource.ErrInvalidOption, i, name)
			}
		}
	}
	return nil
}
