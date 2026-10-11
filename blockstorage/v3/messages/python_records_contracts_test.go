package messages_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type messageCall struct {
	method, path, escaped, query, body string
	headers                            http.Header
}
type messageTransport func(*http.Request) (*http.Response, error)

func (f messageTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	v, err := f(r)
	if v != nil && v.Request == nil {
		v.Request = r
	}
	return v, err
}
func messageWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"actual"}}, Body: io.NopCloser(strings.NewReader(body))}
}

const messageBase = "/cinder/v3/project/messages"

func messageAPI(t *testing.T, version string, reply func(*http.Request) *http.Response) (*messages.API, *[]messageCall) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := &[]messageCall{}
	cloud.Provider.HTTPClient.Transport = messageTransport(func(r *http.Request) (*http.Response, error) {
		body := ""
		if r.Body != nil {
			raw, _ := io.ReadAll(r.Body)
			body = string(raw)
		}
		*calls = append(*calls, messageCall{r.Method, r.URL.Path, r.URL.EscapedPath(), r.URL.RawQuery, body, r.Header.Clone()})
		return reply(r), nil
	})
	client := cloud.Client("volumev3", "/cinder/v3/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	client.Microversion = version
	return messages.New(client), calls
}
func messageField(t *testing.T, v *messages.MessageRecord, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		t.Fatal(key, want, v)
	}
}
func TestPythonCinderMessageGetSourceStockEnvelopesAndStringDescriptors(t *testing.T) {
	row := `{"id":null,"name":false,"created_at":false,"event_id":9007199254740993,"guaranteed_until":["a",null],"message_level":"ERROR","request_id":{"k":true},"resource_type":[],"resource_uuid":null,"user_message":"literal","unknown":9007199254740993}`
	for _, wrapper := range []string{"messages", "message", "flat"} {
		t.Run(wrapper, func(t *testing.T) {
			body := row
			if wrapper != "flat" {
				body = `{"` + wrapper + `":` + row + `}`
			}
			api, calls := messageAPI(t, "3.80", func(*http.Request) *http.Response { return messageWire(200, body) })
			v, err := api.GetRecord(context.Background(), "selected")
			if err != nil || len(*calls) != 1 || (*calls)[0].path != messageBase+"/selected" || (*calls)[0].query != "" || (*calls)[0].body != "" || (*calls)[0].headers.Get("OpenStack-API-Version") != "volume 3.80" {
				t.Fatal(v, err, *calls)
			}
			for key, want := range map[string]string{"id": `null`, "name": `false`, "created_at": `"False"`, "event_id": `"9007199254740993"`, "guaranteed_until": `"['a', None]"`, "request_id": `"{'k': True}"`, "resource_type": `"[]"`, "resource_uuid": `null`, "location": `null`} {
				messageField(t, v, key, want)
			}
			if len(v.Resource.Body) != 11 || v.Resource.Body["unknown"] != nil || string(v.Wire.Body["unknown"]) != "9007199254740993" || v.StatusCode != 200 || string(v.Envelope) != body {
				t.Fatal(v)
			}
			v.Resource.Body["user_message"][1] = 'X'
			v.Header.Set("X-Request-Id", "changed")
			if string(v.Wire.Body["user_message"]) != `"literal"` || v.Wire.Header.Get("X-Request-Id") != "actual" || v.Resource.Header.Get("X-Request-Id") != "actual" {
				t.Fatal("owned proof changed", v)
			}
		})
	}
}
func TestPythonCinderMessageGetStrictSourceEnvelopeAndSeedShapes(t *testing.T) {
	for _, item := range []struct {
		body          string
		valid, strict bool
	}{{`{"message":{"id":"wire","user_message":"text"}}`, true, true}, {`{"messages":{}}`, true, false}, {"", true, false}, {"{bad", true, false}, {`null`, false, false}, {`[]`, false, false}, {`{"messages":null,"message":{"id":"wire"}}`, false, false}, {string([]byte{'{', '"', 'k', '"', ':', '"', 0xff, '"', '}'}), false, false}} {
		api, calls := messageAPI(t, "3.3", func(*http.Request) *http.Response { return messageWire(200, item.body) })
		v, err := api.GetRecord(context.Background(), "seed", messages.WithRecordGetSourceEnvelope(item.strict))
		if v == nil || (err == nil) != item.valid || len(*calls) != 1 || v.StatusCode != 200 || string(v.Envelope) != item.body {
			t.Fatal(item, v, err, *calls)
		}
		if item.valid {
			messageField(t, v, "id", `"seed"`)
			messageField(t, v, "user_message", `null`)
			if item.strict && v.Wire.Body["message"] == nil {
				t.Fatal(v)
			}
		} else {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != item.body {
				t.Fatal(err, proof)
			}
		}
	}
	api, calls := messageAPI(t, "", func(*http.Request) *http.Response { return messageWire(404, `{"missing":true}`) })
	v, err := api.GetRecord(context.Background(), "literal/id %2F")
	if v != nil || !gophercloud.ResponseCodeIs(err, 404) || len(*calls) != 1 || (*calls)[0].escaped != messageBase+"/literal%2Fid%20%252F" || (*calls)[0].headers.Get("OpenStack-API-Version") != "" || (*calls)[0].headers.Get("X-OpenStack-Volume-API-Version") != "" {
		t.Fatal(v, err, *calls)
	}
}
func TestPythonCinderMessageDeleteDefaultStrictAndOriginalStatusPolicy(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 299, 300, 304, 399} {
		api, calls := messageAPI(t, "3.3", func(*http.Request) *http.Response { return messageWire(code, `not JSON`) })
		err := api.DeleteRecord(context.Background(), "id")
		if err != nil || len(*calls) != 1 || (*calls)[0].method != http.MethodDelete || (*calls)[0].path != messageBase+"/id" || (*calls)[0].body != "" {
			t.Fatal(code, err, *calls)
		}
	}
	api, calls := messageAPI(t, "3.0", func(*http.Request) *http.Response { return messageWire(404, `{"missing":true}`) })
	if err := api.DeleteRecord(context.Background(), "id"); err != nil || len(*calls) != 1 || (*calls)[0].headers.Get("OpenStack-API-Version") != "volume 3.0" {
		t.Fatal(err, *calls)
	}
	if err := api.DeleteRecord(context.Background(), "id", messages.WithRecordDeleteIgnoreMissing(false)); !gophercloud.ResponseCodeIs(err, 404) || len(*calls) != 2 {
		t.Fatal(err, *calls)
	}
	api, calls = messageAPI(t, "3.3", func(*http.Request) *http.Response { return messageWire(403, `{"policy":false}`) })
	if err := api.DeleteRecord(context.Background(), "id"); !gophercloud.ResponseCodeIs(err, 403) || len(*calls) != 1 {
		t.Fatal(err, *calls)
	}
}
func collectMessages(seq func(func(*messages.MessageRecord, error) bool)) ([]*messages.MessageRecord, error) {
	var values []*messages.MessageRecord
	for value, err := range seq {
		if err != nil {
			return values, err
		}
		values = append(values, value)
	}
	return values, nil
}
func TestPythonCinderMessageListQueryClassificationOwnedLocalFiltersAndRawCap(t *testing.T) {
	query := map[string]json.RawMessage{"project_id": json.RawMessage(`["p1",null,"p2"]`), "sort": json.RawMessage(`"created_at:desc"`), "message_level": json.RawMessage(`"ERROR"`), "unknown": json.RawMessage(`{bad`)}
	option := messages.WithRecordListOpts(messages.RecordListOpts{Query: query})
	query["project_id"][2] = 'X'
	api, calls := messageAPI(t, "3.37", func(r *http.Request) *http.Response {
		if r.Header.Get("Accept") != "application/json" {
			t.Fatal(r.Header)
		}
		return messageWire(200, `{"messages":[{"id":"a","message_level":"WARN"},{"id":"b","message_level":"ERROR","unknown":null},{"id":"c","message_level":"ERROR"}],"next":"https://wrong.example/messages"}`)
	})
	seq := api.ListRecords(context.Background(), option, messages.WithRecordListLimit(0), messages.WithRecordListMaxItems(2))
	if len(*calls) != 0 {
		t.Fatal("eager list", *calls)
	}
	values, err := collectMessages(seq)
	if err != nil || len(values) != 1 || len(*calls) != 1 {
		t.Fatal(values, err, *calls)
	}
	messageField(t, values[0], "id", `"b"`)
	q, _ := url.ParseQuery((*calls)[0].query)
	if !reflect.DeepEqual(q, url.Values{"project_id": {"p1", "p2"}, "sort": {"created_at:desc"}, "limit": {"2"}}) || values[0].Resource.Body["unknown"] != nil || string(values[0].Wire.Body["unknown"]) != "null" {
		t.Fatal(q, values)
	}
	api, calls = messageAPI(t, "3.3", func(*http.Request) *http.Response {
		return messageWire(200, `{"messages":[{"id":1,"request_id":false},{"id":2,"request_id":"False"}]}`)
	})
	values, err = collectMessages(api.ListRecords(context.Background(), messages.WithRecordListQuery("id", true), messages.WithRecordListQuery("request_id", "False")))
	if err != nil || len(values) != 1 || len(*calls) != 1 {
		t.Fatal(values, err, *calls)
	}
	messageField(t, values[0], "id", `1`)
}
func TestPythonCinderMessageListDictionarySemanticsAfterDescriptorConversion(t *testing.T) {
	for _, item := range []struct {
		value, filter string
		valid, match  bool
	}{{`"truthy"`, `{}`, true, true}, {`"truthy"`, `{"k":true}`, false, false}, {`""`, `{}`, true, false}, {`null`, `null`, true, true}, {`1`, `true`, true, true}} {
		api, _ := messageAPI(t, "3.3", func(*http.Request) *http.Response { return messageWire(200, `{"messages":[{"id":`+item.value+`}]}`) })
		values, err := collectMessages(api.ListRecords(context.Background(), messages.WithRecordListOpts(messages.RecordListOpts{Query: map[string]json.RawMessage{"id": json.RawMessage(item.filter)}})))
		if (err == nil) != item.valid || (len(values) == 1) != item.match {
			t.Fatal(item, values, err)
		}
		if !item.valid {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 200 {
				t.Fatal(err, proof)
			}
		}
	}
}
func TestPythonCinderMessageListLinkedMarkerOffsetAndSinglePage(t *testing.T) {
	for _, kind := range []string{"messages_links", "Link", "next"} {
		t.Run(kind, func(t *testing.T) {
			api, calls := messageAPI(t, "3.37", func(r *http.Request) *http.Response {
				if r.URL.Query().Get("marker") == "last" {
					return messageWire(200, `{"messages":[{"id":"b"}]}`)
				}
				extra := ""
				if kind == "messages_links" {
					extra = `,"messages_links":[{"rel":"next","href":"?marker=last"}]`
				}
				if kind == "next" {
					extra = `,"next":"?marker=last"`
				}
				reply := messageWire(200, `{"messages":[{"id":"a"}]`+extra+`}`)
				if kind == "Link" {
					reply.Header.Set("Link", `<?marker=last>; rel="next"`)
				}
				return reply
			})
			values, err := collectMessages(api.ListRecords(context.Background(), messages.WithRecordListQuery("project_id", "p")))
			if err != nil || len(values) != 2 || len(*calls) != 2 {
				t.Fatal(values, err, *calls)
			}
			for _, call := range *calls {
				q, _ := url.ParseQuery(call.query)
				if q.Get("project_id") != "p" {
					t.Fatal(q)
				}
			}
		})
	}
	api, calls := messageAPI(t, "3.37", func(r *http.Request) *http.Response {
		if r.URL.Query().Get("marker") == "9007199254740993" {
			return messageWire(200, `{"messages":[]}`)
		}
		return messageWire(200, `{"messages":{"id":9007199254740993}}`)
	})
	values, err := collectMessages(api.ListRecords(context.Background(), messages.WithRecordListLimit(5)))
	if err != nil || len(values) != 1 || len(*calls) != 2 {
		t.Fatal(values, err, *calls)
	}
	*calls = nil
	values, err = collectMessages(api.ListRecords(context.Background(), messages.WithRecordListLimit(5), messages.WithRecordListPaginated(false)))
	if err != nil || len(values) != 1 || len(*calls) != 1 {
		t.Fatal(values, err, *calls)
	}
	api, calls = messageAPI(t, "3.37", func(r *http.Request) *http.Response {
		if r.URL.Query().Get("offset") == "2" {
			return messageWire(200, `{"messages":[{"id":"b"}]}`)
		}
		return messageWire(200, `{"messages":[{"id":"a"}],"next":"?offset=2&limit=2"}`)
	})
	values, err = collectMessages(api.ListRecords(context.Background(), messages.WithRecordListQuery("offset", 0)))
	if err != nil || len(values) != 2 || len(*calls) != 2 {
		t.Fatal(values, err, *calls)
	}
}
func TestPythonCinderMessageListEnvelopeConstructorAndContinuationErrors(t *testing.T) {
	for _, body := range []string{`{}`, `{"messages":null}`, `{"messages":[null]}`, `{"messages":[{"connection":null}]}`, `{"messages":[{"microversion":"3.3"}]}`, `{"messages":[{"_synchronized":false}]}`, `{"messages":[{"id":"a"}],"next":"?project_id=changed&marker=a"}`} {
		api, calls := messageAPI(t, "3.3", func(*http.Request) *http.Response { return messageWire(200, body) })
		values, err := collectMessages(api.ListRecords(context.Background(), messages.WithRecordListQuery("project_id", "p")))
		var proof *resource.ResponseError
		if err == nil || len(values) > 1 || len(*calls) != 1 || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body {
			t.Fatal(body, values, err, proof, *calls)
		}
	}
	api, calls := messageAPI(t, "3.3", func(*http.Request) *http.Response { return messageWire(204, "") })
	_, err := collectMessages(api.ListRecords(context.Background()))
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != 204 || len(*calls) != 1 {
		t.Fatal(err, *calls)
	}
}

