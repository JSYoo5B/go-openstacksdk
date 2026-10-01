// Package profiletypes reads Senlin profile schemas and operation catalogs.
package profiletypes

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type API struct {
	client    *gophercloud.ServiceClient
	Resources *resource.Collection[ProfileType]
}

func New(client *gophercloud.ServiceClient) *API {
	return &API{client: client, Resources: rest.Collection(spec(client))}
}

func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

// ProfileType is identified by its exact Name. Version is independently
// returned by newer list responses; the SDK never invents a combined name.
type ProfileType struct {
	resource.Metadata
	Name          string                     `json:"name"`
	Version       *string                    `json:"version"`
	Schema        map[string]json.RawMessage `json:"schema"`
	SupportStatus json.RawMessage            `json:"support_status"`
}

func (value *ProfileType) UnmarshalJSON(data []byte) error {
	type plain ProfileType
	return resource.DecodeObject(data, (*plain)(value), &value.Metadata)
}

// Operations contains plugin-defined operations and parameter schemas.
type Operations struct {
	resource.Metadata
	Operations map[string]json.RawMessage `json:"operations"`
}

func (value *Operations) UnmarshalJSON(data []byte) error {
	type plain Operations
	return resource.DecodeObject(data, (*plain)(value), &value.Metadata)
}

func spec(client *gophercloud.ServiceClient) rest.CollectionSpec[ProfileType] {
	return rest.CollectionSpec[ProfileType]{
		Client: client, Path: "profile-types", Kind: "clustering.profile_types",
		SingleKey: "profile_type", PluralKey: "profile_types", Get: true,
		GetCodes: []int{http.StatusOK}, ListCodes: []int{http.StatusOK},
		ID:         func(value *ProfileType) string { return value.Name },
		Name:       func(value *ProfileType) string { return value.Name },
		Metadata:   func(value *ProfileType) *resource.Metadata { return &value.Metadata },
		Validate:   func(ctx context.Context) error { return senlin.Validate(ctx, client) },
		ValidateID: senlin.Identifier,
		Paging:     rest.PagePolicy[ProfileType]{HTTPLink: true, MaxItemsLimitHint: true, StopOnEmptyPage: true},
	}
}

// Get uses the exact type name as a protected URL segment, including dots and
// plugin-version suffixes; it does not resolve a UUID or perform a list first.
func (a *API) Get(ctx context.Context, name string) (*ProfileType, error) {
	return rest.Collection(spec(a.RawClient())).Get(ctx, name)
}

type ListOpts = senlin.ListOpts
type ListOption = senlin.ListOption

func WithListOptions(value ListOpts) ListOption  { return senlin.Snapshot(value) }
func WithListMaxItems(value int) ListOption      { return senlin.WithMaxItems(value) }
func WithListPaginated(value bool) ListOption    { return senlin.WithPaginated(value) }
func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

// List follows advertised continuation links. It never fabricates a marker
// from type names, because the type catalog documents no marker protocol.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*ProfileType, error] {
	return senlin.ListWithBodyFilters(ctx, spec(a.RawClient()), bodyFilterSpec(), options...)
}

func (a *API) All(ctx context.Context, options ...ListOption) ([]*ProfileType, error) {
	return senlin.All(a.List(ctx, options...))
}

// Operations requires a selected numeric microversion >=1.4 and makes one
// request. It never upgrades the shared client or fetches the type first.
func (a *API) Operations(ctx context.Context, name string) (*Operations, error) {
	client := a.RawClient()
	if err := senlin.RequireVersion(ctx, client, 4); err != nil {
		return nil, request.Wrap("Operations", "clustering.profile_types", err)
	}
	if err := senlin.Identifier(name); err != nil {
		return nil, request.Wrap("Operations", "clustering.profile_types", err)
	}
	response, err := rest.DoJSON(ctx, client, http.MethodGet, client.ServiceURL("profile-types", url.PathEscape(name), "ops"), nil, nil, http.StatusOK)
	if err != nil {
		return nil, request.Wrap("Operations", "clustering.profile_types", err)
	}
	value, err := rest.Decode(response, "", func(value *Operations) *resource.Metadata { return &value.Metadata })
	if err == nil && value.Operations == nil {
		err = response.Fail(fmt.Errorf("response requires an operations object"))
		value = nil
	}
	return value, request.Wrap("Operations", "clustering.profile_types", err)
}
