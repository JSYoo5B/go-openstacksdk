package ikepolicies_test

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
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeVPNTransport func(*http.Request) (*http.Response, error)

func (transport nativeVPNTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeVPNWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeVPNCall struct{ method, path, query, body string }

func nativeVPNAPI(t *testing.T, calls *[]nativeVPNCall, reply func(*http.Request) *http.Response) (*ikepolicies.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeVPNTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeVPNCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return ikepolicies.New(client), cloud
}

func nativeVPNOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "ikepolicies" {
		t.Fatal("generated ikepolicies context", err, wrapped)
	}
}

func nativeVPNStatuses(t *testing.T, name string, accepted []int, envelope string, call func(*ikepolicies.API) error) {
	t.Helper()
	for _, code := range []int{200, 201, 202, 204, 404} {
		if slices.Contains(accepted, code) {
			continue
		}
		t.Run(fmt.Sprintf("%s/%d", name, code), func(t *testing.T) {
			var calls []nativeVPNCall
			api, _ := nativeVPNAPI(t, &calls, func(*http.Request) *http.Response { return nativeVPNWire(code, `{"`+envelope+`":{}}`) })
			err := call(api)
			nativeVPNOperation(t, err, name)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, accepted) || len(calls) != 1 {
				t.Fatal(err, native)
			}
		})
	}
}

func nativeVPNListStatuses(t *testing.T, collection string, list func(*ikepolicies.API) []error) {
	t.Helper()
	for _, tc := range []struct {
		code int
		body string
	}{{404, `{}`}, {200, `{"` + collection + `":[]}`}, {204, ""}} {
		var calls []nativeVPNCall
		api, _ := nativeVPNAPI(t, &calls, func(*http.Request) *http.Response { return nativeVPNWire(tc.code, tc.body) })
		errs := list(api)
		var native gophercloud.ErrUnexpectedResponseCode
		switch {
		case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
		case tc.code == 200 && len(errs) == 0:
		case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
		default:
			t.Fatal(tc.code, errs)
		}
	}
}

const nativeVPNRow = `{"id":"x-1","name":"ike","auth_algorithm":"sha256","encryption_algorithm":"aes-256","pfs":"group14","ike_version":"v2","phase1_negotiation_mode":"main","lifetime":{"units":"seconds","value":3600}}`

