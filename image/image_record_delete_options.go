package image

import (
	"bytes"
	"context"
	"errors"
	"maps"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
)

// ImageRecordDeleteOpts selects whole-image or one-store deletion. A nil
// IgnoreMissing means true. StoreID and StoreRecord are exclusive literal
// selectors; neither performs name lookup or store discovery.
type ImageRecordDeleteOpts struct {
	Headers       map[string]string
	StoreID       *string
	StoreRecord   *serviceinfo.StoreRecord
	IgnoreMissing *bool
}

type ImageRecordDeleteOption func(*ImageRecordDeleteOpts) error

// WithImageRecordDeleteOpts captures and replaces the entire configuration.
// Nil store selectors clear a previous store selection to whole-image mode.
func WithImageRecordDeleteOpts(value ImageRecordDeleteOpts) ImageRecordDeleteOption {
	snapshot := copyImageRecordDeleteOpts(value)
	return func(config *ImageRecordDeleteOpts) error {
		*config = copyImageRecordDeleteOpts(snapshot)
		return nil
	}
}

func WithImageRecordDeleteHeader(key, value string) ImageRecordDeleteOption {
	apply := WithImageMutationHeader(key, value)
	return func(config *ImageRecordDeleteOpts) error {
		return applyImageMemberHeader(&config.Headers, apply)
	}
}

func WithImageRecordDeleteHeaders(values map[string]string) ImageRecordDeleteOption {
	apply := WithImageMutationHeaders(values)
	return func(config *ImageRecordDeleteOpts) error {
		return applyImageMemberHeader(&config.Headers, apply)
	}
}

// WithImageRecordDeleteStoreID selects a literal store and clears StoreRecord.
// An explicitly empty ID is invalid rather than selecting whole-image mode.
func WithImageRecordDeleteStoreID(value string) ImageRecordDeleteOption {
	return func(config *ImageRecordDeleteOpts) error {
		snapshot := value
		config.StoreID, config.StoreRecord = &snapshot, nil
		return nil
	}
}

// WithImageRecordDeleteStoreRecord captures a store's Resource identity and
// clears StoreID. Nil clears both selectors to whole-image mode.
func WithImageRecordDeleteStoreRecord(value *serviceinfo.StoreRecord) ImageRecordDeleteOption {
	snapshot := cloneImageRecordDeleteStore(value)
	return func(config *ImageRecordDeleteOpts) error {
		config.StoreID, config.StoreRecord = nil, cloneImageRecordDeleteStore(snapshot)
		return nil
	}
}

func WithImageRecordDeleteIgnoreMissing(value bool) ImageRecordDeleteOption {
	return func(config *ImageRecordDeleteOpts) error {
		snapshot := value
		config.IgnoreMissing = &snapshot
		return nil
	}
}

func cloneImageRecordDeleteStore(value *serviceinfo.StoreRecord) *serviceinfo.StoreRecord {
	if value == nil {
		return nil
	}
	return &serviceinfo.StoreRecord{
		Resource: value.Resource.Clone(), Wire: value.Wire.Clone(),
		Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(),
		StatusCode: value.StatusCode,
	}
}

func copyImageRecordDeleteOpts(value ImageRecordDeleteOpts) ImageRecordDeleteOpts {
	value.Headers = maps.Clone(value.Headers)
	if value.Headers == nil {
		value.Headers = make(map[string]string)
	}
	if value.StoreID != nil {
		text := *value.StoreID
		value.StoreID = &text
	}
	value.StoreRecord = cloneImageRecordDeleteStore(value.StoreRecord)
	if value.IgnoreMissing != nil {
		flag := *value.IgnoreMissing
		value.IgnoreMissing = &flag
	}
	return value
}

func prepareImageRecordDeleteOptions(ctx context.Context, check func(context.Context) error, options []ImageRecordDeleteOption) (ImageRecordDeleteOpts, error) {
	config := copyImageRecordDeleteOpts(ImageRecordDeleteOpts{})
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return config, err
		}
		if apply == nil {
			return config, uploadInvalid("nil image record deletion option")
		}
		candidate := copyImageRecordDeleteOpts(config)
		if err := errors.Join(apply(&candidate), check(ctx)); err != nil {
			return config, err
		}
		config = copyImageRecordDeleteOpts(candidate)
	}
	var err error
	config.Headers, err = imageMutationHeaders(config.Headers, false, "")
	return config, errors.Join(err, check(ctx))
}
