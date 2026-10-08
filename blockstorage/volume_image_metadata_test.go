package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func vimRequest(t *testing.T, req *http.Request, id, body, version, token string) {
	t.Helper()
	var raw []byte
	var err error
	if req.Body != nil {
		raw, err = io.ReadAll(req.Body)
	}
	method, path := http.MethodPost, vsaBase+"volumes/"+url.PathEscape(id)+"/action"
	if body == "" {
		method, path = http.MethodGet, vsaBase+"volumes/"+url.PathEscape(id)
	}
	current := ""
	if version != "" {
		current = "volume " + version
	}
	if err != nil || string(raw) != body || req.Method != method || req.URL.EscapedPath() != path || req.URL.RawQuery != "" || req.Header.Get("X-Source") != "original" || req.Header.Get("X-Auth-Token") != token || req.Header.Get("OpenStack-API-Version") != current || req.Header.Get("X-OpenStack-Volume-API-Version") != version || req.ContentLength != int64(len(body)) || len(req.TransferEncoding) != 0 {
		t.Error("image metadata changed body, fixed route, framing or captured policy", req.Method, req.URL, string(raw), req.Header, req.ContentLength, req.TransferEncoding, err)
	}
}

func TestSetVolumeImageMetadataPreservesEmptyMapsRawValuesAndOwnedFactories(t *testing.T) {
	cases := []struct {
		name, body string
		options    []blockstorage.VolumeImageMetadataOption
	}{
		{"omitted map", `{"os-set_image_metadata":{"metadata":{}}}`, nil},
		{"typed nil map", `{"os-set_image_metadata":{"metadata":{}}}`, []blockstorage.VolumeImageMetadataOption{blockstorage.WithVolumeImageMetadata(nil)}},
		{"empty HTTP204 acknowledgement", `{"os-set_image_metadata":{"metadata":{}}}`, nil},
		{"raw nil map", `{"os-set_image_metadata":{"metadata":{}}}`, []blockstorage.VolumeImageMetadataOption{blockstorage.WithVolumeImageMetadataRaw(nil)}},
		{"empty keys and literal text", `{"os-set_image_metadata":{"metadata":{"":"","image_name":" /?%#\n\u0000한글"}}}`, []blockstorage.VolumeImageMetadataOption{blockstorage.WithVolumeImageMetadata(map[string]string{"": "", "image_name": " /?%#\n\x00한글"})}},
		{"raw precision and nested JSON", `{"os-set_image_metadata":{"metadata":{"bool":false,"large":9007199254740993,"nested":{"x":[null,"value"]},"null":null}}}`, []blockstorage.VolumeImageMetadataOption{blockstorage.WithVolumeImageMetadataRaw(map[string]json.RawMessage{"bool": json.RawMessage(`false`), "large": json.RawMessage(`9007199254740993`), "nested": json.RawMessage(`{"x":[null,"value"]}`), "null": json.RawMessage(`null`)})}},
		{"replacement removes earlier fields; value merges", `{"os-set_image_metadata":{"metadata":{"next":"2","tail":"3"}}}`, []blockstorage.VolumeImageMetadataOption{blockstorage.WithVolumeImageMetadataValue("old", "1"), blockstorage.WithVolumeImageMetadata(map[string]string{"next": "2"}), blockstorage.WithVolumeImageMetadataValue("tail", "3")}},
		{"full replacement clears earlier map", `{"os-set_image_metadata":{"metadata":{}}}`, []blockstorage.VolumeImageMetadataOption{blockstorage.WithVolumeImageMetadataValue("old", "1"), blockstorage.WithVolumeImageMetadataOptions(blockstorage.VolumeImageMetadataOpts{})}},
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.90")
			id := "한글-ID"
			var calls atomic.Int32
			reply := []byte{0xff, 0x00, '!'}
			code := []int{200, 203, 302, 399}[index%4]
			if tc.name == "empty HTTP204 acknowledgement" {
				reply = nil
				code = 204
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				vimRequest(t, req, id, tc.body, "3.90", "test-token")
				w.Header().Set("X-Proof", "set action")
				w.WriteHeader(code)
				_, _ = w.Write(reply)
			})
			result, err := blockstorage.SetVolumeImageMetadata(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: id}, tc.options...)
			if err != nil || result == nil || !result.Completed || result.VolumeID != id || result.Microversion != "3.90" || len(result.Discovery) != 0 || result.Applied == nil || result.Applied.StatusCode != code || result.Applied.Header.Get("X-Proof") != "set action" || !bytes.Equal(result.Applied.Body, reply) || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
	t.Run("factory map bytes and retained callback config cannot change POST", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := vsaClient(cloud, "3.60")
		raw := json.RawMessage(`9007199254740993`)
		source := map[string]json.RawMessage{"large": raw}
		factory := blockstorage.WithVolumeImageMetadataRaw(source)
		raw[0] = '1'
		source["late"] = json.RawMessage(`true`)
		var retained *blockstorage.VolumeImageMetadataOpts
		var callbacks, calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			calls.Add(1)
			vimRequest(t, req, "id", `{"os-set_image_metadata":{"metadata":{"large":9007199254740993}}}`, "3.60", "test-token")
			w.WriteHeader(203)
		})
		result, err := blockstorage.SetVolumeImageMetadata(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, factory, func(o *blockstorage.VolumeImageMetadataOpts) error { callbacks.Add(1); retained = o; return nil }, func(*blockstorage.VolumeImageMetadataOpts) error {
			callbacks.Add(1)
			retained.Metadata["large"][0] = '2'
			retained.Metadata["late"] = json.RawMessage(`false`)
			client.MoreHeaders["x-source"] = "later ordinary header"
			return nil
		})
		if err != nil || result == nil || !result.Completed || callbacks.Load() != 2 || calls.Load() != 1 {
			t.Fatal(result, err, callbacks.Load(), calls.Load())
		}
	})
}

