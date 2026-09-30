package compute_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"gophercloudsdk/blockstorage/v3/volumes"
	"gophercloudsdk/compute"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

func volumeBootRequest() compute.CreateServerRequest {
	return compute.CreateServerRequest{Name: "volume-server", Flavor: resource.ID("flavor-id")}
}

func TestCreateBootVolumeIDsAndDeletionPolicy(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options []compute.CreateServerOption
		deleted bool
	}{
		{name: "preserve by default"},
		{name: "delete explicitly", options: []compute.CreateServerOption{compute.WithDeleteBootVolumeOnTermination(true)}, deleted: true},
		{name: "false replaces true", options: []compute.CreateServerOption{compute.WithDeleteBootVolumeOnTermination(true), compute.WithDeleteBootVolumeOnTermination(false)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var posts atomic.Int32
			service := compute.New(cloud.Client("compute", "/compute"), compute.Dependencies{
				Volume: func(context.Context, resource.Ref) (string, error) {
					t.Error("an explicit volume ID must bypass the resolver")
					return "", errors.New("unexpected resolver")
				},
				Image: func(context.Context, resource.Ref) (string, error) {
					t.Error("volume boot must not resolve an image")
					return "", errors.New("unexpected resolver")
				},
			})
			cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				var body struct {
					Server map[string]any `json:"server"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				wantMapping := []any{map[string]any{
					"source_type": "volume", "destination_type": "volume", "uuid": "boot-volume-id",
					"boot_index": float64(0), "delete_on_termination": tc.deleted,
				}}
				if !reflect.DeepEqual(body.Server["block_device_mapping_v2"], wantMapping) {
					t.Errorf("mapping=%#v want=%#v", body.Server["block_device_mapping_v2"], wantMapping)
				}
				for key, want := range map[string]any{"name": "volume-server", "imageRef": "", "flavorRef": "flavor-id", "config_drive": false, "vendor_hint": false} {
					if body.Server[key] != want {
						t.Errorf("%s=%#v want=%#v", key, body.Server[key], want)
					}
				}
				testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("IDs must avoid lookup: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected request", 500)
			})
			options := append([]compute.CreateServerOption{compute.WithBootVolume(resource.ID("boot-volume-id")), compute.WithConfigDrive(false), compute.WithField("vendor_hint", false)}, tc.options...)
			// A functional option can be reused without mutating its captured input.
			for range 2 {
				server, err := service.Servers.Create(context.Background(), volumeBootRequest(), options...)
				if err != nil || server == nil || server.ID != "created" || server.Status != "BUILD" {
					t.Fatalf("server=%v err=%v", server, err)
				}
			}
			if posts.Load() != 2 {
				t.Fatalf("posts=%d", posts.Load())
			}
		})
	}
}

func TestCreateBootSourceValidationPrecedesEveryRequest(t *testing.T) {
	cloud := testcloud.New(t)
	var requests, resolutions atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "must not request", 500)
	})
	resolve := func(context.Context, resource.Ref) (string, error) { resolutions.Add(1); return "resolved", nil }
	service := compute.New(cloud.Client("compute", "/compute"), compute.Dependencies{Image: resolve, Volume: resolve, Network: resolve})
	imageRequest := compute.CreateServerRequest{Name: "image-server", Image: resource.Name("ubuntu"), Flavor: resource.Name("small")}
	for _, tc := range []struct {
		name    string
		request compute.CreateServerRequest
		options []compute.CreateServerOption
	}{
		{name: "no source", request: volumeBootRequest()},
		{name: "image and volume", request: imageRequest, options: []compute.CreateServerOption{compute.WithBootVolume(resource.Name("root"))}},
		{name: "empty volume", request: volumeBootRequest(), options: []compute.CreateServerOption{compute.WithBootVolume(resource.Ref{})}},
		{name: "invalid volume ID", request: volumeBootRequest(), options: []compute.CreateServerOption{compute.WithBootVolume(resource.ID("../volume"))}},
		{name: "invalid image", request: compute.CreateServerRequest{Name: "server", Image: resource.Name(""), Flavor: resource.ID("flavor")}},
		{name: "deletion without volume", request: imageRequest, options: []compute.CreateServerOption{compute.WithDeleteBootVolumeOnTermination(false)}},
		{name: "raw mapping override", request: volumeBootRequest(), options: []compute.CreateServerOption{compute.WithBootVolume(resource.Name("root")), compute.WithField("block_device_mapping_v2", []any{})}},
		{name: "legacy mapping override", request: volumeBootRequest(), options: []compute.CreateServerOption{compute.WithBootVolume(resource.Name("root")), compute.WithField("block_device_mapping", map[string]any{})}},
		{name: "invalid waiter", request: volumeBootRequest(), options: []compute.CreateServerOption{compute.WithBootVolume(resource.Name("root")), compute.WithWait(resource.WithTimeout(0))}},
		{name: "nil option", request: volumeBootRequest(), options: []compute.CreateServerOption{compute.WithBootVolume(resource.Name("root")), nil}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := service.Servers.Create(context.Background(), tc.request, tc.options...); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("err=%v", err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Servers.Create(ctx, volumeBootRequest(), compute.WithBootVolume(resource.Name("root"))); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if requests.Load() != 0 || resolutions.Load() != 0 {
		t.Fatalf("requests=%d resolutions=%d", requests.Load(), resolutions.Load())
	}
}

func TestCreateBootVolumeResolvesExactNameOnce(t *testing.T) {
	cloud := testcloud.New(t)
	api := volumes.New(cloud.Client("volumev3", "/volume"))
	service := compute.New(cloud.Client("compute", "/compute"), compute.Dependencies{Volume: api.Resources.ResolveID})
	var lists atomic.Int32
	cloud.Mux.HandleFunc("GET /volume/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if r.URL.Query().Get("name") != "boot[1].*" {
			t.Errorf("query=%s", r.URL.RawQuery)
		}
		testcloud.JSON(w, 200, `{"volumes":[{"id":"wrong","name":"boot[1].*copy"},{"id":"volume-id","name":"boot[1].*"}]}`)
	})
	cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Server struct {
				Mapping []struct {
					UUID string `json:"uuid"`
				} `json:"block_device_mapping_v2"`
			} `json:"server"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body.Server.Mapping) != 1 || body.Server.Mapping[0].UUID != "volume-id" {
			t.Errorf("body=%+v", body)
		}
		testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
	})
	server, err := service.Servers.Create(context.Background(), volumeBootRequest(), compute.WithBootVolume(resource.Name("boot[1].*")))
	if err != nil || server == nil || lists.Load() != 1 {
		t.Fatalf("server=%v err=%v lists=%d", server, err, lists.Load())
	}
}

