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

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

func floatingIPBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var body struct {
		FloatingIP map[string]any `json:"floatingip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if r.Header.Get("X-Auth-Token") != "test-token" {
		t.Error(r.Header)
	}
	return body.FloatingIP
}

func respondFloatingIP(w http.ResponseWriter, code int, port, fixed, status string) {
	testcloud.JSON(w, code, fmt.Sprintf(`{"floatingip":{"id":"fip","floating_network_id":"external","floating_ip_address":"198.51.100.10","port_id":%q,"fixed_ip_address":%q,"status":%q}}`, port, fixed, status))
}

func TestFloatingIPCreateResolvesNamesAndSelectsAcrossPages(t *testing.T) {
	cloud := testcloud.New(t)
	var serverLookups, posts atomic.Int32
	service := network.NewWithDependencies(cloud.Client("network", "/v2.0"), network.Dependencies{
		Server: func(ctx context.Context, ref resource.Ref) (string, error) {
			serverLookups.Add(1)
			if !ref.IsName() || ref.String() != "web" {
				t.Error(ref)
			}
			return "server", nil
		},
	})
	cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("name") {
		case "public":
			if r.URL.Query().Get("router:external") != "true" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, fmt.Sprintf(`{"networks":[{"id":"private","name":"public","router:external":false},{"id":"near","name":"public-copy","router:external":true}],"networks_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2.0/networks?marker=second"))
		case "private":
			testcloud.JSON(w, 200, `{"networks":[{"id":"nat","name":"private"}]}`)
		default:
			if r.URL.Query().Get("marker") != "second" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, `{"networks":[{"id":"external","name":"public","router:external":true}]}`)
		}
	})
	cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("marker") == "second" {
			testcloud.JSON(w, 200, `{"ports":[{"id":"other","device_id":"other-server","network_id":"nat","fixed_ips":[{"ip_address":"10.0.0.99"}]},{"id":"port","device_id":"server","network_id":"nat","fixed_ips":[{"ip_address":"invalid"},{"ip_address":"10.0.0.10"}]}]}`)
			return
		}
		if r.URL.Query().Get("device_id") != "server" || r.URL.Query().Get("network_id") != "nat" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"ports":[{"id":"v6","device_id":"server","network_id":"nat","fixed_ips":[{"ip_address":"2001:db8::1"}]}],"ports_links":[{"rel":"next","href":%q}]}`, cloud.Server.URL+"/v2.0/ports?marker=second"))
	})
	cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		want := map[string]any{"floating_network_id": "external", "port_id": "port", "fixed_ip_address": "10.0.0.10", "description": "public web"}
		if got := floatingIPBody(t, r); !reflect.DeepEqual(got, want) {
			t.Errorf("body=%v want=%v", got, want)
		}
		respondFloatingIP(w, 201, "port", "10.0.0.10", "DOWN")
	})
	created, err := service.FloatingIPs.Create(context.Background(), network.CreateFloatingIPRequest{Network: resource.Name("public")},
		network.WithServer(resource.Name("web")), network.WithNATDestination(resource.Name("private")), network.WithDescription("public web"))
	if err != nil || created.ID != "fip" || created.PortID != "port" {
		t.Fatalf("created=%+v err=%v", created, err)
	}
	if serverLookups.Load() != 1 || posts.Load() != 1 {
		t.Fatalf("lookups=%d posts=%d", serverLookups.Load(), posts.Load())
	}
	if service.Ports != service.API.Ports.Resources || service.FloatingIPs.Collection != service.API.FloatingIPs.Resources {
		t.Fatal("collection clients differ")
	}
}

func TestFloatingIPCandidateAmbiguityNeverCreates(t *testing.T) {
	for _, check := range []struct {
		name, ports string
		want        []string
	}{
		{"two-ports", `[{"id":"b","device_id":"server","fixed_ips":[{"ip_address":"10.0.0.2"}]},{"id":"a","device_id":"server","fixed_ips":[{"ip_address":"10.0.0.1"}]}]`, []string{"a@10.0.0.1", "b@10.0.0.2"}},
		{"two-fixed-ips", `[{"id":"port","device_id":"server","fixed_ips":[{"ip_address":"10.0.0.2"},{"ip_address":"10.0.0.1"}]}]`, []string{"port@10.0.0.1", "port@10.0.0.2"}},
	} {
		t.Run(check.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"ports":`+check.ports+`}`) })
			cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) { t.Fatal("created an ambiguous destination") })
			service := network.New(cloud.Client("network", "/v2.0"))
			created, err := service.FloatingIPs.Create(context.Background(), network.CreateFloatingIPRequest{Network: resource.ID("external")}, network.WithServer(resource.ID("server")))
			var ambiguous *resource.AmbiguousError
			if created != nil || !errors.Is(err, resource.ErrAmbiguous) || !errors.As(err, &ambiguous) || !reflect.DeepEqual(ambiguous.IDs, check.want) {
				t.Fatalf("created=%+v err=%v detail=%+v", created, err, ambiguous)
			}
		})
	}
}

