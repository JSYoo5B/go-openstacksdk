package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionCreateVolumeCachedCinderLiveTokenAndOwnedImageSelection(t *testing.T) {
	cloud := testcloud.New(t)
	var imageLocates, posts, callbacks atomic.Int32
	missing := errors.New("no Glance endpoint")
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) { imageLocates.Add(1); return "", missing }
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/cinder/v3/p/"))
	if err != nil {
		t.Fatal(err)
	}
	cinder, err := conn.BlockStorageV3(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("POST /cinder/v3/p/volumes", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		if r.Header.Get("X-Auth-Token") != "live-token" || r.Header.Get("X-Entry") != "after-option" {
			t.Error(r.Header)
		}
		var body struct {
			Volume map[string]json.RawMessage `json:"volume"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		if string(body.Volume["size"]) != "5" || string(body.Volume["imageRef"]) != `"image-id"` {
			t.Error(body)
		}
		testcloud.JSON(w, 202, `{"volume":{"id":"new-volume","status":"creating","size":5}}`)
	})
	for range 2 {
		image := resource.ID("image-id")
		result, err := conn.CreateVolume(context.Background(), blockstorage.CreateVolumeRequest{Size: 5, Image: &image}, blockstorage.WithCreateVolumeWait(false), func(*blockstorage.CreateVolumeOpts) error {
			callbacks.Add(1)
			current, err := conn.BlockStorageV3(context.Background())
			if err != nil || current != cinder {
				t.Fatal(current, err)
			}
			image = resource.Name("changed by option")
			cinder.RawClient().MoreHeaders = map[string]string{"X-Entry": "after-option"}
			cloud.Provider.SetToken("live-token")
			return nil
		})
		if err != nil || result == nil || result.VolumeID != "new-volume" || result.Created == nil || result.Created.Volume == nil || result.Ready != nil || result.LastAccepted != nil || result.BootableSet != nil {
			t.Fatal(result, err)
		}
	}
	if posts.Load() != 2 || callbacks.Load() != 2 || imageLocates.Load() != 0 {
		t.Fatal(posts.Load(), callbacks.Load(), imageLocates.Load())
	}
	image := resource.Name("source")
	result, err := conn.CreateVolume(context.Background(), blockstorage.CreateVolumeRequest{Size: 5, Image: &image}, blockstorage.WithCreateVolumeWait(false))
	if result != nil || !errors.Is(err, missing) || imageLocates.Load() != 1 || posts.Load() != 2 {
		t.Fatal(result, err, imageLocates.Load(), posts.Load())
	}
}

func TestConnectionCreateVolumeOptionsPrecedeDiscoveryAndReuseMicroversion(t *testing.T) {
	cloud := testcloud.New(t)
	var discoveries, posts, gets, callbacks atomic.Int32
	cloud.Mux.HandleFunc("GET /cinder/v3/{$}", func(w http.ResponseWriter, r *http.Request) {
		discoveries.Add(1)
		testcloud.JSON(w, 200, `{"version":{"id":"v3.0","min_version":"3.0","version":"3.70"}}`)
	})
	cloud.Mux.HandleFunc("POST /cinder/v3/p/volumes", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		if r.Header.Get("OpenStack-API-Version") != "volume 3.70" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 202, `{"volume":{"id":"new-volume","status":"available"}}`)
	})
	cloud.Mux.HandleFunc("GET /cinder/v3/p/volumes/new-volume", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if r.Header.Get("X-OpenStack-Volume-API-Version") != "3.70" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"volume":{"id":"new-volume","status":"available"}}`)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/cinder/v3/p/"), sdk.WithLatestMicroversion(sdk.BlockStorage))
	if err != nil {
		t.Fatal(err)
	}
	rejected := errors.New("option rejected")
	result, err := conn.CreateVolume(context.Background(), blockstorage.CreateVolumeRequest{Size: 1}, func(*blockstorage.CreateVolumeOpts) error {
		callbacks.Add(1)
		if discoveries.Load() != 0 {
			t.Fatal("discovery preceded option")
		}
		return rejected
	})
	if result != nil || !errors.Is(err, rejected) || discoveries.Load() != 0 || posts.Load() != 0 || callbacks.Load() != 1 {
		t.Fatal(result, err, discoveries.Load(), posts.Load(), callbacks.Load())
	}
	for range 2 {
		result, err := conn.CreateVolume(context.Background(), blockstorage.CreateVolumeRequest{Size: 1}, blockstorage.WithCreateVolumeWait(false), blockstorage.WithCreateVolumeBootable(false))
		if err != nil || result == nil || result.Ready == nil || result.BootableSet != nil {
			t.Fatal(result, err)
		}
	}
	if discoveries.Load() != 1 || posts.Load() != 2 || gets.Load() != 2 {
		t.Fatal(discoveries.Load(), posts.Load(), gets.Load())
	}
}

func TestConnectionCreateVolumePreflightBeforeCallbacksAndServiceSelection(t *testing.T) {
	var locates, callbacks atomic.Int32
	provider := &gophercloud.ProviderClient{EndpointLocator: func(gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		return "https://example.test/v3/p/", nil
	}}
	provider.UseTokenLock()
	conn, err := sdk.FromProvider(provider)
	if err != nil {
		t.Fatal(err)
	}
	image := resource.ID("other/image")
	cause := errors.New("caller canceled creation")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	var nilConn *sdk.Connection
	for _, tc := range []struct {
		name  string
		conn  *sdk.Connection
		ctx   context.Context
		input blockstorage.CreateVolumeRequest
		want  error
	}{
		{"nil-context", conn, nil, blockstorage.CreateVolumeRequest{Size: 1}, resource.ErrInvalidOption},
		{"nil-connection", nilConn, context.Background(), blockstorage.CreateVolumeRequest{Size: 1}, resource.ErrInvalidOption},
		{"canceled", conn, ctx, blockstorage.CreateVolumeRequest{Size: 1}, cause},
		{"invalid-image", conn, context.Background(), blockstorage.CreateVolumeRequest{Size: 1, Image: &image}, resource.ErrInvalidOption},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.conn.CreateVolume(tc.ctx, tc.input, func(*blockstorage.CreateVolumeOpts) error { callbacks.Add(1); return nil })
			if result != nil || !errors.Is(err, tc.want) || tc.ctx == ctx && !errors.Is(err, context.Canceled) {
				t.Fatal(result, err)
			}
		})
	}
	negative := -time.Second
	for _, option := range []blockstorage.CreateVolumeOption{nil, blockstorage.WithCreateVolumeWaitPolicy(blockstorage.CreateVolumeWaitOpts{Timeout: &negative}), blockstorage.WithCreateVolumeFields(map[string]json.RawMessage{"base_path": json.RawMessage(`"other"`)})} {
		result, err := conn.CreateVolume(context.Background(), blockstorage.CreateVolumeRequest{Size: 1}, blockstorage.WithCreateVolumeWait(false), option)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(result, err)
		}
	}
	if locates.Load() != 0 || callbacks.Load() != 0 {
		t.Fatal(locates.Load(), callbacks.Load())
	}
}

func TestConnectionCreateVolumeInitializationRetainsCancellationCause(t *testing.T) {
	for _, service := range []string{"cinder", "image"} {
		t.Run(service, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("service initialization canceled")
			var locates, callbacks atomic.Int32
			provider := &gophercloud.ProviderClient{}
			provider.UseTokenLock()
			provider.EndpointLocator = func(o gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if service == "cinder" || o.Type == "image" {
					cancel(cause)
				}
				return "https://example.test/v3/p/", nil
			}
			conn, err := sdk.FromProvider(provider)
			if err != nil {
				t.Fatal(err)
			}
			if service == "image" {
				if _, err := conn.BlockStorageV3(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			image := resource.Name("source")
			result, err := conn.CreateVolume(ctx, blockstorage.CreateVolumeRequest{Size: 1, Image: &image}, func(*blockstorage.CreateVolumeOpts) error { callbacks.Add(1); return nil })
			expected := int32(1)
			if service == "image" {
				expected = 2
			}
			if result != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || callbacks.Load() != 1 || locates.Load() != expected {
				t.Fatal(result, err, callbacks.Load(), locates.Load())
			}
		})
	}
}
