package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/compute/v2/keypairs"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

// openstacksdk ef55d7d: _proxy.py:831–840 and keypair.py:24–76.
const computeKeypairCreatePath = "/reverse/nova/v2.1/project/os-keypairs"

func TestComputeKeypairCreateDirtyBodyAliasesAndNativeCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name, body, logicalName, kind, deleted string
		attrs                                  map[string]any
	}{
		{"empty attrs", `{"keypair":{}}`, `null`, `"ssh"`, `null`, nil},
		{"name is body data", `{"keypair":{"name":"../key/name"}}`, `"../key/name"`, `"ssh"`, `null`, map[string]any{"name": "../key/name"}},
		{"empty name and public key", `{"keypair":{"name":"","public_key":""}}`, `""`, `"ssh"`, `null`, map[string]any{"name": "", "public_key": ""}},
		{"null name wins numeric id", `{"keypair":{"name":null,"public_key":null,"type":null,"user_id":null}}`, `null`, `null`, `null`, map[string]any{"id": 42, "name": nil, "public_key": nil, "type": nil, "user_id": nil}},
		{"numeric input id", `{"keypair":{"name":42}}`, `42`, `"ssh"`, `null`, map[string]any{"id": 42}},
		{"literal owner and no new version gate", `{"keypair":{"name":"key","type":"vendor-type","user_id":""}}`, `"key"`, `"vendor-type"`, `null`, map[string]any{"name": "key", "type": "vendor-type", "user_id": ""}},
		{"all eight dirty fields", `{"keypair":{"created_at":{"source":"literal"},"deleted":"false","fingerprint":false,"name":["passive"],"private_key":0,"public_key":{"n":9007199254740993},"type":"","user_id":"owner+team"}}`, `["passive"]`, `""`, `true`, map[string]any{"created_at": map[string]any{"source": "literal"}, "deleted": "false", "fingerprint": false, "name": []any{"passive"}, "private_key": 0, "public_key": map[string]any{"n": json.Number("9007199254740993")}, "type": "", "user_id": "owner+team"}},
		{"deleted alias remains raw on POST", `{"keypair":{"deleted":[]}}`, `null`, `"ssh"`, `false`, map[string]any{"is_deleted": []any{}}},
		{"bulk canonical deleted wins alias", `{"keypair":{"deleted":false,"name":"canonical"}}`, `"canonical"`, `"ssh"`, `false`, map[string]any{"id": "alias", "name": "canonical", "is_deleted": true, "deleted": false, "unknown": func() {}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.1"
			client.MoreHeaders = map[string]string{"X-Source": "selected"}
			cloud.Provider.SetToken("create-live")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				th.TestHeader(t, r, "X-Auth-Token", "create-live")
				th.TestHeader(t, r, "X-Source", "selected")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.1")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.1")
				body, err := io.ReadAll(r.Body)
				if err != nil || string(body) != tc.body || r.URL.Path != computeKeypairCreatePath || r.URL.RawQuery != "" {
					t.Error("dirty attrs, collection route or no-lookup policy changed", string(body), err, r.URL)
				}
				w.Header().Set("X-Keypair-Proof", "actual")
				testcloud.JSON(w, 201, `{"keypair":{}}`)
			})
			value, err := keypairs.New(client).CreateKeypair(context.Background(), keypairs.WithKeypairCreateOptions(keypairs.KeypairCreateOpts{Attributes: tc.attrs}))
			if err != nil || value == nil || value.Resource == nil || value.Wire == nil || calls.Load() != 1 || len(value.Wire.Body) != 0 || string(value.Resource.Body["name"]) != tc.logicalName || string(value.Resource.Body["id"]) != tc.logicalName || string(value.Resource.Body["type"]) != tc.kind || string(value.Resource.Body["is_deleted"]) != tc.deleted || value.StatusCode != 201 || string(value.Envelope) != `{"keypair":{}}` || value.Header.Get("X-Keypair-Proof") != "actual" {
				t.Fatal("view defaults/aliases rewrote dirty body or lost seed", value, err, calls.Load())
			}
			if _, exists := value.Resource.Body["unknown"]; exists {
				t.Fatal("discarded semantic attribute entered Resource", value.Resource)
			}
			if _, exists := value.Resource.Body["deleted"]; exists || client.Microversion != "2.1" || !reflect.DeepEqual(client.MoreHeaders, map[string]string{"X-Source": "selected"}) {
				t.Fatal("view source names or selected client changed", value.Resource, client)
			}
		})
	}
	for _, tc := range []struct {
		name, body string
		status     int
		want       *keypairs.KeyPair
		typed      bool
	}{
		{"native200", `{"keypair":{"name":"actual","type":"ssh","user_id":"response-owner","public_key":"public","private_key":"private","fingerprint":"fp","vendor":17}}`, 200, &keypairs.KeyPair{Name: "actual", Type: "ssh", UserID: "response-owner", PublicKey: "public", PrivateKey: "private", Fingerprint: "fp"}, false},
		{"native201", `{"keypair":{"name":"actual"}}`, 201, &keypairs.KeyPair{Name: "actual"}, false},
		{"native typed partial", `{"keypair":{"name":17,"fingerprint":"kept"}}`, 201, &keypairs.KeyPair{Fingerprint: "kept"}, true},
		{"native null envelope", `{"keypair":null}`, 200, nil, false},
		{"native strict202", `{"keypair":{"name":"unaccepted"}}`, 202, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				th.TestHeader(t, r, "X-Auth-Token", cloud.Provider.TokenID)
				body, err := io.ReadAll(r.Body)
				if err != nil || string(body) != `{"keypair":{"name":"native-name","public_key":"import-public","type":"x509","user_id":"owner+team","vendor":{"enabled":true}}}` || r.URL.Path != computeKeypairCreatePath || r.URL.RawQuery != "" {
					t.Error("native Create ABI changed", string(body), err, r.URL)
				}
				w.Header().Set("X-Keypair-Proof", tc.name)
				testcloud.JSON(w, tc.status, tc.body)
			})
			value, err := keypairs.New(flavorIdentityClient(cloud)).Create(context.Background(), keypairs.CreateOpts{Name: "native-name", UserID: "owner+team", Type: "x509", PublicKey: "import-public"}, keypairs.WithCreateField("vendor", map[string]any{"enabled": true}))
			if !reflect.DeepEqual(value, tc.want) || calls.Load() != 1 || (err != nil) != (tc.typed || tc.status == 202) {
				t.Fatal(value, tc.want, err, calls.Load())
			}
			if tc.typed {
				var typed *json.UnmarshalTypeError
				if !errors.As(err, &typed) {
					t.Fatal("native typed cause lost", err)
				}
			}
			if tc.status == 202 {
				var original gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &original) || original.Actual != 202 || !reflect.DeepEqual(original.Expected, []int{200, 201}) || original.Method != http.MethodPost || original.URL != cloud.Server.URL+computeKeypairCreatePath || string(original.Body) != tc.body || original.ResponseHeader.Get("X-Keypair-Proof") != tc.name {
					t.Fatal("native strict success policy/physical evidence lost", err, original)
				}
			}
		})
	}
	t.Run("native required name and protected field remain preflight", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(500) })
		api := keypairs.New(flavorIdentityClient(cloud))
		if value, err := api.Create(context.Background(), keypairs.CreateOpts{}); err == nil || value != nil {
			t.Fatal("native Name requirement changed", value, err)
		}
		if value, err := api.Create(context.Background(), keypairs.CreateOpts{Name: "native"}, keypairs.WithCreateField("user_id", "overwritten")); !errors.Is(err, resource.ErrInvalidOption) || value != nil || calls.Load() != 0 {
			t.Fatal("native core field protection changed", value, err, calls.Load())
		}
	})
}

