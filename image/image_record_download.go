package image

import (
	"bytes"
	"context"
	"errors"
	"hash"
	"io"
	"net/url"
	"os"
	"slices"
	"strings"
)

func (s *Service) DownloadImageRecord(ctx context.Context, input ImageRecordDownloadRequest, options ...ImageRecordDownloadOption) (result *ImageRecordDownloadResult, err error) {
	defer func() { err = wrapImageMutationError(ctx, "DownloadImageRecord", err) }()
	p, err := s.prepareImageRecordDownload(ctx, input, slices.Clone(options))
	if err != nil {
		return nil, err
	}
	record, metadata, err := fetchPreparedImageRecord(p.preparedImageRecord, p.seed, p.identity)
	if metadata == nil {
		return nil, err
	}
	result = &ImageRecordDownloadResult{Record: record, Metadata: imageUploadResponse(metadata)}
	if err != nil {
		return result, err
	}
	identity, err := decodeImageRecordString(record.bodyState.current["id"], "fetched image download identity")
	if err == nil {
		err = validateImageRecordIdentity(identity)
	}
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return result, err
	}
	endpoint := imageRecordEndpoint(p.preparedImageRecord, identity) + "/file"
	if len(p.policy.StorePreferences) != 0 {
		endpoint += "?" + url.Values{"prefer": {strings.Join(p.policy.StorePreferences, ",")}}.Encode()
	}
	wire, err := openImageRecordDownload(p.preparedImageRecord, endpoint)
	if wire == nil {
		return result, err
	}
	response := &ImageRecordDownloadResponse{Header: wire.Header.Clone(), StatusCode: wire.StatusCode}
	result.Downloaded = response
	handedOff := false
	defer func() {
		if !handedOff {
			err = errors.Join(err, wire.Body.Close(), p.check(p.ctx))
		}
	}()
	if err != nil {
		return result, err
	}
	digest, selected, err := prepareImageRecordDownloadChecksum(record, wire.Header, p.policy)
	result.Checksum = selected
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return result, err
	}
	if p.input.Output == nil && p.input.Filename == "" && p.policy.Stream {
		if err := imageRecordDownloadCompatibility(selected, response); err != nil {
			return result, err
		}
		if err := p.check(p.ctx); err != nil {
			return result, err
		}
		response.Stream = &imageRecordDownloadStream{body: wire.Body, ctx: p.ctx, check: p.check}
		handedOff = true
		return result, nil
	}
	var buffer bytes.Buffer
	output := p.input.Output
	if p.input.Filename != "" {
		file, openErr := os.Create(p.input.Filename)
		if file != nil {
			defer func() { err = errors.Join(err, file.Close(), p.check(p.ctx)) }()
		}
		if err := errors.Join(openErr, p.check(p.ctx)); err != nil {
			return result, err
		}
		output = file
	}
	buffered := output == nil
	if buffered {
		output = &buffer
	}
	// The write-only view keeps caller Seek/Close/Flush capabilities private.
	var copiedHash *imageRecordDownloadDigest
	var copyHash hash.Hash
	if digest != nil {
		copiedHash = &imageRecordDownloadDigest{Hash: digest, ctx: p.ctx, check: p.check}
		copyHash = copiedHash
	}
	checkCopy := func(ctx context.Context) error {
		var hashErr error
		if copiedHash != nil {
			hashErr = copiedHash.err
		}
		return errors.Join(p.check(ctx), hashErr)
	}
	guarded := imageRecordDownloadWriter{Writer: output, ctx: p.ctx, check: checkCopy}
	reader := &imageRecordDownloadStream{body: wire.Body, ctx: p.ctx, check: checkCopy}
	var complete bool
	var copyErr error
	result.BytesWritten, complete, copyErr = copyDownload(p.ctx, guarded, reader, *p.policy.ChunkSize, copyHash)
	boundaryErr := checkCopy(p.ctx)
	copyErr = errors.Join(copyErr, boundaryErr)
	if boundaryErr != nil {
		complete = false
	}
	if buffered {
		response.Body = bytes.Clone(buffer.Bytes())
	}
	if complete && selected != nil {
		copyErr = errors.Join(copyErr, finishImageRecordDownloadChecksum(digest, selected))
		if guardErr := p.check(p.ctx); guardErr != nil {
			selected.Complete, selected.Verified, selected.Actual = false, false, ""
			copyErr = errors.Join(copyErr, guardErr)
		}
	}
	return result, errors.Join(copyErr, p.check(p.ctx))
}

type imageRecordDownloadWriter struct {
	io.Writer
	ctx   context.Context
	check func(context.Context) error
}

func (w imageRecordDownloadWriter) Write(value []byte) (int, error) {
	if err := w.check(w.ctx); err != nil {
		return 0, err
	}
	n, err := w.Writer.Write(value)
	return n, errors.Join(err, w.check(w.ctx))
}

// The shared copier normally consumes hashes that obey hash.Hash's complete,
// error-free Write contract. A custom factory still cannot hide a partial hash
// Write or change the captured source from its Write/Sum callbacks.
type imageRecordDownloadDigest struct {
	hash.Hash
	ctx   context.Context
	check func(context.Context) error
	err   error
}

func (h *imageRecordDownloadDigest) Write(value []byte) (int, error) {
	if h.err != nil {
		return 0, h.err
	}
	if err := h.check(h.ctx); err != nil {
		h.err = err
		return 0, err
	}
	n, err := h.Hash.Write(value)
	if n < 0 || n > len(value) {
		n, err = 0, errors.Join(err, uploadInvalid("invalid image hash Write count"))
	}
	if n != len(value) && err == nil {
		err = io.ErrShortWrite
	}
	h.err = errors.Join(err, h.check(h.ctx))
	return n, h.err
}
