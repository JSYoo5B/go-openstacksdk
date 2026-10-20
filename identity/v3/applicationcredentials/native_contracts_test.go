package applicationcredentials_test

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
	"time"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeAppCredTransport func(*http.Request) (*http.Response, error)

func (transport nativeAppCredTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeAppCredCall struct{ method, path, query, body string }

func nativeAppCredAPI(t *testing.T, calls *[]nativeAppCredCall, reply func(*http.Request) (int, string)) (*applicationcredentials.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeAppCredTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeAppCredCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return applicationcredentials.New(client), cloud
}

func nativeAppCredOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "applicationcredentials" {
		t.Fatal("generated applicationcredentials context", err, wrapped)
	}
}

const nativeAppCredRow = `{"id":"ac-1","name":"ci","description":"","unrestricted":false,"secret":"s3cr3t","project_id":"p","roles":[{"id":"r1","name":"member"}],"expires_at":"2026-10-11T01:02:03.000000","access_rules":[{"id":"ar-1","path":"/v2.1/servers","method":"GET","service":"compute"}],"links":{"self":"x"}}`

func TestNativeApplicationCredentialRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeAppCredCall
	var cloud *testcloud.Cloud
	api, cloud := nativeAppCredAPI(t, &calls, func(req *http.Request) (int, string) {
		switch {
		case req.Method == http.MethodDelete:
			return 204, ""
		case req.Method == http.MethodPost:
			return 201, `{"application_credential":` + nativeAppCredRow + `}`
		case req.URL.Path == "/keystone/v3/users/u1/application_credentials":
			return 200, `{"application_credentials":[` + nativeAppCredRow + `],"links":{"next":"` + cloud.Server.URL + `/other/appcreds?page=2","previous":null}}`
		case req.URL.Path == "/other/appcreds":
			return 200, `{"application_credentials":[{"id":"ac-2","expires_at":null}],"links":{"next":null}}`
		case req.URL.Path == "/keystone/v3/users/u1/access_rules":
			return 200, `{"access_rules":[{"id":"ar-1","path":"/v2.1/servers","method":"GET","service":"compute"}],"links":{"next":null}}`
		case strings.HasPrefix(req.URL.Path, "/keystone/v3/users/u1/access_rules/"):
			return 200, `{"access_rule":{"id":"ar-1","path":"/v2.1/servers","method":"GET","service":"compute"}}`
		}
		return 200, `{"application_credential":` + nativeAppCredRow + `}`
	})
	expires := time.Date(2026, 10, 11, 1, 2, 3, 500000000, time.FixedZone("KST", 9*3600))
	created, err := api.Create(ctx, "u1", applicationcredentials.CreateOpts{Name: "ci", Secret: "s3cr3t", Roles: []applicationcredentials.Role{{Name: "member"}}, AccessRules: []applicationcredentials.AccessRule{{Path: "/v2.1/servers", Method: "GET", Service: "compute"}}, ExpiresAt: &expires}, applicationcredentials.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "ac-1" && created.Secret == "s3cr3t" && created.Roles[0].ID == "r1" && created.AccessRules[0].ID == "ar-1" && created.ExpiresAt.Second() == 3) {
		t.Fatal(created, err)
	}
	// Unrestricted has no omitempty, so false is always sent.
	if _, err := api.Create(ctx, "u1", applicationcredentials.CreateOpts{Name: "ci"}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "u1", "ac-1")
	if err != nil || got.ProjectID != "p" {
		t.Fatal(got, err)
	}
	var ids []string
	for value, err := range api.List(ctx, "u1", applicationcredentials.WithListOptions(applicationcredentials.ListOpts{Name: "ci"}), applicationcredentials.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	var rules []string
	for value, err := range api.ListAccessRules(ctx, "u1") {
		if err != nil {
			t.Fatal(err)
		}
		rules = append(rules, value.ID+"="+value.Service)
	}
	rule, err := api.GetAccessRule(ctx, "u1", "ar-1")
	if err != nil || rule.Method != "GET" {
		t.Fatal(rule, err)
	}
	if err := api.Delete(ctx, "u1", "ac-1"); err != nil {
		t.Fatal(err)
	}
	if err := api.DeleteAccessRule(ctx, "u1", "ar-1"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, []string{"ac-1", "ac-2"}) || !reflect.DeepEqual(rules, []string{"ar-1=compute"}) {
		t.Fatal(ids, rules)
	}
	base := "/keystone/v3/users/u1"
	want := []nativeAppCredCall{
		// expires_at is formatted without a zone in the timestamp's own location.
		{http.MethodPost, base + "/application_credentials", "", `{"application_credential":{"access_rules":[{"method":"GET","path":"/v2.1/servers","service":"compute"}],"expires_at":"2026-10-11T01:02:03.5","name":"ci","roles":[{"name":"member"}],"secret":"s3cr3t","unrestricted":false,"x_extension":1}}`},
		{http.MethodPost, base + "/application_credentials", "", `{"application_credential":{"name":"ci","unrestricted":false}}`},
		{http.MethodGet, base + "/application_credentials/ac-1", "", ""},
		{http.MethodGet, base + "/application_credentials", "extra=1&name=ci", ""},
		// Keystone paging follows links.next as a string.
		{http.MethodGet, "/other/appcreds", "page=2", ""},
		{http.MethodGet, base + "/access_rules", "", ""},
		{http.MethodGet, base + "/access_rules/ar-1", "", ""},
		{http.MethodDelete, base + "/application_credentials/ac-1", "", ""},
		{http.MethodDelete, base + "/access_rules/ar-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeApplicationCredentialStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*applicationcredentials.API) error
	}{
		// Create accepts only 201.
		{"Create", []int{201}, func(api *applicationcredentials.API) error {
			_, err := api.Create(ctx, "u1", applicationcredentials.CreateOpts{Name: "ci"})
			return err
		}},
		{"Get", []int{200}, func(api *applicationcredentials.API) error { _, err := api.Get(ctx, "u1", "ac-1"); return err }},
		{"GetAccessRule", []int{200}, func(api *applicationcredentials.API) error {
			_, err := api.GetAccessRule(ctx, "u1", "ar-1")
			return err
		}},
		{"Delete", []int{202, 204}, func(api *applicationcredentials.API) error { return api.Delete(ctx, "u1", "ac-1") }},
		{"DeleteAccessRule", []int{202, 204}, func(api *applicationcredentials.API) error { return api.DeleteAccessRule(ctx, "u1", "ar-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeAppCredCall
				api, _ := nativeAppCredAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
				err := call.call(api)
				nativeAppCredOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("decode", func(t *testing.T) {
		var calls []nativeAppCredCall
		api, _ := nativeAppCredAPI(t, &calls, func(req *http.Request) (int, string) {
			if strings.HasSuffix(req.URL.Path, "/zoned") {
				return 200, `{"application_credential":{"expires_at":"2026-10-11T01:02:03Z"}}`
			}
			return 200, `{}`
		})
		// A missing envelope yields nil without an error.
		if got, err := api.Get(ctx, "u1", "missing"); err != nil || got != nil {
			t.Fatal(got, err)
		}
		_, err := api.Get(ctx, "u1", "zoned")
		nativeAppCredOperation(t, err, "Get")
	})
	t.Run("list status and empty page", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"application_credentials":[],"links":{"next":null}}`}} {
			var calls []nativeAppCredCall
			api, _ := nativeAppCredAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, tc.body })
			var errs []error
			for _, err := range api.List(ctx, "u1") {
				errs = append(errs, err)
			}
			if (tc.code == 404) != (len(errs) == 1) {
				t.Fatal(tc.code, errs)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeAppCredCall
		api, _ := nativeAppCredAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{}` })
		for name, err := range map[string]error{
			"name": func() error { _, err := api.Create(ctx, "u1", applicationcredentials.CreateOpts{}); return err }(),
			"core extension": func() error {
				_, err := api.Create(ctx, "u1", applicationcredentials.CreateOpts{Name: "ci"}, applicationcredentials.WithCreateField("secret", "x"))
				return err
			}(),
		} {
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeAppCredOperation(t, err, "Create")
		}
		for _, err := range api.List(ctx, "u1", nil) {
			nativeAppCredOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
