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

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/compute"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

type connectionNovaIPState struct {
	locators, events, tokens []string
	target                   string
	raw                      int
	afterAction              func(string) error
}

func newConnectionNovaIP(t *testing.T, source compute.FloatingIPSource) (*testcloud.Cloud, *sdk.Connection, *connectionNovaIPState) {
	t.Helper()
	cloud := testcloud.New(t)
	state := &connectionNovaIPState{}
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		state.locators = append(state.locators, opts.Type)
		if opts.Type == "network" {
			return "", gophercloud.ErrEndpointNotFound{}
		}
		if opts.Type != "compute" {
			t.Error("unexpected service", opts.Type)
		}
		return cloud.Server.URL + "/compute/", nil
	}
	cloud.Provider.HTTPClient.Transport = serverWorkflowTransport(func(r *http.Request) (*http.Response, error) {
		state.tokens = append(state.tokens, r.Header.Get("X-Auth-Token"))
		if r.Header.Get("X-OpenStack-Nova-API-Version") != "2.35" {
			t.Error("legacy version changed", r.Header)
		}
		body, proof, code := "", "", 200
		switch {
		case r.Method == "GET" && r.URL.Path == "/compute/os-floating-ips":
			state.events = append(state.events, "list")
			if r.URL.RawQuery != "" {
				t.Error("Nova pushed down query", r.URL)
			}
			body = `{"floating_ips":[{"id":1,"ip":"198.51.100.10","pool":"public","instance_id":null},{"id":"2","ip":"198.51.100.11","pool":"public","instance_id":null}]}`
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/compute/os-floating-ips/"):
			id := strings.TrimPrefix(r.URL.Path, "/compute/os-floating-ips/")
			state.events = append(state.events, "get:"+id)
			address := "198.51.100.10"
			if id == "2" {
				address = "198.51.100.11"
			}
			body = fmt.Sprintf(`{"floating_ip":{"id":%q,"ip":%q,"pool":"public","instance_id":null}}`, id, address)
		case r.Method == "POST" && r.URL.Path == "/compute/servers/server/action":
			var request map[string]map[string]string
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			action := request["addFloatingIp"]
			if len(request) != 1 || len(action) != 2 || action["fixed_address"] != "10.0.0.10" {
				t.Error(request)
			}
			state.target = action["address"]
			state.events = append(state.events, "attach:"+state.target)
			code = 202
			proof = "accepted-" + state.target
		case r.Method == "GET" && r.URL.Path == "/compute/servers/server":
			state.raw++
			state.events = append(state.events, "raw")
			body = fmt.Sprintf(`{"server":{"id":"server","status":"BUILD","addresses":{"private":[{"version":4,"addr":%q,"OS-EXT-IPS:type":"floating"}]}}}`, state.target)
		default:
			t.Error("unexpected Nova facade request", r.Method, r.URL)
			return nil, errors.New("unexpected request")
		}
		var reader io.ReadCloser = io.NopCloser(strings.NewReader(body))
		if code == 202 && state.afterAction != nil {
			address := state.target
			reader = readyConnectionCloseBody{ReadCloser: reader, close: func() error { return state.afterAction(address) }}
		}
		return &http.Response{StatusCode: code, Header: http.Header{"X-Facade-Proof": {proof}}, Body: reader, Request: r}, nil
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.Compute, "2.35"), sdk.WithServerAddressPolicy(compute.WithFloatingIPSource(source), compute.WithAddressReachability(false)))
	if err != nil {
		t.Fatal(err)
	}
	return cloud, conn, state
}

func connectionNovaIPOptions(wait bool) []compute.ServerIPOption {
	return []compute.ServerIPOption{compute.WithServerIPAutomaticOptions(compute.WithAutomaticEnsureOptions(network.WithEnsureFixedAddress("10.0.0.10")), compute.WithAutomaticIPPollInterval(time.Millisecond)), compute.WithServerIPWait(wait)}
}

func TestConnectionNovaIPBackendChoiceAndAsyncRawWait(t *testing.T) {
	for _, source := range []compute.FloatingIPSource{compute.FloatingIPNova, compute.FloatingIPNone, compute.FloatingIPNeutron} {
		for _, wait := range []bool{false, true} {
			_, conn, state := newConnectionNovaIP(t, source)
			server := connectionAddressServer()
			server.Status = "SHUTOFF"
			server.Addresses = nil
			result, err := conn.AddIPList(context.Background(), server, []string{"198.51.100.10", "198.51.100.11"}, connectionNovaIPOptions(wait)...)
			if err != nil || result == nil || result.Decision.Backend != compute.FloatingIPNova || result.Assignment != nil || result.NovaAssignment == nil || len(result.Attempts) != 2 || result.Observed != wait {
				t.Fatal(source, wait, result, err, state.events)
			}
			want := []string{"compute"}
			if source == compute.FloatingIPNeutron {
				want = []string{"network", "compute"}
			}
			if !reflect.DeepEqual(state.locators, want) || state.raw != map[bool]int{false: 0, true: 2}[wait] {
				t.Fatal(source, state.locators, state.raw)
			}
			for _, attempt := range result.Attempts {
				if !attempt.Completed || attempt.Observed != wait || attempt.Assignment != nil || !attempt.NovaAssignment.ActionAccepted || attempt.NovaAssignment.ActionResponse.StatusCode != 202 {
					t.Fatal(attempt)
				}
			}
		}
	}
}

