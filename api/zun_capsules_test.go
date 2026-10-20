package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/container/v1/capsules"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const zunCapsuleTemplate = `{"capsuleVersion":"beta","kind":"capsule","spec":{"containers":[{"image":"cirros"}]}}`

// zunCapsuleLegacyBody is the pre-1.32 shape, zunCapsuleV132Body the 1.32 shape.
const zunCapsuleLegacyBody = `{"uuid":"cap-1","meta_name":"web","status":"Running","restart_policy":"always","meta_labels":{"a":"b"},"cpu":0,"created_at":"2026-01-02 03:04:05","containers":[{"uuid":"c-1","name":"app","auto_remove":false,"disk":0}]}`
const zunCapsuleV132Body = `{"uuid":"cap-1","name":"web","status":"Running","restart_policy":{"Name":"always"},"labels":{"a":"b"},"cpu":0,"created_at":"2026-01-02 03:04:05","containers":[{"uuid":"c-1","name":"app","auto_remove":false,"disk":0}]}`

func newZunCapsules(t *testing.T, cloud *testcloud.Cloud, microversion string) *capsules.API {
	t.Helper()
	client := cloud.Client("container", "/v1")
	client.Microversion = microversion
	return capsules.New(client)
}

func zunUnexpected(err error) int {
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if errors.As(err, &unexpected) {
		return unexpected.Actual
	}
	return 0
}

func zunTemplate() *capsules.Template { return &capsules.Template{Bin: []byte(zunCapsuleTemplate)} }

func TestZunCapsuleCreateSendsTemplateStringAndRequires202(t *testing.T) {
	for _, status := range []int{202, 200, 201, 204} {
		cloud := testcloud.New(t)
		var bodies []map[string]json.RawMessage
		var versions []string
		cloud.Mux.HandleFunc("POST /v1/capsules", func(w http.ResponseWriter, r *http.Request) {
			var body map[string]json.RawMessage
			data, _ := io.ReadAll(r.Body)
			if err := json.Unmarshal(data, &body); err != nil {
				t.Errorf("request body %q: %v", data, err)
			}
			bodies = append(bodies, body)
			versions = append(versions, r.Header.Get("OpenStack-API-Version"))
			if status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, status, zunCapsuleV132Body)
		})
		api := newZunCapsules(t, cloud, "1.32")
		value, err := api.Create(context.Background(), capsules.CreateOpts{TemplateOpts: zunTemplate()}, capsules.WithCreateField("vendor", false))
		if (status == 202) != (err == nil) {
			t.Fatalf("status %d value=%v error=%v", status, value, err)
		}
		if status != 202 {
			if zunUnexpected(err) != status {
				t.Fatalf("status %d error=%v", status, err)
			}
			continue
		}
		if len(bodies[0]) != 2 || string(bodies[0]["vendor"]) != `false` {
			t.Fatalf("body=%v", bodies[0])
		}
		var template string
		if err := json.Unmarshal(bodies[0]["template"], &template); err != nil || template != zunCapsuleTemplate {
			t.Fatalf("template=%q error=%v", template, err)
		}
		if versions[0] != "container 1.32" {
			t.Fatalf("versions=%v", versions)
		}
		if got, ok := value.(*capsules.CapsuleV132); !ok || got.UUID != "cap-1" || got.MetaName != "web" || got.RestartPolicy["Name"] != "always" {
			t.Fatalf("value=%#v", value)
		}
	}
}

func TestZunCapsuleResponsesDecodeAsLegacyOrV132(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v1/capsules/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("id") {
		case "legacy":
			testcloud.JSON(w, 200, zunCapsuleLegacyBody)
		case "v132":
			testcloud.JSON(w, 200, zunCapsuleV132Body)
		case "cached":
			testcloud.JSON(w, 203, zunCapsuleV132Body)
		case "created":
			testcloud.JSON(w, 201, zunCapsuleV132Body)
		default:
			testcloud.JSON(w, 404, `{"errors":[{"detail":"gone"}]}`)
		}
	})
	api := newZunCapsules(t, cloud, "")
	ctx := context.Background()
	legacy, err := api.Get(ctx, "legacy")
	base, ok := legacy.(*capsules.Capsule)
	if err != nil || !ok || base.MetaName != "web" || base.RestartPolicy != "always" || base.MetaLabels["a"] != "b" || base.CreatedAt.IsZero() || len(base.Containers) != 1 || base.Containers[0].AutoRemove {
		t.Fatalf("legacy=%#v error=%v", legacy, err)
	}
	for _, id := range []string{"v132", "cached"} {
		value, err := api.Get(ctx, id)
		got, ok := value.(*capsules.CapsuleV132)
		if err != nil || !ok || got.MetaName != "web" || got.RestartPolicy["Name"] != "always" || got.MetaLabels["a"] != "b" || got.CreatedAt.IsZero() {
			t.Fatalf("%s value=%#v error=%v", id, value, err)
		}
	}
	if _, err := api.Get(ctx, "created"); zunUnexpected(err) != 201 {
		t.Fatalf("Get accepts only 200 and 203: %v", err)
	}
	if _, err := api.Get(ctx, "missing"); zunUnexpected(err) != 404 || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
}

func TestZunCapsuleRejectsInvalidInputsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := newZunCapsules(t, cloud, "")
	ctx := context.Background()
	if _, err := api.Create(ctx, capsules.CreateOpts{}); err == nil {
		t.Fatal("missing template must fail locally")
	}
	var invalid capsules.ErrInvalidDataFormat
	if _, err := api.Create(ctx, capsules.CreateOpts{TemplateOpts: &capsules.Template{Bin: []byte("{not: [valid")}}); !errors.As(err, &invalid) {
		t.Fatalf("invalid template must fail locally: %v", err)
	}
	if _, err := api.Create(ctx, capsules.CreateOpts{TemplateOpts: zunTemplate()}, capsules.WithCreateField("template", "shadow")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("template collision: %v", err)
	}
	if _, err := api.Create(ctx, capsules.CreateOpts{TemplateOpts: zunTemplate()}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil create option: %v", err)
	}
	for _, err := range api.List(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil list option: %v", err)
		}
	}
}

