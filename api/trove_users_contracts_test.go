package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/db/v1/databases"
	"github.com/JSYoo5B/go-openstacksdk/db/v1/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

func TestTroveUsersScopeResolvesInstanceOnceAndUsesListLookup(t *testing.T) {
	cloud := testcloud.New(t)
	api := users.New(cloud.Client("database", "/trove"))
	var parents, userLists atomic.Int32
	cloud.Mux.HandleFunc("GET /trove/instances", func(w http.ResponseWriter, r *http.Request) {
		parents.Add(1)
		if r.URL.RawQuery != "" {
			t.Errorf("instance name must be matched locally: %s", r.URL)
		}
		testcloud.JSON(w, 200, `{"instances":[{"id":"wrong","name":"db-server-copy"},{"id":"parent","name":"db-server"}]}`)
	})
	cloud.Mux.HandleFunc("GET /trove/instances/parent/users", func(w http.ResponseWriter, r *http.Request) {
		userLists.Add(1)
		if r.URL.RawQuery == "" {
			testcloud.JSON(w, 200, `{"users":[{"name":"app-copy"}],"users_links":[{"rel":"next","href":"`+cloud.Server.URL+`/trove/instances/parent/users?marker=next"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"users":[{"name":"app","databases":[{"name":"logs","character_set":"utf8mb4","collate":"utf8mb4_unicode_ci"}]}]}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no child GET endpoint exists: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	scope, err := api.InInstance(context.Background(), resource.Name("db-server"))
	if err != nil {
		t.Fatal(err)
	}
	for _, find := range []func() (*users.UserResource, error){
		func() (*users.UserResource, error) { return scope.Find(context.Background(), resource.Name("app")) },
		func() (*users.UserResource, error) { return scope.Get(context.Background(), "app") },
	} {
		value, err := find()
		if err != nil || value == nil || value.Name != "app" || len(value.Databases) != 1 || value.Databases[0].CharSet != "utf8mb4" || value.Databases[0].Collate != "utf8mb4_unicode_ci" {
			t.Fatalf("user=%v err=%v", value, err)
		}
	}
	if parents.Load() != 1 || userLists.Load() != 4 {
		t.Fatalf("parent lookup=%d user lists=%d", parents.Load(), userLists.Load())
	}
}

func TestTroveUserScopeCreateSingleAndBatch(t *testing.T) {
	cloud := testcloud.New(t)
	api := users.New(cloud.Client("database", "/trove"))
	var posts atomic.Int32
	cloud.Mux.HandleFunc("POST /trove/instances/parent/users", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		want := map[string]any{"users": []any{map[string]any{"name": "reader", "password": "fixture-password", "databases": []any{map[string]any{"name": "logs"}}, "host": "192.0.2.10"}}, "vendor_hint": false}
		if posts.Add(1) == 2 {
			want = map[string]any{"users": []any{map[string]any{"name": "reader", "password": "fixture-password"}, map[string]any{"name": "reader", "password": "another-fixture-password", "host": "192.0.2.10", "databases": []any{map[string]any{"name": "app"}, map[string]any{"name": "logs"}}}}}
		}
		if !reflect.DeepEqual(body, want) {
			t.Errorf("body=%#v want=%#v", body, want)
		}
		w.WriteHeader(http.StatusAccepted)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("explicit parent ID and database names do not trigger lookup: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	scope, err := api.InInstance(context.Background(), resource.ID("parent"))
	if err != nil {
		t.Fatal(err)
	}
	if err := scope.Create(context.Background(), users.CreateOpts{Name: "reader", Password: "fixture-password", Databases: databases.BatchCreateOpts{{Name: "logs"}}, Host: "192.0.2.10"}, users.WithCreateField("vendor_hint", false)); err != nil {
		t.Fatal(err)
	}
	if err := scope.CreateBatch(context.Background(), users.BatchCreateOpts{
		{Name: "reader", Password: "fixture-password"},
		{Name: "reader", Password: "another-fixture-password", Host: "192.0.2.10", Databases: databases.BatchCreateOpts{{Name: "app"}, {Name: "logs"}}},
	}); err != nil {
		t.Fatal(err)
	}
	if posts.Load() != 2 {
		t.Fatalf("posts=%d", posts.Load())
	}
}

func TestTroveUserScopeValidationAndUnsupportedPoliciesBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	api := users.New(cloud.Client("database", "/trove"))
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("invalid input must not request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	scope, err := api.InInstance(context.Background(), resource.ID("parent"))
	if err != nil {
		t.Fatal(err)
	}
	for _, batch := range []users.BatchCreateOpts{
		nil,
		{{Name: "", Password: "fixture-password"}},
		{{Name: "../reader", Password: "fixture-password"}},
		{{Name: "root", Password: "fixture-password"}},
		{{Name: "reader"}},
		{{Name: "reader", Password: "fixture-password"}, {Name: "reader", Password: "other", Host: "%"}},
		{{Name: "reader", Password: "fixture-password", Databases: databases.BatchCreateOpts{{Name: ""}}}},
		{{Name: "reader", Password: "fixture-password", Databases: databases.BatchCreateOpts{{Name: "../logs"}}}},
		{{Name: "reader", Password: "fixture-password", Databases: databases.BatchCreateOpts{{Name: strings.Repeat("a", 65)}}}},
		{{Name: "reader", Password: "fixture-password", Databases: databases.BatchCreateOpts{{Name: "logs"}, {Name: "logs"}}}},
	} {
		if err := scope.CreateBatch(context.Background(), batch); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("expected invalid-option error: %v", err)
		}
	}
	valid := users.CreateOpts{Name: "reader", Password: "fixture-password"}
	if err := scope.Create(context.Background(), valid, users.WithCreateOptions(users.BatchCreateOpts{valid, {Name: "writer", Password: "other"}})); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if err := scope.Create(context.Background(), valid, users.WithCreateField("users", []any{})); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := scope.All(context.Background(), resource.WithPageSize(1)); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := scope.All(context.Background(), resource.WithStatus("ACTIVE")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if _, err := scope.Wait(context.Background(), resource.ID("reader"), "ACTIVE"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if err := scope.Delete(context.Background(), resource.ID("../reader")); !errors.Is(err, resource.ErrInvalidOption) {
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
	if err := scope.Create(ctx, valid); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := scope.Get(ctx, "reader"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestTroveUserScopeMissingAndCrossPageAmbiguousNames(t *testing.T) {
	for _, ambiguous := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing", true: "duplicate across hosts and pages"}[ambiguous], func(t *testing.T) {
			cloud := testcloud.New(t)
			api := users.New(cloud.Client("database", "/trove"))
			cloud.Mux.HandleFunc("GET /trove/instances/parent/users", func(w http.ResponseWriter, r *http.Request) {
				if !ambiguous {
					testcloud.JSON(w, 200, `{"users":[]}`)
					return
				}
				if r.URL.RawQuery == "" {
					testcloud.JSON(w, 200, `{"users":[{"name":"reader","host":"%"}],"users_links":[{"rel":"next","href":"`+cloud.Server.URL+`/trove/instances/parent/users?marker=next"}]}`)
					return
				}
				testcloud.JSON(w, 200, `{"users":[{"name":"reader","host":"192.0.2.10"}]}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("missing or ambiguous named deletion must not DELETE: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected request", 500)
			})
			scope, err := api.InInstance(context.Background(), resource.ID("parent"))
			if err != nil {
				t.Fatal(err)
			}
			want := resource.ErrNotFound
			if ambiguous {
				want = resource.ErrAmbiguous
			}
			if _, err := scope.Find(context.Background(), resource.Name("reader")); !errors.Is(err, want) {
				t.Fatalf("find error=%v want=%v", err, want)
			}
			if err := scope.Delete(context.Background(), resource.Name("reader")); ambiguous && !errors.Is(err, resource.ErrAmbiguous) || !ambiguous && err != nil {
				t.Fatalf("named delete error=%v", err)
			}
			if !ambiguous {
				value, err := scope.Find(context.Background(), resource.Name("reader"), resource.WithIgnoreMissing())
				if err != nil || value != nil {
					t.Fatalf("ignored missing user=%v err=%v", value, err)
				}
				if err := scope.Delete(context.Background(), resource.Name("reader"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestTroveUserScopeDeleteLiteralNameMissingAndHTTPFailures(t *testing.T) {
	cloud := testcloud.New(t)
	api := users.New(cloud.Client("database", "/trove"))
	cloud.Mux.HandleFunc("GET /trove/instances/parent/users", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("exact names must not be sent as query: %s", r.URL)
		}
		testcloud.JSON(w, 200, `{"users":[{"name":"reader ?#@-copy"},{"name":"reader ?#@"}]}`)
	})
	var deletes atomic.Int32
	cloud.Mux.HandleFunc("DELETE /trove/instances/parent/users/{user}", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" || r.URL.EscapedPath() != "/trove/instances/parent/users/reader%20%3F%23@@%2525" {
			t.Errorf("literal username escaped incorrectly: %s", r.URL)
		}
		if deletes.Add(1) == 1 {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		if deletes.Load() == 4 {
			testcloud.JSON(w, 409, `{"message":"conflict"}`)
			return
		}
		testcloud.JSON(w, 404, `{"message":"not found"}`)
	})
	scope, err := api.InInstance(context.Background(), resource.ID("parent"))
	if err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(context.Background(), resource.Name("reader ?#@")); err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(context.Background(), resource.ID("reader ?#@")); err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(context.Background(), resource.ID("reader ?#@"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	var responseError gophercloud.ErrUnexpectedResponseCode
	if err := scope.Delete(context.Background(), resource.ID("reader ?#@")); !errors.As(err, &responseError) || responseError.Actual != 409 {
		t.Fatalf("HTTP cause lost: %v", err)
	}
}

func TestTroveUserScopeListFailureAndListBasedWaitDeleted(t *testing.T) {
	cloud := testcloud.New(t)
	api := users.New(cloud.Client("database", "/trove"))
	var lists atomic.Int32
	cloud.Mux.HandleFunc("GET /trove/instances/parent/users", func(w http.ResponseWriter, r *http.Request) {
		switch lists.Add(1) {
		case 1:
			testcloud.JSON(w, 403, `{"message":"denied"}`)
		case 2:
			testcloud.JSON(w, 200, `{"users":[{"name":"reader"}]}`)
		default:
			testcloud.JSON(w, 200, `{"users":[]}`)
		}
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("deletion wait must list, not GET a child: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	scope, err := api.InInstance(context.Background(), resource.ID("parent"))
	if err != nil {
		t.Fatal(err)
	}
	var responseError gophercloud.ErrUnexpectedResponseCode
	if _, err := scope.Get(context.Background(), "reader"); !errors.As(err, &responseError) || responseError.Actual != 403 {
		t.Fatalf("HTTP cause lost: %v", err)
	}
	if err := scope.WaitDeleted(context.Background(), resource.ID("reader"), resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second)); err != nil {
		t.Fatal(err)
	}
	if lists.Load() != 3 {
		t.Fatalf("lists=%d", lists.Load())
	}
}

func TestTroveUserHostScopeKeepsLiteralNameAndResolvedAccountAcrossOperations(t *testing.T) {
	for _, account := range []struct{ name, host string }{
		{"alice@192.0.2.10", "%"},
		{"alice@192.0.2.10", "db.example"},
		{"alice@192.0.2.10", "db.json"},
		{"alice%40192.0.2.10", "%"},
		{"alice%40192.0.2.10", "db%40example"},
	} {
		t.Run(account.name+" at "+account.host, func(t *testing.T) {
			cloud := testcloud.New(t)
			api := users.New(cloud.Client("database", "/trove"))
			name, host := account.name, account.host
			otherHost := "%"
			if host == otherHost {
				otherHost = "db.example"
			}
			var deleted atomic.Bool
			cloud.Mux.HandleFunc("GET /trove/instances/parent/users", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.RawQuery != "" {
					t.Errorf("host filtering must remain local: %s", r.URL)
				}
				accounts := []map[string]any{{"name": name, "host": otherHost}, {"name": "alice", "host": "192.0.2.10"}}
				if !deleted.Load() {
					accounts = append(accounts, map[string]any{"name": name, "host": host})
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(map[string]any{"users": accounts}); err != nil {
					t.Error(err)
				}
			})
			cloud.Mux.HandleFunc("DELETE /trove/instances/parent/users/{user}", func(w http.ResponseWriter, r *http.Request) {
				// PathValue already decodes once, as WSGI does. Trove then
				// unquotes the route value again and splits at the last '@'.
				if strings.Contains(r.PathValue("user"), ".") {
					t.Errorf("router must not interpret literal dots as file types: %q", r.PathValue("user"))
				}
				decoded, err := url.PathUnescape(r.PathValue("user"))
				if err != nil {
					t.Error(err)
				}
				separator := strings.LastIndexByte(decoded, '@')
				if separator < 0 || decoded[:separator] != name || decoded[separator+1:] != host {
					t.Errorf("delete selected another account: %q", decoded)
				}
				protected := strings.ReplaceAll(name+"@"+host, "%", "%25")
				expected := url.PathEscape(strings.ReplaceAll(protected, ".", "%2E"))
				if r.URL.EscapedPath() != "/trove/instances/parent/users/"+expected || r.URL.RawQuery != "" {
					t.Errorf("unsafe account URL: %s", r.URL)
				}
				deleted.Store(true)
				w.WriteHeader(http.StatusAccepted)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("child GET and parent ID lookup must not be requested: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected request", 500)
			})
			allHosts, err := api.InInstance(context.Background(), resource.ID("parent"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := allHosts.Find(context.Background(), resource.Name(name)); !errors.Is(err, resource.ErrAmbiguous) {
				t.Fatal(err)
			}
			scope, err := api.InInstance(context.Background(), resource.ID("parent"), users.WithHost(host))
			if err != nil {
				t.Fatal(err)
			}
			id, err := scope.ResolveID(context.Background(), resource.Name(name))
			if err != nil || id != name {
				t.Fatalf("resolved ID=%q err=%v", id, err)
			}
			value, err := scope.Find(context.Background(), resource.ID(id))
			if err != nil || value == nil || value.Name != name || value.Host != host {
				t.Fatalf("resolved account=%v err=%v", value, err)
			}
			if err := scope.Delete(context.Background(), resource.ID(id)); err != nil {
				t.Fatal(err)
			}
			if err := scope.WaitDeleted(context.Background(), resource.ID(id), resource.WithTimeout(time.Second)); err != nil {
				t.Fatal(err)
			}
			remaining, err := allHosts.All(context.Background())
			if err != nil || len(remaining) != 2 {
				t.Fatalf("other accounts must remain: %v err=%v", remaining, err)
			}
		})
	}
}

func TestTroveUserNamedDeletionAndWaitRetainNonDefaultHost(t *testing.T) {
	cloud := testcloud.New(t)
	api := users.New(cloud.Client("database", "/trove"))
	var phase atomic.Int32
	cloud.Mux.HandleFunc("GET /trove/instances/parent/users", func(w http.ResponseWriter, r *http.Request) {
		switch phase.Load() {
		case 0:
			testcloud.JSON(w, 200, `{"users":[{"name":"reader","host":"db.example"}]}`)
		case 1:
			phase.Add(1)
			testcloud.JSON(w, 200, `{"users":[{"name":"reader","host":"db.example"}]}`)
		case 2:
			phase.Add(1)
			testcloud.JSON(w, 200, `{"users":[{"name":"reader","host":"db.example"},{"name":"reader","host":"%"}]}`)
		default:
			testcloud.JSON(w, 200, `{"users":[{"name":"reader","host":"%"}]}`)
		}
	})
	cloud.Mux.HandleFunc("DELETE /trove/instances/parent/users/{user}", func(w http.ResponseWriter, r *http.Request) {
		account, err := url.PathUnescape(r.PathValue("user"))
		if err != nil || account != "reader@db.example" || r.URL.EscapedPath() != "/trove/instances/parent/users/reader@db%252Eexample" {
			t.Errorf("named delete lost host: %s", r.URL)
		}
		phase.Store(1)
		w.WriteHeader(http.StatusAccepted)
	})
	scope, err := api.InInstance(context.Background(), resource.ID("parent"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scope.ResolveID(context.Background(), resource.Name("reader")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("host must not be lost through a string ID: %v", err)
	}
	if err := scope.Delete(context.Background(), resource.Name("reader")); err != nil {
		t.Fatal(err)
	}
	if err := scope.WaitDeleted(context.Background(), resource.Name("reader"), resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second)); err != nil {
		t.Fatal(err)
	}
	if phase.Load() != 3 {
		t.Fatalf("wait did not follow the fixed account: phase=%d", phase.Load())
	}
}

func TestTroveUserHostScopeCreateDefaultsAndInputOwnership(t *testing.T) {
	cloud := testcloud.New(t)
	api := users.New(cloud.Client("database", "/trove"))
	cloud.Mux.HandleFunc("POST /trove/instances/parent/users", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Users []users.CreateOpts `json:"users"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body.Users) != 1 || body.Users[0].Host != "db.example" {
			t.Errorf("create must carry scope host: %#v", body)
		}
		w.WriteHeader(http.StatusAccepted)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("invalid host input must fail before HTTP: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	for _, host := range []string{"", "name@host", "../host", "host\n"} {
		if _, err := api.InInstance(context.Background(), resource.Name("parent"), users.WithHost(host)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := api.InInstance(context.Background(), resource.Name("parent"), nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	scope, err := api.InInstance(context.Background(), resource.ID("parent"), users.WithHost("db.example"))
	if err != nil {
		t.Fatal(err)
	}
	opts := users.BatchCreateOpts{{Name: "reader", Password: "fixture-password"}}
	if err := scope.CreateBatch(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	if opts[0].Host != "" {
		t.Fatalf("scope mutated caller's options: host=%q", opts[0].Host)
	}
	if err := scope.Create(context.Background(), users.CreateOpts{Name: "reader", Password: "fixture-password", Host: "%"}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestTroveUserNamedDeleteRejectsInvalidResponseIdentityBeforeHTTP(t *testing.T) {
	for _, host := range []string{"../host", "name@host", "host\n"} {
		t.Run(host, func(t *testing.T) {
			cloud := testcloud.New(t)
			api := users.New(cloud.Client("database", "/trove"))
			cloud.Mux.HandleFunc("GET /trove/instances/parent/users", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(map[string]any{"users": []map[string]string{{"name": "reader", "host": host}}}); err != nil {
					t.Error(err)
				}
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("invalid response identity must not trigger DELETE: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected request", 500)
			})
			scope, err := api.InInstance(context.Background(), resource.ID("parent"))
			if err != nil {
				t.Fatal(err)
			}
			if err := scope.Delete(context.Background(), resource.Name("reader")); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestTroveUserDefaultScopeExplicitIDKeepsDefaultHost(t *testing.T) {
	for _, hasDefault := range []bool{true, false} {
		name := "default and alternate hosts"
		if !hasDefault {
			name = "alternate host only"
		}
		t.Run(name, func(t *testing.T) {
			cloud := testcloud.New(t)
			api := users.New(cloud.Client("database", "/trove"))
			var lists atomic.Int32
			var deleted atomic.Bool
			cloud.Mux.HandleFunc("GET /trove/instances/parent/users", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if hasDefault && !deleted.Load() {
					testcloud.JSON(w, 200, `{"users":[{"name":"reader","host":"db.example"},{"name":"reader","host":"%"}]}`)
					return
				}
				testcloud.JSON(w, 200, `{"users":[{"name":"reader","host":"db.example"}]}`)
			})
			cloud.Mux.HandleFunc("DELETE /trove/instances/parent/users/{user}", func(w http.ResponseWriter, r *http.Request) {
				account, err := url.PathUnescape(r.PathValue("user"))
				if err != nil || account != "reader@%" {
					t.Errorf("ID targeted another host: account=%q err=%v", account, err)
				}
				deleted.Store(true)
				if !hasDefault {
					testcloud.JSON(w, 404, `{"message":"not found"}`)
					return
				}
				w.WriteHeader(http.StatusAccepted)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("must use list for lookup/wait: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected request", 500)
			})
			scope, err := api.InInstance(context.Background(), resource.ID("parent"))
			if err != nil {
				t.Fatal(err)
			}
			for _, get := range []func() (*users.UserResource, error){
				func() (*users.UserResource, error) { return scope.Get(context.Background(), "reader") },
				func() (*users.UserResource, error) { return scope.Find(context.Background(), resource.ID("reader")) },
			} {
				value, err := get()
				if hasDefault {
					if err != nil || value == nil || value.Host != "%" {
						t.Fatalf("ID lookup lost default host: value=%v err=%v", value, err)
					}
				} else if !errors.Is(err, resource.ErrNotFound) {
					t.Fatalf("ID must not select non-default account: value=%v err=%v", value, err)
				}
			}
			value, err := scope.Find(context.Background(), resource.Name("reader"))
			if hasDefault && !errors.Is(err, resource.ErrAmbiguous) || !hasDefault && (err != nil || value == nil || value.Host != "db.example") {
				t.Fatalf("name lookup must retain all-host policy: value=%v err=%v", value, err)
			}
			beforeResolve := lists.Load()
			id, err := scope.ResolveID(context.Background(), resource.ID("reader"))
			if err != nil || id != "reader" || lists.Load() != beforeResolve {
				t.Fatalf("explicit ID resolution must avoid HTTP: id=%q err=%v", id, err)
			}
			if err := scope.Delete(context.Background(), resource.ID(id)); err != nil {
				t.Fatal(err)
			}
			if err := scope.WaitDeleted(context.Background(), resource.ID(id), resource.WithTimeout(time.Second)); err != nil {
				t.Fatal(err)
			}
			remaining, err := scope.All(context.Background())
			if err != nil || len(remaining) != 1 || remaining[0].Host != "db.example" {
				t.Fatalf("ID deletion must preserve other host: remaining=%v err=%v", remaining, err)
			}
		})
	}
}
