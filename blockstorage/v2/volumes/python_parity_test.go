package volumes_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pythonVolumeCall struct {
	method, path string
	query        url.Values
	body         map[string]any
	header       http.Header
}

func pythonVolumeWire(code int, body string) *http.Response {
	return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}})
}

func pythonVolumeAPI(t *testing.T, calls *[]pythonVolumeCall, reply func(*http.Request) *http.Response) (*volumes.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		call := pythonVolumeCall{method: req.Method, path: req.URL.Path, query: req.URL.Query(), header: req.Header.Clone()}
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			if len(data) != 0 {
				if err := json.Unmarshal(data, &call.body); err != nil {
					t.Errorf("request body %q: %v", data, err)
				}
			}
		}
		*calls = append(*calls, call)
		return reply(req), nil
	}))
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v2/project/"
	return volumes.New(client), cloud
}

func pythonVolumeJSON(t *testing.T, raw string) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		t.Fatal(raw, err)
	}
	return value
}

func pythonVolumeExpect(t *testing.T, call pythonVolumeCall, method, path, query, body string) {
	t.Helper()
	if call.method != method || call.path != path {
		t.Fatalf("got %s %s, want %s %s", call.method, call.path, method, path)
	}
	wantQuery, _ := url.ParseQuery(query)
	if len(call.query) != 0 || len(wantQuery) != 0 {
		if !reflect.DeepEqual(call.query, wantQuery) {
			t.Fatalf("query %v, want %v", call.query, wantQuery)
		}
	}
	if body == "" {
		if call.body != nil {
			t.Fatalf("unexpected body %v", call.body)
		}
		return
	}
	if want := pythonVolumeJSON(t, body); !reflect.DeepEqual(call.body, want) {
		t.Fatalf("body %v, want %v", call.body, want)
	}
	// Python v2 sends no microversion; the v2 client has none configured.
	if call.header.Get("OpenStack-API-Version") != "" || call.header.Get("X-OpenStack-Volume-API-Version") != "" {
		t.Fatalf("unexpected microversion header %v", call.header)
	}
}

const pythonVolumeRow = `{"id":"vol-1","name":"data","status":"available","size":10,"bootable":"true","encrypted":false,"multiattach":true,"availability_zone":"nova","volume_type":"ssd","snapshot_id":"snap-1","source_volid":"","user_id":"user","consistencygroup_id":"cg","replication_status":"disabled","metadata":{"k":"v"},"attachments":[],"os-vol-host-attr:host":"host@lvm","os-vol-tenant-attr:tenant_id":"project","volume_image_metadata":{"image_id":"img"},"created_at":"2026-10-11T01:02:03.000000"}`

func TestPythonVolumeGetUpdateMatchProxyRequests(t *testing.T) {
	ctx := context.Background()
	var calls []pythonVolumeCall
	api, _ := pythonVolumeAPI(t, &calls, func(*http.Request) *http.Response {
		return pythonVolumeWire(200, `{"volume":`+pythonVolumeRow+`}`)
	})
	got, err := api.Get(ctx, "vol-1")
	if err != nil {
		t.Fatal(err)
	}
	pythonVolumeExpect(t, calls[0], http.MethodGet, "/cinder/v2/project/volumes/vol-1", "", "")
	if !(got.ID == "vol-1" && got.Name == "data" && got.Size == 10 && got.Bootable == "true" && got.Multiattach && got.Host == "host@lvm" && got.TenantID == "project" && got.ConsistencyGroupID == "cg" && got.Metadata["k"] == "v" && got.CreatedAt.Second() == 3) {
		t.Fatalf("%+v", got)
	}

	name, description := "renamed", ""
	updated, err := api.Update(ctx, "vol-1", volumes.UpdateOpts{Name: &name, Description: &description}, volumes.WithUpdateField("x_extension", true))
	if err != nil || updated.ID != "vol-1" {
		t.Fatal(updated, err)
	}
	pythonVolumeExpect(t, calls[1], http.MethodPut, "/cinder/v2/project/volumes/vol-1", "", `{"volume":{"name":"renamed","description":"","x_extension":true}}`)

	// Python skips commit without dirty attributes; Go still sends PUT.
	if _, err := api.Update(ctx, "vol-1", volumes.UpdateOpts{}); err != nil {
		t.Fatal(err)
	}
	pythonVolumeExpect(t, calls[2], http.MethodPut, "/cinder/v2/project/volumes/vol-1", "", `{"volume":{}}`)
}

