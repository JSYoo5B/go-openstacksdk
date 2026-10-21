// Package availabilityzones is an SDK-owned binding for Octavia availability
// zones (/lbaas/availabilityzones, API 2.14+). Gophercloud v2.15.0 has no
// binding for these routes. Zones are addressed by name, not by UUID.
package availabilityzones

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

const kind = "availability zones"

// AvailabilityZone is the declared view of one zone. Body keeps every response
// field exactly as sent so absent, null and false stay distinguishable.
type AvailabilityZone struct {
	Name                      string `json:"name"`
	Description               string `json:"description"`
	Enabled                   bool   `json:"enabled"`
	AvailabilityZoneProfileID string `json:"availability_zone_profile_id"`

	Body       map[string]json.RawMessage `json:"-"`
	Header     http.Header                `json:"-"`
	StatusCode int                        `json:"-"`
}

// API owns the availability zone routes. Resources provides the shared
// Find/All/Remove lookup policies keyed by zone name.
type API struct {
	client    *gophercloud.ServiceClient
	Resources *resource.Collection[AvailabilityZone]
}

func New(client *gophercloud.ServiceClient) *API {
	a := &API{client: client}
	a.Resources = resource.NewCollection(resource.Adapter[AvailabilityZone]{
		Kind:         kind,
		IdentityFind: true,
		Get:          func(ctx context.Context, id string) (*AvailabilityZone, error) { return a.Get(ctx, id) },
		ID:           func(v *AvailabilityZone) string { return v.Name },
		Name:         func(v *AvailabilityZone) string { return v.Name },
		NameQuery:    func(name string) string { return name },
		Delete:       func(ctx context.Context, id string) error { return a.Delete(ctx, id) },
		IterateControlled: func(ctx context.Context, q url.Values, control resource.ListControl) iter.Seq2[*AvailabilityZone, error] {
			return a.iterate(ctx, "List", q, control)
		},
	})
	return a
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

func (a *API) endpoint(parts ...string) string {
	return octaviarest.Endpoint(a.client, append([]string{"availabilityzones"}, parts...)...)
}

func (a *API) check(ctx context.Context) error {
	if a == nil || a.client == nil || ctx == nil {
		return fmt.Errorf("%w: client and context are required", resource.ErrInvalidOption)
	}
	return nil
}

func wrap(op, name string, err error) error {
	if gophercloud.ResponseCodeIs(err, http.StatusNotFound) && name != "" {
		err = &resource.NotFoundError{Resource: "availability zone", Reference: name, Cause: err}
	}
	return request.Wrap(op, kind, err)
}

func decode(raw json.RawMessage, header http.Header, status int) (*AvailabilityZone, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, gophercloud.ErrUnexpectedType{Expected: "availability zone JSON object", Actual: "null"}
	}
	var value AvailabilityZone
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	value.Body, value.Header, value.StatusCode = fields, header.Clone(), status
	return &value, nil
}

func (a *API) single(op, name, method, endpoint string, body any, accepted int) func(context.Context) (*AvailabilityZone, error) {
	return func(ctx context.Context) (*AvailabilityZone, error) {
		wire, err := octaviarest.Do(ctx, a.client, method, endpoint, body, []int{accepted})
		if err != nil {
			return nil, wrap(op, name, err)
		}
		raw, err := octaviarest.Object(wire.Body, "availability_zone")
		if err == nil {
			var value *AvailabilityZone
			if value, err = decode(raw, wire.Header, wire.Status); err == nil {
				return value, nil
			}
		}
		return nil, wrap(op, name, err)
	}
}

func validateName(name string) error {
	return resource.ID(name).Validate()
}

// Get fetches one zone by name (GET .../availabilityzones/{name}, 200).
func (a *API) Get(ctx context.Context, name string) (*AvailabilityZone, error) {
	if err := a.check(ctx); err != nil {
		return nil, wrap("Get", "", err)
	}
	if err := validateName(name); err != nil {
		return nil, wrap("Get", "", err)
	}
	return a.single("Get", name, http.MethodGet, a.endpoint(url.PathEscape(name)), nil, 200)(ctx)
}

// Delete removes one zone by name (DELETE, 204). A missing zone is a
// NotFoundError; use Resources.Delete or Remove for ignore-missing semantics.
// A zone still used by load balancers yields the server's 409.
func (a *API) Delete(ctx context.Context, name string) error {
	if err := a.check(ctx); err != nil {
		return wrap("Delete", "", err)
	}
	if err := validateName(name); err != nil {
		return wrap("Delete", "", err)
	}
	_, err := octaviarest.Do(ctx, a.client, http.MethodDelete, a.endpoint(url.PathEscape(name)), nil, []int{204})
	return wrap("Delete", name, err)
}

// Find resolves a zone by explicit name or ID reference. Names and IDs are the
// same string for availability zones.
func (a *API) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*AvailabilityZone, error) {
	return a.Resources.Find(ctx, ref, options...)
}

// FindIdentity mirrors Python find_availability_zone: GET by name first, then
// an exact-name list. A duplicate list match is an error.
func (a *API) FindIdentity(ctx context.Context, identity string, options ...resource.IdentityFindOption) (*AvailabilityZone, error) {
	return a.Resources.FindIdentity(ctx, identity, options...)
}

func (a *API) All(ctx context.Context, options ...resource.ListOption) ([]*AvailabilityZone, error) {
	return a.Resources.All(ctx, options...)
}

func (a *API) Remove(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) error {
	return a.Resources.Delete(ctx, ref, options...)
}
