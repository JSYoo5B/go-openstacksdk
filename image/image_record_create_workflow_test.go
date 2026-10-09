package image

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	objectapi "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

// Reuse the shared Gophercloud-backed transport fixtures. These tests exercise
// the whole modern entry point rather than replacing it with metadata creation.
func TestImageRecordCreateWorkflowFalseyDataKeepsMetadataOnlyBranch(t *testing.T) {
	for _, test := range []struct {
		name string
		data ImageRecordCreateData
	}{
		{"None", ImageRecordCreateData{}},
		{"nil bytes are present but falsey", ImageRecordCreateBytes(nil)},
		{"empty bytes", ImageRecordCreateBytes([]byte{})},
		{"empty text", ImageRecordCreateText("")},
		{"nil reader is None", ImageRecordCreateReader(nil)},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls, swift := 0, 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls != 1 || req.Method != http.MethodPost || req.URL.EscapedPath() != "/reverse/glance/v2/images" {
					t.Fatal("metadata-only dispatched another phase", calls, req.Method, req.URL)
				}
				imageRecordImportPayload(t, req, `{"name":"metadata","disk_format":"qcow2","container_format":"bare","owner_specified.openstack.md5":"","owner_specified.openstack.sha256":"","owner_specified.openstack.object":"images/metadata"}`)
				// No required ID, status, formats or binary response is imposed here.
				return taskCoreJSON(req, 203, `{}`), nil
			})
			service := NewWithDependencies(client, Dependencies{CreatePolicy: ImageCreatePolicy{UseTasks: json.RawMessage(`true`), RawObjectStoreEnabled: json.RawMessage(`"invalid only when Task is selected"`)}, ObjectStorage: func(context.Context) (*objectapi.Service, error) {
				swift++
				t.Fatal("metadata-only constructed Swift")
				return nil, nil
			}})
			// URI/store/use_import alone do not select upload. Store conflicts and
			// Task availability/timeout remain unobserved in this Source branch.
			got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "metadata", Data: test.data},
				WithImageRecordCreateAllowDuplicates(true), WithImageRecordCreateWait(true), WithImageRecordCreateTimeout("not a timeout"),
				WithImageRecordCreateValidateChecksum(true), WithImageRecordCreateUseImport(true),
				WithImageRecordCreateImportOptions(WithImageRecordImportURI("https://source.example/file"), WithImageRecordImportAllStores(true), WithImageRecordImportStores(ImageRecordImportStore{ID: "store"})))
			if err != nil || got == nil || got.Record == nil || got.Created == nil || got.Created.StatusCode != 203 || got.Outcome != "metadata-only" || got.Reused || got.Uploaded != nil || got.Staged != nil || got.Imported != nil || got.Task != nil || got.TaskWait != nil || got.Swift != nil || got.ChecksumFetched != nil || calls != 1 || swift != 0 {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordCreateWorkflowDefaultsUsePythonTruthiness(t *testing.T) {
	for _, test := range []struct {
		name  string
		value any
	}{
		{"null", nil}, {"false", false}, {"zero", 0}, {"empty string", ""}, {"empty array", []any{}}, {"empty map", map[string]any{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodPost {
					t.Fatal(req.Method)
				}
				imageRecordImportPayload(t, req, `{"name":"selected","disk_format":"raw","container_format":"bare","owner_specified.openstack.md5":"","owner_specified.openstack.sha256":"","owner_specified.openstack.object":"/selected"}`)
				return taskCoreJSON(req, 201, `{"id":null,"status":"future server state"}`), nil
			})
			service := NewWithDependencies(client, Dependencies{CreatePolicy: ImageCreatePolicy{ImageFormat: json.RawMessage(`"raw"`)}})
			got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "selected"}, WithImageRecordCreateAllowDuplicates(true), WithImageRecordCreateContainer(""), WithImageRecordCreateDiskFormat(test.value), WithImageRecordCreateContainerFormat(test.value), WithImageRecordCreateTags(test.value))
			if err != nil || got == nil || got.Record == nil || calls != 1 || got.Outcome != "metadata-only" {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordCreateWorkflowMetadataConversionVendorPropertiesAndMetaOrder(t *testing.T) {
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodPost || calls != 1 {
			t.Fatal(calls, req.Method)
		}
		imageRecordImportPayload(t, req, `{"name":"formal","disk_format":"raw","container_format":"bare","tags":["raw",2],"min_disk":4,"min_ram":0,"virtual_size":-3,"vendor":"legacy properties","flag":"True","list":"[1, 'x']","nil_value":null,"raw_number":900719925474099312345,"owner_specified.openstack.md5":"","owner_specified.openstack.sha256":"","owner_specified.openstack.object":"images/formal"}`)
		return taskCoreJSON(req, 299, `{"id":"server","status":"active"}`), nil
	})
	service := NewWithDependencies(client, Dependencies{CreatePolicy: ImageCreatePolicy{DisableVendorAgent: map[string]any{"vendor": "cloud", "min_disk": "4", "flag": true}}})
	attrs := map[string]any{"name": "kwargs", "vendor": "caller", "min_disk": "2", "min_ram": false, "virtual_size": -3.9, "list": []any{1, "x"}, "nil_value": nil, "properties": map[string]any{"vendor": "legacy properties"}}
	got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "formal", Attributes: attrs}, WithImageRecordCreateAllowDuplicates(true), WithImageRecordCreateTags([]any{"ordinary"}), WithImageRecordCreateMeta(map[string]any{"name": "meta name loses", "disk_format": "raw", "tags": []any{"raw", 2}, "raw_number": json.RawMessage(`900719925474099312345`)}))
	if err != nil || got == nil || got.Record == nil || got.Outcome != "metadata-only" || calls != 1 {
		t.Fatal(got, err, calls)
	}
	if attrs["vendor"] != "caller" || attrs["name"] != "kwargs" || attrs["properties"].(map[string]any)["vendor"] != "legacy properties" {
		t.Fatal("input changed", attrs)
	}
}