func TestCreateBootVolumeLookupFailureDoesNotCreateServer(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want error
	}{
		{name: "missing", body: `{"volumes":[{"id":"other","name":"root-copy"}]}`, want: resource.ErrNotFound},
		{name: "ambiguous", body: `{"volumes":[{"id":"one","name":"root"},{"id":"two","name":"root"}]}`, want: resource.ErrAmbiguous},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			api := volumes.New(cloud.Client("volumev3", "/volume"))
			service := compute.New(cloud.Client("compute", "/compute"), compute.Dependencies{Volume: api.Resources.ResolveID})
			cloud.Mux.HandleFunc("GET /volume/volumes/detail", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, tc.body) })
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("failed lookup must not create: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected request", 500)
			})
			if _, err := service.Servers.Create(context.Background(), volumeBootRequest(), compute.WithBootVolume(resource.Name("root"))); !errors.Is(err, tc.want) {
				t.Fatalf("err=%v want=%v", err, tc.want)
			}
		})
	}
}

func TestCreateBootVolumeResolverErrorsRemainInspectable(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("resolver failure must not make an API call: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	request := volumeBootRequest()
	for _, tc := range []struct {
		name string
		deps compute.Dependencies
		want error
	}{
		{name: "missing resolver", want: resource.ErrUnsupported},
		{name: "empty resolved ID", deps: compute.Dependencies{Volume: func(context.Context, resource.Ref) (string, error) { return "", nil }}, want: resource.ErrInvalidOption},
		{name: "unsafe resolved ID", deps: compute.Dependencies{Volume: func(context.Context, resource.Ref) (string, error) { return "bad/id", nil }}, want: resource.ErrInvalidOption},
		{name: "cancellation", deps: compute.Dependencies{Volume: func(context.Context, resource.Ref) (string, error) { return "", context.Canceled }}, want: context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := compute.New(cloud.Client("compute", "/compute"), tc.deps)
			if _, err := service.Servers.Create(context.Background(), request, compute.WithBootVolume(resource.Name("root"))); !errors.Is(err, tc.want) {
				t.Fatalf("err=%v want=%v", err, tc.want)
			}
		})
	}
	api := volumes.New(cloud.Client("volumev3", "/volume"))
	service := compute.New(cloud.Client("compute", "/compute"), compute.Dependencies{Volume: api.Resources.ResolveID})
	cloud.Mux.HandleFunc("GET /volume/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 403, `{"forbidden":{"message":"denied"}}`)
	})
	_, err := service.Servers.Create(context.Background(), request, compute.WithBootVolume(resource.Name("root")))
	var responseError gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &responseError) || responseError.Actual != 403 {
		t.Fatalf("underlying HTTP error lost: %v", err)
	}
}

func TestCreateBootVolumeWaitFailurePreservesServerAndVolume(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status string
		want   error
	}{
		{name: "failed state", status: "ERROR", want: resource.ErrFailedState},
		{name: "timeout", status: "BUILD", want: context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			service := compute.New(cloud.Client("compute", "/compute"), compute.Dependencies{})
			cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
			})
			cloud.Mux.HandleFunc("GET /compute/servers/created", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"server":{"id":"created","status":"`+tc.status+`"}}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("a wait failure must not delete a server or volume: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected request", 500)
			})
			server, err := service.Servers.Create(context.Background(), volumeBootRequest(), compute.WithBootVolume(resource.ID("volume-id")), compute.WithDeleteBootVolumeOnTermination(true), compute.WithWait(resource.WithTimeout(10*time.Millisecond)))
			if server == nil || server.ID != "created" || server.Status != "BUILD" || !errors.Is(err, tc.want) {
				t.Fatalf("created server lost: server=%v err=%v", server, err)
			}
		})
	}
}

func TestCreateBootVolumeNovaFailurePreservesCause(t *testing.T) {
	cloud := testcloud.New(t)
	service := compute.New(cloud.Client("compute", "/compute"), compute.Dependencies{})
	cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 409, `{"conflictingRequest":{"message":"volume is in use"}}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("creation failure must not delete the existing volume: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	server, err := service.Servers.Create(context.Background(), volumeBootRequest(), compute.WithBootVolume(resource.ID("volume-id")))
	var responseError gophercloud.ErrUnexpectedResponseCode
	if server != nil || !errors.As(err, &responseError) || responseError.Actual != 409 {
		t.Fatalf("server=%v err=%v", server, err)
	}
}
