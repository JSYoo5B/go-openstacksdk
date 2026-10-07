package network_test

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

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func ensureFloatingRequest() network.EnsureFloatingIPRequest {
	return network.EnsureFloatingIPRequest{Server: resource.ID("server"), Network: resource.ID("external")}
}

func respondEnsuredFloatingIP(w http.ResponseWriter, code int, port, fixed, status string, overrides ...map[string]any) {
	fields := map[string]any{"id": "fip", "floating_network_id": "external", "floating_ip_address": "198.51.100.10", "project_id": "owner", "port_id": port, "fixed_ip_address": fixed, "status": status}
	for _, values := range overrides {
		for key, value := range values {
			fields[key] = value
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"floatingip": fields})
}

func ensurePortFixture(t *testing.T, cloud *testcloud.Cloud) {
	t.Helper()
	cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("device_id") != "server" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"ports":[{"id":"port","device_id":"server","network_id":"private","fixed_ips":[{"ip_address":"10.0.0.10"}]}]}`)
	})
}

func TestFloatingIPEnsureFiltersAllPagesAndPreservesRevisionPresence(t *testing.T) {
	for _, tc := range []struct{ name, field, header string }{
		{"absent", "", ""}, {"null", `,"revision_number":null`, ""},
		{"zero", `,"revision_number":0`, "revision_number=0"}, {"positive", `,"revision_number":7`, "revision_number=7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			ensurePortFixture(t, cloud)
			var lists, updates atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("marker") == "" {
					if r.URL.Query().Get("project_id") != "owner" || r.URL.Query().Get("floating_network_id") != "external" {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, fmt.Sprintf(`{"floatingips":[
{"id":"foreign","project_id":"other","floating_network_id":"external","floating_ip_address":"198.51.100.1"},
{"id":"wrong-network","project_id":"owner","floating_network_id":"other","floating_ip_address":"198.51.100.2"},
{"id":"taken","project_id":"owner","floating_network_id":"external","port_id":"other-port","floating_ip_address":"198.51.100.3"},
{"id":"v6","project_id":"owner","floating_network_id":"external","floating_ip_address":"2001:db8::1"},
{"id":"failed","project_id":"owner","floating_network_id":"external","status":"ERROR","floating_ip_address":"198.51.100.4"},
{"id":"conflicting-owner","project_id":"owner","tenant_id":"other","floating_network_id":"external","floating_ip_address":"198.51.100.5"}
],"floatingips_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2.0/floatingips?marker=second"))
					return
				}
				testcloud.JSON(w, 200, `{"floatingips":[{"id":"fip","tenant_id":"owner","floating_network_id":"external","floating_ip_address":"198.51.100.10","port_id":null`+tc.field+`},{"id":"later","project_id":"owner","floating_network_id":"external","floating_ip_address":"198.51.100.11"}]}`)
			})
			cloud.Mux.HandleFunc("PUT /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
				updates.Add(1)
				if lists.Load() != 2 || r.Header.Get("If-Match") != tc.header {
					t.Errorf("lists=%d headers=%v", lists.Load(), r.Header)
				}
				if got := floatingIPBody(t, r); !reflect.DeepEqual(got, map[string]any{"port_id": "port", "fixed_ip_address": "10.0.0.10"}) {
					t.Error(got)
				}
				respondEnsuredFloatingIP(w, 200, "port", "10.0.0.10", "ACTIVE")
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected allocation/lookup/cleanup: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			service := network.New(cloud.Client("network", "/v2.0"))
			result, err := service.FloatingIPs.Ensure(context.Background(), ensureFloatingRequest(), network.WithEnsureProject("owner"))
			if err != nil || result == nil || !result.Reused || result.Allocated || result.FloatingIP.ID != "fip" || result.FloatingIP.PortID != "port" || updates.Load() != 1 {
				t.Fatalf("result=%+v err=%v updates=%d", result, err, updates.Load())
			}
		})
	}
}

