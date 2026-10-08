package objects

import (
	"context"
	"io"
	"net/http"
	"reflect"

	"github.com/JSYoo5B/go-openstacksdk/request"
)

// GetObject reads opaque binary bytes. Partial bytes and the initial metadata
// observation survive read or cleanup failures; no checksum is inferred.
func (a *API) GetObject(ctx context.Context, container, object string, options ...ObjectReadOption) (*GetObjectResult, error) {
	p, err := a.prepareObjectRead(ctx, container, object, options)
	if err != nil {
		return nil, request.Wrap("GetObject", "objects", err)
	}
	body, err := p.open(ctx)
	if body == nil {
		return nil, request.Wrap("GetObject", "objects", err)
	}
	result := &GetObjectResult{Metadata: body.metadata, Header: body.header.Clone(), StatusCode: body.status, NotModified: body.status == http.StatusNotModified, Complete: body.complete}
	if err != nil || body.metadata == nil {
		return result, request.Wrap("GetObject", "objects", err)
	}
	result.Body = make([]byte, 0)
	buffer := make([]byte, p.buffer)
	for !body.done {
		n, _ := body.read(buffer)
		result.Body = append(result.Body, buffer[:n]...)
	}
	result.Complete = body.complete
	err = body.fail(joinMetadataErrors(body.terminalCause, p.check(ctx)), result.Body)
	return result, request.Wrap("GetObject", "objects", err)
}

// DownloadObject copies opaque bytes to output without taking ownership of it.
// A 304 response writes nothing. BytesWritten counts only successful writes.
func (a *API) DownloadObject(ctx context.Context, container, object string, output io.Writer, options ...ObjectReadOption) (*DownloadObjectResult, error) {
	// Capture source/name checks before caller callbacks, including nil output.
	prepared, err := a.captureObjectRead(ctx, container, object)
	if err != nil {
		return nil, request.Wrap("DownloadObject", "objects", err)
	}
	if objectReadNilWriter(output) {
		return nil, request.Wrap("DownloadObject", "objects", metadataContextError(ctx, metadataInvalid("object output writer is required")))
	}
	if err := prepared.finish(ctx, options); err != nil {
		return nil, request.Wrap("DownloadObject", "objects", err)
	}
	body, err := prepared.open(ctx)
	if body == nil {
		return nil, request.Wrap("DownloadObject", "objects", err)
	}
	result := &DownloadObjectResult{Metadata: body.metadata, Header: body.header.Clone(), StatusCode: body.status, NotModified: body.status == http.StatusNotModified, Complete: body.complete}
	if err != nil || body.metadata == nil || result.NotModified {
		return result, request.Wrap("DownloadObject", "objects", err)
	}
	var written int64
	buffer := make([]byte, prepared.buffer)
	for {
		n, readErr := body.read(buffer)
		if n > 0 {
			if checkErr := prepared.check(ctx); checkErr != nil {
				err = joinMetadataErrors(readErr, checkErr)
				break
			}
			count, writeErr := output.Write(buffer[:n])
			if count < 0 || count > n {
				writeErr = joinMetadataErrors(metadataInvalid("invalid object output Write count %d", count), writeErr)
			} else {
				written += int64(count)
				if count != n && writeErr == nil {
					writeErr = io.ErrShortWrite
				}
			}
			checkErr := prepared.check(ctx)
			if writeErr != nil || checkErr != nil {
				err = joinMetadataErrors(readErr, writeErr, checkErr)
				break
			}
		}
		if body.done {
			result.Complete = body.complete
			err = readErr
			break
		}
	}
	result.BytesWritten = written
	result.Complete = body.complete && written == body.count
	err = body.fail(joinMetadataErrors(err, body.closeWire(), prepared.check(ctx)), nil)
	return result, request.Wrap("DownloadObject", "objects", err)
}

// StreamObject opens an eager GET and transfers a closeable body to the caller.
// Read and Close expose accepted response faults with immutable wire evidence.
func (a *API) StreamObject(ctx context.Context, container, object string, options ...ObjectReadOption) (*StreamObjectResult, error) {
	p, err := a.prepareObjectRead(ctx, container, object, options)
	if err != nil {
		return nil, request.Wrap("StreamObject", "objects", err)
	}
	body, err := p.open(ctx)
	if body == nil {
		return nil, request.Wrap("StreamObject", "objects", err)
	}
	result := &StreamObjectResult{Metadata: body.metadata, Header: body.header.Clone(), StatusCode: body.status, NotModified: body.status == http.StatusNotModified, Complete: body.complete}
	if body.metadata != nil {
		body.result = result
		result.Body = body
	}
	return result, request.Wrap("StreamObject", "objects", err)
}

func objectReadNilWriter(output io.Writer) bool {
	if output == nil {
		return true
	}
	value := reflect.ValueOf(output)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	}
	return false
}
