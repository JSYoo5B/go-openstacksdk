package availabilityzones

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/octaviarest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// CreateOpts creates a zone. Name and AvailabilityZoneProfileID are required.
// Nil Description and Enabled are omitted; an empty description and
// Enabled=false are sent.
type CreateOpts struct {
	Name                      string  `json:"name"`
	AvailabilityZoneProfileID string  `json:"availability_zone_profile_id"`
	Description               *string `json:"description,omitempty"`
	Enabled                   *bool   `json:"enabled,omitempty"`
}

// CreateOption is one option of Create: WithCreateOptions, request.Field (an
// extension body member that cannot replace a typed one) or request.Custom
// over *request.Config[CreateOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type CreateOption = request.WriteCall

var createCalls = calls.Spec[request.Config[CreateOpts]]{Field: calls.ConfigField[CreateOpts]}

func cloneCreate(o CreateOpts) CreateOpts {
	if o.Description != nil {
		v := *o.Description
		o.Description = &v
	}
	if o.Enabled != nil {
		v := *o.Enabled
		o.Enabled = &v
	}
	return o
}

func WithCreateOptions(options CreateOpts) CreateOption {
	snapshot := cloneCreate(options)
	return request.Custom(func(c *request.Config[CreateOpts]) error { c.Options = cloneCreate(snapshot); return nil })
}

// Create sends POST .../availabilityzones and expects 201. The request is
// sent once; an uncertain failure is never replayed.
func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*AvailabilityZone, error) {
	if err := a.check(ctx); err != nil {
		return nil, wrap("Create", "", err)
	}
	config, err := calls.ApplyConfig(opts, createCalls, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, false)
	}
	if err == nil && (strings.TrimSpace(config.Options.Name) == "" || strings.TrimSpace(config.Options.AvailabilityZoneProfileID) == "") {
		err = fmt.Errorf("%w: availability zone name and availability_zone_profile_id are required", resource.ErrInvalidOption)
	}
	var body map[string]any
	if err == nil {
		body, err = envelope("availability_zone", config.Options, config.Fields)
	}
	if err != nil {
		return nil, wrap("Create", "", err)
	}
	return a.single("Create", "", http.MethodPost, a.endpoint(), body, 201)(ctx)
}

