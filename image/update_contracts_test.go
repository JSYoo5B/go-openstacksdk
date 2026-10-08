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
	nativeImages "github.com/JSYoo5B/go-openstacksdk/image/v2/images"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const updateContractMedia = "application/openstack-images-v2.1-json-patch"

type updateContractOptions struct {
	update []image.UpdateImageOption
	set    []image.SetImagePropertiesOption
}

func updateContractCall(s *image.Service, ctx context.Context, ref resource.Ref, set bool, o updateContractOptions) (*image.ImageInfo, error) {
	if set {
		return s.SetImageProperties(ctx, ref, o.set...)
	}
	return s.UpdateImage(ctx, ref, o.update...)
}
func updateContractRead(t *testing.T, r *http.Request) ([]byte, []image.ImagePatch) {
	t.Helper()
	raw, e := io.ReadAll(r.Body)
	if e != nil {
		t.Fatal(e)
	}
	var changes []image.ImagePatch
	if e = json.Unmarshal(raw, &changes); e != nil || changes == nil {
		t.Fatalf("patch array %q: %v", raw, e)
	}
	return raw, changes
}
func updateContractWantPatch(t *testing.T, raw []byte, want string) {
	t.Helper()
	var got, expected []image.ImagePatch
	if e := json.Unmarshal(raw, &got); e != nil {
		t.Fatal(e)
	}
	if e := json.Unmarshal([]byte(want), &expected); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("patch %s, want %s", raw, want)
	}
}
func updateContractHeaders() updateContractOptions {
	return updateContractOptions{
		update: []image.UpdateImageOption{image.WithUpdateImageHeader("X-Option", "owned")},
		set:    []image.SetImagePropertiesOption{image.WithSetImagePropertiesHeader("X-Option", "owned")},
	}
}

func TestImageUpdateContractsWireAndReference(t *testing.T) {
	for _, set := range []bool{false, true} {
		t.Run(fmt.Sprintf("direct literal and empty patch set=%v", set), func(t *testing.T) {
			for _, version := range []string{"", "2.0", "2.18"} {
				t.Run("version "+version, func(t *testing.T) {
					cloud := testcloud.New(t)
					c := cloud.Client("image", "/unused/catalog/")
					c.ResourceBase = cloud.Server.URL + taskContractsPrefix
					c.Microversion = version
					c.MoreHeaders = map[string]string{"Content-Type": "source/type", "Accept": "source/accept", "X-Source": "captured"}
					id := "literal:%2F ?#한글"
					var calls atomic.Int32
					cloud.Mux.HandleFunc(taskContractsPrefix, func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						raw, _ := updateContractRead(t, r)
						wantVersion := ""
						if version != "" {
							wantVersion = "image " + version
						}
						if r.Method != "PATCH" || r.RequestURI != taskContractsPrefix+"images/"+url.PathEscape(id) || r.URL.RawQuery != "" || string(raw) != "[]" || r.Header.Get("Content-Type") != updateContractMedia || r.Header.Get("Accept") != "application/json" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Option") != "owned" || r.Header.Get("OpenStack-API-Version") != wantVersion {
							t.Error(r.Method, r.RequestURI, r.Header, string(raw))
						}
						w.Header().Set("X-Request-Id", "actual-task")
						w.Header().Set("Location", "https://passive.invalid/retarget")
						_, _ = io.WriteString(w, imageContractObject)
					})
					value, e := updateContractCall(image.New(c), context.Background(), resource.ID(id), set, updateContractHeaders())
					if e != nil || value == nil || *value.ID != "response-id" || value.StatusCode != 200 || value.Header.Get("Location") != "https://passive.invalid/retarget" || calls.Load() != 1 || c.MoreHeaders["Content-Type"] != "source/type" || c.MoreHeaders["Accept"] != "source/accept" {
						t.Fatal(value, e, c.MoreHeaders, calls.Load())
					}
				})
			}
		})
		t.Run(fmt.Sprintf("exact Name keeps lookup media and then owns patch set=%v", set), func(t *testing.T) {
			cloud := testcloud.New(t)
			c := cloud.Client("image", taskContractsPrefix)
			c.MoreHeaders = map[string]string{"Content-Type": "source/type", "Accept": "source/accept", "X-Source": "captured"}
			var calls, callbacks atomic.Int32
			opts := updateContractHeaders()
			if set {
				opts.set = append(opts.set, func(o *image.SetImagePropertiesOpts) error {
					callbacks.Add(1)
					c.MoreHeaders["X-Source"] = "after callback"
					return nil
				})
			} else {
				opts.update = append(opts.update, func(o *image.UpdateImageOpts) error {
					callbacks.Add(1)
					c.MoreHeaders["X-Source"] = "after callback"
					return nil
				})
			}
			cloud.Mux.HandleFunc(taskContractsPrefix, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Request-Id", "actual-task")
				n := calls.Add(1)
				if r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Option") != "owned" {
					t.Error(r.Header)
				}
				if n < 3 {
					if r.Method != "GET" || r.Header.Get("Content-Type") != "source/type" || r.Header.Get("Accept") != "source/accept" {
						t.Error(r.Method, r.Header)
					}
				}
				switch n {
				case 1:
					if r.URL.Path != taskContractsPrefix+"images" || r.URL.Query().Get("name") != "exact" {
						t.Error(r.URL)
					}
					cloud.Provider.SetToken("fresh")
					_, _ = io.WriteString(w, `{"images":[{"id":"chosen","name":"exact"},{"id":"near","name":"exact suffix"}],"next":"/v2/images?name=exact&marker=second"}`)
				case 2:
					if r.URL.Query().Get("marker") != "second" || r.Header.Get("X-Auth-Token") != "fresh" {
						t.Error(r.URL, r.Header)
					}
					_, _ = io.WriteString(w, `{"images":[{"id":"other","name":"other"}]}`)
				case 3:
					raw, _ := updateContractRead(t, r)
					if r.Method != "PATCH" || r.URL.Path != taskContractsPrefix+"images/chosen" || r.URL.RawQuery != "" || string(raw) != "[]" || r.Header.Get("Content-Type") != updateContractMedia || r.Header.Get("Accept") != "application/json" || r.Header.Get("X-Auth-Token") != "fresh" {
						t.Error(r.Method, r.URL, r.Header, string(raw))
					}
					_, _ = io.WriteString(w, imageContractObject)
				default:
					t.Error("unexpected lookup or mutation replay", n)
				}
			})
			value, e := updateContractCall(image.New(c), context.Background(), resource.Name("exact"), set, opts)
			if e != nil || value == nil || *value.ID != "response-id" || *value.Name != "literal\nname" || calls.Load() != 3 || callbacks.Load() != 1 || c.MoreHeaders["Content-Type"] != "source/type" || c.MoreHeaders["Accept"] != "source/accept" {
				t.Fatal(value, e, calls.Load(), callbacks.Load(), c.MoreHeaders)
			}
		})
		for _, mode := range []string{"missing", "ambiguous", "late HTTP", "unsafe resolved ID", "source drift"} {
			t.Run(fmt.Sprintf("Name failure %s set=%v", mode, set), func(t *testing.T) {
				var calls atomic.Int32
				var c *gophercloud.ServiceClient
				c = taskContractsClient(func(r *http.Request) (*http.Response, error) {
					n := calls.Add(1)
					if r.Method != "GET" {
						t.Error("mutation after failed lookup", r.Method)
					}
					raw := `{"images":[]}`
					code := 200
					switch mode {
					case "ambiguous":
						raw = `{"images":[{"id":"one","name":"exact"},{"id":"two","name":"exact"}]}`
					case "unsafe resolved ID":
						raw = `{"images":[{"id":"../escape","name":"exact"}]}`
					case "source drift":
						raw = `{"images":[{"id":"chosen","name":"exact"}]}`
						c.Microversion = "2.99"
					case "late HTTP":
						if n == 1 {
							raw = `{"images":[{"id":"chosen","name":"exact"}],"next":"/v2/images?name=exact&marker=second"}`
						} else {
							code = 503
							raw = "late lookup failure"
						}
					}
					return taskContractsWire(code, &taskContractsBody{Reader: strings.NewReader(raw)}), nil
				})
				value, e := updateContractCall(image.New(c), context.Background(), resource.Name("exact"), set, updateContractOptions{})
				wantCalls := int32(1)
				if mode == "late HTTP" {
					wantCalls = 2
				}
				if value != nil || e == nil || calls.Load() != wantCalls {
					t.Fatal(value, e, calls.Load())
				}
				switch mode {
				case "missing":
					if !errors.Is(e, resource.ErrNotFound) {
						t.Fatal(e)
					}
				case "ambiguous":
					if !errors.Is(e, resource.ErrAmbiguous) {
						t.Fatal(e)
					}
				case "unsafe resolved ID", "source drift":
					if !errors.Is(e, resource.ErrInvalidOption) {
						t.Fatal(e)
					}
				case "late HTTP":
					if !gophercloud.ResponseCodeIs(e, 503) {
						t.Fatal(e)
					}
				}
			})
		}
	}
	t.Run("native generated Update ABI remains available", func(t *testing.T) {
		var calls atomic.Int32
		c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			raw, _ := updateContractRead(t, r)
			updateContractWantPatch(t, raw, `[{"op":"add","path":"/custom","value":""},{"op":"replace","path":"/protected","value":false}]`)
			if r.Method != "PATCH" || r.Header.Get("Content-Type") != updateContractMedia {
				t.Error(r.Method, r.Header)
			}
			return imageContractJSONResponse(`{"id":"native","name":"native"}`), nil
		})
		value, e := nativeImages.New(c).Update(context.Background(), "fixed", nativeImages.UpdateOpts{nativeImages.UpdateImageProperty{Op: nativeImages.AddOp, Name: "custom", Value: ""}, nativeImages.ReplaceImageProtected{NewProtected: false}})
		if e != nil || value == nil || value.ID != "native" || calls.Load() != 1 {
			t.Fatal(value, e, calls.Load())
		}
	})
}

