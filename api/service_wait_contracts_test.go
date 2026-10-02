package api_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	snapshots2 "gophercloudsdk/blockstorage/v2/snapshots"
	volumes2 "gophercloudsdk/blockstorage/v2/volumes"
	snapshots3 "gophercloudsdk/blockstorage/v3/snapshots"
	volumes3 "gophercloudsdk/blockstorage/v3/volumes"
	"gophercloudsdk/compute"
	"gophercloudsdk/compute/v2/servers"
	"gophercloudsdk/image"
	"gophercloudsdk/image/v2/images"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

// Pinned proxy defaults: generic status waits are unlimited and fail on exact
// ERROR/error; server convenience and deletion waits use 120 seconds. These
// Ref-based Go calls observe native HTTP models rather than seeded Python cache.
type serviceWaitView struct{ ID, Status string }
type serviceWaitCall func(context.Context, resource.Ref, string, ...resource.WaitOption) (*serviceWaitView, error)
type serviceWaitBound struct {
	state, convenience, serverState, legacy serviceWaitCall
	delete, legacyDelete                    func(context.Context, resource.Ref, ...resource.WaitOption) error
	native                                  func(context.Context, string, string) error
}

func serviceWaitProject[T any](call func(context.Context, resource.Ref, string, ...resource.WaitOption) (*T, error), view func(*T) serviceWaitView) serviceWaitCall {
	if call == nil {
		return nil
	}
	return func(ctx context.Context, ref resource.Ref, target string, options ...resource.WaitOption) (*serviceWaitView, error) {
		value, err := call(ctx, ref, target, options...)
		if value == nil {
			return nil, err
		}
		projected := view(value)
		return &projected, err
	}
}

func serviceWaitBind[T any](state, convenience, serverState, legacy func(context.Context, resource.Ref, string, ...resource.WaitOption) (*T, error), delete, legacyDelete func(context.Context, resource.Ref, ...resource.WaitOption) error, native func(context.Context, string, string) error, view func(*T) serviceWaitView) serviceWaitBound {
	return serviceWaitBound{serviceWaitProject(state, view), serviceWaitProject(convenience, view), serviceWaitProject(serverState, view), serviceWaitProject(legacy, view), delete, legacyDelete, native}
}

type serviceWaitFixture struct {
	name, service, prefix, collection string
	defaultTarget, oldFailure         string
	progressSupported                 bool
	bind                              func(*gophercloud.ServiceClient) serviceWaitBound
}

