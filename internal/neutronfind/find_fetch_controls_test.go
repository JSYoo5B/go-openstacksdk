package neutronfind_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func fetchControlsEngine(t *testing.T, cloud *testcloud.Cloud, guard func(context.Context) error) *neutronfind.Engine {
	t.Helper()
	profile := policyProfile(true)
	profile.SourceStatusPolicy = true
	engine, err := neutronfind.New(context.Background(), cloud.Client("network", "/v2.0/"), profile, guard)
	if err != nil {
		t.Fatal(err)
	}
	return engine
}
func fetchControlsQuery(values map[string]string) neutronfind.Options {
	query := make(map[string]json.RawMessage, len(values))
	for key, value := range values {
		query[key] = json.RawMessage(value)
	}
	return neutronfind.Options{Query: query}
}

func TestFindWithFetchControlsRequiresIDUsesSourceTruthiness(t *testing.T) {
	for _, tc := range []struct{ name, raw, path string }{
		{"default", "", "/v2.0/records/target"}, {"true", "true", "/v2.0/records/target"},
		{"null class default", "null", "/v2.0/records/target"}, {"false", "false", "/v2.0/records"},
		{"zero", "0", "/v2.0/records"}, {"float underflow", "1e-4000", "/v2.0/records"},
		{"empty string", `""`, "/v2.0/records"}, {"empty list", "[]", "/v2.0/records"},
		{"truthy string", `"false"`, "/v2.0/records/target"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.EscapedPath() != tc.path || req.URL.RawQuery != "" {
					t.Fatalf("unexpected fetch %s", req.URL)
				}
				return policyWire(200, `{"record":{"id":"actual","name":"target"}}`), nil
			})
			policy := neutronfind.Options{}
			if tc.raw != "" {
				policy = fetchControlsQuery(map[string]string{"requires_id": tc.raw})
			}
			value, err := fetchControlsEngine(t, cloud, nil).FindWithFetchControls(context.Background(), "target", policy)
			if err != nil || value == nil || calls != 1 || string(value.Resource.Body["id"]) != `"actual"` {
				t.Fatal(value, err, calls)
			}
		})
	}
}

func TestFindWithFetchControlsBasePathOwnsBothPhasesAndFallbackOverride(t *testing.T) {
	for _, override := range []bool{false, true} {
		t.Run(map[bool]string{false: "shared literal", true: "list override"}[override], func(t *testing.T) {
			cloud := testcloud.New(t)
			var paths []string
			cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
				paths = append(paths, req.URL.EscapedPath())
				if len(paths) == 1 {
					return policyWire(404, `{"error":"missing"}`), nil
				}
				if req.URL.Query().Get("name") != "target" || req.URL.Query().Has("base_path") || req.URL.Query().Has("requires_id") {
					t.Fatal(req.URL)
				}
				return policyWire(200, `{"records":[{"id":"found","name":"target"}]}`), nil
			})
			policy := fetchControlsQuery(map[string]string{"base_path": `"/alternative/records"`, "requires_id": "false"})
			wantList := "/v2.0/alternative/records"
			if override {
				path := "/fallback/records"
				policy.ListBasePath = &path
				wantList = "/v2.0/fallback/records"
			}
			value, err := fetchControlsEngine(t, cloud, nil).FindWithFetchControls(context.Background(), "target", policy)
			if err != nil || value == nil || !value.FromList() || !reflect.DeepEqual(paths, []string{"/v2.0/alternative/records", wantList}) {
				t.Fatal(value, err, paths)
			}
		})
	}
}

