package network_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func roleFloatingIPCall(ctx context.Context, s *network.Service, kind string, external resource.Ref, create []network.CreateFloatingIPOption, ensure []network.EnsureFloatingIPOption) (*network.FloatingIP, error) {
	if kind == "Create" {
		return s.FloatingIPs.Create(ctx, network.CreateFloatingIPRequest{Network: external}, append([]network.CreateFloatingIPOption{network.WithServer(resource.ID("server"))}, create...)...)
	}
	result, err := s.FloatingIPs.Ensure(ctx, network.EnsureFloatingIPRequest{Server: resource.ID("server"), Network: external}, append([]network.EnsureFloatingIPOption{network.WithEnsureProject("owner"), network.WithEnsureReuse(false)}, ensure...)...)
	if result == nil {
		return nil, err
	}
	return result.FloatingIP, err
}

func TestFloatingIPEnsureSharedSourceCacheAndReset(t *testing.T) {
	for _, configured := range []bool{false, true} {
		t.Run(fmt.Sprintf("configured=%t", configured), func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, posts atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				generation := lists.Add(1)
				if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, fmt.Sprintf(`{"networks":[{"id":"physical","provider:physical_network":"uplink"},{"id":"external-%d","name":"public","router:external":%t},{"id":"other","router:external":true},{"id":"private","name":"private"}]}`, generation, !configured))
			})
			ensurePortFixture(t, cloud)
			cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				call := posts.Add(1)
				generation := 1
				if call == 3 {
					generation = 2
				}
				external := fmt.Sprintf("external-%d", generation)
				body := floatingIPBody(t, r)
				if body["floating_network_id"] != external || body["port_id"] != "port" || body["fixed_ip_address"] != "10.0.0.10" || body["project_id"] != "owner" {
					t.Error(body)
				}
				respondEnsuredFloatingIP(w, 201, "port", "10.0.0.10", "DOWN", map[string]any{"floating_network_id": external})
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected subnet/router/reuse/cleanup request: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			rows := []network.ConfiguredNetwork{{Name: "private", NATDestination: true}}
			if configured {
				rows = append(rows, network.ConfiguredNetwork{Name: "public", NATSource: true, RoutesIPv4Externally: true})
			}
			policy, err := network.PrepareNetworkRoleOptions(network.WithConfiguredNetworks(rows...))
			if err != nil {
				t.Fatal(err)
			}
			s := network.NewWithDependencies(cloud.Client("network", "/v2.0"), network.Dependencies{NetworkRoles: policy})
			// Preparation owns policy and project without resolving any role.
			prepared, err := s.FloatingIPs.PrepareEnsureActive(context.Background(), network.WithEnsureProject("owner"))
			if err != nil || lists.Load() != 0 {
				t.Fatal(prepared, err, lists.Load())
			}
			roles, err := s.Roles.Discover(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			roles.ExternalIPv4Floating[0].ID = "caller-changed"
			for range 2 {
				if _, err := roleFloatingIPCall(context.Background(), s, "Ensure", resource.Ref{}, nil, nil); err != nil {
					t.Fatal(err)
				}
			}
			if lists.Load() != 1 {
				t.Fatal(lists.Load())
			}
			s.Roles.Reset()
			if _, err := roleFloatingIPCall(context.Background(), s, "Ensure", resource.Ref{}, nil, nil); err != nil || lists.Load() != 2 || posts.Load() != 3 {
				t.Fatal(err, lists.Load(), posts.Load())
			}
		})
	}
}

