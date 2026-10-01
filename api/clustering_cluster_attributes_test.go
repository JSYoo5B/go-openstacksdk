package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/clustering/v1/clusterattributes"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func clusterAttributeClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("clustering", "/senlin/v1")
	client.Microversion = "1.2"
	return client
}

func clusterAttributeParent(t *testing.T, cloud *testcloud.Cloud) {
	t.Helper()
	cloud.Mux.HandleFunc("GET /senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("OpenStack-API-Version") != "clustering 1.2" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"cluster":{"id":"canonical-parent","name":"selected"}}`)
	})
}

func TestClusteringClusterAttributesSynchronousCodesAndRawValues(t *testing.T) {
	rows := `[{"id":"object","value":{"big":9007199254740993,"fraction":1.234567890123456789}},{"id":"array","value":[false,null,9007199254740993]},{"id":"number","value":-1.234567890123456789e+9000},{"id":"boolean","value":false},{"id":"string","value":"text"},{"id":"null","value":null},{"id":"omitted","future":true}]`
	expected := []string{`{"big":9007199254740993,"fraction":1.234567890123456789}`, `[false,null,9007199254740993]`, `-1.234567890123456789e+9000`, `false`, `"text"`, `null`, ``}
	for _, code := range []int{200, 202} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			clusterAttributeParent(t, cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /senlin/v1/clusters/canonical-parent/attrs/$.value", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != "clustering 1.2" {
					t.Error(r.URL, r.Header)
				}
				w.Header().Set("Location", "https://foreign.example/actions/no-follow")
				w.Header().Set("X-Request-ID", "attribute-read")
				testcloud.JSON(w, code, `{"cluster_attributes":`+rows+`,"action":"incidental"}`)
			})
			scope, err := clusterattributes.New(clusterAttributeClient(cloud)).InCluster(context.Background(), resource.ID("selected"), "$.value")
			if err != nil {
				t.Fatal(err)
			}
			values, err := scope.All(context.Background())
			if err != nil || len(values) != len(expected) {
				t.Fatal(values, err)
			}
			for index, value := range values {
				if string(value.Value) != expected[index] || value.URIClusterID != "canonical-parent" || value.URIPath != "$.value" || value.StatusCode != code || value.Header.Get("Location") != "https://foreign.example/actions/no-follow" || value.Header.Get("X-Request-ID") != "attribute-read" {
					t.Fatal(index, value)
				}
			}
			if values[5].Value == nil || values[6].Value != nil || string(values[6].Body["future"]) != "true" {
				t.Fatal("null and omitted value were conflated", values)
			}
			values[0].Header.Set("X-Request-ID", "consumer")
			if values[1].Header.Get("X-Request-ID") != "attribute-read" || calls.Load() != 1 {
				t.Fatal("collection read followed a Location or shared headers", calls.Load())
			}
		})
	}
}

func TestClusteringClusterAttributesJSONPathRemainsOneFixedSegment(t *testing.T) {
	for _, input := range []string{"  $.links['a/b'][0]?q#fragment  ", "//foreign.example/path", ".", "..", "none"} {
		t.Run(input, func(t *testing.T) {
			cloud := testcloud.New(t)
			var parents, calls atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
				parents.Add(1)
				testcloud.JSON(w, 200, `{"cluster":{"id":"canonical-parent"}}`)
			})
			path := strings.TrimSpace(input)
			escaped := url.PathEscape(path)
			if path == "." || path == ".." {
				escaped = strings.ReplaceAll(path, ".", "%2E")
			}
			cloud.Mux.HandleFunc("GET /reverse/senlin/v1/clusters/canonical-parent/attrs/{path}", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.EscapedPath() != "/reverse/senlin/v1/clusters/canonical-parent/attrs/"+escaped || r.PathValue("path") != path || r.URL.RawQuery != "" || r.URL.Fragment != "" {
					t.Error(r.URL, r.URL.EscapedPath(), r.PathValue("path"))
				}
				testcloud.JSON(w, 200, `{"cluster_attributes":[{"id":"node","value":0}]}`)
			})
			client := cloud.Client("clustering", "/reverse/senlin/v1")
			client.Microversion = "1.2"
			scope, err := clusterattributes.New(client).InCluster(context.Background(), resource.ID("selected"), input)
			input = "changed" // The fixed scope owns its normalized path.
			if err != nil || scope.Path() != path {
				t.Fatal(scope, err)
			}
			values, err := scope.All(context.Background())
			if err != nil || len(values) != 1 || values[0].URIPath != path || parents.Load() != 1 || calls.Load() != 1 {
				t.Fatal(values, err, parents.Load(), calls.Load())
			}
		})
	}
}

