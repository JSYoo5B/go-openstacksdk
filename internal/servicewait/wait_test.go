package servicewait_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/servicewait"
	"gophercloudsdk/resource"
)

type waitValue struct {
	ID       string
	Status   string `json:"status"`
	State    string `json:"state"`
	Progress int    `json:"progress"`
}

func collection(get func(context.Context, string) (*waitValue, error), deletes *int) *resource.Collection[waitValue] {
	return resource.NewCollection(resource.Adapter[waitValue]{
		Kind: "policy-items", Get: get,
		ID: func(v *waitValue) string { return v.ID }, Status: func(v *waitValue) string { return v.Status },
		Failed: func(state string) bool {
			return strings.HasPrefix(strings.ToLower(state), "error") || strings.EqualFold(state, "failed") || strings.EqualFold(state, "killed")
		},
		Delete: func(context.Context, string) error {
			if deletes != nil {
				*deletes++
			}
			return nil
		},
	})
}

func run(kind string, ctx context.Context, c *resource.Collection[waitValue], options ...resource.WaitOption) (*waitValue, error) {
	switch kind {
	case "state":
		return servicewait.State(ctx, c, resource.ID("fixed"), "ready", "ERROR", options...)
	case "server":
		return servicewait.Server(ctx, c, resource.ID("fixed"), "ready", options...)
	case "delete":
		return nil, servicewait.Delete(ctx, c, resource.ID("fixed"), options...)
	default:
		panic("invalid test policy")
	}
}

func TestServiceWaitDeadlinesDefaultsParentsAndCallerOrdering(t *testing.T) {
	for _, kind := range []string{"state", "server", "delete"} {
		for _, policy := range []struct {
			name    string
			options []resource.WaitOption
			timeout time.Duration
		}{
			{"default", nil, 0},
			{"caller-bounded", []resource.WaitOption{resource.WithTimeout(8 * time.Second)}, 8 * time.Second},
			{"caller-unlimited", []resource.WaitOption{resource.WithUnlimitedWait()}, 0},
			{"timeout-then-unlimited", []resource.WaitOption{resource.WithTimeout(8 * time.Second), resource.WithUnlimitedWait()}, 0},
			{"unlimited-then-timeout", []resource.WaitOption{resource.WithUnlimitedWait(), resource.WithTimeout(8 * time.Second)}, 8 * time.Second},
		} {
			for _, parentDeadline := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/parent=%v", kind, policy.name, parentDeadline), func(t *testing.T) {
					parent := context.Background()
					var cancel context.CancelFunc
					if parentDeadline {
						parent, cancel = context.WithTimeout(parent, 20*time.Second)
						defer cancel()
					}
					want := policy.timeout
					if policy.name == "default" && kind != "state" {
						want = 120 * time.Second
					}
					calls := 0
					before := time.Now()
					c := collection(func(ctx context.Context, id string) (*waitValue, error) {
						calls++
						if id != "fixed" {
							t.Errorf("routeID=%q", id)
						}
						deadline, has := ctx.Deadline()
						if has != (want > 0 || parentDeadline) {
							t.Errorf("deadline=%v present=%v wantedtimeout=%v", deadline, has, want)
						}
						if parentDeadline && (want == 0 || want >= 20*time.Second) {
							original, _ := parent.Deadline()
							if !deadline.Equal(original) {
								t.Errorf("parent deadline changed: %v want %v", deadline, original)
							}
						} else if want > 0 {
							// Wide tolerance measures the installed policy, not scheduling latency.
							remaining := deadline.Sub(before)
							if remaining < want-time.Second || remaining > want+time.Second {
								t.Errorf("deadline delta=%v want=%v", remaining, want)
							}
						}
						if kind == "delete" {
							return nil, gophercloud.ErrUnexpectedResponseCode{Actual: 404}
						}
						return &waitValue{ID: "incidental-response-id", Status: "READY"}, nil
					}, nil)
					value, err := run(kind, parent, c, policy.options...)
					if err != nil || calls != 1 || kind != "delete" && (value == nil || value.Status != "READY") {
						t.Fatalf("value=%v err=%v calls=%d", value, err, calls)
					}
					if parent.Err() != nil {
						t.Fatalf("wait canceled parent: %v", parent.Err())
					}
				})
			}
		}
	}
	// The new service policies do not change generic Collection's five minutes.
	for _, deleted := range []bool{false, true} {
		calls := 0
		before := time.Now()
		c := collection(func(ctx context.Context, id string) (*waitValue, error) {
			calls++
			deadline, has := ctx.Deadline()
			delta := deadline.Sub(before)
			if !has || delta < 299*time.Second || delta > 301*time.Second {
				t.Errorf("generic deadline=%v delta=%v", deadline, delta)
			}
			if deleted {
				return nil, gophercloud.ErrUnexpectedResponseCode{Actual: 404}
			}
			return &waitValue{ID: id, Status: "ready"}, nil
		}, nil)
		var err error
		if deleted {
			err = c.WaitDeleted(context.Background(), resource.ID("fixed"))
		} else {
			_, err = c.Wait(context.Background(), resource.ID("fixed"), "ready")
		}
		if err != nil || calls != 1 {
			t.Fatalf("generic deleted=%v err=%v calls=%d", deleted, err, calls)
		}
	}
}

