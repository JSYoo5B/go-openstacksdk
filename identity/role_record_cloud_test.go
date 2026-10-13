package identity_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func roleCloudFixture(t *testing.T, reply func(http.ResponseWriter, *http.Request)) (*identity.RoleReadCloud, *testcloud.Cloud) {
	t.Helper()
	fixture := testcloud.New(t)
	fixture.Mux.HandleFunc("GET /identity/v3/roles", reply)
	return identity.NewRoleReadCloud(roles.New(fixture.Client("identity", "/identity/v3/"))), fixture
}
func roleCloudField(t *testing.T, row *roles.RoleRecord, key, want string) {
	t.Helper()
	if row == nil || row.Resource == nil || string(row.Resource.Body[key]) != want {
		t.Fatal(key, want, row)
	}
}
func TestPythonRoleCloudSearchDomainServerAndLocalGlobSubset(t *testing.T) {
	calls := 0
	cloud, _ := roleCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, readErr := io.ReadAll(r.Body)
		if r.URL.RawQuery != "domain_id=domain" || r.Header.Get("Accept") != "application/json" || readErr != nil || len(body) != 0 || r.ContentLength != 0 {
			t.Fatal(r.URL, r.Header, string(body), readErr, r.ContentLength)
		}
		w.Header().Set("X-Request-Id", "cloud-role")
		testcloud.JSON(w, 203, `{"roles":[{"id":"a","name":"ops-alpha","description":{"n":0.10000000000000001,"active":true}},{"id":"b","name":"other","description":{"n":0.2}},{"id":"a","name":"ops-beta","description":{"n":0.1,"extra":1}}]}`)
	})
	result, err := cloud.SearchRoles(context.Background(), "ops-*", identity.WithRoleSearchDomainID("domain"), identity.WithRoleSearchFilters(json.RawMessage(`{"description":{"n":0.1}}`)))
	if err != nil || result == nil || len(result.Roles) != 2 || len(result.Inventory) != 3 || result.Expression || calls != 1 {
		t.Fatal(result, err, calls)
	}
	for _, row := range result.Roles {
		roleCloudField(t, row, "id", `"a"`)
		if row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "cloud-role" {
			t.Fatal(row)
		}
	}
	result.Roles[0].Resource.Body["id"][1] = 'X'
	result.Roles[0].Header.Set("X-Request-Id", "caller")
	roleCloudField(t, result.Inventory[0], "id", `"a"`)
	if result.Inventory[0].Header.Get("X-Request-Id") != "cloud-role" {
		t.Fatal(result)
	}
}
func TestPythonRoleCloudSearchStringExpressionsUseMaterializedRows(t *testing.T) {
	calls := 0
	cloud, _ := roleCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.RawQuery != "" {
			t.Fatal(r.URL)
		}
		testcloud.JSON(w, 200, `{"roles":[{"id":"a","name":"same"},{"id":"b","name":"other"}]}`)
	})
	for _, tc := range []struct{ name, filter, want string }{{"", "length(@)", "2"}, {"same", "[0].id", `"a"`}, {"", "[?name==`same`].id", `["a"]`}, {"missing", "length(@)", "0"}, {"", "`null`", "null"}} {
		raw, _ := json.Marshal(tc.filter)
		before := calls
		result, err := cloud.SearchRoles(context.Background(), tc.name, identity.WithRoleSearchFilters(raw))
		if err != nil || result == nil || !result.Expression || string(result.Value) != tc.want || result.Roles != nil || len(result.Inventory) != 2 || calls != before+1 {
			t.Fatal(tc, result, err, calls)
		}
	}
}
func TestPythonRoleCloudFalseyFiltersPreserveDeclaredRows(t *testing.T) {
	calls := 0
	cloud, _ := roleCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		testcloud.JSON(w, 200, `{"roles":[{"id":null,"name":false}]}`)
	})
	for _, raw := range []string{"null", "false", "0", "1e-4000", `""`, "[]", "{}"} {
		result, err := cloud.SearchRoles(context.Background(), "", identity.WithRoleSearchFilters(json.RawMessage(raw)))
		if err != nil || result == nil || len(result.Roles) != 1 || len(result.Inventory) != 1 || result.Expression {
			t.Fatal(raw, result, err)
		}
		roleCloudField(t, result.Roles[0], "id", "null")
		roleCloudField(t, result.Roles[0], "name", "false")
	}
	if calls != 7 {
		t.Fatal(calls)
	}
	result, err := cloud.SearchRoles(context.Background(), "None")
	if err != nil || len(result.Roles) != 1 {
		t.Fatal(result, err)
	}
}
func TestPythonRoleCloudLocalErrorsFollowEveryPageAndKeepInventory(t *testing.T) {
	for _, raw := range []string{`"["`, `{"unknown":true}`, `{"description":{"x":1}}`, `{invalid`} {
		calls := 0
		cloud, _ := roleCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			if calls == 1 {
				testcloud.JSON(w, 203, `{"roles":[{"id":"a","description":"truthy"}],"next":"?marker=a"}`)
			} else {
				testcloud.JSON(w, 200, `{"roles":[{"id":"b","description":"truthy"}]}`)
			}
		})
		result, err := cloud.SearchRoles(context.Background(), "", identity.WithRoleSearchFilters(json.RawMessage(raw)))
		var proof *resource.ResponseError
		if !errors.Is(err, resource.ErrInvalidOption) || result == nil || len(result.Inventory) != 2 || result.Roles != nil || result.Value != nil || calls != 2 || errors.As(err, &proof) {
			t.Fatal(raw, result, err, proof, calls)
		}
		if result.Inventory[0].StatusCode != 203 || result.Inventory[1].StatusCode != 200 {
			t.Fatal(result)
		}
	}
}
func TestPythonRoleCloudOrderedMismatchHidesUnreachedBadShape(t *testing.T) {
	cloud, _ := roleCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"roles":[{"id":"a","name":"other","description":"scalar"}]}`)
	})
	result, err := cloud.SearchRoles(context.Background(), "", identity.WithRoleSearchFilters(json.RawMessage(`{"name":"mismatch","description":{"x":1}}`)))
	if err != nil || result == nil || string(result.Value) != "[]" || result.Roles == nil || len(result.Roles) != 0 {
		t.Fatal(result, err)
	}
	result, err = cloud.SearchRoles(context.Background(), "", identity.WithRoleSearchFilters(json.RawMessage(`{"description":{"x":1},"name":"mismatch"}`)))
	if !errors.Is(err, resource.ErrInvalidOption) || result == nil || len(result.Inventory) != 1 {
		t.Fatal(result, err)
	}
}
func TestPythonRoleCloudLateHTTPFailurePrecedesInvalidLocalFilter(t *testing.T) {
	calls := 0
	cloud, _ := roleCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("X-Request-Id", "late-role")
		if calls == 1 {
			testcloud.JSON(w, 203, `{"roles":[{"id":"a"}],"next":"?marker=a"}`)
		} else {
			testcloud.JSON(w, 503, `{"error":"actual late"}`)
		}
	})
	result, err := cloud.SearchRoles(context.Background(), "", identity.WithRoleSearchFilters(json.RawMessage(`"["`)))
	var proof gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &proof) || proof.Actual != 503 || proof.ResponseHeader.Get("X-Request-Id") != "late-role" || string(proof.Body) != `{"error":"actual late"}` || proof.Method != "GET" || !strings.Contains(proof.URL, "/identity/v3/roles?marker=a") || result == nil || len(result.Inventory) != 1 || result.Value != nil || calls != 2 {
		t.Fatal(result, err, proof, calls)
	}
}
func TestPythonRoleCloudLocationBeforeFilterAndOwnedArguments(t *testing.T) {
	cloudName, projectName := "configured", "current"
	location := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &projectName}}
	raw := json.RawMessage(`{"location":{"cloud":"configured","project":{"id":"scope","name":"current"}}}`)
	headers := map[string]string{"X-Owned": "owned"}
	domain := "domain"
	opt := identity.WithRoleSearchOpts(identity.RoleSearchOpts{DomainID: &domain, Filters: &raw, Headers: headers, Location: &location})
	cloudName = "changed"
	projectName = "changed"
	domain = "changed"
	raw[0] = '['
	headers["X-Owned"] = "changed"
	cloud, _ := roleCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "domain_id=domain" || r.Header.Get("X-Owned") != "owned" {
			t.Fatal(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"roles":[{"id":"a","project_id":"foreign","location":{"cloud":"wire"}}]}`)
	})
	result, err := cloud.SearchRoles(context.Background(), "", opt)
	if err != nil || len(result.Roles) != 1 || len(result.Inventory) != 1 || string(result.Roles[0].Wire.Body["location"]) != `{"cloud":"wire"}` {
		t.Fatal(result, err)
	}
	if _, exists := result.Roles[0].Resource.Body["project_id"]; exists {
		t.Fatal(result)
	}
}
func TestPythonRoleCloudOptionsGuardCallbacksAndSelectedSourceSnapshot(t *testing.T) {
	calls := 0
	cloud, _ := roleCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("X-Snapshot") != "before" || r.Header.Get("X-Late") != "" {
			t.Fatal(r.Header)
		}
		testcloud.JSON(w, 200, `{"roles":[]}`)
	})
	client := cloud.RawAPI().RawClient()
	client.MoreHeaders = map[string]string{"X-Snapshot": "before"}
	_, err := cloud.SearchRoles(context.Background(), "", identity.RoleSearchOption(func(*identity.RoleSearchOpts) error {
		client.MoreHeaders["X-Snapshot"] = "after"
		client.MoreHeaders["X-Late"] = "after"
		return nil
	}))
	if err != nil || calls != 1 {
		t.Fatal(err, calls)
	}
	for _, mode := range []string{"source", "cancel", "nil"} {
		t.Run(mode, func(t *testing.T) {
			before := calls
			seen := 0
			endpoint := client.Endpoint
			defer func() { client.Endpoint = endpoint }()
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("caller canceled role search")
			first := identity.RoleSearchOption(func(*identity.RoleSearchOpts) error {
				if mode == "source" {
					client.Endpoint += "changed"
				} else {
					cancel(cause)
				}
				return nil
			})
			if mode == "nil" {
				first = nil
			}
			_, err := cloud.SearchRoles(ctx, "", first, identity.RoleSearchOption(func(*identity.RoleSearchOpts) error { seen++; client.Endpoint = endpoint; return nil }))
			var proof *resource.ResponseError
			if err == nil || seen != 0 || calls != before || errors.As(err, &proof) {
				t.Fatal(err, seen, calls, proof)
			}
			if mode == "cancel" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
		})
	}
	for _, nilCloud := range []*identity.RoleReadCloud{nil, identity.NewRoleReadCloud(nil)} {
		result, err := nilCloud.SearchRoles(context.Background(), "")
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(result, err)
		}
	}
	if !reflect.DeepEqual(cloud.RawAPI().RawClient(), client) {
		t.Fatal(cloud)
	}
}

