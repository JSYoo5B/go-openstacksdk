package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

// openstacksdk ef55d7d: _proxy.py:2946–2961 -> server.py:938–952.
const computeConsoleOutputPath = "/reverse/nova/v2.1/project/servers/server-id/action"

func TestComputeConsoleOutputLiteralLengthOwnedOptionsAndNativeCompatibility(t *testing.T) {
	length := 0
	frozen := servers.WithConsoleOutputOptions(servers.ConsoleOutputOpts{Length: &length})
	length = 91
	vendor := map[string]any{"values": []string{"original"}}
	extension := servers.WithConsoleOutputField("vendor", vendor)
	vendor["values"].([]string)[0] = "changed"
	for _, tc := range []struct {
		name    string
		options []servers.ConsoleOutputOption
		want    string
	}{
		{"default", nil, `{"os-getConsoleOutput":{}}`},
		{"zero", []servers.ConsoleOutputOption{servers.WithConsoleOutputLength(0)}, `{"os-getConsoleOutput":{"length":0}}`},
		{"positive", []servers.ConsoleOutputOption{servers.WithConsoleOutputLength(5)}, `{"os-getConsoleOutput":{"length":5}}`},
		{"negative forwarded", []servers.ConsoleOutputOption{servers.WithConsoleOutputLength(-3)}, `{"os-getConsoleOutput":{"length":-3}}`},
		{"pointer snapshot", []servers.ConsoleOutputOption{frozen}, `{"os-getConsoleOutput":{"length":0}}`},
		{"last individual wins", []servers.ConsoleOutputOption{servers.WithConsoleOutputLength(5), servers.WithConsoleOutputLength(0)}, `{"os-getConsoleOutput":{"length":0}}`},
		{"bulk restores omission", []servers.ConsoleOutputOption{servers.WithConsoleOutputLength(0), servers.WithConsoleOutputOptions(servers.ConsoleOutputOpts{})}, `{"os-getConsoleOutput":{}}`},
		{"extension snapshot", []servers.ConsoleOutputOption{extension}, `{"os-getConsoleOutput":{"vendor":{"values":["original"]}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.87"
			client.MoreHeaders = map[string]string{"X-Source": "selected"}
			cloud.Provider.SetToken("live-console-token")
			api := servers.New(client)
			var calls, callbacks atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				th.TestHeader(t, r, "X-Auth-Token", "live-console-token")
				th.TestHeader(t, r, "X-Source", "selected")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.87")
				if r.URL.Path != computeConsoleOutputPath || r.URL.RawQuery != "" || r.Header.Get("Accept") != "" {
					t.Error("action route/query/Accept changed", r.URL, r.Header)
				}
				body, err := io.ReadAll(r.Body)
				if err != nil || string(body) != tc.want {
					t.Error("literal optional length or extension changed", string(body), tc.want, err)
				}
				testcloud.JSON(w, 200, `{"output":"콘솔\n\u0000text","vendor":17}`)
			})
			options := append([]servers.ConsoleOutputOption(nil), tc.options...)
			options = append(options, func(_ *request.Config[servers.ConsoleOutputOpts]) error { callbacks.Add(1); return nil })
			// Reuse the helper: its pointed value and raw extension remain owned.
			for repeat := 0; repeat < 2; repeat++ {
				value, err := api.ConsoleOutput(context.Background(), "server-id", options...)
				if err != nil || value != "콘솔\n\x00text" {
					t.Fatal(value, err)
				}
			}
			if calls.Load() != 2 || callbacks.Load() != 2 || api.RawClient() != client || client.ProviderClient != cloud.Provider || client.Microversion != "2.87" || !reflect.DeepEqual(client.MoreHeaders, map[string]string{"X-Source": "selected"}) {
				t.Fatal("options/source ownership or callback count changed", calls.Load(), callbacks.Load(), client)
			}
		})
	}
	t.Run("custom callback values freeze between options", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls, callbacks atomic.Int32
		length := 0
		raw := json.RawMessage(`{"owned":true}`)
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != `{"os-getConsoleOutput":{"length":0,"vendor":{"owned":true}}}` || r.URL.Path != computeConsoleOutputPath {
				t.Error(string(body), err, r.URL)
			}
			testcloud.JSON(w, 200, `{"output":"owned"}`)
		})
		value, err := servers.New(flavorIdentityClient(cloud)).ConsoleOutput(context.Background(), "server-id", func(config *request.Config[servers.ConsoleOutputOpts]) error {
			callbacks.Add(1)
			config.Options.Length = &length
			config.Fields["vendor"] = raw
			return nil
		}, func(_ *request.Config[servers.ConsoleOutputOpts]) error {
			callbacks.Add(1)
			length, raw[0] = 9, 'x'
			return nil
		})
		if err != nil || value != "owned" || calls.Load() != 1 || callbacks.Load() != 2 {
			t.Fatal(value, err, calls.Load(), callbacks.Load())
		}
	})
	t.Run("native signature and zero omission remain unchanged", func(t *testing.T) {
		cloud := testcloud.New(t)
		api := servers.New(flavorIdentityClient(cloud))
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			index := calls.Add(1)
			body, err := io.ReadAll(r.Body)
			want := `{"os-getConsoleOutput":{"length":0}}`
			if index == 2 {
				want = `{"os-getConsoleOutput":{}}`
			}
			if err != nil || string(body) != want || r.URL.Path != computeConsoleOutputPath || r.Method != http.MethodPost {
				t.Error(index, string(body), want, err, r.URL)
			}
			testcloud.JSON(w, 200, `{"output":"same projection"}`)
		})
		owned, err := api.ConsoleOutput(context.Background(), "server-id", servers.WithConsoleOutputLength(0))
		var nativeOptions servers.ShowConsoleOutputOpts
		native, nativeErr := api.ShowConsoleOutput(context.Background(), "server-id", nativeOptions)
		if err != nil || nativeErr != nil || owned != "same projection" || native != owned || calls.Load() != 2 {
			t.Fatal(owned, err, native, nativeErr, calls.Load())
		}
	})
	t.Run("core and unsupported extensions reject before HTTP", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
		api := servers.New(flavorIdentityClient(cloud))
		for _, option := range []servers.ConsoleOutputOption{
			nil,
			servers.WithConsoleOutputField("length", 0),
			servers.WithConsoleOutputField("", true),
			servers.WithConsoleOutputField("vendor", func() {}),
			request.WithQuery[servers.ConsoleOutputOpts]("limit", "1"),
			request.WithHeader[servers.ConsoleOutputOpts]("X-Call", "unsupported"),
			request.WithArgument[servers.ConsoleOutputOpts]("vendor", true),
			func(config *request.Config[servers.ConsoleOutputOpts]) error {
				config.Fields["vendor"] = json.RawMessage(`{`)
				return nil
			},
		} {
			value, err := api.ConsoleOutput(context.Background(), "server-id", option)
			if value != "" || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal("invalid option made a request", value, err, calls.Load())
			}
		}
	})
}

func TestComputeConsoleOutputStringProjectionAndHTTPResponseEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		code             int
		failure, native  bool
	}{
		{"accepted200", `{"output":"value"}`, "value", 200, false, false},
		{"accepted201", `{"output":"value"}`, "value", 201, false, false},
		{"accepted203", `{"output":"value"}`, "value", 203, false, false},
		{"accepted300 without Location", `{"output":"value"}`, "value", 300, false, false},
		{"missing output", `{}`, "", 200, false, false},
		{"null output", `{"output":null}`, "", 200, false, false},
		{"empty output", `{"output":""}`, "", 200, false, false},
		{"native field matching", `{"OUTPUT":"matched"}`, "matched", 200, false, false},
		{"nonstring output", `{"output":false}`, "", 200, true, false},
		{"null root", `null`, "", 200, true, false},
		{"array root", `[]`, "", 200, true, false},
		{"malformed", `{"output":`, "", 200, true, false},
		{"invalid UTF8", "{\"output\":\"\xff\"}", "", 200, true, false},
		{"empty204", "", "", 204, true, false},
		{"bad request", `{"error":"original"}`, "", 400, true, false},
		{"forbidden", `{"error":"original"}`, "", 403, true, false},
		{"not found", `{"error":"original"}`, "", 404, true, false},
		{"native remains strict200", `{"output":"value"}`, "", 203, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, retries atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				if r.URL.Path != computeConsoleOutputPath || r.URL.RawQuery != "" {
					t.Error("unexpected lookup/fallback", r.URL)
				}
				w.Header().Set("X-Console-Proof", tc.name)
				testcloud.JSON(w, tc.code, tc.body)
			})
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			api := servers.New(flavorIdentityClient(cloud))
			var value string
			var err error
			if tc.native {
				value, err = api.ShowConsoleOutput(context.Background(), "server-id", servers.ShowConsoleOutputOpts{})
			} else {
				value, err = api.ConsoleOutput(context.Background(), "server-id")
			}
			if value != tc.want || (err != nil) != tc.failure || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
			if !tc.failure {
				if retries.Load() != 0 {
					t.Fatal("accepted response was retried", retries.Load())
				}
				return
			}
			var operation *resource.OperationError
			wantOperation := "ConsoleOutput"
			if tc.native {
				wantOperation = "ShowConsoleOutput"
			}
			if !errors.As(err, &operation) || operation.Operation != wantOperation || operation.Resource != "servers" {
				t.Fatal("operation context lost", err)
			}
			var proof *resource.ResponseError
			if tc.code >= 400 || tc.native {
				var original gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &original) || original.Actual != tc.code || original.Method != http.MethodPost || original.URL != cloud.Server.URL+computeConsoleOutputPath || string(original.Body) != tc.body || original.ResponseHeader.Get("X-Console-Proof") != tc.name || errors.As(err, &proof) || retries.Load() != 1 {
					t.Fatal("native HTTP rejection changed", err, original, retries.Load())
				}
				if tc.native && !reflect.DeepEqual(original.Expected, []int{200}) || !tc.native && (len(original.Expected) != 200 || original.Expected[0] != 200 || original.Expected[199] != 399) {
					t.Fatal("owned/native success policy changed", original.Expected)
				}
			} else if !errors.As(err, &proof) || proof.StatusCode != tc.code || proof.Header.Get("X-Console-Proof") != tc.name || string(proof.Body) != tc.body || retries.Load() != 0 {
				t.Fatal("accepted processing proof was lost or replayed", err, proof, retries.Load())
			}
		})
	}
}

func TestComputeConsoleOutputSourceContextAndAcceptedBodyBoundaries(t *testing.T) {
	t.Run("preflight source and cancellation", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
		for _, mode := range []string{"nil API", "nil context", "invalid ID", "pre-canceled", "option cancellation", "option source", "reassigned API"} {
			t.Run(mode, func(t *testing.T) {
				client := flavorIdentityClient(cloud)
				api := servers.New(client)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause, original := errors.New("console cancel cause"), errors.New("option original")
				id := "server-id"
				var options []servers.ConsoleOutputOption
				var later atomic.Int32
				switch mode {
				case "nil API":
					api = nil
				case "nil context":
					ctx = nil
				case "invalid ID":
					id = "other/server"
				case "pre-canceled":
					cancel(cause)
				case "option cancellation":
					options = append(options, func(_ *request.Config[servers.ConsoleOutputOpts]) error { cancel(cause); return original })
				case "option source":
					options = append(options, func(_ *request.Config[servers.ConsoleOutputOpts]) error {
						client.ResourceBase = cloud.Server.URL + "/changed/"
						return nil
					})
				case "reassigned API":
					options = append(options, func(_ *request.Config[servers.ConsoleOutputOpts]) error {
						*api = *servers.New(flavorIdentityClient(cloud))
						return nil
					})
				}
				options = append(options, func(_ *request.Config[servers.ConsoleOutputOpts]) error { later.Add(1); return nil })
				value, err := api.ConsoleOutput(ctx, id, options...)
				if value != "" || err == nil || calls.Load() != 0 || later.Load() != 0 {
					t.Fatal("invalid preflight reached callback/HTTP", value, err, calls.Load(), later.Load())
				}
				if mode == "pre-canceled" || mode == "option cancellation" {
					if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || mode == "option cancellation" && !errors.Is(err, original) {
						t.Fatal("cancellation/option cause lost", err)
					}
				} else if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			})
		}
	})
	for _, fault := range []string{"read", "close404"} {
		t.Run("accepted "+fault, func(t *testing.T) {
			cloud := testcloud.New(t)
			const body = `{"output":"accepted"}`
			cause := errors.New("original console read")
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{200}, Method: http.MethodPost, URL: "nested", Body: []byte("nested404")}
			var track *payloadContractTracking
			if fault == "read" {
				track = payloadContractTrack(cloud, cause, nil)
			} else {
				track = payloadContractTrack(cloud, nil, nested)
			}
			var calls, retries atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Console-Proof", fault)
				testcloud.JSON(w, 200, body)
			})
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			value, err := servers.New(flavorIdentityClient(cloud)).ConsoleOutput(context.Background(), "server-id")
			var proof *resource.ResponseError
			physical := track.last(t)
			if value != "" || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || proof.Header.Get("X-Console-Proof") != fault || calls.Load() != 1 || retries.Load() != 0 || track.calls.Load() != 1 || physical.reads.Load() == 0 || physical.closes.Load() != 1 {
				t.Fatal("accepted read/Close evidence or ownership changed", value, err, proof, calls.Load(), retries.Load(), physical)
			}
			if fault == "read" && !errors.Is(err, cause) || fault == "close404" && (!gophercloud.ResponseCodeIs(err, 404) || errors.Is(err, resource.ErrNotFound)) {
				t.Fatal("original accepted failure suppressed", err)
			}
		})
	}
	for _, mode := range []string{"source", "cancel"} {
		t.Run("accepted "+mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("accepted console cancellation")
			const body = `{"output":"observed"}`
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Console-Proof", mode)
				testcloud.JSON(w, 200, body)
			})
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
			value, err := servers.New(client).ConsoleOutput(ctx, "server-id")
			var proof *resource.ResponseError
			if value != "" || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || proof.Header.Get("X-Console-Proof") != mode || calls.Load() != 1 {
				t.Fatal("accepted source/context change lost proof", value, err, proof, calls.Load())
			}
			if mode == "source" && !errors.Is(err, resource.ErrInvalidOption) || mode == "cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
				t.Fatal(err)
			}
		})
	}
	t.Run("native reauthentication keeps body and options once", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.87"
		var calls, callbacks, reauth atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			index := calls.Add(1)
			th.TestMethod(t, r, http.MethodPost)
			th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.87")
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != `{"os-getConsoleOutput":{"length":0}}` || r.URL.Path != computeConsoleOutputPath {
				t.Error(string(body), err, r.URL)
			}
			if index == 1 {
				th.TestHeader(t, r, "X-Auth-Token", "test-token")
				w.WriteHeader(401)
				return
			}
			th.TestHeader(t, r, "X-Auth-Token", "reauth-console-token")
			testcloud.JSON(w, 200, `{"output":"retry accepted"}`)
		})
		cloud.Provider.ReauthFunc = func(context.Context) error {
			reauth.Add(1)
			cloud.Provider.SetToken("reauth-console-token")
			return nil
		}
		api := servers.New(client)
		value, err := api.ConsoleOutput(context.Background(), "server-id", servers.WithConsoleOutputLength(0), func(_ *request.Config[servers.ConsoleOutputOpts]) error { callbacks.Add(1); return nil })
		if err != nil || value != "retry accepted" || calls.Load() != 2 || callbacks.Load() != 1 || reauth.Load() != 1 || api.RawClient() != client || client.ProviderClient != cloud.Provider || client.Microversion != "2.87" {
			t.Fatal("native auth/body/options/source changed", value, err, calls.Load(), callbacks.Load(), reauth.Load())
		}
	})
}
