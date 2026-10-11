package attachments_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func attachmentModeDocument(maximum, minimum string) string {
	body, _ := json.Marshal(map[string]any{"version": map[string]string{"id": "v3.0", "status": "CURRENT", "version": maximum, "min_version": minimum}})
	return string(body)
}

// Reuse the existing public testhelper fixture and recorded transport; real
// discovery uses the catalog v3 endpoint, independently of ResourceBase.
func attachmentModeFixture(t *testing.T, selected, maximum, minimum string, reply func(*http.Request) *http.Response) (*attachments.API, *[]pythonAttachmentCall, *[]http.Header) {
	t.Helper()
	headers := &[]http.Header{}
	api, calls := pythonAttachmentAPI(t, func(req *http.Request) *http.Response {
		*headers = append(*headers, req.Header.Clone())
		if req.Method == http.MethodGet && req.URL.Path == "/cinder/v3/" {
			r := pythonAttachmentWire(200, attachmentModeDocument(maximum, minimum))
			r.Header.Set("X-Request-Id", "capability")
			return r
		}
		return reply(req)
	})
	api.RawClient().Endpoint = strings.TrimSuffix(api.RawClient().ResourceBase, "project/")
	api.RawClient().Microversion = selected
	return api, calls, headers
}

func TestPythonAttachmentRecordModeRequiresAdvertisedRangeAndSelectedDefault(t *testing.T) {
	for _, item := range []struct {
		selected, maximum, minimum, header string
		keepMode                           bool
	}{
		{"3.80", "3.80", "3.60", "3.80", true},
		{"3.80", "3.80", "", "3.80", true},
		{"3.80", "", "3.0", "3.80", true},
		{"3.80", "", "", "3.80", true},
		{"3.53", "3.80", "3.0", "3.53", true},
		{"3.80", "3.53", "3.0", "3.80", true},
		{"3.80", "3.71", "3.0", "3.80", false},
		{"latest", "3.80", "3.0", "latest", true},
		{"3.latest", "3.80", "3.0", "3.latest", false},
		{"", "3.80", "3.0", "3.71", false},
		{"", "3.80", "3.60", "3.71", true},
		{"", "3.90", "3.80", "", true},
		{"", "", "3.0", "", true},
	} {
		t.Run(fmt.Sprintf("%s_%s_%s", item.selected, item.maximum, item.minimum), func(t *testing.T) {
			api, calls, headers := attachmentModeFixture(t, item.selected, item.maximum, item.minimum, func(*http.Request) *http.Response {
				r := pythonAttachmentWire(201, `{"attachment":{"id":"a"}}`)
				r.Header.Set("X-Request-Id", "post")
				return r
			})
			api.RawClient().MoreHeaders = map[string]string{"X-Custom": "owned"}
			v, err := api.CreateRecord(context.Background(), "vol", attachments.WithRecordCreateField("mode", false))
			if err != nil || len(*calls) != 2 || (*calls)[0].method != http.MethodGet || (*calls)[0].path != "/cinder/v3/" || (*calls)[0].body != "" || (*calls)[1].method != http.MethodPost || (*calls)[1].path != pythonAttachmentBase {
				t.Fatal(v, err, *calls)
			}
			if strings.Contains((*calls)[1].body, `"mode":false`) != item.keepMode {
				t.Fatal("default version is insufficient without advertised bounds", item, *calls)
			}
			if (*headers)[0].Get("OpenStack-API-Version") != "" || (*headers)[0].Get("X-OpenStack-Volume-API-Version") != "" || (*headers)[0].Get("X-Custom") != "owned" {
				t.Fatal("discovery headers", *headers)
			}
			if (*headers)[1].Get("X-OpenStack-Volume-API-Version") != item.header || (*headers)[1].Get("X-Custom") != "owned" {
				t.Fatal("action headers", *headers)
			}
			if item.header == "" {
				if (*headers)[1].Get("OpenStack-API-Version") != "" {
					t.Fatal(*headers)
				}
			} else {
				if (*headers)[1].Get("OpenStack-API-Version") != "volume "+item.header {
					t.Fatal(*headers)
				}
			}
			if api.RawClient().Microversion != item.selected || api.RawClient().MoreHeaders["X-Custom"] != "owned" || len(v.Discovery) != 1 || v.Discovery[0].StatusCode != 200 || v.Discovery[0].Header.Get("X-Request-Id") != "capability" || v.StatusCode != 201 || v.Header.Get("X-Request-Id") != "post" {
				t.Fatal("discovery is distinct from POST ACK and shared policy", v, api.RawClient())
			}
			attachmentRecordField(t, v, "attach_mode", `false`)
			v.Discovery[0].Envelope[2] = 'X'
			v.Discovery[0].Header.Set("X-Request-Id", "changed")
			if v.Header.Get("X-Request-Id") != "post" || v.Wire.Header.Get("X-Request-Id") != "post" || string(v.Wire.Body["id"]) != `"a"` {
				t.Fatal("capability proof aliases POST", v)
			}
		})
	}
}

