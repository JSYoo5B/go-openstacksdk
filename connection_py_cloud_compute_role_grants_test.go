package openstack_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pyCloudComputeGrantCloud struct {
	mu      sync.Mutex
	calls   []string
	replies map[string]int
}

func (g *pyCloudComputeGrantCloud) log() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return strings.Join(g.calls, "|")
}

func pyCloudComputeGrantConnection(t *testing.T, replies map[string]int) (*sdk.Connection, *pyCloudComputeGrantCloud) {
	t.Helper()
	cloud := testcloud.New(t)
	g := &pyCloudComputeGrantCloud{replies: replies}
	cloud.Mux.HandleFunc("/keystone/v3/", func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + strings.TrimPrefix(r.URL.Path, "/keystone/v3")
		g.mu.Lock()
		g.calls = append(g.calls, key+map[bool]string{true: "?" + r.URL.RawQuery, false: ""}[r.URL.RawQuery != ""])
		status, ok := g.replies[key]
		g.mu.Unlock()
		if !ok {
			status = 404
		}
		path := strings.TrimPrefix(r.URL.Path, "/keystone/v3")
		switch {
		case r.Method == http.MethodGet && status == 200 && strings.Count(path, "/") == 2:
			kind := strings.Split(path, "/")[1]
			single := map[string]string{"domains": "domain", "roles": "role", "users": "user", "groups": "group", "projects": "project"}[kind]
			testcloud.JSON(w, 200, `{"`+single+`":{"id":"`+strings.Split(path, "/")[2]+`-id","name":"`+strings.Split(path, "/")[2]+`"}}`)
		case r.Method == http.MethodGet && status == 200:
			kind := strings.Trim(path, "/")
			name := r.URL.Query().Get("name")
			testcloud.JSON(w, 200, `{"`+kind+`":[{"id":"`+name+`-id","name":"`+name+`"}]}`)
		case r.Method == http.MethodGet && strings.Count(path, "/") == 2:
			testcloud.JSON(w, 404, `{"error":{"message":"missing"}}`)
		case r.Method == http.MethodGet:
			testcloud.JSON(w, 200, `{"`+strings.Trim(path, "/")+`":[]}`)
		default:
			w.WriteHeader(status)
		}
	})
	c, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Identity, cloud.Server.URL+"/keystone/v3"))
	if err != nil {
		t.Fatal(err)
	}
	return c, g
}

func TestPyCloudComputeGrantRoleProjectUserChecksThenAssigns(t *testing.T) {
	c, g := pyCloudComputeGrantConnection(t, map[string]int{
		"GET /domains/dom": 200,
		"GET /roles":       200, "GET /users": 200, "GET /projects": 200,
		"HEAD /projects/proj-id/users/alice-id/roles/admin-id": 404,
		"PUT /projects/proj-id/users/alice-id/roles/admin-id":  204,
	})
	ok, err := c.GrantRole(context.Background(), "admin", sdk.RoleAssignmentTarget{User: "alice", Project: "proj", Domain: "dom"})
	if err != nil || !ok {
		t.Fatalf("grant: %v %v", ok, err)
	}
	want := "GET /domains/dom|GET /roles/admin|GET /roles?name=admin|GET /users/alice?domain_id=dom-id|GET /users?domain_id=dom-id&name=alice|GET /projects/proj?domain_id=dom-id|GET /projects?domain_id=dom-id&name=proj|HEAD /projects/proj-id/users/alice-id/roles/admin-id|PUT /projects/proj-id/users/alice-id/roles/admin-id"
	if got := g.log(); got != want {
		t.Fatalf("sequence:\n got %s\nwant %s", got, want)
	}
}

func TestPyCloudComputeGrantRoleExistingAssignmentWritesNothing(t *testing.T) {
	c, g := pyCloudComputeGrantConnection(t, map[string]int{
		"GET /roles/admin": 200, "GET /groups/ops": 200,
		"HEAD /domains/dom/groups/ops/roles/admin": 204,
	})
	ok, err := c.GrantRole(context.Background(), "admin", sdk.RoleAssignmentTarget{Group: "ops", Domain: "dom"})
	if err == nil {
		t.Fatalf("domain dom is not stubbed: %v", ok)
	}
	c, g = pyCloudComputeGrantConnection(t, map[string]int{
		"GET /domains/dom": 200, "GET /roles/admin": 200, "GET /groups/ops": 200,
		"HEAD /domains/dom-id/groups/ops-id/roles/admin-id": 204,
	})
	ok, err = c.GrantRole(context.Background(), "admin", sdk.RoleAssignmentTarget{Group: "ops", Domain: "dom"})
	if err != nil || ok {
		t.Fatalf("existing assignment: %v %v", ok, err)
	}
	if strings.Contains(g.log(), "PUT") {
		t.Fatalf("existing assignment was written again: %s", g.log())
	}
}

