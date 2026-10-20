package objects

import (
	"context"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/swiftinfo"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// CreateObject uploads explicit bytes directly, or prepares immutable file or
// Reader content before capability discovery, checksum comparison and optional
// SLO/DLO segmentation. Results retain distinct server-phase evidence.
func (a *API) CreateObject(ctx context.Context, container, object string, input CreateObjectInput, options ...CreateObjectOption) (result *CreateObjectResult, err error) {
	defer func() { err = request.Wrap("CreateObject", "objects", err) }()
	p, err := a.captureCreateObject(ctx, container, object)
	if err != nil {
		return nil, err
	}
	input, kind, err := snapshotCreateObjectInput(input)
	if err != nil {
		return nil, metadataContextError(ctx, err)
	}
	cfg, err := p.applyCreateOptions(ctx, options)
	if err = p.finishCreate(ctx, &cfg, kind, err); err != nil {
		return nil, err
	}
	source, err := p.ownInput(ctx, input, kind)
	if err != nil {
		return nil, err
	}
	defer func() {
		closeErr := source.close()
		guardErr := p.note(ctx)
		err = joinMetadataErrors(err, closeErr, guardErr)
	}()
	result = &CreateObjectResult{Source: kind, Size: source.size}
	if kind == "bytes" {
		result.Mode = "ordinary"
		out := p.exchange(ctx, http.MethodPut, p.metadata.target, "ordinary", &objectCreatePayload{source: source, size: source.size}, objectCreateUploadHeaders(cfg), 1, http.StatusCreated, http.StatusAccepted)
		result.Ordinary = out.phase
		return result, out.err
	}
	generate := cfg.GenerateChecksums == nil || *cfg.GenerateChecksums
	md, sh := cfg.MD5, cfg.SHA256
	if generate && (md == "" || sh == "") {
		md, sh, err = p.hashReader(ctx, source.section(0, source.size))
	}
	result.MD5, result.SHA256 = md, sh
	if err != nil {
		return result, err
	}
	// Only supplied or generated hashes become upload metadata. Hashes later
	// calculated solely for stale comparison do not modify the uploaded object.
	if md != "" {
		cfg.Metadata["x-sdk-md5"] = md
	}
	if sh != "" {
		cfg.Metadata["x-sdk-sha256"] = sh
	}
	requested := swiftinfo.DefaultSegmentSize
	if cfg.SegmentSize != nil {
		requested = *cfg.SegmentSize
	}
	result.Capabilities, err = p.capabilities(ctx, requested)
	if err != nil {
		return result, err
	}
	stale, err := p.stale(ctx, nil, md, sh, func() (string, string, error) { return p.hashReader(ctx, source.section(0, source.size)) })
	if stale != nil {
		result.Discovery = cloneObjectCreateResponse(stale.Discovery)
		result.MD5, result.SHA256 = stale.MD5, stale.SHA256
	}
	if err != nil {
		return result, err
	}
	if err = p.guard(ctx); err != nil {
		return result, objectCreateResponseError(result.Discovery, err)
	}
	if !*stale.Stale {
		result.Skipped = true
		return result, nil
	}
	size := result.Capabilities.Size
	if source.size != 0 && size <= 0 {
		return result, objectCreateResponseError(result.Capabilities.Response, metadataInvalid("nonempty object requires a positive effective segment size"))
	}
	if source.size == 0 || source.size <= size {
		result.Mode = "ordinary"
		out := p.exchange(ctx, http.MethodPut, p.metadata.target, "ordinary", &objectCreatePayload{source: source, size: source.size}, objectCreateUploadHeaders(cfg), 1, http.StatusCreated, http.StatusAccepted)
		result.Ordinary = out.phase
		return result, out.err
	}
	if cfg.UseSLO == nil || *cfg.UseSLO {
		result.Mode = "slo"
	} else {
		result.Mode = "dlo"
	}
	err = p.segmented(ctx, result, source, size, objectCreateUploadHeaders(cfg))
	return result, err
}

func (p *preparedCreateObject) capabilities(ctx context.Context, requested int64) (*ObjectCreateCapabilities, error) {
	target, err := swiftinfo.Target(p.metadata.endpoint)
	if err != nil {
		return nil, err
	}
	out := p.exchange(ctx, http.MethodGet, target, "capabilities", nil, nil, 1, http.StatusOK, http.StatusNotFound, http.StatusPreconditionFailed)
	response := out.phase.Acknowledgement
	if response == nil {
		return nil, out.err
	}
	result := &ObjectCreateCapabilities{Response: cloneObjectCreateResponse(response), RequestedSize: requested}
	if out.err != nil {
		return result, out.err
	}
	maximum, minimum, fallback := int64(0), int64(0), false
	if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusPreconditionFailed {
		maximum, fallback = swiftinfo.FallbackMaxFileSize, true
	} else {
		var sections swiftinfo.Sections
		sections, err = swiftinfo.Decode(response.Body)
		if err == nil {
			maximum, err = swiftinfo.Bound(sections.Swift, "max_file_size")
		}
		if err == nil {
			minimum, err = swiftinfo.Bound(sections.SLO, "min_segment_size")
		}
	}
	if err = joinMetadataErrors(err, p.guard(ctx)); err != nil {
		return result, objectCreateResponseError(response, err)
	}
	result.MaxFileSize, result.MinSegmentSize, result.UsedFallback = maximum, minimum, fallback
	result.Size = swiftinfo.Select(requested, maximum, minimum)
	return result, nil
}
