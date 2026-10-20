package servers_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

const nativeServerActionPath = "/nova/v2.1/servers/s1/action"

func TestNativeServerActionsRoutesBodiesAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeServerCall
	nativeServerRecorder(t, cloud, &calls, func(req *http.Request) *http.Response {
		body := calls[len(calls)-1].body
		switch {
		case strings.HasPrefix(body, `{"rebuild"`):
			return nativeServerWire(202, `{"server":{"id":"s1","name":"vm2","adminPass":"np"}}`)
		case strings.HasPrefix(body, `{"rescue"`):
			return nativeServerWire(200, `{"adminPass":"rescue-pass"}`)
		case strings.HasPrefix(body, `{"createImage"`):
			response := nativeServerWire(202, "")
			response.Header.Set("X-OpenStack-Nova-API-Version", "2.1")
			response.Header.Set("Location", "http://glance/v2/images/img1")
			return response
		case strings.HasPrefix(body, `{"confirmResize"`):
			return nativeServerWire(204, "")
		}
		return nativeServerWire(202, "")
	})
	api := servers.New(nativeServerClient(cloud))
	ctx := context.Background()
	steps := []struct {
		body string
		call func() error
	}{
		{`{"changePassword":{"adminPass":"pw"}}`, func() error { return api.ChangeAdminPassword(ctx, "s1", "pw") }},
		{`{"reboot":{"type":"SOFT","x_extension":1}}`, func() error {
			return api.Reboot(ctx, "s1", servers.RebootOpts{Type: servers.SoftReboot}, servers.WithRebootField("x_extension", 1))
		}},
		{`{"rebuild":{"OS-DCF:diskConfig":"AUTO","adminPass":"p","imageRef":"img","metadata":{"k":"v"},"name":"vm2"}}`, func() error {
			server, err := api.Rebuild(ctx, "s1", servers.RebuildOpts{ImageRef: "img", Name: "vm2", AdminPass: "p", Metadata: map[string]string{"k": "v"}, DiskConfig: servers.Auto})
			if err == nil && (server.ID != "s1" || server.Name != "vm2" || server.AdminPass != "np") {
				return fmt.Errorf("rebuild decode %+v", server)
			}
			return err
		}},
		// imageRef carries no omitempty, so an empty rebuild still sends it.
		{`{"rebuild":{"imageRef":""}}`, func() error { _, err := api.Rebuild(ctx, "s1", servers.RebuildOpts{}); return err }},
		{`{"resize":{"OS-DCF:diskConfig":"MANUAL","flavorRef":"f2"}}`, func() error {
			return api.Resize(ctx, "s1", servers.ResizeOpts{FlavorRef: "f2", DiskConfig: servers.Manual})
		}},
		{`{"confirmResize":null}`, func() error { return api.ConfirmResize(ctx, "s1") }},
		{`{"revertResize":null}`, func() error { return api.RevertResize(ctx, "s1") }},
		{`{"createImage":{"metadata":{"a":"b"},"name":"snap"}}`, func() error {
			id, err := api.CreateImage(ctx, "s1", servers.CreateImageOpts{Name: "snap", Metadata: map[string]string{"a": "b"}})
			if err == nil && id != "img1" {
				return fmt.Errorf("image id %q", id)
			}
			return err
		}},
		{`{"rescue":{"adminPass":"p","rescue_image_ref":"ri"}}`, func() error {
			pass, err := api.Rescue(ctx, "s1", servers.RescueOpts{AdminPass: "p", RescueImageRef: "ri"})
			if err == nil && pass != "rescue-pass" {
				return fmt.Errorf("rescue pass %q", pass)
			}
			return err
		}},
		{`{"unrescue":null}`, func() error { return api.Unrescue(ctx, "s1") }},
		{`{"os-start":null}`, func() error { return api.Start(ctx, "s1") }},
		{`{"os-stop":null}`, func() error { return api.Stop(ctx, "s1") }},
		{`{"pause":null}`, func() error { return api.Pause(ctx, "s1") }},
		{`{"unpause":null}`, func() error { return api.Unpause(ctx, "s1") }},
		{`{"suspend":null}`, func() error { return api.Suspend(ctx, "s1") }},
		{`{"resume":null}`, func() error { return api.Resume(ctx, "s1") }},
		{`{"lock":null}`, func() error { return api.Lock(ctx, "s1") }},
		{`{"unlock":null}`, func() error { return api.Unlock(ctx, "s1") }},
		{`{"shelve":null}`, func() error { return api.Shelve(ctx, "s1") }},
		{`{"shelveOffload":null}`, func() error { return api.ShelveOffload(ctx, "s1") }},
		{`{"unshelve":null}`, func() error { return api.Unshelve(ctx, "s1", servers.UnshelveOpts{}) }},
		// Without an availability zone the action value is null, so an extension sits beside it.
		{`{"unshelve":null,"x_extension":1}`, func() error {
			return api.Unshelve(ctx, "s1", servers.UnshelveOpts{}, servers.WithUnshelveField("x_extension", 1))
		}},
		{`{"unshelve":{"availability_zone":"az1","x_extension":1}}`, func() error {
			return api.Unshelve(ctx, "s1", servers.UnshelveOpts{AvailabilityZone: "az1"}, servers.WithUnshelveField("x_extension", 1))
		}},
	}
	for i, step := range steps {
		if err := step.call(); err != nil {
			t.Fatal(step.body, err)
		}
		want := nativeServerCall{http.MethodPost, nativeServerActionPath, "", step.body}
		if len(calls) != i+1 || calls[i] != want {
			t.Fatalf("step %d: %+v", i, calls)
		}
	}
}

