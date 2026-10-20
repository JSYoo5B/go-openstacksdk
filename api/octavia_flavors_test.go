package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/flavorprofiles"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/flavors"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const octaviaFlavorBody = `{"flavor":{"id":"fl-1","name":"small","description":"d","enabled":false,"flavor_profile_id":"fp-1","vendor_extra":1}}`
const octaviaFlavorProfileBody = `{"flavorprofile":{"id":"fp-1","name":"prof","provider_name":"amphora","flavor_data":"{\"compute_flavor\":\"c1\"}","vendor_extra":1}}`

func octaviaFlavorRequest(t *testing.T, r *http.Request, envelope string) map[string]string {
	t.Helper()
	var body map[string]map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("request body: %v", err)
	}
	if len(body) != 1 || body[envelope] == nil {
		t.Errorf("envelope %q missing: %v", envelope, body)
	}
	got := map[string]string{}
	for key, value := range body[envelope] {
		got[key] = string(value)
	}
	return got
}

func octaviaFlavorUnexpected(t *testing.T, err error, status int) {
	t.Helper()
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &unexpected) || unexpected.Actual != status {
		t.Fatalf("want unexpected status %d, got %v", status, err)
	}
}

func octaviaFlavorBool(value bool) *bool { return &value }

func newOctaviaFlavors(t *testing.T, cloud *testcloud.Cloud) *flavors.API {
	t.Helper()
	return flavors.New(cloud.Client("load-balancer", "/v2"))
}

func newOctaviaFlavorProfiles(t *testing.T, cloud *testcloud.Cloud) *flavorprofiles.API {
	t.Helper()
	return flavorprofiles.New(cloud.Client("load-balancer", "/v2"))
}

func TestOctaviaFlavorCreateEnvelopeExtensionsAndFalseLimitation(t *testing.T) {
	cloud := testcloud.New(t)
	var bodies []map[string]string
	cloud.Mux.HandleFunc("POST /v2/lbaas/flavors", func(w http.ResponseWriter, r *http.Request) {
		bodies = append(bodies, octaviaFlavorRequest(t, r, "flavor"))
		testcloud.JSON(w, 201, octaviaFlavorBody)
	})
	api := newOctaviaFlavors(t, cloud)
	ctx := context.Background()
	value, err := api.Create(ctx, flavors.CreateOpts{Name: "small", FlavorProfileId: "fp-1", Description: "d", Enabled: true}, flavors.WithCreateField("vendor", map[string]bool{"x": false}))
	if err != nil || value == nil || value.ID != "fl-1" || value.Enabled || value.FlavorProfileId != "fp-1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	// Enabled=false is indistinguishable from unset in the native typed options and is omitted.
	if _, err := api.Create(ctx, flavors.CreateOpts{Name: "small", FlavorProfileId: "fp-1"}); err != nil {
		t.Fatal(err)
	}
	want := []map[string]string{
		{"name": `"small"`, "flavor_profile_id": `"fp-1"`, "description": `"d"`, "enabled": `true`, "vendor": `{"x":false}`},
		{"name": `"small"`, "flavor_profile_id": `"fp-1"`},
	}
	if !reflect.DeepEqual(bodies, want) {
		t.Fatalf("bodies=%v", bodies)
	}
}

func TestOctaviaFlavorCreateAcceptsOnly201Or202(t *testing.T) {
	for _, status := range []int{201, 202, 200} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("POST /v2/lbaas/flavors", func(w http.ResponseWriter, r *http.Request) {
			testcloud.JSON(w, status, octaviaFlavorBody)
		})
		value, err := newOctaviaFlavors(t, cloud).Create(context.Background(), flavors.CreateOpts{Name: "n", FlavorProfileId: "p"})
		if (status != 200) != (err == nil) || (err == nil && value.ID != "fl-1") {
			t.Fatalf("status %d value=%+v error=%v", status, value, err)
		}
		if status == 200 {
			octaviaFlavorUnexpected(t, err, 200)
		}
	}
}

