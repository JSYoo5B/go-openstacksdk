package gophercloudsdk_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"

	"gophercloudsdk/compute"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

func TestAvailableIPNeutronInnerListFallbackAllocatesOnSelectedNetwork(t *testing.T) {
	cloud, conn, state := connectionAvailableFixture(t, compute.FloatingIPNeutron)
	state.neutronCode = 404
	state.neutronBody = `{"error":"inner Neutron list unavailable"}`
	availableViewReply(t, cloud, func(r *http.Request) (string, bool) {
		return `{"floating_ips":[{"id":7,"ip":"198.51.100.7","pool":"public","instance_id":null,"project_id":"other"}]}`, r.Method == "GET" && r.URL.Path == "/compute/os-floating-ips"
	})
	result, err := conn.AvailableFloatingIP(context.Background(), connectionAvailableRequest(), compute.WithAvailableIPNetworkOptions(network.WithAvailableProject("owner")))
	want := []string{"GET /network/v2.0/networks", "GET /network/v2.0/subnets", "GET /network/v2.0/floatingips", "GET /compute/os-floating-ips", "POST /network/v2.0/floatingips"}
	if err != nil || result == nil || result.Backend != compute.FloatingIPNeutron || !result.Allocated || result.Reused || result.Nova != nil || result.ID != "allocated-neutron" || result.FallbackError != nil || result.Inventory == nil || result.Inventory.Backend != compute.FloatingIPNova || result.Inventory.FallbackError == nil || result.Creation == nil || !result.Creation.Allocated || result.Creation.Selection.NetworkID != "external" || result.Creation.AllocationResponse == nil || result.Creation.AllocationResponse.StatusCode != 201 || !reflect.DeepEqual(state.events, want) || !reflect.DeepEqual(state.locators, []string{"network", "compute"}) {
		t.Fatal(result, err, state)
	}
	if len(result.Inventory.Pages) != 1 || len(result.Inventory.FloatingIPs) != 1 || result.Inventory.Failure == nil || result.Inventory.Failure.StatusCode != 404 || string(result.Inventory.Failure.Envelope) != state.neutronBody {
		t.Fatal("internal listing proof disappeared", result)
	}
	queryRaw(t, result.Creation.Allocation.Wire, "project_id", `"auth"`)
	queryRaw(t, result.Inventory.FloatingIPs[0].Resource, "floating_network_id", `"public"`)
}

func TestAvailableIPNeutronInnerNovaReuseIgnoresInstanceAndServer(t *testing.T) {
	cloud, conn, state := connectionAvailableFixture(t, compute.FloatingIPNeutron)
	state.neutronCode = 404
	availableViewReply(t, cloud, func(r *http.Request) (string, bool) {
		return `{"floating_ips":[{"id":7,"ip":"passive-not-IPv4","pool":"external","project_id":"owner","port_id":null,"instance_id":"still-attached"}]}`, r.Method == "GET" && r.URL.Path == "/compute/os-floating-ips"
	})
	input := connectionAvailableRequest()
	input.Server = resource.Name("must-not-resolve")
	result, err := conn.AvailableFloatingIP(context.Background(), input, compute.WithAvailableIPNetworkOptions(network.WithAvailableProject("owner")))
	want := []string{"GET /network/v2.0/networks", "GET /network/v2.0/subnets", "GET /network/v2.0/floatingips", "GET /compute/os-floating-ips"}
	if err != nil || result == nil || result.Backend != compute.FloatingIPNova || !result.Reused || result.Allocated || result.Neutron != nil || result.Creation != nil || result.FloatingIP == nil || !result.FloatingIP.Normalized || result.FloatingIP.NormalizationSource != compute.FloatingIPNeutron || result.Inventory == nil || result.Inventory.FallbackError == nil || result.FallbackError != nil || !reflect.DeepEqual(state.events, want) || !reflect.DeepEqual(state.locators, []string{"network", "compute"}) {
		t.Fatal(result, err, state)
	}
	queryRaw(t, result.FloatingIP.Resource, "attached", "false")
	queryRaw(t, result.FloatingIP.Resource, "status", `"UNKNOWN"`)
	queryRaw(t, result.FloatingIP.Resource, "floating_ip_address", `"passive-not-IPv4"`)
	queryRaw(t, result.FloatingIP.Wire, "instance_id", `"still-attached"`)
	result.FloatingIP.Wire.Body["instance_id"][0] = '['
	queryRaw(t, result.Inventory.FloatingIPs[0].Wire, "instance_id", `"still-attached"`)
}

