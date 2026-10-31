package users_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func groupUserAPI(t *testing.T, reply func(*http.Request) *http.Response) (*sdk.API, *testcloud.Cloud) {
	t.Helper()
	f := testcloud.New(t)
	f.SetTransport(testcloud.LinkRequest(func(r *http.Request) (*http.Response, error) { return reply(r), nil }))
	client := f.Client("identity", "/catalog/")
	client.ResourceBase = f.Server.URL + "/keystone/v3/"
	return sdk.New(client), f
}
func groupUserWire(code int, body string) *http.Response {
	return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"group-user-proof"}})
}
func TestPythonGroupUserRecordsFullUserProfileAndFixedGroupScope(t *testing.T) {
	calls, encoded := 0, 0
	body := `{"users":[{"id":"u","name":"member","email":"keep","enabled":1e-4000,"is_enabled":true,"options":5,"federated":false,"password_expires_at":"raw-invalid-date","group_id":"foreign","location":{"cloud":"wire"},"x_unknown":{"keep":1}},{"id":"filtered","email":"skip"}]}`
	api, _ := groupUserAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.URL.EscapedPath() != "/keystone/v3/groups/g%201/users" || r.URL.RawQuery != "domain_id=d&enabled=False&password_expires_at=not-a-time" || r.Method != "GET" || r.Body != nil || r.Header.Get("Accept") != "application/json" {
			t.Fatal(r.URL, r.Header, r.Body)
		}
		return groupUserWire(203, body)
	})
	seq := api.ListUsersInGroupRecords(context.Background(), "g 1", sdk.WithGroupRecordListQuery("domain_id", "d"), sdk.WithGroupRecordListQuery("is_enabled", false), sdk.WithGroupRecordListQuery("enabled", true), sdk.WithGroupRecordListQuery("password_expires_at", "not-a-time"), sdk.WithGroupRecordListQuery("email", "keep"), sdk.WithGroupRecordListQuery("unknown", recordMarshaler{&encoded}))
	if calls != 0 || encoded != 0 {
		t.Fatal(calls, encoded)
	}
	var rows []*sdk.UserRecord
	for row, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if len(rows) != 1 || calls != 1 || encoded != 0 {
		t.Fatal(rows, calls, encoded)
	}
	for key, want := range map[string]string{"is_enabled": "true", "options": "{}", "federated": "[false]", "password_expires_at": `"raw-invalid-date"`, "location": "null"} {
		recordField(t, rows[0], key, want)
	}
	for _, key := range []string{"group_id", "enabled", "x_unknown"} {
		if _, present := rows[0].Resource.Body[key]; present {
			t.Fatal(key, rows[0].Resource)
		}
	}
	if string(rows[0].Envelope) != body || rows[0].StatusCode != 203 || rows[0].Header.Get("X-Request-Id") != "group-user-proof" || string(rows[0].Wire.Body["group_id"]) != `"foreign"` {
		t.Fatal(rows)
	}
	clone := rows[0].Clone()
	clone.Resource.Body["id"][1] = 'X'
	clone.Header.Set("X-Request-Id", "caller")
	if string(rows[0].Resource.Body["id"]) != `"u"` || rows[0].Header.Get("X-Request-Id") != "group-user-proof" {
		t.Fatal(rows)
	}
}
func TestPythonGroupUserRecordsMarkerRawCapAndBreakKeepGroupRoute(t *testing.T) {
	for _, mode := range []string{"marker", "cap", "break"} {
		t.Run(mode, func(t *testing.T) {
			var markers []string
			api, _ := groupUserAPI(t, func(r *http.Request) *http.Response {
				markers = append(markers, r.URL.Query().Get("marker"))
				if r.URL.Path != "/keystone/v3/groups/group-a/users" || r.URL.Query().Get("limit") != "5" || r.URL.Query().Has("email") {
					t.Fatal(r.URL)
				}
				switch len(markers) {
				case 1:
					return groupUserWire(200, `{"users":[{"id":"kept","email":"yes"},{"id":"last-filtered","email":"no"}],"links":{"next":"ignored.invalid"}}`)
				case 2:
					return groupUserWire(200, `{"users":[{"id":"short","email":"yes"}]}`)
				default:
					return groupUserWire(200, `{"users":[]}`)
				}
			})
			opts := []sdk.GroupRecordListOption{sdk.WithGroupRecordListLimit(5), sdk.WithGroupRecordListQuery("email", "yes")}
			if mode == "cap" {
				opts = append(opts, sdk.WithGroupRecordListMaxItems(2))
			}
			if mode == "break" {
				for row, err := range api.ListUsersInGroupRecords(context.Background(), "group-a", opts...) {
					if err != nil || row == nil {
						t.Fatal(row, err)
					}
					row.Resource.Body["id"] = json.RawMessage(`"caller-change"`)
					break
				}
				if !reflect.DeepEqual(markers, []string{""}) {
					t.Fatal(markers)
				}
				return
			}
			rows, err := api.AllUsersInGroupRecords(context.Background(), "group-a", opts...)
			if mode == "cap" {
				if err != nil || len(rows) != 1 || !reflect.DeepEqual(markers, []string{""}) {
					t.Fatal(rows, err, markers)
				}
				return
			}
			if err != nil || len(rows) != 2 || !reflect.DeepEqual(markers, []string{"", "last-filtered", "short"}) {
				t.Fatal(rows, err, markers)
			}
		})
	}
}
func TestPythonGroupUserRecordsSourceLinksAndCrossGroupContinuationProof(t *testing.T) {
	for _, mode := range []string{"dict", "null", "other-group"} {
		calls := 0
		body := `{"users":[{"id":"u"}],"links":{"next":"foreign.invalid"},"users_links":[{"rel":"next","href":"https://foreign.invalid/users"}]}`
		if mode == "null" {
			body = `{"users":[{"id":"u"}],"links":null}`
		}
		if mode == "other-group" {
			body = `{"users":[{"id":"u"}],"next":"/v3/groups/group-b/users?marker=u"}`
		}
		api, _ := groupUserAPI(t, func(*http.Request) *http.Response { calls++; return groupUserWire(203, body) })
		rows, err := api.AllUsersInGroupRecords(context.Background(), "group-a")
		if mode == "dict" {
			if err != nil || len(rows) != 1 || calls != 1 {
				t.Fatal(rows, err, calls)
			}
		} else {
			var proof *resource.ResponseError
			if err == nil || rows != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "group-user-proof" || calls != 1 {
				t.Fatal(mode, rows, err, proof, calls)
			}
		}
	}
}
func TestPythonGroupUserRecordsBasePathBindingAndBooleanLimitPreflight(t *testing.T) {
	calls, encoded := 0, 0
	api, _ := groupUserAPI(t, func(*http.Request) *http.Response { calls++; return groupUserWire(200, `{"users":[]}`) })
	for _, key := range []string{"base_path", "group"} {
		_, err := api.AllUsersInGroupRecords(context.Background(), "g", sdk.WithGroupRecordListQuery(key, recordMarshaler{&encoded}))
		if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || encoded != 0 {
			t.Fatal(key, err, calls, encoded)
		}
	}
	for _, opts := range []sdk.GroupRecordListOpts{{Query: map[string]json.RawMessage{"base_path": json.RawMessage(`null`)}}, {BasePath: new(string)}} {
		_, err := api.AllUsersInGroupRecords(context.Background(), "g", sdk.WithGroupRecordListOpts(opts))
		if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatal(opts, err, calls)
		}
	}
	for _, raw := range []string{`{"base_path":null}`, `{"paginated":false}`} {
		_, err := api.AllUsersInGroupRecords(context.Background(), "g", sdk.WithGroupRecordListQuery("__conflicting_attrs", json.RawMessage(raw)))
		if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatal(raw, err, calls)
		}
	}
	for _, limit := range []bool{true, false} {
		_, err := api.AllUsersInGroupRecords(context.Background(), "g", sdk.WithGroupRecordListQuery("limit", limit))
		var proof *resource.ResponseError
		if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || calls != 0 {
			t.Fatal(limit, err, proof, calls)
		}
	}
	for _, id := range []string{"", ".", "..", "bad\n"} {
		_, err := api.AllUsersInGroupRecords(context.Background(), id)
		if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatal(id, err, calls)
		}
	}
}
func TestPythonGroupUserRecordsOpaqueProxyAndStandaloneProfileIsolation(t *testing.T) {
	var paths []string
	api, _ := groupUserAPI(t, func(r *http.Request) *http.Response {
		paths = append(paths, r.URL.Path)
		return groupUserWire(200, `{"users":[{"id":"u"}]}`)
	})
	result, err := api.ProxyUsersInGroupRecords(context.Background(), "g", sdk.WithGroupRecordListJMESPath("[*].id"))
	if err != nil || result == nil || string(result.JSON) != "null" || len(paths) != 0 {
		t.Fatal(result, err, paths)
	}
	result, err = api.ProxyUsersInGroupRecords(context.Background(), "g", sdk.WithGroupRecordListJMESPath("@"))
	if err != nil || result.JSON != nil {
		t.Fatal(result, err)
	}
	generator, ok := result.Value.(*sdk.RecordListGenerator)
	if !ok {
		t.Fatal(result)
	}
	for row, err := range generator.Records {
		if err != nil || row == nil {
			t.Fatal(row, err)
		}
	}
	if _, err := api.AllRecords(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(paths, []string{"/keystone/v3/groups/g/users", "/keystone/v3/users"}) {
		t.Fatal(paths)
	}
	before := len(paths)
	_, err = api.ProxyUsersInGroupRecords(context.Background(), "g", sdk.WithGroupRecordListJMESPath("length(@)"))
	if !errors.Is(err, resource.ErrInvalidOption) || len(paths) != before {
		t.Fatal(err, paths)
	}
}
func TestPythonGroupUserRecordsLoadedLocalFilterAndExplicitLocation(t *testing.T) {
	name, project := "configured", "current"
	location := resource.CloudLocation{Cloud: &name, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &project}}
	option := sdk.WithGroupRecordListLocation(location)
	name = "changed"
	project = "changed"
	api, _ := groupUserAPI(t, func(*http.Request) *http.Response {
		return groupUserWire(203, `{"users":[{"id":"u","email":0.10000000000000001,"options":{"flag":true,"nested":{"n":1}},"default_project_id":"foreign","project_id":"foreign","location":{"cloud":"wire"}}]}`)
	})
	rows, err := api.AllUsersInGroupRecords(context.Background(), "g", option, sdk.WithGroupRecordListQuery("email", json.RawMessage(`0.1`)), sdk.WithGroupRecordListQuery("options", json.RawMessage(`{"flag":1,"nested":{"n":true}}`)))
	if err != nil || len(rows) != 1 || string(rows[0].Wire.Body["location"]) != `{"cloud":"wire"}` {
		t.Fatal(rows, err)
	}
	var value map[string]json.RawMessage
	_ = json.Unmarshal(rows[0].Resource.Body["location"], &value)
	var scope map[string]json.RawMessage
	_ = json.Unmarshal(value["project"], &scope)
	if string(value["cloud"]) != `"configured"` || string(scope["id"]) != `"scope"` || string(scope["name"]) != `"current"` {
		t.Fatal(rows)
	}
}
func TestPythonGroupUserRecordsOptionsCloneStickySourceAndCancel(t *testing.T) {
	calls := 0
	api, _ := groupUserAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.Header.Get("X-Owned") != "owned" || r.URL.Query().Get("name") != "owned" {
			t.Fatal(r.Header, r.URL)
		}
		return groupUserWire(200, `{"users":[]}`)
	})
	raw := json.RawMessage(`"owned"`)
	headers := map[string]string{"X-Owned": "owned"}
	option := sdk.WithGroupRecordListOpts(sdk.GroupRecordListOpts{Query: map[string]json.RawMessage{"name": raw}, Headers: headers})
	raw[1] = 'X'
	headers["X-Owned"] = "changed"
	if _, err := api.AllUsersInGroupRecords(context.Background(), "g", option); err != nil || calls != 1 {
		t.Fatal(err, calls)
	}
	for _, mode := range []string{"source", "cancel", "nil"} {
		t.Run(mode, func(t *testing.T) {
			before := calls
			seen := 0
			client := api.RawClient()
			endpoint := client.Endpoint
			defer func() { client.Endpoint = endpoint }()
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("group member custom cancel")
			first := sdk.GroupRecordListOption(func(*sdk.GroupRecordListOpts) error {
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
			_, err := api.AllUsersInGroupRecords(ctx, "g", first, sdk.GroupRecordListOption(func(*sdk.GroupRecordListOpts) error { seen++; client.Endpoint = endpoint; return nil }))
			var proof *resource.ResponseError
			if err == nil || seen != 0 || calls != before || errors.As(err, &proof) {
				t.Fatal(err, seen, calls, proof)
			}
			if mode == "cancel" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
		})
	}
}

type groupMemberFaultBody struct {
	first             bool
	closes            *int
	readErr, closeErr error
	read, close       func()
}

func (b *groupMemberFaultBody) Read(p []byte) (int, error) {
	if b.first {
		return 0, io.EOF
	}
	b.first = true
	if b.read != nil {
		b.read()
	}
	n := copy(p, `{"users":`)
	if b.readErr != nil {
		return n, b.readErr
	}
	return n, io.EOF
}
func (b *groupMemberFaultBody) Close() error {
	*b.closes++
	if b.close != nil {
		b.close()
	}
	return b.closeErr
}
func TestPythonGroupUserRecordsPhysicalFaultsKeepActualProofAndStopRetry(t *testing.T) {
	for _, code := range []int{200, 400, 403, 404, 401, 429, 503} {
		for _, phase := range []string{"read", "close", "source", "cancel"} {
			t.Run(fmt.Sprintf("%d/%s", code, phase), func(t *testing.T) {
				calls, closes, hooks := 0, 0, 0
				cause := errors.New("group member physical fault")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				var api *sdk.API
				api, fixture := groupUserAPI(t, func(*http.Request) *http.Response {
					calls++
					b := &groupMemberFaultBody{closes: &closes}
					switch phase {
					case "read":
						b.readErr = cause
					case "close":
						b.closeErr = cause
					case "source":
						b.read = func() { api.RawClient().Endpoint += "changed" }
						b.close = func() { api.RawClient().Endpoint = strings.TrimSuffix(api.RawClient().Endpoint, "changed") }
					case "cancel":
						b.read = func() { cancel(cause) }
					}
					response := groupUserWire(code, "")
					response.Body = b
					return response
				})
				fixture.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks++
					return nil
				}
				fixture.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error { hooks++; return nil }
				fixture.Provider.ReauthFunc = func(context.Context) error { hooks++; return nil }
				fixture.Provider.MaxBackoffRetries = 1
				row, err := api.AllUsersInGroupRecords(ctx, "g")
				if err == nil || row != nil || calls != 1 || closes != 1 || hooks != 0 {
					t.Fatal(row, err, calls, closes, hooks)
				}
				if code < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != code || proof.Header.Get("X-Request-Id") != "group-user-proof" || string(proof.Body) != `{"users":` {
						t.Fatal(err, proof)
					}
				} else {
					original := err
					if code == 401 {
						var auth *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &auth) || auth.ErrOriginal == nil || auth.ErrReauth == nil {
							t.Fatal(err, auth)
						}
						original = auth.ErrOriginal
					}
					var proof gophercloud.ErrUnexpectedResponseCode
					if !errors.As(original, &proof) || proof.Actual != code || proof.ResponseHeader.Get("X-Request-Id") != "group-user-proof" || string(proof.Body) != `{"users":` || proof.Method != "GET" || !strings.Contains(proof.URL, "/keystone/v3/groups/g/users") {
						t.Fatal(err, proof)
					}
				}
				if code != 401 && phase != "source" && !errors.Is(err, cause) {
					t.Fatal(err, cause)
				}
				if code == 401 {
					var reauth *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &reauth) || reauth.ErrReauth == nil {
						t.Fatal(err, reauth)
					}
					// The native 401 wrapper has no Unwrap: its original response
					// and failed reauthentication are independently inspectable.
					// Gophercloud stores ErrOriginal as a response-error value.
					var original gophercloud.ErrUnexpectedResponseCode
					if !errors.As(reauth.ErrOriginal, &original) || original.Actual != 401 || original.Method != "GET" || !strings.Contains(original.URL, "/groups/g/users") || string(original.Body) != `{"users":` || original.ResponseHeader.Get("X-Request-Id") != "group-user-proof" {
						t.Fatal(err, original)
					}
					if phase == "source" {
						if !errors.Is(reauth.ErrReauth, resource.ErrInvalidOption) {
							t.Fatal(err, reauth.ErrReauth)
						}
					} else if !errors.Is(reauth.ErrReauth, cause) {
						t.Fatal(err, reauth.ErrReauth, cause)
					}
					if phase == "cancel" && !errors.Is(reauth.ErrReauth, context.Canceled) {
						t.Fatal(err, reauth.ErrReauth)
					}
				}
			})
		}
	}
}
func TestPythonGroupUserRecordsCleanRetryKeepsGroupScopeAndLiveAuth(t *testing.T) {
	for _, code := range []int{401, 503, 429, 498} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls, reauth, retry, backoff := 0, 0, 0, 0
			api, fixture := groupUserAPI(t, func(r *http.Request) *http.Response {
				calls++
				if r.Method != "GET" || r.Body != nil || r.Header.Get("X-Snapshot") != "before" || r.Header.Get("X-Late") != "" {
					t.Fatal(r.Header, r.Body)
				}
				if calls == 1 {
					if r.Header.Get("X-Auth-Token") != "test-token" {
						t.Fatal(r.Header)
					}
					return groupUserWire(code, `{"error":"native clean"}`)
				}
				if r.Header.Get("X-Auth-Token") != "refreshed" {
					t.Fatal(r.Header)
				}
				return groupUserWire(203, `{"users":[{"id":"a"}]}`)
			})
			api.RawClient().MoreHeaders = map[string]string{"X-Snapshot": "before"}
			update := func() {
				fixture.Provider.SetToken("refreshed")
				api.RawClient().MoreHeaders["X-Snapshot"] = "after"
				api.RawClient().MoreHeaders["X-Late"] = "after"
			}
			fixture.Provider.ReauthFunc = func(context.Context) error { reauth++; update(); return nil }
			fixture.Provider.RetryFunc = func(ctx context.Context, method, u string, opts *gophercloud.RequestOpts, _ error, count uint) error {
				retry++
				update()
				return nil
			}
			fixture.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				backoff++
				update()
				return nil
			}
			fixture.Provider.MaxBackoffRetries = 1
			rows, err := api.AllUsersInGroupRecords(context.Background(), "g")
			if err != nil || len(rows) != 1 || rows[0].StatusCode != 203 || calls != 2 {
				t.Fatal(rows, err, calls)
			}
			if code == 401 && (reauth != 1 || retry != 0 || backoff != 0) || code == 503 && (retry != 1 || reauth != 0 || backoff != 0) || (code == 429 || code == 498) && (backoff != 1 || reauth != 0 || retry != 0) {
				t.Fatal(reauth, retry, backoff)
			}
		})
	}
}

func TestPythonGroupUserRecordsFormalRestorationUsesFullUserSchema(t *testing.T) {
	calls := 0
	api, _ := groupUserAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.URL.Path != "/keystone/v3/groups/g/users" || r.URL.RawQuery != "domain_id=restored&enabled=False" || r.Header.Get("X-Raw") != "owned" || r.Header.Get("OpenStack-API-Version") != "identity 3.7" {
			t.Fatal(r.URL, r.Header)
		}
		return groupUserWire(200, `{"users":[{"id":"u","email":"kept"}]}`)
	})
	rows, err := api.AllUsersInGroupRecords(context.Background(), "g", sdk.WithGroupRecordListQuery("domain_id", "original"), sdk.WithGroupRecordListQuery("__conflicting_attrs", json.RawMessage(`{"domain_id":"restored","is_enabled":false,"email":"kept","headers":{"X-Raw":"owned"},"microversion":"3.7","unknown":{"ignored":true}}`)))
	if err != nil || len(rows) != 1 || calls != 1 {
		t.Fatal(rows, err, calls)
	}
}
