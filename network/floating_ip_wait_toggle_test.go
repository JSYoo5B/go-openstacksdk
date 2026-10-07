package network_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/network"
	"gophercloudsdk/resource"
)

func TestEnsureNoWaitPreservesPreparedPolicyForLaterActiveOverride(t *testing.T) {
	cloud := testcloud.New(t)
	ensurePortFixture(t, cloud)
	var gets, posts atomic.Int32
	cloud.Mux.HandleFunc("POST /v2.0/floatingips", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		respondEnsuredFloatingIP(w, 201, "port", "10.0.0.10", "DOWN")
	})
	cloud.Mux.HandleFunc("GET /v2.0/floatingips/fip", func(w http.ResponseWriter, r *http.Request) {
		status := "DOWN"
		if gets.Add(1) > 1 {
			status = "ACTIVE"
		}
		respondEnsuredFloatingIP(w, 200, "port", "10.0.0.10", status)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	progress := 0
	policy, err := network.PrepareEnsureFloatingIPOptions(context.Background(), network.WithEnsureReuse(false), network.WithEnsureProject("owner"), network.WithEnsureWait(resource.WithTimeout(time.Second), resource.WithPollInterval(time.Millisecond), resource.WithProgressCallback(func(int) { progress++ })), network.WithEnsureNoWait())
	if err != nil {
		t.Fatal(err)
	}
	service := network.New(cloud.Client("network", "/v2.0"))
	for _, wait := range []bool{false, true, false} {
		opts := []network.EnsureFloatingIPOption{network.WithEnsureFloatingIPPolicy(policy)}
		if wait {
			opts = append(opts, network.WithEnsureActive())
		}
		result, err := service.FloatingIPs.Ensure(context.Background(), ensureFloatingRequest(), opts...)
		want := "DOWN"
		if wait {
			want = "ACTIVE"
		}
		if err != nil || result == nil || !result.Allocated || result.FloatingIP.Status != want {
			t.Fatal(result, err)
		}
	}
	if posts.Load() != 3 || gets.Load() != 2 || progress != 1 {
		t.Fatal(posts.Load(), gets.Load(), progress)
	}
}

func TestEnsureNoWaitDoesNotHideInvalidWaitPolicy(t *testing.T) {
	for _, options := range [][]resource.WaitOption{{nil}, {resource.WithTimeout(-time.Second)}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			t.Error("invalid wait policy made HTTP request")
			http.Error(w, "unexpected", 500)
		})
		result, err := network.New(cloud.Client("network", "/v2.0")).FloatingIPs.Ensure(context.Background(), ensureFloatingRequest(), network.WithEnsureWait(options...), network.WithEnsureNoWait())
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(result, err)
		}
	}
}