type messageFaultBody struct {
	data            []byte
	read, close     error
	onRead, onClose func()
}

func (b *messageFaultBody) Read(p []byte) (int, error) {
	n := copy(p, b.data)
	b.data = b.data[n:]
	if b.onRead != nil {
		b.onRead()
		b.onRead = nil
	}
	if len(b.data) == 0 {
		if b.read != nil {
			return n, b.read
		}
		return n, io.EOF
	}
	return n, nil
}
func (b *messageFaultBody) Close() error {
	if b.onClose != nil {
		b.onClose()
	}
	return b.close
}
func TestPythonCinderMessageRejectedStatusFamiliesPreservePhysicalCauses(t *testing.T) {
	for _, method := range []string{"GET", "LIST", "DELETE"} {
		for _, code := range []int{400, 401, 403, 404, 405, 409, 418, 500, 503, 599} {
			t.Run(fmt.Sprintf("%s_%d", method, code), func(t *testing.T) {
				readCause, closeCause := errors.New("read"), errors.New("close")
				closes := 0
				body := `{"failure":true}`
				api, calls := messageAPI(t, "3.3", func(*http.Request) *http.Response {
					return &http.Response{StatusCode: code, Header: http.Header{"X-Request-Id": {"rejected"}}, Body: &messageFaultBody{data: []byte(body), read: readCause, close: closeCause, onClose: func() { closes++ }}}
				})
				var err error
				switch method {
				case "GET":
					_, err = api.GetRecord(context.Background(), "id")
				case "LIST":
					_, err = collectMessages(api.ListRecords(context.Background()))
				case "DELETE":
					err = api.DeleteRecord(context.Background(), "id")
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.As(err, &native) || native.Actual != code || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "rejected" || len(*calls) != 1 || closes != 1 {
					t.Fatal(err, native, *calls, closes)
				}
			})
		}
	}
}
func TestPythonCinderMessageAcceptedPhysicalFaultsAndStickySource(t *testing.T) {
	for _, method := range []string{"GET", "LIST", "DELETE"} {
		for _, sticky := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s_sticky%v", method, sticky), func(t *testing.T) {
				readCause, closeCause := errors.New("accepted read"), errors.New("accepted close")
				closes := 0
				body := `{"messages":{"id":"a"}}`
				if method == "LIST" {
					body = `{"messages":[{"id":"a"}]}`
				}
				api, calls := messageAPI(t, "3.3", func(*http.Request) *http.Response { return nil })
				api.RawClient().HTTPClient.Transport = messageTransport(func(r *http.Request) (*http.Response, error) {
					*calls = append(*calls, messageCall{method: r.Method})
					fault := &messageFaultBody{data: []byte(body), read: readCause, close: closeCause, onClose: func() { closes++ }}
					if sticky {
						fault.read = nil
						fault.close = nil
						fault.onRead = func() { api.RawClient().Microversion = "3.80" }
						fault.onClose = func() { closes++; api.RawClient().Microversion = "3.3" }
					}
					return &http.Response{StatusCode: 200, Header: http.Header{"X-Request-Id": {"accepted"}}, Body: fault}, nil
				})
				var err error
				var value *messages.MessageRecord
				switch method {
				case "GET":
					value, err = api.GetRecord(context.Background(), "id")
				case "LIST":
					_, err = collectMessages(api.ListRecords(context.Background()))
				case "DELETE":
					err = api.DeleteRecord(context.Background(), "id")
				}
				var proof *resource.ResponseError
				if err == nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || closes != 1 || len(*calls) != 1 {
					t.Fatal(value, err, proof, *calls, closes)
				}
				if method == "GET" && (value == nil || value.StatusCode != 200 || value.Resource != nil || string(value.Envelope) != body) {
					t.Fatal(value)
				}
				if sticky {
					if !errors.Is(err, resource.ErrInvalidOption) || api.RawClient().Microversion != "3.3" {
						t.Fatal(err)
					}
				} else {
					if !errors.Is(err, readCause) || !errors.Is(err, closeCause) {
						t.Fatal(err)
					}
				}
			})
		}
	}
}
func TestPythonCinderMessageRetryPreservesCleanMissingAndFixedVersionHeaders(t *testing.T) {
	api, calls := messageAPI(t, "3.3", func(*http.Request) *http.Response { return messageWire(404, `{}`) })
	api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
		return original
	}
	if err := api.DeleteRecord(context.Background(), "id"); err != nil || len(*calls) != 1 {
		t.Fatal(err, *calls)
	}
	for _, method := range []string{"GET", "LIST", "DELETE"} {
		for _, version := range []string{"", "3.3"} {
			api, calls := messageAPI(t, version, func(*http.Request) *http.Response { return messageWire(409, `{"conflict":true}`) })
			api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				opts.MoreHeaders["OpenStack-API-Version"] = "volume 3.80"
				return nil
			}
			var err error
			switch method {
			case "GET":
				_, err = api.GetRecord(context.Background(), "id")
			case "LIST":
				_, err = collectMessages(api.ListRecords(context.Background()))
			case "DELETE":
				err = api.DeleteRecord(context.Background(), "id")
			}
			if !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 409) || len(*calls) != 1 {
				t.Fatal(method, version, err, *calls)
			}
		}
	}
}
func TestPythonCinderMessagePreflightCancellationAndRetainedOptionOwnership(t *testing.T) {
	api, calls := messageAPI(t, "3.3", func(*http.Request) *http.Response { return messageWire(200, `{"messages":{}}`) })
	for _, id := range []string{"", "..", "bad\n"} {
		_, err := api.GetRecord(context.Background(), id)
		if err == nil || len(*calls) != 0 {
			t.Fatal(err, *calls)
		}
	}
	for _, opts := range [][]messages.RecordGetOption{{nil}, {messages.WithRecordGetHeaders(map[string]string{"X-Auth-Token": "bad"})}, {messages.WithRecordGetMicroversion("3.3\n")}} {
		_, err := api.GetRecord(context.Background(), "id", opts...)
		if err == nil || len(*calls) != 0 {
			t.Fatal(err, *calls)
		}
	}
	for _, opts := range [][]messages.RecordListOption{{nil}, {messages.WithRecordListLimit(-1)}, {messages.WithRecordListMaxItems(-1)}, {messages.WithRecordListQuery("limit", []int{1, 2})}, {messages.WithRecordListOpts(messages.RecordListOpts{Query: map[string]json.RawMessage{"event_id": json.RawMessage(`{bad`)}})}} {
		_, err := collectMessages(api.ListRecords(context.Background(), opts...))
		if err == nil || len(*calls) != 0 {
			t.Fatal(err, *calls)
		}
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause, callback := errors.New("caller cause"), errors.New("callback cause")
	_, err := api.GetRecord(ctx, "id", func(*messages.RecordGetOpts) error { cancel(cause); return callback })
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, callback) || len(*calls) != 0 {
		t.Fatal(err, *calls)
	}
	var retained *messages.RecordGetOpts
	_, err = api.GetRecord(context.Background(), "id", func(v *messages.RecordGetOpts) error {
		retained = v
		v.Headers = map[string]string{"X-Custom": "owned"}
		return nil
	}, func(*messages.RecordGetOpts) error {
		retained.Headers["X-Auth-Token"] = "retained mutation"
		return nil
	})
	if err != nil || len(*calls) != 1 || (*calls)[0].headers.Get("X-Custom") != "owned" || api.RawClient().MoreHeaders != nil {
		t.Fatal(err, *calls)
	}
}
func TestPythonCinderMessageConcurrentOwnedOptionsAndLiveReauthentication(t *testing.T) {
	api, _ := messageAPI(t, "3.3", func(*http.Request) *http.Response { return nil })
	api.RawClient().HTTPClient.Transport = messageTransport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("X-Custom") != "owned" {
			t.Error(r.Header)
		}
		return messageWire(200, `{"message":{"id":"a"}}`), nil
	})
	option := messages.WithRecordGetHeaders(map[string]string{"X-Custom": "owned"})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := api.GetRecord(context.Background(), "a", option); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	count := 0
	var tokens []string
	api, calls := messageAPI(t, "3.3", func(r *http.Request) *http.Response {
		count++
		tokens = append(tokens, r.Header.Get("X-Auth-Token"))
		if count == 1 {
			return messageWire(401, `{"expired":true}`)
		}
		return messageWire(200, `{"message":{"id":"a"}}`)
	})
	reauths := 0
	api.RawClient().ReauthFunc = func(context.Context) error { reauths++; api.RawClient().SetToken("fresh"); return nil }
	v, err := api.GetRecord(context.Background(), "a")
	if err != nil || v == nil || reauths != 1 || len(*calls) != 2 || !reflect.DeepEqual(tokens, []string{"test-token", "fresh"}) {
		t.Fatal(v, err, reauths, *calls, tokens)
	}
}

