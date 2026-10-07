package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/orchestration/v1/stackevents"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestHeatStackEventsPaginationUsesLastEventMarkerAndEmptyPage(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET "+heatEventBase+"/events", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("limit") != "1" || r.URL.Query().Get("nested_depth") != "2" || r.Header.Get("X-Events") != "all-pages" {
			t.Errorf("pagination lost query/header: %s %v", r.URL, r.Header)
		}
		w.Header().Set("X-Request-Id", "page-"+r.URL.Query().Get("marker"))
		switch r.URL.Query().Get("marker") {
		case "":
			testcloud.JSON(w, 200, `{"events":[{"id":"first","event_time":"2026-10-01T00:00:00Z"}],"links":[{"rel":"next","href":"https://example.invalid/not-a-marker-page"}]}`)
		case "first":
			testcloud.JSON(w, 200, `{"events":[{"id":"last","event_time":"2026-10-01T00:01:00+09:00"}]}`)
		case "last":
			testcloud.JSON(w, 200, `{"events":[]}`)
		default:
			t.Errorf("unexpected marker: %s", r.URL)
		}
	})
	values, err := heatEventScope(t, cloud).All(context.Background(), stackevents.WithListOptions(stackevents.ListOpts{Limit: 1}), stackevents.WithListQuery("nested_depth", "2"), stackevents.WithListHeader("X-Events", "all-pages"))
	if err != nil || len(values) != 2 || values[0].ID != "first" || values[1].ID != "last" || values[0].Header.Get("X-Request-Id") != "page-" || values[1].Header.Get("X-Request-Id") != "page-first" || calls.Load() != 3 {
		t.Fatalf("values=%v err=%v calls=%d", values, err, calls.Load())
	}
	values[0].Header.Set("X-Request-Id", "mutated")
	if values[1].Header.Get("X-Request-Id") != "page-first" {
		t.Fatal("result headers share caller-mutable state")
	}
}

func TestHeatStackEventsPaginationStopsOnBreakAndCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("marker") != "" {
			t.Errorf("unexpected next page after break/cancellation: %s", r.URL)
		}
		testcloud.JSON(w, 200, `{"events":[{"id":"first"},{"id":"later","event_time":"bad-time"}]}`)
	})
	scope := heatEventScope(t, cloud)
	for value, err := range scope.List(context.Background()) {
		if err != nil || value.ID != "first" {
			t.Fatalf("later field decoded before break: value=%v err=%v", value, err)
		}
		break
	}
	ctx, cancel := context.WithCancel(context.Background())
	var sawCancellation bool
	for value, err := range heatResourceEventScope(t, scope, "random").List(ctx) {
		if value != nil {
			cancel()
			continue
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("between-item cancellation=%v", err)
		}
		sawCancellation = true
	}
	cancel()
	if calls.Load() != 2 || !sawCancellation {
		t.Fatalf("calls=%d cancellation=%v", calls.Load(), sawCancellation)
	}
}

func TestHeatStackEventsPaginationRejectsRepeatedMarkerBeforeRefetch(t *testing.T) {
	for _, resourceName := range []string{"", "random"} {
		t.Run(resourceName, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, `{"events":[{"id":"same"}]}`)
			})
			scope := heatEventScope(t, cloud)
			var err error
			if resourceName == "" {
				_, err = scope.All(context.Background())
			} else {
				_, err = heatResourceEventScope(t, scope, resourceName).All(context.Background())
			}
			var cycle *resource.PaginationCycleError
			if !errors.Is(err, resource.ErrPaginationCycle) || !errors.As(err, &cycle) || cycle.URL == "" || calls.Load() != 2 {
				t.Fatalf("cycle=%v calls=%d", err, calls.Load())
			}
		})
	}
}

