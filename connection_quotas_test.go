package gophercloudsdk_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	tokens "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	sdk "gophercloudsdk"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestConnectionProjectQuotasSuppliesSeparateIdentityClient(t *testing.T) {
	cloud := testcloud.New(t)
	var lookups, gets atomic.Int32
	cloud.Mux.HandleFunc("/keystone/v3/projects", func(w http.ResponseWriter, r *http.Request) {
		lookups.Add(1)
		if r.URL.Query().Get("name") != "tenant" || r.Header.Get("X-OpenStack-Nova-API-Version") != "" {
			t.Errorf("identity request: %s %v", r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"projects":[{"id":"resolved","name":"tenant"}]}`)
	})
	cloud.Mux.HandleFunc("/nova/v2.1/admin/os-quota-sets/resolved", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if r.Header.Get("X-OpenStack-Nova-API-Version") != "2.56" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"quota_set":{"id":"other-response-id","cores":8}}`)
	})
	c, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/nova/v2.1/admin"), sdk.WithEndpoint(sdk.Identity, cloud.Server.URL+"/keystone/v3"), sdk.WithMicroversion(sdk.Compute, "2.56"))
	if err != nil {
		t.Fatal(err)
	}
	scope, err := c.ProjectQuotas(context.Background(), resource.Name("tenant"))
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		v, err := scope.Get(context.Background())
		if err != nil || v.ProjectID != "resolved" || v.ID != "other-response-id" {
			t.Fatalf("get: %+v/%v", v, err)
		}
	}
	if lookups.Load() != 1 || gets.Load() != 2 {
		t.Fatalf("lookups/gets: %d/%d", lookups.Load(), gets.Load())
	}
}

func TestConnectionProjectQuotasNeedsNoIdentityForIDOrAuthScope(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Provider.EndpointLocator = func(options gophercloud.EndpointOpts) (string, error) {
		t.Errorf("unexpected catalog lookup: %+v", options)
		return "", errors.New("unexpected")
	}
	c, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/nova/v2.1/admin"))
	if err != nil {
		t.Fatal(err)
	}
	scope, err := c.ProjectQuotas(context.Background(), resource.ID("explicit"))
	if err != nil || scope.ProjectID() != "explicit" {
		t.Fatalf("ID scope: %v/%v", scope, err)
	}
	if _, err := c.CurrentProjectQuotas(context.Background()); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	auth := tokens.CreateResult{}
	auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": "current", "name": "tenant"}}}
	auth.Header = http.Header{"X-Subject-Token": []string{"auth-token"}}
	if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
		t.Fatal(err)
	}
	scope, err = c.CurrentProjectQuotas(context.Background())
	if err != nil || scope.ProjectID() != "current" {
		t.Fatalf("current scope: %v/%v", scope, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.ProjectQuotas(ctx, resource.Name("tenant")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := c.CurrentProjectQuotas(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
