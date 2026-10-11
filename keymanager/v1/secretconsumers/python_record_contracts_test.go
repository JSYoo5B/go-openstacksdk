package secretconsumers_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretconsumers"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type consumerRecordTransport func(*http.Request) (*http.Response, error)

func (f consumerRecordTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
func consumerRecordWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"X-Receipt": []string{"actual"}}, Body: io.NopCloser(strings.NewReader(body))}
}
func consumerRecordClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("key-manager", "/catalog/v1/")
	client.ResourceBase = cloud.Server.URL + "/barbican/v1/"
	return client
}
func consumerRecordField(t *testing.T, value *resource.RawResource, key, want string) {
	t.Helper()
	if value == nil || string(value.Body[key]) != want {
		t.Fatalf("%s wanted %s: %#v", key, want, value)
	}
}

func TestPythonConsumerRecordCreateSeedWireAndLiteralParent(t *testing.T) {
	cloud := testcloud.New(t)
	client := consumerRecordClient(cloud)
	client.MoreHeaders = map[string]string{"X-Project-Id": "project"}
	requests := 0
	cloud.Provider.HTTPClient.Transport = consumerRecordTransport(func(req *http.Request) (*http.Response, error) {
		requests++
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}
		if req.Method != http.MethodPost || req.URL.EscapedPath() != "/barbican/v1/secrets/s%2F%25/consumers" || req.Header.Get("X-Project-Id") != "project" || req.Header.Get("X-Auth-Token") != "test-token" {
			t.Fatalf("wrong request: %+v", req)
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatal(err)
		}
		want := map[string]json.RawMessage{"id": json.RawMessage("false"), "service": json.RawMessage("null"), "resource_type": json.RawMessage("false"), "resource_id": json.RawMessage(`{"counter":9007199254740993}`)}
		if !reflect.DeepEqual(body, want) {
			t.Fatalf("body=%s", raw)
		}
		return consumerRecordWire(200, `{"name":"server-secret","secret_id":"wrong-parent","secret_ref":"https://foreign/elsewhere","status":"ACTIVE","service":0,"x":{"n":9007199254740993}}`), nil
	})
	opts := secretconsumers.WithCreateRecordAttributes(map[string]any{
		"id": false, "service": nil, "resource_type": false, "resource_id": map[string]any{"counter": json.Number("9007199254740993")},
		"secret_id": "replace", "status": func() {}, "UNKNOWN": make(chan int),
	})
	result, err := secretconsumers.New(client).CreateRecord(context.Background(), "s/%", opts)
	if err != nil || result.SecretID != "s/%" || result.StatusCode != 200 || requests != 1 {
		t.Fatal(result, err, requests)
	}
	consumerRecordField(t, result.Resource, "id", "false")
	consumerRecordField(t, result.Resource, "name", `"server-secret"`)
	consumerRecordField(t, result.Resource, "service", "0")
	consumerRecordField(t, result.Resource, "resource_type", "false")
	consumerRecordField(t, result.Resource, "resource_id", `{"counter":9007199254740993}`)
	consumerRecordField(t, result.Resource, "location", "null")
	if len(result.Resource.Body) != 6 || string(result.Wire.Body["status"]) != `"ACTIVE"` || result.Wire.Body["resource_type"] != nil {
		t.Fatal(result)
	}
	result.Resource.Body["service"][0] = '9'
	result.Envelope[0] = 'x'
	result.Header.Set("X-Receipt", "changed")
	if string(result.Wire.Body["service"]) != "0" || result.Wire.Header.Get("X-Receipt") != "actual" {
		t.Fatal("receipt aliases projected data")
	}
	if client.ResourceBase != cloud.Server.URL+"/barbican/v1/" || client.MoreHeaders["X-Project-Id"] != "project" {
		t.Fatal(client)
	}
}

