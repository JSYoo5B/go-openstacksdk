package rest_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type responseModel struct {
	resource.Metadata
	ID string `json:"id"`
}

func (m *responseModel) UnmarshalJSON(data []byte) error {
	type plain responseModel
	return resource.DecodeObject(data, (*plain)(m), &m.Metadata)
}

type responseTransport func(*http.Request) (*http.Response, error)

func (f responseTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func responseClient(transport responseTransport) *gophercloud.ServiceClient {
	p := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: transport}}
	p.UseTokenLock()
	p.SetToken("old-token")
	return &gophercloud.ServiceClient{ProviderClient: p, Endpoint: "https://service.test/v1/project/", Type: "instance-ha"}
}

func wireResponse(code int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"X-Request-Id": {"req-42"}}, Body: body}
}

func TestRESTResponseSnapshotsMutationAcrossReauth(t *testing.T) {
	body := map[string]any{"segment": map[string]any{"name": "original", "enabled": false}}
	headers := map[string]string{"X-Extension": "initial"}
	codes := []int{202}
	var attempts, reauth int
	client := responseClient(func(r *http.Request) (*http.Response, error) {
		attempts++
		wire, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if r.Method != http.MethodPost || r.URL.String() != "https://service.test/v1/project/segments" || string(wire) != `{"segment":{"enabled":false,"name":"original"}}` || r.Header.Get("X-Extension") != "initial" {
			t.Fatalf("attempt %d changed request: %s %s %s %#v", attempts, r.Method, r.URL, wire, r.Header)
		}
		want := "old-token"
		if attempts == 2 {
			want = "fresh-token"
		}
		if r.Header.Get("X-Auth-Token") != want {
			t.Fatalf("token=%q want=%q", r.Header.Get("X-Auth-Token"), want)
		}
		if attempts == 1 {
			return wireResponse(401, io.NopCloser(strings.NewReader(`{"error":"expired"}`))), nil
		}
		return wireResponse(202, io.NopCloser(strings.NewReader(`{"segment":{"id":"uuid","extension":9007199254740993}}`))), nil
	})
	client.ReauthFunc = func(ctx context.Context) error {
		reauth++
		body["segment"].(map[string]any)["name"] = "mutated"
		headers["X-Extension"] = "mutated"
		codes[0] = 200
		client.SetToken("fresh-token")
		return nil
	}
	response, err := rest.DoJSON(context.Background(), client, http.MethodPost, client.ServiceURL("segments"), body, headers, codes...)
	if err != nil {
		t.Fatal(err)
	}
	model, err := rest.Decode[responseModel](response, "segment", func(m *responseModel) *resource.Metadata { return &m.Metadata })
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 || reauth != 1 || model.ID != "uuid" || model.StatusCode != 202 || model.Header.Get("X-Request-Id") != "req-42" || string(model.Body["extension"]) != "9007199254740993" || client.Token() != "fresh-token" {
		t.Fatalf("accepted evidence or reauth lost: %#v attempts=%d reauth=%d", model, attempts, reauth)
	}
	response.Header.Set("X-Request-Id", "mutated")
	if model.Header.Get("X-Request-Id") != "req-42" {
		t.Fatal("model aliases response headers")
	}
}

type failingBody struct{ sent, closed bool }

func (b *failingBody) Read(p []byte) (int, error) {
	if !b.sent {
		b.sent = true
		return copy(p, `{"segment":`), nil
	}
	return 0, io.ErrUnexpectedEOF
}
func (b *failingBody) Close() error { b.closed = true; return nil }

func TestRESTAcceptedReadFailureKeepsEvidenceWithoutResend(t *testing.T) {
	body := &failingBody{}
	requests := 0
	client := responseClient(func(r *http.Request) (*http.Response, error) { requests++; return wireResponse(202, body), nil })
	response, err := rest.DoJSON(context.Background(), client, http.MethodPost, client.ServiceURL("segments"), map[string]any{"segment": map[string]string{"name": "a"}}, nil, 202)
	var evidence *resource.ResponseError
	if !errors.Is(err, io.ErrUnexpectedEOF) || !errors.As(err, &evidence) || string(evidence.Body) != `{"segment":` || evidence.StatusCode != 202 || evidence.Header.Get("X-Request-Id") != "req-42" || !body.closed || requests != 1 || response == nil {
		t.Fatalf("read failure: response=%#v error=%v requests=%d closed=%v", response, err, requests, body.closed)
	}
	response.Body[0] = 'x'
	response.Header.Set("X-Request-Id", "mutated")
	if string(evidence.Body) != `{"segment":` || evidence.Header.Get("X-Request-Id") != "req-42" {
		t.Fatal("error evidence aliases response")
	}
}

