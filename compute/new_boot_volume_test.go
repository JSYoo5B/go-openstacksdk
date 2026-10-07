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

	"github.com/JSYoo5B/gophercloudsdk/compute"
	"github.com/JSYoo5B/gophercloudsdk/image"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func newVolumeBootRequest() compute.CreateServerRequest {
	return compute.CreateServerRequest{Name: "new-volume-server", Image: resource.ID("image-id"), Flavor: resource.ID("flavor-id")}
}

func TestCreateNewBootVolumeMapping(t *testing.T) {
	for _, tc := range []struct {
		name         string
		microversion string
		options      []compute.CreateServerOption
		volumeType   string
		deleted      bool
	}{
		{name: "preserved by default"},
		{name: "delete on server termination", options: []compute.CreateServerOption{compute.WithDeleteBootVolumeOnTermination(true)}, deleted: true},
		{name: "explicit volume type", microversion: "2.67", options: []compute.CreateServerOption{compute.WithBootVolumeType("fast")}, volumeType: "fast"},
		{name: "three digit minor", microversion: "2.100", options: []compute.CreateServerOption{compute.WithBootVolumeType("fast")}, volumeType: "fast"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("compute", "/compute")
			client.Microversion = tc.microversion
			service := compute.New(client, compute.Dependencies{
				Image: func(context.Context, resource.Ref) (string, error) {
					t.Error("ID must skip image resolution")
					return "", context.Canceled
				},
				Volume: func(context.Context, resource.Ref) (string, error) {
					t.Error("new volume must not resolve an existing volume")
					return "", context.Canceled
				},
			})
			cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Server map[string]any `json:"server"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				mapping := map[string]any{"source_type": "image", "destination_type": "volume", "uuid": "image-id", "boot_index": float64(0), "delete_on_termination": tc.deleted, "volume_size": float64(20)}
				if tc.volumeType != "" {
					mapping["volume_type"] = tc.volumeType
				}
				want := map[string]any{"name": "new-volume-server", "imageRef": "", "flavorRef": "flavor-id", "block_device_mapping_v2": []any{mapping}}
				if tc.microversion != "" {
					want["networks"] = "auto"
				}
				if !reflect.DeepEqual(body.Server, want) {
					t.Errorf("server body=%#v want=%#v", body.Server, want)
				}
				if got := r.Header.Get("OpenStack-API-Version"); tc.microversion != "" && got != "compute "+tc.microversion {
					t.Errorf("OpenStack-API-Version=%q", got)
				}
				testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("new volume creation must use Nova alone: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected request", 500)
			})
			options := append([]compute.CreateServerOption{compute.WithBootVolumeSize(20)}, tc.options...)
			server, err := service.Servers.Create(context.Background(), newVolumeBootRequest(), options...)
			if err != nil || server == nil || server.ID != "created" {
				t.Fatalf("server=%v err=%v", server, err)
			}
		})
	}
}

func TestCreateNewBootVolumeValidationBeforeLookup(t *testing.T) {
	cloud := testcloud.New(t)
	var requests, resolutions atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { requests.Add(1); http.Error(w, "must not request", 500) })
	resolve := func(context.Context, resource.Ref) (string, error) { resolutions.Add(1); return "resolved", nil }
	service := compute.New(cloud.Client("compute", "/compute"), compute.Dependencies{Image: resolve, Volume: resolve})
	request := newVolumeBootRequest()
	request.Image = resource.Name("ubuntu")
	for _, tc := range []struct {
		name    string
		request compute.CreateServerRequest
		options []compute.CreateServerOption
	}{
		{name: "zero size", request: request, options: []compute.CreateServerOption{compute.WithBootVolumeSize(0)}},
		{name: "negative size", request: request, options: []compute.CreateServerOption{compute.WithBootVolumeSize(-1)}},
		{name: "missing image", request: volumeBootRequest(), options: []compute.CreateServerOption{compute.WithBootVolumeSize(20)}},
		{name: "invalid image ID", request: compute.CreateServerRequest{Name: "server", Image: resource.ID("../image"), Flavor: resource.ID("flavor")}, options: []compute.CreateServerOption{compute.WithBootVolumeSize(20)}},
		{name: "existing and new volume", request: volumeBootRequest(), options: []compute.CreateServerOption{compute.WithBootVolume(resource.Name("root")), compute.WithBootVolumeSize(20)}},
		{name: "existing and new volume reversed", request: volumeBootRequest(), options: []compute.CreateServerOption{compute.WithBootVolumeSize(20), compute.WithBootVolume(resource.Name("root"))}},
		{name: "type without new volume", request: request, options: []compute.CreateServerOption{compute.WithBootVolumeType("fast")}},
		{name: "type on existing volume", request: volumeBootRequest(), options: []compute.CreateServerOption{compute.WithBootVolume(resource.Name("root")), compute.WithBootVolumeType("fast")}},
		{name: "empty volume type", request: request, options: []compute.CreateServerOption{compute.WithBootVolumeSize(20), compute.WithBootVolumeType(" ")}},
		{name: "invalid wait", request: request, options: []compute.CreateServerOption{compute.WithBootVolumeSize(20), compute.WithWait(resource.WithTimeout(0))}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := service.Servers.Create(context.Background(), tc.request, tc.options...); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("err=%v", err)
			}
		})
	}
	if requests.Load() != 0 || resolutions.Load() != 0 {
		t.Fatalf("requests=%d resolutions=%d", requests.Load(), resolutions.Load())
	}
}

func TestBootVolumeTypeMicroversionRequirementBeforeLookup(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unsupported volume type must not request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	for _, version := range []string{"", "2.9", "2.10", "2.66", "latest", "2.x", "2.67.1", "1.100"} {
		t.Run(version, func(t *testing.T) {
			client := cloud.Client("compute", "/compute")
			client.Microversion = version
			service := compute.New(client, compute.Dependencies{Image: func(context.Context, resource.Ref) (string, error) {
				t.Error("unsupported volume type must not resolve an image")
				return "", nil
			}})
			request := newVolumeBootRequest()
			request.Image = resource.Name("ubuntu")
			if _, err := service.Servers.Create(context.Background(), request, compute.WithBootVolumeSize(20), compute.WithBootVolumeType("fast")); !errors.Is(err, resource.ErrUnsupported) {
				t.Fatalf("microversion=%q err=%v", version, err)
			}
		})
	}
}

func TestCreateNewBootVolumeResolvesImageAndWaits(t *testing.T) {
	cloud := testcloud.New(t)
	images := image.New(cloud.Client("image", "/image"))
	service := compute.New(cloud.Client("compute", "/compute"), compute.Dependencies{Image: images.Images.ResolveID})
	var imageLists atomic.Int32
	cloud.Mux.HandleFunc("GET /image/images", func(w http.ResponseWriter, r *http.Request) {
		imageLists.Add(1)
		testcloud.JSON(w, 200, `{"images":[{"id":"image-id","name":"ubuntu"}]}`)
	})
	cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Server struct {
				ImageRef string `json:"imageRef"`
				Mapping  []struct {
					UUID string `json:"uuid"`
				} `json:"block_device_mapping_v2"`
			} `json:"server"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body.Server.ImageRef != "" || len(body.Server.Mapping) != 1 || body.Server.Mapping[0].UUID != "image-id" {
			t.Errorf("body=%+v", body)
		}
		testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
	})
	cloud.Mux.HandleFunc("GET /compute/servers/created", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"server":{"id":"created","status":"ACTIVE"}}`)
	})
	request := newVolumeBootRequest()
	request.Image = resource.Name("ubuntu")
	server, err := service.Servers.Create(context.Background(), request, compute.WithBootVolumeSize(20), compute.WithWait())
	if err != nil || server == nil || server.Status != "ACTIVE" || imageLists.Load() != 1 {
		t.Fatalf("server=%v err=%v image lists=%d", server, err, imageLists.Load())
	}
}

func TestCreateNewBootVolumeFailedImageResolutionDoesNotCreate(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("invalid image resolution must not request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	for _, id := range []string{"", "bad/image"} {
		service := compute.New(cloud.Client("compute", "/compute"), compute.Dependencies{Image: func(context.Context, resource.Ref) (string, error) { return id, nil }})
		request := newVolumeBootRequest()
		request.Image = resource.Name("ubuntu")
		if _, err := service.Servers.Create(context.Background(), request, compute.WithBootVolumeSize(20)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("id=%q err=%v", id, err)
		}
	}
}

func TestCreateNewBootVolumeWaitFailurePreservesServer(t *testing.T) {
	cloud := testcloud.New(t)
	service := compute.New(cloud.Client("compute", "/compute"), compute.Dependencies{})
	cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
	})
	cloud.Mux.HandleFunc("GET /compute/servers/created", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"server":{"id":"created","status":"BUILD"}}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("SDK must not roll back the new boot volume on wait failure: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected request", 500)
	})
	server, err := service.Servers.Create(context.Background(), newVolumeBootRequest(), compute.WithBootVolumeSize(20), compute.WithDeleteBootVolumeOnTermination(true), compute.WithWait(resource.WithTimeout(10*time.Millisecond)))
	if server == nil || server.ID != "created" || server.Status != "BUILD" || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("server=%v err=%v", server, err)
	}
}
