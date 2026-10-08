package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type vrcOperation struct {
	name, body string
	call       func(*sdk.Connection, context.Context, blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error)
}

var vrcOperations = []vrcOperation{
	{"ExtendVolume", `{"os-extend":{"new_size":0}}`, func(c *sdk.Connection, ctx context.Context, in blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
		return c.ExtendVolume(ctx, in, 0)
	}},
	{"RetypeVolume", `{"os-retype":{"new_type":""}}`, func(c *sdk.Connection, ctx context.Context, in blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
		return c.RetypeVolume(ctx, in, "", blockstorage.WithVolumeRetypeMigrationPolicy(""))
	}},
	{"CompleteVolumeExtend", `{"os-extend_volume_completion":{"error":true}}`, func(c *sdk.Connection, ctx context.Context, in blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
		return c.CompleteVolumeExtend(ctx, in, blockstorage.WithVolumeExtendCompletionError(true))
	}},
}

func vrcWire(t *testing.T, r *http.Request, body, version string) {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	modern := ""
	if version != "" {
		modern = "volume " + version
	}
	if err != nil || string(raw) != body || r.Method != http.MethodPost || r.URL.Path != vscBase+"volumes/literal-name/action" || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != modern || r.Header.Get("X-OpenStack-Volume-API-Version") != version || r.ContentLength != int64(len(body)) || len(r.TransferEncoding) != 0 {
		t.Error(r.Method, r.URL, string(raw), body, r.Header, err)
	}
}
func TestConnectionVolumeResizeActionsUseCachedCinderAndIgnoreLocation(t *testing.T) {
	cloud := testcloud.New(t)
	var getters, posts atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		getters.Add(1)
		if opts.Type != "block-storage" || opts.Version != 3 {
			t.Error(opts)
		}
		return cloud.Server.URL + vscBase, nil
	}
	bad := string([]byte{0xff})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.80"), sdk.WithCloudLocation(resource.CloudLocation{Cloud: &bad, Zone: json.RawMessage("invalid")}))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.BlockStorageV3(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		n := posts.Add(1)
		if n > 3 {
			t.Error("extra lookup/wait", r.URL)
			w.WriteHeader(500)
			return
		}
		vrcWire(t, r, vrcOperations[n-1].body, "3.80")
		w.WriteHeader(203)
		_, _ = w.Write([]byte{0xff, '{'})
	})
	for _, op := range vrcOperations {
		result, err := op.call(conn, context.Background(), blockstorage.VolumeActionRequest{VolumeID: "literal-name"})
		if err != nil || result == nil || !result.Completed || result.VolumeID != "literal-name" || result.Microversion != "3.80" || len(result.Discovery) != 0 || result.Applied == nil || result.Applied.StatusCode != 203 || len(result.Applied.Body) != 2 || result.Applied.Body[0] != 0xff {
			t.Fatal(op.name, result, err)
		}
	}
	if getters.Load() != 1 || posts.Load() != 3 || service.RawClient().Microversion != "3.80" {
		t.Fatal(getters.Load(), posts.Load(), service.RawClient())
	}
}
func TestConnectionVolumeResizeOriginalsRunOnceBeforeGetterWithOwnedPolicy(t *testing.T) {
	for _, complete := range []bool{false, true} {
		t.Run(map[bool]string{false: "retype", true: "completion"}[complete], func(t *testing.T) {
			cloud := testcloud.New(t)
			var options, getters, posts atomic.Int32
			var retainedRetype *blockstorage.VolumeRetypeOpts
			var retainedComplete *blockstorage.VolumeExtendCompletionOpts
			cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
				getters.Add(1)
				if options.Load() != 1 {
					t.Error("getter before options", options.Load())
				}
				if complete {
					*retainedComplete.Error = false
				} else {
					*retainedRetype.MigrationPolicy = "never"
				}
				return cloud.Server.URL + vscBase, nil
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.80"))
			if err != nil {
				t.Fatal(err)
			}
			body := `{"os-retype":{"migration_policy":"on-demand","new_type":"type/name"}}`
			if complete {
				body = `{"os-extend_volume_completion":{"error":true}}`
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				vrcWire(t, r, body, "3.80")
				w.WriteHeader(204)
			})
			var result *blockstorage.VolumeActionResult
			input := blockstorage.VolumeActionRequest{VolumeID: "literal-name"}
			if complete {
				result, err = conn.CompleteVolumeExtend(context.Background(), input, func(next *blockstorage.VolumeExtendCompletionOpts) error {
					options.Add(1)
					yes := true
					next.Error = &yes
					retainedComplete = next
					return nil
				})
			} else {
				result, err = conn.RetypeVolume(context.Background(), input, "type/name", func(next *blockstorage.VolumeRetypeOpts) error {
					options.Add(1)
					value := "on-demand"
					next.MigrationPolicy = &value
					retainedRetype = next
					return nil
				})
			}
			if err != nil || result == nil || !result.Completed || options.Load() != 1 || getters.Load() != 1 || posts.Load() != 1 {
				t.Fatal(result, err, options.Load(), getters.Load(), posts.Load())
			}
		})
	}
}
func TestConnectionVolumeResizePreflightAndGetterErrorsKeepCauses(t *testing.T) {
	for _, op := range vrcOperations {
		for _, kind := range []string{"nil context", "nil connection", "zero connection", "unsafe ID", "canceled", "getter error", "getter canceled error", "getter canceled success"} {
			t.Run(op.name+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause, getter := errors.New("resize canceled"), errors.New("resize getter")
				var getters, posts atomic.Int32
				cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
					getters.Add(1)
					if kind == "getter canceled error" || kind == "getter canceled success" {
						cancel(cause)
					}
					if kind == "getter canceled success" {
						return cloud.Server.URL + vscBase, nil
					}
					return "", getter
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { posts.Add(1); w.WriteHeader(500) })
				conn, err := sdk.FromProvider(cloud.Provider)
				if err != nil {
					t.Fatal(err)
				}
				var callCtx context.Context = ctx
				id := "literal-name"
				want := resource.ErrInvalidOption
				expected := int32(0)
				switch kind {
				case "nil context":
					callCtx = nil
				case "nil connection":
					conn = nil
				case "zero connection":
					conn = &sdk.Connection{}
				case "unsafe ID":
					id = "a/b"
				case "canceled":
					cancel(cause)
					want = context.Canceled
				case "getter error":
					want = getter
					expected = 1
				case "getter canceled error":
					want = getter
					expected = 1
				case "getter canceled success":
					want = context.Canceled
					expected = 1
				}
				result, err := op.call(conn, callCtx, blockstorage.VolumeActionRequest{VolumeID: id})
				var proof *resource.ResponseError
				if result != nil || !errors.Is(err, want) || errors.As(err, &proof) || getters.Load() != expected || posts.Load() != 0 {
					t.Fatal(result, err, proof, getters.Load(), posts.Load())
				}
				vscError(t, op.name, err)
				if kind == "canceled" || kind == "getter canceled error" || kind == "getter canceled success" {
					if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
						t.Fatal(err)
					}
				}
			})
		}
	}
	for _, kind := range []string{"invalid type", "invalid migration", "nil retype option", "nil completion option", "retype callback error", "completion callback canceled"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			var getters, later atomic.Int32
			cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) { getters.Add(1); return cloud.Server.URL + vscBase, nil }
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("resize original")
			input := blockstorage.VolumeActionRequest{VolumeID: "literal-name"}
			operation := "RetypeVolume"
			want := resource.ErrInvalidOption
			var result *blockstorage.VolumeActionResult
			switch kind {
			case "invalid type":
				result, err = conn.RetypeVolume(ctx, input, string([]byte{0xff}))
			case "invalid migration":
				result, err = conn.RetypeVolume(ctx, input, "type", blockstorage.WithVolumeRetypeMigrationPolicy(string([]byte{0xff})))
			case "nil retype option":
				result, err = conn.RetypeVolume(ctx, input, "type", nil)
			case "nil completion option":
				operation = "CompleteVolumeExtend"
				result, err = conn.CompleteVolumeExtend(ctx, input, nil)
			case "retype callback error":
				want = cause
				result, err = conn.RetypeVolume(ctx, input, "type", func(*blockstorage.VolumeRetypeOpts) error { return cause }, func(*blockstorage.VolumeRetypeOpts) error { later.Add(1); return nil })
			case "completion callback canceled":
				operation = "CompleteVolumeExtend"
				want = cause
				result, err = conn.CompleteVolumeExtend(ctx, input, func(*blockstorage.VolumeExtendCompletionOpts) error { cancel(cause); return cause }, func(*blockstorage.VolumeExtendCompletionOpts) error { later.Add(1); return nil })
			}
			if result != nil || !errors.Is(err, want) || getters.Load() != 0 || later.Load() != 0 {
				t.Fatal(result, err, getters.Load(), later.Load())
			}
			vscError(t, operation, err)
			if kind == "completion callback canceled" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}
