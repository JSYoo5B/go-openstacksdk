package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func imageRecordUploadFormats() []ImageRecordUploadOption {
	return []ImageRecordUploadOption{WithImageRecordUploadContainerFormat("bare"), WithImageRecordUploadDiskFormat("qcow2")}
}

func TestImageRecordUploadFormalFormatsRequireTruthinessWithoutNameOrDefaults(t *testing.T) {
	for _, test := range []struct {
		name            string
		container, disk any
		valid           bool
	}{
		{"both absent", nil, nil, false}, {"null container", nil, "qcow2", false}, {"empty disk", "bare", "", false},
		{"false container", false, "qcow2", false}, {"zero disk", "bare", 0, false}, {"empty array", []any{}, "qcow2", false},
		{"empty object", "bare", map[string]any{}, false}, {"ordinary", "bare", "qcow2", true},
		{"unknown strings", "new-container", "future-disk", true}, {"nonstring raw", []any{"container"}, map[string]any{"disk": true}, true},
		{"numeric and bool", 1, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					if req.Method != http.MethodPost || req.URL.EscapedPath() != "/reverse/glance/v2/images" || req.URL.RawQuery != "" {
						t.Fatal(req.Method, req.URL)
					}
					body := taskCorePayload(t, req)
					expectedContainer, _ := json.Marshal(test.container)
					expectedDisk, _ := json.Marshal(test.disk)
					if len(body) != 2 || string(body["container_format"]) != string(expectedContainer) || string(body["disk_format"]) != string(expectedDisk) {
						t.Fatal(body)
					}
					return taskCoreJSON(req, 201, `{"id":"server","status":"active"}`), nil
				}
				if calls != 2 || req.Method != http.MethodPut || req.URL.EscapedPath() != "/reverse/glance/v2/images/server/file" || imageRecordStageRead(t, req) != "" {
					t.Fatal(calls, req.Method, req.URL)
				}
				return taskCoreJSON(req, 204, ""), nil
			})
			options := []ImageRecordUploadOption{}
			if test.container != nil {
				options = append(options, WithImageRecordUploadContainerFormat(test.container))
			}
			if test.disk != nil {
				options = append(options, WithImageRecordUploadDiskFormat(test.disk))
			}
			got, err := New(client).UploadImageRecord(context.Background(), ImageRecordUploadRequest{}, options...)
			if !test.valid {
				if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
					t.Fatal(got, err, calls)
				}
				return
			}
			if got == nil || got.Record == nil || got.Metadata == nil || got.Uploaded == nil || err != nil || calls != 2 || string(got.Record.Resource.Body["visibility"]) != "null" || string(got.Record.Resource.Body["name"]) != "null" || string(got.Record.Resource.Body["status"]) != `"active"` {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordUploadAllDeclaredFieldsSubmitRawWireWithoutProjectionDefaults(t *testing.T) {
	// Pinned Source inventory is independent of the production descriptor table.
	attrs := map[string]any{}
	want := map[string]json.RawMessage{}
	aliases := map[string]string{"is_hidden": "os_hidden", "is_protected": "protected", "hash_algo": "os_hash_algo", "hash_value": "os_hash_value", "owner_id": "owner", "needs_config_drive": "img_config_drive", "needs_secure_boot": "os_secure_boot", "is_hw_vif_multiqueue_enabled": "hw_vif_multiqueue_enabled", "is_hw_boot_menu_enabled": "hw_boot_menu", "has_auto_disk_config": "auto_disk_config"}
	for _, key := range strings.Fields(imageRecordViewKeys) {
		if key == "location" {
			continue
		}
		attrs[key] = json.RawMessage("null")
		wire := key
		if alias, ok := aliases[key]; ok {
			wire = alias
		}
		want[wire] = json.RawMessage("null")
	}
	attrs["id"] = "constructor"
	attrs["properties"] = map[string]any{"vendor": json.RawMessage(`900719925474099312345`)}
	want["id"] = json.RawMessage(`"constructor"`)
	want["container_format"] = json.RawMessage(`"bare"`)
	want["disk_format"] = json.RawMessage(`"qcow2"`)
	delete(want, "properties")
	want["vendor"] = json.RawMessage(`900719925474099312345`)
	expected, _ := json.Marshal(want)
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			imageRecordImportPayload(t, req, string(expected))
			return taskCoreJSON(req, 203, `{"id":"server","owner":"response owner"}`), nil
		}
		if calls != 2 || req.Method != http.MethodPut || req.URL.EscapedPath() != "/reverse/glance/v2/images/server/file" {
			t.Fatal(calls, req.Method, req.URL)
		}
		return taskCoreJSON(req, 299, "opaque"), nil
	})
	got, err := New(client).UploadImageRecord(context.Background(), ImageRecordUploadRequest{Attributes: attrs}, imageRecordUploadFormats()...)
	if err != nil || got == nil || got.Record == nil || calls != 2 || len(got.Record.Resource.Body) != 65 || len(got.Record.bodyState.dirty) != 0 || !reflect.DeepEqual(got.Record.bodyState.current, got.Record.bodyState.original) {
		t.Fatal(got, err, calls)
	}
	for _, key := range strings.Fields(imageRecordViewKeys) {
		if _, ok := got.Record.Resource.Body[key]; !ok {
			t.Fatal("declared field missing", key)
		}
	}
	if string(got.Record.Resource.Body["owner"]) != `"response owner"` || string(got.Record.Resource.Body["owner_id"]) != `"response owner"` || string(got.Record.Wire.Body["owner"]) != `"response owner"` || string(got.Record.bodyState.current["container_format"]) != `"bare"` {
		t.Fatal(got.Record)
	}
}

