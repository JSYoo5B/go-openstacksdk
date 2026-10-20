package objects

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
)

// ImageImportObjectRequest is the concrete Swift input used by image creation.
// Data is borrowed. DataPresent distinguishes None from present empty content.
// With neither DataPresent nor Filename, Name is also the local filename.
// Headers are ordinary optional headers; the content type, TTL and SDK
// metadata namespace belong to this workflow.
type ImageImportObjectRequest struct {
	Container, Name, Filename string
	Data                      io.Reader
	DataPresent               bool
	MD5, SHA256               string
	Headers                   map[string]string
}

// ImageImportObjectResult retains file/SLO evidence or the single data PUT.
// Uploaded.Attempts also retains a rejected or interrupted physical request.
type ImageImportObjectResult struct {
	Created  *CreateObjectResult
	Uploaded *ObjectCreatePhaseResult
}

type imageImportObjectProfileKey struct{}

// CreateImageImportObject composes the existing Swift upload engines for an
// image import task. Filename uploads retain stale detection and segmentation;
// ordinary data uses one physical PUT without reading, closing, seeking or
// spooling the caller reader before that request or replaying it afterwards.
func (a *API) CreateImageImportObject(ctx context.Context, input ImageImportObjectRequest) (result *ImageImportObjectResult, err error) {
	defer func() { err = request.Wrap("CreateImageImportObject", "objects", err) }()
	if ctx == nil {
		return nil, metadataInvalid("context is required")
	}
	input.Headers = cloneMetadataHeaders(input.Headers)
	ctx = context.WithValue(ctx, imageImportObjectProfileKey{}, true)
	p, err := a.captureCreateObject(ctx, input.Container, input.Name)
	if err != nil {
		return nil, err
	}
	if input.DataPresent && input.Filename != "" {
		return nil, metadataContextError(ctx, metadataInvalid("both filename and data given"))
	}
	if !input.DataPresent && input.Data != nil {
		return nil, metadataContextError(ctx, metadataInvalid("data requires DataPresent"))
	}
	headers, err := imageImportObjectHeaders(input.Headers)
	if err = joinMetadataErrors(err, p.guard(ctx)); err != nil {
		return nil, err
	}
	if !input.DataPresent {
		filename := input.Filename
		if filename == "" {
			filename = input.Name
		}
		generate := false
		cfg := CreateObjectOpts{Headers: headers, Metadata: map[string]string{"x-sdk-autocreated": "true"}, MD5: input.MD5, SHA256: input.SHA256, GenerateChecksums: &generate}
		created, createErr := a.CreateObject(ctx, input.Container, input.Name, CreateObjectInput{Filename: filename}, WithCreateObjectOpts(cfg))
		if created != nil {
			result = &ImageImportObjectResult{Created: created}
		}
		return result, joinMetadataErrors(createErr, p.guard(ctx))
	}
	reader := input.Data
	size := int64(-1)
	if reader == nil {
		reader, size = bytes.NewReader(nil), 0
	} else {
		value := reflect.ValueOf(reader)
		switch value.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
			if value.IsNil() {
				return nil, metadataContextError(ctx, metadataInvalid("nil image import object reader"))
			}
		}
	}
	// Source's ordinary data branch ignores supplied file hashes.
	generate := false
	cfg := CreateObjectOpts{Headers: headers, Metadata: map[string]string{"x-sdk-autocreated": "true"}, GenerateChecksums: &generate}
	if err := p.finishCreate(ctx, &cfg, "reader", nil); err != nil {
		return nil, err
	}
	payload := &objectCreatePayload{source: &objectCreateSource{reader: reader, size: size}, size: size, borrowed: true}
	out := p.exchange(ctx, http.MethodPut, p.metadata.target, "ordinary", payload, objectCreateUploadHeaders(cfg), 1, http.StatusCreated, http.StatusAccepted)
	result = &ImageImportObjectResult{Uploaded: out.phase}
	return result, out.err
}

func imageImportObjectHeaders(values map[string]string) (map[string]string, error) {
	headers, err := validateCreateObjectHeaders(values)
	if err != nil {
		return nil, err
	}
	for key := range headers {
		if strings.EqualFold(key, "Content-Type") || strings.EqualFold(key, "X-Delete-After") {
			return nil, metadataInvalid("image import object header %q is SDK owned", key)
		}
	}
	headers["Content-Type"] = "application/octet-stream"
	headers["X-Delete-After"] = "86400"
	return headers, nil
}

func validateImageImportObjectSourceHeaders(values map[string]string) error {
	headers, err := validateCreateObjectHeaders(values)
	if err != nil {
		return err
	}
	for key, expected := range map[string]string{"Content-Type": "application/octet-stream", "X-Delete-After": "86400"} {
		if actual, exists := headers[key]; exists && actual != expected {
			return metadataInvalid("configured image import object header %q conflicts with SDK ownership", key)
		}
	}
	return nil
}

// The image profile follows Source's generic raise_from_response acceptance,
// preserving the existing explicit missing/capability fallback statuses.
func imageImportObjectCodes(existing []int) []int {
	codes := make([]int, 0, 200+len(existing))
	for code := 200; code < 400; code++ {
		codes = append(codes, code)
	}
	for _, code := range existing {
		if code >= 400 {
			codes = append(codes, code)
		}
	}
	return codes
}
