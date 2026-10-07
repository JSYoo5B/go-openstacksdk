// Package secretstores reads Barbican's available, global-default and preferred
// secret-store backends. It never follows a response's secret_store_ref URL.
package secretstores

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cloudread"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

const kind = "keymanager.secret_stores"

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }

func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

func validate(ctx context.Context, client *gophercloud.ServiceClient) error {
	if ctx == nil {
		return fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if client == nil || client.ProviderClient == nil {
		return fmt.Errorf("%w: a key-manager service client is required", resource.ErrInvalidOption)
	}
	if client.Type != "key-manager" {
		return fmt.Errorf("%w: expected key-manager client, got %q", resource.ErrUnsupported, client.Type)
	}
	return nil
}

// GetGlobalDefault reads the deployment's default backend, not a project's
// preferred backend. A 404 remains an error, including feature-disabled 404s.
func (a *API) GetGlobalDefault(ctx context.Context) (*SecretStore, error) {
	return a.get(ctx, "global-default", "GetGlobalDefault")
}

// GetPreferred reads the backend selected for the authenticated project. It
// performs no project lookup and does not fall back to the global default.
func (a *API) GetPreferred(ctx context.Context) (*SecretStore, error) {
	return a.get(ctx, "preferred", "GetPreferred")
}

func (a *API) get(ctx context.Context, selector, operation string) (*SecretStore, error) {
	client := a.RawClient()
	if err := validate(ctx, client); err != nil {
		return nil, request.Wrap(operation, kind, err)
	}
	response, err := rest.DoJSON(ctx, client, http.MethodGet, client.ServiceURL("secret-stores", selector), nil, nil, http.StatusOK)
	if err != nil {
		var transport *url.Error
		var accepted *resource.ResponseError
		if !errors.As(err, &transport) && !errors.As(err, &accepted) && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			err = &resource.NotFoundError{Resource: kind, Reference: selector, Cause: err}
		}
		return nil, request.Wrap(operation, kind, err)
	}
	value, err := rest.Decode(response, "", func(value *SecretStore) *resource.Metadata { return &value.Metadata })
	return value, request.Wrap(operation, kind, err)
}

func spec(client *gophercloud.ServiceClient) rest.CollectionSpec[SecretStore] {
	return rest.CollectionSpec[SecretStore]{
		Client: client, Path: "secret-stores", Kind: kind, PluralKey: "secret_stores",
		ListCodes: []int{http.StatusOK},
		Metadata:  func(value *SecretStore) *resource.Metadata { return &value.Metadata },
		Validate:  func(ctx context.Context) error { return validate(ctx, client) },
		Paging: rest.PagePolicy[SecretStore]{
			HTTPLink: true, MarkerFallback: true, MarkerOnShortPage: true,
			StopOnEmptyPage: true,
			Marker: func(value *SecretStore) (string, error) {
				// Resource.id bypasses SecretStore.secret_store_id's HREF
				// formatter. The wire marker is the literal id, or the FULL
				// original ref URL, not the derived convenience ID.
				marker := value.SecretStoreRef
				if raw, exists := value.Body["id"]; exists {
					marker = ""
					if err := json.Unmarshal(raw, &marker); err != nil {
						return "", err
					}
				}
				return marker, nil
			},
		},
	}
}

// List is lazy and reusable. It snapshots options independently for every
// iteration. Caps count decoded rows, and an explicit limit enables marker
// fallback; otherwise only advertised continuation links are followed.
func (a *API) List(ctx context.Context, options ...ListOption) iter.Seq2[*SecretStore, error] {
	owned := append([]ListOption(nil), options...)
	return func(yield func(*SecretStore, error) bool) {
		wrap := func(err error) error {
			if err != nil {
				err = cloudread.ContextError(ctx, err)
			}
			return request.Wrap("List", kind, err)
		}
		client := a.RawClient()
		if err := validate(ctx, client); err != nil {
			yield(nil, wrap(err))
			return
		}
		source, err := cloudread.Capture(ctx, client, "key-manager")
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		guard := func(ctx context.Context) error {
			if err := validate(ctx, client); err != nil {
				return cloudread.ContextError(ctx, err)
			}
			return source.Guard(ctx)
		}
		query, control, filters, err := prepareList(owned)
		if err == nil {
			// Capture the selected source before any user option callback.
			// The shared guard also runs around every HTTP attempt and read.
			err = guard(ctx)
		}
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		selected := spec(&source.Client)
		selected.SourceGuard = guard
		for value, err := range listWithFilters(ctx, selected, query, control, filters) {
			if !yield(value, wrap(err)) {
				return
			}
		}
	}
}

// All discards partial rows on a terminal page error and preserves its cause.
func (a *API) All(ctx context.Context, options ...ListOption) ([]*SecretStore, error) {
	var values []*SecretStore
	for value, err := range a.List(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