func TestOctaviaFlavorRejectsInvalidInputsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	ctx := context.Background()
	fl, fp := newOctaviaFlavors(t, cloud), newOctaviaFlavorProfiles(t, cloud)
	validFlavor := flavors.CreateOpts{Name: "n", FlavorProfileId: "p"}
	validProfile := flavorprofiles.CreateOpts{Name: "n", ProviderName: "amphora", FlavorData: "{}"}
	errs := map[string]error{}
	_, errs["flavor protected create"] = fl.Create(ctx, validFlavor, flavors.WithCreateField("name", "x"))
	_, errs["flavor protected update"] = fl.Update(ctx, "fl-1", flavors.UpdateOpts{}, flavors.WithUpdateField("description", "x"))
	_, errs["flavor false enabled extension collides"] = fl.Update(ctx, "fl-1", flavors.UpdateOpts{}, flavors.WithUpdateField("enabled", false))
	_, errs["flavor nil create"] = fl.Create(ctx, validFlavor, nil)
	_, errs["flavor nil update"] = fl.Update(ctx, "fl-1", flavors.UpdateOpts{}, nil)
	_, errs["profile protected create"] = fp.Create(ctx, validProfile, flavorprofiles.WithCreateField("provider_name", "x"))
	_, errs["profile protected update"] = fp.Update(ctx, "fp-1", flavorprofiles.UpdateOpts{}, flavorprofiles.WithUpdateField("flavor_data", "x"))
	_, errs["profile nil create"] = fp.Create(ctx, validProfile, nil)
	_, errs["profile nil update"] = fp.Update(ctx, "fp-1", flavorprofiles.UpdateOpts{}, nil)
	for name, err := range errs {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for _, err := range fl.List(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("flavor nil list: %v", err)
		}
	}
	for _, err := range fp.List(ctx, nil) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("profile nil list: %v", err)
		}
	}
	if _, err := fl.Create(ctx, flavors.CreateOpts{Name: "n"}); err == nil {
		t.Error("flavor create without flavor_profile_id must fail locally")
	}
	if _, err := fp.Create(ctx, flavorprofiles.CreateOpts{Name: "n"}); err == nil {
		t.Error("profile create without provider_name and flavor_data must fail locally")
	}
}

func TestOctaviaFlavorUpdateSendsEmptyStringsAcceptsOnly200AndCannotSendFalse(t *testing.T) {
	for _, status := range []int{200, 202, 201} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("PUT /v2/lbaas/flavors/fl-1", func(w http.ResponseWriter, r *http.Request) {
			// Typed string fields use omitempty, so empty strings and Enabled=false are omitted.
			if got := octaviaFlavorRequest(t, r, "flavor"); !reflect.DeepEqual(got, map[string]string{"name": `"n2"`, "vendor": `null`}) {
				t.Errorf("body=%v", got)
			}
			testcloud.JSON(w, status, octaviaFlavorBody)
		})
		value, err := newOctaviaFlavors(t, cloud).Update(context.Background(), "fl-1", flavors.UpdateOpts{Name: "n2", Description: "", Enabled: false}, flavors.WithUpdateField("vendor", nil))
		if (status == 200) != (err == nil) || (err == nil && value.ID != "fl-1") {
			t.Fatalf("status %d value=%+v error=%v", status, value, err)
		}
		if status != 200 {
			octaviaFlavorUnexpected(t, err, status)
		}
	}
}

func TestOctaviaFlavorGetDeleteAndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/lbaas/flavors/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "missing" {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		testcloud.JSON(w, 200, octaviaFlavorBody)
	})
	cloud.Mux.HandleFunc("DELETE /v2/lbaas/flavors/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("id") {
		case "missing":
			testcloud.JSON(w, 404, `{}`)
		case "ok200":
			testcloud.JSON(w, 200, `{}`)
		case "ok202":
			w.WriteHeader(202)
		default:
			w.WriteHeader(204)
		}
	})
	api := newOctaviaFlavors(t, cloud)
	ctx := context.Background()
	value, err := api.Get(ctx, "fl-1")
	if err != nil || value.Name != "small" || value.Description != "d" || value.Enabled {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	_, err = api.Get(ctx, "missing")
	octaviaFlavorUnexpected(t, err, 404)
	for _, id := range []string{"fl-1", "ok202"} {
		if err := api.Delete(ctx, id); err != nil {
			t.Fatalf("delete %s: %v", id, err)
		}
	}
	octaviaFlavorUnexpected(t, api.Delete(ctx, "ok200"), 200)
	octaviaFlavorUnexpected(t, api.Delete(ctx, "missing"), 404)
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
}