func TestImageRecordCreateWorkflowIntegerConversionPrecedesMetaOverride(t *testing.T) {
	for _, key := range []string{"min_disk", "min_ram", "size", "virtual_size"} {
		t.Run(key, func(t *testing.T) {
			calls := 0
			service := New(taskCoreClient(func(*http.Request) (*http.Response, error) { calls++; t.Fatal("invalid integer POST"); return nil, nil }))
			got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "selected", Attributes: map[string]any{key: nil}}, WithImageRecordCreateAllowDuplicates(true), WithImageRecordCreateMeta(map[string]any{key: 4}))
			if !errors.Is(err, resource.ErrInvalidOption) || got != nil && got.Created != nil || calls != 0 {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordCreateWorkflowMetadataOnlyAcceptsSparseAndOpaqueSuccess(t *testing.T) {
	for _, test := range []struct {
		name, body string
		code       int
	}{
		{"empty object", `{}`, 200}, {"explicit null ID", `{"id":null}`, 203}, {"opaque", `opaque accepted`, 299}, {"empty", ``, 399},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls != 1 || req.Method != http.MethodPost {
					t.Fatal("extra metadata-only phase", calls, req.Method)
				}
				return taskCoreJSON(req, test.code, test.body), nil
			}))
			got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "selected"}, WithImageRecordCreateAllowDuplicates(true), WithImageRecordCreateWait(true), WithImageRecordCreateTimeout(-1))
			if err != nil || got == nil || got.Record == nil || got.Created == nil || got.Created.StatusCode != test.code || string(got.Created.Body) != test.body || calls != 1 || got.Uploaded != nil || got.TaskWait != nil {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordCreateWorkflowDuplicateHashesRespectModernPresence(t *testing.T) {
	for _, test := range []struct {
		name, hashes string
		md5, sha     any
		reuse        bool
	}{
		{"modern MD5", `"owner_specified.openstack.md5":"same","owner_specified.shade.md5":"other"`, "same", nil, true},
		{"legacy MD5", `"owner_specified.shade.md5":"same"`, "same", nil, true},
		{"modern null blocks legacy", `"owner_specified.openstack.md5":null,"owner_specified.shade.md5":"same"`, "same", nil, false},
		{"modern empty blocks legacy", `"owner_specified.openstack.md5":"","owner_specified.shade.md5":"same"`, "same", nil, false},
		{"legacy SHA", `"owner_specified.shade.sha256":"sha"`, nil, "sha", true},
		{"both supplied require both", `"owner_specified.openstack.md5":"same","owner_specified.openstack.sha256":"different"`, "same", "sha", false},
		{"Python numeric equality", `"owner_specified.openstack.md5":1`, true, nil, true},
		{"falsey expected hashes never reuse", `"owner_specified.openstack.md5":"same"`, "", nil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images/selected" {
						t.Fatal(req.Method, req.URL)
					}
					return taskCoreJSON(req, 203, `{"id":"existing","name":"selected",`+test.hashes+`}`), nil
				}
				if calls != 2 || test.reuse || req.Method != http.MethodPost {
					t.Fatal(calls, req.Method)
				}
				return taskCoreJSON(req, 201, `{"id":"new"}`), nil
			}))
			got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "selected"}, WithImageRecordCreateMD5(test.md5), WithImageRecordCreateSHA256(test.sha))
			if err != nil || got == nil || got.Record == nil || got.Reused != test.reuse {
				t.Fatal(got, err, calls)
			}
			if test.reuse {
				if calls != 1 || got.Outcome != "reused" || got.Created != nil || imageRecordText(got.Record, "id") != "existing" {
					t.Fatal(got, calls)
				}
			} else if calls != 2 || got.Created == nil || imageRecordText(got.Record, "id") != "new" {
				t.Fatal(got, calls)
			}
		})
	}
}

