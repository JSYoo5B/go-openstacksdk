package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/keymanager/v1/containers"
	"github.com/JSYoo5B/gophercloudsdk/keymanager/v1/orders"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

// These tests cover the separate owned Fetch lanes. Native Get and
// its alias/list/wait contracts remain covered by their existing tests.
// Existing testcloud, leaf/cached-Connection fixtures, transport and body fault
// wrappers are reused; there is no new fixture framework or transport type.
func TestKeyManagerOwnedMetadataFetchFixedTargetsAndOwnedReceipts(t *testing.T) {
	for _, kind := range []string{"containers", "orders"} {
		for fixtureIndex, fixtureName := range []string{"leaf", "cached connection"} {
			t.Run(kind+"/"+fixtureName, func(t *testing.T) {
				c, foreign := testcloud.New(t), testcloud.New(t)
				var calls, followed atomic.Int32
				foreign.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { followed.Add(1); w.WriteHeader(500) })
				path := "/reverse/barbican/v1/" + kind + "/request-alpha"
				reference := foreign.Server.URL + "/" + kind + "/passive-id?query=ignored#fragment"
				body := fmt.Sprintf(`{"id":null,"name":3,"NAME":false,"status":{"state":"ERROR"},"type":"untyped","created":"not a timestamp","updated":null,"self":{"raw":true},"vendor":{"n":9007199254740993},"%s_ref":%q}`, strings.TrimSuffix(kind, "s"), reference)
				if kind == "containers" {
					body = body[:len(body)-1] + `,"secret_refs":["one",{"arbitrary":true},null],"consumers":{"not":"a native ConsumerRef"}}`
				} else {
					body = body[:len(body)-1] + fmt.Sprintf(`,"secret_ref":%q,"meta":{"extension":{"n":9007199254740993}},"sub_status":false}`, foreign.Server.URL+"/secrets/secret-passive")
				}
				c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					th.TestMethod(t, r, http.MethodGet)
					th.TestHeader(t, r, "X-Auth-Token", "owned-fetch-token")
					th.TestHeader(t, r, "X-Call", "operation")
					th.TestHeader(t, r, "X-Source", "source")
					th.TestHeader(t, r, "Accept", "application/json")
					th.TestHeader(t, r, "OpenStack-API-Version", "key-manager 1.0")
					if r.URL.Path != path || r.URL.RawQuery != "" {
						t.Error(r.Method, r.URL, r.Header)
					}
					if data, err := io.ReadAll(r.Body); err != nil || len(data) != 0 {
						t.Error(string(data), err)
					}
					w.Header().Set("X-Request-ID", "actual receipt")
					testcloud.JSON(w, 200, body)
				})
				var view, wire *resource.RawResource
				var envelope json.RawMessage
				var header http.Header
				var requestID string
				var code int
				var ref resource.Ref
				var err error
				if kind == "containers" {
					api := containerListFixtures()[fixtureIndex].open(t, c)
					api.RawClient().MoreHeaders = map[string]string{"X-Source": "source"}
					api.RawClient().Microversion = "1.0"
					api.Resources = nil // explicit-ID Fetch must use its owned engine
					c.Provider.SetToken("owned-fetch-token")
					value, fetchErr := api.Fetch(context.Background(), resource.ID("request-alpha"), containers.WithFetchHeader("X-Call", "operation"))
					err = fetchErr
					if value == nil || value.ContainerID == nil || *value.ContainerID != "passive-id" {
						t.Fatal(value, err)
					}
					view, wire, envelope, header, requestID, code, ref = value.Resource, value.Wire, value.Envelope, value.Header, value.RequestID, value.StatusCode, value.Ref()
				} else {
					api := orderListFixtures()[fixtureIndex].open(t, c)
					api.RawClient().MoreHeaders = map[string]string{"X-Source": "source"}
					api.RawClient().Microversion = "1.0"
					api.Resources = nil
					c.Provider.SetToken("owned-fetch-token")
					value, fetchErr := api.Fetch(context.Background(), resource.ID("request-alpha"), orders.WithFetchHeader("X-Call", "operation"))
					err = fetchErr
					if value == nil || value.OrderID == nil || *value.OrderID != "passive-id" || value.SecretID == nil || *value.SecretID != "secret-passive" {
						t.Fatal(value, err)
					}
					view, wire, envelope, header, requestID, code, ref = value.Resource, value.Wire, value.Envelope, value.Header, value.RequestID, value.StatusCode, value.Ref()
				}
				if err != nil || view == nil || wire == nil || requestID != "request-alpha" || ref.IsName() || ref.String() != "request-alpha" || code != 200 || view.StatusCode != 200 || wire.StatusCode != 200 || header.Get("X-Request-ID") != "actual receipt" || string(envelope) != body || calls.Load() != 1 || followed.Load() != 0 {
					t.Fatal(view, wire, err, calls.Load(), followed.Load())
				}
				for key, want := range map[string]string{"id": "null", "name": "3", "status": `{"state":"ERROR"}`, "created_at": `"not a timestamp"`, "updated_at": "null", "location": "null"} {
					if string(view.Body[key]) != want {
						t.Fatal(key, string(view.Body[key]), want)
					}
				}
				if _, exists := view.Body["self"]; exists {
					t.Fatal("self entered declared Resource")
				}
				if _, exists := view.Body["vendor"]; exists {
					t.Fatal("unknown extension entered declared Resource")
				}
				if string(wire.Body["vendor"]) != `{"n":9007199254740993}` || string(wire.Body["NAME"]) != "false" || string(wire.Body["self"]) != `{"raw":true}` {
					t.Fatal(wire)
				}
				field := strings.TrimSuffix(kind, "s") + "_ref"
				var passive string
				if err := json.Unmarshal(view.Body[field], &passive); err != nil || passive != reference {
					t.Fatal(passive, err)
				}
				if kind == "containers" {
					if string(view.Body["consumers"]) != `[{"not":"a native ConsumerRef"}]` || string(view.Body["secret_refs"]) != `["one",{"arbitrary":true},null]` {
						t.Fatal(view)
					}
				} else {
					if string(view.Body["meta"]) != `{"extension":{"n":9007199254740993}}` || string(view.Body["sub_status"]) != "false" {
						t.Fatal(view)
					}
				}
				// Receipt, Wire and projection are independent owned results.
				header.Set("X-Request-ID", "caller receipt")
				view.Header.Set("X-Request-ID", "caller view")
				envelope[0] = 'x'
				view.Body["name"][0] = 'x'
				wire.Body["vendor"][0] = 'x'
				if wire.Header.Get("X-Request-ID") != "actual receipt" || string(wire.Body["name"]) != "3" || string(wire.Body["self"]) != `{"raw":true}` || calls.Load() != 1 || followed.Load() != 0 {
					t.Fatal("result fields alias or passive reference followed", wire, calls.Load(), followed.Load())
				}
			})
		}
	}
}

