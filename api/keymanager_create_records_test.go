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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	upcontainers "github.com/gophercloud/gophercloud/v2/openstack/keymanager/v1/containers"
	uporders "github.com/gophercloud/gophercloud/v2/openstack/keymanager/v1/orders"
	upsecrets "github.com/gophercloud/gophercloud/v2/openstack/keymanager/v1/secrets"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

// Existing leaf/cached-Connection fixtures and body wrappers are reused.
// Common failure matrices are representative; each descriptor's body is distinct.
func TestKeyManagerCreateRecordFlatBodiesSeedAndNullResponse(t *testing.T) {
	for _, kind := range []string{"containers", "orders", "secrets"} {
		for fixtureIndex, fixtureName := range []string{"leaf", "cached connection"} {
			for _, mode := range []string{"empty", "seed", "response-null"} {
				t.Run(kind+"/"+fixtureName+"/"+mode, func(t *testing.T) {
					c := testcloud.New(t)
					var calls atomic.Int32
					refKey := map[string]string{"containers": "container_ref", "orders": "order_ref", "secrets": "secret_ref"}[kind]
					ref := "https://foreign.invalid/" + kind + "/new%2fID?query=kept#fragment"
					code := 201
					if kind == "orders" {
						code = 202
					}
					attrs := map[string]any{"ignored": func() {}}
					want := map[string]any{}
					if mode != "empty" {
						attrs["id"], attrs["created_at"], attrs["updated_at"] = "literal-seed", "raw-created", nil
						want["id"], want["created"], want["updated"] = "literal-seed", "raw-created", nil
						attrs[refKey], want[refKey] = "https://input.invalid/"+kind+"/canonical", "https://input.invalid/"+kind+"/canonical"
						attrs[map[string]string{"containers": "container_id", "orders": "order_id", "secrets": "secret_id"}[kind]] = "relative/ignored-alias"
						switch kind {
						case "containers":
							attrs["name"], attrs["type"], attrs["secret_refs"], attrs["consumers"] = "", "generic", []any{}, nil
							want["name"], want["type"], want["secret_refs"], want["consumers"] = "", "generic", []any{}, nil
						case "orders":
							attrs["type"], attrs["meta"], attrs["secret_id"] = "key", map[string]any{"name": "nested", "bit_length": json.Number("9007199254740993")}, "https://input.invalid/secrets/seed%2Fsecret"
							want["type"], want["meta"], want["secret_ref"] = "key", attrs["meta"], attrs["secret_id"]
						case "secrets":
							attrs["name"], attrs["bit_length"], attrs["expires_at"], attrs["payload"], attrs["payload_content_type"], attrs["payload_content_encoding"], attrs["content_types"] = "", json.Number("9007199254740993"), "raw-expiration", "payload-seed", "text/plain", nil, map[string]any{}
							want["name"], want["bit_length"], want["expiration"], want["payload"], want["payload_content_type"], want["payload_content_encoding"], want["content_types"] = "", attrs["bit_length"], "raw-expiration", "payload-seed", "text/plain", nil, map[string]any{}
						}
					}
					responseFields := map[string]any{refKey: ref}
					if mode == "response-null" {
						responseFields[refKey], responseFields["id"], responseFields["created"], responseFields["updated"] = nil, nil, nil, ""
						if kind == "orders" {
							responseFields["secret_ref"], responseFields["meta"] = nil, nil
						}
						if kind == "containers" {
							responseFields["secret_refs"] = nil
						}
						if kind == "secrets" {
							responseFields["payload"] = nil
						}
					}
					body, err := json.Marshal(responseFields)
					if err != nil {
						t.Fatal(err)
					}
					wantBody, err := json.Marshal(want)
					if err != nil {
						t.Fatal(err)
					}
					c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						th.TestMethod(t, r, http.MethodPost)
						th.TestHeader(t, r, "X-Auth-Token", "live-create-token")
						th.TestHeader(t, r, "X-Source", "shared")
						th.TestHeader(t, r, "X-Call", "owned")
						th.TestHeader(t, r, "OpenStack-API-Version", "key-manager 1.0")
						if r.URL.Path != "/reverse/barbican/v1/"+kind || r.URL.RawQuery != "" {
							t.Error("lookup/payload/ref follow instead of flat create", r.Method, r.URL)
						}
						got, readErr := io.ReadAll(r.Body)
						if readErr != nil || !bytes.Equal(got, wantBody) {
							t.Error("flat nullable/alias request changed", string(got), string(wantBody), readErr)
						}
						w.Header().Set("X-Create-Proof", "actual")
						testcloud.JSON(w, code, string(body))
					})
					var client *gophercloud.ServiceClient
					var view, wire *resource.RawResource
					var envelope json.RawMessage
					var header http.Header
					var status int
					var primary, secondary *string
					switch kind {
					case "containers":
						a := containerListFixtures()[fixtureIndex].open(t, c)
						client = a.RawClient()
						client.MoreHeaders = map[string]string{"X-Source": "shared"}
						client.Microversion = "1.0"
						c.Provider.SetToken("live-create-token")
						v, e := a.CreateRecord(context.Background(), containers.WithCreateRecordOptions(containers.CreateRecordOpts{Attributes: attrs}), containers.WithCreateRecordHeader("X-Call", "owned"))
						err = e
						if v == nil {
							t.Fatal(v, err)
						}
						view, wire, envelope, header, status, primary = v.Resource, v.Wire, v.Envelope, v.Header, v.StatusCode, v.ContainerID
					case "orders":
						a := orderListFixtures()[fixtureIndex].open(t, c)
						client = a.RawClient()
						client.MoreHeaders = map[string]string{"X-Source": "shared"}
						client.Microversion = "1.0"
						c.Provider.SetToken("live-create-token")
						v, e := a.CreateRecord(context.Background(), orders.WithCreateRecordOptions(orders.CreateRecordOpts{Attributes: attrs}), orders.WithCreateRecordHeader("X-Call", "owned"))
						err = e
						if v == nil {
							t.Fatal(v, err)
						}
						view, wire, envelope, header, status, primary, secondary = v.Resource, v.Wire, v.Envelope, v.Header, v.StatusCode, v.OrderID, v.SecretID
					case "secrets":
						a := secretListFixtures()[fixtureIndex].open(t, c)
						client = a.RawClient()
						client.MoreHeaders = map[string]string{"X-Source": "shared"}
						client.Microversion = "1.0"
						c.Provider.SetToken("live-create-token")
						v, e := a.CreateRecord(context.Background(), secrets.WithCreateRecordOptions(secrets.CreateRecordOpts{Attributes: attrs}), secrets.WithCreateRecordHeader("X-Call", "owned"))
						err = e
						if v == nil {
							t.Fatal(v, err)
						}
						view, wire, envelope, header, status, primary = v.Resource, v.Wire, v.Envelope, v.Header, v.StatusCode, v.SecretID
					}
					if err != nil || view == nil || wire == nil || status != code || view.StatusCode != code || wire.StatusCode != code || header.Get("X-Create-Proof") != "actual" || string(envelope) != string(body) || calls.Load() != 1 || client.ProviderClient != c.Provider || client.ResourceBase != c.Server.URL+"/reverse/barbican/v1/" {
						t.Fatal("creation/input merge/pure receipt/shared cached source", view, wire, err, calls.Load())
					}
					var wantWire map[string]json.RawMessage
					if err := json.Unmarshal(body, &wantWire); err != nil || !reflect.DeepEqual(wire.Body, wantWire) {
						t.Fatal("Wire invented input seed/defaults", wire, err)
					}
					if mode == "response-null" {
						if primary != nil || secondary != nil || string(view.Body["id"]) != "null" || string(view.Body["created_at"]) != "null" || string(view.Body["updated_at"]) != `""` {
							t.Fatal("response null/empty did not override input", view, primary, secondary)
						}
					} else {
						if primary == nil || *primary != "new%2fID" {
							t.Fatal("passive convenience suffix", primary)
						}
						wantID, _ := json.Marshal(ref)
						if mode == "seed" {
							wantID = json.RawMessage(`"literal-seed"`)
							if string(view.Body["created_at"]) != `"raw-created"` || string(view.Body["updated_at"]) != "null" {
								t.Fatal("ref-only response lost original input", view)
							}
						}
						if string(view.Body["id"]) != string(wantID) {
							t.Fatal("Resource id confused with suffix", view)
						}
						if kind == "orders" && mode == "seed" && (secondary == nil || *secondary != "seed%2Fsecret" || !strings.Contains(string(view.Body["meta"]), "9007199254740993")) {
							t.Fatal("ref-only order lost meta/secret seed", view, secondary)
						}
						if kind == "secrets" && mode == "seed" && (string(view.Body["payload"]) != `"payload-seed"` || string(view.Body["bit_length"]) != "9007199254740993" || string(view.Body["expires_at"]) != `"raw-expiration"`) {
							t.Fatal("ref-only secret lost exact seed", view)
						}
					}
					if mode == "empty" {
						if _, exists := wire.Body["type"]; exists {
							t.Fatal("default type invented in actual receipt", wire)
						}
						if kind != "secrets" && string(view.Body["type"]) != "null" {
							t.Fatal("default create type invented", view)
						}
					}
				})
			}
		}
	}
}