func TestNativeServerActionsStrictStatuses(t *testing.T) {
	ctx := context.Background()
	defaultCodes := []int{201, 202}
	type action struct {
		name     string
		accepted []int
		call     func(*servers.API) error
	}
	actions := []action{
		{"ChangeAdminPassword", defaultCodes, func(api *servers.API) error { return api.ChangeAdminPassword(ctx, "s1", "pw") }},
		{"Reboot", defaultCodes, func(api *servers.API) error {
			return api.Reboot(ctx, "s1", servers.RebootOpts{Type: servers.HardReboot})
		}},
		{"Rebuild", defaultCodes, func(api *servers.API) error {
			_, err := api.Rebuild(ctx, "s1", servers.RebuildOpts{ImageRef: "img"})
			return err
		}},
		{"Resize", defaultCodes, func(api *servers.API) error { return api.Resize(ctx, "s1", servers.ResizeOpts{FlavorRef: "f"}) }},
		{"ConfirmResize", []int{201, 202, 204}, func(api *servers.API) error { return api.ConfirmResize(ctx, "s1") }},
		{"CreateImage", []int{202}, func(api *servers.API) error {
			_, err := api.CreateImage(ctx, "s1", servers.CreateImageOpts{Name: "snap"})
			return err
		}},
		{"Rescue", []int{200}, func(api *servers.API) error { _, err := api.Rescue(ctx, "s1", servers.RescueOpts{}); return err }},
		{"Unshelve", defaultCodes, func(api *servers.API) error { return api.Unshelve(ctx, "s1", servers.UnshelveOpts{}) }},
	}
	for name, call := range map[string]func(*servers.API, context.Context, string) error{
		"RevertResize": (*servers.API).RevertResize, "Unrescue": (*servers.API).Unrescue,
		"Start": (*servers.API).Start, "Stop": (*servers.API).Stop,
		"Pause": (*servers.API).Pause, "Unpause": (*servers.API).Unpause,
		"Suspend": (*servers.API).Suspend, "Resume": (*servers.API).Resume,
		"Lock": (*servers.API).Lock, "Unlock": (*servers.API).Unlock,
		"Shelve": (*servers.API).Shelve, "ShelveOffload": (*servers.API).ShelveOffload,
	} {
		actions = append(actions, action{name, defaultCodes, func(api *servers.API) error { return call(api, ctx, "s1") }})
	}
	for _, call := range actions {
		for _, code := range []int{200, 201, 202, 204, 404} {
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Provider.HTTPClient.Transport = nativeServerTransport(func(req *http.Request) (*http.Response, error) {
					requests.Add(1)
					response := nativeServerWire(code, `{"server":{"id":"s1"},"adminPass":"x","image_id":"img"}`)
					response.Header.Set("X-OpenStack-Nova-API-Version", "2.45")
					return response, nil
				})
				err := call.call(servers.New(nativeServerClient(cloud)))
				if requests.Load() != 1 {
					t.Fatal(requests.Load())
				}
				if contains(call.accepted, code) {
					if err != nil {
						t.Fatal(err)
					}
					return
				}
				nativeServerOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) {
					t.Fatal(err, native)
				}
			})
		}
	}
}