func TestImageUpdateContractsPatchAndProperties(t *testing.T) {
	t.Run("all typed helpers preserve order false zero empty and server policy", func(t *testing.T) {
		var calls atomic.Int32
		c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			raw, _ := updateContractRead(t, r)
			updateContractWantPatch(t, raw, `[{"op":"add","path":"/name","value":""},{"op":"add","path":"/visibility","value":"future"},{"op":"add","path":"/protected","value":false},{"op":"add","path":"/os_hidden","value":false},{"op":"add","path":"/owner","value":""},{"op":"add","path":"/container_format","value":"future"},{"op":"add","path":"/disk_format","value":"future"},{"op":"add","path":"/min_disk","value":-1},{"op":"add","path":"/min_ram","value":0},{"op":"add","path":"/tags","value":[]},{"op":"remove","path":"/a~1b~0c"},{"op":"replace","path":"/name","value":null},{"op":"add","path":"/id","value":"server-owned"}]`)
			return imageContractJSONResponse(`{}`), nil
		})
		value, e := image.New(c).UpdateImage(context.Background(), resource.ID("fixed"), image.WithUpdateImageName(""), image.WithUpdateImageVisibility("future"), image.WithUpdateImageProtected(false), image.WithUpdateImageHidden(false), image.WithUpdateImageOwner(""), image.WithUpdateImageContainerFormat("future"), image.WithUpdateImageDiskFormat("future"), image.WithUpdateImageMinDisk(-1), image.WithUpdateImageMinRAM(0), image.WithUpdateImageTags(), image.WithUpdateImageRemoveField("a/b~c"), image.WithUpdateImageChange(image.ImagePatch{Op: "replace", Path: "/name", Value: json.RawMessage(`null`)}), image.WithUpdateImageField("id", "server-owned"))
		if e != nil || value == nil || calls.Load() != 1 {
			t.Fatal(value, e, calls.Load())
		}
	})
	t.Run("complete replacement append sorted fields and immediate snapshots", func(t *testing.T) {
		raw := json.RawMessage(`9007199254740993`)
		fields := map[string]json.RawMessage{"z": json.RawMessage(`null`), "a/b~c": raw}
		tags := []string{"repeat", "repeat", "line\n"}
		headers := map[string]string{"X-Old": "old"}
		full := image.WithUpdateImageOpts(image.UpdateImageOpts{Headers: headers, Changes: []image.ImagePatch{{Op: "add", Path: "/discard", Value: json.RawMessage(`1`)}}})
		plural := image.WithUpdateImageChanges(image.ImagePatch{Op: "replace", Path: "/name", Value: json.RawMessage(`"first"`)})
		ownedFields := image.WithUpdateImageFields(fields)
		ownedTags := image.WithUpdateImageTags(tags...)
		raw[0] = '1'
		fields["z"] = json.RawMessage(`false`)
		tags[0] = "late"
		headers["X-Old"] = "late"
		c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			bytes, _ := updateContractRead(t, r)
			updateContractWantPatch(t, bytes, `[{"op":"replace","path":"/name","value":"first"},{"op":"add","path":"/name","value":"second"},{"op":"add","path":"/a~1b~0c","value":9007199254740993},{"op":"add","path":"/z","value":null},{"op":"add","path":"/tags","value":["repeat","repeat","line\n"]},{"op":"add","path":"/locations/-","value":{"url":"literal:% ?#","metadata":{}}}]`)
			if r.Header.Get("X-Old") != "old" || r.Header.Get("X-Merged") != "owned" || r.Header.Get("X-Last") != "yes" {
				t.Error(r.Header)
			}
			return imageContractJSONResponse(`{}`), nil
		})
		value, e := image.New(c).UpdateImage(context.Background(), resource.ID("fixed"), full, plural, image.WithUpdateImageField("name", "second"), ownedFields, image.WithUpdateImageFields(nil), ownedTags, image.WithUpdateImageChange(image.ImagePatch{Op: "add", Path: "/locations/-", Value: json.RawMessage(`{"url":"literal:% ?#","metadata":{}}`)}), image.WithUpdateImageHeaders(map[string]string{"X-Merged": "owned"}), image.WithUpdateImageHeader("X-Last", "yes"))
		if e != nil || value == nil {
			t.Fatal(value, e)
		}
	})
	t.Run("literal pointer tokens decode once and interior controls stay JSON data", func(t *testing.T) {
		c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			raw, _ := updateContractRead(t, r)
			updateContractWantPatch(t, raw, `[{"op":"add","path":"/a\nb~1c~0d%2F?~1#","value":{"huge":1e1000}},{"op":"remove","path":"/tricky~01/token\ninside"}]`)
			return imageContractJSONResponse(`{}`), nil
		})
		value, e := image.New(c).UpdateImage(context.Background(), resource.ID("fixed"), image.WithUpdateImageField("a\nb/c~d%2F?/#", json.RawMessage(`{"huge":1e1000}`)), image.WithUpdateImageChange(image.ImagePatch{Op: "remove", Path: "/tricky~01/token\ninside"}))
		if e != nil || value == nil {
			t.Fatal(value, e)
		}
	})
	t.Run("Set single PATCH sorted flattened exact JSON without lookup or coercion", func(t *testing.T) {
		fields := map[string]json.RawMessage{"z": json.RawMessage(`9007199254740993`), "a/b~c": json.RawMessage(`null`)}
		snapshot := image.WithSetImagePropertiesProperties(fields)
		fields["z"][0] = '1'
		var calls atomic.Int32
		c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			raw, _ := updateContractRead(t, r)
			updateContractWantPatch(t, raw, `[{"op":"add","path":"/a~1b~0c","value":null},{"op":"add","path":"/kernel","value":"literal-image-name"},{"op":"add","path":"/properties","value":{"nested":false}},{"op":"add","path":"/protected","value":false},{"op":"add","path":"/z","value":9007199254740993}]`)
			if r.Method != "PATCH" || r.URL.String() != taskContractsBase+"images/fixed" || r.Header.Get("X-Merged") != "owned" || r.Header.Get("X-Last") != "yes" || r.Header.Get("X-Discard") != "" {
				t.Error(r.Method, r.URL, r.Header)
			}
			return imageContractJSONResponse(`{}`), nil
		})
		value, e := image.New(c).SetImageProperties(context.Background(), resource.ID("fixed"), image.WithSetImagePropertiesProperty("discard", 1), image.WithSetImagePropertiesOpts(image.SetImagePropertiesOpts{Headers: map[string]string{"X-Discard": "discard"}, Properties: map[string]json.RawMessage{"discard": json.RawMessage(`1`)}}), image.WithSetImagePropertiesOpts(image.SetImagePropertiesOpts{}), snapshot, image.WithSetImagePropertiesProperty("kernel", "literal-image-name"), image.WithSetImagePropertiesProperty("properties", map[string]any{"nested": false}), image.WithSetImagePropertiesProperty("protected", false), image.WithSetImagePropertiesHeaders(map[string]string{"X-Merged": "owned"}), image.WithSetImagePropertiesHeader("X-Last", "yes"))
		if e != nil || value == nil || calls.Load() != 1 {
			t.Fatal(value, e, calls.Load())
		}
	})
	for _, tc := range []struct {
		name  string
		patch image.ImagePatch
	}{
		{"missing add", image.ImagePatch{Op: "add", Path: "/field"}},
		{"nonnull empty add", image.ImagePatch{Op: "add", Path: "/field", Value: json.RawMessage{}}},
		{"nonnull empty remove", image.ImagePatch{Op: "remove", Path: "/field", Value: json.RawMessage{}}},
		{"remove null", image.ImagePatch{Op: "remove", Path: "/field", Value: json.RawMessage(`null`)}},
		{"unknown op", image.ImagePatch{Op: "test", Path: "/field", Value: json.RawMessage(`false`)}},
		{"bad JSON", image.ImagePatch{Op: "add", Path: "/field", Value: json.RawMessage(`{`)}},
		{"bad UTF8 JSON", image.ImagePatch{Op: "add", Path: "/field", Value: json.RawMessage{'"', 0xff, '"'}}},
	} {
		t.Run(tc.name+" retained through each snapshot", func(t *testing.T) {
			for _, option := range []image.UpdateImageOption{image.WithUpdateImageOpts(image.UpdateImageOpts{Changes: []image.ImagePatch{tc.patch}}), image.WithUpdateImageChanges(tc.patch), image.WithUpdateImageChange(tc.patch)} {
				var calls atomic.Int32
				c := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected lookup") })
				value, e := image.New(c).UpdateImage(context.Background(), resource.Name("exact"), option)
				if value != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
					t.Fatal(value, e, calls.Load())
				}
			}
		})
	}
	for _, path := range []string{"", "field", "/", "/a/", "/a//b", "/a~", "/a~2b", "/ a", "/a ", "/\u001ca", "/a\u001f", "/a/\tchild", string([]byte{'/', 0xff})} {
		t.Run(fmt.Sprintf("unstable pointer %q", path), func(t *testing.T) {
			var calls atomic.Int32
			c := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected") })
			value, e := image.New(c).UpdateImage(context.Background(), resource.Name("exact"), image.WithUpdateImageChange(image.ImagePatch{Op: "remove", Path: path}))
			if value != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(value, e, calls.Load())
			}
		})
	}
	for _, key := range []string{"", " a", "a ", "\u001ca", "a\u001f", string([]byte{255})} {
		t.Run(fmt.Sprintf("unstable root key %q", key), func(t *testing.T) {
			var calls atomic.Int32
			c := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected") })
			s := image.New(c)
			cases := []func() (*image.ImageInfo, error){
				func() (*image.ImageInfo, error) {
					return s.UpdateImage(context.Background(), resource.Name("exact"), image.WithUpdateImageField(key, 1))
				},
				func() (*image.ImageInfo, error) {
					return s.UpdateImage(context.Background(), resource.Name("exact"), image.WithUpdateImageFields(map[string]json.RawMessage{key: json.RawMessage(`1`)}))
				},
				func() (*image.ImageInfo, error) {
					return s.UpdateImage(context.Background(), resource.Name("exact"), image.WithUpdateImageRemoveField(key))
				},
				func() (*image.ImageInfo, error) {
					return s.SetImageProperties(context.Background(), resource.Name("exact"), image.WithSetImagePropertiesProperty(key, 1))
				},
				func() (*image.ImageInfo, error) {
					return s.SetImageProperties(context.Background(), resource.Name("exact"), image.WithSetImagePropertiesProperties(map[string]json.RawMessage{key: json.RawMessage(`1`)}))
				},
			}
			for _, call := range cases {
				value, e := call()
				if value != nil || !errors.Is(e, resource.ErrInvalidOption) {
					t.Fatal(value, e)
				}
			}
			if calls.Load() != 0 {
				t.Fatal(calls.Load())
			}
		})
	}

	for _, raw := range []json.RawMessage{nil, {}, json.RawMessage(`{`), json.RawMessage{'"', 255, '"'}} {
		t.Run(fmt.Sprintf("Set raw value presence invalid %q", raw), func(t *testing.T) {
			for _, option := range []image.SetImagePropertiesOption{image.WithSetImagePropertiesOpts(image.SetImagePropertiesOpts{Properties: map[string]json.RawMessage{"field": raw}}), image.WithSetImagePropertiesProperties(map[string]json.RawMessage{"field": raw}), func(o *image.SetImagePropertiesOpts) error { o.Properties["field"] = raw; return nil }} {
				var calls atomic.Int32
				c := taskContractsClient(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					return nil, errors.New("unexpected Name lookup")
				})
				value, e := image.New(c).SetImageProperties(context.Background(), resource.Name("exact"), option)
				if value != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
					t.Fatal(value, e, calls.Load())
				}
			}
		})
	}
}

