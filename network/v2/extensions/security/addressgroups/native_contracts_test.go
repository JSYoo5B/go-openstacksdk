package addressgroups_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeGroupTransport func(*http.Request) (*http.Response, error)

func (transport nativeGroupTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeGroupWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeGroupCall struct{ method, path, query, body string }

func nativeGroupAPI(t *testing.T, calls *[]nativeGroupCall, reply func(*http.Request) *http.Response) (*addressgroups.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeGroupTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeGroupCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return addressgroups.New(client), cloud
}

func nativeGroupOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "addressgroups" {
		t.Fatal("generated addressgroups context", err, wrapped)
	}
}

const nativeGroupRow = `{"id":"ag-1","name":"web","description":null,"project_id":"p","addresses":["10.0.0.0/24","2001:db8::/64"]}`

func TestNativeAddressGroupRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeGroupCall
	var cloud *testcloud.Cloud
	api, cloud := nativeGroupAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeGroupWire(204, "")
		case req.Method == http.MethodPost:
			return nativeGroupWire(201, `{"address_group":`+nativeGroupRow+`}`)
		case req.URL.Path == "/neutron/v2.0/address-groups":
			return nativeGroupWire(200, `{"address_groups":[`+nativeGroupRow+`],"address_groups_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/address-groups?marker=x"}]}`)
		case req.URL.Path == "/other/address-groups":
			return nativeGroupWire(200, `{"address_groups":[{"id":"ag-2","addresses":null}]}`)
		}
		return nativeGroupWire(200, `{"address_group":`+nativeGroupRow+`}`)
	})
	created, err := api.Create(ctx, addressgroups.CreateOpts{Name: "web", Addresses: []string{"10.0.0.0/24"}}, addressgroups.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "ag-1" && created.Description == "" && len(created.Addresses) == 2) {
		t.Fatal(created, err)
	}
	// An empty, non-nil address list passes the required check and is sent as [].
	if _, err := api.Create(ctx, addressgroups.CreateOpts{ID: "ag-fixed", Addresses: []string{}}); err != nil {
		t.Fatal(err)
	}
	got, err := api.Get(ctx, "ag-1")
	if err != nil || got.ProjectID != "p" {
		t.Fatal(got, err)
	}
	description := ""
	if _, err := api.Update(ctx, "ag-1", addressgroups.UpdateOpts{Description: &description}, addressgroups.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	var rows []*addressgroups.AddressGroup
	for value, err := range api.List(ctx, addressgroups.WithListOptions(addressgroups.ListOpts{Name: "web", Addresses: []string{"10.0.0.0/24", "10.1.0.0/24"}, Limit: 1}), addressgroups.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, value)
	}
	if !(len(rows) == 2 && rows[0].ID == "ag-1" && rows[1].ID == "ag-2" && rows[1].Addresses == nil) {
		t.Fatal(rows)
	}
	if err := api.Delete(ctx, "ag-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 7 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[4].query)
	want := []nativeGroupCall{
		{http.MethodPost, "/neutron/v2.0/address-groups", "", `{"address_group":{"addresses":["10.0.0.0/24"],"name":"web","x_extension":1}}`},
		// A caller-chosen ID is sent when set.
		{http.MethodPost, "/neutron/v2.0/address-groups", "", `{"address_group":{"addresses":[],"id":"ag-fixed"}}`},
		{http.MethodGet, "/neutron/v2.0/address-groups/ag-1", "", ""},
		{http.MethodPut, "/neutron/v2.0/address-groups/ag-1", "", `{"address_group":{"description":"","x_extension":1}}`},
		{http.MethodGet, "/neutron/v2.0/address-groups", calls[4].query, ""},
		{http.MethodGet, "/other/address-groups", "marker=x", ""},
		{http.MethodDelete, "/neutron/v2.0/address-groups/ag-1", "", ""},
	}
	// The address filter is repeated once per value.
	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"name": {"web"}, "addresses": {"10.0.0.0/24", "10.1.0.0/24"}, "limit": {"1"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeAddressGroupAddressOperations(t *testing.T) {
	ctx := context.Background()
	var calls []nativeGroupCall
	api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeGroupWire(200, `{"address_group":`+nativeGroupRow+`}`)
	})
	added, err := api.AddAddresses(ctx, "ag-1", addressgroups.UpdateAddressesOpts{Addresses: []string{"2001:db8::/64"}}, addressgroups.WithAddAddressesField("x_extension", 1))
	if err != nil || added.ID != "ag-1" || len(added.Addresses) != 2 {
		t.Fatal(added, err)
	}
	if _, err := api.RemoveAddresses(ctx, "ag-1", addressgroups.UpdateAddressesOpts{Addresses: []string{}}); err != nil {
		t.Fatal(err)
	}
	want := []nativeGroupCall{
		// The address body has no envelope, so an extension sits beside addresses.
		{http.MethodPut, "/neutron/v2.0/address-groups/ag-1/add_addresses", "", `{"addresses":["2001:db8::/64"],"x_extension":1}`},
		{http.MethodPut, "/neutron/v2.0/address-groups/ag-1/remove_addresses", "", `{"addresses":[]}`},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
	calls = nil
	for name, check := range map[string]struct {
		operation string
		err       error
	}{
		"nil add": {"AddAddresses", func() error {
			_, err := api.AddAddresses(ctx, "ag-1", addressgroups.UpdateAddressesOpts{})
			return err
		}()},
		"nil remove": {"RemoveAddresses", func() error {
			_, err := api.RemoveAddresses(ctx, "ag-1", addressgroups.UpdateAddressesOpts{})
			return err
		}()},
		"addresses extension": {"AddAddresses", func() error {
			_, err := api.AddAddresses(ctx, "ag-1", addressgroups.UpdateAddressesOpts{Addresses: []string{}}, addressgroups.WithAddAddressesField("addresses", nil))
			return err
		}()},
	} {
		if check.err == nil {
			t.Fatal(name, "accepted")
		}
		nativeGroupOperation(t, check.err, check.operation)
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}

func TestNativeAddressGroupStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	addresses := addressgroups.UpdateAddressesOpts{Addresses: []string{}}
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*addressgroups.API) error
	}{
		{"Create", []int{201, 202}, func(api *addressgroups.API) error {
			_, err := api.Create(ctx, addressgroups.CreateOpts{Addresses: []string{}})
			return err
		}},
		{"Get", []int{200}, func(api *addressgroups.API) error { _, err := api.Get(ctx, "ag-1"); return err }},
		{"Update", []int{200}, func(api *addressgroups.API) error {
			_, err := api.Update(ctx, "ag-1", addressgroups.UpdateOpts{})
			return err
		}},
		{"Delete", []int{202, 204}, func(api *addressgroups.API) error { return api.Delete(ctx, "ag-1") }},
		{"AddAddresses", []int{200}, func(api *addressgroups.API) error { _, err := api.AddAddresses(ctx, "ag-1", addresses); return err }},
		{"RemoveAddresses", []int{200}, func(api *addressgroups.API) error {
			_, err := api.RemoveAddresses(ctx, "ag-1", addresses)
			return err
		}},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeGroupCall
				api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response { return nativeGroupWire(code, `{"address_group":{}}`) })
				err := call.call(api)
				nativeGroupOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("list pager status, empty page and bodyless 204", func(t *testing.T) {
		for _, tc := range []struct {
			code int
			body string
		}{{404, `{}`}, {200, `{"address_groups":[]}`}, {204, ""}} {
			var calls []nativeGroupCall
			api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response { return nativeGroupWire(tc.code, tc.body) })
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
		var calls []nativeGroupCall
		api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response { return nativeGroupWire(201, `{}`) })
		for operation, err := range map[string]error{
			"Create": func() error { _, err := api.Create(ctx, addressgroups.CreateOpts{Name: "web"}); return err }(),
			"Update": func() error {
				_, err := api.Update(ctx, "ag-1", addressgroups.UpdateOpts{}, addressgroups.WithUpdateField("name", "x"))
				return err
			}(),
		} {
			if err == nil {
				t.Fatal(operation, "accepted")
			}
			nativeGroupOperation(t, err, operation)
		}
		for _, err := range api.List(ctx, nil) {
			nativeGroupOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
