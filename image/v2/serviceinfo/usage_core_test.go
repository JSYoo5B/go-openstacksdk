package serviceinfo

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestUsageCoreCurrentProjectRouteAndEmptyUsage(t *testing.T) {
	for _, raw := range []string{
		`{"usage":{"image_size_total":{"limit":500,"usage":12},"image_stage_total":{"limit":300,"usage":0},"image_count_total":{"limit":10,"usage":2},"image_count_uploading":{"limit":5,"usage":1}},"unknown":9007199254740993}`,
		`{"usage":{}}`,
	} {
		var requests int
		body := &infoBody{reader: strings.NewReader(raw)}
		client := infoClient(func(req *http.Request) (*http.Response, error) {
			requests++
			if req.Method != http.MethodGet || req.URL.Path != "/reverse/glance/v2/info/usage" || req.URL.RawQuery != "" || req.Body != nil || req.Header.Get("X-Extra") != "owned" {
				t.Fatal("usage request changed", req.Method, req.URL, req.Header)
			}
			return infoResponse(req, body), nil
		})
		value, err := New(client).GetUsageInfo(context.Background(), WithGetUsageInfoHeader("X-Extra", "owned"))
		if err != nil || value == nil || value.Usage == nil || value.StatusCode != 200 || value.Header.Get("X-Actual") != "owned" || requests != 1 || body.closes != 1 {
			t.Fatalf("value=%+v err=%v requests=%d closes=%d", value, err, requests, body.closes)
		}
		if len(value.Usage) != 0 {
			entry := value.Usage["image_size_total"]
			if len(value.Usage) != 4 || entry == nil || entry.Limit == nil || *entry.Limit != 500 || entry.Usage == nil || *entry.Usage != 12 || string(value.Body["unknown"]) != "9007199254740993" {
				t.Fatal("current-project metrics or raw fields lost", value)
			}
		}
	}
}

func TestUsageCoreCanonicalPrecisionAndPresence(t *testing.T) {
	var value UsageInfo
	raw := []byte(`{"usage":{"future /?% 中文":{"limit":-9223372036854775808,"usage":9223372036854775807,"Limit":[],"unknown":9007199254740993},"zero":{"limit":0,"usage":0},"absent":{},"null":{"limit":null,"usage":null},"large":{"usage":9007199254740993},"decoys":{"Limit":5,"Usage":false}},"Usage":[],"created_at":5}`)
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	entry := value.Usage["future /?% 中文"]
	if entry == nil || entry.Limit == nil || *entry.Limit != -9223372036854775808 || entry.Usage == nil || *entry.Usage != 9223372036854775807 || string(entry.Body["Limit"]) != "[]" || string(entry.Body["unknown"]) != "9007199254740993" || string(value.Body["Usage"]) != "[]" || string(value.Body["created_at"]) != "5" {
		t.Fatal("canonical or precise fields lost", value)
	}
	if value.Usage["large"].Usage == nil || *value.Usage["large"].Usage != 9007199254740993 || value.Usage["zero"].Limit == nil || *value.Usage["zero"].Limit != 0 || value.Usage["zero"].Usage == nil || *value.Usage["zero"].Usage != 0 {
		t.Fatal("integer precision or explicit zero lost", value)
	}
	for _, name := range []string{"absent", "null", "decoys"} {
		if value.Usage[name].Limit != nil || value.Usage[name].Usage != nil || value.Usage[name].Body == nil {
			t.Fatal("missing, null or case-variant metric changed presence", name, value.Usage[name])
		}
	}
	rootRaw := string(value.Body["usage"])
	entry.Body["limit"][0] = '0'
	if string(value.Body["usage"]) != rootRaw {
		t.Fatal("nested raw fields alias root evidence", value.Body)
	}
	if *entry.Limit != -9223372036854775808 {
		t.Fatal("raw field mutation changed typed metric", entry)
	}
}

