package availabilityzoneprofiles

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/octaviarest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// CreateOpts creates a profile. All three fields are required by the API.
// AvailabilityZoneData is a JSON document encoded as a string, sent verbatim.
type CreateOpts struct {
	Name                 string `json:"name"`
	ProviderName         string `json:"provider_name"`
	AvailabilityZoneData string `json:"availability_zone_data"`
}

type CreateOption = request.Option[CreateOpts]

func WithCreateOptions(options CreateOpts) CreateOption {
	snapshot := options
	return func(c *request.Config[CreateOpts]) error { c.Options = snapshot; return nil }
}

func WithCreateField(key string, value any) CreateOption {
	return request.WithField[CreateOpts](key, value)
}

// Create sends POST .../availabilityzoneprofiles and expects 201. The request is
// sent once; an uncertain failure is never replayed.
func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*AvailabilityZoneProfile, error) {
	if err := a.check(ctx); err != nil {
		return nil, wrap("Create", "", err)
	}
	config, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, false)
	}
	if err == nil && (strings.TrimSpace(config.Options.Name) == "" || strings.TrimSpace(config.Options.ProviderName) == "" || config.Options.AvailabilityZoneData == "") {
		err = fmt.Errorf("%w: profile name, provider_name and availability_zone_data are required", resource.ErrInvalidOption)
	}
	var body map[string]any
	if err == nil {
		body, err = envelope("availability_zone_profile", config.Options, config.Fields)
	}
	if err != nil {
		return nil, wrap("Create", "", err)
	}
	return a.single("Create", "", http.MethodPost, a.endpoint(), body, 201)(ctx)
}

// UpdateOpts changes mutable fields. Nil fields are omitted; empty strings are
// sent. At least one field or extension is required; Python silently skips the
// request when nothing changed, which Go reports as an invalid option.
type UpdateOpts struct {
	Name                 *string `json:"name,omitempty"`
	ProviderName         *string `json:"provider_name,omitempty"`
	AvailabilityZoneData *string `json:"availability_zone_data,omitempty"`
}

type UpdateOption = request.Option[UpdateOpts]

func WithUpdateOptions(options UpdateOpts) UpdateOption {
	snapshot := cloneUpdate(options)
	return func(c *request.Config[UpdateOpts]) error { c.Options = cloneUpdate(snapshot); return nil }
}

func cloneUpdate(o UpdateOpts) UpdateOpts {
	for _, p := range []**string{&o.Name, &o.ProviderName, &o.AvailabilityZoneData} {
		if *p != nil {
			v := **p
			*p = &v
		}
	}
	return o
}

func WithUpdateField(key string, value any) UpdateOption {
	return request.WithField[UpdateOpts](key, value)
}

// Update sends PUT .../availabilityzoneprofiles/{id} and expects 200.
func (a *API) Update(ctx context.Context, id string, opts UpdateOpts, options ...UpdateOption) (*AvailabilityZoneProfile, error) {
	if err := a.check(ctx); err != nil {
		return nil, wrap("Update", "", err)
	}
	if err := validateID(id); err != nil {
		return nil, wrap("Update", "", err)
	}
	config, err := request.Apply(opts, options...)
	if err == nil {
		err = request.ValidateCapabilities(config, true, false, false)
	}
	var body map[string]any
	if err == nil {
		body, err = envelope("availability_zone_profile", config.Options, config.Fields)
	}
	if err == nil && len(body["availability_zone_profile"].(map[string]any)) == 0 {
		err = fmt.Errorf("%w: availability zone profile update requires at least one field", resource.ErrInvalidOption)
	}
	if err != nil {
		return nil, wrap("Update", "", err)
	}
	return a.single("Update", id, http.MethodPut, a.endpoint(url.PathEscape(id)), body, 200)(ctx)
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

// ListOpts filters profiles. Nil pointers and empty strings are omitted; use the
// pointer fields to send false or an empty-string filter.
type ListOpts struct {
	ID                   *string
	Name                 *string
	ProviderName         *string
	AvailabilityZoneData *string
	Fields               []string
	Limit                int
	Marker               string
	PageReverse          *bool
}

type ListOption = request.Option[ListOpts]

func WithListOptions(options ListOpts) ListOption {
	snapshot := cloneList(options)
	return func(c *request.Config[ListOpts]) error { c.Options = cloneList(snapshot); return nil }
}

func cloneList(o ListOpts) ListOpts {
	o.Fields = append([]string(nil), o.Fields...)
	for _, p := range []**string{&o.ID, &o.Name, &o.ProviderName, &o.AvailabilityZoneData} {
		if *p != nil {
			v := **p
			*p = &v
		}
	}
	for _, p := range []**bool{&o.PageReverse} {
		if *p != nil {
			v := **p
			*p = &v
		}
	}
	return o
}

func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

var coreQuery = map[string]bool{"id": true, "name": true, "provider_name": true, "availability_zone_data": true, "fields": true, "limit": true, "marker": true, "page_reverse": true}

// List streams profiles across pages (GET, 200), following availability_zone_profiles_links.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*AvailabilityZoneProfile, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*AvailabilityZoneProfile, error) bool) {
		if err := a.check(ctx); err != nil {
			yield(nil, wrap("List", "", err))
			return
		}
		config, err := request.Apply(ListOpts{}, options...)
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
	for key, value := range map[string]*string{"id": o.ID, "name": o.Name, "provider_name": o.ProviderName, "availability_zone_data": o.AvailabilityZoneData} {
		if value != nil {
			q.Set(key, *value)
		}
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

func (a *API) iterate(ctx context.Context, op string, query url.Values, control resource.ListControl) iter.Seq2[*AvailabilityZoneProfile, error] {
	return func(yield func(*AvailabilityZoneProfile, error) bool) {
		if err := a.check(ctx); err != nil {
			yield(nil, wrap(op, "", err))
			return
		}
		first := a.endpoint()
		if len(query) > 0 {
			first += "?" + query.Encode()
		}
		count := 0
		for row, err := range octaviarest.Pages(ctx, a.client, first, "availability_zone_profiles", control.SinglePage) {
			if err == nil {
				var value *AvailabilityZoneProfile
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