func TestDeleteVolumeImageMetadataExplicitKeysKeepOrderDuplicatesAndEagerValidation(t *testing.T) {
	t.Run("literal order and duplicates; owned variadic factory", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := vsaClient(cloud, "3.60")
		keys := []string{"z", "", "z", " /?%#\n\x00한글"}
		factory := blockstorage.WithVolumeImageMetadataDeleteKeys(keys...)
		keys[0] = "mutated caller slice"
		bodies := []string{`{"os-unset_image_metadata":{"key":"z"}}`, `{"os-unset_image_metadata":{"key":""}}`, `{"os-unset_image_metadata":{"key":"z"}}`, `{"os-unset_image_metadata":{"key":" /?%#\n\u0000한글"}}`}
		wantKeys := []string{"z", "", "z", " /?%#\n\x00한글"}
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			n := calls.Add(1)
			if n > int32(len(bodies)) {
				t.Error("extra deletion HTTP", n)
				w.WriteHeader(500)
				return
			}
			vimRequest(t, req, "id", bodies[n-1], "3.60", "test-token")
			w.Header().Set("X-Proof", "key acknowledgement")
			w.WriteHeader([]int{200, 203, 302, 399}[n-1])
			_, _ = w.Write([]byte{0xff, 0x00, byte(n)})
		})
		result, err := blockstorage.DeleteVolumeImageMetadata(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, factory)
		if err != nil || result == nil || !result.Completed || result.VolumeID != "id" || result.Microversion != "3.60" || result.Observed != nil || result.Failed != nil || len(result.Discovery) != 0 || len(result.Deleted) != 4 || calls.Load() != 4 {
			t.Fatal(result, err, calls.Load())
		}
		for i, item := range result.Deleted {
			if item.Key != wantKeys[i] || item.Response == nil || item.Response.StatusCode != []int{200, 203, 302, 399}[i] || !bytes.Equal(item.Response.Body, []byte{0xff, 0x00, byte(i + 1)}) {
				t.Fatal(i, item)
			}
		}
		result.Deleted[0].Response.Body[0] = '!'
		result.Deleted[0].Response.Header.Set("X-Proof", "changed first")
		if result.Deleted[2].Response.Body[0] != 0xff || result.Deleted[2].Response.Header.Get("X-Proof") != "key acknowledgement" {
			t.Fatal("duplicate entries share acknowledgement ownership", result.Deleted)
		}
	})
	for _, kind := range []string{"variadic zero keys", "explicit nil slice", "explicit empty slice"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				t.Error("empty explicit delete performed HTTP", req.URL)
				w.WriteHeader(500)
			})
			option := blockstorage.WithVolumeImageMetadataDeleteKeys()
			if kind != "variadic zero keys" {
				var keys []string
				if kind == "explicit empty slice" {
					keys = []string{}
				}
				option = blockstorage.WithVolumeImageMetadataDeleteOptions(blockstorage.VolumeImageMetadataDeleteOpts{Keys: &keys})
			}
			result, err := blockstorage.DeleteVolumeImageMetadata(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, option)
			if err != nil || result == nil || !result.Completed || result.Observed != nil || result.Failed != nil || len(result.Deleted) != 0 || len(result.Discovery) != 0 || calls.Load() != 0 {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
	for _, kind := range []string{"nil context", "canceled context", "nil client", "wrong service", "unsafe ID"} {
		t.Run("explicit empty still validates "+kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "")
			ctx := vsaContext(t)
			id := "id"
			cause := errors.New("empty image deletion cancellation")
			want := resource.ErrInvalidOption
			switch kind {
			case "nil context":
				ctx = nil
			case "canceled context":
				canceled, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx = canceled
				want = context.Canceled
			case "nil client":
				client = nil
			case "wrong service":
				client.Type = "compute"
				want = resource.ErrUnsupported
			case "unsafe ID":
				id = "id/other"
			}
			var calls, originals atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) { calls.Add(1); w.WriteHeader(500) })
			result, err := blockstorage.DeleteVolumeImageMetadata(ctx, client, blockstorage.VolumeActionRequest{VolumeID: id}, func(o *blockstorage.VolumeImageMetadataDeleteOpts) error {
				originals.Add(1)
				keys := []string{}
				o.Keys = &keys
				return nil
			})
			vsaOperation(t, err, "DeleteVolumeImageMetadata")
			var proof *resource.ResponseError
			if result != nil || !errors.Is(err, want) || errors.As(err, &proof) || calls.Load() != 0 || originals.Load() != 0 || kind == "canceled context" && !errors.Is(err, cause) {
				t.Fatal(result, err, proof, calls.Load(), originals.Load())
			}
		})
	}
	t.Run("later invalid key fails before discovery or earlier valid key", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := vsaClient(cloud, "")
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) { calls.Add(1); w.WriteHeader(500) })
		result, err := blockstorage.DeleteVolumeImageMetadata(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, blockstorage.WithVolumeImageMetadataDeleteKeys("first", string([]byte{0xff})))
		vsaOperation(t, err, "DeleteVolumeImageMetadata")
		var proof *resource.ResponseError
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || calls.Load() != 0 {
			t.Fatal(result, err, proof, calls.Load())
		}
	})
}

