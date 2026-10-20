package image

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestImageRecordCreateWholeForwardsOwnerAndVisibilityToServerPolicy(t *testing.T) {
	for _, visibility := range []string{"public", "community"} {
		for _, status := range []int{201, 403} {
			t.Run(visibility+http.StatusText(status), func(t *testing.T) {
				calls := 0
				client := taskCoreClient(func(request *http.Request) (*http.Response, error) {
					calls++
					if request.Method != "POST" || request.URL.Path != "/reverse/glance/v2/images" {
						t.Fatal(request.Method, request.URL)
					}
					body := taskCorePayload(t, request)
					th.AssertEquals(t, `"target-project"`, string(body["owner"]))
					th.AssertEquals(t, `"`+visibility+`"`, string(body["visibility"]))
					return taskCoreJSON(request, status, `{"id":"created","message":"server policy"}`), nil
				})
				result, err := New(client).CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "owner-change", Attributes: map[string]any{"owner": "target-project", "visibility": visibility}}, WithImageRecordCreateAllowDuplicates(true))
				if calls != 1 {
					t.Fatal(calls)
				}
				if status == 201 {
					if err != nil || result == nil || result.Outcome != "metadata-only" {
						t.Fatal(result, err)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != `{"id":"created","message":"server policy"}` {
						t.Fatal(result, err)
					}
				}
			})
		}
	}
}

func TestImageRecordCreateWholeNativeAdminDenialKeepsBranchCleanupBoundary(t *testing.T) {
	t.Run("copy import denial deletes created image", func(t *testing.T) {
		calls := 0
		client := taskCoreClient(func(request *http.Request) (*http.Response, error) {
			calls++
			switch calls {
			case 1:
				return imageCreateUploadSeed(request, "copy-image"), nil
			case 2:
				if request.Method != "POST" || request.URL.Path != "/reverse/glance/v2/images/created/import" {
					t.Fatal(request.Method, request.URL)
				}
				return taskCoreJSON(request, 403, "copy denied"), nil
			case 3:
				if request.Method != "DELETE" {
					t.Fatal(request.Method)
				}
				return taskCoreJSON(request, 204, "cleanup"), nil
			}
			t.Fatal(calls)
			return nil, nil
		})
		result, err := New(client).CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "copy"}, WithImageRecordCreateAllowDuplicates(true), WithImageRecordCreateUseImport(true), WithImageRecordCreateImportOptions(WithImageRecordImportMethod("copy-image")))
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != 403 || result == nil || result.Created == nil || result.Cleanup == nil || result.Cleanup.Acknowledgement.StatusCode != 204 || calls != 3 {
			t.Fatal(result, err, calls)
		}
	})
	t.Run("Task denial retains uploaded object before wait", func(t *testing.T) {
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
				if request.Method != "POST" || request.URL.Path != "/reverse/glance/v2/tasks" {
					t.Fatal(request.Method, request.URL)
				}
				return taskCoreJSON(request, 403, "Task denied"), nil
			}
			t.Fatal(calls)
			return nil, nil
		})
		result, err := service.CreateImageRecord(context.Background(), ImageRecordCreateRequest{Name: "task", Data: ImageRecordCreateText("data")}, imageCreateTaskOptions(WithImageRecordCreateWait(true))...)
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != 403 || result == nil || result.Swift == nil || result.Swift.Object == nil || result.Swift.Cleanup != nil || calls != 3 {
			t.Fatal(result, err, calls)
		}
	})
}
