package aggregates_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonAggregateCall records the wire request openstacksdk would also send.
type pythonAggregateCall struct{ method, path, query, body, version string }

type pythonAggregateTransport func(*http.Request) (*http.Response, error)

func (transport pythonAggregateTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonAggregateWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonAggregateAPI(t *testing.T, reply func(*http.Request) *http.Response) (*aggregates.API, *[]pythonAggregateCall) {
	t.Helper()
	calls := &[]pythonAggregateCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonAggregateTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonAggregateCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("X-OpenStack-Nova-API-Version")})
		return reply(req), nil
	})
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	client.Microversion = "2.81"
	return aggregates.New(client), calls
}

const pythonAggregateBase = "/nova/v2.1/os-aggregates"

const pythonAggregateRow = `{"id":7,"uuid":"agg-uuid","name":"rack1","availability_zone":"az1","hosts":["cmp1"],"metadata":{"availability_zone":"az1"},"created_at":"2026-10-01T01:02:03.000000","updated_at":null,"deleted_at":null,"deleted":false}`

func TestPythonAggregateCreateGetListDeleteMatchProxyRequests(t *testing.T) {
	ctx := context.Background()
	deleteStatus := http.StatusNotFound
	api, calls := pythonAggregateAPI(t, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodDelete:
			return pythonAggregateWire(deleteStatus, `{"itemNotFound":{"message":"gone"}}`)
		case req.Method == http.MethodGet && req.URL.Path == pythonAggregateBase:
			return pythonAggregateWire(200, `{"aggregates":[`+pythonAggregateRow+`,{"id":8,"name":"rack2","hosts":[]}]}`)
		}
		return pythonAggregateWire(200, `{"aggregate":`+pythonAggregateRow+`}`)
	})
	// create_aggregate(name="rack1", availability_zone="az1")
	created, err := api.Create(ctx, aggregates.CreateOpts{Name: "rack1", AvailabilityZone: "az1"})
	if err != nil || created.ID != 7 || created.UUID != "agg-uuid" || created.AvailabilityZone != "az1" || !reflect.DeepEqual(created.Hosts, []string{"cmp1"}) || created.Metadata["availability_zone"] != "az1" {
		t.Fatal(created, err)
	}
	// get_aggregate("7"); Go takes the integer ID.
	if got, err := api.Get(ctx, 7); err != nil || got.Name != "rack1" {
		t.Fatal(got, err)
	}
	// aggregates() and aggregates(name="rack2"), which Python filters locally.
	all, err := api.All(ctx)
	if err != nil || len(all) != 2 {
		t.Fatal(all, err)
	}
	named, err := api.All(ctx, resource.WithName("rack2"))
	if err != nil || len(named) != 1 || named[0].ID != 8 {
		t.Fatal(named, err)
	}
	// delete_aggregate("7") ignores 404; ignore_missing=False reports it.
	if err := api.Remove(ctx, resource.ID("7")); err != nil {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("7"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	if err := api.Delete(ctx, 7); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
	deleteStatus = http.StatusOK
	if err := api.Remove(ctx, resource.ID("7")); err != nil {
		t.Fatal(err)
	}
	want := []pythonAggregateCall{
		{http.MethodPost, pythonAggregateBase, "", `{"aggregate":{"availability_zone":"az1","name":"rack1"}}`, "2.81"},
		{http.MethodGet, pythonAggregateBase + "/7", "", "", "2.81"},
		{http.MethodGet, pythonAggregateBase, "", "", "2.81"},
		{http.MethodGet, pythonAggregateBase, "", "", "2.81"},
		{http.MethodDelete, pythonAggregateBase + "/7", "", "", "2.81"},
		{http.MethodDelete, pythonAggregateBase + "/7", "", "", "2.81"},
		{http.MethodDelete, pythonAggregateBase + "/7", "", "", "2.81"},
		{http.MethodDelete, pythonAggregateBase + "/7", "", "", "2.81"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonAggregateHostAndMetadataActionsReturnAggregate(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonAggregateAPI(t, func(*http.Request) *http.Response {
		return pythonAggregateWire(200, `{"aggregate":`+pythonAggregateRow+`}`)
	})
	steps := []struct {
		python string
		call   func() (*aggregates.Aggregate, error)
	}{
		{"add_host_to_aggregate(7, 'cmp2')", func() (*aggregates.Aggregate, error) {
			return api.AddHost(ctx, 7, aggregates.AddHostOpts{Host: "cmp2"})
		}},
		{"remove_host_from_aggregate(7, 'cmp2')", func() (*aggregates.Aggregate, error) {
			return api.RemoveHost(ctx, 7, aggregates.RemoveHostOpts{Host: "cmp2"})
		}},
		{"set_aggregate_metadata(7, ssd='true', old=None)", func() (*aggregates.Aggregate, error) {
			return api.SetMetadata(ctx, 7, aggregates.SetMetadataOpts{Metadata: map[string]any{"ssd": "true", "old": nil}})
		}},
		// Python sends an empty mapping when neither metadata nor kwargs are given.
		{"set_aggregate_metadata(7)", func() (*aggregates.Aggregate, error) {
			return api.SetMetadata(ctx, 7, aggregates.SetMetadataOpts{Metadata: map[string]any{}})
		}},
	}
	for _, step := range steps {
		got, err := step.call()
		if err != nil || got.ID != 7 || got.Name != "rack1" {
			t.Fatal(step.python, got, err)
		}
	}
	action := pythonAggregateBase + "/7/action"
	want := []pythonAggregateCall{
		{http.MethodPost, action, "", `{"add_host":{"host":"cmp2"}}`, "2.81"},
		{http.MethodPost, action, "", `{"remove_host":{"host":"cmp2"}}`, "2.81"},
		{http.MethodPost, action, "", `{"set_metadata":{"metadata":{"old":null,"ssd":"true"}}}`, "2.81"},
		{http.MethodPost, action, "", `{"set_metadata":{"metadata":{}}}`, "2.81"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonAggregateUpdateCannotSendNullAvailabilityZone(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonAggregateAPI(t, func(*http.Request) *http.Response {
		return pythonAggregateWire(200, `{"aggregate":`+pythonAggregateRow+`}`)
	})
	// update_aggregate(7, name="rack9", availability_zone="az2")
	if got, err := api.Update(ctx, 7, aggregates.UpdateOpts{Name: "rack9", AvailabilityZone: "az2"}); err != nil || got.ID != 7 {
		t.Fatal(got, err)
	}
	// update_aggregate(7, availability_zone=None) removes the zone in Nova; the
	// typed key is omitted when empty and cannot be replaced by an extension.
	if _, err := api.Update(ctx, 7, aggregates.UpdateOpts{}, aggregates.WithUpdateField("availability_zone", nil)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	want := []pythonAggregateCall{
		{http.MethodPut, pythonAggregateBase + "/7", "", `{"aggregate":{"availability_zone":"az2","name":"rack9"}}`, "2.81"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonAggregateFindHasNoIDThenNameFallback(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonAggregateAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Path == pythonAggregateBase {
			return pythonAggregateWire(200, `{"aggregates":[`+pythonAggregateRow+`]}`)
		}
		return pythonAggregateWire(200, `{"aggregate":`+pythonAggregateRow+`}`)
	})
	// find_aggregate("rack1") first GETs os-aggregates/rack1; Go rejects the
	// non-numeric ID before HTTP instead of falling back to the list.
	if _, err := api.Find(ctx, resource.ID("rack1"), resource.WithIgnoreMissing()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if got, err := api.Find(ctx, resource.ID("7")); err != nil || got.Name != "rack1" {
		t.Fatal(got, err)
	}
	if got, err := api.Find(ctx, resource.Name("rack1")); err != nil || got.ID != 7 {
		t.Fatal(got, err)
	}
	want := []pythonAggregateCall{
		{http.MethodGet, pythonAggregateBase + "/7", "", "", "2.81"},
		{http.MethodGet, pythonAggregateBase, "", "", "2.81"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}
