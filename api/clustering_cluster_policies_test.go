package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/clustering/v1/clusterpolicies"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const clusterPolicyRow = `{"id":"binding-id","policy_id":"policy-id","cluster_id":"canonical-cluster","cluster_name":"selected","policy_name":"named-policy","policy_type":"senlin.policy.scaling-1.0","enabled":false,"data":{"big":9007199254740993,"nested":{"fraction":1.234567890123456789}},"future":null}`

func clusterPolicyParent(t *testing.T, cloud *testcloud.Cloud) {
	t.Helper()
	cloud.Mux.HandleFunc("GET /senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"cluster":{"id":"canonical-cluster","name":"selected"}}`)
	})
}

func TestClusteringClusterPoliciesCanonicalScopeAndDirectPolicyRoutes(t *testing.T) {
	cloud := testcloud.New(t)
	var parents, named, children atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/clusters/controller-name", func(w http.ResponseWriter, r *http.Request) {
		parents.Add(1)
		testcloud.JSON(w, 200, `{"cluster":{"id":"canonical-cluster","name":"selected"}}`)
	})
	cloud.Mux.HandleFunc("GET /senlin/v1/clusters", func(w http.ResponseWriter, r *http.Request) {
		named.Add(1)
		if r.URL.Query().Get("name") != "selected" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"clusters":[{"id":"other-cluster","name":"other"},{"id":"canonical-cluster","name":"selected"}]}`)
	})
	cloud.Mux.HandleFunc("GET /senlin/v1/clusters/canonical-cluster/policies/{policy}", func(w http.ResponseWriter, r *http.Request) {
		call := children.Add(1)
		identities := []string{"policy-name", "short123", "714fe676-a08f-4196-b7af-61d52eeded15"}
		if r.PathValue("policy") != identities[(call-1)%3] {
			t.Error("direct policy identity changed", r.URL)
		}
		if r.Header.Get("X-Auth-Token") != "test-token" || r.URL.RawQuery != "" {
			t.Error(r.Header, r.URL)
		}
		w.Header().Set("X-Request-ID", "binding-read")
		w.Header().Set("Location", "https://foreign.example/no-follow")
		testcloud.JSON(w, 200, `{"cluster_policy":`+clusterPolicyRow+`}`)
	})
	api := clusterpolicies.New(cloud.Client("clustering", "/senlin/v1"))
	for _, parent := range []resource.Ref{resource.ID("controller-name"), resource.Name("selected")} {
		scope, err := api.InCluster(context.Background(), parent)
		if err != nil || scope.ClusterID() != "canonical-cluster" || scope.RawClient() != api.RawClient() {
			t.Fatal(scope, err)
		}
		for _, identity := range []string{"policy-name", "short123", "714fe676-a08f-4196-b7af-61d52eeded15"} {
			value, err := scope.Get(context.Background(), identity)
			if err != nil || value.ID != "binding-id" || value.PolicyID != "policy-id" || value.ID == value.PolicyID || value.ClusterID != "canonical-cluster" || value.URIClusterID != scope.ClusterID() || value.IsEnabled || string(value.Data["big"]) != "9007199254740993" || string(value.Body["future"]) != "null" || value.StatusCode != 200 || value.Header.Get("X-Request-ID") != "binding-read" {
				t.Fatal(value, err)
			}
			value.ID, value.PolicyID, value.ClusterID, value.URIClusterID = "consumer-binding", "consumer-policy", "consumer-parent", "consumer-uri"
			value.Header.Set("X-Request-ID", "consumer")
		}
		if scope.ClusterID() != "canonical-cluster" {
			t.Fatal("returned model changed the fixed parent")
		}
	}
	if parents.Load() != 1 || named.Load() != 1 || children.Load() != 6 {
		t.Fatal("parent was refetched or direct policy identity was resolved", parents.Load(), named.Load(), children.Load())
	}
}

func TestClusteringClusterPoliciesWrongCodesRemainNative(t *testing.T) {
	for _, operation := range []string{"Get", "List"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			clusterPolicyParent(t, cloud)
			var calls atomic.Int32
			body := `{"error":"unexpected accepted read"}`
			handler := func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Request-ID", "native-read-evidence")
				testcloud.JSON(w, 202, body)
			}
			cloud.Mux.HandleFunc("GET /senlin/v1/clusters/canonical-cluster/policies", handler)
			cloud.Mux.HandleFunc("GET /senlin/v1/clusters/canonical-cluster/policies/policy-name", handler)
			scope, err := clusterpolicies.New(cloud.Client("clustering", "/senlin/v1")).InCluster(context.Background(), resource.ID("selected"))
			if err != nil {
				t.Fatal(err)
			}
			if operation == "Get" {
				_, err = scope.Get(context.Background(), "policy-name")
			} else {
				_, err = scope.All(context.Background())
			}
			var native gophercloud.ErrUnexpectedResponseCode
			var accepted *resource.ResponseError
			if !errors.As(err, &native) || !gophercloud.ResponseCodeIs(err, 202) || string(native.Body) != body || native.ResponseHeader.Get("X-Request-ID") != "native-read-evidence" || errors.As(err, &accepted) || calls.Load() != 1 {
				t.Fatal(err, calls.Load())
			}
		})
	}
}

func TestClusteringClusterPoliciesExactBodyIdentityOwnsRoutes(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"cluster":{"id":"canonical-cluster","ID":"extension-parent"}}`)
	})
	var reads atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/clusters/canonical-cluster/policies/policy-id", func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		testcloud.JSON(w, 200, `{"cluster_policy":{"id":"binding-id","ID":"extension-binding","policy_id":"policy-id","POLICY_ID":"extension-policy","cluster_id":"canonical-cluster","CLUSTER_ID":"extension-parent"}}`)
	})
	scope, err := clusterpolicies.New(cloud.Client("clustering", "/senlin/v1")).InCluster(context.Background(), resource.ID("selected"))
	if err != nil || scope.ClusterID() != "canonical-cluster" {
		t.Fatal(scope, err)
	}
	value, err := scope.Get(context.Background(), "policy-id")
	if err != nil || value.ID != "binding-id" || value.PolicyID != "policy-id" || value.ClusterID != scope.ClusterID() || string(value.Body["POLICY_ID"]) != `"extension-policy"` {
		t.Fatal(value, err)
	}
	value, err = scope.Resources.Wait(context.Background(), resource.ID(value.PolicyID), "policy-id", resource.WithStatusAttribute("policy_id"), resource.WithFailureStates())
	if err != nil || value.PolicyID != "policy-id" || reads.Load() != 2 {
		t.Fatal(value, err, reads.Load())
	}
}

