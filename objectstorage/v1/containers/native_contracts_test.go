package containers_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeContainerTransport func(*http.Request) (*http.Response, error)

func (transport nativeContainerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeContainerCall struct {
	method, path, query, body string
	header                    http.Header
}

func nativeContainerHeaders(h http.Header) http.Header {
	out := http.Header{}
	for key, values := range h {
		if strings.HasPrefix(key, "X-Container-") || strings.HasPrefix(key, "X-Remove-") || strings.HasPrefix(key, "X-Versions") || strings.HasPrefix(key, "X-History") ||
			key == "X-Newest" || key == "X-Storage-Policy" || key == "If-None-Match" || key == "Content-Type" || key == "Accept" || key == "X-Detect-Content-Type" || key == "X-Extra" {
			out[key] = values
		}
	}
	return out
}

type nativeContainerReply struct {
	code   int
	header http.Header
	body   string
}

func nativeContainerAPI(t *testing.T, calls *[]nativeContainerCall, reply func(*http.Request) nativeContainerReply) *containers.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeContainerTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeContainerCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, nativeContainerHeaders(req.Header)})
		r := reply(req)
		if r.header == nil {
			r.header = http.Header{}
		}
		return &http.Response{StatusCode: r.code, Body: io.NopCloser(strings.NewReader(r.body)), Header: r.header}, nil
	})
	client := cloud.Client("object-store", "/swift/v1/AUTH_project")
	// Item calls use ResourceBase; the account-level list and bulk delete use Endpoint.
	client.ResourceBase = cloud.Server.URL + "/swift/v1/AUTH_project/"
	return containers.New(client)
}

func nativeContainerOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "containers" {
		t.Fatal("generated containers context", err, wrapped)
	}
}

var nativeContainerJSON = http.Header{"Content-Type": {"application/json; charset=utf-8"}}

