package groups_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestGroupResetStatusRecordPostsAction(t *testing.T) {
	api, calls := groupAPI(t, "", func(r *http.Request) *http.Response {
		if r.URL.Path == "/cinder/v3/" {
			return groupWire(200, `{"version":{"id":"v3.0","version":"3.90","min_version":"3.0"}}`)
		}
		return groupWire(202, ``)
	})
	v, err := api.ResetStatusRecord(context.Background(), "g/1", "available")
	if err != nil || calls.Len() != 2 {
		t.Fatal(err, calls.Calls())
	}
	call := calls.Calls()[1]
	if call.Method != http.MethodPost || call.EscapedPath != groupBase+"/g%2F1/action" || call.Body != `{"reset_status":{"status":"available"}}` || call.Header.Get("OpenStack-API-Version") != "volume 3.38" {
		t.Fatal(call)
	}
	if v.Reference != "g/1" || v.StatusCode != 202 || len(v.Discovery) != 1 || v.Header.Get("X-Request-Id") != "actual" {
		t.Fatalf("receipt: %#v", v)
	}

	for _, code := range []int{200, 204, 399, 400, 404, 409, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			api, calls := groupAPI(t, "3.20", func(*http.Request) *http.Response { return groupWire(code, `{"x":1}`) })
			v, err := api.ResetStatusRecord(context.Background(), "g1", "error", groups.WithRecordResetStatusHeaders(map[string]string{"X-Trace": "t"}))
			if calls.Len() != 1 || calls.Calls()[0].Header.Get("OpenStack-API-Version") != "volume 3.20" || calls.Calls()[0].Header.Get("X-Trace") != "t" {
				t.Fatal(calls.Calls())
			}
			if v == nil || v.StatusCode != code || string(v.Envelope) != `{"x":1}` {
				t.Fatalf("receipt: %#v", v)
			}
			if code < 400 {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var wrapped *resource.OperationError
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &wrapped) || wrapped.Operation != "ResetStatusRecord" || wrapped.Resource != "groups" || !errors.As(err, &native) || native.Actual != code {
				t.Fatal(err)
			}
		})
	}

	api, calls = groupAPI(t, "", func(*http.Request) *http.Response { return groupWire(200, `{}`) })
	if _, err := api.ResetStatusRecord(context.Background(), " ", "available"); !errors.Is(err, resource.ErrInvalidOption) || calls.Len() != 0 {
		t.Fatal(err, calls.Calls())
	}
}
