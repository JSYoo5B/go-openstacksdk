package api_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shares"
)

func TestManilaShareGrantAccessSendsExactBodyAndAcceptsOnly200(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, false}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("POST /v2/shares/share-1/action", func(w http.ResponseWriter, r *http.Request) {
			manilaShareVersion(t, r, "2.82")
			want := `{"allow_access":{"access_level":"","access_to":"192.0.2.0/24","access_type":"ip","vendor":{"x":false}}}`
			if got := manilaShareRead(t, r); got != want {
				t.Errorf("body=%s", got)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, `{"access":{"id":"rule-1","share_id":"share-1","access_type":"ip","access_to":"192.0.2.0/24","access_level":"rw","state":"queued_to_apply","access_key":"k"}}`)
		})
		value, err := newManilaShares(cloud, "2.82").GrantAccess(context.Background(), "share-1",
			shares.GrantAccessOpts{AccessType: "ip", AccessTo: "192.0.2.0/24"},
			shares.WithGrantAccessField("vendor", map[string]bool{"x": false}))
		if tc.success != (err == nil) {
			t.Fatalf("status %d value=%+v error=%v", tc.status, value, err)
		}
		if tc.success && (value == nil || value.ID != "rule-1" || value.ShareID != "share-1" || value.AccessLevel != "rw" || value.State != "queued_to_apply" || value.AccessKey != "k") {
			t.Fatalf("value=%+v", value)
		}
		if !tc.success {
			manilaShareCode(t, err, tc.status)
		}
	}
}

func TestManilaShareListAccessRightsPostsNullAccessListAction(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, false}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("POST /v2/shares/share-1/action", func(w http.ResponseWriter, r *http.Request) {
			manilaShareVersion(t, r, "2.45")
			if got := manilaShareRead(t, r); got != `{"access_list":null}` {
				t.Errorf("body=%s", got)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, `{"access_list":[{"id":"rule-1","access_type":"ip","access_to":"192.0.2.1","access_level":"ro","state":"active"},{"id":"rule-2"}]}`)
		})
		values, err := newManilaShares(cloud, "2.45").ListAccessRights(context.Background(), "share-1")
		if tc.success != (err == nil) {
			t.Fatalf("status %d values=%+v error=%v", tc.status, values, err)
		}
		if tc.success && (len(values) != 2 || values[0].ID != "rule-1" || values[0].AccessLevel != "ro" || values[1].AccessTo != "") {
			t.Fatalf("values=%+v", values)
		}
		if !tc.success {
			manilaShareCode(t, err, tc.status)
		}
	}
}

func TestManilaShareListAccessRightsEmptyAndNullListsDecodeWithoutRows(t *testing.T) {
	for _, body := range []string{`{"access_list":[]}`, `{"access_list":null}`, `{}`} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("POST /v2/shares/share-1/action", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, body) })
		values, err := newManilaShares(cloud, "2.45").ListAccessRights(context.Background(), "share-1")
		if err != nil || len(values) != 0 {
			t.Fatalf("body %s values=%v error=%v", body, values, err)
		}
	}
}

func TestManilaShareExportLocationsReadListAndSingle(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/shares/share-1/export_locations", func(w http.ResponseWriter, r *http.Request) {
		manilaShareVersion(t, r, "2.14")
		testcloud.JSON(w, 200, `{"export_locations":[{"id":"loc-1","path":"192.0.2.5:/share","share_instance_id":"inst-1","is_admin_only":false,"preferred":false},{"id":"loc-2","path":"192.0.2.6:/share","is_admin_only":true,"preferred":true}]}`)
	})
	cloud.Mux.HandleFunc("GET /v2/shares/share-1/export_locations/loc-2", func(w http.ResponseWriter, r *http.Request) {
		manilaShareVersion(t, r, "2.14")
		testcloud.JSON(w, 200, `{"export_location":{"id":"loc-2","path":"192.0.2.6:/share","is_admin_only":true,"preferred":true}}`)
	})
	cloud.Mux.HandleFunc("GET /v2/shares/share-1/export_locations/missing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v2/shares/gone/export_locations", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api := newManilaShares(cloud, "2.14")
	values, err := api.ListExportLocations(context.Background(), "share-1")
	if err != nil || len(values) != 2 || values[0].ShareInstanceID != "inst-1" || values[0].IsAdminOnly || values[0].Preferred || !values[1].IsAdminOnly || !values[1].Preferred {
		t.Fatalf("values=%+v error=%v", values, err)
	}
	one, err := api.GetExportLocation(context.Background(), "share-1", "loc-2")
	if err != nil || !reflect.DeepEqual(one.Path, "192.0.2.6:/share") || !one.Preferred || one.ID != "loc-2" {
		t.Fatalf("one=%+v error=%v", one, err)
	}
	_, err = api.GetExportLocation(context.Background(), "share-1", "missing")
	manilaShareCode(t, err, 404)
	_, err = api.ListExportLocations(context.Background(), "gone")
	manilaShareCode(t, err, 404)
}