func TestImageUpdateContractsOptionsAndSource(t *testing.T) {
	for _, set := range []bool{false, true} {
		t.Run(fmt.Sprintf("callback config ownership set=%v", set), func(t *testing.T) {
			var aliasUpdate *image.UpdateImageOpts
			var aliasSet *image.SetImagePropertiesOpts
			var callbacks atomic.Int32
			c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
				raw, _ := updateContractRead(t, r)
				updateContractWantPatch(t, raw, `[{"op":"add","path":"/number","value":1}]`)
				if r.Header.Get("X-First") != "first" || r.Header.Get("X-Last") != "last" {
					t.Error(r.Header)
				}
				return imageContractJSONResponse(`{}`), nil
			})
			opts := updateContractOptions{
				update: []image.UpdateImageOption{func(o *image.UpdateImageOpts) error {
					callbacks.Add(1)
					o.Headers["X-First"] = "first"
					o.Changes = []image.ImagePatch{{Op: "add", Path: "/number", Value: json.RawMessage(`1`)}}
					aliasUpdate = o
					return nil
				}, func(o *image.UpdateImageOpts) error {
					callbacks.Add(1)
					aliasUpdate.Headers["X-First"] = "late"
					aliasUpdate.Changes[0].Value[0] = '2'
					o.Headers["X-Last"] = "last"
					return nil
				}},
				set: []image.SetImagePropertiesOption{func(o *image.SetImagePropertiesOpts) error {
					callbacks.Add(1)
					o.Headers["X-First"] = "first"
					o.Properties["number"] = json.RawMessage(`1`)
					aliasSet = o
					return nil
				}, func(o *image.SetImagePropertiesOpts) error {
					callbacks.Add(1)
					aliasSet.Headers["X-First"] = "late"
					aliasSet.Properties["number"][0] = '2'
					o.Headers["X-Last"] = "last"
					return nil
				}},
			}
			value, e := updateContractCall(image.New(c), context.Background(), resource.ID("fixed"), set, opts)
			if e != nil || value == nil || callbacks.Load() != 2 {
				t.Fatal(value, e, callbacks.Load())
			}
		})
		t.Run(fmt.Sprintf("factory Marshal once reusable owned options set=%v", set), func(t *testing.T) {
			var marshals, calls atomic.Int32
			raw := []byte(`{"n":1e1000}`)
			marshaler := taskContractsMarshaler{calls: &marshals, raw: raw}
			opts := updateContractOptions{update: []image.UpdateImageOption{image.WithUpdateImageField("literal", marshaler)}, set: []image.SetImagePropertiesOption{image.WithSetImagePropertiesProperty("literal", marshaler)}}
			if marshals.Load() != 2 {
				t.Fatal("both factories should snapshot once", marshals.Load())
			}
			raw[2] = 'x'
			c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				body, _ := updateContractRead(t, r)
				updateContractWantPatch(t, body, `[{"op":"add","path":"/literal","value":{"n":1e1000}}]`)
				return imageContractJSONResponse(`{}`), nil
			})
			s := image.New(c)
			var wg sync.WaitGroup
			for i := 0; i < 6; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					v, e := updateContractCall(s, context.Background(), resource.ID("fixed"), set, opts)
					if e != nil || v == nil {
						t.Error(v, e)
					}
				}()
			}
			wg.Wait()
			if marshals.Load() != 2 || calls.Load() != 6 {
				t.Fatal(marshals.Load(), calls.Load())
			}
		})
		t.Run(fmt.Sprintf("factory failure original key check and ordinary Go JSON set=%v", set), func(t *testing.T) {
			cause := errors.New("custom Marshal cause")
			var marshals, calls atomic.Int32
			c := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected") })
			s := image.New(c)
			badKey := string([]byte{255})
			m := taskContractsMarshaler{calls: &marshals, cause: cause}
			opts := updateContractOptions{update: []image.UpdateImageOption{image.WithUpdateImageField(badKey, m)}, set: []image.SetImagePropertiesOption{image.WithSetImagePropertiesProperty(badKey, m)}}
			value, e := updateContractCall(s, context.Background(), resource.Name("exact"), set, opts)
			if value != nil || !errors.Is(e, resource.ErrInvalidOption) || marshals.Load() != 0 || calls.Load() != 0 {
				t.Fatal(value, e, marshals.Load(), calls.Load())
			}
			opts = updateContractOptions{update: []image.UpdateImageOption{image.WithUpdateImageField("key", m)}, set: []image.SetImagePropertiesOption{image.WithSetImagePropertiesProperty("key", m)}}
			value, e = updateContractCall(s, context.Background(), resource.Name("exact"), set, opts)
			var encoding *json.MarshalerError
			if value != nil || !errors.Is(e, cause) || !errors.As(e, &encoding) || calls.Load() != 0 || marshals.Load() != 2 {
				t.Fatal(value, e, calls.Load(), marshals.Load())
			}
			opts = updateContractOptions{update: []image.UpdateImageOption{image.WithUpdateImageField("key", make(chan int))}, set: []image.SetImagePropertiesOption{image.WithSetImagePropertiesProperty("key", make(chan int))}}
			value, e = updateContractCall(s, context.Background(), resource.Name("exact"), set, opts)
			var unsupported *json.UnsupportedTypeError
			if value != nil || !errors.As(e, &unsupported) || calls.Load() != 0 {
				t.Fatal(value, e, calls.Load())
			}
		})
		t.Run(fmt.Sprintf("standard byte encoding and raw literal null set=%v", set), func(t *testing.T) {
			c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
				raw, _ := updateContractRead(t, r)
				updateContractWantPatch(t, raw, `[{"op":"add","path":"/bytes","value":"AP8="},{"op":"add","path":"/null","value":null}]`)
				return imageContractJSONResponse(`{}`), nil
			})
			opts := updateContractOptions{update: []image.UpdateImageOption{image.WithUpdateImageField("bytes", []byte{0, 255}), image.WithUpdateImageField("null", nil)}, set: []image.SetImagePropertiesOption{image.WithSetImagePropertiesProperty("bytes", []byte{0, 255}), image.WithSetImagePropertiesProperty("null", nil)}}
			v, e := updateContractCall(image.New(c), context.Background(), resource.ID("fixed"), set, opts)
			if e != nil || v == nil {
				t.Fatal(v, e)
			}
		})
		for _, mode := range []string{"nil context", "canceled context", "nil service", "nil client", "nil provider", "wrong type", "endpoint query", "source auth", "nil option", "callback error", "option protected media", "option bad header", "source endpoint drift", "source type drift", "source version drift", "provider replacement"} {
			t.Run(fmt.Sprintf("preflight %s set=%v", mode, set), func(t *testing.T) {
				var calls, callbacks atomic.Int32
				cause := errors.New("preflight cause")
				c := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected") })
				s := image.New(c)
				ctx := context.Context(context.Background())
				want := error(resource.ErrInvalidOption)
				opts := updateContractOptions{update: []image.UpdateImageOption{func(*image.UpdateImageOpts) error { callbacks.Add(1); return nil }}, set: []image.SetImagePropertiesOption{func(*image.SetImagePropertiesOpts) error { callbacks.Add(1); return nil }}}
				wantCallbacks := int32(0)
				switch mode {
				case "nil context":
					ctx = nil
				case "canceled context":
					var cancel context.CancelCauseFunc
					ctx, cancel = context.WithCancelCause(ctx)
					cancel(cause)
					want = context.Canceled
				case "nil service":
					s = nil
				case "nil client":
					s = image.New(nil)
				case "nil provider":
					c.ProviderClient = nil
				case "wrong type":
					c.Type = "compute"
					want = resource.ErrUnsupported
				case "endpoint query":
					c.Endpoint = taskContractsBase + "?unsafe=true"
				case "source auth":
					c.MoreHeaders = map[string]string{"X-Auth-Token": "not allowed"}
				case "nil option":
					opts = updateContractOptions{update: []image.UpdateImageOption{nil}, set: []image.SetImagePropertiesOption{nil}}
				case "callback error":
					want = cause
					wantCallbacks = 1
					opts = updateContractOptions{update: []image.UpdateImageOption{func(*image.UpdateImageOpts) error { callbacks.Add(1); return cause }}, set: []image.SetImagePropertiesOption{func(*image.SetImagePropertiesOpts) error { callbacks.Add(1); return cause }}}
				case "option protected media":
					opts = updateContractOptions{update: []image.UpdateImageOption{image.WithUpdateImageHeader("Content-Type", "other")}, set: []image.SetImagePropertiesOption{image.WithSetImagePropertiesHeader("Accept", "other")}}
				case "option bad header":
					opts = updateContractOptions{update: []image.UpdateImageOption{image.WithUpdateImageHeader("X-Bad", "line\n")}, set: []image.SetImagePropertiesOption{image.WithSetImagePropertiesHeader("X-Bad", "line\n")}}
				default:
					wantCallbacks = 1
					mutate := func() {
						switch mode {
						case "source endpoint drift":
							c.Endpoint = "https://foreign.invalid/v2/"
						case "source type drift":
							c.Type = "compute"
						case "source version drift":
							c.Microversion = "2.99"
						case "provider replacement":
							c.ProviderClient = &gophercloud.ProviderClient{}
						}
					}
					opts = updateContractOptions{update: []image.UpdateImageOption{func(*image.UpdateImageOpts) error { callbacks.Add(1); mutate(); return nil }}, set: []image.SetImagePropertiesOption{func(*image.SetImagePropertiesOpts) error { callbacks.Add(1); mutate(); return nil }}}
				}
				value, e := updateContractCall(s, ctx, resource.Name("exact"), set, opts)
				if value != nil || !errors.Is(e, want) || calls.Load() != 0 || callbacks.Load() != wantCallbacks {
					t.Fatal(value, e, calls.Load(), callbacks.Load())
				}
				if mode == "canceled context" && !errors.Is(e, cause) {
					t.Fatal(e)
				}
			})
		}
		for _, id := range []string{"", ".", "..", "slash/id", "back\\id", "control\n", string([]byte{255})} {
			t.Run(fmt.Sprintf("unsafe direct ID %q set=%v", id, set), func(t *testing.T) {
				var calls, callbacks atomic.Int32
				c := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected") })
				opts := updateContractOptions{update: []image.UpdateImageOption{func(*image.UpdateImageOpts) error { callbacks.Add(1); return nil }}, set: []image.SetImagePropertiesOption{func(*image.SetImagePropertiesOpts) error { callbacks.Add(1); return nil }}}
				value, e := updateContractCall(image.New(c), context.Background(), resource.ID(id), set, opts)
				if value != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 || callbacks.Load() != 0 {
					t.Fatal(value, e, calls.Load(), callbacks.Load())
				}
			})
		}
	}
}

