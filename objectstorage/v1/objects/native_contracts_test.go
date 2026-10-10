package objects_test

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeObjectTransport func(*http.Request) (*http.Response, error)

func (transport nativeObjectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeObjectCall struct {
	method, path, query, body string
	header                    http.Header
}

var nativeObjectHeaderNames = []string{"Etag", "Content-Type", "X-Object-Meta-Color", "X-Remove-Object-Meta-Old", "X-Newest", "Range", "If-Modified-Since", "If-Match",
	"Destination", "X-Delete-After", "X-Detect-Content-Type", "Content-Disposition", "X-Extra"}

func nativeObjectHeaders(h http.Header) http.Header {
	out := http.Header{}
	for _, key := range nativeObjectHeaderNames {
		if values, ok := h[key]; ok {
			out[key] = values
		}
	}
	return out
}

type nativeObjectReply struct {
	code   int
	header http.Header
	body   string
}

func nativeObjectAPI(t *testing.T, calls *[]nativeObjectCall, reply func(*http.Request) nativeObjectReply) *objects.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeObjectTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeObjectCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, nativeObjectHeaders(req.Header)})
		r := reply(req)
		if r.header == nil {
			r.header = http.Header{}
		}
		return &http.Response{StatusCode: r.code, Body: io.NopCloser(strings.NewReader(r.body)), Header: r.header}, nil
	})
	client := cloud.Client("object-store", "/swift/v1/AUTH_project")
	client.ResourceBase = cloud.Server.URL + "/swift/v1/AUTH_project/"
	return objects.New(client)
}

func nativeObjectOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "objects" {
		t.Fatal("generated objects context", err, wrapped)
	}
}

const nativeObjectPath = "/swift/v1/AUTH_project/my%20box/dir%2Ffile%3F"

