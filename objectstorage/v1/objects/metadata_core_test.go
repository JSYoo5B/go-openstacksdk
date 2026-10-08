package objects

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const objectMetadataContainer = "컨테이너 %?#:"
const objectMetadataName = "폴더/file %2F?#:"

type objectMetadataTransport func(*http.Request) (*http.Response, error)

func (f objectMetadataTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type objectMetadataBody struct {
	io.Reader
	closeErr error
	closes   atomic.Int32
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
func objectMetadataClient() *gophercloud.ServiceClient {
	p := &gophercloud.ProviderClient{TokenID: "token"}
	p.UseTokenLock()
	return &gophercloud.ServiceClient{ProviderClient: p, Endpoint: "http://swift.invalid/v1/AUTH_a/", Type: "object-store"}
}
func objectMetadataWire(r *http.Request, code int, h http.Header, body io.ReadCloser) *http.Response {
	return &http.Response{Request: r, StatusCode: code, Header: h, Body: body}
}
func objectMetadataOK(r *http.Request, code int, h http.Header) *http.Response {
	return objectMetadataWire(r, code, h, io.NopCloser(strings.NewReader("")))
}
func objectMetadataProof(t *testing.T, err error, code int, body string) *resource.ResponseError {
	t.Helper()
	var p *resource.ResponseError
	if !errors.As(err, &p) || p.StatusCode != code || string(p.Body) != body || p.Header.Get("X-Proof") != "kept" {
		t.Fatalf("missing actual response: %v %#v", err, p)
	}
	return p
}

func TestObjectMetadataCoreRoutesAndMerge(t *testing.T) {
	var calls atomic.Int32
	state := map[string]string{"keep": "untouched", "replace": "old", "remove": "gone"}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if r.URL.EscapedPath() != "/reverse%20proxy/v1/AUTH_a/"+url.PathEscape(objectMetadataContainer)+"/"+url.PathEscape(objectMetadataName) || r.Header.Get("X-Auth-Token") != "token" {
			t.Errorf("wrong route/auth %s %#v", r.RequestURI, r.Header)
		}
		if data, _ := io.ReadAll(r.Body); len(data) != 0 {
			t.Errorf("unexpected request body %q", data)
		}
		if r.Method == "HEAD" {
			if n == 1 {
				if r.URL.RawQuery != "" || r.Header.Get("X-Newest") != "false" || r.Header.Get("X-Call") != "get" {
					t.Errorf("getter %#v", r)
				}
			} else if r.URL.RawQuery != "symlink=get" || r.Header.Get("X-Write") != "" || r.Header.Get("X-Newest") != "" {
				t.Errorf("mutation read %#v", r)
			}
			for k, v := range state {
				w.Header().Set("X-Object-Meta-"+k, v)
			}
			w.Header().Set("Content-Type", "application/observed")
			w.Header().Set("Content-Encoding", "identity")
			w.Header().Set("ETag", "opaque etag")
			w.Header().Set("X-Delete-At", "raw timestamp")
			w.Header().Set("Last-Modified", "not a time")
			w.Header().Set("X-Proof", "kept")
			w.WriteHeader(200)
		} else {
			if r.Method != "POST" || r.URL.RawQuery != "" || r.Header.Get("Content-Encoding") != "identity" || r.Header.Get("ETag") != "" || r.Header.Get("Last-Modified") != "" || r.Header.Get("X-Delete-At") != "raw timestamp" {
				t.Errorf("POST carry %#v", r)
			}
			if n == 3 && (r.Header.Get("Content-Type") != "application/written" || r.Header.Get("X-Write") != "yes") {
				t.Errorf("POST override %#v", r.Header)
			}
			replacement := map[string]string{}
			for key, values := range r.Header {
				lower := strings.ToLower(key)
				if strings.HasPrefix(lower, "x-object-meta-") {
					replacement[strings.TrimPrefix(lower, "x-object-meta-")] = values[0]
				}
				if strings.HasPrefix(lower, "x-account-meta-") || strings.HasPrefix(lower, "x-container-meta-") || strings.HasPrefix(lower, "x-remove-object-meta-") {
					t.Errorf("foreign/remove prefix %s", key)
				}
			}
			state = replacement
			w.WriteHeader(202)
		}
	}))
	defer s.Close()
	c := objectMetadataClient()
	c.Endpoint = s.URL + "/unused/"
	c.ResourceBase = s.URL + "/reverse%20proxy/v1/AUTH_a/"
	c.MoreHeaders = map[string]string{"Content-Type": "application/source", "X-Source": "owned"}
	a := New(c)
	get, err := a.GetMetadata(context.Background(), objectMetadataContainer, objectMetadataName, WithGetMetadataNewest(false), WithGetMetadataHeader("X-Call", "get"))
	if err != nil || get.Metadata.Values["keep"] != "untouched" || *get.Metadata.LastModified != "not a time" {
		t.Fatalf("GET %#v %v", get, err)
	}
	change, err := a.SetMetadata(context.Background(), objectMetadataContainer, objectMetadataName, map[string]string{"Replace": "new", "Empty": ""}, WithMetadataHeaders(map[string]string{"Content-Type": "application/written", "X-Write": "yes"}))
	if err != nil || change.Before.Metadata.Values["replace"] != "old" || change.Before.Metadata.Values["empty"] != "" || change.Acknowledgement.StatusCode != 202 || state["replace"] != "new" || state["keep"] != "untouched" {
		t.Fatalf("merge %#v state%#v %v", change, state, err)
	}
	if _, ok := state["empty"]; !ok {
		t.Fatal("literal empty value lost")
	}
	removed, err := a.DeleteMetadata(context.Background(), objectMetadataContainer, objectMetadataName, []string{"REMOVE"})
	if err != nil || removed.Before.Metadata.Values["remove"] != "gone" {
		t.Fatalf("delete %#v %v", removed, err)
	}
	if _, ok := state["remove"]; ok || state["keep"] != "untouched" || calls.Load() != 5 || c.MoreHeaders["Content-Type"] != "application/source" {
		t.Fatalf("wrong replacement/state or hidden request %#v calls%d source%#v", state, calls.Load(), c.MoreHeaders)
	}
}

