package servers_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// pythonAdminServerCall records the wire request openstacksdk would also send.
type pythonAdminServerCall struct{ method, path, body, version string }

type pythonAdminServerTransport func(*http.Request) (*http.Response, error)

func (transport pythonAdminServerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonAdminServerAPI(t *testing.T) (*servers.API, *[]pythonAdminServerCall) {
	t.Helper()
	calls := &[]pythonAdminServerCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonAdminServerTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonAdminServerCall{req.Method, req.URL.Path, raw, req.Header.Get("X-OpenStack-Nova-API-Version")})
		code := http.StatusAccepted
		if strings.HasPrefix(raw, `{"evacuate"`) {
			code = http.StatusOK
		}
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	client.Microversion = "2.96"
	return servers.New(client), calls
}

func TestPythonAdminServerResetStateMatchesProxyBody(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonAdminServerAPI(t)
	// reset_server_state(server, "active") and reset_server_state(server, "error")
	for _, state := range []string{"active", "error"} {
		if err := api.ResetState(ctx, "s1", servers.ServerState(state)); err != nil {
			t.Fatal(state, err)
		}
	}
	action := "/nova/v2.1/servers/s1/action"
	want := []pythonAdminServerCall{
		{http.MethodPost, action, `{"os-resetState":{"state":"active"}}`, "2.96"},
		{http.MethodPost, action, `{"os-resetState":{"state":"error"}}`, "2.96"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonAdminServerMigrationActionArgumentGaps(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonAdminServerAPI(t)
	host, enabled := "cmp2", true
	// evacuate_server(server) sends {"evacuate": {}}; Go always adds onSharedStorage,
	// which Nova rejects from 2.14.
	if _, err := api.Evacuate(ctx, "s1", servers.EvacuateOpts{}); err != nil {
		t.Fatal(err)
	}
	// migrate_server(server) without host matches; Migrate takes no host option.
	if err := api.Migrate(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	// live_migrate_server(server, host="cmp2", force=True, block_migration=True) at 2.30.
	if err := api.LiveMigrate(ctx, "s1", servers.LiveMigrateOpts{Host: &host, BlockMigration: &enabled}, servers.WithLiveMigrateField("force", true)); err != nil {
		t.Fatal(err)
	}
	// live_migrate_server(server) defaults block_migration to "auto" from 2.25;
	// the typed key cannot carry a string.
	if err := api.LiveMigrate(ctx, "s1", servers.LiveMigrateOpts{}, servers.WithLiveMigrateField("block_migration", "auto")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	action := "/nova/v2.1/servers/s1/action"
	want := []pythonAdminServerCall{
		{http.MethodPost, action, `{"evacuate":{"onSharedStorage":false}}`, "2.96"},
		{http.MethodPost, action, `{"migrate":null}`, "2.96"},
		{http.MethodPost, action, `{"os-migrateLive":{"block_migration":true,"force":true,"host":"cmp2"}}`, "2.96"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}
