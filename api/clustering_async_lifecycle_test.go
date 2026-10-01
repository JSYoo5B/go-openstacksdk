package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/clustering/v1/actions"
	"gophercloudsdk/clustering/v1/clusters"
	"gophercloudsdk/clustering/v1/nodes"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// Source: pinned Resource's current-value/sticky dirty tracking and tombstones
// (resource.py:214-246), dirty-only request body (:1231-1249), clean commit
// short circuit (:1914-1919), shallow response merge/reset (:1380-1387).
// Cluster/Node AsyncResource commits PATCH202 and retain Location actions.
// Immutable route, strict accepted evidence and ownership are Go policies.
const clusteringAsyncTrackedPrefix = "/proxy/tenant/senlin/v1"

type clusteringAsyncTrackedView struct {
	id, name     string
	metadata     *resource.Metadata
	userMetadata map[string]json.RawMessage
	config       map[string]json.RawMessage
	operation    *actions.Submission
}

type clusteringAsyncTrackedHandle struct {
	value, response func() clusteringAsyncTrackedView
	dirty           func() bool
	edit            func(map[string]any) error
	remove          func(string) error
	commit          func(context.Context, map[string]string) (clusteringAsyncTrackedView, error)
	refresh         func(context.Context) (clusteringAsyncTrackedView, error)
	badEdit         func(string) error
	badCommit       func(context.Context, string) error
	snapshotEdit    func() error
}

func clusteringAsyncClusterView(value *clusters.Cluster) clusteringAsyncTrackedView {
	if value == nil {
		return clusteringAsyncTrackedView{}
	}
	return clusteringAsyncTrackedView{value.ID, value.Name, &value.Metadata, value.UserMetadata, value.Config, value.Operation}
}
func clusteringAsyncNodeView(value *nodes.Node) clusteringAsyncTrackedView {
	if value == nil {
		return clusteringAsyncTrackedView{}
	}
	return clusteringAsyncTrackedView{value.ID, value.Name, &value.Metadata, value.UserMetadata, nil, value.Operation}
}

func clusteringAsyncClusterHandle(tracked *clusters.TrackedCluster) *clusteringAsyncTrackedHandle {
	return &clusteringAsyncTrackedHandle{
		value:    func() clusteringAsyncTrackedView { return clusteringAsyncClusterView(tracked.Value()) },
		response: func() clusteringAsyncTrackedView { return clusteringAsyncClusterView(tracked.Response()) },
		dirty:    tracked.Dirty,
		edit: func(input map[string]any) error {
			options := []clusters.UpdateOption{}
			for key, value := range input {
				switch key {
				case "name":
					if value == nil {
						options = append(options, clusters.WithUpdateNameNull())
					} else {
						options = append(options, clusters.WithUpdateName(value.(string)))
					}
				case "profile_id":
					if value == nil {
						options = append(options, clusters.WithUpdateProfileIDNull())
					} else {
						options = append(options, clusters.WithUpdateProfileID(value.(string)))
					}
				case "timeout":
					if value == nil {
						options = append(options, clusters.WithUpdateTimeoutNull())
					} else {
						options = append(options, clusters.WithUpdateTimeout(value.(int)))
					}
				case "metadata":
					options = append(options, clusters.WithUpdateMetadata(value))
				case "config":
					options = append(options, clusters.WithUpdateConfig(value))
				case "profile_only":
					options = append(options, clusters.WithUpdateProfileOnly(value.(bool)))
				default:
					options = append(options, clusters.WithUpdateField(key, value))
				}
			}
			return tracked.Edit(clusters.UpdateOpts{}, options...)
		},
		remove: func(field string) error {
			switch field {
			case "name":
				return tracked.RemoveName()
			case "profile_id":
				return tracked.RemoveProfileID()
			case "timeout":
				return tracked.RemoveTimeout()
			case "config":
				return tracked.RemoveConfig()
			case "metadata":
				return tracked.RemoveMetadata()
			case "profile_only":
				return tracked.RemoveProfileOnly()
			default:
				return fmt.Errorf("unsupported test removal %s", field)
			}
		},
		commit: func(ctx context.Context, headers map[string]string) (clusteringAsyncTrackedView, error) {
			options := []clusters.UpdateOption{}
			for key, value := range headers {
				options = append(options, clusters.WithUpdateHeader(key, value))
			}
			value, err := tracked.Commit(ctx, options...)
			return clusteringAsyncClusterView(value), err
		},
		refresh: func(ctx context.Context) (clusteringAsyncTrackedView, error) {
			value, err := tracked.Refresh(ctx)
			return clusteringAsyncClusterView(value), err
		},
		badEdit: func(mode string) error {
			switch mode {
			case "header":
				return tracked.Edit(clusters.UpdateOpts{}, clusters.WithUpdateHeader("X-Vendor", "value"))
			case "query":
				return tracked.Edit(clusters.UpdateOpts{}, request.WithQuery[clusters.UpdateOpts]("vendor", "value"))
			case "argument":
				return tracked.Edit(clusters.UpdateOpts{}, request.WithArgument[clusters.UpdateOpts]("foreign", true))
			case "nil":
				return tracked.Edit(clusters.UpdateOpts{}, nil)
			case "empty-name":
				return tracked.Edit(clusters.UpdateOpts{}, clusters.WithUpdateName(""))
			case "bad-metadata":
				return tracked.Edit(clusters.UpdateOpts{}, clusters.WithUpdateName("Valid"), clusters.WithUpdateMetadata([]any{}))
			case "bad-json":
				return tracked.Edit(clusters.UpdateOpts{}, clusters.WithUpdateName("Valid"), func(config *request.Config[clusters.UpdateOpts]) error {
					config.Fields["vendor"] = json.RawMessage(`{`)
					return nil
				})
			default:
				return tracked.Edit(clusters.UpdateOpts{}, clusters.WithUpdateName("Valid"), clusters.WithUpdateField(mode, false))
			}
		},
		badCommit: func(ctx context.Context, mode string) error {
			var option clusters.UpdateOption
			switch mode {
			case "name":
				option = clusters.WithUpdateNameNull()
			case "zero":
				option = clusters.WithUpdateTimeout(0)
			case "false":
				option = clusters.WithUpdateProfileOnly(false)
			case "metadata":
				option = clusters.WithUpdateMetadata(nil)
			case "field":
				option = clusters.WithUpdateField("vendor", false)
			case "query":
				option = request.WithQuery[clusters.UpdateOpts]("vendor", "value")
			case "argument":
				option = request.WithArgument[clusters.UpdateOpts]("foreign", true)
			case "auth":
				option = clusters.WithUpdateHeader("X-Auth-Token", "override")
			}
			_, err := tracked.Commit(ctx, option)
			return err
		},
		snapshotEdit: func() error {
			metadata := json.RawMessage(`{"big":9007199254740993,"off":false}`)
			config := json.RawMessage(`{"nested":{"off":false}}`)
			flag := false
			option := clusters.WithUpdateOptions(clusters.UpdateOpts{Name: request.Present("Snapshot"), Metadata: metadata, Config: config, Timeout: request.Present(0), ProfileOnly: &flag})
			metadata[2], config[2], flag = 'X', 'X', true
			return tracked.Edit(clusters.UpdateOpts{}, option)
		},
	}
}

