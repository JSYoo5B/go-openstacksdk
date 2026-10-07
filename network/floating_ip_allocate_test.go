package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

func TestFloatingIPAllocateOrdinaryNetworkLookupAndFreshBody(t *testing.T) {
	for _, code := range []int{200, 201, 202, 299, 399} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			var events []string
			cloud.Mux.HandleFunc("GET /v2.0/networks/public", func(w http.ResponseWriter, r *http.Request) {
				events = append(events, "member")
				testcloud.JSON(w, 404, `{}`)
			})
			cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				events = append(events, "name")
				if r.URL.RawQuery != "name=public" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"networks":[{"id":"ordinary","name":"public","router:external":false}]}`)
			})
			cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				events = append(events, "create")
				var body map[string]map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if !reflect.DeepEqual(body, map[string]map[string]string{"floatingip": {"floating_network_id": "ordinary"}}) {
					t.Error(body)
				}
				w.Header().Set("X-Allocation-Proof", "receipt")
				testcloud.JSON(w, code, `{"floatingip":{"id":"new","floating_ip_address":null,"vendor":9007199254740993}}`)
			})
			name := "public"
			result, err := network.New(cloud.Client("network", "/v2.0/")).FloatingIPs.Allocate(context.Background(), network.AllocateFloatingIPRequest{Network: &name})
			if err != nil || !result.Allocated || result.Selection.NetworkID != "ordinary" || result.Wire.StatusCode != code || result.AllocationResponse.StatusCode != code || string(result.Wire.Body["vendor"]) != "9007199254740993" || !reflect.DeepEqual(events, []string{"member", "name", "create"}) {
				t.Fatal(result, err, events)
			}
		})
	}
}

func TestFloatingIPAllocateExplicitPortIgnoresOtherDestinations(t *testing.T) {
	cloud := testcloud.New(t)
	var resolvers atomic.Int32
	cloud.Mux.HandleFunc("GET /v2.0/networks/net", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"network":{"id":"net"}}`) })
	cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		if !reflect.DeepEqual(body, map[string]map[string]string{"floatingip": {"floating_network_id": "net", "port_id": "port"}}) {
			t.Error(body)
		}
		testcloud.JSON(w, 201, `{"id":"new"}`)
	})
	service := network.NewWithDependencies(cloud.Client("network", "/v2.0/"), network.Dependencies{Server: func(context.Context, resource.Ref) (string, error) {
		resolvers.Add(1)
		return "", errors.New("ignored server")
	}})
	name := "net"
	result, err := service.FloatingIPs.Allocate(context.Background(), network.AllocateFloatingIPRequest{Network: &name, PortID: "port", Server: resource.ID("unsafe/path"), FixedAddress: "invalid", NATDestination: "ignored"})
	if err != nil || result.Selection.PortID != "port" || result.Selection.FixedIPv4 != "" || result.Wire == nil || resolvers.Load() != 0 {
		t.Fatal(result, err)
	}
}

func TestFloatingIPAllocateOptionalServerAndLiteralFixedSelection(t *testing.T) {
	for _, mode := range []string{"no ports", "fixed absent", "literal IPv6", "latest NAT"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			name := "net"
			fixed := ""
			cloud.Mux.HandleFunc("GET /v2.0/networks/net", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"network":{"id":"net"}}`) })
			cloud.Mux.HandleFunc("GET /v2.0/networks/nat", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"network":{"id":"nat"}}`) })
			cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.RawQuery != "device_id=server" {
					t.Error(r.URL)
				}
				body := `{"ports":[]}`
				if mode != "no ports" {
					body = `{"ports":[{"id":"old","network_id":"nat","created_at":"2026-01-01","fixed_ips":[{"ip_address":"10.0.0.1"}]},{"id":"new","network_id":"nat","created_at":"2026-02-01","fixed_ips":[{"ip_address":{}},{"ip_address":"2001:db8::1"},{"ip_address":"10.0.0.2"}]}]}`
				}
				if mode == "latest NAT" {
					body = strings.Replace(body, `"fixed_ips":[{"ip_address":{}}`, `"fixed_ips":[true,{"ip_address":{}}`, 1)
				}
				testcloud.JSON(w, 200, body)
			})
			want := map[string]string{"floating_network_id": "net"}
			if mode == "fixed absent" {
				fixed = "not present"
			}
			if mode == "literal IPv6" {
				fixed = "2001:db8::1"
				want["port_id"] = "new"
				want["fixed_ip_address"] = fixed
			}
			if mode == "latest NAT" {
				want["port_id"] = "new"
				want["fixed_ip_address"] = "10.0.0.2"
			}
			cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				var body map[string]map[string]string
				_ = json.NewDecoder(r.Body).Decode(&body)
				if !reflect.DeepEqual(body["floatingip"], want) {
					t.Error(body, want)
				}
				testcloud.JSON(w, 202, `{"floatingip":{"id":"new"}}`)
			})
			result, err := network.New(cloud.Client("network", "/v2.0/")).FloatingIPs.Allocate(context.Background(), network.AllocateFloatingIPRequest{Network: &name, Server: resource.ID("server"), FixedAddress: fixed, NATDestination: "nat"})
			if err != nil || !result.Allocated || result.Selection.PortID != want["port_id"] || result.Selection.FixedIPv4 != want["fixed_ip_address"] {
				t.Fatal(result, err)
			}
		})
	}
}

