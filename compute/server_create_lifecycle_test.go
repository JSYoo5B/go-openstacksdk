package compute_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
)

type createBodyFixture struct {
	cloud    *testcloud.Cloud
	client   *gophercloud.ServiceClient
	service  *compute.Service
	posts    atomic.Int32
	lookups  atomic.Int32
	body     map[string]any
	hints    any
	hasHints bool
}

func newCreateBodyFixture(t *testing.T, microversion string) *createBodyFixture {
	t.Helper()
	f := &createBodyFixture{cloud: testcloud.New(t)}
	client := f.cloud.Client("compute", "/v2.1")
	client.Microversion = microversion
	f.client = client
	f.service = compute.New(client, compute.Dependencies{Volume: func(_ context.Context, ref resource.Ref) (string, error) {
		f.lookups.Add(1)
		return "resolved-" + ref.String(), nil
	}})
	f.cloud.Mux.HandleFunc("GET /v2.1/os-server-groups", func(w http.ResponseWriter, r *http.Request) {
		f.lookups.Add(1)
		testcloud.JSON(w, 200, `{"server_groups":[{"id":"group-id","name":"anti"},{"id":"dup-1","name":"dup"},{"id":"dup-2","name":"dup"}]}`)
	})
	f.cloud.Mux.HandleFunc("POST /v2.1/servers", func(w http.ResponseWriter, r *http.Request) {
		f.posts.Add(1)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		f.body, _ = body["server"].(map[string]any)
		f.hints, f.hasHints = body["OS-SCH-HNT:scheduler_hints"]
		if _, ok := body["os:scheduler_hints"]; ok {
			t.Error("Gophercloud hint key must not be used")
		}
		testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
	})
	f.cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	return f
}

func (f *createBodyFixture) reset(microversion string) {
	f.client.Microversion = microversion
	f.posts.Store(0)
	f.lookups.Store(0)
	f.body, f.hints, f.hasHints = nil, nil, false
}

func TestServerCreatePropertiesSchedulerHintsGroupAndCount(t *testing.T) {
	f := newCreateBodyFixture(t, "2.90")
	hints := map[string]any{"same_host": []string{"a"}, "group": "replaced", "build_near_host_ip": "10.0.0.1"}
	hintOption := compute.WithSchedulerHints(hints)
	hints["same_host"] = []string{"mutated"}
	file := compute.ServerFile{Path: "/etc/motd", Contents: []byte("hi")}
	personality := compute.WithPersonality(file)
	file.Contents[0] = 'X'
	_, err := f.service.Servers.Create(context.Background(), compute.CreateServerRequest{Name: "web", Image: resource.ID("image"), Flavor: resource.ID("flavor")},
		hintOption, compute.WithServerGroup(resource.Name("anti")), compute.WithServerCount(2, 3), compute.WithDescription(""),
		compute.WithAdminPass("secret"), compute.WithDiskConfig(servers.Manual), compute.WithAccessIPv4("192.0.2.1"), compute.WithAccessIPv6("2001:db8::1"),
		compute.WithTags("a", "b"), compute.WithHostname("guest"), compute.WithHypervisorHostname("node-1"), compute.WithUserData([]byte("abcd")))
	if err != nil {
		t.Fatal(err)
	}
	if !f.hasHints || !reflect.DeepEqual(f.hints, map[string]any{"same_host": []any{"a"}, "group": "group-id", "build_near_host_ip": "10.0.0.1"}) {
		t.Fatal(f.hints)
	}
	want := map[string]any{"min_count": float64(2), "max_count": float64(3), "description": "", "adminPass": "secret", "OS-DCF:diskConfig": "MANUAL",
		"accessIPv4": "192.0.2.1", "accessIPv6": "2001:db8::1", "tags": []any{"a", "b"}, "hostname": "guest", "hypervisor_hostname": "node-1",
		// Python always base64-encodes userdata, even when it already decodes.
		"user_data": "YWJjZA=="}
	for key, value := range want {
		if !reflect.DeepEqual(f.body[key], value) {
			t.Errorf("%s=%#v want %#v", key, f.body[key], value)
		}
	}
	if f.lookups.Load() != 1 || f.posts.Load() != 1 {
		t.Fatal(f.lookups.Load(), f.posts.Load())
	}
	// Personality is rejected at 2.57+, so it is checked on an older client.
	f.reset("2.56")
	old := f
	if _, err := old.service.Servers.Create(context.Background(), compute.CreateServerRequest{Name: "web", Image: resource.ID("image"), Flavor: resource.ID("flavor")}, personality, compute.WithServerGroup(resource.ID("group-id"))); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(old.body["personality"], []any{map[string]any{"path": "/etc/motd", "contents": "aGk="}}) || !reflect.DeepEqual(old.hints, map[string]any{"group": "group-id"}) || old.lookups.Load() != 0 {
		t.Fatal(old.body, old.hints, old.lookups.Load())
	}
	if _, ok := old.body["min_count"]; ok {
		t.Fatal("count must be omitted without WithServerCount")
	}
}