func TestRESTDecodeRequiresChosenEnvelope(t *testing.T) {
	for _, raw := range []string{"", "null", "[]", `{"segments":[]}`, `{"segment":null}`, `{"segment":[]}`, `{"segment":{"id":123}}`} {
		t.Run(raw, func(t *testing.T) {
			response := &rest.Response{Body: json.RawMessage(raw), StatusCode: 202, Header: http.Header{"X-Request-Id": {"req-42"}}}
			_, err := rest.Decode[responseModel](response, "segment", func(m *responseModel) *resource.Metadata { return &m.Metadata })
			var evidence *resource.ResponseError
			if !errors.As(err, &evidence) || evidence.StatusCode != 202 || string(evidence.Body) != raw || evidence.Header.Get("X-Request-Id") != "req-42" {
				t.Fatalf("invalid envelope lost response: %v", err)
			}
		})
	}
	response := &rest.Response{Body: json.RawMessage(`{"id":"uuid"}`), StatusCode: 200}
	model, err := rest.Decode[responseModel](response, "", func(m *responseModel) *resource.Metadata { return &m.Metadata })
	if err != nil || model.ID != "uuid" {
		t.Fatalf("explicit flat envelope: %#v %v", model, err)
	}
	if _, err := rest.Decode[responseModel](response, "", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("missing model metadata: %v", err)
	}
}

func TestRESTRejectsOtherOriginBeforeAuthenticationOrHTTP(t *testing.T) {
	requests := 0
	client := responseClient(func(r *http.Request) (*http.Response, error) {
		requests++
		return wireResponse(200, io.NopCloser(strings.NewReader(`{}`))), nil
	})
	for _, endpoint := range []string{"https://foreign.test/v1/segments", "http://service.test/v1/segments", "https://user:secret@service.test/v1/segments", "https://service.test/v1/segments#fragment", "//service.test/v1/segments", "https:opaque"} {
		if _, err := rest.DoJSON(context.Background(), client, http.MethodGet, endpoint, nil, nil, 200); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("target %q: %v", endpoint, err)
		}
	}
	if requests != 0 {
		t.Fatalf("sent %d invalid requests", requests)
	}
	if _, err := rest.DoJSON(context.Background(), client, http.MethodGet, client.ServiceURL("segments"), nil, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("no success codes: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := rest.DoJSON(ctx, client, http.MethodGet, client.ServiceURL("segments"), nil, nil, 200); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled call: %v", err)
	}
}

func TestRESTNativeHTTPFailureAndEmptyDelete(t *testing.T) {
	requests := 0
	client := responseClient(func(r *http.Request) (*http.Response, error) {
		requests++
		if r.Method == http.MethodDelete {
			return wireResponse(204, http.NoBody), nil
		}
		return wireResponse(403, io.NopCloser(strings.NewReader(`{"error":"forbidden"}`))), nil
	})
	_, err := rest.DoJSON(context.Background(), client, http.MethodGet, client.ServiceURL("segments"), nil, nil, 200)
	var native gophercloud.ErrUnexpectedResponseCode
	var accepted *resource.ResponseError
	if !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != `{"error":"forbidden"}` || errors.As(err, &accepted) {
		t.Fatalf("native failure changed: %T %v", err, err)
	}
	response, err := rest.DoJSON(context.Background(), client, http.MethodDelete, client.ServiceURL("segments", "uuid"), nil, nil, 204)
	if err != nil || response.StatusCode != 204 || len(response.Body) != 0 || requests != 2 {
		t.Fatalf("empty deletion: %#v %v requests=%d", response, err, requests)
	}
}
