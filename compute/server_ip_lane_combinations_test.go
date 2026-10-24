package compute_test

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

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// laneFixture serves the standalone AddIPsToServer pool and automatic lanes
// with two server ports: port-p on private and port-n on nat-net. free makes
// the pool listing return one unattached IP; otherwise the lane allocates.
type laneFixture struct {
	cloud    *testcloud.Cloud
	service  *compute.Service
	network  *network.Service
	mu       sync.Mutex
	events   []string
	bodies   []map[string]any
	free     bool
	bound    [2]string
	rawCalls atomic.Int32
	// shownAfter is the raw GET number from which the server shows the
	// assigned floating address; zero means immediately after attachment.
	shownAfter int32
	// rawStatus gives the raw server status per GET; nil means BUILD.
	rawStatus func(n int32) string
	// ipStatus is the floating IP status in every row; empty means DOWN.
	ipStatus string
}

func (f *laneFixture) event(value string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, value)
}

func (f *laneFixture) trace() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return strings.Join(f.events, ",")
}

func (f *laneFixture) ipRow() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	port, fixed := "null", "null"
	if f.bound[0] != "" {
		port, fixed = fmt.Sprintf("%q", f.bound[0]), fmt.Sprintf("%q", f.bound[1])
	}
	status := f.ipStatus
	if status == "" {
		status = "DOWN"
	}
	return fmt.Sprintf(`{"id":"ip-pool","project_id":"owner","floating_network_id":"pool-network","floating_ip_address":%q,"port_id":%s,"fixed_ip_address":%s,"status":%q,"revision_number":0}`, dispatchAddresses["pool"], port, fixed, status)
}

func newLaneFixture(t *testing.T) *laneFixture {
	t.Helper()
	f := &laneFixture{cloud: testcloud.New(t)}
	f.network = network.New(f.cloud.Client("network", "/v2.0"))
	f.service = compute.New(f.cloud.Client("compute", "/v2.1"), compute.Dependencies{AddressNetworks: func(context.Context) (*network.Service, error) { return f.network, nil }})
	bind := func(body map[string]any) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.bodies = append(f.bodies, body)
		f.bound = [2]string{fmt.Sprint(body["port_id"]), fmt.Sprint(body["fixed_ip_address"])}
	}
	f.cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		f.event("list:" + r.URL.RawQuery)
		f.mu.Lock()
		free, bound := f.free, f.bound[0] != ""
		f.mu.Unlock()
		if !free && !bound {
			testcloud.JSON(w, 200, `{"floatingips":[]}`)
			return
		}
		testcloud.JSON(w, 200, `{"floatingips":[`+f.ipRow()+`]}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.0/floatingips/ip-pool", func(w http.ResponseWriter, r *http.Request) {
		f.event("get")
		testcloud.JSON(w, 200, `{"floatingip":`+f.ipRow()+`}`)
	})
	f.cloud.Mux.HandleFunc("PUT /v2.0/floatingips/ip-pool", func(w http.ResponseWriter, r *http.Request) {
		f.event("put")
		var body map[string]map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		bind(body["floatingip"])
		testcloud.JSON(w, 200, `{"floatingip":`+f.ipRow()+`}`)
	})
	f.cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		f.event("allocate")
		var body map[string]map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		bind(body["floatingip"])
		testcloud.JSON(w, 201, `{"floatingip":`+f.ipRow()+`}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		f.event("ports")
		if r.URL.Query().Get("device_id") != "server" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"ports":[`+sourcePortPrivate+`,`+sourcePortNAT+`]}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.0/ports/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.event("port:" + r.PathValue("id"))
		for _, port := range []string{sourcePortPrivate, sourcePortNAT} {
			if strings.Contains(port, `"id":"`+r.PathValue("id")+`"`) {
				testcloud.JSON(w, 200, `{"port":`+port+`}`)
				return
			}
		}
		http.Error(w, "port missing", 404)
	})
	networks := `[{"id":"pool-network","name":"external","router:external":true},{"id":"private","name":"private"},{"id":"nat-net","name":"nat"}]`
	f.cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		f.event("networks:" + r.URL.RawQuery)
		if name := r.URL.Query().Get("name"); name == "nat" {
			testcloud.JSON(w, 200, `{"networks":[{"id":"nat-net","name":"nat"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"networks":`+networks+`}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.0/networks/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.event("network:" + r.PathValue("id"))
		switch r.PathValue("id") {
		case "pool-network":
			testcloud.JSON(w, 200, `{"network":{"id":"pool-network","name":"external","router:external":true}}`)
		case "nat-net":
			testcloud.JSON(w, 200, `{"network":{"id":"nat-net","name":"nat"}}`)
		default:
			http.Error(w, "network missing", 404)
		}
	})
	f.cloud.Mux.HandleFunc("GET /v2.0/subnets", func(w http.ResponseWriter, r *http.Request) {
		f.event("subnets")
		testcloud.JSON(w, 200, `{"subnets":[]}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.0/routers", func(w http.ResponseWriter, r *http.Request) {
		f.event("routers")
		testcloud.JSON(w, 200, `{"routers":[]}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.1/servers/server", func(w http.ResponseWriter, r *http.Request) {
		n := f.rawCalls.Add(1)
		f.event("raw")
		f.mu.Lock()
		bound := f.bound[0] != ""
		f.mu.Unlock()
		addresses := sourceAddresses()
		if bound && n >= f.shownAfter {
			addresses = sourceAddresses(dispatchAddresses["pool"])
		}
		status := "BUILD"
		if f.rawStatus != nil {
			status = f.rawStatus(n)
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"server":{"id":"server","name":"raw-%d","status":%q,"addresses":%s}}`, n, status, addresses))
	})
	f.cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected lane request", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	return f
}

