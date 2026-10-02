package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/keymanager/v1/secretstores"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// openstacksdk ef55d7d _proxy.py:361–392, SecretStore:17–58 and
// HREFToUUID:18–30; Barbican's published store-backend GETs accept 200.
const secretStoresPath = "/reverse/barbican/v1/secret-stores"

func secretStoresClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("key-manager", "/catalog/v1")
	client.ResourceBase = cloud.Server.URL + "/reverse/barbican/v1/"
	return client
}

func secretStoreRow(t *testing.T, ref string, extra map[string]any) string {
	t.Helper()
	fields := map[string]any{"name": "Different", "status": "ACTIVE", "secret_store_ref": ref}
	for key, value := range extra {
		fields[key] = value
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func secretStorePage(rows, next string) string {
	if next == "" {
		return `{"secret_stores":[` + rows + `]}`
	}
	encoded, _ := json.Marshal(next)
	return `{"secret_stores":[` + rows + `],"next":` + string(encoded) + `}`
}

func secretStoreWantIDs(t *testing.T, values []*secretstores.SecretStore, err error, ids ...string) {
	t.Helper()
	if err != nil || len(values) != len(ids) {
		t.Fatal(values, err, ids)
	}
	for i, id := range ids {
		if values[i] == nil || values[i].ID != id {
			t.Fatal("unexpected passive identifier", values[i], id)
		}
	}
}

func TestKeyManagerSecretStoresSingletonPathsModelsAndPassiveReferences(t *testing.T) {
	cloud, foreign := testcloud.New(t), testcloud.New(t)
	var followed, calls atomic.Int32
	foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { followed.Add(1); w.WriteHeader(500) })
	for _, selector := range []string{"global-default", "preferred"} {
		cloud.Mux.HandleFunc("GET "+secretStoresPath+"/"+selector, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.URL.RawQuery != "" || r.Header.Get("X-Configured") != "preserved" || r.Header.Get("X-Auth-Token") != "test-token" {
				t.Error(r.URL, r.Header)
			}
			w.Header().Set("X-Request-ID", selector)
			ref := foreign.Server.URL + "/unrelated/backend%2Fpart?query=ignored#fragment"
			testcloud.JSON(w, 200, secretStoreRow(t, ref, map[string]any{
				"global_default": false, "crypto_plugin": nil, "secret_store_plugin": "kmip",
				"created": "2016-08-22T23:46:45.114283", "updated": nil,
				"vendor": json.RawMessage(`{"quota":9007199254740993,"fraction":1.000000000000000001}`),
			}))
		})
	}
	client := secretStoresClient(cloud)
	client.MoreHeaders = map[string]string{"X-Configured": "preserved"}
	a := secretstores.New(client)
	if a.RawClient() != client || a.RawClient().ProviderClient != cloud.Provider {
		t.Fatal("API replaced the configured source")
	}
	global, err := a.GetGlobalDefault(context.Background())
	if err != nil || global == nil || global.ID != "backend%2Fpart" || global.StatusCode != 200 || global.Header.Get("X-Request-ID") != "global-default" || global.GlobalDefault == nil || *global.GlobalDefault || global.CryptoPlugin != nil || global.SecretStorePlugin == nil || *global.SecretStorePlugin != "kmip" || global.CreatedAt == nil || *global.CreatedAt != "2016-08-22T23:46:45.114283" || global.UpdatedAt != nil || string(global.Body["updated"]) != "null" || !strings.Contains(string(global.Body["vendor"]), "9007199254740993") {
		t.Fatal(global, err)
	}
	global.Header.Set("X-Request-ID", "caller-change")
	global.Body["vendor"][0] = '['
	*global.GlobalDefault, *global.CreatedAt, *global.SecretStorePlugin = true, "caller-change", "caller-change"
	preferred, err := a.GetPreferred(context.Background())
	if err != nil || preferred == nil || preferred.Header.Get("X-Request-ID") != "preferred" || *preferred.GlobalDefault || *preferred.CreatedAt == "caller-change" || *preferred.SecretStorePlugin != "kmip" || !json.Valid(preferred.Body["vendor"]) || followed.Load() != 0 || calls.Load() != 2 {
		t.Fatal("response ownership or passive references changed", preferred, err, followed.Load(), calls.Load())
	}
	for _, tc := range []struct {
		name, suffix, id string
		extra            map[string]any
	}{
		{"literal-unicode-and-space", "/literal 한 글", "literal 한 글", nil},
		{"encoded-slash-query-fragment", "/encoded%2fpart?x=1#ignored", "encoded%2fpart", nil},
		{"trailing-slash", "/store/", "", nil},
		{"literal-id-priority", "/derived", "raw-id", map[string]any{"id": "raw-id"}},
		{"literal-empty-priority", "/derived", "", map[string]any{"id": ""}},
		{"literal-null-priority", "/derived", "", map[string]any{"id": nil}},
		{"no-casefold-identity", "", "", map[string]any{"secret_store_ref": nil, "SECRET_STORE_REF": "https://foreign.invalid/decoy", "ID": "decoy"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := testcloud.New(t)
			server.Mux.HandleFunc("GET "+secretStoresPath+"/preferred", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, secretStoreRow(t, foreign.Server.URL+tc.suffix, tc.extra))
			})
			value, err := secretstores.New(secretStoresClient(server)).GetPreferred(context.Background())
			if err != nil || value == nil || value.ID != tc.id || value.GlobalDefault != nil || followed.Load() != 0 {
				t.Fatal(value, err, followed.Load())
			}
			if _, exists := value.Body["global_default"]; exists {
				t.Fatal("omitted default flag was fabricated")
			}
		})
	}
	t.Run("canonical-fields-ignore-case-alias-extensions", func(t *testing.T) {
		server := testcloud.New(t)
		var calls atomic.Int32
		body := `{"name":"canonical","NAME":{"wrong":true},"status":"ACTIVE","STATUS":[],"global_default":false,"GLOBAL_DEFAULT":true,"crypto_plugin":null,"CRYPTO_PLUGIN":{},"secret_store_plugin":"kmip","Secret_Store_Plugin":17,"created":"original timestamp","CREATED":false,"updated":null,"UPDATED":[],"secret_store_ref":"https://foreign.invalid/canonical","SECRET_STORE_REF":{},"id":"literal-id","ID":[9007199254740993]}`
		for _, path := range []string{secretStoresPath + "/preferred", secretStoresPath} {
			server.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				response := body
				if r.URL.Path == secretStoresPath {
					response = secretStorePage(body, "")
				}
				testcloud.JSON(w, 200, response)
			})
		}
		api := secretstores.New(secretStoresClient(server))
		check := func(value *secretstores.SecretStore, err error) {
			t.Helper()
			if err != nil || value == nil || value.Name != "canonical" || value.Status != "ACTIVE" || value.GlobalDefault == nil || *value.GlobalDefault || value.CryptoPlugin != nil || value.SecretStorePlugin == nil || *value.SecretStorePlugin != "kmip" || value.CreatedAt == nil || *value.CreatedAt != "original timestamp" || value.UpdatedAt != nil || value.SecretStoreRef != "https://foreign.invalid/canonical" || value.ID != "literal-id" || value.StatusCode != 200 {
				t.Fatal("unknown case aliases affected declared typed fields", value, err)
			}
			for key, raw := range map[string]string{"NAME": `{"wrong":true}`, "STATUS": `[]`, "GLOBAL_DEFAULT": `true`, "CRYPTO_PLUGIN": `{}`, "Secret_Store_Plugin": `17`, "CREATED": `false`, "UPDATED": `[]`, "SECRET_STORE_REF": `{}`, "ID": `[9007199254740993]`} {
				if string(value.Body[key]) != raw {
					t.Fatal("unknown original JSON was lost", key, string(value.Body[key]), raw)
				}
			}
		}
		check(api.GetPreferred(context.Background()))
		values, err := api.All(context.Background())
		if err != nil || len(values) != 1 {
			t.Fatal(values, err)
		}
		check(values[0], nil)
		if calls.Load() != 2 || followed.Load() != 0 {
			t.Fatal("case aliases caused ref follow or extra HTTP", calls.Load(), followed.Load())
		}
	})
}

