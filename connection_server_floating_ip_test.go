package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

func serverFloatingIPRequest() compute.CreateServerWithFloatingIPRequest {
	return compute.CreateServerWithFloatingIPRequest{Server: compute.CreateServerRequest{Name: "web", Image: resource.ID("image"), Flavor: resource.ID("flavor")}, FloatingIPNetwork: resource.ID("external")}
}

type serverIPCounts struct{ creates, serverGets, ports, lists, puts, allocates, ipGets atomic.Int32 }

func setupServerIPFixture(t *testing.T, cloud *testcloud.Cloud, scenario string, inspectCreate ...func(*http.Request)) *serverIPCounts {
	t.Helper()
	auth := tokens.CreateResult{}
	auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": "owner"}}}
	auth.Header = http.Header{"X-Subject-Token": []string{"test-token"}}
	if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
		t.Fatal(err)
	}
	counts := &serverIPCounts{}
	cloud.Mux.HandleFunc("POST /compute/v2.1/project/servers", func(w http.ResponseWriter, r *http.Request) {
		counts.creates.Add(1)
		for _, inspect := range inspectCreate {
			inspect(r)
		}
		id := "created"
		if scenario == "create unsafe ID" {
			id = "bad/id"
		}
		if scenario == "create null" {
			testcloud.JSON(w, 202, `{"server":null}`)
			return
		}
		testcloud.JSON(w, 202, fmt.Sprintf(`{"server":{"id":%q,"status":"BUILD","name":"web"}}`, id))
	})
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/created", func(w http.ResponseWriter, r *http.Request) {
		counts.serverGets.Add(1)
		if scenario == "server HTTP403" {
			testcloud.JSON(w, 403, `{"error":{"message":"denied"}}`)
			return
		}
		id, status, name := "created", "ACTIVE", "web"
		if scenario == "pending" && counts.serverGets.Load() == 1 {
			status = "BUILD"
		}
		if scenario == "server ERROR" {
			status = "ERROR"
		}
		if scenario == "server wrong ID" {
			id = "other"
		}
		if scenario == "server fake ACTIVE" {
			name, status = "ACTIVE", "BUILD"
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"server":{"id":%q,"name":%q,"status":%q,"addresses":{"private":[]}}}`, id, name, status))
	})
	cloud.Mux.HandleFunc("GET /network/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		counts.ports.Add(1)
		if scenario == "pending" && counts.serverGets.Load() != 2 {
			t.Error("IP setup before second ACTIVE observation")
		}
		if counts.serverGets.Load() == 0 || r.URL.Query().Get("device_id") != "created" {
			t.Error("IP setup before same server was ACTIVE", r.URL)
		}
		ips := `[{"ip_address":"10.0.0.10"}]`
		if scenario == "port ambiguous" {
			ips = `[{"ip_address":"10.0.0.10"},{"ip_address":"10.0.0.11"}]`
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"ports":[{"id":"port","device_id":"created","network_id":"private","fixed_ips":%s}]}`, ips))
	})
	cloud.Mux.HandleFunc("GET /network/v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		counts.lists.Add(1)
		if r.URL.Query().Get("project_id") != "owner" {
			t.Error(r.URL)
		}
		if scenario == "list HTTP403" {
			testcloud.JSON(w, 403, `{"error":{"message":"denied"}}`)
			return
		}
		if scenario == "allocation" || strings.HasPrefix(scenario, "IP ") {
			testcloud.JSON(w, 200, `{"floatingips":[]}`)
			return
		}
		testcloud.JSON(w, 200, `{"floatingips":[{"id":"fip","project_id":"owner","floating_network_id":"external","floating_ip_address":"198.51.100.10","revision_number":0}]}`)
	})
	respond := func(w http.ResponseWriter, statusCode int, status string) {
		testcloud.JSON(w, statusCode, fmt.Sprintf(`{"floatingip":{"id":"fip","project_id":"owner","floating_network_id":"external","floating_ip_address":"198.51.100.10","port_id":"port","fixed_ip_address":"10.0.0.10","status":%q}}`, status))
	}
	cloud.Mux.HandleFunc("PUT /network/v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
		counts.puts.Add(1)
		if r.Header.Get("If-Match") != "revision_number=0" {
			t.Error(r.Header)
		}
		if scenario == "PUT412" {
			testcloud.JSON(w, 412, `{"error":{"message":"revision conflict"}}`)
			return
		}
		respond(w, 200, "DOWN")
	})
	cloud.Mux.HandleFunc("POST /network/v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		counts.allocates.Add(1)
		respond(w, 201, "DOWN")
	})
	cloud.Mux.HandleFunc("GET /network/v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
		counts.ipGets.Add(1)
		switch scenario {
		case "IP wait ERROR":
			respond(w, 200, "ERROR")
		case "IP wait403":
			testcloud.JSON(w, 403, `{"error":{"message":"denied"}}`)
		case "IP wrong destination":
			testcloud.JSON(w, 200, `{"floatingip":{"id":"fip","project_id":"owner","floating_network_id":"external","floating_ip_address":"198.51.100.10","port_id":"wrong","fixed_ip_address":"10.0.0.11","status":"ACTIVE"}}`)
		case "IP fake ACTIVE":
			testcloud.JSON(w, 200, `{"floatingip":{"id":"fip","project_id":"owner","floating_network_id":"external","floating_ip_address":"198.51.100.10","port_id":"port","fixed_ip_address":"10.0.0.10","description":"ACTIVE","status":"DOWN"}}`)
		case "IP deadline":
			<-r.Context().Done()
		default:
			respond(w, 200, "ACTIVE")
		}
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected lookup/retry/cleanup", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	return counts
}

