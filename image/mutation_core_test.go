package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type mutationCoreAck struct {
	id, literal string
	body        []byte
	header      http.Header
	status      int
}

func mutationCoreInvoke(service *Service, operation string, ctx context.Context, ref resource.Ref, tag string, options ...ImageMutationOption) (*mutationCoreAck, error) {
	var tagged *ImageTagResult
	var action *ImageActionResult
	var err error
	switch operation {
	case "AddTag":
		tagged, err = service.AddTag(ctx, ref, tag, options...)
	case "RemoveTag":
		tagged, err = service.RemoveTag(ctx, ref, tag, options...)
	case "DeactivateImage":
		action, err = service.DeactivateImage(ctx, ref, options...)
	case "ReactivateImage":
		action, err = service.ReactivateImage(ctx, ref, options...)
	default:
		panic("unknown test operation")
	}
	if tagged != nil {
		return &mutationCoreAck{tagged.ImageID, tagged.Tag, tagged.Body, tagged.Header, tagged.StatusCode}, err
	}
	if action != nil {
		return &mutationCoreAck{action.ImageID, action.Action, action.Body, action.Header, action.StatusCode}, err
	}
	return nil, err
}

func TestImageMutationCoreFixedRoutesAndOpaqueAcknowledgements(t *testing.T) {
	for _, test := range []struct{ operation, method, suffix, literal string }{
		{"AddTag", http.MethodPut, "/tags/" + url.PathEscape(" Mixed %2F ?# 中文 "), " Mixed %2F ?# 中文 "},
		{"RemoveTag", http.MethodDelete, "/tags/" + url.PathEscape(" Mixed %2F ?# 中文 "), " Mixed %2F ?# 中文 "},
		{"DeactivateImage", http.MethodPost, "/actions/deactivate", "deactivate"},
		{"ReactivateImage", http.MethodPost, "/actions/reactivate", "reactivate"},
	} {
		t.Run(test.operation, func(t *testing.T) {
			const id = "fixed-한글"
			raw := []byte{'o', 'p', 'a', 'q', 'u', 'e', 255}
			body := &deleteCoreBody{reader: bytes.NewReader(raw)}
			header := http.Header{"X-Actual": {"before"}, "Location": {"https://foreign.test/other"}}
			var requests int
			client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
				requests++
				path := "/reverse/glance/v2/images/" + url.PathEscape(id) + test.suffix
				if req.Method != test.method || req.URL.EscapedPath() != path || req.URL.RequestURI() != path || req.URL.RawQuery != "" || req.Body != nil || req.Header.Get("X-Extra") != "owned" {
					t.Fatal("mutation route/body/header changed", req.Method, req.URL, req.Header, req.Body)
				}
				return deleteCoreHTTP(204, body, header), nil
			})
			ack, err := mutationCoreInvoke(New(client), test.operation, context.Background(), resource.ID(id), " Mixed %2F ?# 中文 ", WithImageMutationHeader("X-Extra", "owned"))
			if err != nil || ack == nil || ack.id != id || ack.literal != test.literal || ack.status != 204 || !bytes.Equal(ack.body, raw) || ack.header.Get("X-Actual") != "before" || requests != 1 || body.closes != 1 {
				t.Fatalf("ack=%+v err=%v requests=%d closes=%d", ack, err, requests, body.closes)
			}
			header.Set("X-Actual", "wire mutation")
			if ack.header.Get("X-Actual") != "before" {
				t.Fatal("ack aliases wire header", ack.header)
			}
		})
	}
}

func TestImageMutationCoreTagBoundsPrecedeOptionsAndLookup(t *testing.T) {
	for _, tag := range []string{"", ".", "..", "a/b", "a\\b", "a\x00b", "a\nb", "a\u0085b", "bad\xff", strings.Repeat("界", 256)} {
		var requests, callbacks int
		client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
			requests++
			return deleteCoreHTTP(204, http.NoBody, nil), nil
		})
		ack, err := mutationCoreInvoke(New(client), "AddTag", context.Background(), resource.Name("exact"), tag, func(*ImageMutationOpts) error { callbacks++; return nil })
		if ack != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 0 || requests != 0 {
			t.Fatalf("tag=%q ack=%+v err=%v callbacks=%d requests=%d", tag, ack, err, callbacks, requests)
		}
	}
	for _, tag := range []string{strings.Repeat("界", 255), " ", "é", "e\u0301", "%2F", "question?#"} {
		var requests int
		client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
			requests++
			if req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed/tags/"+url.PathEscape(tag) || req.URL.Path != "/reverse/glance/v2/images/fixed/tags/"+tag {
				t.Fatal("literal tag encoded or normalized incorrectly", tag, req.URL)
			}
			return deleteCoreHTTP(204, http.NoBody, nil), nil
		})
		ack, err := mutationCoreInvoke(New(client), "RemoveTag", context.Background(), resource.ID("fixed"), tag)
		if err != nil || ack == nil || ack.literal != tag || requests != 1 {
			t.Fatalf("tag=%q ack=%+v err=%v requests=%d", tag, ack, err, requests)
		}
	}
}

