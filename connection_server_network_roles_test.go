package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionCreateWithFloatingIPSharesDefaultSourceAndNATRoles(t *testing.T) {
	for _, scenario := range []string{"success", "role failure before POST", "port failure after ACTIVE"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, creates, serverGets, ports, posts, ipGets atomic.Int32
			cloud.Mux.HandleFunc("GET /network/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if creates.Load() != 0 || r.URL.RawQuery != "" {
					t.Error("shared role discovery did not precede Nova POST", r.URL)
				}
				if scenario == "role failure before POST" {
					testcloud.JSON(w, 403, `{"error":{"message":"networks denied"}}`)
					return
				}
				// A configured source can intentionally lack router:external.
				testcloud.JSON(w, 200, `{"networks":[{"id":"public-id","name":"public"},{"id":"private-id","name":"private"},{"id":"other-id","name":"other"}]}`)
			})
			cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
				creates.Add(1)
				var body struct{ Server map[string]any }
				_ = json.NewDecoder(r.Body).Decode(&body)
				if lists.Load() != 1 || !reflect.DeepEqual(body.Server["networks"], []any{map[string]any{"uuid": "private-id"}}) {
					t.Error(body.Server, lists.Load())
				}
				testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
			})
			cloud.Mux.HandleFunc("GET /compute/servers/created", func(w http.ResponseWriter, r *http.Request) {
				serverGets.Add(1)
				testcloud.JSON(w, 200, `{"server":{"id":"created","status":"ACTIVE","addresses":{"private":[]}}}`)
			})
			cloud.Mux.HandleFunc("GET /network/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
				ports.Add(1)
				if serverGets.Load() != 1 || r.URL.Query().Get("marker") == "" && r.URL.Query().Get("device_id") != "created" {
					t.Error("port selection before ACTIVE or wrong server", r.URL)
				}
				if scenario == "port failure after ACTIVE" {
					if r.URL.Query().Get("marker") == "last" {
						testcloud.JSON(w, 403, `{"error":{"message":"ports denied"}}`)
					} else {
						testcloud.JSON(w, 200, fmt.Sprintf(`{"ports":[{"id":"port","device_id":"created","network_id":"private-id","fixed_ips":[{"ip_address":"10.0.0.10"}]}],"ports_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/network/v2.0/ports?marker=last"))
					}
					return
				}
				testcloud.JSON(w, 200, `{"ports":[{"id":"other-port","device_id":"created","network_id":"other-id","fixed_ips":[{"ip_address":"10.0.1.10"}]},{"id":"port","device_id":"created","network_id":"private-id","fixed_ips":[{"ip_address":"10.0.0.10"}]}]}`)
			})
			respond := func(w http.ResponseWriter, code int, status string) {
				testcloud.JSON(w, code, fmt.Sprintf(`{"floatingip":{"id":"fip","project_id":"owner","floating_network_id":"public-id","floating_ip_address":"198.51.100.10","port_id":"port","fixed_ip_address":"10.0.0.10","status":%q}}`, status))
			}
			cloud.Mux.HandleFunc("POST /network/v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				var body struct {
					FloatingIP map[string]any `json:"floatingip"`
				}
				_ = json.NewDecoder(r.Body).Decode(&body)
				want := map[string]any{"floating_network_id": "public-id", "port_id": "port", "fixed_ip_address": "10.0.0.10", "project_id": "owner"}
				if lists.Load() != 1 || serverGets.Load() != 1 || ports.Load() != 1 || !reflect.DeepEqual(body.FloatingIP, want) {
					t.Error(body.FloatingIP, lists.Load(), serverGets.Load(), ports.Load())
				}
				respond(w, 201, "DOWN")
			})
			cloud.Mux.HandleFunc("GET /network/v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
				ipGets.Add(1)
				respond(w, 200, "ACTIVE")
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected subnet/router/reuse/cleanup %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute"), sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network"),
				sdk.WithNetworkRoles(network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "public", NATSource: true, RoutesIPv4Externally: true}, network.ConfiguredNetwork{Name: "private", NATDestination: true, DefaultInterface: true})))
			if err != nil {
				t.Fatal(err)
			}
			service, err := conn.Compute(context.Background())
			if err != nil || lists.Load() != 0 {
				t.Fatal(err, lists.Load())
			}
			request := serverFloatingIPRequest()
			request.FloatingIPNetwork = resource.Ref{}
			result, err := service.Servers.CreateWithFloatingIP(context.Background(), request,
				compute.WithWorkflowTimeout(time.Second), compute.WithFloatingIPOptions(network.WithEnsureProject("owner"), network.WithEnsureReuse(false)))
			if lists.Load() != 1 {
				t.Fatal(lists.Load())
			}
			if scenario == "success" {
				if err != nil || result == nil || result.Server.Status != "ACTIVE" || result.Assignment == nil || !result.Assignment.Allocated || result.Assignment.FloatingIP.Status != "ACTIVE" || posts.Load() != 1 || ipGets.Load() != 1 {
					t.Fatal(result, err, posts.Load(), ipGets.Load())
				}
				return
			}
			var response gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &response) || response.Actual != 403 || posts.Load() != 0 || ipGets.Load() != 0 {
				t.Fatal(result, err, posts.Load(), ipGets.Load())
			}
			if scenario == "role failure before POST" {
				if result != nil || creates.Load() != 0 || serverGets.Load() != 0 || ports.Load() != 0 {
					t.Fatal(result, creates.Load(), serverGets.Load(), ports.Load())
				}
			} else if result == nil || result.Server.ID != "created" || result.Server.Status != "ACTIVE" || result.Assignment != nil || ports.Load() != 2 {
				t.Fatal(result, ports.Load())
			}
		})
	}
}
