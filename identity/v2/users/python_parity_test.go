package users_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/identity/v2/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pythonUserTransport func(*http.Request) (*http.Response, error)

func (transport pythonUserTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonUserCall struct{ method, path, query, body string }

func pythonUserAPI(t *testing.T, calls *[]pythonUserCall, reply func(*http.Request) (int, string)) *users.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonUserTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonUserCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v2.0/"
	return users.New(client)
}

const (
	pythonUserBase = "/keystone/v2.0/users"
	pythonUserRow  = `{"id":"u-1","name":"alice","email":"a@example.invalid","enabled":true}`
)

// Python create_user/get_user/update_user use the "user" resource_key; update
// carries the ID in the dirty body because Proxy._update builds the Resource
// from it.
func TestPythonUserCreateGetUpdateRequests(t *testing.T) {
	ctx := context.Background()
	var calls []pythonUserCall
	api := pythonUserAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.Method == http.MethodPut {
			return 200, `{"user":{"id":"u-1","name":"alice","email":"b@example.invalid","enabled":false}}`
		}
		return 200, `{"user":` + pythonUserRow + `}`
	})
	enabled, disabled := true, false
	want := users.User{ID: "u-1", Name: "alice", Email: "a@example.invalid", Enabled: true}
	created, err := api.Create(ctx, users.CreateOpts{Name: "alice", Email: "a@example.invalid", Enabled: &enabled})
	if err != nil || !reflect.DeepEqual(*created, want) {
		t.Fatal(created, err)
	}
	got, err := api.Get(ctx, "u-1")
	if err != nil || !reflect.DeepEqual(*got, want) {
		t.Fatal(got, err)
	}
	updated, err := api.Update(ctx, "u-1", users.UpdateOpts{Email: "b@example.invalid", Enabled: &disabled}, users.WithUpdateField("id", "u-1"))
	if err != nil || updated.Email != "b@example.invalid" || updated.Enabled {
		t.Fatal(updated, err)
	}
	wantCalls := []pythonUserCall{
		{http.MethodPost, pythonUserBase, "", `{"user":{"email":"a@example.invalid","enabled":true,"name":"alice"}}`},
		{http.MethodGet, pythonUserBase + "/u-1", "", ""},
		{http.MethodPut, pythonUserBase + "/u-1", "", `{"user":{"email":"b@example.invalid","enabled":false,"id":"u-1"}}`},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("%+v", calls)
	}
	// Python would POST {"user": {"email": ...}}; Go needs Name or Username.
	calls = nil
	if _, err := api.Create(ctx, users.CreateOpts{Email: "a@example.invalid"}); err == nil || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonUserDeleteIgnoreMissing(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		status  int
		options []resource.LookupOption
		missing bool
	}{
		{"deleted", 204, nil, false},
		{"missing ignored by default", 404, nil, false},
		{"ignore_missing=False", 404, []resource.LookupOption{resource.WithMissingError()}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonUserCall
			api := pythonUserAPI(t, &calls, func(*http.Request) (int, string) { return tc.status, "" })
			err := api.Remove(ctx, resource.ID("u-1"), tc.options...)
			if tc.missing != errors.Is(err, resource.ErrNotFound) || !tc.missing && err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, []pythonUserCall{{http.MethodDelete, pythonUserBase + "/u-1", "", ""}}) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

// pythonFindUser mirrors Resource.find: GET by ID first, then the list matched
// locally by name.
func pythonFindUser(ctx context.Context, api *users.API, nameOrID string, options ...resource.LookupOption) (*users.User, error) {
	found, err := api.Find(ctx, resource.ID(nameOrID), resource.WithIgnoreMissing())
	if err != nil || found != nil {
		return found, err
	}
	return api.Find(ctx, resource.Name(nameOrID), options...)
}

func TestPythonUserFindIDThenNameFallback(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		input   string
		rows    string
		options []resource.LookupOption
		wantID  string
		wantErr error
		calls   int
	}{
		{name: "id hit", input: "u-1", wantID: "u-1", calls: 1},
		{name: "name fallback", input: "alice", rows: pythonUserRow, options: []resource.LookupOption{resource.WithIgnoreMissing()}, wantID: "u-1", calls: 2},
		{name: "missing ignored", input: "ghost", rows: pythonUserRow, options: []resource.LookupOption{resource.WithIgnoreMissing()}, calls: 2},
		{name: "missing strict", input: "ghost", rows: pythonUserRow, wantErr: resource.ErrNotFound, calls: 2},
		{name: "duplicate name", input: "alice", rows: pythonUserRow + `,{"id":"u-2","name":"alice"}`, wantErr: resource.ErrAmbiguous, calls: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonUserCall
			api := pythonUserAPI(t, &calls, func(req *http.Request) (int, string) {
				switch req.URL.Path {
				case pythonUserBase + "/u-1":
					return 200, `{"user":` + pythonUserRow + `}`
				case pythonUserBase:
					return 200, `{"users":[` + tc.rows + `]}`
				}
				return 404, `{"error":{"code":404}}`
			})
			got, err := pythonFindUser(ctx, api, tc.input, tc.options...)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) || got != nil {
					t.Fatal(got, err)
				}
			} else if err != nil || (tc.wantID == "") != (got == nil) || got != nil && got.ID != tc.wantID {
				t.Fatal(got, err)
			}
			want := []pythonUserCall{{http.MethodGet, pythonUserBase + "/" + tc.input, "", ""}, {http.MethodGet, pythonUserBase, "", ""}}[:tc.calls]
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

// Python users() follows users_links and can send limit/marker; the native
// List sends one GET without query and never follows users_links.
func TestPythonUserListSinglePage(t *testing.T) {
	ctx := context.Background()
	var calls []pythonUserCall
	api := pythonUserAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"users":[` + pythonUserRow + `],"users_links":[{"rel":"next","href":"/keystone/v2.0/users?marker=u-1"}]}`
	})
	var ids []string
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"u-1"}) || !reflect.DeepEqual(calls, []pythonUserCall{{http.MethodGet, pythonUserBase, "", ""}}) {
		t.Fatalf("%v %+v", ids, calls)
	}
	calls = nil
	if _, err := api.All(ctx, resource.WithQuery("limit", "1")); !errors.Is(err, resource.ErrUnsupported) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}

func TestPythonUserWaitHelpers(t *testing.T) {
	ctx := context.Background()
	t.Run("status attribute is required", func(t *testing.T) {
		var calls []pythonUserCall
		api := pythonUserAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"user":` + pythonUserRow + `}` })
		// User has no status attribute; Python raises AttributeError before HTTP.
		if _, err := api.WaitFor(ctx, resource.ID("u-1"), "active"); !errors.Is(err, resource.ErrUnsupported) || len(calls) != 0 {
			t.Fatal(err, calls)
		}
	})
	t.Run("deletion polls until 404", func(t *testing.T) {
		var calls []pythonUserCall
		api := pythonUserAPI(t, &calls, func(*http.Request) (int, string) {
			if len(calls) == 1 {
				return 200, `{"user":` + pythonUserRow + `}`
			}
			return 404, ""
		})
		err := api.WaitForDeletion(ctx, resource.ID("u-1"), resource.WithTimeout(2*time.Minute), resource.WithPollInterval(time.Millisecond))
		get := pythonUserCall{http.MethodGet, pythonUserBase + "/u-1", "", ""}
		if err != nil || !reflect.DeepEqual(calls, []pythonUserCall{get, get}) {
			t.Fatal(err, calls)
		}
	})
}
