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

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func bitNoAdmission(t *testing.T, result *blockstorage.ImportVolumeBackupResult, err error) {
	t.Helper()
	bidNoApplied(t, result, err, 0)
	var accepted *resource.ResponseError
	if errors.As(err, &accepted) {
		t.Fatal("unaccepted attempt became applied response", err, accepted)
	}
}

func TestImportVolumeBackupAcceptedFaultsKeepAppliedAndEveryCauseWithoutValue(t *testing.T) {
	for _, fault := range []string{"read", "close", "cancel", "provider", "endpoint", "resource base", "type", "microversion", "joined"} {
		t.Run(fault, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bidClient(cloud, "3.40")
			ctx, cancel := context.WithCancelCause(bmwContext(t))
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("import response Read"), errors.New("import response Close"), errors.New("import cancellation")
			reply := `{"backup":{"id":"wire","status":"available"}}`
			broken := &snapshotReadTransportBody{data: strings.NewReader(reply)}
			wantRead, wantClose := fault == "read" || fault == "joined", fault == "close" || fault == "joined"
			wantCancel := fault == "cancel" || fault == "joined"
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
				return errors.New("accepted import IO cannot retry")
			}
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(req *http.Request) (*http.Response, error) {
				calls.Add(1)
				bidPost(t, req, "3.40", "test-token")
				return snapshotReadTransportResponse(req, 203, broken, "accepted import"), nil
			})
			result, err := blockstorage.ImportVolumeBackup(ctx, client, bidInput())
			var proof *resource.ResponseError
			if result == nil || result.Applied == nil || result.Value != nil || result.Backup != nil || len(result.Discovery) != 0 || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != reply || proof.Header.Get("X-Proof") != "accepted import" || calls.Load() != 1 || retries.Load() != 0 || broken.closes.Load() != 1 {
				t.Fatal(result, err, proof, calls.Load(), retries.Load(), broken.closes.Load())
			}
			if wantRead && !errors.Is(err, readCause) || wantClose && !errors.Is(err, closeCause) || wantCancel && (!errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled)) || wantSource && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("accepted import cause lost", err)
			}
			bmwOperation(t, err, "ImportVolumeBackup")
			bmwPage(t, result.Applied, 203, reply, "accepted import")
			proof.Body[0] = '!'
			proof.Header.Set("X-Proof", "error changed")
			bmwPage(t, result.Applied, 203, reply, "accepted import")
			result.Applied.Body[1] = '?'
			if proof.Body[1] != reply[1] {
				t.Fatal("applied proof aliases response error")
			}
		})
	}
}

