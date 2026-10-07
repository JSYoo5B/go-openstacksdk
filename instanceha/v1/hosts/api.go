package hosts

import (
	"context"
	"fmt"
	"iter"
	"net/url"

	"github.com/JSYoo5B/gophercloudsdk/instanceha/v1/segments"
	"github.com/JSYoo5B/gophercloudsdk/internal/masakari"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API     { return &API{client: client} }
func (a *API) RawClient() *gophercloud.ServiceClient { return a.client }

type SegmentScope struct {
	*resource.Collection[Host]
	Resources *resource.Collection[Host]
	client    *gophercloud.ServiceClient
	segmentID string
	spec      rest.CollectionSpec[Host]
}

// InSegment resolves a name exactly once. Explicit UUIDs require no parent GET.
func (a *API) InSegment(ctx context.Context, ref resource.Ref) (*SegmentScope, error) {
	if err := masakari.Require(ctx, a.client, 0); err != nil {
		return nil, request.Wrap("InSegment", "hosts", err)
	}
	id, err := segments.New(a.client).Resources.ResolveID(ctx, ref)
	if err != nil {
		return nil, request.Wrap("InSegment", "hosts", err)
	}
	scope := &SegmentScope{client: a.client, segmentID: id}
	spec := rest.CollectionSpec[Host]{
		Client: a.client, Path: "segments/" + url.PathEscape(id) + "/hosts", Kind: "hosts", SingleKey: "host", PluralKey: "hosts",
		ID: func(h *Host) string { return h.UUID }, Name: func(h *Host) string { return h.Name },
		Metadata: func(h *Host) *resource.Metadata { h.SegmentID = id; return &h.Metadata },
		Validate: scope.validate, ValidateQuery: scope.validateQuery, ValidateID: masakari.UUID, Get: true, Delete: true,
		Paging: rest.PagePolicy[Host]{HTTPLink: true},
	}
	scope.spec = spec
	scope.Resources = rest.Collection(spec)
	scope.Collection = scope.Resources
	return scope, nil
}

func (s *SegmentScope) SegmentID() string                     { return s.segmentID }
func (s *SegmentScope) RawClient() *gophercloud.ServiceClient { return s.client }
func (s *SegmentScope) validate(ctx context.Context) error {
	if err := masakari.Require(ctx, s.client, 0); err != nil {
		return err
	}
	return masakari.UUID(s.segmentID)
}

func (s *SegmentScope) validateQuery(ctx context.Context, q url.Values) error {
	if err := s.validate(ctx); err != nil {
		return err
	}
	for _, key := range []string{"segment_id", "segment"} {
		if q.Has(key) {
			return fmt.Errorf("%w: %s is the fixed URI parent", resource.ErrInvalidOption, key)
		}
	}
	if values, exists := q["failover_segment_id"]; exists && (len(values) != 1 || values[0] != s.segmentID) {
		return fmt.Errorf("%w: failover_segment_id conflicts with the fixed segment", resource.ErrInvalidOption)
	}
	return nil
}

func (s *SegmentScope) List(ctx context.Context, options ...ListOption) iter.Seq2[*Host, error] {
	options = append([]ListOption(nil), options...)
	return func(yield func(*Host, error) bool) {
		query, err := prepareList(options...)
		if err != nil {
			yield(nil, request.Wrap("List", "hosts", err))
			return
		}
		for value, err := range rest.List(ctx, s.spec, query) {
			if err != nil {
				yield(nil, request.Wrap("List", "hosts", err))
				return
			}
			if !yield(value, nil) {
				return
			}
		}
	}
}

func (s *SegmentScope) All(ctx context.Context, options ...ListOption) ([]*Host, error) {
	values := make([]*Host, 0)
	for value, err := range s.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (s *SegmentScope) Create(ctx context.Context, opts CreateOpts, options ...CreateOption) (*Host, error) {
	if err := s.validate(ctx); err != nil {
		return nil, request.Wrap("Create", "hosts", err)
	}
	config, err := request.Apply(opts, options...)
	if err != nil {
		return nil, request.Wrap("Create", "hosts", err)
	}
	if err := masakari.Required(config.Options.Name, config.Options.Type, config.Options.ControlAttributes); err != nil {
		return nil, request.Wrap("Create", "hosts", err)
	}
	body, err := masakari.Body(config, "host", "id", "uuid", "segment", "segment_id", "failover_segment_id", "failover_segment", "created_at", "updated_at", "deleted", "deleted_at")
	if err != nil {
		return nil, request.Wrap("Create", "hosts", err)
	}
	response, err := rest.DoJSON(ctx, s.client, "POST", s.client.ServiceURL(s.spec.Path), body, config.Headers, 201)
	if err != nil {
		return nil, request.Wrap("Create", "hosts", err)
	}
	value, err := rest.Decode(response, "host", s.spec.Metadata)
	return value, request.Wrap("Create", "hosts", err)
}

func (s *SegmentScope) Update(ctx context.Context, ref resource.Ref, opts UpdateOpts, options ...UpdateOption) (*Host, error) {
	if err := s.validate(ctx); err != nil {
		return nil, request.Wrap("Update", "hosts", err)
	}
	config, err := request.Apply(opts, options...)
	if err != nil {
		return nil, request.Wrap("Update", "hosts", err)
	}
	for _, value := range []*string{config.Options.Name, config.Options.Type} {
		if value != nil {
			if err := masakari.Required(*value); err != nil {
				return nil, request.Wrap("Update", "hosts", err)
			}
		}
	}
	body, err := masakari.Body(config, "host", "id", "uuid", "segment", "segment_id", "failover_segment_id", "failover_segment", "created_at", "updated_at", "deleted", "deleted_at")
	if err != nil {
		return nil, request.Wrap("Update", "hosts", err)
	}
	id, err := s.Resources.ResolveID(ctx, ref)
	if err != nil {
		return nil, request.Wrap("Update", "hosts", err)
	}
	response, err := rest.DoJSON(ctx, s.client, "PUT", s.client.ServiceURL(s.spec.Path, url.PathEscape(id)), body, config.Headers, 200)
	if err != nil {
		return nil, request.Wrap("Update", "hosts", err)
	}
	value, err := rest.Decode(response, "host", s.spec.Metadata)
	return value, request.Wrap("Update", "hosts", err)
}
