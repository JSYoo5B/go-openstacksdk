package blockstorage

import (
	"bytes"
	"context"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudfilter"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// SearchVolumeTypes fully lists and normalizes types before exact/glob identity
// selection and lazy mapping or JMESPath filters. It adds no server filter query.
func SearchVolumeTypes(ctx context.Context, cinder *gophercloud.ServiceClient, input SearchVolumeTypesRequest, options ...VolumeTypeSearchOption) (*SearchVolumeTypesResult, error) {
	p, err := captureVolumeTypesSearch(ctx, cinder, options)
	if err != nil {
		return nil, wrapVolumeTypeError(ctx, "SearchVolumeTypes", err)
	}
	result, err := p.search(ctx, input.NameOrID)
	return result, wrapVolumeTypeError(ctx, "SearchVolumeTypes", err)
}

func (p *preparedVolumeTypes) search(ctx context.Context, nameOrID string) (*SearchVolumeTypesResult, error) {
	entries, views, proof, err := p.collect(ctx)
	result := &SearchVolumeTypesResult{Pages: proof.Pages}
	if err != nil {
		return result, err
	}
	selected, err := cloudfilter.Select(views, nameOrID, p.options.Filters, func() error { return p.reader.guard(ctx) })
	if err != nil {
		return result, fmt.Errorf("%w: local volume type search: %w", resource.ErrInvalidOption, err)
	}
	if err := p.reader.guard(ctx); err != nil {
		return result, err
	}
	if !selected.Expression {
		result.Types = make([]*resource.RawResource, len(selected.Indices))
		for i, index := range selected.Indices {
			result.Types[i] = entries[index].value
		}
	}
	result.Value = bytes.Clone(selected.Value)
	return result, nil
}