func TestFloatingIPEnsureRoleErrorsStopRouterFallbackAndRetry(t *testing.T) {
	for _, scenario := range []string{"network late403", "subnet late403", "configured source missing"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			var fail atomic.Bool
			fail.Store(true)
			var posts atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				if fail.Load() && scenario == "network late403" {
					if r.URL.Query().Get("marker") == "last" {
						testcloud.JSON(w, 403, `{"error":{"message":"networks denied"}}`)
					} else {
						testcloud.JSON(w, 200, fmt.Sprintf(`{"networks":[{"id":"external","name":"public","router:external":true}],"networks_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2.0/networks?marker=last"))
					}
					return
				}
				if fail.Load() && scenario == "configured source missing" {
					testcloud.JSON(w, 200, `{"networks":[{"id":"other","router:external":true}]}`)
					return
				}
				testcloud.JSON(w, 200, `{"networks":[{"id":"external","name":"public","router:external":true}]}`)
			})
			cloud.Mux.HandleFunc("GET /v2.0/subnets", func(w http.ResponseWriter, r *http.Request) {
				if fail.Load() && scenario == "subnet late403" {
					if r.URL.Query().Get("marker") == "last" {
						testcloud.JSON(w, 403, `{"error":{"message":"subnets denied"}}`)
					} else {
						testcloud.JSON(w, 200, fmt.Sprintf(`{"subnets":[],"subnets_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2.0/subnets?marker=last"))
					}
					return
				}
				testcloud.JSON(w, 200, `{"subnets":[]}`)
			})
			cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
				if fail.Load() {
					t.Error("port lookup after failed role discovery")
				}
				testcloud.JSON(w, 200, `{"ports":[{"id":"port","device_id":"server","fixed_ips":[{"ip_address":"10.0.0.10"}]}]}`)
			})
			cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				respondEnsuredFloatingIP(w, 201, "port", "10.0.0.10", "DOWN")
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected router/mutation/cleanup request: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			var opts []network.NetworkRoleOption
			if scenario == "configured source missing" {
				opts = append(opts, network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "public", NATSource: true}))
			}
			policy, err := network.PrepareNetworkRoleOptions(opts...)
			if err != nil {
				t.Fatal(err)
			}
			s := network.NewWithDependencies(cloud.Client("network", "/v2.0"), network.Dependencies{NetworkRoles: policy})
			row, err := roleFloatingIPCall(context.Background(), s, "Ensure", resource.Ref{}, nil, nil)
			if row != nil || err == nil || posts.Load() != 0 {
				t.Fatal(row, err, posts.Load())
			}
			if scenario == "configured source missing" {
				if !errors.Is(err, resource.ErrNotFound) {
					t.Fatal(err)
				}
			} else {
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) || response.Actual != 403 {
					t.Fatal(err)
				}
			}
			fail.Store(false)
			if _, err := roleFloatingIPCall(context.Background(), s, "Ensure", resource.Ref{}, nil, nil); err != nil || posts.Load() != 1 {
				t.Fatal(err, posts.Load())
			}
		})
	}
}

