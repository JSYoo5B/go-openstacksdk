package federation

import (
	"context"
	"encoding/json"
	"iter"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/internal/osfederation"
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

// mappingSpec is the pinned openstacksdk identity v3 Mapping Resource: Body
// id (also the Python name), rules and schema_version; empty QueryParameters
// apart from limit and marker, so every Body attribute filters rows locally.
var mappingSpec = &osfederation.Spec{
	Kind: "mappings", ResourceKey: "mapping", ResourcesKey: "mappings",
	Fields:  []string{"id", "rules", "schema_version"},
	Attrs:   map[string]string{"id": "id", "name": "id", "rules": "rules", "schema_version": "schema_version"},
	Query:   map[string]string{},
	Filters: map[string]string{"id": "id", "name": "id", "rules": "rules", "schema_version": "schema_version"},
}

// MappingRecord is the Python Mapping Resource view. ID is both the Python
// id and name. Rules is the raw rules JSON, so rule keys outside the native
// MappingRule structs are kept. Body keeps the three declared Body keys as
// raw JSON (null when unknown); Wire is the actual unwrapped response object
// (nil without one), Envelope the actual response body, and Header and
// StatusCode the actual response.
type MappingRecord struct {
	ID            string
	SchemaVersion *string
	Rules         json.RawMessage

	Body       map[string]json.RawMessage
	Wire       map[string]json.RawMessage
	Envelope   json.RawMessage
	Header     http.Header
	StatusCode int
}

func mappingRecord(r *osfederation.Record) *MappingRecord {
	if r == nil {
		return nil
	}
	r = r.Clone()
	value := &MappingRecord{ID: r.Text("id"), SchemaVersion: osfederation.Field[string](r, "schema_version"),
		Body: r.View, Wire: r.Wire, Envelope: r.Envelope, Header: r.Header, StatusCode: r.StatusCode}
	if raw := r.View["rules"]; string(raw) != "null" {
		value.Rules = raw
	}
	return value
}

func mappingResult(r *osfederation.Record, err error) (*MappingRecord, error) {
	return mappingRecord(r), err
}

func (a *API) mappingTarget() osfederation.Target {
	var client func() *gophercloud.ServiceClient
	if a != nil {
		client = func() *gophercloud.ServiceClient { return a.client }
	}
	return osfederation.Target{Spec: mappingSpec, Client: client, Path: "OS-FEDERATION/mappings"}
}

// GetMappingRecord implements get_mapping(mapping): GET OS-FEDERATION/mappings/{id}.
func (a *API) GetMappingRecord(ctx context.Context, id string, options ...MappingRecordReadOption) (*MappingRecord, error) {
	return mappingResult(osfederation.Get(ctx, a.mappingTarget(), id, calls.Translate(options, osfederation.ReadCalls)))
}

// UpdateMappingRecord implements update_mapping(mapping, **attrs): PATCH with
// only the given declared attributes. No declared attribute means no request.
func (a *API) UpdateMappingRecord(ctx context.Context, id string, options ...MappingRecordWriteOption) (*MappingRecord, error) {
	return mappingResult(osfederation.Update(ctx, a.mappingTarget(), id, calls.Translate(options, mappingWriteCalls)))
}

// DeleteMappingRecord implements delete_mapping(mapping, ignore_missing=True).
func (a *API) DeleteMappingRecord(ctx context.Context, id string, options ...MappingRecordDeleteOption) error {
	return osfederation.Delete(ctx, a.mappingTarget(), id, calls.Translate(options, osfederation.DeleteIgnoreMissingCalls))
}

// ListMappingRecords implements mappings(**query) as a lazy iterator.
func (a *API) ListMappingRecords(ctx context.Context, options ...MappingRecordListOption) iter.Seq2[*MappingRecord, error] {
	source := osfederation.List(ctx, a.mappingTarget(), calls.Translate(options, mappingListCalls))
	return func(yield func(*MappingRecord, error) bool) {
		for record, err := range source {
			if !yield(mappingRecord(record), err) || err != nil {
				return
			}
		}
	}
}

// AllMappingRecords collects ListMappingRecords; a nonnil empty slice on success.
func (a *API) AllMappingRecords(ctx context.Context, options ...MappingRecordListOption) ([]*MappingRecord, error) {
	values := make([]*MappingRecord, 0)
	for value, err := range a.ListMappingRecords(ctx, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// FindMappingRecord implements find_mapping(name_or_id, ignore_missing=True):
// GET by ID first and, after a clean 400, 403 or 404, the complete list
// matched by exact ID (name is the ID).
func (a *API) FindMappingRecord(ctx context.Context, nameOrID string, options ...MappingRecordFindOption) (*MappingRecord, error) {
	return mappingResult(osfederation.Find(ctx, a.mappingTarget(), nameOrID, nil, calls.Translate(options, osfederation.FindCalls)))
}

// MappingRecordReadOpts carries per-request headers, a Go extension.
type MappingRecordReadOpts = osfederation.ReadOpts

// MappingRecordReadOption is one option of GetMappingRecord: request.Headers
// (replaces the headers) or request.Custom over *MappingRecordReadOpts. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type MappingRecordReadOption = request.GetCall

// MappingRecordWriteOpts carries the Python **attrs of update_mapping as raw
// JSON keyed by rules or schema_version; id and name never enter the body.
type MappingRecordWriteOpts = osfederation.WriteOpts

// MappingRecordWriteOption is one option of UpdateMappingRecord:
// WithMappingRecordWriteOpts, request.Field (one Python attribute; unknown
// names are dropped with their marshal error), request.Headers (replaces the
// headers) or request.Custom over *MappingRecordWriteOpts. Other shared calls
// fail with resource.ErrInvalidOption before HTTP.
type MappingRecordWriteOption = request.WriteCall

func WithMappingRecordWriteOpts(opts MappingRecordWriteOpts) MappingRecordWriteOption {
	return request.Custom(osfederation.WriteSnapshot(opts))
}

// MappingRecordDeleteOpts selects ignore_missing; nil keeps the Python default true.
type MappingRecordDeleteOpts = osfederation.DeleteOpts

// MappingRecordDeleteOption is one option of DeleteMappingRecord:
// request.IgnoreMissing or request.Custom over *MappingRecordDeleteOpts. Other
// shared calls, including request.Header and request.Headers, fail with
// resource.ErrInvalidOption before HTTP.
type MappingRecordDeleteOption = request.DeleteCall

// MappingRecordFindOpts selects ignore_missing; nil keeps the Python default true.
type MappingRecordFindOpts = osfederation.FindOpts

// MappingRecordFindOption is one option of FindMappingRecord:
// request.IgnoreMissing or request.Custom over *MappingRecordFindOpts. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
type MappingRecordFindOption = request.FindCall

// MappingRecordListOpts carries Python **query. Mapping maps no server query
// besides limit and marker, so id, name, rules and schema_version filter rows
// locally and other names are ignored.
type MappingRecordListOpts = osfederation.ListOpts

// MappingRecordListOption is one option of ListMappingRecords and
// AllMappingRecords: WithMappingRecordListOpts, request.Query (one Python
// **query name as JSON; unknown names are dropped with their marshal error),
// request.Limit, request.Marker, request.MaxItems, request.Paginated,
// request.Headers (replaces the headers) or request.Custom over
// *MappingRecordListOpts. Other shared calls fail with
// resource.ErrInvalidOption before HTTP.
type MappingRecordListOption = request.ListCall

func WithMappingRecordListOpts(opts MappingRecordListOpts) MappingRecordListOption {
	return request.Custom(osfederation.ListSnapshot(opts))
}

var (
	mappingWriteCalls = mappingSpec.WriteCalls()
	mappingListCalls  = mappingSpec.ListCalls()
)
