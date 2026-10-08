package objects

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestDirectoryMarkerCoreWireAndMetadata(t *testing.T) {
	for _, code := range []int{201, 202, 404} {
		c := objectMetadataClient()
		c.ResourceBase = "http://swift.invalid/reverse%20proxy/v1/AUTH_a/"
		c.MoreHeaders = map[string]string{"content-type": "source/type", "X-Source": "captured"}
		calls := 0
		c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			data, err := io.ReadAll(r.Body)
			if err != nil || len(data) != 0 || r.Body != http.NoBody || r.ContentLength != 0 || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 {
				t.Error("marker framing", data, err, r.Body, r.ContentLength, r.TransferEncoding)
			}
			want := "/reverse%20proxy/v1/AUTH_a/" + url.PathEscape(objectMetadataContainer) + "/" + url.PathEscape(objectMetadataName)
			if r.Method != "PUT" || r.URL.EscapedPath() != want || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "token" || r.Header.Get("Content-Type") != "application/directory" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Object-Meta-Tag") != "literal" || r.Header.Get("X-Object-Meta-Empty") != "" || r.Header.Get("ETag") != "" {
				t.Error("marker route or headers", r.Method, r.URL, r.Header)
			}
			if _, ok := r.Header["X-Object-Meta-Empty"]; !ok {
				t.Error("empty metadata lost")
			}
			return objectMetadataWire(r, code, http.Header{"X-Proof": {"kept"}, "Last-Modified": {"unparsed"}}, io.NopCloser(strings.NewReader("raw"))), nil
		})
		result, err := New(c).CreateDirectoryMarkerObject(context.Background(), objectMetadataContainer, objectMetadataName, WithDirectoryMarkerHeader("Content-Type", "caller/type"), WithDirectoryMarkerMetadata(map[string]string{"Tag": "literal", "Empty": ""}))
		if result == nil || calls != 1 || result.Source != "bytes" || result.Mode != "ordinary" || result.Size != 0 || result.MD5 != "" || result.SHA256 != "" || result.Capabilities != nil || result.Discovery != nil || result.Manifest != nil || len(result.Segments) != 0 || len(result.Cleanup) != 0 || result.SegmentPrefix != "" || result.Skipped || result.Ordinary == nil || len(result.Ordinary.Attempts) != 1 {
			t.Fatal("marker performed another workflow", result, err, calls)
		}
		if code == 404 {
			if !gophercloud.ResponseCodeIs(err, 404) || result.Ordinary.Acknowledgement != nil || result.Ordinary.Attempts[0].Response.StatusCode != 404 {
				t.Fatal("missing container was created or acknowledged", result, err)
			}
		} else if err != nil || result.Ordinary.Acknowledgement == nil || result.Ordinary.Acknowledgement.StatusCode != code || string(result.Ordinary.Acknowledgement.Body) != "raw" {
			t.Fatal(result, err)
		}
		if c.MoreHeaders["content-type"] != "source/type" {
			t.Fatal("source media was mutated", c.MoreHeaders)
		}
	}
}

func TestDirectoryMarkerCoreAcceptedFaultProof(t *testing.T) {
	for _, mode := range []string{"read", "close", "context", "endpoint", "detect"} {
		t.Run(mode, func(t *testing.T) {
			c := objectMetadataClient()
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("accepted marker fault")
			calls, hooks := 0, 0
			originalEndpoint := c.Endpoint
			body := &readTestBody{data: []byte("raw")}
			switch mode {
			case "read":
				body.readErr = cause
			case "close":
				body.closeErr = cause
			case "context":
				body.afterRead = func() { cancel(cause) }
			case "endpoint":
				body.afterRead = func() { c.Endpoint = "http://other.invalid/" }
				body.afterClose = func() { c.Endpoint = originalEndpoint }
			case "detect":
				body.afterRead = func() { c.MoreHeaders = map[string]string{"x-detect-content-type": "true"} }
				body.afterClose = func() { c.MoreHeaders = nil }
			}
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				return objectMetadataWire(r, 201, http.Header{"X-Proof": {"kept"}}, body), nil
			})
			c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				hooks++
				return nil
			}
			result, err := New(c).CreateDirectoryMarkerObject(ctx, "box", "key")
			if err == nil || result == nil || calls != 1 || hooks != 0 || result.Ordinary.Acknowledgement == nil || result.Ordinary.Acknowledgement.StatusCode != 201 || len(result.Ordinary.Attempts) != 1 || result.Ordinary.Attempts[0].Error == nil || body.closes != 1 {
				t.Fatal("accepted fault lost evidence or retried", result, err, calls, hooks, body.closes)
			}
			if mode == "read" || mode == "close" || mode == "context" {
				if !errors.Is(err, cause) {
					t.Fatal("fault cause lost", err)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("source fault lost", err)
			}
			proof := objectMetadataProof(t, err, 201, "raw")
			result.Ordinary.Acknowledgement.Body[0] = '!'
			result.Ordinary.Acknowledgement.Header.Set("X-Proof", "changed")
			result.Ordinary.Attempts[0].Response.Body[0] = '?'
			if string(proof.Body) != "raw" || proof.Header.Get("X-Proof") != "kept" {
				t.Fatal("error evidence aliases result", proof)
			}
		})
	}
}

