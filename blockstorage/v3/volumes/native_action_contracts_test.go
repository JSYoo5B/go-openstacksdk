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

// nativeVolumeAction is one generated action call with its exact wire body and native OkCodes.
type nativeVolumeAction struct {
	name     string
	call     func(context.Context, *volumes.API) error
	body     string
	accepted []int
}

func nativeVolumeActions() []nativeVolumeAction {
	multipath := false
	return []nativeVolumeAction{
		{"Attach", func(ctx context.Context, api *volumes.API) error {
			return api.Attach(ctx, "vol-1", volumes.AttachOpts{InstanceUUID: "srv", MountPoint: "/dev/vdb", Mode: volumes.ReadOnly}, volumes.WithAttachField("x_extension", 1))
		}, `{"os-attach":{"instance_uuid":"srv","mode":"ro","mountpoint":"/dev/vdb","x_extension":1}}`, []int{202}},
		{"BeginDetaching", func(ctx context.Context, api *volumes.API) error { return api.BeginDetaching(ctx, "vol-1") },
			`{"os-begin_detaching":{}}`, []int{202}},
		// An empty attachment ID leaves an empty action object.
		{"Detach", func(ctx context.Context, api *volumes.API) error {
			return api.Detach(ctx, "vol-1", volumes.DetachOpts{})
		},
			`{"os-detach":{}}`, []int{202}},
		{"Reserve", func(ctx context.Context, api *volumes.API) error { return api.Reserve(ctx, "vol-1") },
			`{"os-reserve":{}}`, []int{200, 201, 202}},
		{"Unreserve", func(ctx context.Context, api *volumes.API) error { return api.Unreserve(ctx, "vol-1") },
			`{"os-unreserve":{}}`, []int{200, 201, 202}},
		{"InitializeConnection", func(ctx context.Context, api *volumes.API) error {
			info, err := api.InitializeConnection(ctx, "vol-1", volumes.InitializeConnectionOpts{Host: "compute-1", Initiator: "iqn", Wwpns: []string{"w1"}, Multipath: &multipath}, volumes.WithInitializeConnectionField("x_extension", 1))
			if err == nil && (info["driver_volume_type"] != "iscsi" || info["data"].(map[string]any)["target_lun"] != float64(1)) {
				return fmt.Errorf("connection info %v", info)
			}
			return err
		}, `{"os-initialize_connection":{"connector":{"host":"compute-1","initiator":"iqn","multipath":false,"wwpns":["w1"]},"x_extension":1}}`, []int{200, 201, 202}},
		{"TerminateConnection", func(ctx context.Context, api *volumes.API) error {
			return api.TerminateConnection(ctx, "vol-1", volumes.TerminateConnectionOpts{Host: "compute-1"})
		}, `{"os-terminate_connection":{"connector":{"host":"compute-1"}}}`, []int{202}},
		{"ExtendSize", func(ctx context.Context, api *volumes.API) error {
			return api.ExtendSize(ctx, "vol-1", volumes.ExtendSizeOpts{NewSize: 20})
		}, `{"os-extend":{"new_size":20}}`, []int{202}},
		{"UploadImage", func(ctx context.Context, api *volumes.API) error {
			image, err := api.UploadImage(ctx, "vol-1", volumes.UploadImageOpts{ImageName: "snap", DiskFormat: "qcow2", Force: true})
			if err == nil && (image.ImageID != "img-1" || image.VolumeType.Name != "ssd" || image.UpdatedAt.Second() != 3) {
				return fmt.Errorf("volume image %+v", image)
			}
			return err
		}, `{"os-volume_upload_image":{"disk_format":"qcow2","force":true,"image_name":"snap"}}`, []int{202}},
		// Metadata has no omitempty, so nil is sent as null.
		{"SetImageMetadata", func(ctx context.Context, api *volumes.API) error {
			return api.SetImageMetadata(ctx, "vol-1", volumes.ImageMetadataOpts{})
		}, `{"os-set_image_metadata":{"metadata":null}}`, []int{200}},
		// Bootable false is sent explicitly.
		{"SetBootable", func(ctx context.Context, api *volumes.API) error {
			return api.SetBootable(ctx, "vol-1", volumes.BootableOpts{})
		}, `{"os-set_bootable":{"bootable":false}}`, []int{200}},
		{"ChangeType", func(ctx context.Context, api *volumes.API) error {
			return api.ChangeType(ctx, "vol-1", volumes.ChangeTypeOpts{NewType: "ssd", MigrationPolicy: volumes.MigrationPolicyOnDemand})
		}, `{"os-retype":{"migration_policy":"on-demand","new_type":"ssd"}}`, []int{202}},
		// Both reimage fields are always sent.
		{"ReImage", func(ctx context.Context, api *volumes.API) error {
			return api.ReImage(ctx, "vol-1", volumes.ReImageOpts{})
		},
			`{"os-reimage":{"image_id":"","reimage_reserved":false}}`, []int{202}},
	}
}