func TestUsageCoreAcceptedSchemaFailuresKeepEvidence(t *testing.T) {
	for _, raw := range [][]byte{
		[]byte(``), []byte(`null`), []byte(`[]`), []byte(`{`), []byte(`{}`), []byte(`{"Usage":{}}`), []byte(`{"quota":{"usage":{}}}`), []byte(`{"usage":null}`), []byte(`{"usage":[]}`), []byte(`{"usage":1}`),
		[]byte(`{"usage":{"entry":null}}`), []byte(`{"usage":{"entry":[]}}`), []byte(`{"usage":{"entry":true}}`), []byte(`{"usage":{"entry":{"limit":"1"}}}`), []byte(`{"usage":{"entry":{"usage":false}}}`), []byte(`{"usage":{"entry":{"limit":1.0}}}`), []byte(`{"usage":{"entry":{"usage":1e3}}}`), []byte(`{"usage":{"entry":{"usage":9223372036854775808}}}`), []byte(`{"usage":{"entry":{"limit":-9223372036854775809}}}`),
		append([]byte(`{"usage":{},"unknown":"`), 0xff, '"', '}'),
	} {
		body := &infoBody{reader: strings.NewReader(string(raw))}
		client := infoClient(func(req *http.Request) (*http.Response, error) { return infoResponse(req, body), nil })
		value, err := New(client).GetUsageInfo(context.Background())
		var accepted *resource.ResponseError
		if value != nil || !errors.As(err, &accepted) || accepted.StatusCode != 200 || accepted.Header.Get("X-Actual") != "owned" || string(accepted.Body) != string(raw) || body.closes != 1 {
			t.Fatalf("body=%q value=%+v accepted=%+v err=%v closes=%d", raw, value, accepted, err, body.closes)
		}
	}
}

func TestUsageCoreOptionsPreflightAndOwnership(t *testing.T) {
	custom := errors.New("option cause")
	for _, option := range []GetUsageInfoOption{
		nil, func(*request.Config[GetUsageInfoOpts]) error { return custom }, request.WithQuery[GetUsageInfoOpts]("project_id", "other"), request.WithField[GetUsageInfoOpts]("usage", true), request.WithArgument[GetUsageInfoOpts]("microversion", "2.8"),
		WithGetUsageInfoHeader("Authorization", "replacement"), WithGetUsageInfoHeader("Accept", "text/plain"), WithGetUsageInfoHeader("OpenStack-API-Version", "image 2.8"), WithGetUsageInfoHeader("Bad Name", "value"), WithGetUsageInfoHeader("X-Extra", "bad\x00value"), WithGetUsageInfoHeader("X-Extra", "bad\xffvalue"),
	} {
		var requests int
		client := infoClient(func(req *http.Request) (*http.Response, error) {
			requests++
			return infoResponse(req, http.NoBody), nil
		})
		value, err := New(client).GetUsageInfo(context.Background(), option)
		if value != nil || err == nil || requests != 0 || !errors.Is(err, custom) && !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("option reached HTTP: value=%+v err=%v requests=%d", value, err, requests)
		}
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(custom)
	for _, test := range []struct {
		ctx    context.Context
		client *gophercloud.ServiceClient
		cause  error
	}{
		{nil, infoClient(nil), resource.ErrInvalidOption}, {ctx, infoClient(nil), custom}, {context.Background(), nil, resource.ErrInvalidOption}, {context.Background(), &gophercloud.ServiceClient{}, resource.ErrInvalidOption},
	} {
		var callbacks int
		value, err := New(test.client).GetUsageInfo(test.ctx, func(*request.Config[GetUsageInfoOpts]) error { callbacks++; return nil })
		if value != nil || !errors.Is(err, test.cause) || callbacks != 0 {
			t.Fatalf("preflight called option: value=%+v err=%v callbacks=%d", value, err, callbacks)
		}
	}
	var retained *request.Config[GetUsageInfoOpts]
	var callbacks int
	headers, err := prepareGetUsageInfo([]GetUsageInfoOption{
		func(config *request.Config[GetUsageInfoOpts]) error {
			callbacks++
			retained = config
			config.Headers["X-Extra"] = "owned"
			return nil
		},
		func(*request.Config[GetUsageInfoOpts]) error {
			callbacks++
			retained.Headers["X-Extra"] = "changed"
			retained.Query.Set("project_id", "other")
			return nil
		},
	})
	if err != nil || callbacks != 2 || headers["X-Extra"] != "owned" {
		t.Fatal("later callback changed prepared ownership", headers, callbacks, err)
	}
	retained.Headers["X-Extra"] = "later"
	if headers["X-Extra"] != "owned" {
		t.Fatal("prepared headers alias retained callback", headers)
	}
	option := WithGetUsageInfoHeader("X-Reusable", "owned")
	var group sync.WaitGroup
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			for range 10 {
				headers, err := prepareGetUsageInfo([]GetUsageInfoOption{option})
				if err != nil || headers["X-Reusable"] != "owned" {
					t.Errorf("reused option changed: headers=%v err=%v", headers, err)
					return
				}
				headers["X-Reusable"] = "caller mutation"
			}
		}()
	}
	group.Wait()
}

