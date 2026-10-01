package segments

import (
	"context"
	"iter"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/masakari"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type API struct {
	*resource.Collection[Segment]
	Resources *resource.Collection[Segment]
	client    *gophercloud.ServiceClient
	spec      rest.CollectionSpec[Segment]
}

// New performs no HTTP. Segment UUIDs, rather than database IDs, identify routes.
func New(client *gophercloud.ServiceClient) *API {
	spec := rest.CollectionSpec[Segment]{
		Client: client, Path: "segments", Kind: "segments", SingleKey: "segment", PluralKey: "segments",
		ID: func(s *Segment) string { return s.UUID }, Name: func(s *Segment) string { return s.Name },
		Metadata: func(s *Segment) *resource.Metadata { return &s.Metadata },
		Validate: func(ctx context.Context) error { return masakari.Require(ctx, client, 0) },
		ValidateQuery: func(ctx context.Context, query url.Values) error {
			minimum := 0
			if query.Has("enabled") {
				minimum = 2
			}
			return masakari.Require(ctx, client, minimum)
		},
		ValidateID: masakari.UUID, Get: true, Delete: true,
		Paging: rest.PagePolicy[Segment]{HTTPLink: true},
	}
	collection := rest.Collection(spec)
	return &API{Collection: collection, Resources: collection, client: client, spec: spec}
}

func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*Segment, error] {
	return func(yield func(*Segment, error) bool) {
		query, err := prepareList(options...)
		if err != nil {
			yield(nil, request.Wrap("List", "segments", err))
			return
		}
		for value, err := range rest.List(ctx, a.spec, query) {
			if err != nil {
				yield(nil, request.Wrap("List", "segments", err))
				return
			}
			if !yield(value, nil) {
				return
			}
		}
	}
}

// All consumes List with the same typed options. Shared local limits remain
// available through Resources.All with resource.ListOption.
func (a *API) All(ctx context.Context, options ...ListOption) ([]*Segment, error) {
	var values []*Segment
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (a *API) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*Segment, error) {
	if err := masakari.Require(ctx, a.client, 0); err != nil {
		return nil, request.Wrap("Create", "segments", err)
	}
	config, err := request.Apply(opts, options...)
	if err != nil {
		return nil, request.Wrap("Create", "segments", err)
	}
	if err := masakari.Required(config.Options.Name, config.Options.RecoveryMethod, config.Options.ServiceType); err != nil {
		return nil, request.Wrap("Create", "segments", err)
	}
	if config.Options.Enabled != nil {
		if err := masakari.Require(ctx, a.client, 2); err != nil {
			return nil, request.Wrap("Create", "segments", err)
		}
	}
	body, err := masakari.Body(config, "segment", "id", "uuid", "created_at", "updated_at", "deleted_at", "deleted")
	if err != nil {
		return nil, request.Wrap("Create", "segments", err)
	}
	response, err := rest.DoJSON(ctx, a.client, "POST", a.client.ServiceURL("segments"), body, config.Headers, 202)
	if err != nil {
		return nil, request.Wrap("Create", "segments", err)
	}
	value, err := rest.Decode(response, "segment", func(s *Segment) *resource.Metadata { return &s.Metadata })
	return value, request.Wrap("Create", "segments", err)
}

// Update resolves a name once before PUT. Fixed UUID references need no GET.
func (a *API) Update(ctx context.Context, ref resource.Ref, opts UpdateOpts, options ...UpdateOption) (*Segment, error) {
	if err := masakari.Require(ctx, a.client, 0); err != nil {
		return nil, request.Wrap("Update", "segments", err)
	}
	config, err := request.Apply(opts, options...)
	if err != nil {
		return nil, request.Wrap("Update", "segments", err)
	}
	for _, value := range []*string{config.Options.Name, config.Options.RecoveryMethod, config.Options.ServiceType} {
		if value != nil {
			if err := masakari.Required(*value); err != nil {
				return nil, request.Wrap("Update", "segments", err)
			}
		}
	}
	if config.Options.Enabled != nil {
		if err := masakari.Require(ctx, a.client, 2); err != nil {
			return nil, request.Wrap("Update", "segments", err)
		}
	}
	body, err := masakari.Body(config, "segment", "id", "uuid", "created_at", "updated_at", "deleted_at", "deleted")
	if err != nil {
		return nil, request.Wrap("Update", "segments", err)
	}
	id, err := a.Resources.ResolveID(ctx, ref)
	if err != nil {
		return nil, request.Wrap("Update", "segments", err)
	}
	response, err := rest.DoJSON(ctx, a.client, "PUT", a.client.ServiceURL("segments", url.PathEscape(id)), body, config.Headers, 200)
	if err != nil {
		return nil, request.Wrap("Update", "segments", err)
	}
	value, err := rest.Decode(response, "segment", func(s *Segment) *resource.Metadata { return &s.Metadata })
	return value, request.Wrap("Update", "segments", err)
}
