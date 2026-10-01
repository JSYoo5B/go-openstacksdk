package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/orchestration/v1/stackresources"
)

func TestHeatStackResourcesRetainRawFieldsAndIndependentHeaders(t *testing.T) {
	cloud := testcloud.New(t)
	value := heatResourceBody("node", "app", "fixed", "CREATE_COMPLETE")
	value["future"] = json.RawMessage(`{"count":9007199254740993}`)
	value["optional"] = nil
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/resources", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "list")
		heatResourceResponse(w, "resources", []any{value, value})
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/resources/node", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "get")
		heatResourceResponse(w, "resource", value)
	})
	scope := heatResourceScope(t, stackresources.New(cloud.Client("orchestration", "/heat")))
	items, err := scope.All(context.Background())
	if err != nil || len(items) != 2 || items[0].StatusCode != 200 || items[0].Header.Get("X-Request-Id") != "list" || string(items[0].Body["future"]) != `{"count":9007199254740993}` || string(items[0].Body["optional"]) != "null" {
		t.Fatalf("list %+v/%v", items, err)
	}
	items[0].Header.Set("X-Request-Id", "changed")
	items[0].Body["future"] = json.RawMessage(`false`)
	if items[1].Header.Get("X-Request-Id") != "list" || string(items[1].Body["future"]) != `{"count":9007199254740993}` {
		t.Fatal("list response metadata aliases")
	}
	got, err := scope.Get(context.Background(), "node")
	if err != nil || !got.Detailed || got.StatusCode != 200 || got.Header.Get("X-Request-Id") != "get" || string(got.Body["future"]) != `{"count":9007199254740993}` || string(got.Body["optional"]) != "null" {
		t.Fatalf("get %+v/%v", got, err)
	}
}

func TestHeatStackResourcesRejectMalformedListEnvelopes(t *testing.T) {
	for _, body := range []string{`null`, `{}`, `{"resources":null}`, `{"resources":{}}`, `{"resources":[null]}`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("GET /heat/stacks/app/fixed/resources", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, body) })
			scope := heatResourceScope(t, stackresources.New(cloud.Client("orchestration", "/heat")))
			if items, err := scope.All(context.Background()); err == nil || items != nil {
				t.Fatalf("malformed list accepted %+v/%v", items, err)
			}
		})
	}
}
