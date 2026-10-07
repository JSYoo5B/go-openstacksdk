package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const vsaBase = "/proxy/cinder/v3/state-project/"
const vsaVersionPath = "/catalog/proxy%20prefix/v3/"
const vsaRootPath = "/catalog/proxy%20prefix/"

type vsaAction struct {
	name, key string
	call      func(context.Context, *gophercloud.ServiceClient, blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error)
}

func vsaActions() []vsaAction {
	return []vsaAction{
		{"ReserveVolume", "os-reserve", blockstorage.ReserveVolume},
		{"UnreserveVolume", "os-unreserve", blockstorage.UnreserveVolume},
		{"BeginVolumeDetaching", "os-begin_detaching", blockstorage.BeginVolumeDetaching},
		{"AbortVolumeDetaching", "os-roll_detaching", blockstorage.AbortVolumeDetaching},
	}
}
func vsaContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}
func vsaClient(cloud *testcloud.Cloud, version string) *gophercloud.ServiceClient {
	client := cloud.Client("volume", "/catalog/proxy%20prefix/v3/catalog-project/")
	client.ResourceBase = cloud.Server.URL + vsaBase
	client.Microversion = version
	client.MoreHeaders = map[string]string{"x-source": "original"}
	return client
}
func vsaPayload(t *testing.T, req *http.Request, id, key, version, token string) []byte {
	t.Helper()
	current := ""
	if version != "" {
		current = "volume " + version
	}
	want := []byte(fmt.Sprintf(`{"%s":null}`, key))
	var body []byte
	var err error
	if req.Body != nil {
		body, err = io.ReadAll(req.Body)
	}
	if err != nil || !bytes.Equal(body, want) || req.Method != http.MethodPost || req.URL.EscapedPath() != vsaBase+"volumes/"+url.PathEscape(id)+"/action" || req.URL.RawQuery != "" || req.Header.Get("X-Source") != "original" || req.Header.Get("X-Auth-Token") != token || req.Header.Get("OpenStack-API-Version") != current || req.Header.Get("X-OpenStack-Volume-API-Version") != version || req.ContentLength != int64(len(want)) || len(req.TransferEncoding) != 0 {
		t.Error("state action changed literal body, route, framing, captured header or version", req.Method, req.URL, string(body), req.Header, req.ContentLength, req.TransferEncoding, err)
	}
	return body
}
func vsaDiscoveryRequest(t *testing.T, req *http.Request, path string) {
	t.Helper()
	var body []byte
	var err error
	if req.Body != nil {
		body, err = io.ReadAll(req.Body)
	}
	if err != nil || len(body) != 0 || req.Method != http.MethodGet || req.URL.EscapedPath() != path || req.URL.RawQuery != "" || req.Header.Get("OpenStack-API-Version") != "" || req.Header.Get("X-OpenStack-Volume-API-Version") != "" || req.Header.Get("X-Source") != "original" || req.Header.Get("X-Auth-Token") != "test-token" || req.ContentLength != 0 || len(req.TransferEncoding) != 0 {
		t.Error("discovery changed captured bodyless route/header policy", req.Method, req.URL, req.Header, string(body), err)
	}
}
func vsaOperation(t *testing.T, err error, name string) {
	t.Helper()
	var operation *resource.OperationError
	if err == nil || !errors.As(err, &operation) || operation.Operation != name || operation.Resource != "volumes" || operation.Cause == nil {
		t.Fatal("state operation/cause changed", err, operation, name)
	}
}

type vsaTransport func(*http.Request) (*http.Response, error)