func TestConnectionServerFloatingIPOrdersWaitsAndPreservesPartialResults(t *testing.T) {
	for _, scenario := range []string{"reuse", "allocation", "reuse false", "policy reset", "server ERROR", "server HTTP403", "server wrong ID", "server fake ACTIVE", "create unsafe ID", "create null", "port ambiguous", "list HTTP403", "PUT412", "IP wait ERROR", "IP wait403", "IP wrong destination", "IP fake ACTIVE"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			counts := setupServerIPFixture(t, cloud, scenario)
			service, err := connection(t, cloud).Compute(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			var options []compute.CreateServerWithFloatingIPOption
			if scenario == "policy reset" {
				plain, prepareErr := network.PrepareEnsureFloatingIPOptions(context.Background())
				if prepareErr != nil {
					t.Fatal(prepareErr)
				}
				options = append(options, compute.WithFloatingIPOptions(network.WithEnsureWait(resource.WithTimeout(time.Second)), network.WithEnsureFloatingIPPolicy(plain)))
			}
			if scenario == "reuse false" {
				options = append(options, compute.WithFloatingIPOptions(network.WithEnsureReuse(false)))
			}
			if scenario == "server fake ACTIVE" {
				options = append(options, compute.WithServerOptions(compute.WithWait(resource.WithStatusAttribute("name"))))
			}
			if scenario == "IP fake ACTIVE" {
				options = append(options, compute.WithFloatingIPOptions(network.WithEnsureWait(resource.WithStatusAttribute("description"))))
			}
			result, err := service.Servers.CreateWithFloatingIP(context.Background(), serverFloatingIPRequest(), options...)
			success := scenario == "reuse" || scenario == "allocation" || scenario == "reuse false" || scenario == "policy reset"
			if counts.creates.Load() != 1 || result == nil || (err == nil) != success {
				t.Fatalf("result=%+v err=%v creates=%d", result, err, counts.creates.Load())
			}
			if scenario == "create null" || scenario == "create unsafe ID" {
				if counts.serverGets.Load() != 0 || counts.ports.Load() != 0 || result.Assignment != nil {
					t.Fatalf("invalid creation advanced: %+v", result)
				}
				return
			}
			if result.Server == nil || result.Server.ID != "created" {
				t.Fatalf("lost actual server: %+v", result)
			}
			if strings.HasPrefix(scenario, "server ") {
				if result.Server.Status != "BUILD" || result.Assignment != nil || counts.ports.Load() != 0 {
					t.Fatalf("server wait failure advanced/lost POST: %+v", result)
				}
				return
			}
			if result.Server.Status != "ACTIVE" || len(result.Server.Addresses["private"].([]any)) != 0 || result.Server.AccessIPv4 != "" {
				t.Error("server replaced or addresses synthesized", result.Server)
			}
			if scenario == "port ambiguous" || scenario == "list HTTP403" {
				if result.Assignment != nil || counts.puts.Load() != 0 || counts.allocates.Load() != 0 {
					t.Fatalf("selection error mutated: %+v", result)
				}
				return
			}
			if result.Assignment == nil || result.Assignment.FloatingIP == nil || result.Assignment.FloatingIP.ID != "fip" {
				t.Fatalf("lost IP: %+v err=%v", result, err)
			}
			if scenario == "PUT412" {
				var code gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &code) || code.Actual != 412 || !result.Assignment.Reused || counts.allocates.Load() != 0 || counts.ipGets.Load() != 0 {
					t.Fatalf("failed claim lost/fell back: %+v err=%v", result, err)
				}
			} else if success && (result.Assignment.FloatingIP.Status != "ACTIVE" || counts.ipGets.Load() != 1) {
				t.Fatalf("mandatory IP wait omitted: %+v", result)
			} else if !success && result.Assignment.FloatingIP.Status != "DOWN" {
				t.Fatalf("wait failure overwrote accepted IP: %+v", result)
			}
		})
	}
}

