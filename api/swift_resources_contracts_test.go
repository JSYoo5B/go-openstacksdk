package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/testcloud"
	swift "gophercloudsdk/objectstorage/v1"
	"gophercloudsdk/objectstorage/v1/objects"
	"gophercloudsdk/resource"
)

func TestSwiftCollectionsOwnMetadataOpaqueKeysAndDeletionPolicies(t *testing.T) {
	const container, key = "container ?#% 한글", "folder/my file?#%.txt"
	cloud := testcloud.New(t)
	containerPath := "/swift/" + url.PathEscape(container)
	objectPath := containerPath + "/" + url.PathEscape(key)
	rootLists, objectLists, heads, deletes := 0, 0, 0, 0
	deleted := false
	cloud.Mux.HandleFunc("/swift/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/swift/":
			rootLists++
			if r.URL.Query().Get("prefix") != container || r.URL.Query().Has("name") {
				t.Errorf("parent query=%v", r.URL.Query())
			}
			if r.URL.Query().Get("marker") != "" {
				testcloud.JSON(w, 200, `[]`)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{{"name": container + "-other", "count": 1, "bytes": 1}, {"name": container, "count": 2, "bytes": 5}})
		case containerPath:
			if r.Method == http.MethodHead {
				heads++
				w.Header().Set("X-Container-Object-Count", "2")
				w.Header().Set("X-Container-Bytes-Used", "5")
				w.Header().Set("X-Container-Meta-Owner", "sdk")
				w.Header().Set("X-Vendor-Flag", "false")
				w.WriteHeader(204)
				return
			}
			objectLists++
			if r.URL.Query().Get("prefix") != key || r.URL.Query().Has("name") {
				t.Errorf("object query=%v", r.URL.Query())
			}
			if r.URL.Query().Get("marker") != "" {
				testcloud.JSON(w, 200, `[]`)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{{"name": key + ".backup", "bytes": 9}, {"name": key, "bytes": 5, "content_type": "text/plain", "hash": "checksum"}})
		case objectPath:
			switch r.Method {
			case http.MethodHead:
				heads++
				if deleted {
					w.WriteHeader(404)
					return
				}
				w.Header().Set("Content-Length", "5")
				w.Header().Set("Content-Type", "text/plain")
				w.Header().Set("ETag", "checksum")
				w.Header().Set("Last-Modified", time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC).Format(http.TimeFormat))
				w.Header().Set("X-Object-Meta-Owner", "sdk")
				w.Header().Set("X-Vendor-Flag", "false")
				w.WriteHeader(200)
			case http.MethodDelete:
				deletes++
				if deleted {
					w.WriteHeader(404)
					return
				}
				deleted = true
				w.WriteHeader(204)
			default:
				t.Errorf("method=%s", r.Method)
			}
		default:
			t.Errorf("wrong escaped path %q", r.URL.EscapedPath())
			w.WriteHeader(404)
		}
	})
	service := swift.New(cloud.Client("object-store", "/swift"))
	ctx := context.Background()
	idScope, err := service.Objects.InContainer(ctx, resource.ID(container))
	if err != nil || rootLists != 0 || heads != 0 {
		t.Fatalf("scope=%v err=%v root=%d heads=%d", idScope, err, rootLists, heads)
	}
	scope, err := service.Objects.InContainer(ctx, resource.Name(container))
	if err != nil || rootLists != 2 || heads != 0 {
		t.Fatalf("scope=%v err=%v root=%d heads=%d", scope, err, rootLists, heads)
	}
	parent, err := service.Containers.Resources.Get(ctx, container)
	if err != nil || parent.Name != container || parent.Count != 2 || parent.Bytes != 5 || parent.Metadata["Owner"] != "sdk" || parent.Header.Get("X-Vendor-Flag") != "false" || parent.Details == nil {
		t.Fatalf("parent=%+v err=%v", parent, err)
	}
	object, err := scope.Get(ctx, key)
	if err != nil || object.Name != key || object.Container != container || object.Bytes != 5 || object.Hash != "checksum" || object.Metadata["Owner"] != "sdk" || object.Header.Get("X-Vendor-Flag") != "false" || object.Details == nil {
		t.Fatalf("object=%+v err=%v", object, err)
	}
	found, err := scope.Find(ctx, resource.Name(key))
	if err != nil || found.Name != key || found.Bytes != 5 || found.Details != nil || objectLists != 2 {
		t.Fatalf("found=%+v err=%v lists=%d", found, err, objectLists)
	}
	if resolved, err := scope.ResolveID(ctx, resource.Name(key)); err != nil || resolved != key || rootLists != 2 {
		t.Fatalf("resolved=%q err=%v root=%d", resolved, err, rootLists)
	}
	before := heads
	if err := scope.Delete(ctx, resource.ID(key)); err != nil || deletes != 1 || heads != before {
		t.Fatalf("delete=%v deletes=%d heads=%d", err, deletes, heads)
	}
	if err := scope.WaitDeleted(ctx, resource.ID(key)); err != nil || heads != before+1 {
		t.Fatalf("wait deleted=%v heads=%d", err, heads)
	}
	if err := scope.Delete(ctx, resource.ID(key)); err != nil {
		t.Fatal(err)
	}
	if err := scope.Delete(ctx, resource.ID(key), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if _, err := service.Objects.InContainer(ctx, resource.ID("invalid/container")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := scope.Get(ctx, ""); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := scope.All(ctx, resource.WithStatus("ready")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
}

func TestSwiftScopeBindsUploadMetadataUpdateAndDownload(t *testing.T) {
	cloud := testcloud.New(t)
	puts, posts, copies := 0, 0, 0
	cloud.Mux.HandleFunc("/swift/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/swift/container/folder%2Ffile.txt" {
			t.Errorf("escaped path=%q", r.URL.EscapedPath())
		}
		switch r.Method {
		case http.MethodPut:
			puts++
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != "hello" || r.Header.Get("X-Object-Meta-Owner") != "sdk" || r.Header.Get("X-Vendor-Flag") != "false" {
				t.Errorf("body=%s header=%v err=%v", body, r.Header, err)
			}
			w.Header().Set("ETag", "checksum")
			w.WriteHeader(201)
		case http.MethodPost:
			posts++
			if r.Header.Get("X-Object-Meta-Owner") != "updated" {
				t.Errorf("headers=%v", r.Header)
			}
			w.WriteHeader(202)
		case http.MethodGet:
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Length", "5")
			w.Header().Set("ETag", "checksum")
			_, _ = io.WriteString(w, "hello")
		case "COPY":
			copies++
			if r.Header.Get("Destination") != "/target/copy" || r.URL.Query().Get("version-id") != "version" {
				t.Errorf("headers=%v query=%v", r.Header, r.URL.Query())
			}
			w.Header().Set("ETag", "checksum")
			w.WriteHeader(201)
		default:
			t.Errorf("unexpected method=%s", r.Method)
		}
	})
	scope, err := swift.New(cloud.Client("object-store", "/swift")).Objects.InContainer(context.Background(), resource.ID("container"))
	if err != nil {
		t.Fatal(err)
	}
	header, err := scope.Create(context.Background(), "folder/file.txt", objects.CreateOpts{Content: strings.NewReader("hello"), Metadata: map[string]string{"Owner": "sdk"}}, objects.WithCreateHeader("X-Vendor-Flag", "false"))
	if err != nil || header.ETag != "checksum" || puts != 1 {
		t.Fatalf("header=%v err=%v puts=%d", header, err, puts)
	}
	if _, err := scope.Update(context.Background(), resource.ID("folder/file.txt"), objects.UpdateOpts{Metadata: map[string]string{"Owner": "updated"}}); err != nil || posts != 1 {
		t.Fatalf("update=%v posts=%d", err, posts)
	}
	if header, err := scope.Copy(context.Background(), resource.ID("folder/file.txt"), objects.CopyOpts{Destination: "/target/copy", ObjectVersionID: "version"}); err != nil || header.ETag != "checksum" || copies != 1 {
		t.Fatalf("copy=%v err=%v copies=%d", header, err, copies)
	}
	download, err := scope.Download(context.Background(), resource.ID("folder/file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer download.Close()
	body, err := io.ReadAll(download)
	if err != nil || string(body) != "hello" || download.Header.ETag != "checksum" {
		t.Fatalf("body=%s metadata=%v err=%v", body, download.Header, err)
	}
}

func TestSwiftScopePreservesAuthorizationFailuresAndCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	cloud.Mux.HandleFunc("/swift/", func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(403) })
	service := swift.New(cloud.Client("object-store", "/swift"))
	scope, err := service.Objects.InContainer(context.Background(), resource.ID("container"))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range []error{
		scope.Delete(context.Background(), resource.ID("key")),
		scope.WaitDeleted(context.Background(), resource.ID("key")),
	} {
		var response gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &response) || response.Actual != 403 || errors.Is(err, resource.ErrNotFound) {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Objects.InContainer(ctx, resource.ID("container")); !errors.Is(err, context.Canceled) || calls != 2 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}
