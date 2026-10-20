package accounts_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeAccountTransport func(*http.Request) (*http.Response, error)

func (transport nativeAccountTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeAccountCall struct {
	method, path string
	header       http.Header
}

// nativeAccountHeaders keeps only the Swift-specific request headers.
func nativeAccountHeaders(h http.Header) http.Header {
	out := http.Header{}
	for key, values := range h {
		if strings.HasPrefix(key, "X-Account-") || strings.HasPrefix(key, "X-Remove-") || key == "X-Newest" || key == "Content-Type" || key == "X-Detect-Content-Type" || key == "X-Extra" {
			out[key] = values
		}
	}
	return out
}

func nativeAccountAPI(t *testing.T, calls *[]nativeAccountCall, code int, header http.Header) *accounts.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeAccountTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, nativeAccountCall{req.Method, req.URL.Path, nativeAccountHeaders(req.Header)})
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader("")), Header: header}, nil
	})
	client := cloud.Client("object-store", "/swift/v1/AUTH_project")
	// The account calls use Endpoint itself, never ResourceBase.
	client.ResourceBase = cloud.Server.URL + "/unused/"
	return accounts.New(client)
}

func nativeAccountOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "accounts" {
		t.Fatal("generated accounts context", err, wrapped)
	}
}

func TestNativeAccountGetHeadsTheEndpointAndDecodesHeaders(t *testing.T) {
	ctx := context.Background()
	var calls []nativeAccountCall
	api := nativeAccountAPI(t, &calls, 204, http.Header{
		"X-Account-Bytes-Used":        {"1024"},
		"X-Account-Container-Count":   {"3"},
		"X-Account-Object-Count":      {"7"},
		"X-Account-Meta-Quota-Bytes":  {"4096"},
		"X-Account-Meta-Temp-Url-Key": {"secret"},
		"X-Trans-Id":                  {"tx1"},
		"Date":                        {"Sun, 11 Oct 2026 01:02:03 GMT"},
	})
	got, err := api.Get(ctx, accounts.WithGetOptions(accounts.GetOpts{Newest: true}), accounts.WithGetHeader("X-Extra", "1"))
	if err != nil || !(got.BytesUsed == 1024 && got.ContainerCount == 3 && got.ObjectCount == 7 && *got.QuotaBytes == 4096 && got.TempURLKey == "secret" && got.TransID == "tx1" && got.Date.Equal(time.Date(2026, 10, 11, 1, 2, 3, 0, time.UTC))) {
		t.Fatal(got, err)
	}
	// A false Newest flag sends no header and an absent quota stays nil.
	calls2 := []nativeAccountCall{}
	plain, err := nativeAccountAPI(t, &calls2, 204, http.Header{}).Get(ctx)
	if err != nil || plain.QuotaBytes != nil || !plain.Date.IsZero() {
		t.Fatal(plain, err)
	}
	// The request goes to the normalized endpoint, which ends with a slash.
	want := []nativeAccountCall{{http.MethodHead, "/swift/v1/AUTH_project/", http.Header{"X-Newest": {"true"}, "X-Extra": {"1"}}}}
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(calls2, []nativeAccountCall{{http.MethodHead, "/swift/v1/AUTH_project/", http.Header{}}}) {
		t.Fatalf("%+v %+v", calls, calls2)
	}
}

func TestNativeAccountUpdatePostsMetadataHeaders(t *testing.T) {
	ctx := context.Background()
	var calls []nativeAccountCall
	api := nativeAccountAPI(t, &calls, 204, http.Header{"X-Trans-Id": {"tx2"}, "Date": {"Sun, 11 Oct 2026 01:02:03 GMT"}})
	contentType, detect := "text/plain", false
	got, err := api.Update(ctx, accounts.UpdateOpts{Metadata: map[string]string{"Color": "blue"}, RemoveMetadata: []string{"Old"}, ContentType: &contentType, DetectContentType: &detect, TempURLKey: "k1", TempURLKey2: "k2"}, accounts.WithUpdateHeader("X-Extra", "1"))
	if err != nil || got.TransID != "tx2" || got.Date.Day() != 11 {
		t.Fatal(got, err)
	}
	want := []nativeAccountCall{{http.MethodPost, "/swift/v1/AUTH_project/", http.Header{
		"X-Account-Meta-Color":          {"blue"},
		"X-Remove-Account-Meta-Old":     {"remove"},
		"Content-Type":                  {"text/plain"},
		"X-Detect-Content-Type":         {"false"},
		"X-Account-Meta-Temp-Url-Key":   {"k1"},
		"X-Account-Meta-Temp-Url-Key-2": {"k2"},
		"X-Extra":                       {"1"},
	}}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeAccountStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, code := range []int{200, 201, 202, 404} {
		var calls []nativeAccountCall
		_, err := nativeAccountAPI(t, &calls, code, http.Header{}).Get(ctx)
		nativeAccountOperation(t, err, "Get")
		var native gophercloud.ErrUnexpectedResponseCode
		// Get accepts only 204.
		if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{204}) || len(calls) != 1 {
			t.Fatal(code, err)
		}
	}
	for _, code := range []int{200, 404} {
		var calls []nativeAccountCall
		_, err := nativeAccountAPI(t, &calls, code, http.Header{}).Update(ctx, accounts.UpdateOpts{})
		nativeAccountOperation(t, err, "Update")
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{201, 202, 204}) || len(calls) != 1 {
			t.Fatal(code, err)
		}
	}
	for name, header := range map[string]http.Header{
		"non-numeric count": {"X-Account-Container-Count": {"many"}},
		"non-RFC1123 date":  {"Date": {"2026-10-11T01:02:03Z"}},
	} {
		var calls []nativeAccountCall
		_, err := nativeAccountAPI(t, &calls, 204, header).Get(ctx)
		if err == nil {
			t.Fatal(name, "decoded")
		}
		nativeAccountOperation(t, err, "Get")
	}
	var calls []nativeAccountCall
	api := nativeAccountAPI(t, &calls, 204, http.Header{})
	for operation, err := range map[string]error{
		"Get": func() error { _, err := api.Get(ctx, accounts.WithGetHeader("X-Newest", "false")); return err }(),
		"Update": func() error {
			_, err := api.Update(ctx, accounts.UpdateOpts{}, accounts.WithUpdateHeader("bad header", "x"))
			return err
		}(),
	} {
		if err == nil {
			t.Fatal(operation, "accepted")
		}
		nativeAccountOperation(t, err, operation)
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}
