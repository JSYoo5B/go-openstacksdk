package blockstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

type volumeTypeEntry struct {
	value  *resource.RawResource
	origin *rest.Response
	raw    json.RawMessage
	index  int
}

// read uses raw entries so an exact find can stop before an unused invalid
// descriptor or row, without the native pager decoding the whole typed page.
func (p *preparedVolumeTypes) read(ctx context.Context, proof *ListVolumeTypesResult, consume func(volumeTypeEntry) (bool, error)) error {
	current := p.reader.initialURL
	seen := make(map[string]bool)
	for {
		if err := p.reader.guard(ctx); err != nil {
			return err
		}
		key, err := p.reader.pageKey(current)
		if err != nil {
			return err
		}
		if seen[key] {
			return &resource.PaginationCycleError{URL: current.String()}
		}
		seen[key] = true
		response, err := p.reader.exchange(ctx, current)
		if response != nil {
			proof.Pages = append(proof.Pages, &VolumeTypesPage{Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode})
		}
		if err != nil {
			return err
		}
		fields, err := attachmentObject(response.Body)
		if err != nil {
			return response.Fail(err)
		}
		raw, present := fields["volume_types"]
		value := bytes.TrimSpace(raw)
		if !present || len(value) == 0 || value[0] != '[' {
			return response.Fail(attachInvalid("type list must contain a nonnull canonical volume_types array"))
		}
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil {
			return response.Fail(err)
		}
		for index, row := range rows {
			if err := p.reader.guard(ctx); err != nil {
				return err
			}
			var value resource.RawResource
			if err := json.Unmarshal(row, &value); err != nil {
				return response.Fail(fmt.Errorf("volume_types[%d]: %w", index, err))
			}
			value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
			more, err := consume(volumeTypeEntry{value: &value, origin: response, raw: bytes.Clone(row), index: index})
			if err != nil {
				return err
			}
			if !more {
				return p.reader.guard(ctx)
			}
		}
		if err := p.reader.guard(ctx); err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		next, err := volumeTypesNext(response, current, fields)
		if err != nil {
			return response.Fail(err)
		}
		if next == nil {
			return p.reader.guard(ctx)
		}
		// Resolver chooses source precedence and query merging; the shared reader
		// admits only the frozen collection's origin and escaped path.
		next, err = p.reader.continuation(current, next.String())
		if err != nil {
			return response.Fail(err)
		}
		key, err = p.reader.pageKey(next)
		if err != nil {
			return response.Fail(err)
		}
		if seen[key] {
			return response.Fail(&resource.PaginationCycleError{URL: next.String()})
		}
		current = next
	}
}

func (p *preparedVolumeTypes) view(ctx context.Context, entry volumeTypeEntry, member bool) (json.RawMessage, error) {
	if err := p.reader.guard(ctx); err != nil {
		return nil, err
	}
	base, err := volumeTypeView(entry.raw, nil, member)
	if err != nil {
		return nil, entry.origin.Fail(fmt.Errorf("volume_types[%d]: %w", entry.index, err))
	}
	wireLocation, err := volumeTypeUsesWireLocation(entry.raw, member)
	if err != nil {
		return nil, entry.origin.Fail(err)
	}
	if wireLocation {
		return base, p.reader.guard(ctx)
	}
	location, err := p.currentLocation()
	if err != nil {
		return nil, err
	}
	view, err := volumeTypeView(entry.raw, location, member)
	if err != nil {
		return nil, entry.origin.Fail(err)
	}
	return view, p.reader.guard(ctx)
}

func (p *preparedVolumeTypes) collect(ctx context.Context) ([]volumeTypeEntry, []json.RawMessage, *ListVolumeTypesResult, error) {
	entries := make([]volumeTypeEntry, 0)
	views := make([]json.RawMessage, 0)
	proof := &ListVolumeTypesResult{}
	err := p.read(ctx, proof, func(entry volumeTypeEntry) (bool, error) {
		view, err := p.view(ctx, entry, false)
		if err != nil {
			return false, err
		}
		entries, views = append(entries, entry), append(views, view)
		return true, nil
	})
	if err != nil {
		return nil, nil, proof, err
	}
	return entries, views, proof, nil
}
