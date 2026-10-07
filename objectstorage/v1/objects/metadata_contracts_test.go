package objects_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	swift "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1"
)

const objectMetadataBase = "https://swift.invalid/reverse/a%20b/v1/AUTH_account/"
const objectMetadataContainer = "한글 space:%2F?#"
const objectMetadataKey = "folder/ 한글:%2F?# /leaf"

var objectMetadataEndpoint = objectMetadataBase + url.PathEscape(objectMetadataContainer) + "/" + url.PathEscape(objectMetadataKey)

type objectMetadataTransport func(*http.Request) (*http.Response, error)

func (f objectMetadataTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := f(r)
	if response != nil && response.Request == nil {
		response.Request = r
	}
	return response, err
}

type objectMetadataBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *objectMetadataBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type objectMetadataReader func([]byte) (int, error)

func (f objectMetadataReader) Read(p []byte) (int, error) { return f(p) }
func objectMetadataClient(f objectMetadataTransport) *gophercloud.ServiceClient {
	p := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: f}}
	p.UseTokenLock()
	p.SetToken("original-token")
	return &gophercloud.ServiceClient{ProviderClient: p, Type: "object-store", Endpoint: objectMetadataBase}
}
func objectMetadataWire(status int, body *objectMetadataBody) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}, "X-Trans-Id": {"actual-object"}}, Body: body}
}
func objectMetadataResponse(status int, body string) *http.Response {
	return objectMetadataWire(status, &objectMetadataBody{Reader: strings.NewReader(body)})
}
func objectMetadataProof(t *testing.T, err error, status int, body string) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Trans-Id") != "actual-object" {
		t.Fatalf("proof=%+v err=%v", proof, err)
	}
	return proof
}
func objectMetadataBodyless(t *testing.T, r *http.Request, own bool) {
	t.Helper()
	expected := objectMetadataEndpoint
	if own {
		expected += "?symlink=get"
	}
	if r.URL.String() != expected || r.Header.Get("X-Auth-Token") != "original-token" {
		t.Errorf("request=%s header=%v", r.URL, r.Header)
	}
	if r.Body != nil {
		b, err := io.ReadAll(r.Body)
		if err != nil || len(b) != 0 {
			t.Errorf("request body=%q err=%v", b, err)
		}
	}
	for key := range r.Header {
		if strings.HasPrefix(strings.ToLower(key), "x-container-meta-") || strings.HasPrefix(strings.ToLower(key), "x-account-meta-") {
			t.Errorf("foreign metadata prefix=%s", key)
		}
	}
}
func objectMetadataCustom(h http.Header) map[string]string {
	values := map[string]string{}
	for key, value := range h {
		if strings.HasPrefix(strings.ToLower(key), "x-object-meta-") {
			values[strings.ToLower(key[len("x-object-meta-"):])] = value[0]
		}
	}
	return values
}

