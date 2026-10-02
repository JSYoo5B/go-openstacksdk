package gophercloudsdk_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestConnectionKeyManagerSecretFindSharedFacadeAndTwoPhaseResults(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, lookups atomic.Int32
	cloud.Provider.EndpointLocator = func(options gophercloud.EndpointOpts) (string, error) {
		lookups.Add(1)
		if options.Type != "key-manager" || options.Region != "region" || options.Availability != gophercloud.AvailabilityInternal {
			t.Errorf("catalog selection: %#v", options)
		}
		return cloud.Server.URL + "/catalog/barbican/v1/", nil
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.Header.Get("X-SDK-Source") != "shared" || r.Header.Get("Accept") != "application/json" && r.Header.Get("Accept") != "text/plain" {
			t.Error(r.Method, r.URL, r.Header)
		}
		switch r.URL.Path {
		case "/reverse/barbican/v1/secrets/friendly":
			if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "metadata-token" {
				t.Error(r.URL, r.Header)
			}
			cloud.Provider.SetToken("payload-token")
			testcloud.JSON(w, 200, `{"name":"different","content_types":{"default":"text/plain"}}`)
		case "/reverse/barbican/v1/secrets/friendly/payload":
			if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "payload-token" || r.Header.Get("Accept") != "text/plain" {
				t.Error(r.URL, r.Header)
			}
			cloud.Provider.SetToken("list-token")
			testcloud.JSON(w, 404, `{"error":"payload missing"}`)
		case "/reverse/barbican/v1/secrets":
			if r.URL.Query().Get("name") != "friendly" || r.Header.Get("X-Auth-Token") != "list-token" {
				t.Error(r.URL, r.Header)
			}
			if r.URL.Query().Get("offset") == "" {
				w.Header().Set("X-Request-ID", "list-one")
				testcloud.JSON(w, 200, `{"secrets":[{"name":"other","secret_ref":"https://passive.invalid/secrets/shared"}],"next":"`+cloud.Server.URL+`/reverse/barbican/v1/secrets?name=friendly&offset=1&limit=1"}`)
				return
			}
			if r.URL.Query().Get("offset") != "1" || r.URL.Query().Get("limit") != "1" {
				t.Error(r.URL)
			}
			w.Header().Set("X-Request-ID", "list-two")
			testcloud.JSON(w, 200, `{"secrets":[{"id":null,"name":"friendly","secret_ref":"https://passive.invalid/secrets/shared","content_types":{"default":"text/plain"},"extension":9007199254740993}]}`)
		case "/reverse/barbican/v1/secrets/native":
			testcloud.JSON(w, 200, `{"name":"native","secret_ref":"https://passive.invalid/secrets/native"}`)
		default:
			t.Error("find performed a final GET, followed a ref, or changed the selected prefix", r.URL)
			w.WriteHeader(http.StatusInternalServerError)
		}
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
	if err != nil || again != service || lookups.Load() != 1 {
		t.Fatal(again, err, lookups.Load())
	}
	client := service.RawClient()
	if service.Secrets.RawClient() != client || client.ProviderClient != cloud.Provider {
		t.Fatal("FindIdentity must share the authenticated Secrets facade")
	}
	client.ResourceBase = cloud.Server.URL + "/reverse/barbican/v1/"
	client.MoreHeaders = map[string]string{"X-SDK-Source": "shared"}
	cloud.Provider.SetToken("metadata-token")
	value, err := service.Secrets.FindIdentity(context.Background(), "friendly", resource.WithIdentityFindIgnoreMissing(false))
	if err != nil || value == nil || value.Name != "friendly" || value.SecretID != "" || value.Payload != nil || string(value.Body["id"]) != "null" || string(value.Body["extension"]) != "9007199254740993" || value.Header.Get("X-Request-ID") != "list-two" || value.StatusCode != 200 || calls.Load() != 4 {
		t.Fatal(value, err, calls.Load())
	}
	native, err := service.Secrets.Get(context.Background(), "native")
	if err != nil || native == nil || native.Name != "native" || calls.Load() != 5 || lookups.Load() != 1 || len(client.MoreHeaders) != 1 || client.MoreHeaders["X-SDK-Source"] != "shared" {
		t.Fatal("native metadata API or shared configuration changed", native, err, calls.Load())
	}
}

func TestConnectionKeyManagerSecretFindCanceledAndStrictAbsence(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.URL.Path {
		case "/barbican/v1/secrets/missing":
			testcloud.JSON(w, 404, `{"error":"missing"}`)
		case "/barbican/v1/secrets":
			if r.URL.Query().Get("name") != "missing" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, `{"secrets":[]}`)
		default:
			t.Error(r.URL)
			w.WriteHeader(500)
		}
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
	if value, err := service.Secrets.FindIdentity(ctx, "missing"); value != nil || !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatal(value, err, calls.Load())
	}
	if value, err := service.Secrets.FindIdentity(context.Background(), "missing", resource.WithIdentityFindIgnoreMissing(false)); value != nil || !errors.Is(err, resource.ErrNotFound) || calls.Load() != 2 {
		t.Fatal(value, err, calls.Load())
	}
	if value, err := service.Secrets.FindIdentity(context.Background(), "missing"); value != nil || err != nil || calls.Load() != 4 {
		t.Fatal(value, err, calls.Load())
	}
}
