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

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/blockstorage/v3/volumes"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

const vuiBody = `{"os-volume_upload_image":{"force":false,"image_name":""}}`
const vuiReply = `{"os-volume_upload_image":{"image_id":"returned/unsafe","large":9007199254740993,"unknown":[null,false]}}`

func vuiPost(t *testing.T, req *http.Request, id, body, version, token string) {
	t.Helper()
	var actual []byte
	var err error
	if req.Body != nil {
		actual, err = io.ReadAll(req.Body)
	}
	current := ""
	if version != "" {
		current = "volume " + version
	}
	if err != nil || string(actual) != body || req.Method != http.MethodPost || req.URL.EscapedPath() != vsaBase+"volumes/"+url.PathEscape(id)+"/action" || req.URL.RawQuery != "" || req.Header.Get("X-Source") != "original" || req.Header.Get("X-Auth-Token") != token || req.Header.Get("OpenStack-API-Version") != current || req.Header.Get("X-OpenStack-Volume-API-Version") != version || req.ContentLength != int64(len(body)) || len(req.TransferEncoding) != 0 {
		t.Error("volume upload changed literal body, fixed route, framing or captured policy", req.Method, req.URL, string(actual), req.Header, req.ContentLength, req.TransferEncoding, err)
	}
}

func TestUploadVolumeToImageSerializesLiteralNamesDefaultsAndOptionalPresence(t *testing.T) {
	cases := []struct {
		name, image, body string
		required          bool
		options           []blockstorage.VolumeImageUploadOption
	}{
		{"required empty name and default false", "", vuiBody, false, nil},
		{"explicit force false", "", vuiBody, false, []blockstorage.VolumeImageUploadOption{blockstorage.WithVolumeImageUploadForce(false)}},
		{"literal control name and true force", " /?%#\n\x00한글", `{"os-volume_upload_image":{"force":true,"image_name":" /?%#\n\u0000한글"}}`, false, []blockstorage.VolumeImageUploadOption{blockstorage.WithVolumeImageUploadForce(true)}},
		{"empty formats present without required gate", "", `{"os-volume_upload_image":{"container_format":"","disk_format":"","force":false,"image_name":""}}`, false, []blockstorage.VolumeImageUploadOption{blockstorage.WithVolumeImageUploadDiskFormat(""), blockstorage.WithVolumeImageUploadContainerFormat("")}},
		{"unknown formats forwarded literal", "", `{"os-volume_upload_image":{"container_format":"container /?%#","disk_format":"server-format","force":false,"image_name":""}}`, false, []blockstorage.VolumeImageUploadOption{blockstorage.WithVolumeImageUploadDiskFormat("server-format"), blockstorage.WithVolumeImageUploadContainerFormat("container /?%#")}},
		{"explicit empty visibility requires gate", "", `{"os-volume_upload_image":{"force":false,"image_name":"","visibility":""}}`, true, []blockstorage.VolumeImageUploadOption{blockstorage.WithVolumeImageUploadVisibility("")}},
		{"explicit protected false requires gate", "", `{"os-volume_upload_image":{"force":false,"image_name":"","protected":false}}`, true, []blockstorage.VolumeImageUploadOption{blockstorage.WithVolumeImageUploadProtected(false)}},
		{"all optional values present including unknown visibility", "", `{"os-volume_upload_image":{"container_format":"bare","disk_format":"qcow2","force":true,"image_name":"","protected":true,"visibility":"server-defined"}}`, true, []blockstorage.VolumeImageUploadOption{blockstorage.WithVolumeImageUploadForce(true), blockstorage.WithVolumeImageUploadDiskFormat("qcow2"), blockstorage.WithVolumeImageUploadContainerFormat("bare"), blockstorage.WithVolumeImageUploadVisibility("server-defined"), blockstorage.WithVolumeImageUploadProtected(true)}},
		{"replacement clears required-gate fields and restores force default", "", vuiBody, false, []blockstorage.VolumeImageUploadOption{blockstorage.WithVolumeImageUploadVisibility("old"), blockstorage.WithVolumeImageUploadProtected(true), blockstorage.WithVolumeImageUploadForce(true), blockstorage.WithVolumeImageUploadOptions(blockstorage.VolumeImageUploadOpts{})}},
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.90")
			id := "한글-ID"
			var calls atomic.Int32
			code := []int{200, 203, 302, 399}[index%4]
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				n := calls.Add(1)
				if tc.required && n == 1 {
					vsaDiscoveryRequest(t, req, vsaVersionPath)
					w.Header().Set("X-Proof", "conditional support")
					testcloud.JSON(w, 300, `{"version":{"id":"v3.0","min_version":"3.0","max_version":"3.50"}}`)
					return
				}
				vuiPost(t, req, id, tc.body, "3.90", "test-token")
				w.Header().Set("X-Proof", "volume upload action")
				testcloud.JSON(w, code, vuiReply)
			})
			result, err := blockstorage.UploadVolumeToImage(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: id}, tc.image, tc.options...)
			wantCalls := int32(1)
			wantPages := 0
			if tc.required {
				wantCalls = 2
				wantPages = 1
			}
			if err != nil || result == nil || !result.Completed || result.VolumeID != id || result.Microversion != "3.90" || client.Microversion != "3.90" || len(result.Discovery) != wantPages || result.Applied == nil || result.Applied.StatusCode != code || result.Applied.Header.Get("X-Proof") != "volume upload action" || string(result.Applied.Body) != vuiReply || string(result.Upload) != `{"image_id":"returned/unsafe","large":9007199254740993,"unknown":[null,false]}` || calls.Load() != wantCalls {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestUploadVolumeToImageAcceptsArbitrarySelectedJSONAndKeepsFailedDecodeProof(t *testing.T) {
	cases := []struct {
		name, reply, value string
		code               int
		invalid            bool
	}{
		{"null is successful and nonnil", `{"os-volume_upload_image":null}`, `null`, 200, false},
		{"false scalar", `{"os-volume_upload_image":false}`, `false`, 203, false},
		{"large integer", `{"os-volume_upload_image":9007199254740993}`, `9007199254740993`, 302, false},
		{"string scalar", `{"os-volume_upload_image":"no image ID"}`, `"no image ID"`, 399, false},
		{"array value", `{"os-volume_upload_image":[null,false,{"large":9007199254740993}]}`, `[null,false,{"large":9007199254740993}]`, 200, false},
		{"object without image ID", `{"os-volume_upload_image":{"a":"b","unknown":[]}}`, `{"a":"b","unknown":[]}`, 203, false},
		{"selected interior precision and escapes", `{"os-volume_upload_image": { "decimal":1.00000000000000001,"escaped":"\u0061" } }`, `{ "decimal":1.00000000000000001,"escaped":"\u0061" }`, 200, false},
		{"last outer duplicate wins", `{"os-volume_upload_image":null,"os-volume_upload_image":{"second":true}}`, `{"second":true}`, 203, false},
		{"missing exact key", `{"other":{"image_id":"unused"}}`, "", 200, true},
		{"wrong case key", `{"OS-VOLUME_UPLOAD_IMAGE":{}}`, "", 203, true},
		{"array root", `[]`, "", 200, true},
		{"null root", `null`, "", 203, true},
		{"scalar root", `false`, "", 399, true},
		{"malformed JSON", `{"os-volume_upload_image":`, "", 200, true},
		{"trailing second value", `{"os-volume_upload_image":{}} null`, "", 203, true},
		{"invalid UTF8", string([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}), "", 200, true},
		{"empty admitted204", ``, "", 204, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.60")
			var calls, retries atomic.Int32
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				retries.Add(1)
				return original
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				vuiPost(t, req, "id", vuiBody, "3.60", "test-token")
				w.Header().Set("X-Proof", "actual upload response")
				testcloud.JSON(w, tc.code, tc.reply)
			})
			result, err := blockstorage.UploadVolumeToImage(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, "")
			if result == nil || result.VolumeID != "id" || result.Microversion != "3.60" || len(result.Discovery) != 0 || result.Applied == nil || result.Applied.StatusCode != tc.code || string(result.Applied.Body) != tc.reply || result.Applied.Header.Get("X-Proof") != "actual upload response" || calls.Load() != 1 || retries.Load() != 0 {
				t.Fatal(result, err, calls.Load(), retries.Load())
			}
			if tc.invalid {
				vsaOperation(t, err, "UploadVolumeToImage")
				var proof *resource.ResponseError
				if result.Completed || result.Upload != nil || !errors.As(err, &proof) || proof.StatusCode != tc.code || string(proof.Body) != tc.reply || proof.Header.Get("X-Proof") != "actual upload response" {
					t.Fatal(result, err, proof)
				}
				proof.Header.Set("X-Proof", "changed error")
				if result.Applied.Header.Get("X-Proof") != "actual upload response" {
					t.Fatal("decode error proof aliases result")
				}
				if len(proof.Body) > 0 {
					proof.Body[0] = '!'
					if string(result.Applied.Body) != tc.reply {
						t.Fatal("decode body aliases result")
					}
				}
				return
			}
			if err != nil || !result.Completed || result.Upload == nil || string(result.Upload) != tc.value {
				t.Fatal(result, err)
			}
			result.Applied.Body[0] = '!'
			result.Applied.Header.Set("X-Proof", "changed physical response")
			if string(result.Upload) != tc.value {
				t.Fatal("selected value aliases physical response")
			}
			result.Upload[0] = '?'
			if result.Applied.Body[0] != '!' {
				t.Fatal("physical response aliases selected value")
			}
		})
	}
}

