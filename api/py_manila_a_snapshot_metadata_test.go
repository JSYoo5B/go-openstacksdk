package api_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/snapshots"
)

func newPyManilaASnapshots(cloud *testcloud.Cloud, version string) *snapshots.API {
	client := cloud.Client("sharev2", "/v2")
	client.Microversion = version
	return snapshots.New(client)
}

// fetch_share_snapshot_metadata and fetch_share_snapshot_metadata_item.
func TestPyManilaASnapshotMetadataReadsWholeMapAndSingleKey(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/snapshots/snap-1/metadata", func(w http.ResponseWriter, r *http.Request) {
		manilaShareVersion(t, r, "2.82")
		testcloud.JSON(w, 200, `{"metadata":{"empty":"","owner":"ops"}}`)
	})
	cloud.Mux.HandleFunc("GET /v2/snapshots/snap-1/metadata/owner", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"meta":{"owner":"ops"}}`)
	})
	cloud.Mux.HandleFunc("GET /v2/snapshots/snap-1/metadata/a%2Fb", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"meta":{"a/b":"slash"}}`)
	})
	cloud.Mux.HandleFunc("GET /v2/snapshots/snap-1/metadata/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/snapshots/none/metadata", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/snapshots/accepted/metadata", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 202, `{"metadata":{}}`) })
	api := newPyManilaASnapshots(cloud, "2.82")
	ctx := context.Background()
	all, err := api.GetMetadata(ctx, "snap-1")
	if err != nil || !reflect.DeepEqual(all, map[string]string{"empty": "", "owner": "ops"}) {
		t.Fatalf("all=%v err=%v", all, err)
	}
	one, err := api.GetMetadatum(ctx, "snap-1", "owner")
	if err != nil || !reflect.DeepEqual(one, map[string]string{"owner": "ops"}) {
		t.Fatalf("one=%v err=%v", one, err)
	}
	escaped, err := api.GetMetadatum(ctx, "snap-1", "a/b")
	if err != nil || escaped["a/b"] != "slash" {
		t.Fatalf("escaped=%v err=%v", escaped, err)
	}
	none, err := api.GetMetadata(ctx, "none")
	if err != nil || none == nil || len(none) != 0 {
		t.Fatalf("none=%#v err=%v", none, err)
	}
	_, err = api.GetMetadatum(ctx, "snap-1", "missing")
	manilaShareCode(t, err, 404)
	_, err = api.GetMetadata(ctx, "accepted")
	manilaShareCode(t, err, 202)
	if _, err := api.GetMetadatum(ctx, "snap-1", ""); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("empty key err=%v", err)
	}
	if _, err := api.GetMetadata(ctx, "a/b"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("slash id err=%v", err)
	}
}

// delete_share_snapshot_metadata stops at the first failing key.
func TestPyManilaASnapshotDeleteMetadataStopsAtFirstFailure(t *testing.T) {
	cloud := testcloud.New(t)
	var order []string
	cloud.Mux.HandleFunc("DELETE /v2/snapshots/snap-1/metadata/{key}", func(w http.ResponseWriter, r *http.Request) {
		order = append(order, r.PathValue("key"))
		if r.PathValue("key") == "b" {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		w.WriteHeader(200)
	})
	api := newPyManilaASnapshots(cloud, "")
	manilaShareCode(t, api.DeleteMetadata(context.Background(), "snap-1", []string{"a", "b", "c"}), 404)
	if !reflect.DeepEqual(order, []string{"a", "b"}) {
		t.Fatalf("order=%v", order)
	}
	order = nil
	if err := api.DeleteMetadata(context.Background(), "snap-1", []string{"a", "c"}); err != nil || len(order) != 2 {
		t.Fatalf("err=%v order=%v", err, order)
	}
	if err := api.DeleteMetadata(context.Background(), "snap-1", []string{"a", ""}); !errors.Is(err, resource.ErrInvalidOption) || len(order) != 2 {
		t.Fatalf("err=%v order=%v", err, order)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := api.DeleteMetadata(ctx, "snap-1", []string{"a"}); !errors.Is(err, context.Canceled) || len(order) != 2 {
		t.Fatalf("err=%v order=%v", err, order)
	}
}
