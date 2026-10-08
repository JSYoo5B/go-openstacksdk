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

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

// openstacksdk ef55d7d: _proxy.py:2935–2944 -> ConsoleAuthToken.fetch.
const computeConsoleTokenPath = "/reverse/nova/v2.1/project/os-console-auth-tokens/console-token"

func TestComputeConsoleAuthTokenFixedLookupAndOwnedVersionSelection(t *testing.T) {
	for _, tc := range []struct {
		name, selected, maximum, minimum, version string
		discover                                  bool
	}{
		{"selected above resource ceiling bypasses discovery", "2.100", "", "", "2.100", false},
		{"global latest is literal selected default", "latest", "", "", "latest", false},
		{"major latest is literal selected default", "2.latest", "", "", "2.latest", false},
		{"unselected ceiling", "", "2.110", "2.1", "2.99", true},
		{"unselected older server has no local gate", "", "2.5", "2.1", "2.5", true},
		{"missing maximum continues versionless", "", "", "2.1", "", true},
		{"minimum above ceiling continues versionless", "", "2.110", "2.100", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Endpoint = cloud.Server.URL + "/reverse/nova/v2.1/project/"
			client.Microversion = tc.selected
			client.MoreHeaders = map[string]string{"X-Token-Source": "selected"}
			wantSourceHeaders := map[string]string{"X-Token-Source": "selected"}
			cloud.Provider.SetToken("provider-live-auth")
			var discoveries, lookups atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "provider-live-auth")
				th.TestHeader(t, r, "X-Token-Source", "selected")
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 || r.ContentLength != 0 || r.URL.RawQuery != "" {
					t.Error("lookup/discovery gained a body or query", r.URL, string(body), r.ContentLength, err)
				}
				if r.URL.Path == computeConsoleDiscoveryPath {
					discoveries.Add(1)
					if !tc.discover {
						t.Error("selected default triggered discovery")
					}
					th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
					th.TestHeaderUnset(t, r, "OpenStack-API-Version")
					testcloud.JSON(w, 201, `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"`+tc.minimum+`","version":"`+tc.maximum+`"}}`)
					return
				}
				lookups.Add(1)
				if r.URL.Path != computeConsoleTokenPath {
					t.Error("token fetch performed another resource request", r.URL)
				}
				if tc.version == "" {
					th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
					th.TestHeaderUnset(t, r, "OpenStack-API-Version")
				} else {
					th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", tc.version)
					th.TestHeader(t, r, "OpenStack-API-Version", "compute "+tc.version)
				}
				w.Header().Set("X-Token-Proof", tc.name)
				testcloud.JSON(w, 203, `{"console":{}}`)
			})
			service := compute.New(client, compute.Dependencies{})
			value, err := service.ValidateConsoleAuthToken(context.Background(), "console-token")
			wantDiscovery := int32(0)
			if tc.discover {
				wantDiscovery = 1
			}
			if err != nil || value == nil || value.Resource == nil || value.Wire == nil || len(value.Resource.Body) != 8 || len(value.Wire.Body) != 0 || string(value.Resource.Body["id"]) != `"console-token"` || string(value.Resource.Body["name"]) != `null` || string(value.Resource.Body["location"]) != `null` || value.StatusCode != 203 || string(value.Envelope) != `{"console":{}}` || value.Header.Get("X-Token-Proof") != tc.name || discoveries.Load() != wantDiscovery || lookups.Load() != 1 || client.Microversion != tc.selected || service.RawClient() != client || !reflect.DeepEqual(client.MoreHeaders, wantSourceHeaders) || client.Endpoint != cloud.Server.URL+"/reverse/nova/v2.1/project/" || client.ResourceBase != cloud.Server.URL+"/reverse/nova/v2.1/project/" {
				t.Fatal("owned selection/default seed changed source or lookup", value, err, discoveries.Load(), lookups.Load(), client)
			}
		})
	}
	t.Run("opaque Unicode token is one member and not auth header", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.100"
		cloud.Provider.SetToken("provider-auth-only")
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestMethod(t, r, http.MethodGet)
			th.TestHeader(t, r, "X-Auth-Token", "provider-auth-only")
			body, err := io.ReadAll(r.Body)
			if err != nil || len(body) != 0 || r.ContentLength != 0 || r.URL.RawQuery != "" || r.URL.Path != "/reverse/nova/v2.1/project/os-console-auth-tokens/token+한글" || r.RequestURI != "/reverse/nova/v2.1/project/os-console-auth-tokens/token+%ED%95%9C%EA%B8%80" {
				t.Error("opaque token was normalized or parsed", r.URL, r.RequestURI, string(body), err)
			}
			testcloud.JSON(w, 200, `{"console":{"instance_uuid":"actual-instance"}}`)
		})
		value, err := compute.New(client, compute.Dependencies{}).ValidateConsoleAuthToken(context.Background(), "token+한글")
		if err != nil || value == nil || value.Resource == nil || string(value.Resource.Body["id"]) != `"token+한글"` || string(value.Resource.Body["instance_uuid"]) != `"actual-instance"` || calls.Load() != 1 {
			t.Fatal(value, err, calls.Load())
		}
	})
}

