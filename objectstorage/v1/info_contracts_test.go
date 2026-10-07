package v1_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/objectstorage/v1"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func infoContractClient(f tempKeyTransport) *gophercloud.ServiceClient {
	c := tempKeyClient(f)
	c.Endpoint = "https://swift.invalid/api%2Fproxy/v1/AUTH_account/"
	c.ResourceBase = "unused resource base"
	return c
}

func infoContractResponse(status int, body *tempKeyBody) *http.Response {
	r := tempKeyResponse(status, body, nil)
	r.Header.Set("X-Trans-Id", "info-proof")
	return r
}

func infoContractWire(status int, body string) *http.Response {
	return infoContractResponse(status, &tempKeyBody{Reader: strings.NewReader(body)})
}

func infoContractProof(t *testing.T, err error, status int, body string) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Trans-Id") != "info-proof" {
		t.Fatalf("proof=%+v err=%v", proof, err)
	}
	return proof
}

func infoContractSize(value int64) *int64 { return &value }

func TestObjectInfoContractsRoutesAndModels(t *testing.T) {
	t.Run("catalog escaped prefix reaches the wire and each fetch is fresh", func(t *testing.T) {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			tempKeyNoBody(t, r)
			if r.Method != http.MethodGet || r.URL.EscapedPath() != "/api%2Fproxy/info" || r.RequestURI != "/api%2Fproxy/info" {
				t.Errorf("wire=%s %s URI=%q", r.Method, r.URL, r.RequestURI)
			}
			w.Header().Set("X-Trans-Id", "wire-info")
			_, _ = io.WriteString(w, `{"swift":{"max_file_size":42}}`)
		}))
		defer server.Close()
		client := &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{HTTPClient: *server.Client()}, Type: "object-store", Endpoint: server.URL + "/api%2Fproxy/v1/AUTH_account/", ResourceBase: "https://foreign.invalid/unused/"}
		service := v1.New(client)
		for range 2 {
			got, err := service.GetInfo(context.Background())
			if err != nil || got == nil || string(got.Swift["max_file_size"]) != "42" || got.StatusCode != 200 || got.Header.Get("X-Trans-Id") != "wire-info" {
				t.Fatalf("info=%+v err=%v", got, err)
			}
		}
		if calls.Load() != 2 {
			t.Fatalf("requests=%d", calls.Load())
		}
	})
	t.Run("version replacement preserves literal catalog path bytes", func(t *testing.T) {
		for _, tc := range []struct{ endpoint, path string }{
			{"https://swift.invalid", "/info"},
			{"https://swift.invalid/prefix/", "/prefix/info"},
			{"https://swift.invalid/swift/v1.0/A", "/swift/info"},
			{"https://swift.invalid/swift/v111/A", "/swift/info"},
			{"https://swift.invalid/swift/v1beta/A", "/swift/infobeta/A"},
			{"https://swift.invalid/info/v1/A", "/info/info"},
			{"https://swift.invalid/swift/v%31/A", "/swift/v%31/A/info"},
			{"https://swift.invalid/prefix;param", "/prefix;param/info"},
		} {
			calls := 0
			client := infoContractClient(func(r *http.Request) (*http.Response, error) {
				calls++
				tempKeyNoBody(t, r)
				if r.Method != http.MethodGet || r.URL.EscapedPath() != tc.path {
					t.Errorf("endpoint=%q wire=%s %s", tc.endpoint, r.Method, r.URL)
				}
				return infoContractWire(200, `{}`), nil
			})
			client.Endpoint = tc.endpoint
			if got, err := v1.New(client).GetInfo(context.Background()); err != nil || got == nil || calls != 1 {
				t.Fatalf("endpoint=%q info=%+v err=%v calls=%d", tc.endpoint, got, err, calls)
			}
		}
	})
	t.Run("canonical maps own bytes while plugins and inherited metadata remain raw", func(t *testing.T) {
		const body = `{"swift":{"max_file_size":9007199254740993,"nullable":null},"slo":null,"bulk_delete":{},"staticweb":{"nested":[]},"tempurl":{},"plugin":{"huge":9223372036854775808123},"admin":{"private":true},"links":"passive","created_at":{"custom":true},"updated_at":42}`
		client := infoContractClient(func(*http.Request) (*http.Response, error) { return infoContractWire(200, body), nil })
		got, err := v1.New(client).GetInfo(context.Background())
		if err != nil || got == nil || got.SLO != nil || got.BulkDelete == nil || got.StaticWeb == nil || got.TempURL == nil || got.CreatedAt != nil || got.UpdatedAt != nil || got.Links != nil {
			t.Fatalf("info=%+v err=%v", got, err)
		}
		if string(got.Body["plugin"]) != `{"huge":9223372036854775808123}` || string(got.Body["admin"]) != `{"private":true}` || string(got.Body["links"]) != `"passive"` || string(got.Swift["max_file_size"]) != "9007199254740993" {
			t.Fatalf("raw=%v swift=%v", got.Body, got.Swift)
		}
		got.Swift["max_file_size"][0] = '1'
		got.Swift["extra"] = json.RawMessage(`true`)
		if string(got.Body["swift"]) != `{"max_file_size":9007199254740993,"nullable":null}` {
			t.Fatal("canonical section aliases raw root proof")
		}
		delete(got.Body, "tempurl")
		if got.TempURL == nil {
			t.Fatal("raw field deletion changed canonical section")
		}
	})
}

