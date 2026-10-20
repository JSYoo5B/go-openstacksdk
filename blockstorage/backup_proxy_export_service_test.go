package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	blockstoragev3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
	backupsv3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type backupProxyExportServiceOutcome struct {
	id       string
	value    json.RawMessage
	exported *backupsv3.ExportRecordResponse
}

func backupProxyExportServiceCall(ctx context.Context, api *backupsv3.API, method, id string) (*backupProxyExportServiceOutcome, error) {
	if method == "ExportRecord" {
		result, err := api.ExportRecord(ctx, id)
		if result == nil {
			return nil, err
		}
		return &backupProxyExportServiceOutcome{id: result.BackupID, exported: result.Exported}, err
	}
	result, err := api.ExportBackup(ctx, id)
	if result == nil {
		return nil, err
	}
	return &backupProxyExportServiceOutcome{id: result.BackupID, value: result.Value, exported: result.Exported}, err
}

func backupProxyExportServiceOperation(t *testing.T, method string, err error) {
	t.Helper()
	var outer *resource.OperationError
	if !errors.As(err, &outer) || outer.Operation != method || outer.Resource != "backups" || outer.Cause == nil {
		t.Fatal("service-facade operation/cause lost", method, err, outer)
	}
}

func TestBackupProxyExportServiceFacadesPreflightKeepOuterMethodContext(t *testing.T) {
	for _, method := range []string{"ExportRecord", "ExportBackup"} {
		for _, state := range []string{"nil API", "nil client", "nil context", "canceled", "empty ID", "unsafe ID", "invalid UTF8 ID", "wrong service", "reserved auth header"} {
			t.Run(method+"/"+state, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := snapshotReadContractClient(cloud)
				api := backupsv3.New(client)
				ctx := bevContext(t)
				id := "literal"
				want := resource.ErrInvalidOption
				cause := errors.New("service export custom preflight cancellation")
				switch state {
				case "nil API":
					api = nil
				case "nil client":
					api = backupsv3.New(nil)
				case "nil context":
					ctx = nil
				case "canceled":
					child, cancel := context.WithCancelCause(ctx)
					defer cancel(nil)
					cancel(cause)
					ctx = child
					want = context.Canceled
				case "empty ID":
					id = ""
				case "unsafe ID":
					id = "a/b"
				case "invalid UTF8 ID":
					id = string([]byte{0xff})
				case "wrong service":
					client.Type = "compute"
					want = resource.ErrUnsupported
				case "reserved auth header":
					client.MoreHeaders["x-auth-token"] = "caller token"
				}
				var calls atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					t.Error("service preflight reached HTTP", r.Method, r.URL)
					w.WriteHeader(500)
				})
				result, err := backupProxyExportServiceCall(ctx, api, method, id)
				var physical *resource.ResponseError
				if result != nil || !errors.Is(err, want) || errors.As(err, &physical) || calls.Load() != 0 {
					t.Fatal(result, err, calls.Load())
				}
				if state == "canceled" && !errors.Is(err, cause) {
					t.Fatal("custom context cause lost", err)
				}
				backupProxyExportServiceOperation(t, method, err)
			})
		}
	}
}

