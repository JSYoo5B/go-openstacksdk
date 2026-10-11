package volumes

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

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudlocation"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func findCodes() []int {
	// Source raise_from_response accepts every valid HTTP status below400.
	codes := make([]int, 300)
	for i := range codes {
		codes[i] = 100 + i
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

// FindRecord mirrors find_volume: GET first, then default /volumes/detail
// with a name query after a clean400/403/404. Details(false) selects /volumes.
// AllProjects(true) adds all_tenants=True only to that fallback list.
// Complete list observation resolves exact ID or name, including duplicates.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*VolumeResourceRecord, error) {
	const op = "FindRecord"
	wrap := func(err error) error { return request.Wrap(op, "volumes", cloudread.ContextError(ctx, err)) }
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if err := findSegment(identity); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(findInvalid("volume API is required"))
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "volume")
	if err != nil {
		return nil, wrap(err)
	}
	guard := func(ctx context.Context) error {
		var changed error
		if a.client != selected {
			changed = findInvalid("selected volume client changed")
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
	var location resource.CloudLocation
	if policy.Location != nil {
		location = policy.Location.Clone()
	} else {
		location.Project.ID, err = cloudlocation.ProjectID(source.Client.ProviderClient)
		if err = errors.Join(err, guard(ctx)); err != nil {
			return nil, wrap(err)
		}
	}
	// Resource.find constructs an existing Volume before fetch/discovery.
	// Seed descriptors must fail here even if an eventual HTTP response would
	// overwrite them. Reuse the exact owned projection rather than inventing
	// an independent constructor validation table.
	constructorSeed := findFields(seed)
	constructorSeed["id"], _ = json.Marshal(identity)
	constructor := &VolumeResourceRecord{}
	if err = errors.Join(findProject(constructor, constructorSeed, location), guard(ctx)); err != nil {
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
	// Without one, Source chooses min(server maximum, Volume max3.71).
	if source.Client.Microversion == "" {
		discovered, readErr := microversions.Read(ctx, source, microversions.CinderProfile)
		for _, response := range discovered.Responses {
			discovery = append(discovery, FindDiscoveryObservation{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode})
		}
		if err = errors.Join(readErr, guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		version, selectErr := microversions.SelectVersion(discovered.Maximum, discovered.Minimum, "3.71")
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
	get := func(_ context.Context, id string) (*VolumeResourceRecord, error) {
		target := source.Client.ServiceURL("volumes", url.PathEscape(id))
		if len(getQuery) > 0 {
			target += "?" + getQuery.Encode()
		}
		response, err := read(target)
		if err != nil {
			return nil, err
		}
		owned := findFields(seed)
		owned["id"], _ = json.Marshal(id)
		value, err := findResponse(response, owned, location)
		if err = errors.Join(err, guard(ctx)); err != nil {
			err = response.Fail(err)
		}
		if value != nil {
			value.Discovery = findDiscovery(discovery)
		}
		return value, err
	}
	list := func(context.Context, url.Values) iter.Seq2[*VolumeResourceRecord, error] {
		return func(yield func(*VolumeResourceRecord, error) bool) {
			if source.Client.MoreHeaders == nil {
				source.Client.MoreHeaders = make(map[string]string)
			}
			if _, set := source.Client.MoreHeaders["Accept"]; !set {
				source.Client.MoreHeaders["Accept"] = "application/json"
			}
			path := "volumes/detail"
			if policy.Details != nil && !*policy.Details {
				path = "volumes"
			}
			var origin *rest.Response
			spec := rest.CollectionSpec[VolumeResourceRecord]{Client: &source.Client, Path: path, Kind: "volumes", PluralKey: "volumes", Metadata: findMetadata, Validate: guard, SourceGuard: guard, ListCodes: findCodes(), ReadPage: func(_ context.Context, target string, _ ...int) (*rest.Response, error) { return read(target) }, ValidateResponse: func(response *rest.Response) error { origin = response; return guard(ctx) }, ValidateItem: func(value *VolumeResourceRecord) error {
				value.Envelope = bytes.Clone(origin.Body)
				value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
				value.Discovery = findDiscovery(discovery)
				if err := findListConstructor(value); err != nil {
					return errors.Join(err, guard(ctx))
				}
				return errors.Join(findProject(value, nil, location), guard(ctx))
			}, Paging: rest.PagePolicy[VolumeResourceRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerFallback: true, Marker: findMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
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
	collection := resource.NewCollection(resource.Adapter[VolumeResourceRecord]{Kind: "volumes", IdentityFind: true, ValidateID: findSegment, IdentityDirectGet: findSegment, IdentityResponseID: func(value *VolumeResourceRecord) (string, error) {
		if value == nil || value.Resource == nil {
			return "", findInvalid("find response resource is missing")
		}
		// Source Body(id) is untyped. A direct successful GET returns it even
		// if numeric/null. For list matching, only an actual string ID equals
		// the caller's string identity; names remain a separate comparison.
		return findText(value, "id"), nil
	}, Name: func(value *VolumeResourceRecord) string { return findText(value, "name") }, Get: get, Iterate: list})
	value, err := collection.FindIdentity(ctx, identity, resource.WithIdentityFindIgnoreMissing(policy.IgnoreMissing == nil || *policy.IgnoreMissing))
	return value, wrap(errors.Join(err, guard(ctx)))
}