func TestConnectionVolumeResizeNegotiationKeepsAcceptedStagesSeparate(t *testing.T) {
	for _, malformed := range []bool{false, true} {
		t.Run(map[bool]string{false: "per-call", true: "malformed discovery"}[malformed], func(t *testing.T) {
			cloud := testcloud.New(t)
			var getters, gets, posts atomic.Int32
			cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) { getters.Add(1); return cloud.Server.URL + vscBase, nil }
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					n := gets.Add(1)
					if r.URL.Path != "/state-connection/v3/" || r.Header.Get("OpenStack-API-Version") != "" {
						t.Error(r.URL, r.Header)
					}
					w.Header().Set("X-Stage", "discovery")
					if malformed {
						testcloud.JSON(w, 300, `[]`)
					} else {
						maximum := "3.99"
						if n > 1 {
							maximum = "3.40"
						}
						testcloud.JSON(w, 300, `{"id":"v3.0","max_version":"`+maximum+`"}`)
					}
					return
				}
				n := posts.Add(1)
				version := "3.71"
				if n > 1 {
					version = "3.40"
				}
				vrcWire(t, r, vrcOperations[n-1].body, version)
				w.Header().Set("X-Stage", "applied")
				w.WriteHeader(204)
			})
			for _, op := range vrcOperations {
				result, err := op.call(conn, context.Background(), blockstorage.VolumeActionRequest{VolumeID: "literal-name"})
				if result == nil || len(result.Discovery) != 1 || result.Discovery[0].Header.Get("X-Stage") != "discovery" {
					t.Fatal(op.name, result, err)
				}
				if malformed {
					var proof *resource.ResponseError
					if err == nil || result.Completed || result.Applied != nil || !errors.As(err, &proof) || proof.StatusCode != 300 || proof.Header.Get("X-Stage") != "discovery" {
						t.Fatal(result, err, proof)
					}
					vscError(t, op.name, err)
				} else if err != nil || !result.Completed || result.Applied == nil || result.Applied.Header.Get("X-Stage") != "applied" {
					t.Fatal(result, err)
				}
			}
			service, err := conn.BlockStorageV3(context.Background())
			if err != nil || service.RawClient().Microversion != "" || getters.Load() != 1 || gets.Load() != 3 || posts.Load() != map[bool]int32{false: 3, true: 0}[malformed] {
				t.Fatal(err, getters.Load(), gets.Load(), posts.Load())
			}
		})
	}
}