func TestAvailableIPNeutronInnerProjectNullAndCanonicalAlias(t *testing.T) {
	for _, test := range []struct {
		name, project    string
		override, reused bool
	}{
		{"unscoped missing project is empty string", "", false, false},
		{"canonical null wins tenant owner", `,"project_id":null,"tenant_id":"owner"`, false, true},
		{"tenant null is unscoped null", `,"tenant_id":null`, false, true},
		{"explicit project rejects canonical null", `,"project_id":null,"tenant_id":"owner"`, true, false},
		{"explicit canonical project wins tenant other", `,"project_id":"owner","tenant_id":"other"`, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud, conn, state := connectionAvailableFixture(t, compute.FloatingIPNeutron)
			state.neutronCode = 404
			body := `{"floating_ips":[{"id":7,"ip":"198.51.100.7","pool":"external","port_id":null,"instance_id":"attached"` + test.project + `}]}`
			availableViewReply(t, cloud, func(r *http.Request) (string, bool) {
				return body, r.Method == "GET" && r.URL.Path == "/compute/os-floating-ips"
			})
			var options []compute.AvailableFloatingIPOption
			if test.override {
				options = append(options, compute.WithAvailableIPNetworkOptions(network.WithAvailableProject("owner")))
			}
			result, err := conn.AvailableFloatingIP(context.Background(), connectionAvailableRequest(), options...)
			want := []string{"GET /network/v2.0/networks", "GET /network/v2.0/subnets", "GET /network/v2.0/floatingips", "GET /compute/os-floating-ips"}
			if !test.reused {
				want = append(want, "POST /network/v2.0/floatingips")
			}
			if err != nil || result == nil || result.Reused != test.reused || result.Allocated == test.reused || result.Inventory == nil || result.Inventory.FallbackError == nil || result.FallbackError != nil || !reflect.DeepEqual(state.events, want) {
				t.Fatal(result, err, state)
			}
			if test.reused {
				if result.Backend != compute.FloatingIPNova || result.Neutron != nil || result.Creation != nil {
					t.Fatal(result)
				}
			} else if result.Backend != compute.FloatingIPNeutron || result.Creation == nil || result.Creation.Selection.NetworkID != "external" {
				t.Fatal(result)
			}
		})
	}
}

func TestAvailableIPNeutronInnerNovaNotFoundStillAllocatesNeutron(t *testing.T) {
	cloud, conn, state := connectionAvailableFixture(t, compute.FloatingIPNeutron)
	state.neutronCode, state.neutronBody = 404, `{"error":"Neutron list missing"}`
	const novaBody = `{"error":"Nova list missing"}`
	availableReply(t, cloud, func(r *http.Request) (int, string, bool) {
		return 404, novaBody, r.Method == "GET" && r.URL.Path == "/compute/os-floating-ips"
	})
	result, err := conn.AvailableFloatingIP(context.Background(), connectionAvailableRequest(), compute.WithAvailableIPNetworkOptions(network.WithAvailableProject("owner")))
	want := []string{"GET /network/v2.0/networks", "GET /network/v2.0/subnets", "GET /network/v2.0/floatingips", "GET /compute/os-floating-ips", "POST /network/v2.0/floatingips"}
	if err != nil || result == nil || !result.Allocated || result.Reused || result.Backend != compute.FloatingIPNeutron || result.FallbackError != nil || result.Inventory == nil || result.Inventory.FallbackError == nil || result.Inventory.SuppressedNotFound == nil || len(result.Inventory.FloatingIPs) != 0 || string(result.Inventory.Value) != "[]" || result.Inventory.Failure == nil || result.Inventory.Failure.Backend != compute.FloatingIPNova || result.Inventory.Failure.StatusCode != 404 || string(result.Inventory.Failure.Envelope) != novaBody || result.Creation == nil || !result.Creation.Allocated || !reflect.DeepEqual(state.events, want) {
		t.Fatal(result, err, state)
	}
}

