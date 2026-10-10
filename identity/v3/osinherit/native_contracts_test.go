package osinherit_test

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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeInheritTransport func(*http.Request) (*http.Response, error)

func (transport nativeInheritTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeInheritCall struct{ method, path, query, body string }

func nativeInheritAPI(t *testing.T, calls *[]nativeInheritCall, reply func(*http.Request) (int, string)) *osinherit.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeInheritTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeInheritCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return osinherit.New(client)
}

func nativeInheritOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "osinherit" {
		t.Fatal("generated osinherit context", err, wrapped)
	}
}

func TestNativeInheritRoutes(t *testing.T) {
	ctx := context.Background()
	var calls []nativeInheritCall
	api := nativeInheritAPI(t, &calls, func(*http.Request) (int, string) { return 204, "" })
	if err := api.Assign(ctx, "r-1", osinherit.AssignOpts{UserID: "u-1", ProjectID: "p-1"}); err != nil {
		t.Fatal(err)
	}
	// WithAssignOptions replaces the positional options.
	if err := api.Assign(ctx, "r-1", osinherit.AssignOpts{}, osinherit.WithAssignOptions(osinherit.AssignOpts{GroupID: "g-1", DomainID: "d-1"})); err != nil {
		t.Fatal(err)
	}
	if err := api.Validate(ctx, "r-1", osinherit.ValidateOpts{GroupID: "g-1", ProjectID: "p-1"}); err != nil {
		t.Fatal(err)
	}
	if err := api.Unassign(ctx, "r-1", osinherit.UnassignOpts{UserID: "u-1", DomainID: "d-1"}); err != nil {
		t.Fatal(err)
	}
	want := []nativeInheritCall{
		{http.MethodPut, "/keystone/v3/OS-INHERIT/projects/p-1/users/u-1/roles/r-1/inherited_to_projects", "", ""},
		{http.MethodPut, "/keystone/v3/OS-INHERIT/domains/d-1/groups/g-1/roles/r-1/inherited_to_projects", "", ""},
		{http.MethodHead, "/keystone/v3/OS-INHERIT/projects/p-1/groups/g-1/roles/r-1/inherited_to_projects", "", ""},
		{http.MethodDelete, "/keystone/v3/OS-INHERIT/domains/d-1/users/u-1/roles/r-1/inherited_to_projects", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeInheritStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*osinherit.API) error
	}{
		{"Assign", []int{204}, func(api *osinherit.API) error {
			return api.Assign(ctx, "r-1", osinherit.AssignOpts{UserID: "u-1", ProjectID: "p-1"})
		}},
		{"Validate", []int{204}, func(api *osinherit.API) error {
			return api.Validate(ctx, "r-1", osinherit.ValidateOpts{UserID: "u-1", ProjectID: "p-1"})
		}},
		{"Unassign", []int{204}, func(api *osinherit.API) error {
			return api.Unassign(ctx, "r-1", osinherit.UnassignOpts{UserID: "u-1", ProjectID: "p-1"})
		}},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeInheritCall
				api := nativeInheritAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
				err := call.call(api)
				nativeInheritOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeInheritCall
		api := nativeInheritAPI(t, &calls, func(*http.Request) (int, string) { return 204, "" })
		for name, opts := range map[string]osinherit.AssignOpts{
			"no actor":    {ProjectID: "p"},
			"two actors":  {UserID: "u", GroupID: "g", ProjectID: "p"},
			"no target":   {UserID: "u"},
			"two targets": {UserID: "u", ProjectID: "p", DomainID: "d"},
		} {
			for operation, err := range map[string]error{
				"Assign":   api.Assign(ctx, "r-1", opts),
				"Validate": api.Validate(ctx, "r-1", osinherit.ValidateOpts(opts)),
				"Unassign": api.Unassign(ctx, "r-1", osinherit.UnassignOpts(opts)),
			} {
				var missing gophercloud.ErrMissingInput
				if !errors.As(err, &missing) {
					t.Fatal(name, operation, err)
				}
				nativeInheritOperation(t, err, operation)
			}
		}
		for operation, err := range map[string]error{
			"Assign":   api.Assign(ctx, "r-1", osinherit.AssignOpts{UserID: "u", ProjectID: "p"}, nil),
			"Validate": api.Validate(ctx, "r-1", osinherit.ValidateOpts{UserID: "u", ProjectID: "p"}, nil),
			"Unassign": api.Unassign(ctx, "r-1", osinherit.UnassignOpts{UserID: "u", ProjectID: "p"}, nil),
		} {
			nativeInheritOperation(t, err, operation)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