func TestNativeVPNPolicyRoutesBodiesPagingAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeVPNCall
	var cloud *testcloud.Cloud
	api, cloud := nativeVPNAPI(t, &calls, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return nativeVPNWire(204, "")
		case req.Method == http.MethodPost:
			return nativeVPNWire(201, `{"ikepolicy":`+nativeVPNRow+`}`)
		case req.URL.Path == "/neutron/v2.0/vpn/ikepolicies":
			return nativeVPNWire(200, `{"ikepolicies":[`+nativeVPNRow+`],"ikepolicies_links":[{"rel":"next","href":"`+cloud.Server.URL+`/other/page?marker=x"}]}`)
		case req.URL.Path == "/other/page":
			return nativeVPNWire(200, `{"ikepolicies":[{"id":"x-2","name":null}]}`)
		}
		return nativeVPNWire(200, `{"ikepolicy":`+nativeVPNRow+`}`)
	})
	created, err := api.Create(ctx, ikepolicies.CreateOpts{Name: "ike", AuthAlgorithm: ikepolicies.AuthAlgorithmSHA256, IKEVersion: ikepolicies.IKEVersionv2, Phase1NegotiationMode: ikepolicies.Phase1NegotiationModeMain, Lifetime: &ikepolicies.LifetimeCreateOpts{Units: ikepolicies.UnitSeconds, Value: 3600}}, ikepolicies.WithCreateField("x_extension", 1))
	if err != nil || !(created.ID == "x-1" && created.Lifetime.Value == 3600 && created.Phase1NegotiationMode == "main") {
		t.Fatal(created, err)
	}
	// An empty lifetime object is still sent.
	if _, err := api.Create(ctx, ikepolicies.CreateOpts{Lifetime: &ikepolicies.LifetimeCreateOpts{}}); err != nil {
		t.Fatal(err)
	}
	if got, err := api.Get(ctx, "x-1"); err != nil || got.ID != "x-1" {
		t.Fatal(got, err)
	}
	empty := ""
	if _, err := api.Update(ctx, "x-1", ikepolicies.UpdateOpts{Name: &empty, Phase1NegotiationMode: ikepolicies.Phase1NegotiationModeMain}, ikepolicies.WithUpdateField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for value, err := range api.List(ctx, ikepolicies.WithListOptions(ikepolicies.ListOpts{Name: "ike", Phase1NegotiationMode: "main"}), ikepolicies.WithListQuery("extra", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID+"/"+value.Name)
	}
	if !reflect.DeepEqual(ids, []string{"x-1/" + created.Name, "x-2/"}) {
		t.Fatal(ids)
	}
	if err := api.Delete(ctx, "x-1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 7 {
		t.Fatalf("%+v", calls)
	}
	query, _ := url.ParseQuery(calls[4].query)
	base := "/neutron/v2.0/vpn/ikepolicies"
	want := []nativeVPNCall{
		{http.MethodPost, base, "", `{"ikepolicy":{"auth_algorithm":"sha256","ike_version":"v2","lifetime":{"units":"seconds","value":3600},"name":"ike","phase1_negotiation_mode":"main","x_extension":1}}`},
		{http.MethodPost, base, "", `{"ikepolicy":{"lifetime":{}}}`},
		{http.MethodGet, base + "/x-1", "", ""},
		// Update spells the negotiation mode key phase_1_negotiation_mode, unlike Create.
		{http.MethodPut, base + "/x-1", "", `{"ikepolicy":{"name":"","phase_1_negotiation_mode":"main","x_extension":1}}`},
		{http.MethodGet, base, calls[4].query, ""},
		{http.MethodGet, "/other/page", "marker=x", ""},
		{http.MethodDelete, base + "/x-1", "", ""},
	}

	if !reflect.DeepEqual(calls, want) || !reflect.DeepEqual(query, url.Values{"name": {"ike"}, "phase_1_negotiation_mode": {"main"}, "extra": {"1"}}) {
		t.Fatalf("%+v %v", calls, query)
	}
}

func TestNativeVPNPolicyStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	nativeVPNStatuses(t, "Create", []int{201, 202}, "ikepolicy", func(api *ikepolicies.API) error {
		_, err := api.Create(ctx, ikepolicies.CreateOpts{Lifetime: &ikepolicies.LifetimeCreateOpts{}})
		return err
	})
	nativeVPNStatuses(t, "Get", []int{200}, "ikepolicy", func(api *ikepolicies.API) error { _, err := api.Get(ctx, "x-1"); return err })
	nativeVPNStatuses(t, "Update", []int{200}, "ikepolicy", func(api *ikepolicies.API) error {
		_, err := api.Update(ctx, "x-1", ikepolicies.UpdateOpts{})
		return err
	})
	nativeVPNStatuses(t, "Delete", []int{202, 204}, "ikepolicy", func(api *ikepolicies.API) error { return api.Delete(ctx, "x-1") })
	nativeVPNListStatuses(t, "ikepolicies", func(api *ikepolicies.API) (errs []error) {
		for _, err := range api.List(ctx) {
			errs = append(errs, err)
		}
		return errs
	})
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeVPNCall
		api, _ := nativeVPNAPI(t, &calls, func(*http.Request) *http.Response { return nativeVPNWire(201, `{}`) })
		for name, err := range map[string]error{
			"core extension": func() error {
				_, err := api.Create(ctx, ikepolicies.CreateOpts{Lifetime: &ikepolicies.LifetimeCreateOpts{}}, ikepolicies.WithCreateField("lifetime", nil))
				return err
			}(),
			"nil option": func() error { _, err := api.Update(ctx, "x-1", ikepolicies.UpdateOpts{}, nil); return err }(),
		} {
			if err == nil {
				t.Fatal(name, "accepted")
			}
			var wrapped *resource.OperationError
			if !errors.As(err, &wrapped) {
				t.Fatal(name, err)
			}
		}
		for _, err := range api.List(ctx, nil) {
			nativeVPNOperation(t, err, "List")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