func TestPythonVolumeCreateEnvelopeAlwaysSendsSize(t *testing.T) {
	ctx := context.Background()
	var calls []pythonVolumeCall
	api, _ := pythonVolumeAPI(t, &calls, func(*http.Request) *http.Response {
		return pythonVolumeWire(202, `{"volume":`+pythonVolumeRow+`}`)
	})
	hints := map[string]any{"same_host": []string{"vol-0"}}
	created, err := api.Create(ctx, volumes.CreateOpts{Size: 10, Name: "data", SnapshotID: "snap-1", ImageID: "img"},
		volumes.WithCreateField("multiattach", true), volumes.WithCreateField("OS-SCH-HNT:scheduler_hints", hints))
	if err != nil || created.ID != "vol-1" {
		t.Fatal(created, err)
	}
	// Python v2 Volume keeps scheduler_hints as a Body attribute inside the envelope.
	pythonVolumeExpect(t, calls[0], http.MethodPost, "/cinder/v2/project/volumes", "", `{"volume":{"size":10,"name":"data","snapshot_id":"snap-1","imageRef":"img","multiattach":true,"OS-SCH-HNT:scheduler_hints":{"same_host":["vol-0"]}}}`)

	// Python create_volume(snapshot_id=...) omits size; Go always sends it.
	if _, err := api.Create(ctx, volumes.CreateOpts{SnapshotID: "snap-1"}); err != nil {
		t.Fatal(err)
	}
	pythonVolumeExpect(t, calls[1], http.MethodPost, "/cinder/v2/project/volumes", "", `{"volume":{"size":0,"snapshot_id":"snap-1"}}`)
	if _, err := api.Create(ctx, volumes.CreateOpts{SnapshotID: "snap-1"}, volumes.WithCreateField("size", nil)); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 2 {
		t.Fatal("size override must be rejected before HTTP", err, len(calls))
	}
}

func TestPythonVolumeDeleteDefaultsCascadeForceAndMissing(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name   string
		status int
		ignore bool
	}{{"deleted", 202, true}, {"no content", 204, true}, {"missing", 404, true}, {"conflict", 409, false}} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonVolumeCall
			api, _ := pythonVolumeAPI(t, &calls, func(*http.Request) *http.Response { return pythonVolumeWire(tc.status, `{}`) })

			err := resource.IgnoreMissing(api.Delete(ctx, "vol-1", volumes.WithDeleteQuery("cascade", "False")))
			if (err == nil) != tc.ignore {
				t.Fatal(err)
			}
			pythonVolumeExpect(t, calls[0], http.MethodDelete, "/cinder/v2/project/volumes/vol-1", "cascade=False", "")

			err = api.Remove(ctx, resource.ID("vol-1"))
			if (err == nil) != tc.ignore {
				t.Fatal(err)
			}
			pythonVolumeExpect(t, calls[1], http.MethodDelete, "/cinder/v2/project/volumes/vol-1", "", "")

			err = api.Remove(ctx, resource.ID("vol-1"), resource.WithMissingError())
			if tc.status == 404 && !errors.Is(err, resource.ErrNotFound) || tc.status < 300 && err != nil {
				t.Fatal(err)
			}
		})
	}

	var calls []pythonVolumeCall
	api, _ := pythonVolumeAPI(t, &calls, func(*http.Request) *http.Response { return pythonVolumeWire(202, ``) })
	if err := api.Delete(ctx, "vol-1", volumes.WithDeleteQuery("cascade", "True")); err != nil {
		t.Fatal(err)
	}
	pythonVolumeExpect(t, calls[0], http.MethodDelete, "/cinder/v2/project/volumes/vol-1", "cascade=True", "")
	if err := api.ForceDelete(ctx, "vol-1"); err != nil {
		t.Fatal(err)
	}
	// Python force=True posts {"os-force_delete": null}.
	pythonVolumeExpect(t, calls[1], http.MethodPost, "/cinder/v2/project/volumes/vol-1/action", "", `{"os-force_delete":""}`)
}

