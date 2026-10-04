package blockstorage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"

	"gophercloudsdk/internal/jsonfilter"
)

type volumeSearchConversion uint8

const (
	volumeSearchUnchanged volumeSearchConversion = iota
	volumeSearchList
	volumeSearchDict
	volumeSearchBoolStr
	volumeSearchBoolean
	volumeSearchInteger
)

type volumeSearchDescriptor struct {
	attribute  string
	wire       string
	conversion volumeSearchConversion
}

// These are the 34 direct v3 Volume Body descriptors followed by the
// inherited id, name and metadata descriptors. Untyped descriptors retain
// their JSON value; they do not borrow the stricter workflow VolumeInfo model.
var volumeSearchDescriptors = [...]volumeSearchDescriptor{
	{"attachments", "attachments", volumeSearchList},
	{"availability_zone", "availability_zone", volumeSearchUnchanged},
	{"backup_id", "backup_id", volumeSearchUnchanged},
	{"consistency_group_id", "consistencygroup_id", volumeSearchUnchanged},
	{"consumes_quota", "consumes_quota", volumeSearchUnchanged},
	{"cluster_name", "cluster_name", volumeSearchUnchanged},
	{"created_at", "created_at", volumeSearchUnchanged},
	{"description", "description", volumeSearchUnchanged},
	{"encryption_key_id", "encryption_key_id", volumeSearchUnchanged},
	{"extended_replication_status", "os-volume-replication:extended_status", volumeSearchUnchanged},
	{"group_id", "group_id", volumeSearchUnchanged},
	{"host", "os-vol-host-attr:host", volumeSearchUnchanged},
	{"image_id", "imageRef", volumeSearchUnchanged},
	{"is_bootable", "bootable", volumeSearchBoolStr},
	{"is_encrypted", "encrypted", volumeSearchBoolStr},
	{"is_multiattach", "multiattach", volumeSearchBoolean},
	{"migration_id", "os-vol-mig-status-attr:name_id", volumeSearchUnchanged},
	{"migration_status", "os-vol-mig-status-attr:migstat", volumeSearchUnchanged},
	{"project_id", "os-vol-tenant-attr:tenant_id", volumeSearchUnchanged},
	{"replication_driver_data", "os-volume-replication:driver_data", volumeSearchUnchanged},
	{"provider_id", "provider_id", volumeSearchUnchanged},
	{"replication_status", "replication_status", volumeSearchUnchanged},
	{"scheduler_hints", "OS-SCH-HNT:scheduler_hints", volumeSearchDict},
	{"service_uuid", "service_uuid", volumeSearchUnchanged},
	{"shared_targets", "shared_targets", volumeSearchBoolean},
	{"size", "size", volumeSearchInteger},
	{"snapshot_id", "snapshot_id", volumeSearchUnchanged},
	{"source_volume_id", "source_volid", volumeSearchUnchanged},
	{"status", "status", volumeSearchUnchanged},
	{"updated_at", "updated_at", volumeSearchUnchanged},
	{"user_id", "user_id", volumeSearchUnchanged},
	{"volume_image_metadata", "volume_image_metadata", volumeSearchUnchanged},
	{"volume_type", "volume_type", volumeSearchUnchanged},
	{"volume_type_id", "volume_type_id", volumeSearchUnchanged},
	{"id", "id", volumeSearchUnchanged},
	{"name", "name", volumeSearchUnchanged},
	{"metadata", "metadata", volumeSearchDict},
}

