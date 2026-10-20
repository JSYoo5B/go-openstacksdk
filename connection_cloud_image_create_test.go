package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type cloudCreateRoute struct {
	route string
	code  int
	body  string
}

// cloudCreateConnection replays one exact route list across Glance and Cinder.
func cloudCreateConnection(t *testing.T, routes []cloudCreateRoute, calls *int) *sdk.Connection {
	t.Helper()
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("token")
	provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
		index := *calls
		*calls++
		route := req.Method + " " + req.URL.String()
		if index >= len(routes) || routes[index].route != route {
			t.Fatal("unexpected request", index, route)
		}
		if req.Body != nil {
			io.Copy(io.Discard, req.Body)
		}
		return &http.Response{Request: req, StatusCode: routes[index].code, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(routes[index].body))}, nil
	})
	conn, err := sdk.FromProvider(provider,
		sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/glance/v2/"),
		sdk.WithEndpointFor(sdk.BlockStorage, "v3", "https://cloud.test/cinder/v3/"),
		sdk.WithMicroversion(sdk.BlockStorage, "3.90"),
		sdk.WithImageCreatePolicy(image.WithImageCreatePolicyFormat("raw"), image.WithImageCreatePolicyVendorAgent(map[string]any{})))
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

const (
	cloudCreatePost   = "POST https://cloud.test/glance/v2/images"
	cloudCreateGet    = "GET https://cloud.test/glance/v2/images/new"
	cloudCreateDelete = "DELETE https://cloud.test/glance/v2/images/new"
	cloudCreateName   = "GET https://cloud.test/glance/v2/images?name=new"
	cloudCreateHidden = "GET https://cloud.test/glance/v2/images?os_hidden=True"
	cloudCreateVolume = "POST https://cloud.test/cinder/v3/volumes/vol/action"
)

func TestConnectionCloudImageCreateBranchesAndWait(t *testing.T) {
	duplicates := image.WithImageRecordCreateAllowDuplicates(true)
	request := sdk.CloudImageCreateRequest{Image: image.ImageRecordCreateRequest{Name: "image"}, PollInterval: time.Millisecond}
	t.Run("glance branch without wait returns created record", func(t *testing.T) {
		calls := 0
		conn := cloudCreateConnection(t, []cloudCreateRoute{{cloudCreatePost, 201, `{"id":"new","status":"queued"}`}}, &calls)
		result, err := conn.CreateCloudImageRecord(context.Background(), request, duplicates)
		if err != nil || calls != 1 || result.Created == nil || result.Image != result.Created.Record || result.WaitLookups != 0 || result.Volume != nil {
			t.Fatal(result, err)
		}
	})
	t.Run("wait polls until status leaves queued and saving", func(t *testing.T) {
		calls := 0
		conn := cloudCreateConnection(t, []cloudCreateRoute{
			{cloudCreatePost, 201, `{"id":"new","status":"queued"}`},
			{cloudCreateGet, 404, `{}`}, {cloudCreateName, 200, `{"images":[]}`}, {cloudCreateHidden, 200, `{"images":[]}`},
			{cloudCreateGet, 200, `{"id":"new","status":"saving"}`},
			{cloudCreateGet, 200, `{"id":"new","status":"killed"}`},
		}, &calls)
		result, err := conn.CreateCloudImageRecord(context.Background(), request, duplicates, image.WithImageRecordCreateWait(true))
		if err != nil || calls != 6 || result.WaitLookups != 3 || result.Image == nil || result.Image != result.WaitLast || string(result.Image.Resource.Body["status"]) != `"killed"` {
			t.Fatal(result, err, calls)
		}
	})
	t.Run("null status is not queued or saving", func(t *testing.T) {
		calls := 0
		conn := cloudCreateConnection(t, []cloudCreateRoute{{cloudCreatePost, 201, `{"id":"new","status":"queued"}`}, {cloudCreateGet, 200, `{"id":"new","status":null}`}}, &calls)
		result, err := conn.CreateCloudImageRecord(context.Background(), request, duplicates, image.WithImageRecordCreateWait("yes"))
		if err != nil || calls != 2 || result.WaitLookups != 1 {
			t.Fatal(result, err)
		}
	})
	t.Run("timeout deletes with wait and keeps both phases", func(t *testing.T) {
		calls := 0
		// The router stops answering queued once the timeout cleanup deletes.
		deleted := false
		provider := &gophercloud.ProviderClient{}
		provider.UseTokenLock()
		provider.SetToken("token")
		provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
			calls++
			route := req.Method + " " + req.URL.String()
			reply := func(code int, body string) (*http.Response, error) {
				return &http.Response{Request: req, StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
			}
			switch {
			case route == cloudCreatePost:
				return reply(201, `{"id":"new","status":"queued"}`)
			case route == cloudCreateDelete:
				deleted = true
				return reply(204, "")
			case route == cloudCreateGet && !deleted:
				return reply(200, `{"id":"new","status":"queued"}`)
			case route == cloudCreateGet:
				return reply(404, `{}`)
			case route == cloudCreateName || route == cloudCreateHidden:
				return reply(200, `{"images":[]}`)
			}
			t.Fatal("unexpected request", route)
			return nil, nil
		})
		conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/glance/v2/"), sdk.WithImageCreatePolicy(image.WithImageCreatePolicyVendorAgent(map[string]any{})))
		if err != nil {
			t.Fatal(err)
		}
		result, err := conn.CreateCloudImageRecord(context.Background(), request, duplicates, image.WithImageRecordCreateWait(true), image.WithImageRecordCreateTimeout(0.03))
		if !errors.Is(err, context.DeadlineExceeded) || result.Cleanup == nil || !result.Cleanup.Deleted || result.Cleanup.WaitLookups != 1 || result.WaitLookups < 1 || result.Image != result.Created.Record || !deleted {
			t.Fatal(result, err)
		}
	})
	t.Run("volume branch maps create options to the Cinder Proxy", func(t *testing.T) {
		calls := 0
		conn := cloudCreateConnection(t, []cloudCreateRoute{{cloudCreateVolume, 202, `{"os-volume_upload_image":{"image_id":"new"}}`}, {cloudCreateGet, 200, `{"id":"new","status":"active"}`}}, &calls)
		volume := request
		volume.VolumeID = "vol"
		result, err := conn.CreateCloudImageRecord(context.Background(), volume, duplicates, image.WithImageRecordCreateDiskFormat("vmdk"), image.WithImageRecordCreateContainerFormat(""), image.WithImageRecordCreateWait(true))
		if err != nil || calls != 2 || result.Volume == nil || result.Created != nil || result.WaitLookups != 1 || string(result.Image.Resource.Body["status"]) != `"active"` {
			t.Fatal(result, err, calls)
		}
		if body := string(result.Volume.Upload.Applied.Body); !strings.Contains(body, "new") {
			t.Fatal(body)
		}
	})
}

