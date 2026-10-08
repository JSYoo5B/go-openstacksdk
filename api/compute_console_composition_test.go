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

	"github.com/JSYoo5B/gophercloudsdk/compute"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

// Python create_console (_proxy.py:2963–3001) owns advertised/selected
// branching; the direct modern and legacy executors have separate tests.
const computeConsoleDiscoveryPath = "/reverse/nova/v2.1/"
const computeConsoleDiscoveryRoot = "/reverse/nova/"
const computeConsoleLegacyPath = "/reverse/nova/v2.1/project/servers/server-id/action"

func TestComputeConsoleCompositionAdvertisedAndSelectedBranches(t *testing.T) {
	for _, tc := range []struct {
		name, selected, discovery, version string
		modern                             bool
		firstRejection, discoveryStatus    int
		noMatchingRow                      bool
	}{
		{"unselected modern ceiling", "", `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.110"}}`, "2.99", true, 0, 300, false},
		{"explicit selected exceeds advertised max", "2.100", `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.90"}}`, "2.100", true, 0, 300, false},
		{"explicit selected below branch", "2.5", `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.100"}}`, "2.5", false, 0, 300, false},
		{"unselected old server", "", `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.5"}}`, "2.5", false, 0, 300, false},
		{"minimum above required uses legacy ceiling", "", `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.7","version":"2.110"}}`, "2.100", false, 0, 300, false},
		{"matched version without bounds", "", `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}]}}`, "", false, 0, 300, false},
		{"global latest does not prove modern", "latest", `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.110"}}`, "latest", false, 0, 300, false},
		{"major latest proves modern", "2.latest", `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.110"}}`, "2.latest", true, 0, 300, false},
		{"v2.0 row ignored for catalog v2.1", "", `{"versions":{"values":[{"id":"v2.0","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.0/"}],"min_version":"2.1","version":"2.5"},{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","max_version":"2.8"}]}}`, "2.8", true, 0, 300, false},
		{"clean404 root fallback", "", `{"versions":[{"id":"v2.0","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.0/"}],"min_version":"2.1","version":"2.5"},{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.100"}]}`, "2.99", true, 404, 300, false},
		{"clean405 root fallback", "2.6", `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.100"}}`, "2.6", true, 405, 300, false},
		{"valid accepted201 discovery", "2.6", `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.100"}}`, "2.6", true, 0, 201, false},
		{"missing status is ineligible", "", `{"version":{"id":"v2.1","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.100"}}`, "", false, 0, 200, true},
		{"missing self link is ineligible", "", `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"collection","href":"/reverse/nova/"}],"min_version":"2.1","version":"2.100"}}`, "", false, 0, 200, true},
		{"unknown status is ineligible", "", `{"version":{"id":"v2.1","status":"vendor","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.100"}}`, "", false, 0, 200, true},
		{"relative self link and deprecated status eligible", "2.6", `{"version":{"id":"v2.1","status":"deprecated","links":[{"rel":"self","href":""}],"min_version":"2.1","version":"2.100"}}`, "2.6", true, 0, 200, false},
		{"missing ID row skipped", "2.6", `{"versions":[{"status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.5"},{"id":"v2.1","status":"SUPPORTED","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.100"}]}`, "2.6", true, 0, 200, false},
		{"ineligible malformed ID row skipped", "2.6", `{"versions":[{"id":false,"status":"EXPERIMENTAL","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.5"},{"id":"v2.1","status":"SUPPORTED","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.100"}]}`, "2.6", true, 0, 200, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Endpoint = cloud.Server.URL + "/reverse/nova/v2.1/project/"
			client.Microversion = tc.selected
			client.MoreHeaders = map[string]string{"X-Console-Source": "selected"}
			cloud.Provider.SetToken("composition-live")
			var gets, posts atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				th.TestHeader(t, r, "X-Auth-Token", "composition-live")
				th.TestHeader(t, r, "X-Console-Source", "selected")
				body, err := io.ReadAll(r.Body)
				if err != nil || r.URL.RawQuery != "" {
					t.Error(err, r.URL)
				}
				if r.Method == http.MethodGet {
					index := gets.Add(1)
					th.TestMethod(t, r, http.MethodGet)
					th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
					th.TestHeaderUnset(t, r, "OpenStack-API-Version")
					wantPath := computeConsoleDiscoveryPath
					if index == 2 && (tc.firstRejection != 0 || tc.noMatchingRow) {
						wantPath = computeConsoleDiscoveryRoot
					}
					if r.URL.Path != wantPath || len(body) != 0 || r.ContentLength != 0 {
						t.Error("discovery scope/body changed", r.URL, string(body), r.ContentLength)
					}
					if index == 1 && tc.firstRejection != 0 {
						testcloud.JSON(w, tc.firstRejection, `{"error":"clean discovery rejection"}`)
						return
					}
					testcloud.JSON(w, tc.discoveryStatus, tc.discovery)
					return
				}
				posts.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				if tc.version == "" {
					th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
				} else {
					th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", tc.version)
					th.TestHeader(t, r, "OpenStack-API-Version", "compute "+tc.version)
				}
				if tc.modern {
					if r.URL.Path != computeRemoteConsolePath || string(body) != `{"remote_console":{"protocol":"vnc","type":"novnc"}}` {
						t.Error("modern winner changed", r.URL, string(body))
					}
					testcloud.JSON(w, 201, `{"remote_console":{"url":"modern","vendor":9007199254740993}}`)
				} else {
					th.TestHeader(t, r, "Accept", "")
					if r.URL.Path != computeConsoleLegacyPath || string(body) != `{"os-getVNCConsole":{"type":"novnc"}}` {
						t.Error("legacy winner changed", r.URL, string(body))
					}
					testcloud.JSON(w, 203, `{"console":{"type":"actual","url":"legacy","vendor":{"n":9007199254740993}}}`)
				}
			})
			service := compute.New(client, compute.Dependencies{})
			value, err := service.CreateConsole(context.Background(), "server-id", "novnc")
			wantGets := int32(1)
			if tc.firstRejection != 0 || tc.noMatchingRow {
				wantGets = 2
			}
			if err != nil || gets.Load() != wantGets || posts.Load() != 1 || client.Microversion != tc.selected || client.Endpoint != cloud.Server.URL+"/reverse/nova/v2.1/project/" || client.ResourceBase != cloud.Server.URL+"/reverse/nova/v2.1/project/" || service.RawClient() != client || service.API.RawClient() != client || !reflect.DeepEqual(client.MoreHeaders, map[string]string{"X-Console-Source": "selected"}) {
				t.Fatal(value, err, gets.Load(), posts.Load(), client)
			}
			if !tc.modern {
				if string(value) != `{"type":"actual","url":"legacy","vendor":{"n":9007199254740993}}` {
					t.Fatal("legacy result narrowed/reseeded", string(value))
				}
				return
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(value, &fields); err != nil || len(fields) != 6 || string(fields["protocol"]) != `null` || string(fields["type"]) != `"novnc"` || string(fields["url"]) != `"modern"` || string(fields["id"]) != `null` || string(fields["name"]) != `null` || string(fields["location"]) != `null` {
				t.Fatal("modern source dictionary projection changed", string(value), err)
			}
		})
	}
}

