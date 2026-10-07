package network_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

func TestFloatingIPEnsureAutomaticNetworkConsumesEmptyAndLaterPages(t *testing.T) {
	for _, scenario := range []string{"external", "empty network page", "external late error", "router", "empty router page", "router late error", "no gateway", "invalid gateway", "empty port page", "empty IP page", "cancel network"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var networkPages, routerPages, portPages, ipPages, updates atomic.Int32
			useRouter := scenario == "router" || scenario == "empty router page" || scenario == "router late error" || scenario == "no gateway" || scenario == "invalid gateway"
			cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				networkPages.Add(1)
				if useRouter {
					testcloud.JSON(w, 200, `{"networks":[{"id":"internal","router:external":false}]}`)
					return
				}
				if r.URL.Query().Get("marker") == "" {
					if r.URL.Query().Get("router:external") != "" {
						t.Error(r.URL)
					}
					rows := `[{"id":"internal","router:external":false}]`
					if scenario == "empty network page" {
						rows = `[]`
					}
					if scenario == "external late error" {
						rows = `[{"id":"external","router:external":true}]`
					}
					testcloud.JSON(w, 200, fmt.Sprintf(`{"networks":%s,"networks_links":[{"rel":"next","href":%q}]}`, rows, cloud.Server.URL+"/v2.0/networks?marker=last"))
					if scenario == "cancel network" {
						cancel()
					}
					return
				}
				if scenario == "external late error" {
					testcloud.JSON(w, 403, `{"error":{"message":"denied"}}`)
					return
				}
				testcloud.JSON(w, 200, `{"networks":[{"id":"external","router:external":true},{"id":"later","router:external":true}]}`)
			})
			cloud.Mux.HandleFunc("GET /v2.0/subnets", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"subnets":[]}`)
			})
			cloud.Mux.HandleFunc("GET /v2.0/routers", func(w http.ResponseWriter, r *http.Request) {
				routerPages.Add(1)
				if !useRouter {
					t.Error("router fallback despite external network")
				}
				if r.URL.Query().Get("marker") == "" {
					rows := `[{"id":"disabled","admin_state_up":false,"external_gateway_info":{"network_id":"other"}},{"id":"no-gateway","admin_state_up":true}]`
					if scenario == "empty router page" {
						rows = `[]`
					}
					if scenario == "router late error" {
						rows = `[{"id":"gateway","admin_state_up":true,"external_gateway_info":{"network_id":"external"}}]`
					}
					testcloud.JSON(w, 200, fmt.Sprintf(`{"routers":%s,"routers_links":[{"rel":"next","href":%q}]}`, rows, cloud.Server.URL+"/v2.0/routers?marker=last"))
					return
				}
				if scenario == "router late error" {
					testcloud.JSON(w, 403, `{"error":{"message":"denied"}}`)
					return
				}
				if scenario == "no gateway" {
					testcloud.JSON(w, 200, `{"routers":[]}`)
					return
				}
				id := "external"
				if scenario == "invalid gateway" {
					id = "bad/id"
				}
				testcloud.JSON(w, 200, fmt.Sprintf(`{"routers":[{"id":"gateway","admin_state_up":true,"external_gateway_info":{"network_id":%q}},{"id":"later","admin_state_up":true,"external_gateway_info":{"network_id":"later"}}]}`, id))
			})
			cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
				portPages.Add(1)
				if useRouter && routerPages.Load() != 2 || !useRouter && networkPages.Load() != 2 {
					t.Error("destination lookup before selection consumed all pages")
				}
				if scenario == "empty port page" && r.URL.Query().Get("marker") == "" {
					testcloud.JSON(w, 200, fmt.Sprintf(`{"ports":[],"ports_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2.0/ports?marker=last"))
					return
				}
				testcloud.JSON(w, 200, `{"ports":[{"id":"port","device_id":"server","network_id":"private","fixed_ips":[{"ip_address":"10.0.0.10"}]}]}`)
			})
			cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				ipPages.Add(1)
				if scenario == "empty IP page" && r.URL.Query().Get("marker") == "" {
					testcloud.JSON(w, 200, fmt.Sprintf(`{"floatingips":[],"floatingips_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2.0/floatingips?marker=last"))
					return
				}
				testcloud.JSON(w, 200, `{"floatingips":[{"id":"fip","project_id":"owner","floating_network_id":"external","floating_ip_address":"198.51.100.10"}]}`)
			})
			cloud.Mux.HandleFunc("PUT /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
				updates.Add(1)
				respondEnsuredFloatingIP(w, 200, "port", "10.0.0.10", "ACTIVE")
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected allocation/cleanup: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			result, err := network.New(cloud.Client("network", "/v2.0")).FloatingIPs.Ensure(ctx, network.EnsureFloatingIPRequest{Server: resource.ID("server")}, network.WithEnsureProject("owner"))
			wantFailure := scenario == "external late error" || scenario == "router late error" || scenario == "no gateway" || scenario == "invalid gateway" || scenario == "cancel network"
			if !wantFailure {
				if err != nil || result == nil || result.FloatingIP.ID != "fip" || updates.Load() != 1 {
					t.Fatalf("result=%+v err=%v updates=%d", result, err, updates.Load())
				}
				if scenario == "empty port page" && portPages.Load() != 2 || scenario == "empty IP page" && ipPages.Load() != 2 {
					t.Fatalf("port pages=%d IP pages=%d", portPages.Load(), ipPages.Load())
				}
				return
			}
			if err == nil || result != nil || portPages.Load() != 0 || ipPages.Load() != 0 || updates.Load() != 0 {
				t.Fatalf("result=%+v err=%v ports=%d IPs=%d updates=%d", result, err, portPages.Load(), ipPages.Load(), updates.Load())
			}
			if scenario == "no gateway" && !errors.Is(err, resource.ErrNotFound) || scenario == "invalid gateway" && !errors.Is(err, resource.ErrInvalidOption) || scenario == "cancel network" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if scenario == "external late error" || scenario == "router late error" {
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) || response.Actual != 403 {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestFloatingIPEnsureWaitCancellationAndTimeoutPreserveAssignment(t *testing.T) {
	for _, scenario := range []string{"cancel callback", "timeout"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			ensurePortFixture(t, cloud)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"floatingips":[]}`)
			})
			cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				respondEnsuredFloatingIP(w, 201, "port", "10.0.0.10", "DOWN")
			})
			var gets atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				respondEnsuredFloatingIP(w, 200, "port", "10.0.0.10", "DOWN")
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Error("unexpected retry/allocation/cleanup", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			wait := []resource.WaitOption{resource.WithTimeout(20 * time.Millisecond), resource.WithPollInterval(time.Second)}
			want := context.DeadlineExceeded
			if scenario == "cancel callback" {
				wait = append(wait, resource.WithProgressCallback(func(int) { cancel() }))
				want = context.Canceled
			}
			result, err := network.New(cloud.Client("network", "/v2.0")).FloatingIPs.Ensure(ctx, ensureFloatingRequest(), network.WithEnsureProject("owner"), network.WithEnsureWait(wait...))
			if !errors.Is(err, want) || result == nil || !result.Allocated || result.FloatingIP.ID != "fip" || result.FloatingIP.PortID != "port" || gets.Load() > 1 {
				t.Fatalf("result=%+v err=%v GETs=%d", result, err, gets.Load())
			}
		})
	}
}

