package connectiontest_test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/projectcleanup"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// cleanupServerAbsence serves one server that stays DELETING for pending reads
// and records the time of every absence read.
func cleanupServerAbsence(t *testing.T, pending int) (*cleanupCloud, *projectcleanup.Plan, func() []time.Time) {
	t.Helper()
	cloud := newCleanupCloud(t)
	var mu sync.Mutex
	var reads []time.Time
	cloud.Mux.HandleFunc("DELETE /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		cloud.record("delete " + r.PathValue("id"))
		w.WriteHeader(204)
	})
	cloud.Mux.HandleFunc("GET /compute/v2.1/project/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		reads = append(reads, time.Now())
		n := len(reads)
		mu.Unlock()
		if n <= pending {
			testcloud.JSON(w, 200, `{"server":{"id":"s1","status":"DELETING"}}`)
			return
		}
		testcloud.JSON(w, 404, `{"itemNotFound":{"code":404}}`)
	})
	cloud.serveCompute(t, `{"id":"s1"}`, ``)
	plan, err := cloud.conn.PlanProjectCleanup(context.Background(), projectcleanup.WithAllResources())
	if err != nil {
		t.Fatal(err)
	}
	return cloud, plan, func() []time.Time { mu.Lock(); defer mu.Unlock(); return append([]time.Time(nil), reads...) }
}

func cleanupGaps(t *testing.T, reads []time.Time, want ...time.Duration) {
	t.Helper()
	if len(reads) != len(want)+1 {
		t.Fatalf("%d absence reads, want %d", len(reads), len(want)+1)
	}
	for i, d := range want {
		if gap := reads[i+1].Sub(reads[i]); gap < d {
			t.Fatalf("pause %d = %v, want at least %v", i+1, gap, d)
		}
	}
}

func TestProjectCleanupWaitStrategyRejectedBeforeAnyRequest(t *testing.T) {
	for name, option := range map[string]projectcleanup.ExecuteOption{
		"invalid":       projectcleanup.WithWaitStrategy(resource.LinearWait(time.Millisecond, -time.Millisecond, 0)),
		"zero":          projectcleanup.WithWaitStrategy(resource.WaitStrategy{}),
		"zero interval": projectcleanup.WithPollInterval(0),
	} {
		cloud, plan, reads := cleanupServerAbsence(t, 0)
		before := len(cloud.events())
		if _, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(name, err)
		}
		if len(cloud.events()) != before || len(reads()) != 0 {
			t.Fatal(name, "request sent before validation", cloud.events(), len(reads()))
		}
	}
}

func TestProjectCleanupWaitStrategyPacesAbsenceWaits(t *testing.T) {
	ms := time.Millisecond
	t.Run("strategy after interval", func(t *testing.T) {
		cloud, plan, reads := cleanupServerAbsence(t, 3)
		result, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithPollInterval(time.Hour), projectcleanup.WithWaitStrategy(resource.ExponentialWait(2*ms, 2, 0)))
		if err != nil || len(result.Outcomes) != 1 || result.Outcomes[0].State != projectcleanup.Deleted {
			t.Fatal(result, err)
		}
		cleanupGaps(t, reads(), 2*ms, 4*ms, 8*ms)
	})
	t.Run("interval after strategy", func(t *testing.T) {
		cloud, plan, reads := cleanupServerAbsence(t, 2)
		result, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan, projectcleanup.WithWaitStrategy(resource.FixedWait(time.Hour)), projectcleanup.WithPollInterval(5*ms))
		if err != nil || result.Outcomes[0].State != projectcleanup.Deleted {
			t.Fatal(result, err)
		}
		cleanupGaps(t, reads(), 5*ms, 5*ms)
	})
	t.Run("default two seconds", func(t *testing.T) {
		cloud, plan, reads := cleanupServerAbsence(t, 1)
		result, err := cloud.conn.ExecuteProjectCleanup(context.Background(), plan)
		if err != nil || result.Outcomes[0].State != projectcleanup.Deleted {
			t.Fatal(result, err)
		}
		cleanupGaps(t, reads(), 2*time.Second)
	})
}