func TestImageUpdateContractsCanonicalResponse(t *testing.T) {
	for _, set := range []bool{false, true} {
		t.Run(fmt.Sprintf("canonical raw model ownership set=%v", set), func(t *testing.T) {
			raw := []byte(imageContractObject)
			body := &taskContractsBody{Reader: bytes.NewReader(raw)}
			wire := taskContractsWire(200, body)
			wire.Header.Set("OpenStack-image-import-methods", "header-decoy")
			wire.Header.Set("Location", "https://foreign.invalid/retarget")
			var calls atomic.Int32
			c := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return wire, nil })
			v, e := updateContractCall(image.New(c), context.Background(), resource.ID("chosen"), set, updateContractOptions{})
			if e != nil || v == nil || *v.ID != "response-id" || *v.Name != "literal\nname" || *v.Size != 9007199254740993 || *v.VirtualSize != 9223372036854775807 || *v.MinDisk != -1 || *v.MinRAM != 0 || *v.Protected || !*v.Hidden || len(v.Tags) != 4 || len(v.Locations) != 1 || v.CreatedAt == nil || *v.CreatedAt != "literal-created" || v.Links != nil || string(v.Body["x-number"]) != "9007199254740995" || string(v.Properties["properties"]) != "[1e1000]" || v.Properties["id"] != nil || v.Body["openstack-image-import-methods"] != nil || v.StatusCode != 200 || body.closes.Load() != 1 || calls.Load() != 1 {
				t.Fatal(v, e, body.closes.Load(), calls.Load())
			}
			wire.Header.Set("X-Request-Id", "late")
			v.Body["x-number"][0] = '1'
			v.Locations[0].Metadata["integer"][0] = '1'
			raw[0] = '!'
			if v.Header.Get("X-Request-Id") != "actual-task" || string(v.Properties["x-number"]) != "9007199254740995" || string(v.Locations[0].Body["metadata"]) != `{"huge":1e1000,"integer":9007199254740995}` {
				t.Fatal("result aliases wire, projection, or nested metadata", v)
			}
		})
		for _, raw := range []string{`{}`, `{"id":null,"name":null,"protected":null,"size":null,"tags":null,"locations":null}`, `{"tags":[],"locations":[],"size":0,"protected":false}`} {
			t.Run(fmt.Sprintf("canonical presence %s set=%v", raw, set), func(t *testing.T) {
				c := taskContractsClient(func(*http.Request) (*http.Response, error) { return imageContractJSONResponse(raw), nil })
				v, e := updateContractCall(image.New(c), context.Background(), resource.ID("fixed"), set, updateContractOptions{})
				if e != nil || v == nil {
					t.Fatal(v, e)
				}
				if strings.Contains(raw, `"tags":[]`) && (v.Tags == nil || v.Locations == nil || v.Size == nil || *v.Size != 0 || v.Protected == nil || *v.Protected) {
					t.Fatal(v)
				}
				if raw == `{}` && (v.ID != nil || v.Size != nil || v.Tags != nil || v.Locations != nil) {
					t.Fatal(v)
				}
			})
		}
		for _, raw := range []string{`null`, `[]`, `{"id":false}`, `{"protected":"false"}`, `{"size":9007199254740993.0}`, `{"size":9223372036854775808}`, `{"tags":[null]}`, `{"locations":[null]}`, `{"locations":[{"metadata":[]}]}`, `{"min_ram":true}`, "{\"unknown\":\"\xff\"}"} {
			t.Run(fmt.Sprintf("atomic malformed model %q set=%v", raw, set), func(t *testing.T) {
				body := &taskContractsBody{Reader: strings.NewReader(raw)}
				var calls atomic.Int32
				c := taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return taskContractsWire(200, body), nil })
				v, e := updateContractCall(image.New(c), context.Background(), resource.ID("fixed"), set, updateContractOptions{})
				taskContractsProof(t, e, 200, []byte(raw))
				if v != nil || calls.Load() != 1 || body.closes.Load() != 1 {
					t.Fatal(v, e, calls.Load(), body.closes.Load())
				}
			})
		}
		for _, failure := range []string{"read", "close", "context", "combined opaque", "endpoint drift", "provider replacement"} {
			t.Run(fmt.Sprintf("accepted evidence %s set=%v", failure, set), func(t *testing.T) {
				readCause, closeCause, cancelCause := errors.New("read cause"), errors.New("close cause"), errors.New("custom cancellation cause")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				raw := []byte(imageContractObject)
				if failure == "read" {
					raw = []byte("read-prefix")
				}
				if failure == "combined opaque" {
					raw = []byte{'o', 'p', 0xff}
				}
				body := &taskContractsBody{Reader: bytes.NewReader(raw)}
				if failure == "read" || failure == "combined opaque" {
					body.Reader = taskContractsReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
				}
				if failure == "close" || failure == "combined opaque" {
					body.closeErr = closeCause
				}
				var calls, hooks atomic.Int32
				var c *gophercloud.ServiceClient
				body.onClose = func() {
					switch failure {
					case "context", "combined opaque":
						cancel(cancelCause)
					case "endpoint drift":
						c.Endpoint = "https://foreign.invalid/v2/"
					case "provider replacement":
						c.ProviderClient = &gophercloud.ProviderClient{}
					}
				}
				c = taskContractsClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return taskContractsWire(200, body), nil })
				c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks.Add(1)
					return nil
				}
				value, e := updateContractCall(image.New(c), ctx, resource.ID("fixed"), set, updateContractOptions{})
				taskContractsProof(t, e, 200, raw)
				if value != nil || calls.Load() != 1 || hooks.Load() != 0 || body.closes.Load() != 1 {
					t.Fatal(value, e, calls.Load(), hooks.Load(), body.closes.Load())
				}
				if (failure == "read" || failure == "combined opaque") && !errors.Is(e, readCause) {
					t.Fatal(e)
				}
				if (failure == "close" || failure == "combined opaque") && !errors.Is(e, closeCause) {
					t.Fatal(e)
				}
				if (failure == "context" || failure == "combined opaque") && (!errors.Is(e, context.Canceled) || !errors.Is(e, cancelCause)) {
					t.Fatal(e)
				}
				if (failure == "endpoint drift" || failure == "provider replacement") && !errors.Is(e, resource.ErrInvalidOption) {
					t.Fatal(e)
				}
			})
		}
	}
}