func (f vsaTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
func vsaResponse(req *http.Request, code int, body []byte, proof string) *http.Response {
	return &http.Response{StatusCode: code, Status: fmt.Sprintf("%d fixture", code), Header: http.Header{"X-Proof": []string{proof}}, Body: io.NopCloser(bytes.NewReader(body)), ContentLength: int64(len(body)), Request: req}
}

type vsaFaultBody struct {
	reader            *bytes.Reader
	readErr, closeErr error
	onClose           func()
}

func (b *vsaFaultBody) Read(dst []byte) (int, error) {
	n, err := b.reader.Read(dst)
	if err == io.EOF && b.readErr != nil {
		return n, b.readErr
	}
	return n, err
}
func (b *vsaFaultBody) Close() error {
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}
func vsaServiceCall(ctx context.Context, api *volumes.API, name, id string) (*volumes.VolumeActionResult, error) {
	switch name {
	case "ReserveVolume":
		return api.ReserveVolume(ctx, id)
	case "UnreserveVolume":
		return api.UnreserveVolume(ctx, id)
	case "BeginVolumeDetaching":
		return api.BeginVolumeDetaching(ctx, id)
	case "AbortVolumeDetaching":
		return api.AbortVolumeDetaching(ctx, id)
	}
	panic("unknown fixture action")
}

func TestVolumeStateActionsPostSourceNullBodiesAndAcceptOpaqueHTTPReplies(t *testing.T) {
	for index, action := range vsaActions() {
		t.Run(action.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.60")
			id := "한글-é"
			code := []int{200, 203, 302, 399}[index]
			reply := []byte{0xff, 0x00, '{', '!'}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				vsaPayload(t, req, id, action.key, "3.60", "test-token")
				w.Header().Set("X-Proof", "opaque")
				w.WriteHeader(code)
				_, _ = w.Write(reply)
			})
			result, err := action.call(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: id})
			if err != nil || result == nil || !result.Completed || result.VolumeID != id || result.Microversion != "3.60" || len(result.Discovery) != 0 || result.Applied == nil || result.Applied.StatusCode != code || result.Applied.Header.Get("X-Proof") != "opaque" || !bytes.Equal(result.Applied.Body, reply) || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			result.Applied.Body[0] = '!'
			result.Applied.Header["X-Proof"][0] = "caller changed first"
			second, err := action.call(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: id})
			if err != nil || second == nil || !second.Completed || second.Applied == nil || !bytes.Equal(second.Applied.Body, reply) || second.Applied.Header.Get("X-Proof") != "opaque" || calls.Load() != 2 || reply[0] != 0xff || client.Microversion != "3.60" {
				t.Fatal("proof aliases another call or client", second, err, calls.Load(), reply, client)
			}
		})
	}
}

func TestVolumeStateActionAdmissionUsesOriginalSub400PolicyAndKeepsNativeRejections(t *testing.T) {
	for _, code := range []int{100, 199, 204, 304, 399, 400, 403, 404, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.60")
			action := vsaActions()[0]
			raw := []byte{0xff, 'a', 0x00}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = vsaTransport(func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				vsaPayload(t, req, "literal-name", action.key, "3.60", "test-token")
				return vsaResponse(req, code, raw, "status"), nil
			})
			result, err := action.call(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "literal-name"})
			if result == nil || result.VolumeID != "literal-name" || result.Microversion != "3.60" || len(result.Discovery) != 0 || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			if code < 400 {
				if err != nil || !result.Completed || result.Applied == nil || result.Applied.StatusCode != code || !bytes.Equal(result.Applied.Body, raw) {
					t.Fatal(result, err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				var proof *resource.ResponseError
				vsaOperation(t, err, action.name)
				if result.Completed || result.Applied != nil || !errors.As(err, &native) || native.Actual != code || native.ResponseHeader.Get("X-Proof") != "status" || !bytes.Equal(native.Body, raw) || errors.As(err, &proof) {
					t.Fatal(result, err, native, proof)
				}
			}
		})
	}
	t.Run("native expanded codes do not expand SDK admission", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := vsaClient(cloud, "3.60")
		var calls, retries atomic.Int32
		cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, original error, _ uint) error {
			if !gophercloud.ResponseCodeIs(original, 503) || retries.Add(1) > 1 {
				return original
			}
			opts.OkCodes = append(opts.OkCodes, 400)
			return nil
		}
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			call := calls.Add(1)
			vsaPayload(t, req, "id", "os-reserve", "3.60", "test-token")
			w.Header().Set("X-Proof", "actual rejected")
			if call == 1 {
				w.WriteHeader(503)
			} else {
				w.WriteHeader(400)
			}
			_, _ = w.Write([]byte("native policy"))
		})
		result, err := blockstorage.ReserveVolume(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"})
		var native gophercloud.ErrUnexpectedResponseCode
		var proof *resource.ResponseError
		if result == nil || result.Completed || result.Applied != nil || !errors.As(err, &native) || native.Actual != 400 || native.ResponseHeader.Get("X-Proof") != "actual rejected" || errors.As(err, &proof) || calls.Load() != 2 || retries.Load() != 1 {
			t.Fatal(result, err, native, proof, calls.Load(), retries.Load())
		}
	})
}

