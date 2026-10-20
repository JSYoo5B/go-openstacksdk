package nodes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

func cloneNodeDiscovery(v microversions.Discovery) microversions.Discovery {
	v.Responses = slices.Clone(v.Responses)
	for i, r := range v.Responses {
		if r != nil {
			c := *r
			c.Body = bytes.Clone(r.Body)
			c.Header = r.Header.Clone()
			v.Responses[i] = &c
		}
	}
	return v
}
func nodeReadClientSnapshot(a *API) gophercloud.ServiceClient {
	c := *a.client
	c.MoreHeaders = maps.Clone(c.MoreHeaders)
	return c
}
func (a *API) versionedReadEngine(ctx context.Context, original *neutronfind.Engine, selected *string, client gophercloud.ServiceClient, headers map[string]string) (*neutronfind.Engine, microversions.Discovery, error) {
	var discovery microversions.Discovery
	if err := original.Check(ctx); err != nil {
		return nil, discovery, err
	}
	source, err := cloudread.Capture(ctx, &client, "baremetal")
	if err != nil {
		return nil, discovery, err
	}
	version := client.Microversion
	if selected != nil {
		version = *selected
	}
	validationClient := client
	validationClient.MoreHeaders = maps.Clone(client.MoreHeaders)
	for key := range validationClient.MoreHeaders {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Ironic-API-Version") {
			delete(validationClient.MoreHeaders, key)
		}
	}
	validationClient.Microversion = version
	validation, validationErr := cloudread.Capture(ctx, &validationClient, "baremetal")
	if validationErr == nil {
		validationErr = validation.WithPolicy(ctx, nil, headers)
	}
	if validationErr != nil {
		return nil, discovery, validationErr
	}
	if version == "" && selected == nil {
		codes := make([]int, 200)
		for i := range codes {
			codes[i] = i + 200
		}
		// Engine.Check includes the outer context guard. Calling it with the
		// discovery context would invoke this very guard recursively; retain
		// the original operation context for the captured engine invariant.
		discoveryContext := rest.WithOperationGuard(ctx, func(context.Context) error { return original.Check(ctx) })
		discovery, err = microversions.Read(discoveryContext, source, microversions.Profile{Major: "1", HeaderService: "baremetal", LegacyVersionHeader: "X-OpenStack-Ironic-API-Version", OkCodes: codes, StopFaultRetryCallbacks: true})
		if err == nil {
			version, err = microversions.SelectVersion(discovery.Maximum, discovery.Minimum, "1.109")
		}
		if err != nil {
			return nil, discovery, err
		}
	}
	// Matching original version headers belong to the captured selection, not a
	// caller override. The new private reader owns both headers at its new version.
	for key := range client.MoreHeaders {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Ironic-API-Version") {
			delete(client.MoreHeaders, key)
		}
	}
	client.Microversion = version
	engine, err := neutronfind.New(ctx, &client, listRecordProfile(), original.Check)
	return engine, discovery, err
}
func nodeWireField(name string) string {
	for _, f := range listRecordProfile().Body {
		if f.Name == name {
			return f.Wire
		}
	}
	return name
}
func nodeCSVFields(raw json.RawMessage) (json.RawMessage, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return bytes.Clone(raw), nil
	}
	var names []string
	if trim := bytes.TrimSpace(raw); len(trim) > 0 && trim[0] == '"' {
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return nil, err
		}
		for _, r := range text {
			names = append(names, string(r))
		}
	} else {
		if err := json.Unmarshal(raw, &names); err != nil {
			return nil, listRecordInvalid("fields must be a JSON string/list of strings or null")
		}
	}
	for i, name := range names {
		names[i] = nodeWireField(name)
	}
	return json.Marshal(strings.Join(names, ","))
}
func normalizeNodeListPolicy(p *RecordListOpts) error {
	detailed := false
	if raw, present := p.Query["details"]; present {
		var err error
		detailed, err = cloudfilter.PythonJSONTruthy(raw)
		if err != nil {
			return err
		}
		delete(p.Query, "details")
		p.order = slices.DeleteFunc(p.order, func(s string) bool { return s == "details" })
	}
	if p.BasePath == nil || *p.BasePath == "" {
		path := "/nodes"
		if detailed {
			path += "/detail"
		}
		p.BasePath = &path
	}
	if raw, present := p.Query["fields"]; present {
		converted, err := nodeCSVFields(raw)
		if err != nil {
			return err
		}
		p.Query["fields"] = converted
	}
	return nil
}

// WithRecordListDetails selects /nodes/detail only when base_path is falsey.
func WithRecordListDetails(v bool) RecordListOption { return WithRecordListQuery("details", v) }

// RecordGetOpts owns get_node's directly declared field selection and transport.
type RecordGetOpts struct {
	Fields       []string
	Headers      map[string]string
	Microversion *string
}
type RecordGetOption func(*RecordGetOpts) error

