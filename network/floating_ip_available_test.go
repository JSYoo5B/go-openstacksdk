package network_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func availableRoles(t *testing.T, cloud *testcloud.Cloud) {
	t.Helper()
	cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"networks":[{"id":"external","name":"public","router:external":true,"subnets":["ext-sub"]},{"id":"private","name":"private","subnets":["priv-sub"]}]}`)
	})
	cloud.Mux.HandleFunc("GET /v2.0/subnets", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"subnets":[{"id":"ext-sub","network_id":"external","ip_version":4},{"id":"priv-sub","network_id":"private","ip_version":4}]}`)
	})
}

func availableRequest(server resource.Ref) network.AvailableFloatingIPRequest {
	return network.AvailableFloatingIPRequest{Networks: []resource.Ref{resource.Name("missing"), resource.Name("public")}, Server: server}
}

func TestFloatingIPAvailableFreeFirstNeverUsesServer(t *testing.T) {
	cloud := testcloud.New(t)
	availableRoles(t, cloud)
	var resolvers, writes atomic.Int32
	service := network.NewWithDependencies(cloud.Client("network", "/v2.0/"), network.Dependencies{Server: func(context.Context, resource.Ref) (string, error) {
		resolvers.Add(1)
		return "", errors.New("must remain lazy")
	}})
	cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Query()) != 0 {
			t.Error("availability pushed down filters", r.URL)
		}
		testcloud.JSON(w, 200, `{"floatingips":[
		{"id":"attached","floating_network_id":"external","project_id":"owner","port_id":"port","floating_ip_address":"198.51.100.1"},
		{"id":"empty-port","floating_network_id":"external","project_id":"owner","port_id":"","floating_ip_address":"198.51.100.2"},
		{"id":"foreign","floating_network_id":"external","project_id":"foreign","port_id":null},
		{"id":"free","floating_network_id":"external","project_id":"owner","port_id":null,"floating_ip_address":"2001:db8::1","status":"ERROR","vendor_number":9007199254740993},
		{"id":"later","floating_network_id":"external","project_id":"owner","floating_ip_address":"198.51.100.4"}]}`)
	})
	cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		writes.Add(1)
		t.Error("free candidate allocated")
		w.WriteHeader(500)
	})
	result, err := service.FloatingIPs.Available(context.Background(), availableRequest(resource.Name("unused-server")), network.WithAvailableProject("owner"))
	if err != nil || result == nil || !result.Reused || result.Allocated || result.AllocationResponse != nil || result.FloatingIP.ID != "free" || result.FloatingIP.Status != "ERROR" || result.Metadata.StatusCode != 200 || string(result.Metadata.Body["vendor_number"]) != "9007199254740993" || string(result.Metadata.Body["port_id"]) != "null" || resolvers.Load() != 0 || writes.Load() != 0 {
		t.Fatal(result, err, resolvers.Load(), writes.Load())
	}
}

