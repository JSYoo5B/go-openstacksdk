package consistencygroupsnapshots_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

// Binding receipt logger around the shared public-testhelper fixture; no new HTTP server harness.
type snapshotCall struct {
	method, path, escaped, query, body string
	headers                            http.Header
}
type snapshotTransport func(*http.Request) (*http.Response, error)

func (f snapshotTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := f(r)
	if response != nil {
		response.Request = r
	}
	return response, err
}
func snapshotWire(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"actual-consistency-group-snapshot"}}, Body: io.NopCloser(strings.NewReader(body))}
}

const snapshotBase = "/cinder/v3/project/cgsnapshots"

func snapshotAPI(t *testing.T, version string, reply func(*http.Request) *http.Response) (*consistencygroupsnapshots.API, *[]snapshotCall) {
	t.Helper()
	fake := testcloud.New(t)
	calls := &[]snapshotCall{}
	fake.Provider.HTTPClient.Transport = snapshotTransport(func(r *http.Request) (*http.Response, error) {
		body := ""
		if r.Body != nil {
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			body = string(raw)
		}
		*calls = append(*calls, snapshotCall{r.Method, r.URL.Path, r.URL.EscapedPath(), r.URL.RawQuery, body, r.Header.Clone()})
		return reply(r), nil
	})
	client := fake.Client("block-storage", "/cinder/v3/")
	client.ResourceBase = fake.Server.URL + "/cinder/v3/project/"
	client.Microversion = version
	return consistencygroupsnapshots.New(client), calls
}
func snapshotField(t *testing.T, value *consistencygroupsnapshots.ConsistencyGroupSnapshotRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatal(key, want, value)
	}
}
func snapshotsCollect(sequence func(func(*consistencygroupsnapshots.ConsistencyGroupSnapshotRecord, error) bool)) ([]*consistencygroupsnapshots.ConsistencyGroupSnapshotRecord, error) {
	var values []*consistencygroupsnapshots.ConsistencyGroupSnapshotRecord
	for value, err := range sequence {
		if err != nil {
			return values, err
		}
		values = append(values, value)
	}
	return values, nil
}

func TestPythonConsistencyGroupSnapshotGetOwnsUntypedDescriptorsEnvelopeAndLocation(t *testing.T) {
	row := `{"id":9007199254740993,"name":false,"created_at":[null,0],"description":{"enabled":false},"consistencygroup_id":0,"project_id":"other","status":"available","metadata":{"unknown":true},"self":"wire","connection":false}`
	for _, wrapper := range []string{"cgsnapshot", "flat"} {
		body := row
		if wrapper != "flat" {
			body = `{"cgsnapshot":` + row + `}`
		}
		api, calls := snapshotAPI(t, "3.80", func(*http.Request) *http.Response { return snapshotWire(203, body) })
		cloud, project := "named-cloud", "current"
		location := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"current"`), Name: &project}}
		value, err := api.GetRecord(context.Background(), "literal /%雪", consistencygroupsnapshots.WithRecordReadLocation(location))
		if err != nil || len(*calls) != 1 || (*calls)[0].escaped != snapshotBase+"/literal%20%2F%25%E9%9B%AA" || (*calls)[0].query != "" || (*calls)[0].body != "" || (*calls)[0].headers.Get("OpenStack-API-Version") != "volume 3.80" {
			t.Fatal(value, err, *calls)
		}
		for key, want := range map[string]string{"id": "9007199254740993", "name": "false", "created_at": "[null,0]", "description": `{"enabled":false}`, "consistencygroup_id": "0"} {
			snapshotField(t, value, key, want)
		}
		if len(value.Resource.Body) != 7 || value.Resource.Body["metadata"] != nil || value.Wire == nil || value.StatusCode != 203 || string(value.Envelope) != body {
			t.Fatal(value)
		}
		var found resource.CloudLocation
		if err := json.Unmarshal(value.Resource.Body["location"], &found); err != nil || found.Cloud == nil || *found.Cloud != "named-cloud" || found.Project.Name == nil || *found.Project.Name != "current" || string(found.Project.ID) != `"current"` {
			t.Fatal(found, err)
		}
		value.Resource.Body["id"][0] = '1'
		value.Header.Set("X-Request-Id", "changed")
		if string(value.Wire.Body["id"]) != "9007199254740993" || value.Wire.Header.Get("X-Request-Id") != "actual-consistency-group-snapshot" {
			t.Fatal("carriers alias", value)
		}
	}
}
func TestPythonConsistencyGroupSnapshotGetOpaqueSeedNullAndInvalidShapeProof(t *testing.T) {
	seed := &consistencygroupsnapshots.ConsistencyGroupSnapshotRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"seed"`), "name": json.RawMessage(`false`), "description": json.RawMessage(`null`)}}}}
	option := consistencygroupsnapshots.WithRecordReadSeed(seed)
	seed.Resource.Body["name"] = json.RawMessage(`true`)
	for _, body := range []string{"", "not JSON", `{}`, `{"cgsnapshot":{"id":null}}`, `null`, `[]`, `{"cgsnapshot":null}`} {
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(204, body) })
		value, err := api.GetRecord(context.Background(), "selected", option)
		invalid := body == `null` || body == `[]` || body == `{"cgsnapshot":null}`
		if value == nil || len(*calls) != 1 || (err != nil) != invalid || string(value.Envelope) != body {
			t.Fatal(body, value, err, *calls)
		}
		if invalid {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 204 || string(proof.Body) != body {
				t.Fatal(proof, err)
			}
		} else {
			snapshotField(t, value, "name", "false")
			if strings.Contains(body, `"id":null`) {
				snapshotField(t, value, "id", "null")
			} else {
				snapshotField(t, value, "id", `"selected"`)
			}
		}
	}
}