func serviceWaitFixtures() []serviceWaitFixture {
	return []serviceWaitFixture{
		{"leaf-server", "compute", "/reverse/nova/v2/project/", "servers", "ACTIVE", "killed", true, func(c *gophercloud.ServiceClient) serviceWaitBound {
			a := servers.New(c)
			return serviceWaitBind(a.WaitForState, func(ctx context.Context, ref resource.Ref, _ string, o ...resource.WaitOption) (*servers.Server, error) {
				return a.WaitForServer(ctx, ref, o...)
			}, a.WaitForServerState, a.WaitFor, a.WaitForDelete, a.WaitForDeletion, a.WaitForStatus, func(v *servers.Server) serviceWaitView { return serviceWaitView{v.ID, v.Status} })
		}},
		{"leaf-volume-v2", "block-storage", "/reverse/cinder/v2/project/", "volumes", "available", "error_extending", true, func(c *gophercloud.ServiceClient) serviceWaitBound {
			a := volumes2.New(c)
			return serviceWaitBind(a.WaitForState, func(ctx context.Context, ref resource.Ref, _ string, o ...resource.WaitOption) (*volumes2.Volume, error) {
				return a.WaitForAvailable(ctx, ref, o...)
			}, nil, a.WaitFor, a.WaitForDelete, a.WaitForDeletion, a.WaitForStatus, func(v *volumes2.Volume) serviceWaitView { return serviceWaitView{v.ID, v.Status} })
		}},
		{"leaf-volume-v3", "block-storage", "/reverse/cinder/v3/project/", "volumes", "available", "error_extending", true, func(c *gophercloud.ServiceClient) serviceWaitBound {
			a := volumes3.New(c)
			return serviceWaitBind(a.WaitForState, func(ctx context.Context, ref resource.Ref, _ string, o ...resource.WaitOption) (*volumes3.Volume, error) {
				return a.WaitForAvailable(ctx, ref, o...)
			}, nil, a.WaitFor, a.WaitForDelete, a.WaitForDeletion, a.WaitForStatus, func(v *volumes3.Volume) serviceWaitView { return serviceWaitView{v.ID, v.Status} })
		}},
		{"leaf-snapshot-v2", "block-storage", "/reverse/cinder/v2/project/", "snapshots", "available", "error_deleting", true, func(c *gophercloud.ServiceClient) serviceWaitBound {
			a := snapshots2.New(c)
			return serviceWaitBind(a.WaitForState, func(ctx context.Context, ref resource.Ref, _ string, o ...resource.WaitOption) (*snapshots2.Snapshot, error) {
				return a.WaitForAvailable(ctx, ref, o...)
			}, nil, a.WaitFor, a.WaitForDelete, a.WaitForDeletion, a.WaitForStatus, func(v *snapshots2.Snapshot) serviceWaitView { return serviceWaitView{v.ID, v.Status} })
		}},
		{"leaf-snapshot-v3", "block-storage", "/reverse/cinder/v3/project/", "snapshots", "available", "error_deleting", true, func(c *gophercloud.ServiceClient) serviceWaitBound {
			a := snapshots3.New(c)
			return serviceWaitBind(a.WaitForState, func(ctx context.Context, ref resource.Ref, _ string, o ...resource.WaitOption) (*snapshots3.Snapshot, error) {
				return a.WaitForAvailable(ctx, ref, o...)
			}, nil, a.WaitFor, a.WaitForDelete, a.WaitForDeletion, a.WaitForStatus, func(v *snapshots3.Snapshot) serviceWaitView { return serviceWaitView{v.ID, v.Status} })
		}},
		{"leaf-image", "image", "/reverse/glance/v2/", "images", "", "killed", true, func(c *gophercloud.ServiceClient) serviceWaitBound {
			a := images.New(c)
			return serviceWaitBind(a.WaitForState, nil, nil, a.WaitFor, a.WaitForDelete, a.WaitForDeletion, nil, func(v *images.Image) serviceWaitView { return serviceWaitView{v.ID, string(v.Status)} })
		}},
		{"manual-compute", "compute", "/reverse/nova/v2/project/", "servers", "ACTIVE", "ERROR", true, func(c *gophercloud.ServiceClient) serviceWaitBound {
			s := compute.New(c, compute.Dependencies{}).Servers
			return serviceWaitBind(s.WaitForState, func(ctx context.Context, ref resource.Ref, _ string, o ...resource.WaitOption) (*compute.Server, error) {
				return s.WaitForServer(ctx, ref, o...)
			}, s.WaitForServerState, s.Wait, s.WaitForDelete, s.WaitDeleted, nil, func(v *compute.Server) serviceWaitView { return serviceWaitView{v.ID, v.Status} })
		}},
		{"manual-cinder", "block-storage", "/reverse/cinder/v3/project/", "volumes", "available", "error_extending", true, func(c *gophercloud.ServiceClient) serviceWaitBound {
			collection := blockstorage.New(c).Volumes
			return serviceWaitBind(func(ctx context.Context, ref resource.Ref, target string, o ...resource.WaitOption) (*blockstorage.Volume, error) {
				return blockstorage.WaitForState(ctx, collection, ref, target, o...)
			}, func(ctx context.Context, ref resource.Ref, _ string, o ...resource.WaitOption) (*blockstorage.Volume, error) {
				return blockstorage.WaitForAvailable(ctx, collection, ref, o...)
			}, nil, collection.Wait, func(ctx context.Context, ref resource.Ref, o ...resource.WaitOption) error {
				return blockstorage.WaitForDelete(ctx, collection, ref, o...)
			}, collection.WaitDeleted, nil, func(v *blockstorage.Volume) serviceWaitView { return serviceWaitView{v.ID, v.Status} })
		}},
		{"manual-glance", "image", "/reverse/glance/v2/", "images", "", "killed", true, func(c *gophercloud.ServiceClient) serviceWaitBound {
			collection := image.New(c).Images
			return serviceWaitBind(func(ctx context.Context, ref resource.Ref, target string, o ...resource.WaitOption) (*image.Image, error) {
				return image.WaitForState(ctx, collection, ref, target, o...)
			}, nil, nil, collection.Wait, func(ctx context.Context, ref resource.Ref, o ...resource.WaitOption) error {
				return image.WaitForDelete(ctx, collection, ref, o...)
			}, collection.WaitDeleted, nil, func(v *image.Image) serviceWaitView { return serviceWaitView{v.ID, string(v.Status)} })
		}},
	}
}

