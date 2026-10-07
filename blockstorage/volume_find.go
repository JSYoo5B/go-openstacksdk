package blockstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/url"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// find is the prepared ordinary lookup shared by GetVolume and VolumeExists.
// It applies no search/glob/filter phase and never reruns original options.
func (p *preparedVolumeSearch) find(ctx context.Context, nameOrID string) (*GetVolumeResult, error) {
	result, _, err := p.findSelected(ctx, nameOrID)
	return result, err
}

// findSelected also retains the already selected wire row. Mutation workflows
// need its raw attribute order and values for dirty comparison, without another
// lookup or consuming unused list rows. Ordinary read behavior remains shared.
func (p *preparedVolumeSearch) findSelected(ctx context.Context, nameOrID string) (*GetVolumeResult, *volumeIdentityRecord, error) {
	result := &GetVolumeResult{}
	collection := p.identityCollection(ctx, result)
	record, err := collection.FindIdentity(ctx, nameOrID)
	if err != nil {
		// Shared identity lookup gives cancellation precedence after GET.
		// Retain this member's actual read/Close/source failure as well.
		// The fallback iterator clears it before a later list can fail.
		if ctx.Err() != nil && p.memberFailure != nil {
			err = errors.Join(p.memberFailure, err)
		}
		return result, nil, err
	}
	if err := p.reader.guard(ctx); err != nil {
		return result, nil, err
	}
	if record != nil {
		result.Value = bytes.Clone(record.view)
		result.Volume = record.entry.volume
	}
	return result, record, nil
}

type volumeIdentityRecord struct {
	entry    getVolumesEntry
	view     json.RawMessage
	id, name string
}

// The binding is library owned. It reuses shared FindIdentity policy while
// retaining actual complete wire rows instead of reserializing native Volume.
func (p *preparedVolumeSearch) identityCollection(ctx context.Context, result *GetVolumeResult) *resource.Collection[volumeIdentityRecord] {
	get := func(ctx context.Context, id string) (*volumeIdentityRecord, error) {
		return p.member(ctx, id, result)
	}
	iterate := func(ctx context.Context, query url.Values, details bool) iter.Seq2[*volumeIdentityRecord, error] {
		return func(yield func(*volumeIdentityRecord, error) bool) {
			p.memberFailure = nil
			reader := *p.reader
			initial := *reader.initialURL
			initial.RawQuery = query.Encode()
			reader.initialURL = &initial
			proof := &GetVolumesResult{}
			err := reader.readVolumes(ctx, proof, func(entry getVolumesEntry) (bool, error) {
				record, err := p.identityRecord(ctx, entry)
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
	return resource.NewCollection(resource.Adapter[volumeIdentityRecord]{Kind: "volume", IdentityFind: true,
		Get: get, IterateIdentity: iterate, IdentityAllProjectsQuery: "all_tenants",
		ID:                 func(value *volumeIdentityRecord) string { return value.id },
		IdentityResponseID: func(value *volumeIdentityRecord) (string, error) { return value.id, nil },
		Name:               func(value *volumeIdentityRecord) string { return value.name }, NameQuery: func(name string) string { return name },
	})
}

func (p *preparedVolumeSearch) identityRecord(ctx context.Context, entry getVolumesEntry) (*volumeIdentityRecord, error) {
	view, err := p.view(ctx, entry)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(view, &fields); err != nil {
		return nil, entry.origin.Fail(err)
	}
	// Ordinary Resource.find compares untyped attributes directly to a string;
	// numeric/bool/container values do not become textual identity matches.
	stringField := func(raw json.RawMessage) string {
		var value string
		if len(raw) > 0 && raw[0] == '"' {
			_ = json.Unmarshal(raw, &value)
		}
		return value
	}
	return &volumeIdentityRecord{entry: entry, view: view, id: stringField(fields["id"]), name: stringField(fields["name"])}, p.reader.guard(ctx)
}
