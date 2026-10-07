package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/compute"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type connectionAvailableState struct {
	locators, events      []string
	deadlines             []time.Time
	pool                  string
	novaFree, neutronFree bool
	neutronCode           int
	neutronBody           string
	neutronPort           string // Optional fixture seam for fresh server allocation.
	neutronMarker         string // Only the pagination cases permit this query.
	catalogError          error
	afterPost             func(string) error
}

func connectionAvailableFixture(t *testing.T, source compute.FloatingIPSource) (*testcloud.Cloud, *sdk.Connection, *connectionAvailableState) {
	t.Helper()
	cloud := testcloud.New(t)
	state := &connectionAvailableState{pool: "public", novaFree: true, neutronFree: true, neutronCode: 200}
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		state.locators = append(state.locators, opts.Type)
		if opts.Type == "network" {
			if state.catalogError != nil {
				return "", state.catalogError
			}
			return cloud.Server.URL + "/network/", nil
		}
		if opts.Type != "compute" {
			t.Error("unexpected service", opts.Type)
		}
		return cloud.Server.URL + "/compute/", nil
	}
	cloud.Provider.HTTPClient.Transport = serverWorkflowTransport(func(r *http.Request) (*http.Response, error) {
		state.events = append(state.events, r.Method+" "+r.URL.Path)
		deadline, _ := r.Context().Deadline()
		state.deadlines = append(state.deadlines, deadline)
		if strings.HasPrefix(r.URL.Path, "/compute/") && r.Header.Get("X-OpenStack-Nova-API-Version") != "2.35" {
			t.Error(r.Header)
		}
		code, body := 200, ""
		switch r.Method + " " + r.URL.Path {
		case "GET /network/v2.0/networks":
			body = `{"networks":[{"id":"external","name":"public","router:external":true,"subnets":["sub"]}]}`
		case "GET /network/v2.0/subnets":
			body = `{"subnets":[{"id":"sub","network_id":"external","ip_version":4}]}`
		case "GET /network/v2.0/floatingips":
			if r.URL.RawQuery != "" && (state.neutronMarker == "" || r.URL.Query().Get("marker") != state.neutronMarker || len(r.URL.Query()) != 1) {
				t.Error("pushed down availability filters", r.URL)
			}
			code, body = state.neutronCode, state.neutronBody
			if body == "" {
				body = `{"floatingips":[]}`
				if state.neutronFree {
					body = `{"floatingips":[{"id":"neutron","floating_network_id":"external","project_id":"owner","port_id":null,"floating_ip_address":"198.51.100.20","status":"ERROR"}]}`
				}
			}
		case "GET /network/v2.0/ports":
			if state.neutronPort == "" || r.URL.Query().Get("device_id") != "server" || len(r.URL.Query()) != 1 {
				t.Error("unexpected availability server port lookup", r.URL)
			}
			body = fmt.Sprintf(`{"ports":[{"id":%q,"device_id":"server","network_id":"private","fixed_ips":[{"ip_address":"10.0.0.8"}]}]}`, state.neutronPort)
		case "POST /network/v2.0/floatingips":
			var request map[string]map[string]any
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			fields := request["floatingip"]
			wantFields := 1
			if state.neutronPort != "" {
				wantFields = 3
				if fields["port_id"] != state.neutronPort || fields["fixed_ip_address"] != "10.0.0.8" {
					t.Error("fresh availability lost selected port/fixed address", fields)
				}
			}
			if len(request) != 1 || len(fields) != wantFields || fields["floating_network_id"] != "external" {
				t.Error("fresh availability must not send project or reuse filters", request)
			}
			code, body = 201, `{"floatingip":{"id":"allocated-neutron","floating_network_id":"external","project_id":"auth","port_id":null,"floating_ip_address":"198.51.100.21","status":"DOWN"}}`
		case "GET /compute/os-floating-ip-pools":
			body = `{"floating_ip_pools":[{"name":"public"},{"name":"second"}]}`
		case "GET /compute/os-floating-ips":
			if r.URL.RawQuery != "" {
				t.Error(r.URL)
			}
			body = `{"floating_ips":[]}`
			if state.novaFree {
				body = fmt.Sprintf(`{"floating_ips":[{"id":9007199254740993,"ip":"198.51.100.10","pool":%q,"instance_id":null}]}`, state.pool)
			}
		case "POST /compute/os-floating-ips":
			var request map[string]string
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if len(request) != 1 || request["pool"] != state.pool {
				t.Error(request)
			}
			body = fmt.Sprintf(`{"floating_ip":{"id":29,"ip":"198.51.100.11","pool":%q,"instance_id":null}}`, state.pool)
		case "GET /compute/os-floating-ips/29":
			body = fmt.Sprintf(`{"floating_ip":{"id":29,"ip":"198.51.100.11","pool":%q,"instance_id":null}}`, state.pool)
		default:
			t.Error("availability must not attach or observe servers", r.Method, r.URL)
			return nil, errors.New("unexpected request")
		}
		var reader io.ReadCloser = io.NopCloser(strings.NewReader(body))
		if r.Method == "POST" && state.afterPost != nil {
			path := r.URL.Path
			reader = readyConnectionCloseBody{ReadCloser: reader, close: func() error { return state.afterPost(path) }}
		}
		return &http.Response{StatusCode: code, Body: reader, Header: http.Header{"X-Available-Proof": {r.URL.Path}}, Request: r}, nil
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.Compute, "2.35"), sdk.WithServerAddressPolicy(compute.WithFloatingIPSource(source)))
	if err != nil {
		t.Fatal(err)
	}
	return cloud, conn, state
}