func TestServerCreateDataVolumeMappingsFollowPythonOrder(t *testing.T) {
	for _, scenario := range []string{"image", "boot volume", "new boot volume"} {
		t.Run(scenario, func(t *testing.T) {
			f := newCreateBodyFixture(t, "2.67")
			request := compute.CreateServerRequest{Name: "web", Image: resource.ID("image"), Flavor: resource.ID("flavor")}
			options := []compute.CreateServerOption{compute.WithDataVolumes(resource.ID("vol-a"), resource.Name("data"))}
			var first map[string]any
			switch scenario {
			case "image":
				first = map[string]any{"source_type": "image", "destination_type": "local", "uuid": "image", "boot_index": float64(0), "delete_on_termination": true}
			case "boot volume":
				request.Image = resource.Ref{}
				options = append(options, compute.WithBootVolume(resource.ID("root")))
				first = map[string]any{"source_type": "volume", "destination_type": "volume", "uuid": "root", "boot_index": float64(0), "delete_on_termination": false}
			case "new boot volume":
				options = append(options, compute.WithBootVolumeSize(50), compute.WithDeleteBootVolumeOnTermination(true))
				first = map[string]any{"source_type": "image", "destination_type": "volume", "uuid": "image", "boot_index": float64(0), "delete_on_termination": true, "volume_size": float64(50)}
			}
			if _, err := f.service.Servers.Create(context.Background(), request, options...); err != nil {
				t.Fatal(err)
			}
			want := []any{first,
				map[string]any{"source_type": "volume", "destination_type": "volume", "uuid": "vol-a", "boot_index": float64(-1), "delete_on_termination": false},
				map[string]any{"source_type": "volume", "destination_type": "volume", "uuid": "resolved-data", "boot_index": float64(-1), "delete_on_termination": false}}
			if !reflect.DeepEqual(f.body["block_device_mapping_v2"], want) || f.lookups.Load() != 1 {
				t.Fatalf("%#v lookups=%d", f.body["block_device_mapping_v2"], f.lookups.Load())
			}
			wantImage := "image"
			if scenario != "image" {
				wantImage = ""
			}
			if f.body["imageRef"] != wantImage {
				t.Fatal(f.body["imageRef"])
			}
		})
	}
}

