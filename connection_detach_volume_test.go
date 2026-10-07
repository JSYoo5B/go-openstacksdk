package gophercloudsdk_test

import (
	"context"
	"errors"
	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestConnectionDetachVolumeSharesCachedClientsAndLiveToken(t *testing.T) {
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
	var gets, deletes, options atomic.Int32
	cloud.Mux.HandleFunc("GET /proxy/cinder/v3/p/volumes/data", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if r.Header.Get("X-Auth-Token") != "live-token" || r.Header.Get("X-Entry") != "after-entry" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `{"volume":{"id":"data","status":"available","attachments":[]}}`)
	})
	cloud.Mux.HandleFunc("DELETE /proxy/nova/v2.1/p/servers/server/os-volume_attachments/data", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		if r.Header.Get("X-Auth-Token") != "live-token" || r.Header.Get("X-Entry") != "after-entry" {
			t.Error(r.Header)
		}
		w.WriteHeader(202)
	})
	result, err := conn.DetachVolume(context.Background(), blockstorage.DetachVolumeRequest{Server: resource.ID("server"), Volume: resource.ID("data")}, func(o *blockstorage.DetachVolumeOpts) error {
		options.Add(1)
		if current, err := conn.ComputeV2(context.Background()); err != nil || current != nova {
			t.Fatal(current, err)
		}
		cloud.Provider.SetToken("live-token")
		nova.RawClient().MoreHeaders["X-Entry"] = "after-entry"
		cinder.RawClient().MoreHeaders["X-Entry"] = "after-entry"
		return nil
	})
	if err != nil || result == nil || result.Deleted == nil || result.Deleted.StatusCode != 202 || result.LastAccepted == nil || result.Ready == nil || gets.Load() != 1 || deletes.Load() != 1 || options.Load() != 1 {
		t.Fatal(result, err, gets.Load(), deletes.Load(), options.Load())
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

func TestConnectionDetachVolumeOptionsPrecedeDiscoveryAndAreCached(t *testing.T) {
	cloud := testcloud.New(t)
	var novaDiscovery, cinderDiscovery, gets, deletes, options atomic.Int32
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
	cloud.Mux.HandleFunc("DELETE /nova/v2.1/p/servers/server/os-volume_attachments/data", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		if r.Header.Get("OpenStack-API-Version") != "compute 2.89" || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.89" {
			t.Error(r.Header)
		}
		w.WriteHeader(204)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/nova/v2.1/p/"), sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/cinder/v3/p/"), sdk.WithLatestMicroversion(sdk.Compute), sdk.WithLatestMicroversion(sdk.BlockStorage))
	if err != nil {
		t.Fatal(err)
	}
	input := blockstorage.DetachVolumeRequest{Server: resource.ID("server"), Volume: resource.ID("data")}
	stopped := errors.New("option stopped")
	result, err := conn.DetachVolume(context.Background(), input, func(o *blockstorage.DetachVolumeOpts) error {
		options.Add(1)
		if novaDiscovery.Load() != 0 || cinderDiscovery.Load() != 0 {
			t.Fatal("discovery ran before options", novaDiscovery.Load(), cinderDiscovery.Load())
		}
		return stopped
	})
	if result != nil || !errors.Is(err, stopped) || gets.Load() != 0 || deletes.Load() != 0 || options.Load() != 1 {
		t.Fatal(result, err, gets.Load(), deletes.Load(), options.Load())
	}
	result, err = conn.DetachVolume(context.Background(), input)
	if err != nil || result == nil || result.Deleted == nil || gets.Load() != 1 || deletes.Load() != 1 || novaDiscovery.Load() != 1 || cinderDiscovery.Load() != 1 {
		t.Fatal(result, err, gets.Load(), deletes.Load(), novaDiscovery.Load(), cinderDiscovery.Load())
	}
	result, err = conn.DetachVolume(context.Background(), input)
	if err != nil || result == nil || result.Ready == nil || gets.Load() != 2 || deletes.Load() != 2 || novaDiscovery.Load() != 1 || cinderDiscovery.Load() != 1 {
		t.Fatal(result, err, gets.Load(), deletes.Load(), novaDiscovery.Load(), cinderDiscovery.Load())
	}
}

