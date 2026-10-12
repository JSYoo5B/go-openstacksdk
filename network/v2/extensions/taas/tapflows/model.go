package tapflows

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecord"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// TapFlow is the Python TapFlow Resource view. Typed fields are filled only
// when the JSON type matches. Resource.Body keeps every declared Body key as
// raw JSON (null when unknown) plus unknown Body keys and location; Wire is
// the actual unwrapped response object (nil without one), Envelope the actual
// response body, and Header and StatusCode the actual response.
type TapFlow struct {
	ID           string
	Name         *string
	Description  *string
	ProjectID    *string
	TenantID     *string
	TapServiceID *string
	Direction    *string
	Status       *string
	SourcePort   *string

	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func fromRecord(r *ownedrecord.Record) *TapFlow {
	if r == nil {
		return nil
	}
	return &TapFlow{ID: ownedrecord.Text(r, "id"),
		Name:         ownedrecord.Field[string](r, "name"),
		Description:  ownedrecord.Field[string](r, "description"),
		ProjectID:    ownedrecord.Field[string](r, "project_id"),
		TenantID:     ownedrecord.Field[string](r, "tenant_id"),
		TapServiceID: ownedrecord.Field[string](r, "tap_service_id"),
		Direction:    ownedrecord.Field[string](r, "direction"),
		Status:       ownedrecord.Field[string](r, "status"),
		SourcePort:   ownedrecord.Field[string](r, "source_port"),
		Resource:     r.Resource, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode}
}

func convert(r *ownedrecord.Record, err error) (*TapFlow, error) {
	return fromRecord(r), err
}
