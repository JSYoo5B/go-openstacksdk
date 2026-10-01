package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/instanceha/v1/segments"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestConnectionSDKOwnedResourcesUseCachedProviderAndExactServiceRoots(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) { t.Fatal("override consulted catalog"); return "", nil }
	const segmentID = "11111111-1111-4111-8111-111111111111"
	cloud.Mux.HandleFunc("/proxy/senlin/v1/build-info", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.Header.Get("OpenStack-API-Version") != "clustering 1.4" {
			t.Errorf("Senlin request: %s %#v", r.Method, r.Header)
		}
		testcloud.JSON(w, 200, `{"build_info":{"api":{"revision":"api-rev"},"engine":{"revision":"engine-rev"}}}`)
	})
	cloud.Mux.HandleFunc("/proxy/ha/v1/project/segments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Header.Get("OpenStack-API-Version") != "instance-ha 1.2" {
			t.Errorf("Masakari request: %s %#v", r.Method, r.Header)
		}
		var body map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || string(body["segment"]["enabled"]) != "false" {
			t.Errorf("body=%v err=%v", body, err)
		}
		testcloud.JSON(w, 202, `{"segment":{"uuid":"`+segmentID+`","id":9007199254740993,"name":"primary","enabled":false}}`)
	})
	cloud.Mux.HandleFunc("/proxy/ha/v1/project/segments/"+segmentID+"/hosts", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("OpenStack-API-Version") != "instance-ha 1.2" {
			t.Errorf("scoped headers: %#v", r.Header)
		}
		testcloud.JSON(w, 200, `{"hosts":[{"uuid":"22222222-2222-4222-8222-222222222222","name":"compute-1"}]}`)
	})
	conn, err := sdk.FromProvider(cloud.Provider,
		sdk.WithEndpoint(sdk.Clustering, cloud.Server.URL+"/proxy/senlin"), sdk.WithMicroversion(sdk.Clustering, "1.4"),
		sdk.WithEndpoint(sdk.InstanceHA, cloud.Server.URL+"/proxy/ha/v1/project"), sdk.WithMicroversion(sdk.InstanceHA, "1.2"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	clustering, err := conn.Clustering(ctx)
	if err != nil {
		t.Fatal(err)
	}
	again, err := conn.ClusteringV1(ctx)
	if err != nil || again != clustering || clustering.RawClient().ProviderClient != cloud.Provider {
		t.Fatalf("Senlin cache/provider: %v", err)
	}
	info, err := clustering.BuildInfo.Get(ctx)
	if err != nil || info.API == nil || info.API.Revision == nil || *info.API.Revision != "api-rev" {
		t.Fatalf("buildinfo: %#v %v", info, err)
	}
	ha, err := conn.InstanceHA(ctx)
	if err != nil {
		t.Fatal(err)
	}
	haAgain, err := conn.InstanceHAV1(ctx)
	if err != nil || haAgain != ha || ha.RawClient().ProviderClient != cloud.Provider || ha.RawClient() == clustering.RawClient() {
		t.Fatalf("Masakari cache/provider: %v", err)
	}
	disabled := false
	segment, err := ha.Segments.Create(ctx, segments.CreateOpts{Name: "primary", RecoveryMethod: "auto", ServiceType: "COMPUTE", Enabled: &disabled})
	if err != nil || segment.UUID != segmentID || string(segment.ID) != "9007199254740993" {
		t.Fatalf("segment: %#v %v", segment, err)
	}
	scope, err := ha.Hosts.InSegment(ctx, resource.ID(segment.UUID))
	if err != nil {
		t.Fatal(err)
	}
	hosts, err := scope.All(ctx)
	if err != nil || len(hosts) != 1 || hosts[0].SegmentID != segmentID {
		t.Fatalf("hosts: %#v %v", hosts, err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := conn.InstanceHA(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("cached canceled getter: %v", err)
	}
}
