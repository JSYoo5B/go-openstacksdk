package image_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pythonRejudgeImageRequest struct {
	method, path, target, body string
}

type pythonRejudgeImageReply struct {
	status int
	body   string
}

// pythonRejudgeImageService serves canned replies in order and records each
// request so a test can compare it with the pinned openstacksdk request.
func pythonRejudgeImageService(t *testing.T, replies ...pythonRejudgeImageReply) (*image.Service, *[]pythonRejudgeImageRequest) {
	t.Helper()
	cloud := testcloud.New(t)
	var seen []pythonRejudgeImageRequest
	cloud.Mux.HandleFunc("/image/v2/", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		seen = append(seen, pythonRejudgeImageRequest{method: r.Method, path: strings.TrimPrefix(r.URL.EscapedPath(), "/image/v2/") + pythonRejudgeImageQuery(r), target: r.Header.Get("X-Image-Cache-Clear-Target"), body: string(raw)})
		if len(seen) > len(replies) {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		reply := replies[len(seen)-1]
		if reply.body == "" {
			w.WriteHeader(reply.status)
			return
		}
		testcloud.JSON(w, reply.status, reply.body)
	})
	return image.New(cloud.Client("image", "/image/v2/")), &seen
}

func pythonRejudgeImageQuery(r *http.Request) string {
	if r.URL.RawQuery == "" {
		return ""
	}
	return "?" + r.URL.RawQuery
}

func pythonRejudgeImageExpect(t *testing.T, seen []pythonRejudgeImageRequest, want ...pythonRejudgeImageRequest) {
	t.Helper()
	if len(seen) != len(want) {
		t.Fatalf("requests = %#v, want %#v", seen, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("request %d = %#v, want %#v", i, seen[i], want[i])
		}
	}
}

// Python Cache.fetch, queue, clear, image_nodes, clean and prune plus
// Proxy._delete(Cache) send these bodyless requests with default arguments.
func TestPythonRejudgeImageCacheDefaultRequests(t *testing.T) {
	ctx := context.Background()
	t.Run("get_image_cache", func(t *testing.T) {
		service, seen := pythonRejudgeImageService(t, pythonRejudgeImageReply{200, `{"cached_images":[{"image_id":"img-1","hits":3,"size":10,"last_accessed":1.5,"last_modified":2.5}],"queued_images":["img-2"]}`})
		value, err := service.GetImageCache(ctx)
		if err != nil || len(value.CachedImages) != 1 || *value.CachedImages[0].ImageID != "img-1" || *value.CachedImages[0].Hits != 3 || len(value.QueuedImages) != 1 || value.QueuedImages[0] != "img-2" {
			t.Fatal(value, err)
		}
		pythonRejudgeImageExpect(t, *seen, pythonRejudgeImageRequest{method: "GET", path: "cache"})
	})
	t.Run("queue_image", func(t *testing.T) {
		service, seen := pythonRejudgeImageService(t, pythonRejudgeImageReply{202, ""})
		if ack, err := service.QueueImage(ctx, resource.ID("img-1")); err != nil || ack.ImageID != "img-1" || ack.StatusCode != 202 {
			t.Fatal(ack, err)
		}
		pythonRejudgeImageExpect(t, *seen, pythonRejudgeImageRequest{method: "PUT", path: "cache/img-1"})
	})
	t.Run("cache_delete_image ignores missing by default", func(t *testing.T) {
		service, seen := pythonRejudgeImageService(t, pythonRejudgeImageReply{204, ""}, pythonRejudgeImageReply{404, `{"message":"missing"}`})
		if ack, err := service.CacheDeleteImage(ctx, resource.ID("img-1")); err != nil || ack == nil || ack.StatusCode != 204 {
			t.Fatal(ack, err)
		}
		if ack, err := service.CacheDeleteImage(ctx, resource.ID("img-1")); err != nil || ack != nil {
			t.Fatal(ack, err)
		}
		pythonRejudgeImageExpect(t, *seen, pythonRejudgeImageRequest{method: "DELETE", path: "cache/img-1"}, pythonRejudgeImageRequest{method: "DELETE", path: "cache/img-1"})
	})
	t.Run("clear_cache target both omits header", func(t *testing.T) {
		service, seen := pythonRejudgeImageService(t, pythonRejudgeImageReply{204, ""})
		if ack, err := service.ClearCache(ctx); err != nil || ack == nil || *ack.Target != image.CacheBoth {
			t.Fatal(ack, err)
		}
		pythonRejudgeImageExpect(t, *seen, pythonRejudgeImageRequest{method: "DELETE", path: "cache"})
	})
	t.Run("cached_image_nodes", func(t *testing.T) {
		service, seen := pythonRejudgeImageService(t, pythonRejudgeImageReply{200, `["http://node-1:9292","http://node-2:9292"]`})
		value, err := service.CachedImageNodes(ctx, resource.ID("img-1"))
		if err != nil || len(value.Nodes) != 2 || value.Nodes[0] != "http://node-1:9292" {
			t.Fatal(value, err)
		}
		pythonRejudgeImageExpect(t, *seen, pythonRejudgeImageRequest{method: "GET", path: "cache/nodes/img-1"})
	})
	t.Run("clean_cache", func(t *testing.T) {
		service, seen := pythonRejudgeImageService(t, pythonRejudgeImageReply{200, ""})
		if ack, err := service.CleanCache(ctx); err != nil || ack == nil {
			t.Fatal(ack, err)
		}
		pythonRejudgeImageExpect(t, *seen, pythonRejudgeImageRequest{method: "POST", path: "cache/clean"})
	})
	t.Run("prune_cache", func(t *testing.T) {
		service, seen := pythonRejudgeImageService(t, pythonRejudgeImageReply{200, `{"total_files_pruned":2,"total_bytes_pruned":1024}`})
		value, err := service.PruneCache(ctx)
		if err != nil || value.TotalFilesPruned != 2 || value.TotalBytesPruned != 1024 {
			t.Fatal(value, err)
		}
		pythonRejudgeImageExpect(t, *seen, pythonRejudgeImageRequest{method: "POST", path: "cache/prune"})
	})
}