func TestFloatingIPEnsurePrefersAlreadyAttachedAndAllocatesOnlyWhenNeeded(t *testing.T) {
	for _, scenario := range []string{"already attached", "none free", "reuse disabled"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			ensurePortFixture(t, cloud)
			var lists, posts atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if scenario == "already attached" {
					testcloud.JSON(w, 200, `{"floatingips":[{"id":"free","project_id":"owner","floating_network_id":"external","floating_ip_address":"198.51.100.11"},{"id":"fip","project_id":"owner","floating_network_id":"external","floating_ip_address":"198.51.100.10","port_id":"port","fixed_ip_address":"10.0.0.10","status":"ACTIVE"}]}`)
				} else {
					testcloud.JSON(w, 200, `{"floatingips":[]}`)
				}
			})
			cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				if got := floatingIPBody(t, r); !reflect.DeepEqual(got, map[string]any{"floating_network_id": "external", "project_id": "owner", "port_id": "port", "fixed_ip_address": "10.0.0.10"}) {
					t.Error(got)
				}
				respondEnsuredFloatingIP(w, 201, "port", "10.0.0.10", "DOWN")
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected mutation/cleanup: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			opts := []network.EnsureFloatingIPOption{network.WithEnsureProject("owner")}
			if scenario == "reuse disabled" {
				opts = []network.EnsureFloatingIPOption{network.WithEnsureReuse(false), network.WithEnsureProject("owner")}
			}
			result, err := network.New(cloud.Client("network", "/v2.0")).FloatingIPs.Ensure(context.Background(), ensureFloatingRequest(), opts...)
			if err != nil || result == nil || result.FloatingIP.ID != "fip" {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if scenario == "already attached" {
				if !result.Reused || result.Allocated || posts.Load() != 0 {
					t.Fatalf("result=%+v posts=%d", result, posts.Load())
				}
			} else if result.Reused || !result.Allocated || posts.Load() != 1 {
				t.Fatalf("result=%+v posts=%d", result, posts.Load())
			}
			wantLists := int32(1)
			if scenario == "reuse disabled" {
				wantLists = 0
			}
			if lists.Load() != wantLists {
				t.Fatal(lists.Load())
			}
		})
	}
}

