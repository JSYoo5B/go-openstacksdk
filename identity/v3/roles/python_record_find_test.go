package roles_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func roleFindField(t *testing.T, r *roles.RoleRecord, key, want string) {
	t.Helper()
	if r == nil || r.Resource == nil || string(r.Resource.Body[key]) != want {
		t.Fatalf("%s want %s: %+v", key, want, r)
	}
}

func TestPythonRoleFindRecordCollectionSeedProjectionAndLocation(t *testing.T) {
	calls := 0
	body := `{"selected":{"id":"actual","name":false,"options":7,"x_extension":{"n":9007199254740993},"location":{"cloud":"forged"}},"role":{"id":"wrong"}}`
	api, _ := roleRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.URL.Path != "/keystone/v3/alternate-roles" || r.URL.Query().Get("domain_id") != "d" || r.Header.Get("X-Typed") != "kept" || r.Header.Get("Cache-Control") != "" {
			t.Fatal(r.URL, r.Header)
		}
		for _, key := range []string{"requires_id", "base_path", "skip_cache", "error_message", "resource_response_key"} {
			if r.URL.Query().Has(key) {
				t.Fatal(r.URL)
			}
		}
		return roleRecordWire(203, body)
	})
	cloudName := "configured"
	location := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`)}}
	row, err := api.FindRecord(context.Background(), "demo",
		roles.WithRecordFindQuery("requires_id", false), roles.WithRecordFindQuery("base_path", "/alternate-roles"),
		roles.WithRecordFindQuery("skip_cache", true), roles.WithRecordFindQuery("error_message", "custom missing"),
		roles.WithRecordFindQuery("resource_response_key", "selected"), roles.WithRecordFindQuery("domain_id", "d"),
		roles.WithRecordFindQuery("description", json.RawMessage(`{"seed":1}`)), roles.WithRecordFindHeaders(map[string]string{"X-Typed": "kept"}), roles.WithRecordFindLocation(location))
	if err != nil || row == nil || calls != 1 || row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "role-proof" || string(row.Envelope) != body {
		t.Fatal(row, err, calls)
	}
	for key, want := range map[string]string{"id": `"actual"`, "name": "false", "description": `{"seed":1}`, "domain_id": `"d"`, "options": "{}"} {
		roleFindField(t, row, key, want)
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

func TestPythonRoleFindRecordCustomEnvelopeFailureKeepsPhysicalEvidence(t *testing.T) {
	calls := 0
	body := `{"selected":[],"role":{"id":"wrong"}}`
	api, _ := roleRecordAPI(t, func(r *http.Request) *http.Response { calls++; return roleRecordWire(202, body) })
	row, err := api.FindRecord(context.Background(), "demo", roles.WithRecordFindQuery("resource_response_key", "selected"))
	var proof *resource.ResponseError
	if row != nil || calls != 1 || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "role-proof" {
		t.Fatal(row, err, proof, calls)
	}
}

func TestPythonRoleFindRecordHeadersAndListRouteAreStageSpecific(t *testing.T) {
	calls := 0
	api, _ := roleRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.Header.Get("X-Typed") != "kept" {
			t.Fatal(r.Header)
		}
		if calls == 1 {
			if r.URL.Path != "/keystone/v3/alternate-roles/demo" || !reflect.DeepEqual(r.URL.Query()["headers"], []string{"Accept", "X-Raw"}) || r.Header.Get("X-Raw") != "typed" {
				t.Fatal(r.URL, r.Header)
			}
			return roleRecordWire(404, `{}`)
		}
		if r.URL.Path != "/keystone/v3/list-roles" || r.URL.Query().Get("name") != "demo" || r.URL.Query().Get("domain_id") != "d" || r.URL.Query().Has("headers") || r.Header.Get("X-Raw") != "raw" || r.Header.Get("Accept") != "application/vnd.role+json" {
			t.Fatal(r.URL, r.Header)
		}
		return roleRecordWire(200, `{"roles":[{"id":"g","name":"demo","description":false,"domain_id":3,"x_extension":7}]}`)
	})
	row, err := api.FindRecord(context.Background(), "demo", roles.WithRecordFindQuery("base_path", "/alternate-roles"), roles.WithRecordFindListBasePath("/list-roles"), roles.WithRecordFindQuery("headers", json.RawMessage(`{"Accept":"application/vnd.role+json","X-Raw":"raw"}`)), roles.WithRecordFindHeaders(map[string]string{"x-raw": "typed", "X-Typed": "kept"}), roles.WithRecordFindQuery("domain_id", "d"))
	if err != nil || row == nil || calls != 2 {
		t.Fatal(row, err, calls)
	}
	for key, want := range map[string]string{"id": `"g"`, "description": "false", "domain_id": "3"} {
		roleFindField(t, row, key, want)
	}
	// Fallback constructs each list row anew, so fetch's unknown seed is absent.
	if row.Resource.Body["headers"] != nil || row.Resource.Body["base_path"] != nil {
		t.Fatal(row.Resource)
	}
}

func TestPythonRoleFindRecordBadHeadersWaitForFallback(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		t.Run(map[bool]string{false: "successful-get", true: "fallback"}[fallback], func(t *testing.T) {
			calls := 0
			api, _ := roleRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				if r.URL.Query().Get("headers") != "X-Raw" {
					t.Fatal(r.URL)
				}
				if fallback {
					return roleRecordWire(403, `{}`)
				}
				return roleRecordWire(200, `{"role":{"id":"g","description":null,"domain_id":null}}`)
			})
			row, err := api.FindRecord(context.Background(), "demo", roles.WithRecordFindQuery("headers", json.RawMessage(`{"X-Raw":7}`)))
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
			roleFindField(t, row, "description", "null")
			roleFindField(t, row, "domain_id", "null")
		})
	}
}

func TestPythonRoleFindRecordOwnOptionsRemainStatelessAndKeepErrorCause(t *testing.T) {
	t.Run("options owned and uncached", func(t *testing.T) {
		query := map[string]json.RawMessage{"base_path": json.RawMessage(`"/owned-roles"`), "requires_id": json.RawMessage(`false`), "resource_response_key": json.RawMessage(`"selected"`), "skip_cache": json.RawMessage(`false`)}
		headers := map[string]string{"X-Owned": "original"}
		option := roles.WithRecordFindOpts(roles.RecordFindOpts{Query: query, Headers: headers})
		copy(query["base_path"], []byte(`"/other-roles"`))
		query["resource_response_key"] = json.RawMessage(`"other"`)
		headers["X-Owned"] = "mutated"
		calls := 0
		api, _ := roleRecordAPI(t, func(r *http.Request) *http.Response {
			calls++
			if r.URL.Path != "/keystone/v3/owned-roles" || r.URL.RawQuery != "" || r.Header.Get("X-Owned") != "original" || r.Header.Get("Cache-Control") != "" {
				t.Fatal(r.URL, r.Header)
			}
			if calls == 1 {
				return roleRecordWire(200, `{"selected":{"id":"first"}}`)
			}
			return roleRecordWire(200, `{"selected":{"id":"second"}}`)
		})
		for _, want := range []string{`"first"`, `"second"`} {
			row, err := api.FindRecord(context.Background(), "demo", option)
			if err != nil || row == nil {
				t.Fatal(row, err)
			}
			roleFindField(t, row, "id", want)
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
		api, _ := roleRecordAPI(t, func(r *http.Request) *http.Response {
			calls++
			if r.URL.Query().Has("error_message") {
				t.Fatal(r.URL)
			}
			return roleRecordWire(500, body)
		})
		row, err := api.FindRecord(context.Background(), "demo", roles.WithRecordFindQuery("error_message", "role lookup failed"))
		var native gophercloud.ErrUnexpectedResponseCode
		if row != nil || calls != 1 || !errors.As(err, &native) || native.Actual != 500 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "role-proof" || !strings.Contains(err.Error(), "role lookup failed") {
			t.Fatal(row, err, native, calls)
		}
	})
}

func TestPythonRoleFindRecordPhysicalFaultsNeverFallback(t *testing.T) {
	for _, mode := range []string{"read", "close", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			cause := errors.New("role controlled physical " + mode)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			body := &testcloud.Body{Data: `{}`}
			if mode == "read" {
				body.ReadErr = cause
			}
			if mode == "close" {
				body.CloseErr = cause
			}
			if mode == "cancel" {
				body.OnClose = func() { cancel(cause) }
			}

			api, _ := roleRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				response := roleRecordWire(404, `{}`)
				response.Body = body
				return response
			})
			row, err := api.FindRecord(ctx, "demo", roles.WithRecordFindQuery("requires_id", false), roles.WithRecordFindQuery("skip_cache", true), roles.WithRecordFindQuery("error_message", "controlled role lookup"))
			if row != nil || calls != 1 || body.Closes() != 1 || !errors.Is(err, cause) {
				t.Fatal(row, err, calls, body.Closes())
			}
		})
	}
}

func TestPythonRoleFindRecordGETSeedPresenceAndPhysicalIsolation(t *testing.T) {
	for _, tc := range []struct{ name, body, id, options string }{
		{"missing id seeded", `{"role":{"name":false,"options":7,"x_extension":1}}`, `"alias"`, `{}`},
		{"explicit null id", `{"role":{"id":null,"description":null,"options":null}}`, `null`, `null`},
		{"scalar id passive", `{"role":{"id":7,"options":{"flag":true}}}`, `7`, `{"flag":true}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			api, _ := roleRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				if r.URL.Path != "/keystone/v3/roles/alias" || r.URL.Query().Get("domain_id") != "seed-domain" || r.URL.Query().Get("x_seed") != "request-only" {
					t.Fatal(r.URL)
				}
				return roleRecordWire(203, tc.body)
			})
			row, err := api.FindRecord(context.Background(), "alias", roles.WithRecordFindQuery("domain_id", "seed-domain"), roles.WithRecordFindQuery("description", "seed"), roles.WithRecordFindQuery("x_seed", "request-only"))
			if err != nil || row == nil || calls != 1 {
				t.Fatal(row, err, calls)
			}
			roleFindField(t, row, "id", tc.id)
			roleFindField(t, row, "options", tc.options)
			roleFindField(t, row, "domain_id", `"seed-domain"`)
			for _, key := range []string{"x_seed", "x_extension"} {
				if row.Resource.Body[key] != nil {
					t.Fatal(row.Resource)
				}
			}
			clone := row.Clone()
			clone.Resource.Body["id"][0] = '['
			clone.Envelope[0] = '['
			clone.Header.Set("X-Request-Id", "changed")
			if string(row.Resource.Body["id"]) != tc.id || string(row.Envelope) != tc.body || row.Header.Get("X-Request-Id") != "role-proof" {
				t.Fatal(row)
			}
		})
	}
}

