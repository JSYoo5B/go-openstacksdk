package objectstorage_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestSwiftCloudContainerCreateOwnsGetOrCreateAndPublicDefault(t *testing.T) {
	for _, tc := range []struct {
		name             string
		existing, public bool
	}{{"existing private", true, false}, {"existing public ignored", true, true}, {"new private", false, false}, {"new public", false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := testcloud.New(t)
			client := fixture.Client("object-store", "/v1/AUTH_project/")
			var methods []string
			fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				methods = append(methods, r.Method)
				if body, err := io.ReadAll(r.Body); err != nil || len(body) != 0 {
					t.Error(string(body), err)
				}
				if r.URL.EscapedPath() != "/v1/AUTH_project/photos%20%252F%3F%23" || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Error(r.URL, r.Header)
				}
				w.Header().Set("X-Proof", fmt.Sprint(len(methods)))
				if r.Method == "HEAD" {
					if len(methods) == 1 && !tc.existing {
						w.WriteHeader(404)
						return
					}
					if _, present := r.Header["X-Container-Read"]; present {
						t.Error("ACL leaked into HEAD", r.Header)
					}
					w.Header().Set("X-Container-Read", ".r:*")
					w.Header().Set("X-Container-Object-Count", "4")
					w.WriteHeader(203)
					return
				}
				values, present := r.Header["X-Container-Read"]
				if tc.public {
					if !present || !reflect.DeepEqual(values, []string{".r:*,.rlistings"}) {
						t.Error(r.Header)
					}
				} else if present {
					t.Error("false must not emit private ACL", r.Header)
				}
				w.WriteHeader(202)
			})
			var options []objectstorage.ContainerCreateOption
			if tc.public {
				options = append(options, objectstorage.WithContainerPublic(true))
			}
			result, err := objectstorage.New(client).CreateContainer(context.Background(), "photos %2F?#", options...)
			if err != nil || result == nil || result.Existing != tc.existing || result.Container == nil || string(result.Container.Resource.Body["count"]) != "4" {
				t.Fatal(result, err)
			}
			want := []string{"HEAD", "PUT", "HEAD"}
			if tc.existing {
				want = []string{"HEAD"}
			}
			if !reflect.DeepEqual(methods, want) {
				t.Fatal(methods)
			}
			if tc.existing {
				if result.Creation != nil || result.Refreshed != nil || result.Container != result.Initial {
					t.Fatal(result)
				}
			} else if result.Initial.StatusCode != 404 || result.Creation.StatusCode != 202 || result.Container != result.Refreshed || result.Container.Header.Get("X-Proof") != "3" {
				t.Fatal(result)
			}
		})
	}
}

func TestSwiftCloudContainerCreateKeepsAcknowledgementWhenSecondHeadDisappearsOrFails(t *testing.T) {
	for _, status := range []int{404, 403, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			fixture := testcloud.New(t)
			var methods []string
			fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				methods = append(methods, r.Method)
				if len(methods) == 1 {
					w.WriteHeader(404)
				} else if len(methods) == 2 {
					w.Header().Set("X-Proof", "created")
					w.WriteHeader(201)
				} else {
					w.Header().Set("X-Proof", "final")
					w.WriteHeader(status)
				}
			})
			result, err := objectstorage.New(fixture.Client("object-store", "/account/")).CreateContainer(context.Background(), "photos")
			if result == nil || result.Container != nil || result.Creation.StatusCode != 201 || result.Creation.Header.Get("X-Proof") != "created" || result.Refreshed.StatusCode != status || !reflect.DeepEqual(methods, []string{"HEAD", "PUT", "HEAD"}) {
				t.Fatal(result, err, methods)
			}
			if status == 404 {
				if err != nil {
					t.Fatal(err)
				}
			} else if !gophercloud.ResponseCodeIs(err, status) {
				t.Fatal(err)
			}
		})
	}
}

