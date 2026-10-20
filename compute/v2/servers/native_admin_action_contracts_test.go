package servers_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

func nativeServerAdminBool(value bool) *bool { return &value }

func nativeServerAdminString(value string) *string { return &value }

func TestNativeServerAdminActionsRoutesBodiesAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeServerCall
	var versions []string
	nativeServerRecorder(t, cloud, &calls, func(req *http.Request) *http.Response {
		versions = append(versions, req.Header.Get("X-OpenStack-Nova-API-Version"))
		body := calls[len(calls)-1].body
		switch {
		case strings.Contains(body, `"adminPass":"p"`):
			return nativeServerWire(200, `{"adminPass":"p"}`)
		case strings.Contains(body, `"host":"h1"`):
			return nativeServerWire(200, `{}`)
		case strings.HasPrefix(body, `{"evacuate"`):
			// Newer microversions answer evacuate without a body.
			return nativeServerWire(200, "")
		}
		return nativeServerWire(202, "")
	})
	client := nativeServerClient(cloud)
	api := servers.New(client)
	ctx := context.Background()
	evacuate := func(want string, opts servers.EvacuateOpts, options ...servers.EvacuateOption) func() error {
		return func() error {
			pass, err := api.Evacuate(ctx, "s1", opts, options...)
			if err == nil && pass != want {
				return fmt.Errorf("evacuate pass %q", pass)
			}
			return err
		}
	}
	steps := []struct {
		body string
		call func() error
	}{
		{`{"evacuate":{"adminPass":"p","host":"h1","onSharedStorage":true,"x_extension":1}}`, evacuate("p",
			servers.EvacuateOpts{Host: "h1", OnSharedStorage: true, AdminPass: "p"}, servers.WithEvacuateField("x_extension", 1))},
		{`{"evacuate":{"host":"h1","onSharedStorage":false}}`, evacuate("", servers.EvacuateOpts{Host: "h1"})},
		// onSharedStorage has no omitempty, so even an empty Evacuate sends it.
		{`{"evacuate":{"onSharedStorage":false}}`, evacuate("", servers.EvacuateOpts{})},
		{`{"forceDelete":""}`, func() error { return api.ForceDelete(ctx, "s1") }},
		{`{"injectNetworkInfo":null}`, func() error { return api.InjectNetworkInfo(ctx, "s1") }},
		{`{"os-migrateLive":{"block_migration":true,"disk_over_commit":false,"force":false,"host":"h2"}}`, func() error {
			return api.LiveMigrate(ctx, "s1", servers.LiveMigrateOpts{Host: nativeServerAdminString("h2"), BlockMigration: nativeServerAdminBool(true), DiskOverCommit: nativeServerAdminBool(false)}, servers.WithLiveMigrateField("force", false))
		}},
		// host has no omitempty, so the scheduler choice is sent as null and block_migration is omitted.
		{`{"os-migrateLive":{"host":null}}`, func() error { return api.LiveMigrate(ctx, "s1", servers.LiveMigrateOpts{}) }},
		{`{"migrate":null}`, func() error { return api.Migrate(ctx, "s1") }},
		{`{"resetNetwork":null}`, func() error { return api.ResetNetwork(ctx, "s1") }},
		{`{"os-resetState":{"state":"active"}}`, func() error { return api.ResetState(ctx, "s1", servers.StateActive) }},
		{`{"os-resetState":{"state":"error"}}`, func() error { return api.ResetState(ctx, "s1", servers.StateError) }},
		// Any state string is passed through without local validation.
		{`{"os-resetState":{"state":"bogus"}}`, func() error { return api.ResetState(ctx, "s1", servers.ServerState("bogus")) }},
	}
	for _, step := range steps {
		if err := step.call(); err != nil {
			t.Fatal(step.body, err)
		}
	}
	// A pinned microversion only adds the header; the bodies stay the same.
	client.Microversion = "2.95"
	if _, err := api.Evacuate(ctx, "s1", servers.EvacuateOpts{}); err != nil {
		t.Fatal(err)
	}
	if err := api.LiveMigrate(ctx, "s1", servers.LiveMigrateOpts{}); err != nil {
		t.Fatal(err)
	}
	want := make([]nativeServerCall, 0, len(steps)+2)
	for _, step := range steps {
		want = append(want, nativeServerCall{http.MethodPost, nativeServerActionPath, "", step.body})
	}
	want = append(want,
		nativeServerCall{http.MethodPost, nativeServerActionPath, "", `{"evacuate":{"onSharedStorage":false}}`},
		nativeServerCall{http.MethodPost, nativeServerActionPath, "", `{"os-migrateLive":{"host":null}}`},
	)
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
	if versions[len(versions)-1] != "2.95" || versions[len(versions)-2] != "2.95" || versions[0] != "" {
		t.Fatal(versions)
	}
}

