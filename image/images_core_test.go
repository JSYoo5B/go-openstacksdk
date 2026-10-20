package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestImageCoreFixedGetAndCanonicalRawOwnership(t *testing.T) {
	for _, id := range []string{"이미지% ?#:parent", strings.Repeat("한", 300)} {
		calls := 0
		service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Method != "GET" || req.Body != nil || req.URL.RawQuery != "" || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(id) {
				t.Fatal(req.Method, req.URL)
			}
			response := taskCoreJSON(req, 200, `{"id":"different","name":"","status":"future","visibility":"future","owner":null,"container_format":"future","disk_format":"future","checksum":"","os_hash_algo":"future","os_hash_value":"","self":"https://foreign/image","file":"foreign","schema":"foreign","direct_url":"foreign","stores":"b,a,b","protected":false,"os_hidden":true,"size":9007199254740993,"virtual_size":-1,"min_disk":0,"min_ram":-3,"created_at":"literal date","updated_at":"","tags":["repeat","repeat",""],"locations":[{"url":"foreign","metadata":{"n":900719925474099312345}}],"properties":false,"metadata":[1],"ID":"alias","links":"foreign","custom":{"n":1.00000000000000000001}}`)
			response.Header.Set("OpenStack-image-import-methods", "a,b")
			response.Header.Set("OpenStack-image-store-ids", "s1,s2")
			return response, nil
		}))
		value, err := service.GetImage(context.Background(), resource.ID(id))
		if err != nil || value == nil || calls != 1 || *value.ID != "different" || *value.Name != "" || value.Owner != nil || *value.Protected || !*value.Hidden || *value.Size != 9007199254740993 || *value.VirtualSize != -1 || *value.MinRAM != -3 || *value.CreatedAt != "literal date" || *value.UpdatedAt != "" || value.Links != nil || value.StatusCode != 200 || value.Header.Get("X-Task-Proof") != "actual" {
			t.Fatal(value, err, calls)
		}
		if !reflect.DeepEqual(value.Tags, []string{"repeat", "repeat", ""}) || string(value.Properties["properties"]) != "false" || string(value.Properties["metadata"]) != "[1]" || string(value.Properties["ID"]) != `"alias"` || string(value.Locations[0].Metadata["n"]) != "900719925474099312345" {
			t.Fatal(value)
		}
		for _, key := range []string{"id", "protected", "size", "created_at", "tags", "locations", "OpenStack-image-import-methods", "OpenStack-image-store-ids"} {
			if _, ok := value.Properties[key]; ok {
				t.Fatal("canonical or header field projected", key)
			}
		}
		value.Properties["custom"][0] = '['
		if string(value.Body["custom"]) != `{"n":1.00000000000000000001}` {
			t.Fatal("property projection aliases raw Body")
		}
		value.Locations[0].Metadata["n"][0] = '8'
		if bytes.Contains(value.Body["locations"], []byte("800719")) || bytes.Contains(value.Locations[0].Body["metadata"], []byte("800719")) {
			t.Fatal("location values alias body")
		}
	}
}

func TestImageCoreCanonicalPresenceAndAtomicDecoder(t *testing.T) {
	for _, body := range []string{`{}`, `{"name":null,"protected":null,"size":null,"tags":null,"locations":null,"created_at":null}`} {
		var v ImageInfo
		if err := json.Unmarshal([]byte(body), &v); err != nil || v.Name != nil || v.Protected != nil || v.Size != nil || v.Tags != nil || v.Locations != nil || v.CreatedAt != nil || v.Properties == nil {
			t.Fatal(v, err)
		}
	}
	var v ImageInfo
	if err := json.Unmarshal([]byte(`{"name":"old","tags":[],"locations":[],"extra":null}`), &v); err != nil || v.Tags == nil || v.Locations == nil {
		t.Fatal(v, err)
	}
	before, _ := json.Marshal(v)
	fields := []string{"id", "name", "status", "visibility", "owner", "container_format", "disk_format", "checksum", "os_hash_algo", "os_hash_value", "self", "file", "schema", "direct_url", "stores", "created_at", "updated_at"}
	for _, key := range fields {
		if err := json.Unmarshal([]byte(fmt.Sprintf("{%q:42}", key)), &v); err == nil {
			t.Fatal("accepted canonical string type", key)
		}
		after, _ := json.Marshal(v)
		if !bytes.Equal(before, after) {
			t.Fatal("partial receiver mutation", key)
		}
	}
	for _, body := range []string{`{"protected":"false"}`, `{"os_hidden":1}`, `{"size":1.5}`, `{"virtual_size":9223372036854775808}`, `{"min_disk":"0"}`, `{"min_ram":false}`, `{"tags":[null]}`, `{"tags":{}}`, `{"locations":[null]}`, `{"locations":[{"metadata":false}]}`, `null`, `[]`} {
		if err := json.Unmarshal([]byte(body), &v); err == nil {
			t.Fatal("accepted wrong model", body)
		}
		after, _ := json.Marshal(v)
		if !bytes.Equal(before, after) {
			t.Fatal("partial receiver mutation", body)
		}
	}
	if err := json.Unmarshal([]byte(`{"ID":"raw","owner_id":"raw","is_hidden":false,"createdAt":42,"properties":[null]}`), &v); err != nil || v.ID != nil || v.Owner != nil || v.Hidden != nil || v.CreatedAt != nil || len(v.Properties) != 5 {
		t.Fatal(v, err)
	}
}

