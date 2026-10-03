package objects

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

func TestObjectReadStreamOwnershipAndCleanup(t *testing.T) {
	t.Run("EOF-close-error-and-caller-mutation", func(t *testing.T) {
		closeFault := errors.New("close-fault")
		body := &readTestBody{data: []byte("data"), closeErr: closeFault}
		api, _ := readTestAPI(func(*http.Request) (*http.Response, error) {
			return readTestResponse(206, http.Header{"X-Object-Meta-K": {"wire"}}, body), nil
		})
		result, err := api.StreamObject(context.Background(), "c", "o", WithObjectReadBufferSize(1))
		if err != nil || result == nil || result.Body == nil {
			t.Fatalf("open: %+v %v", result, err)
		}
		result.BytesRead = 999
		result.Complete = true
		result.NotModified = true
		result.StatusCode = 999
		result.Header.Set("X-Object-Meta-K", "caller")
		buffer := make([]byte, 8)
		n, readErr := result.Body.Read(buffer)
		if n != 4 || string(buffer[:n]) != "data" || !errors.Is(readErr, closeFault) || !result.Complete || result.NotModified || result.BytesRead != 4 || body.reads != 1 || body.closes != 1 {
			t.Fatalf("terminal: n=%d result=%+v err=%v body=%+v", n, result, readErr, body)
		}
		proof := readTestProof(t, readErr, 206)
		if proof.Body != nil || proof.Header.Get("X-Object-Meta-K") != "wire" {
			t.Fatal("mutable exported evidence corrupted proof")
		}
		againN, againErr := result.Body.Read(buffer)
		firstClose := result.Body.Close()
		secondClose := result.Body.Close()
		if againN != 0 || againErr != readErr || !errors.Is(firstClose, closeFault) || secondClose != firstClose || body.closes != 1 {
			t.Fatal("terminal/Close outcome not stable")
		}
	})
	t.Run("caller-buffer-and-zero-read", func(t *testing.T) {
		body := &readTestBody{data: []byte("abc")}
		api, _ := readTestAPI(func(*http.Request) (*http.Response, error) { return readTestResponse(200, http.Header{}, body), nil })
		result, err := api.StreamObject(context.Background(), "c", "o", WithObjectReadBufferSize(16*1024*1024))
		if err != nil {
			t.Fatal(err)
		}
		if n, e := result.Body.Read(nil); n != 0 || e != nil || body.reads != 0 {
			t.Fatal("zero-size read reached response body")
		}
		buffer := make([]byte, 1)
		n, e := result.Body.Read(buffer)
		if n != 1 || e != nil || result.BytesRead != 1 || result.Complete || body.reads != 1 {
			t.Fatalf("caller buffer ignored: %d %v %+v", n, e, result)
		}
		if e := result.Body.Close(); e != nil || body.closes != 1 {
			t.Fatalf("early close: %v", e)
		}
		if n, e := result.Body.Read(buffer); n != 0 || !errors.Is(e, io.ErrClosedPipe) {
			t.Fatalf("closed read: %d %v", n, e)
		}
		if e := result.Body.Close(); e != nil || body.closes != 1 {
			t.Fatal("double Close")
		}
	})
	t.Run("304-close-error-usable-empty-body", func(t *testing.T) {
		closeFault := errors.New("304-close")
		body := &readTestBody{data: []byte("illegal"), closeErr: closeFault}
		api, _ := readTestAPI(func(*http.Request) (*http.Response, error) { return readTestResponse(304, http.Header{}, body), nil })
		result, err := api.StreamObject(context.Background(), "c", "o")
		if result == nil || result.Body == nil || !result.NotModified || !result.Complete || !errors.Is(err, closeFault) || body.reads != 0 || body.closes != 1 {
			t.Fatalf("304: %+v %v", result, err)
		}
		if n, e := result.Body.Read(make([]byte, 2)); n != 0 || !errors.Is(e, closeFault) {
			t.Fatalf("closed 304 fault lost: %d %v", n, e)
		}
		if !errors.Is(result.Body.Close(), closeFault) || body.closes != 1 {
			t.Fatal("304 Close replayed")
		}
	})
}

