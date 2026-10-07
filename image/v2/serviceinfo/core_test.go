package serviceinfo

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type infoTransport func(*http.Request) (*http.Response, error)

func (value infoTransport) RoundTrip(req *http.Request) (*http.Response, error) { return value(req) }

type infoReader func([]byte) (int, error)

func (value infoReader) Read(buffer []byte) (int, error) { return value(buffer) }

type infoBody struct {
	reader   io.Reader
	closeErr error
	closes   int
}

func (value *infoBody) Read(buffer []byte) (int, error) { return value.reader.Read(buffer) }
func (value *infoBody) Close() error                    { value.closes++; return value.closeErr }

func infoClient(transport infoTransport) *gophercloud.ServiceClient {
	provider := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: transport}}
	provider.UseTokenLock()
	provider.SetToken("before")
	return &gophercloud.ServiceClient{ProviderClient: provider, Type: "image", Endpoint: "https://example.test/", ResourceBase: "https://example.test/reverse/glance/v2/"}
}

func infoResponse(req *http.Request, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Request: req, Body: body, Header: http.Header{"X-Actual": {"owned"}, "Content-Type": {"application/json"}}}
}

func TestServiceInfoCoreDiscoveryRoutesAndRawModels(t *testing.T) {
	var paths []string
	client := infoClient(func(req *http.Request) (*http.Response, error) {
		paths = append(paths, req.URL.Path)
		if req.Method != http.MethodGet || req.URL.RawQuery != "" {
			t.Fatal("unexpected request", req.Method, req.URL)
		}
		body := `{"stores":[{"id":"safe data /?","description":"basic","default":"false","properties":{"precision":9007199254740993},"Default":{}}]}`
		switch req.URL.Path {
		case "/reverse/glance/v2/info/import":
			body = `{"import-methods":{"description":"available","type":"array","value":["glance-direct","future-method"],"constraint":9007199254740993},"Import-Methods":5}`
		case "/reverse/glance/v2/info/stores/detail":
			body = `{"stores":[{"id":"detail","type":"file","default":true,"read-only":"true","weight":0}]}`
		}
		return infoResponse(req, io.NopCloser(strings.NewReader(body))), nil
	})
	api := New(client)
	if api.RawClient() != client || (*API)(nil).RawClient() != nil {
		t.Fatal("raw client identity changed")
	}
	info, err := api.GetImportInfo(context.Background())
	if err != nil || info == nil || info.StatusCode != 200 || info.Header.Get("X-Actual") != "owned" || info.ImportMethods == nil || len(info.ImportMethods.Value) != 2 || string(info.ImportMethods.Body["constraint"]) != "9007199254740993" || string(info.Body["Import-Methods"]) != "5" {
		t.Fatalf("info=%+v err=%v", info, err)
	}
	stores, err := api.AllStores(context.Background())
	if err != nil || len(stores) != 1 || stores[0].ID != "safe data /?" || stores[0].IsDefault == nil || *stores[0].IsDefault || string(stores[0].Properties["precision"]) != "9007199254740993" || stores[0].StatusCode != 200 {
		t.Fatalf("stores=%+v err=%v", stores, err)
	}
	details, err := api.AllStores(context.Background(), WithListStoresDetails(true))
	if err != nil || len(details) != 1 || details[0].Type == nil || *details[0].Type != "file" || details[0].ReadOnly == nil || !*details[0].ReadOnly || details[0].Weight == nil || *details[0].Weight != 0 {
		t.Fatalf("details=%+v err=%v", details, err)
	}
	if strings.Join(paths, ",") != "/reverse/glance/v2/info/import,/reverse/glance/v2/info/stores,/reverse/glance/v2/info/stores/detail" {
		t.Fatal("routes changed", paths)
	}
}