func TestFloatingIPEnsureErrorsPreserveSelectionAndNeverFallback(t *testing.T) {
	for _, scenario := range []string{"list403", "late page403", "update409", "revision412", "wait ERROR", "wait wrong destination", "allocation wrong destination", "update wrong ID", "update wrong owner", "update conflicting owner", "wait wrong owner", "wait wrong ID", "wait custom attribute"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			ensurePortFixture(t, cloud)
			var updates, posts atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				if scenario == "list403" || r.URL.Query().Get("marker") == "last" {
					testcloud.JSON(w, 403, `{"error":{"message":"denied"}}`)
					return
				}
				if scenario == "late page403" {
					testcloud.JSON(w, 200, fmt.Sprintf(`{"floatingips":[{"id":"fip","project_id":"owner","floating_network_id":"external","floating_ip_address":"198.51.100.10"}],"floatingips_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2.0/floatingips?marker=last"))
					return
				}
				if scenario == "allocation wrong destination" {
					testcloud.JSON(w, 200, `{"floatingips":[]}`)
					return
				}
				testcloud.JSON(w, 200, `{"floatingips":[{"id":"fip","project_id":"owner","floating_network_id":"external","floating_ip_address":"198.51.100.10","revision_number":3}]}`)
			})
			cloud.Mux.HandleFunc("PUT /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
				updates.Add(1)
				code := 200
				if scenario == "update409" {
					code = 409
				}
				if scenario == "revision412" {
					code = 412
				}
				fields := map[string]any{}
				if scenario == "update wrong ID" {
					fields["id"] = "other"
				}
				if scenario == "update wrong owner" {
					fields["project_id"] = "other"
				}
				if scenario == "update conflicting owner" {
					fields["tenant_id"] = "other"
				}
				respondEnsuredFloatingIP(w, code, "port", "10.0.0.10", "DOWN", fields)
			})
			cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				respondEnsuredFloatingIP(w, 201, "wrong", "10.0.0.11", "DOWN")
			})
			cloud.Mux.HandleFunc("GET /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
				if scenario == "wait ERROR" {
					respondEnsuredFloatingIP(w, 200, "port", "10.0.0.10", "ERROR")
				} else if scenario == "wait wrong owner" {
					respondEnsuredFloatingIP(w, 200, "port", "10.0.0.10", "ACTIVE", map[string]any{"project_id": "other"})
				} else if scenario == "wait wrong ID" {
					respondEnsuredFloatingIP(w, 200, "port", "10.0.0.10", "ACTIVE", map[string]any{"id": "other"})
				} else if scenario == "wait custom attribute" {
					respondEnsuredFloatingIP(w, 200, "port", "10.0.0.10", "ERROR", map[string]any{"description": "ACTIVE"})
				} else {
					respondEnsuredFloatingIP(w, 200, "wrong", "10.0.0.11", "ACTIVE")
				}
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected retry/cleanup: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			opts := []network.EnsureFloatingIPOption{network.WithEnsureProject("owner")}
			if len(scenario) >= 5 && scenario[:5] == "wait " {
				opts = append(opts, network.WithEnsureWait(resource.WithTimeout(time.Second)))
			}
			if scenario == "wait custom attribute" {
				opts = append(opts, network.WithEnsureWait(resource.WithTimeout(time.Second), resource.WithStatusAttribute("description")))
			}
			result, err := network.New(cloud.Client("network", "/v2.0")).FloatingIPs.Ensure(context.Background(), ensureFloatingRequest(), opts...)
			if err == nil {
				t.Fatal("expected failure")
			}
			if scenario == "list403" || scenario == "late page403" {
				if result != nil || updates.Load() != 0 || posts.Load() != 0 {
					t.Fatalf("result=%+v updates=%d posts=%d", result, updates.Load(), posts.Load())
				}
			} else if result == nil || result.FloatingIP == nil || result.FloatingIP.ID != "fip" {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if scenario == "update409" || scenario == "revision412" || scenario == "list403" || scenario == "late page403" {
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) {
					t.Fatal(err)
				}
				want := 403
				if scenario == "update409" {
					want = 409
				}
				if scenario == "revision412" {
					want = 412
				}
				if response.Actual != want {
					t.Fatal(err)
				}
			}
			if scenario == "wait ERROR" && !errors.Is(err, resource.ErrFailedState) {
				t.Fatal(err)
			}
			if scenario != "allocation wrong destination" && posts.Load() != 0 {
				t.Fatal("allocated after another failure")
			}
		})
	}
}

func TestFloatingIPEnsureValidationAndUnknownProjectPrecedeHTTP(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts []network.EnsureFloatingIPOption
		want error
	}{
		{"unknown scope", nil, resource.ErrUnsupported},
		{"nil option", []network.EnsureFloatingIPOption{nil}, resource.ErrInvalidOption},
		{"zero policy", []network.EnsureFloatingIPOption{network.WithEnsureFloatingIPPolicy(network.EnsureFloatingIPPolicy{})}, resource.ErrInvalidOption},
		{"invalid project", []network.EnsureFloatingIPOption{network.WithEnsureProject("bad/id")}, resource.ErrInvalidOption},
		{"IPv6 fixed", []network.EnsureFloatingIPOption{network.WithEnsureFixedAddress("2001:db8::1")}, resource.ErrInvalidOption},
		{"bad wait", []network.EnsureFloatingIPOption{network.WithEnsureWait(resource.WithTimeout(-time.Second))}, resource.ErrInvalidOption},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Error("preflight made HTTP request")
				http.Error(w, "unexpected", 500)
			})
			result, err := network.New(cloud.Client("network", "/v2.0")).FloatingIPs.Ensure(context.Background(), ensureFloatingRequest(), tc.opts...)
			if result != nil || !errors.Is(err, tc.want) {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}