// volumeSearchView projects the pinned Volume.to_dict view, without changing
// the caller's raw wire row. Exact normalized/wire aliases are consumed in
// JSON member order, and the last consumed alias wins. Every known descriptor
// is converted eagerly even when a later filter does not reference it.
//
// Location is independent connection evidence supplied by the caller. The
// row's HTTP location member is deliberately ignored. Nil location denotes
// null; a connected collector supplies its owned computed location object.
func volumeSearchView(row json.RawMessage, location json.RawMessage) (json.RawMessage, error) {
	if !utf8.Valid(row) {
		return nil, fmt.Errorf("volume search row must be valid UTF-8 JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(row))
	opening, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if opening != json.Delim('{') {
		return nil, fmt.Errorf("volume search row must be a JSON object")
	}
	var selected [len(volumeSearchDescriptors)]json.RawMessage
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return nil, err
		}
		for index, descriptor := range volumeSearchDescriptors {
			if key == descriptor.attribute || key == descriptor.wire {
				selected[index] = raw
			}
		}
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("volume search row contains multiple JSON values")
		}
		return nil, err
	}

	var converted [len(volumeSearchDescriptors)]json.RawMessage
	for index, descriptor := range volumeSearchDescriptors {
		converted[index], err = volumeSearchConvert(selected[index], descriptor.conversion)
		if err != nil {
			return nil, fmt.Errorf("volume search field %q: %w", descriptor.attribute, err)
		}
	}
	if location == nil {
		location = json.RawMessage("null")
	}
	if !utf8.Valid(location) || !json.Valid(location) {
		return nil, fmt.Errorf("volume search location must be valid UTF-8 JSON")
	}

	// Resource's MRO yields the direct Body fields, id, name, location, then
	// MetadataMixin.metadata. Preserve that order in the serialized view.
	var view bytes.Buffer
	view.WriteByte('{')
	for index, descriptor := range volumeSearchDescriptors {
		if index > 0 {
			view.WriteByte(',')
		}
		if descriptor.attribute == "metadata" {
			view.WriteString(`"location":`)
			view.Write(location)
			view.WriteByte(',')
		}
		key, _ := json.Marshal(descriptor.attribute)
		view.Write(key)
		view.WriteByte(':')
		view.Write(converted[index])
	}
	view.WriteByte('}')
	return json.RawMessage(view.Bytes()), nil
}

func volumeSearchConvert(raw json.RawMessage, conversion volumeSearchConversion) (json.RawMessage, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return json.RawMessage("null"), nil
	}
	switch conversion {
	case volumeSearchList:
		if raw[0] != '[' {
			wrapped := make(json.RawMessage, 0, len(raw)+2)
			wrapped = append(wrapped, '[')
			wrapped = append(wrapped, raw...)
			return append(wrapped, ']'), nil
		}
	case volumeSearchDict:
		if raw[0] != '{' {
			return json.RawMessage("{}"), nil
		}
	case volumeSearchBoolStr:
		if bytes.Equal(raw, []byte("true")) || bytes.Equal(raw, []byte("false")) {
			return append(json.RawMessage(nil), raw...), nil
		}
		if raw[0] == '"' {
			var value string
			if err := json.Unmarshal(raw, &value); err != nil {
				return nil, err
			}
			switch strings.ToLower(value) {
			case "true":
				return json.RawMessage("true"), nil
			case "false":
				return json.RawMessage("false"), nil
			}
		}
		return nil, fmt.Errorf("BoolStr expects a boolean or a true/false string")
	case volumeSearchBoolean:
		return jsonfilter.BooleanJSON(raw)
	case volumeSearchInteger:
		return volumeSearchIntegerJSON(raw)
	}
	return append(json.RawMessage(nil), raw...), nil
}

// The source int descriptor preserves bool (a Python int subclass), truncates
// JSON floats through their IEEE-754 value, and accepts strings only when
// str.isdigit() succeeds. Unlike caller integer filters, signed/space-padded
// strings and ordinary nonnumeric containers become zero.
func volumeSearchIntegerJSON(raw json.RawMessage) (json.RawMessage, error) {
	if bytes.Equal(raw, []byte("true")) || bytes.Equal(raw, []byte("false")) {
		return append(json.RawMessage(nil), raw...), nil
	}
	if raw[0] == '"' {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		if value == "" {
			return json.RawMessage("0"), nil
		}
		var digits strings.Builder
		nondecimal := false
		for _, character := range value {
			if digit, ok := volumeSearchDecimalDigit(character); ok {
				digits.WriteByte('0' + byte(digit))
			} else if volumeSearchNondecimalDigit(character) {
				nondecimal = true
			} else {
				return json.RawMessage("0"), nil
			}
		}
		if nondecimal {
			return nil, fmt.Errorf("digit-only size string contains a nondecimal Unicode digit")
		}
		integer, ok := new(big.Int).SetString(digits.String(), 10)
		if !ok {
			return nil, fmt.Errorf("size string cannot be converted to an integer")
		}
		return json.RawMessage(integer.String()), nil
	}
	if raw[0] == '-' || (raw[0] >= '0' && raw[0] <= '9') {
		text := string(raw)
		if !strings.ContainsAny(text, ".eE") {
			integer, ok := new(big.Int).SetString(text, 10)
			if !ok {
				return nil, fmt.Errorf("size number cannot be converted to an integer")
			}
			return json.RawMessage(integer.String()), nil
		}
		floating, err := strconv.ParseFloat(text, 64)
		if err != nil || math.IsInf(floating, 0) || math.IsNaN(floating) {
			return nil, fmt.Errorf("size float cannot be converted to an integer")
		}
		integer, _ := new(big.Float).SetFloat64(floating).Int(nil)
		return json.RawMessage(integer.String()), nil
	}
	return json.RawMessage("0"), nil
}