func TestServiceWaitExactFailuresReplacementAndTargetPrecedence(t *testing.T) {
	for _, kind := range []string{"state", "server"} {
		for _, tc := range []struct {
			name, state, target string
			options             []resource.WaitOption
			failed              bool
		}{
			{"default-error", "eRrOr", "ready", nil, true},
			{"prefix-is-not-failure", "error_deleting", "ready", nil, false},
			{"failed-is-not-failure", "failed", "ready", nil, false},
			{"killed-is-not-failure", "killed", "ready", nil, false},
			{"disabled", "ERROR", "ready", []resource.WaitOption{resource.WithFailureStates()}, false},
			{"replaced-default", "ERROR", "ready", []resource.WaitOption{resource.WithFailureStates("BROKEN")}, false},
			{"replacement-failure", "broken", "ready", []resource.WaitOption{resource.WithFailureStates("BROKEN")}, true},
			{"last-wins", "broken", "ready", []resource.WaitOption{resource.WithFailureStates("BROKEN"), resource.WithFailureStates("ERROR")}, false},
			{"target-first", "ERROR", "error", nil, false},
		} {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				calls := 0
				c := collection(func(_ context.Context, id string) (*waitValue, error) {
					calls++
					state := tc.state
					if calls > 1 {
						state = tc.target
					}
					return &waitValue{ID: id, Status: state}, nil
				}, nil)
				options := append([]resource.WaitOption{resource.WithPollInterval(time.Millisecond)}, tc.options...)
				var value *waitValue
				var err error
				if kind == "state" {
					value, err = servicewait.State(context.Background(), c, resource.ID("fixed"), tc.target, "error", options...)
				} else {
					value, err = servicewait.Server(context.Background(), c, resource.ID("fixed"), tc.target, options...)
				}
				if tc.failed {
					var failure *resource.FailedStateError
					if value != nil || !errors.As(err, &failure) || failure.ID != "fixed" || failure.Status != tc.state || calls != 1 {
						t.Fatalf("value=%v err=%v failure=%+v calls=%d", value, err, failure, calls)
					}
				} else {
					want := 2
					if tc.name == "target-first" {
						want = 1
					}
					if err != nil || value == nil || !strings.EqualFold(value.Status, tc.target) || calls != want {
						t.Fatalf("value=%v err=%v calls=%d want=%d", value, err, calls, want)
					}
				}
			})
		}
	}
}

