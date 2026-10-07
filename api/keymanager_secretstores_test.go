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
	th "github.com/gophercloud/gophercloud/v2/testhelper"
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

func TestKeyManagerSecretStoresSemanticBodyAttributesUseOriginalFields(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	encodedRef := "https://foreign.invalid/v1/secret-stores/store%2fpart?tenant=kept#fragment"
	plainRef := "https://foreign.invalid/v1/secret-stores/plain-b"
	created := "2016-08-22T23:46:45.114283+09:00"
	rows := strings.Join([]string{
		secretStoreRow(t, encodedRef, map[string]any{"id": "literal-a", "created": created, "updated": "raw-updated", "label": "A"}),
		secretStoreRow(t, plainRef, map[string]any{"created": nil, "label": "B"}),
		secretStoreRow(t, "https://foreign.invalid/c", map[string]any{"id": nil, "created": "", "updated": nil, "label": "C"}),
		secretStoreRow(t, "https://foreign.invalid/d", map[string]any{"id": "", "updated": "", "label": "D"}),
		secretStoreRow(t, "", map[string]any{"secret_store_ref": nil, "label": "E"}),
		`{"name":"Different","status":"ACTIVE","label":"F"}`,
	}, ",")
	cloud.Mux.HandleFunc("GET "+secretStoresPath, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, http.MethodGet)
		th.TestHeader(t, r, "X-Auth-Token", "test-token")
		calls.Add(1)
		if r.URL.RawQuery != "" {
			t.Error("local Body attributes leaked to the wire", r.URL)
		}
		testcloud.JSON(w, 200, secretStorePage(rows, ""))
	})
	a := secretstores.New(secretStoresClient(cloud))
	for _, tc := range []struct {
		name, field string
		value       any
		labels      []string
	}{
		{"id-is-full-ref", "id", plainRef, []string{"B"}},
		{"literal-id-priority", "id", "literal-a", []string{"A"}},
		{"id-is-not-convenience-id", "id", "plain-b", nil},
		{"literal-null-and-missing-ref", "id", nil, []string{"C", "E", "F"}},
		{"literal-empty-is-not-null", "id", "", []string{"D"}},
		{"derived-id-ignores-literal-id", "secret_store_id", "store%2fpart", []string{"A"}},
		{"derived-id-null-and-omission", "secret_store_id", nil, []string{"E", "F"}},
		{"full-ref-retains-query-fragment", "secret_store_ref", encodedRef, []string{"A"}},
		{"ref-null-and-omission", "secret_store_ref", nil, []string{"E", "F"}},
		{"created-original-text", "created_at", created, []string{"A"}},
		{"created-is-not-parsed", "created_at", "2016-08-22T14:46:45.114283Z", nil},
		{"created-null-and-omission", "created_at", nil, []string{"B", "D", "E", "F"}},
		{"created-empty-is-not-null", "created_at", "", []string{"C"}},
		{"updated-original-text", "updated_at", "raw-updated", []string{"A"}},
		{"updated-null-and-omission", "updated_at", nil, []string{"B", "C", "E", "F"}},
		{"updated-empty-is-not-null", "updated_at", "", []string{"D"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := calls.Load()
			values, err := a.All(context.Background(), secretstores.WithListFilter(tc.field, tc.value))
			var labels, ids []string
			for _, label := range tc.labels {
				ids = append(ids, map[string]string{"A": "literal-a", "B": "plain-b"}[label])
			}
			secretStoreWantIDs(t, values, err, ids...)
			for _, value := range values {
				var label string
				if err := json.Unmarshal(value.Body["label"], &label); err != nil {
					t.Fatal(err)
				}
				labels = append(labels, label)
			}
			if !reflect.DeepEqual(labels, tc.labels) || calls.Load() != before+1 {
				t.Fatal("wrong original-field matches or passive reference follow", labels, tc.labels, calls.Load()-before)
			}
		})
	}
	t.Run("only-selected-formatter-rejects-empty-ref", func(t *testing.T) {
		cloud := testcloud.New(t)
		var requests atomic.Int32
		cloud.Mux.HandleFunc("GET "+secretStoresPath, func(w http.ResponseWriter, r *http.Request) {
			requests.Add(1)
			th.TestMethod(t, r, http.MethodGet)
			th.TestHeader(t, r, "X-Auth-Token", "test-token")
			testcloud.JSON(w, 200, secretStorePage(secretStoreRow(t, "", map[string]any{"id": "literal"}), ""))
		})
		api := secretstores.New(secretStoresClient(cloud))
		values, err := api.All(context.Background(), secretstores.WithListFilter("id", "literal"))
		secretStoreWantIDs(t, values, err, "literal")
		values, err = api.All(context.Background(), secretstores.WithListFilter("secret_store_id", "literal"))
		if values != nil || !errors.Is(err, resource.ErrInvalidOption) || requests.Load() != 2 {
			t.Fatal("selected HREF formatter did not fail once", values, err, requests.Load())
		}
	})

}

