package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	tokens2 "github.com/gophercloud/gophercloud/v2/openstack/identity/v2/tokens"
	tokens3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/loadbalancer/v2/quotas"
	"gophercloudsdk/resource"
)

func TestOctaviaQuotaProjectNameUsesSeparateIdentityClientOnce(t *testing.T) {
	cloud := testcloud.New(t)
	var lookups, gets atomic.Int32
	cloud.Mux.HandleFunc("GET /identity/v3/projects", func(w http.ResponseWriter, r *http.Request) {
		lookups.Add(1)
		if r.URL.Query().Get("name") != "tenant" || r.Header.Get("X-Loadbalancer-Only") != "" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, fmt.Sprintf(`{"projects":[{"id":"near","name":"tenant-copy"}],"links":{"next":%q}}`, cloud.Server.URL+"/identity/v3/page2"))
	})
	cloud.Mux.HandleFunc("GET /identity/v3/page2", func(w http.ResponseWriter, r *http.Request) {
		lookups.Add(1)
		testcloud.JSON(w, 200, `{"projects":[{"id":"project-fixed","name":"tenant"}]}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); testcloud.JSON(w, 200, octaviaQuotaBody) })
	identity := cloud.Client("identity", "/identity/v3")
	api := quotas.New(cloud.Client("load-balancer", "/v2"))
	api.RawClient().MoreHeaders = map[string]string{"X-Loadbalancer-Only": "original"}
	scope, err := api.InProject(context.Background(), resource.Name("tenant"), quotas.WithIdentityClient(identity))
	if err != nil || scope == nil || scope.ProjectID() != "project-fixed" {
		t.Fatal(scope, err)
	}
	for range 2 {
		if _, err := scope.Get(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if gets.Load() != 2 || lookups.Load() != 2 {
		t.Fatal(gets.Load(), lookups.Load())
	}
}

func TestOctaviaQuotaCurrentProjectRecordsAndFreezesV2V3Authentication(t *testing.T) {
	for _, version := range []string{"v2", "v3"} {
		t.Run(version, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets atomic.Int32
			cloud.Mux.HandleFunc("GET /v2/lbaas/quotas/project-fixed", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); testcloud.JSON(w, 200, octaviaQuotaBody) })
			if version == "v2" {
				auth := tokens2.CreateResult{}
				auth.Body = map[string]any{"access": map[string]any{"token": map[string]any{"id": "v2-token", "expires": "2026-10-01T00:00:00.000Z", "tenant": map[string]any{"id": "project-fixed"}}}}
				if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
					t.Fatal(err)
				}
			} else {
				setOctaviaProjectAuth(t, cloud.Provider, "project-fixed")
			}
			api := quotas.New(cloud.Client("load-balancer", "/v2"))
			scope, err := api.CurrentProject(context.Background())
			if err != nil || scope == nil || scope.ProjectID() != "project-fixed" || gets.Load() != 0 {
				t.Fatal(scope, err)
			}
			setOctaviaProjectAuth(t, cloud.Provider, "changed")
			if _, err := scope.Get(context.Background()); err != nil || gets.Load() != 1 {
				t.Fatal(err, gets.Load())
			}
		})
	}
}

func setOctaviaProjectAuth(t *testing.T, provider *gophercloud.ProviderClient, id string) {
	t.Helper()
	auth := tokens3.CreateResult{}
	auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": id}}}
	auth.Header = http.Header{"X-Subject-Token": []string{"project-token"}}
	if err := provider.SetTokenAndAuthResult(auth); err != nil {
		t.Fatal(err)
	}
}

func TestOctaviaQuotaScopePreflightAndLookupFailures(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); http.Error(w, "forbidden", 403) })
	api := quotas.New(cloud.Client("load-balancer", "/v2/project-fixed"))
	if _, err := api.CurrentProject(context.Background()); !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 0 {
		t.Fatal(err)
	}
	for _, id := range []string{"", "bad/path", "defaults", "nul\x00id"} {
		if _, err := api.InProject(context.Background(), resource.ID(id)); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal(id, err)
		}
	}
	for _, option := range []quotas.ProjectOption{nil, quotas.WithIdentityClient(nil), quotas.WithIdentityClient(api.RawClient())} {
		if _, err := api.InProject(context.Background(), resource.Name("tenant"), option); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal(err)
		}
	}
	if _, err := api.InProject(context.Background(), resource.Name("tenant")); !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 0 {
		t.Fatal(err)
	}
	if scope, err := api.InProject(context.Background(), resource.Name("tenant"), quotas.WithIdentityClient(cloud.Client("identity", "/identity/v3"))); scope != nil || !gophercloud.ResponseCodeIs(err, 403) || calls.Load() != 1 {
		t.Fatal(scope, err, calls.Load())
	}
}