func connectionAvailableRequest() compute.AvailableFloatingIPRequest {
	return compute.AvailableFloatingIPRequest{Networks: []resource.Ref{resource.Name("public")}}
}

func TestConnectionAvailableIPNovaAndNoneReuseWithoutAttachment(t *testing.T) {
	for _, source := range []compute.FloatingIPSource{compute.FloatingIPNova, compute.FloatingIPNone} {
		_, conn, state := connectionAvailableFixture(t, source)
		input := connectionAvailableRequest()
		input.Server = resource.ID("ignored/invalid/server")
		result, err := conn.AvailableFloatingIP(context.Background(), input)
		if err != nil || result == nil || result.Backend != compute.FloatingIPNova || result.ID != "9007199254740993" || result.Address != "198.51.100.10" || !result.Reused || result.Allocated || result.Neutron != nil || result.Nova.FloatingIP.InstanceID != nil || result.Nova.FloatingIP.StatusCode != 200 || !reflect.DeepEqual(state.locators, []string{"compute"}) || !reflect.DeepEqual(state.events, []string{"GET /compute/os-floating-ips"}) {
			t.Fatal(result, err, state)
		}
	}
}

func TestConnectionAvailableIPNovaAllocationOnlyAndDefaultPool(t *testing.T) {
	for _, defaultPool := range []bool{false, true} {
		_, conn, state := connectionAvailableFixture(t, compute.FloatingIPNova)
		state.novaFree = false
		input := connectionAvailableRequest()
		if defaultPool {
			input.Networks = nil
		}
		result, err := conn.AvailableFloatingIP(context.Background(), input)
		want := []string{"GET /compute/os-floating-ips", "POST /compute/os-floating-ips", "GET /compute/os-floating-ips/29"}
		if defaultPool {
			want = append([]string{"GET /compute/os-floating-ip-pools"}, want...)
		}
		if err != nil || result == nil || !result.Allocated || result.Reused || result.ID != "29" || result.Nova.AllocationResponse.StatusCode != 200 || result.Nova.FloatingIP.InstanceID != nil || !reflect.DeepEqual(state.events, want) {
			t.Fatal(result, err, state)
		}
	}
}

func TestConnectionAvailableIPNeutronFreeUsesNoCompute(t *testing.T) {
	_, conn, state := connectionAvailableFixture(t, compute.FloatingIPNeutron)
	input := connectionAvailableRequest()
	input.Server = resource.Name("not-looked-up")
	result, err := conn.AvailableFloatingIP(context.Background(), input, compute.WithAvailableIPNetworkOptions(network.WithAvailableProject("owner")))
	if err != nil || result == nil || result.Backend != compute.FloatingIPNeutron || !result.Reused || result.Nova != nil || result.ID != "neutron" || result.Neutron.FloatingIP.Status != "ERROR" || !reflect.DeepEqual(state.locators, []string{"network"}) || len(state.events) != 3 {
		t.Fatal(result, err, state)
	}
}

