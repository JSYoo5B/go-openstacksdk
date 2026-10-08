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

type vcfOperation struct {
	name, key string
	call      func(*sdk.Connection, context.Context, blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error)
}

var vcfOperations = []vcfOperation{
	{"SetVolumeBootableStatus", "os-set_bootable", func(c *sdk.Connection, ctx context.Context, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
		return c.SetVolumeBootableStatus(ctx, input, false)
	}},
	{"SetVolumeReadonly", "os-update_readonly_flag", func(c *sdk.Connection, ctx context.Context, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
		return c.SetVolumeReadonly(ctx, input, blockstorage.WithVolumeReadonly(false))
	}},
}

func vcfWire(t *testing.T, r *http.Request, key string, flag bool, version string) {
	t.Helper()
	field := "bootable"
	if key == "os-update_readonly_flag" {
		field = "readonly"
	}
	want, _ := json.Marshal(map[string]any{key: map[string]bool{field: flag}})
	body, err := io.ReadAll(r.Body)
	modern := ""
	if version != "" {
		modern = "volume " + version
	}
	if err != nil || string(body) != string(want) || r.Method != http.MethodPost || r.URL.Path != vscBase+"volumes/literal-name/action" || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != modern || r.Header.Get("X-OpenStack-Volume-API-Version") != version {
		t.Error(r.Method, r.URL, string(body), string(want), r.Header, err)
	}
}

func TestConnectionVolumeFlagsUseCachedCinderAndIgnoreLocation(t *testing.T) {
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
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.80"), sdk.WithCloudLocation(resource.CloudLocation{Cloud: &bad, Zone: json.RawMessage("not-json")}))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.BlockStorageV3(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		n := posts.Add(1)
		key := "os-set_bootable"
		if n > 1 {
			key = "os-update_readonly_flag"
		}
		vcfWire(t, r, key, n == 3, "3.80")
		w.WriteHeader(203)
		_, _ = w.Write([]byte{0xff, '{'})
	})
	input := blockstorage.VolumeActionRequest{VolumeID: "literal-name"}
	for _, op := range vcfOperations {
		result, err := op.call(conn, context.Background(), input)
		if err != nil || result == nil || !result.Completed || result.VolumeID != input.VolumeID || result.Microversion != "3.80" || len(result.Discovery) != 0 || result.Applied == nil || result.Applied.StatusCode != 203 || len(result.Applied.Body) != 2 || result.Applied.Body[0] != 0xff {
			t.Fatal(op.name, result, err)
		}
	}
	result, err := conn.SetVolumeReadonly(context.Background(), input)
	if err != nil || result == nil || !result.Completed || posts.Load() != 3 || locates.Load() != 1 || service.RawClient().Microversion != "3.80" {
		t.Fatal(result, err, posts.Load(), locates.Load())
	}
}

func TestConnectionVolumeReadonlyPreparesOriginalsOnceBeforeGetter(t *testing.T) {
	cloud := testcloud.New(t)
	var options, gets, posts atomic.Int32
	var retained *blockstorage.VolumeReadonlyOpts
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		gets.Add(1)
		if options.Load() != 1 || retained == nil {
			t.Error("getter preceded original option", options.Load(), retained)
		}
		*retained.Readonly = true
		return cloud.Server.URL + vscBase, nil
	}
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.80"))
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		vcfWire(t, r, "os-update_readonly_flag", false, "3.80")
		w.WriteHeader(204)
	})
	result, err := conn.SetVolumeReadonly(context.Background(), blockstorage.VolumeActionRequest{VolumeID: "literal-name"}, func(next *blockstorage.VolumeReadonlyOpts) error {
		options.Add(1)
		flag := false
		next.Readonly = &flag
		retained = next
		return nil
	})
	if err != nil || result == nil || !result.Completed || options.Load() != 1 || gets.Load() != 1 || posts.Load() != 1 {
		t.Fatal(result, err, options.Load(), gets.Load(), posts.Load())
	}
}

