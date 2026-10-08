package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// openstacksdk ef55d7d: find_secret _proxy.py:302–320 delegates Resource.find
// (2491–2582). Its direct fetch invokes Secret.fetch, but list matches do not.
const secretFindCollectionPath = "/reverse/barbican/v1/secrets"

func secretFindPage(rows, next string) string {
	body := `{"secrets":[` + rows + `]}`
	if next != "" {
		encoded, _ := json.Marshal(next)
		body = `{"secrets":[` + rows + `],"next":` + string(encoded) + `}`
	}
	return body
}

func TestKeyManagerSecretFindDirectFetchAndSeedIdentity(t *testing.T) {
	for _, test := range []struct {
		name, body string
		payload    bool
	}{
		{"absent response ID", `{"name":"actual","secret_ref":"https://foreign.invalid/secrets/other","content_types":{"default":"text/plain"},"vendor":9007199254740993}`, true},
		{"null response ID", `{"id":null,"name":"actual","secret_ref":"https://foreign.invalid/secrets/other"}`, false},
		{"changed response ID", `{"id":"server-other","name":"actual"}`, false},
		{"case ID is extension", `{"ID":false,"name":"actual","secret_ref":null}`, false},
		{"nonstring passive ID", `{"id":{"untyped":true},"name":"actual"}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, payloads, lists atomic.Int32
			cloud.Mux.HandleFunc(secretFetchPath, func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if r.Method != "GET" || r.URL.RawQuery != "" || r.Header.Get("Accept") != "application/json" {
					t.Error(r.Method, r.URL, r.Header)
				}
				w.Header().Set("X-Phase", "metadata")
				testcloud.JSON(w, 200, test.body)
			})
			cloud.Mux.HandleFunc(secretFetchPath+"/payload", func(w http.ResponseWriter, r *http.Request) {
				payloads.Add(1)
				if r.Header.Get("Accept") != "text/plain" {
					t.Error(r.Header)
				}
				w.Header().Set("X-Phase", "payload")
				_, _ = w.Write([]byte("original payload"))
			})
			cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) { lists.Add(1); w.WriteHeader(500) })
			value, err := secrets.New(secretFetchClient(cloud)).FindIdentity(context.Background(), "secret-alpha")
			if err != nil || value == nil || value.SecretID != "secret-alpha" || value.Name != "actual" || value.StatusCode != 200 || value.Header.Get("X-Phase") != "metadata" || gets.Load() != 1 || lists.Load() != 0 || (value.Payload != nil) != test.payload || (payloads.Load() == 1) != test.payload {
				t.Fatal(value, err, gets.Load(), lists.Load(), payloads.Load())
			}
			if test.payload && (value.Payload.Text == nil || *value.Payload.Text != "original payload" || value.Payload.Header.Get("X-Phase") != "payload" || string(value.Body["vendor"]) != "9007199254740993") {
				t.Fatal(value)
			}
			if test.name == "absent response ID" {
				if _, fabricated := value.Body["id"]; fabricated {
					t.Fatal(value.Body)
				}
			}
		})
	}
}

func TestKeyManagerSecretFindFallbackAndMissingPolicies(t *testing.T) {
	for _, test := range []struct {
		name                          string
		getCode, listCode             int
		policy                        resource.FindFallbackPolicy
		strict, empty, payloadFailure bool
		wantList, found, failure      bool
	}{
		{name: "compatible400", getCode: 400, wantList: true, found: true},
		{name: "compatible403", getCode: 403, wantList: true, found: true},
		{name: "compatible404", getCode: 404, wantList: true, found: true},
		{name: "notfoundonly400", getCode: 400, policy: resource.FindFallbackNotFoundOnly, failure: true},
		{name: "notfoundonly403", getCode: 403, policy: resource.FindFallbackNotFoundOnly, failure: true},
		{name: "notfoundonly404", getCode: 404, policy: resource.FindFallbackNotFoundOnly, wantList: true, found: true},
		{name: "never400", getCode: 400, policy: resource.FindFallbackNever, failure: true},
		{name: "never404 ignored", getCode: 404, policy: resource.FindFallbackNever},
		{name: "never404 strict", getCode: 404, policy: resource.FindFallbackNever, strict: true, failure: true},
		{name: "list missing ignored", getCode: 404, wantList: true, empty: true},
		{name: "list missing strict", getCode: 403, wantList: true, empty: true, strict: true, failure: true},
		{name: "list403 terminal", getCode: 404, listCode: 403, wantList: true, failure: true},
		{name: "list404 terminal", getCode: 404, listCode: 404, wantList: true, failure: true},
		{name: "payload400 fallback", getCode: 400, payloadFailure: true, wantList: true, found: true},
		{name: "payload403 fallback", getCode: 403, payloadFailure: true, wantList: true, found: true},
		{name: "payload404 fallback", getCode: 404, payloadFailure: true, wantList: true, found: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, lists, payloads atomic.Int32
			cloud.Mux.HandleFunc(secretFetchPath, func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				if test.payloadFailure {
					testcloud.JSON(w, 200, `{"name":"first response","content_types":{"default":"text/plain"}}`)
					return
				}
				testcloud.JSON(w, test.getCode, `{"error":"direct failure"}`)
			})
			cloud.Mux.HandleFunc(secretFetchPath+"/payload", func(w http.ResponseWriter, r *http.Request) {
				payloads.Add(1)
				testcloud.JSON(w, test.getCode, `{"error":"payload failure"}`)
			})
			cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("name") != "secret-alpha" || r.URL.Query().Has("limit") || r.URL.Query().Has("offset") {
					t.Error(r.URL)
				}
				if test.listCode != 0 {
					testcloud.JSON(w, test.listCode, `{"error":"list failure"}`)
					return
				}
				rows := `{"id":"canonical-row","name":"secret-alpha","secret_ref":"https://foreign.invalid/secrets/never","content_types":{"default":"text/plain"},"vendor":9007199254740993}`
				if test.empty {
					rows = ""
				}
				w.Header().Set("X-Phase", "list")
				testcloud.JSON(w, 200, secretFindPage(rows, ""))
			})
			value, err := secrets.New(secretFetchClient(cloud)).FindIdentity(context.Background(), "secret-alpha", resource.WithIdentityFindFallback(test.policy), resource.WithIdentityFindIgnoreMissing(!test.strict))
			if (value != nil) != test.found || (err != nil) != test.failure || gets.Load() != 1 || (lists.Load() == 1) != test.wantList || (payloads.Load() == 1) != test.payloadFailure {
				t.Fatal(value, err, gets.Load(), lists.Load(), payloads.Load())
			}
			if value != nil && (value.SecretID != "" || value.Payload != nil || value.Name != "secret-alpha" || value.Header.Get("X-Phase") != "list" || string(value.Body["vendor"]) != "9007199254740993") {
				t.Fatal(value)
			}
			if test.strict && test.failure && (test.empty || !test.wantList) && !errors.Is(err, resource.ErrNotFound) {
				t.Fatal(err)
			}
			if test.listCode != 0 {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != test.listCode {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestKeyManagerSecretFindPassiveIDAndExactName(t *testing.T) {
	for _, test := range []struct {
		name, identity, rows, ref string
		found, duplicate          bool
	}{
		{"full ref unique same suffix", "https://foreign.invalid/secrets/shared", `{"name":"different","secret_ref":"https://foreign.invalid/secrets/shared"},{"name":"another","secret_ref":"https://other.invalid/secrets/shared"}`, "https://foreign.invalid/secrets/shared", true, false},
		{"same full ref duplicate", "https://foreign.invalid/secrets/shared", `{"name":"different","secret_ref":"https://foreign.invalid/secrets/shared"},{"name":"another","secret_ref":"https://foreign.invalid/secrets/shared"}`, "", false, true},
		{"null literal blocks alternate", "https://foreign.invalid/secrets/shared", `{"id":null,"name":"different","secret_ref":"https://foreign.invalid/secrets/shared"}`, "", false, false},
		{"null literal name match", " exact name ", `{"id":null,"name":" exact name ","secret_ref":"https://foreign.invalid/secrets/shared"}`, "https://foreign.invalid/secrets/shared", true, false},
		{"empty literal name match", "exact name", `{"id":"","name":"exact name"}`, "", true, false},
		{"number literal is not coerced", "7", `{"id":7,"name":"different","secret_ref":"https://foreign.invalid/secrets/7"}`, "", false, false},
		{"object literal name match", "exact name", `{"id":{"untyped":true},"name":"exact name"}`, "", true, false},
		{"literal encoded slash name", "name%2Fpart", `{"name":"name%2Fpart","secret_ref":"https://foreign.invalid/secret"}`, "https://foreign.invalid/secret", true, false},
		{"canonical ID only", "secret-alpha", `{"id":"secret-alpha","name":"different","secret_ref":"https://foreign.invalid/other"}`, "https://foreign.invalid/other", true, false},
		{"no UUID derivation", "shared", `{"name":"different","secret_ref":"https://foreign.invalid/secrets/shared"}`, "", false, false},
		{"case aliases ignored", "secret-alpha", `{"id":null,"ID":"secret-alpha","name":null,"NAME":"secret-alpha","secret_ref":"https://foreign.invalid/other"}`, "", false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, lists atomic.Int32
			cloud.Mux.HandleFunc(secretFindCollectionPath+"/", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); testcloud.JSON(w, 400, `{}`) })
			cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("name") != test.identity || len(r.URL.Query()) != 1 {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, secretFindPage(test.rows, ""))
			})
			value, err := secrets.New(secretFetchClient(cloud)).FindIdentity(context.Background(), test.identity)
			if (value != nil) != test.found || lists.Load() != 1 || (err != nil) != test.duplicate {
				t.Fatal(value, err, gets.Load(), lists.Load())
			}
			if value != nil && (value.SecretID != "" || value.Payload != nil || value.SecretRef != test.ref) {
				t.Fatal(value)
			}
			if test.duplicate {
				if !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(err)
				}
			}
			if strings.ContainsAny(test.identity, " /%") && gets.Load() != 0 {
				t.Fatal("unsafe identity used GET", gets.Load())
			}
		})
	}
}

func TestKeyManagerSecretFindAllPagesAndTerminalObservations(t *testing.T) {
	for _, mode := range []string{"unique", "duplicate same ID", "late403", "late404", "late typed decode", "late null row", "cycle", "backward", "changed filter", "foreign next", "empty ignores next", "HTTP Link"} {
		t.Run(mode, func(t *testing.T) {
			cloud, foreign := testcloud.New(t), testcloud.New(t)
			var gets, lists, followed atomic.Int32
			foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { followed.Add(1); w.WriteHeader(500) })
			cloud.Mux.HandleFunc(secretFetchPath, func(w http.ResponseWriter, r *http.Request) { gets.Add(1); testcloud.JSON(w, 404, `{}`) })
			cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("name") != "secret-alpha" {
					t.Error(r.URL)
				}
				w.Header().Set("X-Page", fmt.Sprint(lists.Load()))
				if !r.URL.Query().Has("offset") {
					if r.URL.Query().Has("limit") {
						t.Error("invented initial limit", r.URL)
					}
					next := cloud.Server.URL + secretFindCollectionPath + "?offset=10&limit=10"
					if mode == "changed filter" {
						next += "&name=other"
					}
					if mode == "foreign next" {
						next = foreign.Server.URL + secretFindCollectionPath + "?offset=10&limit=10"
					}
					if mode == "HTTP Link" {
						w.Header().Set("Link", "<"+next+">; rel=\"next\"")
						next = ""
					}
					testcloud.JSON(w, 200, secretFindPage(`{"id":"same","name":"secret-alpha","content_types":{"default":"text/plain"},"vendor":9007199254740993}`, next))
					return
				}
				if r.URL.Query().Get("offset") != "10" || r.URL.Query().Get("limit") != "10" {
					t.Error(r.URL)
				}
				rows, next := `{"id":"other","name":"decoy"}`, ""
				switch mode {
				case "late403":
					testcloud.JSON(w, 403, `{"error":"late"}`)
					return
				case "late404":
					testcloud.JSON(w, 404, `{"error":"late"}`)
					return
				case "late typed decode":
					rows = `{"name":false}`
				case "late null row":
					rows = `null`
				case "duplicate same ID":
					rows = `{"id":"same","name":"secret-alpha"}`
				case "cycle":
					next = cloud.Server.URL + secretFindCollectionPath + "?offset=10&limit=10"
				case "backward":
					next = cloud.Server.URL + secretFindCollectionPath + "?offset=0&limit=10"
				case "empty ignores next":
					rows = ""
					next = foreign.Server.URL + "/never-follow"
				}
				testcloud.JSON(w, 200, secretFindPage(rows, next))
			})
			value, err := secrets.New(secretFetchClient(cloud)).FindIdentity(context.Background(), "secret-alpha")
			found := mode == "unique" || mode == "empty ignores next" || mode == "HTTP Link"
			wantLists := int32(2)
			if mode == "foreign next" || mode == "changed filter" {
				wantLists = 1
			}
			if (value != nil) != found || (err == nil) != found || gets.Load() != 1 || lists.Load() != wantLists || followed.Load() != 0 {
				t.Fatal(value, err, gets.Load(), lists.Load(), followed.Load())
			}
			if found && (value.Payload != nil || value.SecretID != "" || value.Header.Get("X-Page") != "1" || string(value.Body["vendor"]) != "9007199254740993") {
				t.Fatal(value)
			}
			switch mode {
			case "duplicate same ID":
				if !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(err)
				}
			case "cycle":
				if !errors.Is(err, resource.ErrPaginationCycle) {
					t.Fatal(err)
				}
			case "backward", "changed filter", "foreign next":
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			case "late typed decode", "late null row":
				var accepted *resource.ResponseError
				if !errors.As(err, &accepted) || accepted.Header.Get("X-Page") != "2" || accepted.StatusCode != 200 || !strings.Contains(string(accepted.Body), `"secrets"`) {
					t.Fatal(err)
				}
			}
		})
	}
	// No limit and no advertised continuation stops; Find does not fabricate
	// UUID markers or perform a final fetch after finding a metadata-only row.
	t.Run("no synthetic marker", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			testcloud.JSON(w, 200, secretFindPage(`{"name":"exact name"}`, ""))
		})
		value, err := secrets.New(secretFetchClient(cloud)).FindIdentity(context.Background(), "exact name")
		if err != nil || value == nil || value.SecretID != "" || value.Payload != nil || calls.Load() != 1 {
			t.Fatal(value, err, calls.Load())
		}
	})
}

func TestKeyManagerSecretFindPreparedOptionsAndConcurrentReuse(t *testing.T) {
	cloud := testcloud.New(t)
	client := secretFetchClient(cloud)
	var applications, gets, lists atomic.Int32
	var retained *resource.IdentityFindOpts
	custom := resource.IdentityFindOption(func(value *resource.IdentityFindOpts) error {
		applications.Add(1)
		ignore := true
		value.IgnoreMissing = &ignore
		retained = value
		return nil
	})
	transport := cloud.Provider.HTTPClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	cloud.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		response, err := transport.RoundTrip(r)
		if r.URL.Path == secretFetchPath {
			*retained.IgnoreMissing = false
			retained.Fallback = resource.FindFallbackNever
		}
		return response, err
	})
	cloud.Mux.HandleFunc(secretFetchPath, func(w http.ResponseWriter, r *http.Request) { gets.Add(1); testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		testcloud.JSON(w, 200, secretFindPage("", ""))
	})
	value, err := secrets.New(client).FindIdentity(context.Background(), "secret-alpha", custom)
	if value != nil || err != nil || applications.Load() != 1 || gets.Load() != 1 || lists.Load() != 1 {
		t.Fatal(value, err, applications.Load(), gets.Load(), lists.Load())
	}
	// Construction owns the IgnoreMissing pointer; applying the same option
	// concurrently creates independent prepared snapshots for each lookup.
	fixture := testcloud.New(t)
	ignore := false
	option := resource.WithIdentityFindOptions(resource.IdentityFindOpts{IgnoreMissing: &ignore})
	ignore = true
	var count atomic.Int32
	fixture.Mux.HandleFunc(secretFetchPath, func(w http.ResponseWriter, r *http.Request) { count.Add(1); testcloud.JSON(w, 404, `{}`) })
	fixture.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		testcloud.JSON(w, 200, secretFindPage("", ""))
	})
	api := secrets.New(secretFetchClient(fixture))
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, err := api.FindIdentity(context.Background(), "secret-alpha", option)
			if value != nil || !errors.Is(err, resource.ErrNotFound) {
				t.Error(value, err)
			}
		}()
	}
	wg.Wait()
	if count.Load() != 16 {
		t.Fatal(count.Load())
	}
}

func TestKeyManagerSecretFindFetchFailuresAreTerminal(t *testing.T) {
	for _, mode := range []string{"metadata203", "metadata204", "malformed metadata", "selection failure", "wrong typed extension", "payload203", "payload204", "invalid UTF8", "read", "transport404", "canceled404"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := secretFetchClient(cloud)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var gets, payloads, lists atomic.Int32
			cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) { lists.Add(1); w.WriteHeader(500) })
			cloud.Mux.HandleFunc(secretFetchPath, func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				w.Header().Set("X-Phase", "metadata")
				code, body := 200, `{"name":"actual","content_types":{"default":"text/plain"}}`
				switch mode {
				case "metadata203":
					code = 203
				case "metadata204":
					code = 204
				case "malformed metadata":
					body = `{"name":"truncated"`
				case "selection failure":
					body = `{"name":"actual","content_types":null}`
				case "wrong typed extension":
					body = `{"name":false}`
				}
				testcloud.JSON(w, code, body)
			})
			cloud.Mux.HandleFunc(secretFetchPath+"/payload", func(w http.ResponseWriter, r *http.Request) {
				payloads.Add(1)
				w.Header().Set("X-Phase", "payload")
				code := 200
				if mode == "payload203" {
					code = 203
				}
				if mode == "payload204" {
					code = 204
				}
				w.WriteHeader(code)
				_, _ = w.Write([]byte{0xff})
			})
			readCause := errors.New("actual read failure")
			transportCause := &gophercloud.ErrUnexpectedResponseCode{Actual: 404}
			if mode == "read" || mode == "transport404" || mode == "canceled404" {
				cloud.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					gets.Add(1)
					if mode == "transport404" {
						return nil, transportCause
					}
					if mode == "canceled404" {
						cancel()
						return &http.Response{StatusCode: 404, Header: http.Header{"X-Phase": {"metadata"}}, Body: io.NopCloser(strings.NewReader(`{"error":"actual"}`)), Request: r}, nil
					}
					return &http.Response{StatusCode: 200, Header: http.Header{"X-Phase": {"metadata"}}, Body: &secretFetchReadFailure{cause: readCause}, Request: r}, nil
				})
			}
			value, err := secrets.New(client).FindIdentity(ctx, "secret-alpha")
			if value != nil || err == nil || gets.Load() != 1 || lists.Load() != 0 {
				t.Fatal(value, err, gets.Load(), lists.Load(), payloads.Load())
			}
			switch mode {
			case "malformed metadata", "selection failure", "wrong typed extension", "invalid UTF8", "read":
				var accepted *resource.ResponseError
				if !errors.As(err, &accepted) || accepted.StatusCode != 200 {
					t.Fatal(err)
				}
				if mode == "invalid UTF8" && (len(accepted.Body) != 1 || accepted.Body[0] != 0xff || accepted.Header.Get("X-Phase") != "payload" || payloads.Load() != 1) {
					t.Fatal(err, payloads.Load())
				}
				if mode == "read" && (!errors.Is(err, readCause) || string(accepted.Body) != "partial bytes") {
					t.Fatal(err)
				}
			case "transport404":
				var transport *url.Error
				if !errors.As(err, &transport) || !errors.Is(err, transportCause) {
					t.Fatal(err)
				}
			case "canceled404":
				if !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
			default:
				var native gophercloud.ErrUnexpectedResponseCode
				want := 203
				if strings.HasSuffix(mode, "204") {
					want = 204
				}
				if !errors.As(err, &native) || native.Actual != want {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestKeyManagerSecretFindSourceSnapshotAndPageRevalidation(t *testing.T) {
	t.Run("fixed prefix original provider current token", func(t *testing.T) {
		cloud, foreign := testcloud.New(t), testcloud.New(t)
		client := secretFetchClient(cloud)
		client.MoreHeaders = map[string]string{"X-Trace": "original", "X-Project-Id": "fixed-project"}
		originalProvider := client.ProviderClient
		var gets, lists, foreignCalls, middleware atomic.Int32
		foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { foreignCalls.Add(1); w.WriteHeader(500) })
		transport := cloud.Provider.HTTPClient.Transport
		if transport == nil {
			transport = http.DefaultTransport
		}
		cloud.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
			middleware.Add(1)
			response, err := transport.RoundTrip(r)
			if r.URL.Path == secretFetchPath {
				originalProvider.SetToken("list-token")
				client.ResourceBase = cloud.Server.URL + "/changed/"
				client.MoreHeaders["X-Trace"] = "changed"
				client.ProviderClient = foreign.Provider
			}
			return response, err
		})
		cloud.Mux.HandleFunc(secretFetchPath, func(w http.ResponseWriter, r *http.Request) { gets.Add(1); testcloud.JSON(w, 404, `{}`) })
		cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
			lists.Add(1)
			if r.Header.Get("X-Auth-Token") != "list-token" || r.Header.Get("X-Trace") != "original" || r.Header.Get("X-Project-Id") != "fixed-project" {
				t.Error(r.Header)
			}
			testcloud.JSON(w, 200, secretFindPage(`{"name":"secret-alpha","secret_ref":"`+foreign.Server.URL+`/secrets/never"}`, ""))
		})
		api := secrets.New(client)
		value, err := api.FindIdentity(context.Background(), "secret-alpha")
		if err != nil || value == nil || value.SecretID != "" || value.Payload != nil || gets.Load() != 1 || lists.Load() != 1 || foreignCalls.Load() != 0 || middleware.Load() != 2 || api.RawClient() != client {
			t.Fatal(value, err, gets.Load(), lists.Load(), foreignCalls.Load(), middleware.Load())
		}
	})
	for _, mode := range []string{"source changes", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := secretFetchClient(cloud)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var gets, lists atomic.Int32
			transport := cloud.Provider.HTTPClient.Transport
			if transport == nil {
				transport = http.DefaultTransport
			}
			cloud.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				response, err := transport.RoundTrip(r)
				if r.URL.Path == secretFindCollectionPath {
					if mode == "source changes" {
						client.Type = "compute"
					} else {
						cancel()
					}
				}
				return response, err
			})
			cloud.Mux.HandleFunc(secretFetchPath, func(w http.ResponseWriter, r *http.Request) { gets.Add(1); testcloud.JSON(w, 404, `{}`) })
			cloud.Mux.HandleFunc(secretFindCollectionPath, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				testcloud.JSON(w, 200, secretFindPage(`{"name":"secret-alpha"}`, cloud.Server.URL+secretFindCollectionPath+"?offset=10&limit=10"))
			})
			value, err := secrets.New(client).FindIdentity(ctx, "secret-alpha")
			if value != nil || err == nil || gets.Load() != 1 || lists.Load() != 1 {
				t.Fatal(value, err, gets.Load(), lists.Load())
			}
			if mode == "source changes" && !errors.Is(err, resource.ErrUnsupported) || mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}

func TestKeyManagerSecretFindCapabilityPreflightAndExistingRef(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, `{}`) })
	for _, identity := range []string{"secret-alpha", "unsafe name"} {
		for _, option := range []resource.IdentityFindOption{
			resource.WithIdentityFindQuery("name", "caller"),
			resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: url.Values{"name": nil}}),
			resource.WithIdentityFindDetails(true), resource.WithIdentityFindDetails(false),
			resource.WithIdentityFindAllProjects(true), resource.WithIdentityFindAllProjects(false),
			resource.WithIdentityFindExtraSpecs(true), resource.WithIdentityFindExtraSpecs(false),
		} {
			value, err := secrets.New(secretFetchClient(cloud)).FindIdentity(context.Background(), identity, option)
			if value != nil || !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 0 {
				t.Fatal(identity, value, err, calls.Load())
			}
		}
	}
	for _, identity := range []string{"", " \t", "a\x00b", string([]byte{0xff})} {
		if _, err := secrets.New(secretFetchClient(cloud)).FindIdentity(context.Background(), identity); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(identity, err)
		}
	}
	api := secrets.New(secretFetchClient(cloud))
	if _, err := api.FindIdentity(context.Background(), "unsafe name", resource.WithIdentityFindFallback(resource.FindFallbackNever)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.FindIdentity(context.Background(), "secret-alpha", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.FindIdentity(context.Background(), "secret-alpha", resource.WithIdentityFindFallback(9)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.FindIdentity(context.Background(), "secret-alpha", resource.WithIdentityFindQuery("max_items", "1")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.FindIdentity(nil, "secret-alpha"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.FindIdentity(ctx, "secret-alpha"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	var nilAPI *secrets.API
	if _, err := nilAPI.FindIdentity(context.Background(), "secret-alpha"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	client := secretFetchClient(cloud)
	var applied atomic.Int32
	_, err := secrets.New(client).FindIdentity(context.Background(), "secret-alpha", func(config *resource.IdentityFindOpts) error { applied.Add(1); client.Type = "compute"; return nil })
	if !errors.Is(err, resource.ErrUnsupported) || applied.Load() != 1 || calls.Load() != 0 {
		t.Fatal(err, applied.Load(), calls.Load())
	}
	t.Run("native Ref lookup remains metadata only", func(t *testing.T) {
		fixture := testcloud.New(t)
		var gets, payloads atomic.Int32
		fixture.Mux.HandleFunc(secretFetchPath, func(w http.ResponseWriter, r *http.Request) {
			gets.Add(1)
			testcloud.JSON(w, 200, `{"name":"native","secret_ref":"https://foreign.invalid/secrets/secret-alpha","content_types":{"default":"text/plain"}}`)
		})
		fixture.Mux.HandleFunc(secretFetchPath+"/payload", func(w http.ResponseWriter, r *http.Request) { payloads.Add(1); w.WriteHeader(500) })
		value, err := secrets.New(secretFetchClient(fixture)).Find(context.Background(), resource.ID("secret-alpha"))
		if err != nil || value == nil || value.Name != "native" || gets.Load() != 1 || payloads.Load() != 0 {
			t.Fatal(value, err, gets.Load(), payloads.Load())
		}
	})
}
