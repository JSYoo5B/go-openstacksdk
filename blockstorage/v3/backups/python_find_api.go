package backups

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func findCodes() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	return codes
}
func findRejections() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 400 + i
	}
	return codes
}

// FindRecord mirrors find_backup: GET first, then default /backups/detail
// with a name query after a clean400/403/404. Details(false) selects /backups.
// Complete list observation resolves exact ID or name, including duplicates.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*BackupResourceRecord, error) {
	const op = "FindRecord"
	wrap := func(err error) error { return request.Wrap(op, "backups", cloudread.ContextError(ctx, err)) }
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if err := findSegment(identity); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(findInvalid("backup API is required"))
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "volume")
	if err != nil {
		return nil, wrap(err)
	}
	guard := func(ctx context.Context) error {
		var changed error
		if a.client != selected {
			changed = findInvalid("selected backup client changed")
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	if err = guard(ctx); err != nil {
		return nil, wrap(err)
	}
	policy := RecordFindOpts{}
	for _, apply := range slices.Clone(options) {
		if err = guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(findInvalid("nil find option"))
		}
		next := copyFindOpts(policy)
		err = apply(&next)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyFindOpts(next)
	}
	getQuery, listQuery, filters, seed, control, err := prepareFindQuery(identity, policy)
	if err == nil {
		err = errors.Join(source.WithPolicy(ctx, nil, policy.Headers), guard(ctx))
	}
	if err != nil {
		return nil, wrap(err)
	}
	var discovery []FindDiscoveryObservation
	discoveryFailure := func(cause error) error {
		if len(discovery) == 0 {
			return cause
		}
		actual := discovery[len(discovery)-1]
		return (&rest.Response{Body: bytes.Clone(actual.Envelope), Header: actual.Header.Clone(), StatusCode: actual.StatusCode}).Fail(cause)
	}
	// Resource._get_microversion short-circuits an explicit session default.
	// Without one, Source chooses min(server maximum, Backup max3.64).
	if source.Client.Microversion == "" {
		discovered, readErr := microversions.Read(ctx, source, microversions.CinderProfile)
		for _, response := range discovered.Responses {
			discovery = append(discovery, FindDiscoveryObservation{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode})
		}
		if err = errors.Join(readErr, guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		version, selectErr := microversions.SelectVersion(discovered.Maximum, discovered.Minimum, "3.64")
		if selectErr != nil {
			return nil, wrap(discoveryFailure(selectErr))
		}
		if err = errors.Join(source.WithPolicy(ctx, &version, nil), guard(ctx)); err != nil {
			return nil, wrap(discoveryFailure(err))
		}
	} else if _, err = microversions.ParseVersion(source.Client.Microversion); err != nil {
		return nil, wrap(findInvalid("invalid selected microversion: %v", err))
	}
	// Selected protocol headers are fixed by each request's private policy.
	// Remove equivalent aliases from the private copy, retaining the original
	// service facts and unrelated caller header precedence.
	for key := range source.Client.MoreHeaders {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Volume-API-Version") {
			delete(source.Client.MoreHeaders, key)
		}
	}
	read := func(target string) (*rest.Response, error) {
		headers := rest.RequestHeaderPolicy{Values: make(map[string]string)}
		if version := source.Client.Microversion; version != "" {
			headers.Values["OpenStack-API-Version"] = "volume " + version
			headers.Values["X-OpenStack-Volume-API-Version"] = version
		} else {
			headers.Absent = []string{"OpenStack-API-Version", "X-OpenStack-Volume-API-Version"}
		}
		return rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &source.Client, guard, http.MethodGet, target, nil, headers, rest.RejectionPolicy{Codes: findRejections(), PreserveCleanRetry: true}, findCodes()...)
	}
	get := func(_ context.Context, id string) (*BackupResourceRecord, error) {
		target := source.Client.ServiceURL("backups", url.PathEscape(id))
		if len(getQuery) > 0 {
			target += "?" + getQuery.Encode()
		}
		response, err := read(target)
		if err != nil {
			return nil, err
		}
		owned := findFields(seed)
		owned["id"], _ = json.Marshal(id)
		value, err := findResponse(response, owned)
		if err = errors.Join(err, guard(ctx)); err != nil {
			err = response.Fail(err)
		}
		if value != nil {
			value.Discovery = findDiscovery(discovery)
		}
		return value, err
	}
	list := func(context.Context, url.Values) iter.Seq2[*BackupResourceRecord, error] {
		return func(yield func(*BackupResourceRecord, error) bool) {
			if source.Client.MoreHeaders == nil {
				source.Client.MoreHeaders = make(map[string]string)
			}
			if _, set := source.Client.MoreHeaders["Accept"]; !set {
				source.Client.MoreHeaders["Accept"] = "application/json"
			}
			path := "backups/detail"
			if policy.Details != nil && !*policy.Details {
				path = "backups"
			}
			var origin *rest.Response
			spec := rest.CollectionSpec[BackupResourceRecord]{Client: &source.Client, Path: path, Kind: "backups", PluralKey: "backups", Metadata: findMetadata, Validate: guard, SourceGuard: guard, ListCodes: findCodes(), ReadPage: func(_ context.Context, target string, _ ...int) (*rest.Response, error) { return read(target) }, ValidateResponse: func(response *rest.Response) error { origin = response; return guard(ctx) }, ValidateItem: func(value *BackupResourceRecord) error {
				value.Envelope = bytes.Clone(origin.Body)
				value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
				value.Discovery = findDiscovery(discovery)
				return errors.Join(findProject(value, nil), guard(ctx))
			}, Paging: rest.PagePolicy[BackupResourceRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerFallback: true, Marker: findMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
			// Explicit initial offset opts into the shared offset pager. It
			// disallows mixing marker and offset and omits marker fallback.
			if listQuery.Has("offset") {
				spec.Paging.OffsetPagination = true
				spec.Paging.MarkerFallback = false
			}
			for value, err := range rest.ListWithControl(ctx, spec, listQuery, control) {
				if err != nil {
					yield(nil, err)
					return
				}
				matched := true
				for key, filter := range filters {
					equal, cause := findLocalMatch(value.Resource.Body[key], filter)
					if cause = errors.Join(cause, guard(ctx)); cause != nil {
						yield(nil, origin.Fail(cause))
						return
					}
					if !equal {
						matched = false
						break
					}
				}
				if matched && !yield(value, nil) {
					return
				}
			}
		}
	}
	collection := resource.NewCollection(resource.Adapter[BackupResourceRecord]{Kind: "backups", IdentityFind: true, ValidateID: findSegment, IdentityDirectGet: findSegment, IdentityResponseID: func(value *BackupResourceRecord) (string, error) {
		if value == nil || value.Resource == nil {
			return "", findInvalid("find response resource is missing")
		}
		// Source Body(id) is untyped. A direct successful GET returns it even
		// if numeric/null. For list matching, only an actual string ID equals
		// the caller's string identity; names remain a separate comparison.
		return findText(value, "id"), nil
	}, Name: func(value *BackupResourceRecord) string { return findText(value, "name") }, Get: get, Iterate: list})
	value, err := collection.FindIdentity(ctx, identity, resource.WithIdentityFindIgnoreMissing(policy.IgnoreMissing == nil || *policy.IgnoreMissing))
	return value, wrap(errors.Join(err, guard(ctx)))
}
