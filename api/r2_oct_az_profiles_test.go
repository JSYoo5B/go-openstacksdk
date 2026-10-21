package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/availabilityzoneprofiles"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const r2OctAZProfileJSON = `{"availability_zone_profile":{"id":"11111111-1111-1111-1111-111111111111","name":"p","provider_name":"amphora","availability_zone_data":"{\"compute_zone\": \"nova\"}"}}`

func r2OctAZProfilesAPI(t *testing.T, cloud *testcloud.Cloud) *availabilityzoneprofiles.API {
	t.Helper()
	return availabilityzoneprofiles.New(cloud.Client("load-balancer", "/v2"))
}

func TestR2OctAZProfileCreateBodyAndValidation(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	var body string
	cloud.Mux.HandleFunc("POST /v2/lbaas/availabilityzoneprofiles", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body = r2OctAZBody(t, r)
		testcloud.JSON(w, 201, r2OctAZProfileJSON)
	})
	api := r2OctAZProfilesAPI(t, cloud)
	got, err := api.Create(context.Background(), availabilityzoneprofiles.CreateOpts{Name: "p", ProviderName: "amphora", AvailabilityZoneData: `{"compute_zone": "nova"}`})
	if err != nil {
		t.Fatal(err)
	}
	if body != `{"availability_zone_profile":{"availability_zone_data":"{\"compute_zone\": \"nova\"}","name":"p","provider_name":"amphora"}}` {
		t.Fatalf("body %s", body)
	}
	if got.ID == "" || got.AvailabilityZoneData != `{"compute_zone": "nova"}` || got.StatusCode != 201 {
		t.Fatalf("%+v", got)
	}
	calls.Store(0)
	for name, call := range map[string]func() error{
		"missing name": func() error {
			_, err := api.Create(context.Background(), availabilityzoneprofiles.CreateOpts{ProviderName: "a", AvailabilityZoneData: "{}"})
			return err
		},
		"missing provider": func() error {
			_, err := api.Create(context.Background(), availabilityzoneprofiles.CreateOpts{Name: "a", AvailabilityZoneData: "{}"})
			return err
		},
		"missing data": func() error {
			_, err := api.Create(context.Background(), availabilityzoneprofiles.CreateOpts{Name: "a", ProviderName: "a"})
			return err
		},
		"core extension": func() error {
			_, err := api.Create(context.Background(), availabilityzoneprofiles.CreateOpts{Name: "a", ProviderName: "a", AvailabilityZoneData: "{}"}, availabilityzoneprofiles.WithCreateField("name", "b"))
			return err
		},
	} {
		if err := call(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("invalid input reached the server")
	}
}

func TestR2OctAZProfileGetUpdateDelete(t *testing.T) {
	const id = "11111111-1111-1111-1111-111111111111"
	cloud := testcloud.New(t)
	var body string
	var deletes atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/lbaas/availabilityzoneprofiles/"+id, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, r2OctAZProfileJSON) })
	cloud.Mux.HandleFunc("PUT /v2/lbaas/availabilityzoneprofiles/"+id, func(w http.ResponseWriter, r *http.Request) {
		body = r2OctAZBody(t, r)
		testcloud.JSON(w, 200, r2OctAZProfileJSON)
	})
	cloud.Mux.HandleFunc("DELETE /v2/lbaas/availabilityzoneprofiles/"+id, func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		testcloud.JSON(w, 409, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v2/lbaas/availabilityzoneprofiles/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("DELETE /v2/lbaas/availabilityzoneprofiles/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api := r2OctAZProfilesAPI(t, cloud)
	if got, err := api.Get(context.Background(), id); err != nil || got.ProviderName != "amphora" {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err := api.Get(context.Background(), "missing"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("%v", err)
	}
	empty := ""
	if _, err := api.Update(context.Background(), id, availabilityzoneprofiles.UpdateOpts{Name: &empty}); err != nil || body != `{"availability_zone_profile":{"name":""}}` {
		t.Fatalf("%v %s", err, body)
	}
	if _, err := api.Update(context.Background(), id, availabilityzoneprofiles.UpdateOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("empty update: %v", err)
	}
	if _, err := api.Update(context.Background(), id, availabilityzoneprofiles.UpdateOpts{}, availabilityzoneprofiles.WithUpdateField("provider_name", "x")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("core collision: %v", err)
	}
	if err := api.Delete(context.Background(), id); err == nil || deletes.Load() != 1 {
		t.Fatalf("409 must surface once: %v %d", err, deletes.Load())
	}
	if err := api.Delete(context.Background(), "missing"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("%v", err)
	}
	if err := api.Remove(context.Background(), resource.ID("missing")); err != nil {
		t.Fatalf("remove ignores missing: %v", err)
	}
	if _, err := api.Get(context.Background(), "a/b"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("%v", err)
	}
}

func TestR2OctAZProfileListQueryPagingAndCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []string
	cloud.Mux.HandleFunc("GET /v2/lbaas/availabilityzoneprofiles", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		if r.URL.Query().Get("marker") == "" {
			testcloud.JSON(w, 200, `{"availability_zone_profiles":[{"id":"a","name":"n"}],"availability_zone_profiles_links":[{"rel":"next","href":"`+cloud.Server.URL+`/v2/lbaas/availabilityzoneprofiles?marker=a"}]}`)
			return
		}
		testcloud.JSON(w, 200, `{"availability_zone_profiles":[{"id":"b","name":"n"}]}`)
	})
	api := r2OctAZProfilesAPI(t, cloud)
	name := "n"
	var ids []string
	for p, err := range api.List(context.Background(), availabilityzoneprofiles.WithListOptions(availabilityzoneprofiles.ListOpts{Name: &name, Fields: []string{"id"}})) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, p.ID)
	}
	if len(ids) != 2 || ids[1] != "b" || queries[0] != "fields=id&name=n" || queries[1] != "fields=id&marker=a&name=n" {
		t.Fatalf("ids=%v queries=%v", ids, queries)
	}
	if all, err := api.All(context.Background(), resource.WithPaginated(false)); err != nil || len(all) != 1 {
		t.Fatalf("single page: %v %v", all, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(ctx, "a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("%v", err)
	}
	if err := api.Delete(ctx, "a"); !errors.Is(err, context.Canceled) {
		t.Fatalf("%v", err)
	}
}

func TestR2OctAZProfileFindIdentityByNameAndDuplicates(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/lbaas/availabilityzoneprofiles/dup", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/lbaas/availabilityzoneprofiles", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"availability_zone_profiles":[{"id":"1","name":"dup"},{"id":"2","name":"dup"}]}`)
	})
	api := r2OctAZProfilesAPI(t, cloud)
	if _, err := api.FindIdentity(context.Background(), "dup"); err == nil {
		t.Fatal("duplicate names must fail")
	}
	if _, err := api.Find(context.Background(), resource.Name("dup")); err == nil {
		t.Fatal("duplicate names must fail")
	}
}
