package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/db/v1/databases"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

func TestTroveDatabasesScopeResolvesInstanceOnceAndUsesListLookup(t *testing.T) {
	cloud := testcloud.New(t)
	api := databases.New(cloud.Client("database", "/trove"))
	var parents, databaseLists atomic.Int32
	cloud.Mux.HandleFunc("GET /trove/instances", func(w http.ResponseWriter, r *http.Request) {
		parents.Add(1)
		if r.URL.RawQuery != "" {
			t.Errorf("Trove instance names must be matched locally: %s", r.URL)
		}
		testcloud.JSON(w, 200, `{"instances":[{"id":"wrong","name":"db-server-copy"},{"id":"parent-id","name":"db-server"}]}`)
	})
	cloud.Mux.HandleFunc("GET /trove/instances/parent-id/databases", func(w http.ResponseWriter, r *http.Request) {
		databaseLists.Add(1)
		if r.URL.RawQuery == "" {
			testcloud.JSON(w, 200, `{"databases":[{"name":"app-copy"}],"databases_links":[{"rel":"next","href":"`+cloud.Server.URL+`/trove/instances/parent-id/databases?marker=next"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"databases":[{"name":"app","character_set":"utf8mb4","collate":"utf8mb4_unicode_ci"}]}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no child GET endpoint exists: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	scope, err := api.InInstance(context.Background(), resource.Name("db-server"))
	if err != nil {
		t.Fatal(err)
	}
	for _, find := range []func() (*databases.Database, error){
		func() (*databases.Database, error) { return scope.Find(context.Background(), resource.Name("app")) },
		func() (*databases.Database, error) { return scope.Get(context.Background(), "app") },
	} {
		value, err := find()
		if err != nil || value == nil || value.Name != "app" || value.CharSet != "utf8mb4" || value.Collate != "utf8mb4_unicode_ci" {
			t.Fatalf("database=%v err=%v", value, err)
		}
	}
	if parents.Load() != 1 || databaseLists.Load() != 4 {
		t.Fatalf("parents=%d database lists=%d", parents.Load(), databaseLists.Load())
	}
}

func TestTroveDatabaseScopeIDSkipsParentLookupAndUsesLiteralNames(t *testing.T) {
	cloud := testcloud.New(t)
	api := databases.New(cloud.Client("database", "/trove"))
	cloud.Mux.HandleFunc("GET /trove/instances/parent/databases", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("database names must not become server filters: %s", r.URL)
		}
		testcloud.JSON(w, 200, `{"databases":[{"name":"app ?#@-copy"},{"name":"app ?#@"}]}`)
	})
	cloud.Mux.HandleFunc("DELETE /trove/instances/parent/databases/{database}", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" || r.URL.EscapedPath() != "/trove/instances/parent/databases/app%20%3F%23@" {
			t.Errorf("name escaping lost: %s", r.URL)
		}
		w.WriteHeader(http.StatusAccepted)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("explicit parent IDs must bypass lookup: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	scope, err := api.InInstance(context.Background(), resource.ID("parent"))
	if err != nil {
		t.Fatal(err)
	}
	value, err := scope.Find(context.Background(), resource.Name("app ?#@"))
	if err != nil || value == nil || value.Name != "app ?#@" {
		t.Fatalf("value=%v err=%v", value, err)
	}
	if err := scope.Delete(context.Background(), resource.ID("app ?#@")); err != nil {
		t.Fatal(err)
	}
}

func TestTroveDatabaseScopeLookupMissingAmbiguousAndHTTPFailures(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		want       error
	}{
		{name: "missing", body: `{"databases":[]}`, status: 200, want: resource.ErrNotFound},
		{name: "ambiguous", body: `{"databases":[{"name":"app"},{"name":"app"}]}`, status: 200, want: resource.ErrAmbiguous},
		{name: "forbidden", body: `{"message":"denied"}`, status: 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			api := databases.New(cloud.Client("database", "/trove"))
			cloud.Mux.HandleFunc("GET /trove/instances/parent/databases", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, tc.status, tc.body) })
			scope, err := api.InInstance(context.Background(), resource.ID("parent"))
			if err != nil {
				t.Fatal(err)
			}
			_, err = scope.Find(context.Background(), resource.Name("app"))
			if tc.want != nil {
				if !errors.Is(err, tc.want) {
					t.Fatalf("err=%v want=%v", err, tc.want)
				}
				return
			}
			var responseError gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &responseError) || responseError.Actual != 403 {
				t.Fatalf("HTTP cause lost: %v", err)
			}
		})
	}
}

func TestTroveDatabaseScopeCreateSingleAndBatch(t *testing.T) {
	cloud := testcloud.New(t)
	api := databases.New(cloud.Client("database", "/trove"))
	var requests atomic.Int32
	cloud.Mux.HandleFunc("POST /trove/instances/parent/databases", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if requests.Add(1) == 1 {
			want := map[string]any{"databases": []any{map[string]any{"name": "app", "character_set": "utf8mb4", "collate": "utf8mb4_unicode_ci"}}, "vendor_hint": false}
			if !reflect.DeepEqual(body, want) {
				t.Errorf("single=%#v want=%#v", body, want)
			}
		} else {
			want := map[string]any{"databases": []any{map[string]any{"name": "app"}, map[string]any{"name": "logs"}}}
			if !reflect.DeepEqual(body, want) {
				t.Errorf("batch=%#v want=%#v", body, want)
			}
		}
		w.WriteHeader(http.StatusAccepted)
	})
	scope, err := api.InInstance(context.Background(), resource.ID("parent"))
	if err != nil {
		t.Fatal(err)
	}
	if err := scope.Create(context.Background(), databases.CreateOpts{Name: "app", CharSet: "utf8mb4", Collate: "utf8mb4_unicode_ci"}, databases.WithCreateField("vendor_hint", false)); err != nil {
		t.Fatal(err)
	}
	if err := scope.CreateBatch(context.Background(), databases.BatchCreateOpts{{Name: "app"}, {Name: "logs"}}); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 2 {
		t.Fatalf("requests=%d", requests.Load())
	}
}

func TestTroveDatabaseScopeValidationAndUnsupportedPoliciesBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	api := databases.New(cloud.Client("database", "/trove"))
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("invalid input must not request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	scope, err := api.InInstance(context.Background(), resource.ID("parent"))
	if err != nil {
		t.Fatal(err)
	}
	for _, batch := range []databases.BatchCreateOpts{nil, {{Name: ""}}, {{Name: "../path"}}, {{Name: strings.Repeat("a", 65)}}, {{Name: "duplicate"}, {Name: "duplicate"}}} {
		if err := scope.CreateBatch(context.Background(), batch); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("batch=%v err=%v", batch, err)
		}
	}
	if err := scope.Create(context.Background(), databases.CreateOpts{Name: "app"}, databases.WithCreateOptions(databases.BatchCreateOpts{{Name: "one"}, {Name: "two"}})); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if err := scope.Create(context.Background(), databases.CreateOpts{Name: "app"}, databases.WithCreateField("databases", []any{})); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := scope.All(context.Background(), resource.WithStatus("ACTIVE")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := scope.All(context.Background(), resource.WithPageSize(1)); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := scope.Wait(context.Background(), resource.ID("app"), "ACTIVE"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := api.InInstance(context.Background(), resource.ID("../parent")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.InInstance(ctx, resource.Name("parent")); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := scope.Create(ctx, databases.CreateOpts{Name: "app"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestTroveDatabaseScopeDeleteMissingPolicyAndListBasedWaitDeleted(t *testing.T) {
	cloud := testcloud.New(t)
	api := databases.New(cloud.Client("database", "/trove"))
	var lists atomic.Int32
	cloud.Mux.HandleFunc("DELETE /trove/instances/parent/databases/app", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{"message":"not found"}`) })
	cloud.Mux.HandleFunc("GET /trove/instances/parent/databases", func(w http.ResponseWriter, r *http.Request) {
		if lists.Add(1) == 1 {
			testcloud.JSON(w, 200, `{"databases":[{"name":"app"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"databases":[]}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("wait deletion must list, not GET a child: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	scope, err := api.InInstance(context.Background(), resource.ID("parent"))
	if err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(context.Background(), resource.ID("app")); err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(context.Background(), resource.ID("app"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if err := scope.WaitDeleted(context.Background(), resource.ID("app"), resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second)); err != nil {
		t.Fatal(err)
	}
	if lists.Load() != 2 {
		t.Fatalf("lists=%d", lists.Load())
	}
}