func TestPythonConsumerRecordMutationStatusesAndNoBodyPolicy(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 302, 399} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := consumerRecordClient(cloud)
			calls := 0
			cloud.Provider.HTTPClient.Transport = consumerRecordTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				return consumerRecordWire(code, "not JSON"), nil
			})
			api := secretconsumers.New(client)
			created, err := api.CreateRecord(context.Background(), "s1", secretconsumers.WithCreateRecordAttribute("resource_type", false))
			if err != nil || created.StatusCode != code || created.Wire != nil || string(created.Envelope) != "not JSON" {
				t.Fatal(created, err)
			}
			consumerRecordField(t, created.Resource, "resource_type", "false")
			consumerRecordField(t, created.Resource, "service", "null")
			deleted, err := api.DeleteRecord(context.Background(), "s1")
			if err != nil || deleted.StatusCode != code || string(deleted.Envelope) != "not JSON" || calls != 2 {
				t.Fatal(deleted, err, calls)
			}
		})
	}
	for _, body := range []string{"null", "[]", "42"} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Provider.HTTPClient.Transport = consumerRecordTransport(func(*http.Request) (*http.Response, error) { return consumerRecordWire(200, body), nil })
			result, err := secretconsumers.New(consumerRecordClient(cloud)).CreateRecord(context.Background(), "s1")
			var accepted *resource.ResponseError
			if result == nil || result.Resource != nil || !errors.As(err, &accepted) || string(accepted.Body) != body || accepted.StatusCode != 200 {
				t.Fatal(result, err)
			}
			if _, err := secretconsumers.New(consumerRecordClient(cloud)).DeleteRecord(context.Background(), "s1"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPythonConsumerRecordOptionsSnapshotAndSourcePreflight(t *testing.T) {
	cloud := testcloud.New(t)
	client := consumerRecordClient(cloud)
	calls := 0
	cloud.Provider.HTTPClient.Transport = consumerRecordTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		body, _ := io.ReadAll(req.Body)
		if string(body) != `{"resource_id":{"owned":false},"resource_type":null,"service":"original"}` {
			t.Fatalf("%s", body)
		}
		return consumerRecordWire(200, `{}`), nil
	})
	input := map[string]any{"resource_id": map[string]any{"owned": false}, "resource_type": nil, "service": "original"}
	option := secretconsumers.WithCreateRecordAttributes(input)
	input["service"] = "changed"
	input["resource_id"].(map[string]any)["owned"] = true
	api := secretconsumers.New(client)
	for i := 0; i < 2; i++ {
		if _, err := api.CreateRecord(context.Background(), "s1", option); err != nil {
			t.Fatal(err)
		}
	}
	cases := []secretconsumers.CreateRecordOption{
		nil,
		secretconsumers.WithCreateRecordAttribute("service", make(chan int)),
		secretconsumers.WithCreateRecordHeader("Content-Type", "text/plain"),
		func(config *request.Config[secretconsumers.RecordMutationOpts]) error {
			config.Fields["secret_id"] = json.RawMessage(`"elsewhere"`)
			return nil
		},
		func(config *request.Config[secretconsumers.RecordMutationOpts]) error {
			config.Query.Set("offset", "1")
			return nil
		},
	}
	for _, option := range cases {
		if _, err := api.CreateRecord(context.Background(), "s1", option); err == nil {
			t.Fatal("invalid option accepted")
		}
	}
	// A callback can reset an unsupported carrier to nil. The next option
	// must still reach SDK capability validation instead of panicking on Set.
	_, nilQueryErr := api.CreateRecord(context.Background(), "s1",
		func(config *request.Config[secretconsumers.RecordMutationOpts]) error {
			config.Query = nil
			return nil
		}, request.WithQuery[secretconsumers.RecordMutationOpts]("offset", "1"))
	if !errors.Is(nilQueryErr, resource.ErrInvalidOption) {
		t.Fatal("nil raw Query escaped mutation preflight", nilQueryErr)
	}
	_, nilQueryErr = api.AllRecords(context.Background(), "s1",
		func(config *request.Config[secretconsumers.ListRecordOpts]) error {
			config.Query = nil
			return nil
		}, request.WithQuery[secretconsumers.ListRecordOpts]("marker", "other"))
	if !errors.Is(nilQueryErr, resource.ErrInvalidOption) {
		t.Fatal("nil raw Query escaped list preflight", nilQueryErr)
	}
	ownErr := errors.New("callback failure")
	_, err := api.CreateRecord(context.Background(), "s1", func(config *request.Config[secretconsumers.RecordMutationOpts]) error {
		client.ResourceBase = cloud.Server.URL + "/changed/"
		return ownErr
	})
	if !errors.Is(err, ownErr) || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	client.ResourceBase = cloud.Server.URL + "/barbican/v1/"
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(ownErr)
	if _, err := api.CreateRecord(ctx, "s1"); !errors.Is(err, ownErr) || !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	for _, id := range []string{"", ".", "..", "\n", " \t", "\xff"} {
		if _, err := api.CreateRecord(context.Background(), id); err == nil {
			t.Fatal(id)
		}
	}
	var nilAPI *secretconsumers.API
	if _, err := nilAPI.CreateRecord(context.Background(), "s1"); err == nil {
		t.Fatal("nil API accepted")
	}
	if _, err := api.CreateRecord(nil, "s1"); err == nil {
		t.Fatal("nil context accepted")
	}
	if calls != 2 {
		t.Fatal("preflight performed HTTP", calls)
	}
}

