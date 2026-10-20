package v1

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func infoCoreProof(t *testing.T, err error, code int, body string) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != body || proof.Header.Get("X-Proof") != "kept" {
		t.Fatalf("lost actual response evidence: %v %#v", err, proof)
	}
	return proof
}

func TestInfoCoreModelAtomicRawSections(t *testing.T) {
	body := `{"swift":{"max_file_size":9223372036854775807,"plugin":{"opaque": [1, 2]}},"slo":{},"bulk_delete":null,"staticweb":{"enabled":true},"tempurl":{"methods":["GET"]},"links":42,"created_at":{},"updated_at":false,"admin":{"hidden":9007199254740993}}`
	var info Info
	if err := json.Unmarshal([]byte(body), &info); err != nil {
		t.Fatal(err)
	}
	if info.Swift == nil || info.SLO == nil || info.BulkDelete != nil || info.StaticWeb == nil || info.TempURL == nil || info.Links != nil || info.CreatedAt != nil || info.UpdatedAt != nil {
		t.Fatal("section presence or passive metadata lost", info)
	}
	if string(info.Swift["max_file_size"]) != "9223372036854775807" || string(info.Swift["plugin"]) != `{"opaque": [1, 2]}` || string(info.Body["admin"]) != `{"hidden":9007199254740993}` {
		t.Fatal("raw JSON converted", info.Body, info.Swift)
	}
	info.Swift["max_file_size"][0] = '!'
	if !strings.Contains(string(info.Body["swift"]), "9223372036854775807") {
		t.Fatal("canonical section aliases complete raw object")
	}
	before := string(info.Body["admin"])
	if err := json.Unmarshal([]byte(`{"swift":{},"slo":[]}`), &info); err == nil || string(info.Body["admin"]) != before || info.SLO == nil {
		t.Fatal("failed decoder partially replaced destination", info, err)
	}
	if err := json.Unmarshal([]byte(`{"swift":{"x":1},"swift":{"x":2},"tempurl":null}`), &info); err != nil || string(info.Swift["x"]) != "2" || info.TempURL != nil {
		t.Fatal("encoding/json last value or null policy changed", info, err)
	}
}

func TestInfoCoreModelFailuresAndAcceptedEvidence(t *testing.T) {
	bodies := []string{"null", "[]", `"text"`, `{"swift":`, string([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'})}
	for _, section := range []string{"swift", "slo", "bulk_delete", "staticweb", "tempurl"} {
		bodies = append(bodies, `{"`+section+`":false}`)
	}
	for _, body := range bodies {
		t.Run(body, func(t *testing.T) {
			c := tempURLKeyTestClient()
			calls, retries := 0, 0
			c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				return tempURLKeyTestWire(r, 200, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader(body))), nil
			})
			c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			if info, err := New(c).GetInfo(context.Background()); info != nil || err == nil {
				t.Fatal("malformed info accepted", info, err)
			} else {
				infoCoreProof(t, err, 200, body)
			}
			result, err := New(c).GetObjectSegmentSize(context.Background())
			proof := infoCoreProof(t, err, 200, body)
			if result == nil || result.Info != nil || result.Size != 0 || result.UsedFallback || result.RequestedSize != 1073741824 || string(result.Body) != body || calls != 2 || retries != 0 {
				t.Fatal("failure result or retry policy lost", result, calls, retries)
			}
			result.Body[0] = '!'
			result.Header.Set("X-Proof", "changed")
			if string(proof.Body) != body || proof.Header.Get("X-Proof") != "kept" {
				t.Fatal("mutable result corrupts error evidence")
			}
		})
	}
}