func TestImageRecordCreateWorkflowReuseStopsBeforeVendorPropertiesAndUploadConflicts(t *testing.T) {
	calls := 0
	service := NewWithDependencies(taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls != 1 || req.Method != http.MethodGet {
			t.Fatal(calls, req.Method)
		}
		return taskCoreJSON(req, 200, `{"id":"existing","owner_specified.openstack.md5":"same"}`), nil
	}), Dependencies{CreatePolicy: ImageCreatePolicy{UseTasks: json.RawMessage(`true`), RawVendorAgent: json.RawMessage(`null`), RawObjectStoreEnabled: json.RawMessage(`"invalid"`)}})
	got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "selected", Data: ImageRecordCreateBytes([]byte("data")), Attributes: map[string]any{"properties": nil, "min_ram": nil}}, WithImageRecordCreateMD5("same"), WithImageRecordCreateUseImport(true), WithImageRecordCreateMeta(map[string]any{"size": "invalid integer"}), WithImageRecordCreateImportOptions(WithImageRecordImportAllStores(true), WithImageRecordImportStores(ImageRecordImportStore{ID: "store"})))
	if err != nil || got == nil || !got.Reused || got.Created != nil || got.Uploaded != nil || calls != 1 {
		t.Fatal(got, err, calls)
	}
}

