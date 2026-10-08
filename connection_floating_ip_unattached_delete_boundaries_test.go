package openstack_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestFloatingIPUnattachedPreflightAndCapturedSource(t *testing.T) {
	var connection *sdk.Connection
	var service *compute.Service
	if result, err := connection.DeleteUnattachedFloatingIPs(context.Background()); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(result, err)
	}
	if result, err := service.DeleteUnattachedFloatingIPs(context.Background()); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(result, err)
	}
	for _, mode := range []string{"nil-context", "nil-option", "source", "timeout", "bulk-timeout", "cancel", "replace-servers"} {
		t.Run(mode, func(t *testing.T) {
			_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNone)
			ctx := context.Background()
			var options []compute.FloatingIPDeleteOption
			cause := errors.New("cleanup preflight cancellation")
			switch mode {
			case "nil-context":
				ctx = nil
			case "nil-option":
				options = []compute.FloatingIPDeleteOption{nil}
			case "source":
				options = []compute.FloatingIPDeleteOption{compute.WithFloatingIPDeleteSource("invalid")}
			case "timeout":
				options = []compute.FloatingIPDeleteOption{compute.WithFloatingIPDeleteTimeout(0)}
			case "bulk-timeout":
				options = []compute.FloatingIPDeleteOption{compute.WithFloatingIPDeleteOptions(compute.FloatingIPDeleteOpts{Timeout: -time.Second})}
			case "cancel":
				cancelled, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx = cancelled
			}
			if mode == "replace-servers" {
				cached, err := conn.Compute(ctx)
				if err != nil {
					t.Fatal(err)
				}
				result, err := cached.DeleteUnattachedFloatingIPs(ctx, func(*compute.FloatingIPDeleteOpts) error {
					cached.Servers = nil
					return nil
				})
				if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(state.events) != 0 || len(state.locators) != 1 {
					t.Fatal(result, err, state)
				}
				return
			}
			result, err := conn.DeleteUnattachedFloatingIPs(ctx, options...)
			if result != nil || err == nil || len(state.events) != 0 || len(state.locators) != 0 {
				t.Fatal(result, err, state)
			}
			if mode == "cancel" {
				if !errors.Is(err, cause) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

// Inject faults across the final observed guard checks without fixing a
// particular internal call count. A completed lower Delete stays counted
// even when a later aggregate guard terminates the cleanup.
func TestFloatingIPUnattachedCompletedDeleteSurvivesAggregateGuardFailure(t *testing.T) {
	run := func(failAt int) (*compute.DeleteUnattachedFloatingIPsResult, error, int) {
		t.Helper()
		_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNeutron)
		state.reply = func(r *http.Request) (int, string) {
			if r.Method == "GET" {
				return 200, `{"floatingips":[{"id":"a"}]}`
			}
			return 204, ""
		}
		calls := 0
		cause := errors.New("aggregate guard fault")
		ctx := rest.WithOperationGuard(context.Background(), func(context.Context) error {
			calls++
			if failAt != 0 && calls == failAt {
				return cause
			}
			return nil
		})
		result, err := conn.DeleteUnattachedFloatingIPs(ctx, compute.WithFloatingIPDeleteRetries(0))
		if failAt != 0 {
			if !errors.Is(err, cause) || (result != nil && result.AllDeleted) || len(unattachedDeletes(state.events)) > 1 {
				t.Fatal(failAt, result, err, state)
			}
		}
		return result, err, calls
	}
	baseline, err, calls := run(0)
	if err != nil || baseline.Count != 1 || !baseline.AllDeleted {
		t.Fatal(baseline, err)
	}
	completed := 0
	for point := max(1, calls-15); point <= calls; point++ {
		t.Run(fmt.Sprint(point), func(t *testing.T) {
			result, _, _ := run(point)
			if result != nil && len(result.Items) == 1 && result.Items[0].Deletion != nil && result.Items[0].Deletion.Deleted {
				completed++
				if result.Count != 1 || result.AllDeleted {
					t.Fatal(result)
				}
			}
		})
	}
	if completed == 0 {
		t.Fatal("fault injection did not reach an aggregate guard after a completed Delete")
	}
}

func TestFloatingIPUnattachedCachedServiceAndConnectionShareDependencies(t *testing.T) {
	_, conn, state := floatingDeleteFixture(t, compute.FloatingIPNone)
	service, err := conn.Compute(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	inventory := false
	state.reply = func(r *http.Request) (int, string) {
		if r.Method == "DELETE" {
			return 204, ""
		}
		if !inventory {
			inventory = true
			return 200, `{"floatingips":[{"id":"a"},{"id":"b"}]}`
		}
		return 200, `{"floatingips":[]}`
	}
	calls := 0
	options := []compute.FloatingIPDeleteOption{
		compute.WithFloatingIPDeleteSource(compute.FloatingIPNeutron),
		compute.WithFloatingIPDeleteTimeout(time.Minute),
		func(*compute.FloatingIPDeleteOpts) error { calls++; return nil },
	}
	for phase := 0; phase < 2; phase++ {
		inventory = false
		state.events, state.deadlines = nil, nil
		var result *compute.DeleteUnattachedFloatingIPsResult
		if phase == 0 {
			result, err = service.DeleteUnattachedFloatingIPs(context.Background(), options...)
		} else {
			result, err = conn.DeleteUnattachedFloatingIPs(context.Background(), options...)
		}
		if err != nil || !result.Eligible || !result.AllDeleted || result.Count != 2 || calls != phase+1 || len(state.events) != 5 || !reflect.DeepEqual(state.locators, []string{"compute", "network"}) {
			t.Fatal(result, err, calls, state)
		}
		for _, deadline := range state.deadlines {
			if deadline.IsZero() || !deadline.Equal(state.deadlines[0]) {
				t.Fatal(state.deadlines)
			}
		}
		for _, item := range result.Items {
			if item.Error != nil || !item.Deletion.Absent || item.Deletion.Backend != compute.FloatingIPNeutron || len(item.Deletion.LastVerification.Pages) != 1 {
				t.Fatal(item)
			}
		}
	}
}
