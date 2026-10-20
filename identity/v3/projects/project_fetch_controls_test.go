package projects_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonProjectFindRecordFetchControlsCollectionProjectionAndConstructorSeed(t *testing.T) {
	calls := 0
	body := `{"selected":{"id":"actual","enabled":0,"is_domain":"","options":5,"tags":"blue","x_extension":{"n":9007199254740993}},"project":{"id":"wrong"}}`
	api, _ := projectRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.URL.Path != "/keystone/v3/alternate-projects" || r.URL.Query().Get("domain_id") != "d" || r.Header.Get("X-Typed") != "kept" || r.Header.Get("Cache-Control") != "" {
			t.Fatal(r.URL, r.Header)
		}
		for _, key := range []string{"requires_id", "base_path", "skip_cache", "error_message", "resource_response_key"} {
			if r.URL.Query().Has(key) {
				t.Fatal(r.URL)
			}
		}
		return projectRecordWire(203, body)
	})
	row, err := api.FindRecord(context.Background(), "demo",
		projects.WithRecordFindQuery("requires_id", false), projects.WithRecordFindQuery("base_path", "/alternate-projects"),
		projects.WithRecordFindQuery("skip_cache", true), projects.WithRecordFindQuery("error_message", "custom missing"),
		projects.WithRecordFindQuery("resource_response_key", "selected"), projects.WithRecordFindQuery("domain_id", "d"),
		projects.WithRecordFindQuery("description", "seeded"), projects.WithRecordFindHeaders(map[string]string{"X-Typed": "kept"}))
	if err != nil || row == nil || calls != 1 || row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "project-proof" || string(row.Envelope) != body {
		t.Fatal(row, err, calls)
	}
	for key, want := range map[string]string{"id": `"actual"`, "is_enabled": "false", "is_domain": "false", "options": "{}", "tags": `["blue"]`, "description": `"seeded"`, "domain_id": `"d"`, "x_extension": `{"n":9007199254740993}`, "requires_id": "false", "base_path": `"/alternate-projects"`, "skip_cache": "true", "error_message": `"custom missing"`, "resource_response_key": `"selected"`} {
		projectRecordField(t, row, key, want)
	}
	// Project allows unknown constructor attributes; they are not physical response fields.
	if row.Wire.Body["skip_cache"] != nil || string(row.Wire.Body["options"]) != "5" || string(row.Wire.Body["enabled"]) != "0" {
		t.Fatal(row.Wire)
	}
}

func TestPythonProjectFindRecordFetchControlsCustomEnvelopeFailureKeepsPhysicalEvidence(t *testing.T) {
	calls := 0
	body := `{"selected":[],"project":{"id":"wrong"}}`
	api, _ := projectRecordAPI(t, func(r *http.Request) *http.Response { calls++; return projectRecordWire(202, body) })
	row, err := api.FindRecord(context.Background(), "demo", projects.WithRecordFindQuery("resource_response_key", "selected"))
	var proof *resource.ResponseError
	if row != nil || calls != 1 || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "project-proof" {
		t.Fatal(row, err, proof, calls)
	}
}

func TestPythonProjectFindRecordFetchControlsHeadersAndListRouteAreStageSpecific(t *testing.T) {
	calls := 0
	api, _ := projectRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.Header.Get("X-Typed") != "kept" {
			t.Fatal(r.Header)
		}
		if calls == 1 {
			if r.URL.Path != "/keystone/v3/alternate-projects/demo" || !reflect.DeepEqual(r.URL.Query()["headers"], []string{"Accept", "X-Raw"}) || r.Header.Get("X-Raw") != "typed" {
				t.Fatal(r.URL, r.Header)
			}
			return projectRecordWire(404, `{}`)
		}
		if r.URL.Path != "/keystone/v3/list-projects" || r.URL.Query().Get("name") != "demo" || r.URL.Query().Get("domain_id") != "d" || r.URL.Query().Has("headers") || r.Header.Get("X-Raw") != "raw" || r.Header.Get("Accept") != "application/vnd.project+json" {
			t.Fatal(r.URL, r.Header)
		}
		return projectRecordWire(200, `{"projects":[{"id":"p","name":"demo","enabled":0,"options":7,"tags":"blue"}]}`)
	})
	row, err := api.FindRecord(context.Background(), "demo", projects.WithRecordFindQuery("base_path", "/alternate-projects"), projects.WithRecordFindListBasePath("/list-projects"), projects.WithRecordFindQuery("headers", json.RawMessage(`{"Accept":"application/vnd.project+json","X-Raw":"raw"}`)), projects.WithRecordFindHeaders(map[string]string{"x-raw": "typed", "X-Typed": "kept"}), projects.WithRecordFindQuery("domain_id", "d"))
	if err != nil || row == nil || calls != 2 {
		t.Fatal(row, err, calls)
	}
	for key, want := range map[string]string{"id": `"p"`, "is_enabled": "false", "options": "{}", "tags": `["blue"]`} {
		projectRecordField(t, row, key, want)
	}
	// Fallback constructs each list row anew, so fetch's unknown seed is absent.
	if row.Resource.Body["headers"] != nil || row.Resource.Body["base_path"] != nil {
		t.Fatal(row.Resource)
	}
}

