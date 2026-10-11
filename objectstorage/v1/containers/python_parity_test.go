package containers_test

import (
	"context"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonContainerCall struct {
	method, path, query, body string
	header                    http.Header
}

// pythonContainerAPI serves the Swift account root /swift/v1/AUTH_p/ and records each request.
func pythonContainerAPI(t *testing.T, reply func(call pythonContainerCall, w http.ResponseWriter)) (*containers.API, func() []pythonContainerCall) {
	t.Helper()
	cloud := testcloud.New(t)
	var mu sync.Mutex
	var calls []pythonContainerCall
	cloud.Mux.HandleFunc("/swift/v1/AUTH_p/", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		call := pythonContainerCall{r.Method, r.URL.EscapedPath(), r.URL.RawQuery, string(body), r.Header.Clone()}
		mu.Lock()
		calls = append(calls, call)
		mu.Unlock()
		reply(call, w)
	})
	return containers.New(cloud.Client("object-store", "/swift/v1/AUTH_p/")), func() []pythonContainerCall {
		mu.Lock()
		defer mu.Unlock()
		return append([]pythonContainerCall(nil), calls...)
	}
}

func TestPythonContainerCloudListContainersPrefixAndCountPaging(t *testing.T) {
	for _, tc := range []struct {
		label    string
		prefix   string
		count    string
		queries  []string
		expected []string
	}{
		{"no prefix single page", "", "2", []string{""}, []string{"alpha", "beta"}},
		{"prefix follows count marker", "al", "5", []string{"prefix=al", "marker=beta&prefix=al"}, []string{"alpha", "beta"}},
	} {
		t.Run(tc.label, func(t *testing.T) {
			api, calls := pythonContainerAPI(t, func(call pythonContainerCall, w http.ResponseWriter) {
				w.Header().Set("X-Account-Container-Count", tc.count)
				if call.query == "marker=beta&prefix=al" {
					testcloud.JSON(w, 200, `[]`)
					return
				}
				testcloud.JSON(w, 200, `[{"name":"alpha","count":1,"bytes":2},{"name":"beta","count":0,"bytes":0}]`)
			})
			var options []containers.RecordListOption
			if tc.prefix != "" {
				options = append(options, containers.WithRecordListPrefix(tc.prefix))
			}
			var names []string
			for record, err := range api.ListRecords(context.Background(), options...) {
				if err != nil {
					t.Fatal(err)
				}
				names = append(names, string(record.Resource.Body["name"]))
			}
			got := calls()
			if len(got) != len(tc.queries) || len(names) != 2 || names[0] != `"alpha"` || names[1] != `"beta"` {
				t.Fatal(names, got)
			}
			for i, call := range got {
				if call.method != "GET" || call.path != "/swift/v1/AUTH_p/" || call.query != tc.queries[i] || call.header.Get("Accept") != "application/json" {
					t.Fatal(i, call)
				}
			}
		})
	}
}

func TestPythonContainerCloudGetContainerHeadAndMissingNone(t *testing.T) {
	api, calls := pythonContainerAPI(t, func(call pythonContainerCall, w http.ResponseWriter) {
		if call.path == "/swift/v1/AUTH_p/missing" {
			w.WriteHeader(404)
			return
		}
		if call.path == "/swift/v1/AUTH_p/denied" {
			w.WriteHeader(403)
			return
		}
		w.Header().Set("X-Container-Object-Count", "3")
		w.Header().Set("X-Container-Bytes-Used", "9")
		w.Header().Set("X-Container-Read", ".r:*,.rlistings")
		w.Header().Set("X-Container-Meta-Owner", "team")
		w.WriteHeader(204)
	})
	found, err := api.Find(context.Background(), resource.ID("photos"), resource.WithIgnoreMissing())
	if err != nil || found == nil || found.Name != "photos" || found.Count != 3 || found.Bytes != 9 || found.Metadata["Owner"] != "team" || found.Header.Get("X-Container-Read") != ".r:*,.rlistings" {
		t.Fatalf("%+v %v", found, err)
	}
	missing, err := api.Find(context.Background(), resource.ID("missing"), resource.WithIgnoreMissing())
	if err != nil || missing != nil {
		t.Fatal(missing, err)
	}
	if _, err := api.Find(context.Background(), resource.ID("denied"), resource.WithIgnoreMissing()); !gophercloud.ResponseCodeIs(err, 403) {
		t.Fatal(err)
	}
	for _, call := range calls() {
		if call.method != "HEAD" || call.query != "" || call.body != "" {
			t.Fatal(call)
		}
	}
	if len(calls()) != 3 {
		t.Fatal(calls())
	}
}

