package volumes_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2"
)

// nativeVolumeAdminActions are the generated actions whose default Cinder policy is admin-only.
func nativeVolumeAdminActions() []nativeVolumeAction {
	return []nativeVolumeAction{
		// ForceDelete sends an empty string, not an empty object, and uses the POST default codes.
		{"ForceDelete", func(ctx context.Context, api *volumes.API) error { return api.ForceDelete(ctx, "vol-1") },
			`{"os-force_delete":""}`, []int{201, 202}},
		{"ResetStatus", func(ctx context.Context, api *volumes.API) error {
			return api.ResetStatus(ctx, "vol-1", volumes.ResetStatusOpts{Status: "available", AttachStatus: "detached"}, volumes.WithResetStatusField("x_extension", 1))
		}, `{"os-reset_status":{"attach_status":"detached","status":"available","x_extension":1}}`, []int{202}},
		{"Unmanage", func(ctx context.Context, api *volumes.API) error { return api.Unmanage(ctx, "vol-1") },
			`{"os-unmanage":{}}`, []int{202}},
	}
}

func TestNativeVolumeAdminActionRoutesBodiesAndStatuses(t *testing.T) {
	ctx := context.Background()
	for _, action := range nativeVolumeAdminActions() {
		t.Run(action.name, func(t *testing.T) {
			var calls []nativeVolumeCall
			api, _ := nativeVolumeAPI(t, &calls, nativeVolumeActionReply(action.accepted[len(action.accepted)-1]))
			if err := action.call(ctx, api); err != nil {
				t.Fatal(err)
			}
			want := []nativeVolumeCall{{http.MethodPost, "/cinder/v3/project/volumes/vol-1/action", "", action.body}}
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("%+v", calls)
			}
		})
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(action.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", action.name, code), func(t *testing.T) {
				var calls []nativeVolumeCall
				api, _ := nativeVolumeAPI(t, &calls, nativeVolumeActionReply(code))
				err := action.call(ctx, api)
				nativeVolumeOperation(t, err, action.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, action.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("reset status always sends status", func(t *testing.T) {
		var calls []nativeVolumeCall
		api, _ := nativeVolumeAPI(t, &calls, nativeVolumeActionReply(202))
		if err := api.ResetStatus(ctx, "vol-1", volumes.ResetStatusOpts{}); err != nil || calls[0].body != `{"os-reset_status":{"status":""}}` {
			t.Fatal(err, calls)
		}
		err := api.ResetStatus(ctx, "vol-1", volumes.ResetStatusOpts{}, volumes.WithResetStatusField("status", "x"))
		nativeVolumeOperation(t, err, "ResetStatus")
		if len(calls) != 1 {
			t.Fatal(calls)
		}
	})
}