func TestServerCreatePropertiesRejectBeforeRequests(t *testing.T) {
	request := compute.CreateServerRequest{Name: "web", Image: resource.ID("image"), Flavor: resource.ID("flavor")}
	f := newCreateBodyFixture(t, "2.90")
	for name, option := range map[string]compute.CreateServerOption{
		"count zero": compute.WithServerCount(0, 1), "count order": compute.WithServerCount(2, 1), "disk": compute.WithDiskConfig("SOMETIMES"),
		"ipv4": compute.WithAccessIPv4("2001:db8::1"), "ipv6": compute.WithAccessIPv6("192.0.2.1"), "hint value": compute.WithSchedulerHints(map[string]any{"x": make(chan int)}),
		"hint key": compute.WithSchedulerHints(map[string]any{" ": 1}), "nil hints": compute.WithSchedulerHints(nil), "tags": compute.WithTags(), "empty tag": compute.WithTags(""),
		"admin": compute.WithAdminPass(""), "volumes": compute.WithDataVolumes(), "zero volume": compute.WithDataVolumes(resource.Ref{}), "group": compute.WithServerGroup(resource.Ref{}),
		"hostname": compute.WithHostname(" "), "hypervisor": compute.WithHypervisorHostname(""), "personality": compute.WithPersonality(), "file": compute.WithPersonality(compute.ServerFile{}),
	} {
		t.Run(name, func(t *testing.T) {
			f.reset("2.90")
			if _, err := f.service.Servers.Create(context.Background(), request, option); !errors.Is(err, resource.ErrInvalidOption) || f.posts.Load() != 0 {
				t.Fatal(err, f.posts.Load())
			}
		})
	}
	for name, scenario := range map[string]struct {
		version string
		option  compute.CreateServerOption
	}{
		"description": {"2.18", compute.WithDescription("d")}, "tags": {"2.51", compute.WithTags("t")}, "hypervisor": {"2.73", compute.WithHypervisorHostname("n")},
		"hostname": {"2.89", compute.WithHostname("h")}, "personality": {"2.57", compute.WithPersonality(compute.ServerFile{Path: "/p"})},
	} {
		t.Run(name, func(t *testing.T) {
			f.reset(scenario.version)
			if _, err := f.service.Servers.Create(context.Background(), request, scenario.option); !errors.Is(err, resource.ErrUnsupported) || f.posts.Load() != 0 {
				t.Fatal(err, f.posts.Load())
			}
		})
	}
	for name, group := range map[string]string{"missing": "absent", "duplicate": "dup"} {
		t.Run(name, func(t *testing.T) {
			f.reset("2.90")
			if _, err := f.service.Servers.Create(context.Background(), request, compute.WithServerGroup(resource.Name(group))); err == nil || f.posts.Load() != 0 || f.lookups.Load() != 1 {
				t.Fatal(err, f.posts.Load(), f.lookups.Load())
			}
		})
	}
}

func TestServerFaultErrorFollowsPythonKeyPresence(t *testing.T) {
	for name, scenario := range map[string]struct {
		fault   string
		has     bool
		message string
	}{
		"message":      {`,"fault":{"message":"No valid host"}`, true, "No valid host"},
		"empty":        {`,"fault":{"message":""}`, true, ""},
		"null message": {`,"fault":{"message":null}`, true, "None"},
		"no message":   {`,"fault":{"code":500}`, false, ""},
		"null fault":   {`,"fault":null`, false, ""},
		"absent":       {``, false, ""},
	} {
		t.Run(name, func(t *testing.T) {
			f := newAutomaticFixture(t)
			f.rawBody = func(int32) (int, string) {
				return 200, `{"server":{"id":"server","status":"ERROR"` + scenario.fault + `}}`
			}
			result, err := f.service.WaitForServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, serverReadyOptions()...)
			var fault *compute.ServerFaultError
			if !errors.As(err, &fault) || !errors.Is(err, resource.ErrFailedState) || fault.HasFaultMessage != scenario.has || fault.FaultMessage != scenario.message || fault.Server == nil || fault.ServerID != "server" || result == nil || result.Server.Status != "ERROR" || f.raw.Load() != 1 {
				t.Fatal(result, err, f.raw.Load())
			}
			prefix := "Error in creating the server (no further information available)"
			if scenario.has {
				prefix = "Error in creating the server. Compute service reports fault: " + scenario.message
			}
			if !strings.HasPrefix(fault.Error(), prefix) {
				t.Fatal(fault.Error())
			}
		})
	}
	for _, message := range []string{"boot failed", ""} {
		f := newAutomaticFixture(t)
		server := automaticServer(t, autoFixed)
		server.Status, server.Fault.Message = "ERROR", message
		_, err := f.service.GetActiveServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: server}, serverReadyOptions()...)
		var fault *compute.ServerFaultError
		if !errors.As(err, &fault) || fault.HasFaultMessage != (message != "") || fault.FaultMessage != message || f.raw.Load() != 0 {
			t.Fatal(err, f.raw.Load())
		}
	}
}

