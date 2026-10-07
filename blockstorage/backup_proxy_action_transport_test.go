package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type backupProxyActionTransportOutcome struct {
	applied   *blockstorage.VolumeBackupMutationPage
	value     json.RawMessage
	completed bool
	restore   *blockstorage.RestoreVolumeBackupResult
	reset     *blockstorage.ResetVolumeBackupStatusResult
}

func backupProxyActionTransportCall(ctx context.Context, client *gophercloud.ServiceClient, family string) (*backupProxyActionTransportOutcome, error) {
	if family == "restore" {
		result, err := blockstorage.RestoreVolumeBackup(ctx, client, blockstorage.RestoreVolumeBackupRequest{BackupID: "literal"}, blockstorage.WithRestoreVolumeBackupName("new"))
		if result == nil {
			return nil, err
		}
		return &backupProxyActionTransportOutcome{applied: result.Applied, value: result.Value, restore: result}, err
	}
	result, err := blockstorage.ResetVolumeBackupStatus(ctx, client, blockstorage.ResetVolumeBackupStatusRequest{BackupID: "literal", Status: "custom"})
	if result == nil {
		return nil, err
	}
	return &backupProxyActionTransportOutcome{applied: result.Applied, completed: result.Completed, reset: result}, err
}
func backupProxyActionTransportPolicy(family string) (string, string, string, string) {
	if family == "restore" {
		return "RestoreVolumeBackup", bmwCollection + "/literal/restore", "3.60", `{"restore":{"name":"new"}}`
	}
	return "ResetVolumeBackupStatus", bmwCollection + "/literal/action", "3.64", `{"os-reset_status":{"status":"custom"}}`
}
func backupProxyActionTransportWire(t *testing.T, req *http.Request, family, token string) {
	t.Helper()
	_, path, version, body := backupProxyActionTransportPolicy(family)
	actual := bmwRequest(t, req, http.MethodPost, path, version, token)
	if string(actual) != body || req.ContentLength != int64(len(body)) || len(req.TransferEncoding) != 0 || req.URL.RawQuery != "" || req.Header.Get("X-Openstack-Volume-Api-Version") != version {
		t.Error("action payload/framing/version changed", string(actual), req.ContentLength, req.TransferEncoding, req.URL, req.Header)
	}
	if req.Body != nil {
		if err := req.Body.Close(); err != nil {
			t.Error(err)
		}
	}
}
func backupProxyActionTransportNoAdmission(t *testing.T, family string, out *backupProxyActionTransportOutcome, err error) {
	t.Helper()
	var accepted *resource.ResponseError
	if out == nil || out.applied != nil || out.value != nil || out.completed || err == nil || errors.As(err, &accepted) {
		t.Fatal(out, err, accepted)
	}
	operation, _, _, _ := backupProxyActionTransportPolicy(family)
	bmwOperation(t, err, operation)
}

