package openstack_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionFlavorFindIdentityExtraSpecs(t *testing.T) {
	cloud := testcloud.New(t)
	conn := connection(t, cloud)
	ctx := context.Background()
	service, err := conn.Compute(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var gets, lists, extras atomic.Int32
	var deny atomic.Bool
	base := "/compute/v2.1/project/flavors"
	cloud.Mux.HandleFunc("GET "+base+"/small", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if q := r.URL.Query(); q.Has("name") || q.Has("is_public") || q.Get("minRam") != "256" || !reflect.DeepEqual(q["tag"], []string{"linux", "arm64"}) {
			t.Errorf("member query = %v", q)
		}
		testcloud.JSON(w, http.StatusForbidden, `{"error":"member denied"}`)
	})
	cloud.Mux.HandleFunc("GET "+base+"/detail", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		q := r.URL.Query()
		if q.Has("name") || q.Get("is_public") != "None" || q.Get("minRam") != "256" || !reflect.DeepEqual(q["tag"], []string{"linux", "arm64"}) {
			t.Errorf("list query = %v", q)
		}
		if q.Get("marker") == "next" {
			testcloud.JSON(w, http.StatusOK, `{"flavors":[{"id":"42","name":"small","ram":512,"swap":"1024","extra_specs":{}}]}`)
			return
		}
		q.Set("marker", "next")
		testcloud.JSON(w, http.StatusOK, `{"flavors":[{"id":"7","name":"other"}],"flavors_links":[{"rel":"next","href":"`+cloud.Server.URL+base+`/detail?`+q.Encode()+`"}]}`)
	})
	cloud.Mux.HandleFunc("GET "+base+"/42/os-extra_specs", func(w http.ResponseWriter, r *http.Request) {
		extras.Add(1)
		if r.URL.RawQuery != "" {
			t.Errorf("lookup query leaked to extra specs: %v", r.URL)
		}
		if deny.Load() {
			testcloud.JSON(w, http.StatusNotFound, `{"error":"extra specs absent"}`)
			return
		}
		testcloud.JSON(w, http.StatusOK, `{"extra_specs":{"hw:cpu_policy":"dedicated"}}`)
	})
	query := resource.IdentityFindOpts{Query: map[string][]string{"minRam": {"256"}, "tag": {"linux", "arm64"}}}
	option := resource.WithIdentityFindOptions(query)
	query.Query["tag"][0] = "changed"
	flavor, err := service.Flavors.FindIdentity(ctx, "small", option)
	if err != nil || flavor == nil || flavor.ID != "42" || flavor.Swap != 1024 || len(flavor.ExtraSpecs) != 0 || extras.Load() != 0 {
		t.Fatalf("default flavor = %#v, err = %v, extras = %d", flavor, err, extras.Load())
	}
	flavor, err = service.Flavors.FindIdentity(ctx, "small", option, resource.WithIdentityFindExtraSpecs(true))
	if err != nil || flavor == nil || flavor.ExtraSpecs["hw:cpu_policy"] != "dedicated" || extras.Load() != 1 {
		t.Fatalf("enriched flavor = %#v, err = %v, extras = %d", flavor, err, extras.Load())
	}
	deny.Store(true)
	flavor, err = service.Flavors.FindIdentity(ctx, "small", option, resource.WithIdentityFindExtraSpecs(true))
	if flavor != nil || !gophercloud.ResponseCodeIs(err, http.StatusNotFound) || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("extra specs failure became lookup absence: %#v, %v", flavor, err)
	}
	if gets.Load() != 3 || lists.Load() != 6 || extras.Load() != 2 {
		t.Fatalf("GET/list/extra specs = %d/%d/%d", gets.Load(), lists.Load(), extras.Load())
	}
}
