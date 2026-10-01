package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/clustering/v1/clusters"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

func TestClusteringClustersAcceptedMalformedObjectRetainsEvidenceWithoutResend(t *testing.T) {
	for _, operation := range []string{"Create", "Update", "Get"} {
		for _, body := range []string{`{`, `{}`, `{"cluster":null}`, `{"cluster":[]}`, `{"cluster":{"min_size":true}}`, `{"cluster":{"metadata":[]}}`} {
			t.Run(operation+"/"+body, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				code := 200
				if operation == "Create" {
					code = 201
				}
				if operation == "Update" {
					code = 202
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("Location", "/v1/actions/accepted-action")
					w.Header().Set("X-Request-Id", "object-evidence")
					testcloud.JSON(w, code, body)
				})
				api := clusters.New(cloud.Client("clustering", "/v1"))
				var err error
				switch operation {
				case "Create":
					_, err = api.Create(context.Background(), clusters.CreateOpts{Name: "valid", ProfileID: "profile"})
				case "Update":
					_, err = api.Update(context.Background(), resource.ID("id"), clusters.UpdateOpts{Name: request.Present("valid")})
				case "Get":
					_, err = api.Get(context.Background(), "id")
				}
				var responseErr *resource.ResponseError
				if !errors.As(err, &responseErr) || responseErr.StatusCode != code || string(responseErr.Body) != body || responseErr.Header.Get("X-Request-Id") != "object-evidence" || calls.Load() != 1 {
					t.Fatal(err, responseErr, calls.Load())
				}
			})
		}
	}
}

func TestClusteringClustersAcceptedMalformedLocationRetainsEvidenceWithoutFollowing(t *testing.T) {
	for _, operation := range []string{"Create", "Update", "Delete"} {
		for _, location := range []string{"", "https://foreign.invalid/v1/actions/action", "/v1/clusters/action", "/v1/actions/action?query=1", "/v1/actions/action#fragment", "/v1/actions/a%2Fb", "duplicate"} {
			t.Run(operation+"/"+location, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				code := 202
				if operation == "Create" {
					code = 201
				}
				body := `{"cluster":{"id":"accepted","status":"UPDATING"}}`
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("X-Request-Id", "location-evidence")
					if location == "duplicate" {
						w.Header().Add("Location", "/v1/actions/first")
						w.Header().Add("Location", "/v1/actions/second")
					} else if location != "" || operation == "Create" {
						w.Header().Set("Location", location)
					}
					testcloud.JSON(w, code, body)
				})
				api := clusters.New(cloud.Client("clustering", "/v1"))
				var err error
				switch operation {
				case "Create":
					_, err = api.Create(context.Background(), clusters.CreateOpts{Name: "valid", ProfileID: "profile"})
				case "Update":
					_, err = api.Update(context.Background(), resource.ID("id"), clusters.UpdateOpts{Name: request.Present("valid")})
				case "Delete":
					_, err = api.Delete(context.Background(), resource.ID("id"))
				}
				var responseErr *resource.ResponseError
				if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &responseErr) || responseErr.StatusCode != code || string(responseErr.Body) != body || responseErr.Header.Get("X-Request-Id") != "location-evidence" || calls.Load() != 1 {
					t.Fatal(err, responseErr, calls.Load())
				}
			})
		}
	}
}

func TestClusteringClustersDeleteAcceptsEmptyBodyButRequiresAction(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("DELETE /v1/clusters/id", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Location", "actions/delete-action")
		w.Header().Set("X-Request-Id", "empty-delete")
		w.WriteHeader(202)
	})
	value, err := clusters.New(cloud.Client("clustering", "/v1")).Delete(context.Background(), resource.ID("id"))
	if err != nil || value.ActionID != "delete-action" || len(value.Body) != 0 || value.StatusCode != 202 || value.Header.Get("X-Request-Id") != "empty-delete" || calls.Load() != 1 {
		t.Fatal(value, err, calls.Load())
	}
}

func TestClusteringClustersExplicitSuccessCodesAndNativeErrors(t *testing.T) {
	for _, operation := range []string{"Create", "Update", "Delete", "Get"} {
		for _, code := range []int{200, 201, 202, 204, 400, 403, 409} {
			expected := 200
			if operation == "Create" {
				expected = 201
			}
			if operation == "Update" || operation == "Delete" {
				expected = 202
			}
			if code == expected || (operation == "Create" && code == http.StatusAccepted) {
				continue
			}
			t.Run(operation+"/"+fmt.Sprint(code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("Location", "actions/accepted")
					testcloud.JSON(w, code, `{"cluster":{"id":"id"}}`)
				})
				api := clusters.New(cloud.Client("clustering", "/v1"))
				var err error
				switch operation {
				case "Create":
					_, err = api.Create(context.Background(), clusters.CreateOpts{Name: "valid", ProfileID: "profile"})
				case "Update":
					_, err = api.Update(context.Background(), resource.ID("id"), clusters.UpdateOpts{Name: request.Present("valid")})
				case "Delete":
					_, err = api.Delete(context.Background(), resource.ID("id"))
				case "Get":
					_, err = api.Get(context.Background(), "id")
				}
				var responseErr *resource.ResponseError
				if !gophercloud.ResponseCodeIs(err, code) || errors.As(err, &responseErr) || calls.Load() != 1 {
					t.Fatal(err, calls.Load())
				}
			})
		}
	}
}

