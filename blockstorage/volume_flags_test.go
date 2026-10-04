package blockstorage_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/blockstorage/v3/volumes"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

type vfaCase struct {
	name, operation, body string
	call                  func(context.Context, *gophercloud.ServiceClient, string) (*blockstorage.VolumeActionResult, error)
}

func vfaCases() []vfaCase {
	no := false
	factory := blockstorage.WithVolumeReadonlyOptions(blockstorage.VolumeReadonlyOpts{Readonly: &no})
	no = true
	return []vfaCase{
		{"bootable false", "SetVolumeBootableStatus", `{"os-set_bootable":{"bootable":false}}`, func(ctx context.Context, client *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.SetVolumeBootableStatus(ctx, client, blockstorage.VolumeActionRequest{VolumeID: id}, false)
		}},
		{"bootable true", "SetVolumeBootableStatus", `{"os-set_bootable":{"bootable":true}}`, func(ctx context.Context, client *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.SetVolumeBootableStatus(ctx, client, blockstorage.VolumeActionRequest{VolumeID: id}, true)
		}},
		{"readonly omitted", "SetVolumeReadonly", `{"os-update_readonly_flag":{"readonly":true}}`, func(ctx context.Context, client *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.SetVolumeReadonly(ctx, client, blockstorage.VolumeActionRequest{VolumeID: id})
		}},
		{"readonly false", "SetVolumeReadonly", `{"os-update_readonly_flag":{"readonly":false}}`, func(ctx context.Context, client *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.SetVolumeReadonly(ctx, client, blockstorage.VolumeActionRequest{VolumeID: id}, blockstorage.WithVolumeReadonly(false))
		}},
		{"readonly complete replacement", "SetVolumeReadonly", `{"os-update_readonly_flag":{"readonly":true}}`, func(ctx context.Context, client *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.SetVolumeReadonly(ctx, client, blockstorage.VolumeActionRequest{VolumeID: id}, blockstorage.WithVolumeReadonly(false), blockstorage.WithVolumeReadonlyOptions(blockstorage.VolumeReadonlyOpts{}))
		}},
		{"readonly factory owns false", "SetVolumeReadonly", `{"os-update_readonly_flag":{"readonly":false}}`, func(ctx context.Context, client *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.SetVolumeReadonly(ctx, client, blockstorage.VolumeActionRequest{VolumeID: id}, factory)
		}},
	}
}
func vfaPayload(t *testing.T, req *http.Request, id, body, version string) {
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
		t.Error("bool action changed literal key/type/value, route, selected source or framing", req.Method, req.URL, string(raw), req.Header, req.ContentLength, req.TransferEncoding, err)
	}
}

func TestVolumeFlagsDirectActionsKeepRequiredBootableAndReadonlyDefaultWithoutLookup(t *testing.T) {
	for index, tc := range vfaCases() {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.60")
			id := "한글-ID"
			reply := []byte{0xff, 0x00, '{', '!'}
			code := []int{203, 302, 399, 200, 203, 203}[index]
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				vfaPayload(t, req, id, tc.body, "3.60")
				w.Header().Set("X-Proof", "opaque bool action")
				w.WriteHeader(code)
				_, _ = w.Write(reply)
			})
			result, err := tc.call(vsaContext(t), client, id)
			if err != nil || result == nil || !result.Completed || result.VolumeID != id || result.Microversion != "3.60" || len(result.Discovery) != 0 || result.Applied == nil || result.Applied.StatusCode != code || result.Applied.Header.Get("X-Proof") != "opaque bool action" || !bytes.Equal(result.Applied.Body, reply) || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			result.Applied.Body[0] = '!'
			result.Applied.Header["X-Proof"][0] = "caller mutated"
			second, err := tc.call(vsaContext(t), client, id)
			if err != nil || second == nil || !second.Completed || second.Applied == nil || !bytes.Equal(second.Applied.Body, reply) || second.Applied.Header.Get("X-Proof") != "opaque bool action" || calls.Load() != 2 || reply[0] != 0xff || client.Microversion != "3.60" {
				t.Fatal("opaque proof aliases caller or another call", second, err, calls.Load(), reply, client)
			}
		})
	}
}

