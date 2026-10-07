package gophercloudsdk_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionVolumeImageUploadUsesCachedCinderAndGatesOnlyOptionalPresence(t *testing.T) {
	cloud := testcloud.New(t)
	var getters, gets, posts atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		getters.Add(1)
		if opts.Type != "block-storage" || opts.Version != 3 {
			t.Error("unexpected Glance, Compute or other service", opts)
		}
		return cloud.Server.URL + vcmBase, nil
	}
	bad := string([]byte{0xff})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.90"), sdk.WithCloudLocation(resource.CloudLocation{Cloud: &bad, Zone: json.RawMessage("invalid"), Project: resource.CloudProject{ID: json.RawMessage("invalid")}}))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.BlockStorageV3(vcmContext(t))
	if err != nil {
		t.Fatal(err)
	}
	client := service.RawClient()
	client.MoreHeaders = map[string]string{"X-Upload-Caller": "captured", "OpenStack-API-Version": "volume 3.90", "X-OpenStack-Volume-API-Version": "3.90"}
	bodies := []string{
		`{"os-volume_upload_image":{"force":false,"image_name":""}}`,
		`{"os-volume_upload_image":{"force":false,"image_name":"new image","visibility":""}}`,
		`{"os-volume_upload_image":{"force":false,"image_name":"new image","protected":false}}`,
	}
	replies := []string{`{"os-volume_upload_image":{"a":"b"}}`, `{"os-volume_upload_image":null}`, `{"os-volume_upload_image":[false,9007199254740993]}`}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			n := gets.Add(1)
			if n > 3 || req.URL.EscapedPath() != "/proxy/cinder/v3/" || req.URL.RawQuery != "" || req.Header.Get("OpenStack-API-Version") != "" || req.Header.Get("X-OpenStack-Volume-API-Version") != "" || req.Header.Get("X-Upload-Caller") != "captured" || req.Header.Get("X-Auth-Token") != "test-token" || req.ContentLength != 0 || len(req.TransferEncoding) != 0 {
				t.Error("required probe changed policy", n, req.URL, req.Header)
			}
			w.Header().Set("X-Upload-Stage", "gate")
			if n == 3 {
				testcloud.JSON(w, 300, `{"id":"v3.0","min_version":"3.2","max_version":"3.99"}`)
			} else {
				testcloud.JSON(w, 300, `{"id":"v3.0","min_version":"3.0","max_version":"3.5"}`)
			}
			return
		}
		n := posts.Add(1)
		if n > int32(len(bodies)) {
			t.Error("unexpected refresh, Glance request or extra action", req.URL)
			w.WriteHeader(500)
			return
		}
		vcmWire(t, req, bodies[n-1], "3.90")
		if req.Header.Get("X-Upload-Caller") != "captured" {
			t.Error(req.Header)
		}
		w.Header().Set("X-Upload-Stage", "action")
		testcloud.JSON(w, 203, replies[n-1])
	})
	input := blockstorage.VolumeActionRequest{VolumeID: "literal-name"}
	result, err := conn.UploadVolumeToImage(vcmContext(t), input, "")
	if err != nil || result == nil || !result.Completed || result.VolumeID != input.VolumeID || result.Microversion != "3.90" || len(result.Discovery) != 0 || result.Applied == nil || result.Applied.StatusCode != 203 || string(result.Upload) != `{"a":"b"}` || gets.Load() != 0 || posts.Load() != 1 {
		t.Fatal("default upload performed a required probe or imposed image_id", result, err, gets.Load(), posts.Load())
	}
	appliedBefore := bytes.Clone(result.Applied.Body)
	result.Upload[0] = '['
	if !bytes.Equal(result.Applied.Body, appliedBefore) {
		t.Fatal("selected Upload shares Applied bytes", result)
	}
	calls := []struct {
		option blockstorage.VolumeImageUploadOption
		value  string
	}{
		{blockstorage.WithVolumeImageUploadVisibility(""), "null"},
		{blockstorage.WithVolumeImageUploadProtected(false), `[false,9007199254740993]`},
	}
	for _, call := range calls {
		result, err := conn.UploadVolumeToImage(vcmContext(t), input, "new image", call.option)
		if err != nil || result == nil || !result.Completed || result.VolumeID != input.VolumeID || result.Microversion != "3.90" || len(result.Discovery) != 1 || result.Discovery[0].StatusCode != 300 || result.Discovery[0].Header.Get("X-Upload-Stage") != "gate" || result.Applied == nil || result.Applied.StatusCode != 203 || result.Applied.Header.Get("X-Upload-Stage") != "action" || result.Applied == result.Discovery[0] || string(result.Upload) != call.value {
			t.Fatal("explicit false/empty presence lost required-support proof or raw selected version", result, err)
		}
	}
	failed, err := conn.UploadVolumeToImage(vcmContext(t), input, "new image", blockstorage.WithVolumeImageUploadProtected(false))
	vscError(t, "UploadVolumeToImage", err)
	if err == nil || failed == nil || failed.Completed || failed.VolumeID != input.VolumeID || failed.Applied != nil || failed.Upload != nil || len(failed.Discovery) != 1 || failed.Discovery[0].StatusCode != 300 || failed.Discovery[0].Header.Get("X-Upload-Stage") != "gate" {
		t.Fatal("selected3.90 bypassed fixed3.1 absent from server range", failed, err)
	}
	if getters.Load() != 1 || gets.Load() != 3 || posts.Load() != 3 || client.Microversion != "3.90" || client.MoreHeaders["OpenStack-API-Version"] != "volume 3.90" || client.MoreHeaders["X-OpenStack-Volume-API-Version"] != "3.90" {
		t.Fatal(getters.Load(), gets.Load(), posts.Load(), client)
	}
}

