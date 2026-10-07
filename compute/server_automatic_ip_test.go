package compute_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/compute"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const autoPort = `{"id":"port","device_id":"server","network_id":"private","fixed_ips":[{"ip_address":"10.0.0.10"}]}`
const autoFixed = `{"private":[{"version":4,"addr":"10.0.0.10","OS-EXT-IPS:type":"fixed"}]}`
const autoFloating = `{"private":[{"version":4,"addr":"8.8.8.8","OS-EXT-IPS:type":"floating"}]}`

func automaticServer(t *testing.T, addresses string) *compute.Server {
	t.Helper()
	var server compute.Server
	if err := json.Unmarshal([]byte(`{"id":"server","status":"ACTIVE","addresses":`+addresses+`}`), &server); err != nil {
		t.Fatal(err)
	}
	return &server
}

type automaticFixture struct {
	cloud                                            *testcloud.Cloud
	service                                          *compute.Service
	network                                          *network.Service
	ports, roles, ips, posts, raw                    atomic.Int32
	ipListBody, portRows, networks, postBody, ipBody string
	roleStatus, portStatus, ipListStatus             int
	rawBody                                          func(int32) (int, string)
}

func newAutomaticFixture(t *testing.T) *automaticFixture {
	t.Helper()
	f := &automaticFixture{cloud: testcloud.New(t), portRows: autoPort, roleStatus: 200, portStatus: 200, ipListStatus: 200,
		networks: `[{"id":"external","name":"external","router:external":true},{"id":"private","name":"private"}]`,
		postBody: `{"floatingip":{"id":"ip","floating_network_id":"external","port_id":"port","fixed_ip_address":"10.0.0.10","floating_ip_address":"8.8.8.8","status":"DOWN"}}`,
		ipBody:   `{"floatingip":{"id":"ip","floating_network_id":"external","port_id":"port","fixed_ip_address":"10.0.0.10","floating_ip_address":"8.8.8.8","status":"ACTIVE"}}`,
	}
	f.rawBody = func(int32) (int, string) {
		return 200, `{"server":{"id":"server","status":"ACTIVE","addresses":` + autoFloating + `}}`
	}
	policy, err := network.PrepareNetworkRoleOptions(network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "private", NATDestination: true}))
	if err != nil {
		t.Fatal(err)
	}
	f.network = network.NewWithDependencies(f.cloud.Client("network", "/v2.0"), network.Dependencies{NetworkRoles: policy})
	f.service = compute.New(f.cloud.Client("compute", "/v2.1"), compute.Dependencies{NetworkPolicy: policy,
		AddressNetworks: func(context.Context) (*network.Service, error) { return f.network, nil },
		NetworkRoles: func(context.Context) (*network.NetworkRoleSnapshot, error) {
			t.Error("unguarded role loader called")
			return nil, errors.New("wrong loader")
		},
	})
	f.cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		f.ports.Add(1)
		if r.URL.Query().Get("device_id") != "server" || r.URL.Query().Get("network_id") != "" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, f.portStatus, `{"ports":[`+f.portRows+`]}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.0/ports/port", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"port":`+autoPort+`}`) })
	f.cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		f.roles.Add(1)
		testcloud.JSON(w, f.roleStatus, `{"networks":`+f.networks+`}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.0/routers", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"routers":[]}`) })
	f.cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		f.ips.Add(1)
		if r.URL.Query().Get("port_id") != "port" {
			t.Error(r.URL)
		}
		body := `{"floatingips":[]}`
		if f.ipListBody != "" {
			body = f.ipListBody
		}
		testcloud.JSON(w, f.ipListStatus, body)
	})
	f.cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		f.posts.Add(1)
		var body map[string]map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		ip := body["floatingip"]
		if ip["port_id"] != "port" || ip["floating_network_id"] != "external" || ip["fixed_ip_address"] != "10.0.0.10" {
			t.Error(body)
		}
		testcloud.JSON(w, 201, f.postBody)
	})
	f.cloud.Mux.HandleFunc("GET /v2.0/floatingips/ip", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, f.ipBody) })
	f.cloud.Mux.HandleFunc("GET /v2.1/servers/server", func(w http.ResponseWriter, r *http.Request) {
		code, body := f.rawBody(f.raw.Add(1))
		testcloud.JSON(w, code, body)
	})
	f.cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	return f
}

