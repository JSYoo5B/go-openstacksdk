package image_test

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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const imageContractObject = `{"id":"response-id","name":"literal\nname","status":"future/status","visibility":"future/visibility","owner":"owner","container_format":"future/container","disk_format":"future/disk","checksum":"literal-checksum","os_hash_algo":"future/hash","os_hash_value":"literal-value","self":"https://passive.invalid/self","file":"https://passive.invalid/file","schema":"https://passive.invalid/schema","direct_url":"https://passive.invalid/data","stores":"one,one, two","created_at":"literal-created","updated_at":"literal-updated","protected":false,"os_hidden":true,"size":9007199254740993,"virtual_size":9223372036854775807,"min_disk":-1,"min_ram":0,"tags":["","repeat","repeat","line\n"],"locations":[{"url":"literal:%2F ?#","metadata":{"huge":1e1000,"integer":9007199254740995},"x-row":true}],"properties":[1e1000],"metadata":17,"owner_id":false,"ID":42,"links":17,"hw_vif_multiqueue_enabled":"false","x-number":9007199254740995}`
const imageContractList = `{"images":[` + imageContractObject + `],"first":"https://passive.invalid/first","schema":42}`

func imageContractRows(s *image.Service, ctx context.Context, iterator bool, opts ...image.ListImagesOption) ([]*image.ImageInfo, error) {
	if !iterator {
		return s.AllImages(ctx, opts...)
	}
	out := make([]*image.ImageInfo, 0)
	for v, e := range s.ListImages(ctx, opts...) {
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, nil
}
func imageContractJSONResponse(raw string) *http.Response {
	return taskContractsWire(200, &taskContractsBody{Reader: strings.NewReader(raw)})
}
func imageContractPtr[T any](v T) *T { return &v }
func imageContractCall(s *image.Service, ctx context.Context, op string) ([]*image.ImageInfo, error) {
	if op == "Get" {
		v, e := s.GetImage(ctx, resource.ID("fixed"))
		if v == nil {
			return nil, e
		}
		return []*image.ImageInfo{v}, e
	}
	return imageContractRows(s, ctx, op == "List")
}

func TestImageContractsGetRouteAndReference(t *testing.T) {
	for _, version := range []string{"", "2.0", "2.18"} {
		t.Run("literal ID version "+version, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", "/unused/catalog/")
			client.ResourceBase = cloud.Server.URL + taskContractsPrefix
			client.Microversion = version
			client.MoreHeaders = map[string]string{"X-Source": "first"}
			id := "literal:%2F ?#한글"
			var calls, callbacks atomic.Int32
			headers := map[string]string{"X-Option": "owned"}
			opts := image.WithGetImageHeaders(headers)
			headers["X-Option"] = "late"
			cloud.Mux.HandleFunc(taskContractsPrefix, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				raw, _ := io.ReadAll(r.Body)
				wantVersion := ""
				if version != "" {
					wantVersion = "image " + version
				}
				if r.Method != "GET" || r.URL.EscapedPath() != taskContractsPrefix+"images/"+url.PathEscape(id) || r.URL.RawQuery != "" || len(raw) != 0 || r.Header.Get("X-Source") != "first" || r.Header.Get("X-Option") != "owned" || r.Header.Get("X-Final") != "last" || r.Header.Get("X-Auth-Token") != "live" || r.Header.Get("OpenStack-API-Version") != wantVersion {
					t.Error(r.Method, r.URL, r.Header, string(raw))
				}
				w.Header().Set("X-Request-Id", "actual-task")
				w.Header().Set("Location", "https://foreign.invalid/retarget")
				w.Header().Set("OpenStack-image-import-methods", "header-only")
				_, _ = io.WriteString(w, imageContractObject)
			})
			v, e := image.New(client).GetImage(context.Background(), resource.ID(id), opts, image.WithGetImageHeader("X-Final", "last"), func(o *image.GetImageOpts) error {
				callbacks.Add(1)
				client.MoreHeaders["X-Source"] = "after callback"
				cloud.Provider.SetToken("live")
				return nil
			})
			if e != nil || v == nil || *v.ID != "response-id" || v.StatusCode != 200 || v.Header.Get("Location") != "https://foreign.invalid/retarget" || v.Body["openstack-image-import-methods"] != nil || calls.Load() != 1 || callbacks.Load() != 1 {
				t.Fatal(v, e, calls.Load(), callbacks.Load())
			}
		})
	}
	t.Run("long safe literal without UUID or length gate", func(t *testing.T) {
		id := strings.Repeat("한", 300) + ":% ?#"
		var calls atomic.Int32
		c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.URL.EscapedPath() != taskContractsPrefix+"images/"+url.PathEscape(id) || r.URL.RawQuery != "" || r.Body != nil {
				t.Error(r.URL, r.Body)
			}
			return imageContractJSONResponse(`{}`), nil
		})
		v, e := image.New(c).GetImage(context.Background(), resource.ID(id))
		if e != nil || v == nil || v.ID != nil || calls.Load() != 1 {
			t.Fatal(v, e, calls.Load())
		}
	})
	for _, id := range []string{"", ".", "..", "a/b", "a\\b", "line\n", string([]byte{0xff})} {
		t.Run("unsafe ID "+id, func(t *testing.T) {
			var calls, callbacks atomic.Int32
			c := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected") })
			v, e := image.New(c).GetImage(context.Background(), resource.ID(id), func(*image.GetImageOpts) error { callbacks.Add(1); return nil })
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 || callbacks.Load() != 0 {
				t.Fatal(v, e, calls.Load(), callbacks.Load())
			}
		})
	}
	t.Run("exact Name all pages then fresh raw GET", func(t *testing.T) {
		cloud := testcloud.New(t)
		c := cloud.Client("image", taskContractsPrefix)
		c.MoreHeaders = map[string]string{"X-Source": "captured"}
		var calls, callbacks atomic.Int32
		cloud.Mux.HandleFunc(taskContractsPrefix, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Request-Id", "actual-task")
			n := calls.Add(1)
			if r.Method != "GET" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Option") != "owned" {
				t.Error(r.Method, r.URL, r.Header)
			}
			switch n {
			case 1:
				if r.URL.Path != taskContractsPrefix+"images" || r.URL.Query().Get("name") != "exact" {
					t.Error(r.URL)
				}
				cloud.Provider.SetToken("refreshed")
				_, _ = io.WriteString(w, `{"images":[{"id":"chosen","name":"exact"},{"id":"prefix","name":"exact suffix"}],"next":"/v2/images?name=exact&marker=second"}`)
			case 2:
				if r.URL.Path != taskContractsPrefix+"images" || r.URL.Query().Get("marker") != "second" || r.Header.Get("X-Auth-Token") != "refreshed" {
					t.Error(r.URL, r.Header)
				}
				_, _ = io.WriteString(w, `{"images":[{"id":"other","name":"other"}]}`)
			case 3:
				if r.URL.Path != taskContractsPrefix+"images/chosen" || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "refreshed" {
					t.Error(r.URL, r.Header)
				}
				_, _ = io.WriteString(w, imageContractObject)
			default:
				t.Error("lookup or Get replay", n)
			}
		})
		v, e := image.New(c).GetImage(context.Background(), resource.Name("exact"), func(o *image.GetImageOpts) error {
			callbacks.Add(1)
			o.Headers["X-Option"] = "owned"
			c.MoreHeaders["X-Source"] = "after callback"
			return nil
		})
		if e != nil || v == nil || *v.ID != "response-id" || *v.Name != "literal\nname" || calls.Load() != 3 || callbacks.Load() != 1 {
			t.Fatal(v, e, calls.Load(), callbacks.Load())
		}
	})
	for _, mode := range []string{"missing", "ambiguous", "late HTTP", "late native model", "unsafe selected ID"} {
		t.Run("Name "+mode, func(t *testing.T) {
			var calls, gets atomic.Int32
			c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if r.URL.Path != taskContractsPrefix+"images" {
					gets.Add(1)
					t.Error(r.URL)
				}
				code, raw := 200, `{"images":[]}`
				switch mode {
				case "ambiguous":
					raw = `{"images":[{"id":"one","name":"exact"},{"id":"two","name":"exact"}]}`
				case "unsafe selected ID":
					raw = `{"images":[{"id":"../bad","name":"exact"}]}`
				case "late HTTP", "late native model":
					if n == 1 {
						raw = `{"images":[{"id":"chosen","name":"exact"}],"next":"/v2/images?marker=second"}`
					} else if mode == "late HTTP" {
						code, raw = 503, "late lookup"
					} else {
						raw = `{"images":[{"id":"other","created_at":"not-native-time"}]}`
					}
				}
				return taskContractsWire(code, &taskContractsBody{Reader: strings.NewReader(raw)}), nil
			})
			v, e := image.New(c).GetImage(context.Background(), resource.Name("exact"))
			if v != nil || e == nil || gets.Load() != 0 {
				t.Fatal(v, e, calls.Load(), gets.Load())
			}
			if mode == "missing" && !errors.Is(e, resource.ErrNotFound) || mode == "ambiguous" && !errors.Is(e, resource.ErrAmbiguous) || mode == "unsafe selected ID" && !errors.Is(e, resource.ErrInvalidOption) || mode == "late HTTP" && !gophercloud.ResponseCodeIs(e, 503) {
				t.Fatal(e)
			}
		})
	}
}

