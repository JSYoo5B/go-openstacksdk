package blockstorage_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type vrrCase struct {
	name, operation, body string
	call                  func(context.Context, *gophercloud.ServiceClient, string) (*blockstorage.VolumeActionResult, error)
}

func vrrCases() []vrrCase {
	cases := []vrrCase{
		{"extend zero", "ExtendVolume", `{"os-extend":{"new_size":0}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.ExtendVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, 0)
		}},
		{"extend negative", "ExtendVolume", `{"os-extend":{"new_size":-7}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.ExtendVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, -7)
		}},
		{"retype empty literal with default never", "RetypeVolume", `{"os-retype":{"migration_policy":"never","new_type":""}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.RetypeVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "")
		}},
		{"retype explicit empty policy omits", "RetypeVolume", `{"os-retype":{"new_type":"literal/type?%#"}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.RetypeVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "literal/type?%#", blockstorage.WithVolumeRetypeMigrationPolicy(""))
		}},
		{"retype body controls and unknown policy", "RetypeVolume", `{"os-retype":{"migration_policy":"server-mode","new_type":"type /?%#\n\u0000"}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.RetypeVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "type /?%#\n\x00", blockstorage.WithVolumeRetypeMigrationPolicy("server-mode"))
		}},
		{"retype replacement restores never", "RetypeVolume", `{"os-retype":{"migration_policy":"never","new_type":"fast"}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.RetypeVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "fast", blockstorage.WithVolumeRetypeMigrationPolicy(""), blockstorage.WithVolumeRetypeOptions(blockstorage.VolumeRetypeOpts{}))
		}},
		{"completion omitted false", "CompleteVolumeExtend", `{"os-extend_volume_completion":{"error":false}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.CompleteVolumeExtend(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id})
		}},
		{"completion explicit false", "CompleteVolumeExtend", `{"os-extend_volume_completion":{"error":false}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.CompleteVolumeExtend(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, blockstorage.WithVolumeExtendCompletionError(false))
		}},
		{"completion true", "CompleteVolumeExtend", `{"os-extend_volume_completion":{"error":true}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.CompleteVolumeExtend(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, blockstorage.WithVolumeExtendCompletionError(true))
		}},
		{"completion replacement restores false", "CompleteVolumeExtend", `{"os-extend_volume_completion":{"error":false}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.CompleteVolumeExtend(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, blockstorage.WithVolumeExtendCompletionError(true), blockstorage.WithVolumeExtendCompletionOptions(blockstorage.VolumeExtendCompletionOpts{}))
		}},
	}
	if strconv.IntSize == 64 {
		large := int64(9007199254740993)
		size := int(large)
		cases = append(cases, vrrCase{"extend exact machine integer above float precision", "ExtendVolume", `{"os-extend":{"new_size":9007199254740993}}`, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.ExtendVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, size)
		}})
	}
	return cases
}
func vrrPayload(t *testing.T, req *http.Request, id, body, version string) {
	t.Helper()
	var raw []byte
	var err error
	if req.Body != nil {
		raw, err = io.ReadAll(req.Body)
	}
	current := ""
	if version != "" {
		current = "volume " + version
	}
	if err != nil || string(raw) != body || req.Method != http.MethodPost || req.URL.EscapedPath() != vsaBase+"volumes/"+url.PathEscape(id)+"/action" || req.URL.RawQuery != "" || req.Header.Get("X-Source") != "original" || req.Header.Get("X-Auth-Token") != "test-token" || req.Header.Get("OpenStack-API-Version") != current || req.Header.Get("X-OpenStack-Volume-API-Version") != version || req.ContentLength != int64(len(body)) || len(req.TransferEncoding) != 0 {
		t.Error("resize action changed source literal body, route, framing or policy", req.Method, req.URL, string(raw), req.Header, req.ContentLength, req.TransferEncoding, err)
	}
}

