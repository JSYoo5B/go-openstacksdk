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
	"gophercloudsdk/keymanager/v1/secretconsumers"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// openstacksdk ef55d7d _proxy.py:544–610 and secret_consumer.py:22–67;
// Barbican's published secret_consumers reference uses GET/POST/DELETE 200.
const secretConsumersPath = "/reverse/barbican/v1/secrets/secret-alpha/consumers"
const secretConsumerRow = `{"service":"image","resource_type":"images","resource_id":"resource-alpha","created":"original timestamp","updated":null,"status":"ACTIVE","vendor":{"integer":9007199254740993}}`
const secretConsumerMutation = `{"name":"actual secret","status":"ACTIVE","secret_ref":"https://foreign.invalid/secrets/other","consumers":[` + secretConsumerRow + `],"created":"original timestamp","updated":null,"bit_length":9007199254740993}`

func secretConsumersClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("key-manager", "/catalog/v1")
	client.ResourceBase = cloud.Server.URL + "/reverse/barbican/v1/"
	return client
}

func secretConsumersScope(t *testing.T, client *gophercloud.ServiceClient) *secretconsumers.SecretScope {
	t.Helper()
	scope, err := secretconsumers.New(client).InSecret(context.Background(), resource.ID("secret-alpha"))
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func secretConsumersPage(rows, next string) string {
	if next == "" {
		return `{"total":2,"consumers":[` + rows + `]}`
	}
	encoded, _ := json.Marshal(next)
	return `{"total":2,"consumers":[` + rows + `],"next":` + string(encoded) + `}`
}

func TestKeyManagerSecretConsumersFixedParentAndActualMutationResponses(t *testing.T) {
	cloud, foreign := testcloud.New(t), testcloud.New(t)
	var calls, followed atomic.Int32
	foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { followed.Add(1); w.WriteHeader(500) })
	client := secretConsumersClient(cloud)
	client.MoreHeaders = map[string]string{"X-Configured": "preserved", "X-Project-Id": "selected-project"}
	scope := secretConsumersScope(t, client)
	if calls.Load() != 0 || scope.SecretID() != "secret-alpha" || scope.RawClient() != client || scope.RawClient().ProviderClient != cloud.Provider {
		t.Fatal(scope, calls.Load())
	}
	cloud.Mux.HandleFunc(secretConsumersPath, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "POST" && r.Method != "DELETE" {
			t.Error(r.Method)
		}
		if r.URL.RawQuery != "" || r.Header.Get("X-Configured") != "preserved" || r.Header.Get("X-Project-Id") != "selected-project" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(r.URL, r.Header)
		}
		body, _ := io.ReadAll(r.Body)
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(body, &fields); err != nil || len(fields) != 4 || string(fields["resource_type"]) != `"lbaas/loadbalancers"` || string(fields["resource_id"]) != `"caller/id?body-only"` || string(fields["vendor"]) != `{"n":9007199254740993}` {
			t.Error(string(body), err)
		}
		w.Header().Set("X-Request-ID", r.Method)
		response := strings.Replace(secretConsumerMutation, "https://foreign.invalid/secrets/other", foreign.Server.URL+"/never-follow", 1)
		testcloud.JSON(w, 200, response)
	})
	input := secretconsumers.ConsumerOpts{Service: "load-balancer", ResourceType: "lbaas/loadbalancers", ResourceID: "caller/id?body-only"}
	create, err := scope.Create(context.Background(), input, secretconsumers.WithCreateField("vendor", json.RawMessage(`{"n":9007199254740993}`)))
	if err != nil || create == nil || create.Name != "actual secret" || create.Status != "ACTIVE" || len(create.Consumers) != 1 || create.Consumers[0].ResourceID != "resource-alpha" || create.StatusCode != 200 || create.Header.Get("X-Request-ID") != "POST" || string(create.Body["bit_length"]) != "9007199254740993" || string(create.Consumers[0].Body["updated"]) != "null" {
		t.Fatal(create, err)
	}
	create.Body["name"][0] = 'x'
	create.Header.Set("X-Request-ID", "caller")
	create.Consumers[0].Body["vendor"][0] = 'x'
	*create.CreatedAt = "caller"
	deleted, err := scope.Delete(context.Background(), input, secretconsumers.WithDeleteField("vendor", json.RawMessage(`{"n":9007199254740993}`)))
	if err != nil || deleted == nil || deleted.Name != "actual secret" || deleted.Header.Get("X-Request-ID") != "DELETE" || *deleted.CreatedAt != "original timestamp" || string(deleted.Consumers[0].Body["vendor"]) != `{"integer":9007199254740993}` || calls.Load() != 2 || followed.Load() != 0 {
		t.Fatal(deleted, err, calls.Load(), followed.Load())
	}
	// No case-insensitive extension may replace canonical typed response fields.
	aliasCloud := testcloud.New(t)
	aliasCloud.Mux.HandleFunc(secretConsumersPath, func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"name":"canonical","NAME":false,"status":"ACTIVE","STATUS":{},"secret_ref":"foreign literal","SECRET_REF":[],"consumers":[{"service":"image","SERVICE":{},"resource_type":"images","RESOURCE_TYPE":false,"resource_id":"raw-id","RESOURCE_ID":[]}],"CONSUMERS":false}`)
	})
	value, err := secretConsumersScope(t, secretConsumersClient(aliasCloud)).Create(context.Background(), input)
	if err != nil || value.Name != "canonical" || value.SecretRef != "foreign literal" || value.Consumers[0].Service != "image" || value.Consumers[0].ResourceID != "raw-id" || string(value.Body["NAME"]) != "false" || string(value.Consumers[0].Body["SERVICE"]) != "{}" {
		t.Fatal(value, err)
	}
}

func TestKeyManagerSecretConsumersMutationSnapshotsAndMissingPolicy(t *testing.T) {
	cloud := testcloud.New(t)
	client := secretConsumersClient(cloud)
	scope := secretConsumersScope(t, client)
	input := secretconsumers.ConsumerOpts{Service: "image", ResourceType: "image", ResourceID: "caller"}
	var mode, calls atomic.Int32
	cloud.Mux.HandleFunc(secretConsumersPath, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if mode.Load() == 0 {
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"vendor":{"n":9007199254740993}`) || r.Header.Get("X-Call") != "owned" {
				t.Error(string(body), r.Header)
			}
			testcloud.JSON(w, 200, secretConsumerMutation)
			return
		}
		w.Header().Set("X-Request-ID", "failure")
		testcloud.JSON(w, int(mode.Load()), `{"failure":"original"}`)
	})
	fields := map[string]any{"n": json.Number("9007199254740993")}
	option := secretconsumers.WithCreateField("vendor", fields)
	fields["n"] = 0
	var retained *request.Config[secretconsumers.ConsumerOpts]
	custom := func(config *request.Config[secretconsumers.ConsumerOpts]) error {
		retained = config
		config.Headers["X-Call"] = "owned"
		return nil
	}
	original := client.HTTPClient.Transport
	client.HTTPClient.Transport = secretConsumerRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		retained.Headers["X-Call"] = "mutated"
		retained.Fields["vendor"][0] = 'x'
		return original.RoundTrip(r)
	})
	if _, err := scope.Create(context.Background(), input, option, custom); err != nil {
		t.Fatal(err)
	}
	client.HTTPClient.Transport = original
	mode.Store(404)
	if value, err := scope.Delete(context.Background(), input); value != nil || err != nil {
		t.Fatal(value, err)
	}
	ignore := false
	strict := secretconsumers.WithDeleteOptions(secretconsumers.DeleteOpts{ConsumerOpts: input, IgnoreMissing: &ignore})
	ignore = true
	if _, err := scope.Delete(context.Background(), input, strict); !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatal(err)
	}
	for _, code := range []int{403, 201, 204} {
		mode.Store(int32(code))
		_, err := scope.Delete(context.Background(), input)
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != code || native.ResponseHeader.Get("X-Request-ID") != "failure" {
			t.Fatal(code, err)
		}
	}
	mode.Store(201)
	if _, err := scope.Create(context.Background(), input); !gophercloud.ResponseCodeIs(err, 201) {
		t.Fatal(err)
	}
	transportFailure := &gophercloud.ErrUnexpectedResponseCode{Actual: 404}
	client.HTTPClient.Transport = secretConsumerRoundTripFunc(func(r *http.Request) (*http.Response, error) { return nil, transportFailure })
	if _, err := scope.Delete(context.Background(), input); !errors.Is(err, transportFailure) {
		t.Fatal("transport404 was ignored", err)
	} else {
		var transport *url.Error
		if !errors.As(err, &transport) {
			t.Fatal("transport cause lost", err)
		}
	}
	client.HTTPClient.Transport = original
	if calls.Load() != 7 {
		t.Fatal(calls.Load())
	}
	for _, status := range []int{404, 200} {
		t.Run(fmt.Sprintf("canceled-after-native-status-%d", status), func(t *testing.T) {
			observed := testcloud.New(t)
			client := secretConsumersClient(observed)
			scope := secretConsumersScope(t, client)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var requests atomic.Int32
			body := secretConsumerMutation
			client.HTTPClient.Transport = secretConsumerRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				requests.Add(1)
				cancel()
				return &http.Response{StatusCode: status, Header: http.Header{"X-Request-Id": {"canceled-status"}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			})
			value, err := scope.Delete(ctx, input)
			if value != nil || !errors.Is(err, context.Canceled) || requests.Load() != 1 {
				t.Fatal("canceled response was suppressed or resent", value, err, requests.Load())
			}
			if status == 404 {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-ID") != "canceled-status" {
					t.Fatal("native404 evidence lost", err)
				}
			} else {
				var accepted *resource.ResponseError
				if !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != body || accepted.Header.Get("X-Request-ID") != "canceled-status" {
					t.Fatal("accepted200 cancellation evidence lost", err)
				}
			}
		})
	}
	for _, body := range []string{`null`, `[]`, `{`, `{"consumers":{}}`, `{"consumers":[null]}`, `{"consumers":[{"service":false}]}`} {
		t.Run(body, func(t *testing.T) {
			bad := testcloud.New(t)
			var count atomic.Int32
			bad.Mux.HandleFunc(secretConsumersPath, func(w http.ResponseWriter, r *http.Request) {
				count.Add(1)
				w.Header().Set("X-Request-ID", "accepted")
				testcloud.JSON(w, 200, body)
			})
			value, err := secretConsumersScope(t, secretConsumersClient(bad)).Create(context.Background(), input)
			var accepted *resource.ResponseError
			if value != nil || !errors.As(err, &accepted) || string(accepted.Body) != body || accepted.StatusCode != 200 || accepted.Header.Get("X-Request-ID") != "accepted" || count.Load() != 1 {
				t.Fatal(value, err, count.Load())
			}
		})
	}
}

