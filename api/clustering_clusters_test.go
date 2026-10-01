package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/clustering/v1/clusters"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

func TestClusteringClustersCreateGetExactModelAndSubmission(t *testing.T) {
	for _, location := range []bool{false, true} {
		t.Run(fmt.Sprint(location), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			client := cloud.Client("clustering", "/catalog/v1")
			client.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + "/proxy/tenant/senlin/v1")
			response := `{"cluster":{"id":"response-id","name":"returned-name","profile_id":"actual-profile","profile_name":"profile.name","project":"project-id","user":"user-id","domain":null,"min_size":0,"max_size":-1,"desired_capacity":9007199254740993,"timeout":9007199254740995,"init_at":"2016-01-01T12:00:00.000000","created_at":"2016-01-01T12:00:01.000000","updated_at":null,"config":{"enabled":false},"metadata":{"large":9007199254740997},"data":{"ratio":1.0000000000000001},"dependents":null,"nodes":["node-id"],"policies":["policy-id"],"status":"CREATING","status_reason":"Initializing","future":null}}`
			cloud.Mux.HandleFunc("POST /proxy/tenant/senlin/v1/clusters", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Custom") != "captured" {
					t.Error(r.URL, r.Header)
				}
				var body map[string]map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				fields := body["cluster"]
				if len(body) != 1 || string(fields["name"]) != `"original"` || string(fields["profile_id"]) != `"profile.name"` || string(fields["desired_capacity"]) != "0" || string(fields["min_size"]) != "0" || string(fields["max_size"]) != "-1" || string(fields["timeout"]) != "null" || string(fields["config"]) != `{"enabled":false}` || string(fields["metadata"]) != `{"large":9007199254740993}` || string(fields["vendor"]) != `{"count":9007199254740995}` {
					t.Error(body)
				}
				w.Header().Set("X-Request-Id", "create-cluster")
				if location {
					w.Header().Set("Location", cloud.Server.URL+"/proxy/tenant/senlin/v1/actions/create-action")
				}
				testcloud.JSON(w, 201, response)
			})
			cloud.Mux.HandleFunc("GET /proxy/tenant/senlin/v1/clusters/short-id", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, response)
			})
			zero, unlimited := 0, -1
			input := clusters.CreateOpts{Name: "original", ProfileID: "profile.name", DesiredCapacity: &zero, MinSize: &zero, MaxSize: &unlimited, Timeout: request.Null[int](), Config: json.RawMessage(`{"enabled":false}`)}
			option := clusters.WithCreateOptions(input)
			input.Config[2], zero, unlimited = 'X', 1, 10
			metadata := map[string]json.Number{"large": "9007199254740993"}
			metadataOption := clusters.WithCreateMetadata(metadata)
			metadata["large"] = "0"
			api := clusters.New(client)
			value, err := api.Create(context.Background(), clusters.CreateOpts{}, option, metadataOption, clusters.WithCreateField("vendor", map[string]json.Number{"count": "9007199254740995"}), clusters.WithCreateHeader("X-Custom", "captured"))
			if err != nil || value.ID != "response-id" || value.ProfileID != "actual-profile" || value.ProfileName == nil || *value.ProfileName != "profile.name" || value.DesiredCapacity == nil || value.DesiredCapacity.String() != "9007199254740993" || value.Timeout == nil || value.Timeout.String() != "9007199254740995" || value.DomainID != nil || value.InitializedAt == nil || value.UpdatedAt != nil || string(value.UserMetadata["large"]) != "9007199254740997" || string(value.Data["ratio"]) != "1.0000000000000001" || string(value.Body["future"]) != "null" || value.StatusCode != 201 || value.Header.Get("X-Request-Id") != "create-cluster" || !reflect.DeepEqual(value.NodeIDs, []string{"node-id"}) || !reflect.DeepEqual(value.PolicyIDs, []string{"policy-id"}) {
				t.Fatal(value, err)
			}
			if location {
				if value.Operation == nil || value.Operation.ActionID != "create-action" || string(value.Operation.Body) != response || value.Operation.Header.Get("X-Request-Id") != "create-cluster" || value.Operation.StatusCode != 201 {
					t.Fatal(value.Operation)
				}
				value.Operation.Header.Set("X-Request-Id", "caller-change")
				value.Operation.Body[0] = '['
				if value.Header.Get("X-Request-Id") != "create-cluster" || string(value.Body["future"]) != "null" {
					t.Fatal("submission aliases model evidence", value)
				}
			} else if value.Operation != nil {
				t.Fatal("invented a create action", value.Operation)
			}
			fetched, err := api.Get(context.Background(), "short-id")
			if err != nil || fetched.ID != "response-id" || fetched.Operation != nil || calls.Load() != 2 {
				t.Fatal(fetched, err, calls.Load())
			}
		})
	}
}

