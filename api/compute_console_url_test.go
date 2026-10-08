package api_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/compute/v2/servers"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

// openstacksdk ef55d7d: _proxy.py:2922–2933 -> server.py:954–975.
const computeConsoleURLPath = "/reverse/nova/v2.1/project/servers/server-id/action"

func TestComputeConsoleURLLegacyActionsSelectedSourceAndUnsupportedTypes(t *testing.T) {
	for _, tc := range []struct{ kind, action, version string }{
		{"novnc", "os-getVNCConsole", "2.100"},
		{"xvpvnc", "os-getVNCConsole", "2.100"},
		{"spice-html5", "os-getSPICEConsole", "2.100"},
		{"spice-direct", "os-getSPICEConsole", "2.100"},
		{"rdp-html5", "os-getRDPConsole", "2.100"},
		{"serial", "os-getSerialConsole", "2.100"},
		{"spice-direct", "os-getSPICEConsole", "2.5"},
		{"novnc", "os-getVNCConsole", ""},
	} {
		t.Run(tc.kind+"/"+tc.version, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = tc.version
			client.MoreHeaders = map[string]string{"X-Source": "selected"}
			cloud.Provider.SetToken("live-console-url-token")
			api := servers.New(client)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				th.TestHeader(t, r, "X-Auth-Token", "live-console-url-token")
				th.TestHeader(t, r, "X-Source", "selected")
				if tc.version == "" {
					th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
					th.TestHeaderUnset(t, r, "OpenStack-API-Version")
				} else {
					th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", tc.version)
					th.TestHeader(t, r, "OpenStack-API-Version", "compute "+tc.version)
				}
				th.TestHeader(t, r, "Accept", "")
				body, err := io.ReadAll(r.Body)
				want := `{"` + tc.action + `":{"type":"` + tc.kind + `"}}`
				if err != nil || string(body) != want || r.URL.Path != computeConsoleURLPath || r.URL.RawQuery != "" {
					t.Error("legacy action changed or performed lookup/discovery/modern fallback", string(body), want, err, r.URL)
				}
				testcloud.JSON(w, 200, `{"console":{"type":"actual-response","url":"literal"}}`)
			})
			value, err := api.ConsoleURL(context.Background(), "server-id", tc.kind)
			if err != nil || string(value) != `{"type":"actual-response","url":"literal"}` || calls.Load() != 1 || api.RawClient() != client || client.ProviderClient != cloud.Provider || client.Microversion != tc.version || !reflect.DeepEqual(client.MoreHeaders, map[string]string{"X-Source": "selected"}) {
				t.Fatal(value, err, calls.Load(), client)
			}
		})
	}
	t.Run("exact unsupported types never request", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
		api := servers.New(flavorIdentityClient(cloud))
		for _, kind := range []string{"", "webmks", "unknown", "NOVNC", " novnc"} {
			value, err := api.ConsoleURL(context.Background(), "server-id", kind)
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(kind, value, err, calls.Load())
			}
		}
	})
}

func TestComputeConsoleURLRawValueAndResponseEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		code             int
		failure          bool
	}{
		{"object and extensions", `{"console":{"type":"foreign","url":"벤더://literal\n\u0000","extra":9007199254740993},"ignored":17}`, `{"type":"foreign","url":"벤더://literal\n\u0000","extra":9007199254740993}`, 200, false},
		{"empty object", `{"console":{}}`, `{}`, 200, false},
		{"null is a returned value", `{"console":null}`, `null`, 200, false},
		{"array is a returned value", `{"console":[null,false,9007199254740993]}`, `[null,false,9007199254740993]`, 201, false},
		{"string is a returned value", `{"console":"literal"}`, `"literal"`, 203, false},
		{"false is a returned value", `{"console":false}`, `false`, 200, false},
		{"zero is a returned value", `{"console":0}`, `0`, 200, false},
		{"300 no Location", `{"console":{"url":"accepted"}}`, `{"url":"accepted"}`, 300, false},
		{"missing console", `{}`, "", 200, true},
		{"exact key case", `{"CONSOLE":{"url":"not-selected"}}`, "", 200, true},
		{"null root", `null`, "", 200, true},
		{"array root", `[]`, "", 200, true},
		{"string root", `"root"`, "", 200, true},
		{"empty204", "", "", 204, true},
		{"opaque accepted", "opaque", "", 202, true},
		{"malformed accepted", `{"console":`, "", 201, true},
		{"invalid UTF8", "{\"console\":\"\xff\"}", "", 200, true},
		{"bad request", `{"error":"original"}`, "", 400, true},
		{"forbidden", `{"error":"original"}`, "", 403, true},
		{"not found does not fallback", `{"error":"original"}`, "", 404, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, retries atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				th.TestHeader(t, r, "X-Auth-Token", "test-token")
				if r.URL.Path != computeConsoleURLPath || r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				w.Header().Set("X-Console-Proof", tc.name)
				testcloud.JSON(w, tc.code, tc.body)
			})
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			value, err := servers.New(flavorIdentityClient(cloud)).ConsoleURL(context.Background(), "server-id", "novnc")
			if string(value) != tc.want || (err != nil) != tc.failure || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
			if !tc.failure {
				if value == nil || retries.Load() != 0 {
					t.Fatal("accepted console value lost or replayed", value, retries.Load())
				}
				return
			}
			var operation *resource.OperationError
			if value != nil || !errors.As(err, &operation) || operation.Operation != "ConsoleURL" || operation.Resource != "servers" {
				t.Fatal("operation context/result lost", value, err)
			}
			var proof *resource.ResponseError
			if tc.code >= 400 {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || errors.As(err, &proof) || native.Actual != tc.code || native.Method != http.MethodPost || native.URL != cloud.Server.URL+computeConsoleURLPath || string(native.Body) != tc.body || native.ResponseHeader.Get("X-Console-Proof") != tc.name || len(native.Expected) != 200 || native.Expected[0] != 200 || native.Expected[199] != 399 || retries.Load() != 1 {
					t.Fatal("native rejection evidence changed", err, native, retries.Load())
				}
			} else if !errors.As(err, &proof) || proof.StatusCode != tc.code || proof.Header.Get("X-Console-Proof") != tc.name || string(proof.Body) != tc.body || retries.Load() != 0 {
				t.Fatal("accepted decode evidence lost/replayed", err, proof, retries.Load())
			}
			if (tc.name == "missing console" || tc.name == "exact key case") && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("missing console was not classified as invalid response", err)
			}
		})
	}
}

func TestComputeConsoleURLSourceContextAndAcceptedFailures(t *testing.T) {
	t.Run("preflight", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
		for _, mode := range []string{"nil API", "nil client", "nil context", "invalid ID", "pre-canceled", "outer guard"} {
			t.Run(mode, func(t *testing.T) {
				api := servers.New(flavorIdentityClient(cloud))
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("console URL preflight cause")
				id := "server-id"
				switch mode {
				case "nil API":
					api = nil
				case "nil client":
					api = servers.New(nil)
				case "nil context":
					ctx = nil
				case "invalid ID":
					id = "other/server"
				case "pre-canceled":
					cancel(cause)
				case "outer guard":
					ctx = rest.WithOperationGuard(ctx, func(context.Context) error { return cause })
				}
				value, err := api.ConsoleURL(ctx, id, "novnc")
				if value != nil || err == nil || calls.Load() != 0 {
					t.Fatal("preflight reached HTTP", value, err, calls.Load())
				}
				if mode == "pre-canceled" {
					if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
						t.Fatal(err)
					}
				} else if mode == "outer guard" {
					if !errors.Is(err, cause) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			})
		}
	})
	for _, mode := range []string{"read", "close404", "source", "context"} {
		t.Run("accepted "+mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			const body = `{"console":{"type":"actual","url":"accepted"}}`
			cause := errors.New("console URL original failure")
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{200}, Method: http.MethodPost, URL: "nested", Body: []byte("nested404")}
			var track *payloadContractTracking
			if mode == "read" {
				track = payloadContractTrack(cloud, cause, nil)
			}
			if mode == "close404" {
				track = payloadContractTrack(cloud, nil, nested)
			}
			if mode == "source" || mode == "context" {
				base := cloud.Provider.HTTPClient.Transport
				if base == nil {
					base = http.DefaultTransport
				}
				cloud.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
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
					if mode == "source" {
						client.ResourceBase = cloud.Server.URL + "/changed/"
					} else {
						cancel(cause)
					}
					return response, nil
				})
			}
			var calls, retries atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				th.TestHeader(t, r, "X-Auth-Token", "test-token")
				w.Header().Set("X-Console-Proof", mode)
				testcloud.JSON(w, 203, body)
			})
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			value, err := servers.New(client).ConsoleURL(ctx, "server-id", "novnc")
			var proof *resource.ResponseError
			if value != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Console-Proof") != mode || calls.Load() != 1 || retries.Load() != 0 {
				t.Fatal("accepted terminal failure lost receipt or replayed", value, err, proof, calls.Load(), retries.Load())
			}
			if mode == "read" && !errors.Is(err, cause) || mode == "close404" && (!gophercloud.ResponseCodeIs(err, 404) || errors.Is(err, resource.ErrNotFound)) || mode == "source" && !errors.Is(err, resource.ErrInvalidOption) || mode == "context" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
				t.Fatal("original failure cause lost", err)
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
