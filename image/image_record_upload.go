package image

import (
	"context"
	"io"
	"slices"
)

// ImageRecordUploadRequest supplies optional borrowed data and raw constructor
// attributes. Name is an optional server-owned attribute. Data nil sends an empty
// binary PUT. There is no Filename parameter: an attribute named filename is
// metadata; callers open and close their own files. No prior image is selected.
type ImageRecordUploadRequest struct {
	Data       io.Reader
	Attributes map[string]any
}

// ImageRecordUploadResult retains the metadata-created Record and independent
// actual phase receipts. PUT does not translate or clean the Record, change its
// import methods, or refresh metadata. Later failures retain accepted creation.
type ImageRecordUploadResult struct {
	Record             *ImageRecord
	Metadata, Uploaded *ImageUploadResponse
}

// UploadImageRecord maps the complete deprecated Proxy.upload_image flow through
// an owned raw metadata constructor, one JSON POST and one binary PUT. Required
// formal formats are truthy JSON values with no implicit defaults. Size inference
// runs after creation and preserves the caller cursor; readers remain borrowed.
// There is no queued gate, GET, task/import/wait, rollback or PUT translation.
func (s *Service) UploadImageRecord(ctx context.Context, input ImageRecordUploadRequest, options ...ImageRecordUploadOption) (*ImageRecordUploadResult, error) {
	fail := func(result *ImageRecordUploadResult, err error) (*ImageRecordUploadResult, error) {
		return result, wrapImageMutationError(ctx, "UploadImageRecord", err)
	}
	p, err := s.prepareImageRecordUpload(ctx, input, slices.Clone(options))
	if err != nil {
		return fail(nil, err)
	}
	record, metadata, err := createPreparedImageRecord(p.preparedImageRecordCreate)
	if metadata == nil {
		return fail(nil, err)
	}
	result := &ImageRecordUploadResult{Record: record, Metadata: imageUploadResponse(metadata)}
	if err != nil {
		return fail(result, err)
	}
	// Source assigns data unconditionally after successful metadata creation,
	// including None. Plain borrowed data remains outside Body and receipts.
	record.data = p.data
	if err := p.check(p.ctx); err != nil {
		return fail(result, err)
	}
	identity, err := decodeImageRecordString(record.bodyState.current["id"], "created image identity")
	if err != nil {
		return fail(result, err)
	}
	if err := validateImageRecordIdentity(identity); err != nil {
		return fail(result, err)
	}
	size := p.policy.Size
	if size == nil && !p.policy.DisableSizeInference {
		size, err = inferImageRecordStageSize(p.preparedImageRecord, p.data)
		if err != nil {
			return fail(result, err)
		}
	}
	if err := p.check(p.ctx); err != nil {
		return fail(result, err)
	}
	response, err := imageRecordDataOnce(p.preparedImageRecord, imageRecordEndpoint(p.preparedImageRecord, identity)+"/file", p.data, size)
	result.Uploaded = imageUploadResponse(response)
	// The opaque acknowledgement cannot affect any creation Record channel.
	// Native Go status errors are explicit even though Source upload discards
	// the Response under its default raise_exc=False adapter policy.
	return fail(result, err)
}