func TestDirectoryMarkerCoreNativeReplay(t *testing.T) {
	for _, mode := range []string{"retry", "backoff", "reauth", "redirect"} {
		t.Run(mode, func(t *testing.T) {
			c := objectMetadataClient()
			c.MoreHeaders = map[string]string{"Content-Type": "source/type", "X-Source": "captured"}
			calls, hooks := 0, 0
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "PUT" || r.Body != http.NoBody || r.ContentLength != 0 || len(r.TransferEncoding) != 0 || r.Header.Get("Content-Type") != "application/directory" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Object-Meta-Tag") != "owned" {
					t.Error("physical marker replay changed", r.Method, r.Header, r.Body, r.ContentLength)
				}
				code := 201
				headers := http.Header{"X-Proof": {"kept"}}
				if calls == 1 {
					switch mode {
					case "retry":
						code = 503
					case "backoff":
						code = 429
					case "reauth":
						code = 401
					case "redirect":
						code = 307
						headers.Set("Location", r.URL.String())
					}
				}
				if mode == "reauth" && calls == 2 && r.Header.Get("X-Auth-Token") != "fresh" {
					t.Error("live authentication lost", r.Header)
				}
				return objectMetadataWire(r, code, headers, io.NopCloser(strings.NewReader("raw"))), nil
			})
			switch mode {
			case "retry":
				c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks++
					c.MoreHeaders["Content-Type"] = "valid/drift"
					c.MoreHeaders["X-Source"] = "valid drift"
					return nil
				}
			case "backoff":
				c.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error { hooks++; return nil }
			case "reauth":
				c.ReauthFunc = func(context.Context) error { hooks++; c.SetToken("fresh"); return nil }
			case "redirect":
				c.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { hooks++; return nil }
			}
			result, err := New(c).CreateDirectoryMarkerObject(context.Background(), "box", "key", WithDirectoryMarkerMetadataValue("Tag", "owned"))
			if err != nil || calls != 2 || hooks != 1 || result == nil || result.Ordinary.Acknowledgement == nil || result.Ordinary.Acknowledgement.StatusCode != 201 || len(result.Ordinary.Attempts) != 2 {
				t.Fatal("native marker replay", result, err, calls, hooks)
			}
			if mode != "redirect" && result.Ordinary.Attempts[0].Error == nil {
				t.Fatal("recovered attempt cause lost")
			}
			result.Ordinary.Acknowledgement.Body[0] = '!'
			if string(result.Ordinary.Attempts[1].Response.Body) != "raw" {
				t.Fatal("attempt and acknowledgement alias")
			}
		})
	}
}

