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

	"github.com/JSYoo5B/gophercloudsdk/clustering/v1/policies"
	"github.com/JSYoo5B/gophercloudsdk/clustering/v1/profiles"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type clusteringLifecycleView struct {
	id, name           string
	metadata           *resource.Metadata
	spec, userMetadata map[string]json.RawMessage
}

type clusteringLifecycleHandle struct {
	value, response            func() clusteringLifecycleView
	dirty                      func() bool
	editName                   func(string) error
	editEmpty                  func() error
	editField                  func(string, any) error
	editMetadata               func(any) error
	removeName, removeMetadata func() error
	commit                     func(context.Context, map[string]string) (clusteringLifecycleView, error)
	refresh                    func(context.Context) (clusteringLifecycleView, error)
	invalidEdit                func(string) error
	invalidCommit              func(context.Context, string) error
}

func lifecycleProfileView(value *profiles.Profile) clusteringLifecycleView {
	if value == nil {
		return clusteringLifecycleView{}
	}
	return clusteringLifecycleView{value.ID, value.Name, &value.Metadata, value.Spec, value.UserMetadata}
}
func lifecyclePolicyView(value *policies.Policy) clusteringLifecycleView {
	if value == nil {
		return clusteringLifecycleView{}
	}
	return clusteringLifecycleView{value.ID, value.Name, &value.Metadata, value.Spec, nil}
}

func lifecycleLoad(ctx context.Context, client *gophercloud.ServiceClient, kind string, ref resource.Ref) (*clusteringLifecycleHandle, error) {
	if kind == "profile" {
		tracked, err := profiles.New(client).Load(ctx, ref)
		if err != nil {
			return nil, err
		}
		return lifecycleProfileHandle(tracked), nil
	}
	tracked, err := policies.New(client).Load(ctx, ref)
	if err != nil {
		return nil, err
	}
	return lifecyclePolicyHandle(tracked), nil
}

func lifecycleProfileHandle(tracked *profiles.TrackedProfile) *clusteringLifecycleHandle {
	return &clusteringLifecycleHandle{
		value:     func() clusteringLifecycleView { return lifecycleProfileView(tracked.Value()) },
		response:  func() clusteringLifecycleView { return lifecycleProfileView(tracked.Response()) },
		dirty:     tracked.Dirty,
		editName:  func(name string) error { return tracked.Edit(profiles.UpdateOpts{}, profiles.WithUpdateName(name)) },
		editEmpty: func() error { return tracked.Edit(profiles.UpdateOpts{}) },
		editField: func(key string, value any) error {
			return tracked.Edit(profiles.UpdateOpts{}, profiles.WithUpdateField(key, value))
		},
		editMetadata:   func(value any) error { return tracked.Edit(profiles.UpdateOpts{}, profiles.WithUpdateMetadata(value)) },
		removeName:     tracked.RemoveName,
		removeMetadata: tracked.RemoveMetadata,
		commit: func(ctx context.Context, headers map[string]string) (clusteringLifecycleView, error) {
			opts := []profiles.UpdateOption{}
			for key, value := range headers {
				opts = append(opts, profiles.WithUpdateHeader(key, value))
			}
			value, err := tracked.Commit(ctx, opts...)
			return lifecycleProfileView(value), err
		},
		refresh: func(ctx context.Context) (clusteringLifecycleView, error) {
			value, err := tracked.Refresh(ctx)
			return lifecycleProfileView(value), err
		},
		invalidEdit: func(mode string) error {
			switch mode {
			case "header":
				return tracked.Edit(profiles.UpdateOpts{}, profiles.WithUpdateHeader("X-Vendor", "value"))
			case "query":
				return tracked.Edit(profiles.UpdateOpts{}, request.WithQuery[profiles.UpdateOpts]("vendor", "value"))
			case "nil":
				return tracked.Edit(profiles.UpdateOpts{}, nil)
			default:
				return tracked.Edit(profiles.UpdateOpts{}, profiles.WithUpdateName("valid"), profiles.WithUpdateField(mode, false))
			}
		},
		invalidCommit: func(ctx context.Context, mode string) error {
			var option profiles.UpdateOption
			switch mode {
			case "name":
				option = profiles.WithUpdateName("valid")
			case "field":
				option = profiles.WithUpdateField("vendor", true)
			case "query":
				option = request.WithQuery[profiles.UpdateOpts]("vendor", "value")
			case "auth":
				option = profiles.WithUpdateHeader("X-Auth-Token", "override")
			}
			_, err := tracked.Commit(ctx, option)
			return err
		},
	}
}

