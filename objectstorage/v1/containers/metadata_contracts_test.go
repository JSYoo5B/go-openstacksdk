package containers_test

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

	"github.com/JSYoo5B/gophercloudsdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	swift "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1"
)

const containerMetadataBase = "https://swift.invalid/reverse/a%20b/v1/AUTH_account/"
const containerMetadataName = "한글 space:%2F?#"

var containerMetadataEndpoint = containerMetadataBase + url.PathEscape(containerMetadataName)

type containerMetadataTransport func(*http.Request) (*http.Response, error)

func (f containerMetadataTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := f(r)
	if response != nil && response.Request == nil {
		response.Request = r
	}
	return response, err
}

type containerMetadataBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *containerMetadataBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type containerMetadataReader func([]byte) (int, error)

func (f containerMetadataReader) Read(p []byte) (int, error) { return f(p) }

func containerMetadataClient(f containerMetadataTransport) *gophercloud.ServiceClient {
	p := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: f}}
	p.UseTokenLock()
	p.SetToken("original-token")
	return &gophercloud.ServiceClient{ProviderClient: p, Type: "object-store", Endpoint: containerMetadataBase}
}

func containerMetadataWire(status int, body *containerMetadataBody) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}, "X-Trans-Id": {"actual-container"}}, Body: body}
}

func containerMetadataResponse(status int, body string) *http.Response {
	return containerMetadataWire(status, &containerMetadataBody{Reader: strings.NewReader(body)})
}

func containerMetadataProof(t *testing.T, err error, status int, body string) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Trans-Id") != "actual-container" {
		t.Fatalf("response proof=%+v err=%v", proof, err)
	}
	return proof
}

func containerMetadataNoBody(t *testing.T, r *http.Request) {
	t.Helper()
	if r.URL.String() != containerMetadataEndpoint || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "original-token" {
		t.Errorf("request=%s headers=%v", r.URL, r.Header)
	}
	for key := range r.Header {
		if strings.HasPrefix(strings.ToLower(key), "x-account-meta-") {
			t.Errorf("account metadata escaped container unit: %s", key)
		}
	}
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) != 0 {
			t.Errorf("unexpected request body=%q err=%v", body, err)
		}
	}
}

