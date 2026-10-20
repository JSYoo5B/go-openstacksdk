package image

import (
	"bytes"
	"encoding/json"
	"errors"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
)

func utf8ImageRecordPropertyJSON(value json.RawMessage) bool {
	return utf8.Valid(value) && json.Valid(value)
}

func convertImageRecordProperty(name string, value json.RawMessage) (json.RawMessage, error) {
	switch name {
	case "min_disk", "min_ram", "size", "virtual_size":
		raw, err := integerImageRecordProperty(value)
		if err != nil {
			return nil, errors.Join(uploadInvalid("image property %q integer conversion failed", name), err)
		}
		return raw, nil
	case "is_protected", "protected", "tags":
		return bytes.Clone(value), nil
	default:
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return json.RawMessage("null"), nil
		}
		text, err := cloudfilter.PythonString(value)
		if err != nil {
			return nil, errors.Join(uploadInvalid("image property %q string conversion failed", name), err)
		}
		return json.Marshal(text)
	}
}

func integerImageRecordProperty(value json.RawMessage) (json.RawMessage, error) {
	if !utf8ImageRecordPropertyJSON(value) {
		return nil, uploadInvalid("image integer property must be complete UTF-8 JSON")
	}
	trimmed := bytes.TrimSpace(value)
	switch trimmed[0] {
	case 't':
		return json.RawMessage("1"), nil
	case 'f':
		return json.RawMessage("0"), nil
	case '"':
		var text string
		if err := json.Unmarshal(trimmed, &text); err != nil {
			return nil, err
		}
		integer, err := jsonfilter.PythonIntegerString(text)
		if err != nil {
			return nil, err
		}
		return json.RawMessage(integer.String()), nil
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return jsonfilter.DescriptorIntegerJSON(trimmed)
	default:
		return nil, uploadInvalid("image integer property requires bool, number or decimal string")
	}
}
