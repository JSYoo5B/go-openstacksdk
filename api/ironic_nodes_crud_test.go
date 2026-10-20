package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ironicNodeVersion is the microversion the fixture client carries. The
// generated API never selects a version itself; Gophercloud turns the client
// field into the X-OpenStack-Ironic-API-Version header.
const ironicNodeVersion = "1.99"

const ironicNodeBody = `{"uuid":"n-1","name":"","provision_state":"available","power_state":null,"maintenance":false,"maintenance_reason":null,"instance_uuid":null,"automated_clean":false,"disable_power_off":false,"inspection_started_at":null,"created_at":"2026-01-02T03:04:05+00:00","extra":{},"traits":[],"vendor_extra":1}`

type ironicNodeCall struct {
	Method string
	Path   string
	Query  url.Values
	Body   string
	Header http.Header
}

type ironicNodeRecorder struct {
	mu    sync.Mutex
	calls []ironicNodeCall
}

func (r *ironicNodeRecorder) all() []ironicNodeCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.calls)
}

func (r *ironicNodeRecorder) count() int { return len(r.all()) }

func (r *ironicNodeRecorder) only(t *testing.T) ironicNodeCall {
	t.Helper()
	calls := r.all()
	if len(calls) != 1 {
		t.Fatalf("calls=%d: %+v", len(calls), calls)
	}
	return calls[0]
}

// ironicNodeServe registers one fixture route that records each request,
// asserts the Ironic microversion header and answers with a fixed status.
func ironicNodeServe(t *testing.T, cloud *testcloud.Cloud, pattern string, status int, body string) *ironicNodeRecorder {
	t.Helper()
	recorder := &ironicNodeRecorder{}
	cloud.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		recorder.mu.Lock()
		recorder.calls = append(recorder.calls, ironicNodeCall{Method: r.Method, Path: r.URL.Path, Query: r.URL.Query(), Body: string(data), Header: r.Header.Clone()})
		recorder.mu.Unlock()
		if got := r.Header.Get("X-OpenStack-Ironic-API-Version"); got != ironicNodeVersion {
			t.Errorf("%s %s microversion header=%q", r.Method, r.URL.Path, got)
		}
		if body == "" {
			w.WriteHeader(status)
			return
		}
		testcloud.JSON(w, status, body)
	})
	return recorder
}

func newIronicNodes(t *testing.T, cloud *testcloud.Cloud) *nodes.API {
	t.Helper()
	client := cloud.Client("baremetal", "/v1")
	client.Microversion = ironicNodeVersion
	return nodes.New(client)
}

func ironicNodeFields(t *testing.T, raw string) map[string]string {
	t.Helper()
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("body %q: %v", raw, err)
	}
	fields := map[string]string{}
	for key, value := range decoded {
		// Re-encode so nested object keys compare in a stable order.
		var nested any
		if err := json.Unmarshal(value, &nested); err != nil {
			t.Fatalf("field %s: %v", key, err)
		}
		canonical, _ := json.Marshal(nested)
		fields[key] = string(canonical)
	}
	return fields
}

func ironicNodeFalse() *bool { value := false; return &value }

// ironicNodeUnreachable fails the test for any HTTP request.
func ironicNodeUnreachable(t *testing.T, cloud *testcloud.Cloud) {
	t.Helper()
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
}

type ironicNodeStatusCase struct {
	name    string
	pattern string
	ok      []int
	bad     []int
	body    string
	call    func(ctx context.Context, api *nodes.API) error
}

// ironicNodeRunStatusCases verifies the exact accepted success codes: every ok
// status succeeds and every bad status returns the native unexpected-code error.
func ironicNodeRunStatusCases(t *testing.T, cases []ironicNodeStatusCase) {
	t.Helper()
	for _, tc := range cases {
		for _, status := range append(slices.Clone(tc.ok), tc.bad...) {
			success := slices.Contains(tc.ok, status)
			cloud := testcloud.New(t)
			body := ""
			if success && status != 204 {
				body = tc.body
			}
			recorder := ironicNodeServe(t, cloud, tc.pattern, status, body)
			err := tc.call(context.Background(), newIronicNodes(t, cloud))
			if success != (err == nil) {
				t.Errorf("%s status %d error=%v", tc.name, status, err)
			}
			if !success {
				var unexpected gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &unexpected) || unexpected.Actual != status {
					t.Errorf("%s status %d error=%v", tc.name, status, err)
				}
			}
			if recorder.count() != 1 {
				t.Errorf("%s status %d calls=%d", tc.name, status, recorder.count())
			}
		}
	}
}

