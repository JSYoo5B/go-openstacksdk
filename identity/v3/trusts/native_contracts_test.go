package trusts_test

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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeTrustTransport func(*http.Request) (*http.Response, error)

func (transport nativeTrustTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeTrustCall struct{ method, path, query, body string }

func nativeTrustAPI(t *testing.T, calls *[]nativeTrustCall, reply func(*http.Request) (int, string)) (*trusts.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeTrustTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeTrustCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return trusts.New(client), cloud
}

func nativeTrustOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "trusts" {
		t.Fatal("generated trusts context", err, wrapped)
	}
}

const nativeTrustRow = `{"id":"tr-1","impersonation":true,"trustee_user_id":"bob","trustor_user_id":"alice","project_id":"p","remaining_uses":3,"roles":[{"id":"r1","name":"member"}],"expires_at":"2026-10-11T01:02:03.000000Z","deleted_at":null}`

func TestNativeTrustRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeTrustCall
	var cloud *testcloud.Cloud
	api, cloud := nativeTrustAPI(t, &calls, func(req *http.Request) (int, string) {
		switch {
		case req.Method == http.MethodDelete:
			return 204, ""
		case req.Method == http.MethodHead:
			return 200, ""
		case req.Method == http.MethodPost:
			return 201, `{"trust":` + nativeTrustRow + `}`
		case req.URL.Path == "/keystone/v3/OS-TRUST/trusts":
			return 200, `{"trusts":[` + nativeTrustRow + `],"links":{"next":"` + cloud.Server.URL + `/other/trusts?page=2"}}`
		case req.URL.Path == "/other/trusts":
			return 200, `{"trusts":[{"id":"tr-2"}],"links":{"next":null}}`
		case req.URL.Path == "/keystone/v3/OS-TRUST/trusts/tr-1/roles":
			return 200, `{"roles":[{"id":"r1","name":"member"}],"links":{"next":null}}`
		case req.URL.Path == "/keystone/v3/OS-TRUST/trusts/tr-1/roles/r1":
			return 200, `{"role":{"id":"r1","name":"member"}}`
		}
		return 200, `{"trust":` + nativeTrustRow + `}`
	})
	expires := time.Date(2026, 10, 11, 1, 2, 3, 0, time.UTC)
	created, err := api.Create(ctx, trusts.CreateOpts{TrusteeUserID: "bob", TrustorUserID: "alice", Impersonation: true, ProjectID: "p", RemainingUses: 3, Roles: []trusts.Role{{Name: "member"}}, ExpiresAt: &expires}, trusts.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "tr-1" && created.Impersonation && created.RemainingUses == 3 && created.ExpiresAt.Equal(expires) && created.DeletedAt.IsZero()) {
		t.Fatal(created, err)
	}
	// Impersonation has no omitempty, so false is always sent; a non-UTC expiry keeps its clock with a literal Z.
	local := time.Date(2026, 10, 11, 10, 2, 3, 0, time.FixedZone("KST", 9*3600))
	if _, err := api.Create(ctx, trusts.CreateOpts{TrusteeUserID: "bob", TrustorUserID: "alice", ExpiresAt: &local}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "tr-1")
	if err != nil || got.ProjectID != "p" {
		t.Fatal(got, err)
	}
	var ids []string
	for value, err := range api.List(ctx, trusts.WithListOptions(trusts.ListOpts{TrustorUserID: "alice"}), trusts.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	var roles []string
	for value, err := range api.ListRoles(ctx, "tr-1") {
		if err != nil {
			t.Fatal(err)
		}
		roles = append(roles, value.Name)
	}
	role, err := api.GetRole(ctx, "tr-1", "r1")
	if err != nil || role.Name != "member" {
		t.Fatal(role, err)
	}
	if err := api.CheckRole(ctx, "tr-1", "r1"); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "tr-1"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, []string{"tr-1", "tr-2"}) || !reflect.DeepEqual(roles, []string{"member"}) {
		t.Fatal(ids, roles)
	}
	base := "/keystone/v3/OS-TRUST/trusts"
	want := []nativeTrustCall{
		{http.MethodPost, base, "", `{"trust":{"expires_at":"2026-10-11T01:02:03Z","impersonation":true,"project_id":"p","remaining_uses":3,"roles":[{"name":"member"}],"trustee_user_id":"bob","trustor_user_id":"alice","x_extension":1}}`},
		{http.MethodPost, base, "", `{"trust":{"expires_at":"2026-10-11T10:02:03Z","impersonation":false,"trustee_user_id":"bob","trustor_user_id":"alice"}}`},
		{http.MethodGet, base + "/tr-1", "", ""},
		{http.MethodGet, base, "extra=1&trustor_user_id=alice", ""},
		{http.MethodGet, "/other/trusts", "page=2", ""},
		{http.MethodGet, base + "/tr-1/roles", "", ""},
		{http.MethodGet, base + "/tr-1/roles/r1", "", ""},
		{http.MethodHead, base + "/tr-1/roles/r1", "", ""},
		{http.MethodDelete, base + "/tr-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeTrustStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	valid := trusts.CreateOpts{TrusteeUserID: "bob", TrustorUserID: "alice"}
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*trusts.API) error
	}{
		{"Create", []int{201}, func(api *trusts.API) error { _, err := api.Create(ctx, valid); return err }},
		{"Get", []int{200}, func(api *trusts.API) error { _, err := api.Get(ctx, "tr-1"); return err }},
		{"GetRole", []int{200}, func(api *trusts.API) error { _, err := api.GetRole(ctx, "tr-1", "r1"); return err }},
		// CheckRole uses the native HEAD default, which accepts only 200.
		{"CheckRole", []int{200}, func(api *trusts.API) error { return api.CheckRole(ctx, "tr-1", "r1") }},
		{"Delete", []int{202, 204}, func(api *trusts.API) error { return api.Delete(ctx, "tr-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeTrustCall
				api, _ := nativeTrustAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
				err := call.call(api)
				nativeTrustOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeTrustCall
		api, _ := nativeTrustAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{}` })
		for name, err := range map[string]error{
			"trustee": func() error { _, err := api.Create(ctx, trusts.CreateOpts{TrustorUserID: "alice"}); return err }(),
			"trustor": func() error { _, err := api.Create(ctx, trusts.CreateOpts{TrusteeUserID: "bob"}); return err }(),
			"core extension": func() error {
				_, err := api.Create(ctx, valid, trusts.WithCreateField("impersonation", true))
				return err
			}(),
		} {
			if err == nil {
				t.Fatal(name, "accepted")
			}
			nativeTrustOperation(t, err, "Create")
		}
		for _, err := range api.List(ctx, nil) {
			nativeTrustOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
