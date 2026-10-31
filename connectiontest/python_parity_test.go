package connectiontest_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

func pythonConnectionClearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"OS_CLOUD", "OS_CLIENT_CONFIG_FILE", "OS_REGION_NAME", "OS_INTERFACE", "OS_PROJECT_ID", "OS_PROJECT_NAME", "OS_TENANT_ID", "OS_TENANT_NAME", "OS_DOMAIN_ID", "OS_DOMAIN_NAME", "OS_TOKEN", "OS_APPLICATION_CREDENTIAL_ID", "OS_APPLICATION_CREDENTIAL_NAME", "OS_APPLICATION_CREDENTIAL_SECRET", "OS_SYSTEM_SCOPE"} {
		t.Setenv(key, "")
	}
}

// pythonConnectionLocation connects through clouds.yaml and returns the connection and auth request count.
func pythonConnectionLocation(t *testing.T, project string) (*sdk.Connection, *atomic.Int32) {
	t.Helper()
	pythonConnectionClearEnv(t)
	cloud := testcloud.New(t)
	path := filepath.Join(t.TempDir(), "clouds.yaml")
	config := fmt.Sprintf("clouds:\n  example:\n    region_name: RegionOne\n    auth:\n      auth_url: %s/v3\n      username: user\n      password: password\n      user_domain_name: Default\n      project_name: demo\n      project_domain_id: default\n", cloud.Server.URL)
	if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != "POST" || r.URL.Path != "/v3/auth/tokens" {
			t.Error("unexpected request", r.Method, r.URL)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("X-Subject-Token", "token")
		token := map[string]any{"expires_at": "2099-01-01T00:00:00Z", "catalog": []any{}, "user": map[string]any{"id": "user-id"}}
		if project != "" {
			token["project"] = map[string]any{"id": project, "name": "token-name", "domain": map[string]any{"id": "default", "name": "Default"}}
		}
		body, _ := json.Marshal(map[string]any{"token": token})
		testcloud.JSON(w, 201, string(body))
	})
	conn, err := sdk.Connect(context.Background(), sdk.WithCloud("example"), sdk.WithCloudFiles(path))
	if err != nil {
		t.Fatal(err)
	}
	return conn, &requests
}

func TestPythonConnectionCurrentLocationProjectAndName(t *testing.T) {
	conn, requests := pythonConnectionLocation(t, "project-id")
	location, err := conn.CurrentLocation()
	if err != nil {
		t.Fatal(err)
	}
	// get_name and current_location.cloud
	if location.Cloud == nil || *location.Cloud != "example" {
		t.Fatal(location.Cloud)
	}
	if location.RegionName == nil || *location.RegionName != "RegionOne" || location.Zone != nil {
		t.Fatal(location.RegionName, string(location.Zone))
	}
	// current_project and current_project_id: id from the token, names from configuration.
	project := location.Project
	if string(project.ID) != `"project-id"` || project.Name == nil || *project.Name != "demo" || project.DomainID == nil || *project.DomainID != "default" || project.DomainName != nil {
		t.Fatalf("%s %+v", project.ID, project)
	}
	encoded, err := json.Marshal(location)
	if err != nil || string(encoded) != `{"cloud":"example","region_name":"RegionOne","zone":null,"project":{"id":"project-id","name":"demo","domain_id":"default","domain_name":null}}` {
		t.Fatal(string(encoded), err)
	}
	if requests.Load() != 1 {
		t.Fatal("location made a request", requests.Load())
	}
}

func TestPythonConnectionCurrentProjectIDUnscopedIsNull(t *testing.T) {
	conn, requests := pythonConnectionLocation(t, "")
	location, err := conn.CurrentLocation()
	if err != nil || location.Project.ID != nil || requests.Load() != 1 {
		t.Fatal(string(location.Project.ID), err, requests.Load())
	}
}