func TestKeyManagerCreateRecordOwnedOptionsAliasesAndIndependentProof(t *testing.T) {
	c := testcloud.New(t)
	a := containerListFixtures()[1].open(t, c)
	var calls, callbacks atomic.Int32
	nested := map[string]any{"name": "original", "secret_ref": "https://passive.invalid/s", "vendor": json.Number("9007199254740993")}
	refs := []any{nested}
	attrs := map[string]any{"name": "replaced", "type": nil, "secret_refs": refs, "created_at": "discarded-alias", "created": nil, "container_ref": "https://input.invalid/valid", "container_id": "relative/ignored"}
	bulk := containers.WithCreateRecordAttributes(attrs)
	vendor := map[string]any{"n": json.Number("9007199254740993")}
	field := containers.WithCreateRecordField("vendor", vendor)
	attrs["name"], nested["name"], vendor["n"], refs[0] = "caller-change", "caller-change", 0, nil
	options := []containers.CreateRecordOption{containers.WithCreateRecordOptions(containers.CreateRecordOpts{Attributes: map[string]any{"name": "discarded", "consumers": []any{"discarded"}}}), field, containers.WithCreateRecordHeader("X-Call", "owned"), bulk, containers.WithCreateRecordAttribute("name", ""), containers.WithCreateRecordAttribute("updated_at", "first"), containers.WithCreateRecordAttribute("updated", "last"), containers.WithCreateRecordAttribute("ignored", func() {})}
	const response = `{"container_ref":"https://foreign.invalid/containers/raw%2Fid?x=1#fragment"}`
	want := `{"container_ref":"https://input.invalid/valid","created":null,"name":"","secret_refs":[{"name":"original","secret_ref":"https://passive.invalid/s","vendor":9007199254740993}],"type":null,"updated":"last","vendor":{"n":9007199254740993}}`
	c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		th.TestMethod(t, r, http.MethodPost)
		th.TestHeader(t, r, "X-Call", "owned")
		if r.URL.Path != containerListPath || r.URL.RawQuery != "" {
			t.Error(r.URL)
		}
		body, e := io.ReadAll(r.Body)
		if e != nil || string(body) != want {
			t.Error("snapshot, bulk scope or alias last-value changed", string(body), e)
		}
		w.Header().Set("X-Create-Proof", "original")
		testcloud.JSON(w, 201, response)
	})
	var retained *request.Config[containers.CreateRecordOpts]
	custom := func(config *request.Config[containers.CreateRecordOpts]) error {
		callbacks.Add(1)
		retained = config
		return nil
	}
	base := c.Provider.HTTPClient.Transport
	c.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		retained.Options.Attributes["name"] = "late"
		retained.Options.Attributes["secret_refs"] = "late"
		retained.Fields["vendor"][0] = 'x'
		retained.Headers["X-Call"] = "late"
		return base.RoundTrip(r)
	})
	value, err := a.CreateRecord(context.Background(), append(append([]containers.CreateRecordOption(nil), options...), custom)...)
	c.Provider.HTTPClient.Transport = base
	if err != nil || value == nil || value.Resource == nil || value.Wire == nil || value.ContainerID == nil || *value.ContainerID != "raw%2Fid" || callbacks.Load() != 1 || calls.Load() != 1 {
		t.Fatal(value, err, callbacks.Load(), calls.Load())
	}
	if string(value.Resource.Body["name"]) != `""` || string(value.Resource.Body["created_at"]) != "null" || string(value.Resource.Body["updated_at"]) != `"last"` || string(value.Resource.Body["consumers"]) != "null" {
		t.Fatal("bulk attrs did not replace only semantic set", value.Resource)
	}
	value.Resource.Body["container_ref"][1] = 'X'
	value.Resource.Header.Set("X-Create-Proof", "resource")
	value.Header.Set("X-Create-Proof", "outer")
	*value.ContainerID = "caller"
	if string(value.Envelope) != response || value.Wire.Header.Get("X-Create-Proof") != "original" || string(value.Wire.Body["container_ref"]) != `"https://foreign.invalid/containers/raw%2Fid?x=1#fragment"` || string(value.Resource.Body["container_id"]) != `"raw%2Fid"` {
		t.Fatal("Resource/header/nullable suffix aliased Wire or envelope", value)
	}
	value.Wire.Body["container_ref"][1] = 'W'
	value.Wire.Header.Set("X-Create-Proof", "wire")
	if string(value.Envelope) != response || value.Header.Get("X-Create-Proof") != "outer" || value.Resource.Header.Get("X-Create-Proof") != "resource" {
		t.Fatal("Wire mutation changed another receipt", value)
	}
	var group sync.WaitGroup
	for range 4 {
		group.Add(1)
		go func() {
			defer group.Done()
			v, e := a.CreateRecord(context.Background(), options...)
			if e != nil || v == nil || v.ContainerID == nil || *v.ContainerID != "raw%2Fid" || v.Header.Get("X-Create-Proof") != "original" {
				t.Error(v, e)
			}
		}()
	}
	group.Wait()
	if calls.Load() != 5 || callbacks.Load() != 1 {
		t.Fatal("immutable option reuse repeated application callbacks", calls.Load(), callbacks.Load())
	}
}