func TestPythonVolumeListDetailDefaultOnly(t *testing.T) {
	ctx := context.Background()
	var calls []pythonVolumeCall
	var cloud *testcloud.Cloud
	api, cloud := pythonVolumeAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Path == "/next/volumes/detail" {
			return pythonVolumeWire(200, `{"volumes":[{"id":"vol-2","name":"data","status":"available"}]}`)
		}
		return pythonVolumeWire(200, `{"volumes":[`+pythonVolumeRow+`],"volumes_links":[{"rel":"next","href":"`+cloud.Server.URL+`/next/volumes/detail?marker=vol-1"}]}`)
	})
	var ids []string
	for value, err := range api.List(ctx, volumes.WithListQuery("all_tenants", "True"), volumes.WithListQuery("name", "data"), volumes.WithListQuery("status", "available"), volumes.WithListQuery("project_id", "project")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"vol-1", "vol-2"}) || len(calls) != 2 {
		t.Fatal(ids, calls)
	}
	// Python volumes(all_projects=True, name=..., status=..., project_id=...) with default details=True.
	pythonVolumeExpect(t, calls[0], http.MethodGet, "/cinder/v2/project/volumes/detail", "all_tenants=True&name=data&status=available&project_id=project", "")
	pythonVolumeExpect(t, calls[1], http.MethodGet, "/next/volumes/detail", "marker=vol-1", "")
}

func TestPythonVolumeFindChainUsesDetailNameQuery(t *testing.T) {
	ctx := context.Background()
	var calls []pythonVolumeCall
	api, _ := pythonVolumeAPI(t, &calls, func(req *http.Request) *http.Response {
		switch req.URL.Path {
		case "/cinder/v2/project/volumes/data":
			return pythonVolumeWire(404, `{"itemNotFound":{"code":404}}`)
		case "/cinder/v2/project/volumes/denied":
			return pythonVolumeWire(403, `{"forbidden":{"code":403}}`)
		}
		return pythonVolumeWire(200, `{"volumes":[`+pythonVolumeRow+`]}`)
	})
	byID, err := api.Find(ctx, resource.ID("data"), resource.WithIgnoreMissing())
	if err != nil || byID != nil {
		t.Fatal(byID, err)
	}
	byName, err := api.Find(ctx, resource.Name("data"))
	if err != nil || byName == nil || byName.ID != "vol-1" {
		t.Fatal(byName, err)
	}
	pythonVolumeExpect(t, calls[0], http.MethodGet, "/cinder/v2/project/volumes/data", "", "")
	pythonVolumeExpect(t, calls[1], http.MethodGet, "/cinder/v2/project/volumes/detail", "name=data", "")

	// Python also falls back after 403/400; the ID lookup reports them instead.
	if _, err := api.Find(ctx, resource.ID("denied"), resource.WithIgnoreMissing()); err == nil {
		t.Fatal("403 must not be ignored")
	}
}

func TestPythonVolumeActionBodiesMatchProxyRequests(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name, body string
		status     int
		call       func(*volumes.API) error
	}{
		{"extend_volume", `{"os-extend":{"new_size":20}}`, 202, func(api *volumes.API) error {
			return api.ExtendSize(ctx, "vol-1", volumes.ExtendSizeOpts{NewSize: 20})
		}},
		{"retype_volume default never", `{"os-retype":{"new_type":"ssd","migration_policy":"never"}}`, 202, func(api *volumes.API) error {
			return api.ChangeType(ctx, "vol-1", volumes.ChangeTypeOpts{NewType: "ssd", MigrationPolicy: volumes.MigrationPolicyNever})
		}},
		{"retype_volume on-demand", `{"os-retype":{"new_type":"ssd","migration_policy":"on-demand"}}`, 202, func(api *volumes.API) error {
			return api.ChangeType(ctx, "vol-1", volumes.ChangeTypeOpts{NewType: "ssd", MigrationPolicy: volumes.MigrationPolicyOnDemand})
		}},
		{"set_volume_bootable_status false", `{"os-set_bootable":{"bootable":false}}`, 200, func(api *volumes.API) error {
			return api.SetBootable(ctx, "vol-1", volumes.BootableOpts{Bootable: false})
		}},
		{"set_volume_image_metadata", `{"os-set_image_metadata":{"metadata":{"os_type":"linux"}}}`, 200, func(api *volumes.API) error {
			return api.SetImageMetadata(ctx, "vol-1", volumes.ImageMetadataOpts{Metadata: map[string]string{"os_type": "linux"}})
		}},
		{"set_volume_image_metadata empty", `{"os-set_image_metadata":{"metadata":{}}}`, 200, func(api *volumes.API) error {
			return api.SetImageMetadata(ctx, "vol-1", volumes.ImageMetadataOpts{Metadata: map[string]string{}})
		}},
		{"attach_volume instance", `{"os-attach":{"mountpoint":"/dev/vdb","instance_uuid":"srv"}}`, 202, func(api *volumes.API) error {
			return api.Attach(ctx, "vol-1", volumes.AttachOpts{MountPoint: "/dev/vdb", InstanceUUID: "srv"})
		}},
		{"attach_volume host", `{"os-attach":{"mountpoint":"/dev/vdb","host_name":"node"}}`, 202, func(api *volumes.API) error {
			return api.Attach(ctx, "vol-1", volumes.AttachOpts{MountPoint: "/dev/vdb", HostName: "node"})
		}},
		{"detach_volume", `{"os-detach":{"attachment_id":"att"}}`, 202, func(api *volumes.API) error {
			return api.Detach(ctx, "vol-1", volumes.DetachOpts{AttachmentID: "att"})
		}},
		{"reset_volume_status all", `{"os-reset_status":{"status":"available","attach_status":"detached","migration_status":"none"}}`, 202, func(api *volumes.API) error {
			return api.ResetStatus(ctx, "vol-1", volumes.ResetStatusOpts{Status: "available", AttachStatus: "detached", MigrationStatus: "none"})
		}},
		// Python omits a None status; Go always sends the status key.
		{"reset_volume_status without status", `{"os-reset_status":{"status":"","attach_status":"detached"}}`, 202, func(api *volumes.API) error {
			return api.ResetStatus(ctx, "vol-1", volumes.ResetStatusOpts{AttachStatus: "detached"})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonVolumeCall
			api, _ := pythonVolumeAPI(t, &calls, func(*http.Request) *http.Response {
				return pythonVolumeWire(tc.status, `{"metadata":{"os_type":"linux"}}`)
			})
			if err := tc.call(api); err != nil {
				t.Fatal(err)
			}
			if len(calls) != 1 {
				t.Fatal(calls)
			}
			pythonVolumeExpect(t, calls[0], http.MethodPost, "/cinder/v2/project/volumes/vol-1/action", "", tc.body)
		})
	}

	// Python v2 Volume._action never raises on HTTP errors; Go returns them.
	var calls []pythonVolumeCall
	api, _ := pythonVolumeAPI(t, &calls, func(*http.Request) *http.Response { return pythonVolumeWire(400, `{"badRequest":{}}`) })
	if err := api.ExtendSize(ctx, "vol-1", volumes.ExtendSizeOpts{NewSize: 5}); err == nil {
		t.Fatal("400 must be reported")
	}
}

