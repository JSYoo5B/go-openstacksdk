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

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

const flavorExtraSpecsFetchPath = flavorIdentityPath + "/7/os-extra_specs"

func TestComputeFlavorFetchExtraSpecsAlwaysGETRawValuesAndOwnedExistingInputs(t *testing.T) {
	for _, tc := range []struct {
		name, body, want, wantView string
		status                     int
	}{
		{"precise nested map", `{"extra_specs":{"cpu":"dedicated","n":9007199254740993,"nested":{"enabled":false}},"id":"wire-other","vendor":9007199254740993}`, `{"cpu":"dedicated","n":9007199254740993,"nested":{"enabled":false}}`, `{"cpu":"dedicated","n":9007199254740993,"nested":{"enabled":false}}`, 200},
		{"omission replaces with empty object", `{"vendor":17}`, `{}`, `{}`, 201},
		{"present null stays null", `{"extra_specs":null}`, `null`, `null`, 203},
		{"raw array with empty dict view", `{"extra_specs":[1,null,{"n":9007199254740993}]}`, `[1,null,{"n":9007199254740993}]`, `{}`, 202},
		{"raw scalar with empty dict view", `{"extra_specs":9007199254740993}`, `9007199254740993`, `{}`, 300},
		{"raw string with empty dict view", `{"extra_specs":"literal"}`, `"literal"`, `{}`, 200},
		{"raw false with empty dict view", `{"extra_specs":false}`, `false`, `{}`, 201},
		{"empty object stays present", `{"extra_specs":{}}`, `{}`, `{}`, 203},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.100"
			client.MoreHeaders = map[string]string{"X-Spec-Source": "selected", "Accept": "application/source+json"}
			cloud.Provider.SetToken("spec-live")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "spec-live")
				th.TestHeader(t, r, "X-Spec-Source", "selected")
				th.TestHeader(t, r, "Accept", "application/source+json")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.100")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.100")
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 || r.ContentLength != 0 || r.URL.RawQuery != "" || r.URL.Path != flavorExtraSpecsFetchPath {
					t.Error("fetch invented lookup/query/body", r.URL, string(body), err)
				}
				w.Header().Set("X-Spec-Proof", tc.name)
				testcloud.JSON(w, tc.status, tc.body)
			})
			value, err := flavors.New(client).FetchExtraSpecs(context.Background(), flavors.FlavorExtraSpecsRequest{ID: "7"})
			if err != nil || value == nil || value.Resource == nil || value.Wire == nil || string(value.ExtraSpecs) != tc.want || string(value.Resource.Body["extra_specs"]) != tc.wantView || string(value.Resource.Body["id"]) != `"7"` || string(value.Envelope) != tc.body || value.StatusCode != tc.status || value.Header.Get("X-Spec-Proof") != tc.name || value.Wire.StatusCode != tc.status || value.Wire.Header.Get("X-Spec-Proof") != tc.name || calls.Load() != 1 || client.Microversion != "2.100" || client.ResourceBase != cloud.Server.URL+"/reverse/nova/v2.1/project/" {
				t.Fatal(value, err, calls.Load(), client)
			}
			if _, present := value.Resource.Body["vendor"]; present {
				t.Fatal("response fields other than extra_specs replaced flavor seed", value.Resource)
			}
			if tc.name == "precise nested map" && (string(value.Wire.Body["id"]) != `"wire-other"` || string(value.Wire.Body["vendor"]) != `9007199254740993`) {
				t.Fatal("actual response Wire was normalized", value.Wire)
			}
			if tc.name == "omission replaces with empty object" {
				if _, present := value.Wire.Body["extra_specs"]; present {
					t.Fatal("omission default leaked into Wire", value.Wire)
				}
			}
			if raw, present := value.Wire.Body["extra_specs"]; present && string(raw) != tc.want {
				t.Fatal("descriptor conversion changed actual Wire", value.Wire, tc.want)
			}
			value.ExtraSpecs[0] = 'x'
			if string(value.Resource.Body["extra_specs"]) != tc.wantView {
				t.Fatal("extra specs bytes alias source view", value)
			}
			value.Resource.Body["extra_specs"][0] = 'x'
			value.Envelope[0] = 'x'
			value.Header.Set("X-Spec-Proof", "changed")
			if value.Wire.Header.Get("X-Spec-Proof") != tc.name {
				t.Fatal("receipt headers alias Wire", value.Wire)
			}
			for _, raw := range value.Wire.Body {
				if !json.Valid(raw) {
					t.Fatal("result bytes alias actual Wire", value.Wire)
				}
			}
		})
	}
	for _, mode := range []string{"native inline", "raw inline", "explicit ID preserves raw input ID", "raw original-name identity", "native name identity"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			native := &flavors.Flavor{ID: "7", Name: "original", Disk: 12, RAM: 2048, VCPUs: 3, Swap: 7, RxTxFactor: 0.5, IsPublic: true, Ephemeral: 5, Description: "detail", ExtraSpecs: map[string]string{"inline": "old"}}
			raw := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"7"`), "name": json.RawMessage(`"original"`), "vendor": json.RawMessage(`{"n":9007199254740993}`), "extra_specs": json.RawMessage(`{"inline":"old"}`)}, Header: http.Header{"X-Seed-Proof": {"input-owned"}}, StatusCode: 207}}
			input := flavors.FlavorExtraSpecsRequest{}
			wantID := `"7"`
			switch mode {
			case "native inline":
				input.Flavor = native
			case "native name identity":
				native.ID = ""
				native.Name = "7"
				input.Flavor = native
				wantID = `""`
			case "explicit ID preserves raw input ID":
				raw.Body["id"] = json.RawMessage(`"seed-other"`)
				input.Resource = raw
				input.ID = "7"
				wantID = `"seed-other"`
			case "raw original-name identity":
				raw.Body["id"] = json.RawMessage(`null`)
				raw.Body["name"] = json.RawMessage(`""`)
				raw.Body["original_name"] = json.RawMessage(`"7"`)
				input.Resource = raw
				wantID = `null`
			default:
				input.Resource = raw
			}
			version := "2.99"
			bulk := flavors.WithFlavorExtraSpecsOptions(flavors.FlavorExtraSpecsOpts{Microversion: &version})
			version = "changed"
			shared := map[string]string{"X-Spec-Trace": "owned", "Accept": "application/vendor+json"}
			var calls, first, last atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.99")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.99")
				th.TestHeader(t, r, "X-Spec-Trace", "owned")
				th.TestHeader(t, r, "Accept", "application/vendor+json")
				if r.URL.Path != flavorExtraSpecsFetchPath || r.URL.RawQuery != "" || r.ContentLength != 0 {
					t.Error("inline avoided fetch or caller input changed route", r.URL)
				}
				w.Header().Set("X-Spec-Proof", "fresh receipt")
				testcloud.JSON(w, 201, `{"extra_specs":{"fresh":"value"},"name":"response-is-not-flavor"}`)
			})
			value, err := flavors.New(client).FetchExtraSpecs(context.Background(), input, bulk, func(c *request.Config[flavors.FlavorExtraSpecsOpts]) error {
				first.Add(1)
				c.Headers = shared
				return nil
			}, func(_ *request.Config[flavors.FlavorExtraSpecsOpts]) error {
				last.Add(1)
				shared["X-Spec-Trace"] = "changed"
				native.ID = "caller-change"
				native.Name = "caller-change"
				native.ExtraSpecs["inline"] = "caller-change"
				raw.Body["id"][0] = 'x'
				raw.Body["vendor"][0] = 'x'
				raw.Header.Set("X-Seed-Proof", "caller-change")
				return nil
			})
			if err != nil || value == nil || value.Resource == nil || value.Wire == nil || string(value.ExtraSpecs) != `{"fresh":"value"}` || string(value.Resource.Body["extra_specs"]) != `{"fresh":"value"}` || string(value.Resource.Body["id"]) != wantID || calls.Load() != 1 || first.Load() != 1 || last.Load() != 1 || client.Microversion != "2.55" {
				t.Fatal("always-fetch or owned seed changed", value, err, calls.Load(), first.Load(), last.Load())
			}
			if input.Flavor != nil {
				wantName := `"original"`
				if mode == "native name identity" {
					wantName = `"7"`
				}
				if string(value.Resource.Body["name"]) != wantName || string(value.Resource.Body["swap"]) != `7` || string(value.Resource.Body["disk"]) != `12` || string(value.Resource.Body["ram"]) != `2048` || string(value.Resource.Body["vcpus"]) != `3` || string(value.Resource.Body["rxtx_factor"]) != `0.5` || string(value.Resource.Body["is_public"]) != `true` || string(value.Resource.Body["ephemeral"]) != `5` || string(value.Resource.Body["description"]) != `"detail"` {
					t.Fatal("native model seed lost known fields", value.Resource)
				}
			} else if string(value.Resource.Body["vendor"]) != `{"n":9007199254740993}` || value.Resource.Header.Get("X-Seed-Proof") != "input-owned" || value.Resource.StatusCode != 207 {
				t.Fatal("raw existing fields or metadata changed", value.Resource)
			}
			if value.Header.Get("X-Spec-Proof") != "fresh receipt" || value.StatusCode != 201 || value.Wire.Header.Get("X-Spec-Proof") != "fresh receipt" || value.Wire.StatusCode != 201 || string(value.Wire.Body["name"]) != `"response-is-not-flavor"` {
				t.Fatal("fresh receipt was replaced by seed metadata", value)
			}
			if _, present := value.Wire.Body["vendor"]; present {
				t.Fatal("input seed leaked into actual Wire", value.Wire)
			}
			value.Resource.Body["extra_specs"][0] = 'x'
			if string(value.ExtraSpecs) != `{"fresh":"value"}` || string(value.Wire.Body["extra_specs"]) != `{"fresh":"value"}` {
				t.Fatal("owned outputs share extra-specs bytes", value)
			}
		})
	}
	for _, tc := range []struct {
		name, selected, max, min, version string
		discover, empty                   bool
	}{
		{"selected above resource ceiling", "2.100", "", "", "2.100", false, false},
		{"global latest selected remains literal", "latest", "", "", "latest", false, false},
		{"unselected capped at2.61", "", "2.110", "2.1", "2.61", true, false},
		{"unselected older maximum", "", "2.55", "2.1", "2.55", true, false},
		{"missing maximum is versionless", "", "", "2.1", "", true, false},
		{"explicit empty overrides selected", "2.100", "", "", "", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Endpoint = cloud.Server.URL + "/reverse/nova/v2.1/project/"
			client.Microversion = tc.selected
			var discoveries, gets atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				th.TestMethod(t, r, http.MethodGet)
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 || r.ContentLength != 0 || r.URL.RawQuery != "" {
					t.Error(r.URL, string(body), err)
				}
				if r.URL.Path == computeConsoleDiscoveryPath {
					discoveries.Add(1)
					th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
					th.TestHeaderUnset(t, r, "OpenStack-API-Version")
					testcloud.JSON(w, 201, `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"`+tc.min+`","version":"`+tc.max+`"}}`)
					return
				}
				gets.Add(1)
				if r.URL.Path != flavorExtraSpecsFetchPath {
					t.Error(r.URL)
				}
				if tc.version == "" {
					th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
					th.TestHeaderUnset(t, r, "OpenStack-API-Version")
				} else {
					th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", tc.version)
					th.TestHeader(t, r, "OpenStack-API-Version", "compute "+tc.version)
				}
				testcloud.JSON(w, 200, `{"extra_specs":{}}`)
			})
			var opts []flavors.FlavorExtraSpecsOption
			if tc.empty {
				opts = append(opts, flavors.WithFlavorExtraSpecsMicroversion(""))
			}
			value, err := flavors.New(client).FetchExtraSpecs(context.Background(), flavors.FlavorExtraSpecsRequest{ID: "7"}, opts...)
			wantDiscovery := int32(0)
			if tc.discover {
				wantDiscovery = 1
			}
			if err != nil || value == nil || string(value.ExtraSpecs) != `{}` || gets.Load() != 1 || discoveries.Load() != wantDiscovery || client.Microversion != tc.selected {
				t.Fatal(value, err, gets.Load(), discoveries.Load(), wantDiscovery, client)
			}
		})
	}
}

func TestComputeFlavorFetchExtraSpecsPreflightAndAcceptedTerminalReceipts(t *testing.T) {
	for _, mode := range []string{"nil API", "nil context", "both existing inputs", "nonstring raw ID", "nil option", "cancel callback", "source version callback", "outer guard", "auth header before discovery"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			api := flavors.New(client)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			input := flavors.FlavorExtraSpecsRequest{ID: "7"}
			cause := errors.New("extra specs outer cause")
			var calls, later atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				t.Error("preflight touched HTTP", r.URL)
				w.WriteHeader(500)
			})
			var options []flavors.FlavorExtraSpecsOption
			expected := resource.ErrInvalidOption
			switch mode {
			case "nil API":
				api = nil
			case "nil context":
				ctx = nil
			case "both existing inputs":
				input.Flavor = &flavors.Flavor{ID: "7"}
				input.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"7"`)}}}
			case "nonstring raw ID":
				input.ID = ""
				input.Resource = &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`7`)}}}
			case "nil option":
				options = append(options, nil)
			case "cancel callback":
				expected = context.Canceled
				options = append(options, func(_ *request.Config[flavors.FlavorExtraSpecsOpts]) error { cancel(); return nil })
			case "source version callback":
				options = append(options, func(_ *request.Config[flavors.FlavorExtraSpecsOpts]) error { client.Microversion = "2.1"; return nil })
			case "outer guard":
				expected = cause
				ctx = rest.WithOperationGuard(ctx, func(context.Context) error { return cause })
			case "auth header before discovery":
				client.Microversion = ""
				client.Endpoint = cloud.Server.URL + "/reverse/nova/v2.1/project/"
				options = append(options, flavors.WithFlavorExtraSpecsHeader("X-Auth-Token", "forged"))
			}
			options = append(options, func(_ *request.Config[flavors.FlavorExtraSpecsOpts]) error { later.Add(1); return nil })
			value, err := api.FetchExtraSpecs(ctx, input, options...)
			if value != nil || !errors.Is(err, expected) || calls.Load() != 0 {
				t.Fatal(value, err, calls.Load())
			}
			if mode != "auth header before discovery" && later.Load() != 0 {
				t.Fatal("preflight ran later callback", later.Load())
			}
		})
	}
	for _, mode := range []string{"malformed JSON", "null root", "array root", "empty204", "invalid UTF8", "accepted read404", "accepted close404", "source after fetch", "forbidden"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			body := `{"extra_specs":{"actual":"value"}}`
			status := 200
			switch mode {
			case "malformed JSON":
				body = `{"extra_specs":`
			case "null root":
				body = `null`
			case "array root":
				body = `[]`
			case "empty204":
				body = ""
				status = 204
			case "invalid UTF8":
				body = "{\"extra_specs\":{\"value\":\"\xff\"}}"
			case "forbidden":
				status = 403
				body = `{"error":"extra specs forbidden"}`
			}
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Method: http.MethodGet, URL: "nested", Body: []byte("original nested404"), ResponseHeader: http.Header{"X-Nested-Proof": {"actual"}}}
			var track *payloadContractTracking
			if mode == "accepted read404" {
				track = payloadContractTrack(cloud, nested, nil)
			} else if mode == "accepted close404" {
				track = payloadContractTrack(cloud, nil, nested)
			} else if mode == "source after fetch" {
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
				th.TestMethod(t, r, http.MethodGet)
				if r.URL.Path != flavorExtraSpecsFetchPath || r.URL.RawQuery != "" {
					t.Error("terminal fetch performed another operation", r.URL)
				}
				w.Header().Set("X-Spec-Proof", mode)
				testcloud.JSON(w, status, body)
			})
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			value, err := flavors.New(client).FetchExtraSpecs(context.Background(), flavors.FlavorExtraSpecsRequest{ID: "7"})
			if status == 403 {
				var native gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.As(err, &native) || native.Actual != 403 || len(native.Expected) != 200 || native.Method != http.MethodGet || native.URL != cloud.Server.URL+flavorExtraSpecsFetchPath || string(native.Body) != body || native.ResponseHeader.Get("X-Spec-Proof") != mode || calls.Load() != 1 || retries.Load() != 1 {
					t.Fatal(value, err, native, calls.Load(), retries.Load())
				}
				return
			}
			var proof *resource.ResponseError
			if value == nil || value.Resource != nil || value.Wire != nil || len(value.ExtraSpecs) != 0 || value.StatusCode != status || string(value.Envelope) != body || value.Header.Get("X-Spec-Proof") != mode || !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Spec-Proof") != mode || errors.Is(err, resource.ErrNotFound) || calls.Load() != 1 || retries.Load() != 0 {
				t.Fatal("accepted failure lost receipt or was replayed", value, err, proof, calls.Load(), retries.Load())
			}
			if mode == "source after fetch" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if track != nil {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.URL != "nested" || native.Actual != 404 || string(native.Body) != "original nested404" || native.ResponseHeader.Get("X-Nested-Proof") != "actual" {
					t.Fatal(err, native)
				}
				physical := track.last(t)
				if track.calls.Load() != 1 || physical.reads.Load() == 0 || physical.closes.Load() != 1 {
					t.Fatal(track.calls.Load(), physical)
				}
			}
		})
	}
}

