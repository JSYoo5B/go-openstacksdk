package objects_test

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/klauspost/compress/zstd"
)

func cloudStreamCollect(stream objects.CloudObjectStream) ([][]byte, error) {
	var chunks [][]byte
	var terminal error
	for chunk, err := range stream.Chunks() {
		if len(chunk) > 0 {
			chunks = append(chunks, chunk)
		}
		if err != nil {
			terminal = err
			break
		}
	}
	return chunks, errors.Join(terminal, stream.Err())
}
func cloudStreamSizes(chunks [][]byte) []int {
	sizes := make([]int, len(chunks))
	for i, chunk := range chunks {
		sizes[i] = len(chunk)
	}
	return sizes
}

func TestPythonCloudObjectStreamLazyDefaultBinaryChunksAndSingleUse(t *testing.T) {
	payload := append(bytes.Repeat([]byte{0, 255}, 1024), 'x')
	body := &objectMetadataBody{Reader: bytes.NewReader(payload)}
	calls, optionCalls := 0, 0
	client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != objectMetadataBase+"box/dir%2Ffile" || r.URL.RawQuery != "" || r.Method != "GET" || r.Body != nil && r.Body != http.NoBody || r.Header.Get("Accept") != "*/*" || r.Header.Get("X-Newest") != "" {
			t.Error(r.URL, r.Header, r.Body)
		}
		return objectMetadataWire(200, body), nil
	})
	stream := objects.New(client).StreamCloudObject(context.Background(), "box", "dir/file", func(cfg *objects.CloudStreamOpts) error { optionCalls++; return nil }, objects.WithCloudStreamQueryString("broken=%\n\xff?#"))
	if calls != 0 || optionCalls != 0 || stream.Receipt() != nil || stream.Err() != nil {
		t.Fatal(calls, optionCalls, stream.Receipt(), stream.Err())
	}
	chunks, err := cloudStreamCollect(stream)
	if err != nil || !bytes.Equal(bytes.Join(chunks, nil), payload) || !reflect.DeepEqual(cloudStreamSizes(chunks), []int{1024, 1024, 1}) || calls != 1 || optionCalls != 1 || body.closes.Load() != 1 {
		t.Fatal(err, cloudStreamSizes(chunks), calls, optionCalls, body.closes.Load())
	}
	chunks[0][0] = 10
	if chunks[1][0] != 0 {
		t.Fatal("chunks alias")
	}
	receipt := stream.Receipt()
	receipt.Header.Set("X-Trans-Id", "forged")
	receipt.StatusCode = 500
	if stream.Receipt().StatusCode != 200 || stream.Receipt().Header.Get("X-Trans-Id") != "actual-object" {
		t.Fatal(stream.Receipt())
	}
	again, e := cloudStreamCollect(stream)
	if len(again) != 0 || e != nil || calls != 1 || stream.Close() != nil || body.closes.Load() != 1 {
		t.Fatal(again, e, calls, body.closes.Load())
	}
}

func TestPythonCloudObjectStreamChunkSizeSnapshotShortReadsAndValidation(t *testing.T) {
	size := 2
	opts := objects.WithCloudStreamOpts(objects.CloudStreamOpts{ChunkSize: &size})
	size = 9
	var retained *int
	reader := strings.NewReader("abcde")
	body := &objectMetadataBody{Reader: objectMetadataReader(func(dst []byte) (int, error) { return reader.Read(dst[:1]) })}
	calls := 0
	api := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return objectMetadataWire(200, body), nil }))
	stream := api.StreamCloudObject(context.Background(), "box", "file", opts, func(cfg *objects.CloudStreamOpts) error { retained = cfg.ChunkSize; return nil })
	var chunks [][]byte
	for chunk, err := range stream.Chunks() {
		if err != nil {
			t.Fatal(err)
		}
		chunks = append(chunks, chunk)
		*retained = 1
	}
	if !reflect.DeepEqual(cloudStreamSizes(chunks), []int{2, 2, 1}) || string(bytes.Join(chunks, nil)) != "abcde" || stream.Err() != nil || calls != 1 || body.closes.Load() != 1 {
		t.Fatal(chunks, stream.Err(), calls, body.closes.Load())
	}
	for _, n := range []int{0, -1, 16777217} {
		s := api.StreamCloudObject(context.Background(), "box", "file", objects.WithCloudStreamChunkSize(n))
		_, e := cloudStreamCollect(s)
		if !errors.Is(e, resource.ErrInvalidOption) {
			t.Fatal(n, e)
		}
	}
	if calls != 1 {
		t.Fatal(calls)
	}
}

