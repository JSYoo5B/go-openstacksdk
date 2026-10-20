package resource_test

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestWaitKeepsExplicitIDWhenResponseIdentityChanges(t *testing.T) {
	for _, responseID := range []string{"", "different-server"} {
		t.Run(fmt.Sprintf("initial response ID %q", responseID), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /compute/servers/requested", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					testcloud.JSON(w, 200, fmt.Sprintf(`{"server":{"id":%q,"status":"BUILD"}}`, responseID))
				} else {
					testcloud.JSON(w, 200, `{"server":{"id":"requested","status":"ACTIVE"}}`)
				}
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("response redirected polling to %s", r.URL)
				http.Error(w, "unexpected target", 500)
			})
			service := compute.New(cloud.Client("compute", "/compute"), compute.Dependencies{})
			ready, err := service.Servers.Wait(context.Background(), resource.ID("requested"), "ACTIVE", resource.WithTimeout(time.Second), resource.WithPollInterval(time.Millisecond))
			if err != nil || ready == nil || ready.ID != "requested" || calls.Load() != 2 {
				t.Fatalf("ready=%v calls=%d err=%v", ready, calls.Load(), err)
			}
		})
	}
}