func TestContainerMetadataContractsWireAndIdentity(t *testing.T) {
	t.Run("literal identity and effective base escaped once without discovery", func(t *testing.T) {
		for _, tc := range []struct{ endpoint, base, name string }{
			{containerMetadataBase, "", containerMetadataName},
			{containerMetadataBase, "", " %2F?# "},
			{containerMetadataBase, "", ".hidden"},
			{"https://swift.invalid/account-endpoint", "https://swift.invalid/p%2Fq/v1/AUTH_other/", containerMetadataName},
		} {
			t.Run(tc.name+tc.base, func(t *testing.T) {
				base := tc.base
				if base == "" {
					base = tc.endpoint
				}
				expected := base + url.PathEscape(tc.name)
				parsed, err := url.Parse(expected)
				if err != nil {
					t.Fatal(err)
				}
				calls := 0
				client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Method != "HEAD" || r.URL.String() != expected || r.URL.EscapedPath() != parsed.EscapedPath() || r.URL.RequestURI() != parsed.RequestURI() || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "original-token" || r.Body != nil {
						t.Errorf("request=%s %s URI=%s body=%v headers=%v", r.Method, r.URL, r.URL.RequestURI(), r.Body, r.Header)
					}
					return containerMetadataResponse(204, ""), nil
				})
				client.Type, client.Endpoint, client.ResourceBase = "", tc.endpoint, tc.base
				result, err := containers.New(client).GetMetadata(context.Background(), tc.name)
				if err != nil || result == nil || result.StatusCode != 204 || result.Metadata == nil || result.Metadata.Values == nil || calls != 1 {
					t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
				}
			})
		}
		var wireCalls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			n := wireCalls.Add(1)
			if r.RequestURI != "/reverse/p%2Fq/v1/AUTH_account/"+url.PathEscape(containerMetadataName) || r.Header.Get("X-Auth-Token") != "original-token" {
				t.Errorf("wire URI=%s headers=%v", r.RequestURI, r.Header)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil || len(body) != 0 {
				t.Errorf("wire body=%q err=%v", body, err)
			}
			if n == 1 && r.Method != "HEAD" {
				t.Errorf("unexpected lookup method=%s", r.Method)
			}
			if n == 2 && (r.Method != "POST" || r.Header.Get("X-Container-Meta-Book") != "wire") {
				t.Errorf("wire set headers=%v", r.Header)
			}
			if n == 3 && (r.Method != "POST" || !reflect.DeepEqual(r.Header.Values("X-Container-Meta-Book"), []string{""})) {
				t.Errorf("wire delete headers=%v", r.Header)
			}
			if len(r.Header.Values("X-Account-Meta-Book")) != 0 {
				t.Errorf("wrong account prefix=%v", r.Header)
			}
			w.Header().Set("X-Container-Object-Count", "0")
			w.WriteHeader(204)
		}))
		defer server.Close()
		client := containerMetadataClient(nil)
		client.HTTPClient = *server.Client()
		client.Endpoint = server.URL + "/endpoint/"
		client.ResourceBase = server.URL + "/reverse/p%2Fq/v1/AUTH_account/"
		api := containers.New(client)
		if result, err := api.GetMetadata(context.Background(), containerMetadataName); err != nil || result == nil || result.Metadata.ObjectCount == nil || *result.Metadata.ObjectCount != 0 {
			t.Fatalf("wire HEAD=%+v err=%v", result, err)
		}
		if _, err := api.SetMetadata(context.Background(), containerMetadataName, map[string]string{"Book": "wire"}); err != nil {
			t.Fatal(err)
		}
		if _, err := api.DeleteMetadata(context.Background(), containerMetadataName, []string{"Book"}); err != nil {
			t.Fatal(err)
		}
		if wireCalls.Load() != 3 {
			t.Fatalf("wire calls=%d", wireCalls.Load())
		}
	})
	t.Run("single POST literal set and empty delete", func(t *testing.T) {
		calls := 0
		client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			containerMetadataNoBody(t, r)
			if r.Method != "POST" || len(r.Header.Values("X-Remove-Container-Meta-Book")) != 0 || len(r.Header.Values("X-Newest")) != 0 {
				t.Errorf("method=%s headers=%v", r.Method, r.Header)
			}
			if calls == 1 && (r.Header.Get("X-Container-Read") != "r:*" || r.Header.Get("Content-Type") != "application/custom" || r.Header.Get("X-Versions-Location") != "previous" || r.Header.Get("X-Container-Sync-Key") != "" || len(r.Header.Values("X-Container-Sync-Key")) != 1 || r.Header.Get("X-Container-Meta-Read_acl") != "custom alias" || r.Header.Get("X-Container-Meta-Sync_key") != "custom sync" || r.Header.Get("X-Container-Meta-Book") != "  한글 %2F\t" || r.Header.Get("X-Container-Meta-Empty") != "" || len(r.Header.Values("X-Container-Meta-Empty")) != 1) {
				t.Errorf("literal set headers=%v", r.Header)
			}
			if calls == 2 && !reflect.DeepEqual(r.Header.Values("X-Container-Meta-Book"), []string{""}) {
				t.Errorf("delete headers=%v", r.Header)
			}
			return containerMetadataResponse(204, "opaque"), nil
		})
		api := containers.New(client)
		if result, err := api.SetMetadata(context.Background(), containerMetadataName, map[string]string{"Book": "  한글 %2F\t", "Empty": "", "read_ACL": "custom alias", "sync_key": "custom sync"}, containers.WithMetadataHeaders(map[string]string{"X-Container-Read": "r:*", "X-Container-Sync-Key": "", "Content-Type": "application/custom", "X-Versions-Location": "previous"})); err != nil || result == nil || string(result.Body) != "opaque" {
			t.Fatalf("set=%+v err=%v", result, err)
		}
		if result, err := api.DeleteMetadata(context.Background(), containerMetadataName, []string{"Book"}); err != nil || result == nil {
			t.Fatalf("delete=%+v err=%v", result, err)
		}
		for _, invoke := range []func() error{
			func() error { _, err := api.SetMetadata(context.Background(), containerMetadataName, nil); return err },
			func() error {
				_, err := api.SetMetadata(context.Background(), containerMetadataName, map[string]string{})
				return err
			},
			func() error {
				_, err := api.DeleteMetadata(context.Background(), containerMetadataName, nil)
				return err
			},
			func() error {
				_, err := api.DeleteMetadata(context.Background(), containerMetadataName, []string{})
				return err
			},
		} {
			if err := invoke(); err != nil {
				t.Fatal(err)
			}
		}
		if calls != 6 {
			t.Fatalf("POSTs=%d; unexpected refresh/fallback", calls)
		}
	})
	t.Run("newest nil false true and clear", func(t *testing.T) {
		want := []string{"", "false", "true", ""}
		calls := 0
		client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
			containerMetadataNoBody(t, r)
			if r.Method != "HEAD" || r.Header.Get("X-Newest") != want[calls] || (want[calls] == "" && len(r.Header.Values("X-Newest")) != 0) {
				t.Errorf("newest=%v wanted=%q", r.Header.Values("X-Newest"), want[calls])
			}
			calls++
			return containerMetadataResponse(204, ""), nil
		})
		for _, opts := range [][]containers.GetMetadataOption{nil, {containers.WithGetMetadataNewest(false)}, {containers.WithGetMetadataNewest(true)}, {containers.WithGetMetadataNewest(true), containers.WithoutGetMetadataNewest()}} {
			if _, err := containers.New(client).GetMetadata(context.Background(), containerMetadataName, opts...); err != nil {
				t.Fatal(err)
			}
		}
		if calls != 4 {
			t.Fatalf("calls=%d", calls)
		}
	})
	t.Run("nullable signed precision and literal metadata", func(t *testing.T) {
		client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
			response := containerMetadataResponse(204, "non-json")
			response.Header["x-container-meta-BOOK"] = []string{"%2F + literal"}
			response.Header["X-Container-Meta-Temp-Url-Key"] = []string{"secret"}
			response.Header["X-Container-Meta-Temp-Url-Key-2"] = []string{"second"}
			response.Header["X-Container-Bytes-Used"] = []string{"+9007199254740993"}
			response.Header["X-Container-Object-Count"] = []string{"-1"}
			response.Header["X-Timestamp"] = []string{""}
			response.Header["Last-Modified"] = []string{"not a date"}
			response.Header["Date"] = []string{"not a timestamp"}
			response.Header["X-Vendor"] = []string{"one", "two"}
			return response, nil
		})
		result, err := containers.New(client).GetMetadata(context.Background(), containerMetadataName)
		if err != nil || result == nil || result.Metadata == nil {
			t.Fatalf("result=%+v err=%v", result, err)
		}
		m := result.Metadata
		if m.BytesUsed == nil || *m.BytesUsed != 9007199254740993 || m.ObjectCount == nil || *m.ObjectCount != -1 || m.Timestamp == nil || *m.Timestamp != "" || m.LastModified == nil || *m.LastModified != "not a date" || m.Values["book"] != "%2F + literal" || m.Values["temp-url-key"] != "secret" || m.Values["temp-url-key-2"] != "second" || string(result.Body) != "non-json" || !reflect.DeepEqual(result.Header["X-Vendor"], []string{"one", "two"}) {
			t.Fatalf("metadata=%+v raw=%+v", m, result)
		}
		client.ProviderClient.HTTPClient.Transport = containerMetadataTransport(func(*http.Request) (*http.Response, error) { return containerMetadataResponse(204, ""), nil })
		missing, err := containers.New(client).GetMetadata(context.Background(), containerMetadataName)
		if err != nil || missing.Metadata.BytesUsed != nil || missing.Metadata.ObjectCount != nil || missing.Metadata.Timestamp != nil || missing.Metadata.LastModified != nil || missing.Metadata.Values == nil {
			t.Fatalf("missing=%+v err=%v", missing, err)
		}
	})
}

