package objects

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func createUploadPrepared(t *testing.T, c *gophercloud.ServiceClient) *preparedCreateObject {
	t.Helper()
	p, err := New(c).captureCreateObject(context.Background(), "box", "key")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestObjectCreateUploadNativePhysicalReplay(t *testing.T) {
	for _, mode := range []string{"retry", "backoff", "reauth", "redirect"} {
		t.Run(mode, func(t *testing.T) {
			c := objectMetadataClient()
			c.MoreHeaders = map[string]string{"X-Source": "captured"}
			calls, hooks := 0, 0
			var prior io.ReadCloser
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "PUT" || r.ContentLength != 6 || len(r.TransferEncoding) != 0 || r.GetBody == nil {
					t.Error("owned physical framing", r.Method, r.ContentLength, r.TransferEncoding, r.GetBody == nil)
				}
				prefix := make([]byte, 2)
				_, err := io.ReadFull(r.Body, prefix)
				if prior != nil {
					if closeErr := prior.Close(); closeErr != nil {
						t.Error(closeErr)
					}
				}
				suffix, readErr := io.ReadAll(r.Body)
				if err != nil || readErr != nil || string(append(prefix, suffix...)) != "frozen" {
					t.Error("physical replay shared a cursor", string(prefix), string(suffix), err, readErr)
				}
				prior = r.Body
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
					t.Error("live auth not refreshed", r.Header)
				}
				return objectMetadataWire(r, code, headers, io.NopCloser(strings.NewReader(strconv.Itoa(code)))), nil
			})
			switch mode {
			case "retry":
				c.RetryFunc = func(ctx context.Context, method, target string, opts *gophercloud.RequestOpts, original error, count uint) error {
					hooks++
					if count != 1 || !gophercloud.ResponseCodeIs(original, 503) {
						t.Error("native retry changed", original, count)
					}
					c.MoreHeaders["X-Source"] = "valid later drift"
					return nil
				}
			case "backoff":
				c.RetryBackoffFunc = func(ctx context.Context, response *gophercloud.ErrUnexpectedResponseCode, original error, count uint) error {
					hooks++
					if response.Actual != 429 || count != 1 {
						t.Error(response, count)
					}
					return nil
				}
			case "reauth":
				c.ReauthFunc = func(context.Context) error { hooks++; c.SetToken("fresh"); return nil }
			case "redirect":
				c.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error { hooks++; return nil }
			}
			p := createUploadPrepared(t, c)
			source := &objectCreateSource{data: []byte("frozen"), size: 6}
			out := p.exchange(context.Background(), "PUT", p.metadata.target, "ordinary", &objectCreatePayload{source: source, size: 6}, nil, 1, 201, 202)
			if out.err != nil || calls != 2 || hooks != 1 || out.phase.Acknowledgement == nil || out.phase.Acknowledgement.StatusCode != 201 || len(out.phase.Attempts) != 2 || out.phase.Attempts[0].Response.StatusCode == 201 || string(out.phase.Attempts[1].Response.Body) != "201" {
				t.Fatal(out, calls, hooks)
			}
			if mode != "redirect" && out.phase.Attempts[0].Error == nil {
				t.Fatal("recovered native error lost")
			}
			out.phase.Acknowledgement.Body[0] = '!'
			out.phase.Acknowledgement.Header.Set("X-Proof", "changed")
			if string(out.phase.Attempts[1].Response.Body) != "201" || out.phase.Attempts[1].Response.Header.Get("X-Proof") != "kept" {
				t.Fatal("physical and acknowledgement evidence alias")
			}
		})
	}
}

