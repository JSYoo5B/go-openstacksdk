package images_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/images"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

const nativeUpdatePath = "/reverse/glance/v2/images/fixed"
const nativeUpdateMedia = "application/openstack-images-v2.1-json-patch"

func nativeUpdateClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("image", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/reverse/glance/v2/"
	return client
}
func nativeUpdateBody(t *testing.T, req *http.Request) json.RawMessage {
	t.Helper()
	raw, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
func nativeUpdateJSONEqual(t *testing.T, got json.RawMessage, want string) {
	t.Helper()
	var actual, expected any
	if err := json.Unmarshal(got, &actual); err != nil {
		t.Fatal(string(got), err)
	}
	if err := json.Unmarshal([]byte(want), &expected); err != nil {
		t.Fatal(want, err)
	}
	th.CheckDeepEquals(t, expected, actual)
}
func nativeUpdateOperation(t *testing.T, err error) {
	t.Helper()
	var operation *resource.OperationError
	if !errors.As(err, &operation) || operation.Operation != "Update" || operation.Resource != "images" {
		t.Fatal("generated operation context", err, operation)
	}
}

// The generated wrapper accepts concrete UpdateOpts, while its inherited Patch
// element interface remains usable for extensions. This intentionally verifies
// that interface; it is not an owned JSON-patch or receipt abstraction.
type nativeUpdateCustomPatch map[string]any

func (patch nativeUpdateCustomPatch) ToImagePatchMap() map[string]any { return patch }

type nativeUpdateMarshalFailure struct{ cause error }

func (value nativeUpdateMarshalFailure) MarshalJSON() ([]byte, error) { return nil, value.cause }

func TestNativeImageUpdateConcretePatchesAndOrder(t *testing.T) {
	cases := []struct {
		name  string
		patch images.Patch
		want  string
	}{
		{"visibility", images.UpdateVisibility{Visibility: "future"}, `{"op":"replace","path":"/visibility","value":"future"}`},
		{"hidden false", images.ReplaceImageHidden{NewHidden: false}, `{"op":"replace","path":"/os_hidden","value":false}`},
		{"name empty", images.ReplaceImageName{NewName: ""}, `{"op":"replace","path":"/name","value":""}`},
		{"checksum empty", images.ReplaceImageChecksum{Checksum: ""}, `{"op":"replace","path":"/checksum","value":""}`},
		{"tags nil", images.ReplaceImageTags{NewTags: nil}, `{"op":"replace","path":"/tags","value":null}`},
		{"tags allocated empty", images.ReplaceImageTags{NewTags: []string{}}, `{"op":"replace","path":"/tags","value":[]}`},
		{"tags duplicate literal", images.ReplaceImageTags{NewTags: []string{"a", "a", ""}}, `{"op":"replace","path":"/tags","value":["a","a",""]}`},
		{"min disk zero", images.ReplaceImageMinDisk{NewMinDisk: 0}, `{"op":"replace","path":"/min_disk","value":0}`},
		{"min ram negative forwarded", images.ReplaceImageMinRam{NewMinRam: -1}, `{"op":"replace","path":"/min_ram","value":-1}`},
		{"protected false", images.ReplaceImageProtected{NewProtected: false}, `{"op":"replace","path":"/protected","value":false}`},
		{"property add empty", images.UpdateImageProperty{Op: images.AddOp, Name: "custom", Value: ""}, `{"op":"add","path":"/custom","value":""}`},
		{"property replace raw pointer path", images.UpdateImageProperty{Op: images.ReplaceOp, Name: "a/b~c", Value: "literal"}, `{"op":"replace","path":"/a/b~c","value":"literal"}`},
		{"property remove omits value", images.UpdateImageProperty{Op: images.RemoveOp, Name: "custom", Value: "ignored"}, `{"op":"remove","path":"/custom"}`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := nativeUpdateClient(cloud)
			client.MoreHeaders = map[string]string{"X-Source": "direct"}
			client.Microversion = "2.10"
			var calls atomic.Int32
			cloud.Mux.HandleFunc(nativeUpdatePath, func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				if req.Method != http.MethodPatch || req.URL.Path != nativeUpdatePath || req.URL.RawQuery != "" || req.Header.Get("Content-Type") != nativeUpdateMedia || req.Header.Get("Accept") != "application/json" || req.Header.Get("X-Source") != "direct" || req.Header.Get("X-Auth-Token") != "test-token" || req.Header.Get("OpenStack-API-Version") != "image 2.10" {
					t.Error(req.Method, req.URL, req.Header)
				}
				nativeUpdateJSONEqual(t, nativeUpdateBody(t, req), "["+test.want+"]")
				testcloud.JSON(w, 200, `{"id":"returned"}`)
			})
			got, err := images.New(client).Update(context.Background(), "fixed", images.UpdateOpts{test.patch})
			if err != nil || got == nil || got.ID != "returned" || calls.Load() != 1 {
				t.Fatal(got, err, calls.Load())
			}
		})
	}
	t.Run("all operations preserve caller order", func(t *testing.T) {
		cloud := testcloud.New(t)
		opts := make(images.UpdateOpts, 0, len(cases))
		want := "["
		for index, test := range cases {
			opts = append(opts, test.patch)
			if index > 0 {
				want += ","
			}
			want += test.want
		}
		want += "]"
		cloud.Mux.HandleFunc(nativeUpdatePath, func(w http.ResponseWriter, req *http.Request) {
			nativeUpdateJSONEqual(t, nativeUpdateBody(t, req), want)
			testcloud.JSON(w, 200, `{}`)
		})
		got, err := images.New(nativeUpdateClient(cloud)).Update(context.Background(), "fixed", opts)
		if got == nil || err != nil {
			t.Fatal(got, err)
		}
	})
}