func TestAvailableIPNeutronInnerStrictMissingAliasIsTerminal(t *testing.T) {
	cloud, conn, state := connectionAvailableFixture(t, compute.FloatingIPNeutron)
	state.neutronCode = 404
	availableViewReply(t, cloud, func(r *http.Request) (string, bool) {
		return `{"floating_ips":[{"id":7,"pool":"external","project_id":"owner","port_id":null,"instance_id":null}]}`, r.Method == "GET" && r.URL.Path == "/compute/os-floating-ips"
	})
	result, err := conn.AvailableFloatingIP(context.Background(), connectionAvailableRequest(), compute.WithAvailableIPStrict(true), compute.WithAvailableIPNetworkOptions(network.WithAvailableProject("owner")))
	want := []string{"GET /network/v2.0/networks", "GET /network/v2.0/subnets", "GET /network/v2.0/floatingips", "GET /compute/os-floating-ips"}
	if err == nil || !strings.Contains(err.Error(), "port_id") || result == nil || result.Reused || result.Allocated || result.FloatingIP != nil || result.Creation != nil || result.FallbackError != nil || result.Inventory == nil || result.Inventory.FallbackError == nil || len(result.Inventory.FloatingIPs) != 1 || !reflect.DeepEqual(state.events, want) {
		t.Fatal(result, err, state)
	}
	if _, present := result.Inventory.FloatingIPs[0].Resource.Body["port_id"]; present {
		t.Fatal("strict alias unexpectedly restored", result.Inventory)
	}
	queryRaw(t, result.Inventory.FloatingIPs[0].Resource, "port", "null")
}

func TestAvailableIPNeutronOuterFallbackRelistsOriginalPoolUnderOneBudget(t *testing.T) {
	for _, reuse := range []bool{false, true} {
		t.Run(fmt.Sprint("outer reuse=", reuse), func(t *testing.T) {
			cloud, conn, state := connectionAvailableFixture(t, compute.FloatingIPNeutron)
			state.neutronCode = 404
			novaReads, outerOptions, innerOptions := 0, 0, 0
			availableReply(t, cloud, func(r *http.Request) (int, string, bool) {
				if r.Method == "POST" && r.URL.Path == "/network/v2.0/floatingips" {
					return 404, `{"error":"Neutron create unavailable"}`, true
				}
				if r.Method == "GET" && r.URL.Path == "/compute/os-floating-ips" {
					novaReads++
					body := `{"floating_ips":[]}`
					if novaReads == 1 {
						body = `{"floating_ips":[{"id":7,"pool":"other","instance_id":null}]}`
					} else if reuse {
						body = `{"floating_ips":[{"id":8,"ip":"198.51.100.8","pool":"public","instance_id":null}]}`
					}
					return 200, body, true
				}
				return 0, "", false
			})
			outer := beforeAvailableOption(func() { outerOptions++ }, compute.WithAvailableIPTimeout(time.Second))
			inner := beforeAvailableOption(func() { innerOptions++ }, network.WithAvailableProject("owner"))
			result, err := conn.AvailableFloatingIP(context.Background(), connectionAvailableRequest(), outer, compute.WithAvailableIPNetworkOptions(inner))
			want := []string{"GET /network/v2.0/networks", "GET /network/v2.0/subnets", "GET /network/v2.0/floatingips", "GET /compute/os-floating-ips", "POST /network/v2.0/floatingips", "GET /compute/os-floating-ips"}
			if !reuse {
				want = append(want, "POST /compute/os-floating-ips", "GET /compute/os-floating-ips/29")
			}
			if err != nil || result == nil || result.Backend != compute.FloatingIPNova || result.Reused != reuse || result.Allocated == reuse || result.FallbackError == nil || result.Inventory == nil || result.Inventory.FallbackError == nil || result.Nova == nil || result.Nova.Inventory == nil || novaReads != 2 || outerOptions != 1 || innerOptions != 1 || !reflect.DeepEqual(state.events, want) {
				t.Fatal(result, err, state, novaReads, outerOptions, innerOptions)
			}
			if result.Inventory == result.Nova.Inventory {
				t.Fatal("outer pool availability reused inner inventory", result)
			}
			var inner404, outer404 gophercloud.ErrUnexpectedResponseCode
			if !errors.As(result.Inventory.FallbackError, &inner404) || inner404.Actual != 404 || !errors.As(result.FallbackError, &outer404) || outer404.Actual != 404 || string(outer404.Body) != `{"error":"Neutron create unavailable"}` || reflect.DeepEqual(inner404.Body, outer404.Body) {
				t.Fatal("inner and outer NotFound proofs were collapsed", result)
			}
			for _, deadline := range state.deadlines {
				if deadline.IsZero() || !deadline.Equal(state.deadlines[0]) {
					t.Fatal("nested or fallback budget restarted", state.deadlines)
				}
			}
			if !reuse && (result.Nova.Creation == nil || result.Nova.Creation.AllocationResponse == nil || result.Nova.Creation.AllocationResponse.StatusCode != 200) {
				t.Fatal(result)
			}
		})
	}
}

