package tags_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/tags"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonTagCall records the wire request openstacksdk would also send.
type pythonTagCall struct{ method, path, body, version string }

type pythonTagTransport func(*http.Request) (*http.Response, error)

func (transport pythonTagTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonTagClient(t *testing.T, reply func(*http.Request) int) (*gophercloud.ServiceClient, *[]pythonTagCall) {
	t.Helper()
	calls := &[]pythonTagCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonTagTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonTagCall{req.Method, req.URL.EscapedPath(), raw, req.Header.Get("X-OpenStack-Nova-API-Version")})
		code := reply(req)
		body := ""
		if code == http.StatusNotFound {
			body = `{"itemNotFound":{"message":"gone"}}`
		}
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	client.Microversion = "2.26"
	return client, calls
}

func TestPythonServerTagAddRemoveAndRemoveAllMatchTagMixin(t *testing.T) {
	ctx := context.Background()
	status := http.StatusNoContent
	client, calls := pythonTagClient(t, func(req *http.Request) int {
		if req.Method == http.MethodPut {
			return http.StatusCreated
		}
		return status
	})
	scope, err := tags.New(client).InServer(ctx, resource.ID("s1"))
	if err != nil {
		t.Fatal(err)
	}
	// add_tag_to_server(server, "blue")
	if err := scope.Add(ctx, "blue"); err != nil {
		t.Fatal(err)
	}
	// remove_tag_from_server(server, "blue") and remove_tags_from_server(server) raise on 404,
	// which WithMissingError reproduces; the Go default ignores 404.
	if err := scope.Remove(ctx, "blue", tags.WithMissingError()); err != nil {
		t.Fatal(err)
	}
	if err := scope.RemoveAll(ctx, tags.WithMissingError()); err != nil {
		t.Fatal(err)
	}
	status = http.StatusNotFound
	if err := scope.Remove(ctx, "blue", tags.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if err := scope.RemoveAll(ctx, tags.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if err := scope.Remove(ctx, "blue"); err != nil {
		t.Fatal(err)
	}
	base := "/nova/v2.1/servers/s1/tags"
	want := []pythonTagCall{
		{http.MethodPut, base + "/blue", "", "2.26"},
		{http.MethodDelete, base + "/blue", "", "2.26"},
		{http.MethodDelete, base, "", "2.26"},
		{http.MethodDelete, base + "/blue", "", "2.26"},
		{http.MethodDelete, base, "", "2.26"},
		{http.MethodDelete, base + "/blue", "", "2.26"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonServerTagNeedsConfiguredMicroversion(t *testing.T) {
	ctx := context.Background()
	client, calls := pythonTagClient(t, func(*http.Request) int { return http.StatusNoContent })
	// Python discovers a microversion of at least 2.26; Go refuses a lower configured one before HTTP.
	client.Microversion = "2.25"
	if _, err := tags.New(client).InServer(ctx, resource.ID("s1")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	// Python joins the tag into the URL unescaped; Go escapes it and rejects slash and comma.
	client.Microversion = "2.90"
	scope, err := tags.New(client).InServer(ctx, resource.ID("s1"))
	if err != nil {
		t.Fatal(err)
	}
	if err := scope.Add(ctx, "a/b"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if err := scope.Add(ctx, "a b"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*calls, []pythonTagCall{{http.MethodPut, "/nova/v2.1/servers/s1/tags/a%20b", "", "2.90"}}) {
		t.Fatalf("%+v", *calls)
	}
}
