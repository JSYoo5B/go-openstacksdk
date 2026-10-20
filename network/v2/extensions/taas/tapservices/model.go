package tapservices

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecord"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// TapService is the Python TapService Resource view. Typed fields are filled only
// when the JSON type matches. Resource.Body keeps every declared Body key as
// raw JSON (null when unknown) plus unknown Body keys and location; Wire is
// the actual unwrapped response object (nil without one), Envelope the actual
// response body, and Header and StatusCode the actual response.
type TapService struct {
	ID          string
	Name        *string
	Description *string
	ProjectID   *string
	TenantID    *string
	PortID      *string
	Status      *string

	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func fromRecord(r *ownedrecord.Record) *TapService {
	if r == nil {
		return nil
	}
	return &TapService{ID: ownedrecord.Text(r, "id"),
		Name:        ownedrecord.Field[string](r, "name"),
		Description: ownedrecord.Field[string](r, "description"),
		ProjectID:   ownedrecord.Field[string](r, "project_id"),
		TenantID:    ownedrecord.Field[string](r, "tenant_id"),
		PortID:      ownedrecord.Field[string](r, "port_id"),
		Status:      ownedrecord.Field[string](r, "status"),
		Resource:    r.Resource, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode}
}

func convert(r *ownedrecord.Record, err error) (*TapService, error) {
	return fromRecord(r), err
}