func TestKeyManagerOwnedMetadataFetchNullableDescriptorsAndAliases(t *testing.T) {
	cases := []struct {
		kind, name, body string
		want             map[string]string
	}{
		{"containers", "missing", `{}`, map[string]string{"consumers": "null", "secret_refs": "null", "container_ref": "null", "container_id": "null", "created_at": "null", "name": "null"}},
		{"containers", "null", `{"consumers":null,"secret_refs":null,"container_ref":null}`, map[string]string{"consumers": "null", "secret_refs": "null", "container_id": "null"}},
		{"containers", "empty", `{"consumers":[],"secret_refs":[]}`, map[string]string{"consumers": "[]", "secret_refs": "[]"}},
		{"containers", "scalar and mixed", `{"consumers":false,"secret_refs":["literal",5,{"extension":[null]},null]}`, map[string]string{"consumers": "[false]", "secret_refs": `["literal",5,{"extension":[null]},null]`}},
		{"containers", "client aliases", `{"container_id":"https://foreign.invalid/containers/alias-id","created_at":"raw alias","updated_at":false}`, map[string]string{"container_ref": `"https://foreign.invalid/containers/alias-id"`, "container_id": `"alias-id"`, "created_at": `"raw alias"`, "updated_at": "false"}},
		{"containers", "canonical null wins", `{"container_ref":null,"container_id":"https://foreign.invalid/containers/ignored","created":null,"created_at":"ignored"}`, map[string]string{"container_id": "null", "container_ref": "null", "created_at": "null"}},
		{"orders", "missing", `{}`, map[string]string{"meta": "null", "name": "null", "creator_id": "null", "order_id": "null", "secret_id": "null", "created_at": "null"}},
		{"orders", "null", `{"meta":null,"order_ref":null,"secret_ref":null}`, map[string]string{"meta": "null", "order_id": "null", "secret_id": "null"}},
		{"orders", "empty dict", `{"meta":{}}`, map[string]string{"meta": "{}"}},
		{"orders", "non dict", `{"meta":[{"arbitrary":true}],"creator_id":{"raw":true}}`, map[string]string{"meta": "{}", "creator_id": `{"raw":true}`}},
		{"orders", "client aliases", `{"order_id":"https://foreign.invalid/orders/alias-order","secret_id":"https://foreign.invalid/secrets/alias-secret","created_at":7,"name":"inherited"}`, map[string]string{"order_ref": `"https://foreign.invalid/orders/alias-order"`, "order_id": `"alias-order"`, "secret_ref": `"https://foreign.invalid/secrets/alias-secret"`, "secret_id": `"alias-secret"`, "created_at": "7", "name": `"inherited"`}},
		{"orders", "canonical null wins", `{"order_ref":null,"order_id":"https://foreign.invalid/orders/ignored","secret_ref":null,"secret_id":"https://foreign.invalid/secrets/ignored"}`, map[string]string{"order_id": "null", "secret_id": "null"}},
	}
	for _, tc := range cases {
		t.Run(tc.kind+"/"+tc.name, func(t *testing.T) {
			c := testcloud.New(t)
			var calls atomic.Int32
			c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				if r.URL.Path != "/reverse/barbican/v1/"+tc.kind+"/id" || r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, tc.body)
			})
			var view, wire *resource.RawResource
			var err error
			if tc.kind == "containers" {
				v, e := containers.New(secretFetchClient(c)).Fetch(context.Background(), resource.ID("id"))
				err = e
				if v != nil {
					view, wire = v.Resource, v.Wire
				}
			} else {
				v, e := orders.New(secretFetchClient(c)).Fetch(context.Background(), resource.ID("id"))
				err = e
				if v != nil {
					view, wire = v.Resource, v.Wire
				}
			}
			if err != nil || view == nil || wire == nil || calls.Load() != 1 {
				t.Fatal(view, wire, err, calls.Load())
			}
			for field, want := range tc.want {
				if string(view.Body[field]) != want {
					t.Fatal(field, string(view.Body[field]), want)
				}
			}
			if string(view.Body["id"]) != `"id"` {
				t.Fatal("missing response ID lost request seed", view)
			}
			var original map[string]json.RawMessage
			if err := json.Unmarshal([]byte(tc.body), &original); err != nil || !reflect.DeepEqual(wire.Body, original) {
				t.Fatal("view coercion changed Wire", wire, err)
			}
		})
	}
}

