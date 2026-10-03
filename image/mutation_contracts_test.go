package image_test

import (
	"bytes"
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
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/image"
	sdkimages "gophercloudsdk/image/v2/images"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

const mutationPrefix = "/reverse/mutations/glance/v2/"
const mutationBase = "https://glance.invalid" + mutationPrefix

type mutationTransport func(*http.Request) (*http.Response, error)

func (f mutationTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	wire, err := f(r)
	if wire != nil && wire.Request == nil {
		wire.Request = r
	}
	return wire, err
}

type mutationBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *mutationBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type mutationReader func([]byte) (int, error)

func (f mutationReader) Read(p []byte) (int, error) { return f(p) }

func mutationWire(code int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{
		"Content-Type": {"application/json"}, "X-Request-Id": {"actual-mutation"},
	}, Body: body}
}

func mutationClient(transport mutationTransport) *gophercloud.ServiceClient {
	provider := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: transport}}
	provider.UseTokenLock()
	provider.SetToken("initial")
	return &gophercloud.ServiceClient{ProviderClient: provider, Type: "image", Endpoint: mutationBase}
}

type mutationAck struct {
	ImageID, Tag, Action string
	Body                 []byte
	Header               http.Header
	StatusCode           int
}

func mutationCall(s *image.Service, ctx context.Context, operation string, ref resource.Ref, tag string, options ...image.ImageMutationOption) (*mutationAck, error) {
	switch operation {
	case "AddTag", "RemoveTag":
		var value *image.ImageTagResult
		var err error
		if operation == "AddTag" {
			value, err = s.AddTag(ctx, ref, tag, options...)
		} else {
			value, err = s.RemoveTag(ctx, ref, tag, options...)
		}
		if value == nil {
			return nil, err
		}
		return &mutationAck{ImageID: value.ImageID, Tag: value.Tag, Body: value.Body, Header: value.Header, StatusCode: value.StatusCode}, err
	case "DeactivateImage", "ReactivateImage":
		var value *image.ImageActionResult
		var err error
		if operation == "DeactivateImage" {
			value, err = s.DeactivateImage(ctx, ref, options...)
		} else {
			value, err = s.ReactivateImage(ctx, ref, options...)
		}
		if value == nil {
			return nil, err
		}
		return &mutationAck{ImageID: value.ImageID, Action: value.Action, Body: value.Body, Header: value.Header, StatusCode: value.StatusCode}, err
	default:
		panic("unknown mutation fixture operation")
	}
}

var mutationOperations = []struct{ name, method, suffix, action string }{
	{"AddTag", http.MethodPut, "tags/literal", ""},
	{"RemoveTag", http.MethodDelete, "tags/literal", ""},
	{"DeactivateImage", http.MethodPost, "actions/deactivate", "deactivate"},
	{"ReactivateImage", http.MethodPost, "actions/reactivate", "reactivate"},
}

func mutationNative404(target string) error {
	return gophercloud.ErrUnexpectedResponseCode{
		Method: http.MethodDelete, URL: target, Expected: []int{204}, Actual: 404,
		Body: []byte("nested missing"), ResponseHeader: http.Header{"X-Nested": {"true"}},
	}
}