func clusteringAsyncNodeHandle(tracked *nodes.TrackedNode) *clusteringAsyncTrackedHandle {
	return &clusteringAsyncTrackedHandle{
		value:    func() clusteringAsyncTrackedView { return clusteringAsyncNodeView(tracked.Value()) },
		response: func() clusteringAsyncTrackedView { return clusteringAsyncNodeView(tracked.Response()) },
		dirty:    tracked.Dirty,
		edit: func(input map[string]any) error {
			options := []nodes.UpdateOption{}
			for key, value := range input {
				switch key {
				case "name":
					if value == nil {
						options = append(options, nodes.WithUpdateNameNull())
					} else {
						options = append(options, nodes.WithUpdateName(value.(string)))
					}
				case "profile_id":
					if value == nil {
						options = append(options, nodes.WithUpdateProfileIDNull())
					} else {
						options = append(options, nodes.WithUpdateProfileID(value.(string)))
					}
				case "role":
					if value == nil {
						options = append(options, nodes.WithUpdateRoleNull())
					} else {
						options = append(options, nodes.WithUpdateRole(value.(string)))
					}
				case "tainted":
					if value == nil {
						options = append(options, nodes.WithUpdateTaintedNull())
					} else {
						options = append(options, nodes.WithUpdateTainted(value.(bool)))
					}
				case "metadata":
					options = append(options, nodes.WithUpdateMetadata(value))
				default:
					options = append(options, nodes.WithUpdateField(key, value))
				}
			}
			return tracked.Edit(nodes.UpdateOpts{}, options...)
		},
		remove: func(field string) error {
			switch field {
			case "name":
				return tracked.RemoveName()
			case "profile_id":
				return tracked.RemoveProfileID()
			case "role":
				return tracked.RemoveRole()
			case "metadata":
				return tracked.RemoveMetadata()
			case "tainted":
				return tracked.RemoveTainted()
			default:
				return fmt.Errorf("unsupported test removal %s", field)
			}
		},
		commit: func(ctx context.Context, headers map[string]string) (clusteringAsyncTrackedView, error) {
			options := []nodes.UpdateOption{}
			for key, value := range headers {
				options = append(options, nodes.WithUpdateHeader(key, value))
			}
			value, err := tracked.Commit(ctx, options...)
			return clusteringAsyncNodeView(value), err
		},
		refresh: func(ctx context.Context) (clusteringAsyncTrackedView, error) {
			value, err := tracked.Refresh(ctx)
			return clusteringAsyncNodeView(value), err
		},
		badEdit: func(mode string) error {
			switch mode {
			case "header":
				return tracked.Edit(nodes.UpdateOpts{}, nodes.WithUpdateHeader("X-Vendor", "value"))
			case "query":
				return tracked.Edit(nodes.UpdateOpts{}, request.WithQuery[nodes.UpdateOpts]("vendor", "value"))
			case "argument":
				return tracked.Edit(nodes.UpdateOpts{}, request.WithArgument[nodes.UpdateOpts]("foreign", true))
			case "nil":
				return tracked.Edit(nodes.UpdateOpts{}, nil)
			case "empty-name":
				return tracked.Edit(nodes.UpdateOpts{}, nodes.WithUpdateName(""))
			case "bad-metadata":
				return tracked.Edit(nodes.UpdateOpts{}, nodes.WithUpdateName("Valid"), nodes.WithUpdateMetadata([]any{}))
			case "bad-json":
				return tracked.Edit(nodes.UpdateOpts{}, nodes.WithUpdateName("Valid"), func(config *request.Config[nodes.UpdateOpts]) error {
					config.Fields["vendor"] = json.RawMessage(`{`)
					return nil
				})
			default:
				return tracked.Edit(nodes.UpdateOpts{}, nodes.WithUpdateName("Valid"), nodes.WithUpdateField(mode, false))
			}
		},
		badCommit: func(ctx context.Context, mode string) error {
			var option nodes.UpdateOption
			switch mode {
			case "name":
				option = nodes.WithUpdateNameNull()
			case "zero":
				option = nodes.WithUpdateRole("")
			case "false":
				option = nodes.WithUpdateTainted(false)
			case "metadata":
				option = nodes.WithUpdateMetadata(nil)
			case "field":
				option = nodes.WithUpdateField("vendor", false)
			case "query":
				option = request.WithQuery[nodes.UpdateOpts]("vendor", "value")
			case "argument":
				option = request.WithArgument[nodes.UpdateOpts]("foreign", true)
			case "auth":
				option = nodes.WithUpdateHeader("X-Auth-Token", "override")
			}
			_, err := tracked.Commit(ctx, option)
			return err
		},
		snapshotEdit: func() error {
			metadata := json.RawMessage(`{"big":9007199254740993,"off":false}`)
			option := nodes.WithUpdateOptions(nodes.UpdateOpts{Name: request.Present("Snapshot"), Metadata: metadata, Tainted: request.Present(false), Role: request.Present("")})
			metadata[2] = 'X'
			return tracked.Edit(nodes.UpdateOpts{}, option)
		},
	}
}

func clusteringAsyncTrackedPlural(kind string) string {
	if kind == "cluster" {
		return "clusters"
	}
	return "nodes"
}
func clusteringAsyncTrackedSeed(kind string) string {
	extra := `,"role":"role","tainted":true,"physical_id":"physical"`
	if kind == "cluster" {
		extra = `,"timeout":30,"config":{"nested":{"off":false}},"profile_only":true,"nodes":["node-id"]`
	}
	return `{"id":"fixed","name":"A","profile_id":"profile","project":"project","status":"ACTIVE","metadata":{"a":1,"b":2},"future":{"big":9007199254740993,"off":false}` + extra + `}`
}
func clusteringAsyncTrackedCloud(t *testing.T) (*testcloud.Cloud, *gophercloud.ServiceClient) {
	t.Helper()
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/catalog/v1")
	client.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + clusteringAsyncTrackedPrefix)
	client.Microversion = "1.13"
	return cloud, client
}
func clusteringAsyncTrackedObject(kind, body string) string {
	return fmt.Sprintf(`{%q:%s}`, kind, body)
}

func clusteringAsyncTrackedTrack(client *gophercloud.ServiceClient, kind, body string, operation *actions.Submission) (*clusteringAsyncTrackedHandle, error) {
	if kind == "cluster" {
		var value clusters.Cluster
		if err := json.Unmarshal([]byte(body), &value); err != nil {
			return nil, err
		}
		value.Header = http.Header{"X-Request-Id": {"seed"}}
		value.StatusCode = 200
		value.Operation = operation
		tracked, err := clusters.New(client).Track(&value)
		if err != nil {
			return nil, err
		}
		return clusteringAsyncClusterHandle(tracked), nil
	}
	var value nodes.Node
	if err := json.Unmarshal([]byte(body), &value); err != nil {
		return nil, err
	}
	value.Header = http.Header{"X-Request-Id": {"seed"}}
	value.StatusCode = 200
	value.Operation = operation
	tracked, err := nodes.New(client).Track(&value)
	if err != nil {
		return nil, err
	}
	return clusteringAsyncNodeHandle(tracked), nil
}
func clusteringAsyncTrackedLoad(ctx context.Context, client *gophercloud.ServiceClient, kind string, ref resource.Ref) (*clusteringAsyncTrackedHandle, error) {
	if kind == "cluster" {
		tracked, err := clusters.New(client).Load(ctx, ref)
		if err != nil {
			return nil, err
		}
		return clusteringAsyncClusterHandle(tracked), nil
	}
	tracked, err := nodes.New(client).Load(ctx, ref)
	if err != nil {
		return nil, err
	}
	return clusteringAsyncNodeHandle(tracked), nil
}
func clusteringAsyncTrackedMust(t *testing.T, client *gophercloud.ServiceClient, kind string) *clusteringAsyncTrackedHandle {
	t.Helper()
	handle, err := clusteringAsyncTrackedTrack(client, kind, clusteringAsyncTrackedSeed(kind), nil)
	if err != nil {
		t.Fatal(err)
	}
	return handle
}

