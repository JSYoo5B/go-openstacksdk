package localips

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecord"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// LocalIP is the Python LocalIP Resource view. Typed fields are filled only
// when the JSON type matches; RevisionNumber is decoded only from a JSON
// integer because Python does not cast it. Resource.Body keeps every declared
// Body key as raw JSON (null when unknown) plus unknown Body keys and
// location; Wire is the actual unwrapped response object (nil without one),
// Envelope the actual response body, and Header and StatusCode the actual
// response.
type LocalIP struct {
	ID             string
	Name           *string
	Description    *string
	ProjectID      *string
	NetworkID      *string
	LocalPortID    *string
	LocalIPAddress *string
	IPMode         *string
	RevisionNumber *int64
	CreatedAt      *string
	UpdatedAt      *string

	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func fromRecord(r *ownedrecord.Record) *LocalIP {
	if r == nil {
		return nil
	}
	return &LocalIP{ID: ownedrecord.Text(r, "id"),
		Name:           ownedrecord.Field[string](r, "name"),
		Description:    ownedrecord.Field[string](r, "description"),
		ProjectID:      ownedrecord.Field[string](r, "project_id"),
		NetworkID:      ownedrecord.Field[string](r, "network_id"),
		LocalPortID:    ownedrecord.Field[string](r, "local_port_id"),
		LocalIPAddress: ownedrecord.Field[string](r, "local_ip_address"),
		IPMode:         ownedrecord.Field[string](r, "ip_mode"),
		RevisionNumber: ownedrecord.Field[int64](r, "revision_number"),
		CreatedAt:      ownedrecord.Field[string](r, "created_at"),
		UpdatedAt:      ownedrecord.Field[string](r, "updated_at"),
		Resource:       r.Resource, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode}
}

func convert(r *ownedrecord.Record, err error) (*LocalIP, error) {
	return fromRecord(r), err
}
