package projectendpoints

import (
	"context"
	"encoding/json"
	"iter"
	"net/http"
	"net/url"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/internal/osfederation"
	"github.com/gophercloud/gophercloud/v2"
)

// recordSpec is the pinned openstacksdk identity v3 ProjectEndpoint Resource,
// which inherits Endpoint: Body id, name, interface, enabled, links,
// region_id, service_id and url; QueryParameters interface, region_id and
// service_id plus limit and marker; other Body attributes filter locally.
var recordSpec = &osfederation.Spec{
	Kind: "projectendpoints", ResourceKey: "endpoint", ResourcesKey: "endpoints",
	Fields: []string{"id", "name", "interface", "enabled", "links", "region_id", "service_id", "url"},
	Attrs: map[string]string{"id": "id", "name": "name", "interface": "interface", "is_enabled": "enabled", "enabled": "enabled",
		"links": "links", "region_id": "region_id", "service_id": "service_id", "url": "url"},
	Query:   map[string]string{"interface": "interface", "region_id": "region_id", "service_id": "service_id"},
	Filters: map[string]string{"id": "id", "name": "name", "is_enabled": "enabled", "links": "links", "url": "url"},
}

// EndpointRecord is the Python ProjectEndpoint Resource view. ProjectID is
// the URI attribute Python sets on every listed row. Typed fields are filled
// only when the JSON type matches. Body keeps the eight declared Body keys as
// raw JSON (null when unknown); Wire is the actual row, Envelope the actual
// page body, and Header and StatusCode the actual page response.
type EndpointRecord struct {
	ID        string
	ProjectID string
	Name      *string
	Interface *string
	Enabled   *bool
	RegionID  *string
	ServiceID *string
	URL       *string
	Links     map[string]any

	Body       map[string]json.RawMessage
	Wire       map[string]json.RawMessage
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

func endpointRecord(r *osfederation.Record, projectID string) *EndpointRecord {
	if r == nil {
		return nil
	}
	r = r.Clone()
	value := &EndpointRecord{ID: r.Text("id"), ProjectID: projectID, Name: osfederation.Field[string](r, "name"),
		Interface: osfederation.Field[string](r, "interface"), Enabled: osfederation.Field[bool](r, "enabled"),
		RegionID: osfederation.Field[string](r, "region_id"), ServiceID: osfederation.Field[string](r, "service_id"),
		URL:  osfederation.Field[string](r, "url"),
		Body: r.View, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode}
	if links := osfederation.Field[map[string]any](r, "links"); links != nil {
		value.Links = *links
	}
	return value
}

func (a *API) recordTarget(projectID string) osfederation.Target {
	var client func() *gophercloud.ServiceClient
	if a != nil {
		client = func() *gophercloud.ServiceClient { return a.client }
	}
	return osfederation.Target{Spec: recordSpec, Client: client,
		Path: "OS-EP-FILTER/projects/" + url.PathEscape(projectID) + "/endpoints",
		Err:  osfederation.ValidateID("project ID", projectID)}
}

// ListRecords implements project_endpoints(project, **query) as a lazy
// iterator over GET OS-EP-FILTER/projects/{project_id}/endpoints.
func (a *API) ListRecords(ctx context.Context, projectID string, options ...RecordListOption) iter.Seq2[*EndpointRecord, error] {
	source := osfederation.List(ctx, a.recordTarget(projectID), options)
	return func(yield func(*EndpointRecord, error) bool) {
		for record, err := range source {
			if !yield(endpointRecord(record, projectID), err) || err != nil {
				return
			}
		}
	}
}

// AllRecords collects ListRecords; a nonnil empty slice on success.
func (a *API) AllRecords(ctx context.Context, projectID string, options ...RecordListOption) ([]*EndpointRecord, error) {
	values := make([]*EndpointRecord, 0)
	for value, err := range a.ListRecords(ctx, projectID, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// RecordListOpts carries Python **query: interface, region_id and
// service_id go to the server; id, name, is_enabled (or enabled), links and
// url filter rows locally; other names are ignored.
type RecordListOpts = osfederation.ListOpts
type RecordListOption func(*RecordListOpts) error

func WithRecordListOpts(opts RecordListOpts) RecordListOption {
	return RecordListOption(osfederation.ListSnapshot(opts))
}
func WithRecordListQuery(key string, value any) RecordListOption {
	return RecordListOption(recordSpec.ListQuery(key, value))
}
func WithRecordListLimit(limit int) RecordListOption {
	return RecordListOption(osfederation.ListLimit(limit))
}
func WithRecordListMarker(marker string) RecordListOption {
	return RecordListOption(osfederation.ListMarker(marker))
}
func WithRecordListMaxItems(count int) RecordListOption {
	return RecordListOption(osfederation.ListMaxItems(count))
}
func WithRecordListPaginated(value bool) RecordListOption {
	return RecordListOption(osfederation.ListPaginated(value))
}
func WithRecordListHeaders(values map[string]string) RecordListOption {
	return RecordListOption(osfederation.ListHeaders(values))
}
