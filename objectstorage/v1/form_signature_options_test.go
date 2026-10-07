package v1

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestFormSignatureOptionsSnapshotsAndReplacement(t *testing.T) {
	t.Run("key pointers maps and retained callbacks are independent", func(t *testing.T) {
		c := tempURLKeyTestClient()
		key := []byte("\xffsecret")
		stamp := signingTestTimestamp()
		newest := true
		headers := map[string]string{"x-trace": "factory"}
		full := WithGenerateFormSignatureOpts(GenerateFormSignatureOpts{Key: key, Digest: TempURLDigestSHA256, Timestamp: &stamp, Headers: headers, Newest: &newest})
		key[0] = '!'
		stamp = stamp.AddDate(1, 0, 0)
		newest = false
		headers["x-trace"] = "outside"
		extra := map[string]string{"X-Trace": "owned"}
		plural := WithGenerateFormSignatureHeaders(extra)
		extra["X-Trace"] = "outside"
		var retained *GenerateFormSignatureOpts
		callbacks := 0
		inspect := func(cfg *GenerateFormSignatureOpts) error {
			callbacks++
			if string(cfg.Key) != "\xffsecret" || cfg.Timestamp.Unix() != 1700000000 || cfg.Newest == nil || !*cfg.Newest || cfg.Headers["x-trace"] != "factory" {
				t.Fatal("factory snapshot lost", cfg)
			}
			retained = cfg
			return nil
		}
		mutate := func(cfg *GenerateFormSignatureOpts) error {
			retained.Key[0] = '!'
			*retained.Timestamp = stamp
			*retained.Newest = false
			retained.Headers["x-trace"] = "retained"
			if string(cfg.Key) != "\xffsecret" || cfg.Timestamp.Unix() != 1700000000 || !*cfg.Newest {
				t.Fatal("retained callback aliases", cfg)
			}
			return nil
		}
		for i := 0; i < 2; i++ {
			result, err := New(c).GenerateFormSignature(context.Background(), "box", signingTestInput(), full, inspect, mutate, plural, WithGenerateFormSignatureHeader("X-Trace", "last"), WithGenerateFormSignatureNewest(false))
			if err != nil || result == nil || result.Signature != "74c3584806cdd2765abf0480a6ccec328683324aff8edc7bb8519f01386bdc5f" || result.Discovery != nil {
				t.Fatal(result, err)
			}
		}
		if callbacks != 2 {
			t.Fatal("option invoked more than once", callbacks)
		}
	})
	t.Run("full replacement clear newest and owned read preferences", func(t *testing.T) {
		c := tempURLKeyTestClient()
		calls := 0
		c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("X-Removed") != "" || r.Header.Get("X-Trace") != "owned" || r.Header.Get("X-Newest") != "false" || len(r.Header.Values("X-Newest")) != 1 {
				t.Fatal("replacement or canonical overwrite lost", r.Header)
			}
			return tempURLKeyTestWire(r, 204, http.Header{"X-Container-Meta-Temp-URL-Key-2": {"container-secondary"}}, io.NopCloser(strings.NewReader("raw"))), nil
		})
		full := WithGenerateFormSignatureOpts(GenerateFormSignatureOpts{Headers: map[string]string{"x-trace": "old"}})
		result, err := New(c).GenerateFormSignature(context.Background(), "box", signingTestInput(), WithGenerateFormSignatureKey([]byte("discard")), WithGenerateFormSignatureHeaders(map[string]string{"X-Removed": "old"}), WithGenerateFormSignatureNewest(true), full, WithGenerateFormSignatureHeader("X-Trace", "owned"), WithoutGenerateFormSignatureNewest(), WithGenerateFormSignatureNewest(false), WithGenerateFormSignatureTimestamp(signingTestTimestamp()), WithGenerateFormSignatureDigest(""))
		if err != nil || result == nil || result.Signature != "088f8af8829bd0f68cdfeb13eabb3a2a8609039b" || result.Digest != TempURLDigestSHA1 || calls != 1 {
			t.Fatal(result, err, calls)
		}
	})
}

func TestFormSignatureOptionsParallelReuseAndInvalidValues(t *testing.T) {
	c := tempURLKeyTestClient()
	headers := map[string]string{"X-Trace": "stable"}
	key := []byte("factory-key")
	options := []GenerateFormSignatureOption{WithGenerateFormSignatureKey(key), WithGenerateFormSignatureTimestamp(signingTestTimestamp()), WithGenerateFormSignatureHeaders(headers)}
	key[0] = '!'
	headers["X-Trace"] = "outside"
	var workers sync.WaitGroup
	for i := 0; i < 8; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			result, err := New(c).GenerateFormSignature(context.Background(), "box", signingTestInput(), options...)
			if err != nil || result == nil || result.Signature != "05c309efdf76da6c30b716b182112a7b8080172c" || result.Discovery != nil {
				t.Errorf("parallel reuse failed %#v %v", result, err)
				return
			}
		}()
	}
	workers.Wait()
	for _, options := range [][]GenerateFormSignatureOption{
		{nil}, {WithGenerateFormSignatureKey([]byte{})}, {WithGenerateFormSignatureOpts(GenerateFormSignatureOpts{Key: make([]byte, 0)})},
		{WithGenerateFormSignatureHeaders(map[string]string{"x-trace": "one", "X-Trace": "two"})},
		{WithGenerateFormSignatureHeader("X-Account-Meta-Key", "owned")}, {WithGenerateFormSignatureHeader("X-Container-Meta-Key", "owned")},
		{WithGenerateFormSignatureHeader("X-Auth-Token", "owned")}, {WithGenerateFormSignatureHeader("X-Newest", "owned")},
		{WithGenerateFormSignatureHeader("X-K", "invalid")}, {WithGenerateFormSignatureHeader("X-Trace", "bad\r\nvalue")},
	} {
		client := tempURLKeyTestClient()
		calls := 0
		client.HTTPClient.Transport = tempURLKeyTestTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("HTTP") })
		result, err := New(client).GenerateFormSignature(context.Background(), "box", signingTestInput(), options...)
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatal(result, err, calls)
		}
	}
}
