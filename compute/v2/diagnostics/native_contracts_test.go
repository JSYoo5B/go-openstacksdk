package diagnostics_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/diagnostics"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeDiagnosticsTransport func(*http.Request) (*http.Response, error)

func (transport nativeDiagnosticsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeDiagnosticsWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeDiagnosticsCall struct{ method, path, query string }

func nativeDiagnosticsAPI(t *testing.T, calls *[]nativeDiagnosticsCall, reply func(*http.Request) *http.Response) *diagnostics.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeDiagnosticsTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, nativeDiagnosticsCall{req.Method, req.URL.Path, req.URL.RawQuery})
		return reply(req), nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	return diagnostics.New(client)
}

func nativeDiagnosticsOperation(t *testing.T, err error) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != "Get" || wrapped.Resource != "diagnostics" {
		t.Fatal("generated diagnostics context", err, wrapped)
	}
}

func TestNativeServerDiagnosticsRouteDecodeAndStatuses(t *testing.T) {
	ctx := context.Background()
	var calls []nativeDiagnosticsCall
	// The pre-2.48 driver-specific and 2.48+ normalized shapes both stay an untyped map.
	api := nativeDiagnosticsAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeDiagnosticsWire(200, `{"state":"running","driver":"libvirt","num_cpus":2,"cpu_details":[{"id":0,"time":17300000000}],"vda_read":77824}`)
	})
	got, err := api.Get(ctx, "s-1")
	if err != nil || got["state"] != "running" || got["num_cpus"] != float64(2) || got["vda_read"] != float64(77824) || len(got["cpu_details"].([]any)) != 1 {
		t.Fatal(got, err)
	}
	if !reflect.DeepEqual(calls, []nativeDiagnosticsCall{{http.MethodGet, "/nova/v2.1/servers/s-1/diagnostics", ""}}) {
		t.Fatalf("%+v", calls)
	}
	for body, wantErr := range map[string]bool{`{}`: false, `null`: false, `[]`: true, `"x"`: true} {
		var calls []nativeDiagnosticsCall
		api := nativeDiagnosticsAPI(t, &calls, func(*http.Request) *http.Response { return nativeDiagnosticsWire(200, body) })
		got, err := api.Get(ctx, "s-1")
		switch {
		case wantErr:
			nativeDiagnosticsOperation(t, err)
		case err != nil || len(got) != 0:
			t.Fatal(body, got, err)
		}
	}
	// No OkCodes are set, so the GET default of 200 applies.
	for _, code := range []int{201, 202, 204, 404} {
		t.Run(fmt.Sprintf("Get/%d", code), func(t *testing.T) {
			var calls []nativeDiagnosticsCall
			api := nativeDiagnosticsAPI(t, &calls, func(*http.Request) *http.Response { return nativeDiagnosticsWire(code, `{}`) })
			_, err := api.Get(ctx, "s-1")
			nativeDiagnosticsOperation(t, err)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || len(calls) != 1 {
				t.Fatal(err, native)
			}
		})
	}
}
