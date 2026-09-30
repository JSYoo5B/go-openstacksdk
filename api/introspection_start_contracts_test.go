package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/baremetalintrospection/v1/introspection"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

func TestIntrospectionStartPreservesBootQueryAndEscapedExtensions(t *testing.T) {
	bootFalse, bootTrue := false, true
	for _, tc := range []struct {
		name    string
		options introspection.StartOpts
		extra   []introspection.StartIntrospectionOption
		boot    string
	}{
		{name: "nil follows server default"},
		{name: "explicit false", options: introspection.StartOpts{ManageBoot: &bootFalse}, boot: "false"},
		{name: "explicit true", options: introspection.StartOpts{ManageBoot: &bootTrue}, boot: "true"},
		{name: "typed option replaces initial options", options: introspection.StartOpts{ManageBoot: &bootTrue}, extra: []introspection.StartIntrospectionOption{introspection.WithStartIntrospectionOptions(introspection.StartOpts{ManageBoot: &bootFalse})}, boot: "false"},
		{name: "extension query retains override policy", options: introspection.StartOpts{ManageBoot: &bootTrue}, extra: []introspection.StartIntrospectionOption{introspection.WithStartIntrospectionQuery("manage_boot", "false")}, boot: "false"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			wantQuery := url.Values{"vendor key": {"a&b +/한글"}}
			if tc.boot != "" {
				wantQuery.Set("manage_boot", tc.boot)
			}
			cloud.Mux.HandleFunc("/v1/introspection/node", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				body, err := io.ReadAll(r.Body)
				if r.Method != http.MethodPost || r.URL.RawQuery != wantQuery.Encode() || len(body) != 0 || err != nil {
					t.Errorf("method=%s query=%q body=%q err=%v; want query=%q", r.Method, r.URL.RawQuery, body, err, wantQuery.Encode())
				}
				w.Header().Set("X-Openstack-Request-Id", "req-start")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write([]byte("accepted without a JSON response"))
			})
			client := cloud.Client("baremetal-introspection", "/unused-endpoint")
			// The helper must retain native ServiceURL's ResourceBase precedence.
			client.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + "/v1")
			api := introspection.New(client)
			options := append([]introspection.StartIntrospectionOption{introspection.WithStartIntrospectionQuery("vendor key", "a&b +/한글")}, tc.extra...)
			if err := api.StartIntrospection(context.Background(), "node", tc.options, options...); err != nil || calls.Load() != 1 {
				t.Fatalf("calls=%d err=%v", calls.Load(), err)
			}
		})
	}
}

func TestIntrospectionStartRejectsQueryAndCapabilityErrorsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusAccepted)
	})
	api := introspection.New(cloud.Client("baremetal-introspection", "/v1"))
	sentinel := errors.New("query policy rejected input")
	for _, tc := range []struct {
		name   string
		option introspection.StartIntrospectionOption
		cause  error
	}{
		{"empty query key", introspection.WithStartIntrospectionQuery(" ", "value"), resource.ErrInvalidOption},
		{"nil option", nil, resource.ErrInvalidOption},
		{"query option error", func(*request.Config[introspection.StartOpts]) error { return sentinel }, sentinel},
		{"unsupported body field", request.WithField[introspection.StartOpts]("vendor", "value"), resource.ErrInvalidOption},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := api.StartIntrospection(context.Background(), "node", introspection.StartOpts{}, tc.option)
			var operation *resource.OperationError
			if !errors.Is(err, tc.cause) || !errors.As(err, &operation) || operation.Operation != "StartIntrospection" || calls.Load() != 0 {
				t.Fatalf("calls=%d operation=%+v err=%v", calls.Load(), operation, err)
			}
		})
	}
}

func TestIntrospectionStartRequires202AndPreservesHTTPCause(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusConflict, http.StatusOK, http.StatusCreated, http.StatusNoContent} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/v1/introspection/node", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Query().Get("vendor") != "a&b" {
					t.Errorf("method=%s query=%s", r.Method, r.URL.RawQuery)
				}
				w.Header().Set("X-Openstack-Request-Id", "req-native-error")
				testcloud.JSON(w, status, `{"error":{"message":"introspection rejected"}}`)
			})
			api := introspection.New(cloud.Client("baremetal-introspection", "/v1"))
			err := api.StartIntrospection(context.Background(), "node", introspection.StartOpts{}, introspection.WithStartIntrospectionQuery("vendor", "a&b"))
			var response gophercloud.ErrUnexpectedResponseCode
			var operation *resource.OperationError
			if !errors.As(err, &response) || response.Actual != status || response.ResponseHeader.Get("X-Openstack-Request-Id") != "req-native-error" || !errors.As(err, &operation) || operation.Operation != "StartIntrospection" {
				t.Fatalf("response=%+v operation=%+v err=%v", response, operation, err)
			}
			if status != http.StatusNoContent && string(response.Body) != `{"error":{"message":"introspection rejected"}}` {
				t.Fatalf("original response body lost: %q", response.Body)
			}
			parsed, err := url.Parse(response.URL)
			if err != nil || parsed.Path != "/v1/introspection/node" || parsed.Query().Get("vendor") != "a&b" {
				t.Fatalf("native error URL=%q parse error=%v", response.URL, err)
			}
		})
	}
}

func TestIntrospectionStartPreservesCanceledContext(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusAccepted)
	})
	api := introspection.New(cloud.Client("baremetal-introspection", "/v1"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := api.StartIntrospection(ctx, "node", introspection.StartOpts{}); !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatalf("calls=%d err=%v", calls.Load(), err)
	}
}
