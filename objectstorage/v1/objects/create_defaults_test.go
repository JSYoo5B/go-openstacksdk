package objects_test

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestNativeObjectCreateEmptyDefaults(t *testing.T) {
	cases := []struct {
		name    string
		opts    objects.CreateOpts
		options []objects.CreateOption
		etag    string
	}{
		{name: "default", etag: "d41d8cd98f00b204e9800998ecf8427e"},
		{name: "explicit no etag", opts: objects.CreateOpts{NoETag: true}},
		{name: "explicit etag", opts: objects.CreateOpts{ETag: "caller", NoETag: true}, etag: "caller"},
		{name: "final option clears body", opts: objects.CreateOpts{Content: strings.NewReader("unused")}, options: []objects.CreateOption{objects.WithCreateOptions(objects.CreateOpts{})}, etag: "d41d8cd98f00b204e9800998ecf8427e"},
		{name: "nil byte option", options: []objects.CreateOption{objects.WithCreateBytes(nil)}, etag: "d41d8cd98f00b204e9800998ecf8427e"},
		{name: "present empty bytes", options: []objects.CreateOption{objects.WithCreateBytes([]byte{})}, etag: "d41d8cd98f00b204e9800998ecf8427e"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, code := range []int{http.StatusCreated, http.StatusAccepted} {
				var calls []nativeObjectCall
				api := nativeObjectAPI(t, &calls, func(*http.Request) nativeObjectReply {
					return nativeObjectReply{code, http.Header{"Etag": {"server-etag"}, "X-Trans-Id": {"actual-put"}}, "ack"}
				})
				opts := append([]objects.CreateOption{}, tc.options...)
				opts = append(opts, objects.WithCreateQuery("version-id", "v+1"), objects.WithCreateHeader("X-Extra", "extension"))
				result, err := api.Create(context.Background(), "my box", "dir/file?", tc.opts, opts...)
				if err != nil || result == nil || result.ETag != "server-etag" || result.TransID != "actual-put" {
					t.Fatalf("result=%+v err=%v", result, err)
				}
				if len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != nativeObjectPath || calls[0].body != "" || calls[0].query != "version-id=v%2B1" || calls[0].header.Get("Etag") != tc.etag || calls[0].header.Get("X-Extra") != "extension" {
					t.Fatalf("calls=%+v", calls)
				}
			}
		})
	}
}

func TestNativeObjectCreateBytesSnapshotAndOptionOrder(t *testing.T) {
	var calls []nativeObjectCall
	api := nativeObjectAPI(t, &calls, func(*http.Request) nativeObjectReply { return nativeObjectReply{201, nil, ""} })
	data := []byte("snapshot")
	option := objects.WithCreateBytes(data)
	data[0] = 'X'
	for range 2 {
		if _, err := api.Create(context.Background(), "c", "o", objects.CreateOpts{Content: strings.NewReader("ignored")}, option); err != nil {
			t.Fatal(err)
		}
	}
	sum := fmt.Sprintf("%x", md5.Sum([]byte("snapshot")))
	if len(calls) != 2 || calls[0].body != "snapshot" || calls[1].body != "snapshot" || calls[0].header.Get("Etag") != sum || calls[1].header.Get("Etag") != sum {
		t.Fatalf("calls=%+v", calls)
	}
	if _, err := api.Create(context.Background(), "c", "o", objects.CreateOpts{}, option, objects.WithCreateOptions(objects.CreateOpts{Content: strings.NewReader("last"), NoETag: true})); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 3 || calls[2].body != "last" || calls[2].header.Get("Etag") != "" {
		t.Fatalf("calls=%+v", calls)
	}
}

