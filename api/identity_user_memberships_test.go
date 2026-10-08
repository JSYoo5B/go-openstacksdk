package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"reflect"
	"strconv"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

func checkMembershipRows[T any](t *testing.T, sequence iter.Seq2[*T, error], wantType reflect.Type, envelope string) {
	t.Helper()
	count := 0
	for value, err := range sequence {
		if err != nil {
			t.Fatal(err)
		}
		if reflect.TypeOf(value) != wantType {
			t.Fatalf("%s row type=%T, want %v", envelope, value, wantType)
		}
		body, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]any
		if err := json.Unmarshal(body, &fields); err != nil {
			t.Fatal(err)
		}
		count++
		if fields["id"] != envelope+strconv.Itoa(count) || fields["domain_id"] != "domain" || fields["name"] != "member" || fields["description"] != "membership" {
			t.Fatalf("%s fields=%s", envelope, body)
		}
		switch row := any(value).(type) {
		case *projects.Project:
			if row.Enabled != (count == 2) || row.ParentID != "parent" || len(row.Tags) != 1 || row.Tags[0] != "access" || row.Options["immutable"] != true || row.Extra["vendor:flag"] != false {
				t.Fatalf("project=%+v", row)
			}
		case *groups.Group:
			if row.Links["self"] != "https://group.invalid/" || row.Extra["vendor:flag"] != false {
				t.Fatalf("group=%+v", row)
			}
		}
	}
	if count != 2 {
		t.Fatalf("%s rows=%d, want 2", envelope, count)
	}
}

func TestIdentityUserMembershipListsDecodeTheirOwnModelsAcrossPages(t *testing.T) {
	for _, envelope := range []string{"projects", "groups"} {
		t.Run(envelope, func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			path := "/prefix/identity/v3/users/user-id/" + envelope
			cloud.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodGet || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Errorf("method=%s token=%q", r.Method, r.Header.Get("X-Auth-Token"))
				}
				if calls == 1 && r.URL.RawQuery != "" || calls == 2 && r.URL.Query().Get("marker") != "next" {
					t.Errorf("unexpected query %q on call %d", r.URL.RawQuery, calls)
				}
				row := map[string]any{"id": envelope + strconv.Itoa(calls), "name": "member", "description": "membership", "domain_id": "domain", "enabled": calls == 2, "parent_id": "parent", "tags": []string{"access"}, "options": map[string]bool{"immutable": true}, "vendor:flag": false, "links": map[string]string{"self": "https://group.invalid/"}}
				var next any
				if calls == 1 {
					next = cloud.Server.URL + path + "?marker=next"
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{envelope: []any{row}, "links": map[string]any{"next": next}})
			})
			api := users.New(cloud.Client("identity", "/prefix/identity/v3"))
			if envelope == "projects" {
				checkMembershipRows(t, api.ListProjects(context.Background(), "user-id"), reflect.TypeFor[*projects.Project](), envelope)
			} else {
				checkMembershipRows(t, api.ListGroups(context.Background(), "user-id"), reflect.TypeFor[*groups.Group](), envelope)
			}
			if calls != 2 {
				t.Fatalf("calls=%d", calls)
			}
		})
	}
}

func TestIdentityUserMembershipListsEmptyAndNoContent(t *testing.T) {
	for _, envelope := range []string{"projects", "groups"} {
		for _, status := range []int{200, 204} {
			t.Run(envelope+"/"+strconv.Itoa(status), func(t *testing.T) {
				cloud := testcloud.New(t)
				calls := 0
				cloud.Mux.HandleFunc("/identity/users/self/"+envelope, func(w http.ResponseWriter, r *http.Request) {
					calls++
					if status == 204 {
						w.WriteHeader(status)
						return
					}
					testcloud.JSON(w, status, `{"`+envelope+`":[],"links":{"next":null}}`)
				})
				api := users.New(cloud.Client("identity", "/identity"))
				if envelope == "projects" {
					for row, err := range api.ListProjects(context.Background(), "self") {
						t.Fatalf("row=%v err=%v", row, err)
					}
				} else {
					for row, err := range api.ListGroups(context.Background(), "self") {
						t.Fatalf("row=%v err=%v", row, err)
					}
				}
				if calls != 1 {
					t.Fatal(calls)
				}
			})
		}
	}
}

