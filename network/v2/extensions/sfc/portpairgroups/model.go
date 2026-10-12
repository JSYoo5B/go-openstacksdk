package portpairgroups

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/internal/sfcrecord"
)

// PortPairGroup is the Python SfcPortPairGroup Resource view. Typed fields are filled only
// when the JSON type matches; Body keeps every declared Body key as raw JSON
// after the Python descriptor conversion (null when unknown), and project_id
// falls back to tenant_id when the resource has no project_id. Wire is the
// actual unwrapped response object (nil without one), Envelope the actual
// response body, and Header and StatusCode the actual response.
type PortPairGroup struct {
	ID                      string
	Name                    string
	Description             string
	ProjectID               string
	TenantID                string
	PortPairs               []string
	PortPairGroupParameters map[string]any
	// TapEnabled is the Python is_tap_enabled attribute.
	TapEnabled *bool

	Body       map[string]json.RawMessage
	Wire       map[string]json.RawMessage
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

func fromRecord(r *sfcrecord.Record) *PortPairGroup {
	if r == nil {
		return nil
	}
	r = r.Clone()
	return &PortPairGroup{ID: r.Text("id"), Name: r.Text("name"), Description: r.Text("description"),
		ProjectID: r.Text("project_id"), TenantID: r.Text("tenant_id"),
		PortPairs:               sfcrecord.Value[[]string](r, "port_pairs"),
		PortPairGroupParameters: sfcrecord.Value[map[string]any](r, "port_pair_group_parameters"),
		TapEnabled:              sfcrecord.Field[bool](r, "tap_enabled"),
		Body:                    r.View, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode}
}

func convert(r *sfcrecord.Record, err error) (*PortPairGroup, error) {
	return fromRecord(r), err
}