func lifecyclePolicyHandle(tracked *policies.TrackedPolicy) *clusteringLifecycleHandle {
	return &clusteringLifecycleHandle{
		value:     func() clusteringLifecycleView { return lifecyclePolicyView(tracked.Value()) },
		response:  func() clusteringLifecycleView { return lifecyclePolicyView(tracked.Response()) },
		dirty:     tracked.Dirty,
		editName:  func(name string) error { return tracked.Edit(policies.UpdateOpts{}, policies.WithUpdateName(name)) },
		editEmpty: func() error { return tracked.Edit(policies.UpdateOpts{}) },
		editField: func(key string, value any) error {
			return tracked.Edit(policies.UpdateOpts{}, policies.WithUpdateField(key, value))
		},
		removeName: tracked.RemoveName,
		commit: func(ctx context.Context, headers map[string]string) (clusteringLifecycleView, error) {
			opts := []policies.UpdateOption{}
			for key, value := range headers {
				opts = append(opts, policies.WithUpdateHeader(key, value))
			}
			value, err := tracked.Commit(ctx, opts...)
			return lifecyclePolicyView(value), err
		},
		refresh: func(ctx context.Context) (clusteringLifecycleView, error) {
			value, err := tracked.Refresh(ctx)
			return lifecyclePolicyView(value), err
		},
		invalidEdit: func(mode string) error {
			switch mode {
			case "header":
				return tracked.Edit(policies.UpdateOpts{}, policies.WithUpdateHeader("X-Vendor", "value"))
			case "query":
				return tracked.Edit(policies.UpdateOpts{}, request.WithQuery[policies.UpdateOpts]("vendor", "value"))
			case "nil":
				return tracked.Edit(policies.UpdateOpts{}, nil)
			default:
				return tracked.Edit(policies.UpdateOpts{}, policies.WithUpdateName("valid"), policies.WithUpdateField(mode, false))
			}
		},
		invalidCommit: func(ctx context.Context, mode string) error {
			var option policies.UpdateOption
			switch mode {
			case "name":
				option = policies.WithUpdateName("valid")
			case "field":
				option = policies.WithUpdateField("vendor", true)
			case "query":
				option = request.WithQuery[policies.UpdateOpts]("vendor", "value")
			case "auth":
				option = policies.WithUpdateHeader("X-Auth-Token", "override")
			}
			_, err := tracked.Commit(ctx, option)
			return err
		},
	}
}

func lifecyclePlural(kind string) string {
	if kind == "policy" {
		return "policies"
	}
	return "profiles"
}

func lifecycleBody(kind string) string {
	return fmt.Sprintf(`{%q:{"id":"fixed","name":"A","type":"immutable","project":"project","spec":{"big":9007199254740993},"metadata":{"a":1,"b":2},"future":{"off":false}}}`, kind)
}

