package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

// The tests use the existing captured-client transport and public Gophercloud
// assertion helpers. Only the non-Task branch is invoked directly here; whole
// route selection and duplicate discovery are tested by CreateImageRecord.
func imageCreateUploadRun(t *testing.T, service *Service, input ImageRecordCreateRequest, options ...ImageRecordCreateOption) (*ImageRecordCreateResult, error) {
	t.Helper()
	p, err := service.prepareImageRecordCreateWorkflow(context.Background(), input, options)
	if err != nil {
		return nil, err
	}
	if err := p.preface(); err != nil {
		return nil, err
	}
	root, props, err := p.imageKwargs()
	if err != nil {
		return nil, err
	}
	result := &ImageRecordCreateResult{}
	err = p.createUploaded(root, props, result)
	return result, err
}
func imageCreateUploadSeed(req *http.Request, methods string) *http.Response {
	response := taskCoreJSON(req, 201, `{"id":"created","status":"active"}`)
	if methods != "" {
		response.Header.Set("OpenStack-image-import-methods", methods)
	}
	return response
}

func TestImageRecordCreateUploadDirectAcceptsOpaqueResponsesWithoutQueuedGateWaitOrRefresh(t *testing.T) {
	for _, test := range []struct {
		code int
		body string
	}{{200, "opaque\xff"}, {201, "not JSON"}, {202, `{"id":"decoy","status":"queued"}`}, {204, ""}, {299, `null`}, {300, `false`}, {304, `[]`}, {399, `"opaque"`}} {
		t.Run(fmt.Sprint(test.code), func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					if req.Method != http.MethodPost || req.URL.EscapedPath() != "/reverse/glance/v2/images" {
						t.Fatal(req.Method, req.URL)
					}
					body := taskCorePayload(t, req)
					th.AssertEquals(t, `"formal"`, string(body["name"]))
					th.AssertEquals(t, `"qcow2"`, string(body["disk_format"]))
					return imageCreateUploadSeed(req, ""), nil
				}
				if calls != 2 || req.Method != http.MethodPut || req.URL.EscapedPath() != "/reverse/glance/v2/images/created/file" {
					t.Fatal(calls, req.Method, req.URL)
				}
				th.AssertEquals(t, "payload", imageRecordStageRead(t, req))
				th.AssertEquals(t, "application/octet-stream", req.Header.Get("Content-Type"))
				th.AssertEquals(t, "", req.Header.Get("Accept"))
				th.AssertEquals(t, "", req.Header.Get("X-OpenStack-Image-Size"))
				return taskCoreJSON(req, test.code, test.body), nil
			})
			got, err := imageCreateUploadRun(t, New(client), ImageRecordCreateRequest{Name: "formal", Data: ImageRecordCreateBytes([]byte("payload"))}, WithImageRecordCreateWait(true), WithImageRecordCreateTimeout(0))
			if err != nil || got == nil || got.Record == nil || got.Created == nil || got.Uploaded == nil || calls != 2 {
				t.Fatal(got, err, calls)
			}
			if got.Outcome != "uploaded" || got.Uploaded.StatusCode != test.code || string(got.Uploaded.Body) != test.body || string(got.Record.Resource.Body["status"]) != `"active"` || got.Record.data == nil || got.ChecksumFetched != nil || got.Cleanup != nil {
				t.Fatal(got)
			}
		})
	}
}

func TestImageRecordCreateUploadImportMethodRejectionAfterCreateHasNoCleanup(t *testing.T) {
	for _, methods := range []string{"", "web-download", " glance-direct", "glance-direct "} {
		t.Run(methods, func(t *testing.T) {
			calls := 0
			reader := &imageUploadCoreReader{reader: strings.NewReader("borrowed")}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls != 1 || req.Method != http.MethodPost {
					t.Fatal(calls, req.Method)
				}
				return imageCreateUploadSeed(req, methods), nil
			})
			got, err := imageCreateUploadRun(t, New(client), ImageRecordCreateRequest{Name: "formal", Data: ImageRecordCreateReader(reader)}, WithImageRecordCreateUseImport(true))
			if got == nil || got.Created == nil || got.Record == nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || got.Cleanup != nil || reader.reads.Load() != 0 || reader.seeks.Load() != 0 || reader.closes.Load() != 0 {
				t.Fatal(got, err, calls, reader)
			}
		})
	}
}