func TestPythonCinderMessageTransportExposedInterimStatus(t *testing.T) {
	for _, operation := range []string{"GET", "LIST", "DELETE"} {
		t.Run(operation, func(t *testing.T) {
			body := `{"message":{"id":"actual"}}`
			if operation == "LIST" {
				body = `{"messages":[{"id":"actual"}]}`
			}
			api, calls := messageAPI(t, "3.3", func(*http.Request) *http.Response { return messageWire(199, body) })
			var err error
			switch operation {
			case "GET":
				var value *messages.MessageRecord
				value, err = api.GetRecord(context.Background(), "requested")
				if value == nil || value.StatusCode != 199 || string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "actual" {
					t.Fatal(value, err)
				}
				messageField(t, value, "id", `"actual"`)
			case "LIST":
				var values []*messages.MessageRecord
				values, err = collectMessages(api.ListRecords(context.Background()))
				if len(values) != 1 || values[0].StatusCode != 199 || string(values[0].Envelope) != body || values[0].Header.Get("X-Request-Id") != "actual" {
					t.Fatal(values, err)
				}
			case "DELETE":
				err = api.DeleteRecord(context.Background(), "requested")
			}
			if err != nil || len(*calls) != 1 {
				t.Fatal(operation, err, *calls)
			}
		})
	}
}

