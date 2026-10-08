package blockstorage_test

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

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type vruAction struct {
	operation, body string
	required        bool
	call            func(context.Context, *gophercloud.ServiceClient, string) (*blockstorage.VolumeActionResult, error)
}

func vruActions() []vruAction {
	return []vruAction{
		{"RevertVolumeToSnapshot", `{"revert":{"snapshot_id":"snapshot"}}`, true, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.RevertVolumeToSnapshot(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id}, "snapshot")
		}},
		{"UnmanageVolume", `{"os-unmanage":null}`, false, func(ctx context.Context, c *gophercloud.ServiceClient, id string) (*blockstorage.VolumeActionResult, error) {
			return blockstorage.UnmanageVolume(ctx, c, blockstorage.VolumeActionRequest{VolumeID: id})
		}},
	}
}

const vruSupport = `{"version":{"id":"v3.0","min_version":"3.0","max_version":"3.99"}}`

func TestVolumeRevertAndUnmanageKeepLiteralBodyDomainsAndOpaqueAcknowledgements(t *testing.T) {
	cases := []struct {
		name, snapshot, body string
		revert               bool
		code                 int
	}{
		{"revert explicit empty snapshot", "", `{"revert":{"snapshot_id":""}}`, true, 200},
		{"revert path-like controls stay body text", "snap /?%#\n\x00", `{"revert":{"snapshot_id":"snap /?%#\n\u0000"}}`, true, 203},
		{"revert Unicode body and current ID", "한글-é", `{"revert":{"snapshot_id":"한글-é"}}`, true, 302},
		{"revert arbitrary literal snapshot", "not-a-UUID", `{"revert":{"snapshot_id":"not-a-UUID"}}`, true, 399},
		{"unmanage 200 null", "", `{"os-unmanage":null}`, false, 200},
		{"unmanage 203 null", "", `{"os-unmanage":null}`, false, 203},
		{"unmanage 302 null", "", `{"os-unmanage":null}`, false, 302},
		{"unmanage 399 null", "", `{"os-unmanage":null}`, false, 399},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.60")
			id := "volume-한글"
			reply := []byte{0xff, 0x00, '{', '!'}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				n := calls.Add(1)
				if tc.revert && n == 1 {
					vsaDiscoveryRequest(t, req, vsaVersionPath)
					w.Header().Set("X-Proof", "support first")
					testcloud.JSON(w, 200, vruSupport)
					return
				}
				vmrRequest(t, req, id, tc.body, "3.60")
				w.Header().Set("X-Proof", "opaque current action")
				w.WriteHeader(tc.code)
				_, _ = w.Write(reply)
			})
			var result *blockstorage.VolumeActionResult
			var err error
			if tc.revert {
				result, err = blockstorage.RevertVolumeToSnapshot(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: id}, tc.snapshot)
			} else {
				result, err = blockstorage.UnmanageVolume(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: id})
			}
			count := int32(1)
			if tc.revert {
				count = 2
			}
			if err != nil || result == nil || !result.Completed || result.VolumeID != id || result.Microversion != "3.60" || len(result.Discovery) != int(count-1) || result.Applied == nil || result.Applied.StatusCode != tc.code || !bytes.Equal(result.Applied.Body, reply) || result.Applied.Header.Get("X-Proof") != "opaque current action" || calls.Load() != count || client.Microversion != "3.60" {
				t.Fatal("body text acquired lookup/state/schema checks or acknowledgement changed", result, err, calls.Load())
			}
			if tc.revert && (result.Discovery[0] == nil || result.Discovery[0].StatusCode != 200 || string(result.Discovery[0].Body) != vruSupport || result.Discovery[0].Header.Get("X-Proof") != "support first") {
				t.Fatal("support/action proof phases collapsed", result)
			}
			result.Applied.Body[0] = '!'
			result.Applied.Header.Set("X-Proof", "caller mutation")
			if reply[0] != 0xff || (tc.revert && result.Discovery[0].Header.Get("X-Proof") != "support first") {
				t.Fatal("acknowledgement aliases fixture or support phase", reply, result)
			}
		})
	}
}