func TestVolumeFlagsDirectPreflightPrecedesReadonlyOriginalsAndGuardsCapturedSource(t *testing.T) {
	for _, operation := range []string{"SetVolumeBootableStatus", "SetVolumeReadonly"} {
		t.Run(operation, func(t *testing.T) {
			for _, kind := range []string{"unsafe ID", "empty ID", "invalid UTF8 ID", "nil client", "nil provider", "wrong role", "reserved auth", "foreign base", "nil context", "custom cancellation"} {
				t.Run(kind, func(t *testing.T) {
					cloud := testcloud.New(t)
					client := vsaClient(cloud, "3.60")
					ctx := vsaContext(t)
					id := "id"
					cause := errors.New("bool action custom cancellation")
					want := resource.ErrInvalidOption
					switch kind {
					case "unsafe ID":
						id = "a%2Fb"
					case "empty ID":
						id = ""
					case "invalid UTF8 ID":
						id = string([]byte{0xff})
					case "nil client":
						client = nil
					case "nil provider":
						client.ProviderClient = nil
					case "wrong role":
						client.Type = "compute"
						want = resource.ErrUnsupported
					case "reserved auth":
						client.MoreHeaders["Cookie"] = "caller"
					case "foreign base":
						client.ResourceBase = "https://foreign.invalid/v3/"
					case "nil context":
						ctx = nil
					case "custom cancellation":
						child, cancel := context.WithCancelCause(ctx)
						cancel(cause)
						ctx = child
						want = context.Canceled
					}
					var calls, originals atomic.Int32
					cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) { calls.Add(1); w.WriteHeader(500) })
					var result *blockstorage.VolumeActionResult
					var err error
					if operation == "SetVolumeBootableStatus" {
						result, err = blockstorage.SetVolumeBootableStatus(ctx, client, blockstorage.VolumeActionRequest{VolumeID: id}, false)
					} else {
						result, err = blockstorage.SetVolumeReadonly(ctx, client, blockstorage.VolumeActionRequest{VolumeID: id}, func(opts *blockstorage.VolumeReadonlyOpts) error { originals.Add(1); return nil })
					}
					vsaOperation(t, err, operation)
					var proof *resource.ResponseError
					if result != nil || !errors.Is(err, want) || errors.As(err, &proof) || calls.Load() != 0 || originals.Load() != 0 {
						t.Fatal(result, err, proof, calls.Load(), originals.Load())
					}
					if kind == "custom cancellation" && !errors.Is(err, cause) {
						t.Fatal("lost custom preflight cause", err)
					}
				})
			}
		})
	}
	t.Run("original cannot swap source then restore in later callback", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := vsaClient(cloud, "3.60")
		original := client.Endpoint
		sentinel := errors.New("readonly callback sentinel")
		var calls, first, later atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) { calls.Add(1); w.WriteHeader(500) })
		result, err := blockstorage.SetVolumeReadonly(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"},
			func(opts *blockstorage.VolumeReadonlyOpts) error {
				first.Add(1)
				client.Endpoint += "changed/"
				return sentinel
			},
			func(opts *blockstorage.VolumeReadonlyOpts) error {
				later.Add(1)
				client.Endpoint = original
				return nil
			})
		vsaOperation(t, err, "SetVolumeReadonly")
		if result != nil || !errors.Is(err, sentinel) || !errors.Is(err, resource.ErrInvalidOption) || first.Load() != 1 || later.Load() != 0 || calls.Load() != 0 {
			t.Fatal(result, err, first.Load(), later.Load(), calls.Load())
		}
	})
}

