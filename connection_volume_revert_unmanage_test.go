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

func TestConnectionVolumeRevertUnmanageUsesCachedCinderAndLiteralSnapshotBody(t *testing.T) {
	cloud := testcloud.New(t)
	var getters, gets, posts atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		getters.Add(1)
		if opts.Type != "block-storage" || opts.Version != 3 {
			t.Error("unexpected service lookup", opts)
		}
		return cloud.Server.URL + vcmBase, nil
	}
	bad := string([]byte{0xff})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.90"), sdk.WithCloudLocation(resource.CloudLocation{Cloud: &bad, Zone: json.RawMessage("invalid")}))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.BlockStorageV3(vcmContext(t))
	if err != nil {
		t.Fatal(err)
	}
	client := service.RawClient()
	client.MoreHeaders = map[string]string{"OpenStack-API-Version": "volume 3.90", "X-OpenStack-Volume-API-Version": "3.90", "X-Source": "original"}
	bodies := []string{`{"revert":{"snapshot_id":"snapshot /?%#\n\u0000"}}`, `{"os-unmanage":null}`}
	raw := []byte{0xff, 0x00, '}'}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			gets.Add(1)
			if req.URL.Path != "/proxy/cinder/v3/" || req.URL.RawQuery != "" || req.Header.Get("OpenStack-API-Version") != "" || req.Header.Get("X-OpenStack-Volume-API-Version") != "" || req.Header.Get("X-Source") != "original" || req.Header.Get("X-Auth-Token") != "test-token" || req.ContentLength != 0 {
				t.Error("required-support probe changed the cached request policy", req.URL, req.Header)
			}
			w.Header().Set("X-Stage", "required 3.40")
			testcloud.JSON(w, 300, `{"id":"v3.0","min_version":"3.0","max_version":"3.50"}`)
			return
		}
		n := posts.Add(1)
		if n > int32(len(bodies)) {
			t.Error("unexpected lookup, state poll or extra action", req.URL)
			w.WriteHeader(500)
			return
		}
		vcmWire(t, req, bodies[n-1], "3.90")
		w.Header().Set("X-Stage", "action")
		w.WriteHeader(203)
		_, _ = w.Write(raw)
	})
	input := blockstorage.VolumeActionRequest{VolumeID: "literal-name"}
	calls := []func() (*blockstorage.VolumeActionResult, error){
		func() (*blockstorage.VolumeActionResult, error) {
			return conn.RevertVolumeToSnapshot(vcmContext(t), input, "snapshot /?%#\n\x00")
		},
		func() (*blockstorage.VolumeActionResult, error) { return conn.UnmanageVolume(vcmContext(t), input) },
	}
	for index, call := range calls {
		result, err := call()
		wantDiscovery := 1 - index
		if err != nil || result == nil || !result.Completed || result.VolumeID != input.VolumeID || result.Microversion != "3.90" || len(result.Discovery) != wantDiscovery || result.Applied == nil || result.Applied.StatusCode != 203 || !bytes.Equal(result.Applied.Body, raw) || result.Applied.Header.Get("X-Stage") != "action" {
			t.Fatal(index, result, err)
		}
		if wantDiscovery == 1 && result.Discovery[0].Header.Get("X-Stage") != "required 3.40" {
			t.Fatal("lost actual support proof", result)
		}
	}
	if getters.Load() != 1 || gets.Load() != 1 || posts.Load() != 2 || client.Microversion != "3.90" || client.MoreHeaders["OpenStack-API-Version"] != "volume 3.90" || client.MoreHeaders["X-OpenStack-Volume-API-Version"] != "3.90" {
		t.Fatal(getters.Load(), gets.Load(), posts.Load(), client)
	}
}

func TestConnectionVolumeRevertUnmanagePreflightAndGetterFaultsKeepCauses(t *testing.T) {
	for _, revert := range []bool{false, true} {
		operation := "UnmanageVolume"
		if revert {
			operation = "RevertVolumeToSnapshot"
		}
		for _, kind := range []string{"nil context", "canceled context", "nil connection", "unsafe ID", "invalid snapshot", "getter error", "getter canceled"} {
			if !revert && kind == "invalid snapshot" {
				continue
			}
			t.Run(operation+" "+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancelCause(vcmContext(t))
				defer cancel(nil)
				cause := errors.New("revert/unmanage caller or getter failed")
				var getters, httpCalls atomic.Int32
				cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
					getters.Add(1)
					if kind == "getter canceled" {
						cancel(cause)
					}
					return "", cause
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) { httpCalls.Add(1); w.WriteHeader(500) })
				conn, err := sdk.FromProvider(cloud.Provider)
				if err != nil {
					t.Fatal(err)
				}
				input, snapshot := blockstorage.VolumeActionRequest{VolumeID: "literal-name"}, "snapshot"
				want, expectedGetters := resource.ErrInvalidOption, int32(0)
				var callContext context.Context = ctx
				switch kind {
				case "nil context":
					callContext = nil
				case "canceled context":
					cancel(cause)
					want = context.Canceled
				case "nil connection":
					conn = nil
				case "unsafe ID":
					input.VolumeID = "volume/path"
				case "invalid snapshot":
					snapshot = string([]byte{0xff})
				case "getter error", "getter canceled":
					want, expectedGetters = cause, 1
				}
				var result *blockstorage.VolumeActionResult
				if revert {
					result, err = conn.RevertVolumeToSnapshot(callContext, input, snapshot)
				} else {
					result, err = conn.UnmanageVolume(callContext, input)
				}
				var proof *resource.ResponseError
				if result != nil || !errors.Is(err, want) || errors.As(err, &proof) || getters.Load() != expectedGetters || httpCalls.Load() != 0 {
					t.Fatal(result, err, proof, getters.Load(), httpCalls.Load())
				}
				vscError(t, operation, err)
				if (kind == "getter canceled" || kind == "canceled context") && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
					t.Fatal("lost custom cancellation cause", err)
				}
			})
		}
	}
}
