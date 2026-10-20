package portforwarding_test

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
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func forwardingRecordBody(t *testing.T, row *portforwarding.PortForwardingRecord, key, want string) {
	t.Helper()
	if row == nil || row.Resource == nil || string(row.Resource.Body[key]) != want {
		t.Fatalf("%s: %#v, want %s", key, row, want)
	}
}
func TestPythonPortForwardingMutationsOwnAttributesDescriptorsAndParent(t *testing.T) {
	ctx := context.Background()
	var calls []nativeForwardCall
	api, _ := nativeForwardAPI(t, &calls, func(req *http.Request) *http.Response {
		return nativeForwardWire(201, `{"port_forwarding":{"id":"pf-1","external_port":"0022","internal_port":"-3","x_reply":{"ok":false},"self":"discard","floatingip_id":"untrusted"}}`)
	})
	supplied := map[string]any{"empty": []string{}, "huge": json.Number("9007199254740993")}
	option := portforwarding.WithRecordAttribute("x_extension", supplied)
	supplied["empty"] = []string{"changed"}
	created, err := api.CreateRecord(ctx, "fip-1", portforwarding.WithRecordAttribute("description", nil), portforwarding.WithRecordAttribute("internal_port", false), portforwarding.WithRecordAttribute("external_port", 0), option)
	if err != nil || created.ParentID != "fip-1" || created.StatusCode != 201 || string(created.Wire.Body["external_port"]) != `"0022"` {
		t.Fatal(created, err)
	}
	forwardingRecordBody(t, created, "external_port", "22")
	forwardingRecordBody(t, created, "internal_port", "0")
	forwardingRecordBody(t, created, "description", "null")
	forwardingRecordBody(t, created, "x_extension", `{"empty":[],"huge":9007199254740993}`)
	forwardingRecordBody(t, created, "floatingip_id", `"fip-1"`)
	forwardingRecordBody(t, created, "location", "null")
	if _, exists := created.Resource.Body["self"]; exists || string(created.Wire.Body["self"]) != `"discard"` {
		t.Fatal(created)
	}
	if calls[0].body != `{"port_forwarding":{"description":null,"external_port":0,"internal_port":false,"x_extension":{"empty":[],"huge":9007199254740993}}}` {
		t.Fatal(calls)
	}
	// Empty update is a Source no-op with an unsynchronized ID-only result.
	before := len(calls)
	unchanged, err := api.UpdateRecord(ctx, "fip-1", "pf-1")
	if err != nil || len(calls) != before || unchanged.StatusCode != 0 || unchanged.Wire != nil {
		t.Fatal(unchanged, err, calls)
	}
	forwardingRecordBody(t, unchanged, "id", `"pf-1"`)
	forwardingRecordBody(t, unchanged, "description", "null")
	// The Source type is Resource, not NetworkResource: these are Body attrs.
	if _, err := api.UpdateRecord(ctx, "fip-1", "pf-1", portforwarding.WithRecordAttribute("description", ""), portforwarding.WithRecordAttribute("if_match", []string{}), portforwarding.WithRecordAttribute("if_revision", 0)); err != nil {
		t.Fatal(err)
	}
	if calls[1].method != http.MethodPut || calls[1].body != `{"port_forwarding":{"description":"","if_match":[],"if_revision":0}}` {
		t.Fatal(calls)
	}
}

func TestPythonPortForwardingEmptyCreateBodylessAckAndIntegerErrors(t *testing.T) {
	ctx := context.Background()
	var calls []nativeForwardCall
	api, _ := nativeForwardAPI(t, &calls, func(*http.Request) *http.Response { return nativeForwardWire(204, "") })
	created, err := api.CreateRecord(ctx, "fip-1")
	if err != nil || created == nil || created.Wire != nil || created.StatusCode != 204 || calls[0].body != `{"port_forwarding":{}}` {
		t.Fatal(created, err, calls)
	}
	forwardingRecordBody(t, created, "id", "null")
	changed, err := api.UpdateRecord(ctx, "fip-1", "pf-1", portforwarding.WithRecordAttribute("internal_port", true), portforwarding.WithRecordAttribute("x_null", nil))
	if err != nil || changed.StatusCode != 204 {
		t.Fatal(changed, err)
	}
	forwardingRecordBody(t, changed, "id", `"pf-1"`)
	forwardingRecordBody(t, changed, "internal_port", "true")
	forwardingRecordBody(t, changed, "x_null", "null")
	// Descriptor failures retain the actual accepted response and original wire.
	var badCalls []nativeForwardCall
	bad, _ := nativeForwardAPI(t, &badCalls, func(*http.Request) *http.Response {
		return nativeForwardWire(200, `{"port_forwarding":{"internal_port":"²"}}`)
	})
	row, err := bad.GetRecord(ctx, "fip-1", "pf-1")
	var proof *resource.ResponseError
	if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || row == nil || string(row.Wire.Body["internal_port"]) != `"²"` {
		t.Fatal(row, err, proof)
	}
}

