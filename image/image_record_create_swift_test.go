package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	objectapi "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func imageCreateTaskServices(t *testing.T, transport taskCoreTransport) (*Service, *objectapi.Service, *int) {
	t.Helper()
	client := taskCoreClient(transport)
	swiftClient := &gophercloud.ServiceClient{ProviderClient: client.ProviderClient, Endpoint: "https://swift.example/v1/AUTH_project/", Type: "object-store"}
	swift := objectapi.New(swiftClient)
	resolutions := new(int)
	service := NewWithDependencies(client, Dependencies{
		CreatePolicy:  ImageCreatePolicy{UseTasks: json.RawMessage(`true`), CloudName: "test cloud"},
		ObjectStorage: func(ctx context.Context) (*objectapi.Service, error) { (*resolutions)++; return swift, nil },
	})
	return service, swift, resolutions
}

func imageCreateTaskOptions(options ...ImageRecordCreateOption) []ImageRecordCreateOption {
	return append([]ImageRecordCreateOption{WithImageRecordCreateAllowDuplicates(true)}, options...)
}

func imageCreateTaskRoute(req *http.Request) string {
	return req.Method + " " + req.URL.Host + req.URL.EscapedPath()
}

func TestImageRecordCreateTaskWholeNoWaitKeepsRawTaskAndAllSourceSuccessCodes(t *testing.T) {
	for _, code := range []int{200, 203, 299, 399} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var routes []string
			service, _, resolutions := imageCreateTaskServices(t, func(req *http.Request) (*http.Response, error) {
				routes = append(routes, imageCreateTaskRoute(req))
				if req.Header.Get("X-Auth-Token") != "first-token" {
					t.Fatal(req.Header)
				}
				switch len(routes) {
				case 1:
					if req.Method != "HEAD" || req.URL.Path != "/v1/AUTH_project/images" {
						t.Fatal(req.Method, req.URL)
					}
					return taskCoreJSON(req, code, "container receipt"), nil
				case 2:
					if req.Method != "PUT" || req.URL.Path != "/v1/AUTH_project/images/task image" || imageRecordStageRead(t, req) != "binary payload" {
						t.Fatal(req.Method, req.URL)
					}
					if req.Header.Get("Content-Type") != "application/octet-stream" || req.Header.Get("X-Delete-After") != "86400" || req.Header.Get("X-Object-Meta-X-Sdk-Autocreated") != "true" || req.Header.Get("X-Trace") != "captured" {
						t.Fatal(req.Header)
					}
					return taskCoreJSON(req, code, "object receipt"), nil
				case 3:
					if req.Method != "POST" || req.URL.Path != "/reverse/glance/v2/tasks" {
						t.Fatal(req.Method, req.URL)
					}
					imageRecordImportPayload(t, req, `{"type":"import","input":{"import_from":"images/task image","image_properties":{"name":"task image"}}}`)
					return taskCoreJSON(req, code, `{"status":null,"result":false,"id":[1],"extra":"passive"}`), nil
				default:
					t.Fatal("unexpected wait, image request or cleanup", routes)
				}
				return nil, nil
			})
			got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "task image", Data: ImageRecordCreateText("binary payload")}, imageCreateTaskOptions(WithImageRecordCreateSwiftHeader("X-Trace", "captured"), WithImageRecordCreateMD5(7), WithImageRecordCreateSHA256([]any{1}))...)
			th.AssertNoErr(t, err)
			if got == nil || got.Outcome != "task" || got.Record != nil || got.Task == nil || got.TaskWait != nil || got.Swift == nil || got.Swift.Cleanup != nil || got.Swift.ContainerCreated != nil || got.Swift.ContainerFetched != nil || got.Created.StatusCode != code || got.Swift.ContainerDiscovery.StatusCode != code || got.Swift.Object.Uploaded.Acknowledgement.StatusCode != code || len(routes) != 3 || *resolutions != 1 {
				t.Fatal(got, routes, *resolutions)
			}
			if string(got.Task.Resource.Body["id"]) != `[1]` || string(got.Task.Resource.Body["status"]) != `null` || string(got.Task.Resource.Body["result"]) != `false` || !bytes.Contains(got.Task.Envelope, []byte(`"extra":"passive"`)) {
				t.Fatal(got.Task)
			}
		})
	}
}