func TestVolumeFlagsNativeRejectionsAndExpandedStatusNeverBecomeAcknowledgements(t *testing.T) {
	for _, tc := range []vfaCase{vfaCases()[0], vfaCases()[3]} {
		t.Run(tc.name, func(t *testing.T) {
			for _, kind := range []string{"native400", "expanded native400"} {
				t.Run(kind, func(t *testing.T) {
					cloud := testcloud.New(t)
					client := vsaClient(cloud, "3.60")
					raw := []byte{0xff, 0x00, 'p'}
					var calls, retries atomic.Int32
					cloud.Provider.HTTPClient.Transport = vsaTransport(func(req *http.Request) (*http.Response, error) {
						call := calls.Add(1)
						vfaPayload(t, req, "id", tc.body, "3.60")
						code := 400
						if kind == "expanded native400" && call == 1 {
							code = 503
						}
						return vsaResponse(req, code, raw, "actual bool rejection"), nil
					})
					if kind == "expanded native400" {
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
					if kind == "expanded native400" {
						expectedCalls = 2
					}
					if result == nil || result.Completed || result.Applied != nil || result.VolumeID != "id" || result.Microversion != "3.60" || len(result.Discovery) != 0 || errors.As(err, &proof) || !errors.As(err, &native) || native.Actual != 400 || native.ResponseHeader.Get("X-Proof") != "actual bool rejection" || !bytes.Equal(native.Body, raw) || calls.Load() != expectedCalls || (kind == "expanded native400" && retries.Load() != 1) {
						t.Fatal(result, err, native, proof, calls.Load(), retries.Load())
					}
				})
			}
		})
	}
}

func TestVolumeFlagsSelectedLiteralAndDiscoveryUseOwnedVolumeMicroversionPolicy(t *testing.T) {
	cases := []struct {
		name, selected, reply, version string
		gets                           int
	}{
		{"selected above ceiling", "3.80", "", "3.80", 0},
		{"selected latest", "latest", "", "latest", 0},
		{"discovered volume ceiling", "", `{"version":{"id":"v3.0","status":"CURRENT","max_version":"3.99","min_version":"3.0"}}`, "3.71", 1},
		{"discovered below ceiling", "", `{"id":"v3.0","max_version":"3.70"}`, "3.70", 1},
		{"minimum above ceiling", "", `{"id":"v3.0","max_version":"3.99","min_version":"3.72"}`, "", 1},
		{"missing maximum stops", "", `{"id":"v3.0","min_version":"bad"}`, "", 1},
	}
	for _, tc := range []vfaCase{vfaCases()[0], vfaCases()[3]} {
		t.Run(tc.name, func(t *testing.T) {
			for _, policy := range cases {
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
							t.Error("bool action acquired lookup, extra negotiation or replay", call, req.URL)
						}
						vfaPayload(t, req, "id", tc.body, policy.version)
						w.Header().Set("X-Proof", "bool action")
						w.WriteHeader(203)
						_, _ = w.Write([]byte("opaque action"))
					})
					result, err := tc.call(vsaContext(t), client, "id")
					if err != nil || result == nil || !result.Completed || result.Microversion != policy.version || len(result.Discovery) != policy.gets || result.Applied == nil || result.Applied.Header.Get("X-Proof") != "bool action" || string(result.Applied.Body) != "opaque action" || calls.Load() != int32(policy.gets+1) || client.Microversion != policy.selected {
						t.Fatal(result, err, calls.Load(), client)
					}
					if policy.gets == 1 && (result.Discovery[0] == nil || result.Discovery[0].StatusCode != 300 || string(result.Discovery[0].Body) != policy.reply || result.Discovery[0].Header.Get("X-Proof") != "version selection") {
						t.Fatal("discovery proof borrowed opaque action", result)
					}
					expectedHeaders := 1
					if policy.selected != "" {
						expectedHeaders = 3
					}
					if len(client.MoreHeaders) != expectedHeaders || client.MoreHeaders["x-source"] != "original" {
						t.Fatal("operation mutated cached service policy", client.MoreHeaders)
					}
				})
			}
		})
	}
}