type snapshotMarshal func() ([]byte, error)

func (f snapshotMarshal) MarshalJSON() ([]byte, error) { return f() }
func TestPythonConsistencyGroupSnapshotCreateDeclaredDirtyAttrsAndImmutableOptions(t *testing.T) {
	attrs := map[string]any{"id": false, "name": "before", "description": map[string]any{"null": nil, "zero": 0, "enabled": false}, "consistencygroup_id": json.Number("9007199254740993"), "metadata": snapshotMarshal(func() ([]byte, error) { t.Fatal("unknown attr was marshaled"); return nil, nil })}
	headers := map[string]string{"X-Owned": "original"}
	option := consistencygroupsnapshots.WithRecordCreateHeaders(headers)
	headers["X-Owned"] = "mutated"
	api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
		return snapshotWire(202, `{"cgsnapshot":{"id":"server","name":null,"status":"creating","unknown":9007199254740993}}`)
	})
	var retained *consistencygroupsnapshots.RecordCreateOpts
	value, err := api.CreateRecord(context.Background(), attrs, option, func(cfg *consistencygroupsnapshots.RecordCreateOpts) error {
		attrs["name"] = "after"
		attrs["description"].(map[string]any)["zero"] = 99
		cfg.Headers["X-Next"] = "first"
		retained = cfg
		return nil
	}, func(*consistencygroupsnapshots.RecordCreateOpts) error {
		retained.Headers["X-Next"] = "after"
		return nil
	})
	if err != nil || len(*calls) != 1 || (*calls)[0].method != "POST" || (*calls)[0].path != snapshotBase || (*calls)[0].headers.Get("X-Owned") != "original" || (*calls)[0].headers.Get("X-Next") != "first" || (*calls)[0].headers.Get("Content-Type") != "application/json" {
		t.Fatal(value, err, *calls)
	}
	var body map[string]map[string]json.RawMessage
	if err := json.Unmarshal([]byte((*calls)[0].body), &body); err != nil || string(body["cgsnapshot"]["id"]) != "false" || string(body["cgsnapshot"]["name"]) != `"before"` || string(body["cgsnapshot"]["consistencygroup_id"]) != "9007199254740993" || body["cgsnapshot"]["metadata"] != nil {
		t.Fatal(body, err)
	}
	snapshotField(t, value, "id", `"server"`)
	snapshotField(t, value, "name", "null")
	snapshotField(t, value, "description", `{"enabled":false,"null":null,"zero":0}`)
	if value.Wire == nil || string(value.Wire.Body["unknown"]) != "9007199254740993" {
		t.Fatal(value)
	}
	for _, attrs := range []map[string]any{nil, {}, {"id": nil, "name": "", "description": false}} {
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(204, "opaque") })
		value, err := api.CreateRecord(context.Background(), attrs)
		if err != nil || len(*calls) != 1 || value.Resource == nil || string(value.Envelope) != "opaque" {
			t.Fatal(value, err, *calls)
		}
		if len(attrs) == 0 && (*calls)[0].body != `{"cgsnapshot":{}}` {
			t.Fatal(*calls)
		}
	}
}
func TestPythonConsistencyGroupSnapshotCreateSourceCaptureMarshalAndConstructorCollisions(t *testing.T) {
	for _, mode := range []string{"marshal source", "marshal cancel", "option source", "option cancel", "connection", "_synchronized", "self"} {
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { t.Fatal("preflight reached HTTP"); return snapshotWire(200, `{}`) })
		ctx, cancel := context.WithCancelCause(context.Background())
		cause := errors.New("callback cause")
		attrs := map[string]any{"consistencygroup_id": "group"}
		opts := []consistencygroupsnapshots.RecordCreateOption{}
		switch mode {
		case "marshal source":
			attrs["name"] = snapshotMarshal(func() ([]byte, error) { api.RawClient().ResourceBase = "https://changed.test/v3/"; return nil, cause })
		case "marshal cancel":
			attrs["name"] = snapshotMarshal(func() ([]byte, error) { cancel(cause); return nil, cause })
		case "option source":
			opts = append(opts, func(*consistencygroupsnapshots.RecordCreateOpts) error {
				api.RawClient().Microversion = "3.80"
				return cause
			})
		case "option cancel":
			opts = append(opts, func(*consistencygroupsnapshots.RecordCreateOpts) error { cancel(cause); return cause })
		default:
			attrs[mode] = nil
		}
		value, err := api.CreateRecord(ctx, attrs, opts...)
		if value != nil || err == nil || len(*calls) != 0 {
			t.Fatal(mode, value, err, *calls)
		}
		if strings.Contains(mode, "source") && (!errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption)) {
			t.Fatal(err)
		}
		if strings.Contains(mode, "cancel") && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
			t.Fatal(err)
		}
		cancel(nil)
	}
}
func TestPythonConsistencyGroupSnapshotListSourceQueriesDetailsFiltersAndRawCap(t *testing.T) {
	for _, details := range []bool{true, false} {
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
			return snapshotWire(200, `{"cgsnapshots":[{"id":"first","name":"data","description":{"enabled":true,"extra":0}},{"id":"second","name":"data","description":{"enabled":false}}]}`)
		})
		options := []consistencygroupsnapshots.RecordListOption{consistencygroupsnapshots.WithRecordListQuery("all_projects", snapshotMarshal(func() ([]byte, error) { t.Fatal("unmapped all_projects was marshaled"); return nil, nil })), consistencygroupsnapshots.WithRecordListQuery("sort_dir", []string{"asc", "desc"}), consistencygroupsnapshots.WithRecordListQuery("sort", "name:asc"), consistencygroupsnapshots.WithRecordListQuery("name", "data"), consistencygroupsnapshots.WithRecordListQuery("description", map[string]any{"enabled": 1}), consistencygroupsnapshots.WithRecordListQuery("unknown", snapshotMarshal(func() ([]byte, error) { t.Fatal("unknown query marshaled"); return nil, nil }))}
		if !details {
			options = append(options, consistencygroupsnapshots.WithRecordListDetails(false))
		}
		sequence := api.ListRecords(context.Background(), options...)
		if len(*calls) != 0 {
			t.Fatal("list was eager")
		}
		values, err := snapshotsCollect(sequence)
		if err != nil || len(values) != 1 || len(*calls) != 1 || (*calls)[0].headers.Get("Accept") != "application/json" {
			t.Fatal(values, err, *calls)
		}
		want := snapshotBase + "/detail"
		if !details {
			want = snapshotBase
		}
		query, _ := url.ParseQuery((*calls)[0].query)
		if (*calls)[0].path != want || query.Has("all_tenants") || query.Has("all_projects") || query.Get("sort") != "name:asc" || !reflect.DeepEqual(query["sort_dir"], []string{"asc", "desc"}) || query.Has("name") || query.Has("description") || query.Has("unknown") {
			t.Fatal(*calls)
		}
		snapshotField(t, values[0], "id", `"first"`)
		values, err = snapshotsCollect(api.ListRecords(context.Background(), consistencygroupsnapshots.WithRecordListMaxItems(1), consistencygroupsnapshots.WithRecordListQuery("id", "second")))
		if err != nil || len(values) != 0 || len(*calls) != 2 || !strings.Contains((*calls)[1].query, "limit=1") {
			t.Fatal(values, err, *calls)
		}
	}
}
func TestPythonConsistencyGroupSnapshotListLocalFloatPolicyAndConstructorCollisions(t *testing.T) {
	api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
		return snapshotWire(200, `{"cgsnapshots":[{"id":"v","consistencygroup_id":9007199254740992}]}`)
	})
	values, err := snapshotsCollect(api.ListRecords(context.Background(), consistencygroupsnapshots.WithRecordListQuery("consistencygroup_id", json.RawMessage(`9007199254740993.0`))))
	if err != nil || len(values) != 0 || len(*calls) != 1 {
		t.Fatal("exact-decimal filter policy changed", values, err, *calls)
	}
	for _, key := range []string{"connection", "microversion", "_synchronized"} {
		body := `{"cgsnapshots":[{"id":"v","` + key + `":null}]}`
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(200, body) })
		values, err := snapshotsCollect(api.ListRecords(context.Background()))
		var proof *resource.ResponseError
		if len(values) != 0 || err == nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || len(*calls) != 1 {
			t.Fatal(key, values, err, proof, *calls)
		}
	}
	for _, description := range []string{`"truthy"`, `false`, `{}`, `{"enabled":true}`} {
		body := `{"cgsnapshots":[{"id":"v","description":` + description + `}]}`
		api, _ := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(200, body) })
		values, err := snapshotsCollect(api.ListRecords(context.Background(), consistencygroupsnapshots.WithRecordListQuery("description", map[string]any{})))
		want := description == `"truthy"` || description == `{"enabled":true}`
		if err != nil || (len(values) == 1) != want {
			t.Fatal(description, values, err)
		}
	}
}
func TestPythonConsistencyGroupSnapshotListPagingUsesRetainedScalarMarkerAndScope(t *testing.T) {
	api, calls := snapshotAPI(t, "3.29", func(r *http.Request) *http.Response {
		if r.URL.Query().Get("marker") == "9007199254740993" {
			return snapshotWire(200, `{"cgsnapshots":[]}`)
		}
		return snapshotWire(200, `{"cgsnapshots":[{"id":9007199254740993,"name":"kept"}]}`)
	})
	var values []*consistencygroupsnapshots.ConsistencyGroupSnapshotRecord
	for value, err := range api.ListRecords(context.Background(), consistencygroupsnapshots.WithRecordListLimit(2)) {
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, value)
		value.Resource.Body["id"] = json.RawMessage(`"mutated"`)
		value.Wire.Body["id"] = json.RawMessage(`"also-mutated"`)
	}
	if len(values) != 1 || len(*calls) != 2 || (*calls)[1].query != "limit=2&marker=9007199254740993" {
		t.Fatal(values, *calls)
	}
	api, calls = snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
		return snapshotWire(200, `{"cgsnapshots":[{"id":"v"}],"cgsnapshots_links":[{"rel":"next","href":"?marker=unvisited"}]}`)
	})
	values, err := snapshotsCollect(api.ListRecords(context.Background(), consistencygroupsnapshots.WithRecordListPaginated(false)))
	if err != nil || len(values) != 1 || len(*calls) != 1 {
		t.Fatal(values, err, *calls)
	}
	api, _ = snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
		return snapshotWire(200, `{"cgsnapshots":[{"id":"v"}],"next":"https://changed.test/cgsnapshots/detail?marker=unsafe"}`)
	})
	_, err = snapshotsCollect(api.ListRecords(context.Background()))
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
func TestPythonConsistencyGroupSnapshotNoAutomaticDiscoveryAndUncappedSelectedVersion(t *testing.T) {
	for _, selected := range []string{"", "3.0", "3.71", "3.80"} {
		api, calls := snapshotAPI(t, selected, func(r *http.Request) *http.Response {
			if r.URL.Path != snapshotBase+"/v" {
				t.Fatal("unexpected discovery/member route", r.URL)
			}
			return snapshotWire(200, `{"cgsnapshot":{"id":"v"}}`)
		})
		value, err := api.GetRecord(context.Background(), "v")
		generic := ""
		if selected != "" {
			generic = "volume " + selected
		}
		if err != nil || value == nil || len(*calls) != 1 || (*calls)[0].headers.Get("OpenStack-API-Version") != generic || (*calls)[0].headers.Get("X-OpenStack-Volume-API-Version") != selected || api.RawClient().Microversion != selected {
			t.Fatal(selected, value, err, *calls)
		}
	}
	for _, override := range []string{"", "3.80"} {
		api, calls := snapshotAPI(t, "3.71", func(*http.Request) *http.Response { return snapshotWire(200, `{}`) })
		api.RawClient().MoreHeaders = map[string]string{"openstack-api-version": "volume 3.71", "x-openstack-volume-api-version": "3.71"}
		value, err := api.GetRecord(context.Background(), "v", consistencygroupsnapshots.WithRecordReadMicroversion(override))
		generic := ""
		if override != "" {
			generic = "volume " + override
		}
		if err != nil || value == nil || len(*calls) != 1 || (*calls)[0].headers.Get("OpenStack-API-Version") != generic || (*calls)[0].headers.Get("X-OpenStack-Volume-API-Version") != override || api.RawClient().Microversion != "3.71" {
			t.Fatal(override, value, err, *calls)
		}
	}
}
func TestPythonConsistencyGroupSnapshotDeleteDefaultMissingStrictAndOpaqueACK(t *testing.T) {
	for _, status := range []int{199, 200, 202, 204, 299, 399, 404, 403, 409, 503} {
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(status, "opaque ACK") })
		value, err := api.DeleteRecord(context.Background(), "v")
		wantError := status >= 400 && status != 404
		if value == nil || (err != nil) != wantError || value.StatusCode != status || value.IgnoredMissing != (status == 404) || string(value.Envelope) != "opaque ACK" || len(*calls) != 1 || (*calls)[0].method != "DELETE" || (*calls)[0].body != "" {
			t.Fatal(status, value, err, *calls)
		}
		if wantError && !gophercloud.ResponseCodeIs(err, status) {
			t.Fatal(err)
		}
	}
	api, _ := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(404, "missing") })
	value, err := api.DeleteRecord(context.Background(), "v", consistencygroupsnapshots.WithRecordDeleteIgnoreMissing(false))
	if value == nil || value.IgnoredMissing || !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatal(value, err)
	}
}