func TestRevertVolumeToSnapshotRequiresAdvertised340AndReusesProbeForOrdinarySelection(t *testing.T) {
	for _, tc := range []struct {
		name, selected, reply, version string
		unsupported                    bool
	}{
		{"selected 339 remains unsupported", "3.39", `{"id":"v3.0","min_version":"3.0","max_version":"3.99"}`, "", true},
		{"required equal both bounds", "3.40", `{"id":"v3.0","min_version":"3.40","max_version":"3.40"}`, "3.40", false},
		{"selected above advertised maximum kept raw", "3.90", `{"id":"v3.0","min_version":"3.0","max_version":"3.50"}`, "3.90", false},
		{"required above advertised maximum", "3.90", `{"id":"v3.0","min_version":"3.0","max_version":"3.39"}`, "", true},
		{"required below advertised minimum", "3.90", `{"id":"v3.0","min_version":"3.41","max_version":"3.99"}`, "", true},
		{"missing minimum", "3.90", `{"id":"v3.0","max_version":"3.99"}`, "", true},
		{"missing maximum", "3.90", `{"id":"v3.0","min_version":"3.0"}`, "", true},
		{"unselected cap uses support advertisement", "", `{"id":"v3.0","min_version":"3.0","max_version":"3.99"}`, "3.71", false},
		{"unselected lower maximum uses same support advertisement", "", `{"id":"v3.0","min_version":"3.0","max_version":"3.45"}`, "3.45", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, tc.selected)
			if tc.selected != "" {
				client.MoreHeaders["OpenStack-API-Version"] = "volume " + tc.selected
				client.MoreHeaders["X-OpenStack-Volume-API-Version"] = tc.selected
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				n := calls.Add(1)
				if n == 1 {
					vsaDiscoveryRequest(t, req, vsaVersionPath)
					w.Header().Set("X-Proof", "340 advertisement")
					testcloud.JSON(w, 300, tc.reply)
					return
				}
				if tc.unsupported || n != 2 {
					t.Error("failed support gate posted or repeated discovery", n, req.URL)
					w.WriteHeader(500)
					return
				}
				vmrRequest(t, req, "id", `{"revert":{"snapshot_id":"snapshot"}}`, tc.version)
				w.Header().Set("X-Proof", "340 action")
				w.WriteHeader(203)
				_, _ = w.Write([]byte{0xff, 0x00})
			})
			result, err := blockstorage.RevertVolumeToSnapshot(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, "snapshot")
			if result == nil || result.VolumeID != "id" || len(result.Discovery) != 1 || result.Discovery[0] == nil || result.Discovery[0].StatusCode != 300 || string(result.Discovery[0].Body) != tc.reply || result.Discovery[0].Header.Get("X-Proof") != "340 advertisement" || client.Microversion != tc.selected || client.MoreHeaders["x-source"] != "original" {
				t.Fatal(result, err, calls.Load(), client)
			}
			if tc.selected != "" && (client.MoreHeaders["OpenStack-API-Version"] != "volume "+tc.selected || client.MoreHeaders["X-OpenStack-Volume-API-Version"] != tc.selected || len(client.MoreHeaders) != 3) {
				t.Fatal("versionless support changed original selected headers", client.MoreHeaders)
			}
			if tc.unsupported {
				vsaOperation(t, err, "RevertVolumeToSnapshot")
				var proof *resource.ResponseError
				if !errors.Is(err, resource.ErrUnsupported) || !errors.As(err, &proof) || proof.StatusCode != 300 || string(proof.Body) != tc.reply || proof.Header.Get("X-Proof") != "340 advertisement" || result.Applied != nil || result.Completed || calls.Load() != 1 {
					t.Fatal(result, err, proof, calls.Load())
				}
			} else if err != nil || !result.Completed || result.Microversion != tc.version || result.Applied == nil || result.Applied.StatusCode != 203 || result.Applied.Header.Get("X-Proof") != "340 action" || calls.Load() != 2 {
				t.Fatal("support proof not reused or selected version was incorrectly capped", result, err, calls.Load())
			}
		})
	}
}