func TestBackupProxyActionTransportAcceptedFaultsKeepCurrentProofAndEveryCause(t *testing.T) {
	for _, family := range []string{"restore", "reset"} {
		for _, fault := range []string{"read", "close", "cancel", "provider", "endpoint", "resource base", "type", "microversion", "joined"} {
			t.Run(family+"/"+fault, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := bmwClient(cloud)
				ctx, cancel := context.WithCancelCause(bmwContext(t))
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("action response Read"), errors.New("action response Close"), errors.New("action parent canceled")
				reply := []byte{0xff, 0, 'c', 'u', 'r', 'r', 'e', 'n', 't'}
				broken := &snapshotReadTransportBody{data: bytes.NewReader(reply)}
				wantRead, wantClose, wantCancel := fault == "read" || fault == "joined", fault == "close" || fault == "joined", fault == "cancel" || fault == "joined"
				wantSource := fault != "read" && fault != "close" && fault != "cancel"
				if wantRead {
					broken.readError = readCause
				}
				if wantClose {
					broken.closeError = closeCause
				}
				broken.onClose = func() {
					if wantSource {
						fact := fault
						if fact == "joined" {
							fact = "resource base"
						}
						snapshotReadTransportMutate(client, fact)
					}
					if wantCancel {
						cancel(cancelCause)
					}
				}
				var calls, retries atomic.Int32
				cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries.Add(1)
					return errors.New("accepted IO cannot retry")
				}
				cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(req *http.Request) (*http.Response, error) {
					calls.Add(1)
					backupProxyActionTransportWire(t, req, family, "test-token")
					return snapshotReadTransportResponse(req, 203, broken, "current action"), nil
				})
				out, err := backupProxyActionTransportCall(ctx, client, family)
				var accepted *resource.ResponseError
				if err == nil || out == nil || out.applied == nil || out.value != nil || out.completed || out.restore != nil && out.restore.Backup != nil || !errors.As(err, &accepted) || accepted.StatusCode != 203 || !bytes.Equal(accepted.Body, reply) || accepted.Header.Get("X-Proof") != "current action" || calls.Load() != 1 || retries.Load() != 0 || broken.closes.Load() != 1 {
					t.Fatal(out, err, accepted, calls.Load(), retries.Load(), broken.closes.Load())
				}
				bmwPage(t, out.applied, 203, string(reply), "current action")
				if wantRead && !errors.Is(err, readCause) || wantClose && !errors.Is(err, closeCause) || wantCancel && (!errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled)) || wantSource && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal("cause lost", err)
				}
				operation, _, _, _ := backupProxyActionTransportPolicy(family)
				bmwOperation(t, err, operation)
				out.applied.Body[0] = '!'
				out.applied.Header.Set("X-Proof", "result changed")
				if !bytes.Equal(accepted.Body, reply) || accepted.Header.Get("X-Proof") != "current action" {
					t.Fatal("result aliases error", accepted)
				}
				accepted.Body[1] = '?'
				accepted.Header.Set("X-Proof", "error changed")
				if out.applied.Body[1] != reply[1] || out.applied.Header.Get("X-Proof") != "result changed" {
					t.Fatal("error aliases result", out.applied)
				}
			})
		}
	}
}

func TestBackupProxyActionTransportNativeAndExpandedRejectionsNeverBecomeApplied(t *testing.T) {
	for _, family := range []string{"restore", "reset"} {
		for _, code := range []int{400, 403, 404, 500} {
			for _, expanded := range []bool{false, true} {
				t.Run(family+"/"+strconv.Itoa(code)+"/expanded="+strconv.FormatBool(expanded), func(t *testing.T) {
					cloud := testcloud.New(t)
					client := bmwClient(cloud)
					var calls, retries atomic.Int32
					if expanded {
						cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, original error, _ uint) error {
							if !gophercloud.ResponseCodeIs(original, 503) {
								return original
							}
							if retries.Add(1) > 1 {
								return errors.Join(original, errors.New("bounded expanded policy fixture"))
							}
							o.OkCodes = append(o.OkCodes, code)
							return nil
						}
					}
					rejected := []byte{0xff, 0, 'n', 'a', 't', 'i', 'v', 'e'}
					cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
						n := calls.Add(1)
						backupProxyActionTransportWire(t, r, family, "test-token")
						if expanded && n == 1 {
							testcloud.JSON(w, 503, `{"error":"retry once"}`)
							return
						}
						w.Header().Set("X-Proof", "rejected action")
						w.WriteHeader(code)
						_, _ = w.Write(rejected)
					})
					out, err := backupProxyActionTransportCall(bmwContext(t), client, family)
					backupProxyActionTransportNoAdmission(t, family, out, err)
					var native gophercloud.ErrUnexpectedResponseCode
					wantCalls := int32(1)
					if expanded {
						wantCalls = 2
					}
					if !errors.As(err, &native) || native.Actual != code || !bytes.Equal(native.Body, rejected) || native.ResponseHeader.Get("X-Proof") != "rejected action" || calls.Load() != wantCalls || retries.Load() != wantCalls-1 {
						t.Fatal(err, native, calls.Load(), retries.Load())
					}
				})
			}
		}
	}
}