func TestVolumeFlagServiceMethodsKeepNative200AndCloudLookupDefaultsIndependent(t *testing.T) {
	cloud := testcloud.New(t)
	client := vsaClient(cloud, "3.60")
	api := volumes.New(client)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		call := calls.Add(1)
		if call == 6 {
			var body []byte
			if req.Body != nil {
				body, _ = io.ReadAll(req.Body)
			}
			if req.Method != http.MethodGet || req.URL.EscapedPath() != vsaBase+"volumes/id" || req.URL.RawQuery != "" || len(body) != 0 {
				t.Error("cloud helper lookup changed", req.Method, req.URL, string(body))
			}
			testcloud.JSON(w, 200, `{"volume":{"id":"id","status":"available"}}`)
			return
		}
		expected := `{"os-set_bootable":{"bootable":false}}`
		if call == 4 {
			expected = `{"os-update_readonly_flag":{"readonly":true}}`
		}
		if call == 5 {
			expected = `{"os-update_readonly_flag":{"readonly":false}}`
		}
		if call == 7 {
			expected = `{"os-set_bootable":{"bootable":true}}`
		}
		vfaPayload(t, req, "id", expected, "3.60")
		if call > 7 {
			t.Error("service flag unexpectedly resolved or replayed", call, req.URL)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("X-Proof", fmt.Sprintf("service%d", call))
		code := 203
		if call == 1 {
			code = 200
		}
		w.WriteHeader(code)
		_, _ = w.Write([]byte{0xff, 0x00})
	})
	if err := api.SetBootable(vsaContext(t), "id", volumes.BootableOpts{}); err != nil || calls.Load() != 1 {
		t.Fatal("native bool zero/default body or200 changed", err, calls.Load())
	}
	err := api.SetBootable(vsaContext(t), "id", volumes.BootableOpts{Bootable: false})
	var native gophercloud.ErrUnexpectedResponseCode
	var nativeOperation *resource.OperationError
	if !errors.As(err, &native) || native.Actual != 203 || len(native.Expected) != 1 || native.Expected[0] != 200 || !errors.As(err, &nativeOperation) || nativeOperation.Operation != "SetBootable" || nativeOperation.Resource != "volumes" || calls.Load() != 2 {
		t.Fatal("native SetBootable widened status policy", err, native, nativeOperation, calls.Load())
	}
	bootable, err := api.SetVolumeBootableStatus(vsaContext(t), "id", false)
	if err != nil || bootable == nil || !bootable.Completed || bootable.Applied == nil || bootable.Applied.StatusCode != 203 || !bytes.Equal(bootable.Applied.Body, []byte{0xff, 0x00}) || len(bootable.Discovery) != 0 || calls.Load() != 3 {
		t.Fatal(bootable, err, calls.Load())
	}
	readonly, err := api.SetVolumeReadonly(vsaContext(t), "id")
	if err != nil || readonly == nil || !readonly.Completed || readonly.Applied == nil || readonly.Applied.Header.Get("X-Proof") != "service4" || calls.Load() != 4 {
		t.Fatal(readonly, err, calls.Load())
	}
	readonly, err = api.SetVolumeReadonly(vsaContext(t), "id", volumes.WithVolumeReadonly(false))
	if err != nil || readonly == nil || !readonly.Completed || readonly.Applied == nil || readonly.Applied.Header.Get("X-Proof") != "service5" || calls.Load() != 5 {
		t.Fatal(readonly, err, calls.Load())
	}
	cloudHelper, err := blockstorage.SetVolumeBootable(vsaContext(t), client, blockstorage.SetVolumeBootableRequest{NameOrID: "id"})
	if err != nil || cloudHelper == nil || cloudHelper.Resolved == nil || cloudHelper.Resolved.Volume == nil || cloudHelper.VolumeID != "id" || cloudHelper.Applied == nil || cloudHelper.Applied.Header.Get("X-Proof") != "service7" || calls.Load() != 7 {
		t.Fatal("existing cloud lookup/defaultTrue changed", cloudHelper, err, calls.Load())
	}
	var absent *volumes.API
	for _, operation := range []string{"SetVolumeBootableStatus", "SetVolumeReadonly"} {
		var result *volumes.VolumeActionResult
		var err error
		var originals atomic.Int32
		if operation == "SetVolumeBootableStatus" {
			result, err = absent.SetVolumeBootableStatus(vsaContext(t), "id", false)
		} else {
			result, err = absent.SetVolumeReadonly(vsaContext(t), "id", func(opts *volumes.VolumeReadonlyOpts) error { originals.Add(1); return nil })
		}
		vsaOperation(t, err, operation)
		var proof *resource.ResponseError
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || originals.Load() != 0 || calls.Load() != 7 {
			t.Fatal("nil API ran originals or acquired proof", result, err, proof, originals.Load(), calls.Load())
		}
	}
}
