package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const r2CloudComputeAggregateRows = `{"aggregates":[` +
	`{"id":1,"name":"rack1","availability_zone":"az1","hosts":["h1"],"metadata":{"ssd":"true"},"deleted":false},` +
	`{"id":2,"name":"rack2","availability_zone":"az2","hosts":[],"metadata":{},"deleted":false}]}`

func TestR2CloudComputeSearchAggregatesFiltersLocally(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	cloud.JSON("GET "+r2CloudComputeBase+"/os-aggregates", 200, r2CloudComputeAggregateRows)
	ctx := context.Background()
	for _, tc := range []struct {
		identity string
		options  []sdk.CloudServerReadOption
		want     []int
	}{
		{"", nil, []int{1, 2}},
		{"rack*", nil, []int{1, 2}},
		{"2", nil, []int{2}},
		{"", []sdk.CloudServerReadOption{sdk.WithCloudServerFilters(json.RawMessage(`{"availability_zone":"az1","metadata":{"ssd":"true"}}`))}, []int{1}},
		{"nope", nil, nil},
	} {
		result, err := cloud.Conn.SearchAggregates(ctx, tc.identity, tc.options...)
		if err != nil || result == nil || len(result.Inventory) != 2 || result.Expression {
			t.Fatal(result, err)
		}
		var got []int
		for _, a := range result.Aggregates {
			got = append(got, a.ID)
		}
		if len(got) != len(tc.want) {
			t.Fatalf("%q: %v", tc.identity, got)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("%q: %v", tc.identity, got)
			}
		}
	}
	result, err := cloud.Conn.SearchAggregates(ctx, "", sdk.WithCloudServerExpression("[?length(hosts)==`0`].name"))
	if err != nil || !result.Expression || string(result.Value) != `["rack2"]` || len(result.Aggregates) != 0 {
		t.Fatal(result, err)
	}
	for _, call := range cloud.Calls() {
		if call.Query != "" {
			t.Fatalf("%+v", call)
		}
	}
}

func TestR2CloudComputeSearchAggregatesRejectsServerOnlyOptionsBeforeHTTP(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	_, err := cloud.Conn.SearchAggregates(context.Background(), "", sdk.WithCloudServerListOptions(servers.WithRecordFindAllProjects(true)))
	if !errors.Is(err, resource.ErrInvalidOption) || len(cloud.Calls()) != 0 {
		t.Fatal(err, cloud.Calls())
	}
	_, err = cloud.Conn.SearchAggregates(context.Background(), "", sdk.WithCloudServerBare(true))
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

// The filters see the Source Aggregate attributes built from the row Nova
// returned: null availability zones and timestamps stay null, absent members
// are null, timestamp strings stay as sent, deleted is is_deleted, and
// location carries the aggregate's zone.
func TestR2CloudComputeSearchAggregatesFilterRawSourceView(t *testing.T) {
	cloud := r2CloudComputeNew(t)
	cloud.JSON("GET "+r2CloudComputeBase+"/os-aggregates", 200, `{"aggregates":[`+
		`{"id":1,"name":"plain","uuid":"u1","availability_zone":null,"hosts":[],"metadata":{},"deleted":false,`+
		`"created_at":"2026-10-24T01:02:03","updated_at":null,"deleted_at":null},`+
		`{"id":2,"name":"zoned","uuid":"u2","availability_zone":"az1","hosts":["h1"],"metadata":{"ssd":"true"},"deleted":false,`+
		`"created_at":"2026-10-24T01:02:03.000000"}]}`)
	ctx := context.Background()
	ids := func(options ...sdk.CloudServerReadOption) []int {
		t.Helper()
		result, err := cloud.Conn.SearchAggregates(ctx, "", options...)
		if err != nil || result.Expression {
			t.Fatal(result, err)
		}
		got := []int{}
		for _, a := range result.Aggregates {
			got = append(got, a.ID)
		}
		return got
	}
	for filter, want := range map[string]string{
		`{"availability_zone":null}`:             `[1]`,
		`{"availability_zone":""}`:               `[]`,
		`{"created_at":"2026-10-24T01:02:03"}`:   `[1]`,
		`{"hosts":["h1"]}`:                       `[2]`,
		`{"is_deleted":false,"updated_at":null}`: `[1,2]`,
		`{"location":{"zone":"az1"}}`:            `[2]`,
		`{"metadata":{"ssd":"true"}}`:            `[2]`,
	} {
		got, _ := json.Marshal(ids(sdk.WithCloudServerFilters(json.RawMessage(filter))))
		if string(got) != want {
			t.Fatalf("%s: %s, want %s", filter, got, want)
		}
	}
	result, err := cloud.Conn.SearchAggregates(ctx, "", sdk.WithCloudServerExpression("[].[created_at, updated_at, deleted_at, is_deleted]"))
	if err != nil || string(result.Value) != `[["2026-10-24T01:02:03",null,null,false],["2026-10-24T01:02:03.000000",null,null,false]]` {
		t.Fatal(result, err)
	}
	current, err := cloud.Conn.CurrentLocation()
	if err != nil {
		t.Fatal(err)
	}
	want, err := current.ForResource(nil, json.RawMessage("null"))
	if err != nil {
		t.Fatal(err)
	}
	result, err = cloud.Conn.SearchAggregates(ctx, "", sdk.WithCloudServerExpression("[0].location"))
	if err != nil || !jsonEqual(t, result.Value, want) {
		t.Fatalf("%s, want %s: %v", result.Value, want, err)
	}
}

func jsonEqual(t *testing.T, a, b json.RawMessage) bool {
	t.Helper()
	var left, right any
	if json.Unmarshal(a, &left) != nil || json.Unmarshal(b, &right) != nil {
		return false
	}
	l, _ := json.Marshal(left)
	r, _ := json.Marshal(right)
	return string(l) == string(r)
}
