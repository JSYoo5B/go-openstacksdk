package snapshots

import (
	"bytes"
	"context"
	"errors"
	"iter"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cinderrequest"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func findSegment(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return findInvalid("nonblank UTF-8 identity required")
	}
	for _, char := range id {
		if unicode.IsControl(char) {
			return findInvalid("identity has controls")
		}
	}
	return nil
}
func findCodes() []int {
	v := make([]int, 200)
	for i := range v {
		v[i] = i + 200
	}
	return v
}
func findRejections() []int {
	v := make([]int, 200)
	for i := range v {
		v[i] = i + 400
	}
	return v
}

// FindRecord follows Source find_snapshot's member-first exact-ID-or-name
// fallback with details and ignore-missing enabled and all-projects omitted.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*SnapshotRecord, error) {
	const operation = "FindRecord"
	wrap := func(err error) error { return request.Wrap(operation, "snapshots", cloudread.ContextError(ctx, err)) }
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(findInvalid("snapshot API required"))
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "volume")
	if err != nil {
		return nil, wrap(err)
	}
	guard := func(ctx context.Context) error {
		var changed error
		if a.client != selected {
			changed = findInvalid("selected snapshot client changed")
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	policy := RecordFindOpts{}
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(findInvalid("nil snapshot find option"))
		}
		next := copyFindOpts(policy)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyFindOpts(next)
	}
	query, filters, control, err := prepareFindQuery(policy, identity)
	if err == nil {
		err = findSegment(identity)
	}
	if err == nil {
		err = errors.Join(source.WithPolicy(ctx, nil, policy.Headers), guard(ctx))
	}
	if err != nil {
		return nil, wrap(err)
	}
	version, discovered, err := cinderrequest.Negotiate(ctx, source, "3.65")
	observations := make([]RecordObservation, len(discovered))
	for i, wire := range discovered {
		observations[i] = RecordObservation{Envelope: bytes.Clone(wire.Body), Header: wire.Header.Clone(), StatusCode: wire.StatusCode}
	}
	proof := func(value *SnapshotRecord) *SnapshotRecord {
		if value == nil {
			return nil
		}
		value.Discovery = make([]RecordObservation, len(observations))
		for i, observed := range observations {
			value.Discovery[i] = RecordObservation{Envelope: bytes.Clone(observed.Envelope), Header: observed.Header.Clone(), StatusCode: observed.StatusCode}
		}
		return value
	}
	if err = errors.Join(err, guard(ctx)); err == nil {
		err = source.WithPolicy(ctx, &version, nil)
	}
	if err != nil {
		if len(observations) != 0 {
			return proof(&SnapshotRecord{}), wrap(err)
		}
		return nil, wrap(err)
	}
	fixed := rest.RequestHeaderPolicy{Values: make(map[string]string)}
	if version != "" {
		fixed.Values["OpenStack-API-Version"] = "volume " + version
		fixed.Values["X-OpenStack-Volume-API-Version"] = version
	} else {
		fixed.Absent = []string{"OpenStack-API-Version", "X-OpenStack-Volume-API-Version"}
	}
	read := func(target string, codes ...int) (*rest.Response, error) {
		return rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &source.Client, guard, http.MethodGet, target, nil, fixed, rest.RejectionPolicy{Codes: findRejections(), PreserveCleanRetry: true}, codes...)
	}
	var failedMember *SnapshotRecord
	get := func(_ context.Context, id string) (*SnapshotRecord, error) {
		wire, err := read(source.Client.ServiceURL("snapshots", url.PathEscape(id)), findCodes()...)
		if err != nil {
			if wire != nil {
				failedMember = &SnapshotRecord{Envelope: bytes.Clone(wire.Body), Header: wire.Header.Clone(), StatusCode: wire.StatusCode}
			}
			return failedMember, err
		}
		value, err := findResponse(wire, id)
		if err = errors.Join(err, guard(ctx)); err != nil {
			failedMember = value
			err = wire.Fail(err)
		}
		return value, err
	}
	path := "snapshots/detail"
	if policy.Details != nil && !*policy.Details {
		path = "snapshots"
	}
	list := func(context.Context, url.Values) iter.Seq2[*SnapshotRecord, error] {
		return func(yield func(*SnapshotRecord, error) bool) {
			var origin *rest.Response
			headers := source.Client.MoreHeaders
			if _, exists := headers["Accept"]; !exists {
				headers["Accept"] = "application/json"
			}
			offsetPaging := query.Has("offset") && !query.Has("marker")
			spec := rest.CollectionSpec[SnapshotRecord]{Client: &source.Client, Path: path, Kind: "snapshots", PluralKey: "snapshots", Metadata: findMetadata, Validate: guard, SourceGuard: guard, ListCodes: findCodes(), ReadPage: func(_ context.Context, target string, codes ...int) (*rest.Response, error) {
				return read(target, codes...)
			}, ValidateResponse: func(wire *rest.Response) error { origin = wire; return guard(ctx) }, ValidateItem: func(value *SnapshotRecord) error {
				value.Envelope = bytes.Clone(origin.Body)
				value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
				return errors.Join(findProject(value, value.fields), guard(ctx))
			}, Paging: rest.PagePolicy[SnapshotRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, OffsetPagination: offsetPaging, MarkerFallback: !offsetPaging, Marker: findMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
			keys := make([]string, 0, len(filters))
			for key := range filters {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for value, err := range rest.ListWithControl(ctx, spec, query, control) {
				if err != nil {
					yield(nil, err)
					return
				}
				matched := true
				for _, key := range keys {
					equal, cause := findLocalValue(value.Resource.Body[key], filters[key])
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
	collection := resource.NewCollection(resource.Adapter[SnapshotRecord]{Kind: "snapshots", IdentityFind: true, ValidateID: findSegment, IdentityDirectGet: findSegment, IdentityResponseID: func(value *SnapshotRecord) (string, error) { return findText(value, "id"), nil }, Name: func(value *SnapshotRecord) string { return findText(value, "name") }, Get: get, Iterate: list})
	value, err := collection.FindIdentity(ctx, identity, resource.WithIdentityFindIgnoreMissing(policy.IgnoreMissing == nil || *policy.IgnoreMissing))
	if value == nil && err != nil && failedMember != nil {
		value = failedMember
	}
	return proof(value), wrap(errors.Join(err, guard(ctx)))
}
