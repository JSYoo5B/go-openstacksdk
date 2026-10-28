package images_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/images"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type novaImageCall struct {
	method, path, query, body string
	header                    http.Header
}
type novaImageCalls struct {
	sync.Mutex
	values []novaImageCall
}

func (c *novaImageCalls) add(v novaImageCall) {
	c.Lock()
	defer c.Unlock()
	c.values = append(c.values, v)
}
func (c *novaImageCalls) snapshot() []novaImageCall {
	c.Lock()
	defer c.Unlock()
	return append([]novaImageCall(nil), c.values...)
}
func novaImageAPI(t *testing.T, version string, dispatch func(*http.Request) (int, string)) (*images.API, *testcloud.Cloud, *novaImageCalls) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := &novaImageCalls{}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		calls.add(novaImageCall{r.Method, r.URL.EscapedPath(), r.URL.RawQuery, string(raw), r.Header.Clone()})
		w.Header().Set("X-Request-Id", "nova-image-request")
		code, body := dispatch(r)
		testcloud.JSON(w, code, body)
	})
	client := cloud.Client("compute", "/nova/v2.1/")
	client.Microversion = version
	return images.New(client), cloud, calls
}
func novaImageField(t *testing.T, v *images.ImageRecord, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		t.Fatalf("%s: %#v want %s", key, v, want)
	}
}
func novaImageIDs(t *testing.T, api *images.API, opts ...images.ListOption) ([]string, error) {
	t.Helper()
	var ids []string
	for v, err := range api.ListRecords(context.Background(), opts...) {
		if err != nil {
			return ids, err
		}
		var id string
		if err := json.Unmarshal(v.Resource.Body["id"], &id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func TestPythonNovaImageExplicitVersionBounds(t *testing.T) {
	for _, version := range []string{"", "2.1", "2.35", "2.36", "2.38", "2.39", "2.104", "latest", "1.0", "3.0", "2.035", "2.35.0"} {
		t.Run(version, func(t *testing.T) {
			api, _, calls := novaImageAPI(t, version, func(r *http.Request) (int, string) {
				if r.Method == http.MethodDelete || r.Method == http.MethodPut || r.Method == http.MethodPost {
					return 202, "opaque"
				}
				if strings.HasSuffix(r.URL.Path, "/metadata") {
					return 200, `{"metadata":{}}`
				}
				if strings.HasSuffix(r.URL.Path, "/detail") {
					return 200, `{"images":[]}`
				}
				return 200, `{"image":{"id":"a"}}`
			})
			regular := version == "" || version == "2.1" || version == "2.35"
			metadata := regular || version == "2.36" || version == "2.38"
			before := len(calls.snapshot())
			_, err := api.GetRecord(context.Background(), "a")
			if (err == nil) != regular {
				t.Fatal(err)
			}
			if !regular && len(calls.snapshot()) != before {
				t.Fatal("unsupported Get made HTTP")
			}
			_, err = novaImageIDs(t, api)
			if (err == nil) != regular {
				t.Fatal(err)
			}
			_, err = api.FindRecord(context.Background(), "a")
			if (err == nil) != regular {
				t.Fatal(err)
			}
			err = api.DeleteRecord(context.Background(), "a")
			if (err == nil) != regular {
				t.Fatal(err)
			}
			before = len(calls.snapshot())
			_, err = api.FetchMetadata(context.Background(), "a")
			if (err == nil) != metadata {
				t.Fatal(err)
			}
			_, err = api.GetMetadata(context.Background(), "a")
			if (err == nil) != metadata {
				t.Fatal(err)
			}
			_, err = api.SetMetadata(context.Background(), "a", nil)
			if (err == nil) != metadata {
				t.Fatal(err)
			}
			err = api.DeleteMetadata(context.Background(), "a")
			if (err == nil) != metadata {
				t.Fatal(err)
			}
			if !metadata && len(calls.snapshot()) != before {
				t.Fatal("unsupported metadata made HTTP")
			}
			for _, call := range calls.snapshot() {
				if version == "" {
					if call.header.Get("X-Openstack-Nova-Api-Version") != "" {
						t.Fatal(call)
					}
				} else if call.header.Get("X-Openstack-Nova-Api-Version") != version {
					t.Fatal("version changed", call)
				}
			}
			if api.RawClient().Microversion != version {
				t.Fatal("implicit downgrade")
			}
		})
	}
	api, _, calls := novaImageAPI(t, "2.36", func(*http.Request) (int, string) { return 404, "missing" })
	err := api.DeleteRecord(context.Background(), "a")
	if !errors.Is(err, resource.ErrUnsupported) || len(calls.snapshot()) != 0 {
		t.Fatal("legacy guard must precede ignore missing", err, calls.snapshot())
	}
}

func TestPythonNovaImageGetProjectionAliasesAndStatuses(t *testing.T) {
	body := `{"image":{"id":"server","name":false,"links":[{"rel":"self"}],"created":"raw date","created_at":42,"min_disk":9,"minDisk":"12","minRam":true,"progress":1.9,"OS-EXT-IMG-SIZE:size":9007199254740993,"metadata":["bad"],"unknown":{"raw":9007199254740993}}}`
	api, _, calls := novaImageAPI(t, "2.35", func(*http.Request) (int, string) { return 200, body })
	v, err := api.GetRecord(context.Background(), "selected")
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"id": `"server"`, "name": `false`, "created_at": `42`, "min_disk": `12`, "min_ram": `true`, "progress": `1`, "size": `9007199254740993`, "metadata": `{}`, "updated_at": `null`, "status": `null`, "location": `null`} {
		novaImageField(t, v, key, want)
	}
	if len(v.Resource.Body) != 12 || v.StatusCode != 200 || calls.snapshot()[0].query != "" || string(v.Wire.Body["metadata"]) != `["bad"]` {
		t.Fatal(v, calls.snapshot())
	}
	v.Resource.Body["size"][0] = '1'
	if string(v.Wire.Body["OS-EXT-IMG-SIZE:size"]) != `9007199254740993` || string(v.Envelope) != body {
		t.Fatal("body evidence alias")
	}
	v.Header.Set("X-Request-Id", "changed")
	if v.Wire.Header.Get("X-Request-Id") != "nova-image-request" || v.Resource.Header.Get("X-Request-Id") != "nova-image-request" {
		t.Fatal("header evidence alias")
	}
	for _, item := range []struct {
		body string
		ok   bool
	}{{`{}`, true}, {"", true}, {"{bad", true}, {`{"name":"root","metadata":null}`, true}, {`[]`, false}, {`null`, false}, {`{"image":null}`, false}} {
		t.Run(item.body, func(t *testing.T) {
			api, _, _ := novaImageAPI(t, "2.1", func(*http.Request) (int, string) { return 200, item.body })
			v, err := api.GetRecord(context.Background(), "seed")
			if (err == nil) != item.ok {
				t.Fatal(v, err)
			}
			if item.ok {
				novaImageField(t, v, "id", `"seed"`)
			} else {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != item.body {
					t.Fatal(v, err, proof)
				}
			}
		})
	}
	for _, code := range []int{200, 201, 202, 204, 299, 301, 399, 400, 403, 404, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			api, _, _ := novaImageAPI(t, "2.35", func(*http.Request) (int, string) { return code, `{}` })
			v, err := api.GetRecord(context.Background(), "a")
			if code < 400 {
				if err != nil || v == nil || v.StatusCode != code {
					t.Fatal(v, err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if v != nil || !errors.As(err, &native) || native.Actual != code {
					t.Fatal(v, err)
				}
			}
		})
	}
}

func TestPythonNovaImageListQueryDetailLocalFiltersAndCaps(t *testing.T) {
	api, _, calls := novaImageAPI(t, "2.35", func(*http.Request) (int, string) {
		return 200, `{"images":[{"id":"a","created":"raw","progress":"4","metadata":{"owner":"ci","extra":true}},{"id":"b","created":"other","progress":4,"metadata":{"owner":"ci"}}]}`
	})
	ids, err := novaImageIDs(t, api, images.WithListQuery("min_disk", []any{true, nil, "4"}), images.WithListQuery("minDisk", 99), images.WithListQuery("min_ram", nil), images.WithListQuery("changes_since", "now"), images.WithListQuery("server", map[string]any{"x": 1}), images.WithListQuery("name", false), images.WithListQuery("status", "ACTIVE"), images.WithListQuery("type", []string{"snapshot", "backup"}), images.WithListQuery("created_at", "raw"), images.WithListQuery("progress", 4), images.WithListQuery("metadata", map[string]any{"owner": "ci"}), images.WithListQuery("unknown", make(chan int)), images.WithListPaginated(false))
	if err != nil || !reflect.DeepEqual(ids, []string{"a"}) {
		t.Fatal(ids, err)
	}
	call := calls.snapshot()[0]
	want := "changes-since=now&minDisk=True&minDisk=4&name=False&server=x&status=ACTIVE&type=snapshot&type=backup"
	if call.path != "/nova/v2.1/images/detail" || call.query != want {
		t.Fatal(call)
	}
	ids, err = novaImageIDs(t, api, images.WithListDetails(false), images.WithListLimit(0), images.WithListMaxItems(1), images.WithListQuery("created_at", "other"))
	if err != nil || len(ids) != 0 {
		t.Fatal("cap must count raw rows before filter", ids, err)
	}
	call = calls.snapshot()[1]
	if call.path != "/nova/v2.1/images" || call.query != "limit=1" {
		t.Fatal(call)
	}
	t.Run("alias order on list rows", func(t *testing.T) {
		api, _, _ := novaImageAPI(t, "2.35", func(*http.Request) (int, string) {
			return 200, `{"images":[{"id":"a","created_at":"first","created":"last","minDisk":5,"min_disk":7}]}`
		})
		for v, err := range api.ListRecords(context.Background(), images.WithListPaginated(false)) {
			if err != nil {
				t.Fatal(err)
			}
			novaImageField(t, v, "created_at", `"last"`)
			novaImageField(t, v, "min_disk", `7`)
		}
	})
	t.Run("singleton and malformed capped tail", func(t *testing.T) {
		api, _, _ := novaImageAPI(t, "2.35", func(*http.Request) (int, string) { return 200, `{"images":[{"id":"a"},42]}` })
		ids, err := novaImageIDs(t, api, images.WithListMaxItems(1))
		if err != nil || !reflect.DeepEqual(ids, []string{"a"}) {
			t.Fatal(ids, err)
		}
		api, _, _ = novaImageAPI(t, "2.35", func(*http.Request) (int, string) { return 200, `{"images":{"id":"singleton"}}` })
		ids, err = novaImageIDs(t, api, images.WithListPaginated(false))
		if err != nil || !reflect.DeepEqual(ids, []string{"singleton"}) {
			t.Fatal(ids, err)
		}
	})
}

func TestPythonNovaImageListPagingAndWireMarker(t *testing.T) {
	for _, shape := range []string{"next", "array", "http", "dictionary"} {
		t.Run(shape, func(t *testing.T) {
			api, _, calls := novaImageAPI(t, "2.35", func(r *http.Request) (int, string) {
				if r.URL.Query().Get("marker") == "next" {
					return 200, `{"images":[{"id":"b"}]}`
				}
				switch shape {
				case "next":
					return 200, `{"images":[{"id":"a"}],"next":"?marker=next"}`
				case "array":
					return 200, `{"images":[{"id":"a"}],"images_links":[{"rel":"next","href":"?marker=next"}]}`
				case "dictionary":
					return 200, `{"images":[{"id":"a"}],"links":{"next":"?marker=next"}}`
				default:
					return 200, `{"images":[{"id":"a"}]}`
				}
			})
			if shape == "http" {
				cloud := api.RawClient().ProviderClient
				original := cloud.HTTPClient.Transport
				cloud.HTTPClient.Transport = testcloud.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
					response, err := original.RoundTrip(r)
					if err == nil && r.URL.Query().Get("marker") == "" {
						response.Header.Set("Link", `<?marker=next>; rel="next"`)
					}
					return response, err
				})
			}
			ids, err := novaImageIDs(t, api)
			if err != nil || !reflect.DeepEqual(ids, []string{"a", "b"}) || len(calls.snapshot()) != 2 {
				t.Fatal(ids, err, calls.snapshot())
			}
		})
	}
	t.Run("short page fallback and independent marker", func(t *testing.T) {
		api, _, calls := novaImageAPI(t, "2.35", func(r *http.Request) (int, string) {
			if r.URL.Query().Get("marker") != "" {
				return 200, `{"images":[]}`
			}
			return 200, `{"images":[{"id":"original"}]}`
		})
		for v, err := range api.ListRecords(context.Background(), images.WithListLimit(2)) {
			if err != nil {
				t.Fatal(err)
			}
			v.Resource.Body["id"] = json.RawMessage(`"changed"`)
			v.Wire.Body["id"] = json.RawMessage(`"also changed"`)
		}
		got := calls.snapshot()
		if len(got) != 2 || got[1].query != "limit=2&marker=original" {
			t.Fatal(got)
		}
	})
	t.Run("early break skips continuation", func(t *testing.T) {
		api, _, calls := novaImageAPI(t, "2.35", func(*http.Request) (int, string) { return 200, `{"images":[{"id":"a"}],"next":"?marker=next"}` })
		for _, err := range api.ListRecords(context.Background()) {
			if err != nil {
				t.Fatal(err)
			}
			break
		}
		if len(calls.snapshot()) != 1 {
			t.Fatal(calls.snapshot())
		}
	})
	t.Run("route escape retains first record and proof", func(t *testing.T) {
		api, _, calls := novaImageAPI(t, "2.35", func(*http.Request) (int, string) { return 200, `{"images":[{"id":"a"}],"next":"/wrong?marker=next"}` })
		ids, err := novaImageIDs(t, api)
		var proof *resource.ResponseError
		if !reflect.DeepEqual(ids, []string{"a"}) || !errors.As(err, &proof) || proof.StatusCode != 200 || len(calls.snapshot()) != 1 {
			t.Fatal(ids, err, calls.snapshot())
		}
	})
}