func TestConnectionVolumeImageUploadOriginalsRunOnceBeforeGetterAndRetainedPointersStayDetached(t *testing.T) {
	for _, required := range []bool{false, true} {
		name := "ordinary"
		if required {
			name = "required"
		}
		t.Run(name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var originals, getters, gets, posts atomic.Int32
			var retained *blockstorage.VolumeImageUploadOpts
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				getters.Add(1)
				if originals.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 || retained == nil {
					t.Error("getter preceded originals or selected another service", originals.Load(), opts, retained)
				}
				if retained == nil {
					return "", errors.New("getter ran before upload originals")
				}
				*retained.Force, *retained.DiskFormat, *retained.ContainerFormat = false, "getter disk", "getter container"
				if required {
					*retained.Visibility, *retained.Protected = "getter visibility", true
				}
				return cloud.Server.URL + vcmBase, nil
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.90"))
			if err != nil {
				t.Fatal(err)
			}
			body := `{"os-volume_upload_image":{"container_format":"literal /?%#\n\u0000","disk_format":"","force":true,"image_name":"image /?%#\n\u0000"}}`
			if required {
				body = `{"os-volume_upload_image":{"container_format":"literal /?%#\n\u0000","disk_format":"","force":true,"image_name":"image /?%#\n\u0000","protected":false,"visibility":""}}`
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodGet {
					gets.Add(1)
					if !required || gets.Load() > 1 || req.URL.EscapedPath() != "/proxy/cinder/v3/" || req.Header.Get("OpenStack-API-Version") != "" || req.Header.Get("X-OpenStack-Volume-API-Version") != "" {
						t.Error("ordinary options became required or probe changed scope", req.URL, req.Header)
					}
					testcloud.JSON(w, 300, `{"id":"v3.0","min_version":"3.0","max_version":"3.5"}`)
					return
				}
				if posts.Add(1) > 1 {
					t.Error("extra action or lookup", req.URL)
					w.WriteHeader(500)
					return
				}
				vcmWire(t, req, body, "3.90")
				testcloud.JSON(w, 200, `{"os-volume_upload_image":{"a":"b"}}`)
			})
			result, err := conn.UploadVolumeToImage(vcmContext(t), blockstorage.VolumeActionRequest{VolumeID: "literal-name"}, "image /?%#\n\x00", func(next *blockstorage.VolumeImageUploadOpts) error {
				originals.Add(1)
				force, disk, container := true, "", "literal /?%#\n\x00"
				next.Force, next.DiskFormat, next.ContainerFormat = &force, &disk, &container
				if required {
					visibility, protected := "", false
					next.Visibility, next.Protected = &visibility, &protected
				}
				retained = next
				return nil
			})
			wantDiscovery := 0
			if required {
				wantDiscovery = 1
			}
			if err != nil || result == nil || !result.Completed || result.Microversion != "3.90" || result.Applied == nil || result.Applied.StatusCode != 200 || string(result.Upload) != `{"a":"b"}` || len(result.Discovery) != wantDiscovery || originals.Load() != 1 || getters.Load() != 1 || gets.Load() != int32(wantDiscovery) || posts.Load() != 1 {
				t.Fatal(result, err, originals.Load(), getters.Load(), gets.Load(), posts.Load())
			}
		})
	}
}

