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

func TestConnectionImageFindIdentityHiddenSearch(t *testing.T) {
	cloud := testcloud.New(t)
	conn := connection(t, cloud)
	ctx := context.Background()
	service, err := conn.Image(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var gets, ordinary, hidden atomic.Int32
	cloud.Mux.HandleFunc("GET /image/v2/images/ubuntu", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if r.URL.Query().Has("name") || r.URL.Query().Get("os_hidden") != "false" {
			t.Errorf("GET query = %v", r.URL.Query())
		}
		testcloud.JSON(w, http.StatusForbidden, `{"error":"member denied"}`)
	})
	cloud.Mux.HandleFunc("GET /image/v2/images", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if !reflect.DeepEqual(q["tag"], []string{"linux", "arm64"}) {
			t.Errorf("tags = %v", q["tag"])
		}
		switch q.Get("os_hidden") {
		case "false":
			ordinary.Add(1)
			if q.Get("name") != "ubuntu" {
				t.Errorf("normal name = %q", q.Get("name"))
			}
			testcloud.JSON(w, http.StatusOK, `{"images":[]}`)
		case "true":
			hidden.Add(1)
			if q.Has("name") {
				t.Errorf("automatic name leaked into hidden search: %v", q)
			}
			if q.Get("owner") == "empty" {
				testcloud.JSON(w, http.StatusOK, `{"images":[]}`)
				return
			}
			testcloud.JSON(w, http.StatusOK, `{"images":[{"id":"hidden-id","name":"ubuntu","os_hidden":true,"hw_architecture":"aarch64"}]}`)
		default:
			t.Errorf("unexpected hidden policy: %v", q)
			testcloud.JSON(w, http.StatusInternalServerError, `{"error":"query"}`)
		}
	})
	input := resource.IdentityFindOpts{Query: map[string][]string{"tag": {"linux", "arm64"}, "os_hidden": {"false"}}}
	option := resource.WithIdentityFindOptions(input)
	input.Query["tag"][0] = "changed"
	image, err := service.Images.FindIdentity(ctx, "ubuntu", option)
	if err != nil || image == nil || image.ID != "hidden-id" || !image.Hidden || image.Properties["hw_architecture"] != "aarch64" {
		t.Fatalf("hidden image = %#v, err = %v", image, err)
	}
	image, err = service.Images.FindIdentity(ctx, "ubuntu", option, resource.WithIdentityFindQuery("owner", "empty"))
	if image != nil || err != nil {
		t.Fatalf("default absent = %#v, %v", image, err)
	}
	image, err = service.Images.FindIdentity(ctx, "ubuntu", option, resource.WithIdentityFindQuery("owner", "empty"), resource.WithIdentityFindIgnoreMissing(false))
	if image != nil || !errors.Is(err, resource.ErrNotFound) || gophercloud.ResponseCodeIs(err, http.StatusForbidden) {
		t.Fatalf("strict logical absence retained member cause: %#v, %v", image, err)
	}
	if gets.Load() != 3 || ordinary.Load() != 3 || hidden.Load() != 3 {
		t.Fatalf("GET/normal/hidden = %d/%d/%d", gets.Load(), ordinary.Load(), hidden.Load())
	}
}
