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

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/network"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

func mutationService(t *testing.T, cloud *testcloud.Cloud) *network.Service {
	t.Helper()
	policy := preparedRoles(t, network.WithConfiguredNetworks(network.ConfiguredNetwork{
		Name: "private", NATDestination: true, DefaultInterface: true,
	}))
	return network.NewWithDependencies(cloud.Client("network", "/v2.0"), network.Dependencies{NetworkRoles: policy})
}

func mutationInventory(cloud *testcloud.Cloud, calls *atomic.Int32) {
	cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		call := calls.Add(1)
		testcloud.JSON(w, 200, fmt.Sprintf(`{"networks":[{"id":"cache-%d","name":"private"}]}`, call))
	})
}

func requireMutationCache(t *testing.T, s *network.Service, calls *atomic.Int32, expected int32) {
	t.Helper()
	rows, err := s.Roles.Discover(context.Background())
	if err != nil || rows == nil || rows.DefaultNetwork == nil || rows.DefaultNetwork.ID != fmt.Sprintf("cache-%d", expected) || calls.Load() != expected {
		t.Fatalf("roles=%+v error=%v inventory calls=%d; expected generation %d", rows, err, calls.Load(), expected)
	}
}

func networkMutationBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var envelope map[string]map[string]any
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		t.Error(err)
	}
	if len(envelope) != 1 || envelope["network"] == nil || r.Header.Get("X-Auth-Token") != "test-token" {
		t.Errorf("request envelope=%v header=%v", envelope, r.Header)
	}
	return envelope["network"]
}

func TestCreateNetworkCloudDefaultsAndOptionPresence(t *testing.T) {
	for _, code := range []int{201, 202} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			var inventory, posts atomic.Int32
			mutationInventory(cloud, &inventory)
			cloud.Mux.HandleFunc("POST /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				call := posts.Add(1)
				want := map[string]any{"name": "", "admin_state_up": true}
				if call == 2 {
					want = map[string]any{"name": "chosen", "admin_state_up": false, "port_security_enabled": false,
						"description": "", "project_id": "", "provider:network_type": "vlan",
						"provider:physical_network": nil, "provider:segmentation_id": "vlan1", "extension:enabled": false}
				}
				if body := networkMutationBody(t, r); !reflect.DeepEqual(body, want) {
					t.Errorf("body=%v want=%v", body, want)
				}
				testcloud.JSON(w, code, `{"network":{"id":"created","name":"chosen","admin_state_up":false,"tags":["owned"],"created_at":"2026-10-07T01:02:03Z"}}`)
			})
			s := mutationService(t, cloud)
			requireMutationCache(t, s, &inventory, 1)
			row, err := s.CreateNetwork(context.Background(), network.CreateNetworkRequest{})
			if err != nil || row == nil || row.ID != "created" || row.CreatedAt.IsZero() || !reflect.DeepEqual(row.Tags, []string{"owned"}) {
				t.Fatal(row, err)
			}
			requireMutationCache(t, s, &inventory, 2)
			row, err = s.CreateNetwork(context.Background(), network.CreateNetworkRequest{Name: "first"},
				network.WithNetworkName("chosen"), network.WithNetworkAdminStateUp(false), network.WithNetworkShared(false),
				network.WithNetworkExternal(false), network.WithNetworkPortSecurity(false), network.WithNetworkMTU(0),
				network.WithNetworkDNSDomain(""), network.WithNetworkDescription(""), network.WithNetworkProjectID(""),
				network.WithNetworkProvider(network.ProviderNetwork{NetworkType: request.Present("vlan"),
					PhysicalNetwork: request.Null[string](), SegmentationID: request.Present[any]("vlan1")}),
				network.WithNetworkField("extension:enabled", false))
			if err != nil || row == nil || posts.Load() != 2 {
				t.Fatal(row, err, posts.Load())
			}
			requireMutationCache(t, s, &inventory, 3)
		})
	}
}