func TestUploadVolumeToImageConditional31GateUsesAdvertisedBoundsAndOrdinaryActionSelection(t *testing.T) {
	cases := []struct {
		name, selected, min, max, want, option string
		unsupported                            bool
	}{
		{"plain selected3.0 has no gate", "3.0", "", "", "3.0", "none", false},
		{"empty visibility selected exceeds maximum", "3.90", "3.0", "3.50", "3.90", "visibility", false},
		{"protected false selected below3.1", "3.0", "3.0", "3.90", "", "protected", true},
		{"fixed required3.1 below server minimum", "3.90", "3.2", "3.99", "", "protected", true},
		{"server maximum below3.1", "3.90", "3.0", "3.0", "", "visibility", true},
		{"minimum missing", "3.90", "", "3.90", "", "protected", true},
		{"maximum missing", "3.90", "3.0", "", "", "protected", true},
		{"wrong selected major", "2.90", "3.0", "3.90", "", "protected", true},
		{"global latest selected unsupported", "latest", "3.0", "3.90", "", "visibility", true},
		{"finite major latest selected retained", "3.latest", "3.0", "3.90", "3.latest", "visibility", false},
		{"unselected shared probe and3.71 cap", "", "3.0", "3.99", "3.71", "protected", false},
		{"unselected lower server maximum", "", "3.0", "3.20", "3.20", "protected", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, tc.selected)
			var calls atomic.Int32
			support := `{"version":{"id":"v3.0","min_version":"` + tc.min + `","max_version":"` + tc.max + `"}}`
			body := vuiBody
			var options []blockstorage.VolumeImageUploadOption
			if tc.option == "protected" {
				body = `{"os-volume_upload_image":{"force":false,"image_name":"","protected":false}}`
				options = []blockstorage.VolumeImageUploadOption{blockstorage.WithVolumeImageUploadProtected(false)}
			}
			if tc.option == "visibility" {
				body = `{"os-volume_upload_image":{"force":false,"image_name":"","visibility":""}}`
				options = []blockstorage.VolumeImageUploadOption{blockstorage.WithVolumeImageUploadVisibility("")}
			}
			if tc.selected != "" {
				client.MoreHeaders["OpenStack-API-Version"] = "volume " + tc.selected
				client.MoreHeaders["X-OpenStack-Volume-API-Version"] = tc.selected
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
				n := calls.Add(1)
				if tc.option != "none" && n == 1 {
					vsaDiscoveryRequest(t, req, vsaVersionPath)
					w.Header().Set("X-Proof", "required3.1 advertisement")
					testcloud.JSON(w, 300, support)
					return
				}
				if tc.unsupported {
					t.Error("unsupported gate sent action", req.URL)
				}
				vuiPost(t, req, "id", body, tc.want, "test-token")
				w.Header().Set("X-Proof", "after conditional gate")
				testcloud.JSON(w, 200, `{"os-volume_upload_image":[]}`)
			})
			result, err := blockstorage.UploadVolumeToImage(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, "", options...)
			if result == nil || result.VolumeID != "id" || client.Microversion != tc.selected {
				t.Fatal(result, err, client.Microversion)
			}
			if tc.unsupported {
				vsaOperation(t, err, "UploadVolumeToImage")
				var proof *resource.ResponseError
				if !errors.Is(err, resource.ErrUnsupported) || !errors.As(err, &proof) || proof.StatusCode != 300 || string(proof.Body) != support || result.Completed || result.Applied != nil || result.Upload != nil || len(result.Discovery) != 1 || string(result.Discovery[0].Body) != support || calls.Load() != 1 {
					t.Fatal(result, err, proof, calls.Load())
				}
				proof.Body[0] = '!'
				if string(result.Discovery[0].Body) != support {
					t.Fatal("required failure aliases discovery observation")
				}
				return
			}
			wantPages, wantCalls := 0, int32(1)
			if tc.option != "none" {
				wantPages = 1
				wantCalls = 2
			}
			if err != nil || !result.Completed || result.Microversion != tc.want || len(result.Discovery) != wantPages || result.Applied == nil || result.Applied.StatusCode != 200 || string(result.Upload) != `[]` || calls.Load() != wantCalls {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestUploadVolumeToImageDirectPreflightAndOwnedOriginalsKeepFixedRequestPolicy(t *testing.T) {
	for _, kind := range []string{"nil context", "canceled context", "nil client", "wrong service", "unsafe volume ID", "invalid image name", "invalid final visibility", "source changed by original"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vsaClient(cloud, "3.60")
			ctx := vsaContext(t)
			id, name := "id", ""
			cause := errors.New("volume image upload cancellation")
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
			case "unsafe volume ID":
				id = "id/other"
			case "invalid image name":
				name = string([]byte{0xff})
			}
			var calls, originals, later atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) { calls.Add(1); w.WriteHeader(500) })
			result, err := blockstorage.UploadVolumeToImage(ctx, client, blockstorage.VolumeActionRequest{VolumeID: id}, name, func(o *blockstorage.VolumeImageUploadOpts) error {
				originals.Add(1)
				if kind == "invalid final visibility" {
					bad := string([]byte{0xff})
					o.Visibility = &bad
				}
				if kind == "source changed by original" {
					client.ResourceBase += "changed/"
				}
				return nil
			}, func(*blockstorage.VolumeImageUploadOpts) error {
				later.Add(1)
				if kind == "source changed by original" {
					client.ResourceBase = cloud.Server.URL + vsaBase
				}
				return nil
			})
			vsaOperation(t, err, "UploadVolumeToImage")
			var proof *resource.ResponseError
			wantOriginals, wantLater := int32(0), int32(0)
			if kind == "invalid final visibility" {
				wantOriginals, wantLater = 1, 1
			}
			if kind == "source changed by original" {
				wantOriginals = 1
			}
			if result != nil || !errors.Is(err, want) || errors.As(err, &proof) || calls.Load() != 0 || originals.Load() != wantOriginals || later.Load() != wantLater || kind == "canceled context" && !errors.Is(err, cause) {
				t.Fatal(result, err, proof, calls.Load(), originals.Load(), later.Load())
			}
		})
	}
	t.Run("factory pointers and captured callback slice stay owned", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := vsaClient(cloud, "3.60")
		disk := "server-format"
		force := false
		factory := blockstorage.WithVolumeImageUploadOptions(blockstorage.VolumeImageUploadOpts{DiskFormat: &disk, Force: &force})
		disk = "mutated factory input"
		force = true
		var retained *blockstorage.VolumeImageUploadOpts
		var originals, later, replaced, calls atomic.Int32
		var options []blockstorage.VolumeImageUploadOption
		options = []blockstorage.VolumeImageUploadOption{factory, func(o *blockstorage.VolumeImageUploadOpts) error {
			originals.Add(1)
			retained = o
			options[2] = func(*blockstorage.VolumeImageUploadOpts) error {
				replaced.Add(1)
				return errors.New("replaced original must not execute")
			}
			return nil
		}, func(*blockstorage.VolumeImageUploadOpts) error {
			later.Add(1)
			*retained.DiskFormat = "mutated retained policy"
			*retained.Force = true
			client.MoreHeaders["x-source"] = "later ordinary header"
			return nil
		}}
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			calls.Add(1)
			vuiPost(t, req, "id", `{"os-volume_upload_image":{"disk_format":"server-format","force":false,"image_name":""}}`, "3.60", "test-token")
			testcloud.JSON(w, 203, `{"os-volume_upload_image":null}`)
		})
		result, err := blockstorage.UploadVolumeToImage(vsaContext(t), client, blockstorage.VolumeActionRequest{VolumeID: "id"}, "", options...)
		if err != nil || result == nil || !result.Completed || string(result.Upload) != `null` || calls.Load() != 1 || originals.Load() != 1 || later.Load() != 1 || replaced.Load() != 0 {
			t.Fatal(result, err, calls.Load(), originals.Load(), later.Load(), replaced.Load())
		}
	})
}

