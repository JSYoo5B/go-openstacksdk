package serviceproviders

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/internal/osfederation"
)

// ServiceProvider is the Python ServiceProvider Resource view. ID is both
// the Python id and name. Typed fields are filled only when the JSON type
// matches; Body keeps every declared Body key as raw JSON (null when unknown).
// Wire is the actual unwrapped response object (nil without one), Envelope
// the actual response body, and Header and StatusCode the actual response.
type ServiceProvider struct {
	ID               string
	AuthURL          *string
	Description      *string
	Enabled          *bool
	RelayStatePrefix *string
	SPURL            *string

	Body       map[string]json.RawMessage
	Wire       map[string]json.RawMessage
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

func fromRecord(r *osfederation.Record) *ServiceProvider {
	if r == nil {
		return nil
	}
	r = r.Clone()
	return &ServiceProvider{ID: r.Text("id"),
		AuthURL:          osfederation.Field[string](r, "auth_url"),
		Description:      osfederation.Field[string](r, "description"),
		Enabled:          osfederation.Field[bool](r, "enabled"),
		RelayStatePrefix: osfederation.Field[string](r, "relay_state_prefix"),
		SPURL:            osfederation.Field[string](r, "sp_url"),
		Body:             r.View, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode}
}

func convert(r *osfederation.Record, err error) (*ServiceProvider, error) {
	return fromRecord(r), err
}
