package image

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	objectapi "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func cloudDeleteService(t *testing.T, useTasks string, routes *[]string, handle func(route string, req *http.Request) *http.Response) *Service {
	t.Helper()
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		route := req.Method + " " + req.URL.Host + req.URL.EscapedPath()
		if req.URL.RawQuery != "" {
			route += "?" + req.URL.RawQuery
		}
		*routes = append(*routes, route)
		if req.URL.Host == "glance.example" && req.Header.Get("X-Cloud") != "delete" {
			t.Fatal(route, req.Header)
		}
		if req.URL.Host == "swift.example" && req.Header.Get("X-Cloud") != "" {
			t.Fatal("image headers do not reach Swift", route)
		}
		return handle(route, req), nil
	})
	swift := objectapi.New(&gophercloud.ServiceClient{ProviderClient: client.ProviderClient, Endpoint: "https://swift.example/v1/AUTH_project/", Type: "object-store"})
	policy := ImageCreatePolicy{}
	if useTasks != "" {
		policy.UseTasks = json.RawMessage(useTasks)
	}
	cloud := "current"
	return NewWithDependencies(client, Dependencies{
		CreatePolicy:  policy,
		ObjectStorage: func(context.Context) (*objectapi.Service, error) { return swift, nil },
		CloudLocation: func() (resource.CloudLocation, error) { return resource.CloudLocation{Cloud: &cloud}, nil },
	})
}

const (
	cloudDeleteGet    = "GET glance.example/reverse/glance/v2/images/img"
	cloudDeleteDelete = "DELETE glance.example/reverse/glance/v2/images/img"
	cloudDeleteName   = "GET glance.example/reverse/glance/v2/images?name=img"
	cloudDeleteHidden = "GET glance.example/reverse/glance/v2/images?os_hidden=True"
)

func cloudDeleteAbsent(route string, req *http.Request) *http.Response {
	if route == cloudDeleteGet {
		return taskCoreJSON(req, 404, `{}`)
	}
	return taskCoreJSON(req, 200, `{"images":[]}`)
}

func TestCloudImageRecordDeleteFindsDeletesCleansAndWaits(t *testing.T) {
	for _, test := range []struct {
		name, useTasks, image, object string
	}{
		// Swift object names keep later slashes; the object engine escapes them.
		{"openstack key with tasks", "true", `{"id":"img","owner_specified.openstack.object":"images/obj/part"}`, "images/obj%2Fpart"},
		{"shade key with truthy tasks", `"yes"`, `{"id":"img","owner_specified.shade.object":"bucket/name"}`, "bucket/name"},
		{"openstack key wins over shade", "1", `{"id":"img","owner_specified.openstack.object":"a/b","owner_specified.shade.object":"c/d"}`, "a/b"},
		{"tasks disabled skips cleanup", "", `{"id":"img","owner_specified.openstack.object":"images/obj"}`, ""},
		{"falsey tasks skips cleanup", "0", `{"id":"img","owner_specified.openstack.object":"images/obj"}`, ""},
		{"no object key", "true", `{"id":"img","other":"value"}`, ""},
		{"fetched image without properties", "true", `{"id":"img"}`, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			var routes []string
			deleted := false
			service := cloudDeleteService(t, test.useTasks, &routes, func(route string, req *http.Request) *http.Response {
				switch {
				case route == cloudDeleteGet && !deleted:
					return taskCoreJSON(req, 200, test.image)
				case route == cloudDeleteDelete:
					deleted = true
					return taskCoreJSON(req, 204, "")
				case strings.HasPrefix(route, "HEAD swift.example"):
					return taskCoreJSON(req, 200, "")
				case strings.HasPrefix(route, "DELETE swift.example"):
					return taskCoreJSON(req, 204, "")
				}
				return cloudDeleteAbsent(route, req)
			})
			result, err := service.DeleteCloudImageRecord(context.Background(), "img", WithImageRecordCloudDeleteHeader("X-Cloud", "delete"), WithImageRecordCloudDeleteWait(true), WithImageRecordCloudDeleteTimeout(5*time.Second), WithImageRecordCloudDeletePollInterval(time.Millisecond))
			if err != nil || !result.Deleted || result.Found == nil || result.Delete == nil || result.WaitLookups != 1 || result.WaitLast != nil {
				t.Fatal(result, err, routes)
			}
			want := []string{cloudDeleteGet, cloudDeleteDelete}
			if test.object != "" {
				want = append(want, "HEAD swift.example/v1/AUTH_project/"+test.object, "DELETE swift.example/v1/AUTH_project/"+test.object)
				if result.Object == nil || result.Object.Deletion == nil || result.Object.Deletion.StatusCode != 204 {
					t.Fatal(result.Object)
				}
			} else if result.Object != nil {
				t.Fatal(result.Object)
			}
			want = append(want, cloudDeleteGet, cloudDeleteName, cloudDeleteHidden)
			if strings.Join(routes, "\n") != strings.Join(want, "\n") {
				t.Fatal(routes)
			}
		})
	}
	t.Run("missing image returns false without delete", func(t *testing.T) {
		var routes []string
		service := cloudDeleteService(t, "true", &routes, cloudDeleteAbsent)
		result, err := service.DeleteCloudImageRecord(context.Background(), "img", WithImageRecordCloudDeleteHeader("X-Cloud", "delete"), WithImageRecordCloudDeleteWait(true), WithImageRecordCloudDeleteTimeout(time.Second))
		if err != nil || result.Deleted || result.Found != nil || result.Delete != nil || len(routes) != 3 {
			t.Fatal(result, err, routes)
		}
	})
	t.Run("no wait returns after cleanup", func(t *testing.T) {
		var routes []string
		service := cloudDeleteService(t, "", &routes, func(route string, req *http.Request) *http.Response {
			if route == cloudDeleteDelete {
				return taskCoreJSON(req, 204, "")
			}
			return taskCoreJSON(req, 200, `{"id":"img"}`)
		})
		result, err := service.DeleteCloudImageRecord(context.Background(), "img", WithImageRecordCloudDeleteHeader("X-Cloud", "delete"))
		if err != nil || !result.Deleted || len(routes) != 2 || result.WaitLookups != 0 {
			t.Fatal(result, err, routes)
		}
	})
}

