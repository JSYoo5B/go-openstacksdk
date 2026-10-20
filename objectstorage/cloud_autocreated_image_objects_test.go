package objectstorage_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestSwiftAutocreatedImagesDefaultDisabledAndOwnedOptions(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("object-store", "/v1/AUTH_project/")
	calls := 0
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls++; t.Error("disabled HTTP") })
	got, err := objectstorage.New(client).DeleteAutocreatedImageObjects(context.Background())
	if err != nil || got == nil || got.Deleted || got.Container != "images" || len(got.Observations) != 0 || calls != 0 {
		t.Fatal(got, err, calls)
	}
	name := "chosen"
	prefix := "seg/"
	opt := objectstorage.WithAutocreatedImageObjectsOpts(objectstorage.AutocreatedImageObjectsOpts{Container: &name, SegmentPrefix: &prefix})
	name = "changed"
	prefix = "changed"
	got, err = objectstorage.New(nil).DeleteAutocreatedImageObjects(context.Background(), opt)
	if err != nil || got.Container != "chosen" {
		t.Fatal(got, err)
	}
}
func TestSwiftAutocreatedImagesExactMarkerLazyDeleteAndPageOrdering(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("object-store", "/v1/AUTH_project/")
	calls := []string{}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.EscapedPath()+"?"+r.URL.RawQuery)
		if r.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(r.Header)
		}
		switch r.Method {
		case "GET":
			if r.URL.Query().Get("prefix") != "seg/" {
				t.Error(r.URL)
			}
			w.Header().Set("X-Container-Object-Count", "3")
			if r.URL.Query().Get("marker") == "" {
				fmt.Fprint(w, `[{"name":"a"},{"name":"b"}]`)
			} else if r.URL.Query().Get("marker") == "b" {
				fmt.Fprint(w, `[{"name":"c"}]`)
			} else {
				t.Error("unexpected marker", r.URL)
				fmt.Fprint(w, `[]`)
			}
		case "HEAD":
			w.Header().Set("X-Static-Large-Object", "")
			switch r.URL.Path {
			case "/v1/AUTH_project/images/a":
				w.Header().Set("X-Object-Meta-X-Sdk-Autocreated", "true")
				w.Header().Set("X-Object-Meta-Proof", "first")
			case "/v1/AUTH_project/images/b":
				w.Header().Set("X-Object-Meta-X-Sdk-Autocreated", "True")
			case "/v1/AUTH_project/images/c":
				w.Header().Set("X-Object-Meta-X-Shade-Autocreated", "true")
			}
			w.WriteHeader(200)
		case "DELETE":
			if r.Header.Get("X-Object-Meta-X-Sdk-Autocreated") != "true" || r.Header.Get("X-Object-Meta-Proof") != "first" {
				t.Error(r.Header)
			}
			w.WriteHeader(404)
		}
	})
	got, err := objectstorage.New(client).DeleteAutocreatedImageObjects(context.Background(), objectstorage.WithAutocreatedImageTasks(true), objectstorage.WithAutocreatedImagePrefix("seg/"))
	want := []string{"GET /v1/AUTH_project/images?format=json&prefix=seg%2F", "HEAD /v1/AUTH_project/images/a?", "DELETE /v1/AUTH_project/images/a?", "HEAD /v1/AUTH_project/images/b?", "GET /v1/AUTH_project/images?format=json&marker=b&prefix=seg%2F", "HEAD /v1/AUTH_project/images/c?"}
	if err != nil || got == nil || !got.Deleted || len(got.Observations) != 3 || !got.Observations[0].IgnoredMissing || got.Observations[1].Matched || got.Observations[2].Matched || !reflect.DeepEqual(calls, want) {
		t.Fatal(got, err, calls)
	}
	if got.Observations[0].Deletion.StatusCode != 404 || got.Observations[0].Metadata.StatusCode != 200 {
		t.Fatal(got)
	}
}
func TestSwiftAutocreatedImagesSLOTruthinessAndSecondHead(t *testing.T) {
	for _, mode := range []string{"false text is truthy", "empty header false", "second head", "second missing", "listed flag retained"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("object-store", "/v1/AUTH_project/")
			heads, deletes := 0, 0
			query := ""
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case "GET":
					if r.URL.Query().Get("marker") != "" {
						fmt.Fprint(w, `[]`)
					} else if mode == "listed flag retained" {
						fmt.Fprint(w, `[{"name":"x","is_static_large_object":true}]`)
					} else {
						fmt.Fprint(w, `[{"name":"x"}]`)
					}
				case "HEAD":
					heads++
					w.Header().Set("X-Object-Meta-X-Sdk-Autocreated", "true")
					w.Header().Set("X-Object-Meta-Proof", fmt.Sprint(heads))
					if mode == "false text is truthy" {
						w.Header().Set("X-Static-Large-Object", "false")
					}
					if mode == "empty header false" {
						w.Header().Set("X-Static-Large-Object", "")
					}
					if heads == 2 && mode == "second missing" {
						w.WriteHeader(404)
						return
					}
					w.WriteHeader(200)
				case "DELETE":
					deletes++
					query = r.URL.RawQuery
					if r.Header.Get("X-Object-Meta-Proof") != fmt.Sprint(heads) {
						t.Error(r.Header)
					}
					w.WriteHeader(202)
				}
			})
			got, err := objectstorage.New(client).DeleteAutocreatedImageObjects(context.Background(), objectstorage.WithAutocreatedImageTasks(true))
			wantHeads := 1
			if mode == "second head" || mode == "second missing" {
				wantHeads = 2
			}
			wantDeletes := 1
			if mode == "second missing" {
				wantDeletes = 0
			}
			wantQuery := ""
			if mode == "false text is truthy" || mode == "listed flag retained" {
				wantQuery = "multipart-manifest=delete"
			}
			if err != nil || got == nil || !got.Deleted || heads != wantHeads || deletes != wantDeletes || query != wantQuery {
				t.Fatal(got, err, heads, deletes, query)
			}
			if mode == "second missing" && (!got.Observations[0].IgnoredMissing || got.Observations[0].Deletion != nil || got.Observations[0].Discovery == nil || got.Observations[0].Discovery.Metadata != nil || got.Observations[0].Discovery.StatusCode != 404) {
				t.Fatal(got)
			}
		})
	}
}
func TestSwiftAutocreatedImagesErrorsStopAfterCompletedPhases(t *testing.T) {
	for _, phase := range []string{"list", "first head", "delete", "second head"} {
		t.Run(phase, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("object-store", "/v1/AUTH_project/")
			calls := []string{}
			heads := 0
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, r.Method)
				w.Header().Set("X-Failure-Proof", phase)
				if phase == "list" {
					w.WriteHeader(500)
					return
				}
				switch r.Method {
				case "GET":
					fmt.Fprint(w, `[{"name":"a"},{"name":"b"}]`)
				case "HEAD":
					heads++
					if phase == "first head" {
						w.WriteHeader(404)
						return
					}
					if phase == "second head" && heads == 2 {
						w.WriteHeader(503)
						return
					}
					w.Header().Set("X-Object-Meta-X-Sdk-Autocreated", "true")
					if phase != "second head" {
						w.Header().Set("X-Static-Large-Object", "")
					}
					w.WriteHeader(200)
				case "DELETE":
					w.WriteHeader(409)
					fmt.Fprint(w, `{"error":"delete"}`)
				}
			})
			got, err := objectstorage.New(client).DeleteAutocreatedImageObjects(context.Background(), objectstorage.WithAutocreatedImageTasks(true))
			var native gophercloud.ErrUnexpectedResponseCode
			if err == nil || got == nil || got.Deleted || !errors.As(err, &native) {
				t.Fatal(got, err, calls)
			}
			want := []string{"GET"}
			switch phase {
			case "first head":
				want = append(want, "HEAD")
			case "delete":
				want = append(want, "HEAD", "DELETE")
			case "second head":
				want = append(want, "HEAD", "HEAD")
			}
			if !reflect.DeepEqual(calls, want) {
				t.Fatal(calls, want)
			}
			if phase != "list" {
				observation := got.Observations[0]
				switch phase {
				case "first head":
					if observation.Metadata == nil || observation.Metadata.Metadata != nil || observation.Metadata.StatusCode != 404 || len(observation.Metadata.Body) != 0 || observation.Metadata.Header.Get("X-Failure-Proof") != phase {
						t.Fatal(observation)
					}
				case "second head":
					if observation.Discovery == nil || observation.Discovery.Metadata != nil || observation.Discovery.StatusCode != 503 || len(observation.Discovery.Body) != 0 || observation.Discovery.Header.Get("X-Failure-Proof") != phase {
						t.Fatal(observation)
					}
				case "delete":
					if observation.Deletion == nil || observation.Deletion.StatusCode != 409 || string(observation.Deletion.Body) != `{"error":"delete"}` || observation.Deletion.Header.Get("X-Failure-Proof") != phase {
						t.Fatal(observation)
					}
				}
			}
		})
	}
}
func TestSwiftAutocreatedImagesPreflightAndCancellation(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("stop cleanup")
	cancel(cause)
	if _, err := objectstorage.New(nil).DeleteAutocreatedImageObjects(ctx); !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	var nilCloud *objectstorage.Cloud
	if _, err := nilCloud.DeleteAutocreatedImageObjects(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := objectstorage.New(nil).DeleteAutocreatedImageObjects(context.Background(), nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	cloud := testcloud.New(t)
	client := cloud.Client("object-store", "/v1/AUTH_project/")
	calls := 0
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls++; t.Error("preflight HTTP") })
	entered := false
	_, err := objectstorage.New(client).DeleteAutocreatedImageObjects(context.Background(), func(cfg *objectstorage.AutocreatedImageObjectsOpts) error {
		entered = true
		client.ResourceBase = "https://changed.invalid/"
		cfg.ImageAPIUseTasks = true
		return nil
	})
	if !entered || !errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(err.Error(), "source or route changed") || calls != 0 {
		t.Fatal(err, calls, entered)
	}
}