func TestImageRecordCreateTaskWholeContainerEnsureAndLateEmptyDataConflict(t *testing.T) {
	for _, present := range []bool{false, true} {
		t.Run(fmt.Sprint(present), func(t *testing.T) {
			var routes []string
			service, _, _ := imageCreateTaskServices(t, func(req *http.Request) (*http.Response, error) {
				routes = append(routes, imageCreateTaskRoute(req))
				if req.URL.Path != "/v1/AUTH_project/images" {
					t.Fatal(req.URL)
				}
				if len(routes) == 1 {
					return taskCoreJSON(req, 404, "missing"), nil
				}
				if len(routes) == 2 && req.Method == "PUT" {
					return taskCoreJSON(req, 203, "created"), nil
				}
				if len(routes) == 3 && req.Method == "HEAD" {
					return taskCoreJSON(req, 299, "refetched"), nil
				}
				t.Fatal(routes)
				return nil, nil
			})
			input := ImageRecordCreateRequest{Name: "missing local file", Filename: "/nonexistent/go-openstacksdk-create-task"}
			if present {
				input.Data = ImageRecordCreateBytes(nil)
			}
			got, err := service.CreateImageRecord(context.Background(), input, imageCreateTaskOptions()...)
			if got == nil || err == nil || got.Swift == nil || got.Swift.ContainerDiscovery.StatusCode != 404 || string(got.Swift.ContainerDiscovery.Body) != "missing" || got.Swift.ContainerCreated.StatusCode != 203 || got.Swift.ContainerFetched.StatusCode != 299 || got.Swift.Object != nil || got.Task != nil || got.Swift.Cleanup != nil || len(routes) != 3 {
				t.Fatal(got, err, routes)
			}
			if present && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordCreateTaskWholeAvailabilityIsLazyAndRawConfigIsBool(t *testing.T) {
	for _, raw := range []string{`false`, `null`, `"true"`, `0`, `[]`, `{}`} {
		t.Run(raw, func(t *testing.T) {
			calls := 0
			service, _, resolutions := imageCreateTaskServices(t, func(req *http.Request) (*http.Response, error) {
				calls++
				t.Fatal(req.Method, req.URL)
				return nil, nil
			})
			service.dependencies.CreatePolicy.RawObjectStoreEnabled = json.RawMessage(raw)
			got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "raw", Data: ImageRecordCreateBytes([]byte("data"))}, imageCreateTaskOptions()...)
			if err == nil || got == nil || got.Swift != nil || calls != 0 || *resolutions != 0 {
				t.Fatal(got, err, calls, *resolutions)
			}
		})
	}
	t.Run("metadata route ignores unavailable malformed task config", func(t *testing.T) {
		calls := 0
		service, _, resolutions := imageCreateTaskServices(t, func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Method != "POST" || req.URL.Path != "/reverse/glance/v2/images" {
				t.Fatal(req.Method, req.URL)
			}
			return taskCoreJSON(req, 201, `{}`), nil
		})
		service.dependencies.CreatePolicy.RawObjectStoreEnabled = json.RawMessage(`"invalid only in Task branch"`)
		got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "no such local file"}, imageCreateTaskOptions()...)
		th.AssertNoErr(t, err)
		if got == nil || got.Outcome != "metadata-only" || got.Task != nil || got.Swift != nil || *resolutions != 0 || calls != 1 {
			t.Fatal(got, *resolutions, calls)
		}
	})
	t.Run("explicit import conflict precedes resolver", func(t *testing.T) {
		service, _, resolutions := imageCreateTaskServices(t, func(req *http.Request) (*http.Response, error) { t.Fatal(req.URL); return nil, nil })
		got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "data", Data: ImageRecordCreateText("data")}, imageCreateTaskOptions(WithImageRecordCreateUseImport(true))...)
		if got == nil || err == nil || *resolutions != 0 || got.Swift != nil {
			t.Fatal(got, err, *resolutions)
		}
	})
}

