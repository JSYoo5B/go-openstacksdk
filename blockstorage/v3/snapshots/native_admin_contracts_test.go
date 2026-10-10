package snapshots_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
	"github.com/gophercloud/gophercloud/v2"
)

func TestNativeSnapshotAdminActions(t *testing.T) {
	ctx := context.Background()
	for _, action := range []struct {
		name string
		call func(*snapshots.API) error
		body string
	}{
		{"ForceDelete", func(api *snapshots.API) error { return api.ForceDelete(ctx, "snap-1") }, `{"os-force_delete":{}}`},
		{"ResetStatus", func(api *snapshots.API) error {
			return api.ResetStatus(ctx, "snap-1", snapshots.ResetStatusOpts{Status: "error"}, snapshots.WithResetStatusField("x_extension", 1))
		}, `{"os-reset_status":{"status":"error","x_extension":1}}`},
		{"UpdateStatus", func(api *snapshots.API) error {
			return api.UpdateStatus(ctx, "snap-1", snapshots.UpdateStatusOpts{Status: "available", Progress: "100%"})
		}, `{"os-update_snapshot_status":{"progress":"100%","status":"available"}}`},
		// Status has no omitempty, so an empty value is still sent.
		{"UpdateStatus", func(api *snapshots.API) error {
			return api.UpdateStatus(ctx, "snap-1", snapshots.UpdateStatusOpts{})
		}, `{"os-update_snapshot_status":{"status":""}}`},
	} {
		t.Run(action.name, func(t *testing.T) {
			var calls []nativeSnapCall
			api, _ := nativeSnapAPI(t, &calls, func(*http.Request) *http.Response { return nativeSnapWire(202, "") })
			if err := action.call(api); err != nil {
				t.Fatal(err)
			}
			want := []nativeSnapCall{{http.MethodPost, "/cinder/v3/project/snapshots/snap-1/action", "", action.body}}
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("%+v", calls)
			}
		})
		// All three actions accept only 202.
		for _, code := range []int{200, 201, 204, 404} {
			if slices.Contains([]int{202}, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", action.name, code), func(t *testing.T) {
				var calls []nativeSnapCall
				api, _ := nativeSnapAPI(t, &calls, func(*http.Request) *http.Response { return nativeSnapWire(code, "") })
				err := action.call(api)
				nativeSnapOperation(t, err, action.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{202}) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeSnapCall
		api, _ := nativeSnapAPI(t, &calls, func(*http.Request) *http.Response { return nativeSnapWire(202, "") })
		for operation, err := range map[string]error{
			"ResetStatus":  api.ResetStatus(ctx, "snap-1", snapshots.ResetStatusOpts{}, snapshots.WithResetStatusField("status", "x")),
			"UpdateStatus": api.UpdateStatus(ctx, "snap-1", snapshots.UpdateStatusOpts{}, nil),
		} {
			nativeSnapOperation(t, err, operation)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
