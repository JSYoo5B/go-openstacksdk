package gophercloudsdk_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/blockstorage/metadata"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

type connectionMetadataScope interface {
	ID() string
	RawClient() *gophercloud.ServiceClient
	Get(context.Context, ...metadata.Option) (*metadata.Result, error)
}

func connectionCinderMetadataFactories() map[string]func(*sdk.Connection, context.Context, resource.Ref) (connectionMetadataScope, error) {
	return map[string]func(*sdk.Connection, context.Context, resource.Ref) (connectionMetadataScope, error){
		"volumes": func(c *sdk.Connection, ctx context.Context, ref resource.Ref) (connectionMetadataScope, error) {
			return c.VolumeMetadata(ctx, ref)
		},
		"snapshots": func(c *sdk.Connection, ctx context.Context, ref resource.Ref) (connectionMetadataScope, error) {
			return c.SnapshotMetadata(ctx, ref)
		},
	}
}

func TestConnectionCinderMetadataSharesLiveVersionedClientAndFixedPrefix(t *testing.T) {
	for collection, bind := range connectionCinderMetadataFactories() {
		t.Run(collection, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/proxy/cinder/v3/project/"+collection+"/parent/metadata", func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				if r.Method != http.MethodGet || r.URL.RawQuery != "" || r.Header.Get("X-Source") != "source" || r.Header.Get("X-Call") != "metadata" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.70" || r.Header.Get("OpenStack-API-Version") != "volume 3.70" || r.Header.Get("X-Auth-Token") != fmt.Sprintf("token-%d", n) {
					t.Errorf("shared request changed: %s %s headers=%v", r.Method, r.URL, r.Header)
				}
				w.Header().Set("ETag", `"revision"`)
				testcloud.JSON(w, 200, `{"metadata":{"owner":"wire"},"wire_number":9007199254740993}`)
			})
			conn, err := sdk.FromProvider(cloud.Provider,
				sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/catalog/v3/project"),
				sdk.WithMicroversion(sdk.BlockStorage, "3.70"))
			if err != nil {
				t.Fatal(err)
			}
			service, err := conn.BlockStorageV3(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			client := service.RawClient()
			client.ResourceBase = cloud.Server.URL + "/proxy/cinder/v3/project/"
			client.MoreHeaders = map[string]string{"X-Source": "source"}
			scope, err := bind(conn, context.Background(), resource.ID("parent"))
			if err != nil || scope.ID() != "parent" || scope.RawClient() != client || calls.Load() != 0 {
				t.Fatalf("ID scope performed discovery or changed client: %v", err)
			}
			// A fixed scope keeps its original URL while authentication remains live.
			client.ResourceBase = cloud.Server.URL + "/later/"
			option := metadata.WithHeader("X-Call", "metadata")
			for n := 1; n <= 2; n++ {
				cloud.Provider.SetToken(fmt.Sprintf("token-%d", n))
				result, err := scope.Get(context.Background(), option)
				if err != nil || result.Metadata["owner"] != "wire" || result.StatusCode != 200 || result.Header.Get("ETag") != `"revision"` || !strings.Contains(string(result.Body), "9007199254740993") {
					t.Fatalf("actual metadata evidence lost: %+v %v", result, err)
				}
			}
			if calls.Load() != 2 || client.ProviderClient != cloud.Provider || client.Microversion != "3.70" || len(client.MoreHeaders) != 1 || client.MoreHeaders["X-Source"] != "source" {
				t.Fatal("scope modified shared configuration or made extra requests")
			}
		})
	}
}

func TestConnectionCinderMetadataResolvesNamesOnceAndPreflightsReferences(t *testing.T) {
	for collection, bind := range connectionCinderMetadataFactories() {
		t.Run(collection, func(t *testing.T) {
			cloud := testcloud.New(t)
			var lookups, gets atomic.Int32
			cloud.Mux.HandleFunc("/cinder/v3/project/"+collection+"/detail", func(w http.ResponseWriter, r *http.Request) {
				lookups.Add(1)
				if r.Method != http.MethodGet || r.URL.Query().Get("name") != "target" {
					t.Errorf("name lookup: %s %s", r.Method, r.URL)
				}
				testcloud.JSON(w, 200, fmt.Sprintf(`{"%s":[{"id":"resolved","name":"target"}]}`, collection))
			})
			cloud.Mux.HandleFunc("/cinder/v3/project/"+collection+"/resolved/metadata", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 200, `{"metadata":{}}`)
			})
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/cinder/v3/project"))
			if err != nil {
				t.Fatal(err)
			}
			canceled, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := bind(conn, canceled, resource.ID("parent")); !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled construction: %v", err)
			}
			for _, ref := range []resource.Ref{{}, resource.ID("../other"), resource.Name(" ")} {
				if _, err := bind(conn, context.Background(), ref); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatalf("invalid reference: %v", err)
				}
			}
			if _, err := bind(nil, context.Background(), resource.ID("parent")); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("nil Connection: %v", err)
			}
			scope, err := bind(conn, context.Background(), resource.Name("target"))
			if err != nil || scope.ID() != "resolved" || lookups.Load() != 1 || gets.Load() != 0 {
				t.Fatalf("single name resolution: %v lookups=%d gets=%d", err, lookups.Load(), gets.Load())
			}
			for range 2 {
				if _, err := scope.Get(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			if lookups.Load() != 1 || gets.Load() != 2 {
				t.Fatal("scope re-resolved its parent")
			}
		})
	}
}