func TestPythonPortForwardingGetFieldsEscapingAndSourceStatusProfile(t *testing.T) {
	ctx := context.Background()
	var calls []nativeForwardCall
	var escaped []string
	api, _ := nativeForwardAPI(t, &calls, func(req *http.Request) *http.Response {
		escaped = append(escaped, req.URL.EscapedPath())
		return nativeForwardWire(299, `{"port_forwarding":{"id":"observed","name":false,"external_port":2222.9}}`)
	})
	fields := []string{"external_port", "vendor:range", "external_port"}
	option := portforwarding.WithRecordFields(fields...)
	fields[0] = "changed"
	got, err := api.GetRecord(ctx, "fip/one", "pf%one", option)
	if err != nil || got.ParentID != "fip/one" || got.StatusCode != 299 {
		t.Fatal(got, err)
	}
	forwardingRecordBody(t, got, "id", `"observed"`)
	forwardingRecordBody(t, got, "external_port", "2222")
	forwardingRecordBody(t, got, "name", "false")
	query, _ := url.ParseQuery(calls[0].query)
	if !reflect.DeepEqual(query, url.Values{"fields": {"external_port", "vendor:range", "external_port"}}) || escaped[0] != "/neutron/v2.0/floatingips/fip%2Fone/port_forwardings/pf%25one" {
		t.Fatal(calls, escaped)
	}
	if _, err := api.GetRecord(ctx, "fip-1", "pf-1", portforwarding.WithRecordFields()); err != nil || calls[1].query != "" {
		t.Fatal(calls, err)
	}
	for _, code := range []int{200, 201, 202, 204, 299, 300, 304, 399, 400} {
		t.Run(fmt.Sprintf("status %d", code), func(t *testing.T) {
			var calls []nativeForwardCall
			other, _ := nativeForwardAPI(t, &calls, func(*http.Request) *http.Response { return nativeForwardWire(code, "") })
			row, err := other.GetRecord(ctx, "fip-1", "pf-1")
			if code < 400 {
				if err != nil || row == nil || row.StatusCode != code {
					t.Fatal(row, err)
				}
				forwardingRecordBody(t, row, "id", `"pf-1"`)
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestPythonPortForwardingListProjectsQueriesFiltersAndLazyPagination(t *testing.T) {
	ctx := context.Background()
	var calls []nativeForwardCall
	api, _ := nativeForwardAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.Header.Get("Accept") != "application/json" || req.Header.Get("X-Trace") != "owned" {
			t.Error(req.Header)
		}
		switch req.URL.Query().Get("marker") {
		case "one":
			return nativeForwardWire(200, `{"port_forwardings":[{"id":"two","name":"second","internal_port":"0022","description":"keep"}]}`)
		case "two":
			return nativeForwardWire(200, `{"port_forwardings":[]}`)
		}
		return nativeForwardWire(200, `{"port_forwardings":[{"id":"one","description":"drop","internal_port":22}],"port_forwardings_links":[{"rel":"next","href":"?marker=one"}]}`)
	})
	protocol := []any{"tcp", nil, false}
	options := []portforwarding.RecordListOption{portforwarding.WithRecordListQuery("protocol", protocol), portforwarding.WithRecordListQuery("external_port", 0), portforwarding.WithRecordListQuery("description", "keep"), portforwarding.WithRecordListQuery("internal_port", 22), portforwarding.WithRecordListQuery("x_unknown", 1), portforwarding.WithRecordListLimit(2), portforwarding.WithRecordListHeaders(map[string]string{"X-Trace": "owned"})}
	protocol[0] = "changed"
	seq := api.ListRecords(ctx, "fip-1", options...)
	if len(calls) != 0 {
		t.Fatal(calls)
	}
	var rows []*portforwarding.PortForwardingRecord
	for row, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if len(rows) != 1 || rows[0].ParentID != "fip-1" || len(calls) != 3 {
		t.Fatal(rows, calls)
	}
	forwardingRecordBody(t, rows[0], "internal_port", "22")
	query, _ := url.ParseQuery(calls[0].query)
	if !reflect.DeepEqual(query, url.Values{"protocol": {"tcp", "False"}, "external_port": {"0"}, "limit": {"2"}}) {
		t.Fatal(query)
	}
	if !strings.Contains(string(rows[0].Envelope), `"id":"two"`) || rows[0].StatusCode != 200 {
		t.Fatal(rows[0])
	}
	// Reiteration is independent; breaking a yield prevents any next HTTP.
	before := len(calls)
	for _, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if len(calls) != before+2 {
		t.Fatal(calls)
	}
}

func TestPythonPortForwardingListCapsBeforeFilterAndRejectsScopeEscape(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name     string
		options  []portforwarding.RecordListOption
		wantRows int
	}{
		{"raw cap", []portforwarding.RecordListOption{portforwarding.WithRecordListMaxItems(1), portforwarding.WithRecordListQuery("description", "keep")}, 0},
		{"single page", []portforwarding.RecordListOption{portforwarding.WithRecordListPaginated(false)}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []nativeForwardCall
			api, _ := nativeForwardAPI(t, &calls, func(*http.Request) *http.Response {
				return nativeForwardWire(200, `{"port_forwardings":[{"id":"one","description":"drop"},{"id":"two","description":"keep"}],"port_forwardings_links":[{"rel":"next","href":"https://escape.invalid/elsewhere"}]}`)
			})
			rows := 0
			for _, err := range api.ListRecords(ctx, "fip-1", tc.options...) {
				if err != nil {
					t.Fatal(err)
				}
				rows++
			}
			if rows != tc.wantRows || len(calls) != 1 {
				t.Fatal(rows, calls)
			}
			if tc.name == "raw cap" {
				q, _ := url.ParseQuery(calls[0].query)
				if q.Get("limit") != "1" {
					t.Fatal(q)
				}
			}
		})
	}
	var calls []nativeForwardCall
	api, _ := nativeForwardAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeForwardWire(200, `{"port_forwardings":[{"id":"one"}],"next":"/neutron/v2.0/floatingips/other/port_forwardings?marker=one"}`)
	})
	var gotErr error
	for _, err := range api.ListRecords(ctx, "fip-1") {
		if err != nil {
			gotErr = err
		}
	}
	var proof *resource.ResponseError
	if !errors.Is(gotErr, resource.ErrInvalidOption) || !errors.As(gotErr, &proof) || proof.StatusCode != 200 || len(calls) != 1 {
		t.Fatal(gotErr, proof, calls)
	}
}

