package portpairs

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/internal/sfcrecord"
)

// PortPair is the Python SfcPortPair Resource view. Typed fields are filled only
// when the JSON type matches; Body keeps every declared Body key as raw JSON
// after the Python descriptor conversion (null when unknown), and project_id
// falls back to tenant_id when the resource has no project_id. Wire is the
// actual unwrapped response object (nil without one), Envelope the actual
// response body, and Header and StatusCode the actual response.
type PortPair struct {
	ID                        string
	Name                      string
	Description               string
	ProjectID                 string
	TenantID                  string
	Ingress                   *string
	Egress                    *string
	ServiceFunctionParameters map[string]any

	Body       map[string]json.RawMessage
	Wire       map[string]json.RawMessage
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

func fromRecord(r *sfcrecord.Record) *PortPair {
	if r == nil {
		return nil
	}
	r = r.Clone()
	return &PortPair{ID: r.Text("id"), Name: r.Text("name"), Description: r.Text("description"),
		ProjectID: r.Text("project_id"), TenantID: r.Text("tenant_id"),
		Ingress:                   sfcrecord.Field[string](r, "ingress"),
		Egress:                    sfcrecord.Field[string](r, "egress"),
		ServiceFunctionParameters: sfcrecord.Value[map[string]any](r, "service_function_parameters"),
		Body:                      r.View, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode}
}

func convert(r *sfcrecord.Record, err error) (*PortPair, error) {
	return fromRecord(r), err
}
