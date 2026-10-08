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

	"github.com/JSYoo5B/gophercloudsdk/compute/v2/remoteconsoles"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

// openstacksdk ef55d7d: _proxy.py:2903–2920, server_remote_console.py:18–77.
const computeRemoteConsolePath = "/reverse/nova/v2.1/project/servers/server-id/remote-consoles"

func TestComputeRemoteConsoleRequestDefaultsDerivationAndSeedOwnership(t *testing.T) {
	tests := []struct {
		name, body, protocol, kind, url string
		options                         []remoteconsoles.ConsoleCreateOption
	}{
		{"empty defaults", `{"remote_console":{}}`, `null`, `null`, `null`, nil},
		{"null protocol", `{"remote_console":{"protocol":"vnc","type":"novnc"}}`, `null`, `"novnc"`, `null`, []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateType("novnc"), remoteconsoles.WithConsoleCreateProtocolValue(request.Null[string]())}},
		{"empty protocol seed", `{"remote_console":{"protocol":"vnc","type":"novnc"}}`, `""`, `"novnc"`, `null`, []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateType("novnc"), remoteconsoles.WithConsoleCreateProtocol("")}},
		{"unknown type", `{"remote_console":{"protocol":null,"type":"vendor-console"}}`, `null`, `"vendor-console"`, `null`, []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateType("vendor-console")}},
		{"explicit protocol preserved", `{"remote_console":{"protocol":"inconsistent","type":"novnc"}}`, `"inconsistent"`, `"novnc"`, `null`, []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateProtocol("inconsistent"), remoteconsoles.WithConsoleCreateType("novnc")}},
		{"null type no derivation", `{"remote_console":{"protocol":"","type":null}}`, `""`, `null`, `null`, []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateTypeValue(request.Null[string]()), remoteconsoles.WithConsoleCreateProtocol("")}},
		{"empty type no derivation", `{"remote_console":{"type":""}}`, `null`, `""`, `null`, []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateType("")}},
		{"URL body attribute", `{"remote_console":{"url":"passive://seed"}}`, `null`, `null`, `"passive://seed"`, []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateURL("passive://seed")}},
		{"null URL", `{"remote_console":{"url":null}}`, `null`, `null`, `null`, []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateURLValue(request.Null[string]())}},
		{"unset restores omission", `{"remote_console":{"protocol":"vnc","type":"novnc"}}`, `null`, `"novnc"`, `null`, []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateProtocol("discarded"), remoteconsoles.WithConsoleCreateProtocolValue(request.Optional[string]{}), remoteconsoles.WithConsoleCreateType("novnc")}},
	}
	for kind, protocol := range map[string]string{"novnc": "vnc", "xvpvnc": "vnc", "spice-html5": "spice", "spice-direct": "spice", "rdp-html5": "rdp", "serial": "serial", "webmks": "mks"} {
		tests = append(tests, struct {
			name, body, protocol, kind, url string
			options                         []remoteconsoles.ConsoleCreateOption
		}{
			"derive " + kind, `{"remote_console":{"protocol":"` + protocol + `","type":"` + kind + `"}}`, `null`, `"` + kind + `"`, `null`, []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateType(kind)},
		})
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.100"
			client.MoreHeaders = map[string]string{"X-Source": "shared"}
			cloud.Provider.SetToken("remote-live-token")
			api := remoteconsoles.New(client)
			var calls, callbacks atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				th.TestHeader(t, r, "X-Auth-Token", "remote-live-token")
				th.TestHeader(t, r, "X-Source", "shared")
				th.TestHeader(t, r, "X-Call", "owned")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.100")
				body, err := io.ReadAll(r.Body)
				if err != nil || string(body) != tc.body || r.URL.Path != computeRemoteConsolePath || r.URL.RawQuery != "" {
					t.Error(string(body), tc.body, err, r.URL)
				}
				w.Header().Set("X-Console-Proof", "actual")
				testcloud.JSON(w, 200, `{"remote_console":{}}`)
			})
			options := append([]remoteconsoles.ConsoleCreateOption(nil), tc.options...)
			options = append(options, remoteconsoles.WithConsoleCreateHeader("X-Call", "owned"), func(_ *request.Config[remoteconsoles.ConsoleCreateOpts]) error { callbacks.Add(1); return nil })
			for repeat := 0; repeat < 2; repeat++ {
				value, err := api.CreateConsole(context.Background(), "server-id", options...)
				if err != nil || value == nil || value.Resource == nil || value.Wire == nil || string(value.Resource.Body["protocol"]) != tc.protocol || string(value.Resource.Body["type"]) != tc.kind || string(value.Resource.Body["url"]) != tc.url || string(value.Resource.Body["server_id"]) != `"server-id"` || len(value.Wire.Body) != 0 || len(value.Resource.Body) != 4 || value.StatusCode != 200 || value.Header.Get("X-Console-Proof") != "actual" || string(value.Envelope) != `{"remote_console":{}}` {
					t.Fatal("transmitted derivation replaced original Resource seed", value, err)
				}
			}
			if calls.Load() != 2 || callbacks.Load() != 2 || api.RawClient() != client || client.ProviderClient != cloud.Provider || client.Microversion != "2.100" || !reflect.DeepEqual(client.MoreHeaders, map[string]string{"X-Source": "shared"}) {
				t.Fatal(calls.Load(), callbacks.Load(), client)
			}
		})
	}
	t.Run("bulk and extension snapshots between callbacks", func(t *testing.T) {
		cloud := testcloud.New(t)
		attrs := remoteconsoles.ConsoleCreateOpts{Type: request.Present("novnc"), URL: request.Present("original")}
		bulk := remoteconsoles.WithConsoleCreateOptions(attrs)
		attrs.Type = request.Present("changed")
		vendor := map[string]any{"n": json.Number("9007199254740993")}
		extension := remoteconsoles.WithConsoleCreateField("vendor", vendor)
		vendor["n"] = false
		raw := json.RawMessage(`{"kept":true}`)
		var calls, callbacks atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != `{"remote_console":{"protocol":"vnc","trace":{"kept":true},"type":"novnc","url":"original","vendor":{"n":9007199254740993}}}` || r.URL.Path != computeRemoteConsolePath {
				t.Error(string(body), err, r.URL)
			}
			testcloud.JSON(w, 200, `{"remote_console":{"url":null}}`)
		})
		value, err := remoteconsoles.New(flavorIdentityClient(cloud)).CreateConsole(context.Background(), "server-id", extension, bulk, func(config *request.Config[remoteconsoles.ConsoleCreateOpts]) error {
			callbacks.Add(1)
			config.Fields["trace"] = raw
			return nil
		}, func(_ *request.Config[remoteconsoles.ConsoleCreateOpts]) error {
			callbacks.Add(1)
			raw[0] = 'x'
			return nil
		})
		if err != nil || value == nil || string(value.Resource.Body["type"]) != `"novnc"` || string(value.Resource.Body["protocol"]) != `null` || string(value.Resource.Body["url"]) != `null` || calls.Load() != 1 || callbacks.Load() != 2 {
			t.Fatal(value, err, calls.Load(), callbacks.Load())
		}
		if _, exists := value.Resource.Body["vendor"]; exists {
			t.Fatal("explicit wire extension became a declared Resource field", value.Resource)
		}
	})
}

