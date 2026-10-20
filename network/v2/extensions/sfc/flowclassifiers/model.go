package flowclassifiers

import (
	"encoding/json"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/sfc/internal/sfcrecord"
)

// FlowClassifier is the Python SfcFlowClassifier Resource view. Typed fields are filled only
// when the JSON type matches; Body keeps every declared Body key as raw JSON
// after the Python descriptor conversion (null when unknown), and project_id
// falls back to tenant_id when the resource has no project_id. Wire is the
// actual unwrapped response object (nil without one), Envelope the actual
// response body, and Header and StatusCode the actual response.
type FlowClassifier struct {
	ID          string
	Name        string
	Description string
	ProjectID   string
	TenantID    string
	// Ethertype is IPv4 or IPv6.
	Ethertype *string
	// Protocol is the IP protocol name; integer spellings stay in Body.
	Protocol                *string
	SourcePortRangeMin      *int
	SourcePortRangeMax      *int
	DestinationPortRangeMin *int
	DestinationPortRangeMax *int
	SourceIPPrefix          *string
	DestinationIPPrefix     *string
	LogicalSourcePort       *string
	LogicalDestinationPort  *string
	L7Parameters            map[string]any

	Body       map[string]json.RawMessage
	Wire       map[string]json.RawMessage
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

func fromRecord(r *sfcrecord.Record) *FlowClassifier {
	if r == nil {
		return nil
	}
	r = r.Clone()
	return &FlowClassifier{ID: r.Text("id"), Name: r.Text("name"), Description: r.Text("description"),
		ProjectID: r.Text("project_id"), TenantID: r.Text("tenant_id"),
		Ethertype:               sfcrecord.Field[string](r, "ethertype"),
		Protocol:                sfcrecord.Field[string](r, "protocol"),
		SourcePortRangeMin:      sfcrecord.Field[int](r, "source_port_range_min"),
		SourcePortRangeMax:      sfcrecord.Field[int](r, "source_port_range_max"),
		DestinationPortRangeMin: sfcrecord.Field[int](r, "destination_port_range_min"),
		DestinationPortRangeMax: sfcrecord.Field[int](r, "destination_port_range_max"),
		SourceIPPrefix:          sfcrecord.Field[string](r, "source_ip_prefix"),
		DestinationIPPrefix:     sfcrecord.Field[string](r, "destination_ip_prefix"),
		LogicalSourcePort:       sfcrecord.Field[string](r, "logical_source_port"),
		LogicalDestinationPort:  sfcrecord.Field[string](r, "logical_destination_port"),
		L7Parameters:            sfcrecord.Value[map[string]any](r, "l7_parameters"),
		Body:                    r.View, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode}
}

func convert(r *sfcrecord.Record, err error) (*FlowClassifier, error) {
	return fromRecord(r), err
}