func TestClusteringClusterAttributesMinimumVersionAndPathPreflight(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Error("invalid input reached HTTP", r.URL)
	})
	client := clusterAttributeClient(cloud)
	api := clusterattributes.New(client)
	for _, version := range []string{"", "1.0", "1.1", "latest"} {
		client.Microversion = version
		if _, err := api.InCluster(context.Background(), resource.ID("selected"), "$.value"); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(version, err)
		}
	}
	client.Microversion = "1.2"
	for _, path := range []string{"", " \t\n ", "None", " None "} {
		if _, err := api.InCluster(context.Background(), resource.ID("selected"), path); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(path, err)
		}
	}
	for _, ref := range []resource.Ref{resource.ID(""), resource.ID("bad/id")} {
		if _, err := api.InCluster(context.Background(), ref, "$.value"); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(ref, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.InCluster(ctx, resource.ID("selected"), "$.value"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	for _, api := range []*clusterattributes.API{nil, clusterattributes.New(nil), clusterattributes.New(&gophercloud.ServiceClient{})} {
		if _, err := api.InCluster(context.Background(), resource.ID("selected"), "$.value"); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringClusterAttributesParentCanonicalizationAndGateRecheck(t *testing.T) {
	for _, mode := range []string{"named", "case-id", "missing-id", "null-id", "downgraded"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := clusterAttributeClient(cloud)
			var parents atomic.Int32
			body := `{"cluster":{"id":"canonical-parent","name":"selected"}}`
			handler := func(w http.ResponseWriter, r *http.Request) {
				parents.Add(1)
				w.Header().Set("X-Request-ID", "parent-proof")
				if mode == "missing-id" {
					body = `{"cluster":{}}`
				}
				if mode == "null-id" {
					body = `{"cluster":{"id":null}}`
				}
				if mode == "case-id" {
					body = `{"cluster":{"id":"canonical-parent","ID":"other-parent"}}`
				}
				if mode == "downgraded" {
					client.Microversion = "1.1"
				}
				if mode == "named" {
					if r.URL.Query().Get("name") != "selected" {
						t.Error(r.URL)
					}
					body = `{"clusters":[{"id":"unrelated-parent","name":"other"},{"id":"canonical-parent","name":"selected"}]}`
				}
				testcloud.JSON(w, 200, body)
			}
			cloud.Mux.HandleFunc("GET /senlin/v1/clusters", handler)
			cloud.Mux.HandleFunc("GET /senlin/v1/clusters/selected", handler)
			ref := resource.ID("selected")
			if mode == "named" {
				ref = resource.Name("selected")
			}
			scope, err := clusterattributes.New(client).InCluster(context.Background(), ref, "$.value")
			if mode == "named" || mode == "case-id" {
				if err != nil || scope.ClusterID() != "canonical-parent" {
					t.Fatal(scope, err)
				}
			} else if mode == "downgraded" {
				if scope != nil || !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal(scope, err)
				}
			} else {
				var proof *resource.ResponseError
				if scope != nil || !errors.As(err, &proof) || string(proof.Body) != body || proof.Header.Get("X-Request-ID") != "parent-proof" || proof.StatusCode != 200 {
					t.Fatal(scope, err, proof)
				}
			}
			if parents.Load() != 1 {
				t.Fatal(parents.Load())
			}
		})
	}
}

func TestClusteringClusterAttributesCanonicalRawFieldsOwnIdentityAndValue(t *testing.T) {
	cloud := testcloud.New(t)
	clusterAttributeParent(t, cloud)
	cloud.Mux.HandleFunc("GET /senlin/v1/clusters/canonical-parent/attrs/$.value", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"cluster_attributes":[{"id":"canonical-node","ID":"other-node","value":9007199254740993,"Value":"other-value"},{"id":"omitted","Value":null}]}`)
	})
	scope, err := clusterattributes.New(clusterAttributeClient(cloud)).InCluster(context.Background(), resource.ID("selected"), "$.value")
	if err != nil {
		t.Fatal(err)
	}
	values, err := scope.All(context.Background())
	if err != nil || len(values) != 2 || values[0].NodeID != "canonical-node" || string(values[0].Value) != "9007199254740993" || values[1].Value != nil {
		t.Fatal(values, err)
	}
	values[0].Body["value"][0] = '0'
	if string(values[0].Value) != "9007199254740993" || string(values[0].Body["ID"]) != `"other-node"` {
		t.Fatal("raw value aliases Body or casefold extension was lost", values[0])
	}
}

func TestClusteringClusterAttributesExactParentNameUsesCanonicalRawField(t *testing.T) {
	for _, row := range []string{`{"id":"other-parent","name":"other","NAME":"selected"}`, `{"id":"other-parent","NAME":"selected"}`, `{"id":"other-parent","name":null,"NAME":"selected"}`} {
		t.Run(row, func(t *testing.T) {
			cloud := testcloud.New(t)
			var parents atomic.Int32
			cloud.Mux.HandleFunc("GET /senlin/v1/clusters", func(w http.ResponseWriter, r *http.Request) {
				parents.Add(1)
				if r.URL.Query().Get("name") != "selected" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"clusters":[`+row+`]}`)
			})
			scope, err := clusterattributes.New(clusterAttributeClient(cloud)).InCluster(context.Background(), resource.Name("selected"), "$.value")
			var missing *resource.NotFoundError
			if scope != nil || !errors.As(err, &missing) || parents.Load() != 1 {
				t.Fatal(scope, err, parents.Load())
			}
		})
	}
}