func TestVolumeStateActionsAcceptedFaultsKeepActualProofAndAllCauses(t *testing.T) {
	for index, kind := range []string{"read", "close", "read and close", "close cancellation and source drift"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.60")
			action := vsaActions()[index]
			ctx, cancel := context.WithCancelCause(vsaContext(t))
			defer cancel(nil)
			readCause, closeCause, callerCause := errors.New("state response read"), errors.New("state response Close"), errors.New("state caller canceled")
			raw := []byte{0xff, 0x00, 'p'}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = vsaTransport(func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				vsaPayload(t, req, "id", action.key, "3.60", "test-token")
				body := &vsaFaultBody{reader: bytes.NewReader(raw)}
				if kind == "read" || kind == "read and close" {
					body.readErr = readCause
				}
				if kind != "read" {
					body.closeErr = closeCause
				}
				if kind == "close cancellation and source drift" {
					body.onClose = func() { client.ResourceBase += "changed/"; cancel(callerCause) }
				}
				response := vsaResponse(req, 203, nil, "fault")
				response.Body = body
				response.ContentLength = int64(len(raw))
				return response, nil
			})
			result, err := action.call(ctx, client, blockstorage.VolumeActionRequest{VolumeID: "id"})
			var proof *resource.ResponseError
			vsaOperation(t, err, action.name)
			if result == nil || result.Completed || result.Applied == nil || result.Applied.StatusCode != 203 || result.Applied.Header.Get("X-Proof") != "fault" || !bytes.Equal(result.Applied.Body, raw) || !errors.As(err, &proof) || proof.StatusCode != 203 || proof.Header.Get("X-Proof") != "fault" || !bytes.Equal(proof.Body, raw) || len(result.Discovery) != 0 || calls.Load() != 1 {
				t.Fatal(result, err, proof, calls.Load())
			}
			if (kind == "read" || kind == "read and close") && !errors.Is(err, readCause) {
				t.Fatal("lost Read cause", err)
			}
			if kind != "read" && !errors.Is(err, closeCause) {
				t.Fatal("lost Close cause", err)
			}
			if kind == "close cancellation and source drift" && (!errors.Is(err, context.Canceled) || !errors.Is(err, callerCause) || !errors.Is(err, resource.ErrInvalidOption)) {
				t.Fatal("lost cancellation/source causes", err)
			}
			result.Applied.Body[0] = '!'
			result.Applied.Header["X-Proof"][0] = "mutated"
			if !bytes.Equal(proof.Body, raw) || proof.Header.Get("X-Proof") != "fault" || raw[0] != 0xff {
				t.Fatal("phase and error proof alias", proof, raw)
			}
		})
	}
}

func TestVolumeStateActionsRejectInvalidIDAndSourceBeforeAnyHTTP(t *testing.T) {
	cases := []string{"empty", "dot", "dot dot", "slash", "backslash", "escaped", "query", "fragment", "space", "Unicode space", "control", "invalid UTF8", "nil client", "nil provider", "nil context", "custom cancellation", "wrong role", "reserved auth", "foreign base"}
	for _, action := range vsaActions() {
		t.Run(action.name, func(t *testing.T) {
			for _, kind := range cases {
				t.Run(kind, func(t *testing.T) {
					cloud := testcloud.New(t)
					client := vsaClient(cloud, "3.60")
					ctx := vsaContext(t)
					id := "id"
					cause := errors.New("state custom cancellation")
					want := resource.ErrInvalidOption
					switch kind {
					case "empty":
						id = ""
					case "dot":
						id = "."
					case "dot dot":
						id = ".."
					case "slash":
						id = "a/b"
					case "backslash":
						id = `a\b`
					case "escaped":
						id = "a%2Fb"
					case "query":
						id = "a?b"
					case "fragment":
						id = "a#b"
					case "space":
						id = "a b"
					case "Unicode space":
						id = "a\u2003b"
					case "control":
						id = "a\x00b"
					case "invalid UTF8":
						id = string([]byte{0xff})
					case "nil client":
						client = nil
					case "nil provider":
						client.ProviderClient = nil
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
						client.MoreHeaders["Authorization"] = "caller"
					case "foreign base":
						client.ResourceBase = "https://foreign.invalid/v3/"
					}
					var calls atomic.Int32
					cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) { calls.Add(1); w.WriteHeader(500) })
					result, err := action.call(ctx, client, blockstorage.VolumeActionRequest{VolumeID: id})
					var proof *resource.ResponseError
					vsaOperation(t, err, action.name)
					if result != nil || !errors.Is(err, want) || errors.As(err, &proof) || calls.Load() != 0 {
						t.Fatal(result, err, proof, calls.Load())
					}
					if kind == "custom cancellation" && !errors.Is(err, cause) {
						t.Fatal("lost preflight cause", err)
					}
				})
			}
		})
	}
}

