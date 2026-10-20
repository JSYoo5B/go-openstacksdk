package image

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
)

// Cloud get_image_id consumes the shared list_images inventory, then returns
// the first exact-or-fnmatch row. Image.find is a different graph.
func resolveImageRecordPropertyID(p *preparedImageRecord, input json.RawMessage) (json.RawMessage, error) {
	rows, err := cloudImageRecordRows(p, true, false)
	if err != nil {
		return nil, err
	}
	// search_images evaluates list_images completely before _filter_list
	// calls str(name_or_id); late lookup failures therefore precede conversion.
	pattern, err := cloudfilter.PythonString(input)
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return nil, err
	}
	selection, err := cloudfilter.Select(rows.views, pattern, nil, func() error { return p.check(p.ctx) })
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return nil, err
	}
	if len(selection.Indices) == 0 {
		return json.RawMessage("null"), nil
	}
	return bytes.Clone(rows.kept[selection.Indices[0]].Resource.Body["id"]), nil
}