func TestKeyManagerSecretStoresSemanticQueryOwnershipAndBodySeparation(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	want := url.Values{"limit": {"7"}, "marker": {"start"}, "name": {"server-pattern"}, "status": {"READY"}, "global_default": {"false"}, "crypto_plugin": {"alpha", "beta"}, "secret_store_plugin": {""}, "created": {"wire-created-a", "wire-created-b"}, "updated": {"wire-updated"}, "created_at": {"raw-wire-extension"}, "vendor": {"kept"}}
	cloud.Mux.HandleFunc("GET "+secretStoresPath, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, http.MethodGet)
		th.TestHeader(t, r, "X-Auth-Token", "test-token")
		calls.Add(1)
		if !reflect.DeepEqual(r.URL.Query(), want) {
			t.Error("query attributes, original captured values or Body separation changed", r.URL.Query(), want)
		}
		// Declared queries belong to the server: differing response values must
		// not be rechecked locally. Only created_at/secret_store_id are local.
		testcloud.JSON(w, 200, secretStorePage(secretStoreRow(t, "https://foreign.invalid/store-one", map[string]any{"global_default": true, "crypto_plugin": "different", "secret_store_plugin": "different", "created": "raw-body", "updated": "different"}), "https://foreign.invalid/not-read"))
	})
	plugins := []string{"alpha", "beta"}
	fields := map[string]any{"limit": 7, "marker": "start", "name": "server-pattern", "status": "READY", "global_default": false, "crypto_plugin": plugins, "secret_store_plugin": "", "created": []string{"wire-created-a", "wire-created-b"}, "updated": "wire-updated", "created_at": "raw-body", "secret_store_id": "store-one", "location": func() {}}
	semantic := secretstores.WithListFilters(fields)
	plugins[0], fields["global_default"], fields["created_at"] = "caller-change", true, "caller-change"
	delete(fields, "secret_store_id")
	falseValue := false
	options := []secretstores.ListOption{semantic, secretstores.WithListOptions(secretstores.ListOpts{Paginated: &falseValue}), secretstores.WithListQuery("created_at", "raw-wire-extension"), secretstores.WithListQuery("vendor", "kept")}
	a := secretstores.New(secretStoresClient(cloud))
	seq := a.List(context.Background(), options...)
	options[0] = secretstores.WithListFilters(map[string]any{"created_at": "caller-change"})
	if calls.Load() != 0 {
		t.Fatal("semantic iterator construction performed HTTP")
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
	var group sync.WaitGroup
	for range 4 {
		group.Add(1)
		go func() {
			defer group.Done()
			values, err := a.All(context.Background(), semantic, secretstores.WithListPaginated(false), secretstores.WithListQuery("created_at", "raw-wire-extension"), secretstores.WithListQuery("vendor", "kept"))
			if err != nil || len(values) != 1 || values[0].ID != "store-one" {
				t.Error(values, err)
			}
		}()
	}
	group.Wait()
	if calls.Load() != 6 {
		t.Fatal("semantic option reuse performed extra HTTP", calls.Load())
	}
}