func TestPyCloudComputeRevokeRoleSystemInheritedAndAbsent(t *testing.T) {
	c, g := pyCloudComputeGrantConnection(t, map[string]int{
		"GET /roles/admin": 200, "GET /users/alice": 200,
		"HEAD /system/users/alice-id/roles/admin-id":   204,
		"DELETE /system/users/alice-id/roles/admin-id": 204,
	})
	ok, err := c.RevokeRole(context.Background(), "admin", sdk.RoleAssignmentTarget{User: "alice", System: true})
	if err != nil || !ok || !strings.HasSuffix(g.log(), "HEAD /system/users/alice-id/roles/admin-id|DELETE /system/users/alice-id/roles/admin-id") {
		t.Fatalf("system revoke: %v %v %s", ok, err, g.log())
	}
	c, g = pyCloudComputeGrantConnection(t, map[string]int{
		"GET /roles/admin": 200, "GET /users/alice": 200, "GET /projects/proj": 200,
		"HEAD /OS-INHERIT/projects/proj-id/users/alice-id/roles/admin-id/inherited_to_projects": 404,
	})
	ok, err = c.RevokeRole(context.Background(), "admin", sdk.RoleAssignmentTarget{User: "alice", Project: "proj", Inherited: true})
	if err != nil || ok || strings.Contains(g.log(), "DELETE") || !strings.Contains(g.log(), "HEAD /OS-INHERIT/projects/proj-id/users/alice-id/roles/admin-id/inherited_to_projects") {
		t.Fatalf("absent inherited revoke: %v %v %s", ok, err, g.log())
	}
}

func TestPyCloudComputeGrantRoleRejectsBadTargetsAndMissingRows(t *testing.T) {
	c, g := pyCloudComputeGrantConnection(t, map[string]int{"GET /roles/admin": 200})
	ctx := context.Background()
	for name, target := range map[string]sdk.RoleAssignmentTarget{
		"both":     {User: "u", Group: "g", Project: "p"},
		"neither":  {Project: "p"},
		"no scope": {User: "u"},
	} {
		if _, err := c.GrantRole(ctx, "admin", target); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if _, err := c.GrantRole(ctx, "", sdk.RoleAssignmentTarget{User: "u", Project: "p"}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if g.log() != "" {
		t.Fatalf("invalid input sent requests: %s", g.log())
	}
	if _, err := c.GrantRole(ctx, "admin", sdk.RoleAssignmentTarget{User: "ghost", Project: "p"}); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing user: %v", err)
	}
	if strings.Contains(g.log(), "PUT") || strings.Contains(g.log(), "HEAD") {
		t.Fatalf("assignment request after failed lookup: %s", g.log())
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := c.RevokeRole(canceled, "admin", sdk.RoleAssignmentTarget{User: "u", Project: "p"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestPyCloudComputeGrantRoleSurfacesNon404ValidationAndWriteFailures(t *testing.T) {
	c, g := pyCloudComputeGrantConnection(t, map[string]int{
		"GET /roles/admin": 200, "GET /users/alice": 200, "GET /projects/proj": 200,
		"HEAD /projects/proj-id/users/alice-id/roles/admin-id": 403,
	})
	if ok, err := c.GrantRole(context.Background(), "admin", sdk.RoleAssignmentTarget{User: "alice", Project: "proj"}); err == nil || ok || strings.Contains(g.log(), "PUT") {
		t.Fatalf("403 HEAD must not be treated as absent: %v %v %s", ok, err, g.log())
	}
	c, _ = pyCloudComputeGrantConnection(t, map[string]int{
		"GET /roles/admin": 200, "GET /users/alice": 200, "GET /projects/proj": 200,
		"PUT /projects/proj-id/users/alice-id/roles/admin-id": 403,
	})
	if ok, err := c.GrantRole(context.Background(), "admin", sdk.RoleAssignmentTarget{User: "alice", Project: "proj"}); err == nil || ok {
		t.Fatalf("failed PUT must not report success: %v %v", ok, err)
	}
}
