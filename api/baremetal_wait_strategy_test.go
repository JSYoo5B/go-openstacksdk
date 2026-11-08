package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	v1 "github.com/JSYoo5B/go-openstacksdk/baremetal/v1"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/allocations"
	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// strategyGets records GET times on one route and answers done after n reads.
type strategyGets struct {
	mu   sync.Mutex
	seen []time.Time
}

func (g *strategyGets) serve(cloud *testcloud.Cloud, pattern string, n int, pending, done string) {
	cloud.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		g.mu.Lock()
		g.seen = append(g.seen, time.Now())
		count := len(g.seen)
		g.mu.Unlock()
		if count < n {
			testcloud.JSON(w, 200, pending)
			return
		}
		testcloud.JSON(w, 200, done)
	})
}

func (g *strategyGets) gaps(t *testing.T, want ...time.Duration) {
	t.Helper()
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.seen) != len(want)+1 {
		t.Fatalf("%d reads, want %d", len(g.seen), len(want)+1)
	}
	for i, d := range want {
		if gap := g.seen[i+1].Sub(g.seen[i]); gap < d {
			t.Fatalf("pause %d = %v, want at least %v", i+1, gap, d)
		}
	}
}

func TestBaremetalWaitStrategyRejectedBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	ctx := context.Background()
	invalid := resource.LinearWait(time.Millisecond, time.Millisecond, time.Microsecond)
	nodeAPI := nodes.New(r2icClient(cloud, "1.99"))
	allocationAPI := allocations.New(r2icClient(cloud, "1.99"))
	machines := r2icService(cloud)
	strategy := v1.WithMachineWaitStrategy(invalid)
	for name, call := range map[string]func() error{
		"allocation wait": func() error {
			_, err := allocationAPI.WaitForActive(ctx, "a", allocations.WithWaitStrategy(invalid))
			return err
		},
		"power wait": func() error {
			_, err := nodeAPI.WaitForPowerState(ctx, "n1", "power on", nodes.WithWaitStrategy(invalid))
			return err
		},
		"reservation wait without reservation": func() error {
			_, err := nodeAPI.WaitForReservation(ctx, &nodes.Node{UUID: "n1"}, nodes.WithWaitStrategy(resource.WaitStrategy{}))
			return err
		},
		"provision wait": func() error {
			_, err := nodeAPI.AwaitProvisionState(ctx, "n1", "active", nodes.WithProvisionWaitStrategy(invalid))
			return err
		},
		"provision waits": func() error {
			_, err := nodeAPI.AwaitProvisionStates(ctx, []string{"n1"}, "active", nodes.WithProvisionWaitStrategy(invalid))
			return err
		},
		"set provision state": func() error {
			_, err := nodeAPI.SetProvisionState(ctx, "n1", nodes.ProvisionStateRequest{Target: "active"}, nodes.WithProvisionStateWait(nodes.WithProvisionWaitStrategy(invalid)))
			return err
		},
		"set power state": func() error {
			return nodeAPI.SetPowerState(ctx, "n1", "power on", nodes.WithPowerStateWait(nodes.WithWaitStrategy(invalid)))
		},
		"machine provision": func() error {
			_, err := machines.NodeSetProvisionState(ctx, "n1", "active", strategy)
			return err
		},
		"activate":   func() error { return machines.ActivateNode(ctx, "n1", strategy) },
		"deactivate": func() error { return machines.DeactivateNode(ctx, "n1", strategy) },
		"inspect": func() error {
			_, err := machines.InspectMachine(ctx, "n1", strategy)
			return err
		},
		"register": func() error {
			_, err := machines.RegisterMachine(ctx, nodes.CreateOpts{Driver: "fake"}, nil, strategy)
			return err
		},
		"unregister": func() error { return machines.UnregisterMachine(ctx, nil, "n1", strategy) },
	} {
		if err := call(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("HTTP before validation", calls.Load())
	}
}

func TestBaremetalWaitStrategyPacesPollsAndLaterOptionWins(t *testing.T) {
	ms := time.Millisecond
	growing := resource.ExponentialWait(2*ms, 2, 0)
	t.Run("allocation", func(t *testing.T) {
		cloud := testcloud.New(t)
		var gets strategyGets
		gets.serve(cloud, "GET /v1/allocations/a", 4, `{"uuid":"a","state":"allocating"}`, `{"uuid":"a","state":"active"}`)
		api := allocations.New(r2icClient(cloud, "1.99"))
		if _, err := api.WaitForActive(context.Background(), "a", allocations.WithWaitStrategy(growing)); err != nil {
			t.Fatal(err)
		}
		gets.gaps(t, 2*ms, 4*ms, 8*ms)
	})
	t.Run("power", func(t *testing.T) {
		cloud := testcloud.New(t)
		var gets strategyGets
		gets.serve(cloud, "GET /v1/nodes/n1", 4, `{"uuid":"n1","power_state":"power off"}`, `{"uuid":"n1","power_state":"power on"}`)
		api := nodes.New(r2icClient(cloud, "1.99"))
		// A later strategy replaces an earlier invalid one.
		if _, err := api.WaitForPowerState(context.Background(), "n1", "power on", nodes.WithWaitStrategy(resource.WaitStrategy{}), nodes.WithWaitStrategy(resource.LinearWait(2*ms, 3*ms, 0))); err != nil {
			t.Fatal(err)
		}
		gets.gaps(t, 2*ms, 5*ms, 8*ms)
	})
	t.Run("provision states", func(t *testing.T) {
		cloud := testcloud.New(t)
		var gets strategyGets
		gets.serve(cloud, "GET /v1/nodes/n1", 4, `{"uuid":"n1","provision_state":"deploying"}`, `{"uuid":"n1","provision_state":"active"}`)
		api := nodes.New(r2icClient(cloud, "1.99"))
		result, err := api.AwaitProvisionStates(context.Background(), []string{"n1"}, "active", nodes.WithProvisionWaitInterval(time.Hour), nodes.WithProvisionWaitStrategy(growing))
		if err != nil || len(result.Success) != 1 {
			t.Fatal(result, err)
		}
		gets.gaps(t, 2*ms, 4*ms, 8*ms)
	})
	t.Run("machine", func(t *testing.T) {
		cloud := testcloud.New(t)
		r2icServe(t, cloud, "PUT /v1/nodes/n1/states/provision", 202, "")
		var gets strategyGets
		gets.serve(cloud, "GET /v1/nodes/n1", 4, `{"uuid":"n1","provision_state":"deploying"}`, `{"uuid":"n1","provision_state":"active"}`)
		node, err := r2icService(cloud).NodeSetProvisionState(context.Background(), "n1", "active", v1.WithMachineWait(true), v1.WithMachineWaitInterval(time.Hour), v1.WithMachineWaitStrategy(growing))
		if err != nil || node.ProvisionState != "active" {
			t.Fatal(node, err)
		}
		gets.gaps(t, 2*ms, 4*ms, 8*ms)
	})
}
