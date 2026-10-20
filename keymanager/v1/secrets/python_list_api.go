package secrets

import (
	"bytes"
	"context"
	"errors"
	"iter"
	"net/http"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func recordListCodes() []int {
	values := make([]int, 300)
	for i := range values {
		values[i] = 100 + i
	}
	return values
}
func recordListRejections() rest.RejectionPolicy {
	values := make([]int, 200)
	for i := range values {
		values[i] = 400 + i
	}
	return rest.RejectionPolicy{Codes: values, PreserveCleanRetry: true}
}
func recordListMarker(value *resource.RawResource) (string, error) {
	raw, present := value.Body["id"]
	if !present {
		raw = value.Body["secret_ref"]
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return "", recordListInvalid("secrets marker is missing")
	}
	marker, err := cloudfilter.PythonString(raw)
	if err != nil || marker == "" {
		return "", recordListInvalid("secrets marker must be nonempty")
	}
	return marker, nil
}

// ListRecords is the owned Source secrets iterator. It performs no I/O until
// iteration, returns complete nullable Resource views and literal HTTP proof,
// and counts raw rows before Body filtering. No member or payload GET is added.
// Returned RequestID is empty: references are passive values, not fetch routes.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*FoundSecretRecord, error] {
	ownedOptions := slices.Clone(options)
	return func(yield func(*FoundSecretRecord, error) bool) {
		wrap := func(err error) error { return request.Wrap("ListRecords", "secrets", cloudread.ContextError(ctx, err)) }
		fail := func(err error) { yield(nil, wrap(err)) }
		if err := cloudread.Context(ctx); err != nil {
			fail(err)
			return
		}
		if a == nil {
			fail(recordListInvalid("secrets API is required"))
			return
		}
		selected := a.client
		source, err := cloudread.Capture(ctx, selected, "key-manager")
		if err != nil {
			fail(err)
			return
		}
		guard := func(ctx context.Context) error {
			var changed error
			if a.client != selected {
				changed = recordListInvalid("selected secrets client changed")
			}
			return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
		}
		policy := RecordListOpts{}
		for _, apply := range ownedOptions {
			if err := guard(ctx); err != nil {
				fail(err)
				return
			}
			if apply == nil {
				fail(recordListInvalid("nil list option"))
				return
			}
			next := ownRecordListOpts(policy)
			if err := errors.Join(apply(&next), guard(ctx)); err != nil {
				fail(err)
				return
			}
			policy = ownRecordListOpts(next)
		}
		query, filters, control, err := prepareRecordListQuery(policy)
		if err == nil {
			err = errors.Join(source.WithPolicy(ctx, policy.Microversion, policy.Headers), guard(ctx))
		}
		if err != nil {
			fail(err)
			return
		}
		headers := map[string]string{"Accept": "application/json"}
		for key, value := range policy.Headers {
			headers[http.CanonicalHeaderKey(key)] = value
		}
		for key := range source.Client.MoreHeaders {
			if strings.EqualFold(key, "Accept") {
				delete(source.Client.MoreHeaders, key)
			}
		}
		var page *rest.Response
		var view *resource.RawResource
		spec := rest.CollectionSpec[resource.RawResource]{
			Client: &source.Client, Path: "secrets", Kind: "secrets", PluralKey: "secrets",
			Metadata: func(value *resource.RawResource) *resource.Metadata { return &value.Metadata },
			Validate: guard, SourceGuard: guard, ListCodes: recordListCodes(),
			ReadPage: func(_ context.Context, target string, _ ...int) (*rest.Response, error) {
				return rest.DoJSONGuardedRejectionsHeaders(ctx, &source.Client, guard, http.MethodGet, target, nil, headers, recordListRejections(), recordListCodes()...)
			},
			ValidateResponse: func(response *rest.Response) error { page = response; return guard(ctx) },
			ValidateItem: func(wire *resource.RawResource) error {
				for _, key := range []string{"connection", "microversion", "_synchronized", "cls"} {
					if _, present := wire.Body[key]; present {
						return errors.Join(recordListInvalid("secrets row %q conflicts with the Source Resource constructor", key), guard(ctx))
					}
				}
				var err error
				view, err = projectFindRecord(wire, findRecordFields(wire.Body), nil)
				return errors.Join(err, guard(ctx))
			},
			Paging: rest.PagePolicy[resource.RawResource]{MarkerOrOffsetPagination: true, HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, VersionedPath: "/v1/secrets", MarkerFallback: true, Marker: recordListMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true},
		}
		for wire, err := range rest.ListWithControl(ctx, spec, query, control) {
			if err != nil {
				fail(err)
				return
			}
			matched := true
			for key, filter := range filters {
				equal, cause := recordListLocalMatch(view.Body[key], filter)
				if cause = errors.Join(cause, guard(ctx)); cause != nil {
					fail(page.Fail(cause))
					return
				}
				if !equal {
					matched = false
					break
				}
			}
			if !matched {
				continue
			}
			value := &FoundSecretRecord{Resource: view.Clone(), Wire: wire.Clone(), Envelope: bytes.Clone(page.Body), Header: page.Header.Clone(), StatusCode: page.StatusCode}

			if !yield(value, nil) {
				return
			}
			if err := guard(ctx); err != nil {
				fail(page.Fail(err))
				return
			}
		}
	}
}

// AllRecords collects ListRecords; a terminal error discards partial rows.
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*FoundSecretRecord, error) {
	var values []*FoundSecretRecord
	for value, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, request.Wrap("AllRecords", "secrets", err)
		}
		values = append(values, value)
	}
	return values, nil
}
