package federation_test

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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/federation"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeMappingTransport func(*http.Request) (*http.Response, error)

func (transport nativeMappingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeMappingWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeMappingCall struct{ method, path, query, body string }

func nativeMappingAPI(t *testing.T, calls *[]nativeMappingCall, reply func(*http.Request) *http.Response) (*federation.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeMappingTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeMappingCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return federation.New(client), cloud
}

func nativeMappingOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "federation" {
		t.Fatal("generated federation context", err, wrapped)
	}
}

const nativeMappingRow = `{"id":"ACME","links":{"self":"x"},"rules":[{"local":[{"user":{"name":"{0}","type":"ephemeral","domain":{"id":"d"}}},{"group":{"id":"g-1"}},{"projects":[{"name":"p","roles":[{"name":"member"}]}]}],"remote":[{"type":"REMOTE_USER"},{"type":"groups","any_one_of":["admins"],"regex":true}]}]}`

func nativeMappingRules() []federation.MappingRule {
	ephemeral, regex := federation.UserTypeEphemeral, true
	return []federation.MappingRule{{
		Local: []federation.RuleLocal{
			{User: &federation.RuleUser{Name: "{0}", Type: &ephemeral, Domain: &federation.Domain{ID: "d"}}},
			{Group: &federation.Group{ID: "g-1"}},
			{Projects: []federation.RuleProject{{Name: "p", Roles: []federation.RuleProjectRole{{Name: "member"}}}}},
		},
		Remote: []federation.RuleRemote{{Type: "REMOTE_USER"}, {Type: "groups", AnyOneOf: []string{"admins"}, Regex: &regex}},
	}}
}

func TestNativeMappingRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeMappingCall
	var cloud *testcloud.Cloud
	api, cloud := nativeMappingAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeMappingWire(204, "")
		case req.Method == http.MethodPut:
			return nativeMappingWire(201, `{"mapping":`+nativeMappingRow+`}`)
		case req.URL.Path == "/keystone/v3/OS-FEDERATION/mappings":
			return nativeMappingWire(200, `{"mappings":[`+nativeMappingRow+`],"links":{"next":"`+cloud.Server.URL+`/other/mappings?page=2"}}`)
		case req.URL.Path == "/other/mappings":
			return nativeMappingWire(200, `{"mappings":[{"id":"OTHER","rules":[]}],"links":{"next":null}}`)
		}
		return nativeMappingWire(200, `{"mapping":`+nativeMappingRow+`}`)
	})
	created, err := api.CreateMapping(ctx, "ACME", federation.CreateMappingOpts{Rules: nativeMappingRules()}, federation.WithCreateMappingField("schema_version", "2.0"))
	if err != nil || !(created.ID == "ACME" && reflect.DeepEqual(created.Rules, nativeMappingRules())) {
		t.Fatal(created, err)
	}
	// Rules has no omitempty, so empty options send a null rule list.
	if _, err := api.CreateMapping(ctx, "EMPTY", federation.CreateMappingOpts{}); err != nil {
		t.Fatal(err)
	}
	got, err := api.GetMapping(ctx, "ACME")
	if err != nil || *got.Rules[0].Local[0].User.Type != federation.UserTypeEphemeral {
		t.Fatal(got, err)
	}
	// A bare rule keeps local and remote as null and always sends the remote type.
	if _, err := api.UpdateMapping(ctx, "ACME", federation.UpdateMappingOpts{Rules: []federation.MappingRule{{}, {Remote: []federation.RuleRemote{{}}}}}); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for value, err := range api.ListMappings(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"ACME", "OTHER"}) {
		t.Fatal(ids)
	}
	if err := api.DeleteMapping(ctx, "ACME"); err != nil {
		t.Fatal(err)
	}
	base := "/keystone/v3/OS-FEDERATION/mappings"
	rules := `[{"local":[{"user":{"domain":{"id":"d"},"name":"{0}","type":"ephemeral"}},{"group":{"id":"g-1"}},{"projects":[{"name":"p","roles":[{"name":"member"}]}]}],"remote":[{"type":"REMOTE_USER"},{"any_one_of":["admins"],"regex":true,"type":"groups"}]}]`
	want := []nativeMappingCall{
		// Create is a PUT to the caller-chosen mapping ID.
		{http.MethodPut, base + "/ACME", "", `{"mapping":{"rules":` + rules + `,"schema_version":"2.0"}}`},
		{http.MethodPut, base + "/EMPTY", "", `{"mapping":{"rules":null}}`},
		{http.MethodGet, base + "/ACME", "", ""},
		{http.MethodPatch, base + "/ACME", "", `{"mapping":{"rules":[{"local":null,"remote":null},{"local":null,"remote":[{"type":""}]}]}}`},
		{http.MethodGet, base, "", ""},
		{http.MethodGet, "/other/mappings", "page=2", ""},
		{http.MethodDelete, base + "/ACME", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeMappingStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*federation.API) error
	}{
		{"CreateMapping", []int{201}, func(api *federation.API) error {
			_, err := api.CreateMapping(ctx, "ACME", federation.CreateMappingOpts{Rules: nativeMappingRules()})
			return err
		}},
		{"GetMapping", []int{200}, func(api *federation.API) error { _, err := api.GetMapping(ctx, "ACME"); return err }},
		{"UpdateMapping", []int{200}, func(api *federation.API) error {
			_, err := api.UpdateMapping(ctx, "ACME", federation.UpdateMappingOpts{Rules: nativeMappingRules()})
			return err
		}},
		{"DeleteMapping", []int{202, 204}, func(api *federation.API) error { return api.DeleteMapping(ctx, "ACME") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeMappingCall
				api, _ := nativeMappingAPI(t, &calls, func(*http.Request) *http.Response { return nativeMappingWire(code, `{}`) })
				err := call.call(api)
				nativeMappingOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			`{}`:                                  false,
			`{"mapping":null}`:                    false,
			`{"mapping":[]}`:                      true,
			`{"mapping":{"rules":[{"local":1}]}}`: true,
		} {
			var calls []nativeMappingCall
			api, _ := nativeMappingAPI(t, &calls, func(*http.Request) *http.Response { return nativeMappingWire(200, body) })
			got, err := api.GetMapping(ctx, "ACME")
			if wantErr {
				nativeMappingOperation(t, err, "GetMapping")
			} else if err != nil || got != nil {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"mappings":[]}`}, {204, ""}} {
			var calls []nativeMappingCall
			api, _ := nativeMappingAPI(t, &calls, func(*http.Request) *http.Response { return nativeMappingWire(tc.code, tc.body) })
			var errs []error
			for _, err := range api.ListMappings(ctx) {
				errs = append(errs, err)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			switch {
			case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
			case tc.code == 200 && len(errs) == 0:
			case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
			default:
				t.Fatal(tc.code, errs)
			}
		}
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeMappingCall
		api, _ := nativeMappingAPI(t, &calls, func(*http.Request) *http.Response { return nativeMappingWire(201, `{}`) })
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create core extension": {"CreateMapping", func() error {
				_, err := api.CreateMapping(ctx, "ACME", federation.CreateMappingOpts{}, federation.WithCreateMappingField("rules", []any{}))
				return err
			}()},
			"update core extension": {"UpdateMapping", func() error {
				_, err := api.UpdateMapping(ctx, "ACME", federation.UpdateMappingOpts{}, federation.WithUpdateMappingField("rules", []any{}))
				return err
			}()},
			"create nil option": {"CreateMapping", func() error {
				_, err := api.CreateMapping(ctx, "ACME", federation.CreateMappingOpts{}, nil)
				return err
			}()},
			"update nil option": {"UpdateMapping", func() error {
				_, err := api.UpdateMapping(ctx, "ACME", federation.UpdateMappingOpts{}, nil)
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeMappingOperation(t, check.err, check.operation)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