func TestImageRecordCreateWorkflowDiscoveryTraversesVisibleThenHidden(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls := 0
			selected := "literal/空 白%?#"
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet {
					t.Fatal("discovery uploaded", req.Method)
				}
				switch calls {
				case 1:
					if req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(selected) {
						t.Fatal(req.URL)
					}
					return taskCoreJSON(req, code, `{"message":"not directly visible"}`), nil
				case 2:
					if req.URL.Query().Get("name") != selected || req.URL.Query().Has("os_hidden") {
						t.Fatal(req.URL)
					}
					return taskCoreJSON(req, 200, `{"images":[],"next":null}`), nil
				case 3:
					if req.URL.Query().Has("name") || req.URL.Query().Get("os_hidden") != "True" {
						t.Fatal(req.URL)
					}
					name, _ := json.Marshal(selected)
					return taskCoreJSON(req, 206, `{"images":[{"id":"hidden","name":`+string(name)+`,"owner_specified.openstack.md5":"same"}],"next":null}`), nil
				default:
					t.Fatal("extra discovery", calls)
					return nil, nil
				}
			}))
			got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: selected}, WithImageRecordCreateMD5("same"))
			if err != nil || got == nil || !got.Reused || got.Record == nil || imageRecordText(got.Record, "id") != "hidden" || calls != 3 {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordCreateWorkflowDuplicateDiscoveryFailureDoesNotCreate(t *testing.T) {
	calls := 0
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodGet {
			t.Fatal("POST after ambiguous discovery", req.Method)
		}
		if calls == 1 {
			return taskCoreJSON(req, 404, `{}`), nil
		}
		if calls != 2 {
			t.Fatal("hidden search after duplicate", calls)
		}
		return taskCoreJSON(req, 200, `{"images":[{"id":"first","name":"selected"},{"id":"second","name":"selected"}]}`), nil
	}))
	got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "selected"}, WithImageRecordCreateMD5("same"))
	if !errors.Is(err, resource.ErrAmbiguous) || got != nil && got.Created != nil || calls != 2 {
		t.Fatal(got, err, calls)
	}
}

func TestImageRecordCreateWorkflowVendorFlagAndPairListFollowSourceUpdate(t *testing.T) {
	for _, test := range []struct {
		name, vendor, expected string
		disabled               bool
		invalid                bool
	}{
		{"mapping", `{"flag":true}`, `"True"`, false, false},
		{"pair list last wins", `[["flag",false],["flag",7]]`, `"7"`, false, false},
		{"null skipped", `null`, `"caller"`, true, false},
		{"null selected", `null`, "", false, true},
		{"bad pair", `[["flag"]]`, "", false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			service := NewWithDependencies(taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if test.invalid {
					t.Fatal("invalid vendor dispatched")
				}
				body := taskCorePayload(t, req)
				th.CheckEquals(t, test.expected, string(body["flag"]))
				return taskCoreJSON(req, 201, `{}`), nil
			}), Dependencies{CreatePolicy: ImageCreatePolicy{RawVendorAgent: json.RawMessage(test.vendor)}})
			got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "selected", Attributes: map[string]any{"flag": "caller"}}, WithImageRecordCreateAllowDuplicates(true), WithImageRecordCreateDisableVendorAgent(!test.disabled))
			if test.invalid {
				if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || got != nil && got.Created != nil {
					t.Fatal(got, err, calls)
				}
			} else if err != nil || got == nil || calls != 1 {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordCreateWorkflowLegacyPropertiesShapeFailsAfterDiscovery(t *testing.T) {
	for _, value := range []any{nil, []any{}, false, "literal"} {
		t.Run(fmt.Sprintf("%T/%v", value, value), func(t *testing.T) {
			calls := 0
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls != 1 || req.Method != http.MethodGet {
					t.Fatal("legacy shape error reached POST", calls, req.Method)
				}
				return taskCoreJSON(req, 200, `{"id":"existing"}`), nil
			}))
			got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "selected", Attributes: map[string]any{"properties": value}})
			if !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || got != nil && got.Created != nil {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordCreateWorkflowIsPublicWarningBelongsToSelectedUploadOnly(t *testing.T) {
	for _, test := range []struct {
		name       string
		raw        any
		visibility string
	}{
		{"null", nil, "private"}, {"false", false, "private"}, {"truthy string false", "false", "public"}, {"truthy array", []any{1}, "public"},
	} {
		for _, upload := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/upload=%t", test.name, upload), func(t *testing.T) {
				calls := 0
				service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					if calls == 1 {
						if req.Method != http.MethodPost {
							t.Fatal(req.Method)
						}
						body := taskCorePayload(t, req)
						if upload {
							if _, ok := body["is_public"]; ok {
								t.Fatal("deprecated property retained", body)
							}
							th.CheckEquals(t, test.visibility, taskCoreText(t, body["visibility"]))
						} else {
							if _, ok := body["visibility"]; ok {
								t.Fatal("metadata branch inferred visibility", body)
							}
							if _, ok := body["is_public"]; !ok {
								t.Fatal("metadata branch discarded ordinary property", body)
							}
						}
						return taskCoreJSON(req, 201, `{"id":"server","status":"active"}`), nil
					}
					if !upload || calls != 2 || req.Method != http.MethodPut || req.URL.EscapedPath() != "/reverse/glance/v2/images/server/file" || imageRecordStageRead(t, req) != "payload" {
						t.Fatal(calls, req.Method, req.URL)
					}
					return taskCoreJSON(req, 204, ""), nil
				}))
				input := ImageRecordCreateRequest{Name: "selected", Attributes: map[string]any{"is_public": test.raw}}
				if upload {
					input.Data = ImageRecordCreateBytes([]byte("payload"))
				}
				got, err := service.CreateImageRecord(context.Background(), input, WithImageRecordCreateAllowDuplicates(true))
				if err != nil || got == nil || upload && (calls != 2 || len(got.Warnings) != 1) || !upload && (calls != 1 || len(got.Warnings) != 0) {
					t.Fatal(got, err, calls)
				}
			})
		}
	}
}

