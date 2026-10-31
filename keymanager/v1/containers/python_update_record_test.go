package containers_test

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func containerUpdateFixture(t *testing.T, reply func(*http.Request) *http.Response) (*containers.API, *testcloud.Cloud, *testcloud.Recorder) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := testcloud.NewRecorder(func(req *http.Request) (*http.Response, error) { return reply(req), nil })
	client := nativeContainerClient(cloud)
	cloud.SetTransport(calls)
	return containers.New(client), cloud, calls
}

// containerUpdateCalls projects the recorded calls onto "METHOD escaped-path body" strings.
func containerUpdateCalls(rec *testcloud.Recorder) []string {
	calls := []string{}
	for _, call := range rec.Calls() {
		calls = append(calls, call.Method+" "+call.EscapedPath+" "+call.Body)
	}
	return calls
}
func TestPythonContainerUpdateRecordNoopSeedHasNoReceiptOrPUT(t *testing.T) {
	for _, id := range []string{"s1", ""} {
		for _, attr := range []string{"", "unknown", "location", "microversion", "if_match"} {
			t.Run(id+"/"+attr, func(t *testing.T) {
				api, _, calls := containerUpdateFixture(t, func(*http.Request) *http.Response {
					t.Error("no-op reached HTTP")
					return nativeContainerWire(200, `{}`)
				})
				var opts []containers.UpdateRecordOption
				if attr != "" {
					opts = append(opts, containers.WithUpdateRecordAttribute(attr, "ignored"))
				}
				got, err := api.UpdateRecord(context.Background(), id, opts...)
				if err != nil || got == nil || got.Committed || got.StatusCode != 0 || got.Header != nil || got.Wire != nil || got.Envelope != nil || calls.Len() != 0 || got.Resource == nil || len(got.Request) != 0 {
					t.Fatal(got, err, containerUpdateCalls(calls))
				}
				want, _ := json.Marshal(id)
				if string(got.Resource.Body["id"]) != string(want) || got.Resource.StatusCode != 0 || string(got.Resource.Body["name"]) != "null" {
					t.Fatal(got)
				}
			})
		}
	}
}
func TestPythonContainerUpdateRecordFlatDirtyBodyNullableAndRawValues(t *testing.T) {
	api, _, calls := containerUpdateFixture(t, func(req *http.Request) *http.Response {
		if req.Header.Get("X-Source") != "direct" || req.Header.Get("X-Auth-Token") != "test-token" || req.Header.Get("Content-Type") != "application/json" {
			t.Error(req.Header)
		}
		return nativeContainerWire(202, `{"status":"ACTIVE","name":null,"unknown":"wire-only","self":"drop"}`)
	})
	got, err := api.UpdateRecord(context.Background(), "c1", containers.WithUpdateRecordAttribute("name", "new"), containers.WithUpdateRecordAttribute("secret_refs", false), containers.WithUpdateRecordAttribute("consumers", map[string]any{"n": 9007199254740993}), containers.WithUpdateRecordAttribute("updated_at", nil), containers.WithUpdateRecordAttribute("unknown", "drop"))
	want := `PUT /barbican/v1/containers/c1 {"consumers":{"n":9007199254740993},"name":"new","secret_refs":false,"updated":null}`
	if err != nil || got == nil || !got.Committed || got.StatusCode != 202 || calls.Len() != 1 || containerUpdateCalls(calls)[0] != want || string(got.Request["name"]) != `"new"` || string(got.Resource.Body["name"]) != "null" || string(got.Resource.Body["consumers"]) != `[{"n":9007199254740993}]` || string(got.Resource.Body["secret_refs"]) != `[false]` || string(got.Resource.Body["status"]) != `"ACTIVE"` || got.Resource.Body["unknown"] != nil || string(got.Wire.Body["unknown"]) != `"wire-only"` || got.Header.Get("X-Container-Proof") != "actual" {
		t.Fatal(got, err, containerUpdateCalls(calls))
	}
	if got.Request["id"] != nil || got.Resource.Body["self"] != nil || string(got.Seed["name"]) != `"new"` {
		t.Fatal(got)
	}
	got.Resource.Body["name"][0] = 'X'
	got.Seed["secret_refs"][0] = 'X'
	got.Header.Set("X-Container-Proof", "changed")
	if string(got.Request["secret_refs"]) != "false" || string(got.Wire.Body["name"]) != "null" || got.Wire.Header.Get("X-Container-Proof") != "actual" {
		t.Fatal("aliases", got)
	}
}
func TestPythonContainerUpdateRecordAliasesOrderResponseMergeAndReferenceProjection(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(fmt.Sprint(reverse), func(t *testing.T) {
			body := `{"updated":"wire","updated_at":"local","container_ref":"https://foreign.invalid/containers/new-ref","id":null}`
			if reverse {
				body = `{"updated_at":"local","updated":"wire","container_ref":"https://foreign.invalid/containers/new-ref","id":null}`
			}
			api, _, calls := containerUpdateFixture(t, func(*http.Request) *http.Response { return nativeContainerWire(200, body) })
			opts := []containers.UpdateRecordOption{containers.WithUpdateRecordAttribute("updated", "wire"), containers.WithUpdateRecordAttribute("updated_at", "local")}
			want := "local"
			if reverse {
				opts[0], opts[1] = opts[1], opts[0]
				want = "wire"
			}
			got, err := api.UpdateRecord(context.Background(), "s1", opts...)
			if err != nil || calls.Len() != 1 || containerUpdateCalls(calls)[0] != `PUT /barbican/v1/containers/s1 {"updated":"`+want+`"}` || string(got.Resource.Body["updated_at"]) != `"`+want+`"` || string(got.Resource.Body["id"]) != "null" || string(got.Resource.Body["container_id"]) != `"new-ref"` {
				t.Fatal(got, err, containerUpdateCalls(calls))
			}
		})
	}
}
func TestPythonContainerUpdateRecordEmptyInvalidJSONAndShapeErrorsKeepActualProof(t *testing.T) {
	for _, body := range []string{"", "{bad", `{}`, `[]`, `null`, `false`, `42`, `"text"`, `{"container_ref":"invalid"}`} {
		t.Run(body, func(t *testing.T) {
			api, _, calls := containerUpdateFixture(t, func(*http.Request) *http.Response { return nativeContainerWire(200, body) })
			got, err := api.UpdateRecord(context.Background(), "s1", containers.WithUpdateRecordAttribute("name", "seed"))
			fail := body == `[]` || body == `null` || body == `false` || body == `42` || body == `"text"` || body == `{"container_ref":"invalid"}`
			if got == nil || got.StatusCode != 200 || string(got.Envelope) != body || calls.Len() != 1 || !got.Committed {
				t.Fatal(got, err, containerUpdateCalls(calls))
			}
			if fail {
				var proof *resource.ResponseError
				if err == nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body {
					t.Fatal(got, err, proof)
				}
			} else if err != nil || string(got.Resource.Body["name"]) != `"seed"` {
				t.Fatal(got, err)
			}
		})
	}
}
func TestPythonContainerUpdateRecordOptionsSnapshotBasePathAndSelectedVersion(t *testing.T) {
	attrs := map[string]json.RawMessage{"name": json.RawMessage(`"secret"`)}
	base, mv := "/alternate/containers", "1.2"
	headers := map[string]string{"X-Trace": "owned"}
	option := containers.WithUpdateRecordOpts(containers.UpdateRecordOpts{Attributes: attrs, BasePath: &base, Microversion: &mv, Headers: headers})
	attrs["name"][1] = 'X'
	base = "changed"
	mv = "1.9"
	headers["X-Trace"] = "changed"
	var seen http.Header
	api, _, calls := containerUpdateFixture(t, func(req *http.Request) *http.Response { seen = req.Header.Clone(); return nativeContainerWire(204, "") })
	var retained *containers.UpdateRecordOpts
	capture := func(c *containers.UpdateRecordOpts) error { retained = c; return nil }
	got, err := api.UpdateRecord(context.Background(), "id/with%", option, capture)
	if err != nil || got.StatusCode != 204 || calls.Len() != 1 || containerUpdateCalls(calls)[0] != `PUT /barbican/v1/alternate/containers/id%2Fwith%25 {"name":"secret"}` || seen.Get("X-Trace") != "owned" || seen.Get("OpenStack-API-Version") != "key-manager 1.2" || api.RawClient().Microversion != "" {
		t.Fatal(got, err, containerUpdateCalls(calls), seen)
	}
	retained.Attributes["name"][1] = 'X'
	_, err = api.UpdateRecord(context.Background(), "s1", option)
	if err != nil || !strings.HasSuffix(containerUpdateCalls(calls)[1], `{"name":"secret"}`) {
		t.Fatal(err, containerUpdateCalls(calls))
	}
}
func TestPythonContainerUpdateRecordPreflightAndConstructorCollisions(t *testing.T) {
	for _, lane := range []string{"nil-context", "cancel", "nil-api", "nil-option", "bad-id", "id", "connection", "bad-ref", "bad-base", "auth-header", "source"} {
		t.Run(lane, func(t *testing.T) {
			api, _, calls := containerUpdateFixture(t, func(*http.Request) *http.Response { return nativeContainerWire(200, `{}`) })
			ctx := context.Background()
			id := "s1"
			opts := []containers.UpdateRecordOption{containers.WithUpdateRecordAttribute("name", "n")}
			switch lane {
			case "nil-context":
				ctx = nil
			case "cancel":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "nil-api":
				api = nil
			case "nil-option":
				opts = append(opts, nil)
			case "bad-id":
				id = ".."
			case "id", "connection":
				opts = append(opts, containers.WithUpdateRecordAttribute(lane, "collision"))
			case "bad-ref":
				opts = append(opts, containers.WithUpdateRecordAttribute("container_ref", "invalid"))
			case "bad-base":
				opts = append(opts, containers.WithUpdateRecordBasePath("../escape"))
			case "auth-header":
				opts = append(opts, containers.WithUpdateRecordHeaders(map[string]string{"Authorization": "foreign"}))
			case "source":
				opts = append(opts, func(*containers.UpdateRecordOpts) error { api.RawClient().ResourceBase += "changed/"; return nil })
			}
			_, err := api.UpdateRecord(ctx, id, opts...)
			if err == nil || calls.Len() != 0 {
				t.Fatal(err, containerUpdateCalls(calls))
			}
		})
	}
}
func TestPythonContainerUpdateRecordAcceptedAndRejectedStatusPolicy(t *testing.T) {
	for _, status := range []int{199, 200, 202, 204, 300, 399, 400, 401, 403, 404, 409, 500, 777} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			api, _, calls := containerUpdateFixture(t, func(*http.Request) *http.Response { return nativeContainerWire(status, `{"name":"reply"}`) })
			got, err := api.UpdateRecord(context.Background(), "s1", containers.WithUpdateRecordAttribute("name", "n"))
			if got == nil || got.StatusCode != status || calls.Len() != 1 || got.Header.Get("X-Container-Proof") != "actual" {
				t.Fatal(got, err, containerUpdateCalls(calls))
			}
			if status < 400 {
				if err != nil || !got.Committed {
					t.Fatal(got, err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if err == nil || !errors.As(err, &native) || native.Actual != status || got.Committed {
					t.Fatal(got, err, native)
				}
			}
		})
	}
}
func TestPythonContainerUpdateRecordPhysicalFailuresNeverReplayAndRetainNativeProof(t *testing.T) {
	for _, status := range []int{200, 400, 401, 429, 503, 777} {
		for _, lane := range []string{"read-close", "cancel", "source"} {
			t.Run(fmt.Sprintf("%d/%s", status, lane), func(t *testing.T) {
				readCause, closeCause, cancelCause := errors.New("read"), errors.New("close"), errors.New("cancel")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				physical := &containerUpdateFaultBody{data: []byte(`{"name":"actual"}`)}
				api, cloud, calls := containerUpdateFixture(t, func(*http.Request) *http.Response {
					return testcloud.StreamResponse(status, physical, http.Header{"X-Actual": {"physical"}})
				})
				client := api.RawClient()
				base := client.ResourceBase
				if lane == "read-close" {
					physical.readErr = readCause
					physical.closeErr = closeCause
				} else if lane == "cancel" {
					physical.read = func() { cancel(cancelCause) }
				} else {
					physical.read = func() { client.ResourceBase += "wrong/" }
					physical.close = func() { client.ResourceBase = base }
				}
				retries, auth, backoff := 0, 0, 0
				sentinel := errors.New("unexpected callback")
				cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return sentinel
				}
				cloud.Provider.ReauthFunc = func(context.Context) error { auth++; return sentinel }
				cloud.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
					backoff++
					return sentinel
				}
				got, err := api.UpdateRecord(ctx, "s1", containers.WithUpdateRecordAttribute("name", "n"))
				if err == nil || got == nil || got.StatusCode != status || string(got.Envelope) != `{"name":"actual"}` || got.Header.Get("X-Actual") != "physical" || calls.Len() != 1 || retries != 0 || auth != 0 || backoff != 0 || got.Committed {
					t.Fatal(got, err, containerUpdateCalls(calls), retries, auth, backoff)
				}
				if lane == "read-close" && (!errors.Is(err, readCause) || !errors.Is(err, closeCause)) || lane == "cancel" && !errors.Is(err, cancelCause) || lane == "source" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			})
		}
	}
}
func TestPythonContainerUpdateRecordCleanRetryAndFailedReauthPreserveBody(t *testing.T) {
	for _, mutate := range []bool{false, true} {
		t.Run(fmt.Sprint(mutate), func(t *testing.T) {
			attempts := 0
			api, cloud, calls := containerUpdateFixture(t, func(*http.Request) *http.Response {
				attempts++
				if attempts == 1 {
					return nativeContainerWire(409, `{"original":true}`)
				}
				return nativeContainerWire(200, `{}`)
			})
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				if mutate {
					o.JSONBody = json.RawMessage(`{"name":"other"}`)
				}
				return nil
			}
			got, err := api.UpdateRecord(context.Background(), "s1", containers.WithUpdateRecordAttribute("name", "owned"))
			if mutate {
				if err == nil || attempts != 1 || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(got, err, containerUpdateCalls(calls))
				}
			} else if err != nil || attempts != 2 || !reflect.DeepEqual(containerUpdateCalls(calls)[0], containerUpdateCalls(calls)[1]) {
				t.Fatal(got, err, containerUpdateCalls(calls))
			}
		})
	}
	t.Run("failed reauth", func(t *testing.T) {
		api, cloud, calls := containerUpdateFixture(t, func(*http.Request) *http.Response { return nativeContainerWire(401, `{"original":true}`) })
		cause := errors.New("auth unavailable")
		cloud.Provider.ReauthFunc = func(context.Context) error { return cause }
		got, err := api.UpdateRecord(context.Background(), "s1", containers.WithUpdateRecordAttribute("name", "n"))
		var unable *gophercloud.ErrUnableToReauthenticate
		if !errors.As(err, &unable) || unable.ErrReauth != cause || got.StatusCode != 401 || string(got.Envelope) != `{"original":true}` || calls.Len() != 1 {
			t.Fatal(got, err, unable, containerUpdateCalls(calls))
		}
	})
}
func TestPythonContainerUpdateRecordRequestsCharsetGapIsExplicit(t *testing.T) {
	body := string([]byte{'{', '"', 'n', 'a', 'm', 'e', '"', ':', '"', 'c', 'a', 'f', 0xe9, '"', '}'})
	api, _, calls := containerUpdateFixture(t, func(*http.Request) *http.Response {
		r := nativeContainerWire(200, body)
		r.Header.Set("Content-Type", "application/json; charset=cp1006")
		return r
	})
	got, err := api.UpdateRecord(context.Background(), "s1", containers.WithUpdateRecordAttribute("name", "seed"))
	var proof *resource.ResponseError
	if !errors.Is(err, resource.ErrUnsupported) || !errors.As(err, &proof) || got.StatusCode != 200 || string(got.Envelope) != body || !got.Committed || calls.Len() != 1 {
		t.Fatal(got, err, proof, containerUpdateCalls(calls))
	}
}