func TestPythonPortForwardingFindKeepsSourceFetchKwargsAndCompleteFallback(t *testing.T) {
	ctx := context.Background()
	var calls []nativeForwardCall
	api, _ := nativeForwardAPI(t, &calls, func(req *http.Request) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/wanted") {
			return nativeForwardWire(403, `{"forbidden":"GET"}`)
		}
		if req.URL.Query().Has("marker") {
			return nativeForwardWire(200, `{"port_forwardings":[]}`)
		}
		return nativeForwardWire(200, `{"port_forwardings":[{"id":"one","name":"wanted","description":"keep"}],"next":"?marker=one"}`)
	})
	row, err := api.FindRecord(ctx, "fip-1", "wanted", portforwarding.WithRecordFindQuery("protocol", []string{"tcp", "udp"}), portforwarding.WithRecordFindQuery("description", "keep"), portforwarding.WithRecordFindQuery("x_unknown", false))
	if err != nil || forwardingRecordID(row) != "one" || len(calls) != 3 {
		t.Fatal(row, err, calls)
	}
	direct, _ := url.ParseQuery(calls[0].query)
	fallback, _ := url.ParseQuery(calls[1].query)
	if !reflect.DeepEqual(direct, url.Values{"floatingip_id": {"fip-1"}, "protocol": {"tcp", "udp"}, "description": {"keep"}, "x_unknown": {"False"}}) || !reflect.DeepEqual(fallback, url.Values{"protocol": {"tcp", "udp"}}) {
		t.Fatal(direct, fallback)
	}
	// A late page failure cannot return the earlier matching row.
	var lateCalls []nativeForwardCall
	late, _ := nativeForwardAPI(t, &lateCalls, func(req *http.Request) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/wanted") {
			return nativeForwardWire(404, `{}`)
		}
		if req.URL.Query().Has("marker") {
			return nativeForwardWire(429, `{"retry":"later"}`)
		}
		return nativeForwardWire(200, `{"port_forwardings":[{"id":"one","name":"wanted"}],"next":"?marker=one"}`)
	})
	row, err = late.FindRecord(ctx, "fip-1", "wanted")
	var native gophercloud.ErrUnexpectedResponseCode
	if row != nil || !errors.As(err, &native) || native.Actual != 429 || len(lateCalls) != 3 {
		t.Fatal(row, err, lateCalls)
	}
}
func forwardingRecordID(row *portforwarding.PortForwardingRecord) string {
	if row == nil || row.Resource == nil {
		return ""
	}
	var id string
	_ = json.Unmarshal(row.Resource.Body["id"], &id)
	return id
}

