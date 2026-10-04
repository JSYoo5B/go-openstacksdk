package gophercloudsdk_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestConnectionVolumeSnapshotByIDValidatesPathBeforeHTTPAndPreservesUnicodeProof(t *testing.T) {
	invalid := []struct{ name, id string }{
		{"empty", ""}, {"dot", "."}, {"parent", ".."},
		{"path", "snapshot/child"}, {"backslash", "snapshot\\child"},
		{"query", "snapshot?query"}, {"fragment", "snapshot#fragment"},
		{"escaped slash", "snapshot%2Fchild"}, {"escaped Unicode", "%EC%8A%A4"},
		{"ASCII space", "snapshot child"}, {"Unicode space", "snapshot\u00a0child"},
		{"Unicode line separator", "snapshot\u2028child"},
		{"NUL", "snapshot\x00child"}, {"DEL", "snapshot\x7fchild"},
		{"invalid UTF8", string([]byte{'i', 0xff})},
	}
	for _, entry := range []string{"direct", "Connection"} {
		for _, tc := range invalid {
			t.Run(entry+"/reject/"+tc.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				var originals, getters, requests atomic.Int32
				cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
					getters.Add(1)
					return cloud.Server.URL + snapshotConnectionPath, nil
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					t.Error("invalid literal ID reached HTTP", r.URL)
					w.WriteHeader(500)
				})
				conn, err := sdk.FromProvider(cloud.Provider)
				if err != nil {
					t.Fatal(err)
				}
				client := cloud.Client("block-storage", snapshotConnectionPath)
				option := func(*blockstorage.VolumeSnapshotReadOpts) error { originals.Add(1); return nil }
				var result *blockstorage.GetVolumeSnapshotResult
				request := blockstorage.GetVolumeSnapshotByIDRequest{ID: tc.id}
				if entry == "direct" {
					result, err = blockstorage.GetVolumeSnapshotByID(context.Background(), client, request, option)
				} else {
					result, err = conn.GetVolumeSnapshotByID(context.Background(), request, option)
				}
				if result != nil || !errors.Is(err, resource.ErrInvalidOption) || originals.Load() != 1 || getters.Load() != 0 || requests.Load() != 0 {
					t.Fatal(result, err, originals.Load(), getters.Load(), requests.Load())
				}
				snapshotConnectionOperation(t, "GetVolumeSnapshotByID", err)
				var physical *resource.ResponseError
				var native gophercloud.ErrUnexpectedResponseCode
				if errors.As(err, &physical) || errors.As(err, &native) {
					t.Fatal("local ID rejection invented HTTP proof", err)
				}
			})
		}
		t.Run(entry+"/valid Unicode member", func(t *testing.T) {
			cloud := testcloud.New(t)
			var originals, getters, requests atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				getters.Add(1)
				if originals.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 {
					t.Error(originals.Load(), opts)
				}
				return cloud.Server.URL + snapshotConnectionPath, nil
			}
			const id = "스냅샷-Δ-😀"
			const body = `{"snapshot":{"status":"available","unknown":9007199254740993}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				bodySent, readErr := io.ReadAll(r.Body)
				if r.Method != http.MethodGet || r.URL.Path != snapshotConnectionPath+"snapshots/"+id || r.URL.EscapedPath() != snapshotConnectionPath+"snapshots/"+url.PathEscape(id) || r.URL.RawQuery != "" || len(bodySent) != 0 || readErr != nil || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.60" {
					t.Error("Unicode ID lost fixed member semantics", r.URL, r.Header, string(bodySent), readErr)
				}
				w.Header().Set("X-ID-Proof", "actual Unicode response")
				testcloud.JSON(w, 203, body)
			})
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.60"))
			if err != nil {
				t.Fatal(err)
			}
			client := cloud.Client("block-storage", snapshotConnectionPath)
			client.Microversion = "3.60"
			option := func(*blockstorage.VolumeSnapshotReadOpts) error { originals.Add(1); return nil }
			var result *blockstorage.GetVolumeSnapshotResult
			request := blockstorage.GetVolumeSnapshotByIDRequest{ID: id}
			if entry == "direct" {
				result, err = blockstorage.GetVolumeSnapshotByID(context.Background(), client, request, option)
			} else {
				result, err = conn.GetVolumeSnapshotByID(context.Background(), request, option)
			}
			wantGetters := int32(0)
			if entry == "Connection" {
				wantGetters = 1
			}
			if err != nil || result == nil || result.Snapshot == nil || result.Observed == nil || result.Observed.StatusCode != 203 || result.Observed.Header.Get("X-ID-Proof") != "actual Unicode response" || string(result.Observed.Body) != body || len(result.Pages) != 0 || result.RequestedID != id || !result.SeededID || originals.Load() != 1 || getters.Load() != wantGetters || requests.Load() != 1 {
				t.Fatal(result, err, originals.Load(), getters.Load(), requests.Load())
			}
			logical := snapshotConnectionFields(t, result.Value)
			seed, _ := json.Marshal(id)
			if len(logical) != 16 || !bytes.Equal(logical["id"], seed) || string(logical["status"]) != `"available"` || result.Snapshot.StatusCode != 203 || result.Snapshot.Header.Get("X-ID-Proof") != "actual Unicode response" || string(result.Snapshot.Body["unknown"]) != "9007199254740993" {
				t.Fatal(string(result.Value), result.Snapshot)
			}
			if _, hasID := result.Snapshot.Body["id"]; hasID {
				t.Fatal("logical seed was fabricated in physical body", result.Snapshot)
			}
			physicalBody := bytes.Clone(result.Observed.Body)
			result.Snapshot.Body["unknown"][0] = '!'
			result.Snapshot.Header.Set("X-ID-Proof", "caller changed raw resource")
			result.Value[0] = '!'
			if !bytes.Equal(result.Observed.Body, physicalBody) || result.Observed.Header.Get("X-ID-Proof") != "actual Unicode response" {
				t.Fatal("Unicode logical/raw result borrows physical proof", result.Observed)
			}
		})
	}
}
