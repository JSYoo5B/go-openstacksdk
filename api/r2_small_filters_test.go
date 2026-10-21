package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/clustertemplates"
	"github.com/JSYoo5B/go-openstacksdk/db/v1/databases"
	"github.com/JSYoo5B/go-openstacksdk/db/v1/flavors"
	"github.com/JSYoo5B/go-openstacksdk/db/v1/instances"
	"github.com/JSYoo5B/go-openstacksdk/db/v1/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestR2SmallTroveBodyFiltersFlavorsInstancesDatabasesUsers(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("database", "/v1.0/tenant")
	cloud.Mux.HandleFunc("/v1.0/tenant/flavors", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"flavors":[{"id":1,"name":"small","ram":512},{"id":2,"name":"big","ram":2048}]}`)
	})
	cloud.Mux.HandleFunc("/v1.0/tenant/instances", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"instances":[{"id":"i1","name":"a","status":"ACTIVE","datastore":{"type":"mysql","version":"8"},"created":"2024-01-01T00:00:00"},{"id":"i2","name":"b","status":"BUILD","datastore":{"type":"pg","version":"15"}}]}`)
	})
	cloud.Mux.HandleFunc("/v1.0/tenant/instances/i1/databases", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"databases":[{"name":"d1","character_set":"utf8","collate":"c1"},{"name":"d2","character_set":"latin1","collate":"c2"}]}`)
	})
	cloud.Mux.HandleFunc("/v1.0/tenant/instances/i1/users", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"users":[{"name":"u1","host":"%","databases":[{"name":"d1"}]},{"name":"u2","host":"10.0.0.1","databases":[]}]}`)
	})
	ctx := context.Background()
	got, err := flavors.New(client).FilteredResources().All(ctx, resource.WithBodyFilter("ram", 2048))
	if err != nil || len(got) != 1 || got[0].Name != "big" {
		t.Fatalf("flavors=%v err=%v", got, err)
	}
	inst, err := instances.New(client).FilteredResources().All(ctx, resource.WithBodyFilter("datastore", map[string]any{"type": "mysql"}), resource.WithBodyFilter("status", "ACTIVE"))
	if err != nil || len(inst) != 1 || inst[0].ID != "i1" {
		t.Fatalf("instances=%v err=%v", inst, err)
	}
	// Python attribute created_at is the wire field created; missing fields are null.
	inst, err = instances.New(client).FilteredResources().All(ctx, resource.WithBodyFilter("created_at", nil))
	if err != nil || len(inst) != 1 || inst[0].ID != "i2" {
		t.Fatalf("null filter instances=%v err=%v", inst, err)
	}
	scope, err := databases.New(client).InInstance(ctx, resource.ID("i1"))
	if err != nil {
		t.Fatal(err)
	}
	dbs, err := scope.FilteredResources().All(ctx, resource.WithBodyFilter("character_set", "latin1"))
	if err != nil || len(dbs) != 1 || dbs[0].Name != "d2" {
		t.Fatalf("databases=%v err=%v", dbs, err)
	}
	uscope, err := users.New(client).InInstance(ctx, resource.ID("i1"), users.WithHost("%"))
	if err != nil {
		t.Fatal(err)
	}
	us, err := uscope.FilteredResources().All(ctx, resource.WithBodyFilter("databases", []any{}))
	if err != nil || len(us) != 1 || us[0].Name != "u2" {
		t.Fatalf("users=%v err=%v", us, err)
	}
	// Unknown attributes and server query extensions fail before HTTP.
	if _, err := flavors.New(client).FilteredResources().All(ctx, resource.WithBodyFilter("nope", 1)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("unknown filter: %v", err)
	}
	if _, err := flavors.New(client).FilteredResources().All(ctx, resource.WithQuery("limit", "1")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("query: %v", err)
	}
}

