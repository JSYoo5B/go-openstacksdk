package image

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type deleteCoreTransport func(*http.Request) (*http.Response, error)

func (value deleteCoreTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := value(request)
	if response != nil && response.Request == nil {
		response.Request = request
	}
	return response, err
}

type deleteCoreBody struct {
	reader   io.Reader
	closes   int
	closeErr error
}

func (value *deleteCoreBody) Read(buffer []byte) (int, error) { return value.reader.Read(buffer) }
func (value *deleteCoreBody) Close() error                    { value.closes++; return value.closeErr }

type deleteCoreReader func([]byte) (int, error)

func (value deleteCoreReader) Read(buffer []byte) (int, error) { return value(buffer) }

func deleteCoreClient(transport deleteCoreTransport) *gophercloud.ServiceClient {
	provider := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: transport}}
	provider.UseTokenLock()
	provider.SetToken("before")
	return &gophercloud.ServiceClient{ProviderClient: provider, Type: "image", Endpoint: "https://example.test/catalog/", ResourceBase: "https://example.test/reverse/glance/v2/"}
}

func deleteCoreHTTP(status int, body io.ReadCloser, headers http.Header) *http.Response {
	if headers == nil {
		headers = make(http.Header)
	}
	if headers.Get("Content-Type") == "" {
		headers.Set("Content-Type", "application/json")
	}
	return &http.Response{StatusCode: status, Body: body, Header: headers}
}

func TestDeleteCoreFixedScopeAndOwnedAcknowledgement(t *testing.T) {
	for _, store := range []string{"", "저장소&=fast"} {
		t.Run(store, func(t *testing.T) {
			const id = "image-한글"
			body := &deleteCoreBody{reader: bytes.NewReader([]byte{'r', 'a', 'w', 255})}
			header := http.Header{"X-Actual": {"before"}, "Location": {"https://foreign.test/elsewhere"}}
			var retained *DeleteImageOpts
			var client *gophercloud.ServiceClient
			var calls, callbacks int
			client = deleteCoreClient(func(r *http.Request) (*http.Response, error) {
				calls++
				path := "/reverse/glance/v2/images/" + url.PathEscape(id)
				if store != "" {
					path = "/reverse/glance/v2/stores/" + url.PathEscape(store) + "/" + url.PathEscape(id)
				}
				if r.Method != http.MethodDelete || r.URL.EscapedPath() != path || r.URL.RawQuery != "" || r.Body != nil || r.Header.Get("X-Snapshot") != "before" || r.Header.Get("X-Auth-Token") != "after" || callbacks != 1 {
					t.Errorf("request=%s %s header=%v body=%v callbacks=%d", r.Method, r.URL, r.Header, r.Body, callbacks)
				}
				*retained.IgnoreMissing = false
				if retained.StoreID != nil {
					*retained.StoreID = "retained mutation"
				}
				return deleteCoreHTTP(204, body, header), nil
			})
			client.MoreHeaders = map[string]string{"X-Snapshot": "before"}
			result, err := New(client).DeleteImage(context.Background(), resource.ID(id), func(value *DeleteImageOpts) error {
				callbacks++
				retained = value
				ignore := true
				value.IgnoreMissing = &ignore
				if store != "" {
					selected := store
					value.StoreID = &selected
				}
				client.MoreHeaders["X-Snapshot"] = "changed"
				client.ResourceBase = "https://example.test/later-valid/"
				client.ProviderClient.SetToken("after")
				return nil
			})
			if err != nil || result == nil || result.ImageID != id || result.StoreID != store || result.StatusCode != 204 || !bytes.Equal(result.Body, []byte{'r', 'a', 'w', 255}) || result.Header.Get("X-Actual") != "before" || calls != 1 || body.closes != 1 {
				t.Fatalf("result=%+v err=%v calls=%d closes=%d", result, err, calls, body.closes)
			}
			header.Set("X-Actual", "wire mutation")
			if result.Header.Get("X-Actual") != "before" {
				t.Fatal("wire header aliases result")
			}
		})
	}
}

