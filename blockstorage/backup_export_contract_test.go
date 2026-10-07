package blockstorage_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const bevCollection = snapshotReadContractBase + "backups"

func bevContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func bevWire(t *testing.T, r *http.Request, id, version, token string) {
	t.Helper()
	path := bevCollection + "/" + id + "/export_record"
	escaped := bevCollection + "/" + url.PathEscape(id) + "/export_record"
	var body []byte
	var err error
	if r.Body != nil {
		body, err = io.ReadAll(r.Body)
	}
	wantVersion := ""
	if version != "" {
		wantVersion = "volume " + version
	}
	if err != nil || len(body) != 0 || r.Method != http.MethodGet || r.URL.Path != path || r.URL.EscapedPath() != escaped || r.URL.RawQuery != "" || r.Header.Get("X-Source") != "entry" || r.Header.Get("X-Auth-Token") != token || r.Header.Get("OpenStack-API-Version") != wantVersion {
		t.Error("opaque export fixed request changed", r.Method, r.URL, r.Header, string(body), err)
	}
}

func bevOperation(t *testing.T, err error) {
	t.Helper()
	var operation *resource.OperationError
	if !errors.As(err, &operation) || operation.Operation != "ExportVolumeBackup" || operation.Resource != "volume backup" || operation.Cause == nil {
		t.Fatal("export operation/cause lost", err)
	}
}

func TestExportVolumeBackupOpaqueSub400ResponseUsesOneBodylessFixedGETAndSelectedMicroversion(t *testing.T) {
	cases := []struct {
		name    string
		code    int
		body    []byte
		version string
	}{
		{"binary", 200, []byte{0xff, 0, 'x', '\n'}, "3.70"},
		{"record without base64 decoding", 203, []byte(`{"backup-record":{"backup_url":"not base64!","n":9007199254740993},"id":"other"}`), "3.60"},
		{"nonobject JSON", 201, []byte(`[false,null,1.0000000000000000001]`), ""},
		{"malformed JSON", 202, []byte(`{broken`), "3.60"},
		{"empty204", 204, nil, "3.60"},
		{"redirect status without location", 302, []byte("opaque redirect"), "3.60"},
		{"last sub400", 399, []byte("opaque399"), "3.60"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			client.Microversion = tc.version
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				bevWire(t, r, "literal-backup-name", tc.version, "test-token")
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("X-Proof", tc.name)
				w.WriteHeader(tc.code)
				_, _ = w.Write(tc.body)
			})
			result, err := blockstorage.ExportVolumeBackup(bevContext(t), client, blockstorage.ExportVolumeBackupRequest{BackupID: "literal-backup-name"})
			if err != nil || result == nil || result.BackupID != "literal-backup-name" || result.Exported == nil || result.Exported.StatusCode != tc.code || !bytes.Equal(result.Exported.Body, tc.body) || result.Exported.Header.Get("X-Proof") != tc.name || calls.Load() != 1 || client.Microversion != tc.version {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestExportVolumeBackupOwnsOpaqueBodyAndHeadersAcrossIndependentCalls(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	body := []byte{0xff, 0, 'a', 'b'}
	original := bytes.Clone(body)
	headers := http.Header{"Content-Type": {"application/octet-stream"}, "X-Proof": {"owned"}, "X-Multiple": {"one", "two"}}
	var calls atomic.Int32
	cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		bevWire(t, r, "literal", "3.60", "test-token")
		return &http.Response{StatusCode: 200, Header: headers, Body: io.NopCloser(bytes.NewReader(body)), Request: r}, nil
	})
	first, err := blockstorage.ExportVolumeBackup(bevContext(t), client, blockstorage.ExportVolumeBackupRequest{BackupID: "literal"})
	if err != nil || first == nil || first.Exported == nil {
		t.Fatal(first, err)
	}
	first.Exported.Body[0] = '!'
	first.Exported.Header["X-Multiple"][0] = "changed"
	first.Exported.Header.Set("X-Proof", "changed")
	if !bytes.Equal(body, original) || headers.Get("X-Proof") != "owned" || headers.Values("X-Multiple")[0] != "one" {
		t.Fatal("result aliases physical fixture")
	}
	second, err := blockstorage.ExportVolumeBackup(bevContext(t), client, blockstorage.ExportVolumeBackupRequest{BackupID: "literal"})
	if err != nil || second == nil || second.Exported == nil || !bytes.Equal(second.Exported.Body, original) || second.Exported.Header.Get("X-Proof") != "owned" || second.Exported.Header.Values("X-Multiple")[0] != "one" || calls.Load() != 2 {
		t.Fatal(second, err, calls.Load())
	}
	body[1] = '?'
	headers["X-Multiple"][1] = "later physical change"
	if !bytes.Equal(second.Exported.Body, original) || second.Exported.Header.Values("X-Multiple")[1] != "two" || first.Exported.Body[0] != '!' {
		t.Fatal("calls share mutable response storage")
	}
}