// Python image_locations lists /images/{id}/locations once and get_task
// fetches /tasks/{id}; neither adds a query string.
func TestPythonRejudgeImageLocationsAndTaskDefaultRequests(t *testing.T) {
	ctx := context.Background()
	t.Run("image_locations", func(t *testing.T) {
		service, seen := pythonRejudgeImageService(t, pythonRejudgeImageReply{200, `[{"url":"file:///var/lib/glance/img-1","metadata":{"store":"file1"}}]`})
		value, err := service.GetImageLocations(ctx, resource.ID("img-1"))
		if err != nil || len(value.Locations) != 1 || *value.Locations[0].URL != "file:///var/lib/glance/img-1" || string(value.Locations[0].Metadata["store"]) != `"file1"` {
			t.Fatal(value, err)
		}
		pythonRejudgeImageExpect(t, *seen, pythonRejudgeImageRequest{method: "GET", path: "images/img-1/locations"})
	})
	t.Run("get_task", func(t *testing.T) {
		service, seen := pythonRejudgeImageService(t, pythonRejudgeImageReply{200, `{"id":"task-1","type":"import","status":"success","owner":"project","message":"","input":{"import_from":"http://example.invalid/x"},"result":{"image_id":"img-1"},"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:01Z","expires_at":"2026-01-03T00:00:00Z","schema":"/v2/schemas/task","self":"/v2/tasks/task-1"}`})
		value, err := service.GetTask(ctx, "task-1")
		if err != nil || *value.ID != "task-1" || *value.Status != "success" || *value.Type != "import" || string(value.Result["image_id"]) != `"img-1"` {
			t.Fatal(value, err)
		}
		pythonRejudgeImageExpect(t, *seen, pythonRejudgeImageRequest{method: "GET", path: "tasks/task-1"})
	})
}
