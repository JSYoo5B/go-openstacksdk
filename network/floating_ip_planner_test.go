package network_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestFloatingIPPlannerSharesOwnedClassificationAndSelectionSnapshot(t *testing.T) {
	cloud := testcloud.New(t)
	var reads atomic.Int32
	cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
		n := reads.Add(1)
		testcloud.JSON(w, 200, fmt.Sprintf(`{"networks":[{"id":"ext-%d","name":"public","router:external":true},{"id":"private-%d","name":"private"}]}`, n, n))
	})
	cloud.Mux.HandleFunc("GET /v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"ports":[{"id":"a","device_id":"server","network_id":"private-1","fixed_ips":[{"ip_address":"10.0.0.1"}]},{"id":"b","device_id":"server","network_id":"private-2","fixed_ips":[{"ip_address":"10.0.0.2"}]}]}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	policy := preparedRoles(t, network.WithConfiguredNetworks(network.ConfiguredNetwork{Name: "private", NATDestination: true}))
	service := network.NewWithDependencies(cloud.Client("network", "/v2.0"), network.Dependencies{NetworkRoles: policy})
	planner, err := service.FloatingIPs.NewPlanner(context.Background())
	if err != nil || reads.Load() != 0 {
		t.Fatal(err, reads.Load())
	}
	roles, err := planner.NetworkRoles(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	roles.NATDestination.ID = "caller-change"
	roles.ExternalIPv4Floating[0].ID = "caller-change"
	service.Roles.Reset()
	plan, err := planner.PrepareEnsure(context.Background(), network.EnsureFloatingIPRequest{Server: resource.ID("server")})
	if err != nil || plan.Selection().NetworkID != "ext-1" || plan.Selection().PortID != "a" || reads.Load() != 1 {
		t.Fatal(plan.Selection(), err, reads.Load())
	}
	next, err := service.FloatingIPs.NewPlanner(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	plan, err = next.PrepareEnsure(context.Background(), network.EnsureFloatingIPRequest{Server: resource.ID("server")})
	if err != nil || plan.Selection().NetworkID != "ext-2" || plan.Selection().PortID != "b" || reads.Load() != 2 {
		t.Fatal(plan.Selection(), err, reads.Load())
	}
}

func TestFloatingIPPlannerOuterGuardStopsColdRoleRetry(t *testing.T) {
	cloud := testcloud.New(t)
	var attempts atomic.Int32
	cloud.Mux.HandleFunc("GET /v2.0/networks", func(w http.ResponseWriter, r *http.Request) { attempts.Add(1); http.Error(w, "retry", 503) })
	changed := false
	client := cloud.Client("network", "/v2.0")
	client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		changed = true
		return nil
	}
	ctx := rest.WithOperationGuard(context.Background(), func(context.Context) error {
		if changed {
			return fmt.Errorf("%w: outer compute source changed", resource.ErrInvalidOption)
		}
		return nil
	})
	planner, err := network.New(client).FloatingIPs.NewPlanner(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = planner.NetworkRoles(ctx)
	var original gophercloud.ErrUnexpectedResponseCode
	if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &original) || original.Actual != 503 || attempts.Load() != 1 {
		t.Fatal(err, original, attempts.Load())
	}
}

func TestFloatingIPPlannerPreflightPreservesNilAndCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	ips := network.New(cloud.Client("network", "/v2.0")).FloatingIPs
	if planner, err := ips.NewPlanner(nil); planner != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(planner, err)
	}
	planner, err := ips.NewPlanner(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("planning canceled")
	cancel(cause)
	if _, err := planner.PrepareEnsure(ctx, network.EnsureFloatingIPRequest{Server: resource.ID("server")}); !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal(err)
	}
	if next, err := ips.NewPlanner(ctx); next != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal(next, err)
	}
}
