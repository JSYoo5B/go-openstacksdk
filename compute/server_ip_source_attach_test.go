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

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const (
	sourcePortPrivate = `{"id":"port-p","device_id":"server","network_id":"private","fixed_ips":[{"ip_address":"10.0.0.10"}]}`
	sourcePortNAT     = `{"id":"port-n","device_id":"server","network_id":"nat-net","fixed_ips":[{"ip_address":"2001:db8::5"},{"ip_address":"10.1.0.20"}]}`
)

// sourceIPFixture serves one Neutron and Compute endpoint for the standalone
// address lane. bound maps an IP key to its "port fixed" association.
type sourceIPFixture struct {
	cloud    *testcloud.Cloud
	service  *compute.Service
	network  *network.Service
	mu       sync.Mutex
	events   []string
	bound    map[string][2]string
	puts     []map[string]any
	ports    []string
	raw      func(n int32) string
	notFound map[string]bool
	rawCalls atomic.Int32
}

func (f *sourceIPFixture) event(value string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, value)
}

func (f *sourceIPFixture) trace() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return strings.Join(f.events, ",")
}

func (f *sourceIPFixture) row(key string) string {
	f.mu.Lock()
	association, ok := f.bound[key]
	f.mu.Unlock()
	port, fixed := "null", "null"
	if ok {
		port, fixed = fmt.Sprintf("%q", association[0]), fmt.Sprintf("%q", association[1])
	}
	return fmt.Sprintf(`{"id":"ip-%s","project_id":"foreign","floating_network_id":"external","floating_ip_address":%q,"port_id":%s,"fixed_ip_address":%s,"status":"DOWN","revision_number":0}`, key, dispatchAddresses[key], port, fixed)
}

func sourceAddresses(floating ...string) string {
	rows := []string{`{"version":4,"addr":"10.0.0.10","OS-EXT-IPS:type":"fixed"}`}
	for _, address := range floating {
		rows = append(rows, fmt.Sprintf(`{"version":4,"addr":%q,"OS-EXT-IPS:type":"floating"}`, address))
	}
	return `{"private":[` + strings.Join(rows, ",") + `]}`
}