func TestConnectionVolumeImageUploadPreflightOptionsLiteralAndGetterFailuresKeepEveryCause(t *testing.T) {
	for _, kind := range []string{"nil connection", "nil context", "already canceled", "nil option", "callback error and cancel", "invalid final option", "invalid image name", "unsafe ID", "getter error", "getter error and cancel", "cached wrong source"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx, cancel := context.WithCancelCause(vcmContext(t))
			defer cancel(nil)
			var selected context.Context = ctx
			callbackCause, cancelCause, getterCause := errors.New("upload Connection callback"), errors.New("upload Connection custom cancel"), errors.New("upload Connection getter")
			var getters, requests atomic.Int32
			first, later := 0, 0
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				getters.Add(1)
				if kind == "getter error and cancel" {
					cancel(cancelCause)
					return "", getterCause
				}
				if kind == "getter error" {
					return "", getterCause
				}
				return cloud.Server.URL + vcmBase, nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) { requests.Add(1); w.WriteHeader(500) })
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.90"))
			if err != nil {
				t.Fatal(err)
			}
			input, imageName := blockstorage.VolumeActionRequest{VolumeID: "literal-name"}, "new image"
			want, wantFirst, wantLater, wantGetters := resource.ErrInvalidOption, 1, 1, int32(0)
			switch kind {
			case "nil connection":
				conn = nil
				wantFirst, wantLater = 0, 0
			case "nil context":
				selected = nil
				wantFirst, wantLater = 0, 0
			case "already canceled":
				cancel(cancelCause)
				want = context.Canceled
				wantFirst, wantLater = 0, 0
			case "nil option":
				wantFirst, wantLater = 0, 0
			case "callback error and cancel":
				want = callbackCause
				wantLater = 0
			case "invalid image name":
				imageName = string([]byte{0xff})
			case "unsafe ID":
				input.VolumeID = "a/b"
			case "getter error", "getter error and cancel":
				want = getterCause
				wantGetters = 1
			case "cached wrong source":
				service, cause := conn.BlockStorageV3(vcmContext(t))
				if cause != nil {
					t.Fatal(cause)
				}
				service.RawClient().Type = "image"
				want, wantGetters = resource.ErrUnsupported, 1
			}
			options := []blockstorage.VolumeImageUploadOption{
				func(next *blockstorage.VolumeImageUploadOpts) error {
					first++
					disk := ""
					if kind == "invalid final option" {
						disk = string([]byte{0xff})
					}
					next.DiskFormat = &disk
					if kind == "callback error and cancel" {
						cancel(cancelCause)
						return callbackCause
					}
					return nil
				},
				func(*blockstorage.VolumeImageUploadOpts) error { later++; return nil },
			}
			if kind == "nil option" {
				options[0] = nil
			}
			result, err := conn.UploadVolumeToImage(selected, input, imageName, options...)
			vscError(t, "UploadVolumeToImage", err)
			var proof *resource.ResponseError
			if result != nil || !errors.Is(err, want) || errors.As(err, &proof) || first != wantFirst || later != wantLater || getters.Load() != wantGetters || requests.Load() != 0 {
				t.Fatal(result, err, first, later, getters.Load(), requests.Load())
			}
			if (kind == "already canceled" || kind == "callback error and cancel" || kind == "getter error and cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
				t.Fatal("custom context identities lost", err)
			}
			if kind == "callback error and cancel" && !errors.Is(err, callbackCause) {
				t.Fatal("callback identity lost", err)
			}
			if (kind == "getter error" || kind == "getter error and cancel") && !errors.Is(err, getterCause) {
				t.Fatal("getter identity lost", err)
			}
		})
	}
}
