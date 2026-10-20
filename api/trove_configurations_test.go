package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/db/v1/configurations"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const troveConfigurationBody = `{"configuration":{"id":"cfg-1","name":"web","description":"","datastore_name":"mysql","datastore_version_id":"ver-1","datastore_version_name":"8.0","values":{"autocommit":false,"max_connections":0},"created":"2026-01-02T03:04:05","updated":"2026-01-02T03:04:06","vendor_extra":1}}`

func troveConfigurationAPI(cloud *testcloud.Cloud) *configurations.API {
	return configurations.New(cloud.Client("database", "/trove"))
}

// troveStatusMatrix exercises one route with 200, 201, 202 and 204 and checks
// that exactly the accepted codes succeed. Rejected codes must surface the
// native unexpected-response error with the observed code, after one request.
func troveStatusMatrix(t *testing.T, method, route, body string, accepted []int, call func(*testcloud.Cloud) error) {
	t.Helper()
	for _, status := range []int{200, 201, 202, 204} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc(method+" "+route, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, status, body)
		})
		err := call(cloud)
		want := slices.Contains(accepted, status)
		if want != (err == nil) {
			t.Fatalf("%s %s status %d accepted=%v error=%v", method, route, status, want, err)
		}
		if !want {
			var unexpected gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &unexpected) || unexpected.Actual != status {
				t.Fatalf("%s %s status %d error=%v", method, route, status, err)
			}
		}
		if calls.Load() != 1 {
			t.Fatalf("%s %s status %d calls=%d", method, route, status, calls.Load())
		}
	}
}

func troveIsNative404(err error) bool {
	var unexpected gophercloud.ErrUnexpectedResponseCode
	return errors.As(err, &unexpected) && unexpected.Actual == 404
}

func troveRawBody(t *testing.T, r *http.Request) map[string]json.RawMessage {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("request body: %v", err)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(data, &body); err != nil {
		t.Errorf("request body %q: %v", data, err)
	}
	return body
}

func troveRawInner(t *testing.T, r *http.Request, key string) map[string]string {
	t.Helper()
	var inner map[string]json.RawMessage
	if err := json.Unmarshal(troveRawBody(t, r)[key], &inner); err != nil {
		t.Errorf("request %s envelope: %v", key, err)
	}
	got := map[string]string{}
	for field, value := range inner {
		got[field] = string(value)
	}
	return got
}

func TestTroveConfigurationCreatePreservesFalseZeroAndExtensionFields(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /trove/configurations", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		got := troveRawInner(t, r, "configuration")
		want := map[string]string{
			"name":      `"web"`,
			"values":    `{"autocommit":false,"max_connections":0}`,
			"datastore": `{"type":"mysql"}`,
			"vendor":    `{"enabled":false}`,
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("body=%v", got)
		}
		testcloud.JSON(w, 200, troveConfigurationBody)
	})
	value, err := troveConfigurationAPI(cloud).Create(context.Background(), configurations.CreateOpts{
		Name:      "web",
		Values:    map[string]any{"autocommit": false, "max_connections": 0},
		Datastore: &configurations.DatastoreOpts{Type: "mysql"},
	}, configurations.WithCreateField("vendor", map[string]bool{"enabled": false}))
	if err != nil || value == nil || value.ID != "cfg-1" || value.DatastoreName != "mysql" || value.DatastoreVersionID != "ver-1" ||
		value.Description != "" || value.Created.IsZero() || value.Values["autocommit"] != false || value.Values["max_connections"] != float64(0) {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestTroveConfigurationCreateAcceptsOnly200(t *testing.T) {
	troveStatusMatrix(t, "POST", "/trove/configurations", troveConfigurationBody, []int{200}, func(cloud *testcloud.Cloud) error {
		_, err := troveConfigurationAPI(cloud).Create(context.Background(), configurations.CreateOpts{Name: "web", Values: map[string]any{}})
		return err
	})
}

func TestTroveConfigurationRejectsInvalidInputsAndExtensionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := troveConfigurationAPI(cloud)
	ctx := context.Background()
	var missing gophercloud.ErrMissingInput
	if _, err := api.Create(ctx, configurations.CreateOpts{Values: map[string]any{}}); !errors.As(err, &missing) {
		t.Fatalf("missing name: %v", err)
	}
	if _, err := api.Create(ctx, configurations.CreateOpts{Name: "web"}); !errors.As(err, &missing) {
		t.Fatalf("missing values: %v", err)
	}
	if _, err := api.Create(ctx, configurations.CreateOpts{Name: "web", Values: map[string]any{}}, configurations.WithCreateField("name", "shadow")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected create field: %v", err)
	}
	if _, err := api.Create(ctx, configurations.CreateOpts{Name: "web", Values: map[string]any{}}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil create option: %v", err)
	}
	if err := api.Update(ctx, "cfg-1", configurations.UpdateOpts{Name: "a"}, configurations.WithUpdateField("name", "shadow")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected update field: %v", err)
	}
	if err := api.Update(ctx, "cfg-1", configurations.UpdateOpts{}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil update option: %v", err)
	}
	if err := api.Replace(ctx, "cfg-1", configurations.UpdateOpts{Values: map[string]any{"a": 1}}, configurations.WithReplaceField("values", 1)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected replace field: %v", err)
	}
	if err := api.Replace(ctx, "cfg-1", configurations.UpdateOpts{}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil replace option: %v", err)
	}
}

