package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionSwiftContainerSearchSharesFacadeAndAutomaticLocationBeforeSelection(t *testing.T) {
	fixture := testcloud.New(t)
	cloudName, projectName := "configured-swift", "configured-project"
	configured := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &projectName}}
	locates, requests, callbacks := 0, 0, 0
	fixture.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates++
		return fixture.Server.URL + "/swift/v1/AUTH_p/", nil
	}
	conn, err := sdk.FromProvider(fixture.Provider, sdk.WithCloudLocation(configured))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.ObjectStorageV1(context.Background())
	if err != nil || service.Cloud == nil || service.Cloud.RawClient() != service.RawClient() || service.Containers.RawClient() != service.RawClient() {
		t.Fatal(service, err)
	}
	fixture.Mux.HandleFunc("GET /swift/v1/AUTH_p/", func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.RawQuery != "" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 203, `[{"name":"alpha","location":{"cloud":"forged"}}]`)
	})
	filter := objectstorage.WithContainerSearchFiltersJSON(json.RawMessage(`{"location":{"cloud":"configured-swift","project":{"id":"scope","name":"configured-project"}}}`))
	result, err := conn.SearchContainers(context.Background(), filter, func(cfg *objectstorage.ContainerSearchOpts) error { callbacks++; cfg.Location = nil; return nil })
	if err != nil || result == nil || len(result.Records) != 1 || len(result.AllRecords) != 1 || requests != 1 || locates != 1 || callbacks != 1 || result.Records[0].StatusCode != 203 {
		t.Fatal(result, err, requests, locates, callbacks)
	}
	if string(result.Records[0].Wire.Body["location"]) != `{"cloud":"forged"}` {
		t.Fatal(result.Records[0].Wire.Body)
	}
	cached, err := conn.ObjectStorage(context.Background())
	if err != nil || cached != service {
		t.Fatal(cached, err)
	}
	result, err = conn.SearchContainers(context.Background(), objectstorage.WithContainerSearchOpts(objectstorage.ContainerSearchOpts{}), objectstorage.WithContainerSearchExpression("[0].location.cloud"))
	if err != nil || result == nil || string(result.Value) != `"configured-swift"` || !result.Expression || result.Records != nil || requests != 2 || locates != 1 {
		t.Fatal(result, err, requests, locates)
	}
}

func TestConnectionSwiftContainerSearchPerCallLocationRemainsOwnedAndCallbackRunsOnce(t *testing.T) {
	fixture := testcloud.New(t)
	conn, err := sdk.FromProvider(fixture.Provider, sdk.WithEndpoint(sdk.ObjectStorage, fixture.Server.URL+"/swift/v1/AUTH_p/"))
	if err != nil {
		t.Fatal(err)
	}
	fixture.Mux.HandleFunc("GET /swift/v1/AUTH_p/", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `[{"name":"alpha"}]`) })
	cloudName := "per-call"
	location := resource.CloudLocation{Cloud: &cloudName}
	option := objectstorage.WithContainerSearchLocation(location)
	cloudName = "caller changed"
	callbacks := 0
	result, err := conn.SearchContainers(context.Background(), option, func(*objectstorage.ContainerSearchOpts) error { callbacks++; return nil })
	if err != nil || result == nil || len(result.Records) != 1 || callbacks != 1 {
		t.Fatal(result, err, callbacks)
	}
	fields := connectionSearchFields(t, result.Records[0].Resource.Body["location"])
	if string(fields["cloud"]) != `"per-call"` {
		t.Fatal(fields)
	}
}

func TestConnectionSwiftContainerSearchPreflightDoesNotDiscoverOrFabricateResponse(t *testing.T) {
	fixture := testcloud.New(t)
	locates := 0
	fixture.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates++
		return "", gophercloud.ErrEndpointNotFound{}
	}
	conn, err := sdk.FromProvider(fixture.Provider)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"nil context", "nil connection", "canceled"} {
		current := conn
		ctx := context.Background()
		want := resource.ErrInvalidOption
		cause := errors.New("swift caller cancel")
		if mode == "nil context" {
			ctx = nil
		}
		if mode == "nil connection" {
			current = nil
		}
		if mode == "canceled" {
			next, cancel := context.WithCancelCause(ctx)
			cancel(cause)
			ctx = next
			want = context.Canceled
		}
		result, err := current.SearchContainers(ctx)
		var proof *resource.ResponseError
		if result != nil || !errors.Is(err, want) || errors.As(err, &proof) || locates != 0 {
			t.Fatal(mode, result, err, proof, locates)
		}
		if mode == "canceled" && !errors.Is(err, cause) {
			t.Fatal(err)
		}
	}
	result, err := conn.SearchContainers(context.Background())
	var missing gophercloud.ErrEndpointNotFound
	var proof *resource.ResponseError
	if result != nil || !errors.As(err, &missing) || errors.As(err, &proof) || locates != 1 {
		t.Fatal(result, err, proof, locates)
	}
}