func TestPythonConsumerRecordDeleteMissingAndRetryPolicy(t *testing.T) {
	cloud := testcloud.New(t)
	client := consumerRecordClient(cloud)
	calls := 0
	cloud.Provider.HTTPClient.Transport = consumerRecordTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		body, _ := io.ReadAll(req.Body)
		if string(body) != `{"resource_id":0,"resource_type":null,"service":false}` {
			t.Fatalf("%s", body)
		}
		return consumerRecordWire(404, `{"error":"missing"}`), nil
	})
	option := secretconsumers.WithDeleteRecordAttributes(map[string]any{"service": false, "resource_type": nil, "resource_id": 0})
	api := secretconsumers.New(client)
	if value, err := api.DeleteRecord(context.Background(), "s1", option); value != nil || err != nil {
		t.Fatal(value, err)
	}
	if _, err := api.DeleteRecord(context.Background(), "s1", option, secretconsumers.WithDeleteRecordIgnoreMissing(false)); !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatal(err)
	}
	cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
		return original
	}
	if value, err := api.DeleteRecord(context.Background(), "s1", option); value != nil || err != nil {
		t.Fatal(value, err)
	}
	retryErr := errors.New("retry refused")
	cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return retryErr }
	if _, err := api.DeleteRecord(context.Background(), "s1", option); !errors.Is(err, retryErr) || !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatal(err)
	}
	if calls != 4 {
		t.Fatal(calls)
	}
}

func TestPythonConsumerRecordListBodyFiltersDefaultsRawCapsAndReuse(t *testing.T) {
	cloud := testcloud.New(t)
	client := consumerRecordClient(cloud)
	calls := 0
	cloud.Provider.HTTPClient.Transport = consumerRecordTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodGet || req.URL.RawQuery != "limit=2" || req.Header.Get("Accept") != "application/json" {
			t.Fatal(req.URL, req.Header)
		}
		return consumerRecordWire(200, `{"consumers":[{"service":true,"resource_type":{"inner":{"flag":1,"extra":"ignored"}},"secret_id":"wire-parent"},{"service":0,"resource_type":{}},{"service":1}],"next":"https://foreign/unsafe"}`), nil
	})
	query := map[string]any{"service": 1, "resource_type": map[string]any{"inner": map[string]any{"flag": true}}, "name": nil, "offset": make(chan int), "status": func() {}}
	option := secretconsumers.WithListRecordQuery(query)
	query["service"] = false
	query["resource_type"].(map[string]any)["inner"].(map[string]any)["flag"] = false
	client.MoreHeaders = map[string]string{"aCcEpT": "text/plain"}
	api := secretconsumers.New(client)
	seq := api.ListRecords(context.Background(), "s1", option, secretconsumers.WithListRecordMaxItems(2))
	if calls != 0 {
		t.Fatal("iterator is eager")
	}
	for repeat := 0; repeat < 2; repeat++ {
		count := 0
		for record, err := range seq {
			if err != nil {
				t.Fatal(err)
			}
			count++
			if record.SecretID != "s1" || string(record.Wire.Body["secret_id"]) != `"wire-parent"` || len(record.Resource.Body) != 6 {
				t.Fatal(record)
			}
			consumerRecordField(t, record.Resource, "name", "null")
			record.Resource.Body["service"] = json.RawMessage("false")
		}
		if count != 1 {
			t.Fatal(count)
		}
	}
	if calls != 2 || client.MoreHeaders["aCcEpT"] != "text/plain" {
		t.Fatal(calls, client.MoreHeaders)
	}
}