func newSourceIPFixture(t *testing.T) *sourceIPFixture {
	t.Helper()
	f := &sourceIPFixture{cloud: testcloud.New(t), bound: map[string][2]string{}, notFound: map[string]bool{}, ports: []string{sourcePortPrivate, sourcePortNAT}}
	f.raw = func(int32) string { return sourceAddresses() }
	f.network = network.New(f.cloud.Client("network", "/v2.0"))
	f.service = compute.New(f.cloud.Client("compute", "/v2.1"), compute.Dependencies{AddressNetworks: func(context.Context) (*network.Service, error) { return f.network, nil }})
	key := func(address string) string {
		for candidate, value := range dispatchAddresses {
			if value == address {
				return candidate
			}
		}
		return ""
	}
	f.cloud.Mux.HandleFunc("GET /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		k := key(r.URL.Query().Get("floating_ip_address"))
		f.event("list:" + k)
		if k == "" || len(r.URL.Query()) != 1 {
			t.Error("invalid address filter", r.URL)
		}
		if f.notFound["list:"+k] {
			http.Error(w, "lookup missing", 404)
			return
		}
		testcloud.JSON(w, 200, `{"floatingips":[`+f.row(k)+`]}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.0/floatingips/{id}", func(w http.ResponseWriter, r *http.Request) {
		k := strings.TrimPrefix(r.PathValue("id"), "ip-")
		f.event("get:" + k)
		testcloud.JSON(w, 200, `{"floatingip":`+f.row(k)+`}`)
	})
	f.cloud.Mux.HandleFunc("PUT /v2.0/floatingips/{id}", func(w http.ResponseWriter, r *http.Request) {
		k := strings.TrimPrefix(r.PathValue("id"), "ip-")
		f.event("put:" + k)
		var body map[string]map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if f.notFound["put:"+k] {
			http.Error(w, "attach missing", 404)
			return
		}
		f.mu.Lock()
		f.puts = append(f.puts, body["floatingip"])
		f.bound[k] = [2]string{fmt.Sprint(body["floatingip"]["port_id"]), fmt.Sprint(body["floatingip"]["fixed_ip_address"])}
		f.mu.Unlock()
		testcloud.JSON(w, 200, `{"floatingip":`+f.row(k)+`}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		f.event("ports")
		if r.URL.Query().Get("device_id") != "server" {
			t.Error(r.URL)
		}
		if f.notFound["ports"] {
			http.Error(w, "ports missing", 404)
			return
		}
		testcloud.JSON(w, 200, `{"ports":[`+strings.Join(f.ports, ",")+`]}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.0/ports/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		f.event("port:" + id)
		for _, port := range f.ports {
			if strings.Contains(port, `"id":"`+id+`"`) {
				testcloud.JSON(w, 200, `{"port":`+port+`}`)
				return
			}
		}
		http.Error(w, "port missing", 404)
	})
	f.cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		f.event("networks:" + r.URL.RawQuery)
		testcloud.JSON(w, 200, `{"networks":[{"id":"nat-net","name":"nat"}]}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.0/subnets", func(w http.ResponseWriter, r *http.Request) {
		f.event("subnets")
		testcloud.JSON(w, 200, `{"subnets":[]}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.0/networks/{id}", func(w http.ResponseWriter, r *http.Request) {
		f.event("network:" + r.PathValue("id"))
		testcloud.JSON(w, 200, `{"network":{"id":"nat-net","name":"nat"}}`)
	})
	f.cloud.Mux.HandleFunc("GET /v2.1/servers/server", func(w http.ResponseWriter, r *http.Request) {
		n := f.rawCalls.Add(1)
		f.event("raw")
		testcloud.JSON(w, 200, fmt.Sprintf(`{"server":{"id":"server","name":"raw-%d","status":"BUILD","addresses":%s}}`, n, f.raw(n)))
	})
	f.cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected source attach request", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	return f
}

func sourceIPOptions(extra ...compute.AutomaticFloatingIPOption) []compute.ServerIPOption {
	common := []compute.AutomaticFloatingIPOption{compute.WithAutomaticIPPollInterval(time.Millisecond), compute.WithAutomaticAddressOptions(compute.WithAddressReachability(false))}
	return standaloneIPOptions(append(common, extra...)...)
}

// Python's add_ip_list passes fixed_address and nat_destination to each
// _attach_ip_to_server; Go also accepts an explicit port. Repeated input runs
// each item again and the third item refreshes the server because the selected
// IP is already associated while the supplied model has no floating address.
func TestSourceIPListDestinationSelectorsAndRepeatedInput(t *testing.T) {
	cases := []struct {
		name        string
		option      []network.EnsureFloatingIPOption
		singlePort  bool
		port, fixed string
		event       string
	}{
		{name: "fixed", option: []network.EnsureFloatingIPOption{network.WithEnsureFixedAddress("10.1.0.20")}, port: "port-n", fixed: "10.1.0.20"},
		{name: "nat-id", option: []network.EnsureFloatingIPOption{network.WithEnsureNATDestination(resource.ID("nat-net"))}, port: "port-n", fixed: "10.1.0.20"},
		{name: "nat-name", option: []network.EnsureFloatingIPOption{network.WithEnsureNATDestination(resource.Name("nat"))}, port: "port-n", fixed: "10.1.0.20", event: "networks:name=nat"},
		{name: "nat-fixed", option: []network.EnsureFloatingIPOption{network.WithEnsureNATDestination(resource.ID("nat-net")), network.WithEnsureFixedAddress("10.1.0.20")}, port: "port-n", fixed: "10.1.0.20"},
		{name: "port", option: []network.EnsureFloatingIPOption{network.WithEnsurePort(resource.ID("port-p"))}, port: "port-p", fixed: "10.0.0.10", event: "port:port-p"},
		{name: "default-single-port", singlePort: true, port: "port-p", fixed: "10.0.0.10"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newSourceIPFixture(t)
			if tc.singlePort {
				f.ports = []string{sourcePortPrivate}
			}
			f.raw = func(int32) string { return sourceAddresses(dispatchAddresses["a"], dispatchAddresses["b"]) }
			server := automaticServer(t, autoFixed)
			server.Status = "SHUTOFF"
			addresses := []string{dispatchAddresses["a"], dispatchAddresses["b"], dispatchAddresses["a"]}
			result, err := f.service.AddIPList(context.Background(), server, addresses, sourceIPOptions(compute.WithAutomaticEnsureOptions(tc.option...))...)
			trace := f.trace()
			if err != nil || result == nil || len(result.Attempts) != 3 || result.Observed || f.rawCalls.Load() != 1 || result.Server.Name != "raw-1" || len(result.Decision.AttachmentSelections) != 3 {
				t.Fatal(result, err, trace)
			}
			want := map[string]any{"port_id": tc.port, "fixed_ip_address": tc.fixed}
			if len(f.puts) != 2 || !reflect.DeepEqual(f.puts[0], want) || !reflect.DeepEqual(f.puts[1], want) || strings.Count(trace, "put:a") != 1 || strings.Count(trace, "put:b") != 1 {
				t.Fatal(f.puts, trace)
			}
			if tc.event != "" && !strings.Contains(trace, tc.event) {
				t.Fatal(trace, tc.event)
			}
			// The refresh runs after the third lookup and before its port lookup.
			third := trace[strings.LastIndex(trace, "list:a"):]
			if !strings.HasPrefix(third, "list:a,raw,") || strings.Contains(third, "put:") {
				t.Fatal(third)
			}
			for i, attempt := range result.Attempts {
				if !attempt.Completed || attempt.Observed || attempt.ShortCircuited || attempt.AttachmentNotFound != nil || attempt.Error != nil || attempt.RequestedAddress != addresses[i] {
					t.Fatal(i, attempt)
				}
				if selection := result.Decision.AttachmentSelections[i]; selection.PortID != tc.port || selection.FixedIPv4 != tc.fixed {
					t.Fatal(i, selection)
				}
			}
		})
	}
}

// The destination follows Python's _nat_destination_port: the most recent
// created_at port on the NAT network and its first IPv4, a literal fixed
// address match, and errors before any mutation when no port qualifies.
func TestSourceIPListNATDestinationPortSelection(t *testing.T) {
	older := `{"id":"port-old","device_id":"server","network_id":"nat-net","created_at":"2024-01-01T00:00:00Z","fixed_ips":[{"ip_address":"10.1.0.30"}]}`
	newer := `{"id":"port-new","device_id":"server","network_id":"nat-net","created_at":"2025-01-01T00:00:00Z","fixed_ips":[{"ip_address":"2001:db8::9"},{"ip_address":"10.1.0.40"},{"ip_address":"10.1.0.41"}]}`
	cases := []struct {
		name        string
		ports       []string
		option      []network.EnsureFloatingIPOption
		port, fixed string
	}{
		{name: "newest-first-ipv4", ports: []string{sourcePortPrivate, older, newer}, option: []network.EnsureFloatingIPOption{network.WithEnsureNATDestination(resource.ID("nat-net"))}, port: "port-new", fixed: "10.1.0.40"},
		{name: "fixed-ignores-nat-and-age", ports: []string{sourcePortPrivate, older, newer}, option: []network.EnsureFloatingIPOption{network.WithEnsureNATDestination(resource.ID("private")), network.WithEnsureFixedAddress("10.1.0.30")}, port: "port-old", fixed: "10.1.0.30"},
		{name: "multiple-ports-without-nat-role"},
		{name: "fixed-not-on-server", option: []network.EnsureFloatingIPOption{network.WithEnsureFixedAddress("10.7.7.7")}},
		{name: "no-ports", ports: []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newSourceIPFixture(t)
			if tc.ports != nil {
				f.ports = tc.ports
			}
			result, err := f.service.AddIPList(context.Background(), automaticServer(t, autoFixed), []string{dispatchAddresses["a"], dispatchAddresses["b"]}, sourceIPOptions(compute.WithAutomaticEnsureOptions(tc.option...))...)
			trace := f.trace()
			if tc.port == "" {
				if err == nil || result == nil || len(result.Attempts) != 1 || result.Attempts[0].Error == nil || result.Attempts[0].AttachmentNotFound != nil || len(f.puts) != 0 || strings.Contains(trace, "list:b") {
					t.Fatal(result, err, trace)
				}
				return
			}
			want := map[string]any{"port_id": tc.port, "fixed_ip_address": tc.fixed}
			if err != nil || len(f.puts) != 2 || !reflect.DeepEqual(f.puts[0], want) || !reflect.DeepEqual(f.puts[1], want) {
				t.Fatal(result, err, f.puts, trace)
			}
		})
	}
}

// Python's _attach_ip_to_server returns the server without attaching or
// waiting when its floating IPv4 is already the requested address, and refreshes
// the server first when it has no floating address but the IP has a port.
func TestSourceIPListShortCircuitAndConditionalRefresh(t *testing.T) {
	cases := []struct {
		name      string
		supplied  string
		bound     bool
		raw       string
		wait      bool
		short     bool
		rawCalls  int32
		wantTrace string
	}{
		{name: "supplied-async", supplied: sourceAddresses(dispatchAddresses["a"]), short: true, wantTrace: "list:a"},
		{name: "supplied-wait-skips-wait", supplied: sourceAddresses(dispatchAddresses["a"]), wait: true, short: true, wantTrace: "list:a"},
		{name: "refresh-match", supplied: autoFixed, bound: true, raw: sourceAddresses(dispatchAddresses["a"]), short: true, rawCalls: 1, wantTrace: "list:a,raw"},
		{name: "refresh-no-match-attaches", supplied: autoFixed, bound: true, raw: sourceAddresses(dispatchAddresses["c"]), rawCalls: 1, wantTrace: "list:a,raw,ports,port:port-p,get:a,put:a"},
		{name: "unassociated-skips-refresh", supplied: autoFixed, wantTrace: "list:a,ports,port:port-p,get:a,put:a"},
		{name: "other-floating-skips-refresh", supplied: sourceAddresses(dispatchAddresses["c"]), bound: true, wantTrace: "list:a,ports,port:port-p,get:a,put:a"},
		{name: "several-floating-never-probes", supplied: sourceAddresses(dispatchAddresses["a"], dispatchAddresses["c"]), wantTrace: "list:a,ports,port:port-p,get:a,put:a"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newSourceIPFixture(t)
			f.ports = []string{sourcePortPrivate}
			if tc.bound {
				f.bound["a"] = [2]string{"port-x", "10.9.0.9"}
			}
			if tc.raw != "" {
				f.raw = func(int32) string { return tc.raw }
			}
			rawClients := 0
			f.service = compute.New(nil, compute.Dependencies{AddressNetworks: func(context.Context) (*network.Service, error) { return f.network, nil }, AddressCompute: func(context.Context) (*gophercloud.ServiceClient, error) {
				rawClients++
				return f.cloud.Client("compute", "/v2.1"), nil
			}})
			options := append(sourceIPOptions(), compute.WithServerIPWait(tc.wait))
			result, err := f.service.AddIPList(context.Background(), automaticServer(t, tc.supplied), []string{dispatchAddresses["a"]}, options...)
			trace := f.trace()
			if tc.wait && !tc.short {
				t.Fatal("invalid case")
			}
			if err != nil || result == nil || len(result.Attempts) != 1 || trace != tc.wantTrace || f.rawCalls.Load() != tc.rawCalls || result.Observed {
				t.Fatal(result, err, trace)
			}
			attempt := result.Attempts[0]
			if !attempt.Completed || attempt.ShortCircuited != tc.short || attempt.Observed || attempt.Error != nil || attempt.Assignment == nil || attempt.Assignment.FloatingIP.ID != "ip-a" || !attempt.Assignment.Reused || result.Assignment != attempt.Assignment {
				t.Fatal(attempt)
			}
			if tc.rawCalls == 1 && (result.Server.Name != "raw-1" || rawClients != 1) {
				t.Fatal(result.Server, rawClients)
			}
			if tc.rawCalls == 0 && rawClients != 0 && !tc.wait {
				t.Fatal("async lane resolved Compute without a refresh", rawClients)
			}
			if tc.short && len(result.Decision.AttachmentSelections) != 0 {
				t.Fatal(result.Decision.AttachmentSelections)
			}
		})
	}
}

// Python's _neutron_attach_ip_to_server NotFound is logged and the item falls
// through to the optional wait; the address lookup itself is not covered.
func TestSourceIPListNeutronNotFoundFallsThroughAfterSelection(t *testing.T) {
	t.Run("put-async", func(t *testing.T) {
		f := newSourceIPFixture(t)
		f.ports = []string{sourcePortPrivate}
		f.notFound["put:b"] = true
		result, err := f.service.AddIPList(context.Background(), automaticServer(t, autoFixed), []string{dispatchAddresses["a"], dispatchAddresses["b"], dispatchAddresses["c"]}, sourceIPOptions()...)
		trace := f.trace()
		if err != nil || result == nil || len(result.Attempts) != 3 || !strings.Contains(trace, "put:b,list:c") || strings.Count(trace, "put:") != 3 {
			t.Fatal(result, err, trace)
		}
		second := result.Attempts[1]
		var native gophercloud.ErrUnexpectedResponseCode
		if second.Completed || second.Error != nil || !errors.As(second.AttachmentNotFound, &native) || native.Actual != 404 || second.Assignment == nil || second.Assignment.FloatingIP.ID != "ip-b" || second.Assignment.FloatingIP.PortID != "" {
			t.Fatal(second)
		}
		if !result.Attempts[0].Completed || !result.Attempts[2].Completed || result.Attempts[2].AttachmentNotFound != nil {
			t.Fatal(result.Attempts)
		}
	})
	t.Run("ports-async", func(t *testing.T) {
		f := newSourceIPFixture(t)
		f.notFound["ports"] = true
		result, err := f.service.AddIPList(context.Background(), automaticServer(t, autoFixed), []string{dispatchAddresses["a"]}, sourceIPOptions()...)
		attempt := result.Attempts[0]
		if err != nil || f.trace() != "list:a,ports" || attempt.Completed || attempt.AttachmentNotFound == nil || attempt.Assignment == nil || attempt.Assignment.FloatingIP.ID != "ip-a" || len(result.Decision.AttachmentSelections) != 0 {
			t.Fatal(result, err, f.trace())
		}
	})
	t.Run("put-wait-observes", func(t *testing.T) {
		f := newSourceIPFixture(t)
		f.ports = []string{sourcePortPrivate}
		f.notFound["put:a"] = true
		// Another actor attaches the address; the wait still observes it.
		f.raw = func(n int32) string {
			if n < 2 {
				return sourceAddresses()
			}
			return sourceAddresses(dispatchAddresses["a"])
		}
		result, err := f.service.AddIPList(context.Background(), automaticServer(t, autoFixed), []string{dispatchAddresses["a"]}, append(sourceIPOptions(), compute.WithServerIPWait(true))...)
		attempt := result.Attempts[0]
		if err != nil || !result.Observed || attempt.Completed || !attempt.Observed || attempt.AttachmentNotFound == nil || f.rawCalls.Load() != 2 {
			t.Fatal(result, err, f.trace())
		}
	})
	t.Run("put-wait-timeout", func(t *testing.T) {
		f := newSourceIPFixture(t)
		f.ports = []string{sourcePortPrivate}
		f.notFound["put:a"] = true
		options := append(sourceIPOptions(compute.WithAutomaticIPTimeout(100*time.Millisecond)), compute.WithServerIPWait(true))
		result, err := f.service.AddIPList(context.Background(), automaticServer(t, autoFixed), []string{dispatchAddresses["a"], dispatchAddresses["b"]}, options...)
		if !errors.Is(err, context.DeadlineExceeded) || result == nil || len(result.Attempts) != 1 || result.Attempts[0].AttachmentNotFound == nil || result.Attempts[0].Error == nil || result.Observed || strings.Contains(f.trace(), "list:b") {
			t.Fatal(result, err, f.trace())
		}
	})
	t.Run("lookup-not-swallowed", func(t *testing.T) {
		f := newSourceIPFixture(t)
		f.notFound["list:b"] = true
		result, err := f.service.AddIPList(context.Background(), automaticServer(t, autoFixed), []string{dispatchAddresses["a"], dispatchAddresses["b"], dispatchAddresses["c"]}, sourceIPOptions(compute.WithAutomaticEnsureOptions(network.WithEnsurePort(resource.ID("port-p"))))...)
		if err == nil || result == nil || len(result.Attempts) != 2 || result.Attempts[1].Error == nil || result.Attempts[1].AttachmentNotFound != nil || strings.Contains(f.trace(), "list:c") {
			t.Fatal(result, err, f.trace())
		}
	})
}

// The raw poll calls WithAutomaticIPProgress between unconverged responses, and
// the refresh, lookups and attachments of every item share one deadline.
func TestSourceIPListRawPollProgressAndDeadlines(t *testing.T) {
	t.Run("progress", func(t *testing.T) {
		f := newSourceIPFixture(t)
		f.ports = []string{sourcePortPrivate}
		f.raw = func(n int32) string {
			if n < 3 {
				return sourceAddresses()
			}
			return sourceAddresses(dispatchAddresses["a"])
		}
		var seen []string
		progress := compute.WithAutomaticIPProgress(func(server *compute.Server) error {
			seen = append(seen, server.Name)
			server.Name = "mutated"
			return nil
		})
		result, err := f.service.AddIPList(context.Background(), automaticServer(t, autoFixed), []string{dispatchAddresses["a"]}, append(sourceIPOptions(progress), compute.WithServerIPWait(true))...)
		if err != nil || !result.Observed || !reflect.DeepEqual(seen, []string{"raw-1", "raw-2"}) || result.Server.Name != "raw-3" {
			t.Fatal(result, err, seen)
		}
	})
	t.Run("progress-error-stops", func(t *testing.T) {
		f := newSourceIPFixture(t)
		f.ports = []string{sourcePortPrivate}
		stop := errors.New("stop polling")
		progress := compute.WithAutomaticIPProgress(func(*compute.Server) error { return stop })
		result, err := f.service.AddIPList(context.Background(), automaticServer(t, autoFixed), []string{dispatchAddresses["a"], dispatchAddresses["b"]}, append(sourceIPOptions(progress), compute.WithServerIPWait(true))...)
		if !errors.Is(err, stop) || len(result.Attempts) != 1 || result.Attempts[0].Completed || result.Assignment == nil || f.rawCalls.Load() != 1 || strings.Contains(f.trace(), "list:b") {
			t.Fatal(result, err, f.trace())
		}
	})
	for _, mode := range []string{"default", "parent", "unlimited"} {
		t.Run("deadline-"+mode, func(t *testing.T) {
			f := newSourceIPFixture(t)
			f.ports = []string{sourcePortPrivate}
			f.bound["b"] = [2]string{"port-x", "10.9.0.9"}
			f.raw = func(n int32) string { return sourceAddresses(dispatchAddresses["c"]) }
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
			parent := 3 * time.Minute
			extra := []compute.AutomaticFloatingIPOption{compute.WithAutomaticEnsureOptions(network.WithEnsureFixedAddress("10.0.0.10"))}
			switch mode {
			case "parent":
				parent = 30 * time.Second
			case "unlimited":
				extra = append(extra, compute.WithUnlimitedAutomaticIPTimeout())
			}
			ctx, cancel := context.WithTimeout(context.Background(), parent)
			defer cancel()
			parentDeadline, _ := ctx.Deadline()
			start := time.Now()
			// The refresh for b and both attachments share one context.
			result, err := f.service.AddIPList(ctx, automaticServer(t, autoFixed), []string{dispatchAddresses["a"], dispatchAddresses["b"]}, sourceIPOptions(extra...)...)
			if err != nil || result == nil || len(result.Attempts) != 2 || f.rawCalls.Load() != 1 || len(deadlines) < 8 {
				t.Fatal(result, err, f.trace(), len(deadlines))
			}
			for _, deadline := range deadlines {
				if !deadline.Equal(deadlines[0]) {
					t.Fatal("deadlines differ", deadlines)
				}
			}
			switch mode {
			case "default":
				if remaining := deadlines[0].Sub(start); remaining > 61*time.Second || remaining < 55*time.Second {
					t.Fatal(remaining)
				}
			default:
				if !deadlines[0].Equal(parentDeadline) {
					t.Fatal(deadlines[0], parentDeadline)
				}
			}
		})
	}
}

// The legacy Nova lane short-circuits on the same server-model check; Nova
// rows carry no port, so there is no refresh, and the action still sends the
// requested fixed address.
func TestSourceIPListNovaShortCircuitSkipsRecheckAndAction(t *testing.T) {
	for _, wait := range []bool{false, true} {
		t.Run(fmt.Sprint(wait), func(t *testing.T) {
			f := newNovaIPFixture(t, "")
			server := automaticServer(t, sourceAddresses(dispatchAddresses["a"]))
			server.Status = "SHUTOFF"
			result, err := f.service.AddIPList(context.Background(), server, []string{dispatchAddresses["a"], dispatchAddresses["b"]}, append(novaIPOptions(), compute.WithServerIPWait(wait))...)
			expected := []string{"list", "list", "get:2", "attach:" + dispatchAddresses["b"]}
			if wait {
				expected = append(expected, "raw")
			}
			if err != nil || result == nil || len(result.Attempts) != 2 || !reflect.DeepEqual(f.trace(), expected) || result.Observed || len(result.Decision.NovaSelections) != 2 {
				t.Fatal(result, err, f.trace())
			}
			first, second := result.Attempts[0], result.Attempts[1]
			if !first.ShortCircuited || !first.Completed || first.Observed || first.NovaAssignment == nil || first.NovaAssignment.ActionAccepted || first.NovaAssignment.FloatingIP.Address != dispatchAddresses["a"] {
				t.Fatal(first)
			}
			if second.ShortCircuited || !second.Completed || second.Observed != wait || !second.NovaAssignment.ActionAccepted {
				t.Fatal(second)
			}
		})
	}
}