func TestImageRecordUploadDescriptorViewAndMetadataReceiptsRemainIndependent(t *testing.T) {
	attrs := map[string]any{"id": "seed", "size": "04", "is_protected": "false", "is_hw_vif_multiqueue_enabled": "false", "instance_type_rxtx_factor": "1.25", "hw_qemu_guest_agent": false, "owner_id": "owner", "min_ram": json.RawMessage(`1e9999`), "tags": []any{"a", 2}, "vendor": json.RawMessage(`1.00000000000000000001`), "OpenStack-image-import-methods": "constructor"}
	calls, locations := 0, 0
	cloud := "captured"
	var metadataHeader, uploadHeader http.Header
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			imageRecordImportPayload(t, req, `{"container_format":"bare","disk_format":"qcow2","id":"seed","size":"04","protected":"false","hw_vif_multiqueue_enabled":"false","instance_type_rxtx_factor":"1.25","hw_qemu_guest_agent":false,"owner":"owner","min_ram":1e9999,"tags":["a",2],"vendor":1.00000000000000000001}`)
			response := taskCoreJSON(req, 203, `{"id":"server","self":"https://foreign.test/decoy","location":{"cloud":"wire"},"name":["passive",null]}`)
			response.Header.Set("OpenStack-image-import-methods", "direct, web")
			metadataHeader = response.Header
			response.Header.Set("Location", "https://foreign.test/redirect")
			return response, nil
		}
		if calls != 2 || req.Method != http.MethodPut || req.URL.EscapedPath() != "/reverse/glance/v2/images/server/file" {
			t.Fatal(calls, req.Method, req.URL)
		}
		response := taskCoreJSON(req, 299, `{"id":"PUT decoy","status":"active"}`)
		response.Header.Set("OpenStack-image-import-methods", "must not consume")
		uploadHeader = response.Header
		return response, nil
	})
	service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return resource.CloudLocation{Cloud: &cloud}, nil }})
	got, err := service.UploadImageRecord(context.Background(), ImageRecordUploadRequest{Attributes: attrs}, imageRecordUploadFormats()...)
	if err != nil || got == nil || got.Record == nil || calls != 2 || locations != 1 || got.Record.StatusCode != 203 || got.Metadata.StatusCode != 203 || got.Uploaded.StatusCode != 299 {
		t.Fatal(got, err, calls, locations)
	}
	for key, value := range map[string]string{"id": `"server"`, "size": "4", "is_protected": "true", "is_hw_vif_multiqueue_enabled": "false", "instance_type_rxtx_factor": "1.25", "hw_qemu_guest_agent": `"False"`, "min_ram": "1e9999", "status": "null", "tags": `["a",2]`} {
		th.CheckEquals(t, value, string(got.Record.Resource.Body[key]))
	}
	th.CheckDeepEquals(t, []string{"direct", " web"}, got.Record.ImportMethods)
	if string(got.Record.bodyState.current["size"]) != `"04"` || string(got.Record.bodyState.current["is_protected"]) != `"false"` || got.Record.Wire == nil || string(got.Record.Wire.Body["self"]) != `"https://foreign.test/decoy"` {
		t.Fatal("projection wrote raw seed", got.Record)
	}
	var location resource.CloudLocation
	th.AssertNoErr(t, json.Unmarshal(got.Record.Resource.Body["location"], &location))
	th.AssertEquals(t, "captured", *location.Cloud)
	if string(imageRecordProperties(t, got.Record)["location"]) != `{"cloud":"wire"}` {
		t.Fatal(got.Record.Resource.Body)
	}
	envelope := string(got.Record.Envelope)
	got.Metadata.Body[0] = '!'
	got.Metadata.Header.Set("X-Task-Proof", "changed")
	got.Uploaded.Body[0] = '!'
	got.Uploaded.Header.Set("X-Task-Proof", "changed")
	got.Record.Resource.Body["id"][1] = '!'
	got.Record.Header.Set("X-Task-Proof", "changed")
	if string(got.Record.Envelope) != envelope || metadataHeader.Get("X-Task-Proof") != "actual" || uploadHeader.Get("X-Task-Proof") != "actual" || string(got.Record.bodyState.current["id"]) != `"server"` || attrs["id"] != "seed" {
		t.Fatal("metadata channels alias")
	}
}