func TestVolumeResizeDirectBodiesPreserveSizesPolicyPresenceAndCompletionDefaults(t *testing.T) {
	for index, tc := range vrrCases() {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.60")
			id := "한글-ID"
			raw := []byte{0xff, 0x00, '{', '!'}
			code := []int{200, 203, 302, 399}[index%4]
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				vrrPayload(t, req, id, tc.body, "3.60")
				w.Header().Set("X-Proof", "opaque resize")
				w.WriteHeader(code)
				_, _ = w.Write(raw)
			})
			result, err := tc.call(vsaContext(t), client, id)
			if err != nil || result == nil || !result.Completed || result.VolumeID != id || result.Microversion != "3.60" || len(result.Discovery) != 0 || result.Applied == nil || result.Applied.StatusCode != code || !bytes.Equal(result.Applied.Body, raw) || result.Applied.Header.Get("X-Proof") != "opaque resize" || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			result.Applied.Body[0] = '!'
			result.Applied.Header["X-Proof"][0] = "caller mutation"
			second, err := tc.call(vsaContext(t), client, id)
			if err != nil || second == nil || !second.Completed || second.Applied == nil || !bytes.Equal(second.Applied.Body, raw) || second.Applied.Header.Get("X-Proof") != "opaque resize" || calls.Load() != 2 || raw[0] != 0xff || client.Microversion != "3.60" {
				t.Fatal("response aliases another invocation or original bytes", second, err, calls.Load(), raw, client)
			}
		})
	}
}

func TestVolumeResizeDirectPreflightValidatesOnlySelectedSourceAndUTF8BodyText(t *testing.T) {
	for _, operation := range []string{"ExtendVolume", "RetypeVolume", "CompleteVolumeExtend"} {
		t.Run(operation, func(t *testing.T) {
			kinds := []string{"unsafe volume ID", "nil client", "nil context", "custom cancellation", "wrong role", "reserved auth"}
			if operation == "RetypeVolume" {
				kinds = append(kinds, "invalid new type UTF8", "invalid selected policy UTF8")
			}
			for _, kind := range kinds {
				t.Run(kind, func(t *testing.T) {
					cloud := testcloud.New(t)
					client := vsaClient(cloud, "3.60")
					ctx := vsaContext(t)
					id, newType := "id", "fast"
					cause := errors.New("resize custom cancellation")
					want := resource.ErrInvalidOption
					switch kind {
					case "unsafe volume ID":
						id = "a%2Fb"
					case "nil client":
						client = nil
					case "nil context":
						ctx = nil
					case "custom cancellation":
						child, cancel := context.WithCancelCause(ctx)
						cancel(cause)
						ctx = child
						want = context.Canceled
					case "wrong role":
						client.Type = "compute"
						want = resource.ErrUnsupported
					case "reserved auth":
						client.MoreHeaders["Cookie"] = "caller"
					case "invalid new type UTF8":
						newType = string([]byte{0xff})
					}
					var calls, originals atomic.Int32
					cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) { calls.Add(1); w.WriteHeader(500) })
					var result *blockstorage.VolumeActionResult
					var err error
					switch operation {
					case "ExtendVolume":
						result, err = blockstorage.ExtendVolume(ctx, client, blockstorage.VolumeActionRequest{VolumeID: id}, 0)
					case "RetypeVolume":
						result, err = blockstorage.RetypeVolume(ctx, client, blockstorage.VolumeActionRequest{VolumeID: id}, newType, func(opts *blockstorage.VolumeRetypeOpts) error {
							originals.Add(1)
							if kind == "invalid selected policy UTF8" {
								bad := string([]byte{0xff})
								opts.MigrationPolicy = &bad
							}
							return nil
						})
					case "CompleteVolumeExtend":
						result, err = blockstorage.CompleteVolumeExtend(ctx, client, blockstorage.VolumeActionRequest{VolumeID: id}, func(opts *blockstorage.VolumeExtendCompletionOpts) error { originals.Add(1); return nil })
					}
					vsaOperation(t, err, operation)
					var proof *resource.ResponseError
					expectedOriginals := int32(0)
					if kind == "invalid selected policy UTF8" {
						expectedOriginals = 1
					}
					if result != nil || !errors.Is(err, want) || errors.As(err, &proof) || calls.Load() != 0 || originals.Load() != expectedOriginals {
						t.Fatal(result, err, proof, calls.Load(), originals.Load())
					}
					if kind == "custom cancellation" && !errors.Is(err, cause) {
						t.Fatal("lost caller cause", err)
					}
				})
			}
		})
	}
}