func TestObjectMetadataCoreEmptyWritesAndNineHeaders(t *testing.T) {
	observed := http.Header{"Content-Type": {""}, "Content-Encoding": {"gzip"}, "Content-Disposition": {"literal%20"}, "X-Delete-At": {"-1"}, "X-Object-Manifest": {"container/prefix"}, "Cache-Control": {"public"}, "Content-Language": {"ko"}, "Expires": {"not-date"}, "X-Robots-Tag": {"none"}, "X-Object-Meta-Keep": {""}, "ETag": {"etag"}, "X-Timestamp": {"raw"}, "Content-Location": {"foreign"}, "X-Static-Large-Object": {"true"}, "X-Unknown": {"one", "two"}}
	for _, remove := range []bool{false, true} {
		for _, empty := range []bool{false, true} {
			t.Run(strconv.FormatBool(remove)+strconv.FormatBool(empty), func(t *testing.T) {
				c := objectMetadataClient()
				calls := 0
				c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					if calls == 1 {
						return objectMetadataOK(r, 200, observed), nil
					}
					if calls != 2 || r.Method != "POST" || r.URL.RawQuery != "" {
						t.Errorf("unexpected request %d %s", calls, r.URL)
					}
					for _, k := range []string{"Content-Type", "Content-Encoding", "Content-Disposition", "X-Delete-At", "X-Object-Manifest", "Cache-Control", "Content-Language", "Expires", "X-Robots-Tag", "X-Object-Meta-Keep"} {
						if values, ok := r.Header[k]; !ok || len(values) != 1 || values[0] != observed[k][0] {
							t.Errorf("lost literal observed %s: %#v", k, r.Header)
						}
					}
					for _, k := range []string{"ETag", "X-Timestamp", "Content-Location", "X-Static-Large-Object", "X-Unknown"} {
						if len(r.Header.Values(k)) != 0 {
							t.Errorf("copied readonly/unknown %s", k)
						}
					}
					return objectMetadataOK(r, 202, http.Header{}), nil
				})
				var got *MetadataChangeResult
				var err error
				if remove {
					var keys []string
					if !empty {
						keys = []string{"absent"}
					}
					got, err = New(c).DeleteMetadata(context.Background(), "container", "object", keys)
				} else {
					var m map[string]string
					if !empty {
						m = map[string]string{}
					}
					got, err = New(c).SetMetadata(context.Background(), "container", "object", m)
				}
				if err != nil || got.Before == nil || got.Acknowledgement == nil || calls != 2 {
					t.Fatalf("empty/absent no-op %#v %v calls%d", got, err, calls)
				}
			})
		}
	}
}

