package introspection

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

type failedStartQuery struct{ err error }

func (opts failedStartQuery) ToStartIntrospectionQuery() (string, error) { return "", opts.err }

func TestStartHelperRetainsSerializerErrorWithoutHTTPRequest(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusAccepted)
	})
	sentinel := errors.New("serializer failed before HTTP")
	result := startIntrospection(context.Background(), cloud.Client("baremetal-introspection", "/v1"), "node", failedStartQuery{sentinel})
	if !errors.Is(result.ExtractErr(), sentinel) || result.Header != nil || calls.Load() != 0 {
		t.Fatalf("calls=%d header=%v err=%v", calls.Load(), result.Header, result.ExtractErr())
	}
}

func TestStartHelperRetainsAcceptedResponseHeaders(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/v1/introspection/node", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Openstack-Request-Id", "req-start")
		w.WriteHeader(http.StatusAccepted)
	})
	result := startIntrospection(context.Background(), cloud.Client("baremetal-introspection", "/v1"), "node", StartOpts{})
	if result.ExtractErr() != nil || result.Header.Get("X-Openstack-Request-Id") != "req-start" {
		t.Fatalf("header=%v err=%v", result.Header, result.ExtractErr())
	}
}