func clusteringAsyncTrackedWire(t *testing.T, r *http.Request, kind string) map[string]json.RawMessage {
	t.Helper()
	if r.Method != http.MethodPatch || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" {
		t.Error(r.Method, r.URL, r.Header)
	}
	var envelope map[string]map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
		t.Error(err)
	}
	if len(envelope) != 1 || envelope[kind] == nil {
		t.Error("wrong patch envelope", envelope)
	}
	return envelope[kind]
}
func clusteringAsyncTrackedAccept(w http.ResponseWriter, kind, body, action string) {
	w.Header().Set("Location", clusteringAsyncTrackedPrefix+"/actions/"+action)
	w.Header().Set("X-Request-ID", action)
	testcloud.JSON(w, 202, clusteringAsyncTrackedObject(kind, body))
}
func clusteringAsyncTrackedAssertOperation(t *testing.T, value clusteringAsyncTrackedView, kind, body, action string) {
	t.Helper()
	if value.operation == nil || value.operation.ActionID != action || value.operation.Location != clusteringAsyncTrackedPrefix+"/actions/"+action || value.operation.StatusCode != 202 || string(value.operation.Body) != clusteringAsyncTrackedObject(kind, body) || value.operation.Header.Get("X-Request-ID") != action {
		t.Fatal("async submission evidence lost", value.operation)
	}
}

func TestClusteringAsyncTrackedNoOpStickyDirtyShallowMergeAndActualResponse(t *testing.T) {
	for _, kind := range []string{"cluster", "node"} {
		t.Run(kind, func(t *testing.T) {
			cloud, client := clusteringAsyncTrackedCloud(t)
			var patches, gets atomic.Int32
			path := clusteringAsyncTrackedPrefix + "/" + clusteringAsyncTrackedPlural(kind) + "/fixed"
			cloud.Mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				t.Error("tracked commit implicitly fetched or polled", r.URL)
				w.WriteHeader(500)
			})
			cloud.Mux.HandleFunc("PATCH "+path, func(w http.ResponseWriter, r *http.Request) {
				body := clusteringAsyncTrackedWire(t, r, kind)
				call := patches.Add(1)
				if r.Header.Get("X-Vendor") != "owned" {
					t.Error(r.Header)
				}
				if call == 1 {
					if len(body) != 1 || string(body["name"]) != `"A"` {
						t.Error(body)
					}
					clusteringAsyncTrackedAccept(w, kind, `{"name":"Normalized","metadata":{"a":3}}`, "action-one")
				} else {
					if len(body) != 2 || string(body["name"]) != `"B"` || string(body["vendor"]) != `{"big":9007199254740995,"off":false}` {
						t.Error(body)
					}
					clusteringAsyncTrackedAccept(w, kind, `{}`, "action-two")
				}
			})
			handle := clusteringAsyncTrackedMust(t, client, kind)
			if err := handle.edit(nil); err != nil {
				t.Fatal(err)
			}
			if err := handle.edit(map[string]any{"name": "A"}); err != nil {
				t.Fatal(err)
			}
			if _, err := handle.commit(context.Background(), map[string]string{"X-Vendor": "owned"}); err != nil || handle.dirty() || patches.Load() != 0 {
				t.Fatal(err, handle.dirty(), patches.Load())
			}
			for _, name := range []string{"B", "A", "A"} {
				if err := handle.edit(map[string]any{"name": name}); err != nil {
					t.Fatal(err)
				}
			}
			if !handle.dirty() {
				t.Fatal("sticky A→B→A was cleared")
			}
			value, err := handle.commit(context.Background(), map[string]string{"X-Vendor": "owned"})
			if err != nil || handle.dirty() || value.id != "fixed" || value.name != "Normalized" || string(value.metadata.Body["project"]) != `"project"` || string(value.metadata.Body["future"]) != `{"big":9007199254740993,"off":false}` || len(value.userMetadata) != 1 || string(value.userMetadata["a"]) != "3" {
				t.Fatal("cache merge/clean contract failed", value, err)
			}
			clusteringAsyncTrackedAssertOperation(t, value, kind, `{"name":"Normalized","metadata":{"a":3}}`, "action-one")
			actual := handle.response()
			if actual.id != "" || len(actual.metadata.Body) != 2 || actual.metadata.Header.Get("X-Request-ID") != "action-one" {
				t.Fatal("actual response contains merged cache fields", actual)
			}
			clusteringAsyncTrackedAssertOperation(t, actual, kind, `{"name":"Normalized","metadata":{"a":3}}`, "action-one")
			vendor := map[string]any{"big": json.Number("9007199254740995"), "off": false}
			if err := handle.edit(map[string]any{"name": "B", "vendor": vendor}); err != nil {
				t.Fatal(err)
			}
			vendor["off"] = true
			value, err = handle.commit(context.Background(), map[string]string{"X-Vendor": "owned"})
			if err != nil || handle.dirty() || value.name != "B" || string(value.metadata.Body["vendor"]) != `{"big":9007199254740995,"off":false}` || len(handle.response().metadata.Body) != 0 {
				t.Fatal("omitted edited fields were not accepted/cleaned", value, err)
			}
			clusteringAsyncTrackedAssertOperation(t, value, kind, `{}`, "action-two")
			if _, err := handle.commit(context.Background(), nil); err != nil || patches.Load() != 2 || gets.Load() != 0 {
				t.Fatal("clean commit made HTTP or action follow", err, patches.Load(), gets.Load())
			}
		})
	}
}

