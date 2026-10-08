package serviceinfo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// GetImportInfoRecord reads /info/import without a resource ID or query. It
// follows pinned Import descriptors: sub-400 status is accepted, JSON syntax
// failures retain bare defaults, and successfully parsed nonobjects fail.
func (a *API) GetImportInfoRecord(ctx context.Context, options ...ImportRecordOption) (*ImportRecord, error) {
	fail := func(err error) (*ImportRecord, error) { return nil, wrapInfoError(ctx, "GetImportInfoRecord", err) }
	owned := slices.Clone(options)
	p, err := a.captureRecord(ctx)
	if err != nil {
		return fail(err)
	}
	headers, err := prepareImportRecord(p.ctx, p.check, owned)
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return fail(err)
	}
	p.addHeaders(headers)
	response, err := rest.DoJSONGuarded(p.ctx, p.client, p.check, http.MethodGet, p.base+"info/import", nil, nil, discoveryRecordCodes()...)
	if err != nil {
		return fail(err)
	}
	record, err := importRecordFromResponse(p, response)
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return fail(response.Fail(err))
	}
	return record, nil
}
func importRecordFromResponse(p *preparedRecordSource, response *rest.Response) (*ImportRecord, error) {
	if !utf8.Valid(response.Body) {
		return nil, infoInvalid("import record response must be UTF-8")
	}
	record := &ImportRecord{Envelope: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	fields := make(map[string]json.RawMessage)
	if json.Valid(response.Body) {
		wire := &resource.RawResource{Metadata: resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode}}
		if err := json.Unmarshal(response.Body, wire); err != nil {
			return nil, err
		}
		var err error
		fields, err = normalizeDiscoveryRecord(importRecordFields[:], response.Body)
		if err != nil {
			return nil, err
		}
		record.Wire = wire
	}
	// Fetch overlays only Body attributes; a supplied computed response
	// location cannot replace the Connection snapshot even for an empty Body.
	var err error
	record.Resource, err = projectDiscoveryRecord(importRecordFields[:], fields, p.location, resource.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode})
	return record, err
}