func (f serviceWaitFixture) path(id string) string { return f.prefix + f.collection + "/" + id }
func (f serviceWaitFixture) body(id, name, status string, progress int) string {
	row := fmt.Sprintf(`{"id":%q,"name":%q,"status":%q,"progress":%d}`, id, name, status, progress)
	if f.collection == "images" {
		return row
	}
	return fmt.Sprintf(`{%q:%s}`, strings.TrimSuffix(f.collection, "s"), row)
}
func (f serviceWaitFixture) listPath() string {
	if f.collection == "servers" || f.collection == "volumes" || strings.Contains(f.name, "snapshot-v3") {
		return f.prefix + f.collection + "/detail"
	}
	return f.prefix + f.collection
}

type serviceWaitDeadline struct {
	deadline time.Time
	set      bool
}

func TestServiceWaitContractsDefaultDeadlinesAndConvenienceTargets(t *testing.T) {
	for _, f := range serviceWaitFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var status atomic.Value
			status.Store("ACTIVE")
			var code atomic.Int32
			code.Store(200)
			observed := make(chan serviceWaitDeadline, 16)
			transport := cloud.Provider.HTTPClient.Transport
			if transport == nil {
				transport = http.DefaultTransport
			}
			cloud.Provider.HTTPClient.Transport = serviceWaitRoundTrip(func(r *http.Request) (*http.Response, error) {
				d, ok := r.Context().Deadline()
				observed <- serviceWaitDeadline{d, ok}
				return transport.RoundTrip(r)
			})
			cloud.Mux.HandleFunc(f.path("selected"), func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Error("wait mutated a resource", r.Method)
				}
				testcloud.JSON(w, int(code.Load()), f.body("selected", "worker", status.Load().(string), 100))
			})
			bound := f.bind(cloud.Client(f.service, f.prefix))
			v, err := bound.state(context.Background(), resource.ID("selected"), "active")
			if err != nil || v == nil || v.Status != "ACTIVE" || (<-observed).set {
				t.Fatal("generic status wait added a deadline or lost case-insensitive target", v, err)
			}
			parent, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			want, _ := parent.Deadline()
			v, err = bound.state(parent, resource.ID("selected"), "ACTIVE")
			got := <-observed
			if err != nil || v == nil || !got.set || !got.deadline.Equal(want) {
				t.Fatal("unlimited state wait changed parent deadline", got, want, err)
			}
			v, err = bound.state(context.Background(), resource.ID("selected"), "ACTIVE", resource.WithUnlimitedWait(), resource.WithTimeout(30*time.Second))
			got = <-observed
			if err != nil || v == nil || !got.set || time.Until(got.deadline) < 20*time.Second || time.Until(got.deadline) > 31*time.Second {
				t.Fatal("last timeout option did not apply", got, err)
			}
			v, err = bound.state(context.Background(), resource.ID("selected"), "ACTIVE", resource.WithTimeout(time.Second), resource.WithUnlimitedWait())
			if err != nil || v == nil || (<-observed).set {
				t.Fatal("last unlimited option did not apply", v, err)
			}
			if bound.convenience != nil {
				status.Store(f.defaultTarget)
				v, err = bound.convenience(context.Background(), resource.ID("selected"), "unused")
				got = <-observed
				finite := f.collection == "servers"
				if err != nil || v == nil || !strings.EqualFold(v.Status, f.defaultTarget) || got.set != finite || finite && (time.Until(got.deadline) < 100*time.Second || time.Until(got.deadline) > 121*time.Second) {
					t.Fatal("convenience default changed", v, got, err)
				}
			}
			if bound.serverState != nil {
				status.Store("SHUTOFF")
				v, err = bound.serverState(context.Background(), resource.ID("selected"), "SHUTOFF")
				got = <-observed
				if err != nil || v == nil || v.Status != "SHUTOFF" || !got.set || time.Until(got.deadline) < 100*time.Second || time.Until(got.deadline) > 121*time.Second {
					t.Fatal("explicit server target lost bounded default", v, got, err)
				}
				v, err = bound.serverState(context.Background(), resource.ID("selected"), "SHUTOFF", resource.WithUnlimitedWait())
				if err != nil || v == nil || (<-observed).set {
					t.Fatal("server override did not remove SDK deadline", v, err)
				}
			}
			code.Store(404)
			err = bound.delete(context.Background(), resource.ID("selected"))
			got = <-observed
			if err != nil || !got.set || time.Until(got.deadline) < 100*time.Second || time.Until(got.deadline) > 121*time.Second {
				t.Fatal("delete observation lost 120-second default", got, err)
			}
			err = bound.delete(context.Background(), resource.ID("selected"), resource.WithUnlimitedWait())
			if err != nil || (<-observed).set {
				t.Fatal("delete unlimited override changed", err)
			}
		})
	}
}