func TestFloatingIPAutomaticNATUsesSharedDestinationAndReset(t *testing.T) {
	for _, kind := range []string{"Create", "Ensure"} {
		for _, configured := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/configured=%t", kind, configured), func(t *testing.T) {
				cloud := testcloud.New(t)
				var lists, subnets, portPages, posts atomic.Int32
				cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
					call := lists.Add(1)
					rows := `[{"id":"a","name":"other"},{"id":"b","name":"chosen"}]`
					if call == 2 {
						rows = `[{"id":"b","name":"other"},{"id":"a","name":"chosen"}]`
					}
					if r.URL.RawQuery != "" {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, `{"networks":`+rows+`}`)
				})
				cloud.Mux.HandleFunc("GET /v2.0/subnets", func(w http.ResponseWriter, r *http.Request) {
					subnets.Add(1)
					testcloud.JSON(w, 200, `{"subnets":[{"network_id":"a","gateway_ip":"10.0.0.254"},{"network_id":"b","gateway_ip":"10.0.1.254"}]}`)
				})
				cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
					portPages.Add(1)
					if r.URL.Query().Get("marker") == "last" {
						testcloud.JSON(w, 200, `{"ports":[{"id":"port-b","device_id":"server","network_id":"b","fixed_ips":[{"ip_address":"10.0.1.10"}]}]}`)
						return
					}
					if r.URL.Query().Get("device_id") != "server" || r.URL.Query().Get("network_id") != "" {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, fmt.Sprintf(`{"ports":[{"id":"foreign","device_id":"other","network_id":"b","fixed_ips":[{"ip_address":"10.0.1.99"}]},{"id":"port-a","device_id":"server","network_id":"a","fixed_ips":[{"ip_address":"10.0.0.10"}]}],"ports_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2.0/ports?marker=last"))
				})
				cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
					call := posts.Add(1)
					port, ip := "port-b", "10.0.1.10"
					if call == 3 {
						port, ip = "port-a", "10.0.0.10"
					}
					body := floatingIPBody(t, r)
					if portPages.Load() != call*2 || body["floating_network_id"] != "external" || body["port_id"] != port || body["fixed_ip_address"] != ip {
						t.Error(body, portPages.Load())
					}
					respondEnsuredFloatingIP(w, 201, port, ip, "DOWN")
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("unexpected reuse/cleanup %s %s", r.Method, r.URL)
					http.Error(w, "unexpected", 500)
				})
				var options []network.NetworkRoleOption
				if configured {
					options = append(options, network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "chosen", NATDestination: true}))
				}
				policy, err := network.PrepareNetworkRoleOptions(options...)
				if err != nil {
					t.Fatal(err)
				}
				s := network.NewWithDependencies(cloud.Client("network", "/v2.0"), network.Dependencies{NetworkRoles: policy})
				for range 2 {
					if _, err := roleFloatingIPCall(context.Background(), s, kind, resource.ID("external"), nil, nil); err != nil {
						t.Fatal(err)
					}
				}
				if lists.Load() != 1 {
					t.Fatal(lists.Load())
				}
				roles, err := s.Roles.Discover(context.Background())
				if err != nil || roles.NATDestination.ID != "b" {
					t.Fatal(roles, err)
				}
				roles.NATDestination.ID = "caller-changed"
				s.Roles.Reset()
				if _, err := roleFloatingIPCall(context.Background(), s, kind, resource.ID("external"), nil, nil); err != nil || lists.Load() != 2 || posts.Load() != 3 {
					t.Fatal(err, lists.Load(), posts.Load())
				}
				wantSubnets := int32(2)
				if configured {
					wantSubnets = 0
				}
				if subnets.Load() != wantSubnets {
					t.Fatal(subnets.Load(), wantSubnets)
				}
			})
		}
	}
}

func TestFloatingIPAutomaticNATFailuresStopAllocation(t *testing.T) {
	for _, kind := range []string{"Create", "Ensure"} {
		for _, scenario := range []string{"no NAT IPv6 second", "selected network absent from ports", "multiple selected IPv4", "role403", "late port403", "subnet cancellation"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var lists, posts atomic.Int32
				cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
					if scenario == "late port403" {
						if r.URL.Query().Get("marker") == "last" {
							testcloud.JSON(w, 403, `{"error":{"message":"ports denied"}}`)
						} else {
							testcloud.JSON(w, 200, fmt.Sprintf(`{"ports":[{"id":"port-a","device_id":"server","network_id":"a","fixed_ips":[{"ip_address":"10.0.0.10"}]}],"ports_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2.0/ports?marker=last"))
						}
						return
					}
					second := `[{"ip_address":"10.0.1.10"}]`
					if scenario == "no NAT IPv6 second" {
						second = `[{"ip_address":"2001:db8::1"}]`
					} else if scenario == "multiple selected IPv4" {
						second = `[{"ip_address":"10.0.1.10"},{"ip_address":"10.0.1.11"}]`
					}
					testcloud.JSON(w, 200, `{"ports":[{"id":"port-a","device_id":"server","network_id":"a","fixed_ips":[{"ip_address":"10.0.0.10"}]},{"id":"port-b","device_id":"server","network_id":"b","fixed_ips":`+second+`}]}`)
				})
				cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					if scenario == "role403" {
						testcloud.JSON(w, 403, `{"error":{"message":"networks denied"}}`)
						return
					}
					testcloud.JSON(w, 200, `{"networks":[{"id":"a"},{"id":"b"},{"id":"absent"}]}`)
				})
				cloud.Mux.HandleFunc("GET /v2.0/subnets", func(w http.ResponseWriter, r *http.Request) {
					if scenario == "subnet cancellation" {
						cancel()
					}
					gateway := `[{"network_id":"b","gateway_ip":"10.0.1.254"}]`
					if scenario == "no NAT IPv6 second" {
						gateway = `[]`
					} else if scenario == "selected network absent from ports" {
						gateway = `[{"network_id":"absent","gateway_ip":"10.0.2.254"}]`
					}
					testcloud.JSON(w, 200, `{"subnets":`+gateway+`}`)
				})
				cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
					posts.Add(1)
					t.Error("allocated after selection failed")
					http.Error(w, "unexpected", 500)
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("unexpected fallback/reuse/cleanup %s %s", r.Method, r.URL)
					http.Error(w, "unexpected", 500)
				})
				s := network.New(cloud.Client("network", "/v2.0"))
				row, err := roleFloatingIPCall(ctx, s, kind, resource.ID("external"), nil, nil)
				if row != nil || err == nil || posts.Load() != 0 {
					t.Fatal(row, err, posts.Load())
				}
				switch scenario {
				case "no NAT IPv6 second", "multiple selected IPv4":
					if !errors.Is(err, resource.ErrAmbiguous) {
						t.Fatal(err)
					}
					if scenario == "no NAT IPv6 second" {
						var ambiguous *resource.AmbiguousError
						if !errors.As(err, &ambiguous) || !reflect.DeepEqual(ambiguous.IDs, []string{"port-a", "port-b"}) {
							t.Fatal(err)
						}
					}
				case "selected network absent from ports":
					if !errors.Is(err, resource.ErrNotFound) {
						t.Fatal(err)
					}
				case "subnet cancellation":
					if !errors.Is(err, context.Canceled) {
						t.Fatal(err)
					}
				default:
					var response gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &response) || response.Actual != 403 {
						t.Fatal(err)
					}
				}
				wantLists := int32(1)
				if scenario == "late port403" {
					wantLists = 0
				}
				if lists.Load() != wantLists {
					t.Fatal(lists.Load(), wantLists)
				}
			})
		}
	}
}