func TestImageContractsCanonicalMetadataAndRawOwnership(t *testing.T) {
	t.Run("all canonical fields raw projection and independence", func(t *testing.T) {
		raw := []byte(imageContractObject)
		wire := taskContractsWire(200, &taskContractsBody{Reader: bytes.NewReader(raw)})
		wire.Header.Set("OpenStack-image-store-ids", "wire-only")
		c := taskContractsClient(func(*http.Request) (*http.Response, error) { return wire, nil })
		v, e := image.New(c).GetImage(context.Background(), resource.ID("chosen"))
		if e != nil || v == nil {
			t.Fatal(v, e)
		}
		for _, field := range []struct {
			name string
			got  *string
			want string
		}{{"id", v.ID, "response-id"}, {"name", v.Name, "literal\nname"}, {"status", v.Status, "future/status"}, {"visibility", v.Visibility, "future/visibility"}, {"owner", v.Owner, "owner"}, {"container", v.ContainerFormat, "future/container"}, {"disk", v.DiskFormat, "future/disk"}, {"checksum", v.Checksum, "literal-checksum"}, {"hashalgo", v.OSHashAlgo, "future/hash"}, {"hashvalue", v.OSHashValue, "literal-value"}, {"self", v.Self, "https://passive.invalid/self"}, {"file", v.File, "https://passive.invalid/file"}, {"schema", v.Schema, "https://passive.invalid/schema"}, {"direct", v.DirectURL, "https://passive.invalid/data"}, {"stores", v.Stores, "one,one, two"}, {"created", v.CreatedAt, "literal-created"}, {"updated", v.UpdatedAt, "literal-updated"}} {
			if field.got == nil || *field.got != field.want {
				t.Error(field.name, field.got)
			}
		}
		if v.Protected == nil || *v.Protected || v.Hidden == nil || !*v.Hidden || v.Size == nil || *v.Size != 9007199254740993 || *v.VirtualSize != 9223372036854775807 || *v.MinDisk != -1 || *v.MinRAM != 0 || !reflect.DeepEqual(v.Tags, []string{"", "repeat", "repeat", "line\n"}) || len(v.Locations) != 1 || *v.Locations[0].URL != "literal:%2F ?#" || string(v.Locations[0].Metadata["huge"]) != "1e1000" || v.Links != nil || v.StatusCode != 200 {
			t.Fatal(v)
		}
		for key, want := range map[string]string{"properties": "[1e1000]", "metadata": "17", "owner_id": "false", "ID": "42", "links": "17", "hw_vif_multiqueue_enabled": `"false"`, "x-number": "9007199254740995"} {
			if string(v.Properties[key]) != want || string(v.Body[key]) != want {
				t.Error(key, v.Properties[key], v.Body[key])
			}
		}
		for _, key := range []string{"id", "name", "tags", "locations", "os_hidden", "created_at", "size", "stores"} {
			if v.Properties[key] != nil {
				t.Error("canonical leaked into projection", key)
			}
		}
		if v.Body["openstack-image-store-ids"] != nil || v.Properties["openstack-image-store-ids"] != nil {
			t.Fatal("header injected into JSON")
		}
		raw[0] = '!'
		wire.Header.Set("X-Request-Id", "late")
		v.Properties["x-number"][0] = '1'
		v.Locations[0].Metadata["integer"][0] = '1'
		*v.ID = "typed"
		v.Tags[0] = "typed"
		*v.Locations[0].URL = "typed"
		if string(v.Body["x-number"]) != "9007199254740995" || !strings.Contains(string(v.Body["locations"]), "9007199254740995") || string(v.Body["id"]) != `"response-id"` || string(v.Body["tags"]) != `["","repeat","repeat","line\n"]` || v.Header.Get("X-Request-Id") != "actual-task" {
			t.Fatal("body aliases borrowed or typed data", v)
		}
		v.Body["metadata"][0] = '9'
		if string(v.Properties["metadata"]) != "17" {
			t.Fatal("projection aliases body")
		}
	})
	for _, raw := range []string{`{}`, `{"id":null,"name":null,"protected":null,"os_hidden":null,"size":null,"virtual_size":null,"min_disk":null,"min_ram":null,"tags":null,"locations":null,"created_at":null}`, `{"ID":17,"Name":false,"Protected":{},"Size":"17","Tags":{},"Locations":false}`, `{"id":"","protected":false,"os_hidden":false,"size":0,"tags":[],"locations":[]}`} {
		t.Run("nullable "+raw, func(t *testing.T) {
			c := taskContractsClient(func(*http.Request) (*http.Response, error) { return imageContractJSONResponse(raw), nil })
			v, e := image.New(c).GetImage(context.Background(), resource.ID("chosen"))
			if e != nil || v == nil || v.Properties == nil {
				t.Fatal(v, e)
			}
			if strings.Contains(raw, `"id":""`) {
				if v.ID == nil || *v.ID != "" || v.Protected == nil || *v.Protected || v.Hidden == nil || *v.Hidden || v.Size == nil || *v.Size != 0 || v.Tags == nil || v.Locations == nil {
					t.Fatal(v)
				}
			} else if v.ID != nil || v.Name != nil || v.Protected != nil || v.Hidden != nil || v.Size != nil || v.Tags != nil || v.Locations != nil || v.CreatedAt != nil {
				t.Fatal("null/decoy seeding", v)
			}
		})
	}
	badFields := []string{`"id":17`, `"name":true`, `"status":[]`, `"visibility":{}`, `"owner":false`, `"container_format":0`, `"disk_format":[]`, `"checksum":{}`, `"os_hash_algo":false`, `"os_hash_value":17`, `"self":[]`, `"file":0`, `"schema":false`, `"direct_url":{}`, `"stores":[]`, `"created_at":[]`, `"updated_at":false`, `"protected":"false"`, `"os_hidden":0`, `"size":"0"`, `"size":1.5`, `"size":1e3`, `"size":9223372036854775808`, `"virtual_size":false`, `"min_disk":[]`, `"min_ram":{}`, `"tags":{}`, `"tags":[null]`, `"tags":[17]`, `"locations":{}`, `"locations":[null]`, `"locations":[[]]`, `"locations":[{"url":17}]`, `"locations":[{"metadata":[]}]`}
	for _, field := range badFields {
		t.Run("wrong canonical "+field, func(t *testing.T) {
			raw := []byte("{" + field + "}")
			body := &taskContractsBody{Reader: bytes.NewReader(raw)}
			c := taskContractsClient(func(*http.Request) (*http.Response, error) { return taskContractsWire(200, body), nil })
			v, e := image.New(c).GetImage(context.Background(), resource.ID("chosen"))
			taskContractsProof(t, e, 200, raw)
			if v != nil || body.closes.Load() != 1 {
				t.Fatal(v, e, body.closes.Load())
			}
		})
	}
	t.Run("public decoder atomic and reset without alias", func(t *testing.T) {
		var v image.ImageInfo
		if e := json.Unmarshal([]byte(imageContractObject), &v); e != nil {
			t.Fatal(e)
		}
		before, _ := json.Marshal(v.Body)
		for _, bad := range []string{`{"id":"changed","size":1.5}`, `{"id":"changed","locations":[{"metadata":[]}]}`, `{"id":"changed","tags":[null]}`} {
			if e := json.Unmarshal([]byte(bad), &v); e == nil {
				t.Fatal(bad)
			}
			after, _ := json.Marshal(v.Body)
			if *v.ID != "response-id" || !bytes.Equal(before, after) {
				t.Fatal("partial decoder mutation", v)
			}
		}
		if e := json.Unmarshal([]byte(`{"name":"next"}`), &v); e != nil || v.ID != nil || v.Tags != nil || v.Locations != nil || len(v.Properties) != 0 || *v.Name != "next" {
			t.Fatal(v, e)
		}
	})
}

