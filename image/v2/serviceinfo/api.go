// Package serviceinfo reads Glance store and import-method discovery data.
// Results describe the current deployment and never gate other image APIs.
package serviceinfo

import (
	"context"
	"errors"
	"iter"
	"net/http"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const kind = "image.service_info"

type API struct{ client *gophercloud.ServiceClient }

func New(client *gophercloud.ServiceClient) *API { return &API{client: client} }

func (a *API) RawClient() *gophercloud.ServiceClient {
	if a == nil {
		return nil
	}
	return a.client
}

// GetImportInfo reads the flat import-method discovery document once. Missing
// and null import-methods remain nil; unknown root and method fields stay raw.
func (a *API) GetImportInfo(ctx context.Context, options ...GetImportInfoOption) (*ImportInfo, error) {
	prepared, err := a.capture(ctx)
	if err != nil {
		return nil, wrapInfoError(ctx, "GetImportInfo", err)
	}
	headers, err := prepareGetImportInfo(append([]GetImportInfoOption(nil), options...))
	if err == nil {
		err = prepared.check(ctx)
	}
	if err != nil {
		return nil, wrapInfoError(ctx, "GetImportInfo", err)
	}
	prepared.addHeaders(headers)
	response, err := rest.DoJSON(ctx, prepared.client, http.MethodGet, prepared.client.ServiceURL("info", "import"), nil, nil, http.StatusOK)
	if err != nil {
		return nil, wrapInfoError(ctx, "GetImportInfo", err)
	}
	if err := validateResponse(response); err != nil {
		return nil, wrapInfoError(ctx, "GetImportInfo", response.Fail(err))
	}
	value, err := rest.Decode(response, "", func(value *ImportInfo) *resource.Metadata { return &value.Metadata })
	return value, wrapInfoError(ctx, "GetImportInfo", err)
}

// ListStores is lazy and reusable. Each iteration owns its options and source
// snapshot. Details selects the detail route locally; it is never a query key.
// Only advertised continuations trigger another page; Glance's discovery
// handlers return their complete store list and ignore pagination queries.
func (a *API) ListStores(ctx context.Context, options ...ListStoresOption) iter.Seq2[*Store, error] {
	owned := append([]ListStoresOption(nil), options...)
	return func(yield func(*Store, error) bool) {
		prepared, err := a.capture(ctx)
		if err != nil {
			yield(nil, wrapInfoError(ctx, "ListStores", err))
			return
		}
		policy, query, headers, control, err := prepareListStores(owned)
		if err == nil {
			err = prepared.check(ctx)
		}
		if err != nil {
			yield(nil, wrapInfoError(ctx, "ListStores", err))
			return
		}
		prepared.addHeaders(headers)
		path := "info/stores"
		if policy.Details {
			path += "/detail"
		}
		spec := rest.CollectionSpec[Store]{
			Client: prepared.client, Path: path, Kind: kind, PluralKey: "stores",
			ListCodes: []int{http.StatusOK},
			Metadata:  func(value *Store) *resource.Metadata { return &value.Metadata },
			Validate:  prepared.check, ValidateResponse: validateResponse,
			Paging: rest.PagePolicy[Store]{
				HTTPLink: true, StopOnEmptyPage: true,
			},
		}
		for value, err := range rest.ListWithControl(ctx, spec, query, control) {
			if !yield(value, wrapInfoError(ctx, "ListStores", err)) {
				return
			}
		}
	}
}

// AllStores returns a nonnil empty slice on successful empty discovery and
// discards partial rows if a later page or row fails.
func (a *API) AllStores(ctx context.Context, options ...ListStoresOption) ([]*Store, error) {
	values := make([]*Store, 0)
	for value, err := range a.ListStores(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func validateResponse(response *rest.Response) error {
	if !utf8.Valid(response.Body) {
		return infoInvalid("discovery response must be valid UTF-8")
	}
	return nil
}

func wrapInfoError(ctx context.Context, operation string, err error) error {
	if err != nil && ctx != nil && ctx.Err() != nil {
		for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
			if cause != nil && !errors.Is(err, cause) {
				err = errors.Join(err, cause)
			}
		}
	}
	return request.Wrap(operation, kind, err)
}
