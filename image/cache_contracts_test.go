package image_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/image"
	sdkimages "github.com/JSYoo5B/gophercloudsdk/image/v2/images"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const cacheContractPrefix = "/reverse/cache-management/glance/v2/"
const cacheContractBase = "https://glance.invalid" + cacheContractPrefix

// A synthetic transport permits opaque 204 bodies that net/http servers suppress.
type cacheContractTransport func(*http.Request) (*http.Response, error)

func (f cacheContractTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	v, e := f(r)
	if v != nil && v.Request == nil {
		v.Request = r
	}
	return v, e
}

type cacheContractBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *cacheContractBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type cacheContractReader func([]byte) (int, error)

func (f cacheContractReader) Read(b []byte) (int, error) { return f(b) }
func cacheContractWire(code int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"actual-cache"}}, Body: body}
}
func cacheContractClient(f cacheContractTransport) *gophercloud.ServiceClient {
	p := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: f}}
	p.UseTokenLock()
	p.SetToken("initial")
	return &gophercloud.ServiceClient{ProviderClient: p, Type: "image", Endpoint: cacheContractBase}
}

type cacheContractOptions struct {
	common   []image.CacheOption
	deletion []image.CacheDeleteOption
	clear    []image.ClearCacheOption
}
type cacheContractValue struct {
	ack   *image.ImageCacheAcknowledgement
	state *image.ImageCache
	nodes *image.CachedImageNodesResult
	prune *image.CachePruneResult
}

func (v *cacheContractValue) header() http.Header {
	switch {
	case v.ack != nil:
		return v.ack.Header
	case v.state != nil:
		return v.state.Header
	case v.nodes != nil:
		return v.nodes.Header
	default:
		return v.prune.Header
	}
}
func (v *cacheContractValue) status() int {
	switch {
	case v.ack != nil:
		return v.ack.StatusCode
	case v.state != nil:
		return v.state.StatusCode
	case v.nodes != nil:
		return v.nodes.StatusCode
	default:
		return v.prune.StatusCode
	}
}
func cacheContractCall(s *image.Service, ctx context.Context, name string, ref resource.Ref, o cacheContractOptions) (*cacheContractValue, error) {
	switch name {
	case "GetImageCache":
		v, e := s.GetImageCache(ctx, o.common...)
		if v == nil {
			return nil, e
		}
		return &cacheContractValue{state: v}, e
	case "QueueImage":
		v, e := s.QueueImage(ctx, ref, o.common...)
		if v == nil {
			return nil, e
		}
		return &cacheContractValue{ack: v}, e
	case "CacheDeleteImage":
		v, e := s.CacheDeleteImage(ctx, ref, o.deletion...)
		if v == nil {
			return nil, e
		}
		return &cacheContractValue{ack: v}, e
	case "ClearCache":
		v, e := s.ClearCache(ctx, o.clear...)
		if v == nil {
			return nil, e
		}
		return &cacheContractValue{ack: v}, e
	case "CachedImageNodes":
		v, e := s.CachedImageNodes(ctx, ref, o.common...)
		if v == nil {
			return nil, e
		}
		return &cacheContractValue{nodes: v}, e
	case "CleanCache":
		v, e := s.CleanCache(ctx, o.common...)
		if v == nil {
			return nil, e
		}
		return &cacheContractValue{ack: v}, e
	case "PruneCache":
		v, e := s.PruneCache(ctx, o.common...)
		if v == nil {
			return nil, e
		}
		return &cacheContractValue{prune: v}, e
	default:
		panic(name)
	}
}

type cacheContractOperation struct {
	name, method, path, raw string
	status                  int
	perID, ack              bool
}

func cacheContractOperations() []cacheContractOperation {
	return []cacheContractOperation{
		{"GetImageCache", "GET", "cache", `{"cached_images":[],"queued_images":[]}`, 200, false, false},
		{"QueueImage", "PUT", "cache/fixed", "", 202, true, true},
		{"CacheDeleteImage", "DELETE", "cache/fixed", "", 204, true, true},
		{"ClearCache", "DELETE", "cache", "", 204, false, true},
		{"CachedImageNodes", "GET", "cache/nodes/fixed", `["https://foreign.invalid/node","","literal","literal"]`, 200, true, false},
		{"CleanCache", "POST", "cache/clean", "", 200, false, true},
		{"PruneCache", "POST", "cache/prune", `{"total_files_pruned":0,"total_bytes_pruned":0}`, 200, false, false},
	}
}
func cacheContractProof(t *testing.T, e error, status int, raw []byte) *resource.ResponseError {
	t.Helper()
	var p *resource.ResponseError
	if !errors.As(e, &p) || p.StatusCode != status || !bytes.Equal(p.Body, raw) || p.Header.Get("X-Request-Id") != "actual-cache" {
		t.Fatalf("accepted cache proof: %v %#v", e, p)
	}
	return p
}

