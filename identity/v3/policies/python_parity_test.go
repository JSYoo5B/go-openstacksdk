package policies_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/policies"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonPolicyTransport func(*http.Request) (*http.Response, error)

func (transport pythonPolicyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonPolicyWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type pythonPolicyCall struct{ method, path, query, body, version string }

type pythonPolicyRecorder struct {
	mu    sync.Mutex
	calls []pythonPolicyCall
}

func (r *pythonPolicyRecorder) snapshot() []pythonPolicyCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]pythonPolicyCall(nil), r.calls...)
}

func pythonPolicyAPI(t *testing.T, recorder *pythonPolicyRecorder, reply func(*http.Request) *http.Response) (*policies.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonPolicyTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		recorder.mu.Lock()
		recorder.calls = append(recorder.calls, pythonPolicyCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("OpenStack-API-Version")})
		recorder.mu.Unlock()
		return reply(req), nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return policies.New(client), cloud
}

const (
	pythonPolicyBase = "/keystone/v3/policies"
	pythonPolicyRow  = `{"id":"pol-1","type":"application/json","blob":"{\"rule\":\"x\"}","project_id":"p","user_id":"u","links":{"self":"x"}}`
)

func pythonPolicyCheck(t *testing.T, got *policies.Policy) {
	t.Helper()
	// Python exposes project_id and user_id as attributes; Go keeps them in Extra.
	if got == nil || got.ID != "pol-1" || got.Type != "application/json" || got.Blob != `{"rule":"x"}` ||
		!reflect.DeepEqual(got.Extra, map[string]any{"project_id": "p", "user_id": "u"}) || got.Links["self"] != "x" {
		t.Fatalf("%+v", got)
	}
}