func TestImageCoreCompletePreflight(t *testing.T) {
	for _, ref := range []resource.Ref{resource.Ref{}, resource.ID("."), resource.ID(".."), resource.ID("a/b"), resource.ID("a\\\\b"), resource.ID("a\n"), resource.ID(string([]byte{0xff})), resource.Name(string([]byte{0xff}))} {
		calls, callbacks := 0, 0
		service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, 200, "{}"), nil }))
		value, err := service.GetImage(context.Background(), ref, func(o *GetImageOpts) error { callbacks++; return nil })
		if value != nil || err == nil || calls != 0 || callbacks != 0 {
			t.Fatal(value, err, calls, callbacks, ref)
		}
	}
	for _, ctx := range []context.Context{nil, func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }()} {
		calls, callbacks := 0, 0
		service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil }))
		if v, e := service.GetImage(ctx, resource.ID("id"), func(o *GetImageOpts) error { callbacks++; return nil }); v != nil || e == nil {
			t.Fatal(v, e)
		}
		if v, e := service.AllImages(ctx, func(o *ListImagesOpts) error { callbacks++; return nil }); v != nil || e == nil || calls != 0 || callbacks != 0 {
			t.Fatal(v, e, calls, callbacks)
		}
	}
	for _, alter := range []func(*gophercloud.ServiceClient){func(c *gophercloud.ServiceClient) { c.ProviderClient = nil }, func(c *gophercloud.ServiceClient) { c.Type = "compute" }, func(c *gophercloud.ServiceClient) { c.MoreHeaders = map[string]string{"X-Auth-Token": "owned"} }} {
		callbacks := 0
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			t.Fatal("source preflight reached HTTP")
			return nil, nil
		})
		alter(client)
		if values, err := New(client).AllImages(context.Background(), func(o *ListImagesOpts) error { callbacks++; return nil }); values != nil || err == nil || callbacks != 0 {
			t.Fatal(values, err, callbacks)
		}
	}
	var nilService *Service
	if v, e := nilService.GetImage(context.Background(), resource.ID("id")); v != nil || e == nil {
		t.Fatal(v, e)
	}
	if v, e := nilService.AllImages(context.Background()); v != nil || e == nil {
		t.Fatal(v, e)
	}
}

func TestImageCoreAdvertisedPagingAndQueryRestoration(t *testing.T) {
	calls := 0
	var client *gophercloud.ServiceClient
	initial := url.Values{"limit": {"0"}, "name": {"line\n"}, "protected": {"false"}, "os_hidden": {"false"}, "size_min": {"0"}, "tag": {"a", "", "a"}, "sort_key": {"name", "id"}, "sort_dir": {"asc", "desc"}, "extra": {"x", "y"}}
	nextQuery := copyImageQuery(initial)
	nextQuery["tag"] = []string{"a"}
	nextQuery["sort_key"] = []string{"id"}
	nextQuery["sort_dir"] = []string{"desc"}
	nextQuery["extra"] = []string{"y"}
	client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		query := req.URL.Query()
		expected := copyImageQuery(initial)
		if calls > 1 {
			expected.Set("marker", fmt.Sprintf("m%d", calls-1))
		}
		if !reflect.DeepEqual(query, expected) || req.Header.Get("X-Option") != "fixed" || req.Header.Get("X-Source") != map[bool]string{false: "first", true: "later"}[calls > 1] || req.Header.Get("X-Auth-Token") != map[bool]string{false: "first-token", true: "second-token"}[calls > 1] {
			t.Fatal(req.URL, req.Header, expected)
		}
		switch calls {
		case 1:
			nextQuery.Set("marker", "m1")
		case 2:
			nextQuery.Set("marker", "m2")
		case 3:
			return taskCoreJSON(req, 200, `{"images":[{"id":"last"}],"first":"foreign","schema":false}`), nil
		default:
			t.Fatal("extra request")
		}
		rows := []any{map[string]any{"id": "first"}}
		if calls == 2 {
			rows = []any{}
		}
		body, _ := json.Marshal(map[string]any{"images": rows, "next": "/v2/images?" + nextQuery.Encode()})
		return taskCoreJSON(req, 200, string(body)), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "first"}
	count := 0
	for value, err := range New(client).ListImages(context.Background(), WithListImagesOpts(ListImagesOpts{Headers: map[string]string{"X-Option": "fixed"}, Limit: taskOptionPointer(0), Name: taskOptionPointer("line\n"), Protected: taskOptionPointer(false), Hidden: taskOptionPointer(false), SizeMin: taskOptionPointer(int64(0)), Tags: []string{"a", "", "a"}, SortKeys: []string{"name", "id"}, SortDirs: []string{"asc", "desc"}, Filters: url.Values{"extra": {"x", "y"}}})) {
		if err != nil || value == nil {
			t.Fatal(value, err)
		}
		count++
		client.MoreHeaders["X-Source"] = "later"
		client.ProviderClient.SetToken("second-token")
	}
	if calls != 3 || count != 2 {
		t.Fatal(calls, count)
	}
}

