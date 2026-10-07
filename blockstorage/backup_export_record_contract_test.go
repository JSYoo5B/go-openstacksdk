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

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func bepOperation(t *testing.T, err error) {
	t.Helper()
	var operation *resource.OperationError
	if !errors.As(err, &operation) || operation.Operation != "ExportVolumeBackupRecord" || operation.Resource != "volume backup" || operation.Cause == nil {
		t.Fatal("parsed export operation/cause lost", err)
	}
}

func TestExportVolumeBackupRecordAcceptsArbitraryJSONAndPreservesLiteralValue(t *testing.T) {
	cases := []struct {
		name, document, value, version string
		code                           int
	}{
		{"null", " \r\n null \t", "null", "3.60", 200},
		{"false", " false ", "false", "3.60", 201},
		{"true", "true", "true", "3.60", 202},
		{"zero", " -0 ", "-0", "", 203},
		{"empty string", ` "" `, `""`, "3.60", 200},
		{"empty list", "[]", "[]", "3.60", 203},
		{"empty object", "{}", "{}", "3.60", 200},
		{"whole record", `{"backup-record":{"backup_url":"not base64!","backup_service":false,"unknown":{"nested":[null,1]}},"outside":true}`, `{"backup-record":{"backup_url":"not base64!","backup_service":false,"unknown":{"nested":[null,1]}},"outside":true}`, "3.70", 302},
		{"large integer", "900719925474099312345678901234567890", "900719925474099312345678901234567890", "3.60", 399},
		{"fraction and exponent", " [1.000000000000000000001,1e9999,-0.0,2E-9999] \n", "[1.000000000000000000001,1e9999,-0.0,2E-9999]", "3.60", 200},
		{"duplicates escapes and interior space", "\n { \"b\": 1, \"a\": \"\\u0061\\/\", \"b\": 2, \"text\": \"한글\" }\t", `{ "b": 1, "a": "\u0061\/", "b": 2, "text": "한글" }`, "3.60", 203},
		{"escaped lone surrogate is a literal", `"\ud800"`, `"\ud800"`, "3.60", 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			client.Microversion = tc.version
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				retries.Add(1)
				return original
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				bevWire(t, r, "literal-backup-name", tc.version, "test-token")
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("X-Proof", tc.name)
				w.WriteHeader(tc.code)
				_, _ = w.Write([]byte(tc.document))
			})
			result, err := blockstorage.ExportVolumeBackupRecord(bevContext(t), client, blockstorage.ExportVolumeBackupRecordRequest{BackupID: "literal-backup-name"})
			if err != nil || result == nil || result.BackupID != "literal-backup-name" || result.Value == nil || string(result.Value) != tc.value || result.Exported == nil || result.Exported.StatusCode != tc.code || string(result.Exported.Body) != tc.document || result.Exported.Header.Get("X-Proof") != tc.name || calls.Load() != 1 || retries.Load() != 0 || client.Microversion != tc.version {
				t.Fatal(result, err, calls.Load(), retries.Load())
			}
		})
	}
}

