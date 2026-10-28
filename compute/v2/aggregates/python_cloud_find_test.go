package aggregates_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// pythonCloudAggregateAPI answers GET os-aggregates/7 and the list; every
// other member GET is a Nova 404, as for a name sent to the member route.
func pythonCloudAggregateAPI(t *testing.T) (*aggregates.API, *testcloud.Recorder) {
	t.Helper()
	return pythonAggregateAPI(t, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == pythonAggregateBase:
			return pythonAggregateWire(200, `{"aggregates":[`+pythonAggregateRow+`,{"id":8,"name":"rack2","hosts":[]}]}`)
		case req.Method == http.MethodGet && req.URL.Path != pythonAggregateBase+"/7":
			return pythonAggregateWire(404, `{"itemNotFound":{"message":"Aggregate could not be found."}}`)
		}
		return pythonAggregateWire(200, `{"aggregate":`+pythonAggregateRow+`}`)
	})
}

func TestPythonCloudGetAggregateFindsByLiteralIdentityThenList(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonCloudAggregateAPI(t)
	// get_aggregate("rack1") is find_aggregate("rack1", ignore_missing=True).
	got, err := api.FindIdentity(ctx, "rack1")
	if err != nil || got == nil || got.ID != 7 || got.UUID != "agg-uuid" || got.AvailabilityZone != "az1" || got.Metadata["availability_zone"] != "az1" {
		t.Fatal(got, err)
	}
	// get_aggregate("7") returns the member GET.
	if got, err := api.FindIdentity(ctx, "7"); err != nil || got == nil || got.Name != "rack1" {
		t.Fatal(got, err)
	}
	// get_aggregate("absent") returns None.
	if got, err := api.FindIdentity(ctx, "absent"); err != nil || got != nil {
		t.Fatal(got, err)
	}
	want := []pythonAggregateCall{
		{http.MethodGet, pythonAggregateBase + "/rack1", "", "", "2.81"},
		{http.MethodGet, pythonAggregateBase, "", "", "2.81"},
		{http.MethodGet, pythonAggregateBase + "/7", "", "", "2.81"},
		{http.MethodGet, pythonAggregateBase + "/absent", "", "", "2.81"},
		{http.MethodGet, pythonAggregateBase, "", "", "2.81"},
	}
	if !reflect.DeepEqual(pythonAggregateCalls(calls), want) {
		t.Fatalf("%+v", pythonAggregateCalls(calls))
	}
}

func TestPythonCloudAggregateActionsResolveTheAggregateFirst(t *testing.T) {
	ctx := context.Background()
	steps := []struct {
		python string
		call   func(*aggregates.API, int) (*aggregates.Aggregate, error)
		body   string
	}{
		{"set_aggregate_metadata('rack1', {'ssd': 'true', 'old': None})", func(api *aggregates.API, id int) (*aggregates.Aggregate, error) {
			return api.SetMetadata(ctx, id, aggregates.SetMetadataOpts{Metadata: map[string]any{"ssd": "true", "old": nil}})
		}, `{"set_metadata":{"metadata":{"old":null,"ssd":"true"}}}`},
		{"add_host_to_aggregate('rack1', 'cmp2')", func(api *aggregates.API, id int) (*aggregates.Aggregate, error) {
			return api.AddHost(ctx, id, aggregates.AddHostOpts{Host: "cmp2"})
		}, `{"add_host":{"host":"cmp2"}}`},
		{"remove_host_from_aggregate('rack1', 'cmp2')", func(api *aggregates.API, id int) (*aggregates.Aggregate, error) {
			return api.RemoveHost(ctx, id, aggregates.RemoveHostOpts{Host: "cmp2"})
		}, `{"remove_host":{"host":"cmp2"}}`},
	}
	for _, step := range steps {
		t.Run(step.python, func(t *testing.T) {
			api, calls := pythonCloudAggregateAPI(t)
			found, err := api.FindIdentity(ctx, "rack1", resource.WithIdentityFindIgnoreMissing(false))
			if err != nil || found == nil {
				t.Fatal(found, err)
			}
			got, err := step.call(api, found.ID)
			if err != nil || got.ID != 7 {
				t.Fatal(got, err)
			}
			want := []pythonAggregateCall{
				{http.MethodGet, pythonAggregateBase + "/rack1", "", "", "2.81"},
				{http.MethodGet, pythonAggregateBase, "", "", "2.81"},
				{http.MethodPost, pythonAggregateBase + "/7/action", "", step.body, "2.81"},
			}
			if !reflect.DeepEqual(pythonAggregateCalls(calls), want) {
				t.Fatalf("%+v", pythonAggregateCalls(calls))
			}
			// Python raises SDKException("Host aggregate ... not found.") before
			// any action; the strict lookup stops with ErrNotFound.
			calls.Reset()
			if missing, err := api.FindIdentity(ctx, "absent", resource.WithIdentityFindIgnoreMissing(false)); missing != nil || !errors.Is(err, resource.ErrNotFound) {
				t.Fatal(missing, err)
			}
			if calls.Len() != 2 {
				t.Fatalf("%+v", pythonAggregateCalls(calls))
			}
		})
	}
}