func TestPythonNovaImageFindSummaryAndDeleteMissing(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			api, _, calls := novaImageAPI(t, "2.35", func(r *http.Request) (int, string) {
				if strings.HasSuffix(r.URL.Path, "/ci") {
					return code, "missing"
				}
				if r.URL.Query().Get("marker") == "next" {
					return 200, `{"images":[{"id":"b","name":"other"}]}`
				}
				return 200, `{"images":[{"id":"a","name":"ci"}],"next":"?marker=next"}`
			})
			v, err := api.FindRecord(context.Background(), "ci")
			if err != nil {
				t.Fatal(err)
			}
			novaImageField(t, v, "id", `"a"`)
			got := calls.snapshot()
			if len(got) != 3 || got[1].path != "/nova/v2.1/images" || got[1].query != "name=ci" || got[2].query != "marker=next&name=ci" {
				t.Fatal(got)
			}
		})
	}
	for _, item := range []struct {
		name, body string
		strict     bool
		target     error
	}{{"missing", `[]`, false, nil}, {"strict", `[]`, true, resource.ErrNotFound}, {"duplicate", `[{"id":"a","name":"ci"},{"id":"b","name":"ci"}]`, false, resource.ErrAmbiguous}} {
		t.Run(item.name, func(t *testing.T) {
			api, _, _ := novaImageAPI(t, "2.35", func(r *http.Request) (int, string) {
				if strings.HasSuffix(r.URL.Path, "/ci") {
					return 404, "missing"
				}
				return 200, `{"images":` + item.body + `}`
			})
			v, err := api.FindRecord(context.Background(), "ci", images.WithFindIgnoreMissing(!item.strict))
			if item.target == nil {
				if v != nil || err != nil {
					t.Fatal(v, err)
				}
			} else if !errors.Is(err, item.target) {
				t.Fatal(v, err)
			}
		})
	}
	t.Run("find successful GET avoids list", func(t *testing.T) {
		api, _, calls := novaImageAPI(t, "2.35", func(*http.Request) (int, string) { return 200, `{}` })
		v, err := api.FindRecord(context.Background(), "ci")
		if err != nil || len(calls.snapshot()) != 1 {
			t.Fatal(v, err, calls.snapshot())
		}
		novaImageField(t, v, "id", `"ci"`)
	})
	t.Run("late find error overrides early match", func(t *testing.T) {
		api, _, calls := novaImageAPI(t, "2.35", func(r *http.Request) (int, string) {
			if strings.HasSuffix(r.URL.Path, "/ci") {
				return 404, "missing"
			}
			if r.URL.Query().Get("marker") == "next" {
				return 500, "late"
			}
			return 200, `{"images":[{"id":"a","name":"ci"}],"next":"?marker=next"}`
		})
		v, err := api.FindRecord(context.Background(), "ci")
		var native gophercloud.ErrUnexpectedResponseCode
		if v != nil || !errors.As(err, &native) || native.Actual != 500 || len(calls.snapshot()) != 3 {
			t.Fatal(v, err, calls.snapshot())
		}
	})
	for _, code := range []int{204, 299, 399, 404, 500} {
		t.Run("delete"+fmt.Sprint(code), func(t *testing.T) {
			api, _, calls := novaImageAPI(t, "2.35", func(*http.Request) (int, string) { return code, "opaque" })
			err := api.DeleteRecord(context.Background(), "a")
			if (err == nil) != (code < 400 || code == 404) {
				t.Fatal(err)
			}
			err = api.DeleteRecord(context.Background(), "a", images.WithDeleteIgnoreMissing(false))
			if (err == nil) != (code < 400) || len(calls.snapshot()) != 2 {
				t.Fatal(err, calls.snapshot())
			}
		})
	}
}

