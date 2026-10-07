package blockstorage

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudfilter"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
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
	result, err = p.find(ctx, input.NameOrID)
	return result, wrapVolumeSearchError(ctx, "GetVolume", err)
}
