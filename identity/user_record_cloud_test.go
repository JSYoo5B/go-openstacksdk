package identity_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func userCloudWire(status int, body string) *http.Response {
	return testcloud.Response(status, body, http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"actual-user-cloud"}})
}
func userCloudFixture(t *testing.T, reply func(*http.Request) *http.Response) (*identity.UserReadCloud, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(r *http.Request) (*http.Response, error) { return reply(r), nil }))
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return identity.NewUserReadCloud(users.New(client)), cloud
}
func userCloudIDs(t *testing.T, rows []*users.UserRecord) []string {
	t.Helper()
	result := make([]string, 0, len(rows))
	for _, row := range rows {
		var id string
		if err := json.Unmarshal(row.Resource.Body["id"], &id); err != nil {
			t.Fatal(err)
		}
		result = append(result, id)
	}
	return result
}

func TestPythonUserCloudListDefaultAndOnlyDeclaredSourceKwargs(t *testing.T) {
	for _, mode := range []string{"default", "typed", "raw"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			cloud, _ := userCloudFixture(t, func(r *http.Request) *http.Response {
				calls++
				if r.Method != "GET" || r.URL.Path != "/keystone/v3/users" || r.Body != nil || r.Header.Get("Accept") != "application/json" || r.Header.Get("OpenStack-API-Version") != "" {
					t.Fatal(r.Method, r.URL, r.Header)
				}
				q := r.URL.Query()
				switch mode {
				case "default":
					if len(q) != 0 {
						t.Fatal(q)
					}
				case "typed":
					if q.Get("domain_id") != "domain" || q.Get("name") != "alice" || len(q) != 2 {
						t.Fatal(q)
					}
				case "raw":
					if q.Has("domain_id") || !reflect.DeepEqual(q["name"], []string{"alice", "False"}) {
						t.Fatal(q)
					}
				}
				return userCloudWire(203, `{"users":[{"id":"one","name":"alice","email":"a","default_project_id":"foreign","x_unknown":true}],"links":{"next":"https://ignored.invalid/v3/users"}}`)
			})
			var opts []identity.UserListOption
			if mode == "typed" {
				opts = []identity.UserListOption{identity.WithUserListDomainID("domain"), identity.WithUserListName("alice")}
			}
			if mode == "raw" {
				opts = []identity.UserListOption{identity.WithUserListQuery("domain_id", nil), identity.WithUserListQuery("name", []any{"alice", nil, false})}
			}
			result, err := cloud.ListUsers(context.Background(), opts...)
			if err != nil || result == nil || !reflect.DeepEqual(userCloudIDs(t, result.Users), []string{"one"}) || len(result.Inventory) != 1 || calls != 1 || !json.Valid(result.Value) {
				t.Fatal(result, err, calls)
			}
			row := result.Users[0]
			if row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "actual-user-cloud" || string(row.Resource.Body["options"]) != `{}` || string(row.Resource.Body["location"]) != "null" {
				t.Fatal(row)
			}
			if _, exists := row.Resource.Body["x_unknown"]; exists {
				t.Fatal(row.Resource.Body)
			}
			row.Resource.Body["id"][1] = 'X'
			row.Header.Set("X-Request-Id", "mutated")
			if string(result.Inventory[0].Resource.Body["id"]) != `"one"` || result.Inventory[0].Header.Get("X-Request-Id") != "actual-user-cloud" {
				t.Fatal(result.Inventory[0])
			}
		})
	}
}

type userCloudBadMarshal struct{ calls *int }

func (v userCloudBadMarshal) MarshalJSON() ([]byte, error) {
	*v.calls++
	return nil, errors.New("must not encode invalid cloud kwarg")
}
func TestPythonUserCloudListValidKwargsRejectBeforeMarshalerAndHTTP(t *testing.T) {
	calls, encoded := 0, 0
	cloud, _ := userCloudFixture(t, func(*http.Request) *http.Response { calls++; return userCloudWire(200, `{"users":[]}`) })
	for _, key := range []string{"email", "is_enabled", "filters", "headers", "microversion", "limit", "marker", "jmespath_filters", "unknown"} {
		result, err := cloud.ListUsers(context.Background(), identity.WithUserListQuery(key, userCloudBadMarshal{&encoded}))
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || encoded != 0 {
			t.Fatal(key, result, err, calls, encoded)
		}
	}
	result, err := cloud.ListUsers(context.Background(), identity.WithUserListOpts(identity.UserListOpts{Query: map[string]json.RawMessage{"email": json.RawMessage(`"a"`)}}))
	if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
		t.Fatal(result, err, calls)
	}
}