func TestPythonCloudObjectStreamSourceSuccessStatusAnd304Body(t *testing.T) {
	for _, status := range []int{100, 199, 200, 203, 204, 206, 304, 399} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			wire := objectMetadataWire(status, &objectMetadataBody{Reader: strings.NewReader("abc")})
			body := wire.Body.(*objectMetadataBody)
			calls := 0
			stream := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return wire, nil })).StreamCloudObject(context.Background(), "box", "file", objects.WithCloudStreamChunkSize(2))
			chunks, e := cloudStreamCollect(stream)
			if e != nil || string(bytes.Join(chunks, nil)) != "abc" || !reflect.DeepEqual(cloudStreamSizes(chunks), []int{2, 1}) || stream.Receipt().StatusCode != status || calls != 1 || body.closes.Load() != 1 {
				t.Fatal(chunks, e, stream.Receipt(), calls, body.closes.Load())
			}
		})
	}
}

func TestPythonCloudObjectStreamClean404OnlyEmptyAndOtherNativeStatuses(t *testing.T) {
	for _, status := range []int{400, 404, 429, 498, 500, 777} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			body := &objectMetadataBody{Reader: strings.NewReader("rejected actual")}
			calls := 0
			stream := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return objectMetadataWire(status, body), nil })).StreamCloudObject(context.Background(), "box", "file")
			chunks, e := cloudStreamCollect(stream)
			if len(chunks) != 0 || calls != 1 || body.closes.Load() != 1 || stream.Receipt().StatusCode != status || stream.Receipt().Header.Get("X-Trans-Id") != "actual-object" {
				t.Fatal(chunks, e, calls, body.closes.Load(), stream.Receipt())
			}
			if status == 404 {
				if e != nil {
					t.Fatal(e)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(e, &native) || native.Actual != status || string(native.Body) != "rejected actual" || native.ResponseHeader.Get("X-Trans-Id") != "actual-object" {
					t.Fatal(e, native)
				}
			}
		})
	}
}

func TestPythonCloudObjectStreamMissingReadCloseFaultsAreNotSuppressed(t *testing.T) {
	for _, mode := range []string{"read", "close", "both"} {
		t.Run(mode, func(t *testing.T) {
			readCause, closeCause := errors.New("missing physical read"), errors.New("missing physical close")
			reader := strings.NewReader("partial404")
			body := &objectMetadataBody{Reader: objectMetadataReader(func(dst []byte) (int, error) {
				n, _ := reader.Read(dst)
				if mode != "close" {
					return n, readCause
				}
				return n, io.EOF
			})}
			if mode != "read" {
				body.closeErr = closeCause
			}
			calls := 0
			stream := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return objectMetadataWire(404, body), nil })).StreamCloudObject(context.Background(), "box", "file")
			_, e := cloudStreamCollect(stream)
			if e == nil || calls != 1 || body.closes.Load() != 1 || stream.Receipt().StatusCode != 404 || mode != "close" && !errors.Is(e, readCause) || mode != "read" && !errors.Is(e, closeCause) {
				t.Fatal(e, calls, body.closes.Load())
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(e, &native) || native.Actual != 404 || string(native.Body) != "partial404" {
				t.Fatal(e, native)
			}
		})
	}
}

