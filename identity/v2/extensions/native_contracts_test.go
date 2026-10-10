package extensions_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeV2ExtensionTransport func(*http.Request) (*http.Response, error)

func (transport nativeV2ExtensionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeV2ExtensionCall struct{ method, path, query, body string }

func nativeV2ExtensionAPI(t *testing.T, calls *[]nativeV2ExtensionCall, reply func(*http.Request) (int, string)) *extensions.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeV2ExtensionTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeV2ExtensionCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v2.0/"
	return extensions.New(client)
}

const nativeV2ExtensionRow = `{"alias":"OS-KSADM","name":"OpenStack Keystone Admin","namespace":"http://docs.openstack.org/identity/api/ext/OS-KSADM/v1.0","updated":"2013-07-11T17:14:00-00:00","description":"admin","links":[{"rel":"describedby","href":"https://example.invalid"}]}`

func TestNativeV2ExtensionRoutesValuesEnvelopeAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeV2ExtensionCall
	api := nativeV2ExtensionAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path == "/keystone/v2.0/extensions" {
			// Keystone v2 nests rows under extensions.values; a next link is never followed.
			return 200, `{"extensions":{"values":[` + nativeV2ExtensionRow + `,{"alias":"OS-EC2"}],"links":{"next":"/never"}},"links":{"next":"/never"}}`
		}
		return 200, `{"extension":` + nativeV2ExtensionRow + `}`
	})
	got, err := api.Get(ctx, "OS-KSADM")
	if err != nil || !(got.Alias == "OS-KSADM" && got.Updated == "2013-07-11T17:14:00-00:00" && got.Namespace != "" && len(got.Links) == 1) {
		t.Fatal(got, err)
	}
	var aliases []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		aliases = append(aliases, value.Alias)
	}
	want := []nativeV2ExtensionCall{
		{http.MethodGet, "/keystone/v2.0/extensions/OS-KSADM", "", ""},
		{http.MethodGet, "/keystone/v2.0/extensions", "", ""},
	}
	if !reflect.DeepEqual(aliases, []string{"OS-KSADM", "OS-EC2"}) || !reflect.DeepEqual(calls, want) {
		t.Fatalf("%v %+v", aliases, calls)
	}
}

func TestNativeV2ExtensionStatusesAndListEnvelope(t *testing.T) {
	ctx := context.Background()
	for _, code := range []int{201, 202, 204, 404} {
		t.Run(fmt.Sprintf("Get/%d", code), func(t *testing.T) {
			var calls []nativeV2ExtensionCall
			api := nativeV2ExtensionAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
			_, err := api.Get(ctx, "OS-KSADM")
			var wrapped *resource.OperationError
			if !errors.As(err, &wrapped) || wrapped.Operation != "Get" || wrapped.Resource != "extensions" {
				t.Fatal(err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || len(calls) != 1 {
				t.Fatal(err, native)
			}
		})
	}
	t.Run("get envelope decode", func(t *testing.T) {
		for body, wantNil := range map[string]bool{`{}`: true, `{"extension":null}`: true, `{"extension":{"alias":"x"}}`: false} {
			var calls []nativeV2ExtensionCall
			api := nativeV2ExtensionAPI(t, &calls, func(*http.Request) (int, string) { return 200, body })
			got, err := api.Get(ctx, "x")
			if err != nil || (got == nil) != wantNil {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("list pager status, envelopes and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{
			{404, `{}`},
			{200, `{"extensions":{"values":[]}}`},
			{200, `{}`},
			// The common flat array envelope is rejected by the v2 values decoder.
			{200, `{"extensions":[{"alias":"OS-KSADM"}]}`},
			{204, ""},
		} {
			var calls []nativeV2ExtensionCall
			api := nativeV2ExtensionAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, tc.body })
			var errs []error
			rows := 0
			for value, err := range api.List(ctx) {
				if err != nil {
					errs = append(errs, err)
				} else if value != nil {
					rows++
				}
			}
			var native gophercloud.ErrUnexpectedResponseCode
			var decode *json.UnmarshalTypeError
			switch {
			case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
			case tc.code == 200 && strings.HasPrefix(tc.body, `{"extensions":[`) && len(errs) == 1 && errors.As(errs[0], &decode) && rows == 0:
			case tc.code == 200 && !strings.HasPrefix(tc.body, `{"extensions":[`) && len(errs) == 0 && rows == 0:
			case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
			default:
				t.Fatal(tc.code, tc.body, errs, rows)
			}
			if len(calls) != 1 {
				t.Fatal(calls)
			}
		}
	})
}
