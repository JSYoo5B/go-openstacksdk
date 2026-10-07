package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"fmt"
	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestConnectCloudLocationMappingIDScopeLosesNativeDomainFactsAndOwnedOverrideRestores(t *testing.T) {
	cloud := testcloud.New(t)
	for _, key := range []string{"OS_CLOUD", "OS_CLIENT_CONFIG_FILE", "OS_REGION_NAME", "OS_INTERFACE", "OS_PROJECT_ID", "OS_PROJECT_NAME", "OS_TENANT_ID", "OS_TENANT_NAME", "OS_DOMAIN_ID", "OS_DOMAIN_NAME", "OS_TOKEN", "OS_APPLICATION_CREDENTIAL_ID", "OS_APPLICATION_CREDENTIAL_NAME", "OS_APPLICATION_CREDENTIAL_SECRET", "OS_SYSTEM_SCOPE"} {
		t.Setenv(key, "")
	}
	path := filepath.Join(t.TempDir(), "clouds.yaml")
	config := fmt.Sprintf("clouds:\n  id-cloud:\n    region_name: configured-region\n    auth:\n      auth_url: %s/v3\n      username: user\n      password: password\n      user_domain_name: UserDomain\n      project_id: requested-scope\n      project_name: ConfiguredProject\n      project_domain_name: ConfiguredProjectDomain\n", cloud.Server.URL)
	if err := os.WriteFile(path, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	var authCalls atomic.Int32
	cloud.Mux.HandleFunc("POST /v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		authCalls.Add(1)
		var body struct {
			Auth struct {
				Scope struct {
					Project map[string]json.RawMessage `json:"project"`
				} `json:"scope"`
			} `json:"auth"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if string(body.Auth.Scope.Project["id"]) != `"requested-scope"` || len(body.Auth.Scope.Project) != 1 {
			t.Error("native clouds parser ID scope changed", body.Auth.Scope.Project)
		}
		w.Header().Set("X-Subject-Token", "recorded")
		testcloud.JSON(w, 201, `{"token":{"expires_at":"2099-01-01T00:00:00Z","catalog":[],"project":{"id":"actual-scope","name":"TokenProject","domain":{"id":"token-domain","name":"TokenDomain"}}}}`)
	})
	conn, err := sdk.Connect(context.Background(), sdk.WithCloud("id-cloud"), sdk.WithCloudFiles(path), sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/volume/v3/project/"))
	if err != nil {
		t.Fatal(err)
	}
	location, err := conn.CurrentLocation()
	if err != nil || location.Cloud == nil || *location.Cloud != "id-cloud" || location.RegionName == nil || *location.RegionName != "configured-region" || string(location.Project.ID) != `"actual-scope"` || location.Project.Name == nil || *location.Project.Name != "ConfiguredProject" || location.Project.DomainID != nil || location.Project.DomainName != nil || authCalls.Load() != 1 {
		t.Fatal("lost native config facts were guessed or actual scope ignored", location, err, authCalls.Load())
	}
	cinder, err := conn.BlockStorageV3(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var restored resource.CloudLocation = location.Clone()
	domain := "ConfiguredProjectDomain"
	restored.Project.DomainName = &domain
	option := sdk.WithCloudLocation(restored)
	domain = "caller mutated"
	restored.Project.ID[0] = '!'
	*restored.Project.Name = "caller mutated"
	adopted, err := sdk.FromProvider(cinder.RawClient().ProviderClient, option)
	if err != nil {
		t.Fatal(err)
	}
	fixed, err := adopted.CurrentLocation()
	if err != nil || fixed.Project.DomainName == nil || *fixed.Project.DomainName != "ConfiguredProjectDomain" || fixed.Project.Name == nil || *fixed.Project.Name != "ConfiguredProject" || string(fixed.Project.ID) != `"actual-scope"` || authCalls.Load() != 1 {
		t.Fatal("owned override failed to restore configured facts without auth I/O", fixed, err, authCalls.Load())
	}
	// CurrentLocation returns another owned copy of the complete override.
	*fixed.Project.DomainName = "result changed"
	fixed.Project.ID[0] = '!'
	again, err := adopted.CurrentLocation()
	if err != nil || again.Project.DomainName == nil || *again.Project.DomainName != "ConfiguredProjectDomain" || string(again.Project.ID) != `"actual-scope"` {
		t.Fatal(again, err)
	}
	// The original Connection still honestly exposes the native information loss.
	original, err := conn.CurrentLocation()
	if err != nil || original.Project.DomainName != nil || original.Project.Name == nil || *original.Project.Name != "ConfiguredProject" {
		t.Fatal(original, err)
	}
}
