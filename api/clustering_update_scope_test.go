package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Pinned update_cluster/update_node forward **attrs to Proxy._update, whose
// base_path parameter alone is passed as a named commit control (proxy.py:716-
// 748). Python changes the route for that commit; Go owns a fixed literal
// collection scope for Update and tracked Load/Track/Commit/Refresh. These are
// isolated HTTP fixtures, not evidence of a deployment's alternate routes.
type clusteringUpdateScopeHandle struct {
	*clusteringAsyncTrackedHandle
	gateNull func() error
}

type clusteringUpdateScopeAPI struct {
	update     func(context.Context, resource.Ref, string, bool, *func()) (clusteringAsyncTrackedView, error)
	baseUpdate func(context.Context) (clusteringAsyncTrackedView, error)
	load       func(context.Context, resource.Ref) (*clusteringUpdateScopeHandle, error)
	track      func(map[string]json.RawMessage) (*clusteringUpdateScopeHandle, error)
}

type clusteringUpdateScopeCase struct {
	kind, version, older, gate string
	new                        func(*gophercloud.ServiceClient, string) (*clusteringUpdateScopeAPI, error)
}

var clusteringUpdateScopeCases = []clusteringUpdateScopeCase{
	{"cluster", "1.6", "1.5", "profile_only", func(client *gophercloud.ServiceClient, path string) (*clusteringUpdateScopeAPI, error) {
		base := clusters.New(client)
		scope, err := base.AtBasePath(path)
		if err != nil {
			return nil, err
		}
		wrap := func(value *clusters.TrackedCluster, err error) (*clusteringUpdateScopeHandle, error) {
			if err != nil {
				return nil, err
			}
			return &clusteringUpdateScopeHandle{clusteringAsyncClusterHandle(value), func() error {
				return value.Edit(clusters.UpdateOpts{}, clusters.WithUpdateProfileOnlyNull())
			}}, nil
		}
		return &clusteringUpdateScopeAPI{
			update: func(ctx context.Context, ref resource.Ref, name string, gated bool, captured *func()) (clusteringAsyncTrackedView, error) {
				options := []clusters.UpdateOption{clusters.WithUpdateName(name), clusters.WithUpdateMetadata(map[string]json.Number{"big": "9007199254740993"}), clusters.WithUpdateField("vendor", false), clusters.WithUpdateHeader("X-Snapshot", "captured")}
				if gated {
					options = append(options, clusters.WithUpdateProfileOnlyNull())
				}
				if captured != nil {
					options = append(options, func(config *request.Config[clusters.UpdateOpts]) error {
						*captured = func() {
							config.Options.Name = request.Present("Altered")
							config.Options.Metadata[2] = 'X'
							config.Headers["X-Snapshot"] = "altered"
							config.Fields["vendor"] = json.RawMessage("true")
						}
						return nil
					})
				}
				value, err := scope.Update(ctx, ref, clusters.UpdateOpts{}, options...)
				return clusteringAsyncClusterView(value), err
			},
			baseUpdate: func(ctx context.Context) (clusteringAsyncTrackedView, error) {
				value, err := base.Update(ctx, resource.ID("base-id"), clusters.UpdateOpts{}, clusters.WithUpdateName("Base"))
				return clusteringAsyncClusterView(value), err
			},
			load: func(ctx context.Context, ref resource.Ref) (*clusteringUpdateScopeHandle, error) {
				return wrap(scope.Load(ctx, ref))
			},
			track: func(body map[string]json.RawMessage) (*clusteringUpdateScopeHandle, error) {
				return wrap(scope.Track(&clusters.Cluster{ID: "typed-shadow", Metadata: resource.Metadata{Body: body}}))
			},
		}, nil
	}},
	{"node", "1.13", "1.12", "tainted", func(client *gophercloud.ServiceClient, path string) (*clusteringUpdateScopeAPI, error) {
		base := nodes.New(client)
		scope, err := base.AtBasePath(path)
		if err != nil {
			return nil, err
		}
		wrap := func(value *nodes.TrackedNode, err error) (*clusteringUpdateScopeHandle, error) {
			if err != nil {
				return nil, err
			}
			return &clusteringUpdateScopeHandle{clusteringAsyncNodeHandle(value), func() error {
				return value.Edit(nodes.UpdateOpts{}, nodes.WithUpdateTaintedNull())
			}}, nil
		}
		return &clusteringUpdateScopeAPI{
			update: func(ctx context.Context, ref resource.Ref, name string, gated bool, captured *func()) (clusteringAsyncTrackedView, error) {
				options := []nodes.UpdateOption{nodes.WithUpdateName(name), nodes.WithUpdateMetadata(map[string]json.Number{"big": "9007199254740993"}), nodes.WithUpdateField("vendor", false), nodes.WithUpdateHeader("X-Snapshot", "captured")}
				if gated {
					options = append(options, nodes.WithUpdateTaintedNull())
				}
				if captured != nil {
					options = append(options, func(config *request.Config[nodes.UpdateOpts]) error {
						*captured = func() {
							config.Options.Name = request.Present("Altered")
							config.Options.Metadata[2] = 'X'
							config.Headers["X-Snapshot"] = "altered"
							config.Fields["vendor"] = json.RawMessage("true")
						}
						return nil
					})
				}
				value, err := scope.Update(ctx, ref, nodes.UpdateOpts{}, options...)
				return clusteringAsyncNodeView(value), err
			},
			baseUpdate: func(ctx context.Context) (clusteringAsyncTrackedView, error) {
				value, err := base.Update(ctx, resource.ID("base-id"), nodes.UpdateOpts{}, nodes.WithUpdateName("Base"))
				return clusteringAsyncNodeView(value), err
			},
			load: func(ctx context.Context, ref resource.Ref) (*clusteringUpdateScopeHandle, error) {
				return wrap(scope.Load(ctx, ref))
			},
			track: func(body map[string]json.RawMessage) (*clusteringUpdateScopeHandle, error) {
				return wrap(scope.Track(&nodes.Node{ID: "typed-shadow", Metadata: resource.Metadata{Body: body}}))
			},
		}, nil
	}},
}

