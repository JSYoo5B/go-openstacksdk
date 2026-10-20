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
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func groupCloudFixture(t *testing.T, reply func(http.ResponseWriter, *http.Request)) (*identity.GroupReadCloud, *testcloud.Cloud) {
	t.Helper()
	fixture := testcloud.New(t)
	fixture.Mux.HandleFunc("GET /identity/v3/groups", reply)
	return identity.NewGroupReadCloud(groups.New(fixture.Client("identity", "/identity/v3/"))), fixture
}
func groupCloudField(t *testing.T, row *groups.GroupRecord, key, want string) {
	t.Helper()
	if row == nil || row.Resource == nil || string(row.Resource.Body[key]) != want {
		t.Fatal(key, want, row)
	}
}
func TestPythonGroupCloudSearchDomainServerAndLocalGlobSubset(t *testing.T) {
	calls := 0
	cloud, _ := groupCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, readErr := io.ReadAll(r.Body)
		if r.URL.RawQuery != "domain_id=domain" || r.Header.Get("Accept") != "application/json" || readErr != nil || len(body) != 0 || r.ContentLength != 0 {
			t.Fatal(r.URL, r.Header, string(body), readErr, r.ContentLength)
		}
		w.Header().Set("X-Request-Id", "cloud-group")
		testcloud.JSON(w, 203, `{"groups":[{"id":"a","name":"ops-alpha","description":{"n":0.10000000000000001,"active":true}},{"id":"b","name":"other","description":{"n":0.2}},{"id":"a","name":"ops-beta","description":{"n":0.1,"extra":1}}]}`)
	})
	result, err := cloud.SearchGroups(context.Background(), "ops-*", identity.WithGroupSearchDomainID("domain"), identity.WithGroupSearchFilters(json.RawMessage(`{"description":{"n":0.1}}`)))
	if err != nil || result == nil || len(result.Groups) != 2 || len(result.Inventory) != 3 || result.Expression || calls != 1 {
		t.Fatal(result, err, calls)
	}
	for _, row := range result.Groups {
		groupCloudField(t, row, "id", `"a"`)
		if row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "cloud-group" {
			t.Fatal(row)
		}
	}
	result.Groups[0].Resource.Body["id"][1] = 'X'
	result.Groups[0].Header.Set("X-Request-Id", "caller")
	groupCloudField(t, result.Inventory[0], "id", `"a"`)
	if result.Inventory[0].Header.Get("X-Request-Id") != "cloud-group" {
		t.Fatal(result)
	}
}
func TestPythonGroupCloudSearchStringExpressionsUseMaterializedRows(t *testing.T) {
	calls := 0
	cloud, _ := groupCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.RawQuery != "" {
			t.Fatal(r.URL)
		}
		testcloud.JSON(w, 200, `{"groups":[{"id":"a","name":"same"},{"id":"b","name":"other"}]}`)
	})
	for _, tc := range []struct{ name, filter, want string }{{"", "length(@)", "2"}, {"same", "[0].id", `"a"`}, {"", "[?name==`same`].id", `["a"]`}, {"missing", "length(@)", "0"}, {"", "`null`", "null"}} {
		raw, _ := json.Marshal(tc.filter)
		before := calls
		result, err := cloud.SearchGroups(context.Background(), tc.name, identity.WithGroupSearchFilters(raw))
		if err != nil || result == nil || !result.Expression || string(result.Value) != tc.want || result.Groups != nil || len(result.Inventory) != 2 || calls != before+1 {
			t.Fatal(tc, result, err, calls)
		}
	}
}
func TestPythonGroupCloudFalseyFiltersPreserveDeclaredRows(t *testing.T) {
	calls := 0
	cloud, _ := groupCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		testcloud.JSON(w, 200, `{"groups":[{"id":null,"name":false}]}`)
	})
	for _, raw := range []string{"null", "false", "0", "1e-4000", `""`, "[]", "{}"} {
		result, err := cloud.SearchGroups(context.Background(), "", identity.WithGroupSearchFilters(json.RawMessage(raw)))
		if err != nil || result == nil || len(result.Groups) != 1 || len(result.Inventory) != 1 || result.Expression {
			t.Fatal(raw, result, err)
		}
		groupCloudField(t, result.Groups[0], "id", "null")
		groupCloudField(t, result.Groups[0], "name", "false")
	}
	if calls != 7 {
		t.Fatal(calls)
	}
	result, err := cloud.SearchGroups(context.Background(), "None")
	if err != nil || len(result.Groups) != 1 {
		t.Fatal(result, err)
	}
}
func TestPythonGroupCloudLocalErrorsFollowEveryPageAndKeepInventory(t *testing.T) {
	for _, raw := range []string{`"["`, `{"unknown":true}`, `{"description":{"x":1}}`, `{invalid`} {
		calls := 0
		cloud, _ := groupCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			if calls == 1 {
				testcloud.JSON(w, 203, `{"groups":[{"id":"a","description":"truthy"}],"next":"?marker=a"}`)
			} else {
				testcloud.JSON(w, 200, `{"groups":[{"id":"b","description":"truthy"}]}`)
			}
		})
		result, err := cloud.SearchGroups(context.Background(), "", identity.WithGroupSearchFilters(json.RawMessage(raw)))
		var proof *resource.ResponseError
		if !errors.Is(err, resource.ErrInvalidOption) || result == nil || len(result.Inventory) != 2 || result.Groups != nil || result.Value != nil || calls != 2 || errors.As(err, &proof) {
			t.Fatal(raw, result, err, proof, calls)
		}
		if result.Inventory[0].StatusCode != 203 || result.Inventory[1].StatusCode != 200 {
			t.Fatal(result)
		}
	}
}
func TestPythonGroupCloudOrderedMismatchHidesUnreachedBadShape(t *testing.T) {
	cloud, _ := groupCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"groups":[{"id":"a","name":"other","description":"scalar"}]}`)
	})
	result, err := cloud.SearchGroups(context.Background(), "", identity.WithGroupSearchFilters(json.RawMessage(`{"name":"mismatch","description":{"x":1}}`)))
	if err != nil || result == nil || string(result.Value) != "[]" || result.Groups == nil || len(result.Groups) != 0 {
		t.Fatal(result, err)
	}
	result, err = cloud.SearchGroups(context.Background(), "", identity.WithGroupSearchFilters(json.RawMessage(`{"description":{"x":1},"name":"mismatch"}`)))
	if !errors.Is(err, resource.ErrInvalidOption) || result == nil || len(result.Inventory) != 1 {
		t.Fatal(result, err)
	}
}
func TestPythonGroupCloudLateHTTPFailurePrecedesInvalidLocalFilter(t *testing.T) {
	calls := 0
	cloud, _ := groupCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("X-Request-Id", "late-group")
		if calls == 1 {
			testcloud.JSON(w, 203, `{"groups":[{"id":"a"}],"next":"?marker=a"}`)
		} else {
			testcloud.JSON(w, 503, `{"error":"actual late"}`)
		}
	})
	result, err := cloud.SearchGroups(context.Background(), "", identity.WithGroupSearchFilters(json.RawMessage(`"["`)))
	var proof gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &proof) || proof.Actual != 503 || proof.ResponseHeader.Get("X-Request-Id") != "late-group" || string(proof.Body) != `{"error":"actual late"}` || proof.Method != "GET" || !strings.Contains(proof.URL, "/identity/v3/groups?marker=a") || result == nil || len(result.Inventory) != 1 || result.Value != nil || calls != 2 {
		t.Fatal(result, err, proof, calls)
	}
}
func TestPythonGroupCloudLocationBeforeFilterAndOwnedArguments(t *testing.T) {
	cloudName, projectName := "configured", "current"
	location := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &projectName}}
	raw := json.RawMessage(`{"location":{"cloud":"configured","project":{"id":"scope","name":"current"}}}`)
	headers := map[string]string{"X-Owned": "owned"}
	domain := "domain"
	opt := identity.WithGroupSearchOpts(identity.GroupSearchOpts{DomainID: &domain, Filters: &raw, Headers: headers, Location: &location})
	cloudName = "changed"
	projectName = "changed"
	domain = "changed"
	raw[0] = '['
	headers["X-Owned"] = "changed"
	cloud, _ := groupCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "domain_id=domain" || r.Header.Get("X-Owned") != "owned" {
			t.Fatal(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"groups":[{"id":"a","project_id":"foreign","location":{"cloud":"wire"}}]}`)
	})
	result, err := cloud.SearchGroups(context.Background(), "", opt)
	if err != nil || len(result.Groups) != 1 || len(result.Inventory) != 1 || string(result.Groups[0].Wire.Body["location"]) != `{"cloud":"wire"}` {
		t.Fatal(result, err)
	}
	if _, exists := result.Groups[0].Resource.Body["project_id"]; exists {
		t.Fatal(result)
	}
}
func TestPythonGroupCloudOptionsGuardCallbacksAndSelectedSourceSnapshot(t *testing.T) {
	calls := 0
	cloud, _ := groupCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("X-Snapshot") != "before" || r.Header.Get("X-Late") != "" {
			t.Fatal(r.Header)
		}
		testcloud.JSON(w, 200, `{"groups":[]}`)
	})
	client := cloud.RawAPI().RawClient()
	client.MoreHeaders = map[string]string{"X-Snapshot": "before"}
	_, err := cloud.SearchGroups(context.Background(), "", identity.GroupSearchOption(func(*identity.GroupSearchOpts) error {
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
			cause := errors.New("caller canceled group search")
			first := identity.GroupSearchOption(func(*identity.GroupSearchOpts) error {
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
			_, err := cloud.SearchGroups(ctx, "", first, identity.GroupSearchOption(func(*identity.GroupSearchOpts) error { seen++; client.Endpoint = endpoint; return nil }))
			var proof *resource.ResponseError
			if err == nil || seen != 0 || calls != before || errors.As(err, &proof) {
				t.Fatal(err, seen, calls, proof)
			}
			if mode == "cancel" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
		})
	}
	for _, nilCloud := range []*identity.GroupReadCloud{nil, identity.NewGroupReadCloud(nil)} {
		result, err := nilCloud.SearchGroups(context.Background(), "")
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(result, err)
		}
	}
	if !reflect.DeepEqual(cloud.RawAPI().RawClient(), client) {
		t.Fatal(cloud)
	}
}