// Python's SDK pin does not pin its Unicode runtime. These digit tables use
// Unicode 16.0, matching the audited Python reference runtime. They include
// isdigit characters that int rejects (for example superscript two), without
// depending on the Go toolchain's Unicode table version. Integer conversion
// uses arbitrary precision rather than CPython's configurable string limit.
type volumeSearchRuneRange struct{ first, last rune }

var volumeSearchDecimalRanges = [...]volumeSearchRuneRange{
	{0x30, 0x39},
	{0x660, 0x669},
	{0x6f0, 0x6f9},
	{0x7c0, 0x7c9},
	{0x966, 0x96f},
	{0x9e6, 0x9ef},
	{0xa66, 0xa6f},
	{0xae6, 0xaef},
	{0xb66, 0xb6f},
	{0xbe6, 0xbef},
	{0xc66, 0xc6f},
	{0xce6, 0xcef},
	{0xd66, 0xd6f},
	{0xde6, 0xdef},
	{0xe50, 0xe59},
	{0xed0, 0xed9},
	{0xf20, 0xf29},
	{0x1040, 0x1049},
	{0x1090, 0x1099},
	{0x17e0, 0x17e9},
	{0x1810, 0x1819},
	{0x1946, 0x194f},
	{0x19d0, 0x19d9},
	{0x1a80, 0x1a89},
	{0x1a90, 0x1a99},
	{0x1b50, 0x1b59},
	{0x1bb0, 0x1bb9},
	{0x1c40, 0x1c49},
	{0x1c50, 0x1c59},
	{0xa620, 0xa629},
	{0xa8d0, 0xa8d9},
	{0xa900, 0xa909},
	{0xa9d0, 0xa9d9},
	{0xa9f0, 0xa9f9},
	{0xaa50, 0xaa59},
	{0xabf0, 0xabf9},
	{0xff10, 0xff19},
	{0x104a0, 0x104a9},
	{0x10d30, 0x10d39},
	{0x10d40, 0x10d49},
	{0x11066, 0x1106f},
	{0x110f0, 0x110f9},
	{0x11136, 0x1113f},
	{0x111d0, 0x111d9},
	{0x112f0, 0x112f9},
	{0x11450, 0x11459},
	{0x114d0, 0x114d9},
	{0x11650, 0x11659},
	{0x116c0, 0x116c9},
	{0x116d0, 0x116e3},
	{0x11730, 0x11739},
	{0x118e0, 0x118e9},
	{0x11950, 0x11959},
	{0x11bf0, 0x11bf9},
	{0x11c50, 0x11c59},
	{0x11d50, 0x11d59},
	{0x11da0, 0x11da9},
	{0x11f50, 0x11f59},
	{0x16130, 0x16139},
	{0x16a60, 0x16a69},
	{0x16ac0, 0x16ac9},
	{0x16b50, 0x16b59},
	{0x16d70, 0x16d79},
	{0x1ccf0, 0x1ccf9},
	{0x1d7ce, 0x1d7ff},
	{0x1e140, 0x1e149},
	{0x1e2f0, 0x1e2f9},
	{0x1e4f0, 0x1e4f9},
	{0x1e5f1, 0x1e5fa},
	{0x1e950, 0x1e959},
	{0x1fbf0, 0x1fbf9},
}

var volumeSearchNondecimalRanges = [...]volumeSearchRuneRange{
	{0xb2, 0xb3},
	{0xb9, 0xb9},
	{0x1369, 0x1371},
	{0x19da, 0x19da},
	{0x2070, 0x2070},
	{0x2074, 0x2079},
	{0x2080, 0x2089},
	{0x2460, 0x2468},
	{0x2474, 0x247c},
	{0x2488, 0x2490},
	{0x24ea, 0x24ea},
	{0x24f5, 0x24fd},
	{0x24ff, 0x24ff},
	{0x2776, 0x277e},
	{0x2780, 0x2788},
	{0x278a, 0x2792},
	{0x10a40, 0x10a43},
	{0x10e60, 0x10e68},
	{0x11052, 0x1105a},
	{0x1f100, 0x1f10a},
}

func volumeSearchDecimalDigit(character rune) (int, bool) {
	for _, interval := range volumeSearchDecimalRanges {
		if character >= interval.first && character <= interval.last {
			return int(character-interval.first) % 10, true
		}
	}
	return 0, false
}

func volumeSearchNondecimalDigit(character rune) bool {
	for _, interval := range volumeSearchNondecimalRanges {
		if character >= interval.first && character <= interval.last {
			return true
		}
	}
	return false
}