func TestComputeRemoteConsoleWinningTypeSelectedVersionGates(t *testing.T) {
	for _, tc := range []struct {
		name, version string
		options       []remoteconsoles.ConsoleCreateOption
		allowed       bool
	}{
		{"webmks unselected", "", []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateType("webmks")}, false},
		{"webmks below", "2.7", []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateType("webmks")}, false},
		{"webmks boundary", "2.8", []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateType("webmks")}, true},
		{"spice-direct below explicit protocol", "2.98", []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateType("spice-direct"), remoteconsoles.WithConsoleCreateProtocol("literal")}, false},
		{"spice-direct boundary", "2.99", []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateType("spice-direct")}, true},
		{"numeric 2.100", "2.100", []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateType("spice-direct")}, true},
		{"finite major latest", "2.latest", []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateType("spice-direct")}, true},
		{"symbolic latest cannot prove gate", "latest", []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateType("webmks")}, false},
		{"malformed selected gate", "2.invalid", []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateType("webmks")}, false},
		{"no local 2.6 prerequisite", "2.5", []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateType("novnc")}, true},
		{"winning type only", "2.5", []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateType("webmks"), remoteconsoles.WithConsoleCreateType("novnc")}, true},
		{"bulk clears earlier gate", "2.7", []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateType("webmks"), remoteconsoles.WithConsoleCreateOptions(remoteconsoles.ConsoleCreateOpts{})}, true},
		{"null clears earlier gate", "", []remoteconsoles.ConsoleCreateOption{remoteconsoles.WithConsoleCreateType("spice-direct"), remoteconsoles.WithConsoleCreateTypeValue(request.Null[string]())}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = tc.version
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				if tc.version == "" {
					th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
				} else {
					th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", tc.version)
				}
				if r.URL.Path != computeRemoteConsolePath || r.URL.RawQuery != "" {
					t.Error("lookup/discovery/legacy request", r.URL)
				}
				testcloud.JSON(w, 201, `{"remote_console":{"url":"literal"}}`)
			})
			value, err := remoteconsoles.New(client).CreateConsole(context.Background(), "server-id", tc.options...)
			if client.Microversion != tc.version {
				t.Fatal("selected microversion was upgraded", client.Microversion)
			}
			if tc.allowed {
				if err != nil || value == nil || value.Resource == nil || calls.Load() != 1 {
					t.Fatal(value, err, calls.Load())
				}
			} else if value != nil || !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 0 {
				t.Fatal("winning type gate did not fail before HTTP", value, err, calls.Load())
			}
		})
	}
}