func TestObjectInfoContractsOptionsAndSource(t *testing.T) {
	t.Run("factory full option and callback snapshots", func(t *testing.T) {
		headers := map[string]string{"x-call": "factory"}
		full := v1.WithGetInfoOpts(v1.GetInfoOpts{Headers: headers})
		headers["x-call"] = "changed"
		extra := map[string]string{"X-Extra": "snapshot"}
		overlay := v1.WithGetInfoHeaders(extra)
		extra["X-Extra"] = "changed"
		callbacks := 0
		client := infoContractClient(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Call") != "final" || r.Header.Get("X-Extra") != "snapshot" || r.Header.Get("X-Discarded") != "" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != "token-two" {
				t.Errorf("headers=%v", r.Header)
			}
			return infoContractWire(200, `{}`), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		var retained *v1.GetInfoOpts
		got, err := v1.New(client).GetInfo(context.Background(), v1.WithGetInfoHeader("X-Discarded", "yes"), full, overlay, v1.WithGetInfoHeader("X-Call", "final"), func(cfg *v1.GetInfoOpts) error {
			callbacks++
			retained = cfg
			client.MoreHeaders["X-Source"] = "valid later value"
			client.SetToken("token-two")
			return nil
		}, func(*v1.GetInfoOpts) error {
			callbacks++
			retained.Headers["X-Call"] = "late retained mutation"
			return nil
		})
		if err != nil || got == nil || callbacks != 2 || client.MoreHeaders["X-Source"] != "valid later value" {
			t.Fatalf("info=%+v err=%v callbacks=%d", got, err, callbacks)
		}
	})
	t.Run("segment size pointer full reset and options can be reused concurrently", func(t *testing.T) {
		var calls atomic.Int32
		client := infoContractClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Call") != "shared" {
				t.Errorf("headers=%v", r.Header)
			}
			return infoContractWire(200, `{"swift":{"max_file_size":2147483648},"slo":{"min_segment_size":0}}`), nil
		})
		size := int64(123)
		option := v1.WithObjectSegmentSizeOpts(v1.ObjectSegmentSizeOpts{Headers: map[string]string{"X-Call": "shared"}, Size: &size})
		size = 456
		service := v1.New(client)
		var wg sync.WaitGroup
		for range 4 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				got, err := service.GetObjectSegmentSize(context.Background(), option)
				if err != nil || got == nil || got.RequestedSize != 123 || got.Size != 123 {
					t.Errorf("segment=%+v err=%v", got, err)
				}
			}()
		}
		wg.Wait()
		got, err := service.GetObjectSegmentSize(context.Background(), option, v1.WithoutObjectSegmentSize())
		if err != nil || got == nil || got.RequestedSize != 1073741824 || got.Size != 1073741824 || calls.Load() != 5 {
			t.Fatalf("segment=%+v err=%v calls=%d", got, err, calls.Load())
		}
	})
	t.Run("identity changes stop before HTTP and auth headers are reserved", func(t *testing.T) {
		for _, mutate := range []func(*v1.Service, *gophercloud.ServiceClient){
			func(_ *v1.Service, c *gophercloud.ServiceClient) { c.Endpoint += "changed/" },
			func(_ *v1.Service, c *gophercloud.ServiceClient) { c.ResourceBase += "changed/" },
			func(_ *v1.Service, c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} },
			func(_ *v1.Service, c *gophercloud.ServiceClient) { c.Type = "compute" },
			func(_ *v1.Service, c *gophercloud.ServiceClient) { c.Microversion = "changed" },
			func(s *v1.Service, _ *gophercloud.ServiceClient) { s.Objects = nil },
		} {
			calls := 0
			client := infoContractClient(func(*http.Request) (*http.Response, error) { calls++; return infoContractWire(200, `{}`), nil })
			service := v1.New(client)
			got, err := service.GetInfo(context.Background(), func(*v1.GetInfoOpts) error { mutate(service, client); return nil })
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatalf("info=%+v err=%v calls=%d", got, err, calls)
			}
		}
		for _, headers := range []map[string]string{{"Cookie": "secret"}, {"X-Service-Token": "secret"}, {"X-Call": "one", "x-call": "two"}} {
			client := infoContractClient(func(*http.Request) (*http.Response, error) {
				t.Error("HTTP after invalid headers")
				return nil, errors.New("unexpected HTTP")
			})
			if got, err := v1.New(client).GetInfo(context.Background(), v1.WithGetInfoHeaders(headers)); got != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("headers=%v info=%+v err=%v", headers, got, err)
			}
		}
	})
}