func TestObjectCreateUploadNativeOwnershipGuards(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*gophercloud.RequestOpts)
	}{
		{"RawBody", func(o *gophercloud.RequestOpts) { o.RawBody = strings.NewReader("foreign") }},
		{"carrier content", func(o *gophercloud.RequestOpts) { o.RawBody.(*bytes.Reader).Reset([]byte("foreign")) }},
		{"carrier position", func(o *gophercloud.RequestOpts) { _, _ = o.RawBody.(*bytes.Reader).Seek(1, io.SeekStart) }},
		{"JSONBody", func(o *gophercloud.RequestOpts) { o.JSONBody = map[string]any{"foreign": true} }},
		{"JSONResponse", func(o *gophercloud.RequestOpts) { o.JSONResponse = &map[string]any{} }},
		{"KeepBody", func(o *gophercloud.RequestOpts) { o.KeepResponseBody = false }},
		{"expanded codes", func(o *gophercloud.RequestOpts) { o.OkCodes = append(o.OkCodes, 503) }},
		{"narrowed codes", func(o *gophercloud.RequestOpts) { o.OkCodes = []int{202} }},
		{"cookie", func(o *gophercloud.RequestOpts) { o.MoreHeaders["cookie"] = "auth" }},
		{"service token", func(o *gophercloud.RequestOpts) { o.MoreHeaders["X-Service-Token"] = "auth" }},
		{"length", func(o *gophercloud.RequestOpts) { o.MoreHeaders["content-length"] = "999" }},
		{"ordinary ETag", func(o *gophercloud.RequestOpts) { o.MoreHeaders["ETag"] = "foreign" }},
		{"copy routing", func(o *gophercloud.RequestOpts) { o.MoreHeaders["X-Copy-From"] = "/other/key" }},
		{"metadata replacement", func(o *gophercloud.RequestOpts) { o.MoreHeaders["X-Object-Meta-Tag"] = "foreign" }},
		{"metadata removal", func(o *gophercloud.RequestOpts) { delete(o.MoreHeaders, "X-Object-Meta-Tag") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := objectMetadataClient()
			calls, hooks := 0, 0
			cause := errors.New("native callback cause")
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				return objectMetadataWire(r, 503, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("rejected"))), nil
			})
			c.RetryFunc = func(ctx context.Context, method, target string, o *gophercloud.RequestOpts, original error, count uint) error {
				hooks++
				tc.mutate(o)
				return cause
			}
			p := createUploadPrepared(t, c)
			source := &objectCreateSource{data: []byte("owned"), size: 5}
			out := p.exchange(context.Background(), "PUT", p.metadata.target, "ordinary", &objectCreatePayload{source: source, size: 5}, map[string]string{"X-Object-Meta-Tag": "frozen"}, 1, 201, 202)
			if !errors.Is(out.err, resource.ErrInvalidOption) || !errors.Is(out.err, cause) || !gophercloud.ResponseCodeIs(out.err, 503) || out.retry || calls != 1 || hooks != 1 || out.phase.Acknowledgement != nil || len(out.phase.Attempts) != 1 || string(out.phase.Attempts[0].Response.Body) != "rejected" {
				t.Fatal("owned mutation replayed", out, calls, hooks)
			}
		})
	}
	for _, tc := range []struct {
		name   string
		mutate func(*http.Request)
	}{
		{"body", func(r *http.Request) { r.Body = io.NopCloser(strings.NewReader("foreign")) }},
		{"length", func(r *http.Request) { r.ContentLength = 999 }},
		{"transfer", func(r *http.Request) { r.TransferEncoding = []string{"chunked"} }},
		{"trailer", func(r *http.Request) { r.Trailer = http.Header{"X-Foreign": {"one"}} }},
		{"GetBody", func(r *http.Request) {
			r.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("foreign")), nil }
		}},
	} {
		t.Run("redirect "+tc.name, func(t *testing.T) {
			c := objectMetadataClient()
			calls := 0
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				return objectMetadataWire(r, 307, http.Header{"Location": {r.URL.String()}}, io.NopCloser(strings.NewReader("redirect"))), nil
			})
			c.HTTPClient.CheckRedirect = func(r *http.Request, via []*http.Request) error { tc.mutate(r); return nil }
			p := createUploadPrepared(t, c)
			source := &objectCreateSource{data: []byte("owned"), size: 5}
			out := p.exchange(context.Background(), "PUT", p.metadata.target, "ordinary", &objectCreatePayload{source: source, size: 5}, nil, 1, 201, 202)
			if !errors.Is(out.err, resource.ErrInvalidOption) || out.retry || calls != 1 || out.phase.Acknowledgement != nil {
				t.Fatal(out, calls)
			}
		})
	}
}