func TestKeyManagerSecretStoresStrictSingletonErrorsAndAcceptedEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		code       int
		accepted   bool
	}{
		{"missing", `{"message":"not configured"}`, 404, false},
		{"forbidden", `{"message":"denied"}`, 403, false},
		{"unexpected-created", `{}`, 201, false},
		{"unexpected-empty", ``, 204, false},
		{"malformed", `{"name":`, 200, true},
		{"array", `[]`, 200, true},
		{"null", `null`, 200, true},
		{"typed-flag", `{"global_default":"false"}`, 200, true},
		{"typed-plugin", `{"crypto_plugin":{}}`, 200, true},
		{"typed-literal-id", `{"id":9}`, 200, true},
		{"invalid-ref", `{"secret_store_ref":"/relative/store"}`, 200, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+secretStoresPath+"/preferred", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Request-ID", "original-response")
				testcloud.JSON(w, tc.code, tc.body)
			})
			value, err := secretstores.New(secretStoresClient(cloud)).GetPreferred(context.Background())
			if value != nil || err == nil || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
			var op *resource.OperationError
			if !errors.As(err, &op) || op.Operation != "GetPreferred" {
				t.Fatal("operation evidence missing", err)
			}
			if tc.accepted {
				var evidence *resource.ResponseError
				if !errors.As(err, &evidence) || evidence.StatusCode != tc.code || string(evidence.Body) != tc.body || evidence.Header.Get("X-Request-ID") != "original-response" {
					t.Fatal("accepted response was discarded", err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != tc.code || string(native.Body) != tc.body || native.ResponseHeader.Get("X-Request-ID") != "original-response" {
					t.Fatal("native HTTP cause was discarded", err)
				}
				if tc.code == 404 && !errors.Is(err, resource.ErrNotFound) {
					t.Fatal("singleton 404 lost strict missing classification", err)
				}
			}
		})
	}
	t.Run("transport-status-is-not-not-found", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := secretStoresClient(cloud)
		native := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{200}, Body: []byte(`{"message":"transport cause"}`)}
		var calls atomic.Int32
		client.HTTPClient.Transport = secretStoreRoundTripper(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			return nil, native
		})
		value, err := secretstores.New(client).GetPreferred(context.Background())
		var transport *url.Error
		var cause gophercloud.ErrUnexpectedResponseCode
		if value != nil || !errors.As(err, &transport) || !errors.As(err, &cause) || errors.Is(err, resource.ErrNotFound) || calls.Load() != 1 {
			t.Fatal("transport status was reclassified as resource absence", value, err, calls.Load())
		}
	})
}

