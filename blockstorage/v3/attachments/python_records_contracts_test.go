package attachments_test

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

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func attachmentRecordField(t *testing.T, v *attachments.AttachmentRecord, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		t.Fatalf("%s: %#v want %s", key, v, want)
	}
}
func attachmentRecordIDs(t *testing.T, ctx context.Context, api *attachments.API, options ...attachments.RecordListOption) ([]string, error) {
	t.Helper()
	var ids []string
	for v, err := range api.ListRecords(ctx, options...) {
		if err != nil {
			return ids, err
		}
		var id string
		if err := json.Unmarshal(v.Resource.Body["id"], &id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func TestPythonAttachmentRecordCreateReservationAndOwnedPartialEvidence(t *testing.T) {
	api, calls := pythonAttachmentAPI(t, func(*http.Request) *http.Response {
		r := pythonAttachmentWire(201, `{"attachment":{"id":"att1","mode":"ro","attach_time":"literal","connection_info":{"large":9007199254740993},"unknown":[false,null]}}`)
		r.Header.Set("X-Request-Id", "proof")
		return r
	})
	fields := map[string]json.RawMessage{"connector": json.RawMessage(`{"host":"compute"}`), "name": json.RawMessage(`""`), "detached_at": json.RawMessage(`null`), "unknown": json.RawMessage(`{bad`)}
	option := attachments.WithRecordCreateOpts(attachments.RecordCreateOpts{Fields: fields})
	fields["connector"][2] = 'X'
	v, err := api.CreateRecord(context.Background(), "vol1", option, attachments.WithRecordCreateField("unused", make(chan int)))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"attachment":{"connector":{"host":"compute"},"detach_time":null,"name":"","volume_uuid":"vol1"}}`
	if len(*calls) != 1 || (*calls)[0].path != pythonAttachmentBase || (*calls)[0].method != http.MethodPost || (*calls)[0].body != want {
		t.Fatal(*calls)
	}
	for key, raw := range map[string]string{"id": `"att1"`, "volume_id": `"vol1"`, "instance": `null`, "attach_mode": `"ro"`, "attached_at": `"literal"`, "detached_at": `null`, "name": `""`, "location": `null`, "connector": `{"host":"compute"}`, "connection_info": `{"large":9007199254740993}`} {
		attachmentRecordField(t, v, key, raw)
	}
	if len(v.Resource.Body) != 11 || v.Wire == nil || string(v.Wire.Body["unknown"]) != `[false,null]` || v.Wire.Body["volume_id"] != nil || v.StatusCode != 201 {
		t.Fatal(v)
	}
	v.Resource.Body["connection_info"][2] = 'X'
	v.Header.Set("X-Request-Id", "changed")
	if !strings.Contains(string(v.Wire.Body["connection_info"]), "large") || v.Wire.Header.Get("X-Request-Id") != "proof" || v.Resource.Header.Get("X-Request-Id") != "proof" || !strings.Contains(string(v.Envelope), "9007199254740993") {
		t.Fatal("evidence aliases view")
	}
	*calls = nil
	if _, err := api.CreateRecord(context.Background(), "vol1"); err != nil || len(*calls) != 1 || (*calls)[0].body != `{"attachment":{"volume_uuid":"vol1"}}` {
		t.Fatal(err, *calls)
	}
}

func TestPythonAttachmentRecordCreateInstanceTruthinessAndModeVersions(t *testing.T) {
	for _, value := range []any{nil, false, 0, "", []any{}, map[string]any{}, "server", 42} {
		t.Run(fmt.Sprintf("instance_%#v", value), func(t *testing.T) {
			api, calls := pythonAttachmentAPI(t, func(*http.Request) *http.Response { return pythonAttachmentWire(202, "") })
			v, err := api.CreateRecord(context.Background(), "vol", attachments.WithRecordCreateField("instance", value), attachments.WithRecordCreateField("connector", false))
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := json.Marshal(value)
			truthy := value == "server" || value == 42
			key := "instance"
			if truthy {
				key = "instance_uuid"
			}
			var envelope map[string]map[string]json.RawMessage
			_ = json.Unmarshal([]byte((*calls)[0].body), &envelope)
			if string(envelope["attachment"][key]) != string(raw) || string(envelope["attachment"]["connector"]) != "false" {
				t.Fatal(*calls)
			}
			attachmentRecordField(t, v, "instance", string(raw))
		})
	}
	for _, version := range []string{"3.0", "3.27", "3.53", "3.54", "3.71", "3.80"} {
		for _, value := range []any{nil, false, 0, "", []any{}, "rw"} {
			t.Run(fmt.Sprintf("mode_%s_%#v", version, value), func(t *testing.T) {
				api, calls, _ := attachmentModeFixture(t, version, "3.90", "3.0", func(*http.Request) *http.Response { return pythonAttachmentWire(204, "") })
				v, err := api.CreateRecord(context.Background(), "vol", attachments.WithRecordCreateField("attach_mode", value))
				if err != nil {
					t.Fatal(err)
				}
				var envelope map[string]map[string]json.RawMessage
				_ = json.Unmarshal([]byte((*calls)[len(*calls)-1].body), &envelope)
				expectedCalls := 2
				if value == "rw" {
					expectedCalls = 1
				}
				if len(*calls) != expectedCalls || len(v.Discovery) != expectedCalls-1 {
					t.Fatal("mode capability request/proof", v, *calls)
				}
				present := version == "3.0" || version == "3.27" || version == "3.53" || value == "rw"
				_, got := envelope["attachment"]["mode"]
				if got != present {
					t.Fatal(version, value, *calls)
				}
				raw, _ := json.Marshal(value)
				attachmentRecordField(t, v, "attach_mode", string(raw))
			})
		}
	}
	for _, version := range []string{"", "latest"} {
		api, calls, _ := attachmentModeFixture(t, version, "3.90", "3.0", func(*http.Request) *http.Response { return pythonAttachmentWire(200, `{}`) })
		value, err := api.CreateRecord(context.Background(), "vol", attachments.WithRecordCreateField("mode", false))
		if err != nil || len(*calls) != 2 || len(value.Discovery) != 1 {
			t.Fatal(value, err, *calls)
		}
		present := strings.Contains((*calls)[1].body, `"mode":false`)
		if present != (version == "latest") {
			t.Fatal("Source selected-default support", version, *calls)
		}
		*calls = nil
		if value, err := api.CreateRecord(context.Background(), "vol", attachments.WithRecordCreateField("mode", false), attachments.WithRecordCreateModeSupported(true)); err != nil || len(*calls) != 1 || strings.Contains((*calls)[0].body, `"mode"`) || len(value.Discovery) != 0 {
			t.Fatal(value, err, *calls)
		}
		*calls = nil
		if value, err := api.CreateRecord(context.Background(), "vol", attachments.WithRecordCreateField("mode", false), attachments.WithRecordCreateModeSupported(false)); err != nil || len(*calls) != 1 || !strings.Contains((*calls)[0].body, `"mode":false`) || len(value.Discovery) != 0 {
			t.Fatal(value, err, *calls)
		}
	}

}

func TestPythonAttachmentRecordCreateAliasOrderAndPreparation(t *testing.T) {
	api, calls := pythonAttachmentAPI(t, func(*http.Request) *http.Response {
		return pythonAttachmentWire(200, `{"attachment":{"mode":"wire","attach_mode":"attribute","attach_time":"wiretime","attached_at":"attributetime","connection_info":42}}`)
	})
	v, err := api.CreateRecord(context.Background(), "vol", attachments.WithRecordCreateField("mode", "ro"), attachments.WithRecordCreateField("attach_mode", "rw"), attachments.WithRecordCreateField("instance_id", "ignored"))
	if err != nil || (*calls)[0].body != `{"attachment":{"mode":"rw","volume_uuid":"vol"}}` {
		t.Fatal(err, *calls)
	}
	attachmentRecordField(t, v, "attach_mode", `"attribute"`)
	attachmentRecordField(t, v, "attached_at", `"attributetime"`)
	attachmentRecordField(t, v, "connection_info", `{}`)
	for _, options := range [][]attachments.RecordCreateOption{
		{nil}, {attachments.WithRecordCreateField("volume_id", "other")}, {attachments.WithRecordCreateOpts(attachments.RecordCreateOpts{Fields: map[string]json.RawMessage{"mode": json.RawMessage(`"ro"`), "attach_mode": json.RawMessage(`"rw"`)}})}, {attachments.WithRecordCreateOpts(attachments.RecordCreateOpts{Fields: map[string]json.RawMessage{"connector": json.RawMessage(`{bad`)}})},
	} {
		*calls = nil
		if _, err := api.CreateRecord(context.Background(), "vol", options...); err == nil || len(*calls) != 0 {
			t.Fatal(err, *calls)
		}
	}
	*calls = nil
	if _, err := api.CreateRecord(context.Background(), " "); !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
		t.Fatal(err, *calls)
	}
}

func TestPythonAttachmentRecordCreateResponseTranslationAndStatusPolicy(t *testing.T) {
	for _, item := range []struct {
		name, body string
		valid      bool
	}{
		{"singular", `{"attachment":{"instance":false,"mode":0}}`, true}, {"flat", `{"id":"id","unknown":true}`, true}, {"empty", `{}`, true}, {"blank", "", true}, {"malformed", "{bad", true}, {"array", `[]`, false}, {"null", `null`, false}, {"wrapped_null", `{"attachment":null}`, false}, {"bad_utf8", string([]byte{'{', '"', 'x', '"', ':', '"', 255, '"', '}'}), false},
	} {
		t.Run(item.name, func(t *testing.T) {
			api, calls := pythonAttachmentAPI(t, func(*http.Request) *http.Response { return pythonAttachmentWire(200, item.body) })
			v, err := api.CreateRecord(context.Background(), "vol")
			if (err == nil) != item.valid || len(*calls) != 1 || v == nil || string(v.Envelope) != item.body || v.StatusCode != 200 {
				t.Fatal(v, err, *calls)
			}
			if !item.valid {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || string(proof.Body) != item.body {
					t.Fatal(err, proof)
				}
			}
		})
	}
	for _, code := range []int{200, 201, 202, 204, 206, 299, 301, 399, 400, 403, 404, 409, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			api, calls := pythonAttachmentAPI(t, func(*http.Request) *http.Response { return pythonAttachmentWire(code, `{}`) })
			v, err := api.CreateRecord(context.Background(), "vol")
			if len(*calls) != 1 {
				t.Fatal(*calls)
			}
			if code < 400 {
				if err != nil || v == nil || v.StatusCode != code {
					t.Fatal(v, err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if v != nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != `{}` {
					t.Fatal(v, err, native)
				}
			}
		})
	}
}

func TestPythonAttachmentRecordListSummaryQueriesAndLinkedPages(t *testing.T) {
	api, calls := pythonAttachmentAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("marker") == "a" {
			return pythonAttachmentWire(200, `{"attachments":[{"id":"b","mode":"ro","attach_time":"literal","connection_info":42}],"attachments_links":[]}`)
		}
		return pythonAttachmentWire(200, `{"attachments":[{"id":"a","mode":"wire","attach_mode":"last","unknown":9007199254740993}],"attachments_links":[{"rel":"next","href":"?marker=a"}]}`)
	})
	headers := map[string]string{"x-custom": "owned"}
	query := map[string]json.RawMessage{"volume_id": json.RawMessage(`"vol"`), "all_projects": json.RawMessage(`true`), "instance_id": json.RawMessage(`["a",null,"b"]`), "unknown": json.RawMessage(`{bad`)}
	option := attachments.WithRecordListOpts(attachments.RecordListOpts{Query: query, Headers: headers})
	query["volume_id"][1] = 'X'
	headers["x-custom"] = "changed"
	seq := api.ListRecords(context.Background(), option)
	if len(*calls) != 0 {
		t.Fatal("eager HTTP")
	}
	var values []*attachments.AttachmentRecord
	for v, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, v)
	}
	if len(values) != 2 || len(*calls) != 2 {
		t.Fatal(values, *calls)
	}
	for _, call := range *calls {
		q, _ := url.ParseQuery(call.query)
		if call.path != pythonAttachmentBase || call.method != http.MethodGet || call.body != "" || q.Get("volume_id") != "vol" || q.Get("all_tenants") != "True" || !reflect.DeepEqual(q["instance_id"], []string{"a", "b"}) || q.Has("unknown") {
			t.Fatal(call)
		}
	}
	attachmentRecordField(t, values[0], "attach_mode", `"last"`)
	attachmentRecordField(t, values[1], "connection_info", `{}`)
	attachmentRecordField(t, values[1], "attached_at", `"literal"`)
	if string(values[0].Wire.Body["unknown"]) != `9007199254740993` || values[0].Wire.Body["location"] != nil {
		t.Fatal(values[0])
	}
	*calls = nil
	if ids, err := attachmentRecordIDs(t, context.Background(), api, option, attachments.WithRecordListPaginated(false)); err != nil || !reflect.DeepEqual(ids, []string{"a"}) || len(*calls) != 1 {
		t.Fatal(ids, err, *calls)
	}
}

func TestPythonAttachmentRecordListServerAliasPrecedence(t *testing.T) {
	for _, item := range []struct {
		client   any
		expected string
	}{{true, "True"}, {false, "False"}, {nil, ""}} {
		api, calls := pythonAttachmentAPI(t, func(*http.Request) *http.Response { return pythonAttachmentWire(200, `{"attachments":[]}`) })
		_, err := attachmentRecordIDs(t, context.Background(), api, attachments.WithRecordListQuery("all_tenants", false), attachments.WithRecordListQuery("all_projects", item.client))
		q, _ := url.ParseQuery((*calls)[0].query)
		if err != nil || q.Get("all_tenants") != item.expected || q.Has("all_projects") {
			t.Fatal(err, q)
		}
	}
	api, calls := pythonAttachmentAPI(t, func(*http.Request) *http.Response { return pythonAttachmentWire(200, `{"attachments":[]}`) })
	_, err := attachmentRecordIDs(t, context.Background(), api, attachments.WithRecordListQuery("all_tenants", true))
	if err != nil || !strings.Contains((*calls)[0].query, "all_tenants=True") {
		t.Fatal(err, *calls)
	}
}

func TestPythonAttachmentRecordListLocalFilteringAndRawRowCap(t *testing.T) {
	api, calls := pythonAttachmentAPI(t, func(*http.Request) *http.Response {
		return pythonAttachmentWire(200, `{"attachments":[{"id":"a","connector":{"host":"wrong"}},{"id":"b","connector":{"host":"right","extra":9007199254740993}},{"id":"c","connector":{"host":"right"}}],"attachments_links":[{"rel":"next","href":"https://evil.example/attachments"}]}`)
	})
	ids, err := attachmentRecordIDs(t, context.Background(), api, attachments.WithRecordListQuery("connector", map[string]any{"host": "right"}), attachments.WithRecordListMaxItems(2), attachments.WithRecordListLimit(0), attachments.WithRecordListQuery("mode", make(chan int)), attachments.WithRecordListQuery("sort_key", make(chan int)))
	if err != nil || !reflect.DeepEqual(ids, []string{"b"}) || len(*calls) != 1 {
		t.Fatal(ids, err, *calls)
	}
	q, _ := url.ParseQuery((*calls)[0].query)
	if q.Get("limit") != "2" || q.Has("connector") || q.Has("mode") || q.Has("sort_key") {
		t.Fatal(q)
	}
	*calls = nil
	for _, err := range api.ListRecords(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if len(*calls) != 1 {
		t.Fatal(*calls)
	}
}

func TestPythonAttachmentRecordListMarkerFallbackAndSingleton(t *testing.T) {
	api, calls := pythonAttachmentAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("marker") == "last" {
			return pythonAttachmentWire(200, `{"attachments":[]}`)
		}
		return pythonAttachmentWire(200, `{"attachments":{"id":"last","connector":null}}`)
	})
	ids, err := attachmentRecordIDs(t, context.Background(), api, attachments.WithRecordListLimit(5))
	if err != nil || !reflect.DeepEqual(ids, []string{"last"}) || len(*calls) != 2 || !strings.Contains((*calls)[1].query, "marker=last") {
		t.Fatal(ids, err, *calls)
	}
	*calls = nil
	ids, err = attachmentRecordIDs(t, context.Background(), api, attachments.WithRecordListLimit(0))
	if err != nil || !reflect.DeepEqual(ids, []string{"last"}) || len(*calls) != 1 {
		t.Fatal(ids, err, *calls)
	}
}

func TestPythonAttachmentRecordListContinuationAndEnvelopeFailures(t *testing.T) {
	for _, body := range []string{`{"attachments":null}`, `{"attachments":[null]}`, `{"wrong":[]}`, `{"attachments":[]`, `{"attachments":[{"id":"a"}],"attachments_links":[{"rel":"next","href":"https://evil.example/attachments"}]}`, `{"attachments":[{"id":"a"}],"next":"?volume_id=changed&marker=a"}`} {
		api, calls := pythonAttachmentAPI(t, func(*http.Request) *http.Response { return pythonAttachmentWire(200, body) })
		_, err := attachmentRecordIDs(t, context.Background(), api, attachments.WithRecordListQuery("volume_id", "vol"))
		var proof *resource.ResponseError
		if err == nil || !errors.As(err, &proof) || string(proof.Body) != body || len(*calls) != 1 {
			t.Fatal(err, proof, *calls)
		}
	}
	api, calls := pythonAttachmentAPI(t, func(*http.Request) *http.Response { return pythonAttachmentWire(200, `{"attachments":[{"id":"a"}]}`) })
	_, err := attachmentRecordIDs(t, context.Background(), api, attachments.WithRecordListLimit(1), attachments.WithRecordListMarker("a"))
	if err == nil || len(*calls) != 1 {
		t.Fatal(err, *calls)
	}
}

func TestPythonAttachmentRecordListVersionHeaderOwnershipAndPreflight(t *testing.T) {
	api, calls := pythonAttachmentAPI(t, func(req *http.Request) *http.Response {
		if req.Header.Get("OpenStack-API-Version") != "volume 3.80" || req.Header.Get("Accept") != "application/json" || req.Header.Get("X-Selected") != "owned" {
			t.Fatal(req.Header)
		}
		return pythonAttachmentWire(200, `{"attachments":[]}`)
	})
	api.RawClient().MoreHeaders = map[string]string{"x-selected": "original"}
	if _, err := attachmentRecordIDs(t, context.Background(), api, attachments.WithRecordListMicroversion("3.80"), attachments.WithRecordListHeaders(map[string]string{"X-Selected": "owned"})); err != nil {
		t.Fatal(err)
	}
	if api.RawClient().Microversion != "3.54" || api.RawClient().MoreHeaders["x-selected"] != "original" {
		t.Fatal("source mutated")
	}
	*calls = nil
	for _, options := range [][]attachments.RecordListOption{{nil}, {attachments.WithRecordListLimit(-1)}, {attachments.WithRecordListMaxItems(-1)}, {attachments.WithRecordListHeaders(map[string]string{"X-Auth-Token": "bad"})}, {attachments.WithRecordListQuery("marker", []string{"a", "b"})}, {attachments.WithRecordListOpts(attachments.RecordListOpts{Query: map[string]json.RawMessage{"instance": json.RawMessage(`{bad`)}})}} {
		_, err := attachmentRecordIDs(t, context.Background(), api, options...)
		if err == nil || len(*calls) != 0 {
			t.Fatal(err, *calls)
		}
	}
}

type attachmentRecordFaultBody struct {
	data        []byte
	read, close error
	afterRead   func()
}

func (b *attachmentRecordFaultBody) Read(p []byte) (int, error) {
	n := copy(p, b.data)
	b.data = b.data[n:]
	if b.afterRead != nil {
		b.afterRead()
		b.afterRead = nil
	}
	if len(b.data) == 0 {
		if b.read != nil {
			return n, b.read
		}
		return n, io.EOF
	}
	return n, nil
}
func (b *attachmentRecordFaultBody) Close() error { return b.close }
func TestPythonAttachmentRecordPhysicalErrorsPreserveStatusBodyAndCauses(t *testing.T) {
	for _, status := range []int{201, 409, 500} {
		for _, list := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d_list_%v", status, list), func(t *testing.T) {
				readCause, closeCause := errors.New("read failed"), errors.New("close failed")
				api, _ := pythonAttachmentAPI(t, func(*http.Request) *http.Response { return nil })
				calls := 0
				api.RawClient().HTTPClient.Transport = pythonAttachmentTransport(func(*http.Request) (*http.Response, error) {
					calls++
					return &http.Response{StatusCode: status, Header: http.Header{"X-Request-Id": {"fault"}}, Body: &attachmentRecordFaultBody{data: []byte(`{"partial":`), read: readCause, close: closeCause}}, nil
				})
				var err error
				var value *attachments.AttachmentRecord
				if list {
					_, err = attachmentRecordIDs(t, context.Background(), api)
				} else {
					value, err = api.CreateRecord(context.Background(), "vol")
				}
				if !errors.Is(err, readCause) || !errors.Is(err, closeCause) || calls != 1 {
					t.Fatal(err, calls)
				}
				if status < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status || proof.Header.Get("X-Request-Id") != "fault" || string(proof.Body) != `{"partial":` {
						t.Fatal(err, proof)
					}
					if !list && (value == nil || string(value.Envelope) != `{"partial":` || value.Resource != nil) {
						t.Fatal(value)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != status || native.ResponseHeader.Get("X-Request-Id") != "fault" || string(native.Body) != `{"partial":` || value != nil {
						t.Fatal(err, native, value)
					}
				}
			})
		}
	}
}

func TestPythonAttachmentRecordSourceChangesCancellationAndRetryOwnership(t *testing.T) {
	t.Run("option_source_change", func(t *testing.T) {
		api, calls := pythonAttachmentAPI(t, func(*http.Request) *http.Response { return pythonAttachmentWire(200, `{}`) })
		_, err := api.CreateRecord(context.Background(), "vol", func(*attachments.RecordCreateOpts) error { api.RawClient().ResourceBase += "changed/"; return nil })
		if !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(err, *calls)
		}
	})
	t.Run("option_cancel", func(t *testing.T) {
		api, calls := pythonAttachmentAPI(t, func(*http.Request) *http.Response { return pythonAttachmentWire(200, `{}`) })
		ctx, cancel := context.WithCancelCause(context.Background())
		cause := errors.New("cancel cause")
		_, err := api.CreateRecord(ctx, "vol", func(*attachments.RecordCreateOpts) error { cancel(cause); return nil })
		if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || len(*calls) != 0 {
			t.Fatal(err, *calls)
		}
	})
	t.Run("response_source_change", func(t *testing.T) {
		api, _ := pythonAttachmentAPI(t, func(*http.Request) *http.Response { return nil })
		api.RawClient().HTTPClient.Transport = pythonAttachmentTransport(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 201, Header: http.Header{}, Body: &attachmentRecordFaultBody{data: []byte(`{"attachment":{"id":"a"}}`), afterRead: func() { api.RawClient().Microversion = "3.55" }}}, nil
		})
		v, err := api.CreateRecord(context.Background(), "vol")
		var proof *resource.ResponseError
		if v == nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 201 || v.Resource != nil {
			t.Fatal(v, err, proof)
		}
	})
	t.Run("retry_changes_body", func(t *testing.T) {
		api, calls := pythonAttachmentAPI(t, func(*http.Request) *http.Response { return pythonAttachmentWire(409, `{"conflict":true}`) })
		api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
			opts.JSONBody = map[string]any{"attachment": map[string]string{"volume_uuid": "other"}}
			return nil
		}
		_, err := api.CreateRecord(context.Background(), "vol")
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 409 || len(*calls) != 1 {
			t.Fatal(err, *calls)
		}
	})
}

func TestPythonAttachmentRecordConcurrentOptionReuse(t *testing.T) {
	api, _ := pythonAttachmentAPI(t, func(*http.Request) *http.Response { return nil })
	api.RawClient().HTTPClient.Transport = pythonAttachmentTransport(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPost {
			data, _ := io.ReadAll(req.Body)
			if string(data) != `{"attachment":{"connector":{"host":"owned"},"volume_uuid":"vol"}}` {
				t.Errorf("body=%s", data)
			}
			return pythonAttachmentWire(201, `{"attachment":{"id":"a"}}`), nil
		}
		if req.URL.Query().Get("volume_id") != "vol" {
			t.Error(req.URL)
		}
		return pythonAttachmentWire(200, `{"attachments":[]}`), nil
	})
	option := attachments.WithRecordCreateField("connector", map[string]any{"host": "owned"})
	listOption := attachments.WithRecordListQuery("volume_id", "vol")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := api.CreateRecord(context.Background(), "vol", option); err != nil {
				t.Error(err)
			}
			for _, err := range api.ListRecords(context.Background(), listOption) {
				if err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
}