func TestImageUpdateContractsNativePolicy(t *testing.T) {
	for _, set := range []bool{false, true} {
		for _, code := range []int{201, 202, 204, 300, 403, 404, 409, 413, 415, 503} {
			t.Run(fmt.Sprintf("strict actual %d set=%v", code, set), func(t *testing.T) {
				var calls atomic.Int32
				body := &taskContractsBody{Reader: strings.NewReader("native failure")}
				c := taskContractsClient(func(r *http.Request) (*http.Response, error) { calls.Add(1); return taskContractsWire(code, body), nil })
				v, e := updateContractCall(image.New(c), context.Background(), resource.ID("fixed"), set, updateContractOptions{})
				var native gophercloud.ErrUnexpectedResponseCode
				var accepted *resource.ResponseError
				if v != nil || !errors.As(e, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || native.Method != "PATCH" || native.URL != taskContractsBase+"images/fixed" || string(native.Body) != "native failure" || native.ResponseHeader.Get("X-Request-Id") != "actual-task" || errors.As(e, &accepted) || calls.Load() != 1 || body.closes.Load() != 1 {
					t.Fatal(v, e, native, calls.Load(), body.closes.Load())
				}
			})
		}
		for _, mode := range []string{"503", "transport", "reauth", "identical JSON replacement"} {
			t.Run(fmt.Sprintf("prebody native policy %s set=%v", mode, set), func(t *testing.T) {
				cause := errors.New("transport cause")
				var calls, hooks atomic.Int32
				var bodies []*taskContractsBody
				var c *gophercloud.ServiceClient
				c = taskContractsClient(func(r *http.Request) (*http.Response, error) {
					n := calls.Add(1)
					raw, _ := updateContractRead(t, r)
					updateContractWantPatch(t, raw, `[{"op":"add","path":"/number","value":1}]`)
					if r.Method != "PATCH" || r.URL.String() != taskContractsBase+"images/fixed" || r.Header.Get("Content-Type") != updateContractMedia || r.Header.Get("Accept") != "application/json" {
						t.Error(r.Method, r.URL, r.Header)
					}
					if n == 1 && mode == "transport" {
						return nil, cause
					}
					code, text := 200, `{}`
					if n == 1 {
						code, text = 503, "original503"
						if mode == "reauth" {
							code = 401
						}
					}
					if n == 2 && (r.Header.Get("X-Auth-Token") != "fresh" || mode != "reauth" && r.Header.Get("X-Hook") != "advanced") {
						t.Error(r.Header)
					}
					b := &taskContractsBody{Reader: strings.NewReader(text)}
					bodies = append(bodies, b)
					return taskContractsWire(code, b), nil
				})
				if mode == "reauth" {
					c.ReauthFunc = func(context.Context) error { hooks.Add(1); c.SetToken("fresh"); return nil }
				} else {
					c.RetryFunc = func(_ context.Context, method, target string, o *gophercloud.RequestOpts, original error, _ uint) error {
						hooks.Add(1)
						if method != "PATCH" || target != taskContractsBase+"images/fixed" || o.JSONBody == nil || o.RawBody != nil || o.JSONResponse != nil || !o.KeepResponseBody || mode == "transport" && !errors.Is(original, cause) || mode != "transport" && !gophercloud.ResponseCodeIs(original, 503) {
							t.Error(method, target, original, o)
						}
						if mode == "identical JSON replacement" {
							o.JSONBody = []map[string]any{{"op": "add", "path": "/number", "value": json.Number("1")}}
						}
						if o.MoreHeaders == nil {
							o.MoreHeaders = make(map[string]string)
						}
						o.MoreHeaders["X-Hook"] = "advanced"
						c.SetToken("fresh")
						return nil
					}
				}
				provider := c.ProviderClient
				originalRetry := reflect.ValueOf(c.RetryFunc).Pointer()
				opts := updateContractOptions{update: []image.UpdateImageOption{image.WithUpdateImageField("number", 1)}, set: []image.SetImagePropertiesOption{image.WithSetImagePropertiesProperty("number", 1)}}
				v, e := updateContractCall(image.New(c), context.Background(), resource.ID("fixed"), set, opts)
				if e != nil || v == nil || calls.Load() != 2 || hooks.Load() != 1 || c.ProviderClient != provider || reflect.ValueOf(c.RetryFunc).Pointer() != originalRetry {
					t.Fatal(v, e, calls.Load(), hooks.Load())
				}
				for _, b := range bodies {
					if b.closes.Load() != 1 {
						t.Fatal(b.closes.Load())
					}
				}
			})
		}
	}
	for _, change := range []string{"changed JSON", "in-place JSON", "JSON null", "KeepResponseBody", "JSONResponse", "RawBody", "unsupported JSON", "expanded OkCodes"} {
		t.Run("fixed owned serialized request guard "+change, func(t *testing.T) {
			callbackCause, readCause, closeCause, cancelCause := errors.New("hook cause"), errors.New("expanded read cause"), errors.New("expanded Close cause"), errors.New("expanded context cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var calls, hooks, borrowedReads atomic.Int32
			borrowed := &taskContractsBody{Reader: taskContractsReader(func([]byte) (int, error) { borrowedReads.Add(1); return 0, io.EOF })}
			var bodies []*taskContractsBody
			c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				raw, _ := updateContractRead(t, r)
				updateContractWantPatch(t, raw, `[{"op":"add","path":"/number","value":1}]`)
				code, text := 503, "original503"
				if n == 2 {
					code, text = 204, "accepted-expanded-opaque"
				}
				b := &taskContractsBody{Reader: strings.NewReader(text)}
				if n == 2 {
					b.Reader = taskContractsReader(func(p []byte) (int, error) { return copy(p, text), readCause })
					b.closeErr = closeCause
					b.onClose = func() { cancel(cancelCause) }
				}
				bodies = append(bodies, b)
				return taskContractsWire(code, b), nil
			})
			c.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, original error, _ uint) error {
				hooks.Add(1)
				if !gophercloud.ResponseCodeIs(original, 503) {
					t.Error(original)
				}
				switch change {
				case "changed JSON":
					o.JSONBody = json.RawMessage(`[{"op":"add","path":"/number","value":2}]`)
				case "in-place JSON":
					raw, ok := o.JSONBody.(json.RawMessage)
					if !ok {
						t.Errorf("callback JSONBody %T", o.JSONBody)
					}
					needle := []byte(`"value":1`)
					at := bytes.Index(raw, needle)
					if at < 0 {
						t.Error(string(raw))
					} else {
						raw[at+len(needle)-1] = '2'
					}
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
			originalRetry := reflect.ValueOf(c.RetryFunc).Pointer()
			value, e := image.New(c).UpdateImage(ctx, resource.ID("fixed"), image.WithUpdateImageField("number", 1))
			if change == "expanded OkCodes" {
				var native gophercloud.ErrUnexpectedResponseCode
				var accepted *resource.ResponseError
				if value != nil || !errors.As(e, &native) || native.Actual != 204 || !reflect.DeepEqual(native.Expected, []int{200}) || native.Method != "PATCH" || native.URL != taskContractsBase+"images/fixed" || string(native.Body) != "accepted-expanded-opaque" || native.ResponseHeader.Get("X-Request-Id") != "actual-task" || errors.As(e, &accepted) || !errors.Is(e, readCause) || !errors.Is(e, closeCause) || !errors.Is(e, context.Canceled) || !errors.Is(e, cancelCause) || calls.Load() != 2 {
					t.Fatal(value, e, native, calls.Load())
				}
			} else if value != nil || !errors.Is(e, resource.ErrInvalidOption) || !errors.Is(e, callbackCause) || !gophercloud.ResponseCodeIs(e, 503) || calls.Load() != 1 {
				t.Fatal(value, e, calls.Load())
			}
			if change == "unsupported JSON" {
				var unsupported *json.UnsupportedTypeError
				if !errors.As(e, &unsupported) {
					t.Fatal(e)
				}
			}
			if hooks.Load() != 1 || borrowedReads.Load() != 0 || borrowed.closes.Load() != 0 || reflect.ValueOf(c.RetryFunc).Pointer() != originalRetry {
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
		t.Run("nested errors never become missing success "+mode, func(t *testing.T) {
			cause := errors.New("nested custom cause")
			nested := &gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{200}}
			var calls atomic.Int32
			c := taskContractsClient(func(*http.Request) (*http.Response, error) {
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
				c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					return errors.Join(cause, nested)
				}
			}
			if mode == "reauth" {
				c.ReauthFunc = func(context.Context) error { return errors.Join(cause, nested) }
			}
			value, e := image.New(c).SetImageProperties(context.Background(), resource.ID("fixed"), image.WithSetImagePropertiesProperty("key", 1))
			if value != nil || e == nil || calls.Load() != 1 {
				t.Fatal(value, e, calls.Load())
			}
			if mode == "reauth" {
				var native *gophercloud.ErrUnableToReauthenticate
				if !errors.As(e, &native) || !errors.Is(native.ErrReauth, cause) || !errors.Is(native.ErrReauth, nested) || !gophercloud.ResponseCodeIs(native.ErrOriginal, 401) {
					t.Fatal(e)
				}
			} else if !errors.Is(e, cause) || !errors.Is(e, nested) || mode == "callback" && !gophercloud.ResponseCodeIs(e, 503) {
				t.Fatal(e)
			}
		})
	}
	for _, mode := range []string{"same target", "foreign target", "other query", "changed method"} {
		t.Run("native bodyful redirect policy "+mode, func(t *testing.T) {
			var calls, redirects atomic.Int32
			var bodies []*taskContractsBody
			c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				raw, _ := updateContractRead(t, r)
				updateContractWantPatch(t, raw, `[{"op":"add","path":"/number","value":1}]`)
				if r.Method != "PATCH" || r.URL.String() != taskContractsBase+"images/fixed" {
					t.Error(r.Method, r.URL)
				}
				b := &taskContractsBody{Reader: strings.NewReader(`{}`)}
				bodies = append(bodies, b)
				if n == 2 {
					return taskContractsWire(200, b), nil
				}
				wire := taskContractsWire(307, b)
				target := taskContractsBase + "images/fixed"
				if mode == "foreign target" {
					target = "https://foreign.invalid/images/fixed"
				}
				if mode == "other query" {
					target += "?retarget=true"
				}
				wire.Header.Set("Location", target)
				return wire, nil
			})
			c.HTTPClient.CheckRedirect = func(r *http.Request, _ []*http.Request) error {
				redirects.Add(1)
				if mode == "changed method" {
					r.Method = "POST"
				}
				return nil
			}
			value, e := image.New(c).UpdateImage(context.Background(), resource.ID("fixed"), image.WithUpdateImageField("number", 1))
			if mode == "same target" {
				if e != nil || value == nil || calls.Load() != 2 {
					t.Fatal(value, e, calls.Load())
				}
			} else if value != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 1 {
				t.Fatal(value, e, calls.Load())
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
	t.Run("advanced native RetryFunc media policy remains native", func(t *testing.T) {
		var calls, hooks atomic.Int32
		c := taskContractsClient(func(r *http.Request) (*http.Response, error) {
			n := calls.Add(1)
			_, _ = updateContractRead(t, r)
			if n == 1 {
				return taskContractsWire(503, &taskContractsBody{Reader: strings.NewReader("original503")}), nil
			}
			if r.Header.Get("Content-Type") != "advanced/native" || r.Header.Get("Accept") != "advanced/accept" {
				t.Error(r.Header)
			}
			return imageContractJSONResponse(`{}`), nil
		})
		c.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			hooks.Add(1)
			o.MoreHeaders = map[string]string{"Content-Type": "advanced/native", "Accept": "advanced/accept"}
			return nil
		}
		value, e := image.New(c).UpdateImage(context.Background(), resource.ID("fixed"))
		if e != nil || value == nil || calls.Load() != 2 || hooks.Load() != 1 {
			t.Fatal(value, e, calls.Load(), hooks.Load())
		}
	})
}