func TestNativeServerCreateImageIDByMicroversion(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name, version, location, contentType, body, want string
		fails                                            bool
	}{
		{name: "2.45 body", version: "2.45", contentType: "application/json", body: `{"image_id":"img2"}`, want: "img2"},
		{name: "2.45 ignores Location", version: "2.90", location: "http://glance/v2/images/other", contentType: "application/json", body: `{"image_id":"img3"}`, want: "img3"},
		// A non-JSON body is skipped, so a successful call returns an empty ID without error.
		{name: "2.45 non JSON", version: "2.45", contentType: "text/plain", body: `{"image_id":"img4"}`, want: ""},
		{name: "2.44 Location", version: "2.44", location: "http://glance/v2/images/img5", contentType: "application/json", body: `{"image_id":"ignored"}`, want: "img5"},
		{name: "2.1 relative Location", version: "2.1", location: "/v2/images/img6", contentType: "application/json", want: "img6"},
		{name: "missing version header", location: "http://glance/v2/images/img7", contentType: "application/json", fails: true},
		{name: "bad version header", version: "latest", location: "http://glance/v2/images/img7", contentType: "application/json", fails: true},
		{name: "missing Location", version: "2.1", contentType: "application/json", fails: true},
		{name: "root Location", version: "2.1", location: "/", contentType: "application/json", fails: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests atomic.Int32
			cloud.Provider.HTTPClient.Transport = nativeServerTransport(func(req *http.Request) (*http.Response, error) {
				requests.Add(1)
				response := nativeServerWire(202, tc.body)
				response.Header = http.Header{"Content-Type": {tc.contentType}}
				if tc.version != "" {
					response.Header.Set("X-OpenStack-Nova-API-Version", tc.version)
				}
				if tc.location != "" {
					response.Header.Set("Location", tc.location)
				}
				return response, nil
			})
			id, err := servers.New(nativeServerClient(cloud)).CreateImage(ctx, "s1", servers.CreateImageOpts{Name: "snap"})
			if requests.Load() != 1 {
				t.Fatal(requests.Load())
			}
			if tc.fails {
				nativeServerOperation(t, err, "CreateImage")
				if id != "" {
					t.Fatal(id)
				}
				return
			}
			if err != nil || id != tc.want {
				t.Fatal(id, err)
			}
		})
	}
}

func TestNativeServerActionsPreflight(t *testing.T) {
	ctx := context.Background()
	cloud := testcloud.New(t)
	var requests atomic.Int32
	cloud.Provider.HTTPClient.Transport = nativeServerTransport(func(req *http.Request) (*http.Response, error) {
		requests.Add(1)
		return nativeServerWire(202, `{}`), nil
	})
	api := servers.New(nativeServerClient(cloud))
	checks := map[string]func() error{
		"reboot without type": func() error { return api.Reboot(ctx, "s1", servers.RebootOpts{}) },
		"reboot type extension": func() error {
			return api.Reboot(ctx, "s1", servers.RebootOpts{Type: servers.SoftReboot}, servers.WithRebootField("type", "HARD"))
		},
		"rebuild bad disk config": func() error {
			_, err := api.Rebuild(ctx, "s1", servers.RebuildOpts{ImageRef: "i", DiskConfig: "BAD"})
			return err
		},
		"rebuild name extension": func() error {
			_, err := api.Rebuild(ctx, "s1", servers.RebuildOpts{ImageRef: "i"}, servers.WithRebuildField("name", "x"))
			return err
		},
		"resize without flavor":     func() error { return api.Resize(ctx, "s1", servers.ResizeOpts{}) },
		"resize bad disk config":    func() error { return api.Resize(ctx, "s1", servers.ResizeOpts{FlavorRef: "f", DiskConfig: "BAD"}) },
		"create image without name": func() error { _, err := api.CreateImage(ctx, "s1", servers.CreateImageOpts{}); return err },
		"rescue nil option":         func() error { _, err := api.Rescue(ctx, "s1", servers.RescueOpts{}, nil); return err },
		"unshelve zone extension": func() error {
			return api.Unshelve(ctx, "s1", servers.UnshelveOpts{}, servers.WithUnshelveField("availability_zone", "az"))
		},
	}
	for name, check := range checks {
		err := check()
		if err == nil {
			t.Fatal(name, "accepted")
		}
		operation := map[string]string{"reboot": "Reboot", "rebuild": "Rebuild", "resize": "Resize", "create": "CreateImage", "rescue": "Rescue", "unshelve": "Unshelve"}[strings.Fields(name)[0]]
		nativeServerOperation(t, err, operation)
	}
	if requests.Load() != 0 {
		t.Fatal(requests.Load())
	}
}