func TestServerWithoutAddressesDeletePolicyIsExplicitAndSingleShot(t *testing.T) {
	for _, scenario := range []string{"get 204", "get 404", "get conflict", "wait", "create", "keep"} {
		t.Run(scenario, func(t *testing.T) {
			f := newAutomaticFixture(t)
			var deletes atomic.Int32
			code := 204
			switch scenario {
			case "get 404":
				code = 404
			case "get conflict":
				code = 409
			}
			f.cloud.Mux.HandleFunc("DELETE /v2.1/servers/server", func(w http.ResponseWriter, r *http.Request) {
				deletes.Add(1)
				if code == 204 {
					w.WriteHeader(204)
					return
				}
				testcloud.JSON(w, code, `{"error":"no"}`)
			})
			f.rawBody = func(int32) (int, string) {
				if deletes.Load() > 0 {
					return 404, `{"itemNotFound":{"message":"gone"}}`
				}
				return 200, `{"server":{"id":"server","status":"ACTIVE","addresses":{}}}`
			}
			policy := compute.DeleteServerWithoutAddresses
			if scenario == "keep" {
				policy = compute.KeepServerWithoutAddresses
			}
			opts := append(serverReadyOptions(), compute.WithServerReadyAutomaticIPOptions(compute.WithAutomaticIPEnabled(false)), compute.WithServerWithoutAddressesPolicy(policy))
			var result *compute.AutomaticServerIPResult
			var err error
			switch scenario {
			case "wait", "keep":
				result, err = f.service.WaitForServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, opts...)
			case "create":
				addAutomaticCreate(f, `{"server":{"id":"server","status":"BUILD"}}`, 202, nil)
				o := automaticCreateOptions()
				o.AutomaticIP = append(o.AutomaticIP, compute.WithAutomaticIPEnabled(false))
				o.ServerWithoutAddresses = compute.DeleteServerWithoutAddresses
				var created *compute.AutomaticServerCreateResult
				created, err = f.service.CreateWithAutomaticFloatingIP(context.Background(), automaticCreateRequest(), o)
				if created == nil || created.Server == nil || created.Server.ID != "server" {
					t.Fatal(created, err)
				}
			default:
				result, err = f.service.GetActiveServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, "{}")}, opts...)
			}
			var unavailable *compute.ServerAddressesUnavailableError
			if !errors.Is(err, compute.ErrServerAddressesUnavailable) || !errors.As(err, &unavailable) {
				t.Fatal(result, err)
			}
			if scenario == "keep" {
				if unavailable.Deletion != nil || deletes.Load() != 0 {
					t.Fatal(unavailable.Deletion, deletes.Load())
				}
				return
			}
			d := unavailable.Deletion
			if d == nil || deletes.Load() != 1 || d.StatusCode != code {
				t.Fatal(d, deletes.Load())
			}
			switch scenario {
			case "get 204":
				if d.Err != nil || d.NotFound || d.Waited || f.raw.Load() != 0 {
					t.Fatal(d, f.raw.Load())
				}
			case "get 404":
				if d.Err != nil || !d.NotFound || strings.Contains(err.Error(), "could not be deleted") {
					t.Fatal(d, err)
				}
			case "get conflict":
				if d.Err == nil || !strings.Contains(err.Error(), "AND then could not be deleted") {
					t.Fatal(d, err)
				}
			case "wait", "create":
				if d.Err != nil || !d.Waited {
					t.Fatal(d, err)
				}
			}
		})
	}
	if _, err := newAutomaticFixture(t).service.WaitForServer(context.Background(), compute.AutomaticFloatingIPRequest{Server: automaticServer(t, autoFixed)}, compute.WithServerWithoutAddressesPolicy(7)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	f := newAutomaticFixture(t)
	posts := addAutomaticCreate(f, `{"server":{"id":"server"}}`, 202, nil)
	o := automaticCreateOptions()
	o.ServerWithoutAddresses = 9
	if _, err := f.service.CreateWithAutomaticFloatingIP(context.Background(), automaticCreateRequest(), o); !errors.Is(err, resource.ErrInvalidOption) || posts.Load() != 0 {
		t.Fatal(err, posts.Load())
	}
}