func TestPythonRoleCloudListForwardsOwnedProxyQueryAndLocalBodyFilters(t *testing.T) {
	calls := 0
	cloud, _ := roleCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, err := io.ReadAll(r.Body)
		if r.URL.RawQuery != "domain_id=d&name=True" || err != nil || len(body) != 0 || r.ContentLength != 0 || r.Header.Get("Accept") != "application/json" {
			t.Fatal(r.URL, string(body), err, r.Header)
		}
		w.Header().Set("X-Request-Id", "cloud-role-list")
		testcloud.JSON(w, 203, `{"roles":[{"id":"a","name":"actual-server-name","options":{"flag":1.0}},{"id":"b","options":{"flag":false}}]}`)
	})
	result, err := cloud.ListRoles(context.Background(), roles.WithRecordListQuery("domain_id", "d"), roles.WithRecordListQuery("name", true), roles.WithRecordListQuery("options", map[string]any{"flag": true}), roles.WithRecordListQuery("unknown", func() {}))
	if err != nil || result == nil || len(result.Roles) != 1 || len(result.Inventory) != 1 || len(result.Value) != 1 || result.JSON == nil || calls != 1 || result.Roles[0].StatusCode != 203 || result.Inventory[0].Header.Get("X-Request-Id") != "cloud-role-list" {
		t.Fatal(result, err, calls)
	}
	result.Roles[0].Resource.Body["id"][1] = 'X'
	result.Roles[0].Header.Set("X-Request-Id", "changed")
	roleCloudField(t, result.Inventory[0], "id", `"a"`)
	view, ok := result.Value[0].(*resource.RawResource)
	if !ok || string(view.Body["id"]) != `"a"` || result.Inventory[0].Header.Get("X-Request-Id") != "cloud-role-list" {
		t.Fatal(result)
	}
}
func TestPythonRoleCloudListLiteralOpaqueAndNonIterableBranchesNeverFabricateHTTP(t *testing.T) {
	calls := 0
	cloud, _ := roleCloudFixture(t, func(w http.ResponseWriter, r *http.Request) { calls++; testcloud.JSON(w, 200, `{"roles":[]}`) })
	for _, tc := range []struct{ expression, want string }{{"`[]`", "[]"}, {"`[1,false,null]`", "[1,false,null]"}, {"`\"ab\"`", `["a","b"]`}, {"`{\"z\":1,\"a\":2}`", `["a","z"]`}} {
		result, err := cloud.ListRoles(context.Background(), roles.WithRecordListJMESPath(tc.expression))
		if err != nil || result == nil || string(result.JSON) != tc.want || result.Roles != nil || len(result.Inventory) != 0 || calls != 0 {
			t.Fatal(tc, result, err, calls)
		}
	}
	result, err := cloud.ListRoles(context.Background(), roles.WithRecordListJMESPath("to_array(@)"))
	if err != nil || result == nil || result.JSON != nil || len(result.Value) != 1 || calls != 0 {
		t.Fatal(result, err, calls)
	}
	if _, ok := result.Value[0].(*roles.RecordListGenerator); !ok {
		t.Fatal(result.Value)
	}
	for _, expr := range []string{"[0].id", "length(@)", "`1`", "`false`"} {
		_, err := cloud.ListRoles(context.Background(), roles.WithRecordListJMESPath(expr))
		var proof *resource.ResponseError
		if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || calls != 0 {
			t.Fatal(expr, err, proof, calls)
		}
	}
}
func TestPythonRoleCloudListOptionCallbacksSnapshotSourceAndLateNativeProof(t *testing.T) {
	calls, callbacks := 0, 0
	cloud, _ := roleCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("X-Selected") != "before" || r.Header.Get("X-Added") != "" {
			t.Fatal(r.Header)
		}
		if calls == 1 {
			testcloud.JSON(w, 200, `{"roles":[{"id":"a"}],"next":"?marker=a"}`)
			return
		}
		w.Header().Set("X-Request-Id", "late-role-native")
		testcloud.JSON(w, 503, `{"error":"actual late"}`)
	})
	cloud.RawAPI().RawClient().MoreHeaders = map[string]string{"X-Selected": "before"}
	result, err := cloud.ListRoles(context.Background(), func(*roles.RecordListOpts) error {
		callbacks++
		cloud.RawAPI().RawClient().MoreHeaders["X-Selected"] = "after"
		cloud.RawAPI().RawClient().MoreHeaders["X-Added"] = "injected"
		return nil
	})
	var proof gophercloud.ErrUnexpectedResponseCode
	if result == nil || len(result.Inventory) != 1 || len(result.Roles) != 1 || !errors.As(err, &proof) || proof.Actual != 503 || string(proof.Body) != `{"error":"actual late"}` || proof.ResponseHeader.Get("X-Request-Id") != "late-role-native" || proof.Method != http.MethodGet || !strings.Contains(proof.URL, "/identity/v3/roles?marker=a") || calls != 2 || callbacks != 1 {
		t.Fatal(result, err, proof, calls, callbacks)
	}
	for _, ctx := range []context.Context{nil, func() context.Context {
		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(errors.New("role cloud canceled"))
		return ctx
	}()} {
		result, err := cloud.ListRoles(ctx)
		var p *resource.ResponseError
		if result != nil || err == nil || errors.As(err, &p) || calls != 2 {
			t.Fatal(result, err, p, calls)
		}
	}
}
