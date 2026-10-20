package groups_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type groupFaultBody struct {
	text                  string
	readErr, closeErr     error
	afterRead, afterClose func()
	closed                int
	read                  bool
}

func (b *groupFaultBody) Read(p []byte) (int, error) {
	if b.read {
		return 0, io.EOF
	}
	b.read = true
	n := copy(p, b.text)
	if b.afterRead != nil {
		b.afterRead()
	}
	if b.readErr != nil {
		return n, b.readErr
	}
	return n, io.EOF
}
func (b *groupFaultBody) Close() error {
	b.closed++
	if b.afterClose != nil {
		b.afterClose()
	}
	return b.closeErr
}

func TestPythonCinderGroupPhysicalAcceptedRejectedAndCancellationProof(t *testing.T) {
	for _, code := range []int{200, 404} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			readCause, closeCause := errors.New("read actual"), errors.New("close actual")
			faulty := &groupFaultBody{text: `{"partial":`, readErr: readCause, closeErr: closeCause}
			api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { r := groupWire(code, ""); r.Body = faulty; return r })
			v, err := api.GetRecord(context.Background(), "id")
			if !errors.Is(err, readCause) || !errors.Is(err, closeCause) || len(*calls) != 1 || faulty.closed != 1 {
				t.Fatal(v, err, *calls, faulty.closed)
			}
			if code == 200 {
				var proof *resource.ResponseError
				if v == nil || v.Resource != nil || v.StatusCode != 200 || string(v.Envelope) != faulty.text || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != faulty.text || proof.Header.Get("X-Request-Id") != "actual" {
					t.Fatal(v, err, proof)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if v != nil || !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != faulty.text || native.ResponseHeader.Get("X-Request-Id") != "actual" {
					t.Fatal(v, err, native)
				}
			}
		})
	}
	closeCause := errors.New("rejected close")
	faulty := &groupFaultBody{text: `{"missing":true}`, closeErr: closeCause}
	api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { r := groupWire(404, ""); r.Body = faulty; return r })
	if err := api.DeleteRecord(context.Background(), "id"); !errors.Is(err, closeCause) || !gophercloud.ResponseCodeIs(err, 404) || len(*calls) != 1 || faulty.closed != 1 {
		t.Fatal(err, *calls)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("actual accepted cancel")
	faulty = &groupFaultBody{text: `{"group":{}}`, afterRead: func() { cancel(cause) }}
	api, calls = groupAPI(t, "3.38", func(*http.Request) *http.Response { r := groupWire(200, ""); r.Body = faulty; return r })
	v, err := api.GetRecord(ctx, "id")
	var proof *resource.ResponseError
	if v == nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.As(err, &proof) || proof.StatusCode != 200 || len(*calls) != 1 || faulty.closed != 1 {
		t.Fatal(v, err, proof, *calls)
	}
}
func TestPythonCinderGroupReadBoundarySourceMutationCannotBeRestoredByClose(t *testing.T) {
	var api *groups.API
	faulty := &groupFaultBody{text: `{"group":{"id":"wire"}}`}
	faulty.afterRead = func() { api.RawClient().Microversion = "3.13" }
	faulty.afterClose = func() { api.RawClient().Microversion = "3.38" }
	var calls *[]groupCall
	api, calls = groupAPI(t, "3.38", func(*http.Request) *http.Response { r := groupWire(200, ""); r.Body = faulty; return r })
	v, err := api.GetRecord(context.Background(), "id")
	var proof *resource.ResponseError
	if v == nil || v.Resource != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != faulty.text || len(*calls) != 1 || faulty.closed != 1 {
		t.Fatal(v, err, proof, *calls)
	}
}
func TestPythonCinderGroupNativeRetryOwnedVersionReservedHeadersAndBody(t *testing.T) {
	mutations := []struct {
		name   string
		change func(*gophercloud.RequestOpts)
	}{
		{"selected_version", func(o *gophercloud.RequestOpts) { o.MoreHeaders["OpenStack-API-Version"] = "volume 3.13" }},
		{"legacy_version", func(o *gophercloud.RequestOpts) { o.MoreHeaders["X-OpenStack-Volume-API-Version"] = "3.13" }},
		{"foreign_version", func(o *gophercloud.RequestOpts) { o.MoreHeaders["X-OpenStack-Nova-API-Version"] = "2.90" }},
		{"authorization", func(o *gophercloud.RequestOpts) { o.MoreHeaders["Authorization"] = "forged" }},
		{"service_token", func(o *gophercloud.RequestOpts) { o.MoreHeaders["X-Service-Token"] = "forged" }},
		{"cookie", func(o *gophercloud.RequestOpts) { o.MoreHeaders["Cookie"] = "forged" }},
		{"framing", func(o *gophercloud.RequestOpts) { o.MoreHeaders["Transfer-Encoding"] = "chunked" }},
		{"body_type", func(o *gophercloud.RequestOpts) { o.MoreHeaders["Content-Type"] = "text/plain" }},
		{"omit_owned_version", func(o *gophercloud.RequestOpts) { o.OmitHeaders = append(o.OmitHeaders, "openstack-api-version") }},
		{"omit_owned_type", func(o *gophercloud.RequestOpts) { o.OmitHeaders = append(o.OmitHeaders, "Content-Type") }},
		{"replace_body", func(o *gophercloud.RequestOpts) {
			o.JSONBody = map[string]any{"group": map[string]string{"name": "forged"}}
		}},
		{"remove_receipt", func(o *gophercloud.RequestOpts) { o.KeepResponseBody = false }},
	}
	for _, item := range mutations {
		t.Run(item.name, func(t *testing.T) {
			api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(503, `{"retry":"actual"}`) })
			callbackCause := errors.New("callback actual")
			api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				item.change(o)
				return callbackCause
			}
			v, err := api.CreateRecord(context.Background(), groups.WithRecordCreateField("name", "original"))
			var native gophercloud.ErrUnexpectedResponseCode
			if v != nil || !errors.Is(err, callbackCause) || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"retry":"actual"}` || native.ResponseHeader.Get("X-Request-Id") != "actual" || !slices.Contains(native.Expected, 202) || slices.Contains(native.Expected, 503) || len(*calls) != 1 {
				t.Fatal(v, err, native, *calls)
			}
		})
	}
}
func TestPythonCinderGroupNativeRetryOrdinaryHeadersLiveTokenAndOriginalStatus(t *testing.T) {
	api, calls := groupAPI(t, "3.38", func(r *http.Request) *http.Response {
		if r.Header.Get("X-Retry") == "allowed" {
			return groupWire(202, `{"group":{"id":"wire"}}`)
		}
		return groupWire(503, `{"retry":true}`)
	})
	provider := api.RawClient().ProviderClient
	provider.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
		o.MoreHeaders["X-Retry"] = "allowed"
		o.MoreHeaders["X-Auth-Token"] = "forged"
		provider.SetToken("fresh")
		return nil
	}
	v, err := api.CreateRecord(context.Background(), groups.WithRecordCreateField("name", false))
	if err != nil || len(*calls) != 2 || (*calls)[1].headers.Get("X-Auth-Token") != "fresh" || (*calls)[1].headers.Get("X-Retry") != "allowed" || (*calls)[0].body != (*calls)[1].body || v.StatusCode != 202 {
		t.Fatal(v, err, *calls)
	}
	groupField(t, v, "name", `false`)
	api, calls = groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(404, `{"missing":"actual"}`) })
	api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
		o.OkCodes = append(o.OkCodes, 404)
		return nil
	}
	err = api.DeleteRecord(context.Background(), "id")
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != `{"missing":"actual"}` || len(*calls) != 2 {
		t.Fatal("native hook cannot invent ignorable absence", err, *calls)
	}
	api, calls = groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(404, `{"missing":true}`) })
	cause := errors.New("stop actual")
	api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, _ error, _ uint) error {
		return cause
	}
	err = api.DeleteRecord(context.Background(), "id")
	if !errors.Is(err, cause) || !gophercloud.ResponseCodeIs(err, 404) || len(*calls) != 1 {
		t.Fatal("mixed404 cannot be suppressed", err, *calls)
	}
	api, calls = groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(404, `{"missing":true}`) })
	api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
		return original
	}
	if err := api.DeleteRecord(context.Background(), "id"); err != nil || len(*calls) != 1 {
		t.Fatal("clean native callback stop must remain suppressible", err, *calls)
	}
}
func TestPythonCinderGroupVersionlessRetryCannotIntroduceVersionHeaders(t *testing.T) {
	for _, header := range []string{"OpenStack-API-Version", "X-OpenStack-Volume-API-Version", "X-OpenStack-Nova-API-Version"} {
		t.Run(header, func(t *testing.T) {
			api, calls := groupAPI(t, "", func(r *http.Request) *http.Response {
				if !strings.Contains(r.URL.Path, "/groups/") {
					return groupWire(200, `{"versions":[]}`)
				}
				return groupWire(503, `{"actual":true}`)
			})
			api.RawClient().ProviderClient.RetryFunc = func(_ context.Context, _ string, target string, o *gophercloud.RequestOpts, original error, _ uint) error {
				if !strings.Contains(target, "/groups/") {
					return original
				}
				o.MoreHeaders[header] = "forged"
				return nil
			}
			v, err := api.GetRecord(context.Background(), "id")
			var native gophercloud.ErrUnexpectedResponseCode
			if v != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || len(*calls) != 3 || (*calls)[2].headers.Get(header) != "" {
				t.Fatal(v, err, native, *calls)
			}
		})
	}
}