func TestImageCacheFixedRoutesAndClearTargets(t *testing.T) {
	t.Run("seven fixed bodyless routes with live auth", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := cloud.Client("image", "/unused/catalog/")
		client.ResourceBase = cloud.Server.URL + cacheContractPrefix
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		client.Microversion = "2.18"
		ops := cacheContractOperations()
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			i := int(calls.Add(1)) - 1
			if i >= len(ops) {
				t.Error("discovery/fallback/fanout", r.URL)
				w.WriteHeader(500)
				return
			}
			op := ops[i]
			raw, e := io.ReadAll(r.Body)
			if e != nil || len(raw) != 0 || r.ContentLength != 0 || r.Method != op.method || r.URL.Path != cacheContractPrefix+op.path || r.URL.RawQuery != "" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Option") != "ordinary" || r.Header.Get("X-Auth-Token") != fmt.Sprint("live-", i) || r.Header.Get("OpenStack-API-Version") != "image 2.18" || r.Header.Get("X-Image-Cache-Clear-Target") != "" {
				t.Error(op.name, r.Method, r.URL, r.Header, string(raw), e)
			}
			w.Header().Set("X-Request-Id", "actual-cache")
			w.Header().Set("Location", "https://foreign.invalid/target")
			w.Header().Set("Link", "<https://foreign.invalid/next>; rel=\"next\"")
			w.WriteHeader(op.status)
			_, _ = io.WriteString(w, op.raw)
		})
		for i, op := range ops {
			cloud.Provider.SetToken(fmt.Sprint("live-", i))
			o := cacheContractOptions{common: []image.CacheOption{image.WithCacheHeader("X-Option", "ordinary")}, deletion: []image.CacheDeleteOption{image.WithCacheDeleteHeader("X-Option", "ordinary")}, clear: []image.ClearCacheOption{image.WithClearCacheHeader("X-Option", "ordinary")}}
			v, e := cacheContractCall(image.New(client), context.Background(), op.name, resource.ID("fixed"), o)
			if e != nil || v == nil || v.status() != op.status || v.header().Get("X-Request-Id") != "actual-cache" || calls.Load() != int32(i+1) {
				t.Fatal(op.name, v, e, calls.Load())
			}
			if v.ack != nil {
				if op.perID && v.ack.ImageID != "fixed" || !op.perID && v.ack.ImageID != "" || op.name == "ClearCache" && (v.ack.Target == nil || *v.ack.Target != image.CacheBoth) || op.name != "ClearCache" && v.ack.Target != nil {
					t.Fatal(op.name, v.ack)
				}
			}
			if v.state != nil && (v.state.CachedImages == nil || v.state.QueuedImages == nil) {
				t.Fatal(v.state)
			}
			if v.nodes != nil && (v.nodes.ImageID != "fixed" || !reflect.DeepEqual(v.nodes.Nodes, []string{"https://foreign.invalid/node", "", "literal", "literal"}) || string(v.nodes.Body) != op.raw) {
				t.Fatal(v.nodes)
			}
		}
	})
	for _, tc := range []struct {
		name, header string
		target       image.CacheTarget
		options      []image.ClearCacheOption
	}{{"default", "", image.CacheBoth, nil}, {"zero replacement", "", image.CacheBoth, []image.ClearCacheOption{image.WithClearCacheOpts(image.ClearCacheOpts{})}}, {"explicit both", "", image.CacheBoth, []image.ClearCacheOption{image.WithClearCacheTarget(image.CacheBoth)}}, {"cache", "cache", image.CacheOnly, []image.ClearCacheOption{image.WithClearCacheTarget(image.CacheOnly)}}, {"queue", "queue", image.QueueOnly, []image.ClearCacheOption{image.WithClearCacheTarget(image.QueueOnly)}}} {
		t.Run("clear "+tc.name, func(t *testing.T) {
			body := &cacheContractBody{Reader: bytes.NewReader([]byte{0, 255, 'x'})}
			var calls atomic.Int32
			client := cacheContractClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				_, present := r.Header["X-Image-Cache-Clear-Target"]
				if r.Method != "DELETE" || r.URL.String() != cacheContractBase+"cache" || r.Body != nil || r.Header.Get("X-Image-Cache-Clear-Target") != tc.header || tc.header == "" && present {
					t.Error(r.Method, r.URL, r.Header, r.Body)
				}
				return cacheContractWire(204, body), nil
			})
			v, e := image.New(client).ClearCache(context.Background(), tc.options...)
			if e != nil || v == nil || v.Target == nil || *v.Target != tc.target || v.ImageID != "" || !bytes.Equal(v.Body, []byte{0, 255, 'x'}) || v.StatusCode != 204 || calls.Load() != 1 || body.closes.Load() != 1 {
				t.Fatal(v, e, calls.Load(), body.closes.Load())
			}
		})
	}
	for _, op := range cacheContractOperations() {
		if !op.ack {
			continue
		}
		t.Run("opaque "+op.name, func(t *testing.T) {
			raw := []byte{0, 255, 'a'}
			body := &cacheContractBody{Reader: bytes.NewReader(raw)}
			client := cacheContractClient(func(*http.Request) (*http.Response, error) { return cacheContractWire(op.status, body), nil })
			v, e := cacheContractCall(image.New(client), context.Background(), op.name, resource.ID("fixed"), cacheContractOptions{})
			if e != nil || v == nil || !bytes.Equal(v.ack.Body, raw) || body.closes.Load() != 1 {
				t.Fatal(v, e, body.closes.Load())
			}
			v.ack.Body[0] = '!'
			if raw[0] != 0 {
				t.Fatal("borrowed bytes mutated")
			}
		})
	}
}

