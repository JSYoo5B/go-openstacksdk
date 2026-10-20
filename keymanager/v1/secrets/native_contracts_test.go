package secrets_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeSecretTransport func(*http.Request) (*http.Response, error)

func (transport nativeSecretTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeSecretWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}, "X-Secret-Proof": {"actual"}}}
}

func nativeSecretClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("key-manager", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/barbican/v1/"
	client.MoreHeaders = map[string]string{"X-Source": "direct"}
	return client
}

func nativeSecretOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "secrets" {
		t.Fatal("generated secret context", err, wrapped)
	}
	var receipt *resource.ResponseError
	if errors.As(err, &receipt) {
		t.Fatal("native secret call fabricated an owned receipt", receipt)
	}
}

const nativeSecretRow = `{"algorithm":"aes","bit_length":256,"content_types":{"default":"text/plain"},"created":"2026-10-10T01:02:03","creator_id":"user","expiration":null,"mode":"cbc","name":"n","secret_ref":"https://kms/v1/secrets/s1","secret_type":"opaque","status":"ACTIVE","updated":"2026-10-10T01:02:04"}`

func TestNativeSecretGetCreateDeleteUpdateRoutesAndBodies(t *testing.T) {
	cloud := testcloud.New(t)
	client := nativeSecretClient(cloud)
	type exchange struct{ method, path, body, contentType, extra string }
	var seen []exchange
	cloud.Provider.HTTPClient.Transport = nativeSecretTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		if req.Header.Get("X-Source") != "direct" || req.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(req.Header)
		}
		seen = append(seen, exchange{req.Method, req.URL.Path, raw, req.Header.Get("Content-Type"), req.Header.Get("X-Extra")})
		switch req.Method {
		case http.MethodPost:
			return nativeSecretWire(201, `{"secret_ref":"https://kms/v1/secrets/s1"}`), nil
		case http.MethodPut, http.MethodDelete:
			return nativeSecretWire(204, ""), nil
		}
		return nativeSecretWire(200, nativeSecretRow), nil
	})
	api := secrets.New(client)
	if api.RawClient() != client {
		t.Fatal("native client identity changed")
	}
	ctx := context.Background()
	got, err := api.Get(ctx, "s1")
	if err != nil || got.Name != "n" || got.BitLength != 256 || got.SecretRef != "https://kms/v1/secrets/s1" || !got.Created.Equal(time.Date(2026, 10, 10, 1, 2, 3, 0, time.UTC)) || !got.Expiration.IsZero() || got.ContentTypes["default"] != "text/plain" {
		t.Fatal(got, err)
	}
	expiration := time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)
	created, err := api.Create(ctx, secrets.CreateOpts{Name: "n", Payload: "p", PayloadContentType: "text/plain", SecretType: secrets.OpaqueSecret, Expiration: &expiration}, secrets.WithCreateField("x_extension", true))
	if err != nil || created.SecretRef != "https://kms/v1/secrets/s1" {
		t.Fatal(created, err)
	}
	if _, err := api.Create(ctx, secrets.CreateOpts{}); err != nil {
		t.Fatal(err)
	}
	if err := api.Update(ctx, "s1", secrets.UpdateOpts{ContentType: "text/plain", Payload: "new"}, secrets.WithUpdateHeader("X-Extra", "1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "a/b"); err != nil {
		t.Fatal(err)
	}
	want := []exchange{
		{http.MethodGet, "/barbican/v1/secrets/s1", "", "", ""},
		{http.MethodPost, "/barbican/v1/secrets", `{"expiration":"2027-01-02T03:04:05","name":"n","payload":"p","payload_content_type":"text/plain","secret_type":"opaque","x_extension":true}`, "application/json", ""},
		{http.MethodPost, "/barbican/v1/secrets", `{}`, "application/json", ""},
		{http.MethodPut, "/barbican/v1/secrets/s1", "new", "text/plain", "1"},
		{http.MethodDelete, "/barbican/v1/secrets/a/b", "", "", ""},
	}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("%+v", seen)
	}
}

func TestNativeSecretStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*secrets.API) error
	}{
		{"Get", []int{200}, func(api *secrets.API) error { _, err := api.Get(ctx, "s1"); return err }},
		{"Create", []int{201}, func(api *secrets.API) error { _, err := api.Create(ctx, secrets.CreateOpts{Name: "n"}); return err }},
		{"Update", []int{204}, func(api *secrets.API) error { return api.Update(ctx, "s1", secrets.UpdateOpts{Payload: "p"}) }},
		{"Delete", []int{202, 204}, func(api *secrets.API) error { return api.Delete(ctx, "s1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404, 409} {
			if contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Provider.HTTPClient.Transport = nativeSecretTransport(func(req *http.Request) (*http.Response, error) {
					requests.Add(1)
					return nativeSecretWire(code, nativeSecretRow), nil
				})
				err := call.call(secrets.New(nativeSecretClient(cloud)))
				nativeSecretOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || native.ResponseHeader.Get("X-Secret-Proof") != "actual" || requests.Load() != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("extension collisions and nil options", func(t *testing.T) {
		cloud := testcloud.New(t)
		var requests atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeSecretTransport(func(req *http.Request) (*http.Response, error) {
			requests.Add(1)
			return nativeSecretWire(201, `{}`), nil
		})
		api := secrets.New(nativeSecretClient(cloud))
		for _, option := range []secrets.CreateOption{secrets.WithCreateField("name", "x"), nil} {
			_, err := api.Create(ctx, secrets.CreateOpts{Name: "n"}, option)
			nativeSecretOperation(t, err, "Create")
		}
		for _, option := range []secrets.UpdateOption{secrets.WithUpdateHeader("Content-Type", "x"), nil} {
			err := api.Update(ctx, "s1", secrets.UpdateOpts{ContentType: "text/plain"}, option)
			nativeSecretOperation(t, err, "Update")
		}
		if requests.Load() != 0 {
			t.Fatal(requests.Load())
		}
	})
}

// List follows the native body next link verbatim and serializes ListOpts.
func TestNativeSecretListQueryPagingAndStop(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	var queries []url.Values
	var paths []string
	cloud.Provider.HTTPClient.Transport = nativeSecretTransport(func(req *http.Request) (*http.Response, error) {
		queries = append(queries, req.URL.Query())
		paths = append(paths, req.URL.Path)
		if calls.Add(1) == 1 {
			return nativeSecretWire(200, `{"secrets":[`+nativeSecretRow+`],"next":"`+cloud.Server.URL+`/other/secrets?offset=1&limit=1","total":2}`), nil
		}
		return nativeSecretWire(200, `{"secrets":[{"name":"second"}],"total":2}`), nil
	})
	created := time.Date(2026, 10, 10, 1, 2, 3, 0, time.UTC)
	acl := true
	var names []string
	for secret, err := range secrets.New(nativeSecretClient(cloud)).List(context.Background(), secrets.WithListOptions(secrets.ListOpts{Limit: 1, Name: "n", Bits: 256, SecretType: secrets.OpaqueSecret, ACLOnly: &acl, CreatedQuery: &secrets.DateQuery{Date: created, Filter: secrets.DateFilterGTE}, Sort: "name:asc"}), secrets.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, secret.Name)
	}
	want := url.Values{"limit": {"1"}, "name": {"n"}, "bits": {"256"}, "secret_type": {"opaque"}, "acl_only": {"true"}, "created": {"gte:2026-10-10T01:02:03Z"}, "sort": {"name:asc"}, "extra": {"1"}}
	if !reflect.DeepEqual(names, []string{"n", "second"}) || !reflect.DeepEqual(queries[0], want) || paths[1] != "/other/secrets" || queries[1].Get("offset") != "1" {
		t.Fatal(names, queries, paths)
	}
	t.Run("empty page and early stop", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeSecretTransport(func(req *http.Request) (*http.Response, error) {
			if calls.Add(1) == 1 && req.URL.Query().Get("name") == "empty" {
				return nativeSecretWire(200, `{"secrets":[],"next":"`+cloud.Server.URL+`/barbican/v1/secrets?offset=9"}`), nil
			}
			return nativeSecretWire(200, `{"secrets":[{"name":"a"},{"name":"b"}],"next":"`+cloud.Server.URL+`/barbican/v1/secrets?offset=2"}`), nil
		})
		api := secrets.New(nativeSecretClient(cloud))
		for value, err := range api.List(context.Background(), secrets.WithListQuery("name", "empty")) {
			t.Fatal("empty page yielded", value, err)
		}
		for value, err := range api.List(context.Background()) {
			if err != nil || value.Name != "a" {
				t.Fatal(value, err)
			}
			break
		}
		if calls.Load() != 2 {
			t.Fatal(calls.Load())
		}
	})
}

func contains(values []int, value int) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