type serverWorkflowTransport func(*http.Request) (*http.Response, error)

func (f serverWorkflowTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestConnectionServerFloatingIPUsesOneDeadlineAcrossBothWaits(t *testing.T) {
	for _, scenario := range []string{"success", "IP deadline", "parent deadline", "unlimited", "limit restored"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			fixture := scenario
			if scenario == "parent deadline" {
				fixture = "IP deadline"
			}
			setupServerIPFixture(t, cloud, fixture)
			base := cloud.Provider.HTTPClient.Transport
			if base == nil {
				base = http.DefaultTransport
			}
			var mutex sync.Mutex
			var deadlines []time.Time
			cloud.Provider.HTTPClient.Transport = serverWorkflowTransport(func(r *http.Request) (*http.Response, error) {
				deadline, set := r.Context().Deadline()
				if set != (scenario != "unlimited") {
					t.Error("unexpected deadline presence", scenario, r.URL)
				}
				mutex.Lock()
				deadlines = append(deadlines, deadline)
				mutex.Unlock()
				return base.RoundTrip(r)
			})
			ctx := context.Background()
			limit := time.Second
			if scenario == "IP deadline" {
				limit = 100 * time.Millisecond
			}
			if scenario == "parent deadline" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
			}
			service, err := connection(t, cloud).Compute(ctx)
			if err != nil {
				t.Fatal(err)
			}
			options := []compute.CreateServerWithFloatingIPOption{compute.WithWorkflowTimeout(limit), compute.WithServerOptions(compute.WithWait(resource.WithUnlimitedWait())), compute.WithFloatingIPOptions(network.WithEnsureWait(resource.WithUnlimitedWait()))}
			if scenario == "unlimited" || scenario == "parent deadline" {
				options = append(options, compute.WithUnlimitedWorkflowTimeout())
			}
			if scenario == "limit restored" {
				options = append(options, compute.WithUnlimitedWorkflowTimeout(), compute.WithWorkflowTimeout(time.Second))
			}
			result, err := service.Servers.CreateWithFloatingIP(ctx, serverFloatingIPRequest(), options...)
			expectTimeout := scenario == "IP deadline" || scenario == "parent deadline"
			if !expectTimeout && err != nil || expectTimeout && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal(err)
			}
			if result == nil || result.Server.Status != "ACTIVE" || result.Assignment == nil || result.Assignment.FloatingIP.ID != "fip" {
				t.Fatalf("lost resources: %+v err=%v", result, err)
			}
			mutex.Lock()
			defer mutex.Unlock()
			if len(deadlines) != 6 {
				t.Fatalf("unexpected stages: %v", deadlines)
			}
			for _, deadline := range deadlines {
				if !deadline.Equal(deadlines[0]) {
					t.Fatalf("budget reset across stages: %v", deadlines)
				}
			}
		})
	}
}

