package gophercloudsdk_test

import (
	"context"
	"errors"
	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
	"net/http"
	"sync/atomic"
	"testing"
)

func TestConnectionAttachVolumeSharesCachedClientsAndLiveToken(t *testing.T) {
	cloud := testcloud.New(t)
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/proxy/nova/v2.1/p/"), sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/proxy/cinder/v3/p/"))
	if err != nil {
		t.Fatal(err)
	}
	nova, err := conn.ComputeV2(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	cinder, err := conn.BlockStorageV3(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	nova.RawClient().MoreHeaders = map[string]string{"X-Entry": "nova"}
	cinder.RawClient().MoreHeaders = map[string]string{"X-Entry": "cinder"}
	var gets, posts, options atomic.Int32
	cloud.Mux.HandleFunc("GET /proxy/cinder/v3/p/volumes/data", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if r.Header.Get("X-Auth-Token") != "live-token" || r.Header.Get("X-Entry") != "cinder" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"volume":{"id":"data","status":"available","attachments":[]}}`)
	})
	cloud.Mux.HandleFunc("POST /proxy/nova/v2.1/p/servers/server/os-volume_attachments", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		if r.Header.Get("X-Auth-Token") != "live-token" || r.Header.Get("X-Entry") != "nova" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"volumeAttachment":{"id":"ack","volumeId":"data","serverId":"server"}}`)
	})
	result, err := conn.AttachVolume(context.Background(), blockstorage.AttachVolumeRequest{Server: resource.ID("server"), Volume: resource.ID("data")}, blockstorage.WithAttachVolumeWait(false), func(o *blockstorage.AttachVolumeOpts) error {
		options.Add(1)
		cloud.Provider.SetToken("live-token")
		nova.RawClient().MoreHeaders["X-Entry"] = "after-entry"
		cinder.RawClient().MoreHeaders["X-Entry"] = "after-entry"
		return nil
	})
	if err != nil || result == nil || result.Created == nil || result.Created.Attachment == nil || result.Checked == nil || result.Ready != nil || gets.Load() != 1 || posts.Load() != 1 || options.Load() != 1 {
		t.Fatal(result, err, gets.Load(), posts.Load(), options.Load())
	}
	againNova, err := conn.ComputeV2(context.Background())
	if err != nil || againNova != nova {
		t.Fatal(againNova, err)
	}
	againCinder, err := conn.BlockStorageV3(context.Background())
	if err != nil || againCinder != cinder {
		t.Fatal(againCinder, err)
	}
	if nova.RawClient().ProviderClient != cloud.Provider || cinder.RawClient().ProviderClient != cloud.Provider {
		t.Fatal("workflow replaced shared provider")
	}
}

func TestConnectionAttachVolumeDiscoveryPrecedesOptionsAndIsCached(t *testing.T) {
	cloud := testcloud.New(t)
	var novaDiscovery, cinderDiscovery, gets, posts, options atomic.Int32
	cloud.Mux.HandleFunc("GET /nova/v2.1/{$}", func(w http.ResponseWriter, r *http.Request) {
		novaDiscovery.Add(1)
		testcloud.JSON(w, 200, `{"version":{"id":"v2.1","min_version":"2.1","version":"2.89"}}`)
	})
	cloud.Mux.HandleFunc("GET /cinder/v3/{$}", func(w http.ResponseWriter, r *http.Request) {
		cinderDiscovery.Add(1)
		testcloud.JSON(w, 200, `{"version":{"id":"v3.0","min_version":"3.0","version":"3.70"}}`)
	})
	cloud.Mux.HandleFunc("GET /cinder/v3/p/volumes/data", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if r.Header.Get("OpenStack-API-Version") != "volume 3.70" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.70" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"volume":{"id":"data","status":"available","attachments":[]}}`)
	})
	cloud.Mux.HandleFunc("POST /nova/v2.1/p/servers/server/os-volume_attachments", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		if r.Header.Get("OpenStack-API-Version") != "compute 2.89" || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.89" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"volumeAttachment":{"volumeId":"data"}}`)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/nova/v2.1/p/"), sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/cinder/v3/p/"), sdk.WithLatestMicroversion(sdk.Compute), sdk.WithLatestMicroversion(sdk.BlockStorage))
	if err != nil {
		t.Fatal(err)
	}
	input := blockstorage.AttachVolumeRequest{Server: resource.ID("server"), Volume: resource.ID("data")}
	stopped := errors.New("option stopped")
	result, err := conn.AttachVolume(context.Background(), input, func(o *blockstorage.AttachVolumeOpts) error {
		options.Add(1)
		if novaDiscovery.Load() != 1 || cinderDiscovery.Load() != 1 {
			t.Fatal("options applied before discovery", novaDiscovery.Load(), cinderDiscovery.Load())
		}
		return stopped
	})
	if result != nil || !errors.Is(err, stopped) || gets.Load() != 0 || posts.Load() != 0 || options.Load() != 1 {
		t.Fatal(result, err, gets.Load(), posts.Load(), options.Load())
	}
	result, err = conn.AttachVolume(context.Background(), input, blockstorage.WithAttachVolumeWait(false))
	if err != nil || result == nil || result.Created == nil || gets.Load() != 1 || posts.Load() != 1 || novaDiscovery.Load() != 1 || cinderDiscovery.Load() != 1 {
		t.Fatal(result, err, gets.Load(), posts.Load(), novaDiscovery.Load(), cinderDiscovery.Load())
	}
}