func TestDeleteCoreOwnedMissingPreservesBodyAndContextFailures(t *testing.T) {
	readCause, closeCause, customCause := errors.New("read failed"), errors.New("close failed"), errors.New("caller canceled")
	for _, test := range []struct {
		name              string
		strict, canceled  bool
		readErr, closeErr error
	}{
		{name: "ignored"}, {name: "strict", strict: true},
		{name: "read failure", readErr: readCause}, {name: "close failure", closeErr: closeCause},
		{name: "read close cancellation", readErr: readCause, closeErr: closeCause, canceled: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			body := &deleteCoreBody{reader: deleteCoreReader(func(buffer []byte) (int, error) {
				if test.canceled {
					cancel(customCause)
				}
				err := test.readErr
				if err == nil {
					err = io.EOF
				}
				return copy(buffer, "missing raw body"), err
			}), closeErr: test.closeErr}
			var calls, retries int
			client := deleteCoreClient(func(r *http.Request) (*http.Response, error) {
				calls++
				return deleteCoreHTTP(404, body, http.Header{"X-Actual": {"missing"}}), nil
			})
			client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return errors.New("handled404 must bypass native retry")
			}
			result, err := New(client).DeleteImage(ctx, resource.ID("fixed"), WithDeleteImageStore("fast"), WithDeleteImageIgnoreMissing(!test.strict))
			failedBody := test.readErr != nil || test.closeErr != nil || test.canceled
			if result != nil || calls != 1 || retries != 0 || body.closes != 1 {
				t.Fatalf("result=%+v err=%v calls=%d retries=%d closes=%d", result, err, calls, retries, body.closes)
			}
			if !failedBody && !test.strict {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != 404 || native.Method != http.MethodDelete || native.URL != "https://example.test/reverse/glance/v2/stores/fast/fixed" || !reflect.DeepEqual(native.Expected, []int{204}) || string(native.Body) != "missing raw body" || native.ResponseHeader.Get("X-Actual") != "missing" || errors.Is(err, resource.ErrNotFound) != test.strict {
				t.Fatalf("native=%+v err=%v", native, err)
			}
			for _, cause := range []error{test.readErr, test.closeErr} {
				if cause != nil && !errors.Is(err, cause) {
					t.Fatalf("cause %v lost: %v", cause, err)
				}
			}
			if test.canceled && (!errors.Is(err, context.Canceled) || !errors.Is(err, customCause)) {
				t.Fatalf("context causes lost: %v", err)
			}
		})
	}
}

func TestDeleteCoreAcknowledgementAndErrorEvidenceAreIndependent(t *testing.T) {
	nested404 := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Body: []byte("decoy")}
	readCause := errors.Join(io.ErrUnexpectedEOF, nested404)
	closeCause, customCause := errors.New("close failed"), errors.New("caller canceled")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	body := &deleteCoreBody{reader: deleteCoreReader(func(buffer []byte) (int, error) {
		cancel(customCause)
		return copy(buffer, "partial acknowledgement"), readCause
	}), closeErr: closeCause}
	header := http.Header{"X-Actual": {"accepted"}}
	var calls int
	client := deleteCoreClient(func(r *http.Request) (*http.Response, error) {
		calls++
		return deleteCoreHTTP(204, body, header), nil
	})
	result, err := New(client).DeleteImage(ctx, resource.ID("fixed"))
	var accepted *resource.ResponseError
	var operation *resource.OperationError
	if result == nil || result.StatusCode != 204 || result.ImageID != "fixed" || string(result.Body) != "partial acknowledgement" || result.Header.Get("X-Actual") != "accepted" || !errors.As(err, &accepted) || accepted.StatusCode != 204 || !errors.As(err, &operation) || operation.Operation != "DeleteImage" || operation.Resource != "image" || errors.Is(err, resource.ErrNotFound) || calls != 1 || body.closes != 1 {
		t.Fatalf("result=%+v accepted=%+v err=%v calls=%d closes=%d", result, accepted, err, calls, body.closes)
	}
	for _, cause := range []error{io.ErrUnexpectedEOF, closeCause, context.Canceled, customCause} {
		if !errors.Is(err, cause) {
			t.Fatalf("cause %v lost: %v", cause, err)
		}
	}
	result.Body[0] = '!'
	result.Header.Set("X-Actual", "result mutation")
	if string(accepted.Body) != "partial acknowledgement" || accepted.Header.Get("X-Actual") != "accepted" || header.Get("X-Actual") != "accepted" {
		t.Fatal("accepted result aliases error or wire evidence")
	}
}