func TestKeyManagerOwnedMetadataFetchIDSeedsAndReferenceFormatting(t *testing.T) {
	cases := []struct {
		name, kind, body string
		wantID, derived  string
		nullDerived, bad bool
	}{
		{name: "request seed remains distinct from HREF", kind: "containers", body: `{"container_ref":"https://foreign.invalid/containers/derived?ignored=1#fragment"}`, wantID: `"request-id"`, derived: "derived"},
		{name: "present null replaces seed", kind: "containers", body: `{"id":null,"container_ref":null}`, wantID: "null", nullDerived: true},
		{name: "response ID remains passive", kind: "containers", body: `{"id":"response-id","container_ref":"https://foreign.invalid/containers/derived"}`, wantID: `"response-id"`, derived: "derived"},
		{name: "percent spelling", kind: "containers", body: `{"container_ref":"https://foreign.invalid/containers/a%2Fb"}`, wantID: `"request-id"`, derived: "a%2Fb"},
		{name: "trailing slash", kind: "containers", body: `{"container_ref":"https://foreign.invalid/containers/"}`, wantID: `"request-id"`, derived: ""},
		{name: "invalid HREF", kind: "containers", body: `{"container_ref":"plain-segment"}`, bad: true},
		{name: "nonstring HREF", kind: "containers", body: `{"container_ref":3}`, bad: true},
		{name: "Order second formatter", kind: "orders", body: `{"order_ref":"https://foreign.invalid/orders/valid","secret_ref":"invalid-secret-HREF"}`, bad: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := testcloud.New(t)
			var calls atomic.Int32
			c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				if r.URL.Path != "/reverse/barbican/v1/"+tc.kind+"/request-id" || r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				w.Header().Set("X-Proof", "HREF")
				testcloud.JSON(w, 200, tc.body)
			})
			var view, wire *resource.RawResource
			var derived *string
			var requestID string
			var ref resource.Ref
			var envelope json.RawMessage
			var err error
			if tc.kind == "containers" {
				v, e := containers.New(secretFetchClient(c)).Fetch(context.Background(), resource.ID("request-id"))
				err = e
				if v != nil {
					view, wire, derived, requestID, ref, envelope = v.Resource, v.Wire, v.ContainerID, v.RequestID, v.Ref(), v.Envelope
				}
			} else {
				v, e := orders.New(secretFetchClient(c)).Fetch(context.Background(), resource.ID("request-id"))
				err = e
				if v != nil {
					view, wire, derived, requestID, ref, envelope = v.Resource, v.Wire, v.OrderID, v.RequestID, v.Ref(), v.Envelope
				}
			}
			if wire == nil || requestID != "request-id" || ref.String() != "request-id" || ref.IsName() || string(envelope) != tc.body || calls.Load() != 1 {
				t.Fatal(view, wire, err, calls.Load())
			}
			if tc.bad {
				var proof *resource.ResponseError
				if err == nil || view != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != tc.body || proof.Header.Get("X-Proof") != "HREF" {
					t.Fatal("formatter failure lost original response", view, wire, err, proof)
				}
				return
			}
			if err != nil || view == nil || string(view.Body["id"]) != tc.wantID {
				t.Fatal(view, err)
			}
			if tc.nullDerived {
				if derived != nil {
					t.Fatal(derived)
				}
			} else if derived == nil || *derived != tc.derived {
				t.Fatal(derived, tc.derived)
			}
		})
	}
}

