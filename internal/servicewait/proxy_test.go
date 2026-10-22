package servicewait_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/servicewait"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// proxyStatusless has string attributes but no status attribute, like most
// Neutron, Keystone, Placement and Inspector models.
type proxyStatusless struct {
	ID    string
	State string `json:"state"`
}

func proxyCollection[T any](get func(context.Context, string) (*T, error), id func(*T) string) *resource.Collection[T] {
	return resource.NewCollection(resource.Adapter[T]{Kind: "proxy-items", Get: get, ID: id})
}

func TestServiceWaitProxyDefaultsDeadlinesIntervalAndStatusAttribute(t *testing.T) {
	ctx := context.Background()
	// Status: no SDK deadline, ERROR fails, status attribute selected without a
	// binding accessor, two-second default interval.
	var fetched []time.Time
	statuses := []string{"BUILD", "active"}
	c := proxyCollection(func(ctx context.Context, id string) (*waitValue, error) {
		if _, has := ctx.Deadline(); has {
			t.Errorf("status wait installed an SDK deadline")
		}
		fetched = append(fetched, time.Now())
		return &waitValue{ID: id, Status: statuses[len(fetched)-1]}, nil
	}, func(v *waitValue) string { return v.ID })
	value, err := servicewait.ProxyStatus(ctx, c, resource.ID("fixed"), "ACTIVE")
	if err != nil || value.Status != "active" || len(fetched) != 2 {
		t.Fatalf("value=%v err=%v fetches=%d", value, err, len(fetched))
	}
	if elapsed := fetched[1].Sub(fetched[0]); elapsed < 1500*time.Millisecond {
		t.Fatalf("default interval too short: %v", elapsed)
	}
	failing := proxyCollection(func(_ context.Context, id string) (*waitValue, error) {
		return &waitValue{ID: id, Status: "error"}, nil
	}, func(v *waitValue) string { return v.ID })
	if _, err := servicewait.ProxyStatus(ctx, failing, resource.ID("fixed"), "ACTIVE"); !errors.Is(err, resource.ErrFailedState) {
		t.Fatalf("ERROR failure err=%v", err)
	}
	// error_deleting is not an exact ERROR match and keeps polling until the
	// caller-bounded deadline.
	if _, err := servicewait.ProxyStatus(ctx, proxyCollection(func(_ context.Context, id string) (*waitValue, error) {
		return &waitValue{ID: id, Status: "error_deleting"}, nil
	}, func(v *waitValue) string { return v.ID }), resource.ID("fixed"), "ACTIVE", resource.WithPollInterval(time.Millisecond), resource.WithTimeout(20*time.Millisecond)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("inexact failure err=%v", err)
	}

	// A model without a status attribute fails before HTTP unless the caller
	// selects one, matching Python's AttributeError for the default attribute.
	calls := 0
	statusless := proxyCollection(func(_ context.Context, id string) (*proxyStatusless, error) {
		calls++
		return &proxyStatusless{ID: id, State: "finished"}, nil
	}, func(v *proxyStatusless) string { return v.ID })
	if _, err := servicewait.ProxyStatus(ctx, statusless, resource.ID("fixed"), "finished"); !errors.Is(err, resource.ErrUnsupported) || calls != 0 {
		t.Fatalf("statusless err=%v calls=%d", err, calls)
	}
	if value, err := servicewait.ProxyStatus(ctx, statusless, resource.ID("fixed"), "FINISHED", resource.WithStatusAttribute("state")); err != nil || value.State != "finished" || calls != 1 {
		t.Fatalf("selected attribute value=%v err=%v calls=%d", value, err, calls)
	}

	// Delete: 120-second SDK deadline, deleted status completes for status
	// models, 404 completes for statusless models.
	before := time.Now()
	deleted := proxyCollection(func(ctx context.Context, id string) (*waitValue, error) {
		deadline, has := ctx.Deadline()
		if remaining := deadline.Sub(before); !has || remaining < 119*time.Second || remaining > 121*time.Second {
			t.Errorf("delete deadline=%v present=%v", deadline, has)
		}
		return &waitValue{ID: id, Status: "DELETED"}, nil
	}, func(v *waitValue) string { return v.ID })
	if err := servicewait.ProxyDelete(ctx, deleted, resource.ID("fixed")); err != nil {
		t.Fatalf("deleted status err=%v", err)
	}
	gets := 0
	gone := proxyCollection(func(_ context.Context, id string) (*proxyStatusless, error) {
		gets++
		if gets < 2 {
			return &proxyStatusless{ID: id, State: "deleted"}, nil
		}
		return nil, gophercloud.ErrUnexpectedResponseCode{Actual: 404}
	}, func(v *proxyStatusless) string { return v.ID })
	if err := servicewait.ProxyDelete(ctx, gone, resource.ID("fixed"), resource.WithPollInterval(time.Millisecond)); err != nil || gets != 2 {
		t.Fatalf("statusless delete err=%v gets=%d", err, gets)
	}
	if err := servicewait.ProxyDelete(ctx, proxyCollection(func(context.Context, string) (*proxyStatusless, error) {
		return nil, gophercloud.ErrUnexpectedResponseCode{Actual: 403}
	}, func(v *proxyStatusless) string { return v.ID }), resource.ID("fixed")); err == nil {
		t.Fatal("403 completed a deletion wait")
	}
	var nilCollection *resource.Collection[waitValue]
	if _, err := servicewait.ProxyStatus(ctx, nilCollection, resource.ID("fixed"), "ACTIVE"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("nil collection err=%v", err)
	}
}
