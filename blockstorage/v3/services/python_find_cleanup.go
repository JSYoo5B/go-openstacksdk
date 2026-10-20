package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/internal/cinderowned"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type FindRecordOpts struct {
	Query         map[string]json.RawMessage
	Headers       map[string]string
	IgnoreMissing *bool
}
type FindRecordOption func(*FindRecordOpts) error

func copyFindRecord(v FindRecordOpts) FindRecordOpts {
	v.Query = cinderowned.CloneFields(v.Query)
	v.Headers = maps.Clone(v.Headers)
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	return v
}
func WithFindRecordOpts(v FindRecordOpts) FindRecordOption {
	owned := copyFindRecord(v)
	return func(o *FindRecordOpts) error { *o = copyFindRecord(owned); return nil }
}
func WithFindRecordQuery(key string, value any) FindRecordOption {
	raw, err := json.Marshal(value)
	return func(o *FindRecordOpts) error {
		if err != nil {
			return err
		}
		if o.Query == nil {
			o.Query = map[string]json.RawMessage{}
		}
		o.Query[key] = bytes.Clone(raw)
		return nil
	}
}
func WithFindRecordIgnoreMissing(v bool) FindRecordOption {
	return func(o *FindRecordOpts) error { x := v; o.IgnoreMissing = &x; return nil }
}

// FindRecord follows Service.find's list-only exact ID/name lookup. It reads
// every page before returning a match, detecting duplicates and late errors.
func (a *API) FindRecord(ctx context.Context, identity string, options ...FindRecordOption) (*ServiceRecord, error) {
	t := a.actionTarget()
	if a == nil || a.client == nil {
		return nil, t.Wrap(ctx, "FindRecord", resource.ErrInvalidOption)
	}
	// Freeze transport headers before option callbacks, as the owned preamble does.
	selected := *a.client
	selected.MoreHeaders = maps.Clone(selected.MoreHeaders)
	p, policy, err := cinderowned.Begin(ctx, t, FindRecordOpts{}, copyFindRecord, func(v FindRecordOpts) map[string]string { return v.Headers }, cinderowned.Version{Ceiling: "3.32"}, options)
	if err != nil {
		return nil, t.Wrap(ctx, "FindRecord", err)
	}
	query := cinderowned.CloneFields(policy.Query)
	if raw, exists := query["list_base_path"]; exists {
		truthy, cause := cloudfilter.PythonJSONTruthy(raw)
		if cause != nil {
			return nil, t.Wrap(ctx, "FindRecord", cause)
		}
		if truthy {
			query["base_path"] = bytes.Clone(raw)
		}
	}
	// Service.find consumes but never forwards its microversion formal. Its
	// direct Resource.list call does not restore Proxy._list conflicting attrs.
	for _, key := range []string{"list_base_path", "microversion", "__conflicting_attrs", "jmespath_filters"} {
		delete(query, key)
	}
	listPolicy, _, err := neutronfind.NormalizeProxyListOptions(neutronfind.Options{Query: query, Headers: policy.Headers})
	if err != nil {
		return nil, t.Wrap(ctx, "FindRecord", err)
	}
	selected.Microversion = p.Microversion()
	profile := neutronfind.Profile{ServiceRole: "volume", Kind: "services", Path: actionPath, Plural: "services", IDKey: "id", NameKey: "name", SourceFindPolicy: true, SourceLinkPolicy: true, SourceStatusPolicy: true, PythonLoadedEquality: true, StopFaultRetryCallbacks: true}
	for _, wire := range actionSpec.Fields {
		name := wire
		if wire == "zone" {
			name = "availability_zone"
		}
		if wire == "frozen" {
			name = "is_frozen"
		}
		field := neutronfind.Descriptor{Name: name, Wire: wire, Missing: "null"}
		if wire == "name" {
			field.Alias = "binary"
		}
		profile.Body = append(profile.Body, field)
	}
	profile.Query = []neutronfind.QueryPair{{Name: "host", Wire: "host"}, {Name: "binary", Wire: "binary"}, {Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}}
	engine, err := neutronfind.New(ctx, &selected, profile, p.Guard)
	if err != nil {
		return nil, t.Wrap(ctx, "FindRecord", err)
	}
	var found *ServiceRecord
	for value, cause := range engine.List(ctx, listPolicy) {
		if cause != nil {
			return nil, t.Wrap(ctx, "FindRecord", cause)
		}
		body := cinderowned.CloneFields(value.Resource.Body)
		body["zone"], body["frozen"] = bytes.Clone(body["availability_zone"]), bytes.Clone(body["is_frozen"])
		row := serviceFromRecord(&cinderowned.Record{View: body, Wire: value.Wire.Body, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode, Discovery: p.Discovery()})
		// The older action converter falls back on null. Resource Body aliases
		// fall back only when absent, so retain the declared list projection.
		row.Body["name"] = bytes.Clone(body["name"])
		id, _ := cloudfilter.PythonString(body["id"])
		var name string
		nameErr := json.Unmarshal(body["name"], &name)
		nameString := len(bytes.TrimSpace(body["name"])) != 0 && bytes.TrimSpace(body["name"])[0] == '"' && nameErr == nil
		if identity != id && (!nameString || identity != name) {
			continue
		}
		if host, present := policy.Query["host"]; present {
			matched, cause := jsonHostEqual(body["host"], host)
			if cause != nil {
				return nil, t.Wrap(ctx, "FindRecord", cause)
			}
			if !matched {
				continue
			}
		}
		if found != nil {
			return nil, t.Wrap(ctx, "FindRecord", resource.ErrAmbiguous)
		}
		found = row
	}
	if found == nil && policy.IgnoreMissing != nil && !*policy.IgnoreMissing {
		return nil, t.Wrap(ctx, "FindRecord", resource.ErrNotFound)
	}
	return found, t.Wrap(ctx, "FindRecord", p.Guard(ctx))
}
func jsonHostEqual(a, b json.RawMessage) (bool, error) { return jsonfilter.EqualPythonLoadedJSON(a, b) }

