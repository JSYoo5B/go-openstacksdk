package api_test

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"

	"github.com/gophercloud/gophercloud/v2"
)

const pyCoreMiscTokenBody = `{"token":{"user":{"id":"user-1","name":"alice"},"catalog":[{"id":"s1","name":"nova","type":"compute","endpoints":[{"id":"e1","interface":"public","region":"r1","region_id":"r1","url":"https://nova.invalid/v2.1"}]}]}}`

func pyCoreMiscKeystone(t *testing.T, posts *atomic.Int32) string {
	t.Helper()
	srv := testcloud.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v3/auth/tokens" {
			http.NotFound(w, r)
			return
		}
		posts.Add(1)
		w.Header().Set("X-Subject-Token", "issued-token")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(pyCoreMiscTokenBody))
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/v3/"
}

func TestPyCoreMiscConnectionRecordsTokenUserAndCatalogWithoutRequests(t *testing.T) {
	var posts atomic.Int32
	endpoint := pyCoreMiscKeystone(t, &posts)
	conn, err := sdk.Connect(context.Background(), sdk.WithAuth(gophercloud.AuthOptions{IdentityEndpoint: endpoint, Username: "alice", Password: "synthetic", DomainID: "default"}))
	if err != nil {
		t.Fatal(err)
	}
	if posts.Load() != 1 {
		t.Fatalf("connect posts: %d", posts.Load())
	}
	if conn.AuthToken() != "issued-token" {
		t.Fatalf("token: %q", conn.AuthToken())
	}
	token, err := conn.Authorize(context.Background())
	if err != nil || token != "issued-token" {
		t.Fatalf("authorize: %q %v", token, err)
	}
	id, err := conn.CurrentUserID()
	if err != nil || id != "user-1" {
		t.Fatalf("user: %q %v", id, err)
	}
	catalog, err := conn.ServiceCatalog()
	if err != nil || len(catalog) != 1 || catalog[0].Type != "compute" || len(catalog[0].Endpoints) != 1 || catalog[0].Endpoints[0].URL != "https://nova.invalid/v2.1" || catalog[0].Endpoints[0].Interface != "public" {
		t.Fatalf("catalog: %+v %v", catalog, err)
	}
	catalog[0].Endpoints[0].URL = "mutated"
	again, _ := conn.ServiceCatalog()
	if again[0].Endpoints[0].URL == "mutated" {
		t.Fatal("catalog aliases recorded response")
	}
	if posts.Load() != 1 {
		t.Fatalf("accessors sent requests: %d", posts.Load())
	}
}

func TestPyCoreMiscConnectionWithoutAuthResultReportsIt(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	conn, err := sdk.FromProvider(provider)
	if err != nil {
		t.Fatal(err)
	}
	if conn.AuthToken() != "" {
		t.Fatal("unexpected token")
	}
	if _, err := conn.CurrentUserID(); !errors.Is(err, sdk.ErrNoAuthResult) {
		t.Fatalf("user: %v", err)
	}
	if _, err := conn.ServiceCatalog(); !errors.Is(err, sdk.ErrNoAuthResult) {
		t.Fatalf("catalog: %v", err)
	}
	if _, err := conn.Authorize(context.Background()); err == nil {
		t.Fatal("authorize without token or reauth must fail")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := conn.Authorize(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
}

func TestPyCoreMiscAuthorizeReauthenticatesOnlyWhenTokenMissing(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	var calls int
	provider.ReauthFunc = func(context.Context) error {
		calls++
		provider.SetToken("fresh")
		return nil
	}
	conn, err := sdk.FromProvider(provider)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		token, err := conn.Authorize(context.Background())
		if err != nil || token != "fresh" {
			t.Fatalf("authorize: %q %v", token, err)
		}
	}
	if calls != 1 {
		t.Fatalf("reauth calls: %d", calls)
	}
}
