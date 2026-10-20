package blockstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func (p *preparedVolumeTypes) find(ctx context.Context, nameOrID string) (*GetVolumeTypeResult, error) {
	result := &GetVolumeTypeResult{}
	record, err := p.identityCollection(ctx, result).FindIdentity(ctx, nameOrID, resource.WithIdentityFindQuery("is_public", "none"))
	if err != nil {
		if ctx.Err() != nil && p.memberFailure != nil {
			err = errors.Join(p.memberFailure, err)
		}
		return result, err
	}
	if err := p.reader.guard(ctx); err != nil {
		return result, err
	}
	if record != nil {
		result.Value = bytes.Clone(record.view)
		result.Type = record.entry.value
	}
	return result, nil
}

type volumeTypeIdentityRecord struct {
	entry    volumeTypeEntry
	view     json.RawMessage
	id, name string
}

// This library-owned binding shares exact identity policy. Type's source query
// mapping has no name key, so fallback lists retain only the visibility query.
func (p *preparedVolumeTypes) identityCollection(ctx context.Context, result *GetVolumeTypeResult) *resource.Collection[volumeTypeIdentityRecord] {
	get := func(ctx context.Context, id string) (*volumeTypeIdentityRecord, error) {
		return p.member(ctx, id, url.Values{"is_public": {"none"}}, result)
	}
	getQuery := func(ctx context.Context, id string, query url.Values) (*volumeTypeIdentityRecord, error) {
		return p.member(ctx, id, query, result)
	}
	iterate := func(ctx context.Context, query url.Values, details bool) iter.Seq2[*volumeTypeIdentityRecord, error] {
		return func(yield func(*volumeTypeIdentityRecord, error) bool) {
			p.memberFailure = nil
			reader := *p.reader
			initial := *reader.initialURL
			initial.RawQuery = query.Encode()
			reader.initialURL = &initial
			prepared := *p
			prepared.reader = &reader
			proof := &ListVolumeTypesResult{}
			err := prepared.read(ctx, proof, func(entry volumeTypeEntry) (bool, error) {
				record, err := prepared.identityRecord(ctx, entry, false)
				if err != nil {
					return false, err
				}
				return yield(record, nil), nil
			})
			result.Pages = append(result.Pages, proof.Pages...)
			if err != nil {
				yield(nil, err)
			}
		}
	}
	return resource.NewCollection(resource.Adapter[volumeTypeIdentityRecord]{
		Kind: "volume type", IdentityFind: true,
		Get: get, GetIdentityQuery: getQuery, IterateIdentity: iterate,
		ID:                 func(value *volumeTypeIdentityRecord) string { return value.id },
		IdentityResponseID: func(value *volumeTypeIdentityRecord) (string, error) { return value.id, nil },
		Name:               func(value *volumeTypeIdentityRecord) string { return value.name },
	})
}

func (p *preparedVolumeTypes) identityRecord(ctx context.Context, entry volumeTypeEntry, member bool) (*volumeTypeIdentityRecord, error) {
	view, err := p.view(ctx, entry, member)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(view, &fields); err != nil {
		return nil, entry.origin.Fail(err)
	}
	// Untyped response attributes are compared to strings without coercion.
	stringField := func(raw json.RawMessage) string {
		var value string
		if len(raw) > 0 && raw[0] == '"' {
			_ = json.Unmarshal(raw, &value)
		}
		return value
	}
	return &volumeTypeIdentityRecord{entry: entry, view: view, id: stringField(fields["id"]), name: stringField(fields["name"])}, p.reader.guard(ctx)
}
