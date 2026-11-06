package defaultrules

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// CreateOpts owns the provided Source attributes. Absent attributes are not
// sent. Headers and Microversion are explicit Go transport conveniences.
type CreateOpts struct {
	Fields       map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}

// CreateOption is one option of Create: WithCreateOpts, WithCreateIfMatch,
// request.Field (one Source attribute; a marshal error fails when applied),
// request.Headers (replaces the headers), request.Microversion or
// request.Custom over *CreateOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

func createPolicy(v CreateOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Fields, Headers: v.Headers, Microversion: v.Microversion, Order: v.order}
}
func copyCreate(v CreateOpts) CreateOpts {
	owned := neutronfind.CloneOptions(createPolicy(v))
	v.Fields, v.Headers, v.Microversion, v.order = owned.Query, owned.Headers, owned.Microversion, owned.Order
	return v
}
func WithCreateOpts(v CreateOpts) CreateOption {
	owned := copyCreate(v)
	return request.Custom(func(c *CreateOpts) error { *c = copyCreate(owned); return nil })
}

// WithCreateIfMatch sets the inherited if_match Header attribute. Values are
// joined with ", " into one If-Match request header, as Source does.
func WithCreateIfMatch(values ...string) CreateOption {
	return neutronfind.KeywordCall(createKeyword, "if_match", append([]string{}, values...))
}

// DeleteOpts defaults IgnoreMissing to true, as delete_default_security_group_rule.
type DeleteOpts struct {
	IgnoreMissing *bool
	Headers       map[string]string
	Microversion  *string
}

// DeleteOption is one option of Delete: request.IgnoreMissing,
// request.Headers (replaces the headers), request.Microversion or
// request.Custom over *DeleteOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type DeleteOption = request.DeleteCall

func deletePolicy(v DeleteOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Headers: v.Headers, Microversion: v.Microversion}
}
func copyDelete(v DeleteOpts) DeleteOpts {
	owned := neutronfind.CloneOptions(deletePolicy(v))
	v.IgnoreMissing, v.Headers, v.Microversion = owned.IgnoreMissing, owned.Headers, owned.Microversion
	return v
}

// GetOpts is the get_default_security_group_rule fields selector plus explicit
// transport settings. Field attribute names map to wire names.
type GetOpts struct {
	Fields       []string
	Headers      map[string]string
	Microversion *string
}

// GetOption is one option of Get: WithGetFields, request.Headers (replaces
// the headers), request.Microversion or request.Custom over *GetOpts. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type GetOption = request.GetCall

func copyGet(v GetOpts) GetOpts {
	v.Fields = slices.Clone(v.Fields)
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithGetFields(values ...string) GetOption {
	owned := slices.Clone(values)
	return request.Custom(func(c *GetOpts) error { c.Fields = slices.Clone(owned); return nil })
}
func getPolicy(v GetOpts) (neutronfind.Options, error) {
	prepared := neutronfind.Options{Microversion: v.Microversion, Headers: v.Headers}
	if len(v.Fields) == 0 {
		return prepared, nil
	}
	fields := slices.Clone(v.Fields)
	for i, name := range fields {
		if !utf8.ValidString(name) {
			return prepared, invalid("fields must be UTF-8")
		}
		for _, field := range profile.Body {
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

// ListOpts owns list kwargs: declared server query fields, other declared Body
// attributes as local filters, and paginated/max_items controls.
type ListOpts struct {
	Query        map[string]json.RawMessage
	Headers      map[string]string
	Microversion *string
	order        []string
}

// ListOption is one option of List: WithListOpts, request.Query (one list
// kwarg as JSON; names Resource.list discards are dropped with their
// marshal error), request.Limit, request.Marker, request.MaxItems,
// request.Paginated, request.Headers (replaces the headers),
// request.Microversion or request.Custom over *ListOpts. Other shared calls
// fail with resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

func listPolicy(v ListOpts) neutronfind.Options {
	return neutronfind.Options{Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, Order: v.order}
}
func copyList(v ListOpts) ListOpts {
	owned := neutronfind.CloneOptions(listPolicy(v))
	v.Query, v.Headers, v.Microversion, v.order = owned.Query, owned.Headers, owned.Microversion, owned.Order
	return v
}
func WithListOpts(v ListOpts) ListOption {
	owned := copyList(v)
	return request.Custom(func(c *ListOpts) error { *c = copyList(owned); return nil })
}

// listIgnores reports the unknown kwargs that Resource.list discards before
// evaluating their value. Controls and invalid keys are kept to fail later.
func listIgnores(key string) bool {
	if strings.TrimSpace(key) == "" || !utf8.ValidString(key) {
		return false
	}
	for _, char := range key {
		if unicode.IsControl(char) {
			return false
		}
	}
	switch key {
	case "paginated", "max_items", "base_path", "list_base_path", "resource_type", "connection", "session", "microversion", "headers", "_synchronized", "self", "cls", "requires_id", "error_message", "skip_cache", "resource_response_key", "location", "__conflicting_attrs":
		return false
	}
	for _, pair := range profile.Query {
		if key == pair.Name || key == pair.Wire {
			return false
		}
	}
	for _, field := range profile.Body {
		if key == field.Name {
			return false
		}
	}
	return true
}

// FindOpts owns find kwargs. Omitted IgnoreMissing is true. Query values go to
// the member GET and, classified as for List, to the fallback list.
type FindOpts struct {
	IgnoreMissing *bool
	Query         map[string]json.RawMessage
	Headers       map[string]string
	Microversion  *string
	order         []string
}

// FindOption is one option of Find: WithFindOpts, request.IgnoreMissing,
// request.Query (one find kwarg as JSON), request.Headers (replaces the
// headers), request.Microversion or request.Custom over *FindOpts. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type FindOption = request.FindCall

func findPolicy(v FindOpts) neutronfind.Options {
	return neutronfind.Options{IgnoreMissing: v.IgnoreMissing, Query: v.Query, Headers: v.Headers, Microversion: v.Microversion, Order: v.order}
}
func copyFind(v FindOpts) FindOpts {
	owned := neutronfind.CloneOptions(findPolicy(v))
	v.IgnoreMissing, v.Query, v.Headers, v.Microversion, v.order = owned.IgnoreMissing, owned.Query, owned.Headers, owned.Microversion, owned.Order
	return v
}
func WithFindOpts(v FindOpts) FindOption {
	owned := copyFind(v)
	return request.Custom(func(c *FindOpts) error { *c = copyFind(owned); return nil })
}

func applyOptions[T any](ctx context.Context, engine *neutronfind.Engine, initial T, clone func(T) T, options []func(*T) error) (T, error) {
	policy := clone(initial)
	for _, apply := range options {
		if err := engine.Check(ctx); err != nil {
			return policy, err
		}
		if apply == nil {
			return policy, invalid("nil option")
		}
		next := clone(policy)
		err := apply(&next)
		if err = errors.Join(err, engine.Check(ctx)); err != nil {
			return policy, err
		}
		policy = clone(next)
	}
	return policy, nil
}