func clusteringUpdateScopeSeed(id string) map[string]json.RawMessage {
	return map[string]json.RawMessage{"id": json.RawMessage(fmt.Sprintf("%q", id)), "name": json.RawMessage(`"Original"`), "metadata": json.RawMessage(`{"old":true}`)}
}

func clusteringUpdateScopeResponse(kind, fields string) string {
	return fmt.Sprintf(`{%q:{%s}}`, kind, fields)
}

func TestClusteringUpdateScopeRejectsUnsafePathsAndNilSourcesWithoutHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	invalid := []string{"", "/clusters", "clusters/", "a//clusters", ".", "a/../clusters", "a/./clusters", "https://foreign/clusters", "a:clusters", "a%2fclusters", "%(id)s/clusters", `a\clusters`, "clusters?query=1", "clusters#fragment", "a clusters", "a\tclusters", "a\nclusters", "a\x00clusters", "a\u00a0clusters", string([]byte{'a', '/', 0xff})}
	for _, entry := range clusteringUpdateScopeCases {
		t.Run(entry.kind, func(t *testing.T) {
			client := cloud.Client("clustering", "/v1")
			for _, path := range invalid {
				if scope, err := entry.new(client, path); scope != nil || !errors.Is(err, resource.ErrInvalidOption) {
					t.Errorf("path %q: %v, %v", path, scope, err)
				}
			}
			for _, source := range []*gophercloud.ServiceClient{nil, {Type: "clustering", Endpoint: client.Endpoint}} {
				if scope, err := entry.new(source, "alternate/"+entry.kind+"s"); scope != nil || !errors.Is(err, resource.ErrInvalidOption) {
					t.Error(scope, err)
				}
			}
		})
	}
	var clusterAPI *clusters.API
	var nodeAPI *nodes.API
	if _, err := clusterAPI.AtBasePath("clusters"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := nodeAPI.AtBasePath("nodes"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("constructor performed HTTP", calls.Load())
	}
}