func TestContainerMetadataContractsPreflightAndOptions(t *testing.T) {
	t.Run("context source and callback causes", func(t *testing.T) {
		calls, callbacks := 0, 0
		client := containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return containerMetadataResponse(204, ""), nil })
		option := func(o *containers.GetMetadataOpts) error { callbacks++; return nil }
		var nilAPI *containers.API
		for _, api := range []*containers.API{nilAPI, containers.New(nil), containers.New(&gophercloud.ServiceClient{})} {
			if result, err := api.GetMetadata(context.Background(), containerMetadataName, option); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("nil source result=%+v err=%v", result, err)
			}
		}
		if result, err := containers.New(client).GetMetadata(nil, containerMetadataName, option); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil context result=%+v err=%v", result, err)
		}
		cause := errors.New("cancelled by caller")
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(cause)
		if result, err := containers.New(client).GetMetadata(ctx, containerMetadataName, option); result != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
			t.Fatalf("cancel result=%+v err=%v", result, err)
		}
		if callbacks != 0 || calls != 0 {
			t.Fatalf("callbacks=%d HTTP=%d", callbacks, calls)
		}
		wrong := *client
		wrong.Type = "image"
		if result, err := containers.New(&wrong).GetMetadata(context.Background(), containerMetadataName, option); result != nil || !errors.Is(err, resource.ErrUnsupported) || callbacks != 0 || calls != 0 {
			t.Fatalf("wrong service result=%+v err=%v callbacks=%d HTTP=%d", result, err, callbacks, calls)
		}
		ctx, cancel = context.WithCancelCause(context.Background())
		optionCause := errors.New("callback failed")
		if _, err := containers.New(client).GetMetadata(ctx, containerMetadataName, func(*containers.GetMetadataOpts) error { cancel(cause); return optionCause }); !errors.Is(err, optionCause) || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
			t.Fatalf("joined callback error=%v", err)
		}
		if _, err := containers.New(client).GetMetadata(context.Background(), containerMetadataName, nil); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatalf("nil callback err=%v HTTP=%d", err, calls)
		}
	})
	t.Run("input and protected headers validate before HTTP", func(t *testing.T) {
		calls := 0
		client := containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return containerMetadataResponse(204, ""), nil })
		api := containers.New(client)
		callbacks := 0
		for _, name := range []string{"", ".", "..", "a/b", "a\\b", "a\n", "a\t", "a\x7f", string([]byte{255})} {
			result, err := api.GetMetadata(context.Background(), name, func(*containers.GetMetadataOpts) error { callbacks++; return nil })
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
				t.Fatalf("name=%q result=%+v err=%v callbacks=%d", name, result, err, callbacks)
			}
			if name == "" {
				var typed swift.ErrEmptyContainerName
				if !errors.As(err, &typed) {
					t.Fatalf("native empty cause=%v", err)
				}
			}
			if name == "a/b" {
				var typed swift.ErrInvalidContainerName
				if !errors.As(err, &typed) {
					t.Fatalf("native slash cause=%v", err)
				}
			}
			if result, err := api.SetMetadata(context.Background(), name, nil); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("set name=%q result=%+v err=%v", name, result, err)
			}
			if result, err := api.DeleteMetadata(context.Background(), name, nil); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("delete name=%q result=%+v err=%v", name, result, err)
			}
		}
		for _, metadata := range []map[string]string{{"": "value"}, {"x-container-meta-book": "v"}, {"Book": "1", "book": "2"}, {"bad key": "v"}, {"한글": "v"}, {"bad:colon": "v"}, {"Book": "a\nb"}, {"Book": "\x7f"}, {"Book": string([]byte{255})}} {
			if result, err := api.SetMetadata(context.Background(), containerMetadataName, metadata); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("metadata=%v result=%+v err=%v", metadata, result, err)
			}
		}
		for _, keys := range [][]string{{""}, {"Book", "book"}, {"Book", "Book"}, {"X-CONTAINER-META-Book"}, {"a/b"}} {
			if result, err := api.DeleteMetadata(context.Background(), containerMetadataName, keys); result != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("keys=%q result=%+v err=%v", keys, result, err)
			}
		}
		for _, key := range []string{"Authorization", "x-auth-token", "Host", "Content-Length", "Transfer-Encoding", "Connection", "Proxy-Connection", "Proxy-Authorization", "Upgrade", "Trailer", "TE", "X-Newest", "X-Container-Meta-Book", "X-Remove-Container-Meta-Book"} {
			client.MoreHeaders = map[string]string{key: "owned"}
			if _, err := api.GetMetadata(context.Background(), containerMetadataName); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("source header %s err=%v", key, err)
			}
			client.MoreHeaders = nil
			if _, err := api.SetMetadata(context.Background(), containerMetadataName, nil, containers.WithMetadataHeader(key, "owned")); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("option header %s err=%v", key, err)
			}
		}
		for _, headers := range []map[string]string{{"X-Trace": "1", "x-trace": "2"}, {"bad key": "v"}, {"X-Trace": "\r"}} {
			if _, err := api.GetMetadata(context.Background(), containerMetadataName, containers.WithGetMetadataHeaders(headers)); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("headers=%v err=%v", headers, err)
			}
		}
		if calls != 0 {
			t.Fatalf("HTTP before validation=%d", calls)
		}
	})
	t.Run("factories and caller payload own snapshots", func(t *testing.T) {
		headers := map[string]string{"x-trace": "initial"}
		newest := false
		getFull := containers.WithGetMetadataOpts(containers.GetMetadataOpts{Headers: headers, Newest: &newest})
		getHeaders := containers.WithGetMetadataHeaders(headers)
		mutationFull := containers.WithMetadataOpts(containers.MetadataOpts{Headers: headers})
		mutationHeaders := containers.WithMetadataHeaders(headers)
		headers["x-trace"], newest = "mutated", true
		calls := 0
		client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("X-Trace") != "final" || (r.Method == "HEAD" && r.Header.Get("X-Newest") != "false") {
				t.Errorf("headers=%v", r.Header)
			}
			return containerMetadataResponse(204, ""), nil
		})
		if _, err := containers.New(client).GetMetadata(context.Background(), containerMetadataName, getHeaders, getFull, func(o *containers.GetMetadataOpts) error {
			if o.Headers["x-trace"] != "initial" || o.Newest == nil || *o.Newest {
				t.Errorf("factory snapshot=%+v", o)
			}
			return nil
		}, containers.WithGetMetadataHeader("X-Trace", "final")); err != nil {
			t.Fatal(err)
		}
		payload := map[string]string{"Book": "before callback"}
		client.ProviderClient.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("X-Trace") != "final" || r.Header.Get("X-Container-Meta-Book") != "before callback" {
				t.Errorf("set snapshot=%v", r.Header)
			}
			return containerMetadataResponse(204, ""), nil
		})
		if _, err := containers.New(client).SetMetadata(context.Background(), containerMetadataName, payload, mutationHeaders, mutationFull, func(o *containers.MetadataOpts) error {
			if o.Headers["x-trace"] != "initial" {
				t.Errorf("mutation factory snapshot=%+v", o)
			}
			payload["Book"] = "after callback"
			return nil
		}, containers.WithMetadataHeader("X-Trace", "final")); err != nil {
			t.Fatal(err)
		}
		keys := []string{"Book"}
		client.ProviderClient.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if !reflect.DeepEqual(r.Header.Values("X-Container-Meta-Book"), []string{""}) || r.Header.Get("X-Container-Meta-Mutated") != "" {
				t.Errorf("delete snapshot=%v", r.Header)
			}
			return containerMetadataResponse(204, ""), nil
		})
		if _, err := containers.New(client).DeleteMetadata(context.Background(), containerMetadataName, keys, func(*containers.MetadataOpts) error { keys[0] = "Mutated"; return nil }); err != nil || calls != 3 {
			t.Fatalf("err=%v calls=%d", err, calls)
		}
	})
	t.Run("copy after every callback and whole replacement", func(t *testing.T) {
		retained := map[string]string{"X-Trace": "owned"}
		pointer := false
		callbacks := 0
		client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Trace") != "owned" || r.Header.Get("X-Newest") != "false" || r.Header.Get("X-Discarded") != "" {
				t.Errorf("retained callback alias=%v", r.Header)
			}
			return containerMetadataResponse(204, ""), nil
		})
		_, err := containers.New(client).GetMetadata(context.Background(), containerMetadataName, containers.WithGetMetadataHeader("X-Discarded", "old"), func(o *containers.GetMetadataOpts) error {
			callbacks++
			*o = containers.GetMetadataOpts{Headers: retained, Newest: &pointer}
			return nil
		}, func(o *containers.GetMetadataOpts) error {
			callbacks++
			retained["X-Trace"], pointer = "stale alias", true
			return nil
		})
		if err != nil || callbacks != 2 {
			t.Fatalf("err=%v callbacks=%d", err, callbacks)
		}
		client.ProviderClient.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Discarded") != "" || len(r.Header.Values("X-Newest")) != 0 {
				t.Errorf("full reset headers=%v", r.Header)
			}
			return containerMetadataResponse(204, ""), nil
		})
		if _, err := containers.New(client).GetMetadata(context.Background(), containerMetadataName, containers.WithGetMetadataNewest(true), containers.WithGetMetadataHeader("X-Discarded", "old"), containers.WithGetMetadataOpts(containers.GetMetadataOpts{})); err != nil {
			t.Fatal(err)
		}
	})
}

