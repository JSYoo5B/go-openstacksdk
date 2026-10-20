package image

import (
	"context"
	"errors"
	"io"
	"os"
	"slices"
	"syscall"
)

// ImageRecordStageRequest selects a literal ID or SDK-produced record and an
// optional binary source. No initial GET or name lookup is performed. A queued
// record is required; an ID-only seed therefore fails before data IO or HTTP.
// Data borrows the reader at its current cursor. Filename is opened by the SDK.
// If both are omitted, the record's private borrowed data is reused, or an empty
// stage PUT is sent when that plain data reference is nil.
type ImageRecordStageRequest struct {
	ID       string
	Record   *ImageRecord
	Data     io.Reader
	Filename string
}

// ImageRecordStageResult retains the stage ACK and required metadata fetch
// independently. Record is the prepared staged record if the final GET fails,
// or its translated/partial final record. Opaque ACK bytes never clean Body.
type ImageRecordStageResult struct {
	Record           *ImageRecord
	Staged, Metadata *ImageUploadResponse
}

// StageImageRecord stages once, translates stage headers without parsing Body,
// then fetches complete metadata using the same captured source/private ID.
// Seekable length inference restores the current cursor; borrowed data is never
// closed. An SDK-owned filename stays open through the final GET and closes
// exactly once on return. It is not retained as a closed record data reference.
func (s *Service) StageImageRecord(ctx context.Context, input ImageRecordStageRequest, options ...ImageRecordStageOption) (result *ImageRecordStageResult, err error) {
	defer func() { err = wrapImageMutationError(ctx, "StageImageRecord", err) }()
	p, err := s.prepareImageRecordStage(ctx, input, slices.Clone(options))
	if err != nil {
		return nil, err
	}
	if p.filename != "" {
		if err := p.check(p.ctx); err != nil {
			return nil, err
		}
		file, openErr := os.Open(p.filename)
		if file != nil {
			defer func() {
				// Preserve borrowed data across records, but never expose a closed
				// SDK-owned file as the next invocation's private fallback.
				p.seed.data = nil
				if result != nil && result.Record != nil {
					result.Record.data = nil
				}
				err = errors.Join(err, file.Close(), p.check(p.ctx))
			}()
		}
		if err := errors.Join(openErr, p.check(p.ctx)); err != nil {
			return nil, err
		}
		// os.Open can open directories on Unix, whereas Python open(..., rb)
		// rejects them. Keep this filename error before staging or reader IO.
		info, statErr := file.Stat()
		if statErr == nil && info.IsDir() {
			statErr = &os.PathError{Op: "open", Path: p.filename, Err: syscall.EISDIR}
		}
		if err := errors.Join(statErr, p.check(p.ctx)); err != nil {
			return nil, err
		}
		p.data, p.seed.data = file, file
	}
	size := p.policy.Size
	if size == nil && !p.policy.DisableSizeInference {
		size, err = inferImageRecordStageSize(p.preparedImageRecord, p.data)
		if err != nil {
			return nil, err
		}
	}
	if err := p.check(p.ctx); err != nil {
		return nil, err
	}
	response, err := imageRecordDataOnce(p.preparedImageRecord,
		imageRecordEndpoint(p.preparedImageRecord, p.identity)+"/stage", p.data, size)
	if response == nil {
		return nil, err
	}
	result = &ImageRecordStageResult{Staged: imageUploadResponse(response)}
	if err != nil {
		return result, err
	}
	if err := translateImageRecordHeaders(p.preparedImageRecord, p.seed, response); err != nil {
		return result, err
	}
	result.Record = p.seed
	record, metadata, err := fetchPreparedImageRecord(p.preparedImageRecord, p.seed, p.identity)
	result.Metadata = imageUploadResponse(metadata)
	if record != nil {
		result.Record = record
	}
	return result, err
}