func TestClusteringUpdateScopeLiteralRoutingLiveTokenAndConcurrentIsolation(t *testing.T) {
	for _, entry := range clusteringUpdateScopeCases {
		t.Run(entry.kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("clustering", "/catalog/v1")
			wirePrefix, decodedPrefix := "/proxy/tenant%20alpha/senlin/v1", "/proxy/tenant alpha/senlin/v1"
			client.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + wirePrefix)
			client.Microversion = entry.version
			client.MoreHeaders = map[string]string{"X-Configured": "unchanged"}
			endpoint, resourceBase, headers := client.Endpoint, client.ResourceBase, maps.Clone(client.MoreHeaders)
			paths := []string{"tenants/project-alpha/" + entry.kind + "s", "tenants/租户/" + entry.kind + "s"}
			var first, fresh atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPatch || r.URL.RawQuery != "" || r.Header.Get("X-Configured") != "unchanged" || r.Header.Get("OpenStack-API-Version") != "clustering "+entry.version {
					t.Error(r.Method, r.URL, r.Header)
				}
				switch r.Header.Get("X-Auth-Token") {
				case "test-token":
					first.Add(1)
				case "fresh-token":
					fresh.Add(1)
				default:
					t.Error(r.Header)
				}
				valid := r.URL.Path == decodedPrefix+"/"+entry.kind+"s/base-id"
				for _, path := range paths {
					if r.URL.Path == decodedPrefix+"/"+path+"/direct-id" {
						valid = true
						segments := strings.Split(path, "/")
						for i := range segments {
							segments[i] = url.PathEscape(segments[i])
						}
						if r.URL.EscapedPath() != wirePrefix+"/"+strings.Join(segments, "/")+"/direct-id" {
							t.Error("literal path was re-escaped", r.URL.EscapedPath())
						}
					}
				}
				if !valid {
					t.Error("unselected route", r.URL)
				}
				w.Header().Set("Location", wirePrefix+"/actions/scoped-action")
				testcloud.JSON(w, 202, clusteringUpdateScopeResponse(entry.kind, `"id":"response-id","name":"Accepted"`))
			})
			left, err := entry.new(client, paths[0])
			if err != nil {
				t.Fatal(err)
			}
			right, err := entry.new(client, paths[1])
			if err != nil {
				t.Fatal(err)
			}
			value, err := left.update(context.Background(), resource.ID("direct-id"), "Before", false, nil)
			if err != nil || value.operation == nil || value.operation.ActionID != "scoped-action" {
				t.Fatal(value, err)
			}
			cloud.Provider.SetToken("fresh-token")
			var wait sync.WaitGroup
			for _, scope := range []*clusteringUpdateScopeAPI{left, right} {
				wait.Add(1)
				go func(scope *clusteringUpdateScopeAPI) {
					defer wait.Done()
					value, err := scope.update(context.Background(), resource.ID("direct-id"), "After", false, nil)
					if err != nil || value.operation == nil || value.operation.Location != wirePrefix+"/actions/scoped-action" {
						t.Error(value, err)
					}
				}(scope)
			}
			wait.Wait()
			if _, err := left.baseUpdate(context.Background()); err != nil {
				t.Fatal(err)
			}
			if first.Load() != 1 || fresh.Load() != 3 || client.Endpoint != endpoint || client.ResourceBase != resourceBase || !reflect.DeepEqual(client.MoreHeaders, headers) {
				t.Fatal(first.Load(), fresh.Load(), client)
			}
		})
	}
}

