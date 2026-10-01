// Package services lists Senlin engine services, available since version 1.7.
package services

import (
	"context"
	"iter"
	"net/http"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type API struct {
	client    *gophercloud.ServiceClient
	Resources *resource.Collection[Service]
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

type Service struct {
	resource.Metadata
	ID             string  `json:"id"`
	Status         string  `json:"status"`
	State          string  `json:"state"`
	Binary         string  `json:"binary"`
	DisabledReason *string `json:"disabled_reason"`
	Host           string  `json:"host"`
	Topic          string  `json:"topic"`
}

func (value *Service) UnmarshalJSON(data []byte) error {
	type plain Service
	return resource.DecodeObject(data, (*plain)(value), &value.Metadata)
}

func spec(client *gophercloud.ServiceClient) rest.CollectionSpec[Service] {
	return rest.CollectionSpec[Service]{
		Client: client, Path: "services", Kind: "clustering.services",
		SingleKey: "service", PluralKey: "services", ListCodes: []int{http.StatusOK},
		ID:     func(value *Service) string { return value.ID },
		Status: func(value *Service) string { return value.Status }, LocalStatus: true,
		Metadata:   func(value *Service) *resource.Metadata { return &value.Metadata },
		Validate:   func(ctx context.Context) error { return senlin.RequireVersion(ctx, client, 7) },
		ValidateID: senlin.Identifier,
		Paging:     rest.PagePolicy[Service]{HTTPLink: true, MaxItemsLimitHint: true, StopOnEmptyPage: true},
	}
}

type ListOpts = senlin.ListOpts
type ListOption = senlin.ListOption

func WithListOptions(value ListOpts) ListOption  { return senlin.Snapshot(value) }
func WithListMaxItems(value int) ListOption      { return senlin.WithMaxItems(value) }
func WithListPaginated(value bool) ListOption    { return senlin.WithPaginated(value) }
func WithListQuery(key, value string) ListOption { return request.WithQuery[ListOpts](key, value) }

// List has no marker fallback: the official service list declares no paging
// parameters. Numeric microversion >=1.7 is checked before each request.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Service, error] {
	return senlin.List(ctx, spec(a.RawClient()), options...)
}

func (a *API) All(ctx context.Context, options ...ListOption) ([]*Service, error) {
	return senlin.All(a.List(ctx, options...))
}