func TestClusteringClustersCreateDefaultsOmitOptionalFields(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /v1/clusters", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body["cluster"]) != 2 || string(body["cluster"]["name"]) != `"name"` || string(body["cluster"]["profile_id"]) != `"profile-short-id"` {
			t.Error(body)
		}
		testcloud.JSON(w, 201, `{"cluster":{"id":"created","name":"name"}}`)
	})
	if _, err := clusters.New(cloud.Client("clustering", "/v1")).Create(context.Background(), clusters.CreateOpts{Name: "name", ProfileID: "profile-short-id"}); err != nil {
		t.Fatal(err)
	}
}

func TestClusteringClustersUpdateNullableFieldsFalseAndAcceptedAction(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("PATCH /v1/clusters/direct-name", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("OpenStack-API-Version") != "clustering 1.6" {
			t.Error(r.Header)
		}
		var body map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		fields := body["cluster"]
		if len(fields) != 7 || string(fields["name"]) != "null" || string(fields["profile_id"]) != "null" || string(fields["timeout"]) != "null" || string(fields["metadata"]) != "{}" || string(fields["config"]) != "null" || string(fields["profile_only"]) != "false" || string(fields["vendor"]) != "false" {
			t.Error(body)
		}
		w.Header().Set("Location", "/v1/actions/update-action")
		w.Header().Set("X-Request-Id", "update-cluster")
		testcloud.JSON(w, 202, `{"cluster":{"id":"response-id","name":"returned","status":"UPDATING","metadata":null}}`)
	})
	client := cloud.Client("clustering", "/v1")
	client.Microversion = "1.6"
	api := clusters.New(client)
	option := clusters.WithUpdateOptions(clusters.UpdateOpts{Name: request.Null[string](), ProfileID: request.Null[string](), Timeout: request.Null[int](), Metadata: json.RawMessage(`{}`)})
	value, err := api.Update(context.Background(), resource.ID("direct-name"), clusters.UpdateOpts{}, option, clusters.WithUpdateConfig(nil), clusters.WithUpdateProfileOnly(false), clusters.WithUpdateField("vendor", false))
	if err != nil || value.ID != "response-id" || value.Operation == nil || value.Operation.ActionID != "update-action" || value.Operation.StatusCode != 202 || value.Operation.Location != "/v1/actions/update-action" || value.Operation.Header.Get("X-Request-Id") != "update-cluster" || calls.Load() != 1 {
		t.Fatal(value, err, calls.Load())
	}
}

func TestClusteringClustersProfileOnlyAnyExplicitValueRequires16BeforeLookup(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); t.Error("unexpected request", r.URL) })
	client := cloud.Client("clustering", "/v1")
	api := clusters.New(client)
	for _, version := range []string{"", "1.0", "1.5", "latest"} {
		client.Microversion = version
		for _, value := range []bool{false, true} {
			if _, err := api.Update(context.Background(), resource.Name("needs-lookup"), clusters.UpdateOpts{}, clusters.WithUpdateProfileOnly(value)); !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal(version, value, err)
			}
		}
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}

