package containers_test

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
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeContainerTransport func(*http.Request) (*http.Response, error)

func (transport nativeContainerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeContainerWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}, "X-Container-Proof": {"actual"}}}
}

func nativeContainerClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("key-manager", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/barbican/v1/"
	client.MoreHeaders = map[string]string{"X-Source": "direct"}
	return client
}

func nativeContainerOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "containers" {
		t.Fatal("generated container context", err, wrapped)
	}
	var receipt *resource.ResponseError
	if errors.As(err, &receipt) {
		t.Fatal("native container call fabricated an owned receipt", receipt)
	}
}

const nativeContainerRow = `{"consumers":[{"name":"c","url":"http://consumer"}],"container_ref":"https://kms/v1/containers/c1","created":"2026-10-10T01:02:03","creator_id":"user","name":"n","secret_refs":[{"name":"s","secret_ref":"https://kms/v1/secrets/s1"}],"status":"ACTIVE","type":"generic","updated":"2026-10-10T01:02:04"}`

func TestNativeContainerRoutesBodiesAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	client := nativeContainerClient(cloud)
	type exchange struct{ method, path, body string }
	var seen []exchange
	cloud.Provider.HTTPClient.Transport = nativeContainerTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		if req.Header.Get("X-Source") != "direct" || req.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(req.Header)
		}
		seen = append(seen, exchange{req.Method, req.URL.Path, raw})
		switch {
		case req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/consumers"):
			return nativeContainerWire(200, nativeContainerRow), nil
		case req.Method == http.MethodPost:
			return nativeContainerWire(201, `{"container_ref":"https://kms/v1/containers/c1"}`), nil
		case req.Method == http.MethodDelete && strings.HasSuffix(req.URL.Path, "/consumers"):
			return nativeContainerWire(200, nativeContainerRow), nil
		case req.Method == http.MethodDelete:
			return nativeContainerWire(204, ""), nil
		}
		return nativeContainerWire(200, nativeContainerRow), nil
	})
	api := containers.New(client)
	if api.RawClient() != client {
		t.Fatal("native client identity changed")
	}
	ctx := context.Background()
	got, err := api.Get(ctx, "c1")
	want := &containers.Container{Consumers: []containers.ConsumerRef{{Name: "c", URL: "http://consumer"}}, ContainerRef: "https://kms/v1/containers/c1", Created: time.Date(2026, 10, 10, 1, 2, 3, 0, time.UTC), CreatorID: "user", Name: "n",
		SecretRefs: []containers.SecretRef{{Name: "s", SecretRef: "https://kms/v1/secrets/s1"}}, Status: "ACTIVE", Type: "generic", Updated: time.Date(2026, 10, 10, 1, 2, 4, 0, time.UTC)}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal(got, err)
	}
	created, err := api.Create(ctx, containers.CreateOpts{Type: containers.GenericContainer, SecretRefs: []containers.SecretRef{{Name: "s", SecretRef: "ref"}}}, containers.WithCreateField("x_extension", 1))
	if err != nil || created.ContainerRef != "https://kms/v1/containers/c1" {
		t.Fatal(created, err)
	}
	if _, err := api.CreateConsumer(ctx, "c1", containers.CreateConsumerOpts{Name: "c", URL: "http://consumer"}); err != nil {
		t.Fatal(err)
	}
	if _, err := api.DeleteConsumer(ctx, "c1", containers.WithDeleteConsumerOptions(containers.DeleteConsumerOpts{Name: "c", URL: "http://consumer"})); err != nil {
		t.Fatal(err)
	}
	if _, err := api.CreateSecretRef(ctx, "c1", containers.SecretRef{Name: "s", SecretRef: "ref"}); err != nil {
		t.Fatal(err)
	}
	if err := api.DeleteSecretRef(ctx, "c1", containers.WithDeleteSecretRefOptions(containers.SecretRef{Name: "s", SecretRef: "ref"})); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "a/b"); err != nil {
		t.Fatal(err)
	}
	wantSeen := []exchange{
		{http.MethodGet, "/barbican/v1/containers/c1", ""},
		// Name has no omitempty and is sent as "".
		{http.MethodPost, "/barbican/v1/containers", `{"name":"","secret_refs":[{"name":"s","secret_ref":"ref"}],"type":"generic","x_extension":1}`},
		{http.MethodPost, "/barbican/v1/containers/c1/consumers", `{"URL":"http://consumer","name":"c"}`},
		{http.MethodDelete, "/barbican/v1/containers/c1/consumers", `{"URL":"http://consumer","name":"c"}`},
		{http.MethodPost, "/barbican/v1/containers/c1/secrets", `{"name":"s","secret_ref":"ref"}`},
		{http.MethodDelete, "/barbican/v1/containers/c1/secrets", `{"name":"s","secret_ref":"ref"}`},
		{http.MethodDelete, "/barbican/v1/containers/a/b", ""},
	}
	if !reflect.DeepEqual(seen, wantSeen) {
		t.Fatalf("%+v", seen)
	}
}

func TestNativeContainerStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*containers.API) error
	}{
		{"Get", []int{200}, func(api *containers.API) error { _, err := api.Get(ctx, "c1"); return err }},
		{"Create", []int{201}, func(api *containers.API) error {
			_, err := api.Create(ctx, containers.CreateOpts{Type: containers.RSAContainer})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *containers.API) error { return api.Delete(ctx, "c1") }},
		{"CreateConsumer", []int{200}, func(api *containers.API) error {
			_, err := api.CreateConsumer(ctx, "c1", containers.CreateConsumerOpts{})
			return err
		}},
		{"DeleteConsumer", []int{200}, func(api *containers.API) error { _, err := api.DeleteConsumer(ctx, "c1"); return err }},
		{"CreateSecretRef", []int{201}, func(api *containers.API) error {
			_, err := api.CreateSecretRef(ctx, "c1", containers.SecretRef{})
			return err
		}},
		{"DeleteSecretRef", []int{204}, func(api *containers.API) error { return api.DeleteSecretRef(ctx, "c1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404, 409} {
			if contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Provider.HTTPClient.Transport = nativeContainerTransport(func(req *http.Request) (*http.Response, error) {
					requests.Add(1)
					return nativeContainerWire(code, nativeContainerRow), nil
				})
				err := call.call(containers.New(nativeContainerClient(cloud)))
				nativeContainerOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || native.ResponseHeader.Get("X-Container-Proof") != "actual" || requests.Load() != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("required type, collisions and nil options", func(t *testing.T) {
		cloud := testcloud.New(t)
		var requests atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeContainerTransport(func(req *http.Request) (*http.Response, error) {
			requests.Add(1)
			return nativeContainerWire(201, `{}`), nil
		})
		api := containers.New(nativeContainerClient(cloud))
		checks := map[string]func() error{
			"missing type": func() error { _, err := api.Create(ctx, containers.CreateOpts{Name: "n"}); return err },
			"create collision": func() error {
				_, err := api.Create(ctx, containers.CreateOpts{Type: containers.GenericContainer}, containers.WithCreateField("type", "x"))
				return err
			},
			"consumer collision": func() error {
				_, err := api.CreateConsumer(ctx, "c1", containers.CreateConsumerOpts{}, containers.WithCreateConsumerField("URL", "x"))
				return err
			},
			"nil option": func() error { return api.DeleteSecretRef(ctx, "c1", nil) },
		}
		for name, check := range checks {
			if err := check(); err == nil {
				t.Fatal(name, "accepted")
			}
		}
		if requests.Load() != 0 {
			t.Fatal(requests.Load())
		}
	})
}

// List and ListConsumers follow body next links verbatim; ListConsumers takes
// the container ListOpts, so its Name field is serialized too.
func TestNativeContainerListsQueryPagingAndStop(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	var paths []string
	cloud.Provider.HTTPClient.Transport = nativeContainerTransport(func(req *http.Request) (*http.Response, error) {
		queries = append(queries, req.URL.Query())
		paths = append(paths, req.URL.Path)
		consumers := strings.HasSuffix(req.URL.Path, "/consumers")
		switch {
		case !consumers && req.URL.Query().Get("offset") == "":
			return nativeContainerWire(200, `{"containers":[`+nativeContainerRow+`],"next":"`+cloud.Server.URL+`/other/containers?offset=1"}`), nil
		case !consumers:
			return nativeContainerWire(200, `{"containers":[{"name":"second"}]}`), nil
		case req.URL.Query().Get("offset") == "":
			return nativeContainerWire(200, `{"consumers":[{"name":"c1","url":"u","created":"2026-10-10T01:02:03"}],"next":"`+cloud.Server.URL+`/barbican/v1/containers/c1/consumers?offset=1"}`), nil
		}
		return nativeContainerWire(200, `{"consumers":[]}`), nil
	})
	api := containers.New(nativeContainerClient(cloud))
	var names []string
	for value, err := range api.List(context.Background(), containers.WithListOptions(containers.ListOpts{Limit: 1, Name: "n"}), containers.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, value.Name)
	}
	if !reflect.DeepEqual(names, []string{"n", "second"}) || !reflect.DeepEqual(queries[0], url.Values{"limit": {"1"}, "name": {"n"}, "extra": {"1"}}) || paths[1] != "/other/containers" {
		t.Fatal(names, queries, paths)
	}
	var consumers []string
	for value, err := range api.ListConsumers(context.Background(), "c1", containers.WithListConsumersOptions(containers.ListOpts{Limit: 1, Name: "ignored-by-server"})) {
		if err != nil {
			t.Fatal(err)
		}
		if !value.Created.Equal(time.Date(2026, 10, 10, 1, 2, 3, 0, time.UTC)) {
			t.Fatal(value)
		}
		consumers = append(consumers, value.Name)
	}
	if !reflect.DeepEqual(consumers, []string{"c1"}) || !reflect.DeepEqual(queries[2], url.Values{"limit": {"1"}, "name": {"ignored-by-server"}}) || paths[2] != "/barbican/v1/containers/c1/consumers" || len(paths) != 4 {
		t.Fatal(consumers, queries, paths)
	}
	t.Run("early stop", func(t *testing.T) {
		var calls atomic.Int32
		cloud := testcloud.New(t)
		cloud.Provider.HTTPClient.Transport = nativeContainerTransport(func(req *http.Request) (*http.Response, error) {
			calls.Add(1)
			return nativeContainerWire(200, `{"containers":[{"name":"a"},{"name":"b"}],"next":"`+cloud.Server.URL+`/barbican/v1/containers?offset=2"}`), nil
		})
		for value, err := range containers.New(nativeContainerClient(cloud)).List(context.Background()) {
			if err != nil || value.Name != "a" {
				t.Fatal(value, err)
			}
			break
		}
		if calls.Load() != 1 {
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