func TestImageMutationCoreExactNameAndStrictLookupFailures(t *testing.T) {
	for _, mode := range []string{"unique", "missing", "duplicate", "late HTTP", "late decode", "bad resolved ID", "provider changed", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("lookup caller cancellation")
			var lists, mutations int
			var client *gophercloud.ServiceClient
			client = deleteCoreClient(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodDelete {
					mutations++
					if req.URL.Path != "/reverse/glance/v2/images/fixed/tags/literal" {
						t.Fatal("lookup did not bind fixed ID", req.URL)
					}
					return deleteCoreHTTP(204, http.NoBody, nil), nil
				}
				lists++
				if req.Method != http.MethodGet || req.URL.Query().Get("name") != "exact" || req.Header.Get("X-Extra") != "owned" {
					t.Fatal("name lookup changed", req.Method, req.URL, req.Header)
				}
				if lists == 1 {
					rows := `[{"id":"near","name":"near-exact"}]`
					if mode == "duplicate" || strings.HasPrefix(mode, "late") {
						rows = `[{"id":"first","name":"exact"}]`
					}
					return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(`{"images":`+rows+`,"next":"/v2/images?name=exact&page=2"}`)), nil), nil
				}
				if mode == "late HTTP" {
					return deleteCoreHTTP(503, io.NopCloser(strings.NewReader("late failure")), nil), nil
				}
				if mode == "late decode" {
					return deleteCoreHTTP(200, io.NopCloser(strings.NewReader("malformed")), nil), nil
				}
				rows := `[{"id":"fixed","name":"exact"}]`
				if mode == "missing" {
					rows = `[]`
				}
				if mode == "bad resolved ID" {
					rows = `[{"id":"unsafe/id","name":"exact"}]`
				}
				if mode == "provider changed" {
					client.ProviderClient = &gophercloud.ProviderClient{}
				}
				if mode == "canceled" {
					cancel(cause)
				}
				return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(`{"images":`+rows+`}`)), nil), nil
			})
			ack, err := mutationCoreInvoke(New(client), "RemoveTag", ctx, resource.Name("exact"), "literal", WithImageMutationHeader("X-Extra", "owned"))
			if mode == "unique" {
				if err != nil || ack == nil || ack.id != "fixed" || lists != 2 || mutations != 1 {
					t.Fatalf("ack=%+v err=%v lists=%d mutations=%d", ack, err, lists, mutations)
				}
				return
			}
			if ack != nil || err == nil || mutations != 0 || lists != 2 {
				t.Fatalf("failed lookup mutated image: ack=%+v err=%v lists=%d mutations=%d", ack, err, lists, mutations)
			}
			switch mode {
			case "missing":
				if !errors.Is(err, resource.ErrNotFound) {
					t.Fatal("logical absence suppressed", err)
				}
			case "duplicate":
				if !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(err)
				}
			case "bad resolved ID", "provider changed":
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			case "canceled":
				if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
					t.Fatal("lookup custom cancellation lost", err)
				}
			}
		})
	}
}

