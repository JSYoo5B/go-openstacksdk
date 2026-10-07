// Package cloudlimits owns the direct cloud volume limits workflow.
package cloudlimits

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudfilter"
	"github.com/JSYoo5B/gophercloudsdk/internal/jsonfilter"
)

type limitsModelConversion uint8

const (
	limitsModelLiteral limitsModelConversion = iota
	limitsModelInteger
	limitsModelAbsolute
	limitsModelRateGroups
	limitsModelRateRules
)

type limitsModelDescriptor struct {
	attribute  string
	wire       string
	conversion limitsModelConversion
}

var limitsModelTopDescriptors = [...]limitsModelDescriptor{
	{"absolute", "absolute", limitsModelAbsolute},
	{"rate", "rate", limitsModelRateGroups},
	{"id", "id", limitsModelLiteral},
	{"name", "name", limitsModelLiteral},
}

var limitsModelAbsoluteDescriptors = [...]limitsModelDescriptor{
	{"max_total_backup_gigabytes", "maxTotalBackupGigabytes", limitsModelInteger},
	{"max_total_backups", "maxTotalBackups", limitsModelInteger},
	{"max_total_snapshots", "maxTotalSnapshots", limitsModelInteger},
	{"max_total_volume_gigabytes", "maxTotalVolumeGigabytes", limitsModelInteger},
	{"max_total_volumes", "maxTotalVolumes", limitsModelInteger},
	{"total_backup_gigabytes_used", "totalBackupGigabytesUsed", limitsModelInteger},
	{"total_backups_used", "totalBackupsUsed", limitsModelInteger},
	{"total_gigabytes_used", "totalGigabytesUsed", limitsModelInteger},
	{"total_snapshots_used", "totalSnapshotsUsed", limitsModelInteger},
	{"total_volumes_used", "totalVolumesUsed", limitsModelInteger},
	{"id", "id", limitsModelLiteral},
	{"name", "name", limitsModelLiteral},
}

var limitsModelGroupDescriptors = [...]limitsModelDescriptor{
	{"limits", "limit", limitsModelRateRules},
	{"regex", "regex", limitsModelLiteral},
	{"uri", "uri", limitsModelLiteral},
	{"id", "id", limitsModelLiteral},
	{"name", "name", limitsModelLiteral},
}

var limitsModelRuleDescriptors = [...]limitsModelDescriptor{
	{"next_available", "next-available", limitsModelLiteral},
	{"remaining", "remaining", limitsModelInteger},
	{"unit", "unit", limitsModelLiteral},
	{"value", "value", limitsModelInteger},
	{"verb", "verb", limitsModelLiteral},
	{"id", "id", limitsModelLiteral},
	{"name", "name", limitsModelLiteral},
}

// normalizeLimits builds the pinned nullable Limits.to_dict view without
// changing raw response fields. Top location belongs to the owned Connection
// snapshot, while nested Resource construction has no Connection.
func normalizeLimits(raw json.RawMessage, location json.RawMessage) (json.RawMessage, error) {
	if len(location) == 0 {
		location = json.RawMessage("null")
	}
	if !utf8.Valid(location) || !json.Valid(location) {
		return nil, fmt.Errorf("limits location must be complete UTF-8 JSON")
	}
	return limitsModelObject(raw, location, limitsModelTopDescriptors[:], false)
}

