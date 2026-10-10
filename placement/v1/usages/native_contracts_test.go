package usages_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/usages"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeUsageTransport func(*http.Request) (*http.Response, error)

func (transport nativeUsageTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeUsageWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeUsageCall struct{ method, path, query, body, version string }

func nativeUsageAPI(t *testing.T, microversion string, calls *[]nativeUsageCall, reply func(*http.Request) *http.Response) *usages.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeUsageTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeUsageCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("OpenStack-API-Version")})
		return reply(req), nil
	})
	client := cloud.Client("placement", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/placement/"
	client.Microversion = microversion
	return usages.New(client)
}

func nativeUsageOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "usages" {
		t.Fatal("generated usages context", err, wrapped)
	}
}

func TestNativeUsagesQueryAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeUsageCall
	api := nativeUsageAPI(t, "1.38", &calls, func(*http.Request) *http.Response {
		return nativeUsageWire(200, `{"usages":{"INSTANCE":{"consumer_count":2,"VCPU":4,"MEMORY_MB":1024},"all":{"consumer_count":2,"VCPU":4}}}`)
	})
	got, err := api.Get(ctx, usages.WithGetOptions(usages.GetOpts{ProjectID: "p-1", UserID: "u-1", ConsumerType: "INSTANCE"}), usages.WithGetQuery("extra", "1"))
	if err != nil || got.Usages["INSTANCE"]["consumer_count"] != 2 || got.Usages["INSTANCE"]["MEMORY_MB"] != 1024 || got.Usages["all"]["VCPU"] != 4 {
		t.Fatal(got, err)
	}
	// ProjectID is documented as required but has no required tag, so an empty query is sent.
	if _, err := api.Get(ctx); err != nil {
		t.Fatal(err)
	}
	want := []nativeUsageCall{
		{http.MethodGet, "/placement/usages", "consumer_type=INSTANCE&extra=1&project_id=p-1&user_id=u-1", "", "placement 1.38"},
		{http.MethodGet, "/placement/usages", "", "", "placement 1.38"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
	t.Run("pre-1.38 flat usages fail the 1.38 decode", func(t *testing.T) {
		var calls []nativeUsageCall
		api := nativeUsageAPI(t, "1.9", &calls, func(*http.Request) *http.Response {
			return nativeUsageWire(200, `{"usages":{"VCPU":2,"MEMORY_MB":512}}`)
		})
		// The facade always uses the native Extract, which expects consumer type groups.
		_, err := api.Get(ctx, usages.WithGetOptions(usages.GetOpts{ProjectID: "p-1"}))
		nativeUsageOperation(t, err, "Get")
		if len(calls) != 1 || calls[0].version != "placement 1.9" || calls[0].query != "project_id=p-1" {
			t.Fatal(calls)
		}
	})
}

func TestNativeUsagesStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, code := range []int{201, 202, 204, 404} {
		t.Run(fmt.Sprintf("Get/%d", code), func(t *testing.T) {
			var calls []nativeUsageCall
			api := nativeUsageAPI(t, "1.38", &calls, func(*http.Request) *http.Response { return nativeUsageWire(code, `{}`) })
			_, err := api.Get(ctx, usages.WithGetOptions(usages.GetOpts{ProjectID: "p-1"}))
			nativeUsageOperation(t, err, "Get")
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || len(calls) != 1 {
				t.Fatal(err, native)
			}
		})
	}
	t.Run("decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			`{}`:            false,
			`null`:          false,
			`{"usages":[]}`: true,
		} {
			var calls []nativeUsageCall
			api := nativeUsageAPI(t, "1.38", &calls, func(*http.Request) *http.Response { return nativeUsageWire(200, body) })
			got, err := api.Get(ctx)
			if wantErr {
				nativeUsageOperation(t, err, "Get")
			} else if err != nil || got == nil || got.Usages != nil {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeUsageCall
		api := nativeUsageAPI(t, "1.38", &calls, func(*http.Request) *http.Response { return nativeUsageWire(200, `{}`) })
		for name, options := range map[string][]usages.GetOption{
			"empty query key": {usages.WithGetQuery(" ", "x")},
			"nil option":      {nil},
		} {
			_, err := api.Get(ctx, options...)
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeUsageOperation(t, err, "Get")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