func laneOptions(extra ...compute.AutomaticFloatingIPOption) []compute.AutomaticFloatingIPOption {
	return append([]compute.AutomaticFloatingIPOption{compute.WithAutomaticIPPollInterval(time.Millisecond), compute.WithAutomaticAddressOptions(compute.WithAddressReachability(false))}, extra...)
}

// The pool selector forwards fixed_address and nat_destination to both the
// reused-IP association and the new allocation, as Python's
// _add_ip_from_pool passes them to create_floating_ip and
// _attach_ip_to_server.
func TestServerIPPoolLaneForwardsFixedAddressAndNATDestination(t *testing.T) {
	cases := []struct {
		name   string
		option []network.EnsureFloatingIPOption
		port   string
		fixed  string
	}{
		{name: "fixed", option: []network.EnsureFloatingIPOption{network.WithEnsureFixedAddress("10.1.0.20")}, port: "port-n", fixed: "10.1.0.20"},
		{name: "fixed-private", option: []network.EnsureFloatingIPOption{network.WithEnsureFixedAddress("10.0.0.10")}, port: "port-p", fixed: "10.0.0.10"},
		{name: "nat-id", option: []network.EnsureFloatingIPOption{network.WithEnsureNATDestination(resource.ID("nat-net"))}, port: "port-n", fixed: "10.1.0.20"},
		{name: "nat-name", option: []network.EnsureFloatingIPOption{network.WithEnsureNATDestination(resource.Name("nat"))}, port: "port-n", fixed: "10.1.0.20"},
		{name: "nat-fixed", option: []network.EnsureFloatingIPOption{network.WithEnsureNATDestination(resource.ID("nat-net")), network.WithEnsureFixedAddress("10.1.0.20")}, port: "port-n", fixed: "10.1.0.20"},
	}
	for _, tc := range cases {
		for _, reuse := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/reuse=%v", tc.name, reuse), func(t *testing.T) {
				f := newLaneFixture(t)
				f.free = reuse
				ensure := append([]network.EnsureFloatingIPOption{network.WithEnsureProject("owner"), network.WithEnsureReuse(reuse)}, tc.option...)
				server := automaticServer(t, autoFixed)
				result, err := f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, standaloneIPOptions(laneOptions(compute.WithFloatingIPPool(resource.ID("pool-network")), compute.WithAutomaticEnsureOptions(ensure...))...)...)
				trace := f.trace()
				if err != nil || result == nil || result.Mode != compute.ServerIPPool || len(result.Attempts) != 1 || !result.Attempts[0].Completed || result.Observed || result.Assignment == nil || result.Assignment.Allocated == reuse || f.rawCalls.Load() != 0 {
					t.Fatal(result, err, trace)
				}
				if len(f.bodies) != 1 || f.bodies[0]["port_id"] != tc.port || f.bodies[0]["fixed_ip_address"] != tc.fixed {
					t.Fatal(f.bodies, trace)
				}
				if selection := result.Decision.Selection; selection.PortID != tc.port || selection.FixedIPv4 != tc.fixed || selection.NetworkID != "pool-network" {
					t.Fatal(selection)
				}
				if reuse != strings.Contains(trace, "put") || reuse == strings.Contains(trace, "allocate") {
					t.Fatal(trace)
				}
			})
		}
	}
}