func TestKeyManagerSecretStoresSemanticLazyControlsCollisionsAndClear(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET "+secretStoresPath, func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, http.MethodGet)
		th.TestHeader(t, r, "X-Auth-Token", "test-token")
		calls.Add(1)
		want := url.Values{"vendor": {"wire"}}
		if r.URL.Query().Has("name") {
			want.Set("name", "raw")
		}
		if !reflect.DeepEqual(r.URL.Query(), want) {
			t.Error("semantic clear changed raw query or leaked local/unknown fields", r.URL.Query(), want)
		}
		testcloud.JSON(w, 200, secretStorePage(secretStoreRow(t, "https://foreign.invalid/kept", map[string]any{"created": "keep"}), ""))
	})
	client := secretStoresClient(cloud)
	a := secretstores.New(client)
	for _, tc := range []struct {
		name    string
		options []secretstores.ListOption
		want    error
	}{
		{"local-control", []secretstores.ListOption{secretstores.WithListFilter("max_items", 1)}, resource.ErrInvalidOption},
		{"session-control", []secretstores.ListOption{secretstores.WithListFilter("session", true)}, resource.ErrUnsupported},
		{"typed-limit-collision", []secretstores.ListOption{secretstores.WithListOptions(secretstores.ListOpts{Limit: 3}), secretstores.WithListFilter("limit", 3)}, resource.ErrInvalidOption},
		{"raw-name-collision", []secretstores.ListOption{secretstores.WithListQuery("name", "same"), secretstores.WithListFilter("name", "same")}, resource.ErrInvalidOption},
		{"null-query-presence-collision", []secretstores.ListOption{secretstores.WithListFilter("name", nil), secretstores.WithListQuery("name", "")}, resource.ErrInvalidOption},
		{"selected-local-encoding-error", []secretstores.ListOption{secretstores.WithListFilter("created_at", func() {})}, resource.ErrInvalidOption},
		{"options-cannot-replace-source", []secretstores.ListOption{secretstores.WithListFilter("created_at", "keep"), func(config *request.Config[secretstores.ListOpts]) error { client.Endpoint += "changed/"; return nil }}, resource.ErrInvalidOption},
	} {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := client.Endpoint
			defer func() { client.Endpoint = endpoint }()
			seq := a.List(context.Background(), tc.options...)
			if calls.Load() != 0 {
				t.Fatal("semantic preflight was eager")
			}
			var seen int
			for value, err := range seq {
				seen++
				if value != nil || !errors.Is(err, tc.want) {
					t.Fatal(value, err)
				}
				if tc.name == "selected-local-encoding-error" {
					var cause *json.UnsupportedTypeError
					if !errors.As(err, &cause) {
						t.Fatal("selected JSON cause was lost", err)
					}
				}
			}
			if seen != 1 || calls.Load() != 0 {
				t.Fatal("invalid semantic option reached HTTP", seen, calls.Load())
			}
		})
	}
	for _, options := range [][]secretstores.ListOption{
		{secretstores.WithListFilters(map[string]any{"created_at": "keep", "location": func() {}, "vendor": json.RawMessage(`{]`)})},
		{secretstores.WithListFilter("created_at", func() {}), secretstores.WithListFilter("session", true), secretstores.WithListFilters(map[string]any{"created_at": "keep"})},
		{secretstores.WithListFilter("created_at", func() {}), secretstores.WithListFilter("created_at", "keep")},
		{secretstores.WithListFilter("name", "ignored"), secretstores.WithListFilter("created_at", "not-kept"), secretstores.WithListFilters(nil), secretstores.WithListQuery("name", "raw")},
	} {
		options = append(options, secretstores.WithListQuery("vendor", "wire"))
		values, err := a.All(context.Background(), options...)
		secretStoreWantIDs(t, values, err, "kept")
	}
	if calls.Load() != 4 {
		t.Fatal("unknown/discarded values or semantic replacement changed HTTP", calls.Load())
	}
}

