package api_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

const (
	r2IronicBVersion = "1.112"
	r2IronicBInspVer = "1.3"
)

type r2IronicBCall struct {
	Method   string
	URI      string
	Path     string
	RawQuery string
	Body     string
	Header   http.Header
}

type r2IronicBFake struct {
	mu    sync.Mutex
	calls []r2IronicBCall
}

func (f *r2IronicBFake) all() []r2IronicBCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]r2IronicBCall(nil), f.calls...)
}

func (f *r2IronicBFake) only(t *testing.T) r2IronicBCall {
	t.Helper()
	calls := f.all()
	if len(calls) != 1 {
		t.Fatalf("calls=%d: %+v", len(calls), calls)
	}
	return calls[0]
}

// r2IronicBServe answers every request through respond and records it. A body
// of "" answers with only the status line.
func r2IronicBServe(t *testing.T, cloud *testcloud.Cloud, respond func(call r2IronicBCall) (int, string)) *r2IronicBFake {
	t.Helper()
	fake := &r2IronicBFake{}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		call := r2IronicBCall{Method: r.Method, URI: r.RequestURI, Path: r.URL.EscapedPath(), RawQuery: r.URL.RawQuery, Body: string(data), Header: r.Header.Clone()}
		fake.mu.Lock()
		fake.calls = append(fake.calls, call)
		fake.mu.Unlock()
		status, body := respond(call)
		if body == "" {
			w.WriteHeader(status)
			return
		}
		testcloud.JSON(w, status, body)
	})
	return fake
}

// r2IronicBFixed answers every request with one status and body.
func r2IronicBFixed(t *testing.T, cloud *testcloud.Cloud, status int, body string) *r2IronicBFake {
	t.Helper()
	return r2IronicBServe(t, cloud, func(r2IronicBCall) (int, string) { return status, body })
}

func r2IronicBClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("baremetal", "/v1")
	client.Microversion = r2IronicBVersion
	return client
}

func r2IronicBInspectorClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("baremetal-introspection", "/v1")
	client.Microversion = r2IronicBInspVer
	return client
}

func r2IronicBCanceled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func r2IronicBPtr[T any](v T) *T { return &v }

func r2IronicBNoHTTP(t *testing.T, fake *r2IronicBFake) {
	t.Helper()
	if n := len(fake.all()); n != 0 {
		t.Fatalf("unexpected HTTP requests: %+v", fake.all())
	}
}

func r2IronicBContains(t *testing.T, got string, parts ...string) {
	t.Helper()
	for _, p := range parts {
		if !strings.Contains(got, p) {
			t.Fatalf("%q does not contain %q", got, p)
		}
	}
}
