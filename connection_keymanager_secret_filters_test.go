package gophercloudsdk_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestConnectionKeyManagerSecretFiltersSharedFacadeAndRawProperties(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, lookups atomic.Int32
	cloud.Provider.EndpointLocator = func(options gophercloud.EndpointOpts) (string, error) {
		lookups.Add(1)
		if options.Type != "key-manager" || options.Region != "region" || options.Availability != gophercloud.AvailabilityInternal {
			t.Error("catalog selection", options)
		}
		return cloud.Server.URL + "/catalog/barbican/v1/", nil
	}
	const stamp = "2026-10-01T12:30:00.123456"
	const prefix = "/reverse/barbican/v1/secrets"
	cloud.Mux.HandleFunc("GET "+prefix, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		want := url.Values{"alg": {"aes", "rsa"}, "bits": {"256"}, "created": {"gte:" + stamp}}
		if r.URL.Query().Get("offset") != "" {
			want["offset"], want["limit"] = []string{"2"}, []string{"2"}
		}
		if !reflect.DeepEqual(r.URL.Query(), want) || r.Header.Get("X-SDK-Source") != "shared" || r.Header.Get("Accept") != "application/json" {
			t.Error("semantic classification or shared client changed", r.URL, r.Header)
		}
		if r.URL.Query().Get("offset") == "" {
			if r.Header.Get("X-Auth-Token") != "first-token" {
				t.Error(r.Header)
			}
			cloud.Provider.SetToken("second-token")
			testcloud.JSON(w, 200, `{"secrets":[{"name":"raw-id-is-full-ref","secret_ref":"https://passive.invalid/secrets/selected","created":"`+stamp+`","bit_length":128,"content_types":{"default":"text/plain"}},{"id":null,"name":"wrong-content-type","secret_ref":"https://passive.invalid/secrets/selected","created":"`+stamp+`","bit_length":128,"content_types":{"default":"application/octet-stream"}}],"next":"`+cloud.Server.URL+prefix+`?alg=aes&alg=rsa&bits=256&created=gte%3A`+stamp+`&offset=2&limit=2"}`)
			return
		}
		if r.Header.Get("X-Auth-Token") != "second-token" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"secrets":[{"id":null,"name":"selected","secret_ref":"https://passive.invalid/secrets/selected","created":"`+stamp+`","bit_length":128,"content_types":{"default":"text/plain","extra":"preserved"}}]}`)
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
	if err != nil || again != service || lookups.Load() != 1 || service.Secrets.RawClient() != service.RawClient() {
		t.Fatal("shared facade was replaced", again, err, lookups.Load())
	}
	client := service.RawClient()
	client.ResourceBase = cloud.Server.URL + "/reverse/barbican/v1/"
	client.MoreHeaders = map[string]string{"X-SDK-Source": "shared"}
	cloud.Provider.SetToken("first-token")
	values, err := service.Secrets.Resources.All(context.Background(), resource.WithFilters(map[string]any{
		"algorithm": []string{"aes", "rsa"}, "alg": "ignored-wire-alias",
		"bits": 256, "created": "gte:" + stamp, "created_at": stamp,
		"content_types": map[string]any{"default": "text/plain"},
		"id":            nil, "secret_id": "selected", "unknown": make(chan int),
	}))
	if err != nil || len(values) != 1 || values[0].Name != "selected" || values[0].BitLength != 128 || values[0].ContentTypes["extra"] != "preserved" || values[0].Created.IsZero() || calls.Load() != 2 || lookups.Load() != 1 {
		t.Fatal("raw local comparison, typed projection or pagination changed", values, err, calls.Load(), lookups.Load())
	}
	if len(client.MoreHeaders) != 1 || client.MoreHeaders["X-SDK-Source"] != "shared" || client.ProviderClient != cloud.Provider {
		t.Fatal("semantic list changed shared configuration")
	}
}

func TestConnectionKeyManagerSecretFiltersPreflightAndNativeQueryLane(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /barbican/v1/secrets", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if !reflect.DeepEqual(r.URL.Query(), url.Values{"algorithm": {"raw-extension"}}) {
			t.Error("raw query was silently treated as a semantic filter", r.URL)
		}
		testcloud.JSON(w, 200, `{"secrets":[{"name":"native","secret_ref":"https://passive.invalid/secrets/native","status":"ACTIVE"}]}`)
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
	if values, err := service.Secrets.Resources.All(ctx, resource.WithFilter("status", "ACTIVE")); len(values) != 0 || !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatal("canceled semantic list started HTTP", values, err, calls.Load())
	}
	if values, err := service.Secrets.Resources.All(context.Background(), resource.WithFilter("algorithm", "aes"), resource.WithQuery("alg", "raw")); len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
		t.Fatal("conflicting query sources started HTTP", values, err, calls.Load())
	}
	values, err := service.Secrets.Resources.All(context.Background(), resource.WithQuery("algorithm", "raw-extension"), resource.WithQuery("status", "ignored-server-query"))
	if err != nil || len(values) != 1 || values[0].Name != "native" || calls.Load() != 1 {
		t.Fatal("existing native query lane changed", values, err, calls.Load())
	}
}
