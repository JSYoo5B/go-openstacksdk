package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const r2CloudComputeServerRows = `{"servers":[` +
	`{"id":"s1","name":"web-1","status":"ACTIVE","tenant_id":"p","metadata":{"role":"web"},"addresses":{}},` +
	`{"id":"s2","name":"web-2","status":"SHUTOFF","tenant_id":"p","metadata":{"role":"web"},"addresses":{}},` +
	`{"id":"s3","name":"db-1","status":"ACTIVE","tenant_id":"p","metadata":{},"addresses":{}}]}`

func r2CloudComputeServerIDs(t *testing.T, rows []*servers.ServerFindRecord) []string {
	t.Helper()
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		var id string
		if err := json.Unmarshal(row.Resource.Body["id"], &id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids
}

func TestR2CloudComputeListServerRecordsUsesDetailPathAndAllProjects(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/detail", 200, r2CloudComputeServerRows)
	rows, err := cloud.Conn.ListServerRecords(context.Background())
	if err != nil || len(rows) != 3 {
		t.Fatal(rows, err)
	}
	if got := strings.Join(r2CloudComputeServerIDs(t, rows), ","); got != "s1,s2,s3" {
		t.Fatal(got)
	}
	if rows[0].StatusCode != 200 || rows[0].Wire == nil || string(rows[0].Resource.Body["status"]) != `"ACTIVE"` {
		t.Fatalf("%+v", rows[0])
	}
	if _, err = cloud.Conn.ListServerRecords(context.Background(), sdk.WithCloudServerListOptions(servers.WithRecordFindAllProjects(true))); err != nil {
		t.Fatal(err)
	}
	calls := cloud.Calls()
	if len(calls) != 2 || calls[0].Query != "" || calls[1].Query != "all_tenants=True" || calls[1].Method != http.MethodGet {
		t.Fatalf("%+v", calls)
	}
}

func TestR2CloudComputeListServerRecordsWithoutDetailsAndQueryPushdown(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers", 200, `{"servers":[{"id":"s1","name":"web-1"}]}`)
	rows, err := cloud.Conn.ListServerRecords(context.Background(), sdk.WithCloudServerListOptions(
		servers.WithRecordFindDetails(false), servers.WithRecordFindQuery("status", "ACTIVE")))
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	calls := cloud.Calls()
	if len(calls) != 1 || calls[0].Path != r2CloudComputeBase+"/servers" || calls[0].Query != "status=ACTIVE" {
		t.Fatalf("%+v", calls)
	}
}

func TestR2CloudComputeSearchServerRecordsFiltersLocally(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/detail", 200, r2CloudComputeServerRows)
	ctx := context.Background()
	for _, tc := range []struct {
		name     string
		identity string
		options  []sdk.CloudServerReadOption
		want     string
	}{
		{"all", "", nil, "s1,s2,s3"},
		{"exact id", "s2", nil, "s2"},
		{"glob name", "web-*", nil, "s1,s2"},
		{"dictionary", "", []sdk.CloudServerReadOption{sdk.WithCloudServerFilters(json.RawMessage(`{"status":"ACTIVE","metadata":{"role":"web"}}`))}, "s1"},
		{"glob and dictionary", "web-*", []sdk.CloudServerReadOption{sdk.WithCloudServerFilters(json.RawMessage(`{"status":"SHUTOFF"}`))}, "s2"},
		{"missing", "absent", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := cloud.Conn.SearchServerRecords(ctx, tc.identity, tc.options...)
			if err != nil || result == nil || result.Expression || len(result.Inventory) != 3 {
				t.Fatal(result, err)
			}
			if got := strings.Join(r2CloudComputeServerIDs(t, result.Records), ","); got != tc.want {
				t.Fatal(got)
			}
			var value []json.RawMessage
			if err := json.Unmarshal(result.Value, &value); err != nil || len(value) != len(result.Records) {
				t.Fatal(string(result.Value), err)
			}
		})
	}
	for _, call := range cloud.Calls() {
		if call.Query != "" {
			t.Fatalf("filters must not reach Nova: %+v", call)
		}
	}
}

func TestR2CloudComputeSearchServerRecordsExpressionAndInvalidFilters(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/detail", 200, r2CloudComputeServerRows)
	ctx := context.Background()
	result, err := cloud.Conn.SearchServerRecords(ctx, "", sdk.WithCloudServerExpression("[?status=='ACTIVE'].name"))
	if err != nil || result == nil || !result.Expression || len(result.Records) != 0 || string(result.Value) != `["web-1","db-1"]` {
		t.Fatalf("%+v %v", result, err)
	}
	calls := len(cloud.Calls())
	_, err = cloud.Conn.SearchServerRecords(ctx, "", sdk.WithCloudServerFilters(json.RawMessage(`{`)))
	if !errors.Is(err, resource.ErrInvalidOption) || len(cloud.Calls()) != calls {
		t.Fatal(err, len(cloud.Calls()), calls)
	}
}

