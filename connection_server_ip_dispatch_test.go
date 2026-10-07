package gophercloudsdk_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/compute"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

type connectionDispatchState struct {
	bound   map[string]bool
	target  string
	lists   []string
	gets    map[string]int
	puts    int
	locates int
	mutate  func() error
}

func connectionDispatchIP(key string, bound bool, status string) string {
	port, fixed := "", ""
	if bound {
		port, fixed = "port", "10.0.0.10"
	}
	address := "198.51.100.10"
	if key == "b" {
		address = "198.51.100.11"
	}
	return fmt.Sprintf(`{"id":"ip-%s","project_id":"foreign","floating_network_id":"existing-network","floating_ip_address":%q,"port_id":%q,"fixed_ip_address":%q,"status":%q,"revision_number":0}`, key, address, port, fixed, status)
}

func connectionDispatchFixture(t *testing.T) (*testcloud.Cloud, *sdk.Connection, *connectionDispatchState, func() int32) {
	t.Helper()
	cloud, conn, _, raw := connectionReadyFixture(t, "")
	state := &connectionDispatchState{bound: make(map[string]bool), gets: make(map[string]int)}
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		state.locates++
		if opts.Type != "compute" {
			t.Error("unexpected locator", opts.Type)
		}
		return cloud.Server.URL + "/compute/", nil
	}
	base := cloud.Provider.HTTPClient.Transport
	cloud.Provider.HTTPClient.Transport = serverWorkflowTransport(func(r *http.Request) (*http.Response, error) {
		var body string
		proof := ""
		switch {
		case r.URL.Path == "/network/v2.0/floatingips":
			key := "a"
			if r.URL.Query().Get("floating_ip_address") == "198.51.100.11" {
				key = "b"
			}
			if len(r.URL.Query()) != 1 || r.URL.Query().Get("floating_ip_address") == "" {
				t.Error(r.URL)
			}
			state.lists = append(state.lists, key)
			body = `{"floatingips":[` + connectionDispatchIP(key, state.bound[key], "DOWN") + `]}`
		case strings.HasPrefix(r.URL.Path, "/network/v2.0/floatingips/ip-"):
			key := strings.TrimPrefix(r.URL.Path, "/network/v2.0/floatingips/ip-")
			status := "DOWN"
			if r.Method == "PUT" {
				state.puts++
				state.bound[key], state.target = true, key
				if r.Header.Get("If-Match") != "revision_number=0" {
					t.Error(r.Header)
				}
				proof = "accepted-" + key
			} else {
				state.gets[key]++
				if state.bound[key] {
					status = "ACTIVE"
				}
			}
			body = `{"floatingip":` + connectionDispatchIP(key, state.bound[key], status) + `}`
		case r.URL.Path == "/compute/servers/server" && state.target != "":
			raw.Add(1)
			address := "198.51.100.10"
			if state.target == "b" {
				address = "198.51.100.11"
			}
			body = fmt.Sprintf(`{"server":{"id":"server","name":"observed-%s","status":"ACTIVE","addresses":{"private":[{"version":4,"addr":%q,"OS-EXT-IPS:type":"floating"}]}}}`, state.target, address)
		default:
			return base.RoundTrip(r)
		}
		var reader io.ReadCloser = io.NopCloser(strings.NewReader(body))
		if r.Method == "PUT" && state.target == "b" && state.mutate != nil {
			reader = readyConnectionCloseBody{ReadCloser: reader, close: state.mutate}
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"X-Facade-Proof": {proof}}, Body: reader, Request: r}, nil
	})
	return cloud, conn, state, raw.Load
}

func TestConnectionExplicitIPDispatchSharesLazyAsyncAndOrderedWaitPolicies(t *testing.T) {
	for _, wait := range []bool{false, true} {
		t.Run(fmt.Sprint(wait), func(t *testing.T) {
			_, conn, state, raw := connectionDispatchFixture(t)
			input := compute.AutomaticFloatingIPRequest{Server: connectionAddressServer(), Network: resource.ID("ignored/network")}
			input.Server.Status = "ACTIVE"
			opts := append(connectionReadyOptions(), compute.WithServerReadyAutomaticIPOptions(compute.WithFloatingIPAddresses("198.51.100.10", "198.51.100.11"), compute.WithAutomaticIPEnabled(false)))
			call := conn.GetActiveServer
			if wait {
				call = conn.WaitForServer
			}
			result, err := call(context.Background(), input, opts...)
			if err != nil || result == nil || result.Mode != compute.ServerIPExplicit || len(result.Attempts) != 2 || result.Observed != wait || result.Assignment.FloatingIP.ID != "ip-b" || state.puts != 2 || strings.Join(state.lists, ",") != "a,b" {
				t.Fatal(result, err, state)
			}
			wantGets, wantRaw, wantLocates := 1, int32(0), 0
			if wait {
				wantGets, wantRaw, wantLocates = 2, 3, 1
			}
			if state.gets["a"] != wantGets || state.gets["b"] != wantGets || raw() != wantRaw || state.locates != wantLocates {
				t.Fatal(state, raw())
			}
			for _, attempt := range result.Attempts {
				if !attempt.Completed || attempt.Observed != wait || attempt.Error != nil || attempt.Assignment.Allocated {
					t.Fatal(attempt)
				}
			}
		})
	}
}

func TestConnectionOrderedIPSourceFailureKeepsFirstObservationAndAcceptedSecondIP(t *testing.T) {
	_, conn, state, raw := connectionDispatchFixture(t)
	state.mutate = func() error {
		service, err := conn.Compute(context.Background())
		if err == nil {
			service.API = nil
		}
		return err
	}
	input := compute.AutomaticFloatingIPRequest{Server: connectionAddressServer()}
	input.Server.Status = "ACTIVE"
	opts := append(connectionReadyOptions(), compute.WithServerReadyAutomaticIPOptions(compute.WithFloatingIPAddresses("198.51.100.10", "198.51.100.11", "198.51.100.12")))
	result, err := conn.WaitForServer(context.Background(), input, opts...)
	var proof *resource.ResponseError
	if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Facade-Proof") != "accepted-b" || result == nil || result.Observed || result.Server.Name != "observed-a" || len(result.Attempts) != 2 || !result.Attempts[0].Completed || !result.Attempts[0].Observed || result.Attempts[1].Completed || result.Attempts[1].Observed || result.Attempts[1].Error == nil || result.Assignment.FloatingIP.ID != "ip-b" || result.Assignment.FloatingIP.PortID != "port" || state.puts != 2 || raw() != 2 || strings.Join(state.lists, ",") != "a,b" || state.gets["b"] != 1 {
		t.Fatal(result, err, proof, state, raw())
	}
}
