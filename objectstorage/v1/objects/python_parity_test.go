package objects_test

import (
	"context"
	"io"
	"net/http"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonObjectCall struct {
	method, path, query, body string
	header                    http.Header
}

// pythonObjectAPI serves the Swift account root /swift/v1/AUTH_p/ and records each request.
func pythonObjectAPI(t *testing.T, reply func(call pythonObjectCall, w http.ResponseWriter)) (*objects.API, func() []pythonObjectCall) {
	t.Helper()
	cloud := testcloud.New(t)
	var mu sync.Mutex
	var calls []pythonObjectCall
	cloud.Mux.HandleFunc("/swift/v1/AUTH_p/", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		call := pythonObjectCall{r.Method, r.URL.EscapedPath(), r.URL.RawQuery, string(body), r.Header.Clone()}
		mu.Lock()
		calls = append(calls, call)
		mu.Unlock()
		reply(call, w)
	})
	return objects.New(cloud.Client("object-store", "/swift/v1/AUTH_p/")), func() []pythonObjectCall {
		mu.Lock()
		defer mu.Unlock()
		return append([]pythonObjectCall(nil), calls...)
	}
}

func TestPythonObjectCloudGetObjectMetadataHead(t *testing.T) {
	api, calls := pythonObjectAPI(t, func(call pythonObjectCall, w http.ResponseWriter) {
		if call.path == "/swift/v1/AUTH_p/box/missing" {
			w.WriteHeader(404)
			return
		}
		w.Header().Set("X-Object-Meta-Owner", "team")
		w.Header().Set("X-Object-Meta-X-Sdk-Md5", "abc")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(200)
	})
	result, err := api.HeadMetadata(context.Background(), "box", "dir/file.txt")
	if err != nil || result.Metadata == nil || len(result.Metadata.Values) != 2 || result.Metadata.Values["owner"] != "team" || result.Metadata.Values["x-sdk-md5"] != "abc" {
		t.Fatalf("%+v %v", result, err)
	}
	if _, err := api.HeadMetadata(context.Background(), "box", "missing"); !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatal(err)
	}
	got := calls()
	// Python joins the raw name; Go escapes "/" and Swift decodes both to the same object path.
	if len(got) != 2 || got[0].method != "HEAD" || got[0].path != "/swift/v1/AUTH_p/box/dir%2Ffile.txt" || got[0].query != "" || got[0].body != "" {
		t.Fatal(got)
	}
}

func TestPythonObjectCloudUpdateObjectMergesFreshMetadata(t *testing.T) {
	api, calls := pythonObjectAPI(t, func(call pythonObjectCall, w http.ResponseWriter) {
		if call.method == "HEAD" {
			w.Header().Set("X-Object-Meta-Old", "keep")
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Last-Modified", "Mon, 01 Jan 2024 00:00:00 GMT")
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(202)
	})
	// Python update_object(container, name, metadata={"new": "v", "skip": ""}, content_type="app/x").
	result, err := api.SetMetadataFromFresh(context.Background(), "box", "obj", map[string]any{"new": "v", "skip": "", "content_type": "app/x"})
	if err != nil || result.Before == nil || result.Acknowledgement == nil || result.Acknowledgement.StatusCode != 202 || result.Refreshed != nil {
		t.Fatalf("%+v %v", result, err)
	}
	got := calls()
	if len(got) != 2 || got[0].method != "HEAD" || got[1].method != "POST" || got[1].path != "/swift/v1/AUTH_p/box/obj" || got[1].query != "" || got[1].body != "" {
		t.Fatal(got)
	}
	post := got[1].header
	if post.Get("X-Object-Meta-Old") != "keep" || post.Get("X-Object-Meta-New") != "v" || post.Get("Content-Type") != "app/x" {
		t.Fatal(post)
	}
	if _, present := post["X-Object-Meta-Skip"]; present {
		t.Fatal("falsey metadata value was sent", post)
	}
}

func TestPythonObjectCloudDeleteObjectOutcomes(t *testing.T) {
	for _, tc := range []struct {
		label      string
		head, del  int
		slo        bool
		requests   int
		deleted    bool
		failStatus int
	}{
		{"plain object", 200, 204, false, 2, true, 0},
		{"static large object deletes segments", 200, 200, true, 2, true, 0},
		{"missing at HEAD is false", 404, 0, false, 1, false, 0},
		{"missing at DELETE is false", 200, 404, false, 2, false, 0},
		{"conflict stays an error", 200, 409, false, 2, false, 409},
	} {
		t.Run(tc.label, func(t *testing.T) {
			api, calls := pythonObjectAPI(t, func(call pythonObjectCall, w http.ResponseWriter) {
				if call.method == "HEAD" {
					if tc.slo {
						w.Header().Set("X-Static-Large-Object", "True")
					}
					w.WriteHeader(tc.head)
					return
				}
				if tc.slo {
					testcloud.JSON(w, tc.del, `{"Number Deleted":3,"Number Not Found":0,"Errors":[],"Response Status":"200 OK","Response Body":""}`)
					return
				}
				w.WriteHeader(tc.del)
			})
			result, err := api.DeleteObject(context.Background(), "box", "obj")
			if tc.failStatus != 0 {
				if !gophercloud.ResponseCodeIs(err, tc.failStatus) {
					t.Fatal(result, err)
				}
			} else if err != nil || result == nil || result.IgnoredMissing == tc.deleted {
				t.Fatalf("%+v %v", result, err)
			}
			got := calls()
			if len(got) != tc.requests || got[0].method != "HEAD" || got[0].path != "/swift/v1/AUTH_p/box/obj" || got[0].query != "" {
				t.Fatal(got)
			}
			if tc.requests == 2 {
				query := ""
				if tc.slo {
					query = "multipart-manifest=delete"
				}
				if got[1].method != "DELETE" || got[1].path != "/swift/v1/AUTH_p/box/obj" || got[1].query != query {
					t.Fatal(got[1])
				}
			}
		})
	}
}