func TestPythonPortForwardingFindMissingDuplicateAndDirectSuccess(t *testing.T) {
	ctx := context.Background()
	for _, test := range []struct {
		name, body string
		strict     bool
		expected   error
	}{
		{"missing default", `{"port_forwardings":[]}`, false, nil},
		{"missing strict", `{"port_forwardings":[]}`, true, resource.ErrNotFound},
		{"duplicate", `{"port_forwardings":[{"id":"one","name":"wanted"},{"id":"wanted","name":"other"}]}`, false, resource.ErrAmbiguous},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []nativeForwardCall
			api, _ := nativeForwardAPI(t, &calls, func(req *http.Request) *http.Response {
				if strings.HasSuffix(req.URL.Path, "/wanted") {
					return nativeForwardWire(400, `{}`)
				}
				return nativeForwardWire(200, test.body)
			})
			row, err := api.FindRecord(ctx, "fip-1", "wanted", portforwarding.WithRecordFindIgnoreMissing(!test.strict))
			if row != nil || !errors.Is(err, test.expected) || len(calls) != 2 {
				t.Fatal(row, err, calls)
			}
		})
	}
	var calls []nativeForwardCall
	api, _ := nativeForwardAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeForwardWire(200, `{"port_forwarding":{"id":"different","name":"not-wanted"}}`)
	})
	row, err := api.FindRecord(ctx, "fip-1", "wanted")
	if err != nil || forwardingRecordID(row) != "different" || len(calls) != 1 {
		t.Fatal(row, err, calls)
	}
}

type forwardingFailBody struct {
	data              string
	readErr, closeErr error
	close             func()
}

func (b *forwardingFailBody) Read(p []byte) (int, error) {
	if b.data == "" {
		return 0, io.EOF
	}
	n := copy(p, b.data)
	b.data = b.data[n:]
	return n, b.readErr
}
func (b *forwardingFailBody) Close() error {
	if b.close != nil {
		b.close()
	}
	return b.closeErr
}

func TestPythonPortForwardingCleanOnlyFindAndDeleteRejectedBodyCauses(t *testing.T) {
	ctx := context.Background()
	sentinel := errors.New("rejected body handling")
	for _, phase := range []string{"find-read", "find-close", "delete-read", "delete-close", "delete-clean", "delete-strict", "find-route-change"} {
		t.Run(phase, func(t *testing.T) {
			var calls []nativeForwardCall
			var api *portforwarding.API
			api, _ = nativeForwardAPI(t, &calls, func(req *http.Request) *http.Response {
				response := nativeForwardWire(404, `{"missing":"actual body"}`)
				body := &forwardingFailBody{data: `{"missing":"actual body"}`}
				response.Body = body
				if strings.HasSuffix(phase, "read") {
					body.readErr = sentinel
				}
				if strings.HasSuffix(phase, "close") {
					body.closeErr = sentinel
				}
				if phase == "find-route-change" {
					body.close = func() { api.RawClient().ResourceBase = "http://changed.invalid/" }
				}
				return response
			})
			var err error
			if strings.HasPrefix(phase, "find") {
				var row *portforwarding.PortForwardingRecord
				row, err = api.FindRecord(ctx, "fip-1", "pf-1")
				if row != nil {
					t.Fatal(row)
				}
			} else {
				var options []portforwarding.RecordDeleteOption
				if phase == "delete-strict" {
					options = append(options, portforwarding.WithRecordDeleteIgnoreMissing(false))
				}
				err = api.DeleteRecord(ctx, "fip-1", "pf-1", options...)
			}
			if phase == "delete-clean" {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 404 || !strings.Contains(string(native.Body), "actual body") {
					t.Fatal(err, native)
				}
				if strings.HasSuffix(phase, "read") || strings.HasSuffix(phase, "close") {
					if !errors.Is(err, sentinel) {
						t.Fatal(err)
					}
				}
				if phase == "find-route-change" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			}
			if len(calls) != 1 {
				t.Fatal(calls)
			}
		})
	}
}

