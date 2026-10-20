package identity_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func projectCloudFixture(t *testing.T, reply func(http.ResponseWriter, *http.Request)) (*identity.ProjectReadCloud, *testcloud.Cloud) {
	t.Helper()
	fixture := testcloud.New(t)
	fixture.Mux.HandleFunc("GET /identity/v3/projects", reply)
	api := projects.New(fixture.Client("identity", "/identity/v3/"))
	return identity.NewProjectReadCloud(api), fixture
}
func TestPythonProjectCloudListTruthfulNameDomainOverrideAndLocalBodyFilter(t *testing.T) {
	calls := 0
	var query url.Values
	cloud, _ := projectCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		query = r.URL.Query()
		body, readErr := io.ReadAll(r.Body)
		if r.Header.Get("Accept") != "application/json" || readErr != nil || len(body) != 0 || r.ContentLength != 0 {
			t.Fatal(r.Header, string(body), readErr, r.ContentLength)
		}
		w.Header().Set("X-Request-Id", "cloud-project")
		testcloud.JSON(w, 203, `{"projects":[{"id":"a","name":"override-name","description":"keep","enabled":true},{"id":"b","description":"skip"}]}`)
	})
	result, err := cloud.ListProjects(context.Background(), identity.WithProjectReadDomainID("override-domain"), identity.WithProjectReadNameOrID("override-name"), identity.WithProjectReadFilters(json.RawMessage(`{"domain_id":"filter-domain","name":"filter-name","description":"keep","is_enabled":false}`)))
	if err != nil || result == nil || len(result.Projects) != 1 || len(result.Inventory) != 1 || calls != 1 {
		t.Fatal(result, err, calls)
	}
	want := url.Values{"domain_id": {"override-domain"}, "name": {"override-name"}, "enabled": {"False"}}
	if !reflect.DeepEqual(query, want) {
		t.Fatal(query, want)
	}
	if result.Projects[0].StatusCode != 203 || result.Projects[0].Header.Get("X-Request-Id") != "cloud-project" || result.JSON == nil {
		t.Fatal(result)
	}
}
func TestPythonProjectCloudFalseyFiltersAndTruthyNonMappingRejectBeforeHTTP(t *testing.T) {
	calls := 0
	cloud, _ := projectCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.RawQuery != "" {
			t.Fatal(r.URL)
		}
		testcloud.JSON(w, 200, `{"projects":[]}`)
	})
	for _, raw := range []string{`null`, `false`, `0`, `1e-4000`, `""`, `[]`, `{}`} {
		result, err := cloud.ListProjects(context.Background(), identity.WithProjectReadFilters(json.RawMessage(raw)))
		if err != nil || result == nil || string(result.JSON) != "[]" || result.Projects == nil {
			t.Fatal(raw, result, err)
		}
	}
	before := calls
	for _, raw := range []string{`"length(@)"`, `true`, `1`, `[1]`, `{bad`} {
		for _, search := range []bool{false, true} {
			var err error
			if search {
				_, err = cloud.SearchProjects(context.Background(), "", identity.WithProjectReadFilters(json.RawMessage(raw)))
			} else {
				_, err = cloud.ListProjects(context.Background(), identity.WithProjectReadFilters(json.RawMessage(raw)))
			}
			var proof *resource.ResponseError
			if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || calls != before {
				t.Fatal(raw, search, err, proof, calls)
			}
		}
	}
}
func TestPythonProjectCloudSearchGlobNestedLoadedEqualityAndWireAliasFailures(t *testing.T) {
	calls := 0
	cloud, _ := projectCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("name") != "" || r.URL.Query().Get("domain_id") != "domain" {
			t.Fatal(r.URL)
		}
		testcloud.JSON(w, 200, `{"projects":[{"id":"a","name":"team-alpha","options":{"n":0.10000000000000001}},{"id":"b","name":"other","options":{"n":0.2}}]}`)
	})
	result, err := cloud.SearchProjects(context.Background(), "team-*", identity.WithProjectReadDomainID("domain"), identity.WithProjectReadFilters(json.RawMessage(`{"options":{"n":0.1}}`)))
	if err != nil || result == nil || len(result.Projects) != 1 || len(result.Inventory) != 1 || string(result.Projects[0].Resource.Body["id"]) != `"a"` || calls != 1 {
		t.Fatal(result, err, calls)
	}
	cloud, _ = projectCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"projects":[{"id":"a","enabled":true}]}`)
	})
	result, err = cloud.SearchProjects(context.Background(), "", identity.WithProjectReadFilters(json.RawMessage(`{"enabled":true}`)))
	if !errors.Is(err, resource.ErrInvalidOption) || result == nil || len(result.Inventory) != 1 || result.Inventory[0].StatusCode != 200 {
		t.Fatal(result, err)
	}
}
func TestPythonProjectCloudProxyLiteralAndOpaqueListResultsDoNotInventHTTP(t *testing.T) {
	calls := 0
	cloud, _ := projectCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		testcloud.JSON(w, 200, `{"projects":[{"id":"a"}]}`)
	})
	for _, test := range []struct{ filter, want string }{{`{"jmespath_filters":"\u0060[]\u0060"}`, `[]`}, {`{"jmespath_filters":"\u0060[1,false,null]\u0060"}`, `[1,false,null]`}, {`{"jmespath_filters":"\u0060\"ab\"\u0060"}`, `["a","b"]`}, {`{"jmespath_filters":"\u0060{\"z\":1,\"a\":2}\u0060"}`, `["a","z"]`}} {
		result, err := cloud.ListProjects(context.Background(), identity.WithProjectReadFilters(json.RawMessage(test.filter)))
		if err != nil || result == nil || string(result.JSON) != test.want || result.Projects != nil || len(result.Inventory) != 0 || calls != 0 {
			t.Fatal(test, result, err, calls)
		}
	}
	result, err := cloud.ListProjects(context.Background(), identity.WithProjectReadFilters(json.RawMessage(`{"jmespath_filters":"to_array(@)"}`)))
	if err != nil || result == nil || result.JSON != nil || len(result.Value) != 1 || len(result.Inventory) != 0 || calls != 0 {
		t.Fatal(result, err, calls)
	}
	if _, ok := result.Value[0].(*projects.RecordListGenerator); !ok {
		t.Fatal(result.Value)
	}
	for _, expr := range []string{"[0].id", "length(@)"} {
		raw, _ := json.Marshal(map[string]string{"jmespath_filters": expr})
		_, err = cloud.ListProjects(context.Background(), identity.WithProjectReadFilters(raw))
		if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatal(expr, err, calls)
		}
	}
	searched, err := cloud.SearchProjects(context.Background(), "", identity.WithProjectReadFilters(json.RawMessage(`{"jmespath_filters":"\u0060[]\u0060"}`)))
	if err != nil || searched == nil || string(searched.Value) != "[]" || searched.Inventory == nil || calls != 0 {
		t.Fatal(searched, err, calls)
	}
}
func TestPythonProjectCloudLocationBeforeLocalSelectionAndOwnedOverrides(t *testing.T) {
	cloudName, projectName := "configured", "current"
	location := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &projectName}}
	option := identity.WithProjectReadLocation(location)
	cloudName = "caller changed"
	projectName = "caller changed"
	cloud, _ := projectCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"projects":[{"id":"a","location":{"cloud":"wire"}},{"id":"b","project_id":"foreign"}]}`)
	})
	result, err := cloud.SearchProjects(context.Background(), "", option, identity.WithProjectReadFilters(json.RawMessage(`{"location":{"cloud":"configured","project":{"id":"scope","name":"current"}}}`)))
	if err != nil || result == nil || len(result.Projects) != 1 || len(result.Inventory) != 2 || string(result.Projects[0].Wire.Body["location"]) != `{"cloud":"wire"}` {
		t.Fatal(result, err)
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(result.Inventory[1].Resource.Body["location"], &fields)
	var project map[string]json.RawMessage
	_ = json.Unmarshal(fields["project"], &project)
	if string(project["id"]) != `"foreign"` || string(project["name"]) != "null" {
		t.Fatal(project)
	}
}
func TestPythonProjectCloudSourceHeaderSnapshotCallbackCancellationAndLatePageEvidence(t *testing.T) {
	calls := 0
	cloud, fixture := projectCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("X-Snapshot") != "before" || r.Header.Get("X-Late") != "" {
			t.Fatal(r.Header)
		}
		if calls == 1 {
			testcloud.JSON(w, 200, `{"projects":[{"id":"a"}],"next":"?marker=a"}`)
			return
		}
		w.Header().Set("X-Request-Id", "actual-failure")
		testcloud.JSON(w, 503, `{"error":"actual"}`)
	})
	cloud.RawAPI().RawClient().MoreHeaders = map[string]string{"X-Snapshot": "before"}
	callbacks := 0
	result, err := cloud.ListProjects(context.Background(), func(*identity.ProjectReadOpts) error {
		callbacks++
		cloud.RawAPI().RawClient().MoreHeaders["X-Snapshot"] = "after"
		cloud.RawAPI().RawClient().MoreHeaders["X-Late"] = "injected"
		return nil
	})
	var proof gophercloud.ErrUnexpectedResponseCode
	if err == nil || result == nil || len(result.Inventory) != 1 || !errors.As(err, &proof) || proof.Actual != 503 || proof.ResponseHeader.Get("X-Request-Id") != "actual-failure" || string(proof.Body) != `{"error":"actual"}` || proof.Method != http.MethodGet || !strings.Contains(proof.URL, "/identity/v3/projects?marker=a") || calls != 2 || callbacks != 1 {
		t.Fatal(result, err, proof, calls, callbacks)
	}
	cause := errors.New("project callback cancellation")
	ctx, cancel := context.WithCancelCause(context.Background())
	before := calls
	_, err = cloud.ListProjects(ctx, func(*identity.ProjectReadOpts) error { cancel(cause); return nil })
	if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || calls != before {
		t.Fatal(err, calls)
	}
	_ = fixture
}