func TestComputeConsoleAuthTokenSeededResourceActualWireAndResponseEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, body, id, nameField string
		status                    int
		failure, wire             bool
	}{
		{"all passive fields", `{"console":{"id":9007199254740993,"name":null,"instance_uuid":["passive"],"host":{"node":"literal"},"port":"5900","tls_port":null,"internal_access_path":false,"location":{"cloud":"forged"},"vendor":{"n":9007199254740993}}}`, `9007199254740993`, `null`, 200, false, true},
		{"flat null overrides seeded id", `{"id":null,"name":17,"instance_uuid":null,"host":"actual-host","port":5900,"tls_port":"15900","internal_access_path":"/internal","vendor":17}`, `null`, `17`, 201, false, true},
		{"empty wrapped preserves seeded id", `{"console":{}}`, `"console-token"`, `null`, 203, false, true},
		{"empty accepted preserves seed", "", `"console-token"`, `null`, 204, false, false},
		{"opaque accepted preserves seed", "accepted opaque", `"console-token"`, `null`, 202, false, false},
		{"malformed accepted preserves seed", `{"console":`, `"console-token"`, `null`, 201, false, false},
		{"invalid UTF8 follows documented Go policy", "{\"console\":{\"host\":\"\xff\"}}", `"console-token"`, `null`, 201, false, false},
		{"valid null root fails mapping", `null`, "", "", 200, true, false},
		{"valid array root fails mapping", `[]`, "", "", 200, true, false},
		{"valid null console fails mapping", `{"console":null}`, "", "", 200, true, false},
		{"valid scalar console fails mapping", `{"console":17}`, "", "", 200, true, false},
		{"forbidden remains HTTP failure", `{"error":"token forbidden"}`, "", "", 403, true, false},
		{"missing remains HTTP failure", `{"error":"token missing"}`, "", "", 404, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.100"
			var calls, retries atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				if r.URL.Path != computeConsoleTokenPath || r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				w.Header().Set("X-Token-Proof", tc.name)
				testcloud.JSON(w, tc.status, tc.body)
			})
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			value, err := compute.New(client, compute.Dependencies{}).ValidateConsoleAuthToken(context.Background(), "console-token")
			if (err != nil) != tc.failure || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
			if tc.status >= 400 {
				var original gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.As(err, &original) || original.Actual != tc.status || len(original.Expected) != 200 || original.Method != http.MethodGet || original.URL != cloud.Server.URL+computeConsoleTokenPath || string(original.Body) != tc.body || original.ResponseHeader.Get("X-Token-Proof") != tc.name || retries.Load() != 1 {
					t.Fatal("token HTTP proof hidden by missing-ignore/validation", value, err, original, retries.Load())
				}
				return
			}
			if value == nil || value.StatusCode != tc.status || string(value.Envelope) != tc.body || value.Header.Get("X-Token-Proof") != tc.name || retries.Load() != 0 {
				t.Fatal("accepted token lookup receipt lost/replayed", value, err, retries.Load())
			}
			if tc.failure {
				var proof *resource.ResponseError
				if value.Resource != nil || !errors.As(err, &proof) || proof.StatusCode != tc.status || string(proof.Body) != tc.body || proof.Header.Get("X-Token-Proof") != tc.name {
					t.Fatal("valid nonobject became successful empty Resource", value, err, proof)
				}
				return
			}
			if value.Resource == nil || len(value.Resource.Body) != 8 || (value.Wire != nil) != tc.wire || string(value.Resource.Body["id"]) != tc.id || string(value.Resource.Body["name"]) != tc.nameField || string(value.Resource.Body["location"]) != `null` || value.Resource.StatusCode != tc.status || value.Resource.Header.Get("X-Token-Proof") != tc.name {
				t.Fatal("seed/default/null/passive mapping changed", value)
			}
			if _, present := value.Resource.Body["vendor"]; present {
				t.Fatal("unknown field entered Resource", value.Resource)
			}
			if tc.name == "all passive fields" {
				for key, want := range map[string]string{"instance_uuid": `["passive"]`, "host": `{"node":"literal"}`, "port": `"5900"`, "tls_port": `null`, "internal_access_path": `false`} {
					if string(value.Resource.Body[key]) != want {
						t.Fatal(key, string(value.Resource.Body[key]), want)
					}
				}
				if string(value.Wire.Body["vendor"]) != `{"n":9007199254740993}` || string(value.Wire.Body["location"]) != `{"cloud":"forged"}` {
					t.Fatal("actual wire lost precise unknown fields", value.Wire)
				}
			}
			if tc.name == "flat null overrides seeded id" {
				if string(value.Resource.Body["port"]) != `5900` || string(value.Resource.Body["tls_port"]) != `"15900"` || string(value.Resource.Body["internal_access_path"]) != `"/internal"` {
					t.Fatal("known fields forced into native types", value.Resource)
				}
			}
			if tc.wire {
				if tc.name == "empty wrapped preserves seeded id" && len(value.Wire.Body) != 0 {
					t.Fatal("seed/defaults synthesized into Wire", value.Wire)
				}
				value.Resource.Body["id"][0] = 'x'
				value.Resource.Header.Set("X-Token-Proof", "Resource changed")
				if value.Header.Get("X-Token-Proof") != tc.name || value.Wire.Header.Get("X-Token-Proof") != tc.name {
					t.Fatal("Resource metadata aliases raw receipt", value)
				}
				value.Header.Set("X-Token-Proof", "record changed")
				value.Envelope[0] = 'x'
				for _, raw := range value.Wire.Body {
					if !json.Valid(raw) {
						t.Fatal("Resource/Envelope mutation reached raw wire fields", value.Wire)
					}
				}
				if value.Wire.Header.Get("X-Token-Proof") != tc.name {
					t.Fatal("record header aliases Wire header", value.Wire)
				}
			}
		})
	}
	t.Run("owned location is recorded once before discovery", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Endpoint = cloud.Server.URL + "/reverse/nova/v2.1/project/"
		cloudName, region := "configured", "region-one"
		project := json.RawMessage(`"scope"`)
		var locations, discoveries, lookups atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			th.TestMethod(t, r, http.MethodGet)
			if locations.Load() != 1 {
				t.Error("location was not owned before discovery/lookup", locations.Load())
			}
			if r.URL.Path == computeConsoleDiscoveryPath {
				discoveries.Add(1)
				cloudName = "changed in discovery"
				region = "changed in discovery"
				project[1] = 'x'
				testcloud.JSON(w, 200, `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.100"}}`)
				return
			}
			lookups.Add(1)
			if r.URL.Path != computeConsoleTokenPath {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, `{"console":{"location":{"cloud":"forged"},"host":"actual"}}`)
		})
		service := compute.New(client, compute.Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
			locations.Add(1)
			return resource.CloudLocation{Cloud: &cloudName, RegionName: &region, Project: resource.CloudProject{ID: project}}, nil
		}})
		value, err := service.ValidateConsoleAuthToken(context.Background(), "console-token")
		if err != nil || value == nil || value.Resource == nil || value.Wire == nil || locations.Load() != 1 || discoveries.Load() != 1 || lookups.Load() != 1 || string(value.Wire.Body["location"]) != `{"cloud":"forged"}` {
			t.Fatal(value, err, locations.Load(), discoveries.Load(), lookups.Load())
		}
		var location resource.CloudLocation
		if decodeErr := json.Unmarshal(value.Resource.Body["location"], &location); decodeErr != nil || location.Cloud == nil || *location.Cloud != "configured" || location.RegionName == nil || *location.RegionName != "region-one" || string(location.Project.ID) != `"scope"` {
			t.Fatal("computed location rerun, response-forged or caller-aliased", string(value.Resource.Body["location"]), decodeErr)
		}
	})
}