func TestPythonAttachmentRecordModeFiniteDiscoveryFallbackKeepsUnknownSupport(t *testing.T) {
	for _, status := range []int{404, 405} {
		api, calls := pythonAttachmentAPI(t, func(req *http.Request) *http.Response {
			if req.Method == http.MethodGet {
				return pythonAttachmentWire(status, `{"missing":true}`)
			}
			return pythonAttachmentWire(201, `{"attachment":{"id":"a"}}`)
		})
		api.RawClient().Endpoint = strings.TrimSuffix(api.RawClient().ResourceBase, "project/")
		api.RawClient().Microversion = "3.80"
		v, err := api.CreateRecord(context.Background(), "vol", attachments.WithRecordCreateField("mode", false))
		if err != nil || len(*calls) != 3 || (*calls)[0].path != "/cinder/v3/" || (*calls)[1].path != "/cinder/" || (*calls)[2].method != http.MethodPost || !strings.Contains((*calls)[2].body, `"mode":false`) || len(v.Discovery) != 0 || v.StatusCode != 201 {
			t.Fatal(v, err, *calls)
		}
	}
	api, calls := pythonAttachmentAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Path == "/cinder/v3/" {
			return pythonAttachmentWire(200, `{"versions":[{"id":"v2.0","version":"2.0","min_version":"2.0"}]}`)
		}
		if req.Method == http.MethodGet {
			return pythonAttachmentWire(200, attachmentModeDocument("3.80", "3.60"))
		}
		return pythonAttachmentWire(201, `{"attachment":{"id":"a"}}`)
	})
	api.RawClient().Endpoint = strings.TrimSuffix(api.RawClient().ResourceBase, "project/")
	v, err := api.CreateRecord(context.Background(), "vol", attachments.WithRecordCreateField("mode", false))
	if err != nil || len(*calls) != 3 || !strings.Contains((*calls)[2].body, `"mode":false`) || len(v.Discovery) != 2 || v.StatusCode != 201 {
		t.Fatal(v, err, *calls)
	}
}

func TestPythonAttachmentRecordModeDiscoveryFailureRetainsOnlyDiscoveryProof(t *testing.T) {
	for _, item := range []struct {
		name, body  string
		read, close error
	}{
		{"malformed", `{"version":`, nil, nil},
		{"invalid_bound", attachmentModeDocument("bad", "3.0"), nil, nil},
		{"read", `{"partial":`, errors.New("discovery read"), nil},
		{"close", attachmentModeDocument("3.80", "3.0"), nil, errors.New("discovery close")},
	} {
		t.Run(item.name, func(t *testing.T) {
			api, _ := pythonAttachmentAPI(t, func(*http.Request) *http.Response { return nil })
			calls := 0
			api.RawClient().Endpoint = strings.TrimSuffix(api.RawClient().ResourceBase, "project/")
			api.RawClient().HTTPClient.Transport = pythonAttachmentTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet {
					t.Fatal("POST after broken discovery")
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"X-Request-Id": {"discovery-fault"}}, Body: &attachmentRecordFaultBody{data: []byte(item.body), read: item.read, close: item.close}}, nil
			})
			v, err := api.CreateRecord(context.Background(), "vol", attachments.WithRecordCreateField("mode", false))
			var proof *resource.ResponseError
			if err == nil || calls != 1 || v == nil || len(v.Discovery) != 1 || string(v.Discovery[0].Envelope) != item.body || v.Discovery[0].StatusCode != 200 || v.Discovery[0].Header.Get("X-Request-Id") != "discovery-fault" || v.StatusCode != 0 || v.Envelope != nil || v.Wire != nil || v.Resource != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != item.body {
				t.Fatal(v, err, proof, calls)
			}
			if item.read != nil && !errors.Is(err, item.read) || item.close != nil && !errors.Is(err, item.close) {
				t.Fatal("physical cause lost", err)
			}
		})
	}
}