func TestVolumeRevertAndUnmanageFailuresKeepOnlyTheActualDiscoveryOrActionProof(t *testing.T) {
	t.Run("malformed admitted support never posts", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := vsaClient(cloud, "3.60")
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = vsaTransport(func(req *http.Request) (*http.Response, error) {
			calls.Add(1)
			vsaDiscoveryRequest(t, req, vsaVersionPath)
			return vsaResponse(req, 200, []byte(`{`), "malformed support"), nil
		})
		result, err := blockstorage.RevertVolumeToSnapshot(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, "snapshot")
		vsaOperation(t, err, "RevertVolumeToSnapshot")
		var proof *resource.ResponseError
		if result == nil || result.Completed || result.Applied != nil || len(result.Discovery) != 1 || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != `{` || proof.Header.Get("X-Proof") != "malformed support" || errors.Is(err, resource.ErrUnsupported) || calls.Load() != 1 {
			t.Fatal(result, err, proof, calls.Load())
		}
	})
	t.Run("clean fallback then native action 404 does not borrow support", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := vsaClient(cloud, "3.60")
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = vsaTransport(func(req *http.Request) (*http.Response, error) {
			switch calls.Add(1) {
			case 1:
				vsaDiscoveryRequest(t, req, vsaVersionPath)
				return vsaResponse(req, 404, []byte("version rejected"), "native discovery"), nil
			case 2:
				vsaDiscoveryRequest(t, req, vsaRootPath)
				return vsaResponse(req, 300, []byte(vruSupport), "accepted root support"), nil
			case 3:
				vmrRequest(t, req, "id", `{"revert":{"snapshot_id":"snapshot"}}`, "3.60")
				return vsaResponse(req, 404, []byte("action rejected"), "native current action"), nil
			default:
				t.Error("native action failure retried discovery or POST", req.URL)
				return vsaResponse(req, 500, nil, "unexpected"), nil
			}
		})
		result, err := blockstorage.RevertVolumeToSnapshot(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, "snapshot")
		vsaOperation(t, err, "RevertVolumeToSnapshot")
		var proof *resource.ResponseError
		var native gophercloud.ErrUnexpectedResponseCode
		if result == nil || result.Completed || result.Applied != nil || len(result.Discovery) != 1 || string(result.Discovery[0].Body) != vruSupport || result.Discovery[0].Header.Get("X-Proof") != "accepted root support" || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "action rejected" || native.ResponseHeader.Get("X-Proof") != "native current action" || errors.As(err, &proof) || calls.Load() != 3 {
			t.Fatal("rejected action borrowed an accepted support response", result, err, native, proof, calls.Load())
		}
	})
	for _, tc := range []struct {
		name           string
		action         vruAction
		faultDiscovery bool
	}{
		{"revert support joined faults", vruActions()[0], true},
		{"revert action joined faults", vruActions()[0], false},
		{"unmanage action joined faults", vruActions()[1], false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.60")
			ctx, cancel := context.WithCancelCause(vsaContext(t))
			defer cancel(nil)
			readCause, closeCause, callerCause := errors.New("lifecycle accepted Read"), errors.New("lifecycle accepted Close"), errors.New("lifecycle custom caller cancellation")
			raw := []byte{0xff, 0x00, 'p'}
			code := 203
			proofName := "faulty action"
			if tc.faultDiscovery {
				raw = []byte(vruSupport)
				code = 300
				proofName = "faulty support"
			}
			var calls, retries, closes atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				retries.Add(1)
				return original
			}
			cloud.Provider.HTTPClient.Transport = vsaTransport(func(req *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if tc.action.required && n == 1 {
					vsaDiscoveryRequest(t, req, vsaVersionPath)
					if !tc.faultDiscovery {
						return vsaResponse(req, 200, []byte(vruSupport), "earlier clean support"), nil
					}
				} else {
					vmrRequest(t, req, "id", tc.action.body, "3.60")
				}
				broken := &vsaFaultBody{reader: bytes.NewReader(raw), readErr: readCause, closeErr: closeCause, onClose: func() { closes.Add(1); client.ResourceBase += "changed/"; cancel(callerCause) }}
				response := vsaResponse(req, code, nil, proofName)
				response.Body = broken
				response.ContentLength = int64(len(raw))
				return response, nil
			})
			result, err := tc.action.call(ctx, client, "id")
			vsaOperation(t, err, tc.action.operation)
			var proof *resource.ResponseError
			if result == nil || result.Completed || !errors.As(err, &proof) || proof.StatusCode != code || !bytes.Equal(proof.Body, raw) || proof.Header.Get("X-Proof") != proofName || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, callerCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption) || retries.Load() != 0 || closes.Load() != 1 {
				t.Fatal("accepted lifecycle failure lost current proof or joined causes", result, err, proof, retries.Load(), closes.Load())
			}
			current := result.Applied
			expectedCalls := int32(1)
			if tc.faultDiscovery {
				if len(result.Discovery) != 1 || result.Applied != nil {
					t.Fatal(result)
				}
				current = result.Discovery[0]
			} else if tc.action.required {
				expectedCalls = 2
				if len(result.Discovery) != 1 || string(result.Discovery[0].Body) != vruSupport || result.Discovery[0].Header.Get("X-Proof") != "earlier clean support" {
					t.Fatal(result)
				}
			} else if len(result.Discovery) != 0 {
				t.Fatal(result)
			}
			if current == nil || current.StatusCode != code || !bytes.Equal(current.Body, raw) || current.Header.Get("X-Proof") != proofName || calls.Load() != expectedCalls {
				t.Fatal(result, err, calls.Load())
			}
			proof.Body[0] = '!'
			proof.Header.Set("X-Proof", "caller error mutation")
			if !bytes.Equal(current.Body, raw) || current.Header.Get("X-Proof") != proofName {
				t.Fatal("physical result aliases error proof", current, proof)
			}
		})
	}
}