func TestUpdateNetworkPreservesFalseEmptyNullAndRevision(t *testing.T) {
	for _, code := range []int{200, 201} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			var inventory, puts atomic.Int32
			mutationInventory(cloud, &inventory)
			cloud.Mux.HandleFunc("PUT /v2.0/networks/chosen", func(w http.ResponseWriter, r *http.Request) {
				puts.Add(1)
				want := map[string]any{"name": "", "admin_state_up": false, "shared": false, "router:external": false,
					"port_security_enabled": false, "dns_domain": nil, "mtu": float64(68), "provider:segmentation_id": float64(0)}
				if body := networkMutationBody(t, r); !reflect.DeepEqual(body, want) || r.Header.Get("If-Match") != "revision_number=0" {
					t.Error(body, r.Header)
				}
				testcloud.JSON(w, code, `{"network":{"id":"chosen","name":"","admin_state_up":false}}`)
			})
			cloud.Mux.HandleFunc("GET /v2.0/networks/chosen", func(w http.ResponseWriter, r *http.Request) {
				t.Error("explicit ID with fields must not GET")
				http.Error(w, "unexpected", 500)
			})
			s := mutationService(t, cloud)
			requireMutationCache(t, s, &inventory, 1)
			row, err := s.UpdateNetwork(context.Background(), resource.ID("chosen"), network.WithNetworkName(""),
				network.WithNetworkAdminStateUp(false), network.WithNetworkShared(false), network.WithNetworkExternal(false),
				network.WithNetworkPortSecurity(false), network.WithNetworkDNSDomainValue(request.Null[string]()),
				network.WithNetworkMTU(68), network.WithNetworkRevision(0),
				network.WithNetworkProvider(network.ProviderNetwork{SegmentationID: request.Present[any](0)}))
			if err != nil || row == nil || row.ID != "chosen" || puts.Load() != 1 {
				t.Fatal(row, err, puts.Load())
			}
			requireMutationCache(t, s, &inventory, 2)
		})
	}
}

