package extraroutes_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/extraroutes"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeRouteTransport func(*http.Request) (*http.Response, error)

func (transport nativeRouteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return transport(req)
}

type nativeRouteCall struct{ method, path, body string }

func nativeRouteAPI(t *testing.T, code int, calls *[]nativeRouteCall) *extraroutes.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeRouteTransport(func(req *http.Request) (*http.Response, error) {
		raw, _ := io.ReadAll(req.Body)
		*calls = append(*calls, nativeRouteCall{req.Method, req.URL.Path, string(raw)})
		body := `{"router":{"id":"r1","routes":[{"destination":"10.1.0.0/24","nexthop":"10.0.0.9"}]}}`
		return &http.Response{StatusCode: code, Request: req, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return extraroutes.New(client)
}

func nativeRouteOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "extraroutes" {
		t.Fatal("generated extraroutes context", err, wrapped)
	}
}

func TestNativeExtraRoutesAddRemoveBodiesAndStatuses(t *testing.T) {
	ctx := context.Background()
	var calls []nativeRouteCall
	api := nativeRouteAPI(t, 200, &calls)
	routes := &[]routers.Route{{DestinationCIDR: "10.1.0.0/24", NextHop: "10.0.0.9"}}
	added, err := api.Add(ctx, "r1", extraroutes.Opts{Routes: routes}, extraroutes.WithAddField("x_extension", 1))
	if err != nil || added.ID != "r1" || added.Routes[0].NextHop != "10.0.0.9" {
		t.Fatal(added, err)
	}
	if _, err := api.Remove(ctx, "r1", extraroutes.Opts{Routes: routes}); err != nil {
		t.Fatal(err)
	}
	// A nil route list sends an empty router object rather than routes: null.
	if _, err := api.Add(ctx, "r1", extraroutes.Opts{}); err != nil {
		t.Fatal(err)
	}
	want := []nativeRouteCall{
		{http.MethodPut, "/neutron/v2.0/routers/r1/add_extraroutes", `{"router":{"routes":[{"destination":"10.1.0.0/24","nexthop":"10.0.0.9"}],"x_extension":1}}`},
		{http.MethodPut, "/neutron/v2.0/routers/r1/remove_extraroutes", `{"router":{"routes":[{"destination":"10.1.0.0/24","nexthop":"10.0.0.9"}]}}`},
		{http.MethodPut, "/neutron/v2.0/routers/r1/add_extraroutes", `{"router":{}}`},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
	for _, code := range []int{201, 202, 204, 404} {
		for _, operation := range []string{"Add", "Remove"} {
			var calls []nativeRouteCall
			api := nativeRouteAPI(t, code, &calls)
			var err error
			if operation == "Add" {
				_, err = api.Add(ctx, "r1", extraroutes.Opts{Routes: routes})
			} else {
				_, err = api.Remove(ctx, "r1", extraroutes.Opts{Routes: routes})
			}
			nativeRouteOperation(t, err, operation)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || len(calls) != 1 {
				t.Fatal(operation, err)
			}
		}
	}
	calls = nil
	api = nativeRouteAPI(t, 200, &calls)
	for operation, err := range map[string]error{
		"Add": func() error {
			_, err := api.Add(ctx, "r1", extraroutes.Opts{}, extraroutes.WithAddField("routes", nil))
			return err
		}(),
		"Remove": func() error { _, err := api.Remove(ctx, "r1", extraroutes.Opts{}, nil); return err }(),
	} {
		if err == nil {
			t.Fatal(operation, "accepted")
		}
		nativeRouteOperation(t, err, operation)
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}
