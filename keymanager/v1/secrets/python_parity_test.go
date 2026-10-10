package secrets_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonSecretTransport func(*http.Request) (*http.Response, error)

func (transport pythonSecretTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonSecretCall struct{ method, path, contentType, body string }

// Python update_secret commits the Secret Resource with PUT and a JSON object
// of dirty attributes, e.g. {"id": "s1", "payload": "new", ...}, and accepts any
// status below 400. The native Update is Barbican's payload upload: the raw
// payload is the body, Content-Type is a header and only 204 succeeds.
func TestPythonSecretUpdateSendsRawPayload(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		status int
		ok     bool
	}{
		{"204", 204, true},
		{"200 with body", 200, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls []pythonSecretCall
			cloud.Provider.HTTPClient.Transport = pythonSecretTransport(func(req *http.Request) (*http.Response, error) {
				data, _ := io.ReadAll(req.Body)
				calls = append(calls, pythonSecretCall{req.Method, req.URL.Path, req.Header.Get("Content-Type"), string(data)})
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(`{"secret_ref":"https://kms/v1/secrets/s1"}`)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
			})
			client := cloud.Client("key-manager", "/catalog/unused/")
			client.ResourceBase = cloud.Server.URL + "/barbican/v1/"
			err := secrets.New(client).Update(ctx, "s1", secrets.UpdateOpts{ContentType: "text/plain", Payload: "new"})
			if tc.ok != (err == nil) || !tc.ok && !gophercloud.ResponseCodeIs(err, tc.status) {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, []pythonSecretCall{{http.MethodPut, "/barbican/v1/secrets/s1", "text/plain", "new"}}) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}
