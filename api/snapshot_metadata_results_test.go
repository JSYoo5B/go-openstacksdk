package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	snapshots2 "gophercloudsdk/blockstorage/v2/snapshots"
	snapshots3 "gophercloudsdk/blockstorage/v3/snapshots"
	"gophercloudsdk/internal/snapshotmetadata"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

type snapshotMetadataCall func(context.Context, *gophercloud.ServiceClient, string) (map[string]any, error)

type snapshotMetadataFixture struct {
	version           string
	prepare           func(map[string]any, any) snapshotMetadataCall
	protectedMetadata snapshotMetadataCall
	getID             func(context.Context, *gophercloud.ServiceClient, string) (string, error)
}

func snapshotMetadataFixtures() []snapshotMetadataFixture {
	return []snapshotMetadataFixture{
		{version: "v2", prepare: func(metadata map[string]any, vendor any) snapshotMetadataCall {
			options := []snapshots2.UpdateMetadataOption{snapshots2.WithUpdateMetadataOptions(snapshots2.UpdateMetadataOpts{Metadata: metadata})}
			if vendor != nil {
				options = append(options, snapshots2.WithUpdateMetadataField("vendor", vendor))
			}
			return func(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
				return snapshots2.New(client).UpdateMetadata(ctx, id, snapshots2.UpdateMetadataOpts{}, options...)
			}
		}, protectedMetadata: func(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
			return snapshots2.New(client).UpdateMetadata(ctx, id, snapshots2.UpdateMetadataOpts{}, snapshots2.WithUpdateMetadataField("metadata", map[string]any{}))
		}, getID: func(ctx context.Context, client *gophercloud.ServiceClient, id string) (string, error) {
			value, err := snapshots2.New(client).Get(ctx, id)
			if value == nil {
				return "", err
			}
			return value.ID, err
		}},
		{version: "v3", prepare: func(metadata map[string]any, vendor any) snapshotMetadataCall {
			options := []snapshots3.UpdateMetadataOption{snapshots3.WithUpdateMetadataOptions(snapshots3.UpdateMetadataOpts{Metadata: metadata})}
			if vendor != nil {
				options = append(options, snapshots3.WithUpdateMetadataField("vendor", vendor))
			}
			return func(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
				return snapshots3.New(client).UpdateMetadata(ctx, id, snapshots3.UpdateMetadataOpts{}, options...)
			}
		}, protectedMetadata: func(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
			return snapshots3.New(client).UpdateMetadata(ctx, id, snapshots3.UpdateMetadataOpts{}, snapshots3.WithUpdateMetadataField("metadata", map[string]any{}))
		}, getID: func(ctx context.Context, client *gophercloud.ServiceClient, id string) (string, error) {
			value, err := snapshots3.New(client).Get(ctx, id)
			if value == nil {
				return "", err
			}
			return value.ID, err
		}},
	}
}

func snapshotMetadataPrefix(f snapshotMetadataFixture) string {
	return "/cinder/" + f.version + "/project/"
}

func TestSnapshotMetadataResultsReturnOnlyMetadataAndExactNativeNumbers(t *testing.T) {
	for _, f := range snapshotMetadataFixtures() {
		t.Run(f.version, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("PUT "+snapshotMetadataPrefix(f)+"snapshots/selected/metadata", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !reflect.DeepEqual(body, map[string]any{"metadata": map[string]any{"owner": "team"}}) || len(r.URL.Query()) != 0 {
					t.Error(body, err, r.URL)
				}
				testcloud.JSON(w, 200, `{"metadata":{"owner":"server-team","large":9007199254740993,"fraction":1.2300,"nullable":null,"nested":{"items":[false,9007199254740993,null]}},"snapshot":{"id":"unrelated"}}`)
			})
			value, err := f.prepare(map[string]any{"owner": "team"}, nil)(context.Background(), cloud.Client("block-storage", snapshotMetadataPrefix(f)), "selected")
			want := map[string]any{"owner": "server-team", "large": json.Number("9007199254740993"), "fraction": json.Number("1.2300"), "nullable": nil, "nested": map[string]any{"items": []any{false, json.Number("9007199254740993"), nil}}}
			if err != nil || !reflect.DeepEqual(value, want) || calls.Load() != 1 {
				t.Fatal("metadata projection or native number precision changed", value, err, calls.Load())
			}
		})
	}
}