func TestImageRecordCreatePrefaceFilenameInferencePreservesSourceBugAndDotfiles(t *testing.T) {
	dir := t.TempDir()
	for _, base := range []string{"disk.qcow2", ".disk", "..disk", ".disk.raw", "disk."} {
		t.Run(base, func(t *testing.T) {
			filename := filepath.Join(dir, base)
			th.AssertNoErr(t, os.WriteFile(filename, []byte("payload"), 0600))
			calls := 0
			service := NewWithDependencies(taskCoreClient(func(*http.Request) (*http.Response, error) { calls++; t.Fatal("preface dispatched"); return nil, nil }), Dependencies{CreatePolicy: ImageCreatePolicy{ImageFormat: json.RawMessage(`null`)}})
			p, err := service.prepareImageRecordCreateWorkflow(context.Background(), ImageRecordCreateRequest{Name: filename}, nil)
			th.AssertNoErr(t, err)
			th.AssertNoErr(t, p.preface())
			want := base
			switch base {
			case "disk.qcow2":
				want = "disk"
			case ".disk.raw":
				want = ".disk"
			case "disk.":
				want = "disk"
			}
			if p.input.Name != want || p.input.Filename != filename || string(p.policy.DiskFormat) != "null" || calls != 0 {
				t.Fatal(p.input, p.policy, calls)
			}
		})
	}
	name := filepath.Join(dir, "absent-original")
	th.AssertNoErr(t, os.WriteFile(name+".qcow2", []byte("payload"), 0600))
	service := New(taskCoreClient(func(*http.Request) (*http.Response, error) { t.Fatal("preface dispatched"); return nil, nil }))
	p, err := service.prepareImageRecordCreateWorkflow(context.Background(), ImageRecordCreateRequest{Name: name}, nil)
	th.AssertNoErr(t, err)
	th.AssertNoErr(t, p.preface())
	if p.input.Filename != "" || p.input.Name != name {
		t.Fatal("corrected pinned Source typo silently", p.input)
	}
}

func TestImageRecordCreatePrefaceConfiguredFormatTypeIsCheckedOnlyDuringInference(t *testing.T) {
	for _, mode := range []string{"absent", "directory", "truthy bytes bypass", "explicit filename bypass"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			input := ImageRecordCreateRequest{Name: filepath.Join(dir, "absent")}
			switch mode {
			case "directory":
				input.Name = dir
			case "truthy bytes bypass":
				input.Data = ImageRecordCreateBytes([]byte("payload"))
			case "explicit filename bypass":
				input.Filename = filepath.Join(dir, "unused explicit file")
			}
			service := NewWithDependencies(taskCoreClient(func(*http.Request) (*http.Response, error) { t.Fatal("preface dispatched"); return nil, nil }), Dependencies{CreatePolicy: ImageCreatePolicy{ImageFormat: json.RawMessage(`false`)}})
			p, err := service.prepareImageRecordCreateWorkflow(context.Background(), input, []ImageRecordCreateOption{WithImageRecordCreateDiskFormat("formal format cannot replace cloud helper format")})
			th.AssertNoErr(t, err)
			err = p.preface()
			if mode == "absent" || mode == "directory" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal("cloud helper format was skipped", err)
				}
			} else {
				th.AssertNoErr(t, err)
			}
		})
	}
}

