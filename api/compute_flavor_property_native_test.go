package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/compute/v2/flavors"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestComputeFlavorGetNativeBindingDTOAndOriginalHTTPErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body, failure string
		status              int
		want                *flavors.Flavor
	}{
		{"all native fields with numeric swap", `{"flavor":{"id":"7","name":"small","disk":12,"ram":2048,"vcpus":3,"swap":7,"rxtx_factor":0.5,"os-flavor-access:is_public":true,"OS-FLV-EXT-DATA:ephemeral":5,"description":"detail","extra_specs":{"cpu":"dedicated"},"vendor":17}}`, "", 200, &flavors.Flavor{ID: "7", Name: "small", Disk: 12, RAM: 2048, VCPUs: 3, Swap: 7, RxTxFactor: 0.5, IsPublic: true, Ephemeral: 5, Description: "detail", ExtraSpecs: map[string]string{"cpu": "dedicated"}}},
		{"string swap converts then truncates", `{"flavor":{"id":"7","swap":"7.9"}}`, "", 200, &flavors.Flavor{ID: "7", Swap: 7}},
		{"blank swap is zero", `{"flavor":{"id":"7","swap":""}}`, "", 200, &flavors.Flavor{ID: "7"}},
		{"missing public and extra specs keep native zero values", `{"flavor":{"id":"7"}}`, "", 200, &flavors.Flavor{ID: "7"}},
		{"nullable native fields use zero values", `{"flavor":{"id":null,"name":null,"ram":null,"swap":null,"os-flavor-access:is_public":null,"extra_specs":null}}`, "", 200, &flavors.Flavor{}},
		{"missing flavor wrapper is nil", `{"id":"7"}`, "", 200, nil},
		{"null flavor wrapper is nil", `{"flavor":null}`, "", 200, nil},
		{"type failure returns allocated unassigned native flavor", `{"flavor":{"id":"7","name":"kept in decoder temporary","ram":"wrong"}}`, "type", 200, &flavors.Flavor{}},
		{"invalid string swap keeps assigned native fields", `{"flavor":{"id":"7","name":"partial","swap":"wrong"}}`, "number", 200, &flavors.Flavor{ID: "7", Name: "partial"}},
		{"203 is original HTTP rejection", `{"flavor":{"id":"ignored"}}`, "http", 203, nil},
		{"204 is original HTTP rejection", "", "http", 204, nil},
		{"403 retains original body and header", `{"error":"flavor forbidden"}`, "http", 403, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			client.MoreHeaders = map[string]string{"X-Flavor-Source": "native"}
			cloud.Provider.SetToken("flavor-live")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "flavor-live")
				th.TestHeader(t, r, "X-Flavor-Source", "native")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.55")
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 || r.ContentLength != 0 || r.URL.RawQuery != "" || r.URL.Path != flavorIdentityPath+"/7" {
					t.Error("native Get changed its binding", r.URL, string(body), err)
				}
				w.Header().Set("X-Flavor-Proof", tc.name)
				testcloud.JSON(w, tc.status, tc.body)
			})
			value, err := flavors.New(client).Get(context.Background(), "7")
			if !reflect.DeepEqual(value, tc.want) || (err != nil) != (tc.failure != "") || calls.Load() != 1 || client.Microversion != "2.55" || client.ResourceBase != cloud.Server.URL+"/reverse/nova/v2.1/project/" {
				t.Fatal(value, err, tc.want, calls.Load())
			}
			switch tc.failure {
			case "http":
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &proof) || proof.Actual != tc.status || !reflect.DeepEqual(proof.Expected, []int{200}) || proof.Method != http.MethodGet || proof.URL != cloud.Server.URL+flavorIdentityPath+"/7" || string(proof.Body) != tc.body || proof.ResponseHeader.Get("X-Flavor-Proof") != tc.name {
					t.Fatal("native Get lost actual HTTP proof", err, proof)
				}
			case "type":
				var typed *json.UnmarshalTypeError
				if !errors.As(err, &typed) {
					t.Fatal(err)
				}
			case "number":
				var numeric *strconv.NumError
				if !errors.As(err, &numeric) {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestComputeFlavorGetExtraSpecNativeBindingReturnsWholeStringMap(t *testing.T) {
	for _, tc := range []struct {
		name, body, failure string
		status              int
		want                map[string]string
	}{
		{"whole map includes unknown keys and null string member", `{"cpu":"dedicated","other":"retained","blank":null}`, "", 200, map[string]string{"cpu": "dedicated", "other": "retained", "blank": ""}},
		{"missing requested property still returns whole map", `{"other":"retained"}`, "", 200, map[string]string{"other": "retained"}},
		{"null root is native nil map", `null`, "", 200, nil},
		{"empty root is empty map", `{}`, "", 200, map[string]string{}},
		{"numeric member returns partial map and type error", `{"cpu":7,"other":"retained"}`, "type", 200, map[string]string{"cpu": "", "other": "retained"}},
		{"array root returns type error", `[]`, "type", 200, nil},
		{"203 is strict native rejection", `{"cpu":"ignored"}`, "http", 203, nil},
		{"204 is strict native rejection", "", "http", 204, nil},
		{"404 preserves original HTTP proof", `{"error":"property missing"}`, "http", 404, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			cloud.Provider.SetToken("property-native-live")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "property-native-live")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.55")
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 || r.ContentLength != 0 || r.URL.RawQuery != "" || r.URL.Path != flavorExtraSpecsFetchPath+"/cpu" {
					t.Error("native property binding changed", r.URL, string(body), err)
				}
				w.Header().Set("X-Property-Proof", tc.name)
				testcloud.JSON(w, tc.status, tc.body)
			})
			value, err := flavors.New(client).GetExtraSpec(context.Background(), "7", "cpu")
			if !reflect.DeepEqual(value, tc.want) || (err != nil) != (tc.failure != "") || calls.Load() != 1 || client.Microversion != "2.55" {
				t.Fatal(value, err, tc.want, calls.Load())
			}
			if tc.failure == "http" {
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &proof) || proof.Actual != tc.status || !reflect.DeepEqual(proof.Expected, []int{200}) || proof.Method != http.MethodGet || proof.URL != cloud.Server.URL+flavorExtraSpecsFetchPath+"/cpu" || string(proof.Body) != tc.body || proof.ResponseHeader.Get("X-Property-Proof") != tc.name {
					t.Fatal("native property lost actual HTTP proof", err, proof)
				}
			} else if tc.failure == "type" {
				var typed *json.UnmarshalTypeError
				if !errors.As(err, &typed) {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestComputeFlavorListDetailNativeBindingEagerOptionsLazyLinkedPages(t *testing.T) {
	for _, mode := range []string{"default omits is_public", "typed query with raw overrides", "two linked pages", "empty200", "empty204", "accepted300", "late403", "break before second page", "malformed first page once", "typed malformed row once", "first203 rejection", "option error before HTTP"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			cloud.Provider.SetToken("list-detail-live")
			cause := errors.New("native detail option cause")
			var calls, callbacks atomic.Int32
			options := []flavors.ListDetailOption{func(_ *request.Config[flavors.ListOpts]) error {
				callbacks.Add(1)
				if mode == "option error before HTTP" {
					return cause
				}
				return nil
			}}
			wantQuery := url.Values{}
			if mode == "typed query with raw overrides" {
				options = append(options,
					flavors.WithListDetailOptions(flavors.ListOpts{ChangesSince: "2026-01-01T00:00:00Z", MinDisk: 12, MinRAM: 1024, SortDir: "asc", SortKey: "name", Marker: "start", Limit: 2, AccessType: flavors.AllAccess}),
					flavors.WithListDetailQuery("limit", "1"), flavors.WithListDetailQuery("is_public", "false"), flavors.WithListDetailQuery("vendor", "literal"))
				wantQuery = url.Values{"changes-since": {"2026-01-01T00:00:00Z"}, "minDisk": {"12"}, "minRam": {"1024"}, "sort_dir": {"asc"}, "sort_key": {"name"}, "marker": {"start"}, "limit": {"1"}, "is_public": {"false"}, "vendor": {"literal"}}
			}
			linked := mode == "two linked pages" || mode == "late403" || mode == "break before second page"
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				call := calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "list-detail-live")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.55")
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 || r.ContentLength != 0 || r.URL.Path != flavorIdentityPath+"/detail" || callbacks.Load() != 1 {
					t.Error("native detail route/body/options changed", r.URL, string(body), callbacks.Load(), err)
				}
				if call == 1 {
					if !reflect.DeepEqual(r.URL.Query(), wantQuery) {
						t.Error("native detail default or query merge changed", r.URL.Query(), wantQuery)
					}
				} else if !linked || call != 2 || r.URL.RawQuery != "marker=next" {
					t.Error("native detail invented continuation", call, r.URL)
				}
				w.Header().Set("X-List-Proof", mode)
				if call == 2 {
					if mode == "late403" {
						testcloud.JSON(w, 403, `{"error":"late detail forbidden"}`)
					} else {
						testcloud.JSON(w, 200, flavorIdentityPage(`{"id":"8","name":"second","swap":""}`, ""))
					}
					return
				}
				switch mode {
				case "empty200":
					testcloud.JSON(w, 200, flavorIdentityPage("", cloud.Server.URL+flavorIdentityPath+"/detail?marker=never"))
				case "empty204":
					// The native page parser decodes only a JSON Content-Type.
					w.WriteHeader(204)
				case "malformed first page once":
					testcloud.JSON(w, 200, `{"flavors":!}`)
				case "typed malformed row once":
					testcloud.JSON(w, 200, flavorIdentityPage(`{"id":"7","ram":"wrong"}`, ""))
				case "first203 rejection":
					testcloud.JSON(w, 203, `{"error":"detail status rejected"}`)
				default:
					status := 200
					if mode == "accepted300" {
						status = 300
					}
					next := ""
					if linked {
						next = cloud.Server.URL + flavorIdentityPath + "/detail?marker=next"
					}
					testcloud.JSON(w, status, flavorIdentityPage(`{"id":"7","name":"first","swap":"3.9","extra_specs":{"cpu":"dedicated"}}`, next))
				}
			})
			sequence := flavors.New(client).ListDetail(context.Background(), options...)
			if callbacks.Load() != 1 || calls.Load() != 0 {
				t.Fatal("native options were lazy or HTTP was eager", callbacks.Load(), calls.Load())
			}
			var ids []string
			var final error
			failures := 0
			for value, err := range sequence {
				if err != nil {
					failures++
					final = err
					if value != nil {
						t.Error("native list error yielded value", value)
					}
					continue
				}
				if value == nil {
					t.Fatal("native detail yielded nil success")
				}
				ids = append(ids, value.ID)
				if value.ID == "7" && (value.Name != "first" || value.Swap != 3 || !reflect.DeepEqual(value.ExtraSpecs, map[string]string{"cpu": "dedicated"}) || value.IsPublic) {
					t.Error("native detail DTO changed", value)
				}
				if mode == "break before second page" {
					break
				}
			}
			wantIDs := []string{"7"}
			wantCalls, wantFailures := int32(1), 0
			switch mode {
			case "two linked pages":
				wantIDs, wantCalls = []string{"7", "8"}, 2
			case "late403":
				wantCalls, wantFailures = 2, 1
			case "empty200", "empty204":
				wantIDs = nil
			case "malformed first page once", "typed malformed row once", "first203 rejection":
				wantIDs, wantFailures = nil, 1
			case "option error before HTTP":
				wantIDs, wantCalls, wantFailures = nil, 0, 1
			}
			if !reflect.DeepEqual(ids, wantIDs) || calls.Load() != wantCalls || failures != wantFailures || callbacks.Load() != 1 || client.Microversion != "2.55" {
				t.Fatal(ids, wantIDs, final, calls.Load(), wantCalls, failures, wantFailures)
			}
			if mode == "late403" || mode == "first203 rejection" {
				status, target, body := 403, cloud.Server.URL+flavorIdentityPath+"/detail?marker=next", `{"error":"late detail forbidden"}`
				if mode == "first203 rejection" {
					status, target, body = 203, cloud.Server.URL+flavorIdentityPath+"/detail", `{"error":"detail status rejected"}`
				}
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(final, &proof) || proof.Actual != status || !reflect.DeepEqual(proof.Expected, []int{200, 204, 300}) || proof.Method != http.MethodGet || proof.URL != target || string(proof.Body) != body || proof.ResponseHeader.Get("X-List-Proof") != mode {
					t.Fatal("native pagination HTTP proof changed", final, proof)
				}
			} else if mode == "malformed first page once" {
				var syntax *json.SyntaxError
				if !errors.As(final, &syntax) {
					t.Fatal(final)
				}
			} else if mode == "typed malformed row once" {
				var typed *json.UnmarshalTypeError
				if !errors.As(final, &typed) {
					t.Fatal(final)
				}
			} else if mode == "option error before HTTP" && !errors.Is(final, cause) {
				t.Fatal(final)
			}
		})
	}
}