func TestServiceInfoCoreCanonicalAndNullableDecoding(t *testing.T) {
	var store Store
	if err := json.Unmarshal([]byte(`{"id":null,"description":null,"default":null,"properties":{},"type":"","read-only":"tr\u0075e","weight":-9223372036854775808,"ID":5,"Default":[]}`), &store); err != nil || store.ID != "" || store.Description != "" || store.IsDefault != nil || store.Properties == nil || store.Type == nil || *store.Type != "" || store.ReadOnly == nil || !*store.ReadOnly || store.Weight == nil || *store.Weight != -9223372036854775808 || string(store.Body["ID"]) != "5" {
		t.Fatalf("store=%+v err=%v", store, err)
	}
	for _, body := range []string{
		`null`, `[]`, `{"id":1}`, `{"description":false}`, `{"default":"False"}`, `{"default":1}`, `{"read-only":{}}`, `{"properties":[]}`, `{"type":1}`, `{"weight":1.5}`, `{"weight":"1"}`, `{"weight":9223372036854775808}`,
	} {
		if err := json.Unmarshal([]byte(body), &Store{}); err == nil {
			t.Fatal("invalid canonical store accepted", body)
		}
	}
	for _, body := range []string{`{}`, `{"import-methods":null}`, `{"import_info":{"import-methods":{"value":["decoy"]}}}`} {
		var info ImportInfo
		if err := json.Unmarshal([]byte(body), &info); err != nil || info.ImportMethods != nil || info.Body == nil {
			t.Fatalf("info=%+v body=%s err=%v", info, body, err)
		}
	}
	for _, body := range []string{`{"import-methods":[]}`, `{"import-methods":{"value":[null]}}`, `{"import-methods":{"value":[1]}}`, `{"import-methods":{"value":{}}}`, `{"import-methods":{"description":5}}`} {
		if err := json.Unmarshal([]byte(body), &ImportInfo{}); err == nil {
			t.Fatal("invalid methods accepted", body)
		}
	}
	for _, test := range []struct {
		body    string
		wantNil bool
	}{
		{`{"import-methods":{}}`, true}, {`{"import-methods":{"value":null}}`, true}, {`{"import-methods":{"value":[]}}`, false},
	} {
		var info ImportInfo
		if err := json.Unmarshal([]byte(test.body), &info); err != nil || info.ImportMethods == nil || (info.ImportMethods.Value == nil) != test.wantNil {
			t.Fatalf("value nullable distinction lost: %+v err=%v", info, err)
		}
	}
}

func TestServiceInfoCorePagingUsesAdvertisedLinksAndStopsAtControls(t *testing.T) {
	var requests int
	client := infoClient(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path != "/reverse/glance/v2/info/stores/detail" || req.URL.Query().Get("limit") != "2" || req.URL.Query().Get("filter") != "a & b" {
			t.Fatal("paging route or query changed", req.URL)
		}
		body := `{"stores":[{"id":"raw /?% 中文"}],"next":"?marker=advertised"}`
		if requests == 2 {
			if req.URL.Query().Get("marker") != "advertised" {
				t.Fatal("caller-mutated ID replaced advertised cursor", req.URL)
			}
			body = `{"stores":[],"next":"https://foreign.test/ignored"}`
		}
		return infoResponse(req, io.NopCloser(strings.NewReader(body))), nil
	})
	for value, err := range New(client).ListStores(context.Background(), WithListStoresOptions(ListStoresOpts{Details: true, Limit: 2}), WithListStoresQuery("filter", "a & b")) {
		if err != nil {
			t.Fatal(err)
		}
		value.ID = "caller mutation"
		value.Body["id"] = json.RawMessage(`"other caller mutation"`)
	}
	if requests != 2 {
		t.Fatal("advertised continuation lost", requests)
	}
	for _, test := range []struct {
		option ListStoresOption
		body   string
	}{
		{WithListStoresMaxItems(1), `{"stores":[{"id":"one"},{"id":5}],"next":"https://foreign.test/unused"}`},
		{WithListStoresPaginated(false), `{"stores":[{"id":"one"}],"next":"https://foreign.test/unused"}`},
	} {
		requests = 0
		client = infoClient(func(req *http.Request) (*http.Response, error) {
			requests++
			if req.URL.Query().Has("limit") {
				t.Fatal("local control generated wire limit", req.URL)
			}
			return infoResponse(req, io.NopCloser(strings.NewReader(test.body))), nil
		})
		values, err := New(client).AllStores(context.Background(), test.option)
		if err != nil || len(values) != 1 || requests != 1 {
			t.Fatalf("cap/first-page control failed: values=%+v err=%v requests=%d", values, err, requests)
		}
	}
}