// laneDeadlines records every HTTP deadline of the fixture.
func laneDeadlines(t *testing.T, f *laneFixture) func() []time.Time {
	var mu sync.Mutex
	var deadlines []time.Time
	base := f.cloud.Provider.HTTPClient.Transport
	f.cloud.Provider.HTTPClient.Transport = automaticTransport(func(r *http.Request) (*http.Response, error) {
		deadline, ok := r.Context().Deadline()
		if !ok {
			t.Error("request without deadline", r.URL)
		}
		mu.Lock()
		deadlines = append(deadlines, deadline)
		mu.Unlock()
		return base.RoundTrip(r)
	})
	return func() []time.Time {
		mu.Lock()
		defer mu.Unlock()
		return append([]time.Time(nil), deadlines...)
	}
}

func checkLaneDeadlines(t *testing.T, deadlines []time.Time, start time.Time, parent time.Time, mode string) {
	t.Helper()
	if len(deadlines) == 0 {
		t.Fatal("no requests")
	}
	for _, deadline := range deadlines {
		if !deadline.Equal(deadlines[0]) {
			t.Fatal("renewed budget", deadlines)
		}
	}
	switch mode {
	case "default":
		if remaining := deadlines[0].Sub(start); remaining > 61*time.Second || remaining < 59*time.Second {
			t.Fatal("default budget", remaining)
		}
	case "override":
		if remaining := deadlines[0].Sub(start); remaining > 18*time.Second || remaining < 16*time.Second {
			t.Fatal("override budget", remaining)
		}
	default:
		if !deadlines[0].Equal(parent) {
			t.Fatal(mode, deadlines[0], parent)
		}
	}
}

// With WithServerIPWait the pool lane polls raw Nova for the exact address,
// calls the progress callback between unconverged responses and shares one
// deadline: 60 seconds by default, an override, an earlier parent, or the
// parent alone when unlimited.
func TestServerIPPoolLaneWaitProgressAndBudget(t *testing.T) {
	for _, reuse := range []bool{true, false} {
		t.Run(fmt.Sprintf("progress/reuse=%v", reuse), func(t *testing.T) {
			f := newLaneFixture(t)
			f.free, f.shownAfter = reuse, 3
			var seen []string
			progress := compute.WithAutomaticIPProgress(func(server *compute.Server) error {
				seen = append(seen, server.Name)
				return nil
			})
			options := laneOptions(progress, compute.WithFloatingIPPool(resource.ID("pool-network")), compute.WithAutomaticEnsureOptions(network.WithEnsureProject("owner"), network.WithEnsureReuse(reuse), network.WithEnsurePort(resource.ID("port-p"))))
			result, err := f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, append(standaloneIPOptions(options...), compute.WithServerIPWait(true))...)
			if err != nil || !result.Observed || !result.Attempts[0].Observed || !result.Attempts[0].Completed || result.Server.Name != "raw-3" || strings.Join(seen, ",") != "raw-1,raw-2" || result.Assignment.FloatingIP.Status != "DOWN" {
				t.Fatal(result, err, seen, f.trace())
			}
		})
	}
	t.Run("progress-error-stops", func(t *testing.T) {
		f := newLaneFixture(t)
		f.free, f.shownAfter = true, 3
		stop := errors.New("stop polling")
		options := laneOptions(compute.WithAutomaticIPProgress(func(*compute.Server) error { return stop }), compute.WithFloatingIPPool(resource.ID("pool-network")), compute.WithAutomaticEnsureOptions(network.WithEnsureProject("owner"), network.WithEnsurePort(resource.ID("port-p"))))
		result, err := f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, append(standaloneIPOptions(options...), compute.WithServerIPWait(true))...)
		if !errors.Is(err, stop) || result == nil || result.Observed || result.Attempts[0].Completed || result.Assignment == nil || f.rawCalls.Load() != 1 {
			t.Fatal(result, err, f.trace())
		}
	})
	for _, mode := range []string{"default", "override", "parent", "unlimited"} {
		t.Run("deadline-"+mode, func(t *testing.T) {
			f := newLaneFixture(t)
			f.free = true
			deadlines := laneDeadlines(t, f)
			length := 3 * time.Minute
			extra := []compute.AutomaticFloatingIPOption{compute.WithFloatingIPPool(resource.ID("pool-network")), compute.WithAutomaticEnsureOptions(network.WithEnsureProject("owner"), network.WithEnsurePort(resource.ID("port-p")))}
			switch mode {
			case "override":
				extra = append(extra, compute.WithAutomaticIPTimeout(17*time.Second))
			case "parent":
				length = 30 * time.Second
			case "unlimited":
				extra = append(extra, compute.WithUnlimitedAutomaticIPTimeout())
			}
			ctx, cancel := context.WithTimeout(context.Background(), length)
			defer cancel()
			parent, _ := ctx.Deadline()
			start := time.Now()
			result, err := f.service.AddIPsToServer(ctx, compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, append(standaloneIPOptions(laneOptions(extra...)...), compute.WithServerIPWait(true))...)
			if err != nil || !result.Observed {
				t.Fatal(result, err, f.trace())
			}
			checkLaneDeadlines(t, deadlines(), start, parent, mode)
		})
	}
}