func TestImageMutationsFixedRoutesAndAcknowledgements(t *testing.T) {
	for _, operation := range mutationOperations {
		t.Run(operation.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", "/catalog/unused/")
			client.ResourceBase = cloud.Server.URL + mutationPrefix
			client.MoreHeaders = map[string]string{"X-Source": "captured"}
			client.Microversion = "2.10"
			cloud.Provider.SetToken("latest")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				raw, err := io.ReadAll(r.Body)
				if err != nil || len(raw) != 0 || r.ContentLength != 0 || r.Method != operation.method || r.RequestURI != mutationPrefix+"images/fixed/"+operation.suffix || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Option") != "ordinary" || r.Header.Get("X-Auth-Token") != "latest" || r.Header.Get("OpenStack-API-Version") != "image 2.10" {
					t.Error(r.Method, r.RequestURI, r.Header, string(raw), err)
				}
				w.Header().Set("X-Request-Id", "actual-mutation")
				w.Header().Set("Location", "https://foreign.invalid/images/decoy")
				w.WriteHeader(204)
			})
			value, err := mutationCall(image.New(client), context.Background(), operation.name, resource.ID("fixed"), "literal", image.WithImageMutationHeader("X-Option", "ordinary"))
			if err != nil || value == nil || value.ImageID != "fixed" || value.Action != operation.action || value.StatusCode != 204 || len(value.Body) != 0 || value.Header.Get("X-Request-Id") != "actual-mutation" || value.Header.Get("Location") == "" || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
			if operation.action == "" && value.Tag != "literal" {
				t.Fatal("tag identity changed", value)
			}
		})
	}
	t.Run("repeated dedicated addition never reads or replaces all tags", func(t *testing.T) {
		var calls atomic.Int32
		client := mutationClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Method != http.MethodPut || r.URL.String() != mutationBase+"images/fixed/tags/literal" || r.Body != nil {
				t.Error(r.Method, r.URL, r.Body)
			}
			return mutationWire(204, io.NopCloser(strings.NewReader(""))), nil
		})
		service := image.New(client)
		for range 2 {
			value, err := service.AddTag(context.Background(), resource.ID("fixed"), "literal")
			if err != nil || value == nil || value.Tag != "literal" {
				t.Fatal(value, err)
			}
		}
		if calls.Load() != 2 {
			t.Fatal(calls.Load())
		}
	})
	t.Run("opaque acknowledgement does not decode or retarget", func(t *testing.T) {
		raw := []byte{0, 255, '{', '}', '\n'}
		body := &mutationBody{Reader: bytes.NewReader(raw)}
		wire := mutationWire(204, body)
		wire.Header.Set("Location", "https://foreign.invalid/images/other")
		var calls atomic.Int32
		client := mutationClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.URL.String() != mutationBase+"images/fixed/actions/reactivate" || r.Body != nil {
				t.Error(r.URL, r.Body)
			}
			return wire, nil
		})
		value, err := image.New(client).ReactivateImage(context.Background(), resource.ID("fixed"))
		if err != nil || value == nil || value.ImageID != "fixed" || value.Action != "reactivate" || !bytes.Equal(value.Body, raw) || value.StatusCode != 204 || body.closes.Load() != 1 || calls.Load() != 1 {
			t.Fatal(value, err, body.closes.Load(), calls.Load())
		}
		wire.Header.Set("X-Request-Id", "later")
		value.Body[0] = '!'
		if raw[0] != 0 || value.Header.Get("X-Request-Id") != "actual-mutation" {
			t.Fatal("acknowledgement aliases wire bytes or headers", value)
		}
	})
}