func TestClusteringTrackedNoOpStickyDirtyMergeAndSeparateResponse(t *testing.T) {
	for _, kind := range []string{"profile", "policy"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, patches atomic.Int32
			cloud.Mux.HandleFunc("GET /reverse/senlin/v1/"+lifecyclePlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				w.Header().Set("X-Request-ID", "initial")
				testcloud.JSON(w, 200, lifecycleBody(kind))
			})
			cloud.Mux.HandleFunc("PATCH /reverse/senlin/v1/"+lifecyclePlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				call := patches.Add(1)
				want := fmt.Sprintf(`{%q:{"name":"A"}}`, kind)
				if call == 2 {
					want = fmt.Sprintf(`{%q:{"name":"B","vendor":{"big":9007199254740995,"off":false}}}`, kind)
				}
				if string(body) != want || r.Header.Get("X-Vendor") != "sent" || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Error(string(body), want, r.URL, r.Header)
				}
				w.Header().Set("X-Request-ID", "partial-response")
				testcloud.JSON(w, 200, fmt.Sprintf(`{%q:{"name":"normalized","metadata":{"a":3}}}`, kind))
			})
			handle, err := lifecycleLoad(context.Background(), cloud.Client("clustering", "/reverse/senlin/v1"), kind, resource.ID("fixed"))
			if err != nil {
				t.Fatal(err)
			}
			if err := handle.editEmpty(); err != nil {
				t.Fatal(err)
			}
			if err := handle.editName("A"); err != nil {
				t.Fatal(err)
			}
			if _, err := handle.commit(context.Background(), map[string]string{"X-Vendor": "sent"}); err != nil || handle.dirty() || patches.Load() != 0 {
				t.Fatal(err, handle.dirty(), patches.Load())
			}
			for _, name := range []string{"B", "A", "A"} {
				if err := handle.editName(name); err != nil {
					t.Fatal(err)
				}
			}
			if !handle.dirty() {
				t.Fatal("A→B→A lost sticky dirty state")
			}
			value, err := handle.commit(context.Background(), map[string]string{"X-Vendor": "sent"})
			if err != nil || handle.dirty() || value.name != "normalized" || string(value.spec["big"]) != "9007199254740993" || value.id != "fixed" || string(value.metadata.Body["project"]) != `"project"` || string(value.metadata.Body["future"]) != `{"off":false}` {
				t.Fatal(value, err)
			}
			if kind == "profile" && (len(value.userMetadata) != 1 || string(value.userMetadata["a"]) != "3") {
				t.Fatal("nested object recursively merged", value)
			}
			actual := handle.response()
			if len(actual.metadata.Body) != 2 || actual.id != "" || len(actual.spec) != 0 || actual.metadata.Header.Get("X-Request-ID") != "partial-response" {
				t.Fatal("cache presented as raw response", actual)
			}
			vendor := map[string]any{"big": json.Number("9007199254740995"), "off": false}
			if err := handle.editField("vendor", vendor); err != nil {
				t.Fatal(err)
			}
			vendor["off"] = true
			if err := handle.editName("B"); err != nil {
				t.Fatal(err)
			}
			if _, err := handle.commit(context.Background(), map[string]string{"X-Vendor": "sent"}); err != nil || handle.dirty() || string(handle.value().metadata.Body["vendor"]) != `{"big":9007199254740995,"off":false}` {
				t.Fatal("omitted edited field was lost/not clean", err, handle.value())
			}
			if _, err := handle.commit(context.Background(), nil); err != nil || patches.Load() != 2 || gets.Load() != 1 {
				t.Fatal("clean Commit sent HTTP", err, gets.Load(), patches.Load())
			}
		})
	}
}