func TestPythonVolumeUploadAndCreateImage(t *testing.T) {
	ctx := context.Background()
	var calls []pythonVolumeCall
	api, cloud := pythonVolumeAPI(t, &calls, func(*http.Request) *http.Response {
		return pythonVolumeWire(202, `{"os-volume_upload_image":{"id":"vol-1","image_id":"img-1","image_name":"backup","disk_format":"qcow2","container_format":"bare","status":"uploading","size":10}}`)
	})

	// upload_volume_to_image(volume, "backup", force=True, disk_format="raw", container_format="ovf")
	uploaded, err := api.UploadImage(ctx, "vol-1", volumes.UploadImageOpts{ImageName: "backup", Force: true, DiskFormat: "raw", ContainerFormat: "ovf"})
	if err != nil || uploaded.ImageID != "img-1" || uploaded.VolumeID != "vol-1" || uploaded.Status != "uploading" {
		t.Fatal(uploaded, err)
	}
	pythonVolumeExpect(t, calls[0], http.MethodPost, "/cinder/v2/project/volumes/vol-1/action", "", `{"os-volume_upload_image":{"image_name":"backup","force":true,"disk_format":"raw","container_format":"ovf"}}`)

	// create_image(name, volume, allow_duplicates=False, None, None) uses image_format qcow2 and bare.
	uploaded, err = api.UploadImage(ctx, "vol-1", volumes.UploadImageOpts{ImageName: "backup", Force: false, DiskFormat: "qcow2", ContainerFormat: "bare"})
	if err != nil {
		t.Fatal(err)
	}
	pythonVolumeExpect(t, calls[1], http.MethodPost, "/cinder/v2/project/volumes/vol-1/action", "", `{"os-volume_upload_image":{"image_name":"backup","disk_format":"qcow2","container_format":"bare"}}`)
	glance := image.New(cloud.Client("image", "/image/v2/"))
	record, err := glance.ExistingImageRecord(ctx, uploaded.ImageID)
	if err != nil || record == nil || record.Resource == nil {
		t.Fatal(record, err)
	}
	var view struct {
		ID string `json:"id"`
	}
	if err := record.Resource.Decode(&view); err != nil || view.ID != "img-1" {
		t.Fatal(view, err)
	}
	if len(calls) != 2 {
		t.Fatal("Image.existing must not send HTTP", calls)
	}
}