// The automatic lane reuses a free IP on the external network or allocates
// on the selected port. Unlike Python, whose auto_ip branch passes
// nat_destination only to _needs_floating_ip and ignores fixed_address, Go
// applies both selectors to classification and the destination. Without a
// selector two candidate ports are ambiguous before any mutation, as Python's
// _nat_destination_port raises without a configured NAT destination.
func TestServerIPAutomaticLaneDestinationSelectorsWaitAndProgress(t *testing.T) {
	for _, name := range []string{"nat-id", "nat-name", "fixed", "none"} {
		for _, reuse := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/reuse=%v", name, reuse), func(t *testing.T) {
				f := newLaneFixture(t)
				f.free = reuse
				ensure := []network.EnsureFloatingIPOption{network.WithEnsureProject("owner"), network.WithEnsureReuse(reuse)}
				switch name {
				case "nat-id":
					ensure = append(ensure, network.WithEnsureNATDestination(resource.ID("nat-net")))
				case "nat-name":
					ensure = append(ensure, network.WithEnsureNATDestination(resource.Name("nat")))
				case "fixed":
					ensure = append(ensure, network.WithEnsureFixedAddress("10.1.0.20"))
				}
				result, err := f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, standaloneIPOptions(laneOptions(compute.WithAutomaticEnsureOptions(ensure...))...)...)
				trace := f.trace()
				if name == "none" {
					if err == nil || !errors.Is(err, resource.ErrAmbiguous) || result == nil || result.Assignment != nil || result.Decision.Needed || len(f.bodies) != 0 || strings.Contains(trace, "put") || strings.Contains(trace, "allocate") {
						t.Fatal(result, err, trace)
					}
					return
				}
				if err != nil || result.Mode != compute.ServerIPAutomatic || !result.Decision.Needed || result.Observed || result.Assignment == nil || result.Assignment.Allocated == reuse || f.rawCalls.Load() != 0 {
					t.Fatal(result, err, trace)
				}
				if len(f.bodies) != 1 || f.bodies[0]["port_id"] != "port-n" || f.bodies[0]["fixed_ip_address"] != "10.1.0.20" || reuse != strings.HasSuffix(trace, "put") {
					t.Fatal(f.bodies, trace)
				}
			})
		}
	}
	t.Run("wait-progress", func(t *testing.T) {
		f := newLaneFixture(t)
		f.shownAfter = 3
		var seen []string
		progress := compute.WithAutomaticIPProgress(func(server *compute.Server) error {
			seen = append(seen, server.Name)
			return nil
		})
		options := laneOptions(progress, compute.WithAutomaticEnsureOptions(network.WithEnsureProject("owner"), network.WithEnsureReuse(false), network.WithEnsurePort(resource.ID("port-p"))))
		result, err := f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, append(standaloneIPOptions(options...), compute.WithServerIPWait(true))...)
		if err != nil || !result.Observed || result.Server.Name != "raw-3" || strings.Join(seen, ",") != "raw-1,raw-2" || !result.Assignment.Allocated || f.bodies[0]["port_id"] != "port-p" {
			t.Fatal(result, err, seen, f.trace())
		}
	})
	for _, mode := range []string{"default", "parent", "unlimited"} {
		t.Run("deadline-"+mode, func(t *testing.T) {
			f := newLaneFixture(t)
			deadlines := laneDeadlines(t, f)
			length := 3 * time.Minute
			extra := []compute.AutomaticFloatingIPOption{compute.WithAutomaticEnsureOptions(network.WithEnsureProject("owner"), network.WithEnsureReuse(false), network.WithEnsurePort(resource.ID("port-p")))}
			switch mode {
			case "parent":
				length = 30 * time.Second
			case "unlimited":
				extra = append(extra, compute.WithUnlimitedAutomaticIPTimeout())
			}
			ctx, cancel := context.WithTimeout(context.Background(), length)
			defer cancel()
			parent, _ := ctx.Deadline()
			start := time.Now()
			result, err := f.service.AddIPsToServer(ctx, compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, append(standaloneIPOptions(laneOptions(extra...)...), compute.WithServerIPWait(true))...)
			if err != nil || !result.Observed {
				t.Fatal(result, err, f.trace())
			}
			checkLaneDeadlines(t, deadlines(), start, parent, mode)
		})
	}
}

