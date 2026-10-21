package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/acceleratorrequests"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/attributes"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/deployables"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/deviceprofiles"
	"github.com/JSYoo5B/go-openstacksdk/accelerator/v2/devices"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const pyHAAccelUUID = "22222222-2222-4222-8222-222222222222"

type pyHAAccelKind struct {
	name    string
	path    string
	listKey string
	item    string
	get     func(*testcloud.Cloud, context.Context, string) error
	all     func(*testcloud.Cloud, context.Context, ...resource.ListOption) (int, error)
	iter    func(*testcloud.Cloud, context.Context) (int, error)
}

func pyHAAccelKinds() []pyHAAccelKind {
	return []pyHAAccelKind{
		{"deployables", "/v2/deployables", "deployables", `{"uuid":"` + pyHAAccelUUID + `","name":"d"}`,
			func(c *testcloud.Cloud, ctx context.Context, id string) error {
				_, err := deployables.New(c.Client("accelerator", "/v2")).Get(ctx, id)
				return err
			},
			func(c *testcloud.Cloud, ctx context.Context, o ...resource.ListOption) (int, error) {
				v, err := deployables.New(c.Client("accelerator", "/v2")).All(ctx, o...)
				return len(v), err
			},
			func(c *testcloud.Cloud, ctx context.Context) (int, error) {
				n := 0
				for _, err := range deployables.New(c.Client("accelerator", "/v2")).List(ctx) {
					if err != nil {
						return n, err
					}
					n++
					break
				}
				return n, nil
			}},
		{"devices", "/v2/devices", "devices", `{"uuid":"` + pyHAAccelUUID + `","id":1}`,
			func(c *testcloud.Cloud, ctx context.Context, id string) error {
				_, err := devices.New(c.Client("accelerator", "/v2")).Get(ctx, id)
				return err
			},
			func(c *testcloud.Cloud, ctx context.Context, o ...resource.ListOption) (int, error) {
				v, err := devices.New(c.Client("accelerator", "/v2")).All(ctx, o...)
				return len(v), err
			},
			func(c *testcloud.Cloud, ctx context.Context) (int, error) {
				n := 0
				for _, err := range devices.New(c.Client("accelerator", "/v2")).List(ctx) {
					if err != nil {
						return n, err
					}
					n++
					break
				}
				return n, nil
			}},
		{"device_profiles", "/v2/device_profiles", "device_profiles", `{"uuid":"` + pyHAAccelUUID + `","name":"p","groups":[]}`,
			func(c *testcloud.Cloud, ctx context.Context, id string) error {
				_, err := deviceprofiles.New(c.Client("accelerator", "/v2")).Get(ctx, id)
				return err
			},
			func(c *testcloud.Cloud, ctx context.Context, o ...resource.ListOption) (int, error) {
				v, err := deviceprofiles.New(c.Client("accelerator", "/v2")).All(ctx, o...)
				return len(v), err
			},
			func(c *testcloud.Cloud, ctx context.Context) (int, error) {
				n := 0
				for _, err := range deviceprofiles.New(c.Client("accelerator", "/v2")).List(ctx) {
					if err != nil {
						return n, err
					}
					n++
					break
				}
				return n, nil
			}},
		{"accelerator_requests", "/v2/accelerator_requests", "arqs", `{"uuid":"` + pyHAAccelUUID + `","state":"Initial"}`,
			func(c *testcloud.Cloud, ctx context.Context, id string) error {
				_, err := acceleratorrequests.New(c.Client("accelerator", "/v2")).Get(ctx, id)
				return err
			},
			func(c *testcloud.Cloud, ctx context.Context, o ...resource.ListOption) (int, error) {
				v, err := acceleratorrequests.New(c.Client("accelerator", "/v2")).All(ctx, o...)
				return len(v), err
			},
			func(c *testcloud.Cloud, ctx context.Context) (int, error) {
				n := 0
				for _, err := range acceleratorrequests.New(c.Client("accelerator", "/v2")).List(ctx) {
					if err != nil {
						return n, err
					}
					n++
					break
				}
				return n, nil
			}},
		{"attributes", "/v2/attributes", "attributes", `{"uuid":"` + pyHAAccelUUID + `","key":"k","value":"v","deployable_id":1}`,
			func(c *testcloud.Cloud, ctx context.Context, id string) error {
				_, err := attributes.New(c.Client("accelerator", "/v2")).Get(ctx, id)
				return err
			},
			func(c *testcloud.Cloud, ctx context.Context, o ...resource.ListOption) (int, error) {
				v, err := attributes.New(c.Client("accelerator", "/v2")).All(ctx, o...)
				return len(v), err
			},
			func(c *testcloud.Cloud, ctx context.Context) (int, error) {
				n := 0
				for _, err := range attributes.New(c.Client("accelerator", "/v2")).List(ctx) {
					if err != nil {
						return n, err
					}
					n++
					break
				}
				return n, nil
			}},
	}
}

