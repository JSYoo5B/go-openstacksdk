package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestConnectionNetworkMutationRefreshesGetterDefaultNICAndFloatingRoles(t *testing.T) {
	for _, operation := range []string{"create", "update", "delete"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			var inventories, mutations, servers, allocations atomic.Int32
			cloud.Mux.HandleFunc("GET /network/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				generation := inventories.Add(1)
				testcloud.JSON(w, 200, fmt.Sprintf(`{"networks":[{"id":"private-%d","name":"private"},{"id":"external-%d","name":"public","router:external":false}]}`, generation, generation))
			})
			cloud.Mux.HandleFunc("GET /network/v2.0/networks/mutation", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"network":{"id":"mutation"}}`) })
			for _, method := range []string{"POST", "PUT", "DELETE"} {
				path := "/network/v2.0/networks/mutation"
				if method == "POST" {
					path = "/network/v2.0/networks"
				}
				cloud.Mux.HandleFunc(method+" "+path, func(w http.ResponseWriter, r *http.Request) {
					mutations.Add(1)
					code := 201
					if r.Method == "PUT" {
						code = 200
					}
					if r.Method == "DELETE" {
						w.WriteHeader(204)
						return
					}
					testcloud.JSON(w, code, `{"network":{"id":"mutation"}}`)
				})
			}
			cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
				servers.Add(1)
				var body struct{ Server map[string]any }
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if !reflect.DeepEqual(body.Server["networks"], []any{map[string]any{"uuid": "private-2"}}) {
					t.Error(body.Server)
				}
				testcloud.JSON(w, 202, `{"server":{"id":"server"}}`)
			})
			cloud.Mux.HandleFunc("GET /network/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("device_id") != "server" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"ports":[{"id":"selected","device_id":"server","network_id":"private-2","fixed_ips":[{"ip_address":"10.0.0.10"}]},{"id":"other","device_id":"server","network_id":"other","fixed_ips":[{"ip_address":"2001:db8::1"}]}]}`)
			})
			cloud.Mux.HandleFunc("POST /network/v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				allocations.Add(1)
				var body struct{ FloatingIP map[string]any }
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body.FloatingIP["floating_network_id"] != "external-2" || body.FloatingIP["port_id"] != "selected" || body.FloatingIP["fixed_ip_address"] != "10.0.0.10" {
					t.Error(body)
				}
				testcloud.JSON(w, 201, `{"floatingip":{"id":"ip","floating_network_id":"external-2","port_id":"selected","fixed_ip_address":"10.0.0.10","floating_ip_address":"203.0.113.4","project_id":"owner","status":"DOWN"}}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected request %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute"), sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network"),
				sdk.WithNetworkRoles(network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "private", DefaultInterface: true, NATDestination: true}, network.ConfiguredNetwork{Name: "public", NATSource: true, RoutesIPv4Externally: true})))
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			old, err := conn.GetNetworkRoles(ctx)
			if err != nil || old.DefaultNetwork.ID != "private-1" || inventories.Load() != 1 {
				t.Fatal(old, err, inventories.Load())
			}
			s, err := conn.Network(ctx)
			if err != nil {
				t.Fatal(err)
			}
			same, err := conn.Network(ctx)
			if err != nil || same != s {
				t.Fatal(same, err)
			}
			switch operation {
			case "create":
				_, err = s.CreateNetwork(ctx, network.CreateNetworkRequest{Name: "mutation"})
			case "update":
				_, err = s.UpdateNetwork(ctx, resource.ID("mutation"), network.WithNetworkDescription("updated"))
			case "delete":
				var deleted bool
				deleted, err = s.DeleteNetwork(ctx, resource.ID("mutation"))
				if !deleted {
					t.Fatal(deleted, err)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			fresh, err := conn.GetDefaultNetwork(ctx)
			if err != nil || fresh.ID != "private-2" {
				t.Fatal(fresh, err)
			}
			compute, err := conn.Compute(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := compute.Servers.Create(ctx, nicConnectionRequest()); err != nil {
				t.Fatal(err)
			}
			result, err := s.FloatingIPs.Ensure(ctx, network.EnsureFloatingIPRequest{Server: resource.ID("server")}, network.WithEnsureReuse(false), network.WithEnsureProject("owner"))
			if err != nil || result == nil || result.FloatingIP == nil || result.FloatingIP.FloatingNetworkID != "external-2" || inventories.Load() != 2 || mutations.Load() != 1 || servers.Load() != 1 || allocations.Load() != 1 {
				t.Fatal(result, err, inventories.Load(), mutations.Load(), servers.Load(), allocations.Load())
			}
		})
	}
}