func TestObjectMetadataCoreAtomicProjection(t *testing.T) {
	for i, h := range []http.Header{{"X-Object-Meta-K": {"bad"}}, {"X-Object-Meta-Key": {"a"}, "x-object-meta-key": {"b"}}, {"X-Object-Meta-Key": {}}, {"X-Object-Meta-": {"bad"}}, {"Content-Length": {"9223372036854775808"}}, {"Content-Length": {" 1"}}, {"Content-Encoding": {"one", "two"}}, {"ETag": {"bad\x00"}}} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			h.Set("X-Proof", "kept")
			body := &objectMetadataBody{Reader: strings.NewReader("raw")}
			c := objectMetadataClient()
			calls := 0
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				return objectMetadataWire(r, 200, h, body), nil
			})
			got, err := New(c).SetMetadata(context.Background(), "container", "object", nil)
			if got == nil || got.Before.Metadata != nil || got.Acknowledgement != nil || calls != 1 || body.closes.Load() != 1 {
				t.Fatalf("non-atomic %#v %v", got, err)
			}
			proof := objectMetadataProof(t, err, 200, "raw")
			got.Before.Header.Set("X-Proof", "changed")
			got.Before.Body[0] = 'X'
			if proof.Header.Get("X-Proof") != "kept" || string(proof.Body) != "raw" {
				t.Fatal("evidence aliases result")
			}
			if i == 4 {
				var e *strconv.NumError
				if !errors.As(err, &e) {
					t.Fatalf("lost numeric cause %v", err)
				}
			}
		})
	}
	c := objectMetadataClient()
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		return objectMetadataOK(r, 200, http.Header{"Content-Length": {"-2"}, "X-Object-Meta-Temp-URL-Key": {"literal%20"}, "Last-Modified": {""}, "X-Unknown": {"one", "two"}}), nil
	})
	got, err := New(c).GetMetadata(context.Background(), "container", "object")
	if err != nil || *got.Metadata.ContentLength != -2 || got.Metadata.LastModified == nil || *got.Metadata.LastModified != "" || got.Metadata.Timestamp != nil {
		t.Fatalf("presence %#v %v", got, err)
	}
	got.Header.Set("X-Object-Meta-Temp-URL-Key", "changed")
	if got.Metadata.Values["temp-url-key"] != "literal%20" {
		t.Fatal("projection aliases raw")
	}
}

func TestObjectMetadataCoreSymlinkAndStrictStatus(t *testing.T) {
	for _, h := range []http.Header{{"X-Symlink-Target": {""}}, {"x-symlink-target": {"container/object?version-id=x"}}, {"X-Symlink-Target-Bytes": {"one", "two"}}, {"X-Symlink-Target-Account": {"bad\n"}}} {
		h.Set("X-Proof", "kept")
		c := objectMetadataClient()
		calls := 0
		c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) { calls++; return objectMetadataOK(r, 200, h), nil })
		got, err := New(c).DeleteMetadata(context.Background(), "container", "object", nil)
		if err == nil || got == nil || got.Before.Metadata != nil || got.Acknowledgement != nil || calls != 1 {
			t.Fatalf("link accepted %#v %v", got, err)
		}
		objectMetadataProof(t, err, 200, "")
		if len(h.Values("X-Symlink-Target")) != 0 && !errors.Is(err, resource.ErrUnsupported) {
			t.Fatalf("link unsupported cause %v", err)
		}
	}
	for _, code := range []int{201, 204, 404} {
		c := objectMetadataClient()
		c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) { return objectMetadataOK(r, code, http.Header{}), nil })
		got, err := New(c).GetMetadata(context.Background(), "container", "object")
		if got != nil || !gophercloud.ResponseCodeIs(err, code) {
			t.Fatalf("unexpected HEAD accepted%d %#v %v", code, got, err)
		}
	}
	for _, code := range []int{200, 201, 204, 404} {
		c := objectMetadataClient()
		calls := 0
		c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return objectMetadataOK(r, 200, http.Header{"X-Object-Meta-Key": {"old"}}), nil
			}
			return objectMetadataOK(r, code, http.Header{}), nil
		})
		got, err := New(c).SetMetadata(context.Background(), "container", "object", nil)
		if got == nil || got.Before.Metadata.Values["key"] != "old" || got.Acknowledgement != nil || !gophercloud.ResponseCodeIs(err, code) || calls != 2 {
			t.Fatalf("unexpected POST accepted%d %#v %v", code, got, err)
		}
	}
}