func TestComputeKeypairCreateResponseSeedWireAndPassiveErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body, logicalName, kind, deleted string
		status                                 int
		failure, wire                          bool
	}{
		{"empty wrapped keeps seed", `{"keypair":{}}`, `"seed"`, `"ssh"`, `true`, 200, false, true},
		{"nested numeric response id", `{"keypair":{"id":42,"type":null,"deleted":[],"private_key":null,"vendor":{"n":9007199254740993}}}`, `42`, `null`, `false`, 201, false, true},
		{"name null beats response id", `{"keypair":{"id":42,"name":null,"type":"","deleted":null}}`, `null`, `""`, `null`, 203, false, true},
		{"flat response overlays seed", `{"name":"actual","created_at":"now","fingerprint":"actual-fp","public_key":"actual-public","private_key":"actual-private","user_id":"actual-owner","deleted":0,"vendor":17}`, `"actual"`, `"ssh"`, `false`, 300, false, true},
		{"accepted empty", "", `"seed"`, `"ssh"`, `true`, 204, false, false},
		{"accepted malformed", `{"keypair":`, `"seed"`, `"ssh"`, `true`, 201, false, false},
		{"accepted opaque", "accepted non-JSON", `"seed"`, `"ssh"`, `true`, 202, false, false},
		{"valid null root", `null`, "", "", "", 200, true, false},
		{"valid array envelope", `{"keypair":[]}`, "", "", "", 200, true, false},
		{"valid null envelope", `{"keypair":null}`, "", "", "", 200, true, false},
		{"owner forbidden", `{"error":"actual forbidden"}`, "", "", "", 403, true, false},
		{"create missing is failure", `{"error":"actual missing"}`, "", "", "", 404, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, retries atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				if r.URL.Path != computeKeypairCreatePath || r.URL.RawQuery != "" {
					t.Error("create performed lookup/followup", r.URL)
				}
				w.Header().Set("X-Keypair-Proof", tc.name)
				testcloud.JSON(w, tc.status, tc.body)
			})
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			value, err := keypairs.New(flavorIdentityClient(cloud)).CreateKeypair(context.Background(), keypairs.WithKeypairCreateOptions(keypairs.KeypairCreateOpts{Attributes: map[string]any{"name": "seed", "public_key": "seed-public", "private_key": "seed-private", "is_deleted": "false"}}))
			if calls.Load() != 1 || (err != nil) != tc.failure {
				t.Fatal(value, err, calls.Load())
			}
			if tc.status >= 400 {
				var original gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.As(err, &original) || original.Actual != tc.status || len(original.Expected) != 200 || original.Method != http.MethodPost || original.URL != cloud.Server.URL+computeKeypairCreatePath || string(original.Body) != tc.body || original.ResponseHeader.Get("X-Keypair-Proof") != tc.name || retries.Load() != 1 {
					t.Fatal("native HTTP cause or owner error was hidden", value, err, original, retries.Load())
				}
				return
			}
			if value == nil || value.StatusCode != tc.status || value.Header.Get("X-Keypair-Proof") != tc.name || string(value.Envelope) != tc.body || retries.Load() != 0 {
				t.Fatal("accepted receipt lost or response replayed", value, err, retries.Load())
			}
			if tc.failure {
				var proof *resource.ResponseError
				if value.Resource != nil || !errors.As(err, &proof) || proof.StatusCode != tc.status || string(proof.Body) != tc.body || proof.Header.Get("X-Keypair-Proof") != tc.name {
					t.Fatal("parsed nonobject became empty Resource", value, err, proof)
				}
				return
			}
			if value.Resource == nil || (value.Wire != nil) != tc.wire || string(value.Resource.Body["name"]) != tc.logicalName || string(value.Resource.Body["id"]) != tc.logicalName || string(value.Resource.Body["type"]) != tc.kind || string(value.Resource.Body["is_deleted"]) != tc.deleted || value.Resource.StatusCode != tc.status || value.Resource.Header.Get("X-Keypair-Proof") != tc.name {
				t.Fatal("response alias/default/raw bool did not overlay source seed", value)
			}
			if _, present := value.Resource.Body["vendor"]; present {
				t.Fatal("unknown response entered declared Resource", value.Resource)
			}
			if tc.name == "nested numeric response id" && (string(value.Wire.Body["id"]) != `42` || string(value.Wire.Body["vendor"]) != `{"n":9007199254740993}` || string(value.Resource.Body["private_key"]) != `null`) {
				t.Fatal("passive number or response-only wire was normalized", value)
			}
			if tc.name == "flat response overlays seed" {
				for key, want := range map[string]string{"created_at": `"now"`, "fingerprint": `"actual-fp"`, "public_key": `"actual-public"`, "private_key": `"actual-private"`, "user_id": `"actual-owner"`} {
					if string(value.Resource.Body[key]) != want {
						t.Fatal(key, string(value.Resource.Body[key]), want)
					}
				}
			}
			if tc.wire {
				if _, present := value.Wire.Body["type"]; !present && string(value.Resource.Body["type"]) != `"ssh"` {
					t.Fatal("view-only ssh default changed", value)
				}
				if _, present := value.Wire.Body["public_key"]; !present && string(value.Resource.Body["public_key"]) != `"seed-public"` {
					t.Fatal("missing response lost input seed", value)
				}
				originalEnvelope := string(value.Envelope)
				value.Resource.Body["name"][0] = 'x'
				value.Resource.Header.Set("X-Keypair-Proof", "resource changed")
				value.Header.Set("X-Keypair-Proof", "record changed")
				if value.Wire.Header.Get("X-Keypair-Proof") != tc.name || string(value.Envelope) != originalEnvelope {
					t.Fatal("Resource/Header aliases raw receipt", value)
				}
				if name, present := value.Wire.Body["name"]; present && !json.Valid(name) {
					t.Fatal("Resource mutation reached actual wire bytes", value.Wire)
				}
				if len(value.Envelope) != 0 {
					value.Envelope[0] = 'x'
					for _, field := range value.Wire.Body {
						if !json.Valid(field) {
							t.Fatal("record envelope aliases Wire field bytes", value.Wire)
						}
					}
				}
			}
		})
	}
}

