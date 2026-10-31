package connectiontest_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionIdentityRecordReadsUseCachedServiceAndLocationBeforeSelection(t *testing.T) {
	cloud := testcloud.New(t)
	locates := 0
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates++
		return cloud.Server.URL + "/identity/v3/", nil
	}
	name := "configured"
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithCloudLocation(resource.CloudLocation{Cloud: &name, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`)}}))
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("GET /identity/v3/users", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"users":[{"id":"u","name":"alice","default_project_id":"other","location":{"cloud":"wire"}}]}`)
	})
	cloud.Mux.HandleFunc("GET /identity/v3/roles", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"roles":[{"id":"r","name":"reader","location":{"cloud":"wire"}}]}`)
	})
	cloud.Mux.HandleFunc("GET /identity/v3/domains", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"domains":[{"id":"d","name":"example","location":{"cloud":"wire"}}]}`)
	})
	filter := json.RawMessage(`{"location":{"cloud":"configured","project":{"id":"scope"}}}`)
	users, err := conn.SearchUserRecords(context.Background(), "alice", identity.WithUserSearchFilters(filter))
	if err != nil || users == nil || len(users.Users) != 1 {
		t.Fatal(users, err)
	}
	role, err := conn.SearchRoleRecords(context.Background(), "reader", identity.WithRoleSearchFilters(filter))
	if err != nil || role == nil || len(role.Roles) != 1 {
		t.Fatal(role, err)
	}
	domain, err := conn.SearchDomainRecords(context.Background(), identity.WithDomainReadNameOrID("example"), identity.WithDomainReadFilters(filter))
	if err != nil || domain == nil || len(domain.Domains) != 1 {
		t.Fatal(domain, err)
	}
	if string(users.Users[0].Wire.Body["location"]) != `{"cloud":"wire"}` {
		t.Fatal(users.Users[0].Wire)
	}
	if _, err := conn.ListUserRecords(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ListRoleRecords(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ListDomainRecords(context.Background()); err != nil {
		t.Fatal(err)
	}
	if locates != 1 {
		t.Fatal(locates)
	}
}

func TestConnectionIdentityRecordReadsRespectExplicitLocationAndPreflight(t *testing.T) {
	cloud := testcloud.New(t)
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Identity, cloud.Server.URL+"/identity/v3/"))
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("GET /identity/v3/roles", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"roles":[{"id":"r","name":"reader"}]}`)
	})
	explicit := "explicit"
	option := roles.WithRecordListLocation(resource.CloudLocation{Cloud: &explicit})
	explicit = "changed"
	result, err := conn.ListRoleRecords(context.Background(), option)
	if err != nil || result == nil || len(result.Roles) != 1 {
		t.Fatal(result, err)
	}
	var location map[string]json.RawMessage
	if err := json.Unmarshal(result.Roles[0].Resource.Body["location"], &location); err != nil || string(location["cloud"]) != `"explicit"` {
		t.Fatal(location, err)
	}
	if _, err := conn.ListUserRecords(nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("cancel before identity bridge")
	cancel(cause)
	if _, err := conn.ListDomainRecords(ctx); !errors.Is(err, cause) {
		t.Fatal(err)
	}
}