func TestServiceWaitContractsExactFailuresAndCallerOverrides(t *testing.T) {
	for _, f := range serviceWaitFixtures() {
		for _, scenario := range []string{"exact error", "broader old failure is not exact", "empty override", "custom snapshotted failure"} {
			t.Run(f.name+"/"+scenario, func(t *testing.T) {
				cloud := testcloud.New(t)
				var gets atomic.Int32
				cloud.Mux.HandleFunc(f.path("selected"), func(w http.ResponseWriter, r *http.Request) {
					state := "ACTIVE"
					if gets.Add(1) == 1 {
						switch scenario {
						case "exact error", "empty override":
							state = "eRrOr"
						case "broader old failure is not exact":
							state = "error_extending"
							if f.service != "block-storage" {
								state = "killed"
							}
						case "custom snapshotted failure":
							state = "WARNING"
						}
					}
					testcloud.JSON(w, 200, f.body("selected", "worker", state, 10))
				})
				options := []resource.WaitOption{resource.WithPollInterval(time.Millisecond)}
				if scenario == "empty override" {
					options = append(options, resource.WithFailureStates("WARNING"), resource.WithFailureStates())
				}
				if scenario == "custom snapshotted failure" {
					states := []string{"WARNING"}
					options = append(options, resource.WithFailureStates(states...))
					states[0] = "ERROR"
				}
				v, err := f.bind(cloud.Client(f.service, f.prefix)).state(context.Background(), resource.ID("selected"), "ACTIVE", options...)
				failed := scenario == "exact error" || scenario == "custom snapshotted failure"
				if failed {
					var failure *resource.FailedStateError
					if v != nil || !errors.Is(err, resource.ErrFailedState) || !errors.As(err, &failure) || failure.ID != "selected" || gets.Load() != 1 {
						t.Fatal(v, failure, err, gets.Load())
					}
				} else if err != nil || v == nil || v.Status != "ACTIVE" || gets.Load() != 2 {
					t.Fatal("exact failure/default override changed", v, err, gets.Load())
				}
			})
		}
	}
}