func TestPythonCinderMessageRetryReservedHeaderAbsenceAndSelectedOmission(t *testing.T) {
	cases := []struct {
		name, key, value string
		omit             bool
	}{
		{"authorization", "aUtHoRiZaTiOn", "Bearer retry", false},
		{"service token", "x-service-token", "retry", false},
		{"host", "Host", "foreign.invalid", false},
		{"cookie", "Cookie", "session=retry", false},
		{"content length", "Content-Length", "1", false},
		{"transfer encoding", "Transfer-Encoding", "chunked", false},
		{"connection", "Connection", "close", false},
		{"trailer", "Trailer", "X-Retry", false},
		{"te", "te", "trailers", false},
		{"upgrade", "Upgrade", "websocket", false},
		{"foreign Nova version", "x-openstack-nova-api-version", "2.95", false},
		{"omit selected version", "OpenStack-API-Version", "", true},
		{"omit selected legacy version", "X-OpenStack-Volume-API-Version", "", true},
	}
	for _, operation := range []string{"GET", "LIST", "DELETE"} {
		for _, version := range []string{"", "3.3"} {
			for _, tc := range cases {
				if tc.omit && version == "" {
					continue
				}
				t.Run(operation+"/"+version+"/"+tc.name, func(t *testing.T) {
					body := `{"retry":"original rejection"}`
					closes, callbacks := 0, 0
					callbackCause := errors.New("retry callback cause")
					api, calls := messageAPI(t, version, func(*http.Request) *http.Response {
						response := messageWire(503, body)
						response.Body = &messageFaultBody{data: []byte(body), onClose: func() { closes++ }}
						return response
					})
					api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
						callbacks++
						if tc.omit {
							opts.OmitHeaders = append(opts.OmitHeaders, tc.key)
						} else {
							opts.MoreHeaders[tc.key] = tc.value
						}
						return callbackCause
					}
					var err error
					switch operation {
					case "GET":
						value, failure := api.GetRecord(context.Background(), "requested")
						err = failure
						if value != nil && value.Resource != nil {
							t.Fatal("rejected response became Resource", value)
						}
					case "LIST":
						values, failure := collectMessages(api.ListRecords(context.Background()))
						err = failure
						if len(values) != 0 {
							t.Fatal("rejected list yielded records", values)
						}
					case "DELETE":
						err = api.DeleteRecord(context.Background(), "requested")
					}
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, callbackCause) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "actual" || len(*calls) != 1 || closes != 1 || callbacks != 1 {
						t.Fatal(err, native, *calls, closes, callbacks)
					}
					wantMethod := http.MethodGet
					wantPath := messageBase + "/requested"
					if operation == "DELETE" {
						wantMethod = http.MethodDelete
					} else if operation == "LIST" {
						wantPath = messageBase
					}
					if native.Method != wantMethod || !strings.HasSuffix(native.URL, wantPath) || (*calls)[0].headers.Get("X-Auth-Token") != "test-token" || (*calls)[0].body != "" || api.RawClient().Microversion != version || len(api.RawClient().MoreHeaders) != 0 {
						t.Fatal("lost actual method/route or mutated caller", native, *calls, api.RawClient())
					}
				})
			}
		}
	}
}