func TestNativeImageUpdateEmptyReplacementAndInheritedPatchInterface(t *testing.T) {
	for _, test := range []struct {
		name    string
		opts    images.UpdateOpts
		options []images.UpdateOption
		want    string
	}{
		{"nil opts", nil, nil, `[]`}, {"allocated empty opts", images.UpdateOpts{}, nil, `[]`},
		{"WithUpdateOptions replaces base", images.UpdateOpts{images.ReplaceImageName{NewName: "discard"}}, []images.UpdateOption{images.WithUpdateOptions(images.UpdateOpts{images.ReplaceImageProtected{NewProtected: false}})}, `[{"op":"replace","path":"/protected","value":false}]`},
		{"WithUpdateOptions nil clears base", images.UpdateOpts{images.ReplaceImageName{NewName: "discard"}}, []images.UpdateOption{images.WithUpdateOptions(nil)}, `[]`},
		{"nil-error option mutates typed patch set", nil, []images.UpdateOption{func(config *request.Config[images.UpdateOpts]) error {
			config.Options = images.UpdateOpts{images.ReplaceImageMinRam{NewMinRam: 0}}
			return nil
		}}, `[{"op":"replace","path":"/min_ram","value":0}]`},
		{"custom inherited Patch preserves extension", images.UpdateOpts{nativeUpdateCustomPatch{"op": "test", "path": "/custom", "value": json.RawMessage(`{"nested":[false,null]}`)}}, nil, `[{"op":"test","path":"/custom","value":{"nested":[false,null]}}]`},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(nativeUpdatePath, func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				nativeUpdateJSONEqual(t, nativeUpdateBody(t, req), test.want)
				testcloud.JSON(w, 200, `{}`)
			})
			got, err := images.New(nativeUpdateClient(cloud)).Update(context.Background(), "fixed", test.opts, test.options...)
			if got == nil || err != nil || calls.Load() != 1 {
				t.Fatal(got, err, calls.Load())
			}
		})
	}
}

func TestNativeImageUpdateRejectsRequestExtensionsAndMarshalFailuresBeforeHTTP(t *testing.T) {
	marker := errors.New("caller option or marshal failure")
	for _, test := range []struct {
		name    string
		opts    images.UpdateOpts
		options []images.UpdateOption
		cause   error
	}{
		{"nil option", nil, []images.UpdateOption{nil}, resource.ErrInvalidOption},
		{"JSON extension", nil, []images.UpdateOption{request.WithField[images.UpdateOpts]("extra", false)}, resource.ErrInvalidOption},
		{"query extension", nil, []images.UpdateOption{request.WithQuery[images.UpdateOpts]("extra", "x")}, resource.ErrInvalidOption},
		{"header extension", nil, []images.UpdateOption{request.WithHeader[images.UpdateOpts]("X-Extra", "x")}, resource.ErrInvalidOption},
		{"argument extension", nil, []images.UpdateOption{request.WithArgument[images.UpdateOpts]("extra", false)}, resource.ErrInvalidOption},
		{"option error", nil, []images.UpdateOption{func(*request.Config[images.UpdateOpts]) error { return marker }}, marker},
		{"custom patch marshal error", images.UpdateOpts{nativeUpdateCustomPatch{"op": "test", "path": "/custom", "value": nativeUpdateMarshalFailure{cause: marker}}}, nil, marker},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(nativeUpdatePath, func(w http.ResponseWriter, req *http.Request) { calls.Add(1); testcloud.JSON(w, 200, `{}`) })
			got, err := images.New(nativeUpdateClient(cloud)).Update(context.Background(), "fixed", test.opts, test.options...)
			if got != nil || !errors.Is(err, test.cause) || calls.Load() != 0 {
				t.Fatal(got, err, calls.Load())
			}
			nativeUpdateOperation(t, err)
		})
	}
}

