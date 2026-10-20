package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/db/v1/databases"
	"github.com/JSYoo5B/go-openstacksdk/db/v1/datastores"
	"github.com/JSYoo5B/go-openstacksdk/db/v1/flavors"
	"github.com/JSYoo5B/go-openstacksdk/db/v1/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func troveMiscDatabases(cloud *testcloud.Cloud) *databases.API {
	return databases.New(cloud.Client("database", "/trove"))
}

func troveMiscUsers(cloud *testcloud.Cloud) *users.API {
	return users.New(cloud.Client("database", "/trove"))
}

func TestTroveMiscDatabaseCreateBodyOmitsEmptyOptionalsAndKeepsExtensions(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("POST /trove/instances/inst-1/databases", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		got := troveRawBody(t, r)
		if string(got["databases"]) != `[{"character_set":"utf8mb4","collate":"utf8mb4_unicode_ci","name":"app"},{"name":"logs"}]` ||
			string(got["vendor"]) != `{"enabled":false}` || len(got) != 2 {
			t.Errorf("body=%v", got)
		}
		w.WriteHeader(202)
	})
	err := troveMiscDatabases(cloud).Create(context.Background(), "inst-1", databases.BatchCreateOpts{
		{Name: "app", CharSet: "utf8mb4", Collate: "utf8mb4_unicode_ci"},
		{Name: "logs"},
	}, databases.WithCreateField("vendor", map[string]bool{"enabled": false}))
	if err != nil || calls.Load() != 1 {
		t.Fatalf("error=%v calls=%d", err, calls.Load())
	}
}

func TestTroveMiscDatabaseCreateAcceptsOnly201Or202(t *testing.T) {
	troveStatusMatrix(t, "POST", "/trove/instances/inst-1/databases", `{}`, []int{201, 202}, func(cloud *testcloud.Cloud) error {
		return troveMiscDatabases(cloud).Create(context.Background(), "inst-1", databases.BatchCreateOpts{{Name: "app"}})
	})
}

func TestTroveMiscDatabaseCreateRejectsInvalidInputsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := troveMiscDatabases(cloud)
	ctx := context.Background()
	var invalid gophercloud.ErrInvalidInput
	var missing gophercloud.ErrMissingInput
	long := make([]byte, 65)
	for i := range long {
		long[i] = 'a'
	}
	if err := api.Create(ctx, "inst-1", databases.BatchCreateOpts{{Name: string(long)}}); !errors.As(err, &invalid) {
		t.Fatalf("long name: %v", err)
	}
	if err := api.Create(ctx, "inst-1", databases.BatchCreateOpts{{Name: "ok"}, {}}); !errors.As(err, &missing) {
		t.Fatalf("missing name: %v", err)
	}
	if err := api.Create(ctx, "inst-1", databases.BatchCreateOpts{{Name: "ok"}}, databases.WithCreateField("databases", 1)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected databases field: %v", err)
	}
	if err := api.Create(ctx, "inst-1", databases.BatchCreateOpts{{Name: "ok"}}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option: %v", err)
	}
}

func TestTroveMiscDatabaseDeleteStatusesAndNative404(t *testing.T) {
	troveStatusMatrix(t, "DELETE", "/trove/instances/inst-1/databases/app", `{}`, []int{202, 204}, func(cloud *testcloud.Cloud) error {
		return troveMiscDatabases(cloud).Delete(context.Background(), "inst-1", "app")
	})
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("DELETE /trove/instances/inst-1/databases/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	if err := troveMiscDatabases(cloud).Delete(context.Background(), "inst-1", "missing"); !troveIsNative404(err) || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native delete keeps the 404 response error: %v", err)
	}
}