func TestFloatingIPFixedAddressAndExplicitPortSelection(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(fmt.Sprintf("explicit=%t", explicit), func(t *testing.T) {
			cloud := testcloud.New(t)
			port := `{"id":"port","device_id":"server","network_id":"nat","fixed_ips":[{"ip_address":"10.0.0.1"},{"ip_address":"10.0.0.2"},{"ip_address":"::ffff:10.0.0.2"}]}`
			if explicit {
				cloud.Mux.HandleFunc("GET /v2.0/ports/port", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"port":`+port+`}`) })
				cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) { t.Fatal("listed despite explicit port") })
			} else {
				cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"ports":[`+port+`]}`) })
			}
			cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				if got := floatingIPBody(t, r); got["port_id"] != "port" || got["fixed_ip_address"] != "10.0.0.2" {
					t.Error(got)
				}
				respondFloatingIP(w, 201, "port", "10.0.0.2", "DOWN")
			})
			service := network.NewWithDependencies(cloud.Client("network", "/v2.0"), network.Dependencies{Server: func(context.Context, resource.Ref) (string, error) {
				t.Fatal("resolved an explicit server ID")
				return "", nil
			}})
			options := []network.CreateFloatingIPOption{network.WithServer(resource.ID("server")), network.WithNATDestination(resource.ID("nat")), network.WithFixedAddress("10.0.0.2")}
			if explicit {
				options = append(options, network.WithPort(resource.ID("port")))
			}
			created, err := service.FloatingIPs.Create(context.Background(), network.CreateFloatingIPRequest{Network: resource.ID("external")}, options...)
			if err != nil || created.FixedIP != "10.0.0.2" {
				t.Fatalf("created=%+v err=%v", created, err)
			}
		})
	}
}

func TestFloatingIPAllocatesWithoutAssociationAndSnapshotsExtensions(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) { t.Fatal("unassociated allocation listed ports") })
	cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) { t.Fatal("explicit network ID was looked up") })
	cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		want := map[string]any{"floating_network_id": "external", "floating_ip_address": "198.51.100.10", "vendor": map[string]any{"enabled": false}}
		if got := floatingIPBody(t, r); !reflect.DeepEqual(got, want) {
			t.Errorf("got=%v want=%v", got, want)
		}
		respondFloatingIP(w, 201, "", "", "DOWN")
	})
	value := map[string]any{"enabled": false}
	option := network.WithFloatingIPField("vendor", value)
	value["enabled"] = true
	service := network.New(cloud.Client("network", "/v2.0"))
	for range 2 {
		created, err := service.FloatingIPs.Create(context.Background(), network.CreateFloatingIPRequest{Network: resource.ID("external")}, network.WithFloatingIPAddress("198.51.100.10"), option)
		if err != nil || created.ID != "fip" || created.PortID != "" {
			t.Fatalf("created=%+v err=%v", created, err)
		}
	}
}