func TestPythonNovaImageMetadataFetchAliasSetDeleteContracts(t *testing.T) {
	api, _, calls := novaImageAPI(t, "2.38", func(r *http.Request) (int, string) {
		if r.Method == http.MethodGet {
			return 200, `{"metadata":{"owner":"ci","raw":9007199254740993},"id":"should not overlay","unknown":false}`
		}
		return 202, `{"metadata":{"owner":"server ignored"}}`
	})
	v, err := api.FetchMetadata(context.Background(), "folder/image%")
	if err != nil {
		t.Fatal(err)
	}
	novaImageField(t, v, "id", `"folder/image%"`)
	novaImageField(t, v, "metadata", `{"owner":"ci","raw":9007199254740993}`)
	if string(v.Wire.Body["id"]) != `"should not overlay"` {
		t.Fatal(v)
	}
	alias, err := api.GetMetadata(context.Background(), "folder/image%")
	if err != nil || !reflect.DeepEqual(alias.Resource.Body, v.Resource.Body) {
		t.Fatal(alias, err)
	}
	metadata := map[string]any{"owner": "ci", "empty": "", "none": nil, "count": json.Number("9007199254740993")}
	v, err = api.SetMetadata(context.Background(), "folder/image%", metadata, func(*images.ReadOpts) error { metadata["owner"] = "changed"; return nil })
	if err != nil {
		t.Fatal(err)
	}
	novaImageField(t, v, "metadata", `{"count":9007199254740993,"empty":"","none":null,"owner":"ci"}`)
	if v.Wire != nil {
		t.Fatal("set must not decode replacement metadata", v)
	}
	got := calls.snapshot()
	if len(got) != 3 || got[0].path != "/nova/v2.1/images/folder%2Fimage%25/metadata" || got[0].query != "" || got[2].method != http.MethodPost || got[2].body != `{"metadata":{"count":9007199254740993,"empty":"","none":null,"owner":"ci"}}` {
		t.Fatal(got)
	}
	if err := api.DeleteMetadata(context.Background(), "a"); err != nil {
		t.Fatal(err)
	}
	got = calls.snapshot()
	if got[3].method != http.MethodPut || got[3].body != `{"metadata":{}}` {
		t.Fatal(got)
	}
	before := len(got)
	if err := api.DeleteMetadata(context.Background(), "a", images.WithDeleteMetadataKeys()); err != nil || len(calls.snapshot()) != before {
		t.Fatal("explicit empty keys must be no-op", err, calls.snapshot())
	}
	keys := []string{"a/b%", "second", "a/b%"}
	option := images.WithDeleteMetadataKeys(keys...)
	keys[0] = "changed"
	if err := api.DeleteMetadata(context.Background(), "a", option); err != nil {
		t.Fatal(err)
	}
	got = calls.snapshot()
	if len(got) != before+2 || got[before].method != http.MethodDelete || got[before].path != "/nova/v2.1/images/a/metadata/a%2Fb%25" || got[before+1].path != "/nova/v2.1/images/a/metadata/second" {
		t.Fatal(got)
	}
	for _, item := range []struct {
		body  string
		ok    bool
		value string
	}{{`{}`, true, `null`}, {`{"metadata":null}`, true, `null`}, {`{"metadata":false}`, true, `{}`}, {"", false, ""}, {"{bad", false, ""}, {`null`, false, ""}, {`[]`, false, ""}} {
		t.Run("fetch"+item.body, func(t *testing.T) {
			api, _, _ := novaImageAPI(t, "2.38", func(*http.Request) (int, string) { return 200, item.body })
			v, err := api.FetchMetadata(context.Background(), "a")
			if (err == nil) != item.ok {
				t.Fatal(v, err)
			}
			if item.ok {
				novaImageField(t, v, "metadata", item.value)
			} else {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 {
					t.Fatal(v, err)
				}
			}
		})
	}
	t.Run("item missing is strict and stops", func(t *testing.T) {
		api, _, calls := novaImageAPI(t, "2.38", func(*http.Request) (int, string) { return 404, "missing" })
		err := api.DeleteMetadata(context.Background(), "a", images.WithDeleteMetadataKeys("one", "two"))
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != 404 || len(calls.snapshot()) != 1 {
			t.Fatal(err, calls.snapshot())
		}
	})
	t.Run("validate all keys before any mutation", func(t *testing.T) {
		api, _, calls := novaImageAPI(t, "2.38", func(*http.Request) (int, string) { return 202, "opaque" })
		err := api.DeleteMetadata(context.Background(), "a", images.WithDeleteMetadataKeys("one", ".."))
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls.snapshot()) != 0 {
			t.Fatal(err, calls.snapshot())
		}
	})
	t.Run("accepted source change stops later metadata keys", func(t *testing.T) {
		api, cloud, calls := novaImageAPI(t, "2.38", func(*http.Request) (int, string) { return 202, "opaque" })
		client := api.RawClient()
		original := client.ResourceBase
		defer func() { client.ResourceBase = original }()
		transport := cloud.Provider.HTTPClient.Transport
		cloud.SetTransport(testcloud.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			response, err := transport.RoundTrip(r)
			if err == nil {
				client.ResourceBase = client.Endpoint + "changed/"
			}
			return response, err
		}))
		err := api.DeleteMetadata(context.Background(), "a", images.WithDeleteMetadataKeys("one", "two"))
		var proof *resource.ResponseError
		if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 202 || len(calls.snapshot()) != 1 {
			t.Fatal(err, proof, calls.snapshot())
		}
	})
	t.Run("set nil uses empty and accepts opaque", func(t *testing.T) {
		api, _, calls := novaImageAPI(t, "2.38", func(*http.Request) (int, string) { return 399, "not JSON" })
		v, err := api.SetMetadata(context.Background(), "a", nil)
		if err != nil {
			t.Fatal(err)
		}
		novaImageField(t, v, "metadata", `{}`)
		if calls.snapshot()[0].body != `{"metadata":{}}` {
			t.Fatal(calls.snapshot())
		}
	})
}