func TestPythonCloudObjectStreamPartialReadCloseAndImmutableCurrentChunkProof(t *testing.T) {
	readCause, closeCause := errors.New("partial stream read"), errors.New("partial stream close")
	reads := 0
	body := &objectMetadataBody{Reader: objectMetadataReader(func(dst []byte) (int, error) {
		reads++
		if reads == 1 {
			return copy(dst, "ok"), nil
		}
		return copy(dst, "bad"), errors.Join(io.EOF, readCause)
	}), closeErr: closeCause}
	stream := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return objectMetadataWire(206, body), nil })).StreamCloudObject(context.Background(), "box", "file", objects.WithCloudStreamChunkSize(3))
	chunks, e := cloudStreamCollect(stream)
	if !errors.Is(e, readCause) || !errors.Is(e, closeCause) || string(bytes.Join(chunks, nil)) != "okb" || body.closes.Load() != 1 {
		t.Fatal(chunks, e, body.closes.Load())
	}
	proof := objectMetadataProof(t, e, 206, "b")
	proof.Header.Set("X-Trans-Id", "forged")
	proof.Body[0] = 'x'
	proof.StatusCode = 500
	objectMetadataProof(t, stream.Err(), 206, "b")
	if stream.Receipt().StatusCode != 206 || stream.Receipt().Header.Get("X-Trans-Id") != "actual-object" {
		t.Fatal(stream.Receipt())
	}
}

func TestPythonCloudObjectStreamBreakClosesOnceAndRetainsCleanupFailure(t *testing.T) {
	closeCause := errors.New("early cleanup")
	body := &objectMetadataBody{Reader: strings.NewReader("abcdef"), closeErr: closeCause}
	calls := 0
	api := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return objectMetadataWire(200, body), nil }))
	stream := api.StreamCloudObject(context.Background(), "box", "file", objects.WithCloudStreamChunkSize(2))
	for chunk, e := range stream.Chunks() {
		if e != nil || string(chunk) != "ab" {
			t.Fatal(chunk, e)
		}
		break
	}
	if !errors.Is(stream.Err(), closeCause) || !errors.Is(stream.Close(), closeCause) || body.closes.Load() != 1 || calls != 1 {
		t.Fatal(stream.Err(), calls, body.closes.Load())
	}
	objectMetadataProof(t, stream.Err(), 200, "")
	unopened := api.StreamCloudObject(context.Background(), "box", "file")
	if unopened.Close() != nil {
		t.Fatal(unopened.Err())
	}
	chunks, e := cloudStreamCollect(unopened)
	if len(chunks) != 0 || e != nil || calls != 1 || unopened.Receipt() != nil {
		t.Fatal(chunks, e, calls, unopened.Receipt())
	}
}

func TestPythonCloudObjectStreamStickyRestoredSourceAndCancellationAfterActualRead(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("stream cancelled cause")
	var client *gophercloud.ServiceClient
	wire := objectMetadataWire(203, &objectMetadataBody{})
	body := wire.Body.(*objectMetadataBody)
	body.Reader = objectMetadataReader(func(dst []byte) (int, error) {
		client.Endpoint = objectMetadataBase + "changed/"
		wire.StatusCode = 500
		wire.Header.Set("X-Trans-Id", "forged")
		return copy(dst, "x"), io.EOF
	})
	body.onClose = func() { client.Endpoint = objectMetadataBase; cancel(cause) }
	calls := 0
	client = objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return wire, nil })
	stream := objects.New(client).StreamCloudObject(ctx, "box", "file")
	chunks, e := cloudStreamCollect(stream)
	if string(bytes.Join(chunks, nil)) != "x" || !errors.Is(e, resource.ErrInvalidOption) || !errors.Is(e, cause) || !errors.Is(e, context.Canceled) || calls != 1 || body.closes.Load() != 1 {
		t.Fatal(chunks, e, calls, body.closes.Load())
	}
	objectMetadataProof(t, e, 203, "x")
	if stream.Receipt().StatusCode != 203 || stream.Receipt().Header.Get("X-Trans-Id") != "actual-object" {
		t.Fatal(stream.Receipt())
	}
}