func TestImageCorePagingGuardsAndLocalConsumption(t *testing.T) {
	body := `{"images":[{"id":"first"},{"tags":[null]}],"next":42}`
	calls := 0
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, 200, body), nil }))
	if values, err := service.AllImages(context.Background(), WithListImagesMaxItems(1)); err != nil || len(values) != 1 {
		t.Fatal(values, err)
	}
	for value, err := range service.ListImages(context.Background()) {
		if err != nil || *value.ID != "first" {
			t.Fatal(value, err)
		}
		break
	}
	values, err := service.AllImages(context.Background())
	if values != nil || err == nil || calls != 3 {
		t.Fatal(values, err, calls)
	}
	taskCoreProof(t, err, 200, body)
	for _, link := range []any{42, "https://foreign/v2/images?marker=m", "//glance.example/reverse/glance/v2/images?marker=m", "/reverse/glance/v2/tasks?marker=m", "/reverse/glance/v2/./images?marker=m", "/reverse/glance/v2/%69mages?marker=m", "?marker=m&extra=changed", "?marker=m", "?marker=m&extra=x&extra=y", "?marker=m&marker=n&extra=a&extra=b", "?marker=&extra=a&extra=b", "?marker=m&extra=a%ZZ", "?marker=m&extra=a&extra=b#foreign", "?marker=m&extra=a&extra=b\n"} {
		t.Run(fmt.Sprint(link), func(t *testing.T) {
			count := 0
			wire, _ := json.Marshal(map[string]any{"images": []any{}, "next": link})
			values, err := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
				count++
				return taskCoreJSON(req, 200, string(wire)), nil
			})).AllImages(context.Background(), WithListImagesFilter("extra", "a", "b"))
			if values != nil || err == nil || count != 1 {
				t.Fatal(values, err, count)
			}
			taskCoreProof(t, err, 200, string(wire))
		})
	}
	for _, body := range []string{`{}`, `{"images":null}`, `{"Images":[]}`, `{"images":{}}`, `{"images":[null]}`, `null`, `[]`, `{"images":[],`, string([]byte{'{', '"', 'i', 'm', 'a', 'g', 'e', 's', '"', ':', '[', ']', ',', '"', 'x', '"', ':', '"', 0xff, '"', '}'})} {
		values, err := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreJSON(req, 200, body), nil })).AllImages(context.Background())
		if values != nil || err == nil {
			t.Fatal(values, err, body)
		}
		taskCoreProof(t, err, 200, body)
	}
	t.Run("same marker cycle", func(t *testing.T) {
		body := `{"images":[],"next":"?marker=initial"}`
		calls := 0
		values, err := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, 200, body), nil })).AllImages(context.Background(), WithListImagesMarker("initial"))
		if values != nil || !errors.Is(err, resource.ErrPaginationCycle) || calls != 1 {
			t.Fatal(values, err, calls)
		}
		taskCoreProof(t, err, 200, body)
	})
	t.Run("encoded captured prefix", func(t *testing.T) {
		calls := 0
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return taskCoreJSON(req, 200, `{"images":[],"next":"https://glance.example/reverse%20proxy/glance/v2/images?marker=m"}`), nil
			}
			return taskCoreJSON(req, 200, `{"images":[]}`), nil
		})
		client.Endpoint = "https://glance.example/reverse%20proxy/glance/v2/"
		if values, err := New(client).AllImages(context.Background()); values == nil || err != nil || calls != 2 {
			t.Fatal(values, err, calls)
		}
	})
	t.Run("SinglePage skips next only", func(t *testing.T) {
		calls := 0
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			return taskCoreJSON(req, 200, `{"images":[{}],"next":42}`), nil
		})
		if values, err := New(client).AllImages(context.Background(), WithListImagesSinglePage(true)); err != nil || len(values) != 1 || calls != 1 {
			t.Fatal(values, err, calls)
		}
	})
}