func TestImageCacheCanonicalModelsAndRawPrecision(t *testing.T) {
	t.Run("numeric epochs and passive independent metadata", func(t *testing.T) {
		raw := `{"cached_images":[{"image_id":"","hits":0,"size":9223372036854775807,"last_accessed":9007199254740993.000000001,"last_modified":-1.234e-9,"links":true,"created_at":{},"checksum":{"passive":true},"future":9007199254740995},{}],"queued_images":["","foreign?id","dup","dup"],"links":false,"updated_at":[],"Cached_Images":12}`
		body := &cacheContractBody{Reader: strings.NewReader(raw)}
		wire := cacheContractWire(200, body)
		body.onClose = func() { wire.Header.Set("X-Request-Id", "changed") }
		var calls atomic.Int32
		client := cacheContractClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return wire, nil })
		v, e := image.New(client).GetImageCache(context.Background())
		if e != nil || v == nil || len(v.CachedImages) != 2 || calls.Load() != 1 || body.closes.Load() != 1 {
			t.Fatal(v, e)
		}
		row := v.CachedImages[0]
		if row.ImageID == nil || *row.ImageID != "" || row.Hits == nil || *row.Hits != 0 || row.Size == nil || *row.Size != 9223372036854775807 || row.LastAccessed == nil || row.LastAccessed.String() != "9007199254740993.000000001" || row.LastModified == nil || row.LastModified.String() != "-1.234e-9" || v.CachedImages[1].LastAccessed != nil || v.CachedImages[1].ImageID != nil || !reflect.DeepEqual(v.QueuedImages, []string{"", "foreign?id", "dup", "dup"}) {
			t.Fatal(v, row)
		}
		var fields map[string]json.RawMessage
		_ = json.Unmarshal([]byte(raw), &fields)
		if !reflect.DeepEqual(v.Body, fields) || string(row.Body["future"]) != "9007199254740995" || string(row.Body["checksum"]) != `{"passive":true}` || v.Links != nil || v.CreatedAt != nil || v.UpdatedAt != nil || row.Links != nil || row.CreatedAt != nil || row.UpdatedAt != nil {
			t.Fatal(v, row)
		}
		for _, r := range v.CachedImages {
			if r.StatusCode != 200 || r.Header.Get("X-Request-Id") != "actual-cache" {
				t.Fatal(r)
			}
		}
		v.Header.Set("X-Request-Id", "root")
		row.Header.Set("X-Request-Id", "row")
		if v.CachedImages[1].Header.Get("X-Request-Id") != "actual-cache" || wire.Header.Get("X-Request-Id") != "changed" {
			t.Fatal("header alias")
		}
		v.Body["cached_images"][0] = '!'
		row.Body["last_accessed"][0] = '!'
		*row.ImageID = "caller"
		if row.LastAccessed.String() != "9007199254740993.000000001" || string(row.Body["image_id"]) != `""` || string(v.CachedImages[1].Body["image_id"]) != "" {
			t.Fatal("raw/typed alias", row)
		}
	})
	cases := []struct{ op, raw string }{
		{"GetImageCache", `null`}, {"GetImageCache", `[]`}, {"GetImageCache", `{}`}, {"GetImageCache", `{"Cached_Images":[],"queued_images":[]}`}, {"GetImageCache", `{"cached_images":null,"queued_images":[]}`}, {"GetImageCache", `{"cached_images":{},"queued_images":[]}`}, {"GetImageCache", `{"cached_images":[null],"queued_images":[]}`}, {"GetImageCache", `{"cached_images":[12],"queued_images":[]}`}, {"GetImageCache", `{"cached_images":[],"queued_images":null}`}, {"GetImageCache", `{"cached_images":[],"queued_images":[null]}`}, {"GetImageCache", `{"cached_images":[],"queued_images":[12]}`},
		{"PruneCache", `null`}, {"PruneCache", `[]`}, {"PruneCache", `{}`}, {"PruneCache", `{"total_files_pruned":0,"total_bytes_pruned":null}`}, {"PruneCache", `{"Total_Files_Pruned":0,"total_bytes_pruned":0}`},
		{"CachedImageNodes", `null`}, {"CachedImageNodes", `{}`}, {"CachedImageNodes", `[null]`}, {"CachedImageNodes", `[12]`}, {"CachedImageNodes", `[false]`},
	}
	for _, field := range []string{"image_id", "hits", "size", "last_accessed", "last_modified"} {
		bad := []string{`true`, `{}`, `[]`}
		if field == "image_id" {
			bad = append(bad, `12`)
		} else {
			bad = append(bad, `"123"`)
			if field == "hits" || field == "size" {
				bad = append(bad, `1.0`, `1e2`, `9223372036854775808`)
			}
		}
		for _, value := range bad {
			cases = append(cases, struct{ op, raw string }{"GetImageCache", `{"cached_images":[{"` + field + `":` + value + `}],"queued_images":[]}`})
		}
	}
	for _, field := range []string{"total_files_pruned", "total_bytes_pruned"} {
		for _, value := range []string{`null`, `"1"`, `true`, `1.0`, `1e2`, `9223372036854775808`} {
			other := "total_files_pruned"
			if field == other {
				other = "total_bytes_pruned"
			}
			cases = append(cases, struct{ op, raw string }{"PruneCache", `{"` + other + `":0,"` + field + `":` + value + `}`})
		}
	}
	for _, op := range []string{"GetImageCache", "PruneCache", "CachedImageNodes"} {
		raw := `["` + string([]byte{255}) + `"]`
		if op == "GetImageCache" {
			raw = `{"cached_images":[],"queued_images":[],"unknown":"` + string([]byte{255}) + `"}`
		} else if op == "PruneCache" {
			raw = `{"total_files_pruned":0,"total_bytes_pruned":0,"unknown":"` + string([]byte{255}) + `"}`
		}
		cases = append(cases, struct{ op, raw string }{op, raw})
	}
	for _, tc := range cases {
		t.Run(tc.op+" rejects "+tc.raw, func(t *testing.T) {
			body := &cacheContractBody{Reader: strings.NewReader(tc.raw)}
			var calls, retries atomic.Int32
			client := cacheContractClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return cacheContractWire(200, body), nil })
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries.Add(1)
				return errors.New("decode replay")
			}
			v, e := cacheContractCall(image.New(client), context.Background(), tc.op, resource.ID("fixed"), cacheContractOptions{})
			cacheContractProof(t, e, 200, []byte(tc.raw))
			if v != nil || calls.Load() != 1 || retries.Load() != 0 || body.closes.Load() != 1 {
				t.Fatal(v, e, calls.Load(), retries.Load(), body.closes.Load())
			}
		})
	}
	for _, raw := range []string{`{"cached_images":[{"image_id":null,"hits":null,"size":null,"last_accessed":null,"last_modified":null}],"queued_images":[]}`, `{"cached_images":[{"hits":-9223372036854775808,"size":0,"last_accessed":1e100,"last_modified":0}],"queued_images":[]}`} {
		t.Run("valid optional "+raw, func(t *testing.T) {
			client := cacheContractClient(func(*http.Request) (*http.Response, error) {
				return cacheContractWire(200, io.NopCloser(strings.NewReader(raw))), nil
			})
			v, e := image.New(client).GetImageCache(context.Background())
			if e != nil || v == nil || v.QueuedImages == nil {
				t.Fatal(v, e)
			}
			if strings.Contains(raw, "null") && (v.CachedImages[0].Hits != nil || v.CachedImages[0].LastAccessed != nil) {
				t.Fatal(v)
			}
		})
	}
	t.Run("prune signed counts raw precision and empty nodes", func(t *testing.T) {
		raw := `{"total_files_pruned":-9223372036854775808,"total_bytes_pruned":9223372036854775807,"unknown":9007199254740995,"links":false}`
		client := cacheContractClient(func(r *http.Request) (*http.Response, error) {
			payload := raw
			if strings.Contains(r.URL.Path, "nodes/") {
				payload = `[]`
			}
			return cacheContractWire(200, io.NopCloser(strings.NewReader(payload))), nil
		})
		v, e := image.New(client).PruneCache(context.Background())
		if e != nil || v == nil || v.TotalFilesPruned != -9223372036854775808 || v.TotalBytesPruned != 9223372036854775807 || string(v.Body["unknown"]) != "9007199254740995" || v.Links != nil {
			t.Fatal(v, e)
		}
		n, e := image.New(client).CachedImageNodes(context.Background(), resource.ID("fixed"))
		if e != nil || n == nil || n.Nodes == nil || len(n.Nodes) != 0 || string(n.Body) != "[]" {
			t.Fatal(n, e)
		}
	})
	t.Run("model failure is atomic", func(t *testing.T) {
		v := image.ImageCache{QueuedImages: []string{"keep"}}
		before := v
		if e := json.Unmarshal([]byte(`{"cached_images":[{}],"queued_images":[null]}`), &v); e == nil || !reflect.DeepEqual(v, before) {
			t.Fatal(v, e)
		}
		row := image.CachedImage{ImageID: new(string)}
		*row.ImageID = "keep"
		old := row
		if e := json.Unmarshal([]byte(`{"image_id":"changed","size":"bad"}`), &row); e == nil || !reflect.DeepEqual(row, old) {
			t.Fatal(row, e)
		}
	})
}