func TestFloatingIPAvailableAllocationUsesAuthenticatedProjectAndOptionalPort(t *testing.T) {
	for _, scenario := range []string{"no server", "no ports", "recent NAT port", "fixed absent"} {
		for _, code := range []int{201, 202} {
			t.Run(fmt.Sprint(scenario, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				availableRoles(t, cloud)
				policy, err := network.PrepareNetworkRoleOptions(network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "private", NATDestination: true}))
				if err != nil {
					t.Fatal(err)
				}
				service := network.NewWithDependencies(cloud.Client("network", "/v2.0/"), network.Dependencies{NetworkRoles: policy})
				var ports, posts atomic.Int32
				cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"floatingips":[]}`) })
				cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
					ports.Add(1)
					if r.URL.Query().Get("device_id") != "server" {
						t.Error(r.URL)
					}
					body := `{"ports":[]}`
					if scenario == "recent NAT port" || scenario == "fixed absent" {
						body = `{"ports":[{"id":"old","device_id":"server","network_id":"private","created_at":"2026-01-01T00:00:00Z","fixed_ips":[{"ip_address":"10.0.0.10"}]},{"id":"new","device_id":"server","network_id":"private","created_at":"2026-02-01T00:00:00Z","fixed_ips":[{"ip_address":"2001:db8::1"},{"ip_address":"10.0.0.20"}]}]}`
					}
					testcloud.JSON(w, 200, body)
				})
				cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
					posts.Add(1)
					var body map[string]map[string]any
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Fatal(err)
					}
					fip := body["floatingip"]
					want := 1
					if scenario == "recent NAT port" {
						want = 3
						if fip["port_id"] != "new" || fip["fixed_ip_address"] != "10.0.0.20" {
							t.Error(body)
						}
					}
					if len(body) != 1 || len(fip) != want || fip["floating_network_id"] != "external" {
						t.Error(body)
					}
					port, fixed := "null", "null"
					if scenario == "recent NAT port" {
						port, fixed = `"new"`, `"10.0.0.20"`
					}
					w.Header().Set("X-Allocation-Proof", "new")
					testcloud.JSON(w, code, fmt.Sprintf(`{"floatingip":{"id":"allocated","floating_network_id":"external","floating_ip_address":"198.51.100.10","project_id":"actual-auth-project","port_id":%s,"fixed_ip_address":%s,"status":"DOWN"}}`, port, fixed))
				})
				server := resource.ID("server")
				if scenario == "no server" {
					server = resource.Ref{}
				}
				opts := []network.AvailableFloatingIPOption{network.WithAvailableProject("filter-only-project")}
				if scenario == "fixed absent" {
					opts = append(opts, network.WithAvailableFixedAddress("10.0.0.99"))
				}
				result, err := service.FloatingIPs.Available(context.Background(), availableRequest(server), opts...)
				wantPorts := int32(1)
				if scenario == "no server" {
					wantPorts = 0
				}
				if err != nil || result == nil || result.Reused || !result.Allocated || result.FloatingIP.ProjectID != "actual-auth-project" || result.FloatingIP.Status != "DOWN" || result.AllocationResponse.StatusCode != code || result.AllocationResponse.Header.Get("X-Allocation-Proof") != "new" || posts.Load() != 1 || ports.Load() != wantPorts {
					t.Fatal(result, err, ports.Load(), posts.Load())
				}
			})
		}
	}
}

func TestFloatingIPAvailableNullableCurrentProjectAndPort(t *testing.T) {
	for _, field := range []string{`"project_id":null`, `"tenant_id":null`, `"project_id":null,"tenant_id":"foreign"`, `"project_id":null,"port_id":null`} {
		t.Run(field, func(t *testing.T) {
			cloud := testcloud.New(t)
			availableRoles(t, cloud)
			cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"floatingips":[{"id":"wrong","floating_network_id":"external","project_id":""},{"id":"free","floating_network_id":"external",`+field+`}]}`)
			})
			result, err := network.New(cloud.Client("network", "/v2.0/")).FloatingIPs.Available(context.Background(), availableRequest(resource.Ref{}))
			if err != nil || result == nil || !result.Reused || result.FloatingIP.ID != "free" {
				t.Fatal(result, err)
			}
		})
	}
}

func TestFloatingIPAvailableConsumesEveryPageBeforeReturningOrAllocating(t *testing.T) {
	for _, last := range []string{"ok", "denied", "malformed", "204"} {
		t.Run(last, func(t *testing.T) {
			cloud := testcloud.New(t)
			availableRoles(t, cloud)
			var lists, posts atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("marker") == "" {
					testcloud.JSON(w, 200, fmt.Sprintf(`{"floatingips":[{"id":"first","floating_network_id":"external","project_id":"owner","port_id":null}],"floatingips_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2.0/floatingips?marker=next"))
					return
				}
				switch last {
				case "ok":
					testcloud.JSON(w, 200, `{"floatingips":[]}`)
				case "denied":
					w.Header().Set("X-Last", "denied")
					testcloud.JSON(w, 403, `{"error":"later"}`)
				case "malformed":
					testcloud.JSON(w, 200, `{"floatingips":`)
				case "204":
					w.WriteHeader(204)
				}
			})
			cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				t.Error("unexpected allocation")
				w.WriteHeader(500)
			})
			result, err := network.New(cloud.Client("network", "/v2.0/")).FloatingIPs.Available(context.Background(), availableRequest(resource.Ref{}), network.WithAvailableProject("owner"))
			if (last == "ok") != (err == nil) || (last == "ok" && (result == nil || result.FloatingIP.ID != "first")) || (last != "ok" && result != nil) || lists.Load() != 2 || posts.Load() != 0 {
				t.Fatal(result, err, lists.Load(), posts.Load())
			}
			if last == "denied" && !strings.Contains(err.Error(), "403") {
				t.Fatal(err)
			}
		})
	}
}
