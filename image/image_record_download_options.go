package image

import (
	"context"
	"errors"
	"hash"
	"slices"
	"unicode/utf8"
)

// ImageRecordDownloadHashFactory extends the SDK's primary digest selection.
// ErrImageRecordDownloadHashUnsupported selects the ordinary MD5 fallback.
// The SDK supplies all defaults; callers need a factory only for extra hashes.
type ImageRecordDownloadHashFactory func(string) (hash.Hash, error)

// ImageRecordDownloadOpts selects buffering, output or an unread stream.
// Output/path in the request takes precedence over Stream. Nil ChunkSize uses
// 1 MiB; nil VerifyChecksum verifies available hashes in consumed modes.
type ImageRecordDownloadOpts struct {
	Stream           bool
	ChunkSize        *int
	StorePreferences []string
	VerifyChecksum   *bool
	HashFactory      ImageRecordDownloadHashFactory
	Headers          map[string]string
}
type ImageRecordDownloadOption func(*ImageRecordDownloadOpts) error

func WithImageRecordDownloadOpts(value ImageRecordDownloadOpts) ImageRecordDownloadOption {
	snapshot := copyImageRecordDownloadOpts(value)
	return func(config *ImageRecordDownloadOpts) error {
		*config = copyImageRecordDownloadOpts(snapshot)
		return nil
	}
}
func WithImageRecordDownloadStream(value bool) ImageRecordDownloadOption {
	return func(config *ImageRecordDownloadOpts) error { config.Stream = value; return nil }
}
func WithImageRecordDownloadChunkSize(value int) ImageRecordDownloadOption {
	return func(config *ImageRecordDownloadOpts) error { owned := value; config.ChunkSize = &owned; return nil }
}
func WithImageRecordDownloadStorePreferences(values ...string) ImageRecordDownloadOption {
	snapshot := slices.Clone(values)
	return func(config *ImageRecordDownloadOpts) error {
		config.StorePreferences = slices.Clone(snapshot)
		return nil
	}
}
func WithImageRecordDownloadChecksumVerification(value bool) ImageRecordDownloadOption {
	return func(config *ImageRecordDownloadOpts) error {
		owned := value
		config.VerifyChecksum = &owned
		return nil
	}
}
func WithImageRecordDownloadHashFactory(value ImageRecordDownloadHashFactory) ImageRecordDownloadOption {
	return func(config *ImageRecordDownloadOpts) error { config.HashFactory = value; return nil }
}
func WithImageRecordDownloadHeader(key, value string) ImageRecordDownloadOption {
	return func(config *ImageRecordDownloadOpts) error {
		return mergeImageRecordHeaders(&config.Headers, map[string]string{key: value})
	}
}
func WithImageRecordDownloadHeaders(values map[string]string) ImageRecordDownloadOption {
	snapshot := copyImageRecordHeaders(values)
	return func(config *ImageRecordDownloadOpts) error { return mergeImageRecordHeaders(&config.Headers, snapshot) }
}
func copyImageRecordDownloadOpts(value ImageRecordDownloadOpts) ImageRecordDownloadOpts {
	value.ChunkSize = copyCreateImportPointer(value.ChunkSize)
	value.VerifyChecksum = copyCreateImportPointer(value.VerifyChecksum)
	value.StorePreferences = slices.Clone(value.StorePreferences)
	value.Headers = copyImageRecordHeaders(value.Headers)
	return value
}
func prepareImageRecordDownloadOptions(ctx context.Context, check func(context.Context) error, options []ImageRecordDownloadOption) (ImageRecordDownloadOpts, error) {
	var config ImageRecordDownloadOpts
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return config, err
		}
		if apply == nil {
			return config, uploadInvalid("nil image record download option")
		}
		candidate := copyImageRecordDownloadOpts(config)
		if err := errors.Join(apply(&candidate), check(ctx)); err != nil {
			return config, err
		}
		config = copyImageRecordDownloadOpts(candidate)
	}
	if config.ChunkSize == nil {
		value := defaultDownloadChunkSize
		config.ChunkSize = &value
	}
	if *config.ChunkSize <= 0 || *config.ChunkSize > maxDownloadChunkSize {
		return config, uploadInvalid("image download chunk size must be between 1 byte and 64 MiB")
	}
	if config.VerifyChecksum == nil {
		value := true
		config.VerifyChecksum = &value
	}
	for _, preference := range config.StorePreferences {
		if !utf8.ValidString(preference) {
			return config, uploadInvalid("image download preference must be valid UTF-8")
		}
	}
	var err error
	config.Headers, err = imageMutationHeaders(config.Headers, false, "")
	return config, errors.Join(err, check(ctx))
}