func TestExportVolumeBackupPreflightCapturesSourceBeforeSafeIDAndNeverMakesLookupRequests(t *testing.T) {
	unsafe := []string{"", ".", "..", "part/id", "part\\id", "%2F", "has space", "has\ncontrol", "\u00a0", string([]byte{0xff})}
	for _, id := range unsafe {
		t.Run("invalid ID "+id, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(500) })
			result, err := blockstorage.ExportVolumeBackup(bevContext(t), client, blockstorage.ExportVolumeBackupRequest{BackupID: id})
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(result, err, calls.Load())
			}
			bevOperation(t, err)
		})
	}
	for _, kind := range []string{"nil context", "canceled", "nil client", "wrong role before unsafe ID", "source query", "reserved source auth"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			ctx := bevContext(t)
			cause := errors.New("export custom cancellation")
			id := "literal"
			want := resource.ErrInvalidOption
			switch kind {
			case "nil context":
				ctx = nil
			case "canceled":
				child, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx = child
				want = context.Canceled
			case "nil client":
				client = nil
			case "wrong role before unsafe ID":
				client.Type = "compute"
				id = "%2F"
				want = resource.ErrUnsupported
			case "source query":
				client.ResourceBase += "?extra=1"
			case "reserved source auth":
				client.MoreHeaders["x-auth-token"] = "caller token"
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(500) })
			result, err := blockstorage.ExportVolumeBackup(ctx, client, blockstorage.ExportVolumeBackupRequest{BackupID: id})
			if result != nil || !errors.Is(err, want) || calls.Load() != 0 {
				t.Fatal(result, err, calls.Load())
			}
			if kind == "canceled" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			bevOperation(t, err)
		})
	}
	t.Run("Unicode literal single segment", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := snapshotReadContractClient(cloud)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			bevWire(t, r, "백업", "3.60", "test-token")
			w.WriteHeader(204)
		})
		result, err := blockstorage.ExportVolumeBackup(bevContext(t), client, blockstorage.ExportVolumeBackupRequest{BackupID: "백업"})
		if err != nil || result == nil || result.BackupID != "백업" || result.Exported == nil || result.Exported.StatusCode != 204 || calls.Load() != 1 {
			t.Fatal(result, err, calls.Load())
		}
	})
}

func TestExportVolumeBackupNativeRejectionsAreTerminalAndDoNotInventAdmittedProof(t *testing.T) {
	for _, code := range []int{400, 403, 404, 503} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			body := `{"error":"export rejected"}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				bevWire(t, r, "literal", "3.60", "test-token")
				w.Header().Set("X-Proof", "native rejection")
				testcloud.JSON(w, code, body)
			})
			result, err := blockstorage.ExportVolumeBackup(bevContext(t), client, blockstorage.ExportVolumeBackupRequest{BackupID: "literal"})
			var native gophercloud.ErrUnexpectedResponseCode
			var physical *resource.ResponseError
			if result == nil || result.BackupID != "literal" || result.Exported != nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != body || native.ResponseHeader.Get("X-Proof") != "native rejection" || errors.As(err, &physical) || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			bevOperation(t, err)
		})
	}
}