func TestImageContractsPagingAndConsumption(t *testing.T) {
	t.Run("short and empty advertised pages refresh headers and auth", func(t *testing.T) {
		c := taskContractsClient(nil)
		c.MoreHeaders = map[string]string{"X-Source": "first"}
		var calls atomic.Int32
		initial := url.Values{"name": {"line\n"}, "limit": {"4"}, "tag": {"first", "second", "first"}, "sort_key": {"name", "size"}, "sort_dir": {"asc", "desc"}, "x-property": {"one", "two"}}
		c.ProviderClient.HTTPClient.Transport = taskContractsTransport(func(r *http.Request) (*http.Response, error) {
			n := calls.Add(1)
			want := initial.Encode()
			wantHeader, wantToken := "first", "initial"
			if n > 1 {
				q := url.Values{}
				for k, v := range initial {
					q[k] = append([]string(nil), v...)
				}
				q.Set("marker", fmt.Sprintf("m%d", n-1))
				want = q.Encode()
				wantHeader, wantToken = fmt.Sprintf("source%d", n-1), fmt.Sprintf("token%d", n-1)
			}
			if r.Method != "GET" || r.Body != nil || r.URL.EscapedPath() != taskContractsPrefix+"images" || r.URL.RawQuery != want || r.Header.Get("X-Source") != wantHeader || r.Header.Get("X-Fixed") != "option" || r.Header.Get("X-Auth-Token") != wantToken {
				t.Error(n, r.URL, r.Header, want)
			}
			if n == 1 {
				return imageContractJSONResponse(`{"images":[{"id":"first"}],"next":"/v2/images?name=line%0A&limit=4&tag=second&sort_key=size&sort_dir=desc&x-property=two&marker=m1"}`), nil
			}
			if n == 2 {
				c.MoreHeaders["X-Source"] = "source2"
				c.ProviderClient.SetToken("token2")
				return imageContractJSONResponse(`{"images":[],"next":"?name=line%0A&limit=4&tag=first&sort_key=name&sort_dir=asc&x-property=one&marker=m2"}`), nil
			}
			if n == 3 {
				return imageContractJSONResponse(`{"images":[{"id":"last"}],"next":null,"first":"https://foreign.invalid/images"}`), nil
			}
			t.Error("replay", n)
			return nil, errors.New("extra page")
		})
		s := image.New(c)
		opts := []image.ListImagesOption{image.WithListImagesName("line\n"), image.WithListImagesLimit(4), image.WithListImagesTags("first", "second", "first"), image.WithListImagesSortKeys("name", "size"), image.WithListImagesSortDirs("asc", "desc"), image.WithListImagesFilter("x-property", "one", "two"), image.WithListImagesHeader("X-Fixed", "option")}
		got := []string{}
		for v, e := range s.ListImages(context.Background(), opts...) {
			if e != nil {
				t.Fatal(e)
			}
			got = append(got, *v.ID)
			if len(got) == 1 {
				c.MoreHeaders["X-Source"] = "source1"
				c.ProviderClient.SetToken("token1")
			}
		}
		// The empty second page has no row callback; advance its ordinary policy at transport return.
		if !reflect.DeepEqual(got, []string{"first", "last"}) || calls.Load() != 3 {
			t.Fatal(got, calls.Load())
		}
	})
	for _, mode := range []string{"cap", "break", "single", "unlimited"} {
		t.Run("consumed rows "+mode, func(t *testing.T) {
			raw := []byte(`{"images":[{"id":"first"},null],"next":17}`)
			var calls atomic.Int32
			body := &taskContractsBody{Reader: bytes.NewReader(raw)}
			c := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return taskContractsWire(200, body), nil })
			s := image.New(c)
			if mode == "break" {
				for v, e := range s.ListImages(context.Background()) {
					if e != nil || v == nil {
						t.Fatal(v, e)
					}
					break
				}
				if calls.Load() != 1 || body.closes.Load() != 1 {
					t.Fatal(calls.Load(), body.closes.Load())
				}
				return
			}
			opts := []image.ListImagesOption{}
			if mode == "cap" {
				opts = append(opts, image.WithListImagesMaxItems(1))
			}
			if mode == "single" {
				raw = []byte(`{"images":[{"id":"first"}],"next":17}`)
				body.Reader = bytes.NewReader(raw)
				opts = append(opts, image.WithListImagesSinglePage(true))
			}
			rows, e := s.AllImages(context.Background(), opts...)
			if mode == "unlimited" {
				taskContractsProof(t, e, 200, raw)
				if rows != nil {
					t.Fatal(rows)
				}
			} else if e != nil || len(rows) != 1 {
				t.Fatal(rows, e)
			}
			if calls.Load() != 1 || body.closes.Load() != 1 {
				t.Fatal(calls.Load(), body.closes.Load())
			}
		})
	}
	for _, raw := range []string{`{}`, `null`, `[]`, `{"images":null}`, `{"Images":[]}`, `{"images":{}}`, `{"images":[{}]} trailing`, "{\"images\":[{}],\"x\":\"" + string([]byte{0xff}) + "\"}"} {
		t.Run("envelope "+raw, func(t *testing.T) {
			c := taskContractsClient(func(*http.Request) (*http.Response, error) { return imageContractJSONResponse(raw), nil })
			rows, e := image.New(c).AllImages(context.Background(), image.WithListImagesMaxItems(1))
			taskContractsProof(t, e, 200, []byte(raw))
			if rows != nil {
				t.Fatal(rows, e)
			}
		})
	}
	for _, tail := range []string{"", `,"next":null`, `,"next":""`} {
		t.Run("only canonical next stops "+tail, func(t *testing.T) {
			var calls atomic.Int32
			c := taskContractsClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				r := imageContractJSONResponse(`{"images":[],"Next":"https://foreign.invalid","first":"https://foreign.invalid","links":[{"rel":"next","href":"https://foreign.invalid"}]` + tail + `}`)
				r.Header.Set("Link", `<https://foreign.invalid/images>; rel="next"`)
				return r, nil
			})
			rows, e := image.New(c).AllImages(context.Background())
			if rows == nil || len(rows) != 0 || e != nil || calls.Load() != 1 {
				t.Fatal(rows, e, calls.Load())
			}
		})
	}
	t.Run("full ordered query next and late HTTP no partial All", func(t *testing.T) {
		var calls atomic.Int32
		c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			n := calls.Add(1)
			if n == 1 {
				return imageContractJSONResponse(`{"images":[{"id":"first"}],"next":"/v2/images?tag=a&tag=b&marker=second"}`), nil
			}
			if !reflect.DeepEqual(r.URL.Query()["tag"], []string{"a", "b"}) {
				t.Error(r.URL)
			}
			return taskContractsWire(503, &taskContractsBody{Reader: strings.NewReader("late page")}), nil
		})
		rows, e := image.New(c).AllImages(context.Background(), image.WithListImagesTags("a", "b"))
		if rows != nil || !gophercloud.ResponseCodeIs(e, 503) || calls.Load() != 2 {
			t.Fatal(rows, e, calls.Load())
		}
	})
	t.Run("exact captured encoded reverse prefix accepted", func(t *testing.T) {
		base := "https://glance.invalid/reverse%20prefix/glance/v2/"
		var calls atomic.Int32
		c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			n := calls.Add(1)
			if r.URL.EscapedPath() != "/reverse%20prefix/glance/v2/images" {
				t.Error(r.URL)
			}
			if n == 1 {
				return imageContractJSONResponse(`{"images":[],"next":"` + base + `images?marker=next"}`), nil
			}
			return imageContractJSONResponse(`{"images":[]}`), nil
		})
		c.ResourceBase = base
		rows, e := image.New(c).AllImages(context.Background())
		if rows == nil || e != nil || calls.Load() != 2 {
			t.Fatal(rows, e, calls.Load())
		}
	})
	badNext := []string{`17`, `[]`, `"https://foreign.invalid/v2/images?tag=a&marker=n"`, `"//glance.invalid/v2/images?tag=a&marker=n"`, `"https://user@glance.invalid/v2/images?tag=a&marker=n"`, `"/v2/images/other?tag=a&marker=n"`, `"/v2/./images?tag=a&marker=n"`, `"/v2/%69mages?tag=a&marker=n"`, `"/v2/images?tag=a&marker=n#fragment"`, `"/v2/images?tag=a&marker=n%zz"`, `"/v2/images?marker=n"`, `"/v2/images?tag=changed&marker=n"`, `"/v2/images?tag=a&tag=b&tag=a&marker=n"`, `"/v2/images?tag=b&tag=a&marker=n"`, `"/v2/images?tag=a&other=1&marker=n"`, `"/v2/images?tag=a"`, `"/v2/images?tag=a&marker="`, `"/v2/images?tag=a&marker=n&marker=x"`, `"/v2/images?tag=a&marker=initial"`, `"/v2/images?tag=a&marker=n\n"`}
	for _, link := range badNext {
		t.Run("next guard "+link, func(t *testing.T) {
			var calls atomic.Int32
			raw := []byte(`{"images":[{}],"next":` + link + `}`)
			c := taskContractsClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return taskContractsWire(200, &taskContractsBody{Reader: bytes.NewReader(raw)}), nil
			})
			rows, e := image.New(c).AllImages(context.Background(), image.WithListImagesTags("a", "b"), image.WithListImagesMarker("initial"))
			taskContractsProof(t, e, 200, raw)
			if rows != nil || calls.Load() != 1 {
				t.Fatal(rows, e, calls.Load())
			}
		})
	}
	t.Run("repeated marker and restored URL cycle", func(t *testing.T) {
		var calls atomic.Int32
		raw := `{"images":[],"next":"/v2/images?marker=again"}`
		c := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return imageContractJSONResponse(raw), nil })
		rows, e := image.New(c).AllImages(context.Background())
		taskContractsProof(t, e, 200, []byte(raw))
		if rows != nil || !errors.Is(e, resource.ErrPaginationCycle) || calls.Load() != 2 {
			t.Fatal(rows, e, calls.Load())
		}
	})
}

