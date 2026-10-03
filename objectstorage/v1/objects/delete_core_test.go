package objects

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

const deleteCoreBulkOK = `{"Response Status":"200 OK","Response Body":"","Number Deleted":1,"Number Not Found":0,"Errors":[]}`

func TestObjectDeleteCoreWorkflowAndAcknowledgements(t *testing.T) {
	for _, tc := range []struct {
		name           string
		known          *bool
		head, deletion int
		slo            bool
	}{
		{"probe ordinary204", nil, 200, 204, false}, {"probe nativeHEAD204", nil, 204, 202, false},
		{"probe SLO200", nil, 200, 200, true}, {"probe SLO202", nil, 200, 202, true},
		{"known ordinary202", deleteCoreBool(false), 0, 202, false}, {"known ordinary204", deleteCoreBool(false), 0, 204, false},
		{"known SLO200", deleteCoreBool(true), 0, 200, true}, {"known SLO204", deleteCoreBool(true), 0, 204, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := objectMetadataClient()
			calls := 0
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Body != nil {
					t.Fatal("delete workflow sent a body")
				}
				if r.Method == "HEAD" {
					if tc.known != nil || calls != 1 || r.URL.RawQuery != "" {
						t.Fatal("unexpected discovery", calls, r.URL)
					}
					h := http.Header{"X-Proof": {"kept"}}
					if tc.slo {
						h.Set("X-Static-Large-Object", "true")
					}
					return objectMetadataWire(r, tc.head, h, io.NopCloser(strings.NewReader("head proof"))), nil
				}
				if r.Method != "DELETE" || (r.URL.Query().Get("multipart-manifest") == "delete") != tc.slo {
					t.Fatal("wrong deletion", r.Method, r.URL)
				}
				body := "ack proof"
				if tc.deletion == 200 {
					body = deleteCoreBulkOK
				}
				return objectMetadataWire(r, tc.deletion, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader(body))), nil
			})
			result, err := New(c).DeleteObject(context.Background(), "box", "folder/key", WithDeleteObjectOpts(DeleteObjectOpts{StaticLargeObject: tc.known}))
			wantCalls := 1
			if tc.known == nil {
				wantCalls++
			}
			if err != nil || result == nil || result.Deletion == nil || result.Deletion.StatusCode != tc.deletion || result.StaticLargeObject == nil || *result.StaticLargeObject != tc.slo || result.IgnoredMissing || calls != wantCalls || (result.Discovery != nil) != (tc.known == nil) || (result.Bulk != nil) != (tc.deletion == 200) {
				t.Fatal(result, err, calls)
			}
			if result.Discovery != nil && (result.Discovery.StatusCode != tc.head || string(result.Discovery.Body) != "head proof") {
				t.Fatal("discovery lost", result.Discovery)
			}
			if result.Bulk != nil {
				result.Bulk.Header.Set("X-Proof", "changed")
				if result.Deletion.Header.Get("X-Proof") != "kept" {
					t.Fatal("bulk header aliases acknowledgement")
				}
			}
		})
	}
}

func deleteCoreBool(value bool) *bool { return &value }

func TestObjectDeleteCoreSLOFlagProjection(t *testing.T) {
	for _, tc := range []struct {
		name          string
		headers       http.Header
		flag, invalid bool
	}{
		{"missing", nil, false, false}, {"no values", http.Header{"X-Static-Large-Object": nil}, false, false},
		{"true", http.Header{"x-static-large-object": {" \tTrUe\t "}}, true, false}, {"one", http.Header{"X-Static-Large-Object": {"1"}}, true, false},
		{"false", http.Header{"X-Static-Large-Object": {"FALSE"}}, false, false}, {"zero", http.Header{"X-Static-Large-Object": {"0"}}, false, false},
		{"empty", http.Header{"X-Static-Large-Object": {""}}, false, true}, {"other truthy", http.Header{"X-Static-Large-Object": {"yes"}}, false, true},
		{"multiple", http.Header{"X-Static-Large-Object": {"true", "false"}}, false, true},
		{"aliases", http.Header{"X-Static-Large-Object": {"true"}, "x-static-large-object": {"true"}}, false, true},
		{"control", http.Header{"X-Static-Large-Object": {"true\n"}}, false, true}, {"UTF8", http.Header{"X-Static-Large-Object": {"\xff"}}, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := objectMetadataClient()
			calls := 0
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					h := tc.headers.Clone()
					if h == nil {
						h = make(http.Header)
					}
					h.Set("X-Proof", "kept")
					return objectMetadataWire(r, 200, h, io.NopCloser(strings.NewReader("head"))), nil
				}
				return objectMetadataWire(r, 204, http.Header{}, io.NopCloser(strings.NewReader(""))), nil
			})
			result, err := New(c).DeleteObject(context.Background(), "box", "key")
			if tc.invalid {
				objectMetadataProof(t, err, 200, "head")
				if result == nil || result.StaticLargeObject != nil || result.Deletion != nil || calls != 1 {
					t.Fatal("invalid flag reached delete", result, err, calls)
				}
			} else if err != nil || result == nil || result.StaticLargeObject == nil || *result.StaticLargeObject != tc.flag || calls != 2 {
				t.Fatal(result, err, calls)
			}
		})
	}
}