func TestClusteringTrackedSnapshotsAndAuthoritativeTrackIdentity(t *testing.T) {
	for _, kind := range []string{"profile", "policy"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			var patches atomic.Int32
			cloud.Mux.HandleFunc("GET /senlin/v1/"+lifecyclePlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Request-ID", "original")
				testcloud.JSON(w, 200, lifecycleBody(kind))
			})
			cloud.Mux.HandleFunc("PATCH /senlin/v1/"+lifecyclePlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) {
				patches.Add(1)
				testcloud.JSON(w, 200, fmt.Sprintf(`{%q:{"name":"B"}}`, kind))
			})
			client := cloud.Client("clustering", "/senlin/v1")
			var handle *clusteringLifecycleHandle
			if kind == "profile" {
				api := profiles.New(client)
				input, err := api.Get(context.Background(), "fixed")
				if err != nil {
					t.Fatal(err)
				}
				input.ID, input.Name = "typed-id-mutation", "typed-name-mutation"
				tracked, err := api.Track(input)
				if err != nil {
					t.Fatal(err)
				}
				handle = lifecycleProfileHandle(tracked)
				input.Spec["big"][0] = '0'
				input.Body["id"][1] = 'X'
				input.Header.Set("X-Request-ID", "input-change")
			} else {
				api := policies.New(client)
				input, err := api.Get(context.Background(), "fixed")
				if err != nil {
					t.Fatal(err)
				}
				input.ID, input.Name = "typed-id-mutation", "typed-name-mutation"
				tracked, err := api.Track(input)
				if err != nil {
					t.Fatal(err)
				}
				handle = lifecyclePolicyHandle(tracked)
				input.Spec["big"][0] = '0'
				input.Body["id"][1] = 'X'
				input.Header.Set("X-Request-ID", "input-change")
			}
			for _, getter := range []func() clusteringLifecycleView{handle.value, handle.response} {
				view := getter()
				if view.id != "fixed" || view.name != "A" || string(view.spec["big"]) != "9007199254740993" || view.metadata.Header.Get("X-Request-ID") != "original" {
					t.Fatal("Track ignored authoritative Body or shared input", view)
				}
				view.metadata.Body["id"][1] = 'Y'
				view.spec["big"][0] = '1'
				view.metadata.Header.Set("X-Request-ID", "getter-change")
			}
			if handle.value().id != "fixed" || handle.response().metadata.Header.Get("X-Request-ID") != "original" || handle.dirty() {
				t.Fatal("getters changed handle", handle.value())
			}
			if err := handle.editName("B"); err != nil {
				t.Fatal(err)
			}
			if _, err := handle.commit(context.Background(), nil); err != nil || patches.Load() != 1 {
				t.Fatal("typed ID mutation retargeted request", err, patches.Load())
			}
		})
	}
}

func TestClusteringTrackedManualSeedAndInvalidCachedModels(t *testing.T) {
	for _, kind := range []string{"profile", "policy"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				w.WriteHeader(http.StatusInternalServerError)
			})
			client := cloud.Client("clustering", "/senlin/v1")
			track := func(body map[string]json.RawMessage, id string) (*clusteringLifecycleHandle, error) {
				if kind == "profile" {
					value := &profiles.Profile{ID: id, Name: "local", Spec: map[string]json.RawMessage{"big": json.RawMessage("9007199254740993")}}
					value.Body = body
					tracked, err := profiles.New(client).Track(value)
					if err != nil {
						return nil, err
					}
					return lifecycleProfileHandle(tracked), nil
				}
				value := &policies.Policy{ID: id, Name: "local", Spec: map[string]json.RawMessage{"big": json.RawMessage("9007199254740993")}}
				value.Body = body
				tracked, err := policies.New(client).Track(value)
				if err != nil {
					return nil, err
				}
				return lifecyclePolicyHandle(tracked), nil
			}
			handle, err := track(nil, "fixed")
			if err != nil {
				t.Fatal(err)
			}
			for _, getter := range []func() clusteringLifecycleView{handle.value, handle.response} {
				value := getter()
				if value.id != "fixed" || value.name != "local" || value.metadata.StatusCode != 0 || len(value.metadata.Header) != 0 || string(value.spec["big"]) != "9007199254740993" {
					t.Fatal("manual seed invented HTTP evidence or lost values", value)
				}
			}
			if _, err := handle.commit(context.Background(), nil); err != nil || handle.dirty() || requests.Load() != 0 {
				t.Fatal("clean local seed made a request", err, requests.Load())
			}
			for _, body := range []map[string]json.RawMessage{
				{},
				{"id": json.RawMessage("null")},
				{"id": json.RawMessage(`""`)},
				{"id": json.RawMessage(`"bad/id"`)},
				{"id": json.RawMessage(`"fixed"`), "name": json.RawMessage("false")},
				{"id": json.RawMessage(`"fixed"`), "spec": json.RawMessage("[1]")},
				{"id": json.RawMessage(`"fixed"`), "vendor": json.RawMessage("{")},
			} {
				if _, err := track(body, "typed-fallback-must-not-be-used"); err == nil {
					t.Fatal("invalid authoritative cached body accepted", body)
				}
			}
			if kind == "profile" {
				if _, err := profiles.New(client).Track(nil); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if _, err := policies.New(client).Track(nil); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if requests.Load() != 0 {
				t.Fatal("Track performed HTTP", requests.Load())
			}
		})
	}
}