func TestFloatingIPInvalidConfigurationHasNoHTTPRequests(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("invalid configuration made a request", r.URL)
		w.WriteHeader(500)
	})
	service := network.New(cloud.Client("network", "/v2.0"))
	for _, options := range [][]network.CreateFloatingIPOption{
		{nil}, {network.WithServer(resource.Ref{})}, {network.WithPort(resource.ID("../bad"))},
		{network.WithFixedAddress("10.0.0.1")}, {network.WithNATDestination(resource.ID("nat"))}, {network.WithWait()},
		{network.WithFixedAddress("2001:db8::1")}, {network.WithFixedAddress("invalid")},
		{network.WithFloatingIPAddress("2001:db8::1")},
		{network.WithServer(resource.ID("server")), network.WithWait(resource.WithTimeout(0))},
		{network.WithFloatingIPField("port_id", "override")}, {network.WithFloatingIPField("", true)},
		{network.WithFloatingIPField("invalid", make(chan bool))},
	} {
		created, err := service.FloatingIPs.Create(context.Background(), network.CreateFloatingIPRequest{Network: resource.Name("public")}, options...)
		if created != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("created=%+v err=%v", created, err)
		}
	}
	if _, err := service.FloatingIPs.Create(context.Background(), network.CreateFloatingIPRequest{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := service.FloatingIPs.Create(context.Background(), network.CreateFloatingIPRequest{Network: resource.ID("external")}, network.WithServer(resource.Name("web"))); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.FloatingIPs.Create(ctx, network.CreateFloatingIPRequest{Network: resource.ID("external")}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestFloatingIPExternalNameAndDestinationFailuresNeverCreate(t *testing.T) {
	for _, check := range []struct {
		name, networks, ports string
		networkRef            resource.Ref
		options               []network.CreateFloatingIPOption
		target                error
	}{
		{"external-duplicate", `[{"id":"a","name":"public","router:external":true},{"id":"b","name":"public","router:external":true}]`, `[]`, resource.Name("public"), nil, resource.ErrAmbiguous},
		{"internal-only", `[{"id":"a","name":"public","router:external":false}]`, `[]`, resource.Name("public"), nil, resource.ErrNotFound},
		{"v6-only", `[]`, `[{"id":"port","device_id":"server","fixed_ips":[{"ip_address":"2001:db8::1"}]}]`, resource.ID("external"), []network.CreateFloatingIPOption{network.WithServer(resource.ID("server"))}, resource.ErrNotFound},
		{"wrong-server", `[]`, `[{"id":"port","device_id":"other","fixed_ips":[{"ip_address":"10.0.0.1"}]}]`, resource.ID("external"), []network.CreateFloatingIPOption{network.WithServer(resource.ID("server"))}, resource.ErrNotFound},
		{"wrong-fixed", `[]`, `[{"id":"port","device_id":"server","fixed_ips":[{"ip_address":"10.0.0.1"}]}]`, resource.ID("external"), []network.CreateFloatingIPOption{network.WithServer(resource.ID("server")), network.WithFixedAddress("10.0.0.2")}, resource.ErrNotFound},
		{"wrong-network", `[]`, `[{"id":"port","device_id":"server","network_id":"other","fixed_ips":[{"ip_address":"10.0.0.1"}]}]`, resource.ID("external"), []network.CreateFloatingIPOption{network.WithServer(resource.ID("server")), network.WithNATDestination(resource.ID("nat"))}, resource.ErrNotFound},
	} {
		t.Run(check.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"networks":`+check.networks+`}`)
			})
			cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"ports":`+check.ports+`}`) })
			cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) { t.Fatal("created despite lookup failure") })
			service := network.New(cloud.Client("network", "/v2.0"))
			created, err := service.FloatingIPs.Create(context.Background(), network.CreateFloatingIPRequest{Network: check.networkRef}, check.options...)
			if created != nil || !errors.Is(err, check.target) {
				t.Fatalf("created=%+v err=%v", created, err)
			}
		})
	}
}

func TestFloatingIPWaitAndPostCreationFailurePreserveAllocation(t *testing.T) {
	for _, check := range []struct {
		name, returnedPort, pollStatus string
		wait                           bool
		pollCode                       int
		target                         error
	}{
		{"active", "port", "ACTIVE", true, 200, nil},
		{"failed", "port", "ERROR", true, 200, resource.ErrFailedState},
		{"timeout", "port", "DOWN", true, 200, context.DeadlineExceeded},
		{"poll-forbidden", "port", "", true, 403, nil},
		{"poll-not-found", "port", "", true, 404, resource.ErrNotFound},
		{"late-wrong-association", "port", "ACTIVE", true, 200, nil},
		{"wrong-association", "other", "", false, 0, nil},
	} {
		t.Run(check.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var polls, posts, deletes atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/ports/port", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"port":{"id":"port","fixed_ips":[{"ip_address":"10.0.0.1"}]}}`)
			})
			cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				respondFloatingIP(w, 201, check.returnedPort, "10.0.0.1", "DOWN")
			})
			cloud.Mux.HandleFunc("GET /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
				polls.Add(1)
				if check.pollCode != 200 {
					testcloud.JSON(w, check.pollCode, `{}`)
					return
				}
				port := "port"
				if check.name == "late-wrong-association" {
					port = "other"
				}
				respondFloatingIP(w, 200, port, "10.0.0.1", check.pollStatus)
			})
			cloud.Mux.HandleFunc("DELETE /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) { deletes.Add(1); w.WriteHeader(204) })
			options := []network.CreateFloatingIPOption{network.WithPort(resource.ID("port"))}
			if check.wait {
				timeout := 3 * time.Second
				if check.name == "timeout" {
					timeout = 35 * time.Millisecond
				}
				options = append(options, network.WithWait(resource.WithTimeout(timeout), resource.WithPollInterval(time.Millisecond)))
			}
			service := network.New(cloud.Client("network", "/v2.0"))
			created, err := service.FloatingIPs.Create(context.Background(), network.CreateFloatingIPRequest{Network: resource.ID("external")}, options...)
			if created == nil || created.ID != "fip" || posts.Load() != 1 || deletes.Load() != 0 {
				t.Fatalf("created=%+v err=%v posts=%d deletes=%d", created, err, posts.Load(), deletes.Load())
			}
			if check.name == "active" {
				if err != nil || created.Status != "ACTIVE" {
					t.Fatalf("created=%+v err=%v", created, err)
				}
			} else {
				if err == nil || created.Status != "DOWN" || (check.target != nil && !errors.Is(err, check.target)) {
					t.Fatalf("created=%+v err=%v", created, err)
				}
			}
			if check.pollCode == 403 {
				var responseErr gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &responseErr) || responseErr.Actual != 403 {
					t.Fatalf("lost HTTP error: %v", err)
				}
			}
			if check.wait && polls.Load() == 0 {
				t.Fatal("wait did not poll stable ID")
			}
		})
	}
}

