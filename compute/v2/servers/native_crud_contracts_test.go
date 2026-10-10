package servers_test

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

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeServerTransport func(*http.Request) (*http.Response, error)

func (transport nativeServerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeServerWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}, "X-Server-Proof": {"actual"}}}
}

func nativeServerClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	client.MoreHeaders = map[string]string{"X-Source": "direct"}
	return client
}

func nativeServerOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "servers" {
		t.Fatal("generated server context", err, wrapped)
	}
	var receipt *resource.ResponseError
	if errors.As(err, &receipt) {
		t.Fatal("native server call fabricated an owned receipt", receipt)
	}
}

const nativeServerRow = `{"id":"s1","name":"vm","status":"ACTIVE","created":"2026-10-10T01:02:03Z","image":"","flavor":{"id":"f1"},"addresses":{"net":[{"addr":"10.0.0.2","version":4}]},"metadata":{"k":"v"},"tags":["t"],"OS-EXT-SRV-ATTR:host":"h"}`

type nativeServerCall struct{ method, path, query, body string }

func nativeServerRecorder(t *testing.T, cloud *testcloud.Cloud, calls *[]nativeServerCall, reply func(*http.Request) *http.Response) {
	t.Helper()
	cloud.Provider.HTTPClient.Transport = nativeServerTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		if req.Header.Get("X-Source") != "direct" || req.Header.Get("X-Auth-Token") != "test-token" {
			t.Error(req.Header)
		}
		*calls = append(*calls, nativeServerCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
}

func TestNativeServerCRUDRoutesBodiesAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	client := nativeServerClient(cloud)
	var calls []nativeServerCall
	nativeServerRecorder(t, cloud, &calls, func(req *http.Request) *http.Response {
		switch req.Method {
		case http.MethodPost:
			return nativeServerWire(202, `{"server":{"id":"s1","adminPass":"secret"}}`)
		case http.MethodDelete:
			return nativeServerWire(204, "")
		case http.MethodPut:
			return nativeServerWire(200, `{"server":`+nativeServerRow+`}`)
		}
		return nativeServerWire(203, `{"server":`+nativeServerRow+`}`)
	})
	api := servers.New(client)
	if api.RawClient() != client {
		t.Fatal("native client identity changed")
	}
	ctx := context.Background()
	got, err := api.Get(ctx, "s1")
	if err != nil || got.ID != "s1" || got.Status != "ACTIVE" || got.Image != nil || got.Flavor["id"] != "f1" || got.Metadata["k"] != "v" || got.Tags == nil || (*got.Tags)[0] != "t" || got.Host != "h" || !got.Created.Equal(time.Date(2026, 10, 10, 1, 2, 3, 0, time.UTC)) {
		t.Fatal("203 is accepted and the empty image string decodes to nil", got, err)
	}
	config := true
	created, err := api.Create(ctx, servers.CreateOpts{Name: "vm", ImageRef: "img", FlavorRef: "f1", Networks: "auto", SecurityGroups: []string{"default"}, UserData: []byte("#!/bin/sh"), ConfigDrive: &config, Min: 1, Max: 2},
		servers.WithCreateField("x_extension", 1), servers.WithCreateHintOpts(servers.SchedulerHintOpts{Query: []any{"=", "$free_ram_mb", 1024}}))
	if err != nil || created.ID != "s1" || created.AdminPass != "secret" {
		t.Fatal(created, err)
	}
	hostname := "host"
	updated, err := api.Update(ctx, "s1", servers.UpdateOpts{Name: "renamed", Hostname: &hostname})
	if err != nil || updated.ID != "s1" {
		t.Fatal(updated, err)
	}
	if err := api.Delete(ctx, "a/b"); err != nil {
		t.Fatal(err)
	}
	want := []nativeServerCall{
		{http.MethodGet, "/nova/v2.1/servers/s1", "", ""},
		// The single server envelope receives the extension; hints stay top level.
		{http.MethodPost, "/nova/v2.1/servers", "", `{"os:scheduler_hints":{"query":"[\"=\",\"$free_ram_mb\",1024]"},"server":{"config_drive":true,"flavorRef":"f1","imageRef":"img","max_count":2,"min_count":1,"name":"vm","networks":"auto","security_groups":[{"name":"default"}],"user_data":"IyEvYmluL3No","x_extension":1}}`},
		{http.MethodPut, "/nova/v2.1/servers/s1", "", `{"server":{"hostname":"host","name":"renamed"}}`},
		{http.MethodDelete, "/nova/v2.1/servers/a/b", "", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeServerCRUDStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*servers.API) error
	}{
		{"Get", []int{200, 203}, func(api *servers.API) error { _, err := api.Get(ctx, "s1"); return err }},
		{"Create", []int{200, 202}, func(api *servers.API) error {
			_, err := api.Create(ctx, servers.CreateOpts{Name: "vm", FlavorRef: "f"})
			return err
		}},
		{"Update", []int{200}, func(api *servers.API) error {
			_, err := api.Update(ctx, "s1", servers.UpdateOpts{Name: "n"})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *servers.API) error { return api.Delete(ctx, "s1") }},
	} {
		for _, code := range []int{200, 201, 202, 203, 204, 404, 409} {
			if contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Provider.HTTPClient.Transport = nativeServerTransport(func(req *http.Request) (*http.Response, error) {
					requests.Add(1)
					return nativeServerWire(code, `{"server":`+nativeServerRow+`}`), nil
				})
				err := call.call(servers.New(nativeServerClient(cloud)))
				nativeServerOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || native.ResponseHeader.Get("X-Server-Proof") != "actual" || requests.Load() != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("required name, invalid networks, collisions and nil options", func(t *testing.T) {
		cloud := testcloud.New(t)
		var requests atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeServerTransport(func(req *http.Request) (*http.Response, error) {
			requests.Add(1)
			return nativeServerWire(202, `{"server":{}}`), nil
		})
		api := servers.New(nativeServerClient(cloud))
		checks := map[string]func() error{
			"missing name": func() error { _, err := api.Create(ctx, servers.CreateOpts{FlavorRef: "f"}); return err },
			"invalid networks string": func() error {
				_, err := api.Create(ctx, servers.CreateOpts{Name: "vm", Networks: "other"})
				return err
			},
			"envelope collision": func() error {
				_, err := api.Create(ctx, servers.CreateOpts{Name: "vm"}, servers.WithCreateField("name", "x"))
				return err
			},
			"invalid group hint": func() error {
				_, err := api.Create(ctx, servers.CreateOpts{Name: "vm"}, servers.WithCreateHintOpts(servers.SchedulerHintOpts{Group: "not-a-uuid"}))
				return err
			},
			"nil option": func() error { _, err := api.Update(ctx, "s1", servers.UpdateOpts{}, nil); return err },
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

// List uses servers/detail and ListSimple uses servers; both follow the
// servers_links next href verbatim.
func TestNativeServerListsQueryPagingAndStop(t *testing.T) {
	for _, simple := range []bool{false, true} {
		t.Run(fmt.Sprint("simple=", simple), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls []nativeServerCall
			nativeServerRecorder(t, cloud, &calls, func(req *http.Request) *http.Response {
				if req.URL.Query().Get("marker") == "" {
					return nativeServerWire(200, `{"servers":[{"id":"a","name":"x"}],"servers_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/servers?marker=a"}]}`)
				}
				return nativeServerWire(200, `{"servers":[{"id":"b"}]}`)
			})
			api := servers.New(nativeServerClient(cloud))
			opts := servers.ListOpts{Name: "x", Status: "ACTIVE", Limit: 1, Tags: "t", AllTenants: true}
			var seq func(func(*servers.Server, error) bool)
			if simple {
				seq = api.ListSimple(context.Background(), servers.WithListSimpleOptions(opts), servers.WithListSimpleQuery("extra", "1"))
			} else {
				seq = api.List(context.Background(), servers.WithListOptions(opts), servers.WithListQuery("extra", "1"))
			}
			var ids []string
			for value, err := range seq {
				if err != nil {
					t.Fatal(err)
				}
				ids = append(ids, value.ID)
			}
			path := "/nova/v2.1/servers/detail"
			if simple {
				path = "/nova/v2.1/servers"
			}
			first, _ := url.ParseQuery(calls[0].query)
			want := url.Values{"name": {"x"}, "status": {"ACTIVE"}, "limit": {"1"}, "tags": {"t"}, "all_tenants": {"true"}, "extra": {"1"}}
			if !reflect.DeepEqual(ids, []string{"a", "b"}) || calls[0].path != path || !reflect.DeepEqual(first, want) || calls[1].path != "/other/servers" {
				t.Fatal(ids, calls)
			}
		})
	}
	t.Run("early stop", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls []nativeServerCall
		nativeServerRecorder(t, cloud, &calls, func(req *http.Request) *http.Response {
			return nativeServerWire(200, `{"servers":[{"id":"a"},{"id":"b"}],"servers_links":[{"rel":"next","href":"`+cloud.Server.URL+`/nova/v2.1/servers/detail?marker=b"}]}`)
		})
		for value, err := range servers.New(nativeServerClient(cloud)).List(context.Background()) {
			if err != nil || value.ID != "a" {
				t.Fatal(value, err)
			}
			break
		}
		if len(calls) != 1 {
			t.Fatal(calls)
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