func TestVolumeResizeOriginalHTTPPolicyKeepsRejectedNativeEvidence(t *testing.T) {
	for _, tc := range []vrrCase{vrrCases()[0], vrrCases()[2], vrrCases()[6]} {
		t.Run(tc.name, func(t *testing.T) {
			for _, kind := range []string{"native400", "native expanded400"} {
				t.Run(kind, func(t *testing.T) {
					cloud := testcloud.New(t)
					client := vsaClient(cloud, "3.60")
					raw := []byte{0xff, 0x00, 'p'}
					var calls, retries atomic.Int32
					cloud.Provider.HTTPClient.Transport = vsaTransport(func(req *http.Request) (*http.Response, error) {
						call := calls.Add(1)
						vrrPayload(t, req, "id", tc.body, "3.60")
						code := 400
						if kind == "native expanded400" && call == 1 {
							code = 503
						}
						return vsaResponse(req, code, raw, "current rejected resize"), nil
					})
					if kind == "native expanded400" {
						cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, original error, _ uint) error {
							if !gophercloud.ResponseCodeIs(original, 503) || retries.Add(1) > 1 {
								return original
							}
							opts.OkCodes = append(opts.OkCodes, 400)
							return nil
						}
					}
					result, err := tc.call(vsaContext(t), client, "id")
					vsaOperation(t, err, tc.operation)
					var native gophercloud.ErrUnexpectedResponseCode
					var proof *resource.ResponseError
					expectedCalls := int32(1)
					if kind == "native expanded400" {
						expectedCalls = 2
					}
					if result == nil || result.Completed || result.Applied != nil || result.VolumeID != "id" || result.Microversion != "3.60" || len(result.Discovery) != 0 || !errors.As(err, &native) || native.Actual != 400 || !bytes.Equal(native.Body, raw) || native.ResponseHeader.Get("X-Proof") != "current rejected resize" || errors.As(err, &proof) || calls.Load() != expectedCalls || (kind == "native expanded400" && retries.Load() != 1) {
						t.Fatal(result, err, native, proof, calls.Load(), retries.Load())
					}
				})
			}
		})
	}
}

