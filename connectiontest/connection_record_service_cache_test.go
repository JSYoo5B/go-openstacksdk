package connectiontest_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionRouterCreateOwnsDefaultLocationAndCachedService(t *testing.T) {
	cloud := testcloud.New(t)
	locates, calls := 0, 0
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates++
		return cloud.Server.URL + "/network/", nil
	}
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("POST /network/v2.0/routers", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("X-Auth-Token") != "test-token" {
			t.Fatal(r.Header)
		}
		testcloud.JSON(w, 201, `{"router":{"id":"new","name":"edge","project_id":"project"}}`)
	})
	callbacks := 0
	options := []network.RouterCreateOption{network.WithRouterCreateName("edge"), func(cfg *network.RouterCreateOpts) error { callbacks++; return nil }}
	value, err := conn.CreateRouterRecord(context.Background(), options...)
	if err != nil || value == nil || value.Router == nil || value.StatusCode != 201 || value.Wire == nil || calls != 1 || locates != 1 || callbacks != 1 {
		t.Fatal(value, err, calls, locates, callbacks)
	}
	if string(value.Router.Body["location"]) == "null" || len(value.Router.Body["location"]) == 0 || !strings.Contains(string(value.Router.Body["location"]), `"project"`) {
		t.Fatal(value.Router.Body)
	}
	_, err = conn.CreateRouterRecord(context.Background(), options...)
	if err != nil || locates != 1 || calls != 2 || callbacks != 2 {
		t.Fatal(err, locates, calls, callbacks)
	}
}

func TestConnectionObjectGetTextSharesCachedSwiftAndRetainsReceipt(t *testing.T) {
	cloud := testcloud.New(t)
	locates, calls := 0, 0
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates++
		return cloud.Server.URL + "/swift/v1/AUTH_p/", nil
	}
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("GET /swift/v1/AUTH_p/bucket/object", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Fatal(r.URL, r.Header)
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Request-Id", "get-text")
		w.WriteHeader(200)
		_, _ = w.Write([]byte("안녕"))
	})
	for range 2 {
		value, err := conn.GetObjectText(context.Background(), "bucket", "object", objectstorage.WithObjectGetQueryString("ignored"))
		if err != nil || value == nil || value.Body == nil || *value.Body != "안녕" || value.Receipt == nil || value.Receipt.StatusCode != 200 || value.Headers["x-request-id"] != "get-text" {
			t.Fatal(value, err)
		}
	}
	if locates != 1 || calls != 2 {
		t.Fatal(locates, calls)
	}
}

func TestConnectionCycle7PreflightPreservesCancellationBeforeDiscovery(t *testing.T) {
	for _, mode := range []string{"nil context", "nil connection", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			locates := 0
			cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
				locates++
				return "", gophercloud.ErrEndpointNotFound{}
			}
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			want := error(resource.ErrInvalidOption)
			switch mode {
			case "nil context":
				ctx = nil
			case "nil connection":
				conn = nil
			case "cancelled":
				cause := errors.New("custom cycle cancellation")
				cancelCtx, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx = cancelCtx
				want = cause
			}
			router, err := conn.CreateRouterRecord(ctx)
			if router != nil || !errors.Is(err, want) {
				t.Fatal(router, err)
			}
			subnet, err := conn.CreateSubnetRecord(ctx, "network")
			if subnet != nil || !errors.Is(err, want) {
				t.Fatal(subnet, err)
			}
			text, err := conn.GetObjectText(ctx, "bucket", "object")
			if text != nil || !errors.Is(err, want) || locates != 0 {
				t.Fatal(text, err, locates)
			}
		})
	}
}

func TestConnectionSubnetCreateSharesNetworkCacheAndLocation(t *testing.T) {
	cloud := testcloud.New(t)
	locates, reads, posts := 0, 0, 0
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates++
		return cloud.Server.URL + "/network/", nil
	}
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("GET /network/v2.0/networks/net", func(w http.ResponseWriter, r *http.Request) {
		reads++
		testcloud.JSON(w, 200, `{"network":{"id":"net","name":"net"}}`)
	})
	cloud.Mux.HandleFunc("POST /network/v2.0/subnets", func(w http.ResponseWriter, r *http.Request) {
		posts++
		testcloud.JSON(w, 201, `{"subnet":{"id":"new","network_id":"net","cidr":"10.0.0.0/24","project_id":"project"}}`)
	})
	for range 2 {
		value, err := conn.CreateSubnetRecord(context.Background(), "net", network.WithSubnetCreateCIDR("10.0.0.0/24"))
		if err != nil || value == nil || value.Resource == nil || value.Network == nil || value.StatusCode != 201 || len(value.Resource.Body["location"]) == 0 || string(value.Resource.Body["location"]) == "null" {
			t.Fatal(value, err)
		}
	}
	if locates != 1 || reads != 2 || posts != 2 {
		t.Fatal(locates, reads, posts)
	}
}
