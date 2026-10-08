package resource_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type waitEntry struct {
	ID     string
	Status string
}

func waitCollection(get func(context.Context, string) (*waitEntry, error)) *resource.Collection[waitEntry] {
	return resource.NewCollection(resource.Adapter[waitEntry]{
		Kind: "entry", Get: get,
		ID:     func(v *waitEntry) string { return v.ID },
		Status: func(v *waitEntry) string { return v.Status },
		Failed: func(s string) bool { return strings.EqualFold(s, "error") },
	})
}

func TestWaitFailurePolicyReplacementAndTargetPrecedence(t *testing.T) {
	for _, test := range []struct {
		name, state string
		options     []resource.WaitOption
		failed      bool
	}{
		{name: "service default", state: "ERROR", failed: true},
		{name: "custom case insensitive", state: "broken", options: []resource.WaitOption{resource.WithFailureStates("BROKEN")}, failed: true},
		{name: "replacement", state: "ERROR", options: []resource.WaitOption{resource.WithFailureStates("BROKEN")}},
		{name: "disabled", state: "ERROR", options: []resource.WaitOption{resource.WithFailureStates()}},
		{name: "last wins", state: "broken", options: []resource.WaitOption{resource.WithFailureStates("BROKEN"), resource.WithFailureStates("ERROR")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			collection := waitCollection(func(_ context.Context, id string) (*waitEntry, error) {
				calls++
				state := test.state
				if calls > 1 {
					state = "ready"
				}
				return &waitEntry{ID: id, Status: state}, nil
			})
			options := append([]resource.WaitOption{resource.WithPollInterval(time.Millisecond)}, test.options...)
			value, err := collection.Wait(context.Background(), resource.ID("fixed"), "READY", options...)
			if test.failed {
				var failure *resource.FailedStateError
				if value != nil || !errors.As(err, &failure) || failure.ID != "fixed" || failure.Status != test.state || calls != 1 {
					t.Fatalf("value=%v calls=%d error=%v", value, calls, err)
				}
			} else if err != nil || value == nil || value.Status != "ready" || calls != 2 {
				t.Fatalf("value=%v calls=%d error=%v", value, calls, err)
			}
		})
	}
	collection := waitCollection(func(_ context.Context, id string) (*waitEntry, error) {
		return &waitEntry{ID: id, Status: "ERROR"}, nil
	})
	if value, err := collection.Wait(context.Background(), resource.ID("fixed"), "error", resource.WithFailureStates("ERROR")); err != nil || value == nil {
		t.Fatalf("target must take precedence: value=%v error=%v", value, err)
	}
}

func TestWaitFailureStatesSnapshotAndValidationBeforeFetch(t *testing.T) {
	states := []string{"BROKEN"}
	option := resource.WithFailureStates(states...)
	states[0] = "other"
	calls := 0
	collection := waitCollection(func(_ context.Context, id string) (*waitEntry, error) {
		calls++
		return &waitEntry{ID: id, Status: "broken"}, nil
	})
	if _, err := collection.Wait(context.Background(), resource.ID("fixed"), "ready", option); !errors.Is(err, resource.ErrFailedState) || calls != 1 {
		t.Fatalf("snapshot calls=%d error=%v", calls, err)
	}
	for _, options := range [][]resource.WaitOption{
		{resource.WithFailureStates(" ")}, {resource.WithTimeout(0)}, {nil},
	} {
		calls = 0
		if _, err := collection.Wait(context.Background(), resource.ID("fixed"), "ready", options...); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatalf("invalid options calls=%d error=%v", calls, err)
		}
	}
}

func TestWaitUnlimitedAndBoundedOptionsPreserveParentContext(t *testing.T) {
	for _, deletion := range []bool{false, true} {
		for _, bounded := range []bool{false, true} {
			for _, parentDeadline := range []bool{false, true} {
				name := strings.Join([]string{map[bool]string{false: "status", true: "deletion"}[deletion], map[bool]string{false: "unlimited", true: "bounded"}[bounded], map[bool]string{false: "cancel", true: "deadline"}[parentDeadline]}, "/")
				t.Run(name, func(t *testing.T) {
					parent, cancel := context.WithCancel(context.Background())
					defer cancel()
					if parentDeadline {
						var deadlineCancel context.CancelFunc
						parent, deadlineCancel = context.WithTimeout(parent, time.Minute)
						defer deadlineCancel()
					}
					calls := 0
					collection := waitCollection(func(ctx context.Context, id string) (*waitEntry, error) {
						calls++
						deadline, ok := ctx.Deadline()
						if ok != (bounded || parentDeadline) {
							t.Fatalf("unexpected deadline: %v %v", deadline, ok)
						}
						if parentDeadline {
							want, _ := parent.Deadline()
							if !deadline.Equal(want) {
								t.Fatalf("parent deadline changed: %v want %v", deadline, want)
							}
						}
						cancel()
						return &waitEntry{ID: id, Status: "ready"}, nil
					})
					options := []resource.WaitOption{resource.WithTimeout(time.Hour), resource.WithUnlimitedWait()}
					if bounded {
						options = append(options, resource.WithTimeout(2*time.Hour))
					}
					var err error
					if deletion {
						err = collection.WaitDeleted(parent, resource.ID("fixed"), options...)
					} else {
						_, err = collection.Wait(parent, resource.ID("fixed"), "ready", options...)
					}
					if !errors.Is(err, context.Canceled) || calls != 1 {
						t.Fatalf("calls=%d error=%v", calls, err)
					}
				})
			}
		}
	}
}

func TestWaitMissingSuccessfulResponseDoesNotPanic(t *testing.T) {
	collection := waitCollection(func(context.Context, string) (*waitEntry, error) { return nil, nil })
	if _, err := collection.Wait(context.Background(), resource.ID("fixed"), "ready"); !errors.Is(err, resource.ErrFailedState) {
		t.Fatalf("error=%v", err)
	}
}
