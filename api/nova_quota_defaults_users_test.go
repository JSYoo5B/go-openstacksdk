package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/compute/v2/quotasets"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestNovaQuotaDefaultsPreserveProjectAndRawResponse(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/nova/os-quota-sets/project-fixed/defaults", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.URL.RawQuery != "" || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.56" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Client-Extension") != "preserved" {
			t.Errorf("defaults request=%s %s header=%s", r.Method, r.URL, r.Header)
		}
		w.Header().Add("X-Openstack-Request-Id", "req-defaults")
		w.Header().Add("X-Vendor", "one")
		w.Header().Add("X-Vendor", "two")
		testcloud.JSON(w, 200, `{"quota_set":{"id":"wire-project","cores":-1,"instances":0,"networks":8,"counter":9007199254740993,"vendor":{"array":[1,null]},"optional":null}}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("defaults changed the target: %s %s", r.Method, r.URL)
		testcloud.JSON(w, 404, "{}")
	})
	client := cloud.Client("compute", "/nova")
	client.Microversion = "2.56"
	client.MoreHeaders = map[string]string{"X-Client-Extension": "preserved"}
	scope, err := quotasets.New(client).InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil || calls.Load() != 0 {
		t.Fatalf("scope=%v calls=%d err=%v", scope, calls.Load(), err)
	}
	first, err := scope.Defaults(context.Background())
	if err != nil || first == nil || first.ProjectID != "project-fixed" || first.ID != "wire-project" || first.Cores != -1 || first.Instances != 0 || string(first.Body["networks"]) != "8" || string(first.Body["counter"]) != "9007199254740993" || string(first.Body["optional"]) != "null" || string(first.Body["vendor"]) != `{"array":[1,null]}` || first.Header.Get("X-Openstack-Request-Id") != "req-defaults" || len(first.Header.Values("X-Vendor")) != 2 {
		t.Fatalf("defaults=%+v err=%v", first, err)
	}
	first.Header["X-Vendor"][0] = "changed"
	first.Body["counter"][0] = '0'
	second, err := scope.Defaults(context.Background())
	if err != nil || second.Header.Values("X-Vendor")[0] != "one" || string(second.Body["counter"]) != "9007199254740993" || calls.Load() != 2 || scope.ProjectID() != "project-fixed" {
		t.Fatalf("second=%+v calls=%d err=%v", second, calls.Load(), err)
	}
}

func TestNovaQuotaDefaultsPreserveHTTPAndDecodeErrors(t *testing.T) {
	for _, status := range []int{403, 404, 203} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			body := `{"error":{"message":"defaults denied"}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Openstack-Request-Id", "denied-defaults")
				testcloud.JSON(w, status, body)
			})
			value, err := newProjectQuotaScope(t, cloud).Defaults(context.Background())
			var response gophercloud.ErrUnexpectedResponseCode
			var operation *resource.OperationError
			if value != nil || !errors.As(err, &response) || response.Actual != status || string(response.Body) != body || response.ResponseHeader.Get("X-Openstack-Request-Id") != "denied-defaults" || !errors.As(err, &operation) || operation.Operation != "Defaults" || errors.Is(err, resource.ErrNotFound) != (status == 404) {
				t.Fatalf("quota=%v response=%+v operation=%+v err=%v", value, response, operation, err)
			}
		})
	}
	for _, body := range []string{`{}`, `{"quota_set":null}`, `{"quota_set":[]}`, `{"quota_set":"bad"}`, `{"quota_set":{"cores":"bad"}}`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, body) })
			value, err := newProjectQuotaScope(t, cloud).Defaults(context.Background())
			if value != nil || err == nil {
				t.Fatalf("defaults=%v err=%v", value, err)
			}
			if body == `{"quota_set":{"cores":"bad"}}` {
				var decode *json.UnmarshalTypeError
				if !errors.As(err, &decode) {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestNovaQuotaDefaultsRespectContext(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); <-r.Context().Done() })
	scope := newProjectQuotaScope(t, cloud)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if value, err := scope.Defaults(ctx); value != nil || !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatalf("defaults=%v calls=%d err=%v", value, calls.Load(), err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if value, err := scope.Defaults(ctx); value != nil || !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 1 {
		t.Fatalf("defaults=%v calls=%d err=%v", value, calls.Load(), err)
	}
}
