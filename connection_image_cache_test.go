package gophercloudsdk_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/image"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionImageCacheSharesClientDefaultsAndPassiveMeasurements(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("cache-0")
	const node = "https://foreign.test/node/%2F?key=literal#fragment"
	calls, retries := 0, 0
	paths := []string{"cache", "cache/fixed", "cache/fixed", "cache", "cache/nodes/fixed", "cache/clean", "cache/prune", "cache/missing"}
	methods := []string{"GET", "PUT", "DELETE", "DELETE", "GET", "POST", "POST", "DELETE"}
	codes := []int{200, 202, 204, 204, 200, 200, 200, 404}
	bodies := []string{
		`{"cached_images":[{"image_id":"passive","hits":9007199254740993,"size":null,"last_accessed":1.00000000000000001,"last_modified":1e+3,"links":42}],"queued_images":["opaque%2F?key=value#fragment",""],"created_at":false}`,
		"opaque queue", "opaque deletion", "opaque clear",
		`["https://foreign.test/node/%2F?key=literal#fragment",""]`, "opaque clean",
		`{"total_files_pruned":9007199254740993,"total_bytes_pruned":0,"links":42}`, "missing entry",
	}
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		if calls >= len(paths) {
			t.Fatalf("unexpected continuation: %s", r.URL)
		}
		target := ""
		if calls == 3 {
			target = "queue"
		}
		if r.Method != methods[calls] || r.URL.String() != "https://cloud.test/reverse/glance/v2/"+paths[calls] || r.Body != nil || r.Header.Get("X-Source") != "shared" || r.Header.Get("X-Call") != "cache" || r.Header.Get("X-Auth-Token") != fmt.Sprintf("cache-%d", calls) || r.Header.Get("X-Image-Cache-Clear-Target") != target {
			t.Fatalf("request %d: %s %s headers=%v body=%v", calls, r.Method, r.URL, r.Header, r.Body)
		}
		n := calls
		calls++
		return &http.Response{StatusCode: codes[n], Header: http.Header{"X-Proof": {fmt.Sprintf("cache-%d", n)}, "Link": {`<https://foreign.test/next>; rel="next"`}}, Body: io.NopCloser(strings.NewReader(bodies[n]))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/reverse/glance/v2/"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	service, err := conn.Image(ctx)
	if err != nil {
		t.Fatal(err)
	}
	versioned, err := conn.ImageV2(ctx)
	if err != nil {
		t.Fatal(err)
	}
	client := versioned.RawClient()
	client.MoreHeaders = map[string]string{"X-Source": "shared"}
	client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		retries++
		return nil
	}
	if service.RawClient() != client || service.API.RawClient() != client || client.ProviderClient != provider {
		t.Fatal("cache operations replaced shared native client")
	}
	options := []image.CacheOption{image.WithCacheOpts(image.CacheOpts{Headers: map[string]string{"X-Call": "initial"}}), image.WithCacheHeader("X-Call", "intermediate"), image.WithCacheHeaders(map[string]string{"x-call": "cache"})}
	advance := func() { provider.SetToken(fmt.Sprintf("cache-%d", calls)) }
	snapshot, err := service.GetImageCache(ctx, options...)
	if err != nil || snapshot == nil || snapshot.StatusCode != 200 || len(snapshot.CachedImages) != 1 || len(snapshot.QueuedImages) != 2 || snapshot.QueuedImages[0] != "opaque%2F?key=value#fragment" || snapshot.CreatedAt != nil {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
	row := snapshot.CachedImages[0]
	if row.ImageID == nil || *row.ImageID != "passive" || row.Hits == nil || *row.Hits != 9007199254740993 || row.Size != nil || row.LastAccessed == nil || row.LastAccessed.String() != "1.00000000000000001" || row.LastModified == nil || row.LastModified.String() != "1e+3" || row.Links != nil || string(row.Body["links"]) != "42" || row.StatusCode != 200 {
		t.Fatalf("row=%+v", row)
	}
	row.Header.Set("X-Proof", "caller row")
	row.Body["hits"][0] = '!'
	if snapshot.Header.Get("X-Proof") != "cache-0" || !strings.Contains(string(snapshot.Body["cached_images"]), "9007199254740993") {
		t.Fatal("cache row aliases parent evidence")
	}
	advance()
	queued, err := service.QueueImage(ctx, resource.ID("fixed"), options...)
	if err != nil || queued == nil || queued.ImageID != "fixed" || queued.Target != nil || queued.StatusCode != 202 || string(queued.Body) != "opaque queue" {
		t.Fatalf("queue=%+v err=%v", queued, err)
	}
	advance()
	deleted, err := service.CacheDeleteImage(ctx, resource.ID("fixed"), image.WithCacheDeleteOpts(image.CacheDeleteOpts{Headers: map[string]string{"X-Call": "initial"}}), image.WithCacheDeleteHeader("X-Call", "intermediate"), image.WithCacheDeleteHeaders(map[string]string{"x-call": "cache"}), image.WithCacheDeleteIgnoreMissing(false))
	if err != nil || deleted == nil || deleted.ImageID != "fixed" || deleted.Target != nil || deleted.StatusCode != 204 || string(deleted.Body) != "opaque deletion" {
		t.Fatalf("delete=%+v err=%v", deleted, err)
	}
	advance()
	cleared, err := service.ClearCache(ctx, image.WithClearCacheOpts(image.ClearCacheOpts{Headers: map[string]string{"X-Call": "initial"}}), image.WithClearCacheHeader("X-Call", "intermediate"), image.WithClearCacheHeaders(map[string]string{"x-call": "cache"}), image.WithClearCacheTarget(image.QueueOnly))
	if err != nil || cleared == nil || cleared.ImageID != "" || cleared.Target == nil || *cleared.Target != image.QueueOnly || cleared.StatusCode != 204 || string(cleared.Body) != "opaque clear" {
		t.Fatalf("clear=%+v err=%v", cleared, err)
	}
	advance()
	nodes, err := service.CachedImageNodes(ctx, resource.ID("fixed"), options...)
	if err != nil || nodes == nil || nodes.ImageID != "fixed" || nodes.StatusCode != 200 || len(nodes.Nodes) != 2 || nodes.Nodes[0] != node || nodes.Nodes[1] != "" {
		t.Fatalf("nodes=%+v err=%v", nodes, err)
	}
	nodes.Nodes[0] = "caller"
	if !strings.Contains(string(nodes.Body), node) {
		t.Fatal("node strings alias raw response")
	}
	advance()
	cleaned, err := service.CleanCache(ctx, options...)
	if err != nil || cleaned == nil || cleaned.Target != nil || cleaned.StatusCode != 200 || string(cleaned.Body) != "opaque clean" {
		t.Fatalf("clean=%+v err=%v", cleaned, err)
	}
	advance()
	pruned, err := service.PruneCache(ctx, options...)
	if err != nil || pruned == nil || pruned.StatusCode != 200 || pruned.TotalFilesPruned != 9007199254740993 || pruned.TotalBytesPruned != 0 || pruned.Links != nil || string(pruned.Body["links"]) != "42" {
		t.Fatalf("prune=%+v err=%v", pruned, err)
	}
	advance()
	missing, err := service.CacheDeleteImage(ctx, resource.ID("missing"), image.WithCacheDeleteOpts(image.CacheDeleteOpts{Headers: map[string]string{"X-Call": "cache"}}))
	if err != nil || missing != nil || calls != 8 || retries != 0 {
		t.Fatalf("default404=%+v err=%v calls=%d retries=%d", missing, err, calls, retries)
	}
	if image.CacheBoth != 0 || image.CacheOnly != 1 || image.QueueOnly != 2 {
		t.Fatal("cache target zero default changed")
	}
}