func TestImageContractsConcreteQueriesAndSource(t *testing.T) {
	t.Run("all literal query helpers preserve presence and repeated data", func(t *testing.T) {
		want := url.Values{"id": {""}, "name": {"line\nname"}, "visibility": {"future"}, "member_status": {""}, "owner": {"owner"}, "status": {"in:future,queued"}, "created_at": {"not-ISO\r\n"}, "updated_at": {""}, "container_format": {""}, "disk_format": {"in:qcow2,iso"}, "limit": {"0"}, "size_min": {"0"}, "size_max": {"0"}, "protected": {"false"}, "os_hidden": {"false"}, "sort_key": {"", "size", "size"}, "sort_dir": {"future", "", "future"}, "tag": {"", "line\n", "repeat", "repeat"}, "metadata": {"literal\x00data", ""}}
		var calls atomic.Int32
		c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if !reflect.DeepEqual(r.URL.Query(), want) || r.URL.Path != taskContractsPrefix+"images" || r.Body != nil || r.Header.Get("X-Option") != "owned" || r.Header.Get("X-Last") != "last" {
				t.Error(r.URL.Query(), want, r.Header, r.Body)
			}
			return imageContractJSONResponse(`{"images":[]}`), nil
		})
		headers := map[string]string{"X-Option": "owned"}
		h := image.WithListImagesHeaders(headers)
		headers["X-Option"] = "late"
		rows, e := image.New(c).AllImages(context.Background(), h, image.WithListImagesHeader("X-Last", "last"), image.WithListImagesID(""), image.WithListImagesName("line\nname"), image.WithListImagesVisibility("future"), image.WithListImagesMemberStatus(""), image.WithListImagesOwner("owner"), image.WithListImagesStatus("in:future,queued"), image.WithListImagesCreatedAt("not-ISO\r\n"), image.WithListImagesUpdatedAt(""), image.WithListImagesContainerFormat(""), image.WithListImagesDiskFormat("in:qcow2,iso"), image.WithListImagesLimit(0), image.WithListImagesSizeMin(0), image.WithListImagesSizeMax(0), image.WithListImagesProtected(false), image.WithListImagesHidden(false), image.WithListImagesMarker(""), image.WithListImagesSortKeys("", "size", "size"), image.WithListImagesSortDirs("future", "", "future"), image.WithListImagesTags("", "line\n", "repeat", "repeat"), image.WithListImagesFilter("metadata", "literal\x00data", ""), image.WithListImagesMaxItems(0), image.WithListImagesSinglePage(false))
		if rows == nil || e != nil || calls.Load() != 1 {
			t.Fatal(rows, e, calls.Load())
		}
	})
	t.Run("pointer nil absent explicit empty Sort and no invented defaults", func(t *testing.T) {
		var calls atomic.Int32
		c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			n := calls.Add(1)
			want := ""
			if n == 2 {
				want = "sort="
			}
			if r.URL.RawQuery != want {
				t.Error(n, r.URL)
			}
			return imageContractJSONResponse(`{"images":[]}`), nil
		})
		s := image.New(c)
		for _, opts := range [][]image.ListImagesOption{nil, {image.WithListImagesSort("")}} {
			rows, e := s.AllImages(context.Background(), opts...)
			if rows == nil || e != nil {
				t.Fatal(rows, e)
			}
		}
		if calls.Load() != 2 {
			t.Fatal(calls.Load())
		}
	})
	t.Run("full replacement and factory maps slices pointers owned", func(t *testing.T) {
		value := image.ListImagesOpts{Headers: map[string]string{"X-Owned": "original"}, Name: imageContractPtr("owned"), Limit: imageContractPtr(2), SizeMin: imageContractPtr(int64(9)), SizeMax: imageContractPtr(int64(1)), Protected: imageContractPtr(false), Hidden: imageContractPtr(true), Tags: []string{"a", "b"}, Filters: url.Values{"x": {"one", "two"}}}
		full := image.WithListImagesOpts(value)
		*value.Name = "changed"
		*value.Limit = 99
		*value.SizeMin = 99
		*value.Protected = true
		value.Tags[0] = "changed"
		value.Headers["X-Owned"] = "changed"
		value.Filters["x"][0] = "changed"
		filters := url.Values{"replaced": {"one", "two"}, "deleted": {"old"}}
		f := image.WithListImagesFilters(filters)
		filters["replaced"][0] = "late"
		var calls atomic.Int32
		c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			want := url.Values{"name": {"owned"}, "limit": {"2"}, "size_min": {"9"}, "size_max": {"1"}, "protected": {"false"}, "os_hidden": {"true"}, "tag": {"a", "b"}, "replaced": {"one", "two"}, "single": {""}}
			if !reflect.DeepEqual(r.URL.Query(), want) || r.Header.Get("X-Owned") != "original" || r.Header.Get("X-Discard") != "" {
				t.Error(r.URL.Query(), want, r.Header)
			}
			return imageContractJSONResponse(`{"images":[]}`), nil
		})
		rows, e := image.New(c).AllImages(context.Background(), image.WithListImagesHeader("X-Discard", "before replacement"), image.WithListImagesStatus("before"), full, f, image.WithListImagesFilter("deleted"), image.WithListImagesFilter("single", ""))
		if rows == nil || e != nil || calls.Load() != 1 {
			t.Fatal(rows, e, calls.Load())
		}
		getValue := image.GetImageOpts{Headers: map[string]string{"X-Get": "owned"}}
		getFull := image.WithGetImageOpts(getValue)
		getValue.Headers["X-Get"] = "changed"
		c.HTTPClient.Transport = taskContractsTransport(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Get") != "owned" || r.Header.Get("X-Discard") != "" {
				t.Error(r.Header)
			}
			return imageContractJSONResponse(`{}`), nil
		})
		v, e := image.New(c).GetImage(context.Background(), resource.ID("fixed"), image.WithGetImageHeader("X-Discard", "discard"), getFull)
		if v == nil || e != nil {
			t.Fatal(v, e)
		}
	})
	t.Run("callbacks copied individually and reusable parallel lazy options", func(t *testing.T) {
		var calls, callbacks atomic.Int32
		var mu sync.Mutex
		retained := make([]*image.ListImagesOpts, 0)
		c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Owned") != "owned" || r.URL.Query().Get("name") != "owned" || r.URL.Query().Get("x") != "owned" {
				t.Error(r.URL, r.Header)
			}
			return imageContractJSONResponse(`{"images":[{}]}`), nil
		})
		first := func(o *image.ListImagesOpts) error {
			callbacks.Add(1)
			if o.Headers == nil || o.Filters == nil {
				t.Error("callback maps not initialized")
			}
			o.Headers["X-Owned"] = "owned"
			o.Name = imageContractPtr("owned")
			o.Filters["x"] = []string{"owned"}
			mu.Lock()
			retained = append(retained, o)
			mu.Unlock()
			return nil
		}
		second := func(o *image.ListImagesOpts) error {
			mu.Lock()
			prior := retained[len(retained)-1]
			prior.Headers["X-Owned"] = "late"
			*prior.Name = "late"
			prior.Filters["x"][0] = "late"
			mu.Unlock()
			if o.Headers["X-Owned"] != "owned" || *o.Name != "owned" || o.Filters.Get("x") != "owned" {
				t.Error("callback data aliases prior", o)
			}
			return nil
		}
		// Each sequence copies its function slice; callback storage is separate per iteration.
		opts := []image.ListImagesOption{first, second}
		seq := image.New(c).ListImages(context.Background(), opts...)
		opts[0] = func(*image.ListImagesOpts) error { return errors.New("changed slice") }
		if calls.Load() != 0 || callbacks.Load() != 0 {
			t.Fatal("iterator eager")
		}
		for n := 0; n < 2; n++ {
			for v, e := range seq {
				if v == nil || e != nil {
					t.Fatal(v, e)
				}
			}
		}
		// The independent iterator is safe to invoke in parallel when no caller-owned options are mutated.
		parallel := image.New(c).ListImages(context.Background(), image.WithListImagesHeader("X-Owned", "owned"), image.WithListImagesName("owned"), image.WithListImagesFilter("x", "owned"))
		var wg sync.WaitGroup
		for n := 0; n < 4; n++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for v, e := range parallel {
					if v == nil || e != nil {
						t.Error(v, e)
					}
				}
			}()
		}
		wg.Wait()
		if calls.Load() != 6 || callbacks.Load() != 2 {
			t.Fatal(calls.Load(), callbacks.Load())
		}
	})
	bad := []struct {
		name string
		opts []image.ListImagesOption
	}{
		{"negative limit", []image.ListImagesOption{image.WithListImagesLimit(-1)}}, {"negative min", []image.ListImagesOption{image.WithListImagesSizeMin(-1)}}, {"negative max", []image.ListImagesOption{image.WithListImagesSizeMax(-1)}}, {"negative cap", []image.ListImagesOption{image.WithListImagesMaxItems(-1)}}, {"Sort presence classic conflict", []image.ListImagesOption{image.WithListImagesSort(""), image.WithListImagesSortKeys("name")}}, {"classic directions count", []image.ListImagesOption{image.WithListImagesSortKeys("name"), image.WithListImagesSortDirs("asc", "desc")}}, {"empty filter key", []image.ListImagesOption{image.WithListImagesFilter("", "x")}}, {"control filter key", []image.ListImagesOption{image.WithListImagesFilter("line\n", "x")}}, {"empty filter values", []image.ListImagesOption{image.WithListImagesFilters(url.Values{"x": nil})}}, {"invalid UTF8 query", []image.ListImagesOption{image.WithListImagesName(string([]byte{0xff}))}}, {"nil callback", []image.ListImagesOption{nil}}, {"protected header", []image.ListImagesOption{image.WithListImagesHeader("X-Auth-Token", "override")}}, {"invalid header", []image.ListImagesOption{image.WithListImagesHeader("X-Bad", "line\n")}},
	}
	for _, key := range []string{"limit", "marker", "id", "name", "visibility", "member_status", "owner", "status", "size_min", "size_max", "protected", "os_hidden", "sort", "sort_key", "sort_dir", "tag", "created_at", "updated_at", "container_format", "disk_format", "deleted", "max_items", "single_page"} {
		bad = append(bad, struct {
			name string
			opts []image.ListImagesOption
		}{"reserved " + key, []image.ListImagesOption{image.WithListImagesFilter(key, "value")}})
	}
	for _, tt := range bad {
		t.Run("preflight "+tt.name, func(t *testing.T) {
			var calls atomic.Int32
			c := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected") })
			rows, e := image.New(c).AllImages(context.Background(), tt.opts...)
			if rows != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(rows, e, calls.Load())
			}
		})
	}
	for _, mode := range []string{"nil context", "canceled context", "wrong type", "nil provider", "bad target", "bad microversion", "source protected header", "callback cause", "get bad header"} {
		t.Run("source preflight "+mode, func(t *testing.T) {
			var calls, callbacks atomic.Int32
			cause := errors.New("preflight cause")
			c := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected") })
			ctx := context.Background()
			want := resource.ErrInvalidOption
			getOpts := []image.GetImageOption{func(*image.GetImageOpts) error {
				callbacks.Add(1)
				if mode == "callback cause" {
					return cause
				}
				return nil
			}}
			switch mode {
			case "nil context":
				ctx = nil
			case "canceled context":
				var cancel context.CancelCauseFunc
				ctx, cancel = context.WithCancelCause(ctx)
				cancel(cause)
				want = context.Canceled
			case "wrong type":
				c.Type = "compute"
				want = resource.ErrUnsupported
			case "nil provider":
				c.ProviderClient = nil
			case "bad target":
				c.ResourceBase = "https://glance.invalid/v2/?route=bad"
			case "bad microversion":
				c.Microversion = "bad\n"
			case "source protected header":
				c.MoreHeaders = map[string]string{"X-Auth-Token": "override"}
			case "callback cause":
				want = cause
			case "get bad header":
				getOpts = []image.GetImageOption{image.WithGetImageHeader("X-Auth-Token", "override")}
			}
			v, e := image.New(c).GetImage(ctx, resource.Name("exact"), getOpts...)
			if v != nil || !errors.Is(e, want) || calls.Load() != 0 || mode != "callback cause" && callbacks.Load() != 0 {
				t.Fatal(v, e, calls.Load(), callbacks.Load())
			}
			if mode == "canceled context" && !errors.Is(e, cause) {
				t.Fatal(e)
			}
		})
	}
	for _, field := range []string{"provider", "endpoint", "base", "type", "microversion"} {
		t.Run("callback source drift "+field, func(t *testing.T) {
			var calls atomic.Int32
			c := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected") })
			v, e := image.New(c).GetImage(context.Background(), resource.Name("exact"), func(*image.GetImageOpts) error {
				switch field {
				case "provider":
					c.ProviderClient = &gophercloud.ProviderClient{}
				case "endpoint":
					c.Endpoint = "https://other.invalid/v2/"
				case "base":
					c.ResourceBase = "https://other.invalid/v2/"
				case "type":
					c.Type = "compute"
				case "microversion":
					c.Microversion = "2.99"
				}
				return nil
			})
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(v, e, calls.Load())
			}
		})
	}
	t.Run("source drift after native Name stops fresh GET", func(t *testing.T) {
		var calls atomic.Int32
		var c *gophercloud.ServiceClient
		c = taskContractsClient(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			b := &taskContractsBody{Reader: strings.NewReader(`{"images":[{"id":"chosen","name":"exact"}]}`), onClose: func() { c.Microversion = "2.99" }}
			return taskContractsWire(200, b), nil
		})
		v, e := image.New(c).GetImage(context.Background(), resource.Name("exact"))
		if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 1 {
			t.Fatal(v, e, calls.Load())
		}
	})
	for _, mode := range []string{"change", "break"} {
		t.Run("source after yielded row "+mode, func(t *testing.T) {
			raw := []byte(`{"images":[{},{}],"next":"/v2/images?marker=next"}`)
			var calls atomic.Int32
			c := taskContractsClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return taskContractsWire(200, &taskContractsBody{Reader: bytes.NewReader(raw)}), nil
			})
			var rows int
			var final error
			for v, e := range image.New(c).ListImages(context.Background()) {
				if e != nil {
					final = e
					break
				}
				if v == nil {
					t.Fatal("nil row")
				}
				rows++
				c.Microversion = "2.99"
				if mode == "break" {
					break
				}
			}
			if rows != 1 || calls.Load() != 1 {
				t.Fatal(rows, final, calls.Load())
			}
			if mode == "break" {
				if final != nil {
					t.Fatal(final)
				}
			} else {
				taskContractsProof(t, final, 200, raw)
				if !errors.Is(final, resource.ErrInvalidOption) {
					t.Fatal(final)
				}
			}
		})
	}
}

