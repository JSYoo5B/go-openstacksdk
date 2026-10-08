package resource_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestWaitDeletedOptInGuardProtectsEveryCompletionBoundary(t *testing.T) {
	for _, phase := range []string{"before fetch", "missing response", "nil response", "deleted response", "progress callback"} {
		t.Run(phase, func(t *testing.T) {
			fault := errors.New("captured wait source changed")
			changed := phase == "before fetch"
			gets, callbacks, guards := 0, 0, 0
			collection := resource.NewCollection(resource.Adapter[waitEntry]{
				Kind:   "entry",
				ID:     func(value *waitEntry) string { return value.ID },
				Status: func(value *waitEntry) string { return value.Status },
				WaitGuard: func(context.Context) error {
					guards++
					if changed {
						return fault
					}
					return nil
				},
				Get: func(context.Context, string) (*waitEntry, error) {
					gets++
					if phase != "progress callback" {
						changed = true
					}
					switch phase {
					case "missing response":
						return nil, gophercloud.ErrUnexpectedResponseCode{Actual: 404}
					case "nil response":
						return nil, nil
					case "deleted response":
						return &waitEntry{ID: "fixed", Status: "deleted"}, nil
					default:
						return &waitEntry{ID: "fixed", Status: "active"}, nil
					}
				},
			})
			err := collection.WaitDeleted(context.Background(), resource.ID("fixed"),
				resource.WithPollInterval(time.Millisecond),
				resource.WithProgressCallback(func(progress int) {
					callbacks++
					if progress != 0 {
						t.Fatal(progress)
					}
					changed = true
				}))
			wantGets, wantGuards, wantCallbacks := 1, 2, 0
			if phase == "before fetch" {
				wantGets, wantGuards = 0, 1
			}
			if phase == "progress callback" {
				wantGuards, wantCallbacks = 3, 1
			}
			if !errors.Is(err, fault) || gets != wantGets || guards != wantGuards || callbacks != wantCallbacks {
				t.Fatalf("err=%v gets=%d guards=%d callbacks=%d", err, gets, guards, callbacks)
			}
			if phase == "missing response" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 404 {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestWaitDeletedGuardPreservesFetchFailureAndLegacyNilGuard(t *testing.T) {
	t.Run("physical and source causes remain available", func(t *testing.T) {
		physical, source := errors.New("physical observation failed"), errors.New("source changed while reading")
		gets := 0
		collection := resource.NewCollection(resource.Adapter[waitEntry]{
			Kind: "entry", Get: func(context.Context, string) (*waitEntry, error) { gets++; return nil, physical },
			WaitGuard: func(context.Context) error {
				if gets != 0 {
					return source
				}
				return nil
			},
		})
		err := collection.WaitDeleted(context.Background(), resource.ID("fixed"))
		if !errors.Is(err, physical) || !errors.Is(err, source) || gets != 1 {
			t.Fatal(err, gets)
		}
	})
	for _, terminal := range []string{"404", "nil", "deleted"} {
		t.Run(fmt.Sprintf("nil guard preserves %s", terminal), func(t *testing.T) {
			gets, callbacks := 0, 0
			collection := resource.NewCollection(resource.Adapter[waitEntry]{
				Kind: "entry", Status: func(value *waitEntry) string { return value.Status },
				Get: func(context.Context, string) (*waitEntry, error) {
					gets++
					if gets == 1 {
						return &waitEntry{ID: "fixed", Status: "active"}, nil
					}
					switch terminal {
					case "404":
						return nil, gophercloud.ErrUnexpectedResponseCode{Actual: 404}
					case "nil":
						return nil, nil
					default:
						return &waitEntry{ID: "fixed", Status: "deleted"}, nil
					}
				},
			})
			err := collection.WaitDeleted(context.Background(), resource.ID("fixed"), resource.WithPollInterval(time.Millisecond), resource.WithProgressCallback(func(progress int) {
				if progress != 0 {
					t.Fatal(progress)
				}
				callbacks++
			}))
			if err != nil || gets != 2 || callbacks != 1 {
				t.Fatal(err, gets, callbacks)
			}
		})
	}
}

func TestWaitDeletedCanceledReadRetainsPhysicalResponseEvidence(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	marker := errors.New("canceled at physical read boundary")
	physical := &resource.ResponseError{StatusCode: 201, Header: http.Header{"X-Proof": {"accepted"}}, Body: []byte(`{"status":"active"}`)}
	gets := 0
	collection := resource.NewCollection(resource.Adapter[waitEntry]{
		Kind: "entry", Get: func(context.Context, string) (*waitEntry, error) {
			gets++
			cancel(marker)
			physical.Cause = errors.Join(ctx.Err(), marker)
			return nil, physical
		},
	})
	err := collection.WaitDeleted(ctx, resource.ID("fixed"))
	var accepted *resource.ResponseError
	if !errors.As(err, &accepted) || accepted != physical || !errors.Is(err, context.Canceled) || !errors.Is(err, marker) || gets != 1 || accepted.StatusCode != 201 || accepted.Header.Get("X-Proof") != "accepted" || string(accepted.Body) != `{"status":"active"}` {
		t.Fatal(err, accepted, gets)
	}
}
