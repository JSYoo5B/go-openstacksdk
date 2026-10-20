package openstack_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func defaultNetworkCloudOptions(t *testing.T, cloud *testcloud.Cloud, settings string) []sdk.ConnectionOption {
	t.Helper()
	for _, key := range []string{"OS_CLOUD", "OS_CLIENT_CONFIG_FILE", "OS_REGION_NAME", "OS_INTERFACE", "OS_AUTH_URL", "OS_USERNAME", "OS_PASSWORD", "OS_USERID", "OS_PROJECT_ID", "OS_PROJECT_NAME", "OS_TENANT_ID", "OS_TENANT_NAME", "OS_DOMAIN_ID", "OS_DOMAIN_NAME", "OS_APPLICATION_CREDENTIAL_ID", "OS_APPLICATION_CREDENTIAL_NAME", "OS_APPLICATION_CREDENTIAL_SECRET", "OS_SYSTEM_SCOPE", "OS_PASSCODE"} {
		t.Setenv(key, "")
	}
	path := filepath.Join(t.TempDir(), "clouds.yaml")
	body := fmt.Sprintf("clouds:\n  dev:\n    auth:\n      auth_url: %s/v3\n      username: cloud-user\n      password: cloud-password\n      user_domain_name: Default\n%s", cloud.Server.URL, settings)
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("POST /v3/auth/tokens", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Subject-Token", "authenticated-token")
		testcloud.JSON(w, 201, `{"token":{"expires_at":"2099-01-01T00:00:00Z","catalog":[]}}`)
	})
	return []sdk.ConnectionOption{sdk.WithCloud("dev"), sdk.WithCloudFiles(path),
		sdk.WithEndpoint(sdk.Compute, cloud.Server.URL+"/compute"), sdk.WithEndpoint(sdk.Network, cloud.Server.URL+"/network")}
}

func TestConnectConfiguredDefaultNetworkMatchesNamesOrIDsAcrossAllPages(t *testing.T) {
	for _, scenario := range []string{"name", "ID", "cross collision", "duplicate name", "missing", "forbidden last page", "unsafe ID", "canceled"} {
		t.Run(scenario, func(t *testing.T) {
			cloud := testcloud.New(t)
			opts := defaultNetworkCloudOptions(t, cloud, "    networks: [{name: selected, default_interface: true}]\n")
			var lists, creates atomic.Int32
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cloud.Mux.HandleFunc("GET /network/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("name") != "" || r.URL.Query().Get("id") != "" || r.Header.Get("X-Auth-Token") != "authenticated-token" {
					t.Errorf("unfiltered shared-token query=%q headers=%v", r.URL.RawQuery, r.Header)
				}
				rows := []any{map[string]any{"id": "other", "name": "selected-copy"}}
				links := []any{}
				if r.URL.Query().Get("marker") == "" {
					if scenario == "cross collision" {
						rows = []any{map[string]any{"id": "selected", "name": "other"}}
					} else if scenario == "duplicate name" || scenario == "forbidden last page" {
						rows = []any{map[string]any{"id": "first", "name": "selected"}}
					}
					links = []any{map[string]any{"rel": "next", "href": cloud.Server.URL + "/network/v2.0/networks?marker=next"}}
				} else {
					if scenario == "forbidden last page" {
						testcloud.JSON(w, 403, `{"error":{"message":"next page denied"}}`)
						return
					}
					if scenario == "canceled" {
						cancel()
					}
					if scenario != "missing" {
						id, name := "chosen", "selected"
						if scenario == "ID" {
							id, name = "selected", "different-name"
						} else if scenario == "unsafe ID" {
							id = "bad/id"
						}
						rows = []any{map[string]any{"id": id, "name": name}}
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"networks": rows, "networks_links": links})
			})
			cloud.Mux.HandleFunc("GET /network/v2.0/subnets", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"subnets":[]}`)
			})
			cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
				creates.Add(1)
				var body struct{ Server map[string]any }
				_ = json.NewDecoder(r.Body).Decode(&body)
				wantID := "chosen"
				if scenario == "ID" {
					wantID = "selected"
				}
				if !reflect.DeepEqual(body.Server["networks"], []any{map[string]any{"uuid": wantID}}) || body.Server["imageRef"] != "image-id" || body.Server["flavorRef"] != "flavor-id" || r.Header.Get("X-Auth-Token") != "authenticated-token" {
					t.Errorf("body=%#v headers=%v", body.Server, r.Header)
				}
				testcloud.JSON(w, 202, `{"server":{"id":"created","status":"BUILD"}}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				http.Error(w, "unexpected", 500)
			})
			conn, err := sdk.Connect(ctx, opts...)
			if err != nil {
				t.Fatal(err)
			}
			service, err := conn.Compute(ctx)
			if err != nil || lists.Load() != 0 {
				t.Fatalf("construction err=%v lists=%d", err, lists.Load())
			}
			row, err := service.Servers.Create(ctx, nicConnectionRequest())
			switch scenario {
			case "name", "ID":
				if err != nil || row == nil || row.ID != "created" || creates.Load() != 1 {
					t.Fatalf("row=%v err=%v creates=%d", row, err, creates.Load())
				}
			case "cross collision", "duplicate name":
				var ambiguous *resource.AmbiguousError
				if !errors.As(err, &ambiguous) || len(ambiguous.IDs) != 2 || !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(err)
				}
			case "missing":
				if !errors.Is(err, resource.ErrNotFound) {
					t.Fatal(err)
				}
			case "unsafe ID":
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			case "canceled":
				if !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
			default:
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) || response.Actual != 403 {
					t.Fatal(err)
				}
			}
			if lists.Load() != 2 {
				t.Fatalf("lists=%d", lists.Load())
			}
			if scenario != "name" && scenario != "ID" && (row != nil || creates.Load() != 0) {
				t.Fatalf("row=%v creates=%d", row, creates.Load())
			}
		})
	}
}

