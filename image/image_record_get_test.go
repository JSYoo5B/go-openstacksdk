package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

// Independent inventory from pinned Image, inherited Resource and TagMixin;
// expected projection does not depend on the implementation's descriptor table.
const imageRecordViewKeys = `checksum container_format created_at disk_format is_hidden is_protected hash_algo hash_value min_disk min_ram name owner owner_id properties size store status updated_at virtual_size visibility file locations direct_url url metadata architecture hypervisor_type instance_type_rxtx_factor instance_uuid needs_config_drive kernel_id os_distro os_version needs_secure_boot os_shutdown_timeout ramdisk_id vm_mode hw_cpu_sockets hw_cpu_cores hw_cpu_threads hw_disk_bus hw_cpu_policy hw_cpu_thread_policy hw_rng_model hw_machine_type hw_scsi_model hw_serial_port_count hw_video_model hw_video_ram hw_watchdog_action os_command_line hw_vif_model is_hw_vif_multiqueue_enabled is_hw_boot_menu_enabled vmware_adaptertype vmware_ostype has_auto_disk_config os_type os_admin_user hw_qemu_guest_agent os_require_quiesce schema id tags location`

func imageRecordValues(value *resource.RawResource) map[string]string {
	result := make(map[string]string)
	if value != nil {
		for key, raw := range value.Body {
			result[key] = string(raw)
		}
	}
	return result
}
func imageRecordDefaults(properties string) map[string]string {
	result := make(map[string]string)
	for _, key := range strings.Fields(imageRecordViewKeys) {
		result[key] = "null"
	}
	result["tags"], result["properties"] = "[]", properties
	return result
}
func imageRecordProperties(t *testing.T, value *ImageRecord) map[string]json.RawMessage {
	t.Helper()
	var result map[string]json.RawMessage
	if err := json.Unmarshal(value.Resource.Body["properties"], &result); err != nil {
		t.Fatal(err)
	}
	return result
}

// Reuse taskCore's physical IO fixture; only a missing Close callback is added.
type imageRecordCloseBody struct {
	*taskCoreBody
	after func()
}

func (body *imageRecordCloseBody) Close() error {
	err := body.taskCoreBody.Close()
	if body.after != nil {
		body.after()
	}
	return err
}

