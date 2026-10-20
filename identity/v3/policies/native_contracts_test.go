package policies_test

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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/policies"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativePolicyTransport func(*http.Request) (*http.Response, error)

func (transport nativePolicyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativePolicyWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativePolicyCall struct{ method, path, query, body string }

func nativePolicyAPI(t *testing.T, calls *[]nativePolicyCall, reply func(*http.Request) *http.Response) (*policies.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativePolicyTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativePolicyCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return policies.New(client), cloud
}

func nativePolicyOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "policies" {
		t.Fatal("generated policies context", err, wrapped)
	}
}

const nativePolicyRow = `{"id":"pol-1","type":"application/json","blob":"{\"rule\":\"x\"}","links":{"self":"x"},"project_id":"p"}`

var nativePolicyCreate = policies.CreateOpts{Type: "application/json", Blob: []byte(`{"rule":"x"}`)}

func TestNativePolicyRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativePolicyCall
	var cloud *testcloud.Cloud
	api, cloud := nativePolicyAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativePolicyWire(204, "")
		case req.Method == http.MethodPost:
			return nativePolicyWire(201, `{"policy":`+nativePolicyRow+`}`)
		case req.URL.Path == "/keystone/v3/policies":
			return nativePolicyWire(200, `{"policies":[`+nativePolicyRow+`],"links":{"next":"`+cloud.Server.URL+`/other/policies?page=2"}}`)
		case req.URL.Path == "/other/policies":
			return nativePolicyWire(200, `{"policies":[{"id":"pol-2","type":"text/plain","extra":{"k":"v"}}],"links":{"next":null}}`)
		}
		return nativePolicyWire(200, `{"policy":`+nativePolicyRow+`}`)
	})
	create := nativePolicyCreate
	create.Extra = map[string]any{"project_id": "p"}
	created, err := api.Create(ctx, create, policies.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "pol-1" && created.Blob == `{"rule":"x"}` && reflect.DeepEqual(created.Extra, map[string]any{"project_id": "p"})) {
		t.Fatal(created, err)
	}
	// A non-nil empty blob passes the required check and is sent as an empty string.
	if _, err := api.Create(ctx, policies.CreateOpts{Type: "t", Blob: []byte{}}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "pol-1")
	if err != nil || got.Type != "application/json" {
		t.Fatal(got, err)
	}
	if _, err := api.Update(ctx, "pol-1", policies.UpdateOpts{Type: "text/plain", Blob: []byte("b"), Extra: map[string]any{"project_id": "q"}}); err != nil {
		t.Fatal(err)
	}
	// Update omits an empty blob, so an extension may carry a blob key.
	if _, err := api.Update(ctx, "pol-1", policies.UpdateOpts{}, policies.WithUpdateField("blob", "raw")); err != nil {
		t.Fatal(err)
	}
	var rows []*policies.Policy
	for value, err := range api.List(ctx, policies.WithListOptions(policies.ListOpts{Type: "application/json", Filters: map[string]string{"type__contains": "json"}}), policies.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "pol-1" && rows[1].ID == "pol-2" && reflect.DeepEqual(rows[1].Extra, map[string]any{"k": "v"})) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "pol-1"); err != nil {
		t.Fatal(err)
	}
	base := "/keystone/v3/policies"
	want := []nativePolicyCall{
		{http.MethodPost, base, "", `{"policy":{"blob":"{\"rule\":\"x\"}","project_id":"p","type":"application/json","x_extension":1}}`},
		{http.MethodPost, base, "", `{"policy":{"blob":"","type":"t"}}`},
		{http.MethodGet, base + "/pol-1", "", ""},
		{http.MethodPatch, base + "/pol-1", "", `{"policy":{"blob":"b","project_id":"q","type":"text/plain"}}`},
		{http.MethodPatch, base + "/pol-1", "", `{"policy":{"blob":"raw"}}`},
		// BuildQueryString does not skip the q:"-" Filters map, so it also sends a
		// "-" parameter holding a Python dict string next to the real filter.
		{http.MethodGet, base, "-=%7B%27type__contains%27%3A%27json%27%7D&extra=1&type=application%2Fjson&type__contains=json", ""},
		{http.MethodGet, "/other/policies", "page=2", ""},
		{http.MethodDelete, base + "/pol-1", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativePolicyStatusesDecodeAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*policies.API) error
	}{
		{"Create", []int{201}, func(api *policies.API) error { _, err := api.Create(ctx, nativePolicyCreate); return err }},
		{"Get", []int{200}, func(api *policies.API) error { _, err := api.Get(ctx, "pol-1"); return err }},
		{"Update", []int{200}, func(api *policies.API) error {
			_, err := api.Update(ctx, "pol-1", policies.UpdateOpts{})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *policies.API) error { return api.Delete(ctx, "pol-1") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativePolicyCall
				api, _ := nativePolicyAPI(t, &calls, func(*http.Request) *http.Response { return nativePolicyWire(code, `{}`) })
				err := call.call(api)
				nativePolicyOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("envelope decode", func(t *testing.T) {
		for body, wantErr := range map[string]bool{
			`{}`:              false,
			`{"policy":null}`: false,
			`{"policy":[]}`:   true,
		} {
			var calls []nativePolicyCall
			api, _ := nativePolicyAPI(t, &calls, func(*http.Request) *http.Response { return nativePolicyWire(200, body) })
			got, err := api.Get(ctx, "pol-1")
			if wantErr {
				nativePolicyOperation(t, err, "Get")
			} else if err != nil || got != nil {
				t.Fatal(body, got, err)
			}
		}
	})
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"policies":[]}`}, {204, ""}} {
			var calls []nativePolicyCall
			api, _ := nativePolicyAPI(t, &calls, func(*http.Request) *http.Response { return nativePolicyWire(tc.code, tc.body) })
			var errs []error
			for _, err := range api.List(ctx) {
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
		var calls []nativePolicyCall
		api, _ := nativePolicyAPI(t, &calls, func(*http.Request) *http.Response { return nativePolicyWire(201, `{}`) })
		long := strings.Repeat("t", 256)
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"create type": {"Create", func() error { _, err := api.Create(ctx, policies.CreateOpts{Blob: []byte("b")}); return err }()},
			"create blob": {"Create", func() error { _, err := api.Create(ctx, policies.CreateOpts{Type: "t"}); return err }()},
			"create type length": {"Create", func() error {
				_, err := api.Create(ctx, policies.CreateOpts{Type: long, Blob: []byte("b")})
				return err
			}()},
			"update type length": {"Update", func() error {
				_, err := api.Update(ctx, "pol-1", policies.UpdateOpts{Type: long})
				return err
			}()},
			// blob is not a tagged core field, but it is already in the create body.
			"create blob extension": {"Create", func() error {
				_, err := api.Create(ctx, nativePolicyCreate, policies.WithCreateField("blob", "x"))
				return err
			}()},
			"update core extension": {"Update", func() error {
				_, err := api.Update(ctx, "pol-1", policies.UpdateOpts{}, policies.WithUpdateField("type", "x"))
				return err
			}()},
			"create nil option": {"Create", func() error {
				_, err := api.Create(ctx, nativePolicyCreate, nil)
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativePolicyOperation(t, check.err, check.operation)
		}
		var length policies.StringFieldLengthExceedsLimit
		if _, err := api.Create(ctx, policies.CreateOpts{Type: long, Blob: []byte("b")}); !errors.As(err, &length) || length.Limit != 255 {
			t.Fatal(err)
		}
		for _, err := range api.List(ctx, nil) {
			nativePolicyOperation(t, err, "List")
		}
		// A filter without a TYPE__COMPARATOR shape fails in the native query builder, unwrapped.
		for _, err := range api.List(ctx, policies.WithListOptions(policies.ListOpts{Filters: map[string]string{"type": "x"}})) {
			var invalid policies.InvalidListFilter
			var wrapped *resource.OperationError
			if !errors.As(err, &invalid) || invalid.FilterName != "type" || errors.As(err, &wrapped) {
				t.Fatal(err)
			}
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
