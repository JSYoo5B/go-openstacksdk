package api_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/baremetalintrospection/v1/introspection"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestPyCloudIronicIntrospectionStartAndGetPostsThenReadsOnceWithoutReplay(t *testing.T) {
	cloud := testcloud.New(t)
	rec := &pyCloudIronicRecorder{}
	cloud.Mux.HandleFunc("/v1/introspection/{id}", func(w http.ResponseWriter, r *http.Request) {
		body := rec.add(r)
		if r.Method == http.MethodPost {
			if body != "" {
				t.Errorf("POST body=%q", body)
			}
			if r.PathValue("id") == "refused" {
				testcloud.JSON(w, 409, `{"error":"busy"}`)
				return
			}
			w.WriteHeader(202)
			return
		}
		if r.PathValue("id") == "no-status" {
			testcloud.JSON(w, 503, `{"error":"later"}`)
			return
		}
		testcloud.JSON(w, 200, `{"uuid":"node-a","finished":false,"state":"waiting","error":null}`)
	})
	api := newIronicIntrospection(t, cloud, "1.13")
	ctx := context.Background()
	manageBoot := false
	value, err := api.StartAndGet(ctx, "node-a", introspection.StartOpts{ManageBoot: &manageBoot})
	if err != nil || value.UUID != "node-a" || value.Finished || value.State != "waiting" {
		t.Fatalf("value=%+v err=%v", value, err)
	}
	pyCloudIronicRequireLines(t, rec, "POST /v1/introspection/node-a?manage_boot=false", "GET /v1/introspection/node-a")

	_, err = api.StartAndGet(ctx, "refused", introspection.StartOpts{})
	ironicRequireStatus(t, err, 409)
	var started *introspection.StartedError
	if errors.As(err, &started) {
		t.Fatalf("a rejected start is not a started introspection: %v", err)
	}
	_, err = api.StartAndGet(ctx, "no-status", introspection.StartOpts{})
	if !errors.As(err, &started) || started.NodeID != "no-status" {
		t.Fatalf("err=%v", err)
	}
	ironicRequireStatus(t, err, 503)
	// refused: 1 POST; no-status: 1 POST + 1 GET. Nothing is replayed.
	if got := len(rec.snapshot()); got != 5 {
		t.Fatalf("requests=%q", rec.snapshot())
	}
}

func TestPyCloudIronicIntrospectionStartAndGetRejectsEmptyIdentityAndCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	ironicForbidHTTP(t, cloud)
	api := newIronicIntrospection(t, cloud, "1.13")
	if _, err := api.StartAndGet(context.Background(), "", introspection.StartOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("err=%v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.StartAndGet(ctx, "node-a", introspection.StartOpts{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

func TestPyCloudIronicIntrospectionAbortIgnoresMissingByDefaultAndKeepsOtherErrors(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /v1/introspection/{id}/abort", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.PathValue("id") {
		case "ok":
			w.WriteHeader(202)
		case "gone":
			testcloud.JSON(w, 404, `{"error":"no such"}`)
		case "conflict":
			testcloud.JSON(w, 409, `{"error":"finished"}`)
		default:
			testcloud.JSON(w, 200, `{}`)
		}
	})
	api := newIronicIntrospection(t, cloud, "1.13")
	ctx := context.Background()
	if err := api.AbortIntrospectionWithOptions(ctx, "ok"); err != nil {
		t.Fatal(err)
	}
	if err := api.AbortIntrospectionWithOptions(ctx, "gone"); err != nil {
		t.Fatalf("missing must be ignored by default: %v", err)
	}
	ironicRequireStatus(t, api.AbortIntrospectionWithOptions(ctx, "gone", introspection.WithAbortIgnoreMissing(false)), 404)
	ironicRequireStatus(t, api.AbortIntrospectionWithOptions(ctx, "conflict"), 409)
	ironicRequireStatus(t, api.AbortIntrospectionWithOptions(ctx, "odd-success"), 200)
	if err := api.AbortIntrospectionWithOptions(ctx, "ok", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option err=%v", err)
	}
	if calls.Load() != 5 {
		t.Fatalf("one request per call expected, got %d", calls.Load())
	}
}