type attachmentModeRestoringBody struct {
	*attachmentRecordFaultBody
	restore func()
}

func (b *attachmentModeRestoringBody) Close() error {
	err := b.attachmentRecordFaultBody.Close()
	b.restore()
	return err
}

func TestPythonAttachmentRecordModeDiscoverySourceAndCancellationStopBeforePost(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		api, _ := pythonAttachmentAPI(t, func(*http.Request) *http.Response { return nil })
		calls := 0
		ctx, cancel := context.WithCancelCause(context.Background())
		cause := errors.New("capability cancelled")
		defer cancel(nil)
		api.RawClient().Endpoint = strings.TrimSuffix(api.RawClient().ResourceBase, "project/")
		api.RawClient().HTTPClient.Transport = pythonAttachmentTransport(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Method != http.MethodGet {
				t.Fatal("POST after invalid source")
			}
			body := &attachmentRecordFaultBody{data: []byte(attachmentModeDocument("3.80", "3.0")), afterRead: func() {
				if cancelled {
					cancel(cause)
				} else {
					api.RawClient().Microversion = "3.80"
				}
			}}
			var responseBody interface {
				Read([]byte) (int, error)
				Close() error
			} = body
			if !cancelled {
				responseBody = &attachmentModeRestoringBody{attachmentRecordFaultBody: body, restore: func() { api.RawClient().Microversion = "3.54" }}
			}
			return &http.Response{StatusCode: 200, Header: http.Header{}, Body: responseBody}, nil
		})
		v, err := api.CreateRecord(ctx, "vol", attachments.WithRecordCreateField("mode", false))
		var proof *resource.ResponseError
		if err == nil || calls != 1 || v == nil || len(v.Discovery) != 1 || v.StatusCode != 0 || !errors.As(err, &proof) || proof.StatusCode != 200 {
			t.Fatal(v, err, proof, calls)
		}
		if cancelled {
			if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
				t.Fatal(err)
			}
		} else {
			if !errors.Is(err, resource.ErrInvalidOption) || api.RawClient().Microversion != "3.54" {
				t.Fatal("source mutation was restored at Close but must stay invalid", err)
			}
		}
	}
}

func TestPythonAttachmentRecordModeProtectsNegotiatedHeadersAndPostAcknowledgement(t *testing.T) {
	t.Run("retry_version_header", func(t *testing.T) {
		api, calls, headers := attachmentModeFixture(t, "", "3.80", "3.0", func(*http.Request) *http.Response { return pythonAttachmentWire(409, `{"conflict":true}`) })
		api.RawClient().RetryFunc = func(_ context.Context, method string, _ string, opts *gophercloud.RequestOpts, original error, _ uint) error {
			if method == http.MethodPost {
				opts.MoreHeaders["X-OpenStack-Volume-API-Version"] = "3.54"
				return nil
			}
			return original
		}
		v, err := api.CreateRecord(context.Background(), "vol", attachments.WithRecordCreateField("mode", false))
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 409 || len(*calls) != 2 || (*headers)[1].Get("X-OpenStack-Volume-API-Version") != "3.71" || v == nil || len(v.Discovery) != 1 || v.StatusCode != 0 || v.Resource != nil || api.RawClient().Microversion != "" {
			t.Fatal(v, err, native, *calls, *headers)
		}
	})
	t.Run("accepted_post_body_fault", func(t *testing.T) {
		readCause, closeCause := errors.New("POST read"), errors.New("POST close")
		api, calls, _ := attachmentModeFixture(t, "3.80", "3.80", "3.0", func(*http.Request) *http.Response {
			return &http.Response{StatusCode: 201, Header: http.Header{"X-Request-Id": {"post-fault"}}, Body: &attachmentRecordFaultBody{data: []byte(`{"attachment":`), read: readCause, close: closeCause}}
		})
		v, err := api.CreateRecord(context.Background(), "vol", attachments.WithRecordCreateField("mode", false))
		var proof *resource.ResponseError
		if !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.As(err, &proof) || proof.StatusCode != 201 || v == nil || len(v.Discovery) != 1 || v.Discovery[0].StatusCode != 200 || v.StatusCode != 201 || string(v.Envelope) != `{"attachment":` || v.Header.Get("X-Request-Id") != "post-fault" || v.Resource != nil || len(*calls) != 2 {
			t.Fatal(v, err, proof, *calls)
		}
	})
}
