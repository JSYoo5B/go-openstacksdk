package serviceinfo_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	nativeimport "gophercloudsdk/image/v2/imageimport"
	"gophercloudsdk/image/v2/serviceinfo"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

const usagePrefix = "/reverse/usage/glance/v2/"

type usageTransport func(*http.Request) (*http.Response, error)

func (f usageTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	wire, err := f(r)
	if wire != nil && wire.Request == nil {
		wire.Request = r
	}
	return wire, err
}

type usageBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *usageBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type usageReader func([]byte) (int, error)

func (f usageReader) Read(p []byte) (int, error) { return f(p) }

func usageWire(code int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{
		"Content-Type": {"application/json"}, "X-Request-Id": {"actual-usage"},
	}, Body: body}
}

func usageClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("image", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + usagePrefix
	return client
}

func usageProof(t *testing.T, err error, raw string) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != raw || proof.Header.Get("X-Request-Id") != "actual-usage" {
		t.Fatalf("accepted usage evidence lost: %v %#v", err, proof)
	}
	return proof
}

func TestServiceInfoUsageDefaultRouteAndPassiveModels(t *testing.T) {
	for _, mode := range []string{"four resources", "empty usage", "dynamic resources"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := usageClient(cloud)
			client.MoreHeaders = map[string]string{"X-Source": "captured"}
			client.Microversion = "2.10"
			cloud.Provider.SetToken("latest")
			var calls atomic.Int32
			raw := `{"usage":{"image_size_total":{"limit":100,"usage":10},"image_stage_total":{"limit":50,"usage":1},"image_count_total":{"limit":12,"usage":2},"image_count_uploading":{"limit":3,"usage":0}},"next":"https://foreign.invalid/quota"}`
			if mode == "empty usage" {
				raw = `{"usage":{},"future":{"value":9223372036854775808}}`
			} else if mode == "dynamic resources" {
				raw = `{"usage":{"wire%literal /?#":{"limit":-1,"usage":9007199254740993,"unknown":{"large":9223372036854775808,"fraction":1.000000000000000000001}},"":{},"future-resource":{"usage":0}}}`
			}
			body := &usageBody{Reader: strings.NewReader(raw)}
			wire := usageWire(200, body)
			wire.Header.Set("Location", "https://foreign.invalid/not-a-quota-target")
			cloud.Provider.HTTPClient.Transport = usageTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Method != http.MethodGet || r.URL.Path != usagePrefix+"info/usage" || r.URL.RawQuery != "" || r.Body != nil || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Option") != "ordinary" || r.Header.Get("X-Auth-Token") != "latest" || r.Header.Get("OpenStack-API-Version") != "image 2.10" {
					t.Error(r.Method, r.URL, r.Header, r.Body)
				}
				return wire, nil
			})
			api := serviceinfo.New(client)
			value, err := api.GetUsageInfo(context.Background(), serviceinfo.WithGetUsageInfoHeader("X-Option", "ordinary"))
			if err != nil || value == nil || value.Usage == nil || value.StatusCode != 200 || value.Header.Get("X-Request-Id") != "actual-usage" || value.Header.Get("Location") == "" || string(value.Body["usage"]) == "" || api.RawClient() != client || calls.Load() != 1 || body.closes.Load() != 1 {
				t.Fatal(value, err, calls.Load(), body.closes.Load())
			}
			switch mode {
			case "four resources":
				if len(value.Usage) != 4 || *value.Usage["image_size_total"].Limit != 100 || *value.Usage["image_stage_total"].Usage != 1 || *value.Usage["image_count_total"].Limit != 12 || value.Usage["image_count_uploading"].Usage == nil || *value.Usage["image_count_uploading"].Usage != 0 || string(value.Body["next"]) != `"https://foreign.invalid/quota"` {
					t.Fatal(value)
				}
			case "empty usage":
				if len(value.Usage) != 0 || string(value.Body["future"]) != `{"value":9223372036854775808}` {
					t.Fatal(value)
				}
			case "dynamic resources":
				entry := value.Usage["wire%literal /?#"]
				if len(value.Usage) != 3 || entry == nil || *entry.Limit != -1 || *entry.Usage != 9007199254740993 || string(entry.Body["unknown"]) != `{"large":9223372036854775808,"fraction":1.000000000000000000001}` || value.Usage[""] == nil || value.Usage[""].Limit != nil || value.Usage["future-resource"].Usage == nil || *value.Usage["future-resource"].Usage != 0 {
					t.Fatal(value)
				}
				*entry.Usage = 17
				value.Body["usage"][0] = '['
				if string(entry.Body["usage"]) != "9007199254740993" || *value.Usage["future-resource"].Usage != 0 {
					t.Fatal("caller changes aliased nested evidence", value)
				}
			}
			wire.Header.Set("X-Request-Id", "wire-later")
			if value.Header.Get("X-Request-Id") != "actual-usage" || client.MoreHeaders["X-Source"] != "captured" {
				t.Fatal("wire metadata or original client changed", value, client)
			}
		})
	}
}

