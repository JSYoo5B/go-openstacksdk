package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/softwareconfigs"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/softwaredeployments"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const r2ManilaHeatConfigID = "11111111-1111-4111-8111-111111111111"
const r2ManilaHeatDeploymentID = "22222222-2222-4222-8222-222222222222"

func r2ManilaHeatBody(t *testing.T, r *http.Request) map[string]json.RawMessage {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(data, &body); err != nil {
		t.Fatalf("body %q: %v", data, err)
	}
	return body
}

func TestR2ManilaHeatSoftwareConfigCreateOmitsNullsAndPreservesExactJSON(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var requests atomic.Int32
	cloud.Mux.HandleFunc("/heat/v1/software_configs", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		body := r2ManilaHeatBody(t, r)
		if r.Method != "POST" || len(body) != 5 || body["software_config"] != nil || string(body["name"]) != `""` || string(body["group"]) != "null" ||
			string(body["config"]) != `{"big":9007199254740993,"ok":false}` || string(body["inputs"]) != `[]` || string(body["options"]) != `{"x":0}` || body["outputs"] != nil {
			t.Errorf("%s body=%v", r.Method, body)
		}
		w.Header().Set("X-Request-Id", "req-config")
		testcloud.JSON(w, 200, `{"software_config":{"id":"`+r2ManilaHeatConfigID+`","name":"","group":null,"config":{"big":9007199254740993,"ok":false},"inputs":[],"outputs":null,"options":{"x":0},"creation_time":"2026-10-01T00:00:00Z","future":1}}`)
	})
	api := softwareconfigs.New(cloud.Client("orchestration", "/heat/v1"))
	value, err := api.Create(context.Background(), softwareconfigs.CreateOpts{Name: request.Present(""), Group: request.Null[string]()},
		softwareconfigs.WithCreateConfig(map[string]any{"big": json.Number("9007199254740993"), "ok": false}),
		softwareconfigs.WithCreateInputs([]any{}), softwareconfigs.WithCreateConfigOptions(map[string]int{"x": 0}))
	if err != nil {
		t.Fatal(err)
	}
	if value.ID != r2ManilaHeatConfigID || string(value.Config) != `{"big":9007199254740993,"ok":false}` || string(value.Body["outputs"]) != "null" || string(value.Body["future"]) != "1" ||
		value.CreationTime != "2026-10-01T00:00:00Z" || value.StatusCode != 200 || value.Header.Get("X-Request-Id") != "req-config" || requests.Load() != 1 {
		t.Fatalf("value=%+v requests=%d", value, requests.Load())
	}
}

func TestR2ManilaHeatSoftwareConfigCreateEmptyBodyAndStatusPolicy(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		status int
		fail   bool
	}{{200, false}, {201, false}, {202, true}, {500, true}} {
		cloud := testcloud.New(t)
		var requests atomic.Int32
		cloud.Mux.HandleFunc("/heat/v1/software_configs", func(w http.ResponseWriter, r *http.Request) {
			requests.Add(1)
			if body := r2ManilaHeatBody(t, r); len(body) != 0 {
				t.Errorf("body=%v", body)
			}
			testcloud.JSON(w, test.status, `{"software_config":{"id":"`+r2ManilaHeatConfigID+`"}}`)
		})
		_, err := softwareconfigs.New(cloud.Client("orchestration", "/heat/v1")).Create(context.Background(), softwareconfigs.CreateOpts{})
		if (err != nil) != test.fail || requests.Load() != 1 {
			t.Fatalf("status %d err=%v requests=%d", test.status, err, requests.Load())
		}
	}
}

