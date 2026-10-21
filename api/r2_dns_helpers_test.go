package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

func r2dnsClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	return cloud.Client("dns", "/v2")
}

// r2dnsBody reads a request body as raw JSON fields, preserving null.
func r2dnsBody(t *testing.T, r *http.Request) map[string]string {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("read body: %v", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Errorf("body %q: %v", raw, err)
	}
	out := map[string]string{}
	for key, value := range fields {
		out[key] = string(value)
	}
	return out
}

func r2dnsCounter(t *testing.T, cloud *testcloud.Cloud, pattern string, status int, body string) *atomic.Int32 {
	t.Helper()
	var calls atomic.Int32
	cloud.Mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if body == "" {
			w.WriteHeader(status)
			return
		}
		testcloud.JSON(w, status, body)
	})
	return &calls
}
