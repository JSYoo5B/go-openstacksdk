package localipassociations

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/internal/ownedrecord"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// LocalIPAssociation is the Python LocalIPAssociation Resource view.
// LocalIPID is the bound URI attribute. Typed fields are filled only when the
// JSON type matches. Resource.Body keeps every declared Body key as raw JSON
// (null when unknown) plus unknown Body keys and location; Wire is the actual
// unwrapped response object (nil without one), Envelope the actual response
// body, and Header and StatusCode the actual response.
type LocalIPAssociation struct {
	LocalIPID      string
	ID             string
	Name           *string
	FixedPortID    *string
	FixedIP        *string
	Host           *string
	LocalIPAddress *string

	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
}

func fromRecord(r *ownedrecord.Record) *LocalIPAssociation {
	if r == nil {
		return nil
	}
	return &LocalIPAssociation{LocalIPID: r.Parent, ID: ownedrecord.Text(r, "id"),
		Name:           ownedrecord.Field[string](r, "name"),
		FixedPortID:    ownedrecord.Field[string](r, "fixed_port_id"),
		FixedIP:        ownedrecord.Field[string](r, "fixed_ip"),
		Host:           ownedrecord.Field[string](r, "host"),
		LocalIPAddress: ownedrecord.Field[string](r, "local_ip_address"),
		Resource:       r.Resource, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode}
}

func convert(r *ownedrecord.Record, err error) (*LocalIPAssociation, error) {
	return fromRecord(r), err
}
