package blockstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cloudfilter"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

// GetVolume uses shared GET-first exact identity lookup when filters are
// omitted or null. Every explicit nonnull filter uses full SearchVolumes and
// Python's outer truthiness, len, index-0 selection on its arbitrary JSON result.
func GetVolume(ctx context.Context, cinder *gophercloud.ServiceClient, input GetVolumeRequest, options ...VolumeSearchOption) (*GetVolumeResult, error) {
	p, err := captureVolumeSearch(ctx, cinder, options)
	if err != nil {
		return nil, wrapVolumeSearchError(ctx, "GetVolume", err)
	}
	result := &GetVolumeResult{}
	filtered := p.options.Filters != nil && !bytes.Equal(bytes.TrimSpace(*p.options.Filters), []byte("null"))
	if filtered {
		search, err := p.search(ctx, input.NameOrID)
		result.Pages = search.Pages
		if err != nil {
			return result, wrapVolumeSearchError(ctx, "GetVolume", err)
		}
		selected, err := cloudfilter.First(search.Value)
		if err != nil {
			var multiple *cloudfilter.MultipleError
			if errors.As(err, &multiple) {
				err = &VolumeSelectionError{NameOrID: input.NameOrID, Length: multiple.Length}
			} else {
				err = fmt.Errorf("%w: local volume selection: %w", resource.ErrInvalidOption, err)
			}
			return result, wrapVolumeSearchError(ctx, "GetVolume", err)
		}
		if err := p.reader.guard(ctx); err != nil {
			return result, wrapVolumeSearchError(ctx, "GetVolume", err)
		}
		result.Value = bytes.Clone(selected)
		if selected != nil && len(search.Volumes) == 1 {
			result.Volume = search.Volumes[0]
		}
		return result, nil
	}
	collection := p.identityCollection(ctx, result)
	record, err := collection.FindIdentity(ctx, input.NameOrID)
	if err != nil {
		return result, wrapVolumeSearchError(ctx, "GetVolume", err)
	}
	if err := p.reader.guard(ctx); err != nil {
		return result, wrapVolumeSearchError(ctx, "GetVolume", err)
	}
	if record != nil {
		result.Value = bytes.Clone(record.view)
		result.Volume = record.entry.volume
	}
	return result, nil
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
		if err := p.reader.guard(ctx); err != nil {
			return nil, err
		}
		target := p.reader.cinder.client.ServiceURL("volumes", url.PathEscape(id))
		if err := validateAttachTarget(&p.reader.cinder.client, target); err != nil {
			return nil, err
		}
		response, err := rest.DoJSON(ctx, &p.reader.cinder.client, http.MethodGet, target, nil, nil, http.StatusOK)
		if response != nil {
			result.Observed = &GetVolumesPage{Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
		}
		if observed := p.reader.guard(ctx); observed != nil {
			err = errors.Join(err, observed)
			if response != nil {
				err = response.Fail(err)
			}
		}
		if err != nil {
			return nil, attachContextError(ctx, err)
		}
		fields, err := attachmentObject(response.Body)
		if err != nil {
			return nil, response.Fail(err)
		}
		row, present := fields["volume"]
		if !present {
			return nil, response.Fail(attachInvalid("member response must contain canonical volume object"))
		}
		var volume resource.RawResource
		if err := json.Unmarshal(row, &volume); err != nil {
			return nil, response.Fail(err)
		}
		volume.Header, volume.StatusCode = response.Header.Clone(), response.StatusCode
		return p.identityRecord(ctx, getVolumesEntry{volume: &volume, origin: response, raw: bytes.Clone(row)})
	}
	iterate := func(ctx context.Context, query url.Values, details bool) iter.Seq2[*volumeIdentityRecord, error] {
		return func(yield func(*volumeIdentityRecord, error) bool) {
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
