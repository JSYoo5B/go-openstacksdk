package backups_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage/v3/backups"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

const bueBase = "/proxy/cinder/v3/update-project/"

func bueContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}
func bueAPI(cloud *testcloud.Cloud) *backups.API {
	client := cloud.Client("volume", bueBase)
	client.Microversion = "3.43"
	client.MoreHeaders = map[string]string{"X-Source": "original"}
	return backups.New(client)
}
func bueRequest(t *testing.T, req *http.Request, want string) {
	t.Helper()
	var raw []byte
	var err error
	if req.Body != nil {
		raw, err = io.ReadAll(req.Body)
	}
	if err != nil || !bytes.Equal(raw, []byte(want)) || req.Method != http.MethodPut || req.URL.EscapedPath() != bueBase+"backups/id" || req.URL.RawQuery != "" || req.Header.Get("X-Source") != "original" || req.Header.Get("X-Auth-Token") != "test-token" || req.Header.Get("OpenStack-API-Version") != "volume 3.43" || req.Header.Get("X-OpenStack-Volume-API-Version") != "3.43" {
		t.Error("backup update changed literal body, envelope, route or selected native policy", req.Method, req.URL, string(raw), req.Header, err)
	}
}
func bueOperation(t *testing.T, err error) {
	t.Helper()
	var operation *resource.OperationError
	if err == nil || !errors.As(err, &operation) || operation.Operation != "Update" || operation.Resource != "backups" || operation.Cause == nil {
		t.Fatal("lost existing Update operation/cause", err, operation)
	}
}

