package blockstorage_test

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestDeleteVolumeBackupWaitStrategy(t *testing.T) {
	cloud := testcloud.New(t)
	var mu sync.Mutex
	var deleted bool
	var polls []time.Time
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodDelete:
			deleted = true
			w.WriteHeader(202)
		case !deleted:
			testcloud.JSON(w, 200, `{"backup":{"id":"actual","status":"available"}}`)
		default:
			polls = append(polls, time.Now())
			if len(polls) < 4 {
				testcloud.JSON(w, 200, `{"backup":{"id":"actual","status":"deleting"}}`)
				return
			}
			testcloud.JSON(w, 404, `{"itemNotFound":{"message":"gone"}}`)
		}
	})
	ms := time.Millisecond
	growing := resource.ExponentialWait(2*ms, 2, 0)
	result, err := bmwDelete(t, bmwClient(cloud), blockstorage.WithDeleteVolumeBackupWait(true), blockstorage.WithDeleteVolumeBackupWaitOptions(resource.WithPollInterval(time.Minute), resource.WithWaitStrategy(growing)))
	if err != nil || result == nil || len(polls) != 4 {
		t.Fatal(result, err, len(polls))
	}
	for i, want := range []time.Duration{2 * ms, 4 * ms, 8 * ms} {
		if gap := polls[i+1].Sub(polls[i]); gap < want {
			t.Fatalf("pause %d = %v, want at least %v", i+1, gap, want)
		}
	}
}
