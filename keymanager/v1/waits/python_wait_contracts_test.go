package waits_test

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
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/waits"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func wire(status int, body string) *http.Response {
	return testcloud.Response(status, body, http.Header{"X-Request-Id": []string{"wait-proof"}})
}
func fixture(t *testing.T, reply func(*http.Request, int) *http.Response) (*waits.API, *gophercloud.ServiceClient, *[]string) {
	t.Helper()
	cloud := testcloud.New(t)
	client := cloud.Client("key-manager", "/barbican/v1/")
	calls := &[]string{}
	cloud.SetTransport(testcloud.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, req.Method+" "+req.URL.EscapedPath())
		if req.Method != "GET" || req.URL.RawQuery != "" || req.Header.Get("X-Auth-Token") == "" {
			t.Fatal(req)
		}
		return reply(req, len(*calls)), nil
	}))
	return waits.New(client), client, calls
}
func raw(t *testing.T, body string) *resource.RawResource {
	t.Helper()
	var v resource.RawResource
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		t.Fatal(err)
	}
	return &v
}
func str(v string) *string { return &v }
func field(t *testing.T, r *waits.Record, key, want string) {
	t.Helper()
	if r == nil || r.Resource == nil || string(r.Resource.Body[key]) != want {
		t.Fatal(key, want, r)
	}
}

func TestPythonKeymanagerWaitCachedStatesAndIndependentTargetSnapshot(t *testing.T) {
	for _, state := range []string{"ACTIVE", "", "İ", "ΟΣ"} {
		t.Run(state, func(t *testing.T) {
			api, _, calls := fixture(t, func(*http.Request, int) *http.Response { t.Fatal("cached success fetched"); return nil })
			body, _ := json.Marshal(map[string]any{"id": "response-id", "status": state, "container_ref": "https://passive.example/c", "name": "retained"})
			view := raw(t, string(body))
			wireView := view.Clone()
			target := waits.ForContainer(&containers.FetchedContainer{RequestID: "fixed/id", Resource: view, Wire: wireView, Envelope: body, Header: http.Header{"X-Original": []string{"yes"}}, StatusCode: 203})
			view.Body["status"] = json.RawMessage(`"changed"`)
			wireView.Body["name"][0] = 'x'
			body[0] = 'x'
			want := state
			switch state {
			case "ACTIVE":
				want = "active"
			case "İ":
				want = "i\u0307"
			case "ΟΣ":
				want = "ος"
			}
			value, err := api.WaitForStatus(context.Background(), target, &want, waits.WithTimeout(0), waits.WithInterval(-time.Second))
			if err != nil || len(*calls) != 0 || value.RequestID != "fixed/id" || value.StatusCode != 203 || value.Header.Get("X-Original") != "yes" {
				t.Fatal(value, err, *calls)
			}
			original, _ := json.Marshal(state)
			field(t, value, "status", string(original))
			value.Resource.Body["status"][0] = 'x'
			value.Header.Set("X-Original", "changed")
			again, err := api.WaitForStatus(context.Background(), target, &want)
			if err != nil || string(again.Resource.Body["status"]) != string(original) || again.Header.Get("X-Original") != "yes" {
				t.Fatal(again, err)
			}
		})
	}
	api, _, calls := fixture(t, func(*http.Request, int) *http.Response { t.Fatal("null cached state fetched"); return nil })
	value, err := api.WaitForStatus(context.Background(), waits.Secret("s"), nil)
	if err != nil || len(*calls) != 0 || value.StatusCode != 0 {
		t.Fatal(value, err, *calls)
	}
	field(t, value, "status", `null`)
}

func TestPythonKeymanagerWaitNormalizedComparisonIsNotUnicodeEqualFold(t *testing.T) {
	api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(200, `{"status":"i"}`) })
	target := waits.ForOrder(&orders.FetchedOrder{RequestID: "o", Resource: raw(t, `{"status":"İ"}`)})
	value, err := api.WaitForStatus(context.Background(), target, str("i"))
	if err != nil || len(*calls) != 1 {
		t.Fatal(value, err, *calls)
	}
	field(t, value, "status", `"i"`)
}