func TestSnapshotMetadataResultsKeepTypedMapOwnershipAndFrozenExtensions(t *testing.T) {
	for _, f := range snapshotMetadataFixtures() {
		t.Run(f.version, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("PUT "+snapshotMetadataPrefix(f)+"snapshots/selected/metadata", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				var body map[string]any
				want := map[string]any{"metadata": map[string]any{"owner": "mutated", "vendor": map[string]any{"enabled": false}}}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !reflect.DeepEqual(body, want) {
					t.Error("typed map ownership or snapshotted extension changed", body, err)
				}
				testcloud.JSON(w, 200, `{"metadata":{"owner":"returned"}}`)
			})
			metadata, vendor := map[string]any{"owner": "frozen"}, map[string]any{"enabled": false}
			call := f.prepare(metadata, vendor)
			// Typed option structs retain caller-owned maps. Extension values are
			// JSON snapshots. Mutate before calls, never during concurrent reuse.
			metadata["owner"], vendor["enabled"] = "mutated", true
			client := cloud.Client("block-storage", snapshotMetadataPrefix(f))
			var wait sync.WaitGroup
			for range 4 {
				wait.Add(1)
				go func() {
					defer wait.Done()
					value, err := call(context.Background(), client, "selected")
					if err != nil || value["owner"] != "returned" {
						t.Error(value, err)
					}
				}()
			}
			wait.Wait()
			if calls.Load() != 4 || metadata["owner"] != "mutated" || vendor["enabled"] != true {
				t.Fatal("request ownership changed", calls.Load(), metadata, vendor)
			}
		})
	}
}

func TestSnapshotMetadataResultsRejectAcceptedWrongEnvelopesWithoutReplay(t *testing.T) {
	for _, f := range snapshotMetadataFixtures() {
		for _, body := range []string{`null`, `[]`, `"root"`, `{}`, `{"Metadata":{}}`, `{"snapshot":{"id":"wrong"}}`, `{"metadata":null}`, `{"metadata":[]}`, `{"metadata":"wrong"}`, `{"metadata":false}`, `{"metadata":23}`} {
			t.Run(f.version+"/"+body, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Method != "PUT" || r.URL.Path != snapshotMetadataPrefix(f)+"snapshots/selected/metadata" {
						t.Error("accepted mutation replayed on another route", r.Method, r.URL)
					}
					testcloud.JSON(w, 200, body)
				})
				value, err := f.prepare(map[string]any{"owner": "team"}, nil)(context.Background(), cloud.Client("block-storage", snapshotMetadataPrefix(f)), "selected")
				var envelope *snapshotmetadata.EnvelopeError
				if value != nil || !errors.As(err, &envelope) || calls.Load() != 1 {
					t.Fatal("wrong accepted shape must return an error once", value, err, calls.Load())
				}
			})
		}
	}
}

func TestSnapshotMetadataResultsPreserveNativeFailuresAndPreflight(t *testing.T) {
	for _, f := range snapshotMetadataFixtures() {
		t.Run(f.version, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				switch r.URL.Path {
				case snapshotMetadataPrefix(f) + "snapshots/bad-status/metadata":
					testcloud.JSON(w, 400, `{"metadata":{"owner":"must-not-return"}}`)
				case snapshotMetadataPrefix(f) + "snapshots/rejected-201/metadata":
					testcloud.JSON(w, 201, `{"metadata":{"owner":"must-not-return"}}`)
				default:
					testcloud.JSON(w, 200, `{`)
				}
			})
			client := cloud.Client("block-storage", snapshotMetadataPrefix(f))
			call := f.prepare(map[string]any{"owner": "team"}, nil)
			for _, status := range []int{400, 201} {
				id := "bad-status"
				if status == 201 {
					id = "rejected-201"
				}
				value, err := call(context.Background(), client, id)
				var envelope *snapshotmetadata.EnvelopeError
				if value != nil || !gophercloud.ResponseCodeIs(err, status) || errors.As(err, &envelope) {
					t.Fatal("native status cause was replaced", value, err)
				}
			}
			if value, err := call(context.Background(), client, "broken-json"); value != nil || !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatal("native JSON read cause changed", value, err)
			}
			before := calls.Load()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if value, err := call(ctx, client, "selected"); value != nil || !errors.Is(err, context.Canceled) || calls.Load() != before {
				t.Fatal("canceled mutation reached server", value, err, calls.Load())
			}
			if value, err := f.prepare(nil, make(chan int))(context.Background(), client, "selected"); value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != before {
				t.Fatal("invalid extension reached server", value, err, calls.Load())
			}
		})
	}
}