func nativeVolumeActionReply(code int) func(*http.Request) *http.Response {
	return func(*http.Request) *http.Response {
		return nativeVolumeWire(code, `{"connection_info":{"driver_volume_type":"iscsi","data":{"target_lun":1}},"os-volume_upload_image":{"id":"vol-1","image_id":"img-1","updated_at":"2026-10-11T01:02:03.000000","volume_type":{"name":"ssd","created_at":"2026-10-11T01:02:03"}}}`)
	}
}

func TestNativeVolumeActionRoutesAndBodies(t *testing.T) {
	ctx := context.Background()
	for _, action := range nativeVolumeActions() {
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
	}
}

func TestNativeVolumeActionStrictStatuses(t *testing.T) {
	ctx := context.Background()
	for _, action := range nativeVolumeActions() {
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
}

func TestNativeVolumeActionPreflightAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeVolumeCall
	api, _ := nativeVolumeAPI(t, &calls, nativeVolumeActionReply(202))
	for name, check := range map[string]struct {
		operation string
		err       error
	}{
		"retype type":           {"ChangeType", api.ChangeType(ctx, "vol-1", volumes.ChangeTypeOpts{})},
		"attach core extension": {"Attach", api.Attach(ctx, "vol-1", volumes.AttachOpts{}, volumes.WithAttachField("mode", "rw"))},
		"detach nil option":     {"Detach", api.Detach(ctx, "vol-1", volumes.DetachOpts{}, nil)},
	} {
		if check.err == nil {
			t.Fatal(name, "accepted")
		}
		nativeVolumeOperation(t, check.err, check.operation)
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
	t.Run("required new size zero is still sent", func(t *testing.T) {
		var calls []nativeVolumeCall
		api, _ := nativeVolumeAPI(t, &calls, nativeVolumeActionReply(202))
		// The native required tag does not reject an integer zero.
		if err := api.ExtendSize(ctx, "vol-1", volumes.ExtendSizeOpts{}); err != nil || calls[0].body != `{"os-extend":{"new_size":0}}` {
			t.Fatal(err, calls)
		}
	})
	t.Run("upload image keeps a zero value for a missing key", func(t *testing.T) {
		var calls []nativeVolumeCall
		api, _ := nativeVolumeAPI(t, &calls, func(*http.Request) *http.Response { return nativeVolumeWire(202, `{"other":{}}`) })
		image, err := api.UploadImage(ctx, "vol-1", volumes.UploadImageOpts{})
		if err != nil || image.ImageID != "" {
			t.Fatal(image, err)
		}
	})
	t.Run("initialize connection without connection info", func(t *testing.T) {
		var calls []nativeVolumeCall
		api, _ := nativeVolumeAPI(t, &calls, func(*http.Request) *http.Response { return nativeVolumeWire(200, `{}`) })
		info, err := api.InitializeConnection(ctx, "vol-1", volumes.InitializeConnectionOpts{})
		if err != nil || info != nil || calls[0].body != `{"os-initialize_connection":{"connector":{}}}` {
			t.Fatal(info, err, calls)
		}
	})
	t.Run("upload image zoned timestamp", func(t *testing.T) {
		var calls []nativeVolumeCall
		api, _ := nativeVolumeAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeVolumeWire(202, `{"os-volume_upload_image":{"updated_at":"2026-10-11T01:02:03Z"}}`)
		})
		_, err := api.UploadImage(ctx, "vol-1", volumes.UploadImageOpts{})
		nativeVolumeOperation(t, err, "UploadImage")
	})
}
