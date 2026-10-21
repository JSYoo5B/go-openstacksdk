package deploytemplates

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

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

type CreateOption = request.Option[CreateOpts]

// WithCreateField adds a JSON member this package does not model. It cannot
// replace a typed member.
func WithCreateField(key string, value any) CreateOption {
	return request.WithField[CreateOpts](key, value)
}

func WithCreateHeader(key, value string) CreateOption {
	return request.WithHeader[CreateOpts](key, value)
}

// UpdateOpts selects members to set. Every set member is sent as an "add"
// operation on its path, ordered by path, like the Python resource commit. Use
// Patch for remove, replace, test or explicit null.
type UpdateOpts struct {
	Name  *string                    `json:"name,omitempty"`
	Steps []Step                     `json:"steps,omitzero"`
	Extra map[string]json.RawMessage `json:"extra,omitzero"`
}

type UpdateOption = request.Option[UpdateOpts]

func WithUpdateField(key string, value any) UpdateOption {
	return request.WithField[UpdateOpts](key, value)
}
func WithUpdateHeader(key, value string) UpdateOption {
	return request.WithHeader[UpdateOpts](key, value)
}

// GetOpts selects the members to fetch. Names are server member names such as
// uuid or name; they are sent as a comma separated fields query.
type GetOpts struct{ Fields []string }

type GetOption = request.Option[GetOpts]

func WithGetFields(fields ...string) GetOption {
	owned := append([]string(nil), fields...)
	return func(c *request.Config[GetOpts]) error { c.Options.Fields = owned; return nil }
}
func WithGetHeader(key, value string) GetOption { return request.WithHeader[GetOpts](key, value) }

// ListOpts controls a list. Detail true sends detail=true (false sends
// nothing). Limit is a page size: the walk continues until an empty page.
// MaxItems caps the rows yielded and, without Limit, is also sent as limit.
// SinglePage stops after the first page.
type ListOpts struct {
	Detail     bool
	Fields     []string
	Limit      *int
	Marker     *string
	MaxItems   int
	SinglePage bool
}

type ListOption = request.Option[ListOpts]

func WithListDetail(detail bool) ListOption {
	return func(c *request.Config[ListOpts]) error { c.Options.Detail = detail; return nil }
}
func WithListFields(fields ...string) ListOption {
	owned := append([]string(nil), fields...)
	return func(c *request.Config[ListOpts]) error { c.Options.Fields = owned; return nil }
}
func WithListLimit(limit int) ListOption {
	return func(c *request.Config[ListOpts]) error { c.Options.Limit = &limit; return nil }
}
func WithListMarker(marker string) ListOption {
	return func(c *request.Config[ListOpts]) error { c.Options.Marker = &marker; return nil }
}
func WithListMaxItems(n int) ListOption {
	return func(c *request.Config[ListOpts]) error { c.Options.MaxItems = n; return nil }
}
func WithListSinglePage() ListOption {
	return func(c *request.Config[ListOpts]) error { c.Options.SinglePage = true; return nil }
}

// WithListQuery adds a server-side filter or sort query such as sort_key. It
// cannot replace detail, fields, limit or marker.
func WithListQuery(key, value string) ListOption  { return request.WithQuery[ListOpts](key, value) }
func WithListHeader(key, value string) ListOption { return request.WithHeader[ListOpts](key, value) }

// DeleteOpts: IgnoreMissing defaults to true, as in Python.
type DeleteOpts struct{ IgnoreMissing bool }

type DeleteOption = request.Option[DeleteOpts]

func WithDeleteIgnoreMissing(ignore bool) DeleteOption {
	return func(c *request.Config[DeleteOpts]) error { c.Options.IgnoreMissing = ignore; return nil }
}
func WithDeleteHeader(key, value string) DeleteOption {
	return request.WithHeader[DeleteOpts](key, value)
}

func listConfig(options []ListOption) (ironicrest.ListConfig, error) {
	config, err := request.Apply(ListOpts{}, options...)
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