func TestConnectionAvailableIPNotFoundFallbackAndOtherFailuresStayDistinct(t *testing.T) {
	for _, scenario := range []string{"semantic", "list404", "catalog absent", "list403", "list204", "malformed", "mixed catalog", "accepted404retry"} {
		t.Run(scenario, func(t *testing.T) {
			cloud, conn, state := connectionAvailableFixture(t, compute.FloatingIPNeutron)
			input := connectionAvailableRequest()
			cause := errors.New("catalog failed")
			var retries int
			switch scenario {
			case "semantic":
				input.Networks = []resource.Ref{resource.Name("legacy")}
				state.pool = "legacy"
			case "list404":
				state.neutronCode = 404
				state.neutronBody = `{"error":"no floating IP extension"}`
			case "catalog absent":
				state.catalogError = fmt.Errorf("catalog: %w", gophercloud.ErrEndpointNotFound{})
			case "list403":
				state.neutronCode = 403
			case "list204":
				state.neutronCode = 204
			case "malformed":
				state.neutronBody = `{"floatingips":`
			case "mixed catalog":
				state.catalogError = errors.Join(gophercloud.ErrEndpointNotFound{}, cause)
			case "accepted404retry":
				state.neutronCode = 404
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, _ error, _ uint) error {
					retries++
					options.OkCodes = append(options.OkCodes, 404)
					return nil
				}
			}
			result, err := conn.AvailableFloatingIP(context.Background(), input, compute.WithAvailableIPNetworkOptions(network.WithAvailableProject("owner")))
			if scenario == "list404" {
				want := []string{"GET /network/v2.0/networks", "GET /network/v2.0/subnets", "GET /network/v2.0/floatingips", "GET /compute/os-floating-ips", "POST /network/v2.0/floatingips"}
				if err != nil || result == nil || result.Backend != compute.FloatingIPNeutron || !result.Allocated || result.Reused || result.Nova != nil || result.FallbackError != nil || result.Inventory == nil || result.Inventory.Backend != compute.FloatingIPNova || result.Inventory.FallbackError == nil || result.Creation == nil || !result.Creation.Allocated || !reflect.DeepEqual(state.events, want) || !reflect.DeepEqual(state.locators, []string{"network", "compute"}) {
					t.Fatal("inner List404 must return to Neutron allocation", result, err, state)
				}
				return
			}
			fallback := scenario == "semantic" || scenario == "catalog absent"
			if fallback {
				if err != nil || result == nil || result.Backend != compute.FloatingIPNova || !result.Reused || result.Neutron != nil || result.Nova == nil || (scenario != "catalog absent" && result.FallbackError == nil) || !reflect.DeepEqual(state.locators, []string{"network", "compute"}) {
					t.Fatal(result, err, state)
				}
			} else {
				if err == nil || result == nil || result.Nova != nil || !reflect.DeepEqual(state.locators, []string{"network"}) {
					t.Fatal(result, err, state)
				}
				if scenario == "mixed catalog" && !errors.Is(err, cause) {
					t.Fatal(err)
				}
				if scenario == "accepted404retry" {
					var terminal interface{ TerminalSDKFailure() bool }
					var native gophercloud.ErrUnexpectedResponseCode
					if retries != 1 || len(state.events) != 4 || !errors.As(err, &terminal) || !terminal.TerminalSDKFailure() || !errors.As(err, &native) || native.Actual != 404 {
						t.Fatal(err, retries, native, state)
					}
				}
			}
		})
	}
}

func TestConnectionAvailableIPAcceptedFailuresKeepBackendAndAllocation(t *testing.T) {
	for _, source := range []compute.FloatingIPSource{compute.FloatingIPNova, compute.FloatingIPNeutron} {
		for _, failure := range []string{"close", "source", "cancel"} {
			t.Run(string(source)+failure, func(t *testing.T) {
				_, conn, state := connectionAvailableFixture(t, source)
				state.novaFree, state.neutronFree = false, false
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("accepted close failed")
				state.afterPost = func(string) error {
					switch failure {
					case "close":
						return cause
					case "source":
						if source == compute.FloatingIPNova {
							cached, _ := conn.Compute(context.Background())
							cached.API = nil
						} else {
							cached, _ := conn.Network(context.Background())
							cached.API = nil
						}
					case "cancel":
						cancel(cause)
					}
					return nil
				}
				result, err := conn.AvailableFloatingIP(ctx, connectionAvailableRequest(), compute.WithAvailableIPNetworkOptions(network.WithAvailableProject("owner")))
				var proof *resource.ResponseError
				if err == nil || result == nil || !result.Allocated || result.Reused || result.Backend != source || !errors.As(err, &proof) || proof.Header.Get("X-Available-Proof") == "" || len(proof.Body) == 0 {
					t.Fatal(result, err, proof, state)
				}
				if failure != "source" && !errors.Is(err, cause) {
					t.Fatal(err)
				}
				if failure == "source" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				if source == compute.FloatingIPNova {
					if result.ID != "29" || result.Nova.FloatingIP == nil || result.Nova.AllocationResponse.StatusCode != 200 || len(state.events) != 2 {
						t.Fatal(result, state)
					}
				} else if result.ID != "allocated-neutron" || result.Neutron.AllocationResponse.StatusCode != 201 || !reflect.DeepEqual(state.locators, []string{"network"}) || len(state.events) != 4 {
					t.Fatal(result, state)
				}
			})
		}
	}
}