func TestKeyManagerOwnedMetadataFetchHTTPAndStrictAcceptedBodies(t *testing.T) {
	cases := []struct {
		name, kind string
		code       int
		body       string
		ok, native bool
	}{
		{"valid203 passive ERROR", "containers", 203, `{"status":"ERROR"}`, true, false},
		{"valid302 without redirect", "containers", 302, `{}`, true, false},
		{"forbidden", "containers", 403, `{"error":"actual denied"}`, false, true},
		{"Order missing", "orders", 404, `{"error":"actual missing"}`, false, true},
		{"null", "containers", 200, `null`, false, false},
		{"array", "containers", 200, `[]`, false, false},
		{"truncated JSON", "containers", 200, `{"name":`, false, false},
		{"empty204", "containers", 204, ``, false, false},
		{"non UTF8 object", "containers", 200, "{\"name\":\"\xff\"}", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := testcloud.New(t)
			var calls atomic.Int32
			c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				if r.URL.Path != "/reverse/barbican/v1/"+tc.kind+"/id" || r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				w.Header().Set("X-Proof", tc.name)
				testcloud.JSON(w, tc.code, tc.body)
			})
			var view, wire *resource.RawResource
			var envelope json.RawMessage
			var code int
			var header http.Header
			var resultPresent bool
			var err error
			if tc.kind == "containers" {
				v, e := containers.New(secretFetchClient(c)).Fetch(context.Background(), resource.ID("id"))
				err = e
				resultPresent = v != nil
				if v != nil {
					view, wire, envelope, code, header = v.Resource, v.Wire, v.Envelope, v.StatusCode, v.Header
				}
			} else {
				v, e := orders.New(secretFetchClient(c)).Fetch(context.Background(), resource.ID("id"))
				err = e
				resultPresent = v != nil
				if v != nil {
					view, wire, envelope, code, header = v.Resource, v.Wire, v.Envelope, v.StatusCode, v.Header
				}
			}
			if calls.Load() != 1 {
				t.Fatal("GET repeated", calls.Load())
			}
			if tc.ok {
				if err != nil || !resultPresent || view == nil || wire == nil || code != tc.code || header.Get("X-Proof") != tc.name || string(envelope) != tc.body {
					t.Fatal(view, wire, err, code)
				}
				return
			}
			var operation *resource.OperationError
			if err == nil || !errors.As(err, &operation) || operation.Operation != "Fetch" || operation.Resource != tc.kind {
				t.Fatal(err, operation)
			}
			if tc.native {
				var rejection gophercloud.ErrUnexpectedResponseCode
				var accepted *resource.ResponseError
				if resultPresent || !errors.As(err, &rejection) || rejection.Actual != tc.code || string(rejection.Body) != tc.body || rejection.ResponseHeader.Get("X-Proof") != tc.name || errors.As(err, &accepted) {
					t.Fatal(resultPresent, err, rejection)
				}
				return
			}
			var proof *resource.ResponseError
			if !resultPresent || view != nil || wire != nil || !errors.As(err, &proof) || code != tc.code || proof.StatusCode != tc.code || string(proof.Body) != tc.body || string(envelope) != tc.body || header.Get("X-Proof") != tc.name || proof.Header.Get("X-Proof") != tc.name {
				t.Fatal("accepted decode error lost physical response", view, wire, err, proof)
			}
		})
	}
}