func TestComputeKeypairCreateOwnedOptionsSourceAndAcceptedFailures(t *testing.T) {
	t.Run("snapshot helpers bulk reset and individual alias last wins", func(t *testing.T) {
		cloud := testcloud.New(t)
		seed := map[string]any{"name": "seed", "public_key": map[string]any{"n": json.Number("9007199254740993")}, "type": "x509"}
		bulk := keypairs.WithKeypairCreateOptions(keypairs.KeypairCreateOpts{Attributes: seed})
		seed["name"] = "caller changed"
		seed["public_key"].(map[string]any)["n"] = false
		vendor := map[string]any{"enabled": true}
		extension := keypairs.WithKeypairCreateField("vendor", vendor)
		vendor["enabled"] = false
		raw := json.RawMessage(`{"kept":true}`)
		callbackAttribute := map[string]any{"kept": true}
		var calls, callbacks atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestMethod(t, r, http.MethodPost)
			th.TestHeader(t, r, "X-Create-Extension", "kept")
			th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != `{"keypair":{"created_at":{"kept":true},"deleted":false,"name":"last","public_key":{"n":9007199254740993},"trace":{"kept":true},"type":"x509","user_id":"owner","vendor":{"enabled":true}}}` || r.URL.Path != computeKeypairCreatePath || r.URL.RawQuery != "" {
				t.Error("snapshot/bulk/individual ownership changed", string(body), err, r.URL)
			}
			testcloud.JSON(w, 200, `{"keypair":{}}`)
		})
		options := []keypairs.KeypairCreateOption{
			keypairs.WithKeypairCreateName("discarded"), keypairs.WithKeypairCreatePublicKey("discarded"), keypairs.WithKeypairCreateType("discarded"), bulk,
			keypairs.WithKeypairCreateAttribute("id", "last"), keypairs.WithKeypairCreateAttribute("deleted", true), keypairs.WithKeypairCreateAttribute("is_deleted", false), keypairs.WithKeypairCreateUserID("owner"), extension,
			keypairs.WithKeypairCreateHeader("X-Create-Extension", "kept"),
			func(config *request.Config[keypairs.KeypairCreateOpts]) error {
				callbacks.Add(1)
				config.Fields["trace"] = raw
				config.Options.Attributes["created_at"] = callbackAttribute
				return nil
			},
			func(_ *request.Config[keypairs.KeypairCreateOpts]) error {
				callbacks.Add(1)
				raw[0] = 'x'
				callbackAttribute["kept"] = false
				return nil
			},
		}
		value, err := keypairs.New(flavorIdentityClient(cloud)).CreateKeypair(context.Background(), options...)
		if err != nil || value == nil || string(value.Resource.Body["name"]) != `"last"` || string(value.Resource.Body["is_deleted"]) != `false` || string(value.Resource.Body["public_key"]) != `{"n":9007199254740993}` || string(value.Resource.Body["created_at"]) != `{"kept":true}` || calls.Load() != 1 || callbacks.Load() != 2 {
			t.Fatal(value, err, calls.Load(), callbacks.Load())
		}
	})
	for _, mode := range []string{"id field", "name field", "type field", "user_id field", "deleted field", "is_deleted field", "query", "source", "reassigned API", "cancel"} {
		t.Run("preflight "+mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			api := keypairs.New(client)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause, original := errors.New("create canceled"), errors.New("original callback error")
			var calls, later atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(500) })
			var option keypairs.KeypairCreateOption
			switch mode {
			case "query":
				option = request.WithQuery[keypairs.KeypairCreateOpts]("user_id", "wrong owner channel")
			case "source":
				option = func(_ *request.Config[keypairs.KeypairCreateOpts]) error {
					client.ResourceBase = cloud.Server.URL + "/changed/"
					return nil
				}
			case "reassigned API":
				option = func(_ *request.Config[keypairs.KeypairCreateOpts]) error {
					*api = *keypairs.New(flavorIdentityClient(cloud))
					return nil
				}
			case "cancel":
				option = func(_ *request.Config[keypairs.KeypairCreateOpts]) error { cancel(cause); return original }
			default:
				option = keypairs.WithKeypairCreateField(mode[:len(mode)-len(" field")], "override")
			}
			value, err := api.CreateKeypair(ctx, option, func(_ *request.Config[keypairs.KeypairCreateOpts]) error { later.Add(1); return nil })
			if value != nil || err == nil || calls.Load() != 0 {
				t.Fatal("invalid option reached create allocation", value, err, calls.Load())
			}
			if mode == "source" || mode == "reassigned API" || mode == "cancel" {
				if later.Load() != 0 {
					t.Fatal("later callback ran after guarded failure", later.Load())
				}
			}
			if mode == "cancel" {
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, original) {
					t.Fatal("callback/cancellation cause lost", err)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
	for _, mode := range []string{"read404", "close404", "source after response"} {
		t.Run("accepted "+mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			const body = `{"keypair":{"name":"allocated"}}`
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{200}, Method: http.MethodPost, URL: "nested", Body: []byte("original nested404"), ResponseHeader: http.Header{"X-Nested-Proof": {"actual"}}}
			var track *payloadContractTracking
			if mode == "read404" {
				track = payloadContractTrack(cloud, nested, nil)
			} else if mode == "close404" {
				track = payloadContractTrack(cloud, nil, nested)
			} else {
				base := cloud.Provider.HTTPClient.Transport
				if base == nil {
					base = http.DefaultTransport
				}
				cloud.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					response, err := base.RoundTrip(r)
					if err == nil {
						client.ResourceBase = cloud.Server.URL + "/changed/"
					}
					return response, err
				})
			}
			var calls, retries atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				w.Header().Set("X-Keypair-Proof", mode)
				testcloud.JSON(w, 201, body)
			})
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			value, err := keypairs.New(client).CreateKeypair(context.Background(), keypairs.WithKeypairCreateName("requested"))
			var proof *resource.ResponseError
			if value == nil || value.Resource != nil || value.StatusCode != 201 || string(value.Envelope) != body || value.Header.Get("X-Keypair-Proof") != mode || !errors.As(err, &proof) || proof.StatusCode != 201 || string(proof.Body) != body || proof.Header.Get("X-Keypair-Proof") != mode || calls.Load() != 1 || retries.Load() != 0 || errors.Is(err, resource.ErrNotFound) {
				t.Fatal("accepted allocation failure replayed or receipt lost", value, err, proof, calls.Load(), retries.Load())
			}
			if mode == "source after response" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else {
				var original gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &original) || original.Actual != 404 || original.URL != "nested" || string(original.Body) != "original nested404" || original.ResponseHeader.Get("X-Nested-Proof") != "actual" {
					t.Fatal("nested physical cause lost", err, original)
				}
				physical := track.last(t)
				if track.calls.Load() != 1 || physical.reads.Load() == 0 || physical.closes.Load() != 1 {
					t.Fatal(track.calls.Load(), physical)
				}
			}
		})
	}
	t.Run("native retry source mutation cannot resend allocation", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		var calls, retries, callbacks atomic.Int32
		const body = "original create503"
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestMethod(t, r, http.MethodPost)
			w.Header().Set("X-Keypair-Proof", "retry")
			testcloud.JSON(w, 503, body)
		})
		cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, _ error, _ uint) error {
			retries.Add(1)
			client.ResourceBase = cloud.Server.URL + "/changed/"
			return nil
		}
		value, err := keypairs.New(client).CreateKeypair(context.Background(), keypairs.WithKeypairCreateName("key"), func(_ *request.Config[keypairs.KeypairCreateOpts]) error { callbacks.Add(1); return nil })
		var original gophercloud.ErrUnexpectedResponseCode
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &original) || original.Actual != 503 || original.Method != http.MethodPost || original.URL != cloud.Server.URL+computeKeypairCreatePath || string(original.Body) != body || original.ResponseHeader.Get("X-Keypair-Proof") != "retry" || calls.Load() != 1 || retries.Load() != 1 || callbacks.Load() != 1 {
			t.Fatal("source mutation resent allocation/lost original HTTP", value, err, original, calls.Load(), retries.Load(), callbacks.Load())
		}
	})
	t.Run("token freshness does not invalidate fixed source", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		var calls, callbacks atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestMethod(t, r, http.MethodPost)
			th.TestHeader(t, r, "X-Auth-Token", "refreshed-create-token")
			testcloud.JSON(w, 201, `{"keypair":{"name":"actual"}}`)
		})
		value, err := keypairs.New(client).CreateKeypair(context.Background(), keypairs.WithKeypairCreateName("key"), func(_ *request.Config[keypairs.KeypairCreateOpts]) error {
			callbacks.Add(1)
			cloud.Provider.SetToken("refreshed-create-token")
			return nil
		})
		if err != nil || value == nil || string(value.Resource.Body["name"]) != `"actual"` || calls.Load() != 1 || callbacks.Load() != 1 {
			t.Fatal(value, err, calls.Load(), callbacks.Load())
		}
	})
}