func TestServiceInfoUsageCanonicalPresenceAndSchemaFailures(t *testing.T) {
	t.Run("canonical presence precision and case decoys", func(t *testing.T) {
		cloud := testcloud.New(t)
		raw := `{"usage":{"missing":{},"null":{"limit":null,"usage":null},"zero":{"limit":0,"usage":0,"LIMIT":"decoy","Usage":true},"bounds":{"limit":-9223372036854775808,"usage":9223372036854775807},"precise":{"usage":9007199254740993}},"Usage":12,"quota":{"usage":false}}`
		cloud.Provider.HTTPClient.Transport = usageTransport(func(*http.Request) (*http.Response, error) {
			return usageWire(200, io.NopCloser(strings.NewReader(raw))), nil
		})
		value, err := serviceinfo.New(usageClient(cloud)).GetUsageInfo(context.Background())
		if err != nil || value == nil || len(value.Usage) != 5 {
			t.Fatal(value, err)
		}
		missing, null, zero, bounds := value.Usage["missing"], value.Usage["null"], value.Usage["zero"], value.Usage["bounds"]
		if missing.Limit != nil || missing.Usage != nil || null.Limit != nil || null.Usage != nil || string(null.Body["limit"]) != "null" || zero.Limit == nil || *zero.Limit != 0 || zero.Usage == nil || *zero.Usage != 0 || string(zero.Body["LIMIT"]) != `"decoy"` || string(zero.Body["Usage"]) != "true" || *bounds.Limit != math.MinInt64 || *bounds.Usage != math.MaxInt64 || *value.Usage["precise"].Usage != 9007199254740993 || string(value.Body["Usage"]) != "12" || string(value.Body["quota"]) != `{"usage":false}` {
			t.Fatal("canonical values or presence changed", value)
		}
		if _, present := missing.Body["limit"]; present {
			t.Fatal("missing metric was synthesized")
		}
	})
	invalid := []string{"", "null", "[]", `"root"`, `{}`, `{"Usage":{}}`, `{"usage":null}`, `{"usage":[]}`, `{"usage":true}`, `{"quota":{"usage":{}}}`, `{"usage":{"fixed":null}}`, `{"usage":{"fixed":[]}}`, `{"usage":{"fixed":true}}`, `{"usage":{"fixed":"entry"}}`, `{"usage":{"fixed":1}}`, `{"usage":`}
	for _, field := range []string{"limit", "usage"} {
		for _, metric := range []string{`"1"`, "true", "{}", "[]", "1.0", "1e0", "1.5", "9223372036854775808", "-9223372036854775809"} {
			invalid = append(invalid, fmt.Sprintf(`{"usage":{"fixed":{%q:%s}}}`, field, metric))
		}
	}
	for _, pattern := range []string{`{"usage":{},"unknown":"%s"}`, `{"usage":{"%s":{}}}`, `{"usage":{"fixed":{"unknown":"%s"}}}`} {
		invalid = append(invalid, fmt.Sprintf(pattern, string([]byte{0xff})))
	}
	for index, raw := range invalid {
		t.Run(fmt.Sprintf("invalid schema %d", index), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			body := &usageBody{Reader: strings.NewReader(raw)}
			cloud.Provider.HTTPClient.Transport = usageTransport(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return usageWire(200, body), nil
			})
			value, err := serviceinfo.New(usageClient(cloud)).GetUsageInfo(context.Background())
			if value != nil || err == nil || calls.Load() != 1 || body.closes.Load() != 1 {
				t.Fatal(value, err, calls.Load(), body.closes.Load())
			}
			usageProof(t, err, raw)
		})
	}
}