func TestObjectDeleteCoreMissingAndPhaseProofs(t *testing.T) {
	for _, headMissing := range []bool{true, false} {
		for _, ignore := range []bool{true, false} {
			c := objectMetadataClient()
			calls := 0
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				code := 404
				if !headMissing && r.Method == "HEAD" {
					code = 200
				}
				return objectMetadataWire(r, code, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader(r.Method))), nil
			})
			result, err := New(c).DeleteObject(context.Background(), "box", "key", WithDeleteObjectIgnoreMissing(ignore))
			if ignore {
				if err != nil || result == nil || !result.IgnoredMissing || result.Discovery == nil || (result.StaticLargeObject == nil) != headMissing || (result.Deletion == nil) != headMissing {
					t.Fatal("missing proof lost", result, err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 404 || (result == nil) != headMissing || result != nil && (result.Discovery == nil || result.Deletion != nil || result.IgnoredMissing) {
					t.Fatal("strict404 was suppressed", result, err)
				}
			}
			if calls != 1 && headMissing || calls != 2 && !headMissing {
				t.Fatal("missing phases replayed", calls)
			}
		}
	}
	for _, headMissing := range []bool{true, false} {
		c := objectMetadataClient()
		fault := errors.New("dirty404 close")
		calls := 0
		c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if !headMissing && r.Method == "HEAD" {
				return objectMetadataOK(r, 200, http.Header{}), nil
			}
			return objectMetadataWire(r, 404, http.Header{"X-Proof": {"kept"}}, &objectMetadataBody{Reader: strings.NewReader("dirty"), closeErr: fault}), nil
		})
		result, err := New(c).DeleteObject(context.Background(), "box", "key")
		proof := objectMetadataProof(t, err, 404, "dirty")
		if result == nil || result.IgnoredMissing || !errors.Is(err, fault) {
			t.Fatal("dirty404 ignored", result, err)
		}
		response := result.Discovery
		if !headMissing {
			response = result.Deletion
		}
		response.Body[0] = '!'
		response.Header.Set("X-Proof", "changed")
		if string(proof.Body) != "dirty" || proof.Header.Get("X-Proof") != "kept" {
			t.Fatal("result corrupts error proof")
		}
	}
}

