package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/identity/v2/tokens"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestIdentityV2AuthenticationRetainsEveryResponseView(t *testing.T) {
	cloud := testcloud.New(t)
	const body = `{"access":{"token":{"id":"issued-token","expires":"2030-01-02T03:04:05Z","tenant":{"id":"project","name":"dev"}},"user":{"id":"user","name":"alice","roles":[{"name":"member"}]},"serviceCatalog":[{"name":"nova","type":"compute","endpoints":[{"publicURL":"https://nova.invalid/v2/project","region":"region"}]}],"vendor:access":{"enabled":false}}}`
	posts, gets := 0, 0
	cloud.Mux.HandleFunc("/identity/v2/tokens", func(w http.ResponseWriter, r *http.Request) {
		posts++
		if r.Method != http.MethodPost {
			t.Error(r.Method)
		}
		var input struct {
			Auth map[string]any `json:"auth"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		if input.Auth["tenantId"] != "project" || input.Auth["vendor:enabled"] != false {
			t.Errorf("input=%v", input.Auth)
		}
		w.Header().Set("X-Openstack-Request-Id", "req-create")
		testcloud.JSON(w, http.StatusOK, body)
	})
	cloud.Mux.HandleFunc("/identity/v2/tokens/issued-token", func(w http.ResponseWriter, r *http.Request) {
		gets++
		if r.Method != http.MethodGet || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Errorf("method=%s auth=%q", r.Method, r.Header.Get("X-Auth-Token"))
		}
		w.Header().Set("X-Openstack-Request-Id", "req-get")
		testcloud.JSON(w, http.StatusNonAuthoritativeInfo, body)
	})
	api := tokens.New(cloud.Client("identity", "/identity/v2"))
	ctx := context.Background()
	auth, err := api.Create(ctx, tokens.AuthOptions{TokenID: "initial-token", TenantID: "project"}, tokens.WithCreateField("vendor:enabled", false))
	if err != nil {
		t.Fatal(err)
	}
	check := func(result *tokens.Authentication, requestID string) {
		t.Helper()
		if result.Token.ID != "issued-token" || result.Token.Tenant.ID != "project" || !result.Token.ExpiresAt.Equal(time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)) {
			t.Fatalf("token=%+v", result.Token)
		}
		if result.User.ID != "user" || len(result.User.Roles) != 1 || result.User.Roles[0].Name != "member" {
			t.Fatalf("user=%+v", result.User)
		}
		if len(result.Catalog.Entries) != 1 || result.Catalog.Entries[0].Endpoints[0].Region != "region" || result.Header.Get("X-Openstack-Request-Id") != requestID {
			t.Fatalf("catalog=%+v header=%v", result.Catalog, result.Header)
		}
		var response struct {
			Access map[string]json.RawMessage `json:"access"`
		}
		if err := json.Unmarshal(result.Body, &response); err != nil || string(response.Access["vendor:access"]) != `{"enabled":false}` {
			t.Fatalf("body=%s err=%v", result.Body, err)
		}
	}
	check(auth, "req-create")
	got, err := api.Get(ctx, "issued-token")
	if err != nil {
		t.Fatal(err)
	}
	check(got, "req-get")
	if posts != 1 || gets != 1 {
		t.Fatalf("posts=%d gets=%d", posts, gets)
	}
	if result, err := api.Create(ctx, tokens.AuthOptions{TokenID: "initial-token"}, tokens.WithCreateField("tenantId", "override")); result != nil || !errors.Is(err, resource.ErrInvalidOption) || posts != 1 {
		t.Fatalf("result=%v err=%v posts=%d", result, err, posts)
	}
}

func TestIdentityV2AuthenticationPropagatesHTTPAndDecodeErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
	}{
		{"forbidden", `{"error":"forbidden"}`, 403},
		{"invalid-expiry", `{"access":{"token":{"id":"token","expires":"invalid"}}}`, 200},
		{"invalid-user", `{"access":{"token":{"id":"token","expires":"2030-01-02T03:04:05Z"},"user":{"roles":false}}}`, 200},
		{"invalid-catalog", `{"access":{"token":{"id":"token","expires":"2030-01-02T03:04:05Z"},"serviceCatalog":false}}`, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/identity/tokens/token", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, tc.status, tc.body) })
			value, err := tokens.New(cloud.Client("identity", "/identity")).Get(context.Background(), "token")
			var operation *resource.OperationError
			if value != nil || err == nil || !errors.As(err, &operation) {
				t.Fatalf("value=%v err=%v", value, err)
			}
			if tc.status == 403 {
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) || response.Actual != 403 {
					t.Fatal(err)
				}
			}
		})
	}
}
