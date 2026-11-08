package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	v1 "github.com/JSYoo5B/go-openstacksdk/baremetal/v1"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/allocations"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestBaremetalWaitUnsupportedControlsRejectedBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	ctx := context.Background()
	nodeAPI := nodes.New(r2icClient(cloud, "1.99"))
	allocationAPI := allocations.New(r2icClient(cloud, "1.99"))
	machines := r2icService(cloud)
	callback := resource.WithProgressCallback(func(int) error { return nil })
	for name, call := range map[string]func() error{
		"allocation failure states": func() error {
			_, err := allocationAPI.WaitForActive(ctx, "a", allocations.WithAllocationWait(resource.WithFailureStates("error")))
			return err
		},
		"power progress": func() error {
			_, err := nodeAPI.WaitForPowerState(ctx, "n1", "power on", resource.WithProgress(make(chan int)))
			return err
		},
		"reservation status attribute": func() error {
			_, err := nodeAPI.WaitForReservation(ctx, &nodes.Node{UUID: "n1"}, resource.WithStatusAttribute("reservation"))
			return err
		},
		"provision callback": func() error {
			_, err := nodeAPI.AwaitProvisionStates(ctx, []string{"n1"}, "active", nodes.WithProvisionWait(callback))
			return err
		},
		"set power state failure states": func() error {
			return nodeAPI.SetPowerState(ctx, "n1", "power on", nodes.WithPowerStateWait(resource.WithFailureStates()))
		},
		"machine progress": func() error {
			return machines.ActivateNode(ctx, "n1", v1.WithMachineWaitOptions(callback))
		},
	} {
		if err := call(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("HTTP before validation", calls.Load())
	}
}

func TestBaremetalWaitRepeatedOptionsLaterWins(t *testing.T) {
	ms := time.Millisecond
	growing := resource.ExponentialWait(2*ms, 2, 0)
	t.Run("allocation", func(t *testing.T) {
		cloud := testcloud.New(t)
		var gets strategyGets
		gets.serve(cloud, "GET /v1/allocations/a", 4, `{"uuid":"a","state":"allocating"}`, `{"uuid":"a","state":"active"}`)
		api := allocations.New(r2icClient(cloud, "1.99"))
		if _, err := api.WaitForActive(context.Background(), "a", allocations.WithAllocationWait(resource.WithPollInterval(time.Hour), resource.WithTimeout(time.Millisecond)), allocations.WithIgnoreError(true), allocations.WithAllocationWait(resource.WithWaitStrategy(growing), resource.WithUnlimitedWait())); err != nil {
			t.Fatal(err)
		}
		gets.gaps(t, 2*ms, 4*ms, 8*ms)
	})
	t.Run("provision", func(t *testing.T) {
		cloud := testcloud.New(t)
		var gets strategyGets
		gets.serve(cloud, "GET /v1/nodes/n1", 3, `{"uuid":"n1","provision_state":"deploying"}`, `{"uuid":"n1","provision_state":"active"}`)
		api := nodes.New(r2icClient(cloud, "1.99"))
		if _, err := api.AwaitProvisionState(context.Background(), "n1", "active", nodes.WithProvisionWait(resource.WithWaitStrategy(growing)), nodes.WithProvisionWait(resource.WithPollInterval(3*ms))); err != nil {
			t.Fatal(err)
		}
		gets.gaps(t, 3*ms, 3*ms)
	})
	t.Run("machine", func(t *testing.T) {
		cloud := testcloud.New(t)
		r2icServe(t, cloud, "PUT /v1/nodes/n1/states/provision", 202, "")
		var gets strategyGets
		gets.serve(cloud, "GET /v1/nodes/n1", 4, `{"uuid":"n1","provision_state":"deploying"}`, `{"uuid":"n1","provision_state":"active"}`)
		node, err := r2icService(cloud).NodeSetProvisionState(context.Background(), "n1", "active", v1.WithMachineWait(true), v1.WithMachineWaitOptions(resource.WithPollInterval(time.Hour)), v1.WithMachineWaitOptions(resource.WithWaitStrategy(growing)))
		if err != nil || node.ProvisionState != "active" {
			t.Fatal(node, err)
		}
		gets.gaps(t, 2*ms, 4*ms, 8*ms)
	})
}

func TestBaremetalWaitTimeoutOptionBoundsHelperWaits(t *testing.T) {
	cloud := testcloud.New(t)
	var gets atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/allocations/a", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		testcloud.JSON(w, 200, `{"uuid":"a","state":"allocating"}`)
	})
	cloud.Mux.HandleFunc("GET /v1/nodes/n1", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		testcloud.JSON(w, 200, `{"uuid":"n1","power_state":"power off","provision_state":"deploying"}`)
	})
	ctx := context.Background()
	wait := []resource.WaitOption{resource.WithTimeout(20 * time.Millisecond), resource.WithPollInterval(time.Millisecond)}
	if _, err := allocations.New(r2icClient(cloud, "1.99")).WaitForActive(ctx, "a", allocations.WithAllocationWait(wait...)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("allocation: %v", err)
	}
	api := nodes.New(r2icClient(cloud, "1.99"))
	if _, err := api.WaitForPowerState(ctx, "n1", "power on", wait...); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("power: %v", err)
	}
	result, err := api.AwaitProvisionStates(ctx, []string{"n1"}, "active", nodes.WithContinueOnFailure(), nodes.WithProvisionWait(wait...))
	if err != nil || len(result.Timeout) != 1 || result.Timeout[0].UUID != "n1" {
		t.Fatalf("provision states: %+v %v", result, err)
	}
	if gets.Load() < 3 {
		t.Fatal("expected reads before each timeout", gets.Load())
	}
}