func TestIronicNodeCreateSendsFlatBodyKeepsFalseAndExtensions(t *testing.T) {
	cloud := testcloud.New(t)
	recorder := ironicNodeServe(t, cloud, "POST /v1/nodes", 201, ironicNodeBody)
	value, err := newIronicNodes(t, cloud).Create(context.Background(), nodes.CreateOpts{
		Driver: "ipmi", Name: "bm-1", AutomatedClean: ironicNodeFalse(), DisablePowerOff: ironicNodeFalse(),
		Extra: map[string]any{"rack": nil, "spare": false},
	}, nodes.WithCreateField("vendor", map[string]any{"enabled": false}))
	if err != nil || value == nil || value.UUID != "n-1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	call := recorder.only(t)
	want := map[string]string{"driver": `"ipmi"`, "name": `"bm-1"`, "automated_clean": `false`, "disable_power_off": `false`, "extra": `{"rack":null,"spare":false}`, "vendor": `{"enabled":false}`}
	if got := ironicNodeFields(t, call.Body); !reflect.DeepEqual(got, want) {
		t.Fatalf("body=%v", got)
	}
	if value.Name != "" || value.Maintenance || value.AutomatedClean == nil || *value.AutomatedClean || value.InspectionStartedAt != nil || value.Traits == nil || len(value.Traits) != 0 || value.CreatedAt.IsZero() {
		t.Fatalf("decoded node=%+v", value)
	}
}

func TestIronicNodeCreateOmitsUnsetOptionalFields(t *testing.T) {
	cloud := testcloud.New(t)
	recorder := ironicNodeServe(t, cloud, "POST /v1/nodes", 202, ironicNodeBody)
	if _, err := newIronicNodes(t, cloud).Create(context.Background(), nodes.CreateOpts{Driver: "fake-hardware"}); err != nil {
		t.Fatal(err)
	}
	if got := ironicNodeFields(t, recorder.only(t).Body); !reflect.DeepEqual(got, map[string]string{"driver": `"fake-hardware"`}) {
		t.Fatalf("body=%v", got)
	}
}

func TestIronicNodeRejectsProtectedOrNilOptionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	ironicNodeUnreachable(t, cloud)
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	if _, err := api.Create(ctx, nodes.CreateOpts{Driver: "ipmi"}, nodes.WithCreateField("driver", "shadow")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected create field: %v", err)
	}
	if _, err := api.Create(ctx, nodes.CreateOpts{}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil create option: %v", err)
	}
	if _, err := api.Update(ctx, "n-1", nodes.UpdateOpts{}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil update option: %v", err)
	}
	for range api.List(ctx, nil) {
		// One invalid-option error is yielded and no request is sent.
	}
	for _, err := range api.ListDetail(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list detail option: %v", err)
		}
	}
	for _, err := range api.ListDetail(ctx, nodes.WithListDetailOptions(nodes.ListOpts{Fields: []string{"uuid"}})) {
		if err == nil {
			t.Fatal("detail listing with fields must fail before HTTP")
		}
	}
}

func TestIronicNodeUpdateSendsJSONPatchWithExplicitValues(t *testing.T) {
	cloud := testcloud.New(t)
	recorder := ironicNodeServe(t, cloud, "PATCH /v1/nodes/n-1", 200, ironicNodeBody)
	value, err := newIronicNodes(t, cloud).Update(context.Background(), "n-1", nodes.UpdateOpts{
		nodes.UpdateOperation{Op: nodes.ReplaceOp, Path: "/name", Value: ""},
		nodes.UpdateOperation{Op: nodes.ReplaceOp, Path: "/maintenance", Value: false},
		nodes.UpdateOperation{Op: nodes.AddOp, Path: "/extra/rack", Value: nil},
		nodes.UpdateOperation{Op: nodes.AddOp, Path: "/extra/spares", Value: 0},
		nodes.UpdateOperation{Op: nodes.RemoveOp, Path: "/extra/old"},
	})
	if err != nil || value == nil || value.UUID != "n-1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	call := recorder.only(t)
	var patch []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(call.Body), &patch); err != nil || len(patch) != 5 {
		t.Fatalf("patch=%s error=%v", call.Body, err)
	}
	for i, want := range []struct{ op, path, value string }{
		{`"replace"`, `"/name"`, `""`}, {`"replace"`, `"/maintenance"`, `false`}, {`"add"`, `"/extra/rack"`, ``}, {`"add"`, `"/extra/spares"`, `0`}, {`"remove"`, `"/extra/old"`, ``},
	} {
		got := patch[i]
		value, has := got["value"]
		if string(got["op"]) != want.op || string(got["path"]) != want.path || (want.value == "") == has || (has && string(value) != want.value) {
			t.Errorf("patch[%d]=%v", i, got)
		}
	}
}

func TestIronicNodeUpdateRejectsInvalidPatchBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	ironicNodeUnreachable(t, cloud)
	if _, err := newIronicNodes(t, cloud).Update(context.Background(), "n-1", nodes.UpdateOpts{nodes.UpdateOperation{Path: "/name"}}); err == nil {
		t.Fatal("a patch without op must fail")
	}
}