func TestSnapshotMetadataResultsKeepLiveNativeClientAndSnapshotGet(t *testing.T) {
	for _, f := range snapshotMetadataFixtures() {
		t.Run(f.version, func(t *testing.T) {
			cloud := testcloud.New(t)
			var mutations, gets atomic.Int32
			prefix := "/reverse/" + f.version + "/project/"
			client := cloud.Client("block-storage", "/catalog/unused/")
			client.ResourceBase = cloud.Server.URL + prefix
			client.MoreHeaders = map[string]string{"X-SDK-Source": "native"}
			if f.version == "v3" {
				client.Microversion = "3.62"
			}
			cloud.Mux.HandleFunc("PUT "+prefix+"snapshots/selected/metadata", func(w http.ResponseWriter, r *http.Request) {
				call := mutations.Add(1)
				wantToken := fmt.Sprintf("token-%d", call)
				if r.Header.Get("X-Auth-Token") != wantToken || r.Header.Get("X-SDK-Source") != "native" || r.Header.Get("X-OpenStack-Volume-API-Version") != client.Microversion {
					t.Error("client source/header/token changed", r.Header)
				}
				testcloud.JSON(w, 200, `{"metadata":{"owner":"returned"}}`)
			})
			cloud.Mux.HandleFunc("GET "+prefix+"snapshots/selected", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 200, `{"snapshot":{"id":"selected","metadata":{"owner":"snapshot-model"}}}`)
			})
			call := f.prepare(map[string]any{"owner": "requested"}, nil)
			for i := 1; i <= 2; i++ {
				cloud.Provider.SetToken(fmt.Sprintf("token-%d", i))
				if value, err := call(context.Background(), client, "selected"); err != nil || value["owner"] != "returned" || gets.Load() != 0 {
					t.Fatal("metadata mutation performed extra Get", value, err, gets.Load())
				}
			}
			if id, err := f.getID(context.Background(), client, "selected"); err != nil || id != "selected" || gets.Load() != 1 || mutations.Load() != 2 || client.ProviderClient != cloud.Provider || client.MoreHeaders["X-SDK-Source"] != "native" {
				t.Fatal("Snapshot getter or shared native client changed", id, err, gets.Load(), mutations.Load())
			}
		})
	}
}

func TestSnapshotMetadataResultsKeepOmissionAndCoreFieldProtection(t *testing.T) {
	for _, f := range snapshotMetadataFixtures() {
		t.Run(f.version, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("PUT "+snapshotMetadataPrefix(f)+"snapshots/selected/metadata", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				var body map[string]any
				want := map[string]any{}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !reflect.DeepEqual(body, want) {
					t.Error("native omitempty or explicit extension changed", body, err)
				}
				testcloud.JSON(w, 200, `{"metadata":{},"Metadata":{"wrong-case":"ignored"}}`)
			})
			client := cloud.Client("block-storage", snapshotMetadataPrefix(f))
			for _, call := range []snapshotMetadataCall{f.prepare(nil, nil), f.prepare(map[string]any{}, nil)} {
				value, err := call(context.Background(), client, "selected")
				if err != nil || value == nil || len(value) != 0 {
					t.Fatal("valid empty metadata object changed", value, err)
				}
			}
			if value, err := f.protectedMetadata(context.Background(), client, "selected"); value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 2 {
				t.Fatal("core metadata field bypassed existing extension protection", value, err, calls.Load())
			}
			if calls.Load() != 2 {
				t.Fatal(calls.Load())
			}
		})
	}
}