func TestR2ManilaHeatSoftwareConfigCreateRejectsProtectedFieldsBeforeHTTPAndCancelled(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var requests atomic.Int32
	cloud.Mux.HandleFunc("/heat/v1/software_configs", func(w http.ResponseWriter, r *http.Request) { requests.Add(1) })
	api := softwareconfigs.New(cloud.Client("orchestration", "/heat/v1"))
	for _, option := range []softwareconfigs.CreateOption{
		softwareconfigs.WithCreateField("id", "x"), softwareconfigs.WithCreateField("creation_time", "x"),
		softwareconfigs.WithCreateField("config", "x"), softwareconfigs.WithCreateField("name", "x"),
		softwareconfigs.WithCreateHeader("X-Auth-Token", "x"), softwareconfigs.WithCreateConfig(func() {}),
	} {
		if _, err := api.Create(context.Background(), softwareconfigs.CreateOpts{}, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("err=%v", err)
		}
	}
	if _, err := api.Create(context.Background(), softwareconfigs.CreateOpts{}, softwareconfigs.WithCreateField("custom", map[string]int{"a": 1})); err != nil {
		// The fixture handler returns an empty 200, which is not a valid envelope; only HTTP use matters here.
		if requests.Load() != 1 {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	before := requests.Load()
	if _, err := api.Create(ctx, softwareconfigs.CreateOpts{}); !errors.Is(err, context.Canceled) || requests.Load() != before {
		t.Fatalf("err=%v requests=%d", err, requests.Load())
	}
}

func TestR2ManilaHeatSoftwareConfigGetDeleteSemantics(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var gets, deletes atomic.Int32
	cloud.Mux.HandleFunc("/heat/v1/software_configs/"+r2ManilaHeatConfigID, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			gets.Add(1)
			testcloud.JSON(w, 200, `{"software_config":{"id":"`+r2ManilaHeatConfigID+`","name":"n","config":"#!/bin/sh"}}`)
		case "DELETE":
			deletes.Add(1)
			w.WriteHeader(204)
		}
	})
	cloud.Mux.HandleFunc("/heat/v1/software_configs/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{"error":{"code":404}}`) })
	cloud.Mux.HandleFunc("/heat/v1/software_configs/broken", func(w http.ResponseWriter, r *http.Request) { deletes.Add(1); testcloud.JSON(w, 500, `{}`) })
	api := softwareconfigs.New(cloud.Client("orchestration", "/heat/v1"))
	value, err := api.Get(context.Background(), r2ManilaHeatConfigID)
	if err != nil || string(value.Config) != `"#!/bin/sh"` || value.Name != "n" {
		t.Fatal(value, err)
	}
	if _, err := api.Get(context.Background(), "missing"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	for _, id := range []string{"a/b", "", "..", "a b"} {
		if _, err := api.Get(context.Background(), id); err == nil {
			t.Fatalf("id %q accepted", id)
		}
	}
	if err := api.Delete(context.Background(), r2ManilaHeatConfigID); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(context.Background(), "missing"); err != nil {
		t.Fatalf("default ignore missing: %v", err)
	}
	if err := api.Delete(context.Background(), "missing", resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if err := api.Delete(context.Background(), "broken"); err == nil || deletes.Load() != 2 {
		t.Fatalf("err=%v deletes=%d", err, deletes.Load())
	}
}

func TestR2ManilaHeatSoftwareConfigListQueryPagingAndErrors(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var queries []string
	cloud.Mux.HandleFunc("/heat/v1/software_configs", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		switch r.URL.Query().Get("marker") {
		case "":
			testcloud.JSON(w, 200, `{"software_configs":[{"id":"a","name":"A"},{"id":"b","name":"B"}]}`)
		case "b":
			testcloud.JSON(w, 200, `{"software_configs":[{"id":"c"}]}`)
		default:
			testcloud.JSON(w, 200, `{"software_configs":[]}`)
		}
	})
	api := softwareconfigs.New(cloud.Client("orchestration", "/heat/v1"))
	all, err := api.All(context.Background())
	if err != nil || len(all) != 2 || len(queries) != 1 || queries[0] != "" {
		t.Fatalf("all=%v err=%v queries=%v", all, err, queries)
	}
	queries = nil
	all, err = api.All(context.Background(), softwareconfigs.WithListLimit(2), softwareconfigs.WithListQuery("tenant_hint", "x"))
	if err != nil || len(all) != 3 || all[2].ID != "c" || len(queries) != 3 || !strings.Contains(queries[1], "marker=b") || !strings.Contains(queries[1], "limit=2") || !strings.Contains(queries[0], "tenant_hint=x") {
		t.Fatalf("all=%v err=%v queries=%v", all, err, queries)
	}
	n := len(queries)
	for _, option := range []softwareconfigs.ListOption{softwareconfigs.WithListLimit(0), softwareconfigs.WithListMarker(""), softwareconfigs.WithListQuery("limit", "1"), softwareconfigs.WithListQuery("marker", "1")} {
		if _, err := api.All(context.Background(), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("err=%v", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.All(ctx); !errors.Is(err, context.Canceled) || len(queries) != n {
		t.Fatalf("err=%v queries=%v", err, queries)
	}
}

func TestR2ManilaHeatSoftwareDeploymentCreateAndUpdateSendOnlyChosenMembers(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var puts, posts atomic.Int32
	cloud.Mux.HandleFunc("/heat/v1/software_deployments", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		body := r2ManilaHeatBody(t, r)
		if r.Method != "POST" || len(body) != 5 || string(body["config_id"]) != `"cfg"` || string(body["server_id"]) != `"srv"` || string(body["action"]) != `"CREATE"` ||
			string(body["status"]) != `"IN_PROGRESS"` || string(body["input_values"]) != `{"n":9007199254740993}` || body["software_deployment"] != nil {
			t.Errorf("body=%v", body)
		}
		testcloud.JSON(w, 200, `{"software_deployment":{"id":"`+r2ManilaHeatDeploymentID+`","config_id":"cfg","server_id":"srv","status":"IN_PROGRESS","input_values":{"n":9007199254740993},"output_values":null,"creation_time":"t0","updated_time":null,"stack_user_project_id":"p"}}`)
	})
	cloud.Mux.HandleFunc("/heat/v1/software_deployments/"+r2ManilaHeatDeploymentID, func(w http.ResponseWriter, r *http.Request) {
		puts.Add(1)
		body := r2ManilaHeatBody(t, r)
		if r.Method != "PUT" || len(body) != 3 || string(body["status"]) != `"COMPLETE"` || string(body["status_reason"]) != "null" || string(body["output_values"]) != `{}` {
			t.Errorf("%s body=%v", r.Method, body)
		}
		testcloud.JSON(w, 200, `{"software_deployment":{"id":"`+r2ManilaHeatDeploymentID+`","status":"COMPLETE","status_reason":null,"output_values":{},"updated_time":"t1"}}`)
	})
	api := softwaredeployments.New(cloud.Client("orchestration", "/heat/v1"))
	created, err := api.Create(context.Background(), softwaredeployments.CreateOpts{}, softwaredeployments.WithCreateConfigID("cfg"), softwaredeployments.WithCreateServerID("srv"),
		softwaredeployments.WithCreateAction("CREATE"), softwaredeployments.WithCreateStatus("IN_PROGRESS"),
		softwaredeployments.WithCreateInputValues(map[string]any{"n": json.Number("9007199254740993")}))
	if err != nil || created.ID != r2ManilaHeatDeploymentID || string(created.InputValues) != `{"n":9007199254740993}` || string(created.Body["output_values"]) != "null" || created.UpdatedTime != "" || created.StackUserProjectID != "p" {
		t.Fatalf("created=%+v err=%v", created, err)
	}
	updated, err := api.Update(context.Background(), created.ID, softwaredeployments.UpdateOpts{StatusReason: request.Null[string]()},
		softwaredeployments.WithUpdateStatus("COMPLETE"), softwaredeployments.WithUpdateOutputValues(map[string]any{}))
	if err != nil || updated.Status != "COMPLETE" || string(updated.Body["status_reason"]) != "null" || updated.UpdatedTime != "t1" || puts.Load() != 1 || posts.Load() != 1 {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
}

func TestR2ManilaHeatSoftwareDeploymentUpdateRejectsEmptyAndProtectedBeforeHTTP(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var requests atomic.Int32
	cloud.Mux.HandleFunc("/heat/v1/software_deployments/", func(w http.ResponseWriter, r *http.Request) { requests.Add(1) })
	api := softwaredeployments.New(cloud.Client("orchestration", "/heat/v1"))
	ctx := context.Background()
	cases := []struct {
		id      string
		options []softwaredeployments.UpdateOption
	}{
		{r2ManilaHeatDeploymentID, nil},
		{r2ManilaHeatDeploymentID, []softwaredeployments.UpdateOption{softwaredeployments.WithUpdateField("id", "x")}},
		{r2ManilaHeatDeploymentID, []softwaredeployments.UpdateOption{softwaredeployments.WithUpdateField("updated_time", "x")}},
		{r2ManilaHeatDeploymentID, []softwaredeployments.UpdateOption{softwaredeployments.WithUpdateStatus("a"), softwaredeployments.WithUpdateField("status", "b")}},
		{"a/b", []softwaredeployments.UpdateOption{softwaredeployments.WithUpdateStatus("a")}},
	}
	for i, c := range cases {
		if _, err := api.Update(ctx, c.id, softwaredeployments.UpdateOpts{}, c.options...); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("case %d err=%v", i, err)
		}
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := api.Update(cancelled, r2ManilaHeatDeploymentID, softwaredeployments.UpdateOpts{}, softwaredeployments.WithUpdateStatus("a")); !errors.Is(err, context.Canceled) || requests.Load() != 0 {
		t.Fatalf("err=%v requests=%d", err, requests.Load())
	}
}

func TestR2ManilaHeatSoftwareDeploymentUpdateFailureIsNotReplayed(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var requests atomic.Int32
	cloud.Mux.HandleFunc("/heat/v1/software_deployments/"+r2ManilaHeatDeploymentID, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		testcloud.JSON(w, 500, `{"error":"boom"}`)
	})
	_, err := softwaredeployments.New(cloud.Client("orchestration", "/heat/v1")).Update(context.Background(), r2ManilaHeatDeploymentID, softwaredeployments.UpdateOpts{}, softwaredeployments.WithUpdateStatus("a"))
	if err == nil || requests.Load() != 1 {
		t.Fatalf("err=%v requests=%d", err, requests.Load())
	}
}

func TestR2ManilaHeatSoftwareDeploymentGetDeleteListServerFilter(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var queries []string
	cloud.Mux.HandleFunc("/heat/v1/software_deployments", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		testcloud.JSON(w, 200, `{"software_deployments":[{"id":"a","server_id":"srv"}]}`)
	})
	cloud.Mux.HandleFunc("/heat/v1/software_deployments/"+r2ManilaHeatDeploymentID, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			w.WriteHeader(204)
			return
		}
		testcloud.JSON(w, 200, `{"software_deployment":{"id":"`+r2ManilaHeatDeploymentID+`","status":"X"}}`)
	})
	cloud.Mux.HandleFunc("/heat/v1/software_deployments/gone", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api := softwaredeployments.New(cloud.Client("orchestration", "/heat/v1"))
	rows, err := api.All(context.Background(), softwaredeployments.WithListServerID("srv"))
	if err != nil || len(rows) != 1 || len(queries) != 1 || queries[0] != "server_id=srv" {
		t.Fatalf("rows=%v err=%v queries=%v", rows, err, queries)
	}
	for _, option := range []softwaredeployments.ListOption{softwaredeployments.WithListServerID(""), softwaredeployments.WithListQuery("server_id", "x")} {
		if _, err := api.All(context.Background(), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if value, err := api.Get(context.Background(), r2ManilaHeatDeploymentID); err != nil || value.Status != "X" {
		t.Fatal(value, err)
	}
	if _, err := api.Get(context.Background(), "gone"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if err := api.Delete(context.Background(), r2ManilaHeatDeploymentID); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(context.Background(), "gone"); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(context.Background(), "gone", resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
}