func TestNativeObjectCreateTypedNilReadersAndPreflight(t *testing.T) {
	var typedNil *bytes.Reader
	for _, tc := range []struct {
		name string
		opts objects.CreateOpts
	}{
		{"checksum", objects.CreateOpts{Content: typedNil}},
		{"no checksum", objects.CreateOpts{Content: typedNil, NoETag: true}},
		{"supplied checksum", objects.CreateOpts{Content: typedNil, ETag: "caller"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []nativeObjectCall
			api := nativeObjectAPI(t, &calls, func(*http.Request) nativeObjectReply { return nativeObjectReply{201, nil, ""} })
			result, err := api.Create(context.Background(), "c", "o", tc.opts)
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
				t.Fatalf("result=%+v err=%v calls=%v", result, err, calls)
			}
			nativeObjectOperation(t, err, "Create")
			if _, err := api.Create(context.Background(), "c", "o", tc.opts, objects.WithCreateBytes([]byte("valid"))); err != nil {
				t.Fatal(err)
			}
			if len(calls) != 1 || calls[0].body != "valid" {
				t.Fatal(calls)
			}
		})
	}
	for _, options := range [][]objects.CreateOption{{nil}, {objects.WithCreateHeader("Content-Type", "collision")}} {
		var calls []nativeObjectCall
		api := nativeObjectAPI(t, &calls, func(*http.Request) nativeObjectReply { return nativeObjectReply{201, nil, ""} })
		_, err := api.Create(context.Background(), "c", "o", objects.CreateOpts{}, options...)
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatalf("err=%v calls=%v", err, calls)
		}
		nativeObjectOperation(t, err, "Create")
	}
}

type nativeCreateErrorReader struct {
	cause  error
	closes int
}

func (r *nativeCreateErrorReader) Read([]byte) (int, error) { return 0, r.cause }
func (r *nativeCreateErrorReader) Close() error             { r.closes++; return nil }

type nativeCreateSeekErrorReader struct {
	cause  error
	reader *strings.Reader
}

func (r *nativeCreateSeekErrorReader) Read(p []byte) (int, error)     { return r.reader.Read(p) }
func (r *nativeCreateSeekErrorReader) Seek(int64, int) (int64, error) { return 0, r.cause }

func TestNativeObjectCreateDefaultReaderErrorsRetainCause(t *testing.T) {
	readCause, seekCause := errors.New("read cause"), errors.New("seek cause")
	reader := &nativeCreateErrorReader{cause: readCause}
	for _, tc := range []struct {
		name   string
		reader io.Reader
		cause  error
	}{
		{"read", reader, readCause},
		{"seek", &nativeCreateSeekErrorReader{cause: seekCause, reader: strings.NewReader("value")}, seekCause},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []nativeObjectCall
			api := nativeObjectAPI(t, &calls, func(*http.Request) nativeObjectReply { return nativeObjectReply{201, nil, ""} })
			_, err := api.Create(context.Background(), "c", "o", objects.CreateOpts{Content: tc.reader})
			if !errors.Is(err, tc.cause) || len(calls) != 0 {
				t.Fatalf("err=%v calls=%v", err, calls)
			}
			nativeObjectOperation(t, err, "Create")
		})
	}
	if reader.closes != 0 {
		t.Fatalf("preflight reader was closed %d times", reader.closes)
	}
}

func TestNativeObjectCreateEmptyNativeRetryAndStatus(t *testing.T) {
	var calls []nativeObjectCall
	api := nativeObjectAPI(t, &calls, func(*http.Request) nativeObjectReply {
		if len(calls) == 1 {
			return nativeObjectReply{503, nil, "first rejected"}
		}
		return nativeObjectReply{201, http.Header{"Etag": {"second"}}, ""}
	})
	hooks := 0
	api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		hooks++
		return nil
	}
	result, err := api.Create(context.Background(), "c", "o", objects.CreateOpts{})
	if err != nil || result == nil || result.ETag != "second" || len(calls) != 2 || hooks != 1 {
		t.Fatalf("result=%+v err=%v calls=%+v hooks=%d", result, err, calls, hooks)
	}
	for _, call := range calls {
		if call.body != "" || call.method != http.MethodPut || call.header.Get("Etag") != "d41d8cd98f00b204e9800998ecf8427e" {
			t.Fatal(call)
		}
	}
	calls = nil
	api = nativeObjectAPI(t, &calls, func(*http.Request) nativeObjectReply {
		return nativeObjectReply{204, http.Header{"Etag": {"rejected"}}, ""}
	})
	_, err = api.Create(context.Background(), "c", "o", objects.CreateOpts{})
	var rejected gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &rejected) || rejected.Actual != 204 || len(calls) != 1 {
		t.Fatalf("err=%v calls=%v", err, calls)
	}
	nativeObjectOperation(t, err, "Create")
}
