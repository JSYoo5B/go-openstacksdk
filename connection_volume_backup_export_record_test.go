package gophercloudsdk_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	v3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func bepConnectionContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestConnectionBackupExportRecordPreflightAndSafeIDRejectBeforeCinder(t *testing.T) {
	for _, state := range []string{"nil context", "nil connection", "zero connection", "already canceled", "empty ID", "dot", "parent", "slash", "backslash", "query", "fragment", "escaped", "space", "unicode space", "control", "invalid UTF8"} {
		t.Run(state, func(t *testing.T) {
			cloud := testcloud.New(t)
			var locates, requests atomic.Int32
			cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				return "", errors.New("preflight must not select Cinder")
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				t.Error("preflight reached HTTP", r.Method, r.URL)
				w.WriteHeader(500)
			})
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancelCause(bepConnectionContext(t))
			defer cancel(nil)
			cause := errors.New("export preflight custom cancel")
			var selected context.Context = ctx
			id := "literal-backup"
			want := resource.ErrInvalidOption
			switch state {
			case "nil context":
				selected = nil
			case "nil connection":
				conn = nil
			case "zero connection":
				conn = &sdk.Connection{}
			case "already canceled":
				cancel(cause)
				want = context.Canceled
			case "empty ID":
				id = ""
			case "dot":
				id = "."
			case "parent":
				id = ".."
			case "slash":
				id = "a/b"
			case "backslash":
				id = "a\\b"
			case "query":
				id = "a?x=1"
			case "fragment":
				id = "a#x"
			case "escaped":
				id = "a%2Fb"
			case "space":
				id = "a b"
			case "unicode space":
				id = "a\u00a0b"
			case "control":
				id = "a\x00b"
			case "invalid UTF8":
				id = string([]byte{0xff})
			}
			result, err := conn.ExportVolumeBackupRecord(selected, blockstorage.ExportVolumeBackupRecordRequest{BackupID: id})
			var physical *resource.ResponseError
			if result != nil || !errors.Is(err, want) || errors.As(err, &physical) || locates.Load() != 0 || requests.Load() != 0 {
				t.Fatal(result, err, locates.Load(), requests.Load())
			}
			if state == "already canceled" && !errors.Is(err, cause) {
				t.Fatal("preflight lost context.Cause", err)
			}
			backupConnectionOperation(t, "ExportVolumeBackupRecord", err)
		})
	}
}

func TestConnectionBackupExportRecordOnlyCachedCinderPreservesSelectedPolicyWithoutLocationConsumption(t *testing.T) {
	for _, scope := range []string{"invalid configured location", "invalid recorded scope"} {
		t.Run(scope, func(t *testing.T) {
			cloud := testcloud.New(t)
			var locates, requests atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if opts.Type != "block-storage" || opts.Version != 3 {
					t.Error("export selected another service/version", opts)
				}
				return cloud.Server.URL + backupConnectionPath, nil
			}
			options := []sdk.ConnectionOption{sdk.WithMicroversion(sdk.BlockStorage, "3.80")}
			scopeErr := errors.New("recorded scope extraction is intentionally invalid")
			token := "test-token"
			if scope == "invalid configured location" {
				badCloud := string([]byte{0xff})
				bad := resource.CloudLocation{Cloud: &badCloud, Zone: json.RawMessage("not-json"), Project: resource.CloudProject{ID: json.RawMessage("not-json")}}
				if _, err := bad.ForResource(nil, nil); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal("fixture location is not invalid", err)
				}
				options = append(options, sdk.WithCloudLocation(bad))
			} else {
				recorded := &v3.CreateResult{}
				recorded.Header = http.Header{"X-Subject-Token": {"scope-token"}}
				if err := cloud.Provider.SetTokenAndAuthResult(recorded); err != nil {
					t.Fatal(err)
				}
				recorded.Err = scopeErr
				token = "scope-token"
			}
			conn, err := sdk.FromProvider(cloud.Provider, options...)
			if err != nil {
				t.Fatal(err)
			}
			if scope == "invalid recorded scope" {
				if _, err := conn.CurrentLocation(); !errors.Is(err, scopeErr) {
					t.Fatal("fixture scope is not invalid", err)
				}
			}
			firstBody := []byte(` {"backup-record":{"backup_url":"not base64!"},"n":9007199254740993,"n":2} `)
			cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/", func(w http.ResponseWriter, r *http.Request) {
				call := requests.Add(1)
				id := "plain-name"
				if call == 2 {
					id = "literal:한글"
				}
				backupConnectionWire(t, r, backupConnectionPath+"backups/"+id+"/export_record", token, "3.80")
				if r.URL.RawQuery != "" || r.Header.Get("OpenStack-API-Version") != "volume 3.80" || call == 2 && r.Header.Get("X-Captured") != "owned-later" {
					t.Error(r.URL, r.Header)
				}
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("X-Export-Proof", "opaque")
				if call == 1 {
					w.WriteHeader(203)
					_, _ = w.Write(firstBody)
				} else {
					w.WriteHeader(203)
					_, _ = w.Write([]byte(`null`))
				}
			})
			var cached *gophercloud.ServiceClient
			var prior *blockstorage.ExportVolumeBackupRecordResult
			for call, id := range []string{"plain-name", "literal:한글"} {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				result, err := conn.ExportVolumeBackupRecord(ctx, blockstorage.ExportVolumeBackupRecordRequest{BackupID: id})
				cancel()
				if err != nil || result == nil || result.BackupID != id || result.Exported == nil || result.Exported.Header.Get("X-Export-Proof") != "opaque" {
					t.Fatal(result, err)
				}
				if call == 0 && (result.Exported.StatusCode != 203 || !bytes.Equal(result.Exported.Body, firstBody) || string(result.Value) != string(firstBody[1:len(firstBody)-1])) || call == 1 && (result.Exported.StatusCode != 203 || string(result.Exported.Body) != "null" || result.Value == nil || string(result.Value) != "null") {
					t.Fatal("export decoded, normalized or discarded raw response", result)
				}
				if _, err := json.Marshal(result); err != nil {
					t.Fatal("opaque []byte response is not JSON serializable", err)
				}
				service, err := conn.BlockStorageV3(bepConnectionContext(t))
				if err != nil || service == nil || cached != nil && cached != service.RawClient() {
					t.Fatal("export replaced cached v3 service", service, err)
				}
				cached = service.RawClient()
				if cached.Microversion != "3.80" || cached.ProviderClient != cloud.Provider {
					t.Fatal("export forced/capped microversion or provider", cached)
				}
				if call == 0 {
					prior = result
					prior.Exported.Body[0] = '!'
					prior.Exported.Header.Set("X-Export-Proof", "caller-mutated")
					cached.MoreHeaders = map[string]string{"X-Captured": "owned-later"}
					cloud.Provider.SetToken("export-live-token")
					token = "export-live-token"
				} else if result.Exported == prior.Exported || result.Exported.Header.Get("X-Export-Proof") != "opaque" {
					t.Fatal("separate calls share physical proof", prior, result)
				}
			}
			if locates.Load() != 1 || requests.Load() != 2 {
				t.Fatal("export fetched/resolved/waited or selected another service", locates.Load(), requests.Load())
			}
		})
	}
}