// limitsModelObject consumes parsed dictionary order. ObjectMembers replaces a
// repeated JSON key at its first insertion slot, so a later alias can still win
// over the final textual occurrence of an earlier duplicate key.
func limitsModelObject(raw, location json.RawMessage, descriptors []limitsModelDescriptor, nested bool) (json.RawMessage, error) {
	members, err := cloudfilter.ObjectMembers(raw)
	if err != nil {
		return nil, err
	}
	selected := make([]json.RawMessage, len(descriptors))
	var wireLocation json.RawMessage
	recognized := false
	for _, member := range members {
		if nested && member.Key == "self" {
			return nil, fmt.Errorf("nested limits Resource cannot receive reserved self constructor argument")
		}
		if nested && member.Key == "connection" {
			truthy, err := limitsModelConnectionTruthy(member.Value)
			if err != nil {
				return nil, fmt.Errorf("nested limits connection argument: %w", err)
			}
			if truthy {
				return nil, fmt.Errorf("nested limits Resource cannot receive a truthy JSON connection argument")
			}
		}
		if member.Key == "location" {
			wireLocation = member.Value
		}
		for index, descriptor := range descriptors {
			if member.Key == descriptor.attribute || member.Key == descriptor.wire {
				selected[index] = member.Value
				recognized = true
			}
		}
	}
	if nested {
		location = json.RawMessage("null")
		if !recognized && wireLocation != nil {
			location = wireLocation
		}
	}

	converted := make([]json.RawMessage, len(descriptors))
	for index, descriptor := range descriptors {
		converted[index], err = limitsModelConvert(selected[index], descriptor.conversion)
		if err != nil {
			return nil, fmt.Errorf("limits field %q: %w", descriptor.attribute, err)
		}
	}

	var output bytes.Buffer
	output.WriteByte('{')
	for index, descriptor := range descriptors {
		if index != 0 {
			output.WriteByte(',')
		}
		key, _ := json.Marshal(descriptor.attribute)
		output.Write(key)
		output.WriteByte(':')
		output.Write(converted[index])
	}
	if len(descriptors) != 0 {
		output.WriteByte(',')
	}
	output.WriteString(`"location":`)
	output.Write(location)
	output.WriteByte('}')
	return json.RawMessage(output.Bytes()), nil
}

func limitsModelConvert(raw json.RawMessage, conversion limitsModelConversion) (json.RawMessage, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return json.RawMessage("null"), nil
	}
	switch conversion {
	case limitsModelInteger:
		return jsonfilter.DescriptorIntegerJSON(raw)
	case limitsModelAbsolute:
		return limitsModelResource(raw, limitsModelAbsoluteDescriptors[:])
	case limitsModelRateGroups:
		return limitsModelList(raw, limitsModelGroupDescriptors[:])
	case limitsModelRateRules:
		return limitsModelList(raw, limitsModelRuleDescriptors[:])
	default:
		return bytes.Clone(raw), nil
	}
}

// Resource conversion differs from a missing descriptor: a nonobject list item
// (including null) becomes plain {}, while an actual {} gains nullable fields.
func limitsModelResource(raw json.RawMessage, descriptors []limitsModelDescriptor) (json.RawMessage, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return json.RawMessage("{}"), nil
	}
	return limitsModelObject(raw, nil, descriptors, true)
}

func limitsModelList(raw json.RawMessage, descriptors []limitsModelDescriptor) (json.RawMessage, error) {
	items := []json.RawMessage{raw}
	if raw[0] == '[' {
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, err
		}
	}
	var output bytes.Buffer
	output.WriteByte('[')
	for index, item := range items {
		value, err := limitsModelResource(item, descriptors)
		if err != nil {
			return nil, fmt.Errorf("limits list item %d: %w", index, err)
		}
		if index != 0 {
			output.WriteByte(',')
		}
		output.Write(value)
	}
	output.WriteByte(']')
	return json.RawMessage(output.Bytes()), nil
}

// Nested Resource constructors consume connection before Body attributes. A
// falsey JSON value behaves like no Connection; a truthy value lacks the source
// current_location attribute. Decimal/exponent truth follows the parsed Python
// float, including underflow to zero and overflow to infinity.
func limitsModelConnectionTruthy(raw json.RawMessage) (bool, error) {
	raw = bytes.TrimSpace(raw)
	switch raw[0] {
	case 'n', 'f':
		return false, nil
	case 't':
		return true, nil
	case '"':
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return false, err
		}
		return value != "", nil
	case '[':
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return false, err
		}
		return len(items) != 0, nil
	case '{':
		members, err := cloudfilter.ObjectMembers(raw)
		return len(members) != 0, err
	default:
		if !strings.ContainsAny(string(raw), ".eE") {
			for _, digit := range raw {
				if digit >= '1' && digit <= '9' {
					return true, nil
				}
			}
			return false, nil
		}
		value, err := strconv.ParseFloat(string(raw), 64)
		if err != nil {
			if numberError, ok := err.(*strconv.NumError); !ok || numberError.Err != strconv.ErrRange {
				return false, err
			}
		}
		return value != 0, nil
	}
}
