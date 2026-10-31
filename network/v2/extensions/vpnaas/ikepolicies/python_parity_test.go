package ikepolicies_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonIKEPolicyCall records the wire request openstacksdk would also send.
type pythonIKEPolicyCall struct{ method, path, query, body string }

func pythonIKEPolicyWire(code int, body string) *http.Response {
	return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}})
}

func pythonIKEPolicyAPI(t *testing.T, reply func(*http.Request) *http.Response) (*ikepolicies.API, *[]pythonIKEPolicyCall, *testcloud.Cloud) {
	t.Helper()
	calls := &[]pythonIKEPolicyCall{}
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonIKEPolicyCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return ikepolicies.New(client), calls, cloud
}

const (
	pythonIKEPolicyBase = "/neutron/v2.0/vpn/ikepolicies"
	pythonIKEPolicyRow  = `{"id":"ike-1","name":"ike","description":"d","auth_algorithm":"sha256","encryption_algorithm":"aes-256","pfs":"group14","ike_version":"v2","phase1_negotiation_mode":"main","lifetime":{"units":"seconds","value":7200},"project_id":"p","tenant_id":"p"}`
)

func TestPythonIKEPolicyProxyRequests(t *testing.T) {
	ctx := context.Background()
	deletes := []int{404, 404, 409, 204, 404}
	api, calls, _ := pythonIKEPolicyAPI(t, func(req *http.Request) *http.Response {
		switch req.Method {
		case http.MethodDelete:
			code := deletes[0]
			deletes = deletes[1:]
			return pythonIKEPolicyWire(code, `{"NeutronError":{"type":"x"}}`)
		case http.MethodPost:
			return pythonIKEPolicyWire(201, `{"ikepolicy":`+pythonIKEPolicyRow+`}`)
		}
		return pythonIKEPolicyWire(200, `{"ikepolicy":`+pythonIKEPolicyRow+`}`)
	})
	// create_vpn_ike_policy(name="ike", auth_algorithm="sha256", pfs="group14",
	// ike_version="v2", phase1_negotiation_mode="main", lifetime={"units": "seconds", "value": 7200})
	created, err := api.Create(ctx, ikepolicies.CreateOpts{
		Name: "ike", AuthAlgorithm: ikepolicies.AuthAlgorithmSHA256, PFS: ikepolicies.PFSGroup14, IKEVersion: ikepolicies.IKEVersionv2,
		Phase1NegotiationMode: "main", Lifetime: &ikepolicies.LifetimeCreateOpts{Units: ikepolicies.UnitSeconds, Value: 7200},
	})
	if err != nil || created.ID != "ike-1" || created.EncryptionAlgorithm != "aes-256" || created.Phase1NegotiationMode != "main" || created.Lifetime.Value != 7200 || created.Lifetime.Units != "seconds" {
		t.Fatal(created, err)
	}
	// get_vpn_ike_policy("ike-1")
	if got, err := api.Get(ctx, "ike-1"); err != nil || got.IKEVersion != "v2" || got.PFS != "group14" {
		t.Fatal(got, err)
	}
	// update_vpn_ike_policy("ike-1", name="", phase1_negotiation_mode="aggressive"): the typed
	// Phase1NegotiationMode sends phase_1_negotiation_mode, so the Python key needs WithUpdateField.
	empty := ""
	if _, err := api.Update(ctx, "ike-1", ikepolicies.UpdateOpts{Name: &empty}, ikepolicies.WithUpdateField("phase1_negotiation_mode", "aggressive")); err != nil {
		t.Fatal(err)
	}
	if _, err := api.Update(ctx, "ike-1", ikepolicies.UpdateOpts{Phase1NegotiationMode: "aggressive"}); err != nil {
		t.Fatal(err)
	}
	// delete_vpn_ike_policy("ike-1") ignores 404; ignore_missing=False reports it.
	if err := api.Remove(ctx, resource.ID("ike-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("ike-1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("ike-1")); !gophercloud.ResponseCodeIs(err, http.StatusConflict) {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("ike-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, "ike-1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	want := []pythonIKEPolicyCall{
		{http.MethodPost, pythonIKEPolicyBase, "", `{"ikepolicy":{"auth_algorithm":"sha256","ike_version":"v2","lifetime":{"units":"seconds","value":7200},"name":"ike","pfs":"group14","phase1_negotiation_mode":"main"}}`},
		{http.MethodGet, pythonIKEPolicyBase + "/ike-1", "", ""},
		{http.MethodPut, pythonIKEPolicyBase + "/ike-1", "", `{"ikepolicy":{"name":"","phase1_negotiation_mode":"aggressive"}}`},
		{http.MethodPut, pythonIKEPolicyBase + "/ike-1", "", `{"ikepolicy":{"phase_1_negotiation_mode":"aggressive"}}`},
	}
	for range 5 {
		want = append(want, pythonIKEPolicyCall{http.MethodDelete, pythonIKEPolicyBase + "/ike-1", "", ""})
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonIKEPolicyListQueryAndLinks(t *testing.T) {
	ctx := context.Background()
	var cloud *testcloud.Cloud
	api, calls, cloud := pythonIKEPolicyAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("marker") == "" {
			return pythonIKEPolicyWire(200, `{"ikepolicies":[`+pythonIKEPolicyRow+`],"ikepolicies_links":[{"rel":"next","href":"`+cloud.Server.URL+pythonIKEPolicyBase+`?limit=1&marker=ike-1&phase1_negotiation_mode=main&pfs=group14"}]}`)
		}
		return pythonIKEPolicyWire(200, `{"ikepolicies":[]}`)
	})
	// vpn_ike_policies(pfs="group14", phase1_negotiation_mode="main", limit=1); the typed
	// ListOpts field would send phase_1_negotiation_mode, so the Python key is a raw query.
	var ids []string
	for value, err := range api.List(ctx, ikepolicies.WithListOptions(ikepolicies.ListOpts{PFS: "group14"}), ikepolicies.WithListQuery("phase1_negotiation_mode", "main"), ikepolicies.WithListQuery("limit", "1")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID+"/"+value.Phase1NegotiationMode)
	}
	if !reflect.DeepEqual(ids, []string{"ike-1/main"}) || len(*calls) != 2 {
		t.Fatal(ids, *calls)
	}
	first, _ := url.ParseQuery((*calls)[0].query)
	next, _ := url.ParseQuery((*calls)[1].query)
	if !reflect.DeepEqual(first, url.Values{"pfs": {"group14"}, "phase1_negotiation_mode": {"main"}, "limit": {"1"}}) ||
		!reflect.DeepEqual(next, url.Values{"pfs": {"group14"}, "phase1_negotiation_mode": {"main"}, "limit": {"1"}, "marker": {"ike-1"}}) {
		t.Fatal(first, next)
	}
}

func TestPythonIKEPolicyFindChainAndGetWithoutFields(t *testing.T) {
	ctx := context.Background()
	api, calls, _ := pythonIKEPolicyAPI(t, func(req *http.Request) *http.Response {
		switch {
		case req.URL.Path == pythonIKEPolicyBase+"/ike-1":
			return pythonIKEPolicyWire(200, `{"ikepolicy":`+pythonIKEPolicyRow+`}`)
		case req.URL.Path == pythonIKEPolicyBase && req.URL.Query().Get("name") == "ike":
			return pythonIKEPolicyWire(200, `{"ikepolicies":[`+pythonIKEPolicyRow+`]}`)
		case req.URL.Path == pythonIKEPolicyBase:
			return pythonIKEPolicyWire(200, `{"ikepolicies":[]}`)
		}
		return pythonIKEPolicyWire(404, `{"NeutronError":{"type":"IKEPolicyNotFound"}}`)
	})
	// find_vpn_ike_policy("ike"): GET by ID, then list filtered by name.
	if got, err := api.Find(ctx, resource.ID("ike"), resource.WithIgnoreMissing()); err != nil || got != nil {
		t.Fatal(got, err)
	}
	if got, err := api.Find(ctx, resource.Name("ike"), resource.WithIgnoreMissing()); err != nil || got.ID != "ike-1" {
		t.Fatal(got, err)
	}
	if got, err := api.Find(ctx, resource.ID("ike-1"), resource.WithIgnoreMissing()); err != nil || got.Name != "ike" {
		t.Fatal(got, err)
	}
	if _, err := api.Find(ctx, resource.ID("none")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := api.Find(ctx, resource.Name("none")); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	// get_vpn_ike_policy(..., fields=[...]) has no Go counterpart: Get never sends a query.
	if _, err := api.Get(ctx, "ike-1"); err != nil {
		t.Fatal(err)
	}
	want := []pythonIKEPolicyCall{
		{http.MethodGet, pythonIKEPolicyBase + "/ike", "", ""},
		{http.MethodGet, pythonIKEPolicyBase, "name=ike", ""},
		{http.MethodGet, pythonIKEPolicyBase + "/ike-1", "", ""},
		{http.MethodGet, pythonIKEPolicyBase + "/none", "", ""},
		{http.MethodGet, pythonIKEPolicyBase, "name=none", ""},
		{http.MethodGet, pythonIKEPolicyBase + "/ike-1", "", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}
