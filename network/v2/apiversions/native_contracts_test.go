package apiversions_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/apiversions"
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

func nativeVersionAPI(t *testing.T, endpoint string, paths *[]string, code int, body func(*http.Request) string) *apiversions.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeVersionTransport(func(req *http.Request) (*http.Response, error) {
		*paths = append(*paths, req.Method+" "+req.URL.Path+"?"+req.URL.RawQuery)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body(req))), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("network", endpoint)
	// The version calls derive their URL from Endpoint, never from ResourceBase.
	client.ResourceBase = cloud.Server.URL + "/unused/"
	return apiversions.New(client)
}

func TestNativeNetworkAPIVersionsUseTheUnversionedRoot(t *testing.T) {
	ctx := context.Background()
	var paths []string
	api := nativeVersionAPI(t, "/neutron/v2.0/?x=1", &paths, 200, func(req *http.Request) string {
		if req.URL.Path == "/neutron/" {
			return `{"versions":[{"id":"v2.0","status":"CURRENT","links":[]}],"links":{"next":"/never"}}`
		}
		return `{"resources":[{"name":"network","collection":"networks","links":[]},{"name":"port","collection":"ports"}]}`
	})
	var versions []string
	for value, err := range api.ListVersions(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		versions = append(versions, value.ID+"="+value.Status)
	}
	var resources []string
	for value, err := range api.ListVersionResources(ctx, "v2.0/") {
		if err != nil {
			t.Fatal(err)
		}
		resources = append(resources, value.Name+"="+value.Collection)
	}
	// The versioned endpoint is cut at its version segment and the query is dropped;
	// a trailing slash in the version argument is trimmed before one is added back.
	want := []string{"GET /neutron/?", "GET /neutron/v2.0/?"}
	if !reflect.DeepEqual(versions, []string{"v2.0=CURRENT"}) || !reflect.DeepEqual(resources, []string{"network=networks", "port=ports"}) || !reflect.DeepEqual(paths, want) {
		t.Fatal(versions, resources, paths)
	}
}

func TestNativeNetworkAPIVersionsStatuses(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		code int
		body string
	}{{404, `{}`}, {200, `{"versions":[],"resources":[]}`}, {204, ""}} {
		for name, list := range map[string]func(*apiversions.API) []error{
			"versions": func(api *apiversions.API) (errs []error) {
				for _, err := range api.ListVersions(ctx) {
					errs = append(errs, err)
				}
				return errs
			},
			"resources": func(api *apiversions.API) (errs []error) {
				for _, err := range api.ListVersionResources(ctx, "v2.0") {
					errs = append(errs, err)
				}
				return errs
			},
		} {
			var paths []string
			errs := list(nativeVersionAPI(t, "/neutron/v2.0/", &paths, tc.code, func(*http.Request) string { return tc.body }))
			var native gophercloud.ErrUnexpectedResponseCode
			switch {
			case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
			case tc.code == 200 && len(errs) == 0:
			case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
			default:
				t.Fatal(name, tc.code, errs)
			}
		}
	}
}