func TestObjectMetadataContractsWireAndIdentity(t *testing.T) {
	t.Run("literal whole object keys and real query-free getter", func(t *testing.T) {
		for _, tc := range []struct{ base, container, key string }{
			{objectMetadataBase, objectMetadataContainer, objectMetadataKey},
			{"https://swift.invalid/p%2Fq/v1/AUTH_other/", " container ", "/leading//trailing/"},
			{objectMetadataBase, ".hidden", " space%2F?#: /inside/.hidden"},
		} {
			calls := 0
			expected := tc.base + url.PathEscape(tc.container) + "/" + url.PathEscape(tc.key)
			parsed, _ := url.Parse(expected)
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "HEAD" || r.URL.String() != expected || r.URL.EscapedPath() != parsed.EscapedPath() || r.URL.RequestURI() != parsed.RequestURI() || r.URL.RawQuery != "" || r.Body != nil {
					t.Errorf("literal request=%s %s", r.Method, r.URL)
				}
				response := objectMetadataResponse(200, "")
				response.Header.Set("X-Object-Meta-Book", "한글\t literal %2F ")
				return response, nil
			})
			client.Endpoint = "https://swift.invalid/account-endpoint"
			client.ResourceBase = tc.base
			result, err := objects.New(client).GetMetadata(context.Background(), tc.container, tc.key)
			if err != nil || result == nil || result.StatusCode != 200 || result.Metadata.Values["book"] != "한글\t literal %2F " || calls != 1 {
				t.Fatalf("result=%+v err=%v HTTP=%d", result, err, calls)
			}
		}
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			expected := "/v1/AUTH_a/" + url.PathEscape(objectMetadataContainer) + "/" + url.PathEscape(objectMetadataKey)
			if r.Method != "HEAD" || r.URL.EscapedPath() != expected || r.RequestURI != expected || r.URL.RawQuery != "" || r.Header.Get("X-Newest") != "false" || r.Header.Get("X-Auth-Token") != "wire-token" {
				t.Errorf("wire=%s %s headers=%v", r.Method, r.RequestURI, r.Header)
			}
			w.Header().Set("X-Object-Meta-Empty", "")
			w.Header().Set("Content-Length", "0")
			w.WriteHeader(200)
		}))
		defer server.Close()
		provider := &gophercloud.ProviderClient{HTTPClient: *server.Client()}
		provider.UseTokenLock()
		provider.SetToken("wire-token")
		client := &gophercloud.ServiceClient{ProviderClient: provider, Type: "object-store", Endpoint: server.URL + "/v1/AUTH_a/"}
		result, err := objects.New(client).GetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, objects.WithGetMetadataNewest(false))
		if err != nil || result == nil || result.Metadata.ContentLength == nil || *result.Metadata.ContentLength != 0 || result.Metadata.Values["empty"] != "" || calls.Load() != 1 {
			t.Fatalf("real HEAD=%+v err=%v HTTP=%d", result, err, calls.Load())
		}
	})
	t.Run("fresh full merge nine mutable fields and write precedence", func(t *testing.T) {
		observed := map[string]string{"Content-Type": "observed/type", "Content-Encoding": "gzip", "Content-Disposition": "inline", "X-Delete-At": "not-an-epoch", "X-Object-Manifest": "container/prefix", "Cache-Control": "max-age=20", "Content-Language": "ko", "Expires": "not-a-date", "X-Robots-Tag": "noindex"}
		input := map[string]string{"Book": "new", "Empty": "", "read_ACL": "literal-custom"}
		calls := 0
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			objectMetadataBodyless(t, r, r.Method == "HEAD")
			if r.Method == "HEAD" {
				if r.Header.Get("X-Only-Write") != "" || r.Header.Get("X-Source") != "captured" || r.Header.Get("Content-Type") != "source/type" {
					t.Errorf("read options leaked=%v", r.Header)
				}
				response := objectMetadataResponse(200, "")
				for k, v := range observed {
					response.Header.Set(k, v)
				}
				response.Header.Set("X-Object-Meta-Book", "old")
				response.Header.Set("X-Object-Meta-Keep", "preserved")
				for _, k := range []string{"ETag", "Content-Length", "Last-Modified", "Date", "X-Timestamp", "Content-Location", "X-Static-Large-Object", "X-Unknown-System"} {
					response.Header.Set(k, "readonly")
				}
				response.Header.Set("Content-Length", "123")
				return response, nil
			}
			if r.Method != "POST" || r.Header.Get("X-Only-Write") != "option" || r.Header.Get("Content-Type") != "option/type" || r.Header.Get("Content-Encoding") != "source-encoding" || r.Header.Get("Content-Disposition") != "" {
				t.Errorf("write precedence=%v", r.Header)
			}
			for k, v := range observed {
				if k == "Content-Type" || k == "Content-Encoding" || k == "Content-Disposition" {
					continue
				}
				if r.Header.Get(k) != v {
					t.Errorf("lost mutable %s=%q", k, r.Header.Get(k))
				}
			}
			for _, k := range []string{"ETag", "Content-Length", "Last-Modified", "Date", "X-Timestamp", "Content-Location", "X-Static-Large-Object", "X-Unknown-System"} {
				if _, ok := r.Header[http.CanonicalHeaderKey(k)]; ok {
					t.Errorf("readonly copied=%s", k)
				}
			}
			want := map[string]string{"book": "new", "keep": "preserved", "empty": "", "read_acl": "literal-custom"}
			if !reflect.DeepEqual(objectMetadataCustom(r.Header), want) {
				t.Errorf("full replacement=%v", objectMetadataCustom(r.Header))
			}
			return objectMetadataResponse(202, "opaque acknowledgement"), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "captured", "Content-Type": "source/type", "Content-Encoding": "source-encoding"}
		source := client.MoreHeaders
		result, err := objects.New(client).SetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, input, objects.WithMetadataHeaders(map[string]string{"Content-Type": "option/type", "Content-Disposition": "", "X-Only-Write": "option"}))
		if err != nil || result == nil || result.Before.Metadata.Values["book"] != "old" || result.Acknowledgement == nil || result.Acknowledgement.StatusCode != 202 || string(result.Acknowledgement.Body) != "opaque acknowledgement" || calls != 2 || client.MoreHeaders["Content-Type"] != "source/type" {
			t.Fatalf("result=%+v err=%v HTTP=%d source=%v", result, err, calls, client.MoreHeaders)
		}
		source["X-Identity"] = "same map"
		if client.MoreHeaders["X-Identity"] != "same map" {
			t.Fatal("original source header storage replaced")
		}
	})
	t.Run("delete subtracts while nil empty absent always write", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			set   bool
			input map[string]string
			keys  []string
			want  map[string]string
		}{
			{name: "delete selected", keys: []string{"BOOK"}, want: map[string]string{"keep": "yes", "empty": ""}},
			{name: "delete absent", keys: []string{"absent"}, want: map[string]string{"book": "old", "keep": "yes", "empty": ""}},
			{name: "delete nil", want: map[string]string{"book": "old", "keep": "yes", "empty": ""}},
			{name: "delete empty", keys: []string{}, want: map[string]string{"book": "old", "keep": "yes", "empty": ""}},
			{name: "set nil", set: true, want: map[string]string{"book": "old", "keep": "yes", "empty": ""}},
			{name: "set empty", set: true, input: map[string]string{}, want: map[string]string{"book": "old", "keep": "yes", "empty": ""}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				calls := 0
				client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
					calls++
					objectMetadataBodyless(t, r, r.Method == "HEAD")
					if r.Method == "HEAD" {
						response := objectMetadataResponse(200, "")
						response.Header.Set("X-Object-Meta-Book", "old")
						response.Header.Set("X-Object-Meta-Keep", "yes")
						response.Header.Set("X-Object-Meta-Empty", "")
						return response, nil
					}
					if r.Method != "POST" || !reflect.DeepEqual(objectMetadataCustom(r.Header), tc.want) {
						t.Errorf("delete payload=%s %v", r.Method, r.Header)
					}
					for k := range r.Header {
						if strings.HasPrefix(strings.ToLower(k), "x-remove-object-meta-") {
							t.Errorf("native removal alias used=%s", k)
						}
					}
					return objectMetadataResponse(202, ""), nil
				})
				api := objects.New(client)
				var result *objects.MetadataChangeResult
				var err error
				if tc.set {
					result, err = api.SetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, tc.input)
				} else {
					result, err = api.DeleteMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, tc.keys)
				}
				if err != nil || result == nil || result.Before == nil || result.Acknowledgement == nil || calls != 2 {
					t.Fatalf("result=%+v err=%v HTTP=%d", result, err, calls)
				}
			})
		}
	})
	t.Run("existing native Get Update and object scope remain compatible", func(t *testing.T) {
		for _, status := range []int{200, 204} {
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				if r.Method != "HEAD" || r.URL.RawQuery != "version-id=literal%2Fversion" || r.Header.Get("X-Newest") != "true" {
					t.Errorf("native getter=%s %s %v", r.Method, r.URL, r.Header)
				}
				response := objectMetadataResponse(status, "")
				response.Header.Set("Date", "Fri, 17 Jan 2014 16:09:56 GMT")
				return response, nil
			})
			value, err := objects.New(client).Get(context.Background(), "container", "folder/key", objects.WithGetOptions(objects.GetOpts{Newest: true, ObjectVersionID: "literal/version"}))
			if err != nil || value == nil {
				t.Fatalf("native Get%d=%+v err=%v", status, value, err)
			}
		}
		for _, status := range []int{201, 202, 204} {
			calls := 0
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "POST" || r.URL.RawQuery != "" || r.Header.Get("X-Object-Meta-Only") != "native" || r.Header.Get("X-Remove-Object-Meta-Gone") != "remove" {
					t.Errorf("native update=%s %s %v", r.Method, r.URL, r.Header)
				}
				return objectMetadataResponse(status, ""), nil
			})
			value, err := objects.New(client).Update(context.Background(), "container", "folder/key", objects.UpdateOpts{Metadata: map[string]string{"Only": "native"}, RemoveMetadata: []string{"Gone"}})
			if calls != 1 || (status == 204 && err == nil) || (status != 204 && (err != nil || value == nil)) {
				t.Fatalf("native Update%d=%+v err=%v HTTP=%d", status, value, err, calls)
			}
		}
		calls := 0
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method != "HEAD" || r.URL.RawQuery != "" {
				t.Errorf("scope discovery=%s %s", r.Method, r.URL)
			}
			response := objectMetadataResponse(204, "")
			response.Header.Set("Date", "Fri, 17 Jan 2014 16:09:56 GMT")
			response.Header.Set("Content-Length", "7")
			response.Header.Set("X-Object-Meta-Native", "case-preserved")
			return response, nil
		})
		scope, err := objects.New(client).InContainer(context.Background(), resource.ID("container"))
		if err != nil || calls != 0 {
			t.Fatalf("scope=%+v err=%v HTTP=%d", scope, err, calls)
		}
		value, err := scope.Get(context.Background(), "folder/key")
		if err != nil || value == nil || value.Name != "folder/key" || value.Container != "container" || value.Details.ContentLength != 7 || value.Metadata["Native"] != "case-preserved" || calls != 1 {
			t.Fatalf("old scope=%+v err=%v HTTP=%d", value, err, calls)
		}
	})
}