func TestKeyManagerSecretConsumersNameScopeAndFixedIdentity(t *testing.T) {
	for _, kind := range []string{"success", "duplicate", "changed-source"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := secretConsumersClient(cloud)
			var lists, posts atomic.Int32
			path := "/reverse/barbican/v1/secrets"
			cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("name") != "exact name" {
					t.Error(r.URL)
				}
				if r.URL.Query().Get("offset") == "" {
					testcloud.JSON(w, 200, `{"secrets":[{"name":"decoy","secret_ref":"`+cloud.Server.URL+`/v1/secrets/decoy"}],"next":"`+cloud.Server.URL+path+`?name=exact+name&offset=1"}`)
					return
				}
				rows := `{"name":"exact name","secret_ref":"https://foreign.invalid/secrets/secret-alpha"}`
				if kind == "duplicate" {
					rows += `,{"name":"exact name","secret_ref":"https://foreign.invalid/secrets/other"}`
				}
				testcloud.JSON(w, 200, `{"secrets":[`+rows+`]}`)
			})
			cloud.Mux.HandleFunc("POST "+secretConsumersPath, func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				testcloud.JSON(w, 200, `{"secret_ref":"https://foreign.invalid/retargeted","consumers":[]}`)
			})
			if kind == "changed-source" {
				original := client.HTTPClient.Transport
				client.HTTPClient.Transport = secretConsumerRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					response, err := original.RoundTrip(r)
					if lists.Load() == 2 {
						client.Type = "compute"
					}
					return response, err
				})
			}
			scope, err := secretconsumers.New(client).InSecret(context.Background(), resource.Name("exact name"))
			if kind == "duplicate" {
				if !errors.Is(err, resource.ErrAmbiguous) || scope != nil || posts.Load() != 0 {
					t.Fatal(scope, err)
				}
				return
			}
			if kind == "changed-source" {
				if !errors.Is(err, resource.ErrUnsupported) || scope != nil || posts.Load() != 0 {
					t.Fatal(scope, err)
				}
				return
			}
			if err != nil || scope.SecretID() != "secret-alpha" || lists.Load() != 2 {
				t.Fatal(scope, err, lists.Load())
			}
			for i := 0; i < 2; i++ {
				if _, err := scope.Create(context.Background(), secretconsumers.ConsumerOpts{Service: "image", ResourceType: "image", ResourceID: "id"}); err != nil {
					t.Fatal(err)
				}
			}
			if lists.Load() != 2 || posts.Load() != 2 {
				t.Fatal(lists.Load(), posts.Load())
			}
		})
	}
	t.Run("unicode-parent-escaped-once", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.URL.EscapedPath() != "/reverse/barbican/v1/secrets/%EB%B9%84%EB%B0%80/consumers" {
				t.Error(r.URL.EscapedPath())
			}
			testcloud.JSON(w, 200, `{"consumers":[]}`)
		})
		scope, err := secretconsumers.New(secretConsumersClient(cloud)).InSecret(context.Background(), resource.ID("비밀"))
		if err != nil || calls.Load() != 0 {
			t.Fatal(scope, err)
		}
		if _, err := scope.All(context.Background()); err != nil || calls.Load() != 1 {
			t.Fatal(err, calls.Load())
		}
	})
}

