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

func TestConnectionProjectReadsShareCachedIdentityAndInjectLocationBeforeSelection(t *testing.T) {
	fixture := testcloud.New(t)
	locates, calls, callbacks := 0, 0, 0
	fixture.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates++
		return fixture.Server.URL + "/identity/v3/", nil
	}
	cloudName, projectName := "configured", "current"
	connection, err := sdk.FromProvider(fixture.Provider, sdk.WithCloudLocation(resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &projectName}}))
	if err != nil {
		t.Fatal(err)
	}
	service, err := connection.IdentityV3(context.Background())
	if err != nil || service.Projects.RawClient() != service.RawClient() {
		t.Fatal(service, err)
	}
	fixture.Mux.HandleFunc("GET /identity/v3/projects", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("name") != "" {
			t.Fatal(r.URL)
		}
		testcloud.JSON(w, 200, `{"projects":[{"id":"a","name":"team","location":{"cloud":"wire"}}]}`)
	})
	result, err := connection.SearchProjects(context.Background(), "team", identity.WithProjectReadFilters(json.RawMessage(`{"location":{"cloud":"configured","project":{"id":"scope","name":"current"}}}`)), func(cfg *identity.ProjectReadOpts) error { callbacks++; cfg.Location = nil; return nil })
	if err != nil || result == nil || len(result.Projects) != 1 || len(result.Inventory) != 1 || locates != 1 || calls != 1 || callbacks != 1 || string(result.Projects[0].Wire.Body["location"]) != `{"cloud":"wire"}` {
		t.Fatal(result, err, locates, calls, callbacks)
	}
	listed, err := connection.ListProjects(context.Background(), identity.WithProjectReadOpts(identity.ProjectReadOpts{}))
	if err != nil || listed == nil || len(listed.Projects) != 1 || locates != 1 || calls != 2 {
		t.Fatal(listed, err, locates, calls)
	}
	fields := connectionSearchFields(t, listed.Projects[0].Resource.Body["location"])
	if string(fields["cloud"]) != `"configured"` {
		t.Fatal(fields)
	}
}
func TestConnectionProjectReadsPerCallLocationSnapshotAndNoNamePushdownForSearch(t *testing.T) {
	fixture := testcloud.New(t)
	connection, err := sdk.FromProvider(fixture.Provider, sdk.WithEndpoint(sdk.Identity, fixture.Server.URL+"/identity/v3/"))
	if err != nil {
		t.Fatal(err)
	}
	fixture.Mux.HandleFunc("GET /identity/v3/projects", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("domain_id") != "domain" || r.URL.Query().Get("name") != "" {
			t.Fatal(r.URL)
		}
		testcloud.JSON(w, 200, `{"projects":[{"id":"a","name":"team-one"},{"id":"b","name":"other"}]}`)
	})
	perCall := "owned-per-call"
	option := identity.WithProjectReadLocation(resource.CloudLocation{Cloud: &perCall})
	perCall = "changed"
	callbacks := 0
	result, err := connection.SearchProjects(context.Background(), "team-*", identity.WithProjectReadDomainID("domain"), identity.WithProjectReadNameOrID("ignored-list-name"), option, func(*identity.ProjectReadOpts) error { callbacks++; return nil })
	if err != nil || result == nil || len(result.Projects) != 1 || len(result.Inventory) != 2 || callbacks != 1 {
		t.Fatal(result, err, callbacks)
	}
	fields := connectionSearchFields(t, result.Projects[0].Resource.Body["location"])
	if string(fields["cloud"]) != `"owned-per-call"` {
		t.Fatal(fields)
	}
}
func TestConnectionProjectReadsPreflightAndMissingEndpointPreserveRealConfigurationCause(t *testing.T) {
	fixture := testcloud.New(t)
	locates := 0
	fixture.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates++
		return "", gophercloud.ErrEndpointNotFound{}
	}
	connection, err := sdk.FromProvider(fixture.Provider)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"nil context", "nil connection", "cancel"} {
		ctx := context.Background()
		current := connection
		want := resource.ErrInvalidOption
		cause := errors.New("project caller canceled")
		if mode == "nil context" {
			ctx = nil
		}
		if mode == "nil connection" {
			current = nil
		}
		if mode == "cancel" {
			next, cancel := context.WithCancelCause(ctx)
			cancel(cause)
			ctx = next
			want = context.Canceled
		}
		result, err := current.SearchProjects(ctx, "")
		var proof *resource.ResponseError
		if result != nil || !errors.Is(err, want) || errors.As(err, &proof) || locates != 0 {
			t.Fatal(mode, result, err, proof, locates)
		}
		if mode == "cancel" && !errors.Is(err, cause) {
			t.Fatal(err)
		}
	}
	result, err := connection.ListProjects(context.Background())
	var missing gophercloud.ErrEndpointNotFound
	var proof *resource.ResponseError
	if result != nil || !errors.As(err, &missing) || errors.As(err, &proof) || locates != 1 {
		t.Fatal(result, err, proof, locates)
	}
}