func TestClusteringClustersNameUpdateFreezesBodyHeadersAndVersionGate(t *testing.T) {
	for _, changeVersion := range []bool{false, true} {
		t.Run(fmt.Sprint(changeVersion), func(t *testing.T) {
			cloud := testcloud.New(t)
			var lookups, updates atomic.Int32
			client := cloud.Client("clustering", "/v1")
			client.Microversion = "1.6"
			var captured *request.Config[clusters.UpdateOpts]
			cloud.Mux.HandleFunc("GET /v1/clusters", func(w http.ResponseWriter, r *http.Request) {
				lookups.Add(1)
				if r.URL.Query().Get("name") != "cluster.a" {
					t.Error(r.URL)
				}
				captured.Headers["X-Custom"] = "caller-change"
				captured.Options.Metadata = json.RawMessage(`{"label":"changed"}`)
				*captured.Options.ProfileOnly = true
				testcloud.JSON(w, 200, `{"clusters":[{"id":"not-selected","name":"clusterXa"},{"id":"stable","name":"cluster.a"}]}`)
			})
			cloud.Mux.HandleFunc("PATCH /v1/clusters/stable", func(w http.ResponseWriter, r *http.Request) {
				updates.Add(1)
				if r.Header.Get("X-Custom") != "original" {
					t.Error(r.Header)
				}
				var body map[string]map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if string(body["cluster"]["metadata"]) != `{"label":"original"}` || string(body["cluster"]["profile_only"]) != "false" {
					t.Error(body)
				}
				w.Header().Set("Location", "actions/name-update")
				testcloud.JSON(w, 202, `{"cluster":{"id":"stable","status":"UPDATING"}}`)
			})
			if changeVersion {
				parent := client.ProviderClient.HTTPClient.Transport
				if parent == nil {
					parent = http.DefaultTransport
				}
				client.ProviderClient.HTTPClient.Transport = clusterRoundTrip(func(r *http.Request) (*http.Response, error) {
					response, err := parent.RoundTrip(r)
					if r.Method == "GET" {
						client.Microversion = "1.5"
					}
					return response, err
				})
			}
			capture := func(config *request.Config[clusters.UpdateOpts]) error { captured = config; return nil }
			value, err := clusters.New(client).Update(context.Background(), resource.Name("cluster.a"), clusters.UpdateOpts{Metadata: json.RawMessage(`{"label":"original"}`)}, clusters.WithUpdateProfileOnly(false), clusters.WithUpdateHeader("X-Custom", "original"), capture)
			if changeVersion {
				if !errors.Is(err, resource.ErrUnsupported) || lookups.Load() != 1 || updates.Load() != 0 {
					t.Fatal(value, err, lookups.Load(), updates.Load())
				}
			} else if err != nil || value.Operation == nil || value.Operation.ActionID != "name-update" || lookups.Load() != 1 || updates.Load() != 1 {
				t.Fatal(value, err, lookups.Load(), updates.Load())
			}
		})
	}
}

type clusterRoundTrip func(*http.Request) (*http.Response, error)

func (run clusterRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return run(r) }

func TestClusteringClustersDeleteNormalForceBodiesAndSubmission(t *testing.T) {
	for _, force := range []bool{false, true} {
		t.Run(fmt.Sprint(force), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("DELETE /v1/clusters/direct-name", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				if force && string(body) != `{"force":true}` || !force && len(body) != 0 || r.URL.RawQuery != "" || r.Header.Get("X-Custom") != "original" {
					t.Error(string(body), r.URL, r.Header)
				}
				w.Header().Set("Location", "/v1/actions/deleted-action")
				w.Header().Set("X-Request-Id", "delete-cluster")
				testcloud.JSON(w, 202, `{"message":"accepted","future":9007199254740993}`)
			})
			api := clusters.New(cloud.Client("clustering", "/v1"))
			value, err := api.Delete(context.Background(), resource.ID("direct-name"), clusters.WithDeleteForce(force), clusters.WithDeleteHeader("X-Custom", "original"))
			if err != nil || value == nil || value.ActionID != "deleted-action" || value.StatusCode != 202 || value.Header.Get("X-Request-Id") != "delete-cluster" || string(value.Body) != `{"message":"accepted","future":9007199254740993}` || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
			if err := api.Resources.Delete(context.Background(), resource.ID("direct-name")); !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 1 {
				t.Fatal(err, calls.Load())
			}
		})
	}
}

