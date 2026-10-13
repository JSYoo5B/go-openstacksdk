package identity_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func roleGetFixture(t *testing.T, reply func(*http.Request) *http.Response) (*identity.RoleReadCloud, *testcloud.Cloud) {
	t.Helper()
	fixture := testcloud.New(t)
	fixture.Provider.HTTPClient.Transport = userCloudHTTP(func(q *http.Request) (*http.Response, error) { return reply(q), nil })
	client := fixture.Client("identity", "/catalog/unused/")
	client.ResourceBase = fixture.Server.URL + "/keystone/v3/"
	return identity.NewRoleReadCloud(roles.New(client)), fixture
}
func roleGetWire(status int, body string) *http.Response {
	r := userCloudWire(status, body)
	r.Header.Set("X-Request-Id", "actual-role-get")
	return r
}
func TestPythonCloudGetRoleFiltersNoneVersusFalseySearchAndNoFinalFind(t *testing.T) {
	for _, filter := range []string{"omitted", "null", "{}", "[]", "false", "0", `""`} {
		t.Run(filter, func(t *testing.T) {
			calls := 0
			deprecated := filter != "omitted" && filter != "null"
			cloud, _ := roleGetFixture(t, func(q *http.Request) *http.Response {
				calls++
				if deprecated {
					if q.URL.Path != "/keystone/v3/roles" || q.URL.RawQuery != "domain_id=domain" {
						t.Fatal(q.URL)
					}
					return roleGetWire(203, `{"roles":[{"id":"search","name":"reader"}]}`)
				}
				if q.URL.Path != "/keystone/v3/roles/reader" || q.URL.RawQuery != "domain_id=domain" {
					t.Fatal(q.URL)
				}
				return roleGetWire(200, `{"role":{"id":"lookup","name":"reader"}}`)
			})
			opts := []identity.RoleGetOption{identity.WithRoleGetDomainID("domain")}
			if filter != "omitted" {
				opts = append(opts, identity.WithRoleGetFilters(json.RawMessage(filter)))
			}
			result, err := cloud.GetRole(context.Background(), "reader", opts...)
			if err != nil || result == nil || result.Role == nil || result.DeprecatedFilters != deprecated || calls != 1 {
				t.Fatal(result, err, calls)
			}
			if deprecated {
				if result.Lookup != nil || len(result.Inventory) != 1 || result.Role.StatusCode != 203 || string(result.Role.Resource.Body["id"]) != `"search"` {
					t.Fatal(result)
				}
			} else {
				if result.Lookup == nil || len(result.Inventory) != 0 || string(result.Role.Resource.Body["id"]) != `"lookup"` {
					t.Fatal(result)
				}
			}
		})
	}
}
func TestPythonCloudGetRoleFilteredGlobLoadedMappingDuplicatesAndMissing(t *testing.T) {
	for _, mode := range []string{"single", "duplicate", "missing", "ordered mismatch"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			cloud, _ := roleGetFixture(t, func(q *http.Request) *http.Response {
				calls++
				if q.URL.Path != "/keystone/v3/roles" || q.URL.Query().Has("name") {
					t.Fatal(q.URL)
				}
				return roleGetWire(203, `{"roles":[{"id":"one","name":"read-alpha","description":{"n":0.10000000000000001}},{"id":"one","name":"read-beta","description":{"n":0.2}}]}`)
			})
			name, filter := "read-*", json.RawMessage(`{"description":{"n":0.1}}`)
			if mode == "duplicate" {
				filter = json.RawMessage(`{}`)
			}
			if mode == "missing" {
				name = "missing"
			}
			if mode == "ordered mismatch" {
				filter = json.RawMessage(`{"name":"wrong","unknown":true}`)
			}
			result, err := cloud.GetRole(context.Background(), name, identity.WithRoleGetFilters(filter))
			if result == nil || len(result.Inventory) != 2 || calls != 1 || result.Lookup != nil {
				t.Fatal(result, err, calls)
			}
			switch mode {
			case "single":
				if err != nil || result.Role == nil || string(result.Role.Resource.Body["name"]) != `"read-alpha"` || result.Role.StatusCode != 203 {
					t.Fatal(result, err)
				}
				result.Role.Resource.Body["id"][1] = 'X'
				if string(result.Inventory[0].Resource.Body["id"]) != `"one"` {
					t.Fatal(result)
				}
			case "duplicate":
				if !errors.Is(err, resource.ErrAmbiguous) || result.Role != nil || result.Value != nil {
					t.Fatal(result, err)
				}
			default:
				if err != nil || result.Role != nil || string(result.Value) != "null" {
					t.Fatal(result, err)
				}
			}
		})
	}
}
func TestPythonCloudGetRolePolymorphicTruthLenAndIntegerZero(t *testing.T) {
	for _, tc := range []struct{ expression, want, kind string }{
		{"`null`", "null", "value"}, {"`false`", "null", "value"}, {"`0`", "null", "value"}, {"`[]`", "null", "value"}, {"`{}`", "null", "value"}, {"`\"\"`", "null", "value"},
		{"`[null]`", "null", "value"}, {"`[false]`", "false", "value"}, {"`[0]`", "0", "value"}, {"`\"界\"`", `"界"`, "value"}, {"[0:1].id", `"one"`, "value"},
		{"`[1,2]`", "", "ambiguous"}, {"`{\"a\":1,\"b\":2}`", "", "ambiguous"}, {"`\"ab\"`", "", "ambiguous"}, {"`{\"a\":1}`", "", "invalid"}, {"`true`", "", "invalid"}, {"`1`", "", "invalid"}, {"`1.5`", "", "invalid"},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			calls := 0
			cloud, _ := roleGetFixture(t, func(q *http.Request) *http.Response {
				calls++
				if q.URL.Path != "/keystone/v3/roles" {
					t.Fatal(q.URL)
				}
				return roleGetWire(203, `{"roles":[{"id":"one","name":"reader"}]}`)
			})
			result, err := cloud.GetRole(context.Background(), "reader", identity.WithRoleGetExpression(tc.expression))
			if result == nil || !result.Expression || !result.DeprecatedFilters || result.Role != nil || result.Lookup != nil || len(result.Inventory) != 1 || calls != 1 {
				t.Fatal(result, err, calls)
			}
			switch tc.kind {
			case "value":
				if err != nil || string(result.Value) != tc.want {
					t.Fatal(tc, result, err)
				}
			case "ambiguous":
				if !errors.Is(err, resource.ErrAmbiguous) || result.Value != nil {
					t.Fatal(result, err)
				}
			case "invalid":
				var proof *resource.ResponseError
				if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || result.Value != nil {
					t.Fatal(result, err)
				}
			}
		})
	}
}
func TestPythonCloudGetRoleRawFalseyDomainAlwaysForwarded(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		query url.Values
	}{
		{"null", url.Values{}}, {`""`, url.Values{"domain_id": {""}}}, {"false", url.Values{"domain_id": {"False"}}}, {"0", url.Values{"domain_id": {"0"}}}, {`["domain",null,false]`, url.Values{"domain_id": {"domain", "False"}}},
	} {
		for _, filtered := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", tc.raw, filtered), func(t *testing.T) {
				calls := 0
				cloud, _ := roleGetFixture(t, func(q *http.Request) *http.Response {
					calls++
					if !reflect.DeepEqual(q.URL.Query(), tc.query) {
						t.Fatal(q.URL.Query(), tc.query)
					}
					if filtered {
						return roleGetWire(200, `{"roles":[{"id":"one","name":"reader"}]}`)
					}
					return roleGetWire(200, `{"role":{"id":"one","name":"reader"}}`)
				})
				opts := []identity.RoleGetOption{identity.WithRoleGetDomainIDJSON(json.RawMessage(tc.raw))}
				if filtered {
					opts = append(opts, identity.WithRoleGetFilters(json.RawMessage(`{}`)))
				}
				result, err := cloud.GetRole(context.Background(), "reader", opts...)
				if err != nil || result.Role == nil || calls != 1 {
					t.Fatal(result, err, calls)
				}
				if !filtered && string(result.Role.Resource.Body["domain_id"]) != tc.raw {
					t.Fatal(result)
				}
			})
		}
	}
}
func TestPythonCloudGetRoleLateHTTPPrecedesLocalErrorAndKeepsActualInventory(t *testing.T) {
	for _, mode := range []string{"late", "local", "unvisited"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			cloud, _ := roleGetFixture(t, func(q *http.Request) *http.Response {
				calls++
				if mode == "unvisited" {
					return roleGetWire(200, `{"roles":[]}`)
				}
				if calls == 1 {
					return roleGetWire(203, `{"roles":[{"id":"one","name":"reader"}],"next":"?marker=one"}`)
				}
				if mode == "late" {
					return roleGetWire(503, `{"error":"late"}`)
				}
				return roleGetWire(200, `{"roles":[{"id":"two","name":"reader"}]}`)
			})
			raw := json.RawMessage(`"["`)
			if mode == "unvisited" {
				raw = json.RawMessage(`true`)
			}
			result, err := cloud.GetRole(context.Background(), "reader", identity.WithRoleGetFilters(raw))
			if result == nil || result.Role != nil || result.Lookup != nil {
				t.Fatal(result, err)
			}
			switch mode {
			case "late":
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &proof) || proof.Actual != 503 || len(result.Inventory) != 1 || calls != 2 {
					t.Fatal(result, err, calls)
				}
			case "local":
				var proof *resource.ResponseError
				if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || len(result.Inventory) != 2 || calls != 2 {
					t.Fatal(result, err, calls)
				}
			case "unvisited":
				if err != nil || string(result.Value) != "null" || calls != 1 {
					t.Fatal(result, err, calls)
				}
			}
		})
	}
}
func TestPythonCloudGetRoleDefaultFinderFallbackMissingDuplicateAndVersion(t *testing.T) {
	for _, mode := range []string{"missing", "single", "duplicate", "late"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			cloud, _ := roleGetFixture(t, func(q *http.Request) *http.Response {
				calls++
				if q.URL.Query().Get("domain_id") != "domain" {
					t.Fatal(q.URL)
				}
				if strings.HasSuffix(q.URL.Path, "/reader") {
					if q.Header.Get("OpenStack-API-Version") != "identity 3.8" {
						t.Fatal(q.Header)
					}
					return roleGetWire(403, `{"error":"member denied"}`)
				}
				if q.Header.Get("OpenStack-API-Version") != "identity 3.1" || q.URL.Query().Get("name") != "reader" {
					t.Fatal(q.URL, q.Header)
				}
				if mode == "missing" {
					return roleGetWire(200, `{"roles":[]}`)
				}
				if calls == 2 {
					return roleGetWire(203, `{"roles":[{"id":"one","name":"reader"}],"next":"?name=reader&domain_id=domain&marker=one"}`)
				}
				if mode == "late" {
					return roleGetWire(503, `{"error":"late"}`)
				}
				if mode == "duplicate" {
					return roleGetWire(200, `{"roles":[{"id":"one","name":"reader"}]}`)
				}
				return roleGetWire(200, `{"roles":[]}`)
			})
			cloud.RawAPI().RawClient().Microversion = "3.1"
			result, err := cloud.GetRole(context.Background(), "reader", identity.WithRoleGetDomainID("domain"), identity.WithRoleGetMicroversion("3.8"))
			switch mode {
			case "missing":
				if err != nil || result.Role != nil || result.Lookup != nil || string(result.Value) != "null" || calls != 2 {
					t.Fatal(result, err, calls)
				}
			case "single":
				if err != nil || result.Role == nil || result.Lookup == nil || result.Role.StatusCode != 203 || calls != 3 {
					t.Fatal(result, err, calls)
				}
			case "duplicate":
				if !errors.Is(err, resource.ErrAmbiguous) || result.Role != nil || result.Value != nil || calls != 3 {
					t.Fatal(result, err, calls)
				}
			case "late":
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &proof) || proof.Actual != 503 || result.Role != nil || result.Value != nil || calls != 3 {
					t.Fatal(result, err, calls)
				}
			}
		})
	}
}
func TestPythonCloudGetRoleOptsOnceOwnValuesSourceSnapshotAndLocation(t *testing.T) {
	for _, filtered := range []bool{false, true} {
		t.Run(fmt.Sprint(filtered), func(t *testing.T) {
			domain := json.RawMessage(`"domain"`)
			cloudName := "configured"
			location := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`)}}
			headers := map[string]string{"X-Owned": "owned"}
			raw := json.RawMessage(`{}`)
			opts := identity.RoleGetOpts{DomainID: domain, Headers: headers, Location: &location}
			if filtered {
				opts.Filters = &raw
			}
			snapshot := identity.WithRoleGetOpts(opts)
			domain[1] = 'X'
			cloudName = "changed"
			headers["X-Owned"] = "changed"
			raw[0] = '['
			calls, callbacks := 0, 0
			cloud, _ := roleGetFixture(t, func(q *http.Request) *http.Response {
				calls++
				if q.Header.Get("X-Selected") != "before" || q.Header.Get("X-Late") != "" || q.Header.Get("X-Owned") != "owned" || q.URL.Query().Get("domain_id") != "domain" {
					t.Fatal(q.Header, q.URL)
				}
				if filtered {
					return roleGetWire(203, `{"roles":[{"id":"one","name":"reader","project_id":"foreign","location":{"cloud":"wire"}}]}`)
				}
				return roleGetWire(200, `{"role":{"id":"one","name":"reader","project_id":"foreign","location":{"cloud":"wire"}}}`)
			})
			client := cloud.RawAPI().RawClient()
			client.MoreHeaders = map[string]string{"X-Selected": "before"}
			result, err := cloud.GetRole(context.Background(), "reader", snapshot, func(*identity.RoleGetOpts) error {
				callbacks++
				client.MoreHeaders["X-Selected"] = "after"
				client.MoreHeaders["X-Late"] = "after"
				return nil
			})
			if err != nil || result.Role == nil || calls != 1 || callbacks != 1 || !strings.Contains(string(result.Role.Resource.Body["location"]), `"cloud":"configured"`) || string(result.Role.Wire.Body["location"]) != `{"cloud":"wire"}` {
				t.Fatal(result, err, calls, callbacks)
			}
		})
	}
}
func TestPythonCloudGetRoleOwnerCancellationPrivateGuardAndPhysical401(t *testing.T) {
	calls := 0
	cloud, _ := roleGetFixture(t, func(*http.Request) *http.Response { calls++; return roleGetWire(200, `{"role":{"id":"one"}}`) })
	for _, mode := range []string{"nil", "source", "cancel", "private"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("role getter guard")
			seen := 0
			client := cloud.RawAPI().RawClient()
			old := client.Endpoint
			defer func() { client.Endpoint = old }()
			var option identity.RoleGetOption = func(*identity.RoleGetOpts) error {
				if mode == "source" {
					client.Endpoint = "https://changed.invalid/"
				} else if mode == "cancel" {
					cancel(cause)
				}
				return nil
			}
			if mode == "nil" {
				option = nil
			}
			if mode == "private" {
				ctx = rest.WithOperationGuard(ctx, func(context.Context) error { return cause })
			}
			result, err := cloud.GetRole(ctx, "one", option, func(*identity.RoleGetOpts) error { seen++; client.Endpoint = old; return nil })
			var proof *resource.ResponseError
			if result != nil || err == nil || seen != 0 || calls != 0 || errors.As(err, &proof) {
				t.Fatal(result, err, seen, calls, proof)
			}
			if mode == "cancel" && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
				t.Fatal(err)
			}
			if mode == "private" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
		})
	}
	for _, empty := range []*identity.RoleReadCloud{nil, identity.NewRoleReadCloud(nil)} {
		result, err := empty.GetRole(context.Background(), "one")
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(result, err)
		}
	}
	retries, reauth := 0, 0
	cloud, fixture := roleGetFixture(t, func(*http.Request) *http.Response {
		calls++
		r := roleGetWire(401, `{"error":"physical401"}`)
		r.ContentLength = 128
		r.Body = &getterPhysicalReadBody{Reader: strings.NewReader(`{"error":"physical401"}`)}
		return r
	})
	fixture.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		retries++
		return nil
	}
	fixture.Provider.ReauthFunc = func(context.Context) error { reauth++; return errors.New("unexpected physical-fault reauthentication") }
	result, err := cloud.GetRole(context.Background(), "one")
	var native *gophercloud.ErrUnableToReauthenticate
	if result == nil || result.Role != nil || result.Value != nil || !errors.Is(err, io.ErrUnexpectedEOF) || !errors.As(err, &native) || native.ErrOriginal == nil || native.ErrReauth == nil || calls != 1 || retries != 0 || reauth != 0 {
		t.Fatal(result, err, native, calls, retries, reauth)
	}
	var proof gophercloud.ErrUnexpectedResponseCode
	if !errors.As(native.ErrOriginal, &proof) || proof.Actual != 401 || proof.ResponseHeader.Get("X-Request-Id") != "actual-role-get" {
		t.Fatal(err, proof)
	}
}