func TestR2CloudComputeSearchServerRecordsKeepsConsumedRowsOnPageFailure(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	cloud.Handle("GET "+r2CloudComputeBase+"/servers/detail", func(w http.ResponseWriter, r *http.Request, _ string) {
		if r.URL.Query().Get("marker") == "" {
			testcloud.JSON(w, 200, `{"servers":[{"id":"s1","name":"a"}],"servers_links":[{"rel":"next","href":"`+cloud.Server.URL+r2CloudComputeBase+`/servers/detail?marker=s1"}]}`)
			return
		}
		testcloud.JSON(w, 500, `{"computeFault":{"message":"boom","code":500}}`)
	})
	result, err := cloud.Conn.SearchServerRecords(context.Background(), "")
	if err == nil || result == nil || len(result.Inventory) != 1 || len(result.Records) != 0 {
		t.Fatalf("%+v %v", result, err)
	}
}

func TestR2CloudComputeServerReadsHonorCancellationBeforeHTTP(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := cloud.Conn.ListServerRecords(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := cloud.Conn.SearchServerRecords(ctx, ""); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := cloud.Conn.GetServerByID(ctx, "s1"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if len(cloud.Calls()) != 0 {
		t.Fatalf("%+v", cloud.Calls())
	}
}

func TestR2CloudComputeGetServerByIDReturnsNilOnNotFoundAndExpandsFound(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/gone", 404, `{"itemNotFound":{"message":"Instance gone could not be found.","code":404}}`)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/boom", 500, `{"computeFault":{"message":"x","code":500}}`)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s1", 200, `{"server":{"id":"s1","name":"web-1","status":"BUILD","accessIPv4":"203.0.113.9","addresses":{"private":[{"addr":"10.0.0.5","version":4,"OS-EXT-IPS:type":"fixed"}]}}}`)
	cloud.JSON("GET /network/v2.0/networks", 200, `{"networks":[]}`)
	ctx := context.Background()
	if got, err := cloud.Conn.GetServerByID(ctx, "gone"); got != nil || err != nil {
		t.Fatal(got, err)
	}
	if got, err := cloud.Conn.GetServerByID(ctx, "boom"); got != nil || err == nil {
		t.Fatal(got, err)
	}
	got, err := cloud.Conn.GetServerByID(ctx, "s1", sdk.WithCloudServerBare(true))
	if err != nil || got == nil || got.Server == nil || got.Server.ID != "s1" || got.Interfaces != nil {
		t.Fatalf("%+v %v", got, err)
	}
	got, err = cloud.Conn.GetServerByID(ctx, "s1")
	if err != nil || got == nil || got.Interfaces == nil || got.Interfaces.ServerID != "s1" || got.Interfaces.AccessIPv4 != "203.0.113.9" {
		t.Fatalf("%+v %v", got, err)
	}
	for _, call := range cloud.Calls() {
		if call.Method != http.MethodGet {
			t.Fatalf("%+v", call)
		}
	}
}

func TestR2CloudComputeUpdateServerFindsStrictlyAndSendsOnlySuppliedAttributes(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/web-1", 404, `{"itemNotFound":{"message":"nf","code":404}}`)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s1", 200, `{"server":{"id":"s1","name":"web-1","status":"ACTIVE"}}`)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/detail", 200, `{"servers":[{"id":"s1","name":"web-1","status":"ACTIVE"}]}`)
	cloud.Handle("PUT "+r2CloudComputeBase+"/servers/s1", func(w http.ResponseWriter, _ *http.Request, _ string) {
		testcloud.JSON(w, 200, `{"server":{"id":"s1","name":"renamed","status":"ACTIVE","description":null}}`)
	})
	ctx := context.Background()
	update := sdk.CloudServerUpdate{Name: request.Present("renamed"), Description: request.Null[string]()}
	got, err := cloud.Conn.UpdateServer(ctx, "web-1", update, sdk.WithCloudServerBare(true))
	if err != nil || got == nil || !got.Sent || got.Server == nil || got.Server.Name != "renamed" || got.Record == nil {
		t.Fatalf("%+v %v", got, err)
	}
	var put string
	for _, call := range cloud.Calls() {
		if call.Method == http.MethodPut {
			put = call.Body
		}
	}
	if put != `{"server":{"description":null,"name":"renamed"}}` {
		t.Fatal(put)
	}
	// Nothing to update: Python's commit sends nothing.
	before := len(cloud.Calls())
	got, err = cloud.Conn.UpdateServer(ctx, "s1", sdk.CloudServerUpdate{})
	if err != nil || got == nil || got.Sent || got.Server != nil || got.Record == nil {
		t.Fatalf("%+v %v", got, err)
	}
	for _, call := range cloud.Calls()[before:] {
		if call.Method == http.MethodPut {
			t.Fatalf("%+v", call)
		}
	}
}

func TestR2CloudComputeUpdateServerRejectsBadInputBeforeHTTPAndMissingServer(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/absent", 404, `{"itemNotFound":{"message":"nf","code":404}}`)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/detail", 200, `{"servers":[]}`)
	ctx := context.Background()
	for _, name := range []request.Optional[string]{request.Present(""), request.Null[string]()} {
		if _, err := cloud.Conn.UpdateServer(ctx, "s1", sdk.CloudServerUpdate{Name: name}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if len(cloud.Calls()) != 0 {
		t.Fatalf("%+v", cloud.Calls())
	}
	if _, err := cloud.Conn.UpdateServer(ctx, "absent", sdk.CloudServerUpdate{Name: request.Present("x")}); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	for _, call := range cloud.Calls() {
		if call.Method == http.MethodPut {
			t.Fatalf("%+v", call)
		}
	}
}