func TestClusteringClustersDeleteDefaultsIgnoreOnlyMissingAndForceIsStrict(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("DELETE /v1/clusters/{identity}", func(w http.ResponseWriter, r *http.Request) {
		code := 404
		switch r.PathValue("identity") {
		case "denied":
			code = 403
		case "used":
			code = 409
		}
		testcloud.JSON(w, code, `{"error":{"message":"request failed"}}`)
	})
	api := clusters.New(cloud.Client("clustering", "/v1"))
	if value, err := api.Delete(context.Background(), resource.ID("missing")); err != nil || value != nil {
		t.Fatal(value, err)
	}
	if _, err := api.Delete(context.Background(), resource.ID("missing"), clusters.WithDeleteIgnoreMissing(false)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := api.Delete(context.Background(), resource.ID("missing"), clusters.WithDeleteForce(true)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if value, err := api.Delete(context.Background(), resource.ID("missing"), clusters.WithDeleteForce(true), clusters.WithDeleteIgnoreMissing(true)); err != nil || value != nil {
		t.Fatal(value, err)
	}
	for identity, code := range map[string]int{"denied": 403, "used": 409} {
		for _, force := range []bool{false, true} {
			if _, err := api.Delete(context.Background(), resource.ID(identity), clusters.WithDeleteForce(force), clusters.WithDeleteIgnoreMissing(true)); !gophercloud.ResponseCodeIs(err, code) {
				t.Fatal(identity, force, err)
			}
		}
	}
}

func TestClusteringClustersFindAndNamedDeleteResolveOnce(t *testing.T) {
	cloud := testcloud.New(t)
	var lookups, deletes atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/clusters", func(w http.ResponseWriter, r *http.Request) {
		lookups.Add(1)
		switch r.URL.Query().Get("name") {
		case "cluster.a":
			testcloud.JSON(w, 200, `{"clusters":[{"id":"wrong","name":"clusterXa"},{"id":"stable","name":"cluster.a"}]}`)
		case "duplicate":
			testcloud.JSON(w, 200, `{"clusters":[{"id":"a","name":"duplicate"},{"id":"b","name":"duplicate"}]}`)
		case "missing-id":
			testcloud.JSON(w, 200, `{"clusters":[{"name":"missing-id","id":null}]}`)
		default:
			testcloud.JSON(w, 200, `{"clusters":[]}`)
		}
	})
	cloud.Mux.HandleFunc("DELETE /v1/clusters/stable", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		w.Header().Set("Location", "actions/name-delete")
		w.WriteHeader(202)
	})
	api := clusters.New(cloud.Client("clustering", "/v1"))
	value, err := api.Delete(context.Background(), resource.Name("cluster.a"))
	if err != nil || value.ActionID != "name-delete" || lookups.Load() != 1 || deletes.Load() != 1 {
		t.Fatal(value, err, lookups.Load(), deletes.Load())
	}
	if found, err := api.Find(context.Background(), resource.Name("missing")); found != nil || err != nil {
		t.Fatal(found, err)
	}
	if _, err := api.Find(context.Background(), resource.Name("missing"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := api.Resources.Find(context.Background(), resource.Name("missing")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if submission, err := api.Delete(context.Background(), resource.Name("missing")); submission != nil || err != nil {
		t.Fatal(submission, err)
	}
	if _, err := api.Delete(context.Background(), resource.Name("missing"), clusters.WithDeleteForce(true)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := api.Delete(context.Background(), resource.Name("duplicate")); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatal(err)
	}
	if _, err := api.Delete(context.Background(), resource.Name("missing-id")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if deletes.Load() != 1 {
		t.Fatal("invalid lookup performed deletion", deletes.Load())
	}
}

func TestClusteringClustersListShortPageFiltersAndSnapshots(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/clusters", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("limit") != "5" || q.Get("name") != "selected" || q.Get("status") != "ACTIVE" || q.Get("sort") != "init_at:desc,name" || q.Get("global_project") != "false" || q.Get("vendor") != "retained" || q.Has("metadata") || q.Has("project_id") || q.Has("desired_capacity") {
			t.Error(q)
		}
		switch calls.Add(1) {
		case 1:
			if q.Has("marker") {
				t.Error(q)
			}
			testcloud.JSON(w, 200, `{"clusters":[{"id":"selected","name":"selected","project":"p","desired_capacity":9007199254740993,"metadata":{"nested":{"ratio":1.0,"flag":false,"extra":1}}},{"id":"filtered","name":"selected","project":"p","desired_capacity":9007199254740992,"metadata":{"nested":{"ratio":1.0,"flag":false}}}]}`)
		case 2:
			if q.Get("marker") != "filtered" {
				t.Error(q)
			}
			testcloud.JSON(w, 200, `{"clusters":[{"id":"selected-2","name":"selected","project":"p","desired_capacity":9007199254740993,"metadata":{"nested":{"ratio":1e0,"flag":false}}}]}`)
		case 3:
			if q.Get("marker") != "selected-2" {
				t.Error(q)
			}
			testcloud.JSON(w, 200, `{"clusters":[]}`)
		default:
			t.Error(calls.Load())
		}
	})
	global := false
	option := clusters.WithListOptions(clusters.ListOpts{Limit: 5, Name: "selected", Status: "ACTIVE", Sort: "init_at:desc,name", GlobalProject: &global})
	metadata := map[string]any{"nested": map[string]any{"ratio": json.Number("1"), "flag": false}}
	filter := clusters.WithListFilter("metadata", metadata)
	global, metadata["nested"] = true, nil
	options := []clusters.ListOption{option, filter, clusters.WithListFilter("project_id", "p"), clusters.WithListFilter("desired_capacity", json.Number("9007199254740993")), clusters.WithListQuery("vendor", "retained")}
	iterator := clusters.New(cloud.Client("clustering", "/v1")).List(context.Background(), options...)
	options[0] = clusters.WithListGlobalProject(true)
	if calls.Load() != 0 {
		t.Fatal("eager iterator")
	}
	for range 2 {
		var ids []string
		for value, err := range iterator {
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, value.ID)
			value.ID = "caller-mutation"
		}
		if !reflect.DeepEqual(ids, []string{"selected", "selected-2"}) || calls.Load() != 3 {
			t.Fatal(ids, calls.Load())
		}
		calls.Store(0)
	}
}

