package softwaredeployments

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/internal/swcommon"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const kind = "software_deployments"

// API manages /software_deployments.
type API struct {
	client     *gophercloud.ServiceClient
	spec       rest.CollectionSpec[SoftwareDeployment]
	collection *resource.Collection[SoftwareDeployment]
}

func New(client *gophercloud.ServiceClient) *API {
	spec := rest.CollectionSpec[SoftwareDeployment]{
		Client: client, Path: "software_deployments", Kind: kind, SingleKey: "software_deployment", PluralKey: "software_deployments",
		ID:       func(c *SoftwareDeployment) string { return c.ID },
		Metadata: func(c *SoftwareDeployment) *resource.Metadata { return &c.Metadata },
		Get:      true, Delete: true, DeleteCodes: []int{204},
		Paging: rest.PagePolicy[SoftwareDeployment]{SourceLinkPolicy: true, IgnoreFalseyNext: true, MarkerFallback: true,
			Marker: func(c *SoftwareDeployment) (string, error) { return c.ID, nil }, MarkerOnShortPage: true, StopOnEmptyPage: true},
	}
	return &API{client: client, spec: spec, collection: rest.Collection(spec)}
}

func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

func (a *API) validate(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if a == nil || a.client == nil || a.client.ProviderClient == nil {
		return fmt.Errorf("%w: Heat service client is required", resource.ErrInvalidOption)
	}
	return ctx.Err()
}

// Create POSTs the unwrapped deployment body once and accepts HTTP 200 or
// 201. Omitted members are not sent; null, empty and false-like values are.
// A failed or uncertain request is never replayed.
func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*SoftwareDeployment, error) {
	if err := a.validate(ctx); err != nil {
		return nil, request.Wrap("Create", kind, err)
	}
	config, err := request.Apply(opts, options...)
	var body []byte
	if err == nil {
		body, err = swcommon.Body(config, "id", "creation_time", "updated_time")
	}
	if err != nil {
		return nil, request.Wrap("Create", kind, err)
	}
	response, err := rest.DoJSON(ctx, a.client, "POST", a.client.ServiceURL("software_deployments"), json.RawMessage(body), config.Headers, 200, 201)
	if err != nil {
		return nil, request.Wrap("Create", kind, err)
	}
	value, err := rest.Decode(response, "software_deployment", a.spec.Metadata)
	return value, request.Wrap("Create", kind, err)
}

// Update PUTs only the members that are set (Python commits dirty attributes
// only) to /software_deployments/{id} and accepts HTTP 200. An update with no
// member to send fails before HTTP; Python silently sends nothing. The request
// is sent once and never replayed after an uncertain failure.
func (a *API) Update(ctx context.Context, id string, opts UpdateOpts, options ...UpdateOption) (*SoftwareDeployment, error) {
	if err := a.validate(ctx); err != nil {
		return nil, request.Wrap("Update", kind, err)
	}
	if err := resource.ID(id).Validate(); err != nil {
		return nil, request.Wrap("Update", kind, err)
	}
	config, err := request.Apply(opts, options...)
	var body []byte
	if err == nil {
		body, err = swcommon.Body(config, "id", "creation_time", "updated_time")
	}
	if err == nil && string(body) == "{}" {
		err = fmt.Errorf("%w: update requires at least one member", resource.ErrInvalidOption)
	}
	if err != nil {
		return nil, request.Wrap("Update", kind, err)
	}
	response, err := rest.DoJSON(ctx, a.client, "PUT", a.client.ServiceURL("software_deployments", url.PathEscape(id)), json.RawMessage(body), config.Headers, 200)
	if err != nil {
		return nil, request.Wrap("Update", kind, err)
	}
	value, err := rest.Decode(response, "software_deployment", a.spec.Metadata)
	return value, request.Wrap("Update", kind, err)
}

// Get reads one deployment by ID.
func (a *API) Get(ctx context.Context, id string) (*SoftwareDeployment, error) {
	if err := a.validate(ctx); err != nil {
		return nil, request.Wrap("Get", kind, err)
	}
	return a.collection.Get(ctx, id)
}

// Delete removes one deployment by ID. A missing deployment is ignored
// by default, like Python's ignore_missing=True; resource.WithMissingError
// reports it. Success is HTTP 204.
func (a *API) Delete(ctx context.Context, id string, options ...resource.LookupOption) error {
	if err := a.validate(ctx); err != nil {
		return request.Wrap("Delete", kind, err)
	}
	return a.collection.Delete(ctx, resource.ID(id), options...)
}

// List streams deployment summaries. Without a limit only the first page is
// read. With a limit, pages continue by marker until an empty page.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*SoftwareDeployment, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*SoftwareDeployment, error) bool) {
		if err := a.validate(ctx); err != nil {
			yield(nil, request.Wrap("List", kind, err))
			return
		}
		config, err := request.Apply(ListOpts{}, options...)
		query := url.Values{}
		if err == nil {
			err = swcommon.Paging(query, config.Options.Limit, config.Options.Marker)
		}
		if err == nil && config.Options.ServerID != nil {
			if *config.Options.ServerID == "" {
				err = fmt.Errorf("%w: server_id filter must not be empty", resource.ErrInvalidOption)
			} else {
				query.Set("server_id", *config.Options.ServerID)
			}
		}
		if err == nil {
			query, err = swcommon.ListQuery(config, query, "limit", "marker", "server_id")
		}
		if err != nil {
			yield(nil, request.Wrap("List", kind, err))
			return
		}
		for value, err := range rest.List(ctx, a.spec, query) {
			if err != nil {
				yield(nil, request.Wrap("List", kind, err))
				return
			}
			if !yield(value, nil) {
				return
			}
		}
	}
}

func (a *API) All(ctx context.Context, options ...ListOption) ([]*SoftwareDeployment, error) {
	values := make([]*SoftwareDeployment, 0)
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