// UpdateOpts changes mutable fields only. Name and profile are immutable.
// At least one field or extension is required; Python silently skips the
// request when nothing changed, which Go reports as an invalid option.
type UpdateOpts struct {
	Description *string `json:"description,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
}

// UpdateOption is one option of Update: WithUpdateOptions, request.Field or
// request.Custom over *request.Config[UpdateOpts]. Other shared calls fail
// with resource.ErrInvalidOption before HTTP.
type UpdateOption = request.WriteCall

var updateCalls = calls.Spec[request.Config[UpdateOpts]]{Field: calls.ConfigField[UpdateOpts]}

func WithUpdateOptions(options UpdateOpts) UpdateOption {
	snapshot := cloneUpdate(options)
	return request.Custom(func(c *request.Config[UpdateOpts]) error { c.Options = cloneUpdate(snapshot); return nil })
}

func cloneUpdate(o UpdateOpts) UpdateOpts {
	if o.Description != nil {
		v := *o.Description
		o.Description = &v
	}
	if o.Enabled != nil {
		v := *o.Enabled
		o.Enabled = &v
	}
	return o
}

// Update sends PUT .../availabilityzones/{name} and expects 200.
func (a *API) Update(ctx context.Context, name string, opts UpdateOpts, options ...UpdateOption) (*AvailabilityZone, error) {
	if err := a.check(ctx); err != nil {
		return nil, wrap("Update", "", err)
	}
	if err := validateName(name); err != nil {
		return nil, wrap("Update", "", err)
	}
	config, err := calls.ApplyConfig(opts, updateCalls, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, false)
	}
	var body map[string]any
	if err == nil {
		body, err = envelope("availability_zone", config.Options, config.Fields)
	}
	if err == nil && len(body["availability_zone"].(map[string]any)) == 0 {
		err = fmt.Errorf("%w: availability zone update requires at least one field", resource.ErrInvalidOption)
	}
	if err != nil {
		return nil, wrap("Update", "", err)
	}
	return a.single("Update", name, http.MethodPut, a.endpoint(url.PathEscape(name)), body, 200)(ctx)
}

func envelope[T any](key string, options T, extra map[string]json.RawMessage) (map[string]any, error) {
	encoded, err := json.Marshal(options)
	if err != nil {
		return nil, err
	}
	var core map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &core); err != nil {
		return nil, err
	}
	fields := make(map[string]any, len(core))
	for k, v := range core {
		fields[k] = v
	}
	return request.MergeFieldsFor(map[string]any{key: fields}, extra, options)
}

// ListOpts filters zones. Nil pointers and empty strings are omitted; use the
// pointer fields to send false or an empty-string filter.
type ListOpts struct {
	Name                      *string
	Description               *string
	AvailabilityZoneProfileID *string
	Enabled                   *bool
	Fields                    []string
	Limit                     int
	Marker                    string
	PageReverse               *bool
}

// ListOption is one option of List: WithListOptions, request.Query with a
// string or []string value (it cannot replace a core list option) or
// request.Custom over *request.Config[ListOpts]. The page limit and marker are
// ListOpts members; request.Limit, request.Marker and other shared calls fail
// with resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

var listCalls = calls.Spec[request.Config[ListOpts]]{Query: calls.ConfigQuery[ListOpts]}

func WithListOptions(options ListOpts) ListOption {
	snapshot := cloneList(options)
	return request.Custom(func(c *request.Config[ListOpts]) error { c.Options = cloneList(snapshot); return nil })
}

func cloneList(o ListOpts) ListOpts {
	o.Fields = append([]string(nil), o.Fields...)
	for _, p := range []**string{&o.Name, &o.Description, &o.AvailabilityZoneProfileID} {
		if *p != nil {
			v := **p
			*p = &v
		}
	}
	for _, p := range []**bool{&o.Enabled, &o.PageReverse} {
		if *p != nil {
			v := **p
			*p = &v
		}
	}
	return o
}

var coreQuery = map[string]bool{"name": true, "description": true, "availability_zone_profile_id": true, "enabled": true, "fields": true, "limit": true, "marker": true, "page_reverse": true}

// List streams zones across pages (GET, 200), following availability_zones_links.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*AvailabilityZone, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*AvailabilityZone, error) bool) {
		if err := a.check(ctx); err != nil {
			yield(nil, wrap("List", "", err))
			return
		}
		config, err := calls.ApplyConfig(ListOpts{}, listCalls, options...)
		if err == nil {
			err = request.ValidateCapabilities(config, false, true, false)
		}
		var query url.Values
		if err == nil {
			query, err = listQuery(config)
		}
		if err != nil {
			yield(nil, wrap("List", "", err))
			return
		}
		for v, err := range a.iterate(ctx, "List", query, resource.ListControl{}) {
			if !yield(v, err) || err != nil {
				return
			}
		}
	}
}

func listQuery(config request.Config[ListOpts]) (url.Values, error) {
	o := config.Options
	if o.Limit < 0 {
		return nil, fmt.Errorf("%w: page limit must be non-negative", resource.ErrInvalidOption)
	}
	q := url.Values{}
	for key, value := range map[string]*string{"name": o.Name, "description": o.Description, "availability_zone_profile_id": o.AvailabilityZoneProfileID} {
		if value != nil {
			q.Set(key, *value)
		}
	}
	if o.Enabled != nil {
		q.Set("enabled", strconv.FormatBool(*o.Enabled))
	}
	for _, f := range o.Fields {
		if strings.TrimSpace(f) == "" {
			return nil, fmt.Errorf("%w: field must not be empty", resource.ErrInvalidOption)
		}
		q.Add("fields", f)
	}
	if o.Limit > 0 {
		q.Set("limit", strconv.Itoa(o.Limit))
	}
	if o.Marker != "" {
		q.Set("marker", o.Marker)
	}
	if o.PageReverse != nil {
		// Octavia compares the raw value with the case-sensitive string "True".
		q.Set("page_reverse", map[bool]string{true: "True", false: "False"}[*o.PageReverse])
	}
	for key, values := range config.Query {
		if coreQuery[key] {
			return nil, fmt.Errorf("%w: query extension %q is a core list option", resource.ErrInvalidOption, key)
		}
		q[key] = append([]string(nil), values...)
	}
	return q, nil
}

func (a *API) iterate(ctx context.Context, op string, query url.Values, control resource.ListControl) iter.Seq2[*AvailabilityZone, error] {
	return func(yield func(*AvailabilityZone, error) bool) {
		if err := a.check(ctx); err != nil {
			yield(nil, wrap(op, "", err))
			return
		}
		first := a.endpoint()
		if len(query) > 0 {
			first += "?" + query.Encode()
		}
		count := 0
		for row, err := range octaviarest.Pages(ctx, a.client, first, "availability_zones", control.SinglePage) {
			if err == nil {
				var value *AvailabilityZone
				if value, err = decode(row.Raw, row.Header, row.Status); err == nil {
					count++
					if !yield(value, nil) {
						return
					}
					if control.MaxItems > 0 && count >= control.MaxItems {
						return
					}
					continue
				}
			}
			yield(nil, wrap(op, "", err))
			return
		}
	}
}