func TestAvailableIPNeutronListFailuresNeverAllocateMatchingPrefix(t *testing.T) {
	for _, mode := range []string{"early descriptor", "late page", "late Nova normalization", "inner close", "inner source", "inner cancel"} {
		t.Run(mode, func(t *testing.T) {
			cloud, conn, state := connectionAvailableFixture(t, compute.FloatingIPNeutron)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("inner availability inventory failure")
			state.neutronCode = 404
			const match = `{"id":"match","floating_network_id":"external","project_id":"owner","port_id":null}`
			if mode == "early descriptor" || mode == "late page" {
				state.neutronCode = 200
				state.neutronMarker = "late"
			}
			availableReply(t, cloud, func(r *http.Request) (int, string, bool) {
				if r.Method == "GET" && r.URL.Path == "/network/v2.0/floatingips" && state.neutronCode == 200 {
					if r.URL.Query().Get("marker") != "" {
						return 403, `{"error":"later page denied"}`, true
					}
					row := match
					if mode == "early descriptor" {
						row = `{"id":"bad","floating_network_id":"external","project_id":"owner","port_id":null,"revision_number":"²"}`
					}
					return 200, `{"floatingips":[` + row + `],"floatingips_links":[{"rel":"next","href":"?marker=late"}]}`, true
				}
				if r.Method == "GET" && r.URL.Path == "/compute/os-floating-ips" {
					body := `{"floating_ips":[{"id":7,"pool":"external","project_id":"owner","port_id":null,"instance_id":null}]}`
					if mode == "late Nova normalization" {
						body = `{"floating_ips":[{"id":7,"pool":"external","project_id":"owner","port_id":null,"instance_id":null},{"pool":"other","instance_id":"attached"}]}`
					}
					return 200, body, true
				}
				return 0, "", false
			})
			if strings.HasPrefix(mode, "inner ") {
				next := cloud.Provider.HTTPClient.Transport
				cloud.Provider.HTTPClient.Transport = serverWorkflowTransport(func(r *http.Request) (*http.Response, error) {
					response, err := next.RoundTrip(r)
					if err == nil && response != nil && r.URL.Path == "/compute/os-floating-ips" {
						response.Body = readyConnectionCloseBody{ReadCloser: response.Body, close: func() error {
							switch mode {
							case "inner source":
								cached, loadErr := conn.Network(context.Background())
								if loadErr != nil {
									return loadErr
								}
								cached.API = nil
							case "inner cancel":
								cancel(cause)
							default:
								return cause
							}
							return nil
						}}
					}
					return response, err
				})
			}
			result, err := conn.AvailableFloatingIP(ctx, connectionAvailableRequest(), compute.WithAvailableIPTimeout(time.Second), compute.WithAvailableIPNetworkOptions(network.WithAvailableProject("owner")))
			want := []string{"GET /network/v2.0/networks", "GET /network/v2.0/subnets", "GET /network/v2.0/floatingips"}
			if mode != "early descriptor" {
				path := "GET /compute/os-floating-ips"
				if mode == "late page" {
					path = "GET /network/v2.0/floatingips"
				}
				want = append(want, path)
			}
			if err == nil || result == nil || result.Reused || result.Allocated || result.FloatingIP != nil || result.Creation != nil || result.FallbackError != nil || result.Inventory == nil || !reflect.DeepEqual(state.events, want) {
				t.Fatal("partial inventory caused allocation or reuse", result, err, state)
			}
			if result.Inventory.Failure == nil {
				t.Fatal("inventory failure lost actual response proof", result, err)
			}
			if mode == "late page" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != `{"error":"later page denied"}` || len(result.Inventory.Pages) != 1 || result.Inventory.Failure.StatusCode != 403 {
					t.Fatal(native, result, err)
				}
			} else {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 {
					t.Fatal(proof, err)
				}
			}
			if mode == "early descriptor" && result.Inventory.FallbackError != nil {
				t.Fatal(result)
			}
			if mode == "inner close" && !errors.Is(err, cause) || mode == "inner source" && !errors.Is(err, resource.ErrInvalidOption) || mode == "inner cancel" && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
				t.Fatal(err)
			}
		})
	}
}

