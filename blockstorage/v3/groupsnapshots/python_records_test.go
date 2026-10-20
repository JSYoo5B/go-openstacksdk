package groupsnapshots_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots"
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
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"actual-group-snapshot"}}, Body: io.NopCloser(strings.NewReader(body))}
}

const snapshotBase = "/cinder/v3/project/group_snapshots"

func snapshotAPI(t *testing.T, version string, reply func(*http.Request) *http.Response) (*groupsnapshots.API, *[]snapshotCall) {
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
	return groupsnapshots.New(client), calls
}
func snapshotField(t *testing.T, value *groupsnapshots.GroupSnapshotRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatal(key, want, value)
	}
}
func snapshotsCollect(sequence func(func(*groupsnapshots.GroupSnapshotRecord, error) bool)) ([]*groupsnapshots.GroupSnapshotRecord, error) {
	var values []*groupsnapshots.GroupSnapshotRecord
	for value, err := range sequence {
		if err != nil {
			return values, err
		}
		values = append(values, value)
	}
	return values, nil
}

func TestPythonGroupSnapshotGetOwnsUntypedDescriptorsEnvelopeAndLocation(t *testing.T) {
	row := `{"id":9007199254740993,"name":false,"created_at":[null,0],"description":{"enabled":false},"group_id":null,"group_type_id":0,"project_id":"other","status":"available","metadata":{"unknown":true},"self":"wire","connection":false}`
	for _, wrapper := range []string{"group_snapshot", "flat"} {
		body := row
		if wrapper != "flat" {
			body = `{"group_snapshot":` + row + `}`
		}
		api, calls := snapshotAPI(t, "3.80", func(*http.Request) *http.Response { return snapshotWire(203, body) })
		cloud, project := "named-cloud", "current"
		location := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"current"`), Name: &project}}
		value, err := api.GetRecord(context.Background(), "literal /%雪", groupsnapshots.WithRecordReadLocation(location))
		if err != nil || len(*calls) != 1 || (*calls)[0].escaped != snapshotBase+"/literal%20%2F%25%E9%9B%AA" || (*calls)[0].query != "" || (*calls)[0].body != "" || (*calls)[0].headers.Get("OpenStack-API-Version") != "volume 3.80" {
			t.Fatal(value, err, *calls)
		}
		for key, want := range map[string]string{"id": "9007199254740993", "name": "false", "created_at": "[null,0]", "description": `{"enabled":false}`, "group_id": "null", "group_type_id": "0", "project_id": `"other"`} {
			snapshotField(t, value, key, want)
		}
		if len(value.Resource.Body) != 9 || value.Resource.Body["metadata"] != nil || value.Wire == nil || value.StatusCode != 203 || string(value.Envelope) != body {
			t.Fatal(value)
		}
		var found resource.CloudLocation
		if err := json.Unmarshal(value.Resource.Body["location"], &found); err != nil || found.Cloud == nil || *found.Cloud != "named-cloud" || found.Project.Name != nil || string(found.Project.ID) != `"other"` {
			t.Fatal(found, err)
		}
		value.Resource.Body["id"][0] = '1'
		value.Header.Set("X-Request-Id", "changed")
		if string(value.Wire.Body["id"]) != "9007199254740993" || value.Wire.Header.Get("X-Request-Id") != "actual-group-snapshot" {
			t.Fatal("carriers alias", value)
		}
	}
}
func TestPythonGroupSnapshotGetOpaqueSeedNullAndInvalidShapeProof(t *testing.T) {
	seed := &groupsnapshots.GroupSnapshotRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"seed"`), "name": json.RawMessage(`false`), "description": json.RawMessage(`null`)}}}}
	option := groupsnapshots.WithRecordReadSeed(seed)
	seed.Resource.Body["name"] = json.RawMessage(`true`)
	for _, body := range []string{"", "not JSON", `{}`, `{"group_snapshot":{"id":null}}`, `null`, `[]`, `{"group_snapshot":null}`} {
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(204, body) })
		value, err := api.GetRecord(context.Background(), "selected", option)
		invalid := body == `null` || body == `[]` || body == `{"group_snapshot":null}`
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
func TestPythonGroupSnapshotCreateDeclaredDirtyAttrsAndImmutableOptions(t *testing.T) {
	attrs := map[string]any{"id": false, "group_id": "group", "name": "before", "description": map[string]any{"null": nil, "zero": 0, "enabled": false}, "group_type_id": json.Number("9007199254740993"), "metadata": snapshotMarshal(func() ([]byte, error) { t.Fatal("unknown attr was marshaled"); return nil, nil })}
	headers := map[string]string{"X-Owned": "original"}
	option := groupsnapshots.WithRecordCreateHeaders(headers)
	headers["X-Owned"] = "mutated"
	api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
		return snapshotWire(202, `{"group_snapshot":{"id":"server","name":null,"status":"creating","unknown":9007199254740993}}`)
	})
	var retained *groupsnapshots.RecordCreateOpts
	value, err := api.CreateRecord(context.Background(), attrs, option, func(cfg *groupsnapshots.RecordCreateOpts) error {
		attrs["name"] = "after"
		attrs["description"].(map[string]any)["zero"] = 99
		cfg.Headers["X-Next"] = "first"
		retained = cfg
		return nil
	}, func(*groupsnapshots.RecordCreateOpts) error { retained.Headers["X-Next"] = "after"; return nil })
	if err != nil || len(*calls) != 1 || (*calls)[0].method != "POST" || (*calls)[0].path != snapshotBase || (*calls)[0].headers.Get("X-Owned") != "original" || (*calls)[0].headers.Get("X-Next") != "first" || (*calls)[0].headers.Get("Content-Type") != "application/json" {
		t.Fatal(value, err, *calls)
	}
	var body map[string]map[string]json.RawMessage
	if err := json.Unmarshal([]byte((*calls)[0].body), &body); err != nil || string(body["group_snapshot"]["id"]) != "false" || string(body["group_snapshot"]["name"]) != `"before"` || string(body["group_snapshot"]["group_type_id"]) != "9007199254740993" || body["group_snapshot"]["metadata"] != nil {
		t.Fatal(body, err)
	}
	snapshotField(t, value, "id", `"server"`)
	snapshotField(t, value, "name", "null")
	snapshotField(t, value, "description", `{"enabled":false,"null":null,"zero":0}`)
	if value.Wire == nil || string(value.Wire.Body["unknown"]) != "9007199254740993" || len(value.Discovery) != 0 {
		t.Fatal(value)
	}
	for _, attrs := range []map[string]any{nil, {}, {"id": nil, "name": "", "description": false}} {
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(204, "opaque") })
		value, err := api.CreateRecord(context.Background(), attrs)
		if err != nil || len(*calls) != 1 || value.Resource == nil || string(value.Envelope) != "opaque" {
			t.Fatal(value, err, *calls)
		}
		if len(attrs) == 0 && (*calls)[0].body != `{"group_snapshot":{}}` {
			t.Fatal(*calls)
		}
	}
}
func TestPythonGroupSnapshotCreateSourceCaptureMarshalAndConstructorCollisions(t *testing.T) {
	for _, mode := range []string{"marshal source", "marshal cancel", "option source", "option cancel", "connection", "_synchronized", "self"} {
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { t.Fatal("preflight reached HTTP"); return snapshotWire(200, `{}`) })
		ctx, cancel := context.WithCancelCause(context.Background())
		cause := errors.New("callback cause")
		attrs := map[string]any{"group_id": "group"}
		opts := []groupsnapshots.RecordCreateOption{}
		switch mode {
		case "marshal source":
			attrs["name"] = snapshotMarshal(func() ([]byte, error) { api.RawClient().ResourceBase = "https://changed.test/v3/"; return nil, cause })
		case "marshal cancel":
			attrs["name"] = snapshotMarshal(func() ([]byte, error) { cancel(cause); return nil, cause })
		case "option source":
			opts = append(opts, func(*groupsnapshots.RecordCreateOpts) error { api.RawClient().Microversion = "3.80"; return cause })
		case "option cancel":
			opts = append(opts, func(*groupsnapshots.RecordCreateOpts) error { cancel(cause); return cause })
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
func TestPythonGroupSnapshotListSourceQueriesDetailsFiltersAndRawCap(t *testing.T) {
	for _, details := range []bool{true, false} {
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
			return snapshotWire(200, `{"group_snapshots":[{"id":"first","name":"data","description":{"enabled":true,"extra":0}},{"id":"second","name":"data","description":{"enabled":false}}]}`)
		})
		options := []groupsnapshots.RecordListOption{groupsnapshots.WithRecordListQuery("all_projects", false), groupsnapshots.WithRecordListQuery("sort_dir", []string{"asc", "desc"}), groupsnapshots.WithRecordListQuery("sort", "name:asc"), groupsnapshots.WithRecordListQuery("name", "data"), groupsnapshots.WithRecordListQuery("description", map[string]any{"enabled": 1}), groupsnapshots.WithRecordListQuery("unknown", snapshotMarshal(func() ([]byte, error) { t.Fatal("unknown query marshaled"); return nil, nil }))}
		if !details {
			options = append(options, groupsnapshots.WithRecordListDetails(false))
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
		if (*calls)[0].path != want || query.Get("all_tenants") != "False" || query.Get("sort") != "name:asc" || !reflect.DeepEqual(query["sort_dir"], []string{"asc", "desc"}) || query.Has("name") || query.Has("description") || query.Has("unknown") {
			t.Fatal(*calls)
		}
		snapshotField(t, values[0], "id", `"first"`)
		values, err = snapshotsCollect(api.ListRecords(context.Background(), groupsnapshots.WithRecordListMaxItems(1), groupsnapshots.WithRecordListQuery("id", "second")))
		if err != nil || len(values) != 0 || len(*calls) != 2 || !strings.Contains((*calls)[1].query, "limit=1") {
			t.Fatal(values, err, *calls)
		}
	}
}
func TestPythonGroupSnapshotListLocalFloatPolicyAndConstructorCollisions(t *testing.T) {
	api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
		return snapshotWire(200, `{"group_snapshots":[{"id":"v","group_type_id":9007199254740992}]}`)
	})
	values, err := snapshotsCollect(api.ListRecords(context.Background(), groupsnapshots.WithRecordListQuery("group_type_id", json.RawMessage(`9007199254740993.0`))))
	if err != nil || len(values) != 0 || len(*calls) != 1 {
		t.Fatal("exact-decimal filter policy changed", values, err, *calls)
	}
	for _, key := range []string{"connection", "microversion", "_synchronized"} {
		body := `{"group_snapshots":[{"id":"v","` + key + `":null}]}`
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(200, body) })
		values, err := snapshotsCollect(api.ListRecords(context.Background()))
		var proof *resource.ResponseError
		if len(values) != 0 || err == nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || len(*calls) != 1 {
			t.Fatal(key, values, err, proof, *calls)
		}
	}
	for _, description := range []string{`"truthy"`, `false`, `{}`, `{"enabled":true}`} {
		body := `{"group_snapshots":[{"id":"v","description":` + description + `}]}`
		api, _ := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(200, body) })
		values, err := snapshotsCollect(api.ListRecords(context.Background(), groupsnapshots.WithRecordListQuery("description", map[string]any{})))
		want := description == `"truthy"` || description == `{"enabled":true}`
		if err != nil || (len(values) == 1) != want {
			t.Fatal(description, values, err)
		}
	}
}
func TestPythonGroupSnapshotListPagingUsesRetainedScalarMarkerAndScope(t *testing.T) {
	api, calls := snapshotAPI(t, "3.29", func(r *http.Request) *http.Response {
		if r.URL.Query().Get("marker") == "9007199254740993" {
			return snapshotWire(200, `{"group_snapshots":[]}`)
		}
		return snapshotWire(200, `{"group_snapshots":[{"id":9007199254740993,"name":"kept"}]}`)
	})
	var values []*groupsnapshots.GroupSnapshotRecord
	for value, err := range api.ListRecords(context.Background(), groupsnapshots.WithRecordListLimit(2), groupsnapshots.WithRecordListAllProjects(true)) {
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, value)
		value.Resource.Body["id"] = json.RawMessage(`"mutated"`)
		value.Wire.Body["id"] = json.RawMessage(`"also-mutated"`)
	}
	if len(values) != 1 || len(*calls) != 2 || (*calls)[1].query != "all_tenants=True&limit=2&marker=9007199254740993" {
		t.Fatal(values, *calls)
	}
	api, calls = snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
		return snapshotWire(200, `{"group_snapshots":[{"id":"v"}],"group_snapshots_links":[{"rel":"next","href":"?marker=unvisited"}]}`)
	})
	values, err := snapshotsCollect(api.ListRecords(context.Background(), groupsnapshots.WithRecordListPaginated(false)))
	if err != nil || len(values) != 1 || len(*calls) != 1 {
		t.Fatal(values, err, *calls)
	}
	api, _ = snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
		return snapshotWire(200, `{"group_snapshots":[{"id":"v"}],"next":"https://changed.test/group_snapshots/detail?marker=unsafe"}`)
	})
	_, err = snapshotsCollect(api.ListRecords(context.Background()))
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
func TestPythonGroupSnapshotDefaultDiscoveryAndExplicitVersion(t *testing.T) {
	for _, selected := range []string{"", "3.80"} {
		api, calls := snapshotAPI(t, selected, func(r *http.Request) *http.Response {
			if r.URL.Path == "/cinder/v3/" {
				return snapshotWire(200, `{"version":{"id":"v3.0","status":"CURRENT","min_version":"3.0","version":"3.80"}}`)
			}
			return snapshotWire(200, `{"group_snapshot":{"id":"v"}}`)
		})
		value, err := api.GetRecord(context.Background(), "v")
		want := selected
		count := 1
		discovery := 0
		if selected == "" {
			want = "3.29"
			count = 2
			discovery = 1
		}
		if err != nil || value == nil || len(*calls) != count || len(value.Discovery) != discovery || (*calls)[count-1].headers.Get("OpenStack-API-Version") != "volume "+want || api.RawClient().Microversion != selected {
			t.Fatal(selected, value, err, *calls)
		}
	}
	api, calls := snapshotAPI(t, "3.80", func(*http.Request) *http.Response { return snapshotWire(200, `{}`) })
	_, err := api.GetRecord(context.Background(), "v", groupsnapshots.WithRecordReadMicroversion(""))
	if err != nil || len(*calls) != 1 || (*calls)[0].headers.Get("OpenStack-API-Version") != "" {
		t.Fatal(err, *calls)
	}
}
func TestPythonGroupSnapshotDeleteDefaultMissingStrictAndOpaqueACK(t *testing.T) {
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
	value, err := api.DeleteRecord(context.Background(), "v", groupsnapshots.WithRecordDeleteIgnoreMissing(false))
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
func TestPythonGroupSnapshotPhysicalFaultsNeverReplayIgnoreOrInventSuccess(t *testing.T) {
	for _, operation := range []string{"get", "list", "create", "delete"} {
		for _, status := range []int{203, 404, 503} {
			t.Run(fmt.Sprintf("%s/%d", operation, status), func(t *testing.T) {
				readCause, closeCause, cancelCause := errors.New("read"), errors.New("close"), errors.New("cancel")
				ctx, cancel := context.WithCancelCause(context.Background())
				var client *gophercloud.ServiceClient
				var closes atomic.Int32
				original := ""
				body := `{"group_snapshot":{"id":"v"}}`
				if operation == "list" {
					body = `{"group_snapshots":[{"id":"v"}]}`
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
					value, e := api.CreateRecord(ctx, map[string]any{"group_id": "g"})
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
func TestPythonGroupSnapshotRetryOwnsJSONHeadersAndSelectedVersion(t *testing.T) {
	for _, mode := range []string{"clean", "body", "version", "header", "denied"} {
		attrs := map[string]any{"group_id": "g", "name": "before"}
		count := 0
		cause := errors.New("retry denied")
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
			count++
			if count == 1 {
				return snapshotWire(503, "first")
			}
			return snapshotWire(202, `{"group_snapshot":{"id":"v"}}`)
		})
		api.RawClient().MoreHeaders = map[string]string{"Openstack-Api-Version": "volume 3.29", "X-Openstack-Volume-Api-Version": "3.29"}
		api.RawClient().RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
			attrs["name"] = "after"
			switch mode {
			case "body":
				opts.JSONBody = json.RawMessage(`{"group_snapshot":{}}`)
			case "version":
				opts.MoreHeaders["OpenStack-API-Version"] = "volume 3.80"
			case "header":
				opts.MoreHeaders["X-Owned"] = "changed"
			case "denied":
				return cause
			}
			return nil
		}
		value, err := api.CreateRecord(context.Background(), attrs, groupsnapshots.WithRecordCreateHeaders(map[string]string{"X-Owned": "fixed"}))
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
func TestPythonGroupSnapshotOptionsPreflightOwnershipAndNilAPI(t *testing.T) {
	api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
		t.Fatal("invalid option reached HTTP")
		return snapshotWire(200, `{}`)
	})
	for _, id := range []string{"", " ", ".", "..", "control\n", string([]byte{0xff})} {
		if _, err := api.GetRecord(context.Background(), id); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(id, err)
		}
	}
	for _, opts := range [][]groupsnapshots.RecordListOption{{nil}, {groupsnapshots.WithRecordListMaxItems(-1)}, {groupsnapshots.WithRecordListAllProjects(false), groupsnapshots.WithRecordListQuery("all_tenants", nil)}, {groupsnapshots.WithRecordListHeaders(map[string]string{"X-Auth-Token": "caller"})}} {
		_, err := snapshotsCollect(api.ListRecords(context.Background(), opts...))
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`not JSON`)}}
	if _, err := api.GetRecord(context.Background(), "v", groupsnapshots.WithRecordReadLocation(location)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.CreateRecord(context.Background(), nil, groupsnapshots.WithRecordCreateLocation(location)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := snapshotsCollect(api.ListRecords(context.Background(), groupsnapshots.WithRecordListLocation(location))); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if len(*calls) != 0 {
		t.Fatal(*calls)
	}
	var nilAPI *groupsnapshots.API
	if _, err := nilAPI.GetRecord(context.Background(), "v"); err == nil {
		t.Fatal("nil API accepted")
	}
}

func TestPythonGroupSnapshotCreateJSONFramingPrecedesDiscovery(t *testing.T) {
	for _, serviceHeader := range []bool{true, false} {
		api, calls := snapshotAPI(t, "", func(*http.Request) *http.Response {
			t.Fatal("invalid JSON framing reached discovery or POST")
			return snapshotWire(200, `{}`)
		})
		options := []groupsnapshots.RecordCreateOption{}
		if serviceHeader {
			api.RawClient().MoreHeaders = map[string]string{"content-type": "application/custom"}
		} else {
			options = append(options, groupsnapshots.WithRecordCreateHeaders(map[string]string{"CONTENT-TYPE": "application/custom"}))
		}
		value, err := api.CreateRecord(context.Background(), map[string]any{"group_id": "g"}, options...)
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(serviceHeader, value, err, *calls)
		}
	}
}
func TestPythonGroupSnapshotDiscoveryPhysicalFailureRetainsActualProof(t *testing.T) {
	for _, operation := range []string{"get", "create", "list", "delete"} {
		t.Run(operation, func(t *testing.T) {
			readCause, closeCause := errors.New("discovery read"), errors.New("discovery close")
			var closes atomic.Int32
			body := `{"version":{"id":"v3.0","status":"CURRENT","min_version":"3.0","version":"3.80"}}`
			api, calls := snapshotAPI(t, "", func(r *http.Request) *http.Response {
				if r.URL.Path != "/cinder/v3/" {
					t.Fatal("discovery physical failure reached member", r.URL)
				}
				response := snapshotWire(200, body)
				response.Body = &snapshotFaultBody{data: []byte(body), read: readCause, close: closeCause, closes: &closes}
				return response
			})
			api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return nil }
			var err error
			switch operation {
			case "get":
				value, cause := api.GetRecord(context.Background(), "v")
				err = cause
				if value != nil {
					t.Fatal(value)
				}
			case "create":
				value, cause := api.CreateRecord(context.Background(), map[string]any{"group_id": "g"})
				err = cause
				if value != nil {
					t.Fatal(value)
				}
			case "list":
				values, cause := snapshotsCollect(api.ListRecords(context.Background()))
				err = cause
				if len(values) != 0 {
					t.Fatal(values)
				}
			case "delete":
				value, cause := api.DeleteRecord(context.Background(), "v")
				err = cause
				if value != nil {
					t.Fatal(value)
				}
			}
			var proof *resource.ResponseError
			if !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || len(*calls) != 1 || closes.Load() != 1 {
				t.Fatal(proof, err, *calls, closes.Load())
			}
		})
	}
}

func TestPythonGroupSnapshotRawPagingInputPrecedesDiscovery(t *testing.T) {
	for _, query := range []map[string]any{{"limit": -1}, {"limit": []int{1, 2}}, {"limit": false}, {"marker": " "}, {"marker": []string{"one", "two"}}, {"max_items": false}} {
		api, calls := snapshotAPI(t, "", func(*http.Request) *http.Response {
			t.Fatal("invalid pagination input reached discovery or list")
			return snapshotWire(200, `{}`)
		})
		options := []groupsnapshots.RecordListOption{}
		for key, value := range query {
			options = append(options, groupsnapshots.WithRecordListQuery(key, value))
		}
		values, err := snapshotsCollect(api.ListRecords(context.Background(), options...))
		if len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(query, values, err, *calls)
		}
	}
}

