package containers

import (
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

func TestContainerTempURLKeyCoreWireAndStatus(t *testing.T) {
	for _, tc := range []struct {
		name, key string
		secondary bool
		status    int
	}{
		{"primary", "  책%20\tkey  ", false, 204}, {"secondary empty", "", true, 204},
		{"reject201", "key", false, 201}, {"reject202", "key", false, 202},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := containerMetadataClient("http://cloud.invalid/reverse%20proxy/v1/AUTH_x/")
			c.ResourceBase = "http://cloud.invalid/data%25/v1/AUTH_x/"
			calls := 0
			c.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "POST" || r.URL.EscapedPath() != "/data%25/v1/AUTH_x/"+url.PathEscape("백업 %2F?#") || r.URL.RawQuery != "" {
					t.Fatalf("wrong request %s %s", r.Method, r.URL)
				}
				if r.Body != nil {
					data, err := io.ReadAll(r.Body)
					if err != nil || len(data) != 0 {
						t.Fatalf("unexpected body %q %v", data, err)
					}
				}
				name := "X-Container-Meta-Temp-URL-Key"
				if tc.secondary {
					name += "-2"
				}
				if values := r.Header.Values(name); len(values) != 1 || values[0] != tc.key {
					t.Fatalf("key transformed/missing: %#v", r.Header)
				}
				other := "X-Container-Meta-Temp-URL-Key-2"
				if tc.secondary {
					other = "X-Container-Meta-Temp-URL-Key"
				}
				if len(r.Header.Values(other)) != 0 || len(r.Header.Values("X-Account-Meta-Temp-URL-Key")) != 0 {
					t.Fatal("key/prefix leaked", r.Header)
				}
				return containerMetadataWire(r, tc.status, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("ack"))), nil
			})
			result, err := New(c).SetTempURLKey(context.Background(), "백업 %2F?#", tc.key, WithSetTempURLKeySecondary(tc.secondary))
			if calls != 1 {
				t.Fatal("implicit refresh/replay", calls)
			}
			if tc.status != 204 {
				if result != nil || !gophercloud.ResponseCodeIs(err, tc.status) {
					t.Fatalf("unexpected status treated as success: %#v %v", result, err)
				}
				return
			}
			if err != nil || result == nil || result.StatusCode != 204 || string(result.Body) != "ack" || result.Header.Get("X-Proof") != "kept" {
				t.Fatalf("lost acknowledgement: %#v %v", result, err)
			}
		})
	}
}

func TestContainerTempURLKeyCorePreflightAndCallbackCause(t *testing.T) {
	for _, key := range []string{"bad\nkey", "bad\rkey", "bad\x00key", "bad\x7fkey", string([]byte{255})} {
		c := containerMetadataClient("http://cloud.invalid/v1/AUTH_x/")
		calls, callbacks := 0, 0
		c.HTTPClient.Transport = containerMetadataTransport(func(*http.Request) (*http.Response, error) { calls++; t.Fatal("invalid key sent"); return nil, nil })
		result, err := New(c).SetTempURLKey(context.Background(), "백업 %2F?#", key, func(*SetTempURLKeyOpts) error { callbacks++; return nil })
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 {
			t.Fatalf("key preflight failed %#v %v %d/%d", result, err, calls, callbacks)
		}
	}
	c := containerMetadataClient("http://cloud.invalid/v1/AUTH_x/")
	callbacks := 0
	if result, err := New(c).SetTempURLKey(nil, "백업 %2F?#", "key", func(*SetTempURLKeyOpts) error { callbacks++; return nil }); result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
		t.Fatal(result, err, callbacks)
	}
	cause, optionErr := errors.New("cancel cause"), errors.New("callback cause")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	result, err := New(c).SetTempURLKey(ctx, "백업 %2F?#", "key", func(*SetTempURLKeyOpts) error { c.Endpoint += "changed/"; cancel(cause); return optionErr })
	for _, expected := range []error{optionErr, cause, context.Canceled, resource.ErrInvalidOption} {
		if !errors.Is(err, expected) {
			t.Fatalf("lost cause %v: %v", expected, err)
		}
	}
	if result != nil {
		t.Fatal("invented response", result)
	}
}

func TestContainerTempURLKeyCoreAcceptedFailuresAndNativeHooks(t *testing.T) {
	t.Run("read close context and source", func(t *testing.T) {
		c := containerMetadataClient("http://cloud.invalid/v1/AUTH_x/")
		readErr, closeErr, cause := errors.New("read"), errors.New("close"), errors.New("cancel")
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		body := &containerMetadataBody{Reader: containerMetadataReader(func(p []byte) (int, error) { return copy(p, "x"), readErr }), closeErr: closeErr, onClose: func() { c.Endpoint += "changed/"; cancel(cause) }}
		c.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) {
			return containerMetadataWire(r, 204, http.Header{"X-Proof": {"kept"}}, body), nil
		})
		result, err := New(c).SetTempURLKey(ctx, "백업 %2F?#", "key")
		if result == nil || result.StatusCode != 204 || string(result.Body) != "x" || body.closes.Load() != 1 {
			t.Fatalf("lost actual response %#v %v closes=%d", result, err, body.closes.Load())
		}
		containerMetadataProof(t, err, 204, "x")
		for _, expected := range []error{readErr, closeErr, cause, context.Canceled, resource.ErrInvalidOption} {
			if !errors.Is(err, expected) {
				t.Fatalf("lost %v: %v", expected, err)
			}
		}
		var proof *resource.ResponseError
		if !errors.As(err, &proof) {
			t.Fatal(err)
		}
		result.Body[0] = '!'
		result.Header.Set("X-Proof", "changed")
		if string(proof.Body) != "x" || proof.Header.Get("X-Proof") != "kept" {
			t.Fatal("result/error evidence aliases")
		}
	})
	t.Run("native original503 callback404", func(t *testing.T) {
		c := containerMetadataClient("http://cloud.invalid/v1/AUTH_x/")
		calls := 0
		nested := &gophercloud.ErrUnexpectedResponseCode{Actual: 404}
		c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return nested }
		c.HTTPClient.Transport = containerMetadataTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			return containerMetadataWire(r, 503, http.Header{}, io.NopCloser(strings.NewReader("native"))), nil
		})
		result, err := New(c).SetTempURLKey(context.Background(), "백업 %2F?#", "key")
		if result != nil || calls != 1 || !errors.Is(err, nested) || !gophercloud.ResponseCodeIs(err, 503) {
			t.Fatalf("native cause lost %#v %v %d", result, err, calls)
		}
	})
}