func TestBackupUpdateTypedFieldsKeepNilOmissionAndExplicitEmptyMetadata(t *testing.T) {
	empty := ""
	name, description := "  literal/name ?#  ", "description\n\x00"
	for _, tc := range []struct {
		name string
		opts backups.UpdateOpts
		body string
	}{
		{"all nil omitted", backups.UpdateOpts{}, `{"backup":{}}`},
		{"metadata nil with literal text", backups.UpdateOpts{Name: &name, Description: &description}, `{"backup":{"description":"description\n\u0000","name":"  literal/name ?#  "}}`},
		{"explicit empty metadata clears", backups.UpdateOpts{Metadata: map[string]string{}}, `{"backup":{"metadata":{}}}`},
		{"empty name description retained", backups.UpdateOpts{Name: &empty, Description: &empty, Metadata: map[string]string{}}, `{"backup":{"description":"","metadata":{},"name":""}}`},
		{"nonempty metadata retains empty value", backups.UpdateOpts{Metadata: map[string]string{"empty": "", "literal": "a/b?%#"}}, `{"backup":{"metadata":{"empty":"","literal":"a/b?%#"}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			api := bueAPI(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				bueRequest(t, req, tc.body)
				testcloud.JSON(w, 200, `{"backup":{"id":"actual-summary-id","name":"actual-summary-name","links":[]}}`)
			})
			actual, err := api.Update(bueContext(t), "id", tc.opts)
			if err != nil || actual == nil || actual.ID != "actual-summary-id" || actual.Name != "actual-summary-name" || actual.Description != "" || actual.Metadata != nil || calls.Load() != 1 {
				t.Fatal("update fabricated submitted fields or skipped explicit clear", actual, err, calls.Load())
			}
		})
	}
}

func TestBackupUpdateExtensionsStayInsideOwnedBackupEnvelopeAndProtectCoreInputs(t *testing.T) {
	cloud := testcloud.New(t)
	api := bueAPI(cloud)
	labels := []string{"original"}
	extension := map[string]any{"enabled": false, "labels": labels}
	field := backups.WithUpdateField("vendor:setting", extension)
	extension["enabled"] = true
	labels[0] = "caller changed"
	oldName, oldDescription, replacement := "old name", "old description", "replacement"
	original := backups.UpdateOpts{Name: &oldName, Description: &oldDescription, Metadata: map[string]string{"old": "old"}}
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		call := calls.Add(1)
		switch call {
		case 1:
			bueRequest(t, req, `{"backup":{"metadata":{},"vendor:setting":{"enabled":false,"labels":["original"]}}}`)
		case 2:
			bueRequest(t, req, `{"backup":{"name":"replacement","vendor:setting":{"enabled":false,"labels":["original"]}}}`)
		default:
			t.Error("core collision sent HTTP", call, req.URL)
			w.WriteHeader(500)
			return
		}
		testcloud.JSON(w, 200, `{"backup":{"id":"actual","name":"server"}}`)
	})
	actual, err := api.Update(bueContext(t), "id", original, field, backups.WithUpdateOptions(backups.UpdateOpts{Metadata: map[string]string{}}))
	if err != nil || actual == nil || actual.ID != "actual" || actual.Name != "server" || actual.Metadata != nil || calls.Load() != 1 {
		t.Fatal(actual, err, calls.Load())
	}
	actual, err = api.Update(bueContext(t), "id", original, field, backups.WithUpdateOptions(backups.UpdateOpts{Name: &replacement}))
	if err != nil || actual == nil || actual.ID != "actual" || calls.Load() != 2 || oldName != "old name" || oldDescription != "old description" || original.Metadata["old"] != "old" {
		t.Fatal("replacement mutated original or reused caller extension contents", actual, err, calls.Load(), original)
	}
	for _, key := range []string{"name", "description", "metadata"} {
		for _, present := range []bool{false, true} {
			opts := backups.UpdateOpts{}
			if present {
				opts = original
			}
			actual, err = api.Update(bueContext(t), "id", opts, backups.WithUpdateField(key, "smuggled core"))
			bueOperation(t, err)
			if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 2 {
				t.Fatal("core field replaced typed input or omission", key, present, actual, err, calls.Load())
			}
			if actual != nil && (actual.ID != "" || actual.Name != "" || actual.Description != "" || actual.Metadata != nil) {
				t.Fatal("local collision acquired a backup response", actual)
			}
		}
	}
}

func TestBackupUpdateExtractsActualSummaryAndKeepsNativeAndOptionFailures(t *testing.T) {
	for _, tc := range []struct {
		name, reply  string
		metadataKind string
	}{
		{"summary omits submitted metadata", `{"backup":{"id":"response-id","name":"response-name","links":[]}}`, "absent"},
		{"explicit server metadata wins", `{"backup":{"id":"response-id","name":"response-name","metadata":{"server":"actual"}}}`, "server"},
		{"explicit null metadata", `{"backup":{"id":"response-id","name":"response-name","metadata":null}}`, "absent"},
		{"explicit empty metadata", `{"backup":{"id":"response-id","name":"response-name","metadata":{}}}`, "empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			api := bueAPI(cloud)
			name, description := "submitted-name", "submitted-description"
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				bueRequest(t, req, `{"backup":{"description":"submitted-description","metadata":{"client":"submitted"},"name":"submitted-name"}}`)
				testcloud.JSON(w, 200, tc.reply)
			})
			actual, err := api.Update(bueContext(t), "id", backups.UpdateOpts{Name: &name, Description: &description, Metadata: map[string]string{"client": "submitted"}})
			if err != nil || actual == nil || actual.ID != "response-id" || actual.Name != "response-name" || actual.Description != "" || calls.Load() != 1 {
				t.Fatal(actual, err, calls.Load())
			}
			switch tc.metadataKind {
			case "absent":
				if actual.Metadata != nil {
					t.Fatal("submitted metadata echoed despite absent/null response", actual.Metadata)
				}
			case "server":
				if actual.Metadata == nil || len(*actual.Metadata) != 1 || (*actual.Metadata)["server"] != "actual" {
					t.Fatal("response metadata replaced by submitted metadata", actual.Metadata)
				}
			case "empty":
				if actual.Metadata == nil || *actual.Metadata == nil || len(*actual.Metadata) != 0 {
					t.Fatal("lost explicit empty response metadata", actual.Metadata)
				}
			}
		})
	}
	for _, code := range []int{201, 400, 403, 404} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			api := bueAPI(cloud)
			reply := `{"backup":{"id":"rejected-must-not-extract","metadata":{"client":"poison"}}}`
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				bueRequest(t, req, `{"backup":{"metadata":{}}}`)
				w.Header().Set("X-Proof", "actual rejected response")
				testcloud.JSON(w, code, reply)
			})
			actual, err := api.Update(bueContext(t), "id", backups.UpdateOpts{Metadata: map[string]string{}})
			bueOperation(t, err)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || len(native.Expected) != 1 || native.Expected[0] != 200 || string(native.Body) != reply || native.ResponseHeader.Get("X-Proof") != "actual rejected response" || calls.Load() != 1 {
				t.Fatal("native Update status/cause/body/header changed", actual, err, native, calls.Load())
			}
			if actual != nil && (actual.ID != "" || actual.Name != "" || actual.Metadata != nil) {
				t.Fatal("native rejection extracted its error payload", actual)
			}
		})
	}
	for _, kind := range []string{"nil option", "callback error"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			api := bueAPI(cloud)
			failure := errors.New("caller update option failed")
			var calls, originals atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) { calls.Add(1); w.WriteHeader(500) })
			var option backups.UpdateOption
			want := resource.ErrInvalidOption
			if kind == "callback error" {
				want = failure
				option = func(config *request.Config[backups.UpdateOpts]) error { originals.Add(1); return failure }
			}
			actual, err := api.Update(bueContext(t), "id", backups.UpdateOpts{}, option)
			bueOperation(t, err)
			expectedOriginals := int32(0)
			if kind == "callback error" {
				expectedOriginals = 1
			}
			if actual != nil || !errors.Is(err, want) || calls.Load() != 0 || originals.Load() != expectedOriginals {
				t.Fatal("option error did HTTP or lost original cause", actual, err, calls.Load(), originals.Load())
			}
		})
	}
}
