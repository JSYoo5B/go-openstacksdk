package blockstorage

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cloudfilter"
	"gophercloudsdk/resource"
)

// GetVolumeType uses exact GET-first lookup with is_public=none when filters
// are omitted/null. Every nonnull filter uses full SearchVolumeTypes selection.
func GetVolumeType(ctx context.Context, cinder *gophercloud.ServiceClient, input GetVolumeTypeRequest, options ...VolumeTypeSearchOption) (*GetVolumeTypeResult, error) {
	p, err := captureVolumeTypesSearch(ctx, cinder, options)
	if err != nil {
		return nil, wrapVolumeTypeError(ctx, "GetVolumeType", err)
	}
	if p.options.Filters == nil || bytes.Equal(bytes.TrimSpace(*p.options.Filters), []byte("null")) {
		result, err := p.find(ctx, input.NameOrID)
		return result, wrapVolumeTypeError(ctx, "GetVolumeType", err)
	}
	search, err := p.search(ctx, input.NameOrID)
	result := &GetVolumeTypeResult{Pages: search.Pages}
	if err != nil {
		return result, wrapVolumeTypeError(ctx, "GetVolumeType", err)
	}
	selected, err := cloudfilter.First(search.Value)
	if err != nil {
		var multiple *cloudfilter.MultipleError
		if errors.As(err, &multiple) {
			err = &VolumeTypeSelectionError{NameOrID: input.NameOrID, Length: multiple.Length}
		} else {
			err = fmt.Errorf("%w: local type selection: %w", resource.ErrInvalidOption, err)
		}
		return result, wrapVolumeTypeError(ctx, "GetVolumeType", err)
	}
	if err := p.reader.guard(ctx); err != nil {
		return result, wrapVolumeTypeError(ctx, "GetVolumeType", err)
	}
	result.Value = bytes.Clone(selected)
	if selected != nil && len(search.Types) == 1 {
		result.Type = search.Types[0]
	}
	return result, nil
}