func TestImageRecordGetProjectionAndIndependentReceipts(t *testing.T) {
	const raw = `{"id":900719925474099312345,"name":["passive",null],"owner":"wire owner","created_at":{"literal":true},"min_disk":false,"virtual_size":1e400,"locations":false,"metadata":{"exact":1.00000000000000000001},"self":"https://foreign.test/image","schema":false,"location":{"cloud":"wire"},"vendor":1e400,"OpenStack-image-import-methods":"body decoy"}`
	calls, locations := 0, 0
	cloud := "captured"
	var actualHeader http.Header
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape("a /한:%?\\b") || req.URL.RawQuery != "" || req.Body != nil || req.Header.Get("Accept") != "application/json" {
			t.Fatal(req.Method, req.URL, req.Header)
		}
		reply := taskCoreJSON(req, 203, raw)
		reply.Header[http.CanonicalHeaderKey("OpenStack-image-import-methods")] = []string{"a,, a", "b, b"}
		actualHeader = reply.Header
		return reply, nil
	})
	service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
		locations++
		return resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"token project"`)}}, nil
	}})
	got, err := service.GetImageRecord(context.Background(), ImageRecordRequest{ID: "a /한:%?\\b"})
	if err != nil || got == nil || calls != 1 || locations != 1 || len(got.Resource.Body) != 65 || got.Wire == nil {
		t.Fatal(got, err, calls, locations)
	}
	want := imageRecordDefaults(`{}`)
	for key, value := range map[string]string{"id": "900719925474099312345", "name": `["passive",null]`, "owner": `"wire owner"`, "owner_id": `"wire owner"`, "created_at": `{"literal":true}`, "min_disk": "false", "virtual_size": "1e400", "locations": "false", "metadata": `{"exact":1.00000000000000000001}`, "schema": "false"} {
		want[key] = value
	}
	// Compare all declared fields separately from current location and packed extras.
	want["location"] = string(got.Resource.Body["location"])
	want["properties"] = string(got.Resource.Body["properties"])
	th.CheckDeepEquals(t, want, imageRecordValues(got.Resource))
	var captured resource.CloudLocation
	if err := json.Unmarshal(got.Resource.Body["location"], &captured); err != nil || captured.Cloud == nil || *captured.Cloud != "captured" || string(captured.Project.ID) != `"token project"` {
		t.Fatal(captured, err)
	}
	props := imageRecordProperties(t, got)
	th.CheckDeepEquals(t, map[string]string{"location": `{"cloud":"wire"}`, "vendor": "1e400", "OpenStack-image-import-methods": `"body decoy"`}, func() map[string]string {
		m := map[string]string{}
		for k, v := range props {
			m[k] = string(v)
		}
		return m
	}())
	th.CheckDeepEquals(t, []string{"a", "", " a", " b", " b"}, got.ImportMethods)
	for _, view := range []*resource.RawResource{got.Resource, got.Wire} {
		if view.StatusCode != 203 || view.Header.Get("X-Task-Proof") != "actual" {
			t.Fatal(view)
		}
	}
	if got.StatusCode != 203 || string(got.Envelope) != raw || got.Header.Get("X-Task-Proof") != "actual" || string(got.Wire.Body["self"]) != `"https://foreign.test/image"` || string(got.Wire.Body["location"]) != `{"cloud":"wire"}` {
		t.Fatal(got)
	}
	got.Resource.Body["metadata"][0] = '!'
	got.Resource.Body["id"][0] = '0'
	got.Resource.Header.Set("X-Task-Proof", "view changed")
	got.Wire.Header.Set("X-Task-Proof", "wire changed")
	got.Header.Set("X-Task-Proof", "record changed")
	got.ImportMethods[0] = "changed"
	if string(got.Wire.Body["metadata"]) != `{"exact":1.00000000000000000001}` || string(got.Wire.Body["id"]) != "900719925474099312345" || string(got.Envelope) != raw || actualHeader.Get("X-Task-Proof") != "actual" {
		t.Fatal("owned receipt channels alias", got)
	}
	got.Wire.Body["vendor"][0] = '!'
	th.AssertEquals(t, "1e400", string(imageRecordProperties(t, got)["vendor"]))
	got.Envelope[0] = '!'
	if actualHeader.Get("X-Task-Proof") != "actual" {
		t.Fatal("actual response header mutated")
	}
}

func TestImageRecordGetPackingAliasesAndSeedOverlay(t *testing.T) {
	for _, test := range []struct{ name, raw, id, owner, properties string }{
		{"empty body clears seeded properties", `{}`, `"seed"`, "null", `{}`},
		{"invalid syntax retains seed properties", `not JSON`, `"seed"`, "null", `{"seeded":true}`},
		{"present null id", `{"id":null}`, "null", "null", `{}`},
		{"name does not replace id", `{"name":"response name"}`, `"seed"`, "null", `{}`},
		{"owner alias later", `{"owner":"first","owner_id":"last"}`, `"seed"`, `"last"`, `{}`},
		{"wire owner later", `{"owner_id":"first","owner":"last"}`, `"seed"`, `"last"`, `{}`},
		{"duplicate first position", `{"owner":"first","owner_id":"middle","owner":"last"}`, `"seed"`, `"middle"`, `{}`},
		{"null properties wrapped on fetch", `{"properties":null}`, `"seed"`, "null", `{"properties":null}`},
		{"false properties wrapped on fetch", `{"properties":false}`, `"seed"`, "null", `{"properties":false}`},
		{"dictionary property bytes", `{"properties":{"n":1.00000000000000000001}}`, `"seed"`, "null", `{"n":1.00000000000000000001}`},
		{"unknown overlays named properties", `{"properties":{"vendor":"old","keep":null},"vendor":false,"self":"passive"}`, `"seed"`, "null", `{"keep":null,"vendor":false}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			seed := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"seed"`), "checksum": json.RawMessage(`"seed checksum"`), "properties": json.RawMessage(`{"seeded":true}`)}, Header: http.Header{"X-Task-Proof": {"seed header"}}, StatusCode: 500}}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreJSON(req, 200, test.raw), nil })
			got, err := New(client).GetImageRecord(context.Background(), ImageRecordRequest{Resource: seed})
			if got == nil || err != nil {
				t.Fatal(got, err)
			}
			th.AssertEquals(t, test.id, string(got.Resource.Body["id"]))
			th.AssertEquals(t, test.owner, string(got.Resource.Body["owner"]))
			th.AssertEquals(t, test.owner, string(got.Resource.Body["owner_id"]))
			th.AssertEquals(t, `"seed checksum"`, string(got.Resource.Body["checksum"]))
			th.AssertEquals(t, test.properties, string(got.Resource.Body["properties"]))
			if got.Resource.Header.Get("X-Task-Proof") != "actual" || got.Resource.StatusCode != 200 || seed.Header.Get("X-Task-Proof") != "seed header" || seed.StatusCode != 500 {
				t.Fatal("seed supplied receipt metadata", got, seed)
			}
		})
	}
	for _, response := range []string{"opaque accepted body", `{"broken":`} {
		t.Run("seed property layers/"+response, func(t *testing.T) {
			seed := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"selected"`), "x": json.RawMessage(`"resource x"`), "resource_only": json.RawMessage(`900719925474099312345`)}}}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreJSON(req, 200, response), nil })
			got, err := New(client).GetImageRecord(context.Background(), ImageRecordRequest{Resource: seed, Attributes: map[string]any{"y": "request y"}}, WithImageRecordAttributes(map[string]any{"z": "option z", "x": "option override"}))
			if got == nil || err != nil || got.Wire != nil {
				t.Fatal(got, err)
			}
			props := imageRecordProperties(t, got)
			want := map[string]json.RawMessage{"x": json.RawMessage(`"option override"`), "resource_only": json.RawMessage(`900719925474099312345`), "y": json.RawMessage(`"request y"`), "z": json.RawMessage(`"option z"`)}
			th.CheckDeepEquals(t, want, props)
			th.AssertEquals(t, `"resource x"`, string(seed.Body["x"]))
		})
	}
	t.Run("later options replace resource identity and attributes", func(t *testing.T) {
		seed := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"before"`), "name": json.RawMessage(`"before name"`)}}}
		calls := 0
		service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.URL.EscapedPath() != "/reverse/glance/v2/images/after%2Fid" {
				t.Fatal(req.URL)
			}
			return taskCoreJSON(req, 200, `{}`), nil
		}))
		got, err := service.GetImageRecord(context.Background(), ImageRecordRequest{Resource: seed}, WithImageRecordAttribute("id", "after/id"), WithImageRecordAttributes(map[string]any{"name": "after name"}))
		if got == nil || err != nil || calls != 1 || string(got.Resource.Body["id"]) != `"after/id"` || string(got.Resource.Body["name"]) != `"after name"` || string(seed.Body["id"]) != `"before"` {
			t.Fatal(got, err, calls, seed)
		}
	})
}

func TestImageRecordGetDescriptorIntegration(t *testing.T) {
	// Exhaustive Unicode, IEEE and Python repr matrices belong to the reused
	// jsonfilter/cloudfilter tests; these verify Image dispatch and raw isolation.
	for _, test := range []struct {
		field, raw, want string
		bad              bool
	}{
		{"size", "true", "true", false}, {"size", "3.9", "3", false}, {"size", `"１２"`, "12", false}, {"size", `"invalid"`, "0", false},
		{"instance_type_rxtx_factor", "true", "1", false}, {"instance_type_rxtx_factor", `"1.25"`, "1.25", false}, {"instance_type_rxtx_factor", "{}", "0", false},
		{"metadata", "false", "{}", false}, {"metadata", "null", "null", false},
		{"os_hidden", `"false"`, "true", false}, {"protected", "[]", "false", false}, {"os_require_quiesce", "[0]", "true", false},
		{"tags", `"tag"`, `["tag"]`, false}, {"tags", "null", "null", false}, {"tags", `[null,false,{"big":900719925474099312345}]`, `[null,false,{"big":900719925474099312345}]`, false},
		{"hw_vif_multiqueue_enabled", `"FaLsE"`, "false", false}, {"hw_vif_multiqueue_enabled", "true", "true", false}, {"hw_vif_multiqueue_enabled", "null", "null", false},
		{"hw_qemu_guest_agent", "true", `"True"`, false}, {"hw_qemu_guest_agent", `[null,false]`, `"[None, False]"`, false},
		{"instance_type_rxtx_factor", `"not float"`, "", true}, {"instance_type_rxtx_factor", `"NaN"`, "", true}, {"instance_type_rxtx_factor", "1e400", "", true},
		{"hw_vif_multiqueue_enabled", `" false "`, "", true}, {"hw_vif_multiqueue_enabled", "1", "", true},
	} {
		t.Run(test.field+"/"+test.raw, func(t *testing.T) {
			raw := fmt.Sprintf("{%q:%s}", test.field, test.raw)
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreJSON(req, 200, raw), nil })
			got, err := New(client).GetImageRecord(context.Background(), ImageRecordRequest{ID: "selected"})
			if test.bad {
				if got != nil || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(got, err)
				}
				taskCoreProof(t, err, 200, raw)
				return
			}
			if got == nil || err != nil {
				t.Fatal(got, err)
			}
			key := test.field
			switch key {
			case "os_hidden":
				key = "is_hidden"
			case "protected":
				key = "is_protected"
			case "hw_vif_multiqueue_enabled":
				key = "is_hw_vif_multiqueue_enabled"
			}
			th.AssertEquals(t, test.want, string(got.Resource.Body[key]))
			th.AssertEquals(t, test.raw, string(got.Wire.Body[test.field]))
		})
	}
}

func TestImageRecordGetAcceptedToleranceAndNativeErrors(t *testing.T) {
	for _, test := range []struct {
		code         int
		raw          string
		good, parsed bool
	}{
		{200, `{}`, true, true}, {201, `{"id":"different"}`, true, true}, {299, `{}`, true, true}, {300, `{}`, true, true}, {399, `{}`, true, true},
		{204, "", true, false}, {202, " \n\t ", true, false}, {304, "not JSON", true, false}, {200, `{"broken":`, true, false},
		{200, `null`, false, true}, {200, `[]`, false, true}, {200, `false`, false, true}, {200, `42`, false, true}, {200, `"text"`, false, true}, {200, "{\"extra\":\"\xff\"}", false, true},
	} {
		t.Run(fmt.Sprintf("%d/%q", test.code, test.raw), func(t *testing.T) {
			calls, retries := 0, 0
			body := &taskCoreBody{reader: strings.NewReader(test.raw)}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				response := taskCoreHTTP(req, test.code, body)
				response.Header.Set("Content-Type", "text/plain")
				return response, nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			got, err := New(client).GetImageRecord(context.Background(), ImageRecordRequest{ID: "selected"})
			if test.good {
				if got == nil || err != nil || (got.Wire != nil) != test.parsed || got.StatusCode != test.code || string(got.Envelope) != test.raw || got.ImportMethods == nil || len(got.ImportMethods) != 0 {
					t.Fatal(got, err)
				}
				want := imageRecordDefaults("null")
				if test.parsed {
					want["properties"] = "{}"
				}
				want["id"] = `"selected"`
				if test.raw == `{"id":"different"}` {
					want["id"] = `"different"`
				}
				th.CheckDeepEquals(t, want, imageRecordValues(got.Resource))
			} else {
				if got != nil || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(got, err)
				}
				taskCoreProof(t, err, test.code, test.raw)
			}
			if calls != 1 || retries != 0 || body.closes != 1 {
				t.Fatal("accepted body replay", calls, retries, body.closes)
			}
		})
	}
	for _, code := range []int{400, 403, 404, 500} {
		t.Run(fmt.Sprintf("native/%d", code), func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return taskCoreJSON(req, code, "native rejected image"), nil
			})
			got, err := New(client).GetImageRecord(context.Background(), ImageRecordRequest{ID: "selected"})
			var native gophercloud.ErrUnexpectedResponseCode
			if got != nil || !errors.As(err, &native) || native.Actual != code || native.Method != http.MethodGet || native.URL != "https://glance.example/reverse/glance/v2/images/selected" || string(native.Body) != "native rejected image" || calls != 1 {
				t.Fatal(got, err, native, calls)
			}
		})
	}
}

func TestImageRecordGetLocationAndConcreteSnapshots(t *testing.T) {
	cloud := "captured"
	facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"token project"`)}}
	attributes := map[string]any{"name": "factory name", "owner": "image owner"}
	headers := map[string]string{"X-Option": "factory header"}
	option := WithImageRecordOpts(ImageRecordOpts{Headers: headers, Attributes: attributes})
	attributes["name"] = "caller changed"
	headers["X-Option"] = "caller changed"
	seed := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"seed"`), "checksum": json.RawMessage(`"seed checksum"`)}}}
	calls, callbacks, locations := 0, 0, 0
	var retained *ImageRecordOpts
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Source") != "captured" || req.Header.Get("X-Option") != "factory header" || req.Header.Get("X-Final") != "yes" || req.Header.Get("X-Auth-Token") != "live" || req.Header.Get("OpenStack-API-Version") != "image 2.10" || !strings.HasSuffix(req.URL.Path, "/seed") {
			t.Fatal(req.URL, req.Header)
		}
		retained.Attributes["name"] = "retained changed"
		retained.Headers["X-Option"] = "retained changed"
		return taskCoreJSON(req, 200, `{}`), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "captured"}
	client.Microversion = "2.10"
	service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
		locations++
		client.MoreHeaders["X-Source"] = "getter changed"
		seed.Body["id"][1] = 'X'
		seed.Body["checksum"][1] = 'X'
		return facts, nil
	}})
	got, err := service.GetImageRecord(context.Background(), ImageRecordRequest{Resource: seed}, option, func(value *ImageRecordOpts) error {
		callbacks++
		retained = value
		cloud = "after option"
		facts.Project.ID[1] = 'X'
		client.SetToken("live")
		return WithImageRecordHeader("X-Final", "yes")(value)
	})
	if got == nil || err != nil || calls != 1 || callbacks != 1 || locations != 1 || string(got.Resource.Body["name"]) != `"factory name"` || string(got.Resource.Body["checksum"]) != `"seed checksum"` {
		t.Fatal(got, err, calls, callbacks, locations)
	}
	var location resource.CloudLocation
	if err := json.Unmarshal(got.Resource.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != "captured" || string(location.Project.ID) != `"token project"` {
		t.Fatal(location, err)
	}
	th.AssertEquals(t, `"image owner"`, string(got.Resource.Body["owner_id"]))
}

func TestImageRecordGetReadCloseAndStickyGuardEvidence(t *testing.T) {
	const raw = `{"id":"actual"}`
	for _, mode := range []string{"read", "close", "cancel", "source drift", "read drift restored on Close", "outer drift restored on Close"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New("owned image body failure")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			calls, retries := 0, 0
			var client *gophercloud.ServiceClient
			body := &taskCoreBody{reader: strings.NewReader(raw)}
			outerInvalid := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerInvalid {
					return cause
				}
				return nil
			})
			action := func() {}
			switch mode {
			case "read":
				body.reader = &taskCoreReader{body: raw, err: cause}
			case "close":
				body.closeErr = errors.Join(cause, gophercloud.ErrUnexpectedResponseCode{Actual: 404})
			case "cancel":
				action = func() { cancel(cause) }
			case "source drift", "read drift restored on Close":
				action = func() { client.Endpoint = "https://foreign.test/" }
			case "outer drift restored on Close":
				action = func() { outerInvalid = true }
			}
			if mode != "read" {
				body.reader = &taskCoreReader{body: raw, err: io.EOF, action: action}
			}
			selected := io.ReadCloser(body)
			if strings.Contains(mode, "restored") {
				selected = &imageRecordCloseBody{taskCoreBody: body, after: func() { client.Endpoint = "https://glance.example/reverse/glance/v2/"; outerInvalid = false }}
			}
			client = taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, 201, selected), nil })
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			got, err := New(client).GetImageRecord(ctx, ImageRecordRequest{ID: "selected"})
			if got != nil || err == nil || calls != 1 || retries != 0 || body.closes != 1 {
				t.Fatal(got, err, calls, retries, body.closes)
			}
			if strings.Contains(mode, "source") || mode == "read drift restored on Close" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, cause) {
				t.Fatal("IO/ancestor cause lost", err)
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if errors.Is(err, resource.ErrNotFound) {
				t.Fatal("nested404 interpreted as absence", err)
			}
			taskCoreProof(t, err, 201, raw)
		})
	}
}

func TestImageRecordGetCompletePreflightAndLegacyBoundary(t *testing.T) {
	for _, test := range []struct {
		name    string
		req     ImageRecordRequest
		options []ImageRecordOption
	}{
		{"missing identity", ImageRecordRequest{}, nil}, {"only name", ImageRecordRequest{Attributes: map[string]any{"name": "name"}}, nil},
		{"ambiguous sources", ImageRecordRequest{ID: "selected", Resource: &resource.RawResource{}}, nil},
		{"literal id attr collision", ImageRecordRequest{ID: "selected", Attributes: map[string]any{"id": "selected"}}, nil},
		{"literal option id null collision", ImageRecordRequest{ID: "selected"}, []ImageRecordOption{WithImageRecordAttribute("id", nil)}},
		{"blank identity", ImageRecordRequest{ID: " "}, nil}, {"dot identity", ImageRecordRequest{ID: "."}, nil}, {"dotdot identity", ImageRecordRequest{ID: ".."}, nil}, {"control identity", ImageRecordRequest{ID: "bad\n"}, nil}, {"invalid UTF8", ImageRecordRequest{ID: string([]byte{0xff})}, nil},
		{"untyped id cannot route", ImageRecordRequest{Attributes: map[string]any{"id": false}}, nil},
		{"nil option", ImageRecordRequest{ID: "selected"}, []ImageRecordOption{nil}},
		{"owned token", ImageRecordRequest{ID: "selected"}, []ImageRecordOption{WithImageRecordHeader("X-Auth-Token", "foreign")}},
		{"header newline", ImageRecordRequest{ID: "selected"}, []ImageRecordOption{WithImageRecordHeader("X-Extra", "\n")}},
		{"unsupported attribute", ImageRecordRequest{ID: "selected"}, []ImageRecordOption{WithImageRecordAttribute("vendor", make(chan int))}},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, 200, `{}`), nil }))
			got, err := service.GetImageRecord(context.Background(), test.req, test.options...)
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(got, err, calls)
			}
		})
	}
	for _, ctx := range []context.Context{nil, func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }()} {
		calls, callbacks := 0, 0
		got, err := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil })).GetImageRecord(ctx, ImageRecordRequest{ID: "selected"}, func(*ImageRecordOpts) error { callbacks++; return nil })
		if got != nil || err == nil || calls != 0 || callbacks != 0 {
			t.Fatal(got, err, calls, callbacks)
		}
	}
	var nilService *Service
	if got, err := nilService.GetImageRecord(context.Background(), ImageRecordRequest{ID: "selected"}); got != nil || err == nil {
		t.Fatal(got, err)
	}
	t.Run("legacy finite model remains strict", func(t *testing.T) {
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			return taskCoreJSON(req, 200, `{"size":"3","tags":false,"metadata":[1]}`), nil
		})
		got, err := New(client).GetImage(context.Background(), resource.ID("selected"))
		if got != nil || err == nil {
			t.Fatal("legacy decoder changed", got, err)
		}
		typed := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreJSON(req, 201, `{}`), nil }))
		if got, err := typed.GetImage(context.Background(), resource.ID("selected")); got != nil || err == nil {
			t.Fatal("legacy status changed", got, err)
		}
	})
	// Confirm the independent descriptor inventory has no accidental duplicates.
	if fields := strings.Fields(imageRecordViewKeys); len(fields) != 65 || len(imageRecordDefaults("null")) != 65 {
		t.Fatal(fields)
	}
}

// Existing marshaler fixtures supply fixed bytes or errors. This callback-only
// adapter tests capture before user code; transport and IO still use taskCore.
type imageRecordMarshalCallback func() ([]byte, error)

func (callback imageRecordMarshalCallback) MarshalJSON() ([]byte, error) { return callback() }

func TestImageRecordGetCapturesOptionsAndTopLevelAttributesBeforeMarshal(t *testing.T) {
	calls, marshals := 0, 0
	options := []ImageRecordOption{WithImageRecordHeader("X-Option", "captured")}
	attributes := map[string]any{"zzz_later": "captured scalar"}
	attributes["a_callback"] = imageRecordMarshalCallback(func() ([]byte, error) {
		marshals++
		options[0] = WithImageRecordHeader("X-Option", "changed by marshaler")
		attributes["zzz_later"] = "changed by marshaler"
		return []byte(`{"owned":true}`), nil
	})
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Option") != "captured" || req.URL.EscapedPath() != "/reverse/glance/v2/images/selected" {
			t.Fatal(req.URL, req.Header)
		}
		// Tolerated opaque response retains the constructor's captured properties.
		return taskCoreJSON(req, 200, "opaque response"), nil
	})
	got, err := New(client).GetImageRecord(context.Background(), ImageRecordRequest{ID: "selected", Attributes: attributes}, options...)
	if got == nil || err != nil || calls != 1 || marshals != 1 || got.Wire != nil {
		t.Fatal(got, err, calls, marshals)
	}
	th.CheckDeepEquals(t, map[string]json.RawMessage{"a_callback": json.RawMessage(`{"owned":true}`), "zzz_later": json.RawMessage(`"captured scalar"`)}, imageRecordProperties(t, got))
	if attributes["zzz_later"] != "changed by marshaler" {
		t.Fatal("mutation callback did not run", attributes)
	}
}