func TestImageMutationCoreAcceptedFailuresKeepIndependentEvidence(t *testing.T) {
	readCause, closeCause, customCause := errors.New("read failed"), errors.New("close failed"), errors.New("caller canceled")
	for _, test := range []struct {
		operation         string
		readErr, closeErr error
		canceled          bool
	}{
		{"AddTag", readCause, nil, false}, {"RemoveTag", nil, closeCause, false}, {"DeactivateImage", readCause, closeCause, true}, {"ReactivateImage", nil, closeCause, false},
	} {
		t.Run(test.operation, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			header := http.Header{"X-Actual": {"before"}}
			body := &deleteCoreBody{reader: deleteCoreReader(func(buffer []byte) (int, error) {
				header.Set("X-Actual", "changed during read")
				if test.canceled {
					cancel(customCause)
				}
				err := test.readErr
				if err == nil {
					err = io.EOF
				}
				return copy(buffer, "raw acknowledgement"), err
			}), closeErr: test.closeErr}
			var requests, retries int
			client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
				requests++
				return deleteCoreHTTP(204, body, header), nil
			})
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			ack, err := mutationCoreInvoke(New(client), test.operation, ctx, resource.ID("fixed"), "literal")
			var accepted *resource.ResponseError
			var operation *resource.OperationError
			if ack == nil || ack.id != "fixed" || ack.status != 204 || string(ack.body) != "raw acknowledgement" || ack.header.Get("X-Actual") != "before" || !errors.As(err, &accepted) || accepted.StatusCode != 204 || !errors.As(err, &operation) || operation.Operation != test.operation || operation.Resource != "image" || requests != 1 || retries != 0 || body.closes != 1 {
				t.Fatalf("ack=%+v accepted=%+v err=%v requests=%d retries=%d closes=%d", ack, accepted, err, requests, retries, body.closes)
			}
			for _, cause := range []error{test.readErr, test.closeErr} {
				if cause != nil && !errors.Is(err, cause) {
					t.Fatal("accepted body cause lost", cause, err)
				}
			}
			if test.canceled && (!errors.Is(err, context.Canceled) || !errors.Is(err, customCause)) {
				t.Fatal("accepted context causes lost", err)
			}
			ack.body[0] = '!'
			ack.header.Set("X-Actual", "caller mutation")
			if string(accepted.Body) != "raw acknowledgement" || accepted.Header.Get("X-Actual") != "before" {
				t.Fatal("acknowledgement aliases error evidence", accepted)
			}
		})
	}
}

func TestImageMutationCoreSourcePreflightAndCapturedLiveAuth(t *testing.T) {
	var client *gophercloud.ServiceClient
	var retained *ImageMutationOpts
	var requests int
	client = deleteCoreClient(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path != "/reverse/glance/v2/images/fixed/actions/deactivate" || req.Header.Get("X-Snapshot") != "before" || req.Header.Get("X-Extra") != "owned" || req.Header.Get("X-Auth-Token") != "after" {
			t.Fatal("captured source or live token changed", req.URL, req.Header)
		}
		retained.Headers["X-Extra"] = "retained during request"
		return deleteCoreHTTP(204, http.NoBody, nil), nil
	})
	client.MoreHeaders = map[string]string{"X-Snapshot": "before"}
	ack, err := mutationCoreInvoke(New(client), "DeactivateImage", context.Background(), resource.ID("fixed"), "", func(value *ImageMutationOpts) error {
		retained = value
		value.Headers["X-Extra"] = "owned"
		client.ResourceBase = "https://example.test/later/v2/"
		client.MoreHeaders["X-Snapshot"] = "after"
		client.SetToken("after")
		return nil
	})
	if err != nil || ack == nil || requests != 1 {
		t.Fatalf("ack=%+v err=%v requests=%d", ack, err, requests)
	}
	custom := errors.New("preflight caller cause")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(custom)
	for _, test := range []struct {
		name string
		ctx  context.Context
		edit func(*gophercloud.ServiceClient)
	}{
		{"nil context", nil, nil}, {"canceled", ctx, nil}, {"nil provider", context.Background(), func(c *gophercloud.ServiceClient) { c.ProviderClient = nil }},
		{"wrong service", context.Background(), func(c *gophercloud.ServiceClient) { c.Type = "compute" }}, {"foreign base", context.Background(), func(c *gophercloud.ServiceClient) { c.ResourceBase = "https://foreign.test/v2/" }},
		{"bad UTF8", context.Background(), func(c *gophercloud.ServiceClient) { c.ResourceBase += "bad\xff/" }}, {"query base", context.Background(), func(c *gophercloud.ServiceClient) { c.ResourceBase += "?query=yes" }},
		{"protected source header", context.Background(), func(c *gophercloud.ServiceClient) { c.MoreHeaders = map[string]string{"Authorization": "override"} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			var requests, callbacks int
			client := deleteCoreClient(func(*http.Request) (*http.Response, error) {
				requests++
				return deleteCoreHTTP(204, http.NoBody, nil), nil
			})
			if test.edit != nil {
				test.edit(client)
			}
			ack, err := mutationCoreInvoke(New(client), "ReactivateImage", test.ctx, resource.ID("fixed"), "", func(*ImageMutationOpts) error { callbacks++; return nil })
			if ack != nil || err == nil || requests != 0 || callbacks != 0 {
				t.Fatalf("source preflight reached callback/HTTP: ack=%+v err=%v requests=%d callbacks=%d", ack, err, requests, callbacks)
			}
			if test.name == "canceled" && (!errors.Is(err, custom) || !errors.Is(err, context.Canceled)) {
				t.Fatal("preflight caller cause lost", err)
			}
		})
	}
	for _, tamper := range []func(*Service){
		func(s *Service) { s.client.ProviderClient = &gophercloud.ProviderClient{} },
		func(s *Service) { s.client = deleteCoreClient(nil) },
		func(s *Service) {
			s.client.Endpoint = "https://foreign.test/"
			s.client.ResourceBase = "https://foreign.test/v2/"
		},
	} {
		requests = 0
		client := deleteCoreClient(func(*http.Request) (*http.Response, error) {
			requests++
			return deleteCoreHTTP(204, http.NoBody, nil), nil
		})
		service := New(client)
		ack, err := mutationCoreInvoke(service, "AddTag", context.Background(), resource.ID("fixed"), "tag", func(*ImageMutationOpts) error { tamper(service); return nil })
		if ack != nil || !errors.Is(err, resource.ErrInvalidOption) || requests != 0 {
			t.Fatalf("source changed before mutation: ack=%+v err=%v requests=%d", ack, err, requests)
		}
	}
}