func TestFloatingIPEnsureRejectsEmptyPageCyclesBeforeRefetchOrMutation(t *testing.T) {
	for _, endpoint := range []string{"networks", "routers", "ports", "floatingips"} {
		t.Run(endpoint, func(t *testing.T) {
			cloud := testcloud.New(t)
			var cycles atomic.Int32
			for _, path := range []string{"networks", "routers", "ports", "floatingips"} {
				cloud.Mux.HandleFunc("GET /v2.0/"+path, func(w http.ResponseWriter, r *http.Request) {
					if path == endpoint {
						cycles.Add(1)
						testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[],%q:[{"rel":"next","href":%q}]}`, path, path+"_links", cloud.Server.URL+r.URL.RequestURI()))
						return
					}
					switch path {
					case "networks":
						testcloud.JSON(w, 200, `{"networks":[]}`)
					case "ports":
						testcloud.JSON(w, 200, `{"ports":[{"id":"port","device_id":"server","fixed_ips":[{"ip_address":"10.0.0.10"}]}]}`)
					default:
						t.Error("unexpected earlier/later lookup", path)
						http.Error(w, "unexpected", 500)
					}
				})
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Error("unexpected mutation/cleanup", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			request := ensureFloatingRequest()
			if endpoint == "networks" || endpoint == "routers" {
				request.Network = resource.Ref{}
			}
			result, err := network.New(cloud.Client("network", "/v2.0")).FloatingIPs.Ensure(context.Background(), request, network.WithEnsureProject("owner"))
			var cycle *resource.PaginationCycleError
			if !errors.As(err, &cycle) || result != nil || cycles.Load() != 1 {
				t.Fatalf("result=%+v err=%v cycle GETs=%d", result, err, cycles.Load())
			}
		})
	}
}