func TestPythonKeymanagerWaitSynchronizedSeedMergeAndGenericMalformedBody(t *testing.T) {
	bodies := []string{`{"status":"BUILD","created":null,"x_unknown":{"n":9007199254740993}}`, `not JSON`, `{"status":"ACTIVE","id":null,"updated":false}`}
	api, _, calls := fixture(t, func(req *http.Request, n int) *http.Response {
		if req.URL.EscapedPath() != "/barbican/v1/orders/fixed%2Fid" {
			t.Fatal(req.URL)
		}
		return wire(202, bodies[n-1])
	})
	target := waits.ForOrder(&orders.FetchedOrder{RequestID: "fixed/id", Resource: raw(t, `{"status":"BUILD","name":"original","created_at":"old","meta":{"x":1},"id":"seed"}`)})
	progress := []int{}
	value, err := api.WaitForStatus(context.Background(), target, str("active"), waits.WithInterval(time.Nanosecond), waits.WithProgressCallback(func(v int) error { progress = append(progress, v); return nil }))
	if err != nil || len(*calls) != 3 || !reflect.DeepEqual(progress, []int{0, 0}) || value.StatusCode != 202 || string(value.Envelope) != bodies[2] {
		t.Fatal(value, err, *calls, progress)
	}
	field(t, value, "name", `"original"`)
	field(t, value, "created_at", `null`)
	field(t, value, "updated_at", `false`)
	field(t, value, "id", `null`)
	field(t, value, "meta", `{"x":1}`)
	if value.Resource.Body["x_unknown"] != nil || value.Wire.Body["name"] != nil {
		t.Fatal(value)
	}
}

func TestPythonKeymanagerWaitNullableAndInvalidSelectedAttributesKeepProof(t *testing.T) {
	for _, body := range []string{`{"status":false}`, `{"status":0}`, `null`, `[]`} {
		api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(200, body) })
		value, err := api.WaitForStatus(context.Background(), waits.Container("c"), str("active"))
		var proof *resource.ResponseError
		if value == nil || value.Resource != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || len(*calls) != 1 {
			t.Fatal(value, err, proof, *calls)
		}
	}
	api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(200, `{"status":null}`) })
	target := waits.ForOrder(&orders.FetchedOrder{RequestID: "o", Resource: raw(t, `{"status":"BUILD"}`)})
	value, err := api.WaitForStatus(context.Background(), target, nil)
	if err != nil || len(*calls) != 1 {
		t.Fatal(value, err, *calls)
	}
	field(t, value, "status", `null`)
	api, _, calls = fixture(t, func(*http.Request, int) *http.Response {
		t.Fatal("invalid initial selected attribute fetched")
		return nil
	})
	target = waits.ForOrder(&orders.FetchedOrder{RequestID: "o", Resource: raw(t, `{"status":false}`)})
	_, err = api.WaitForStatus(context.Background(), target, str("active"))
	if !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
		t.Fatal(err, *calls)
	}
}

func TestPythonKeymanagerWaitFailureDefaultsExplicitEmptyAndCustomAttribute(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		api, _, calls := fixture(t, func(_ *http.Request, n int) *http.Response {
			if n == 1 {
				return wire(200, `{"status":"eRrOr","progress":999}`)
			}
			return wire(200, `{"status":"ACTIVE"}`)
		})
		options := []waits.Option{waits.WithInterval(time.Nanosecond)}
		progress := []int{}
		options = append(options, waits.WithProgressCallback(func(v int) error { progress = append(progress, v); return nil }))
		if disabled {
			options = append(options, waits.WithFailures())
		}
		value, err := api.WaitForStatus(context.Background(), waits.Secret("s"), str("active"), options...)
		if disabled {
			if err != nil || len(*calls) != 2 || !reflect.DeepEqual(progress, []int{0}) {
				t.Fatal(value, err, *calls, progress)
			}
		} else {
			var proof *resource.ResponseError
			if value == nil || value.Resource != nil || !errors.Is(err, resource.ErrFailedState) || !errors.As(err, &proof) || len(*calls) != 1 || len(progress) != 0 {
				t.Fatal(value, err, *calls, progress)
			}
		}
	}
	api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(200, `{"sub_status":"Ready"}`) })
	value, err := api.WaitForStatus(context.Background(), waits.Order("o"), str("ready"), waits.WithAttribute("sub_status"))
	if err != nil || len(*calls) != 1 {
		t.Fatal(value, err, *calls)
	}
	field(t, value, "status", `null`)
	field(t, value, "sub_status", `"Ready"`)
}