func TestPythonCinderMessageRetryOrdinaryHeadersAndLiveTokenRemainAvailable(t *testing.T) {
	for _, operation := range []string{"GET", "LIST", "DELETE"} {
		t.Run(operation, func(t *testing.T) {
			count := 0
			var tokens []string
			api, calls := messageAPI(t, "3.3", func(r *http.Request) *http.Response {
				count++
				tokens = append(tokens, r.Header.Get("X-Auth-Token"))
				if r.Header.Get("OpenStack-API-Version") != "volume 3.3" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.3" {
					t.Fatal(r.Header)
				}
				if count == 1 {
					return messageWire(503, `{"temporary":true}`)
				}
				if r.Header.Get("X-Retry-Probe") != "ordinary" {
					t.Fatal("ordinary retry header lost", r.Header)
				}
				if operation == "LIST" {
					return messageWire(200, `{"messages":[{"id":"actual"}]}`)
				}
				return messageWire(200, `{"message":{"id":"actual"}}`)
			})
			api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				opts.MoreHeaders["X-Retry-Probe"] = "ordinary"
				api.RawClient().SetToken("fresh")
				return nil
			}
			var err error
			switch operation {
			case "GET":
				_, err = api.GetRecord(context.Background(), "requested")
			case "LIST":
				_, err = collectMessages(api.ListRecords(context.Background()))
			case "DELETE":
				err = api.DeleteRecord(context.Background(), "requested")
			}
			if err != nil || len(*calls) != 2 || !reflect.DeepEqual(tokens, []string{"test-token", "fresh"}) || len(api.RawClient().MoreHeaders) != 0 {
				t.Fatal(err, *calls, tokens, api.RawClient().MoreHeaders)
			}
		})
	}
}