func automaticOptions() []compute.AutomaticFloatingIPOption {
	return []compute.AutomaticFloatingIPOption{
		compute.WithAutomaticAddressOptions(compute.WithAddressReachability(false)),
		compute.WithAutomaticEnsureOptions(network.WithEnsureReuse(false), network.WithEnsureWait(resource.WithPollInterval(time.Millisecond))),
		compute.WithAutomaticIPPollInterval(time.Millisecond), compute.WithAutomaticIPTimeout(time.Second),
	}
}

func TestAutomaticIPKnownSkipsAvoidAllServiceAndOwnerWork(t *testing.T) {
	for _, scenario := range []string{"disabled", "none", "private nil", "access malformed", "floating6", "floating empty", "empty map", "null rows", "empty rows"} {
		t.Run(scenario, func(t *testing.T) {
			calls := 0
			service := compute.New(nil, compute.Dependencies{
				AddressNetworks: func(context.Context) (*network.Service, error) { calls++; return nil, errors.New("must skip network") },
				AddressCompute: func(context.Context) (*gophercloud.ServiceClient, error) {
					calls++
					return nil, errors.New("must skip compute")
				},
				NetworkRoles: func(context.Context) (*network.NetworkRoleSnapshot, error) {
					calls++
					return nil, errors.New("must skip roles")
				},
			})
			server := automaticServer(t, `{}`)
			want := compute.AutomaticIPNoFixedAddress
			var options []compute.AutomaticFloatingIPOption
			switch scenario {
			case "disabled":
				options = append(options, compute.WithAutomaticIPEnabled(false))
				server.Addresses = nil
				want = compute.AutomaticIPDisabled
			case "none":
				options = append(options, compute.WithAutomaticAddressOptions(compute.WithFloatingIPSource(compute.FloatingIPNone)))
				server.Addresses = nil
				want = compute.AutomaticIPDisabled
			case "private nil":
				options = append(options, compute.WithAutomaticAddressOptions(compute.WithPrivateCloud(true)))
				server.Addresses = nil
				want = compute.AutomaticIPPrivateCloud
			case "access malformed":
				server.AccessIPv4 = "8.8.8.8"
				server.Addresses = map[string]any{"invalid": true}
				want = compute.AutomaticIPExistingPublicIPv4
			case "floating6":
				server = automaticServer(t, `{"net":[{"version":6,"addr":"2001:db8::1","OS-EXT-IPS:type":"floating"}]}`)
				want = compute.AutomaticIPExistingFloating
			case "floating empty":
				server = automaticServer(t, `{"net":[{"version":5,"addr":"","OS-EXT-IPS:type":"floating"}]}`)
				want = compute.AutomaticIPExistingFloating
			case "null rows":
				server = automaticServer(t, `{"net":null}`)
			case "empty rows":
				server = automaticServer(t, `{"net":[]}`)
			}
			result, err := service.EnsureServerFloatingIP(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, options...)
			if err != nil || result == nil || result.Decision.Needed || result.Decision.Reason != want || result.Assignment != nil || result.Observed || calls != 0 {
				t.Fatal(result, err, calls)
			}
		})
	}
}

