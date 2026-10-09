package image

import (
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Whole-image mutations share one immutable constructor and descriptor view.
// Private current/original/dirty components and passive fetch channels stay
// distinct; neither the descriptor projection nor an opaque ACK cleans Body.
func imageRecordMutationSeed(id string, record *ImageRecord) (*ImageRecord, string, error) {
	if id != "" && record != nil {
		return nil, "", uploadInvalid("select image ID or Record, not both")
	}
	seed := cloneImageRecord(record)
	if seed == nil {
		if err := validateImageRecordIdentity(id); err != nil {
			return nil, "", err
		}
		rawID, _ := json.Marshal(id)
		seed = &ImageRecord{bodyState: pendingImageRecordBodyState(map[string]json.RawMessage{"id": rawID})}
	} else if seed.bodyState == nil {
		return nil, "", uploadInvalid("image Record must retain SDK-produced raw body state")
	}
	identity, err := decodeImageRecordString(seed.bodyState.current["id"], "image Record identity")
	if err != nil {
		return nil, "", err
	}
	if err := validateImageRecordIdentity(identity); err != nil {
		return nil, "", err
	}
	return seed, identity, nil
}

func projectImageRecordMutationSeed(seed *ImageRecord, location json.RawMessage) error {
	// Source _get_resource(existing)._update() invokes the Image header collector
	// even with no supplied attributes, resetting its plain import-method list.
	seed.ImportMethods = make([]string, 0)
	var passive resource.Metadata
	if seed.Resource != nil {
		passive = seed.Resource.Metadata
	}
	view, err := projectImageRecord(seed.bodyState.current, location,
		resource.Metadata{Header: seed.Header.Clone(), StatusCode: seed.StatusCode})
	if err != nil {
		return err
	}
	// These already-owned passive fields do not enter Body or the route.
	view.CreatedAt, view.UpdatedAt, view.Links = passive.CreatedAt, passive.UpdatedAt, passive.Links
	seed.Resource = view
	return nil
}