func TestTroveMiscDatabaseListFollowsLinksAndNativeModelOmitsCharacterSet(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /trove/instances/inst-1/databases", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "app" {
			testcloud.JSON(w, 200, `{"databases":[{"name":"logs"}],"databases_links":[]}`)
			return
		}
		next := cloud.Server.URL + "/trove/instances/inst-1/databases?marker=app"
		testcloud.JSON(w, 200, `{"databases":[{"name":"app","character_set":"utf8mb4","collate":"utf8mb4_unicode_ci"}],"databases_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	var got []*databases.Database
	for value, err := range troveMiscDatabases(cloud).List(context.Background(), "inst-1") {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, value)
	}
	if len(got) != 2 || got[0].Name != "app" || got[1].Name != "logs" || len(requests) != 2 || requests[1].Get("marker") != "app" {
		t.Fatalf("got=%v requests=%v", got, requests)
	}
	// Gophercloud's Database has no character_set tag; the scoped API owns the
	// conversion, while the native API returns the model unchanged.
	if got[0].Collate != "utf8mb4_unicode_ci" || got[0].CharSet != "" {
		t.Fatalf("native model decoding changed: %+v", got[0])
	}
}

func TestTroveMiscDatabaseListNoContentAndHTTPFailure(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /trove/instances/inst-1/databases", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	for _, err := range troveMiscDatabases(cloud).List(context.Background(), "inst-1") {
		t.Fatalf("204 list yielded %v", err)
	}
	cloud = testcloud.New(t)
	cloud.Mux.HandleFunc("GET /trove/instances/missing/databases", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	for _, err := range troveMiscDatabases(cloud).List(context.Background(), "missing") {
		if !troveIsNative404(err) {
			t.Fatalf("list 404: %v", err)
		}
	}
}

func TestTroveMiscUserCreateBodyKeepsHostAndOmitsEmptyOptionals(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /trove/instances/inst-1/users", func(w http.ResponseWriter, r *http.Request) {
		got := troveRawBody(t, r)
		want := `[{"databases":[{"name":"app"}],"host":"192.0.2.10","name":"reader","password":"fixture-password"},{"name":"plain","password":"fixture-password"}]`
		if string(got["users"]) != want || string(got["vendor"]) != `false` || len(got) != 2 {
			t.Errorf("body=%v", got)
		}
		w.WriteHeader(202)
	})
	err := troveMiscUsers(cloud).Create(context.Background(), "inst-1", users.BatchCreateOpts{
		{Name: "reader", Password: "fixture-password", Host: "192.0.2.10", Databases: databases.BatchCreateOpts{{Name: "app"}}},
		{Name: "plain", Password: "fixture-password"},
	}, users.WithCreateField("vendor", false))
	if err != nil {
		t.Fatal(err)
	}
	troveStatusMatrix(t, "POST", "/trove/instances/inst-1/users", `{}`, []int{201, 202}, func(cloud *testcloud.Cloud) error {
		return troveMiscUsers(cloud).Create(context.Background(), "inst-1", users.BatchCreateOpts{{Name: "u", Password: "p"}})
	})
}

func TestTroveMiscUserCreateRejectsInvalidInputsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := troveMiscUsers(cloud)
	ctx := context.Background()
	var invalid gophercloud.ErrInvalidInput
	var missing gophercloud.ErrMissingInput
	if err := api.Create(ctx, "inst-1", users.BatchCreateOpts{{Name: "root", Password: "p"}}); !errors.As(err, &invalid) {
		t.Fatalf("reserved root: %v", err)
	}
	if err := api.Create(ctx, "inst-1", users.BatchCreateOpts{{Name: "u"}}); !errors.As(err, &missing) {
		t.Fatalf("missing password: %v", err)
	}
	if err := api.Create(ctx, "inst-1", users.BatchCreateOpts{{Password: "p"}}); !errors.As(err, &missing) {
		t.Fatalf("missing name: %v", err)
	}
	if err := api.Create(ctx, "inst-1", users.BatchCreateOpts{{Name: "u", Password: "p"}}, users.WithCreateField("users", 1)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("protected users field: %v", err)
	}
	if err := api.Create(ctx, "inst-1", users.BatchCreateOpts{{Name: "u", Password: "p"}}, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil option: %v", err)
	}
}

func TestTroveMiscUserDeleteUsesLiteralAccountPathAndStatuses(t *testing.T) {
	cloud := testcloud.New(t)
	var paths []string
	cloud.Mux.HandleFunc("DELETE /trove/instances/inst-1/users/", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.EscapedPath())
		if r.URL.Path == "/trove/instances/inst-1/users/missing" {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		w.WriteHeader(204)
	})
	api := troveMiscUsers(cloud)
	ctx := context.Background()
	if err := api.Delete(ctx, "inst-1", "reader"); err != nil {
		t.Fatal(err)
	}
	// The native API does not interpret a host: a host-qualified account is the
	// caller's literal path segment.
	if err := api.Delete(ctx, "inst-1", "reader@192.0.2.10"); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "inst-1", "missing"); !troveIsNative404(err) || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native delete keeps the 404 response error: %v", err)
	}
	if want := []string{"/trove/instances/inst-1/users/reader", "/trove/instances/inst-1/users/reader@192.0.2.10", "/trove/instances/inst-1/users/missing"}; !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths=%v", paths)
	}
	troveStatusMatrix(t, "DELETE", "/trove/instances/inst-1/users/reader", `{}`, []int{202, 204}, func(cloud *testcloud.Cloud) error {
		return troveMiscUsers(cloud).Delete(context.Background(), "inst-1", "reader")
	})
}

func TestTroveMiscUserListNativeModelDropsHostAndScopeFiltersByHost(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /trove/instances/inst-1/users", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "reader" {
			testcloud.JSON(w, 200, `{"users":[{"name":"reader","host":"192.0.2.10"},{"name":"other","host":"192.0.2.10"}],"users_links":[]}`)
			return
		}
		next := cloud.Server.URL + "/trove/instances/inst-1/users?marker=reader"
		testcloud.JSON(w, 200, `{"users":[{"name":"reader","host":"%","databases":[{"name":"app","character_set":"utf8mb4"}]},{"name":"nohost"}],"users_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	ctx := context.Background()
	var native []*users.User
	for value, err := range troveMiscUsers(cloud).List(ctx, "inst-1") {
		if err != nil {
			t.Fatal(err)
		}
		native = append(native, value)
	}
	if len(native) != 4 || native[0].Name != "reader" || native[2].Name != "reader" || len(native[0].Databases) != 1 || native[0].Databases[0].Name != "app" || len(requests) != 2 {
		t.Fatalf("native=%v requests=%v", native, requests)
	}
	// The scoped API, not the native list, owns host selection.
	hostScope, err := troveMiscUsers(cloud).InInstance(ctx, resource.ID("inst-1"), users.WithHost("192.0.2.10"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for value, err := range hostScope.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, value.Name+"@"+value.Host)
	}
	if want := []string{"reader@192.0.2.10", "other@192.0.2.10"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("host-filtered names=%v", names)
	}
	defaultHost, err := troveMiscUsers(cloud).InInstance(ctx, resource.ID("inst-1"), users.WithHost("%"))
	if err != nil {
		t.Fatal(err)
	}
	names = nil
	for value, err := range defaultHost.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, value.Name+"@"+value.Host)
	}
	if want := []string{"reader@%", "nohost@"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("a missing host counts as '%%': %v", names)
	}
	allHosts, err := troveMiscUsers(cloud).InInstance(ctx, resource.ID("inst-1"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := allHosts.Find(ctx, resource.Name("reader")); !errors.Is(err, resource.ErrAmbiguous) {
		t.Fatalf("same name on two hosts without WithHost: %v", err)
	}
	if value, err := hostScope.Find(ctx, resource.Name("reader")); err != nil || value == nil || value.Host != "192.0.2.10" {
		t.Fatalf("host scope find: %+v %v", value, err)
	}
}

func TestTroveMiscDatastoreGetListVersionsAndModels(t *testing.T) {
	cloud := testcloud.New(t)
	var lists atomic.Int32
	cloud.Mux.HandleFunc("GET /trove/datastores", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if r.URL.RawQuery != "" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		next := cloud.Server.URL + "/trove/datastores?marker=ds-1"
		testcloud.JSON(w, 200, `{"datastores":[{"id":"ds-1","name":"mysql","default_version":"ver-1","versions":[{"id":"ver-1","name":"8.0"}]},{"id":"ds-2","name":"postgresql","default_version":"","versions":[]}],"datastores_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	cloud.Mux.HandleFunc("GET /trove/datastores/ds-1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"datastore":{"id":"ds-1","name":"mysql","default_version":"ver-1","versions":[{"id":"ver-1","name":"8.0"}]}}`)
	})
	cloud.Mux.HandleFunc("GET /trove/datastores/ds-1/versions", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"versions":[{"id":"ver-1","name":"8.0"},{"id":"ver-2","name":"8.4"}]}`)
	})
	cloud.Mux.HandleFunc("GET /trove/datastores/ds-1/versions/ver-1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"version":{"id":"ver-1","name":"8.0","links":[{"rel":"self","href":"http://example.invalid/v"}]}}`)
	})
	cloud.Mux.HandleFunc("GET /trove/datastores/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /trove/datastores/ds-1/versions/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api := datastores.New(cloud.Client("database", "/trove"))
	ctx := context.Background()
	var names []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, value.Name)
	}
	if !reflect.DeepEqual(names, []string{"mysql", "postgresql"}) || lists.Load() != 1 {
		t.Fatalf("names=%v lists=%d: pinned native list is a single page", names, lists.Load())
	}
	ds, err := api.Get(ctx, "ds-1")
	if err != nil || ds.Name != "mysql" || ds.DefaultVersion != "ver-1" || len(ds.Versions) != 1 || ds.Versions[0].ID != "ver-1" {
		t.Fatalf("datastore=%+v error=%v", ds, err)
	}
	var versions []string
	for value, err := range api.ListVersions(ctx, "ds-1") {
		if err != nil {
			t.Fatal(err)
		}
		versions = append(versions, value.ID+":"+value.Name)
	}
	if !reflect.DeepEqual(versions, []string{"ver-1:8.0", "ver-2:8.4"}) {
		t.Fatalf("versions=%v", versions)
	}
	version, err := api.GetVersion(ctx, "ds-1", "ver-1")
	if err != nil || version.ID != "ver-1" || version.Name != "8.0" || len(version.Links) != 1 {
		t.Fatalf("version=%+v error=%v", version, err)
	}
	if _, err := api.Get(ctx, "missing"); !troveIsNative404(err) || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
	if _, err := api.GetVersion(ctx, "ds-1", "missing"); !troveIsNative404(err) {
		t.Fatalf("native version get 404: %v", err)
	}
	if value, err := api.Find(ctx, resource.Name("postgresql")); err != nil || value == nil || value.ID != "ds-2" {
		t.Fatalf("name find: %+v %v", value, err)
	}
	if _, err := api.Find(ctx, resource.Name("absent")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing name: %v", err)
	}
	if _, err := api.All(ctx, resource.WithQuery("vendor", "x")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("extension query: %v", err)
	}
}