func TestDirectoryMarkerCoreNativeOwnership(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*gophercloud.RequestOpts)
	}{
		{"media replacement", func(o *gophercloud.RequestOpts) { o.MoreHeaders["Content-Type"] = "foreign/type" }},
		{"media removal", func(o *gophercloud.RequestOpts) { delete(o.MoreHeaders, "Content-Type") }},
		{"media alias", func(o *gophercloud.RequestOpts) { o.MoreHeaders["content-type"] = "application/directory" }},
		{"detect media", func(o *gophercloud.RequestOpts) { o.MoreHeaders["X-Detect-Content-Type"] = "true" }},
		{"raw body", func(o *gophercloud.RequestOpts) { o.RawBody = strings.NewReader("foreign") }},
		{"raw carrier", func(o *gophercloud.RequestOpts) { o.RawBody.(*bytes.Reader).Reset([]byte("foreign")) }},
		{"JSON body", func(o *gophercloud.RequestOpts) { o.JSONBody = map[string]string{"foreign": "body"} }},
		{"success codes", func(o *gophercloud.RequestOpts) { o.OkCodes = []int{201, 202, 503} }},
		{"metadata removal", func(o *gophercloud.RequestOpts) { delete(o.MoreHeaders, "X-Object-Meta-Tag") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := objectMetadataClient()
			calls, hooks := 0, 0
			cause := errors.New("native marker policy cause")
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				return objectMetadataWire(r, 503, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("rejected"))), nil
			})
			c.RetryFunc = func(ctx context.Context, method, target string, opts *gophercloud.RequestOpts, original error, count uint) error {
				hooks++
				tc.change(opts)
				return cause
			}
			result, err := New(c).CreateDirectoryMarkerObject(context.Background(), "box", "key", WithDirectoryMarkerMetadataValue("Tag", "owned"))
			if !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || result == nil || calls != 1 || hooks != 1 || result.Ordinary.Acknowledgement != nil || len(result.Ordinary.Attempts) != 1 || string(result.Ordinary.Attempts[0].Response.Body) != "rejected" {
				t.Fatal("owned marker policy replayed", result, err, calls, hooks)
			}
		})
	}
}

func TestDirectoryMarkerCoreRedirectOwnership(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*http.Request)
	}{
		{"media replacement", func(r *http.Request) { r.Header.Set("Content-Type", "foreign/type") }},
		{"media removal", func(r *http.Request) { r.Header.Del("Content-Type") }},
		{"media alias", func(r *http.Request) { r.Header["content-type"] = []string{"application/directory"} }},
		{"media multiple", func(r *http.Request) {
			r.Header["Content-Type"] = []string{"application/directory", "application/directory"}
		}},
		{"detect media", func(r *http.Request) { r.Header.Set("X-Detect-Content-Type", "false") }},
		{"body", func(r *http.Request) { r.Body = io.NopCloser(strings.NewReader("foreign")) }},
		{"length", func(r *http.Request) { r.ContentLength = 7 }},
		{"transfer", func(r *http.Request) { r.TransferEncoding = []string{"chunked"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := objectMetadataClient()
			calls := 0
			cause := errors.New("redirect marker policy cause")
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				return objectMetadataWire(r, 307, http.Header{"Location": {r.URL.String()}, "X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("redirect"))), nil
			})
			c.HTTPClient.CheckRedirect = func(r *http.Request, via []*http.Request) error { tc.change(r); return cause }
			result, err := New(c).CreateDirectoryMarkerObject(context.Background(), "box", "key")
			if !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || result == nil || calls != 1 || result.Ordinary.Acknowledgement != nil || len(result.Ordinary.Attempts) != 1 || result.Ordinary.Attempts[0].Response.StatusCode != 307 {
				t.Fatal("redirect marker ownership lost", result, err, calls)
			}
		})
	}
}

func TestDirectoryMarkerCoreOrdinaryPolicyIsolation(t *testing.T) {
	c := objectMetadataClient()
	c.MoreHeaders = map[string]string{"Content-Type": "source/type", "X-Detect-Content-Type": "true"}
	calls, callbacks := 0, 0
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Content-Type") != "ordinary/type" || r.Header.Get("X-Detect-Content-Type") != "true" || r.Header.Get("X-Object-Meta-Tag") != "ordinary" {
			t.Error("marker changed ordinary producer policy", r.Header)
		}
		return objectMetadataOK(r, 201, http.Header{}), nil
	})
	api := New(c)
	marker, err := api.CreateDirectoryMarkerObject(context.Background(), "box", "key", func(*DirectoryMarkerOpts) error { callbacks++; return nil })
	if marker != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 {
		t.Fatal("source detect reached marker callback or wire", marker, err, calls, callbacks)
	}
	ordinary, err := api.CreateObject(context.Background(), "box", "key", CreateObjectInput{Data: []byte{}}, WithCreateObjectHeader("Content-Type", "ordinary/type"), WithCreateObjectMetadataValue("Tag", "ordinary"))
	if err != nil || ordinary == nil || ordinary.Ordinary.Acknowledgement == nil || calls != 1 {
		t.Fatal("ordinary create inherited marker restriction", ordinary, err, calls)
	}
}