func TestComputeRemoteConsoleResponseEvidenceSourceAndNativeCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name, body          string
		code                int
		failure, wire       bool
		protocol, kind, url string
	}{
		{"nested200", `{"remote_console":{"protocol":null,"type":false,"url":"remote://literal","server_id":"foreign","vendor":{"n":9007199254740993}}}`, 200, false, true, `null`, `false`, `"remote://literal"`},
		{"nested203 null overrides", `{"remote_console":{"url":null}}`, 203, false, true, `"chosen"`, `"novnc"`, `null`},
		{"flat300 no Location", `{"protocol":"actual","type":null,"url":"flat","vendor":17}`, 300, false, true, `"actual"`, `null`, `"flat"`},
		{"empty200 tolerated", "", 200, false, false, `"chosen"`, `"novnc"`, `"seed"`},
		{"empty204 tolerated", "", 204, false, false, `"chosen"`, `"novnc"`, `"seed"`},
		{"opaque202 tolerated", "accepted opaque", 202, false, false, `"chosen"`, `"novnc"`, `"seed"`},
		{"malformed201 tolerated", `{"remote_console":`, 201, false, false, `"chosen"`, `"novnc"`, `"seed"`},
		{"invalid UTF8 tolerated", "{\"url\":\"\xff\"}", 201, false, false, `"chosen"`, `"novnc"`, `"seed"`},
		{"parsed null root", `null`, 200, true, false, "", "", ""},
		{"parsed array root", `[]`, 200, true, false, "", "", ""},
		{"parsed string root", `"parsed"`, 200, true, false, "", "", ""},
		{"present null envelope", `{"remote_console":null}`, 200, true, false, "", "", ""},
		{"present array envelope", `{"remote_console":[]}`, 200, true, false, "", "", ""},
		{"forbidden", `{"error":"original"}`, 403, true, false, "", "", ""},
		{"not found", `{"error":"original"}`, 404, true, false, "", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, retries atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				if r.URL.Path != computeRemoteConsolePath || r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				w.Header().Set("X-Console-Proof", tc.name)
				testcloud.JSON(w, tc.code, tc.body)
			})
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			value, err := remoteconsoles.New(flavorIdentityClient(cloud)).CreateConsole(context.Background(), "server-id", remoteconsoles.WithConsoleCreateProtocol("chosen"), remoteconsoles.WithConsoleCreateType("novnc"), remoteconsoles.WithConsoleCreateURL("seed"))
			if calls.Load() != 1 || (err != nil) != tc.failure {
				t.Fatal(value, err, calls.Load())
			}
			if tc.code >= 400 {
				var native gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.As(err, &native) || native.Actual != tc.code || native.Method != http.MethodPost || native.URL != cloud.Server.URL+computeRemoteConsolePath || string(native.Body) != tc.body || native.ResponseHeader.Get("X-Console-Proof") != tc.name || len(native.Expected) != 200 || retries.Load() != 1 {
					t.Fatal("native error/proof lost", value, err, native, retries.Load())
				}
				return
			}
			if value == nil || value.StatusCode != tc.code || string(value.Envelope) != tc.body || value.Header.Get("X-Console-Proof") != tc.name || retries.Load() != 0 {
				t.Fatal("accepted proof lost/replayed", value, err, retries.Load())
			}
			if tc.failure {
				var proof *resource.ResponseError
				if value.Resource != nil || !errors.As(err, &proof) || proof.StatusCode != tc.code || string(proof.Body) != tc.body || proof.Header.Get("X-Console-Proof") != tc.name {
					t.Fatal(value, err, proof)
				}
				return
			}
			if value.Resource == nil || (value.Wire != nil) != tc.wire || string(value.Resource.Body["protocol"]) != tc.protocol || string(value.Resource.Body["type"]) != tc.kind || string(value.Resource.Body["url"]) != tc.url || string(value.Resource.Body["server_id"]) != `"server-id"` || value.Resource.StatusCode != tc.code || value.Resource.Header.Get("X-Console-Proof") != tc.name {
				t.Fatal("response did not override seed/null/default/URI correctly", value)
			}
			if _, exists := value.Resource.Body["vendor"]; exists {
				t.Fatal("unknown response became declared field", value.Resource)
			}
			if tc.wire {
				if value.Wire.StatusCode != tc.code || value.Wire.Header.Get("X-Console-Proof") != tc.name {
					t.Fatal(value.Wire)
				}
				if tc.name == "nested200" && (string(value.Wire.Body["server_id"]) != `"foreign"` || string(value.Wire.Body["vendor"]) != `{"n":9007199254740993}`) {
					t.Fatal("wire parent or precise extension changed", value.Wire)
				}
				if tc.name == "nested203 null overrides" {
					if _, present := value.Wire.Body["protocol"]; present {
						t.Fatal("seed entered Wire", value.Wire)
					}
				}
				value.Header.Set("X-Console-Proof", "changed")
				value.Resource.Header.Set("X-Console-Proof", "changed")
				value.Resource.Body["url"][0] = 'x'
				if value.Wire.Header.Get("X-Console-Proof") != tc.name || !json.Valid(value.Wire.Body["url"]) {
					t.Fatal("Resource/Header aliases Wire", value.Wire)
				}
			}
		})
	}
	t.Run("native Create ABI required inputs and strict200", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := remoteconsoles.New(flavorIdentityClient(cloud))
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			index := calls.Add(1)
			th.TestMethod(t, r, http.MethodPost)
			th.TestHeader(t, r, "X-Auth-Token", cloud.Provider.TokenID)
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != `{"remote_console":{"protocol":"vnc","type":"novnc","url":"native-request-extension"}}` || r.URL.Path != computeRemoteConsolePath || r.URL.RawQuery != "" {
				t.Error(string(body), err, r.URL)
			}
			w.Header().Set("X-Console-Proof", "native")
			code := 200
			if index == 2 {
				code = 203
			}
			response := `{"remote_console":{"protocol":"vnc","type":"novnc","url":"native","vendor":17}}`
			if index == 3 {
				response = `{"remote_console":{"protocol":"vnc","type":"novnc","url":17}}`
			}
			if index == 4 {
				response = `{"remote_console":null}`
			}
			testcloud.JSON(w, code, response)
		})
		for _, missing := range []remoteconsoles.CreateOpts{{}, {Type: remoteconsoles.ConsoleTypeNoVNC}, {Protocol: remoteconsoles.ConsoleProtocolVNC}} {
			if value, err := api.Create(context.Background(), "server-id", missing); value != nil || err == nil || calls.Load() != 0 {
				t.Fatal("native required fields changed", value, err, calls.Load())
			}
		}
		opts := remoteconsoles.CreateOpts{Protocol: remoteconsoles.ConsoleProtocolVNC, Type: remoteconsoles.ConsoleTypeNoVNC}
		option := remoteconsoles.WithCreateField("url", "native-request-extension")
		value, err := api.Create(context.Background(), "server-id", opts, option)
		if err != nil || value == nil || value.Protocol != "vnc" || value.Type != "novnc" || value.URL != "native" || calls.Load() != 1 {
			t.Fatal(value, err, calls.Load())
		}
		value, err = api.Create(context.Background(), "server-id", opts, option)
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || !errors.As(err, &native) || native.Actual != 203 || !reflect.DeepEqual(native.Expected, []int{200}) || native.Method != http.MethodPost || native.URL != cloud.Server.URL+computeRemoteConsolePath || native.ResponseHeader.Get("X-Console-Proof") != "native" || string(native.Body) != `{"remote_console":{"protocol":"vnc","type":"novnc","url":"native","vendor":17}}` || calls.Load() != 2 {
			t.Fatal("native accepted status or original evidence changed", value, err, native, calls.Load())
		}
		value, err = api.Create(context.Background(), "server-id", opts, option)
		var typed *json.UnmarshalTypeError
		if value == nil || value.Protocol != "vnc" || value.Type != "novnc" || value.URL != "" || !errors.As(err, &typed) || calls.Load() != 3 {
			t.Fatal("native typed partial projection changed", value, err, calls.Load())
		}
		value, err = api.Create(context.Background(), "server-id", opts, option)
		if value != nil || err != nil || calls.Load() != 4 {
			t.Fatal("native null envelope policy changed", value, err, calls.Load())
		}
	})
	t.Run("preflight core source and cancellation", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
		for _, mode := range []string{"protocol field", "type field", "url field", "parent field", "nil option", "query", "source", "cancel"} {
			t.Run(mode, func(t *testing.T) {
				client := flavorIdentityClient(cloud)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause, original := errors.New("remote cancel cause"), errors.New("original option error")
				var option remoteconsoles.ConsoleCreateOption
				var later atomic.Int32
				switch mode {
				case "protocol field":
					option = remoteconsoles.WithConsoleCreateField("protocol", "vnc")
				case "type field":
					option = remoteconsoles.WithConsoleCreateField("type", "novnc")
				case "url field":
					option = remoteconsoles.WithConsoleCreateField("url", "value")
				case "parent field":
					option = remoteconsoles.WithConsoleCreateField("server_id", "other")
				case "query":
					option = request.WithQuery[remoteconsoles.ConsoleCreateOpts]("limit", "1")
				case "source":
					option = func(_ *request.Config[remoteconsoles.ConsoleCreateOpts]) error {
						client.ResourceBase = cloud.Server.URL + "/changed/"
						return nil
					}
				case "cancel":
					option = func(_ *request.Config[remoteconsoles.ConsoleCreateOpts]) error { cancel(cause); return original }
				}
				value, err := remoteconsoles.New(client).CreateConsole(ctx, "server-id", option, func(_ *request.Config[remoteconsoles.ConsoleCreateOpts]) error { later.Add(1); return nil })
				if value != nil || err == nil || calls.Load() != 0 {
					t.Fatal("invalid preflight reached HTTP", value, err, calls.Load())
				}
				if mode == "source" || mode == "cancel" || mode == "nil option" {
					if later.Load() != 0 {
						t.Fatal("later callback ran after source/context/option failure", later.Load())
					}
				}
				if mode == "cancel" {
					if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, original) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			})
		}
	})
	for _, mode := range []string{"read", "close404", "source after response"} {
		t.Run("accepted "+mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			const body = `{"remote_console":{"url":"accepted"}}`
			cause := errors.New("original remote response read")
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{200}, Method: http.MethodPost, URL: "nested", Body: []byte("nested404")}
			var track *payloadContractTracking
			if mode == "read" {
				track = payloadContractTrack(cloud, cause, nil)
			}
			if mode == "close404" {
				track = payloadContractTrack(cloud, nil, nested)
			}
			if mode == "source after response" {
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
				w.Header().Set("X-Console-Proof", mode)
				testcloud.JSON(w, 201, body)
			})
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			value, err := remoteconsoles.New(client).CreateConsole(context.Background(), "server-id", remoteconsoles.WithConsoleCreateType("novnc"))
			var proof *resource.ResponseError
			if value == nil || value.Resource != nil || value.StatusCode != 201 || string(value.Envelope) != body || value.Header.Get("X-Console-Proof") != mode || !errors.As(err, &proof) || proof.StatusCode != 201 || string(proof.Body) != body || proof.Header.Get("X-Console-Proof") != mode || calls.Load() != 1 || retries.Load() != 0 {
				t.Fatal("accepted failure lost partial receipt/replayed", value, err, proof, calls.Load(), retries.Load())
			}
			if mode == "read" && !errors.Is(err, cause) || mode == "close404" && (!gophercloud.ResponseCodeIs(err, 404) || errors.Is(err, resource.ErrNotFound)) || mode == "source after response" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if track != nil {
				physical := track.last(t)
				if track.calls.Load() != 1 || physical.reads.Load() == 0 || physical.closes.Load() != 1 {
					t.Fatal(track.calls.Load(), physical)
				}
			}
		})
	}
}