func TestKeyManagerSecretStoresListQuerySnapshotsAndConcurrentReuse(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	var captured *request.Config[secretstores.ListOpts]
	want := url.Values{"name": {"server-pattern"}, "status": {"ACTIVE"}, "global_default": {"false"}, "crypto_plugin": {"plugin"}, "secret_store_plugin": {"store"}, "created": {"raw-created"}, "updated": {"raw-updated"}, "vendor": {"first", "second"}, "empty": {""}}
	cloud.Mux.HandleFunc("GET "+secretStoresPath, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if !reflect.DeepEqual(r.URL.Query(), want) {
			t.Error("query snapshot changed", r.URL.Query(), want)
		}
		if captured != nil {
			captured.Query.Set("vendor", "changed")
			*captured.Options.GlobalDefault, *captured.Options.Paginated = true, true
		}
		testcloud.JSON(w, 200, secretStorePage(secretStoreRow(t, "https://foreign.invalid/store-one", nil), "https://foreign.invalid/not-read"))
	})
	falseValue := false
	options := secretstores.WithListOptions(secretstores.ListOpts{Name: "server-pattern", Status: "ACTIVE", GlobalDefault: &falseValue, CryptoPlugin: "plugin", SecretStorePlugin: "store", Created: "raw-created", Updated: "raw-updated", Paginated: &falseValue})
	falseValue = true
	query := func(config *request.Config[secretstores.ListOpts]) error {
		config.Query = url.Values{"vendor": {"first", "second"}, "empty": {""}, "nil": nil}
		captured = config
		return nil
	}
	a := secretstores.New(secretStoresClient(cloud))
	opts := []secretstores.ListOption{options, query}
	seq := a.List(context.Background(), opts...)
	opts[0] = secretstores.WithListGlobalDefault(true)
	if calls.Load() != 0 {
		t.Fatal("iterator construction was eager")
	}
	for range 2 {
		var values []*secretstores.SecretStore
		for value, err := range seq {
			if err != nil {
				t.Fatal(err)
			}
			values = append(values, value)
		}
		secretStoreWantIDs(t, values, nil, "store-one")
	}
	// Concurrent reuse contains only immutable library-owned option captures;
	// the custom option above deliberately retains its mutable Config for the
	// separate sequential snapshot proof.
	captured = nil
	concurrent := []secretstores.ListOption{options, secretstores.WithListQuery("empty", ""), func(config *request.Config[secretstores.ListOpts]) error {
		config.Query["vendor"] = []string{"first", "second"}
		return nil
	}}
	var group sync.WaitGroup
	for range 6 {
		group.Add(1)
		go func() {
			defer group.Done()
			values, err := a.All(context.Background(), concurrent...)
			if err != nil || len(values) != 1 || values[0].ID != "store-one" {
				t.Error(values, err)
			}
		}()
	}
	group.Wait()
	if calls.Load() != 8 {
		t.Fatal("reuse performed extra HTTP", calls.Load())
	}
}