func TestClusteringUpdateScopeNameLookupUsesAllScopedPagesAndFrozenInputs(t *testing.T) {
	for _, entry := range clusteringUpdateScopeCases {
		t.Run(entry.kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("clustering", clusteringAsyncTrackedPrefix)
			client.Microversion = entry.version
			path := "tenants/project-alpha/" + entry.kind + "s"
			scope, err := entry.new(client, path)
			if err != nil {
				t.Fatal(err)
			}
			var reads, patches, foreign atomic.Int32
			var rejectNext atomic.Bool
			var captured func()
			cloud.Mux.HandleFunc("GET "+clusteringAsyncTrackedPrefix+"/"+path, func(w http.ResponseWriter, r *http.Request) {
				reads.Add(1)
				if r.URL.Query().Get("name") != "Original" {
					t.Error(r.URL)
				}
				if r.URL.Query().Get("marker") == "" {
					captured()
					nextPath := path
					if rejectNext.Load() {
						nextPath = entry.kind + "s"
					}
					w.Header().Set("Link", fmt.Sprintf(`<%s/%s?name=Original&marker=next>; rel="next"`, clusteringAsyncTrackedPrefix, nextPath))
					testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[{"id":"decoy","name":"Elsewhere"}]}`, entry.kind+"s"))
				} else {
					if r.URL.Query().Get("marker") != "next" {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[{"id":"canonical-id","name":"Original"}]}`, entry.kind+"s"))
				}
			})
			cloud.Mux.HandleFunc("GET "+clusteringAsyncTrackedPrefix+"/"+entry.kind+"s", func(w http.ResponseWriter, r *http.Request) {
				foreign.Add(1)
				w.WriteHeader(500)
			})
			cloud.Mux.HandleFunc("PATCH "+clusteringAsyncTrackedPrefix+"/"+path+"/canonical-id", func(w http.ResponseWriter, r *http.Request) {
				patches.Add(1)
				var body map[string]map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				fields := body[entry.kind]
				if len(body) != 1 || len(fields) != 4 || string(fields["name"]) != `"Requested"` || string(fields["metadata"]) != `{"big":9007199254740993}` || string(fields["vendor"]) != "false" || string(fields[entry.gate]) != "null" || r.Header.Get("X-Snapshot") != "captured" || r.Header.Get("OpenStack-API-Version") != "clustering "+entry.version {
					t.Error(body, r.Header)
				}
				w.Header().Set("Location", "actions/name-action")
				testcloud.JSON(w, 202, clusteringUpdateScopeResponse(entry.kind, `"id":"different-response-id","name":"Accepted"`))
			})
			value, err := scope.update(context.Background(), resource.Name("Original"), "Requested", true, &captured)
			if err != nil || value.operation == nil || value.operation.ActionID != "name-action" || reads.Load() != 2 || patches.Load() != 1 {
				t.Fatal(value, err, reads.Load(), patches.Load())
			}
			rejectNext.Store(true)
			if _, err := scope.update(context.Background(), resource.Name("Original"), "Requested", true, &captured); err == nil || reads.Load() != 3 || patches.Load() != 1 || foreign.Load() != 0 {
				t.Fatal("continuation escaped the selected collection", err, reads.Load(), patches.Load(), foreign.Load())
			}
		})
	}
}