func TestPythonKeymanagerWaitOptionOwnershipAndPreflight(t *testing.T) {
	for _, tc := range []struct {
		target  waits.Target
		options []waits.Option
		want    error
	}{
		{waits.Target{}, nil, resource.ErrInvalidOption}, {waits.Secret(""), nil, resource.ErrInvalidOption}, {waits.Order("o"), []waits.Option{nil}, resource.ErrInvalidOption},
		{waits.Container("c"), []waits.Option{waits.WithAttribute("unknown")}, resource.ErrUnsupported},
		{waits.Order("o"), []waits.Option{waits.WithHeader("Authorization", "forged")}, resource.ErrInvalidOption},
		{waits.Order("o"), []waits.Option{waits.WithMicroversion("bad\nversion")}, resource.ErrInvalidOption},
		{waits.Order("o"), []waits.Option{waits.WithInterval(-time.Second)}, resource.ErrInvalidOption},
		{waits.Order("o"), []waits.Option{waits.WithTimeout(0)}, context.DeadlineExceeded},
	} {
		api, _, calls := fixture(t, func(*http.Request, int) *http.Response { t.Fatal("invalid option fetched"); return nil })
		_, err := api.WaitForStatus(context.Background(), tc.target, str("active"), tc.options...)
		if !errors.Is(err, tc.want) || len(*calls) != 0 {
			t.Fatal(err, tc.want, *calls)
		}
	}
	interval, timeout := time.Nanosecond, time.Second
	attribute, version := "status", "1.1"
	headers := map[string]string{"X-Option": "original"}
	option := waits.WithOptions(waits.Options{Interval: &interval, Timeout: &timeout, Attribute: &attribute, Microversion: &version, Headers: headers})
	interval = -1
	timeout = 0
	attribute = "unknown"
	version = "changed"
	headers["X-Option"] = "changed"
	api, client, calls := fixture(t, func(req *http.Request, n int) *http.Response {
		if req.Header.Get("X-Option") != "original" || req.Header.Get("OpenStack-API-Version") != "key-manager 1.1" {
			t.Fatal(req.Header)
		}
		return wire(200, `{"status":"ACTIVE"}`)
	})
	client.MoreHeaders = map[string]string{"aCcEpT": "configured"}
	value, err := api.WaitForStatus(context.Background(), waits.Container("c"), str("active"), option)
	if err != nil || len(*calls) != 1 || client.Microversion != "" || client.MoreHeaders["aCcEpT"] != "configured" {
		t.Fatal(value, err, *calls, client)
	}
	*calls = nil
	_, err = api.WaitForStatus(context.Background(), waits.Order("o"), str("active"), func(*waits.Options) error { client.ResourceBase = client.Endpoint + "changed/"; return nil })
	if !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
		t.Fatal(err, *calls)
	}
}