func TestServiceInfoUsageCompletePreflightAndPreparedOptions(t *testing.T) {
	for _, mode := range []string{"nil context", "canceled", "nil API", "nil client", "nil provider", "wrong type", "endpoint query", "foreign base", "source header", "source microversion"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := usageClient(cloud)
			api, ctx := serviceinfo.New(client), context.Background()
			cause := errors.New("usage cancellation")
			want := error(resource.ErrInvalidOption)
			switch mode {
			case "nil context":
				ctx = nil
			case "canceled":
				canceled, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx, want = canceled, context.Canceled
			case "nil API":
				api = nil
			case "nil client":
				api = serviceinfo.New(nil)
			case "nil provider":
				client.ProviderClient = nil
			case "wrong type":
				client.Type, want = "compute", resource.ErrUnsupported
			case "endpoint query":
				client.Endpoint += "?project_id=other"
			case "foreign base":
				client.ResourceBase = "https://foreign.invalid/v2/"
			case "source header":
				client.MoreHeaders = map[string]string{"X-Auth-Token": "foreign"}
			case "source microversion":
				client.Microversion = "2.10\ninvalid"
			}
			var calls, callbacks atomic.Int32
			cloud.Provider.HTTPClient.Transport = usageTransport(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return nil, errors.New("preflight reached HTTP")
			})
			value, err := api.GetUsageInfo(ctx, func(*request.Config[serviceinfo.GetUsageInfoOpts]) error { callbacks.Add(1); return nil })
			if value != nil || !errors.Is(err, want) || mode == "canceled" && !errors.Is(err, cause) || calls.Load() != 0 || callbacks.Load() != 0 {
				t.Fatal(value, err, calls.Load(), callbacks.Load())
			}
		})
	}
	for _, mode := range []string{"nil option", "callback error", "query", "JSON field", "argument", "microversion", "owned header", "invalid header"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = usageTransport(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return nil, errors.New("option reached HTTP")
			})
			cause := errors.New("usage callback")
			want := error(resource.ErrInvalidOption)
			option := serviceinfo.GetUsageInfoOption(func(c *request.Config[serviceinfo.GetUsageInfoOpts]) error {
				switch mode {
				case "callback error":
					return cause
				case "query":
					c.Query.Set("project_id", "other")
				case "JSON field":
					c.Fields["usage"] = json.RawMessage(`{}`)
				case "argument":
					c.Arguments["project_id"] = "other"
				case "microversion":
					c.Headers["OpenStack-API-Version"] = "image 2.10"
				case "owned header":
					c.Headers["Accept"] = "application/json"
				case "invalid header":
					c.Headers["X-Invalid"] = "newline\n"
				}
				return nil
			})
			if mode == "nil option" {
				option = nil
			} else if mode == "callback error" {
				want = cause
			}
			value, err := serviceinfo.New(usageClient(cloud)).GetUsageInfo(context.Background(), option)
			if value != nil || !errors.Is(err, want) || calls.Load() != 0 {
				t.Fatal(value, err, calls.Load())
			}
		})
	}
	t.Run("owned callback slice and retained config", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls, callbacks atomic.Int32
		var retained *request.Config[serviceinfo.GetUsageInfoOpts]
		var options []serviceinfo.GetUsageInfoOption
		options = []serviceinfo.GetUsageInfoOption{func(c *request.Config[serviceinfo.GetUsageInfoOpts]) error {
			callbacks.Add(1)
			c.Headers["X-Option"] = "prepared"
			retained = c
			options[1] = nil
			return nil
		}, func(c *request.Config[serviceinfo.GetUsageInfoOpts]) error {
			callbacks.Add(1)
			retained.Headers["X-Option"] = "changed"
			if c.Headers["X-Option"] != "prepared" {
				t.Error("retained callback config changed next callback")
			}
			return nil
		}}
		cloud.Provider.HTTPClient.Transport = usageTransport(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			retained.Headers["X-Option"] = "late"
			if r.Header.Get("X-Option") != "prepared" {
				t.Error(r.Header)
			}
			return usageWire(200, io.NopCloser(strings.NewReader(`{"usage":{}}`))), nil
		})
		value, err := serviceinfo.New(usageClient(cloud)).GetUsageInfo(context.Background(), options...)
		if err != nil || value == nil || calls.Load() != 1 || callbacks.Load() != 2 {
			t.Fatal(value, err, calls.Load(), callbacks.Load())
		}
	})
	t.Run("ordinary header options reusable in parallel", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = usageTransport(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.URL.Path != usagePrefix+"info/usage" || r.URL.RawQuery != "" || r.Body != nil || r.Header.Get("X-Option") != "reusable" {
				t.Error(r.URL, r.Header, r.Body)
			}
			return usageWire(200, io.NopCloser(strings.NewReader(`{"usage":{}}`))), nil
		})
		api := serviceinfo.New(usageClient(cloud))
		option := serviceinfo.WithGetUsageInfoHeader("X-Option", "reusable")
		var workers sync.WaitGroup
		for i := 0; i < 4; i++ {
			workers.Add(1)
			go func() {
				defer workers.Done()
				value, err := api.GetUsageInfo(context.Background(), option)
				if err != nil || value == nil || value.Usage == nil {
					t.Error(value, err)
				}
			}()
		}
		workers.Wait()
		if calls.Load() != 4 {
			t.Fatal(calls.Load())
		}
	})
}

