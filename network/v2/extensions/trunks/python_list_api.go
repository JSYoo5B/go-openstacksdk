package trunks

import (
	"context"
	"errors"
	"iter"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// TrunkListRecord reuses the existing owned Trunk descriptor and HTTP receipt
// model. Resource, Wire and Envelope remain independent received snapshots.
type TrunkListRecord = TrunkFindRecord

// Trunk inherits Resource + TagMixin, not NetworkResource. Reuse the existing
// reviewed descriptor/query inventory; no revision or If-Match is invented.
var listRecordProfile = func() neutronfind.Profile {
	profile := neutronfind.Profile{Kind: "trunks", Path: "trunks", Singular: "trunk", Plural: "trunks", IDKey: "id", NameKey: "name", PreserveUnknownBody: true, VersionedPath: "/v2.0/trunks", SourceFindPolicy: true, StopFaultRetryCallbacks: true}
	for _, field := range findDescriptors {
		profile.Body = append(profile.Body, neutronfind.Descriptor{Name: field.name, Wire: field.wire, Kind: neutronfind.DescriptorKind(field.kind), Missing: field.missing, Alias: field.alias})
	}
	for _, pair := range findQueryPairs {
		profile.Query = append(profile.Query, neutronfind.QueryPair{Name: pair.name, Wire: pair.wire})
	}
	return profile
}()

func listRecordWrap(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "trunks", cloudread.ContextError(ctx, err))
}
func (a *API) listRecordEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, findInvalid("configured trunks API required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, listRecordProfile, func(context.Context) error {
		if a.client != selected {
			return findInvalid("selected trunks API client changed")
		}
		return nil
	})
}
func listRecord(value *neutronfind.Record) *TrunkListRecord {
	if value == nil {
		return nil
	}
	return &TrunkListRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode, row: value.Row(), fromList: value.FromList()}
}

// ListRecords maps the pinned trunks generator. It is lazy and reusable;
// each iteration captures the source before applying independently owned
// options. Explicit microversion applies to every page, without a member GET.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*TrunkListRecord, error] {
	snapshot := slices.Clone(options)
	return func(yield func(*TrunkListRecord, error) bool) {
		engine, err := a.listRecordEngine(ctx)
		if err != nil {
			yield(nil, listRecordWrap(ctx, "ListRecords", err))
			return
		}
		policy := copyRecordList(RecordListOpts{})
		for _, apply := range snapshot {
			if err := engine.Check(ctx); err != nil {
				yield(nil, listRecordWrap(ctx, "ListRecords", err))
				return
			}
			if apply == nil {
				yield(nil, listRecordWrap(ctx, "ListRecords", findInvalid("nil list option")))
				return
			}
			next := copyRecordList(policy)
			if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
				yield(nil, listRecordWrap(ctx, "ListRecords", err))
				return
			}
			policy = copyRecordList(next)
		}
		for key := range policy.Query {
			if key == "allow_unknown_params" || key == "jmespath_filters" {
				yield(nil, listRecordWrap(ctx, "ListRecords", findInvalid("Source control %q is outside this owned record list", key)))
				return
			}
		}
		for key := range policy.Headers {
			if strings.EqualFold(key, "OpenStack-API-Version") || strings.HasPrefix(strings.ToLower(key), "x-openstack-") && strings.HasSuffix(strings.ToLower(key), "-api-version") {
				yield(nil, listRecordWrap(ctx, "ListRecords", findInvalid("protocol Header %q requires owned microversion policy", key)))
				return
			}
		}
		for value, err := range engine.List(ctx, recordListOptions(policy)) {
			if !yield(listRecord(value), listRecordWrap(ctx, "ListRecords", err)) {
				return
			}
		}
	}
}

// AllRecords returns all rows only after a successful complete observed list.
// Late HTTP, page, projection, source or cancellation errors discard partials.
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*TrunkListRecord, error) {
	var values []*TrunkListRecord
	for value, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, listRecordWrap(ctx, "AllRecords", err)
		}
		values = append(values, value)
	}
	return values, nil
}
