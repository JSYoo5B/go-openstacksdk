package image

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func cloudWaitSeed(t *testing.T, id string) *ImageRecord {
	t.Helper()
	raw, _ := json.Marshal(id)
	return &ImageRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": raw, "status": json.RawMessage(`"active"`)}}}}
}

func TestCloudImageRecordWaitPollsFindUntilExactActive(t *testing.T) {
	responses := []struct {
		code int
		body string
	}{
		{404, `{}`}, {200, `{"images":[]}`}, {200, `{"images":[]}`}, // missing: find returns nil
		{200, `{"id":"img","status":"queued"}`},
		{200, `{"id":"img","status":"ACTIVE"}`}, // case-sensitive equality
		{200, `{"id":"img","status":null}`},
		{200, `{"id":"img","status":1}`},
		{203, `{"id":"img","status":"active"}`},
	}
	calls := 0
	service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
		if req.Header.Get("X-Cloud") != "wait" {
			t.Fatal(req.Header)
		}
		if calls == 1 && req.URL.EscapedPath() != "/reverse/glance/v2/images/img" {
			t.Fatal(req.URL)
		}
		response := responses[calls-1]
		return taskCoreJSON(req, response.code, response.body)
	})
	// The supplied record status is ignored; Cloud always looks the image up.
	result, err := service.WaitForCloudImageRecord(context.Background(), cloudWaitSeed(t, "img"), WithImageRecordCloudWaitHeader("X-Cloud", "wait"), WithImageRecordCloudWaitPollInterval(time.Millisecond))
	if err != nil || calls != len(responses) || result.Lookups != 6 || result.Image == nil || result.Image != result.Last || result.Image.StatusCode != 203 {
		t.Fatal(result, err, calls)
	}
	var location resource.CloudLocation
	if err := json.Unmarshal(result.Image.Resource.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != "current" {
		t.Fatal(location, err)
	}
}

func TestCloudImageRecordWaitFailuresKeepLastRecord(t *testing.T) {
	t.Run("exact error state", func(t *testing.T) {
		calls := 0
		service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
			if calls == 1 {
				return taskCoreJSON(req, 200, `{"id":"img","status":"Error"}`)
			}
			return taskCoreJSON(req, 202, `{"id":"img","status":"error"}`)
		})
		result, err := service.WaitForCloudImageRecord(context.Background(), cloudWaitSeed(t, "img"), WithImageRecordCloudWaitPollInterval(time.Millisecond))
		var failed *resource.FailedStateError
		var response *resource.ResponseError
		if !errors.As(err, &failed) || failed.ID != "img" || !errors.As(err, &response) || response.StatusCode != 202 || calls != 2 || result.Image != nil || result.Last == nil || result.Lookups != 2 {
			t.Fatal(result, err, calls)
		}
	})
	t.Run("timeout keeps last receipt", func(t *testing.T) {
		calls := 0
		service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
			return taskCoreJSON(req, 200, `{"id":"img","status":"saving"}`)
		})
		result, err := service.WaitForCloudImageRecord(context.Background(), cloudWaitSeed(t, "img"), WithImageRecordCloudWaitTimeout(20*time.Millisecond), WithImageRecordCloudWaitPollInterval(5*time.Millisecond))
		var response *resource.ResponseError
		if !errors.Is(err, context.DeadlineExceeded) || !errors.As(err, &response) || response.StatusCode != 200 || calls < 2 || result.Last == nil || result.Image != nil || result.Lookups != calls {
			t.Fatal(result, err, calls)
		}
	})
	t.Run("nonpositive timeout expires before lookup", func(t *testing.T) {
		for _, timeout := range []time.Duration{0, -time.Second} {
			calls := 0
			service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
				t.Fatal("unexpected request", req.URL)
				return nil
			})
			result, err := service.WaitForCloudImageRecord(context.Background(), cloudWaitSeed(t, "img"), WithImageRecordCloudWaitTimeout(timeout))
			if !errors.Is(err, context.DeadlineExceeded) || calls != 0 || result.Lookups != 0 || result.Last != nil {
				t.Fatal(result, err)
			}
		}
	})
	t.Run("terminal lookup error stops polling", func(t *testing.T) {
		calls := 0
		service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
			if calls == 1 {
				return taskCoreJSON(req, 200, `{"id":"img","status":"queued"}`)
			}
			return taskCoreJSON(req, 500, `{"message":"server"}`)
		})
		result, err := service.WaitForCloudImageRecord(context.Background(), cloudWaitSeed(t, "img"), WithImageRecordCloudWaitUnlimited(), WithImageRecordCloudWaitPollInterval(time.Millisecond))
		if !gophercloud.ResponseCodeIs(err, http.StatusInternalServerError) || calls != 2 || result.Last == nil || result.Lookups != 2 {
			t.Fatal(result, err, calls)
		}
	})
	t.Run("cancellation during sleep", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		cause := errors.New("caller stopped")
		calls := 0
		service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
			cancel(cause)
			return taskCoreJSON(req, 200, `{"id":"img","status":"queued"}`)
		})
		result, err := service.WaitForCloudImageRecord(ctx, cloudWaitSeed(t, "img"), WithImageRecordCloudWaitPollInterval(time.Hour))
		if !errors.Is(err, cause) || calls != 1 || result == nil || result.Lookups != 1 {
			t.Fatal(result, err, calls)
		}
	})
}

func TestCloudImageRecordWaitInputFailsBeforeHTTP(t *testing.T) {
	calls := 0
	service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
		t.Fatal("unexpected request", req.URL)
		return nil
	})
	ctx := context.Background()
	numeric := &ImageRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`7`)}}}}
	for name, call := range map[string]func() error{
		"nil record": func() error { _, err := service.WaitForCloudImageRecord(ctx, nil); return err },
		"numeric id": func() error { _, err := service.WaitForCloudImageRecord(ctx, numeric); return err },
		"blank id":   func() error { _, err := service.WaitForCloudImageRecord(ctx, cloudWaitSeed(t, " ")); return err },
		"nil option": func() error { _, err := service.WaitForCloudImageRecord(ctx, cloudWaitSeed(t, "img"), nil); return err },
		"zero interval": func() error {
			_, err := service.WaitForCloudImageRecord(ctx, cloudWaitSeed(t, "img"), WithImageRecordCloudWaitPollInterval(0))
			return err
		},
		"unlimited+timeout": func() error {
			_, err := service.WaitForCloudImageRecord(ctx, cloudWaitSeed(t, "img"), WithImageRecordCloudWaitOpts(ImageRecordCloudWaitOpts{Unlimited: true, Timeout: new(time.Duration)}))
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(err, calls)
			}
		})
	}
}
