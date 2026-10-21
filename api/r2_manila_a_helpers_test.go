package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

func r2ManilaAClient(cloud *testcloud.Cloud, version string) *gophercloud.ServiceClient {
	client := cloud.Client("sharev2", "/v2")
	client.Microversion = version
	return client
}

func r2ManilaARead(t *testing.T, r *http.Request) string {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
	}
	return string(data)
}

func r2ManilaAVersion(t *testing.T, r *http.Request, want string) {
	t.Helper()
	if got := r.Header.Get("X-OpenStack-Manila-API-Version"); got != want {
		t.Errorf("microversion header = %q, want %q", got, want)
	}
}

func r2ManilaACode(t *testing.T, err error, want int) {
	t.Helper()
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != want {
		t.Fatalf("err = %v, want native HTTP %d", err, want)
	}
}

// r2ManilaAGuard fails the test when anything reaches the fixture server.
func r2ManilaAGuard(t *testing.T, cloud *testcloud.Cloud) *atomic.Int32 {
	t.Helper()
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Errorf("unexpected request %s %s", r.Method, r.URL)
		testcloud.JSON(w, 500, `{}`)
	})
	return &calls
}

func r2ManilaACanceled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