func TestComputeFlavorGetExtraSpecsPropertyRawValuesLiteralRouteAndVersionBinding(t *testing.T) {
	const property = "hw:cpu/+%?#한 글"
	propertyKey, _ := json.Marshal(property)
	for _, tc := range []struct {
		name, raw string
		present   bool
		status    int
	}{
		{"precise scalar stays raw", `9007199254740993`, true, 200},
		{"explicit null is present", `null`, true, 203},
		{"missing property is absent null", `null`, false, 201},
		{"string stays raw", `"dedicated"`, true, 202},
		{"array stays raw", `[1,null,{"n":9007199254740993}]`, true, 300},
		{"map stays raw", `{"enabled":false,"n":9007199254740993}`, true, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.100"
			cloud.Provider.SetToken("property-owned-live")
			body := `{"vendor":9007199254740993}`
			if tc.present {
				body = `{` + string(propertyKey) + `:` + tc.raw + `,"vendor":9007199254740993}`
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "property-owned-live")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.100")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.100")
				th.TestHeader(t, r, "Accept", "application/json")
				payload, err := io.ReadAll(r.Body)
				if err != nil || len(payload) != 0 || r.ContentLength != 0 || r.URL.RawQuery != "" || r.URL.EscapedPath() != flavorExtraSpecsFetchPath+"/"+url.PathEscape(property) {
					t.Error("property route was normalized or whole-specs fetched", r.URL, string(payload), err)
				}
				w.Header().Set("X-Property-Proof", tc.name)
				testcloud.JSON(w, tc.status, body)
			})
			value, err := flavors.New(client).GetExtraSpecsProperty(context.Background(), flavors.FlavorExtraSpecsRequest{ID: "7"}, property)
			if err != nil || value == nil || value.Wire == nil || string(value.Value) != tc.raw || value.Present != tc.present || string(value.Envelope) != body || value.StatusCode != tc.status || value.Header.Get("X-Property-Proof") != tc.name || string(value.Wire.Body["vendor"]) != `9007199254740993` || calls.Load() != 1 || client.Microversion != "2.100" {
				t.Fatal(value, err, calls.Load())
			}
			wireRaw, wirePresent := value.Wire.Body[property]
			if wirePresent != tc.present || wirePresent && string(wireRaw) != tc.raw {
				t.Fatal("property default or alias leaked into actual Wire", value.Wire)
			}
			value.Value[0] = 'x'
			if tc.present && string(value.Wire.Body[property]) != tc.raw {
				t.Fatal("property Value aliases Wire bytes", value)
			}
			value.Envelope[0] = 'x'
			value.Header.Set("X-Property-Proof", "changed")
			if value.Wire.Header.Get("X-Property-Proof") != tc.name || !json.Valid(value.Wire.Body["vendor"]) {
				t.Fatal("property receipt aliases Wire", value)
			}
		})
	}
	for _, mode := range []string{"native inline identity snapshot", "raw input explicit route ID", "unselected discovers2.61 once", "explicit empty is versionless"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Endpoint = cloud.Server.URL + "/reverse/nova/v2.1/project/"
			client.Microversion = "2.55"
			native := &flavors.Flavor{ID: "7", ExtraSpecs: map[string]string{"cpu": "inline"}}
			raw := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"other"`), "extra_specs": json.RawMessage(`{"cpu":"inline"}`), "vendor": json.RawMessage(`17`)}}}
			input := flavors.FlavorExtraSpecsRequest{ID: "7"}
			var options []flavors.FlavorExtraSpecsOption
			version := "2.55"
			switch mode {
			case "native inline identity snapshot":
				input = flavors.FlavorExtraSpecsRequest{Flavor: native}
			case "raw input explicit route ID":
				input.Resource = raw
			case "unselected discovers2.61 once":
				client.Microversion, version = "", "2.61"
			case "explicit empty is versionless":
				options = append(options, flavors.WithFlavorExtraSpecsMicroversion(""))
				version = ""
			}
			var callbacks, discoveries, gets atomic.Int32
			options = append(options, func(_ *request.Config[flavors.FlavorExtraSpecsOpts]) error {
				callbacks.Add(1)
				native.ID = "caller-change"
				raw.Body["id"] = json.RawMessage(`"caller-change"`)
				return nil
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				th.TestMethod(t, r, http.MethodGet)
				body, err := io.ReadAll(r.Body)
				if err != nil || len(body) != 0 || r.ContentLength != 0 || r.URL.RawQuery != "" || callbacks.Load() != 1 {
					t.Error(r.URL, string(body), err, callbacks.Load())
				}
				if r.URL.Path == computeConsoleDiscoveryPath {
					discoveries.Add(1)
					th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
					th.TestHeaderUnset(t, r, "OpenStack-API-Version")
					testcloud.JSON(w, 200, `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.110"}}`)
					return
				}
				gets.Add(1)
				if r.URL.Path != flavorExtraSpecsFetchPath+"/cpu" {
					t.Error("property lost its captured identity", r.URL)
				}
				if version == "" {
					th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
					th.TestHeaderUnset(t, r, "OpenStack-API-Version")
				} else {
					th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", version)
					th.TestHeader(t, r, "OpenStack-API-Version", "compute "+version)
				}
				testcloud.JSON(w, 200, `{"cpu":"fresh"}`)
			})
			selected := client.Microversion
			value, err := flavors.New(client).GetExtraSpecsProperty(context.Background(), input, "cpu", options...)
			wantDiscovery := int32(0)
			if mode == "unselected discovers2.61 once" {
				wantDiscovery = 1
			}
			if err != nil || value == nil || value.Wire == nil || !value.Present || string(value.Value) != `"fresh"` || gets.Load() != 1 || discoveries.Load() != wantDiscovery || callbacks.Load() != 1 || client.Microversion != selected || native.ExtraSpecs["cpu"] != "inline" || string(raw.Body["extra_specs"]) != `{"cpu":"inline"}` {
				t.Fatal(value, err, gets.Load(), discoveries.Load(), callbacks.Load(), client)
			}
			if _, present := value.Wire.Body["id"]; present {
				t.Fatal("input flavor seed leaked into property Wire", value.Wire)
			}
		})
	}
}

