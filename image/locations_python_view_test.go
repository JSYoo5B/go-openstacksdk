package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func locationViewService(t *testing.T, status int, body string, cloudErr error) *Service {
	t.Helper()
	client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
		return deleteCoreHTTP(status, io.NopCloser(strings.NewReader(body)), http.Header{"X-Proof": {"actual"}}), nil
	})
	cloud := "current"
	return NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
		return resource.CloudLocation{Cloud: &cloud}, cloudErr
	}})
}

func locationViewFields(t *testing.T, view *resource.RawResource) map[string]string {
	t.Helper()
	fields := make(map[string]string, len(view.Body))
	for key, raw := range view.Body {
		fields[key] = string(raw)
	}
	var location resource.CloudLocation
	if err := json.Unmarshal(view.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != "current" {
		t.Fatal(view.Body["location"], err)
	}
	delete(fields, "location")
	return fields
}

// Python Resource.create accepts every status below 400 and keeps the seeded
// ImageLocation when the response has no parseable JSON body.
func TestImageLocationPythonViewSeedAcrossAcceptedStatuses(t *testing.T) {
	for _, status := range []int{200, 201, 202, 204, 299, 300, 399} {
		for _, body := range []string{"", "not json", "{"} {
			t.Run(fmt.Sprintf("%d/%q", status, body), func(t *testing.T) {
				service := locationViewService(t, status, body, nil)
				result, err := service.AddImageLocation(context.Background(), resource.ID("img"), "http://store/obj", WithImageLocationValidation("sha512", "abc"))
				if err != nil || result == nil || result.StatusCode != status || string(result.Body) != body || result.Header.Get("X-Proof") != "actual" || result.Resource == nil || result.Resource.StatusCode != status {
					t.Fatal(result, err)
				}
				want := map[string]string{"id": "null", "name": "null", "image_id": `"img"`, "url": `"http://store/obj"`, "validation_data": `{"os_hash_algo":"sha512","os_hash_value":"abc"}`, "metadata": "null"}
				if got := locationViewFields(t, result.Resource); !reflect.DeepEqual(got, want) {
					t.Fatal(got)
				}
			})
		}
	}
	t.Run("default validation data is an empty object", func(t *testing.T) {
		result, err := locationViewService(t, 202, "", nil).AddImageLocation(context.Background(), resource.ID("img"), "http://store/obj")
		if err != nil || string(result.Resource.Body["validation_data"]) != "{}" {
			t.Fatal(result, err)
		}
	})
}

func TestImageLocationPythonViewOverlayAndFailures(t *testing.T) {
	t.Run("JSON object overlays declared fields only", func(t *testing.T) {
		body := `{"id":"loc","name":null,"url":"server-url","validation_data":{"os_hash_algo":"x"},"metadata":{"store":"a"},"self":"/v2/x","image_id":"other","status":"active"}`
		result, err := locationViewService(t, 200, body, nil).AddImageLocation(context.Background(), resource.ID("img"), "http://store/obj")
		if err != nil || result == nil || result.URL != "http://store/obj" {
			t.Fatal(result, err)
		}
		want := map[string]string{"id": `"loc"`, "name": "null", "image_id": `"img"`, "url": `"server-url"`, "validation_data": `{"os_hash_algo":"x"}`, "metadata": `{"store":"a"}`}
		if got := locationViewFields(t, result.Resource); !reflect.DeepEqual(got, want) {
			t.Fatal(got)
		}
	})
	for _, body := range []string{`[]`, `"text"`, `1`, `null`, `true`, `{"metadata":"x"}`, `{"validation_data":[["a","b"]]}`} {
		t.Run("rejected "+body, func(t *testing.T) {
			result, err := locationViewService(t, 202, body, nil).AddImageLocation(context.Background(), resource.ID("img"), "http://store/obj")
			var response *resource.ResponseError
			if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &response) || response.StatusCode != 202 || string(response.Body) != body || result == nil || result.Resource != nil || result.StatusCode != 202 {
				t.Fatal(result, err)
			}
		})
	}
	t.Run("location failure keeps acknowledgement", func(t *testing.T) {
		cause := errors.New("location unavailable")
		result, err := locationViewService(t, 202, "", cause).AddImageLocation(context.Background(), resource.ID("img"), "http://store/obj")
		var response *resource.ResponseError
		if !errors.Is(err, cause) || !errors.As(err, &response) || result == nil || result.Resource != nil {
			t.Fatal(result, err)
		}
	})
	t.Run("no location dependency projects null", func(t *testing.T) {
		client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
			return deleteCoreHTTP(202, http.NoBody, nil), nil
		})
		result, err := New(client).AddImageLocation(context.Background(), resource.ID("img"), "http://store/obj")
		if err != nil || string(result.Resource.Body["location"]) != "null" {
			t.Fatal(result, err)
		}
	})
}
