package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func pyLBLoadBalancerJSON(prov, oper string) string {
	return fmt.Sprintf(`{"loadbalancer":{"id":"lb-1","name":"web","provisioning_status":%q,"operating_status":%q}}`, prov, oper)
}

// Python wait_for_load_balancer resolves the load balancer once, then polls
// provisioning_status until the target (default ACTIVE) or an ERROR state.
func TestPyLBWaitForLoadBalancerPollsProvisioningStatusAndResolvesNameOnce(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, lists atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/lbaas/loadbalancers", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if r.URL.Query().Get("name") != "web" {
			t.Errorf("name filter=%q", r.URL.Query().Get("name"))
		}
		testcloud.JSON(w, 200, `{"loadbalancers":[{"id":"lb-1","name":"web","provisioning_status":"PENDING_CREATE"}]}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/loadbalancers/lb-1", func(w http.ResponseWriter, r *http.Request) {
		if gets.Add(1) < 3 {
			testcloud.JSON(w, 200, pyLBLoadBalancerJSON("PENDING_UPDATE", "OFFLINE"))
			return
		}
		testcloud.JSON(w, 200, pyLBLoadBalancerJSON("ACTIVE", "ONLINE"))
	})
	api := newOctaviaLoadBalancers(t, cloud)
	value, err := api.WaitFor(context.Background(), resource.Name("web"), "active", resource.WithPollInterval(time.Millisecond))
	if err != nil || value.ProvisioningStatus != "ACTIVE" || lists.Load() != 1 || gets.Load() != 3 {
		t.Fatalf("value=%+v err=%v lists=%d gets=%d", value, err, lists.Load(), gets.Load())
	}
}

func TestPyLBWaitForLoadBalancerErrorFailureTimeoutAndOverrides(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	status := atomic.Value{}
	status.Store("ERROR")
	cloud.Mux.HandleFunc("GET /v2/lbaas/loadbalancers/lb-1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, pyLBLoadBalancerJSON(status.Load().(string), "ONLINE"))
	})
	api := newOctaviaLoadBalancers(t, cloud)
	ctx := context.Background()
	var failed *resource.FailedStateError
	if _, err := api.WaitFor(ctx, resource.ID("lb-1"), "ACTIVE", resource.WithPollInterval(time.Millisecond)); !errors.As(err, &failed) || failed.Status != "ERROR" {
		t.Fatalf("default failure: %v", err)
	}
	// Python failures=[] disables failure states; the wait then runs until the bound.
	_, err := api.WaitFor(ctx, resource.ID("lb-1"), "ACTIVE", resource.WithFailureStates(), resource.WithPollInterval(time.Millisecond), resource.WithTimeout(30*time.Millisecond))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("empty failure list: %v", err)
	}
	// A custom failure list replaces the default.
	status.Store("PENDING_DELETE")
	if _, err := api.WaitFor(ctx, resource.ID("lb-1"), "ACTIVE", resource.WithFailureStates("pending_delete"), resource.WithPollInterval(time.Millisecond)); !errors.As(err, &failed) || failed.Status != "PENDING_DELETE" {
		t.Fatalf("custom failure: %v", err)
	}
	// The target is matched before failure states and case-insensitively.
	status.Store("Error")
	if value, err := api.WaitFor(ctx, resource.ID("lb-1"), "ERROR", resource.WithFailureStates("ERROR")); err != nil || value.ProvisioningStatus != "Error" {
		t.Fatalf("target before failure: %v %+v", err, value)
	}
}

// Python wait_for_status(attribute=...) waits on any named status attribute and
// accepts wait=None (forever). Go selects an exported field by wire name.
func TestPyLBGenericWaitSelectsAttributeAndAcceptsUnlimitedWait(t *testing.T) {
	cloud := testcloud.New(t)
	var gets atomic.Int32
	var progress []int
	cloud.Mux.HandleFunc("GET /v2/lbaas/loadbalancers/lb-1", func(w http.ResponseWriter, r *http.Request) {
		if gets.Add(1) < 2 {
			testcloud.JSON(w, 200, pyLBLoadBalancerJSON("ACTIVE", "OFFLINE"))
			return
		}
		testcloud.JSON(w, 200, pyLBLoadBalancerJSON("ACTIVE", "ONLINE"))
	})
	api := newOctaviaLoadBalancers(t, cloud)
	value, err := api.WaitFor(context.Background(), resource.ID("lb-1"), "ONLINE",
		resource.WithStatusAttribute("operating_status"), resource.WithUnlimitedWait(), resource.WithPollInterval(time.Millisecond),
		resource.WithProgressCallback(func(n int) { progress = append(progress, n) }))
	if err != nil || value.OperatingStatus != "ONLINE" || gets.Load() != 2 || len(progress) != 1 {
		t.Fatalf("value=%+v err=%v gets=%d progress=%v", value, err, gets.Load(), progress)
	}
	if _, err := api.WaitFor(context.Background(), resource.ID("lb-1"), "ONLINE", resource.WithStatusAttribute("no_such_field")); err == nil {
		t.Fatal("unknown status attribute must fail")
	}
}

// Python wait_for_delete polls until the object is gone (404).
func TestPyLBWaitForDeletionCompletesOnNotFoundAndKeepsOtherErrors(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var gets atomic.Int32
	mode := atomic.Value{}
	mode.Store("gone")
	cloud.Mux.HandleFunc("GET /v2/lbaas/loadbalancers/lb-1", func(w http.ResponseWriter, r *http.Request) {
		n := gets.Add(1)
		switch {
		case mode.Load() == "forbidden":
			testcloud.JSON(w, 403, `{}`)
		case n < 3:
			testcloud.JSON(w, 200, pyLBLoadBalancerJSON("PENDING_DELETE", "OFFLINE"))
		default:
			testcloud.JSON(w, 404, `{}`)
		}
	})
	api := newOctaviaLoadBalancers(t, cloud)
	if err := api.WaitForDeletion(context.Background(), resource.ID("lb-1"), resource.WithPollInterval(time.Millisecond)); err != nil || gets.Load() != 3 {
		t.Fatalf("err=%v gets=%d", err, gets.Load())
	}
	mode.Store("forbidden")
	err := api.WaitForDeletion(context.Background(), resource.ID("lb-1"), resource.WithPollInterval(time.Millisecond))
	if err == nil || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("403 must remain an error: %v", err)
	}
}