// create_policy, get_policy and update_policy send the same POST, GET and PATCH
// bodies Python builds from the dirty Policy attributes.
func TestPythonPolicyCreateGetUpdate(t *testing.T) {
	ctx := context.Background()
	var recorder pythonPolicyRecorder
	api, _ := pythonPolicyAPI(t, &recorder, func(req *http.Request) *http.Response {
		switch req.Method {
		case http.MethodPost:
			return pythonPolicyWire(201, `{"policy":`+pythonPolicyRow+`}`)
		case http.MethodGet:
			if strings.HasSuffix(req.URL.Path, "/missing") {
				return pythonPolicyWire(404, `{"error":{"code":404}}`)
			}
		}
		return pythonPolicyWire(200, `{"policy":`+pythonPolicyRow+`}`)
	})
	// create_policy(type=..., blob=..., project_id="p", user_id="u")
	created, err := api.Create(ctx, policies.CreateOpts{Type: "application/json", Blob: []byte(`{"rule":"x"}`), Extra: map[string]any{"project_id": "p", "user_id": "u"}})
	if err != nil {
		t.Fatal(err)
	}
	pythonPolicyCheck(t, created)
	// get_policy("pol-1"), directly and through the collection.
	got, err := api.Get(ctx, "pol-1")
	if err != nil {
		t.Fatal(err)
	}
	pythonPolicyCheck(t, got)
	found, err := api.Find(ctx, resource.ID("pol-1"))
	if err != nil {
		t.Fatal(err)
	}
	pythonPolicyCheck(t, found)
	// get_policy raises NotFoundException; Go reports the 404 on both paths.
	if _, err := api.Get(ctx, "missing"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	if _, err := api.Find(ctx, resource.ID("missing")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	// update_policy("pol-1", type="text/plain", blob="", user_id=None)
	updated, err := api.Update(ctx, "pol-1", policies.UpdateOpts{Type: "text/plain", Extra: map[string]any{"user_id": nil}}, policies.WithUpdateField("blob", ""))
	if err != nil {
		t.Fatal(err)
	}
	pythonPolicyCheck(t, updated)
	// update_policy("pol-1") sends nothing in Python; Go still sends an empty PATCH.
	if _, err := api.Update(ctx, "pol-1", policies.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	want := []pythonPolicyCall{
		{http.MethodPost, pythonPolicyBase, "", `{"policy":{"blob":"{\"rule\":\"x\"}","project_id":"p","type":"application/json","user_id":"u"}}`, ""},
		{http.MethodGet, pythonPolicyBase + "/pol-1", "", "", ""},
		{http.MethodGet, pythonPolicyBase + "/pol-1", "", "", ""},
		{http.MethodGet, pythonPolicyBase + "/missing", "", "", ""},
		{http.MethodGet, pythonPolicyBase + "/missing", "", "", ""},
		{http.MethodPatch, pythonPolicyBase + "/pol-1", "", `{"policy":{"blob":"","type":"text/plain","user_id":null}}`, ""},
		{http.MethodPatch, pythonPolicyBase + "/pol-1", "", `{"policy":{}}`, ""},
	}
	if got := recorder.snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("%+v", got)
	}
	// Keystone requires type and blob; Go rejects their absence before HTTP.
	if _, err := api.Create(ctx, policies.CreateOpts{Blob: []byte("b")}); err == nil {
		t.Fatal("missing type accepted")
	}
	if len(recorder.snapshot()) != len(want) {
		t.Fatal(recorder.snapshot())
	}
}

// delete_policy defaults to ignore_missing=True.
func TestPythonPolicyDeleteIgnoreMissing(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		code    int
		call    func(*policies.API) error
		wantErr func(error) bool
	}{
		{"Remove 204", 204, func(api *policies.API) error { return api.Remove(ctx, resource.ID("pol-1")) }, func(err error) bool { return err == nil }},
		{"Remove 404", 404, func(api *policies.API) error { return api.Remove(ctx, resource.ID("pol-1")) }, func(err error) bool { return err == nil }},
		{"Remove 409", 409, func(api *policies.API) error { return api.Remove(ctx, resource.ID("pol-1")) }, func(err error) bool {
			return gophercloud.ResponseCodeIs(err, http.StatusConflict)
		}},
		{"Remove strict 404", 404, func(api *policies.API) error {
			return api.Remove(ctx, resource.ID("pol-1"), resource.WithMissingError())
		}, func(err error) bool { return errors.Is(err, resource.ErrNotFound) }},
		{"IgnoreMissing 204", 204, func(api *policies.API) error { return resource.IgnoreMissing(api.Delete(ctx, "pol-1")) }, func(err error) bool { return err == nil }},
		{"IgnoreMissing 404", 404, func(api *policies.API) error { return resource.IgnoreMissing(api.Delete(ctx, "pol-1")) }, func(err error) bool { return err == nil }},
		{"IgnoreMissing 409", 409, func(api *policies.API) error { return resource.IgnoreMissing(api.Delete(ctx, "pol-1")) }, func(err error) bool {
			return gophercloud.ResponseCodeIs(err, http.StatusConflict)
		}},
		{"Delete 404", 404, func(api *policies.API) error { return api.Delete(ctx, "pol-1") }, func(err error) bool {
			return gophercloud.ResponseCodeIs(err, http.StatusNotFound)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var recorder pythonPolicyRecorder
			api, _ := pythonPolicyAPI(t, &recorder, func(*http.Request) *http.Response { return pythonPolicyWire(tc.code, "") })
			if err := tc.call(api); !tc.wantErr(err) {
				t.Fatal(err)
			}
			want := []pythonPolicyCall{{http.MethodDelete, pythonPolicyBase + "/pol-1", "", "", ""}}
			if got := recorder.snapshot(); !reflect.DeepEqual(got, want) {
				t.Fatalf("%+v", got)
			}
		})
	}
}

// policies(): the bare call and the type filter match Python's results, but the
// local Body filters Python applies have no Go equivalent.
func TestPythonPolicyListFilters(t *testing.T) {
	ctx := context.Background()
	var recorder pythonPolicyRecorder
	var cloud *testcloud.Cloud
	api, cloud := pythonPolicyAPI(t, &recorder, func(req *http.Request) *http.Response {
		if req.URL.Path == "/next/policies" {
			return pythonPolicyWire(200, `{"policies":[{"id":"pol-2","type":"text/plain","blob":"b"}],"links":{"next":null}}`)
		}
		return pythonPolicyWire(200, `{"policies":[`+pythonPolicyRow+`],"links":{"next":"`+cloud.Server.URL+`/next/policies?marker=pol-1"}}`)
	})
	var ids []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	all, err := api.All(ctx, resource.WithQuery("type", "application/json"), resource.WithQuery("limit", "1"))
	if err != nil || len(all) != 2 {
		t.Fatal(all, err)
	}
	if !reflect.DeepEqual(ids, []string{"pol-1", "pol-2"}) {
		t.Fatal(ids)
	}
	// Python policies(user_id="u") filters locally after GET /policies.
	if _, err := api.All(ctx, resource.WithBodyFilter("user_id", "u")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := api.All(ctx, resource.WithName("x")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	want := []pythonPolicyCall{
		{http.MethodGet, pythonPolicyBase, "", "", ""},
		{http.MethodGet, "/next/policies", "marker=pol-1", "", ""},
		{http.MethodGet, pythonPolicyBase, "limit=1&type=application%2Fjson", "", ""},
		{http.MethodGet, "/next/policies", "marker=pol-1", "", ""},
	}
	if got := recorder.snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("%+v", got)
	}
}

// find_policy falls back to listing after GET 400/403/404 and matches id or
// name; the Go collection only offers the ID GET.
func TestPythonPolicyFindHasNoListFallback(t *testing.T) {
	ctx := context.Background()
	for _, code := range []int{400, 403, 404} {
		var recorder pythonPolicyRecorder
		api, _ := pythonPolicyAPI(t, &recorder, func(*http.Request) *http.Response { return pythonPolicyWire(code, `{}`) })
		got, err := api.Find(ctx, resource.ID("pol-1"), resource.WithIgnoreMissing())
		switch {
		case code == 404 && (err != nil || got != nil):
			t.Fatal(code, got, err)
		case code != 404 && !gophercloud.ResponseCodeIs(err, code):
			t.Fatal(code, err)
		}
		want := []pythonPolicyCall{{http.MethodGet, pythonPolicyBase + "/pol-1", "", "", ""}}
		if got := recorder.snapshot(); !reflect.DeepEqual(got, want) {
			t.Fatalf("%d %+v", code, got)
		}
	}
	var recorder pythonPolicyRecorder
	api, _ := pythonPolicyAPI(t, &recorder, func(*http.Request) *http.Response { return pythonPolicyWire(200, `{"policies":[]}`) })
	if _, err := api.Find(ctx, resource.Name("pol-1"), resource.WithIgnoreMissing()); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := api.Resources.FindIdentity(ctx, "pol-1"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if got := recorder.snapshot(); len(got) != 0 {
		t.Fatalf("%+v", got)
	}
}
