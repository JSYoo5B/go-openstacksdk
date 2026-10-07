package gophercloudsdk_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"gophercloudsdk/compute"
	"gophercloudsdk/resource"
)

func TestConnectionStandaloneIPHelpersShareLazyAsyncAndRawWait(t *testing.T) {
	for _, list := range []bool{false, true} {
		for _, wait := range []bool{false, true} {
			t.Run(fmt.Sprint(list, wait), func(t *testing.T) {
				_, conn, state, raw := connectionDispatchFixture(t)
				server := connectionAddressServer()
				server.Status = "BUILD"
				server.Addresses = nil
				addresses := []string{"198.51.100.10", "198.51.100.11"}
				options := []compute.ServerIPOption{compute.WithServerIPWait(wait), compute.WithServerIPAutomaticOptions(compute.WithFloatingIPAddresses(addresses...))}
				var result *compute.AutomaticServerIPResult
				var err error
				if list {
					result, err = conn.AddIPList(context.Background(), server, addresses, options...)
				} else {
					result, err = conn.AddIPsToServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, options...)
				}
				wantRaw, wantLocates := int32(0), 0
				if wait {
					wantRaw, wantLocates = 2, 1
				}
				if err != nil || result == nil || result.Observed != wait || len(result.Attempts) != 2 || state.puts != 2 || state.locates != wantLocates || raw() != wantRaw || state.gets["a"] != 1 || state.gets["b"] != 1 || strings.Join(state.lists, ",") != "a,b" {
					t.Fatal(result, err, state, raw())
				}
				for _, attempt := range result.Attempts {
					if !attempt.Completed || attempt.Observed != wait || attempt.Assignment.FloatingIP.Status != "DOWN" {
						t.Fatal(attempt)
					}
				}
			})
		}
	}
}

func TestConnectionStandaloneEmptyIPListAvoidsEveryServiceLookup(t *testing.T) {
	_, conn, state, raw := connectionDispatchFixture(t)
	options := []compute.ServerIPOption{compute.WithServerIPWait(true), compute.WithServerIPAutomaticOptions(compute.WithFloatingIPPool(resource.ID("ignored/pool")), compute.WithFloatingIPAddresses("ignored"))}
	result, err := conn.AddIPList(context.Background(), connectionAddressServer(), nil, options...)
	if err != nil || result == nil || result.Mode != compute.ServerIPExplicit || result.Decision.Reason != compute.AutomaticIPEmptyAddressList || result.Observed || state.locates != 0 || state.puts != 0 || len(state.lists) != 0 || raw() != 0 {
		t.Fatal(result, err, state, raw())
	}
}

func TestConnectionStandaloneIPSourceFailureRetainsFirstObservationAndSecondIP(t *testing.T) {
	_, conn, state, raw := connectionDispatchFixture(t)
	state.mutate = func() error {
		service, err := conn.Compute(context.Background())
		if err == nil {
			service.API = nil
		}
		return err
	}
	options := []compute.ServerIPOption{compute.WithServerIPWait(true)}
	result, err := conn.AddIPList(context.Background(), connectionAddressServer(), []string{"198.51.100.10", "198.51.100.11", "198.51.100.12"}, options...)
	var proof *resource.ResponseError
	if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Facade-Proof") != "accepted-b" || result == nil || result.Observed || result.Server.Name != "observed-a" || len(result.Attempts) != 2 || !result.Attempts[0].Completed || !result.Attempts[0].Observed || result.Attempts[1].Completed || result.Attempts[1].Error == nil || result.Assignment.FloatingIP.ID != "ip-b" || result.Assignment.FloatingIP.PortID != "port" || raw() != 1 || state.puts != 2 || state.gets["b"] != 1 || strings.Join(state.lists, ",") != "a,b" {
		t.Fatal(result, err, proof, state, raw())
	}
}
