package image

import (
	"bytes"
	"encoding/json"
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
// This is not a mutable Resource dirty/header state machine.
type imageRecordBodyState struct {
	current, original map[string]json.RawMessage
}

func newImageRecordBodyState(fields map[string]json.RawMessage) *imageRecordBodyState {
	return &imageRecordBodyState{current: copyTaskRawMap(fields), original: copyTaskRawMap(fields)}
}

func cloneImageRecordBodyState(state *imageRecordBodyState) *imageRecordBodyState {
	if state == nil {
		return nil
	}
	return &imageRecordBodyState{current: copyTaskRawMap(state.current), original: copyTaskRawMap(state.original)}
}