func TestKeyManagerSecretConsumersOffsetPagingAndQueryOwnership(t *testing.T) {
	cloud := testcloud.New(t)
	client := secretConsumersClient(cloud)
	scope := secretConsumersScope(t, client)
	var calls atomic.Int32
	var retained *request.Config[secretconsumers.ListOpts]
	cloud.Mux.HandleFunc("GET "+secretConsumersPath, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		query := r.URL.Query()
		if !reflect.DeepEqual(query["vendor"], []string{"first", "second"}) || query.Get("empty") != "" || !query.Has("empty") || query.Has("nil") || query.Has("marker") || query.Get("limit") != "2" || query.Get("service") != "wire-only" {
			t.Error(query)
		}
		retained.Query["vendor"][0] = "retained mutation"
		retained.Options.Limit = 99
		if query.Get("offset") == "1" {
			testcloud.JSON(w, 200, secretConsumersPage(secretConsumerRow, cloud.Server.URL+secretConsumersPath+"?offset=3&limit=2"))
		} else {
			if query.Get("offset") != "3" {
				t.Error(query)
			}
			testcloud.JSON(w, 200, secretConsumersPage(secretConsumerRow, ""))
		}
	})
	custom := func(config *request.Config[secretconsumers.ListOpts]) error {
		retained = config
		config.Query["vendor"] = []string{"first", "second"}
		config.Query["nil"] = nil
		return nil
	}
	options := []secretconsumers.ListOption{secretconsumers.WithListOptions(secretconsumers.ListOpts{Limit: 2, Offset: 1}), secretconsumers.WithListQuery("service", "wire-only"), secretconsumers.WithListQuery("empty", ""), custom}
	sequence := scope.List(context.Background(), options...)
	options[0] = nil
	for run := 0; run < 2; run++ {
		rows := 0
		for value, err := range sequence {
			if err != nil || value == nil || value.Service != "image" || string(value.Body["vendor"]) != `{"integer":9007199254740993}` {
				t.Fatal(value, err)
			}
			rows++
		}
		if rows != 2 {
			t.Fatal(rows)
		}
	}
	if calls.Load() != 4 {
		t.Fatal(calls.Load())
	}
	t.Run("default-offset-server-limit-and-no-fallback", func(t *testing.T) {
		fresh := testcloud.New(t)
		var count atomic.Int32
		fresh.Mux.HandleFunc(secretConsumersPath, func(w http.ResponseWriter, r *http.Request) {
			count.Add(1)
			q := r.URL.Query()
			if q.Has("marker") {
				t.Error(q)
			}
			if count.Load() == 1 {
				if q.Has("offset") || q.Has("limit") {
					t.Error(q)
				}
				testcloud.JSON(w, 200, secretConsumersPage(secretConsumerRow, fresh.Server.URL+secretConsumersPath+"?offset=10&limit=10"))
				return
			}
			if q.Get("offset") != "10" || q.Get("limit") != "10" {
				t.Error(q)
			}
			testcloud.JSON(w, 200, secretConsumersPage(secretConsumerRow, ""))
		})
		values, err := secretConsumersScope(t, secretConsumersClient(fresh)).All(context.Background())
		if err != nil || len(values) != 2 || count.Load() != 2 {
			t.Fatal(values, err, count.Load())
		}
	})
	t.Run("library-option-concurrent-reuse", func(t *testing.T) {
		fresh := testcloud.New(t)
		fresh.Mux.HandleFunc(secretConsumersPath, func(w http.ResponseWriter, r *http.Request) {
			testcloud.JSON(w, 200, secretConsumersPage(secretConsumerRow, "https://foreign.invalid/never"))
		})
		selected := false
		option := secretconsumers.WithListOptions(secretconsumers.ListOpts{Paginated: &selected})
		selected = true
		fixed := secretConsumersScope(t, secretConsumersClient(fresh))
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				values, err := fixed.All(context.Background(), option)
				if err != nil || len(values) != 1 {
					t.Error(values, err)
				}
			}()
		}
		wg.Wait()
	})
}

