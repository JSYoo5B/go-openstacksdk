package openstack_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func volumeImageCreateConnection(t *testing.T, reply string, body *string, posts *atomic.Int32, options ...sdk.ConnectionOption) *sdk.Connection {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		if opts.Type == "image" {
			return cloud.Server.URL + "/glance/v2/", nil
		}
		return cloud.Server.URL + vcmBase, nil
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		posts.Add(1)
		vcmWire(t, req, *body, "3.90")
		testcloud.JSON(w, 202, reply)
	})
	name := "volume-cloud"
	options = append([]sdk.ConnectionOption{sdk.WithMicroversion(sdk.BlockStorage, "3.90"), sdk.WithCloudLocation(resource.CloudLocation{Cloud: &name})}, options...)
	conn, err := sdk.FromProvider(cloud.Provider, options...)
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func TestConnectionCreateVolumeImageRecordDefaultsUploadAndExistingImage(t *testing.T) {
	for _, test := range []struct {
		name    string
		input   sdk.VolumeImageCreateRequest
		options []sdk.ConnectionOption
		body    string
	}{
		{"cloud qcow2 default and bare", sdk.VolumeImageCreateRequest{Name: "img", VolumeID: "literal-name"}, nil,
			`{"os-volume_upload_image":{"container_format":"bare","disk_format":"qcow2","force":false,"image_name":"img"}}`},
		{"configured format", sdk.VolumeImageCreateRequest{Name: "img", VolumeID: "literal-name"}, []sdk.ConnectionOption{sdk.WithImageCreatePolicy(image.WithImageCreatePolicyFormat("raw"))},
			`{"os-volume_upload_image":{"container_format":"bare","disk_format":"raw","force":false,"image_name":"img"}}`},
		{"configured null omits disk format", sdk.VolumeImageCreateRequest{Name: "img", VolumeID: "literal-name"}, []sdk.ConnectionOption{sdk.WithImageCreatePolicy(image.WithImageCreatePolicyFormat(nil))},
			`{"os-volume_upload_image":{"container_format":"bare","force":false,"image_name":"img"}}`},
		{"explicit formats and duplicates", sdk.VolumeImageCreateRequest{Name: "img", VolumeID: "literal-name", AllowDuplicates: true, DiskFormat: "vmdk", ContainerFormat: "ovf"}, []sdk.ConnectionOption{sdk.WithImageCreatePolicy(image.WithImageCreatePolicyFormat("raw"))},
			`{"os-volume_upload_image":{"container_format":"ovf","disk_format":"vmdk","force":true,"image_name":"img"}}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var posts atomic.Int32
			body := test.body
			conn := volumeImageCreateConnection(t, `{"os-volume_upload_image":{"image_id":"new-image","status":"uploading"}}`, &body, &posts, test.options...)
			result, err := conn.CreateVolumeImageRecord(vcmContext(t), test.input)
			if err != nil || posts.Load() != 1 || result.Upload == nil || !result.Upload.Completed || result.Image == nil {
				t.Fatal(result, err, posts.Load())
			}
			// Image.existing builds a synchronized id-only record without Glance HTTP.
			fields := result.Image.Resource.Body
			var location resource.CloudLocation
			if string(fields["id"]) != `"new-image"` || string(fields["status"]) != "null" || len(fields) != 65 || json.Unmarshal(fields["location"], &location) != nil || location.Cloud == nil || *location.Cloud != "volume-cloud" {
				t.Fatal(fields)
			}
		})
	}
}

func TestConnectionCreateVolumeImageRecordFailuresKeepUploadEvidence(t *testing.T) {
	const body = `{"os-volume_upload_image":{"container_format":"bare","disk_format":"qcow2","force":false,"image_name":"img"}}`
	for _, reply := range []string{
		`{"os-volume_upload_image":{"status":"uploading"}}`,
		`{"os-volume_upload_image":{"image_id":null}}`,
		`{"os-volume_upload_image":{"image_id":7}}`,
		`{"os-volume_upload_image":{"image_id":" "}}`,
		`{"os-volume_upload_image":["image_id"]}`,
	} {
		t.Run(reply, func(t *testing.T) {
			var posts atomic.Int32
			wire := body
			conn := volumeImageCreateConnection(t, reply, &wire, &posts)
			result, err := conn.CreateVolumeImageRecord(vcmContext(t), sdk.VolumeImageCreateRequest{Name: "img", VolumeID: "literal-name"})
			var operation *resource.OperationError
			if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "CreateVolumeImageRecord" || result == nil || result.Upload == nil || result.Image != nil || posts.Load() != 1 {
				t.Fatal(result, err)
			}
		})
	}
	t.Run("non-string configured format fails before HTTP", func(t *testing.T) {
		var posts atomic.Int32
		wire := body
		conn := volumeImageCreateConnection(t, `{}`, &wire, &posts, sdk.WithImageCreatePolicy(image.WithImageCreatePolicyFormat(7)))
		result, err := conn.CreateVolumeImageRecord(vcmContext(t), sdk.VolumeImageCreateRequest{Name: "img", VolumeID: "literal-name"})
		if !errors.Is(err, resource.ErrInvalidOption) || result != nil || posts.Load() != 0 {
			t.Fatal(result, err)
		}
	})
	t.Run("invalid volume ID fails before HTTP", func(t *testing.T) {
		var posts atomic.Int32
		wire := body
		conn := volumeImageCreateConnection(t, `{}`, &wire, &posts)
		if _, err := conn.CreateVolumeImageRecord(vcmContext(t), sdk.VolumeImageCreateRequest{Name: "img"}); err == nil || posts.Load() != 0 {
			t.Fatal(err)
		}
	})
}