func TestConnectionDetachVolumeValidatesBeforeClientDiscovery(t *testing.T) {
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
	input := blockstorage.DetachVolumeRequest{Server: resource.ID("server"), Volume: resource.ID("data")}
	option := func(o *blockstorage.DetachVolumeOpts) error { callbacks.Add(1); return nil }
	cause := errors.New("caller canceled")
	canceled, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	var nilConn *sdk.Connection
	for _, tc := range []struct {
		name  string
		conn  *sdk.Connection
		ctx   context.Context
		input blockstorage.DetachVolumeRequest
		want  error
	}{
		{"nil-context", conn, nil, input, resource.ErrInvalidOption},
		{"nil-connection", nilConn, context.Background(), input, resource.ErrInvalidOption},
		{"canceled", conn, canceled, input, cause},
		{"missing-server", conn, context.Background(), blockstorage.DetachVolumeRequest{Volume: input.Volume}, resource.ErrInvalidOption},
		{"invalid-id", conn, context.Background(), blockstorage.DetachVolumeRequest{Server: input.Server, Volume: resource.ID("a/b")}, resource.ErrInvalidOption},
		{"invalid-utf8", conn, context.Background(), blockstorage.DetachVolumeRequest{Server: resource.Name(string([]byte{255})), Volume: input.Volume}, resource.ErrInvalidOption},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.conn.DetachVolume(tc.ctx, tc.input, option)
			if result != nil || !errors.Is(err, tc.want) {
				t.Fatal(result, err)
			}
			if tc.ctx == canceled && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
	negative := -time.Second
	result, err := conn.DetachVolume(context.Background(), input, blockstorage.WithDetachVolumeWait(false), blockstorage.WithDetachVolumeWaitPolicy(blockstorage.DetachVolumeWaitOpts{Timeout: &negative}))
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(result, err)
	}
	result, err = conn.DetachVolume(context.Background(), input, nil)
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(result, err)
	}
	if locates.Load() != 0 || callbacks.Load() != 0 {
		t.Fatal(locates.Load(), callbacks.Load())
	}
}

func TestConnectionDetachVolumeServiceInitializationCancellationCause(t *testing.T) {
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
			result, err := conn.DetachVolume(ctx, blockstorage.DetachVolumeRequest{Server: resource.ID("server"), Volume: resource.ID("data")}, func(*blockstorage.DetachVolumeOpts) error { callbacks.Add(1); return nil })
			if result != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || callbacks.Load() != 1 {
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

func TestConnectionDetachVolumeNoWaitNeedsOnlyNova(t *testing.T) {
	cloud := testcloud.New(t)
	var novaLocates, cinderLocates, deletes, callbacks atomic.Int32
	noCinder := errors.New("no Cinder endpoint in this cloud")
	cloud.Provider.EndpointLocator = func(o gophercloud.EndpointOpts) (string, error) {
		if o.Type == "compute" {
			novaLocates.Add(1)
			return cloud.Server.URL + "/nova/v2.1/p/", nil
		}
		cinderLocates.Add(1)
		return "", noCinder
	}
	cloud.Mux.HandleFunc("DELETE /nova/v2.1/p/servers/server/os-volume_attachments/data", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		w.WriteHeader(204)
	})
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		result, err := conn.DetachVolume(context.Background(), blockstorage.DetachVolumeRequest{Server: resource.ID("server"), Volume: resource.ID("data")}, blockstorage.WithDetachVolumeWait(false), func(*blockstorage.DetachVolumeOpts) error { callbacks.Add(1); return nil })
		if err != nil || result == nil || result.Deleted == nil || result.Deleted.StatusCode != 204 || result.LastAccepted != nil || result.Ready != nil {
			t.Fatal(result, err)
		}
	}
	if novaLocates.Load() != 1 || cinderLocates.Load() != 0 || deletes.Load() != 2 || callbacks.Load() != 2 {
		t.Fatal(novaLocates.Load(), cinderLocates.Load(), deletes.Load(), callbacks.Load())
	}

	result, err := conn.DetachVolume(context.Background(), blockstorage.DetachVolumeRequest{Server: resource.ID("server"), Volume: resource.Name("data")}, blockstorage.WithDetachVolumeWait(false))
	if result != nil || !errors.Is(err, noCinder) || novaLocates.Load() != 1 || cinderLocates.Load() != 1 || deletes.Load() != 2 {
		t.Fatal(result, err, novaLocates.Load(), cinderLocates.Load(), deletes.Load())
	}
}
