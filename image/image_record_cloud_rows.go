package image

import (
	"encoding/json"
	"errors"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// cloudImageRecordInventory is Cloud list_images: every default image page,
// with member_status=all for show_all, then Python's status.lower() filter.
// Inventory keeps every consumed row; kept rows and their views are aligned.
type cloudImageRecordInventory struct {
	inventory, kept []*ImageRecord
	views           []json.RawMessage
}

func cloudImageRecordRows(p *preparedImageRecord, filterDeleted, showAll bool) (cloudImageRecordInventory, error) {
	result := cloudImageRecordInventory{inventory: []*ImageRecord{}, kept: []*ImageRecord{}, views: []json.RawMessage{}}
	var listOptions []ImageRecordListOption
	if showAll {
		// Source show_all overrides filter_deleted before the v2 query is chosen.
		filterDeleted = false
		listOptions = append(listOptions, WithImageRecordListFilter("member_status", "all"))
	}
	parameters, err := prepareImageRecordList(p.ctx, p.check, listOptions)
	if err != nil {
		return result, err
	}
	for record, err := range listImageRecordsPrepared(p, parameters) {
		if err != nil {
			return result, err
		}
		result.inventory = append(result.inventory, record)
		// The returned record owns the actual complete list-page receipt. Keep
		// that evidence when the additional Cloud status consumer rejects it.
		receipt := &rest.Response{Body: record.Envelope, Header: record.Header, StatusCode: record.StatusCode}
		if filterDeleted {
			// This owned Go profile requires paired Unicode JSON strings; Source
			// .lower() alone also permits Python strings with lone surrogates.
			status, err := decodeImageRecordString(record.Resource.Body["status"], "Cloud image status")
			if err = errors.Join(err, p.check(p.ctx)); err != nil {
				return result, receipt.Fail(err)
			}
			if cloudfilter.PythonLower(status) == "deleted" {
				continue
			}
		}
		view, err := imageRecordObject(record.Resource.Body)
		if err = errors.Join(err, p.check(p.ctx)); err != nil {
			return result, receipt.Fail(err)
		}
		result.kept = append(result.kept, record)
		result.views = append(result.views, view)
	}
	return result, p.check(p.ctx)
}
