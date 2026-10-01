package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"gophercloudsdk/baremetalintrospection/v1/introspection"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestIntrospectionFinishedWaitRejectsAttributeReplacementBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		t.Error("fixed completion waiter made an HTTP request")
		http.Error(w, "unexpected", 500)
	})
	api := introspection.New(cloud.Client("baremetal-introspection", "/v1"))
	value, err := api.WaitUntilFinished(context.Background(), resource.ID("node"), resource.WithStatusAttribute("state"))
	if value != nil || !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 0 {
		t.Fatalf("value=%v calls=%d error=%v", value, calls.Load(), err)
	}
}

func TestIntrospectionServiceErrorSurvivesDisabledFailureStates(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v1/introspection/node", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"uuid":"node","finished":true,"state":"finished","error":"inspection failed"}`)
	})
	api := introspection.New(cloud.Client("baremetal-introspection", "/v1"))
	var failure *introspection.IntrospectionFailureError
	value, err := api.WaitUntilFinished(context.Background(), resource.ID("node"), resource.WithFailureStates())
	if value != nil || !errors.As(err, &failure) || failure.Message != "inspection failed" || failure.Details == nil || !failure.Details.Finished {
		t.Fatalf("value=%v failure=%v error=%v", value, failure, err)
	}
}