func TestObjectInfoContractsAcceptedResponse(t *testing.T) {
	t.Run("invalid roots and sections fail atomically with actual accepted proof", func(t *testing.T) {
		for _, body := range []string{`null`, `[]`, `{`, "{\"plugin\":\"\xff\"}", `{"swift":[]}`, `{"tempurl":true}`} {
			calls := 0
			client := infoContractClient(func(*http.Request) (*http.Response, error) { calls++; return infoContractWire(200, body), nil })
			got, err := v1.New(client).GetInfo(context.Background())
			if got != nil || err == nil || calls != 1 {
				t.Fatalf("body=%q info=%+v err=%v calls=%d", body, got, err, calls)
			}
			infoContractProof(t, err, 200, body)
			segment, err := v1.New(client).GetObjectSegmentSize(context.Background())
			if segment == nil || segment.Info != nil || segment.Size != 0 || segment.UsedFallback || string(segment.Body) != body || segment.StatusCode != 200 || calls != 2 {
				t.Fatalf("body=%q segment=%+v err=%v calls=%d", body, segment, err, calls)
			}
			infoContractProof(t, err, 200, body)
		}
	})
	t.Run("read close cancellation and source errors preserve partial proof without replay", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		readErr, closeErr, cause := errors.New("read failure"), errors.New("close failure"), errors.New("caller cause")
		calls, retries := 0, 0
		var client *gophercloud.ServiceClient
		body := &tempKeyBody{Reader: tempKeyReader(func(p []byte) (int, error) { return copy(p, "partial"), errors.Join(io.EOF, readErr) }), closeErr: closeErr, onClose: func() { client.ResourceBase += "changed"; cancel(cause) }}
		client = infoContractClient(func(*http.Request) (*http.Response, error) { calls++; return infoContractResponse(200, body), nil })
		client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			retries++
			return nil
		}
		got, err := v1.New(client).GetObjectSegmentSize(ctx)
		if got == nil || got.Info != nil || got.Size != 0 || got.UsedFallback || calls != 1 || retries != 0 || body.closes.Load() != 1 {
			t.Fatalf("segment=%+v err=%v calls=%d retries=%d closes=%d", got, err, calls, retries, body.closes.Load())
		}
		for _, expected := range []error{io.EOF, readErr, closeErr, context.Canceled, cause, resource.ErrInvalidOption} {
			if !errors.Is(err, expected) {
				t.Errorf("missing cause=%v err=%v", expected, err)
			}
		}
		proof := infoContractProof(t, err, 200, "partial")
		got.Body[0] = 'X'
		got.Header.Set("X-Trans-Id", "changed")
		if string(proof.Body) != "partial" || proof.Header.Get("X-Trans-Id") != "info-proof" {
			t.Fatal("mutable result aliases error proof")
		}
	})
	t.Run("expanded native accepted codes cannot broaden the public status", func(t *testing.T) {
		calls := 0
		body := &tempKeyBody{Reader: strings.NewReader("unexpected201")}
		client := infoContractClient(func(*http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return infoContractWire(503, "retry"), nil
			}
			return infoContractResponse(201, body), nil
		})
		client.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, _ error, _ uint) error {
			options.OkCodes = []int{200, 201}
			return nil
		}
		got, err := v1.New(client).GetInfo(context.Background())
		var native gophercloud.ErrUnexpectedResponseCode
		var proof *resource.ResponseError
		if got != nil || !errors.As(err, &native) || native.Actual != 201 || !reflect.DeepEqual(native.Expected, []int{200}) || string(native.Body) != "unexpected201" || errors.As(err, &proof) || calls != 2 || body.closes.Load() != 1 {
			t.Fatalf("info=%+v err=%v native=%+v calls=%d closes=%d", got, err, native, calls, body.closes.Load())
		}
	})
}