func TestFloatingIPAllocateLatePortsAndNATFailurePreventMutation(t *testing.T) {
	for _, mode := range []string{"late denied", "NAT missing", "no IPv4"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			name := "net"
			var posts atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/networks/net", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"network":{"id":"net"}}`) })
			cloud.Mux.HandleFunc("GET /v2.0/networks/nat", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
			cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"networks":[]}`) })
			cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("marker") != "" {
					testcloud.JSON(w, 403, `{"error":"late"}`)
					return
				}
				body := `{"ports":[{"id":"p","network_id":"nat","fixed_ips":[{"ip_address":"::1"}]}]}`
				if mode == "NAT missing" {
					body = `{"ports":[{"id":"p","network_id":"nat"},{"id":"p2","network_id":"nat"}]}`
				}
				if mode == "late denied" {
					body = `{"ports":[{"id":"p","fixed_ips":[{"ip_address":"10.0.0.1"}]}],"ports_links":[{"rel":"next","href":"/v2.0/ports?marker=p"}]}`
				}
				testcloud.JSON(w, 200, body)
			})
			cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) { posts.Add(1); w.WriteHeader(500) })
			result, err := network.New(cloud.Client("network", "/v2.0/")).FloatingIPs.Allocate(context.Background(), network.AllocateFloatingIPRequest{Network: &name, Server: resource.ID("server"), NATDestination: "nat"})
			if err == nil || result != nil || posts.Load() != 0 {
				t.Fatal(result, err, posts.Load())
			}
			if mode == "NAT missing" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestFloatingIPAllocateDefaultRoleAndRouter(t *testing.T) {
	for _, router := range []bool{false, true} {
		t.Run(fmt.Sprint(router), func(t *testing.T) {
			cloud := testcloud.New(t)
			want := "external"
			if !router {
				availableRoles(t, cloud)
			} else {
				want = "gateway"
				cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"networks":[]}`) })
				cloud.Mux.HandleFunc("GET /v2.0/subnets", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"subnets":[]}`) })
				cloud.Mux.HandleFunc("GET /v2.0/routers", func(w http.ResponseWriter, r *http.Request) {
					testcloud.JSON(w, 200, `{"routers":[{"id":"disabled","admin_state_up":false,"external_gateway_info":{"network_id":"wrong"}},{"id":"r","admin_state_up":true,"external_gateway_info":{"network_id":"gateway"}}]}`)
				})
			}
			cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				var body map[string]map[string]string
				_ = json.NewDecoder(r.Body).Decode(&body)
				if body["floatingip"]["floating_network_id"] != want {
					t.Error(body)
				}
				testcloud.JSON(w, 201, `{"floatingip":{"id":"new"}}`)
			})
			result, err := network.New(cloud.Client("network", "/v2.0/")).FloatingIPs.Allocate(context.Background(), network.AllocateFloatingIPRequest{})
			if err != nil || !result.Allocated || result.Selection.NetworkID != want {
				t.Fatal(result, err)
			}
		})
	}
}

func TestFloatingIPAvailabilityPlanPreservesTypedNATReferences(t *testing.T) {
	const uuidName = "550e8400-e29b-41d4-a716-446655440000"
	for _, scenario := range []struct {
		name       string
		ref        resource.Ref
		wantLookup bool
		missing    bool
	}{
		{name: "UUID-shaped Name is exact name", ref: resource.Name(uuidName), wantLookup: true},
		{name: "non-UUID ID bypasses lookup", ref: resource.ID("chosen")},
		{name: "missing typed Name is terminal NAT error", ref: resource.Name("missing"), wantLookup: true, missing: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var events []string
			cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				if name := r.URL.Query().Get("name"); name != "" {
					events = append(events, "NAT name")
					if !scenario.wantLookup || r.URL.RawQuery != "name="+scenario.ref.String() {
						t.Error("NAT Ref lost name/ID identity", r.URL)
					}
					if scenario.missing {
						testcloud.JSON(w, 200, `{"networks":[]}`)
					} else {
						// Server-side filtering is not trusted: the SDK must keep
						// the literal UUID-shaped name and ignore the other row.
						testcloud.JSON(w, 200, fmt.Sprintf(`{"networks":[{"id":"wrong","name":"other"},{"id":"chosen","name":%q}]}`, uuidName))
					}
					return
				}
				events = append(events, "roles")
				if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				// Reuse the ordinary Available role fixture's network/subnet
				// topology while also serving the exact-name NAT query here.
				testcloud.JSON(w, 200, `{"networks":[{"id":"external","name":"public","router:external":true,"subnets":["ext-sub"]},{"id":"private","name":"private","subnets":["priv-sub"]}]}`)
			})
			cloud.Mux.HandleFunc("GET /v2.0/subnets", func(w http.ResponseWriter, r *http.Request) {
				events = append(events, "subnets")
				testcloud.JSON(w, 200, `{"subnets":[{"id":"ext-sub","network_id":"external","ip_version":4},{"id":"priv-sub","network_id":"private","ip_version":4}]}`)
			})
			cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
				events = append(events, "ports")
				if r.URL.RawQuery != "device_id=server" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"ports":[{"id":"other-port","network_id":"wrong","created_at":"2026-03-01","fixed_ips":[{"ip_address":"10.0.0.99"}]},{"id":"chosen-port","network_id":"chosen","created_at":"2026-01-01","fixed_ips":[{"ip_address":"10.0.0.2"}]}]}`)
			})
			cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				events = append(events, "POST")
				if scenario.missing {
					t.Error("missing typed NAT must not allocate")
				}
				var body map[string]map[string]string
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				want := map[string]map[string]string{"floatingip": {"floating_network_id": "external", "port_id": "chosen-port", "fixed_ip_address": "10.0.0.2"}}
				if !reflect.DeepEqual(body, want) {
					t.Error(body, want)
				}
				w.Header().Set("X-Allocation-Proof", "typed NAT")
				testcloud.JSON(w, 202, `{"floatingip":{"id":"fresh","port_id":"chosen-port"}}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected HTTP; availability plan must preserve typed NAT and allocate directly: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			ctx := context.Background()
			service := network.New(cloud.Client("network", "/v2.0/"))
			policy, err := network.PrepareAvailableFloatingIPOptions(ctx, network.WithAvailableNATDestination(scenario.ref))
			if err != nil {
				t.Fatal(err)
			}
			plan, err := service.FloatingIPs.PrepareAvailability(ctx, availableRequest(resource.ID("server")), policy)
			if err != nil || plan == nil || plan.NetworkID() != "external" || !reflect.DeepEqual(events, []string{"roles", "subnets"}) {
				t.Fatal(plan, err, events)
			}
			result, err := plan.Allocate(ctx)
			wantEvents := []string{"roles", "subnets", "ports"}
			if scenario.wantLookup {
				wantEvents = append(wantEvents, "NAT name")
			}
			if scenario.missing {
				if result != nil || !errors.Is(err, resource.ErrNotFound) || !errors.Is(err, resource.ErrInvalidOption) || !reflect.DeepEqual(events, wantEvents) {
					t.Fatal(result, err, events, wantEvents)
				}
				return
			}
			wantEvents = append(wantEvents, "POST")
			if err != nil || result == nil || !result.Allocated || result.Selection.NetworkID != "external" || result.Selection.PortID != "chosen-port" || result.Selection.PortNetworkID != "chosen" || result.Selection.FixedIPv4 != "10.0.0.2" || result.AllocationResponse == nil || result.AllocationResponse.StatusCode != 202 || result.AllocationResponse.Header.Get("X-Allocation-Proof") != "typed NAT" || result.Wire == nil || string(result.Wire.Body["port_id"]) != `"chosen-port"` || !reflect.DeepEqual(events, wantEvents) {
				t.Fatal(result, err, events, wantEvents)
			}
		})
	}
}