func TestVolumeRevertAndUnmanagePreflightAndServiceMethodsPreserveNativeSeparation(t *testing.T) {
	for _, action := range vruActions() {
		for _, kind := range []string{"nil context", "canceled context", "nil client", "nil provider", "wrong role", "reserved auth", "empty ID", "path ID", "escaped ID", "space ID", "invalid UTF8 ID"} {
			t.Run(action.operation+" "+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.60")
				ctx := vsaContext(t)
				id := "id"
				cause := errors.New("lifecycle preflight custom cancellation")
				var want error = resource.ErrInvalidOption
				switch kind {
				case "nil context":
					ctx = nil
				case "canceled context":
					c, cancel := context.WithCancelCause(ctx)
					cancel(cause)
					ctx = c
					want = context.Canceled
				case "nil client":
					client = nil
				case "nil provider":
					client.ProviderClient = nil
				case "wrong role":
					client.Type = "compute"
					want = resource.ErrUnsupported
				case "reserved auth":
					client.MoreHeaders["Authorization"] = "caller auth"
				case "empty ID":
					id = ""
				case "path ID":
					id = "a/b"
				case "escaped ID":
					id = "%2f"
				case "space ID":
					id = "a b"
				case "invalid UTF8 ID":
					id = string([]byte{0xff})
				}
				var calls atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
					calls.Add(1)
					t.Error("invalid lifecycle admission sent HTTP", req.URL)
					w.WriteHeader(500)
				})
				result, err := action.call(ctx, client, id)
				vsaOperation(t, err, action.operation)
				var proof *resource.ResponseError
				if result != nil || !errors.Is(err, want) || errors.As(err, &proof) || calls.Load() != 0 || (kind == "canceled context" && !errors.Is(err, cause)) {
					t.Fatal("preflight acquired physical proof or lost current cause", result, err, proof, calls.Load())
				}
			})
		}
	}
	t.Run("body UTF8 after source admission but before required GET", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := vsaClient(cloud, "3.60")
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) { calls.Add(1); w.WriteHeader(500) })
		bad := string([]byte{0xff})
		result, err := blockstorage.RevertVolumeToSnapshot(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, bad)
		vsaOperation(t, err, "RevertVolumeToSnapshot")
		var proof *resource.ResponseError
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || calls.Load() != 0 {
			t.Fatal(result, err, proof, calls.Load())
		}
		client.Type = "compute"
		result, err = blockstorage.RevertVolumeToSnapshot(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, bad)
		if result != nil || !errors.Is(err, resource.ErrUnsupported) || errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal("body validation hid prior wrong selected service", result, err, calls.Load())
		}
	})
	t.Run("service new actions keep old native Unmanage unchanged", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := vsaClient(cloud, "3.60")
		api := volumes.New(client)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			n := calls.Add(1)
			switch n {
			case 1, 2:
				vmrRequest(t, req, "id", `{"os-unmanage":{}}`, "3.60")
				if n == 1 {
					w.WriteHeader(202)
				} else {
					w.Header().Set("X-Proof", "old native rejection")
					w.WriteHeader(203)
				}
				_, _ = w.Write([]byte("old native opaque"))
			case 3:
				vmrRequest(t, req, "id", `{"os-unmanage":null}`, "3.60")
				w.Header().Set("X-Proof", "SDK null action")
				w.WriteHeader(200)
				_, _ = w.Write([]byte{0xff, 0x00})
			case 4:
				vsaDiscoveryRequest(t, req, vsaVersionPath)
				w.Header().Set("X-Proof", "service support")
				testcloud.JSON(w, 300, vruSupport)
			case 5:
				vmrRequest(t, req, "id", `{"revert":{"snapshot_id":""}}`, "3.60")
				w.Header().Set("X-Proof", "service revert")
				w.WriteHeader(399)
				_, _ = w.Write([]byte{0xff, 0x00})
			default:
				t.Error("unexpected lifecycle service HTTP", n, req.URL)
				w.WriteHeader(500)
			}
		})
		if err := api.Unmanage(vsaContext(t), "id"); err != nil || calls.Load() != 1 {
			t.Fatal("old native202/emptyobject changed", err, calls.Load())
		}
		err := api.Unmanage(vsaContext(t), "id")
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != 203 || len(native.Expected) != 1 || native.Expected[0] != 202 || native.ResponseHeader.Get("X-Proof") != "old native rejection" || calls.Load() != 2 {
			t.Fatal("new SDK policy widened existing native Unmanage", err, native, calls.Load())
		}
		unmanaged, err := api.UnmanageVolume(vsaContext(t), "id")
		if err != nil || unmanaged == nil || !unmanaged.Completed || unmanaged.Applied == nil || unmanaged.Applied.StatusCode != 200 || !bytes.Equal(unmanaged.Applied.Body, []byte{0xff, 0x00}) || unmanaged.Applied.Header.Get("X-Proof") != "SDK null action" || len(unmanaged.Discovery) != 0 || calls.Load() != 3 {
			t.Fatal(unmanaged, err, calls.Load())
		}
		reverted, err := api.RevertVolumeToSnapshot(vsaContext(t), "id", "")
		if err != nil || reverted == nil || !reverted.Completed || reverted.Applied == nil || reverted.Applied.StatusCode != 399 || reverted.Applied.Header.Get("X-Proof") != "service revert" || len(reverted.Discovery) != 1 || reverted.Discovery[0].Header.Get("X-Proof") != "service support" || calls.Load() != 5 {
			t.Fatal(reverted, err, calls.Load())
		}
		var absent *volumes.API
		missing, err := absent.UnmanageVolume(vsaContext(t), "id")
		vsaOperation(t, err, "UnmanageVolume")
		if missing != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 5 {
			t.Fatal(missing, err, calls.Load())
		}
		missing, err = absent.RevertVolumeToSnapshot(vsaContext(t), "id", "")
		vsaOperation(t, err, "RevertVolumeToSnapshot")
		if missing != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 5 {
			t.Fatal(missing, err, calls.Load())
		}
	})
}

