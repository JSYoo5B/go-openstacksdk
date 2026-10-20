package addressgroups

import (
	"bytes"
	"context"
	"errors"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// AddressGroupListRecord reuses the existing Source view/evidence model.
// Resource contains descriptor values, Wire is the actual row, and Envelope
// is the whole page. This Resource has neither tag defaults nor Header attrs.
type AddressGroupListRecord = AddressGroupFindRecord

// listRecordProfile derives the exact private class descriptors/query mapping
// already verified for Find, while opting only this new List into Source policy.
func listRecordProfile() neutronfind.Profile {
	p := neutronfind.Profile{Kind: "addressgroups", Path: "address-groups", Singular: "address_group", Plural: "address_groups", IDKey: "id", NameKey: "name", PreserveUnknownBody: true, SourceFindPolicy: true, StopFaultRetryCallbacks: true, VersionedPath: "/v2.0/address-groups"}
	for _, field := range findDescriptors {
		p.Body = append(p.Body, neutronfind.Descriptor{Name: field.name, Wire: field.wire, Kind: neutronfind.DescriptorKind(field.kind), Missing: field.missing, Alias: field.alias})
	}
	for _, pair := range findQueryPairs {
		p.Query = append(p.Query, neutronfind.QueryPair{Name: pair.name, Wire: pair.wire})
	}
	return p
}
func listRecordWrap(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "addressgroups", cloudread.ContextError(ctx, err))
}
func listRecordFromEngine(value *neutronfind.Record) *AddressGroupListRecord {
	if value == nil {
		return nil
	}
	return &AddressGroupListRecord{Resource: value.Resource.Clone(), Wire: value.Wire.Clone(), Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(), StatusCode: value.StatusCode, row: value.Row(), fromList: value.FromList()}
}

// ListRecords lazily supplies the pinned Source address_groups query, Body
// descriptor/filter/default and complete owned paging contracts. No name hint
// or identity lookup is added; native List/All/FindRecord retain their profiles.
func (a *API) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*AddressGroupListRecord, error] {
	snapshot := listRecordOptionSnapshot(options)
	return func(yield func(*AddressGroupListRecord, error) bool) {
		const operation = "ListRecords"
		fail := func(err error) { yield(nil, listRecordWrap(ctx, operation, err)) }
		if err := cloudread.Context(ctx); err != nil {
			fail(err)
			return
		}
		if a == nil {
			fail(findInvalid("address-groups API required"))
			return
		}
		selected := a.client
		engine, err := neutronfind.New(ctx, selected, listRecordProfile(), func(context.Context) error {
			if a.client != selected {
				return findInvalid("selected address-groups list client changed")
			}
			return nil
		})
		if err != nil {
			fail(err)
			return
		}
		config := RecordListOpts{}
		for _, apply := range snapshot {
			if err := engine.Check(ctx); err != nil {
				fail(err)
				return
			}
			if apply == nil {
				fail(findInvalid("nil list record option"))
				return
			}
			next := copyListRecordOptions(config)
			if err := errors.Join(apply(&next), engine.Check(ctx)); err != nil {
				fail(err)
				return
			}
			config = copyListRecordOptions(next)
		}
		for key := range config.Query {
			if err := listRecordKey(key); err != nil {
				fail(err)
				return
			}
		}
		if err := engine.Check(ctx); err != nil {
			fail(err)
			return
		}
		for value, err := range engine.List(ctx, listEngineOptions(config)) {
			if !yield(listRecordFromEngine(value), listRecordWrap(ctx, operation, err)) {
				return
			}
		}
	}
}

// AllRecords collects ListRecords and drops partial rows on any failure.
func (a *API) AllRecords(ctx context.Context, options ...RecordListOption) ([]*AddressGroupListRecord, error) {
	var values []*AddressGroupListRecord
	for value, err := range a.ListRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