func TestClusteringTrackedExplicitLoadKeepsRequestIDAndNameResolvesOnce(t *testing.T) {
	for _, kind := range []string{"profile", "policy"} {
		for _, identity := range []string{"changed", "null", "omitted", "name"} {
			t.Run(kind+"/"+identity, func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets, lists, patches atomic.Int32
				idField := `"id":"changed-response-id",`
				if identity == "null" {
					idField = `"id":null,`
				} else if identity == "omitted" {
					idField = ""
				}
				cloud.Mux.HandleFunc("GET /senlin/v1/"+lifecyclePlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) {
					gets.Add(1)
					testcloud.JSON(w, 200, `{`+fmt.Sprintf(`%q`, kind)+`:{`+idField+`"name":"A"}}`)
				})
				cloud.Mux.HandleFunc("GET /senlin/v1/"+lifecyclePlural(kind), func(w http.ResponseWriter, r *http.Request) {
					lists.Add(1)
					if r.URL.Query().Get("name") != "selected" {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, `{`+fmt.Sprintf(`%q`, lifecyclePlural(kind))+`:[{"id":"fixed","name":"selected"}]}`)
				})
				cloud.Mux.HandleFunc("PATCH /senlin/v1/"+lifecyclePlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) {
					patches.Add(1)
					testcloud.JSON(w, 200, `{`+fmt.Sprintf(`%q`, kind)+`:{`+idField+`"name":"server-name"}}`)
				})
				ref := resource.ID("fixed")
				if identity == "name" {
					ref = resource.Name("selected")
				}
				handle, err := lifecycleLoad(context.Background(), cloud.Client("clustering", "/senlin/v1"), kind, ref)
				if err != nil {
					t.Fatal(err)
				}
				for _, name := range []string{"B", "C"} {
					if err := handle.editName(name); err != nil {
						t.Fatal(err)
					}
					if _, err := handle.commit(context.Background(), nil); err != nil {
						t.Fatal(err)
					}
				}
				wantGets, wantLists := int32(1), int32(0)
				if identity == "name" {
					wantGets, wantLists = 0, 1
				}
				if gets.Load() != wantGets || lists.Load() != wantLists || patches.Load() != 2 {
					t.Fatal("response changed fixed ID/name resolved again", gets.Load(), lists.Load(), patches.Load())
				}
			})
		}
	}
}