// Failure injection only: shared REST owns all body reading, closing and retries.
type snapshotFaultBody struct {
	data            []byte
	read, close     error
	onRead, onClose func()
	closes          *atomic.Int32
}

func (b *snapshotFaultBody) Read(p []byte) (int, error) {
	if len(b.data) > 0 {
		n := copy(p, b.data)
		b.data = b.data[n:]
		if b.onRead != nil {
			b.onRead()
			b.onRead = nil
		}
		return n, nil
	}
	if b.read != nil {
		return 0, b.read
	}
	return 0, io.EOF
}
func (b *snapshotFaultBody) Close() error {
	if b.closes != nil {
		b.closes.Add(1)
	}
	if b.onClose != nil {
		b.onClose()
	}
	return b.close
}
func TestPythonConsistencyGroupSnapshotPhysicalFaultsNeverReplayIgnoreOrInventSuccess(t *testing.T) {
	for _, operation := range []string{"get", "list", "create", "delete"} {
		for _, status := range []int{203, 404, 503} {
			t.Run(fmt.Sprintf("%s/%d", operation, status), func(t *testing.T) {
				readCause, closeCause, cancelCause := errors.New("read"), errors.New("close"), errors.New("cancel")
				ctx, cancel := context.WithCancelCause(context.Background())
				var client *gophercloud.ServiceClient
				var closes atomic.Int32
				original := ""
				body := `{"cgsnapshot":{"id":"v"}}`
				if operation == "list" {
					body = `{"cgsnapshots":[{"id":"v"}]}`
				}
				api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
					response := snapshotWire(status, body)
					response.Body = &snapshotFaultBody{data: []byte(body), read: readCause, close: closeCause, closes: &closes, onRead: func() { client.ResourceBase = original + "changed/" }, onClose: func() { client.ResourceBase = original; cancel(cancelCause) }}
					return response
				})
				client = api.RawClient()
				original = client.ResourceBase
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return nil }
				var err error
				switch operation {
				case "get":
					value, e := api.GetRecord(ctx, "v")
					err = e
					if value == nil || value.Resource != nil {
						t.Fatal(value, e)
					}
				case "create":
					value, e := api.CreateRecord(ctx, map[string]any{"consistencygroup_id": "g"})
					err = e
					if value == nil || value.Resource != nil {
						t.Fatal(value, e)
					}
				case "delete":
					value, e := api.DeleteRecord(ctx, "v")
					err = e
					if value == nil || value.IgnoredMissing {
						t.Fatal(value, e)
					}
				case "list":
					values, e := snapshotsCollect(api.ListRecords(ctx))
					err = e
					if len(values) != 0 {
						t.Fatal(values, e)
					}
				}
				if len(*calls) != 1 || closes.Load() != 1 {
					t.Fatal(*calls, closes.Load(), err)
				}
				for _, cause := range []error{readCause, closeCause, cancelCause, context.Canceled, resource.ErrInvalidOption} {
					if !errors.Is(err, cause) {
						t.Fatal(cause, err)
					}
				}
				if status < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body {
						t.Fatal(proof, err)
					}
				} else if !gophercloud.ResponseCodeIs(err, status) {
					t.Fatal(err)
				}
				cancel(nil)
			})
		}
	}
}
func TestPythonConsistencyGroupSnapshotRetryOwnsJSONHeadersAndSelectedVersion(t *testing.T) {
	for _, mode := range []string{"clean", "body", "version", "header", "denied"} {
		attrs := map[string]any{"consistencygroup_id": "g", "name": "before"}
		count := 0
		cause := errors.New("retry denied")
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
			count++
			if count == 1 {
				return snapshotWire(503, "first")
			}
			return snapshotWire(202, `{"cgsnapshot":{"id":"v"}}`)
		})
		api.RawClient().MoreHeaders = map[string]string{"Openstack-Api-Version": "volume 3.29", "X-Openstack-Volume-Api-Version": "3.29"}
		api.RawClient().RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
			attrs["name"] = "after"
			switch mode {
			case "body":
				opts.JSONBody = json.RawMessage(`{"cgsnapshot":{}}`)
			case "version":
				opts.MoreHeaders["OpenStack-API-Version"] = "volume 3.80"
			case "header":
				opts.MoreHeaders["X-Owned"] = "changed"
			case "denied":
				return cause
			}
			return nil
		}
		value, err := api.CreateRecord(context.Background(), attrs, consistencygroupsnapshots.WithRecordCreateHeaders(map[string]string{"X-Owned": "fixed"}))
		if mode == "clean" {
			if err != nil || len(*calls) != 2 || (*calls)[0].body != (*calls)[1].body {
				t.Fatal(value, err, *calls)
			}
			snapshotField(t, value, "name", `"before"`)
		} else {
			if len(*calls) != 1 || !gophercloud.ResponseCodeIs(err, 503) {
				t.Fatal(mode, value, err, *calls)
			}
			if mode == "denied" {
				if !errors.Is(err, cause) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		}
	}
}
func TestPythonConsistencyGroupSnapshotOptionsPreflightOwnershipAndNilAPI(t *testing.T) {
	api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
		t.Fatal("invalid option reached HTTP")
		return snapshotWire(200, `{}`)
	})
	for _, id := range []string{"", " ", ".", "..", "control\n", string([]byte{0xff})} {
		if _, err := api.GetRecord(context.Background(), id); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(id, err)
		}
	}
	for _, opts := range [][]consistencygroupsnapshots.RecordListOption{{nil}, {consistencygroupsnapshots.WithRecordListMaxItems(-1)}, {consistencygroupsnapshots.WithRecordListLimit(-1)}, {consistencygroupsnapshots.WithRecordListHeaders(map[string]string{"X-Auth-Token": "caller"})}} {
		_, err := snapshotsCollect(api.ListRecords(context.Background(), opts...))
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`not JSON`)}}
	if _, err := api.GetRecord(context.Background(), "v", consistencygroupsnapshots.WithRecordReadLocation(location)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.CreateRecord(context.Background(), nil, consistencygroupsnapshots.WithRecordCreateLocation(location)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := snapshotsCollect(api.ListRecords(context.Background(), consistencygroupsnapshots.WithRecordListLocation(location))); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if len(*calls) != 0 {
		t.Fatal(*calls)
	}
	var nilAPI *consistencygroupsnapshots.API
	if _, err := nilAPI.GetRecord(context.Background(), "v"); err == nil {
		t.Fatal("nil API accepted")
	}
}

func TestPythonConsistencyGroupSnapshotCreateJSONFramingPrecedesHTTP(t *testing.T) {
	for _, serviceHeader := range []bool{true, false} {
		api, calls := snapshotAPI(t, "", func(*http.Request) *http.Response {
			t.Fatal("invalid JSON framing reached POST")
			return snapshotWire(200, `{}`)
		})
		options := []consistencygroupsnapshots.RecordCreateOption{}
		if serviceHeader {
			api.RawClient().MoreHeaders = map[string]string{"content-type": "application/custom"}
		} else {
			options = append(options, consistencygroupsnapshots.WithRecordCreateHeaders(map[string]string{"CONTENT-TYPE": "application/custom"}))
		}
		value, err := api.CreateRecord(context.Background(), map[string]any{"consistencygroup_id": "g"}, options...)
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(serviceHeader, value, err, *calls)
		}
	}
}