func TestServiceInfoUsageAcceptedFailuresAndStrictStatuses(t *testing.T) {
	for _, mode := range []string{"read", "close", "read close", "read close cancel", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("usage read"), errors.New("usage close"), errors.New("usage context cause")
			raw := `{"usage":{"fixed":{"limit":3,"usage":1}}}`
			body := &usageBody{Reader: strings.NewReader(raw)}
			if strings.Contains(mode, "read") {
				raw = raw[:18]
				body.Reader = usageReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
			}
			if strings.Contains(mode, "close") {
				body.closeErr = closeCause
			}
			wire := usageWire(200, body)
			body.onClose = func() {
				wire.Header.Set("X-Request-Id", "after-close")
				wire.StatusCode = 299
				if strings.Contains(mode, "cancel") {
					cancel(cancelCause)
				}
			}
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries.Add(1)
				return errors.New("accepted replay")
			}
			cloud.Provider.HTTPClient.Transport = usageTransport(func(*http.Request) (*http.Response, error) { calls.Add(1); return wire, nil })
			value, err := serviceinfo.New(usageClient(cloud)).GetUsageInfo(ctx)
			if value != nil || strings.Contains(mode, "read") && !errors.Is(err, readCause) || strings.Contains(mode, "close") && !errors.Is(err, closeCause) || strings.Contains(mode, "cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) || calls.Load() != 1 || retries.Load() != 0 || body.closes.Load() != 1 {
				t.Fatal(value, err, calls.Load(), retries.Load(), body.closes.Load())
			}
			usageProof(t, err, raw)
		})
	}
	for _, code := range []int{201, 202, 204, 206, 300, 400, 403, 404, 429, 498, 500} {
		t.Run(fmt.Sprintf("reject%d", code), func(t *testing.T) {
			cloud := testcloud.New(t)
			body := &usageBody{Reader: strings.NewReader("native usage rejection")}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = usageTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Method != http.MethodGet || r.URL.Path != usagePrefix+"info/usage" || r.URL.RawQuery != "" {
					t.Error("usage rejection invoked fallback", r.Method, r.URL)
				}
				return usageWire(code, body), nil
			})
			value, err := serviceinfo.New(usageClient(cloud)).GetUsageInfo(context.Background())
			var native gophercloud.ErrUnexpectedResponseCode
			var accepted *resource.ResponseError
			if value != nil || !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || native.Method != http.MethodGet || native.URL != cloud.Server.URL+usagePrefix+"info/usage" || string(native.Body) != "native usage rejection" || native.ResponseHeader.Get("X-Request-Id") != "actual-usage" || errors.As(err, &accepted) || calls.Load() != 1 || body.closes.Load() != 1 {
				t.Fatal(value, err, native, calls.Load(), body.closes.Load())
			}
		})
	}
	t.Run("transport nested404 stays terminal", func(t *testing.T) {
		cloud := testcloud.New(t)
		cause := errors.Join(errors.New("usage transport"), gophercloud.ErrUnexpectedResponseCode{Method: "GET", URL: "https://foreign.invalid/decoy", Actual: 404})
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = usageTransport(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, cause })
		value, err := serviceinfo.New(usageClient(cloud)).GetUsageInfo(context.Background())
		if value != nil || !errors.Is(err, cause) || calls.Load() != 1 {
			t.Fatal(value, err, calls.Load())
		}
	})
	t.Run("prior discovery and native Get stay callable", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = usageTransport(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			raw := `{"usage":{}}`
			switch r.URL.Path {
			case usagePrefix + "info/import":
				raw = `{"import-methods":{"description":"native","type":"array","value":["web-download"]}}`
			case usagePrefix + "info/stores":
				raw = `{"stores":[]}`
			case usagePrefix + "info/usage":
			default:
				t.Error(r.URL)
			}
			return usageWire(200, io.NopCloser(strings.NewReader(raw))), nil
		})
		client := usageClient(cloud)
		api := serviceinfo.New(client)
		usage, usageErr := api.GetUsageInfo(context.Background())
		imported, importErr := api.GetImportInfo(context.Background())
		stores, storeErr := api.AllStores(context.Background())
		native, nativeErr := nativeimport.New(client).Get(context.Background())
		if usageErr != nil || usage == nil || importErr != nil || imported == nil || imported.ImportMethods.Description != "native" || storeErr != nil || stores == nil || len(stores) != 0 || nativeErr != nil || native == nil || native.ImportMethods.Description != "native" || calls.Load() != 4 {
			t.Fatal(usage, usageErr, imported, importErr, stores, storeErr, native, nativeErr, calls.Load())
		}
	})
}