func TestImageMutationsLiteralTagIdentityAndCompletePreflight(t *testing.T) {
	for _, tc := range []struct{ tag, escaped string }{
		{" Mixed Case ", "%20Mixed%20Case%20"}, {" ", "%20"},
		{"한글?x#%2F+@", "%ED%95%9C%EA%B8%80%3Fx%23%252F+@"},
		{"%2F", "%252F"}, {"..x", "..x"}, {"é", "%C3%A9"}, {"e\u0301", "e%CC%81"},
		{strings.Repeat("界", 255), strings.Repeat("%E7%95%8C", 255)},
	} {
		t.Run("literal "+tc.tag, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				want := mutationPrefix + "images/fixed/tags/" + tc.escaped
				decoded, err := url.PathUnescape(strings.TrimPrefix(r.URL.EscapedPath(), mutationPrefix+"images/fixed/tags/"))
				if r.Method != http.MethodPut || r.RequestURI != want || r.URL.EscapedPath() != want || r.URL.RawQuery != "" || decoded != tc.tag || err != nil {
					t.Error(r.Method, r.RequestURI, r.URL, decoded, err)
				}
				w.WriteHeader(204)
			})
			value, err := image.New(cloud.Client("image", mutationPrefix)).AddTag(context.Background(), resource.ID("fixed"), tc.tag)
			if err != nil || value == nil || value.Tag != tc.tag || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
		})
	}
	for _, tag := range []string{"", ".", "..", "slash/tag", "back\\tag", "\t", "\n", string(rune(127)), string(rune(128)), string([]byte{255}), strings.Repeat("界", 256)} {
		t.Run(fmt.Sprintf("unsafe tag %q", tag), func(t *testing.T) {
			var calls, callbacks atomic.Int32
			client := mutationClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return mutationWire(500, io.NopCloser(strings.NewReader("unexpected"))), nil
			})
			value, err := image.New(client).RemoveTag(context.Background(), resource.Name("needle"), tag, func(*image.ImageMutationOpts) error { callbacks.Add(1); return nil })
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 || callbacks.Load() != 0 {
				t.Fatal(value, err, calls.Load(), callbacks.Load())
			}
		})
	}
	for _, mode := range []string{"nil service", "nil client", "nil provider", "wrong type", "foreign base", "query base", "base no slash", "invalid endpoint UTF8", "zero ref", "unsafe ID", "invalid UTF8 ID", "invalid UTF8 name", "nil context", "canceled", "source auth", "source bad key", "source bad value", "source aliases", "source version", "invalid microversion", "nil option", "callback error", "option bad header", "option aliases", "provider replacement", "foreign source after callback"} {
		t.Run(mode, func(t *testing.T) {
			var calls, callbacks atomic.Int32
			client := mutationClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return mutationWire(500, io.NopCloser(strings.NewReader("unexpected"))), nil
			})
			cause := errors.New("preflight callback or cancellation")
			var ctx context.Context = context.Background()
			ref := resource.ID("fixed")
			want := error(resource.ErrInvalidOption)
			wantCallbacks := int32(0)
			option := image.ImageMutationOption(func(*image.ImageMutationOpts) error { callbacks.Add(1); return nil })
			switch mode {
			case "nil client":
				client = nil
			case "nil provider":
				client.ProviderClient = nil
			case "wrong type":
				client.Type, want = "compute", resource.ErrUnsupported
			case "foreign base":
				client.ResourceBase = "https://foreign.invalid/v2/"
			case "query base":
				client.ResourceBase = mutationBase + "?project_id=foreign"
			case "base no slash":
				client.Endpoint = strings.TrimSuffix(mutationBase, "/")
			case "invalid endpoint UTF8":
				client.Endpoint = mutationBase + string([]byte{255}) + "/"
			case "zero ref":
				ref = resource.Ref{}
			case "unsafe ID":
				ref = resource.ID("../escape")
			case "invalid UTF8 ID":
				ref = resource.ID(string([]byte{255}))
			case "invalid UTF8 name":
				ref = resource.Name(string([]byte{255}))
			case "nil context":
				ctx = nil
			case "canceled":
				parent, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx, want = parent, context.Canceled
			case "source auth":
				client.MoreHeaders = map[string]string{"x-auth-token": "foreign"}
			case "source bad key":
				client.MoreHeaders = map[string]string{"Bad Key": "value"}
			case "source bad value":
				client.MoreHeaders = map[string]string{"X-Source": "bad\r\nvalue"}
			case "source aliases":
				client.MoreHeaders = map[string]string{"X-Source": "a", "x-source": "b"}
			case "source version":
				client.MoreHeaders = map[string]string{"OpenStack-API-Version": "image 2.99"}
			case "invalid microversion":
				client.Microversion = "2.10\nbad"
			case "nil option":
				option = nil
			case "callback error":
				want, wantCallbacks = cause, 1
				option = func(*image.ImageMutationOpts) error { callbacks.Add(1); return cause }
			case "option bad header", "option aliases":
				wantCallbacks = 1
				option = func(c *image.ImageMutationOpts) error {
					callbacks.Add(1)
					c.Headers = map[string]string{"X-Option": "bad\nvalue"}
					if mode == "option aliases" {
						c.Headers = map[string]string{"X-Option": "a", "x-option": "b"}
					}
					return nil
				}
			case "provider replacement", "foreign source after callback":
				wantCallbacks = 1
				option = func(*image.ImageMutationOpts) error {
					callbacks.Add(1)
					if mode == "provider replacement" {
						client.ProviderClient = &gophercloud.ProviderClient{}
					} else {
						client.ResourceBase = "https://foreign.invalid/v2/"
					}
					return nil
				}
			}
			service := image.New(client)
			if mode == "nil service" {
				service = nil
			}
			value, err := mutationCall(service, ctx, "DeactivateImage", ref, "", option)
			if value != nil || !errors.Is(err, want) || calls.Load() != 0 || callbacks.Load() != wantCallbacks {
				t.Fatal(value, err, calls.Load(), callbacks.Load())
			}
			if mode == "canceled" && !errors.Is(err, cause) {
				t.Fatal("custom cancellation lost", err)
			}
		})
	}
	for _, header := range []string{"X-Auth-Token", "X-Service-Token", "Authorization", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "Te", "Upgrade", "Accept", "Content-Type", "OpenStack-API-Version", "X-Openstack-Glance-Api-Version", "X-Openstack-Image-Size"} {
		t.Run("protected option "+header, func(t *testing.T) {
			var calls atomic.Int32
			client := mutationClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return mutationWire(500, io.NopCloser(strings.NewReader(""))), nil
			})
			value, err := image.New(client).AddTag(context.Background(), resource.Name("needle"), "literal", image.WithImageMutationHeader(header, "foreign"))
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(value, err, calls.Load())
			}
		})
	}
}