func TestIdentityUserMembershipListsRetainRowsBeforeContinuationFailure(t *testing.T) {
	for _, envelope := range []string{"projects", "groups"} {
		for _, scenario := range []string{"forbidden", "decode", "cancel"} {
			t.Run(envelope+"/"+scenario, func(t *testing.T) {
				cloud := testcloud.New(t)
				calls := 0
				path := "/identity/users/self/" + envelope
				cloud.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
					calls++
					if calls == 1 {
						testcloud.JSON(w, 200, `{"`+envelope+`":[{"id":"first"}],"links":{"next":"`+cloud.Server.URL+path+`?marker=next"}}`)
					} else if scenario == "forbidden" {
						testcloud.JSON(w, 403, `{"error":{"message":"later page denied"}}`)
					} else {
						testcloud.JSON(w, 200, `{"`+envelope+`":[{"id":false}],"links":{"next":null}}`)
					}
				})
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				api := users.New(cloud.Client("identity", "/identity"))
				rows, failures := 0, 0
				check := func(id string, err error) {
					t.Helper()
					if err == nil {
						rows++
						if id != "first" {
							t.Fatal(id)
						}
						if scenario == "cancel" {
							cancel()
						}
						return
					}
					failures++
					if scenario == "cancel" && !errors.Is(err, context.Canceled) {
						t.Fatal(err)
					}
					if scenario == "forbidden" {
						var response gophercloud.ErrUnexpectedResponseCode
						if !errors.As(err, &response) || response.Actual != 403 {
							t.Fatal(err)
						}
					}
				}
				if envelope == "projects" {
					for row, err := range api.ListProjects(ctx, "self") {
						id := ""
						if row != nil {
							id = row.ID
						}
						check(id, err)
					}
				} else {
					for row, err := range api.ListGroups(ctx, "self") {
						id := ""
						if row != nil {
							id = row.ID
						}
						check(id, err)
					}
				}
				wantCalls := 2
				if scenario == "cancel" {
					wantCalls = 1
				}
				if rows != 1 || failures != 1 || calls != wantCalls {
					t.Fatalf("rows=%d failures=%d calls=%d", rows, failures, calls)
				}
			})
		}
	}
}

func TestIdentityUserMembershipListsPropagateErrorsAndStopEarly(t *testing.T) {
	for _, envelope := range []string{"projects", "groups"} {
		for _, scenario := range []string{"break", "forbidden", "decode", "cancel"} {
			t.Run(envelope+"/"+scenario, func(t *testing.T) {
				cloud := testcloud.New(t)
				calls := 0
				path := "/identity/users/user-id/" + envelope
				cloud.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
					calls++
					if calls > 1 {
						t.Error("unexpected continuation after consumer break")
					}
					switch scenario {
					case "forbidden":
						testcloud.JSON(w, 403, `{"error":{"message":"membership denied"}}`)
					case "decode":
						testcloud.JSON(w, 200, `{"`+envelope+`":[{"id":false}],"links":{"next":null}}`)
					default:
						testcloud.JSON(w, 200, `{"`+envelope+`":[{"id":"one"},{"id":"two"}],"links":{"next":"`+cloud.Server.URL+path+`?marker=next"}}`)
					}
				})
				api := users.New(cloud.Client("identity", "/identity"))
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if scenario == "cancel" {
					cancel()
				}
				check := func(err error) {
					t.Helper()
					if scenario == "break" {
						if err != nil {
							t.Fatal(err)
						}
						return
					}
					if err == nil {
						t.Fatal("missing error")
					}
					if scenario == "cancel" && !errors.Is(err, context.Canceled) {
						t.Fatal(err)
					}
					if scenario == "forbidden" {
						var response gophercloud.ErrUnexpectedResponseCode
						if !errors.As(err, &response) || response.Actual != 403 {
							t.Fatal(err)
						}
					}
				}
				yields := 0
				if envelope == "projects" {
					for _, err := range api.ListProjects(ctx, "user-id") {
						yields++
						check(err)
						break
					}
				} else {
					for _, err := range api.ListGroups(ctx, "user-id") {
						yields++
						check(err)
						break
					}
				}
				wantCalls := 1
				if scenario == "cancel" {
					wantCalls = 0
				}
				if yields != 1 || calls != wantCalls {
					t.Fatalf("yields=%d calls=%d", yields, calls)
				}
			})
		}
	}
}

func TestIdentityLocalUserAndExtensionPageExtractorsRemainSeparate(t *testing.T) {
	cloud := testcloud.New(t)
	for _, path := range []string{"/identity/v3/users", "/identity/v3/groups/group/users"} {
		cloud.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			testcloud.JSON(w, 200, `{"users":[{"id":"user","name":"alice","email":"alice@example.invalid","default_project_id":"project"}],"links":{"next":null}}`)
		})
	}
	api := users.New(cloud.Client("identity", "/identity/v3"))
	for _, sequence := range []iter.Seq2[*users.User, error]{api.List(context.Background()), api.ListInGroup(context.Background(), "group")} {
		count := 0
		for user, err := range sequence {
			if err != nil || user.ID != "user" || user.DefaultProjectID != "project" || user.Extra["email"] != "alice@example.invalid" {
				t.Fatalf("user=%+v err=%v", user, err)
			}
			count++
		}
		if count != 1 {
			t.Fatal(count)
		}
	}
	cloud.Mux.HandleFunc("/identity/v2/extensions", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"extensions":{"values":[{"alias":"OS-KSADM","name":"admin","description":"wrapped","namespace":"https://example.invalid/","updated":"2013-01-01T00:00:00Z"}]}}`)
	})
	count := 0
	for extension, err := range extensions.New(cloud.Client("identity", "/identity/v2")).List(context.Background()) {
		if err != nil || extension.Alias != "OS-KSADM" || extension.Description != "wrapped" {
			t.Fatalf("extension=%+v err=%v", extension, err)
		}
		count++
	}
	if count != 1 {
		t.Fatal(count)
	}
}
