package deploytemplates

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/ironicrest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// PatchOperation is one RFC 6902 operation sent by Patch. Paths use the server
// member names, for example /name.
type PatchOperation = ironicrest.PatchOperation

// CreateOpts creates a deploy template. Nil pointers, nil slices and nil maps are
// omitted; an empty string, false, an empty map or an empty slice is sent.
type CreateOpts struct {
	Name  string                     `json:"name"`
	Steps []Step                     `json:"steps,omitzero"`
	UUID  *string                    `json:"uuid,omitempty"`
	Extra map[string]json.RawMessage `json:"extra,omitzero"`
}

// CreateOption is one option of Create: request.Field (a JSON member this package does not model; it
// cannot replace a typed member),
// request.Header or request.Custom over *request.Config[CreateOpts]. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

var createCalls = calls.Spec[request.Config[CreateOpts]]{Field: calls.ConfigField[CreateOpts], Header: calls.ConfigHeader[CreateOpts]}

// UpdateOpts selects members to set. Every set member is sent as an "add"
// operation on its path, ordered by path, like the Python resource commit. Use
// Patch for remove, replace, test or explicit null.
type UpdateOpts struct {
	Name  *string                    `json:"name,omitempty"`
	Steps []Step                     `json:"steps,omitzero"`
	Extra map[string]json.RawMessage `json:"extra,omitzero"`
}

// UpdateOption is one option of Update: request.Field, request.Header or
// request.Custom over *request.Config[UpdateOpts]. A field cannot replace a
// typed member. Other shared calls fail with resource.ErrInvalidOption before
// HTTP.
type UpdateOption = request.WriteCall

var updateCalls = calls.Spec[request.Config[UpdateOpts]]{Field: calls.ConfigField[UpdateOpts], Header: calls.ConfigHeader[UpdateOpts]}

// GetOpts selects the members to fetch. Names are server member names such as
// uuid or name; they are sent as a comma separated fields query.
type GetOpts struct{ Fields []string }

// GetOption is one option of Get: WithGetFields, request.Header or
// request.Custom over *request.Config[GetOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type GetOption = request.GetCall

var getCalls = calls.Spec[request.Config[GetOpts]]{Header: calls.ConfigHeader[GetOpts]}

func WithGetFields(fields ...string) GetOption {
	owned := append([]string(nil), fields...)
	return request.Custom(func(c *request.Config[GetOpts]) error { c.Options.Fields = owned; return nil })
}

// ListOpts controls a list. Detail true sends detail=true (false sends
// nothing). Limit is a page size: the walk continues until an empty page.
// MaxItems caps the rows yielded and, without Limit, is also sent as limit.
// SinglePage stops after the first page. request.Limit, request.Marker,
// request.MaxItems and request.Paginated(false) set Limit, Marker, MaxItems and
// SinglePage.
type ListOpts struct {
	Detail     bool
	Fields     []string
	Limit      *int
	Marker     *string
	MaxItems   int
	SinglePage bool
}

// ListOption is one option of List: WithListDetail, WithListFields,
// request.Limit, request.Marker, request.MaxItems, request.Paginated,
// request.Query with a string or []string value (a server-side filter or sort
// such as sort_key; it cannot replace detail, fields, limit or marker),
// request.Header or request.Custom over *request.Config[ListOpts]. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

var listCalls = calls.Spec[request.Config[ListOpts]]{
	Header: calls.ConfigHeader[ListOpts],
	Query:  calls.ConfigQuery[ListOpts],
	Paging: calls.Paging[request.Config[ListOpts]]{
		Limit:     func(c *request.Config[ListOpts], n int) error { c.Options.Limit = &n; return nil },
		Marker:    func(c *request.Config[ListOpts], marker string) error { c.Options.Marker = &marker; return nil },
		MaxItems:  func(c *request.Config[ListOpts], n int) error { c.Options.MaxItems = n; return nil },
		Paginated: func(c *request.Config[ListOpts], paginated bool) error { c.Options.SinglePage = !paginated; return nil },
	},
}

func WithListDetail(detail bool) ListOption {
	return request.Custom(func(c *request.Config[ListOpts]) error { c.Options.Detail = detail; return nil })
}
func WithListFields(fields ...string) ListOption {
	owned := append([]string(nil), fields...)
	return request.Custom(func(c *request.Config[ListOpts]) error { c.Options.Fields = owned; return nil })
}

// DeleteOpts: IgnoreMissing defaults to true, as in Python.
type DeleteOpts struct{ IgnoreMissing bool }

// DeleteOption is one option of Delete: request.IgnoreMissing,
// request.Header or request.Custom over *request.Config[DeleteOpts]. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type DeleteOption = request.DeleteCall

var deleteCalls = calls.Spec[request.Config[DeleteOpts]]{
	Header:        calls.ConfigHeader[DeleteOpts],
	IgnoreMissing: func(c *request.Config[DeleteOpts], ignore bool) error { c.Options.IgnoreMissing = ignore; return nil },
}

func listConfig(options []ListOption) (ironicrest.ListConfig, error) {
	config, err := calls.ApplyConfig(ListOpts{}, listCalls, options...)
	if err != nil {
		return ironicrest.ListConfig{}, err
	}
	if err := request.ValidateCapabilities(config, false, true, true); err != nil {
		return ironicrest.ListConfig{}, err
	}
	if err := ironicrest.CheckQuery(config.Query, "detail", "fields", "limit", "marker"); err != nil {
		return ironicrest.ListConfig{}, err
	}
	o := config.Options
	query := url.Values{}
	for k, v := range config.Query {
		query[k] = append([]string(nil), v...)
	}
	if o.Detail {
		query.Set("detail", "true")
	}
	if len(o.Fields) > 0 {
		csv, err := ironicrest.Fields(o.Fields)
		if err != nil {
			return ironicrest.ListConfig{}, err
		}
		query.Set("fields", csv)
	}
	if o.Limit != nil {
		if *o.Limit <= 0 {
			return ironicrest.ListConfig{}, fmt.Errorf("%w: list limit must be positive", resource.ErrInvalidOption)
		}
		query.Set("limit", fmt.Sprint(*o.Limit))
	}
	if o.Marker != nil {
		if strings.TrimSpace(*o.Marker) == "" {
			return ironicrest.ListConfig{}, fmt.Errorf("%w: list marker must not be empty", resource.ErrInvalidOption)
		}
		query.Set("marker", *o.Marker)
	}
	if o.MaxItems < 0 {
		return ironicrest.ListConfig{}, fmt.Errorf("%w: max items must not be negative", resource.ErrInvalidOption)
	}
	return ironicrest.ListConfig{Query: query, Headers: config.Headers, Paginated: !o.SinglePage, MaxItems: o.MaxItems}, nil
}