func TestObjectMetadataContractsPreflightAndOptions(t *testing.T) {
	t.Run("context source and literal name preflight before callbacks", func(t *testing.T) {
		calls, callbacks := 0, 0
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
		api := objects.New(client)
		result, err := api.GetMetadata(nil, objectMetadataContainer, objectMetadataKey, func(*objects.GetMetadataOpts) error { callbacks++; return nil })
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 {
			t.Fatalf("nil context result=%+v err=%v HTTP=%d callback=%d", result, err, calls, callbacks)
		}
		var nilAPI *objects.API
		result, err = nilAPI.GetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil API=%+v err=%v", result, err)
		}
		canceled, cancel := context.WithCancelCause(context.Background())
		cause := errors.New("preflight cancellation")
		cancel(cause)
		if result, err = api.GetMetadata(canceled, objectMetadataContainer, objectMetadataKey); result != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls != 0 {
			t.Fatalf("canceled=%+v err=%v HTTP=%d", result, err, calls)
		}
		for _, tc := range []struct{ container, key string }{
			{"", objectMetadataKey}, {"with/slash", objectMetadataKey}, {".", objectMetadataKey}, {"..", objectMetadataKey}, {"bad\\container", objectMetadataKey}, {"bad\x7f", objectMetadataKey}, {string([]byte{0xff}), objectMetadataKey},
			{objectMetadataContainer, ""}, {objectMetadataContainer, "."}, {objectMetadataContainer, "../key"}, {objectMetadataContainer, "key/./leaf"}, {objectMetadataContainer, "key/.."}, {objectMetadataContainer, "bad\\key"}, {objectMetadataContainer, "bad\nkey"}, {objectMetadataContainer, string([]byte{0xff})},
		} {
			result, err = api.GetMetadata(context.Background(), tc.container, tc.key, func(*objects.GetMetadataOpts) error { callbacks++; return nil })
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatalf("identity=%+v result=%+v err=%v HTTP=%d", tc, result, err, calls)
			}
			if tc.container == "" {
				var native swift.ErrEmptyContainerName
				if !errors.As(err, &native) {
					t.Fatalf("native empty container cause=%v", err)
				}
			}
			if tc.container == "with/slash" {
				var native swift.ErrInvalidContainerName
				if !errors.As(err, &native) {
					t.Fatalf("native invalid container cause=%v", err)
				}
			}
			if tc.key == "" {
				var native swift.ErrEmptyObjectName
				if !errors.As(err, &native) {
					t.Fatalf("native empty object cause=%v", err)
				}
			}
		}
		for _, tc := range []struct{ endpoint, base string }{
			{"https://swift.invalid/a?query=1", ""}, {"https://swift.invalid/a#fragment", ""}, {"ftp://swift.invalid/a/", ""}, {"https://user@swift.invalid/a/", ""}, {"https://swift.invalid/a", ""},
			{objectMetadataBase, "https://other.invalid/a/"}, {objectMetadataBase, "https://swift.invalid/a"}, {objectMetadataBase, "https://swift.invalid/a/?"}, {objectMetadataBase, "https://swift.invalid/a/#fragment"}, {"https://swift.invalid/a?x=1", objectMetadataBase},
		} {
			local := objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
			local.Endpoint, local.ResourceBase = tc.endpoint, tc.base
			callbacks = 0
			result, err = objects.New(local).GetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, func(*objects.GetMetadataOpts) error { callbacks++; return nil })
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 {
				t.Fatalf("source=%+v result=%+v err=%v HTTP=%d callback=%d", tc, result, err, calls, callbacks)
			}
		}
	})
	t.Run("suffix values aliases and all reserved namespaces", func(t *testing.T) {
		calls := 0
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
		api := objects.New(client)
		for _, input := range []map[string]string{
			{"": "x"}, {"X-Object-Meta-Book": "x"}, {"x-object-meta-book": "x"}, {"Book": "a", "book": "b"}, {"bad key": "x"}, {"K": "x"}, {"Book": "line\nvalue"}, {"Book": "bad\x7f"}, {"Book": string([]byte{0xff})},
		} {
			result, err := api.SetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, input)
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatalf("input=%v result=%+v err=%v HTTP=%d", input, result, err, calls)
			}
		}
		for _, keys := range [][]string{{""}, {"Book", "book"}, {"X-Object-Meta-Book"}, {"with/slash"}, {"K"}} {
			result, err := api.DeleteMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, keys)
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatalf("keys=%v result=%+v err=%v HTTP=%d", keys, result, err, calls)
			}
		}
		reserved := []string{"Authorization", "X-Auth-Token", "Host", "Content-Length", "Transfer-Encoding", "Connection", "Proxy-Connection", "Proxy-Authorization", "Upgrade", "Trailer", "TE", "X-Newest", "X-Object-Meta-Book", "X-Remove-Object-Meta-Book", "X-Symlink-Target", "X-Object-Sysmeta-Private", "X-Object-Transient-Sysmeta-Private"}
		for _, key := range reserved {
			client.MoreHeaders = nil
			if result, err := api.GetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, objects.WithGetMetadataHeader(strings.ToLower(key), "owned")); result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatalf("get reserved=%s result=%+v err=%v", key, result, err)
			}
			if result, err := api.SetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, nil, objects.WithMetadataHeader(strings.ToLower(key), "owned")); result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatalf("write reserved=%s result=%+v err=%v", key, result, err)
			}
			client.MoreHeaders = map[string]string{strings.ToLower(key): "owned"}
			if result, err := api.DeleteMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, nil); result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatalf("source reserved=%s result=%+v err=%v", key, result, err)
			}
		}
		client.MoreHeaders = nil
		for _, headers := range []map[string]string{{"X-Trace": "one", "x-trace": "two"}, {"X-K": "x"}, {"bad key": "x"}, {"X-Trace": "bad\rvalue"}} {
			if result, err := api.GetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, objects.WithGetMetadataHeaders(headers)); result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatalf("headers=%v result=%+v err=%v", headers, result, err)
			}
		}
		if result, err := api.SetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, nil, nil); result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatalf("nil callback=%+v err=%v", result, err)
		}
	})
	t.Run("full replacement factory snapshots and canonical last header", func(t *testing.T) {
		newest := false
		headers := map[string]string{"x-call": "old"}
		full := objects.WithGetMetadataOpts(objects.GetMetadataOpts{Headers: headers, Newest: &newest})
		headers["x-call"] = "caller-mutated"
		newest = true
		extra := map[string]string{"X-Extra": "factory"}
		plural := objects.WithGetMetadataHeaders(extra)
		extra["X-Extra"] = "caller-mutated"
		calls := 0
		var retained *objects.GetMetadataOpts
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("X-Call") != "new" || r.Header.Get("X-Extra") != "factory" || r.Header.Get("X-Newest") != "false" || r.Header.Get("X-Discarded") != "" {
				t.Errorf("owned options=%v", r.Header)
			}
			return objectMetadataResponse(200, ""), nil
		})
		options := []objects.GetMetadataOption{objects.WithGetMetadataHeader("X-Discarded", "discarded"), full, plural, objects.WithGetMetadataHeader("X-Call", "new"), func(o *objects.GetMetadataOpts) error { retained = o; return nil }, func(*objects.GetMetadataOpts) error {
			retained.Headers["X-Call"] = "retained"
			*retained.Newest = true
			return nil
		}}
		for i := 0; i < 2; i++ {
			if _, err := objects.New(client).GetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, options...); err != nil {
				t.Fatal(err)
			}
		}
		if calls != 2 {
			t.Fatal(calls)
		}
		for _, tc := range []struct {
			options []objects.GetMetadataOption
			want    string
			present bool
		}{
			{[]objects.GetMetadataOption{objects.WithGetMetadataNewest(true)}, "true", true},
			{[]objects.GetMetadataOption{objects.WithGetMetadataNewest(false)}, "false", true},
			{[]objects.GetMetadataOption{objects.WithGetMetadataNewest(true), objects.WithoutGetMetadataNewest()}, "", false},
			{[]objects.GetMetadataOption{objects.WithGetMetadataNewest(true), objects.WithGetMetadataOpts(objects.GetMetadataOpts{})}, "", false},
		} {
			client.ProviderClient.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				_, present := r.Header["X-Newest"]
				if present != tc.present || r.Header.Get("X-Newest") != tc.want {
					t.Errorf("newest=%v", r.Header)
				}
				return objectMetadataResponse(200, ""), nil
			})
			if _, err := objects.New(client).GetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, tc.options...); err != nil {
				t.Fatal(err)
			}
		}
		mutation := objects.WithMetadataOpts(objects.MetadataOpts{Headers: map[string]string{"x-call": "old"}})
		client.ProviderClient.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			if r.Method == "HEAD" {
				if r.Header.Get("X-Call") != "" {
					t.Errorf("write option on read=%v", r.Header)
				}
				return objectMetadataResponse(200, ""), nil
			}
			if r.Header.Get("X-Call") != "new" || r.Header.Get("X-Discarded") != "" {
				t.Errorf("mutation options=%v", r.Header)
			}
			return objectMetadataResponse(202, ""), nil
		})
		if _, err := objects.New(client).SetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, nil, objects.WithMetadataHeader("X-Discarded", "discarded"), mutation, objects.WithMetadataHeader("X-Call", "new")); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("inputs copied before callbacks and callback causes kept", func(t *testing.T) {
		input := map[string]string{"Book": "original"}
		keys := []string{"Book"}
		writeHeaders := map[string]string{"X-Trace": "factory"}
		factory := objects.WithMetadataHeaders(writeHeaders)
		writeHeaders["X-Trace"] = "later"
		var retained *objects.MetadataOpts
		calls := 0
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method == "HEAD" {
				response := objectMetadataResponse(200, "")
				response.Header.Set("X-Object-Meta-Book", "old")
				response.Header.Set("X-Object-Meta-Keep", "yes")
				return response, nil
			}
			if r.Header.Get("X-Object-Meta-Book") != "original" || r.Header.Get("X-Trace") != "factory" {
				t.Errorf("input snapshot=%v", r.Header)
			}
			return objectMetadataResponse(202, ""), nil
		})
		_, err := objects.New(client).SetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, input, factory, func(o *objects.MetadataOpts) error { retained = o; input["Book"] = "changed"; return nil }, func(*objects.MetadataOpts) error { retained.Headers["X-Trace"] = "retained"; return nil })
		if err != nil || calls != 2 {
			t.Fatalf("snapshot err=%v HTTP=%d", err, calls)
		}
		calls = 0
		client.ProviderClient.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method == "HEAD" {
				response := objectMetadataResponse(200, "")
				response.Header.Set("X-Object-Meta-Book", "old")
				response.Header.Set("X-Object-Meta-Keep", "yes")
				return response, nil
			}
			if _, ok := objectMetadataCustom(r.Header)["book"]; ok || r.Header.Get("X-Object-Meta-Keep") != "yes" {
				t.Errorf("delete slice snapshot=%v", r.Header)
			}
			return objectMetadataResponse(202, ""), nil
		})
		if _, err := objects.New(client).DeleteMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, keys, func(*objects.MetadataOpts) error { keys[0] = "Keep"; return nil }); err != nil || calls != 2 {
			t.Fatalf("delete snapshot err=%v HTTP=%d", err, calls)
		}
		cause := errors.New("callback cause")
		ctx, cancel := context.WithCancelCause(context.Background())
		calls = 0
		result, err := objects.New(client).SetMetadata(ctx, objectMetadataContainer, objectMetadataKey, nil, func(*objects.MetadataOpts) error { cancel(cause); client.Endpoint += "changed"; return cause })
		if result != nil || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatalf("joined preflight=%+v err=%v HTTP=%d", result, err, calls)
		}
	})
}