func TestContainerMetadataContractsAtomicHeaders(t *testing.T) {
	t.Run("atomic malformed and duplicate canonical headers", func(t *testing.T) {
		cases := []http.Header{
			{"X-Container-Meta-": {"empty suffix"}}, {"X-Container-Meta-Bad Key": {"v"}}, {"X-Container-Meta-K": {"must not lowercase into ASCII"}}, {"X-Container-Meta-Book": {"1"}, "x-container-meta-book": {"2"}},
			{"X-Container-Meta-Book": {}}, {"X-Container-Meta-Book": {"1", "2"}}, {"X-Container-Meta-Book": {string([]byte{255})}},
			{"X-Container-Bytes-Used": {}}, {"X-Container-Bytes-Used": {"1", "2"}}, {"X-Container-Bytes-Used": {"1"}, "x-container-bytes-used": {"1"}},
			{"X-Timestamp": {}}, {"X-Timestamp": {"one", "two"}}, {"X-Timestamp": {"\x00"}},
			{"Last-Modified": {}}, {"Last-Modified": {"one", "two"}}, {"Last-Modified": {"one"}, "last-modified": {"two"}}, {"Last-Modified": {"\n"}}, {"X-Container-Object-Count": {"1.2"}},
		}
		for _, value := range []string{"", " 1", "1 ", "1.0", "0x10", "9223372036854775808", "-9223372036854775809"} {
			cases = append(cases, http.Header{"X-Container-Bytes-Used": {value}})
		}
		for i, headers := range cases {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				body := &containerMetadataBody{Reader: strings.NewReader("raw-proof")}
				client := containerMetadataClient(func(*http.Request) (*http.Response, error) {
					response := containerMetadataWire(204, body)
					for key, values := range headers {
						response.Header[key] = values
					}
					return response, nil
				})
				result, err := containers.New(client).GetMetadata(context.Background(), containerMetadataName)
				if result == nil || result.Metadata != nil || err == nil || result.StatusCode != 204 || string(result.Body) != "raw-proof" || body.closes.Load() != 1 {
					t.Fatalf("result=%+v err=%v closes=%d", result, err, body.closes.Load())
				}
				containerMetadataProof(t, err, 204, "raw-proof")
				if headers.Get("X-Container-Bytes-Used") == "9223372036854775808" {
					var parse *strconv.NumError
					if !errors.As(err, &parse) {
						t.Fatalf("lost strconv cause=%v", err)
					}
				}
			})
		}
	})
	t.Run("success typed projection owns raw storage", func(t *testing.T) {
		header := http.Header{"X-Container-Meta-Book": {"original"}, "X-Container-Bytes-Used": {"3"}, "X-Timestamp": {"raw timestamp"}, "Last-Modified": {"raw last modified"}, "X-Vendor": {"one", "two"}}
		client := containerMetadataClient(func(*http.Request) (*http.Response, error) {
			response := containerMetadataResponse(204, "body")
			for key, values := range header {
				response.Header[key] = values
			}
			return response, nil
		})
		result, err := containers.New(client).GetMetadata(context.Background(), containerMetadataName)
		if err != nil {
			t.Fatal(err)
		}
		header["X-Container-Meta-Book"][0] = "wire changed"
		header["X-Vendor"][0] = "wire changed"
		result.Header["X-Container-Meta-Book"][0] = "raw changed"
		result.Header["X-Container-Bytes-Used"][0] = "99"
		result.Header["X-Timestamp"][0] = "raw changed"
		result.Header["Last-Modified"][0] = "raw changed"
		if result.Metadata.Values["book"] != "original" || *result.Metadata.BytesUsed != 3 || *result.Metadata.Timestamp != "raw timestamp" || *result.Metadata.LastModified != "raw last modified" || result.Header["X-Vendor"][0] != "one" || string(result.Body) != "body" {
			t.Fatalf("aliased result=%+v metadata=%+v", result, result.Metadata)
		}
	})
	t.Run("read close custom cancellation retained once", func(t *testing.T) {
		readCause, closeCause, cancelCause := errors.New("HEAD read"), errors.New("HEAD close"), errors.New("custom cancellation")
		ctx, cancel := context.WithCancelCause(context.Background())
		body := &containerMetadataBody{Reader: containerMetadataReader(func(p []byte) (int, error) { return copy(p, "partial"), readCause }), closeErr: closeCause, onClose: func() { cancel(cancelCause) }}
		calls, retries := 0, 0
		client := containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return containerMetadataWire(204, body), nil })
		client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			retries++
			return nil
		}
		result, err := containers.New(client).GetMetadata(ctx, containerMetadataName)
		if result == nil || result.Metadata != nil || string(result.Body) != "partial" || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) || calls != 1 || retries != 0 || body.closes.Load() != 1 {
			t.Fatalf("result=%+v err=%v calls=%d retries=%d closes=%d", result, err, calls, retries, body.closes.Load())
		}
		proof := containerMetadataProof(t, err, 204, "partial")
		result.Body[0] = 'X'
		result.Header.Set("X-Trans-Id", "changed")
		if string(proof.Body) != "partial" || proof.Header.Get("X-Trans-Id") != "actual-container" {
			t.Fatalf("proof aliases result=%+v", proof)
		}
	})
	t.Run("unknown repeated quota dates remain passive", func(t *testing.T) {
		client := containerMetadataClient(func(*http.Request) (*http.Response, error) {
			response := containerMetadataResponse(204, "")
			response.Header["X-Container-Meta-Quota-Bytes"] = []string{"not a counter"}
			response.Header["Date"] = []string{"invalid", "repeated"}
			response.Header["X-Storage-Policy"] = []string{"future", "future2"}
			response.Header["X-Container-Read"] = []string{"invalid ACL", "repeated"}
			response.Header["X-Versions-Enabled"] = []string{"not a boolean"}
			response.Header["Last-Modified"] = []string{"  literal date \t"}
			response.Header["X-Timestamp"] = []string{"  literal %2F\t"}
			return response, nil
		})
		result, err := containers.New(client).GetMetadata(context.Background(), containerMetadataName)
		if err != nil || result.Metadata.Values["quota-bytes"] != "not a counter" || *result.Metadata.Timestamp != "  literal %2F\t" || len(result.Header["Date"]) != 2 || len(result.Header["X-Storage-Policy"]) != 2 || *result.Metadata.LastModified != "  literal date \t" || len(result.Header["X-Container-Read"]) != 2 {
			t.Fatalf("result=%+v err=%v", result, err)
		}
	})
}

