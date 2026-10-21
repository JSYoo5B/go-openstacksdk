package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

func TestPyLBAmphoraConfigureSendsBodylessPutAndRequires202(t *testing.T) {
	for _, status := range []int{202, 200, 204, 404} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("PUT /v2/octavia/amphorae/amp-1/config", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if data, _ := io.ReadAll(r.Body); len(data) != 0 {
				t.Errorf("configure body=%q", data)
			}
			w.WriteHeader(status)
		})
		err := newOctaviaAmphorae(t, cloud).Configure(context.Background(), "amp-1")
		if (status == 202) != (err == nil) || calls.Load() != 1 {
			t.Fatalf("status %d calls=%d error=%v", status, calls.Load(), err)
		}
		if status != 202 {
			octaviaAmphoraUnexpected(t, err, status)
		}
	}
}

func TestPyLBAmphoraConfigureCanceledContextSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1) })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := newOctaviaAmphorae(t, cloud).Configure(ctx, "amp-1"); !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
}