func TestNativeObjectCreateUpdateDeleteHeadersAndEscaping(t *testing.T) {
	ctx := context.Background()
	var calls []nativeObjectCall
	api := nativeObjectAPI(t, &calls, func(req *http.Request) nativeObjectReply {
		if req.Method == http.MethodDelete || req.Method == http.MethodPost {
			return nativeObjectReply{202, nil, ""}
		}
		return nativeObjectReply{201, http.Header{"Etag": {"x"}, "Last-Modified": {"Sun, 11 Oct 2026 01:02:03 GMT"}}, ""}
	})
	sum := fmt.Sprintf("%x", md5.Sum([]byte("payload")))
	created, err := api.Create(ctx, "my box", "dir/file?", objects.CreateOpts{Content: strings.NewReader("payload"), ContentType: "text/plain", Metadata: map[string]string{"Color": "blue"}, MultipartManifest: "put"}, objects.WithCreateHeader("X-Extra", "1"))
	if err != nil || created.ETag != "x" || created.LastModified.Day() != 11 {
		t.Fatal(created, err)
	}
	// An explicit ETag is kept, and NoETag does not remove it because it deletes the lowercase key.
	if _, err := api.Create(ctx, "my box", "dir/file?", objects.CreateOpts{Content: strings.NewReader("payload"), ETag: "given", NoETag: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Create(ctx, "my box", "dir/file?", objects.CreateOpts{Content: strings.NewReader("payload"), NoETag: true}); err != nil {
		t.Fatal(err)
	}
	disposition, detect, after := "", false, int64(60)
	if _, err := api.Update(ctx, "my box", "dir/file?", objects.UpdateOpts{Metadata: map[string]string{"Color": "red"}, RemoveMetadata: []string{"Old"}, ContentDisposition: &disposition, DetectContentType: &detect, DeleteAfter: &after}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Delete(ctx, "my box", "dir/file?", objects.WithDeleteOptions(objects.DeleteOpts{MultipartManifest: "delete", ObjectVersionID: "v1"})); err != nil {
		t.Fatal(err)
	}
	want := []nativeObjectCall{
		// The object name is path-escaped, so its slash becomes %2F; the MD5 of the content is sent as ETag.
		{http.MethodPut, nativeObjectPath, "multipart-manifest=put", "payload", http.Header{"Etag": {sum}, "Content-Type": {"text/plain"}, "X-Object-Meta-Color": {"blue"}, "X-Extra": {"1"}}},
		{http.MethodPut, nativeObjectPath, "", "payload", http.Header{"Etag": {"given"}}},
		{http.MethodPut, nativeObjectPath, "", "payload", http.Header{}},
		{http.MethodPost, nativeObjectPath, "", "", http.Header{"X-Object-Meta-Color": {"red"}, "X-Remove-Object-Meta-Old": {"remove"}, "Content-Disposition": {""}, "X-Detect-Content-Type": {"false"}, "X-Delete-After": {"60"}}},
		{http.MethodDelete, nativeObjectPath, "multipart-manifest=delete&version-id=v1", "", http.Header{}},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeObjectGetDownloadAndCopy(t *testing.T) {
	ctx := context.Background()
	var calls []nativeObjectCall
	api := nativeObjectAPI(t, &calls, func(req *http.Request) nativeObjectReply {
		switch req.Method {
		case http.MethodHead:
			return nativeObjectReply{200, http.Header{"X-Static-Large-Object": {"True"}, "X-Delete-At": {"1760144523"}, "Content-Length": {"7"}}, ""}
		case "COPY":
			return nativeObjectReply{201, http.Header{"X-Copied-From": {"my%20box/dir/file%3F"}}, ""}
		}
		return nativeObjectReply{206, http.Header{"Content-Length": {"3"}, "Etag": {"e"}}, "pay"}
	})
	got, err := api.Get(ctx, "my box", "dir/file?", objects.WithGetOptions(objects.GetOpts{Newest: true, ObjectVersionID: "v1"}))
	if err != nil || !(got.StaticLargeObject && got.DeleteAt.Unix() == 1760144523 && got.ContentLength == 7) {
		t.Fatal(got, err)
	}
	since := time.Date(2026, 10, 11, 1, 2, 3, 0, time.UTC)
	download, err := api.Download(ctx, "my box", "dir/file?", objects.WithDownloadOptions(objects.DownloadOpts{Range: "bytes=0-2", IfModifiedSince: since, IfMatch: "e"}))
	if err != nil {
		t.Fatal(err)
	}
	content, _ := io.ReadAll(download)
	if string(content) != "pay" || download.Header.ETag != "e" || download.Close() != nil {
		t.Fatal(string(content), download.Header)
	}
	if _, err := api.Copy(ctx, "my box", "dir/file?", objects.CopyOpts{Destination: "/other box/a/b?", ObjectVersionID: "v1"}); err != nil {
		t.Fatal(err)
	}
	want := []nativeObjectCall{
		{http.MethodHead, nativeObjectPath, "version-id=v1", "", http.Header{"X-Newest": {"true"}}},
		// Conditional times are sent in RFC1123.
		{http.MethodGet, nativeObjectPath, "", "", http.Header{"Range": {"bytes=0-2"}, "If-Modified-Since": {"Sun, 11 Oct 2026 01:02:03 UTC"}, "If-Match": {"e"}}},
		// COPY escapes each Destination segment; the object part keeps its slash because only two splits are made.
		{"COPY", nativeObjectPath, "version-id=v1", "", http.Header{"Destination": {"/other%20box/a%2Fb%3F"}}},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeObjectListAndBulkDelete(t *testing.T) {
	ctx := context.Background()
	var calls []nativeObjectCall
	api := nativeObjectAPI(t, &calls, func(req *http.Request) nativeObjectReply {
		if req.Method == http.MethodPost {
			return nativeObjectReply{200, http.Header{"Content-Type": {"application/json"}}, `{"Number Deleted":2,"Number Not Found":0,"Errors":[]}`}
		}
		json := http.Header{"Content-Type": {"application/json"}}
		switch req.URL.Query().Get("marker") {
		case "":
			return nativeObjectReply{200, json, `[{"name":"a","bytes":1,"hash":"h","content_type":"text/plain","last_modified":"2026-10-11T01:02:03.123456"},{"subdir":"dir/"}]`}
		case "dir/":
			return nativeObjectReply{200, json, `[{"name":"z","bytes":2}]`}
		}
		return nativeObjectReply{200, json, `[]`}
	})
	var rows []string
	for value, err := range api.List(ctx, "my box", objects.WithListOptions(objects.ListOpts{Delimiter: "/", Versions: true}), objects.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value.Name+"|"+value.Subdir)
	}
	deleted, err := api.BulkDelete(ctx, "my box", []string{"a", "dir/b c"})
	if err != nil || deleted.NumberDeleted != 2 {
		t.Fatal(deleted, err)
	}
	if !reflect.DeepEqual(rows, []string{"a|", "|dir/", "z|"}) || len(calls) != 4 {
		t.Fatal(rows, calls)
	}
	list := http.Header{"Content-Type": {"application/json"}}
	want := []nativeObjectCall{
		{http.MethodGet, "/swift/v1/AUTH_project/my%20box", "delimiter=%2F&extra=1&versions=true", "", list},
		// A subdir row has no name, so its subdir becomes the next marker.
		{http.MethodGet, "/swift/v1/AUTH_project/my%20box", "delimiter=%2F&extra=1&marker=dir%2F&versions=true", "", list},
		{http.MethodGet, "/swift/v1/AUTH_project/my%20box", "delimiter=%2F&extra=1&marker=z&versions=true", "", list},
		{http.MethodPost, "/swift/v1/AUTH_project/", "bulk-delete=true", "my%20box/a\nmy%20box/dir%2Fb%20c\n", http.Header{"Content-Type": {"text/plain"}}},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeObjectCreateTempURL(t *testing.T) {
	ctx := context.Background()
	at := time.Unix(1760144523, 0)
	sign := func(newHash func() interface {
		Write([]byte) (int, error)
		Sum([]byte) []byte
	}, method string, expiry int64) string {
		h := newHash()
		h.Write([]byte(fmt.Sprintf("%s\n%d\n/v1/AUTH_project/my box/a b", method, expiry)))
		return fmt.Sprintf("%x", h.Sum(nil))
	}
	var calls []nativeObjectCall
	api := nativeObjectAPI(t, &calls, func(req *http.Request) nativeObjectReply {
		if req.URL.Path == "/swift/v1/AUTH_project/my box" {
			return nativeObjectReply{204, http.Header{}, ""}
		}
		return nativeObjectReply{204, http.Header{"X-Account-Meta-Temp-Url-Key": {"acct"}}, ""}
	})
	// An explicit key needs no HTTP; the signature uses the unescaped object path after /v1/.
	signed, err := api.CreateTempURL(ctx, "my box", "a b", objects.CreateTempURLOpts{Method: objects.HTTPMethod("GET"), TTL: 60, Timestamp: at, TempURLKey: "secret"})
	want := sign(func() interface {
		Write([]byte) (int, error)
		Sum([]byte) []byte
	} {
		return hmac.New(sha1.New, []byte("secret"))
	}, "GET", at.Unix()+60)
	if err != nil || !strings.HasSuffix(signed, "/swift/v1/AUTH_project/my%20box/a%20b?temp_url_sig="+want+"&temp_url_expires=1760144583") || len(calls) != 0 {
		t.Fatal(signed, err, calls)
	}
	sha, err := api.CreateTempURL(ctx, "my box", "a b", objects.CreateTempURLOpts{Method: objects.HTTPMethod("PUT"), TTL: 0, Timestamp: at, Digest: "sha256"})
	want = sign(func() interface {
		Write([]byte) (int, error)
		Sum([]byte) []byte
	} {
		return hmac.New(sha256.New, []byte("acct"))
	}, "PUT", at.Unix())
	if err != nil || !strings.Contains(sha, "temp_url_sig="+want) {
		t.Fatal(sha, err)
	}
	// Without an explicit key the container is checked first, then the account.
	wantCalls := []nativeObjectCall{{http.MethodHead, "/swift/v1/AUTH_project/my%20box", "", "", http.Header{}}, {http.MethodHead, "/swift/v1/AUTH_project/", "", "", http.Header{}}}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("%+v", calls)
	}
	t.Run("errors are native, not wrapped", func(t *testing.T) {
		var calls []nativeObjectCall
		api := nativeObjectAPI(t, &calls, func(*http.Request) nativeObjectReply { return nativeObjectReply{204, nil, ""} })
		_, err := api.CreateTempURL(ctx, "c", "o", objects.CreateTempURLOpts{Timestamp: at})
		var missing objects.ErrTempURLKeyNotFound
		var wrapped *resource.OperationError
		if !errors.As(err, &missing) || errors.As(err, &wrapped) {
			t.Fatal(err)
		}
		_, err = api.CreateTempURL(ctx, "c", "o", objects.CreateTempURLOpts{TempURLKey: "k", Digest: "md5"})
		var digest objects.ErrTempURLDigestNotValid
		if !errors.As(err, &digest) || digest.Digest != "md5" {
			t.Fatal(err)
		}
		if _, err := api.CreateTempURL(ctx, "c", "o", objects.CreateTempURLOpts{TempURLKey: "k", Split: "/v9/"}); err == nil || !strings.Contains(err.Error(), "URL prefix") {
			t.Fatal(err)
		}
		_, err = api.CreateTempURL(ctx, "c", "o", objects.CreateTempURLOpts{TempURLKey: "k"}, nil)
		nativeObjectOperation(t, err, "CreateTempURL")
	})
}

func TestNativeObjectStatusesNamesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*objects.API) error
	}{
		{"Create", []int{201, 202}, func(api *objects.API) error {
			_, err := api.Create(ctx, "c", "o", objects.CreateOpts{Content: strings.NewReader("")})
			return err
		}},
		{"Get", []int{200, 204}, func(api *objects.API) error { _, err := api.Get(ctx, "c", "o"); return err }},
		{"Download", []int{200, 206, 304}, func(api *objects.API) error { _, err := api.Download(ctx, "c", "o"); return err }},
		{"Update", []int{201, 202}, func(api *objects.API) error {
			_, err := api.Update(ctx, "c", "o", objects.UpdateOpts{})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *objects.API) error { _, err := api.Delete(ctx, "c", "o"); return err }},
		{"Copy", []int{201}, func(api *objects.API) error {
			_, err := api.Copy(ctx, "c", "o", objects.CopyOpts{Destination: "/c/o2"})
			return err
		}},
		{"BulkDelete", []int{200}, func(api *objects.API) error { _, err := api.BulkDelete(ctx, "c", []string{"o"}); return err }},
	} {
		for _, code := range []int{200, 201, 202, 204, 206, 404} {
			accepted := false
			for _, ok := range call.accepted {
				accepted = accepted || ok == code
			}
			if accepted {
				continue
			}
			var calls []nativeObjectCall
			api := nativeObjectAPI(t, &calls, func(*http.Request) nativeObjectReply { return nativeObjectReply{code, nil, ""} })
			err := call.call(api)
			nativeObjectOperation(t, err, call.name)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
				t.Fatal(call.name, code, err)
			}
		}
	}
	t.Run("names, destinations and extensions", func(t *testing.T) {
		var calls []nativeObjectCall
		api := nativeObjectAPI(t, &calls, func(*http.Request) nativeObjectReply { return nativeObjectReply{201, nil, ""} })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"empty object": {"Get", func() error { _, err := api.Get(ctx, "c", ""); return err }()},
			"slash container": {"Create", func() error {
				_, err := api.Create(ctx, "a/b", "o", objects.CreateOpts{Content: strings.NewReader("")})
				return err
			}()},
			"destination missing": {"Copy", func() error { _, err := api.Copy(ctx, "c", "o", objects.CopyOpts{}); return err }()},
			"destination relative": {"Copy", func() error {
				_, err := api.Copy(ctx, "c", "o", objects.CopyOpts{Destination: "c/o2/x"})
				return err
			}()},
			"destination short": {"Copy", func() error { _, err := api.Copy(ctx, "c", "o", objects.CopyOpts{Destination: "/c"}); return err }()},
			"bulk empty object": {"BulkDelete", func() error { _, err := api.BulkDelete(ctx, "c", []string{"o", ""}); return err }()},
			"core header": {"Create", func() error {
				_, err := api.Create(ctx, "c", "o", objects.CreateOpts{Content: strings.NewReader("")}, objects.WithCreateHeader("Content-Type", "x"))
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeObjectOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, "a/b") {
			if err == nil {
				t.Fatal("slash list accepted")
			}
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}

func TestNativeObjectCreateNilContentPanics(t *testing.T) {
	var calls []nativeObjectCall
	api := nativeObjectAPI(t, &calls, func(*http.Request) nativeObjectReply { return nativeObjectReply{201, nil, ""} })
	defer func() {
		// The native ETag computation reads a nil Content reader before any request.
		if recover() == nil || len(calls) != 0 {
			t.Fatal("no panic", calls)
		}
	}()
	_, _ = api.Create(context.Background(), "c", "o", objects.CreateOpts{})
}