func TestObjectMetadataCorePhaseFailuresAndCauses(t *testing.T) {
	for _, phase := range []string{"HEAD", "POST"} {
		t.Run(phase, func(t *testing.T) {
			readErr := errors.New("read")
			closeErr := errors.New("close")
			cause := errors.New("custom cancel")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			body := &objectMetadataBody{Reader: objectMetadataReader(func(p []byte) (int, error) { return copy(p, "raw"), readErr }), closeErr: closeErr, onClose: func() { cancel(cause) }}
			c := objectMetadataClient()
			calls := 0
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != phase {
					return objectMetadataOK(r, 200, http.Header{"X-Object-Meta-Key": {"old"}}), nil
				}
				code := 200
				if phase == "POST" {
					code = 202
				}
				return objectMetadataWire(r, code, http.Header{"X-Proof": {"kept"}}, body), nil
			})
			got, err := New(c).SetMetadata(ctx, "container", "object", map[string]string{"Key": "new"})
			for _, e := range []error{readErr, closeErr, context.Canceled, cause} {
				if !errors.Is(err, e) {
					t.Fatalf("lost %v: %v", e, err)
				}
			}
			code := 200
			if phase == "POST" {
				code = 202
				if got.Acknowledgement == nil || got.Before.Metadata.Values["key"] != "old" || string(got.Acknowledgement.Body) != "raw" || calls != 2 {
					t.Fatalf("POST proof %#v", got)
				}
			} else if got.Before.Metadata != nil || got.Acknowledgement != nil || calls != 1 {
				t.Fatalf("HEAD proof %#v", got)
			}
			if body.closes.Load() != 1 {
				t.Fatalf("body closes%d", body.closes.Load())
			}
			objectMetadataProof(t, err, code, "raw")
		})
	}
}

func TestObjectMetadataCoreSourceAndLiveAuth(t *testing.T) {
	c := objectMetadataClient()
	var calls atomic.Int32
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		n := calls.Add(1)
		if n == 1 {
			if r.Header.Get("X-Auth-Token") != "token" {
				t.Error("initial auth")
			}
			return objectMetadataWire(r, 200, http.Header{"X-Object-Meta-Keep": {"old"}}, &objectMetadataBody{Reader: strings.NewReader(""), onClose: func() { c.SetToken("fresh") }}), nil
		}
		if r.Header.Get("X-Auth-Token") != "fresh" {
			t.Error("stale auth")
		}
		return objectMetadataOK(r, 202, http.Header{}), nil
	})
	if _, err := New(c).SetMetadata(context.Background(), "container", "object", nil); err != nil || calls.Load() != 2 {
		t.Fatalf("live auth %v", err)
	}
	for _, retarget := range []func(*gophercloud.ServiceClient){func(c *gophercloud.ServiceClient) { c.Endpoint = "http://swift.invalid/other/" }, func(c *gophercloud.ServiceClient) { c.ResourceBase = "http://swift.invalid/other/" }, func(c *gophercloud.ServiceClient) { c.Microversion = "changed" }, func(c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} }, func(c *gophercloud.ServiceClient) { c.MoreHeaders = map[string]string{"X-Object-Meta-Foreign": "bad"} }} {
		c := objectMetadataClient()
		body := &objectMetadataBody{Reader: strings.NewReader("raw"), onClose: func() { retarget(c) }}
		calls := 0
		c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			return objectMetadataWire(r, 200, http.Header{"X-Proof": {"kept"}}, body), nil
		})
		got, err := New(c).SetMetadata(context.Background(), "container", "object", nil)
		if got == nil || got.Before.Metadata != nil || got.Acknowledgement != nil || calls != 1 || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("source changed %#v %v", got, err)
		}
		objectMetadataProof(t, err, 200, "raw")
	}
}