func TestCloudImageRecordDeleteLateFailuresKeepPhases(t *testing.T) {
	for _, test := range []struct {
		name, image string
		listed      bool
	}{
		// A direct GET packs properties into a dictionary; a listed row keeps null.
		{"listed null properties raise after delete", `{"id":"img","properties":null}`, true},
		{"non-string object location", `{"id":"img","owner_specified.openstack.object":7}`, false},
		{"null object location", `{"id":"img","owner_specified.openstack.object":null}`, false},
		{"location without separator", `{"id":"img","owner_specified.shade.object":"plain"}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var routes []string
			service := cloudDeleteService(t, "true", &routes, func(route string, req *http.Request) *http.Response {
				switch {
				case route == cloudDeleteDelete:
					return taskCoreJSON(req, 204, "")
				case test.listed && route == cloudDeleteGet:
					return taskCoreJSON(req, 404, `{}`)
				case test.listed:
					return taskCoreJSON(req, 203, `{"images":[`+test.image+`]}`)
				}
				return taskCoreJSON(req, 203, test.image)
			})
			result, err := service.DeleteCloudImageRecord(context.Background(), "img", WithImageRecordCloudDeleteHeader("X-Cloud", "delete"), WithImageRecordCloudDeleteWait(true), WithImageRecordCloudDeleteTimeout(time.Second))
			want := 2
			if test.listed {
				want = 3
			}
			var response *resource.ResponseError
			if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &response) || response.StatusCode != 203 || result.Deleted || result.Delete == nil || result.Object != nil || len(routes) != want {
				t.Fatal(result, err, routes)
			}
		})
	}
	t.Run("glance delete failure stops before cleanup", func(t *testing.T) {
		var routes []string
		service := cloudDeleteService(t, "true", &routes, func(route string, req *http.Request) *http.Response {
			if route == cloudDeleteDelete {
				return taskCoreJSON(req, 409, `{"message":"protected"}`)
			}
			return taskCoreJSON(req, 200, `{"id":"img","owner_specified.openstack.object":"a/b"}`)
		})
		result, err := service.DeleteCloudImageRecord(context.Background(), "img", WithImageRecordCloudDeleteHeader("X-Cloud", "delete"))
		var operation *resource.OperationError
		if !gophercloud.ResponseCodeIs(err, http.StatusConflict) || !errors.As(err, &operation) || operation.Operation != "DeleteCloudImageRecord" || errors.As(operation.Cause, &operation) || result.Deleted || result.Found == nil || len(routes) != 2 {
			t.Fatal(result, err, routes)
		}
	})
	t.Run("wait timeout keeps last found record", func(t *testing.T) {
		var routes []string
		service := cloudDeleteService(t, "", &routes, func(route string, req *http.Request) *http.Response {
			if route == cloudDeleteDelete {
				return taskCoreJSON(req, 204, "")
			}
			return taskCoreJSON(req, 200, `{"id":"img","status":"deleted"}`)
		})
		result, err := service.DeleteCloudImageRecord(context.Background(), "img", WithImageRecordCloudDeleteHeader("X-Cloud", "delete"), WithImageRecordCloudDeleteWait(true), WithImageRecordCloudDeleteTimeout(15*time.Millisecond), WithImageRecordCloudDeletePollInterval(5*time.Millisecond))
		if !errors.Is(err, context.DeadlineExceeded) || result.Deleted || result.WaitLookups < 1 || result.WaitLast == nil {
			t.Fatal(result, err, routes)
		}
	})
}

func TestCloudImageRecordDeleteInputFailsBeforeHTTP(t *testing.T) {
	var routes []string
	service := cloudDeleteService(t, "true", &routes, func(route string, req *http.Request) *http.Response {
		t.Fatal("unexpected request", route)
		return nil
	})
	ctx := context.Background()
	for name, call := range map[string]func() error{
		"blank identity": func() error { _, err := service.DeleteCloudImageRecord(ctx, " "); return err },
		"nil option":     func() error { _, err := service.DeleteCloudImageRecord(ctx, "img", nil); return err },
		"zero interval": func() error {
			_, err := service.DeleteCloudImageRecord(ctx, "img", WithImageRecordCloudDeletePollInterval(0))
			return err
		},
		"invalid header": func() error {
			_, err := service.DeleteCloudImageRecord(ctx, "img", WithImageRecordCloudDeleteHeader("X-Bad", "a\nb"))
			return err
		},
		"invalid tasks config": func() error {
			bad := NewWithDependencies(service.client, Dependencies{CreatePolicy: ImageCreatePolicy{UseTasks: json.RawMessage(`{`)}})
			_, err := bad.DeleteCloudImageRecord(ctx, "img")
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); !errors.Is(err, resource.ErrInvalidOption) || len(routes) != 0 {
				t.Fatal(err, routes)
			}
		})
	}
}
