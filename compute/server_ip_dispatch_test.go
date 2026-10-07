package compute_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/compute"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

var dispatchAddresses = map[string]string{"a": "198.51.100.10", "b": "198.51.100.11", "c": "198.51.100.12", "pool": "198.51.100.20"}

type dispatchFixture struct {
	cloud   *testcloud.Cloud
	service *compute.Service
	network *network.Service
	fail    string
	mu      sync.Mutex
	events  []string
	bound   map[string]bool
	target  string
	raw     atomic.Int32
	posts   atomic.Int32
}

func (f *dispatchFixture) event(value string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, value)
}

func (f *dispatchFixture) trace() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.events...)
}

func dispatchIP(key string, attached bool, status string) string {
	port, fixed, owner, external := "", "", "foreign", "selected-ip-network"
	if attached {
		port, fixed = "port", "10.0.0.10"
	}
	if key == "pool" {
		owner, external = "owner", "pool-network"
	}
	return fmt.Sprintf(`{"id":"ip-%s","project_id":%q,"floating_network_id":%q,"floating_ip_address":%q,"port_id":%q,"fixed_ip_address":%q,"status":%q,"revision_number":0}`, key, owner, external, dispatchAddresses[key], port, fixed, status)
}

func newDispatchFixture(t *testing.T, fail string) *dispatchFixture {
	t.Helper()
	f := &dispatchFixture{cloud: testcloud.New(t), fail: fail, bound: make(map[string]bool)}
	f.network = network.New(f.cloud.Client("network", "/v2.0"))
	f.service = compute.New(f.cloud.Client("compute", "/v2.1"), compute.Dependencies{AddressNetworks: func(context.Context) (*network.Service, error) { return f.network, nil }})
	f.cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		key := "pool"
		if address := r.URL.Query().Get("floating_ip_address"); address != "" {
			key = ""
			for candidate, value := range dispatchAddresses {
				if value == address {
					key = candidate
				}
			}
			if key == "" || len(r.URL.Query()) != 1 {
				t.Error("invalid explicit filter", r.URL)
			}
		} else if r.URL.Query().Get("project_id") != "owner" || r.URL.Query().Get("floating_network_id") != "pool-network" {
			t.Error("invalid pool filter", r.URL)
		}
		f.event("list:" + key)
		if key == "b" && f.fail == "missing" {
			testcloud.JSON(w, 200, `{"floatingips":[]}`)
			return
		}
		if key == "b" && f.fail == "HTTP" {
			http.Error(w, "second lookup denied", 403)
			return
		}
		f.mu.Lock()
		attached := f.bound[key]
		f.mu.Unlock()
		row := dispatchIP(key, attached, "DOWN")
		if key == "b" && f.fail == "ambiguous" {
			row += "," + strings.Replace(row, "ip-b", "duplicate-b", 1)
		}
		testcloud.JSON(w, 200, `{"floatingips":[`+row+`]}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		f.event("ports")
		if r.URL.Query().Get("device_id") != "server" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"ports":[`+autoPort+`]}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.0/ports/port", func(w http.ResponseWriter, r *http.Request) {
		f.event("port")
		testcloud.JSON(w, 200, `{"port":`+autoPort+`}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.0/floatingips/{id}", func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.PathValue("id"), "ip-")
		f.event("get:" + key)
		f.mu.Lock()
		attached := f.bound[key]
		f.mu.Unlock()
		status := "DOWN"
		if attached && !(key == "b" && f.fail == "IP cancel") {
			status = "ACTIVE"
		}
		testcloud.JSON(w, 200, `{"floatingip":`+dispatchIP(key, attached, status)+`}`)
	})
	f.cloud.Mux.HandleFunc("PUT /v2.0/floatingips/{id}", func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.PathValue("id"), "ip-")
		f.event("put:" + key)
		var body map[string]map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(body, map[string]map[string]any{"floatingip": {"port_id": "port", "fixed_ip_address": "10.0.0.10"}}) || r.Header.Get("If-Match") != "revision_number=0" {
			t.Error(body, r.Header)
		}
		if key == "b" && f.fail == "412" {
			http.Error(w, "second conflict", 412)
			return
		}
		f.mu.Lock()
		f.bound[key], f.target = true, key
		f.mu.Unlock()
		row := dispatchIP(key, true, "DOWN")
		if key == "b" && f.fail == "wrong address" {
			row = strings.Replace(row, dispatchAddresses["b"], dispatchAddresses["c"], 1)
		}
		w.Header().Set("X-Dispatch-Proof", "accepted-"+key)
		testcloud.JSON(w, 200, `{"floatingip":`+row+`}`)
	})
	f.cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		f.event("allocate:pool")
		f.posts.Add(1)
		var body map[string]map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(body["floatingip"], map[string]any{"floating_network_id": "pool-network", "port_id": "port", "fixed_ip_address": "10.0.0.10", "project_id": "owner"}) {
			t.Error(body)
		}
		f.mu.Lock()
		f.bound["pool"], f.target = true, "pool"
		f.mu.Unlock()
		testcloud.JSON(w, 201, `{"floatingip":`+dispatchIP("pool", true, "DOWN")+`}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.1/servers/server", func(w http.ResponseWriter, r *http.Request) {
		n := f.raw.Add(1)
		f.mu.Lock()
		key := f.target
		f.mu.Unlock()
		f.event("raw:" + key)
		addresses := autoFixed
		if key != "" && !(key == "b" && (f.fail == "raw cancel" || f.fail == "raw source")) {
			addresses = fmt.Sprintf(`{"private":[{"version":4,"addr":"10.0.0.10","OS-EXT-IPS:type":"fixed"},{"version":4,"addr":%q,"OS-EXT-IPS:type":"floating"}]}`, dispatchAddresses[key])
		}
		testcloud.JSON(w, 203, fmt.Sprintf(`{"server":{"id":"server","name":"raw-%d","status":"ACTIVE","addresses":%s}}`, n, addresses))
	})
	f.cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected dispatch request", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	return f
}

func dispatchOptions() []compute.AutomaticFloatingIPOption {
	return []compute.AutomaticFloatingIPOption{
		compute.WithAutomaticIPTimeout(time.Second), compute.WithAutomaticIPPollInterval(time.Millisecond),
		compute.WithAutomaticEnsureOptions(network.WithEnsureProject("owner"), network.WithEnsureWait(resource.WithPollInterval(time.Millisecond))),
		compute.WithAutomaticAddressOptions(compute.WithAddressReachability(false)),
	}
}

func dispatchInput(t *testing.T) compute.AutomaticFloatingIPRequest {
	return compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed), Network: resource.ID("ignored/network")}
}

func TestServerIPDispatchPreservesOrderedDuplicateInputAndOneCompletedStepAtATime(t *testing.T) {
	f := newDispatchFixture(t, "")
	addresses := []string{dispatchAddresses["a"], dispatchAddresses["a"], dispatchAddresses["b"]}
	option := compute.WithFloatingIPAddresses(addresses...)
	addresses[0] = dispatchAddresses["c"]
	opts := append(dispatchOptions(), option, compute.WithAutomaticIPEnabled(false), compute.WithAutomaticAddressOptions(compute.WithPrivateCloud(true)))
	result, err := f.service.EnsureServerFloatingIP(context.Background(), dispatchInput(t), opts...)
	if err != nil || result == nil || !result.Observed || result.Mode != compute.ServerIPExplicit || result.Decision.Reason != compute.AutomaticIPAddressesRequested || result.Assignment.FloatingIP.ID != "ip-b" || len(result.Attempts) != 3 || result.Server.Name != "raw-3" || f.posts.Load() != 0 {
		t.Fatal(result, err, f.trace())
	}
	for i, attempt := range result.Attempts {
		want := dispatchAddresses["a"]
		if i == 2 {
			want = dispatchAddresses["b"]
		}
		if attempt.Index != i || attempt.RequestedAddress != want || !attempt.Completed || !attempt.Observed || attempt.Error != nil || attempt.Assignment == nil || !attempt.Assignment.Reused || attempt.Assignment.Allocated {
			t.Fatal(attempt)
		}
	}
	want := []string{"list:a", "ports", "port", "get:a", "put:a", "get:a", "raw:a", "list:a", "ports", "port", "get:a", "get:a", "raw:a", "list:b", "ports", "port", "get:b", "put:b", "get:b", "raw:b"}
	if !reflect.DeepEqual(f.trace(), want) || len(result.Decision.AttachmentSelections) != 3 || result.Decision.Selection != (network.FloatingIPSelection{}) {
		t.Fatal(f.trace(), result.Decision)
	}
}

func TestServerIPDispatchSecondFailureRetainsFirstCompletionAndCurrentPartial(t *testing.T) {
	for _, failure := range []string{"missing", "ambiguous", "HTTP", "412", "wrong address", "close", "source", "IP cancel", "raw cancel", "raw source"} {
		t.Run(failure, func(t *testing.T) {
			f := newDispatchFixture(t, failure)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("second IP failure")
			base := f.cloud.Provider.HTTPClient.Transport
			if failure == "close" || failure == "source" {
				f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
					response, err := base.RoundTrip(r)
					if err == nil && r.Method == "PUT" && r.URL.Path == "/v2.0/floatingips/ip-b" {
						original := response.Body
						response.Body = automaticCloseBody{ReadCloser: original, close: func() error {
							if failure == "source" {
								f.network.API = nil
							}
							return cause
						}}
					}
					return response, err
				})
			}
			opts := append(dispatchOptions(), compute.WithFloatingIPAddresses(dispatchAddresses["a"], dispatchAddresses["b"], dispatchAddresses["c"]))
			if failure == "IP cancel" {
				opts = append(opts, compute.WithAutomaticEnsureOptions(network.WithEnsureWait(resource.WithProgressCallback(func(int) { cancel(cause) }))))
			}
			if failure == "raw cancel" || failure == "raw source" {
				opts = append(opts, compute.WithAutomaticIPProgress(func(*compute.Server) error {
					if failure == "raw source" {
						f.service.API = nil
					} else {
						cancel(cause)
					}
					return nil
				}))
			}
			result, err := f.service.EnsureServerFloatingIP(ctx, dispatchInput(t), opts...)
			if err == nil || result == nil || result.Observed || len(result.Attempts) != 2 || !result.Attempts[0].Completed || !result.Attempts[0].Observed || result.Attempts[0].Assignment.FloatingIP.ID != "ip-a" || result.Attempts[1].Completed || result.Attempts[1].Observed || result.Attempts[1].Error == nil || f.posts.Load() != 0 {
				t.Fatal(result, err, f.trace())
			}
			if strings.Contains(strings.Join(f.trace(), ","), ":c") {
				t.Fatal("started third IP", f.trace())
			}
			wantID, wantServer := "ip-b", "raw-1"
			if failure == "missing" || failure == "ambiguous" || failure == "HTTP" {
				wantID = "ip-a"
				if result.Attempts[1].Assignment != nil {
					t.Fatal(result.Attempts[1])
				}
			}
			if failure == "raw cancel" || failure == "raw source" {
				wantServer = "raw-2"
			}
			if result.Assignment.FloatingIP.ID != wantID || result.Server.Name != wantServer {
				t.Fatal(result.Assignment, result.Server, err)
			}
			switch failure {
			case "missing":
				if !errors.Is(err, resource.ErrNotFound) {
					t.Fatal(err)
				}
			case "ambiguous":
				if !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(err)
				}
			case "HTTP", "412":
				var native gophercloud.ErrUnexpectedResponseCode
				want := 403
				if failure == "412" {
					want = 412
				}
				if !errors.As(err, &native) || native.Actual != want {
					t.Fatal(err, native)
				}
			case "wrong address":
				var proof *resource.ResponseError
				if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || result.Assignment.FloatingIP.PortID != "" {
					t.Fatal(err, proof, result.Assignment)
				}
			case "close", "source":
				var proof *resource.ResponseError
				if !errors.Is(err, cause) || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Dispatch-Proof") != "accepted-b" || result.Assignment.FloatingIP.PortID != "port" {
					t.Fatal(err, proof, result.Assignment)
				}
				if failure == "source" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			case "IP cancel", "raw cancel":
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
					t.Fatal(err)
				}
			case "raw source":
				if !errors.Is(err, resource.ErrInvalidOption) || f.raw.Load() != 2 {
					t.Fatal(err, f.raw.Load())
				}
			}
		})
	}
}

func TestServerIPDispatchPoolWinsIndependentOfOptionOrderAndIgnoresIPSelectors(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for _, reuse := range []bool{false, true} {
			t.Run(fmt.Sprintf("reverse=%v/reuse=%v", reverse, reuse), func(t *testing.T) {
				f := newDispatchFixture(t, "")
				pool, ips := compute.WithFloatingIPPool(resource.ID("pool-network")), compute.WithFloatingIPAddresses("invalid ignored address")
				selectors := []compute.AutomaticFloatingIPOption{pool, ips}
				if reverse {
					selectors = []compute.AutomaticFloatingIPOption{ips, pool}
				}
				opts := append(dispatchOptions(), selectors...)
				opts = append(opts, compute.WithAutomaticEnsureOptions(network.WithEnsureReuse(reuse)), compute.WithAutomaticIPEnabled(false), compute.WithAutomaticAddressOptions(compute.WithPrivateCloud(true)))
				input := dispatchInput(t)
				input.Server = automaticServer(t, autoFloating)
				input.Server.AccessIPv4 = "8.8.4.4"
				result, err := f.service.EnsureServerFloatingIP(context.Background(), input, opts...)
				if err != nil || result == nil || !result.Observed || result.Mode != compute.ServerIPPool || result.Decision.Reason != compute.AutomaticIPPoolRequested || result.Decision.Selection.NetworkID != "pool-network" || len(result.Decision.AttachmentSelections) != 0 || len(result.Attempts) != 1 || !result.Attempts[0].Completed || result.Assignment.Allocated == reuse || result.Assignment.Reused != reuse || result.Assignment.FloatingIP.ID != "ip-pool" {
					t.Fatal(result, err, f.trace())
				}
				trace := strings.Join(f.trace(), ",")
				if strings.Contains(trace, "list:a") || strings.Contains(trace, "list:b") || strings.Contains(trace, "list:c") {
					t.Fatal(trace)
				}
				if reuse && f.posts.Load() != 0 || !reuse && f.posts.Load() != 1 {
					t.Fatal(f.posts.Load(), trace)
				}
			})
		}
	}
}

func TestServerIPDispatchReadOnlyPlanKeepsOrderedSelectionsAndNeverClaimsMutation(t *testing.T) {
	for _, fail := range []string{"", "missing"} {
		f := newDispatchFixture(t, fail)
		opts := append(dispatchOptions(), compute.WithFloatingIPAddresses(dispatchAddresses["a"], dispatchAddresses["b"]))
		decision, err := f.service.PlanServerFloatingIP(context.Background(), dispatchInput(t), opts...)
		want := 2
		if fail != "" {
			want = 1
		}
		if decision == nil || !decision.Needed || decision.Mode != compute.ServerIPExplicit || decision.Backend != compute.FloatingIPNeutron || len(decision.AttachmentSelections) != want || decision.Selection != (network.FloatingIPSelection{}) || f.raw.Load() != 0 || f.posts.Load() != 0 || strings.Contains(strings.Join(f.trace(), ","), "put:") {
			t.Fatal(decision, err, f.trace())
		}
		if fail == "" && err != nil || fail != "" && !errors.Is(err, resource.ErrNotFound) {
			t.Fatal(err)
		}
	}
}

func TestServerIPDispatchBackendAndSkippedAutomaticConditionsStayDistinct(t *testing.T) {
	for _, source := range []compute.FloatingIPSource{compute.FloatingIPNeutron, compute.FloatingIPNova, compute.FloatingIPNone} {
		for _, pool := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/pool=%v", source, pool), func(t *testing.T) {
				calls := 0
				service := compute.New(nil, compute.Dependencies{AddressNetworks: func(context.Context) (*network.Service, error) { calls++; return nil, nil }})
				selector := compute.WithFloatingIPAddresses(dispatchAddresses["a"])
				if pool {
					selector = compute.WithFloatingIPPool(resource.ID("pool-network"))
				}
				opts := []compute.AutomaticFloatingIPOption{selector, compute.WithAutomaticIPEnabled(false), compute.WithAutomaticAddressOptions(compute.WithFloatingIPSource(source), compute.WithPrivateCloud(true))}
				input := dispatchInput(t)
				decision, err := service.PlanServerFloatingIP(context.Background(), input, opts...)
				if err != nil || decision == nil || !decision.Needed || decision.Backend != compute.FloatingIPNova {
					t.Fatal(decision, err)
				}
				result, err := service.EnsureServerFloatingIP(context.Background(), input, opts...)
				if !errors.Is(err, resource.ErrUnsupported) || result == nil || !result.Decision.Needed || len(result.Attempts) != 0 || result.Assignment != nil || result.Observed {
					t.Fatal(result, err)
				}
				want := 0
				if source == compute.FloatingIPNeutron {
					want = 2
				}
				if calls != want {
					t.Fatal(calls)
				}
			})
		}
	}
}

func TestServerIPDispatchClearsSelectorsAndValidatesOnlyWinningSelector(t *testing.T) {
	f := newDispatchFixture(t, "")
	opts := append(dispatchOptions(), compute.WithFloatingIPPool(resource.ID("bad/pool")), compute.WithFloatingIPAddresses(dispatchAddresses["a"]), compute.WithFloatingIPPool(resource.Ref{}))
	result, err := f.service.EnsureServerFloatingIP(context.Background(), dispatchInput(t), opts...)
	if err != nil || result.Mode != compute.ServerIPExplicit || result.Assignment.FloatingIP.ID != "ip-a" {
		t.Fatal(result, err)
	}
	opts = []compute.AutomaticFloatingIPOption{compute.WithFloatingIPAddresses("invalid"), compute.WithFloatingIPAddresses(), compute.WithFloatingIPPool(resource.Ref{}), compute.WithAutomaticIPEnabled(false)}
	input := dispatchInput(t)
	input.Network = resource.Ref{}
	result, err = f.service.EnsureServerFloatingIP(context.Background(), input, opts...)
	if err != nil || result.Mode != compute.ServerIPAutomatic || result.Decision.Reason != compute.AutomaticIPDisabled {
		t.Fatal(result, err)
	}
	for _, selector := range []compute.AutomaticFloatingIPOption{compute.WithFloatingIPAddresses("invalid"), compute.WithFloatingIPPool(resource.ID("bad/pool"))} {
		before := len(f.trace())
		if result, err := f.service.EnsureServerFloatingIP(context.Background(), dispatchInput(t), selector); result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(f.trace()) != before {
			t.Fatal(result, err, f.trace())
		}
	}
}