func TestImageCachePreparedOptionsAndLiveSource(t *testing.T) {
	for _, op := range cacheContractOperations() {
		for _, origin := range []string{"source", "option", "source after callback"} {
			t.Run(op.name+" reserved target "+origin, func(t *testing.T) {
				var calls atomic.Int32
				client := cacheContractClient(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					return cacheContractWire(500, io.NopCloser(strings.NewReader("unexpected"))), nil
				})
				o := cacheContractOptions{}
				key := "x-IMAGE-cache-CLEAR-target"
				if origin == "source" {
					client.MoreHeaders = map[string]string{key: ""}
				} else if origin == "option" {
					o.common = []image.CacheOption{image.WithCacheHeader(key, "cache")}
					o.deletion = []image.CacheDeleteOption{image.WithCacheDeleteHeader(key, "cache")}
					o.clear = []image.ClearCacheOption{image.WithClearCacheHeader(key, "queue")}
				} else {
					o.common = []image.CacheOption{func(*image.CacheOpts) error { client.MoreHeaders = map[string]string{key: "queue"}; return nil }}
					o.deletion = []image.CacheDeleteOption{func(*image.CacheDeleteOpts) error { client.MoreHeaders = map[string]string{key: "queue"}; return nil }}
					o.clear = []image.ClearCacheOption{func(*image.ClearCacheOpts) error { client.MoreHeaders = map[string]string{key: "queue"}; return nil }}
				}
				v, e := cacheContractCall(image.New(client), context.Background(), op.name, resource.Name("needle"), o)
				if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
					t.Fatal(v, e, calls.Load())
				}
			})
		}
	}
	for _, mode := range []string{"nil context", "canceled", "nil service", "nil client", "nil provider", "wrong type", "foreign base", "query base", "no slash", "source auth", "source aliases", "invalid version", "zero ref", "unsafe ID", "bad UTF8", "nil option", "callback error", "option aliases", "provider replacement"} {
		t.Run(mode, func(t *testing.T) {
			var calls, callbacks atomic.Int32
			client := cacheContractClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return cacheContractWire(500, io.NopCloser(strings.NewReader("unexpected"))), nil
			})
			ctx := context.Background()
			ref := resource.ID("fixed")
			cause := errors.New("preflight cause")
			want := error(resource.ErrInvalidOption)
			expected := int32(0)
			option := image.CacheOption(func(*image.CacheOpts) error { callbacks.Add(1); return nil })
			switch mode {
			case "nil context":
				ctx = nil
			case "canceled":
				c, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx = c
				want = context.Canceled
			case "nil client":
				client = nil
			case "nil provider":
				client.ProviderClient = nil
			case "wrong type":
				client.Type = "compute"
				want = resource.ErrUnsupported
			case "foreign base":
				client.ResourceBase = "https://foreign.invalid/v2/"
			case "query base":
				client.ResourceBase = cacheContractBase + "?q=x"
			case "no slash":
				client.Endpoint = strings.TrimSuffix(cacheContractBase, "/")
			case "source auth":
				client.MoreHeaders = map[string]string{"Authorization": "foreign"}
			case "source aliases":
				client.MoreHeaders = map[string]string{"X-Source": "a", "x-source": "b"}
			case "invalid version":
				client.Microversion = "2.18\nbad"
			case "zero ref":
				ref = resource.Ref{}
			case "unsafe ID":
				ref = resource.ID("../escape")
			case "bad UTF8":
				ref = resource.Name(string([]byte{255}))
			case "nil option":
				option = nil
			case "callback error":
				expected = 1
				want = cause
				option = func(*image.CacheOpts) error { callbacks.Add(1); return cause }
			case "option aliases":
				expected = 1
				option = func(o *image.CacheOpts) error {
					callbacks.Add(1)
					o.Headers = map[string]string{"X-Option": "a", "x-option": "b"}
					return nil
				}
			case "provider replacement":
				expected = 1
				option = func(*image.CacheOpts) error {
					callbacks.Add(1)
					client.ProviderClient = &gophercloud.ProviderClient{}
					return nil
				}
			}
			s := image.New(client)
			if mode == "nil service" {
				s = nil
			}
			v, e := s.QueueImage(ctx, ref, option)
			if v != nil || !errors.Is(e, want) || calls.Load() != 0 || callbacks.Load() != expected || mode == "canceled" && !errors.Is(e, cause) {
				t.Fatal(v, e, calls.Load(), callbacks.Load())
			}
		})
	}
	for _, target := range []image.CacheTarget{3, 255} {
		t.Run(fmt.Sprint("invalid enum ", target), func(t *testing.T) {
			var calls atomic.Int32
			client := cacheContractClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected") })
			v, e := image.New(client).ClearCache(context.Background(), image.WithClearCacheTarget(target))
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(v, e, calls.Load())
			}
		})
	}
	t.Run("header helpers snapshots callbacks and live source", func(t *testing.T) {
		input := map[string]string{"X-Option": "owned"}
		helper := image.WithCacheHeaders(input)
		input["X-Option"] = "mutated"
		var retained *image.CacheOpts
		var callbacks, calls atomic.Int32
		client := cacheContractClient(nil)
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		client.Microversion = "2.18"
		client.HTTPClient.Transport = cacheContractTransport(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.URL.String() != cacheContractBase+"cache/fixed" || r.Method != "PUT" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Option") != "owned" || r.Header.Get("X-Extra") != "last" || r.Header.Get("X-Auth-Token") != "live" || r.Header.Get("OpenStack-API-Version") != "image 2.18" {
				t.Error(r.URL, r.Header)
			}
			return cacheContractWire(202, io.NopCloser(strings.NewReader(""))), nil
		})
		options := []image.CacheOption{helper, func(o *image.CacheOpts) error {
			callbacks.Add(1)
			retained = o
			client.ResourceBase = "https://glance.invalid/later/v2/"
			client.MoreHeaders = map[string]string{"X-Source": "later"}
			client.SetToken("live")
			return nil
		}, func(o *image.CacheOpts) error {
			callbacks.Add(1)
			retained.Headers["X-Option"] = "retained mutation"
			if o.Headers["X-Option"] != "owned" {
				t.Error(o)
			}
			o.Headers["X-Extra"] = "last"
			return nil
		}}
		v, e := image.New(client).QueueImage(context.Background(), resource.ID("fixed"), options...)
		if e != nil || v == nil || callbacks.Load() != 2 || calls.Load() != 1 || input["X-Option"] != "mutated" {
			t.Fatal(v, e, callbacks.Load(), calls.Load())
		}
	})
	t.Run("full replacement and bool pointer ownership", func(t *testing.T) {
		flag := false
		headers := map[string]string{"X-Option": "owned"}
		helper := image.WithCacheDeleteOpts(image.CacheDeleteOpts{Headers: headers, IgnoreMissing: &flag})
		flag = true
		headers["X-Option"] = "caller"
		var retained *image.CacheDeleteOpts
		var calls atomic.Int32
		client := cacheContractClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Option") != "owned" || r.Header.Get("X-Dropped") != "" || r.Header.Get("X-Last") != "last" {
				t.Error(r.Header)
			}
			return cacheContractWire(404, io.NopCloser(strings.NewReader("native404"))), nil
		})
		v, e := image.New(client).CacheDeleteImage(context.Background(), resource.ID("fixed"), image.WithCacheDeleteHeader("X-Dropped", "before"), helper, func(o *image.CacheDeleteOpts) error {
			retained = o
			if o.IgnoreMissing == nil || *o.IgnoreMissing {
				t.Error(o)
			}
			return nil
		}, func(o *image.CacheDeleteOpts) error {
			*retained.IgnoreMissing = true
			retained.Headers["X-Option"] = "retained"
			if *o.IgnoreMissing || o.Headers["X-Option"] != "owned" {
				t.Error(o)
			}
			o.Headers["X-Last"] = "last"
			return nil
		})
		if v != nil || !gophercloud.ResponseCodeIs(e, 404) || calls.Load() != 1 {
			t.Fatal(v, e, calls.Load())
		}
	})
	t.Run("clear replacement and helper overrides", func(t *testing.T) {
		headers := map[string]string{"X-Option": "owned"}
		helper := image.WithClearCacheOpts(image.ClearCacheOpts{Headers: headers, Target: image.CacheOnly})
		headers["X-Option"] = "caller"
		client := cacheContractClient(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Dropped") != "" || r.Header.Get("X-Option") != "last" || r.Header.Get("X-Extra") != "owned" || r.Header.Get("X-Image-Cache-Clear-Target") != "queue" {
				t.Error(r.Header)
			}
			return cacheContractWire(204, io.NopCloser(strings.NewReader(""))), nil
		})
		v, e := image.New(client).ClearCache(context.Background(), image.WithClearCacheHeader("X-Dropped", "before"), helper, image.WithClearCacheHeaders(map[string]string{"X-Extra": "owned"}), image.WithClearCacheHeader("x-option", "last"), image.WithClearCacheTarget(image.QueueOnly))
		if e != nil || v == nil || v.Target == nil || *v.Target != image.QueueOnly {
			t.Fatal(v, e)
		}
	})
	t.Run("parallel reusable helpers", func(t *testing.T) {
		headers := map[string]string{"X-Option": "owned"}
		helper := image.WithCacheOpts(image.CacheOpts{Headers: headers})
		var calls atomic.Int32
		client := cacheContractClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Option") != "owned" {
				t.Error(r.Header)
			}
			return cacheContractWire(200, io.NopCloser(strings.NewReader(`{"cached_images":[],"queued_images":[]}`))), nil
		})
		var wg sync.WaitGroup
		for i := 0; i < 12; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, e := image.New(client).GetImageCache(context.Background(), helper)
				if e != nil || v == nil {
					t.Error(v, e)
				}
			}()
		}
		wg.Wait()
		if calls.Load() != 12 || headers["X-Option"] != "owned" || len(headers) != 1 {
			t.Fatal(calls.Load(), headers)
		}
	})
}