// CleanupWorkersOpts carries JSON-domain filter values. The Go defaults bind
// binary/host/cluster_name to explicit null; Python requires those keywords.
type CleanupWorkersOpts struct {
	Filters map[string]json.RawMessage
	Headers map[string]string
}
type CleanupWorkersOption func(*CleanupWorkersOpts) error

func copyCleanupWorkers(v CleanupWorkersOpts) CleanupWorkersOpts {
	v.Filters = cinderowned.CloneFields(v.Filters)
	v.Headers = maps.Clone(v.Headers)
	return v
}
func WithCleanupWorkersOpts(v CleanupWorkersOpts) CleanupWorkersOption {
	owned := copyCleanupWorkers(v)
	return func(o *CleanupWorkersOpts) error { *o = copyCleanupWorkers(owned); return nil }
}
func WithCleanupWorkerFilter(key string, value any) CleanupWorkersOption {
	raw, err := json.Marshal(value)
	return func(o *CleanupWorkersOpts) error {
		if err != nil {
			return err
		}
		if o.Filters == nil {
			o.Filters = map[string]json.RawMessage{}
		}
		o.Filters[key] = bytes.Clone(raw)
		return nil
	}
}

type CleanupWorkersResult struct {
	JSON json.RawMessage
	ActionReceipt
}

func (a *API) CleanupServiceWorkers(ctx context.Context, options ...CleanupWorkersOption) (*CleanupWorkersResult, error) {
	const op = "CleanupServiceWorkers"
	t := a.actionTarget()
	// Keyword binding fails before version negotiation in the Source method.
	p, err := t.Prepare(ctx)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	policy, err := cinderowned.Apply(ctx, p.Guard, CleanupWorkersOpts{}, copyCleanupWorkers, options)
	if err != nil {
		return nil, t.Wrap(ctx, op, err)
	}
	body := map[string]json.RawMessage{}
	for _, key := range []string{"binary", "host", "cluster_name", "disabled", "is_up", "resource_id", "resource_type", "service_id"} {
		body[key] = json.RawMessage("null")
	}
	body["is_up"] = json.RawMessage("false")
	for key, value := range policy.Filters {
		if _, ok := body[key]; !ok {
			return nil, t.Wrap(ctx, op, cinderowned.Invalid("unknown cleanup keyword %q", key))
		}
		if value == nil {
			value = json.RawMessage("null")
		}
		if !json.Valid(value) {
			return nil, t.Wrap(ctx, op, resource.ErrInvalidOption)
		}
		body[key] = bytes.Clone(value)
	}
	err = p.Headers(policy.Headers)
	if err == nil {
		err = p.Select(cinderowned.Version{Ceiling: "3.32", Minimum: "3.24"})
	}
	if err != nil {
		return &CleanupWorkersResult{ActionReceipt: ActionReceipt{Discovery: p.Discovery()}}, t.Wrap(ctx, op, err)
	}
	response, err := p.Do(http.MethodPut, p.ServiceURL("workers", "cleanup"), body)
	proof := receipt(p, response, err)
	var result *CleanupWorkersResult
	if proof != nil {
		result = &CleanupWorkersResult{ActionReceipt: *proof}
	}
	if err != nil {
		return result, t.Wrap(ctx, op, err)
	}
	if !json.Valid(response.Body) {
		return result, t.Wrap(ctx, op, response.Fail(errors.New("cleanup response must contain JSON")))
	}
	result.JSON = bytes.Clone(response.Body)
	return result, t.Wrap(ctx, op, p.Guard(ctx))
}
