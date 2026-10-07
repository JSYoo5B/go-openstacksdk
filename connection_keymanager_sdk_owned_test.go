package gophercloudsdk_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/keymanager/v1/quotas"
	"github.com/JSYoo5B/gophercloudsdk/keymanager/v1/secretstores"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionKeyManagerSDKOwnedSharedClientAndFixedProject(t *testing.T) {
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
		if r.Header.Get("X-SDK-Source") != "shared" || r.Header.Get("OpenStack-API-Version") != "" {
			t.Error("shared request policy", r.Header)
		}
		if r.URL.RawQuery != "" {
			t.Error("unexpected query", r.URL)
		}
		w.Header().Set("X-Request-ID", "barbican-response")
		switch r.Method + " " + r.URL.Path {
		case "GET /reverse/barbican/v1/secret-stores/global-default":
			if r.Header.Get("X-Auth-Token") != "store-token" {
				t.Error(r.Header)
			}
			testcloud.JSON(w, 200, `{"name":"global","secret_store_ref":"https://never-follow.test/secret-stores/store-a","global_default":false}`)
		case "GET /reverse/barbican/v1/secret-stores/preferred":
			if r.Header.Get("X-Auth-Token") != "store-token" {
				t.Error(r.Header)
			}
			testcloud.JSON(w, 200, `{"name":"preferred","secret_store_ref":"https://never-follow.test/secret-stores/store-b"}`)
		case "GET /reverse/barbican/v1/secret-stores":
			if r.Header.Get("X-Auth-Token") != "store-token" {
				t.Error(r.Header)
			}
			testcloud.JSON(w, 200, `{"secret_stores":[{"name":"listed","secret_store_ref":"https://never-follow.test/secret-stores/store-c"}]}`)
		case "GET /reverse/barbican/v1/quotas":
			if r.Header.Get("X-Auth-Token") != "effective-token" {
				t.Error(r.Header)
			}
			testcloud.JSON(w, 200, `{"quotas":{"secrets":9007199254740993,"orders":null}}`)
		case "GET /reverse/barbican/v1/project-quotas/project-one":
			if r.Header.Get("X-Auth-Token") != "project-token" {
				t.Error(r.Header)
			}
			testcloud.JSON(w, 200, `{"project_quotas":{"secrets":0,"id":"incidental-project","project_id":"different-project"}}`)
		case "PUT /reverse/barbican/v1/project-quotas/project-one":
			if r.Header.Get("X-Auth-Token") != "replace-token" {
				t.Error(r.Header)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != `{"project_quotas":{"secrets":0}}` {
				t.Error(string(body), err)
			}
			w.WriteHeader(http.StatusNoContent)
		case "DELETE /reverse/barbican/v1/project-quotas/project-one":
			if r.Header.Get("X-Auth-Token") != "delete-token" {
				t.Error(r.Header)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Error("invented lookup, ref follow or changed fixed project", r.Method, r.URL)
			w.WriteHeader(500)
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
	if service.SecretStores.RawClient() != client || service.Quotas.RawClient() != client || service.Secrets.RawClient() != client || client.ProviderClient != cloud.Provider || client.Type != "key-manager" {
		t.Fatal("SDK-owned and native APIs do not share the authenticated client")
	}
	client.ResourceBase = cloud.Server.URL + "/reverse/barbican/v1/"
	client.MoreHeaders = map[string]string{"X-SDK-Source": "shared"}
	project, err := service.Quotas.InProject(context.Background(), resource.ID("project-one"))
	if err != nil || project.RawClient() != client || project.ProjectID() != "project-one" || calls.Load() != 0 {
		t.Fatal(project, err, calls.Load())
	}
	cloud.Provider.SetToken("store-token")
	global, err := service.SecretStores.GetGlobalDefault(context.Background())
	if err != nil || global == nil || global.ID != "store-a" || global.GlobalDefault == nil || *global.GlobalDefault || global.Header.Get("X-Request-ID") != "barbican-response" {
		t.Fatal(global, err)
	}
	preferred, err := service.SecretStores.GetPreferred(context.Background())
	if err != nil || preferred == nil || preferred.ID != "store-b" {
		t.Fatal(preferred, err)
	}
	listed, err := service.SecretStores.All(context.Background(), secretstores.WithListFilter("secret_store_id", "store-c"))
	if err != nil || len(listed) != 1 || listed[0].ID != "store-c" {
		t.Fatal(listed, err)
	}
	cloud.Provider.SetToken("effective-token")
	effective, err := service.Quotas.Get(context.Background())
	if err != nil || effective == nil || string(effective.Secrets) != "9007199254740993" || string(effective.Orders) != "null" {
		t.Fatal(effective, err)
	}
	cloud.Provider.SetToken("project-token")
	configured, err := project.Get(context.Background())
	if err != nil || configured == nil || string(configured.Secrets) != "0" || project.ProjectID() != "project-one" {
		t.Fatal(configured, err)
	}
	cloud.Provider.SetToken("replace-token")
	accepted, err := project.Update(context.Background(), quotas.UpdateOpts{}, quotas.WithUpdateSecrets(0))
	if err != nil || accepted == nil || accepted.StatusCode != 204 || len(accepted.Body) != 0 || accepted.Header.Get("X-Request-ID") != "barbican-response" {
		t.Fatal(accepted, err)
	}
	cloud.Provider.SetToken("delete-token")
	if err := project.Delete(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 7 || lookups.Load() != 1 {
		t.Fatal("unexpected HTTP or catalog work", calls.Load(), lookups.Load())
	}
}

func TestConnectionKeyManagerSDKOwnedCancellationAndStrictMissing(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/barbican/v1/secret-stores/preferred" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 404, `{"error":"preferred is unset"}`)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.KeyManager, cloud.Server.URL+"/barbican/v1/"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.KeyManagerV1(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := conn.KeyManager(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := service.SecretStores.GetGlobalDefault(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := service.Quotas.Get(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := service.Quotas.InProject(ctx, resource.ID("project")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := service.Quotas.InProject(context.Background(), resource.Name("project")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("preflight started HTTP", calls.Load())
	}
	if value, err := service.SecretStores.GetPreferred(context.Background()); value != nil || !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatal(value, err)
	}
	if calls.Load() != 1 {
		t.Fatal("preferred missing caused global fallback", calls.Load())
	}
}
