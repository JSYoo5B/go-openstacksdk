package secrets

import (
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/gophercloudsdk/internal/jsonfilter"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// secretBodyFilterValue projects the pinned Python attribute from an original
// list row after native whole-page decoding. IDs are passive values, never URLs
// used for requests. Only selected attributes invoke their descriptor behavior.
func secretBodyFilterValue(record *resource.BodyRecord[Secret], key string) (json.RawMessage, error) {
	if record == nil {
		return nil, fmt.Errorf("%w: secret Body record is required", resource.ErrInvalidOption)
	}
	if record.Fields == nil {
		return nil, fmt.Errorf("%w: null secret list row is not an object", resource.ErrInvalidOption)
	}
	field := key
	switch key {
	case "id":
		// Resource.id gives literal key presence priority, including null,
		// empty, numeric and object values. Its alternate is the full raw ref;
		// it bypasses the HREF formatter used by secret_id.
		if _, exists := record.Fields["id"]; !exists {
			field = "secret_ref"
		}
	case "secret_id":
		return secretBodyReferenceID(record.Fields)
	case "created_at":
		field = "created"
	case "updated_at":
		field = "updated"
	case "expires_at":
		field = "expiration"
	case "bit_length", "content_types", "secret_ref", "status", "payload", "payload_content_type", "payload_content_encoding":
	default:
		return nil, fmt.Errorf("%w: unsupported secret Body filter %q", resource.ErrInvalidOption, key)
	}
	return resource.BodyRecordField(record.Fields, field, resource.BodyFieldJSON)
}

func secretBodyReferenceID(fields map[string]json.RawMessage) (json.RawMessage, error) {
	raw, err := resource.BodyRecordField(fields, "secret_ref", resource.BodyFieldJSON)
	if err != nil {
		return nil, err
	}
	value, err := jsonfilter.ReferenceLastComponent(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: secret_ref formatter: %w", resource.ErrInvalidOption, err)
	}
	return value, nil
}
