package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

const vscBase = "/state-connection/v3/project/"

type vscOperation struct {
	name, key string
	call      func(*sdk.Connection, context.Context, blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error)
}

var vscOperations = []vscOperation{
	{"ReserveVolume", "os-reserve", (*sdk.Connection).ReserveVolume},
	{"UnreserveVolume", "os-unreserve", (*sdk.Connection).UnreserveVolume},
	{"BeginVolumeDetaching", "os-begin_detaching", (*sdk.Connection).BeginVolumeDetaching},
	{"AbortVolumeDetaching", "os-roll_detaching", (*sdk.Connection).AbortVolumeDetaching},
}

func vscCall(ctx context.Context, conn *sdk.Connection, input blockstorage.VolumeActionRequest, op vscOperation) (*blockstorage.VolumeActionResult, error) {
	return op.call(conn, ctx, input)
}
func vscError(t *testing.T, name string, err error) {
	t.Helper()
	var op *resource.OperationError
	if !errors.As(err, &op) || op.Operation != name || op.Resource != "volumes" {
		t.Fatal("wrong outer operation", op, err)
	}
}
func vscWire(t *testing.T, r *http.Request, key, version, token string) {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	modern := ""
	if version != "" {
		modern = "volume " + version
	}
	var fields map[string]json.RawMessage
	decode := json.Unmarshal(body, &fields)
	if err != nil || decode != nil || len(fields) != 1 || string(fields[key]) != "null" || r.Method != http.MethodPost || r.URL.Path != vscBase+"volumes/literal-name/action" || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != token || r.Header.Get("OpenStack-API-Version") != modern || r.Header.Get("X-OpenStack-Volume-API-Version") != version {
		t.Error(r.Method, r.URL, string(body), r.Header, err, decode)
	}
}
func TestConnectionVolumeStateActionsUseOnlyCachedCinderAndIgnoreLocation(t *testing.T) {
	cloud := testcloud.New(t)
	var locates, posts atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		if opts.Type != "block-storage" || opts.Version != 3 {
			t.Error(opts)
		}
		return cloud.Server.URL + vscBase, nil
	}
	bad := string([]byte{0xff})
	location := resource.CloudLocation{Cloud: &bad, Zone: json.RawMessage("not-json"), Project: resource.CloudProject{ID: json.RawMessage("not-json")}}
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.80"), sdk.WithCloudLocation(location))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.BlockStorageV3(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	service.RawClient().MoreHeaders = map[string]string{"X-State-Caller": "captured"}
	cloud.Provider.TokenID = "later-token"
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		n := int(posts.Add(1)) - 1
		if n >= len(vscOperations) {
			t.Error("unexpected HTTP", r.URL)
			w.WriteHeader(500)
			return
		}
		vscWire(t, r, vscOperations[n].key, "3.80", "later-token")
		if r.Header.Get("X-State-Caller") != "captured" {
			t.Error(r.Header)
		}
		w.Header().Set("X-State-Proof", vscOperations[n].name)
		w.WriteHeader(203)
		_, _ = w.Write([]byte{0xff, '{'})
	})
	for _, op := range vscOperations {
		result, err := vscCall(context.Background(), conn, blockstorage.VolumeActionRequest{VolumeID: "literal-name"}, op)
		if err != nil || result == nil || !result.Completed || result.VolumeID != "literal-name" || result.Microversion != "3.80" || len(result.Discovery) != 0 || result.Applied == nil || result.Applied.StatusCode != 203 || len(result.Applied.Body) != 2 || result.Applied.Body[0] != 0xff || result.Applied.Header.Get("X-State-Proof") != op.name {
			t.Fatal(op.name, result, err)
		}
	}
	if locates.Load() != 1 || posts.Load() != 4 || service.RawClient().Microversion != "3.80" {
		t.Fatal(locates.Load(), posts.Load(), service.RawClient())
	}
}
func TestConnectionVolumeStateActionsPreflightNeverSelectsCinder(t *testing.T) {
	for _, op := range vscOperations {
		for _, kind := range []string{"nil context", "nil connection", "zero connection", "empty ID", "unsafe ID", "invalid UTF8", "cancel"} {
			t.Run(op.name+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) { calls.Add(1); return cloud.Server.URL + vscBase, nil }
				conn, err := sdk.FromProvider(cloud.Provider)
				if err != nil {
					t.Fatal(err)
				}
				ctx := context.Background()
				id := "literal-name"
				cause := errors.New("state action parent")
				want := resource.ErrInvalidOption
				switch kind {
				case "nil context":
					ctx = nil
				case "nil connection":
					conn = nil
				case "zero connection":
					conn = &sdk.Connection{}
				case "empty ID":
					id = ""
				case "unsafe ID":
					id = "a/b"
				case "invalid UTF8":
					id = string([]byte{0xff})
				case "cancel":
					child, cancel := context.WithCancelCause(ctx)
					cancel(cause)
					ctx = child
					want = context.Canceled
				}
				result, err := vscCall(ctx, conn, blockstorage.VolumeActionRequest{VolumeID: id}, op)
				var physical *resource.ResponseError
				if result != nil || !errors.Is(err, want) || errors.As(err, &physical) || calls.Load() != 0 {
					t.Fatal(result, err, physical, calls.Load())
				}
				if kind == "cancel" && !errors.Is(err, cause) {
					t.Fatal(err)
				}
				vscError(t, op.name, err)
			})
		}
	}
}
func TestConnectionVolumeStateActionsGetterErrorsKeepOperationAndCancellation(t *testing.T) {
	for _, op := range vscOperations {
		for _, kind := range []string{"error", "cancel with error", "cancel after successful getter"} {
			t.Run(op.name+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				var selects, wire atomic.Int32
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("state getter cancellation")
				getter := errors.New("Cinder unavailable")
				cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
					selects.Add(1)
					if kind != "error" {
						cancel(cause)
					}
					if kind == "cancel after successful getter" {
						return cloud.Server.URL + vscBase, nil
					}
					return "", getter
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { wire.Add(1); w.WriteHeader(500) })
				conn, err := sdk.FromProvider(cloud.Provider)
				if err != nil {
					t.Fatal(err)
				}
				result, err := vscCall(ctx, conn, blockstorage.VolumeActionRequest{VolumeID: "literal-name"}, op)
				var physical *resource.ResponseError
				if result != nil || err == nil || errors.As(err, &physical) || selects.Load() != 1 || wire.Load() != 0 {
					t.Fatal(result, err, physical, selects.Load(), wire.Load())
				}
				if kind != "cancel after successful getter" && !errors.Is(err, getter) {
					t.Fatal(err)
				}
				if kind != "error" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
					t.Fatal(err)
				}
				vscError(t, op.name, err)
			})
		}
	}
}
func TestConnectionVolumeStateActionsOwnDiscoveryPerCallAndKeepStageProof(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		t.Run(map[bool]string{false: "per-call negotiation", true: "accepted malformed discovery"}[invalid], func(t *testing.T) {
			cloud := testcloud.New(t)
			var locates, gets, posts atomic.Int32
			cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) { locates.Add(1); return cloud.Server.URL + vscBase, nil }
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			service, err := conn.BlockStorageV3(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					n := gets.Add(1)
					if r.URL.Path != "/state-connection/v3/" || r.Header.Get("OpenStack-API-Version") != "" || r.Header.Get("X-OpenStack-Volume-API-Version") != "" {
						t.Error(r.URL, r.Header)
					}
					w.Header().Set("X-Stage", "discovery")
					if invalid {
						testcloud.JSON(w, 300, `[]`)
					} else {
						max := "3.90"
						if n > 1 {
							max = "3.40"
						}
						testcloud.JSON(w, 300, `{"id":"v3.0","max_version":"`+max+`"}`)
					}
					return
				}
				n := posts.Add(1)
				version := "3.71"
				if n > 1 {
					version = "3.40"
				}
				vscWire(t, r, "os-reserve", version, "test-token")
				w.Header().Set("X-Stage", "applied")
				w.WriteHeader(204)
			})
			iterations := 2
			if invalid {
				iterations = 1
			}
			for n := 0; n < iterations; n++ {
				result, err := conn.ReserveVolume(context.Background(), blockstorage.VolumeActionRequest{VolumeID: "literal-name"})
				if result == nil || len(result.Discovery) != 1 || result.Discovery[0].StatusCode != 300 || result.Discovery[0].Header.Get("X-Stage") != "discovery" {
					t.Fatal(result, err)
				}
				if invalid {
					var proof *resource.ResponseError
					if err == nil || result.Completed || result.Applied != nil || !errors.As(err, &proof) || proof.StatusCode != 300 || proof.Header.Get("X-Stage") != "discovery" {
						t.Fatal(result, err, proof)
					}
					vscError(t, "ReserveVolume", err)
				} else {
					if err != nil || !result.Completed || result.Applied == nil || result.Applied.Header.Get("X-Stage") != "applied" {
						t.Fatal(result, err)
					}
					result.Discovery[0].Body[0] = 'x'
				}
			}
			if locates.Load() != 1 || gets.Load() != int32(iterations) || posts.Load() != map[bool]int32{false: 2, true: 0}[invalid] || service.RawClient().Microversion != "" {
				t.Fatal(locates.Load(), gets.Load(), posts.Load(), service.RawClient())
			}
		})
	}
}
