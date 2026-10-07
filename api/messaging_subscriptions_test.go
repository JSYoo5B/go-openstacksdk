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

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/messaging/v2/subscriptions"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const subscriptionsPrefix = "/reverse/zaqar/v2"
const subscriptionsClientID = "11111111-1111-4111-8111-111111111111"

func subscriptionClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	c := cloud.Client("message", "/catalog/v2")
	c.ResourceBase = cloud.Server.URL + subscriptionsPrefix + "/"
	c.MoreHeaders = map[string]string{"Client-ID": subscriptionsClientID, "X-PROJECT-ID": "configured-project", "X-Configured": "kept"}
	return c
}
func subscriptionScope(t *testing.T, c *gophercloud.ServiceClient, queue string) *subscriptions.QueueScope {
	t.Helper()
	scope, err := subscriptions.New(c).InQueue(context.Background(), queue)
	if err != nil {
		t.Fatal(err)
	}
	return scope
}
func TestMessagingSubscriptionsFourRoutesFixedQueueAndRawModels(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	client := subscriptionClient(cloud)
	route := subscriptionsPrefix + "/queues/queue+α/subscriptions"
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if r.Header.Get("Client-ID") != subscriptionsClientID || r.Header.Get("X-PROJECT-ID") != "configured-project" || r.Header.Get("X-Configured") != "kept" || r.Header.Get("X-Auth-Token") != fmt.Sprintf("token-%d", n) {
			t.Error(r.Header)
		}
		w.Header().Set("X-Request-ID", "response")
		switch r.Method + " " + r.URL.Path {
		case "POST " + route:
			raw, _ := io.ReadAll(r.Body)
			if string(raw) != `{"subscriber":"mailto:example@example.test"}` {
				t.Error(string(raw))
			}
			w.Header().Set("Location", cloud.Server.URL+route)
			testcloud.JSON(w, 201, `{"subscription_id":"sub+α","SUBSCRIPTION_ID":[1]}`)
		case "GET " + route + "/sub+α":
			testcloud.JSON(w, 200, `{"id":null,"ID":"decoy","subscription_id":"alternate","SUBSCRIPTION_ID":{},"subscriber":null,"SUBSCRIBER":false,"source":"other-queue","ttl":9007199254740993,"TTL":[],"age":null,"options":{"flag":false,"large":9007199254740993,"nested":null}}`)
		case "GET " + route:
			testcloud.JSON(w, 200, `{"subscriptions":[]}`)
		case "DELETE " + route + "/sub+α":
			w.WriteHeader(204)
		default:
			t.Error("queue lookup/ref follow/changed scope", r.Method, r.URL)
			w.WriteHeader(500)
		}
	})
	api := subscriptions.New(client)
	scope := subscriptionScope(t, client, "queue+α")
	if api.RawClient() != client || scope.RawClient() != client || scope.QueueName() != "queue+α" || calls.Load() != 0 {
		t.Fatal("scope construction changed source or performed HTTP")
	}
	cloud.Provider.SetToken("token-1")
	created, err := scope.Create(context.Background(), subscriptions.CreateOpts{Subscriber: "mailto:example@example.test"})
	if err != nil || created.ID != "sub+α" || created.SubscriptionID != "sub+α" || created.StatusCode != 201 || created.Header.Get("Location") != cloud.Server.URL+route {
		t.Fatal(created, err)
	}
	if _, present := created.Body["ttl"]; present {
		t.Fatal("server default fabricated")
	}
	cloud.Provider.SetToken("token-2")
	got, err := scope.Get(context.Background(), "sub+α")
	if err != nil || got.ID != "" || got.SubscriptionID != "alternate" || got.Source != "other-queue" || string(got.TTL) != "9007199254740993" || string(got.Age) != "null" || string(got.Body["ID"]) != `"decoy"` || got.Subscriber != "" {
		t.Fatal(got, err)
	}
	got.TTL[0] = '1'
	if string(got.Body["ttl"]) != "9007199254740993" {
		t.Fatal("typed raw field aliases Body")
	}
	cloud.Provider.SetToken("token-3")
	values, err := scope.All(context.Background())
	if err != nil || len(values) != 0 {
		t.Fatal(values, err)
	}
	cloud.Provider.SetToken("token-4")
	if err := scope.Delete(context.Background(), "sub+α"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 4 || scope.QueueName() != "queue+α" || client.ResourceBase != cloud.Server.URL+subscriptionsPrefix+"/" || client.MoreHeaders["Client-ID"] != subscriptionsClientID {
		t.Fatal("unexpected source mutation or HTTP", calls.Load())
	}
}
func TestMessagingSubscriptionsCreateSnapshotsExactMetadataAndIdentity(t *testing.T) {
	cloud := testcloud.New(t)
	c := subscriptionClient(cloud)
	scope := subscriptionScope(t, c, "queue")
	var calls atomic.Int32
	expected := `{"large":9007199254740993,"false":false,"null":null}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "POST" || r.Header.Get("Client-ID") != "typed-client" || r.Header.Get("X-PROJECT-ID") != "typed-project" || r.Header.Get("X-Trace") != "last" {
			t.Error(r.Method, r.Header)
		}
		var fields map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&fields); err != nil {
			t.Error(err)
		}
		if string(fields["ttl"]) != "9007199254740993" || string(fields["options"]) != expected || string(fields["vendor"]) != `{"n":9007199254740993}` {
			t.Error(fields)
		}
		if _, present := fields["RequestIdentity"]; present {
			t.Error("identity leaked into body")
		}
		if _, present := fields["ClientID"]; present {
			t.Error("client ID leaked into body")
		}
		testcloud.JSON(w, 201, `{"subscription_id":"created"}`)
	})
	opts := subscriptions.CreateOpts{Subscriber: "https://subscriber.invalid", Options: json.RawMessage(expected), RequestIdentity: subscriptions.RequestIdentity{ClientID: "typed-client", ProjectID: "typed-project"}}
	snapshot := subscriptions.WithCreateOptions(opts)
	opts.Options[0] = '['
	opts.ClientID = "mutated"
	extras := map[string]json.RawMessage{"n": json.RawMessage("9007199254740993")}
	field := subscriptions.WithCreateField("vendor", extras)
	extras["n"][0] = '1'
	reusable := []subscriptions.CreateOption{snapshot, subscriptions.WithCreateTTL(60), subscriptions.WithCreateTTL(9007199254740993), field, subscriptions.WithCreateHeader("x-trace", "first"), subscriptions.WithCreateHeader("X-Trace", "last")}
	var group sync.WaitGroup
	for range 4 {
		group.Add(1)
		go func() {
			defer group.Done()
			v, err := scope.Create(context.Background(), subscriptions.CreateOpts{}, reusable...)
			if err != nil || v.ID != "created" {
				t.Error(v, err)
			}
		}()
	}
	group.Wait()
	if calls.Load() != 4 || c.MoreHeaders["X-PROJECT-ID"] != "configured-project" || c.MoreHeaders["Client-ID"] != subscriptionsClientID {
		t.Fatal(c.MoreHeaders, calls.Load())
	}
}
func TestMessagingSubscriptionsPythonMarkerPagingIgnoresLinksAndKeepsSource(t *testing.T) {
	for _, mode := range []string{"unlimited", "short-initial", "empty", "explicit-full"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			c := subscriptionClient(cloud)
			scope := subscriptionScope(t, c, "queue")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				page := calls.Add(1)
				query := r.URL.Query()
				if r.URL.Path != subscriptionsPrefix+"/queues/queue/subscriptions" || !reflect.DeepEqual(query["tag"], []string{"first", "second"}) || r.Header.Get("X-Trace") != "frozen" || r.Header.Get("Client-ID") != subscriptionsClientID {
					t.Error(r.URL, r.Header)
				}
				wantToken := "test-token"
				if page > 1 {
					wantToken = "rotated"
				}
				if r.Header.Get("X-Auth-Token") != wantToken {
					t.Error(r.Header)
				}
				rows := `{"id":"a"},{"subscription_id":"b"}`
				switch mode {
				case "unlimited":
					if page == 1 {
						if query.Has("limit") || query.Has("marker") {
							t.Error(query)
						}
					} else {
						if query.Get("limit") != "2" || query.Get("marker") != "b" {
							t.Error(query)
						}
						rows = `{"id":"c"}`
					}
				case "short-initial":
					if query.Get("limit") != "3" {
						t.Error(query)
					}
				case "empty":
					rows = ""
				case "explicit-full":
					if query.Get("limit") != "1" {
						t.Error(query)
					}
					rows = `{"id":"a"}`
					if page == 2 {
						rows = `{"id":"b"}`
						if query.Get("marker") != "a" {
							t.Error(query)
						}
					}
					if page == 3 {
						rows = ""
						if query.Get("marker") != "b" {
							t.Error(query)
						}
					}
				}
				testcloud.JSON(w, 200, `{"subscriptions":[`+rows+`],"links":[{"rel":"next","href":"https://never-follow.invalid/decoy"}],"next":"https://never-follow.invalid/decoy"}`)
			})
			original := c.HTTPClient.Transport
			c.HTTPClient.Transport = subscriptionRoundTrip(func(r *http.Request) (*http.Response, error) {
				res, err := original.RoundTrip(r)
				cloud.Provider.SetToken("rotated")
				return res, err
			})
			opts := []subscriptions.ListOption{subscriptions.WithListHeader("X-Trace", "frozen"), func(cfg *request.Config[subscriptions.ListOpts]) error {
				cfg.Query["tag"] = []string{"first", "second"}
				return nil
			}}
			if mode == "short-initial" {
				opts = append(opts, subscriptions.WithListOptions(subscriptions.ListOpts{Limit: 3}))
			}
			if mode == "explicit-full" {
				opts = append(opts, subscriptions.WithListOptions(subscriptions.ListOpts{Limit: 1}))
			}
			values, err := scope.All(context.Background(), opts...)
			wantRows, wantCalls := 3, int32(2)
			switch mode {
			case "short-initial":
				wantRows, wantCalls = 2, 1
			case "empty":
				wantRows, wantCalls = 0, 1
			case "explicit-full":
				wantRows, wantCalls = 2, 3
			}
			if err != nil || len(values) != wantRows || calls.Load() != wantCalls {
				t.Fatal(values, err, calls.Load())
			}
		})
	}
}
func TestMessagingSubscriptionsListControlsLazyDecodeAndReuse(t *testing.T) {
	for _, mode := range []string{"cap", "single", "break", "bad-consumed", "bad-json", "reuse"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			c := subscriptionClient(cloud)
			scope := subscriptionScope(t, c, "queue")
			var calls atomic.Int32
			body := `{"subscriptions":[{"id":"first"},{"subscriber":false}]}`
			if mode == "single" || mode == "break" || mode == "reuse" {
				body = `{"subscriptions":[{"id":"first"}]}`
			}
			if mode == "bad-json" {
				body = `{"subscriptions":[{"id":"first"},`
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Has("max_items") || r.URL.Query().Has("paginated") || r.URL.Query().Has("limit") {
					t.Error(r.URL)
				}
				w.Header().Set("X-Request-ID", "whole-page")
				testcloud.JSON(w, 200, body)
			})
			opts := []subscriptions.ListOption{}
			if mode == "cap" || mode == "bad-json" {
				opts = append(opts, subscriptions.WithListMaxItems(1))
			}
			if mode == "single" || mode == "reuse" {
				value := false
				opts = append(opts, subscriptions.WithListOptions(subscriptions.ListOpts{Paginated: &value}))
				value = true
			}
			seq := scope.List(context.Background(), opts...)
			opts = append(opts, subscriptions.WithListMaxItems(-1))
			if calls.Load() != 0 {
				t.Fatal("eager")
			}
			seen, errorCount := 0, 0
			for value, err := range seq {
				if mode == "bad-consumed" || mode == "bad-json" {
					if err != nil {
						errorCount++
						var evidence *resource.ResponseError
						if !errors.As(err, &evidence) || string(evidence.Body) != body || evidence.Header.Get("X-Request-ID") != "whole-page" {
							t.Fatal(err, evidence)
						}
						continue
					}
				} else if err != nil || value.ID != "first" {
					t.Fatal(value, err)
				}
				seen++
				if mode == "break" {
					break
				}
			}
			if mode == "bad-json" {
				if seen != 0 {
					t.Fatal(seen)
				}
			} else if seen != 1 {
				t.Fatal(seen)
			}
			if (mode == "bad-json" || mode == "bad-consumed") && errorCount != 1 {
				t.Fatal("accepted page error was swallowed", errorCount)
			}
			if mode == "reuse" {
				for _, err := range seq {
					if err != nil {
						t.Fatal(err)
					}
				}
				if calls.Load() != 2 {
					t.Fatal(calls.Load())
				}
			} else if calls.Load() != 1 {
				t.Fatal("control did not stop", calls.Load())
			}
		})
	}
}
func TestMessagingSubscriptionsInvalidOptionsAndMissingClientIdentity(t *testing.T) {
	cloud := testcloud.New(t)
	c := subscriptionClient(cloud)
	scope := subscriptionScope(t, c, "queue")
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	for _, option := range []subscriptions.CreateOption{
		func(cfg *request.Config[subscriptions.CreateOpts]) error {
			cfg.Headers["X-Trace"] = "first"
			cfg.Headers["x-trace"] = "second"
			return nil
		},
		nil, subscriptions.WithCreateSubscriber(""), subscriptions.WithCreateDeliveryOptions(nil), subscriptions.WithCreateDeliveryOptions([]string{}), subscriptions.WithCreateField("subscriber", "other"), subscriptions.WithCreateField("TTL", 3), subscriptions.WithCreateField("age", 2), subscriptions.WithCreateField("queue_name", "other"), subscriptions.WithCreateHeader("Client-ID", "other"), subscriptions.WithCreateHeader("X-PROJECT-ID", "other"), subscriptions.WithCreateHeader("X-Auth-Token", "other"), subscriptions.WithCreateHeader("bad?header", "x"), subscriptions.WithCreateHeader("X-Test", "bad\x00"), request.WithQuery[subscriptions.CreateOpts]("x", "y"), request.WithArgument[subscriptions.CreateOpts]("x", "y"),
	} {
		if _, err := scope.Create(context.Background(), subscriptions.CreateOpts{Subscriber: "mailto:example@example.test"}, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := scope.Create(context.Background(), subscriptions.CreateOpts{Subscriber: "http://example.invalid", TTL: request.Null[int64]()}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, option := range []subscriptions.ListOption{nil, subscriptions.WithListMaxItems(-1), subscriptions.WithListOptions(subscriptions.ListOpts{Limit: -1}), subscriptions.WithListQuery("limit", "0"), subscriptions.WithListQuery("marker", " "), subscriptions.WithListQuery("Queue_Name", "other"), subscriptions.WithListQuery("headers", "x"), subscriptions.WithListQuery("max_items", "1"), request.WithField[subscriptions.ListOpts]("x", 1), request.WithArgument[subscriptions.ListOpts]("x", 1)} {
		if values, err := scope.All(context.Background(), option); values != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(values, err)
		}
	}
	for _, bad := range []string{"", ".", "..", "has/slash", "percent%20", "space name", string([]byte{0xff})} {
		if _, err := subscriptions.New(c).InQueue(context.Background(), bad); err == nil {
			t.Fatal(bad)
		}
		if _, err := scope.Get(context.Background(), bad); err == nil {
			t.Fatal(bad)
		}
		if err := scope.Delete(context.Background(), bad); err == nil {
			t.Fatal(bad)
		}
	}
	for _, api := range []*subscriptions.API{nil, subscriptions.New(nil), subscriptions.New(&gophercloud.ServiceClient{}), subscriptions.New(cloud.Client("compute", "/v2"))} {
		if _, err := api.InQueue(context.Background(), "queue"); err == nil {
			t.Fatal("invalid source")
		}
	}
	var nilScope *subscriptions.QueueScope
	if _, err := nilScope.Get(context.Background(), "id"); err == nil {
		t.Fatal("nil scope")
	}
	if _, err := (&subscriptions.QueueScope{}).All(context.Background()); err == nil {
		t.Fatal("zero scope")
	}
	c.MoreHeaders = map[string]string{}
	if _, err := scope.Get(context.Background(), "id"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("missing client id", err)
	}
	if calls.Load() != 0 {
		t.Fatal("preflight sent HTTP", calls.Load())
	}
}
func TestMessagingSubscriptionsTypedRequestIdentityAllOperations(t *testing.T) {
	cloud := testcloud.New(t)
	c := subscriptionClient(cloud)
	c.MoreHeaders = map[string]string{}
	scope := subscriptionScope(t, c, "queue")
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Client-ID") != "explicit-client" || r.Header.Get("X-PROJECT-ID") != "explicit-project" {
			t.Error(r.Header)
		}
		switch r.Method {
		case "POST":
			testcloud.JSON(w, 201, `{"subscription_id":"id"}`)
		case "GET":
			if strings.HasSuffix(r.URL.Path, "/id") {
				testcloud.JSON(w, 200, `{"id":"id"}`)
			} else {
				testcloud.JSON(w, 200, `{"subscriptions":[]}`)
			}
		case "DELETE":
			w.WriteHeader(204)
		}
	})
	if _, err := scope.Create(context.Background(), subscriptions.CreateOpts{Subscriber: "http://example.invalid"}, subscriptions.WithCreateClientID("explicit-client"), subscriptions.WithCreateProjectID("explicit-project")); err != nil {
		t.Fatal(err)
	}
	if _, err := scope.Get(context.Background(), "id", subscriptions.WithGetClientID("explicit-client"), subscriptions.WithGetProjectID("explicit-project")); err != nil {
		t.Fatal(err)
	}
	if _, err := scope.All(context.Background(), subscriptions.WithListClientID("explicit-client"), subscriptions.WithListProjectID("explicit-project")); err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(context.Background(), "id", subscriptions.WithDeleteClientID("explicit-client"), subscriptions.WithDeleteProjectID("explicit-project")); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 4 || len(c.MoreHeaders) != 0 {
		t.Fatal("shared identity mutated", c.MoreHeaders, calls.Load())
	}
}
func TestMessagingSubscriptionsMutationAndReadEvidenceNeverResend(t *testing.T) {
	for _, tc := range []struct{ body, id string }{
		{`{}`, ""}, {`{"subscription_id":null}`, ""}, {`{"subscription_id":""}`, ""},
		{`{"id":"literal-only"}`, "literal-only"}, {`{"SUBSCRIPTION_ID":"alias"}`, ""},
		{`{"subscription_id":"bad/slash"}`, "bad/slash"},
	} {
		t.Run("optional-create-"+tc.body, func(t *testing.T) {
			cloud := testcloud.New(t)
			scope := subscriptionScope(t, subscriptionClient(cloud), "queue")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Location", "https://never-follow.invalid/collection")
				w.Header().Set("X-Request-ID", "optional-id")
				testcloud.JSON(w, 201, tc.body)
			})
			value, err := scope.Create(context.Background(), subscriptions.CreateOpts{Subscriber: "http://example.invalid"}, subscriptions.WithCreateTTL(3600), subscriptions.WithCreateDeliveryOptions(map[string]any{"example": true}))
			if err != nil || value == nil || value.ID != tc.id || value.StatusCode != 201 || value.Header.Get("Location") != "https://never-follow.invalid/collection" || value.Header.Get("X-Request-ID") != "optional-id" || value.Subscriber != "" || len(value.TTL) != 0 || len(value.Options) != 0 || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
			for _, key := range []string{"subscriber", "ttl", "options"} {
				if _, present := value.Body[key]; present {
					t.Fatal("request attributes seeded response", key)
				}
			}
			if value.ID == "" || strings.Contains(value.ID, "/") {
				if _, err := scope.Get(context.Background(), value.ID); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				if err := scope.Delete(context.Background(), value.ID); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				if calls.Load() != 1 {
					t.Fatal("unusable returned ID or Location followed", calls.Load())
				}
			}
		})
	}
	for _, body := range []string{`null`, `[]`, `{"id":9}`, `{"subscriber":false}`, `{"ttl":`} {
		t.Run("get-accepted-"+body, func(t *testing.T) {
			cloud := testcloud.New(t)
			scope := subscriptionScope(t, subscriptionClient(cloud), "queue")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Request-ID", "get-evidence")
				testcloud.JSON(w, 200, body)
			})
			value, err := scope.Get(context.Background(), "fixed-id")
			var evidence *resource.ResponseError
			if value != nil || !errors.As(err, &evidence) || evidence.StatusCode != 200 || string(evidence.Body) != body || evidence.Header.Get("X-Request-ID") != "get-evidence" || calls.Load() != 1 {
				t.Fatal(value, err, evidence, calls.Load())
			}
		})
	}
	for _, operation := range []string{"create", "get", "list"} {
		t.Run("accepted-read-"+operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := subscriptionClient(cloud)
			scope := subscriptionScope(t, client, "queue")
			cause := errors.New("accepted subscription read failed")
			var calls atomic.Int32
			code := 200
			if operation == "create" {
				code = 201
			}
			client.HTTPClient.Transport = subscriptionRoundTrip(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				return &http.Response{StatusCode: code, Header: http.Header{"X-Request-Id": {"read-evidence"}}, Body: io.NopCloser(io.MultiReader(strings.NewReader("prefix"), subscriptionErrorReader{cause})), Request: r}, nil
			})
			var err error
			switch operation {
			case "create":
				_, err = scope.Create(context.Background(), subscriptions.CreateOpts{Subscriber: "http://example.invalid"})
			case "get":
				_, err = scope.Get(context.Background(), "id")
			case "list":
				_, err = scope.All(context.Background())
			}
			var evidence *resource.ResponseError
			if !errors.Is(err, cause) || !errors.As(err, &evidence) || evidence.StatusCode != code || string(evidence.Body) != "prefix" || evidence.Header.Get("X-Request-ID") != "read-evidence" || calls.Load() != 1 {
				t.Fatal(err, evidence, calls.Load())
			}
		})
	}
	for _, op := range []string{"create", "get", "delete"} {
		for _, code := range []int{200, 201, 202, 204, 403, 404, 409} {
			expected := map[string]int{"create": 201, "get": 200, "delete": 204}[op]
			if code == expected {
				continue
			}
			t.Run(fmt.Sprintf("%s-%d", op, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				c := subscriptionClient(cloud)
				scope := subscriptionScope(t, c, "queue")
				var calls atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					testcloud.JSON(w, code, `{"error":"native"}`)
				})
				var err error
				switch op {
				case "create":
					_, err = scope.Create(context.Background(), subscriptions.CreateOpts{Subscriber: "http://example.invalid"})
				case "get":
					_, err = scope.Get(context.Background(), "id")
				case "delete":
					err = scope.Delete(context.Background(), "id", subscriptions.WithDeleteIgnoreMissing(false))
				}
				if !gophercloud.ResponseCodeIs(err, code) || calls.Load() != 1 {
					t.Fatal(err, calls.Load())
				}
			})
		}
	}
	for _, body := range []string{`{"subscription_id":9}`, `[]`, `null`, `{"subscription_id":`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			c := subscriptionClient(cloud)
			scope := subscriptionScope(t, c, "queue")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Location", "https://foreign.invalid/looks-like-id")
				w.Header().Set("X-Request-ID", "accepted")
				testcloud.JSON(w, 201, body)
			})
			v, err := scope.Create(context.Background(), subscriptions.CreateOpts{Subscriber: "http://example.invalid"})
			var evidence *resource.ResponseError
			if v != nil || !errors.As(err, &evidence) || evidence.StatusCode != 201 || string(evidence.Body) != body || evidence.Header.Get("X-Request-ID") != "accepted" || calls.Load() != 1 {
				t.Fatal(v, err, evidence, calls.Load())
			}
		})
	}
	cloud := testcloud.New(t)
	c := subscriptionClient(cloud)
	scope := subscriptionScope(t, c, "queue")
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 404, `{"error":"absent"}`)
	})
	if err := scope.Delete(context.Background(), "id"); err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(context.Background(), "id", subscriptions.WithDeleteIgnoreMissing(false)); !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatal(calls.Load())
	}
	for _, accepted := range []bool{false, true} {
		var sent atomic.Int32
		native := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{204}}
		c.HTTPClient.Transport = subscriptionRoundTrip(func(r *http.Request) (*http.Response, error) {
			sent.Add(1)
			if !accepted {
				return nil, native
			}
			return &http.Response{StatusCode: 204, Header: http.Header{"X-Request-Id": {"accepted"}}, Body: io.NopCloser(io.MultiReader(strings.NewReader("prefix"), subscriptionErrorReader{native})), Request: r}, nil
		})
		err := scope.Delete(context.Background(), "id")
		if err == nil || !gophercloud.ResponseCodeIs(err, 404) || sent.Load() != 1 {
			t.Fatal("wrapped404 ignored", err, sent.Load())
		}
		if accepted {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 204 || string(proof.Body) != "prefix" {
				t.Fatal(err, proof)
			}
		} else {
			var transport *url.Error
			if !errors.As(err, &transport) {
				t.Fatal(err)
			}
		}
	}
	t.Run("canceled-native404-is-not-ignored", func(t *testing.T) {
		client := subscriptionClient(testcloud.New(t))
		scope := subscriptionScope(t, client, "queue")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var sent atomic.Int32
		client.HTTPClient.Transport = subscriptionRoundTrip(func(r *http.Request) (*http.Response, error) {
			sent.Add(1)
			cancel()
			return &http.Response{StatusCode: 404, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("absent")), Request: r}, nil
		})
		if err := scope.Delete(ctx, "id"); !errors.Is(err, context.Canceled) || sent.Load() != 1 {
			t.Fatal(err, sent.Load())
		}
	})
}
func TestMessagingSubscriptionsListTerminalErrorsAndSourceCancellation(t *testing.T) {
	for _, mode := range []string{"cycle", "null-marker", "missing-envelope", "null-envelope", "bad-consumed", "late-http", "source", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			c := subscriptionClient(cloud)
			scope := subscriptionScope(t, c, "queue")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				page := calls.Add(1)
				body := `{"subscriptions":[{"id":"a"}]}`
				switch mode {
				case "null-marker":
					body = `{"subscriptions":[{"id":null,"subscription_id":"alternate"}]}`
				case "missing-envelope":
					body = `{}`
				case "null-envelope":
					body = `{"subscriptions":null}`
				case "bad-consumed":
					if page == 2 {
						body = `{"subscriptions":[{"subscriber":false}]}`
					}
				case "late-http":
					if page == 2 {
						testcloud.JSON(w, 403, `{"error":"denied"}`)
						return
					}
				case "cancel":
					if page == 2 {
						cancel()
					}
				}
				w.Header().Set("X-Request-ID", "page")
				testcloud.JSON(w, 200, body)
			})
			if mode == "source" {
				original := c.HTTPClient.Transport
				c.HTTPClient.Transport = subscriptionRoundTrip(func(r *http.Request) (*http.Response, error) {
					response, err := original.RoundTrip(r)
					c.Type = "compute"
					return response, err
				})
			}
			values, err := scope.All(ctx)
			if values != nil || err == nil {
				t.Fatal(values, err)
			}
			if mode == "cycle" && !errors.Is(err, resource.ErrPaginationCycle) {
				t.Fatal(err)
			}
			if mode == "late-http" && !gophercloud.ResponseCodeIs(err, 403) {
				t.Fatal(err)
			}
			if mode == "source" && !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal(err)
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			wantCalls := int32(2)
			if mode == "null-marker" || mode == "missing-envelope" || mode == "null-envelope" || mode == "source" {
				wantCalls = 1
			}
			if calls.Load() != wantCalls {
				t.Fatal(calls.Load(), wantCalls)
			}
		})
	}
	cloud := testcloud.New(t)
	c := subscriptionClient(cloud)
	scope := subscriptionScope(t, c, "queue")
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := scope.Create(ctx, subscriptions.CreateOpts{Subscriber: "http://example.invalid"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := scope.Get(ctx, "id"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := scope.Delete(ctx, "id"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := scope.All(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := scope.Get(nil, "id"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := scope.Create(context.Background(), subscriptions.CreateOpts{Subscriber: "http://example.invalid"}, func(cfg *request.Config[subscriptions.CreateOpts]) error { c.Type = "compute"; return nil }); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}

type subscriptionRoundTrip func(*http.Request) (*http.Response, error)

func (f subscriptionRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type subscriptionErrorReader struct{ err error }

func (r subscriptionErrorReader) Read([]byte) (int, error) { return 0, r.err }
