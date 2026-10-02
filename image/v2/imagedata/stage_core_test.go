package imagedata

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	nativeimages "github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"gophercloudsdk/image/v2/images"
	"gophercloudsdk/resource"
)

type stageCoreTransport func(*http.Request) (*http.Response, error)

func (run stageCoreTransport) RoundTrip(req *http.Request) (*http.Response, error) { return run(req) }

func stageCoreClient(transport http.RoundTripper) *gophercloud.ServiceClient {
	return &gophercloud.ServiceClient{Type: "image", Endpoint: "https://example.invalid/v2/", ResourceBase: "https://example.invalid/reverse/v2/", ProviderClient: &gophercloud.ProviderClient{TokenID: "first", HTTPClient: http.Client{Transport: transport}}}
}

func stageCoreResponse(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"X-Evidence": {fmt.Sprint(code)}}, Body: io.NopCloser(strings.NewReader(body))}
}

type stageCoreReader struct {
	input                *strings.Reader
	reads, seeks, closes int
}

func (reader *stageCoreReader) Read(data []byte) (int, error) {
	reader.reads++
	return reader.input.Read(data)
}
func (reader *stageCoreReader) Seek(offset int64, whence int) (int64, error) {
	reader.seeks++
	return reader.input.Seek(offset, whence)
}
func (reader *stageCoreReader) Close() error { reader.closes++; return nil }

func TestStageCoreKnownSnapshotBorrowedCursorAndHeaderPhases(t *testing.T) {
	for _, size := range []*int64{nil, stageTestSize(0), stageTestSize(4)} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			seed := &images.Image{ID: "fixed", Status: images.ImageStatus("queued")}
			reader := &stageCoreReader{input: strings.NewReader("prefix")}
			_, _ = reader.input.Seek(2, io.SeekStart)
			var captured *StageOpts
			var calls int
			var client *gophercloud.ServiceClient
			client = stageCoreClient(stageCoreTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Header.Get("X-Note") != "initial" {
					t.Errorf("lost header snapshot: %#v", req.Header)
				}
				if calls == 1 {
					if req.Method != http.MethodPut || req.URL.Path != "/reverse/v2/images/fixed/stage" || req.Header.Get("Content-Type") != "application/octet-stream" || req.Header.Get("Accept") != "" || req.GetBody != nil || req.ContentLength != 0 {
						t.Errorf("binary request: %s %s %#v length%d replay%v", req.Method, req.URL, req.Header, req.ContentLength, req.GetBody != nil)
					}
					wantedSize := ""
					if size != nil {
						wantedSize = fmt.Sprint(*size)
					}
					if req.Header.Get("X-OpenStack-Image-Size") != wantedSize {
						t.Errorf("size header %q want%q", req.Header.Get("X-OpenStack-Image-Size"), wantedSize)
					}
					if _, ok := req.Body.(io.Seeker); ok {
						t.Error("request exposed caller seeker")
					}
					body, err := io.ReadAll(req.Body)
					if err != nil || string(body) != "efix" {
						t.Errorf("current cursor body %q %v", body, err)
					}
					_ = req.Body.Close()
					captured.Headers["X-Note"] = "during"
					if captured.Size != nil {
						*captured.Size = 99
					}
					client.ProviderClient.SetToken("live-followup")
					return stageCoreResponse(204, "actual ack"), nil
				}
				if req.Method != http.MethodGet || req.URL.Path != "/reverse/v2/images/fixed" || req.Header.Get("Accept") != "metadata-accept" || req.Header.Get("Content-Type") != "metadata-type" || req.Header.Get("X-OpenStack-Image-Size") != "" || req.Header.Get("X-Auth-Token") != "live-followup" {
					t.Errorf("metadata phase %s %s %#v", req.Method, req.URL, req.Header)
				}
				return stageCoreResponse(200, `{"id":"response-decoy","status":"uploading"}`), nil
			}))
			client.MoreHeaders = map[string]string{"accept": "metadata-accept", "content-type": "metadata-type"}
			options := []StageOption{WithStageOpts(StageOpts{Size: size, Headers: map[string]string{"X-Note": "initial"}}), func(config *StageOpts) error {
				captured = config
				seed.ID, seed.Status = "changed", images.ImageStatus("active")
				return nil
			}}
			result, err := New(client).StageKnownImage(context.Background(), seed, reader, options...)
			if err != nil || calls != 2 || result == nil || result.ImageID != "fixed" || result.Image == nil || result.Image.ID != "response-decoy" || result.StatusCode != 200 || result.Acknowledgement.StatusCode != 204 || string(result.Acknowledgement.Body) != "actual ack" || reader.seeks != 0 || reader.closes != 0 || reader.reads == 0 || len(options) != 2 {
				t.Fatalf("result%#v err%v calls%d reader%#v", result, err, calls, reader)
			}
			result.Header.Set("X-Evidence", "outer changed")
			result.Body[0] = '!'
			if result.Acknowledgement.Header.Get("X-Evidence") != "204" || string(result.Acknowledgement.Body) != "actual ack" {
				t.Fatal("metadata and ack share evidence")
			}
		})
	}
}