func TestTroveConfigurationUpdateAndReplaceBodiesAndStatusCodes(t *testing.T) {
	cloud := testcloud.New(t)
	bodies := map[string]map[string]string{}
	for _, method := range []string{"PATCH", "PUT"} {
		cloud.Mux.HandleFunc(method+" /trove/configurations/cfg-1", func(w http.ResponseWriter, r *http.Request) {
			bodies[method] = troveRawInner(t, r, "configuration")
			w.WriteHeader(202)
		})
	}
	api := troveConfigurationAPI(cloud)
	empty := ""
	opts := configurations.UpdateOpts{Values: map[string]any{"autocommit": false}, Description: &empty}
	if err := api.Update(context.Background(), "cfg-1", opts, configurations.WithUpdateField("vendor", false)); err != nil {
		t.Fatal(err)
	}
	if err := api.Replace(context.Background(), "cfg-1", opts, configurations.WithReplaceField("vendor", false)); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"values": `{"autocommit":false}`, "description": `""`, "vendor": `false`}
	for _, method := range []string{"PATCH", "PUT"} {
		if !reflect.DeepEqual(bodies[method], want) {
			t.Fatalf("%s body=%v", method, bodies[method])
		}
	}
	// A zero UpdateOpts omits every field: name "" and a nil description are not sent.
	cloud = testcloud.New(t)
	cloud.Mux.HandleFunc("PUT /trove/configurations/cfg-1", func(w http.ResponseWriter, r *http.Request) {
		if got := troveRawInner(t, r, "configuration"); len(got) != 0 {
			t.Errorf("zero replace body=%v", got)
		}
		w.WriteHeader(202)
	})
	if err := troveConfigurationAPI(cloud).Replace(context.Background(), "cfg-1", configurations.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	// Update uses PATCH defaults (200, 202, 204); Replace uses PUT defaults (201, 202).
	troveStatusMatrix(t, "PATCH", "/trove/configurations/cfg-1", `{}`, []int{200, 202, 204}, func(cloud *testcloud.Cloud) error {
		return troveConfigurationAPI(cloud).Update(context.Background(), "cfg-1", configurations.UpdateOpts{Name: "n"})
	})
	troveStatusMatrix(t, "PUT", "/trove/configurations/cfg-1", `{}`, []int{201, 202}, func(cloud *testcloud.Cloud) error {
		return troveConfigurationAPI(cloud).Replace(context.Background(), "cfg-1", configurations.UpdateOpts{Name: "n"})
	})
}

func TestTroveConfigurationGetAndDeleteStatusesAndMissingPolicies(t *testing.T) {
	troveStatusMatrix(t, "GET", "/trove/configurations/cfg-1", troveConfigurationBody, []int{200}, func(cloud *testcloud.Cloud) error {
		value, err := troveConfigurationAPI(cloud).Get(context.Background(), "cfg-1")
		if err == nil && (value == nil || value.Name != "web") {
			t.Errorf("value=%+v", value)
		}
		return err
	})
	troveStatusMatrix(t, "DELETE", "/trove/configurations/cfg-1", `{}`, []int{202, 204}, func(cloud *testcloud.Cloud) error {
		return troveConfigurationAPI(cloud).Delete(context.Background(), "cfg-1")
	})
	cloud := testcloud.New(t)
	var deletes atomic.Int32
	cloud.Mux.HandleFunc("GET /trove/configurations/missing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("DELETE /trove/configurations/missing", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		testcloud.JSON(w, 404, `{}`)
	})
	api := troveConfigurationAPI(cloud)
	ctx := context.Background()
	if _, err := api.Get(ctx, "missing"); !troveIsNative404(err) || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
	if err := api.Delete(ctx, "missing"); !troveIsNative404(err) || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native delete keeps the 404 response error: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("missing"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("Remove with missing error: %v", err)
	}
	if deletes.Load() == 0 {
		t.Fatalf("Remove never attempted the delete")
	}
}

