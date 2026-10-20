package portchains

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/internal/sfcrecord"
)

// PortChain is the Python SfcPortChain Resource view. Typed fields are filled only
// when the JSON type matches; Body keeps every declared Body key as raw JSON
// after the Python descriptor conversion (null when unknown), and project_id
// falls back to tenant_id when the resource has no project_id. Wire is the
// actual unwrapped response object (nil without one), Envelope the actual
// response body, and Header and StatusCode the actual response.
type PortChain struct {
	ID              string
	Name            string
	Description     string
	ProjectID       string
	TenantID        string
	PortPairGroups  []string
	FlowClassifiers []string
	ChainParameters map[string]any

	Body       map[string]json.RawMessage
	Wire       map[string]json.RawMessage
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

func fromRecord(r *sfcrecord.Record) *PortChain {
	if r == nil {
		return nil
	}
	r = r.Clone()
	return &PortChain{ID: r.Text("id"), Name: r.Text("name"), Description: r.Text("description"),
		ProjectID: r.Text("project_id"), TenantID: r.Text("tenant_id"),
		PortPairGroups:  sfcrecord.Value[[]string](r, "port_pair_groups"),
		FlowClassifiers: sfcrecord.Value[[]string](r, "flow_classifiers"),
		ChainParameters: sfcrecord.Value[map[string]any](r, "chain_parameters"),
		Body:            r.View, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode}
}

func convert(r *sfcrecord.Record, err error) (*PortChain, error) {
	return fromRecord(r), err
}