func TestUsageCoreBodyErrorsAndStrictStatus(t *testing.T) {
	readCause, closeCause, customCause := errors.New("read cause"), errors.New("close cause"), errors.New("cancel cause")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	body := &infoBody{reader: infoReader(func(buffer []byte) (int, error) { cancel(customCause); return copy(buffer, `{"usage":`), readCause }), closeErr: closeCause}
	var requests, retries int
	client := infoClient(func(req *http.Request) (*http.Response, error) { requests++; return infoResponse(req, body), nil })
	client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		retries++
		return nil
	}
	value, err := New(client).GetUsageInfo(ctx)
	var accepted *resource.ResponseError
	if value != nil || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != `{"usage":` || accepted.Header.Get("X-Actual") != "owned" || requests != 1 || retries != 0 || body.closes != 1 {
		t.Fatalf("value=%+v accepted=%+v err=%v requests=%d retries=%d closes=%d", value, accepted, err, requests, retries, body.closes)
	}
	for _, cause := range []error{readCause, closeCause, customCause, context.Canceled} {
		if !errors.Is(err, cause) {
			t.Fatal("accepted cause lost", cause, err)
		}
	}
	for _, status := range []int{201, 202, 204, 206, 403, 404, 498} {
		body = &infoBody{reader: strings.NewReader(`{"usage":{}}`)}
		client = infoClient(func(req *http.Request) (*http.Response, error) {
			response := infoResponse(req, body)
			response.StatusCode = status
			return response, nil
		})
		value, err := New(client).GetUsageInfo(context.Background())
		var native gophercloud.ErrUnexpectedResponseCode
		accepted = nil
		if value != nil || !errors.As(err, &native) || native.Actual != status || len(native.Expected) != 1 || native.Expected[0] != 200 || errors.As(err, &accepted) || body.closes != 1 {
			t.Fatalf("status=%d value=%+v native=%+v err=%v closes=%d", status, value, native, err, body.closes)
		}
	}
}

func TestUsageCoreCapturedSourceAndNativeRetry(t *testing.T) {
	var requests, callbacks, retries int
	client := infoClient(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path != "/reverse/glance/v2/info/usage" || req.Header.Get("X-Source") != "before" || req.Header.Get("X-Auth-Token") != "after" {
			t.Fatal("captured route/headers or live auth changed", req.URL, req.Header)
		}
		response := infoResponse(req, io.NopCloser(strings.NewReader(`{"usage":{}}`)))
		if requests == 1 {
			response.StatusCode = 503
		}
		return response, nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "before"}
	client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		retries++
		return nil
	}
	value, err := New(client).GetUsageInfo(context.Background(), func(*request.Config[GetUsageInfoOpts]) error {
		callbacks++
		client.ResourceBase = "https://example.test/other/v2/"
		client.MoreHeaders["X-Source"] = "after"
		client.SetToken("after")
		return nil
	})
	if err != nil || value == nil || value.Usage == nil || requests != 2 || callbacks != 1 || retries != 1 {
		t.Fatalf("value=%+v err=%v requests=%d callbacks=%d retries=%d", value, err, requests, callbacks, retries)
	}
	for _, tamper := range []func(*gophercloud.ServiceClient){
		func(client *gophercloud.ServiceClient) { client.ProviderClient = &gophercloud.ProviderClient{} },
		func(client *gophercloud.ServiceClient) {
			client.Endpoint = "https://foreign.test/"
			client.ResourceBase = "https://foreign.test/v2/"
		},
	} {
		requests = 0
		client = infoClient(func(req *http.Request) (*http.Response, error) {
			requests++
			return infoResponse(req, http.NoBody), nil
		})
		value, err := New(client).GetUsageInfo(context.Background(), func(*request.Config[GetUsageInfoOpts]) error { tamper(client); return nil })
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || requests != 0 {
			t.Fatalf("source retarget reached HTTP: value=%+v err=%v requests=%d", value, err, requests)
		}
	}
}