func TestKeyManagerCreateRecordRequestTypesAndResourceDescriptorsAreSeparate(t *testing.T) {
	for _, kind := range []string{"containers", "orders", "secrets"} {
		t.Run(kind, func(t *testing.T) {
			c := testcloud.New(t)
			var calls atomic.Int32
			field, raw, wantView := "secret_refs", any("scalar-ref"), `["scalar-ref"]`
			if kind == "orders" {
				field, raw, wantView = "meta", []any{"raw-meta"}, `{}`
			}
			if kind == "secrets" {
				field, raw, wantView = "content_types", false, `{}`
			}
			refKey := map[string]string{"containers": "container_ref", "orders": "order_ref", "secrets": "secret_ref"}[kind]
			response := fmt.Sprintf(`{%q:%q}`, refKey, "https://foreign.invalid/"+kind+"/actual")
			wantBody, e := json.Marshal(map[string]any{field: raw})
			if e != nil {
				t.Fatal(e)
			}
			c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				if r.URL.Path != "/reverse/barbican/v1/"+kind || r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				body, e := io.ReadAll(r.Body)
				if e != nil || !bytes.Equal(body, wantBody) {
					t.Error("input coerced before POST", string(body), string(wantBody), e)
				}
				testcloud.JSON(w, 202, response)
			})
			var view, wire *resource.RawResource
			var err error
			switch kind {
			case "containers":
				v, e := containerListFixtures()[0].open(t, c).CreateRecord(context.Background(), containers.WithCreateRecordAttribute(field, raw))
				err = e
				if v != nil {
					view, wire = v.Resource, v.Wire
				}
			case "orders":
				v, e := orderListFixtures()[0].open(t, c).CreateRecord(context.Background(), orders.WithCreateRecordAttribute(field, raw))
				err = e
				if v != nil {
					view, wire = v.Resource, v.Wire
				}
			case "secrets":
				v, e := secretListFixtures()[0].open(t, c).CreateRecord(context.Background(), secrets.WithCreateRecordAttribute(field, raw))
				err = e
				if v != nil {
					view, wire = v.Resource, v.Wire
				}
			}
			if err != nil || view == nil || wire == nil || string(view.Body[field]) != wantView || calls.Load() != 1 {
				t.Fatal("response-only list/dict descriptor or seed changed", view, wire, err, calls.Load())
			}
			if _, exists := wire.Body[field]; exists {
				t.Fatal("input normalization invented an actual response field", wire)
			}
		})
	}

	t.Run("sole-order-meta-field-extension", func(t *testing.T) {
		c := testcloud.New(t)
		var calls atomic.Int32
		c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestMethod(t, r, http.MethodPost)
			if r.URL.Path != orderListPath || r.URL.RawQuery != "" {
				t.Error(r.URL)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != `{"meta":{"n":9007199254740993},"vendor":{"proof":"owned"}}` {
				t.Error("sole meta map became an envelope", string(body), err)
			}
			testcloud.JSON(w, 202, `{"order_ref":"https://foreign.invalid/orders/actual"}`)
		})
		value, err := orderListFixtures()[1].open(t, c).CreateRecord(context.Background(), orders.WithCreateRecordAttribute("meta", map[string]any{"n": json.Number("9007199254740993")}), orders.WithCreateRecordField("vendor", map[string]any{"proof": "owned"}))
		if err != nil || value == nil || value.Resource == nil || value.Wire == nil || string(value.Resource.Body["meta"]) != `{"n":9007199254740993}` || string(value.Resource.Body["type"]) != "null" || calls.Load() != 1 {
			t.Fatal(value, err, calls.Load())
		}
		if _, present := value.Wire.Body["meta"]; present {
			t.Fatal("Wire gained request metadata", value.Wire)
		}
	})
}

