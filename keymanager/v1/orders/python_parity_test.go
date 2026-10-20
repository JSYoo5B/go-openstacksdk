package orders_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pythonOrderTransport func(*http.Request) (*http.Response, error)

func (transport pythonOrderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonOrderCall struct{ method, path string }

// pythonOrderAPI replies with the given statuses in order and repeats the last.
func pythonOrderAPI(t *testing.T, calls *[]pythonOrderCall, statuses ...string) *orders.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonOrderTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, pythonOrderCall{req.Method, req.URL.Path})
		status := statuses[min(len(*calls), len(statuses))-1]
		code, body := 200, `{"order_ref":"https://kms/v1/orders/o1","status":"`+status+`","type":"key"}`
		if status == "" {
			code, body = 404, `{"code":404}`
		}
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("key-manager", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/barbican/v1/"
	return orders.New(client)
}

func pythonOrderGets(n int) []pythonOrderCall {
	calls := make([]pythonOrderCall, n)
	for i := range calls {
		calls[i] = pythonOrderCall{http.MethodGet, "/barbican/v1/orders/o1"}
	}
	return calls
}

// resource.wait_for_status polls fetch(skip_cache=True) every interval with
// failures=['ERROR'] compared case-insensitively and no timeout by default.
func TestPythonOrderWaitForStatus(t *testing.T) {
	ctx := context.Background()
	pythonDefaults := []resource.WaitOption{resource.WithUnlimitedWait(), resource.WithFailureStates("ERROR"), resource.WithPollInterval(time.Millisecond)}
	for _, tc := range []struct {
		name     string
		statuses []string
		options  []resource.WaitOption
		failed   string
		timeout  bool
		gets     int
	}{
		{name: "reaches target", statuses: []string{"PENDING", "PENDING", "ACTIVE"}, options: pythonDefaults, gets: 3},
		{name: "ERROR fails", statuses: []string{"PENDING", "error"}, options: pythonDefaults, failed: "error", gets: 2},
		// Only the listed failures stop the wait, exactly like Python.
		{name: "other error-like states keep polling", statuses: []string{"error_custom", "ACTIVE"}, options: pythonDefaults, gets: 2},
		{name: "attribute", statuses: []string{"ACTIVE"}, options: append([]resource.WaitOption{resource.WithStatusAttribute("type")}, pythonDefaults...), gets: 1},
		{name: "wait seconds", statuses: []string{"PENDING"}, options: []resource.WaitOption{resource.WithTimeout(20 * time.Millisecond), resource.WithPollInterval(time.Millisecond)}, timeout: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonOrderCall
			api := pythonOrderAPI(t, &calls, tc.statuses...)
			var progress []int
			options := append(append([]resource.WaitOption(nil), tc.options...), resource.WithProgressCallback(func(value int) { progress = append(progress, value) }))
			target := "active"
			if tc.name == "attribute" {
				target = "KEY"
			}
			got, err := api.WaitFor(ctx, resource.ID("o1"), target, options...)
			var failed *resource.FailedStateError
			switch {
			case tc.timeout:
				if !errors.Is(err, context.DeadlineExceeded) || got != nil {
					t.Fatal(got, err)
				}
				return
			case tc.failed != "":
				if !errors.As(err, &failed) || failed.Status != tc.failed || got != nil {
					t.Fatal(got, err)
				}
			default:
				if err != nil || got == nil || got.OrderRef != "https://kms/v1/orders/o1" {
					t.Fatal(got, err)
				}
			}
			// The progress callback sees 0 after each nonterminal observation.
			if !reflect.DeepEqual(calls, pythonOrderGets(tc.gets)) || len(progress) != tc.gets-1 {
				t.Fatal(calls, progress)
			}
		})
	}
}

// resource.wait_for_delete polls fetch until NotFoundException; the key manager
// proxy default is wait=120 seconds.
func TestPythonOrderWaitForDelete(t *testing.T) {
	var calls []pythonOrderCall
	api := pythonOrderAPI(t, &calls, "ACTIVE", "ACTIVE", "")
	err := api.WaitForDeletion(context.Background(), resource.ID("o1"), resource.WithTimeout(120*time.Second), resource.WithPollInterval(time.Millisecond))
	if err != nil || !reflect.DeepEqual(calls, pythonOrderGets(3)) {
		t.Fatal(err, calls)
	}
}
