package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/db/v1/instances"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Shared calls that the write category admits but Trove UpdateAttributes
// never supported fail before HTTP; Field and Header apply in caller order.
func TestDBSharedCalls(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	var header http.Header
	var body map[string]map[string]json.RawMessage
	cloud.Mux.HandleFunc("/v1.0/tenant/instances/i1", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		header = r.Header.Clone()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		w.WriteHeader(202)
	})
	api := instances.New(cloud.Client("database", "/v1.0/tenant"))
	attrs := instances.UpdateAttributes{Name: request.Present("n")}

	for _, call := range []request.WriteCall{request.Microversion("1.0"), request.Headers(nil)} {
		entry, err := request.Entry(call)
		if err != nil {
			t.Fatal(err)
		}
		_, err = api.UpdateAttributes(ctx, resource.ID("i1"), attrs, call)
		requireUnsupportedCall(t, err, entry.Kind.String())
	}
	if n := calls.Load(); n != 0 {
		t.Fatalf("HTTP requests = %d", n)
	}

	if _, err := api.UpdateAttributes(ctx, resource.ID("i1"), attrs,
		request.Field("vendor", "first"), request.Header("X-Vendor", "first"),
		request.Field("vendor", "second"), request.Header("X-Vendor", "second")); err != nil {
		t.Fatal(err)
	}
	if string(body["instance"]["vendor"]) != `"second"` || header.Get("X-Vendor") != "second" {
		t.Fatal(body, header)
	}
}
