package blockstorage_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestExportVolumeBackupRecordAcceptedIOCancellationAndSourceFailuresKeepCurrentOwnedOpaqueProof(t *testing.T) {
	for _, fault := range []string{"read", "close", "cancel", "joined", "provider", "endpoint", "resource base", "type", "microversion"} {
		t.Run(fault, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			ctx, cancel := context.WithCancelCause(bevContext(t))
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("opaque export read"), errors.New("opaque export close"), errors.New("opaque export cancellation")
			original := []byte(` {"ready":true,"n":9007199254740993} `)
			broken := &snapshotReadTransportBody{data: bytes.NewReader(original)}
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
					if fault == "joined" {
						fact = "resource base"
					}
					snapshotReadTransportMutate(client, fact)
				}
				if wantCancel {
					cancel(cancelCause)
				}
			}
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				retries.Add(1)
				return original
			}
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				bevWire(t, r, "literal", "3.60", "test-token")
				return snapshotReadTransportResponse(r, 203, broken, "current opaque"), nil
			})
			result, err := blockstorage.ExportVolumeBackupRecord(ctx, client, blockstorage.ExportVolumeBackupRecordRequest{BackupID: "literal"})
			var physical *resource.ResponseError
			if err == nil || result == nil || result.BackupID != "literal" || result.Value != nil || result.Exported == nil || result.Exported.StatusCode != 203 || !bytes.Equal(result.Exported.Body, original) || result.Exported.Header.Get("X-Proof") != "current opaque" || !errors.As(err, &physical) || physical.StatusCode != 203 || !bytes.Equal(physical.Body, original) || physical.Header.Get("X-Proof") != "current opaque" || calls.Load() != 1 || retries.Load() != 0 || broken.closes.Load() != 1 {
				t.Fatal(result, err, calls.Load(), retries.Load(), broken.closes.Load())
			}
			if wantRead && !errors.Is(err, readCause) || wantClose && !errors.Is(err, closeCause) || wantCancel && (!errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled)) || wantSource && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("lost simultaneous cause", err)
			}
			bepOperation(t, err)
			result.Exported.Body[0] = '!'
			result.Exported.Header.Set("X-Proof", "result changed")
			if !bytes.Equal(physical.Body, original) || physical.Header.Get("X-Proof") != "current opaque" {
				t.Fatal("error aliases exported proof", physical)
			}
			physical.Body[1] = '?'
			physical.Header.Set("X-Proof", "error changed")
			if result.Exported.Body[1] != original[1] || result.Exported.Header.Get("X-Proof") != "result changed" {
				t.Fatal("exported proof aliases error")
			}
		})
	}
}