func TestContainerMetadataContractsMutationEvidence(t *testing.T) {
	t.Run("accepted opaque POST read close context evidence", func(t *testing.T) {
		for _, deletion := range []bool{false, true} {
			t.Run(strconv.FormatBool(deletion), func(t *testing.T) {
				readCause, closeCause, cause := errors.New("POST read"), errors.New("POST close"), errors.New("POST cancelled")
				ctx, cancel := context.WithCancelCause(context.Background())
				body := &containerMetadataBody{Reader: containerMetadataReader(func(p []byte) (int, error) { return copy(p, "\xffopaque"), readCause }), closeErr: closeCause, onClose: func() { cancel(cause) }}
				calls := 0
				client := containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return containerMetadataWire(204, body), nil })
				api := containers.New(client)
				var result *containers.MetadataResponse
				var err error
				if deletion {
					result, err = api.DeleteMetadata(ctx, containerMetadataName, []string{"Book"})
				} else {
					result, err = api.SetMetadata(ctx, containerMetadataName, map[string]string{"Book": "value"})
				}
				if result == nil || result.StatusCode != 204 || string(result.Body) != "\xffopaque" || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || calls != 1 || body.closes.Load() != 1 {
					t.Fatalf("result=%+v err=%v calls=%d closes=%d", result, err, calls, body.closes.Load())
				}
				proof := containerMetadataProof(t, err, 204, "\xffopaque")
				result.Body[0] = 'X'
				result.Header.Set("X-Trans-Id", "changed")
				if string(proof.Body) != "\xffopaque" || proof.Header.Get("X-Trans-Id") != "actual-container" {
					t.Fatalf("ack proof aliased=%+v", proof)
				}
			})
		}
	})
	t.Run("native failures never create ACK or suppress missing", func(t *testing.T) {
		for _, status := range []int{200, 201, 202, 401, 404, 503} {
			t.Run(strconv.Itoa(status), func(t *testing.T) {
				calls := 0
				body := &containerMetadataBody{Reader: strings.NewReader(`{"error":"actual"}`)}
				client := containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return containerMetadataWire(status, body), nil })
				result, err := containers.New(client).DeleteMetadata(context.Background(), containerMetadataName, []string{"Book"})
				var native gophercloud.ErrUnexpectedResponseCode
				if result != nil || !errors.As(err, &native) || native.Actual != status || !reflect.DeepEqual(native.Expected, []int{204}) || native.Method != "POST" || native.URL != containerMetadataEndpoint || calls != 1 || body.closes.Load() != 1 {
					t.Fatalf("result=%+v err=%v native=%+v calls=%d closes=%d", result, err, native, calls, body.closes.Load())
				}
			})
		}
		nested := &gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{204}, Method: "POST", URL: containerMetadataEndpoint}
		calls, hooks := 0, 0
		nestedClient := containerMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			return containerMetadataResponse(503, `{"error":"original"}`), nil
		})
		nestedClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			hooks++
			return nested
		}
		if result, err := containers.New(nestedClient).DeleteMetadata(context.Background(), containerMetadataName, nil); result != nil || !errors.Is(err, nested) || !gophercloud.ResponseCodeIs(err, 503) || calls != 1 || hooks != 1 {
			t.Fatalf("nested404 result=%+v err=%v calls=%d hooks=%d", result, err, calls, hooks)
		}
		transportCause := errors.New("transport failed")
		client := containerMetadataClient(func(*http.Request) (*http.Response, error) { return nil, transportCause })
		if result, err := containers.New(client).SetMetadata(context.Background(), containerMetadataName, nil); result != nil || !errors.Is(err, transportCause) {
			t.Fatalf("transport result=%+v err=%v", result, err)
		}
		client = containerMetadataClient(func(*http.Request) (*http.Response, error) {
			return containerMetadataResponse(200, "not accepted HEAD"), nil
		})
		result, err := containers.New(client).GetMetadata(context.Background(), containerMetadataName)
		var native gophercloud.ErrUnexpectedResponseCode
		if result != nil || !errors.As(err, &native) || native.Actual != 200 || native.Method != "HEAD" || !reflect.DeepEqual(native.Expected, []int{204}) {
			t.Fatalf("HEAD200 result=%+v err=%v native=%+v", result, err, native)
		}
	})
	t.Run("accepted source failure retains raw proof without typed metadata", func(t *testing.T) {
		for _, head := range []bool{false, true} {
			t.Run(strconv.FormatBool(head), func(t *testing.T) {
				var client *gophercloud.ServiceClient
				body := &containerMetadataBody{Reader: strings.NewReader("already applied"), onClose: func() { client.ResourceBase = containerMetadataBase }}
				client = containerMetadataClient(func(*http.Request) (*http.Response, error) {
					response := containerMetadataWire(204, body)
					response.Header.Set("X-Container-Meta-Book", "valid")
					return response, nil
				})
				api := containers.New(client)
				if head {
					result, err := api.GetMetadata(context.Background(), containerMetadataName)
					if result == nil || result.Metadata != nil || !errors.Is(err, resource.ErrInvalidOption) || body.closes.Load() != 1 {
						t.Fatalf("HEAD=%+v err=%v closes=%d", result, err, body.closes.Load())
					}
					containerMetadataProof(t, err, 204, "already applied")
				} else {
					result, err := api.SetMetadata(context.Background(), containerMetadataName, nil)
					if result == nil || !errors.Is(err, resource.ErrInvalidOption) || body.closes.Load() != 1 {
						t.Fatalf("POST=%+v err=%v closes=%d", result, err, body.closes.Load())
					}
					containerMetadataProof(t, err, 204, "already applied")
				}
			})
		}
	})
	t.Run("POST is raw only and never refreshed projection", func(t *testing.T) {
		calls := 0
		wireHeader := http.Header{"X-Container-Meta-Book": {"one", "two"}, "X-Container-Bytes-Used": {"not numeric"}}
		client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method != "POST" {
				t.Errorf("unexpected refresh=%s", r.Method)
			}
			response := containerMetadataResponse(204, "not JSON")
			for key, values := range wireHeader {
				response.Header[key] = values
			}
			return response, nil
		})
		result, err := containers.New(client).SetMetadata(context.Background(), containerMetadataName, map[string]string{"Book": "submitted"})
		wireHeader["X-Container-Meta-Book"][0] = "mutated"
		if err != nil || result == nil || calls != 1 || string(result.Body) != "not JSON" || !reflect.DeepEqual(result.Header["X-Container-Meta-Book"], []string{"one", "two"}) {
			t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
		}
	})
}