func TestNativeContainerHeaderCallsAndEscaping(t *testing.T) {
	ctx := context.Background()
	var calls []nativeContainerCall
	api := nativeContainerAPI(t, &calls, func(req *http.Request) nativeContainerReply {
		if req.Method == http.MethodHead {
			return nativeContainerReply{204, http.Header{
				"X-Container-Bytes-Used":   {"10"},
				"X-Container-Object-Count": {"2"},
				"X-Container-Read":         {".r:*,.rlistings"},
				"X-Versions-Enabled":       {"True"},
				"X-Timestamp":              {"1760144523.12345"},
				"X-Storage-Policy":         {"gold"},
				"Date":                     {"Sun, 11 Oct 2026 01:02:03 GMT"},
			}, ""}
		}
		if req.Method == http.MethodDelete {
			return nativeContainerReply{204, nil, ""}
		}
		return nativeContainerReply{201, http.Header{"X-Trans-Id": {"tx"}}, ""}
	})
	created, err := api.Create(ctx, "my box?#", containers.CreateOpts{Metadata: map[string]string{"Color": "blue"}, ContainerRead: ".r:*", StoragePolicy: "gold", VersionsEnabled: true, DetectContentType: false}, containers.WithCreateHeader("X-Extra", "1"))
	if err != nil || created.TransID != "tx" {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "my box?#", containers.WithGetOptions(containers.GetOpts{Newest: true}))
	if err != nil || !(got.BytesUsed == 10 && got.ObjectCount == 2 && reflect.DeepEqual(got.Read, []string{".r:*", ".rlistings"}) && got.VersionsEnabled && got.StoragePolicy == "gold" && got.Timestamp == 1760144523.12345 && got.Date.Day() == 11) {
		t.Fatal(got, err)
	}
	// An absent ACL header splits into one empty entry.
	if !reflect.DeepEqual(got.Write, []string{""}) {
		t.Fatal(got.Write)
	}
	read, enabled := "", false
	if _, err := api.Update(ctx, "my box?#", containers.UpdateOpts{Metadata: map[string]string{"Color": "red"}, RemoveMetadata: []string{"Old"}, ContainerRead: &read, VersionsEnabled: &enabled, RemoveVersionsLocation: "x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Delete(ctx, "my box?#"); err != nil {
		t.Fatal(err)
	}
	path := "/swift/v1/AUTH_project/my%20box%3F%23"
	// Gophercloud adds Accept: application/json to every request.
	want := []nativeContainerCall{
		// A false bool header is omitted.
		{http.MethodPut, path, "", "", http.Header{"Accept": {"application/json"}, "X-Container-Meta-Color": {"blue"}, "X-Container-Read": {".r:*"}, "X-Storage-Policy": {"gold"}, "X-Versions-Enabled": {"true"}, "X-Extra": {"1"}}},
		{http.MethodHead, path, "", "", http.Header{"Accept": {"application/json"}, "X-Newest": {"true"}}},
		// Update pointers send explicit empty and false values.
		{http.MethodPost, path, "", "", http.Header{"Accept": {"application/json"}, "X-Container-Meta-Color": {"red"}, "X-Remove-Container-Meta-Old": {"remove"}, "X-Container-Read": {""}, "X-Versions-Enabled": {"false"}, "X-Remove-Versions-Location": {"x"}}},
		{http.MethodDelete, path, "", "", http.Header{"Accept": {"application/json"}}},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeContainerListMarkerPagingAndBulkDelete(t *testing.T) {
	ctx := context.Background()
	var calls []nativeContainerCall
	api := nativeContainerAPI(t, &calls, func(req *http.Request) nativeContainerReply {
		if req.Method == http.MethodPost {
			return nativeContainerReply{200, nativeContainerJSON, `{"Response Status":"200 OK","Response Body":"","Errors":[["/b","409 Conflict"]],"Number Deleted":1,"Number Not Found":1}`}
		}
		switch req.URL.Query().Get("marker") {
		case "":
			return nativeContainerReply{200, nativeContainerJSON, `[{"name":"a","count":1,"bytes":5},{"name":"b c","count":0,"bytes":0}]`}
		case "b c":
			return nativeContainerReply{200, nativeContainerJSON, `[{"name":"d","count":2,"bytes":9}]`}
		}
		return nativeContainerReply{200, nativeContainerJSON, `[]`}
	})
	var names []string
	for value, err := range api.List(ctx, containers.WithListOptions(containers.ListOpts{Prefix: "p", Limit: 2}), containers.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, fmt.Sprintf("%s/%d/%d", value.Name, value.Count, value.Bytes))
	}
	deleted, err := api.BulkDelete(ctx, []string{"a", "b c"})
	if err != nil || deleted.NumberDeleted != 1 || deleted.NumberNotFound != 1 || deleted.Errors[0][1] != "409 Conflict" {
		t.Fatal(deleted, err)
	}
	if !reflect.DeepEqual(names, []string{"a/1/5", "b c/0/0", "d/2/9"}) || len(calls) != 4 {
		t.Fatal(names, calls)
	}
	listHeader := http.Header{"Accept": {"application/json"}, "Content-Type": {"application/json"}}
	want := []nativeContainerCall{
		{http.MethodGet, "/swift/v1/AUTH_project/", "extra=1&limit=2&prefix=p", "", listHeader},
		// Marker paging repeats the request with the last name until an empty page.
		{http.MethodGet, "/swift/v1/AUTH_project/", "extra=1&limit=2&marker=b+c&prefix=p", "", listHeader},
		{http.MethodGet, "/swift/v1/AUTH_project/", "extra=1&limit=2&marker=d&prefix=p", "", listHeader},
		// Bulk delete posts path-escaped names, one per line.
		{http.MethodPost, "/swift/v1/AUTH_project/", "bulk-delete=true", "a\nb%20c\n", http.Header{"Accept": {"application/json"}, "Content-Type": {"text/plain"}}},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeContainerStatusesNamesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*containers.API) error
	}{
		{"Create", []int{201, 202, 204}, func(api *containers.API) error {
			_, err := api.Create(ctx, "c", containers.CreateOpts{})
			return err
		}},
		{"Get", []int{200, 204}, func(api *containers.API) error { _, err := api.Get(ctx, "c"); return err }},
		{"Update", []int{201, 202, 204}, func(api *containers.API) error {
			_, err := api.Update(ctx, "c", containers.UpdateOpts{})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *containers.API) error { _, err := api.Delete(ctx, "c"); return err }},
		{"BulkDelete", []int{200}, func(api *containers.API) error { _, err := api.BulkDelete(ctx, []string{"c"}); return err }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			accepted := false
			for _, ok := range call.accepted {
				accepted = accepted || ok == code
			}
			if accepted {
				continue
			}
			var calls []nativeContainerCall
			api := nativeContainerAPI(t, &calls, func(*http.Request) nativeContainerReply { return nativeContainerReply{code, nil, ""} })
			err := call.call(api)
			nativeContainerOperation(t, err, call.name)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
				t.Fatal(call.name, code, err)
			}
		}
	}
	t.Run("container names and header extensions", func(t *testing.T) {
		var calls []nativeContainerCall
		api := nativeContainerAPI(t, &calls, func(*http.Request) nativeContainerReply { return nativeContainerReply{204, nil, ""} })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"empty create": {"Create", func() error { _, err := api.Create(ctx, "", containers.CreateOpts{}); return err }()},
			"slash get":    {"Get", func() error { _, err := api.Get(ctx, "a/b"); return err }()},
			"slash delete": {"Delete", func() error { _, err := api.Delete(ctx, "a/b"); return err }()},
			"slash bulk":   {"BulkDelete", func() error { _, err := api.BulkDelete(ctx, []string{"ok", "a/b"}); return err }()},
			"core header": {"Create", func() error {
				_, err := api.Create(ctx, "c", containers.CreateOpts{}, containers.WithCreateHeader("X-Storage-Policy", "x"))
				return err
			}()},
			"update nil": {"Update", func() error { _, err := api.Update(ctx, "c", containers.UpdateOpts{}, nil); return err }()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeContainerOperation(t, check.err, check.operation)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
	t.Run("list status and empty page", func(t *testing.T) {
		for _, tc := range []nativeContainerReply{{404, nil, ""}, {200, nativeContainerJSON, `[]`}, {204, nil, ""}} {
			var calls []nativeContainerCall
			api := nativeContainerAPI(t, &calls, func(*http.Request) nativeContainerReply { return tc })
			var errs []error
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			switch {
			case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native):
			case tc.code == 200 && len(errs) == 0:
			// A bodyless 204 is an empty listing for this marker pager.
			case tc.code == 204 && len(errs) == 0:
			default:
				t.Fatal(tc.code, errs)
			}
		}
	})
	t.Run("versions flag decode", func(t *testing.T) {
		var calls []nativeContainerCall
		api := nativeContainerAPI(t, &calls, func(*http.Request) nativeContainerReply {
			return nativeContainerReply{204, http.Header{"X-Versions-Enabled": {"maybe"}}, ""}
		})
		_, err := api.Get(ctx, "c")
		nativeContainerOperation(t, err, "Get")
	})
}
