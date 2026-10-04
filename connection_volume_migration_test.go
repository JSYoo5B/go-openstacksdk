package gophercloudsdk_test

import (
	"bytes"
	"context"
	"encoding/json"
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

const vcmBase = "/proxy/cinder/v3/migration-project/"

func vcmContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func vcmWire(t *testing.T, req *http.Request, body, version string) {
	t.Helper()
	raw, err := io.ReadAll(req.Body)
	if err != nil || string(raw) != body || req.Method != http.MethodPost || req.URL.EscapedPath() != vcmBase+"volumes/literal-name/action" || req.URL.RawQuery != "" || req.Header.Get("X-Auth-Token") != "test-token" || req.Header.Get("OpenStack-API-Version") != "volume "+version || req.Header.Get("X-OpenStack-Volume-API-Version") != version || req.ContentLength != int64(len(body)) || len(req.TransferEncoding) != 0 {
		t.Error(req.Method, req.URL, string(raw), body, req.Header, err)
	}
}

func TestConnectionVolumeMigrationUsesCachedCinderAndSelectedVersionStillRequiresDiscovery(t *testing.T) {
	cloud := testcloud.New(t)
	var getters, gets, posts atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		getters.Add(1)
		if opts.Type != "block-storage" || opts.Version != 3 {
			t.Error(opts)
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
	bodies := []string{`{"os-reset_status":{}}`, `{"os-migrate_volume":{"cluster":""}}`, `{"os-migrate_volume_completion":{"error":false,"new_volume":"new/name ?#\n\u0000"}}`}
	raw := []byte{0xff, 0x00, '{'}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			gets.Add(1)
			if req.URL.Path != "/proxy/cinder/v3/" || req.URL.RawQuery != "" || req.Header.Get("OpenStack-API-Version") != "" || req.Header.Get("X-OpenStack-Volume-API-Version") != "" || req.Header.Get("X-Source") != "original" || req.Header.Get("X-Auth-Token") != "test-token" || req.ContentLength != 0 {
				t.Error("selected action leaked version to discovery", req.URL, req.Header, req.ContentLength)
			}
			w.Header().Set("X-Stage", "gate")
			testcloud.JSON(w, 300, `{"id":"v3.0","min_version":"3.0","max_version":"3.20"}`)
			return
		}
		n := posts.Add(1)
		if n > int32(len(bodies)) {
			t.Error("extra lookup or wait", req.URL)
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
		func() (*blockstorage.VolumeActionResult, error) { return conn.ResetVolumeStatus(vcmContext(t), input) },
		func() (*blockstorage.VolumeActionResult, error) {
			return conn.MigrateVolume(vcmContext(t), input, blockstorage.WithVolumeMigrationCluster(""))
		},
		func() (*blockstorage.VolumeActionResult, error) {
			return conn.CompleteVolumeMigration(vcmContext(t), input, "new/name ?#\n\x00")
		},
	}
	for index, call := range calls {
		result, err := call()
		wantDiscovery := 0
		if index == 1 {
			wantDiscovery = 1
		}
		if err != nil || result == nil || !result.Completed || result.VolumeID != input.VolumeID || result.Microversion != "3.90" || len(result.Discovery) != wantDiscovery || result.Applied == nil || result.Applied.StatusCode != 203 || !bytes.Equal(result.Applied.Body, raw) || result.Applied.Header.Get("X-Stage") != "action" {
			t.Fatal(index, result, err)
		}
		if wantDiscovery == 1 && result.Discovery[0].Header.Get("X-Stage") != "gate" {
			t.Fatal("lost required-support evidence", result)
		}
	}
	if getters.Load() != 1 || gets.Load() != 1 || posts.Load() != 3 || client.Microversion != "3.90" || client.MoreHeaders["OpenStack-API-Version"] != "volume 3.90" || client.MoreHeaders["X-OpenStack-Volume-API-Version"] != "3.90" {
		t.Fatal(getters.Load(), gets.Load(), posts.Load(), client)
	}
}

func TestConnectionVolumeMigrationOriginalsRunOnceBeforeGetterAndRetainedValuesStayDetached(t *testing.T) {
	for _, kind := range []string{"reset", "migration", "completion"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			var originals, getters, posts atomic.Int32
			var retainedString *string
			var retainedFlag *bool
			cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
				getters.Add(1)
				if originals.Load() != 1 {
					t.Error("getter ran before original", originals.Load())
				}
				if retainedString != nil {
					*retainedString = "getter changed caller state"
				}
				if retainedFlag != nil {
					*retainedFlag = false
				}
				return cloud.Server.URL + vcmBase, nil
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.90"))
			if err != nil {
				t.Fatal(err)
			}
			body := map[string]string{"reset": `{"os-reset_status":{"status":"literal"}}`, "migration": `{"os-migrate_volume":{"force_host_copy":true,"host":"literal"}}`, "completion": `{"os-migrate_volume_completion":{"error":true,"new_volume":""}}`}[kind]
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				posts.Add(1)
				vcmWire(t, req, body, "3.90")
				w.WriteHeader(204)
			})
			input := blockstorage.VolumeActionRequest{VolumeID: "literal-name"}
			var result *blockstorage.VolumeActionResult
			switch kind {
			case "reset":
				result, err = conn.ResetVolumeStatus(vcmContext(t), input, func(next *blockstorage.VolumeStatusResetOpts) error {
					originals.Add(1)
					value := "literal"
					next.Status = &value
					retainedString = next.Status
					return nil
				})
			case "migration":
				result, err = conn.MigrateVolume(vcmContext(t), input, func(next *blockstorage.VolumeMigrationOpts) error {
					originals.Add(1)
					value, yes := "literal", true
					next.Host, next.ForceHostCopy = &value, &yes
					retainedString, retainedFlag = next.Host, next.ForceHostCopy
					return nil
				})
			case "completion":
				result, err = conn.CompleteVolumeMigration(vcmContext(t), input, "", func(next *blockstorage.VolumeMigrationCompletionOpts) error {
					originals.Add(1)
					yes := true
					next.Error = &yes
					retainedFlag = next.Error
					return nil
				})
			}
			if err != nil || result == nil || !result.Completed || originals.Load() != 1 || getters.Load() != 1 || posts.Load() != 1 {
				t.Fatal(result, err, originals.Load(), getters.Load(), posts.Load())
			}
		})
	}
}