func TestClusteringClusterAttributesLazyExplicitPagingAndVersionGuard(t *testing.T) {
	for _, mode := range []string{"linked", "wrong-path", "downgraded"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			clusterAttributeParent(t, cloud)
			client := clusterAttributeClient(cloud)
			var pages atomic.Int32
			body := `{"cluster_attributes":[{"id":"node","value":false}]}`
			cloud.Mux.HandleFunc("GET /senlin/v1/clusters/canonical-parent/attrs/$.value", func(w http.ResponseWriter, r *http.Request) {
				pages.Add(1)
				if r.URL.Query().Get("marker") == "" {
					path := "$.value"
					if mode == "wrong-path" {
						path = "$.other"
					}
					w.Header().Set("Link", fmt.Sprintf(`</senlin/v1/clusters/canonical-parent/attrs/%s?marker=opaque>; rel="next"`, path))
				} else if r.URL.Query().Get("marker") != "opaque" {
					t.Error(r.URL)
				}
				w.Header().Set("X-Request-ID", "page-proof")
				testcloud.JSON(w, 200, body)
			})
			scope, err := clusterattributes.New(client).InCluster(context.Background(), resource.ID("selected"), "$.value")
			if err != nil {
				t.Fatal(err)
			}
			iterator := scope.List(context.Background())
			if pages.Load() != 0 {
				t.Fatal("list was eager")
			}
			seen := 0
			for value, pageErr := range iterator {
				if pageErr != nil {
					err = pageErr
					continue
				}
				seen++
				value.NodeID, value.URIClusterID, value.URIPath = "consumer", "consumer-parent", "consumer-path"
				if mode == "downgraded" {
					client.Microversion = "1.1"
				}
			}
			if mode == "linked" {
				if err != nil || seen != 2 || pages.Load() != 2 {
					t.Fatal(seen, err, pages.Load())
				}
				for _, pageErr := range iterator {
					if pageErr != nil {
						t.Fatal(pageErr)
					}
					break
				}
				if pages.Load() != 3 {
					t.Fatal("break fetched another page", pages.Load())
				}
			} else if mode == "downgraded" {
				if !errors.Is(err, resource.ErrUnsupported) || seen != 1 || pages.Load() != 1 {
					t.Fatal(seen, err, pages.Load())
				}
			} else {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || string(proof.Body) != body || proof.Header.Get("X-Request-ID") != "page-proof" || seen != 1 || pages.Load() != 1 {
					t.Fatal(seen, err, proof, pages.Load())
				}
			}
		})
	}
}

func TestClusteringClusterAttributesResponseErrorsPreserveEvidence(t *testing.T) {
	for _, code := range []int{200, 202, 201, 403} {
		bodies := []string{`{"cluster_attributes":null}`, `{"cluster_attributes":[{"id":null,"value":null}]}`, `{"cluster_attributes":[false]}`}
		if code == 201 || code == 403 {
			bodies = []string{`{"error":"native failure"}`}
		}
		for _, body := range bodies {
			t.Run(fmt.Sprint(code)+body, func(t *testing.T) {
				cloud := testcloud.New(t)
				clusterAttributeParent(t, cloud)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("GET /senlin/v1/clusters/canonical-parent/attrs/$.value", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("X-Request-ID", "original-evidence")
					testcloud.JSON(w, code, body)
				})
				scope, err := clusterattributes.New(clusterAttributeClient(cloud)).InCluster(context.Background(), resource.ID("selected"), "$.value")
				if err != nil {
					t.Fatal(err)
				}
				_, err = scope.All(context.Background())
				var proof *resource.ResponseError
				if code == 200 || code == 202 {
					if !errors.As(err, &proof) || string(proof.Body) != body || proof.StatusCode != code || proof.Header.Get("X-Request-ID") != "original-evidence" {
						t.Fatal(err, proof)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || !gophercloud.ResponseCodeIs(err, code) || string(native.Body) != body || native.ResponseHeader.Get("X-Request-ID") != "original-evidence" || errors.As(err, &proof) {
						t.Fatal(err)
					}
				}
				if calls.Load() != 1 {
					t.Fatal("collection response failure resent or followed Location", calls.Load())
				}
			})
		}
	}
}

func TestClusteringClusterAttributesListOnlyCapabilitiesAndQueries(t *testing.T) {
	cloud := testcloud.New(t)
	clusterAttributeParent(t, cloud)
	scope, err := clusterattributes.New(clusterAttributeClient(cloud)).InCluster(context.Background(), resource.ID("selected"), "$.value")
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"limit", "marker", "vendor"} {
		if _, err := scope.Resources.All(context.Background(), resource.WithQuery(query, "value")); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(query, err)
		}
	}
	if _, err := scope.Resources.Get(context.Background(), "node"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if err := scope.Resources.Delete(context.Background(), resource.ID("node")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := scope.Resources.Wait(context.Background(), resource.ID("node"), "ACTIVE"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if err := scope.Resources.WaitDeleted(context.Background(), resource.ID("node")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := scope.All(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	var nilScope *clusterattributes.Scope
	if _, err := nilScope.All(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