func TestImageCoreExactNameSourceAndOwnedFailures(t *testing.T) {
	t.Run("Name exact across pages then fresh GET", func(t *testing.T) {
		calls := 0
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			switch calls {
			case 1:
				if req.URL.Query().Get("name") != "Parent" {
					t.Fatal(req.URL)
				}
				return taskCoreJSON(req, 200, `{"images":[{"id":"lower","name":"parent"}],"next":"/v2/images?marker=second"}`), nil
			case 2:
				return taskCoreJSON(req, 200, `{"images":[{"id":"selected","name":"Parent"}]}`), nil
			case 3:
				if req.URL.Path != "/reverse/glance/v2/images/selected" {
					t.Fatal(req.URL)
				}
				return taskCoreJSON(req, 200, `{"id":"passive","name":"fresh"}`), nil
			default:
				t.Fatal("extra Name request")
				return nil, nil
			}
		})
		v, e := New(client).GetImage(context.Background(), resource.Name("Parent"))
		if e != nil || *v.ID != "passive" || *v.Name != "fresh" || calls != 3 {
			t.Fatal(v, e, calls)
		}
	})
	for _, tc := range []struct {
		body  string
		cause error
	}{{`{"images":[]}`, resource.ErrNotFound}, {`{"images":[{"id":"one","name":"Parent"},{"id":"two","name":"Parent"}]}`, resource.ErrAmbiguous}} {
		calls := 0
		v, e := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, 200, tc.body), nil })).GetImage(context.Background(), resource.Name("Parent"))
		if v != nil || !errors.Is(e, tc.cause) || calls != 1 {
			t.Fatal(v, e, calls)
		}
	}
	for _, field := range []string{"endpoint", "version", "provider"} {
		t.Run(field, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil })
			client.ResourceBase = client.Endpoint
			v, e := New(client).GetImage(context.Background(), resource.Name("Parent"), func(o *GetImageOpts) error {
				switch field {
				case "endpoint":
					client.Endpoint = "https://glance.example/changed/"
				case "version":
					client.Microversion = "changed"
				case "provider":
					client.ProviderClient = &gophercloud.ProviderClient{}
				}
				return nil
			})
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(v, e, calls)
			}
		})
	}
	t.Run("accepted read close cancellation", func(t *testing.T) {
		readErr, closeErr, cause := errors.New("read"), errors.New("close"), errors.New("cancel cause")
		ctx, cancel := context.WithCancelCause(context.Background())
		body := &taskCoreBody{reader: &taskCoreReader{body: "prefix", err: readErr, action: func() { cancel(cause) }}, closeErr: closeErr}
		calls := 0
		v, e := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, 200, body), nil })).GetImage(ctx, resource.ID("id"))
		if v != nil || !errors.Is(e, readErr) || !errors.Is(e, closeErr) || !errors.Is(e, cause) || !errors.Is(e, context.Canceled) || calls != 1 || body.closes != 1 {
			t.Fatal(v, e, calls, body.closes)
		}
		taskCoreProof(t, e, 200, "prefix")
	})
	for _, code := range []int{201, 202, 204, 206, 403, 404, 429, 500} {
		v, e := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreJSON(req, code, "native"), nil })).GetImage(context.Background(), resource.ID("id"))
		var proof *resource.ResponseError
		if v != nil || !gophercloud.ResponseCodeIs(e, code) || errors.As(e, &proof) {
			t.Fatal(v, e, code)
		}
	}
	t.Run("native accepted code expansion", func(t *testing.T) {
		calls := 0
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return taskCoreJSON(req, 503, "retry"), nil
			}
			return taskCoreJSON(req, 202, "expanded"), nil
		})
		client.ProviderClient.RetryFunc = func(ctx context.Context, m, u string, o *gophercloud.RequestOpts, e error, n uint) error {
			o.OkCodes = append(o.OkCodes, 202)
			return nil
		}
		v, e := New(client).GetImage(context.Background(), resource.ID("id"))
		var native gophercloud.ErrUnexpectedResponseCode
		if v != nil || !errors.As(e, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, []int{200}) || string(native.Body) != "expanded" || calls != 2 {
			t.Fatal(v, e, native, calls)
		}
	})
	t.Run("accepted body source drift", func(t *testing.T) {
		var client *gophercloud.ServiceClient
		wire := `{"images":[]}`
		body := &taskCoreBody{reader: &taskCoreReader{body: wire, err: io.EOF, action: func() { client.Microversion = "changed" }}}
		client = taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreHTTP(req, 200, body), nil })
		values, e := New(client).AllImages(context.Background())
		if values != nil || !errors.Is(e, resource.ErrInvalidOption) || body.closes != 1 {
			t.Fatal(values, e, body.closes)
		}
		taskCoreProof(t, e, 200, wire)
	})
}
