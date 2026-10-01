package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/clustering/v1/policies"
	"gophercloudsdk/clustering/v1/profiles"
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

func TestConnectionSenlinProfilesAndPoliciesShareSelectedVersionAndFreshToken(t *testing.T) {
	cloud := testcloud.New(t)
	check := func(r *http.Request) {
		if r.Header.Get("X-Auth-Token") != "fresh-token" || r.Header.Get("OpenStack-API-Version") != "clustering 1.2" {
			t.Errorf("request lost shared auth/version: %#v", r.Header)
		}
	}
	cloud.Mux.HandleFunc("POST /reverse/senlin/v1/profiles", func(w http.ResponseWriter, r *http.Request) {
		check(r)
		var body map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || string(body["profile"]["name"]) != `"template"` || string(body["profile"]["metadata"]) != `{}` {
			t.Errorf("profile body=%v err=%v", body, err)
		}
		testcloud.JSON(w, 201, `{"profile":{"id":"created-profile","name":"template","spec":{"type":"os.nova.server","version":"1.0","properties":{}},"metadata":{}}}`)
	})
	cloud.Mux.HandleFunc("POST /reverse/senlin/v1/policies/validate", func(w http.ResponseWriter, r *http.Request) {
		check(r)
		var body map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body["policy"]) != 1 || body["policy"]["spec"] == nil {
			t.Errorf("validation body=%v err=%v", body, err)
		}
		testcloud.JSON(w, 200, `{"policy":{"id":null,"name":null,"type":"senlin.policy.scaling","spec":{"type":"senlin.policy.scaling","version":"1.0","properties":{}},"created_at":null}}`)
	})
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.Clustering, cloud.Server.URL+"/reverse/senlin"), sdk.WithMicroversion(sdk.Clustering, "1.2"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := conn.Clustering(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if service.Profiles.RawClient() != service.RawClient() || service.Policies.RawClient() != service.RawClient() {
		t.Fatal("Senlin resource APIs do not share the configured client")
	}
	cloud.Provider.SetToken("fresh-token")
	profile, err := service.Profiles.Create(context.Background(), profiles.CreateOpts{Name: "template", Spec: json.RawMessage(`{"type":"os.nova.server","version":"1.0","properties":{}}`), Metadata: json.RawMessage(`{}`)})
	if err != nil || profile == nil || profile.StatusCode != 201 || string(profile.Body["metadata"]) != `{}` {
		t.Fatalf("profile=%+v err=%v", profile, err)
	}
	policy, err := service.Policies.Validate(context.Background(), policies.ValidateOpts{Spec: json.RawMessage(`{"type":"senlin.policy.scaling","version":"1.0","properties":{}}`)})
	if err != nil || policy == nil || policy.StatusCode != 200 || string(policy.Body["id"]) != "null" {
		t.Fatalf("policy=%+v err=%v", policy, err)
	}
}