func TestServiceWaitDeleteObservesAbsenceWithoutFailureOrMutation(t *testing.T) {
	for _, terminal := range []string{"native404", "nil", "deleted"} {
		t.Run(terminal, func(t *testing.T) {
			gets, deletes := 0, 0
			var progress []int
			c := collection(func(_ context.Context, id string) (*waitValue, error) {
				gets++
				if id != "fixed" {
					t.Errorf("routeID=%q", id)
				}
				if gets == 1 {
					return &waitValue{ID: "response-decoy", Status: "ERROR", Progress: 37}, nil
				}
				switch terminal {
				case "native404":
					return nil, gophercloud.ErrUnexpectedResponseCode{Actual: 404}
				case "nil":
					return nil, nil
				default:
					return &waitValue{ID: "another-decoy", Status: "DELETED", Progress: 100}, nil
				}
			}, &deletes)
			err := servicewait.Delete(context.Background(), c, resource.ID("fixed"), resource.WithPollInterval(time.Millisecond), resource.WithFailureStates("ERROR"), resource.WithProgressCallback(func(value int) { progress = append(progress, value) }))
			if err != nil || gets != 2 || deletes != 0 || !reflect.DeepEqual(progress, []int{37}) {
				t.Fatalf("err=%v gets=%d deletes=%d progress=%v", err, gets, deletes, progress)
			}
		})
	}
	for _, code := range []int{403, 409, 500} {
		calls := 0
		c := collection(func(context.Context, string) (*waitValue, error) {
			calls++
			return nil, gophercloud.ErrUnexpectedResponseCode{Actual: code}
		}, nil)
		err := servicewait.Delete(context.Background(), c, resource.ID("fixed"))
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != code || calls != 1 {
			t.Fatalf("native=%+v err=%v calls=%d", native, err, calls)
		}
	}
}

func TestServiceWaitDefaultTwoSecondIntervalAndCallerShortening(t *testing.T) {
	var fetched []time.Time
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c := collection(func(_ context.Context, id string) (*waitValue, error) {
		fetched = append(fetched, time.Now())
		status := "building"
		if len(fetched) == 2 {
			status = "ready"
		}
		return &waitValue{ID: id, Status: status}, nil
	}, nil)
	if _, err := servicewait.State(ctx, c, resource.ID("fixed"), "ready", "ERROR"); err != nil || len(fetched) != 2 {
		t.Fatalf("default err=%v fetches=%d", err, len(fetched))
	}
	if elapsed := fetched[1].Sub(fetched[0]); elapsed < 1500*time.Millisecond {
		t.Fatalf("default interval too short: %v", elapsed)
	}
	// A one-second parent budget cannot reach a second poll at the two-second
	// default. The caller's ten-millisecond interval must allow it to finish.
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	fetched = nil
	if _, err := servicewait.Server(ctx, c, resource.ID("fixed"), "ready", resource.WithPollInterval(10*time.Millisecond)); err != nil || len(fetched) != 2 {
		t.Fatalf("caller interval err=%v fetches=%d", err, len(fetched))
	}
}