func TestConnectionServerFloatingIPKeepsBootAndDefaultNetworkPolicies(t *testing.T) {
	for _, boot := range []string{"image", "existing volume", "new volume"} {
		t.Run(boot, func(t *testing.T) {
			cloud := testcloud.New(t)
			setupServerIPFixture(t, cloud, "reuse", func(r *http.Request) {
				var body struct {
					Server map[string]any `json:"server"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				nics, _ := body.Server["networks"].([]any)
				if len(nics) != 1 || nics[0].(map[string]any)["uuid"] != "private" {
					t.Error(body.Server)
				}
				if boot == "image" {
					if body.Server["imageRef"] != "image" {
						t.Error(body.Server)
					}
				} else {
					mapping := body.Server["block_device_mapping_v2"].([]any)[0].(map[string]any)
					want := "volume"
					if boot == "new volume" {
						want = "image"
					}
					if mapping["source_type"] != want || mapping["delete_on_termination"] != false {
						t.Error(mapping)
					}
				}
			})
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute/v2.1/project"), sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network"), sdk.WithDefaultNetwork(resource.ID("private")))
			if err != nil {
				t.Fatal(err)
			}
			service, err := conn.Compute(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			request := serverFloatingIPRequest()
			var options []compute.CreateServerOption
			if boot == "existing volume" {
				request.Server.Image = resource.Ref{}
				options = append(options, compute.WithBootVolume(resource.ID("volume")))
			}
			if boot == "new volume" {
				options = append(options, compute.WithBootVolumeSize(10))
			}
			result, err := service.Servers.CreateWithFloatingIP(context.Background(), request, compute.WithServerOptions(options...))
			if err != nil || result == nil || result.Server.ID != "created" || result.Assignment.FloatingIP.Status != "ACTIVE" {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}

func TestConnectionServerFloatingIPOptionsSnapshotConcurrentReuse(t *testing.T) {
	cloud := testcloud.New(t)
	counts := setupServerIPFixture(t, cloud, "reuse false", func(r *http.Request) {
		var body struct {
			Server map[string]any `json:"server"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		metadata, _ := body.Server["metadata"].(map[string]any)
		if metadata["owner"] != "original" {
			t.Error("option slice/body changed", body.Server)
		}
	})
	service, err := connection(t, cloud).Compute(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	metadata := map[string]string{"owner": "original"}
	serverInput := []compute.CreateServerOption{compute.WithMetadata(metadata)}
	ipInput := []network.EnsureFloatingIPOption{network.WithEnsureReuse(false)}
	options := []compute.CreateServerWithFloatingIPOption{compute.WithServerOptions(serverInput...), compute.WithFloatingIPOptions(ipInput...)}
	serverInput[0] = compute.WithMetadata(map[string]string{"owner": "replaced"})
	ipInput[0] = network.WithEnsureReuse(true)
	metadata["owner"] = "mutated"
	var wg sync.WaitGroup
	results := make([]*compute.ServerFloatingIPResult, 2)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var err error
			results[i], err = service.Servers.CreateWithFloatingIP(context.Background(), serverFloatingIPRequest(), options...)
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if counts.creates.Load() != 2 || counts.allocates.Load() != 2 || counts.lists.Load() != 0 || counts.ipGets.Load() != 2 {
		t.Fatalf("creates=%d allocations=%d lists=%d GETs=%d", counts.creates.Load(), counts.allocates.Load(), counts.lists.Load(), counts.ipGets.Load())
	}
	if results[0] == nil || results[1] == nil || results[0].Server == results[1].Server || results[0].Assignment == results[1].Assignment || results[0].Assignment.FloatingIP == results[1].Assignment.FloatingIP {
		t.Fatal("results alias across calls", results)
	}
}

func TestConnectionServerFloatingIPPollsServerBeforeStartingIP(t *testing.T) {
	cloud := testcloud.New(t)
	counts := setupServerIPFixture(t, cloud, "pending")
	service, err := connection(t, cloud).Compute(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var callbacks atomic.Int32
	result, err := service.Servers.CreateWithFloatingIP(context.Background(), serverFloatingIPRequest(), compute.WithServerOptions(compute.WithWait(resource.WithPollInterval(time.Millisecond), resource.WithProgressCallback(func(int) { callbacks.Add(1) }))))
	if err != nil || result == nil || result.Server.Status != "ACTIVE" || result.Assignment == nil || counts.serverGets.Load() != 2 || callbacks.Load() != 1 || counts.ports.Load() != 1 {
		t.Fatalf("result=%+v err=%v serverGETs=%d callbacks=%d ports=%d", result, err, counts.serverGets.Load(), callbacks.Load(), counts.ports.Load())
	}
}

func TestConnectionServerFloatingIPMissingNetworkStopsBeforeNovaPOST(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected mutation/lookup", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	missing := errors.New("Neutron endpoint missing")
	var catalogs atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		catalogs.Add(1)
		if opts.Type != "network" {
			t.Error(opts)
		}
		return "", missing
	}
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute/v2.1/project"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.Compute(context.Background())
	if err != nil || catalogs.Load() != 0 {
		t.Fatalf("eager network dependency: %v catalogs=%d", err, catalogs.Load())
	}
	result, err := service.Servers.CreateWithFloatingIP(context.Background(), serverFloatingIPRequest())
	if result != nil || !errors.Is(err, missing) || catalogs.Load() != 1 {
		t.Fatalf("result=%+v err=%v catalogs=%d", result, err, catalogs.Load())
	}
}