func vruPost(t *testing.T, req *http.Request, id, body, version, token string) {
	t.Helper()
	var actual []byte
	var err error
	if req.Body != nil {
		actual, err = io.ReadAll(req.Body)
	}
	if err != nil || string(actual) != body || req.Method != http.MethodPost || req.URL.EscapedPath() != vsaBase+"volumes/"+url.PathEscape(id)+"/action" || req.URL.RawQuery != "" || req.Header.Get("X-Source") != "original" || req.Header.Get("X-Auth-Token") != token || req.Header.Get("OpenStack-API-Version") != "volume "+version || req.Header.Get("X-OpenStack-Volume-API-Version") != version || req.ContentLength != int64(len(body)) || len(req.TransferEncoding) != 0 {
		t.Error("lifecycle retry changed fixed route, exact JSON, captured policy or live token", req.URL, string(actual), req.Header, req.ContentLength, req.TransferEncoding, err)
	}
}

func TestVolumeRevertAndUnmanageLiveAuthAndBoundedRetryPreserveBodyAndOriginalHTTPPolicy(t *testing.T) {
	for _, action := range vruActions() {
		for _, kind := range []string{"successful retry", "expanded rejection", "changed retry body"} {
			t.Run(action.operation+" "+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vsaClient(cloud, "3.90")
				var calls, posts, reauths, retries atomic.Int32
				cloud.Provider.ReauthFunc = func(context.Context) error {
					if reauths.Add(1) > 1 {
						return errors.New("bounded lifecycle reauth exhausted")
					}
					cloud.Provider.SetToken("reauth-token")
					client.MoreHeaders["x-source"] = "later ordinary header"
					return nil
				}
				cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, options *gophercloud.RequestOpts, original error, _ uint) error {
					if !gophercloud.ResponseCodeIs(original, 503) || retries.Add(1) > 1 {
						return original
					}
					raw, ok := options.JSONBody.(json.RawMessage)
					if !ok || string(raw) != action.body || options.RawBody != nil || options.JSONResponse != nil || !options.KeepResponseBody {
						t.Error("native retry changed SDK request ownership", options, string(raw))
					}
					if kind == "changed retry body" {
						options.JSONBody = json.RawMessage(`{"changed":null}`)
						return nil
					}
					options.JSONBody = json.RawMessage(bytes.Clone(raw))
					if len(raw) != 0 {
						raw[0] = '!'
					}
					options.MoreHeaders["X-Native"] = "owned retry header"
					if kind == "expanded rejection" {
						options.OkCodes = append(options.OkCodes, 400)
					}
					cloud.Provider.SetToken("retry-token")
					return nil
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
					n := calls.Add(1)
					if action.required && n == 1 {
						vsaDiscoveryRequest(t, req, vsaVersionPath)
						w.Header().Set("X-Proof", "earlier auth-independent support")
						testcloud.JSON(w, 200, vruSupport)
						return
					}
					post := posts.Add(1)
					vruPost(t, req, "id", action.body, "3.90", map[int32]string{1: "test-token", 2: "reauth-token", 3: "retry-token"}[post])
					w.Header().Set("X-Proof", "current action attempt")
					switch post {
					case 1:
						testcloud.JSON(w, 401, `{"error":"expired"}`)
					case 2:
						testcloud.JSON(w, 503, `{"error":"retry once"}`)
					case 3:
						if req.Header.Get("X-Native") != "owned retry header" {
							t.Error(req.Header)
						}
						if kind == "expanded rejection" {
							testcloud.JSON(w, 400, `{"error":"original SDK policy"}`)
						} else {
							w.WriteHeader(203)
							_, _ = w.Write([]byte{0xff, 0x00})
						}
					default:
						t.Error("lifecycle action replayed beyond bounded native hooks", post, req.URL)
						w.WriteHeader(500)
					}
				})
				result, err := action.call(vsaContext(t), client, "id")
				expectedPosts := int32(3)
				if kind == "changed retry body" {
					expectedPosts = 2
				}
				expectedCalls := expectedPosts
				if action.required {
					expectedCalls++
				}
				if result == nil || result.VolumeID != "id" || result.Microversion != "3.90" || len(result.Discovery) != int(expectedCalls-expectedPosts) || calls.Load() != expectedCalls || posts.Load() != expectedPosts || reauths.Load() != 1 || retries.Load() != 1 || client.Microversion != "3.90" || client.MoreHeaders["x-source"] != "later ordinary header" {
					t.Fatal(result, err, calls.Load(), posts.Load(), reauths.Load(), retries.Load(), client)
				}
				if action.required && (result.Discovery[0] == nil || string(result.Discovery[0].Body) != vruSupport || result.Discovery[0].Header.Get("X-Proof") != "earlier auth-independent support") {
					t.Fatal("action retry replaced support proof", result)
				}
				if kind == "successful retry" {
					if err != nil || !result.Completed || result.Applied == nil || result.Applied.StatusCode != 203 || !bytes.Equal(result.Applied.Body, []byte{0xff, 0x00}) || result.Applied.Header.Get("X-Proof") != "current action attempt" {
						t.Fatal(result, err)
					}
					return
				}
				vsaOperation(t, err, action.operation)
				var native gophercloud.ErrUnexpectedResponseCode
				var proof *resource.ResponseError
				code := 400
				if kind == "changed retry body" {
					code = 503
				}
				if result.Completed || result.Applied != nil || !errors.As(err, &native) || native.Actual != code || native.ResponseHeader.Get("X-Proof") != "current action attempt" || errors.As(err, &proof) || (kind == "changed retry body" && !errors.Is(err, resource.ErrInvalidOption)) {
					t.Fatal("retry/body/status mutation became acknowledgement or borrowed support", result, err, native, proof)
				}
				if kind == "expanded rejection" && (len(native.Expected) != 300 || native.Expected[0] != 100 || native.Expected[299] != 399 || string(native.Body) != `{"error":"original SDK policy"}`) {
					t.Fatal("expanded native OkCodes changed original error policy", native)
				}
				if kind == "changed retry body" && string(native.Body) != `{"error":"retry once"}` {
					t.Fatal("body guard lost native503 evidence", native)
				}
			})
		}
	}
}