func TestFindWithFetchControlsResponseKeyAndAcceptedFailureEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, control, body, want string
		fail                      bool
	}{
		{"custom", `"custom"`, `{"custom":{"id":"custom-id"},"record":{"id":"default-id"}}`, `"custom-id"`, false},
		{"absent custom falls back", `"custom"`, `{"record":{"id":"default-id"}}`, `"default-id"`, false},
		{"null chooses default", `null`, `{"record":{"id":"default-id"}}`, `"default-id"`, false},
		{"present malformed custom", `"custom"`, `{"custom":[],"record":{"id":"default-id"}}`, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				response := policyWire(202, tc.body)
				response.Header.Set("X-Request-ID", "fetch-key-proof")
				return response, nil
			})
			value, err := fetchControlsEngine(t, cloud, nil).FindWithFetchControls(context.Background(), "target", fetchControlsQuery(map[string]string{"resource_response_key": tc.control}))
			if tc.fail {
				var response *resource.ResponseError
				if value != nil || calls != 1 || !errors.As(err, &response) || response.StatusCode != 202 || response.Header.Get("X-Request-ID") != "fetch-key-proof" || string(response.Body) != tc.body {
					t.Fatal("malformed selected envelope lost actual accepted response", value, err, calls)
				}
				return
			}
			if calls != 1 || value == nil || value.StatusCode != 202 || value.Header.Get("X-Request-ID") != "fetch-key-proof" || string(value.Envelope) != tc.body {
				t.Fatal(value, err, calls)
			}
			if err != nil || string(value.Resource.Body["id"]) != tc.want {
				t.Fatal(value, err)
			}
		})
	}
}

func TestFindWithFetchControlsRawHeadersAreGETQueryThenFallbackHeaders(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			if !reflect.DeepEqual(req.URL.Query()["headers"], []string{"Accept", "X-Fetch-Proof"}) || req.Header.Get("X-Fetch-Proof") != "owned-loses" {
				t.Fatal(req.URL, req.Header)
			}
			return policyWire(404, `{"error":"missing"}`), nil
		}
		if req.URL.Query().Has("headers") || req.Header.Get("Accept") != "application/vnd.source+json" || req.Header.Get("X-Fetch-Proof") != "raw-wins" || req.Header.Get("X-Owned") != "owned" {
			t.Fatal(req.URL, req.Header)
		}
		return policyWire(200, `{"records":[{"id":"found","name":"target"}]}`), nil
	})
	policy := fetchControlsQuery(map[string]string{"headers": `{"Accept":"application/vnd.source+json","X-Fetch-Proof":"raw-wins"}`})
	policy.Headers = map[string]string{"X-Owned": "owned", "x-fetch-proof": "owned-loses"}
	value, err := fetchControlsEngine(t, cloud, nil).FindWithFetchControls(context.Background(), "target", policy)
	if err != nil || value == nil || calls != 2 {
		t.Fatal(value, err, calls)
	}
}

func TestFindWithFetchControlsInvalidRawHeadersAreDeferredUntilFallback(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		t.Run(map[bool]string{false: "successful GET", true: "fallback"}[fallback], func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.Query().Get("headers") != "X-Proof" {
					t.Fatal(req.URL)
				}
				if fallback {
					return policyWire(404, `{"error":"missing"}`), nil
				}
				return policyWire(200, `{"record":{"id":"found"}}`), nil
			})
			value, err := fetchControlsEngine(t, cloud, nil).FindWithFetchControls(context.Background(), "target", fetchControlsQuery(map[string]string{"headers": `{"X-Proof":2}`}))
			if calls != 1 {
				t.Fatal(calls)
			}
			if fallback {
				if err == nil || value != nil {
					t.Fatal(value, err)
				}
			} else if err != nil || value == nil {
				t.Fatal(value, err)
			}
		})
	}
}

func TestFindWithFetchControlsSkipCacheIsLocalAndFetchesRemainStateless(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.RawQuery != "" {
			t.Fatal(req.URL)
		}
		if calls == 1 {
			return policyWire(200, `{"record":{"id":"one"}}`), nil
		}
		return policyWire(200, `{"record":{"id":"two"}}`), nil
	})
	engine := fetchControlsEngine(t, cloud, nil)
	for i, raw := range []string{"false", `{"ignored":true}`} {
		value, err := engine.FindWithFetchControls(context.Background(), "target", fetchControlsQuery(map[string]string{"skip_cache": raw}))
		want := []string{`"one"`, `"two"`}[i]
		if err != nil || value == nil || string(value.Resource.Body["id"]) != want {
			t.Fatal(value, err)
		}
	}
	if calls != 2 {
		t.Fatal(calls)
	}
}