func TestDeleteCoreNameAbsenceAndSourceFailureBoundaries(t *testing.T) {
	for _, mode := range []string{"unique", "missing", "strict missing", "duplicate", "list404", "list custom404", "provider changed", "invalid source", "canceled absence"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("lookup failed")
			var client *gophercloud.ServiceClient
			var gets, deletes int
			client = deleteCoreClient(func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodDelete {
					deletes++
					if r.URL.Path != "/reverse/glance/v2/stores/fast/fixed" {
						t.Error("wrong fixed scope", r.URL)
					}
					return deleteCoreHTTP(204, io.NopCloser(strings.NewReader("")), http.Header{}), nil
				}
				gets++
				if r.Method != http.MethodGet || r.URL.Query().Get("name") != "exact" {
					t.Error("wrong lookup", r.Method, r.URL)
				}
				if mode == "list404" {
					return deleteCoreHTTP(404, io.NopCloser(strings.NewReader("list missing")), http.Header{}), nil
				}
				if mode == "list custom404" {
					return nil, errors.Join(gophercloud.ErrUnexpectedResponseCode{Actual: 404}, cause)
				}
				if gets == 1 {
					images := `[{"id":"near","name":"near-exact"}]`
					if mode == "duplicate" {
						images = `[{"id":"first","name":"exact"}]`
					}
					return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(`{"images":`+images+`,"next":"/v2/images?name=exact&page=2"}`)), http.Header{}), nil
				}
				images := `[]`
				if mode == "unique" || mode == "duplicate" {
					images = `[{"id":"fixed","name":"exact"}]`
				}
				if mode == "provider changed" {
					client.ProviderClient = &gophercloud.ProviderClient{}
				}
				if mode == "invalid source" {
					client.Endpoint = "https://foreign.test/"
				}
				if mode == "canceled absence" {
					cancel(cause)
				}
				return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(`{"images":`+images+`}`)), http.Header{}), nil
			})
			result, err := New(client).DeleteImage(ctx, resource.Name("exact"), WithDeleteImageStore("fast"), WithDeleteImageIgnoreMissing(mode != "strict missing"))
			if mode == "unique" {
				if err != nil || result == nil || gets != 2 || deletes != 1 {
					t.Fatalf("result=%+v err=%v gets=%d deletes=%d", result, err, gets, deletes)
				}
				return
			}
			if result != nil || deletes != 0 {
				t.Fatalf("result=%+v err=%v deletes=%d", result, err, deletes)
			}
			switch mode {
			case "missing":
				if err != nil || gets != 2 {
					t.Fatalf("ignored absence err=%v gets=%d", err, gets)
				}
			case "strict missing":
				if !errors.Is(err, resource.ErrNotFound) || gets != 2 {
					t.Fatalf("strict absence err=%v gets=%d", err, gets)
				}
			case "duplicate":
				if !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(err)
				}
			case "list404":
				if !gophercloud.ResponseCodeIs(err, 404) || errors.Is(err, resource.ErrNotFound) {
					t.Fatal("HTTP lookup failure became logical absence", err)
				}
			case "list custom404", "canceled absence":
				if !errors.Is(err, cause) {
					t.Fatal("lookup/context cause lost", err)
				}
			default:
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestDeleteCoreActualProtocolAndNativeRetryBoundaries(t *testing.T) {
	for _, status := range []int{200, 202, 206, 302, 307, 403, 409} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			body := &deleteCoreBody{reader: strings.NewReader("native rejection")}
			var calls int
			client := deleteCoreClient(func(r *http.Request) (*http.Response, error) {
				calls++
				return deleteCoreHTTP(status, body, http.Header{"Location": {"https://foreign.test/elsewhere"}, "X-Actual": {"rejected"}}), nil
			})
			result, err := New(client).DeleteImage(context.Background(), resource.ID("fixed"))
			var native gophercloud.ErrUnexpectedResponseCode
			if result != nil || !errors.As(err, &native) || native.Actual != status || string(native.Body) != "native rejection" || native.ResponseHeader.Get("X-Actual") != "rejected" || calls != 1 || body.closes != 1 {
				t.Fatalf("result=%+v native=%+v err=%v calls=%d closes=%d", result, native, err, calls, body.closes)
			}
		})
	}
	for _, broaden := range []bool{false, true} {
		t.Run(fmt.Sprintf("retry broadens=%v", broaden), func(t *testing.T) {
			bodies := []*deleteCoreBody{{reader: strings.NewReader("retry")}, {reader: strings.NewReader("final")}}
			var calls, retries int
			client := deleteCoreClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != http.MethodDelete || r.URL.String() != "https://example.test/reverse/glance/v2/images/fixed" {
					t.Fatal("retry retargeted", r.Method, r.URL)
				}
				status := 503
				if calls == 2 {
					status = 204
					if broaden {
						status = 200
					}
				}
				return deleteCoreHTTP(status, bodies[calls-1], http.Header{}), nil
			})
			client.ProviderClient.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				retries++
				if broaden {
					opts.OkCodes = []int{200}
				}
				return nil
			}
			result, err := New(client).DeleteImage(context.Background(), resource.ID("fixed"))
			if calls != 2 || retries != 1 || bodies[0].closes != 1 || bodies[1].closes != 1 {
				t.Fatalf("calls=%d retries=%d closes=%d/%d", calls, retries, bodies[0].closes, bodies[1].closes)
			}
			if broaden {
				if result != nil || !gophercloud.ResponseCodeIs(err, 200) {
					t.Fatalf("configured retry fabricated acknowledgement: result=%+v err=%v", result, err)
				}
			} else if err != nil || result == nil || result.StatusCode != 204 {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}
