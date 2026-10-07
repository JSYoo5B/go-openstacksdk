package objects

import (
	"context"
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/request"
)

// CreateDirectoryMarkerObject writes an empty object with the owned
// application/directory media type and retains the upload's physical evidence.
func (a *API) CreateDirectoryMarkerObject(ctx context.Context, container, name string, options ...DirectoryMarkerOption) (result *CreateObjectResult, err error) {
	defer func() { err = request.Wrap("CreateDirectoryMarkerObject", "objects", err) }()
	p, err := a.captureCreateObject(ctx, container, name)
	if err != nil {
		return nil, err
	}
	p.headerPolicy = directoryMarkerHeaderPolicy
	if err = p.guard(ctx); err != nil {
		return nil, err
	}
	cfg, err := p.applyDirectoryMarkerOptions(ctx, options)
	if err == nil {
		cfg.Headers, err = validateDirectoryMarkerHeaders(cfg.Headers)
	}
	upload := CreateObjectOpts{Headers: cfg.Headers, Metadata: cfg.Metadata}
	if err = p.finishCreate(ctx, &upload, "bytes", err); err != nil {
		return nil, err
	}
	upload.Headers["Content-Type"] = "application/directory"
	source, err := p.ownInput(ctx, CreateObjectInput{Data: []byte{}}, "bytes")
	if err != nil {
		return nil, err
	}
	defer func() { err = joinMetadataErrors(err, source.close(), p.note(ctx)) }()
	result = &CreateObjectResult{Source: "bytes", Mode: "ordinary", Size: source.size}
	out := p.exchange(ctx, http.MethodPut, p.metadata.target, "directory-marker", &objectCreatePayload{source: source, size: source.size}, objectCreateUploadHeaders(upload), 1, http.StatusCreated, http.StatusAccepted)
	result.Ordinary = out.phase
	return result, out.err
}