func TestObjectMetadataContractsAtomicHeaders(t *testing.T) {
	t.Run("nullable precision literal fields and passive unknown headers", func(t *testing.T) {
		literal := map[string]string{"ETag": "\"literal hash\"", "Content-Type": "", "Content-Encoding": "gzip", "Content-Disposition": "한글\tinline", "X-Timestamp": "1.0000000000000000000001", "Last-Modified": "not a date", "X-Delete-At": "not an integer", "X-Object-Manifest": "container/%2F", "Cache-Control": "", "Content-Language": "ko", "Expires": "not a date", "X-Robots-Tag": "noindex"}
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
			response := objectMetadataResponse(200, "opaque HEAD proof")
			for k, v := range literal {
				response.Header.Set(k, v)
			}
			response.Header.Set("Content-Length", "+9223372036854775807")
			response.Header.Set("X-Object-Meta-Temp-URL-Key", "literal secret %2F")
			response.Header.Set("X-Object-Meta-Empty", "")
			response.Header["X-Unknown-System"] = []string{"one", "two"}
			return response, nil
		})
		result, err := objects.New(client).GetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey)
		if err != nil || result == nil || result.Metadata == nil || result.Metadata.ContentLength == nil || *result.Metadata.ContentLength != 9223372036854775807 || string(result.Body) != "opaque HEAD proof" {
			t.Fatalf("projection=%+v err=%v", result, err)
		}
		m := result.Metadata
		fields := map[string]*string{"ETag": m.ETag, "Content-Type": m.ContentType, "Content-Encoding": m.ContentEncoding, "Content-Disposition": m.ContentDisposition, "X-Timestamp": m.Timestamp, "Last-Modified": m.LastModified, "X-Delete-At": m.DeleteAt, "X-Object-Manifest": m.ObjectManifest, "Cache-Control": m.CacheControl, "Content-Language": m.ContentLanguage, "Expires": m.Expires, "X-Robots-Tag": m.RobotsTag}
		for k, v := range fields {
			if v == nil || *v != literal[k] {
				t.Errorf("literal %s=%v", k, v)
			}
		}
		if m.Values == nil || m.Values["temp-url-key"] != "literal secret %2F" || m.Values["empty"] != "" || !reflect.DeepEqual(result.Header["X-Unknown-System"], []string{"one", "two"}) {
			t.Fatalf("raw/lowercase=%+v header=%v", m, result.Header)
		}
		for _, value := range []string{"", "0", "-7"} {
			client.ProviderClient.HTTPClient.Transport = objectMetadataTransport(func(*http.Request) (*http.Response, error) {
				response := objectMetadataResponse(200, "")
				response.Header.Del("Content-Type")
				if value != "" {
					response.Header.Set("Content-Length", value)
				}
				return response, nil
			})
			result, err = objects.New(client).GetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey)
			if err != nil || result.Metadata.Values == nil || result.Metadata.ETag != nil || result.Metadata.ContentType != nil {
				t.Fatalf("missing fields=%+v err=%v", result, err)
			}
			if value == "" {
				if result.Metadata.ContentLength != nil {
					t.Fatal("missing length became zero")
				}
			} else {
				want, _ := strconv.ParseInt(value, 10, 64)
				if result.Metadata.ContentLength == nil || *result.Metadata.ContentLength != want {
					t.Fatalf("length=%+v", result.Metadata)
				}
			}
		}
	})
	t.Run("malformed known headers fail atomic with actual accepted proof", func(t *testing.T) {
		cases := []struct {
			name   string
			header http.Header
			number bool
		}{
			{"Kelvin original suffix", http.Header{"X-Object-Meta-K": {"bad"}}, false},
			{"empty suffix", http.Header{"X-Object-Meta-": {"bad"}}, false},
			{"suffix aliases", http.Header{"X-Object-Meta-Book": {"one"}, "x-object-meta-book": {"two"}}, false},
			{"duplicate custom value", http.Header{"X-Object-Meta-Book": {"one", "two"}}, false},
			{"zero custom values", http.Header{"X-Object-Meta-Book": {}}, false},
			{"invalid UTF8 value", http.Header{"X-Object-Meta-Book": {string([]byte{0xff})}}, false},
			{"control custom value", http.Header{"X-Object-Meta-Book": {"line\nvalue"}}, false},
			{"length aliases", http.Header{"Content-Length": {"1"}, "content-length": {"2"}}, false},
			{"duplicate literal", http.Header{"Last-Modified": {"one", "two"}}, false},
			{"zero literal", http.Header{"Content-Encoding": {}}, false},
			{"fraction", http.Header{"Content-Length": {"1.5"}}, true},
			{"whitespace", http.Header{"Content-Length": {" 1"}}, true},
			{"overflow", http.Header{"Content-Length": {"9223372036854775808"}}, true},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				body := &objectMetadataBody{Reader: strings.NewReader("accepted raw HEAD")}
				calls := 0
				client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
					calls++
					response := objectMetadataWire(200, body)
					for k, v := range tc.header {
						response.Header[k] = v
					}
					response.Header.Set("X-Object-Meta-Good", "would-be-partial")
					return response, nil
				})
				result, err := objects.New(client).GetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey)
				if result == nil || result.Metadata != nil || result.StatusCode != 200 || string(result.Body) != "accepted raw HEAD" || err == nil || calls != 1 || body.closes.Load() != 1 {
					t.Fatalf("result=%+v err=%v HTTP=%d closes=%d", result, err, calls, body.closes.Load())
				}
				proof := objectMetadataProof(t, err, 200, "accepted raw HEAD")
				if !reflect.DeepEqual(proof.Header, result.Header) {
					t.Fatal("whole header evidence lost")
				}
				if tc.number {
					var number *strconv.NumError
					if !errors.As(err, &number) {
						t.Fatalf("numeric cause=%v", err)
					}
				}
			})
		}
	})
	t.Run("read Close and custom cancellation independently retained once", func(t *testing.T) {
		readCause := errors.New("HEAD read cause")
		closeCause := errors.New("HEAD Close cause")
		ctxCause := errors.New("HEAD context cause")
		ctx, cancel := context.WithCancelCause(context.Background())
		calls, hooks := 0, 0
		body := &objectMetadataBody{Reader: objectMetadataReader(func(p []byte) (int, error) { return copy(p, "partial HEAD"), readCause }), closeErr: closeCause, onClose: func() { cancel(ctxCause) }}
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return objectMetadataWire(200, body), nil })
		client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			hooks++
			return nil
		}
		result, err := objects.New(client).GetMetadata(ctx, objectMetadataContainer, objectMetadataKey)
		if result == nil || result.Metadata != nil || result.StatusCode != 200 || string(result.Body) != "partial HEAD" || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, ctxCause) || calls != 1 || hooks != 0 || body.closes.Load() != 1 {
			t.Fatalf("result=%+v err=%v HTTP=%d hooks=%d closes=%d", result, err, calls, hooks, body.closes.Load())
		}
		objectMetadataProof(t, err, 200, "partial HEAD")
		for _, phase := range []string{"Read", "Close"} {
			t.Run(phase, func(t *testing.T) {
				cause := errors.New("only " + phase)
				body := &objectMetadataBody{Reader: strings.NewReader("complete bytes")}
				if phase == "Read" {
					body.Reader = objectMetadataReader(func(p []byte) (int, error) { return copy(p, "complete bytes"), cause })
				} else {
					body.closeErr = cause
				}
				client := objectMetadataClient(func(*http.Request) (*http.Response, error) { return objectMetadataWire(200, body), nil })
				result, err := objects.New(client).GetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey)
				if result == nil || result.Metadata != nil || !errors.Is(err, cause) || body.closes.Load() != 1 {
					t.Fatalf("single %s result=%+v err=%v", phase, result, err)
				}
				objectMetadataProof(t, err, 200, "complete bytes")
			})
		}
	})
	t.Run("typed raw headers and phase evidence have independent ownership", func(t *testing.T) {
		before := objectMetadataResponse(200, "before body")
		before.Header.Set("Content-Length", "0")
		before.Header.Set("X-Object-Meta-Book", "old")
		before.Header.Set("ETag", "observed-etag")
		ack := objectMetadataResponse(202, "ack body")
		ack.Header.Set("X-Object-Meta-Book", "not projected after POST")
		calls := 0
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method == "HEAD" {
				return before, nil
			}
			return ack, nil
		})
		result, err := objects.New(client).SetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, map[string]string{"Book": "new"})
		if err != nil || result == nil || result.Before.Metadata.Values["book"] != "old" || result.Before.Metadata.ETag == nil || result.Acknowledgement == nil || calls != 2 {
			t.Fatalf("result=%+v err=%v", result, err)
		}
		before.Header.Set("X-Object-Meta-Book", "transport changed")
		ack.Header.Set("X-Trans-Id", "transport changed")
		result.Before.Metadata.Values["book"] = "typed changed"
		*result.Before.Metadata.ETag = "typed changed"
		*result.Before.Metadata.ContentLength = 5
		if result.Before.Header.Get("X-Object-Meta-Book") != "old" || result.Before.Header.Get("ETag") != "observed-etag" || result.Before.Header.Get("Content-Length") != "0" || result.Acknowledgement.Header.Get("X-Trans-Id") != "actual-object" {
			t.Fatal("typed/raw/transport header alias")
		}
		result.Before.Header.Set("X-Trans-Id", "before changed")
		result.Before.Body[0] = 'B'
		if result.Acknowledgement.Header.Get("X-Trans-Id") != "actual-object" || string(result.Acknowledgement.Body) != "ack body" {
			t.Fatal("phase evidence aliases")
		}
	})
}