func TestImageRecordCreatePrefaceTruthyFilenameAndDataFailBeforeHashOrDiscovery(t *testing.T) {
	for _, data := range []ImageRecordCreateData{ImageRecordCreateBytes([]byte("x")), ImageRecordCreateText("x"), ImageRecordCreateReader(strings.NewReader(""))} {
		calls := 0
		service := New(taskCoreClient(func(*http.Request) (*http.Response, error) { calls++; t.Fatal("conflict dispatched"); return nil, nil }))
		got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "selected", Filename: "/missing-file-must-not-open", Data: data}, WithImageRecordCreateValidateChecksum(true))
		if !errors.Is(err, resource.ErrInvalidOption) || !strings.Contains(err.Error(), "mutually exclusive") || calls != 0 || got != nil && got.Created != nil {
			t.Fatal(got, err, calls)
		}
	}
	for _, data := range []ImageRecordCreateData{ImageRecordCreateBytes(nil), ImageRecordCreateText("")} {
		service := New(taskCoreClient(func(*http.Request) (*http.Response, error) { t.Fatal("preface dispatched"); return nil, nil }))
		p, err := service.prepareImageRecordCreateWorkflow(context.Background(), ImageRecordCreateRequest{Name: "selected", Filename: "/not-opened-in-preface", Data: data}, nil)
		th.AssertNoErr(t, err)
		th.AssertNoErr(t, p.preface())
		if !p.input.Data.present() || p.input.Data.truthy() {
			t.Fatal("falsey presence lost")
		}
	}
}

func TestImageRecordCreatePrefaceChecksumValidationComputesBothOrNeither(t *testing.T) {
	payload := []byte("payload\x00\xff")
	md5Sum, shaSum := md5.Sum(payload), sha256.Sum256(payload)
	wantMD5, wantSHA := hex.EncodeToString(md5Sum[:]), hex.EncodeToString(shaSum[:])
	filename := filepath.Join(t.TempDir(), "disk")
	th.AssertNoErr(t, os.WriteFile(filename, payload, 0600))
	for _, mode := range []string{"bytes", "file", "supplied MD5 prevents file read", "supplied SHA prevents file read", "empty bytes", "disabled"} {
		t.Run(mode, func(t *testing.T) {
			input := ImageRecordCreateRequest{Name: "selected"}
			options := []ImageRecordCreateOption{WithImageRecordCreateValidateChecksum(true)}
			switch mode {
			case "bytes":
				input.Data = ImageRecordCreateBytes(payload)
			case "file":
				input.Filename = filename
			case "supplied MD5 prevents file read":
				input.Filename = filename + ".missing"
				options = append(options, WithImageRecordCreateMD5("supplied"))
			case "supplied SHA prevents file read":
				input.Filename = filename + ".missing"
				options = append(options, WithImageRecordCreateSHA256("supplied"))
			case "empty bytes":
				input.Data = ImageRecordCreateBytes(nil)
			case "disabled":
				input.Data = ImageRecordCreateBytes(payload)
				options = []ImageRecordCreateOption{WithImageRecordCreateValidateChecksum(false)}
			}
			service := New(taskCoreClient(func(*http.Request) (*http.Response, error) { t.Fatal("preface dispatched"); return nil, nil }))
			p, err := service.prepareImageRecordCreateWorkflow(context.Background(), input, options)
			th.AssertNoErr(t, err)
			th.AssertNoErr(t, p.preface())
			switch mode {
			case "bytes", "file":
				th.CheckEquals(t, wantMD5, taskCoreText(t, p.policy.MD5))
				th.CheckEquals(t, wantSHA, taskCoreText(t, p.policy.SHA256))
			case "supplied MD5 prevents file read":
				if taskCoreText(t, p.policy.MD5) != "supplied" || p.policy.SHA256 != nil {
					t.Fatal(p.policy)
				}
			case "supplied SHA prevents file read":
				if taskCoreText(t, p.policy.SHA256) != "supplied" || p.policy.MD5 != nil {
					t.Fatal(p.policy)
				}
			default:
				if p.policy.MD5 != nil || p.policy.SHA256 != nil {
					t.Fatal("unexpected hashing", p.policy)
				}
			}
		})
	}
}

