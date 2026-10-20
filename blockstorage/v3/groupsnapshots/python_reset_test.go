package groupsnapshots_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestGroupSnapshotResetStatusRecordPostsActionWithEmptyAccept(t *testing.T) {
	api, calls := snapshotAPI(t, "", func(r *http.Request) *http.Response {
		if r.URL.Path == "/cinder/v3/" {
			return snapshotWire(200, `{"version":{"id":"v3.0","version":"3.90","min_version":"3.0"}}`)
		}
		return snapshotWire(202, ``)
	})
	v, err := api.ResetStatusRecord(context.Background(), "s1", "available", groupsnapshots.WithRecordResetStatusHeaders(map[string]string{"Accept": "text/plain"}))
	if err != nil || len(*calls) != 2 {
		t.Fatal(err, *calls)
	}
	call := (*calls)[1]
	if call.method != http.MethodPost || call.escaped != snapshotBase+"/s1/action" || call.body != `{"reset_status":{"status":"available"}}` || call.headers.Get("OpenStack-API-Version") != "volume 3.29" {
		t.Fatal(call)
	}
	if accept := call.headers.Values("Accept"); len(accept) != 1 || accept[0] != "" {
		t.Fatalf("Accept: %q", accept)
	}
	if v.Reference != "s1" || v.StatusCode != 202 || len(v.Discovery) != 1 {
		t.Fatalf("receipt: %#v", v)
	}

	for _, code := range []int{200, 204, 400, 404, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			api, calls := snapshotAPI(t, "3.14", func(*http.Request) *http.Response { return snapshotWire(code, `{"x":1}`) })
			v, err := api.ResetStatusRecord(context.Background(), "s1", "error")
			if len(*calls) != 1 || (*calls)[0].headers.Get("OpenStack-API-Version") != "volume 3.14" {
				t.Fatal(*calls)
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
			if !errors.As(err, &wrapped) || wrapped.Operation != "ResetStatusRecord" || wrapped.Resource != "group_snapshots" || !errors.As(err, &native) || native.Actual != code {
				t.Fatal(err)
			}
		})
	}
}