func TestObjectMetadataContractsMutationEvidence(t *testing.T) {
	t.Run("accepted read failure preserves Before and prevents write", func(t *testing.T) {
		for _, failure := range []string{"read", "Close", "projection", "source", "context"} {
			t.Run(failure, func(t *testing.T) {
				cause := errors.New("read phase " + failure)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				calls := 0
				body := &objectMetadataBody{Reader: strings.NewReader("read phase proof")}
				var client *gophercloud.ServiceClient
				response := objectMetadataWire(200, body)
				response.Header.Set("X-Object-Meta-Good", "observed")
				switch failure {
				case "read":
					body.Reader = objectMetadataReader(func(p []byte) (int, error) { return copy(p, "read phase proof"), cause })
				case "Close":
					body.closeErr = cause
				case "projection":
					response.Header["X-Object-Meta-Good"] = []string{"one", "two"}
				case "source":
					body.onClose = func() { client.ResourceBase = objectMetadataBase }
				case "context":
					body.onClose = func() { cancel(cause) }
				}
				client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Method != "HEAD" {
						t.Error("failed read caused POST")
					}
					return response, nil
				})
				result, err := objects.New(client).SetMetadata(ctx, objectMetadataContainer, objectMetadataKey, map[string]string{"Book": "new"})
				if result == nil || result.Before == nil || result.Before.Metadata != nil || result.Before.StatusCode != 200 || result.Acknowledgement != nil || err == nil || calls != 1 || body.closes.Load() != 1 {
					t.Fatalf("result=%+v err=%v HTTP=%d closes=%d", result, err, calls, body.closes.Load())
				}
				objectMetadataProof(t, err, 200, "read phase proof")
				if (failure == "read" || failure == "Close" || failure == "context") && !errors.Is(err, cause) {
					t.Fatalf("underlying cause lost=%v", err)
				}
				if failure == "source" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatalf("source cause lost=%v", err)
				}
				if failure == "context" && !errors.Is(err, context.Canceled) {
					t.Fatalf("context cause lost=%v", err)
				}
			})
		}
		for _, status := range []int{204, 401, 404, 503} {
			calls := 0
			client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
				calls++
				return objectMetadataResponse(status, `{"error":"read rejected"}`), nil
			})
			result, err := objects.New(client).DeleteMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, nil)
			var native gophercloud.ErrUnexpectedResponseCode
			if result != nil || !errors.As(err, &native) || native.Actual != status || !reflect.DeepEqual(native.Expected, []int{200}) || calls != 1 {
				t.Fatalf("unaccepted read%d=%+v err=%v HTTP=%d", status, result, err, calls)
			}
		}
	})
	t.Run("native post failures keep completed Before without acknowledgement", func(t *testing.T) {
		for _, status := range []int{200, 201, 204, 400, 401, 403, 404, 409, 412, 503} {
			t.Run(strconv.Itoa(status), func(t *testing.T) {
				calls := 0
				client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Method == "HEAD" {
						response := objectMetadataResponse(200, "accepted read")
						response.Header.Set("X-Object-Meta-Book", "old")
						return response, nil
					}
					return objectMetadataResponse(status, `{"error":"post rejected"}`), nil
				})
				result, err := objects.New(client).SetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, map[string]string{"Book": "new"})
				var native gophercloud.ErrUnexpectedResponseCode
				if result == nil || result.Before.Metadata.Values["book"] != "old" || string(result.Before.Body) != "accepted read" || result.Acknowledgement != nil || !errors.As(err, &native) || native.Actual != status || native.Method != "POST" || native.URL != objectMetadataEndpoint || !reflect.DeepEqual(native.Expected, []int{202}) || string(native.Body) != `{"error":"post rejected"}` || native.ResponseHeader.Get("X-Trans-Id") != "actual-object" || calls != 2 {
					t.Fatalf("post%d result=%+v native=%+v err=%v HTTP=%d", status, result, native, err, calls)
				}
			})
		}
		for _, cause := range []error{errors.New("POST transport"), &gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{202}}} {
			calls := 0
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method == "HEAD" {
					return objectMetadataResponse(200, "accepted read"), nil
				}
				return nil, cause
			})
			result, err := objects.New(client).DeleteMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, nil)
			if result == nil || result.Before == nil || result.Before.Metadata == nil || result.Acknowledgement != nil || !errors.Is(err, cause) || calls != 2 {
				t.Fatalf("post cause=%v result=%+v err=%v HTTP=%d", cause, result, err, calls)
			}
		}
	})
	t.Run("opaque accepted post read Close context and source evidence", func(t *testing.T) {
		for _, failure := range []string{"read Close context", "source"} {
			t.Run(failure, func(t *testing.T) {
				readCause := errors.New("POST read")
				closeCause := errors.New("POST Close")
				ctxCause := errors.New("POST context")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				calls, hooks := 0, 0
				body := &objectMetadataBody{Reader: strings.NewReader("opaque post proof")}
				var client *gophercloud.ServiceClient
				if failure == "source" {
					body.onClose = func() { client.Type = "changed" }
				} else {
					body.Reader = objectMetadataReader(func(p []byte) (int, error) { return copy(p, "opaque post proof"), readCause })
					body.closeErr = closeCause
					body.onClose = func() { cancel(ctxCause) }
				}
				client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Method == "HEAD" {
						response := objectMetadataResponse(200, "before")
						response.Header.Set("X-Object-Meta-Book", "old")
						return response, nil
					}
					response := objectMetadataWire(202, body)
					response.Header["X-Object-Meta-Malformed"] = nil
					return response, nil
				})
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks++
					return nil
				}
				result, err := objects.New(client).SetMetadata(ctx, objectMetadataContainer, objectMetadataKey, map[string]string{"Book": "new"})
				if result == nil || result.Before.Metadata.Values["book"] != "old" || result.Acknowledgement == nil || result.Acknowledgement.StatusCode != 202 || string(result.Acknowledgement.Body) != "opaque post proof" || err == nil || calls != 2 || hooks != 0 || body.closes.Load() != 1 {
					t.Fatalf("result=%+v err=%v HTTP=%d hooks=%d closes=%d", result, err, calls, hooks, body.closes.Load())
				}
				proof := objectMetadataProof(t, err, 202, "opaque post proof")
				if failure == "source" {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				} else {
					for _, cause := range []error{readCause, closeCause, context.Canceled, ctxCause} {
						if !errors.Is(err, cause) {
							t.Fatalf("cause=%v lost in=%v", cause, err)
						}
					}
				}
				proof.Header.Set("X-Trans-Id", "error changed")
				if result.Before.Header.Get("X-Trans-Id") != "actual-object" {
					t.Fatal("post error and Before headers aliased")
				}
			})
		}
	})
	t.Run("own symlink refusal and raced applied redirect never retarget", func(t *testing.T) {
		for _, header := range []http.Header{
			{"X-Symlink-Target": {"target/object"}},
			{"X-Symlink-Target": {""}},
			{"X-Symlink-Target": {"current/object?version-id=stored-version"}, "X-Object-Version-Id": {"stored-version"}},
			{"X-Symlink-Target": {"target/object"}, "X-Symlink-Target-Account": {"one", "two"}},
			{"X-Symlink-Target": {"target/object"}, "X-Symlink-Target-Bytes": {"bad\nvalue"}},
			{"X-Symlink-Target": {"target/object"}, "x-symlink-target": {"other/object"}},
		} {
			calls := 0
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				objectMetadataBodyless(t, r, true)
				response := objectMetadataResponse(200, "link evidence")
				for k, v := range header {
					response.Header[k] = v
				}
				return response, nil
			})
			result, err := objects.New(client).SetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, nil)
			if result == nil || result.Before == nil || result.Before.Metadata != nil || result.Acknowledgement != nil || err == nil || calls != 1 {
				t.Fatalf("symlink=%v result=%+v err=%v HTTP=%d", header, result, err, calls)
			}
			objectMetadataProof(t, err, 200, "link evidence")
			if len(header) == 1 || header.Get("X-Object-Version-Id") != "" {
				if !errors.Is(err, resource.ErrUnsupported) {
					t.Fatalf("observed link not unsupported=%v", err)
				}
			}
		}
		calls := 0
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method == "HEAD" {
				if r.URL.RawQuery != "" {
					t.Fatal("normal getter owns link query")
				}
				response := objectMetadataResponse(200, "")
				response.Header.Set("X-Symlink-Target", "server-followed-target")
				return response, nil
			}
			t.Fatal("unexpected POST")
			return nil, nil
		})
		if value, err := objects.New(client).GetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey); err != nil || value == nil || calls != 1 {
			t.Fatalf("normal getter=%+v err=%v HTTP=%d", value, err, calls)
		}
		calls = 0
		client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method == "HEAD" {
				return objectMetadataResponse(200, "before raced link"), nil
			}
			response := objectMetadataResponse(307, "metadata may already be applied")
			response.Header.Set("Location", "https://other.invalid/target/object")
			return response, nil
		})
		result, err := objects.New(client).DeleteMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, nil)
		if result == nil || result.Before == nil || string(result.Before.Body) != "before raced link" || result.Acknowledgement != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 2 {
			t.Fatalf("raced link=%+v err=%v HTTP=%d", result, err, calls)
		}
	})
}