func TestImageMutationsExactNameResolutionAndFailures(t *testing.T) {
	for _, mode := range []string{"one exact match all pages", "safe source changes after first page", "UUID-shaped name", "missing", "ambiguous", "duplicate same ID", "unsafe ID", "empty ID", "late HTTP404", "late malformed JSON", "late native model decode", "late read failure", "transport nested404", "lookup and provider failure", "cancel after lookup"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("lookup failure cause")
			name := "needle"
			if mode == "UUID-shaped name" {
				name = "b2173dd3-7ad6-4362-baa6-a68bce3565cb"
			}
			var sequence []string
			var bodies []*mutationBody
			client := mutationClient(nil)
			client.HTTPClient.Transport = mutationTransport(func(r *http.Request) (*http.Response, error) {
				sequence = append(sequence, r.Method+" "+r.URL.String())
				if r.Header.Get("X-Option") != "lookup-and-mutation" {
					t.Error("prepared lookup header lost", r.Header)
				}
				if mode == "safe source changes after first page" && len(sequence) > 1 && r.Header.Get("X-Auth-Token") != "after-page" {
					t.Error("live lookup/mutation auth lost", r.Header)
				}
				if r.Method != http.MethodGet {
					if r.Method != http.MethodPost || r.URL.String() != mutationBase+"images/fixed/actions/deactivate" || r.Body != nil {
						t.Error("resolved target changed", r.Method, r.URL, r.Body)
					}
					return mutationWire(204, io.NopCloser(strings.NewReader(""))), nil
				}
				if r.URL.Path != mutationPrefix+"images" {
					t.Error("Name lookup performed a GET-first or fallback", r.URL)
				}
				if mode == "transport nested404" || mode == "lookup and provider failure" {
					if mode == "lookup and provider failure" {
						client.ProviderClient = &gophercloud.ProviderClient{}
					}
					return nil, errors.Join(cause, mutationNative404(r.URL.String()))
				}
				first := r.URL.Query().Get("marker") == ""
				payload, code := "{\"images\":[]}", 200
				if first && r.URL.Query().Get("name") != name {
					t.Error("exact literal Name lost", r.URL)
				}
				if mode != "missing" {
					if first {
						payload = fmt.Sprintf("{\"images\":[{\"id\":\"fixed\",\"name\":%q,\"status\":\"killed\",\"protected\":true}],\"next\":\"/v2/images?marker=second\"}", name)
					} else {
						payload = "{\"images\":[{\"id\":\"near\",\"name\":\"needle-suffix\"}]}"
					}
				}
				switch mode {
				case "UUID-shaped name":
					payload = fmt.Sprintf("{\"images\":[{\"id\":\"fixed\",\"name\":%q}]}", name)
				case "ambiguous", "duplicate same ID":
					if !first {
						id := "other"
						if mode == "duplicate same ID" {
							id = "fixed"
						}
						payload = fmt.Sprintf("{\"images\":[{\"id\":%q,\"name\":\"needle\"}]}", id)
					}
				case "unsafe ID", "empty ID":
					id := "../escape"
					if mode == "empty ID" {
						id = ""
					}
					payload = fmt.Sprintf("{\"images\":[{\"id\":%q,\"name\":\"needle\"}]}", id)
				case "late HTTP404":
					if !first {
						code, payload = 404, "wire list failure"
					}
				case "late malformed JSON":
					if !first {
						payload = "{\"images\":"
					}
				case "late native model decode":
					if !first {
						payload = "{\"images\":[{\"id\":\"other\",\"name\":12}]}"
					}
				}
				body := &mutationBody{Reader: strings.NewReader(payload)}
				if mode == "safe source changes after first page" && first {
					body.onClose = func() {
						client.ResourceBase = "https://glance.invalid/later/v2/"
						client.MoreHeaders = map[string]string{"X-Option": "later-source"}
						client.SetToken("after-page")
					}
				}
				if mode == "late read failure" && !first {
					body.Reader = mutationReader(func(p []byte) (int, error) { return copy(p, "{\"images\":"), cause })
				}
				if mode == "cancel after lookup" {
					body.onClose = func() { cancel(cause) }
				}
				bodies = append(bodies, body)
				return mutationWire(code, body), nil
			})
			value, err := image.New(client).DeactivateImage(ctx, resource.Name(name), image.WithImageMutationHeader("X-Option", "lookup-and-mutation"))
			success := mode == "one exact match all pages" || mode == "safe source changes after first page" || mode == "UUID-shaped name"
			if success {
				wantCalls := 3
				if mode == "UUID-shaped name" {
					wantCalls = 2
				}
				if err != nil || value == nil || value.ImageID != "fixed" || len(sequence) != wantCalls {
					t.Fatal(value, err, sequence)
				}
			} else if value != nil || err == nil || strings.Contains(strings.Join(sequence, "|"), "POST ") {
				t.Fatal(value, err, sequence)
			}
			if mode == "missing" && !errors.Is(err, resource.ErrNotFound) || (mode == "ambiguous" || mode == "duplicate same ID") && !errors.Is(err, resource.ErrAmbiguous) || (mode == "unsafe ID" || mode == "empty ID" || mode == "lookup and provider failure") && !errors.Is(err, resource.ErrInvalidOption) || mode == "late HTTP404" && !gophercloud.ResponseCodeIs(err, 404) {
				t.Fatal(value, err, sequence)
			}
			if (mode == "late read failure" || mode == "transport nested404" || mode == "lookup and provider failure" || mode == "cancel after lookup") && !errors.Is(err, cause) {
				t.Fatal("lookup cause lost", err)
			}
			if mode == "cancel after lookup" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			for _, body := range bodies {
				if body.closes.Load() != 1 {
					t.Fatal("native lookup body ownership changed", body.closes.Load())
				}
			}
		})
	}
}