func TestKeyManagerSecretStoresMarkerFallbackUsesFullRawReference(t *testing.T) {
	for _, tc := range []struct {
		name, ref, marker, id string
		extra                 map[string]any
	}{
		{"full-ref", "https://foreign.invalid/store%2Fpart?x=1#fragment", "https://foreign.invalid/store%2Fpart?x=1#fragment", "store%2Fpart", nil},
		{"literal-ref", "https://foreign.invalid/한 글", "https://foreign.invalid/한 글", "한 글", nil},
		{"trailing-ref", "https://foreign.invalid/store/", "https://foreign.invalid/store/", "", nil},
		{"literal-id-priority", "https://foreign.invalid/derived", "wire-id", "wire-id", map[string]any{"id": "wire-id"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+secretStoresPath, func(w http.ResponseWriter, r *http.Request) {
				page := calls.Add(1)
				if r.URL.Query().Get("limit") != "5" || r.URL.Query().Get("vendor") != "retained" || page == 1 && r.URL.Query().Has("marker") || page == 2 && r.URL.Query().Get("marker") != tc.marker {
					t.Error("fallback marker is not the source's raw identity", r.URL)
				}
				rows := ""
				if page == 1 {
					rows = secretStoreRow(t, tc.ref, tc.extra)
				}
				testcloud.JSON(w, 200, secretStorePage(rows, ""))
			})
			values, err := secretstores.New(secretStoresClient(cloud)).All(context.Background(), secretstores.WithListOptions(secretstores.ListOpts{Limit: 5}), secretstores.WithListQuery("vendor", "retained"))
			secretStoreWantIDs(t, values, err, tc.id)
			if calls.Load() != 2 {
				t.Fatal("short-page fallback or empty-page stop changed", calls.Load())
			}
		})
	}
	for _, id := range []any{nil, ""} {
		t.Run(fmt.Sprintf("empty-literal-id-%v", id), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			body := secretStorePage(secretStoreRow(t, "https://foreign.invalid/derived", map[string]any{"id": id}), "")
			cloud.Mux.HandleFunc("GET "+secretStoresPath, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, body) })
			values, err := secretstores.New(secretStoresClient(cloud)).All(context.Background(), secretstores.WithListOptions(secretstores.ListOpts{Limit: 1}))
			var evidence *resource.ResponseError
			if values != nil || !errors.As(err, &evidence) || !errors.Is(err, resource.ErrInvalidOption) || string(evidence.Body) != body || calls.Load() != 1 {
				t.Fatal("empty raw marker bypassed guard", values, err, calls.Load())
			}
		})
	}
}