func TestImageRecordCreateTaskWholeSuccessMergesUnconvertedPropertiesAndFinallyDeletesObject(t *testing.T) {
	var routes []string
	service, _, _ := imageCreateTaskServices(t, func(req *http.Request) (*http.Response, error) {
		routes = append(routes, imageCreateTaskRoute(req))
		switch len(routes) {
		case 1:
			return taskCoreJSON(req, 204, ""), nil
		case 2:
			if imageRecordStageRead(t, req) != "bytes" {
				t.Fatal("payload")
			}
			return taskCoreJSON(req, 201, ""), nil
		case 3:
			return taskCoreJSON(req, 201, `{"status":"SuCcEsS","result":{"image_id":"imported"}}`), nil
		case 4:
			if req.Method != "GET" || req.URL.Path != "/reverse/glance/v2/images/imported" {
				t.Fatal(req.Method, req.URL)
			}
			return taskCoreJSON(req, 200, `{"id":"imported","name":"existing","disk_format":"qcow2","container_format":"bare","existing_property":"keep","status":"active"}`), nil
		case 5:
			if req.Method != "PATCH" || req.URL.Path != "/reverse/glance/v2/images/imported" || req.Header.Get("Content-Type") != imageRecordPatchContentType {
				t.Fatal(req.Method, req.URL, req.Header)
			}
			body, err := io.ReadAll(req.Body)
			th.AssertNoErr(t, err)
			var patches []map[string]json.RawMessage
			th.AssertNoErr(t, json.Unmarshal(body, &patches))
			values := map[string]string{}
			for _, patch := range patches {
				var path string
				th.AssertNoErr(t, json.Unmarshal(patch["path"], &path))
				values[path] = string(patch["value"])
			}
			if values["/raw_prop"] != `{"n":900719925474099312345}` || values["/nullable"] != "null" || values["/tags"] != `["task-tag"]` || values["/visibility"] != `"public"` {
				t.Fatal(string(body), values)
			}
			for _, absent := range []string{"/disk_format", "/container_format", "/meta_only", "/existing_property", "/size", "/is_public"} {
				if _, present := values[absent]; present {
					t.Fatal("unexpected property conversion/meta/format replacement", absent, string(body))
				}
			}
			return taskCoreJSON(req, 203, `{}`), nil
		case 6:
			if req.Method != "HEAD" || req.URL.Path != "/v1/AUTH_project/images/input" {
				t.Fatal(req.Method, req.URL)
			}
			return taskCoreJSON(req, 204, ""), nil
		case 7:
			if req.Method != "DELETE" || req.URL.Path != "/v1/AUTH_project/images/input" {
				t.Fatal(req.Method, req.URL)
			}
			return taskCoreJSON(req, 204, "deleted"), nil
		default:
			t.Fatal(routes)
		}
		return nil, nil
	})
	input := ImageRecordCreateRequest{Name: "input", Data: ImageRecordCreateBytes([]byte("bytes")), Attributes: map[string]any{"raw_prop": json.RawMessage(`{"n":900719925474099312345}`), "nullable": nil, "is_public": true}}
	got, err := service.CreateImageRecord(context.Background(), input, imageCreateTaskOptions(WithImageRecordCreateWait(true), WithImageRecordCreateTimeout("ignored for seeded success"), WithImageRecordCreateTags([]string{"task-tag"}), WithImageRecordCreateMeta(map[string]any{"meta_only": "ignored"}), WithImageRecordCreateSize(-7))...)
	th.AssertNoErr(t, err)
	if got == nil || got.Outcome != "task-completed" || got.Record == nil || got.Updated != got.Record || got.TaskWait.Fetches != 0 || got.TaskWait.Recreations != 0 || got.TaskImageFetched == nil || got.TaskImageFetched.StatusCode != 200 || !bytes.Contains(got.TaskImageFetched.Body, []byte(`"existing_property":"keep"`)) || len(got.Warnings) != 1 || got.Swift.Cleanup.Deletion.StatusCode != 204 || len(routes) != 7 {
		t.Fatal(got, routes)
	}
	var props map[string]json.RawMessage
	th.AssertNoErr(t, json.Unmarshal(got.Record.Resource.Body["properties"], &props))
	// A valid sparse PATCH response resets properties to the Source empty map;
	// the request assertion above independently proves preservation on submission.
	if len(props) != 0 || string(got.Record.Resource.Body["disk_format"]) != `"qcow2"` || string(got.Record.Resource.Body["container_format"]) != `"bare"` {
		t.Fatal(got.Record.Resource.Body)
	}
}