func TestPythonNovaImageOwnedOptionsGuardsAndEvidence(t *testing.T) {
	api, _, calls := novaImageAPI(t, "2.35", func(r *http.Request) (int, string) {
		if r.Header.Get("X-Owned") != "old" {
			t.Error(r.Header)
		}
		return 200, `{"images":[{"id":"a","created":"ci"}]}`
	})
	limit := 2
	onePage := false
	details := false
	query := map[string]json.RawMessage{"created_at": json.RawMessage(`"ci"`)}
	headers := map[string]string{"X-Owned": "old"}
	option := images.WithListOpts(images.ListOpts{Limit: &limit, Paginated: &onePage, Details: &details, Query: query, Headers: headers})
	limit = 9
	details = true
	query["created_at"] = json.RawMessage(`"bad"`)
	headers["X-Owned"] = "changed"
	seq := api.ListRecords(context.Background(), option)
	if len(calls.snapshot()) != 0 {
		t.Fatal("eager iterator")
	}
	for range 2 {
		count := 0
		for v, err := range seq {
			if err != nil {
				t.Fatal(err)
			}
			count++
			novaImageField(t, v, "id", `"a"`)
		}
		if count != 1 {
			t.Fatal(count)
		}
	}
	if got := calls.snapshot(); len(got) != 2 || got[0].path != "/nova/v2.1/images" || got[0].query != "limit=2" {
		t.Fatal(got)
	}
	t.Run("retained callback cannot mutate applied options", func(t *testing.T) {
		var retained *images.ListOpts
		ids, err := novaImageIDs(t, api, func(c *images.ListOpts) error {
			retained = c
			c.Headers = map[string]string{"X-Owned": "old"}
			c.Query = map[string]json.RawMessage{"created_at": json.RawMessage(`"ci"`)}
			c.Paginated = &onePage
			return nil
		}, func(*images.ListOpts) error {
			retained.Headers["X-Owned"] = "bad"
			retained.Query["created_at"] = json.RawMessage(`"bad"`)
			return nil
		})
		if err != nil || !reflect.DeepEqual(ids, []string{"a"}) {
			t.Fatal(ids, err)
		}
	})
	for _, item := range []struct {
		name string
		call func() error
	}{{"nil read", func() error { _, err := api.GetRecord(context.Background(), "a", nil); return err }}, {"nil delete", func() error { return api.DeleteRecord(context.Background(), "a", nil) }}, {"nil find", func() error { _, err := api.FindRecord(context.Background(), "a", nil); return err }}, {"nil list", func() error { _, err := novaImageIDs(t, api, nil); return err }}, {"nil metadata delete", func() error { return api.DeleteMetadata(context.Background(), "a", nil) }}, {"negative limit", func() error { _, err := novaImageIDs(t, api, images.WithListLimit(-1)); return err }}, {"negative cap", func() error { _, err := novaImageIDs(t, api, images.WithListMaxItems(-1)); return err }}, {"protected header", func() error {
		_, err := api.GetRecord(context.Background(), "a", images.WithReadHeaders(map[string]string{"X-Auth-Token": "bad"}))
		return err
	}}, {"invalid identifier", func() error { _, err := api.GetRecord(context.Background(), ".."); return err }}} {
		t.Run(item.name, func(t *testing.T) {
			before := len(calls.snapshot())
			err := item.call()
			if !errors.Is(err, resource.ErrInvalidOption) || len(calls.snapshot()) != before {
				t.Fatal(err, calls.snapshot())
			}
		})
	}
	t.Run("callback cancellation preserves both causes", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		cause, callbackErr := errors.New("caller cause"), errors.New("callback error")
		before := len(calls.snapshot())
		_, err := api.FindRecord(ctx, "ci", func(*images.FindOpts) error { cancel(cause); return callbackErr })
		if !errors.Is(err, cause) || !errors.Is(err, callbackErr) || !errors.Is(err, context.Canceled) || len(calls.snapshot()) != before {
			t.Fatal(err, calls.snapshot())
		}
	})
	t.Run("callback source mutation preserves both errors", func(t *testing.T) {
		client := api.RawClient()
		original := client.Microversion
		defer func() { client.Microversion = original }()
		before := len(calls.snapshot())
		callbackErr := errors.New("callback error")
		_, err := api.GetRecord(context.Background(), "a", func(*images.ReadOpts) error { client.Microversion = "2.104"; return callbackErr })
		if !errors.Is(err, callbackErr) || !errors.Is(err, resource.ErrInvalidOption) || len(calls.snapshot()) != before {
			t.Fatal(err, calls.snapshot())
		}
	})
	t.Run("accepted read close failure has real receipt", func(t *testing.T) {
		api, cloud, _ := novaImageAPI(t, "2.35", func(*http.Request) (int, string) { return 200, `{}` })
		readErr, closeErr := errors.New("read failed"), errors.New("close failed")
		cloud.SetTransport(testcloud.RoundTripperFunc(func(*http.Request) (*http.Response, error) {
			return testcloud.StreamResponse(200, &novaImageFaultBody{data: []byte(`{"image":`), read: readErr, close: closeErr}, http.Header{"X-Request-Id": {"partial"}}), nil
		}))
		v, err := api.GetRecord(context.Background(), "a")
		var proof *resource.ResponseError
		if v == nil || !errors.Is(err, readErr) || !errors.Is(err, closeErr) || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-Id") != "partial" || string(v.Envelope) != `{"image":` {
			t.Fatal(v, err, proof)
		}
	})
	t.Run("member escapes once", func(t *testing.T) {
		api, _, calls := novaImageAPI(t, "2.35", func(*http.Request) (int, string) { return 200, `{}` })
		_, err := api.GetRecord(context.Background(), "folder/image%", images.WithReadHeaders(map[string]string{"X-Owned": "old"}))
		if err != nil || calls.snapshot()[0].path != "/nova/v2.1/images/folder%2Fimage%25" {
			t.Fatal(err, calls.snapshot())
		}
	})
}

