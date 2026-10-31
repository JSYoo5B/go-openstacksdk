package connectiontest_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func pyCloudComputeAggregateConnection(t *testing.T, deleteStatus int) (*sdk.Connection, *pyCloudComputeIdentityLog) {
	t.Helper()
	cloud := testcloud.New(t)
	log := &pyCloudComputeIdentityLog{}
	cloud.Mux.HandleFunc("/nova/v2.1/admin/os-aggregates", func(w http.ResponseWriter, r *http.Request) {
		log.add(r.Method + " /os-aggregates")
		testcloud.JSON(w, 200, `{"aggregates":[{"id":7,"name":"web","availability_zone":"az1","hosts":[],"metadata":{}}]}`)
	})
	cloud.Mux.HandleFunc("/nova/v2.1/admin/os-aggregates/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		line := r.Method + " " + strings.TrimPrefix(r.URL.Path, "/nova/v2.1/admin")
		if len(body) > 0 {
			line += " " + string(body)
		}
		log.add(line)
		switch {
		case r.Method == http.MethodGet:
			testcloud.JSON(w, 404, `{"itemNotFound":{"message":"nope"}}`)
		case r.Method == http.MethodPut:
			testcloud.JSON(w, 200, `{"aggregate":{"id":7,"name":"web","availability_zone":null}}`)
		case r.Method == http.MethodDelete:
			testcloud.JSON(w, deleteStatus, `{}`)
		}
	})
	c, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/nova/v2.1/admin"))
	if err != nil {
		t.Fatal(err)
	}
	return c, log
}

func TestPyCloudComputeUpdateAggregateFindsThenSendsOnlySuppliedNull(t *testing.T) {
	c, log := pyCloudComputeAggregateConnection(t, 200)
	record, err := c.UpdateAggregate(context.Background(), "web", aggregates.WithUpdateRecordAvailabilityZone(request.Null[string]()))
	if err != nil || record.Aggregate == nil || record.Aggregate.ID != 7 || !record.Sent {
		t.Fatalf("update: %+v %v", record, err)
	}
	want := `GET /os-aggregates/web|GET /os-aggregates|PUT /os-aggregates/7 {"aggregate":{"availability_zone":null}}`
	if log.String() != want {
		t.Fatalf("requests:\n got %s\nwant %s", log.String(), want)
	}
}

func TestPyCloudComputeUpdateAggregateMissingSendsNoPut(t *testing.T) {
	c, log := pyCloudComputeAggregateConnection(t, 200)
	if _, err := c.UpdateAggregate(context.Background(), "ghost", aggregates.WithUpdateRecordName(request.Present("x"))); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	if strings.Contains(log.String(), "PUT") {
		t.Fatal(log.String())
	}
}

func TestPyCloudComputeDeleteAggregateDigitsSkipLookupAndMapMissingToFalse(t *testing.T) {
	c, log := pyCloudComputeAggregateConnection(t, 200)
	if ok, err := c.DeleteAggregate(context.Background(), "7"); err != nil || !ok {
		t.Fatalf("delete: %v %v", ok, err)
	}
	if log.String() != "DELETE /os-aggregates/7" {
		t.Fatalf("requests: %s", log.String())
	}
	c, log = pyCloudComputeAggregateConnection(t, 404)
	if ok, err := c.DeleteAggregate(context.Background(), "7"); err != nil || ok {
		t.Fatalf("404 delete: %v %v", ok, err)
	}
}

func TestPyCloudComputeDeleteAggregateByNameAndFailures(t *testing.T) {
	c, log := pyCloudComputeAggregateConnection(t, 200)
	if ok, err := c.DeleteAggregate(context.Background(), "web"); err != nil || !ok {
		t.Fatalf("delete by name: %v %v", ok, err)
	}
	if want := "GET /os-aggregates/web|GET /os-aggregates|DELETE /os-aggregates/7"; log.String() != want {
		t.Fatalf("requests: %s", log.String())
	}
	if ok, err := c.DeleteAggregate(context.Background(), "ghost"); err != nil || ok {
		t.Fatalf("missing: %v %v", ok, err)
	}
	if strings.Count(log.String(), "DELETE") != 1 {
		t.Fatal("DELETE sent for a missing aggregate")
	}
	c, _ = pyCloudComputeAggregateConnection(t, 500)
	if ok, err := c.DeleteAggregate(context.Background(), "7"); err == nil || ok {
		t.Fatalf("500 must be an error: %v %v", ok, err)
	}
	if _, err := c.DeleteAggregate(context.Background(), ""); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.DeleteAggregate(ctx, "7"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