func TestObjectMetadataContractsSourceAndNativePolicy(t *testing.T) {
	t.Run("source captured across phases live token and identity guard", func(t *testing.T) {
		calls := 0
		var client *gophercloud.ServiceClient
		before := objectMetadataResponse(200, "before")
		before.Header.Set("Content-Type", "observed/type")
		before.Header.Set("X-Object-Meta-Keep", "observed")
		before.Body.(*objectMetadataBody).onClose = func() { client.MoreHeaders["X-Source"] = "during read"; client.ProviderClient.SetToken("post-token") }
		client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("X-Source") != "captured" || r.Header.Get("Content-Type") != "source/type" {
				t.Errorf("captured source=%v", r.Header)
			}
			if r.Method == "HEAD" {
				if r.Header.Get("X-Only-Write") != "" || r.Header.Get("X-Auth-Token") != "read-token" || r.URL.String() != objectMetadataEndpoint+"?symlink=get" {
					t.Errorf("read headers=%v URL=%s", r.Header, r.URL)
				}
				return before, nil
			}
			if r.Header.Get("X-Only-Write") != "option" || r.Header.Get("X-Auth-Token") != "post-token" || r.URL.String() != objectMetadataEndpoint {
				t.Errorf("post headers=%v URL=%s", r.Header, r.URL)
			}
			return objectMetadataResponse(202, ""), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "captured", "Content-Type": "source/type"}
		provider := client.ProviderClient
		result, err := objects.New(client).SetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, nil, objects.WithMetadataHeader("X-Only-Write", "option"), func(*objects.MetadataOpts) error {
			client.MoreHeaders["X-Source"] = "after callback"
			provider.SetToken("read-token")
			return nil
		})
		if err != nil || result == nil || calls != 2 || client.MoreHeaders["X-Source"] != "during read" || client.MoreHeaders["Content-Type"] != "source/type" || client.ProviderClient != provider {
			t.Fatalf("result=%+v err=%v HTTP=%d source=%v", result, err, calls, client.MoreHeaders)
		}
		client.ProviderClient.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Source") != "during read" || r.Header.Get("X-Only-Write") != "" || r.Header.Get("X-Object-Meta-Keep") != "" || r.URL.RawQuery != "" {
				t.Errorf("future call leaked=%v URL=%s", r.Header, r.URL)
			}
			return objectMetadataResponse(200, ""), nil
		})
		if _, err := objects.New(client).GetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey); err != nil {
			t.Fatal(err)
		}
		for _, mutate := range []func(*gophercloud.ServiceClient){
			func(c *gophercloud.ServiceClient) { c.Endpoint += "other" }, func(c *gophercloud.ServiceClient) { c.ResourceBase = objectMetadataBase }, func(c *gophercloud.ServiceClient) { c.Type = "image" }, func(c *gophercloud.ServiceClient) { c.Microversion = "changed" }, func(c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} }, func(c *gophercloud.ServiceClient) { c.MoreHeaders = map[string]string{"X-Symlink-Target": "bad"} },
		} {
			calls = 0
			local := objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return objectMetadataResponse(200, ""), nil })
			result, err := objects.New(local).SetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, nil, func(*objects.MetadataOpts) error { mutate(local); return nil })
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatalf("source guard=%+v err=%v HTTP=%d", result, err, calls)
			}
		}
		for _, local := range []*gophercloud.ServiceClient{nil, {Type: "object-store", Endpoint: objectMetadataBase}} {
			callbacks := 0
			result, err := objects.New(local).GetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, func(*objects.GetMetadataOpts) error { callbacks++; return nil })
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
				t.Fatalf("missing source=%+v err=%v callbacks=%d", result, err, callbacks)
			}
		}
	})
	t.Run("per-request prebody retry never restarts RMW and advanced policy", func(t *testing.T) {
		heads, posts, hooks := 0, 0, 0
		rejected := &objectMetadataBody{Reader: strings.NewReader(`{"error":"post retry"}`)}
		client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			if r.Method == "HEAD" {
				heads++
				response := objectMetadataResponse(200, "one observed read")
				response.Header.Set("X-Object-Meta-Keep", "observed")
				return response, nil
			}
			posts++
			if posts == 1 {
				return objectMetadataWire(503, rejected), nil
			}
			if r.Header.Get("X-Object-Meta-Keep") != "observed" {
				t.Errorf("replacement changed on retry=%v", r.Header)
			}
			return objectMetadataResponse(202, ""), nil
		})
		client.RetryFunc = func(_ context.Context, method, url string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			hooks++
			var native gophercloud.ErrUnexpectedResponseCode
			if method != "POST" || url != objectMetadataEndpoint || !errors.As(err, &native) || native.Actual != 503 {
				t.Errorf("retry original=%s %s %v", method, url, err)
			}
			return nil
		}
		result, err := objects.New(client).DeleteMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, nil)
		if err != nil || result == nil || result.Acknowledgement == nil || heads != 1 || posts != 2 || hooks != 1 || rejected.closes.Load() != 1 {
			t.Fatalf("retry result=%+v err=%v heads=%d posts=%d hooks=%d closes=%d", result, err, heads, posts, hooks, rejected.closes.Load())
		}
		heads, posts, hooks = 0, 0, 0
		client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			if r.Method == "HEAD" {
				heads++
				return objectMetadataResponse(200, ""), nil
			}
			posts++
			if posts == 1 {
				return objectMetadataResponse(503, `{"error":"advanced"}`), nil
			}
			if r.Header.Get("X-Advanced") != "provider-owned" || r.Header.Get("X-Object-Meta-Book") != "advanced-native" {
				t.Errorf("advanced native headers=%v", r.Header)
			}
			return objectMetadataResponse(202, ""), nil
		})
		client.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			hooks++
			o.MoreHeaders = map[string]string{"X-Advanced": "provider-owned", "X-Object-Meta-Book": "advanced-native"}
			return nil
		}
		provider := client.ProviderClient
		if result, err := objects.New(client).SetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, map[string]string{"Book": "ordinary"}); err != nil || result == nil || heads != 1 || posts != 2 || hooks != 1 || client.ProviderClient != provider || client.MoreHeaders != nil {
			t.Fatalf("advanced result=%+v err=%v heads=%d posts=%d hooks=%d", result, err, heads, posts, hooks)
		}
		calls := 0
		transportCause := errors.New("native prebody transport")
		client = objectMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return nil, transportCause
			}
			return objectMetadataResponse(200, ""), nil
		})
		client.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			if !errors.Is(err, transportCause) {
				t.Errorf("transport cause=%v", err)
			}
			return nil
		}
		if value, err := objects.New(client).GetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey); err != nil || value == nil || calls != 2 {
			t.Fatalf("transport retry=%+v err=%v HTTP=%d", value, err, calls)
		}
	})
	t.Run("ownership guards actual codes and native reauth evidence", func(t *testing.T) {
		for _, change := range []struct {
			name string
			fn   func(*gophercloud.RequestOpts)
		}{
			{"KeepResponseBody", func(o *gophercloud.RequestOpts) { o.KeepResponseBody = false }},
			{"JSONResponse", func(o *gophercloud.RequestOpts) { o.JSONResponse = new(any) }},
			{"RawBody", func(o *gophercloud.RequestOpts) { o.RawBody = strings.NewReader("changed") }},
			{"JSON null", func(o *gophercloud.RequestOpts) { o.JSONBody = json.RawMessage("null") }},
			{"encoding cause", func(o *gophercloud.RequestOpts) { o.JSONBody = make(chan int) }},
		} {
			t.Run(change.name, func(t *testing.T) {
				calls, hooks := 0, 0
				body := &objectMetadataBody{Reader: strings.NewReader(`{"error":"first post"}`)}
				client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
					calls++
					objectMetadataBodyless(t, r, r.Method == "HEAD")
					if r.Method == "HEAD" {
						return objectMetadataResponse(200, "before guard"), nil
					}
					return objectMetadataWire(503, body), nil
				})
				callbackCause := errors.New("hook cause")
				client.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
					hooks++
					change.fn(o)
					return callbackCause
				}
				result, err := objects.New(client).SetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, nil)
				var native gophercloud.ErrUnexpectedResponseCode
				if result == nil || result.Before == nil || result.Acknowledgement != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, callbackCause) || !errors.As(err, &native) || native.Actual != 503 || calls != 2 || hooks != 1 || body.closes.Load() != 1 {
					t.Fatalf("guard result=%+v err=%v HTTP=%d hooks=%d closes=%d", result, err, calls, hooks, body.closes.Load())
				}
				if change.name == "encoding cause" {
					var encoding *json.UnsupportedTypeError
					if !errors.As(err, &encoding) {
						t.Fatalf("encoding cause=%v", err)
					}
				}
			})
		}
		for _, phase := range []string{"HEAD", "POST"} {
			calls := 0
			closeCause := errors.New("expanded status Close")
			body := &objectMetadataBody{Reader: strings.NewReader("unexpected accepted code"), closeErr: closeCause}
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if phase == "POST" && r.Method == "HEAD" {
					return objectMetadataResponse(200, "before expanded"), nil
				}
				first := calls == 1 || (phase == "POST" && calls == 2)
				if first {
					return objectMetadataResponse(503, `{"error":"expand"}`), nil
				}
				status := 204
				if phase == "POST" {
					status = 201
				}
				return objectMetadataWire(status, body), nil
			})
			client.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				o.OkCodes = []int{200, 201, 202, 204}
				return nil
			}
			var err error
			var native gophercloud.ErrUnexpectedResponseCode
			if phase == "HEAD" {
				result, e := objects.New(client).GetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey)
				err = e
				if result != nil || calls != 2 {
					t.Fatalf("unexpected HEAD result=%+v HTTP=%d", result, calls)
				}
			} else {
				result, e := objects.New(client).SetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, nil)
				err = e
				if result == nil || result.Before == nil || result.Acknowledgement != nil || calls != 3 {
					t.Fatalf("unexpected POST result=%+v HTTP=%d", result, calls)
				}
			}
			want := 200
			actual := 204
			if phase == "POST" {
				want = 202
				actual = 201
			}
			if !errors.As(err, &native) || native.Actual != actual || !reflect.DeepEqual(native.Expected, []int{want}) || string(native.Body) != "unexpected accepted code" || native.ResponseHeader.Get("X-Trans-Id") != "actual-object" || !errors.Is(err, closeCause) || body.closes.Load() != 1 {
				t.Fatalf("actual status=%+v err=%v closes=%d", native, err, body.closes.Load())
			}
		}
		heads, posts, auth := 0, 0, 0
		var client *gophercloud.ServiceClient
		client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
			if r.Method == "HEAD" {
				heads++
				return objectMetadataResponse(200, "before auth"), nil
			}
			posts++
			if posts == 1 {
				return objectMetadataResponse(401, `{"error":"auth"}`), nil
			}
			if r.Header.Get("X-Auth-Token") != "refreshed-token" {
				t.Errorf("reauth token=%v", r.Header)
			}
			return objectMetadataResponse(202, ""), nil
		})
		client.ReauthFunc = func(context.Context) error { auth++; client.ProviderClient.SetToken("refreshed-token"); return nil }
		if result, err := objects.New(client).SetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, nil); err != nil || result == nil || heads != 1 || posts != 2 || auth != 1 {
			t.Fatalf("reauth result=%+v err=%v heads=%d posts=%d auth=%d", result, err, heads, posts, auth)
		}
		reauthCause := errors.New("native reauth cause")
		client = objectMetadataClient(func(*http.Request) (*http.Response, error) {
			return objectMetadataResponse(401, `{"error":"original auth"}`), nil
		})
		client.ReauthFunc = func(context.Context) error { return reauthCause }
		value, err := objects.New(client).GetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey)
		var unable *gophercloud.ErrUnableToReauthenticate
		if value != nil || !errors.As(err, &unable) || unable.ErrReauth != reauthCause {
			t.Fatalf("reauth failure=%+v err=%v fields=%+v", value, err, unable)
		}
		var original gophercloud.ErrUnexpectedResponseCode
		if !errors.As(unable.ErrOriginal, &original) || original.Actual != 401 {
			t.Fatalf("native ErrOriginal=%v", unable.ErrOriginal)
		}
	})
	t.Run("fixed request redirects source drift and cancellation", func(t *testing.T) {
		for _, tc := range []struct {
			post     bool
			status   int
			location string
		}{
			{false, 307, "https://other.invalid/stolen"}, {false, 307, objectMetadataEndpoint + "?version-id=unexpected"}, {true, 307, objectMetadataBase + "other"}, {true, 302, objectMetadataEndpoint},
		} {
			calls := 0
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if tc.post && r.Method == "HEAD" {
					return objectMetadataResponse(200, "before redirect"), nil
				}
				response := objectMetadataResponse(tc.status, "")
				response.Header.Set("Location", tc.location)
				return response, nil
			})
			var err error
			if tc.post {
				result, e := objects.New(client).SetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey, nil)
				err = e
				if result == nil || result.Before == nil || result.Acknowledgement != nil || calls != 2 {
					t.Fatalf("post redirect result=%+v HTTP=%d", result, calls)
				}
			} else {
				value, e := objects.New(client).GetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey)
				err = e
				if value != nil || calls != 1 {
					t.Fatalf("get redirect=%+v HTTP=%d", value, calls)
				}
			}
			if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("redirect=%+v err=%v", tc, err)
			}
		}
		calls, checks := 0, 0
		client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				response := objectMetadataResponse(307, "")
				response.Header.Set("Location", objectMetadataEndpoint)
				return response, nil
			}
			return objectMetadataResponse(200, ""), nil
		})
		client.HTTPClient.CheckRedirect = func(r *http.Request, via []*http.Request) error {
			checks++
			if r.URL.String() != objectMetadataEndpoint || len(via) != 1 {
				t.Errorf("native redirect=%s via=%d", r.URL, len(via))
			}
			return nil
		}
		if value, err := objects.New(client).GetMetadata(context.Background(), objectMetadataContainer, objectMetadataKey); err != nil || value == nil || calls != 2 || checks != 1 {
			t.Fatalf("identical native redirect=%+v err=%v HTTP=%d checks=%d", value, err, calls, checks)
		}
		ctx, cancel := context.WithCancelCause(context.Background())
		cause := errors.New("retry context cause")
		calls = 0
		client = objectMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			return objectMetadataResponse(503, `{"error":"before cancel"}`), nil
		})
		client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			cancel(cause)
			return nil
		}
		value, err := objects.New(client).GetMetadata(ctx, objectMetadataContainer, objectMetadataKey)
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || !errors.As(err, &native) || native.Actual != 503 || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls != 1 {
			t.Fatalf("retry cancel=%+v err=%v HTTP=%d", value, err, calls)
		}
	})
}