func TestStageCoreFreshCanonicalQueuedAndWholeNativeDecode(t *testing.T) {
	for _, test := range []struct {
		name, body string
		valid      bool
	}{
		{"canonical", `{"id":"decoy","status":"queued"}`, true},
		{"missing", `{}`, false}, {"case-only", `{"Status":"queued"}`, false},
		{"uppercase", `{"status":"QUEUED"}`, false}, {"prefix", `{"status":"queued-pending"}`, false},
		{"null", `{"status":null}`, false}, {"typed", `{"status":false}`, false},
		{"native-type", `{"status":"queued","min_ram":"bad"}`, false},
		{"native-time", `{"status":"queued","created_at":"bad"}`, false},
		{"not-object", `[]`, false}, {"trailing", `{"status":"queued"}{}`, false},
		{"utf8", "{\"status\":\"queued\",\"vendor\":\"" + string([]byte{255}) + "\"}", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls, puts int
			reader := &stageCoreReader{input: strings.NewReader("data")}
			client := stageCoreClient(stageCoreTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return stageCoreResponse(200, test.body), nil
				}
				if req.Method == http.MethodPut {
					puts++
					if req.URL.Path != "/reverse/v2/images/requested/stage" {
						t.Error("response ID rerouted PUT")
					}
					_, _ = io.ReadAll(req.Body)
					return stageCoreResponse(204, ""), nil
				}
				return stageCoreResponse(200, `{"status":"uploading"}`), nil
			}))
			result, err := New(client).StageImage(context.Background(), resource.ID("requested"), reader)
			if test.valid {
				if err != nil || result == nil || puts != 1 || calls != 3 {
					t.Fatalf("valid: %#v %v calls%d puts%d", result, err, calls, puts)
				}
			} else {
				var evidence *resource.ResponseError
				if result != nil || err == nil || calls != 1 || puts != 0 || reader.reads != 0 || !errors.As(err, &evidence) || string(evidence.Body) != test.body || evidence.StatusCode != 200 {
					t.Fatalf("bad metadata: %#v %v calls%d puts%d reads%d proof%#v", result, err, calls, puts, reader.reads, evidence)
				}
			}
		})
	}
}