func TestServiceWaitContractsNameBindingLiveClientAndProgress(t *testing.T) {
	for _, f := range serviceWaitFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lists, gets, wrong, middleware atomic.Int32
			client := cloud.Client(f.service, "/discovery/unused/")
			client.ResourceBase = cloud.Server.URL + f.prefix
			client.MoreHeaders = map[string]string{"X-Wait-Trace": "configured"}
			if f.service == "compute" {
				client.Microversion = "2.90"
			} else if strings.Contains(f.prefix, "/v3/") {
				client.Microversion = "3.15"
			}
			type contextKey struct{}
			transport := cloud.Provider.HTTPClient.Transport
			if transport == nil {
				transport = http.DefaultTransport
			}
			cloud.Provider.HTTPClient.Transport = serviceWaitRoundTrip(func(r *http.Request) (*http.Response, error) {
				middleware.Add(1)
				if r.Context().Value(contextKey{}) != "caller" {
					t.Error("caller context was replaced")
				}
				return transport.RoundTrip(r)
			})
			cloud.Mux.HandleFunc(f.listPath(), func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				name := "worker"
				if f.collection == "servers" {
					name = "^worker$"
				}
				if r.Method != http.MethodGet || r.URL.Query().Get("name") != name || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Error(r.Method, r.URL, r.Header)
				}
				testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[{"id":"selected","name":"worker","status":"BUILD","progress":10}]}`, f.collection))
			})
			cloud.Mux.HandleFunc(f.path("selected"), func(w http.ResponseWriter, r *http.Request) {
				n := gets.Add(1)
				if r.Method != http.MethodGet || r.Header.Get("X-Wait-Trace") != "configured" {
					t.Error(r.Method, r.Header)
				}
				if f.service == "compute" && r.Header.Get("X-OpenStack-Nova-API-Version") != "2.90" || f.service == "block-storage" && client.Microversion != "" && r.Header.Get("X-OpenStack-Volume-API-Version") != "3.15" {
					t.Error("native version header changed", r.Header)
				}
				if n == 1 {
					if r.Header.Get("X-Auth-Token") != "test-token" {
						t.Error(r.Header)
					}
					cloud.Provider.SetToken("next-token")
					testcloud.JSON(w, 200, f.body("response-must-not-retarget", "response-name", "BUILD", 40))
				} else {
					if r.Header.Get("X-Auth-Token") != "next-token" {
						t.Error("provider token was copied rather than live", r.Header)
					}
					testcloud.JSON(w, 200, f.body("changed-response-id", "changed-response-name", "ACTIVE", 100))
				}
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				wrong.Add(1)
				t.Error("wait changed fixed parent or method", r.Method, r.URL)
				testcloud.JSON(w, 500, `{}`)
			})
			var progress []int
			options := []resource.WaitOption{resource.WithPollInterval(time.Millisecond)}
			if f.progressSupported {
				options = append(options, resource.WithProgressCallback(func(n int) { progress = append(progress, n) }))
			}
			v, err := f.bind(client).state(context.WithValue(context.Background(), contextKey{}, "caller"), resource.Name("worker"), "ACTIVE", options...)
			if err != nil || v == nil || v.ID != "changed-response-id" || v.Status != "ACTIVE" || lists.Load() != 1 || gets.Load() != 2 || wrong.Load() != 0 || middleware.Load() != 3 || client.ResourceBase != cloud.Server.URL+f.prefix {
				t.Fatal(v, err, lists.Load(), gets.Load(), wrong.Load(), middleware.Load())
			}
			if f.progressSupported {
				want := []int{0, 0}
				if f.collection == "servers" {
					want = []int{10, 40}
				}
				if !reflect.DeepEqual(progress, want) {
					t.Fatal("progress did not use native model or reported terminal success", progress, want)
				}
			}
		})
	}
}

func TestServiceWaitContractsDeletionOnlyObservesAndRejectsTerminalErrors(t *testing.T) {
	for _, f := range serviceWaitFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, mutations, lists atomic.Int32
			cloud.Mux.HandleFunc(f.path("selected"), func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					mutations.Add(1)
					testcloud.JSON(w, 500, `{}`)
					return
				}
				if gets.Add(1) == 1 {
					testcloud.JSON(w, 200, f.body("different-response-id", "worker", "BUILD", 10))
					return
				}
				testcloud.JSON(w, 404, `{"missing":"selected"}`)
			})
			cloud.Mux.HandleFunc(f.listPath(), func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				testcloud.JSON(w, 200, fmt.Sprintf(`{%q:[{"id":"selected","name":"worker","status":"BUILD"}]}`, f.collection))
			})
			var progress []int
			options := []resource.WaitOption{resource.WithPollInterval(time.Millisecond)}
			if f.progressSupported {
				options = append(options, resource.WithProgressCallback(func(n int) { progress = append(progress, n) }))
			}
			err := f.bind(cloud.Client(f.service, f.prefix)).delete(context.Background(), resource.Name("worker"), options...)
			if err != nil || gets.Load() != 2 || lists.Load() != 1 || mutations.Load() != 0 {
				t.Fatal("delete wait mutated or rerouted after returned ID changed", err, gets.Load(), lists.Load(), mutations.Load())
			}
			if f.progressSupported {
				want := []int{0}
				if f.collection == "servers" {
					want = []int{10}
				}
				if !reflect.DeepEqual(progress, want) {
					t.Fatal("delete progress included missing response", progress, want)
				}
			}
		})
		for _, outcome := range []string{"403", "transport404"} {
			t.Run(f.name+"/"+outcome, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc(f.path("selected"), func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					testcloud.JSON(w, 403, `{"forbidden":true}`)
				})
				var transportCause error
				if outcome == "transport404" {
					transportCause = &gophercloud.ErrUnexpectedResponseCode{Actual: 404}
					cloud.Provider.HTTPClient.Transport = serviceWaitRoundTrip(func(r *http.Request) (*http.Response, error) { calls.Add(1); return nil, transportCause })
				}
				err := f.bind(cloud.Client(f.service, f.prefix)).delete(context.Background(), resource.ID("selected"))
				if err == nil || calls.Load() != 1 {
					t.Fatal("failed read was treated as deletion", err, calls.Load())
				}
				if outcome == "403" {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != 403 {
						t.Fatal(err)
					}
				} else if !errors.Is(err, transportCause) || errors.Is(err, resource.ErrNotFound) {
					t.Fatal("transport cause changed", err)
				}
			})
		}
	}
}

func TestServiceWaitContractsStatusAttributesAndNativeProgressLimits(t *testing.T) {
	for _, f := range serviceWaitFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(f.path("selected"), func(w http.ResponseWriter, r *http.Request) {
				call := calls.Add(1)
				if f.name == "leaf-snapshot-v3" && call > 1 {
					status := "BUILD"
					if call == 3 {
						status = "ACTIVE"
					}
					testcloud.JSON(w, 200, fmt.Sprintf(`{"snapshot":{"id":"selected","status":%q,"os-extended-snapshot-attributes:progress":"70%%"}}`, status))
					return
				}
				testcloud.JSON(w, 200, f.body("selected", "ACTIVE", "ERROR", 100))
			})
			bound := f.bind(cloud.Client(f.service, f.prefix))
			v, err := bound.state(context.Background(), resource.ID("selected"), "active", resource.WithStatusAttribute("name"))
			if err != nil || v == nil || v.Status != "ERROR" || calls.Load() != 1 {
				t.Fatal("selected status attribute was ignored or model was fabricated", v, err, calls.Load())
			}
			for _, attribute := range []string{"missing_attribute", "size"} {
				if _, err := bound.state(context.Background(), resource.ID("selected"), "ACTIVE", resource.WithStatusAttribute(attribute)); !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 1 {
					t.Fatal("invalid native attribute reached HTTP", attribute, err, calls.Load())
				}
			}
			if f.name == "leaf-snapshot-v3" {
				// The native string uses an extension tag, not canonical progress.
				// The callback reports zero without interpreting that string.
				var progress []int
				v, err := bound.state(context.Background(), resource.ID("selected"), "ACTIVE", resource.WithPollInterval(time.Millisecond), resource.WithProgressCallback(func(value int) { progress = append(progress, value) }))
				if err != nil || v == nil || v.Status != "ACTIVE" || calls.Load() != 3 || !reflect.DeepEqual(progress, []int{0}) {
					t.Fatal("native extension progress was interpreted as canonical progress", v, err, calls.Load(), progress)
				}
			}
		})
	}
}

func TestServiceWaitContractsPreflightAndCallerCancellation(t *testing.T) {
	for _, f := range serviceWaitFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			arrived := make(chan struct{}, 1)
			cloud.Mux.HandleFunc(f.path("selected"), func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				arrived <- struct{}{}
				<-r.Context().Done()
			})
			bound := f.bind(cloud.Client(f.service, f.prefix))
			for _, bad := range []struct {
				ref     resource.Ref
				target  string
				options []resource.WaitOption
			}{
				{resource.ID("bad/id"), "ACTIVE", nil}, {resource.ID("selected"), " ", nil}, {resource.Name(""), "ACTIVE", nil},
				{resource.ID("selected"), "ACTIVE", []resource.WaitOption{nil}}, {resource.ID("selected"), "ACTIVE", []resource.WaitOption{resource.WithTimeout(0)}},
				{resource.ID("selected"), "ACTIVE", []resource.WaitOption{resource.WithPollInterval(0)}}, {resource.ID("selected"), "ACTIVE", []resource.WaitOption{resource.WithFailureStates(" ")}},
			} {
				if _, err := bound.state(context.Background(), bad.ref, bad.target, bad.options...); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
					t.Fatal("invalid wait performed HTTP", bad, err, calls.Load())
				}
			}
			if _, err := bound.state(nil, resource.ID("selected"), "ACTIVE"); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal("nil wait context was accepted", err, calls.Load())
			}
			if err := bound.delete(nil, resource.ID("selected")); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal("nil delete context was accepted", err, calls.Load())
			}
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() { _, err := bound.state(ctx, resource.ID("selected"), "ACTIVE"); done <- err }()
			<-arrived
			cancel()
			if err := <-done; !errors.Is(err, context.Canceled) || calls.Load() != 1 {
				t.Fatal(err, calls.Load())
			}
			if err := bound.delete(ctx, resource.ID("selected")); !errors.Is(err, context.Canceled) || calls.Load() != 1 {
				t.Fatal("canceled delete observation performed HTTP", err, calls.Load())
			}
		})
	}
	t.Run("nil API and collection guards", func(t *testing.T) {
		ctx := context.Background()
		ref := resource.ID("selected")
		var server *servers.API
		var volume2 *volumes2.API
		var volume3 *volumes3.API
		var snapshot2 *snapshots2.API
		var snapshot3 *snapshots3.API
		var glance *images.API
		var manual *compute.Servers
		for _, call := range []func() error{
			func() error { _, err := server.WaitForState(ctx, ref, "ACTIVE"); return err }, func() error { return server.WaitForDelete(ctx, ref) },
			func() error { _, err := volume2.WaitForState(ctx, ref, "available"); return err }, func() error { return volume2.WaitForDelete(ctx, ref) },
			func() error { _, err := volume3.WaitForAvailable(ctx, ref); return err }, func() error { return volume3.WaitForDelete(ctx, ref) },
			func() error { _, err := snapshot2.WaitForAvailable(ctx, ref); return err }, func() error { return snapshot2.WaitForDelete(ctx, ref) },
			func() error { _, err := snapshot3.WaitForState(ctx, ref, "available"); return err }, func() error { return snapshot3.WaitForDelete(ctx, ref) },
			func() error { _, err := glance.WaitForState(ctx, ref, "ACTIVE"); return err }, func() error { return glance.WaitForDelete(ctx, ref) },
			func() error { _, err := manual.WaitForServer(ctx, ref); return err }, func() error { return manual.WaitForDelete(ctx, ref) },
			func() error { _, err := compute.WaitForState[compute.Server](ctx, nil, ref, "ACTIVE"); return err },
			func() error { _, err := blockstorage.WaitForAvailable[blockstorage.Volume](ctx, nil, ref); return err },
			func() error { return image.WaitForDelete[image.Image](ctx, nil, ref) },
			func() error { _, err := (&servers.API{}).WaitForServerState(ctx, ref, "ACTIVE"); return err },
			func() error { _, err := (&volumes2.API{}).WaitForAvailable(ctx, ref); return err },
			func() error { _, err := (&volumes3.API{}).WaitForState(ctx, ref, "available"); return err },
			func() error { return (&snapshots2.API{}).WaitForDelete(ctx, ref) },
			func() error { return (&snapshots3.API{}).WaitForDelete(ctx, ref) },
			func() error { _, err := (&images.API{}).WaitForState(ctx, ref, "ACTIVE"); return err },
		} {
			if err := call(); !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal("nil API/collection did not use owned guard", err)
			}
		}
	})
}

func TestServiceWaitContractsLegacyAndNativeSignaturesStayUnchanged(t *testing.T) {
	for _, f := range serviceWaitFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			observed := make(chan serviceWaitDeadline, 2)
			transport := cloud.Provider.HTTPClient.Transport
			if transport == nil {
				transport = http.DefaultTransport
			}
			cloud.Provider.HTTPClient.Transport = serviceWaitRoundTrip(func(r *http.Request) (*http.Response, error) {
				d, ok := r.Context().Deadline()
				observed <- serviceWaitDeadline{d, ok}
				return transport.RoundTrip(r)
			})
			var code atomic.Int32
			code.Store(200)
			var status atomic.Value
			status.Store("ACTIVE")
			cloud.Mux.HandleFunc(f.path("selected"), func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, int(code.Load()), f.body("selected", "worker", status.Load().(string), 100))
			})
			bound := f.bind(cloud.Client(f.service, f.prefix))
			v, err := bound.legacy(context.Background(), resource.ID("selected"), "ACTIVE")
			got := <-observed
			if err != nil || v == nil || !got.set || time.Until(got.deadline) < 280*time.Second || time.Until(got.deadline) > 301*time.Second {
				t.Fatal("old WaitFor/Wait five-minute policy changed", v, got, err)
			}
			status.Store(f.oldFailure)
			v, err = bound.legacy(context.Background(), resource.ID("selected"), "ACTIVE")
			got = <-observed
			if v != nil || !errors.Is(err, resource.ErrFailedState) || !got.set {
				t.Fatal("old binding failure predicate changed", v, got, err)
			}
			code.Store(404)
			err = bound.legacyDelete(context.Background(), resource.ID("selected"))
			got = <-observed
			if err != nil || !got.set || time.Until(got.deadline) < 280*time.Second || time.Until(got.deadline) > 301*time.Second {
				t.Fatal("old deletion five-minute policy changed", got, err)
			}
		})
		if f.bind(nil).native == nil {
			continue
		}
		t.Run(f.name+"/native-first-predicate", func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			cloud.Provider.HTTPClient.Transport = serviceWaitRoundTrip(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Context().Err() != context.Canceled || r.Method != http.MethodGet || r.URL.Path != f.path("selected") {
					t.Error("native first predicate lost caller context or route", r.Context().Err(), r.Method, r.URL)
				}
				// A custom transport can complete the native first predicate even
				// with canceled context. Preserve that native predicate-first ABI.
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(f.body("selected", "worker", "ACTIVE", 100))), Request: r}, nil
			})
			bound := f.bind(cloud.Client(f.service, f.prefix))
			var native func(context.Context, string, string) error = bound.native
			if err := native(ctx, "selected", "ACTIVE"); err != nil || calls.Load() != 1 {
				t.Fatal("native WaitForStatus signature/predicate behavior changed", err, calls.Load())
			}
			if _, err := bound.state(ctx, resource.ID("selected"), "ACTIVE"); !errors.Is(err, context.Canceled) || calls.Load() != 1 {
				t.Fatal("owned wait failed to preflight canceled context", err, calls.Load())
			}
		})
	}
}

type serviceWaitRoundTrip func(*http.Request) (*http.Response, error)

func (f serviceWaitRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
