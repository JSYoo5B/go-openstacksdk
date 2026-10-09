package image

import (
	"context"
	"errors"
	"io"
	"slices"
	"unicode/utf8"
)

// ImageRecordCloudDownloadRequest names the Cloud download_image image and
// its required destination. Exactly one of Output or Filename is required.
type ImageRecordCloudDownloadRequest struct {
	NameOrID string
	Output   io.Writer
	Filename string
}

// ImageRecordCloudDownloadOpts carries the Cloud chunk_size and stream
// arguments. Nil ChunkSize means 1 MiB. Stream is forwarded, but the required
// destination takes precedence over stream-only mode. Headers are a Go
// extension applied to the find and download phases.
type ImageRecordCloudDownloadOpts struct {
	Headers   map[string]string
	ChunkSize *int
	Stream    bool
}
type ImageRecordCloudDownloadOption func(*ImageRecordCloudDownloadOpts) error

func copyImageRecordCloudDownload(value ImageRecordCloudDownloadOpts) ImageRecordCloudDownloadOpts {
	value.Headers = copyImageRecordHeaders(value.Headers)
	value.ChunkSize = copyCreateImportPointer(value.ChunkSize)
	return value
}
func WithImageRecordCloudDownloadOpts(value ImageRecordCloudDownloadOpts) ImageRecordCloudDownloadOption {
	owned := copyImageRecordCloudDownload(value)
	return func(config *ImageRecordCloudDownloadOpts) error {
		*config = copyImageRecordCloudDownload(owned)
		return nil
	}
}
func WithImageRecordCloudDownloadHeader(key, value string) ImageRecordCloudDownloadOption {
	return func(config *ImageRecordCloudDownloadOpts) error {
		return mergeImageRecordHeaders(&config.Headers, map[string]string{key: value})
	}
}
func WithImageRecordCloudDownloadHeaders(values map[string]string) ImageRecordCloudDownloadOption {
	owned := copyImageRecordHeaders(values)
	return func(config *ImageRecordCloudDownloadOpts) error {
		return mergeImageRecordHeaders(&config.Headers, owned)
	}
}
func WithImageRecordCloudDownloadChunkSize(value int) ImageRecordCloudDownloadOption {
	return func(config *ImageRecordCloudDownloadOpts) error {
		owned := value
		config.ChunkSize = &owned
		return nil
	}
}
func WithImageRecordCloudDownloadStream(value bool) ImageRecordCloudDownloadOption {
	return func(config *ImageRecordCloudDownloadOpts) error { config.Stream = value; return nil }
}

func prepareImageRecordCloudDownload(ctx context.Context, check func(context.Context) error, options []ImageRecordCloudDownloadOption) (ImageRecordCloudDownloadOpts, error) {
	value := copyImageRecordCloudDownload(ImageRecordCloudDownloadOpts{})
	for _, apply := range options {
		if err := check(ctx); err != nil {
			return value, err
		}
		if apply == nil {
			return value, uploadInvalid("nil image cloud download option")
		}
		owned := copyImageRecordCloudDownload(value)
		if err := errors.Join(apply(&owned), check(ctx)); err != nil {
			return value, err
		}
		value = copyImageRecordCloudDownload(owned)
	}
	headers, err := imageMutationHeaders(value.Headers, false, "")
	value.Headers = headers
	return value, errors.Join(err, check(ctx))
}

// ImageRecordCloudDownloadResult keeps the strict find result separately from
// the lower download phases, including partial download evidence.
type ImageRecordCloudDownloadResult struct {
	Found    *ImageRecord
	Download *ImageRecordDownloadResult
}

// DownloadCloudImageRecord implements Cloud download_image. It requires one
// destination before HTTP, finds the image with ignore_missing false and then
// runs the Proxy download_image graph on the found record: mandatory metadata
// fetch, binary GET and default checksum verification.
func (s *Service) DownloadCloudImageRecord(ctx context.Context, input ImageRecordCloudDownloadRequest, options ...ImageRecordCloudDownloadOption) (*ImageRecordCloudDownloadResult, error) {
	const operation = "DownloadCloudImageRecord"
	fail := func(result *ImageRecordCloudDownloadResult, err error) (*ImageRecordCloudDownloadResult, error) {
		return result, wrapImageMutationError(ctx, operation, err)
	}
	output := input.Output != nil
	if output && downloadNilWriter(input.Output) {
		return fail(nil, uploadInvalid("image download writer must not be typed nil"))
	}
	switch {
	case !output && input.Filename == "":
		return fail(nil, uploadInvalid("no output specified: an output path or writer is necessary to write the image data to"))
	case output && input.Filename != "":
		return fail(nil, uploadInvalid("both an output path and writer were provided, however only one can be used at once"))
	}
	if !utf8.ValidString(input.Filename) {
		return fail(nil, uploadInvalid("image download filename must be valid UTF-8"))
	}
	if err := validateImageRecordIdentity(input.NameOrID); err != nil {
		return fail(nil, err)
	}
	owned := slices.Clone(options)
	var policy ImageRecordCloudDownloadOpts
	p, err := s.prepareImageRecord(ctx, func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		var err error
		policy, err = prepareImageRecordCloudDownload(opctx, check, owned)
		return policy.Headers, err
	})
	if err != nil {
		return fail(nil, err)
	}
	parameters, err := prepareImageRecordList(p.ctx, p.check, nil)
	if err != nil {
		return fail(nil, err)
	}
	strict := false
	found, err := findPreparedImageRecord(p, input.NameOrID, FindImageRecordOpts{IgnoreMissing: &strict}, parameters)
	if err != nil {
		return fail(nil, err)
	}
	result := &ImageRecordCloudDownloadResult{Found: found}
	if err := p.check(p.ctx); err != nil {
		return fail(result, err)
	}
	download := ImageRecordDownloadOpts{Stream: policy.Stream, ChunkSize: policy.ChunkSize, Headers: policy.Headers}
	result.Download, err = s.DownloadImageRecord(ctx, ImageRecordDownloadRequest{Record: found, Output: input.Output, Filename: input.Filename}, WithImageRecordDownloadOpts(download))
	return result, renameImageOperation(err, operation)
}