func copyNodeGet(v RecordGetOpts) RecordGetOpts {
	v.Fields = slices.Clone(v.Fields)
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithRecordGetOpts(v RecordGetOpts) RecordGetOption {
	owned := copyNodeGet(v)
	return func(c *RecordGetOpts) error { *c = copyNodeGet(owned); return nil }
}
func WithRecordGetFields(v ...string) RecordGetOption {
	owned := slices.Clone(v)
	return func(c *RecordGetOpts) error { c.Fields = slices.Clone(owned); return nil }
}
func WithRecordGetHeaders(v map[string]string) RecordGetOption {
	owned := maps.Clone(v)
	return func(c *RecordGetOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordGetMicroversion(v string) RecordGetOption {
	return func(c *RecordGetOpts) error { x := v; c.Microversion = &x; return nil }
}
func (a *API) GetRecord(ctx context.Context, identity string, options ...RecordGetOption) (*NodeRecord, error) {
	wrap := func(err error) error { return request.Wrap("GetRecord", "nodes", cloudread.ContextError(ctx, err)) }
	original, err := a.listRecordRecordEngine(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	client := nodeReadClientSnapshot(a)
	policy := RecordGetOpts{}
	for _, apply := range slices.Clone(options) {
		if err := original.Check(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(listRecordInvalid("nil get option"))
		}
		next := copyNodeGet(policy)
		if err := errors.Join(apply(&next), original.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyNodeGet(next)
	}
	if err := nodeReadIdentity(identity); err != nil {
		return nil, wrap(err)
	}
	engine, discovery, err := a.versionedReadEngine(ctx, original, policy.Microversion, client, policy.Headers)
	if err != nil {
		return &NodeRecord{Discovery: cloneNodeDiscovery(discovery)}, wrap(err)
	}
	call := neutronfind.Options{Headers: policy.Headers}
	if len(policy.Fields) > 0 {
		fields := slices.Clone(policy.Fields)
		for i, name := range fields {
			fields[i] = nodeWireField(name)
		}
		raw, _ := json.Marshal(strings.Join(fields, ","))
		call.Query = map[string]json.RawMessage{"fields": raw}
		call.Order = []string{"fields"}
	}
	value, err := engine.Get(ctx, identity, call)
	result := listRecordListRecord(value)
	if result != nil {
		result.Discovery = cloneNodeDiscovery(discovery)
	}
	return result, wrap(err)
}

// RecordFindOpts owns find_node's default detailed fallback and logical absence.
type RecordFindOpts struct {
	Details       *bool
	IgnoreMissing *bool
	Headers       map[string]string
	Microversion  *string
}
type RecordFindOption func(*RecordFindOpts) error

func copyNodeFind(v RecordFindOpts) RecordFindOpts {
	v.Headers = maps.Clone(v.Headers)
	if v.Details != nil {
		x := *v.Details
		v.Details = &x
	}
	if v.IgnoreMissing != nil {
		x := *v.IgnoreMissing
		v.IgnoreMissing = &x
	}
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithRecordFindOpts(v RecordFindOpts) RecordFindOption {
	owned := copyNodeFind(v)
	return func(c *RecordFindOpts) error { *c = copyNodeFind(owned); return nil }
}
func WithRecordFindDetails(v bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.Details = &x; return nil }
}
func WithRecordFindIgnoreMissing(v bool) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}
func WithRecordFindHeaders(v map[string]string) RecordFindOption {
	owned := maps.Clone(v)
	return func(c *RecordFindOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithRecordFindMicroversion(v string) RecordFindOption {
	return func(c *RecordFindOpts) error { x := v; c.Microversion = &x; return nil }
}
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*NodeRecord, error) {
	wrap := func(err error) error { return request.Wrap("FindRecord", "nodes", cloudread.ContextError(ctx, err)) }
	original, err := a.listRecordRecordEngine(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	client := nodeReadClientSnapshot(a)
	policy := RecordFindOpts{}
	for _, apply := range slices.Clone(options) {
		if err := original.Check(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(listRecordInvalid("nil find option"))
		}
		next := copyNodeFind(policy)
		if err := errors.Join(apply(&next), original.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyNodeFind(next)
	}
	if err := nodeReadIdentity(identity); err != nil {
		return nil, wrap(err)
	}
	engine, discovery, err := a.versionedReadEngine(ctx, original, policy.Microversion, client, policy.Headers)
	if err != nil {
		return &NodeRecord{Discovery: cloneNodeDiscovery(discovery)}, wrap(err)
	}
	path := "/nodes/detail"
	if policy.Details != nil && !*policy.Details {
		path = "/nodes"
	}
	value, err := engine.Find(ctx, identity, neutronfind.Options{ListBasePath: &path, IgnoreMissing: policy.IgnoreMissing, Headers: policy.Headers})
	result := listRecordListRecord(value)
	if result != nil {
		result.Discovery = cloneNodeDiscovery(discovery)
	}
	return result, wrap(err)
}

func nodeReadIdentity(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) || strings.IndexFunc(id, unicode.IsControl) >= 0 {
		return listRecordInvalid("nonblank literal Node name/ID without controls or dot segments is required")
	}
	return nil
}