func TestConnectionVolumeMigrationPreflightAndGetterFailuresRetainCauses(t *testing.T) {
	for _, kind := range []string{"nil option", "original error", "invalid body", "nil connection", "unsafe ID", "getter canceled"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx, cancel := context.WithCancelCause(vcmContext(t))
			defer cancel(nil)
			cause := errors.New("migration caller or getter failed")
			var getters, posts, later atomic.Int32
			cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
				getters.Add(1)
				cancel(cause)
				return "", cause
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) { posts.Add(1); w.WriteHeader(500) })
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			operation, want, expectedGetters := "MigrateVolume", resource.ErrInvalidOption, int32(0)
			input := blockstorage.VolumeActionRequest{VolumeID: "literal-name"}
			var result *blockstorage.VolumeActionResult
			switch kind {
			case "nil option":
				result, err = conn.MigrateVolume(ctx, input, nil)
			case "original error":
				want = cause
				result, err = conn.MigrateVolume(ctx, input, func(*blockstorage.VolumeMigrationOpts) error { return cause }, func(*blockstorage.VolumeMigrationOpts) error { later.Add(1); return nil })
			case "invalid body":
				operation = "CompleteVolumeMigration"
				result, err = conn.CompleteVolumeMigration(ctx, input, string([]byte{0xff}))
			case "nil connection":
				conn = nil
				result, err = conn.MigrateVolume(ctx, input)
			case "unsafe ID":
				input.VolumeID = "a/b"
				result, err = conn.MigrateVolume(ctx, input)
			case "getter canceled":
				want, expectedGetters = cause, 1
				result, err = conn.MigrateVolume(ctx, input)
			}
			var proof *resource.ResponseError
			if result != nil || !errors.Is(err, want) || errors.As(err, &proof) || getters.Load() != expectedGetters || posts.Load() != 0 || later.Load() != 0 {
				t.Fatal(result, err, proof, getters.Load(), posts.Load(), later.Load())
			}
			vscError(t, operation, err)
			if kind == "getter canceled" && !errors.Is(err, context.Canceled) {
				t.Fatal("lost getter context cause", err)
			}
		})
	}
}
