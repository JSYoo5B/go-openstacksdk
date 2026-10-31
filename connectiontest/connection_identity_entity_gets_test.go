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

func TestConnectionIdentityEntityGetsUseCachedAPIAndOwnedLocation(t *testing.T) {
	cloud := testcloud.New(t)
	locates := 0
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) { locates++; return cloud.Server.URL + "/v3/", nil }
	configured := "configured"
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithCloudLocation(resource.CloudLocation{Cloud: &configured}))
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("GET /v3/users/u", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"user":{"id":"u","name":"alice","default_project_id":"foreign","location":{"cloud":"forged"}}}`)
	})
	cloud.Mux.HandleFunc("GET /v3/roles/r", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"role":{"id":"r","name":"reader","location":{"cloud":"forged"}}}`)
	})
	user, err := conn.GetUserRecord(context.Background(), "u")
	if err != nil || user == nil || user.User == nil {
		t.Fatal(user, err)
	}
	role, err := conn.GetRoleRecord(context.Background(), "r")
	if err != nil || role == nil || role.Role == nil || locates != 1 {
		t.Fatal(role, err, locates)
	}
	var location map[string]json.RawMessage
	if err := json.Unmarshal(user.User.Resource.Body["location"], &location); err != nil || string(location["cloud"]) != `"configured"` {
		t.Fatal(location, err)
	}
	if string(user.User.Wire.Body["location"]) != `{"cloud":"forged"}` || string(user.User.Wire.Body["default_project_id"]) != `"foreign"` {
		t.Fatal(user.User)
	}
	explicit := "explicit"
	option := identity.WithRoleGetLocation(resource.CloudLocation{Cloud: &explicit})
	explicit = "changed"
	role, err = conn.GetRoleRecord(context.Background(), "r", option)
	if err != nil || role == nil || role.Role == nil {
		t.Fatal(role, err)
	}
	if err := json.Unmarshal(role.Role.Resource.Body["location"], &location); err != nil || string(location["cloud"]) != `"explicit"` {
		t.Fatal(location, err)
	}
}

func TestConnectionIdentityEntityGetsPreserveNilContextAndCancellationCause(t *testing.T) {
	cloud := testcloud.New(t)
	locates := 0
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates++
		return "", gophercloud.ErrEndpointNotFound{}
	}
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.GetUserRecord(nil, "u"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("cancel before identity getter")
	cancel(cause)
	if _, err := conn.GetRoleRecord(ctx, "r"); !errors.Is(err, cause) || locates != 0 {
		t.Fatal(err, locates)
	}
}

func TestConnectionDomainProjectGroupGetsPreserveBranchAndLocationSelection(t *testing.T) {
	cloud := testcloud.New(t)
	locates := 0
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) { locates++; return cloud.Server.URL + "/v3/", nil }
	configured := "configured"
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithCloudLocation(resource.CloudLocation{Cloud: &configured}))
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	cloud.Mux.HandleFunc("GET /v3/domains/d", func(w http.ResponseWriter, r *http.Request) {
		calls++
		testcloud.JSON(w, 200, `{"domain":{"id":"d","name":"example","options":{"flag":true},"location":{"cloud":"forged"}}}`)
	})
	cloud.Mux.HandleFunc("GET /v3/projects", func(w http.ResponseWriter, r *http.Request) {
		calls++
		testcloud.JSON(w, 200, `{"projects":[{"id":"p","name":"project","location":{"cloud":"forged"}}]}`)
	})
	cloud.Mux.HandleFunc("GET /v3/groups", func(w http.ResponseWriter, r *http.Request) {
		calls++
		testcloud.JSON(w, 200, `{"groups":[{"id":"g","name":"group","location":{"cloud":"forged"}}]}`)
	})
	domain, err := conn.GetDomainRecord(context.Background(), identity.WithDomainGetID("d"), identity.WithDomainGetFilters(struct{ Ignore string }{"no serialization"}))
	if err != nil || domain == nil || domain.Record == nil {
		t.Fatal(domain, err)
	}
	var location map[string]json.RawMessage
	if err := json.Unmarshal(domain.Record.Resource.Body["location"], &location); err != nil || string(location["cloud"]) != `"configured"` {
		t.Fatal(location, err)
	}
	filters := json.RawMessage(`{"location":{"cloud":"configured"}}`)
	project, err := conn.GetProjectRecord(context.Background(), "project", identity.WithProjectGetFilters(filters))
	if err != nil || project == nil || project.Project == nil {
		t.Fatal(project, err)
	}
	group, err := conn.GetGroupRecord(context.Background(), "group", identity.WithGroupGetFilters(filters))
	if err != nil || group == nil || group.Group == nil || locates != 1 || calls != 3 {
		t.Fatal(group, err, locates, calls)
	}
	if string(domain.Record.Resource.Body["options"]) != `{"flag":true}` || string(project.Project.Wire.Body["location"]) != `{"cloud":"forged"}` {
		t.Fatal(domain, project)
	}
	if _, err := conn.GetDomainRecord(context.Background()); !errors.Is(err, resource.ErrInvalidOption) || calls != 3 {
		t.Fatal(err, calls)
	}
}
