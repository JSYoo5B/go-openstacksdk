package federationprotocols

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/internal/osfederation"
)

// FederationProtocol is the Python FederationProtocol Resource view. ID is
// both the Python id and name. IdentityProviderID is the URI attribute
// idp_id taken from the call. MappingID is filled only for a JSON string;
// Body keeps the declared Body keys id and mapping_id as raw JSON. Wire is
// the actual unwrapped response object (nil without one), Envelope the
// actual response body, and Header and StatusCode the actual response.
type FederationProtocol struct {
	ID                 string
	IdentityProviderID string
	MappingID          *string

	Body       map[string]json.RawMessage
	Wire       map[string]json.RawMessage
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

func fromRecord(r *osfederation.Record, idpID string) *FederationProtocol {
	if r == nil {
		return nil
	}
	r = r.Clone()
	return &FederationProtocol{ID: r.Text("id"), IdentityProviderID: idpID,
		MappingID: osfederation.Field[string](r, "mapping_id"),
		Body:      r.View, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode}
}

func convert(idpID string) func(*osfederation.Record, error) (*FederationProtocol, error) {
	return func(r *osfederation.Record, err error) (*FederationProtocol, error) {
		return fromRecord(r, idpID), err
	}
}