func TestImageMutationsPreparedOptionsAndLiveSource(t *testing.T) {
	t.Run("helper snapshots replacement merge and callback aliases", func(t *testing.T) {
		var calls atomic.Int32
		client := mutationClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Replaced") != "" || r.Header.Get("X-Kept") != "snapshot" || r.Header.Get("X-Merged") != "snapshot" || r.Header.Get("X-Source") != "ordinary-override" || r.Header.Get("X-Retained") != "original" || r.Header.Get("X-Last") != "applied" {
				t.Error(r.Header)
			}
			return mutationWire(204, io.NopCloser(strings.NewReader(""))), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "source"}
		replacement := map[string]string{"X-Kept": "snapshot"}
		merged := map[string]string{"X-Merged": "snapshot", "x-source": "ordinary-override"}
		replaceOption := image.WithImageMutationOpts(image.ImageMutationOpts{Headers: replacement})
		mergeOption := image.WithImageMutationHeaders(merged)
		replacement["X-Kept"], merged["X-Merged"] = "caller-mutated", "caller-mutated"
		var retained *image.ImageMutationOpts
		var callbacks atomic.Int32
		options := []image.ImageMutationOption{
			image.WithImageMutationHeader("X-Replaced", "discarded"),
			replaceOption, mergeOption, nil, nil,
		}
		options[3] = func(c *image.ImageMutationOpts) error {
			callbacks.Add(1)
			c.Headers["X-Retained"] = "original"
			retained = c
			options[4] = nil
			return nil
		}
		options[4] = func(c *image.ImageMutationOpts) error {
			callbacks.Add(1)
			retained.Headers["X-Retained"] = "later-mutated"
			c.Headers["X-Last"] = "applied"
			return nil
		}
		value, err := image.New(client).AddTag(context.Background(), resource.ID("fixed"), "literal", options...)
		if err != nil || value == nil || calls.Load() != 1 || callbacks.Load() != 2 || client.MoreHeaders["X-Source"] != "source" {
			t.Fatal(value, err, calls.Load(), callbacks.Load(), client.MoreHeaders)
		}
	})
	t.Run("standard header helper reused in parallel", func(t *testing.T) {
		var calls atomic.Int32
		client := mutationClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Parallel") != "captured" || r.Body != nil {
				t.Error(r.Header, r.Body)
			}
			return mutationWire(204, io.NopCloser(strings.NewReader(""))), nil
		})
		headers := map[string]string{"X-Parallel": "captured"}
		option := image.WithImageMutationHeaders(headers)
		headers["X-Parallel"] = "changed"
		service := image.New(client)
		var wait sync.WaitGroup
		for _, op := range mutationOperations {
			wait.Add(1)
			go func(operation string) {
				defer wait.Done()
				value, err := mutationCall(service, context.Background(), operation, resource.ID("fixed"), "literal", option)
				if err != nil || value == nil {
					t.Error(value, err)
				}
			}(op.name)
		}
		wait.Wait()
		if calls.Load() != 4 {
			t.Fatal(calls.Load())
		}
	})
	t.Run("captured route headers and live original token", func(t *testing.T) {
		var calls atomic.Int32
		client := mutationClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.URL.String() != mutationBase+"images/fixed/actions/reactivate" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != "latest" {
				t.Error(r.URL, r.Header)
			}
			return mutationWire(204, io.NopCloser(strings.NewReader(""))), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		option := image.ImageMutationOption(func(*image.ImageMutationOpts) error {
			client.ResourceBase = "https://glance.invalid/later/v2/"
			client.MoreHeaders = map[string]string{"X-Source": "later"}
			client.SetToken("latest")
			return nil
		})
		value, err := image.New(client).ReactivateImage(context.Background(), resource.ID("fixed"), option)
		if err != nil || value == nil || calls.Load() != 1 || client.MoreHeaders["X-Source"] != "later" {
			t.Fatal(value, err, calls.Load(), client.MoreHeaders)
		}
	})
}

