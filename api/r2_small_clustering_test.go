package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestR2SmallClusteringPythonMetadataRoutesAndBodies(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/senlin/v1")
	var mu sync.Mutex
	var seen []string
	cloud.Mux.HandleFunc("/senlin/v1/clusters/c1/metadata", func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		mu.Lock()
		seen = append(seen, r.Method+" "+string(data))
		mu.Unlock()
		w.Header().Set("X-Request-ID", "meta")
		testcloud.JSON(w, 200, `{"opaque":true}`)
	})
	cloud.Mux.HandleFunc("/senlin/v1/clusters/c1/metadata/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Method+" "+r.URL.EscapedPath())
		mu.Unlock()
		w.WriteHeader(204)
	})
	profile := clusters.New(client).PythonMetadata()
	ctx := context.Background()
	receipt, err := profile.SetMetadata(ctx, "c1", map[string]any{"k": 1, "n": nil})
	if err != nil || receipt.StatusCode != 200 || receipt.Header.Get("X-Request-ID") != "meta" || string(receipt.Body) != `{"opaque":true}` {
		t.Fatalf("receipt=%v err=%v", receipt, err)
	}
	if _, err := profile.SetMetadata(ctx, "c1", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := profile.ReplaceMetadata(ctx, "c1", map[string]any{"a": "b"}); err != nil {
		t.Fatal(err)
	}
	cleared, err := profile.DeleteMetadata(ctx, "c1", nil)
	if err != nil || cleared.Cleared == nil || cleared.Deleted != nil {
		t.Fatalf("cleared=%v err=%v", cleared, err)
	}
	deleted, err := profile.DeleteMetadata(ctx, "c1", []string{"a/b", "c"})
	if err != nil || len(deleted.Deleted) != 2 || deleted.Deleted[0].Receipt.StatusCode != 204 {
		t.Fatalf("deleted=%v err=%v", deleted, err)
	}
	if none, err := profile.DeleteMetadata(ctx, "c1", []string{}); err != nil || len(none.Deleted) != 0 {
		t.Fatalf("none=%v err=%v", none, err)
	}
	want := []string{`POST {"metadata":{"k":1,"n":null}}`, `POST {"metadata":{}}`, `PUT {"metadata":{"a":"b"}}`, `PUT {"metadata":{}}`,
		"DELETE /senlin/v1/clusters/c1/metadata/a%2Fb", "DELETE /senlin/v1/clusters/c1/metadata/c"}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("requests=%q", seen)
	}
}

func TestR2SmallClusteringPythonMetadataStock404PartialDeleteAndRejection(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("clustering", "/senlin/v1")
	var calls []string
	cloud.Mux.HandleFunc("/senlin/v1/clusters/c1/metadata/", func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		if r.URL.Path == "/senlin/v1/clusters/c1/metadata/bad" {
			testcloud.JSON(w, 404, `{"error":{"code":404}}`)
			return
		}
		w.WriteHeader(204)
	})
	cloud.Mux.HandleFunc("/senlin/v1/clusters/c1/metadata", func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method)
		testcloud.JSON(w, 404, `{"error":{"code":404}}`)
	})
	profile := clusters.New(client).PythonMetadata()
	ctx := context.Background()
	if _, err := profile.SetMetadata(ctx, "c1", map[string]any{"a": 1}); err == nil || len(calls) != 1 {
		t.Fatalf("stock 404 must stay an error without fallback: %v %v", err, calls)
	}
	partial, err := profile.DeleteMetadata(ctx, "c1", []string{"ok", "bad", "never"})
	if err == nil || len(partial.Deleted) != 1 || partial.Deleted[0].Key != "ok" || len(calls) != 3 {
		t.Fatalf("partial=%v err=%v calls=%v", partial, err, calls)
	}
	n := len(calls)
	if _, err := profile.DeleteMetadata(ctx, "c1", []string{" "}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := profile.SetMetadata(ctx, "bad id/../", nil); err == nil {
		t.Fatal("identifier must be validated")
	}
	if _, err := profile.SetMetadata(ctx, "c1", map[string]any{"x": func() {}}); err == nil {
		t.Fatal("unencodable value must fail")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := profile.ReplaceMetadata(canceled, "c1", nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if len(calls) != n {
		t.Fatalf("rejected input reached HTTP: %v", calls)
	}
}