// Every Cyborg collection lists without a query by default, forwards only
// explicit WithQuery values, returns a non-nil empty slice and rejects caller
// limit/marker before HTTP.
func TestPyHAAccelListsQueryEmptyAndCallerPaging(t *testing.T) {
	for _, k := range pyHAAccelKinds() {
		t.Run(k.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var queries []string
			cloud.Mux.HandleFunc("GET "+k.path, func(w http.ResponseWriter, r *http.Request) {
				queries = append(queries, r.URL.RawQuery)
				testcloud.JSON(w, 200, `{"`+k.listKey+`":[`+k.item+`]}`)
			})
			if n, err := k.all(cloud, context.Background()); err != nil || n != 1 {
				t.Fatalf("plain: %d %v", n, err)
			}
			if n, err := k.all(cloud, context.Background(), resource.WithQuery("x", "y")); err != nil || n != 1 {
				t.Fatalf("query: %d %v", n, err)
			}
			for _, key := range []string{"limit", "marker"} {
				if _, err := k.all(cloud, context.Background(), resource.WithQuery(key, "1")); !errors.Is(err, resource.ErrUnsupported) {
					t.Fatalf("%s: %v", key, err)
				}
			}
			if len(queries) != 2 || queries[0] != "" || queries[1] != "x=y" {
				t.Fatalf("queries=%q", queries)
			}
			empty := testcloud.New(t)
			empty.Mux.HandleFunc("GET "+k.path, func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"`+k.listKey+`":[]}`)
			})
			if n, err := k.all(empty, context.Background()); err != nil || n != 0 {
				t.Fatalf("empty: %d %v", n, err)
			}
		})
	}
}

// The iterator is lazy: breaking after the first item never requests the next
// advertised page.
func TestPyHAAccelListIteratorBreakStopsPaging(t *testing.T) {
	for _, k := range pyHAAccelKinds() {
		t.Run(k.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET "+k.path, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 200, `{"`+k.listKey+`":[`+k.item+`],"next":"?marker=second"}`)
			})
			if n, err := k.iter(cloud, context.Background()); err != nil || n != 1 || calls.Load() != 1 {
				t.Fatalf("n=%d err=%v calls=%d", n, err, calls.Load())
			}
		})
	}
}

// Getters send one GET with no body, keep native 403/404 causes and send
// nothing for a canceled context.
func TestPyHAAccelGettersHTTPErrorsAndCancellation(t *testing.T) {
	for _, k := range pyHAAccelKinds() {
		t.Run(k.name, func(t *testing.T) {
			for _, status := range []int{403, 404} {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc(k.path+"/"+pyHAAccelUUID, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Method != "GET" || r.URL.RawQuery != "" {
						t.Errorf("request %s %s", r.Method, r.URL)
					}
					testcloud.JSON(w, status, `{"error":"x"}`)
				})
				if err := k.get(cloud, context.Background(), pyHAAccelUUID); !cyborgCloudIsStatus(err, status) || calls.Load() != 1 {
					t.Fatalf("%d: err=%v calls=%d", status, err, calls.Load())
				}
			}
			cloud, calls := cyborgCloudNoHTTP(t)
			if err := k.get(cloud, cyborgCloudCanceled(), pyHAAccelUUID); !errors.Is(err, context.Canceled) || calls.Load() != 0 {
				t.Fatalf("canceled: %v %d", err, calls.Load())
			}
		})
	}
}

// Deletes succeed only on 204, keep other statuses as native errors after one
// attempt and never send a request for a canceled context.
func TestPyHAAccelDeletesRequire204AndKeepNativeErrors(t *testing.T) {
	type del struct {
		name, path string
		fn         func(*testcloud.Cloud, context.Context) error
	}
	cases := []del{
		{"attributes", "/v2/attributes/" + pyHAAccelUUID, func(c *testcloud.Cloud, ctx context.Context) error {
			return attributes.New(c.Client("accelerator", "/v2")).Delete(ctx, resource.ID(pyHAAccelUUID), resource.WithMissingError())
		}},
		{"device_profiles", "/v2/device_profiles/" + pyHAAccelUUID, func(c *testcloud.Cloud, ctx context.Context) error {
			return deviceprofiles.New(c.Client("accelerator", "/v2")).Delete(ctx, resource.ID(pyHAAccelUUID), resource.WithMissingError())
		}},
		{"accelerator_requests", "/v2/accelerator_requests", func(c *testcloud.Cloud, ctx context.Context) error {
			return acceleratorrequests.New(c.Client("accelerator", "/v2")).Delete(ctx, resource.ID(pyHAAccelUUID), resource.WithMissingError())
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, status := range []int{204, 200, 403, 500} {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc(tc.path, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Method != "DELETE" {
						t.Errorf("method %s", r.Method)
					}
					if status == 204 {
						w.WriteHeader(204)
						return
					}
					testcloud.JSON(w, status, `{"error":"x"}`)
				})
				err := tc.fn(cloud, context.Background())
				if status == 204 && err != nil || status != 204 && !cyborgCloudIsStatus(err, status) || calls.Load() != 1 {
					t.Fatalf("%d: err=%v calls=%d", status, err, calls.Load())
				}
			}
			cloud, calls := cyborgCloudNoHTTP(t)
			if err := tc.fn(cloud, cyborgCloudCanceled()); !errors.Is(err, context.Canceled) || calls.Load() != 0 {
				t.Fatalf("canceled: %v %d", err, calls.Load())
			}
			hang := testcloud.New(t)
			var hung atomic.Int32
			cyborgCloudHang(hang, "DELETE "+tc.path, &hung)
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			if err := tc.fn(hang, ctx); !errors.Is(err, context.DeadlineExceeded) || hung.Load() != 1 {
				t.Fatalf("deadline: %v %d", err, hung.Load())
			}
		})
	}
}

// Device Enable and Disable both POST to the UUID action endpoint without a
// body, require 200 and fail before HTTP for a canceled context; the numeric
// database id is never used for routing.
func TestPyHAAccelDeviceActionsPositiveStrictAndCanceled(t *testing.T) {
	cloud := testcloud.New(t)
	var paths []string
	status := 200
	cloud.Mux.HandleFunc("/v2/devices/", func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		if r.ContentLength > 0 {
			t.Errorf("body sent")
		}
		w.WriteHeader(status)
	})
	a := devices.New(cloud.Client("accelerator", "/v2"))
	for _, fn := range []func(context.Context, resource.Ref, ...devices.ActionOption) (interface{}, error){
		func(ctx context.Context, r resource.Ref, o ...devices.ActionOption) (interface{}, error) {
			return a.Enable(ctx, r, o...)
		},
		func(ctx context.Context, r resource.Ref, o ...devices.ActionOption) (interface{}, error) {
			return a.Disable(ctx, r, o...)
		},
	} {
		status = 200
		if _, err := fn(context.Background(), resource.ID(pyHAAccelUUID)); err != nil {
			t.Fatal(err)
		}
		for _, bad := range []int{202, 204, 500} {
			status = bad
			if _, err := fn(context.Background(), resource.ID(pyHAAccelUUID)); !cyborgCloudIsStatus(err, bad) {
				t.Fatalf("status %d: %v", bad, err)
			}
		}
	}
	want := "POST /v2/devices/" + pyHAAccelUUID
	if len(paths) != 8 || paths[0] != want+"/enable" || paths[4] != want+"/disable" {
		t.Fatalf("paths=%q", paths)
	}
	n := len(paths)
	if _, err := a.Disable(cyborgCloudCanceled(), resource.ID(pyHAAccelUUID)); !errors.Is(err, context.Canceled) || len(paths) != n {
		t.Fatalf("canceled: %v", err)
	}
}

// Cyborg waiters poll with fixed identity: devices use status, ARQs use state,
// deletion succeeds on 404, configured failure states are terminal and an
// expired deadline keeps its cause.
func TestPyHAAccelWaitersStatusFailureDeletionAndDeadline(t *testing.T) {
	cloud := testcloud.New(t)
	var polls atomic.Int32
	cloud.Mux.HandleFunc("/v2/devices/"+pyHAAccelUUID, func(w http.ResponseWriter, r *http.Request) {
		status := "maintaining"
		if polls.Add(1) > 1 {
			status = "enabled"
		}
		testcloud.JSON(w, 200, `{"uuid":"`+pyHAAccelUUID+`","status":"`+status+`"}`)
	})
	cloud.Mux.HandleFunc("/v2/accelerator_requests/failing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"uuid":"failing","state":"Custom"}`)
	})
	cloud.Mux.HandleFunc("/v2/accelerator_requests/gone", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{"error":"gone"}`)
	})
	client := cloud.Client("accelerator", "/v2")
	client.Microversion = "2.3"
	d := devices.New(client)
	v, err := d.Wait(context.Background(), resource.ID(pyHAAccelUUID), "enabled", resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second))
	if err != nil || v.Status != "enabled" || polls.Load() != 2 {
		t.Fatalf("device wait: %+v %v %d", v, err, polls.Load())
	}
	a := acceleratorrequests.New(cloud.Client("accelerator", "/v2"))
	if _, err := a.Wait(context.Background(), resource.ID("failing"), "Bound", resource.WithFailureStates("Custom")); !errors.Is(err, resource.ErrFailedState) {
		t.Fatalf("failure: %v", err)
	}
	if err := a.WaitDeleted(context.Background(), resource.ID("gone")); err != nil {
		t.Fatalf("deleted: %v", err)
	}
	hang := testcloud.New(t)
	var hung atomic.Int32
	cyborgCloudHang(hang, "GET /v2/accelerator_requests/slow", &hung)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := acceleratorrequests.New(hang.Client("accelerator", "/v2")).Wait(ctx, resource.ID("slow"), "Bound"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("deadline: %v", err)
	}
	if _, err := a.Wait(cyborgCloudCanceled(), resource.ID("x"), "Bound"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled: %v", err)
	}
}

// Programming sends one PATCH to the corrected /program URL, requires 200,
// rejects reserved header overrides and invalid documents before HTTP, and
// keeps a deadline cause without replay.
func TestPyHAAccelProgrammingStatusHeadersAndCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	status := 200
	path := "/v2/deployables/" + pyHAAccelUUID + "/program"
	cloud.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "PATCH" {
			t.Errorf("method %s", r.Method)
		}
		testcloud.JSON(w, status, `{"uuid":"`+pyHAAccelUUID+`","name":"d"}`)
	})
	a := deployables.New(cloud.Client("accelerator", "/v2"))
	ref := resource.ID(pyHAAccelUUID)
	if _, err := a.Program(context.Background(), ref, pyHAAccelUUID); err != nil || calls.Load() != 1 {
		t.Fatalf("program: %v %d", err, calls.Load())
	}
	status = 202
	if _, err := a.Program(context.Background(), ref, pyHAAccelUUID); !cyborgCloudIsStatus(err, 202) || calls.Load() != 2 {
		t.Fatalf("202: %v %d", err, calls.Load())
	}
	for _, opt := range []deployables.ProgramOption{deployables.WithProgramHeader("X-Auth-Token", "x"), deployables.WithProgramHeader("Content-Type", "text/plain"), deployables.WithProgramHeader("openstack-api-version", "accelerator 2.0")} {
		if _, err := a.Program(context.Background(), ref, pyHAAccelUUID, opt); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("reserved header: %v", err)
		}
	}
	if _, err := a.Patch(context.Background(), ref, []deployables.ProgramOperation{{Op: "add", Path: "/program"}}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("document: %v", err)
	}
	if _, err := a.Program(cyborgCloudCanceled(), ref, pyHAAccelUUID); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("rejected input reached HTTP: %d", calls.Load())
	}
	hang := testcloud.New(t)
	var hung atomic.Int32
	cyborgCloudHang(hang, "PATCH "+path, &hung)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := deployables.New(hang.Client("accelerator", "/v2")).Program(ctx, ref, pyHAAccelUUID); !errors.Is(err, context.DeadlineExceeded) || hung.Load() != 1 {
		t.Fatalf("deadline: %v %d", err, hung.Load())
	}
}