func TestImageCacheExactNameResolutionAndDirectIdentity(t *testing.T) {
	for _, op := range cacheContractOperations() {
		if !op.perID {
			continue
		}
		t.Run(op.name+" literal escaped ID", func(t *testing.T) {
			id := "한글-7"
			var calls atomic.Int32
			client := cacheContractClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				path := cacheContractPrefix + "cache/"
				if op.name == "CachedImageNodes" {
					path += "nodes/"
				}
				want := path + "%ED%95%9C%EA%B8%80-7"
				if r.Method != op.method || r.URL.EscapedPath() != want || r.URL.RawQuery != "" || r.Body != nil {
					t.Error(r.Method, r.URL, r.Body)
				}
				return cacheContractWire(op.status, io.NopCloser(strings.NewReader(op.raw))), nil
			})
			v, e := cacheContractCall(image.New(client), context.Background(), op.name, resource.ID(id), cacheContractOptions{})
			if e != nil || v == nil || calls.Load() != 1 || v.ack != nil && v.ack.ImageID != id || v.nodes != nil && v.nodes.ImageID != id {
				t.Fatal(v, e, calls.Load())
			}
		})
		for _, id := range []string{"%2F", "query?id", "fragment#id", "slash/id", "back\\id", ".", ".."} {
			t.Run(op.name+" unsafe ID "+id, func(t *testing.T) {
				var calls atomic.Int32
				client := cacheContractClient(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					return nil, errors.New("unexpected HTTP")
				})
				value, err := cacheContractCall(image.New(client), context.Background(), op.name, resource.ID(id), cacheContractOptions{})
				if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
					t.Fatal(value, err, calls.Load())
				}
			})
		}
	}
	for _, mode := range []string{"unique last page", "UUID name", "missing", "ambiguous", "late HTTP404", "late read", "unsafe resolved ID", "cancel after lookup", "provider replaced", "target after lookup", "failed lookup and target"} {
		for _, op := range cacheContractOperations() {
			if !op.perID {
				continue
			}
			t.Run(op.name+" "+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("lookup cause")
				name := "needle"
				if mode == "UUID name" {
					name = "b2173dd3-7ad6-4362-baa6-a68bce3565cb"
				}
				var requests []string
				var bodies []*cacheContractBody
				client := cacheContractClient(nil)
				client.MoreHeaders = map[string]string{"X-Source": "captured"}
				client.HTTPClient.Transport = cacheContractTransport(func(r *http.Request) (*http.Response, error) {
					requests = append(requests, r.Method+" "+r.URL.String())
					if r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Option") != "lookup-and-cache" {
						t.Error(r.Header)
					}
					if r.URL.Path != cacheContractPrefix+"images" {
						if r.Method != op.method || r.URL.String() != cacheContractBase+op.path || r.Body != nil {
							t.Error("retargeted cache", r.Method, r.URL, r.Body)
						}
						return cacheContractWire(op.status, io.NopCloser(strings.NewReader(op.raw))), nil
					}
					first := r.URL.Query().Get("marker") == ""
					if first && r.URL.Query().Get("name") != name {
						t.Error(r.URL)
					}
					payload := fmt.Sprintf(`{"images":[{"id":"near","name":"needle-suffix"}],"next":"/v2/images?marker=second"}`)
					code := 200
					if !first {
						payload = fmt.Sprintf(`{"images":[{"id":"fixed","name":%q,"status":"killed","protected":true}]}`, name)
					}
					switch mode {
					case "UUID name":
						payload = fmt.Sprintf(`{"images":[{"id":"fixed","name":%q}]}`, name)
					case "missing":
						payload = `{"images":[]}`
					case "ambiguous":
						payload = `{"images":[{"id":"fixed","name":"needle"},{"id":"other","name":"needle"}]}`
					case "unsafe resolved ID":
						payload = `{"images":[{"id":"../escape","name":"needle"}]}`
					case "late HTTP404", "late read":
						if first {
							payload = `{"images":[{"id":"fixed","name":"needle"}],"next":"/v2/images?marker=second"}`
						} else {
							code = 404
							payload = "actual list404"
						}
					case "failed lookup and target":
						client.MoreHeaders = map[string]string{"x-image-cache-clear-target": "queue"}
						code = 404
						payload = "actual list404"
					}
					body := &cacheContractBody{Reader: strings.NewReader(payload)}
					if mode == "late read" && !first {
						code = 200
						body.Reader = cacheContractReader(func(p []byte) (int, error) { return copy(p, `{"images":`), cause })
					}
					if mode == "cancel after lookup" {
						body.onClose = func() { cancel(cause) }
					}
					if mode == "provider replaced" {
						body.onClose = func() { client.ProviderClient = &gophercloud.ProviderClient{} }
					}
					if mode == "target after lookup" {
						body.onClose = func() { client.MoreHeaders = map[string]string{"X-Image-Cache-Clear-Target": "cache"} }
					}
					bodies = append(bodies, body)
					return cacheContractWire(code, body), nil
				})
				o := cacheContractOptions{common: []image.CacheOption{image.WithCacheHeader("X-Option", "lookup-and-cache")}, deletion: []image.CacheDeleteOption{image.WithCacheDeleteHeader("X-Option", "lookup-and-cache")}, clear: nil}
				v, e := cacheContractCall(image.New(client), ctx, op.name, resource.Name(name), o)
				success := mode == "unique last page" || mode == "UUID name"
				if success {
					want := 3
					if mode == "UUID name" {
						want = 2
					}
					if e != nil || v == nil || len(requests) != want || v.ack != nil && v.ack.ImageID != "fixed" || v.nodes != nil && v.nodes.ImageID != "fixed" {
						t.Fatal(v, e, requests)
					}
				} else {
					if v != nil || e == nil {
						t.Fatal(v, e, requests)
					}
					for _, r := range requests {
						if !strings.Contains(r, "/images") {
							t.Fatal("mutation after failed lookup", requests)
						}
					}
				}
				if mode == "missing" && !errors.Is(e, resource.ErrNotFound) || mode == "ambiguous" && !errors.Is(e, resource.ErrAmbiguous) || mode == "unsafe resolved ID" && !errors.Is(e, resource.ErrInvalidOption) || strings.Contains(mode, "target") && !errors.Is(e, resource.ErrInvalidOption) || mode == "provider replaced" && !errors.Is(e, resource.ErrInvalidOption) || (mode == "late HTTP404" || mode == "failed lookup and target") && !gophercloud.ResponseCodeIs(e, 404) || (mode == "late read" || mode == "cancel after lookup") && !errors.Is(e, cause) || mode == "cancel after lookup" && !errors.Is(e, context.Canceled) {
					t.Fatal(v, e, requests)
				}
				for _, body := range bodies {
					if body.closes.Load() != 1 {
						t.Fatal("native lookup body ownership", body.closes.Load())
					}
				}
			})
		}
	}
}