func TestPythonCloudObjectStreamNative401PhysicalFaultDoesNotInvokeHooks(t *testing.T) {
	readCause, closeCause, unexpected := errors.New("401 physical read"), errors.New("401 physical close"), errors.New("unexpected reauthentication")
	calls, retries, reauths, backoffs, reads := 0, 0, 0, 0, 0
	body := &objectMetadataBody{Reader: objectMetadataReader(func(dst []byte) (int, error) {
		reads++
		if reads > 1 {
			return 0, io.EOF
		}
		return copy(dst, "denied"), readCause
	}), closeErr: closeCause}
	client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
		calls++
		if calls > 1 {
			return objectMetadataResponse(500, "bounded unexpected replay"), nil
		}
		return objectMetadataWire(401, body), nil
	})
	client.ProviderClient.ReauthFunc = func(context.Context) error { reauths++; return unexpected }
	client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		retries++
		return unexpected
	}
	client.ProviderClient.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
		backoffs++
		return unexpected
	}
	stream := objects.New(client).StreamCloudObject(context.Background(), "box", "file")
	_, e := cloudStreamCollect(stream)
	var native *gophercloud.ErrUnableToReauthenticate
	if !errors.As(e, &native) || native.ErrOriginal == nil || native.ErrReauth == nil || !errors.Is(native.ErrReauth, readCause) || !errors.Is(native.ErrReauth, closeCause) || calls != 1 || retries != 0 || reauths != 0 || backoffs != 0 || body.closes.Load() != 1 {
		t.Fatal(e, native, calls, retries, reauths, backoffs, body.closes.Load())
	}
	var receipt gophercloud.ErrUnexpectedResponseCode
	if !errors.As(native.ErrOriginal, &receipt) || receipt.Actual != 401 || string(receipt.Body) != "denied" || receipt.ResponseHeader.Get("X-Trans-Id") != "actual-object" || stream.Receipt().StatusCode != 401 {
		t.Fatal(e, receipt, stream.Receipt())
	}
}

func TestPythonCloudObjectStreamCleanNativeRenewalAndBackoffReuseFixedRoute(t *testing.T) {
	for _, status := range []int{401, 429, 498} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			first, last := &objectMetadataBody{Reader: strings.NewReader("rejected")}, &objectMetadataBody{Reader: strings.NewReader("ok")}
			calls, renewals, backoffs := 0, 0, 0
			client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.String() != objectMetadataBase+"box/file" || r.Header.Get("Accept") != "*/*" || r.Header.Get("X-Newest") != "" {
					t.Error(r)
				}
				if calls == 1 {
					return objectMetadataWire(status, first), nil
				}
				return objectMetadataWire(200, last), nil
			})
			client.ProviderClient.ReauthFunc = func(context.Context) error { renewals++; client.ProviderClient.SetToken("renewed"); return nil }
			client.ProviderClient.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				backoffs++
				return nil
			}
			stream := objects.New(client).StreamCloudObject(context.Background(), "box", "file")
			chunks, e := cloudStreamCollect(stream)
			if e != nil || string(bytes.Join(chunks, nil)) != "ok" || calls != 2 || first.closes.Load() != 1 || last.closes.Load() != 1 || status == 401 && renewals != 1 || status != 401 && backoffs != 1 {
				t.Fatal(chunks, e, calls, renewals, backoffs, first.closes.Load(), last.closes.Load())
			}
		})
	}
}

func cloudStreamEncode(t *testing.T, encoding string, input []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	switch encoding {
	case "gzip", "x-gzip":
		w := gzip.NewWriter(&out)
		_, _ = w.Write(input)
		if e := w.Close(); e != nil {
			t.Fatal(e)
		}
	case "deflate":
		w := zlib.NewWriter(&out)
		_, _ = w.Write(input)
		if e := w.Close(); e != nil {
			t.Fatal(e)
		}
	case "raw-deflate":
		w, e := flate.NewWriter(&out, flate.DefaultCompression)
		if e != nil {
			t.Fatal(e)
		}
		_, _ = w.Write(input)
		if e := w.Close(); e != nil {
			t.Fatal(e)
		}
	case "zstd":
		w, e := zstd.NewWriter(&out, zstd.WithEncoderConcurrency(1))
		if e != nil {
			t.Fatal(e)
		}
		_, _ = w.Write(input)
		if e := w.Close(); e != nil {
			t.Fatal(e)
		}
	default:
		return bytes.Clone(input)
	}
	return out.Bytes()
}

