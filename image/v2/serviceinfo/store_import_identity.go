package serviceinfo

import (
	"bytes"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
)

// ImportIdentity returns a copy of this SDK-produced store's private JSON ID.
// It preserves a missing ID as null and accepts the untyped Source descriptor.
// Public Resource, Wire, Envelope or Header edits cannot retarget an import.
// A caller-constructed StoreRecord has no private identity and is rejected.
func (value *StoreRecord) ImportIdentity() (json.RawMessage, error) {
	if value == nil || value.importID == nil {
		return nil, infoInvalid("SDK-produced store record is required for import identity")
	}
	return bytes.Clone(value.importID), nil
}

// StoreImportIdentity locally converts JSON constructor attributes to the
// untyped Store ID used by Image.import_image. It consumes only canonical id,
// uses null when absent, and never falls back to name or performs HTTP.
// Singular compatibility headers require a string at the importing layer;
// plural store IDs can retain arbitrary complete JSON values.
func StoreImportIdentity(attributes json.RawMessage) (json.RawMessage, error) {
	members, err := cloudfilter.ObjectMembers(attributes)
	if err != nil {
		return nil, infoInvalid("store import attributes must be a complete UTF-8 JSON object: %v", err)
	}
	identity := json.RawMessage("null")
	for _, member := range members {
		switch member.Key {
		case "connection", "_synchronized", "microversion":
			return nil, infoInvalid("store import attribute %q collides with a source constructor argument", member.Key)
		case "id":
			identity = bytes.Clone(member.Value)
		}
	}
	return identity, nil
}
