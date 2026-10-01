package clusters

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
	Resources *resource.Collection[Cluster]
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

func spec(client *gophercloud.ServiceClient) rest.CollectionSpec[Cluster] {
	return rest.CollectionSpec[Cluster]{
		Client: client, Path: "clusters", Kind: "clustering.clusters", SingleKey: "cluster", PluralKey: "clusters", Get: true,
		GetCodes: []int{http.StatusOK}, ListCodes: []int{http.StatusOK},
		ID: func(value *Cluster) string { return value.ID }, Name: func(value *Cluster) string { return value.Name },
		Status:        func(value *Cluster) string { return value.Status },
		NameQuery:     func(name string) string { return name },
		Metadata:      func(value *Cluster) *resource.Metadata { return &value.Metadata },
		Validate:      func(ctx context.Context) error { return senlin.Validate(ctx, client) },
		ValidateQuery: func(ctx context.Context, query url.Values) error { return senlin.SortGrammar(query.Get("sort")) },
		ValidateID:    senlin.Identifier,
		Paging: rest.PagePolicy[Cluster]{MaxItemsLimitHint: true, StopOnEmptyPage: true, HTTPLink: true, MarkerFallback: true, MarkerOnShortPage: true,
			Marker: func(value *Cluster) (string, error) {
				if value == nil {
					return "", fmt.Errorf("%w: missing cluster pagination row", resource.ErrInvalidOption)
				}
				return value.ID, senlin.Identifier(value.ID)
			}},
	}
}

// Get sends the supplied name, UUID or short ID directly to Senlin.
func (a *API) Get(ctx context.Context, identity string) (*Cluster, error) {
	return rest.Collection(spec(a.RawClient())).Get(ctx, identity)
}

// Find defaults to ignoring absence like the pinned Python proxy. Resources.Find
// keeps the shared strict default; caller options may override either policy.
func (a *API) Find(ctx context.Context, ref resource.Ref, options ...resource.LookupOption) (*Cluster, error) {
	options = append([]resource.LookupOption{resource.WithIgnoreMissing()}, options...)
	return rest.Collection(spec(a.RawClient())).Find(ctx, ref, options...)
}

func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Cluster, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*Cluster, error) bool) {
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
			yield(nil, request.Wrap("List", "clustering.clusters", err))
			return
		}
		for value, err := range rest.ListWithControl(ctx, spec(a.RawClient()), query, rest.ListControl{MaxItems: config.Options.MaxItems, SinglePage: config.Options.Paginated != nil && !*config.Options.Paginated, LimitHint: true}) {
			if err != nil {
				yield(nil, request.Wrap("List", "clustering.clusters", err))
				return
			}
			matched, err := senlin.MatchFilters(value.Body, filters)
			if err != nil {
				yield(nil, request.Wrap("List", "clustering.clusters", err))
				return
			}
			if matched && !yield(value, nil) {
				return
			}
		}
	}
}

func (a *API) All(ctx context.Context, options ...ListOption) ([]*Cluster, error) {
	return senlin.All(a.List(ctx, options...))
}