func TestHeatStackEventsMalformedResponsesAndLaterErrorsRemainObservable(t *testing.T) {
	for _, body := range []string{`{}`, `{"events":null}`, `{"events":{}}`, `{"events":[null]}`, `{"events":[42]}`, `{"events":[{}]}`, `{"events":[{"id":"a","event_time":"not-time"}]}`, `{"events":[{"id":"a","resource_type":false}]}`, `{"events":[{"id":"first"},{"id":"later","event_time":"bad-time"}]}`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, body) })
			if values, err := heatEventScope(t, cloud).All(context.Background()); values != nil || err == nil {
				t.Fatalf("malformed/partial result reported complete: %v %v", values, err)
			}
		})
	}
	for _, body := range []string{`{}`, `{"event":null}`, `{"event":[]}`, `{"event":{}}`, `{"event":{"id":"a","event_time":"bad-time"}}`, `{"event":{"id":"a","resource_type":false}}`} {
		t.Run("get "+body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, body) })
			if value, err := heatEventScope(t, cloud).Get(context.Background(), "random", "event-id"); value != nil || err == nil {
				t.Fatalf("malformed detail accepted: %v %v", value, err)
			}
		})
	}
}

func TestHeatStackEventsPreservesGetAndPaginationHTTPErrors(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Request-Id", "failed-event")
				testcloud.JSON(w, status, `{"error":{"message":"event unavailable"}}`)
			})
			scope := heatEventScope(t, cloud)
			bound := heatResourceEventScope(t, scope, "random")
			for _, run := range []func() error{
				func() error { _, err := scope.Get(context.Background(), "random", "event-id"); return err },
				func() error { _, err := scope.All(context.Background()); return err },
				func() error { _, err := bound.All(context.Background()); return err },
			} {
				err := run()
				var failure gophercloud.ErrUnexpectedResponseCode
				if !gophercloud.ResponseCodeIs(err, status) || !errors.As(err, &failure) || failure.ResponseHeader.Get("X-Request-Id") != "failed-event" || len(failure.Body) == 0 || failure.URL == "" {
					t.Fatalf("original HTTP failure lost: %v", err)
				}
			}
			_, err := bound.Get(context.Background(), "event-id")
			if status == 404 && !errors.Is(err, resource.ErrNotFound) || status != 404 && errors.Is(err, resource.ErrNotFound) {
				t.Fatalf("GET absence policy=%v", err)
			}
		})
	}
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			testcloud.JSON(w, 200, `{"events":[{"id":"first"}]}`)
			return
		}
		testcloud.JSON(w, 403, `{"message":"second page forbidden"}`)
	})
	values, err := heatEventScope(t, cloud).All(context.Background())
	if values != nil || !gophercloud.ResponseCodeIs(err, 403) || calls.Load() != 2 {
		t.Fatalf("later-page error ignored: values=%v err=%v calls=%d", values, err, calls.Load())
	}
}

func TestHeatStackEventsParentErrorsAndDuplicateNamesRemainObservable(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		code int
		want error
	}{
		{"missing", `{"stacks":[]}`, 200, resource.ErrNotFound},
		{"ambiguous", `{"stacks":[{"stack_name":"app","id":"a"},{"stack_name":"app","id":"b"}]}`, 200, resource.ErrAmbiguous},
		{"forbidden", `{"message":"denied"}`, 403, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, tc.code, tc.body)
			})
			value, err := stackevents.New(cloud.Client("orchestration", "/heat")).InStack(context.Background(), resource.Name("app"))
			if value != nil || err == nil || tc.want != nil && !errors.Is(err, tc.want) || tc.code != 200 && !gophercloud.ResponseCodeIs(err, tc.code) || calls.Load() != 1 {
				t.Fatalf("scope=%v err=%v requests=%d", value, err, calls.Load())
			}
		})
	}
}

func TestHeatStackEventsInFlightCancellationReachesHTTPRequest(t *testing.T) {
	cloud := testcloud.New(t)
	started := make(chan struct{})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	})
	scope := heatEventScope(t, cloud)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := scope.All(ctx); done <- err }()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("in-flight list cancellation=%v", err)
	}
}