func TestSwiftCloudContainerSetAccessOwnsTranslationAndRefreshDefault(t *testing.T) {
	for _, access := range []string{"public", "private"} {
		for _, refresh := range []bool{false, true} {
			t.Run(fmt.Sprint(access, refresh), func(t *testing.T) {
				fixture := testcloud.New(t)
				var methods []string
				fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					methods = append(methods, r.Method)
					if body, err := io.ReadAll(r.Body); err != nil || len(body) != 0 {
						t.Error(string(body), err)
					}
					if r.Method == "POST" {
						want := ""
						if access == "public" {
							want = ".r:*,.rlistings"
						}
						values, present := r.Header["X-Container-Read"]
						if !present || !reflect.DeepEqual(values, []string{want}) {
							t.Error(r.Header)
						}
						w.Header().Set("X-Proof", "post")
						w.WriteHeader(202)
					} else {
						if _, present := r.Header["X-Container-Read"]; present {
							t.Error("ACL leaked", r.Header)
						}
						w.Header().Set("X-Container-Read", "actually observed")
						w.Header().Set("X-Proof", "head")
						w.WriteHeader(203)
					}
				})
				var opts []objectstorage.ContainerAccessOption
				if refresh {
					opts = append(opts, objectstorage.WithContainerAccessRefresh(true))
				}
				result, err := objectstorage.New(fixture.Client("object-store", "/account/")).SetContainerAccess(context.Background(), "photos", access, opts...)
				if err != nil || result == nil || result.Requested != access || result.Acknowledgement.StatusCode != 202 || string(result.Acknowledgement.Resource.Body["read_ACL"]) != "null" {
					t.Fatal(result, err)
				}
				if refresh {
					if !reflect.DeepEqual(methods, []string{"POST", "HEAD"}) || result.Container != result.Refreshed || string(result.Container.Resource.Body["read_ACL"]) != `"actually observed"` {
						t.Fatal(result, methods)
					}
				} else if !reflect.DeepEqual(methods, []string{"POST"}) || result.Refreshed != nil || result.Container != result.Acknowledgement {
					t.Fatal(result, methods)
				}
			})
		}
	}
}

func TestSwiftCloudContainerSetAccessRejectsUnknownBeforeCallbacksOrHTTP(t *testing.T) {
	fixture := testcloud.New(t)
	calls, options := 0, 0
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(204) })
	cloud := objectstorage.New(fixture.Client("object-store", "/account/"))
	for _, access := range []string{"", "PUBLIC", "Public", ".r:*,.rlistings", "private ", "public\n"} {
		result, err := cloud.SetContainerAccess(context.Background(), "photos", access, func(*objectstorage.ContainerAccessOpts) error { options++; return nil })
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(access, result, err)
		}
	}
	if calls != 0 || options != 0 {
		t.Fatal(calls, options)
	}
}

func TestSwiftCloudContainerGetAccessClassifiesOnlyExactObservedACL(t *testing.T) {
	for _, tc := range []struct {
		acl, access string
		present     bool
	}{{"", "private", false}, {"", "private", true}, {".r:*,.rlistings", "public", true}, {".r:*", "", true}, {" .r:*,.rlistings", "public", true}, {".rlistings,.r:*", "", true}, {"private", "", true}} {
		t.Run(fmt.Sprint(tc.acl, tc.present), func(t *testing.T) {
			fixture := testcloud.New(t)
			calls := 0
			fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != "HEAD" || r.URL.RawQuery != "" {
					t.Error(r)
				}
				if tc.present {
					w.Header().Set("X-Container-Read", tc.acl)
				}
				w.Header().Set("X-Proof", "actual")
				w.WriteHeader(203)
			})
			result, err := objectstorage.New(fixture.Client("object-store", "/account/")).GetContainerAccess(context.Background(), "photos")
			if result == nil || result.Container.StatusCode != 203 || result.Container.Header.Get("X-Proof") != "actual" || result.Access != tc.access || calls != 1 {
				t.Fatal(result, err, calls)
			}
			if tc.access == "" {
				if err == nil {
					t.Fatal("unknown ACL succeeded")
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSwiftCloudContainerGetAccessMissingAndOtherStatusesKeepEvidence(t *testing.T) {
	for _, status := range []int{404, 403, 409, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			fixture := testcloud.New(t)
			fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Proof", "actual")
				w.WriteHeader(status)
			})
			result, err := objectstorage.New(fixture.Client("object-store", "/account/")).GetContainerAccess(context.Background(), "photos")
			if result == nil || result.Access != "" || result.Container.Resource != nil || result.Container.StatusCode != status || result.Container.Header.Get("X-Proof") != "actual" || !gophercloud.ResponseCodeIs(err, status) {
				t.Fatal(result, err)
			}
			if (status == 404) != errors.Is(err, resource.ErrNotFound) {
				t.Fatal(err)
			}
		})
	}
}