func TestObjectInfoContractsSegmentBounds(t *testing.T) {
	for _, tc := range []struct {
		name, body              string
		requested               *int64
		request, max, min, size int64
	}{
		{"default", `{"swift":{"max_file_size":2147483648},"slo":{"min_segment_size":1}}`, nil, 1073741824, 2147483648, 1, 1073741824},
		{"explicit zero", `{"swift":{"max_file_size":100},"slo":{"min_segment_size":0}}`, infoContractSize(0), 0, 100, 0, 0},
		{"above max", `{"swift":{"max_file_size":100},"slo":{"min_segment_size":10}}`, infoContractSize(200), 200, 100, 10, 100},
		{"below min", `{"swift":{"max_file_size":100},"slo":{"min_segment_size":10}}`, infoContractSize(1), 1, 100, 10, 10},
		{"inverted max first", `{"swift":{"max_file_size":20},"slo":{"min_segment_size":40}}`, infoContractSize(30), 30, 20, 40, 20},
		{"inverted min second", `{"swift":{"max_file_size":20},"slo":{"min_segment_size":40}}`, infoContractSize(10), 10, 20, 40, 40},
		{"missing sections", `{}`, nil, 1073741824, 0, 0, 0},
		{"null bounds", `{"swift":{"max_file_size":null},"slo":null}`, nil, 1073741824, 0, 0, 0},
		{"exact large integers", `{"swift":{"max_file_size":9223372036854775807},"slo":{"min_segment_size":0}}`, infoContractSize(9007199254740993), 9007199254740993, 9223372036854775807, 0, 9007199254740993},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := infoContractClient(func(*http.Request) (*http.Response, error) { calls++; return infoContractWire(200, tc.body), nil })
			got, err := v1.New(client).GetObjectSegmentSize(context.Background(), v1.WithObjectSegmentSizeOpts(v1.ObjectSegmentSizeOpts{Size: tc.requested}))
			if err != nil || got == nil || got.RequestedSize != tc.request || got.Size != tc.size || got.MaxFileSize != tc.max || got.MinSegmentSize != tc.min || got.Info == nil || got.UsedFallback || got.StatusCode != 200 || string(got.Body) != tc.body || calls != 1 {
				t.Fatalf("segment=%+v err=%v calls=%d", got, err, calls)
			}
			got.Info.Header.Set("X-Trans-Id", "changed")
			if got.Header.Get("X-Trans-Id") != "info-proof" {
				t.Fatal("Info aliases result response header")
			}
		})
	}
	t.Run("invalid bounds retain decoded info and atomic bounds", func(t *testing.T) {
		for _, token := range []string{"-1", "1.5", "1e3", `"123"`, "true", "9223372036854775808"} {
			body := `{"swift":{"max_file_size":100},"slo":{"min_segment_size":` + token + `}}`
			client := infoContractClient(func(*http.Request) (*http.Response, error) { return infoContractWire(200, body), nil })
			got, err := v1.New(client).GetObjectSegmentSize(context.Background())
			if got == nil || got.Info == nil || got.Size != 0 || got.MaxFileSize != 0 || got.MinSegmentSize != 0 || got.UsedFallback || string(got.Info.Swift["max_file_size"]) != "100" {
				t.Fatalf("token=%s segment=%+v err=%v", token, got, err)
			}
			infoContractProof(t, err, 200, body)
		}
	})
}

