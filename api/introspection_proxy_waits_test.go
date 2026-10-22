package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	inspector "github.com/JSYoo5B/go-openstacksdk/baremetalintrospection/v1"
	"github.com/JSYoo5B/go-openstacksdk/baremetalintrospection/v1/introspection"
	"github.com/JSYoo5B/go-openstacksdk/baremetalintrospection/v1/rules"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestIntrospectionProxyWaitsAttributeStatusAndGETOnlyDeletion(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var stateGets, failedGets, goneGets, ruleGets, readableGets atomic.Int32
	only := func(counter *atomic.Int32, reply func(n int32, w http.ResponseWriter)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("waiter sent %s %s", r.Method, r.URL.Path)
			}
			reply(counter.Add(1), w)
		}
	}
	cloud.Mux.HandleFunc("/v1/introspection/node-1", only(&stateGets, func(n int32, w http.ResponseWriter) {
		state := "waiting"
		if n >= 2 {
			state = "Processing"
		}
		testcloud.JSON(w, 200, `{"uuid":"node-1","finished":false,"state":"`+state+`","error":null}`)
	}))
	cloud.Mux.HandleFunc("/v1/introspection/node-2", only(&failedGets, func(_ int32, w http.ResponseWriter) {
		testcloud.JSON(w, 200, `{"uuid":"node-2","finished":true,"state":"error","error":"boom"}`)
	}))
	cloud.Mux.HandleFunc("/v1/introspection/node-3", only(&goneGets, func(n int32, w http.ResponseWriter) {
		if n < 3 {
			testcloud.JSON(w, 200, `{"uuid":"node-3","finished":true,"state":"finished","error":null}`)
			return
		}
		testcloud.JSON(w, 404, `{"error":{"message":"not found"}}`)
	}))
	cloud.Mux.HandleFunc("/v1/introspection/node-4", only(&readableGets, func(_ int32, w http.ResponseWriter) {
		testcloud.JSON(w, 200, `{"uuid":"node-4","finished":true,"state":"aborted","error":"Canceled by operator"}`)
	}))
	cloud.Mux.HandleFunc("/v1/rules/rule-1", only(&ruleGets, func(n int32, w http.ResponseWriter) {
		if n < 2 {
			testcloud.JSON(w, 200, `{"uuid":"rule-1","description":"d","conditions":[],"actions":[]}`)
			return
		}
		testcloud.JSON(w, 404, `{"error":{"message":"not found"}}`)
	}))
	client := cloud.Client("baremetal-introspection", "/v1")
	collection := introspection.New(client).Resources
	ctx := context.Background()
	fast := resource.WithPollInterval(time.Millisecond)

	// Python's default status attribute does not exist on Introspection.
	if _, err := inspector.WaitForStatus(ctx, collection, resource.ID("node-1"), "processing", fast); !errors.Is(err, resource.ErrUnsupported) || stateGets.Load() != 0 {
		t.Fatalf("default attribute err=%v gets=%d", err, stateGets.Load())
	}
	value, err := inspector.WaitForStatus(ctx, collection, resource.ID("node-1"), "PROCESSING", fast, resource.WithStatusAttribute("state"))
	if err != nil || value.State != "Processing" || value.UUID != "node-1" || stateGets.Load() != 2 {
		t.Fatalf("value=%v err=%v gets=%d", value, err, stateGets.Load())
	}
	var failed *resource.FailedStateError
	if _, err := inspector.WaitForStatus(ctx, collection, resource.ID("node-2"), "finished", fast, resource.WithStatusAttribute("state")); !errors.As(err, &failed) || failed.Status != "error" || failedGets.Load() != 1 {
		t.Fatalf("error state err=%v gets=%d", err, failedGets.Load())
	}
	// The boolean finished flag is not a string attribute.
	if _, err := inspector.WaitForStatus(ctx, collection, resource.ID("node-1"), "true", fast, resource.WithStatusAttribute("finished")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("boolean attribute err=%v", err)
	}

	// Deletion waiting is a GET-only observation: there is no DELETE route.
	if err := inspector.WaitForDelete(ctx, collection, resource.ID("node-3"), fast); err != nil || goneGets.Load() != 3 {
		t.Fatalf("delete err=%v gets=%d", err, goneGets.Load())
	}
	// An aborted introspection remains readable, so only the deadline ends it.
	if err := inspector.WaitForDelete(ctx, collection, resource.ID("node-4"), fast, resource.WithTimeout(30*time.Millisecond)); !errors.Is(err, context.DeadlineExceeded) || readableGets.Load() < 2 {
		t.Fatalf("readable delete err=%v gets=%d", err, readableGets.Load())
	}
	if err := inspector.WaitForDelete(ctx, rules.New(client).Resources(), resource.ID("rule-1"), fast); err != nil || ruleGets.Load() != 2 {
		t.Fatalf("rule delete err=%v gets=%d", err, ruleGets.Load())
	}
}
