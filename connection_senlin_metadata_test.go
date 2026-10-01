package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	clustering "gophercloudsdk/clustering/v1"
	"gophercloudsdk/clustering/v1/clusters"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

func connectionMetadataService(t *testing.T, cloud *testcloud.Cloud) (*sdk.Connection, *clustering.Service) {
	t.Helper()
	conn, err := sdk.FromProvider(cloud.Provider,
		sdk.WithEndpoint(sdk.Clustering, cloud.Server.URL+"/reverse/senlin"),
		sdk.WithMicroversion(sdk.Clustering, "1.13"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.ClusteringV1(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return conn, service
}

func TestConnectionSenlinMetadataSharesSourceAndLocksResolvedRoute(t *testing.T) {
	cloud := testcloud.New(t)
	var lists, gets, patches, unexpected atomic.Int32
	input := map[string]any{"nested": map[string]any{"replacement": false}, "new": json.Number("9007199254740995"), "nullable": nil}
	var prepared *request.Config[clusters.MetadataOpts]
	check := func(r *http.Request, token string) {
		if r.Header.Get("X-Auth-Token") != token || r.Header.Get("OpenStack-API-Version") != "clustering 1.13" {
			t.Errorf("shared source lost: %s %s %#v", r.Method, r.URL, r.Header)
		}
	}
	cloud.Mux.HandleFunc("GET /reverse/senlin/v1/clusters", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		check(r, "read-token")
		if r.URL.Query().Get("name") != "selected" {
			t.Error(r.URL)
		}
		input["new"] = "changed during lookup"
		input["nested"].(map[string]any)["replacement"] = true
		prepared.Headers["X-Metadata-Trace"] = "changed during lookup"
		testcloud.JSON(w, 200, `{"clusters":[{"id":"other","name":"other","NAME":"selected"},{"id":"canonical","ID":"spoof","name":"selected","NAME":"other"}]}`)
	})
	cloud.Mux.HandleFunc("GET /reverse/senlin/v1/clusters/canonical", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		check(r, "read-token")
		if r.Header.Get("X-Metadata-Trace") != "prepared" || r.URL.RawQuery != "" {
			t.Error(r.Header, r.URL)
		}
		cloud.Provider.SetToken("patch-token")
		testcloud.JSON(w, 200, `{"cluster":{"id":"incidental-read-id","metadata":{"kept":9007199254740993,"nested":{"original":true},"nullable":"old"}}}`)
	})
	const accepted = `{"cluster":{"id":"incidental-accepted-id","metadata":{"actual":"queued response"},"status":"ACTIVE"},"future":9007199254740997}`
	cloud.Mux.HandleFunc("PATCH /reverse/senlin/v1/clusters/canonical", func(w http.ResponseWriter, r *http.Request) {
		patches.Add(1)
		check(r, "patch-token")
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != `{"cluster":{"metadata":{"kept":9007199254740993,"nested":{"replacement":false},"new":9007199254740995,"nullable":null}}}` || r.Header.Get("X-Metadata-Trace") != "prepared" || r.URL.RawQuery != "" {
			t.Error(string(body), err, r.Header, r.URL)
		}
		w.Header().Set("Location", "actions/metadata-action")
		w.Header().Set("X-Request-ID", "accepted-metadata")
		testcloud.JSON(w, 202, accepted)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		unexpected.Add(1)
		t.Error("metadata operation followed an action, invented a subresource or changed its route", r.Method, r.URL)
		w.WriteHeader(500)
	})
	conn, service := connectionMetadataService(t, cloud)
	again, err := conn.Clustering(context.Background())
	if err != nil || again != service || service.Clusters.RawClient() != service.RawClient() || service.Actions.RawClient() != service.RawClient() || service.RawClient().ProviderClient != cloud.Provider {
		t.Fatal("metadata API does not use the cached shared source", err)
	}
	cloud.Provider.SetToken("read-token")
	capture := func(config *request.Config[clusters.MetadataOpts]) error { prepared = config; return nil }
	result, err := service.Clusters.SetMetadata(context.Background(), resource.Name("selected"), input,
		clusters.WithMetadataHeader("X-Metadata-Trace", "prepared"), capture)
	if err != nil || result == nil || !result.Changed || result.Operation == nil || result.Operation.ActionID != "metadata-action" || result.Operation.Location != "actions/metadata-action" || result.Operation.StatusCode != 202 || string(result.Operation.Body) != accepted || result.Operation.Header.Get("X-Request-ID") != "accepted-metadata" {
		t.Fatal(result, err)
	}
	if string(result.Previous["nested"]) != `{"original":true}` || string(result.Requested["nested"]) != `{"replacement":false}` || string(result.Requested["new"]) != "9007199254740995" || string(result.ResponseCluster.UserMetadata["actual"]) != `"queued response"` || result.ResponseCluster.ID != "incidental-accepted-id" || !result.PreviousPresent || result.PreviousNull {
		t.Fatal("observed, requested and accepted states were conflated", result)
	}
	result.Operation.Header.Set("X-Request-ID", "consumer")
	result.Operation.Body[0] = '['
	if result.ResponseCluster.Operation.Header.Get("X-Request-ID") != "accepted-metadata" || string(result.ResponseCluster.Operation.Body) != accepted || lists.Load() != 1 || gets.Load() != 1 || patches.Load() != 1 || unexpected.Load() != 0 {
		t.Fatal("response evidence aliased or lookup/action request repeated", result, lists.Load(), gets.Load(), patches.Load(), unexpected.Load())
	}
}

func TestConnectionSenlinMetadataReadAliasesAndOwnedDelete(t *testing.T) {
	cloud := testcloud.New(t)
	var gets, patches, unexpected atomic.Int32
	keys := []string{"drop"}
	cloud.Mux.HandleFunc("GET /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
		call := gets.Add(1)
		if r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != "clustering 1.13" {
			t.Error(r.Header)
		}
		if call == 3 {
			keys[0] = "keep"
			if r.Header.Get("X-Metadata-Trace") != "delete" {
				t.Error(r.Header)
			}
		}
		w.Header().Set("Location", "https://incidental.invalid/unrelated")
		testcloud.JSON(w, 200, `{"cluster":{"id":null,"metadata":{"keep":9007199254740993,"drop":null},"Metadata":{"spoof":true}}}`)
	})
	cloud.Mux.HandleFunc("PATCH /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
		patches.Add(1)
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != `{"cluster":{"metadata":{"keep":9007199254740993}}}` || r.Header.Get("X-Metadata-Trace") != "delete" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != "clustering 1.13" {
			t.Error(string(body), err, r.Header)
		}
		w.Header().Set("Location", "/reverse/senlin/v1/actions/deletion-action")
		testcloud.JSON(w, 202, `{"cluster":{"metadata":{"keep":9007199254740993}}}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		unexpected.Add(1)
		t.Error("metadata read/delete invented or followed a route", r.Method, r.URL)
		w.WriteHeader(500)
	})
	_, service := connectionMetadataService(t, cloud)
	for _, read := range []func(context.Context, resource.Ref) (*clusters.MetadataView, error){service.Clusters.FetchMetadata, service.Clusters.GetMetadata} {
		view, err := read(context.Background(), resource.ID("selected"))
		if err != nil || view == nil || !view.Present || view.Null || string(view.Values["keep"]) != "9007199254740993" || string(view.Values["drop"]) != "null" || len(view.Values) != 2 || view.Cluster.Operation != nil || view.Cluster.Header.Get("Location") != "https://incidental.invalid/unrelated" {
			t.Fatal(view, err)
		}
	}
	result, err := service.Clusters.DeleteMetadata(context.Background(), resource.ID("selected"), keys, clusters.WithMetadataHeader("X-Metadata-Trace", "delete"))
	if err != nil || result == nil || !result.Changed || len(result.Previous) != 2 || len(result.Requested) != 1 || result.Operation == nil || result.Operation.ActionID != "deletion-action" || gets.Load() != 3 || patches.Load() != 1 || unexpected.Load() != 0 {
		t.Fatal(result, err, gets.Load(), patches.Load(), unexpected.Load())
	}
}

func TestConnectionSenlinMetadataAcceptedFailuresKeepOriginalEvidence(t *testing.T) {
	for _, mode := range []string{"missing-location", "wrong-prefix", "malformed-body", "unexpected-200"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, patches, unexpected atomic.Int32
			body := `{"cluster":{"id":"incidental","metadata":{"queued":true}}}`
			code := 202
			if mode == "malformed-body" {
				body = `{"cluster":null,"future":9007199254740993}`
			} else if mode == "unexpected-200" {
				code = 200
			}
			cloud.Mux.HandleFunc("GET /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 200, `{"cluster":{"id":"other","metadata":{}}}`)
			})
			cloud.Mux.HandleFunc("PATCH /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
				patches.Add(1)
				if r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != "clustering 1.13" {
					t.Error(r.Header)
				}
				if mode == "wrong-prefix" {
					w.Header().Set("Location", "/senlin/v1/actions/foreign-prefix")
				} else if mode != "missing-location" {
					w.Header().Set("Location", "/reverse/senlin/v1/actions/accepted")
				}
				w.Header().Set("X-Request-ID", "original-error-evidence")
				testcloud.JSON(w, code, body)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				unexpected.Add(1)
				t.Error("failed metadata replacement followed or changed a route", r.Method, r.URL)
				w.WriteHeader(500)
			})
			_, service := connectionMetadataService(t, cloud)
			result, err := service.Clusters.SetMetadata(context.Background(), resource.ID("selected"), map[string]any{"new": false})
			if result != nil || err == nil || gets.Load() != 1 || patches.Load() != 1 || unexpected.Load() != 0 {
				t.Fatal(result, err, gets.Load(), patches.Load(), unexpected.Load())
			}
			var evidence *resource.ResponseError
			if mode == "unexpected-200" {
				var native gophercloud.ErrUnexpectedResponseCode
				if errors.As(err, &evidence) || !errors.As(err, &native) || native.Actual != 200 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-ID") != "original-error-evidence" {
					t.Fatal(err)
				}
			} else if !errors.As(err, &evidence) || evidence.StatusCode != 202 || string(evidence.Body) != body || evidence.Header.Get("X-Request-ID") != "original-error-evidence" {
				t.Fatal(err, evidence)
			}
		})
	}
}

type connectionMetadataTransport func(*http.Request) (*http.Response, error)

func (f connectionMetadataTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestConnectionSenlinMetadataNoOpAndSourceRecheck(t *testing.T) {
	for _, mode := range []string{"empty-keys", "option-changed-type", "read-changed-type", "read-conflicting-version", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, patches atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 200, `{"cluster":{"metadata":{"keep":true}}}`)
			})
			cloud.Mux.HandleFunc("PATCH /reverse/senlin/v1/clusters/selected", func(w http.ResponseWriter, r *http.Request) {
				patches.Add(1)
				t.Error("changed source passed the metadata PATCH preflight")
				w.WriteHeader(500)
			})
			_, service := connectionMetadataService(t, cloud)
			client := service.RawClient()
			if mode == "read-changed-type" || mode == "read-conflicting-version" {
				transport := cloud.Provider.HTTPClient.Transport
				if transport == nil {
					transport = http.DefaultTransport
				}
				cloud.Provider.HTTPClient.Transport = connectionMetadataTransport(func(r *http.Request) (*http.Response, error) {
					response, err := transport.RoundTrip(r)
					if err == nil && r.Method == http.MethodGet {
						if mode == "read-changed-type" {
							client.Type = "compute"
						} else {
							client.MoreHeaders = map[string]string{"openstack-api-version": "clustering 1.2"}
						}
					}
					return response, err
				})
			}
			ctx := context.Background()
			var result *clusters.MetadataResult
			var err error
			if mode == "canceled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
				result, err = service.Clusters.DeleteMetadata(ctx, resource.Name("selected"), []string{})
			} else if mode == "option-changed-type" {
				result, err = service.Clusters.DeleteMetadata(ctx, resource.Name("selected"), []string{}, func(*request.Config[clusters.MetadataOpts]) error { client.Type = "compute"; return nil })
			} else if mode == "empty-keys" {
				result, err = service.Clusters.DeleteMetadata(ctx, resource.Name("selected"), []string{})
			} else {
				result, err = service.Clusters.SetMetadata(ctx, resource.ID("selected"), map[string]any{"new": true})
			}
			expectedGets := int32(0)
			if mode == "read-changed-type" || mode == "read-conflicting-version" {
				expectedGets = 1
			}
			if gets.Load() != expectedGets || patches.Load() != 0 {
				t.Fatal("unexpected HTTP count", gets.Load(), patches.Load(), err)
			}
			if mode == "empty-keys" {
				if err != nil || result == nil || result.Changed || result.Operation != nil || result.ResponseCluster != nil || result.Previous != nil || result.Requested != nil {
					t.Fatal("zero HTTP no-op fabricated an observation", result, err)
				}
			} else if mode == "canceled" {
				if result != nil || !errors.Is(err, context.Canceled) {
					t.Fatal(result, err)
				}
			} else if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(result, err)
			}
		})
	}
}