func TestVolumeResizeSelectedLiteralAndDiscoveryKeepActionPayloadAndOwnedPolicy(t *testing.T) {
	policies := []struct {
		name, selected, reply, version string
		gets                           int
	}{
		{"selected below hypothetical API minimum", "3.1", "", "3.1", 0},
		{"selected above class cap", "3.80", "", "3.80", 0},
		{"selected latest", "latest", "", "latest", 0},
		{"discovered ceiling", "", `{"version":{"id":"v3.0","status":"CURRENT","max_version":"3.99","min_version":"3.0"}}`, "3.71", 1},
		{"discovered below ceiling", "", `{"id":"v3.0","max_version":"3.70"}`, "3.70", 1},
		{"minimum above class ceiling", "", `{"id":"v3.0","max_version":"3.99","min_version":"3.72"}`, "", 1},
	}
	for _, tc := range []vrrCase{vrrCases()[0], vrrCases()[2], vrrCases()[6]} {
		t.Run(tc.name, func(t *testing.T) {
			for _, policy := range policies {
				t.Run(policy.name, func(t *testing.T) {
					cloud := testcloud.New(t)
					client := vsaClient(cloud, policy.selected)
					if policy.selected != "" {
						client.MoreHeaders["OpenStack-API-Version"] = "volume " + policy.selected
						client.MoreHeaders["X-OpenStack-Volume-API-Version"] = policy.selected
					}
					var calls atomic.Int32
					cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
						call := calls.Add(1)
						if call == 1 && policy.gets == 1 {
							vsaDiscoveryRequest(t, req, vsaVersionPath)
							w.Header().Set("X-Proof", "version selection")
							testcloud.JSON(w, 300, policy.reply)
							return
						}
						if call != int32(policy.gets+1) {
							t.Error("resize action resolved, replayed or negotiated again", call, req.URL)
						}
						vrrPayload(t, req, "id", tc.body, policy.version)
						w.Header().Set("X-Proof", "resize action")
						w.WriteHeader(203)
						_, _ = w.Write([]byte("opaque"))
					})
					result, err := tc.call(vsaContext(t), client, "id")
					if err != nil || result == nil || !result.Completed || result.Microversion != policy.version || len(result.Discovery) != policy.gets || result.Applied == nil || result.Applied.StatusCode != 203 || string(result.Applied.Body) != "opaque" || result.Applied.Header.Get("X-Proof") != "resize action" || calls.Load() != int32(policy.gets+1) || client.Microversion != policy.selected {
						t.Fatal(result, err, calls.Load(), client)
					}
					if policy.gets == 1 && (result.Discovery[0] == nil || result.Discovery[0].StatusCode != 300 || string(result.Discovery[0].Body) != policy.reply || result.Discovery[0].Header.Get("X-Proof") != "version selection") {
						t.Fatal("action proof replaced discovery", result)
					}
					expectedHeaders := 1
					if policy.selected != "" {
						expectedHeaders = 3
					}
					if len(client.MoreHeaders) != expectedHeaders || client.MoreHeaders["x-source"] != "original" {
						t.Fatal("operation mutated caller source policy", client.MoreHeaders)
					}
				})
			}
		})
	}
}

