package blockstorage_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestExportVolumeBackupSameTargetRedirectCannotAddPhysicalBodyFraming(t *testing.T) {
	for _, kind := range []string{"body with length", "body without declared length", "empty nonnil body", "content length only", "transfer encoding only"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var physical, redirects atomic.Int32
			cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
				redirects.Add(1)
				if len(via) != 1 || next.Method != http.MethodGet || next.URL.String() != via[0].URL.String() {
					t.Error("fixture changed the fixed method or target", next.Method, next.URL, via)
				}
				switch kind {
				case "body with length":
					next.Body = io.NopCloser(strings.NewReader("redirect body"))
					next.ContentLength = int64(len("redirect body"))
				case "body without declared length":
					next.Body = io.NopCloser(strings.NewReader("redirect body"))
					next.ContentLength = 0
				case "empty nonnil body":
					next.Body = io.NopCloser(strings.NewReader(""))
					next.ContentLength = 0
				case "content length only":
					next.Body = nil
					next.ContentLength = 1
				case "transfer encoding only":
					next.Body = nil
					next.ContentLength = 0
					next.TransferEncoding = []string{"chunked"}
				}
				return nil
			}
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
				call := physical.Add(1)
				if call == 1 {
					bevWire(t, r, "literal", "3.60", "test-token")
					return &http.Response{StatusCode: 302, Header: http.Header{"Location": {r.URL.String()}}, Body: io.NopCloser(strings.NewReader("redirect acknowledgement")), Request: r}, nil
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"X-Proof": {"must not reach redirected transport"}}, Body: io.NopCloser(strings.NewReader("unexpected physical export")), Request: r}, nil
			})
			result, err := blockstorage.ExportVolumeBackup(bevContext(t), client, blockstorage.ExportVolumeBackupRequest{BackupID: "literal"})
			var admitted *resource.ResponseError
			if result == nil || result.BackupID != "literal" || result.Exported != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &admitted) || physical.Load() != 1 || redirects.Load() != 1 {
				t.Fatal("same-target GET gained a body or framing before physical rejection", result, err, physical.Load(), redirects.Load())
			}
			bevOperation(t, err)
		})
	}
}

func TestExportVolumeBackupRedirectBodyRejectionSurvivesNativeRetryRestorationAndJoinedCauses(t *testing.T) {
	for _, state := range []string{"restore and swallow", "restore with callback error", "restore and cancel", "restore callback error and cancel"} {
		t.Run(state, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			ctx, cancel := context.WithCancelCause(bevContext(t))
			defer cancel(nil)
			callbackCause := errors.New("native export retry callback failed")
			cancelCause := errors.New("native export retry canceled parent")
			var physical, redirects, retries atomic.Int32
			var redirected *http.Request
			var firstRejection error
			cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
				redirects.Add(1)
				if len(via) != 1 || next.Method != http.MethodGet || next.URL.String() != via[0].URL.String() {
					t.Error("fixture changed the fixed method or target", next.Method, next.URL, via)
				}
				next.Body = io.NopCloser(strings.NewReader("redirect body"))
				next.ContentLength = int64(len("redirect body"))
				next.TransferEncoding = []string{"chunked"}
				redirected = next
				return nil
			}
			cloud.Provider.RetryFunc = func(_ context.Context, method, target string, options *gophercloud.RequestOpts, original error, _ uint) error {
				if retries.Add(1) > 1 {
					return errors.Join(original, errors.New("bounded native export retry fixture exhausted"))
				}
				if method != http.MethodGet || !strings.HasSuffix(target, "/backups/literal/export_record") || options == nil || options.JSONBody != nil || options.RawBody != nil || options.JSONResponse != nil || !options.KeepResponseBody || redirected == nil {
					t.Error("native retry fixture did not observe the original bodyless export", method, target, options, original)
				}
				firstRejection = original
				// Restore the same object that the redirect callback changed.
				// The policy error must remain terminal after this repair.
				if redirected != nil {
					if redirected.Body != nil {
						_ = redirected.Body.Close()
					}
					redirected.Body = nil
					redirected.ContentLength = 0
					redirected.TransferEncoding = nil
				}
				if strings.Contains(state, "cancel") {
					cancel(cancelCause)
				}
				if strings.Contains(state, "callback error") {
					return callbackCause
				}
				return nil
			}
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
				call := physical.Add(1)
				if call == 1 {
					bevWire(t, r, "literal", "3.60", "test-token")
					return &http.Response{StatusCode: 302, Header: http.Header{"Location": {r.URL.String()}}, Body: io.NopCloser(strings.NewReader("redirect acknowledgement")), Request: r}, nil
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"X-Proof": {"must not resend"}}, Body: io.NopCloser(strings.NewReader("unexpected resend")), Request: r}, nil
			})
			result, err := blockstorage.ExportVolumeBackup(ctx, client, blockstorage.ExportVolumeBackupRequest{BackupID: "literal"})
			var admitted *resource.ResponseError
			if result == nil || result.BackupID != "literal" || result.Exported != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &admitted) || physical.Load() != 1 || redirects.Load() != 1 || retries.Load() != 1 || firstRejection == nil || !errors.Is(err, firstRejection) {
				t.Fatal("restoration or native callback erased the physical body rejection", result, err, physical.Load(), redirects.Load(), retries.Load())
			}
			if strings.Contains(state, "callback error") && !errors.Is(err, callbackCause) {
				t.Fatal("native callback cause lost", err)
			}
			if strings.Contains(state, "cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
				t.Fatal("parent cancellation causes lost", err)
			}
			if client.Microversion != "3.60" || client.MoreHeaders["x-source"] != "entry" {
				t.Fatal("export changed the source client", client)
			}
			bevOperation(t, err)
		})
	}
}