func TestImageMutationCoreNativeStatusesAndPrebodyOwnership(t *testing.T) {
	for _, operation := range []string{"AddTag", "RemoveTag", "DeactivateImage", "ReactivateImage"} {
		for _, status := range []int{200, 201, 202, 206, 404, 413} {
			body := &deleteCoreBody{reader: strings.NewReader("native rejection")}
			client := deleteCoreClient(func(*http.Request) (*http.Response, error) {
				return deleteCoreHTTP(status, body, http.Header{"X-Actual": {"rejected"}}), nil
			})
			ack, err := mutationCoreInvoke(New(client), operation, context.Background(), resource.ID("fixed"), "tag")
			var native gophercloud.ErrUnexpectedResponseCode
			var accepted *resource.ResponseError
			if ack != nil || !errors.As(err, &native) || native.Actual != status || len(native.Expected) != 1 || native.Expected[0] != 204 || string(native.Body) != "native rejection" || native.ResponseHeader.Get("X-Actual") != "rejected" || errors.As(err, &accepted) || body.closes != 1 {
				t.Fatalf("operation=%s status=%d ack=%+v native=%+v err=%v closes=%d", operation, status, ack, native, err, body.closes)
			}
		}
	}
	for _, test := range []struct {
		name   string
		tamper func(*gophercloud.RequestOpts)
	}{
		{"safe retry", nil}, {"expanded codes", func(o *gophercloud.RequestOpts) { o.OkCodes = []int{202} }},
		{"KeepBody", func(o *gophercloud.RequestOpts) { o.KeepResponseBody = false }}, {"JSONResponse", func(o *gophercloud.RequestOpts) { o.JSONResponse = &struct{}{} }},
		{"RawBody", func(o *gophercloud.RequestOpts) { o.RawBody = strings.NewReader("unexpected") }}, {"JSONnull", func(o *gophercloud.RequestOpts) { o.JSONBody = json.RawMessage(`null`) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			var requests, retries int
			var bodies []*deleteCoreBody
			client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.Method != http.MethodPost || req.URL.String() != "https://example.test/reverse/glance/v2/images/fixed/actions/reactivate" || req.Body != nil {
					t.Fatal("retry changed method/target/body", req.Method, req.URL, req.Body)
				}
				status := 503
				if requests == 2 {
					status = 204
					if test.name == "expanded codes" {
						status = 202
					}
				}
				body := &deleteCoreBody{reader: strings.NewReader(fmt.Sprintf("actual%d", status))}
				bodies = append(bodies, body)
				return deleteCoreHTTP(status, body, nil), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, _ error, _ uint) error {
				retries++
				if test.tamper != nil {
					test.tamper(options)
				}
				return nil
			}
			ack, err := mutationCoreInvoke(New(client), "ReactivateImage", context.Background(), resource.ID("fixed"), "")
			if retries != 1 {
				t.Fatal("wrong prebody retry count", retries, err)
			}
			for _, body := range bodies {
				if body.closes != 1 {
					t.Fatal("retry body ownership changed", body.closes, err)
				}
			}
			switch test.name {
			case "safe retry":
				if err != nil || ack == nil || requests != 2 || string(ack.body) != "actual204" {
					t.Fatalf("ack=%+v err=%v requests=%d", ack, err, requests)
				}
			case "expanded codes":
				var native gophercloud.ErrUnexpectedResponseCode
				if ack != nil || !errors.As(err, &native) || native.Actual != 202 || len(native.Expected) != 1 || native.Expected[0] != 204 || requests != 2 {
					t.Fatalf("expanded policy fabricated ack=%+v native=%+v err=%v requests=%d", ack, native, err, requests)
				}
			default:
				if ack != nil || !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || requests != 1 {
					t.Fatalf("body tamper replayed: ack=%+v err=%v requests=%d", ack, err, requests)
				}
			}
		})
	}
}