func TestFloatingIPHTTPFailureDoesNotFallbackOrReuse(t *testing.T) {
	for _, code := range []int{403, 404, 409} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, code, `{}`) })
			cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) { t.Fatal("new allocation tried to reuse existing IP") })
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Error("unexpected fallback", r.URL)
				w.WriteHeader(500)
			})
			service := network.New(cloud.Client("network", "/v2.0"))
			created, err := service.FloatingIPs.Create(context.Background(), network.CreateFloatingIPRequest{Network: resource.ID("external")})
			var responseErr gophercloud.ErrUnexpectedResponseCode
			if created != nil || !errors.As(err, &responseErr) || responseErr.Actual != code {
				t.Fatalf("created=%+v err=%v", created, err)
			}
		})
	}
}

func TestFloatingIPRejectsResponsesForAnotherResource(t *testing.T) {
	for _, wrong := range []string{"port", "floating-ip"} {
		t.Run(wrong, func(t *testing.T) {
			cloud := testcloud.New(t)
			var posts atomic.Int32
			cloud.Mux.HandleFunc("GET /v2.0/ports/port", func(w http.ResponseWriter, r *http.Request) {
				id := "port"
				if wrong == "port" {
					id = "other"
				}
				testcloud.JSON(w, 200, fmt.Sprintf(`{"port":{"id":%q,"fixed_ips":[{"ip_address":"10.0.0.1"}]}}`, id))
			})
			cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				respondFloatingIP(w, 201, "port", "10.0.0.1", "DOWN")
			})
			cloud.Mux.HandleFunc("GET /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"floatingip":{"id":"other","port_id":"port","fixed_ip_address":"10.0.0.1","status":"ACTIVE"}}`)
			})
			service := network.New(cloud.Client("network", "/v2.0"))
			created, err := service.FloatingIPs.Create(context.Background(), network.CreateFloatingIPRequest{Network: resource.ID("external")}, network.WithPort(resource.ID("port")), network.WithWait())
			if err == nil {
				t.Fatal("accepted another resource")
			}
			if wrong == "port" && (created != nil || posts.Load() != 0) {
				t.Fatalf("created=%+v posts=%d", created, posts.Load())
			}
			if wrong == "floating-ip" && (created == nil || created.ID != "fip" || posts.Load() != 1) {
				t.Fatalf("created=%+v posts=%d", created, posts.Load())
			}
		})
	}
}

func TestFloatingIPCancelledWaitPreservesAllocation(t *testing.T) {
	cloud := testcloud.New(t)
	polling := make(chan struct{})
	cloud.Mux.HandleFunc("GET /v2.0/ports/port", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"port":{"id":"port","fixed_ips":[{"ip_address":"10.0.0.1"}]}}`)
	})
	cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) { respondFloatingIP(w, 201, "port", "10.0.0.1", "DOWN") })
	cloud.Mux.HandleFunc("GET /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) { close(polling); <-r.Context().Done() })
	service := network.New(cloud.Client("network", "/v2.0"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan struct {
		created *network.FloatingIP
		err     error
	}, 1)
	go func() {
		created, err := service.FloatingIPs.Create(ctx, network.CreateFloatingIPRequest{Network: resource.ID("external")}, network.WithPort(resource.ID("port")), network.WithWait())
		result <- struct {
			created *network.FloatingIP
			err     error
		}{created, err}
	}()
	<-polling
	cancel()
	got := <-result
	if got.created == nil || got.created.ID != "fip" || !errors.Is(got.err, context.Canceled) {
		t.Fatalf("created=%+v err=%v", got.created, got.err)
	}
}