func TestConnectionCloudImageCreateVolumeBodyAndPreflight(t *testing.T) {
	var body map[string]map[string]json.RawMessage
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("token")
	calls := 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method+" "+req.URL.String() != cloudCreateVolume {
			t.Fatal(req.URL)
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		return &http.Response{Request: req, StatusCode: 202, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"os-volume_upload_image":{"image_id":"new"}}`))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/glance/v2/"), sdk.WithEndpointFor(sdk.BlockStorage, "v3", "https://cloud.test/cinder/v3/"), sdk.WithMicroversion(sdk.BlockStorage, "3.90"))
	if err != nil {
		t.Fatal(err)
	}
	request := sdk.CloudImageCreateRequest{Image: image.ImageRecordCreateRequest{Name: "image"}, VolumeID: "vol"}
	result, err := conn.CreateCloudImageRecord(context.Background(), request, image.WithImageRecordCreateAllowDuplicates(1), image.WithImageRecordCreateDiskFormat(nil))
	upload := body["os-volume_upload_image"]
	if err != nil || calls != 1 || string(upload["force"]) != "true" || string(upload["disk_format"]) != `"qcow2"` || string(upload["container_format"]) != `"bare"` || string(upload["image_name"]) != `"image"` || result.Image == nil {
		t.Fatal(result, err, upload)
	}
	for name, options := range map[string][]image.ImageRecordCreateOption{
		"non-string disk format":   {image.WithImageRecordCreateDiskFormat(7)},
		"non-numeric wait timeout": {image.WithImageRecordCreateWait(true), image.WithImageRecordCreateTimeout("soon")},
		"nil option":               {nil},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := conn.CreateCloudImageRecord(context.Background(), request, options...); !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
				t.Fatal(err, calls)
			}
		})
	}
	if _, err := conn.CreateCloudImageRecord(context.Background(), sdk.CloudImageCreateRequest{PollInterval: -1}); !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
		t.Fatal(err)
	}
}