func TestComputeConsoleCompositionProtocolTypeAndLocationProjection(t *testing.T) {
	for _, tc := range []struct {
		name, selected, maximum, kind, requestBody, response, protocol string
		options                                                        []compute.ConsoleOption
		modern, unsupported                                            bool
	}{
		{"default null protocol view", "2.6", "2.100", "novnc", `{"remote_console":{"protocol":"vnc","type":"novnc"}}`, `{"remote_console":{"url":"actual"}}`, `null`, nil, true, false},
		{"explicit protocol preserved", "2.6", "2.100", "novnc", `{"remote_console":{"protocol":"custom","type":"novnc"}}`, `{"remote_console":{"url":"actual"}}`, `"custom"`, []compute.ConsoleOption{compute.WithConsoleProtocol("custom")}, true, false},
		{"empty protocol seed survives derived request", "2.6", "2.100", "novnc", `{"remote_console":{"protocol":"vnc","type":"novnc"}}`, `{"remote_console":{"url":"actual"}}`, `""`, []compute.ConsoleOption{compute.WithConsoleProtocol("")}, true, false},
		{"unset restores explicit null facade default", "2.6", "2.100", "novnc", `{"remote_console":{"protocol":"vnc","type":"novnc"}}`, `{"remote_console":{"url":"actual"}}`, `null`, []compute.ConsoleOption{compute.WithConsoleProtocol("discarded"), compute.WithConsoleProtocolValue(request.Optional[string]{})}, true, false},
		{"unknown modern type server validated", "2.6", "2.100", "vendor", `{"remote_console":{"protocol":null,"type":"vendor"}}`, `{"remote_console":{"url":"actual"}}`, `null`, nil, true, false},
		{"webmks selected below gate", "2.6", "2.100", "webmks", "", "", "", nil, true, true},
		{"webmks advertised below gate", "2.100", "2.7", "webmks", "", "", "", nil, true, true},
		{"webmks supported boundary", "2.8", "2.8", "webmks", `{"remote_console":{"protocol":"mks","type":"webmks"}}`, `{"remote_console":{"url":"actual"}}`, `null`, nil, true, false},
		{"spice-direct advertised below gate", "2.100", "2.98", "spice-direct", "", "", "", nil, true, true},
		{"spice-direct supported boundary", "", "2.110", "spice-direct", `{"remote_console":{"protocol":"spice","type":"spice-direct"}}`, `{"remote_console":{"url":"actual"}}`, `null`, nil, true, false},
		{"legacy ignores protocol", "2.5", "2.100", "serial", `{"os-getSerialConsole":{"type":"serial"}}`, `{"console":[null,17,{"vendor":9007199254740993}]}`, "", []compute.ConsoleOption{compute.WithConsoleProtocol("ignored")}, false, false},
		{"legacy spice-direct has no modern gate", "2.5", "2.100", "spice-direct", `{"os-getSPICEConsole":{"type":"spice-direct"}}`, `{"console":null}`, "", nil, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Endpoint = cloud.Server.URL + "/reverse/nova/v2.1/project/"
			client.Microversion = tc.selected
			var gets, posts atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					gets.Add(1)
					if r.URL.Path != computeConsoleDiscoveryPath || r.URL.RawQuery != "" || r.ContentLength != 0 {
						t.Error(r.URL, r.ContentLength)
					}
					th.TestHeaderUnset(t, r, "X-OpenStack-Nova-API-Version")
					testcloud.JSON(w, 200, `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"`+tc.maximum+`"}}`)
					return
				}
				posts.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				body, err := io.ReadAll(r.Body)
				wantPath := computeConsoleLegacyPath
				if tc.modern {
					wantPath = computeRemoteConsolePath
				}
				if err != nil || r.URL.Path != wantPath || r.URL.RawQuery != "" || string(body) != tc.requestBody {
					t.Error(string(body), err, r.URL, tc.requestBody)
				}
				testcloud.JSON(w, 200, tc.response)
			})
			value, err := compute.New(client, compute.Dependencies{}).CreateConsole(context.Background(), "server-id", tc.kind, tc.options...)
			if gets.Load() != 1 || client.Microversion != tc.selected {
				t.Fatal("branch discovery or selected source mutated", err, gets.Load(), client.Microversion)
			}
			if tc.unsupported {
				if value != nil || !errors.Is(err, resource.ErrUnsupported) || posts.Load() != 0 {
					t.Fatal("type gate ignored advertised or selected support", string(value), err, posts.Load())
				}
				return
			}
			if err != nil || posts.Load() != 1 {
				t.Fatal(string(value), err, posts.Load())
			}
			if tc.modern {
				var fields map[string]json.RawMessage
				if err := json.Unmarshal(value, &fields); err != nil || len(fields) != 6 || string(fields["protocol"]) != tc.protocol || string(fields["type"]) != `"`+tc.kind+`"` || string(fields["location"]) != `null` {
					t.Fatal(string(value), err)
				}
			} else if tc.kind == "serial" && string(value) != `[null,17,{"vendor":9007199254740993}]` || tc.kind == "spice-direct" && string(value) != `null` {
				t.Fatal("legacy raw value changed", string(value))
			}
		})
	}
	t.Run("modern response IDs and owned computed location", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Endpoint = cloud.Server.URL + "/reverse/nova/v2.1/project/"
		client.Microversion = "2.6"
		cloudName, region := "configured", "region-one"
		project := json.RawMessage(`"scope"`)
		var locations, gets, posts atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				gets.Add(1)
				testcloud.JSON(w, 200, `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.100"}}`)
				return
			}
			posts.Add(1)
			if locations.Load() != 1 {
				t.Error("location was not snapshotted before POST", locations.Load())
			}
			cloudName = "caller changed"
			region = "caller changed"
			project[1] = 'x'
			testcloud.JSON(w, 201, `{"remote_console":{"id":9007199254740993,"name":null,"protocol":null,"type":false,"url":"actual","server_id":"foreign","location":{"cloud":"forged"},"vendor":17}}`)
		})
		service := compute.New(client, compute.Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
			locations.Add(1)
			return resource.CloudLocation{Cloud: &cloudName, RegionName: &region, Project: resource.CloudProject{ID: project}}, nil
		}})
		value, err := service.CreateConsole(context.Background(), "server-id", "novnc")
		var fields map[string]json.RawMessage
		if decodeErr := json.Unmarshal(value, &fields); err != nil || decodeErr != nil || len(fields) != 6 || string(fields["id"]) != `9007199254740993` || string(fields["name"]) != `null` || string(fields["protocol"]) != `null` || string(fields["type"]) != `false` || string(fields["url"]) != `"actual"` || gets.Load() != 1 || posts.Load() != 1 || locations.Load() != 1 {
			t.Fatal(string(value), err, decodeErr, gets.Load(), posts.Load(), locations.Load())
		}
		var location resource.CloudLocation
		if decodeErr := json.Unmarshal(fields["location"], &location); decodeErr != nil || location.Cloud == nil || *location.Cloud != "configured" || location.RegionName == nil || *location.RegionName != "region-one" || string(location.Project.ID) != `"scope"` || string(location.Zone) != `null` {
			t.Fatal("response forged location or callback ownership leaked", string(fields["location"]), decodeErr)
		}
	})
}

