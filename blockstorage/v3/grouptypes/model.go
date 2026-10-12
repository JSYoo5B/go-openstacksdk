package grouptypes

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/internal/cinderowned"
)

// GroupType is the Python GroupType Resource view. Typed fields are filled
// only when the JSON type matches; Body keeps every declared Body key as raw
// JSON (group_specs defaults to {}, other unknown keys are null). Wire is the
// actual unwrapped response object (nil without one), Envelope the actual
// response body, Header and StatusCode the actual response, and Discovery the
// actual version discovery responses of the call.
type GroupType struct {
	ID          string
	Name        string
	Description *string
	IsPublic    *bool
	GroupSpecs  map[string]string

	Body       map[string]json.RawMessage
	Wire       map[string]json.RawMessage
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
	Discovery  []cinderowned.Observation
}

// SpecProperty is the result of the single group spec calls. Value is the
// response member named Key (JSON null when GET finds no member); the other
// fields keep the actual response.
type SpecProperty struct {
	GroupTypeID string
	Key         string
	Value       json.RawMessage

	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

func fromRecord(r *cinderowned.Record) *GroupType {
	if r == nil {
		return nil
	}
	r = r.Clone()
	value := &GroupType{ID: r.Text("id"), Name: r.Text("name"),
		Description: cinderowned.Field[string](r, "description"),
		IsPublic:    cinderowned.Field[bool](r, "is_public"),
		Body:        r.View, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode, Discovery: r.Discovery}
	if specs := cinderowned.Field[map[string]string](r, "group_specs"); specs != nil {
		value.GroupSpecs = *specs
	}
	return value
}

func convert(r *cinderowned.Record, err error) (*GroupType, error) {
	return fromRecord(r), err
}
