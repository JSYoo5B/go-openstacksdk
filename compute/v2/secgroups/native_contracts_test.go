package secgroups_test

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

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeSecGroupTransport func(*http.Request) (*http.Response, error)

func (transport nativeSecGroupTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeSecGroupCall struct{ method, path, body string }

func nativeSecGroupAPI(t *testing.T, calls *[]nativeSecGroupCall, reply func(*http.Request) (int, string)) *secgroups.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeSecGroupTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeSecGroupCall{req.Method, req.URL.Path, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	return secgroups.New(client)
}

func nativeSecGroupOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "secgroups" {
		t.Fatal("generated secgroups context", err, wrapped)
	}
}

// Nova returns numeric IDs on nova-network and UUID strings on Neutron.
const nativeSecGroupRow = `{"id":7,"name":"web","description":"d","tenant_id":"p","rules":[{"id":"r-1","parent_group_id":7,"from_port":22,"to_port":22,"ip_protocol":"tcp","ip_range":{"cidr":"0.0.0.0/0"},"group":{"tenant_id":"p","name":"web"}}]}`

func TestNativeSecurityGroupRoutesBodiesAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeSecGroupCall
	api := nativeSecGroupAPI(t, &calls, func(req *http.Request) (int, string) {
		switch {
		case req.Method == http.MethodDelete:
			return 202, ""
		case strings.HasSuffix(req.URL.Path, "/action"):
			return 202, ""
		case req.URL.Path == "/nova/v2.1/os-security-group-rules":
			return 200, `{"security_group_rule":{"id":9,"parent_group_id":"sg-uuid","from_port":0,"to_port":0,"ip_protocol":"icmp","ip_range":{},"group":{"name":"peer"}}}`
		case req.Method == http.MethodGet && (req.URL.Path == "/nova/v2.1/os-security-groups" || strings.HasSuffix(req.URL.Path, "/os-security-groups")):
			return 200, `{"security_groups":[` + nativeSecGroupRow + `,{"id":"uuid-2","name":"default","rules":[]}]}`
		}
		return 200, `{"security_group":` + nativeSecGroupRow + `}`
	})
	created, err := api.Create(ctx, secgroups.CreateOpts{Name: "web", Description: "d"}, secgroups.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "7" && created.Rules[0].ID == "r-1" && created.Rules[0].ParentGroupID == "7" && created.Rules[0].IPRange.CIDR == "0.0.0.0/0" && created.Rules[0].Group.Name == "web") {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "7")
	if err != nil || got.TenantID != "p" {
		t.Fatal(got, err)
	}
	empty := ""
	if _, err := api.Update(ctx, "7", secgroups.UpdateOpts{Description: &empty}); err != nil {
		t.Fatal(err)
	}
	collect := func(seq func(func(*secgroups.SecurityGroup, error) bool)) []string {
		var ids []string
		for value, err := range seq {
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, value.ID)
		}
		return ids
	}
	all := collect(api.List(ctx))
	byServer := collect(api.ListByServer(ctx, "srv"))
	rule, err := api.CreateRule(ctx, secgroups.CreateRuleOpts{ParentGroupID: "sg-uuid", IPProtocol: "icmp", FromGroupID: "peer"})
	if err != nil || rule.ID != "9" || rule.ParentGroupID != "sg-uuid" || rule.Group.Name != "peer" {
		t.Fatal(rule, err)
	}
	if err := api.DeleteRule(ctx, "9"); err != nil {
		t.Fatal(err)
	}
	if err := api.AddServer(ctx, "srv", "web"); err != nil {
		t.Fatal(err)
	}
	if err := api.RemoveServer(ctx, "srv", "web"); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "7"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(all, []string{"7", "uuid-2"}) || !reflect.DeepEqual(byServer, all) {
		t.Fatal(all, byServer)
	}
	base := "/nova/v2.1/"
	want := []nativeSecGroupCall{
		{http.MethodPost, base + "os-security-groups", `{"security_group":{"description":"d","name":"web","x_extension":1}}`},
		{http.MethodGet, base + "os-security-groups/7", ""},
		{http.MethodPut, base + "os-security-groups/7", `{"security_group":{"description":""}}`},
		{http.MethodGet, base + "os-security-groups", ""},
		{http.MethodGet, base + "servers/srv/os-security-groups", ""},
		// Port bounds have no omitempty, so zero is sent; a peer group replaces the CIDR.
		{http.MethodPost, base + "os-security-group-rules", `{"security_group_rule":{"from_port":0,"group_id":"peer","ip_protocol":"icmp","parent_group_id":"sg-uuid","to_port":0}}`},
		{http.MethodDelete, base + "os-security-group-rules/9", ""},
		{http.MethodPost, base + "servers/srv/action", `{"addSecurityGroup":{"name":"web"}}`},
		{http.MethodPost, base + "servers/srv/action", `{"removeSecurityGroup":{"name":"web"}}`},
		{http.MethodDelete, base + "os-security-groups/7", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeSecurityGroupStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	rule := secgroups.CreateRuleOpts{ParentGroupID: "7", IPProtocol: "tcp", CIDR: "0.0.0.0/0"}
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*secgroups.API) error
	}{
		// Create and CreateRule accept only 200.
		{"Create", []int{200}, func(api *secgroups.API) error {
			_, err := api.Create(ctx, secgroups.CreateOpts{Name: "web"})
			return err
		}},
		{"Get", []int{200}, func(api *secgroups.API) error { _, err := api.Get(ctx, "7"); return err }},
		{"Update", []int{200}, func(api *secgroups.API) error {
			_, err := api.Update(ctx, "7", secgroups.UpdateOpts{})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *secgroups.API) error { return api.Delete(ctx, "7") }},
		{"CreateRule", []int{200}, func(api *secgroups.API) error { _, err := api.CreateRule(ctx, rule); return err }},
		{"DeleteRule", []int{202, 204}, func(api *secgroups.API) error { return api.DeleteRule(ctx, "9") }},
		// The server actions use the native POST default.
		{"AddServer", []int{201, 202}, func(api *secgroups.API) error { return api.AddServer(ctx, "srv", "web") }},
		{"RemoveServer", []int{201, 202}, func(api *secgroups.API) error { return api.RemoveServer(ctx, "srv", "web") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeSecGroupCall
				api := nativeSecGroupAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
				err := call.call(api)
				nativeSecGroupOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("list status and empty page", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"security_groups":[]}`}} {
			var calls []nativeSecGroupCall
			api := nativeSecGroupAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, tc.body })
			var errs []error
			for _, err := range api.List(ctx) {
				errs = append(errs, err)
			}
			if (tc.code == 404) != (len(errs) == 1) {
				t.Fatal(tc.code, errs)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeSecGroupCall
		api := nativeSecGroupAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{}` })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"name": {"Create", func() error { _, err := api.Create(ctx, secgroups.CreateOpts{}); return err }()},
			"parent": {"CreateRule", func() error {
				_, err := api.CreateRule(ctx, secgroups.CreateRuleOpts{IPProtocol: "tcp", CIDR: "c"})
				return err
			}()},
			"protocol": {"CreateRule", func() error {
				_, err := api.CreateRule(ctx, secgroups.CreateRuleOpts{ParentGroupID: "7", CIDR: "c"})
				return err
			}()},
			"no target": {"CreateRule", func() error {
				_, err := api.CreateRule(ctx, secgroups.CreateRuleOpts{ParentGroupID: "7", IPProtocol: "tcp"})
				return err
			}()},
			"core extension": {"Create", func() error {
				_, err := api.Create(ctx, secgroups.CreateOpts{Name: "web"}, secgroups.WithCreateField("name", "x"))
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeSecGroupOperation(t, check.err, check.operation)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