func TestKeyManagerSecretConsumersListControlsAndAcceptedRows(t *testing.T) {
	for _, mode := range []string{"cap", "single", "break", "empty", "uncapped", "malformed-json"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			body := secretConsumersPage(secretConsumerRow+`,{"service":false}`, "https://foreign.invalid/never")
			if mode == "empty" {
				body = secretConsumersPage("", "https://foreign.invalid/never")
			}
			if mode == "single" || mode == "break" {
				body = secretConsumersPage(secretConsumerRow, "https://foreign.invalid/never")
			}
			if mode == "malformed-json" {
				body = `{"consumers":[`
			}
			cloud.Mux.HandleFunc(secretConsumersPath, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" {
					t.Error("local control leaked", r.URL)
				}
				w.Header().Set("X-Page", "owned")
				testcloud.JSON(w, 200, body)
			})
			scope := secretConsumersScope(t, secretConsumersClient(cloud))
			if mode == "break" {
				for value, err := range scope.List(context.Background()) {
					if err != nil || value == nil {
						t.Fatal(value, err)
					}
					break
				}
				if calls.Load() != 1 {
					t.Fatal(calls.Load())
				}
				return
			}
			var options []secretconsumers.ListOption
			if mode == "cap" || mode == "malformed-json" {
				options = append(options, secretconsumers.WithListMaxItems(1))
			}
			if mode == "single" {
				options = append(options, secretconsumers.WithListPaginated(false))
			}
			values, err := scope.All(context.Background(), options...)
			if mode == "uncapped" || mode == "malformed-json" {
				var accepted *resource.ResponseError
				if values != nil || !errors.As(err, &accepted) || string(accepted.Body) != body || accepted.StatusCode != 200 || accepted.Header.Get("X-Page") != "owned" {
					t.Fatal(values, err)
				}
			} else if err != nil || (mode == "empty" && len(values) != 0) || (mode != "empty" && len(values) != 1) {
				t.Fatal(values, err)
			}
			if calls.Load() != 1 {
				t.Fatal(calls.Load())
			}
		})
	}
}

