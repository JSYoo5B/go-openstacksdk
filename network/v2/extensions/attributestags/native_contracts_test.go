package attributestags_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeTagTransport func(*http.Request) (*http.Response, error)

func (transport nativeTagTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeTagCall struct{ method, path, body string }

func nativeTagAPI(t *testing.T, calls *[]nativeTagCall, reply func(*http.Request) (int, string)) *attributestags.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeTagTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeTagCall{req.Method, req.URL.Path, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return attributestags.New(client)
}

func nativeTagStatus(t *testing.T, err error, operation string, code int, expected []int) {
	t.Helper()
	var wrapped *resource.OperationError
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "attributestags" || !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, expected) {
		t.Fatal(operation, code, err)
	}
}

func TestNativeAttributeTagRoutesAndBodies(t *testing.T) {
	ctx := context.Background()
	var calls []nativeTagCall
	api := nativeTagAPI(t, &calls, func(req *http.Request) (int, string) {
		switch {
		case req.Method == http.MethodDelete:
			return 204, ""
		case req.Method == http.MethodPut && strings.HasSuffix(req.URL.Path, "/tags"):
			return 200, `{"tags":["a","b"]}`
		case req.Method == http.MethodPut:
			return 201, ""
		case strings.HasSuffix(req.URL.Path, "/tags"):
			return 200, `{"tags":["a","b"]}`
		case strings.HasSuffix(req.URL.Path, "/missing"):
			return 404, `{}`
		}
		return 204, ""
	})
	replaced, err := api.ReplaceAll(ctx, "networks", "n1", attributestags.ReplaceAllOpts{Tags: []string{"a", "b"}}, attributestags.WithReplaceAllField("x_extension", 1))
	if err != nil || !reflect.DeepEqual(replaced, []string{"a", "b"}) {
		t.Fatal(replaced, err)
	}
	// An empty, non-nil tag list clears every tag.
	if _, err := api.ReplaceAll(ctx, "ports", "p1", attributestags.ReplaceAllOpts{Tags: []string{}}); err != nil {
		t.Fatal(err)
	}
	listed, err := api.List(ctx, "networks", "n1")
	if err != nil || !reflect.DeepEqual(listed, []string{"a", "b"}) {
		t.Fatal(listed, err)
	}
	if err := api.Add(ctx, "networks", "n1", "c"); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "networks", "n1", "c"); err != nil {
		t.Fatal(err)
	}
	if err := api.DeleteAll(ctx, "networks", "n1"); err != nil {
		t.Fatal(err)
	}
	present, err := api.Confirm(ctx, "networks", "n1", "a")
	if err != nil || !present {
		t.Fatal(present, err)
	}
	// Confirm folds 404 into false without an error.
	absent, err := api.Confirm(ctx, "networks", "n1", "missing")
	if err != nil || absent {
		t.Fatal(absent, err)
	}
	// The resource type and tag are inserted without escaping.
	if err := api.Add(ctx, "qos/policies", "q1", "x/y"); err != nil {
		t.Fatal(err)
	}
	base := "/neutron/v2.0/networks/n1/tags"
	want := []nativeTagCall{
		// The tag body has no envelope, so an extension sits beside tags.
		{http.MethodPut, base, `{"tags":["a","b"],"x_extension":1}`},
		{http.MethodPut, "/neutron/v2.0/ports/p1/tags", `{"tags":[]}`},
		{http.MethodGet, base, ""},
		{http.MethodPut, base + "/c", ""},
		{http.MethodDelete, base + "/c", ""},
		{http.MethodDelete, base, ""},
		{http.MethodGet, base + "/a", ""},
		{http.MethodGet, base + "/missing", ""},
		{http.MethodPut, "/neutron/v2.0/qos/policies/q1/tags/x/y", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeAttributeTagStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*attributestags.API) error
	}{
		{"ReplaceAll", []int{200}, func(api *attributestags.API) error {
			_, err := api.ReplaceAll(ctx, "networks", "n1", attributestags.ReplaceAllOpts{Tags: []string{}})
			return err
		}},
		{"List", []int{200}, func(api *attributestags.API) error { _, err := api.List(ctx, "networks", "n1"); return err }},
		// Add accepts only 201 and the deletes only 204, not the default 202.
		{"Add", []int{201}, func(api *attributestags.API) error { return api.Add(ctx, "networks", "n1", "a") }},
		{"Delete", []int{204}, func(api *attributestags.API) error { return api.Delete(ctx, "networks", "n1", "a") }},
		{"DeleteAll", []int{204}, func(api *attributestags.API) error { return api.DeleteAll(ctx, "networks", "n1") }},
		{"Confirm", []int{204}, func(api *attributestags.API) error { _, err := api.Confirm(ctx, "networks", "n1", "a"); return err }},
	} {
		for _, code := range []int{200, 201, 202, 204, 500} {
			if code == call.accepted[0] {
				continue
			}
			var calls []nativeTagCall
			err := call.call(nativeTagAPI(t, &calls, func(*http.Request) (int, string) { return code, `{"tags":[]}` }))
			nativeTagStatus(t, err, call.name, code, call.accepted)
		}
	}
	t.Run("confirm keeps other errors", func(t *testing.T) {
		var calls []nativeTagCall
		present, err := nativeTagAPI(t, &calls, func(*http.Request) (int, string) { return 409, `{}` }).Confirm(ctx, "networks", "n1", "a")
		if present {
			t.Fatal(present)
		}
		nativeTagStatus(t, err, "Confirm", 409, []int{204})
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeTagCall
		api := nativeTagAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{}` })
		for name, err := range map[string]error{
			"nil tags": func() error {
				_, err := api.ReplaceAll(ctx, "networks", "n1", attributestags.ReplaceAllOpts{})
				return err
			}(),
			"tags extension": func() error {
				_, err := api.ReplaceAll(ctx, "networks", "n1", attributestags.ReplaceAllOpts{Tags: []string{}}, attributestags.WithReplaceAllField("tags", nil))
				return err
			}(),
			"nil option": func() error {
				_, err := api.ReplaceAll(ctx, "networks", "n1", attributestags.ReplaceAllOpts{Tags: []string{}}, nil)
				return err
			}(),
		} {
			var wrapped *resource.OperationError
			if !errors.As(err, &wrapped) || wrapped.Operation != "ReplaceAll" {
				t.Fatal(name, err)
			}
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