func TestClusteringUpdateScopeTrackedLoadAndTrackKeepScopedIdentity(t *testing.T) {
	for _, entry := range clusteringUpdateScopeCases {
		for _, mode := range []string{"ID", "Name", "Track"} {
			t.Run(entry.kind+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("clustering", clusteringAsyncTrackedPrefix)
				client.Microversion = entry.version
				path := "alternate/" + entry.kind + "s"
				scope, err := entry.new(client, path)
				if err != nil {
					t.Fatal(err)
				}
				fixedID := "canonical-id"
				if mode == "ID" {
					fixedID = "request-id"
				}
				var lists, gets, patches atomic.Int32
				cloud.Mux.HandleFunc("GET "+clusteringAsyncTrackedPrefix+"/"+path, func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					if mode != "Name" || r.URL.Query().Get("name") != "Original" {
						t.Error(r.URL)
					}
					if r.URL.Query().Get("marker") == "" {
						w.Header().Set("Link", fmt.Sprintf(`<%s/%s?name=Original&marker=next>; rel="next"`, clusteringAsyncTrackedPrefix, path))
						testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[{"id":"decoy","Name":"Original"}]}`, entry.kind+"s"))
					} else {
						testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[{"id":"canonical-id","ID":"shadow-id","name":"Original","metadata":{"old":true}}]}`, entry.kind+"s"))
					}
				})
				cloud.Mux.HandleFunc("GET "+clusteringAsyncTrackedPrefix+"/"+path+"/"+fixedID, func(w http.ResponseWriter, r *http.Request) {
					read := gets.Add(1)
					if mode == "ID" && read == 1 {
						testcloud.JSON(w, 200, clusteringUpdateScopeResponse(entry.kind, `"id":"changed-load-id","name":"Original","metadata":{"old":true}`))
					} else {
						w.Header().Set("X-Request-ID", "refresh-scoped")
						testcloud.JSON(w, 200, clusteringUpdateScopeResponse(entry.kind, `"id":null,"ID":"refresh-shadow","name":null,"Name":"refresh-shadow","metadata":{"observed":9007199254740993}`))
					}
				})
				cloud.Mux.HandleFunc("PATCH "+clusteringAsyncTrackedPrefix+"/"+path+"/"+fixedID, func(w http.ResponseWriter, r *http.Request) {
					patches.Add(1)
					body, _ := io.ReadAll(r.Body)
					if string(body) != clusteringUpdateScopeResponse(entry.kind, `"name":"Requested"`) {
						t.Error(string(body))
					}
					w.Header().Set("Location", clusteringAsyncTrackedPrefix+"/actions/tracked-scoped")
					testcloud.JSON(w, 202, clusteringUpdateScopeResponse(entry.kind, `"id":"changed-patch-id","name":"Accepted"`))
				})
				var handle *clusteringUpdateScopeHandle
				switch mode {
				case "ID":
					handle, err = scope.load(context.Background(), resource.ID(fixedID))
				case "Name":
					handle, err = scope.load(context.Background(), resource.Name("Original"))
				case "Track":
					seed := clusteringUpdateScopeSeed(fixedID)
					seed["ID"] = json.RawMessage(`"shadow-id"`)
					handle, err = scope.track(seed)
					seed["id"][1] = 'X'
				}
				if err != nil {
					t.Fatal(err)
				}
				if _, err := handle.commit(context.Background(), nil); err != nil || patches.Load() != 0 {
					t.Fatal("clean commit sent HTTP", err, patches.Load())
				}
				callerValue := handle.value()
				callerValue.metadata.Body["id"] = json.RawMessage(`"caller-retarget"`)
				if err := handle.edit(map[string]any{"name": "Requested"}); err != nil {
					t.Fatal(err)
				}
				accepted, err := handle.commit(context.Background(), nil)
				if err != nil || accepted.operation == nil || accepted.operation.ActionID != "tracked-scoped" || accepted.id != "changed-patch-id" || patches.Load() != 1 {
					t.Fatal(accepted, err, patches.Load())
				}
				refreshed, err := handle.refresh(context.Background())
				if err != nil || refreshed.id != "" || refreshed.name != "" || refreshed.operation != nil || string(refreshed.userMetadata["observed"]) != "9007199254740993" || handle.dirty() {
					t.Fatal(refreshed, err)
				}
				wantLists, wantGets := int32(0), int32(1)
				if mode == "Name" {
					wantLists = 2
				}
				if mode == "ID" {
					wantGets = 2
				}
				if lists.Load() != wantLists || gets.Load() != wantGets {
					t.Fatal(lists.Load(), gets.Load())
				}
			})
		}
	}
}

