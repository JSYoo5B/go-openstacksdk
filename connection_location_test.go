package openstack_test

import (
	"context"
	"encoding/json"
	"fmt"
	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestConnectVolumeSearchCapturesAutomaticCloudRegionConfiguredProjectAndForeignOwner(t *testing.T) {
	for _, source := range []string{"explicit", "cloud", "environment"} {
		t.Run(source, func(t *testing.T) {
			cloud := testcloud.New(t)
			for _, key := range []string{"OS_CLOUD", "OS_CLIENT_CONFIG_FILE", "OS_REGION_NAME", "OS_INTERFACE", "OS_AUTH_URL", "OS_USERNAME", "OS_PASSWORD", "OS_DOMAIN_ID", "OS_DOMAIN_NAME", "OS_USER_DOMAIN_ID", "OS_USER_DOMAIN_NAME", "OS_PROJECT_ID", "OS_PROJECT_NAME", "OS_TENANT_ID", "OS_TENANT_NAME", "OS_APPLICATION_CREDENTIAL_ID", "OS_APPLICATION_CREDENTIAL_NAME", "OS_APPLICATION_CREDENTIAL_SECRET", "OS_SYSTEM_SCOPE", "OS_TOKEN"} {
				t.Setenv(key, "")
			}
			var authCalls, listCalls atomic.Int32
			cloud.Mux.HandleFunc("POST /v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
				authCalls.Add(1)
				w.Header().Set("X-Subject-Token", "authenticated")
				testcloud.JSON(w, 201, `{"token":{"expires_at":"2099-01-01T00:00:00Z","catalog":[],"project":{"id":"scope","name":"token-name","domain":{"id":"token-domain","name":"token-domain-name"}}}}`)
			})
			cloud.Mux.HandleFunc("GET /auto-location/v3/p/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
				listCalls.Add(1)
				if r.Header.Get("X-Auth-Token") != "authenticated" {
					t.Error(r.Header)
				}
				testcloud.JSON(w, 200, `{"volumes":[{"id":"same","project_id":"scope","availability_zone":false,"location":{"cloud":"wire"}},{"id":"foreign","os-vol-tenant-attr:tenant_id":"elsewhere","availability_zone":[1]}]}`)
			})
			opts := []sdk.ConnectionOption{sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/auto-location/v3/p/"), sdk.WithRegion("")}
			wantCloud := "127.0.0.1"
			wantDomain := "ProjectDomain"
			switch source {
			case "explicit":
				opts = append(opts, sdk.WithAuth(gophercloud.AuthOptions{IdentityEndpoint: cloud.Server.URL + "/v3", Username: "user", Password: "password", DomainName: "UserDomain", TenantName: "Configured", Scope: &gophercloud.AuthScope{ProjectName: "Configured", DomainName: "ProjectDomain"}}))
			case "cloud":
				wantCloud = "dev"
				path := filepath.Join(t.TempDir(), "clouds.yaml")
				body := fmt.Sprintf("clouds:\n  dev:\n    region_name: configured-region\n    auth:\n      auth_url: %s/v3\n      username: user\n      password: password\n      user_domain_name: UserDomain\n      project_name: Configured\n      project_domain_name: ProjectDomain\n", cloud.Server.URL)
				if err := os.WriteFile(path, []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
				t.Setenv("OS_CLOUD", "dev")
				opts = append(opts, sdk.WithCloudFiles(path))
			case "environment":
				wantDomain = "UserDomain"
				t.Setenv("OS_AUTH_URL", cloud.Server.URL+"/v3")
				t.Setenv("OS_USERNAME", "user")
				t.Setenv("OS_PASSWORD", "password")
				t.Setenv("OS_DOMAIN_NAME", "UserDomain")
				t.Setenv("OS_PROJECT_NAME", "Configured")
			}
			conn, err := sdk.Connect(context.Background(), opts...)
			if err != nil {
				t.Fatal(err)
			}
			location, err := conn.CurrentLocation()
			if err != nil || location.Cloud == nil || *location.Cloud != wantCloud || location.RegionName == nil || *location.RegionName != "" || string(location.Project.ID) != `"scope"` || location.Project.Name == nil || *location.Project.Name != "Configured" || location.Project.DomainName == nil || *location.Project.DomainName != wantDomain {
				t.Fatal(location, err)
			}
			search, err := conn.SearchVolumes(context.Background(), blockstorage.SearchVolumesRequest{})
			if err != nil || search == nil || len(search.Volumes) != 2 || authCalls.Load() != 1 || listCalls.Load() != 1 {
				t.Fatal(search, err, authCalls.Load(), listCalls.Load())
			}
			var rows []map[string]json.RawMessage
			_ = json.Unmarshal(search.Value, &rows)
			same := connectionSearchFields(t, rows[0]["location"])
			project := connectionSearchFields(t, same["project"])
			if string(same["cloud"]) != `"`+wantCloud+`"` || string(same["zone"]) != "false" || string(project["name"]) != `"Configured"` || string(project["domain_name"]) != `"`+wantDomain+`"` {
				t.Fatal(string(search.Value))
			}
			foreign := connectionSearchFields(t, rows[1]["location"])
			project = connectionSearchFields(t, foreign["project"])
			if string(foreign["zone"]) != "[1]" || string(project["id"]) != `"elsewhere"` || string(project["name"]) != "null" || string(project["domain_id"]) != "null" || string(project["domain_name"]) != "null" {
				t.Fatal(string(search.Value))
			}
		})
	}
}