func TestExportVolumeBackupRecordNativeReauthAndBoundedRetryUseLiveTokensAndOwnedHeaders(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls, reauths, retries atomic.Int32
	cloud.Provider.ReauthFunc = func(context.Context) error {
		reauths.Add(1)
		cloud.Provider.SetToken("reauth-token")
		client.MoreHeaders["x-source"] = "changed ordinary header"
		return nil
	}
	cloud.Provider.RetryFunc = func(_ context.Context, method, target string, o *gophercloud.RequestOpts, original error, _ uint) error {
		if !gophercloud.ResponseCodeIs(original, 503) {
			return original
		}
		if retries.Add(1) > 1 {
			return errors.Join(original, errors.New("export retry fixture exhausted"))
		}
		if method != http.MethodGet || !strings.HasSuffix(target, "/backups/literal/export_record") || o.JSONBody != nil || o.RawBody != nil || o.JSONResponse != nil || !o.KeepResponseBody {
			t.Error("opaque request changed", method, target, o)
		}
		o.MoreHeaders["X-Native"] = "retry"
		cloud.Provider.SetToken("retry-token")
		return nil
	}
	body := []byte(` {"backup-record":{"backup_url":"arbitrary!"},"n":9007199254740993,"n":2} `)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		token := "test-token"
		if n == 2 {
			token = "reauth-token"
		}
		if n == 3 {
			token = "retry-token"
		}
		bevWire(t, r, "literal", "3.60", token)
		switch n {
		case 1:
			testcloud.JSON(w, 401, `{"error":"expired"}`)
		case 2:
			testcloud.JSON(w, 503, `{"error":"retry once"}`)
		case 3:
			if r.Header.Get("X-Native") != "retry" {
				t.Error(r.Header)
			}
			w.Header().Set("X-Proof", "accepted once")
			w.WriteHeader(203)
			_, _ = w.Write(body)
		default:
			t.Error("SDK export replayed", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.ExportVolumeBackupRecord(bevContext(t), client, blockstorage.ExportVolumeBackupRecordRequest{BackupID: "literal"})
	if err != nil || result == nil || result.BackupID != "literal" || string(result.Value) != string(body[1:len(body)-1]) || result.Exported == nil || result.Exported.StatusCode != 203 || !bytes.Equal(result.Exported.Body, body) || result.Exported.Header.Get("X-Proof") != "accepted once" || calls.Load() != 3 || reauths.Load() != 1 || retries.Load() != 1 || client.Microversion != "3.60" || client.MoreHeaders["x-source"] != "changed ordinary header" {
		t.Fatal(result, err, calls.Load(), reauths.Load(), retries.Load())
	}
}

func TestExportVolumeBackupRecordRejectsNativeRetryBodyDecoderRetentionOrSourceMutationBeforeResend(t *testing.T) {
	for _, kind := range []string{"body", "raw body", "decoder", "retention", "provider", "endpoint", "resource base", "type", "microversion", "body with callback error"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls, retries atomic.Int32
			callbackCause := errors.New("export retry callback failure")
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, original error, _ uint) error {
				if !gophercloud.ResponseCodeIs(original, 503) {
					return original
				}
				if retries.Add(1) > 1 {
					return errors.Join(original, errors.New("export mutation fixture exhausted"))
				}
				switch kind {
				case "body", "body with callback error":
					o.JSONBody = map[string]any{"changed": true}
				case "raw body":
					o.RawBody = strings.NewReader("changed")
				case "decoder":
					o.JSONResponse = new(any)
				case "retention":
					o.KeepResponseBody = false
				default:
					snapshotReadTransportMutate(client, kind)
				}
				if kind == "body with callback error" {
					return callbackCause
				}
				return nil
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				bevWire(t, r, "literal", "3.60", "test-token")
				w.Header().Set("X-Proof", "original native failure")
				testcloud.JSON(w, 503, `{"error":"retry attempt"}`)
			})
			result, err := blockstorage.ExportVolumeBackupRecord(bevContext(t), client, blockstorage.ExportVolumeBackupRecordRequest{BackupID: "literal"})
			var native gophercloud.ErrUnexpectedResponseCode
			var physical *resource.ResponseError
			if result == nil || result.BackupID != "literal" || result.Value != nil || result.Exported != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || native.ResponseHeader.Get("X-Proof") != "original native failure" || errors.As(err, &physical) || calls.Load() != 1 || retries.Load() != 1 {
				t.Fatal(result, err, calls.Load(), retries.Load())
			}
			if kind == "body with callback error" && !errors.Is(err, callbackCause) {
				t.Fatal("callback cause lost", err)
			}
			bepOperation(t, err)
		})
	}
}

func TestExportVolumeBackupRecordNativeExpandedStatusCannotCreateAdmittedProof(t *testing.T) {
	for _, code := range []int{400, 403, 404, 500} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, original error, _ uint) error {
				if !gophercloud.ResponseCodeIs(original, 503) {
					return original
				}
				if retries.Add(1) > 1 {
					return errors.Join(original, errors.New("export expanded-status fixture exhausted"))
				}
				o.OkCodes = append(o.OkCodes, code)
				return nil
			}
			body := []byte{0xff, 0, 'r', 'e', 'j'}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				bevWire(t, r, "literal", "3.60", "test-token")
				if n == 1 {
					testcloud.JSON(w, 503, `{"error":"retry once"}`)
					return
				}
				if n != 2 {
					t.Error("SDK replay after original policy rejection", r.URL)
				}
				w.Header().Set("X-Proof", "expanded rejected")
				w.WriteHeader(code)
				_, _ = w.Write(body)
			})
			result, err := blockstorage.ExportVolumeBackupRecord(bevContext(t), client, blockstorage.ExportVolumeBackupRecordRequest{BackupID: "literal"})
			var native gophercloud.ErrUnexpectedResponseCode
			var physical *resource.ResponseError
			if result == nil || result.BackupID != "literal" || result.Value != nil || result.Exported != nil || !errors.As(err, &native) || native.Actual != code || !bytes.Equal(native.Body, body) || native.ResponseHeader.Get("X-Proof") != "expanded rejected" || errors.As(err, &physical) || calls.Load() != 2 || retries.Load() != 1 {
				t.Fatal(result, err, calls.Load(), retries.Load())
			}
			bepOperation(t, err)
		})
	}
}