func TestKeyManagerSecretStoresListControlsAndLazyRowEvidence(t *testing.T) {
	for _, mode := range []string{"default", "cap", "explicit-limit", "single", "break", "empty", "invalid-consumed", "malformed-page"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			row := secretStoreRow(t, "https://foreign.invalid/first", nil)
			rows, next := row+`,{"global_default":"wrong"}`, "https://foreign.invalid/not-read"
			var options []secretstores.ListOption
			wantLimit := ""
			switch mode {
			case "default":
				rows, next = row, ""
			case "cap":
				options, wantLimit = []secretstores.ListOption{secretstores.WithListMaxItems(1)}, "1"
			case "explicit-limit":
				options, wantLimit = []secretstores.ListOption{secretstores.WithListOptions(secretstores.ListOpts{Limit: 9}), secretstores.WithListMaxItems(1)}, "9"
			case "single":
				rows, options = row, []secretstores.ListOption{secretstores.WithListPaginated(false)}
			case "break":
				rows = row
			case "empty":
				rows = ""
			}
			body := secretStorePage(rows, next)
			if mode == "malformed-page" {
				body = `{"secret_stores":[` + row + `,`
				options, wantLimit = []secretstores.ListOption{secretstores.WithListMaxItems(1)}, "1"
			}
			cloud.Mux.HandleFunc("GET "+secretStoresPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("limit") != wantLimit || r.URL.Query().Has("max_items") || r.URL.Query().Has("paginated") {
					t.Error(r.URL)
				}
				w.Header().Set("X-Request-ID", "whole-page")
				testcloud.JSON(w, 200, body)
			})
			a := secretstores.New(secretStoresClient(cloud))
			if mode == "break" {
				var seen int
				for value, err := range a.List(context.Background()) {
					if err != nil || value == nil || value.ID != "first" {
						t.Fatal(value, err)
					}
					seen++
					break
				}
				if seen != 1 || calls.Load() != 1 {
					t.Fatal(seen, calls.Load())
				}
				return
			}
			values, err := a.All(context.Background(), options...)
			if mode == "invalid-consumed" || mode == "malformed-page" {
				var evidence *resource.ResponseError
				if values != nil || !errors.As(err, &evidence) || evidence.StatusCode != 200 || evidence.Header.Get("X-Request-ID") != "whole-page" || string(evidence.Body) != body {
					t.Fatal("accepted whole-page evidence lost", values, err)
				}
			} else if mode == "empty" {
				secretStoreWantIDs(t, values, err)
			} else {
				secretStoreWantIDs(t, values, err, "first")
			}
			if calls.Load() != 1 {
				t.Fatal("local stop performed extra HTTP", calls.Load())
			}
		})
	}
}