type novaImageFaultBody struct {
	data            []byte
	read, close     error
	closes          int
	onRead, onClose func()
}

func (b *novaImageFaultBody) Read(p []byte) (int, error) {
	if len(b.data) > 0 {
		n := copy(p, b.data)
		b.data = b.data[n:]
		if b.onRead != nil {
			b.onRead()
		}
		return n, nil
	}
	if b.onRead != nil {
		b.onRead()
	}
	return 0, b.read
}
func (b *novaImageFaultBody) Close() error {
	b.closes++
	if b.onClose != nil {
		b.onClose()
	}
	return b.close
}

func TestPythonNovaImageRejectedResponseFailuresStopDeleteAndFind(t *testing.T) {
	for _, operation := range []string{"delete default", "delete strict", "find"} {
		statuses := []int{404}
		if operation == "find" {
			statuses = []int{400, 403, 404}
		}
		for _, status := range statuses {
			for _, kind := range []string{"read", "close", "read and close", "cancel after read", "cancel after close", "source restored by Close", "retry callback"} {
				t.Run(fmt.Sprintf("%s/%d/%s", operation, status, kind), func(t *testing.T) {
					api, cloud, _ := novaImageAPI(t, "2.35", func(*http.Request) (int, string) { return 200, `{}` })
					calls := 0
					readCause, closeCause, cancelCause := errors.New("physical rejected Read"), errors.New("physical rejected Close"), errors.New("caller canceled in rejected body")
					retryCause := errors.New("rejected callback failed")
					if kind == "retry callback" {
						api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return retryCause }
					}
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					body := &novaImageFaultBody{data: []byte("physical rejection"), read: io.EOF}
					if kind == "read" || kind == "read and close" {
						body.read = readCause
					}
					if kind == "close" || kind == "read and close" {
						body.close = closeCause
					}
					if kind == "cancel after read" {
						body.onRead = func() { cancel(cancelCause) }
					}
					if kind == "cancel after close" {
						body.onClose = func() { cancel(cancelCause) }
					}
					originalEndpoint := api.RawClient().Endpoint
					if kind == "source restored by Close" {
						body.onRead = func() { api.RawClient().Endpoint = originalEndpoint + "changed/" }
						body.onClose = func() { api.RawClient().Endpoint = originalEndpoint }
					}
					cloud.SetTransport(testcloud.RoundTripperFunc(func(*http.Request) (*http.Response, error) {
						calls++
						if calls > 1 {
							return testcloud.Response(200, `{"images":[{"id":"found","name":"ci"}]}`, make(http.Header)), nil
						}
						return testcloud.StreamResponse(status, body, http.Header{"X-Request-Id": {"rejected-proof"}}), nil
					}))
					var err error
					if operation == "find" {
						value, findErr := api.FindRecord(ctx, "ci")
						err = findErr
						if value != nil {
							t.Fatal("faulty member rejection produced a lookup result", value, err)
						}
					} else if strings.Contains(operation, "strict") {
						err = api.DeleteRecord(ctx, "ci", images.WithDeleteIgnoreMissing(false))
					} else {
						err = api.DeleteRecord(ctx, "ci")
					}
					if err == nil || calls != 1 || body.closes != 1 {
						t.Fatal("faulty rejection established absence or retried", err, calls, body.closes)
					}
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != status || string(native.Body) != "physical rejection" || native.ResponseHeader.Get("X-Request-Id") != "rejected-proof" {
						t.Fatal("native rejection proof lost", err, native)
					}
					if kind == "read" || kind == "read and close" {
						if !errors.Is(err, readCause) {
							t.Fatal(err)
						}
					}
					if kind == "close" || kind == "read and close" {
						if !errors.Is(err, closeCause) {
							t.Fatal(err)
						}
					}
					if kind == "retry callback" && !errors.Is(err, retryCause) {
						t.Fatal(err)
					}
					if strings.HasPrefix(kind, "cancel") {
						if !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) {
							t.Fatal(err)
						}
					}
					if kind == "source restored by Close" {
						if !errors.Is(err, resource.ErrInvalidOption) || api.RawClient().Endpoint != originalEndpoint {
							t.Fatal("source read failure lost after restoration", err)
						}
					}
				})
			}
		}
	}
}

