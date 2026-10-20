package clusters

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/internal/senlin"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// UpdateScope fixes a literal collection path for Update and tracked lifecycle
// requests. It does not change the API's default routes, service endpoint or
// action collection, and does not expose unrelated CRUD or command operations.
type UpdateScope struct {
	api  *API
	path string
}

// AtBasePath selects an unescaped relative collection path below the service's
// resource base. URLs, templates, empty/dot segments, whitespace and preescaped
// paths are rejected. Construction performs no HTTP or version negotiation.
func (a *API) AtBasePath(path string) (*UpdateScope, error) {
	client := a.RawClient()
	if client == nil || client.ProviderClient == nil {
		return nil, request.Wrap("AtBasePath", "clustering.clusters", fmt.Errorf("%w: cluster service client is required", resource.ErrInvalidOption))
	}
	path, err := senlin.CollectionPath(path)
	if err != nil {
		return nil, request.Wrap("AtBasePath", "clustering.clusters", err)
	}
	return &UpdateScope{api: a, path: path}, nil
}

func (scope *UpdateScope) validate() error {
	if scope == nil || scope.api == nil || scope.path == "" {
		return fmt.Errorf("%w: cluster update scope is required", resource.ErrInvalidOption)
	}
	return nil
}

// Update uses the fixed collection for lookup and PATCH. Concrete update
// options, asynchronous response evidence and version gates remain unchanged.
func (scope *UpdateScope) Update(ctx context.Context, ref resource.Ref, opts UpdateOpts, options ...UpdateOption) (*Cluster, error) {
	if err := scope.validate(); err != nil {
		return nil, request.Wrap("Update", "clustering.clusters", err)
	}
	return scope.api.updateAt(ctx, ref, scope.path, opts, options...)
}

// Load resolves one explicit reference within this collection and keeps both
// the chosen identity and collection fixed for later Commit and Refresh calls.
func (scope *UpdateScope) Load(ctx context.Context, ref resource.Ref) (*TrackedCluster, error) {
	if err := scope.validate(); err != nil {
		return nil, request.Wrap("Load", "clustering.clusters", err)
	}
	return scope.api.loadAt(ctx, ref, scope.path)
}

// Track wraps a cached model without HTTP and attaches the fixed collection to
// its lifecycle. Raw Body identity and snapshot policies match API.Track.
func (scope *UpdateScope) Track(value *Cluster) (*TrackedCluster, error) {
	if err := scope.validate(); err != nil {
		return nil, request.Wrap("Track", "clustering.clusters", err)
	}
	return scope.api.trackCluster(value, "", scope.path)
}