func TestPythonCloudObjectStreamRequestsCodecsAndReverseEncodingChain(t *testing.T) {
	input := []byte{0, 255, 'a', 'b', 'c', 'd', 'e'}
	for _, format := range []string{"gzip", "x-gzip", "deflate", "raw-deflate", "zstd", "br", "unknown", "gzip, zstd", "unknown, gzip"} {
		t.Run(format, func(t *testing.T) {
			encoding := format
			payload := cloudStreamEncode(t, format, input)
			switch format {
			case "raw-deflate":
				encoding = "deflate"
			case "gzip, zstd":
				payload = cloudStreamEncode(t, "zstd", cloudStreamEncode(t, "gzip", input))
			case "unknown, gzip":
				payload = cloudStreamEncode(t, "gzip", cloudStreamEncode(t, "raw-deflate", input))
			}
			body := &objectMetadataBody{Reader: bytes.NewReader(payload)}
			wire := objectMetadataWire(200, body)
			wire.Header.Set("Content-Encoding", encoding)
			stream := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return wire, nil })).StreamCloudObject(context.Background(), "box", "file", objects.WithCloudStreamChunkSize(2))
			chunks, e := cloudStreamCollect(stream)
			if e != nil || !bytes.Equal(bytes.Join(chunks, nil), input) || !reflect.DeepEqual(cloudStreamSizes(chunks), []int{2, 2, 2, 1}) || body.closes.Load() != 1 || stream.Receipt().Header.Get("Content-Encoding") != encoding {
				t.Fatal(chunks, e, body.closes.Load(), stream.Receipt())
			}
		})
	}
}

func TestPythonCloudObjectStreamEmptyBodiesAndMalformedCodecKeepReceipt(t *testing.T) {
	for _, encoding := range []string{"", "gzip", "deflate"} {
		t.Run("empty/"+encoding, func(t *testing.T) {
			body := &objectMetadataBody{Reader: bytes.NewReader(nil)}
			wire := objectMetadataWire(200, body)
			wire.Header.Set("Content-Encoding", encoding)
			stream := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return wire, nil })).StreamCloudObject(context.Background(), "box", "file")
			chunks, e := cloudStreamCollect(stream)
			if len(chunks) != 0 || e != nil || body.closes.Load() != 1 {
				t.Fatal(chunks, e, body.closes.Load())
			}
		})
	}
	for _, encoding := range []string{"gzip", "zstd", "gzip,gzip,gzip,gzip,gzip,gzip"} {
		t.Run("malformed/"+encoding, func(t *testing.T) {
			body := &objectMetadataBody{Reader: strings.NewReader("broken")}
			wire := objectMetadataWire(203, body)
			wire.Header.Set("Content-Encoding", encoding)
			stream := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return wire, nil })).StreamCloudObject(context.Background(), "box", "file")
			_, e := cloudStreamCollect(stream)
			if e == nil || body.closes.Load() != 1 || stream.Receipt().StatusCode != 203 {
				t.Fatal(e, body.closes.Load(), stream.Receipt())
			}
			objectMetadataProof(t, e, 203, "")
		})
	}
}

func TestPythonCloudObjectStreamNilCancelledAndOptionSourceMutationBeforeHTTP(t *testing.T) {
	calls, options := 0, 0
	client := objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return objectMetadataResponse(200, "ok"), nil })
	api := objects.New(client)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, stream := range []objects.CloudObjectStream{(*objects.API)(nil).StreamCloudObject(context.Background(), "box", "file"), api.StreamCloudObject(nil, "box", "file"), api.StreamCloudObject(ctx, "box", "file"), api.StreamCloudObject(context.Background(), "", "file"), api.StreamCloudObject(context.Background(), "box", ""), api.StreamCloudObject(context.Background(), "box", "file", nil)} {
		_, e := cloudStreamCollect(stream)
		if e == nil {
			t.Fatal(e)
		}
	}
	stream := api.StreamCloudObject(context.Background(), "box", "file", func(*objects.CloudStreamOpts) error {
		options++
		client.Endpoint = objectMetadataBase + "changed/"
		return nil
	}, func(*objects.CloudStreamOpts) error { options++; client.Endpoint = objectMetadataBase; return nil })
	_, e := cloudStreamCollect(stream)
	if !errors.Is(e, resource.ErrInvalidOption) || calls != 0 || options != 1 {
		t.Fatal(e, calls, options)
	}
}