func TestNativeImageUpdateStrictHTTPContextAndRetryPolicy(t *testing.T) {
	for _, code := range []int{201, 203, 204, 404, 503} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(nativeUpdatePath, func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Native-Proof", "actual")
				testcloud.JSON(w, code, `{"id":"rejected"}`)
			})
			got, err := images.New(nativeUpdateClient(cloud)).Update(context.Background(), "fixed", nil)
			var native gophercloud.ErrUnexpectedResponseCode
			if got != nil || !errors.As(err, &native) || native.Actual != code || native.Method != http.MethodPatch || native.URL != cloud.Server.URL+nativeUpdatePath || !reflect.DeepEqual(native.Expected, []int{200}) || native.ResponseHeader.Get("X-Native-Proof") != "actual" || calls.Load() != 1 {
				t.Fatal(got, err, native, calls.Load())
			}
			wantBody := `{"id":"rejected"}`
			if code == 204 {
				wantBody = ""
			}
			th.AssertEquals(t, wantBody, string(native.Body))
			nativeUpdateOperation(t, err)
		})
	}
	t.Run("canceled caller never reaches HTTP", func(t *testing.T) {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc(nativeUpdatePath, func(w http.ResponseWriter, req *http.Request) { calls.Add(1); testcloud.JSON(w, 200, `{}`) })
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		got, err := images.New(nativeUpdateClient(cloud)).Update(ctx, "fixed", nil)
		if got != nil || !errors.Is(err, context.Canceled) || calls.Load() != 0 {
			t.Fatal(got, err, calls.Load())
		}
		nativeUpdateOperation(t, err)
	})
	t.Run("native rejected-response retry retains patch and live token", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := nativeUpdateClient(cloud)
		var calls atomic.Int32
		ctx := context.WithValue(context.Background(), "native-update-proof", "caller")
		cloud.Mux.HandleFunc(nativeUpdatePath, func(w http.ResponseWriter, req *http.Request) {
			call := calls.Add(1)
			nativeUpdateJSONEqual(t, nativeUpdateBody(t, req), `[{"op":"replace","path":"/name","value":"new"}]`)
			expected := "test-token"
			if call == 2 {
				expected = "rotated"
			}
			if req.Header.Get("X-Auth-Token") != expected {
				t.Error(req.Header)
			}
			if call == 1 {
				testcloud.JSON(w, 503, `{"message":"retry"}`)
				return
			}
			testcloud.JSON(w, 200, `{"id":"after retry"}`)
		})
		hooks := 0
		client.RetryFunc = func(hookCtx context.Context, method, target string, opts *gophercloud.RequestOpts, err error, retries uint) error {
			hooks++
			if hookCtx.Value("native-update-proof") != "caller" || method != http.MethodPatch || target != cloud.Server.URL+nativeUpdatePath || !gophercloud.ResponseCodeIs(err, 503) || retries != 1 || !reflect.DeepEqual(opts.OkCodes, []int{200}) {
				t.Error(method, target, err, retries)
			}
			client.SetToken("rotated")
			return nil
		}
		got, err := images.New(client).Update(ctx, "fixed", images.UpdateOpts{images.ReplaceImageName{NewName: "new"}})
		if got == nil || err != nil || got.ID != "after retry" || calls.Load() != 2 || hooks != 1 {
			t.Fatal(got, err, calls.Load(), hooks)
		}
	})
	t.Run("native accepted JSON decoder may invoke RetryFunc", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := nativeUpdateClient(cloud)
		var calls atomic.Int32
		cloud.Mux.HandleFunc(nativeUpdatePath, func(w http.ResponseWriter, req *http.Request) {
			if calls.Add(1) == 1 {
				testcloud.JSON(w, 200, `{"broken":`)
				return
			}
			testcloud.JSON(w, 200, `{"id":"decoded retry"}`)
		})
		hooks := 0
		client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
			hooks++
			var syntax *json.SyntaxError
			if err == nil || !(errors.Is(err, io.ErrUnexpectedEOF) || errors.As(err, &syntax)) {
				t.Error(err)
			}
			return nil
		}
		got, err := images.New(client).Update(context.Background(), "fixed", nil)
		if got == nil || err != nil || got.ID != "decoded retry" || calls.Load() != 2 || hooks != 1 {
			t.Fatal(got, err, calls.Load(), hooks)
		}
	})
}

