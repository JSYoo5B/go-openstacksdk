package gophercloudsdk_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	sdk "gophercloudsdk"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestConnectionSenlinParentScopesShareLiveAuthVersionAndFixedRoutes(t *testing.T) {
	cloud := testcloud.New(t)
	var parents, policies, attributes atomic.Int32
	cloud.Provider.SetToken("initial-token")
	cloud.Mux.HandleFunc("GET /proxy/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
		ordinal := parents.Add(1)
		token := "initial-token"
		if ordinal > 1 {
			token = "renewed-token"
		}
		if r.Header.Get("X-Auth-Token") != token || r.Header.Get("OpenStack-API-Version") != "clustering 1.2" {
			t.Error(r.Header)
		}
		cloud.Provider.SetToken("renewed-token")
		testcloud.JSON(w, 200, `{"cluster":{"id":"canonical-cluster","name":"selected"}}`)
	})
	cloud.Mux.HandleFunc("GET /proxy/senlin/v1/clusters/canonical-cluster/policies/policy-id", func(w http.ResponseWriter, r *http.Request) {
		policies.Add(1)
		if r.Header.Get("X-Auth-Token") != "renewed-token" || r.Header.Get("OpenStack-API-Version") != "clustering 1.2" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"cluster_policy":{"id":"binding-id","policy_id":"policy-id","cluster_id":"canonical-cluster"}}`)
	})
	cloud.Mux.HandleFunc("GET /proxy/senlin/v1/clusters/canonical-cluster/attrs/details.cpu", func(w http.ResponseWriter, r *http.Request) {
		attributes.Add(1)
		if r.Header.Get("X-Auth-Token") != "renewed-token" || r.Header.Get("OpenStack-API-Version") != "clustering 1.2" || r.URL.RawQuery != "" {
			t.Error(r.Header, r.URL)
		}
		w.Header().Set("X-Request-ID", "attributes-evidence")
		testcloud.JSON(w, 200, `{"cluster_attributes":[{"id":"node-id","value":9007199254740993}]}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Error("scope changed identity or followed an unrelated endpoint", r.Method, r.URL)
		w.WriteHeader(500)
	})
	conn, err := sdk.FromProvider(cloud.Provider,
		sdk.WithEndpoint(sdk.Clustering, cloud.Server.URL+"/proxy/senlin"),
		sdk.WithMicroversion(sdk.Clustering, "1.2"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	service, err := conn.Clustering(ctx)
	if err != nil {
		t.Fatal(err)
	}
	bindings, err := service.ClusterPolicies.InCluster(ctx, resource.ID("selected"))
	if err != nil || bindings.RawClient() != service.RawClient() || bindings.ClusterID() != "canonical-cluster" {
		t.Fatal(bindings, err)
	}
	values, err := service.ClusterAttributes.InCluster(ctx, resource.ID("selected"), " details.cpu ")
	if err != nil || values.RawClient() != service.RawClient() || values.ClusterID() != "canonical-cluster" || values.Path() != "details.cpu" {
		t.Fatal(values, err)
	}
	policy, err := bindings.Get(ctx, "policy-id")
	if err != nil || policy.ID != "binding-id" || policy.PolicyID != "policy-id" {
		t.Fatal(policy, err)
	}
	policy.ClusterID, policy.URIClusterID = "consumer-parent", "consumer-uri"
	rows, err := values.All(ctx)
	if err != nil || len(rows) != 1 || string(rows[0].Value) != "9007199254740993" || rows[0].URIClusterID != "canonical-cluster" || rows[0].Header.Get("X-Request-ID") != "attributes-evidence" {
		t.Fatal(rows, err)
	}
	rows[0].URIClusterID, rows[0].URIPath = "consumer-parent", "consumer-path"
	service.RawClient().Microversion = "1.1"
	if rows, err := values.All(ctx); len(rows) != 0 || !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal("attribute scope did not recheck its version", rows, err)
	}
	service.RawClient().Type = "compute"
	if value, err := bindings.Get(ctx, "policy-id"); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("binding scope did not recheck its service", value, err)
	}
	if parents.Load() != 2 || policies.Load() != 1 || attributes.Load() != 1 || bindings.ClusterID() != "canonical-cluster" || values.Path() != "details.cpu" {
		t.Fatal(parents.Load(), policies.Load(), attributes.Load())
	}
}
