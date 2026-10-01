package senlin_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/internal/senlin"
	"gophercloudsdk/resource"
)

func TestAsyncLocationUsesSelectedActionCollectionWithoutHTTP(t *testing.T) {
	client := &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{}, Endpoint: "https://example.invalid/proxy/v1/tenant/"}
	for _, location := range []string{
		"https://example.invalid/proxy/v1/tenant/actions/action-1",
		"/proxy/v1/tenant/actions/action-1",
		"actions/action-1",
		"actions/action%2D1",
	} {
		t.Run(location, func(t *testing.T) {
			response := &rest.Response{Body: json.RawMessage(`{"vendor":9007199254740993}`), Header: http.Header{"Location": {location}, "X-Request-Id": {"accepted"}}, StatusCode: http.StatusAccepted}
			originalBody := append([]byte(nil), response.Body...)
			originalHeader := response.Header.Clone()
			id, err := senlin.ActionID(client, response)
			if err != nil || id != "action-1" {
				t.Fatal(id, err)
			}
			if string(response.Body) != string(originalBody) || !reflect.DeepEqual(response.Header, originalHeader) {
				t.Fatal("decoding mutated accepted response evidence")
			}
		})
	}
	// ResourceBase can contain an escaped project segment distinct from Endpoint.
	client.ResourceBase = "https://example.invalid/proxy/v1/tenant%2Fpart/"
	response := &rest.Response{Header: http.Header{"location": {"actions/same-action"}}, StatusCode: 202}
	if id, err := senlin.ActionID(client, response); err != nil || id != "same-action" {
		t.Fatal(id, err)
	}
}

func TestAsyncMalformedLocationRetainsAcceptedResponseEvidence(t *testing.T) {
	client := &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{}, Endpoint: "https://example.invalid/proxy/v1/tenant/"}
	for _, location := range []string{
		"", " ", "actions/", "actions/action-1/", "actions/action-1/extra", "clusters/action-1",
		"https://other.invalid/proxy/v1/tenant/actions/action-1", "http://example.invalid/proxy/v1/tenant/actions/action-1",
		"https://user@example.invalid/proxy/v1/tenant/actions/action-1", "actions/action-1?token=anything", "actions/action-1?", "actions/action-1#fragment", "actions/action-1#",
		"../actions/action-1", "actions/../actions/action-1", "actions/%2e%2e", "actions/action%2F1", "actions/action%5C1", "actions/action%251", "actions/action%201", "actions/action%0A1", "actions/%zz",
		"/v1/tenant/actions/action-1", "https:actions/action-1",
	} {
		t.Run(location, func(t *testing.T) {
			response := &rest.Response{Body: json.RawMessage(`not necessarily JSON`), Header: http.Header{"Location": {location}, "X-Request-Id": {"accepted"}}, StatusCode: http.StatusAccepted}
			id, err := senlin.ActionID(client, response)
			var evidence *resource.ResponseError
			if id != "" || !errors.As(err, &evidence) || !errors.Is(err, resource.ErrInvalidOption) || evidence.StatusCode != 202 || string(evidence.Body) != `not necessarily JSON` || evidence.Header.Get("X-Request-Id") != "accepted" {
				t.Fatalf("lost accepted failure evidence: id=%q err=%v evidence=%+v", id, err, evidence)
			}
			response.Body[0] = 'X'
			response.Header.Set("X-Request-Id", "changed")
			if string(evidence.Body) != `not necessarily JSON` || evidence.Header.Get("X-Request-Id") != "accepted" {
				t.Fatal("accepted error evidence aliases response")
			}
		})
	}
}

func TestAsyncLocationRequiresOneHeaderAndSource(t *testing.T) {
	client := &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{}, Endpoint: "https://example.invalid/v1/"}
	for _, headers := range []http.Header{
		nil, {"Location": {}}, {"Location": {"actions/a", "actions/a"}}, {"Location": {"actions/a"}, "location": {"actions/b"}},
	} {
		if _, err := senlin.ActionID(client, &rest.Response{Header: headers, StatusCode: 202}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(headers, err)
		}
	}
	if _, err := senlin.ActionID(client, nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, source := range []*gophercloud.ServiceClient{nil, {}, {ProviderClient: &gophercloud.ProviderClient{}, Endpoint: "invalid"}} {
		var evidence *resource.ResponseError
		_, err := senlin.ActionID(source, &rest.Response{Header: http.Header{"Location": {"actions/a"}}, StatusCode: 202})
		if !errors.As(err, &evidence) || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(source, err)
		}
	}
}

func TestSenlinHeadersRejectOwnedAndMalformedValues(t *testing.T) {
	for _, headers := range []map[string]string{
		{"x-AUTH-token": "override"}, {"oPENsTACK-API-vERSION": "clustering 1.6"}, {"content-length": "1"},
		{"Host": "other"}, {"Authorization": "override"}, {"Cookie": "override"}, {"X-Service-Token": "override"},
		{"Content-Type": "application/json"}, {"X Vendor": "bad"}, {"X-Vendor": "bad\r\nheader"}, {"": "empty"},
	} {
		if err := senlin.Headers(headers); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(headers, err)
		}
	}
	headers := map[string]string{"X-Vendor": "extension"}
	if err := senlin.Headers(headers); err != nil || headers["X-Vendor"] != "extension" {
		t.Fatal(headers, err)
	}
}