func TestNativeImageUpdateExtractsRichImageUnknownPropertiesAndLastHeaders(t *testing.T) {
	const raw = `{"id":"response id","name":"literal name","status":"future","tags":["a","a",""],"container_format":"future","disk_format":"future","min_disk":-1,"min_ram":0,"owner":"owner","protected":false,"visibility":"future","os_hidden":true,"checksum":"","size":42.75,"metadata":{"known":"metadata"},"created_at":"2024-01-02T03:04:05Z","updated_at":"2024-01-03T04:05:06Z","file":"foreign file","schema":"foreign schema","virtual_size":-1,"self":"passive self","properties":"literal properties","vendor":{"n":9007199254740993,"false":false},"nullable":null,"openstack-image-import-methods":"body import","openstack-image-store-ids":"body store"}`
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc(nativeUpdatePath, func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("OpenStack-image-import-methods", "discarded first")
		w.Header().Add("OpenStack-image-import-methods", " last,, second, ")
		w.Header().Add("OpenStack-image-store-ids", "discarded store")
		w.Header().Add("OpenStack-image-store-ids", " store2,store3 ")
		testcloud.JSON(w, 200, raw)
	})
	got, err := images.New(nativeUpdateClient(cloud)).Update(context.Background(), "fixed", nil)
	if got == nil || err != nil || got.ID != "response id" || got.Name != "literal name" || got.Status != "future" || got.ContainerFormat != "future" || got.DiskFormat != "future" || got.MinDiskGigabytes != -1 || got.MinRAMMegabytes != 0 || got.Owner != "owner" || got.Protected || !got.Hidden || got.Visibility != "future" || got.SizeBytes != 42 || got.VirtualSize != -1 || got.File != "foreign file" || got.Schema != "foreign schema" {
		t.Fatal(got, err)
	}
	th.CheckDeepEquals(t, []string{"a", "a", ""}, got.Tags)
	th.CheckDeepEquals(t, map[string]string{"known": "metadata"}, got.Metadata)
	th.CheckDeepEquals(t, []string{"last", " second"}, got.OpenStackImageImportMethods)
	th.CheckDeepEquals(t, []string{"store2", "store3"}, got.OpenStackImageStoreIDs)
	wantProperties := map[string]any{"properties": "literal properties", "vendor": map[string]any{"n": float64(9007199254740992), "false": false}, "nullable": nil}
	th.CheckDeepEquals(t, wantProperties, got.Properties)
	created, _ := time.Parse(time.RFC3339, "2024-01-02T03:04:05Z")
	updated, _ := time.Parse(time.RFC3339, "2024-01-03T04:05:06Z")
	if !got.CreatedAt.Equal(created) || !got.UpdatedAt.Equal(updated) {
		t.Fatal(got.CreatedAt, got.UpdatedAt)
	}
}

func TestNativeImageUpdateExtractorErrorsKeepNativePartialResults(t *testing.T) {
	for _, test := range []struct {
		name, raw string
		present   bool
		id        string
		good      bool
	}{
		{"empty object", `{}`, true, "", true}, {"JSON null", `null`, false, "", true},
		{"malformed JSON", `{"broken":`, false, "", false}, {"empty response", "", false, "", false},
		{"wrong name type", `{"id":"partial","name":42}`, true, "", false},
		{"wrong metadata type", `{"id":"partial","metadata":[1]}`, true, "", false},
		{"wrong properties type", `{"id":"partial","properties":{}}`, true, "", false},
		{"wrong size retains assigned fields", `{"id":"partial","size":"wrong"}`, true, "partial", false},
		{"wrong tags type", `{"id":"partial","tags":false}`, true, "", false},
		{"invalid timestamp", `{"id":"partial","created_at":"literal date"}`, true, "", false},
		{"top-level array", `[]`, true, "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc(nativeUpdatePath, func(w http.ResponseWriter, req *http.Request) { calls.Add(1); testcloud.JSON(w, 200, test.raw) })
			got, err := images.New(nativeUpdateClient(cloud)).Update(context.Background(), "fixed", nil)
			if (got != nil) != test.present || (err == nil) != test.good || calls.Load() != 1 {
				t.Fatal(got, err, calls.Load())
			}
			if got != nil && got.ID != test.id {
				t.Fatal("native extraction partial changed", got, err)
			}
			if err != nil {
				nativeUpdateOperation(t, err)
				var proof *resource.ResponseError
				if errors.As(err, &proof) {
					t.Fatal("native wrapper fabricated owned receipt", proof)
				}
			}
		})
	}
}
