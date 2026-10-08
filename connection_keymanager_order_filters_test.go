package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionKeyManagerOrderFiltersSharedFacadeAndDistinctReferences(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, lookups atomic.Int32
	cloud.Provider.EndpointLocator = func(options gophercloud.EndpointOpts) (string, error) {
		lookups.Add(1)
		if options.Type != "key-manager" || options.Region != "region" || options.Availability != gophercloud.AvailabilityInternal {
			t.Error("catalog selection", options)
		}
		return cloud.Server.URL + "/catalog/barbican/v1/", nil
	}
	const stamp = "2026-10-02T12:30:00.123456"
	const prefix = "/reverse/barbican/v1/orders"
	const meta = `{"name":"native-meta","bit_length":256,"vendor":{"active":false,"ordered":[1,9007199254740993]}}`
	const references = `"order_ref":"https://passive.invalid/orders/order-choice?ignored=1#fragment","secret_ref":"https://passive.invalid/secrets/secret-choice"`
	cloud.Mux.HandleFunc("GET "+prefix, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := url.Values{"limit": {"2"}, "marker": {"first", "second"}}
		if r.URL.Query().Get("offset") != "" {
			want["offset"] = []string{"2"}
		}
		if !reflect.DeepEqual(r.URL.Query(), want) || r.Header.Get("X-SDK-Source") != "shared" || r.Header.Get("Accept") != "application/json" {
			t.Error("local properties leaked into query or shared configuration changed", r.URL, r.Header)
		}
		if r.URL.Query().Get("offset") == "" {
			if r.Header.Get("X-Auth-Token") != "first-token" {
				t.Error(r.Header)
			}
			cloud.Provider.SetToken("second-token")
			testcloud.JSON(w, 200, `{"orders":[{"id":null,"name":"other",`+references+`,"created":"`+stamp+`","type":"key","meta":`+meta+`},{"name":"desired",`+references+`,"created":"`+stamp+`","type":"key","meta":`+meta+`}],"next":"`+cloud.Server.URL+prefix+`?limit=2&marker=first&marker=second&offset=2"}`)
			return
		}
		if r.Header.Get("X-Auth-Token") != "second-token" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"orders":[{"id":null,"name":"desired",`+references+`,"created":"`+stamp+`","type":"key","meta":`+meta+`}]}`)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithRegion("region"), sdk.WithInterface(gophercloud.AvailabilityInternal))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.KeyManagerV1(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	again, err := conn.KeyManager(context.Background())
	if err != nil || again != service || lookups.Load() != 1 || service.Orders.RawClient() != service.RawClient() {
		t.Fatal("shared facade was replaced", again, err, lookups.Load())
	}
	client := service.RawClient()
	client.ResourceBase = cloud.Server.URL + "/reverse/barbican/v1/"
	client.MoreHeaders = map[string]string{"X-SDK-Source": "shared"}
	cloud.Provider.SetToken("first-token")
	values, err := service.Orders.Resources.All(context.Background(), resource.WithFilters(map[string]any{
		"limit": json.Number("2"), "marker": []string{"first", "second"},
		"name": "desired", "created_at": stamp, "id": nil, "order_id": "order-choice", "secret_id": "secret-choice", "type": "key",
		"meta": json.RawMessage(meta), "offset": make(chan int),
	}))
	if err != nil || len(values) != 1 || values[0].Type != "key" || values[0].Created.IsZero() || values[0].Meta.Name != "native-meta" || values[0].Meta.BitLength != 256 || calls.Load() != 2 || lookups.Load() != 1 {
		t.Fatal("top-level name, original metadata or distinct references changed", values, err, calls.Load(), lookups.Load())
	}
	if len(client.MoreHeaders) != 1 || client.MoreHeaders["X-SDK-Source"] != "shared" || client.ProviderClient != cloud.Provider {
		t.Fatal("semantic list changed shared configuration")
	}
}

func TestConnectionKeyManagerOrderFiltersPreflightAndRawNameNamespace(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /barbican/v1/orders", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if !reflect.DeepEqual(r.URL.Query(), url.Values{"name": {"raw-extension"}, "offset": {"4"}}) {
			t.Error("raw name/offset was classified as a semantic field", r.URL)
		}
		testcloud.JSON(w, 200, `{"orders":[{"name":"server-returned","order_ref":"https://passive.invalid/orders/native","secret_ref":"https://passive.invalid/secrets/other","meta":{"name":"different-meta"},"status":"ACTIVE"}]}`)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.KeyManager, cloud.Server.URL+"/barbican/v1/"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.KeyManager(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if values, err := service.Orders.Resources.All(ctx, resource.WithFilter("name", "desired")); len(values) != 0 || !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatal("canceled semantic list started HTTP", values, err, calls.Load())
	}
	if values, err := service.Orders.Resources.All(context.Background(), resource.WithFilter("marker", nil), resource.WithQuery("marker", "raw")); len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
		t.Fatal("present null marker lost collision ownership", values, err, calls.Load())
	}
	if values, err := service.Orders.Resources.All(context.Background(), resource.WithName("different-meta")); len(values) != 0 || !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 0 {
		t.Fatal("native Meta.Name was promoted into a top-level name", values, err, calls.Load())
	}
	values, err := service.Orders.Resources.All(context.Background(), resource.WithQuery("name", "raw-extension"), resource.WithQuery("offset", "4"), resource.WithQuery("status", "ignored"))
	if err != nil || len(values) != 1 || values[0].Meta.Name != "different-meta" || calls.Load() != 1 {
		t.Fatal("existing raw query lane changed", values, err, calls.Load())
	}
}