func TestIronicNodeSuccessCodesAreExactForCRUD(t *testing.T) {
	ironicNodeRunStatusCases(t, []ironicNodeStatusCase{
		{name: "Create", pattern: "POST /v1/nodes", ok: []int{201, 202}, bad: []int{200, 204}, body: ironicNodeBody,
			call: func(ctx context.Context, api *nodes.API) error {
				_, err := api.Create(ctx, nodes.CreateOpts{Driver: "ipmi"})
				return err
			}},
		{name: "Get", pattern: "GET /v1/nodes/n-1", ok: []int{200}, bad: []int{201, 204}, body: ironicNodeBody,
			call: func(ctx context.Context, api *nodes.API) error { _, err := api.Get(ctx, "n-1"); return err }},
		{name: "Update", pattern: "PATCH /v1/nodes/n-1", ok: []int{200}, bad: []int{201, 202}, body: ironicNodeBody,
			call: func(ctx context.Context, api *nodes.API) error {
				_, err := api.Update(ctx, "n-1", nodes.UpdateOpts{nodes.UpdateOperation{Op: nodes.ReplaceOp, Path: "/name", Value: "x"}})
				return err
			}},
		{name: "Delete", pattern: "DELETE /v1/nodes/n-1", ok: []int{202, 204}, bad: []int{200},
			call: func(ctx context.Context, api *nodes.API) error { return api.Delete(ctx, "n-1") }},
		{name: "Validate", pattern: "GET /v1/nodes/n-1/validate", ok: []int{200}, bad: []int{202, 204}, body: `{}`,
			call: func(ctx context.Context, api *nodes.API) error { _, err := api.Validate(ctx, "n-1"); return err }},
	})
}