func TestFloatingIPEnsureDisabledRolesStillUseRouterFallback(t *testing.T) {
	cloud := testcloud.New(t)
	var routers, posts atomic.Int32
	cloud.Mux.HandleFunc("GET /v2.0/routers", func(w http.ResponseWriter, r *http.Request) {
		routers.Add(1)
		testcloud.JSON(w, 200, `{"routers":[{"id":"router","admin_state_up":true,"external_gateway_info":{"network_id":"external"}}]}`)
	})
	ensurePortFixture(t, cloud)
	cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		if got := floatingIPBody(t, r); got["floating_network_id"] != "external" || got["port_id"] != "port" {
			t.Error(got)
		}
		respondEnsuredFloatingIP(w, 201, "port", "10.0.0.10", "DOWN")
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected role inventory/reuse/cleanup %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	policy, err := network.PrepareNetworkRoleOptions(network.WithExternalNetworkDiscovery(false), network.WithInternalNetworkDiscovery(false), network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "missing", NATSource: true}))
	if err != nil {
		t.Fatal(err)
	}
	s := network.NewWithDependencies(cloud.Client("network", "/v2.0"), network.Dependencies{NetworkRoles: policy})
	row, err := roleFloatingIPCall(context.Background(), s, "Ensure", resource.Ref{}, nil, nil)
	if err != nil || row == nil || routers.Load() != 1 || posts.Load() != 1 {
		t.Fatal(row, err, routers.Load(), posts.Load())
	}
}

