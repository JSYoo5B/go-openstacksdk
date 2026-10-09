package image

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// Cloud get_image_id consumes all default image pages, filters deleted status,
// then returns the first exact-or-fnmatch row. Image.find is a different graph.
func resolveImageRecordPropertyID(p *preparedImageRecord, input json.RawMessage) (json.RawMessage, error) {
	parameters, err := prepareImageRecordList(p.ctx, p.check, nil)
	if err != nil {
		return nil, err
	}
	rows := make([]json.RawMessage, 0)
	identities := make([]json.RawMessage, 0)
	for record, err := range listImageRecordsPrepared(p, parameters) {
		if err != nil {
			return nil, err
		}
		// The returned record owns the actual complete list-page receipt. Keep
		// that evidence when the additional Cloud status consumer rejects it.
		receipt := &rest.Response{Body: record.Envelope, Header: record.Header, StatusCode: record.StatusCode}
		// This owned Go profile requires paired Unicode JSON strings; Source
		// .lower() alone also permits Python strings with lone surrogates.
		status, err := decodeImageRecordString(record.Resource.Body["status"], "lookup image status")
		if err = errors.Join(err, p.check(p.ctx)); err != nil {
			return nil, receipt.Fail(err)
		}
		if cloudfilter.PythonLower(status) == "deleted" {
			continue
		}
		view, err := imageRecordObject(record.Resource.Body)
		if err = errors.Join(err, p.check(p.ctx)); err != nil {
			return nil, receipt.Fail(err)
		}
		rows = append(rows, view)
		identities = append(identities, bytes.Clone(record.Resource.Body["id"]))
	}
	if err := p.check(p.ctx); err != nil {
		return nil, err
	}
	// search_images evaluates list_images completely before _filter_list
	// calls str(name_or_id); late lookup failures therefore precede conversion.
	pattern, err := cloudfilter.PythonString(input)
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return nil, err
	}
	selection, err := cloudfilter.Select(rows, pattern, nil, func() error { return p.check(p.ctx) })
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return nil, err
	}
	if len(selection.Indices) == 0 {
		return json.RawMessage("null"), nil
	}
	return bytes.Clone(identities[selection.Indices[0]]), nil
}