func TestImageRecordCreatePrefaceChecksumRejectsTextAndBorrowedReaderWithoutReading(t *testing.T) {
	for _, mode := range []string{"text", "reader"} {
		t.Run(mode, func(t *testing.T) {
			reader := &taskCoreReader{body: "payload"}
			data := ImageRecordCreateReader(reader)
			if mode == "text" {
				data = ImageRecordCreateText("payload")
			}
			calls := 0
			service := New(taskCoreClient(func(*http.Request) (*http.Response, error) {
				calls++
				t.Fatal("checksum rejection dispatched")
				return nil, nil
			}))
			got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "selected", Data: data}, WithImageRecordCreateValidateChecksum(true), WithImageRecordCreateMD5("already supplied"))
			if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || reader.read || got != nil && got.Created != nil {
				t.Fatal(got, err, calls, reader.read)
			}
		})
	}
}

func TestImageRecordCreateWorkflowCallbacksAndSourceAreCapturedOnce(t *testing.T) {
	for _, mode := range []string{"stable", "location changes source", "option changes source", "option cancels"} {
		t.Run(mode, func(t *testing.T) {
			calls, locations, applied := 0, 0, 0
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			marker := errors.New("create callback cancelled")
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, 201, `{}`), nil })
			service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
				locations++
				if mode == "location changes source" {
					client.Endpoint = "https://foreign.example/"
				}
				return resource.CloudLocation{}, nil
			}})
			got, err := service.CreateImageRecord(ctx, ImageRecordCreateRequest{Name: "selected"}, WithImageRecordCreateAllowDuplicates(true), func(c *ImageRecordCreateOpts) error {
				applied++
				if mode == "option changes source" {
					client.ResourceBase = "https://foreign.example/"
				}
				if mode == "option cancels" {
					cancel(marker)
				}
				return nil
			})
			if mode == "stable" {
				if err != nil || got == nil || calls != 1 || locations != 1 || applied != 1 {
					t.Fatal(got, err, calls, locations, applied)
				}
			} else {
				if err == nil || calls != 0 || locations != 1 || mode == "location changes source" && applied != 0 || mode != "location changes source" && applied != 1 {
					t.Fatal(got, err, calls, locations, applied)
				}
				if mode == "option cancels" && !errors.Is(err, marker) {
					t.Fatal("cancel cause lost", err)
				}
			}
		})
	}
}

func TestImageRecordCreateWorkflowNilAndCancelledContextsDoNotInvokeCallbacks(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	marker := errors.New("already cancelled")
	cancel(marker)
	for _, selected := range []context.Context{nil, ctx} {
		calls, locations, applied := 0, 0, 0
		service := NewWithDependencies(taskCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil }), Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return resource.CloudLocation{}, nil }})
		got, err := service.CreateImageRecord(selected, ImageRecordCreateRequest{Name: "selected"}, func(*ImageRecordCreateOpts) error { applied++; return nil })
		if err == nil || calls != 0 || locations != 0 || applied != 0 || got != nil {
			t.Fatal(got, err, calls, locations, applied)
		}
		if selected != nil && !errors.Is(err, marker) {
			t.Fatal(err)
		}
	}
}