func TestKeyManagerSecretStoresContinuationGuardsAndTerminalFailures(t *testing.T) {
	for _, mode := range []string{"next", "plural-links", "http-link", "foreign", "wrong-path", "filter-change", "cycle", "late-http", "cancel", "changed-type"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := secretStoresClient(cloud)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+secretStoresPath, func(w http.ResponseWriter, r *http.Request) {
				page := calls.Add(1)
				if r.URL.Query().Get("vendor") != "retained" {
					t.Error("continuation dropped initial filter", r.URL)
				}
				rows := secretStoreRow(t, "https://foreign.invalid/first", nil)
				next := cloud.Server.URL + secretStoresPath + "?marker=second"
				if page > 1 {
					rows = secretStoreRow(t, "https://foreign.invalid/second", nil)
					next = ""
					if mode == "cycle" {
						next = cloud.Server.URL + secretStoresPath + "?marker=second"
					} else if mode == "late-http" {
						w.WriteHeader(404)
						return
					} else if mode == "cancel" {
						cancel()
					}
				} else {
					switch mode {
					case "foreign":
						next = "https://foreign.invalid/secret-stores?marker=second"
					case "wrong-path":
						next = cloud.Server.URL + "/other/secret-stores?marker=second"
					case "filter-change":
						next += "&vendor=replaced"
					}
				}
				w.Header().Set("X-Request-ID", "page-evidence")
				if mode == "http-link" && next != "" {
					w.Header().Set("Link", "<"+next+">; rel=\"next\"")
					next = ""
				}
				body := secretStorePage(rows, next)
				if mode == "plural-links" && next != "" {
					encoded, _ := json.Marshal(next)
					body = `{"secret_stores":[` + rows + `],"secret_stores_links":[{"rel":"next","href":` + string(encoded) + `}]}`
				}
				testcloud.JSON(w, 200, body)
			})
			if mode == "changed-type" {
				original := client.HTTPClient.Transport
				client.HTTPClient.Transport = secretStoreRoundTripper(func(r *http.Request) (*http.Response, error) {
					response, err := original.RoundTrip(r)
					client.Type = "network"
					return response, err
				})
			}
			values, err := secretstores.New(client).All(ctx, secretstores.WithListQuery("vendor", "retained"))
			wantCalls := int32(2)
			switch mode {
			case "next", "plural-links", "http-link":
				secretStoreWantIDs(t, values, err, "first", "second")
			case "foreign", "wrong-path", "filter-change":
				wantCalls = 1
				var evidence *resource.ResponseError
				if values != nil || !errors.As(err, &evidence) || evidence.Header.Get("X-Request-ID") != "page-evidence" || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(values, err)
				}
			case "cycle":
				if values != nil || !errors.Is(err, resource.ErrPaginationCycle) {
					t.Fatal(values, err)
				}
			case "late-http":
				if values != nil || !gophercloud.ResponseCodeIs(err, 404) {
					t.Fatal("list 404 was hidden", values, err)
				}
			case "cancel":
				if values != nil || !errors.Is(err, context.Canceled) {
					t.Fatal(values, err)
				}
			case "changed-type":
				wantCalls = 1
				if values != nil || !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal("source type was not rechecked", values, err)
				}
			}
			if calls.Load() != wantCalls {
				t.Fatal("terminal page performed extra HTTP", calls.Load(), wantCalls)
			}
		})
	}
}

type secretStoreRoundTripper func(*http.Request) (*http.Response, error)

