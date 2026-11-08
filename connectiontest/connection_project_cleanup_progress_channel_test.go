package connectiontest_test

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/projectcleanup"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func progressChannelCleanup(t *testing.T) (*cleanupCloud, *projectcleanup.Plan) {
	t.Helper()
	cloud := newCleanupCloud(t)
	cloud.Mux.HandleFunc("DELETE /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		cloud.record("delete " + r.PathValue("id"))
		w.WriteHeader(204)
	})
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		cloud.record("get " + r.PathValue("id"))
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.serveCompute(t, `{"id":"a"},{"id":"b"},{"id":"c"}`, ``)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	return cloud, plan
}

func cleanupEventName(e projectcleanup.Event) string {
	return e.Outcome.Candidate.ID + "=" + string(e.Outcome.State)
}

func TestProjectCleanupProgressChannelFollowsCallbackOrder(t *testing.T) {
	cloud, plan := progressChannelCleanup(t)
	var sent []string
	observer := testcloud.NewProgress(t, func(e projectcleanup.Event) { sent = append(sent, cleanupEventName(e)) })
	var called []string
	result, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithParallelism(2), projectcleanup.WithProgress(observer.C),
		projectcleanup.WithProgressCallback(func(e projectcleanup.Event) error { called = append(called, cleanupEventName(e)); return nil }))
	observer.Done()
	if err != nil || result.Count(projectcleanup.Deleted) != 3 || len(called) != 6 || !slices.Equal(sent, called) {
		t.Fatalf("result %+v err %v callback %v channel %v", result, err, called, sent)
	}
	// The callback runs before the send of the same event.
	buffered := make(chan projectcleanup.Event, 8)
	var queuedAtCallback []int
	_, err = cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithProgress(buffered),
		projectcleanup.WithProgressCallback(func(projectcleanup.Event) error {
			queuedAtCallback = append(queuedAtCallback, len(buffered))
			return nil
		}))
	if err != nil || !slices.Equal(queuedAtCallback, []int{0, 1, 2, 3, 4, 5}) || len(buffered) != 6 {
		t.Fatalf("err %v queued at callback %v", err, queuedAtCallback)
	}
	if _, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithProgress(nil)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil progress channel: %v", err)
	}
}

func TestProjectCleanupProgressCallbackErrorSendsNothing(t *testing.T) {
	cloud, plan := progressChannelCleanup(t)
	stop := errors.New("stop")
	sent := make(chan projectcleanup.Event, 8)
	result, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithProgress(sent),
		projectcleanup.WithProgressCallback(func(projectcleanup.Event) error { return stop }))
	if !errors.Is(err, stop) || len(sent) != 0 || result.Outcomes[0].State != projectcleanup.Accepted || result.Count(projectcleanup.NotAttempted) != 2 {
		t.Fatalf("err %v sent %d result %+v", err, len(sent), result)
	}
}

func TestProjectCleanupProgressChannelBlockedSendStopsOnCancellation(t *testing.T) {
	cloud, plan := progressChannelCleanup(t)
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	stop := errors.New("stopped while progress blocked")
	// Nobody receives, so the first send blocks until the caller cancels
	// shortly after the first DELETE response.
	base := cloud.Provider.HTTPClient.Transport
	var once sync.Once
	cloud.SetTransport(testcloud.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		response, err := base.RoundTrip(r)
		if r.Method == http.MethodDelete {
			once.Do(func() { time.AfterFunc(20*time.Millisecond, func() { cancel(stop) }) })
		}
		return response, err
	}))
	blocked := make(chan projectcleanup.Event)
	result, err := cloud.conn.ExecuteProjectCleanup(ctx, plan, projectcleanup.WithProgress(blocked))
	select {
	case e := <-blocked:
		t.Fatalf("progress sent after Execute returned: %+v", e)
	default:
	}
	if !errors.Is(err, context.Canceled) || !errors.Is(context.Cause(ctx), stop) || result.Outcomes[0].State != projectcleanup.Accepted || result.Count(projectcleanup.NotAttempted) != 2 {
		t.Fatalf("err %v result %+v", err, result)
	}
	if got := cloud.events(); !slices.Equal(got, []string{"list servers", "list groups", "delete a"}) {
		t.Fatalf("events %v", got)
	}
}