func TestIronicNodeGetKeepsNativeNotFoundAndNullsAndVersionHeader(t *testing.T) {
	cloud := testcloud.New(t)
	ironicNodeServe(t, cloud, "GET /v1/nodes/n-1", 200, ironicNodeBody)
	ironicNodeServe(t, cloud, "GET /v1/nodes/missing", 404, `{"error_message":"not found"}`)
	api := newIronicNodes(t, cloud)
	value, err := api.Get(context.Background(), "n-1")
	if err != nil || value.Name != "" || value.PowerState != "" || value.InstanceUUID != "" || value.AutomatedClean == nil || *value.AutomatedClean {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if _, err := api.Get(context.Background(), "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
}

func TestIronicNodeWithoutClientMicroversionSendsNoVersionHeader(t *testing.T) {
	cloud := testcloud.New(t)
	var header atomic.Value
	cloud.Mux.HandleFunc("GET /v1/nodes/n-1", func(w http.ResponseWriter, r *http.Request) {
		header.Store(r.Header.Clone())
		testcloud.JSON(w, 200, ironicNodeBody)
	})
	api := nodes.New(cloud.Client("baremetal", "/v1"))
	if _, err := api.Get(context.Background(), "n-1"); err != nil {
		t.Fatal(err)
	}
	sent, _ := header.Load().(http.Header)
	if _, present := sent["X-Openstack-Ironic-Api-Version"]; present {
		t.Fatalf("the API must not choose a microversion itself: %v", sent)
	}
}

func TestIronicNodeDeleteIsStrictAndRemoveIgnoresMissingByDefault(t *testing.T) {
	cloud := testcloud.New(t)
	deletes := ironicNodeServe(t, cloud, "DELETE /v1/nodes/n-1", 204, "")
	ironicNodeServe(t, cloud, "DELETE /v1/nodes/missing", 404, `{"error_message":"gone"}`)
	ironicNodeServe(t, cloud, "GET /v1/nodes/missing", 404, `{"error_message":"gone"}`)
	api := newIronicNodes(t, cloud)
	ctx := context.Background()
	if err := api.Delete(ctx, "n-1"); err != nil {
		t.Fatal(err)
	}
	if call := deletes.only(t); call.Body != "" || len(call.Query) != 0 {
		t.Fatalf("delete call=%+v", call)
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if err := api.Delete(ctx, "missing"); !errors.As(err, &unexpected) || unexpected.Actual != 404 {
		t.Fatalf("native delete keeps the 404 response error: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores a missing node by default: %v", err)
	}
}

func TestIronicNodeListFiltersFollowLinksAndKeepExplicitFalseQuery(t *testing.T) {
	cloud := testcloud.New(t)
	var mu sync.Mutex
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.URL.Query())
		mu.Unlock()
		if r.Header.Get("X-OpenStack-Ironic-API-Version") != ironicNodeVersion {
			t.Errorf("microversion header=%q", r.Header.Get("X-OpenStack-Ironic-API-Version"))
		}
		if r.URL.Query().Get("marker") == "n-1" {
			testcloud.JSON(w, 200, `{"nodes":[{"uuid":"n-2"}],"nodes_links":[]}`)
			return
		}
		next := cloud.Server.URL + "/v1/nodes?marker=n-1&limit=1"
		testcloud.JSON(w, 200, `{"nodes":[{"uuid":"n-1"}],"nodes_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	var ids []string
	for value, err := range newIronicNodes(t, cloud).List(context.Background(),
		nodes.WithListOptions(nodes.ListOpts{Driver: "ipmi", ProvisionState: nodes.Available, Fields: []string{"uuid", "name"}, Limit: 1, Maintenance: false}),
		nodes.WithListQuery("maintenance", "false"), nodes.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.UUID)
	}
	if !reflect.DeepEqual(ids, []string{"n-1", "n-2"}) || len(requests) != 2 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	first := requests[0]
	if first.Get("driver") != "ipmi" || first.Get("provision_state") != "available" || first.Get("fields") != "uuid,name" || first.Get("limit") != "1" || first.Get("maintenance") != "false" || first.Get("vendor") != "x" {
		t.Fatalf("first query=%v", first)
	}
	if requests[1].Get("marker") != "n-1" {
		t.Fatalf("second query=%v", requests[1])
	}
}

func TestIronicNodeTypedBoolFalseIsOmittedButTrueIsSent(t *testing.T) {
	cloud := testcloud.New(t)
	recorder := ironicNodeServe(t, cloud, "GET /v1/nodes", 200, `{"nodes":[]}`)
	api := newIronicNodes(t, cloud)
	for _, opts := range []nodes.ListOpts{{Maintenance: false}, {Maintenance: true, Associated: true}} {
		for _, err := range api.List(context.Background(), nodes.WithListOptions(opts)) {
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	calls := recorder.all()
	if len(calls) != 2 || calls[0].Query.Has("maintenance") || calls[1].Query.Get("maintenance") != "true" || calls[1].Query.Get("associated") != "true" {
		t.Fatalf("calls=%+v", calls)
	}
}

func TestIronicNodeListDetailUsesDetailRouteAndCommonLookups(t *testing.T) {
	cloud := testcloud.New(t)
	var mu sync.Mutex
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v1/nodes/detail", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		queries = append(queries, r.URL.Query())
		mu.Unlock()
		testcloud.JSON(w, 200, `{"nodes":[{"uuid":"n-1","name":"bm-1","provision_state":"active"},{"uuid":"n-2","name":"bm-2","provision_state":"available"}]}`)
	})
	ironicNodeServe(t, cloud, "GET /v1/nodes/n-1", 200, `{"uuid":"n-1","name":"bm-1","provision_state":"active"}`)
	api := newIronicNodes(t, cloud)
	var ids []string
	for value, err := range api.ListDetail(context.Background(), nodes.WithListDetailOptions(nodes.ListOpts{Owner: "proj"}), nodes.WithListDetailQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.UUID)
	}
	if !reflect.DeepEqual(ids, []string{"n-1", "n-2"}) || queries[0].Get("owner") != "proj" || queries[0].Get("vendor") != "x" {
		t.Fatalf("ids=%v queries=%v", ids, queries)
	}
	found, err := api.Find(context.Background(), resource.ID("n-1"))
	if err != nil || found == nil || found.UUID != "n-1" {
		t.Fatalf("find=%+v error=%v", found, err)
	}
}

func TestIronicNodeCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	api := newIronicNodes(t, cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(canceled, "n-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if _, err := api.Create(canceled, nodes.CreateOpts{Driver: "ipmi"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled create: %v", err)
	}
	if _, err := api.Update(canceled, "n-1", nodes.UpdateOpts{nodes.UpdateOperation{Op: nodes.RemoveOp, Path: "/extra/x"}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled update: %v", err)
	}
	if err := api.Delete(canceled, "n-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled delete: %v", err)
	}
	if _, err := api.Validate(canceled, "n-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled validate: %v", err)
	}
	for _, err := range api.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled list: %v", err)
		}
	}
	for _, err := range api.ListDetail(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled list detail: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Create(deadline, nodes.CreateOpts{Driver: "ipmi"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}

func TestIronicNodeValidateDecodesPerInterfaceResults(t *testing.T) {
	cloud := testcloud.New(t)
	ironicNodeServe(t, cloud, "GET /v1/nodes/n-1/validate", 200, `{"power":{"result":true,"reason":null},"boot":{"result":false,"reason":"missing image"},"management":{"result":null,"reason":"not supported"}}`)
	result, err := newIronicNodes(t, cloud).Validate(context.Background(), "n-1")
	if err != nil || !result.Power.Result || result.Boot.Result || result.Boot.Reason != "missing image" || result.Management.Reason != "not supported" {
		t.Fatalf("result=%+v error=%v", result, err)
	}
}