func TestComputeConsoleAuthTokenPreflightAcceptedFaultsAndLiveNativePolicy(t *testing.T) {
	for _, mode := range []string{"nil context", "nil facade", "empty ID", "control ID", "invalid UTF8 ID", "nil source", "blank source version headers", "outer guard", "pre-canceled", "location source", "location cancellation", "location facade"} {
		t.Run("preflight "+mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.100"
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause, original := errors.New("token guard cause"), errors.New("location callback cause")
			var calls, locations atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(500) })
			var service *compute.Service
			dependencies := compute.Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
				locations.Add(1)
				switch mode {
				case "location source":
					client.ResourceBase = cloud.Server.URL + "/changed/"
				case "location cancellation":
					cancel(cause)
					return resource.CloudLocation{}, original
				case "location facade":
					service.API = nil
				}
				return resource.CloudLocation{}, nil
			}}
			service = compute.New(client, dependencies)
			token := "console-token"
			switch mode {
			case "nil context":
				ctx = nil
			case "nil facade":
				service = nil
			case "empty ID":
				token = ""
			case "control ID":
				token = "bad\x00token"
			case "invalid UTF8 ID":
				token = "bad\xfftoken"
			case "nil source":
				service = compute.New(nil, dependencies)
			case "blank source version headers":
				client.Microversion = ""
				client.MoreHeaders = map[string]string{"OpenStack-API-Version": "", "X-OpenStack-Nova-API-Version": ""}
			case "outer guard":
				ctx = rest.WithOperationGuard(ctx, func(context.Context) error { return cause })
			case "pre-canceled":
				cancel(cause)
			}
			value, err := service.ValidateConsoleAuthToken(ctx, token)
			if value != nil || err == nil || calls.Load() != 0 {
				t.Fatal("invalid preflight performed authenticated I/O", value, err, calls.Load())
			}
			if mode == "location source" || mode == "location cancellation" || mode == "location facade" {
				if locations.Load() != 1 {
					t.Fatal(locations.Load())
				}
			} else if locations.Load() != 0 {
				t.Fatal("invalid invocation reached location callback", locations.Load())
			}
			if mode == "pre-canceled" || mode == "location cancellation" {
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || mode == "location cancellation" && !errors.Is(err, original) {
					t.Fatal("context/callback causes lost", err)
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
	for _, tc := range []struct {
		name                    string
		discover, close, source bool
	}{{"lookup read404", false, false, false}, {"lookup Close404", false, true, false}, {"discovery Close404", true, true, false}, {"lookup source after response", false, false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Endpoint = cloud.Server.URL + "/reverse/nova/v2.1/project/"
			client.Microversion = "2.100"
			body, status, path := `{"console":{"host":"actual"}}`, 203, computeConsoleTokenPath
			if tc.discover {
				client.Microversion = ""
				body, status, path = `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.100"}}`, 300, computeConsoleDiscoveryPath
			}
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{200}, Method: http.MethodGet, URL: "nested", Body: []byte("original nested404"), ResponseHeader: http.Header{"X-Nested-Proof": {"actual"}}}
			var track *payloadContractTracking
			if tc.source {
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
			} else if tc.close {
				track = payloadContractTrack(cloud, nil, nested)
			} else {
				track = payloadContractTrack(cloud, nested, nil)
			}
			var calls, retries atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				if r.URL.Path != path {
					t.Error("accepted fault performed fallback/followup", r.URL, path)
				}
				w.Header().Set("X-Token-Proof", tc.name)
				testcloud.JSON(w, status, body)
			})
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			value, err := compute.New(client, compute.Dependencies{}).ValidateConsoleAuthToken(context.Background(), "console-token")
			var proof *resource.ResponseError
			var original gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Token-Proof") != tc.name || errors.Is(err, resource.ErrNotFound) || calls.Load() != 1 || retries.Load() != 0 {
				t.Fatal("accepted fault ignored/replayed or current evidence lost", value, err, proof, original, calls.Load(), retries.Load())
			}
			if tc.source {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.As(err, &original) || original.Actual != 404 || original.URL != "nested" || string(original.Body) != "original nested404" || original.ResponseHeader.Get("X-Nested-Proof") != "actual" {
				t.Fatal("nested physical cause lost", err, original)
			}
			if tc.discover {
				if value != nil {
					t.Fatal("discovery failure became lookup result", value)
				}
			} else if value == nil || value.Resource != nil || value.Wire != nil || value.StatusCode != status || string(value.Envelope) != body || value.Header.Get("X-Token-Proof") != tc.name {
				t.Fatal("lookup partial receipt lost", value)
			}
			if track != nil {
				physical := track.last(t)
				if track.calls.Load() != 1 || physical.reads.Load() == 0 || physical.closes.Load() != 1 {
					t.Fatal(track.calls.Load(), physical)
				}
			}
		})
	}
	for _, mode := range []string{"source", "version", "body"} {
		t.Run("native retry "+mode+" mutation prevents another lookup", func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.100"
			var calls, retries atomic.Int32
			const body = "original token503"
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				if r.URL.Path != computeConsoleTokenPath {
					t.Error(r.URL)
				}
				w.Header().Set("X-Token-Proof", "retry")
				testcloud.JSON(w, 503, body)
			})
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, _ error, _ uint) error {
				retries.Add(1)
				switch mode {
				case "source":
					client.ResourceBase = cloud.Server.URL + "/changed/"
				case "version":
					if options.MoreHeaders == nil {
						options.MoreHeaders = make(map[string]string)
					}
					options.MoreHeaders["OpenStack-API-Version"] = "compute 2.5"
					options.MoreHeaders["X-OpenStack-Nova-API-Version"] = "2.5"
				case "body":
					options.JSONBody = map[string]any{"vendor": true}
				}
				return nil
			}
			value, err := compute.New(client, compute.Dependencies{}).ValidateConsoleAuthToken(context.Background(), "console-token")
			var original gophercloud.ErrUnexpectedResponseCode
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &original) || original.Actual != 503 || original.Method != http.MethodGet || original.URL != cloud.Server.URL+computeConsoleTokenPath || string(original.Body) != body || original.ResponseHeader.Get("X-Token-Proof") != "retry" || calls.Load() != 1 || retries.Load() != 1 {
				t.Fatal("retry changed token source or lost original HTTP", value, err, original, calls.Load(), retries.Load())
			}
		})
	}
	t.Run("live reauthentication keeps the fixed console token", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Microversion = "2.100"
		var calls, reauths atomic.Int32
		cloud.Provider.ReauthFunc = func(context.Context) error {
			if reauths.Add(1) > 1 {
				return errors.New("bounded token reauth exhausted")
			}
			cloud.Provider.SetToken("refreshed-provider-auth")
			return nil
		}
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			index := calls.Add(1)
			th.TestMethod(t, r, http.MethodGet)
			th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.100")
			body, err := io.ReadAll(r.Body)
			if err != nil || len(body) != 0 || r.ContentLength != 0 || r.URL.RawQuery != "" || r.URL.Path != computeConsoleTokenPath {
				t.Error("reauth changed console token request", r.URL, string(body), err)
			}
			if index == 1 {
				th.TestHeader(t, r, "X-Auth-Token", "test-token")
				testcloud.JSON(w, 401, `{"error":"expired auth"}`)
				return
			}
			th.TestHeader(t, r, "X-Auth-Token", "refreshed-provider-auth")
			w.Header().Set("X-Token-Proof", "live reauth")
			testcloud.JSON(w, 200, `{"console":{"host":"actual"}}`)
		})
		value, err := compute.New(client, compute.Dependencies{}).ValidateConsoleAuthToken(context.Background(), "console-token")
		if err != nil || value == nil || value.Resource == nil || string(value.Resource.Body["id"]) != `"console-token"` || string(value.Resource.Body["host"]) != `"actual"` || value.Header.Get("X-Token-Proof") != "live reauth" || calls.Load() != 2 || reauths.Load() != 1 || client.Microversion != "2.100" {
			t.Fatal(value, err, calls.Load(), reauths.Load(), client)
		}
	})
}
