package image

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// Stage translates only response headers. Body is opaque evidence and cannot
// replace or clean the current/original/dirty components of the supplied Image.
func translateImageRecordHeaders(p *preparedImageRecord, seed *ImageRecord, response *rest.Response) error {
	if err := p.check(p.ctx); err != nil {
		return response.Fail(err)
	}
	next := cloneImageRecord(seed)
	next.Envelope = bytes.Clone(response.Body)
	next.Header, next.StatusCode = response.Header.Clone(), response.StatusCode
	next.Wire = nil
	if err := projectImageRecordMutationSeed(next, p.location); err != nil {
		return response.Fail(err)
	}
	next.ImportMethods = imageRecordHTTPImportMethods(response.Header)
	if err := p.check(p.ctx); err != nil {
		return response.Fail(err)
	}
	*seed = *next
	return nil
}

// Already-prepared workflows fetch through the captured source and whole raw
// seed, rather than recapturing public views or constructing a new clean Image.
func fetchPreparedImageRecord(p *preparedImageRecord, seed *ImageRecord, id string) (*ImageRecord, *rest.Response, error) {
	if err := p.check(p.ctx); err != nil {
		return nil, nil, err
	}
	response, err := rest.DoJSONGuardedRejections(p.ctx, p.client, p.check, http.MethodGet,
		imageRecordEndpoint(p, id), nil, nil,
		rest.RejectionPolicy{Codes: imageRecordTagRejectionCodes(), PreserveCleanRetry: true}, imageRecordCodes()...)
	if response == nil {
		return nil, nil, err
	}
	if err != nil {
		return imageRecordUpdateReceipt(seed, response), response, err
	}
	record, err := imageRecordFromResponse(p.ctx, p.check, seed.bodyState.current, p.location, response)
	if err != nil {
		return imageRecordUpdateReceipt(seed, response), response, err
	}
	if !json.Valid(response.Body) {
		record.bodyState = cloneImageRecordBodyState(seed.bodyState)
	}
	record.data = seed.data
	if err := p.check(p.ctx); err != nil {
		return record, response, response.Fail(err)
	}
	return record, response, nil
}