func TestSwiftCloudContainerOptionsOwnSnapshotsAndBulkReset(t *testing.T) {
	fixture := testcloud.New(t)
	client := fixture.Client("object-store", "/account/")
	var kept *objectstorage.ContainerCreateOpts
	headers := map[string]string{"X-Trace": "captured"}
	option := objectstorage.WithContainerCreateHeaders(headers)
	headers["X-Trace"] = "caller"
	calls := 0
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		kept.Headers["X-Trace"] = "late"
		if r.Header.Get("X-Trace") != "captured" {
			t.Error(r.Header)
		}
		if r.Method == "PUT" {
			if _, present := r.Header["X-Container-Read"]; present {
				t.Error(r.Header)
			}
		}
		if calls == 1 {
			w.WriteHeader(404)
		} else {
			w.WriteHeader(204)
		}
	})
	result, err := objectstorage.New(client).CreateContainer(context.Background(), "photos", objectstorage.WithContainerPublic(true), objectstorage.WithContainerCreateOpts(objectstorage.ContainerCreateOpts{}), option, func(c *objectstorage.ContainerCreateOpts) error { kept = c; return nil })
	if err != nil || result.Container == nil || calls != 3 || client.MoreHeaders != nil {
		t.Fatal(result, err, calls, client.MoreHeaders)
	}
}

func TestSwiftCloudContainerInvalidSourceAndInputsPrecedeCallbacks(t *testing.T) {
	fixture := testcloud.New(t)
	client := fixture.Client("object-store", "/account/")
	calls, options := 0, 0
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(204) })
	option := func(*objectstorage.ContainerCreateOpts) error { options++; return nil }
	for _, name := range []string{"", ".", "..", "a/b", "bad\\name", "bad\nname"} {
		result, err := objectstorage.New(client).CreateContainer(context.Background(), name, option)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(name, result, err)
		}
	}
	if _, err := objectstorage.New(client).CreateContainer(nil, "photos", option); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := (*objectstorage.Cloud)(nil).CreateContainer(context.Background(), "photos", option); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	wrong := *client
	wrong.Type = "image"
	if _, err := objectstorage.New(&wrong).CreateContainer(context.Background(), "photos", option); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("cancel before workflow")
	cancel(cause)
	if _, err := objectstorage.New(client).CreateContainer(ctx, "photos", option); !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal(err)
	}
	if calls != 0 || options != 0 {
		t.Fatal(calls, options)
	}
}

func TestSwiftCloudContainerOptionsCannotRerouteOrReplaceOwnedACL(t *testing.T) {
	fixture := testcloud.New(t)
	client := fixture.Client("object-store", "/account/")
	calls := 0
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(204) })
	result, err := objectstorage.New(client).CreateContainer(context.Background(), "photos", func(*objectstorage.ContainerCreateOpts) error { client.ResourceBase = client.Endpoint; return nil })
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
		t.Fatal(result, err, calls)
	}
	client.ResourceBase = ""
	for _, headers := range []map[string]string{{"X-Container-Read": "fake"}, {"Authorization": "fake"}, {"X-Probe": "bad\nvalue"}, {"x-probe": "one", "X-Probe": "two"}} {
		result, err := objectstorage.New(client).CreateContainer(context.Background(), "photos", objectstorage.WithContainerCreateHeaders(headers))
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatal(result, err, calls)
		}
	}
}