func TestInfoCoreSegmentSelection(t *testing.T) {
	for _, tc := range []struct {
		name, body                            string
		size                                  *int64
		requested, selected, maximum, minimum int64
	}{
		{"default", `{"swift":{"max_file_size":2000000000},"slo":{"min_segment_size":1}}`, nil, 1073741824, 1073741824, 2000000000, 1},
		{"empty sections", `{}`, nil, 1073741824, 0, 0, 0},
		{"nullable bounds", `{"swift":{"max_file_size":null},"slo":null}`, nil, 1073741824, 0, 0, 0},
		{"zero below minimum", `{"swift":{"max_file_size":100},"slo":{"min_segment_size":10}}`, ptrInt64(0), 0, 10, 100, 10},
		{"at maximum", `{"swift":{"max_file_size":100},"slo":{"min_segment_size":10}}`, ptrInt64(100), 100, 100, 100, 10},
		{"above maximum", `{"swift":{"max_file_size":100},"slo":{"min_segment_size":10}}`, ptrInt64(101), 101, 100, 100, 10},
		{"inverted maximum first", `{"swift":{"max_file_size":5},"slo":{"min_segment_size":10}}`, ptrInt64(7), 7, 5, 5, 10},
		{"inverted minimum second", `{"swift":{"max_file_size":5},"slo":{"min_segment_size":10}}`, ptrInt64(4), 4, 10, 5, 10},
		{"exact large bound", `{"swift":{"max_file_size":9223372036854775807},"slo":{"min_segment_size":9007199254740993}}`, ptrInt64(math.MaxInt64), math.MaxInt64, math.MaxInt64, math.MaxInt64, 9007199254740993},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := tempURLKeyTestClient()
			c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
				if r.Method != "GET" || r.URL.EscapedPath() != "/reverse%20proxy/info" || r.Body != nil {
					t.Fatal("capability read changed request", r.Method, r.URL, r.Body)
				}
				return tempURLKeyTestWire(r, 200, http.Header{}, io.NopCloser(strings.NewReader(tc.body))), nil
			})
			result, err := New(c).GetObjectSegmentSize(context.Background(), WithObjectSegmentSizeOpts(ObjectSegmentSizeOpts{Size: tc.size}))
			if err != nil || result == nil || result.RequestedSize != tc.requested || result.Size != tc.selected || result.MaxFileSize != tc.maximum || result.MinSegmentSize != tc.minimum || result.Info == nil || result.UsedFallback {
				t.Fatal(result, err)
			}
		})
	}
}

func ptrInt64(value int64) *int64 { return &value }

func TestInfoCoreSegmentBoundsAtomic(t *testing.T) {
	for _, bound := range []string{"-1", "1.0", "1e3", `"100"`, "true", "[]", "{}", "9223372036854775808"} {
		for _, section := range []string{"swift", "slo"} {
			t.Run(section+bound, func(t *testing.T) {
				body := `{"swift":{"max_file_size":100},"slo":{"min_segment_size":10}}`
				if section == "swift" {
					body = `{"swift":{"max_file_size":` + bound + `},"slo":{"min_segment_size":10}}`
				} else {
					body = `{"swift":{"max_file_size":100},"slo":{"min_segment_size":` + bound + `}}`
				}
				c := tempURLKeyTestClient()
				c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
					return tempURLKeyTestWire(r, 200, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader(body))), nil
				})
				result, err := New(c).GetObjectSegmentSize(context.Background(), WithObjectSegmentSize(20))
				proof := infoCoreProof(t, err, 200, body)
				if result == nil || result.Info == nil || result.RequestedSize != 20 || result.Size != 0 || result.MaxFileSize != 0 || result.MinSegmentSize != 0 || result.UsedFallback {
					t.Fatal("invalid bounds committed partially", result, err)
				}
				result.Info.Body["swift"][0] = '!'
				result.Info.Header.Set("X-Proof", "changed")
				if string(proof.Body) != body || proof.Header.Get("X-Proof") != "kept" || string(result.Body) != body || result.Header.Get("X-Proof") != "kept" {
					t.Fatal("Info/result/error evidence aliases")
				}
			})
		}
	}
}