func TestImageRecordCreateTaskWholeFailureDiagnosesOriginalTaskOnlyForResourceFailure(t *testing.T) {
	for _, mode := range []string{"task failure", "http failure", "zero timeout", "bad timeout", "bad image result", "image failure", "patch failure"} {
		t.Run(mode, func(t *testing.T) {
			var routes []string
			service, _, _ := imageCreateTaskServices(t, func(req *http.Request) (*http.Response, error) {
				routes = append(routes, imageCreateTaskRoute(req))
				if req.URL.Host == "swift.example" {
					if req.Method == "HEAD" {
						return taskCoreJSON(req, 204, ""), nil
					}
					if req.Method == "PUT" {
						_ = imageRecordStageRead(t, req)
						return taskCoreJSON(req, 201, ""), nil
					}
					if req.Method == "DELETE" {
						return taskCoreJSON(req, 204, "cleaned"), nil
					}
				}
				if req.Method == "POST" {
					if mode == "bad image result" {
						return taskCoreJSON(req, 201, `{"id":"initial","status":"success","result":{}}`), nil
					}
					if mode == "image failure" || mode == "patch failure" {
						return taskCoreJSON(req, 201, `{"id":"initial","status":"success","result":{"image_id":"image"}}`), nil
					}
					return taskCoreJSON(req, 201, `{"id":"initial","status":"pending"}`), nil
				}
				if req.URL.Path == "/reverse/glance/v2/tasks/initial" && req.Method == "GET" {
					if mode == "http failure" {
						return taskCoreJSON(req, 503, "Task GET rejected"), nil
					}
					if len(routes) == 4 {
						return taskCoreJSON(req, 200, `{"id":"diagnostic-current","status":"failure","message":"failure"}`), nil
					}
					t.Fatal("diagnostic must use mutated original raw ID", req.URL)
				}
				if req.URL.Path == "/reverse/glance/v2/tasks/diagnostic-current" && req.Method == "GET" {
					return taskCoreJSON(req, 200, `{"id":"diagnostic","status":"failure","message":"details"}`), nil
				}
				if req.URL.Path == "/reverse/glance/v2/images/image" && req.Method == "GET" {
					if mode == "image failure" {
						return taskCoreJSON(req, 500, "image rejected"), nil
					}
					return taskCoreJSON(req, 200, `{"id":"image"}`), nil
				}
				if req.Method == "PATCH" {
					return taskCoreJSON(req, 500, "patch rejected"), nil
				}
				t.Fatal("unexpected request", mode, routes)
				return nil, nil
			})
			options := imageCreateTaskOptions(WithImageRecordCreateWait(true))
			if mode == "zero timeout" {
				options = append(options, WithImageRecordCreateTimeout(0))
			}
			if mode == "bad timeout" {
				options = append(options, WithImageRecordCreateTimeout("3600"))
			}
			got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "input", Data: ImageRecordCreateText("data")}, options...)
			if got == nil || err == nil || got.Swift == nil || got.Swift.Cleanup == nil || got.Swift.Cleanup.Deletion == nil || got.Swift.Cleanup.Deletion.StatusCode != 204 {
				t.Fatal(got, err, routes)
			}
			var failed *ImageRecordCreateTaskFailureError
			if mode == "task failure" {
				if !errors.As(err, &failed) || got.TaskDiagnostic == nil || string(got.TaskDiagnostic.Resource.Body["id"]) != `"diagnostic"` || got.TaskDiagnosticResponse.StatusCode != 200 || len(routes) != 7 {
					t.Fatal(got, err, routes)
				}
			} else if errors.As(err, &failed) || got.TaskDiagnostic != nil || got.TaskDiagnosticResponse != nil {
				t.Fatal(got, err, routes)
			}
			if mode == "zero timeout" && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordCreateTaskWholeFailuresBeforeWaitLeaveObjectAndContainer(t *testing.T) {
	for _, phase := range []string{"container HEAD", "container PUT", "container refetch", "object PUT", "task POST", "accepted task close"} {
		t.Run(phase, func(t *testing.T) {
			var routes []string
			closeErr := errors.New("accepted task Close failed")
			service, _, _ := imageCreateTaskServices(t, func(req *http.Request) (*http.Response, error) {
				routes = append(routes, imageCreateTaskRoute(req))
				switch len(routes) {
				case 1:
					if phase == "container HEAD" {
						return taskCoreJSON(req, 500, "HEAD rejected"), nil
					}
					return taskCoreJSON(req, 404, "missing"), nil
				case 2:
					if phase == "container PUT" {
						return taskCoreJSON(req, 500, "PUT rejected"), nil
					}
					return taskCoreJSON(req, 201, "container created"), nil
				case 3:
					if phase == "container refetch" {
						return taskCoreJSON(req, 500, "refetch rejected"), nil
					}
					return taskCoreJSON(req, 204, ""), nil
				case 4:
					_ = imageRecordStageRead(t, req)
					if phase == "object PUT" {
						return taskCoreJSON(req, 500, "object rejected"), nil
					}
					return taskCoreJSON(req, 201, "object uploaded"), nil
				case 5:
					if phase == "task POST" {
						return taskCoreJSON(req, 500, "task rejected"), nil
					}
					return taskCoreHTTP(req, 201, &taskCoreBody{reader: strings.NewReader(`{"id":"task","status":"success"}`), closeErr: closeErr}), nil
				default:
					t.Fatal("cleanup started before wait", routes)
				}
				return nil, nil
			})
			got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "input", Data: ImageRecordCreateText("data")}, imageCreateTaskOptions(WithImageRecordCreateWait(true))...)
			if got == nil || err == nil || got.TaskWait != nil || got.Swift == nil || got.Swift.Cleanup != nil {
				t.Fatal(got, err, routes)
			}
			if phase == "accepted task close" && (!errors.Is(err, closeErr) || got.Created == nil || got.Created.StatusCode != 201) {
				t.Fatal(got, err, routes)
			}
		})
	}
}