func TestImageMutationsAcceptedFailuresAndStrictStatuses(t *testing.T) {
	for _, operation := range mutationOperations {
		for _, mode := range []string{"read", "Close", "read Close custom cancellation", "cancel only"} {
			t.Run(operation.name+" "+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("accepted read"), errors.New("accepted Close"), errors.New("accepted cancellation")
				raw := []byte{0, 255, 'a', 'c', 'k'}
				body := &mutationBody{Reader: bytes.NewReader(raw)}
				if strings.Contains(mode, "read") {
					body.Reader = mutationReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
				}
				if strings.Contains(mode, "Close") {
					body.closeErr = closeCause
				}
				wire := mutationWire(204, body)
				body.onClose = func() {
					wire.Header.Set("X-Request-Id", "changed during Close")
					if mode == "read Close custom cancellation" || mode == "cancel only" {
						cancel(cancelCause)
					}
				}
				var calls, retries atomic.Int32
				client := mutationClient(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					if r.Method != operation.method || r.URL.String() != mutationBase+"images/fixed/"+operation.suffix || r.Body != nil {
						t.Error(r.Method, r.URL, r.Body)
					}
					return wire, nil
				})
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries.Add(1)
					return errors.New("accepted body replay")
				}
				value, err := mutationCall(image.New(client), ctx, operation.name, resource.ID("fixed"), "literal")
				var proof *resource.ResponseError
				if value == nil || err == nil || !errors.As(err, &proof) || value.ImageID != "fixed" || value.StatusCode != 204 || !bytes.Equal(value.Body, raw) || value.Header.Get("X-Request-Id") != "actual-mutation" || proof.StatusCode != 204 || !bytes.Equal(proof.Body, raw) || proof.Header.Get("X-Request-Id") != "actual-mutation" || body.closes.Load() != 1 || calls.Load() != 1 || retries.Load() != 0 {
					t.Fatal(value, err, proof, body.closes.Load(), calls.Load(), retries.Load())
				}
				if value.Action != operation.action || operation.action == "" && value.Tag != "literal" {
					t.Fatal("accepted error retargeted acknowledgement identity", value)
				}
				if strings.Contains(mode, "read") && !errors.Is(err, readCause) || strings.Contains(mode, "Close") && !errors.Is(err, closeCause) || (mode == "read Close custom cancellation" || mode == "cancel only") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal("accepted cause lost", err)
				}
				value.Body[0] = '!'
				value.Header.Set("X-Request-Id", "caller")
				if proof.Body[0] != 0 || proof.Header.Get("X-Request-Id") != "actual-mutation" || raw[0] != 0 {
					t.Fatal("result aliases error or input evidence", value, proof)
				}
			})
		}
	}
	for _, code := range []int{200, 201, 202, 206, 300, 400, 403, 404, 413, 429, 498, 500} {
		t.Run(fmt.Sprint("strict actual status ", code), func(t *testing.T) {
			body := &mutationBody{Reader: strings.NewReader("native raw failure")}
			var calls atomic.Int32
			client := mutationClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return mutationWire(code, body), nil
			})
			value, err := image.New(client).RemoveTag(context.Background(), resource.ID("fixed"), "literal")
			var native gophercloud.ErrUnexpectedResponseCode
			var proof *resource.ResponseError
			if value != nil || !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{204}) || native.Method != http.MethodDelete || native.URL != mutationBase+"images/fixed/tags/literal" || string(native.Body) != "native raw failure" || native.ResponseHeader.Get("X-Request-Id") != "actual-mutation" || errors.As(err, &proof) || calls.Load() != 1 || body.closes.Load() != 1 {
				t.Fatal(value, err, native, calls.Load(), body.closes.Load())
			}
		})
	}
	t.Run("transport and callback nested404 remain failures", func(t *testing.T) {
		for _, callback := range []bool{false, true} {
			cause := errors.New("nested missing custom cause")
			nested := errors.Join(mutationNative404(mutationBase))
			var calls atomic.Int32
			client := mutationClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if !callback {
					return nil, errors.Join(cause, nested)
				}
				return mutationWire(503, io.NopCloser(strings.NewReader("retry original"))), nil
			})
			if callback {
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					return errors.Join(cause, nested)
				}
			}
			value, err := image.New(client).RemoveTag(context.Background(), resource.ID("fixed"), "literal")
			// Native ResponseCodeIs observes the first HTTP error in a join;
			// preserve the separate nested404 cause alongside the original503.
			if value != nil || !errors.Is(err, cause) || !errors.Is(err, nested) || !gophercloud.ResponseCodeIs(nested, 404) || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
			if callback && !gophercloud.ResponseCodeIs(err, 503) {
				t.Fatal("original callback status lost", err)
			}
		}
	})
}