func TestPythonUserCloudSearchExactSourceUUIDDetectionAndNameHint(t *testing.T) {
	for _, tc := range []struct {
		name string
		hint bool
	}{
		{"", false}, {"alice", true}, {"team*", true}, {"01234567-89ab-cdef-0123-456789abcdef", false}, {"0123456789ABCDEF0123456789ABCDEF", false}, {"{0123456789abcdef0123456789abcdef}", false}, {"urn:uuid:01234567-89ab-cdef-0123-456789abcdef", false}, {"0-123456789abcdef0123456789abcdef", false}, {"URN:UUID:0123456789abcdef0123456789abcdef", true}, {"0123456789abcdef0123456789abcdeg", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			cloud, _ := userCloudFixture(t, func(r *http.Request) *http.Response {
				calls++
				q := r.URL.Query()
				if q.Get("domain_id") != "d" || q.Has("name") != tc.hint || tc.hint && q.Get("name") != tc.name || q.Has("id") {
					t.Fatal(tc, q)
				}
				row, _ := json.Marshal(map[string]any{"id": tc.name, "name": tc.name})
				return userCloudWire(200, `{"users":[`+string(row)+`]}`)
			})
			result, err := cloud.SearchUsers(context.Background(), tc.name, identity.WithUserSearchDomainID("d"))
			if err != nil || result == nil || len(result.Users) != 1 || len(result.Inventory) != 1 || calls != 1 {
				t.Fatal(tc, result, err, calls)
			}
		})
	}
}

func TestPythonUserCloudSearchGlobOrderedLoadedFiltersAndDeclaredMembership(t *testing.T) {
	calls := 0
	cloud, _ := userCloudFixture(t, func(r *http.Request) *http.Response {
		calls++
		if r.URL.Query().Get("name") != "team*" || r.URL.Query().Get("domain_id") != "d" || r.URL.Query().Has("options") || r.URL.Query().Has("is_enabled") {
			t.Fatal(r.URL)
		}
		return userCloudWire(200, `{"users":[{"id":"one","name":"team-alpha","enabled":true,"options":{"n":9007199254740992,"nested":{"ok":1},"extra":1}},{"id":"two","name":"other","enabled":true,"options":{"n":9007199254740992}}]}`)
	})
	result, err := cloud.SearchUsers(context.Background(), "team*", identity.WithUserSearchDomainID("d"), identity.WithUserSearchFilters(json.RawMessage(`{"is_enabled":true,"options":{"n":9007199254740993.0,"nested":{"ok":true}}}`)))
	if err != nil || result == nil || !reflect.DeepEqual(userCloudIDs(t, result.Users), []string{"one"}) || len(result.Inventory) != 2 || calls != 1 {
		t.Fatal(result, err, calls)
	}
	result.Users[0].Resource.Body["id"][1] = 'X'
	if string(result.Inventory[0].Resource.Body["id"]) != `"one"` {
		t.Fatal(result.Inventory)
	}
	// The wire spelling is not a declared Resource member in the Cloud filter.
	result, err = cloud.SearchUsers(context.Background(), "team*", identity.WithUserSearchDomainID("d"), identity.WithUserSearchFilters(json.RawMessage(`{"enabled":true}`)))
	var receipt *resource.ResponseError
	if result == nil || err == nil || result.Value != nil || len(result.Inventory) != 2 || errors.As(err, &receipt) {
		t.Fatal(result, err, receipt)
	}
}

func TestPythonUserCloudSearchExpressionsMaterializeThenReturnArbitraryJSON(t *testing.T) {
	for _, tc := range []struct{ expression, want string }{
		{"length(@)", "2"}, {"[0].email", `"a"`}, {"[?is_enabled].id", `["one"]`}, {"`true`", "true"}, {"`null`", "null"}, {"keys(`[\"unused\"]`)", "ERROR"},
	} {
		calls := 0
		cloud, _ := userCloudFixture(t, func(*http.Request) *http.Response {
			calls++
			return userCloudWire(200, `{"users":[{"id":"one","email":"a","enabled":true},{"id":"two","enabled":false}]}`)
		})
		raw, _ := json.Marshal(tc.expression)
		result, err := cloud.SearchUsers(context.Background(), "", identity.WithUserSearchFilters(raw))
		if tc.want == "ERROR" {
			if result == nil || err == nil || result.Value != nil || len(result.Inventory) != 2 || calls != 1 {
				t.Fatal(result, err, calls)
			}
			continue
		}
		if err != nil || result == nil || string(result.Value) != tc.want || !result.Expression || result.Users != nil || len(result.Inventory) != 2 || calls != 1 {
			t.Fatal(tc, result, err, calls)
		}
	}
}

