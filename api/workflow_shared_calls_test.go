package api_test

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/workflow/v2/workflows"
)

// Mistral workflow Update takes no extension body field or header, so every
// shared call the write category admits fails before HTTP; its query
// extension stays workflows.WithUpdateQuery.
func TestWorkflowSharedCallsUnsupportedBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	api := workflows.New(mistralClient(cloud))
	for _, call := range []request.WriteCall{request.Field("vendor", true), request.Header("X-Test", "1"), request.Headers(nil), request.Microversion("2.0")} {
		entry, err := request.Entry(call)
		if err != nil {
			t.Fatal(err)
		}
		_, err = api.Update(context.Background(), workflows.UpdateOpts{Definition: strings.NewReader(pyMsgWfDefinition)}, call)
		requireUnsupportedCall(t, err, entry.Kind.String())
	}
	if n := calls.Load(); n != 0 {
		t.Fatalf("HTTP requests = %d", n)
	}
}