func TestConnectionAttachVolumeValidatesBeforeClientDiscovery(t *testing.T) {
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
	input := blockstorage.AttachVolumeRequest{Server: resource.ID("server"), Volume: resource.ID("data")}
	option := func(o *blockstorage.AttachVolumeOpts) error { callbacks.Add(1); return nil }
	cause := errors.New("caller canceled")
	canceled, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	var nilConn *sdk.Connection
	for _, tc := range []struct {
		name  string
		conn  *sdk.Connection
		ctx   context.Context
		input blockstorage.AttachVolumeRequest
		want  error
	}{
		{"nil-context", conn, nil, input, resource.ErrInvalidOption},
		{"nil-connection", nilConn, context.Background(), input, resource.ErrInvalidOption},
		{"canceled", conn, canceled, input, cause},
		{"missing-server", conn, context.Background(), blockstorage.AttachVolumeRequest{Volume: input.Volume}, resource.ErrInvalidOption},
		{"invalid-id", conn, context.Background(), blockstorage.AttachVolumeRequest{Server: input.Server, Volume: resource.ID("a/b")}, resource.ErrInvalidOption},
		{"invalid-utf8", conn, context.Background(), blockstorage.AttachVolumeRequest{Server: resource.Name(string([]byte{255})), Volume: input.Volume}, resource.ErrInvalidOption},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.conn.AttachVolume(tc.ctx, tc.input, option)
			if result != nil || !errors.Is(err, tc.want) {
				t.Fatal(result, err)
			}
			if tc.ctx == canceled && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
	if locates.Load() != 0 || callbacks.Load() != 0 {
		t.Fatal(locates.Load(), callbacks.Load())
	}
}

func TestConnectionAttachVolumeServiceInitializationCancellationCause(t *testing.T) {
	for _, canceledService := range []string{"compute", "cinder"} {
		t.Run(canceledService, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("service initialization stopped")
			var locates, callbacks atomic.Int32
			provider := &gophercloud.ProviderClient{}
			provider.UseTokenLock()
			provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if canceledService == "compute" || opts.Type != "compute" {
					cancel(cause)
				}
				return "https://example.test/v3/p/", nil
			}
			conn, err := sdk.FromProvider(provider)
			if err != nil {
				t.Fatal(err)
			}
			if canceledService == "cinder" {
				if _, err := conn.ComputeV2(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			result, err := conn.AttachVolume(ctx, blockstorage.AttachVolumeRequest{Server: resource.ID("server"), Volume: resource.ID("data")}, func(*blockstorage.AttachVolumeOpts) error { callbacks.Add(1); return nil })
			if result != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || callbacks.Load() != 0 {
				t.Fatal(result, err, callbacks.Load())
			}
			wanted := int32(1)
			if canceledService == "cinder" {
				wanted = 2
			}
			if locates.Load() != wanted {
				t.Fatal(locates.Load(), wanted)
			}
		})
	}
}