func TestClusteringClustersNamedDeleteFreezesHeadersForceAndRechecksSource(t *testing.T) {
	for _, changeType := range []bool{false, true} {
		t.Run(fmt.Sprint(changeType), func(t *testing.T) {
			cloud := testcloud.New(t)
			var lookups, deletes atomic.Int32
			client := cloud.Client("clustering", "/v1")
			var captured *request.Config[clusters.DeleteOpts]
			cloud.Mux.HandleFunc("GET /v1/clusters", func(w http.ResponseWriter, r *http.Request) {
				lookups.Add(1)
				captured.Headers["X-Custom"] = "changed"
				captured.Options.Force = false
				testcloud.JSON(w, 200, `{"clusters":[{"id":"stable","name":"selected"}]}`)
			})
			cloud.Mux.HandleFunc("DELETE /v1/clusters/stable", func(w http.ResponseWriter, r *http.Request) {
				deletes.Add(1)
				body, err := io.ReadAll(r.Body)
				if err != nil || string(body) != `{"force":true}` || r.Header.Get("X-Custom") != "original" {
					t.Error(string(body), err, r.Header)
				}
				w.Header().Set("Location", "actions/name-delete")
				w.WriteHeader(202)
			})
			if changeType {
				parent := client.ProviderClient.HTTPClient.Transport
				if parent == nil {
					parent = http.DefaultTransport
				}
				client.ProviderClient.HTTPClient.Transport = clusterRoundTrip(func(r *http.Request) (*http.Response, error) {
					response, err := parent.RoundTrip(r)
					if r.Method == "GET" {
						client.Type = "compute"
					}
					return response, err
				})
			}
			capture := func(config *request.Config[clusters.DeleteOpts]) error { captured = config; return nil }
			value, err := clusters.New(client).Delete(context.Background(), resource.Name("selected"), clusters.WithDeleteForce(true), clusters.WithDeleteHeader("X-Custom", "original"), capture)
			if changeType {
				if !errors.Is(err, resource.ErrInvalidOption) || lookups.Load() != 1 || deletes.Load() != 0 {
					t.Fatal(value, err, lookups.Load(), deletes.Load())
				}
			} else if err != nil || value.ActionID != "name-delete" || lookups.Load() != 1 || deletes.Load() != 1 {
				t.Fatal(value, err, lookups.Load(), deletes.Load())
			}
		})
	}
}

func TestClusteringClustersContextAndSourceValidationBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); t.Error("unexpected request", r.URL) })
	client := cloud.Client("clustering", "/v1")
	api := clusters.New(client)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	operations := []func() error{
		func() error {
			_, err := api.Create(ctx, clusters.CreateOpts{Name: "valid", ProfileID: "profile"})
			return err
		},
		func() error {
			_, err := api.Update(ctx, resource.Name("lookup"), clusters.UpdateOpts{Name: request.Present("valid")})
			return err
		},
		func() error { _, err := api.Delete(ctx, resource.ID("id")); return err },
		func() error { _, err := api.Get(ctx, "id"); return err },
		func() error { _, err := api.Find(ctx, resource.Name("lookup")); return err },
		func() error { _, err := api.All(ctx); return err },
	}
	for _, operation := range operations {
		if err := operation(); !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	}
	client.Type = "compute"
	if _, err := api.Get(context.Background(), "id"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	client.Type = "clustering"
	client.Microversion = "1.6"
	client.MoreHeaders = map[string]string{"openstack-api-version": "clustering 1.5"}
	if _, err := api.Delete(context.Background(), resource.Name("lookup")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := clusters.New(nil).Create(context.Background(), clusters.CreateOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	var absent *clusters.API
	if absent.RawClient() != nil {
		t.Fatal("nil API client")
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringClustersOptionalPresentValuesAndRawNullRemainDistinct(t *testing.T) {
	cloud := testcloud.New(t)
	var bodies []map[string]json.RawMessage
	cloud.Mux.HandleFunc("PATCH /v1/clusters/id", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		bodies = append(bodies, body["cluster"])
		w.Header().Set("Location", "actions/optional-values")
		testcloud.JSON(w, 202, `{"cluster":{"id":"id"}}`)
	})
	api := clusters.New(cloud.Client("clustering", "/v1"))
	options := []clusters.UpdateOption{
		clusters.WithUpdateOptions(clusters.UpdateOpts{Name: request.Present("valid"), ProfileID: request.Present("profile"), Timeout: request.Present(0)}),
		clusters.WithUpdateOptions(clusters.UpdateOpts{Name: request.Null[string](), ProfileID: request.Null[string](), Timeout: request.Null[int](), Metadata: json.RawMessage(`null`)}),
		clusters.WithUpdateOptions(clusters.UpdateOpts{Metadata: json.RawMessage(`{}`)}),
	}
	for _, option := range options {
		if _, err := api.Update(context.Background(), resource.ID("id"), clusters.UpdateOpts{}, option); err != nil {
			t.Fatal(err)
		}
	}
	if len(bodies) != 3 || string(bodies[0]["name"]) != `"valid"` || string(bodies[0]["profile_id"]) != `"profile"` || string(bodies[0]["timeout"]) != "0" || string(bodies[1]["name"]) != "null" || string(bodies[1]["profile_id"]) != "null" || string(bodies[1]["timeout"]) != "null" || string(bodies[1]["metadata"]) != "null" || len(bodies[2]) != 1 || string(bodies[2]["metadata"]) != "{}" {
		t.Fatal(bodies)
	}
}