func TestZunCapsuleTemplateYAMLIsSentVerbatim(t *testing.T) {
	cloud := testcloud.New(t)
	const yaml = "capsuleVersion: beta\nkind: capsule\n"
	var template string
	cloud.Mux.HandleFunc("POST /v1/capsules", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Template string `json:"template"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		template = body.Template
		testcloud.JSON(w, 202, zunCapsuleV132Body)
	})
	if _, err := newZunCapsules(t, cloud, "").Create(context.Background(), capsules.CreateOpts{TemplateOpts: &capsules.Template{Bin: []byte(yaml)}}); err != nil {
		t.Fatal(err)
	}
	if template != yaml {
		t.Fatalf("template=%q", template)
	}
}

func TestZunCapsuleListQueriesPagingAndV132Rows(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	var version string
	cloud.Mux.HandleFunc("GET /v1/capsules", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		version = r.Header.Get("OpenStack-API-Version")
		if r.URL.Query().Get("marker") == "cap-1" {
			testcloud.JSON(w, 200, `{"capsules":[{"uuid":"cap-2","name":"two","restart_policy":{},"labels":{}}]}`)
			return
		}
		next := cloud.Server.URL + "/v1/capsules?marker=cap-1&limit=1"
		testcloud.JSON(w, 200, `{"capsules":[`+zunCapsuleV132Body+`],"next":"`+next+`"}`)
	})
	api := newZunCapsules(t, cloud, "1.32")
	var ids []string
	for value, err := range api.List(context.Background(),
		capsules.WithListOptions(capsules.ListOpts{Limit: 1, SortKey: "name", SortDir: "desc", AllProjects: true}),
		capsules.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.UUID+":"+value.MetaName)
	}
	if !reflect.DeepEqual(ids, []string{"cap-1:web", "cap-2:two"}) || len(queries) != 2 {
		t.Fatalf("ids=%v queries=%v", ids, queries)
	}
	first := queries[0]
	if first.Get("limit") != "1" || first.Get("sort_key") != "name" || first.Get("sort_dir") != "desc" || first.Get("all_projects") != "true" || first.Get("vendor") != "x" {
		t.Fatalf("first query=%v", first)
	}
	if queries[1].Get("marker") != "cap-1" || version != "container 1.32" {
		t.Fatalf("second query=%v version=%q", queries[1], version)
	}
	// AllProjects is a plain bool: false is omitted instead of sent as false.
	queries = nil
	for _, err := range api.List(context.Background(), capsules.WithListOptions(capsules.ListOpts{AllProjects: false})) {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if len(queries) == 0 || queries[0].Has("all_projects") {
		t.Fatalf("explicit false query=%v", queries)
	}
}

func TestZunCapsuleListRejectsLegacyRowShape(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v1/capsules", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"capsules":[`+zunCapsuleLegacyBody+`]}`)
	})
	var rows int
	var failure error
	for value, err := range newZunCapsules(t, cloud, "").List(context.Background()) {
		if err != nil {
			failure = err
			break
		}
		if value != nil {
			rows++
		}
	}
	if failure == nil || rows != 0 {
		t.Fatalf("legacy rows=%d error=%v", rows, failure)
	}
}

func TestZunCapsuleDeleteStatusesAndNativeMissing(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("DELETE /v1/capsules/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("id") {
		case "missing":
			testcloud.JSON(w, 404, `{}`)
		case "ok200":
			testcloud.JSON(w, 200, `{}`)
		case "ok202":
			w.WriteHeader(202)
		default:
			w.WriteHeader(204)
		}
	})
	api := newZunCapsules(t, cloud, "")
	ctx := context.Background()
	for _, id := range []string{"cap-1", "ok202"} {
		if err := api.Delete(ctx, id); err != nil {
			t.Fatalf("delete %s: %v", id, err)
		}
	}
	if err := api.Delete(ctx, "ok200"); zunUnexpected(err) != 200 {
		t.Fatalf("200 is not an accepted delete code: %v", err)
	}
	if err := api.Delete(ctx, "missing"); zunUnexpected(err) != 404 || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native delete keeps the 404 response error: %v", err)
	}
}

func TestZunCapsuleOmitsVersionHeaderWithoutClientMicroversion(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v1/capsules/cap-1", func(w http.ResponseWriter, r *http.Request) {
		if _, present := r.Header["Openstack-Api-Version"]; present {
			t.Errorf("unexpected version header %q", r.Header.Get("OpenStack-API-Version"))
		}
		testcloud.JSON(w, 200, zunCapsuleLegacyBody)
	})
	if _, err := newZunCapsules(t, cloud, "").Get(context.Background(), "cap-1"); err != nil {
		t.Fatal(err)
	}
}

func TestZunCapsuleCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	api := newZunCapsules(t, cloud, "")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Create(canceled, capsules.CreateOpts{TemplateOpts: zunTemplate()}); !errors.Is(err, context.Canceled) {
		t.Fatalf("create: %v", err)
	}
	if _, err := api.Get(canceled, "cap-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("get: %v", err)
	}
	if err := api.Delete(canceled, "cap-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("delete: %v", err)
	}
	for _, err := range api.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("list: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Create(deadline, capsules.CreateOpts{TemplateOpts: zunTemplate()}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("an uncertain create must not be replayed: %d calls", calls.Load())
	}
}
