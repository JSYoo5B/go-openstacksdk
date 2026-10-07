package nodes

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/internal/senlin"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// UpdateScope fixes a literal collection path for node updates and tracked
// lifecycle calls. It retains the API's client and shared authentication;
// asynchronous action Locations remain relative to that client's service root.
type UpdateScope struct {
	api  *API
	path string
}

// AtBasePath creates an update scope without HTTP. The relative literal path
// is validated and escaped once; context, service and version validation occurs
// when a request is made. This scope exposes Update, Load and Track only.
func (a *API) AtBasePath(path string) (*UpdateScope, error) {
	canonical, err := senlin.CollectionPath(path)
	if err == nil {
		client := a.RawClient()
		if client == nil || client.ProviderClient == nil {
			err = fmt.Errorf("%w: node update scope requires a service client and provider", resource.ErrInvalidOption)
		}
	}
	if err != nil {
		return nil, request.Wrap("AtBasePath", "clustering.nodes", err)
	}
	return &UpdateScope{api: a, path: canonical}, nil
}

func (scope *UpdateScope) validate() error {
	if scope == nil || scope.api == nil || scope.path == "" {
		return fmt.Errorf("%w: node update scope is required", resource.ErrInvalidOption)
	}
	return nil
}

// Update uses the scoped path for name resolution and the final PATCH.
func (scope *UpdateScope) Update(ctx context.Context, ref resource.Ref, opts UpdateOpts, options ...UpdateOption) (*Node, error) {
	if err := scope.validate(); err != nil {
		return nil, request.Wrap("Update", "clustering.nodes", err)
	}
	return scope.api.updateAt(ctx, ref, scope.path, opts, options...)
}

// Load obtains one strict reference and retains its scoped path for Commit and
// Refresh. An explicit ID remains the fixed route regardless of response IDs.
func (scope *UpdateScope) Load(ctx context.Context, ref resource.Ref) (*TrackedNode, error) {
	if err := scope.validate(); err != nil {
		return nil, request.Wrap("Load", "clustering.nodes", err)
	}
	return scope.api.loadAt(ctx, ref, scope.path)
}

// Track takes ownership of a cached model without HTTP and fixes the scoped
// path for later Commit and Refresh. Cached Body controls its canonical ID.
func (scope *UpdateScope) Track(value *Node) (*TrackedNode, error) {
	if err := scope.validate(); err != nil {
		return nil, request.Wrap("Track", "clustering.nodes", err)
	}
	return scope.api.trackNode(value, "", scope.path)
}