func TestPythonCloudObjectStreamCloseFromOptionsAndConsumerStopsBeforeAnotherRead(t *testing.T) {
	calls, reads := 0, 0
	body := &objectMetadataBody{Reader: objectMetadataReader(func(dst []byte) (int, error) { reads++; return copy(dst, "abcdef"), nil })}
	api := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { calls++; return objectMetadataWire(200, body), nil }))
	var unopened objects.CloudObjectStream
	unopened = api.StreamCloudObject(context.Background(), "box", "file", func(*objects.CloudStreamOpts) error { return unopened.Close() })
	chunks, e := cloudStreamCollect(unopened)
	if len(chunks) != 0 || e != nil || calls != 0 || unopened.Receipt() != nil {
		t.Fatal(chunks, e, calls, unopened.Receipt())
	}
	stream := api.StreamCloudObject(context.Background(), "box", "file", objects.WithCloudStreamChunkSize(2))
	yields := 0
	for chunk, e := range stream.Chunks() {
		yields++
		if e != nil || string(chunk) != "ab" {
			t.Fatal(chunk, e)
		}
		if e := stream.Close(); e != nil {
			t.Fatal(e)
		}
	}
	if yields != 1 || calls != 1 || reads != 1 || body.closes.Load() != 1 || stream.Err() != nil {
		t.Fatal(yields, calls, reads, body.closes.Load(), stream.Err())
	}
}

func TestPythonCloudObjectStreamConcurrentCloseDuringGETOwnsLateBody(t *testing.T) {
	for _, fault := range []bool{false, true} {
		t.Run(fmt.Sprint(fault), func(t *testing.T) {
			entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
			closeCause := errors.New("late accepted close")
			body := &objectMetadataBody{Reader: strings.NewReader("late unread")}
			if fault {
				body.closeErr = closeCause
			}
			reads := 0
			body.Reader = objectMetadataReader(func(dst []byte) (int, error) { reads++; return copy(dst, "unexpected read"), io.EOF })
			stream := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) {
				close(entered)
				<-release
				return objectMetadataWire(200, body), nil
			})).StreamCloudObject(context.Background(), "box", "file")
			go func() {
				chunks, e := cloudStreamCollect(stream)
				if len(chunks) != 0 {
					done <- fmt.Errorf("unexpected chunks %v", chunks)
					return
				}
				done <- e
			}()
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("GET did not start")
			}
			if e := stream.Close(); e != nil {
				t.Fatal(e)
			}
			close(release)
			var e error
			select {
			case e = <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("late GET did not close")
			}
			if reads != 0 || body.closes.Load() != 1 || stream.Receipt().StatusCode != 200 || fault && !errors.Is(e, closeCause) || !fault && e != nil {
				t.Fatal(e, reads, body.closes.Load(), stream.Receipt())
			}
			if fault {
				objectMetadataProof(t, stream.Err(), 200, "")
			}
		})
	}
}

func TestPythonCloudObjectStreamNativePolicyErrorCopiesUnaryActualReceipt(t *testing.T) {
	calls := 0
	client := objectMetadataClient(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return objectMetadataResponse(500, "retry"), nil
		}
		return objectMetadataResponse(404, "actual404"), nil
	})
	client.ProviderClient.RetryFunc = func(ctx context.Context, method, url string, opts *gophercloud.RequestOpts, e error, count uint) error {
		if count > 1 {
			return errors.New("bounded unexpected retry")
		}
		opts.OkCodes = []int{404}
		return nil
	}
	stream := objects.New(client).StreamCloudObject(context.Background(), "box", "file")
	_, e := cloudStreamCollect(stream)
	var native gophercloud.ErrUnexpectedResponseCode
	var marker interface{ TerminalSDKFailure() bool }
	if e == nil || !errors.As(e, &native) || native.Actual != 404 || string(native.Body) != "actual404" || !errors.As(e, &marker) || !marker.TerminalSDKFailure() || calls != 2 {
		t.Fatal(e, native, calls)
	}
	native.Body[0] = 'x'
	native.ResponseHeader.Set("X-Trans-Id", "forged")
	if !errors.As(stream.Err(), &native) || string(native.Body) != "actual404" || native.ResponseHeader.Get("X-Trans-Id") != "actual-object" {
		t.Fatal(stream.Err(), native)
	}
}

