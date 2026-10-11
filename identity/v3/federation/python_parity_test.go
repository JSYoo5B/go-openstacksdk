package federation_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/federation"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonMappingTransport func(*http.Request) (*http.Response, error)

func (transport pythonMappingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonMappingWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type pythonMappingCall struct{ method, path, query, body string }

type pythonMappingRecorder struct {
	mu    sync.Mutex
	calls []pythonMappingCall
}

func (r *pythonMappingRecorder) snapshot() []pythonMappingCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]pythonMappingCall(nil), r.calls...)
}

func pythonMappingAPI(t *testing.T, recorder *pythonMappingRecorder, reply func(*http.Request) *http.Response) *federation.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonMappingTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		recorder.mu.Lock()
		recorder.calls = append(recorder.calls, pythonMappingCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		recorder.mu.Unlock()
		return reply(req), nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return federation.New(client)
}

const (
	pythonMappingBase = "/keystone/v3/OS-FEDERATION/mappings"
	// schema_version and the remote "extra" key are Python-visible data.
	pythonMappingRow = `{"id":"ACME","schema_version":"2.0","links":{"self":"x"},"rules":[{"local":[{"group":{"id":"g-1"}}],"remote":[{"type":"REMOTE_USER","extra":1}]}]}`
)

func pythonMappingRules() []federation.MappingRule {
	return []federation.MappingRule{{
		Local:  []federation.RuleLocal{{Group: &federation.Group{ID: "g-1"}}},
		Remote: []federation.RuleRemote{{Type: "REMOTE_USER"}},
	}}
}

// delete_mapping defaults to ignore_missing=True; resource.IgnoreMissing gives
// the generated DeleteMapping that default.
func TestPythonMappingDeleteIgnoreMissing(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		code   int
		ignore bool
		ok     func(error) bool
	}{
		{"ignore 204", 204, true, func(err error) bool { return err == nil }},
		{"ignore 404", 404, true, func(err error) bool { return err == nil }},
		{"ignore 409", 409, true, func(err error) bool { return gophercloud.ResponseCodeIs(err, http.StatusConflict) }},
		{"strict 404", 404, false, func(err error) bool { return gophercloud.ResponseCodeIs(err, http.StatusNotFound) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var recorder pythonMappingRecorder
			api := pythonMappingAPI(t, &recorder, func(*http.Request) *http.Response { return pythonMappingWire(tc.code, "") })
			err := api.DeleteMapping(ctx, "ACME")
			if tc.ignore {
				err = resource.IgnoreMissing(err)
			}
			if !tc.ok(err) {
				t.Fatal(err)
			}
			want := []pythonMappingCall{{http.MethodDelete, pythonMappingBase + "/ACME", "", ""}}
			if got := recorder.snapshot(); !reflect.DeepEqual(got, want) {
				t.Fatalf("%+v", got)
			}
		})
	}
}

// The mapping routes match Python, but the typed Mapping drops schema_version
// and unknown rule keys, update always sends rules, and list takes no query.
func TestPythonMappingRequestsAndDecodeGaps(t *testing.T) {
	ctx := context.Background()
	var recorder pythonMappingRecorder
	api := pythonMappingAPI(t, &recorder, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodPut:
			return pythonMappingWire(201, `{"mapping":`+pythonMappingRow+`}`)
		case req.URL.Path == pythonMappingBase:
			return pythonMappingWire(200, `{"mappings":[`+pythonMappingRow+`],"links":{"next":null}}`)
		}
		return pythonMappingWire(200, `{"mapping":`+pythonMappingRow+`}`)
	})
	// create_mapping(id="ACME", rules=[...], schema_version="2.0") keeps id in the body.
	created, err := api.CreateMapping(ctx, "ACME", federation.CreateMappingOpts{Rules: pythonMappingRules()},
		federation.WithCreateMappingField("id", "ACME"), federation.WithCreateMappingField("schema_version", "2.0"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := api.GetMapping(ctx, "ACME")
	if err != nil {
		t.Fatal(err)
	}
	// update_mapping("ACME", schema_version="2.0") sends only schema_version in Python.
	updated, err := api.UpdateMapping(ctx, "ACME", federation.UpdateMappingOpts{}, federation.WithUpdateMappingField("schema_version", "2.0"))
	if err != nil {
		t.Fatal(err)
	}
	var listed []*federation.Mapping
	for value, err := range api.ListMappings(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		listed = append(listed, value)
	}
	if len(listed) != 1 {
		t.Fatal(listed)
	}
	for _, value := range []*federation.Mapping{created, got, updated, listed[0]} {
		if value == nil || value.ID != "ACME" || !reflect.DeepEqual(value.Rules, pythonMappingRules()) {
			t.Fatalf("%+v", value)
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), "schema_version") || strings.Contains(string(encoded), "extra") {
			t.Fatal(string(encoded))
		}
	}
	if _, ok := reflect.TypeOf(federation.Mapping{}).FieldByName("SchemaVersion"); ok {
		t.Fatal("Mapping now exposes schema_version")
	}
	// rules is a core field, so Python's raw rule list cannot be sent as JSON.
	if _, err := api.UpdateMapping(ctx, "ACME", federation.UpdateMappingOpts{}, federation.WithUpdateMappingField("rules", []any{map[string]any{"remote": []any{}}})); err == nil {
		t.Fatal("raw rules accepted")
	}
	rules := `[{"local":[{"group":{"id":"g-1"}}],"remote":[{"type":"REMOTE_USER"}]}]`
	want := []pythonMappingCall{
		{http.MethodPut, pythonMappingBase + "/ACME", "", `{"mapping":{"id":"ACME","rules":` + rules + `,"schema_version":"2.0"}}`},
		{http.MethodGet, pythonMappingBase + "/ACME", "", ""},
		{http.MethodPatch, pythonMappingBase + "/ACME", "", `{"mapping":{"rules":null,"schema_version":"2.0"}}`},
		{http.MethodGet, pythonMappingBase, "", ""},
	}
	if got := recorder.snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("%+v", got)
	}
}