func TestPythonProjectFindRecordFetchControlsBadHeadersWaitForFallback(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		t.Run(map[bool]string{false: "successful-get", true: "fallback"}[fallback], func(t *testing.T) {
			calls := 0
			api, _ := projectRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				if r.URL.Query().Get("headers") != "X-Raw" {
					t.Fatal(r.URL)
				}
				if fallback {
					return projectRecordWire(403, `{}`)
				}
				return projectRecordWire(200, `{"project":{"id":"p","options":null,"tags":null}}`)
			})
			row, err := api.FindRecord(context.Background(), "demo", projects.WithRecordFindQuery("headers", json.RawMessage(`{"X-Raw":7}`)))
			if calls != 1 {
				t.Fatal(calls)
			}
			if fallback {
				if row != nil || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(row, err)
				}
				return
			}
			if err != nil || row == nil {
				t.Fatal(row, err)
			}
			projectRecordField(t, row, "headers", `{"X-Raw":7}`)
			projectRecordField(t, row, "options", "null")
			projectRecordField(t, row, "tags", "null")
		})
	}
}

func TestPythonProjectFindRecordFetchControlsOwnOptionsRemainStatelessAndKeepErrorCause(t *testing.T) {
	t.Run("options owned and uncached", func(t *testing.T) {
		query := map[string]json.RawMessage{"base_path": json.RawMessage(`"/owned-projects"`), "requires_id": json.RawMessage(`false`), "resource_response_key": json.RawMessage(`"selected"`), "skip_cache": json.RawMessage(`false`)}
		headers := map[string]string{"X-Owned": "original"}
		option := projects.WithRecordFindOpts(projects.RecordFindOpts{Query: query, Headers: headers})
		copy(query["base_path"], []byte(`"/other-projects"`))
		query["resource_response_key"] = json.RawMessage(`"other"`)
		headers["X-Owned"] = "mutated"
		calls := 0
		api, _ := projectRecordAPI(t, func(r *http.Request) *http.Response {
			calls++
			if r.URL.Path != "/keystone/v3/owned-projects" || r.URL.RawQuery != "" || r.Header.Get("X-Owned") != "original" || r.Header.Get("Cache-Control") != "" {
				t.Fatal(r.URL, r.Header)
			}
			if calls == 1 {
				return projectRecordWire(200, `{"selected":{"id":"first"}}`)
			}
			return projectRecordWire(200, `{"selected":{"id":"second"}}`)
		})
		for _, want := range []string{`"first"`, `"second"`} {
			row, err := api.FindRecord(context.Background(), "demo", option)
			if err != nil || row == nil {
				t.Fatal(row, err)
			}
			projectRecordField(t, row, "id", want)
			projectRecordField(t, row, "skip_cache", "false")
		}
		if calls != 2 {
			t.Fatal(calls)
		}
	})
	t.Run("custom error retains native response", func(t *testing.T) {
		calls := 0
		body := `{"error":{"message":"upstream"}}`
		api, _ := projectRecordAPI(t, func(r *http.Request) *http.Response {
			calls++
			if r.URL.Query().Has("error_message") {
				t.Fatal(r.URL)
			}
			return projectRecordWire(500, body)
		})
		row, err := api.FindRecord(context.Background(), "demo", projects.WithRecordFindQuery("error_message", "project lookup failed"))
		var native gophercloud.ErrUnexpectedResponseCode
		if row != nil || calls != 1 || !errors.As(err, &native) || native.Actual != 500 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "project-proof" || !strings.Contains(err.Error(), "project lookup failed") {
			t.Fatal(row, err, native, calls)
		}
	})
}

func TestPythonProjectFindRecordFetchControlsPhysicalFaultsNeverFallback(t *testing.T) {
	for _, mode := range []string{"read", "close", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			cause := errors.New("project controlled physical " + mode)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			body := &projectRecordFaultBody{text: `{"error":{"message":"unclean missing"}}`}
			if mode == "read" {
				body.read = cause
			}
			if mode == "close" {
				body.close = cause
			}
			if mode == "cancel" {
				body.hook = func() { cancel(cause) }
			}
			api, _ := projectRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				response := projectRecordWire(404, body.text)
				response.Body = body
				return response
			})
			row, err := api.FindRecord(ctx, "demo", projects.WithRecordFindQuery("requires_id", false), projects.WithRecordFindQuery("skip_cache", true), projects.WithRecordFindQuery("error_message", "controlled project lookup"))
			if row != nil || calls != 1 || body.closed != 1 || !errors.Is(err, cause) {
				t.Fatal(row, err, calls, body.closed)
			}
		})
	}
}