func TestTroveConfigurationListIsSinglePageAndRejectsExtensionQueries(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /trove/configurations", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		next := cloud.Server.URL + "/trove/configurations?marker=cfg-1"
		testcloud.JSON(w, 200, `{"configurations":[{"id":"cfg-1","name":"one"},{"id":"cfg-2","name":"two"}],"configurations_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	api := troveConfigurationAPI(cloud)
	var ids []string
	for value, err := range api.List(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"cfg-1", "cfg-2"}) || len(requests) != 1 || len(requests[0]) != 0 {
		t.Fatalf("ids=%v requests=%v: pinned native list is a single page without filters", ids, requests)
	}
	requests = nil
	limited, err := api.All(context.Background(), resource.WithMaxItems(1))
	if err != nil || len(limited) != 1 || limited[0].ID != "cfg-1" {
		t.Fatalf("limited=%v error=%v", limited, err)
	}
	// Configurations expose no status attribute: a status filter is unsupported
	// (not silently dropped) and no request is sent.
	requests = nil
	if _, err := api.All(context.Background(), resource.WithStatus("ACTIVE")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("status filter: %v", err)
	}
	if len(requests) != 0 {
		t.Fatalf("status filter reached HTTP: %v", requests)
	}
	requests = nil
	if _, err := api.All(context.Background(), resource.WithQuery("vendor", "x")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("extension query: %v", err)
	}
	if len(requests) != 0 {
		t.Fatalf("unsupported query reached HTTP: %v", requests)
	}
}

func TestTroveConfigurationListAcceptsNoContentAndRejectsOtherStatuses(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /trove/configurations", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	got, err := troveConfigurationAPI(cloud).All(context.Background())
	if err != nil || len(got) != 0 {
		t.Fatalf("empty 204 list: %v %v", got, err)
	}
	cloud = testcloud.New(t)
	cloud.Mux.HandleFunc("GET /trove/configurations", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	for _, err := range troveConfigurationAPI(cloud).List(context.Background()) {
		if !troveIsNative404(err) {
			t.Fatalf("list 404: %v", err)
		}
	}
}

func TestTroveConfigurationFindResolvesExactNamesAndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	var lists, gets atomic.Int32
	cloud.Mux.HandleFunc("GET /trove/configurations", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		testcloud.JSON(w, 200, `{"configurations":[{"id":"cfg-1","name":"web"},{"id":"cfg-2","name":"web-2"},{"id":"cfg-3","name":"dup"},{"id":"cfg-4","name":"dup"}]}`)
	})
	cloud.Mux.HandleFunc("GET /trove/configurations/cfg-1", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		testcloud.JSON(w, 200, troveConfigurationBody)
	})
	api := troveConfigurationAPI(cloud)
	ctx := context.Background()
	if value, err := api.Find(ctx, resource.Name("web")); err != nil || value == nil || value.ID != "cfg-1" {
		t.Fatalf("name find: %+v %v", value, err)
	}
	if value, err := api.Find(ctx, resource.ID("cfg-1")); err != nil || value == nil || value.Name != "web" || gets.Load() != 1 {
		t.Fatalf("id find: %+v %v gets=%d", value, err, gets.Load())
	}
	if _, err := api.Find(ctx, resource.Name("dup")); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatalf("duplicate name: %v", err)
	}
	if _, err := api.Find(ctx, resource.Name("absent")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing name: %v", err)
	}
	if value, err := api.Find(ctx, resource.Name("absent"), resource.WithIgnoreMissing()); err != nil || value != nil {
		t.Fatalf("ignore missing: %+v %v", value, err)
	}
}

func TestTroveConfigurationListInstancesFollowsLinks(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /trove/configurations/cfg-1/instances", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "i-1" {
			testcloud.JSON(w, 200, `{"instances":[{"id":"i-2","name":"two","status":"ACTIVE"}],"instances_links":[]}`)
			return
		}
		next := cloud.Server.URL + "/trove/configurations/cfg-1/instances?marker=i-1"
		testcloud.JSON(w, 200, `{"instances":[{"id":"i-1","name":"one","status":"BUILD","volume":{"size":0}}],"instances_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	var ids []string
	for value, err := range troveConfigurationAPI(cloud).ListInstances(context.Background(), "cfg-1") {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID+":"+value.Status)
	}
	if !reflect.DeepEqual(ids, []string{"i-1:BUILD", "i-2:ACTIVE"}) || len(requests) != 2 || requests[1].Get("marker") != "i-1" {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
}

func TestTroveConfigurationParametersListAndGet(t *testing.T) {
	cloud := testcloud.New(t)
	const param = `{"name":"max_connections","type":"integer","min":0,"max":4096,"restart_required":false}`
	var paths []string
	for _, route := range []string{
		"/trove/datastores/mysql/versions/ver-1/parameters",
		"/trove/datastores/versions/ver-1/parameters",
	} {
		cloud.Mux.HandleFunc("GET "+route, func(w http.ResponseWriter, r *http.Request) {
			paths = append(paths, r.URL.Path)
			testcloud.JSON(w, 200, `{"configuration-parameters":[`+param+`,{"name":"read_only","type":"boolean","restart_required":true}]}`)
		})
		cloud.Mux.HandleFunc("GET "+route+"/max_connections", func(w http.ResponseWriter, r *http.Request) {
			paths = append(paths, r.URL.Path)
			testcloud.JSON(w, 200, param)
		})
		cloud.Mux.HandleFunc("GET "+route+"/missing", func(w http.ResponseWriter, r *http.Request) {
			testcloud.JSON(w, 404, `{}`)
		})
	}
	api := troveConfigurationAPI(cloud)
	ctx := context.Background()
	check := func(label string, params []*configurations.Param) {
		t.Helper()
		if len(params) != 2 || params[0].Name != "max_connections" || params[0].Max != 4096 || params[0].Min != 0 || params[0].RestartRequired ||
			params[1].Name != "read_only" || !params[1].RestartRequired {
			t.Fatalf("%s params=%+v", label, params)
		}
	}
	var datastoreParams, globalParams []*configurations.Param
	for value, err := range api.ListDatastoreParams(ctx, "mysql", "ver-1") {
		if err != nil {
			t.Fatal(err)
		}
		datastoreParams = append(datastoreParams, value)
	}
	for value, err := range api.ListGlobalParams(ctx, "ver-1") {
		if err != nil {
			t.Fatal(err)
		}
		globalParams = append(globalParams, value)
	}
	check("datastore", datastoreParams)
	check("global", globalParams)
	one, err := api.GetDatastoreParam(ctx, "mysql", "ver-1", "max_connections")
	if err != nil || one.Name != "max_connections" || one.Type != "integer" || one.Max != 4096 || one.RestartRequired {
		t.Fatalf("datastore param=%+v error=%v", one, err)
	}
	one, err = api.GetGlobalParam(ctx, "ver-1", "max_connections")
	if err != nil || one.Name != "max_connections" || one.Max != 4096 {
		t.Fatalf("global param=%+v error=%v", one, err)
	}
	if _, err := api.GetDatastoreParam(ctx, "mysql", "ver-1", "missing"); !troveIsNative404(err) {
		t.Fatalf("datastore param 404: %v", err)
	}
	if _, err := api.GetGlobalParam(ctx, "ver-1", "missing"); !troveIsNative404(err) {
		t.Fatalf("global param 404: %v", err)
	}
	if len(paths) != 4 {
		t.Fatalf("paths=%v", paths)
	}
}

func TestTroveConfigurationCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
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
	api := troveConfigurationAPI(cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	drain := func(label string, seq func(func(struct{}, error) bool)) {
		t.Helper()
		var got error
		seq(func(_ struct{}, err error) bool { got = err; return false })
		if !errors.Is(got, context.Canceled) {
			t.Errorf("%s: %v", label, got)
		}
	}
	checks := map[string]error{}
	_, checks["Create"] = api.Create(canceled, configurations.CreateOpts{Name: "n", Values: map[string]any{}})
	_, checks["Get"] = api.Get(canceled, "cfg-1")
	_, checks["GetDatastoreParam"] = api.GetDatastoreParam(canceled, "d", "v", "p")
	_, checks["GetGlobalParam"] = api.GetGlobalParam(canceled, "v", "p")
	checks["Update"] = api.Update(canceled, "cfg-1", configurations.UpdateOpts{Name: "n"})
	checks["Replace"] = api.Replace(canceled, "cfg-1", configurations.UpdateOpts{Name: "n"})
	checks["Delete"] = api.Delete(canceled, "cfg-1")
	for label, err := range checks {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("pre-canceled %s: %v", label, err)
		}
	}
	drain("List", func(yield func(struct{}, error) bool) {
		for _, err := range api.List(canceled) {
			if !yield(struct{}{}, err) {
				return
			}
		}
	})
	drain("ListInstances", func(yield func(struct{}, error) bool) {
		for _, err := range api.ListInstances(canceled, "cfg-1") {
			if !yield(struct{}{}, err) {
				return
			}
		}
	})
	drain("ListDatastoreParams", func(yield func(struct{}, error) bool) {
		for _, err := range api.ListDatastoreParams(canceled, "d", "v") {
			if !yield(struct{}{}, err) {
				return
			}
		}
	})
	drain("ListGlobalParams", func(yield func(struct{}, error) bool) {
		for _, err := range api.ListGlobalParams(canceled, "v") {
			if !yield(struct{}{}, err) {
				return
			}
		}
	})
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Create(deadline, configurations.CreateOpts{Name: "n", Values: map[string]any{}}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