func TestClusteringTrackedDeletionNullAndMetadataPresence(t *testing.T) {
	for _, kind := range []string{"profile", "policy"} {
		t.Run(kind+"/name", func(t *testing.T) {
			cloud := testcloud.New(t)
			var patches atomic.Int32
			cloud.Mux.HandleFunc("GET /senlin/v1/"+lifecyclePlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, lifecycleBody(kind)) })
			cloud.Mux.HandleFunc("PATCH /senlin/v1/"+lifecyclePlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) {
				patches.Add(1)
				body, _ := io.ReadAll(r.Body)
				if string(body) != fmt.Sprintf(`{%q:{"name":null}}`, kind) {
					t.Error(string(body))
				}
				testcloud.JSON(w, 200, fmt.Sprintf(`{%q:{}}`, kind))
			})
			handle, err := lifecycleLoad(context.Background(), cloud.Client("clustering", "/senlin/v1"), kind, resource.ID("fixed"))
			if err != nil {
				t.Fatal(err)
			}
			if err := handle.removeName(); err != nil {
				t.Fatal(err)
			}
			if _, exists := handle.value().metadata.Body["name"]; exists || !handle.dirty() {
				t.Fatal("removed name stays present", handle.value())
			}
			if _, err := handle.commit(context.Background(), nil); err != nil || handle.dirty() {
				t.Fatal(err)
			}
			if err := handle.removeName(); err != nil {
				t.Fatal(err)
			}
			if _, err := handle.commit(context.Background(), nil); err != nil || patches.Load() != 1 {
				t.Fatal("removing absent field created dirty request", err, patches.Load())
			}
		})
	}
	cloud := testcloud.New(t)
	var patches atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/profiles/fixed", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, lifecycleBody("profile")) })
	cloud.Mux.HandleFunc("PATCH /senlin/v1/profiles/fixed", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		want := `{"profile":{"metadata":null}}`
		if patches.Add(1) == 3 {
			want = `{"profile":{"metadata":{}}}`
		}
		if string(body) != want {
			t.Error(string(body), want)
		}
		testcloud.JSON(w, 200, `{"profile":{}}`)
	})
	handle, err := lifecycleLoad(context.Background(), cloud.Client("clustering", "/senlin/v1"), "profile", resource.ID("fixed"))
	if err != nil {
		t.Fatal(err)
	}
	if err := handle.editMetadata(nil); err != nil {
		t.Fatal(err)
	}
	if string(handle.value().metadata.Body["metadata"]) != "null" {
		t.Fatal("null metadata was omitted", handle.value())
	}
	if _, err := handle.commit(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if err := handle.removeMetadata(); err != nil {
		t.Fatal(err)
	}
	if _, exists := handle.value().metadata.Body["metadata"]; exists {
		t.Fatal("removal kept a null field", handle.value())
	}
	if _, err := handle.commit(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if err := handle.editMetadata(map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if string(handle.value().metadata.Body["metadata"]) != "{}" {
		t.Fatal(handle.value())
	}
	if _, err := handle.commit(context.Background(), nil); err != nil || patches.Load() != 3 {
		t.Fatal(err, patches.Load())
	}
}

func TestClusteringTrackedInvalidEditsAndCommitOptionsStayAtomic(t *testing.T) {
	for _, kind := range []string{"profile", "policy"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			var patches atomic.Int32
			cloud.Mux.HandleFunc("GET /senlin/v1/"+lifecyclePlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, lifecycleBody(kind)) })
			cloud.Mux.HandleFunc("PATCH /senlin/v1/"+lifecyclePlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) { patches.Add(1); w.WriteHeader(500) })
			client := cloud.Client("clustering", "/senlin/v1")
			handle, err := lifecycleLoad(context.Background(), client, kind, resource.ID("fixed"))
			if err != nil {
				t.Fatal(err)
			}
			for _, mode := range []string{"id", "ID", "name", "NAME", "spec", "Spec", "project", "created_at", "links", "header", "query", "nil"} {
				if err := handle.invalidEdit(mode); !errors.Is(err, resource.ErrInvalidOption) || handle.dirty() || handle.value().name != "A" {
					t.Fatal("invalid edit partly applied or broke typed cache", mode, err, handle.value())
				}
			}
			if kind == "profile" {
				if err := handle.editMetadata([]any{}); !errors.Is(err, resource.ErrInvalidOption) || handle.dirty() {
					t.Fatal(err)
				}
			} else if err := handle.invalidEdit("data"); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			for _, mode := range []string{"name", "field", "query", "auth", "nil"} {
				if err := handle.invalidCommit(context.Background(), mode); !errors.Is(err, resource.ErrInvalidOption) || handle.dirty() {
					t.Fatal("invalid Commit bypassed clean validation", mode, err)
				}
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := handle.commit(ctx, nil); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			client.Type = "compute"
			if _, err := handle.commit(context.Background(), nil); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			client.Type, client.Microversion = "clustering", "latest"
			if _, err := handle.commit(context.Background(), nil); !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal(err)
			}
			if patches.Load() != 0 {
				t.Fatal("invalid clean operation reached HTTP", patches.Load())
			}
		})
	}
}