func TestObjectInfoContractsNativePolicy(t *testing.T) {
	t.Run("accepted transport drift preserves proof even when Close restores the source", func(t *testing.T) {
		const raw = `{"swift":{"max_file_size":100}}`
		calls := 0
		var client *gophercloud.ServiceClient
		body := &tempKeyBody{Reader: strings.NewReader(raw), onClose: func() { client.Endpoint = "https://swift.invalid/api%2Fproxy/v1/AUTH_account/" }}
		client = infoContractClient(func(*http.Request) (*http.Response, error) {
			calls++
			client.Endpoint = "https://foreign.invalid/changed/"
			return infoContractResponse(200, body), nil
		})
		got, err := v1.New(client).GetObjectSegmentSize(context.Background())
		if got == nil || got.Info != nil || got.Size != 0 || got.UsedFallback || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || body.closes.Load() != 1 {
			t.Fatalf("segment=%+v err=%v calls=%d closes=%d", got, err, calls, body.closes.Load())
		}
		infoContractProof(t, err, 200, raw)
	})
	t.Run("retry header overwrite and live original authentication", func(t *testing.T) {
		calls, hooks := 0, 0
		var client *gophercloud.ServiceClient
		client = infoContractClient(func(r *http.Request) (*http.Response, error) {
			calls++
			tempKeyNoBody(t, r)
			if r.Method != http.MethodGet || r.URL.String() != "https://swift.invalid/api%2Fproxy/info" {
				t.Errorf("wire=%s %s", r.Method, r.URL)
			}
			if calls == 1 {
				if r.Header.Get("X-Capture") != "original" || r.Header.Get("X-Auth-Token") != "token-one" {
					t.Errorf("first headers=%v", r.Header)
				}
				return infoContractWire(503, "retry"), nil
			}
			if r.Header.Get("X-Capture") != "" || r.Header.Get("X-Native") != "advanced" || r.Header.Get("X-Auth-Token") != "token-two" {
				t.Errorf("retry headers=%v", r.Header)
			}
			return infoContractWire(200, `{}`), nil
		})
		client.MoreHeaders = map[string]string{"X-Capture": "original"}
		provider := client.ProviderClient
		client.RetryFunc = func(_ context.Context, method, target string, options *gophercloud.RequestOpts, original error, count uint) error {
			hooks++
			if method != http.MethodGet || target != "https://swift.invalid/api%2Fproxy/info" || count != 1 || !gophercloud.ResponseCodeIs(original, 503) {
				t.Errorf("hook=%s %s count=%d err=%v", method, target, count, original)
			}
			client.MoreHeaders["X-Capture"] = "valid later source"
			client.SetToken("token-two")
			options.MoreHeaders = map[string]string{"X-Native": "advanced"}
			return nil
		}
		got, err := v1.New(client).GetInfo(context.Background())
		if err != nil || got == nil || calls != 2 || hooks != 1 || client.ProviderClient != provider || client.MoreHeaders["X-Capture"] != "valid later source" {
			t.Fatalf("info=%+v err=%v calls=%d hooks=%d", got, err, calls, hooks)
		}
	})
	t.Run("retry source and body ownership changes stop before the next attempt", func(t *testing.T) {
		for _, sourceChange := range []bool{true, false} {
			calls := 0
			client := infoContractClient(func(*http.Request) (*http.Response, error) { calls++; return infoContractWire(503, "original503"), nil })
			client.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, _ error, _ uint) error {
				if sourceChange {
					client.ResourceBase += "changed"
				} else {
					options.RawBody = strings.NewReader("injected")
				}
				return nil
			}
			got, err := v1.New(client).GetInfo(context.Background())
			var native gophercloud.ErrUnexpectedResponseCode
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != "original503" || calls != 1 {
				t.Fatalf("sourceChange=%v info=%+v err=%v native=%+v calls=%d", sourceChange, got, err, native, calls)
			}
		}
	})
}