func TestKeyManagerCreateRecordAcceptedReceiptsAndNativeFailures(t *testing.T) {
	valid := `{"container_ref":"https://foreign.invalid/containers/created"}`
	for _, tc := range []struct {
		name      string
		code      int
		body      string
		fault     string
		wantError bool
	}{
		{"accepted-200", 200, valid, "", false}, {"accepted-201", 201, valid, "", false}, {"accepted-202", 202, valid, "", false}, {"accepted-299", 299, valid, "", false}, {"accepted-399", 399, valid, "", false},
		{"malformed", 201, `{"container_ref":`, "", true}, {"null-root", 201, `null`, "", true}, {"array-root", 201, `[]`, "", true}, {"invalid-utf8", 201, "{\"container_ref\":\"\xff\"}", "", true}, {"empty-204", 204, "", "", true}, {"invalid-response-ref", 201, `{"container_ref":"relative/invalid"}`, "", true},
		{"accepted-read", 201, valid, "read", true}, {"accepted-close404", 201, valid, "close404", true},
		{"forbidden", 403, `{"error":"original"}`, "", true}, {"missing", 404, `{"error":"original"}`, "", true}, {"conflict", 409, `{"error":"original"}`, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := testcloud.New(t)
			a := containerListFixtures()[0].open(t, c)
			var calls, retries atomic.Int32
			c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				if r.URL.Path != containerListPath || r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				w.Header().Set("X-Create-Proof", tc.name)
				testcloud.JSON(w, tc.code, tc.body)
			})
			cause := errors.New("accepted create read")
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{201}, Method: http.MethodPost, URL: c.Server.URL + containerListPath, Body: []byte("nested404"), ResponseHeader: http.Header{"X-Nested-Proof": {"kept"}}}
			var track *payloadContractTracking
			if tc.fault == "read" {
				track = payloadContractTrack(c, cause, nil)
			}
			if tc.fault == "close404" {
				track = payloadContractTrack(c, nil, nested)
			}
			c.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			value, err := a.CreateRecord(context.Background())
			if calls.Load() != 1 {
				t.Fatal("create performed lookup/fallback/replay", calls.Load(), value, err)
			}
			var operation *resource.OperationError
			if !tc.wantError {
				if err != nil || value == nil || value.Resource == nil || value.Wire == nil || value.ContainerID == nil || *value.ContainerID != "created" || value.StatusCode != tc.code || string(value.Envelope) != tc.body || value.Header.Get("X-Create-Proof") != tc.name || retries.Load() != 0 {
					t.Fatal(value, err, retries.Load())
				}
				return
			}
			if err == nil || !errors.As(err, &operation) || operation.Operation != "CreateRecord" || operation.Resource != "containers" {
				t.Fatal("public owned operation cause", value, err, operation)
			}
			if tc.code >= 400 {
				var native gophercloud.ErrUnexpectedResponseCode
				var proof *resource.ResponseError
				if value != nil || !errors.As(err, &native) || native.Actual != tc.code || native.Method != http.MethodPost || native.URL != c.Server.URL+containerListPath || string(native.Body) != tc.body || native.ResponseHeader.Get("X-Create-Proof") != tc.name || len(native.Expected) != 200 || native.Expected[0] != 200 || native.Expected[199] != 399 || errors.As(err, &proof) || retries.Load() != 1 {
					t.Fatal("native rejection lost original evidence", value, err, native, retries.Load())
				}
				return
			}
			var proof *resource.ResponseError
			if value == nil || value.Resource != nil || value.StatusCode != tc.code || value.Header.Get("X-Create-Proof") != tc.name || string(value.Envelope) != tc.body || !errors.As(err, &proof) || proof.StatusCode != tc.code || proof.Header.Get("X-Create-Proof") != tc.name || string(proof.Body) != tc.body || retries.Load() != 0 {
				t.Fatal("accepted partial proof lost or processing failure replayed", value, err, proof, retries.Load())
			}
			if tc.name == "invalid-response-ref" && (value.Wire == nil || string(value.Wire.Body["container_ref"]) != `"relative/invalid"` || !errors.Is(err, resource.ErrInvalidOption)) {
				t.Fatal("invalid view lost valid raw response", value, err)
			}
			if tc.fault == "read" && !errors.Is(err, cause) {
				t.Fatal("read cause lost", err)
			}
			if tc.fault == "close404" {
				var original gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &original) || original.Actual != 404 || string(original.Body) != "nested404" || original.ResponseHeader.Get("X-Nested-Proof") != "kept" || errors.Is(err, resource.ErrNotFound) {
					t.Fatal("accepted nested404 ignored/reclassified", err, original)
				}
			}
			if track != nil {
				body := track.last(t)
				if track.calls.Load() != 1 || body.reads.Load() == 0 || body.closes.Load() != 1 {
					t.Fatal("body ownership", track.calls.Load(), body.reads.Load(), body.closes.Load())
				}
			}
		})
	}
}