func TestImageRecordUploadConflictingAttributesAndPropertiesFollowSourceConstructorOrder(t *testing.T) {
	for _, test := range []struct {
		name  string
		attrs map[string]any
		want  string
	}{
		{"truthy hook overrides then disappears", map[string]any{"name": "before", "__conflicting_attrs": map[string]any{"name": "after", "disk_format": nil, "base_path": "metadata path", "resource_type": "metadata type"}}, `{"container_format":"bare","disk_format":null,"name":"after","base_path":"metadata path","resource_type":"metadata type"}`},
		{"false hook retained", map[string]any{"__conflicting_attrs": false}, `{"container_format":"bare","disk_format":"qcow2","__conflicting_attrs":false}`},
		{"empty hook retained", map[string]any{"__conflicting_attrs": map[string]any{}}, `{"container_format":"bare","disk_format":"qcow2","__conflicting_attrs":{}}`},
		{"null hook retained", map[string]any{"__conflicting_attrs": nil}, `{"container_format":"bare","disk_format":"qcow2","__conflicting_attrs":null}`},
		{"properties override formal formats and ID", map[string]any{"id": "flat", "properties": map[string]any{"id": "property", "container_format": false, "disk_format": nil, "name": "from properties"}}, `{"id":"property","container_format":false,"disk_format":null,"name":"from properties"}`},
		{"truthy string properties", map[string]any{"properties": "literal"}, `{"container_format":"bare","disk_format":"qcow2","properties":"literal"}`},
		{"truthy scalar properties omitted", map[string]any{"properties": 42}, `{"container_format":"bare","disk_format":"qcow2"}`},
		{"empty properties omitted", map[string]any{"properties": map[string]any{}}, `{"container_format":"bare","disk_format":"qcow2"}`},
		{"truthy list properties omitted", map[string]any{"properties": []any{1}}, `{"container_format":"bare","disk_format":"qcow2"}`},
		{"filename is only metadata", map[string]any{"filename": "/no-such-owned-upload-file", "data": "ordinary metadata"}, `{"container_format":"bare","disk_format":"qcow2","filename":"/no-such-owned-upload-file","data":"ordinary metadata"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					imageRecordImportPayload(t, req, test.want)
					return taskCoreJSON(req, 201, `{"id":"server"}`), nil
				}
				if calls != 2 || req.Method != http.MethodPut || imageRecordStageRead(t, req) != "" {
					t.Fatal(calls, req.Method)
				}
				return taskCoreJSON(req, 204, ""), nil
			})
			got, err := New(client).UploadImageRecord(context.Background(), ImageRecordUploadRequest{Attributes: test.attrs}, imageRecordUploadFormats()...)
			if got == nil || err != nil || calls != 2 {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordUploadRejectsFixedControlsAndInvalidConstructorBeforeHTTP(t *testing.T) {
	cases := []struct {
		name   string
		attrs  map[string]any
		option ImageRecordUploadOption
	}{
		{"truthy nonobject hook", map[string]any{"__conflicting_attrs": []any{1}}, nil},
		{"integer projection overflow", map[string]any{"size": json.RawMessage(`1e9999`)}, nil},
		{"float projection invalid", map[string]any{"instance_type_rxtx_factor": "invalid"}, nil},
		{"boolstr projection invalid", map[string]any{"is_hw_vif_multiqueue_enabled": "invalid"}, nil},
		{"invalid raw JSON", map[string]any{"vendor": json.RawMessage(`{`)}, nil},
		{"unsupported value", map[string]any{"vendor": make(chan int)}, nil},
		{"invalid UTF8", map[string]any{"vendor": string([]byte{0xff})}, nil},
		{"Content-Type owned", nil, WithImageRecordUploadHeader("Content-Type", "spoof")},
		{"Accept owned", nil, WithImageRecordUploadHeader("Accept", "spoof")},
		{"Size header owned", nil, WithImageRecordUploadHeader("X-OpenStack-Image-Size", "spoof")},
	}
	for _, key := range []string{"base_path", "resource_type", "self", "connection", "_synchronized", "microversion"} {
		cases = append(cases, struct {
			name   string
			attrs  map[string]any
			option ImageRecordUploadOption
		}{"top " + key, map[string]any{key: "spoof"}, nil})
	}
	for _, key := range []string{"self", "connection", "_synchronized", "microversion"} {
		cases = append(cases, struct {
			name   string
			attrs  map[string]any
			option ImageRecordUploadOption
		}{"inner " + key, map[string]any{"__conflicting_attrs": map[string]any{key: "spoof"}}, nil})
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(*http.Request) (*http.Response, error) {
				calls++
				t.Fatal("invalid constructor dispatched")
				return nil, nil
			})
			options := imageRecordUploadFormats()
			if test.option != nil {
				options = append(options, test.option)
			}
			got, err := New(client).UploadImageRecord(context.Background(), ImageRecordUploadRequest{Attributes: test.attrs}, options...)
			if got != nil || err == nil || calls != 0 {
				t.Fatal(got, err, calls)
			}
		})
	}
	t.Run("typed nil reader", func(t *testing.T) {
		calls := 0
		client := taskCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
		var reader *strings.Reader
		got, err := New(client).UploadImageRecord(context.Background(), ImageRecordUploadRequest{Data: reader}, imageRecordUploadFormats()...)
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatal(got, err, calls)
		}
	})
}

func TestImageRecordUploadCreationResponseMayReplaceArbitraryInitialIdentity(t *testing.T) {
	for _, initial := range []string{`null`, `false`, `42`, `[]`, `{"id":"not identity"}`, `""`, `".."`, `"constructor"`} {
		t.Run(initial, func(t *testing.T) {
			calls := 0
			id := "a /한:%?\\b"
			rawID, _ := json.Marshal(id)
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					body := taskCorePayload(t, req)
					th.AssertEquals(t, initial, string(body["id"]))
					return taskCoreJSON(req, 201, `{"id":`+string(rawID)+`,"file":"https://foreign.test/decoy"}`), nil
				}
				if calls != 2 || req.Method != http.MethodPut || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(id)+"/file" || req.URL.RawQuery != "" {
					t.Fatal(calls, req.Method, req.URL)
				}
				return taskCoreJSON(req, 204, "opaque"), nil
			})
			got, err := New(client).UploadImageRecord(context.Background(), ImageRecordUploadRequest{Attributes: map[string]any{"id": json.RawMessage(initial)}}, imageRecordUploadFormats()...)
			if err != nil || got == nil || got.Uploaded == nil || calls != 2 || string(got.Record.bodyState.current["id"]) != string(rawID) || len(got.Record.bodyState.dirty) != 0 {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordUploadInvalidMetadataSyntaxPreservesConstructorDirtyIDAndLaterCommit(t *testing.T) {
	for _, raw := range []string{"", "not JSON", `{"broken":`} {
		t.Run(raw, func(t *testing.T) {
			calls := 0
			var handler taskCoreTransport
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			handler = func(req *http.Request) (*http.Response, error) {
				if calls == 1 {
					imageRecordImportPayload(t, req, `{"id":"seed","container_format":"bare","disk_format":"qcow2","name":"pending","vendor":false}`)
					response := taskCoreJSON(req, 299, raw)
					response.Header.Set("OpenStack-image-import-methods", "create")
					return response, nil
				}
				if calls != 2 || req.Method != http.MethodPut {
					t.Fatal(calls, req.Method)
				}
				return taskCoreJSON(req, 204, "opaque"), nil
			}
			got, err := service.UploadImageRecord(context.Background(), ImageRecordUploadRequest{Attributes: map[string]any{"id": "seed", "name": "pending", "vendor": false, "OpenStack-image-import-methods": "constructor"}}, imageRecordUploadFormats()...)
			if err != nil || got == nil || got.Record == nil || got.Record.Wire != nil || calls != 2 || string(got.Record.Envelope) != raw || len(got.Record.bodyState.dirty) != 5 || string(got.Record.bodyState.original["id"]) != `"seed"` || len(got.Record.bodyState.original) != 1 {
				t.Fatal(got, err, calls)
			}
			if _, ok := got.Record.bodyState.dirty["id"]; !ok {
				t.Fatal("metadata constructor id was not dirty")
			}
			th.CheckDeepEquals(t, []string{"create"}, got.Record.ImportMethods)
			before := cloneImageRecord(got.Record)
			handler = func(req *http.Request) (*http.Response, error) {
				if calls != 3 || req.Method != http.MethodPatch || req.URL.EscapedPath() != "/reverse/glance/v2/images/seed" {
					t.Fatal(calls, req.Method, req.URL)
				}
				patches := imageRecordUpdatePayload(t, req)
				paths := map[string]bool{}
				for _, patch := range patches {
					paths[patch.Path] = true
				}
				th.CheckDeepEquals(t, map[string]bool{"/container_format": true, "/disk_format": true, "/name": true, "/vendor": true}, paths)
				return taskCoreJSON(req, 200, `{}`), nil
			}
			updated, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: got.Record})
			if err != nil || updated == nil || calls != 3 || len(updated.bodyState.dirty) != 0 || !reflect.DeepEqual(before, got.Record) {
				t.Fatal(updated, err, calls)
			}
		})
	}
}

func TestImageRecordUploadMetadataTranslationFailuresKeepActualCreatedEvidence(t *testing.T) {
	for _, raw := range []string{`null`, `[]`, `"scalar"`, `42`, `{"id":"server","size":1e9999}`, `{"id":"server","instance_type_rxtx_factor":"invalid"}`, "{\"id\":\"server\",\"name\":\"\xff\"}"} {
		t.Run(raw, func(t *testing.T) {
			calls := 0
			reader := &imageUploadCoreReader{reader: strings.NewReader("untouched")}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls != 1 || req.Method != http.MethodPost {
					t.Fatal(calls, req.Method)
				}
				return taskCoreJSON(req, 203, raw), nil
			})
			got, err := New(client).UploadImageRecord(context.Background(), ImageRecordUploadRequest{Data: reader, Attributes: map[string]any{"id": "seed", "name": "submitted"}}, imageRecordUploadFormats()...)
			if got == nil || got.Record == nil || got.Metadata == nil || got.Uploaded != nil || err == nil || calls != 1 || got.Metadata.StatusCode != 203 || string(got.Metadata.Body) != raw || reader.reads.Load() != 0 || reader.seeks.Load() != 0 || reader.closes.Load() != 0 {
				t.Fatal(got, err, calls, reader)
			}
			taskCoreProof(t, err, 203, raw)
		})
	}
}

func TestImageRecordUploadIdentityValidationOccursOnlyAfterMetadataAndBeforeBinaryIO(t *testing.T) {
	for _, rawID := range []string{"missing", `null`, `false`, `42`, `[]`, `""`, `" "`, `"."`, `".."`, `"line\ncontrol"`, `"\ud800"`} {
		t.Run(rawID, func(t *testing.T) {
			calls := 0
			reader := &imageUploadCoreReader{reader: strings.NewReader("untouched")}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls != 1 || req.Method != http.MethodPost {
					t.Fatal(calls, req.Method)
				}
				response := `{"id":` + rawID + `}`
				if rawID == "missing" {
					response = `{}`
				}
				return taskCoreJSON(req, 201, response), nil
			})
			got, err := New(client).UploadImageRecord(context.Background(), ImageRecordUploadRequest{Data: reader}, imageRecordUploadFormats()...)
			if got == nil || got.Record == nil || got.Metadata == nil || got.Uploaded != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || reader.reads.Load() != 0 || reader.seeks.Load() != 0 {
				t.Fatal(got, err, calls, reader)
			}
		})
	}
}

func TestImageRecordUploadAcceptedBoundariesKeepOpaquePUTAndNeverFetch(t *testing.T) {
	for _, code := range []int{200, 201, 204, 299, 300, 304, 399} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls, redirects := 0, 0
			var header http.Header
			raw := "opaque\xff"
			if code == 204 {
				raw = ""
			}
			if code == 299 {
				raw = `{"id":"PUT decoy","size":1e9999,"status":"active"}`
			}
			if code == 304 {
				raw = "null"
			}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					if req.Method != http.MethodPost {
						t.Fatal(req.Method)
					}
					response := taskCoreJSON(req, code, `{"id":"created","status":"saving"}`)
					response.Header.Set("OpenStack-image-import-methods", "metadata only")
					return response, nil
				}
				if calls != 2 || req.Method != http.MethodPut || req.URL.EscapedPath() != "/reverse/glance/v2/images/created/file" || req.URL.RawQuery != "" || req.GetBody != nil || req.Header.Get("Content-Type") != "application/octet-stream" || req.Header.Get("Accept") != "" {
					t.Fatal(calls, req.Method, req.URL, req.Header)
				}
				response := taskCoreJSON(req, code, raw)
				response.Header.Set("OpenStack-image-import-methods", "ignored upload header")
				response.Header.Set("Location", "https://foreign.test/upload")
				header = response.Header
				return response, nil
			})
			client.ProviderClient.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects++; return nil }
			got, err := New(client).UploadImageRecord(context.Background(), ImageRecordUploadRequest{}, imageRecordUploadFormats()...)
			if err != nil || got == nil || got.Record == nil || got.Uploaded == nil || calls != 2 || redirects != 0 || got.Metadata.StatusCode != code || got.Uploaded.StatusCode != code || string(got.Uploaded.Body) != raw || string(got.Record.Envelope) != `{"id":"created","status":"saving"}` || string(got.Record.Resource.Body["status"]) != `"saving"` || len(got.Record.bodyState.dirty) != 0 {
				t.Fatal(got, err, calls, redirects)
			}
			th.CheckDeepEquals(t, []string{"metadata only"}, got.Record.ImportMethods)
			got.Uploaded.Header.Set("X-Task-Proof", "mutated")
			if len(got.Uploaded.Body) > 0 {
				got.Uploaded.Body[0] = '!'
			}
			if header.Get("X-Task-Proof") != "actual" || got.Record.Header.Get("X-Task-Proof") != "actual" {
				t.Fatal("PUT receipt aliased metadata")
			}
		})
	}
}

func TestImageRecordUploadInfersTotalSizeAfterPOSTAndRestoresBorrowedCursor(t *testing.T) {
	for _, mode := range []string{"default total", "plain reader", "explicit zero", "explicit negative", "inference off", "without restores inference", "explicit overrides disabled", "inference enabled again", "full options replace", "without preserves disabled"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			raw := strings.NewReader("prefixDATA")
			_, _ = raw.Seek(6, io.SeekStart)
			seeker := &imageRecordStageSeeker{reader: raw, action: func(int) {
				if calls != 1 {
					t.Fatal("size inferred before metadata", calls)
				}
			}}
			var data io.Reader = seeker
			options := imageRecordUploadFormats()
			size := "10"
			infer := true
			switch mode {
			case "plain reader":
				data = borrowedUploadReader{Reader: raw}
				size = ""
				infer = false
			case "explicit zero":
				options = append(options, WithImageRecordUploadSize(0))
				size = "0"
				infer = false
			case "explicit negative":
				options = append(options, WithImageRecordUploadSize(-7))
				size = "-7"
				infer = false
			case "inference off":
				options = append(options, WithImageRecordUploadSizeInference(false))
				size = ""
				infer = false
			case "without restores inference":
				options = append(options, WithImageRecordUploadSize(99), WithoutImageRecordUploadSize())
			case "explicit overrides disabled":
				options = append(options, WithImageRecordUploadSizeInference(false), WithImageRecordUploadSize(0))
				size = "0"
				infer = false
			case "inference enabled again":
				options = append(options, WithImageRecordUploadSizeInference(false), WithImageRecordUploadSizeInference(true))
			case "full options replace":
				options = append(options, WithImageRecordUploadSize(-9), WithImageRecordUploadSizeInference(false), WithImageRecordUploadOpts(ImageRecordUploadOpts{ContainerFormat: json.RawMessage(`"bare"`), DiskFormat: json.RawMessage(`"qcow2"`)}))
			case "without preserves disabled":
				options = append(options, WithImageRecordUploadSize(99), WithImageRecordUploadSizeInference(false), WithoutImageRecordUploadSize())
				size = ""
				infer = false
			}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					if len(seeker.whences) != 0 || req.Header.Get("X-OpenStack-Image-Size") != "" {
						t.Fatal(seeker.whences, req.Header)
					}
					return taskCoreJSON(req, 201, `{"id":"created"}`), nil
				}
				if calls != 2 || req.Method != http.MethodPut || req.Header.Get("X-OpenStack-Image-Size") != size || req.ContentLength != 0 || imageRecordStageRead(t, req) != "DATA" {
					t.Fatal(calls, req.Method, req.Header, req.ContentLength)
				}
				return taskCoreJSON(req, 204, ""), nil
			})
			got, err := New(client).UploadImageRecord(context.Background(), ImageRecordUploadRequest{Data: data}, options...)
			if got == nil || err != nil || calls != 2 || got.Record.data != data {
				t.Fatal(got, err, calls)
			}
			if infer {
				th.CheckDeepEquals(t, []int{io.SeekCurrent, io.SeekEnd, io.SeekStart}, seeker.whences)
				th.CheckDeepEquals(t, []int64{0, 0, 6}, seeker.offsets)
			} else if len(seeker.whences) != 0 {
				t.Fatal(seeker.whences)
			}
		})
	}
}

func TestImageRecordUploadSeekFailuresRetainCreatedMetadataAndESPIPEUnknown(t *testing.T) {
	for _, test := range []struct {
		name    string
		at      int
		cause   error
		allowed bool
	}{
		{"current failure", 1, errors.New("current seek"), false}, {"end failure", 2, errors.New("end seek"), false}, {"restore failure", 3, errors.New("restore seek"), false},
		{"current ESPIPE", 1, syscall.ESPIPE, true}, {"end ESPIPE", 2, syscall.ESPIPE, true}, {"restore ESPIPE", 3, syscall.ESPIPE, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			seeker := &imageRecordStageSeeker{reader: strings.NewReader("data"), failAt: test.at, cause: test.cause}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 201, `{"id":"created"}`), nil
				}
				if calls != 2 || req.Method != http.MethodPut || req.Header.Get("X-OpenStack-Image-Size") != "" {
					t.Fatal(calls, req.Method, req.Header)
				}
				return taskCoreJSON(req, 204, ""), nil
			})
			got, err := New(client).UploadImageRecord(context.Background(), ImageRecordUploadRequest{Data: seeker}, imageRecordUploadFormats()...)
			if got == nil || got.Record == nil || got.Metadata == nil || got.Record.data != seeker {
				t.Fatal(got, err)
			}
			if test.allowed {
				if err != nil || calls != 2 || got.Uploaded == nil {
					t.Fatal(got, err, calls)
				}
			} else if !errors.Is(err, test.cause) || calls != 1 || got.Uploaded != nil {
				t.Fatal(got, err, calls)
			}
		})
	}
	for _, mode := range []string{"cancel during Seek", "source during Seek"} {
		t.Run(mode, func(t *testing.T) {
			marker := errors.New("upload seek canceled")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls != 1 {
					t.Fatal("guard failed beforePUT", calls)
				}
				return taskCoreJSON(req, 201, `{"id":"created"}`), nil
			})
			seeker := &imageRecordStageSeeker{reader: strings.NewReader("data"), action: func(n int) {
				if n == 1 {
					if strings.HasPrefix(mode, "source") {
						client.Endpoint = "https://foreign.test/"
					} else {
						cancel(marker)
					}
				}
			}}
			got, err := New(client).UploadImageRecord(ctx, ImageRecordUploadRequest{Data: seeker}, imageRecordUploadFormats()...)
			if got == nil || got.Metadata == nil || got.Record == nil || got.Uploaded != nil || err == nil || calls != 1 || len(seeker.whences) != 1 {
				t.Fatal(got, err, calls, seeker.whences)
			}
			if strings.HasPrefix(mode, "source") {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, marker) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordUploadCapturesAttributesOptionsLocationAndHeadersOnce(t *testing.T) {
	calls, callbacks, locations := 0, 0, 0
	cloud := "captured cloud"
	headers := map[string]string{"X-Note": "captured"}
	attrs := map[string]any{"name": "captured option", "vendor": []any{"first"}}
	size := int64(0)
	base := WithImageRecordUploadOpts(ImageRecordUploadOpts{ContainerFormat: json.RawMessage(`"bare"`), DiskFormat: json.RawMessage(`"qcow2"`), Attributes: attrs, Headers: headers, Size: &size})
	attrs["name"] = "external changed"
	attrs["vendor"].([]any)[0] = "external changed"
	headers["X-Note"] = "external changed"
	size = 9
	requestAttrs := map[string]any{"name": "request", "request_only": true}
	reader := &imageUploadCoreReader{reader: strings.NewReader("captured data")}
	var retained *ImageRecordUploadOpts
	options := []ImageRecordUploadOption{base, func(config *ImageRecordUploadOpts) error {
		callbacks++
		retained = config
		return WithImageRecordUploadHeader("X-Added", "yes")(config)
	}}
	var client *gophercloud.ServiceClient
	client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Source") != "captured source" || req.Header.Get("X-Note") != "captured" || req.Header.Get("X-Added") != "yes" || req.Header.Get("X-Auth-Token") != "live token" {
			t.Fatal(req.Header)
		}
		if calls == 1 {
			imageRecordImportPayload(t, req, `{"container_format":"bare","disk_format":"qcow2","name":"captured option","vendor":["first"],"request_only":true}`)
			retained.Headers["X-Note"] = "retained changed"
			*retained.Size = 12
			retained.Attributes["name"] = "retained changed"
			cloud = "later"
			return taskCoreJSON(req, 201, `{"id":"created"}`), nil
		}
		if calls != 2 || req.Header.Get("X-OpenStack-Image-Size") != "0" || imageRecordStageRead(t, req) != "captured data" {
			t.Fatal(calls, req.Header)
		}
		return taskCoreJSON(req, 204, ""), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "captured source", "Content-Type": "configured metadata", "Accept": "configured accept"}
	service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
		locations++
		requestAttrs["request_only"] = false
		options[1] = func(*ImageRecordUploadOpts) error { t.Fatal("uncaptured option slice"); return nil }
		client.MoreHeaders["X-Source"] = "later source"
		client.SetToken("live token")
		return resource.CloudLocation{Cloud: &cloud}, nil
	}})
	got, err := service.UploadImageRecord(context.Background(), ImageRecordUploadRequest{Data: reader, Attributes: requestAttrs}, options...)
	if got == nil || err != nil || calls != 2 || callbacks != 1 || locations != 1 || got.Record.data != reader || reader.closes.Load() != 0 || reader.seeks.Load() != 0 {
		t.Fatal(got, err, calls, callbacks, locations, reader)
	}
	var facts resource.CloudLocation
	th.AssertNoErr(t, json.Unmarshal(got.Record.Resource.Body["location"], &facts))
	th.AssertEquals(t, "captured cloud", *facts.Cloud)
}

func TestImageRecordUploadOptionMergeReplacementAndStickyCallbackGuards(t *testing.T) {
	for _, mode := range []string{"bulk merge", "full replace", "cancel", "source", "binding", "outer", "callback cause"} {
		t.Run(mode, func(t *testing.T) {
			calls, callbacks, later := 0, 0, 0
			marker := errors.New("upload option guard")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			outerBad := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerBad {
					return marker
				}
				return nil
			})
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					want := `{"container_format":"bare","disk_format":"qcow2","request":true,"final":"new"}`
					if mode == "bulk merge" {
						want = `{"container_format":"bare","disk_format":"qcow2","request":true,"discarded":"old","final":"new"}`
					}
					imageRecordImportPayload(t, req, want)
					header := ""
					if mode == "bulk merge" {
						header = "old"
					}
					th.AssertEquals(t, header, req.Header.Get("X-Discarded"))
					th.AssertEquals(t, "new", req.Header.Get("X-Final"))
					return taskCoreJSON(req, 201, `{"id":"created"}`), nil
				}
				return taskCoreJSON(req, 204, ""), nil
			})
			service := New(client)
			options := append(imageRecordUploadFormats(), WithImageRecordUploadAttribute("discarded", "old"), WithImageRecordUploadHeader("X-Discarded", "old"))
			if mode == "bulk merge" {
				options = append(options, WithImageRecordUploadAttributes(map[string]any{"final": "new"}), WithImageRecordUploadHeaders(map[string]string{"X-Final": "new"}))
			} else if mode == "full replace" {
				options = append(options, WithImageRecordUploadOpts(ImageRecordUploadOpts{ContainerFormat: json.RawMessage(`"bare"`), DiskFormat: json.RawMessage(`"qcow2"`), Attributes: map[string]any{"final": "new"}, Headers: map[string]string{"X-Final": "new"}}))
			} else {
				options = append(options, func(*ImageRecordUploadOpts) error {
					callbacks++
					switch mode {
					case "cancel":
						cancel(marker)
					case "source":
						client.Endpoint = "https://foreign.test/"
					case "binding":
						service.API = nil
					case "outer":
						outerBad = true
					case "callback cause":
						return marker
					}
					return nil
				}, func(*ImageRecordUploadOpts) error { later++; return nil })
			}
			got, err := service.UploadImageRecord(ctx, ImageRecordUploadRequest{Attributes: map[string]any{"request": true}}, options...)
			if mode == "bulk merge" || mode == "full replace" {
				if got == nil || err != nil || calls != 2 {
					t.Fatal(got, err, calls)
				}
				return
			}
			if got != nil || err == nil || calls != 0 || callbacks != 1 || later != 0 {
				t.Fatal(got, err, calls, callbacks, later)
			}
			if mode == "source" || mode == "binding" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, marker) {
				t.Fatal(err)
			}
		})
	}
	for _, mode := range []string{"request marshaler source", "request marshaler cancel", "option marshaler source"} {
		t.Run(mode, func(t *testing.T) {
			calls, later := 0, 0
			marker := errors.New("upload marshaler guard")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			client := taskCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			value := imageRecordMarshalCallback(func() ([]byte, error) {
				if strings.Contains(mode, "source") {
					client.Endpoint = "https://foreign.test/"
				} else {
					cancel(marker)
				}
				return []byte(`"captured"`), nil
			})
			request := ImageRecordUploadRequest{}
			options := imageRecordUploadFormats()
			if strings.HasPrefix(mode, "request") {
				request.Attributes = map[string]any{"vendor": value}
			} else {
				options = append(options, func(config *ImageRecordUploadOpts) error {
					config.Attributes = map[string]any{"vendor": value}
					return nil
				})
			}
			options = append(options, func(*ImageRecordUploadOpts) error { later++; return nil })
			got, err := New(client).UploadImageRecord(ctx, request, options...)
			if got != nil || err == nil || calls != 0 || later != 0 {
				t.Fatal(got, err, calls, later)
			}
			if strings.Contains(mode, "source") {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, marker) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordUploadAcceptedPhysicalFailuresKeepActualPhaseReceipts(t *testing.T) {
	for _, phase := range []string{"metadata", "upload"} {
		for _, mode := range []string{"read", "close", "cancel", "source restored on Close", "outer restored on Close"} {
			t.Run(phase+"/"+mode, func(t *testing.T) {
				calls, retries := 0, 0
				marker := errors.New("upload physical phase")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				outerBad := false
				ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
					if outerBad {
						return marker
					}
					return nil
				})
				var client *gophercloud.ServiceClient
				body := &taskCoreBody{reader: strings.NewReader("actual phase bytes")}
				action := func() {}
				switch mode {
				case "read":
					body.reader = &taskCoreReader{body: "actual phase bytes", err: marker}
				case "close":
					body.closeErr = marker
				case "cancel":
					action = func() { cancel(marker) }
				case "source restored on Close":
					action = func() { client.Endpoint = "https://foreign.test/" }
				case "outer restored on Close":
					action = func() { outerBad = true }
				}
				if mode != "read" {
					body.reader = &taskCoreReader{body: "actual phase bytes", err: io.EOF, action: action}
				}
				var selected io.ReadCloser = body
				if strings.Contains(mode, "restored") {
					selected = &imageRecordCloseBody{taskCoreBody: body, after: func() { client.Endpoint = "https://glance.example/reverse/glance/v2/"; outerBad = false }}
				}
				client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					if phase == "upload" && calls == 1 {
						return taskCoreJSON(req, 201, `{"id":"created"}`), nil
					}
					expected := http.MethodPost
					if phase == "upload" {
						expected = http.MethodPut
					}
					if req.Method != expected {
						t.Fatal(req.Method)
					}
					return taskCoreHTTP(req, 299, selected), nil
				})
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					retries++
					return err
				}
				reader := &imageUploadCoreReader{reader: strings.NewReader("borrowed")}
				options := append(imageRecordUploadFormats(), WithImageRecordUploadSize(0))
				got, err := New(client).UploadImageRecord(ctx, ImageRecordUploadRequest{Data: reader, Attributes: map[string]any{"id": "seed", "name": "submitted"}}, options...)
				if got == nil || got.Record == nil || got.Metadata == nil || err == nil || retries != 0 || body.closes != 1 || reader.closes.Load() != 0 || reader.seeks.Load() != 0 {
					t.Fatal(got, err, calls, retries, body.closes, reader)
				}
				if phase == "metadata" {
					if calls != 1 || got.Uploaded != nil || got.Metadata.StatusCode != 299 || string(got.Metadata.Body) != "actual phase bytes" || reader.reads.Load() != 0 || got.Record.data != nil {
						t.Fatal(got, calls, reader)
					}
				} else {
					if calls != 2 || got.Uploaded == nil || got.Uploaded.StatusCode != 299 || string(got.Uploaded.Body) != "actual phase bytes" || got.Metadata.StatusCode != 201 || got.Record.StatusCode != 201 || got.Record.data != reader {
						t.Fatal(got, calls)
					}
				}
				if strings.HasPrefix(mode, "source") {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, marker) {
					t.Fatal(err)
				}
				taskCoreProof(t, err, 299, "actual phase bytes")
			})
		}
	}
}

func TestImageRecordUploadMetadataRetryKeepsSafeJSONScopeHeadersAndLiveAuth(t *testing.T) {
	for _, mode := range []string{"ordinary retry", "expanded OkCodes", "changed body", "owned headers", "source mutation", "hook cause", "reauth"} {
		t.Run(mode, func(t *testing.T) {
			calls, retries, reauth := 0, 0, 0
			marker := errors.New("upload metadata native hook")
			var client *gophercloud.ServiceClient
			client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls <= 2 {
					if req.Method != http.MethodPost || req.URL.EscapedPath() != "/reverse/glance/v2/images" || req.URL.RawQuery != "" {
						t.Fatal(calls, req.Method, req.URL)
					}
					imageRecordImportPayload(t, req, `{"container_format":"bare","disk_format":"qcow2","name":"once"}`)
					if calls == 1 {
						code := 503
						if mode == "reauth" {
							code = 401
						}
						return taskCoreJSON(req, code, "initial rejection"), nil
					}
					if req.Header.Get("X-Auth-Token") != "retry token" || req.Header.Get("Content-Type") != "application/json" || req.Header.Get("Accept") != "application/json" {
						t.Fatal(req.Header)
					}
					if mode == "expanded OkCodes" {
						return taskCoreJSON(req, 418, "actual rejection"), nil
					}
					return taskCoreJSON(req, 203, `{"id":"created"}`), nil
				}
				if calls != 3 || req.Method != http.MethodPut || req.Header.Get("X-Auth-Token") != "retry token" {
					t.Fatal(calls, req.Method, req.Header)
				}
				return taskCoreJSON(req, 204, ""), nil
			})
			client.RetryFunc = func(_ context.Context, method, endpoint string, opts *gophercloud.RequestOpts, original error, count uint) error {
				retries++
				if method != http.MethodPost || endpoint != "https://glance.example/reverse/glance/v2/images" {
					t.Fatal(method, endpoint)
				}
				if count > 1 || mode == "reauth" {
					return original
				}
				client.SetToken("retry token")
				opts.MoreHeaders = map[string]string{"X-Retry": "ordinary", "Content-Type": "application/json", "Accept": "application/json"}
				switch mode {
				case "expanded OkCodes":
					opts.OkCodes = append(opts.OkCodes, 418)
				case "changed body":
					opts.JSONBody = map[string]any{"spoof": true}
				case "owned headers":
					opts.MoreHeaders["Content-Type"] = "spoof"
				case "source mutation":
					client.Endpoint = "https://foreign.test/"
				case "hook cause":
					return errors.Join(original, marker)
				}
				return nil
			}
			if mode == "reauth" {
				client.ProviderClient.ReauthFunc = func(context.Context) error { reauth++; client.SetToken("retry token"); return nil }
			}
			options := append(imageRecordUploadFormats(), WithImageRecordUploadAttribute("name", "once"))
			got, err := New(client).UploadImageRecord(context.Background(), ImageRecordUploadRequest{}, options...)
			if mode == "ordinary retry" || mode == "reauth" {
				if got == nil || got.Record == nil || got.Metadata == nil || got.Uploaded == nil || err != nil || calls != 3 || got.Metadata.StatusCode != 203 {
					t.Fatal(got, err, calls, retries, reauth)
				}
				if mode == "reauth" && reauth != 1 {
					t.Fatal(reauth)
				}
				return
			}
			expectedCalls, code, body := 1, 503, "initial rejection"
			if mode == "expanded OkCodes" {
				expectedCalls, code, body = 2, 418, "actual rejection"
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if got != nil || err == nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != body || calls != expectedCalls {
				t.Fatal(got, err, native, calls, retries)
			}
			if mode == "changed body" || mode == "owned headers" || mode == "source mutation" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if mode == "hook cause" && !errors.Is(err, marker) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordUploadBinaryRejectionsNeverReplayReauthenticateOrBackoff(t *testing.T) {
	for _, code := range []int{400, 401, 404, 429, 500, 599} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls, retries, reauth, backoff := 0, 0, 0, 0
			reader := &imageUploadCoreReader{reader: strings.NewReader("prefix-tail")}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 201, `{"id":"created"}`), nil
				}
				if calls != 2 || req.Method != http.MethodPut {
					t.Fatal("binary replay", calls, req.Method)
				}
				p := make([]byte, 6)
				n, err := io.ReadFull(req.Body, p)
				if err != nil || n != 6 || string(p) != "prefix" {
					t.Fatal(n, err, string(p))
				}
				_ = req.Body.Close()
				return taskCoreJSON(req, code, "native upload rejection"), nil
			})
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			client.ProviderClient.ReauthFunc = func(context.Context) error { reauth++; return nil }
			client.ProviderClient.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				backoff++
				return nil
			}
			options := append(imageRecordUploadFormats(), WithImageRecordUploadSizeInference(false))
			got, err := New(client).UploadImageRecord(context.Background(), ImageRecordUploadRequest{Data: reader}, options...)
			var native gophercloud.ErrUnexpectedResponseCode
			if got == nil || got.Record == nil || got.Metadata == nil || got.Uploaded != nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != "native upload rejection" || native.ResponseHeader.Get("X-Task-Proof") != "actual" || calls != 2 || retries != 0 || reauth != 0 || backoff != 0 || reader.closes.Load() != 0 || reader.seeks.Load() != 0 || reader.reader.(*strings.Reader).Len() != 5 || got.Record.data != reader {
				t.Fatal(got, err, native, calls, retries, reauth, backoff, reader)
			}
		})
	}
}

func TestImageRecordUploadBorrowedReadAndTransportFailuresKeepCreatedRecord(t *testing.T) {
	for _, mode := range []string{"reader error", "transport error", "cancel reader", "source reader"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			marker := errors.New("upload borrowed input")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var client *gophercloud.ServiceClient
			input := &taskCoreReader{body: "partial", err: marker}
			if mode == "cancel reader" {
				input.err = io.EOF
				input.action = func() { cancel(marker) }
			}
			if mode == "source reader" {
				input.err = io.EOF
				input.action = func() { client.Endpoint = "https://foreign.test/" }
			}
			client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 201, `{"id":"created"}`), nil
				}
				if calls != 2 || req.Method != http.MethodPut {
					t.Fatal(calls, req.Method)
				}
				if mode == "transport error" {
					return nil, marker
				}
				_, err := io.ReadAll(req.Body)
				if mode == "reader error" {
					return nil, err
				}
				return taskCoreJSON(req, 204, "physical accepted"), nil
			})
			got, err := New(client).UploadImageRecord(ctx, ImageRecordUploadRequest{Data: input}, imageRecordUploadFormats()...)
			if got == nil || got.Record == nil || got.Metadata == nil || err == nil || calls != 2 || got.Record.data != input {
				t.Fatal(got, err, calls)
			}
			if mode == "reader error" || mode == "transport error" {
				if got.Uploaded != nil || !errors.Is(err, marker) {
					t.Fatal(got, err)
				}
			} else {
				if got.Uploaded == nil || string(got.Uploaded.Body) != "physical accepted" {
					t.Fatal(got, err)
				}
				if mode == "source reader" {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, marker) {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestImageRecordUploadRejectedMetadataNeverAssignsDataOrStartsBinary(t *testing.T) {
	for _, code := range []int{400, 401, 404, 429, 500, 599} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls := 0
			reader := &imageUploadCoreReader{reader: strings.NewReader("untouched")}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodPost {
					t.Fatal(req.Method)
				}
				return taskCoreJSON(req, code, "native metadata rejection"), nil
			})
			got, err := New(client).UploadImageRecord(context.Background(), ImageRecordUploadRequest{Data: reader}, imageRecordUploadFormats()...)
			var native gophercloud.ErrUnexpectedResponseCode
			if got != nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != "native metadata rejection" || native.ResponseHeader.Get("X-Task-Proof") != "actual" || calls != 1 || reader.reads.Load() != 0 || reader.seeks.Load() != 0 || reader.closes.Load() != 0 {
				t.Fatal(got, err, native, calls, reader)
			}
		})
	}
	for _, mode := range []string{"transport", "rejected read", "rejected close"} {
		t.Run(mode, func(t *testing.T) {
			calls, retries := 0, 0
			marker := errors.New("upload metadata failure")
			body := &taskCoreBody{reader: strings.NewReader("native metadata rejection")}
			if mode == "rejected read" {
				body.reader = &taskCoreReader{body: "native metadata rejection", err: marker}
			}
			if mode == "rejected close" {
				body.closeErr = marker
			}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if mode == "transport" {
					return nil, marker
				}
				return taskCoreHTTP(req, 503, body), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				retries++
				return original
			}
			reader := &imageUploadCoreReader{reader: strings.NewReader("untouched")}
			got, err := New(client).UploadImageRecord(context.Background(), ImageRecordUploadRequest{Data: reader}, imageRecordUploadFormats()...)
			if got != nil || !errors.Is(err, marker) || calls != 1 || reader.reads.Load() != 0 || reader.seeks.Load() != 0 || reader.closes.Load() != 0 {
				t.Fatal(got, err, calls, retries, reader)
			}
			if mode != "transport" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != "native metadata rejection" || body.closes != 1 {
					t.Fatal(err, native, body.closes)
				}
			}
		})
	}
}

func TestImageRecordUploadFactoryOptionsRemainReusableAfterResultAndConfigEdits(t *testing.T) {
	attributes := map[string]any{"name": "captured", "nested": map[string]any{"precise": json.RawMessage(`900719925474099312345`)}}
	rawFormat := json.RawMessage(`"bare"`)
	headers := map[string]string{"X-Captured": "yes"}
	size := int64(0)
	option := WithImageRecordUploadOpts(ImageRecordUploadOpts{ContainerFormat: rawFormat, DiskFormat: json.RawMessage(`"qcow2"`), Attributes: attributes, Headers: headers, Size: &size})
	rawFormat[1] = '!'
	attributes["name"] = "changed"
	attributes["nested"].(map[string]any)["precise"] = false
	headers["X-Captured"] = "changed"
	size = 99
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		th.AssertEquals(t, "yes", req.Header.Get("X-Captured"))
		if req.Method == http.MethodPost {
			imageRecordImportPayload(t, req, `{"container_format":"bare","disk_format":"qcow2","name":"captured","nested":{"precise":900719925474099312345}}`)
			return taskCoreJSON(req, 201, `{"id":"created"}`), nil
		}
		if req.Method != http.MethodPut || req.Header.Get("X-OpenStack-Image-Size") != "0" {
			t.Fatal(req.Method, req.Header)
		}
		return taskCoreJSON(req, 204, "opaque"), nil
	})
	service := New(client)
	first, err := service.UploadImageRecord(context.Background(), ImageRecordUploadRequest{}, option)
	th.AssertNoErr(t, err)
	first.Record.Resource.Body["name"][1] = '!'
	first.Record.bodyState.current["container_format"][1] = '!'
	first.Metadata.Header.Set("X-Task-Proof", "first changed")
	first.Metadata.Body[0] = '!'
	second, err := service.UploadImageRecord(context.Background(), ImageRecordUploadRequest{}, option)
	if err != nil || second == nil || second.Record == nil || calls != 4 || string(second.Record.Resource.Body["name"]) != `"captured"` || string(second.Record.bodyState.current["container_format"]) != `"bare"` || second.Metadata.Header.Get("X-Task-Proof") != "actual" || string(second.Metadata.Body) != `{"id":"created"}` {
		t.Fatal(second, err, calls)
	}
}

func TestImageRecordUploadCallerOpenedFilenameRemainsBorrowedAndNeverReopened(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "owned-upload-borrowed-*")
	th.AssertNoErr(t, err)
	defer file.Close()
	_, err = file.WriteString("prefixDATA")
	th.AssertNoErr(t, err)
	_, err = file.Seek(6, io.SeekStart)
	th.AssertNoErr(t, err)
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			body := taskCorePayload(t, req)
			th.AssertEquals(t, "/not-the-caller-file", taskCoreText(t, body["filename"]))
			if len(body) != 3 {
				t.Fatal(body)
			}
			return taskCoreJSON(req, 201, `{"id":"created"}`), nil
		}
		if calls != 2 || req.Method != http.MethodPut || req.Header.Get("X-OpenStack-Image-Size") != "10" || imageRecordStageRead(t, req) != "DATA" {
			t.Fatal(calls, req.Method, req.Header)
		}
		_ = req.Body.Close()
		return taskCoreJSON(req, 204, ""), nil
	})
	got, err := New(client).UploadImageRecord(context.Background(), ImageRecordUploadRequest{Data: file, Attributes: map[string]any{"filename": "/not-the-caller-file"}}, imageRecordUploadFormats()...)
	if err != nil || got == nil || got.Record == nil || got.Record.data != file || calls != 2 {
		t.Fatal(got, err, calls)
	}
	_, err = file.Seek(0, io.SeekStart)
	th.AssertNoErr(t, err)
	actual, err := io.ReadAll(file)
	th.AssertNoErr(t, err)
	th.AssertEquals(t, "prefixDATA", string(actual))
}
