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