func TestClusteringTrackedCommitFailureKeepsDirtyAndResponseEvidence(t *testing.T) {
	fixtures := []struct {
		name string
		code int
		body string
	}{
		{"forbidden", 403, `{"error":"denied"}`},
		{"conflict", 409, `{"error":"conflict"}`},
		{"wrong-success-code", 201, `{}`},
		{"accepted-wrong-envelope", 200, `{"wrong":{}}`},
		{"accepted-null", 200, `null`},
		{"accepted-invalid-json", 200, `{`},
	}
	for _, kind := range []string{"profile", "policy"} {
		for _, fixture := range fixtures {
			t.Run(kind+"/"+fixture.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				var patches atomic.Int32
				cloud.Mux.HandleFunc("GET /senlin/v1/"+lifecyclePlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("X-Request-ID", "last-good")
					testcloud.JSON(w, 200, lifecycleBody(kind))
				})
				cloud.Mux.HandleFunc("PATCH /senlin/v1/"+lifecyclePlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) {
					patches.Add(1)
					w.Header().Set("X-Request-ID", "failure-evidence")
					testcloud.JSON(w, fixture.code, fixture.body)
				})
				handle, err := lifecycleLoad(context.Background(), cloud.Client("clustering", "/senlin/v1"), kind, resource.ID("fixed"))
				if err != nil {
					t.Fatal(err)
				}
				if err := handle.editName("B"); err != nil {
					t.Fatal(err)
				}
				_, err = handle.commit(context.Background(), nil)
				if fixture.code == 200 {
					var evidence *resource.ResponseError
					if !errors.As(err, &evidence) || evidence.StatusCode != 200 || string(evidence.Body) != fixture.body || evidence.Header.Get("X-Request-ID") != "failure-evidence" {
						t.Fatal(err, evidence)
					}
				} else if !gophercloud.ResponseCodeIs(err, fixture.code) {
					t.Fatal("native error lost", err)
				}
				if !handle.dirty() || handle.value().name != "B" || handle.response().name != "A" || handle.response().metadata.Header.Get("X-Request-ID") != "last-good" || patches.Load() != 1 {
					t.Fatal("failed response changed cache/dirty or replayed", handle.value(), handle.response(), patches.Load())
				}
			})
		}
		t.Run(kind+"/transport", func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("GET /senlin/v1/"+lifecyclePlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, lifecycleBody(kind)) })
			handle, err := lifecycleLoad(context.Background(), cloud.Client("clustering", "/senlin/v1"), kind, resource.ID("fixed"))
			if err != nil {
				t.Fatal(err)
			}
			if err := handle.editName("B"); err != nil {
				t.Fatal(err)
			}
			original := errors.New("original tracked transport error")
			var requests atomic.Int32
			cloud.Provider.HTTPClient.Transport = clusteringWaitTransport(func(r *http.Request) (*http.Response, error) { requests.Add(1); return nil, original })
			if _, err := handle.commit(context.Background(), nil); !errors.Is(err, original) || !handle.dirty() || requests.Load() != 1 {
				t.Fatal(err, handle.dirty(), requests.Load())
			}
		})
	}
}