func TestConnectionBackupExportRecordGetterFailuresJoinCustomAndContextCauses(t *testing.T) {
	for _, state := range []string{"getter error", "getter error and cancellation", "successful canceled getter"} {
		t.Run(state, func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx, cancel := context.WithCancelCause(bepConnectionContext(t))
			defer cancel(nil)
			getterErr, cause := errors.New("Cinder export getter failed"), errors.New("getter canceled export")
			var locates, requests atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if opts.Type != "block-storage" || opts.Version != 3 {
					t.Error(opts)
				}
				if state != "getter error" {
					cancel(cause)
				}
				if state == "successful canceled getter" {
					return cloud.Server.URL + backupConnectionPath, nil
				}
				return "", getterErr
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				t.Error("failed getter reached export HTTP", r.Method, r.URL)
				w.WriteHeader(500)
			})
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.80"))
			if err != nil {
				t.Fatal(err)
			}
			result, err := conn.ExportVolumeBackupRecord(ctx, blockstorage.ExportVolumeBackupRecordRequest{BackupID: "literal"})
			var physical *resource.ResponseError
			if result != nil || err == nil || errors.As(err, &physical) || locates.Load() != 1 || requests.Load() != 0 {
				t.Fatal(result, err, locates.Load(), requests.Load())
			}
			backupConnectionOperation(t, "ExportVolumeBackupRecord", err)
			if state != "successful canceled getter" && !errors.Is(err, getterErr) {
				t.Fatal("getter cause lost", err)
			}
			if state != "getter error" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
				t.Fatal("getter cancellation causes lost", err)
			}
		})
	}
}