func TestBackupProxyActionTransportLiveReauthAndOwnedBodyRetryReachOneAcceptedAction(t *testing.T) {
	for _, family := range []string{"restore", "reset"} {
		t.Run(family, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var calls, reauths, retries atomic.Int32
			cloud.Provider.ReauthFunc = func(context.Context) error {
				reauths.Add(1)
				cloud.Provider.SetToken("reauth-token")
				client.MoreHeaders["x-source"] = "later ordinary source header"
				return nil
			}
			cloud.Provider.RetryFunc = func(_ context.Context, method, target string, o *gophercloud.RequestOpts, original error, _ uint) error {
				if !gophercloud.ResponseCodeIs(original, 503) {
					return original
				}
				if retries.Add(1) > 1 {
					return errors.Join(original, errors.New("bounded action retry fixture"))
				}
				_, path, _, expected := backupProxyActionTransportPolicy(family)
				old, ok := o.JSONBody.(json.RawMessage)
				if !ok || string(old) != expected || method != http.MethodPost || !strings.HasSuffix(target, path) || o.RawBody != nil || o.JSONResponse != nil || !o.KeepResponseBody {
					t.Error(method, target, o, string(old))
				}
				o.JSONBody = json.RawMessage(bytes.Clone(old))
				if len(old) > 0 {
					old[0] = '!'
				}
				o.MoreHeaders["X-Native"] = "owned body clone"
				cloud.Provider.SetToken("retry-token")
				return nil
			}
			reply := `{"restore":{"status":"available","object_count":[]}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				token := "test-token"
				if n == 2 {
					token = "reauth-token"
				}
				if n == 3 {
					token = "retry-token"
				}
				backupProxyActionTransportWire(t, r, family, token)
				switch n {
				case 1:
					testcloud.JSON(w, 401, `{"error":"expired"}`)
				case 2:
					testcloud.JSON(w, 503, `{"error":"retry once"}`)
				case 3:
					if r.Header.Get("X-Native") != "owned body clone" {
						t.Error(r.Header)
					}
					w.Header().Set("X-Proof", "accepted once")
					testcloud.JSON(w, 203, reply)
				default:
					t.Error("extra action", r.URL)
					w.WriteHeader(500)
				}
			})
			out, err := backupProxyActionTransportCall(bmwContext(t), client, family)
			if err != nil || out == nil || out.applied == nil || family == "restore" && out.value == nil || family == "reset" && !out.completed || calls.Load() != 3 || reauths.Load() != 1 || retries.Load() != 1 || client.Microversion != "3.60" || client.MoreHeaders["x-source"] != "later ordinary source header" {
				t.Fatal(out, err, calls.Load(), reauths.Load(), retries.Load())
			}
			bmwPage(t, out.applied, 203, reply, "accepted once")
		})
	}
}

func TestBackupProxyActionTransportRetryMutationStopsBeforeSecondPhysicalRequest(t *testing.T) {
	for _, family := range []string{"restore", "reset"} {
		for _, kind := range []string{"body", "raw body", "decoder", "retention", "provider", "endpoint", "resource base", "type", "microversion", "body and callback error"} {
			t.Run(family+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := bmwClient(cloud)
				var calls, retries atomic.Int32
				callbackCause := errors.New("action retry callback")
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, original error, _ uint) error {
					if !gophercloud.ResponseCodeIs(original, 503) {
						return original
					}
					if retries.Add(1) > 1 {
						return errors.Join(original, errors.New("bounded mutation fixture"))
					}
					switch kind {
					case "body", "body and callback error":
						o.JSONBody = json.RawMessage(`{"changed":true}`)
					case "raw body":
						o.RawBody = strings.NewReader("changed")
					case "decoder":
						o.JSONResponse = new(any)
					case "retention":
						o.KeepResponseBody = false
					default:
						snapshotReadTransportMutate(client, kind)
					}
					if kind == "body and callback error" {
						return callbackCause
					}
					return nil
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					backupProxyActionTransportWire(t, r, family, "test-token")
					w.Header().Set("X-Proof", "native503")
					testcloud.JSON(w, 503, `{"error":"original rejected action"}`)
				})
				out, err := backupProxyActionTransportCall(bmwContext(t), client, family)
				backupProxyActionTransportNoAdmission(t, family, out, err)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || native.ResponseHeader.Get("X-Proof") != "native503" || calls.Load() != 1 || retries.Load() != 1 || kind == "body and callback error" && !errors.Is(err, callbackCause) {
					t.Fatal(err, native, calls.Load(), retries.Load())
				}
			})
		}
	}
}

func TestBackupProxyActionTransportSameTarget307CannotChangePhysicalBodyOrFraming(t *testing.T) {
	for _, family := range []string{"restore", "reset"} {
		for _, kind := range []string{"changed bytes", "extra byte", "truncated bytes", "nil body", "wrong length", "unknown length", "transfer encoding", "transfer header", "content length header", "duplicate length header"} {
			t.Run(family+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := bmwClient(cloud)
				var physical, redirects atomic.Int32
				_, _, _, expected := backupProxyActionTransportPolicy(family)
				cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
					redirects.Add(1)
					if len(via) != 1 || next.Method != http.MethodPost || next.URL.String() != via[0].URL.String() {
						t.Error("fixture changed route", next.Method, next.URL, via)
					}
					if next.Body != nil {
						_ = next.Body.Close()
					}
					data := expected
					switch kind {
					case "changed bytes":
						data = "!" + expected[1:]
					case "extra byte":
						data = expected + "x"
					case "truncated bytes":
						data = expected[:len(expected)-1]
					}
					next.Body = io.NopCloser(strings.NewReader(data))
					next.ContentLength = int64(len(expected))
					switch kind {
					case "nil body":
						next.Body = nil
					case "wrong length":
						next.ContentLength++
					case "unknown length":
						next.ContentLength = -1
					case "transfer encoding":
						next.TransferEncoding = []string{"chunked"}
					case "transfer header":
						next.Header["transfer-encoding"] = []string{"chunked"}
					case "content length header":
						next.Header["content-length"] = []string{"0"}
					case "duplicate length header":
						next.Header["Content-Length"] = []string{strconv.Itoa(len(expected)), strconv.Itoa(len(expected))}
					}
					return nil
				}
				cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(req *http.Request) (*http.Response, error) {
					n := physical.Add(1)
					if n == 1 {
						backupProxyActionTransportWire(t, req, family, "test-token")
						return &http.Response{StatusCode: 307, Header: http.Header{"Location": {req.URL.String()}}, Body: io.NopCloser(strings.NewReader("redirect")), Request: req}, nil
					}
					return snapshotReadTransportResponse(req, 203, io.NopCloser(strings.NewReader(`{}`)), "must not arrive"), nil
				})
				out, err := backupProxyActionTransportCall(bmwContext(t), client, family)
				backupProxyActionTransportNoAdmission(t, family, out, err)
				if !errors.Is(err, resource.ErrInvalidOption) || physical.Load() != 1 || redirects.Load() != 1 {
					t.Fatal(out, err, physical.Load(), redirects.Load())
				}
			})
		}
	}
}

func TestBackupProxyActionTransportPhysicalReadCloseFaultsStayTerminalAfterRetryRestoration(t *testing.T) {
	for _, family := range []string{"restore", "reset"} {
		for _, state := range []string{"read", "close", "read close callback cancel source"} {
			t.Run(family+"/"+state, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := bmwClient(cloud)
				ctx, cancel := context.WithCancelCause(bmwContext(t))
				defer cancel(nil)
				readCause, closeCause, callbackCause, cancelCause := errors.New("physical POST Read"), errors.New("physical POST Close"), errors.New("physical retry callback"), errors.New("physical retry cancellation")
				_, _, _, expected := backupProxyActionTransportPolicy(family)
				broken := &snapshotReadTransportBody{data: strings.NewReader(expected)}
				if strings.Contains(state, "read") {
					broken.readError = readCause
				}
				if strings.Contains(state, "close") {
					broken.closeError = closeCause
				}
				var physical, redirects, retries atomic.Int32
				var redirected *http.Request
				var firstRejected error
				cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
					redirects.Add(1)
					if next.Body != nil {
						_ = next.Body.Close()
					}
					next.Body = broken
					next.ContentLength = int64(len(expected))
					redirected = next
					return nil
				}
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, original error, _ uint) error {
					if retries.Add(1) > 1 {
						return errors.Join(original, errors.New("bounded physical fault fixture"))
					}
					firstRejected = original
					if redirected == nil || o == nil {
						t.Error("missing original redirect", redirected, o)
					} else {
						redirected.Body = io.NopCloser(strings.NewReader(expected))
						redirected.ContentLength = int64(len(expected))
						redirected.TransferEncoding = nil
						redirected.Header.Del("Transfer-Encoding")
						redirected.Header.Del("Content-Length")
					}
					if strings.Contains(state, "source") {
						client.Microversion = "3.61"
					}
					if strings.Contains(state, "cancel") {
						cancel(cancelCause)
					}
					if strings.Contains(state, "callback") {
						return callbackCause
					}
					return nil
				}
				cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(req *http.Request) (*http.Response, error) {
					n := physical.Add(1)
					if n == 1 {
						backupProxyActionTransportWire(t, req, family, "test-token")
						return &http.Response{StatusCode: 307, Header: http.Header{"Location": {req.URL.String()}}, Body: io.NopCloser(strings.NewReader("redirect")), Request: req}, nil
					}
					return snapshotReadTransportResponse(req, 203, io.NopCloser(strings.NewReader(`{}`)), "must not resend"), nil
				})
				out, err := backupProxyActionTransportCall(ctx, client, family)
				backupProxyActionTransportNoAdmission(t, family, out, err)
				if physical.Load() != 1 || redirects.Load() != 1 || retries.Load() != 1 || broken.closes.Load() != 1 || firstRejected == nil || !errors.Is(err, firstRejected) {
					t.Fatal(err, physical.Load(), redirects.Load(), retries.Load(), broken.closes.Load())
				}
				if strings.Contains(state, "read") && !errors.Is(err, readCause) || strings.Contains(state, "close") && !errors.Is(err, closeCause) || strings.Contains(state, "callback") && !errors.Is(err, callbackCause) || strings.Contains(state, "cancel") && (!errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled)) || strings.Contains(state, "source") && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal("physical/callback cause lost", err)
				}
			})
		}
	}
}

func TestBackupProxyActionTransportResetRetryVersionAliasesAreValidatedBeforeResend(t *testing.T) {
	for _, kind := range []string{"equivalent lowercase aliases", "delete current", "delete legacy", "wrong current", "wrong legacy", "conflicting alias", "omit mixedcase"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			var physical, retries atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, original error, _ uint) error {
				if !gophercloud.ResponseCodeIs(original, 503) {
					return original
				}
				if retries.Add(1) > 1 {
					return errors.Join(original, errors.New("bounded version alias fixture"))
				}
				switch kind {
				case "equivalent lowercase aliases":
					for key := range o.MoreHeaders {
						if strings.EqualFold(key, "Openstack-Api-Version") || strings.EqualFold(key, "X-Openstack-Volume-Api-Version") {
							delete(o.MoreHeaders, key)
						}
					}
					o.MoreHeaders["openstack-api-version"] = "volume 3.64"
					o.MoreHeaders["x-openstack-volume-api-version"] = "3.64"
				case "delete current":
					for key := range o.MoreHeaders {
						if strings.EqualFold(key, "Openstack-Api-Version") {
							delete(o.MoreHeaders, key)
						}
					}
				case "delete legacy":
					for key := range o.MoreHeaders {
						if strings.EqualFold(key, "X-Openstack-Volume-Api-Version") {
							delete(o.MoreHeaders, key)
						}
					}
				case "wrong current":
					o.MoreHeaders["Openstack-Api-Version"] = "volume 3.60"
				case "wrong legacy":
					o.MoreHeaders["X-Openstack-Volume-Api-Version"] = "3.60"
				case "conflicting alias":
					o.MoreHeaders["openstack-api-version"] = "volume 3.60"
				case "omit mixedcase":
					o.OmitHeaders = append(o.OmitHeaders, "x-OPENSTACK-volume-API-version")
				}
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := physical.Add(1)
				backupProxyActionTransportWire(t, r, "reset", "test-token")
				if n == 1 {
					testcloud.JSON(w, 503, `{"error":"version retry"}`)
					return
				}
				w.Header().Set("X-Proof", "opaque reset")
				testcloud.JSON(w, 203, "opaque")
			})
			out, err := backupProxyActionTransportCall(bmwContext(t), client, "reset")
			if kind == "equivalent lowercase aliases" {
				if err != nil || out == nil || !out.completed || out.applied == nil || physical.Load() != 2 || retries.Load() != 1 {
					t.Fatal(out, err, physical.Load(), retries.Load())
				}
			} else {
				backupProxyActionTransportNoAdmission(t, "reset", out, err)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || physical.Load() != 1 || retries.Load() != 1 {
					t.Fatal(err, physical.Load(), retries.Load())
				}
			}
			if client.Microversion != "3.60" || client.MoreHeaders["x-source"] != "original" {
				t.Fatal("forced action changed source", client)
			}
		})
	}
}

func TestBackupProxyActionTransportPhysicalResetHeaderRejectionSurvivesRetryRepair(t *testing.T) {
	for _, kind := range []string{"deleted current", "deleted legacy", "changed current", "duplicate legacy", "raw conflicting alias", "headers with callback cancel"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			ctx, cancel := context.WithCancelCause(bmwContext(t))
			defer cancel(nil)
			callbackCause, cancelCause := errors.New("header repair callback"), errors.New("header repair cancellation")
			var physical, redirects, retries atomic.Int32
			var redirected *http.Request
			var firstRejected error
			cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
				redirects.Add(1)
				redirected = next
				switch kind {
				case "deleted current":
					next.Header.Del("Openstack-Api-Version")
				case "deleted legacy":
					next.Header.Del("X-Openstack-Volume-Api-Version")
				case "changed current", "headers with callback cancel":
					next.Header.Set("Openstack-Api-Version", "volume 3.60")
				case "duplicate legacy":
					next.Header.Add("X-Openstack-Volume-Api-Version", "3.64")
				case "raw conflicting alias":
					next.Header["x-openstack-volume-api-version"] = []string{"3.60"}
				}
				return nil
			}
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, original error, _ uint) error {
				if retries.Add(1) > 1 {
					return errors.Join(original, errors.New("bounded physical header fixture"))
				}
				firstRejected = original
				if redirected != nil {
					for key := range redirected.Header {
						if strings.EqualFold(key, "Openstack-Api-Version") || strings.EqualFold(key, "X-Openstack-Volume-Api-Version") {
							delete(redirected.Header, key)
						}
					}
					redirected.Header.Set("Openstack-Api-Version", "volume 3.64")
					redirected.Header.Set("X-Openstack-Volume-Api-Version", "3.64")
				}
				if strings.Contains(kind, "callback") {
					cancel(cancelCause)
					return callbackCause
				}
				return nil
			}
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(req *http.Request) (*http.Response, error) {
				n := physical.Add(1)
				if n == 1 {
					backupProxyActionTransportWire(t, req, "reset", "test-token")
					return &http.Response{StatusCode: 307, Header: http.Header{"Location": {req.URL.String()}}, Body: io.NopCloser(strings.NewReader("redirect")), Request: req}, nil
				}
				return snapshotReadTransportResponse(req, 203, io.NopCloser(strings.NewReader("opaque")), "must not resend"), nil
			})
			out, err := backupProxyActionTransportCall(ctx, client, "reset")
			backupProxyActionTransportNoAdmission(t, "reset", out, err)
			if !errors.Is(err, resource.ErrInvalidOption) || physical.Load() != 1 || redirects.Load() != 1 || retries.Load() != 1 || firstRejected == nil || !errors.Is(err, firstRejected) {
				t.Fatal(err, physical.Load(), redirects.Load(), retries.Load())
			}
			if strings.Contains(kind, "callback") && (!errors.Is(err, callbackCause) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled)) {
				t.Fatal("joined header causes lost", err)
			}
		})
	}
}

func TestBackupProxyActionTransportRedirectScopeCannotChangeMethodPathOrOrigin(t *testing.T) {
	for _, family := range []string{"restore", "reset"} {
		for _, kind := range []string{"method", "path", "origin"} {
			t.Run(family+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				other := testcloud.New(t)
				client := bmwClient(cloud)
				var physical, stolen, redirects atomic.Int32
				cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
					redirects.Add(1)
					if kind == "method" {
						next.Method = http.MethodGet
					}
					return nil
				}
				other.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					stolen.Add(1)
					t.Error("token escaped action scope", r.Header)
					w.WriteHeader(500)
				})
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					n := physical.Add(1)
					backupProxyActionTransportWire(t, r, family, "test-token")
					if n != 1 {
						t.Error("changed target reached physical HTTP", r.URL)
					}
					target := cloud.Server.URL + r.URL.Path
					if kind == "path" {
						target = cloud.Server.URL + "/changed"
					}
					if kind == "origin" {
						target = other.Server.URL + "/stolen"
					}
					w.Header().Set("Location", target)
					w.WriteHeader(307)
				})
				out, err := backupProxyActionTransportCall(bmwContext(t), client, family)
				backupProxyActionTransportNoAdmission(t, family, out, err)
				if !errors.Is(err, resource.ErrInvalidOption) || physical.Load() != 1 || redirects.Load() != 1 || stolen.Load() != 0 {
					t.Fatal(err, physical.Load(), redirects.Load(), stolen.Load())
				}
			})
		}
	}
}