func TestImportVolumeBackupLiveReauthAndBoundedRetryKeepLiteralOwnedBody(t *testing.T) {
	cloud := testcloud.New(t)
	client := bidClient(cloud, "3.40")
	var calls, reauths, retries atomic.Int32
	cloud.Provider.ReauthFunc = func(context.Context) error {
		if reauths.Add(1) > 1 {
			return errors.New("bounded import reauth fixture exhausted")
		}
		cloud.Provider.SetToken("reauth-token")
		client.MoreHeaders["x-source"] = "later ordinary header"
		return nil
	}
	cloud.Provider.RetryFunc = func(_ context.Context, method, target string, opts *gophercloud.RequestOpts, original error, _ uint) error {
		if !gophercloud.ResponseCodeIs(original, 503) {
			return original
		}
		if retries.Add(1) > 1 {
			return errors.Join(original, errors.New("bounded import retry fixture exhausted"))
		}
		old, ok := opts.JSONBody.(json.RawMessage)
		if !ok || string(old) != bidBody || method != http.MethodPost || !strings.HasSuffix(target, bmwCollection+"/import_record") || opts.RawBody != nil || opts.JSONResponse != nil || !opts.KeepResponseBody {
			t.Error("retry acquired alternate request ownership", method, target, opts, string(old))
		}
		opts.JSONBody = json.RawMessage(bytes.Clone(old))
		if len(old) != 0 {
			old[0] = '!'
		}
		opts.MoreHeaders["X-Native"] = "owned clone"
		cloud.Provider.SetToken("retry-token")
		return nil
	}
	reply := `{"backup":{"id":"imported","status":"available"}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		n := calls.Add(1)
		token := map[int32]string{1: "test-token", 2: "reauth-token", 3: "retry-token"}[n]
		bidPost(t, req, "3.40", token)
		switch n {
		case 1:
			testcloud.JSON(w, 401, `{"error":"expired"}`)
		case 2:
			testcloud.JSON(w, 503, `{"error":"retry once"}`)
		case 3:
			if req.Header.Get("X-Native") != "owned clone" {
				t.Error(req.Header)
			}
			w.Header().Set("X-Proof", "accepted once")
			testcloud.JSON(w, 203, reply)
		default:
			t.Error("import replayed", req.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.ImportVolumeBackup(bmwContext(t), client, bidInput())
	if err != nil || result == nil || result.Value == nil || result.Backup == nil || len(result.Discovery) != 0 || result.Microversion != "3.40" || calls.Load() != 3 || reauths.Load() != 1 || retries.Load() != 1 || client.Microversion != "3.40" || client.MoreHeaders["x-source"] != "later ordinary header" {
		t.Fatal(result, err, calls.Load(), reauths.Load(), retries.Load())
	}
	bmwPage(t, result.Applied, 203, reply, "accepted once")
}

func TestImportVolumeBackupRetryMutationCannotSendChangedTargetBodyOrVersion(t *testing.T) {
	kinds := []string{"body", "raw body", "decoder", "retention", "provider", "endpoint", "resource base", "type", "microversion", "delete current", "delete legacy", "wrong current", "wrong legacy", "conflicting alias", "omit mixedcase", "body-callback-cancel"}
	for _, kind := range kinds {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bidClient(cloud, "3.40")
			ctx, cancel := context.WithCancelCause(bmwContext(t))
			defer cancel(nil)
			callbackCause, cancelCause := errors.New("import retry callback"), errors.New("import retry cancellation")
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, original error, _ uint) error {
				if !gophercloud.ResponseCodeIs(original, 503) {
					return original
				}
				if retries.Add(1) > 1 {
					return errors.Join(original, errors.New("bounded import mutation fixture exhausted"))
				}
				switch kind {
				case "body", "body-callback-cancel":
					opts.JSONBody = json.RawMessage(`{"changed":true}`)
				case "raw body":
					opts.RawBody = strings.NewReader("changed")
				case "decoder":
					opts.JSONResponse = new(any)
				case "retention":
					opts.KeepResponseBody = false
				case "delete current", "delete legacy":
					name := "OpenStack-API-Version"
					if kind == "delete legacy" {
						name = "X-OpenStack-Volume-API-Version"
					}
					for key := range opts.MoreHeaders {
						if strings.EqualFold(key, name) {
							delete(opts.MoreHeaders, key)
						}
					}
				case "wrong current":
					opts.MoreHeaders["Openstack-Api-Version"] = "volume 3.64"
				case "wrong legacy":
					opts.MoreHeaders["X-Openstack-Volume-Api-Version"] = "3.64"
				case "conflicting alias":
					opts.MoreHeaders["openstack-api-version"] = "volume 3.64"
				case "omit mixedcase":
					opts.OmitHeaders = append(opts.OmitHeaders, "x-OPENSTACK-volume-API-version")
				default:
					snapshotReadTransportMutate(client, kind)
				}
				if kind == "body-callback-cancel" {
					cancel(cancelCause)
					return callbackCause
				}
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				bidPost(t, req, "3.40", "test-token")
				w.Header().Set("X-Proof", "native rejection")
				testcloud.JSON(w, 503, `{"error":"retry once"}`)
			})
			result, err := blockstorage.ImportVolumeBackup(ctx, client, bidInput())
			bitNoAdmission(t, result, err)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || native.ResponseHeader.Get("X-Proof") != "native rejection" || calls.Load() != 1 || retries.Load() != 1 {
				t.Fatal(result, err, native, calls.Load(), retries.Load())
			}
			if kind == "body-callback-cancel" && (!errors.Is(err, callbackCause) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled)) {
				t.Fatal("retry joined causes lost", err)
			}
		})
	}
	t.Run("absent-discovered-version-cannot-be-injected", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := bidClient(cloud, "")
		var calls, retries atomic.Int32
		metadata := `{"id":"v3.0"}`
		cloud.Provider.RetryFunc = func(_ context.Context, method, _ string, opts *gophercloud.RequestOpts, original error, _ uint) error {
			if !gophercloud.ResponseCodeIs(original, 503) {
				return original
			}
			if retries.Add(1) > 1 {
				return errors.Join(original, errors.New("bounded absent-version fixture exhausted"))
			}
			if method != http.MethodPost {
				t.Error("retried discovery instead of import", method)
			}
			opts.MoreHeaders["OpenStack-API-Version"] = "volume 3.64"
			return nil
		}
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			switch calls.Add(1) {
			case 1:
				bidGet(t, req, bidVersionPath, "test-token")
				w.Header().Set("X-Proof", "metadata without max")
				testcloud.JSON(w, 200, metadata)
			case 2:
				bidPost(t, req, "", "test-token")
				testcloud.JSON(w, 503, `{"error":"retry import once"}`)
			default:
				t.Error("version injected into second physical import", req.URL)
				w.WriteHeader(500)
			}
		})
		result, err := blockstorage.ImportVolumeBackup(bmwContext(t), client, bidInput())
		bidNoApplied(t, result, err, 1)
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || calls.Load() != 2 || retries.Load() != 1 {
			t.Fatal(result, err, calls.Load(), retries.Load())
		}
		bmwPage(t, result.Discovery[0], 200, metadata, "metadata without max")
	})
}

func TestImportVolumeBackupPhysicalRedirectCannotChangeBodyFramingOrScope(t *testing.T) {
	kinds := []string{"method", "path", "origin", "changed bytes", "extra byte", "truncated bytes", "nil body", "wrong length", "unknown length", "transfer encoding", "transfer header", "content length header", "duplicate length header", "delete current", "delete legacy", "changed version", "duplicate legacy", "conflicting alias", "body-fault-repair", "version-fault-repair"}
	for _, kind := range kinds {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			other := testcloud.New(t)
			client := bidClient(cloud, "3.40")
			var calls, redirects, retries atomic.Int32
			var redirected *http.Request
			var firstRejected error
			readCause, closeCause := errors.New("redirected import Read"), errors.New("redirected import Close")
			broken := &snapshotReadTransportBody{data: strings.NewReader(bidBody), readError: readCause, closeError: closeCause}
			cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
				redirects.Add(1)
				redirected = next
				if len(via) != 1 {
					t.Error("unexpected redirect chain", len(via))
				}
				switch kind {
				case "method":
					next.Method = http.MethodGet
				case "delete current", "version-fault-repair":
					next.Header.Del("OpenStack-API-Version")
				case "delete legacy":
					next.Header.Del("X-OpenStack-Volume-API-Version")
				case "changed version":
					next.Header.Set("OpenStack-API-Version", "volume 3.64")
				case "duplicate legacy":
					next.Header.Add("X-OpenStack-Volume-API-Version", "3.40")
				case "conflicting alias":
					next.Header["x-openstack-volume-api-version"] = []string{"3.64"}
				}
				if kind == "method" || kind == "path" || kind == "origin" || strings.Contains(kind, "current") || strings.Contains(kind, "legacy") || kind == "changed version" || kind == "conflicting alias" || kind == "version-fault-repair" {
					return nil
				}
				if next.Body != nil {
					_ = next.Body.Close()
				}
				data := bidBody
				switch kind {
				case "changed bytes":
					data = "!" + bidBody[1:]
				case "extra byte":
					data += "x"
				case "truncated bytes":
					data = data[:len(data)-1]
				}
				next.Body = io.NopCloser(strings.NewReader(data))
				next.ContentLength = int64(len(bidBody))
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
					next.Header["Content-Length"] = []string{strconv.Itoa(len(bidBody)), strconv.Itoa(len(bidBody))}
				case "body-fault-repair":
					next.Body = broken
				}
				return nil
			}
			if strings.HasSuffix(kind, "fault-repair") {
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
					if retries.Add(1) > 1 {
						return errors.Join(original, errors.New("bounded physical repair fixture exhausted"))
					}
					firstRejected = original
					if redirected == nil {
						t.Error("no physical rejection to repair")
					} else {
						redirected.Body = io.NopCloser(strings.NewReader(bidBody))
						redirected.ContentLength = int64(len(bidBody))
						redirected.Header.Set("OpenStack-API-Version", "volume 3.40")
						redirected.Header.Set("X-OpenStack-Volume-API-Version", "3.40")
					}
					return nil
				}
			}
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(req *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if n != 1 {
					t.Error("changed physical request reached transport", req.Method, req.URL, req.Header)
					return snapshotReadTransportResponse(req, 203, io.NopCloser(strings.NewReader(`{}`)), "must not arrive"), nil
				}
				bidPost(t, req, "3.40", "test-token")
				target := req.URL.String()
				if kind == "path" {
					target = cloud.Server.URL + "/changed"
				}
				if kind == "origin" {
					target = other.Server.URL + "/stolen"
				}
				return &http.Response{StatusCode: 307, Header: http.Header{"Location": {target}}, Body: io.NopCloser(strings.NewReader("redirect")), Request: req}, nil
			})
			result, err := blockstorage.ImportVolumeBackup(bmwContext(t), client, bidInput())
			bitNoAdmission(t, result, err)
			if kind != "body-fault-repair" && !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 || redirects.Load() != 1 {
				t.Fatal(result, err, calls.Load(), redirects.Load())
			}
			if strings.HasSuffix(kind, "fault-repair") && (retries.Load() != 1 || firstRejected == nil || !errors.Is(err, firstRejected)) {
				t.Fatal("physical fault did not remain sticky after repair", err, retries.Load(), firstRejected)
			}
			if kind == "body-fault-repair" && (!errors.Is(err, readCause) || !errors.Is(err, closeCause) || broken.closes.Load() != 1) {
				t.Fatal("physical IO cause or Close count lost", err, broken.closes.Load())
			}
		})
	}
}