func TestClusteringClustersDefaultsAndInvalidOptionsPreflight(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v1/clusters", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.RawQuery != "" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"clusters":[]}`)
	})
	api := clusters.New(cloud.Client("clustering", "/v1"))
	values, err := api.All(context.Background())
	if err != nil || values == nil || len(values) != 0 || calls.Load() != 1 {
		t.Fatal(values, err, calls.Load())
	}
	for _, option := range []clusters.ListOption{nil, clusters.WithListOptions(clusters.ListOpts{Limit: -1}), clusters.WithListOptions(clusters.ListOpts{Sort: "name:wrong"}), clusters.WithListOptions(clusters.ListOpts{Marker: "bad/marker"}), clusters.WithListQuery("name", "hidden"), clusters.WithListQuery("status", "hidden"), clusters.WithListQuery("metadata", "hidden"), clusters.WithListFilter("unknown", true), request.WithHeader[clusters.ListOpts]("X-Auth-Token", "value"), request.WithArgument[clusters.ListOpts]("unknown", true)} {
		if _, err := api.All(context.Background(), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := api.Update(context.Background(), resource.Name("lookup"), clusters.UpdateOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, option := range []clusters.UpdateOption{clusters.WithUpdateField("min_size", 0), clusters.WithUpdateField("max_size", -1), clusters.WithUpdateField("desired_capacity", 0), clusters.WithUpdateField("profile_only", false), clusters.WithUpdateField("name", nil), clusters.WithUpdateField("id", "hidden")} {
		if _, err := api.Update(context.Background(), resource.Name("lookup"), clusters.UpdateOpts{Name: request.Present("valid")}, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	for _, option := range []clusters.DeleteOption{nil, request.WithField[clusters.DeleteOpts]("force", true), request.WithQuery[clusters.DeleteOpts]("force", "true"), clusters.WithDeleteHeader("X-Auth-Token", "owned"), clusters.WithDeleteHeader("OpenStack-API-Version", "clustering 1.6"), request.WithArgument[clusters.DeleteOpts]("unknown", true)} {
		if _, err := api.Delete(context.Background(), resource.Name("lookup"), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("preflight performed lookup/mutation", calls.Load())
	}
}

func TestClusteringClustersMutationSchemasProtectCoreAndResponseFields(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); t.Error("unexpected request", r.URL) })
	api := clusters.New(cloud.Client("clustering", "/v1"))
	base := clusters.CreateOpts{Name: "valid", ProfileID: "profile"}
	for _, opts := range []clusters.CreateOpts{{}, {Name: "valid"}, {Name: "1bad", ProfileID: "profile"}, {Name: strings.Repeat("a", 255), ProfileID: "profile"}, {Name: "valid", ProfileID: "profile", Config: json.RawMessage(`[]`)}, {Name: "valid", ProfileID: "profile", Metadata: json.RawMessage(`false`)}} {
		if _, err := api.Create(context.Background(), opts); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(opts, err)
		}
	}
	for _, option := range []clusters.CreateOption{clusters.WithCreateField("metadata", nil), clusters.WithCreateField("profile_id", "hidden"), clusters.WithCreateField("profile_only", false), clusters.WithCreateField("status", "ACTIVE"), clusters.WithCreateHeader("Content-Type", "text/plain"), request.WithQuery[clusters.CreateOpts]("query", "hidden")} {
		if _, err := api.Create(context.Background(), base, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	for _, opts := range []clusters.UpdateOpts{{Name: request.Present("")}, {ProfileID: request.Present("")}, {Config: json.RawMessage(`false`)}, {Metadata: json.RawMessage(`[]`)}} {
		if _, err := api.Update(context.Background(), resource.Name("lookup"), opts); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(opts, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal(calls.Load())
	}
}
