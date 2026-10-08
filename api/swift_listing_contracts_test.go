package api_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
)

func TestSwiftContainerListRetainsCountsAndPagesLazily(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	cloud.Mux.HandleFunc("/swift/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodGet || r.URL.Query().Get("limit") != "1" {
			t.Errorf("method=%s query=%v", r.Method, r.URL.Query())
		}
		switch r.URL.Query().Get("marker") {
		case "":
			testcloud.JSON(w, 200, `[{"name":"first","count":2,"bytes":5}]`)
		case "first":
			testcloud.JSON(w, 200, `[{"name":"second","count":3,"bytes":8}]`)
		case "second":
			testcloud.JSON(w, 200, `[]`)
		default:
			t.Errorf("marker=%q", r.URL.Query().Get("marker"))
		}
	})
	api := containers.New(cloud.Client("object-store", "/swift"))
	opts := containers.WithListOptions(containers.ListOpts{Limit: 1})
	for value, err := range api.List(context.Background(), opts) {
		if err != nil || value.Name != "first" || value.Count != 2 || value.Bytes != 5 {
			t.Fatalf("value=%v err=%v", value, err)
		}
		break
	}
	if calls != 1 {
		t.Fatalf("early break fetched %d pages", calls)
	}
	var values []*containers.Container
	for value, err := range api.List(context.Background(), opts) {
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, value)
	}
	if len(values) != 2 || values[1].Name != "second" || values[1].Count != 3 || values[1].Bytes != 8 || calls != 4 {
		t.Fatalf("values=%v calls=%d", values, calls)
	}
}

func TestSwiftObjectListRetainsMetadataAndSubdirectoryEntries(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/swift/container", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("delimiter") != "/" {
			t.Errorf("query=%v", r.URL.Query())
		}
		if r.URL.Query().Get("marker") != "" {
			testcloud.JSON(w, 200, `[]`)
			return
		}
		testcloud.JSON(w, 200, `[{"name":"folder/a.txt","bytes":5,"content_type":"text/plain","hash":"checksum","last_modified":"2030-01-02T03:04:05.123456","is_latest":true,"version_id":"version"},{"subdir":"other/"}]`)
	})
	var values []*objects.Object
	for value, err := range objects.New(cloud.Client("object-store", "/swift")).List(context.Background(), "container", objects.WithListOptions(objects.ListOpts{Delimiter: "/"})) {
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, value)
	}
	if len(values) != 2 {
		t.Fatal(values)
	}
	item := values[0]
	if item.Name != "folder/a.txt" || item.Bytes != 5 || item.ContentType != "text/plain" || item.Hash != "checksum" || item.VersionID != "version" || !item.IsLatest || !item.LastModified.Equal(time.Date(2030, 1, 2, 3, 4, 5, 123456000, time.UTC)) {
		t.Fatalf("object=%+v", item)
	}
	if values[1].Subdir != "other/" || values[1].Name != "" {
		t.Fatalf("subdir=%+v", values[1])
	}
}