func TestOctaviaFlavorListFiltersFollowLinksAndPreserveFalse(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v2/lbaas/flavors", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Get("marker") == "fl-1" {
			testcloud.JSON(w, 200, `{"flavors":[{"id":"fl-2","name":"two"}],"flavors_links":[]}`)
			return
		}
		next := cloud.Server.URL + "/v2/lbaas/flavors?marker=fl-1"
		testcloud.JSON(w, 200, `{"flavors":[{"id":"fl-1","name":"small"}],"flavors_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	api := newOctaviaFlavors(t, cloud)
	var ids []string
	for value, err := range api.List(context.Background(),
		flavors.WithListOptions(flavors.ListOpts{FlavorProfileID: "fp-1", Enabled: octaviaFlavorBool(false), Fields: []string{"id", "name"}}),
		flavors.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"fl-1", "fl-2"}) || len(requests) != 2 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	if first := requests[0]; first.Get("enabled") != "false" || first.Get("flavor_profile_id") != "fp-1" || !reflect.DeepEqual(first["fields"], []string{"id", "name"}) || first.Get("vendor") != "x" {
		t.Fatalf("first query=%v", first)
	}
	if requests[1].Get("marker") != "fl-1" {
		t.Fatalf("second query=%v", requests[1])
	}
}

func TestOctaviaFlavorCommonLookupFindsExactName(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/lbaas/flavors/small", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/flavors", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"flavors":[{"id":"fl-1","name":"small"},{"id":"fl-2","name":"small-2"}]}`)
	})
	value, err := newOctaviaFlavors(t, cloud).Find(context.Background(), resource.Name("small"))
	if err != nil || value == nil || value.ID != "fl-1" || len(queries) == 0 || queries[0].Get("name") != "small" {
		t.Fatalf("value=%+v error=%v queries=%v", value, err, queries)
	}
}

func TestOctaviaFlavorProfileCreateEnvelopeAndExtensionFields(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{201, true}, {202, true}, {200, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("POST /v2/lbaas/flavorprofiles", func(w http.ResponseWriter, r *http.Request) {
			want := map[string]string{"name": `"prof"`, "provider_name": `"amphora"`, "flavor_data": `"{\"compute_flavor\":\"c1\"}"`, "vendor": `false`}
			if got := octaviaFlavorRequest(t, r, "flavorprofile"); !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			testcloud.JSON(w, tc.status, octaviaFlavorProfileBody)
		})
		value, err := newOctaviaFlavorProfiles(t, cloud).Create(context.Background(),
			flavorprofiles.CreateOpts{Name: "prof", ProviderName: "amphora", FlavorData: `{"compute_flavor":"c1"}`},
			flavorprofiles.WithCreateField("vendor", false))
		if tc.success != (err == nil) || (tc.success && (value.ID != "fp-1" || value.ProviderName != "amphora" || value.FlavorData != `{"compute_flavor":"c1"}`)) {
			t.Fatalf("status %d value=%+v error=%v", tc.status, value, err)
		}
		if !tc.success {
			octaviaFlavorUnexpected(t, err, tc.status)
		}
	}
}

func TestOctaviaFlavorProfileUpdateAccepts200Or202AndOmitsEmptyStrings(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, true}, {201, false}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("PUT /v2/lbaas/flavorprofiles/fp-1", func(w http.ResponseWriter, r *http.Request) {
			if got := octaviaFlavorRequest(t, r, "flavorprofile"); !reflect.DeepEqual(got, map[string]string{"name": `"p2"`, "vendor": `null`}) {
				t.Errorf("body=%v", got)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, octaviaFlavorProfileBody)
		})
		value, err := newOctaviaFlavorProfiles(t, cloud).Update(context.Background(), "fp-1", flavorprofiles.UpdateOpts{Name: "p2"}, flavorprofiles.WithUpdateField("vendor", nil))
		if tc.success != (err == nil) || (tc.success && value.ID != "fp-1") {
			t.Fatalf("status %d value=%+v error=%v", tc.status, value, err)
		}
		if !tc.success {
			octaviaFlavorUnexpected(t, err, tc.status)
		}
	}
}