func TestClusteringAsyncTrackedScalarPresenceAndRemovalTombstones(t *testing.T) {
	for _, kind := range []string{"cluster", "node"} {
		t.Run(kind+"/explicit-values", func(t *testing.T) {
			cloud, client := clusteringAsyncTrackedCloud(t)
			var patches atomic.Int32
			cloud.Mux.HandleFunc("PATCH "+clusteringAsyncTrackedPrefix+"/"+clusteringAsyncTrackedPlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) {
				body := clusteringAsyncTrackedWire(t, r, kind)
				call := patches.Add(1)
				want := map[string]string{"name": "null", "profile_id": "null", "metadata": "{}"}
				if kind == "cluster" {
					want["timeout"], want["config"], want["profile_only"] = "0", "{}", "false"
				} else {
					want["role"], want["tainted"] = `""`, "false"
				}
				if call == 2 {
					want = map[string]string{"name": `"Restored"`, "profile_id": `"new-profile"`, "metadata": "null"}
					if kind == "cluster" {
						want["timeout"], want["config"] = "null", "null"
					} else {
						want["role"], want["tainted"] = "null", "null"
					}
				}
				if len(body) != len(want) {
					t.Error(body, want)
				}
				for key, raw := range want {
					if string(body[key]) != raw {
						t.Error(key, string(body[key]), raw)
					}
				}
				clusteringAsyncTrackedAccept(w, kind, `{}`, "accepted")
			})
			handle := clusteringAsyncTrackedMust(t, client, kind)
			input := map[string]any{"name": nil, "profile_id": nil, "metadata": map[string]any{}}
			if kind == "cluster" {
				input["timeout"], input["config"], input["profile_only"] = 0, map[string]any{}, false
			} else {
				input["role"], input["tainted"] = "", false
			}
			if err := handle.edit(input); err != nil {
				t.Fatal(err)
			}
			if _, err := handle.commit(context.Background(), nil); err != nil || handle.dirty() {
				t.Fatal(err)
			}
			input = map[string]any{"name": "Restored", "profile_id": "new-profile", "metadata": nil}
			if kind == "cluster" {
				input["timeout"], input["config"] = nil, nil
			} else {
				input["role"], input["tainted"] = nil, nil
			}
			if err := handle.edit(input); err != nil {
				t.Fatal(err)
			}
			if _, err := handle.commit(context.Background(), nil); err != nil || handle.dirty() || patches.Load() != 2 {
				t.Fatal(err, patches.Load())
			}
		})
		fields := []string{"name", "profile_id", "metadata", "role", "tainted"}
		if kind == "cluster" {
			fields = []string{"name", "profile_id", "timeout", "config", "metadata", "profile_only"}
		}
		for _, field := range fields {
			t.Run(kind+"/remove-"+field, func(t *testing.T) {
				cloud, client := clusteringAsyncTrackedCloud(t)
				var patches atomic.Int32
				cloud.Mux.HandleFunc("PATCH "+clusteringAsyncTrackedPrefix+"/"+clusteringAsyncTrackedPlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) {
					patches.Add(1)
					body := clusteringAsyncTrackedWire(t, r, kind)
					if len(body) != 1 || string(body[field]) != "null" {
						t.Error(body)
					}
					clusteringAsyncTrackedAccept(w, kind, `{}`, "removed")
				})
				handle := clusteringAsyncTrackedMust(t, client, kind)
				if err := handle.remove(field); err != nil {
					t.Fatal(err)
				}
				if _, exists := handle.value().metadata.Body[field]; exists || !handle.dirty() {
					t.Fatal("removed field still present", field, handle.value())
				}
				if _, err := handle.commit(context.Background(), nil); err != nil || handle.dirty() {
					t.Fatal(err)
				}
				if err := handle.remove(field); err != nil {
					t.Fatal(err)
				}
				if _, err := handle.commit(context.Background(), nil); err != nil || patches.Load() != 1 {
					t.Fatal("removing absent field repeated patch", err, patches.Load())
				}
			})
		}
	}
}

func TestClusteringAsyncTrackedInvalidEditsAndHeaderOnlyCommitAreAtomic(t *testing.T) {
	for _, kind := range []string{"cluster", "node"} {
		t.Run(kind, func(t *testing.T) {
			cloud, client := clusteringAsyncTrackedCloud(t)
			var requests atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(500) })
			handle := clusteringAsyncTrackedMust(t, client, kind)
			modes := []string{"id", "ID", "name", "NAME", "profile_id", "PROFILE_ID", "metadata", "Metadata", "project", "status", "data", "profile_name", "created_at", "header", "query", "argument", "nil", "empty-name", "bad-metadata", "bad-json"}
			if kind == "cluster" {
				modes = append(modes, "nodes", "desired_capacity", "min_size", "max_size", "config", "Config", "timeout", "Timeout", "profile_only", "PROFILE_ONLY")
			} else {
				modes = append(modes, "physical_id", "index", "cluster_id", "role", "Role", "tainted", "Tainted")
			}
			before := handle.value().metadata.Body
			for _, mode := range modes {
				err := handle.badEdit(mode)
				// A malformed RawMessage preserves the JSON encoder's cause; SDK
				// option/collision validation uses ErrInvalidOption.
				if err == nil || (mode != "bad-json" && !errors.Is(err, resource.ErrInvalidOption)) || handle.dirty() || !reflect.DeepEqual(handle.value().metadata.Body, before) {
					t.Fatal("failed edit partially mutated cache", mode, err, handle.value())
				}
			}
			for _, mode := range []string{"name", "zero", "false", "metadata", "field", "query", "argument", "auth", "nil"} {
				if err := handle.badCommit(context.Background(), mode); !errors.Is(err, resource.ErrInvalidOption) || handle.dirty() {
					t.Fatal("clean commit bypassed header-only validation", mode, err)
				}
			}
			if requests.Load() != 0 {
				t.Fatal(requests.Load())
			}
		})
	}
}

func TestClusteringAsyncTrackedModelAndOperationDeepOwnership(t *testing.T) {
	for _, kind := range []string{"cluster", "node"} {
		t.Run(kind, func(t *testing.T) {
			cloud, client := clusteringAsyncTrackedCloud(t)
			var requests, patches atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(500) })
			operation := &actions.Submission{ActionID: "seed-action", Location: "actions/seed-action", Body: json.RawMessage(`{"seed":9007199254740993}`), Header: http.Header{"X-Operation": {"seed"}}, StatusCode: 202}
			cloud.Mux.HandleFunc("PATCH "+clusteringAsyncTrackedPrefix+"/"+clusteringAsyncTrackedPlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) {
				patches.Add(1)
				clusteringAsyncTrackedWire(t, r, kind)
				clusteringAsyncTrackedAccept(w, kind, `{}`, "owned-route")
			})
			body := clusteringAsyncTrackedSeed(kind)
			var handle *clusteringAsyncTrackedHandle
			if kind == "cluster" {
				var model clusters.Cluster
				if err := json.Unmarshal([]byte(body), &model); err != nil {
					t.Fatal(err)
				}
				model.ID = "typed-shadow"
				model.Body["ID"] = json.RawMessage(`"case-shadow"`)
				model.Header = http.Header{"X-Request-Id": {"seed"}}
				model.StatusCode = 200
				model.Operation = operation
				tracked, err := clusters.New(client).Track(&model)
				if err != nil {
					t.Fatal(err)
				}
				handle = clusteringAsyncClusterHandle(tracked)
				model.ID = "mutated"
				model.Body["id"] = json.RawMessage(`"mutated"`)
				model.UserMetadata["a"][0] = '9'
				model.Header.Set("X-Request-ID", "mutated")
			} else {
				var model nodes.Node
				if err := json.Unmarshal([]byte(body), &model); err != nil {
					t.Fatal(err)
				}
				model.ID = "typed-shadow"
				model.Body["ID"] = json.RawMessage(`"case-shadow"`)
				model.Header = http.Header{"X-Request-Id": {"seed"}}
				model.StatusCode = 200
				model.Operation = operation
				tracked, err := nodes.New(client).Track(&model)
				if err != nil {
					t.Fatal(err)
				}
				handle = clusteringAsyncNodeHandle(tracked)
				model.ID = "mutated"
				model.Body["id"] = json.RawMessage(`"mutated"`)
				model.UserMetadata["a"][0] = '9'
				model.Header.Set("X-Request-ID", "mutated")
			}
			operation.ActionID = "mutated"
			operation.Body[2] = 'X'
			operation.Header.Set("X-Operation", "mutated")
			for _, get := range []func() clusteringAsyncTrackedView{handle.value, handle.response} {
				snapshot := get()
				if snapshot.id != "fixed" || snapshot.name != "A" || string(snapshot.userMetadata["a"]) != "1" || snapshot.metadata.Header.Get("X-Request-ID") != "seed" || snapshot.operation == nil || snapshot.operation.ActionID != "seed-action" || string(snapshot.operation.Body) != `{"seed":9007199254740993}` || snapshot.operation.Header.Get("X-Operation") != "seed" {
					t.Fatal("input alias mutated owned cache/evidence", snapshot)
				}
				snapshot.metadata.Body["id"][1] = 'X'
				snapshot.metadata.Header.Set("X-Request-ID", "getter-change")
				snapshot.userMetadata["a"][0] = '7'
				snapshot.operation.Body[2] = 'X'
				snapshot.operation.ActionID = "getter-change"
				snapshot.operation.Header.Set("X-Operation", "getter-change")
			}
			if handle.value().id != "fixed" || handle.response().operation.ActionID != "seed-action" || handle.value().metadata.Header.Get("X-Request-ID") != "seed" || string(handle.value().userMetadata["a"]) != "1" {
				t.Fatal("getter aliases owned state", handle.value(), handle.response())
			}
			if err := handle.snapshotEdit(); err != nil {
				t.Fatal(err)
			}
			snapshot := handle.value()
			if snapshot.name != "Snapshot" || string(snapshot.userMetadata["big"]) != "9007199254740993" || string(snapshot.userMetadata["off"]) != "false" {
				t.Fatal("UpdateOpts snapshot aliased inputs", snapshot)
			}
			if kind == "cluster" && (string(snapshot.metadata.Body["timeout"]) != "0" || string(snapshot.metadata.Body["profile_only"]) != "false" || string(snapshot.config["nested"]) != `{"off":false}`) {
				t.Fatal(snapshot)
			}
			if kind == "node" && (string(snapshot.metadata.Body["tainted"]) != "false" || string(snapshot.metadata.Body["role"]) != `""`) {
				t.Fatal(snapshot)
			}
			if requests.Load() != 0 {
				t.Fatal("Track/Edit performed HTTP", requests.Load())
			}
			if _, err := handle.commit(context.Background(), nil); err != nil || handle.dirty() || patches.Load() != 1 || requests.Load() != 0 {
				t.Fatal("input/getter ID aliases changed the canonical fixed route", err, patches.Load(), requests.Load())
			}
		})
	}
}