func TestClusteringClusterPoliciesExactNameIgnoresCaseFoldedExtensions(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /senlin/v1/clusters", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"clusters":[{"id":"other-cluster","name":"other","NAME":"selected"}]}`)
	})
	clusterPolicyParent(t, cloud)
	cloud.Mux.HandleFunc("GET /senlin/v1/clusters/canonical-cluster/policies", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"cluster_policies":[{"id":"binding-id","policy_id":"other-policy","cluster_id":"canonical-cluster","policy_name":"other","POLICY_NAME":"selected"}]}`)
	})
	api := clusterpolicies.New(cloud.Client("clustering", "/senlin/v1"))
	if scope, err := api.InCluster(context.Background(), resource.Name("selected")); scope != nil || !errors.Is(err, resource.ErrNotFound) {
		t.Fatal("extension name selected another cluster", scope, err)
	}
	scope, err := api.InCluster(context.Background(), resource.ID("selected"))
	if err != nil {
		t.Fatal(err)
	}
	if value, err := scope.Resources.Find(context.Background(), resource.Name("selected")); value != nil || !errors.Is(err, resource.ErrNotFound) {
		t.Fatal("extension name selected another policy", value, err)
	}
}

func TestClusteringClusterPoliciesParentIdentityEvidenceAndSourceRecheck(t *testing.T) {
	for _, body := range []string{`{"cluster":{}}`, `{"cluster":{"id":null}}`, `{"cluster":{"id":"bad/id"}}`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Request-ID", "parent-evidence")
				testcloud.JSON(w, 200, body)
			})
			scope, err := clusterpolicies.New(cloud.Client("clustering", "/senlin/v1")).InCluster(context.Background(), resource.ID("selected"))
			var evidence *resource.ResponseError
			if scope != nil || !errors.As(err, &evidence) || evidence.StatusCode != 200 || string(evidence.Body) != body || evidence.Header.Get("X-Request-ID") != "parent-evidence" || calls.Load() != 1 {
				t.Fatal(scope, err, evidence, calls.Load())
			}
		})
	}
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/senlin/v1")
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		client.Type = "compute"
		testcloud.JSON(w, 200, `{"cluster":{"id":"canonical-cluster","name":"selected"}}`)
	})
	if scope, err := clusterpolicies.New(client).InCluster(context.Background(), resource.ID("selected")); scope != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
		t.Fatal(scope, err, calls.Load())
	}
}

func TestClusteringClusterPoliciesRequiredRowsRetainWholeResponse(t *testing.T) {
	for _, operation := range []string{"Get", "List"} {
		for _, invalid := range []string{"missing-id", "null-id", "missing-policy", "null-policy", "missing-parent", "null-parent", "wrong-parent", "bad-id", "wrong-type"} {
			t.Run(operation+"/"+invalid, func(t *testing.T) {
				cloud := testcloud.New(t)
				clusterPolicyParent(t, cloud)
				row := map[string]any{"id": "binding-id", "policy_id": "policy-id", "cluster_id": "canonical-cluster"}
				switch invalid {
				case "missing-id":
					delete(row, "id")
				case "null-id":
					row["id"] = nil
				case "missing-policy":
					delete(row, "policy_id")
				case "null-policy":
					row["policy_id"] = nil
				case "missing-parent":
					delete(row, "cluster_id")
				case "null-parent":
					row["cluster_id"] = nil
				case "wrong-parent":
					row["cluster_id"] = "foreign-cluster"
				case "bad-id":
					row["id"] = "bad/id"
				case "wrong-type":
					row["policy_id"] = true
				}
				raw, _ := json.Marshal(row)
				body := `{"cluster_policy":` + string(raw) + `}`
				if operation == "List" {
					body = `{"cluster_policies":[` + string(raw) + `],"extra":true}`
				}
				var calls atomic.Int32
				handler := func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("X-Request-ID", "row-evidence")
					testcloud.JSON(w, 200, body)
				}
				cloud.Mux.HandleFunc("GET /senlin/v1/clusters/canonical-cluster/policies", handler)
				cloud.Mux.HandleFunc("GET /senlin/v1/clusters/canonical-cluster/policies/policy-name", handler)
				scope, err := clusterpolicies.New(cloud.Client("clustering", "/senlin/v1")).InCluster(context.Background(), resource.ID("selected"))
				if err != nil {
					t.Fatal(err)
				}
				if operation == "Get" {
					_, err = scope.Get(context.Background(), "policy-name")
				} else {
					_, err = scope.All(context.Background())
				}
				var evidence *resource.ResponseError
				if !errors.As(err, &evidence) || string(evidence.Body) != body || evidence.StatusCode != 200 || evidence.Header.Get("X-Request-ID") != "row-evidence" || calls.Load() != 1 {
					t.Fatal(err, evidence, calls.Load())
				}
			})
		}
	}
}

func TestClusteringClusterPoliciesLazyLinkedPagesSnapshotsAndNoFallback(t *testing.T) {
	cloud := testcloud.New(t)
	clusterPolicyParent(t, cloud)
	var lists atomic.Int32
	var captured *request.Config[clusterpolicies.ListOpts]
	cloud.Mux.HandleFunc("GET /senlin/v1/clusters/canonical-cluster/policies", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		query := r.URL.Query()
		if query.Get("enabled") != "false" || query.Get("policy_name") != "named-policy" || query.Get("policy_type") != "vendor.policy-1.0" || query.Get("sort") != "vendor_order:desc" || query.Get("vendor") != "retained" || query.Has("limit") {
			t.Error(query)
		}
		*captured.Options.Enabled = true
		captured.Options.PolicyName = "changed"
		captured.Query.Set("vendor", "changed")
		if query.Get("marker") == "" {
			w.Header().Set("Link", `</senlin/v1/clusters/canonical-cluster/policies?marker=opaque-server-token>; rel="next"`)
		} else if query.Get("marker") != "opaque-server-token" {
			t.Error("invented a marker from response IDs", query)
		}
		testcloud.JSON(w, 200, `{"cluster_policies":[`+clusterPolicyRow+`]}`)
	})
	scope, err := clusterpolicies.New(cloud.Client("clustering", "/senlin/v1")).InCluster(context.Background(), resource.ID("selected"))
	if err != nil {
		t.Fatal(err)
	}
	enabled := false
	option := clusterpolicies.WithListOptions(clusterpolicies.ListOpts{Enabled: &enabled, PolicyName: "named-policy", PolicyType: "vendor.policy-1.0", Sort: "vendor_order:desc"})
	enabled = true
	capture := func(config *request.Config[clusterpolicies.ListOpts]) error { captured = config; return nil }
	iterator := scope.List(context.Background(), option, clusterpolicies.WithListQuery("vendor", "retained"), capture)
	if lists.Load() != 0 {
		t.Fatal("list was eager")
	}
	for range 2 {
		seen := 0
		for row, err := range iterator {
			if err != nil {
				t.Fatal(err)
			}
			seen++
			row.ID, row.PolicyID, row.URIClusterID = "consumer", "consumer-policy", "consumer-parent"
		}
		if seen != 2 {
			t.Fatal(seen)
		}
	}
	if lists.Load() != 4 {
		t.Fatal("unlinked page triggered an invented marker fallback", lists.Load())
	}
	for _, err := range iterator {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if lists.Load() != 5 {
		t.Fatal("break fetched another page", lists.Load())
	}
}

func TestClusteringClusterPoliciesExplicitLinksKeepParentAndSource(t *testing.T) {
	for _, mode := range []string{"wrong-parent", "changed-source", "native-error"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			clusterPolicyParent(t, cloud)
			client := cloud.Client("clustering", "/senlin/v1")
			var pages atomic.Int32
			body := `{"cluster_policies":[` + clusterPolicyRow + `]}`
			cloud.Mux.HandleFunc("GET /senlin/v1/clusters/canonical-cluster/policies", func(w http.ResponseWriter, r *http.Request) {
				pages.Add(1)
				if r.URL.Query().Get("marker") != "" {
					testcloud.JSON(w, 403, `{"error":"late forbidden"}`)
					return
				}
				parent := "canonical-cluster"
				if mode == "wrong-parent" {
					parent = "foreign-cluster"
				}
				w.Header().Set("Link", fmt.Sprintf(`</senlin/v1/clusters/%s/policies?marker=next>; rel="next"`, parent))
				w.Header().Set("X-Request-ID", "page-evidence")
				testcloud.JSON(w, 200, body)
			})
			scope, err := clusterpolicies.New(client).InCluster(context.Background(), resource.ID("selected"))
			if err != nil {
				t.Fatal(err)
			}
			seen := 0
			for _, pageErr := range scope.List(context.Background()) {
				if pageErr != nil {
					err = pageErr
					continue
				}
				seen++
				if mode == "changed-source" {
					client.Type = "compute"
				}
			}
			if seen != 1 || err == nil {
				t.Fatal(seen, err)
			}
			if mode == "wrong-parent" {
				var evidence *resource.ResponseError
				if !errors.As(err, &evidence) || string(evidence.Body) != body || evidence.Header.Get("X-Request-ID") != "page-evidence" || pages.Load() != 1 {
					t.Fatal(err, evidence, pages.Load())
				}
			} else if mode == "changed-source" {
				if !errors.Is(err, resource.ErrInvalidOption) || pages.Load() != 1 {
					t.Fatal(err, pages.Load())
				}
			} else if !gophercloud.ResponseCodeIs(err, 403) || pages.Load() != 2 {
				t.Fatal(err, pages.Load())
			}
		})
	}
}

func TestClusteringClusterPoliciesPreflightAndScopedWaits(t *testing.T) {
	cloud := testcloud.New(t)
	clusterPolicyParent(t, cloud)
	var reads atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/clusters/canonical-cluster/policies/policy-name", func(w http.ResponseWriter, r *http.Request) {
		if reads.Add(1) == 1 {
			testcloud.JSON(w, 200, `{"cluster_policy":`+clusterPolicyRow+`}`)
		} else {
			w.WriteHeader(404)
		}
	})
	client := cloud.Client("clustering", "/senlin/v1")
	scope, err := clusterpolicies.New(client).InCluster(context.Background(), resource.ID("selected"))
	if err != nil {
		t.Fatal(err)
	}
	for _, option := range []clusterpolicies.ListOption{nil, clusterpolicies.WithListQuery("enabled", "true"), clusterpolicies.WithListQuery("cluster_id", "foreign"), request.WithField[clusterpolicies.ListOpts]("vendor", true), request.WithHeader[clusterpolicies.ListOpts]("X-Auth-Token", "foreign"), request.WithArgument[clusterpolicies.ListOpts]("vendor", true), clusterpolicies.WithListSort("name:sideways")} {
		if _, err := scope.All(context.Background(), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	for _, key := range []string{"limit", "marker"} {
		if _, err := scope.All(context.Background(), clusterpolicies.WithListQuery(key, "2")); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(key, err)
		}
		if _, err := scope.Resources.All(context.Background(), resource.WithQuery(key, "2")); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(key, err)
		}
	}
	if _, err := scope.Resources.All(context.Background(), resource.WithQuery("cluster_id", "foreign")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := scope.WaitForStatus(context.Background(), resource.ID("policy-name"), "ACTIVE"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if err := scope.Resources.Delete(context.Background(), resource.ID("policy-name")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if reads.Load() != 0 {
		t.Fatal("unsupported input or waiter reached HTTP", reads.Load())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := scope.Get(ctx, "policy-name"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := scope.All(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := scope.WaitForDelete(ctx, resource.ID("policy-name")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := scope.WaitForDelete(context.Background(), resource.ID("policy-name"), resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second)); err != nil || reads.Load() != 2 {
		t.Fatal("wait redirected to the binding ID or response PolicyID", err, reads.Load())
	}
	for _, api := range []*clusterpolicies.API{nil, clusterpolicies.New(nil), clusterpolicies.New(&gophercloud.ServiceClient{})} {
		if _, err := api.InCluster(context.Background(), resource.ID("selected")); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	var nilScope *clusterpolicies.Scope
	if _, err := nilScope.Get(context.Background(), "policy"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