func TestNativeServerAdminActionsStrictStatuses(t *testing.T) {
	ctx := context.Background()
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*servers.API) error
	}{
		// Evacuate accepts only 200, while the other admin actions use the POST defaults.
		{"Evacuate", []int{200}, func(api *servers.API) error { _, err := api.Evacuate(ctx, "s1", servers.EvacuateOpts{}); return err }},
		{"ForceDelete", []int{201, 202}, func(api *servers.API) error { return api.ForceDelete(ctx, "s1") }},
		{"InjectNetworkInfo", []int{201, 202}, func(api *servers.API) error { return api.InjectNetworkInfo(ctx, "s1") }},
		{"LiveMigrate", []int{201, 202}, func(api *servers.API) error { return api.LiveMigrate(ctx, "s1", servers.LiveMigrateOpts{}) }},
		{"Migrate", []int{201, 202}, func(api *servers.API) error { return api.Migrate(ctx, "s1") }},
		{"ResetNetwork", []int{201, 202}, func(api *servers.API) error { return api.ResetNetwork(ctx, "s1") }},
		{"ResetState", []int{201, 202}, func(api *servers.API) error { return api.ResetState(ctx, "s1", servers.StateActive) }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls []nativeServerCall
				nativeServerRecorder(t, cloud, &calls, func(*http.Request) *http.Response { return nativeServerWire(code, `{}`) })
				err := call.call(servers.New(nativeServerClient(cloud)))
				nativeServerOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
}

func TestNativeServerAdminActionsPreflight(t *testing.T) {
	cloud := testcloud.New(t)
	var calls []nativeServerCall
	nativeServerRecorder(t, cloud, &calls, func(*http.Request) *http.Response { return nativeServerWire(202, "") })
	api := servers.New(nativeServerClient(cloud))
	ctx := context.Background()
	evacuate := func(options ...servers.EvacuateOption) error {
		_, err := api.Evacuate(ctx, "s1", servers.EvacuateOpts{}, options...)
		return err
	}
	for name, check := range map[string]struct {
		operation string
		err       error
	}{
		"evacuate shared storage": {"Evacuate", evacuate(servers.WithEvacuateField("onSharedStorage", true))},
		"evacuate omitted host":   {"Evacuate", evacuate(servers.WithEvacuateField("host", "h"))},
		"evacuate nil option":     {"Evacuate", evacuate(nil)},
		// block_migration is a typed *bool, so the 2.25 "auto" value cannot be sent as an extension either.
		"live migrate auto": {"LiveMigrate", api.LiveMigrate(ctx, "s1", servers.LiveMigrateOpts{}, servers.WithLiveMigrateField("block_migration", "auto"))},
		"live migrate host": {"LiveMigrate", api.LiveMigrate(ctx, "s1", servers.LiveMigrateOpts{}, servers.WithLiveMigrateField("host", "h"))},
		"live migrate nil":  {"LiveMigrate", api.LiveMigrate(ctx, "s1", servers.LiveMigrateOpts{}, nil)},
	} {
		if check.err == nil {
			t.Fatal(name, "accepted")
		}
		nativeServerOperation(t, check.err, check.operation)
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}