func TestStageCoreBinaryFailuresDoNotReplayOrRedirect(t *testing.T) {
	transportCause := errors.New("transport failed after prefix")
	for _, code := range []int{401, 429, 498, 503, 0} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var requests, callbacks int
			reader := &stageCoreReader{input: strings.NewReader("payload")}
			client := stageCoreClient(stageCoreTransport(func(req *http.Request) (*http.Response, error) {
				requests++
				prefix := make([]byte, 3)
				_, _ = io.ReadFull(req.Body, prefix)
				_ = req.Body.Close()
				if string(prefix) != "pay" {
					t.Errorf("replayed/current cursor %q", prefix)
				}
				if code == 0 {
					return nil, transportCause
				}
				return stageCoreResponse(code, "actual failure"), nil
			}))
			client.ProviderClient.ReauthFunc = func(context.Context) error { callbacks++; return errors.New("unexpected reauth") }
			client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				callbacks++
				return errors.New("unexpected retry")
			}
			client.ProviderClient.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				callbacks++
				return errors.New("unexpected backoff")
			}
			client.ProviderClient.MaxBackoffRetries = 5
			result, err := New(client).StageKnownImage(context.Background(), &images.Image{ID: "fixed", Status: images.ImageStatus("queued")}, reader)
			if result != nil || err == nil || requests != 1 || callbacks != 0 || reader.seeks != 0 || reader.closes != 0 || client.ProviderClient.ReauthFunc == nil || client.ProviderClient.RetryFunc == nil || client.ProviderClient.RetryBackoffFunc == nil || client.ProviderClient.MaxBackoffRetries != 5 {
				t.Fatalf("binary replay/policy: %#v %v requests%d callbacks%d reader%#v", result, err, requests, callbacks, reader)
			}
			if code == 0 && !errors.Is(err, transportCause) || code != 0 && !gophercloud.ResponseCodeIs(err, code) {
				t.Fatalf("lost native cause: %v", err)
			}
		})
	}
	var calls, sourceRedirects atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		writer.Header().Set("Location", "/redirected")
		writer.WriteHeader(http.StatusSeeOther)
	}))
	defer server.Close()
	client := &gophercloud.ServiceClient{Type: "image", Endpoint: server.URL + "/v2/", ProviderClient: &gophercloud.ProviderClient{HTTPClient: *server.Client()}}
	client.ProviderClient.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { sourceRedirects.Add(1); return nil }
	result, err := New(client).StageKnownImage(context.Background(), &images.Image{ID: "id", Status: images.ImageStatus("queued")}, strings.NewReader("data"))
	if result != nil || !gophercloud.ResponseCodeIs(err, 303) || calls.Load() != 1 || sourceRedirects.Load() != 0 {
		t.Fatalf("redirect followed: %#v %v calls%d policy%d", result, err, calls.Load(), sourceRedirects.Load())
	}
}

type stageCoreReadFailure struct{ cause error }

func (reader stageCoreReadFailure) Read(data []byte) (int, error) {
	return copy(data, "part"), reader.cause
}
func (reader stageCoreReadFailure) Close() error { return nil }

func TestStageCoreAcknowledgementAndFollowupFailureEvidence(t *testing.T) {
	readCause := errors.New("accepted read failed")
	for _, mode := range []string{"stage-read", "get-read", "get-decode", "get-utf8", "get-native"} {
		t.Run(mode, func(t *testing.T) {
			var calls int
			client := stageCoreClient(stageCoreTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					response := stageCoreResponse(204, "accepted-stage")
					if mode == "stage-read" {
						response.Body = stageCoreReadFailure{readCause}
					}
					return response, nil
				}
				switch mode {
				case "get-read":
					response := stageCoreResponse(200, "")
					response.Body = stageCoreReadFailure{readCause}
					return response, nil
				case "get-decode":
					return stageCoreResponse(200, `{"min_ram":false}`), nil
				case "get-utf8":
					return stageCoreResponse(200, "{\"vendor\":\""+string([]byte{255})+"\"}"), nil
				default:
					return stageCoreResponse(404, "actual missing"), nil
				}
			}))
			result, err := New(client).StageKnownImage(context.Background(), &images.Image{ID: "id", Status: images.ImageStatus("queued")}, strings.NewReader("data"))
			if result == nil || result.Image != nil || err == nil || result.Acknowledgement == nil || result.Acknowledgement.StatusCode != 204 || result.Acknowledgement.Header.Get("X-Evidence") != "204" {
				t.Fatalf("missing ack: %#v %v", result, err)
			}
			if mode == "stage-read" {
				var evidence *resource.ResponseError
				if calls != 1 || result.StatusCode != 0 || result.Header != nil || result.Body != nil || string(result.Acknowledgement.Body) != "part" || !errors.Is(err, readCause) || !errors.As(err, &evidence) || evidence.StatusCode != 204 || string(evidence.Body) != "part" {
					t.Fatalf("stage read: %#v %v proof%#v calls%d", result, err, evidence, calls)
				}
				result.Acknowledgement.Header.Set("X-Evidence", "changed")
				result.Acknowledgement.Body[0] = '!'
				if evidence.Header.Get("X-Evidence") != "204" || string(evidence.Body) != "part" {
					t.Fatal("ack and error share evidence")
				}
				return
			}
			if calls != 2 || string(result.Acknowledgement.Body) != "accepted-stage" {
				t.Fatalf("unexpected retry: %#v %v calls%d", result, err, calls)
			}
			if mode == "get-native" {
				if result.StatusCode != 0 || result.Body != nil || result.Header != nil || !gophercloud.ResponseCodeIs(err, 404) {
					t.Fatalf("invented GET success: %#v %v", result, err)
				}
				return
			}
			var evidence *resource.ResponseError
			if result.StatusCode != 200 || result.Header.Get("X-Evidence") != "200" || !errors.As(err, &evidence) || evidence.StatusCode != 200 || string(evidence.Body) != string(result.Body) {
				t.Fatalf("GET proof: %#v %v proof%#v", result, err, evidence)
			}
			if mode == "get-read" && !errors.Is(err, readCause) {
				t.Fatal(err)
			}
			result.Body[0] = '!'
			result.Header.Set("X-Evidence", "changed")
			if evidence.Header.Get("X-Evidence") != "200" || string(evidence.Body) == string(result.Body) || string(result.Acknowledgement.Body) != "accepted-stage" {
				t.Fatal("result/error/ack share evidence")
			}
		})
	}
}