func TestComputeFlavorGetExtraSpecsPropertyPreflightAndBindingTerminalReceipts(t *testing.T) {
	for _, property := range []string{"", ".", "..", "bad\x01property"} {
		t.Run("invalid property "+strconv.Quote(property), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			var calls, callbacks atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				t.Error("invalid property touched discovery or GET", r.URL)
				w.WriteHeader(500)
			})
			value, err := flavors.New(client).GetExtraSpecsProperty(context.Background(), flavors.FlavorExtraSpecsRequest{ID: "7"}, property, func(_ *request.Config[flavors.FlavorExtraSpecsOpts]) error {
				callbacks.Add(1)
				return nil
			})
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 || callbacks.Load() != 0 {
				t.Fatal(value, err, calls.Load(), callbacks.Load())
			}
		})
	}
	for _, mode := range []string{"accepted read404", "accepted close404", "source after property", "array root rejected", "actual404 is strict"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Microversion = "2.55"
			body, status := `{"cpu":"actual","vendor":17}`, 200
			if mode == "array root rejected" {
				body = `[]`
			} else if mode == "actual404 is strict" {
				body, status = `{"error":"property absent"}`, 404
			}
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Method: http.MethodGet, URL: "nested-property", Body: []byte("physical nested404"), ResponseHeader: http.Header{"X-Nested-Proof": {"actual"}}}
			var track *payloadContractTracking
			if mode == "accepted read404" {
				track = payloadContractTrack(cloud, nested, nil)
			} else if mode == "accepted close404" {
				track = payloadContractTrack(cloud, nil, nested)
			} else if mode == "source after property" {
				base := cloud.Provider.HTTPClient.Transport
				if base == nil {
					base = http.DefaultTransport
				}
				cloud.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					response, err := base.RoundTrip(r)
					if err == nil {
						client.Microversion = "2.99"
					}
					return response, err
				})
			}
			var calls, retries atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				if r.URL.Path != flavorExtraSpecsFetchPath+"/cpu" || r.URL.RawQuery != "" || r.ContentLength != 0 {
					t.Error("property terminal path introduced fallback", r.URL)
				}
				w.Header().Set("X-Property-Proof", mode)
				testcloud.JSON(w, status, body)
			})
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			value, err := flavors.New(client).GetExtraSpecsProperty(context.Background(), flavors.FlavorExtraSpecsRequest{ID: "7"}, "cpu")
			if mode == "actual404 is strict" {
				var proof gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.As(err, &proof) || proof.Actual != 404 || len(proof.Expected) != 200 || proof.Method != http.MethodGet || proof.URL != cloud.Server.URL+flavorExtraSpecsFetchPath+"/cpu" || string(proof.Body) != body || proof.ResponseHeader.Get("X-Property-Proof") != mode || calls.Load() != 1 || retries.Load() != 1 || errors.Is(err, resource.ErrNotFound) {
					t.Fatal("actual property404 was normalized or retried", value, err, proof, calls.Load(), retries.Load())
				}
				return
			}
			var receipt *resource.ResponseError
			if value == nil || value.Wire != nil || len(value.Value) != 0 || value.Present || value.StatusCode != status || string(value.Envelope) != body || value.Header.Get("X-Property-Proof") != mode || !errors.As(err, &receipt) || receipt.StatusCode != status || string(receipt.Body) != body || receipt.Header.Get("X-Property-Proof") != mode || calls.Load() != 1 || retries.Load() != 0 || errors.Is(err, resource.ErrNotFound) {
				t.Fatal("accepted property failure lost receipt or replayed", value, err, receipt, calls.Load(), retries.Load())
			}
			if mode == "source after property" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if track != nil {
				var proof gophercloud.ErrUnexpectedResponseCode
				physical := track.last(t)
				if !errors.As(err, &proof) || proof.Actual != 404 || proof.URL != "nested-property" || string(proof.Body) != "physical nested404" || proof.ResponseHeader.Get("X-Nested-Proof") != "actual" || track.calls.Load() != 1 || physical.reads.Load() == 0 || physical.closes.Load() != 1 {
					t.Fatal("property physical cause changed", err, proof, track.calls.Load(), physical)
				}
			}
		})
	}
}
