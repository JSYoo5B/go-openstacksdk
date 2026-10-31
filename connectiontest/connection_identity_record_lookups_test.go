package connectiontest_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionIdentityRecordLookupsUseCachedAPIAndLocationBeforeSelection(t *testing.T) {
	cloud := testcloud.New(t)
	locates := 0
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates++
		return cloud.Server.URL + "/identity/v3/", nil
	}
	configured := "configured"
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithCloudLocation(resource.CloudLocation{Cloud: &configured}))
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("GET /identity/v3/services", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"services":[{"id":"s","name":"compute","location":{"cloud":"forged"}}]}`)
	})
	cloud.Mux.HandleFunc("GET /identity/v3/services/s", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"service":{"id":"s","name":"compute","location":{"cloud":"forged"}}}`)
	})
	cloud.Mux.HandleFunc("GET /identity/v3/endpoints", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"endpoints":[{"id":"e","name":"public","location":{"cloud":"forged"}}]}`)
	})
	cloud.Mux.HandleFunc("GET /identity/v3/endpoints/e", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"endpoint":{"id":"e","name":"public","location":{"cloud":"forged"}}}`)
	})
	filter := json.RawMessage(`{"location":{"cloud":"configured"}}`)
	services, err := conn.SearchServiceRecords(context.Background(), "compute", identity.WithServiceReadFilters(filter))
	if err != nil || services == nil || len(services.Services) != 1 {
		t.Fatal(services, err)
	}
	endpoints, err := conn.SearchEndpointRecords(context.Background(), "public", identity.WithEndpointReadFilters(filter))
	if err != nil || endpoints == nil || len(endpoints.Endpoints) != 1 {
		t.Fatal(endpoints, err)
	}
	if _, err := conn.GetServiceRecord(context.Background(), "s"); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.GetEndpointRecord(context.Background(), "e"); err != nil {
		t.Fatal(err)
	}
	if locates != 1 || string(services.Services[0].Wire.Body["location"]) != `{"cloud":"forged"}` {
		t.Fatal(locates, services)
	}
}

func TestConnectionIdentityRecordLookupsPreflightAndExplicitLocation(t *testing.T) {
	cloud := testcloud.New(t)
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Identity, cloud.Server.URL+"/identity/v3/"))
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("GET /identity/v3/services", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"services":[{"id":"s","name":"compute"}]}`)
	})
	explicit := "explicit"
	option := identity.WithServiceReadLocation(resource.CloudLocation{Cloud: &explicit})
	explicit = "changed"
	services, err := conn.SearchServiceRecords(context.Background(), "", option, identity.WithServiceReadFilters(json.RawMessage(`{"location":{"cloud":"explicit"}}`)))
	if err != nil || services == nil || len(services.Services) != 1 {
		t.Fatal(services, err)
	}
	if _, err := conn.GetEndpointRecord(nil, "e"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("cancel before lookup")
	cancel(cause)
	if _, err := conn.SearchServiceRecords(ctx, ""); !errors.Is(err, cause) {
		t.Fatal(err)
	}
}
