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
