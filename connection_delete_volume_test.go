package gophercloudsdk_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestConnectionDeleteVolumeCachedCinderLiveTokenAndNoWaitStillLocates(t *testing.T) {
	cloud := testcloud.New(t)
	var locates, gets, deletes, callbacks atomic.Int32
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		return "", errors.New("unexpected service selection")
	}
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/cinder/v3/p/"))
	if err != nil {
		t.Fatal(err)
	}
	cinder, err := conn.BlockStorageV3(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	opaque := []byte{0, 0xff, 'x'}
	cloud.Mux.HandleFunc("GET /cinder/v3/p/volumes/volume", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if r.Header.Get("X-Auth-Token") != "live-token" || r.Header.Get("X-Entry") != "after-option" || r.URL.RawQuery != "" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"volume":{"id":"volume","status":null,"metadata":{"number":9007199254740993}}}`)
	})
	cloud.Mux.HandleFunc("DELETE /cinder/v3/p/volumes/volume", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) != 0 || r.URL.RawQuery != "cascade=false" || r.Header.Get("X-Auth-Token") != "live-token" || r.Header.Get("X-Entry") != "after-option" {
			t.Error(r.URL, r.Header, body, err)
		}
		w.Header().Set("X-Proof", "deleted")
		w.WriteHeader(202)
		_, _ = w.Write(opaque)
	})
	for range 2 {
		input := blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume")}
		result, err := conn.DeleteVolume(context.Background(), input, blockstorage.WithDeleteVolumeWait(false), func(*blockstorage.DeleteVolumeOpts) error {
			callbacks.Add(1)
			current, err := conn.BlockStorageV3(context.Background())
			if err != nil || current != cinder {
				t.Error(current, err)
				return errors.New("cached client changed")
			}
			input.Volume = resource.Name("changed by option")
			cinder.RawClient().MoreHeaders = map[string]string{"X-Entry": "after-option"}
			cloud.Provider.SetToken("live-token")
			return nil
		})
		if err != nil || result == nil || !result.Found || !result.Deleted || result.VolumeID != "volume" || result.Located == nil || result.Located.Volume == nil || result.Located.Volume.Status != nil || result.Deletion == nil || result.Deletion.StatusCode != 202 || !bytes.Equal(result.Deletion.Body, opaque) || result.Deletion.Header.Get("X-Proof") != "deleted" || result.Ready != nil || result.LastAccepted != nil || result.Absent != nil {
			t.Fatal(result, err)
		}
	}
	if gets.Load() != 2 || deletes.Load() != 2 || callbacks.Load() != 2 || locates.Load() != 0 {
		t.Fatal(gets.Load(), deletes.Load(), callbacks.Load(), locates.Load())
	}
}

func TestConnectionDeleteVolumeOptionsPrecedeDiscoveryAndReuseSelectedForceProtocol(t *testing.T) {
	cloud := testcloud.New(t)
	var discoveries, gets, deletes, callbacks atomic.Int32
	cloud.Mux.HandleFunc("GET /cinder/v3/{$}", func(w http.ResponseWriter, r *http.Request) {
		discoveries.Add(1)
		testcloud.JSON(w, 200, `{"version":{"id":"v3.0","min_version":"3.0","version":"3.70"}}`)
	})
	cloud.Mux.HandleFunc("GET /cinder/v3/p/volumes/volume", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if r.Header.Get("OpenStack-API-Version") != "volume 3.70" || r.URL.RawQuery != "" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"volume":{"id":"volume","status":"error_deleting"}}`)
	})
	cloud.Mux.HandleFunc("DELETE /cinder/v3/p/volumes/volume", func(w http.ResponseWriter, r *http.Request) {
		step := deletes.Add(1)
		query := "cascade=false&force=false"
		if step == 2 {
			query = "cascade=false&force=true"
		}
		if r.Header.Get("X-OpenStack-Volume-API-Version") != "3.70" || r.URL.RawQuery != query {
			t.Error(r.URL, r.Header)
		}
		w.WriteHeader(204)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/cinder/v3/p/"), sdk.WithLatestMicroversion(sdk.BlockStorage))
	if err != nil {
		t.Fatal(err)
	}
	rejected := errors.New("option rejected")
	input := blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume")}
	result, err := conn.DeleteVolume(context.Background(), input, func(*blockstorage.DeleteVolumeOpts) error {
		callbacks.Add(1)
		if discoveries.Load() != 0 {
			t.Error("discovery preceded options")
		}
		return rejected
	})
	if result != nil || !errors.Is(err, rejected) || discoveries.Load() != 0 || gets.Load() != 0 || deletes.Load() != 0 {
		t.Fatal(result, err, discoveries.Load())
	}
	for _, force := range []bool{false, true} {
		result, err := conn.DeleteVolume(context.Background(), input, blockstorage.WithDeleteVolumeWait(false), blockstorage.WithDeleteVolumeForce(force), func(*blockstorage.DeleteVolumeOpts) error { callbacks.Add(1); return nil })
		if err != nil || result == nil || !result.Found || !result.Deleted || result.Located == nil || result.Deletion == nil || result.Deletion.StatusCode != 204 || result.LastAccepted != nil {
			t.Fatal(result, err)
		}
	}
	if discoveries.Load() != 1 || gets.Load() != 2 || deletes.Load() != 2 || callbacks.Load() != 3 {
		t.Fatal(discoveries.Load(), gets.Load(), deletes.Load(), callbacks.Load())
	}
}

func TestConnectionDeleteVolumePreflightSelectsNoService(t *testing.T) {
	cloud := testcloud.New(t)
	var locates atomic.Int32
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		return "", errors.New("unexpected locator")
	}
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("caller canceled delete")
	canceled, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	for _, name := range []string{"nil context", "nil connection", "canceled", "unsafe reference"} {
		t.Run(name, func(t *testing.T) {
			ctx, current, input := context.Background(), conn, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume")}
			switch name {
			case "nil context":
				ctx = nil
			case "nil connection":
				current = nil
			case "canceled":
				ctx = canceled
			case "unsafe reference":
				input.Volume = resource.ID("../other")
			}
			calls := 0
			result, err := current.DeleteVolume(ctx, input, func(*blockstorage.DeleteVolumeOpts) error { calls++; return nil })
			if result != nil || err == nil || calls != 0 || locates.Load() != 0 {
				t.Fatal(result, err, calls, locates.Load())
			}
			if name == "canceled" {
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
	negative, zero := -time.Second, time.Duration(0)
	for _, invalid := range []blockstorage.DeleteVolumeOption{nil, blockstorage.WithDeleteVolumeWaitPolicy(blockstorage.DeleteVolumeWaitOpts{Timeout: &negative}), blockstorage.WithDeleteVolumeWaitPolicy(blockstorage.DeleteVolumeWaitOpts{PollInterval: &zero})} {
		calls := 0
		result, err := conn.DeleteVolume(context.Background(), blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume")}, func(*blockstorage.DeleteVolumeOpts) error { calls++; return nil }, blockstorage.WithDeleteVolumeWait(false), invalid)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || locates.Load() != 0 {
			t.Fatal(result, err, calls, locates.Load())
		}
	}
}

func TestConnectionDeleteVolumeInitializationRetainsCancellationCause(t *testing.T) {
	cloud := testcloud.New(t)
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("Cinder initialization canceled")
	var locates atomic.Int32
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) { locates.Add(1); cancel(cause); return "", cause }
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	result, err := conn.DeleteVolume(ctx, blockstorage.DeleteVolumeRequest{Volume: resource.ID("volume")}, func(*blockstorage.DeleteVolumeOpts) error { calls++; return nil })
	var operation *resource.OperationError
	if result != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.As(err, &operation) || operation.Operation != "DeleteVolume" || calls != 1 || locates.Load() != 1 {
		t.Fatal(result, err, calls, locates.Load())
	}
}