func TestVolumeStateActionsSelectedVersionKeepsOwnedHeadersAndLiveNativeAuthentication(t *testing.T) {
	for _, version := range []string{"3.80", "latest"} {
		t.Run(version, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, version)
			client.MoreHeaders["OpenStack-API-Version"] = "volume " + version
			client.MoreHeaders["X-OpenStack-Volume-API-Version"] = version
			var calls, reauths, retries atomic.Int32
			cloud.Provider.ReauthFunc = func(context.Context) error {
				if reauths.Add(1) > 1 {
					return errors.New("bounded state reauth exhausted")
				}
				cloud.Provider.SetToken("reauth-token")
				client.MoreHeaders["x-source"] = "later caller header"
				return nil
			}
			cloud.Provider.RetryFunc = func(_ context.Context, method, target string, opts *gophercloud.RequestOpts, original error, _ uint) error {
				if !gophercloud.ResponseCodeIs(original, 503) {
					return original
				}
				if retries.Add(1) > 1 {
					return errors.Join(original, errors.New("bounded state retry exhausted"))
				}
				originalBody, ok := opts.JSONBody.(json.RawMessage)
				if !ok || string(originalBody) != `{"os-roll_detaching":null}` || method != http.MethodPost || !strings.HasSuffix(target, vsaBase+"volumes/id/action") || opts.RawBody != nil || opts.JSONResponse != nil || !opts.KeepResponseBody {
					t.Error("native retry changed ownership", method, target, opts)
				}
				opts.JSONBody = json.RawMessage(bytes.Clone(originalBody))
				if len(originalBody) != 0 {
					originalBody[0] = '!'
				}
				cloud.Provider.SetToken("retry-token")
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				call := calls.Add(1)
				token := map[int32]string{1: "test-token", 2: "reauth-token", 3: "retry-token"}[call]
				vsaPayload(t, req, "id", "os-roll_detaching", version, token)
				switch call {
				case 1:
					testcloud.JSON(w, 401, `{"error":"expired"}`)
				case 2:
					testcloud.JSON(w, 503, `{"error":"retry once"}`)
				case 3:
					w.Header().Set("X-Proof", "live auth")
					w.WriteHeader(203)
					_, _ = w.Write([]byte("opaque"))
				default:
					t.Error("state action replayed", req.URL)
					w.WriteHeader(500)
				}
			})
			result, err := blockstorage.AbortVolumeDetaching(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"})
			if err != nil || result == nil || !result.Completed || result.Microversion != version || len(result.Discovery) != 0 || result.Applied == nil || string(result.Applied.Body) != "opaque" || result.Applied.Header.Get("X-Proof") != "live auth" || calls.Load() != 3 || reauths.Load() != 1 || retries.Load() != 1 || client.Microversion != version || client.MoreHeaders["x-source"] != "later caller header" {
				t.Fatal(result, err, calls.Load(), reauths.Load(), retries.Load(), client)
			}
		})
	}
}