func beforeAvailableOption[T any](before func(), next func(*T) error) func(*T) error {
	return func(o *T) error { before(); return next(o) }
}

func TestAvailableIPServiceOptionsCannotReplaceCapturedSource(t *testing.T) {
	for _, scenario := range []string{"nil servers", "replace servers", "nil API", "endpoint", "nested nil servers"} {
		t.Run(scenario, func(t *testing.T) {
			_, conn, state := connectionAvailableFixture(t, compute.FloatingIPNova)
			service, err := conn.Compute(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			mutate := func() {
				switch scenario {
				case "nil servers", "nested nil servers":
					service.Servers = nil
				case "replace servers":
					service.Servers = &compute.Servers{}
				case "nil API":
					service.API = nil
				case "endpoint":
					service.RawClient().Endpoint += "changed/"
				}
			}
			option := beforeAvailableOption(mutate, compute.WithAvailableIPSource(compute.FloatingIPNova))
			if scenario == "nested nil servers" {
				option = compute.WithAvailableIPNetworkOptions(beforeAvailableOption(mutate, network.WithAvailableProject("owner")))
			}
			result, err := service.AvailableFloatingIP(context.Background(), connectionAvailableRequest(), option)
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(state.events) != 0 || !reflect.DeepEqual(state.locators, []string{"compute"}) {
				t.Fatal(result, err, state)
			}
		})
	}
}

func TestConnectionAvailableIPBudgetAndPreparedOptionsOwnInputs(t *testing.T) {
	for _, scenario := range []string{"default", "override", "parent", "unlimited"} {
		_, conn, state := connectionAvailableFixture(t, compute.FloatingIPNova)
		state.novaFree = false
		ctx := context.Background()
		var parent time.Time
		if scenario == "parent" || scenario == "unlimited" {
			duration := 3 * time.Second
			if scenario == "unlimited" {
				duration = 3 * time.Minute
			}
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, duration)
			defer cancel()
			parent, _ = ctx.Deadline()
		}
		input := connectionAvailableRequest()
		var outer, inner int
		innerOption := beforeAvailableOption(func() { inner++ }, network.WithAvailableProject("owner"))
		outerOption := beforeAvailableOption(func() { outer++; input.Networks[0] = resource.Name("changed") }, compute.WithAvailableIPNetworkOptions(innerOption))
		opts := []compute.AvailableFloatingIPOption{outerOption}
		if scenario != "default" {
			opts = append(opts, compute.WithAvailableIPTimeout(17*time.Second))
		}
		if scenario == "unlimited" {
			opts = append(opts, compute.WithUnlimitedAvailableIPTimeout())
		}
		start := time.Now()
		result, err := conn.AvailableFloatingIP(ctx, input, opts...)
		if err != nil || result == nil || result.Nova == nil || result.Nova.FloatingIP.Pool != "public" || outer != 1 || inner != 1 || len(state.deadlines) != 3 {
			t.Fatal(result, err, outer, inner, state)
		}
		first := state.deadlines[0]
		for _, deadline := range state.deadlines {
			if !deadline.Equal(first) {
				t.Fatal("budget restarted", state.deadlines)
			}
		}
		switch scenario {
		case "default":
			if !first.IsZero() {
				t.Fatal("getter added SDK deadline", first)
			}
		case "override":
			if first.Sub(start) < 16*time.Second || first.Sub(start) > 18*time.Second {
				t.Fatal(first.Sub(start))
			}
		default:
			if !first.Equal(parent) {
				t.Fatal(first, parent)
			}
		}
	}
}

func TestConnectionAvailableIPVersionMultiplePoolsAndContextPreflight(t *testing.T) {
	for _, scenario := range []string{"version", "multiple pools", "nil context", "canceled"} {
		_, conn, state := connectionAvailableFixture(t, compute.FloatingIPNova)
		input := connectionAvailableRequest()
		ctx := context.Background()
		if scenario == "version" {
			cached, err := conn.Compute(ctx)
			if err != nil {
				t.Fatal(err)
			}
			cached.RawClient().Microversion = "2.36"
		}
		if scenario == "multiple pools" {
			input.Networks = append(input.Networks, resource.Name("second"))
		}
		if scenario == "nil context" {
			ctx = nil
		}
		if scenario == "canceled" {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			cancel()
		}
		result, err := conn.AvailableFloatingIP(ctx, input)
		if err == nil || len(state.events) != 0 {
			t.Fatal(result, err, state)
		}
		if scenario == "version" && !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(err)
		}
		if scenario == "canceled" && !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		if scenario != "version" && len(state.locators) != 0 {
			t.Fatal(state)
		}
	}
}