func TestKeyManagerCreateRecordPreflightSourceContextAndNativeReplay(t *testing.T) {
	t.Run("preflight", func(t *testing.T) {
		c := testcloud.New(t)
		var calls atomic.Int32
		c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
		for _, mode := range []string{"nil API", "nil context", "nil provider", "wrong service", "nil option", "known field", "known alias", "selected encoding", "bad input ref", "query", "argument", "cancel", "option source"} {
			t.Run(mode, func(t *testing.T) {
				client := secretFetchClient(c)
				a := containers.New(client)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("create preflight cancelled")
				var options []containers.CreateRecordOption
				var later atomic.Int32
				want := resource.ErrInvalidOption
				switch mode {
				case "nil API":
					a = nil
				case "nil context":
					ctx = nil
				case "nil provider":
					client.ProviderClient = nil
				case "wrong service":
					client.Type = "compute"
					want = resource.ErrUnsupported
				case "nil option":
					options = []containers.CreateRecordOption{nil}
				case "known field":
					options = []containers.CreateRecordOption{containers.WithCreateRecordField("container_ref", "https://input.invalid/id")}
				case "known alias":
					options = []containers.CreateRecordOption{containers.WithCreateRecordField("created_at", "input")}
				case "selected encoding":
					options = []containers.CreateRecordOption{containers.WithCreateRecordAttribute("secret_refs", func() {})}
				case "bad input ref":
					options = []containers.CreateRecordOption{containers.WithCreateRecordAttribute("container_id", "relative/invalid")}
				case "query":
					options = []containers.CreateRecordOption{request.WithQuery[containers.CreateRecordOpts]("vendor", "query")}
				case "argument":
					options = []containers.CreateRecordOption{func(config *request.Config[containers.CreateRecordOpts]) error {
						config.Arguments["base_path"] = "other"
						return nil
					}}
				case "cancel":
					cancel(cause)
					want = context.Canceled
				case "option source":
					options = []containers.CreateRecordOption{func(config *request.Config[containers.CreateRecordOpts]) error {
						client.ResourceBase = c.Server.URL + "/changed/v1/"
						return nil
					}, func(config *request.Config[containers.CreateRecordOpts]) error { later.Add(1); return nil }}
				}
				value, err := a.CreateRecord(ctx, options...)
				if value != nil || !errors.Is(err, want) || calls.Load() != 0 || later.Load() != 0 {
					t.Fatal("preflight consumed wrong source/next callback/HTTP", value, err, calls.Load(), later.Load())
				}
				if mode == "cancel" && !errors.Is(err, cause) {
					t.Fatal("custom cancellation cause lost", err)
				}
				if mode == "selected encoding" {
					var cause *json.UnsupportedTypeError
					if !errors.As(err, &cause) {
						t.Fatal("selected JSON error lost", err)
					}
				}
			})
		}
	})
	for _, mode := range []string{"accepted source", "accepted cancel"} {
		t.Run(mode, func(t *testing.T) {
			c := testcloud.New(t)
			a := containerListFixtures()[1].open(t, c)
			var calls, retries atomic.Int32
			const body = `{"container_ref":"https://foreign.invalid/containers/created"}`
			c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				w.Header().Set("X-Create-Proof", mode)
				testcloud.JSON(w, 201, body)
			})
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("accepted create cancelled")
			base := c.Provider.HTTPClient.Transport
			c.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
				response, err := base.RoundTrip(r)
				if err != nil {
					return response, err
				}
				data, readErr := io.ReadAll(response.Body)
				closeErr := response.Body.Close()
				if readErr != nil || closeErr != nil {
					return nil, errors.Join(readErr, closeErr)
				}
				response.Body = io.NopCloser(bytes.NewReader(data))
				if mode == "accepted source" {
					a.RawClient().ResourceBase = c.Server.URL + "/changed/v1/"
				} else {
					cancel(cause)
				}
				return response, nil
			})
			c.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			value, err := a.CreateRecord(ctx)
			var proof *resource.ResponseError
			if value == nil || value.Resource != nil || value.StatusCode != 201 || string(value.Envelope) != body || value.Header.Get("X-Create-Proof") != mode || !errors.As(err, &proof) || proof.StatusCode != 201 || string(proof.Body) != body || calls.Load() != 1 || retries.Load() != 0 {
				t.Fatal("accepted source/context change lost physical proof", value, err, proof, calls.Load(), retries.Load())
			}
			if mode == "accepted source" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if mode == "accepted cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
				t.Fatal("typed/custom cancellation lost", err)
			}
		})
	}
	for _, mode := range []string{"safe live token", "source retry", "expanded404"} {
		t.Run(mode, func(t *testing.T) {
			c := testcloud.New(t)
			a := containerListFixtures()[1].open(t, c)
			var calls, retries, callbacks atomic.Int32
			const accepted = `{"container_ref":"https://foreign.invalid/containers/created"}`
			c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				if r.URL.Path != containerListPath || r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				requestBody, e := io.ReadAll(r.Body)
				if e != nil || string(requestBody) != `{"name":"owned"}` {
					t.Error("retry reapplied options or changed serialized body", string(requestBody), e)
				}
				if mode == "safe live token" {
					token := "test-token"
					if n > 1 {
						token = "refreshed-token"
						th.TestHeader(t, r, "X-Native-Retry", "kept")
					}
					th.TestHeader(t, r, "X-Auth-Token", token)
				}
				w.Header().Set("X-Create-Proof", mode)
				if mode == "expanded404" {
					testcloud.JSON(w, 404, `{"error":"original"}`)
				} else if n == 1 {
					testcloud.JSON(w, 503, `{"error":"original"}`)
				} else {
					testcloud.JSON(w, 201, accepted)
				}
			})
			c.Provider.RetryFunc = func(_ context.Context, method, target string, options *gophercloud.RequestOpts, original error, count uint) error {
				retries.Add(1)
				if method != http.MethodPost || target != c.Server.URL+containerListPath || count != 1 {
					t.Error(method, target, count)
				}
				if retries.Load() > 1 {
					return original
				}
				if mode == "source retry" {
					a.RawClient().ResourceBase = c.Server.URL + "/changed/v1/"
				} else if mode == "expanded404" {
					options.OkCodes = append(options.OkCodes, 404)
				} else {
					c.Provider.SetToken("refreshed-token")
					if options.MoreHeaders == nil {
						options.MoreHeaders = map[string]string{}
					}
					options.MoreHeaders["X-Native-Retry"] = "kept"
				}
				return nil
			}
			value, err := a.CreateRecord(context.Background(), containers.WithCreateRecordAttribute("name", "owned"), func(config *request.Config[containers.CreateRecordOpts]) error { callbacks.Add(1); return nil })
			if callbacks.Load() != 1 || retries.Load() != 1 {
				t.Fatal("native resend reapplied user options", value, err, callbacks.Load(), retries.Load())
			}
			if mode == "safe live token" {
				if err != nil || value == nil || value.ContainerID == nil || *value.ContainerID != "created" || string(value.Resource.Body["name"]) != `"owned"` || calls.Load() != 2 {
					t.Fatal(value, err, calls.Load())
				}
				return
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if err == nil || !errors.As(err, &native) || string(native.Body) != `{"error":"original"}` || native.ResponseHeader.Get("X-Create-Proof") != mode {
				t.Fatal("native rejected request lost actual evidence", value, err, native)
			}
			if mode == "source retry" {
				if !errors.Is(err, resource.ErrInvalidOption) || native.Actual != 503 || calls.Load() != 1 {
					t.Fatal(value, err, native, calls.Load())
				}
			} else {
				var terminal interface{ TerminalSDKFailure() bool }
				if native.Actual != 404 || calls.Load() != 2 || !errors.As(err, &terminal) || !terminal.TerminalSDKFailure() {
					t.Fatal("expanded rejection became owned accepted creation", value, err, native, calls.Load())
				}
			}
		})
	}
}