func TestVolumeStateActionsNegotiateFiniteVolumeCapWithoutChangingBackupCap(t *testing.T) {
	cases := []struct {
		name, first, second, version       string
		firstCode, secondCode, pages, gets int
	}{
		{"volume cap", `{"version":{"id":"v3.0","status":"CURRENT","max_version":"3.99","min_version":"3.0"}}`, "", "3.71", 300, 0, 1, 1},
		{"server below cap", `{"id":"v3.0","max_version":"3.70"}`, "", "3.70", 200, 0, 1, 1},
		{"minimum above cap", `{"id":"v3.0","max_version":"3.99","min_version":"3.72"}`, "", "", 200, 0, 1, 1},
		{"missing maximum stops", `{"id":"v3.0","min_version":"bad"}`, "", "", 200, 0, 1, 1},
		{"accepted empty then useful", `{}`, `{"versions":{"values":[{"id":"v3.0","status":"SUPPORTED","max_version":"3.99"}]}}`, "3.71", 200, 300, 2, 2},
		{"clean unavailable finite fallback", `unavailable`, `unavailable`, "", 404, 405, 0, 2},
		{"v2 then missing max", `{"versions":[{"id":"v2.0","max_version":"2.99"}]}`, `{"id":"v3.0"}`, "", 200, 200, 2, 2},
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "")
			action := vsaActions()[index%4]
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				call := calls.Add(1)
				if req.Method == http.MethodGet {
					path := vsaVersionPath
					reply, code := tc.first, tc.firstCode
					if call == 2 {
						path = vsaRootPath
						reply, code = tc.second, tc.secondCode
					}
					vsaDiscoveryRequest(t, req, path)
					w.Header().Set("X-Proof", fmt.Sprintf("discovery%d", call))
					testcloud.JSON(w, code, reply)
					return
				}
				if call != int32(tc.gets+1) {
					t.Error("state discovery replayed or followed extra candidates", call, req.URL)
				}
				vsaPayload(t, req, "id", action.key, tc.version, "test-token")
				w.Header().Set("X-Proof", "action")
				w.WriteHeader(203)
				_, _ = w.Write([]byte("ack"))
			})
			result, err := action.call(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"})
			if err != nil || result == nil || !result.Completed || result.Microversion != tc.version || result.Applied == nil || result.Applied.StatusCode != 203 || result.Applied.Header.Get("X-Proof") != "action" || string(result.Applied.Body) != "ack" || len(result.Discovery) != tc.pages || calls.Load() != int32(tc.gets+1) || client.Microversion != "" || len(client.MoreHeaders) != 1 {
				t.Fatal(result, err, calls.Load(), client)
			}
			for _, page := range result.Discovery {
				if page == nil || (page.StatusCode != 200 && page.StatusCode != 300) || page.Header.Get("X-Proof") == "action" || bytes.Equal(page.Body, result.Applied.Body) {
					t.Fatal("discovery/action physical proof merged", result)
				}
			}
		})
	}
	t.Run("same server advertisement retains backup ceiling", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := vsaClient(cloud, "")
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			call := calls.Add(1)
			if call == 1 || call == 3 {
				vsaDiscoveryRequest(t, req, vsaVersionPath)
				testcloud.JSON(w, 200, `{"id":"v3.0","max_version":"3.99"}`)
				return
			}
			if call == 2 {
				vsaPayload(t, req, "id", "os-reserve", "3.71", "test-token")
				w.WriteHeader(203)
				return
			}
			var body []byte
			var err error
			if req.Body != nil {
				body, err = io.ReadAll(req.Body)
			}
			if call != 4 || req.Method != http.MethodPost || req.URL.EscapedPath() != vsaBase+"backups/import_record" || req.URL.RawQuery != "" || req.Header.Get("OpenStack-API-Version") != "volume 3.64" || req.Header.Get("X-OpenStack-Volume-API-Version") != "3.64" || err != nil || string(body) != `{"backup-record":{"backup_service":"driver","backup_url":"opaque"}}` {
				t.Error("volume cap leaked into backup", req.URL, req.Header, string(body), err, call)
			}
			testcloud.JSON(w, 201, `{"backup":{"id":"imported"}}`)
		})
		volume, err := blockstorage.ReserveVolume(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"})
		if err != nil || volume == nil || !volume.Completed || volume.Microversion != "3.71" || calls.Load() != 2 {
			t.Fatal(volume, err, calls.Load())
		}
		backup, err := blockstorage.ImportVolumeBackup(vsaContext(t), client, blockstorage.ImportVolumeBackupRequest{BackupService: "driver", BackupURL: "opaque"})
		if err != nil || backup == nil || backup.Value == nil || backup.Microversion != "3.64" || string(backup.BackupID) != `"imported"` || len(backup.Discovery) != 1 || calls.Load() != 4 || client.Microversion != "" {
			t.Fatal(backup, err, calls.Load(), client)
		}
	})
}