func TestConnectionBackupExportRecordAcceptedOpaqueIOOwnsCurrentProofAndAllCauses(t *testing.T) {
	for _, state := range []string{"IO", "IO and cancellation", "IO and source drift", "IO source drift and cancellation"} {
		t.Run(state, func(t *testing.T) {
			cloud := testcloud.New(t)
			backupConnectionEndpoint(t, cloud)
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.80"))
			if err != nil {
				t.Fatal(err)
			}
			service, err := conn.BlockStorageV3(bepConnectionContext(t))
			if err != nil {
				t.Fatal(err)
			}
			source := service.RawClient()
			base, timeout := context.WithTimeout(context.Background(), 5*time.Second)
			defer timeout()
			ctx, cancel := context.WithCancelCause(base)
			defer cancel(nil)
			readErr, closeErr, cause := errors.New("export opaque Read"), errors.New("export opaque Close"), errors.New("export Close canceled parent")
			body := []byte(` {"logical":"must remain unpublished","n":9007199254740993} `)
			var requests, retries atomic.Int32
			cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries.Add(1)
				return errors.New("accepted export IO must not enter native retry")
			}
			cloud.Provider.HTTPClient.Transport = backupConnectionTransport(func(r *http.Request) (*http.Response, error) {
				requests.Add(1)
				backupConnectionWire(t, r, backupConnectionPath+"backups/literal/export_record", "test-token", "3.80")
				if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				fault := &backupConnectionFaultBody{reader: strings.NewReader(string(body)), readErr: readErr, closeErr: closeErr}
				fault.onClose = func() {
					if strings.Contains(state, "source drift") {
						source.Microversion = "3.81"
					}
					if strings.Contains(state, "cancellation") {
						cancel(cause)
					}
				}
				return &http.Response{StatusCode: 203, Header: http.Header{"Content-Type": {"application/octet-stream"}, "X-Actual": {"opaque-export"}}, Body: fault, Request: r}, nil
			})
			result, err := conn.ExportVolumeBackupRecord(ctx, blockstorage.ExportVolumeBackupRecordRequest{BackupID: "literal"})
			var physical *resource.ResponseError
			if result == nil || result.BackupID != "literal" || result.Value != nil || result.Exported == nil || !errors.As(err, &physical) || !errors.Is(err, readErr) || !errors.Is(err, closeErr) || physical.StatusCode != 203 || !bytes.Equal(physical.Body, body) || physical.Header.Get("X-Actual") != "opaque-export" || result.Exported.StatusCode != 203 || !bytes.Equal(result.Exported.Body, body) || result.Exported.Header.Get("X-Actual") != "opaque-export" || requests.Load() != 1 || retries.Load() != 0 {
				t.Fatal(result, err, physical, requests.Load(), retries.Load())
			}
			backupConnectionOperation(t, "ExportVolumeBackupRecord", err)
			if strings.Contains(state, "source drift") && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("source drift cause lost", err)
			}
			if strings.Contains(state, "cancellation") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
				t.Fatal("accepted IO cancellation causes lost", err)
			}
			physical.Body[0] = '!'
			physical.Header.Set("X-Actual", "error-mutated")
			if !bytes.Equal(result.Exported.Body, body) || result.Exported.Header.Get("X-Actual") != "opaque-export" {
				t.Fatal("result aliases physical error proof", result)
			}
			result.Exported.Body[1] = 0xee
			result.Exported.Header.Set("X-Actual", "result-mutated")
			if physical.Body[1] != body[1] || physical.Header.Get("X-Actual") != "error-mutated" {
				t.Fatal("error aliases result proof", physical)
			}
		})
	}
}

func TestConnectionBackupExportRecordParserFailureRetainsOpaqueProofWithParsedOperationAndNoNativeRetry(t *testing.T) {
	cloud := testcloud.New(t)
	var locates, calls, retries atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		if opts.Type != "block-storage" || opts.Version != 3 {
			t.Error(opts)
		}
		return cloud.Server.URL + backupConnectionPath, nil
	}
	cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
		retries.Add(1)
		return original
	}
	body := []byte(` {"record":true} null `)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		backupConnectionWire(t, r, backupConnectionPath+"backups/literal/export_record", "test-token", "3.80")
		if r.URL.RawQuery != "" {
			t.Error(r.URL)
		}
		w.Header().Set("X-Actual", "malformed-record")
		w.WriteHeader(203)
		_, _ = w.Write(body)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.80"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := conn.ExportVolumeBackupRecord(bepConnectionContext(t), blockstorage.ExportVolumeBackupRecordRequest{BackupID: "literal"})
	var physical *resource.ResponseError
	var native gophercloud.ErrUnexpectedResponseCode
	if err == nil || result == nil || result.BackupID != "literal" || result.Value != nil || result.Exported == nil || result.Exported.StatusCode != 203 || !bytes.Equal(result.Exported.Body, body) || result.Exported.Header.Get("X-Actual") != "malformed-record" || !errors.As(err, &physical) || physical.StatusCode != 203 || !bytes.Equal(physical.Body, body) || physical.Header.Get("X-Actual") != "malformed-record" || errors.As(err, &native) || calls.Load() != 1 || locates.Load() != 1 || retries.Load() != 0 {
		t.Fatal(result, err, calls.Load(), locates.Load(), retries.Load())
	}
	backupConnectionOperation(t, "ExportVolumeBackupRecord", err)
	result.Exported.Body[1] = '!'
	result.Exported.Header.Set("X-Actual", "result")
	if !bytes.Equal(physical.Body, body) || physical.Header.Get("X-Actual") != "malformed-record" {
		t.Fatal("parse error proof aliases returned opaque result", physical)
	}
	physical.Body[2] = '?'
	physical.Header.Set("X-Actual", "error")
	if result.Exported.Body[2] != body[2] || result.Exported.Header.Get("X-Actual") != "result" {
		t.Fatal("opaque result aliases parse error")
	}
}