func TestAvailableIPNeutronFreshPortSeedsAndAcceptedFailures(t *testing.T) {
	for _, mode := range []string{"missing", "null", "wrong", "close", "source", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			cloud, conn, state := connectionAvailableFixture(t, compute.FloatingIPNeutron)
			state.neutronFree, state.neutronPort = false, "selected-port"
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("accepted attached availability failure")
			const responsePrefix = `{"floatingip":{"id":"attached","floating_network_id":"external","fixed_ip_address":"10.0.0.8","status":"DOWN"`
			port := `,"port_id":"selected-port"`
			if mode == "missing" {
				port = ""
			}
			if mode == "null" {
				port = `,"port_id":null`
			}
			if mode == "wrong" {
				port = `,"port_id":"wrong"`
			}
			body := responsePrefix + port + `}}`
			availableReply(t, cloud, func(r *http.Request) (int, string, bool) {
				return 203, body, r.Method == "POST" && r.URL.Path == "/network/v2.0/floatingips"
			})
			if mode == "close" || mode == "source" || mode == "cancel" {
				next := cloud.Provider.HTTPClient.Transport
				cloud.Provider.HTTPClient.Transport = serverWorkflowTransport(func(r *http.Request) (*http.Response, error) {
					response, err := next.RoundTrip(r)
					if err == nil && response != nil && r.Method == "POST" && r.URL.Path == "/network/v2.0/floatingips" {
						response.Body = readyConnectionCloseBody{ReadCloser: response.Body, close: func() error {
							switch mode {
							case "source":
								cached, loadErr := conn.Network(context.Background())
								if loadErr != nil {
									return loadErr
								}
								cached.API = nil
							case "cancel":
								cancel(cause)
							default:
								return cause
							}
							return nil
						}}
					}
					return response, err
				})
			}
			input := connectionAvailableRequest()
			input.Server = resource.ID("server")
			result, err := conn.AvailableFloatingIP(ctx, input, compute.WithAvailableIPTimeout(time.Second), compute.WithAvailableIPNetworkOptions(network.WithAvailableProject("owner")))
			want := []string{"GET /network/v2.0/networks", "GET /network/v2.0/subnets", "GET /network/v2.0/floatingips", "GET /network/v2.0/ports", "POST /network/v2.0/floatingips"}
			if result == nil || (err != nil) != (mode != "missing") || !result.Allocated || result.Reused || result.Backend != compute.FloatingIPNeutron || result.FallbackError != nil || result.Nova != nil || result.Creation == nil || !result.Creation.Allocated || result.Creation.Waited || len(result.Creation.Observations) != 0 || result.Creation.Cleanup != nil || result.Creation.Selection.ServerID != "server" || result.Creation.Selection.PortID != "selected-port" || result.Creation.Selection.FixedIPv4 != "10.0.0.8" || result.Creation.AllocationResponse == nil || result.Creation.AllocationResponse.StatusCode != 203 || string(result.Creation.AllocationResponse.Envelope) != body || !reflect.DeepEqual(state.events, want) || !reflect.DeepEqual(state.locators, []string{"network"}) {
				t.Fatal(result, err, state)
			}
			queryRaw(t, result.Creation.Allocation.Wire, "id", `"attached"`)
			if mode == "missing" {
				queryRaw(t, result.FloatingIP.Resource, "port_id", `"selected-port"`)
				if _, present := result.FloatingIP.Wire.Body["port_id"]; present {
					t.Fatal("request seed leaked into Wire", result)
				}
			}
			if mode == "null" || mode == "wrong" {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body {
					t.Fatal(err, proof)
				}
				if mode == "null" {
					queryRaw(t, result.Creation.Allocation.Resource, "port_id", "null")
				} else {
					queryRaw(t, result.Creation.Allocation.Resource, "port_id", `"wrong"`)
				}
			}
			if mode == "close" && !errors.Is(err, cause) || mode == "source" && !errors.Is(err, resource.ErrInvalidOption) || mode == "cancel" && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
				t.Fatal(err)
			}
			for _, deadline := range state.deadlines {
				if deadline.IsZero() || !deadline.Equal(state.deadlines[0]) {
					t.Fatal("fresh selection budget restarted", state.deadlines)
				}
			}
		})
	}
}