// The legacy Nova pool and automatic lanes of AddIPsToServer forward the fixed
// address in the addFloatingIp action (the fixture rejects an action without
// it) and, with WithServerIPWait, observe the exact address in raw Nova after
// the accepted action, as Python's _attach_ip_to_server waits after
// _nova_attach_ip_to_server.
func TestServerIPNovaPoolAndAutomaticLanesWait(t *testing.T) {
	cases := []struct {
		name   string
		extra  []compute.AutomaticFloatingIPOption
		mode   compute.ServerIPDispatchMode
		expect []string
	}{
		{name: "pool-reuse", extra: []compute.AutomaticFloatingIPOption{compute.WithFloatingIPPool(resource.Name("public"))}, mode: compute.ServerIPPool, expect: []string{"list", "get:2", "attach:" + dispatchAddresses["b"], "raw"}},
		{name: "pool-allocate", extra: []compute.AutomaticFloatingIPOption{compute.WithFloatingIPPool(resource.Name("public")), compute.WithAutomaticEnsureOptions(network.WithEnsureReuse(false))}, mode: compute.ServerIPPool, expect: []string{"allocate", "get:29", "get:29", "attach:" + dispatchAddresses["pool"], "raw"}},
		{name: "automatic", mode: compute.ServerIPAutomatic, expect: []string{"pools", "list", "get:2", "attach:" + dispatchAddresses["b"], "raw"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newNovaIPFixture(t, "")
			server := automaticServer(t, autoFixed)
			server.Status = "BUILD"
			result, err := f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, append(novaIPOptions(tc.extra...), compute.WithServerIPWait(true))...)
			if err != nil || result == nil || result.Mode != tc.mode || result.NovaAssignment == nil || !result.NovaAssignment.ActionAccepted || result.Assignment != nil || !result.Observed || strings.Join(f.trace(), ",") != strings.Join(tc.expect, ",") {
				t.Fatal(result, err, f.trace())
			}
		})
	}
}

// AddIPsToServer with an address selector runs Python's add_ip_list branch,
// so it shares the _attach_ip_to_server steps of positional AddIPList: the
// supplied model's only floating address short-circuits, an associated IP on
// a model without floating addresses refreshes the server once, and a
// Neutron 404 from the association PUT is recorded before the next address.
func TestServerIPAddressSelectorFollowsSourceAttachSteps(t *testing.T) {
	f := newSourceIPFixture(t)
	f.ports = []string{sourcePortPrivate}
	f.notFound["put:b"] = true
	server := automaticServer(t, sourceAddresses(dispatchAddresses["a"]))
	options := sourceIPOptions(compute.WithFloatingIPAddresses(dispatchAddresses["a"], dispatchAddresses["b"], dispatchAddresses["c"]), compute.WithFloatingIPPool(resource.Ref{}))
	result, err := f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, options...)
	trace := f.trace()
	if err != nil || result == nil || result.Mode != compute.ServerIPExplicit || len(result.Attempts) != 3 || trace != "list:a,list:b,ports,port:port-p,get:b,put:b,list:c,ports,port:port-p,get:c,put:c" {
		t.Fatal(result, err, trace)
	}
	first, second, third := result.Attempts[0], result.Attempts[1], result.Attempts[2]
	var native gophercloud.ErrUnexpectedResponseCode
	if !first.ShortCircuited || !first.Completed || second.Completed || second.Error != nil || !errors.As(second.AttachmentNotFound, &native) || native.Actual != 404 || !third.Completed || third.ShortCircuited {
		t.Fatal(result.Attempts)
	}
	t.Run("refresh", func(t *testing.T) {
		f := newSourceIPFixture(t)
		f.ports = []string{sourcePortPrivate}
		f.bound["a"] = [2]string{"port-p", "10.0.0.10"}
		f.raw = func(int32) string { return sourceAddresses(dispatchAddresses["a"]) }
		result, err := f.service.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, sourceIPOptions(compute.WithFloatingIPAddresses(dispatchAddresses["a"]))...)
		if err != nil || f.trace() != "list:a,raw" || !result.Attempts[0].ShortCircuited || result.Server.Name != "raw-1" {
			t.Fatal(result, err, f.trace())
		}
	})
}