func containerUpdateCharsetJSON(text string, width int, order binary.ByteOrder, bom []byte) []byte {
	data := append([]byte(nil), bom...)
	if width == 2 {
		for _, v := range utf16.Encode([]rune(text)) {
			var b [2]byte
			order.PutUint16(b[:], v)
			data = append(data, b[:]...)
		}
	} else {
		for _, v := range text {
			var b [4]byte
			order.PutUint32(b[:], uint32(v))
			data = append(data, b[:]...)
		}
	}
	return data
}
func TestPythonContainerUpdateRecordExplicitCharsetsDecodeViewsKeepPhysicalEnvelope(t *testing.T) {
	cases := []struct {
		name, charset string
		body          []byte
	}{
		{"latin1", "ISO-8859-1", []byte{'{', '"', 'n', 'a', 'm', 'e', '"', ':', '"', 'c', 'a', 'f', 0xe9, '"', '}'}},
		{"utf16-bom", "utf-16", containerUpdateCharsetJSON(`{"name":"café"}`, 2, binary.LittleEndian, []byte{0xff, 0xfe})},
		{"utf32be", "utf-32-be", containerUpdateCharsetJSON(`{"name":"café"}`, 4, binary.BigEndian, nil)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			contentType := "application/json; charset=" + tc.charset
			api, _, calls := containerUpdateFixture(t, func(*http.Request) *http.Response {
				r := nativeContainerWire(200, string(tc.body))
				r.Header.Set("Content-Type", contentType)
				return r
			})
			got, err := api.UpdateRecord(context.Background(), "s1", containers.WithUpdateRecordAttribute("name", "seed"))
			if err != nil || got == nil || got.Resource == nil || got.Wire == nil || !got.Committed || got.StatusCode != 200 || string(got.Envelope) != string(tc.body) || got.Header.Get("Content-Type") != contentType || calls.Len() != 1 {
				t.Fatal(got, err, containerUpdateCalls(calls))
			}
			for _, view := range []*resource.RawResource{got.Resource, got.Wire} {
				var name string
				if json.Unmarshal(view.Body["name"], &name) != nil || name != "café" || view.Header.Get("Content-Type") != contentType || view.StatusCode != 200 {
					t.Fatal(view)
				}
			}
			got.Wire.Body["name"][1] = 'X'
			var name string
			if json.Unmarshal(got.Resource.Body["name"], &name) != nil || name != "café" || string(got.Envelope) != string(tc.body) {
				t.Fatal("normalized views/raw envelope alias", got)
			}
		})
	}
}
func TestPythonContainerUpdateRecordDecodedShapeFailureKeepsOriginalEncodedProof(t *testing.T) {
	body := containerUpdateCharsetJSON(`[]`, 2, binary.LittleEndian, []byte{0xff, 0xfe})
	contentType := "application/json; charset=utf-16"
	api, _, calls := containerUpdateFixture(t, func(*http.Request) *http.Response {
		r := nativeContainerWire(200, string(body))
		r.Header.Set("Content-Type", contentType)
		return r
	})
	got, err := api.UpdateRecord(context.Background(), "s1", containers.WithUpdateRecordAttribute("name", "seed"))
	var proof *resource.ResponseError
	if err == nil || !errors.As(err, &proof) || got == nil || !got.Committed || got.StatusCode != 200 || string(got.Envelope) != string(body) || proof.StatusCode != 200 || string(proof.Body) != string(body) || proof.Header.Get("Content-Type") != contentType || got.Wire != nil || calls.Len() != 1 {
		t.Fatal(got, err, proof, containerUpdateCalls(calls))
	}
	proof.Body[0] = 0
	proof.Header.Set("Content-Type", "changed")
	if string(got.Envelope) != string(body) || got.Header.Get("Content-Type") != contentType {
		t.Fatal("error proof/raw receipt alias", got)
	}
}

type containerUpdateFaultBody struct {
	data              []byte
	readErr, closeErr error
	read, close       func()
}

func (b *containerUpdateFaultBody) Read(p []byte) (int, error) {
	if b.read != nil {
		b.read()
	}
	n := copy(p, b.data)
	b.data = b.data[n:]
	if len(b.data) == 0 {
		if b.readErr != nil {
			return n, b.readErr
		}
		return n, io.EOF
	}
	return n, nil
}
func (b *containerUpdateFaultBody) Close() error {
	if b.close != nil {
		b.close()
	}
	return b.closeErr
}