func TestPythonCloudObjectStreamCompressedRejectionsValidateBeforeIgnoring404(t *testing.T) {
	for _, encoding := range []string{"gzip", "deflate", "zstd"} {
		for _, valid := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%v", encoding, valid), func(t *testing.T) {
				payload := []byte("broken")
				if encoding == "deflate" {
					payload = []byte{0xff, 0xff, 0xff}
				}
				if valid {
					payload = cloudStreamEncode(t, encoding, []byte("not found"))
				}
				body := &objectMetadataBody{Reader: bytes.NewReader(payload)}
				wire := objectMetadataWire(404, body)
				wire.Header.Set("Content-Encoding", encoding)
				stream := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return wire, nil })).StreamCloudObject(context.Background(), "box", "file")
				chunks, e := cloudStreamCollect(stream)
				if len(chunks) != 0 || body.closes.Load() != 1 || stream.Receipt().StatusCode != 404 {
					t.Fatal(chunks, e, body.closes.Load(), stream.Receipt())
				}
				if valid {
					if e != nil {
						t.Fatal(e)
					}
				} else {
					if e == nil {
						t.Fatal("decoder fault hidden by 404")
					}
					objectMetadataProof(t, e, 404, string(payload))
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(e, &native) || !bytes.Equal(native.Body, payload) {
						t.Fatal(e, native)
					}
				}
			})
		}
	}
}

func TestPythonCloudObjectStreamDeflateFallsBackAfterValidZlibPrefix(t *testing.T) {
	payload := append([]byte{0x78, 0x9c, 0x00, 0x63, 0xff}, bytes.Repeat([]byte{'A'}, 156)...)
	payload = append(payload, 1, 0, 0, 255, 255)
	for _, size := range []int{2, 16, 1024} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			body := &objectMetadataBody{Reader: bytes.NewReader(payload)}
			wire := objectMetadataWire(200, body)
			wire.Header.Set("Content-Encoding", "deflate")
			stream := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return wire, nil })).StreamCloudObject(context.Background(), "box", "file", objects.WithCloudStreamChunkSize(size))
			chunks, e := cloudStreamCollect(stream)
			if e != nil || !bytes.Equal(bytes.Join(chunks, nil), bytes.Repeat([]byte{'A'}, 156)) || body.closes.Load() != 1 {
				t.Fatal(chunks, e, body.closes.Load())
			}
		})
	}
}

func TestPythonCloudObjectStreamEncodingOverrideIsOwnedThroughCallbacksAndNativeRetry(t *testing.T) {
	calls := 0
	client := objectMetadataClient(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Accept-Encoding") != "identity" {
			t.Error(r.Header)
		}
		return objectMetadataResponse(200, "ok"), nil
	})
	client.MoreHeaders = map[string]string{"Accept-Encoding": "identity"}
	stream := objects.New(client).StreamCloudObject(context.Background(), "box", "file", func(*objects.CloudStreamOpts) error { client.MoreHeaders["Accept-Encoding"] = "gzip"; return nil })
	chunks, e := cloudStreamCollect(stream)
	if e != nil || string(bytes.Join(chunks, nil)) != "ok" || calls != 1 {
		t.Fatal(chunks, e, calls)
	}
	calls = 0
	client = objectMetadataClient(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Accept-Encoding") != "gzip, deflate, zstd" {
			t.Error(r.Header)
		}
		return objectMetadataResponse(500, "actual retry fault"), nil
	})
	client.ProviderClient.RetryFunc = func(ctx context.Context, method, url string, opts *gophercloud.RequestOpts, e error, count uint) error {
		opts.MoreHeaders["Accept-Encoding"] = "identity"
		return nil
	}
	stream = objects.New(client).StreamCloudObject(context.Background(), "box", "file")
	_, e = cloudStreamCollect(stream)
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.Is(e, resource.ErrInvalidOption) || !errors.As(e, &native) || native.Actual != 500 || string(native.Body) != "actual retry fault" || calls != 1 {
		t.Fatal(e, native, calls)
	}
}