func TestDeleteVolumeImageMetadataAllReadsOnlyCanonicalImageFieldAndKeepsObservation(t *testing.T) {
	cases := []struct {
		name, reply string
		keys        []string
		invalid     bool
	}{
		{"image keys sorted; unrelated fields and values ignored", `{"volume":{"id":"changed/unsafe","name":[],"location":false,"metadata":{"wrong":true},"volume_image_metadata":{"z":null,"":false,"a":{"untyped":[1]}}}}`, []string{"", "a", "z"}, false},
		{"absent image ignores ordinary metadata", `{"volume":{"metadata":{"wrong":"present"}}}`, nil, false},
		{"null image", `{"volume":{"volume_image_metadata":null}}`, nil, false},
		{"empty image object", `{"volume":{"volume_image_metadata":{}}}`, nil, false},
		{"wrong case image field ignored", `{"volume":{"Volume_Image_Metadata":{"wrong":"present"}}}`, nil, false},
		{"nonobject image array", `{"volume":{"volume_image_metadata":[]}}`, nil, true},
		{"nonobject image scalar", `{"volume":{"volume_image_metadata":false}}`, nil, true},
		{"missing canonical envelope", `{"Volume":{"volume_image_metadata":{"wrong":"present"}}}`, nil, true},
		{"null canonical volume", `{"volume":null}`, nil, true},
		{"malformed accepted JSON", `{"volume":`, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.60")
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				n := calls.Add(1)
				if n == 1 {
					vimRequest(t, req, "id", "", "3.60", "test-token")
					w.Header().Set("X-Proof", "actual member observation")
					testcloud.JSON(w, 203, tc.reply)
					return
				}
				i := int(n) - 2
				if i >= len(tc.keys) {
					t.Error("default all followed wrong field or changed ID", req.URL, n)
					w.WriteHeader(500)
					return
				}
				encoded, _ := json.Marshal(tc.keys[i])
				vimRequest(t, req, "id", `{"os-unset_image_metadata":{"key":`+string(encoded)+`}}`, "3.60", "test-token")
				w.Header().Set("X-Proof", "current key action")
				w.WriteHeader(399)
				_, _ = w.Write([]byte{0xff, 0x00})
			})
			result, err := blockstorage.DeleteVolumeImageMetadata(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, blockstorage.WithVolumeImageMetadataDeleteAll())
			if result == nil || result.VolumeID != "id" || result.Observed == nil || result.Observed.StatusCode != 203 || string(result.Observed.Body) != tc.reply || result.Observed.Header.Get("X-Proof") != "actual member observation" || result.Failed != nil || len(result.Discovery) != 0 {
				t.Fatal(result, err, calls.Load())
			}
			if tc.invalid {
				vsaOperation(t, err, "DeleteVolumeImageMetadata")
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != tc.reply || result.Completed || len(result.Deleted) != 0 || calls.Load() != 1 {
					t.Fatal(result, err, proof, calls.Load())
				}
				proof.Body[0] = '!'
				if string(result.Observed.Body) != tc.reply {
					t.Fatal("observation aliases schema error proof")
				}
				return
			}
			if err != nil || !result.Completed || len(result.Deleted) != len(tc.keys) || calls.Load() != int32(1+len(tc.keys)) {
				t.Fatal(result, err, calls.Load())
			}
			for i, item := range result.Deleted {
				if item.Key != tc.keys[i] || item.Response == nil || item.Response.StatusCode != 399 || !bytes.Equal(item.Response.Body, []byte{0xff, 0x00}) {
					t.Fatal(i, item)
				}
			}
			if len(result.Deleted) > 0 {
				result.Observed.Body[0] = '!'
				result.Observed.Header.Set("X-Proof", "changed observation")
				if result.Deleted[0].Response.Header.Get("X-Proof") != "current key action" || result.Deleted[0].Response.Body[0] != 0xff {
					t.Fatal("member observation aliases action proof")
				}
			}
		})
	}
}

