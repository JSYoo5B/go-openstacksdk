package blockstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func (p *preparedGetVolumes) selectVolumes(ctx context.Context, entries []getVolumesEntry) ([]*resource.RawResource, error) {
	selected := make([]*resource.RawResource, 0)
	for _, entry := range entries {
		if err := p.guard(ctx); err != nil {
			return nil, err
		}
		rows, err := getVolumesAttachments(entry.volume.Body)
		if err != nil {
			return nil, entry.origin.Fail(fmt.Errorf("volumes[%d].attachments: %w", entry.index, err))
		}
		for index, row := range rows {
			if err := p.guard(ctx); err != nil {
				return nil, err
			}
			fields, err := attachmentObject(row)
			if err != nil {
				return nil, entry.origin.Fail(fmt.Errorf("volumes[%d].attachments[%d]: %w", entry.index, index, err))
			}
			server, present := fields["server_id"]
			if !present {
				return nil, entry.origin.Fail(attachInvalid("volumes[%d].attachments[%d] is missing canonical server_id", entry.index, index))
			}
			// Source order reads attachment.server_id before caller server.id.
			// Invalid caller evidence is local and never gains an HTTP response.
			identity, err := p.serverIdentity()
			if err != nil {
				return nil, err
			}
			matched, err := jsonfilter.EqualPythonJSON(server, identity)
			if err != nil {
				return nil, entry.origin.Fail(fmt.Errorf("volumes[%d].attachments[%d].server_id: %w", entry.index, index, err))
			}
			if err := p.guard(ctx); err != nil {
				return nil, err
			}
			if matched {
				selected = append(selected, entry.volume)
			}
		}
	}
	return selected, p.guard(ctx)
}

// The selected v3 Resource descriptor uses type=list: an array stays ordered,
// while every nonnull nonarray value becomes one row. Device is never consumed.
func getVolumesAttachments(fields map[string]json.RawMessage) ([]json.RawMessage, error) {
	raw, present := fields["attachments"]
	value := bytes.TrimSpace(raw)
	if !present || len(value) == 0 || bytes.Equal(value, []byte("null")) {
		return nil, attachInvalid("attachments is null or missing")
	}
	if value[0] != '[' {
		return []json.RawMessage{raw}, nil
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func (p *preparedGetVolumes) serverIdentity() (json.RawMessage, error) {
	if p.options.ServerFields == nil {
		return json.Marshal(p.input.ServerID)
	}
	raw, present := (*p.options.ServerFields)["id"]
	if !present {
		return nil, attachInvalid("provided server is missing canonical id")
	}
	if raw == nil {
		return json.RawMessage("null"), nil
	}
	if !utf8.Valid(raw) {
		return nil, attachInvalid("provided server id must be valid UTF-8 JSON")
	}
	var value json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%w: provided server id JSON: %w", resource.ErrInvalidOption, err)
	}
	return raw, nil
}
