package ec2credentials_test

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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/ec2credentials"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeEC2Transport func(*http.Request) (*http.Response, error)

func (transport nativeEC2Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeEC2Call struct{ method, path, query, body string }

func nativeEC2API(t *testing.T, calls *[]nativeEC2Call, reply func(*http.Request) (int, string)) (*ec2credentials.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeEC2Transport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeEC2Call{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return ec2credentials.New(client), cloud
}

func nativeEC2Operation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "ec2credentials" {
		t.Fatal("generated ec2credentials context", err, wrapped)
	}
}

const nativeEC2Row = `{"user_id":"u1","tenant_id":"p","access":"ak","secret":"sk","trust_id":null,"links":{"self":"x"}}`

func TestNativeEC2CredentialRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeEC2Call
	var cloud *testcloud.Cloud
	api, cloud := nativeEC2API(t, &calls, func(req *http.Request) (int, string) {
		switch {
		case req.Method == http.MethodDelete:
			return 204, ""
		case req.Method == http.MethodPost:
			return 201, `{"credential":` + nativeEC2Row + `}`
		case req.URL.Path == "/keystone/v3/users/u1/credentials/OS-EC2":
			return 200, `{"credentials":[` + nativeEC2Row + `],"links":{"next":"` + cloud.Server.URL + `/other/ec2?page=2"}}`
		case req.URL.Path == "/other/ec2":
			return 200, `{"credentials":[{"access":"ak2"}],"links":{"next":null}}`
		}
		return 200, `{"credential":` + nativeEC2Row + `}`
	})
	created, err := api.Create(ctx, "u1", ec2credentials.CreateOpts{TenantID: "p"}, ec2credentials.WithCreateField("x_extension", 1))
	if err != nil || created.Access != "ak" || created.Secret != "sk" || created.TrustID != "" {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "u1", "ak")
	if err != nil || got.TenantID != "p" {
		t.Fatal(got, err)
	}
	var access []string
	for value, err := range api.List(ctx, "u1") {
		if err != nil {
			t.Fatal(err)
		}
		access = append(access, value.Access)
	}
	if err := api.Delete(ctx, "u1", "ak"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(access, []string{"ak", "ak2"}) {
		t.Fatal(access)
	}
	base := "/keystone/v3/users/u1/credentials/OS-EC2"
	want := []nativeEC2Call{
		// The create body has no envelope, so an extension sits beside tenant_id.
		{http.MethodPost, base, "", `{"tenant_id":"p","x_extension":1}`},
		{http.MethodGet, base + "/ak", "", ""},
		{http.MethodGet, base, "", ""},
		{http.MethodGet, "/other/ec2", "page=2", ""},
		{http.MethodDelete, base + "/ak", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeEC2CredentialStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*ec2credentials.API) error
	}{
		{"Create", []int{201}, func(api *ec2credentials.API) error {
			_, err := api.Create(ctx, "u1", ec2credentials.CreateOpts{TenantID: "p"})
			return err
		}},
		{"Get", []int{200}, func(api *ec2credentials.API) error { _, err := api.Get(ctx, "u1", "ak"); return err }},
		{"Delete", []int{202, 204}, func(api *ec2credentials.API) error { return api.Delete(ctx, "u1", "ak") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeEC2Call
				api, _ := nativeEC2API(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
				err := call.call(api)
				nativeEC2Operation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeEC2Call
		api, _ := nativeEC2API(t, &calls, func(*http.Request) (int, string) { return 201, `{}` })
		for name, err := range map[string]error{
			"tenant": func() error { _, err := api.Create(ctx, "u1", ec2credentials.CreateOpts{}); return err }(),
			"core extension": func() error {
				_, err := api.Create(ctx, "u1", ec2credentials.CreateOpts{TenantID: "p"}, ec2credentials.WithCreateField("tenant_id", "x"))
				return err
			}(),
		} {
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeEC2Operation(t, err, "Create")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