func TestServiceInfoUsageProviderSourceAndRetryBoundaries(t *testing.T) {
	for _, change := range []string{"valid prefix headers token", "provider", "type", "foreign base"} {
		t.Run("callback changes "+change, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := usageClient(cloud)
			client.MoreHeaders = map[string]string{"X-Source": "captured"}
			var calls atomic.Int32
			cloud.Provider.HTTPClient.Transport = usageTransport(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.URL.Path != usagePrefix+"info/usage" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != "after-callback" {
					t.Error("callback changed captured scope", r.URL, r.Header)
				}
				return usageWire(200, io.NopCloser(strings.NewReader(`{"usage":{}}`))), nil
			})
			value, err := serviceinfo.New(client).GetUsageInfo(context.Background(), func(*request.Config[serviceinfo.GetUsageInfoOpts]) error {
				switch change {
				case "valid prefix headers token":
					client.ResourceBase = cloud.Server.URL + "/later/v2/"
					client.MoreHeaders = map[string]string{"X-Source": "later"}
				case "provider":
					client.ProviderClient = &gophercloud.ProviderClient{HTTPClient: cloud.Provider.HTTPClient}
				case "type":
					client.Type = "compute"
				case "foreign base":
					client.ResourceBase = "https://foreign.invalid/v2/"
				}
				cloud.Provider.SetToken("after-callback")
				return nil
			})
			if change == "valid prefix headers token" {
				if err != nil || value == nil || calls.Load() != 1 {
					t.Fatal(value, err, calls.Load())
				}
			} else {
				want := error(resource.ErrInvalidOption)
				if change == "type" {
					want = resource.ErrUnsupported
				}
				if value != nil || !errors.Is(err, want) || calls.Load() != 0 {
					t.Fatal(value, err, calls.Load())
				}
			}
		})
	}
	t.Run("native prebody hooks preserve route liveauth and original provider", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls, reauth, backoff, retry atomic.Int32
		var bodies []*usageBody
		cloud.Provider.SetToken("initial")
		cloud.Provider.ReauthFunc = func(context.Context) error { reauth.Add(1); cloud.Provider.SetToken("reauth"); return nil }
		cloud.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
			backoff.Add(1)
			cloud.Provider.SetToken("backoff")
			return nil
		}
		cloud.Provider.RetryFunc = func(_ context.Context, method, endpoint string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			retry.Add(1)
			if method != http.MethodGet || endpoint != cloud.Server.URL+usagePrefix+"info/usage" || !gophercloud.ResponseCodeIs(err, 503) {
				return err
			}
			cloud.Provider.SetToken("retry")
			return nil
		}
		originalRetry := reflect.ValueOf(cloud.Provider.RetryFunc).Pointer()
		cloud.Provider.HTTPClient.Transport = usageTransport(func(r *http.Request) (*http.Response, error) {
			n := int(calls.Add(1)) - 1
			codes, tokens := []int{401, 429, 503, 200}, []string{"initial", "reauth", "backoff", "retry"}
			if n >= len(codes) {
				return nil, errors.New("unexpected usage replay")
			}
			if r.Method != http.MethodGet || r.URL.Path != usagePrefix+"info/usage" || r.URL.RawQuery != "" || r.Body != nil || r.Header.Get("X-Auth-Token") != tokens[n] {
				t.Error(r.Method, r.URL, r.Header, r.Body)
			}
			body := &usageBody{Reader: strings.NewReader(`{"usage":{}}`)}
			bodies = append(bodies, body)
			return usageWire(codes[n], body), nil
		})
		value, err := serviceinfo.New(usageClient(cloud)).GetUsageInfo(context.Background())
		if err != nil || value == nil || calls.Load() != 4 || reauth.Load() != 1 || backoff.Load() != 1 || retry.Load() != 1 || reflect.ValueOf(cloud.Provider.RetryFunc).Pointer() != originalRetry {
			t.Fatal(value, err, calls.Load(), reauth.Load(), backoff.Load(), retry.Load())
		}
		for _, body := range bodies {
			if body.closes.Load() != 1 {
				t.Fatal(body.closes.Load())
			}
		}
	})
	for _, change := range []string{"KeepResponseBody", "JSONResponse", "RawBody", "JSONBody", "OkCodes"} {
		t.Run("retry guard "+change, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, callbacks, borrowedReads atomic.Int32
			borrowed := &usageBody{Reader: usageReader(func([]byte) (int, error) { borrowedReads.Add(1); return 0, io.EOF })}
			readCause, closeCause := errors.New("unexpected status read"), errors.New("unexpected status close")
			var bodies []*usageBody
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				callbacks.Add(1)
				switch change {
				case "KeepResponseBody":
					opts.KeepResponseBody = false
				case "JSONResponse":
					opts.JSONResponse = new(any)
				case "RawBody":
					opts.RawBody = borrowed
				case "JSONBody":
					opts.JSONBody = json.RawMessage("null")
				case "OkCodes":
					opts.OkCodes = []int{200, 202}
				}
				return nil
			}
			cloud.Provider.HTTPClient.Transport = usageTransport(func(r *http.Request) (*http.Response, error) {
				if r.Body != nil || r.URL.Path != usagePrefix+"info/usage" || r.URL.RawQuery != "" {
					t.Error(r.Body, r.URL)
				}
				body := &usageBody{Reader: strings.NewReader("original503")}
				code := 503
				if calls.Add(1) == 2 {
					code = 202
					body.Reader = usageReader(func(p []byte) (int, error) { return copy(p, "actual202"), readCause })
					body.closeErr = closeCause
				}
				bodies = append(bodies, body)
				return usageWire(code, body), nil
			})
			value, err := serviceinfo.New(usageClient(cloud)).GetUsageInfo(context.Background())
			wantCode, wantCalls, wantBody := 503, int32(1), "original503"
			if change == "OkCodes" {
				wantCode, wantCalls, wantBody = 202, 2, "actual202"
				if !errors.Is(err, readCause) || !errors.Is(err, closeCause) {
					t.Fatal("unexpected accepted status lost read/Close causes", err)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("bodyless ownership guard was bypassed", err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if value != nil || !errors.As(err, &native) || native.Actual != wantCode || string(native.Body) != wantBody || !reflect.DeepEqual(native.Expected, []int{200}) || native.ResponseHeader.Get("X-Request-Id") != "actual-usage" || calls.Load() != wantCalls || callbacks.Load() != 1 || borrowedReads.Load() != 0 || borrowed.closes.Load() != 0 {
				t.Fatal(value, err, native, calls.Load(), callbacks.Load(), borrowedReads.Load(), borrowed.closes.Load())
			}
			for _, body := range bodies {
				if body.closes.Load() != 1 {
					t.Fatal(body.closes.Load())
				}
			}
		})
	}
	t.Run("native reauth failure fields remain inspectable", func(t *testing.T) {
		cloud := testcloud.New(t)
		cause := errors.New("usage reauth failed")
		var calls, reauthCalls atomic.Int32
		cloud.Provider.ReauthFunc = func(context.Context) error { reauthCalls.Add(1); return cause }
		body := &usageBody{Reader: strings.NewReader("unauthorized")}
		cloud.Provider.HTTPClient.Transport = usageTransport(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return usageWire(401, body), nil
		})
		value, err := serviceinfo.New(usageClient(cloud)).GetUsageInfo(context.Background())
		var reauth *gophercloud.ErrUnableToReauthenticate
		if value != nil || !errors.As(err, &reauth) || !errors.Is(reauth.ErrReauth, cause) || !gophercloud.ResponseCodeIs(reauth.ErrOriginal, 401) || calls.Load() != 1 || reauthCalls.Load() != 1 || body.closes.Load() != 1 {
			t.Fatal(value, err, reauth, calls.Load(), reauthCalls.Load(), body.closes.Load())
		}
	})
	for _, code := range []int{302, 307, 308} {
		t.Run(fmt.Sprintf("redirect%d", code), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, policies atomic.Int32
			body := &usageBody{Reader: strings.NewReader("redirect evidence")}
			cloud.Provider.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { policies.Add(1); return nil }
			originalPolicy := reflect.ValueOf(cloud.Provider.HTTPClient.CheckRedirect).Pointer()
			cloud.Provider.HTTPClient.Transport = usageTransport(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				wire := usageWire(code, body)
				wire.Header.Set("Location", "https://foreign.invalid/usage?project_id=other")
				return wire, nil
			})
			value, err := serviceinfo.New(usageClient(cloud)).GetUsageInfo(context.Background())
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 || policies.Load() != 1 || body.closes.Load() != 1 || reflect.ValueOf(cloud.Provider.HTTPClient.CheckRedirect).Pointer() != originalPolicy {
				t.Fatal(value, err, calls.Load(), policies.Load(), body.closes.Load())
			}
		})
	}
}