func (f secretStoreRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type secretStoreReadFailure struct{ cause error }

func (r secretStoreReadFailure) Read([]byte) (int, error) { return 0, r.cause }

func TestKeyManagerSecretStoresLiveSourcePreflightAndReadFailures(t *testing.T) {
	cloud := testcloud.New(t)
	client := secretStoresClient(cloud)
	client.MoreHeaders = map[string]string{"X-Configured": "kept"}
	var calls, middleware atomic.Int32
	cloud.Mux.HandleFunc("GET "+secretStoresPath, func(w http.ResponseWriter, r *http.Request) {
		page := calls.Add(1)
		wantToken := "test-token"
		if page > 1 {
			wantToken = "new-token"
		}
		if r.Header.Get("X-Auth-Token") != wantToken || r.Header.Get("X-Configured") != "kept" || r.Header.Get("X-Middleware") != "original-provider" {
			t.Error("configured source was replaced", r.Header)
		}
		rows, next := secretStoreRow(t, "https://foreign.invalid/one", nil), cloud.Server.URL+secretStoresPath+"?marker=second"
		if page > 1 {
			rows, next = secretStoreRow(t, "https://foreign.invalid/two", nil), ""
		}
		testcloud.JSON(w, 200, secretStorePage(rows, next))
	})
	original := client.HTTPClient.Transport
	client.HTTPClient.Transport = secretStoreRoundTripper(func(r *http.Request) (*http.Response, error) {
		middleware.Add(1)
		r.Header.Set("X-Middleware", "original-provider")
		response, err := original.RoundTrip(r)
		client.ProviderClient.SetToken("new-token")
		return response, err
	})
	a := secretstores.New(client)
	values, err := a.All(context.Background())
	secretStoreWantIDs(t, values, err, "one", "two")
	if middleware.Load() != 2 || a.RawClient() != client || a.RawClient().ProviderClient != cloud.Provider {
		t.Fatal("provider middleware or source sharing changed", middleware.Load())
	}
	before := calls.Load()
	for _, options := range [][]secretstores.ListOption{
		{nil}, {secretstores.WithListMaxItems(-1)}, {secretstores.WithListOptions(secretstores.ListOpts{Limit: -1})},
		{secretstores.WithListQuery("max_items", "1")}, {secretstores.WithListQuery("Paginated", "false")},
		{secretstores.WithListQuery("limit", "0")}, {secretstores.WithListQuery("marker", " ")},
		{request.WithField[secretstores.ListOpts]("name", "body")}, {request.WithHeader[secretstores.ListOpts]("X-Extension", "header")},
	} {
		seq := a.List(context.Background(), options...)
		if calls.Load() != before {
			t.Fatal("preflight was eager")
		}
		var seen int
		for value, err := range seq {
			if value != nil || err == nil {
				t.Fatal(value, err)
			}
			seen++
		}
		if seen != 1 || calls.Load() != before {
			t.Fatal("invalid options performed HTTP", seen, calls.Load())
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if values, err := a.All(ctx); values != nil || !errors.Is(err, context.Canceled) || calls.Load() != before {
		t.Fatal(values, err, calls.Load())
	}
	if value, err := a.GetGlobalDefault(ctx); value != nil || !errors.Is(err, context.Canceled) || calls.Load() != before {
		t.Fatal(value, err, calls.Load())
	}
	if values, err := a.All(nil); values != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != before {
		t.Fatal("nil context performed HTTP", values, err, calls.Load())
	}
	for _, invalid := range []*secretstores.API{nil, secretstores.New(nil), secretstores.New(&gophercloud.ServiceClient{}), secretstores.New(cloud.Client("network", "/catalog/v1"))} {
		if value, err := invalid.GetPreferred(context.Background()); value != nil || err == nil {
			t.Fatal(value, err)
		}
		if values, err := invalid.All(context.Background()); values != nil || err == nil || calls.Load() != before {
			t.Fatal(values, err, calls.Load())
		}
	}
	if values, err := a.All(context.Background(), func(config *request.Config[secretstores.ListOpts]) error { client.Type = "network"; return nil }); values != nil || !errors.Is(err, resource.ErrUnsupported) || calls.Load() != before {
		t.Fatal("options changed source before unchecked HTTP", values, err)
	}
	client.Type = "key-manager"
	cause := errors.New("accepted response read failed")
	client.HTTPClient.Transport = secretStoreRoundTripper(func(r *http.Request) (*http.Response, error) {
		r.Header.Set("X-Middleware", "original-provider")
		response, err := original.RoundTrip(r)
		if err == nil {
			_ = response.Body.Close()
			response.Body = io.NopCloser(secretStoreReadFailure{cause})
		}
		return response, err
	})
	values, err = a.All(context.Background())
	var evidence *resource.ResponseError
	if values != nil || !errors.As(err, &evidence) || !errors.Is(err, cause) || evidence.StatusCode != 200 || calls.Load() != before+1 {
		t.Fatal("read error lost evidence or retried", values, err, calls.Load())
	}
}