func TestPythonCloudObjectStreamGzipCleanEOFTrailerTailHeaderAndCompressed404(t *testing.T) {
	input := []byte{0, 255, 'a', 'b', 'c', 'd', 'e', 'f'}
	encoded := cloudStreamEncode(t, "gzip", input)
	cases := []struct {
		name          string
		payload, want []byte
	}{{"missingCRC1", encoded[:len(encoded)-1], input}, {"missingCRC4", encoded[:len(encoded)-4], input}, {"missingCRC8", encoded[:len(encoded)-8], input}, {"shortHeader1", encoded[:1], nil}, {"shortHeader2", encoded[:2], nil}, {"trailingShortJunk", append(bytes.Clone(encoded), []byte("junk")...), input}, {"trailingLongJunk", append(bytes.Clone(encoded), bytes.Repeat([]byte("junk"), 4)...), input}, {"twoMembers", append(bytes.Clone(encoded), encoded...), append(bytes.Clone(input), input...)}}
	for _, tc := range cases {
		for _, status := range []int{200, 404} {
			t.Run(fmt.Sprintf("%s/%d", tc.name, status), func(t *testing.T) {
				body := &objectMetadataBody{Reader: bytes.NewReader(tc.payload)}
				wire := objectMetadataWire(status, body)
				wire.Header.Set("Content-Encoding", "gzip")
				stream := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return wire, nil })).StreamCloudObject(context.Background(), "box", "file", objects.WithCloudStreamChunkSize(2))
				chunks, e := cloudStreamCollect(stream)
				want := tc.want
				if status == 404 {
					want = nil
				}
				if e != nil || !bytes.Equal(bytes.Join(chunks, nil), want) || body.closes.Load() != 1 || stream.Receipt().StatusCode != status {
					t.Fatal(chunks, e, body.closes.Load(), stream.Receipt())
				}
			})
		}
	}
	badCRC := bytes.Clone(encoded)
	badCRC[len(badCRC)-8] ^= 1
	for _, status := range []int{200, 404} {
		t.Run(fmt.Sprintf("badCRC/%d", status), func(t *testing.T) {
			body := &objectMetadataBody{Reader: bytes.NewReader(badCRC)}
			wire := objectMetadataWire(status, body)
			wire.Header.Set("Content-Encoding", "gzip")
			stream := objects.New(objectMetadataClient(func(*http.Request) (*http.Response, error) { return wire, nil })).StreamCloudObject(context.Background(), "box", "file")
			_, e := cloudStreamCollect(stream)
			if !errors.Is(e, gzip.ErrChecksum) || body.closes.Load() != 1 || stream.Receipt().StatusCode != status {
				t.Fatal(e, body.closes.Load(), stream.Receipt())
			}
		})
	}
}

func TestPythonCloudObjectStreamGzipEOFNormalizationNeverHidesPhysicalFaults(t *testing.T) {
	encoded := cloudStreamEncode(t, "gzip", []byte("partial compressed"))
	encoded = encoded[:len(encoded)-8]
	for _, mode := range []string{"read", "close", "cancel and restored source"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("gzip physical read"), errors.New("gzip physical close"), errors.New("gzip custom cancellation")
			var client *gophercloud.ServiceClient
			reader := bytes.NewReader(encoded)
			body := &objectMetadataBody{Reader: objectMetadataReader(func(dst []byte) (int, error) {
				n, _ := reader.Read(dst)
				if mode == "read" {
					return n, readCause
				}
				if mode == "cancel and restored source" {
					client.Endpoint = objectMetadataBase + "changed/"
				}
				return n, io.EOF
			})}
			if mode == "close" {
				body.closeErr = closeCause
			}
			if mode == "cancel and restored source" {
				body.onClose = func() { client.Endpoint = objectMetadataBase; cancel(cancelCause) }
			}
			wire := objectMetadataWire(203, body)
			wire.Header.Set("Content-Encoding", "gzip")
			client = objectMetadataClient(func(*http.Request) (*http.Response, error) { return wire, nil })
			stream := objects.New(client).StreamCloudObject(ctx, "box", "file")
			_, e := cloudStreamCollect(stream)
			if e == nil || body.closes.Load() != 1 || mode == "read" && !errors.Is(e, readCause) || mode == "close" && !errors.Is(e, closeCause) || mode == "cancel and restored source" && (!errors.Is(e, resource.ErrInvalidOption) || !errors.Is(e, context.Canceled) || !errors.Is(e, cancelCause)) {
				t.Fatal(e, body.closes.Load())
			}
			objectMetadataProof(t, e, 203, string(encoded))
		})
	}
}