func TestConnectionNovaIPReadsLiveTokenAndEmptyListStaysLazy(t *testing.T) {
	cloud, conn, state := newConnectionNovaIP(t, compute.FloatingIPNova)
	state.afterAction = func(address string) error {
		if address == "198.51.100.10" {
			cloud.Provider.SetToken("next-token")
		}
		return nil
	}
	result, err := conn.AddIPList(context.Background(), connectionAddressServer(), []string{"198.51.100.10", "198.51.100.11"}, connectionNovaIPOptions(false)...)
	if err != nil || result == nil || len(state.tokens) != 6 || !reflect.DeepEqual(state.tokens, []string{"test-token", "test-token", "test-token", "next-token", "next-token", "next-token"}) {
		t.Fatal(result, err, state.tokens)
	}
	for _, source := range []compute.FloatingIPSource{compute.FloatingIPNova, compute.FloatingIPNone} {
		_, conn, state = newConnectionNovaIP(t, source)
		result, err = conn.AddIPList(context.Background(), connectionAddressServer(), nil, append(connectionNovaIPOptions(true), compute.WithServerIPAutomaticOptions(compute.WithFloatingIPPool(resource.ID("ignored/pool"))))...)
		if err != nil || result == nil || result.Decision.Needed || result.Decision.Reason != compute.AutomaticIPEmptyAddressList || len(state.locators) != 0 || len(state.events) != 0 || len(state.tokens) != 0 || result.NovaAssignment != nil {
			t.Fatal(source, result, err, state)
		}
	}
}

func TestConnectionNovaIPAcceptedCachedSourceFailureRetainsHistory(t *testing.T) {
	_, conn, state := newConnectionNovaIP(t, compute.FloatingIPNova)
	cached, err := conn.Compute(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	state.afterAction = func(address string) error {
		if address == "198.51.100.11" {
			cached.API = nil
		}
		return nil
	}
	result, err := conn.AddIPList(context.Background(), connectionAddressServer(), []string{"198.51.100.10", "198.51.100.11", "198.51.100.12"}, connectionNovaIPOptions(true)...)
	var proof *resource.ResponseError
	if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 202 || proof.Header.Get("X-Facade-Proof") != "accepted-198.51.100.11" || len(proof.Body) != 0 || result == nil || len(result.Attempts) != 2 || !result.Attempts[0].Completed || !result.Attempts[0].Observed || result.Attempts[1].Completed || result.Attempts[1].Error == nil || result.Observed || result.NovaAssignment == nil || !result.NovaAssignment.ActionAccepted || result.NovaAssignment.FloatingIP.ID != "2" || result.NovaAssignment.FloatingIP.InstanceID != nil || state.raw != 1 || !reflect.DeepEqual(state.locators, []string{"compute"}) || !reflect.DeepEqual(state.events, []string{"list", "get:1", "attach:198.51.100.10", "raw", "list", "get:2", "attach:198.51.100.11"}) {
		t.Fatal(result, err, proof, state)
	}
}

func TestConnectionNovaIPMixedCatalogFailureCannotSelectBackend(t *testing.T) {
	sentinel := errors.New("catalog source failed")
	for _, scenario := range []struct {
		name       string
		err, cause error
	}{
		{"value", gophercloud.ErrEndpointNotFound{}, nil},
		{"pointer", &gophercloud.ErrEndpointNotFound{}, nil},
		{"single wrapper", fmt.Errorf("catalog: %w", &gophercloud.ErrEndpointNotFound{}), nil},
		{"mixed failure", errors.Join(gophercloud.ErrEndpointNotFound{}, sentinel), sentinel},
		{"wrapped mixed failure", fmt.Errorf("catalog: %w", errors.Join(&gophercloud.ErrEndpointNotFound{}, sentinel)), sentinel},
		{"mixed cancellation", errors.Join(gophercloud.ErrEndpointNotFound{}, context.Canceled), context.Canceled},
	} {
		for _, consumer := range []string{"add", "roles"} {
			t.Run(scenario.name+"/"+consumer, func(t *testing.T) {
				cloud, conn, state := newConnectionNovaIP(t, compute.FloatingIPNeutron)
				locate := cloud.Provider.EndpointLocator
				cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
					if opts.Type == "network" {
						state.locators = append(state.locators, opts.Type)
						return "", scenario.err
					}
					return locate(opts)
				}
				var result *compute.AutomaticServerIPResult
				var err error
				if consumer == "add" {
					result, err = conn.AddIPList(context.Background(), connectionAddressServer(), []string{"198.51.100.10"}, connectionNovaIPOptions(false)...)
				} else {
					_, err = conn.GetNetworkRoles(context.Background())
				}
				if scenario.cause == nil {
					if err != nil || (consumer == "add" && (result == nil || result.NovaAssignment == nil || !result.NovaAssignment.ActionAccepted)) {
						t.Fatal(result, err, state)
					}
					return
				}
				if !errors.Is(err, scenario.cause) || !reflect.DeepEqual(state.locators, []string{"network"}) || len(state.events) != 0 || len(state.tokens) != 0 || (result != nil && (result.NovaAssignment != nil || result.Assignment != nil || len(result.Attempts) != 0 || result.Decision.Backend != compute.FloatingIPNeutron)) {
					t.Fatal(result, err, state)
				}
			})
		}
	}
}