func TestContainerMetadataContractsSourceAndNativePolicy(t *testing.T) {
	t.Run("captured headers live token and request isolation", func(t *testing.T) {
		client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Auth-Token") != "latest-token" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Trace") != "option" || r.Header.Get("Accept") != "source/accept" || r.Header.Get("Content-Type") != "source/type" || r.URL.String() != containerMetadataEndpoint {
				t.Errorf("request headers=%v URL=%s", r.Header, r.URL)
			}
			return containerMetadataResponse(204, ""), nil
		})
		original := client.ProviderClient
		client.MoreHeaders = map[string]string{"X-Source": "captured", "X-Trace": "source", "Accept": "source/accept", "Content-Type": "source/type"}
		_, err := containers.New(client).GetMetadata(context.Background(), containerMetadataName, containers.WithGetMetadataHeader("X-Trace", "option"), func(*containers.GetMetadataOpts) error {
			client.MoreHeaders["X-Source"] = "later"
			original.SetToken("latest-token")
			return nil
		})
		if err != nil || client.MoreHeaders["X-Trace"] != "source" || client.ProviderClient != original {
			t.Fatalf("err=%v source=%v", err, client.MoreHeaders)
		}
		client.ProviderClient.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Source") != "later" || r.Header.Get("X-Trace") != "source" || len(r.Header.Values("X-Newest")) != 0 || len(r.Header.Values("X-Container-Meta-Book")) != 0 {
				t.Errorf("future request leaked previous headers=%v", r.Header)
			}
			return containerMetadataResponse(204, ""), nil
		})
		if _, err := containers.New(client).GetMetadata(context.Background(), containerMetadataName); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("fixed identity and current header validation", func(t *testing.T) {
		for _, mutate := range []func(*gophercloud.ServiceClient){
			func(c *gophercloud.ServiceClient) { c.Endpoint += "/other" },
			func(c *gophercloud.ServiceClient) { c.ResourceBase = containerMetadataBase },
			func(c *gophercloud.ServiceClient) { c.Type = "image" },
			func(c *gophercloud.ServiceClient) { c.Microversion = "changed" },
			func(c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} },
			func(c *gophercloud.ServiceClient) { c.MoreHeaders = map[string]string{"X-Container-Meta-Owned": "bad"} },
		} {
			calls := 0
			client := containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return containerMetadataResponse(204, ""), nil })
			result, err := containers.New(client).GetMetadata(context.Background(), containerMetadataName, func(*containers.GetMetadataOpts) error { mutate(client); return nil })
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatalf("result=%+v err=%v HTTP=%d", result, err, calls)
			}
		}
		for _, tc := range []struct{ endpoint, base, version string }{
			{"https://swift.invalid/a?query=1", "", ""}, {"https://swift.invalid/a#fragment", "", ""},
			{"ftp://swift.invalid/a/", "", ""}, {"https://user@swift.invalid/a/", "", ""},
			{"https://swift.invalid/a", "", ""}, {containerMetadataBase, "https://other.invalid/a/", ""},
			{containerMetadataBase, "https://swift.invalid/a", ""}, {containerMetadataBase, "https://swift.invalid/a/?", ""},
			{containerMetadataBase, "https://swift.invalid/a/?query=1", ""}, {containerMetadataBase, "https://swift.invalid/a/#fragment", ""},
			{"https://swift.invalid/a?query=1", containerMetadataBase, ""}, {containerMetadataBase, "", "\n"},
		} {
			calls, callbacks := 0, 0
			client := containerMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
			client.Endpoint, client.ResourceBase, client.Microversion = tc.endpoint, tc.base, tc.version
			result, err := containers.New(client).GetMetadata(context.Background(), containerMetadataName, func(*containers.GetMetadataOpts) error { callbacks++; return nil })
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 {
				t.Fatalf("source=%+v result=%+v err=%v calls=%d callbacks=%d", tc, result, err, calls, callbacks)
			}
		}
	})
	t.Run("native prebody retries and ownership guard", func(t *testing.T) {
		for _, change := range []struct {
			name string
			fn   func(*gophercloud.RequestOpts)
		}{
			{"keep body", func(o *gophercloud.RequestOpts) { o.KeepResponseBody = false }},
			{"JSON response", func(o *gophercloud.RequestOpts) { o.JSONResponse = new(any) }},
			{"raw body", func(o *gophercloud.RequestOpts) { o.RawBody = strings.NewReader("changed") }},
			{"JSON null", func(o *gophercloud.RequestOpts) { o.JSONBody = json.RawMessage("null") }},
			{"encoding failure", func(o *gophercloud.RequestOpts) { o.JSONBody = make(chan int) }},
		} {
			t.Run(change.name, func(t *testing.T) {
				calls, hooks := 0, 0
				body := &containerMetadataBody{Reader: strings.NewReader(`{"error":"first"}`)}
				client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
					calls++
					containerMetadataNoBody(t, r)
					return containerMetadataWire(503, body), nil
				})
				client.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
					hooks++
					change.fn(o)
					return nil
				}
				result, err := containers.New(client).SetMetadata(context.Background(), containerMetadataName, nil)
				var native gophercloud.ErrUnexpectedResponseCode
				if result != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || calls != 1 || hooks != 1 || body.closes.Load() != 1 {
					t.Fatalf("result=%+v err=%v calls=%d hooks=%d closes=%d", result, err, calls, hooks, body.closes.Load())
				}
				if change.name == "encoding failure" {
					var encoding *json.UnsupportedTypeError
					if !errors.As(err, &encoding) {
						t.Fatalf("encoding cause lost=%v", err)
					}
				}
			})
		}
		calls, hooks := 0, 0
		client := containerMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return containerMetadataResponse(503, `{"error":"retry"}`), nil
			}
			if r.Header.Get("X-Advanced") != "provider-owned" || r.Header.Get("X-Container-Meta-Book") != "advanced-native" {
				t.Errorf("advanced header=%v", r.Header)
			}
			return containerMetadataResponse(204, ""), nil
		})
		original := client.ProviderClient
		client.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			hooks++
			o.MoreHeaders = map[string]string{"X-Advanced": "provider-owned", "X-Container-Meta-Book": "advanced-native"}
			return nil
		}
		if _, err := containers.New(client).SetMetadata(context.Background(), containerMetadataName, nil); err != nil || calls != 2 || hooks != 1 || client.ProviderClient != original || client.MoreHeaders != nil {
			t.Fatalf("retry err=%v calls=%d hooks=%d source=%v", err, calls, hooks, client.MoreHeaders)
		}
		calls, hooks = 0, 0
		transportCause := errors.New("retry transport")
		client = containerMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return nil, transportCause
			}
			return containerMetadataResponse(204, ""), nil
		})
		client.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			hooks++
			if !errors.Is(err, transportCause) {
				t.Errorf("original transport cause=%v", err)
			}
			return nil
		}
		if _, err := containers.New(client).GetMetadata(context.Background(), containerMetadataName); err != nil || calls != 2 || hooks != 1 {
			t.Fatalf("transport retry err=%v calls=%d hooks=%d", err, calls, hooks)
		}
	})
	t.Run("actual status gate reauth and native ABI", func(t *testing.T) {
		calls := 0
		closeCause := errors.New("expanded code close")
		body := &containerMetadataBody{Reader: strings.NewReader("unexpected accepted body"), closeErr: closeCause}
		client := containerMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return containerMetadataResponse(503, `{"error":"retry"}`), nil
			}
			return containerMetadataWire(202, body), nil
		})
		client.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			o.OkCodes = []int{202}
			return nil
		}
		result, err := containers.New(client).SetMetadata(context.Background(), containerMetadataName, nil)
		var native gophercloud.ErrUnexpectedResponseCode
		if result != nil || !errors.As(err, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, []int{204}) || string(native.Body) != "unexpected accepted body" || native.ResponseHeader.Get("X-Trans-Id") != "actual-container" || !errors.Is(err, closeCause) || calls != 2 || body.closes.Load() != 1 {
			t.Fatalf("result=%+v err=%v native=%+v calls=%d", result, err, native, calls)
		}
		calls, reauth := 0, 0
		client = containerMetadataClient(func(r *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return containerMetadataResponse(401, `{"error":"expired"}`), nil
			}
			if r.Header.Get("X-Auth-Token") != "reauthenticated" {
				t.Errorf("token=%v", r.Header)
			}
			return containerMetadataResponse(204, ""), nil
		})
		client.ReauthFunc = func(context.Context) error { reauth++; client.SetToken("reauthenticated"); return nil }
		if _, err := containers.New(client).GetMetadata(context.Background(), containerMetadataName); err != nil || calls != 2 || reauth != 1 {
			t.Fatalf("reauth err=%v calls=%d reauth=%d", err, calls, reauth)
		}
		reauthCause := errors.New("reauth rejected")
		client = containerMetadataClient(func(*http.Request) (*http.Response, error) {
			return containerMetadataResponse(401, `{"error":"expired"}`), nil
		})
		client.ReauthFunc = func(context.Context) error { return reauthCause }
		_, err = containers.New(client).GetMetadata(context.Background(), containerMetadataName)
		var reauthError *gophercloud.ErrUnableToReauthenticate
		if !errors.As(err, &reauthError) || reauthError.ErrReauth != reauthCause || !errors.As(reauthError.ErrOriginal, &native) || native.Actual != 401 {
			t.Fatalf("native reauth fields=%+v err=%v", reauthError, err)
		}
		calls, redirects := 0, 0
		client = containerMetadataClient(func(*http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				response := containerMetadataResponse(307, "")
				response.Header.Set("Location", containerMetadataEndpoint)
				return response, nil
			}
			return containerMetadataResponse(204, ""), nil
		})
		client.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects++; return nil }
		if _, err := containers.New(client).GetMetadata(context.Background(), containerMetadataName); err != nil || calls != 2 || redirects != 1 {
			t.Fatalf("same-target redirect err=%v calls=%d redirects=%d", err, calls, redirects)
		}
		for _, tc := range []struct {
			status   int
			location string
			post     bool
		}{
			{307, "https://other.invalid/stolen", false},
			{307, containerMetadataBase + "other", false},
			{302, containerMetadataEndpoint, true},
		} {
			calls = 0
			client = containerMetadataClient(func(*http.Request) (*http.Response, error) {
				calls++
				response := containerMetadataResponse(tc.status, "")
				response.Header.Set("Location", tc.location)
				return response, nil
			})
			var err error
			if tc.post {
				_, err = containers.New(client).SetMetadata(context.Background(), containerMetadataName, nil)
			} else {
				_, err = containers.New(client).GetMetadata(context.Background(), containerMetadataName)
			}
			if !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
				t.Fatalf("redirect=%+v err=%v calls=%d", tc, err, calls)
			}
		}
		client = containerMetadataClient(func(r *http.Request) (*http.Response, error) {
			status := 204
			if r.Method == "POST" {
				status = 202
			}
			response := containerMetadataResponse(status, "")
			response.Header.Set("Date", "Fri, 17 Jan 2014 16:09:56 GMT")
			response.Header.Set("X-Timestamp", "123.5")
			response.Header.Set("X-Container-Meta-Native", "preserved")
			return response, nil
		})
		api := containers.New(client)
		var get *containers.GetHeader
		get, err = api.Get(context.Background(), containerMetadataName, containers.WithGetOptions(containers.GetOpts{Newest: true}))
		if err != nil || get == nil {
			t.Fatalf("native Get=%+v err=%v", get, err)
		}
		var update *containers.UpdateHeader
		update, err = api.Update(context.Background(), containerMetadataName, containers.UpdateOpts{Metadata: map[string]string{"Book": "native"}}, containers.WithUpdateHeader("X-Trace", "native"))
		if err != nil || update == nil {
			t.Fatalf("native202 Update=%+v err=%v", update, err)
		}
		record, err := api.Resources.Get(context.Background(), containerMetadataName)
		if err != nil || record == nil || record.Name != containerMetadataName || record.Details == nil || record.Details.Timestamp != 123.5 || record.Metadata["Native"] != "preserved" {
			t.Fatalf("native Resource=%+v err=%v", record, err)
		}
		client.ProviderClient.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) {
			response := containerMetadataResponse(200, "")
			response.Header.Set("Date", "Fri, 17 Jan 2014 16:09:56 GMT")
			return response, nil
		})
		get, err = api.Get(context.Background(), containerMetadataName)
		if err != nil || get == nil {
			t.Fatalf("native HEAD200=%+v err=%v", get, err)
		}
		if client.Endpoint != containerMetadataBase {
			t.Fatal("native client changed")
		}
	})
}