func TestImageRecordCreateUploadStoreImplicationAndEmptyStorePresence(t *testing.T) {
	for _, test := range []struct {
		name string
		opts []ImageRecordImportOption
		want string
	}{
		{"plural", []ImageRecordImportOption{WithImageRecordImportStores(ImageRecordImportStore{ID: "one"}, ImageRecordImportStore{RawID: json.RawMessage(`7`)})}, `{"method":{"name":"glance-direct"},"stores":["one",7]}`},
		{"all", []ImageRecordImportOption{WithImageRecordImportAllStores(true)}, `{"method":{"name":"glance-direct"},"all_stores":true}`},
		{"must", []ImageRecordImportOption{WithImageRecordImportAllStoresMustSucceed("truthy")}, `{"method":{"name":"glance-direct"},"all_stores_must_succeed":"truthy"}`},
		{"empty suppresses", []ImageRecordImportOption{WithImageRecordImportOpts(ImageRecordImportOpts{Stores: []ImageRecordImportStore{}, AllStores: json.RawMessage(`true`), AllStoresMustSucceed: json.RawMessage(`false`)})}, `{"method":{"name":"glance-direct"}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				switch calls {
				case 1:
					return imageCreateUploadSeed(req, "glance-direct"), nil
				case 2:
					if req.Method != http.MethodPut || req.URL.EscapedPath() != "/reverse/glance/v2/images/created/stage" {
						t.Fatal(req.Method, req.URL)
					}
					th.AssertEquals(t, "payload", imageRecordStageRead(t, req))
					response := taskCoreJSON(req, 399, "opaque stage")
					response.Header.Set("OpenStack-image-import-methods", "stage decoy")
					return response, nil
				case 3:
					if req.Method != http.MethodPost || req.URL.EscapedPath() != "/reverse/glance/v2/images/created/import" {
						t.Fatal(req.Method, req.URL)
					}
					imageRecordImportPayload(t, req, test.want)
					th.AssertEquals(t, "", req.Header.Get("X-Image-Meta-Store"))
					return taskCoreJSON(req, 299, "opaque import"), nil
				default:
					t.Fatal("unexpected refresh or wait", calls)
					return nil, nil
				}
			})
			client.MoreHeaders = map[string]string{"X-Image-Meta-Store": "configured decoy"}
			got, err := imageCreateUploadRun(t, New(client), ImageRecordCreateRequest{Name: "formal", Data: ImageRecordCreateBytes([]byte("payload"))}, WithImageRecordCreateImportOptions(test.opts...))
			if got == nil || got.Record == nil || got.Staged == nil || got.Imported == nil || err != nil || calls != 3 || got.Outcome != "imported" || got.Uploaded != nil || got.ChecksumFetched != nil || len(got.Record.ImportMethods) != 0 || got.Record.StatusCode != 399 || string(got.Staged.Body) != "opaque stage" {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordCreateUploadStoreConflictPrecedesFileOpenAndMetadataConversion(t *testing.T) {
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		t.Fatal("unexpected HTTP", req)
		return nil, nil
	})
	got, err := imageCreateUploadRun(t, New(client), ImageRecordCreateRequest{Name: "formal", Filename: filepath.Join(t.TempDir(), "absent"), Attributes: map[string]any{"min_disk": nil}}, WithImageRecordCreateMD5("supplied"), WithImageRecordCreateImportOptions(WithImageRecordImportAllStores(true), WithImageRecordImportStores(ImageRecordImportStore{ID: "one"})))
	if got == nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || got.Created != nil || got.Cleanup != nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatal(got, err, calls)
	}
}

func TestImageRecordCreateUploadImportMethodSpecificForwardingAndNoBinaryRead(t *testing.T) {
	for _, test := range []struct{ method, want string }{
		{"web-download", `{"method":{"name":"web-download","uri":"https://source.example/image"}}`},
		{"glance-download", `{"method":{"name":"glance-download","glance_region":"region","glance_image_id":"remote","glance_service_interface":"internal"}}`},
		{"copy-image", `{"method":{"name":"copy-image"}}`},
		{"future-method", `{"method":{"name":"future-method"}}`},
	} {
		t.Run(test.method, func(t *testing.T) {
			calls := 0
			reader := &imageUploadCoreReader{reader: strings.NewReader("borrowed")}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return imageCreateUploadSeed(req, test.method), nil
				}
				if calls != 2 || req.Method != http.MethodPost || !strings.HasSuffix(req.URL.Path, "/import") {
					t.Fatal(calls, req.Method, req.URL)
				}
				imageRecordImportPayload(t, req, test.want)
				return taskCoreJSON(req, 202, "accepted"), nil
			})
			got, err := imageCreateUploadRun(t, New(client), ImageRecordCreateRequest{Name: "formal", Data: ImageRecordCreateReader(reader)}, WithImageRecordCreateUseImport(true), WithImageRecordCreateSize("ignored nonbinary size"), WithImageRecordCreateImportOptions(WithImageRecordImportMethod(test.method), WithImageRecordImportURI("https://source.example/image"), WithImageRecordImportRemoteRegion("region"), WithImageRecordImportRemoteImageID("remote"), WithImageRecordImportRemoteServiceInterface("internal")))
			if got == nil || got.Imported == nil || err != nil || calls != 2 || got.Staged != nil || got.Uploaded != nil || reader.reads.Load() != 0 || reader.seeks.Load() != 0 || reader.closes.Load() != 0 {
				t.Fatal(got, err, calls, reader)
			}
		})
	}
}

func TestImageRecordCreateUploadSizePreservesPythonIntegersAndRejectsOtherTypesInsideCleanup(t *testing.T) {
	for _, test := range []struct {
		name  string
		size  any
		want  string
		valid bool
	}{
		{"bool true", true, "True", true}, {"bool false", false, "False", true}, {"negative", -9, "-9", true}, {"zero", 0, "0", true}, {"negative zero", json.RawMessage(`-0`), "0", true}, {"big integer", json.RawMessage(`900719925474099312345`), "900719925474099312345", true},
		{"fraction", 1.25, "", false}, {"integral float", json.RawMessage(`1.0`), "", false}, {"exponent", json.RawMessage(`1e2`), "", false}, {"string", "4", "", false}, {"array", []any{}, "", false}, {"object", map[string]any{}, "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return imageCreateUploadSeed(req, ""), nil
				}
				if test.valid {
					if calls != 2 || req.Method != http.MethodPut {
						t.Fatal(calls, req.Method)
					}
					th.AssertEquals(t, test.want, req.Header.Get("X-OpenStack-Image-Size"))
					return taskCoreJSON(req, 204, ""), nil
				}
				if calls != 2 || req.Method != http.MethodDelete || req.URL.EscapedPath() != "/reverse/glance/v2/images/created" {
					t.Fatal(calls, req.Method, req.URL)
				}
				return taskCoreJSON(req, 204, "cleanup"), nil
			})
			got, err := imageCreateUploadRun(t, New(client), ImageRecordCreateRequest{Name: "formal", Data: ImageRecordCreateBytes([]byte("payload"))}, WithImageRecordCreateSize(test.size))
			if got == nil || got.Created == nil || calls != 2 {
				t.Fatal(got, err, calls)
			}
			if test.valid {
				if err != nil || got.Uploaded == nil || got.Cleanup != nil {
					t.Fatal(got, err)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) || got.Uploaded != nil || got.Cleanup == nil || got.Cleanup.Acknowledgement == nil {
				t.Fatal(got, err)
			}
		})
	}
}

func TestImageRecordCreateUploadReaderSizeRestoresCursorAndBorrowedLifetime(t *testing.T) {
	reader := &imageRecordStageSeeker{reader: strings.NewReader("abcdef")}
	if _, err := reader.reader.Seek(2, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return imageCreateUploadSeed(req, ""), nil
		}
		if calls != 2 || req.Method != http.MethodPut {
			t.Fatal(calls, req.Method)
		}
		th.AssertEquals(t, "6", req.Header.Get("X-OpenStack-Image-Size"))
		th.AssertEquals(t, "cdef", imageRecordStageRead(t, req))
		return taskCoreJSON(req, 204, ""), nil
	})
	got, err := imageCreateUploadRun(t, New(client), ImageRecordCreateRequest{Name: "formal", Data: ImageRecordCreateReader(reader)})
	if got == nil || got.Record == nil || got.Record.data != reader || err != nil || calls != 2 || len(reader.whences) != 3 {
		t.Fatal(got, err, calls, reader)
	}
}

func TestImageRecordCreateUploadFilenameOpenedBeforeConversionAndNotRetainedClosed(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "image.raw")
	if err := os.WriteFile(filename, []byte("contents"), 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return imageCreateUploadSeed(req, ""), nil
		}
		if calls != 2 || req.Method != http.MethodPut {
			t.Fatal(calls, req.Method)
		}
		th.AssertEquals(t, "8", req.Header.Get("X-OpenStack-Image-Size"))
		th.AssertEquals(t, "contents", imageRecordStageRead(t, req))
		return taskCoreJSON(req, 204, ""), nil
	})
	got, err := imageCreateUploadRun(t, New(client), ImageRecordCreateRequest{Name: "formal", Filename: filename}, WithImageRecordCreateMD5("supplied"))
	if got == nil || got.Record == nil || got.Record.data != nil || err != nil || calls != 2 {
		t.Fatal(got, err, calls)
	}
	calls = 0
	got, err = imageCreateUploadRun(t, New(client), ImageRecordCreateRequest{Name: "formal", Filename: filename + "absent", Attributes: map[string]any{"min_disk": nil}}, WithImageRecordCreateMD5("supplied"))
	if got == nil || !errors.Is(err, os.ErrNotExist) || errors.Is(err, resource.ErrInvalidOption) || calls != 0 || got.Created != nil || got.Cleanup != nil {
		t.Fatal(got, err, calls)
	}
}

func TestImageRecordCreateUploadChecksumUsesFlatMetaBeforeConstructorHook(t *testing.T) {
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1:
			body := taskCorePayload(t, req)
			th.AssertEquals(t, `"constructor decoy"`, string(body[imageCreateMD5Key]))
			return imageCreateUploadSeed(req, ""), nil
		case 2:
			return taskCoreJSON(req, 204, ""), nil
		case 3:
			if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images/created" {
				t.Fatal(req.Method, req.URL)
			}
			return taskCoreJSON(req, 203, `{"checksum":1,"status":"saving"}`), nil
		default:
			t.Fatal("unexpected cleanup or final wait", calls)
			return nil, nil
		}
	})
	got, err := imageCreateUploadRun(t, New(client), ImageRecordCreateRequest{Name: "formal", Data: ImageRecordCreateBytes([]byte("contents"))}, WithImageRecordCreateValidateChecksum(true), WithImageRecordCreateMD5("formal"), WithImageRecordCreateMeta(map[string]any{imageCreateMD5Key: true, imageCreateSHA256Key: nil, "__conflicting_attrs": map[string]any{imageCreateMD5Key: "constructor decoy"}}))
	if got == nil || got.ChecksumFetched == nil || got.Checksum == nil || !got.Checksum.Compared || !got.Checksum.Matched || string(got.Checksum.MD5) != "true" || string(got.Checksum.Actual) != "1" || err != nil || calls != 3 || got.Cleanup != nil {
		t.Fatal(got, err, calls)
	}
}

func TestImageRecordCreateUploadFalseyRemoteChecksumRemainsUnverified(t *testing.T) {
	for _, actual := range []string{"null", "false", "0", `""`, `[]`, `{}`} {
		t.Run(actual, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				switch calls {
				case 1:
					return imageCreateUploadSeed(req, ""), nil
				case 2:
					return taskCoreJSON(req, 204, ""), nil
				case 3:
					return taskCoreJSON(req, 200, `{"checksum":`+actual+`}`), nil
				default:
					t.Fatal(calls, req.Method)
					return nil, nil
				}
			})
			got, err := imageCreateUploadRun(t, New(client), ImageRecordCreateRequest{Name: "formal", Data: ImageRecordCreateBytes([]byte("contents"))}, WithImageRecordCreateValidateChecksum(true), WithImageRecordCreateMD5("expected"))
			if got == nil || got.Checksum == nil || got.Checksum.Compared || got.Checksum.Matched || got.ChecksumFetched == nil || string(got.Checksum.Actual) != actual || got.Cleanup != nil || err != nil || calls != 3 {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordCreateUploadChecksumFailureDeletesFetchedPrivateCurrentID(t *testing.T) {
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1:
			return imageCreateUploadSeed(req, ""), nil
		case 2:
			return taskCoreJSON(req, 204, ""), nil
		case 3:
			return taskCoreJSON(req, 203, `{"id":"fetched-current","checksum":"mismatch"}`), nil
		case 4:
			if req.Method != http.MethodDelete || req.URL.EscapedPath() != "/reverse/glance/v2/images/fetched-current" {
				t.Fatal(req.Method, req.URL)
			}
			return taskCoreJSON(req, 204, "opaque cleanup"), nil
		default:
			t.Fatal(calls, req.Method)
			return nil, nil
		}
	})
	got, err := imageCreateUploadRun(t, New(client), ImageRecordCreateRequest{Name: "formal", Data: ImageRecordCreateBytes([]byte("contents"))}, WithImageRecordCreateValidateChecksum(true), WithImageRecordCreateMD5("expected"))
	if got == nil || got.Checksum == nil || !got.Checksum.Compared || got.Checksum.Matched || got.ChecksumFetched == nil || got.Cleanup == nil || got.Cleanup.Acknowledgement == nil || got.Cleanup.Acknowledgement.ImageID != "fetched-current" || !errors.Is(err, resource.ErrInvalidOption) || calls != 4 {
		t.Fatal(got, err, calls)
	}
}

func TestImageRecordCreateUploadNativeFailuresAndCleanMissingCleanupPreserveOriginalEvidence(t *testing.T) {
	for _, phase := range []string{"upload", "stage", "import", "checksum"} {
		t.Run(phase, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return imageCreateUploadSeed(req, "glance-direct"), nil
				}
				if req.Method == http.MethodDelete {
					return taskCoreJSON(req, 404, `{"message":"already gone"}`), nil
				}
				fail := phase == "upload" && strings.HasSuffix(req.URL.Path, "/file") || phase == "stage" && strings.HasSuffix(req.URL.Path, "/stage") || phase == "import" && strings.HasSuffix(req.URL.Path, "/import") || phase == "checksum" && req.Method == http.MethodGet
				if fail {
					return taskCoreJSON(req, 409, `{"message":"original failure"}`), nil
				}
				return taskCoreJSON(req, 204, ""), nil
			})
			opts := []ImageRecordCreateOption{WithImageRecordCreateMD5("expected")}
			if phase == "stage" || phase == "import" {
				opts = append(opts, WithImageRecordCreateUseImport(true))
			}
			if phase == "checksum" {
				opts = append(opts, WithImageRecordCreateValidateChecksum(true))
			}
			got, err := imageCreateUploadRun(t, New(client), ImageRecordCreateRequest{Name: "formal", Data: ImageRecordCreateBytes([]byte("contents"))}, opts...)
			if got == nil || got.Created == nil || err == nil || got.Cleanup != nil || strings.Contains(err.Error(), "already gone") {
				t.Fatal(got, err, calls)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != 409 || string(native.Body) != `{"message":"original failure"}` || native.ResponseHeader.Get("X-Task-Proof") != "actual" {
				t.Fatal(err, native)
			}
			switch phase {
			case "upload":
				if got.Uploaded != nil || calls != 3 {
					t.Fatal(got, calls)
				}
			case "stage":
				if got.Staged != nil || got.Imported != nil || calls != 3 {
					t.Fatal(got, calls)
				}
			case "import":
				if got.Staged == nil || got.Imported != nil || calls != 4 {
					t.Fatal(got, calls)
				}
			case "checksum":
				if got.Uploaded == nil || got.ChecksumFetched != nil || calls != 4 {
					t.Fatal(got, calls)
				}
			}
		})
	}
}

func TestImageRecordCreateUploadAcceptedResponseFaultAndCleanupFaultAreJoinedWithReceipts(t *testing.T) {
	uploadClose, cleanupClose := errors.New("upload response close"), errors.New("cleanup response close")
	calls := 0
	var uploadBody, cleanupBody *taskCoreBody
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1:
			return imageCreateUploadSeed(req, ""), nil
		case 2:
			uploadBody = &taskCoreBody{reader: strings.NewReader("upload acknowledgement"), closeErr: uploadClose}
			return taskCoreHTTP(req, 299, uploadBody), nil
		case 3:
			if req.Method != http.MethodDelete {
				t.Fatal(req.Method)
			}
			cleanupBody = &taskCoreBody{reader: strings.NewReader("cleanup acknowledgement"), closeErr: cleanupClose}
			return taskCoreHTTP(req, 399, cleanupBody), nil
		default:
			t.Fatal(calls)
			return nil, nil
		}
	})
	got, err := imageCreateUploadRun(t, New(client), ImageRecordCreateRequest{Name: "formal", Data: ImageRecordCreateBytes([]byte("contents"))})
	if got == nil || got.Uploaded == nil || got.Cleanup == nil || got.Cleanup.Acknowledgement == nil || got.Cleanup.Record != nil || !errors.Is(err, uploadClose) || !errors.Is(err, cleanupClose) || uploadBody.closes != 1 || cleanupBody.closes != 1 || calls != 3 {
		t.Fatal(got, err, calls)
	}
	th.AssertEquals(t, "upload acknowledgement", string(got.Uploaded.Body))
	th.AssertEquals(t, "cleanup acknowledgement", string(got.Cleanup.Acknowledgement.Body))
}

func TestImageRecordCreateUploadImportFormatsFailAfterStageAndTriggerImageCleanup(t *testing.T) {
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1:
			body := taskCorePayload(t, req)
			th.AssertEquals(t, "null", string(body["disk_format"]))
			return imageCreateUploadSeed(req, "glance-direct"), nil
		case 2:
			if req.Method != http.MethodPut || !strings.HasSuffix(req.URL.Path, "/stage") {
				t.Fatal(req.Method, req.URL)
			}
			return taskCoreJSON(req, 204, "stage"), nil
		case 3:
			if req.Method != http.MethodDelete || req.URL.EscapedPath() != "/reverse/glance/v2/images/created" {
				t.Fatal(req.Method, req.URL)
			}
			return taskCoreJSON(req, 204, "cleanup"), nil
		default:
			t.Fatal("format failure unexpectedly imported", calls, req.Method)
			return nil, nil
		}
	})
	got, err := imageCreateUploadRun(t, New(client), ImageRecordCreateRequest{Name: "formal", Data: ImageRecordCreateBytes([]byte("payload"))}, WithImageRecordCreateUseImport(true), WithImageRecordCreateMeta(map[string]any{"disk_format": nil}))
	if got == nil || got.Created == nil || got.Staged == nil || got.Imported != nil || got.Cleanup == nil || got.Cleanup.Acknowledgement == nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 3 {
		t.Fatal(got, err, calls)
	}
}

func TestImageRecordCreateUploadRestoredSourceFailureNeverUsesDetachedCleanup(t *testing.T) {
	calls := 0
	var clientEndpoint string
	var client = taskCoreClient(nil)
	clientEndpoint = client.Endpoint
	reader := &taskCoreReader{body: "payload", action: func() { client.Endpoint = "https://foreign.example/" }}
	client.HTTPClient.Transport = taskCoreTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return imageCreateUploadSeed(req, ""), nil
		}
		if calls != 2 || req.Method != http.MethodPut {
			t.Fatal("source failure continued into cleanup", calls, req.Method)
		}
		_, readErr := io.ReadAll(req.Body)
		if !errors.Is(readErr, resource.ErrInvalidOption) {
			t.Fatal("missing guarded data error", readErr)
		}
		client.Endpoint = clientEndpoint
		return nil, readErr
	})
	got, err := imageCreateUploadRun(t, New(client), ImageRecordCreateRequest{Name: "formal", Data: ImageRecordCreateReader(reader)})
	if got == nil || got.Created == nil || got.Record == nil || got.Uploaded != nil || got.Cleanup != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 2 || client.Endpoint != clientEndpoint {
		t.Fatal(got, err, calls)
	}
}