func TestImageContractsResponseOwnershipAndNativePolicy(t *testing.T) {
	for _, op := range []string{"Get", "List", "All"} {
		for _, failure := range []string{"read", "close", "context", "combined opaque", "source"} {
			t.Run(fmt.Sprintf("accepted %s %s", op, failure), func(t *testing.T) {
				readCause, closeCause, cancelCause := errors.New("read cause"), errors.New("Close cause"), errors.New("context custom cause")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				raw := []byte(imageContractList)
				if op == "Get" {
					raw = []byte(imageContractObject)
				}
				if failure == "read" {
					raw = []byte("partial-before-read-error")
				}
				if failure == "combined opaque" {
					raw = []byte{'o', 'p', 'a', 'q', 'u', 'e', 0xff}
				}
				body := &taskContractsBody{Reader: bytes.NewReader(raw)}
				if failure == "read" || failure == "combined opaque" {
					body.Reader = taskContractsReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
				}
				if failure == "close" || failure == "combined opaque" {
					body.closeErr = closeCause
				}
				var calls, hooks atomic.Int32
				var client *gophercloud.ServiceClient
				body.onClose = func() {
					if failure == "context" || failure == "combined opaque" {
						cancel(cancelCause)
					}
					if failure == "source" {
						client.Microversion = "2.19"
					}
				}
				client = taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return taskContractsWire(200, body), nil })
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks.Add(1)
					return nil
				}
				rows, err := imageContractCall(image.New(client), ctx, op)
				taskContractsProof(t, err, 200, raw)
				if rows != nil || calls.Load() != 1 || hooks.Load() != 0 || body.closes.Load() != 1 {
					t.Fatal(rows, err, calls.Load(), hooks.Load(), body.closes.Load())
				}
				if (failure == "read" || failure == "combined opaque") && !errors.Is(err, readCause) {
					t.Fatal(err)
				}
				if (failure == "close" || failure == "combined opaque") && !errors.Is(err, closeCause) {
					t.Fatal(err)
				}
				if (failure == "context" || failure == "combined opaque") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal(err)
				}
				if failure == "source" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			})
		}
	}
	for _, code := range []int{201, 202, 204, 300, 403, 404} {
		t.Run(fmt.Sprintf("strict actual %d", code), func(t *testing.T) {
			var calls atomic.Int32
			body := &taskContractsBody{Reader: strings.NewReader("actual failure")}
			client := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return taskContractsWire(code, body), nil })
			rows, err := image.New(client).AllImages(context.Background())
			var native gophercloud.ErrUnexpectedResponseCode
			var accepted *resource.ResponseError
			if rows != nil || !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || native.Method != "GET" || native.URL != taskContractsBase+"images" || string(native.Body) != "actual failure" || native.ResponseHeader.Get("X-Request-Id") != "actual-task" || errors.As(err, &accepted) || calls.Load() != 1 || body.closes.Load() != 1 {
				t.Fatal(rows, err, native, calls.Load(), body.closes.Load())
			}
		})
	}
	t.Run("prebody retry preserves original provider and bodyless route", func(t *testing.T) {
		for _, mode := range []string{"503", "transport", "reauth"} {
			t.Run(mode, func(t *testing.T) {
				cause := errors.New("transport cause")
				var calls, hooks atomic.Int32
				var bodies []*taskContractsBody
				var client *gophercloud.ServiceClient
				client = taskContractsClient(func(r *http.Request) (*http.Response, error) {
					n := calls.Add(1)
					if r.Method != "GET" || r.URL.String() != taskContractsBase+"images" || r.Body != nil {
						t.Error(r.Method, r.URL, r.Body)
					}
					if n == 1 && mode == "transport" {
						return nil, cause
					}
					code, raw := 200, imageContractList
					if n == 1 {
						code, raw = 503, "original503"
						if mode == "reauth" {
							code = 401
						}
					}
					if n == 2 && (r.Header.Get("X-Auth-Token") != "refreshed" || mode != "reauth" && r.Header.Get("X-Hook") != "advanced") {
						t.Error(r.Header)
					}
					b := &taskContractsBody{Reader: strings.NewReader(raw)}
					bodies = append(bodies, b)
					return taskContractsWire(code, b), nil
				})
				if mode == "reauth" {
					client.ReauthFunc = func(context.Context) error { hooks.Add(1); client.SetToken("refreshed"); return nil }
				} else {
					client.RetryFunc = func(_ context.Context, method, target string, o *gophercloud.RequestOpts, original error, _ uint) error {
						hooks.Add(1)
						if method != "GET" || target != taskContractsBase+"images" || o.JSONBody != nil || o.RawBody != nil || o.JSONResponse != nil || !o.KeepResponseBody || mode == "503" && !gophercloud.ResponseCodeIs(original, 503) || mode == "transport" && !errors.Is(original, cause) {
							t.Error(original, o)
						}
						o.MoreHeaders = map[string]string{"X-Hook": "advanced"}
						client.SetToken("refreshed")
						return nil
					}
				}
				provider := client.ProviderClient
				rows, err := image.New(client).AllImages(context.Background())
				if err != nil || len(rows) != 1 || calls.Load() != 2 || hooks.Load() != 1 || client.ProviderClient != provider {
					t.Fatal(rows, err, calls.Load(), hooks.Load())
				}
				for _, b := range bodies {
					if b.closes.Load() != 1 {
						t.Fatal(b.closes.Load())
					}
				}
			})
		}
	})
	for _, change := range []string{"JSON null", "KeepResponseBody", "JSONResponse", "RawBody", "unsupported JSON", "expanded OkCodes"} {
		t.Run("shared request guard "+change, func(t *testing.T) {
			var calls, hooks, borrowedReads atomic.Int32
			callbackCause, readCause, closeCause, cancelCause := errors.New("hook cause"), errors.New("expanded Read cause"), errors.New("expanded Close cause"), errors.New("expanded context cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			borrowed := &taskContractsBody{Reader: taskContractsReader(func([]byte) (int, error) { borrowedReads.Add(1); return 0, io.EOF })}
			var bodies []*taskContractsBody
			client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if r.Body != nil {
					t.Error("request acquired body", r.Body)
				}
				code, raw := 503, "original503"
				if n == 2 {
					code, raw = 204, "private204"
				}
				b := &taskContractsBody{Reader: strings.NewReader(raw)}
				if n == 2 {
					b.Reader = taskContractsReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
					b.closeErr = closeCause
					b.onClose = func() { cancel(cancelCause) }
				}
				bodies = append(bodies, b)
				return taskContractsWire(code, b), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, original error, _ uint) error {
				hooks.Add(1)
				if !gophercloud.ResponseCodeIs(original, 503) {
					t.Error(original)
				}
				switch change {
				case "JSON null":
					o.JSONBody = json.RawMessage(`null`)
				case "KeepResponseBody":
					o.KeepResponseBody = false
				case "JSONResponse":
					o.JSONResponse = new(any)
				case "RawBody":
					o.RawBody = borrowed
				case "unsupported JSON":
					o.JSONBody = make(chan int)
				case "expanded OkCodes":
					o.OkCodes = []int{200, 204}
					return nil
				}
				return callbackCause
			}
			originalHook := reflect.ValueOf(client.RetryFunc).Pointer()
			rows, err := image.New(client).AllImages(ctx)
			if change == "expanded OkCodes" {
				var native gophercloud.ErrUnexpectedResponseCode
				var accepted *resource.ResponseError
				if rows != nil || !errors.As(err, &native) || native.Actual != 204 || !reflect.DeepEqual(native.Expected, []int{200}) || native.Method != "GET" || native.URL != taskContractsBase+"images" || string(native.Body) != "private204" || native.ResponseHeader.Get("X-Request-Id") != "actual-task" || errors.As(err, &accepted) || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) || calls.Load() != 2 {
					t.Fatal(rows, err, native, calls.Load())
				}
			} else if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, callbackCause) || !gophercloud.ResponseCodeIs(err, 503) || calls.Load() != 1 {
				t.Fatal(rows, err, calls.Load())
			}
			if change == "unsupported JSON" {
				var typed *json.UnsupportedTypeError
				if !errors.As(err, &typed) {
					t.Fatal(err)
				}
			}
			if hooks.Load() != 1 || borrowedReads.Load() != 0 || borrowed.closes.Load() != 0 || reflect.ValueOf(client.RetryFunc).Pointer() != originalHook {
				t.Fatal(hooks.Load(), borrowedReads.Load(), borrowed.closes.Load())
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
	for _, mode := range []string{"transport", "callback", "reauth"} {
		t.Run("no nested missing suppression "+mode, func(t *testing.T) {
			cause := errors.New("nested cause")
			nested := &gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{200}}
			var calls atomic.Int32
			client := taskContractsClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				if mode == "transport" {
					return nil, errors.Join(cause, nested)
				}
				code := 503
				if mode == "reauth" {
					code = 401
				}
				return taskContractsWire(code, &taskContractsBody{Reader: strings.NewReader("native failure")}), nil
			})
			if mode == "callback" {
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					return errors.Join(cause, nested)
				}
			}
			if mode == "reauth" {
				client.ReauthFunc = func(context.Context) error { return errors.Join(cause, nested) }
			}
			rows, err := image.New(client).AllImages(context.Background())
			if rows != nil || err == nil || calls.Load() != 1 {
				t.Fatal(rows, err, calls.Load())
			}
			if mode == "reauth" {
				var native *gophercloud.ErrUnableToReauthenticate
				if !errors.As(err, &native) || !errors.Is(native.ErrReauth, cause) || !errors.Is(native.ErrReauth, nested) || !gophercloud.ResponseCodeIs(native.ErrOriginal, 401) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, cause) || !errors.Is(err, nested) || mode == "callback" && !gophercloud.ResponseCodeIs(err, 503) {
				t.Fatal(err)
			}
		})
	}
	for _, mode := range []string{"same target", "foreign target", "other query", "changed method"} {
		t.Run("native redirect policy "+mode, func(t *testing.T) {
			var calls, redirects atomic.Int32
			var bodies []*taskContractsBody
			client := taskContractsClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if r.Method != "GET" || r.URL.String() != taskContractsBase+"images" || r.Body != nil {
					t.Error(r.Method, r.URL, r.Body)
				}
				b := &taskContractsBody{Reader: strings.NewReader(imageContractList)}
				bodies = append(bodies, b)
				if n == 2 {
					return taskContractsWire(200, b), nil
				}
				wire := taskContractsWire(307, b)
				target := taskContractsBase + "images"
				if mode == "foreign target" {
					target = "https://foreign.invalid/images"
				}
				if mode == "other query" {
					target += "?limit=1"
				}
				wire.Header.Set("Location", target)
				return wire, nil
			})
			client.HTTPClient.CheckRedirect = func(r *http.Request, _ []*http.Request) error {
				redirects.Add(1)
				if mode == "changed method" {
					r.Method = "POST"
				}
				return nil
			}
			rows, err := image.New(client).AllImages(context.Background())
			if mode == "same target" {
				if err != nil || len(rows) != 1 || calls.Load() != 2 {
					t.Fatal(rows, err, calls.Load())
				}
			} else if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
				t.Fatal(rows, err, calls.Load())
			}
			if redirects.Load() != 1 {
				t.Fatal(redirects.Load())
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
}