func TestServiceWaitPreflightAndCallerCancellationBeforeAnotherFetch(t *testing.T) {
	for _, kind := range []string{"state", "server", "delete"} {
		t.Run(kind, func(t *testing.T) {
			calls := 0
			c := collection(func(context.Context, string) (*waitValue, error) {
				calls++
				return &waitValue{ID: "fixed", Status: "building"}, nil
			}, nil)
			if _, err := run(kind, nil, c); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatalf("nil ctx err=%v calls=%d", err, calls)
			}
			if _, err := run(kind, context.Background(), nil); !errors.Is(err, resource.ErrUnsupported) || calls != 0 {
				t.Fatalf("nil collection err=%v calls=%d", err, calls)
			}
			for _, options := range [][]resource.WaitOption{{nil}, {resource.WithTimeout(0)}, {resource.WithTimeout(-time.Second)}, {resource.WithPollInterval(0)}, {resource.WithPollInterval(-time.Second)}, {resource.WithFailureStates(" ")}, {resource.WithStatusAttribute("missing")}, {resource.WithProgressCallback(nil)}} {
				_, err := run(kind, context.Background(), c, options...)
				if err == nil || calls != 0 {
					t.Fatalf("invalid opts err=%v calls=%d", err, calls)
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := run(kind, ctx, c); !errors.Is(err, context.Canceled) || calls != 0 {
				t.Fatalf("canceled parent err=%v calls=%d", err, calls)
			}
			ctx, cancel = context.WithCancel(context.Background())
			defer cancel()
			c = collection(func(context.Context, string) (*waitValue, error) {
				calls++
				return &waitValue{ID: "fixed", Status: "building", Progress: 9}, nil
			}, nil)
			callbacks := 0
			_, err := run(kind, ctx, c, resource.WithProgressCallback(func(int) { callbacks++; cancel() }))
			if !errors.Is(err, context.Canceled) || calls != 1 || callbacks != 1 {
				t.Fatalf("callback cancellation err=%v calls=%d callbacks=%d", err, calls, callbacks)
			}
			// Cancellation while a fetch is active is also terminal. No polling
			// interval or second request follows the actual returned context error.
			ctx, cancel = context.WithCancel(context.Background())
			defer cancel()
			calls = 0
			c = collection(func(callctx context.Context, _ string) (*waitValue, error) {
				calls++
				cancel()
				<-callctx.Done()
				return nil, callctx.Err()
			}, nil)
			_, err = run(kind, ctx, c)
			if !errors.Is(err, context.Canceled) || calls != 1 {
				t.Fatalf("inflight cancellation err=%v calls=%d", err, calls)
			}
		})
	}
	calls := 0
	c := collection(func(context.Context, string) (*waitValue, error) { calls++; return nil, nil }, nil)
	for _, invoke := range []func() error{
		func() error {
			_, err := servicewait.State(context.Background(), c, resource.ID("fixed"), " ", "ERROR")
			return err
		},
		func() error {
			_, err := servicewait.Server(context.Background(), c, resource.ID("fixed"), "")
			return err
		},
		func() error {
			_, err := servicewait.State(context.Background(), c, resource.ID("fixed"), "ready", "")
			return err
		},
		func() error { return servicewait.Delete(context.Background(), c, resource.ID("bad/route")) },
	} {
		if err := invoke(); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatalf("invalid ref/target/failure err=%v calls=%d", err, calls)
		}
	}
}

func TestServiceWaitReusableCallerOptionsRemainOwnedAndSelectAttributes(t *testing.T) {
	states := []string{"BROKEN"}
	options := make([]resource.WaitOption, 3, 8)
	options[0] = resource.WithFailureStates(states...)
	options[1] = resource.WithStatusAttribute("state")
	options[2] = resource.WithPollInterval(time.Millisecond)
	states[0] = "changed after construction"
	for _, kind := range []string{"state", "server"} {
		for repeat := 0; repeat < 2; repeat++ {
			calls := 0
			c := collection(func(context.Context, string) (*waitValue, error) {
				calls++
				return &waitValue{ID: "fixed", Status: "ERROR", State: "broken"}, nil
			}, nil)
			_, err := run(kind, context.Background(), c, options...)
			var failure *resource.FailedStateError
			if !errors.As(err, &failure) || failure.Status != "broken" || calls != 1 {
				t.Fatalf("kind=%s repeat=%d err=%v calls=%d", kind, repeat, err, calls)
			}
			for index, extra := range options[:cap(options)][len(options):] {
				if extra != nil {
					t.Fatalf("caller option slice tail overwritten at %d", index+len(options))
				}
			}
		}
	}
	// The same attribute selector can choose deletion completion without making
	// the caller's exact failure policy a deletion failure predicate.
	calls := 0
	c := collection(func(context.Context, string) (*waitValue, error) {
		calls++
		return &waitValue{ID: "fixed", Status: "ERROR", State: "deleted"}, nil
	}, nil)
	if err := servicewait.Delete(context.Background(), c, resource.ID("fixed"), options...); err != nil || calls != 1 {
		t.Fatalf("attribute delete err=%v calls=%d", err, calls)
	}
}