func TestExportVolumeBackupRecordRejectsPhysicalRedirectMutationAndKeepsStickyCause(t *testing.T) {
	for _, kind := range []string{"changed path", "another origin", "nonempty body", "empty body", "content length", "transfer encoding", "restore body with callback error and cancel"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			other := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			ctx, cancel := context.WithCancelCause(bevContext(t))
			defer cancel(nil)
			callbackCause, cancelCause := errors.New("record redirect retry callback"), errors.New("record redirect canceled parent")
			var calls, stolen, redirects, retries atomic.Int32
			var redirected *http.Request
			var firstRejection error
			other.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				stolen.Add(1)
				t.Error("redirect reached another origin", r.Header)
				w.WriteHeader(500)
			})
			cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
				redirects.Add(1)
				if len(via) != 1 {
					t.Error("unexpected redirect chain", via)
				}
				switch kind {
				case "changed path", "another origin":
				case "nonempty body", "restore body with callback error and cancel":
					next.Body = io.NopCloser(strings.NewReader("changed body"))
					next.ContentLength = int64(len("changed body"))
				case "empty body":
					next.Body = io.NopCloser(strings.NewReader(""))
					next.ContentLength = 0
				case "content length":
					next.Body = nil
					next.ContentLength = 1
				case "transfer encoding":
					next.Body = nil
					next.ContentLength = 0
					next.TransferEncoding = []string{"chunked"}
				}
				redirected = next
				return nil
			}
			if kind == "restore body with callback error and cancel" {
				cloud.Provider.RetryFunc = func(_ context.Context, method, target string, o *gophercloud.RequestOpts, original error, _ uint) error {
					if retries.Add(1) > 1 {
						return errors.Join(original, errors.New("bounded record redirect retry exhausted"))
					}
					if method != http.MethodGet || !strings.HasSuffix(target, "/backups/literal/export_record") || o == nil || o.JSONBody != nil || o.RawBody != nil || o.JSONResponse != nil || !o.KeepResponseBody || redirected == nil {
						t.Error(method, target, o, original)
					}
					firstRejection = original
					if redirected != nil {
						if redirected.Body != nil {
							_ = redirected.Body.Close()
						}
						redirected.Body = nil
						redirected.ContentLength = 0
						redirected.TransferEncoding = nil
					}
					cancel(cancelCause)
					return callbackCause
				}
			}
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
				if calls.Add(1) != 1 {
					t.Error("redirect or SDK resend reached physical transport", r.URL)
					return snapshotReadTransportResponse(r, 200, io.NopCloser(strings.NewReader(`null`)), "must not reach"), nil
				}
				bevWire(t, r, "literal", "3.60", "test-token")
				target := r.URL.String()
				if kind == "changed path" {
					target = cloud.Server.URL + "/changed/export"
				}
				if kind == "another origin" {
					target = other.Server.URL + "/changed/export"
				}
				return &http.Response{StatusCode: 302, Header: http.Header{"Location": {target}}, Body: io.NopCloser(strings.NewReader("opaque redirect")), Request: r}, nil
			})
			result, err := blockstorage.ExportVolumeBackupRecord(ctx, client, blockstorage.ExportVolumeBackupRecordRequest{BackupID: "literal"})
			var admitted *resource.ResponseError
			if result == nil || result.BackupID != "literal" || result.Value != nil || result.Exported != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &admitted) || calls.Load() != 1 || stolen.Load() != 0 || redirects.Load() != 1 {
				t.Fatal(result, err, calls.Load(), stolen.Load(), redirects.Load())
			}
			if kind == "restore body with callback error and cancel" && (retries.Load() != 1 || firstRejection == nil || !errors.Is(err, firstRejection) || !errors.Is(err, callbackCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
				t.Fatal("sticky redirect rejection or joined causes lost", err, retries.Load())
			}
			bepOperation(t, err)
		})
	}
}