func TestAutomaticIPNilOnlyRefreshPreservesActualEvidence(t *testing.T) {
	for _, scenario := range []string{"floating", "still nil", "access nil", "203", "404", "malformed", "wrong ID", "ERROR"} {
		t.Run(scenario, func(t *testing.T) {
			f := newAutomaticFixture(t)
			f.rawBody = func(int32) (int, string) {
				code, body := 200, `{"server":{"id":"server","status":"ACTIVE","addresses":`+autoFloating+`}}`
				switch scenario {
				case "still nil":
					body = `{"server":{"id":"server","status":"ACTIVE","addresses":null}}`
				case "access nil":
					body = `{"server":{"id":"server","status":"ACTIVE","accessIPv4":"8.8.8.8","addresses":null}}`
				case "203":
					code = 203
				case "404":
					code, body = 404, `{"error":"missing"}`
				case "malformed":
					body = `{"server":{"id":"server","status":"ACTIVE","addresses":{"net":[{"addr":"8.8.8.8"}]}}}`
				case "wrong ID":
					body = strings.Replace(body, `"id":"server"`, `"id":"other"`, 1)
				case "ERROR":
					body = `{"server":{"id":"server","status":"ERROR","fault":{"message":"boot failed"},"addresses":{}}}`
				}
				return code, body
			}
			result, err := f.service.EnsureServerFloatingIP(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, `null`)}, automaticOptions()...)
			failed := scenario == "404" || scenario == "malformed" || scenario == "wrong ID" || scenario == "ERROR"
			if (err != nil) != failed || result == nil || result.Assignment != nil || f.raw.Load() != 1 || f.ports.Load() != 0 || f.roles.Load() != 0 || f.posts.Load() != 0 {
				t.Fatal(result, err, f.raw.Load(), f.ports.Load())
			}
			if scenario == "ERROR" && (!errors.Is(err, resource.ErrFailedState) || result.Server.Fault.Message != "boot failed") {
				t.Fatal(result, err)
			}
			if scenario == "malformed" {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 || result.Server.Addresses == nil {
					t.Fatal(result, err, proof)
				}
			}
			if scenario == "access nil" && result.Decision.Reason != compute.AutomaticIPExistingPublicIPv4 {
				t.Fatal(result.Decision)
			}
		})
	}
}

func TestAutomaticIPBackendAndCompletedAbsenceStayDistinct(t *testing.T) {
	for _, scenario := range []string{"missing catalog", "Nova network", "no external", "no ports", "IPv6 only", "late denied"} {
		t.Run(scenario, func(t *testing.T) {
			f := newAutomaticFixture(t)
			server := automaticServer(t, autoFixed)
			server.Status = "BUILD"
			options := automaticOptions()
			switch scenario {
			case "missing catalog":
				server.Status = "ACTIVE"
				f.service = compute.New(nil, compute.Dependencies{AddressNetworks: func(context.Context) (*network.Service, error) { return nil, nil }})
			case "Nova network":
				options = append(options, compute.WithAutomaticAddressOptions(compute.WithFloatingIPSource(compute.FloatingIPNova)))
			case "no external":
				f.networks = `[{"id":"private","name":"private"}]`
			case "no ports":
				f.portRows = ""
			case "IPv6 only":
				server = automaticServer(t, `{"private":[{"version":6,"addr":"2001:db8::1","OS-EXT-IPS:type":"fixed"}]}`)
				server.Status = "BUILD"
				f.portRows = strings.ReplaceAll(autoPort, "10.0.0.10", "2001:db8::1")
			case "late denied":
				f.portStatus = 403
			}
			result, err := f.service.EnsureServerFloatingIP(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, options...)
			if result == nil || f.posts.Load() != 0 || f.raw.Load() != 0 {
				t.Fatal(result, err)
			}
			switch scenario {
			case "missing catalog":
				if !errors.Is(err, resource.ErrUnsupported) || !result.Decision.Needed || result.Decision.Backend != compute.FloatingIPNova {
					t.Fatal(result, err)
				}
			case "Nova network":
				if !errors.Is(err, resource.ErrInvalidOption) || !result.Decision.Needed || result.Decision.Backend != compute.FloatingIPNova {
					t.Fatal(result, err)
				}
			case "no external", "no ports":
				if err != nil || result.Decision.Needed || result.Decision.Reason == compute.AutomaticIPUndetermined {
					t.Fatal(result, err)
				}
			default:
				if err == nil || result.Decision.Reason != compute.AutomaticIPUndetermined {
					t.Fatal(result, err)
				}
			}
		})
	}
}