func TestComputeConsoleCompositionGuardsDiscoveryFailuresAndNoErrorFallback(t *testing.T) {
	t.Run("discovery403 stops before root fallback and action", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := flavorIdentityClient(cloud)
		client.Endpoint = cloud.Server.URL + "/reverse/nova/v2.1/project/"
		var calls atomic.Int32
		const body = `{"error":"discovery forbidden"}`
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			th.TestMethod(t, r, http.MethodGet)
			if r.URL.Path != computeConsoleDiscoveryPath {
				t.Error("forbidden discovery fell back", r.URL)
			}
			w.Header().Set("X-Console-Proof", "discovery403")
			testcloud.JSON(w, 403, body)
		})
		value, err := compute.New(client, compute.Dependencies{}).CreateConsole(context.Background(), "server-id", "novnc")
		var original gophercloud.ErrUnexpectedResponseCode
		if value != nil || !errors.As(err, &original) || original.Actual != 403 || original.Method != http.MethodGet || original.URL != cloud.Server.URL+computeConsoleDiscoveryPath || string(original.Body) != body || original.ResponseHeader.Get("X-Console-Proof") != "discovery403" || calls.Load() != 1 {
			t.Fatal("discovery HTTP evidence lost or another endpoint attempted", string(value), err, original, calls.Load())
		}
	})
	for _, mode := range []string{"option source", "option cancellation", "location error", "location source", "service API", "service Servers", "remote API"} {
		t.Run("guard "+mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Endpoint = cloud.Server.URL + "/reverse/nova/v2.1/project/"
			client.Microversion = "2.6"
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause, original := errors.New("console guard cause"), errors.New("original option cause")
			var gets, posts, later atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					gets.Add(1)
					testcloud.JSON(w, 200, `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.100"}}`)
					return
				}
				posts.Add(1)
				w.WriteHeader(500)
			})
			var service *compute.Service
			dependencies := compute.Dependencies{}
			if mode == "location error" || mode == "location source" {
				dependencies.CloudLocation = func() (resource.CloudLocation, error) {
					if mode == "location source" {
						client.ResourceBase = cloud.Server.URL + "/changed/"
						return resource.CloudLocation{}, nil
					}
					return resource.CloudLocation{}, cause
				}
			}
			service = compute.New(client, dependencies)
			option := func(_ *request.Config[compute.ConsoleOpts]) error {
				switch mode {
				case "option source":
					client.ResourceBase = cloud.Server.URL + "/changed/"
				case "option cancellation":
					cancel(cause)
					return original
				case "service API":
					service.API = compute.New(client, compute.Dependencies{}).API
				case "service Servers":
					service.Servers = compute.New(client, compute.Dependencies{}).Servers
				case "remote API":
					service.API.RemoteConsoles = compute.New(client, compute.Dependencies{}).API.RemoteConsoles
				}
				return nil
			}
			value, err := service.CreateConsole(ctx, "server-id", "novnc", option, func(_ *request.Config[compute.ConsoleOpts]) error { later.Add(1); return nil })
			if value != nil || err == nil || posts.Load() != 0 {
				t.Fatal("guard failure performed console mutation", string(value), err, gets.Load(), posts.Load())
			}
			if mode == "location error" || mode == "location source" {
				if gets.Load() != 1 || later.Load() != 1 {
					t.Fatal(gets.Load(), later.Load())
				}
			} else if gets.Load() != 0 || later.Load() != 0 {
				t.Fatal("later callback/discovery ran after invalid source", gets.Load(), later.Load())
			}
			if mode == "option cancellation" {
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, original) {
					t.Fatal(err)
				}
			} else if mode == "location error" {
				if !errors.Is(err, cause) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
	for _, mode := range []string{"read404", "close404", "source after discovery", "cancel after discovery", "empty204"} {
		t.Run("accepted discovery "+mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Endpoint = cloud.Server.URL + "/reverse/nova/v2.1/project/"
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("discovery canceled")
			body, code := `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.100"}}`, 300
			if mode == "empty204" {
				body, code = "", 204
			}
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{200, 300}, Method: http.MethodGet, URL: "nested", Body: []byte("original nested discovery404"), ResponseHeader: http.Header{"X-Nested-Proof": {"actual"}}}
			var track *payloadContractTracking
			if mode == "read404" {
				track = payloadContractTrack(cloud, nested, nil)
			} else if mode == "close404" {
				track = payloadContractTrack(cloud, nil, nested)
			} else if mode != "empty204" {
				base := cloud.Provider.HTTPClient.Transport
				if base == nil {
					base = http.DefaultTransport
				}
				cloud.Provider.HTTPClient.Transport = secretFetchRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					response, err := base.RoundTrip(r)
					if err == nil {
						if mode == "source after discovery" {
							client.ResourceBase = cloud.Server.URL + "/changed/"
						} else {
							cancel(cause)
						}
					}
					return response, err
				})
			}
			var gets, posts, retries atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					gets.Add(1)
					th.TestMethod(t, r, http.MethodGet)
					if r.URL.Path != computeConsoleDiscoveryPath {
						t.Error(r.URL)
					}
					w.Header().Set("X-Console-Proof", mode)
					testcloud.JSON(w, code, body)
					return
				}
				posts.Add(1)
				w.WriteHeader(500)
			})
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries.Add(1)
				return err
			}
			value, err := compute.New(client, compute.Dependencies{}).CreateConsole(ctx, "server-id", "novnc")
			var proof *resource.ResponseError
			if value != nil || !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != body || proof.Header.Get("X-Console-Proof") != mode || gets.Load() != 1 || posts.Load() != 0 || retries.Load() != 0 {
				t.Fatal("accepted discovery failure fell back/replayed/lost proof", string(value), err, proof, gets.Load(), posts.Load(), retries.Load())
			}
			if mode == "source after discovery" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if mode == "cancel after discovery" {
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
					t.Fatal(err)
				}
			} else if mode != "empty204" {
				var original gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &original) || original.Actual != 404 || original.URL != "nested" || string(original.Body) != "original nested discovery404" || original.ResponseHeader.Get("X-Nested-Proof") != "actual" {
					t.Fatal(err, original)
				}
				physical := track.last(t)
				if track.calls.Load() != 1 || physical.reads.Load() == 0 || physical.closes.Load() != 1 {
					t.Fatal(track.calls.Load(), physical)
				}
			}
		})
	}
	for _, tc := range []struct {
		name, selected, body string
		status               int
		modern, failure      bool
	}{
		{"modern403 no legacy fallback", "2.6", `{"error":"modern forbidden"}`, 403, true, true},
		{"modern404 no legacy fallback", "2.6", `{"error":"modern missing"}`, 404, true, true},
		{"legacy404 no modern fallback", "2.5", `{"error":"legacy missing"}`, 404, false, true},
		{"modern opaque keeps original seed", "2.6", "accepted opaque", 202, true, false},
		{"legacy opaque remains processing error", "2.5", "accepted opaque", 202, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := flavorIdentityClient(cloud)
			client.Endpoint = cloud.Server.URL + "/reverse/nova/v2.1/project/"
			client.Microversion = tc.selected
			var gets, posts atomic.Int32
			wantPath := computeConsoleLegacyPath
			if tc.modern {
				wantPath = computeRemoteConsolePath
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					gets.Add(1)
					testcloud.JSON(w, 200, `{"version":{"id":"v2.1","status":"CURRENT","links":[{"rel":"self","href":"/reverse/nova/v2.1/"}],"min_version":"2.1","version":"2.100"}}`)
					return
				}
				posts.Add(1)
				th.TestMethod(t, r, http.MethodPost)
				if r.URL.Path != wantPath {
					t.Error("failed winning path fell back", r.URL, wantPath)
				}
				w.Header().Set("X-Console-Proof", tc.name)
				testcloud.JSON(w, tc.status, tc.body)
			})
			value, err := compute.New(client, compute.Dependencies{}).CreateConsole(context.Background(), "server-id", "novnc")
			if (err != nil) != tc.failure || gets.Load() != 1 || posts.Load() != 1 {
				t.Fatal(string(value), err, gets.Load(), posts.Load())
			}
			if tc.status >= 400 {
				var original gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.As(err, &original) || original.Actual != tc.status || original.Method != http.MethodPost || original.URL != cloud.Server.URL+wantPath || string(original.Body) != tc.body || original.ResponseHeader.Get("X-Console-Proof") != tc.name || len(original.Expected) != 200 {
					t.Fatal("original winning HTTP proof lost", string(value), err, original)
				}
			} else if tc.failure {
				var proof *resource.ResponseError
				if value != nil || !errors.As(err, &proof) || proof.StatusCode != tc.status || string(proof.Body) != tc.body || proof.Header.Get("X-Console-Proof") != tc.name {
					t.Fatal("legacy processing proof lost", string(value), err, proof)
				}
			} else {
				var fields map[string]json.RawMessage
				if decodeErr := json.Unmarshal(value, &fields); decodeErr != nil || len(fields) != 6 || string(fields["type"]) != `"novnc"` || string(fields["protocol"]) != `null` || string(fields["url"]) != `null` || string(fields["location"]) != `null` {
					t.Fatal("modern malformed-body tolerance changed", string(value), decodeErr)
				}
			}
		})
	}
}