func TestExportVolumeBackupRecordRejectsInvalidDocumentWithoutParserRetryAndKeepsActualProof(t *testing.T) {
	cases := []struct {
		name string
		code int
		body []byte
	}{
		{"empty", 200, nil},
		{"empty204", 204, nil},
		{"whitespace", 203, []byte(" \r\n\t")},
		{"truncated", 202, []byte(`{"record":`)},
		{"two values", 200, []byte(`{"a":1} null`)},
		{"trailing garbage", 200, []byte(`[] #tail`)},
		{"NaN", 200, []byte(`NaN`)},
		{"Infinity", 200, []byte(`Infinity`)},
		{"nested nonfinite", 200, []byte(`[1,NaN]`)},
		{"leading zero", 200, []byte(`01`)},
		{"raw control in string", 200, []byte{'"', 'a', '\n', 'b', '"'}},
		{"invalid UTF8 string", 200, []byte{'"', 0xff, '"'}},
		{"invalid UTF8 trailing bytes", 200, []byte{'n', 'u', 'l', 'l', 0xff}},
		{"UTF8 BOM", 200, []byte{0xef, 0xbb, 0xbf, 'n', 'u', 'l', 'l'}},
		{"UTF16LE BOM", 200, []byte{0xff, 0xfe, 'n', 0, 'u', 0, 'l', 0, 'l', 0}},
		{"UTF16BE", 200, []byte{0, 'n', 0, 'u', 0, 'l', 0, 'l'}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				retries.Add(1)
				return original
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				bevWire(t, r, "literal", "3.60", "test-token")
				w.Header().Set("X-Proof", tc.name)
				w.WriteHeader(tc.code)
				_, _ = w.Write(tc.body)
			})
			result, err := blockstorage.ExportVolumeBackupRecord(bevContext(t), client, blockstorage.ExportVolumeBackupRecordRequest{BackupID: "literal"})
			var physical *resource.ResponseError
			var native gophercloud.ErrUnexpectedResponseCode
			if err == nil || result == nil || result.BackupID != "literal" || result.Value != nil || result.Exported == nil || result.Exported.StatusCode != tc.code || !bytes.Equal(result.Exported.Body, tc.body) || result.Exported.Header.Get("X-Proof") != tc.name || !errors.As(err, &physical) || physical.StatusCode != tc.code || !bytes.Equal(physical.Body, tc.body) || physical.Header.Get("X-Proof") != tc.name || errors.As(err, &native) || calls.Load() != 1 || retries.Load() != 0 {
				t.Fatal(result, err, calls.Load(), retries.Load())
			}
			bepOperation(t, err)
			if (tc.name == "empty" || tc.name == "empty204" || tc.name == "whitespace") && !errors.Is(err, io.EOF) {
				t.Fatal("empty document parser cause lost", err)
			}
			if tc.name == "truncated" && !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatal("incomplete document parser cause lost", err)
			}
			if tc.name == "NaN" || tc.name == "trailing garbage" {
				var syntax *json.SyntaxError
				if !errors.As(err, &syntax) {
					t.Fatal("native or local wrapper replaced parser syntax cause", err)
				}
			}
			result.Exported.Header.Set("X-Proof", "result changed")
			physical.Header.Set("X-Proof", "error changed")
			if result.Exported.Header.Get("X-Proof") != "result changed" || physical.Header.Get("X-Proof") != "error changed" {
				t.Fatal("error and result alias accepted headers")
			}
			if len(tc.body) > 0 {
				result.Exported.Body[0] ^= 1
				if !bytes.Equal(physical.Body, tc.body) {
					t.Fatal("parse error aliases result body")
				}
				physical.Body[len(tc.body)-1] ^= 1
				if len(tc.body) > 1 && result.Exported.Body[len(tc.body)-1] != tc.body[len(tc.body)-1] {
					t.Fatal("result aliases parse error body")
				}
			}
		})
	}
}

func TestExportVolumeBackupRecordHTTPRejectionPrecedesJSONParsingAndHasOnlyNativeProof(t *testing.T) {
	for _, code := range []int{400, 403, 404, 503} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			body := []byte{0xff, '{', 'b', 'r', 'o', 'k', 'e', 'n'}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				bevWire(t, r, "literal", "3.60", "test-token")
				w.Header().Set("X-Proof", "native only")
				w.WriteHeader(code)
				_, _ = w.Write(body)
			})
			result, err := blockstorage.ExportVolumeBackupRecord(bevContext(t), client, blockstorage.ExportVolumeBackupRecordRequest{BackupID: "literal"})
			var native gophercloud.ErrUnexpectedResponseCode
			var physical *resource.ResponseError
			if result == nil || result.BackupID != "literal" || result.Value != nil || result.Exported != nil || !errors.As(err, &native) || native.Actual != code || !bytes.Equal(native.Body, body) || native.ResponseHeader.Get("X-Proof") != "native only" || errors.As(err, &physical) || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			bepOperation(t, err)
		})
	}
}