func TestKeyManagerOwnedMetadataFetchAcceptedFailuresRetainRawEvidence(t *testing.T) {
	// Common body errors need one engine lane each, rather than every error
	// repeated for both services. Both public wrappers still receive a partial.
	for _, tc := range []struct{ kind, mode string }{{"containers", "close"}, {"containers", "read"}, {"orders", "source"}, {"orders", "cancel"}} {
		t.Run(tc.kind+"/"+tc.mode, func(t *testing.T) {
			c := testcloud.New(t)
			client := secretFetchClient(c)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("owned metadata " + tc.mode + " failure")
			var calls, retries atomic.Int32
			const body = `{"name":"known","meta":{"n":9007199254740993},"vendor":null}`
			c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				if r.URL.Path != "/reverse/barbican/v1/"+tc.kind+"/id" {
					t.Error(r.URL)
				}
				w.Header().Set("X-Proof", tc.mode)
				testcloud.JSON(w, 200, body)
			})
			c.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			var track *payloadContractTracking
			if tc.mode == "close" {
				track = payloadContractTrack(c, nil, cause)
			} else if tc.mode == "read" {
				track = payloadContractTrack(c, cause, nil)
			} else {
				base := c.Provider.HTTPClient.Transport
				if base == nil {
					base = http.DefaultTransport
				}
				c.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					response, err := base.RoundTrip(r)
					if err != nil {
						return response, err
					}
					// Consume physical body before cancellation so complete known
					// bytes are deterministic; use the existing RoundTrip adapter.
					data, readErr := io.ReadAll(response.Body)
					closeErr := response.Body.Close()
					if readErr != nil || closeErr != nil {
						return nil, errors.Join(readErr, closeErr)
					}
					response.Body = io.NopCloser(bytes.NewReader(data))
					if tc.mode == "source" {
						client.ResourceBase = c.Server.URL + "/changed/v1/"
					} else {
						cancel(cause)
					}
					return response, nil
				})
			}
			var view, wire *resource.RawResource
			var envelope json.RawMessage
			var header http.Header
			var code int
			var requestID string
			var err error
			if tc.kind == "containers" {
				v, e := containers.New(client).Fetch(ctx, resource.ID("id"))
				err = e
				if v != nil {
					view, wire, envelope, header, code, requestID = v.Resource, v.Wire, v.Envelope, v.Header, v.StatusCode, v.RequestID
				}
			} else {
				v, e := orders.New(client).Fetch(ctx, resource.ID("id"))
				err = e
				if v != nil {
					view, wire, envelope, header, code, requestID = v.Resource, v.Wire, v.Envelope, v.Header, v.StatusCode, v.RequestID
				}
			}
			var proof *resource.ResponseError
			if err == nil || view != nil || wire == nil || requestID != "id" || code != 200 || string(envelope) != body || string(wire.Body["name"]) != `"known"` || header.Get("X-Proof") != tc.mode || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || proof.Header.Get("X-Proof") != tc.mode || calls.Load() != 1 || retries.Load() != 0 {
				t.Fatal("accepted read failure discarded known object or repeated HTTP", view, wire, err, proof, calls.Load(), retries.Load())
			}
			if tc.mode == "source" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if tc.mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if track != nil {
				physical := track.last(t)
				if track.calls.Load() != 1 || physical.reads.Load() == 0 || physical.closes.Load() != 1 {
					t.Fatal("body not closed exactly once", track.calls.Load(), physical.reads.Load(), physical.closes.Load())
				}
			}
			header.Set("X-Proof", "caller")
			envelope[0] = 'x'
			if wire.Header.Get("X-Proof") != tc.mode || string(proof.Body) != body || proof.Header.Get("X-Proof") != tc.mode {
				t.Fatal("error/result receipts alias", wire, proof)
			}
		})
	}
}

