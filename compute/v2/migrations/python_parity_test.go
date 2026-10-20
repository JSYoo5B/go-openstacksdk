package migrations_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/migrations"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

// pythonMigrationCall records the wire request openstacksdk would also send.
type pythonMigrationCall struct{ method, path, query, version string }

type pythonMigrationTransport func(*http.Request) (*http.Response, error)

func (transport pythonMigrationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonMigrationWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonMigrationString(value string) *string { return &value }

const pythonMigrationRow = `{"id":12,"uuid":"mig-uuid","instance_uuid":"s1","migration_type":"live-migration","status":"completed","source_compute":"cmp1","source_node":"cmp1.node","dest_compute":"cmp2","dest_node":"cmp2.node","dest_host":"192.0.2.20","old_instance_type_id":1,"new_instance_type_id":1,"user_id":"u1","project_id":"p1","created_at":"2026-10-01T01:02:03.000000","updated_at":"2026-10-01T01:05:00.000000"}`

func TestPythonMigrationListMapsQueryAndReadsOnePage(t *testing.T) {
	ctx := context.Background()
	var calls []pythonMigrationCall
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonMigrationTransport(func(req *http.Request) (*http.Response, error) {
		calls = append(calls, pythonMigrationCall{req.Method, req.URL.Path, req.URL.RawQuery, req.Header.Get("X-OpenStack-Nova-API-Version")})
		// Python would follow this next link; Go reads one page.
		return pythonMigrationWire(200, `{"migrations":[`+pythonMigrationRow+`],"migrations_links":[{"rel":"next","href":"`+cloud.Server.URL+`/nova/v2.1/os-migrations?limit=1&marker=mig-uuid"}]}`), nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	client.Microversion = "2.80"
	api := migrations.New(client)

	since := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	limit := 1
	// migrations() and migrations(host=, status=, migration_type=, source_compute=,
	// user_id=, project_id=, changes_since=, changes_before=, server_id=, limit=, marker=)
	for _, options := range [][]migrations.ListOption{
		nil,
		{migrations.WithListOptions(migrations.ListOpts{
			Host: pythonMigrationString("cmp1"), Status: pythonMigrationString("completed"), MigrationType: pythonMigrationString("live-migration"),
			SourceCompute: pythonMigrationString("cmp1"), UserID: pythonMigrationString("u1"), ProjectID: pythonMigrationString("p1"),
			ChangesSince: &since, ChangesBefore: &before, InstanceID: pythonMigrationString("s1"), Limit: &limit, Marker: pythonMigrationString("m0"),
		})},
	} {
		var rows []*migrations.Migration
		for value, err := range api.List(ctx, options...) {
			if err != nil {
				t.Fatal(err)
			}
			rows = append(rows, value)
		}
		if len(rows) != 1 {
			t.Fatal(rows)
		}
		row := rows[0]
		if row.UUID != "mig-uuid" || row.ID != 12 || row.InstanceID != "s1" || row.MigrationType != "live-migration" || row.Status != "completed" || row.SourceCompute != "cmp1" || row.SourceNode != "cmp1.node" || row.DestCompute != "cmp2" || row.DestNode != "cmp2.node" || row.DestHost != "192.0.2.20" || row.OldInstanceTypeID != 1 || row.NewInstanceTypeID != 1 || row.UserID != "u1" || row.ProjectID != "p1" || row.CreatedAt.IsZero() || row.UpdatedAt.IsZero() {
			t.Fatal(row)
		}
	}
	want := []pythonMigrationCall{
		{http.MethodGet, "/nova/v2.1/os-migrations", "", "2.80"},
		{http.MethodGet, "/nova/v2.1/os-migrations", "changes-before=2026-10-02T00%3A00%3A00Z&changes-since=2026-10-01T00%3A00%3A00Z&host=cmp1&instance_uuid=s1&limit=1&marker=m0&migration_type=live-migration&project_id=p1&source_compute=cmp1&status=completed&user_id=u1", "2.80"},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
}