func TestPythonConsumerRecordListPagingFixedParentAndMarkerFallback(t *testing.T) {
	t.Run("advertised offset and singleton", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := consumerRecordClient(cloud)
		calls := 0
		cloud.Provider.HTTPClient.Transport = consumerRecordTransport(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				if req.URL.RawQuery != "" {
					t.Fatal(req.URL)
				}
				return consumerRecordWire(200, `{"consumers":{"service":null,"resource_id":9007199254740993},"next":"/v1/secrets/s1/consumers?offset=1&limit=10"}`), nil
			}
			if req.URL.Query().Get("offset") != "1" || req.URL.Query().Get("limit") != "10" {
				t.Fatal(req.URL)
			}
			return consumerRecordWire(200, `{"consumers":[],"next":"https://foreign/ignored"}`), nil
		})
		records, err := secretconsumers.New(client).AllRecords(context.Background(), "s1")
		if err != nil || len(records) != 1 || calls != 2 {
			t.Fatal(records, err, calls)
		}
		consumerRecordField(t, records[0].Resource, "resource_id", "9007199254740993")
	})
	t.Run("marker fallback uses last raw row", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := consumerRecordClient(cloud)
		calls := 0
		cloud.Provider.HTTPClient.Transport = consumerRecordTransport(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return consumerRecordWire(200, `{"consumers":[{"id":"last","service":false}]}`), nil
			}
			if req.URL.Query().Get("marker") != "last" || req.URL.Query().Get("limit") != "5" {
				t.Fatal(req.URL)
			}
			return consumerRecordWire(200, `{"consumers":[]}`), nil
		})
		records, err := secretconsumers.New(client).AllRecords(context.Background(), "s1", secretconsumers.WithListRecordQuery(map[string]any{"service": true, "limit": 5}))
		if err != nil || len(records) != 0 || calls != 2 {
			t.Fatal(records, err, calls)
		}
	})
	for _, link := range []string{"https://foreign/v1/secrets/s1/consumers?offset=1", "/barbican/v1/secrets/other/consumers?offset=1", "/barbican/v1/secrets/s1/consumers?offset=0", "/barbican/v1/secrets/s1/consumers?offset=1&service=evil"} {
		t.Run(link, func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			cloud.Provider.HTTPClient.Transport = consumerRecordTransport(func(*http.Request) (*http.Response, error) {
				calls++
				raw, _ := json.Marshal(link)
				return consumerRecordWire(200, `{"consumers":[{}],"next":`+string(raw)+`}`), nil
			})
			rows, err := secretconsumers.New(consumerRecordClient(cloud)).AllRecords(context.Background(), "s1")
			var accepted *resource.ResponseError
			if err == nil || rows != nil || !errors.As(err, &accepted) || calls != 1 || accepted.StatusCode != 200 {
				t.Fatal(rows, err, calls)
			}
		})
	}
	t.Run("missing fallback id stops without replay", func(t *testing.T) {
		cloud := testcloud.New(t)
		calls := 0
		cloud.Provider.HTTPClient.Transport = consumerRecordTransport(func(*http.Request) (*http.Response, error) {
			calls++
			return consumerRecordWire(200, `{"consumers":[{}]}`), nil
		})
		_, err := secretconsumers.New(consumerRecordClient(cloud)).AllRecords(context.Background(), "s1", secretconsumers.WithListRecordQuery(map[string]any{"limit": 1}))
		if !errors.Is(err, resource.ErrPaginationCycle) || calls != 1 {
			t.Fatal(err, calls)
		}
	})
}