func TestPythonKeymanagerWaitDeletionOriginalAndFetchedReturnContracts(t *testing.T) {
	for _, lane := range []string{"clean404", "fetched-deleted", "cached-deleted"} {
		body := `{"error":"gone"}`
		status := 404
		if lane != "clean404" {
			body = `{"status":"DeLeTeD","name":"new"}`
			status = 200
		}
		api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(status, body) })
		seed := `{"status":"BUILD","name":"original"}`
		if lane == "cached-deleted" {
			seed = `{"status":"deleted","name":"original"}`
		}
		value, err := api.WaitForDelete(context.Background(), waits.ForContainer(&containers.FetchedContainer{RequestID: "c", Resource: raw(t, seed)}))
		if err != nil || len(*calls) != 1 || value.StatusCode != status || string(value.Envelope) != body {
			t.Fatal(value, err, *calls)
		}
		if lane == "clean404" {
			field(t, value, "name", `"original"`)
			field(t, value, "status", `"BUILD"`)
			if value.Resource.StatusCode != 0 || string(value.Wire.Body["error"]) != `"gone"` {
				t.Fatal(value)
			}
		} else {
			field(t, value, "name", `"new"`)
			field(t, value, "status", `"DeLeTeD"`)
		}
	}
	for _, status := range []int{200, 403, 503} {
		api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(status, `{"status":null}`) })
		value, err := api.WaitForDelete(context.Background(), waits.Order("o"))
		if err == nil || value == nil || value.Resource != nil || len(*calls) != 1 {
			t.Fatal(value, err, *calls)
		}
	}
}

func TestPythonKeymanagerSecretWaitPayloadSelectionUsesPreviousSynchronizedResource(t *testing.T) {
	var accepts []string
	api, _, calls := fixture(t, func(req *http.Request, n int) *http.Response {
		if strings.HasSuffix(req.URL.Path, "/payload") {
			accepts = append(accepts, req.Header.Get("Accept"))
			if n == 2 {
				return wire(200, "text payload")
			}
			return wire(200, string([]byte{0, 255, 1}))
		}
		state := "BUILD"
		if n == 3 {
			state = "ACTIVE"
		}
		return wire(200, `{"status":"`+state+`","id":"returned/id","secret_ref":"https://passive.example/v1/secrets/returned","payload_content_type":"application/octet-stream","content_types":{"default":"ignored"}}`)
	})
	target := waits.ForSecret(&secrets.FoundSecretRecord{RequestID: "fixed/id", Resource: raw(t, `{"status":"BUILD","payload_content_type":"text/plain","name":"original"}`)})
	value, err := api.WaitForStatus(context.Background(), target, str("active"), waits.WithInterval(time.Nanosecond))
	if err != nil || len(*calls) != 4 || !reflect.DeepEqual(accepts, []string{"text/plain", "application/octet-stream"}) || value.Payload == nil || !reflect.DeepEqual(value.Payload.Body, []byte{0, 255, 1}) || value.Payload.Text != nil {
		t.Fatal(value, err, *calls, accepts)
	}
	for _, call := range *calls {
		if !strings.Contains(call, "/secrets/fixed%2Fid") {
			t.Fatal(*calls)
		}
	}
	field(t, value, "payload", `"AP8B"`)
	field(t, value, "name", `"original"`)
	field(t, value, "id", `"returned/id"`)
	value.Resource.Body["payload"][0] = 'x'
	value.Payload.Body[0] = 'x'
	if string(value.Wire.Body["payload_content_type"]) != `"application/octet-stream"` || value.Wire.Body["payload"] != nil {
		t.Fatal(value)
	}
}

func TestPythonKeymanagerSecretWaitMetadataDefaultPayloadAndStrictFailures(t *testing.T) {
	for _, body := range []string{`{"status":"ACTIVE","payload_content_type":"text/plain"}`, `{"status":"ACTIVE","content_types":{"default":null}}`} {
		api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(200, body) })
		value, err := api.WaitForStatus(context.Background(), waits.Secret("s"), str("active"))
		if err != nil || len(*calls) != 1 || value.Payload != nil {
			t.Fatal(value, err, *calls)
		}
	}
	for _, body := range []string{`not JSON`, `{"status":"ACTIVE","content_types":null}`, `{"status":"ACTIVE","content_types":{}}`, `{"status":"ACTIVE","content_types":{"default":false}}`} {
		api, _, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(200, body) })
		value, err := api.WaitForStatus(context.Background(), waits.Secret("s"), str("active"))
		var proof *resource.ResponseError
		if value == nil || value.Resource != nil || !errors.As(err, &proof) || string(proof.Body) != body || len(*calls) != 1 {
			t.Fatal(value, err, *calls)
		}
	}
	for _, deleting := range []bool{false, true} {
		api, _, calls := fixture(t, func(req *http.Request, n int) *http.Response {
			if n == 1 {
				return wire(200, `{"status":"ACTIVE","content_types":{"default":"text/plain"}}`)
			}
			if req.Header.Get("Accept") != "text/plain" {
				t.Fatal(req.Header)
			}
			return wire(404, `payload missing`)
		})
		var value *waits.Record
		var err error
		if deleting {
			value, err = api.WaitForDelete(context.Background(), waits.Secret("s"))
		} else {
			value, err = api.WaitForStatus(context.Background(), waits.Secret("s"), str("active"))
		}
		var native gophercloud.ErrUnexpectedResponseCode
		if value == nil || value.Resource != nil || value.StatusCode != 200 || value.Payload == nil || value.Payload.StatusCode != 404 || string(value.Payload.Body) != `payload missing` || !errors.As(err, &native) || native.Actual != 404 || len(*calls) != 2 {
			t.Fatal(value, err, native, *calls)
		}
	}
}