func TestObjectDeleteCoreBulkDecoderAndFailure(t *testing.T) {
	const valid = `{"Response Status":"206 Partial Content","Response Body":"","Number Deleted":9223372036854775807,"Number Not Found":9007199254740993,"Errors":[],"extension":{"large":9223372036854775808123},"created_at":{},"links":false}`
	var decoded ObjectDeleteBulkInfo
	if err := json.Unmarshal([]byte(" \r\n\t"+valid), &decoded); err != nil || decoded.ResponseCode != 206 || decoded.NumberDeleted != 9223372036854775807 || decoded.NumberNotFound != 9007199254740993 || decoded.CreatedAt != nil || decoded.Links != nil || string(decoded.Body["extension"]) != `{"large":9223372036854775808123}` {
		t.Fatal(decoded, err)
	}
	for _, body := range []string{
		"null", "[]", "{}", strings.Replace(valid, `"206 Partial Content"`, `"200 \u0085control"`, 1),
		strings.Replace(valid, `"206 Partial Content"`, `"200 "`, 1), strings.Replace(valid, `"206 Partial Content"`, `"600 Error"`, 1),
		strings.Replace(valid, `"Response Body":""`, `"Response Body":null`, 1), strings.Replace(valid, "9223372036854775807", "1e3", 1),
		strings.Replace(valid, "9007199254740993", "-1", 1), strings.Replace(valid, `"Errors":[]`, `"Errors":null`, 1),
		strings.Replace(valid, `"Errors":[]`, `"Errors":[["one"]]`, 1), strings.Replace(valid, `"Errors":[]`, `"Errors":[["one",null]]`, 1),
	} {
		if err := json.Unmarshal([]byte(body), &decoded); err == nil || decoded.ResponseCode != 206 || string(decoded.Body["extension"]) != `{"large":9223372036854775808123}` {
			t.Fatal("decoder was not atomic", decoded, err)
		}
	}
	for _, body := range []string{
		`{"Response Status":"404 Not Found","Response Body":"missing manifest","Number Deleted":0,"Number Not Found":1,"Errors":[]}`,
		`{"Response Status":"200 OK","Response Body":"partial","Number Deleted":1,"Number Not Found":0,"Errors":[["/box/key%20encoded","arbitrary lookup message"]]}`,
	} {
		c := objectMetadataClient()
		c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			return objectMetadataWire(r, 200, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader(body))), nil
		})
		result, err := New(c).DeleteObject(context.Background(), "box", "key", WithDeleteObjectStaticLargeObject(true))
		proof := objectMetadataProof(t, err, 200, body)
		var embedded *ObjectDeleteBulkError
		if result == nil || result.Bulk == nil || result.IgnoredMissing || !errors.As(err, &embedded) {
			t.Fatal("embedded error suppressed", result, err)
		}
		status, message := embedded.ResponseStatus, embedded.ResponseBody
		result.Bulk.ResponseStatus, result.Bulk.ResponseBody = "changed", "changed"
		if len(result.Bulk.Errors) != 0 {
			result.Bulk.Errors[0].Error = "changed"
			if embedded.Errors[0].Error != "arbitrary lookup message" {
				t.Fatal("embedded error aliases result")
			}
		}
		result.Deletion.Body[0] = '!'
		if string(proof.Body) != body || embedded.ResponseStatus != status || embedded.ResponseBody != message {
			t.Fatal("bulk evidence aliases result")
		}
	}
	for _, body := range []string{"null", strings.Replace(valid, `"Errors":[]`, `"Errors":false`, 1)} {
		c := objectMetadataClient()
		c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			return objectMetadataWire(r, 200, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader(body))), nil
		})
		result, err := New(c).DeleteObject(context.Background(), "box", "key", WithDeleteObjectStaticLargeObject(true))
		objectMetadataProof(t, err, 200, body)
		if result == nil || result.Deletion == nil || result.Bulk != nil || result.IgnoredMissing {
			t.Fatal("parse failure result lost", result, err)
		}
	}
}

func TestObjectDeleteCoreBodyAndSourceFailures(t *testing.T) {
	for _, driftInRead := range []bool{false, true} {
		c := objectMetadataClient()
		original := c.Endpoint
		calls := 0
		body := &objectMetadataBody{Reader: strings.NewReader("head bytes"), onClose: func() { c.Endpoint = original }}
		if driftInRead {
			body.Reader = objectMetadataReader(func(p []byte) (int, error) { c.Endpoint += "changed"; return copy(p, "head bytes"), io.EOF })
		}
		c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if !driftInRead {
				c.Endpoint += "changed"
			}
			return objectMetadataWire(r, 200, http.Header{"X-Proof": {"kept"}}, body), nil
		})
		result, err := New(c).DeleteObject(context.Background(), "box", "key")
		objectMetadataProof(t, err, 200, "head bytes")
		if result == nil || result.StaticLargeObject != nil || result.Deletion != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || body.closes.Load() != 1 {
			t.Fatal("sticky source fault lost", result, err, calls)
		}
	}
	c := objectMetadataClient()
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	readErr, closeErr, cause := errors.New("read fault"), errors.New("close fault"), errors.New("caller cancellation")
	calls, retries := 0, 0
	body := &objectMetadataBody{Reader: objectMetadataReader(func(p []byte) (int, error) { return copy(p, "partial delete"), errors.Join(io.EOF, readErr) }), closeErr: closeErr, onClose: func() { c.ResourceBase = "changed"; cancel(cause) }}
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method == "HEAD" {
			return objectMetadataOK(r, 200, http.Header{}), nil
		}
		return objectMetadataWire(r, 202, http.Header{"X-Proof": {"kept"}}, body), nil
	})
	c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		retries++
		return nil
	}
	result, err := New(c).DeleteObject(ctx, "box", "key")
	objectMetadataProof(t, err, 202, "partial delete")
	if result == nil || result.Discovery == nil || result.Deletion == nil || result.StaticLargeObject == nil || *result.StaticLargeObject || result.Bulk != nil || calls != 2 || retries != 0 || body.closes.Load() != 1 {
		t.Fatal(result, err, calls, retries)
	}
	for _, expected := range []error{io.EOF, readErr, closeErr, cause, context.Canceled, resource.ErrInvalidOption} {
		if !errors.Is(err, expected) {
			t.Fatal("lost cause", expected, err)
		}
	}
}