func TestStageCorePreflightSourceAndCustomCancellationCauses(t *testing.T) {
	var calls, optionCalls int
	client := stageCoreClient(stageCoreTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return stageCoreResponse(200, `{"status":"queued"}`), nil
	}))
	api := New(client)
	var typedNil *stageCoreReader
	for _, run := range []func() error{
		func() error { _, err := api.StageImage(nil, resource.ID("id"), strings.NewReader("")); return err },
		func() error {
			_, err := (*API)(nil).StageImage(context.Background(), resource.ID("id"), strings.NewReader(""))
			return err
		},
		func() error {
			_, err := api.StageImage(context.Background(), resource.ID("id"), nil, func(*StageOpts) error { optionCalls++; return nil })
			return err
		},
		func() error { _, err := api.StageImage(context.Background(), resource.ID("id"), typedNil); return err },
		func() error {
			_, err := api.StageImage(context.Background(), resource.ID("../bad"), strings.NewReader(""))
			return err
		},
		func() error {
			_, err := api.StageKnownImage(context.Background(), nil, strings.NewReader(""))
			return err
		},
		func() error {
			_, err := api.StageKnownImage(context.Background(), &images.Image{ID: "id", Status: images.ImageStatus("QUEUED")}, strings.NewReader(""))
			return err
		},
		func() error {
			_, err := api.StageImage(context.Background(), resource.ID("id"), strings.NewReader(""), WithStageSize(-1))
			return err
		},
		func() error {
			_, err := api.StageImage(context.Background(), resource.ID("id"), strings.NewReader(""), WithStageHeader("Accept", "bad"))
			return err
		},
	} {
		if err := run(); err == nil {
			t.Error("preflight accepted invalid input")
		}
	}
	client.MoreHeaders = map[string]string{"X-OpenStack-Image-Size": "1"}
	if _, err := api.StageImage(context.Background(), resource.ID("id"), strings.NewReader("")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls != 0 || optionCalls != 0 {
		t.Fatalf("preflight side effects HTTP%d options%d", calls, optionCalls)
	}
	client.MoreHeaders = nil
	client.ProviderClient.HTTPClient.Transport = stageCoreTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		client.Type = "other"
		return stageCoreResponse(200, `{"status":"queued"}`), nil
	})
	result, err := api.StageImage(context.Background(), resource.ID("id"), strings.NewReader(""))
	if result != nil || calls != 1 || !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("changed source continued: %#v %v calls%d", result, err, calls)
	}
	for _, phase := range []string{"initial", "binary", "followup"} {
		t.Run(phase, func(t *testing.T) {
			customCause := errors.New("caller supplied cancellation cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var requests int
			native := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Body: []byte("actual native"), ResponseHeader: http.Header{"X-Native": {"actual"}}}
			client := stageCoreClient(stageCoreTransport(func(req *http.Request) (*http.Response, error) {
				requests++
				if phase == "followup" && requests == 1 {
					return stageCoreResponse(204, "actual-stage"), nil
				}
				cancel(customCause)
				return nil, native
			}))
			var result *StageImageResult
			var err error
			if phase == "initial" {
				result, err = New(client).StageImage(ctx, resource.ID("id"), strings.NewReader(""))
			} else {
				result, err = New(client).StageKnownImage(ctx, &images.Image{ID: "id", Status: images.ImageStatus("queued")}, strings.NewReader(""))
			}
			var original gophercloud.ErrUnexpectedResponseCode
			if !errors.Is(err, context.Canceled) || !errors.Is(err, customCause) || !errors.As(err, &original) || string(original.Body) != "actual native" || original.ResponseHeader.Get("X-Native") != "actual" {
				t.Fatalf("lost causes: %v native%#v", err, original)
			}
			if phase == "followup" {
				if result == nil || result.Acknowledgement.StatusCode != 204 || requests != 2 {
					t.Fatalf("lost ack %#v requests%d", result, requests)
				}
			} else if result != nil || requests != 1 {
				t.Fatalf("invented acknowledgement %#v requests%d", result, requests)
			}
		})
	}
}

