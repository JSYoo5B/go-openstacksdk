package defaulttypes

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/internal/cinderowned"
)

// DefaultType is the Python DefaultType Resource view. ID is the project ID
// used in the route (the Python id); ProjectID and VolumeTypeID come from the
// response body. Body keeps every declared Body key as raw JSON (null when
// unknown); Wire, Envelope, Header and StatusCode keep the actual response
// and Discovery the actual version discovery responses of the call.
type DefaultType struct {
	ID           string
	ProjectID    *string
	VolumeTypeID *string

	Body       map[string]json.RawMessage
	Wire       map[string]json.RawMessage
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
	Discovery  []cinderowned.Observation
}

func fromRecord(r *cinderowned.Record) *DefaultType {
	if r == nil {
		return nil
	}
	r = r.Clone()
	return &DefaultType{ID: r.Text("id"),
		ProjectID:    cinderowned.Field[string](r, "project_id"),
		VolumeTypeID: cinderowned.Field[string](r, "volume_type_id"),
		Body:         r.View, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode, Discovery: r.Discovery}
}

func convert(r *cinderowned.Record, err error) (*DefaultType, error) {
	return fromRecord(r), err
}