func TestImageRecordCreateTaskWholeJoinsCleanupFailureAndStopsAfterSwiftSourceDrift(t *testing.T) {
	t.Run("cleanup error joins timeout", func(t *testing.T) {
		var routes []string
		service, _, _ := imageCreateTaskServices(t, func(req *http.Request) (*http.Response, error) {
			routes = append(routes, imageCreateTaskRoute(req))
			switch len(routes) {
			case 1:
				return taskCoreJSON(req, 204, ""), nil
			case 2:
				_ = imageRecordStageRead(t, req)
				return taskCoreJSON(req, 201, ""), nil
			case 3:
				return taskCoreJSON(req, 201, `{"id":"task","status":"pending"}`), nil
			case 4:
				return taskCoreJSON(req, 204, ""), nil
			case 5:
				return taskCoreJSON(req, 503, "cleanup rejected"), nil
			default:
				t.Fatal(routes)
			}
			return nil, nil
		})
		got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "input", Data: ImageRecordCreateText("data")}, imageCreateTaskOptions(WithImageRecordCreateWait(true), WithImageRecordCreateTimeout(0))...)
		var native gophercloud.ErrUnexpectedResponseCode
		if got == nil || !errors.Is(err, context.DeadlineExceeded) || !errors.As(err, &native) || native.Actual != 503 || got.Swift.Cleanup == nil || got.Swift.Cleanup.Discovery.StatusCode != 204 || got.Swift.Cleanup.Deletion != nil || len(routes) != 5 {
			t.Fatal(got, err, routes)
		}
	})
	for _, drift := range []string{"endpoint", "provider", "object API", "container API", "resource base", "type", "microversion"} {
		t.Run(drift, func(t *testing.T) {
			calls := 0
			var swift *objectapi.Service
			service, bound, _ := imageCreateTaskServices(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if calls != 1 {
					t.Fatal("drift allowed later HTTP", req.Method, req.URL)
				}
				switch drift {
				case "endpoint":
					swift.RawClient().Endpoint += "changed/"
				case "provider":
					swift.RawClient().ProviderClient = &gophercloud.ProviderClient{}
				case "object API":
					swift.Objects = nil
				case "container API":
					swift.Containers = nil
				case "resource base":
					swift.RawClient().ResourceBase = "https://swift.example/changed/"
				case "type":
					swift.RawClient().Type = "image"
				case "microversion":
					swift.RawClient().Microversion = "future"
				}
				return taskCoreJSON(req, 204, "observed"), nil
			})
			swift = bound
			got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "input", Data: ImageRecordCreateText("data")}, imageCreateTaskOptions()...)
			if got == nil || !errors.Is(err, resource.ErrInvalidOption) || got.Swift == nil || got.Swift.ContainerDiscovery == nil || got.Swift.ContainerDiscovery.StatusCode != 204 || got.Swift.Object != nil || calls != 1 {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordCreateTaskWholeSLOCleanupUsesExistingBulkDeleteEvidence(t *testing.T) {
	var routes []string
	service, _, _ := imageCreateTaskServices(t, func(req *http.Request) (*http.Response, error) {
		routes = append(routes, imageCreateTaskRoute(req))
		switch len(routes) {
		case 1:
			return taskCoreJSON(req, 204, ""), nil
		case 2:
			_ = imageRecordStageRead(t, req)
			return taskCoreJSON(req, 201, ""), nil
		case 3:
			return taskCoreJSON(req, 201, `{"status":"pending","id":"task"}`), nil
		case 4:
			response := taskCoreJSON(req, 204, "")
			response.Header.Set("X-Static-Large-Object", "true")
			return response, nil
		case 5:
			if req.Method != "DELETE" || req.URL.Query().Get("multipart-manifest") != "delete" {
				t.Fatal(req.Method, req.URL)
			}
			return taskCoreJSON(req, 200, `{"Response Status":"200 OK","Response Body":"","Number Deleted":3,"Number Not Found":0,"Errors":[]}`), nil
		default:
			t.Fatal(routes)
		}
		return nil, nil
	})
	got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "input", Data: ImageRecordCreateText("data")}, imageCreateTaskOptions(WithImageRecordCreateWait(true), WithImageRecordCreateTimeout(0))...)
	if got == nil || !errors.Is(err, context.DeadlineExceeded) || got.Swift.Cleanup == nil || got.Swift.Cleanup.Bulk == nil || got.Swift.Cleanup.Bulk.NumberDeleted != 3 || got.Swift.Cleanup.StaticLargeObject == nil || !*got.Swift.Cleanup.StaticLargeObject || len(routes) != 5 {
		t.Fatal(got, err, routes)
	}
	want := []string{"HEAD swift.example/v1/AUTH_project/images", "PUT swift.example/v1/AUTH_project/images/input", "POST glance.example/reverse/glance/v2/tasks", "HEAD swift.example/v1/AUTH_project/images/input", "DELETE swift.example/v1/AUTH_project/images/input"}
	if !reflect.DeepEqual(routes, want) {
		t.Fatal(routes)
	}
}