func TestAutomaticIPSupplementedExistingAddressSkipsAndFailuresStayVisible(t *testing.T) {
	for _, denied := range []bool{false, true} {
		t.Run(fmt.Sprint(denied), func(t *testing.T) {
			f := newAutomaticFixture(t)
			f.ipListBody = `{"floatingips":[{"port_id":"port","fixed_ip_address":"10.0.0.10","floating_ip_address":"8.8.8.8"}]}`
			if denied {
				f.ipListStatus = 403
			}

			result, err := f.service.EnsureServerFloatingIP(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, automaticOptions()...)
			if (err != nil) != denied || result == nil || result.Assignment != nil || result.Observed || f.posts.Load() != 0 || f.roles.Load() != 0 || f.raw.Load() != 0 || f.ips.Load() != 1 {
				t.Fatal(result, err)
			}
			if !denied && (result.Decision.Reason != compute.AutomaticIPExistingFloating || !result.Decision.Addresses.Addresses["private"][1].Supplemental) {
				t.Fatal(result.Decision)
			}
			if denied && result.Decision.Addresses.SupplementalError == nil {
				t.Fatal("lost supplement diagnostic")
			}
		})
	}
}

func TestAutomaticIPExecutesFixedTargetAndObservesSecondRawFloatingIPv4(t *testing.T) {
	f := newAutomaticFixture(t)
	f.rawBody = func(n int32) (int, string) {
		rows := `{"version":4,"addr":"9.9.9.9","OS-EXT-IPS:type":"floating"}`
		if n == 2 {
			rows += `,{"version":4,"addr":"8.8.8.8","OS-EXT-IPS:type":"floating"}`
		}
		return 200, `{"server":{"id":"server","status":"ACTIVE","name":"latest","addresses":{"private":[` + rows + `]}}}`
	}
	server := automaticServer(t, autoFixed)
	before, _ := json.Marshal(server)
	progress := 0
	options := append(automaticOptions(), compute.WithAutomaticIPProgress(func(value *compute.Server) error {
		progress++
		server.ID = "caller-mutated"
		value.ID, value.Name = "callback-mutated", "callback-mutated"
		value.Addresses["private"] = []any{}
		f.network.Roles.Reset()
		return nil
	}))
	result, err := f.service.EnsureServerFloatingIP(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, options...)
	server.ID = "server"
	after, _ := json.Marshal(server)
	if err != nil || result == nil || !result.Observed || result.Server.ID != "server" || result.Server.Name != "latest" || !result.Assignment.Allocated || result.Decision.Selection.PortID != "port" || f.posts.Load() != 1 || f.raw.Load() != 2 || f.ports.Load() != 2 || f.roles.Load() != 1 || progress != 1 || string(before) != string(after) {
		t.Fatal(result, err, f.posts.Load(), f.raw.Load(), progress)
	}
}

func TestAutomaticIPRawNonEvidenceCancelRetainsAssignmentAndLastServer(t *testing.T) {
	for _, scenario := range []string{"fixed target", "floating6 target", "other floating", "AccessIPv4 target", "BUILD target", "empty", "ERROR", "404", "malformed"} {
		t.Run(scenario, func(t *testing.T) {
			f := newAutomaticFixture(t)
			f.rawBody = func(int32) (int, string) {
				status, addresses, access, code := "ACTIVE", autoFixed, "", 200
				switch scenario {
				case "fixed target":
					addresses = strings.ReplaceAll(autoFloating, `"floating"`, `"fixed"`)
				case "floating6 target":
					addresses = strings.ReplaceAll(autoFloating, `"version":4`, `"version":6`)
				case "other floating":
					addresses = strings.ReplaceAll(autoFloating, "8.8.8.8", "9.9.9.9")
				case "AccessIPv4 target":
					access = `,"accessIPv4":"8.8.8.8"`
				case "BUILD target":
					status, addresses = "BUILD", autoFloating
				case "empty":
					addresses = `null`
				case "ERROR":
					status = "ERROR"
				case "404":
					code = 404
				case "malformed":
					addresses = `{"net":[{"addr":"8.8.8.8"}]}`
				}
				return code, `{"server":{"id":"server","status":"` + status + `","name":"last raw","fault":{"message":"raw fault"},"addresses":` + addresses + access + `}}`
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("stop observation")
			options := append(automaticOptions(), compute.WithAutomaticIPProgress(func(value *compute.Server) error { value.Name = "callback-mutated"; cancel(cause); return nil }))
			result, err := f.service.EnsureServerFloatingIP(ctx, compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, options...)
			if err == nil || result == nil || result.Observed || result.Assignment == nil || !result.Assignment.Allocated || result.Assignment.FloatingIP.ID != "ip" || f.raw.Load() != 1 || f.posts.Load() != 1 {
				t.Fatal(result, err)
			}
			if scenario != "404" && (result.Server.Name != "last raw" || result.Server.Fault.Message != "raw fault") {
				t.Fatal(result.Server)
			}
			if scenario != "ERROR" && scenario != "404" && scenario != "malformed" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
				t.Fatal(err)
			}
			if scenario == "ERROR" && !errors.Is(err, resource.ErrFailedState) {
				t.Fatal(err)
			}
		})
	}
}