func TestConnectDefaultNetworkOverridesDisableAndExplicitCreateSelection(t *testing.T) {
	for _, tc := range []struct {
		name       string
		connection []sdk.ConnectionOption
		create     []compute.CreateServerOption
		want       any
	}{
		{"override ID", []sdk.ConnectionOption{sdk.WithDefaultNetwork(resource.ID("explicit-default"))}, nil, []any{map[string]any{"uuid": "explicit-default"}}},
		{"disable", []sdk.ConnectionOption{sdk.WithoutDefaultNetwork()}, nil, "auto"},
		{"explicit empty NIC", nil, []compute.CreateServerOption{compute.WithNetworkInterfaces(compute.ServerNetworkInterface{})}, []any{map[string]any{}}},
		{"explicit mode", nil, []compute.CreateServerOption{compute.WithNetworkMode("none")}, "none"},
		{"explicit network overrides both defaults", []sdk.ConnectionOption{sdk.WithDefaultNetwork(resource.Name("another-missing"))}, []compute.CreateServerOption{compute.WithNetworks(resource.ID("create-network"))}, []any{map[string]any{"uuid": "create-network"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			opts := defaultNetworkCloudOptions(t, cloud, "    networks: [{name: missing, default_interface: true}]\n")
			opts = append(opts, sdk.WithMicroversion(sdk.Compute, "2.37"))
			opts = append(opts, tc.connection...)
			var lists atomic.Int32
			cloud.Mux.HandleFunc("GET /network/v2.0/networks", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				http.Error(w, "must bypass YAML default", 403)
			})
			cloud.Mux.HandleFunc("POST /compute/servers", func(w http.ResponseWriter, r *http.Request) {
				var body struct{ Server map[string]any }
				_ = json.NewDecoder(r.Body).Decode(&body)
				if !reflect.DeepEqual(body.Server["networks"], tc.want) {
					t.Errorf("body=%#v want=%#v", body.Server, tc.want)
				}
				testcloud.JSON(w, 202, `{"server":{"id":"created"}}`)
			})
			conn, err := sdk.Connect(context.Background(), opts...)
			if err != nil {
				t.Fatal(err)
			}
			service, err := conn.Compute(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			row, err := service.Servers.Create(context.Background(), nicConnectionRequest(), tc.create...)
			if err != nil || row == nil || row.ID != "created" || lists.Load() != 0 {
				t.Fatalf("row=%v err=%v lists=%d", row, err, lists.Load())
			}
		})
	}
}