func TestKeyManagerSecretConsumersContinuationGuardsAndTerminalFailures(t *testing.T) {
	for _, mode := range []string{"same-offset", "backward", "foreign", "wrong-parent", "changed-filter", "late404", "late-malformed", "source-recheck", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := secretConsumersClient(cloud)
			var calls atomic.Int32
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cloud.Mux.HandleFunc(secretConsumersPath, func(w http.ResponseWriter, r *http.Request) {
				count := calls.Add(1)
				if count == 2 {
					if mode == "late404" {
						testcloud.JSON(w, 404, `{"late":true}`)
					} else {
						testcloud.JSON(w, 200, `{"consumers":[{"resource_id":false}]}`)
					}
					return
				}
				next := cloud.Server.URL + secretConsumersPath + "?offset=3&vendor=fixed"
				switch mode {
				case "same-offset":
					next = cloud.Server.URL + secretConsumersPath + "?offset=2"
				case "backward":
					next = cloud.Server.URL + secretConsumersPath + "?offset=1"
				case "foreign":
					next = "https://foreign.invalid/consumers?offset=3"
				case "wrong-parent":
					next = cloud.Server.URL + strings.Replace(secretConsumersPath, "secret-alpha", "other", 1) + "?offset=3"
				case "changed-filter":
					next = cloud.Server.URL + secretConsumersPath + "?offset=3&vendor=changed"
				}
				testcloud.JSON(w, 200, secretConsumersPage(secretConsumerRow, next))
			})
			if mode == "source-recheck" {
				original := client.HTTPClient.Transport
				client.HTTPClient.Transport = secretConsumerRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					response, err := original.RoundTrip(r)
					client.Type = "compute"
					return response, err
				})
			}
			scope := secretConsumersScope(t, client)
			options := []secretconsumers.ListOption{secretconsumers.WithListOptions(secretconsumers.ListOpts{Offset: 2}), secretconsumers.WithListQuery("vendor", "fixed")}
			if mode == "cancel" {
				var terminal error
				for value, err := range scope.List(ctx, options...) {
					if err != nil {
						terminal = err
					} else if value != nil {
						cancel()
					}
				}
				if !errors.Is(terminal, context.Canceled) || calls.Load() != 1 {
					t.Fatal(terminal, calls.Load())
				}
				return
			}
			values, err := scope.All(ctx, options...)
			if values != nil || err == nil {
				t.Fatal(values, err)
			}
			switch mode {
			case "same-offset":
				if !errors.Is(err, resource.ErrPaginationCycle) {
					t.Fatal(err)
				}
			case "source-recheck":
				if !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal(err)
				}
			case "late404":
				if !gophercloud.ResponseCodeIs(err, 404) {
					t.Fatal(err)
				}
			case "late-malformed":
				var accepted *resource.ResponseError
				if !errors.As(err, &accepted) || !strings.Contains(string(accepted.Body), "resource_id") {
					t.Fatal(err)
				}
			default:
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			}
			want := int32(1)
			if mode == "late404" || mode == "late-malformed" {
				want = 2
			}
			if calls.Load() != want {
				t.Fatal(calls.Load(), want)
			}
		})
	}
}