func TestPythonUserCloudSearchFalseyAndReachedInvalidFilters(t *testing.T) {
	for _, filter := range []string{"null", "false", "0", "1e-400", `""`, "[]", "{}"} {
		calls := 0
		cloud, _ := userCloudFixture(t, func(*http.Request) *http.Response { calls++; return userCloudWire(200, `{"users":[{"id":"one"}]}`) })
		result, err := cloud.SearchUsers(context.Background(), "", identity.WithUserSearchFilters(json.RawMessage(filter)))
		if err != nil || result == nil || !reflect.DeepEqual(userCloudIDs(t, result.Users), []string{"one"}) || result.Expression || calls != 1 {
			t.Fatal(filter, result, err, calls)
		}
	}
	for _, filter := range []string{"true", "[1]", `{"unknown":null}`, `"["`} {
		calls := 0
		cloud, _ := userCloudFixture(t, func(*http.Request) *http.Response { calls++; return userCloudWire(200, `{"users":[{"id":"one"}]}`) })
		result, err := cloud.SearchUsers(context.Background(), "", identity.WithUserSearchFilters(json.RawMessage(filter)))
		if result == nil || err == nil || result.Value != nil || len(result.Inventory) != 1 || calls != 1 {
			t.Fatal(filter, result, err, calls)
		}
	}
	// A truthy nonmapping filter is never evaluated if identifier phase selects 0.
	cloud, _ := userCloudFixture(t, func(*http.Request) *http.Response {
		return userCloudWire(200, `{"users":[{"id":"one","name":"other"}]}`)
	})
	result, err := cloud.SearchUsers(context.Background(), "missing", identity.WithUserSearchFilters(json.RawMessage("true")))
	if err != nil || result == nil || string(result.Value) != "[]" || result.Users == nil {
		t.Fatal(result, err)
	}
}

