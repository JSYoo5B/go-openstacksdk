package blockstorage_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	v3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestBackupProxyImportUsesFullDisconnectedModelWhileNativeImportRetainsBase64AndTypedResult(t *testing.T) {
	cloud := testcloud.New(t)
	client := bmwClient(cloud)
	api := v3.New(client).Backups
	var calls atomic.Int32
	reply := `{"backup":{"id":"imported","name":"actual","unknown":9007199254740993,"force":"false"}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		call := calls.Add(1)
		body := smwFields(t, bmwRequest(t, r, http.MethodPost, bmwCollection+"/import_record", "3.60", "test-token"))
		record := smwFields(t, body["backup-record"])
		wantURL := `"AAEC"`
		if call == 2 || call == 4 {
			wantURL = `"not base64!"`
		}
		if len(body) != 1 || len(record) != 2 || string(record["backup_service"]) != `"driver"` || string(record["backup_url"]) != wantURL || r.URL.RawQuery != "" {
			t.Error(body, record, call, r.URL)
		}
		code := 201
		if call >= 3 {
			code = 203
		}
		if call > 4 {
			t.Error("import used resolver or replay", r.URL)
			code = 500
		}
		testcloud.JSON(w, code, reply)
	})
	native, err := api.Import(bmwContext(t), backups.ImportOpts{BackupService: "driver", BackupURL: []byte{0, 1, 2}})
	if err != nil || native == nil || native.ID != "imported" || native.Name != "actual" || calls.Load() != 1 {
		t.Fatal(native, err, calls.Load())
	}
	converted, err := api.ImportBackup(bmwContext(t), "driver", "not base64!")
	if err != nil || converted == nil || string(converted.BackupID) != `"imported"` || converted.Backup == nil || converted.Applied == nil || converted.Value == nil || converted.Microversion != "3.60" || len(converted.Discovery) != 0 || calls.Load() != 2 {
		t.Fatal(converted, err, calls.Load())
	}
	fields := bipNullableFields(t, converted.Value)
	if string(fields["force"]) != "true" || string(fields["name"]) != `"actual"` || string(converted.Backup.Body["force"]) != `"false"` || string(converted.Backup.Body["unknown"]) != "9007199254740993" {
		t.Fatal(converted, string(converted.Value))
	}
	if _, present := fields["unknown"]; present {
		t.Fatal("native or physical DTO replaced Source model", fields)
	}
	_, err = api.Import(bmwContext(t), backups.ImportOpts{BackupService: "driver", BackupURL: []byte{0, 1, 2}})
	var nativeCode gophercloud.ErrUnexpectedResponseCode
	var operation *resource.OperationError
	if err == nil || !errors.As(err, &nativeCode) || nativeCode.Actual != 203 || !errors.As(err, &operation) || operation.Operation != "Import" || operation.Resource != "backups" || calls.Load() != 3 {
		t.Fatal(err, nativeCode, operation, calls.Load())
	}
	converted, err = api.ImportBackup(bmwContext(t), "driver", "not base64!")
	if err != nil || converted == nil || converted.Applied == nil || converted.Applied.StatusCode != 203 || string(converted.Applied.Body) != reply || converted.Value == nil || calls.Load() != 4 || client.Microversion != "3.60" {
		t.Fatal(converted, err, calls.Load(), client)
	}
}

func TestBackupProxyImportPreflightKeepsCorrectOuterOperationAndCauses(t *testing.T) {
	for _, kind := range []string{"nil API", "nil context", "custom cancellation", "wrong role", "invalid service", "invalid record", "reserved auth", "accepted descriptor failure"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			api := backups.New(client)
			ctx := bmwContext(t)
			service, record := "driver", "record"
			cause := errors.New("service import custom cause")
			want := resource.ErrInvalidOption
			switch kind {
			case "nil API":
				api = nil
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
			case "invalid service":
				service = string([]byte{0xff})
			case "invalid record":
				record = string([]byte{0xff})
			case "reserved auth":
				client.MoreHeaders["x-auth-token"] = "caller"
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if kind != "accepted descriptor failure" {
					t.Error("service preflight reached HTTP", r.URL)
					w.WriteHeader(500)
					return
				}
				bipPayload(t, r, service, record, "3.60", "test-token")
				w.Header().Set("X-Proof", "descriptor")
				testcloud.JSON(w, 202, `{"backup":{"object_count":"²","unknown":false}}`)
			})
			result, err := api.ImportBackup(ctx, service, record)
			var outer *resource.OperationError
			var proof *resource.ResponseError
			if err == nil || !errors.As(err, &outer) || outer.Operation != "ImportBackup" || outer.Resource != "backups" || outer.Cause == nil {
				t.Fatal(result, err, outer)
			}
			if kind == "accepted descriptor failure" {
				if result == nil || result.Value != nil || result.Backup == nil || result.Applied == nil || !errors.As(err, &proof) || proof.StatusCode != 202 || proof.Header.Get("X-Proof") != "descriptor" || calls.Load() != 1 || string(result.Backup.Body["unknown"]) != "false" {
					t.Fatal(result, err, proof, calls.Load())
				}
			} else if result != nil || !errors.Is(err, want) || errors.As(err, &proof) || calls.Load() != 0 {
				t.Fatal(result, err, proof, calls.Load())
			}
			if kind == "custom cancellation" && !errors.Is(err, cause) {
				t.Fatal("service operation lost cancellation cause", err)
			}
		})
	}
}