func TestAutomaticIPAcceptedAllocationAndActualIPReadinessKeepPartialResources(t *testing.T) {
	for _, scenario := range []string{"malformed allocation", "false status"} {
		t.Run(scenario, func(t *testing.T) {
			f := newAutomaticFixture(t)
			options := automaticOptions()
			if scenario == "malformed allocation" {
				f.postBody = `{"floatingip":`
			} else {
				f.ipBody = strings.ReplaceAll(f.ipBody, `"status":"ACTIVE"`, `"status":"DOWN","description":"ACTIVE"`)
				options = append(options, compute.WithAutomaticEnsureOptions(network.WithEnsureWait(resource.WithStatusAttribute("description"))))
			}
			result, err := f.service.EnsureServerFloatingIP(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, options...)
			if err == nil || result == nil || result.Server.ID != "server" || result.Assignment == nil || !result.Assignment.Allocated || result.Observed || f.posts.Load() != 1 || f.raw.Load() != 0 {
				t.Fatal(result, err)
			}
			if scenario == "malformed allocation" {
				var proof *resource.ResponseError
				if result.Assignment.FloatingIP != nil || !errors.As(err, &proof) || proof.StatusCode != 201 {
					t.Fatal(result, err, proof)
				}
			}
		})
	}
}

func TestAutomaticIPColdRoleRetryChecksOuterComputeSource(t *testing.T) {
	f := newAutomaticFixture(t)
	f.roleStatus = 503
	f.cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		f.service.RawClient().Endpoint = f.cloud.Server.URL + "/replacement/"
		return nil
	}
	server := automaticServer(t, autoFixed)
	server.Status = "BUILD"
	decision, err := f.service.PlanServerFloatingIP(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, automaticOptions()...)
	var proof gophercloud.ErrUnexpectedResponseCode
	if decision == nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.Actual != 503 || f.roles.Load() != 1 || f.ports.Load() != 0 || f.posts.Load() != 0 {
		t.Fatal(decision, err, proof, f.roles.Load())
	}
}

func TestAutomaticIPStaticValidationAndDeadlineDoNotPermitMutation(t *testing.T) {
	f := newAutomaticFixture(t)
	for _, option := range []compute.AutomaticFloatingIPOption{nil, compute.WithAutomaticIPTimeout(0), compute.WithAutomaticIPPollInterval(0), compute.WithAutomaticIPProgress(nil), compute.WithAutomaticEnsureOptions(network.WithEnsureWait(resource.WithTimeout(-time.Second)))} {
		result, err := f.service.EnsureServerFloatingIP(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, option)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || f.posts.Load() != 0 || f.ports.Load() != 0 {
			t.Fatal(result, err)
		}
	}
	f.rawBody = func(int32) (int, string) {
		return 200, `{"server":{"id":"server","status":"ACTIVE","addresses":` + autoFixed + `}}`
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	options := append(automaticOptions(), compute.WithUnlimitedAutomaticIPTimeout(), compute.WithAutomaticIPPollInterval(time.Second))
	result, err := f.service.EnsureServerFloatingIP(ctx, compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, options...)
	if result == nil || result.Assignment == nil || !errors.Is(err, context.DeadlineExceeded) || result.Observed || f.posts.Load() != 1 || f.raw.Load() != 1 {
		t.Fatal(result, err)
	}
}