func TestUpdateNetworkExactNamePagesAndNoChangeLookup(t *testing.T) {
	for _, scenario := range []string{"unique", "empty options", "missing", "ambiguous", "late403", "unsafe ID"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			var pages, puts atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				pages.Add(1)
				if r.URL.Query().Get("name") != "chosen" {
					t.Error(r.URL)
				}
				if r.URL.Query().Get("marker") == "" {
					testcloud.JSON(w, 200, fmt.Sprintf(`{"networks":[{"id":"prefix","name":"chosen-extra"}],"networks_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2.0/networks?name=chosen&marker=last"))
					return
				}
				switch scenario {
				case "missing":
					testcloud.JSON(w, 200, `{"networks":[]}`)
				case "ambiguous":
					testcloud.JSON(w, 200, `{"networks":[{"id":"first","name":"chosen"},{"id":"second","name":"chosen"}]}`)
				case "late403":
					testcloud.JSON(w, 403, `{"error":"denied"}`)
				case "unsafe ID":
					testcloud.JSON(w, 200, `{"networks":[{"id":"bad/id","name":"chosen"}]}`)
				default:
					testcloud.JSON(w, 200, `{"networks":[{"id":"selected","name":"chosen"}]}`)
				}
			})
			cloud.Mux.HandleFunc("PUT /v2.0/networks/selected", func(w http.ResponseWriter, r *http.Request) {
				puts.Add(1)
				testcloud.JSON(w, 200, `{"network":{"id":"selected","name":"chosen"}}`)
			})
			s := mutationService(t, cloud)
			opts := []network.NetworkOption{network.WithNetworkName("chosen")}
			if scenario == "empty options" {
				opts = nil
			}
			row, err := s.UpdateNetwork(context.Background(), resource.Name("chosen"), opts...)
			switch scenario {
			case "unique", "empty options":
				if err != nil || row == nil || row.ID != "selected" {
					t.Fatal(row, err)
				}
			case "missing":
				if !errors.Is(err, resource.ErrNotFound) {
					t.Fatal(err)
				}
			case "ambiguous":
				if !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(err)
				}
			case "unsafe ID":
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			default:
				if !gophercloud.ResponseCodeIs(err, 403) {
					t.Fatal(err)
				}
			}
			wantPuts := int32(0)
			if scenario == "unique" {
				wantPuts = 1
			}
			if pages.Load() != 2 || puts.Load() != wantPuts {
				t.Fatal(pages.Load(), puts.Load())
			}
		})
	}
}

func TestCreateNetworkAvailabilityZoneExtensionAndOwnedHints(t *testing.T) {
	for _, scenario := range []string{"empty", "owned", "missing", "403", "malformed"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			var probes, posts atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/extensions", func(w http.ResponseWriter, r *http.Request) {
				probes.Add(1)
				switch scenario {
				case "missing":
					testcloud.JSON(w, 200, `{"extensions":[{"alias":"availability_zone"}]}`)
				case "403":
					testcloud.JSON(w, 403, `{}`)
				case "malformed":
					testcloud.JSON(w, 200, `{"extensions":false}`)
				default:
					testcloud.JSON(w, 200, `{"extensions":[{"alias":"network_availability_zone"}]}`)
				}
			})
			cloud.Mux.HandleFunc("POST /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				want := []any{}
				if scenario == "owned" {
					want = []any{"zone-a"}
				}
				if body := networkMutationBody(t, r); !reflect.DeepEqual(body["availability_zone_hints"], want) {
					t.Error(body)
				}
				testcloud.JSON(w, 201, `{"network":{"id":"created"}}`)
			})
			hints := []string{}
			if scenario == "owned" {
				hints = []string{"zone-a"}
			}
			option := network.WithNetworkAvailabilityZoneHints(hints...)
			if len(hints) > 0 {
				hints[0] = "caller-changed"
			}
			s := mutationService(t, cloud)
			for range 2 {
				row, err := s.CreateNetwork(context.Background(), network.CreateNetworkRequest{Name: "chosen"}, option)
				switch scenario {
				case "empty", "owned":
					if err != nil || row == nil {
						t.Fatal(row, err)
					}
				case "missing":
					if !errors.Is(err, resource.ErrUnsupported) {
						t.Fatal(err)
					}
				case "403":
					if !gophercloud.ResponseCodeIs(err, 403) {
						t.Fatal(err)
					}
				default:
					if err == nil {
						t.Fatal("malformed extension response accepted")
					}
				}
			}
			wantPosts := int32(0)
			if scenario == "empty" || scenario == "owned" {
				wantPosts = 2
			}
			if probes.Load() != 2 || posts.Load() != wantPosts {
				t.Fatal(probes.Load(), posts.Load())
			}
		})
	}
}

func TestNetworkMutationPreflightStopsEveryLookupAndMutation(t *testing.T) {
	tests := []struct {
		name   string
		create bool
		opts   []network.NetworkOption
	}{
		{"nil", true, []network.NetworkOption{nil}},
		{"negative MTU", true, []network.NetworkOption{network.WithNetworkMTU(-1)}},
		{"small MTU cannot be hidden", true, []network.NetworkOption{network.WithNetworkMTU(67), network.WithNetworkMTU(1500)}},
		{"update zero cannot be hidden", false, []network.NetworkOption{network.WithNetworkMTU(0), network.WithNetworkMTU(1500)}},
		{"create revision", true, []network.NetworkOption{network.WithNetworkRevision(0)}},
		{"negative revision", false, []network.NetworkOption{network.WithNetworkRevision(-1)}},
		{"update project", false, []network.NetworkOption{network.WithNetworkProjectID("owner")}},
		{"update AZ", false, []network.NetworkOption{network.WithNetworkAvailabilityZoneHints()}},
		{"blank extension", true, []network.NetworkOption{network.WithNetworkField(" ", 1)}},
		{"reserved extension", true, []network.NetworkOption{network.WithNetworkField("router:external", false)}},
		{"marshal cannot be hidden", true, []network.NetworkOption{network.WithNetworkField("extension:value", make(chan int)), network.WithNetworkField("extension:value", 1)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("preflight sent %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			s := mutationService(t, cloud)
			var err error
			if tt.create {
				_, err = s.CreateNetwork(context.Background(), network.CreateNetworkRequest{}, tt.opts...)
			} else {
				_, err = s.UpdateNetwork(context.Background(), resource.Name("chosen"), tt.opts...)
			}
			if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestNetworkMutationSuccessFailureAndMissingCacheBoundaries(t *testing.T) {
	for _, operation := range []string{"create", "update", "delete", "collection delete", "empty update"} {
		for _, code := range []int{0, 403, 409, 412, 404} {
			t.Run(fmt.Sprintf("%s/%d", operation, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var inventory, mutations atomic.Int32
				mutationInventory(cloud, &inventory)
				cloud.Mux.HandleFunc("GET /v2.0/networks/chosen", func(w http.ResponseWriter, r *http.Request) {
					if operation == "empty update" && code != 0 {
						testcloud.JSON(w, code, `{}`)
						return
					}
					testcloud.JSON(w, 200, `{"network":{"id":"chosen","name":"chosen"}}`)
				})
				cloud.Mux.HandleFunc("POST /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
					mutations.Add(1)
					status := code
					if status == 0 {
						status = 201
					}
					testcloud.JSON(w, status, `{"network":{"id":"chosen"}}`)
				})
				cloud.Mux.HandleFunc("PUT /v2.0/networks/chosen", func(w http.ResponseWriter, r *http.Request) {
					mutations.Add(1)
					status := code
					if status == 0 {
						status = 200
					}
					testcloud.JSON(w, status, `{"network":{"id":"chosen"}}`)
				})
				cloud.Mux.HandleFunc("DELETE /v2.0/networks/chosen", func(w http.ResponseWriter, r *http.Request) {
					mutations.Add(1)
					status := code
					if status == 0 {
						status = 204
					}
					w.WriteHeader(status)
				})
				s := mutationService(t, cloud)
				requireMutationCache(t, s, &inventory, 1)
				var err error
				deleted := false
				switch operation {
				case "create":
					_, err = s.CreateNetwork(context.Background(), network.CreateNetworkRequest{Name: "chosen"})
				case "update":
					_, err = s.UpdateNetwork(context.Background(), resource.ID("chosen"), network.WithNetworkName("chosen"))
				case "delete":
					deleted, err = s.DeleteNetwork(context.Background(), resource.ID("chosen"))
				case "collection delete":
					err = s.Networks.Delete(context.Background(), resource.ID("chosen"))
				case "empty update":
					_, err = s.UpdateNetwork(context.Background(), resource.ID("chosen"))
				}
				reset := code == 0 || operation == "delete" && code == 404
				ignored := operation == "collection delete" && code == 404
				if reset || ignored {
					if err != nil {
						t.Fatal(err)
					}
				} else if !gophercloud.ResponseCodeIs(err, code) {
					t.Fatal(err)
				}
				if operation == "update" && code == 404 && !errors.Is(err, resource.ErrNotFound) {
					t.Fatal(err)
				}
				if operation == "delete" && deleted != reset {
					t.Fatal(deleted, err)
				}
				want := int32(1)
				if reset {
					want = 2
				}
				requireMutationCache(t, s, &inventory, want)
				wantMutations := int32(1)
				if operation == "empty update" {
					wantMutations = 0
				}
				if mutations.Load() != wantMutations {
					t.Fatal(mutations.Load())
				}
			})
		}
	}
}

func TestDeleteNetworkInitialMissingDoesNotMutateOrReset(t *testing.T) {
	for _, ref := range []resource.Ref{resource.ID("missing"), resource.Name("missing")} {
		t.Run(ref.String(), func(t *testing.T) {
			cloud := testcloud.New(t)
			var inventory atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("name") == "missing" {
					testcloud.JSON(w, 200, `{"networks":[]}`)
					return
				}
				call := inventory.Add(1)
				testcloud.JSON(w, 200, fmt.Sprintf(`{"networks":[{"id":"cache-%d","name":"private"}]}`, call))
			})
			cloud.Mux.HandleFunc("GET /v2.0/networks/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
			cloud.Mux.HandleFunc("DELETE /v2.0/networks/missing", func(w http.ResponseWriter, r *http.Request) {
				t.Error("delete after initial missing")
				w.WriteHeader(204)
			})
			s := mutationService(t, cloud)
			requireMutationCache(t, s, &inventory, 1)
			deleted, err := s.DeleteNetwork(context.Background(), ref)
			if err != nil || deleted {
				t.Fatal(deleted, err)
			}
			requireMutationCache(t, s, &inventory, 1)
		})
	}
}
