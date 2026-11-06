// Package policytypes reads the Senlin policy type catalog and schemas.
package policytypes

import (
	"context"
	"encoding/json"
	"iter"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/senlin"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type API struct {
	client    *gophercloud.ServiceClient
	Resources *resource.Collection[PolicyType]
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

type PolicyType struct {
	resource.Metadata
	Name          string                     `json:"name"`
	Version       *string                    `json:"version"`
	Schema        map[string]json.RawMessage `json:"schema"`
	SupportStatus json.RawMessage            `json:"support_status"`
}

func (value *PolicyType) UnmarshalJSON(data []byte) error {
	type plain PolicyType
	return resource.DecodeObject(data, (*plain)(value), &value.Metadata)
}

func spec(client *gophercloud.ServiceClient) rest.CollectionSpec[PolicyType] {
	return rest.CollectionSpec[PolicyType]{
		Client: client, Path: "policy-types", Kind: "clustering.policy_types",
		SingleKey: "policy_type", PluralKey: "policy_types", Get: true,
		GetCodes: []int{http.StatusOK}, ListCodes: []int{http.StatusOK},
		ID:         func(value *PolicyType) string { return value.Name },
		Name:       func(value *PolicyType) string { return value.Name },
		Metadata:   func(value *PolicyType) *resource.Metadata { return &value.Metadata },
		Validate:   func(ctx context.Context) error { return senlin.Validate(ctx, client) },
		ValidateID: senlin.Identifier,
		Paging:     rest.PagePolicy[PolicyType]{HTTPLink: true, MaxItemsLimitHint: true, StopOnEmptyPage: true},
	}
}

// Get uses the exact policy type name, including its dotted plugin suffix.
func (a *API) Get(ctx context.Context, name string) (*PolicyType, error) {
	return rest.Collection(spec(a.RawClient())).Get(ctx, name)
}

type ListOpts = senlin.ListOpts

// ListOption is one option of List and All: WithListOptions, WithListFilter,
// request.Header, request.Microversion, request.Query (a string or []string
// value), request.MaxItems, request.Paginated or request.Custom over
// *request.Config[ListOpts]. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type ListOption = request.ListCall

func WithListOptions(value ListOpts) ListOption { return request.Custom(senlin.Snapshot(value)) }

// List follows advertised same-path links without a guessed marker fallback.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*PolicyType, error] {
	return senlin.ListWithClientBodyFilters(ctx, a.RawClient(), spec, bodyFilterSpec(), calls.Options(options, senlin.SharedListCalls)...)
}

func (a *API) All(ctx context.Context, options ...ListOption) ([]*PolicyType, error) {
	return senlin.All(a.List(ctx, options...))
}