func TestBackupProxyExportServiceFacadesSeparateOpaqueJSONAndLegacyTypedProjection(t *testing.T) {
	cases := []struct {
		name, method string
		status       int
		body, value  []byte
	}{
		{"opaque invalid UTF8", "ExportRecord", 203, []byte{0xff, 0, 'x'}, nil},
		{"opaque malformed JSON", "ExportRecord", 202, []byte(`{broken`), nil},
		{"parsed null is present", "ExportBackup", 203, []byte(" \nnull\t"), []byte("null")},
		{"parsed array", "ExportBackup", 201, []byte(` ["x",false,9007199254740993] `), []byte(`["x",false,9007199254740993]`)},
		{"parsed literals and duplicate keys", "ExportBackup", 203, []byte(` {"n":9007199254740993,"n":1.2300,"escaped":"\u0041"} `), []byte(`{"n":9007199254740993,"n":1.2300,"escaped":"\u0041"}`)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			client.Microversion = "3.80"
			service := blockstoragev3.New(client)
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				retries.Add(1)
				return original
			}
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				bevWire(t, r, "literal-name", "3.80", "test-token")
				return snapshotReadTransportResponse(r, tc.status, io.NopCloser(bytes.NewReader(tc.body)), "service proof"), nil
			})
			result, err := backupProxyExportServiceCall(bevContext(t), service.Backups, tc.method, "literal-name")
			if err != nil || result == nil || result.id != "literal-name" || result.exported == nil || result.exported.StatusCode != tc.status || !bytes.Equal(result.exported.Body, tc.body) || result.exported.Header.Get("X-Proof") != "service proof" || !bytes.Equal(result.value, tc.value) || calls.Load() != 1 || retries.Load() != 0 || service.Backups.RawClient() != client || client.Microversion != "3.80" || client.MoreHeaders["x-source"] != "entry" {
				t.Fatal(result, err, calls.Load(), retries.Load())
			}
			if tc.method == "ExportBackup" {
				if result.value == nil {
					t.Fatal("successful JSON null/value became absence", result)
				}
				saved := bytes.Clone(result.exported.Body)
				result.value[0] = '!'
				if !bytes.Equal(result.exported.Body, saved) {
					t.Fatal("logical JSON aliases physical evidence", result)
				}
			} else if result.value != nil {
				t.Fatal("opaque export invented parsed JSON", result)
			}
			result.exported.Body[0] = '?'
			result.exported.Header.Set("X-Proof", "caller changed")
			if tc.body[0] == '?' {
				t.Fatal("facade physical proof aliases fixture bytes")
			}
		})
	}
	for _, status := range []int{200, 203} {
		t.Run("unchanged native "+http.StatusText(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			client.Microversion = "3.80"
			api := blockstoragev3.New(client).Backups
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				bevWire(t, r, "literal", "3.80", "test-token")
				testcloud.JSON(w, status, `{"backup-record":{"backup_service":"backend","backup_url":"cmF3"}}`)
			})
			typed, err := api.Export(bevContext(t), "literal")
			if status == 200 {
				if err != nil || typed == nil || typed.BackupService != "backend" || !bytes.Equal(typed.BackupURL, []byte("raw")) {
					t.Fatal(typed, err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if typed == nil || typed.BackupService != "" || len(typed.BackupURL) != 0 || !errors.As(err, &native) || native.Actual != 203 || len(native.Expected) != 1 || native.Expected[0] != 200 {
					t.Fatal("legacy Export policy/projection changed", typed, err, native)
				}
				backupProxyExportServiceOperation(t, "Export", err)
			}
			if calls.Load() != 1 || api.RawClient() != client || client.Microversion != "3.80" {
				t.Fatal(calls.Load(), client)
			}
		})
	}
}

func TestBackupProxyExportServiceParsedFailuresKeepCurrentProofAndNeverRetryDecoder(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   []byte
		joined bool
	}{
		{"empty admitted204", 204, nil, false},
		{"malformed JSON", 200, []byte(`{"broken":`), false},
		{"multiple documents", 202, []byte(`null true`), false},
		{"invalid UTF8 JSON", 203, []byte{'"', 0xff, '"'}, false},
		{"accepted IO source and cancel", 200, []byte(`{"broken":`), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			client.Microversion = "3.80"
			api := backupsv3.New(client)
			ctx, cancel := context.WithCancelCause(bevContext(t))
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("parsed service export Read"), errors.New("parsed service export Close"), errors.New("parsed service export custom cancel")
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				retries.Add(1)
				return original
			}
			cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				bevWire(t, r, "literal", "3.80", "test-token")
				body := &snapshotReadTransportBody{data: bytes.NewReader(tc.body)}
				if tc.joined {
					body.readError, body.closeError = readCause, closeCause
					body.onClose = func() { client.Microversion = "3.81"; cancel(cancelCause) }
				}
				return snapshotReadTransportResponse(r, tc.status, body, "current parsed service phase"), nil
			})
			result, err := api.ExportBackup(ctx, "literal")
			var physical *resource.ResponseError
			if result == nil || result.BackupID != "literal" || result.Value != nil || result.Exported == nil || result.Exported.StatusCode != tc.status || !bytes.Equal(result.Exported.Body, tc.body) || result.Exported.Header.Get("X-Proof") != "current parsed service phase" || !errors.As(err, &physical) || physical.StatusCode != tc.status || !bytes.Equal(physical.Body, tc.body) || physical.Header.Get("X-Proof") != "current parsed service phase" || calls.Load() != 1 || retries.Load() != 0 {
				t.Fatal(result, err, physical, calls.Load(), retries.Load())
			}
			backupProxyExportServiceOperation(t, "ExportBackup", err)
			if tc.joined && (!errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
				t.Fatal("joined accepted causes lost", err)
			}
			physical.Header.Set("X-Proof", "error changed")
			if len(physical.Body) > 0 {
				physical.Body[0] = '!'
			}
			if !bytes.Equal(result.Exported.Body, tc.body) || result.Exported.Header.Get("X-Proof") != "current parsed service phase" {
				t.Fatal("facade result aliases error evidence", result)
			}
			result.Exported.Header.Set("X-Proof", "result changed")
			if physical.Header.Get("X-Proof") != "error changed" {
				t.Fatal("facade error aliases result evidence", physical)
			}
		})
	}
}
