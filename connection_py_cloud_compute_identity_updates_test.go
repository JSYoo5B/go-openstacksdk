package openstack_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type pyCloudComputeIdentityLog struct {
	mu    sync.Mutex
	calls []string
}

func (l *pyCloudComputeIdentityLog) add(s string) {
	l.mu.Lock()
	l.calls = append(l.calls, s)
	l.mu.Unlock()
}

func (l *pyCloudComputeIdentityLog) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.calls, "|")
}

func pyCloudComputeIdentityConnection(t *testing.T) (*sdk.Connection, *pyCloudComputeIdentityLog) {
	t.Helper()
	cloud := testcloud.New(t)
	log := &pyCloudComputeIdentityLog{}
	cloud.Mux.HandleFunc("/keystone/v3/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		line := r.Method + " " + strings.TrimPrefix(r.URL.Path, "/keystone/v3")
		if r.URL.RawQuery != "" {
			line += "?" + r.URL.RawQuery
		}
		if len(body) > 0 {
			line += " " + string(body)
		}
		log.add(line)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/keystone/v3/domains/finance":
			testcloud.JSON(w, 404, `{}`)
		case r.Method == http.MethodGet && r.URL.Path == "/keystone/v3/domains":
			testcloud.JSON(w, 200, `{"domains":[{"id":"d1","name":"finance"}]}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/keystone/v3/domains/d1":
			testcloud.JSON(w, 200, `{"domain":{"id":"d1","name":"finance","enabled":false,"description":""}}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/keystone/v3/domains/d9":
			testcloud.JSON(w, 200, `{"domain":{"id":"d9","name":"x","enabled":true}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/keystone/v3/groups/ops":
			testcloud.JSON(w, 404, `{}`)
		case r.Method == http.MethodGet && r.URL.Path == "/keystone/v3/groups":
			if r.URL.Query().Get("name") == "ops" {
				testcloud.JSON(w, 200, `{"groups":[{"id":"g1","name":"ops","description":"old","domain_id":"d1"}]}`)
				return
			}
			testcloud.JSON(w, 200, `{"groups":[]}`)
		case r.Method == http.MethodPatch && r.URL.Path == "/keystone/v3/groups/g1":
			testcloud.JSON(w, 200, `{"group":{"id":"g1","name":"ops","description":"new"}}`)
		default:
			testcloud.JSON(w, 404, `{}`)
		}
	})
	c, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Identity, cloud.Server.URL+"/keystone/v3"))
	if err != nil {
		t.Fatal(err)
	}
	return c, log
}

func TestPyCloudComputeUpdateDomainIDSkipsLookupAndDropsEmptyName(t *testing.T) {
	c, log := pyCloudComputeIdentityConnection(t)
	off := false
	empty := ""
	got, err := c.UpdateDomain(context.Background(), resource.ID("d9"), sdk.DomainUpdate{Enabled: &off, Description: &empty})
	if err != nil || got.ID != "d9" {
		t.Fatalf("update: %+v %v", got, err)
	}
	if s := log.String(); s != `PATCH /domains/d9 {"domain":{"description":"","enabled":false}}` {
		t.Fatalf("requests: %s", s)
	}
}

func TestPyCloudComputeUpdateDomainByNameLooksUpAndNoopSendsNothing(t *testing.T) {
	c, log := pyCloudComputeIdentityConnection(t)
	if got, err := c.UpdateDomain(context.Background(), resource.Name("finance"), sdk.DomainUpdate{Name: "finance"}); err != nil || got.ID != "d1" {
		t.Fatalf("update: %+v %v", got, err)
	}
	if got, err := c.UpdateDomain(context.Background(), resource.Name("finance"), sdk.DomainUpdate{}); err != nil || got.ID != "d1" {
		t.Fatalf("noop: %+v %v", got, err)
	}
	s := log.String()
	if strings.Count(s, "PATCH") != 1 || !strings.Contains(s, "GET /domains/finance|GET /domains?name=finance|PATCH /domains/d1 ") {
		t.Fatalf("requests: %s", s)
	}
	if _, err := c.UpdateDomain(context.Background(), resource.Name("ghost"), sdk.DomainUpdate{Name: "x"}); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	if strings.Count(log.String(), "PATCH") != 1 {
		t.Fatal("PATCH sent for a missing domain")
	}
}

func TestPyCloudComputeUpdateGroupSendsOnlyChangedAttributes(t *testing.T) {
	c, log := pyCloudComputeIdentityConnection(t)
	same := "old"
	if got, err := c.UpdateGroup(context.Background(), "ops", sdk.GroupUpdate{Name: "ops", Description: &same}); err != nil || got.ID != "g1" || got.Description != "old" {
		t.Fatalf("unchanged: %+v %v", got, err)
	}
	if strings.Contains(log.String(), "PATCH") {
		t.Fatalf("unchanged values were sent: %s", log.String())
	}
	changed := "new"
	got, err := c.UpdateGroup(context.Background(), "ops", sdk.GroupUpdate{Name: "ops", Description: &changed}, resource.WithIdentityFindQuery("domain_id", "d1"))
	if err != nil || got.Description != "new" {
		t.Fatalf("changed: %+v %v", got, err)
	}
	s := log.String()
	if !strings.Contains(s, "GET /groups/ops?domain_id=d1|GET /groups?domain_id=d1&name=ops|PATCH /groups/g1 {\"group\":{\"description\":\"new\"}}") {
		t.Fatalf("requests: %s", s)
	}
}

func TestPyCloudComputeUpdateGroupMissingAndCanceled(t *testing.T) {
	c, log := pyCloudComputeIdentityConnection(t)
	if _, err := c.UpdateGroup(context.Background(), "ghost", sdk.GroupUpdate{Name: "x"}); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	before := log.String()
	if _, err := c.UpdateGroup(ctx, "ops", sdk.GroupUpdate{Name: "x"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := c.UpdateDomain(ctx, resource.ID("d1"), sdk.DomainUpdate{Name: "x"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if log.String() != before || strings.Contains(log.String(), "PATCH") {
		t.Fatalf("requests after cancel: %s", log.String())
	}
}