func TestPythonKeymanagerWaitDefaultTimeoutsAndZeroIntervalPolicy(t *testing.T) {
	for _, deleting := range []bool{false, true} {
		api, _, calls := fixture(t, func(req *http.Request, n int) *http.Response {
			deadline, has := req.Context().Deadline()
			if deleting {
				if !has || time.Until(deadline) < 119*time.Second || time.Until(deadline) > 121*time.Second {
					t.Fatal(deadline, has)
				}
			} else if has {
				t.Fatal("status unexpectedly bounded", deadline)
			}
			state := "ACTIVE"
			if deleting {
				state = "deleted"
			}
			return wire(200, `{"status":"`+state+`"}`)
		})
		var err error
		if deleting {
			_, err = api.WaitForDelete(context.Background(), waits.Order("o"), waits.WithInterval(0))
		} else {
			_, err = api.WaitForStatus(context.Background(), waits.Order("o"), str("active"), waits.WithInterval(0))
		}
		if err != nil || len(*calls) != 1 {
			t.Fatal(err, *calls)
		}
	}
	// Establish an accepted observation before allowing the SDK deadline to
	// end the wait. The callback synchronizes on the actual polling context,
	// rather than assuming a first HTTP read finishes in a tiny race budget.
	var polling context.Context
	api, _, calls := fixture(t, func(req *http.Request, _ int) *http.Response {
		polling = req.Context()
		return wire(200, `{"status":"BUILD"}`)
	})
	value, err := api.WaitForStatus(context.Background(), waits.Container("c"), str("active"), waits.WithInterval(0), waits.WithTimeout(2*time.Second), waits.WithProgressCallback(func(progress int) error {
		if progress != 0 || polling == nil || polling.Err() != nil {
			t.Fatal("first observation did not reach callback before deadline", progress, polling)
		}
		<-polling.Done()
		return nil
	}))
	var proof *resource.ResponseError
	if value == nil || value.Resource != nil || !errors.Is(err, context.DeadlineExceeded) || !errors.As(err, &proof) || proof.StatusCode != 200 || len(*calls) != 1 {
		t.Fatal(value, err, proof, *calls)
	}
}

func TestPythonKeymanagerWaitCallbackFailureCancellationAndSourceAreTerminal(t *testing.T) {
	for _, lane := range []string{"callback", "cancel", "source"} {
		api, client, calls := fixture(t, func(*http.Request, int) *http.Response { return wire(200, `{"status":"BUILD","progress":999}`) })
		callbackErr, cancelCause := errors.New("callback failed"), errors.New("callback cancelled")
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		value, err := api.WaitForStatus(ctx, waits.Container("c"), str("active"), waits.WithProgressCallback(func(progress int) error {
			if progress != 0 {
				t.Fatal(progress)
			}
			switch lane {
			case "callback":
				return callbackErr
			case "cancel":
				cancel(cancelCause)
			case "source":
				client.Endpoint += "changed/"
			}
			return nil
		}))
		var proof *resource.ResponseError
		if value == nil || value.Resource != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-Id") != "wait-proof" || len(*calls) != 1 {
			t.Fatal(value, err, proof, *calls)
		}
		switch lane {
		case "callback":
			if !errors.Is(err, callbackErr) {
				t.Fatal(err)
			}
		case "cancel":
			if !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		case "source":
			if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		}
	}
}

