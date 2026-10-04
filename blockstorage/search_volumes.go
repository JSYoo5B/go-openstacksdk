package blockstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cloudfilter"
	"gophercloudsdk/internal/cloudlocation"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

type preparedVolumeSearch struct {
	reader        *preparedGetVolumes
	options       VolumeSearchOpts
	memberFailure error
}

// SearchVolumes materializes and normalizes Cinder's complete detail list,
// then applies exact/glob identity and lazy dictionary or JMESPath filters.
// All row conversions precede continuation and local filter evaluation.
func SearchVolumes(ctx context.Context, cinder *gophercloud.ServiceClient, input SearchVolumesRequest, options ...VolumeSearchOption) (*SearchVolumesResult, error) {
	p, err := captureVolumeSearch(ctx, cinder, options)
	if err != nil {
		return nil, wrapVolumeSearchError(ctx, "SearchVolumes", err)
	}
	result, err := p.search(ctx, input.NameOrID)
	return result, wrapVolumeSearchError(ctx, "SearchVolumes", err)
}

func captureVolumeSearch(ctx context.Context, cinder *gophercloud.ServiceClient, options []VolumeSearchOption) (*preparedVolumeSearch, error) {
	reader, err := captureGetVolumes(ctx, cinder, GetVolumesRequest{}, nil)
	if err != nil {
		return nil, err
	}
	p := &preparedVolumeSearch{reader: reader}
	p.options, err = applyVolumeSearchOptions(options, func() error { return reader.guard(ctx) })
	if err != nil {
		return nil, err
	}
	return p, reader.guard(ctx)
}

func (p *preparedVolumeSearch) search(ctx context.Context, nameOrID string) (*SearchVolumesResult, error) {
	proof := &GetVolumesResult{}
	result := &SearchVolumesResult{}
	entries := make([]getVolumesEntry, 0)
	views := make([]json.RawMessage, 0)
	err := p.reader.readVolumes(ctx, proof, func(entry getVolumesEntry) (bool, error) {
		view, err := p.view(ctx, entry)
		if err != nil {
			return false, err
		}
		entries = append(entries, entry)
		views = append(views, view)
		return true, nil
	})
	result.Pages = proof.Pages
	if err != nil {
		return result, err
	}
	selected, err := cloudfilter.Select(views, nameOrID, p.options.Filters, func() error { return p.reader.guard(ctx) })
	if err != nil {
		return result, fmt.Errorf("%w: local volume search: %w", resource.ErrInvalidOption, err)
	}
	if err := p.reader.guard(ctx); err != nil {
		return result, err
	}
	if !selected.Expression {
		result.Volumes = make([]*resource.RawResource, len(selected.Indices))
		for i, index := range selected.Indices {
			result.Volumes[i] = entries[index].volume
		}
	}
	result.Value = bytes.Clone(selected.Value)
	return result, nil
}

func (p *preparedVolumeSearch) view(ctx context.Context, entry getVolumesEntry) (json.RawMessage, error) {
	if err := p.reader.guard(ctx); err != nil {
		return nil, err
	}
	// The first pass consumes aliases in wire order. Computed location uses
	// those exact normalized project/zone values, never a random map alias.
	base, err := volumeSearchView(entry.raw, nil)
	if err != nil {
		return nil, entry.origin.Fail(fmt.Errorf("volumes[%d]: %w", entry.index, err))
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(base, &fields); err != nil {
		return nil, entry.origin.Fail(err)
	}
	var location resource.CloudLocation
	if p.options.Location != nil {
		location = p.options.Location.Clone()
	} else {
		location.Project.ID, err = cloudlocation.ProjectID(p.reader.cinder.provider)
		if err != nil {
			return nil, err
		}
	}
	computed, err := location.ForResource(fields["project_id"], fields["availability_zone"])
	if err != nil {
		return nil, err
	}
	view, err := volumeSearchView(entry.raw, computed)
	if err != nil {
		return nil, entry.origin.Fail(fmt.Errorf("volumes[%d]: %w", entry.index, err))
	}
	return view, p.reader.guard(ctx)
}

func wrapVolumeSearchError(ctx context.Context, operation string, err error) error {
	if err == nil {
		return nil
	}
	return request.Wrap(operation, "volume", attachContextError(ctx, err))
}