func TestSwiftCloudContainerReusableOptionsAreConcurrentAndIsolated(t *testing.T) {
	fixture := testcloud.New(t)
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Trace") != "captured" {
			t.Error(r.Header)
		}
		w.WriteHeader(204)
	})
	cloud := objectstorage.New(fixture.Client("object-store", "/account/"))
	headers := map[string]string{"X-Trace": "captured"}
	option := objectstorage.WithContainerReadHeaders(headers)
	headers["X-Trace"] = "caller"
	var wait sync.WaitGroup
	for i := 0; i < 12; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := cloud.GetContainerAccess(context.Background(), "photos", option)
			if err != nil || result.Access != "private" {
				t.Error(result, err)
			}
		}()
	}
	wait.Wait()
}

func TestSwiftCloudContainerSetAccessRefreshFailurePreservesPostAndStrictMissing(t *testing.T) {
	for _, status := range []int{404, 403, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			fixture := testcloud.New(t)
			var methods []string
			fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				methods = append(methods, r.Method)
				if r.Method == "POST" {
					w.Header().Set("X-Proof", "acknowledged")
					w.WriteHeader(202)
				} else {
					w.Header().Set("X-Proof", "failed refresh")
					w.WriteHeader(status)
				}
			})
			result, err := objectstorage.New(fixture.Client("object-store", "/account/")).SetContainerAccess(context.Background(), "photos", "public", objectstorage.WithContainerAccessRefresh(true))
			if !gophercloud.ResponseCodeIs(err, status) || result == nil || result.Container != nil || result.Acknowledgement.StatusCode != 202 || result.Acknowledgement.Header.Get("X-Proof") != "acknowledged" || result.Refreshed.StatusCode != status || !reflect.DeepEqual(methods, []string{"POST", "HEAD"}) {
				t.Fatal(result, err, methods)
			}
		})
	}
}

func TestSwiftCloudContainerNilOptionsAndCallbackCancellationStayBeforeHTTP(t *testing.T) {
	fixture := testcloud.New(t)
	calls := 0
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(204) })
	cloud := objectstorage.New(fixture.Client("object-store", "/account/"))
	if r, e := cloud.CreateContainer(context.Background(), "photos", nil); r != nil || !errors.Is(e, resource.ErrInvalidOption) {
		t.Fatal(r, e)
	}
	if r, e := cloud.SetContainerAccess(context.Background(), "photos", "public", nil); r != nil || !errors.Is(e, resource.ErrInvalidOption) {
		t.Fatal(r, e)
	}
	if r, e := cloud.GetContainerAccess(context.Background(), "photos", nil); r != nil || !errors.Is(e, resource.ErrInvalidOption) {
		t.Fatal(r, e)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause, callbackCause := errors.New("cancel from option"), errors.New("option cause")
	r, e := cloud.CreateContainer(ctx, "photos", func(c *objectstorage.ContainerCreateOpts) error { c.Headers = nil; cancel(cause); return callbackCause })
	if r != nil || !errors.Is(e, context.Canceled) || !errors.Is(e, cause) || !errors.Is(e, callbackCause) || calls != 0 {
		t.Fatal(r, e, calls)
	}
}

func TestSwiftCloudContainerNilHeaderCallbacksCanUseOwnedTypedHelpers(t *testing.T) {
	fixture := testcloud.New(t)
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Trace") != "after nil" {
			t.Error(r.Header)
		}
		w.WriteHeader(204)
	})
	cloud := objectstorage.New(fixture.Client("object-store", "/account/"))
	if _, err := cloud.CreateContainer(context.Background(), "photos", func(c *objectstorage.ContainerCreateOpts) error { c.Headers = nil; return nil }, objectstorage.WithContainerCreateHeaders(map[string]string{"X-Trace": "after nil"})); err != nil {
		t.Fatal(err)
	}
	if _, err := cloud.SetContainerAccess(context.Background(), "photos", "private", func(c *objectstorage.ContainerAccessOpts) error { c.Headers = nil; return nil }, objectstorage.WithContainerAccessHeaders(map[string]string{"X-Trace": "after nil"})); err != nil {
		t.Fatal(err)
	}
	if _, err := cloud.GetContainerAccess(context.Background(), "photos", func(c *objectstorage.ContainerReadOpts) error { c.Headers = nil; return nil }, objectstorage.WithContainerReadHeaders(map[string]string{"X-Trace": "after nil"})); err != nil {
		t.Fatal(err)
	}
}