func TestStageCoreGETHeaderProjectionMatchesNativeAndPreservesRawEvidence(t *testing.T) {
	for _, test := range []struct {
		name    string
		body    string
		headers http.Header
	}{
		{"header-override-csv-last", `{"id":"response-decoy","status":"uploading","openstack-image-import-methods":"body-method","openstack-image-store-ids":"body-store","vendor":9007199254740993}`, http.Header{
			"Openstack-Image-Import-Methods": {"ignored-first", "glance-direct, web-download,glance-download"}, "Openstack-Image-Store-Ids": {"ignored-first", "fast,reliable"},
		}},
		{"body-capabilities-no-header", `{"status":"uploading","openstack-image-import-methods":"body-one, body-two","openstack-image-store-ids":"body-store"}`, make(http.Header)},
		{"header-overrides-incompatible-body-type", `{"status":"uploading","openstack-image-import-methods":false,"openstack-image-store-ids":123}`, http.Header{
			"openstack-image-import-methods": {"glance-direct"}, "OPENSTACK-IMAGE-STORE-IDS": {"fast"},
		}},
		{"last-empty-replaces-body", `{"status":"uploading","openstack-image-import-methods":"body-method","openstack-image-store-ids":"body-store"}`, http.Header{
			"Openstack-Image-Import-Methods": {"glance-direct", ""}, "Openstack-Image-Store-Ids": {"fast", ""},
		}},
		{"invalid-body-without-header", `{"status":"uploading","openstack-image-import-methods":false}`, make(http.Header)},
	} {
		t.Run(test.name, func(t *testing.T) {
			var native nativeimages.GetResult
			decoder := json.NewDecoder(bytes.NewBufferString(test.body))
			decoder.UseNumber()
			if err := decoder.Decode(&native.Body); err != nil {
				t.Fatal(err)
			}
			native.Header = test.headers.Clone()
			want, nativeErr := native.Extract()
			var calls int
			client := stageCoreClient(stageCoreTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					// Initial native extraction must also apply headers before its
					// full model validation, while queued remains a raw body check.
					response := stageCoreResponse(200, `{"status":"queued","openstack-image-import-methods":false}`)
					response.Header.Set("OpenStack-Image-Import-Methods", "glance-direct")
					return response, nil
				}
				if calls == 2 {
					return stageCoreResponse(204, "actual stage proof"), nil
				}
				response := stageCoreResponse(200, test.body)
				response.Header = test.headers.Clone()
				return response, nil
			}))
			result, err := New(client).StageImage(context.Background(), resource.ID("fixed"), strings.NewReader("data"))
			if result == nil || calls != 3 || result.ImageID != "fixed" || result.StatusCode != 200 || string(result.Body) != test.body || !reflect.DeepEqual(result.Header, test.headers) || result.Acknowledgement == nil || string(result.Acknowledgement.Body) != "actual stage proof" {
				t.Fatalf("lost actual evidence: %#v %v calls%d", result, err, calls)
			}
			if nativeErr == nil {
				if err != nil || !reflect.DeepEqual(result.Image, want) {
					t.Fatalf("native model mismatch: got%#v want%#v error%v", result.Image, want, err)
				}
			} else {
				var evidence *resource.ResponseError
				var decodeCause *json.UnmarshalTypeError
				if result.Image != nil || !errors.As(err, &evidence) || evidence.StatusCode != 200 || string(evidence.Body) != test.body || !errors.As(err, &decodeCause) {
					t.Fatalf("native failure lost evidence: %#v %v proof%#v", result, err, evidence)
				}
			}
			if !reflect.DeepEqual(test.headers, result.Header) || string(result.Body) != test.body {
				t.Fatal("native header projection mutated original response evidence")
			}
		})
	}
}
