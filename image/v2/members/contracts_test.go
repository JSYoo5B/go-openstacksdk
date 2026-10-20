package members_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/members"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type memberTransport func(*http.Request) (*http.Response, error)

func (transport memberTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func memberWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}, "X-Member-Proof": {"actual"}}}
}

func memberClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("image", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/reverse/glance/v2/"
	client.MoreHeaders = map[string]string{"X-Source": "direct"}
	return client
}

func memberOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "members" {
		t.Fatal("generated member context", err, wrapped)
	}
	var receipt *resource.ResponseError
	if errors.As(err, &receipt) {
		t.Fatal("native member call fabricated an owned receipt", receipt)
	}
}

const memberRow = `{"created_at":"2026-10-10T00:00:00Z","image_id":"img","member_id":"project","schema":"/v2/schemas/member","status":"pending","updated_at":"2026-10-10T00:00:01Z"}`

// Each generated method keeps the native raw ServiceURL segments, fixed body
// and the pinned single OkCodes value, then decodes the native Member.
func TestNativeImageMembersRoutesBodiesAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	client := memberClient(cloud)
	type exchange struct{ method, path, body string }
	var seen []exchange
	cloud.Provider.HTTPClient.Transport = memberTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		if req.Header.Get("Accept") != "application/json" || req.Header.Get("X-Source") != "direct" || req.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(req.Header)
		}
		seen = append(seen, exchange{req.Method, req.URL.Path, raw})
		switch req.Method {
		case http.MethodDelete:
			return memberWire(204, ""), nil
		case http.MethodGet:
			if strings.HasSuffix(req.URL.Path, "/members") {
				return memberWire(200, `{"members":[`+memberRow+`,{"member_id":"other"}],"schema":"/v2/schemas/members"}`), nil
			}
		}
		return memberWire(200, memberRow), nil
	})
	api := members.New(client)
	if api.RawClient() != client {
		t.Fatal("native client identity changed")
	}
	ctx := context.Background()
	created, err := api.Create(ctx, "img", "project")
	if err != nil || created.MemberID != "project" || created.ImageID != "img" || created.Status != "pending" || !created.CreatedAt.Equal(time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)) || created.Schema != "/v2/schemas/member" {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "img", "project")
	if err != nil || got.MemberID != "project" {
		t.Fatal(got, err)
	}
	var ids []string
	for member, err := range api.List(ctx, "img") {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, member.MemberID)
	}
	updated, err := api.Update(ctx, "img", "project", members.UpdateOpts{Status: "accepted"})
	if err != nil || updated.Status != "pending" {
		t.Fatal("native Update returns the decoded response, not the request", updated, err)
	}
	if _, err := api.Update(ctx, "img", "project", members.UpdateOpts{}, members.WithUpdateField("note", map[string]any{"n": 1})); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "img", "project"); err != nil {
		t.Fatal(err)
	}
	// Raw IDs are joined without escaping, like other native ServiceURL calls.
	if _, err := api.Get(ctx, "a/b", "c?d=1"); err != nil {
		t.Fatal(err)
	}
	want := []exchange{
		{http.MethodPost, "/reverse/glance/v2/images/img/members", `{"member":"project"}`},
		{http.MethodGet, "/reverse/glance/v2/images/img/members/project", ""},
		{http.MethodGet, "/reverse/glance/v2/images/img/members", ""},
		{http.MethodPut, "/reverse/glance/v2/images/img/members/project", `{"status":"accepted"}`},
		{http.MethodPut, "/reverse/glance/v2/images/img/members/project", `{"note":{"n":1},"status":""}`},
		{http.MethodDelete, "/reverse/glance/v2/images/img/members/project", ""},
		{http.MethodGet, "/reverse/glance/v2/images/a/b/members/c", ""},
	}
	if !reflect.DeepEqual(ids, []string{"project", "other"}) || !reflect.DeepEqual(seen, want) {
		t.Fatal(ids, seen)
	}
}

func TestNativeImageMembersStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	calls := []struct {
		name     string
		accepted int
		call     func(*members.API) error
	}{
		{"Create", 200, func(api *members.API) error { _, err := api.Create(ctx, "img", "p"); return err }},
		{"Get", 200, func(api *members.API) error { _, err := api.Get(ctx, "img", "p"); return err }},
		{"Update", 200, func(api *members.API) error {
			_, err := api.Update(ctx, "img", "p", members.UpdateOpts{Status: "accepted"})
			return err
		}},
		{"Delete", 204, func(api *members.API) error { return api.Delete(ctx, "img", "p") }},
	}
	for _, call := range calls {
		for _, code := range []int{200, 201, 202, 204, 404, 409} {
			if code == call.accepted {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Provider.HTTPClient.Transport = memberTransport(func(req *http.Request) (*http.Response, error) {
					requests.Add(1)
					return memberWire(code, memberRow), nil
				})
				err := call.call(members.New(memberClient(cloud)))
				memberOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{call.accepted}) || native.ResponseHeader.Get("X-Member-Proof") != "actual" || requests.Load() != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("List rejection and empty single page", func(t *testing.T) {
		cloud := testcloud.New(t)
		code := 403
		cloud.Provider.HTTPClient.Transport = memberTransport(func(req *http.Request) (*http.Response, error) {
			return memberWire(code, `{"members":[]}`), nil
		})
		api := members.New(memberClient(cloud))
		for _, err := range api.List(ctx, "img") {
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != 403 {
				t.Fatal(err)
			}
		}
		code = 200
		count := 0
		for member, err := range api.List(ctx, "img") {
			count++
			t.Fatal("empty page yielded", member, err)
		}
		if count != 0 {
			t.Fatal(count)
		}
	})
	t.Run("Update extension collision and nil option", func(t *testing.T) {
		cloud := testcloud.New(t)
		var requests atomic.Int32
		cloud.Provider.HTTPClient.Transport = memberTransport(func(req *http.Request) (*http.Response, error) {
			requests.Add(1)
			return memberWire(200, memberRow), nil
		})
		api := members.New(memberClient(cloud))
		for _, option := range []members.UpdateOption{members.WithUpdateField("status", "x"), nil} {
			_, err := api.Update(ctx, "img", "p", members.UpdateOpts{Status: "accepted"}, option)
			memberOperation(t, err, "Update")
			if requests.Load() != 0 {
				t.Fatal(err, requests.Load())
			}
		}
	})
}
