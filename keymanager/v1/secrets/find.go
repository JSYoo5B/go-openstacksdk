package secrets

import (
	"context"
	"encoding/json"
	"iter"
	"maps"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// FindIdentity uses a strict metadata/payload Fetch for a safe direct identity.
// Compatible HTTP 400/403/404 may fall back to an all-page metadata-only search.
// List results never trigger a subsequent secret or payload GET. Response IDs
// and references are compared passively and never become request targets.
func (a *API) FindIdentity(ctx context.Context, identity string, options ...resource.IdentityFindOption) (*FetchedSecret, error) {
	wrap := func(err error) error { return request.Wrap("FindIdentity", "secrets", err) }
	prepared, err := resource.PrepareIdentityFindOptions(options...)
	if err != nil {
		return nil, wrap(err)
	}
	// The pinned direct find_secret declaration accepts only ignore_missing.
	// Fallback is an explicit Go policy; generic query/mode/enrichment inputs
	// must not be silently accepted on either safe-ID or list-only branches.
	if len(prepared.Query) != 0 || prepared.Details != nil || prepared.AllProjects != nil || prepared.GetExtraSpecs != nil {
		return nil, wrap(resource.ErrUnsupported)
	}
	if a == nil {
		return nil, wrap(resource.ErrInvalidOption)
	}
	source := a.client
	if err := validateFetch(ctx, source); err != nil {
		return nil, wrap(err)
	}
	// Service configuration is a per-call snapshot. Its provider remains the
	// original shared pointer, so authentication/token changes stay live.
	client := *source
	client.MoreHeaders = maps.Clone(source.MoreHeaders)
	validate := func(ctx context.Context) error { return validateFetch(ctx, source) }
	spec := rest.CollectionSpec[FetchedSecret]{
		Client: &client, Path: "secrets", Kind: "secrets", PluralKey: "secrets",
		Metadata:  func(value *FetchedSecret) *resource.Metadata { return &value.Metadata },
		Validate:  validate,
		ListCodes: []int{http.StatusOK},
		Paging: rest.PagePolicy[FetchedSecret]{
			OffsetPagination: true, HTTPLink: true, StopOnEmptyPage: true,
		},
	}
	binding := resource.NewCollection(resource.Adapter[FetchedSecret]{
		Kind: "secrets", IdentityFind: true, ValidateID: validateFetchID,
		IdentityResponseID: secretFindResponseID,
		Name:               func(value *FetchedSecret) string { return value.Name },
		NameQuery:          func(value string) string { return value },
		Get: func(ctx context.Context, id string) (*FetchedSecret, error) {
			if err := validate(ctx); err != nil {
				return nil, err
			}
			value, err := New(&client).Fetch(ctx, resource.ID(id))
			if err == nil {
				err = validate(ctx)
			}
			return value, err
		},
		Iterate: func(ctx context.Context, query url.Values) iter.Seq2[*FetchedSecret, error] {
			return rest.List(ctx, spec, query)
		},
	})
	// User callbacks have already run once. The common finder receives only an
	// owned prepared snapshot, preserving shared input/fallback/context rules.
	return binding.FindIdentity(ctx, identity, resource.WithIdentityFindOptions(prepared))
}

func secretFindResponseID(value *FetchedSecret) (string, error) {
	if value == nil {
		return "", resource.ErrInvalidOption
	}
	if raw, exists := value.Body["id"]; exists {
		var id string
		// Python's untyped literal id wins even when null or nonstring. Such
		// values cannot equal the string input; do not coerce or use an alias.
		if err := json.Unmarshal(raw, &id); err != nil {
			return "", nil
		}
		return id, nil
	}
	if value.SecretID != "" {
		// Direct Fetch retains the request seed out of band. List models never
		// receive a seed and use the full original reference as their alternate.
		return value.SecretID, nil
	}
	return value.SecretRef, nil
}