func TestPythonRoleFindRecordCleanFallbackKeepsDeclaredQueriesAndLocalFilters(t *testing.T) {
	for _, status := range []int{400, 403, 404} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls := 0
			api, _ := roleRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				if calls == 1 {
					return roleRecordWire(status, `{}`)
				}
				if r.URL.Path != "/keystone/v3/roles" || r.URL.Query().Get("domain_id") != "d" || r.URL.Query().Get("name") != "reader" || r.URL.Query().Has("description") || r.URL.Query().Has("options") || r.URL.Query().Has("links") || r.URL.Query().Has("x_unknown") {
					t.Fatal(r.URL)
				}
				return roleRecordWire(200, `{"roles":[{"id":"skip","name":"reader","description":0.2,"options":{"flag":1.0}},{"id":"wrong-link","name":"reader","description":0.10000000000000001,"options":{"flag":1.0},"links":{"self":"no"}},{"id":"kept","name":"reader","description":0.10000000000000001,"options":{"flag":1.0},"links":{"self":"yes"}}]}`)
			})
			row, err := api.FindRecord(context.Background(), "reader", roles.WithRecordFindQuery("domain_id", "d"), roles.WithRecordFindQuery("description", json.RawMessage(`0.1`)), roles.WithRecordFindQuery("options", map[string]any{"flag": true}), roles.WithRecordFindQuery("links", map[string]any{"self": "yes"}), roles.WithRecordFindQuery("x_unknown", "request-only"))
			if err != nil || row == nil || calls != 2 {
				t.Fatal(row, err, calls)
			}
			roleFindField(t, row, "id", `"kept"`)
		})
	}
}