type closingBody struct {
	io.Reader
	close func() error
}

func (b *closingBody) Close() error { return b.close() }

type hookedBody struct {
	io.ReadCloser
	after    func()
	terminal error
	done     bool
}

func (b *hookedBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err == io.EOF && !b.done {
		b.done = true
		b.after()
		return n, b.terminal
	}
	return n, err
}

func TestPythonKeymanagerWaitCompoundPhysicalFaultsRetainActualProofAndNeverReplay(t *testing.T) {
	for _, lane := range []string{"status", "delete", "payload"} {
		for _, status := range []int{200, 404, 429, 503} {
			t.Run(fmt.Sprintf("%s/%d", lane, status), func(t *testing.T) {
				readCause, closeCause, cancelCause := errors.New("read failed"), errors.New("close failed"), errors.New("caller cancel in Close")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				var client *gophercloud.ServiceClient
				originalBase := ""
				closes, retries := 0, 0
				body := `{"status":"ACTIVE"}`
				if lane == "delete" {
					body = `{"status":"deleted"}`
				}
				if lane == "payload" {
					body = "payload bytes"
				}
				api, selected, calls := fixture(t, func(_ *http.Request, n int) *http.Response {
					if lane == "payload" && n == 1 {
						return wire(200, `{"status":"ACTIVE","content_types":{"default":"text/plain"}}`)
					}
					response := wire(status, body)
					response.Body = &hookedBody{ReadCloser: &closingBody{Reader: strings.NewReader(body), close: func() error { closes++; client.ResourceBase = originalBase; cancel(cancelCause); return closeCause }}, after: func() { client.ResourceBase = client.Endpoint + "changed/" }, terminal: readCause}
					return response
				})
				client = selected
				originalBase = client.ResourceBase
				client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				var value *waits.Record
				var err error
				switch lane {
				case "status":
					value, err = api.WaitForStatus(ctx, waits.Order("o"), str("active"))
				case "delete":
					value, err = api.WaitForDelete(ctx, waits.Order("o"))
				case "payload":
					value, err = api.WaitForStatus(ctx, waits.Secret("s"), str("active"))
				}
				wantCalls := 1
				if lane == "payload" {
					wantCalls = 2
				}
				if value == nil || value.Resource != nil || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) || len(*calls) != wantCalls || closes != 1 || retries != 0 {
					t.Fatal(value, err, *calls, closes, retries)
				}
				if status < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "wait-proof" {
						t.Fatal(err, proof)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "wait-proof" {
						t.Fatal(err, native)
					}
				}
			})
		}
	}
}

func TestPythonKeymanagerWaitCleanNativeRetryPreservesSelectedPolicyAndLiveToken(t *testing.T) {
	api, client, calls := fixture(t, func(req *http.Request, n int) *http.Response {
		if req.Header.Get("OpenStack-API-Version") != "key-manager 1.1" || req.Header.Get("X-Selected") != "owned" {
			t.Fatal(req.Header)
		}
		if n == 1 {
			return wire(503, `{"error":"temporary"}`)
		}
		if req.Header.Get("X-Retry") != "native" || req.Header.Get("X-Auth-Token") != "live-token" {
			t.Fatal(req.Header)
		}
		return wire(200, `{"status":"ACTIVE"}`)
	})
	client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, options *gophercloud.RequestOpts, original error, _ uint) error {
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(original, &native) || native.Actual != 503 {
			t.Fatal(original)
		}
		options.MoreHeaders["X-Retry"] = "native"
		client.ProviderClient.SetToken("live-token")
		return nil
	}
	value, err := api.WaitForStatus(context.Background(), waits.Order("o"), str("active"), waits.WithMicroversion("1.1"), waits.WithHeader("X-Selected", "owned"))
	if err != nil || len(*calls) != 2 || value.StatusCode != 200 || client.Microversion != "" {
		t.Fatal(value, err, *calls)
	}
}