func TestPythonConsistencyGroupSnapshotRawPagingInputPrecedesHTTP(t *testing.T) {
	for _, query := range []map[string]any{{"limit": -1}, {"limit": []int{1, 2}}, {"limit": false}, {"marker": " "}, {"marker": []string{"one", "two"}}, {"max_items": false}} {
		api, calls := snapshotAPI(t, "", func(*http.Request) *http.Response {
			t.Fatal("invalid pagination input reached list")
			return snapshotWire(200, `{}`)
		})
		options := []consistencygroupsnapshots.RecordListOption{}
		for key, value := range query {
			options = append(options, consistencygroupsnapshots.WithRecordListQuery(key, value))
		}
		values, err := snapshotsCollect(api.ListRecords(context.Background(), options...))
		if len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(query, values, err, *calls)
		}
	}
}

func TestPythonConsistencyGroupSnapshotBoundClsPresenceAndGetWireBoundary(t *testing.T) {
	for _, input := range []any{nil, false, 0, snapshotMarshal(func() ([]byte, error) { t.Fatal("colliding cls was marshaled"); return nil, nil })} {
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
			t.Fatal("bound cls input reached POST")
			return snapshotWire(200, `{}`)
		})
		value, err := api.CreateRecord(context.Background(), map[string]any{"consistencygroup_id": "g", "cls": input})
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(input, value, err, *calls)
		}
	}
	for _, raw := range []string{`null`, `false`, `0`} {
		body := `{"cgsnapshots":[{"id":"v","cls":` + raw + `}]}`
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(200, body) })
		values, err := snapshotsCollect(api.ListRecords(context.Background()))
		var proof *resource.ResponseError
		if len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || len(*calls) != 1 {
			t.Fatal(raw, values, err, proof, *calls)
		}
		body = `{"cgsnapshot":{"id":"v","cls":` + raw + `}}`
		api, calls = snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(200, body) })
		value, err := api.GetRecord(context.Background(), "v")
		if err != nil || value == nil || value.Wire == nil || string(value.Wire.Body["cls"]) != raw || value.Resource.Body["cls"] != nil || len(*calls) != 1 {
			t.Fatal(raw, value, err, *calls)
		}
	}
}
func TestPythonConsistencyGroupSnapshotListDefaultAcceptOwnsSourceAndExplicitOption(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		requests := 0
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
			requests++
			if requests == 1 {
				return snapshotWire(503, "first")
			}
			return snapshotWire(200, `{"cgsnapshots":[]}`)
		})
		api.RawClient().MoreHeaders = map[string]string{"Accept": "application/source", "X-Source": "retained"}
		api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return nil }
		options := []consistencygroupsnapshots.RecordListOption{}
		want := "application/json"
		if explicit {
			options = append(options, consistencygroupsnapshots.WithRecordListHeaders(map[string]string{"aCcEpT": "application/operation"}))
			want = "application/operation"
		}
		values, err := snapshotsCollect(api.ListRecords(context.Background(), options...))
		if err != nil || len(values) != 0 || len(*calls) != 2 || api.RawClient().MoreHeaders["Accept"] != "application/source" {
			t.Fatal(explicit, values, err, *calls, api.RawClient().MoreHeaders)
		}
		for _, call := range *calls {
			if call.headers.Get("Accept") != want || call.headers.Get("X-Source") != "retained" {
				t.Fatal(explicit, call, want)
			}
		}
	}
}
func TestPythonConsistencyGroupSnapshotListAdvertisedOffsetAndCursorScope(t *testing.T) {
	t.Run("advertised offset preserves initial defaults and local predicates", func(t *testing.T) {
		api, calls := snapshotAPI(t, "3.29", func(r *http.Request) *http.Response {
			switch r.URL.Query().Get("offset") {
			case "0":
				return snapshotWire(200, `{"cgsnapshots":[{"id":"skip","status":"creating"},{"id":"first","status":"available"}],"next":"?offset=2"}`)
			case "2":
				return snapshotWire(200, `{"cgsnapshots":[{"id":"last","status":"available"}],"next":"?offset=4"}`)
			case "4":
				return snapshotWire(200, `{"cgsnapshots":[]}`)
			default:
				t.Fatal(r.URL)
				return nil
			}
		})
		values, err := snapshotsCollect(api.ListRecords(context.Background(), consistencygroupsnapshots.WithRecordListLimit(2), consistencygroupsnapshots.WithRecordListQuery("offset", 0), consistencygroupsnapshots.WithRecordListQuery("sort", "name:asc"), consistencygroupsnapshots.WithRecordListQuery("status", "available")))
		if err != nil || len(values) != 2 || len(*calls) != 3 {
			t.Fatal(values, err, *calls)
		}
		snapshotField(t, values[0], "id", `"first"`)
		snapshotField(t, values[1], "id", `"last"`)
		for index, call := range *calls {
			q, _ := url.ParseQuery(call.query)
			if q.Get("offset") != fmt.Sprint(index*2) || q.Get("limit") != "2" || q.Has("all_tenants") || q.Has("all_projects") || q.Get("sort") != "name:asc" || q.Has("status") || q.Has("marker") {
				t.Fatal(call)
			}
		}
	})
	t.Run("cursor family switches retain filter and limit", func(t *testing.T) {
		api, calls := snapshotAPI(t, "3.29", func(r *http.Request) *http.Response {
			if r.URL.Query().Get("marker") == "start" {
				return snapshotWire(200, `{"cgsnapshots":[{"id":"first"}],"next":"?offset=2"}`)
			}
			if r.URL.Query().Get("offset") == "2" {
				return snapshotWire(200, `{"cgsnapshots":[{"id":"last"}],"next":"?marker=last"}`)
			}
			if r.URL.Query().Get("marker") == "last" {
				return snapshotWire(200, `{"cgsnapshots":[]}`)
			}
			t.Fatal(r.URL)
			return nil
		})
		values, err := snapshotsCollect(api.ListRecords(context.Background(), consistencygroupsnapshots.WithRecordListLimit(2), consistencygroupsnapshots.WithRecordListMarker("start")))
		if err != nil || len(values) != 2 || len(*calls) != 3 {
			t.Fatal(values, err, *calls)
		}
		q, _ := url.ParseQuery((*calls)[1].query)
		if q.Get("offset") != "2" || q.Has("marker") || q.Get("limit") != "2" || q.Has("all_tenants") || q.Has("all_projects") {
			t.Fatal(q)
		}
		q, _ = url.ParseQuery((*calls)[2].query)
		if q.Get("marker") != "last" || q.Has("offset") || q.Get("limit") != "2" || q.Has("all_tenants") || q.Has("all_projects") {
			t.Fatal(q)
		}
	})
	for _, test := range []struct {
		name, initial, next string
		cycle               bool
	}{
		{"offset cycle", "0", "?offset=0", true}, {"offset backwards", "5", "?offset=3", false}, {"negative offset", "0", "?offset=-1", false}, {"two cursor families", "0", "?offset=2&marker=a", false}, {"filter replacement", "0", "?offset=2&sort=changed", false}, {"limit replacement", "0", "?offset=2&limit=3", false}, {"foreign route", "0", "https://foreign.invalid/cgsnapshots/detail?offset=2", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := `{"cgsnapshots":[{"id":"first"}],"next":` + fmt.Sprintf("%q", test.next) + `}`
			api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(200, body) })
			values, err := snapshotsCollect(api.ListRecords(context.Background(), consistencygroupsnapshots.WithRecordListLimit(2), consistencygroupsnapshots.WithRecordListQuery("offset", test.initial), consistencygroupsnapshots.WithRecordListQuery("sort", "name:asc")))
			var proof *resource.ResponseError
			var cycle *resource.PaginationCycleError
			if len(values) != 1 || err == nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || len(*calls) != 1 || (test.cycle && !errors.As(err, &cycle)) || (!test.cycle && !errors.Is(err, resource.ErrInvalidOption)) {
				t.Fatal(test, values, err, proof, *calls)
			}
		})
	}
}