func TestPythonUserCloudSearchAllPagesBeforeLocalFailureAndLateHTTPWins(t *testing.T) {
	for _, late := range []bool{false, true} {
		calls := 0
		cloud, _ := userCloudFixture(t, func(r *http.Request) *http.Response {
			calls++
			if r.URL.Query().Get("domain_id") != "d" || r.URL.Query().Has("name") {
				t.Fatal(r.URL)
			}
			if calls == 1 {
				return userCloudWire(203, `{"users":[{"id":"one"}],"next":"?marker=one"}`)
			}
			if late {
				return userCloudWire(503, `{"error":"late actual failure"}`)
			}
			return userCloudWire(200, `{"users":[{"id":"two"}]}`)
		})
		result, err := cloud.SearchUsers(context.Background(), "", identity.WithUserSearchDomainID("d"), identity.WithUserSearchFilters(json.RawMessage(`"["`)))
		if result == nil || err == nil || result.Value != nil || result.Users != nil || calls != 2 {
			t.Fatal(result, err, calls)
		}
		if late {
			var native gophercloud.ErrUnexpectedResponseCode
			if len(result.Inventory) != 1 || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"error":"late actual failure"}` || native.ResponseHeader.Get("X-Request-Id") != "actual-user-cloud" {
				t.Fatal(result, err, native)
			}
		} else {
			var receipt *resource.ResponseError
			if len(result.Inventory) != 2 || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &receipt) {
				t.Fatal(result, err, receipt)
			}
		}
	}
}

func TestPythonUserCloudListOptionSnapshotsHeadersAndCallbacksOwnMaps(t *testing.T) {
	calls := 0
	cloud, _ := userCloudFixture(t, func(r *http.Request) *http.Response {
		calls++
		if r.Header.Get("X-Trace") != "owned" || r.Header.Get("OpenStack-API-Version") != "identity 3.9" || r.URL.RawQuery != "domain_id=d&name=alice" {
			t.Fatal(r.Header, r.URL)
		}
		return userCloudWire(200, `{"users":[]}`)
	})
	config := identity.UserListOpts{Query: map[string]json.RawMessage{"name": json.RawMessage(`"alice"`)}, Headers: map[string]string{"X-Trace": "owned"}}
	snapshot := identity.WithUserListOpts(config)
	config.Query["name"][1] = 'X'
	config.Headers["X-Trace"] = "mutated"
	var retained map[string]json.RawMessage
	result, err := cloud.ListUsers(context.Background(), snapshot, identity.WithUserListMicroversion("3.9"), func(c *identity.UserListOpts) error {
		c.Query["domain_id"] = json.RawMessage(`"d"`)
		retained = c.Query
		return nil
	}, func(c *identity.UserListOpts) error { retained["name"][1] = 'Y'; return nil })
	if err != nil || result == nil || result.Users == nil || len(result.Users) != 0 || calls != 1 {
		t.Fatal(result, err, calls)
	}
	// Default callback header/query maps are initialized and remain copied.
	result, err = cloud.ListUsers(context.Background(), func(c *identity.UserListOpts) error {
		c.Query["domain_id"] = json.RawMessage(`"d"`)
		c.Query["name"] = json.RawMessage(`"alice"`)
		c.Headers["X-Trace"] = "owned"
		v := "3.9"
		c.Microversion = &v
		return nil
	})
	if err != nil || calls != 2 {
		t.Fatal(result, err, calls)
	}
}

func TestPythonUserCloudSelectedSourceCancellationAndPreflightCannotCreateReceipt(t *testing.T) {
	for _, operation := range []string{"list", "search"} {
		for _, mode := range []string{"source", "cancel"} {
			calls := 0
			cloud, _ := userCloudFixture(t, func(*http.Request) *http.Response { calls++; return userCloudWire(200, `{"users":[]}`) })
			ctx, cancel := context.WithCancelCause(context.Background())
			cause := errors.New("caller stopped User read")
			callback := errors.New("option also failed")
			later := false
			change := func() error {
				if mode == "source" {
					cloud.RawAPI().RawClient().Endpoint = "https://changed.invalid/v3/"
				} else {
					cancel(cause)
				}
				return callback
			}
			var err error
			if operation == "list" {
				_, err = cloud.ListUsers(ctx, func(*identity.UserListOpts) error { return change() }, func(*identity.UserListOpts) error { later = true; return nil })
			} else {
				_, err = cloud.SearchUsers(ctx, "", func(*identity.UserSearchOpts) error { return change() }, func(*identity.UserSearchOpts) error { later = true; return nil })
			}
			var receipt *resource.ResponseError
			if !errors.Is(err, callback) || calls != 0 || later || errors.As(err, &receipt) || mode == "cancel" && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
				t.Fatal(operation, mode, err, calls, later, receipt)
			}
			cancel(nil)
		}
	}
	var empty *identity.UserReadCloud
	if _, err := empty.ListUsers(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := empty.SearchUsers(context.Background(), ""); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestPythonUserCloudLocationBeforeFilteringIgnoresDefaultProjectAndRemoteForgery(t *testing.T) {
	cloudName, projectName := "configured", "current"
	location := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &projectName}}
	option := identity.WithUserSearchLocation(location)
	cloudName = "changed"
	location.Project.ID[1] = 'X'
	calls := 0
	cloud, _ := userCloudFixture(t, func(*http.Request) *http.Response {
		calls++
		return userCloudWire(200, `{"users":[{"id":"one","default_project_id":"foreign","project_id":"forged","location":{"cloud":"server"}}]}`)
	})
	result, err := cloud.SearchUsers(context.Background(), "", option, identity.WithUserSearchFilters(json.RawMessage(`{"location":{"cloud":"configured","project":{"id":"scope","name":"current"}}}`)))
	if err != nil || result == nil || len(result.Users) != 1 || calls != 1 {
		t.Fatal(result, err, calls)
	}
	row := result.Users[0]
	if string(row.Wire.Body["location"]) != `{"cloud":"server"}` || string(row.Resource.Body["default_project_id"]) != `"foreign"` {
		t.Fatal(row)
	}
	if _, exists := row.Resource.Body["project_id"]; exists {
		t.Fatal(row.Resource.Body)
	}
	result, err = cloud.SearchUsers(context.Background(), "", identity.WithUserSearchLocation(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`bad`)}}))
	if result == nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
		t.Fatal(result, err, calls)
	}
}

func TestPythonUserCloudEmptyListAndPythonNoneIdentifier(t *testing.T) {
	for _, raw := range []string{`{"users":[]}`, `{"users":[{}]}`} {
		calls := 0
		cloud, _ := userCloudFixture(t, func(r *http.Request) *http.Response {
			calls++
			if r.URL.Query().Get("name") != "None" {
				t.Fatal(r.URL)
			}
			return userCloudWire(200, raw)
		})
		result, err := cloud.SearchUsers(context.Background(), "None")
		if err != nil || result == nil || result.Users == nil || result.Inventory == nil || calls != 1 {
			t.Fatal(result, err, calls)
		}
		want := 0
		if raw == `{"users":[{}]}` {
			want = 1
		}
		if len(result.Users) != want {
			t.Fatal(result)
		}
	}
}

func TestPythonUserCloudSearchOpaqueLocalErrorsNeverBorrowPhysicalProof(t *testing.T) {
	calls := 0
	cloud, _ := userCloudFixture(t, func(*http.Request) *http.Response {
		calls++
		return userCloudWire(203, `{"users":[{"id":"one","name":"alice"}]}`)
	})
	result, err := cloud.SearchUsers(context.Background(), "", identity.WithUserSearchFilters(json.RawMessage(`"sort_by(@,&password)"`)))
	var receipt *resource.ResponseError
	var native gophercloud.ErrUnexpectedResponseCode
	if result == nil || err == nil || result.Value != nil || len(result.Inventory) != 1 || result.Inventory[0].StatusCode != 203 || errors.As(err, &receipt) || errors.As(err, &native) || calls != 1 {
		t.Fatal(fmt.Sprint(result), err, calls)
	}
}