func TestVolumeResizeServiceMethodsPreserveNative202BuildersAndNilAPILabels(t *testing.T) {
	cloud := testcloud.New(t)
	client := vsaClient(cloud, "3.60")
	api := volumes.New(client)
	var calls atomic.Int32
	expected := map[int32]string{
		1: `{"os-extend":{"new_size":0}}`, 2: `{"os-extend":{"new_size":-7}}`, 3: `{"os-extend":{"new_size":0}}`,
		4: `{"os-retype":{"new_type":"fast"}}`, 5: `{"os-retype":{"new_type":"fast"}}`, 6: `{"os-retype":{"migration_policy":"never","new_type":""}}`, 7: `{"os-retype":{"new_type":"fast"}}`,
		8: `{"os-extend_volume_completion":{"error":false}}`, 9: `{"os-extend_volume_completion":{"error":true}}`,
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		call := calls.Add(1)
		if call > 9 {
			t.Error("service action resolved or replayed", call, req.URL)
			w.WriteHeader(500)
			return
		}
		vrrPayload(t, req, "id", expected[call], "3.60")
		code := 203
		if call == 1 || call == 4 {
			code = 202
		}
		w.Header().Set("X-Proof", fmt.Sprintf("service%d", call))
		w.WriteHeader(code)
		_, _ = w.Write([]byte{0xff, 0x00})
	})
	if err := api.ExtendSize(vsaContext(t), "id", volumes.ExtendSizeOpts{}); err != nil || calls.Load() != 1 {
		t.Fatal("native current numeric-zero rule or202 changed", err, calls.Load())
	}
	err := api.ExtendSize(vsaContext(t), "id", volumes.ExtendSizeOpts{NewSize: -7})
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != 203 || len(native.Expected) != 1 || native.Expected[0] != 202 || calls.Load() != 2 {
		t.Fatal("native ExtendSize policy widened", err, native, calls.Load())
	}
	extended, err := api.ExtendVolume(vsaContext(t), "id", 0)
	if err != nil || extended == nil || !extended.Completed || extended.Applied == nil || extended.Applied.StatusCode != 203 || !bytes.Equal(extended.Applied.Body, []byte{0xff, 0x00}) || calls.Load() != 3 {
		t.Fatal(extended, err, calls.Load())
	}
	if err := api.ChangeType(vsaContext(t), "id", volumes.ChangeTypeOpts{NewType: "fast"}); err != nil || calls.Load() != 4 {
		t.Fatal("native omitted policy or202 changed", err, calls.Load())
	}
	err = api.ChangeType(vsaContext(t), "id", volumes.ChangeTypeOpts{NewType: "fast"})
	if !errors.As(err, &native) || native.Actual != 203 || len(native.Expected) != 1 || native.Expected[0] != 202 || calls.Load() != 5 {
		t.Fatal("native ChangeType policy widened", err, native, calls.Load())
	}
	err = api.ChangeType(vsaContext(t), "id", volumes.ChangeTypeOpts{})
	var missing gophercloud.ErrMissingInput
	if !errors.As(err, &missing) || missing.Argument != "NewType" || calls.Load() != 5 {
		t.Fatal("native required string validation changed", err, missing, calls.Load())
	}
	retyped, err := api.RetypeVolume(vsaContext(t), "id", "")
	if err != nil || retyped == nil || !retyped.Completed || retyped.Applied == nil || retyped.Applied.Header.Get("X-Proof") != "service6" || calls.Load() != 6 {
		t.Fatal("new Retype rejected Source body empty string", retyped, err, calls.Load())
	}
	retyped, err = api.RetypeVolume(vsaContext(t), "id", "fast", volumes.WithVolumeRetypeMigrationPolicy(""))
	if err != nil || retyped == nil || !retyped.Completed || retyped.Applied == nil || retyped.Applied.Header.Get("X-Proof") != "service7" || calls.Load() != 7 {
		t.Fatal(retyped, err, calls.Load())
	}
	completed, err := api.CompleteVolumeExtend(vsaContext(t), "id")
	if err != nil || completed == nil || !completed.Completed || completed.Applied == nil || completed.Applied.Header.Get("X-Proof") != "service8" || calls.Load() != 8 {
		t.Fatal(completed, err, calls.Load())
	}
	completed, err = api.CompleteVolumeExtend(vsaContext(t), "id", volumes.WithVolumeExtendCompletionError(true))
	if err != nil || completed == nil || !completed.Completed || completed.Applied == nil || completed.Applied.Header.Get("X-Proof") != "service9" || calls.Load() != 9 {
		t.Fatal(completed, err, calls.Load())
	}
	var absent *volumes.API
	for _, operation := range []string{"ExtendVolume", "RetypeVolume", "CompleteVolumeExtend"} {
		var result *volumes.VolumeActionResult
		var err error
		var originals atomic.Int32
		switch operation {
		case "ExtendVolume":
			result, err = absent.ExtendVolume(vsaContext(t), "id", 0)
		case "RetypeVolume":
			result, err = absent.RetypeVolume(vsaContext(t), "id", "fast", func(opts *volumes.VolumeRetypeOpts) error { originals.Add(1); return nil })
		case "CompleteVolumeExtend":
			result, err = absent.CompleteVolumeExtend(vsaContext(t), "id", func(opts *volumes.VolumeExtendCompletionOpts) error { originals.Add(1); return nil })
		}
		vsaOperation(t, err, operation)
		var proof *resource.ResponseError
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || originals.Load() != 0 || calls.Load() != 9 {
			t.Fatal("nilAPI consumed callbacks or proof", result, err, proof, originals.Load(), calls.Load())
		}
	}
}