func TestObjectCreateUploadPhysicalFaultEvidence(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		status                 int
		read, close, transport bool
	}{
		{"accepted read", 201, true, false, false}, {"accepted close", 202, false, true, false},
		{"rejected read and close", 503, true, true, false}, {"discarded wire with transport error", 201, false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := objectMetadataClient()
			readCause, closeCause, transportCause := errors.New("read cause"), errors.New("close cause"), errors.New("transport cause")
			calls, hooks := 0, 0
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				body := &readTestBody{data: []byte("raw")}
				if tc.read {
					body.readErr = readCause
				}
				if tc.close {
					body.closeErr = closeCause
				}
				wire := objectMetadataWire(r, tc.status, http.Header{"X-Proof": {"kept"}}, body)
				if tc.transport {
					return wire, transportCause
				}
				return wire, nil
			})
			if !tc.transport {
				c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks++
					return nil
				}
			}
			p := createUploadPrepared(t, c)
			source := &objectCreateSource{data: []byte("owned"), size: 5}
			out := p.exchange(context.Background(), "PUT", p.metadata.target, "ordinary", &objectCreatePayload{source: source, size: 5}, nil, 1, 201, 202)
			if out.err == nil || calls != 1 || hooks != 0 || len(out.phase.Attempts) != 1 || out.phase.Attempts[0].Response == nil || out.phase.Attempts[0].Response.StatusCode != tc.status || out.phase.Attempts[0].Error == nil {
				t.Fatal(out, calls, hooks)
			}
			if tc.read && !errors.Is(out.err, readCause) || tc.close && !errors.Is(out.err, closeCause) || tc.transport && !errors.Is(out.err, transportCause) {
				t.Fatal("fault cause lost", out.err)
			}
			accepted := tc.status < 300 && !tc.transport
			if (out.phase.Acknowledgement != nil) != accepted || out.retry != tc.transport || !out.ambiguous {
				t.Fatal("uncertain wire was acknowledged or retried", out)
			}
			if accepted {
				proof := objectMetadataProof(t, out.err, tc.status, "raw")
				out.phase.Acknowledgement.Body[0] = '!'
				out.phase.Acknowledgement.Header.Set("X-Proof", "changed")
				out.phase.Attempts[0].Response.Body[0] = '?'
				if string(proof.Body) != "raw" || proof.Header.Get("X-Proof") != "kept" {
					t.Fatal("error proof aliases result")
				}
			}
		})
	}
}

func TestObjectCreateUploadSegmentRoundsAndExactCleanup(t *testing.T) {
	for _, unconfirmed := range []bool{false, true} {
		c := objectMetadataClient()
		var mu sync.Mutex
		puts, deletes := map[int]int{}, map[int]int{}
		cleanupCause := errors.New("cleanup ordering cause")
		c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			name := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
			index, err := strconv.Atoi(name)
			if err != nil {
				t.Error("unexpected manifest/original name", r.URL)
				return nil, err
			}
			mu.Lock()
			defer mu.Unlock()
			if r.Method == "DELETE" {
				deletes[index]++
				code := 202
				if index == 1 {
					code = 404
				}
				body := &objectMetadataBody{Reader: strings.NewReader("cleanup")}
				if unconfirmed {
					body.closeErr = cleanupCause
				}
				return objectMetadataWire(r, code, http.Header{"X-Proof": {"kept"}}, body), nil
			}
			puts[index]++
			data, err := io.ReadAll(r.Body)
			if err != nil || string(data) != string([]byte("abcdef")[index*2:index*2+2]) || r.Header.Get("If-None-Match") != "*" || r.Header.Get("ETag") != "" {
				t.Error("segment replay or condition", index, string(data), r.Header, err)
			}
			code := 201
			if unconfirmed && index == 1 {
				code = 202
			} else if !unconfirmed && index == 1 && puts[index] == 1 {
				code = 400
			} else if !unconfirmed && index == 2 {
				code = 503
			}
			return objectMetadataWire(r, code, http.Header{"X-Proof": {"kept"}, "ETag": {`"observed"`}}, io.NopCloser(strings.NewReader("segment"))), nil
		})
		p := createUploadPrepared(t, c)
		source := &objectCreateSource{data: []byte("abcdef"), size: 6}
		result := &CreateObjectResult{Mode: "slo", Size: 6}
		err := p.segmented(context.Background(), result, source, 2, nil)
		if err == nil || result.Manifest != nil || len(result.Segments) != 3 || result.SegmentPrefix == "" || result.ManifestAmbiguous {
			t.Fatal(result, err)
		}
		if unconfirmed {
			var semantic *ObjectCreateUnconfirmedSegmentError
			if !errors.As(err, &semantic) || semantic.Name != result.Segments[1].Name || !errors.Is(err, cleanupCause) || puts[1] != 1 || result.Segments[1].Created || !result.Segments[1].Ambiguous || result.Segments[1].Upload.Acknowledgement.StatusCode != 202 || deletes[1] != 0 || len(result.Cleanup) != 2 {
				t.Fatal("unconfirmed name became owned", result, err, puts, deletes)
			}
			if strings.Index(err.Error(), "does not confirm") > strings.Index(err.Error(), cleanupCause.Error()) {
				t.Fatal("cleanup cause replaced original ordering", err)
			}
		} else {
			if puts[0] != 1 || puts[1] != 2 || puts[2] != 2 || deletes[0] != 1 || deletes[1] != 1 || deletes[2] != 0 || len(result.Cleanup) != 2 || result.Segments[1].Ambiguous || !result.Segments[1].Created || !result.Segments[2].Ambiguous {
				t.Fatal(result, err, puts, deletes)
			}
			attempts := result.Segments[1].Upload.Attempts
			if len(attempts) != 2 || attempts[0].LogicalAttempt != 1 || attempts[1].LogicalAttempt != 2 || attempts[0].Error == nil || result.Segments[1].ETag == nil || *result.Segments[1].ETag != "observed" {
				t.Fatal("logical retry evidence lost", attempts, result.Segments[1])
			}
		}
	}
}

