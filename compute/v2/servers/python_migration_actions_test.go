package servers_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func pythonMigrationAPI(t *testing.T, version string, status int) (*servers.API, *testcloud.Recorder) {
	t.Helper()
	client, _, calls := pythonServerCloud(t, func(req *http.Request) *http.Response {
		if values, exists := req.Header["Accept"]; !exists || len(values) != 1 || values[0] != "" {
			t.Error("action must send the empty Accept header", req.Header)
		}
		return pythonServerWire(status, `{"badRequest":{"message":"no"}}`)
	})
	client.Microversion = version
	return servers.New(client), calls
}

func TestPythonMigrationEvacuateAndMigrateSendOnlySuppliedKeys(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonMigrationAPI(t, "2.96", http.StatusAccepted)
	steps := []struct {
		body string
		call func() error
	}{
		// evacuate_server(server)
		{`{"evacuate":{}}`, func() error { return api.EvacuateWithOptions(ctx, "s1") }},
		// evacuate_server(server, host="h", admin_pass="", force=False)
		{`{"evacuate":{"adminPass":"","force":false,"host":"h"}}`, func() error {
			return api.EvacuateWithOptions(ctx, "s1", servers.WithEvacuateServerHost("h"), servers.WithEvacuateServerAdminPass(""), servers.WithEvacuateServerForce(false))
		}},
		// evacuate_server(server, on_shared_storage=True)
		{`{"evacuate":{"onSharedStorage":true}}`, func() error {
			return api.EvacuateWithOptions(ctx, "s1", servers.WithEvacuateServerOnSharedStorage(true))
		}},
		{`{"evacuate":{"host":"h","x-ext":1}}`, func() error {
			return api.EvacuateWithOptions(ctx, "s1", servers.WithEvacuateServerOptions(servers.EvacuateServerOpts{Host: pythonMigrationPtr("h")}), servers.WithEvacuateServerField("x-ext", 1))
		}},
		// migrate_server(server)
		{`{"migrate":null}`, func() error { return api.MigrateWithOptions(ctx, "s1") }},
		// migrate_server(server, host="") keeps Python's truthiness check.
		{`{"migrate":null}`, func() error { return api.MigrateWithOptions(ctx, "s1", servers.WithMigrateServerHost("")) }},
		// migrate_server(server, host="h")
		{`{"migrate":{"host":"h"}}`, func() error { return api.MigrateWithOptions(ctx, "s1", servers.WithMigrateServerHost("h")) }},
	}
	for i, step := range steps {
		if err := step.call(); err != nil {
			t.Fatal(step.body, err)
		}
		want := pythonServerCall{http.MethodPost, pythonServerBase + "/action", "", step.body, "2.96"}
		if calls.Len() != i+1 || (pythonServerCalls(calls))[i] != want {
			t.Fatalf("step %d: %+v", i, pythonServerCalls(calls))
		}
	}

	old, oldCalls := pythonMigrationAPI(t, "2.55", http.StatusAccepted)
	if err := old.MigrateWithOptions(ctx, "s1", servers.WithMigrateServerHost("h")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if err := old.MigrateWithOptions(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	if want := (pythonServerCall{http.MethodPost, pythonServerBase + "/action", "", `{"migrate":null}`, "2.55"}); oldCalls.Len() != 1 || (pythonServerCalls(oldCalls))[0] != want {
		t.Fatalf("%+v", pythonServerCalls(oldCalls))
	}
}

func TestPythonMigrationLiveMigrateFollowsSelectedMicroversionShape(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name, client, version, body string
		options                     []servers.LiveMigrateServerOption
	}{
		// live_migrate_server(server) on a 2.30+ cloud.
		{"default 2.30", "2.96", "2.30", `{"os-migrateLive":{"block_migration":"auto","host":null}}`, nil},
		{"latest", "latest", "2.30", `{"os-migrateLive":{"block_migration":"auto","host":null}}`, nil},
		{"host force 2.30", "2.30", "2.30", `{"os-migrateLive":{"block_migration":false,"force":true,"host":"h"}}`, []servers.LiveMigrateServerOption{
			servers.WithLiveMigrateServerHost("h"), servers.WithLiveMigrateServerForce(true), servers.WithLiveMigrateServerBlockMigration(false),
		}},
		{"host without force 2.30", "2.96", "2.30", `{"os-migrateLive":{"block_migration":"auto","host":"h"}}`, []servers.LiveMigrateServerOption{
			servers.WithLiveMigrateServerHost("h"),
		}},
		{"force without host 2.30", "2.96", "2.30", `{"os-migrateLive":{"block_migration":true,"host":null}}`, []servers.LiveMigrateServerOption{
			servers.WithLiveMigrateServerForce(true), servers.WithLiveMigrateServerBlockMigration(true),
		}},
		{"default 2.25", "2.29", "2.25", `{"os-migrateLive":{"block_migration":"auto","host":null}}`, nil},
		{"host force 2.25", "2.25", "2.25", `{"os-migrateLive":{"block_migration":"auto","host":"h"}}`, []servers.LiveMigrateServerOption{
			servers.WithLiveMigrateServerHost("h"), servers.WithLiveMigrateServerForce(true), servers.WithLiveMigrateServerBlockMigration(true), servers.WithLiveMigrateServerAutoBlockMigration(),
		}},
		{"default legacy", "2.24", "2.24", `{"os-migrateLive":{"block_migration":false,"disk_over_commit":false,"host":null}}`, nil},
		{"legacy values", "2.1", "2.1", `{"os-migrateLive":{"block_migration":true,"disk_over_commit":true,"host":"h"}}`, []servers.LiveMigrateServerOption{
			servers.WithLiveMigrateServerOptions(servers.LiveMigrateServerOpts{Host: "h", Force: true, BlockMigration: pythonMigrationPtr(true), DiskOverCommit: pythonMigrationPtr(true)}),
		}},
		{"unversioned client", "", "", `{"os-migrateLive":{"block_migration":false,"disk_over_commit":false,"host":null}}`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			api, calls := pythonMigrationAPI(t, tc.client, http.StatusAccepted)
			if err := api.LiveMigrateWithOptions(ctx, "s1", tc.options...); err != nil {
				t.Fatal(err)
			}
			want := pythonServerCall{http.MethodPost, pythonServerBase + "/action", "", tc.body, tc.version}
			if calls.Len() != 1 || (pythonServerCalls(calls))[0] != want {
				t.Fatalf("%+v", pythonServerCalls(calls))
			}
		})
	}
	rejected := []struct {
		name, client string
		options      []servers.LiveMigrateServerOption
	}{
		{"host without force 2.25", "2.29", []servers.LiveMigrateServerOption{servers.WithLiveMigrateServerHost("h")}},
		{"host without force legacy", "2.24", []servers.LiveMigrateServerOption{servers.WithLiveMigrateServerHost("h")}},
		{"auto legacy", "2.24", []servers.LiveMigrateServerOption{servers.WithLiveMigrateServerAutoBlockMigration()}},
		{"auto and boolean", "2.96", []servers.LiveMigrateServerOption{
			servers.WithLiveMigrateServerOptions(servers.LiveMigrateServerOpts{AutoBlockMigration: true, BlockMigration: pythonMigrationPtr(false)}),
		}},
		{"version header conflicts with body version", "2.96", []servers.LiveMigrateServerOption{servers.WithLiveMigrateServerHeader("X-OpenStack-Nova-API-Version", "2.96")}},
		{"core key extension", "2.96", []servers.LiveMigrateServerOption{servers.WithLiveMigrateServerField("block_migration", "auto")}},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			api, calls := pythonMigrationAPI(t, tc.client, http.StatusAccepted)
			var operation *resource.OperationError
			if err := api.LiveMigrateWithOptions(ctx, "s1", tc.options...); !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "LiveMigrateWithOptions" {
				t.Fatal(err)
			}
			if calls.Len() != 0 {
				t.Fatalf("%+v", pythonServerCalls(calls))
			}
		})
	}
}

func TestPythonMigrationActionsStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	calls := map[string]func(*servers.API) error{
		"EvacuateWithOptions":    func(a *servers.API) error { return a.EvacuateWithOptions(ctx, "s1") },
		"MigrateWithOptions":     func(a *servers.API) error { return a.MigrateWithOptions(ctx, "s1") },
		"LiveMigrateWithOptions": func(a *servers.API) error { return a.LiveMigrateWithOptions(ctx, "s1") },
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			// Python raise_from_response accepts every status below 400.
			for _, status := range []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent, 302} {
				api, _ := pythonMigrationAPI(t, "2.96", status)
				if err := call(api); err != nil {
					t.Fatal(status, err)
				}
			}
			for _, status := range []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict} {
				api, _ := pythonMigrationAPI(t, "2.96", status)
				var operation *resource.OperationError
				err := call(api)
				if !gophercloud.ResponseCodeIs(err, status) || !errors.As(err, &operation) || operation.Operation != name || operation.Resource != "servers" {
					t.Fatal(status, err)
				}
			}
			var nilAPI *servers.API
			if err := call(nilAPI); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
	api, recorded := pythonMigrationAPI(t, "2.96", http.StatusAccepted)
	for _, err := range []error{
		api.EvacuateWithOptions(ctx, " "),
		api.EvacuateWithOptions(ctx, "s1", nil),
		api.EvacuateWithOptions(ctx, "s1", servers.WithEvacuateServerField("onSharedStorage", false)),
		api.MigrateWithOptions(ctx, "s1", servers.WithMigrateServerField("host", "h")),
		api.LiveMigrateWithOptions(ctx, "", servers.WithLiveMigrateServerHost("h")),
	} {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if recorded.Len() != 0 {
		t.Fatalf("%+v", pythonServerCalls(recorded))
	}
}

func pythonMigrationPtr[T any](value T) *T { return &value }
