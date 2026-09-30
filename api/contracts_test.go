package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"gophercloudsdk/compute/v2/servers"
	"gophercloudsdk/image/v2/images"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/network/v2/ports"
	"gophercloudsdk/objectstorage/v1/objects"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

func TestPortExtensionsTypedPaginationAndErrors(t *testing.T) {
	cloud := testcloud.New(t)
	posts, next := 0, 0
	cloud.Mux.HandleFunc("/network/v2.0/ports", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			posts++
			var body struct {
				Port map[string]any `json:"port"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body.Port["network_id"] != "net-id" || body.Port["binding:host_id"] != "node" || body.Port["vendor:flag"] != false {
				t.Errorf("body=%v", body)
			}
			testcloud.JSON(w, 201, `{"port":{"id":"port-id","name":"port"}}`)
		case http.MethodGet:
			if r.URL.Query().Get("limit") != "5" || r.URL.Query().Get("vendor:filter") != "a&b" {
				t.Errorf("query=%v", r.URL.Query())
			}
			testcloud.JSON(w, 200, `{"ports":[{"id":"one"}],"ports_links":[{"rel":"next","href":"`+cloud.Server.URL+`/network/next"}]}`)
		default:
			t.Errorf("method=%s", r.Method)
		}
	})
	cloud.Mux.HandleFunc("/network/next", func(w http.ResponseWriter, r *http.Request) {
		next++
		testcloud.JSON(w, 200, `{"ports":[{"id":"two"}]}`)
	})
	cloud.Mux.HandleFunc("/network/v2.0/ports/forbidden", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 403, `{"error":"forbidden"}`) })
	api := ports.New(cloud.Client("network", "/network/v2.0"))
	port, err := api.Create(context.Background(), ports.CreateOpts{NetworkID: "net-id", Name: "port"}, ports.WithCreateField("binding:host_id", "node"), ports.WithCreateField("vendor:flag", false))
	if err != nil || port.ID != "port-id" {
		t.Fatalf("port=%v err=%v", port, err)
	}
	if _, err := api.Create(context.Background(), ports.CreateOpts{NetworkID: "net-id"}, ports.WithCreateField("name", "smuggled")); !errors.Is(err, resource.ErrInvalidOption) || posts != 1 {
		t.Fatalf("posts=%d err=%v", posts, err)
	}
	opts := []ports.ListOption{ports.WithListOptions(ports.ListOpts{Limit: 5}), ports.WithListQuery("vendor:filter", "a&b")}
	for value, err := range api.List(context.Background(), opts...) {
		if err != nil || value.ID != "one" {
			t.Fatalf("value=%v err=%v", value, err)
		}
		break
	}
	if next != 0 {
		t.Fatal("early break fetched next page")
	}
	var ids []string
	for value, err := range api.List(context.Background(), opts...) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if strings.Join(ids, ",") != "one,two" || next != 1 {
		t.Fatalf("ids=%v next=%d", ids, next)
	}
	_, err = api.Get(context.Background(), "forbidden")
	var response gophercloud.ErrUnexpectedResponseCode
	var operation *resource.OperationError
	if !errors.As(err, &response) || response.Actual != 403 || !errors.As(err, &operation) {
		t.Fatalf("err=%v", err)
	}
}

func TestServerSecondaryBuilderAndTypedList(t *testing.T) {
	cloud := testcloud.New(t)
	group := "11111111-1111-4111-8111-111111111111"
	cloud.Mux.HandleFunc("/compute/servers", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		var hints map[string]string
		if err := json.Unmarshal(body["os:scheduler_hints"], &hints); err != nil {
			t.Error(err)
		}
		if hints["group"] != group {
			t.Errorf("hints=%v", hints)
		}
		testcloud.JSON(w, 202, `{"server":{"id":"server-id"}}`)
	})
	cloud.Mux.HandleFunc("/compute/servers/detail", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"servers":[{"id":"server-id","status":"ACTIVE"}]}`)
	})
	api := servers.New(cloud.Client("compute", "/compute"))
	server, err := api.Create(context.Background(), servers.CreateOpts{Name: "vm", ImageRef: "image-id", FlavorRef: "flavor-id"}, servers.WithCreateHintOpts(servers.SchedulerHintOpts{Group: group}))
	if err != nil || server.ID != "server-id" {
		t.Fatalf("server=%v err=%v", server, err)
	}
	if _, err := api.Create(context.Background(), servers.CreateOpts{Name: "vm", ImageRef: "image-id", FlavorRef: "flavor-id"}, request.WithArgument[servers.CreateOpts]("hintOpts", 123)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	count := 0
	for server, err := range api.List(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		if server.Status != "ACTIVE" {
			t.Fatal(server)
		}
		count++
	}
	if count != 1 {
		t.Fatal(count)
	}
}

func TestImagePatchPreservesFalseAndEmptyValues(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/image/images/image-id", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.Header.Get("Content-Type") != "application/openstack-images-v2.1-json-patch" {
			t.Errorf("method=%s headers=%v", r.Method, r.Header)
		}
		var patches []map[string]any
		if err := json.NewDecoder(r.Body).Decode(&patches); err != nil {
			t.Error(err)
		}
		if len(patches) != 2 || patches[0]["value"] != false || patches[1]["value"] != "" {
			t.Errorf("patches=%v", patches)
		}
		testcloud.JSON(w, 200, `{"id":"image-id","protected":false,"name":""}`)
	})
	image, err := images.New(cloud.Client("image", "/image")).Update(context.Background(), "image-id", images.UpdateOpts{images.ReplaceImageProtected{NewProtected: false}, images.ReplaceImageName{NewName: ""}})
	if err != nil || image.ID != "image-id" || image.Protected {
		t.Fatalf("image=%v err=%v", image, err)
	}
}

func TestSwiftUploadAndDownloadPreserveContent(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/swift/container/hello.txt", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != "hello" || r.Header.Get("X-Object-Meta-Owner") != "sdk" {
				t.Errorf("body=%s headers=%v err=%v", body, r.Header, err)
			}
			w.Header().Set("ETag", "etag")
			w.WriteHeader(201)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		if r.Header.Get("X-Vendor-Setting") != "false" || r.URL.Query().Get("vendor") != "a&b" {
			t.Errorf("headers=%v query=%v", r.Header, r.URL.Query())
		}
		w.Header().Set("Content-Length", "5")
		w.Header().Set("ETag", "etag")
		_, _ = io.WriteString(w, "hello")
	})
	api := objects.New(cloud.Client("object-store", "/swift"))
	header, err := api.Create(context.Background(), "container", "hello.txt", objects.CreateOpts{Content: strings.NewReader("hello"), Metadata: map[string]string{"Owner": "sdk"}})
	if err != nil || header.ETag != "etag" {
		t.Fatalf("header=%v err=%v", header, err)
	}
	if _, err := api.Download(context.Background(), "container", "hello.txt", request.WithField[objects.DownloadOpts]("ignored", true)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	download, err := api.Download(context.Background(), "container", "hello.txt", objects.WithDownloadHeader("X-Vendor-Setting", "false"), objects.WithDownloadQuery("vendor", "a&b"))
	if err != nil {
		t.Fatal(err)
	}
	defer download.Close()
	body, err := io.ReadAll(download)
	if err != nil || string(body) != "hello" || download.Header.ETag != "etag" {
		t.Fatalf("body=%s header=%v err=%v", body, download.Header, err)
	}
}