func TestPythonRoleFindRecordCompleteSearchAmbiguityLateFailureAndMissing(t *testing.T) {
	for _, mode := range []string{"single after full scan", "duplicate id", "late native error", "missing default", "missing strict", "dictionary next ignored"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			api, _ := roleRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				if calls == 1 {
					return roleRecordWire(404, `{}`)
				}
				if mode == "missing default" || mode == "missing strict" {
					return roleRecordWire(200, `{"roles":[]}`)
				}
				if calls == 2 {
					if mode == "dictionary next ignored" {
						return roleRecordWire(200, `{"roles":[{"id":"one","name":"reader"}],"links":{"next":"?name=reader&marker=one"}}`)
					}
					return roleRecordWire(200, `{"roles":[{"id":"one","name":"reader"}],"links":[{"rel":"next","href":"?name=reader&marker=one"}]}`)
				}
				if r.URL.Query().Get("marker") != "one" {
					t.Fatal(r.URL)
				}
				if mode == "duplicate id" {
					return roleRecordWire(200, `{"roles":[{"id":"one","name":"reader"}]}`)
				}
				if mode == "late native error" {
					return roleRecordWire(503, `{"error":"late proof"}`)
				}
				return roleRecordWire(200, `{"roles":[]}`)
			})
			options := []roles.RecordFindOption{}
			if mode == "missing strict" {
				options = append(options, roles.WithRecordFindIgnoreMissing(false))
			}
			row, err := api.FindRecord(context.Background(), "reader", options...)
			switch mode {
			case "dictionary next ignored":
				if err != nil || row == nil || calls != 2 {
					t.Fatal(row, err, calls)
				}
				roleFindField(t, row, "id", `"one"`)
			case "single after full scan":
				if err != nil || row == nil || calls != 3 {
					t.Fatal(row, err, calls)
				}
				roleFindField(t, row, "id", `"one"`)
			case "duplicate id":
				if row != nil || !errors.Is(err, resource.ErrAmbiguous) || calls != 3 {
					t.Fatal(row, err, calls)
				}
			case "late native error":
				var native gophercloud.ErrUnexpectedResponseCode
				if row != nil || !errors.As(err, &native) || native.Actual != 503 || calls != 3 {
					t.Fatal(row, err, native, calls)
				}
			case "missing default":
				if row != nil || err != nil || calls != 2 {
					t.Fatal(row, err, calls)
				}
			case "missing strict":
				if row != nil || !errors.Is(err, resource.ErrNotFound) || calls != 2 {
					t.Fatal(row, err, calls)
				}
			}
		})
	}
}