func TestExportVolumeBackupRecordValuePhysicalProofAndSeparateCallsOwnDifferentStorage(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	document := []byte(` { "n":9007199254740993, "n":2, "s":"\u0061" } `)
	want := bytes.Clone(document[1 : len(document)-1])
	bodyBefore := bytes.Clone(document)
	headers := http.Header{"X-Proof": {"owned"}, "X-Multiple": {"one", "two"}}
	var calls atomic.Int32
	cloud.Provider.HTTPClient.Transport = snapshotReadTransport(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		bevWire(t, r, "literal", "3.60", "test-token")
		return &http.Response{StatusCode: 203, Header: headers, Body: io.NopCloser(bytes.NewReader(document)), Request: r}, nil
	})
	first, err := blockstorage.ExportVolumeBackupRecord(bevContext(t), client, blockstorage.ExportVolumeBackupRecordRequest{BackupID: "literal"})
	if err != nil || first == nil || first.Exported == nil || !bytes.Equal(first.Value, want) {
		t.Fatal(first, err)
	}
	first.Value[0] = '!'
	first.Exported.Body[0] = '?'
	first.Exported.Header["X-Multiple"][0] = "caller"
	if !bytes.Equal(document, bodyBefore) || first.Exported.Body[1] != '{' || first.Value[1] != ' ' || headers.Values("X-Multiple")[0] != "one" {
		t.Fatal("literal value, proof, or fixture aliases")
	}
	second, err := blockstorage.ExportVolumeBackupRecord(bevContext(t), client, blockstorage.ExportVolumeBackupRecordRequest{BackupID: "literal"})
	if err != nil || second == nil || second.Exported == nil || !bytes.Equal(second.Value, want) || !bytes.Equal(second.Exported.Body, bodyBefore) || second.Exported.Header.Values("X-Multiple")[0] != "one" || calls.Load() != 2 {
		t.Fatal(second, err, calls.Load())
	}
	document[2] = '#'
	headers["X-Multiple"][1] = "fixture"
	if !bytes.Equal(second.Value, want) || !bytes.Equal(second.Exported.Body, bodyBefore) || second.Exported.Header.Values("X-Multiple")[1] != "two" || first.Value[0] != '!' || first.Exported.Body[0] != '?' {
		t.Fatal("separate calls or physical storage alias")
	}
	second.Value[1] = '!'
	if second.Exported.Body[2] != bodyBefore[2] || first.Value[1] != want[1] {
		t.Fatal("second Value aliases proof or previous Value")
	}
}

func TestExportVolumeBackupRecordPreflightUsesCapturedSourceAndSafeLiteralID(t *testing.T) {
	for _, kind := range []string{"nil context", "canceled", "nil client", "wrong role before unsafe ID", "source query", "reserved auth", "empty ID", "path ID", "escaped ID", "space ID", "invalid UTF8 ID"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			ctx := bevContext(t)
			cause := errors.New("record preflight canceled")
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
				client.ResourceBase += "?x=1"
			case "reserved auth":
				client.MoreHeaders["x-auth-token"] = "caller"
			case "empty ID":
				id = ""
			case "path ID":
				id = "a/b"
			case "escaped ID":
				id = "a%2Fb"
			case "space ID":
				id = "a\u00a0b"
			case "invalid UTF8 ID":
				id = string([]byte{0xff})
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); w.WriteHeader(500) })
			result, err := blockstorage.ExportVolumeBackupRecord(ctx, client, blockstorage.ExportVolumeBackupRecordRequest{BackupID: id})
			var physical *resource.ResponseError
			if result != nil || !errors.Is(err, want) || errors.As(err, &physical) || calls.Load() != 0 {
				t.Fatal(result, err, calls.Load())
			}
			if kind == "canceled" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			bepOperation(t, err)
		})
	}
	t.Run("Unicode literal", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := snapshotReadContractClient(cloud)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			bevWire(t, r, "백업:한글", "3.60", "test-token")
			testcloud.JSON(w, 203, "null")
		})
		result, err := blockstorage.ExportVolumeBackupRecord(bevContext(t), client, blockstorage.ExportVolumeBackupRecordRequest{BackupID: "백업:한글"})
		if err != nil || result == nil || result.BackupID != "백업:한글" || result.Value == nil || string(result.Value) != "null" || result.Exported == nil || calls.Load() != 1 {
			t.Fatal(result, err, calls.Load())
		}
	})
}
