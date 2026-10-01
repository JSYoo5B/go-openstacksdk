// Package policies manages Senlin policy definitions and validates policy specs.
package policies

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
	Resources *resource.Collection[Policy]
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

func spec(client *gophercloud.ServiceClient) rest.CollectionSpec[Policy] {
	return rest.CollectionSpec[Policy]{
		Client: client, Path: "policies", Kind: "clustering.policies",
		SingleKey: "policy", PluralKey: "policies", Get: true, Delete: true,
		GetCodes: []int{http.StatusOK}, ListCodes: []int{http.StatusOK}, DeleteCodes: []int{http.StatusNoContent},
		ID:            func(value *Policy) string { return value.ID },
		Name:          func(value *Policy) string { return value.Name },
		NameQuery:     func(name string) string { return name },
		Metadata:      func(value *Policy) *resource.Metadata { return &value.Metadata },
		Validate:      func(ctx context.Context) error { return senlin.Validate(ctx, client) },
		ValidateID:    senlin.Identifier,
		ValidateQuery: func(_ context.Context, query url.Values) error { return validateSort(query.Get("sort")) },
		Paging: rest.PagePolicy[Policy]{HTTPLink: true, MarkerFallback: true, MarkerOnShortPage: true,
			Marker: func(value *Policy) (string, error) {
				if value == nil {
					return "", fmt.Errorf("%w: policy pagination row is missing", resource.ErrInvalidOption)
				}
				return value.ID, senlin.Identifier(value.ID)
			}},
	}
}

// Get accepts a name, UUID or short ID directly, as the Senlin controller does.
func (a *API) Get(ctx context.Context, identity string) (*Policy, error) {
	return rest.Collection(spec(a.RawClient())).Get(ctx, identity)
}

// Find resolves an explicit ID or exact name. Like Python find_policy, missing
// resources return nil by default; WithMissingError makes them errors.
func (a *API) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*Policy, error) {
	options = append([]resource.LookupOption{resource.WithIgnoreMissing()}, options...)
	return rest.Collection(spec(a.RawClient())).Find(ctx, ref, options...)
}

// Delete ignores missing policies by default and preserves conflicts and other
// HTTP errors. A name is resolved once before deleting the response ID.
func (a *API) Delete(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) error {
	return rest.Collection(spec(a.RawClient())).Delete(ctx, ref, options...)
}

// List is lazy and reusable. A consumer break stops fetching additional pages.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Policy, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*Policy, error) bool) {
		config, err := request.Apply(ListOpts{}, options...)
		var query url.Values
		var filters map[string]json.RawMessage
		if err == nil {
			query, err = listQuery(config)
		}
		if err == nil {
			filters, err = prepareFilters(config)
		}
		if err != nil {
			yield(nil, request.Wrap("List", "clustering.policies", err))
			return
		}
		for value, err := range rest.List(ctx, spec(a.RawClient()), query) {
			if err != nil {
				yield(nil, request.Wrap("List", "clustering.policies", err))
				return
			}
			matched, err := senlin.MatchFilters(value.Body, filters)
			if err != nil {
				yield(nil, request.Wrap("List", "clustering.policies", err))
				return
			}
			if matched && !yield(value, nil) {
				return
			}
		}
	}
}

func (a *API) All(ctx context.Context, options ...ListOption) ([]*Policy, error) {
	return senlin.All(a.List(ctx, options...))
}

// Policy retains raw spec/runtime values without float64 conversion. Null and
// omitted fields remain distinguishable in Metadata.Body, including null IDs.
type Policy struct {
	resource.Metadata
	ID        string                     `json:"id"`
	Name      string                     `json:"name"`
	Type      string                     `json:"type"`
	ProjectID string                     `json:"project"`
	DomainID  string                     `json:"domain"`
	UserID    string                     `json:"user"`
	Spec      map[string]json.RawMessage `json:"spec"`
	Data      map[string]json.RawMessage `json:"data"`
}

func (value *Policy) UnmarshalJSON(data []byte) error {
	type plain Policy
	return resource.DecodeObject(data, (*plain)(value), &value.Metadata)
}
