package blockstorage_test

import (
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestSnapshotAndBackupWaitStrategyRejectedBeforeHTTP(t *testing.T) {
	invalid := resource.LinearWait(time.Millisecond, -time.Millisecond, 0)
	valid := resource.FixedWait(time.Millisecond)
	for name, policy := range map[string]blockstorage.SnapshotMutationWaitOpts{
		"invalid": {WaitStrategy: &invalid},
		"zero":    {WaitStrategy: &resource.WaitStrategy{}},
		"both":    {WaitStrategy: &valid, PollInterval: smwDuration(time.Millisecond)},
	} {
		t.Run(name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			if _, err := smwCreate(t, smwClient(cloud), blockstorage.WithCreateVolumeSnapshotWaitPolicy(policy)); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("snapshot create", err)
			}
			if _, err := smwDelete(t, smwClient(cloud), "name", blockstorage.WithDeleteVolumeSnapshotWaitPolicy(policy)); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("snapshot delete", err)
			}
			if _, err := bmwCreate(t, bmwClient(cloud), blockstorage.WithCreateVolumeBackupWaitPolicy(policy)); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("backup create", err)
			}
			if calls.Load() != 0 {
				t.Fatal("HTTP sent before validation", calls.Load())
			}
		})
	}
}

func TestSnapshotWaitStrategyPacesPolls(t *testing.T) {
	cloud := testcloud.New(t)
	var mu sync.Mutex
	var polls []time.Time
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			testcloud.JSON(w, 202, `{"snapshot":{"id":"first","status":"creating"}}`)
			return
		}
		mu.Lock()
		polls = append(polls, time.Now())
		n := len(polls)
		mu.Unlock()
		status := "creating"
		if n == 4 {
			status = "available"
		}
		testcloud.JSON(w, 200, `{"snapshot":{"status":"`+status+`"}}`)
	})
	ms := time.Millisecond
	strategy := resource.ExponentialWait(5*ms, 2, 0)
	result, err := smwCreate(t, smwClient(cloud), blockstorage.WithCreateVolumeSnapshotWaitPolicy(blockstorage.SnapshotMutationWaitOpts{WaitStrategy: &strategy}))
	if err != nil || result == nil || result.Ready == nil || len(polls) != 4 {
		t.Fatal(result, err, len(polls))
	}
	for i, want := range []time.Duration{5 * ms, 10 * ms, 20 * ms} {
		if gap := polls[i+1].Sub(polls[i]); gap < want {
			t.Fatalf("pause %d = %v, want at least %v", i+1, gap, want)
		}
	}
}
