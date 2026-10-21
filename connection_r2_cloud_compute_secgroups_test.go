package openstack_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func r2CloudComputeSecGroupFixture(t *testing.T, removeStatus map[string]int) *r2CloudComputeCloud {
	t.Helper()
	cloud := r2CloudComputeNew(t)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/s1", 200, `{"server":{"id":"s1","name":"web-1","addresses":{}}}`)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/absent", 404, `{"itemNotFound":{"message":"nf","code":404}}`)
	cloud.JSON("GET "+r2CloudComputeBase+"/servers/detail", 200, `{"servers":[]}`)
	cloud.Handle("GET /network/v2.0/security-groups/{id}", func(w http.ResponseWriter, _ *http.Request, _ string) {
		testcloud.JSON(w, 404, `{"NeutronError":{"message":"nf","type":"SecurityGroupNotFound"}}`)
	})
	cloud.JSON("GET /network/v2.0/security-groups", 200, `{"security_groups":[{"id":"sg-a","name":"alpha"},{"id":"sg-b","name":""},{"id":"sg-c","name":"gamma"}]}`)
	cloud.Handle("POST "+r2CloudComputeBase+"/servers/s1/action", func(w http.ResponseWriter, _ *http.Request, body string) {
		for name, status := range removeStatus {
			if strings.Contains(body, `"`+name+`"`) {
				testcloud.JSON(w, status, `{"itemNotFound":{"message":"nf","code":404}}`)
				return
			}
		}
		w.WriteHeader(202)
	})
	return cloud
}

func r2CloudComputeActionBodies(cloud *r2CloudComputeCloud) []string {
	var out []string
	for _, call := range cloud.Calls() {
		if call.Method == http.MethodPost {
			out = append(out, call.Body)
		}
	}
	return out
}

func TestR2CloudComputeAddServerSecurityGroupsResolvesAllThenSendsInOrder(t *testing.T) {
	cloud := r2CloudComputeSecGroupFixture(t, nil)
	ok, err := cloud.Conn.AddServerSecurityGroups(context.Background(), "s1", "alpha", "sg-b", "gamma")
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	want := `{"addSecurityGroup":{"name":"alpha"}}|{"addSecurityGroup":{"name":"sg-b"}}|{"addSecurityGroup":{"name":"gamma"}}`
	if got := strings.Join(r2CloudComputeActionBodies(cloud), "|"); got != want {
		t.Fatal(got)
	}
}

func TestR2CloudComputeSecurityGroupsMissingServerOrGroupSendsNoAction(t *testing.T) {
	cloud := r2CloudComputeSecGroupFixture(t, nil)
	ctx := context.Background()
	for _, call := range []func() (bool, error){
		func() (bool, error) { return cloud.Conn.AddServerSecurityGroups(ctx, "absent", "alpha") },
		func() (bool, error) { return cloud.Conn.AddServerSecurityGroups(ctx, "s1", "alpha", "missing") },
		func() (bool, error) { return cloud.Conn.RemoveServerSecurityGroups(ctx, "absent", "alpha") },
		func() (bool, error) { return cloud.Conn.RemoveServerSecurityGroups(ctx, "s1", "missing", "alpha") },
		func() (bool, error) { return cloud.Conn.AddServerSecurityGroups(ctx, "s1") },
	} {
		if ok, err := call(); ok || err != nil {
			t.Fatal(ok, err)
		}
	}
	if bodies := r2CloudComputeActionBodies(cloud); len(bodies) != 0 {
		t.Fatal(bodies)
	}
}

func TestR2CloudComputeSecurityGroupsRejectEmptyIdentitiesBeforeHTTP(t *testing.T) {
	cloud := r2CloudComputeSecGroupFixture(t, nil)
	if _, err := cloud.Conn.AddServerSecurityGroups(context.Background(), "", "alpha"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := cloud.Conn.RemoveServerSecurityGroups(context.Background(), "s1", ""); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if len(cloud.Calls()) != 0 {
		t.Fatal(cloud.Calls())
	}
}

func TestR2CloudComputeRemoveServerSecurityGroupsContinuesPast404AndReportsFalse(t *testing.T) {
	cloud := r2CloudComputeSecGroupFixture(t, map[string]int{"alpha": 404})
	ok, err := cloud.Conn.RemoveServerSecurityGroups(context.Background(), "s1", "alpha", "gamma")
	if err != nil || ok {
		t.Fatal(ok, err)
	}
	if bodies := r2CloudComputeActionBodies(cloud); len(bodies) != 2 || bodies[1] != `{"removeSecurityGroup":{"name":"gamma"}}` {
		t.Fatal(bodies)
	}
	cloud = r2CloudComputeSecGroupFixture(t, nil)
	if ok, err = cloud.Conn.RemoveServerSecurityGroups(context.Background(), "s1", "alpha"); err != nil || !ok {
		t.Fatal(ok, err)
	}
}

func TestR2CloudComputeSecurityGroupActionFailureStopsSequenceWithoutReplay(t *testing.T) {
	cloud := r2CloudComputeSecGroupFixture(t, map[string]int{"alpha": 500})
	ok, err := cloud.Conn.AddServerSecurityGroups(context.Background(), "s1", "alpha", "gamma")
	if err == nil || ok || len(r2CloudComputeActionBodies(cloud)) != 1 {
		t.Fatal(ok, err, r2CloudComputeActionBodies(cloud))
	}
	cloud = r2CloudComputeSecGroupFixture(t, map[string]int{"alpha": 500})
	ok, err = cloud.Conn.RemoveServerSecurityGroups(context.Background(), "s1", "alpha", "gamma")
	if err == nil || ok || len(r2CloudComputeActionBodies(cloud)) != 1 {
		t.Fatal(ok, err)
	}
}