func TestUploadVolumeToImageServiceKeepsNativeUploadImageOmission202AndTypedResult(t *testing.T) {
	cloud := testcloud.New(t)
	client := vsaClient(cloud, "3.60")
	api := volumes.New(client)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		switch n := calls.Add(1); n {
		case 1:
			vuiPost(t, req, "id", `{"os-volume_upload_image":{}}`, "3.60", "test-token")
			testcloud.JSON(w, 202, `{"os-volume_upload_image":{"image_id":"native-image","image_name":"native","unknown":"not in typed model"}}`)
		case 2:
			vuiPost(t, req, "id", `{"os-volume_upload_image":{}}`, "3.60", "test-token")
			w.Header().Set("X-Proof", "native202 only")
			testcloud.JSON(w, 200, `{"os-volume_upload_image":{"image_id":"must reject"}}`)
		case 3:
			vuiPost(t, req, "id", vuiBody, "3.60", "test-token")
			testcloud.JSON(w, 200, `{"os-volume_upload_image":{"a":"b"}}`)
		case 4:
			vsaDiscoveryRequest(t, req, vsaVersionPath)
			w.Header().Set("X-Proof", "service explicit false gate")
			testcloud.JSON(w, 300, `{"version":{"id":"v3.0","min_version":"3.0","max_version":"3.90"}}`)
		case 5:
			vuiPost(t, req, "id", `{"os-volume_upload_image":{"force":false,"image_name":"","protected":false}}`, "3.60", "test-token")
			testcloud.JSON(w, 203, `{"os-volume_upload_image":["arbitrary"]}`)
		default:
			t.Error("unexpected native/new service upload request", n, req.URL)
			w.WriteHeader(500)
		}
	})
	nativeValue, err := api.UploadImage(vsaContext(t), "id", volumes.UploadImageOpts{})
	if err != nil || nativeValue.ImageID != "native-image" || nativeValue.ImageName != "native" || calls.Load() != 1 {
		t.Fatal(nativeValue, err, calls.Load())
	}
	encoded, _ := json.Marshal(nativeValue)
	if bytes.Contains(encoded, []byte("not in typed model")) {
		t.Fatal("typed native extraction gained unknown field", string(encoded))
	}
	_, err = api.UploadImage(vsaContext(t), "id", volumes.UploadImageOpts{})
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != 200 || len(native.Expected) != 1 || native.Expected[0] != 202 || native.ResponseHeader.Get("X-Proof") != "native202 only" || calls.Load() != 2 {
		t.Fatal("new API broadened native UploadImage", err, native, calls.Load())
	}
	arbitrary, err := api.UploadVolumeToImage(vsaContext(t), "id", "")
	if err != nil || arbitrary == nil || !arbitrary.Completed || string(arbitrary.Upload) != `{"a":"b"}` || len(arbitrary.Discovery) != 0 || calls.Load() != 3 {
		t.Fatal(arbitrary, err, calls.Load())
	}
	conditional, err := api.UploadVolumeToImage(vsaContext(t), "id", "", volumes.WithVolumeImageUploadProtected(false))
	if err != nil || conditional == nil || !conditional.Completed || string(conditional.Upload) != `["arbitrary"]` || len(conditional.Discovery) != 1 || conditional.Discovery[0].Header.Get("X-Proof") != "service explicit false gate" || calls.Load() != 5 {
		t.Fatal(conditional, err, calls.Load())
	}
	var absent *volumes.API
	missing, err := absent.UploadVolumeToImage(vsaContext(t), "id", "")
	vsaOperation(t, err, "UploadVolumeToImage")
	if missing != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 5 {
		t.Fatal(missing, err, calls.Load())
	}
}