func TestPythonConsistencyGroupSnapshotClassDefaultsUseNoDiscoveryForAllOperations(t *testing.T) {
	for _, operation := range []string{"get", "find", "list", "create", "delete"} {
		t.Run(operation, func(t *testing.T) {
			api, calls := snapshotAPI(t, "", func(r *http.Request) *http.Response {
				if !strings.HasPrefix(r.URL.Path, snapshotBase) {
					t.Fatal("class with maxNone attempted discovery", r.URL)
				}
				if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/detail") {
					return snapshotWire(200, `{"cgsnapshots":[]}`)
				}
				if operation == "find" && strings.HasSuffix(r.URL.Path, "/v") {
					return snapshotWire(404, "missing")
				}
				if r.Method == http.MethodDelete {
					return snapshotWire(202, "actual delete acknowledgement")
				}
				return snapshotWire(202, `{"cgsnapshot":{"id":"v"}}`)
			})
			var err error
			switch operation {
			case "get":
				_, err = api.GetRecord(context.Background(), "v")
			case "find":
				_, err = api.FindRecord(context.Background(), "v")
			case "list":
				_, err = snapshotsCollect(api.ListRecords(context.Background()))
			case "create":
				_, err = api.CreateRecord(context.Background(), map[string]any{"consistencygroup_id": "source", "name": nil, "description": false})
			case "delete":
				_, err = api.DeleteRecord(context.Background(), "v")
			}
			count := 1
			if operation == "find" {
				count = 2
			}
			if err != nil || len(*calls) != count {
				t.Fatal(operation, err, *calls)
			}
			for _, call := range *calls {
				if call.headers.Get("OpenStack-API-Version") != "" || call.headers.Get("X-OpenStack-Volume-API-Version") != "" {
					t.Fatal("default version headers must remain absent", call)
				}
			}
		})
	}
}

func TestPythonConsistencyGroupSnapshotCreateMissingRequiredFieldKeepsServerRejection(t *testing.T) {
	body := `{"badRequest":{"message":"consistencygroup_id must be provided"}}`
	api, calls := snapshotAPI(t, "", func(r *http.Request) *http.Response {
		if r.Method != http.MethodPost || r.URL.Path != snapshotBase {
			t.Fatal(r.Method, r.URL)
		}
		return snapshotWire(400, body)
	})
	value, err := api.CreateRecord(context.Background(), nil)
	var native gophercloud.ErrUnexpectedResponseCode
	if value == nil || value.Resource != nil || value.Wire != nil || value.StatusCode != 400 || string(value.Envelope) != body || !errors.As(err, &native) || native.Actual != 400 || string(native.Body) != body || len(*calls) != 1 || (*calls)[0].body != `{"cgsnapshot":{}}` {
		t.Fatal(value, err, native, *calls)
	}
	if value.Header.Get("X-Request-Id") != "actual-consistency-group-snapshot" || native.ResponseHeader.Get("X-Request-Id") != "actual-consistency-group-snapshot" {
		t.Fatal(value, native)
	}
}
