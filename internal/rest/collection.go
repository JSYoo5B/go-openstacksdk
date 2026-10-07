package rest

import (
	"context"
	"fmt"
	"iter"
	"net/http"
	"net/url"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

// CollectionSpec describes a service's actual read/delete capabilities. Service
// bindings retain ownership of schemas, paths, identifiers and version gates.
type CollectionSpec[T any] struct {
	Client                           *gophercloud.ServiceClient
	Path, Kind, SingleKey, PluralKey string
	ID, Name, Status                 func(*T) string
	Metadata                         func(*T) *resource.Metadata
	Validate                         func(context.Context) error
	// SourceGuard checks library-owned source identity on every HTTP attempt,
	// including retries/authentication and accepted response read/Close.
	SourceGuard   func(context.Context) error
	ValidateQuery func(context.Context, url.Values) error
	// ValidateInitialQuery applies only to caller input, before the first HTTP
	// request. ValidateQuery also checks server-provided continuation queries.
	ValidateInitialQuery func(context.Context, url.Values) error
	ValidateID           func(string) error
	// ValidateResponse checks the whole accepted GET/list body before JSON
	// decoding. Unset hooks preserve the existing decoder policy.
	ValidateResponse func(*Response) error
	// ValidateItem checks service-specific response invariants after decoding.
	// Failure retains the original singular response or whole list page.
	ValidateItem                     func(*T) error
	Get, Delete                      bool
	GetCodes, ListCodes, DeleteCodes []int
	Failed                           func(string) bool
	LocalStatus                      bool
	NameQuery                        func(string) string
	NameQueryKey                     string
	Paging                           PagePolicy[T]
}

func (s CollectionSpec[T]) validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.Client == nil || s.Client.ProviderClient == nil || strings.TrimSpace(s.Path) == "" {
		return fmt.Errorf("%w: collection client and path are required", resource.ErrInvalidOption)
	}
	if s.Validate != nil {
		return s.Validate(ctx)
	}
	return nil
}

func successCodes(codes []int, fallback int) []int {
	if len(codes) == 0 {
		return []int{fallback}
	}
	return append([]int(nil), codes...)
}

// Collection supplies only enabled operations to the shared lookup/wait layer.
// An omitted operation remains unsupported; construction performs no HTTP.
func Collection[T any](spec CollectionSpec[T]) *resource.Collection[T] {
	adapter := resource.Adapter[T]{Kind: spec.Kind, ID: spec.ID, Name: spec.Name,
		Status: spec.Status, Failed: spec.Failed, LocalStatus: spec.LocalStatus,
		NameQuery: spec.NameQuery, NameQueryKey: spec.NameQueryKey, ValidateID: spec.ValidateID}
	adapter.Iterate = func(ctx context.Context, query url.Values) iter.Seq2[*T, error] {
		return List(ctx, spec, query)
	}
	adapter.IterateControlled = func(ctx context.Context, query url.Values, control resource.ListControl) iter.Seq2[*T, error] {
		return ListWithControl(ctx, spec, query, ListControl{MaxItems: control.MaxItems,
			SinglePage: control.SinglePage, LimitHint: spec.Paging.MaxItemsLimitHint})
	}
	if spec.Get {
		adapter.Get = func(ctx context.Context, id string) (*T, error) {
			if err := spec.validate(ctx); err != nil {
				return nil, err
			}
			response, err := DoJSONGuarded(ctx, spec.Client, spec.SourceGuard, http.MethodGet, spec.Client.ServiceURL(spec.Path, url.PathEscape(id)), nil, nil, successCodes(spec.GetCodes, http.StatusOK)...)
			if err != nil {
				return nil, err
			}
			if spec.ValidateResponse != nil {
				if err := spec.ValidateResponse(response); err != nil {
					return nil, response.Fail(err)
				}
			}
			value, err := Decode(response, spec.SingleKey, spec.Metadata)
			if err != nil {
				return nil, err
			}
			if spec.ValidateItem != nil {
				if err := spec.ValidateItem(value); err != nil {
					return nil, response.Fail(err)
				}
			}
			return value, nil
		}
	}
	if spec.Delete {
		adapter.Delete = func(ctx context.Context, id string) error {
			if err := spec.validate(ctx); err != nil {
				return err
			}
			_, err := DoJSONGuarded(ctx, spec.Client, spec.SourceGuard, http.MethodDelete, spec.Client.ServiceURL(spec.Path, url.PathEscape(id)), nil, nil, successCodes(spec.DeleteCodes, http.StatusNoContent)...)
			return err
		}
	}
	return resource.NewCollection(adapter)
}