func TestFloatingIPExplicitAndSinglePortSelectionBypassRoles(t *testing.T) {
	for _, kind := range []string{"Create", "Ensure"} {
		for _, scenario := range []string{"single owned port", "fixed", "port", "NAT", "external name"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				cloud := testcloud.New(t)
				var lists, posts atomic.Int32
				port := `{"id":"port-a","device_id":"server","network_id":"a","fixed_ips":[{"ip_address":"10.0.0.10"}]}`
				cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
					if scenario == "port" {
						t.Error("listed despite explicit port")
					}
					if r.URL.Query().Get("device_id") != "server" || scenario == "NAT" && r.URL.Query().Get("network_id") != "a" {
						t.Error(r.URL)
					}
					second := `{"id":"port-b","device_id":"server","network_id":"b","fixed_ips":[{"ip_address":"10.0.1.10"}]}`
					if scenario == "single owned port" || scenario == "external name" {
						second = `{"id":"foreign","device_id":"other","fixed_ips":[{"ip_address":"10.0.1.10"}]}`
					}
					testcloud.JSON(w, 200, `{"ports":[`+port+`,`+second+`]}`)
				})
				cloud.Mux.HandleFunc("GET /v2.0/ports/port-a", func(w http.ResponseWriter, r *http.Request) {
					testcloud.JSON(w, 200, `{"port":`+port+`}`)
				})
				cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					if scenario != "external name" || r.URL.Query().Get("name") != "public" || r.URL.Query().Get("router:external") != "true" {
						t.Error("unexpected role inventory", r.URL)
					}
					testcloud.JSON(w, 200, `{"networks":[{"id":"external","name":"public","router:external":true}]}`)
				})
				cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
					posts.Add(1)
					body := floatingIPBody(t, r)
					if body["floating_network_id"] != "external" || body["port_id"] != "port-a" || body["fixed_ip_address"] != "10.0.0.10" {
						t.Error(body)
					}
					respondEnsuredFloatingIP(w, 201, "port-a", "10.0.0.10", "DOWN")
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("unexpected role/reuse/cleanup %s %s", r.Method, r.URL)
					http.Error(w, "unexpected", 500)
				})
				policy, err := network.PrepareNetworkRoleOptions(network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "missing", NATSource: true, NATDestination: true, DefaultInterface: true}))
				if err != nil {
					t.Fatal(err)
				}
				s := network.NewWithDependencies(cloud.Client("network", "/v2.0"), network.Dependencies{NetworkRoles: policy})
				var create []network.CreateFloatingIPOption
				var ensure []network.EnsureFloatingIPOption
				switch scenario {
				case "fixed":
					create = append(create, network.WithFixedAddress("10.0.0.10"))
					ensure = append(ensure, network.WithEnsureFixedAddress("10.0.0.10"))
				case "port":
					create = append(create, network.WithPort(resource.ID("port-a")))
					ensure = append(ensure, network.WithEnsurePort(resource.ID("port-a")))
				case "NAT":
					create = append(create, network.WithNATDestination(resource.ID("a")))
					ensure = append(ensure, network.WithEnsureNATDestination(resource.ID("a")))
				}
				external := resource.ID("external")
				if scenario == "external name" {
					external = resource.Name("public")
				}
				row, err := roleFloatingIPCall(context.Background(), s, kind, external, create, ensure)
				if err != nil || row == nil || row.PortID != "port-a" || posts.Load() != 1 {
					t.Fatal(row, err, posts.Load())
				}
				wantLists := int32(0)
				if scenario == "external name" {
					wantLists = 1
				}
				if lists.Load() != wantLists {
					t.Fatal(lists.Load(), wantLists)
				}
			})
		}
	}
}
