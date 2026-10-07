package network_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestFloatingIPPlanRejectsInitialPublicDependencyReplacement(t *testing.T) {
	for _, field := range []string{"API", "ports", "networks", "roles", "IPs", "API ports", "API IPs"} {
		t.Run(field, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected HTTP %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			client := cloud.Client("network", "/v2.0")
			service, other := network.New(client), network.New(client)
			ips := service.FloatingIPs
			switch field {
			case "API":
				service.API = nil
			case "ports":
				service.Ports = other.Ports
			case "networks":
				service.Networks = other.Networks
			case "roles":
				service.Roles = other.Roles
			case "IPs":
				service.FloatingIPs = other.FloatingIPs
			case "API ports":
				service.API.Ports = other.API.Ports
			case "API IPs":
				service.API.FloatingIPs = other.API.FloatingIPs
			}
			plan, err := ips.PrepareEnsure(context.Background(), ensureFloatingRequest())
			if !errors.Is(err, resource.ErrInvalidOption) || plan.Selection() != (network.FloatingIPSelection{}) {
				t.Fatal(err, plan.Selection())
			}
		})
	}
}

func TestFloatingIPPlanUsesOneRoleSnapshotAcrossReset(t *testing.T) {
	cloud := testcloud.New(t)
	var reads atomic.Int32
	cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		n := reads.Add(1)
		testcloud.JSON(w, 200, fmt.Sprintf(`{"networks":[{"id":"ext-%d","name":"public","router:external":true},{"id":"private-%d","name":"private"}]}`, n, n))
	})
	policy := preparedRoles(t, network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "private", NATDestination: true}))
	service := network.NewWithDependencies(cloud.Client("network", "/v2.0"), network.Dependencies{NetworkRoles: policy})
	cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		service.Roles.Reset()
		testcloud.JSON(w, 200, `{"ports":[{"id":"a","device_id":"server","network_id":"private-1","fixed_ips":[{"ip_address":"10.0.0.1"}]},{"id":"b","device_id":"server","network_id":"private-2","fixed_ips":[{"ip_address":"10.0.0.2"}]}]}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	for i := int32(1); i <= 2; i++ {
		plan, err := service.FloatingIPs.PrepareEnsure(context.Background(), network.EnsureFloatingIPRequest{Server: resource.ID("server")})
		wantPort := map[int32]string{1: "a", 2: "b"}[i]
		if err != nil || plan.Selection().NetworkID != fmt.Sprintf("ext-%d", i) || plan.Selection().PortID != wantPort || reads.Load() != i {
			t.Fatalf("plan=%+v err=%v reads=%d", plan.Selection(), err, reads.Load())
		}
	}
}

func TestFloatingIPPlanColdRoleReadGuardsAcceptedCloseBeforeNextHTTP(t *testing.T) {
	for _, mutation := range []string{"endpoint", "public API"} {
		t.Run(mutation, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("network", "/v2.0")
			service := network.New(client)
			var attempts atomic.Int32
			client.ProviderClient.HTTPClient.Transport = planTransport(func(r *http.Request) (*http.Response, error) {
				attempts.Add(1)
				if r.URL.Path != "/v2.0/networks" {
					t.Error("unexpected follow-up", r.URL)
				}
				body := planCloseBody{Reader: strings.NewReader(`{"networks":[{"id":"external","router:external":true}]}`), close: func() error {
					if mutation == "endpoint" {
						client.Endpoint = cloud.Server.URL + "/replacement/"
					} else {
						service.API = nil
					}
					return nil
				}}
				return &http.Response{StatusCode: 200, Header: http.Header{"X-Request-Id": {"role-close"}}, Body: body, Request: r}, nil
			})
			_, err := service.FloatingIPs.PrepareEnsure(context.Background(), network.EnsureFloatingIPRequest{Server: resource.ID("server")})
			var accepted *resource.ResponseError
			if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &accepted) || accepted.StatusCode != 200 || accepted.Header.Get("X-Request-Id") != "role-close" || attempts.Load() != 1 {
				t.Fatal(err, accepted, attempts.Load())
			}
		})
	}
}

func TestFloatingIPPlanPOSTCodesAndInvariantFailuresKeepAcceptedResources(t *testing.T) {
	for _, code := range []int{201, 202} {
		for _, scenario := range []string{"valid", "wrong owner", "malformed"} {
			t.Run(fmt.Sprintf("%d/%s", code, scenario), func(t *testing.T) {
				cloud := testcloud.New(t)
				ensurePortFixture(t, cloud)
				plannedPortRead(t, cloud)
				var posts atomic.Int32
				cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
					posts.Add(1)
					w.Header().Set("X-Request-Id", "allocation")
					if scenario == "malformed" {
						testcloud.JSON(w, code, `{"floatingip":`)
						return
					}
					extra := map[string]any{}
					if scenario == "wrong owner" {
						extra["project_id"] = "other"
					}
					respondEnsuredFloatingIP(w, code, "port", "10.0.0.10", "DOWN", extra)
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					t.Errorf("unexpected fallback/cleanup %s %s", r.Method, r.URL)
					http.Error(w, "unexpected", 500)
				})
				ips := network.New(cloud.Client("network", "/v2.0")).FloatingIPs
				plan, err := ips.PrepareEnsure(context.Background(), ensureFloatingRequest(), network.WithEnsureReuse(false), network.WithEnsureProject("owner"))
				if err != nil {
					t.Fatal(err)
				}
				result, err := ips.EnsurePrepared(context.Background(), plan)
				if result == nil || !result.Allocated || result.Reused || posts.Load() != 1 {
					t.Fatalf("result=%+v err=%v", result, err)
				}
				if scenario == "valid" {
					if err != nil || result.FloatingIP.ID != "fip" {
						t.Fatal(err, result)
					}
					return
				}
				var accepted *resource.ResponseError
				if err == nil || !errors.As(err, &accepted) || accepted.StatusCode != code || accepted.Header.Get("X-Request-Id") != "allocation" {
					t.Fatal(err, accepted)
				}
				if scenario == "wrong owner" && (result.FloatingIP == nil || result.FloatingIP.ID != "fip") {
					t.Fatal(result)
				}
				if scenario == "malformed" && result.FloatingIP != nil {
					t.Fatal(result)
				}
			})
		}
	}
}

func TestFloatingIPPlanWaitAndPreflightPreserveCancellationCause(t *testing.T) {
	cloud := testcloud.New(t)
	ensurePortFixture(t, cloud)
	plannedPortRead(t, cloud)
	cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		respondEnsuredFloatingIP(w, 201, "port", "10.0.0.10", "DOWN")
	})
	var gets atomic.Int32
	cloud.Mux.HandleFunc("GET /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		respondEnsuredFloatingIP(w, 200, "port", "10.0.0.10", "DOWN")
	})
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("operator canceled observation")
	ips := network.New(cloud.Client("network", "/v2.0")).FloatingIPs
	plan, err := ips.PrepareEnsure(ctx, ensureFloatingRequest(), network.WithEnsureReuse(false), network.WithEnsureWait(resource.WithProgressCallback(func(int) { cancel(cause) }), resource.WithPollInterval(time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	result, err := ips.EnsurePrepared(ctx, plan)
	if result == nil || !result.Allocated || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || gets.Load() != 1 {
		t.Fatal(result, err, gets.Load())
	}
	if _, err := ips.PrepareEnsure(ctx, ensureFloatingRequest()); !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal(err)
	}
}

func TestFloatingIPPlanRevisionHeadersRejectRetryChangesAndSourceOverride(t *testing.T) {
	for _, scenario := range []string{"delete", "replace", "source override"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			ensurePortFixture(t, cloud)
			plannedPortRead(t, cloud)
			cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"floatingips":[{"id":"free","project_id":"owner","floating_network_id":"external","floating_ip_address":"198.51.100.11","revision_number":0}]}`)
			})
			var updates atomic.Int32
			cloud.Mux.HandleFunc("PUT /v2.0/floatingips/free", func(w http.ResponseWriter, r *http.Request) {
				updates.Add(1)
				if r.Header.Get("If-Match") != "revision_number=0" {
					t.Error(r.Header)
				}
				http.Error(w, "original PUT503", 503)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected allocation/cleanup %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			client := cloud.Client("network", "/v2.0")
			if scenario == "source override" {
				client.MoreHeaders = map[string]string{"If-Match": "revision_number=7"}
			} else {
				client.ProviderClient.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
					if scenario == "delete" {
						delete(opts.MoreHeaders, "If-Match")
					} else {
						opts.MoreHeaders["If-Match"] = "revision_number=5"
					}
					return nil
				}
			}
			ips := network.New(client).FloatingIPs
			plan, err := ips.PrepareEnsure(context.Background(), ensureFloatingRequest(), network.WithEnsureProject("owner"))
			if err != nil {
				t.Fatal(err)
			}
			result, err := ips.EnsurePrepared(context.Background(), plan)
			wantUpdates := int32(1)
			if scenario == "source override" {
				wantUpdates = 0
			}
			if result == nil || !result.Reused || result.Allocated || result.FloatingIP.ID != "free" || result.FloatingIP.PortID != "" || !errors.Is(err, resource.ErrInvalidOption) || updates.Load() != wantUpdates {
				t.Fatal(result, err, updates.Load())
			}
			if scenario != "source override" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 503 || !strings.Contains(string(native.Body), "original PUT503") {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestFloatingIPPlanPUTFailurePreservesOriginalCandidateAndHTTPProof(t *testing.T) {
	for _, scenario := range []string{"wrong ID", "wrong fixed", "revision412"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			ensurePortFixture(t, cloud)
			plannedPortRead(t, cloud)
			cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"floatingips":[{"id":"free","project_id":"owner","floating_network_id":"external","floating_ip_address":"198.51.100.11","revision_number":0}]}`)
			})
			var updates atomic.Int32
			cloud.Mux.HandleFunc("PUT /v2.0/floatingips/free", func(w http.ResponseWriter, r *http.Request) {
				updates.Add(1)
				w.Header().Set("X-Request-Id", "association")
				if scenario == "revision412" {
					http.Error(w, "revision changed", 412)
					return
				}
				extra := map[string]any{"id": "free"}
				if scenario == "wrong ID" {
					extra["id"] = "different"
				} else {
					extra["fixed_ip_address"] = "10.0.0.99"
				}
				respondEnsuredFloatingIP(w, 200, "port", "10.0.0.10", "ACTIVE", extra)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected allocation/cleanup %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			ips := network.New(cloud.Client("network", "/v2.0")).FloatingIPs
			plan, err := ips.PrepareEnsure(context.Background(), ensureFloatingRequest(), network.WithEnsureProject("owner"))
			if err != nil {
				t.Fatal(err)
			}
			result, err := ips.EnsurePrepared(context.Background(), plan)
			if err == nil || result == nil || !result.Reused || result.Allocated || result.FloatingIP.ID != "free" || result.FloatingIP.PortID != "" || updates.Load() != 1 {
				t.Fatal(result, err, updates.Load())
			}
			if scenario == "revision412" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 412 {
					t.Fatal(err)
				}
			} else {
				var accepted *resource.ResponseError
				if !errors.As(err, &accepted) || accepted.StatusCode != 200 || accepted.Header.Get("X-Request-Id") != "association" || len(accepted.Body) == 0 {
					t.Fatal(err, accepted)
				}
			}
		})
	}
}