func TestKeyManagerOwnedMetadataFetchPreparedOptionsAndSourceBoundaries(t *testing.T) {
	t.Run("Order callbacks and owned header slice snapshot", func(t *testing.T) {
		c := testcloud.New(t)
		api := orders.New(secretFetchClient(c))
		var calls atomic.Int32
		headers := map[string]string{"X-Shared": "initial"}
		counts := [3]int{}
		var options []orders.FetchOption
		owned := orders.WithFetchHeader("X-Owned", "literal")
		options = []orders.FetchOption{
			func(config *request.Config[orders.FetchOpts]) error {
				counts[0]++
				config.Headers = headers
				options[1] = func(*request.Config[orders.FetchOpts]) error {
					t.Error("caller replaced captured option")
					return errors.New("poison")
				}
				return nil
			},
			func(config *request.Config[orders.FetchOpts]) error { counts[1]++; return owned(config) },
			func(*request.Config[orders.FetchOpts]) error {
				counts[2]++
				headers["X-Shared"] = "caller changed"
				api.Resources = nil
				return nil
			},
		}
		c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.URL.Path != orderListPath+"/id" || r.URL.RawQuery != "" || r.Header.Get("X-Shared") != "initial" || r.Header.Get("X-Owned") != "literal" {
				t.Error(r.URL, r.Header)
			}
			testcloud.JSON(w, 200, `{"meta":{"arbitrary":true}}`)
		})
		value, err := api.Fetch(context.Background(), resource.ID("id"), options...)
		if err != nil || value == nil || value.Resource == nil || calls.Load() != 1 || counts != [3]int{1, 1, 1} || headers["X-Shared"] != "caller changed" || string(value.Resource.Body["meta"]) != `{"arbitrary":true}` {
			t.Fatal(value, err, calls.Load(), counts)
		}
	})
	for _, mode := range []string{"empty ID", "path ID", "control ID", "full HREF ID", "Name", "nil context", "cancelled", "nil API", "nil provider", "wrong service", "bad endpoint", "auth header", "unsupported query", "nil option", "source before later option", "provider before later option", "version before later option", "cancel before later option"} {
		t.Run(mode, func(t *testing.T) {
			c := testcloud.New(t)
			client := secretFetchClient(c)
			api := containers.New(client)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("preflight custom cause")
			ref := resource.ID("id")
			var options []containers.FetchOption
			var calls, later atomic.Int32
			want := resource.ErrInvalidOption
			switch mode {
			case "empty ID":
				ref = resource.ID("")
			case "path ID":
				ref = resource.ID("a/b")
			case "control ID":
				ref = resource.ID("a\x00b")
			case "full HREF ID":
				ref = resource.ID("https://foreign.invalid/containers/id")
			case "Name":
				ref = resource.Name("literal")
				want = resource.ErrUnsupported
			case "nil context":
				ctx = nil
			case "cancelled":
				cancel(cause)
				want = context.Canceled
			case "nil API":
				api = nil
			case "nil provider":
				client.ProviderClient = nil
			case "wrong service":
				client.Type = "compute"
				want = resource.ErrUnsupported
			case "bad endpoint":
				client.Endpoint = "https://bad.invalid/?query=not-route"
			case "auth header":
				options = []containers.FetchOption{containers.WithFetchHeader("X-Auth-Token", "not allowed")}
			case "unsupported query":
				options = []containers.FetchOption{request.WithQuery[containers.FetchOpts]("filter", "ignored")}
			case "nil option":
				options = []containers.FetchOption{nil}
			default:
				options = []containers.FetchOption{func(*request.Config[containers.FetchOpts]) error {
					switch mode {
					case "source before later option":
						client.ResourceBase = c.Server.URL + "/changed/v1/"
					case "provider before later option":
						client.ProviderClient = &gophercloud.ProviderClient{}
					case "version before later option":
						client.Microversion = "1.1"
					case "cancel before later option":
						cancel(cause)
						want = context.Canceled
					}
					return nil
				}, func(*request.Config[containers.FetchOpts]) error { later.Add(1); return nil }}
			}
			c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, `{}`) })
			value, err := api.Fetch(ctx, ref, options...)
			if value != nil || !errors.Is(err, want) || calls.Load() != 0 || later.Load() != 0 {
				t.Fatal(mode, value, err, want, calls.Load(), later.Load())
			}
			if (mode == "cancelled" || mode == "cancel before later option") && !errors.Is(err, cause) {
				t.Fatal("custom cancellation cause lost", err)
			}
		})
	}
	t.Run("retry cannot replace captured route", func(t *testing.T) {
		c := testcloud.New(t)
		client := secretFetchClient(c)
		var calls, retries atomic.Int32
		c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.URL.Path != containerListPath+"/id" || r.Method != http.MethodGet {
				t.Error(r.URL, r.Method)
			}
			w.Header().Set("X-Proof", "original retry failure")
			testcloud.JSON(w, 503, `{"error":"original retry failure"}`)
		})
		c.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, _ error, _ uint) error {
			retries.Add(1)
			client.ResourceBase = c.Server.URL + "/changed/v1/"
			return nil
		}
		value, err := containers.New(client).Fetch(context.Background(), resource.ID("id"))
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"error":"original retry failure"}` || native.ResponseHeader.Get("X-Proof") != "original retry failure" || calls.Load() != 1 || retries.Load() != 1 {
			t.Fatal(value, err, native, calls.Load(), retries.Load())
		}
	})
}