func TestPythonContainerCloudCreateContainerSequence(t *testing.T) {
	for _, tc := range []struct {
		label   string
		exists  bool
		attrs   map[string]any
		methods []string
		acl     string
	}{
		{"existing returns first HEAD", true, nil, []string{"HEAD"}, ""},
		{"missing private creates then refreshes", false, nil, []string{"HEAD", "PUT", "HEAD"}, ""},
		{"missing public sends read ACL", false, map[string]any{"read_ACL": ".r:*,.rlistings"}, []string{"HEAD", "PUT", "HEAD"}, ".r:*,.rlistings"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			created := tc.exists
			api, calls := pythonContainerAPI(t, func(call pythonContainerCall, w http.ResponseWriter) {
				switch call.method {
				case "PUT":
					created = true
					w.WriteHeader(201)
				case "HEAD":
					if !created {
						w.WriteHeader(404)
						return
					}
					w.WriteHeader(204)
				}
			})
			ctx := context.Background()
			found, err := api.Find(ctx, resource.ID("box"), resource.WithIgnoreMissing())
			if err == nil && found == nil {
				if _, err = api.CreateRecord(ctx, "box", tc.attrs); err == nil {
					found, err = api.Find(ctx, resource.ID("box"), resource.WithIgnoreMissing())
				}
			}
			if err != nil || found == nil || found.Name != "box" {
				t.Fatal(found, err)
			}
			got := calls()
			if len(got) != len(tc.methods) {
				t.Fatal(got)
			}
			for i, call := range got {
				if call.method != tc.methods[i] || call.path != "/swift/v1/AUTH_p/box" || call.query != "" || call.body != "" {
					t.Fatal(i, call)
				}
				if call.method == "PUT" && call.header.Get("X-Container-Read") != tc.acl {
					t.Fatal(call.header)
				}
			}
		})
	}
}

func TestPythonContainerCloudDeleteContainerOutcomes(t *testing.T) {
	for _, tc := range []struct {
		label    string
		status   int
		deleted  bool
		conflict bool
	}{
		{"deleted", 204, true, false},
		{"missing is false", 404, false, false},
		{"nonempty conflict is an error", 409, false, true},
	} {
		t.Run(tc.label, func(t *testing.T) {
			api, calls := pythonContainerAPI(t, func(_ pythonContainerCall, w http.ResponseWriter) { w.WriteHeader(tc.status) })
			result, err := api.DeleteRecord(context.Background(), "box")
			if tc.conflict {
				if !gophercloud.ResponseCodeIs(err, 409) {
					t.Fatal(result, err)
				}
			} else if err != nil || result == nil || result.IgnoredMissing == tc.deleted || result.StatusCode != tc.status {
				t.Fatal(result, err)
			}
			if got := calls(); len(got) != 1 || got[0].method != "DELETE" || got[0].path != "/swift/v1/AUTH_p/box" || got[0].query != "" {
				t.Fatal(got)
			}
		})
	}
}

func TestPythonContainerCloudUpdateContainerAndAccessPost(t *testing.T) {
	for _, tc := range []struct {
		label  string
		values map[string]any
		want   map[string]string
	}{
		{"update headers", map[string]any{"owner": "team", "X-Container-Meta-Exact": "1", "write_ACL": "user"}, map[string]string{"X-Container-Meta-Owner": "team", "X-Container-Meta-Exact": "1", "X-Container-Write": "user"}},
		{"public access literal", map[string]any{"read_ACL": ".r:*,.rlistings"}, map[string]string{"X-Container-Read": ".r:*,.rlistings"}},
		{"private access literal", map[string]any{"read_ACL": ""}, map[string]string{"X-Container-Read": ""}},
	} {
		t.Run(tc.label, func(t *testing.T) {
			api, calls := pythonContainerAPI(t, func(_ pythonContainerCall, w http.ResponseWriter) { w.WriteHeader(204) })
			result, err := api.SetMetadataAndRefresh(context.Background(), "box", tc.values, containers.WithMetadataRefresh(false))
			if err != nil || result.Acknowledgement == nil || result.Acknowledgement.StatusCode != 204 || result.Refreshed != nil {
				t.Fatal(result, err)
			}
			got := calls()
			if len(got) != 1 || got[0].method != "POST" || got[0].path != "/swift/v1/AUTH_p/box" || got[0].body != "" {
				t.Fatal(got)
			}
			for key, value := range tc.want {
				if values, present := got[0].header[key]; !present || len(values) != 1 || values[0] != value {
					t.Fatal(key, got[0].header)
				}
			}
		})
	}
}

func TestPythonContainerCloudUpdateMissingContainerIsError(t *testing.T) {
	api, _ := pythonContainerAPI(t, func(_ pythonContainerCall, w http.ResponseWriter) { w.WriteHeader(404) })
	_, err := api.SetMetadataAndRefresh(context.Background(), "gone", map[string]any{"owner": "x"}, containers.WithMetadataRefresh(false))
	if !gophercloud.ResponseCodeIs(err, 404) || resource.IgnoreMissing(err) != nil {
		t.Fatal(err)
	}
}