type consumerRecordFaultBody struct {
	data              []byte
	readErr, closeErr error
	read, close       func()
	done              bool
}

func (b *consumerRecordFaultBody) Read(p []byte) (int, error) {
	if b.done {
		return 0, io.EOF
	}
	b.done = true
	if b.read != nil {
		b.read()
	}
	n := copy(p, b.data)
	return n, b.readErr
}
func (b *consumerRecordFaultBody) Close() error {
	if b.close != nil {
		b.close()
	}
	return b.closeErr
}

func TestPythonConsumerRecordPhysicalFaultsPreserveAllCauses(t *testing.T) {
	for _, operation := range []string{"create", "delete", "list", "rejected-delete"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := consumerRecordClient(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readErr, closeErr, cause := errors.New("physical read"), errors.New("physical close"), errors.New("custom cancel")
			calls := 0
			body := `{"service":"actual"}`
			code := 200
			if operation == "list" {
				body = `{"consumers":[{}]}`
			}
			if operation == "rejected-delete" {
				code = 404
			}
			cloud.Provider.HTTPClient.Transport = consumerRecordTransport(func(*http.Request) (*http.Response, error) {
				calls++
				response := consumerRecordWire(code, body)
				response.Body = &consumerRecordFaultBody{data: []byte(body), readErr: readErr, closeErr: closeErr,
					read:  func() { client.ResourceBase = cloud.Server.URL + "/changed/" },
					close: func() { client.ResourceBase = cloud.Server.URL + "/barbican/v1/"; cancel(cause) }}
				return response, nil
			})
			api := secretconsumers.New(client)
			var err error
			switch operation {
			case "create":
				result, prior := api.CreateRecord(ctx, "s1")
				err = prior
				if result == nil || result.Resource != nil || string(result.Envelope) != body {
					t.Fatal(result, err)
				}
			case "delete":
				result, prior := api.DeleteRecord(ctx, "s1")
				err = prior
				if result == nil || string(result.Envelope) != body {
					t.Fatal(result, err)
				}
			case "list":
				result, prior := api.AllRecords(ctx, "s1")
				err = prior
				if result != nil {
					t.Fatal(result)
				}
			case "rejected-delete":
				result, prior := api.DeleteRecord(ctx, "s1")
				err = prior
				if result != nil {
					t.Fatal(result)
				}
			}
			for _, expected := range []error{readErr, closeErr, cause, context.Canceled, resource.ErrInvalidOption} {
				if !errors.Is(err, expected) {
					t.Fatalf("missing %v: %v", expected, err)
				}
			}
			if operation != "rejected-delete" {
				var accepted *resource.ResponseError
				if !errors.As(err, &accepted) || accepted.StatusCode != code || string(accepted.Body) != body || accepted.Header.Get("X-Receipt") != "actual" {
					t.Fatal(err, accepted)
				}
			} else if !gophercloud.ResponseCodeIs(err, 404) {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatal("physical faults replayed", calls)
			}
		})
	}
}

func TestPythonConsumerRecordRetriesCannotChangeBodyOrListAccept(t *testing.T) {
	for _, operation := range []string{"create", "delete", "list"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := consumerRecordClient(cloud)
			calls := 0
			cloud.Provider.HTTPClient.Transport = consumerRecordTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				return consumerRecordWire(503, `{"error":"retry"}`), nil
			})
			cloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				if operation == "list" {
					opts.MoreHeaders["Accept"] = "text/plain"
				} else {
					opts.JSONBody = json.RawMessage(`{"service":"changed"}`)
				}
				return nil
			}
			api := secretconsumers.New(client)
			var err error
			switch operation {
			case "create":
				_, err = api.CreateRecord(context.Background(), "s1")
			case "delete":
				_, err = api.DeleteRecord(context.Background(), "s1")
			case "list":
				_, err = api.AllRecords(context.Background(), "s1")
			}
			if !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
				t.Fatal(err, calls)
			}
		})
	}
}
