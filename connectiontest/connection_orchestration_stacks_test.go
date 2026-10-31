package connectiontest_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionStackWorkflowsUseCachedOrchestrationEndpoint(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	cloud.Provider.EndpointLocator = func(o gophercloud.EndpointOpts) (string, error) {
		if o.Type == "orchestration" {
			return cloud.Server.URL + "/heat/", nil
		}
		return "", gophercloud.ErrEndpointNotFound{}
	}
	cloud.Mux.HandleFunc("GET /heat/stacks", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"stacks":[{"id":"1","stack_name":"app","stack_status":"CREATE_COMPLETE"}]}`)
	})
	cloud.Mux.HandleFunc("GET /heat/stacks/gone", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{}`)
	})
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	found, err := conn.SearchStacks(ctx, stackcloud.SearchRequest{NameOrID: "a*"})
	if err != nil || len(found.Stacks) != 1 || found.Stacks[0].ID != "1" {
		t.Fatalf("search: %+v %v", found, err)
	}
	deleted, err := conn.DeleteStack(ctx, stackcloud.DeleteRequest{NameOrID: "gone"})
	if err != nil || deleted.Deleted {
		t.Fatalf("delete missing: %+v %v", deleted, err)
	}
	var nilConn *sdk.Connection
	if _, err := nilConn.CreateStack(ctx, stackcloud.CreateRequest{Name: "app"}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil connection: %v", err)
	}
	var noContext context.Context
	if _, err := conn.UpdateStack(noContext, stackcloud.UpdateRequest{NameOrID: "app"}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil context: %v", err)
	}
}