func TestClusteringUpdateScopePendingNullGatesAndSourceRecheck(t *testing.T) {
	for _, entry := range clusteringUpdateScopeCases {
		t.Run(entry.kind+"/pending", func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("clustering", "/v1")
			client.Microversion = entry.older
			path := "alternate/" + entry.kind + "s"
			scope, err := entry.new(client, path)
			if err != nil {
				t.Fatal(err)
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				count := calls.Add(1)
				if r.Method != http.MethodPatch || r.URL.Path != "/v1/"+path+"/fixed-id" {
					t.Error(r.Method, r.URL)
				}
				var body map[string]map[string]json.RawMessage
				_ = json.NewDecoder(r.Body).Decode(&body)
				if count == 1 {
					if len(body[entry.kind]) != 1 || string(body[entry.kind]["name"]) != `"Requested"` || r.Header.Get("OpenStack-API-Version") != "clustering "+entry.older {
						t.Error(body, r.Header)
					}
				} else if len(body[entry.kind]) != 1 || string(body[entry.kind][entry.gate]) != "null" || r.Header.Get("OpenStack-API-Version") != "clustering "+entry.version {
					t.Error(body, r.Header)
				}
				w.Header().Set("Location", "/v1/actions/null-action")
				testcloud.JSON(w, 202, clusteringUpdateScopeResponse(entry.kind, `"id":"fixed-id"`))
			})
			if _, err := scope.update(context.Background(), resource.Name("Original"), "Requested", true, nil); !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 0 {
				t.Fatal(err, calls.Load())
			}
			seed := clusteringUpdateScopeSeed("fixed-id")
			seed[entry.gate] = json.RawMessage("true")
			handle, err := scope.track(seed)
			if err != nil {
				t.Fatal(err)
			}
			if err := handle.edit(map[string]any{"name": "Requested"}); err != nil {
				t.Fatal(err)
			}
			if _, err := handle.commit(context.Background(), nil); err != nil {
				t.Fatal("cached gate field blocked unrelated edit", err)
			}
			if err := handle.gateNull(); err != nil {
				t.Fatal(err)
			}
			if _, err := handle.commit(context.Background(), nil); !errors.Is(err, resource.ErrUnsupported) || !handle.dirty() || calls.Load() != 1 {
				t.Fatal(err, handle.dirty(), calls.Load())
			}
			client.Microversion = entry.version
			if _, err := handle.commit(context.Background(), nil); err != nil || handle.dirty() || calls.Load() != 2 {
				t.Fatal(err, handle.dirty(), calls.Load())
			}
			client.Type = "compute"
			if _, err := handle.commit(context.Background(), nil); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 2 {
				t.Fatal("clean scope bypassed source validation", err, calls.Load())
			}
		})
		for _, changed := range []string{"type", "version", "header"} {
			t.Run(entry.kind+"/lookup-"+changed, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("clustering", "/v1")
				client.Microversion = entry.version
				path := "alternate/" + entry.kind + "s"
				scope, err := entry.new(client, path)
				if err != nil {
					t.Fatal(err)
				}
				var reads, patches atomic.Int32
				cloud.Mux.HandleFunc("GET /v1/"+path, func(w http.ResponseWriter, r *http.Request) {
					reads.Add(1)
					if changed == "type" {
						client.Type = "compute"
					} else if changed == "version" {
						client.Microversion = entry.older
					} else {
						client.MoreHeaders = map[string]string{"OpenStack-API-Version": "clustering 1.0"}
					}
					testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[{"id":"fixed-id","name":"Original"}]}`, entry.kind+"s"))
				})
				cloud.Mux.HandleFunc("PATCH /v1/"+path+"/fixed-id", func(w http.ResponseWriter, r *http.Request) { patches.Add(1); w.WriteHeader(500) })
				_, err = scope.update(context.Background(), resource.Name("Original"), "Requested", true, nil)
				if err == nil || reads.Load() != 1 || patches.Load() != 0 {
					t.Fatal(err, reads.Load(), patches.Load())
				}
			})
		}
	}
}

func TestClusteringUpdateScopeFailuresRetainAcceptedEvidenceDirtyAndFixedRoute(t *testing.T) {
	for _, entry := range clusteringUpdateScopeCases {
		for _, mode := range []string{"missing-location", "scoped-action-location", "malformed-body", "unexpected-status"} {
			t.Run(entry.kind+"/"+mode, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("clustering", "/v1")
				client.Microversion = entry.version
				path := "alternate/" + entry.kind + "s"
				scope, err := entry.new(client, path)
				if err != nil {
					t.Fatal(err)
				}
				handle, err := scope.track(clusteringUpdateScopeSeed("fixed-id"))
				if err != nil {
					t.Fatal(err)
				}
				if err := handle.edit(map[string]any{"name": "Requested"}); err != nil {
					t.Fatal(err)
				}
				status := 202
				body := clusteringUpdateScopeResponse(entry.kind, `"id":"server-retarget","name":"Accepted","vendor":{"large":9007199254740993}`)
				if mode == "malformed-body" {
					body = fmt.Sprintf(`{%q:[]}`, entry.kind)
				}
				if mode == "unexpected-status" {
					status = 200
				}
				var patches, gets atomic.Int32
				cloud.Mux.HandleFunc("PATCH /v1/"+path+"/fixed-id", func(w http.ResponseWriter, r *http.Request) {
					patches.Add(1)
					w.Header().Set("X-Request-ID", "accepted-failure")
					if mode == "scoped-action-location" {
						w.Header().Set("Location", "/v1/"+path+"/actions/wrong-collection")
					} else if mode != "missing-location" {
						w.Header().Set("Location", "/v1/actions/real-action")
					}
					testcloud.JSON(w, status, body)
				})
				cloud.Mux.HandleFunc("GET /v1/"+path+"/fixed-id", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					testcloud.JSON(w, 500, `{"error":"refresh failure"}`)
				})
				_, err = handle.commit(context.Background(), nil)
				if err == nil || patches.Load() != 1 || !handle.dirty() || handle.value().name != "Requested" || handle.response().name != "Original" || handle.response().operation != nil || handle.response().metadata.StatusCode != 0 {
					t.Fatal(err, patches.Load(), handle.value(), handle.response())
				}
				var accepted *resource.ResponseError
				if mode == "unexpected-status" {
					if !gophercloud.ResponseCodeIs(err, 200) || errors.As(err, &accepted) {
						t.Fatal("unexpected status lost native error", err)
					}
				} else if !errors.As(err, &accepted) || accepted.StatusCode != 202 || string(accepted.Body) != body || accepted.Header.Get("X-Request-ID") != "accepted-failure" {
					t.Fatal("accepted evidence lost", err, accepted)
				}
				if _, err := handle.refresh(context.Background()); !gophercloud.ResponseCodeIs(err, 500) || gets.Load() != 1 || !handle.dirty() || handle.response().name != "Original" {
					t.Fatal(err, gets.Load(), handle.response())
				}
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				if _, err := handle.commit(ctx, nil); !errors.Is(err, context.Canceled) || patches.Load() != 1 {
					t.Fatal(err, patches.Load())
				}
				if _, err := handle.refresh(ctx); !errors.Is(err, context.Canceled) || gets.Load() != 1 {
					t.Fatal(err, gets.Load())
				}
				if _, err := scope.update(ctx, resource.ID("fixed-id"), "Canceled", false, nil); !errors.Is(err, context.Canceled) || patches.Load() != 1 {
					t.Fatal(err, patches.Load())
				}
			})
		}
	}
}