func TestObjectReadStreamGuards(t *testing.T) {
	t.Run("EOF-with-custom-cancellation-cause", func(t *testing.T) {
		cause := errors.New("caller-cause")
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		body := &readTestBody{data: []byte("data"), afterRead: func() { cancel(cause) }}
		api, _ := readTestAPI(func(*http.Request) (*http.Response, error) { return readTestResponse(200, http.Header{}, body), nil })
		result, err := api.StreamObject(ctx, "c", "o")
		if err != nil {
			t.Fatal(err)
		}
		n, err := result.Body.Read(make([]byte, 4))
		if n != 4 || result.BytesRead != 4 || !result.Complete || result.Metadata == nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || body.closes != 1 {
			t.Fatalf("EOF observation/cause: %+v n=%d %v", result, n, err)
		}
		readTestProof(t, err, 200)
	})
	t.Run("read-header-reservation-after-open", func(t *testing.T) {
		body := &readTestBody{data: []byte("do-not-read")}
		api, source := readTestAPI(func(*http.Request) (*http.Response, error) { return readTestResponse(200, http.Header{}, body), nil })
		result, err := api.StreamObject(context.Background(), "c", "o")
		if err != nil {
			t.Fatal(err)
		}
		source.MoreHeaders = map[string]string{"If-Match": "injected"}
		n, err := result.Body.Read(make([]byte, 4))
		if n != 0 || !errors.Is(err, resource.ErrInvalidOption) || body.reads != 0 || body.closes != 1 || result.Metadata == nil || result.Complete {
			t.Fatalf("late read guard: %+v n=%d %v", result, n, err)
		}
	})
	t.Run("Close-checks-post-cleanup-source", func(t *testing.T) {
		body := &readTestBody{data: []byte("unused")}
		api, source := readTestAPI(func(*http.Request) (*http.Response, error) {
			return readTestResponse(200, http.Header{"X-Object-Meta-K": {"wire"}}, body), nil
		})
		body.afterClose = func() { source.MoreHeaders = map[string]string{"Cookie": "injected"} }
		result, err := api.StreamObject(context.Background(), "c", "o")
		if err != nil {
			t.Fatal(err)
		}
		result.Header.Set("X-Object-Meta-K", "caller")
		first := result.Body.Close()
		second := result.Body.Close()
		if !errors.Is(first, resource.ErrInvalidOption) || first != second || body.closes != 1 || body.reads != 0 || result.Complete {
			t.Fatalf("Close source guard: %+v %v %v", result, first, second)
		}
		if readTestProof(t, first, 200).Header.Get("X-Object-Meta-K") != "wire" {
			t.Fatal("Close proof uses exported header")
		}
	})
}

func TestObjectReadStreamBoundaries(t *testing.T) {
	for _, test := range []struct {
		name  string
		body  *readTestBody
		cause error
		reads int
	}{
		{"no-progress", &readTestBody{zero: true}, io.ErrNoProgress, 100},
		{"invalid-count", &readTestBody{invalid: true, readErr: io.EOF}, resource.ErrInvalidOption, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			api, _ := readTestAPI(func(*http.Request) (*http.Response, error) {
				return readTestResponse(200, http.Header{}, test.body), nil
			})
			result, err := api.StreamObject(context.Background(), "c", "o")
			if err != nil {
				t.Fatal(err)
			}
			var n int
			for i := 0; i < test.reads; i++ {
				n, err = result.Body.Read(make([]byte, 4))
				if err != nil {
					break
				}
			}
			if n != 0 || !errors.Is(err, test.cause) || test.body.reads != test.reads || test.body.closes != 1 || result.BytesRead != 0 || result.Complete {
				t.Fatalf("read boundary: %+v %d %v body=%+v", result, n, err, test.body)
			}
		})
	}
	t.Run("retry-body-ownership", func(t *testing.T) {
		calls := 0
		api, source := readTestAPI(func(*http.Request) (*http.Response, error) {
			calls++
			return readTestResponse(503, http.Header{}, io.NopCloser(strings.NewReader("original"))), nil
		})
		source.ProviderClient.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
			opts.JSONBody = map[string]string{"changed": "body"}
			return nil
		}
		result, err := api.GetObject(context.Background(), "c", "o")
		if result != nil || calls != 1 || !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) {
			t.Fatalf("unsafe retry: %+v %v calls=%d", result, err, calls)
		}
	})
	t.Run("expanded-native-code-no-read", func(t *testing.T) {
		calls := 0
		closeFault := errors.New("expanded-close")
		body := &readTestBody{data: []byte("not-owned"), closeErr: closeFault}
		api, source := readTestAPI(func(*http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return readTestResponse(503, http.Header{}, io.NopCloser(strings.NewReader("original"))), nil
			}
			return readTestResponse(205, http.Header{"X-Proof": {"expanded"}}, body), nil
		})
		source.ProviderClient.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
			opts.OkCodes = append(opts.OkCodes, 205)
			return nil
		}
		result, err := api.GetObject(context.Background(), "c", "o")
		var native gophercloud.ErrUnexpectedResponseCode
		if result != nil || !errors.As(err, &native) || native.Actual != 205 || len(native.Expected) != 3 || native.Body != nil || native.ResponseHeader.Get("X-Proof") != "expanded" || !errors.Is(err, closeFault) || calls != 2 || body.reads != 0 || body.closes != 1 {
			t.Fatalf("expanded ownership: %+v %v native=%+v body=%+v", result, err, native, body)
		}
		var accepted *resource.ResponseError
		if errors.As(err, &accepted) {
			t.Fatal("expanded native status presented as owned success")
		}
	})
}