func TestClusteringTrackedRefreshMergeResetAndFailurePreservesChanges(t *testing.T) {
	for _, kind := range []string{"profile", "policy"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets atomic.Int32
			cloud.Mux.HandleFunc("GET /senlin/v1/"+lifecyclePlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) {
				switch gets.Add(1) {
				case 1:
					testcloud.JSON(w, 200, lifecycleBody(kind))
				case 2:
					testcloud.JSON(w, 200, fmt.Sprintf(`{%q:{"name":"server","spec":{"new":false}}}`, kind))
				default:
					testcloud.JSON(w, 403, `{"error":"refresh-failed"}`)
				}
			})
			handle, err := lifecycleLoad(context.Background(), cloud.Client("clustering", "/senlin/v1"), kind, resource.ID("fixed"))
			if err != nil {
				t.Fatal(err)
			}
			if err := handle.editName("B"); err != nil {
				t.Fatal(err)
			}
			if err := handle.editField("vendor", false); err != nil {
				t.Fatal(err)
			}
			value, err := handle.refresh(context.Background())
			if err != nil || handle.dirty() || value.name != "server" || len(value.spec) != 1 || string(value.spec["new"]) != "false" || string(value.metadata.Body["vendor"]) != "false" {
				t.Fatal("Refresh failed field merge or omitted-field clean", value, err)
			}
			if err := handle.editName("C"); err != nil {
				t.Fatal(err)
			}
			if _, err := handle.refresh(context.Background()); !gophercloud.ResponseCodeIs(err, 403) || !handle.dirty() || handle.value().name != "C" || handle.response().name != "server" {
				t.Fatal("failed Refresh reset cache", err, handle.value())
			}
		})
	}
}

func TestClusteringTrackedNewEditsSurviveCommitAndRefreshInFlight(t *testing.T) {
	for _, kind := range []string{"profile", "policy"} {
		for _, refresh := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/refresh=%t", kind, refresh), func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets, patches atomic.Int32
				var handle *clusteringLifecycleHandle
				cloud.Mux.HandleFunc("GET /senlin/v1/"+lifecyclePlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) {
					if gets.Add(1) == 1 {
						testcloud.JSON(w, 200, lifecycleBody(kind))
						return
					}
					if err := handle.editName("during"); err != nil {
						t.Error(err)
					}
					if err := handle.editField("newvendor", false); err != nil {
						t.Error(err)
					}
					testcloud.JSON(w, 200, fmt.Sprintf(`{%q:{"id":"response-other","name":"server","newvendor":true}}`, kind))
				})
				cloud.Mux.HandleFunc("PATCH /senlin/v1/"+lifecyclePlural(kind)+"/fixed", func(w http.ResponseWriter, r *http.Request) {
					body, _ := io.ReadAll(r.Body)
					call := patches.Add(1)
					if call == 1 && !refresh {
						if string(body) != fmt.Sprintf(`{%q:{"name":"submitted"}}`, kind) {
							t.Error(string(body))
						}
						if err := handle.editName("during"); err != nil {
							t.Error(err)
						}
						if err := handle.editField("newvendor", false); err != nil {
							t.Error(err)
						}
						testcloud.JSON(w, 200, fmt.Sprintf(`{%q:{"id":"response-other","name":"server","newvendor":true}}`, kind))
					} else {
						if string(body) != fmt.Sprintf(`{%q:{"name":"during","newvendor":false}}`, kind) {
							t.Error(string(body))
						}
						testcloud.JSON(w, 200, fmt.Sprintf(`{%q:{}}`, kind))
					}
				})
				var err error
				handle, err = lifecycleLoad(context.Background(), cloud.Client("clustering", "/senlin/v1"), kind, resource.ID("fixed"))
				if err != nil {
					t.Fatal(err)
				}
				if err := handle.editName("submitted"); err != nil {
					t.Fatal(err)
				}
				if refresh {
					_, err = handle.refresh(context.Background())
				} else {
					_, err = handle.commit(context.Background(), nil)
				}
				if err != nil || !handle.dirty() || handle.value().name != "during" || string(handle.value().metadata.Body["newvendor"]) != "false" || handle.response().name != "server" {
					t.Fatal("response erased newer changes", err, handle.value(), handle.response())
				}
				if _, err := handle.commit(context.Background(), nil); err != nil || handle.dirty() || handle.value().name != "during" {
					t.Fatal(err, handle.value())
				}
			})
		}
	}
}