func TestObjectDeleteCoreNativePolicyGuards(t *testing.T) {
	for _, hook := range []string{"retry", "backoff", "reauth", "redirect"} {
		t.Run(hook, func(t *testing.T) {
			c := objectMetadataClient()
			cause := errors.New("hook cause")
			calls, hooks := 0, 0
			mutate := func() error { hooks++; c.ResourceBase = "changed"; return cause }
			code := map[string]int{"retry": 503, "backoff": 429, "reauth": 401, "redirect": 307}[hook]
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				h := http.Header{}
				if hook == "redirect" {
					h.Set("Location", r.URL.String())
				}
				return objectMetadataWire(r, code, h, io.NopCloser(strings.NewReader("original"))), nil
			})
			switch hook {
			case "retry":
				c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return mutate() }
			case "backoff":
				c.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error { return mutate() }
			case "reauth":
				c.ReauthFunc = func(context.Context) error { return mutate() }
			case "redirect":
				c.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { return mutate() }
			}
			result, err := New(c).DeleteObject(context.Background(), "box", "key", WithDeleteObjectStaticLargeObject(false))
			causeErr := err
			if hook == "reauth" {
				var native *gophercloud.ErrUnableToReauthenticate
				if !errors.As(err, &native) || !gophercloud.ResponseCodeIs(native.ErrOriginal, http.StatusUnauthorized) {
					t.Fatal("native reauthentication evidence lost", err)
				}
				causeErr = native.ErrReauth
			}
			if result != nil || !errors.Is(causeErr, cause) || !errors.Is(causeErr, resource.ErrInvalidOption) || calls != 1 || hooks != 1 {
				t.Fatal("native source guard failed", result, err, calls, hooks)
			}
		})
	}
	t.Run("same-target redirect Accept aliases remain SDK owned", func(t *testing.T) {
		c := objectMetadataClient()
		calls := 0
		c.HTTPClient.CheckRedirect = func(next *http.Request, _ []*http.Request) error {
			next.Header = http.Header{"Accept": {"text/xml"}, "accept": {"text/plain"}, "X-Trace": {"redirect"}}
			return nil
		}
		c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return objectMetadataWire(r, 307, http.Header{"Location": {r.URL.String()}}, io.NopCloser(strings.NewReader(""))), nil
			}
			accepts := 0
			for key, values := range r.Header {
				if strings.EqualFold(key, "Accept") {
					accepts++
					if len(values) != 1 || values[0] != "application/json" {
						t.Fatal("Accept override reached wire", r.Header)
					}
				}
			}
			if accepts != 1 || r.Header.Get("X-Trace") != "redirect" || r.Header.Get("X-Auth-Token") != "token" {
				t.Fatal("redirect policy lost", r.Header)
			}
			return objectMetadataWire(r, 200, http.Header{}, io.NopCloser(strings.NewReader(deleteCoreBulkOK))), nil
		})
		result, err := New(c).DeleteObject(context.Background(), "box", "key", WithDeleteObjectStaticLargeObject(true))
		if result == nil || result.Bulk == nil || err != nil || calls != 2 {
			t.Fatal(result, err, calls)
		}
	})
}