func TestVolumeImageMetadataNegotiatesOnceAndStopsAtCurrentNativeFailureWithoutBorrowedProof(t *testing.T) {
	for _, all := range []bool{false, true} {
		t.Run(map[bool]string{false: "explicit keys", true: "all keys"}[all], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "")
			var calls, posts atomic.Int32
			support := `{"version":{"id":"v3.0","min_version":"3.0","max_version":"3.99"}}`
			member := `{"volume":{"volume_image_metadata":{"c":false,"b":null,"a":"v"}}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				n := calls.Add(1)
				if n == 1 {
					vsaDiscoveryRequest(t, req, vsaVersionPath)
					w.Header().Set("X-Proof", "single batch discovery")
					testcloud.JSON(w, 300, support)
					return
				}
				if all && n == 2 {
					vimRequest(t, req, "id", "", "3.71", "test-token")
					w.Header().Set("X-Proof", "batch member")
					testcloud.JSON(w, 200, member)
					return
				}
				p := posts.Add(1)
				if p > 2 {
					t.Error("batch continued after first action failure", p)
					w.WriteHeader(500)
					return
				}
				key := map[int32]string{1: "a", 2: "b"}[p]
				vimRequest(t, req, "id", `{"os-unset_image_metadata":{"key":"`+key+`"}}`, "3.71", "test-token")
				if p == 1 {
					w.Header().Set("X-Proof", "first deletion")
					testcloud.JSON(w, 203, "opaque first")
				} else {
					w.Header().Set("X-Proof", "current native failure")
					testcloud.JSON(w, 400, `{"error":"second rejected"}`)
				}
			})
			option := blockstorage.WithVolumeImageMetadataDeleteKeys("a", "b", "c")
			if all {
				option = blockstorage.WithVolumeImageMetadataDeleteAll()
			}
			result, err := blockstorage.DeleteVolumeImageMetadata(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, option)
			vsaOperation(t, err, "DeleteVolumeImageMetadata")
			var native gophercloud.ErrUnexpectedResponseCode
			var proof *resource.ResponseError
			wantCalls := int32(3)
			if all {
				wantCalls = 4
			}
			if result == nil || result.VolumeID != "id" || result.Microversion != "3.71" || result.Completed || len(result.Discovery) != 1 || result.Discovery[0].Header.Get("X-Proof") != "single batch discovery" || string(result.Discovery[0].Body) != support || len(result.Deleted) != 1 || result.Deleted[0].Key != "a" || result.Deleted[0].Response == nil || result.Deleted[0].Response.StatusCode != 203 || string(result.Deleted[0].Response.Body) != "opaque first" || result.Failed == nil || result.Failed.Key != "b" || result.Failed.Response != nil || !errors.As(err, &native) || native.Actual != 400 || string(native.Body) != `{"error":"second rejected"}` || native.ResponseHeader.Get("X-Proof") != "current native failure" || errors.As(err, &proof) || calls.Load() != wantCalls || posts.Load() != 2 {
				t.Fatal(result, err, native, proof, calls.Load(), posts.Load())
			}
			if all {
				if result.Observed == nil || string(result.Observed.Body) != member || result.Observed.Header.Get("X-Proof") != "batch member" {
					t.Fatal(result.Observed)
				}
			} else if result.Observed != nil {
				t.Fatal("explicit keys acquired unnecessary member proof", result.Observed)
			}
		})
	}
	t.Run("malformed discovery blocks set with only actual discovery proof", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := vsaClient(cloud, "")
		var calls atomic.Int32
		reply := `{"version":null}`
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			calls.Add(1)
			vsaDiscoveryRequest(t, req, vsaVersionPath)
			w.Header().Set("X-Proof", "invalid discovery")
			testcloud.JSON(w, 200, reply)
		})
		result, err := blockstorage.SetVolumeImageMetadata(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"})
		vsaOperation(t, err, "SetVolumeImageMetadata")
		var proof *resource.ResponseError
		if result == nil || result.Completed || result.Applied != nil || len(result.Discovery) != 1 || string(result.Discovery[0].Body) != reply || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != reply || proof.Header.Get("X-Proof") != "invalid discovery" || calls.Load() != 1 {
			t.Fatal(result, err, proof, calls.Load())
		}
	})
}

func TestVolumeImageMetadataServiceFacadePreservesNativeSetAndPublicPreflight(t *testing.T) {
	cloud := testcloud.New(t)
	client := vsaClient(cloud, "3.60")
	api := volumes.New(client)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		n := calls.Add(1)
		switch n {
		case 1:
			vimRequest(t, req, "id", `{"os-set_image_metadata":{"metadata":null}}`, "3.60", "test-token")
			testcloud.JSON(w, 200, `{}`)
		case 2:
			vimRequest(t, req, "id", `{"os-set_image_metadata":{"metadata":{"old":"native"}}}`, "3.60", "test-token")
			w.Header().Set("X-Proof", "native set rejection")
			testcloud.JSON(w, 201, `{"metadata":{"old":"native"}}`)
		case 3:
			vimRequest(t, req, "id", `{"os-set_image_metadata":{"metadata":{}}}`, "3.60", "test-token")
			w.WriteHeader(203)
			_, _ = w.Write([]byte{0xff, 0x00})
		case 4:
			vimRequest(t, req, "id", `{"os-unset_image_metadata":{"key":""}}`, "3.60", "test-token")
			w.WriteHeader(200)
		case 5:
			vimRequest(t, req, "id", `{"os-unset_image_metadata":{"key":"a"}}`, "3.60", "test-token")
			w.WriteHeader(399)
		default:
			t.Error("unexpected service metadata HTTP", n, req.URL)
			w.WriteHeader(500)
		}
	})
	if err := api.SetImageMetadata(vsaContext(t), "id", volumes.ImageMetadataOpts{}); err != nil || calls.Load() != 1 {
		t.Fatal("native nil map/body or200 policy changed", err, calls.Load())
	}
	err := api.SetImageMetadata(vsaContext(t), "id", volumes.ImageMetadataOpts{Metadata: map[string]string{"old": "native"}})
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != 201 || len(native.Expected) != 1 || native.Expected[0] != 200 || native.ResponseHeader.Get("X-Proof") != "native set rejection" || calls.Load() != 2 {
		t.Fatal("new policy widened native SetImageMetadata", err, native, calls.Load())
	}
	set, err := api.SetVolumeImageMetadata(vsaContext(t), "id", volumes.WithVolumeImageMetadata(nil))
	if err != nil || set == nil || !set.Completed || set.Applied == nil || set.Applied.StatusCode != 203 || !bytes.Equal(set.Applied.Body, []byte{0xff, 0x00}) || calls.Load() != 3 {
		t.Fatal(set, err, calls.Load())
	}
	deleted, err := api.DeleteVolumeImageMetadata(vsaContext(t), "id", volumes.WithVolumeImageMetadataDeleteKeys("", "a"))
	if err != nil || deleted == nil || !deleted.Completed || deleted.Observed != nil || deleted.Failed != nil || len(deleted.Deleted) != 2 || deleted.Deleted[0].Key != "" || deleted.Deleted[1].Key != "a" || calls.Load() != 5 {
		t.Fatal(deleted, err, calls.Load())
	}
	var missing *volumes.API
	absent, err := missing.SetVolumeImageMetadata(vsaContext(t), "id")
	vsaOperation(t, err, "SetVolumeImageMetadata")
	if absent != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 5 {
		t.Fatal(absent, err, calls.Load())
	}
	missingDelete, err := missing.DeleteVolumeImageMetadata(vsaContext(t), "id", volumes.WithVolumeImageMetadataDeleteKeys())
	vsaOperation(t, err, "DeleteVolumeImageMetadata")
	if missingDelete != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 5 {
		t.Fatal(missingDelete, err, calls.Load())
	}
	for _, kind := range []string{"unsafe ID", "source changes in original", "invalid active raw value"} {
		t.Run(kind, func(t *testing.T) {
			var originals, later atomic.Int32
			id := "id"
			if kind == "unsafe ID" {
				id = "id/other"
			}
			base := client.ResourceBase
			result, err := blockstorage.SetVolumeImageMetadata(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: id}, func(o *blockstorage.VolumeImageMetadataOpts) error {
				originals.Add(1)
				if kind == "source changes in original" {
					client.ResourceBase += "changed/"
				}
				if kind == "invalid active raw value" {
					o.Metadata = map[string]json.RawMessage{"bad": json.RawMessage(`null true`)}
				}
				return nil
			}, func(*blockstorage.VolumeImageMetadataOpts) error {
				later.Add(1)
				if kind == "source changes in original" {
					client.ResourceBase = base
				}
				return nil
			})
			client.ResourceBase = base
			vsaOperation(t, err, "SetVolumeImageMetadata")
			wantOriginals, wantLater := int32(1), int32(1)
			if kind == "unsafe ID" {
				wantOriginals, wantLater = 0, 0
			}
			if kind == "source changes in original" {
				wantLater = 0
			}
			var proof *resource.ResponseError
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || originals.Load() != wantOriginals || later.Load() != wantLater || calls.Load() != 5 {
				t.Fatal(result, err, proof, originals.Load(), later.Load(), calls.Load())
			}
		})
	}
}