func TestComputeFlavorListExtraSpecsNativeBindingKeepsStrictStringMapABI(t *testing.T) {
	for _, tc := range []struct {
		name, body         string
		status             int
		want               map[string]string
		failure, typeError bool
	}{
		{"native strings and null member", `{"extra_specs":{"cpu":"dedicated","number":"1","blank":null},"vendor":17}`, 200, map[string]string{"cpu": "dedicated", "number": "1", "blank": ""}, false, false},
		{"native omitted is nil", `{}`, 200, nil, false, false},
		{"native null is nil", `{"extra_specs":null}`, 200, nil, false, false},
		{"native empty is owned map", `{"extra_specs":{}}`, 200, map[string]string{}, false, false},
		{"native numeric member preserves partial map and type error", `{"extra_specs":{"bad":7,"cpu":"retained"}}`, 200, map[string]string{"bad": "", "cpu": "retained"}, true, true},
		{"native array root is a type error", `[]`, 200, nil, true, true},
		{"native malformed is syntax error", `{"extra_specs":!}`, 200, nil, true, false},
		{"native truncated is unexpected EOF", `{"extra_specs":`, 200, nil, true, false},
		{"native201 is rejected", `{"extra_specs":{"ignored":"value"}}`, 201, nil, true, false},
		{"native204 is rejected", "", 204, nil, true, false},
		{"native403 is original HTTP", `{"error":"native specs forbidden"}`, 403, nil, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			client.MoreHeaders = map[string]string{"X-Spec-Source": "native"}
			cloud.Provider.SetToken("native-spec-live")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "native-spec-live")
				th.TestHeader(t, r, "X-Spec-Source", "native")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.55")
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 || r.ContentLength != 0 || r.URL.RawQuery != "" || r.URL.Path != flavorExtraSpecsFetchPath {
					t.Error(r.URL, string(body), err)
				}
				w.Header().Set("X-Spec-Proof", tc.name)
				testcloud.JSON(w, tc.status, tc.body)
			})
			value, err := flavors.New(client).ListExtraSpecs(context.Background(), "7")
			if !reflect.DeepEqual(value, tc.want) || (err != nil) != tc.failure || calls.Load() != 1 || client.Microversion != "2.55" || client.ResourceBase != cloud.Server.URL+"/reverse/nova/v2.1/project/" {
				t.Fatal(value, err, tc.want, calls.Load())
			}
			if tc.status != 200 {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != tc.status || !reflect.DeepEqual(native.Expected, []int{200}) || native.Method != http.MethodGet || native.URL != cloud.Server.URL+flavorExtraSpecsFetchPath || string(native.Body) != tc.body || native.ResponseHeader.Get("X-Spec-Proof") != tc.name {
					t.Fatal("native HTTP proof changed", err, native)
				}
			} else if tc.typeError {
				var typed *json.UnmarshalTypeError
				if !errors.As(err, &typed) {
					t.Fatal(err)
				}
			} else if tc.name == "native truncated is unexpected EOF" {
				if !errors.Is(err, io.ErrUnexpectedEOF) {
					t.Fatal(err)
				}
			} else if tc.failure {
				var syntax *json.SyntaxError
				if !errors.As(err, &syntax) {
					t.Fatal(err)
				}
			}
		})
	}
}