func TestImageCacheAcceptedEvidenceAndFailureRetention(t *testing.T) {
	for _, op := range cacheContractOperations() {
		for _, mode := range []string{"read", "Close", "read Close cancel", "cancel only"} {
			t.Run(op.name+" accepted "+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("cache Read"), errors.New("cache Close"), errors.New("cache cancellation")
				raw := []byte(op.raw)
				if op.ack {
					raw = []byte{0, 255, 'x'}
				}
				body := &cacheContractBody{Reader: bytes.NewReader(raw)}
				if strings.Contains(mode, "read") {
					body.Reader = cacheContractReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
				}
				if strings.Contains(mode, "Close") {
					body.closeErr = closeCause
				}
				wire := cacheContractWire(op.status, body)
				body.onClose = func() {
					wire.Header.Set("X-Request-Id", "changed during Close")
					if strings.Contains(mode, "cancel") {
						cancel(cancelCause)
					}
				}
				var calls, retries atomic.Int32
				client := cacheContractClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return wire, nil })
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries.Add(1)
					return errors.New("accepted replay")
				}
				v, e := cacheContractCall(image.New(client), ctx, op.name, resource.ID("fixed"), cacheContractOptions{})
				proof := cacheContractProof(t, e, op.status, raw)
				if calls.Load() != 1 || retries.Load() != 0 || body.closes.Load() != 1 || op.ack && v == nil || !op.ack && v != nil {
					t.Fatal(op.name, v, e, calls.Load(), retries.Load(), body.closes.Load())
				}
				if strings.Contains(mode, "read") && !errors.Is(e, readCause) || strings.Contains(mode, "Close") && !errors.Is(e, closeCause) || strings.Contains(mode, "cancel") && (!errors.Is(e, context.Canceled) || !errors.Is(e, cancelCause)) {
					t.Fatal("accepted cause lost", e)
				}
				if op.ack {
					if !bytes.Equal(v.ack.Body, raw) || v.ack.StatusCode != op.status || v.ack.Header.Get("X-Request-Id") != "actual-cache" || op.perID && v.ack.ImageID != "fixed" || op.name == "ClearCache" && (v.ack.Target == nil || *v.ack.Target != image.CacheBoth) {
						t.Fatal(v.ack)
					}
					v.ack.Body[0] = '!'
					v.ack.Header.Set("X-Request-Id", "caller")
					if proof.Body[0] != raw[0] || proof.Header.Get("X-Request-Id") != "actual-cache" {
						t.Fatal("ack/proof alias", v.ack, proof)
					}
				}
			})
		}
	}
	for _, op := range cacheContractOperations() {
		for _, code := range []int{200, 202, 204, 206, 403, 404, 409, 500} {
			if code == op.status || op.name == "CacheDeleteImage" && code == 404 {
				continue
			}
			t.Run(fmt.Sprintf("%s actual%d", op.name, code), func(t *testing.T) {
				body := &cacheContractBody{Reader: strings.NewReader("native status proof")}
				var calls atomic.Int32
				client := cacheContractClient(func(r *http.Request) (*http.Response, error) { calls.Add(1); return cacheContractWire(code, body), nil })
				v, e := cacheContractCall(image.New(client), context.Background(), op.name, resource.ID("fixed"), cacheContractOptions{})
				var native gophercloud.ErrUnexpectedResponseCode
				var accepted *resource.ResponseError
				expected := []int{op.status}
				if op.name == "CacheDeleteImage" {
					expected = append(expected, 404)
				}
				if v != nil || !errors.As(e, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, expected) || native.Method != op.method || native.URL != cacheContractBase+op.path || string(native.Body) != "native status proof" || native.ResponseHeader.Get("X-Request-Id") != "actual-cache" || errors.As(e, &accepted) || calls.Load() != 1 || body.closes.Load() != 1 {
					t.Fatal(v, e, native, calls.Load(), body.closes.Load())
				}
			})
		}
	}
	for _, op := range cacheContractOperations() {
		t.Run(op.name+" callback expanded codes owned then rejected", func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("unexpected Read"), errors.New("unexpected Close"), errors.New("unexpected cancel")
			unexpected := 200
			if op.status == 200 {
				unexpected = 202
			}
			var calls, retries atomic.Int32
			var bodies []*cacheContractBody
			client := cacheContractClient(func(*http.Request) (*http.Response, error) {
				n := calls.Add(1)
				code, raw := 503, "original503"
				if n == 2 {
					code, raw = unexpected, "unexpected accepted bytes"
				}
				body := &cacheContractBody{Reader: strings.NewReader(raw)}
				if n == 2 {
					body.Reader = cacheContractReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
					body.closeErr = closeCause
					body.onClose = func() { cancel(cancelCause) }
				}
				bodies = append(bodies, body)
				return cacheContractWire(code, body), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, e error, _ uint) error {
				retries.Add(1)
				if !gophercloud.ResponseCodeIs(e, 503) {
					return e
				}
				o.OkCodes = []int{unexpected}
				return nil
			}
			v, e := cacheContractCall(image.New(client), ctx, op.name, resource.ID("fixed"), cacheContractOptions{})
			var native gophercloud.ErrUnexpectedResponseCode
			var accepted *resource.ResponseError
			expected := []int{op.status}
			if op.name == "CacheDeleteImage" {
				expected = append(expected, 404)
			}
			if v != nil || !errors.As(e, &native) || native.Actual != unexpected || !reflect.DeepEqual(native.Expected, expected) || string(native.Body) != "unexpected accepted bytes" || native.ResponseHeader.Get("X-Request-Id") != "actual-cache" || !errors.Is(e, readCause) || !errors.Is(e, closeCause) || !errors.Is(e, context.Canceled) || !errors.Is(e, cancelCause) || errors.As(e, &accepted) || calls.Load() != 2 || retries.Load() != 1 {
				t.Fatal(v, e, native, calls.Load(), retries.Load())
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
}

func TestImageCacheProviderHooksAndCauseSafeMissing(t *testing.T) {
	for _, mode := range []string{"default", "explicit true", "nil replacement", "explicit false", "false native retry"} {
		t.Run("physical DELETE404 "+mode, func(t *testing.T) {
			var calls, retries atomic.Int32
			var bodies []*cacheContractBody
			client := cacheContractClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if r.Method != "DELETE" || r.URL.String() != cacheContractBase+"cache/fixed" || r.Body != nil {
					t.Error(r.Method, r.URL, r.Body)
				}
				code := 404
				if mode == "false native retry" && n == 2 {
					code = 204
				}
				body := &cacheContractBody{Reader: strings.NewReader("physical bytes")}
				bodies = append(bodies, body)
				return cacheContractWire(code, body), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, e error, _ uint) error {
				retries.Add(1)
				if mode == "false native retry" && gophercloud.ResponseCodeIs(e, 404) {
					return nil
				}
				return e
			}
			var o []image.CacheDeleteOption
			switch mode {
			case "explicit true":
				o = []image.CacheDeleteOption{image.WithCacheDeleteIgnoreMissing(true)}
			case "nil replacement":
				o = []image.CacheDeleteOption{image.WithCacheDeleteIgnoreMissing(false), image.WithCacheDeleteOpts(image.CacheDeleteOpts{})}
			case "explicit false", "false native retry":
				o = []image.CacheDeleteOption{image.WithCacheDeleteIgnoreMissing(false)}
			}
			v, e := image.New(client).CacheDeleteImage(context.Background(), resource.ID("fixed"), o...)
			switch mode {
			case "explicit false":
				if v != nil || !gophercloud.ResponseCodeIs(e, 404) || calls.Load() != 1 || retries.Load() != 1 {
					t.Fatal(v, e, calls.Load(), retries.Load())
				}
			case "false native retry":
				if e != nil || v == nil || v.StatusCode != 204 || calls.Load() != 2 || retries.Load() != 1 {
					t.Fatal(v, e, calls.Load(), retries.Load())
				}
			default:
				if v != nil || e != nil || calls.Load() != 1 || retries.Load() != 0 {
					t.Fatal(v, e, calls.Load(), retries.Load())
				}
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
	for _, mode := range []string{"read", "Close", "cancel", "read Close cancel"} {
		t.Run("owned404 failure "+mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("404 Read"), errors.New("404 Close"), errors.New("404 cancellation")
			raw := []byte("physical404 partial evidence")
			body := &cacheContractBody{Reader: bytes.NewReader(raw)}
			if strings.Contains(mode, "read") {
				body.Reader = cacheContractReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
			}
			if strings.Contains(mode, "Close") {
				body.closeErr = closeCause
			}
			if strings.Contains(mode, "cancel") {
				body.onClose = func() { cancel(cancelCause) }
			}
			var calls, retries atomic.Int32
			client := cacheContractClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return cacheContractWire(404, body), nil })
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries.Add(1)
				return errors.New("404 retry")
			}
			v, e := image.New(client).CacheDeleteImage(ctx, resource.ID("fixed"))
			cacheContractProof(t, e, 404, raw)
			if v != nil || calls.Load() != 1 || retries.Load() != 0 || body.closes.Load() != 1 || strings.Contains(mode, "read") && !errors.Is(e, readCause) || strings.Contains(mode, "Close") && !errors.Is(e, closeCause) || strings.Contains(mode, "cancel") && (!errors.Is(e, context.Canceled) || !errors.Is(e, cancelCause)) {
				t.Fatal(v, e, calls.Load(), retries.Load(), body.closes.Load())
			}
		})
	}
	for _, mode := range []string{"transport nested404", "callback nested404", "reauth nested404"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New("nested policy cause")
			nested := gophercloud.ErrUnexpectedResponseCode{Method: "DELETE", URL: "not-wire://cache", Expected: []int{204}, Actual: 404, Body: []byte("nested404")}
			var calls, retries atomic.Int32
			body := &cacheContractBody{Reader: strings.NewReader("native failure")}
			client := cacheContractClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				if mode == "transport nested404" {
					return nil, errors.Join(cause, nested)
				}
				code := 503
				if mode == "reauth nested404" {
					code = 401
				}
				return cacheContractWire(code, body), nil
			})
			if mode == "callback nested404" {
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries.Add(1)
					return errors.Join(cause, nested)
				}
			}
			if mode == "reauth nested404" {
				client.ReauthFunc = func(context.Context) error { retries.Add(1); return errors.Join(cause, nested) }
			}
			v, e := image.New(client).CacheDeleteImage(context.Background(), resource.ID("fixed"))
			if v != nil || e == nil || calls.Load() != 1 {
				t.Fatal(v, e, calls.Load())
			}
			if mode == "reauth nested404" {
				var native *gophercloud.ErrUnableToReauthenticate
				if !errors.As(e, &native) || !gophercloud.ResponseCodeIs(native.ErrOriginal, 401) || !errors.Is(native.ErrReauth, cause) || !gophercloud.ResponseCodeIs(native.ErrReauth, 404) {
					t.Fatal(e, native)
				}
			} else if !errors.Is(e, cause) || mode == "callback nested404" && !gophercloud.ResponseCodeIs(e, 503) || mode == "transport nested404" && !gophercloud.ResponseCodeIs(e, 404) {
				t.Fatal(e)
			}
			if mode != "transport nested404" && body.closes.Load() != 1 {
				t.Fatal(body.closes.Load())
			}
		})
	}
	t.Run("configured prebody native hooks and original provider", func(t *testing.T) {
		var calls, reauth, backoff, retries atomic.Int32
		var bodies []*cacheContractBody
		transportCause := errors.New("prebody transport")
		client := cacheContractClient(nil)
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		client.ReauthFunc = func(context.Context) error { reauth.Add(1); client.SetToken("reauth"); return nil }
		client.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
			backoff.Add(1)
			client.SetToken("backoff")
			return nil
		}
		client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, e error, _ uint) error {
			retries.Add(1)
			if !o.KeepResponseBody || o.JSONBody != nil || o.JSONResponse != nil || o.RawBody != nil {
				t.Error(o)
			}
			if gophercloud.ResponseCodeIs(e, 503) {
				client.SetToken("retry503")
				return nil
			}
			if errors.Is(e, transportCause) {
				client.SetToken("retrytransport")
				return nil
			}
			return e
		}
		originalProvider, originalRetry := client.ProviderClient, reflect.ValueOf(client.RetryFunc).Pointer()
		client.HTTPClient.Transport = cacheContractTransport(func(r *http.Request) (*http.Response, error) {
			i := int(calls.Add(1)) - 1
			codes := []int{401, 429, 503, 0, 202}
			tokens := []string{"initial", "reauth", "backoff", "retry503", "retrytransport"}
			if i >= len(codes) {
				return nil, errors.New("replay")
			}
			if r.Method != "PUT" || r.URL.String() != cacheContractBase+"cache/fixed" || r.Body != nil || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != tokens[i] {
				t.Error(r.Method, r.URL, r.Header, r.Body)
			}
			if codes[i] == 0 {
				return nil, transportCause
			}
			body := &cacheContractBody{Reader: strings.NewReader("ack")}
			bodies = append(bodies, body)
			return cacheContractWire(codes[i], body), nil
		})
		v, e := image.New(client).QueueImage(context.Background(), resource.ID("fixed"))
		if e != nil || v == nil || v.StatusCode != 202 || calls.Load() != 5 || reauth.Load() != 1 || backoff.Load() != 1 || retries.Load() != 2 || client.ProviderClient != originalProvider || reflect.ValueOf(client.RetryFunc).Pointer() != originalRetry {
			t.Fatal(v, e, calls.Load(), reauth.Load(), backoff.Load(), retries.Load())
		}
		for _, b := range bodies {
			if b.closes.Load() != 1 {
				t.Fatal(b.closes.Load())
			}
		}
	})
	for _, change := range []string{"JSON null", "JSON object", "KeepResponseBody", "JSONResponse", "RawBody", "unsupported JSONBody"} {
		t.Run("retry guard "+change, func(t *testing.T) {
			cause := errors.New("callback cause")
			var calls, retries, borrowedReads atomic.Int32
			body := &cacheContractBody{Reader: strings.NewReader("original503")}
			borrowed := &cacheContractBody{Reader: cacheContractReader(func([]byte) (int, error) { borrowedReads.Add(1); return 0, io.EOF })}
			client := cacheContractClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Body != nil || r.Method != "DELETE" || r.URL.String() != cacheContractBase+"cache/fixed" {
					t.Error(r.Method, r.URL, r.Body)
				}
				return cacheContractWire(503, body), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, e error, _ uint) error {
				retries.Add(1)
				if !gophercloud.ResponseCodeIs(e, 503) {
					t.Error(e)
				}
				switch change {
				case "JSON null":
					o.JSONBody = json.RawMessage("null")
				case "JSON object":
					o.JSONBody = map[string]any{}
				case "KeepResponseBody":
					o.KeepResponseBody = false
				case "JSONResponse":
					o.JSONResponse = new(any)
				case "RawBody":
					o.RawBody = borrowed
				case "unsupported JSONBody":
					o.JSONBody = make(chan int)
				}
				return cause
			}
			v, e := image.New(client).CacheDeleteImage(context.Background(), resource.ID("fixed"))
			var native gophercloud.ErrUnexpectedResponseCode
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) || !errors.Is(e, cause) || !errors.As(e, &native) || native.Actual != 503 || string(native.Body) != "original503" || calls.Load() != 1 || retries.Load() != 1 || body.closes.Load() != 1 || borrowedReads.Load() != 0 || borrowed.closes.Load() != 0 {
				t.Fatal(v, e, native, calls.Load(), retries.Load(), body.closes.Load())
			}
			if change == "unsupported JSONBody" {
				var encoding *json.UnsupportedTypeError
				if !errors.As(e, &encoding) {
					t.Fatal("encoding cause lost", e)
				}
			}
		})
	}
	for _, redirect := range []string{"same target", "foreign origin", "changed path", "changed query", "changed method"} {
		t.Run("configured redirect "+redirect, func(t *testing.T) {
			var calls, redirects atomic.Int32
			var bodies []*cacheContractBody
			client := cacheContractClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				body := &cacheContractBody{Reader: strings.NewReader("redirect proof")}
				bodies = append(bodies, body)
				if n == 2 {
					return cacheContractWire(202, body), nil
				}
				code := 307
				target := cacheContractBase + "cache/fixed"
				switch redirect {
				case "foreign origin":
					target = "https://foreign.invalid/cache/fixed"
				case "changed path":
					target = cacheContractBase + "cache/other"
				case "changed query":
					target += "?q=x"
				case "changed method":
					code = 303
				}
				wire := cacheContractWire(code, body)
				wire.Header.Set("Location", target)
				return wire, nil
			})
			client.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects.Add(1); return nil }
			v, e := image.New(client).QueueImage(context.Background(), resource.ID("fixed"))
			if redirect == "same target" {
				if e != nil || v == nil || calls.Load() != 2 || redirects.Load() != 1 {
					t.Fatal(v, e, calls.Load(), redirects.Load())
				}
			} else if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 1 {
				t.Fatal(v, e, calls.Load(), redirects.Load())
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
	for _, code := range []int{202, 204} {
		for _, native := range []bool{true, false} {
			t.Run(fmt.Sprintf("existing native/manual images delete %d/%t", code, native), func(t *testing.T) {
				body := &cacheContractBody{Reader: strings.NewReader("native raw")}
				var calls atomic.Int32
				client := cacheContractClient(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					if r.Method != "DELETE" || r.URL.String() != cacheContractBase+"images/fixed" {
						t.Error(r.Method, r.URL)
					}
					return cacheContractWire(code, body), nil
				})
				var e error
				if native {
					e = sdkimages.New(client).Delete(context.Background(), "fixed")
				} else {
					e = image.New(client).Images.Delete(context.Background(), resource.ID("fixed"))
				}
				if e != nil || calls.Load() != 1 || body.closes.Load() != 1 {
					t.Fatal(e, calls.Load(), body.closes.Load())
				}
			})
		}
	}
}
