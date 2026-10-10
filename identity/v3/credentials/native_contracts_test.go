package credentials_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeCredTransport func(*http.Request) (*http.Response, error)

func (transport nativeCredTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeCredCall struct{ method, path, query, body string }

func nativeCredAPI(t *testing.T, calls *[]nativeCredCall, reply func(*http.Request) (int, string)) (*credentials.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeCredTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeCredCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return credentials.New(client), cloud
}

func nativeCredOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "credentials" {
		t.Fatal("generated credentials context", err, wrapped)
	}
}

const nativeCredRow = `{"id":"c-1","blob":"{\"access\":\"a\"}","user_id":"u1","type":"ec2","project_id":"p","links":{"self":"x"}}`

func TestNativeCredentialRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeCredCall
	var cloud *testcloud.Cloud
	api, cloud := nativeCredAPI(t, &calls, func(req *http.Request) (int, string) {
		switch {
		case req.Method == http.MethodDelete:
			return 204, ""
		case req.Method == http.MethodPost:
			return 201, `{"credential":` + nativeCredRow + `}`
		case req.URL.Path == "/keystone/v3/credentials":
			return 200, `{"credentials":[` + nativeCredRow + `],"links":{"next":"` + cloud.Server.URL + `/other/credentials?page=2"}}`
		case req.URL.Path == "/other/credentials":
			return 200, `{"credentials":[{"id":"c-2","project_id":null}],"links":{"next":null}}`
		}
		return 200, `{"credential":` + nativeCredRow + `}`
	})
	created, err := api.Create(ctx, credentials.CreateOpts{Blob: `{"access":"a"}`, Type: "ec2", UserID: "u1", ProjectID: "p"}, credentials.WithCreateField("x_extension", 1))
	if err != nil || created.ID != "c-1" || created.Blob != `{"access":"a"}` {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "c-1")
	if err != nil || got.Type != "ec2" {
		t.Fatal(got, err)
	}
	if _, err := api.Update(ctx, "c-1", credentials.UpdateOpts{Blob: "new"}, credentials.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for value, err := range api.List(ctx, credentials.WithListOptions(credentials.ListOpts{UserID: "u1", Type: "ec2"}), credentials.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if err := api.Delete(ctx, "c-1"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, []string{"c-1", "c-2"}) {
		t.Fatal(ids)
	}
	want := []nativeCredCall{
		{http.MethodPost, "/keystone/v3/credentials", "", `{"credential":{"blob":"{\"access\":\"a\"}","project_id":"p","type":"ec2","user_id":"u1","x_extension":1}}`},
		{http.MethodGet, "/keystone/v3/credentials/c-1", "", ""},
		// Update uses PATCH and omits empty fields.
		{http.MethodPatch, "/keystone/v3/credentials/c-1", "", `{"credential":{"blob":"new","x_extension":1}}`},
		{http.MethodGet, "/keystone/v3/credentials", "extra=1&type=ec2&user_id=u1", ""},
		{http.MethodGet, "/other/credentials", "page=2", ""},
		{http.MethodDelete, "/keystone/v3/credentials/c-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeCredentialStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	valid := credentials.CreateOpts{Blob: "b", Type: "ec2", UserID: "u1"}
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*credentials.API) error
	}{
		{"Create", []int{201}, func(api *credentials.API) error { _, err := api.Create(ctx, valid); return err }},
		{"Get", []int{200}, func(api *credentials.API) error { _, err := api.Get(ctx, "c-1"); return err }},
		{"Update", []int{200}, func(api *credentials.API) error {
			_, err := api.Update(ctx, "c-1", credentials.UpdateOpts{})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *credentials.API) error { return api.Delete(ctx, "c-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeCredCall
				api, _ := nativeCredAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
				err := call.call(api)
				nativeCredOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("missing envelope yields nil", func(t *testing.T) {
		var calls []nativeCredCall
		api, _ := nativeCredAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{}` })
		if got, err := api.Get(ctx, "c-1"); err != nil || got != nil {
			t.Fatal(got, err)
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeCredCall
		api, _ := nativeCredAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{}` })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"blob":           {"Create", func() error { _, err := api.Create(ctx, credentials.CreateOpts{Type: "ec2", UserID: "u1"}); return err }()},
			"type":           {"Create", func() error { _, err := api.Create(ctx, credentials.CreateOpts{Blob: "b", UserID: "u1"}); return err }()},
			"user":           {"Create", func() error { _, err := api.Create(ctx, credentials.CreateOpts{Blob: "b", Type: "ec2"}); return err }()},
			"core extension": {"Create", func() error { _, err := api.Create(ctx, valid, credentials.WithCreateField("blob", "x")); return err }()},
			"update nil":     {"Update", func() error { _, err := api.Update(ctx, "c-1", credentials.UpdateOpts{}, nil); return err }()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeCredOperation(t, check.err, check.operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeCredOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