func TestKeyManagerSecretStoresSemanticRawCapAndContinuation(t *testing.T) {
	for _, mode := range []string{"all-pages", "cap-one", "cap-two", "single-page", "filtered-marker", "late-404", "break"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			firstRef := "https://foreign.invalid/first%2fraw?query=kept#fragment"
			cloud.Mux.HandleFunc("GET "+secretStoresPath, func(w http.ResponseWriter, r *http.Request) {
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "test-token")
				page := calls.Add(1)
				want := url.Values{"vendor": {"kept"}}
				switch mode {
				case "cap-one":
					want.Set("limit", "1")
				case "cap-two":
					want.Set("limit", "2")
				case "filtered-marker":
					want.Set("limit", "9")
				}
				if page > 1 {
					marker := "second"
					if mode == "filtered-marker" {
						marker = firstRef
					}
					want.Set("marker", marker)
				}
				if !reflect.DeepEqual(r.URL.Query(), want) {
					t.Error("raw-row cap, full-ref marker or preserved query changed", r.URL.Query(), want)
				}
				if page > 1 {
					if mode == "late-404" {
						testcloud.JSON(w, 404, `{"error":"next-page"}`)
						return
					}
					rows := ""
					if mode != "filtered-marker" {
						rows = secretStoreRow(t, "https://foreign.invalid/later", map[string]any{"created": "keep"})
					}
					testcloud.JSON(w, 200, secretStorePage(rows, ""))
					return
				}
				rows := secretStoreRow(t, firstRef, map[string]any{"created": "not-kept"})
				next := cloud.Server.URL + secretStoresPath + "?vendor=kept&marker=second"
				if mode == "filtered-marker" {
					next = ""
				} else {
					rows += "," + secretStoreRow(t, "https://foreign.invalid/same-page", map[string]any{"created": "keep"})
					if mode == "cap-one" || mode == "cap-two" || mode == "single-page" || mode == "break" {
						next = "https://foreign.invalid/must-not-follow"
					}
				}
				testcloud.JSON(w, 200, secretStorePage(rows, next))
			})
			options := []secretstores.ListOption{secretstores.WithListFilter("created_at", "keep"), secretstores.WithListQuery("vendor", "kept")}
			wantIDs, wantCalls := []string{"same-page", "later"}, int32(2)
			switch mode {
			case "cap-one":
				options = append(options, secretstores.WithListMaxItems(1))
				wantIDs, wantCalls = nil, 1
			case "cap-two":
				options = append(options, secretstores.WithListMaxItems(2))
				wantIDs, wantCalls = []string{"same-page"}, 1
			case "single-page":
				options = append(options, secretstores.WithListPaginated(false))
				wantIDs, wantCalls = []string{"same-page"}, 1
			case "filtered-marker":
				options = append(options, secretstores.WithListOptions(secretstores.ListOpts{Limit: 9}))
				wantIDs = nil
			case "late-404":
				wantIDs = []string{"same-page"}
			case "break":
				wantIDs, wantCalls = []string{"same-page"}, 1
			}
			a := secretstores.New(secretStoresClient(cloud))
			var values []*secretstores.SecretStore
			var terminal error
			for value, err := range a.List(context.Background(), options...) {
				if err != nil {
					if value != nil {
						t.Fatal(value, err)
					}
					terminal = err
					break
				}
				values = append(values, value)
				if mode == "break" {
					break
				}
			}
			if mode == "late-404" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(terminal, &native) || native.Actual != 404 || string(native.Body) != `{"error":"next-page"}` {
					t.Fatal("filtered iterator lost late HTTP error", terminal)
				}
			} else if terminal != nil {
				t.Fatal(terminal)
			}
			secretStoreWantIDs(t, values, nil, wantIDs...)
			if calls.Load() != wantCalls {
				t.Fatal("cap refilled filtered rows or fetched an ignored continuation", calls.Load(), wantCalls)
			}
		})
	}
}
