package api_test

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shares"
)

func TestManilaShareMetadataReadsWholeMapAndSingleKey(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/shares/share-1/metadata", func(w http.ResponseWriter, r *http.Request) {
		manilaShareVersion(t, r, "2.82")
		testcloud.JSON(w, 200, `{"metadata":{"empty":"","owner":"ops"}}`)
	})
	cloud.Mux.HandleFunc("GET /v2/shares/share-1/metadata/owner", func(w http.ResponseWriter, r *http.Request) {
		manilaShareVersion(t, r, "2.82")
		testcloud.JSON(w, 200, `{"meta":{"owner":"ops"}}`)
	})
	cloud.Mux.HandleFunc("GET /v2/shares/share-1/metadata/missing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	cloud.Mux.HandleFunc("GET /v2/shares/empty/metadata", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"metadata":{}}`)
	})
	api := newManilaShares(cloud, "2.82")
	all, err := api.GetMetadata(context.Background(), "share-1")
	if err != nil || !reflect.DeepEqual(all, map[string]string{"empty": "", "owner": "ops"}) {
		t.Fatalf("all=%v error=%v", all, err)
	}
	one, err := api.GetMetadatum(context.Background(), "share-1", "owner")
	if err != nil || !reflect.DeepEqual(one, map[string]string{"owner": "ops"}) {
		t.Fatalf("one=%v error=%v", one, err)
	}
	none, err := api.GetMetadata(context.Background(), "empty")
	if err != nil || none == nil || len(none) != 0 {
		t.Fatalf("none=%#v error=%v", none, err)
	}
	_, err = api.GetMetadatum(context.Background(), "share-1", "missing")
	manilaShareCode(t, err, 404)
}

func TestManilaShareSetAndUpdateMetadataPostExactBodiesAndAcceptOnly200(t *testing.T) {
	type call func(api *shares.API, opts map[string]string) (map[string]string, error)
	ctx := context.Background()
	calls := map[string]call{
		"SetMetadata": func(api *shares.API, m map[string]string) (map[string]string, error) {
			return api.SetMetadata(ctx, "share-1", shares.SetMetadataOpts{Metadata: m})
		},
		"UpdateMetadata": func(api *shares.API, m map[string]string) (map[string]string, error) {
			return api.UpdateMetadata(ctx, "share-1", shares.UpdateMetadataOpts{Metadata: m})
		},
	}
	for name, run := range calls {
		for _, tc := range []struct {
			metadata map[string]string
			body     string
		}{
			{map[string]string{"empty": "", "k": "v"}, `{"metadata":{"empty":"","k":"v"}}`},
			{map[string]string{}, `{"metadata":{}}`},
			{nil, `{"metadata":null}`},
		} {
			for _, status := range []int{200, 201, 202, 204} {
				cloud := testcloud.New(t)
				cloud.Mux.HandleFunc("POST /v2/shares/share-1/metadata", func(w http.ResponseWriter, r *http.Request) {
					manilaShareVersion(t, r, "2.82")
					if got := manilaShareRead(t, r); got != tc.body {
						t.Errorf("%s body=%s want %s", name, got, tc.body)
					}
					if status == 204 {
						w.WriteHeader(204)
						return
					}
					testcloud.JSON(w, status, `{"metadata":{"k":"v"}}`)
				})
				value, err := run(newManilaShares(cloud, "2.82"), tc.metadata)
				if (status == 200) != (err == nil) || (status == 200 && !reflect.DeepEqual(value, map[string]string{"k": "v"})) {
					t.Fatalf("%s status %d value=%v error=%v", name, status, value, err)
				}
				if status != 200 {
					manilaShareCode(t, err, status)
				}
			}
		}
	}
}

func TestManilaShareDeleteMetadatumAcceptsOnly200(t *testing.T) {
	for _, status := range []int{200, 202, 204} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("DELETE /v2/shares/share-1/metadata/owner", func(w http.ResponseWriter, r *http.Request) {
			manilaShareVersion(t, r, "2.82")
			if body := manilaShareRead(t, r); body != "" {
				t.Errorf("body=%q", body)
			}
			w.WriteHeader(status)
		})
		err := newManilaShares(cloud, "2.82").DeleteMetadatum(context.Background(), "share-1", "owner")
		if (status == 200) != (err == nil) {
			t.Fatalf("status %d error=%v", status, err)
		}
		if status != 200 {
			manilaShareCode(t, err, status)
		}
	}
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("DELETE /v2/shares/share-1/metadata/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	manilaShareCode(t, newManilaShares(cloud, "2.82").DeleteMetadatum(context.Background(), "share-1", "missing"), 404)
}
