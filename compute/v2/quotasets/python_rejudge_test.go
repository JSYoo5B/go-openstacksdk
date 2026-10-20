package quotasets_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pythonRejudgeQuotaRequest struct {
	method, path, query, body string
}

func pythonRejudgeQuotaScope(t *testing.T, reply string) (*quotasets.ProjectQuotaScope, *[]pythonRejudgeQuotaRequest) {
	t.Helper()
	cloud := testcloud.New(t)
	var seen []pythonRejudgeQuotaRequest
	cloud.Mux.HandleFunc("/compute/v2.1/", func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		seen = append(seen, pythonRejudgeQuotaRequest{method: r.Method, path: r.URL.Path, query: r.URL.RawQuery, body: string(raw)})
		testcloud.JSON(w, http.StatusOK, reply)
	})
	scope, err := quotasets.New(cloud.Client("compute", "/compute/v2.1/")).InProject(context.Background(), resource.ID("project-1"))
	if err != nil {
		t.Fatal(err)
	}
	return scope, &seen
}

// Python get_quota_set_defaults fetches /os-quota-sets/{project}/defaults
// without a query and exposes the limits from the quota_set envelope.
func TestPythonRejudgeComputeQuotaDefaultsRequest(t *testing.T) {
	scope, seen := pythonRejudgeQuotaScope(t, `{"quota_set":{"id":"project-1","cores":20,"instances":10,"ram":51200,"key_pairs":100,"server_groups":10,"server_group_members":10,"metadata_items":128,"injected_files":5,"injected_file_content_bytes":10240,"injected_file_path_bytes":255}}`)
	value, err := scope.Defaults(context.Background())
	if err != nil || value.Cores != 20 || value.Instances != 10 || value.RAM != 51200 || value.ProjectID != "project-1" {
		t.Fatal(value, err)
	}
	if len(*seen) != 1 || (*seen)[0] != (pythonRejudgeQuotaRequest{method: "GET", path: "/compute/v2.1/os-quota-sets/project-1/defaults"}) {
		t.Fatalf("%#v", *seen)
	}
}

// Python update_quota_set(project, **attrs) without user commits only the
// supplied attributes in a quota_set PUT with no query string.
func TestPythonRejudgeComputeQuotaUpdateRequest(t *testing.T) {
	cores, ram := 40, -1
	cases := []struct {
		name string
		opts quotasets.UpdateOpts
		body string
	}{
		{"supplied limits", quotasets.UpdateOpts{Cores: &cores, RAM: &ram}, `{"quota_set":{"cores":40,"ram":-1}}`},
		// Python skips HTTP when no attribute is dirty; Go sends an empty
		// quota_set and Nova answers with the unchanged limits.
		{"no attributes", quotasets.UpdateOpts{}, `{"quota_set":{}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scope, seen := pythonRejudgeQuotaScope(t, `{"quota_set":{"cores":40,"instances":10,"ram":-1}}`)
			value, err := scope.Update(context.Background(), tc.opts)
			if err != nil || value.Cores != 40 || value.RAM != -1 {
				t.Fatal(value, err)
			}
			if len(*seen) != 1 || (*seen)[0] != (pythonRejudgeQuotaRequest{method: "PUT", path: "/compute/v2.1/os-quota-sets/project-1", body: tc.body}) {
				t.Fatalf("%#v", *seen)
			}
		})
	}
}