func TestR2SmallTroveUpdateAttributes(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("database", "/v1.0/tenant")
	var methods []string
	var body map[string]map[string]json.RawMessage
	cloud.Mux.HandleFunc("/v1.0/tenant/instances/i1", func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		w.WriteHeader(202)
	})
	api := instances.New(client)
	receipt, err := api.UpdateAttributes(context.Background(), resource.ID("i1"), instances.UpdateAttributes{
		Name: request.Present(""), Hostname: request.Null[string](), Volume: request.Present(json.RawMessage(`{"size":2}`)),
	}, instances.WithUpdateAttributeField("custom", false))
	if err != nil || receipt.StatusCode != 202 || len(receipt.Body) != 0 {
		t.Fatalf("receipt=%v err=%v", receipt, err)
	}
	b := body["instance"]
	if !reflect.DeepEqual(methods, []string{"PUT"}) || string(b["name"]) != `""` || string(b["hostname"]) != "null" || string(b["volume"]) != `{"size":2}` || string(b["custom"]) != "false" || len(b) != 4 {
		t.Fatalf("methods=%v body=%v", methods, b)
	}
	for name, call := range map[string]func() error{
		"empty": func() error {
			_, err := api.UpdateAttributes(context.Background(), resource.ID("i1"), instances.UpdateAttributes{})
			return err
		},
		"core": func() error {
			_, err := api.UpdateAttributes(context.Background(), resource.ID("i1"), instances.UpdateAttributes{Name: request.Present("x")}, instances.WithUpdateAttributeField("name", "y"))
			return err
		},
		"id": func() error {
			_, err := api.UpdateAttributes(context.Background(), resource.ID("i1"), instances.UpdateAttributes{Name: request.Present("x")}, instances.WithUpdateAttributeField("id", "y"))
			return err
		},
		"auth hdr": func() error {
			_, err := api.UpdateAttributes(context.Background(), resource.ID("i1"), instances.UpdateAttributes{Name: request.Present("x")}, instances.WithUpdateAttributeHeader("X-Auth-Token", "y"))
			return err
		},
	} {
		if err := call(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if len(methods) != 1 {
		t.Fatalf("rejected input reached HTTP: %v", methods)
	}
	cloud.Mux.HandleFunc("/v1.0/tenant/instances/i2", func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, "i2")
		testcloud.JSON(w, 500, `{"x":1}`)
	})
	if _, err := api.UpdateAttributes(context.Background(), resource.ID("i2"), instances.UpdateAttributes{Name: request.Present("x")}); err == nil || len(methods) != 2 {
		t.Fatalf("server error must surface once: %v %v", err, methods)
	}
}

func TestR2SmallMagnumPythonListPagingAndFilters(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("container-infra", "/v1")
	var markers []string
	cloud.Mux.HandleFunc("/v1/clusters", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		markers = append(markers, q.Get("marker")+"|"+q.Get("limit"))
		switch q.Get("marker") {
		case "":
			testcloud.JSON(w, 200, `{"clusters":[{"uuid":"u1","name":"a","node_count":3,"master_lb_enabled":true},{"uuid":"u2","name":"b","node_count":1,"master_lb_enabled":false}]}`)
		case "u2":
			testcloud.JSON(w, 200, `{"clusters":[{"uuid":"u3","name":"c","node_count":3}]}`)
		default:
			testcloud.JSON(w, 200, `{"clusters":[]}`)
		}
	})
	ctx := context.Background()
	all, err := clusters.New(client).PythonResources().All(ctx, resource.WithPageSize(2))
	if err != nil || len(all) != 3 || all[2].Cluster.UUID != "u3" {
		t.Fatalf("all=%v err=%v", all, err)
	}
	if !reflect.DeepEqual(markers, []string{"|2", "u2|2", "u3|2"}) {
		t.Fatalf("markers=%v", markers)
	}
	markers = nil
	// Integer filter and boolean alias.
	got, err := clusters.New(client).PythonResources().All(ctx, resource.WithBodyFilter("node_count", 3))
	if err != nil || len(got) != 1 || len(markers) != 1 {
		t.Fatalf("got=%v err=%v markers=%v", got, err, markers)
	}
	got, err = clusters.New(client).PythonResources().All(ctx, resource.WithBodyFilter("is_master_lb_enabled", false))
	if err != nil || len(got) != 1 {
		t.Fatalf("bool alias got=%v err=%v", got, err)
	}
	if _, err := clusters.New(client).PythonResources().All(ctx, resource.WithBodyFilter("bogus", 1)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestR2SmallMagnumTemplatesPythonListFilterAndLinkHeader(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("container-infra", "/v1")
	var calls int
	cloud.Mux.HandleFunc("/v1/clustertemplates", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("marker") == "m2" {
			testcloud.JSON(w, 200, `{"clustertemplates":[{"uuid":"t2","name":"two","public":false,"labels":{"a":"1","b":"2"}}]}`)
			return
		}
		w.Header().Set("Link", `<`+cloud.Server.URL+`/v1/clustertemplates?marker=m2>; rel="next"`)
		testcloud.JSON(w, 200, `{"clustertemplates":[{"uuid":"t1","name":"one","public":true,"labels":{"a":"1"}}]}`)
	})
	ctx := context.Background()
	all, err := clustertemplates.New(client).PythonResources().All(ctx, resource.WithBodyFilter("is_public", false), resource.WithBodyFilter("labels", map[string]any{"b": "2"}))
	if err != nil || len(all) != 1 || all[0].ClusterTemplate.UUID != "t2" || calls != 2 {
		t.Fatalf("all=%v err=%v calls=%d", all, err, calls)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := clustertemplates.New(client).PythonResources().All(cancelled); err == nil || calls != 2 {
		t.Fatalf("cancellation: %v calls=%d", err, calls)
	}
}
