package apiversions_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/apiversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeVersionTransport func(*http.Request) (*http.Response, error)

func (transport nativeVersionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeVersionAPI(t *testing.T, endpoint string, paths *[]string, code int, body string) *apiversions.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeVersionTransport(func(req *http.Request) (*http.Response, error) {
		*paths = append(*paths, req.Method+" "+req.URL.Path+"?"+req.URL.RawQuery)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("block-storage", endpoint)
	// The version list derives its URL from Endpoint, never from ResourceBase.
	client.ResourceBase = cloud.Server.URL + "/unused/"
	return apiversions.New(client)
}

func TestNativeVolumeAPIVersionsUseTheUnversionedRoot(t *testing.T) {
	ctx := context.Background()
	var paths []string
	api := nativeVersionAPI(t, "/cinder/v3/project?x=1", &paths, 300, `{"versions":[{"id":"v3.0","status":"CURRENT","version":"3.70","min_version":"3.0","updated":"2023-03-13T00:00:00Z"},{"id":"v2.0","status":"DEPRECATED","updated":"2017-02-25T12:00:00Z"}]}`)
	var versions []apiversions.APIVersion
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		versions = append(versions, *value)
	}
	// Cinder answers the root with 300 Multiple Choices, which the native pager accepts.
	if len(versions) != 2 || versions[0].Version != "3.70" || versions[0].MinVersion != "3.0" || !versions[1].Updated.Equal(time.Date(2017, 2, 25, 12, 0, 0, 0, time.UTC)) {
		t.Fatal(versions)
	}
	// The project-scoped versioned endpoint is cut at its version segment and the query dropped.
	if !reflect.DeepEqual(paths, []string{"GET /cinder/?"}) {
		t.Fatal(paths)
	}
}

func TestNativeVolumeAPIVersionsStatusesAndDecode(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		code int
		body string
	}{{404, `{}`}, {200, `{"versions":[]}`}, {204, ""}, {200, `{"versions":[{"id":"v3.0","updated":"not-a-time"}]}`}} {
		var paths []string
		var errs []error
		for _, err := range nativeVersionAPI(t, "/cinder/v3/", &paths, tc.code, tc.body).List(ctx) {
			errs = append(errs, err)
		}
		var native gophercloud.ErrUnexpectedResponseCode
		switch {
		case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
		case tc.body == `{"versions":[]}` && len(errs) == 0:
		case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
		// The updated field is a strict RFC3339 time.
		case strings.Contains(tc.body, "not-a-time") && len(errs) == 1:
		default:
			t.Fatal(tc.code, tc.body, errs)
		}
	}
}