func TestInfoCorePhysicalFallbackAndDirtyResponses(t *testing.T) {
	for _, code := range []int{404, 412} {
		for _, size := range []int64{0, 1073741824, 2684354562} {
			c := tempURLKeyTestClient()
			body := "opaque fallback \xff"
			c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
				return tempURLKeyTestWire(r, code, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader(body))), nil
			})
			result, err := New(c).GetObjectSegmentSize(context.Background(), WithObjectSegmentSize(size))
			want := size
			if size > 2684354561 {
				want = 2684354561
			}
			if err != nil || result == nil || !result.UsedFallback || result.Size != want || result.MaxFileSize != 2684354561 || result.MinSegmentSize != 0 || result.Info != nil || result.StatusCode != code || string(result.Body) != body {
				t.Fatal("physical fallback lost", result, err)
			}
		}
		for _, closeFault := range []bool{false, true} {
			c := tempURLKeyTestClient()
			fault := errors.New("body fault")
			body := &tempURLKeyTestBody{Reader: strings.NewReader("opaque")}
			if closeFault {
				body.closeErr = fault
			} else {
				read := false
				body.Reader = tempURLKeyTestReader(func(p []byte) (int, error) {
					if read {
						return 0, io.EOF
					}
					read = true
					return copy(p, "opaque"), errors.Join(io.EOF, fault)
				})
			}
			c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
				return tempURLKeyTestWire(r, code, http.Header{"X-Proof": {"kept"}}, body), nil
			})
			result, err := New(c).GetObjectSegmentSize(context.Background())
			infoCoreProof(t, err, code, "opaque")
			if result == nil || result.UsedFallback || result.Size != 0 || !errors.Is(err, fault) || body.closes.Load() != 1 {
				t.Fatal("dirty response fell back", result, err, body.closes.Load())
			}
		}
	}
}

func TestInfoCoreCallbackAndWireGuards(t *testing.T) {
	t.Run("callback guard stops later callbacks", func(t *testing.T) {
		c := tempURLKeyTestClient()
		cause := errors.New("callback cause")
		later, calls := 0, 0
		c.HTTPClient.Transport = tempURLKeyTestTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, cause })
		result, err := New(c).GetInfo(context.Background(), func(*GetInfoOpts) error { c.ResourceBase = "changed"; return cause }, func(*GetInfoOpts) error { later++; return nil })
		if result != nil || !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || later != 0 || calls != 0 {
			t.Fatal(result, err, later, calls)
		}
	})
	for _, restore := range []bool{false, true} {
		c := tempURLKeyTestClient()
		original := c.Endpoint
		body := &tempURLKeyTestBody{Reader: strings.NewReader(`{"swift":{}}`)}
		c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
			c.Endpoint = "http://cloud.invalid/changed"
			if restore {
				body.onClose = func() { c.Endpoint = original }
			}
			return tempURLKeyTestWire(r, 200, http.Header{"X-Proof": {"kept"}}, body), nil
		})
		result, err := New(c).GetObjectSegmentSize(context.Background())
		infoCoreProof(t, err, 200, `{"swift":{}}`)
		if result == nil || result.Info != nil || result.Size != 0 || !errors.Is(err, resource.ErrInvalidOption) || body.closes.Load() != 1 {
			t.Fatal("wire source fault lost", result, err, restore)
		}
	}
	t.Run("close cancellation retains custom cause", func(t *testing.T) {
		c := tempURLKeyTestClient()
		cause := errors.New("close cancellation")
		ctx, cancel := context.WithCancelCause(context.Background())
		body := &tempURLKeyTestBody{Reader: strings.NewReader("opaque"), onClose: func() { cancel(cause) }}
		c.HTTPClient.Transport = tempURLKeyTestTransport(func(r *http.Request) (*http.Response, error) {
			return tempURLKeyTestWire(r, 404, http.Header{"X-Proof": {"kept"}}, body), nil
		})
		result, err := New(c).GetObjectSegmentSize(ctx)
		infoCoreProof(t, err, 404, "opaque")
		if result == nil || result.UsedFallback || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
			t.Fatal(result, err)
		}
	})
}