func TestKeyManagerSecretConsumersLiveSourceAndLazyPreflight(t *testing.T) {
	for _, location := range []string{"source", "custom-config"} {
		for _, conflict := range []bool{false, true} {
			t.Run(fmt.Sprintf("header-case-%s-conflict-%t", location, conflict), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := secretConsumersClient(cloud)
				scope := secretConsumersScope(t, client)
				var calls atomic.Int32
				cloud.Mux.HandleFunc(secretConsumersPath, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Header.Get("X-Trace") != "first" {
						t.Error(r.Header)
					}
					testcloud.JSON(w, 200, secretConsumerMutation)
				})
				headers := map[string]string{"X-Trace": "first", "x-trace": "first"}
				if conflict {
					headers["x-trace"] = "second"
				}
				var create []secretconsumers.CreateOption
				var remove []secretconsumers.DeleteOption
				if location == "source" {
					client.MoreHeaders = headers
				} else {
					create = append(create, func(config *request.Config[secretconsumers.ConsumerOpts]) error { config.Headers = headers; return nil })
					remove = append(remove, func(config *request.Config[secretconsumers.DeleteOpts]) error { config.Headers = headers; return nil })
				}
				input := secretconsumers.ConsumerOpts{Service: "image", ResourceType: "image", ResourceID: "id"}
				created, createErr := scope.Create(context.Background(), input, create...)
				deleted, deleteErr := scope.Delete(context.Background(), input, remove...)
				if conflict {
					if created != nil || deleted != nil || !errors.Is(createErr, resource.ErrInvalidOption) || !errors.Is(deleteErr, resource.ErrInvalidOption) || calls.Load() != 0 {
						t.Fatal(created, deleted, createErr, deleteErr, calls.Load())
					}
					if location == "source" {
						if rows, err := scope.All(context.Background()); rows != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
							t.Fatal(rows, err, calls.Load())
						}
					}
				} else if createErr != nil || deleteErr != nil || created == nil || deleted == nil || created.StatusCode != 200 || deleted.StatusCode != 200 || calls.Load() != 2 {
					t.Fatal(created, deleted, createErr, deleteErr, calls.Load())
				}
				if headers["X-Trace"] != "first" || (conflict && headers["x-trace"] != "second") || (!conflict && headers["x-trace"] != "first") {
					t.Fatal("caller header map changed", headers)
				}
			})
		}
	}
	cloud := testcloud.New(t)
	client := secretConsumersClient(cloud)
	client.MoreHeaders = map[string]string{"X-Project-Id": "unchanged"}
	var calls atomic.Int32
	cloud.Mux.HandleFunc(secretConsumersPath, func(w http.ResponseWriter, r *http.Request) {
		count := calls.Add(1)
		if r.Header.Get("X-Middleware") != "original" || r.Header.Get("X-Project-Id") != "unchanged" {
			t.Error(r.Header)
		}
		if count == 1 {
			if r.Header.Get("X-Auth-Token") != "test-token" {
				t.Error(r.Header)
			}
			cloud.Provider.SetToken("rotated")
			testcloud.JSON(w, 200, secretConsumersPage(secretConsumerRow, cloud.Server.URL+secretConsumersPath+"?offset=1"))
		} else {
			if r.Header.Get("X-Auth-Token") != "rotated" {
				t.Error(r.Header)
			}
			testcloud.JSON(w, 200, secretConsumersPage(secretConsumerRow, ""))
		}
	})
	original := client.HTTPClient.Transport
	client.HTTPClient.Transport = secretConsumerRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		r.Header.Set("X-Middleware", "original")
		return original.RoundTrip(r)
	})
	scope := secretConsumersScope(t, client)
	if values, err := scope.All(context.Background()); err != nil || len(values) != 2 || calls.Load() != 2 || scope.RawClient() != client || scope.RawClient().ProviderClient != cloud.Provider {
		t.Fatal(values, err, calls.Load())
	}
	invalid := [][]secretconsumers.ListOption{
		{nil}, {secretconsumers.WithListOptions(secretconsumers.ListOpts{Limit: -1})}, {secretconsumers.WithListOptions(secretconsumers.ListOpts{Offset: -1})}, {secretconsumers.WithListMaxItems(-1)},
		{secretconsumers.WithListQuery("offset", "-1")}, {secretconsumers.WithListQuery("offset", "0"), func(c *request.Config[secretconsumers.ListOpts]) error {
			c.Query["offset"] = []string{"0", "1"}
			return nil
		}},
		{secretconsumers.WithListQuery("marker", "invented")}, {secretconsumers.WithListQuery("limit", "0")}, {secretconsumers.WithListQuery("max_items", "1")},
		{secretconsumers.WithListOptions(secretconsumers.ListOpts{Limit: 2}), secretconsumers.WithListQuery("limit", "2")},
		{secretconsumers.WithListOptions(secretconsumers.ListOpts{Offset: 1}), secretconsumers.WithListQuery("offset", "1")},
		{func(c *request.Config[secretconsumers.ListOpts]) error {
			c.Headers["X-Unknown"] = "unsupported"
			return nil
		}},
	}
	for i, options := range invalid {
		sequence := scope.List(context.Background(), options...)
		if calls.Load() != 2 {
			t.Fatal("not lazy", i)
		}
		emitted := 0
		for value, err := range sequence {
			emitted++
			if value != nil || err == nil {
				t.Fatal(i, value, err)
			}
		}
		if emitted != 1 || calls.Load() != 2 {
			t.Fatal(i, emitted, calls.Load())
		}
	}
	input := secretconsumers.ConsumerOpts{Service: "image", ResourceType: "image", ResourceID: "id"}
	for _, option := range []secretconsumers.CreateOption{nil, secretconsumers.WithCreateField("SERVICE", "shadow"), secretconsumers.WithCreateField("secret_id", "other"), secretconsumers.WithCreateHeader("X-Auth-Token", "foreign"), secretconsumers.WithCreateField("vendor", json.RawMessage(`{`)), func(c *request.Config[secretconsumers.ConsumerOpts]) error {
		c.Query.Set("vendor", "unsupported")
		return nil
	}} {
		if _, err := scope.Create(context.Background(), input, option); err == nil || calls.Load() != 2 {
			t.Fatal(err, calls.Load())
		}
	}
	if _, err := scope.Create(context.Background(), secretconsumers.ConsumerOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := scope.All(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := scope.Create(ctx, input); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := scope.All(nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	var missing *secretconsumers.SecretScope
	if _, err := missing.Delete(context.Background(), input); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, bad := range []*gophercloud.ServiceClient{nil, {Type: "key-manager"}, {ProviderClient: cloud.Provider, Type: "compute"}} {
		if _, err := secretconsumers.New(bad).InSecret(context.Background(), resource.ID("id")); err == nil {
			t.Fatal(bad)
		}
	}
	for _, id := range []string{"", "../other", "a%2Fb", "a?b", "a b"} {
		if _, err := secretconsumers.New(client).InSecret(context.Background(), resource.ID(id)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(id, err)
		}
	}
	if _, err := scope.All(context.Background(), func(c *request.Config[secretconsumers.ListOpts]) error { client.Type = "compute"; return nil }); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	client.Type = "key-manager"
	if calls.Load() != 2 || client.MoreHeaders["X-Project-Id"] != "unchanged" {
		t.Fatal(calls.Load(), client.MoreHeaders)
	}
	readFailure := fmt.Errorf("original read failure")
	client.HTTPClient.Transport = secretConsumerRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"X-Request-Id": {"read-failure"}}, Body: &secretConsumerFailingRead{cause: readFailure}, Request: r}, nil
	})
	_, err := scope.Delete(context.Background(), input)
	var accepted *resource.ResponseError
	if !errors.Is(err, readFailure) || !errors.As(err, &accepted) || accepted.StatusCode != 200 || accepted.Header.Get("X-Request-ID") != "read-failure" {
		t.Fatal(err)
	}
}

type secretConsumerFailingRead struct{ cause error }

func (r *secretConsumerFailingRead) Read([]byte) (int, error) { return 0, r.cause }
func (r *secretConsumerFailingRead) Close() error             { return nil }

type secretConsumerRoundTripFunc func(*http.Request) (*http.Response, error)

func (f secretConsumerRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