func TestPythonRoleFindRecordOptionLifecycleVersionsAndPreflight(t *testing.T) {
	t.Run("GET version separate from fallback", func(t *testing.T) {
		calls := 0
		api, _ := roleRecordAPI(t, func(r *http.Request) *http.Response {
			calls++
			if r.Header.Get("X-Owned") != "yes" {
				t.Fatal(r.Header)
			}
			if calls == 1 {
				if r.Header.Get("OpenStack-API-Version") != "identity 3.9" {
					t.Fatal(r.Header)
				}
				return roleRecordWire(404, `{}`)
			}
			if r.Header.Get("OpenStack-API-Version") != "" || r.URL.Path != "/keystone/v3/fallback-roles" {
				t.Fatal(r.URL, r.Header)
			}
			return roleRecordWire(200, `{"roles":[{"id":"r","name":"reader"}]}`)
		})
		row, err := api.FindRecord(context.Background(), "reader", roles.WithRecordFindMicroversion("3.9"), roles.WithRecordFindHeaders(map[string]string{"X-Owned": "yes"}), roles.WithRecordFindListBasePath("fallback-roles"))
		if err != nil || row == nil || calls != 2 {
			t.Fatal(row, err, calls)
		}
	})
	for _, mode := range []string{"nil option", "callback error", "source mutation", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			later := 0
			cause := errors.New("role option failure")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			api, fixture := roleRecordAPI(t, func(r *http.Request) *http.Response { calls++; return roleRecordWire(200, `{"role":{}}`) })
			var option roles.RecordFindOption
			if mode != "nil option" {
				option = func(*roles.RecordFindOpts) error {
					switch mode {
					case "callback error":
						return cause
					case "source mutation":
						*api = *roles.New(fixture.Client("identity", "/replacement/"))
					case "cancel":
						cancel(cause)
					}
					return nil
				}
			}
			row, err := api.FindRecord(ctx, "reader", option, func(*roles.RecordFindOpts) error { later++; return nil })
			if row != nil || err == nil || calls != 0 || later != 0 {
				t.Fatal(row, err, calls, later)
			}
			if mode == "callback error" || mode == "cancel" {
				if !errors.Is(err, cause) {
					t.Fatal(err)
				}
			}
		})
	}
}