func TestClusteringAsyncTrackedFixedIDAndNameLookupOnce(t *testing.T) {
	for _, kind := range []string{"cluster", "node"} {
		for _, byName := range []bool{false, true} {
			for _, responseID := range []string{`"changed"`, `null`} {
				t.Run(fmt.Sprintf("%s/name=%t/response-id=%s", kind, byName, responseID), func(t *testing.T) {
					cloud, client := clusteringAsyncTrackedCloud(t)
					plural := clusteringAsyncTrackedPlural(kind)
					route := "requested"
					if byName {
						route = "canonical"
					}
					var gets, lists, patches atomic.Int32
					cloud.Mux.HandleFunc("GET "+clusteringAsyncTrackedPrefix+"/"+plural+"/requested", func(w http.ResponseWriter, r *http.Request) {
						gets.Add(1)
						testcloud.JSON(w, 200, clusteringAsyncTrackedObject(kind, `{"id":"canonical","ID":"case-shadow","name":"A"}`))
					})
					cloud.Mux.HandleFunc("GET "+clusteringAsyncTrackedPrefix+"/"+plural, func(w http.ResponseWriter, r *http.Request) {
						lists.Add(1)
						if r.URL.Query().Get("name") != "selected" {
							t.Error(r.URL)
						}
						testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[{"id":"case-only","Name":"selected"},{"id":"null-name","name":null,"Name":"selected"},{"id":"decoy","name":"another","Name":"selected"},{"id":"canonical","ID":"case-shadow","name":"selected","Name":"case-shadow"}]}`, plural))
					})
					cloud.Mux.HandleFunc("PATCH "+clusteringAsyncTrackedPrefix+"/"+plural+"/"+route, func(w http.ResponseWriter, r *http.Request) {
						patches.Add(1)
						clusteringAsyncTrackedWire(t, r, kind)
						clusteringAsyncTrackedAccept(w, kind, `{"id":`+responseID+`,"name":"Server"}`, "fixed-route")
					})
					ref := resource.ID("requested")
					if byName {
						ref = resource.Name("selected")
					}
					handle, err := clusteringAsyncTrackedLoad(context.Background(), client, kind, ref)
					if err != nil || handle.value().id != "canonical" {
						t.Fatal(handle, err)
					}
					for _, name := range []string{"B", "C"} {
						if err := handle.edit(map[string]any{"name": name}); err != nil {
							t.Fatal(err)
						}
						if _, err := handle.commit(context.Background(), nil); err != nil {
							t.Fatal(err)
						}
					}
					wantGets, wantLists := int32(1), int32(0)
					if byName {
						wantGets, wantLists = 0, 1
					}
					if gets.Load() != wantGets || lists.Load() != wantLists || patches.Load() != 2 {
						t.Fatal("response identity changed route or Name resolved again", gets.Load(), lists.Load(), patches.Load())
					}
				})
			}
		}
	}
	for _, kind := range []string{"cluster", "node"} {
		for _, initial := range []struct{ name, body, id, resourceName string }{
			{"omitted", `{}`, "", ""},
			{"null-with-case-shadows", `{"id":null,"ID":"case-shadow","name":null,"Name":"case-shadow"}`, "", ""},
			{"different-with-case-shadows", `{"id":"different","ID":"case-shadow","name":"A","Name":"case-shadow"}`, "different", "A"},
		} {
			t.Run(kind+"/explicit-load-"+initial.name, func(t *testing.T) {
				cloud, client := clusteringAsyncTrackedCloud(t)
				var gets, patches atomic.Int32
				path := clusteringAsyncTrackedPrefix + "/" + clusteringAsyncTrackedPlural(kind) + "/requested"
				cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
					if gets.Add(1) == 1 {
						testcloud.JSON(w, 200, clusteringAsyncTrackedObject(kind, initial.body))
					} else {
						testcloud.JSON(w, 200, clusteringAsyncTrackedObject(kind, `{"id":null,"name":"Refreshed"}`))
					}
				})
				cloud.Mux.HandleFunc("PATCH "+path, func(w http.ResponseWriter, r *http.Request) {
					patches.Add(1)
					clusteringAsyncTrackedAccept(w, kind, `{"id":"changed","name":"Accepted"}`, "explicit-route")
				})
				handle, err := clusteringAsyncTrackedLoad(context.Background(), client, kind, resource.ID("requested"))
				if err != nil || handle.value().id != initial.id || handle.value().name != initial.resourceName {
					t.Fatal("explicit route required a returned ID or accepted a case shadow", handle, err)
				}
				if err := handle.edit(map[string]any{"name": "Changed"}); err != nil {
					t.Fatal(err)
				}
				if _, err := handle.commit(context.Background(), nil); err != nil {
					t.Fatal(err)
				}
				if value, err := handle.refresh(context.Background()); err != nil || value.name != "Refreshed" || value.operation != nil || gets.Load() != 2 || patches.Load() != 1 {
					t.Fatal("explicit requested route was lost after changed response ID", value, err, gets.Load(), patches.Load())
				}
			})
		}
	}
}

func TestClusteringAsyncTrackedFailuresKeepDirtyAndAcceptedEvidence(t *testing.T) {
	fixtures := []struct {
		name           string
		code           int
		body, location string
	}{
		{"forbidden", 403, `{"error":"denied"}`, ""},
		{"conflict", 409, `{"error":"conflict"}`, ""},
		{"wrong-success-code", 200, `{}`, ""},
		{"accepted-invalid-json", 202, `{`, "actions/accepted"},
		{"accepted-wrong-envelope", 202, `{"wrong":{}}`, "actions/accepted"},
		{"accepted-null-resource", 202, `RESOURCE_NULL`, "actions/accepted"},
		{"accepted-bad-type", 202, `RESOURCE_BAD`, "actions/accepted"},
		{"accepted-empty-id", 202, `RESOURCE_EMPTYID`, "actions/accepted"},
		{"accepted-unsafe-id", 202, `RESOURCE_UNSAFEID`, "actions/accepted"},
		{"accepted-case-alias-type", 202, `RESOURCE_CASE_TYPE`, "actions/accepted"},
		{"missing-location", 202, `RESOURCE_VALID`, ""},
		{"duplicate-location", 202, `RESOURCE_VALID`, "actions/accepted"},
		{"foreign-location", 202, `RESOURCE_VALID`, "https://example.invalid/actions/accepted"},
		{"wrong-location-path", 202, `RESOURCE_VALID`, "profiles/accepted"},
		{"location-query", 202, `RESOURCE_VALID`, "actions/accepted?query=bad"},
	}
	for _, kind := range []string{"cluster", "node"} {
		for _, fixture := range fixtures {
			t.Run(kind+"/"+fixture.name, func(t *testing.T) {
				cloud, client := clusteringAsyncTrackedCloud(t)
				body := fixture.body
				switch body {
				case "RESOURCE_NULL":
					body = clusteringAsyncTrackedObject(kind, `null`)
				case "RESOURCE_BAD":
					body = clusteringAsyncTrackedObject(kind, `{"id":7}`)
				case "RESOURCE_EMPTYID":
					body = clusteringAsyncTrackedObject(kind, `{"id":"","name":"Server"}`)
				case "RESOURCE_UNSAFEID":
					body = clusteringAsyncTrackedObject(kind, `{"id":"../invalid","name":"Server"}`)
				case "RESOURCE_CASE_TYPE":
					body = clusteringAsyncTrackedObject(kind, `{"id":"fixed","ID":7}`)
				case "RESOURCE_VALID":
					body = clusteringAsyncTrackedObject(kind, `{"name":"Server"}`)
				}
				var patches, gets atomic.Int32
				cloud.Mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); w.WriteHeader(500) })
				cloud.Mux.HandleFunc("PATCH "+clusteringAsyncTrackedPrefix+"/"+clusteringAsyncTrackedPlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) {
					patches.Add(1)
					w.Header().Set("X-Request-ID", "failed-commit")
					if fixture.location != "" {
						w.Header().Set("Location", fixture.location)
						if fixture.name == "duplicate-location" {
							w.Header().Add("Location", "actions/second")
						}
					}
					testcloud.JSON(w, fixture.code, body)
				})
				handle := clusteringAsyncTrackedMust(t, client, kind)
				if err := handle.edit(map[string]any{"name": "B"}); err != nil {
					t.Fatal(err)
				}
				value, err := handle.commit(context.Background(), nil)
				if fixture.code == 202 {
					var evidence *resource.ResponseError
					if !errors.As(err, &evidence) || evidence.StatusCode != 202 || string(evidence.Body) != body || evidence.Header.Get("X-Request-ID") != "failed-commit" || evidence.Header.Get("Location") != fixture.location {
						t.Fatal("accepted failure evidence lost", err, evidence)
					}
				} else if !gophercloud.ResponseCodeIs(err, fixture.code) {
					t.Fatal("native error lost", err)
				}
				if value.metadata != nil || !handle.dirty() || handle.value().name != "B" || handle.response().name != "A" || handle.response().operation != nil || patches.Load() != 1 || gets.Load() != 0 {
					t.Fatal("failure accepted/replayed/followed action", value, handle.value(), handle.response(), patches.Load(), gets.Load())
				}
			})
		}
		t.Run(kind+"/native-read-and-transport", func(t *testing.T) {
			for _, accepted := range []bool{false, true} {
				t.Run(fmt.Sprintf("accepted=%t", accepted), func(t *testing.T) {
					cloud, client := clusteringAsyncTrackedCloud(t)
					handle := clusteringAsyncTrackedMust(t, client, kind)
					if err := handle.edit(map[string]any{"name": "B"}); err != nil {
						t.Fatal(err)
					}
					cause := errors.New("original async lifecycle I/O failure")
					body := clusteringAsyncTrackedObject(kind, `{"name":"Server"}`)
					var calls atomic.Int32
					cloud.Provider.HTTPClient.Transport = clusteringAsyncTrackedTransport(func(r *http.Request) (*http.Response, error) {
						calls.Add(1)
						if r.Method != http.MethodPatch {
							t.Error(r.Method)
						}
						if !accepted {
							return nil, cause
						}
						return &http.Response{StatusCode: 202, Header: http.Header{"Location": {"actions/accepted"}, "X-Request-Id": {"read-failure"}}, Body: &clusteringAsyncTrackedReadFailure{[]byte(body), cause}, Request: r}, nil
					})
					_, err := handle.commit(context.Background(), nil)
					if !errors.Is(err, cause) || !handle.dirty() || calls.Load() != 1 || handle.response().operation != nil {
						t.Fatal(err, handle.dirty(), calls.Load())
					}
					if accepted {
						var evidence *resource.ResponseError
						if !errors.As(err, &evidence) || evidence.StatusCode != 202 || string(evidence.Body) != body || evidence.Header.Get("X-Request-ID") != "read-failure" {
							t.Fatal(err, evidence)
						}
					}
				})
			}
		})
	}
}

type clusteringAsyncTrackedTransport func(*http.Request) (*http.Response, error)

func (run clusteringAsyncTrackedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return run(r)
}

type clusteringAsyncTrackedReadFailure struct {
	body  []byte
	cause error
}

func (reader *clusteringAsyncTrackedReadFailure) Read(buffer []byte) (int, error) {
	count := copy(buffer, reader.body)
	reader.body = reader.body[count:]
	return count, reader.cause
}
func (*clusteringAsyncTrackedReadFailure) Close() error { return nil }

func TestClusteringAsyncTrackedRefreshResetsOperationAndFailureKeepsChanges(t *testing.T) {
	for _, kind := range []string{"cluster", "node"} {
		t.Run(kind, func(t *testing.T) {
			cloud, client := clusteringAsyncTrackedCloud(t)
			var gets, patches atomic.Int32
			path := clusteringAsyncTrackedPrefix + "/" + clusteringAsyncTrackedPlural(kind) + "/fixed"
			cloud.Mux.HandleFunc("PATCH "+path, func(w http.ResponseWriter, r *http.Request) {
				patches.Add(1)
				clusteringAsyncTrackedAccept(w, kind, `{"name":"Accepted"}`, "last-action")
			})
			cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
				if gets.Add(1) == 1 {
					w.Header().Set("X-Request-ID", "refreshed")
					testcloud.JSON(w, 200, clusteringAsyncTrackedObject(kind, `{"id":null,"name":"Refreshed","metadata":{"new":false}}`))
				} else {
					testcloud.JSON(w, 403, `{"error":"refresh-denied"}`)
				}
			})
			handle := clusteringAsyncTrackedMust(t, client, kind)
			if err := handle.edit(map[string]any{"name": "B"}); err != nil {
				t.Fatal(err)
			}
			if _, err := handle.commit(context.Background(), nil); err != nil {
				t.Fatal(err)
			}
			if err := handle.edit(map[string]any{"name": "Pending", "vendor": false}); err != nil {
				t.Fatal(err)
			}
			value, err := handle.refresh(context.Background())
			if err != nil || handle.dirty() || value.name != "Refreshed" || value.operation != nil || handle.response().operation != nil || value.metadata.StatusCode != 200 || value.metadata.Header.Get("X-Request-ID") != "refreshed" || len(value.userMetadata) != 1 || string(value.userMetadata["new"]) != "false" || string(value.metadata.Body["vendor"]) != "false" {
				t.Fatal("Refresh failed shallow merge/reset or retained action", value, err)
			}
			if err := handle.edit(map[string]any{"name": "Newer"}); err != nil {
				t.Fatal(err)
			}
			if _, err := handle.refresh(context.Background()); !gophercloud.ResponseCodeIs(err, 403) || !handle.dirty() || handle.value().name != "Newer" || handle.response().name != "Refreshed" || gets.Load() != 2 || patches.Load() != 1 {
				t.Fatal("failed Refresh changed state", err, handle.value(), handle.response())
			}
		})
		for _, bad := range []string{`{"id":"../invalid","name":"Server"}`, `{"id":"","name":"Server"}`, `{"id":"fixed","name":false}`, `{"id":7,"name":"Server"}`} {
			t.Run(kind+"/malformed-refresh="+bad, func(t *testing.T) {
				cloud, client := clusteringAsyncTrackedCloud(t)
				var gets atomic.Int32
				body := clusteringAsyncTrackedObject(kind, bad)
				cloud.Mux.HandleFunc("GET "+clusteringAsyncTrackedPrefix+"/"+clusteringAsyncTrackedPlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					w.Header().Set("X-Request-ID", "malformed-refresh")
					testcloud.JSON(w, 200, body)
				})
				operation := &actions.Submission{ActionID: "prior", Body: json.RawMessage(`{"prior":true}`), Header: http.Header{"X-Prior": {"kept"}}, StatusCode: 202}
				handle, err := clusteringAsyncTrackedTrack(client, kind, clusteringAsyncTrackedSeed(kind), operation)
				if err != nil {
					t.Fatal(err)
				}
				if err := handle.edit(map[string]any{"name": "Pending"}); err != nil {
					t.Fatal(err)
				}
				value, err := handle.refresh(context.Background())
				var evidence *resource.ResponseError
				if value.metadata != nil || !errors.As(err, &evidence) || evidence.StatusCode != 200 || string(evidence.Body) != body || evidence.Header.Get("X-Request-ID") != "malformed-refresh" || !handle.dirty() || handle.value().name != "Pending" || handle.response().operation.ActionID != "prior" || gets.Load() != 1 {
					t.Fatal("postdecode/typed Refresh error lost full envelope or accepted pending state", value, err, evidence)
				}
			})
		}
	}
}

func TestClusteringAsyncTrackedNewEditsSurviveCommitAndRefreshInFlight(t *testing.T) {
	for _, kind := range []string{"cluster", "node"} {
		for _, refresh := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/refresh=%t", kind, refresh), func(t *testing.T) {
				cloud, client := clusteringAsyncTrackedCloud(t)
				var gets, patches atomic.Int32
				handle := clusteringAsyncTrackedMust(t, client, kind)
				path := clusteringAsyncTrackedPrefix + "/" + clusteringAsyncTrackedPlural(kind) + "/fixed"
				during := func() {
					if err := handle.edit(map[string]any{"name": "During", "newvendor": false}); err != nil {
						t.Error(err)
					}
				}
				cloud.Mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					during()
					testcloud.JSON(w, 200, clusteringAsyncTrackedObject(kind, `{"id":"response-other","name":"Server","newvendor":true}`))
				})
				cloud.Mux.HandleFunc("PATCH "+path, func(w http.ResponseWriter, r *http.Request) {
					body := clusteringAsyncTrackedWire(t, r, kind)
					call := patches.Add(1)
					if call == 1 && !refresh {
						if len(body) != 1 || string(body["name"]) != `"Submitted"` {
							t.Error(body)
						}
						during()
						clusteringAsyncTrackedAccept(w, kind, `{"id":"response-other","name":"Server","newvendor":true}`, "in-flight")
					} else {
						if len(body) != 2 || string(body["name"]) != `"During"` || string(body["newvendor"]) != "false" {
							t.Error(body)
						}
						clusteringAsyncTrackedAccept(w, kind, `{}`, "newer-accepted")
					}
				})
				if err := handle.edit(map[string]any{"name": "Submitted"}); err != nil {
					t.Fatal(err)
				}
				var value clusteringAsyncTrackedView
				var err error
				if refresh {
					value, err = handle.refresh(context.Background())
				} else {
					value, err = handle.commit(context.Background(), nil)
				}
				if err != nil || !handle.dirty() || value.name != "During" || string(value.metadata.Body["newvendor"]) != "false" || handle.response().name != "Server" || string(handle.response().metadata.Body["newvendor"]) != "true" {
					t.Fatal("accepted response erased newer revisions", value, err, handle.response())
				}
				if refresh && value.operation != nil {
					t.Fatal("Refresh retained async operation", value.operation)
				}
				if _, err := handle.commit(context.Background(), nil); err != nil || handle.dirty() || handle.value().name != "During" {
					t.Fatal(err, handle.value())
				}
				wantGets, wantPatches := int32(0), int32(2)
				if refresh {
					wantGets, wantPatches = 1, 1
				}
				if gets.Load() != wantGets || patches.Load() != wantPatches {
					t.Fatal(gets.Load(), patches.Load())
				}
			})
		}
	}
}

func TestClusteringAsyncTrackedPendingPresenceVersionGates(t *testing.T) {
	for _, kind := range []string{"cluster", "node"} {
		for _, mode := range []string{"false", "null", "remove"} {
			if kind == "cluster" && mode == "null" {
				continue
			}
			t.Run(kind+"/"+mode, func(t *testing.T) {
				cloud, client := clusteringAsyncTrackedCloud(t)
				gate, below, minimum := "tainted", "1.12", "1.13"
				if kind == "cluster" {
					gate, below, minimum = "profile_only", "1.5", "1.6"
				}
				client.Microversion = below
				var patches atomic.Int32
				cloud.Mux.HandleFunc("PATCH "+clusteringAsyncTrackedPrefix+"/"+clusteringAsyncTrackedPlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) {
					body := clusteringAsyncTrackedWire(t, r, kind)
					if patches.Add(1) == 1 {
						if len(body) != 1 || string(body["name"]) != `"NameOnly"` || r.Header.Get("OpenStack-API-Version") != "clustering "+below {
							t.Error(body, r.Header)
						}
					} else {
						want := "false"
						if mode != "false" {
							want = "null"
						}
						if len(body) != 1 || string(body[gate]) != want || r.Header.Get("OpenStack-API-Version") != "clustering "+minimum {
							t.Error(body, r.Header)
						}
					}
					clusteringAsyncTrackedAccept(w, kind, `{}`, "gated")
				})
				handle := clusteringAsyncTrackedMust(t, client, kind)
				if _, err := handle.commit(context.Background(), nil); err != nil || patches.Load() != 0 {
					t.Fatal("clean cached gated field imposed a version requirement", err, patches.Load())
				}
				if err := handle.edit(map[string]any{"name": "NameOnly"}); err != nil {
					t.Fatal(err)
				}
				if _, err := handle.commit(context.Background(), nil); err != nil {
					t.Fatal("unmodified cached gate blocked unrelated edit", err)
				}
				var err error
				switch mode {
				case "false":
					err = handle.edit(map[string]any{gate: false})
				case "null":
					err = handle.edit(map[string]any{gate: nil})
				case "remove":
					err = handle.remove(gate)
				}
				if err != nil || !handle.dirty() {
					t.Fatal("offline pending gate edit failed", err)
				}
				if _, err := handle.commit(context.Background(), nil); !errors.Is(err, resource.ErrUnsupported) || !handle.dirty() || patches.Load() != 1 {
					t.Fatal("presence gate did not precede HTTP", err, handle.dirty(), patches.Load())
				}
				client.Microversion = minimum
				if _, err := handle.commit(context.Background(), nil); err != nil || handle.dirty() || patches.Load() != 2 {
					t.Fatal(err, handle.dirty(), patches.Load())
				}
				client.Microversion = below
				if _, err := handle.commit(context.Background(), nil); err != nil || patches.Load() != 2 {
					t.Fatal("clean accepted field imposed a gate", err, patches.Load())
				}
			})
		}
	}
}

func TestClusteringAsyncTrackedOfflineEditAndCleanSourceContextValidation(t *testing.T) {
	for _, kind := range []string{"cluster", "node"} {
		t.Run(kind, func(t *testing.T) {
			cloud, client := clusteringAsyncTrackedCloud(t)
			var requests atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(500) })
			handle := clusteringAsyncTrackedMust(t, client, kind)
			clean := clusteringAsyncTrackedMust(t, client, kind)
			canceled, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := handle.commit(canceled, nil); !errors.Is(err, context.Canceled) {
				t.Fatal("clean cache bypassed context", err)
			}
			client.Type = "compute"
			if _, err := handle.commit(context.Background(), nil); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("clean cache bypassed source type", err)
			}
			if err := handle.edit(map[string]any{"name": "Offline"}); err != nil || !handle.dirty() {
				t.Fatal("Edit depended on source availability", err)
			}
			client.Type, client.Microversion = "clustering", "latest"
			for _, checked := range []*clusteringAsyncTrackedHandle{handle, clean} {
				if _, err := checked.commit(context.Background(), nil); !errors.Is(err, resource.ErrUnsupported) {
					t.Fatal("latest source bypassed clean/dirty validation", err)
				}
			}
			client.Microversion = "1.13"
			client.MoreHeaders = map[string]string{"OpenStack-API-Version": "clustering 1.12"}
			for _, checked := range []*clusteringAsyncTrackedHandle{handle, clean} {
				if _, err := checked.commit(context.Background(), nil); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal("source header conflict bypassed clean/dirty validation", err)
				}
			}
			client.MoreHeaders = nil
			client.ProviderClient = nil
			for _, checked := range []*clusteringAsyncTrackedHandle{handle, clean} {
				if _, err := checked.commit(context.Background(), nil); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal("nil provider bypassed clean/dirty validation", err)
				}
			}
			if !handle.dirty() || clean.dirty() {
				t.Fatal("invalid source changed dirty state")
			}
			if requests.Load() != 0 {
				t.Fatal("invalid source/context sent HTTP", requests.Load())
			}
		})
	}
}

func TestClusteringAsyncTrackedNameLoadAbsenceDuplicatesAndNilHandles(t *testing.T) {
	for _, kind := range []string{"cluster", "node"} {
		for _, mode := range []string{"missing", "duplicate", "case-only-name", "null-name", "case-decoy", "missing-id", "null-id", "case-only-id"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				cloud, client := clusteringAsyncTrackedCloud(t)
				var lists, gets atomic.Int32
				plural := clusteringAsyncTrackedPlural(kind)
				rows := map[string]string{"missing": `[]`, "duplicate": `[{"id":"one","name":"selected"},{"id":"two","name":"selected"}]`, "case-only-name": `[{"id":"one","Name":"selected"}]`, "null-name": `[{"id":"one","name":null,"Name":"selected"}]`, "case-decoy": `[{"id":"one","name":"other","Name":"selected"}]`, "missing-id": `[{"name":"selected"}]`, "null-id": `[{"id":null,"name":"selected"}]`, "case-only-id": `[{"ID":"shadow","name":"selected"}]`}[mode]
				body := fmt.Sprintf(`{%q:%s}`, plural, rows)
				cloud.Mux.HandleFunc("GET "+clusteringAsyncTrackedPrefix+"/"+plural, func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					w.Header().Set("X-Request-ID", "name-load")
					testcloud.JSON(w, 200, body)
				})
				cloud.Mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) { gets.Add(1); w.WriteHeader(500) })
				handle, err := clusteringAsyncTrackedLoad(context.Background(), client, kind, resource.Name("selected"))
				want := resource.ErrNotFound
				if mode == "duplicate" {
					want = resource.ErrAmbiguous
				} else if mode == "missing-id" || mode == "null-id" || mode == "case-only-id" {
					want = resource.ErrInvalidOption
					var evidence *resource.ResponseError
					if !errors.As(err, &evidence) || evidence.StatusCode != 200 || string(evidence.Body) != body || evidence.Header.Get("X-Request-ID") != "name-load" {
						t.Fatal("invalid canonical Name Load identity lost response evidence", err, evidence)
					}
				}
				if handle != nil || !errors.Is(err, want) || lists.Load() != 1 || gets.Load() != 0 {
					t.Fatal(handle, err, lists.Load(), gets.Load())
				}
			})
		}
		t.Run(kind+"/nil-and-manual", func(t *testing.T) {
			var handle *clusteringAsyncTrackedHandle
			if kind == "cluster" {
				handle = clusteringAsyncClusterHandle(nil)
				if _, err := clusters.New(nil).Track(nil); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else {
				handle = clusteringAsyncNodeHandle(nil)
				if _, err := nodes.New(nil).Track(nil); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			}
			if handle.value().metadata != nil || handle.response().metadata != nil || handle.dirty() {
				t.Fatal("nil handle fabricated state")
			}
			if err := handle.edit(nil); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			fields := []string{"name", "profile_id", "metadata", "role", "tainted"}
			if kind == "cluster" {
				fields = []string{"name", "profile_id", "metadata", "timeout", "config", "profile_only"}
			}
			for _, field := range fields {
				if err := handle.remove(field); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(field, err)
				}
			}
			if _, err := handle.commit(context.Background(), nil); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if _, err := handle.refresh(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if kind == "cluster" {
				manual, err := clusters.New(nil).Track(&clusters.Cluster{ID: "manual", Name: "Local"})
				if err != nil {
					t.Fatal(err)
				}
				if manual.Value().StatusCode != 0 || manual.Response().StatusCode != 0 || manual.Response().Operation != nil {
					t.Fatal("manual seed fabricated HTTP evidence", manual.Response())
				}
				if err := manual.Edit(clusters.UpdateOpts{}, clusters.WithUpdateName("Offline")); err != nil {
					t.Fatal(err)
				}
				if _, err := manual.Commit(context.Background()); !errors.Is(err, resource.ErrInvalidOption) || !manual.Dirty() {
					t.Fatal(err)
				}
			} else {
				manual, err := nodes.New(nil).Track(&nodes.Node{ID: "manual", Name: "Local"})
				if err != nil {
					t.Fatal(err)
				}
				if manual.Value().StatusCode != 0 || manual.Response().StatusCode != 0 || manual.Response().Operation != nil {
					t.Fatal("manual seed fabricated HTTP evidence", manual.Response())
				}
				if err := manual.Edit(nodes.UpdateOpts{}, nodes.WithUpdateName("Offline")); err != nil {
					t.Fatal(err)
				}
				if _, err := manual.Commit(context.Background()); !errors.Is(err, resource.ErrInvalidOption) || !manual.Dirty() {
					t.Fatal(err)
				}
			}
		})
	}
}

var _ io.ReadCloser = (*clusteringAsyncTrackedReadFailure)(nil)