func TestImageMutationsProviderHooksAndNativeCompatibility(t *testing.T) {
	t.Run("prebody reauth backoff retry retain target and live auth", func(t *testing.T) {
		var calls, reauth, backoff, retries atomic.Int32
		var bodies []*mutationBody
		client := mutationClient(nil)
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		client.ReauthFunc = func(context.Context) error { reauth.Add(1); client.SetToken("reauth"); return nil }
		client.RetryBackoffFunc = func(_ context.Context, native *gophercloud.ErrUnexpectedResponseCode, _ error, _ uint) error {
			backoff.Add(1)
			if native.Actual != 429 || native.Method != http.MethodPost || native.URL != mutationBase+"images/fixed/actions/deactivate" {
				t.Error(native)
			}
			client.SetToken("backoff")
			return nil
		}
		client.RetryFunc = func(_ context.Context, method, target string, options *gophercloud.RequestOpts, original error, _ uint) error {
			retries.Add(1)
			if method != http.MethodPost || target != mutationBase+"images/fixed/actions/deactivate" || !gophercloud.ResponseCodeIs(original, 503) || options.JSONBody != nil || options.RawBody != nil {
				return errors.New("unexpected retry fixture")
			}
			client.SetToken("retry")
			return nil
		}
		originalRetry := reflect.ValueOf(client.RetryFunc).Pointer()
		client.HTTPClient.Transport = mutationTransport(func(r *http.Request) (*http.Response, error) {
			n := int(calls.Add(1)) - 1
			codes, tokens := []int{401, 429, 503, 204}, []string{"initial", "reauth", "backoff", "retry"}
			if n >= len(codes) {
				return nil, errors.New("unexpected mutation replay")
			}
			if r.Method != http.MethodPost || r.URL.String() != mutationBase+"images/fixed/actions/deactivate" || r.Body != nil || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != tokens[n] {
				t.Error(r.Method, r.URL, r.Body, r.Header)
			}
			body := &mutationBody{Reader: strings.NewReader(fmt.Sprint(codes[n]))}
			bodies = append(bodies, body)
			return mutationWire(codes[n], body), nil
		})
		value, err := image.New(client).DeactivateImage(context.Background(), resource.ID("fixed"))
		if err != nil || value == nil || string(value.Body) != "204" || calls.Load() != 4 || reauth.Load() != 1 || backoff.Load() != 1 || retries.Load() != 1 || reflect.ValueOf(client.RetryFunc).Pointer() != originalRetry {
			t.Fatal(value, err, calls.Load(), reauth.Load(), backoff.Load(), retries.Load())
		}
		for _, body := range bodies {
			if body.closes.Load() != 1 {
				t.Fatal(body.closes.Load())
			}
		}
	})
	t.Run("configured transport failure retry", func(t *testing.T) {
		cause := errors.New("prebody transport failure")
		var calls, retries atomic.Int32
		client := mutationClient(nil)
		client.HTTPClient.Transport = mutationTransport(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPut || r.URL.String() != mutationBase+"images/fixed/tags/literal" || r.Body != nil {
				t.Error(r.Method, r.URL, r.Body)
			}
			if calls.Add(1) == 1 {
				return nil, cause
			}
			if r.Header.Get("X-Auth-Token") != "after-transport" {
				t.Error(r.Header)
			}
			return mutationWire(204, io.NopCloser(strings.NewReader(""))), nil
		})
		client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
			retries.Add(1)
			if !errors.Is(original, cause) {
				return original
			}
			client.SetToken("after-transport")
			return nil
		}
		value, err := image.New(client).AddTag(context.Background(), resource.ID("fixed"), "literal")
		if err != nil || value == nil || calls.Load() != 2 || retries.Load() != 1 {
			t.Fatal(value, err, calls.Load(), retries.Load())
		}
	})
	for _, change := range []string{"KeepResponseBody", "JSONResponse", "JSONBody null", "RawBody", "unsupported JSONBody", "expanded OkCodes"} {
		t.Run("shared retry ownership "+change, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			callbackCause, readCause, closeCause, cancelCause := errors.New("callback cause"), errors.New("expanded read"), errors.New("expanded Close"), errors.New("expanded cancellation")
			var calls, retries, borrowedReads atomic.Int32
			borrowed := &mutationBody{Reader: mutationReader(func([]byte) (int, error) { borrowedReads.Add(1); return 0, io.EOF })}
			var bodies []*mutationBody
			client := mutationClient(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodPost || r.URL.String() != mutationBase+"images/fixed/actions/reactivate" || r.Body != nil {
					t.Error(r.Method, r.URL, r.Body)
				}
				n := calls.Add(1)
				if n > 2 {
					return nil, errors.New("extra retry")
				}
				code := 503
				body := &mutationBody{Reader: strings.NewReader("rejected503")}
				if n == 2 {
					code = 202
					body.Reader = mutationReader(func(p []byte) (int, error) { return copy(p, "unexpected202"), readCause })
					body.closeErr = closeCause
					body.onClose = func() { cancel(cancelCause) }
				}
				bodies = append(bodies, body)
				return mutationWire(code, body), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, original error, _ uint) error {
				retries.Add(1)
				if !gophercloud.ResponseCodeIs(original, 503) || !options.KeepResponseBody || options.JSONResponse != nil || options.JSONBody != nil || options.RawBody != nil {
					t.Error(options, original)
				}
				switch change {
				case "KeepResponseBody":
					options.KeepResponseBody = false
				case "JSONResponse":
					options.JSONResponse = new(any)
				case "JSONBody null":
					options.JSONBody = json.RawMessage("null")
				case "RawBody":
					options.RawBody = borrowed
				case "unsupported JSONBody":
					options.JSONBody = make(chan int)
				case "expanded OkCodes":
					options.OkCodes = []int{204, 202}
					return nil
				}
				return callbackCause
			}
			originalRetry := reflect.ValueOf(client.RetryFunc).Pointer()
			value, err := image.New(client).ReactivateImage(ctx, resource.ID("fixed"))
			if value != nil || err == nil || retries.Load() != 1 || borrowedReads.Load() != 0 || borrowed.closes.Load() != 0 || reflect.ValueOf(client.RetryFunc).Pointer() != originalRetry {
				t.Fatal(value, err, calls.Load(), retries.Load(), borrowedReads.Load(), borrowed.closes.Load())
			}
			if change == "expanded OkCodes" {
				var native gophercloud.ErrUnexpectedResponseCode
				var proof *resource.ResponseError
				if calls.Load() != 2 || !errors.As(err, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, []int{204}) || string(native.Body) != "unexpected202" || native.ResponseHeader.Get("X-Request-Id") != "actual-mutation" || errors.As(err, &proof) || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) {
					t.Fatal("unexpected actual status ownership lost", err, native, calls.Load())
				}
			} else if calls.Load() != 1 || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, callbackCause) || !gophercloud.ResponseCodeIs(err, 503) {
				t.Fatal("body change lost original failure", err, calls.Load())
			}
			if change == "unsupported JSONBody" {
				var encoding *json.UnsupportedTypeError
				if !errors.As(err, &encoding) {
					t.Fatal("encoding cause lost", err)
				}
			}
			for _, body := range bodies {
				if body.closes.Load() != 1 {
					t.Fatal(body.closes.Load())
				}
			}
		})
	}
	t.Run("failed native reauth preserves direct fields", func(t *testing.T) {
		cause := errors.New("reauth cause")
		body := &mutationBody{Reader: strings.NewReader("original401")}
		var calls, reauth atomic.Int32
		client := mutationClient(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return mutationWire(401, body), nil
		})
		client.ReauthFunc = func(context.Context) error { reauth.Add(1); return errors.Join(cause, mutationNative404(mutationBase)) }
		value, err := image.New(client).RemoveTag(context.Background(), resource.ID("fixed"), "literal")
		var native *gophercloud.ErrUnableToReauthenticate
		if value != nil || !errors.As(err, &native) || !errors.Is(native.ErrReauth, cause) || !gophercloud.ResponseCodeIs(native.ErrReauth, 404) || !gophercloud.ResponseCodeIs(native.ErrOriginal, 401) || calls.Load() != 1 || reauth.Load() != 1 || body.closes.Load() != 1 {
			t.Fatal(value, err, calls.Load(), reauth.Load(), body.closes.Load())
		}
	})
	for _, mode := range []string{"foreign origin", "other path", "other query", "POST method changes", "identical target"} {
		t.Run("redirect "+mode, func(t *testing.T) {
			var calls, redirects atomic.Int32
			var bodies []*mutationBody
			target := mutationBase + "images/fixed/actions/deactivate"
			client := mutationClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if r.Method != http.MethodPost || r.URL.String() != target {
					t.Error("redirect escaped fixed scope", r.Method, r.URL)
				}
				body := &mutationBody{Reader: strings.NewReader("")}
				bodies = append(bodies, body)
				if n == 2 {
					return mutationWire(204, body), nil
				}
				code, location := 307, target
				switch mode {
				case "foreign origin":
					location = "https://foreign.invalid/images/other"
				case "other path":
					location = mutationBase + "images/other/actions/deactivate"
				case "other query":
					code, location = 308, target+"?project_id=other"
				case "POST method changes":
					code = 302
				}
				wire := mutationWire(code, body)
				wire.Header.Set("Location", location)
				return wire, nil
			})
			client.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects.Add(1); return nil }
			originalPolicy := reflect.ValueOf(client.HTTPClient.CheckRedirect).Pointer()
			value, err := image.New(client).DeactivateImage(context.Background(), resource.ID("fixed"))
			if mode == "identical target" {
				if err != nil || value == nil || value.StatusCode != 204 || calls.Load() != 2 {
					t.Fatal(value, err, calls.Load())
				}
			} else if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
			if redirects.Load() != 1 || reflect.ValueOf(client.HTTPClient.CheckRedirect).Pointer() != originalPolicy {
				t.Fatal("original redirect policy changed", redirects.Load())
			}
			for _, body := range bodies {
				if body.closes.Load() != 1 {
					t.Fatal(body.closes.Load())
				}
			}
		})
	}
	t.Run("native full-set tags PATCH retains separate contract", func(t *testing.T) {
		var calls atomic.Int32
		client := mutationClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			raw, err := io.ReadAll(r.Body)
			if err != nil || r.Method != http.MethodPatch || r.URL.String() != mutationBase+"images/fixed" || r.Header.Get("Content-Type") != "application/openstack-images-v2.1-json-patch" || string(raw) != "[{\"op\":\"replace\",\"path\":\"/tags\",\"value\":[\"whole\",\"set\"]}]" {
				t.Error(r.Method, r.URL, r.Header, string(raw), err)
			}
			return mutationWire(200, io.NopCloser(strings.NewReader("{\"id\":\"fixed\",\"tags\":[\"whole\",\"set\"],\"status\":\"deactivated\"}"))), nil
		})
		value, err := sdkimages.New(client).Update(context.Background(), "fixed", sdkimages.UpdateOpts{sdkimages.ReplaceImageTags{NewTags: []string{"whole", "set"}}})
		if err != nil || value == nil || value.ID != "fixed" || !reflect.DeepEqual(value.Tags, []string{"whole", "set"}) || value.Status != sdkimages.ImageStatusDeactivated || calls.Load() != 1 {
			t.Fatal(value, err, calls.Load())
		}
	})
}