func TestAvailableIPNeutronFreshWithoutPortKeepsPassiveResponse(t *testing.T) {
	cloud, conn, state := connectionAvailableFixture(t, compute.FloatingIPNeutron)
	state.neutronFree = false
	const body = `{"floatingip":{"id":null,"floating_ip_address":null,"floating_network_id":"other","fixed_ip_address":"2001:db8::1","port_id":"passive-port","status":"ERROR"}}`
	availableReply(t, cloud, func(r *http.Request) (int, string, bool) {
		return 203, body, r.Method == "POST" && r.URL.Path == "/network/v2.0/floatingips"
	})
	result, err := conn.AvailableFloatingIP(context.Background(), connectionAvailableRequest())
	want := []string{"GET /network/v2.0/networks", "GET /network/v2.0/subnets", "GET /network/v2.0/floatingips", "POST /network/v2.0/floatingips"}
	if err != nil || result == nil || !result.Allocated || result.Reused || result.FloatingIP == nil || result.Creation == nil || result.Creation.Selection.PortID != "" || result.Creation.Selection.NetworkID != "external" || result.Creation.AllocationResponse == nil || result.Creation.AllocationResponse.StatusCode != 203 || string(result.Creation.AllocationResponse.Envelope) != body || !reflect.DeepEqual(state.events, want) {
		t.Fatal(result, err, state)
	}
	for _, row := range []*resource.RawResource{result.FloatingIP.Resource, result.FloatingIP.Wire} {
		queryRaw(t, row, "id", "null")
		queryRaw(t, row, "floating_ip_address", "null")
		queryRaw(t, row, "floating_network_id", `"other"`)
		queryRaw(t, row, "fixed_ip_address", `"2001:db8::1"`)
		queryRaw(t, row, "port_id", `"passive-port"`)
	}
}