func TestFindWithFetchControlsCustomErrorPreservesNativeFailureAndCleanFallback(t *testing.T) {
	for _, status := range []int{500, 404} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			body := `{"error":"upstream proof"}`
			cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.Query().Has("error_message") {
					t.Fatal(req.URL)
				}
				if calls > 1 {
					return policyWire(200, `{"records":[{"id":"found","name":"target"}]}`), nil
				}
				response := policyWire(status, body)
				response.Header.Set("X-Request-ID", "failure-proof")
				return response, nil
			})
			value, err := fetchControlsEngine(t, cloud, nil).FindWithFetchControls(context.Background(), "target", fetchControlsQuery(map[string]string{"error_message": `"custom lookup failed"`}))
			if status == 404 {
				if err != nil || value == nil || calls != 2 {
					t.Fatal(value, err, calls)
				}
				return
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if value != nil || calls != 1 || !errors.As(err, &native) || native.Actual != 500 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-ID") != "failure-proof" || !strings.Contains(err.Error(), "custom lookup failed") {
				t.Fatal(value, err, calls, native)
			}
		})
	}
}

func TestFindWithFetchControlsOwnsInputAndRejectsUnsafeRoutes(t *testing.T) {
	t.Run("inputs remain unchanged", func(t *testing.T) {
		cloud := testcloud.New(t)
		cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
			return policyWire(200, `{"record":{"id":"found"}}`), nil
		})
		policy := fetchControlsQuery(map[string]string{"requires_id": "false", "base_path": `"/custom"`, "headers": `{"X-Proof":"owned"}`})
		policy.Order = []string{"requires_id", "base_path", "headers"}
		before, _ := json.Marshal(policy)
		_, err := fetchControlsEngine(t, cloud, nil).FindWithFetchControls(context.Background(), "target", policy)
		after, _ := json.Marshal(policy)
		if err != nil || string(before) != string(after) {
			t.Fatal(err, string(before), string(after))
		}
	})
	for _, path := range []string{"https://other.invalid/records", "//other.invalid/records", "../records", "/records?admin=true", "/records/%(unknown)s"} {
		t.Run(path, func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) { calls++; return policyWire(200, `{"record":{}}`), nil })
			raw, _ := json.Marshal(path)
			value, err := fetchControlsEngine(t, cloud, nil).FindWithFetchControls(context.Background(), "target", neutronfind.Options{Query: map[string]json.RawMessage{"base_path": raw}})
			if err == nil || value != nil || calls != 0 {
				t.Fatal(value, err, calls)
			}
		})
	}
	t.Run("cancellation and captured source guard", func(t *testing.T) {
		cloud := testcloud.New(t)
		calls := 0
		cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) { calls++; return policyWire(200, `{"record":{}}`), nil })
		engine := fetchControlsEngine(t, cloud, nil)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if value, err := engine.FindWithFetchControls(ctx, "target", neutronfind.Options{}); value != nil || !errors.Is(err, context.Canceled) {
			t.Fatal(value, err)
		}
		// A client change after capture must fail before any request can escape.
		guardedClient := cloud.Client("network", "/v2.0/")
		guarded, err := neutronfind.New(context.Background(), guardedClient, policyProfile(true), nil)
		if err != nil {
			t.Fatal(err)
		}
		guardedClient.Endpoint = cloud.Server.URL + "/changed/"
		if value, err := guarded.FindWithFetchControls(context.Background(), "target", neutronfind.Options{}); value != nil || err == nil {
			t.Fatal(value, err)
		}
		if calls != 0 {
			t.Fatal(calls)
		}
	})
	t.Run("ordinary Find remains rejecting", func(t *testing.T) {
		cloud := testcloud.New(t)
		calls := 0
		cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) { calls++; return policyWire(200, `{"record":{}}`), nil })
		value, err := fetchControlsEngine(t, cloud, nil).Find(context.Background(), "target", fetchControlsQuery(map[string]string{"requires_id": "false"}))
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatal(value, err, calls)
		}
	})
}
