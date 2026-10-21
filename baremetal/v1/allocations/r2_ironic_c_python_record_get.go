package allocations

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// AllocationRecord separates the declared Source view from actual wire and HTTP evidence.
type AllocationRecord struct {
	Resource, Wire *resource.RawResource
	Envelope       json.RawMessage
	Header         http.Header
	StatusCode     int
	Discovery      microversions.Discovery
}

func (r *AllocationRecord) Clone() *AllocationRecord {
	if r == nil {
		return nil
	}
	x := *r
	x.Resource = r.Resource.Clone()
	x.Wire = r.Wire.Clone()
	x.Envelope = bytes.Clone(r.Envelope)
	x.Header = r.Header.Clone()
	x.Discovery = cloneDiscovery(r.Discovery)
	return &x
}

func listRecordInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// Profile generated from the pinned Allocation direct Body declarations and Resource MRO.
func listRecordProfile() neutronfind.Profile {
	return neutronfind.Profile{
		ServiceRole: "baremetal", Kind: "allocations", Path: "allocations", Plural: "allocations", IDKey: "id", NameKey: "name", VersionedPath: "/v1/allocations",
		SourceFindPolicy: true, SourceStatusPolicy: true, SourceLinkPolicy: true, StopFaultRetryCallbacks: true, PythonLoadedEquality: true,
		Body:  []neutronfind.Descriptor{{Name: "candidate_nodes", Wire: "candidate_nodes", Missing: "null", Kind: neutronfind.List}, {Name: "created_at", Wire: "created_at", Missing: "null"}, {Name: "extra", Wire: "extra", Missing: "null", Kind: neutronfind.Dictionary}, {Name: "id", Wire: "uuid", Missing: "null"}, {Name: "last_error", Wire: "last_error", Missing: "null"}, {Name: "links", Wire: "links", Missing: "null", Kind: neutronfind.List}, {Name: "name", Wire: "name", Missing: "null"}, {Name: "node", Wire: "node", Missing: "null"}, {Name: "node_id", Wire: "node_uuid", Missing: "null"}, {Name: "owner", Wire: "owner", Missing: "null"}, {Name: "resource_class", Wire: "resource_class", Missing: "null"}, {Name: "state", Wire: "state", Missing: "null"}, {Name: "traits", Wire: "traits", Missing: "null", Kind: neutronfind.List}, {Name: "updated_at", Wire: "updated_at", Missing: "null"}},
		Query: []neutronfind.QueryPair{{Name: "node", Wire: "node"}, {Name: "resource_class", Wire: "resource_class"}, {Name: "state", Wire: "state"}, {Name: "owner", Wire: "owner"}, {Name: "fields", Wire: "fields"}, {Name: "limit", Wire: "limit"}, {Name: "marker", Wire: "marker"}},
	}
}
func cloneDiscovery(v microversions.Discovery) microversions.Discovery {
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
func readClientSnapshot(a *API) gophercloud.ServiceClient {
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
			version, err = microversions.SelectVersion(discovery.Maximum, discovery.Minimum, "1.60")
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
func wireField(name string) string {
	for _, f := range listRecordProfile().Body {
		if f.Name == name {
			return f.Wire
		}
	}
	return name
}

// RecordGetOpts owns get_allocation's directly declared field selection and transport.
type RecordGetOpts struct {
	Fields       []string
	Headers      map[string]string
	Microversion *string
}
type RecordGetOption func(*RecordGetOpts) error

func copyGet(v RecordGetOpts) RecordGetOpts {
	v.Fields = slices.Clone(v.Fields)
	v.Headers = maps.Clone(v.Headers)
	if v.Microversion != nil {
		x := *v.Microversion
		v.Microversion = &x
	}
	return v
}
func WithRecordGetOpts(v RecordGetOpts) RecordGetOption {
	owned := copyGet(v)
	return func(c *RecordGetOpts) error { *c = copyGet(owned); return nil }
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
func (a *API) GetRecord(ctx context.Context, identity string, options ...RecordGetOption) (*AllocationRecord, error) {
	wrap := func(err error) error {
		return request.Wrap("GetRecord", "allocations", cloudread.ContextError(ctx, err))
	}
	original, err := a.listRecordRecordEngine(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	client := readClientSnapshot(a)
	policy := RecordGetOpts{}
	for _, apply := range slices.Clone(options) {
		if err := original.Check(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(listRecordInvalid("nil get option"))
		}
		next := copyGet(policy)
		if err := errors.Join(apply(&next), original.Check(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyGet(next)
	}
	if err := readIdentity(identity); err != nil {
		return nil, wrap(err)
	}
	engine, discovery, err := a.versionedReadEngine(ctx, original, policy.Microversion, client, policy.Headers)
	if err != nil {
		return &AllocationRecord{Discovery: cloneDiscovery(discovery)}, wrap(err)
	}
	call := neutronfind.Options{Headers: policy.Headers}
	if len(policy.Fields) > 0 {
		fields := slices.Clone(policy.Fields)
		for i, name := range fields {
			fields[i] = wireField(name)
		}
		raw, _ := json.Marshal(strings.Join(fields, ","))
		call.Query = map[string]json.RawMessage{"fields": raw}
		call.Order = []string{"fields"}
	}
	value, err := engine.Get(ctx, identity, call)
	result := listRecordListRecord(value)
	if result != nil {
		result.Discovery = cloneDiscovery(discovery)
	}
	return result, wrap(err)
}

func readIdentity(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) || strings.IndexFunc(id, unicode.IsControl) >= 0 {
		return listRecordInvalid("nonblank literal Allocation identity without controls or dot segments is required")
	}
	return nil
}
func (a *API) listRecordRecordEngine(ctx context.Context) (*neutronfind.Engine, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, listRecordInvalid("configured Allocation API required")
	}
	selected := a.client
	return neutronfind.New(ctx, selected, listRecordProfile(), func(context.Context) error {
		if a.client != selected {
			return listRecordInvalid("selected Allocation API client changed")
		}
		return nil
	})
}

func listRecordListRecord(value *neutronfind.Record) *AllocationRecord {
	if value == nil {
		return nil
	}
	return &AllocationRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode}
}