func TestObjectCreateUploadManifestAttemptsAndAmbiguity(t *testing.T) {
	for _, tc := range []struct {
		name               string
		codes              []int
		transport          bool
		success, ambiguous bool
		cleanup            int
	}{
		{"three clean rejections", []int{400, 400, 400}, false, false, false, 2},
		{"manifest412 remains retryable", []int{412, 412, 412}, false, false, false, 2},
		{"recovered errors retained", []int{400, 400, 201}, false, true, true, 0},
		{"server uncertainty sticky", []int{503, 400, 400}, false, false, true, 0},
		{"transport uncertainty sticky", []int{0, 400, 400}, true, false, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := objectMetadataClient()
			var mu sync.Mutex
			manifests, deletes := 0, 0
			transportCause := errors.New("manifest transport cause")
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				mu.Lock()
				defer mu.Unlock()
				if r.Method == "DELETE" {
					deletes++
					return objectMetadataOK(r, 204, http.Header{}), nil
				}
				if r.URL.Query().Get("multipart-manifest") != "put" {
					return objectMetadataOK(r, 201, http.Header{}), nil
				}
				code := tc.codes[manifests]
				manifests++
				if code == 0 {
					return nil, transportCause
				}
				return objectMetadataWire(r, code, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader("manifest"))), nil
			})
			p := createUploadPrepared(t, c)
			source := &objectCreateSource{data: []byte("abcd"), size: 4}
			result := &CreateObjectResult{Mode: "slo", Size: 4}
			err := p.segmented(context.Background(), result, source, 2, nil)
			if (err == nil) != tc.success || manifests != 3 || deletes != tc.cleanup || result.Manifest == nil || len(result.Manifest.Attempts) != 3 || result.ManifestAmbiguous != tc.ambiguous || len(result.Cleanup) != tc.cleanup {
				t.Fatal(result, err, manifests, deletes)
			}
			if result.Manifest.Attempts[0].Error == nil || result.Manifest.Attempts[2].LogicalAttempt != 3 {
				t.Fatal("physical/logical history missing", result.Manifest)
			}
			if tc.transport && result.Manifest.Attempts[0].Response != nil {
				t.Fatal("transport fabricated response", result.Manifest)
			}
			if tc.success && (result.Manifest.Acknowledgement == nil || result.Manifest.Acknowledgement.StatusCode != 201) {
				t.Fatal("recovery did not succeed", result, err)
			}
		})
	}
	c := objectMetadataClient()
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("manifest canceled")
	manifests, deletes := 0, 0
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method == "DELETE" {
			deletes++
			return objectMetadataOK(r, 204, http.Header{}), nil
		}
		if r.URL.Query().Get("multipart-manifest") != "put" {
			return objectMetadataOK(r, 201, http.Header{}), nil
		}
		manifests++
		return objectMetadataWire(r, 201, http.Header{"X-Proof": {"kept"}}, &objectMetadataBody{Reader: strings.NewReader("accepted"), onClose: func() { cancel(cause) }}), nil
	})
	p := createUploadPrepared(t, c)
	source := &objectCreateSource{data: []byte("abcd"), size: 4}
	result := &CreateObjectResult{Mode: "slo", Size: 4}
	err := p.segmented(ctx, result, source, 2, nil)
	if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || manifests != 1 || deletes != 0 || !result.ManifestAmbiguous || result.Manifest.Acknowledgement == nil {
		t.Fatal("accepted cancellation resent or cleaned", result, err, manifests, deletes)
	}
	objectMetadataProof(t, err, 201, "accepted")
}