func TestConnectionVolumeFlagsPreflightAndGetterErrorsKeepCauses(t *testing.T) {
	for _, op := range vcfOperations {
		for _, kind := range []string{"nil context", "nil connection", "zero connection", "unsafe ID", "cancel", "getter error", "getter canceled error", "getter canceled success"} {
			t.Run(op.name+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				var getters, wire atomic.Int32
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause, getter := errors.New("flag cancellation"), errors.New("flag getter")
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
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { wire.Add(1); w.WriteHeader(500) })
				conn, err := sdk.FromProvider(cloud.Provider)
				if err != nil {
					t.Fatal(err)
				}
				var callCtx context.Context = ctx
				id := "literal-name"
				want, expectedGetters := resource.ErrInvalidOption, int32(0)
				switch kind {
				case "nil context":
					callCtx = nil
				case "nil connection":
					conn = nil
				case "zero connection":
					conn = &sdk.Connection{}
				case "unsafe ID":
					id = "a/b"
				case "cancel":
					cancel(cause)
					want = context.Canceled
				case "getter error":
					want = getter
					expectedGetters = 1
				case "getter canceled error":
					want = getter
					expectedGetters = 1
				case "getter canceled success":
					want = context.Canceled
					expectedGetters = 1
				}
				result, err := op.call(conn, callCtx, blockstorage.VolumeActionRequest{VolumeID: id})
				var physical *resource.ResponseError
				if result != nil || !errors.Is(err, want) || errors.As(err, &physical) || getters.Load() != expectedGetters || wire.Load() != 0 {
					t.Fatal(result, err, physical, getters.Load(), wire.Load())
				}
				if kind == "cancel" || kind == "getter canceled error" || kind == "getter canceled success" {
					if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
						t.Fatal(err)
					}
				}
				vscError(t, op.name, err)
			})
		}
	}
	for _, kind := range []string{"nil option", "callback error", "callback canceled"} {
		t.Run("readonly/"+kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			var getters, later atomic.Int32
			cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) { getters.Add(1); return cloud.Server.URL + vscBase, nil }
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("readonly option error")
			var first blockstorage.VolumeReadonlyOption
			want := resource.ErrInvalidOption
			if kind != "nil option" {
				want = cause
				first = func(*blockstorage.VolumeReadonlyOpts) error {
					if kind == "callback canceled" {
						cancel(cause)
					}
					return cause
				}
			}
			result, err := conn.SetVolumeReadonly(ctx, blockstorage.VolumeActionRequest{VolumeID: "literal-name"}, first, func(*blockstorage.VolumeReadonlyOpts) error { later.Add(1); return nil })
			if result != nil || !errors.Is(err, want) || getters.Load() != 0 || later.Load() != 0 {
				t.Fatal(result, err, getters.Load(), later.Load())
			}
			if kind == "callback canceled" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			vscError(t, "SetVolumeReadonly", err)
		})
	}
}

func TestConnectionVolumeFlagsKeepNegotiationAndAppliedStagesSeparate(t *testing.T) {
	for _, malformed := range []bool{false, true} {
		t.Run(map[bool]string{false: "per call", true: "malformed discovery"}[malformed], func(t *testing.T) {
			cloud := testcloud.New(t)
			var locates, gets, posts atomic.Int32
			cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) { locates.Add(1); return cloud.Server.URL + vscBase, nil }
			conn, err := sdk.FromProvider(cloud.Provider)
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
					if malformed {
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
				op := vcfOperations[n-1]
				version := "3.71"
				if n > 1 {
					version = "3.40"
				}
				vcfWire(t, r, op.key, false, version)
				w.Header().Set("X-Stage", "applied")
				w.WriteHeader(204)
			})
			for _, op := range vcfOperations {
				result, err := op.call(conn, context.Background(), blockstorage.VolumeActionRequest{VolumeID: "literal-name"})
				if result == nil || len(result.Discovery) != 1 || result.Discovery[0].Header.Get("X-Stage") != "discovery" {
					t.Fatal(op.name, result, err)
				}
				if malformed {
					var proof *resource.ResponseError
					if err == nil || result.Completed || result.Applied != nil || !errors.As(err, &proof) || proof.StatusCode != 300 || proof.Header.Get("X-Stage") != "discovery" {
						t.Fatal(op.name, result, err, proof)
					}
					vscError(t, op.name, err)
				} else if err != nil || !result.Completed || result.Applied == nil || result.Applied.Header.Get("X-Stage") != "applied" {
					t.Fatal(op.name, result, err)
				}
			}
			service, err := conn.BlockStorageV3(context.Background())
			if err != nil || service.RawClient().Microversion != "" || locates.Load() != 1 || gets.Load() != 2 || posts.Load() != map[bool]int32{false: 2, true: 0}[malformed] {
				t.Fatal(err, locates.Load(), gets.Load(), posts.Load())
			}
		})
	}
}
