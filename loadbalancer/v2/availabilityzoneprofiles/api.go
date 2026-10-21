// Package availabilityzoneprofiles is an SDK-owned binding for Octavia
// availability zone profiles (/lbaas/availabilityzoneprofiles, API 2.14+).
// Gophercloud v2.15.0 has no binding for these routes. Profiles are addressed
// by UUID and every operation needs administrative privileges.
package availabilityzoneprofiles

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/octaviarest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const kind = "availability zone profiles"

// AvailabilityZoneProfile is the declared view of one profile. Body keeps every
// response field exactly as sent so absent, null and empty values stay
// distinguishable. AvailabilityZoneData is the profile's JSON string, unparsed.
type AvailabilityZoneProfile struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	ProviderName         string `json:"provider_name"`
	AvailabilityZoneData string `json:"availability_zone_data"`

	Body       map[string]json.RawMessage `json:"-"`
	Header     http.Header                `json:"-"`
	StatusCode int                        `json:"-"`
}

// API owns the availability zone profile routes. Resources provides the shared
// Find/All/Remove lookup policies keyed by profile ID.
type API struct {
	client    *gophercloud.ServiceClient
	Resources *resource.Collection[AvailabilityZoneProfile]
}

func New(client *gophercloud.ServiceClient) *API {
	a := &API{client: client}
	a.Resources = resource.NewCollection(resource.Adapter[AvailabilityZoneProfile]{
		Kind:         kind,
		IdentityFind: true,
		Get:          func(ctx context.Context, id string) (*AvailabilityZoneProfile, error) { return a.Get(ctx, id) },
		ID:           func(v *AvailabilityZoneProfile) string { return v.ID },
		Name:         func(v *AvailabilityZoneProfile) string { return v.Name },
		NameQuery:    func(name string) string { return name },
		Delete:       func(ctx context.Context, id string) error { return a.Delete(ctx, id) },
		IterateControlled: func(ctx context.Context, q url.Values, control resource.ListControl) iter.Seq2[*AvailabilityZoneProfile, error] {
			return a.iterate(ctx, "List", q, control)
		},
	})
	return a
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

func (a *API) endpoint(parts ...string) string {
	return octaviarest.Endpoint(a.client, append([]string{"availabilityzoneprofiles"}, parts...)...)
}

func (a *API) check(ctx context.Context) error {
	if a == nil || a.client == nil || ctx == nil {
		return fmt.Errorf("%w: client and context are required", resource.ErrInvalidOption)
	}
	return nil
}

func wrap(op, id string, err error) error {
	if gophercloud.ResponseCodeIs(err, http.StatusNotFound) && id != "" {
		err = &resource.NotFoundError{Resource: "availability zone profile", Reference: id, Cause: err}
	}
	return request.Wrap(op, kind, err)
}

func decode(raw json.RawMessage, header http.Header, status int) (*AvailabilityZoneProfile, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, gophercloud.ErrUnexpectedType{Expected: "availability zone profile JSON object", Actual: "null"}
	}
	var value AvailabilityZoneProfile
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	value.Body, value.Header, value.StatusCode = fields, header.Clone(), status
	return &value, nil
}

func (a *API) single(op, name, method, endpoint string, body any, accepted int) func(context.Context) (*AvailabilityZoneProfile, error) {
	return func(ctx context.Context) (*AvailabilityZoneProfile, error) {
		wire, err := octaviarest.Do(ctx, a.client, method, endpoint, body, []int{accepted})
		if err != nil {
			return nil, wrap(op, name, err)
		}
		raw, err := octaviarest.Object(wire.Body, "availability_zone_profile")
		if err == nil {
			var value *AvailabilityZoneProfile
			if value, err = decode(raw, wire.Header, wire.Status); err == nil {
				return value, nil
			}
		}
		return nil, wrap(op, name, err)
	}
}

func validateID(id string) error {
	return resource.ID(id).Validate()
}

// Get fetches one profile by ID (GET .../availabilityzoneprofiles/{id}, 200).
func (a *API) Get(ctx context.Context, id string) (*AvailabilityZoneProfile, error) {
	if err := a.check(ctx); err != nil {
		return nil, wrap("Get", "", err)
	}
	if err := validateID(id); err != nil {
		return nil, wrap("Get", "", err)
	}
	return a.single("Get", id, http.MethodGet, a.endpoint(url.PathEscape(id)), nil, 200)(ctx)
}

// Delete removes one profile by ID (DELETE, 204). A missing profile is a
// NotFoundError; use Remove for ignore-missing semantics. A profile still used
// by an availability zone yields the server's 409.
func (a *API) Delete(ctx context.Context, id string) error {
	if err := a.check(ctx); err != nil {
		return wrap("Delete", "", err)
	}
	if err := validateID(id); err != nil {
		return wrap("Delete", "", err)
	}
	_, err := octaviarest.Do(ctx, a.client, http.MethodDelete, a.endpoint(url.PathEscape(id)), nil, []int{204})
	return wrap("Delete", id, err)
}

// Find resolves a profile by explicit ID or exact name.
func (a *API) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*AvailabilityZoneProfile, error) {
	return a.Resources.Find(ctx, ref, options...)
}

// FindIdentity mirrors Python find_availability_zone_profile: GET by ID first, then
// an exact-name list. A duplicate list match is an error.
func (a *API) FindIdentity(ctx context.Context, identity string, options ...resource.IdentityFindOption) (*AvailabilityZoneProfile, error) {
	return a.Resources.FindIdentity(ctx, identity, options...)
}

func (a *API) All(ctx context.Context, options ...resource.ListOption) ([]*AvailabilityZoneProfile, error) {
	return a.Resources.All(ctx, options...)
}

func (a *API) Remove(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) error {
	return a.Resources.Delete(ctx, ref, options...)
}
