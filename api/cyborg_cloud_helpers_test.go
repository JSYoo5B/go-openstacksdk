package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

const cyborgCloudProfileUUID = "11111111-1111-4111-8111-111111111111"

// cyborgCloudNoHTTP returns a cloud whose every request fails the test, plus
// the request counter.
func cyborgCloudNoHTTP(t *testing.T) (*testcloud.Cloud, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	return cloud, &calls
}

func cyborgCloudCanceled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

// cyborgCloudHang serves path by blocking until the client gives up.
func cyborgCloudHang(cloud *testcloud.Cloud, pattern string, calls *atomic.Int32) {
	cloud.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	})
}

func cyborgCloudDecode(t *testing.T, r *http.Request) json.RawMessage {
	t.Helper()
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		t.Errorf("request body: %v", err)
	}
	return raw
}

func cyborgCloudIsStatus(err error, status int) bool {
	var unexpected gophercloud.ErrUnexpectedResponseCode
	return errors.As(err, &unexpected) && unexpected.Actual == status
}