func TestVolumeStateActionsDiscoveryFailuresRetainCurrentStageAndNeverBorrowProof(t *testing.T) {
	cases := []struct {
		name, first, second                string
		firstCode, secondCode, pages, gets int
		native                             bool
	}{
		{"malformed accepted first", `{`, "", 200, 0, 1, 1, false},
		{"accepted null", `null`, "", 300, 0, 1, 1, false},
		{"invalid accepted UTF8", string([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}), "", 200, 0, 1, 1, false},
		{"later native forbidden", `{}`, `denied`, 200, 403, 1, 2, true},
		{"later accepted malformed", `{}`, `[`, 200, 300, 2, 2, false},
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "")
			action := vsaActions()[index%4]
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				call := calls.Add(1)
				path, code, reply := vsaVersionPath, tc.firstCode, tc.first
				if call == 2 {
					path, code, reply = vsaRootPath, tc.secondCode, tc.second
				}
				if call > int32(tc.gets) || req.Method != http.MethodGet {
					t.Error("failed discovery continued to action", call, req.URL)
					w.WriteHeader(500)
					return
				}
				vsaDiscoveryRequest(t, req, path)
				w.Header().Set("X-Proof", fmt.Sprintf("current%d", call))
				testcloud.JSON(w, code, reply)
			})
			result, err := action.call(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"})
			vsaOperation(t, err, action.name)
			var proof *resource.ResponseError
			if result == nil || result.Completed || result.Applied != nil || result.VolumeID != "id" || len(result.Discovery) != tc.pages || calls.Load() != int32(tc.gets) {
				t.Fatal(result, err, calls.Load())
			}
			if tc.native {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != tc.secondCode || string(native.Body) != tc.second || native.ResponseHeader.Get("X-Proof") != "current2" || errors.As(err, &proof) || string(result.Discovery[0].Body) != tc.first {
					t.Fatal("native rejection borrowed earlier proof", result, err, native, proof)
				}
			} else {
				expected := tc.first
				code := tc.firstCode
				if tc.gets == 2 {
					expected, code = tc.second, tc.secondCode
				}
				if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != expected || proof.Header.Get("X-Proof") != fmt.Sprintf("current%d", tc.gets) || string(result.Discovery[len(result.Discovery)-1].Body) != expected {
					t.Fatal(result, err, proof)
				}
			}
		})
	}
}

func TestVolumeStateActionServiceMethodsPreserveNativeBodiesCodesAndNilAPILabels(t *testing.T) {
	for _, action := range vsaActions() {
		t.Run(action.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.60")
			api := volumes.New(client)
			var calls atomic.Int32
			hasNative := action.name != "AbortVolumeDetaching"
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				call := calls.Add(1)
				if !hasNative || call == 3 {
					vsaPayload(t, req, "id", action.key, "3.60", "test-token")
					w.Header().Set("X-Proof", "new null action")
					w.WriteHeader(203)
					_, _ = w.Write([]byte{0xff, 0x00})
					return
				}
				raw, err := io.ReadAll(req.Body)
				if err != nil || req.Method != http.MethodPost || req.URL.EscapedPath() != vsaBase+"volumes/id/action" || req.URL.RawQuery != "" || string(raw) != fmt.Sprintf(`{"%s":{}}`, action.key) {
					t.Error("native request policy changed", req.URL, string(raw), err)
				}
				code := 200
				if action.name == "BeginVolumeDetaching" {
					code = 202
				}
				if call == 2 {
					code = 203
				}
				w.WriteHeader(code)
			})
			if hasNative {
				nativeCall := func() error {
					switch action.name {
					case "ReserveVolume":
						return api.Reserve(vsaContext(t), "id")
					case "UnreserveVolume":
						return api.Unreserve(vsaContext(t), "id")
					case "BeginVolumeDetaching":
						return api.BeginDetaching(vsaContext(t), "id")
					}
					panic("unknown native fixture")
				}
				if err := nativeCall(); err != nil || calls.Load() != 1 {
					t.Fatal(err, calls.Load())
				}
				err := nativeCall()
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 203 || calls.Load() != 2 {
					t.Fatal("native accepted source-only status", err, native, calls.Load())
				}
			}
			result, err := vsaServiceCall(vsaContext(t), api, action.name, "id")
			wantCalls := int32(1)
			if hasNative {
				wantCalls = 3
			}
			if err != nil || result == nil || !result.Completed || result.Applied == nil || result.Applied.StatusCode != 203 || !bytes.Equal(result.Applied.Body, []byte{0xff, 0x00}) || result.Applied.Header.Get("X-Proof") != "new null action" || len(result.Discovery) != 0 || calls.Load() != wantCalls {
				t.Fatal(result, err, calls.Load())
			}
			var nilAPI *volumes.API
			absent, err := vsaServiceCall(vsaContext(t), nilAPI, action.name, "id")
			vsaOperation(t, err, action.name)
			var proof *resource.ResponseError
			if absent != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || calls.Load() != wantCalls {
				t.Fatal(absent, err, proof, calls.Load())
			}
		})
	}
}