func TestPyCloudIronicIntrospectionDataRawSelectsRouteVersionAndKeepsDocument(t *testing.T) {
	cloud := testcloud.New(t)
	rec := &pyCloudIronicRecorder{}
	versions := make(chan string, 4)
	cloud.Mux.HandleFunc("GET /v1/introspection/{id}/data", func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		versions <- r.Header.Get(ironicInspectorVersionHeader)
		testcloud.JSON(w, 200, `{"inventory":{"cpu":{"count":9007199254740993}},"extra":null,"flag":false}`)
	})
	cloud.Mux.HandleFunc("GET /v1/introspection/{id}/data/unprocessed", func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		versions <- r.Header.Get(ironicInspectorVersionHeader)
		if r.PathValue("id") == "gone" {
			testcloud.JSON(w, 404, `{"error":"none"}`)
			return
		}
		testcloud.JSON(w, 200, `{"raw":[1,2,3]}`)
	})
	api := newIronicIntrospection(t, cloud, "1.13")
	ctx := context.Background()
	processed, err := api.GetIntrospectionDataRaw(ctx, "node-a", true)
	if err != nil || string(processed) != `{"inventory":{"cpu":{"count":9007199254740993}},"extra":null,"flag":false}` {
		t.Fatalf("processed=%s err=%v", processed, err)
	}
	raw, err := api.GetIntrospectionDataRaw(ctx, "node-a", false)
	if err != nil || string(raw) != `{"raw":[1,2,3]}` {
		t.Fatalf("raw=%s err=%v", raw, err)
	}
	if got := []string{<-versions, <-versions}; !reflect.DeepEqual(got, []string{"1.13", "1.17"}) {
		t.Fatalf("versions=%q: processed keeps the client version, unprocessed pins 1.17", got)
	}
	_, err = api.GetIntrospectionDataRaw(ctx, "gone", false)
	ironicRequireStatus(t, err, 404)
	if _, err := api.GetIntrospectionDataRaw(ctx, "", true); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("empty id err=%v", err)
	}
	pyCloudIronicRequireLines(t, rec, "GET /v1/introspection/node-a/data", "GET /v1/introspection/node-a/data/unprocessed", "GET /v1/introspection/gone/data/unprocessed")
}

func TestPyCloudIronicIntrospectionDataRawRejectsNon200AndKeepsCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/introspection/{id}/data", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 202, `{}`)
	})
	api := newIronicIntrospection(t, cloud, "")
	_, err := api.GetIntrospectionDataRaw(context.Background(), "node-a", true)
	ironicRequireStatus(t, err, 202)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.GetIntrospectionDataRaw(ctx, "node-a", true); !errors.Is(err, context.Canceled) || calls.Load() != 1 {
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
}

func TestPyCloudIronicIntrospectionWaitIgnoringErrorReturnsFinishedErrorState(t *testing.T) {
	cloud := testcloud.New(t)
	var reads atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/introspection/{id}", func(w http.ResponseWriter, r *http.Request) {
		n := reads.Add(1)
		if r.PathValue("id") == "stuck" {
			testcloud.JSON(w, 200, `{"uuid":"stuck","finished":false,"state":"waiting","error":null}`)
			return
		}
		if n < 2 {
			testcloud.JSON(w, 200, `{"uuid":"node-a","finished":false,"state":"waiting","error":null}`)
			return
		}
		testcloud.JSON(w, 200, `{"uuid":"node-a","finished":true,"state":"error","error":"Canceled by operator"}`)
	})
	api := newIronicIntrospection(t, cloud, "1.13")
	ctx := context.Background()
	value, err := api.WaitUntilFinishedIgnoringError(ctx, resource.ID("node-a"), resource.WithPollInterval(time.Millisecond))
	if err != nil || !value.Finished || value.State != "error" || value.Error != "Canceled by operator" {
		t.Fatalf("value=%+v err=%v", value, err)
	}
	// The strict waiter keeps failing on the same service error.
	reads.Store(1)
	_, err = api.WaitUntilFinished(ctx, resource.ID("node-a"), resource.WithPollInterval(time.Millisecond))
	var failure *introspection.IntrospectionFailureError
	if !errors.As(err, &failure) || !errors.Is(err, resource.ErrFailedState) {
		t.Fatalf("strict err=%v", err)
	}
	_, err = api.WaitUntilFinishedIgnoringError(ctx, resource.ID("stuck"), resource.WithPollInterval(time.Millisecond), resource.WithTimeout(30*time.Millisecond))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout err=%v", err)
	}
	if _, err := api.WaitUntilFinishedIgnoringError(ctx, resource.Name("node-a")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("names are unsupported: %v", err)
	}
}