func TestPythonPortForwardingOptionsPreflightSnapshotsAndCancellation(t *testing.T) {
	ctx := context.Background()
	var calls []nativeForwardCall
	api, _ := nativeForwardAPI(t, &calls, func(*http.Request) *http.Response { return nativeForwardWire(200, `{"port_forwarding":{}}`) })
	for _, err := range []error{
		func() error { _, err := api.CreateRecord(ctx, "fip-1", nil); return err }(),
		func() error {
			_, err := api.UpdateRecord(ctx, "fip-1", "pf-1", portforwarding.WithRecordAttribute("id", "escape"))
			return err
		}(),
		func() error {
			_, err := api.CreateRecord(ctx, "fip-1", portforwarding.WithRecordAttribute("floatingip_id", "escape"))
			return err
		}(),
		func() error { _, err := api.GetRecord(ctx, "..", "pf-1"); return err }(),
		func() error {
			_, err := api.FindRecord(ctx, "fip-1", "pf-1", portforwarding.WithRecordFindQuery("base_path", "/escape"))
			return err
		}(),
		api.DeleteRecord(nil, "fip-1", "pf-1"),
	} {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	for _, err := range api.ListRecords(ctx, "fip-1", portforwarding.WithRecordListLimit(-1)) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
	cancellation := errors.New("caller cause")
	cancelled, cancel := context.WithCancelCause(ctx)
	cancel(cancellation)
	_, err := api.CreateRecord(cancelled, "fip-1")
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cancellation) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	// Options keep independent RawMessage, map and slice snapshots, including
	// custom option retention. The transport cannot alter the sent request.
	raw := json.RawMessage(`{"stable":false}`)
	input := portforwarding.RecordMutationOpts{Fields: map[string]json.RawMessage{"x_owned": raw}}
	bulk := portforwarding.WithRecordMutationOpts(input)
	copy(raw, []byte(`{"changed":true}`))
	delete(input.Fields, "x_owned")
	_, err = api.CreateRecord(ctx, "fip-1", bulk)
	if err != nil || calls[0].body != `{"port_forwarding":{"x_owned":{"stable":false}}}` {
		t.Fatal(err, calls)
	}
	// A custom callback changing the selected route fails before dispatch.
	source := api.RawClient()
	before := len(calls)
	_, err = api.UpdateRecord(ctx, "fip-1", "pf-1", func(*portforwarding.RecordMutationOpts) error {
		source.Endpoint = "http://changed.invalid/"
		return nil
	})
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
		t.Fatal(err, calls)
	}
}

func TestPythonPortForwardingAcceptedResponseCauseAndCleanRetryPolicy(t *testing.T) {
	ctx := context.Background()
	sentinel := errors.New("physical close failed")
	var calls []nativeForwardCall
	api, _ := nativeForwardAPI(t, &calls, func(*http.Request) *http.Response {
		response := nativeForwardWire(200, "")
		response.Body = &forwardingFailBody{data: `{"port_forwarding":{"id":"pf-1"}}`, closeErr: sentinel}
		return response
	})
	_, err := api.GetRecord(ctx, "fip-1", "pf-1")
	var proof *resource.ResponseError
	if !errors.Is(err, sentinel) || !errors.As(err, &proof) || proof.StatusCode != 200 || !strings.Contains(string(proof.Body), "pf-1") || len(calls) != 1 {
		t.Fatal(err, proof, calls)
	}
	// Returning the native clean rejection from RetryFunc leaves it ignorable.
	var retryCalls []nativeForwardCall
	retry, retryCloud := nativeForwardAPI(t, &retryCalls, func(*http.Request) *http.Response { return nativeForwardWire(404, `{"missing":true}`) })
	retryCloud.Provider.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
		return err
	}
	if err := retry.DeleteRecord(ctx, "fip-1", "pf-1"); err != nil || len(retryCalls) != 1 {
		t.Fatal(err, retryCalls)
	}
	retryCloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return sentinel }
	err = retry.DeleteRecord(ctx, "fip-1", "pf-1")
	if !errors.Is(err, sentinel) || len(retryCalls) != 2 {
		t.Fatal(err, retryCalls)
	}
}
