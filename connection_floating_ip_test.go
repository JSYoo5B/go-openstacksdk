package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestConnectionFloatingIPResolvesComputeAndNeutronReferences(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		server, external          resource.Ref
		serverBody                string
		wantServers, wantNetworks int32
		wantErr                   error
	}{
		{name: "names resolve in their own service", server: resource.Name("web[1].*"), external: resource.Name("public"), serverBody: `{"servers":[{"id":"other","name":"web[1].*copy"},{"id":"server-id","name":"web[1].*"}]}`, wantServers: 1, wantNetworks: 1},
		{name: "IDs skip service lookup", server: resource.ID("server-id"), external: resource.ID("external-id")},
		{name: "ambiguous server prevents allocation", server: resource.Name("web[1].*"), external: resource.Name("public"), serverBody: `{"servers":[{"id":"one","name":"web[1].*"},{"id":"two","name":"web[1].*"}]}`, wantServers: 1, wantErr: resource.ErrAmbiguous},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var servers, networks, ports, creates atomic.Int32
			cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/detail", func(w http.ResponseWriter, r *http.Request) {
				servers.Add(1)
				if r.URL.Query().Get("name") != `^web\[1\]\.\*$` {
					t.Errorf("Nova name=%q", r.URL.Query().Get("name"))
				}
				testcloud.JSON(w, 200, tc.serverBody)
			})
			cloud.Mux.HandleFunc("GET /network/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				networks.Add(1)
				if r.URL.Query().Get("name") != "public" || r.URL.Query().Get("router:external") != "true" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"networks":[{"id":"external-id","name":"public","router:external":true}]}`)
			})
			cloud.Mux.HandleFunc("GET /network/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
				ports.Add(1)
				if r.URL.Query().Get("device_id") != "server-id" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"ports":[{"id":"port-id","device_id":"server-id","network_id":"private-id","fixed_ips":[{"subnet_id":"subnet","ip_address":"10.0.0.3"}]}]}`)
			})
			cloud.Mux.HandleFunc("POST /network/v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				creates.Add(1)
				var body struct {
					FloatingIP map[string]any `json:"floatingip"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				for key, want := range map[string]string{"floating_network_id": "external-id", "port_id": "port-id", "fixed_ip_address": "10.0.0.3"} {
					if body.FloatingIP[key] != want {
						t.Errorf("%s=%v want=%s", key, body.FloatingIP[key], want)
					}
				}
				testcloud.JSON(w, 201, `{"floatingip":{"id":"floating-id","floating_network_id":"external-id","port_id":"port-id","fixed_ip_address":"10.0.0.3","floating_ip_address":"198.51.100.3","status":"DOWN"}}`)
			})
			service, err := connection(t, cloud).Network(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			floating, err := service.FloatingIPs.Create(context.Background(), network.CreateFloatingIPRequest{Network: tc.external}, network.WithServer(tc.server))
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("floating=%v err=%v", floating, err)
			}
			if servers.Load() != tc.wantServers || networks.Load() != tc.wantNetworks {
				t.Fatalf("server lookups=%d network lookups=%d", servers.Load(), networks.Load())
			}
			if tc.wantErr != nil {
				if floating != nil || ports.Load() != 0 || creates.Load() != 0 {
					t.Fatalf("floating=%v ports=%d creates=%d", floating, ports.Load(), creates.Load())
				}
			} else if floating == nil || floating.ID != "floating-id" || ports.Load() != 1 || creates.Load() != 1 {
				t.Fatalf("floating=%v ports=%d creates=%d", floating, ports.Load(), creates.Load())
			}
		})
	}
}