func TestOctaviaFlavorProfileGetDeleteListAndFind(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/lbaas/flavorprofiles/prof", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/flavorprofiles/{id}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "missing" {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		testcloud.JSON(w, 200, octaviaFlavorProfileBody)
	})
	cloud.Mux.HandleFunc("DELETE /v2/lbaas/flavorprofiles/{id}", func(w http.ResponseWriter, r *http.Request) {
		switch r.PathValue("id") {
		case "missing":
			testcloud.JSON(w, 404, `{}`)
		case "ok200":
			testcloud.JSON(w, 200, `{}`)
		case "ok202":
			w.WriteHeader(202)
		default:
			w.WriteHeader(204)
		}
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/flavorprofiles", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		if r.URL.Query().Get("marker") == "fp-1" {
			testcloud.JSON(w, 200, `{"flavorprofiles":[{"id":"fp-2","name":"two"}],"flavorprofiles_links":[]}`)
			return
		}
		next := cloud.Server.URL + "/v2/lbaas/flavorprofiles?marker=fp-1"
		testcloud.JSON(w, 200, `{"flavorprofiles":[{"id":"fp-1","name":"prof"}],"flavorprofiles_links":[{"rel":"next","href":"`+next+`"}]}`)
	})
	api := newOctaviaFlavorProfiles(t, cloud)
	ctx := context.Background()
	value, err := api.Get(ctx, "fp-1")
	if err != nil || value.Name != "prof" || value.ProviderName != "amphora" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	_, err = api.Get(ctx, "missing")
	octaviaFlavorUnexpected(t, err, 404)
	for _, id := range []string{"fp-1", "ok202"} {
		if err := api.Delete(ctx, id); err != nil {
			t.Fatalf("delete %s: %v", id, err)
		}
	}
	octaviaFlavorUnexpected(t, api.Delete(ctx, "ok200"), 200)
	octaviaFlavorUnexpected(t, api.Delete(ctx, "missing"), 404)
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
	var ids []string
	for value, err := range api.List(ctx, flavorprofiles.WithListOptions(flavorprofiles.ListOpts{ProviderName: "amphora", Fields: []string{"id", "name"}}), flavorprofiles.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"fp-1", "fp-2"}) || len(queries) != 2 {
		t.Fatalf("ids=%v queries=%v", ids, queries)
	}
	if first := queries[0]; first.Get("provider_name") != "amphora" || !reflect.DeepEqual(first["fields"], []string{"id", "name"}) || first.Get("vendor") != "x" || queries[1].Get("marker") != "fp-1" {
		t.Fatalf("queries=%v", queries)
	}
	queries = nil
	if found, err := api.Find(ctx, resource.Name("prof")); err != nil || found.ID != "fp-1" || queries[0].Get("name") != "prof" {
		t.Fatalf("found=%+v error=%v queries=%v", found, err, queries)
	}
}

func TestOctaviaFlavorCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	fl, fp := newOctaviaFlavors(t, cloud), newOctaviaFlavorProfiles(t, cloud)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	validFlavor := flavors.CreateOpts{Name: "n", FlavorProfileId: "p"}
	validProfile := flavorprofiles.CreateOpts{Name: "n", ProviderName: "amphora", FlavorData: "{}"}
	errs := map[string]error{}
	_, errs["flavor get"] = fl.Get(canceled, "fl-1")
	_, errs["flavor create"] = fl.Create(canceled, validFlavor)
	_, errs["flavor update"] = fl.Update(canceled, "fl-1", flavors.UpdateOpts{})
	errs["flavor delete"] = fl.Delete(canceled, "fl-1")
	_, errs["profile get"] = fp.Get(canceled, "fp-1")
	_, errs["profile create"] = fp.Create(canceled, validProfile)
	_, errs["profile update"] = fp.Update(canceled, "fp-1", flavorprofiles.UpdateOpts{})
	errs["profile delete"] = fp.Delete(canceled, "fp-1")
	for _, err := range fl.List(canceled) {
		errs["flavor list"] = err
	}
	for _, err := range fp.List(canceled) {
		errs["profile list"] = err
	}
	for name, err := range errs {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("pre-canceled %s: %v", name, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := fl.Create(deadline, validFlavor); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("flavor in-flight deadline: %v", err)
	}
	if _, err := fp.Create(deadline, validProfile); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("profile in-flight deadline: %v", err)
	}
}