func TestObjectInfoContractsFallbackEvidence(t *testing.T) {
	t.Run("only clean physical404 and412 use fallback and bypass native retry", func(t *testing.T) {
		for _, status := range []int{404, 412} {
			for _, requested := range []int64{1073741824, 9007199254740993} {
				calls, hooks := 0, 0
				client := infoContractClient(func(*http.Request) (*http.Response, error) {
					calls++
					return infoContractWire(status, "opaque fallback body"), nil
				})
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks++
					return errors.New("should bypass")
				}
				got, err := v1.New(client).GetObjectSegmentSize(context.Background(), v1.WithObjectSegmentSize(requested))
				want := requested
				if requested > 2684354561 {
					want = 2684354561
				}
				if err != nil || got == nil || got.Info != nil || got.RequestedSize != requested || got.Size != want || got.MaxFileSize != 2684354561 || got.MinSegmentSize != 0 || !got.UsedFallback || got.StatusCode != status || string(got.Body) != "opaque fallback body" || got.Header.Get("X-Trans-Id") != "info-proof" || calls != 1 || hooks != 0 {
					t.Fatalf("status=%d segment=%+v err=%v calls=%d hooks=%d", status, got, err, calls, hooks)
				}
			}
		}
	})
	t.Run("dirty physical fallback responses retain causes and never select a size", func(t *testing.T) {
		for _, status := range []int{404, 412} {
			cause := errors.New("body close failure")
			var client *gophercloud.ServiceClient
			body := &tempKeyBody{Reader: strings.NewReader("dirty physical"), closeErr: cause, onClose: func() { client.MoreHeaders["Cookie"] = "reserved late header" }}
			calls := 0
			client = infoContractClient(func(*http.Request) (*http.Response, error) { calls++; return infoContractResponse(status, body), nil })
			client.MoreHeaders = map[string]string{"X-Ordinary": "valid"}
			got, err := v1.New(client).GetObjectSegmentSize(context.Background())
			if got == nil || got.Size != 0 || got.UsedFallback || got.Info != nil || !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || body.closes.Load() != 1 {
				t.Fatalf("status=%d segment=%+v err=%v calls=%d", status, got, err, calls)
			}
			infoContractProof(t, err, status, "dirty physical")
		}
	})
	t.Run("fallback status cannot hide read or custom context failures", func(t *testing.T) {
		for _, status := range []int{404, 412} {
			ctx, cancel := context.WithCancelCause(context.Background())
			readErr, cause := errors.New("fallback read failure"), errors.New("fallback caller cause")
			body := &tempKeyBody{Reader: tempKeyReader(func(p []byte) (int, error) { return copy(p, "partial fallback"), errors.Join(io.EOF, readErr) }), onClose: func() { cancel(cause) }}
			client := infoContractClient(func(*http.Request) (*http.Response, error) { return infoContractResponse(status, body), nil })
			got, err := v1.New(client).GetObjectSegmentSize(ctx)
			cancel(nil)
			if got == nil || got.Size != 0 || got.UsedFallback || body.closes.Load() != 1 {
				t.Fatalf("status=%d segment=%+v err=%v closes=%d", status, got, err, body.closes.Load())
			}
			for _, expected := range []error{io.EOF, readErr, context.Canceled, cause} {
				if !errors.Is(err, expected) {
					t.Errorf("status=%d missing=%v err=%v", status, expected, err)
				}
			}
			infoContractProof(t, err, status, "partial fallback")
		}
	})
	t.Run("nested transport retry and reauth404 are errors", func(t *testing.T) {
		for _, mode := range []string{"transport", "retry", "reauth"} {
			nested := &gophercloud.ErrUnexpectedResponseCode{Actual: 404, Body: []byte("nested404")}
			calls := 0
			client := infoContractClient(func(*http.Request) (*http.Response, error) {
				calls++
				if mode == "transport" {
					return nil, nested
				}
				if mode == "reauth" {
					return infoContractWire(401, "actual401"), nil
				}
				return infoContractWire(503, "actual503"), nil
			})
			if mode == "retry" {
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return nested }
			}
			if mode == "reauth" {
				client.ReauthFunc = func(context.Context) error { return nested }
			}
			got, err := v1.New(client).GetObjectSegmentSize(context.Background())
			if got != nil || err == nil || calls != 1 {
				t.Fatalf("mode=%s segment=%+v err=%v calls=%d", mode, got, err, calls)
			}
			if mode == "reauth" {
				var reauth *gophercloud.ErrUnableToReauthenticate
				if !errors.As(err, &reauth) || reauth.ErrReauth != nested || !gophercloud.ResponseCodeIs(reauth.ErrOriginal, 401) {
					t.Fatalf("reauth=%+v err=%v", reauth, err)
				}
			} else if !errors.Is(err, nested) {
				t.Fatalf("mode=%s nested error lost=%v", mode, err)
			}
		}
	})
	t.Run("GetInfo404 keeps its native retry policy and other statuses do not fallback", func(t *testing.T) {
		calls, hooks := 0, 0
		client := infoContractClient(func(*http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return infoContractWire(404, "retry public404"), nil
			}
			return infoContractWire(200, `{}`), nil
		})
		client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			hooks++
			return nil
		}
		if got, err := v1.New(client).GetInfo(context.Background()); err != nil || got == nil || calls != 2 || hooks != 1 {
			t.Fatalf("info=%+v err=%v calls=%d hooks=%d", got, err, calls, hooks)
		}
		for _, status := range []int{201, 401, 403, 500} {
			client := infoContractClient(func(*http.Request) (*http.Response, error) { return infoContractWire(status, "rejected"), nil })
			got, err := v1.New(client).GetObjectSegmentSize(context.Background())
			var native gophercloud.ErrUnexpectedResponseCode
			if got != nil || !errors.As(err, &native) || native.Actual != status {
				t.Fatalf("status=%d segment=%+v err=%v native=%+v", status, got, err, native)
			}
		}
	})
}
