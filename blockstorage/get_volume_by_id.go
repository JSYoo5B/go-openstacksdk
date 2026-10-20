package blockstorage

import (
	"bytes"
	"context"
	"unicode"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// GetVolumeByID performs one logical member GET, without name search or list
// fallback. Rejected statuses including 400/403/404 remain errors. The ID must
// be one unescaped UTF-8 URL path segment without whitespace or controls.
func GetVolumeByID(ctx context.Context, cinder *gophercloud.ServiceClient, input GetVolumeByIDRequest, options ...VolumeReadOption) (*GetVolumeResult, error) {
	p, err := captureVolumeRead(ctx, cinder, options)
	if err != nil {
		return nil, wrapVolumeSearchError(ctx, "GetVolumeByID", err)
	}
	if err := validateVolumeReadID(input.ID); err != nil {
		return nil, wrapVolumeSearchError(ctx, "GetVolumeByID", err)
	}
	result := &GetVolumeResult{}
	record, err := p.member(ctx, input.ID, result)
	if err != nil {
		return result, wrapVolumeSearchError(ctx, "GetVolumeByID", err)
	}
	if err := p.reader.guard(ctx); err != nil {
		return result, wrapVolumeSearchError(ctx, "GetVolumeByID", err)
	}
	result.Value = bytes.Clone(record.view)
	result.Volume = record.entry.volume
	return result, nil
}

func validateVolumeReadID(id string) error {
	if err := resource.ID(id).Validate(); err != nil {
		return err
	}
	if !attachText(id) {
		return attachInvalid("volume ID must be valid UTF-8 without controls")
	}
	for _, char := range id {
		if unicode.IsSpace(char) {
			return attachInvalid("volume ID must not contain whitespace")
		}
	}
	return nil
}