func TestPythonGroupSnapshotBoundClsPresenceAndGetWireBoundary(t *testing.T) {
	for _, input := range []any{nil, false, 0, snapshotMarshal(func() ([]byte, error) { t.Fatal("colliding cls was marshaled"); return nil, nil })} {
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
			t.Fatal("bound cls input reached POST")
			return snapshotWire(200, `{}`)
		})
		value, err := api.CreateRecord(context.Background(), map[string]any{"group_id": "g", "cls": input})
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(input, value, err, *calls)
		}
	}
	for _, raw := range []string{`null`, `false`, `0`} {
		body := `{"group_snapshots":[{"id":"v","cls":` + raw + `}]}`
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(200, body) })
		values, err := snapshotsCollect(api.ListRecords(context.Background()))
		var proof *resource.ResponseError
		if len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || len(*calls) != 1 {
			t.Fatal(raw, values, err, proof, *calls)
		}
		body = `{"group_snapshot":{"id":"v","cls":` + raw + `}}`
		api, calls = snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(200, body) })
		value, err := api.GetRecord(context.Background(), "v")
		if err != nil || value == nil || value.Wire == nil || string(value.Wire.Body["cls"]) != raw || value.Resource.Body["cls"] != nil || len(*calls) != 1 {
			t.Fatal(raw, value, err, *calls)
		}
	}
}
func TestPythonGroupSnapshotListDefaultAcceptOwnsSourceAndExplicitOption(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		requests := 0
		api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response {
			requests++
			if requests == 1 {
				return snapshotWire(503, "first")
			}
			return snapshotWire(200, `{"group_snapshots":[]}`)
		})
		api.RawClient().MoreHeaders = map[string]string{"Accept": "application/source", "X-Source": "retained"}
		api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return nil }
		options := []groupsnapshots.RecordListOption{}
		want := "application/json"
		if explicit {
			options = append(options, groupsnapshots.WithRecordListHeaders(map[string]string{"aCcEpT": "application/operation"}))
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
func TestPythonGroupSnapshotListAdvertisedOffsetAndCursorScope(t *testing.T) {
	t.Run("advertised offset preserves initial defaults and local predicates", func(t *testing.T) {
		api, calls := snapshotAPI(t, "3.29", func(r *http.Request) *http.Response {
			switch r.URL.Query().Get("offset") {
			case "0":
				return snapshotWire(200, `{"group_snapshots":[{"id":"skip","status":"creating"},{"id":"first","status":"available"}],"next":"?offset=2"}`)
			case "2":
				return snapshotWire(200, `{"group_snapshots":[{"id":"last","status":"available"}],"next":"?offset=4"}`)
			case "4":
				return snapshotWire(200, `{"group_snapshots":[]}`)
			default:
				t.Fatal(r.URL)
				return nil
			}
		})
		values, err := snapshotsCollect(api.ListRecords(context.Background(), groupsnapshots.WithRecordListLimit(2), groupsnapshots.WithRecordListAllProjects(false), groupsnapshots.WithRecordListQuery("offset", 0), groupsnapshots.WithRecordListQuery("sort", "name:asc"), groupsnapshots.WithRecordListQuery("status", "available")))
		if err != nil || len(values) != 2 || len(*calls) != 3 {
			t.Fatal(values, err, *calls)
		}
		snapshotField(t, values[0], "id", `"first"`)
		snapshotField(t, values[1], "id", `"last"`)
		for index, call := range *calls {
			q, _ := url.ParseQuery(call.query)
			if q.Get("offset") != fmt.Sprint(index*2) || q.Get("limit") != "2" || q.Get("all_tenants") != "False" || q.Get("sort") != "name:asc" || q.Has("status") || q.Has("marker") {
				t.Fatal(call)
			}
		}
	})
	t.Run("cursor family switches retain filter and limit", func(t *testing.T) {
		api, calls := snapshotAPI(t, "3.29", func(r *http.Request) *http.Response {
			if r.URL.Query().Get("marker") == "start" {
				return snapshotWire(200, `{"group_snapshots":[{"id":"first"}],"next":"?offset=2"}`)
			}
			if r.URL.Query().Get("offset") == "2" {
				return snapshotWire(200, `{"group_snapshots":[{"id":"last"}],"next":"?marker=last"}`)
			}
			if r.URL.Query().Get("marker") == "last" {
				return snapshotWire(200, `{"group_snapshots":[]}`)
			}
			t.Fatal(r.URL)
			return nil
		})
		values, err := snapshotsCollect(api.ListRecords(context.Background(), groupsnapshots.WithRecordListLimit(2), groupsnapshots.WithRecordListMarker("start"), groupsnapshots.WithRecordListAllProjects(true)))
		if err != nil || len(values) != 2 || len(*calls) != 3 {
			t.Fatal(values, err, *calls)
		}
		q, _ := url.ParseQuery((*calls)[1].query)
		if q.Get("offset") != "2" || q.Has("marker") || q.Get("limit") != "2" || q.Get("all_tenants") != "True" {
			t.Fatal(q)
		}
		q, _ = url.ParseQuery((*calls)[2].query)
		if q.Get("marker") != "last" || q.Has("offset") || q.Get("limit") != "2" || q.Get("all_tenants") != "True" {
			t.Fatal(q)
		}
	})
	for _, test := range []struct {
		name, initial, next string
		cycle               bool
	}{
		{"offset cycle", "0", "?offset=0", true}, {"offset backwards", "5", "?offset=3", false}, {"negative offset", "0", "?offset=-1", false}, {"two cursor families", "0", "?offset=2&marker=a", false}, {"filter replacement", "0", "?offset=2&sort=changed", false}, {"limit replacement", "0", "?offset=2&limit=3", false}, {"foreign route", "0", "https://foreign.invalid/group_snapshots/detail?offset=2", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := `{"group_snapshots":[{"id":"first"}],"next":` + fmt.Sprintf("%q", test.next) + `}`
			api, calls := snapshotAPI(t, "3.29", func(*http.Request) *http.Response { return snapshotWire(200, body) })
			values, err := snapshotsCollect(api.ListRecords(context.Background(), groupsnapshots.WithRecordListLimit(2), groupsnapshots.WithRecordListQuery("offset", test.initial), groupsnapshots.WithRecordListQuery("sort", "name:asc")))
			var proof *resource.ResponseError
			var cycle *resource.PaginationCycleError
			if len(values) != 1 || err == nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || len(*calls) != 1 || (test.cycle && !errors.As(err, &cycle)) || (!test.cycle && !errors.Is(err, resource.ErrInvalidOption)) {
				t.Fatal(test, values, err, proof, *calls)
			}
		})
	}
}
