package image

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
)

// Every declared, passive Wire and receipt channel is independently owned.
func cloneImageRecord(value *ImageRecord) *ImageRecord {
	if value == nil {
		return nil
	}
	return &ImageRecord{
		Resource: value.Resource.Clone(), Wire: value.Wire.Clone(),
		Envelope: bytes.Clone(value.Envelope), Header: value.Header.Clone(),
		StatusCode: value.StatusCode, ImportMethods: slices.Clone(value.ImportMethods),
		bodyState: cloneImageRecordBodyState(value.bodyState),
	}
}

// Computed location and receipt metadata never become a fetched Body seed.
func imageRecordBodySnapshot(value *ImageRecord) map[string]json.RawMessage {
	body := make(map[string]json.RawMessage, len(imageRecordFields))
	for _, field := range imageRecordFields {
		if raw, present := value.Resource.Body[field.canonical]; present {
			body[field.canonical] = bytes.Clone(raw)
		}
	}
	return body
}

// The immutable record retains raw current/original components separately from
// its declared view. Absent fields remain absent and getter conversions/defaults
// never synthesize values for future dirty comparison or JSON Patch generation.
// A sticky pending set survives immutable update results and tolerated invalid
// JSON. This does not expose a mutable Resource or a header/URI state machine.
type imageRecordBodyState struct {
	current, original map[string]json.RawMessage
	dirty             map[string]struct{}
}

func newImageRecordBodyState(fields map[string]json.RawMessage) *imageRecordBodyState {
	return &imageRecordBodyState{current: copyTaskRawMap(fields), original: copyTaskRawMap(fields), dirty: make(map[string]struct{})}
}

func cloneImageRecordBodyState(state *imageRecordBodyState) *imageRecordBodyState {
	if state == nil {
		return nil
	}
	return &imageRecordBodyState{current: copyTaskRawMap(state.current), original: copyTaskRawMap(state.original), dirty: maps.Clone(state.dirty)}
}

// A constructor with a literal ID starts unsynchronized; a tolerated invalid
// GET does not establish a fetched baseline for its supplied raw attributes.
func pendingImageRecordBodyState(fields map[string]json.RawMessage) *imageRecordBodyState {
	state := newImageRecordBodyState(fields)
	state.original = make(map[string]json.RawMessage)
	if id, present := fields["id"]; present {
		state.original["id"] = bytes.Clone(id)
	}
	for key := range fields {
		if key != "id" {
			state.dirty[key] = struct{}{}
		}
	}
	return state
}
