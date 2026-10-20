package image

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// ImageRecordImportRequest selects a literal image ID or an SDK-produced record.
// The private record supplies formats and pending state; no initial GET, name
// resolution, store discovery or import-method discovery is performed.
type ImageRecordImportRequest struct {
	ID     string
	Record *ImageRecord
}

// ImageRecordImportResult separates the prepared local record from the actual
// asynchronous acknowledgement. A successful submission does not clean Body,
// change image status, consume response import-method headers or finish storage.
// An accepted response handling failure retains only Acknowledgement and error.
type ImageRecordImportResult struct {
	Record          *ImageRecord
	Acknowledgement *imageimport.ImportResult
}

// ImportImageRecord submits the complete owned Proxy.import_image input flow.
// Container and disk formats use Python truthiness in the JSON domain. Source
// method, URI, complete remote tuple and store controls are compiled by the SDK;
// enabled methods, permissions and asynchronous backend work remain server-owned.
func (s *Service) ImportImageRecord(ctx context.Context, input ImageRecordImportRequest, options ...ImageRecordImportOption) (*ImageRecordImportResult, error) {
	fail := func(result *ImageRecordImportResult, err error) (*ImageRecordImportResult, error) {
		return result, wrapImageMutationError(ctx, "ImportImageRecord", err)
	}
	p, err := s.prepareImageRecordImport(ctx, input, slices.Clone(options))
	if err != nil {
		return fail(nil, err)
	}
	// Python Proxy.request defaults raise_exc=False. This Go profile keeps the
	// SDK's explicit native 400..599 rejection policy and accepts 200..399.
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(p.ctx, p.client, p.check, http.MethodPost,
		imageRecordEndpoint(p.preparedImageRecord, p.identity)+"/import", p.body, p.headerPolicy,
		rest.RejectionPolicy{Codes: imageRecordTagRejectionCodes(), PreserveCleanRetry: true}, imageRecordCodes()...)
	if guardErr := p.check(p.ctx); guardErr != nil {
		if response != nil {
			guardErr = response.Fail(guardErr)
		}
		err = errors.Join(err, guardErr)
	}
	if response == nil {
		return fail(nil, err)
	}
	result := &ImageRecordImportResult{Acknowledgement: &imageimport.ImportResult{
		ImageID: p.identity, Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode,
	}}
	if err != nil {
		return fail(result, err)
	}
	result.Record = p.seed
	return result, nil
}
