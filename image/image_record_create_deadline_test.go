package image

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	objectapi "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestImageRecordCreateTaskOwnDeadlineKeepsParentLiveForFinallyCleanup(t *testing.T) {
	parent := context.Background()
	calls := 0
	service, _, _ := imageCreateTaskServices(t, func(request *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1:
			return taskCoreJSON(request, 204, ""), nil
		case 2:
			_ = imageRecordStageRead(t, request)
			return taskCoreJSON(request, 201, ""), nil
		case 3:
			return taskCoreJSON(request, 201, `{"id":"task","status":"pending"}`), nil
		case 4:
			if request.Method != "GET" || request.URL.Path != "/reverse/glance/v2/tasks/task" {
				t.Fatal(request.Method, request.URL)
			}
			<-request.Context().Done()
			return nil, request.Context().Err()
		case 5:
			if request.Method != "HEAD" || request.Context().Err() != nil {
				t.Fatal(request.Method, request.Context().Err())
			}
			return taskCoreJSON(request, 204, ""), nil
		case 6:
			if request.Method != "DELETE" || request.Context().Err() != nil {
				t.Fatal(request.Method, request.Context().Err())
			}
			return taskCoreJSON(request, 204, "cleanup"), nil
		default:
			t.Fatal("unexpected extra request", calls, request.Method)
		}
		return nil, nil
	})
	result, err := service.CreateImageRecord(parent, ImageRecordCreateRequest{Name: "input", Data: ImageRecordCreateText("bytes")}, imageCreateTaskOptions(WithImageRecordCreateWait(true), WithImageRecordCreateTimeout(0.005))...)
	if !errors.Is(err, context.DeadlineExceeded) || parent.Err() != nil || result == nil || result.Swift == nil || result.Swift.Cleanup == nil || result.Swift.Cleanup.Discovery == nil || result.Swift.Cleanup.Deletion == nil || result.Swift.Cleanup.Deletion.StatusCode != 204 || calls != 6 {
		t.Fatal(result, err, calls, parent.Err())
	}
}

func TestImageRecordCreateNativeRequestDeadlineKeepsParentLiveForCleanup(t *testing.T) {
	for _, task := range []bool{false, true} {
		t.Run(map[bool]string{false: "checksum", true: "task image fetch"}[task], func(t *testing.T) {
			parent := context.Background()
			calls := 0
			transport := func(request *http.Request) (*http.Response, error) {
				calls++
				if request.Method == "GET" {
					return taskCoreHTTP(request, 403, &taskCoreBody{reader: &taskCoreReader{
						body: "timed out rejection", err: context.DeadlineExceeded,
						action: func() { <-request.Context().Done() },
					}}), nil
				}
				if request.Context().Err() != nil {
					t.Fatal("cleanup inherited a request timeout", request.Context().Err())
				}
				if task {
					switch calls {
					case 1, 5:
						return taskCoreJSON(request, 204, ""), nil
					case 2:
						_ = imageRecordStageRead(t, request)
						return taskCoreJSON(request, 201, ""), nil
					case 3:
						return taskCoreJSON(request, 201, `{"id":"task","status":"success","result":{"image_id":"created"}}`), nil
					case 6:
						return taskCoreJSON(request, 204, "cleanup"), nil
					}
				} else {
					switch calls {
					case 1:
						return imageCreateUploadSeed(request, ""), nil
					case 2:
						_ = imageRecordStageRead(t, request)
						return taskCoreJSON(request, 204, ""), nil
					case 4:
						return taskCoreJSON(request, 204, "cleanup"), nil
					}
				}
				t.Fatal("unexpected request", calls, request.Method, request.URL)
				return nil, nil
			}
			var service *Service
			options := []ImageRecordCreateOption{WithImageRecordCreateAllowDuplicates(true), WithImageRecordCreateValidateChecksum(true), WithImageRecordCreateMD5("expected")}
			if task {
				service, _, _ = imageCreateTaskServices(t, transport)
				service.RawClient().HTTPClient.Timeout = 20 * time.Millisecond
				options = imageCreateTaskOptions(WithImageRecordCreateWait(true))
			} else {
				client := taskCoreClient(transport)
				client.HTTPClient.Timeout = 20 * time.Millisecond
				service = New(client)
			}
			result, err := service.CreateImageRecord(parent, ImageRecordCreateRequest{Name: "input", Data: ImageRecordCreateBytes([]byte("bytes"))}, options...)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.Is(err, context.DeadlineExceeded) || !errors.As(err, &native) || native.Actual != 403 || parent.Err() != nil || result == nil {
				t.Fatal(result, err, calls, parent.Err())
			}
			if task {
				if calls != 6 || result.Swift == nil || result.Swift.Cleanup == nil || result.Swift.Cleanup.Deletion == nil || result.Swift.Cleanup.Deletion.StatusCode != 204 {
					t.Fatal(result, err, calls)
				}
			} else if calls != 4 || result.Cleanup == nil || result.Cleanup.Acknowledgement == nil || result.Cleanup.Acknowledgement.StatusCode != 204 {
				t.Fatal(result, err, calls)
			}
		})
	}
}

func TestImageRecordCreateTaskWaitKeepsCallerCancellationAndSourceFailureSticky(t *testing.T) {
	for _, drift := range []bool{false, true} {
		t.Run(map[bool]string{false: "caller cancellation", true: "source drift with child deadline"}[drift], func(t *testing.T) {
			parent, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			marker := errors.New("caller stopped image creation")
			calls := 0
			var swift *objectapi.Service
			service, bound, _ := imageCreateTaskServices(t, func(request *http.Request) (*http.Response, error) {
				calls++
				switch calls {
				case 1:
					return taskCoreJSON(request, 204, ""), nil
				case 2:
					_ = imageRecordStageRead(t, request)
					return taskCoreJSON(request, 201, ""), nil
				case 3:
					return taskCoreJSON(request, 201, `{"id":"task","status":"pending"}`), nil
				case 4:
					if drift {
						swift.RawClient().Endpoint += "changed/"
					} else {
						cancel(marker)
					}
					<-request.Context().Done()
					return nil, request.Context().Err()
				default:
					t.Fatal("failed caller/source permitted later cleanup", calls, request.Method)
				}
				return nil, nil
			})
			swift = bound
			result, err := service.CreateImageRecord(parent, ImageRecordCreateRequest{Name: "input", Data: ImageRecordCreateText("bytes")}, imageCreateTaskOptions(WithImageRecordCreateWait(true), WithImageRecordCreateTimeout(0.005))...)
			if result == nil || result.Swift == nil || result.Swift.Object == nil || result.Swift.Cleanup != nil || calls != 4 {
				t.Fatal(result, err, calls)
			}
			if drift {
				if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, resource.ErrInvalidOption) || parent.Err() != nil {
					t.Fatal(err, parent.Err())
				}
			} else if !errors.Is(err, context.Canceled) || !errors.Is(err, marker) {
				t.Fatal(err)
			}
		})
	}
}
