package blockstorage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	volumesv2 "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v2/volumes"
	volumesv3 "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// VolumeAttachDeviceInput accepts existing Cinder representations without a
// caller adapter. It is a closed constraint for GetVolumeAttachDevice.
type VolumeAttachDeviceInput interface {
	*AttachVolumeObservation | *VolumeInfo | *volumesv2.Volume | *volumesv3.Volume
}

// GetVolumeAttachDevice returns a copied device from the first attachment whose
// current ServerID exactly equals serverID. It performs no lookup or HTTP.
// A matched empty device is a nonnil pointer; a matched null device and no match
// both return nil. Missing consumed fields and nil attachments are local errors.
// Owned nullable fields are authoritative; Body only distinguishes key omission.
// For manually constructed records with Body == nil, nil pointers mean null.
// Native models cannot distinguish omitted/null/empty strings: their current
// strings are used literally, including an empty ServerID or Device.
func GetVolumeAttachDevice[V VolumeAttachDeviceInput](volume V, serverID string) (*string, error) {
	const operation = "GetVolumeAttachDevice"
	switch value := any(volume).(type) {
	case *AttachVolumeObservation:
		if value != nil {
			return ownedVolumeAttachDevice(value.Attachments, serverID)
		}
	case *VolumeInfo:
		if value != nil {
			return ownedVolumeAttachDevice(value.Attachments, serverID)
		}
	case *volumesv2.Volume:
		if value != nil {
			if value.Attachments == nil {
				return nil, volumeAttachDeviceError(operation, "attachments is null or missing", nil)
			}
			for _, row := range value.Attachments {
				if row.ServerID == serverID {
					device := row.Device
					return &device, nil
				}
			}
			return nil, nil
		}
	case *volumesv3.Volume:
		if value != nil {
			if value.Attachments == nil {
				return nil, volumeAttachDeviceError(operation, "attachments is null or missing", nil)
			}
			for _, row := range value.Attachments {
				if row.ServerID == serverID {
					device := row.Device
					return &device, nil
				}
			}
			return nil, nil
		}
	}
	return nil, volumeAttachDeviceError(operation, "volume is nil", nil)
}

func ownedVolumeAttachDevice(rows []*AttachVolumeRecord, serverID string) (*string, error) {
	const operation = "GetVolumeAttachDevice"
	if rows == nil {
		return nil, volumeAttachDeviceError(operation, "attachments is null or missing", nil)
	}
	for index, row := range rows {
		if row == nil {
			return nil, volumeAttachDeviceError(operation, fmt.Sprintf("attachments[%d] must be an object", index), nil)
		}
		if row.ServerID == nil {
			if row.Body != nil {
				if _, present := row.Body["server_id"]; !present {
					return nil, volumeAttachDeviceError(operation, fmt.Sprintf("attachments[%d] is missing server_id", index), nil)
				}
			}
			continue
		}
		if *row.ServerID != serverID {
			continue
		}
		if row.Device == nil {
			if row.Body != nil {
				if _, present := row.Body["device"]; !present {
					return nil, volumeAttachDeviceError(operation, fmt.Sprintf("attachments[%d] is missing device", index), nil)
				}
			}
			return nil, nil
		}
		device := *row.Device
		return &device, nil
	}
	return nil, nil
}

// GetVolumeAttachDeviceFields reads canonical attachments from supplied raw
// fields, without decoding the whole volume model. Only visited server_id and
// the first matched device are consumed. A returned nonnull JSON value is copied
// unchanged, including nonstrings: Python's typing.cast performs no conversion.
// Empty arrays, objects and strings are empty iterables; null attachments fail.
// All attachments JSON must be valid UTF-8 and syntactically valid, even when a
// later row's schema would never be visited. Unrelated volume fields are ignored.
func GetVolumeAttachDeviceFields(volume map[string]json.RawMessage, serverID string) (json.RawMessage, error) {
	const operation = "GetVolumeAttachDeviceFields"
	raw, present := volume["attachments"]
	if !present {
		return nil, volumeAttachDeviceError(operation, "volume is missing attachments", nil)
	}
	if raw == nil {
		return nil, volumeAttachDeviceError(operation, "attachments is null", nil)
	}
	if !utf8.Valid(raw) {
		return nil, volumeAttachDeviceError(operation, "attachments must be UTF-8 JSON", nil)
	}
	value := bytes.TrimSpace(raw)
	if len(value) == 0 {
		return nil, volumeAttachDeviceError(operation, "attachments JSON is empty", nil)
	}
	// Validate syntax before shape selection, preserving the original parse cause.
	var parsed json.RawMessage
	if err := json.Unmarshal(value, &parsed); err != nil {
		return nil, volumeAttachDeviceError(operation, "invalid attachments JSON", err)
	}
	switch value[0] {
	case '[':
		var rows []json.RawMessage
		if err := json.Unmarshal(value, &rows); err != nil {
			return nil, volumeAttachDeviceError(operation, "invalid attachments array", err)
		}
		for index, row := range rows {
			fields, err := attachmentObject(row)
			if err != nil {
				return nil, volumeAttachDeviceError(operation, fmt.Sprintf("attachments[%d] must be an object", index), err)
			}
			server, present := fields["server_id"]
			if !present {
				return nil, volumeAttachDeviceError(operation, fmt.Sprintf("attachments[%d] is missing server_id", index), nil)
			}
			server = bytes.TrimSpace(server)
			if len(server) == 0 || server[0] != '"' {
				continue
			}
			var id string
			if err := json.Unmarshal(server, &id); err != nil {
				return nil, volumeAttachDeviceError(operation, fmt.Sprintf("attachments[%d].server_id is invalid JSON", index), err)
			}
			if id != serverID {
				continue
			}
			device, present := fields["device"]
			if !present {
				return nil, volumeAttachDeviceError(operation, fmt.Sprintf("attachments[%d] is missing device", index), nil)
			}
			if bytes.Equal(bytes.TrimSpace(device), []byte("null")) {
				return nil, nil
			}
			return bytes.Clone(device), nil
		}
		return nil, nil
	case '{':
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(value, &fields); err != nil {
			return nil, volumeAttachDeviceError(operation, "invalid attachments object", err)
		}
		if len(fields) == 0 {
			return nil, nil
		}
	case '"':
		var text string
		if err := json.Unmarshal(value, &text); err != nil {
			return nil, volumeAttachDeviceError(operation, "invalid attachments string", err)
		}
		if text == "" {
			return nil, nil
		}
	default:
		return nil, volumeAttachDeviceError(operation, "attachments is not iterable", nil)
	}
	return nil, volumeAttachDeviceError(operation, "attachments[0] must be an object", nil)
}

func volumeAttachDeviceError(operation, message string, cause error) error {
	var err error
	if cause != nil {
		err = fmt.Errorf("%w: %s: %w", resource.ErrInvalidOption, message, cause)
	} else {
		err = fmt.Errorf("%w: %s", resource.ErrInvalidOption, message)
	}
	return &resource.OperationError{Operation: operation, Resource: "volume", Cause: err}
}
