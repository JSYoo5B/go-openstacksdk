package groups_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonGroupFindRecordFetchControlsCollectionDeclaredSeedAndLocation(t *testing.T) {
	calls := 0
	body := `{"selected":{"id":"actual","name":false,"x_extension":{"n":9007199254740993},"location":{"cloud":"forged"}},"group":{"id":"wrong"}}`
	api, _ := groupRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.URL.Path != "/keystone/v3/alternate-groups" || r.URL.Query().Get("domain_id") != "d" || r.Header.Get("X-Typed") != "kept" || r.Header.Get("Cache-Control") != "" {
			t.Fatal(r.URL, r.Header)
		}
		for _, key := range []string{"requires_id", "base_path", "skip_cache", "error_message", "resource_response_key"} {
			if r.URL.Query().Has(key) {
				t.Fatal(r.URL)
			}
		}
		return groupRecordWire(203, body)
	})
	cloudName := "configured"
	location := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`)}}
	row, err := api.FindRecord(context.Background(), "demo",
		groups.WithRecordFindQuery("requires_id", false), groups.WithRecordFindQuery("base_path", "/alternate-groups"),
		groups.WithRecordFindQuery("skip_cache", true), groups.WithRecordFindQuery("error_message", "custom missing"),
		groups.WithRecordFindQuery("resource_response_key", "selected"), groups.WithRecordFindQuery("domain_id", "d"),
		groups.WithRecordFindQuery("description", json.RawMessage(`{"seed":1}`)), groups.WithRecordFindHeaders(map[string]string{"X-Typed": "kept"}), groups.WithRecordFindLocation(location))
	if err != nil || row == nil || calls != 1 || row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "group-proof" || string(row.Envelope) != body {
		t.Fatal(row, err, calls)
	}
	for key, want := range map[string]string{"id": `"actual"`, "name": "false", "description": `{"seed":1}`, "domain_id": `"d"`} {
		groupRecordField(t, row, key, want)
	}
	for _, key := range []string{"requires_id", "base_path", "skip_cache", "error_message", "resource_response_key", "x_extension"} {
		if _, present := row.Resource.Body[key]; present {
			t.Fatal(key, row.Resource)
		}
	}
	var loc map[string]json.RawMessage
	if err := json.Unmarshal(row.Resource.Body["location"], &loc); err != nil || string(loc["cloud"]) != `"configured"` {
		t.Fatal(loc, err)
	}
	if string(row.Wire.Body["location"]) != `{"cloud":"forged"}` || string(row.Wire.Body["x_extension"]) != `{"n":9007199254740993}` {
		t.Fatal(row.Wire)
	}
}

func TestPythonGroupFindRecordFetchControlsCustomEnvelopeFailureKeepsPhysicalEvidence(t *testing.T) {
	calls := 0
	body := `{"selected":[],"group":{"id":"wrong"}}`
	api, _ := groupRecordAPI(t, func(r *http.Request) *http.Response { calls++; return groupRecordWire(202, body) })
	row, err := api.FindRecord(context.Background(), "demo", groups.WithRecordFindQuery("resource_response_key", "selected"))
	var proof *resource.ResponseError
	if row != nil || calls != 1 || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "group-proof" {
		t.Fatal(row, err, proof, calls)
	}
}

func TestPythonGroupFindRecordFetchControlsHeadersAndListRouteAreStageSpecific(t *testing.T) {
	calls := 0
	api, _ := groupRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.Header.Get("X-Typed") != "kept" {
			t.Fatal(r.Header)
		}
		if calls == 1 {
			if r.URL.Path != "/keystone/v3/alternate-groups/demo" || !reflect.DeepEqual(r.URL.Query()["headers"], []string{"Accept", "X-Raw"}) || r.Header.Get("X-Raw") != "typed" {
				t.Fatal(r.URL, r.Header)
			}
			return groupRecordWire(404, `{}`)
		}
		if r.URL.Path != "/keystone/v3/list-groups" || r.URL.Query().Get("name") != "demo" || r.URL.Query().Get("domain_id") != "d" || r.URL.Query().Has("headers") || r.Header.Get("X-Raw") != "raw" || r.Header.Get("Accept") != "application/vnd.group+json" {
			t.Fatal(r.URL, r.Header)
		}
		return groupRecordWire(200, `{"groups":[{"id":"g","name":"demo","description":false,"domain_id":3,"x_extension":7}]}`)
	})
	row, err := api.FindRecord(context.Background(), "demo", groups.WithRecordFindQuery("base_path", "/alternate-groups"), groups.WithRecordFindListBasePath("/list-groups"), groups.WithRecordFindQuery("headers", json.RawMessage(`{"Accept":"application/vnd.group+json","X-Raw":"raw"}`)), groups.WithRecordFindHeaders(map[string]string{"x-raw": "typed", "X-Typed": "kept"}), groups.WithRecordFindQuery("domain_id", "d"))
	if err != nil || row == nil || calls != 2 {
		t.Fatal(row, err, calls)
	}
	for key, want := range map[string]string{"id": `"g"`, "description": "false", "domain_id": "3"} {
		groupRecordField(t, row, key, want)
	}
	// Fallback constructs each list row anew, so fetch's unknown seed is absent.
	if row.Resource.Body["headers"] != nil || row.Resource.Body["base_path"] != nil {
		t.Fatal(row.Resource)
	}
}

func TestPythonGroupFindRecordFetchControlsBadHeadersWaitForFallback(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		t.Run(map[bool]string{false: "successful-get", true: "fallback"}[fallback], func(t *testing.T) {
			calls := 0
			api, _ := groupRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				if r.URL.Query().Get("headers") != "X-Raw" {
					t.Fatal(r.URL)
				}
				if fallback {
					return groupRecordWire(403, `{}`)
				}
				return groupRecordWire(200, `{"group":{"id":"g","description":null,"domain_id":null}}`)
			})
			row, err := api.FindRecord(context.Background(), "demo", groups.WithRecordFindQuery("headers", json.RawMessage(`{"X-Raw":7}`)))
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
			if row.Resource.Body["headers"] != nil {
				t.Fatal(row.Resource)
			}
			groupRecordField(t, row, "description", "null")
			groupRecordField(t, row, "domain_id", "null")
		})
	}
}

func TestPythonGroupFindRecordFetchControlsOwnOptionsRemainStatelessAndKeepErrorCause(t *testing.T) {
	t.Run("options owned and uncached", func(t *testing.T) {
		query := map[string]json.RawMessage{"base_path": json.RawMessage(`"/owned-groups"`), "requires_id": json.RawMessage(`false`), "resource_response_key": json.RawMessage(`"selected"`), "skip_cache": json.RawMessage(`false`)}
		headers := map[string]string{"X-Owned": "original"}
		option := groups.WithRecordFindOpts(groups.RecordFindOpts{Query: query, Headers: headers})
		copy(query["base_path"], []byte(`"/other-groups"`))
		query["resource_response_key"] = json.RawMessage(`"other"`)
		headers["X-Owned"] = "mutated"
		calls := 0
		api, _ := groupRecordAPI(t, func(r *http.Request) *http.Response {
			calls++
			if r.URL.Path != "/keystone/v3/owned-groups" || r.URL.RawQuery != "" || r.Header.Get("X-Owned") != "original" || r.Header.Get("Cache-Control") != "" {
				t.Fatal(r.URL, r.Header)
			}
			if calls == 1 {
				return groupRecordWire(200, `{"selected":{"id":"first"}}`)
			}
			return groupRecordWire(200, `{"selected":{"id":"second"}}`)
		})
		for _, want := range []string{`"first"`, `"second"`} {
			row, err := api.FindRecord(context.Background(), "demo", option)
			if err != nil || row == nil {
				t.Fatal(row, err)
			}
			groupRecordField(t, row, "id", want)
			if row.Resource.Body["skip_cache"] != nil {
				t.Fatal(row.Resource)
			}
		}
		if calls != 2 {
			t.Fatal(calls)
		}
	})
	t.Run("custom error retains native response", func(t *testing.T) {
		calls := 0
		body := `{"error":{"message":"upstream"}}`
		api, _ := groupRecordAPI(t, func(r *http.Request) *http.Response {
			calls++
			if r.URL.Query().Has("error_message") {
				t.Fatal(r.URL)
			}
			return groupRecordWire(500, body)
		})
		row, err := api.FindRecord(context.Background(), "demo", groups.WithRecordFindQuery("error_message", "group lookup failed"))
		var native gophercloud.ErrUnexpectedResponseCode
		if row != nil || calls != 1 || !errors.As(err, &native) || native.Actual != 500 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "group-proof" || !strings.Contains(err.Error(), "group lookup failed") {
			t.Fatal(row, err, native, calls)
		}
	})
}

func TestPythonGroupFindRecordFetchControlsPhysicalFaultsNeverFallback(t *testing.T) {
	for _, mode := range []string{"read", "close", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			cause := errors.New("group controlled physical " + mode)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			closes := 0
			body := &groupFaultBody{closes: &closes}
			if mode == "read" {
				body.readErr = cause
			}
			if mode == "close" {
				body.closeErr = cause
			}
			if mode == "cancel" {
				body.close = func() { cancel(cause) }
			}

			api, _ := groupRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				response := groupRecordWire(404, `{}`)
				response.Body = body
				return response
			})
			row, err := api.FindRecord(ctx, "demo", groups.WithRecordFindQuery("requires_id", false), groups.WithRecordFindQuery("skip_cache", true), groups.WithRecordFindQuery("error_message", "controlled group lookup"))
			if row != nil || calls != 1 || closes != 1 || !errors.Is(err, cause) {
				t.Fatal(row, err, calls, closes)
			}
		})
	}
}