func TestImageRecordCreateTaskWholeFilenameReusesSwiftCapabilityStaleAndArbitraryHashProfile(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "task.img")
	th.AssertNoErr(t, os.WriteFile(filename, []byte("file data"), 0600))
	for _, stale := range []bool{true, false} {
		t.Run(fmt.Sprint(stale), func(t *testing.T) {
			var routes []string
			service, _, _ := imageCreateTaskServices(t, func(req *http.Request) (*http.Response, error) {
				routes = append(routes, imageCreateTaskRoute(req))
				if req.Method == "HEAD" && req.URL.Path == "/v1/AUTH_project/images" {
					return taskCoreJSON(req, 204, ""), nil
				}
				if req.Method == "GET" && req.URL.Path == "/info" {
					return taskCoreJSON(req, 200, `{ "swift": { "max_file_size": 1024 }, "slo": { "min_segment_size": 1 } }`), nil
				}
				if req.Method == "HEAD" && req.URL.Path == "/v1/AUTH_project/images/input" {
					if stale {
						return taskCoreJSON(req, 404, "missing"), nil
					}
					response := taskCoreJSON(req, 204, "")
					response.Header.Set("X-Object-Meta-X-Sdk-Md5", "arbitrary MD5")
					response.Header.Set("X-Object-Meta-X-Sdk-Sha256", "arbitrary SHA256")
					return response, nil
				}
				if req.Method == "PUT" && req.URL.Path == "/v1/AUTH_project/images/input" {
					if !stale || imageRecordStageRead(t, req) != "file data" || req.Header.Get("X-Object-Meta-X-Sdk-Md5") != "arbitrary MD5" || req.Header.Get("X-Object-Meta-X-Sdk-Sha256") != "arbitrary SHA256" || req.Header.Get("X-Delete-After") != "86400" || req.Header.Get("Content-Type") != "application/octet-stream" {
						t.Fatal(req.Method, req.URL, req.Header)
					}
					return taskCoreJSON(req, 299, "uploaded"), nil
				}
				if req.Method == "POST" && req.URL.Path == "/reverse/glance/v2/tasks" {
					return taskCoreJSON(req, 201, `{}`), nil
				}
				t.Fatal("unexpected filename workflow phase", routes)
				return nil, nil
			})
			got, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "input", Filename: filename}, imageCreateTaskOptions(WithImageRecordCreateMD5("arbitrary MD5"), WithImageRecordCreateSHA256("arbitrary SHA256"))...)
			th.AssertNoErr(t, err)
			if got == nil || got.Outcome != "task" || got.Swift == nil || got.Swift.Object == nil || got.Swift.Object.Created == nil || got.Swift.Object.Uploaded != nil || got.Swift.Object.Created.Skipped == stale || got.Swift.Object.Created.Capabilities == nil || got.Swift.Object.Created.Discovery == nil || got.Task == nil || got.Swift.Cleanup != nil {
				t.Fatal(got, routes)
			}
			if stale && (got.Swift.Object.Created.Ordinary == nil || got.Swift.Object.Created.Ordinary.Acknowledgement.StatusCode != 299 || len(routes) != 5) {
				t.Fatal(got, routes)
			}
			if !stale && (got.Swift.Object.Created.Ordinary != nil || len(routes) != 4) {
				t.Fatal(got, routes)
			}
		})
	}
}
