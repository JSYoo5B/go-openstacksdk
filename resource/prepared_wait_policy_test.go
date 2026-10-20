package resource

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPreparedWaitPolicyDoesNotReapplyApplicationOptions(t *testing.T) {
	type model struct {
		ID, Status string
		Progress   int
	}
	calls, gets, progress := 0, 0, 0
	policy, err := PrepareWaitOptionsFor[model](func(o *waitOptions) error {
		calls++
		if calls != 1 {
			return errors.New("applied twice")
		}
		o.interval = time.Millisecond
		o.progressCallback = func(n int) { progress = n }
		return nil
	})
	if err != nil || policy.ValidateFixedStatus() != nil {
		t.Fatal(policy, err)
	}
	c := NewCollection(Adapter[model]{Kind: "model", ID: func(m *model) string { return m.ID }, Status: func(m *model) string { return m.Status },
		Get: func(context.Context, string) (*model, error) {
			gets++
			if gets == 1 {
				return &model{ID: "fixed", Status: "BUILD", Progress: 25}, nil
			}
			return &model{ID: "fixed", Status: "ACTIVE"}, nil
		}})
	value, err := c.Wait(context.Background(), ID("fixed"), "ACTIVE", WithWaitPolicy(policy))
	if err != nil || value == nil || calls != 1 || gets != 2 || progress != 25 {
		t.Fatal(value, err, calls, gets, progress)
	}
	custom, err := PrepareWaitOptionsFor[model](WithStatusAttribute("Status"))
	if err != nil || !errors.Is(custom.ValidateFixedStatus(), ErrUnsupported) {
		t.Fatal(custom, err)
	}
	if !errors.Is((WaitPolicy{}).ValidateFixedStatus(), ErrInvalidOption) {
		t.Fatal("zero policy accepted")
	}
}

func TestWaitGuardStopsAfterProgressBeforeAnotherFetch(t *testing.T) {
	type model struct {
		ID, Status string
		Progress   int
	}
	changed := false
	gets := 0
	cause := errors.New("source changed in progress")
	c := NewCollection(Adapter[model]{Kind: "model", ID: func(m *model) string { return m.ID }, Status: func(m *model) string { return m.Status },
		Get: func(context.Context, string) (*model, error) {
			gets++
			return &model{ID: "fixed", Status: "BUILD"}, nil
		},
		WaitGuard: func(context.Context) error {
			if changed {
				return cause
			}
			return nil
		},
	})
	_, err := c.Wait(context.Background(), ID("fixed"), "ACTIVE", WithPollInterval(time.Hour), WithProgressCallback(func(int) { changed = true }))
	if !errors.Is(err, cause) || gets != 1 {
		t.Fatal(err, gets)
	}
}
