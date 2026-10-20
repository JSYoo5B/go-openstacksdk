package orders_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeOrderTransport func(*http.Request) (*http.Response, error)

func (transport nativeOrderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeOrderWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}, "X-Order-Proof": {"actual"}}}
}

func nativeOrderClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("key-manager", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/barbican/v1/"
	client.MoreHeaders = map[string]string{"X-Source": "direct"}
	return client
}

const nativeOrderRow = `{"created":"2026-10-10T01:02:03","creator_id":"user","error_reason":"","error_status_code":"","meta":{"algorithm":"aes","bit_length":256,"expiration":"2027-01-02T03:04:05","mode":"cbc","name":"n","payload_content_type":"application/octet-stream"},"order_ref":"https://kms/v1/orders/o1","secret_ref":"https://kms/v1/secrets/s1","status":"ACTIVE","sub_status":"Unknown","type":"key","updated":"2026-10-10T01:02:04"}`

func TestNativeOrderRoutesBodiesAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	client := nativeOrderClient(cloud)
	type exchange struct{ method, path, body string }
	var seen []exchange
	cloud.Provider.HTTPClient.Transport = nativeOrderTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		if req.Header.Get("X-Source") != "direct" || req.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(req.Header)
		}
		seen = append(seen, exchange{req.Method, req.URL.Path, raw})
		switch req.Method {
		case http.MethodPost:
			return nativeOrderWire(202, `{"order_ref":"https://kms/v1/orders/o1"}`), nil
		case http.MethodDelete:
			return nativeOrderWire(204, ""), nil
		}
		return nativeOrderWire(200, nativeOrderRow), nil
	})
	api := orders.New(client)
	if api.RawClient() != client {
		t.Fatal("native client identity changed")
	}
	ctx := context.Background()
	got, err := api.Get(ctx, "o1")
	if err != nil || got.OrderRef != "https://kms/v1/orders/o1" || got.Meta.BitLength != 256 || !got.Meta.Expiration.Equal(time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)) || !got.Created.Equal(time.Date(2026, 10, 10, 1, 2, 3, 0, time.UTC)) || got.SubStatus != "Unknown" {
		t.Fatal(got, err)
	}
	expiration := time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)
	created, err := api.Create(ctx, orders.CreateOpts{Type: orders.KeyOrder, Meta: orders.MetaOpts{Algorithm: "aes", BitLength: 256, Mode: "cbc", Expiration: &expiration}}, orders.WithCreateField("x_extension", 1))
	if err != nil || created.OrderRef != "https://kms/v1/orders/o1" {
		t.Fatal(created, err)
	}
	if _, err := api.Create(ctx, orders.CreateOpts{}); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "a/b"); err != nil {
		t.Fatal(err)
	}
	want := []exchange{
		{http.MethodGet, "/barbican/v1/orders/o1", ""},
		{http.MethodPost, "/barbican/v1/orders", `{"meta":{"algorithm":"aes","bit_length":256,"expiration":"2027-01-02T03:04:05","mode":"cbc"},"type":"key","x_extension":1}`},
		// Type and the non-omitempty meta fields are sent even when empty.
		{http.MethodPost, "/barbican/v1/orders", `{"meta":{"algorithm":"","bit_length":0,"mode":""},"type":""}`},
		{http.MethodDelete, "/barbican/v1/orders/a/b", ""},
	}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("%+v", seen)
	}
}

func TestNativeOrderStrictStatusesListAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*orders.API) error
	}{
		{"Get", []int{200}, func(api *orders.API) error { _, err := api.Get(ctx, "o1"); return err }},
		{"Create", []int{202}, func(api *orders.API) error {
			_, err := api.Create(ctx, orders.CreateOpts{Type: orders.KeyOrder})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *orders.API) error { return api.Delete(ctx, "o1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404, 409} {
			if code == call.accepted[0] || len(call.accepted) > 1 && code == call.accepted[1] {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Provider.HTTPClient.Transport = nativeOrderTransport(func(req *http.Request) (*http.Response, error) {
					requests.Add(1)
					return nativeOrderWire(code, nativeOrderRow), nil
				})
				err := call.call(orders.New(nativeOrderClient(cloud)))
				var wrapped *resource.OperationError
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &wrapped) || wrapped.Operation != call.name || wrapped.Resource != "orders" || !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || requests.Load() != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("collision and nil option", func(t *testing.T) {
		cloud := testcloud.New(t)
		var requests atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeOrderTransport(func(req *http.Request) (*http.Response, error) {
			requests.Add(1)
			return nativeOrderWire(202, `{}`), nil
		})
		api := orders.New(nativeOrderClient(cloud))
		for _, option := range []orders.CreateOption{orders.WithCreateField("meta", map[string]any{}), nil} {
			if _, err := api.Create(ctx, orders.CreateOpts{Type: orders.KeyOrder}, option); err == nil {
				t.Fatal("accepted")
			}
		}
		if requests.Load() != 0 {
			t.Fatal(requests.Load())
		}
	})
	t.Run("list query, next link and stop", func(t *testing.T) {
		cloud := testcloud.New(t)
		var queries []url.Values
		var paths []string
		cloud.Provider.HTTPClient.Transport = nativeOrderTransport(func(req *http.Request) (*http.Response, error) {
			queries = append(queries, req.URL.Query())
			paths = append(paths, req.URL.Path)
			if len(paths) == 1 {
				return nativeOrderWire(200, `{"orders":[`+nativeOrderRow+`],"next":"`+cloud.Server.URL+`/other/orders?offset=1"}`), nil
			}
			return nativeOrderWire(200, `{"orders":[]}`), nil
		})
		var refs []string
		for value, err := range orders.New(nativeOrderClient(cloud)).List(ctx, orders.WithListOptions(orders.ListOpts{Limit: 1}), orders.WithListQuery("extra", "1")) {
			if err != nil {
				t.Fatal(err)
			}
			refs = append(refs, value.OrderRef)
		}
		if !reflect.DeepEqual(refs, []string{"https://kms/v1/orders/o1"}) || !reflect.DeepEqual(queries[0], url.Values{"limit": {"1"}, "extra": {"1"}}) || len(paths) != 2 || paths[1] != "/other/orders" {
			t.Fatal(refs, queries, paths)
		}
	})
}
