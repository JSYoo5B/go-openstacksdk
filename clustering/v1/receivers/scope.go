package receivers

import (
	"context"
	"fmt"

	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// UpdateScope fixes a literal collection path for receiver updates and tracked
// lifecycle calls. It retains the API's client and shared authentication;
// receiver action strings and channel values remain ordinary response data.
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
			err = fmt.Errorf("%w: receiver update scope requires a service client and provider", resource.ErrInvalidOption)
		}
	}
	if err != nil {
		return nil, request.Wrap("AtBasePath", "clustering.receivers", err)
	}
	return &UpdateScope{api: a, path: canonical}, nil
}

func (scope *UpdateScope) validate() error {
	if scope == nil || scope.api == nil || scope.path == "" {
		return fmt.Errorf("%w: receiver update scope is required", resource.ErrInvalidOption)
	}
	return nil
}

// Update uses the scoped path for name resolution and the synchronous PATCH.
func (scope *UpdateScope) Update(ctx context.Context, ref resource.Ref, opts UpdateOpts, options ...UpdateOption) (*Receiver, error) {
	if err := scope.validate(); err != nil {
		return nil, request.Wrap("Update", "clustering.receivers", err)
	}
	return scope.api.updateAt(ctx, ref, scope.path, opts, options...)
}

// Load obtains one strict reference and retains its scoped path for Commit and
// Refresh. An explicit ID remains the fixed route regardless of response IDs.
func (scope *UpdateScope) Load(ctx context.Context, ref resource.Ref) (*TrackedReceiver, error) {
	if err := scope.validate(); err != nil {
		return nil, request.Wrap("Load", "clustering.receivers", err)
	}
	return scope.api.loadAt(ctx, ref, scope.path)
}

// Track takes ownership of a cached model without HTTP and fixes the scoped
// path for later Commit and Refresh. Cached Body controls its canonical ID.
func (scope *UpdateScope) Track(value *Receiver) (*TrackedReceiver, error) {
	if err := scope.validate(); err != nil {
		return nil, request.Wrap("Track", "clustering.receivers", err)
	}
	return scope.api.trackReceiver(value, "", scope.path)
}