func TestPythonNovaImageCleanRejectionsRemainEligibleAfterNativeRetryHook(t *testing.T) {
	for _, operation := range []string{"delete default", "delete strict", "find"} {
		statuses := []int{404}
		if operation == "find" {
			statuses = []int{400, 403, 404}
		}
		for _, status := range statuses {
			t.Run(fmt.Sprintf("%s/%d", operation, status), func(t *testing.T) {
				api, cloud, _ := novaImageAPI(t, "2.35", func(*http.Request) (int, string) { return 200, `{}` })
				ctx := context.Background()
				calls, retries := 0, 0
				body := &novaImageFaultBody{data: []byte("clean rejection"), read: io.EOF}
				api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
					retries++
					return original
				}
				cloud.SetTransport(testcloud.RoundTripperFunc(func(*http.Request) (*http.Response, error) {
					calls++
					if calls == 1 {
						return testcloud.StreamResponse(status, body, http.Header{"X-Request-Id": {"clean-proof"}}), nil
					}
					return testcloud.Response(200, `{"images":[{"id":"found","name":"ci"}]}`, make(http.Header)), nil
				}))
				if operation == "find" {
					value, err := api.FindRecord(ctx, "ci")
					if err != nil || calls != 2 {
						t.Fatal(value, err, calls)
					}
					novaImageField(t, value, "id", `"found"`)
				} else {
					var err error
					if strings.Contains(operation, "strict") {
						err = api.DeleteRecord(ctx, "ci", images.WithDeleteIgnoreMissing(false))
					} else {
						err = api.DeleteRecord(ctx, "ci")
					}
					if !strings.Contains(operation, "strict") {
						if err != nil {
							t.Fatal("identical native retry rejection was marked mixed", err)
						}
					} else {
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "clean rejection" {
							t.Fatal(err, native)
						}
					}
					if calls != 1 {
						t.Fatal("stop hook replayed DELETE", calls)
					}
				}
				if retries != 1 || body.closes != 1 {
					t.Fatal(retries, body.closes)
				}
			})
		}
	}
}
