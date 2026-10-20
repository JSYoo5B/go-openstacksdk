package identityproviders

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/internal/osfederation"
)

// IdentityProvider is the Python IdentityProvider Resource view. ID is both
// the Python id and name. Typed fields are filled only when the JSON type
// matches; Body keeps every declared Body key as raw JSON (null when unknown).
// Wire is the actual unwrapped response object (nil without one), Envelope
// the actual response body, and Header and StatusCode the actual response.
type IdentityProvider struct {
	ID               string
	DomainID         *string
	Description      *string
	Enabled          *bool
	RemoteIDs        []string
	AuthorizationTTL *int64

	Body       map[string]json.RawMessage
	Wire       map[string]json.RawMessage
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

func fromRecord(r *osfederation.Record) *IdentityProvider {
	if r == nil {
		return nil
	}
	r = r.Clone()
	return &IdentityProvider{ID: r.Text("id"),
		DomainID:         osfederation.Field[string](r, "domain_id"),
		Description:      osfederation.Field[string](r, "description"),
		Enabled:          osfederation.Field[bool](r, "enabled"),
		RemoteIDs:        osfederation.Strings(r, "remote_ids"),
		AuthorizationTTL: osfederation.Field[int64](r, "authorization_ttl"),
		Body:             r.View, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode}
}

func convert(r *osfederation.Record, err error) (*IdentityProvider, error) {
	return fromRecord(r), err
}