func TestTroveMiscDatastoreStatusesAndNoContent(t *testing.T) {
	troveStatusMatrix(t, "GET", "/trove/datastores/ds-1", `{"datastore":{"id":"ds-1"}}`, []int{200}, func(cloud *testcloud.Cloud) error {
		_, err := datastores.New(cloud.Client("database", "/trove")).Get(context.Background(), "ds-1")
		return err
	})
	troveStatusMatrix(t, "GET", "/trove/datastores/ds-1/versions/ver-1", `{"version":{"id":"ver-1"}}`, []int{200}, func(cloud *testcloud.Cloud) error {
		_, err := datastores.New(cloud.Client("database", "/trove")).GetVersion(context.Background(), "ds-1", "ver-1")
		return err
	})
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /trove/datastores", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	cloud.Mux.HandleFunc("GET /trove/datastores/ds-1/versions", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	api := datastores.New(cloud.Client("database", "/trove"))
	for _, err := range api.List(context.Background()) {
		t.Fatalf("204 datastore list yielded %v", err)
	}
	for _, err := range api.ListVersions(context.Background(), "ds-1") {
		t.Fatalf("204 version list yielded %v", err)
	}
}

func TestTroveMiscFlavorGetListAndLookup(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /trove/flavors", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "1" {
			testcloud.JSON(w, 200, `{"flavors":[{"id":2,"name":"m1.large","ram":4096,"str_id":"2"}],"flavors_links":[]}`)
			return
		}
		next := cloud.Server.URL + "/trove/flavors?marker=1"
		testcloud.JSON(w, 200, `{"flavors":[{"id":1,"name":"m1.tiny","ram":0,"str_id":"1"}],"flavors_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	cloud.Mux.HandleFunc("GET /trove/flavors/1", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"flavor":{"id":1,"name":"m1.tiny","ram":0,"str_id":"1","links":[{"rel":"self","href":"http://example.invalid/f"}]}}`)
	})
	cloud.Mux.HandleFunc("GET /trove/flavors/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api := flavors.New(cloud.Client("database", "/trove"))
	ctx := context.Background()
	var names []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, value.Name)
	}
	if !reflect.DeepEqual(names, []string{"m1.tiny", "m1.large"}) || len(requests) != 2 || len(requests[0]) != 0 || requests[1].Get("marker") != "1" {
		t.Fatalf("names=%v requests=%v", names, requests)
	}
	flavor, err := api.Get(ctx, "1")
	if err != nil || flavor.ID != 1 || flavor.RAM != 0 || flavor.StrID != "1" || len(flavor.Links) != 1 {
		t.Fatalf("flavor=%+v error=%v", flavor, err)
	}
	if _, err := api.Get(ctx, "missing"); !troveIsNative404(err) || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native get keeps the 404 response error: %v", err)
	}
	if value, err := api.Find(ctx, resource.ID("1")); err != nil || value == nil || value.Name != "m1.tiny" {
		t.Fatalf("id find: %+v %v", value, err)
	}
	if value, err := api.Find(ctx, resource.Name("m1.large")); err != nil || value == nil || value.ID != 2 {
		t.Fatalf("name find: %+v %v", value, err)
	}
	if _, err := api.Find(ctx, resource.Name("absent")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing name: %v", err)
	}
	if _, err := api.All(ctx, resource.WithQuery("vendor", "x")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("extension query: %v", err)
	}
	troveStatusMatrix(t, "GET", "/trove/flavors/1", `{"flavor":{"id":1}}`, []int{200}, func(cloud *testcloud.Cloud) error {
		_, err := flavors.New(cloud.Client("database", "/trove")).Get(context.Background(), "1")
		return err
	})
}

func TestTroveMiscCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	client := cloud.Client("database", "/trove")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	checks := map[string]error{}
	checks["databases.Create"] = databases.New(client).Create(canceled, "i", databases.BatchCreateOpts{{Name: "a"}})
	checks["databases.Delete"] = databases.New(client).Delete(canceled, "i", "a")
	checks["users.Create"] = users.New(client).Create(canceled, "i", users.BatchCreateOpts{{Name: "u", Password: "p"}})
	checks["users.Delete"] = users.New(client).Delete(canceled, "i", "u")
	_, checks["datastores.Get"] = datastores.New(client).Get(canceled, "d")
	_, checks["datastores.GetVersion"] = datastores.New(client).GetVersion(canceled, "d", "v")
	_, checks["flavors.Get"] = flavors.New(client).Get(canceled, "1")
	for _, err := range databases.New(client).List(canceled, "i") {
		checks["databases.List"] = err
	}
	for _, err := range users.New(client).List(canceled, "i") {
		checks["users.List"] = err
	}
	for _, err := range datastores.New(client).List(canceled) {
		checks["datastores.List"] = err
	}
	for _, err := range datastores.New(client).ListVersions(canceled, "d") {
		checks["datastores.ListVersions"] = err
	}
	for _, err := range flavors.New(client).List(canceled) {
		checks["flavors.List"] = err
	}
	if len(checks) != 12 {
		t.Fatalf("checked %d operations", len(checks))
	}
	for label, err := range checks {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("pre-canceled %s: %v", label, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if err := databases.New(client).Create(deadline, "i", databases.BatchCreateOpts{{Name: "a"}}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("deadline create must not be replayed: calls=%d", calls.Load())
	}
}