func TestKeyManagerNativeCreateAliasesBodiesAndStatusRemainUnchanged(t *testing.T) {
	var _ containers.CreateOpts = upcontainers.CreateOpts{}
	var _ containers.CreateResult = upcontainers.CreateResult{}
	var _ *containers.Container = (*upcontainers.Container)(nil)
	var _ orders.CreateOpts = uporders.CreateOpts{}
	var _ orders.CreateResult = uporders.CreateResult{}
	var _ *orders.Order = (*uporders.Order)(nil)
	var _ secrets.CreateOpts = upsecrets.CreateOpts{}
	var _ secrets.CreateResult = upsecrets.CreateResult{}
	var _ *secrets.Secret = (*upsecrets.Secret)(nil)
	for _, kind := range []string{"containers", "orders", "secrets"} {
		for _, code := range []int{200, 201, 202} {
			t.Run(fmt.Sprintf("%s/%d", kind, code), func(t *testing.T) {
				c := testcloud.New(t)
				var calls atomic.Int32
				refKey := map[string]string{"containers": "container_ref", "orders": "order_ref", "secrets": "secret_ref"}[kind]
				body := fmt.Sprintf(`{%q:%q}`, refKey, "https://foreign.invalid/"+kind+"/native")
				c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					th.TestMethod(t, r, http.MethodPost)
					if r.URL.Path != "/reverse/barbican/v1/"+kind || r.URL.RawQuery != "" {
						t.Error(r.URL)
					}
					w.Header().Set("X-Native-Proof", "unchanged")
					testcloud.JSON(w, code, body)
				})
				var value any
				var err error
				var expected int
				switch kind {
				case "containers":
					a := containerListFixtures()[0].open(t, c)
					var create func(context.Context, containers.CreateOpts, ...containers.CreateOption) (*containers.Container, error) = a.Create
					value, err = create(context.Background(), containers.CreateOpts{Type: containers.GenericContainer, Name: "input-only"})
					expected = 201
				case "orders":
					a := orderListFixtures()[0].open(t, c)
					var create func(context.Context, orders.CreateOpts, ...orders.CreateOption) (*orders.Order, error) = a.Create
					value, err = create(context.Background(), orders.CreateOpts{Type: orders.KeyOrder, Meta: orders.MetaOpts{Name: "input-only"}})
					expected = 202
				case "secrets":
					a := secretListFixtures()[0].open(t, c)
					var create func(context.Context, secrets.CreateOpts, ...secrets.CreateOption) (*secrets.Secret, error) = a.Create
					value, err = create(context.Background(), secrets.CreateOpts{Name: "input-only"})
					expected = 201
				}
				if calls.Load() != 1 {
					t.Fatal("native create added lookup/refresh", value, err, calls.Load())
				}
				if code == expected {
					if err != nil {
						t.Fatal(value, err)
					}
					switch v := value.(type) {
					case *containers.Container:
						if v == nil || v.Name != "" || v.ContainerRef != "https://foreign.invalid/containers/native" {
							t.Fatal("native model gained input merge", v)
						}
					case *orders.Order:
						if v == nil || v.OrderRef != "https://foreign.invalid/orders/native" || v.Meta.Name != "" {
							t.Fatal("native model gained input merge", v)
						}
					case *secrets.Secret:
						if v == nil || v.Name != "" || v.SecretRef != "https://foreign.invalid/secrets/native" {
							t.Fatal("native model gained input merge", v)
						}
					}
					return
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{expected}) || native.Method != http.MethodPost || string(native.Body) != body || native.ResponseHeader.Get("X-Native-Proof") != "unchanged" {
					t.Fatal("native create status/cause changed", value, err, native)
				}
			})
		}
	}
	t.Run("native container still requires type", func(t *testing.T) {
		c := testcloud.New(t)
		var calls atomic.Int32
		c.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
		value, err := containerListFixtures()[0].open(t, c).Create(context.Background(), containers.CreateOpts{})
		if value != nil || err == nil || calls.Load() != 0 {
			t.Fatal("native builder validation changed", value, err, calls.Load())
		}
	})
}