func TestServiceInfoCoreServerIgnoresPaginationQueries(t *testing.T) {
	for _, test := range []struct {
		name, path, limit, marker string
		options                   []ListStoresOption
	}{
		{"local cap three", "/info/stores", "", "", []ListStoresOption{WithListStoresMaxItems(3)}},
		{"local cap twenty detail", "/info/stores/detail", "", "", []ListStoresOption{WithListStoresDetails(true), WithListStoresMaxItems(20)}},
		{"explicit typed inputs", "/info/stores", "1", "start", []ListStoresOption{WithListStoresOptions(ListStoresOpts{Limit: 1, Marker: "start"})}},
		{"explicit raw inputs", "/info/stores", "1", "raw", []ListStoresOption{WithListStoresQuery("limit", "1"), WithListStoresQuery("marker", "raw")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var requests int
			client := infoClient(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.URL.Path != "/reverse/glance/v2"+test.path || req.URL.Query().Get("limit") != test.limit || req.URL.Query().Get("marker") != test.marker {
					t.Fatal("explicit query or route changed", req.URL)
				}
				// Actual Glance discovery handlers return the entire configured
				// list and ignore limit/marker. Empty IDs are passive row data.
				return infoResponse(req, io.NopCloser(strings.NewReader(`{"stores":[{"id":""},{"id":"two"}]}`))), nil
			})
			values, err := New(client).AllStores(context.Background(), test.options...)
			if err != nil || len(values) != 2 || values[0].ID != "" || values[1].ID != "two" || requests != 1 {
				t.Fatalf("invented repeated page: values=%+v err=%v requests=%d", values, err, requests)
			}
		})
	}
}

func TestServiceInfoCoreBodyAndModelFailuresKeepEvidence(t *testing.T) {
	readCause, closeCause, customCause := errors.New("read failure"), errors.New("close failure"), errors.New("caller cancellation")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	body := &infoBody{reader: infoReader(func(buffer []byte) (int, error) {
		cancel(customCause)
		return copy(buffer, "actual partial body"), readCause
	}), closeErr: closeCause}
	var requests, retries int
	client := infoClient(func(req *http.Request) (*http.Response, error) { requests++; return infoResponse(req, body), nil })
	client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		retries++
		return nil
	}
	value, err := New(client).GetImportInfo(ctx)
	var accepted *resource.ResponseError
	if value != nil || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != "actual partial body" || accepted.Header.Get("X-Actual") != "owned" || requests != 1 || retries != 0 || body.closes != 1 {
		t.Fatalf("value=%+v accepted=%+v err=%v requests=%d retries=%d closes=%d", value, accepted, err, requests, retries, body.closes)
	}
	for _, cause := range []error{readCause, closeCause, customCause, context.Canceled} {
		if !errors.Is(err, cause) {
			t.Fatal("body/context cause lost", cause, err)
		}
	}
	for _, raw := range [][]byte{[]byte(`{"stores":null}`), []byte(`{"stores":[{"id":"first"},{"default":"invalid"}]}`), append([]byte(`{"stores":[],"unknown":"`), 0xff, '"', '}')} {
		body = &infoBody{reader: strings.NewReader(string(raw))}
		client = infoClient(func(req *http.Request) (*http.Response, error) { return infoResponse(req, body), nil })
		values, err := New(client).AllStores(context.Background())
		accepted = nil
		if values != nil || !errors.As(err, &accepted) || string(accepted.Body) != string(raw) || accepted.StatusCode != 200 || body.closes != 1 {
			t.Fatalf("malformed accepted evidence lost values=%+v accepted=%+v err=%v closes=%d", values, accepted, err, body.closes)
		}
	}
}

func TestServiceInfoCoreCapturesSourceAndGuardsLaterPages(t *testing.T) {
	var requests, callbacks int
	client := infoClient(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path != "/reverse/glance/v2/info/stores" || req.Header.Get("X-Source") != "before" || req.Header.Get("X-Auth-Token") != "after" {
			t.Fatal("source snapshot or live auth lost", req.URL, req.Header)
		}
		return infoResponse(req, io.NopCloser(strings.NewReader(`{"stores":[{"id":"one"}],"next":"?marker=one"}`))), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "before"}
	api := New(client)
	option := func(*request.Config[ListStoresOpts]) error {
		callbacks++
		client.ResourceBase = "https://example.test/other/v2/"
		client.MoreHeaders["X-Source"] = "after"
		client.SetToken("after")
		return nil
	}
	iterator := api.ListStores(context.Background(), option)
	if requests != 0 || callbacks != 0 {
		t.Fatal("list construction was not lazy")
	}
	var terminal error
	for value, err := range iterator {
		if err != nil {
			terminal = err
			break
		}
		if value.ID != "one" {
			t.Fatal(value)
		}
		client.ProviderClient = &gophercloud.ProviderClient{}
	}
	if !errors.Is(terminal, resource.ErrInvalidOption) || requests != 1 || callbacks != 1 {
		t.Fatalf("later provider replacement not guarded terminal=%v requests=%d callbacks=%d", terminal, requests, callbacks)
	}
	callbacks = 0
	if _, err := api.GetImportInfo(nil, func(*request.Config[GetImportInfoOpts]) error { callbacks++; return nil }); !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 {
		t.Fatal("source/context preflight ran options", err, callbacks)
	}
}
