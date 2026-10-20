package api_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/common/extensions"
	dnsquotas "github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/utils"
	"github.com/gophercloud/gophercloud/v2"
)

func TestCommonOpsExtensionURLsAreServiceRelative(t *testing.T) {
	cloud := testcloud.New(t)
	api := extensions.New(cloud.Client("compute", "/compute/v2"))
	ctx := context.Background()
	if got, want := api.ExtensionURL(ctx, "os-keypairs"), cloud.Server.URL+"/compute/v2/extensions/os-keypairs"; got != want {
		t.Fatalf("ExtensionURL=%q want %q", got, want)
	}
	if got, want := api.ListExtensionURL(ctx), cloud.Server.URL+"/compute/v2/extensions"; got != want {
		t.Fatalf("ListExtensionURL=%q want %q", got, want)
	}
}

func TestCommonOpsExtensionGetDecodesEnvelopeAndKeepsNotFound(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /compute/v2/extensions/{alias}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("alias") == "missing" {
			testcloud.JSON(w, 404, `{"itemNotFound":{"message":"nope"}}`)
			return
		}
		testcloud.JSON(w, 200, `{"extension":{"alias":"os-keypairs","name":"Keypairs","namespace":"ns","description":"d","updated":"2013-01-01T00:00:00Z","links":[]}}`)
	})
	api := extensions.New(cloud.Client("compute", "/compute/v2"))
	got, err := api.Get(context.Background(), "os-keypairs")
	if err != nil || got.Alias != "os-keypairs" || got.Name != "Keypairs" || got.Description != "d" || got.Namespace != "ns" {
		t.Fatalf("extension=%+v err=%v", got, err)
	}
	_, err = api.Get(context.Background(), "missing")
	var notFound gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &notFound) || notFound.Actual != 404 {
		t.Fatalf("error=%v", err)
	}
}

func TestCommonOpsExtensionListStreamsSinglePage(t *testing.T) {
	cloud := testcloud.New(t)
	var requests atomic.Int32
	cloud.Mux.HandleFunc("GET /compute/v2/extensions", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		testcloud.JSON(w, 200, `{"extensions":[{"alias":"a","name":"A"},{"alias":"b","name":"B"}]}`)
	})
	var aliases []string
	for extension, err := range extensions.New(cloud.Client("compute", "/compute/v2")).List(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		aliases = append(aliases, extension.Alias)
	}
	if strings.Join(aliases, ",") != "a,b" || requests.Load() != 1 {
		t.Fatalf("aliases=%v requests=%d", aliases, requests.Load())
	}
}

func TestCommonOpsDNSQuotasURLIsProjectScoped(t *testing.T) {
	cloud := testcloud.New(t)
	api := dnsquotas.New(cloud.Client("dns", "/dns/v2"))
	if got, want := api.URL(context.Background(), "project-1"), cloud.Server.URL+"/dns/v2/quotas/project-1"; got != want {
		t.Fatalf("URL=%q want %q", got, want)
	}
}

func commonOpsIdentityCloud(t *testing.T, discovery string) (*testcloud.Cloud, *atomic.Int32) {
	t.Helper()
	cloud := testcloud.New(t)
	var requests atomic.Int32
	cloud.Provider.IdentityBase = cloud.Server.URL + "/"
	cloud.Provider.IdentityEndpoint = cloud.Server.URL + "/identity/"
	cloud.Mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		testcloud.JSON(w, 300, strings.ReplaceAll(discovery, "BASE", cloud.Server.URL))
	})
	return cloud, &requests
}

func TestCommonOpsChooseVersionPicksHighestPriorityGoodStatus(t *testing.T) {
	cloud, requests := commonOpsIdentityCloud(t, `{"versions":{"values":[
		{"id":"v2.0","status":"stable","links":[{"rel":"self","href":"BASE/v2.0/"}]},
		{"id":"v3.14","status":"current","links":[{"rel":"self","href":"BASE/v3/"}]},
		{"id":"v4.0","status":"experimental","links":[{"rel":"self","href":"BASE/v4/"}]}]}}`)
	recognized := []*utils.Version{{ID: "v2", Suffix: "/v2.0/", Priority: 20}, {ID: "v3", Suffix: "/v3/", Priority: 30}, {ID: "v4", Suffix: "/v4/", Priority: 40}}
	version, endpoint, err := utils.New(cloud.Client("identity", "/identity")).ChooseVersion(context.Background(), recognized)
	if err != nil || version.ID != "v3" || endpoint != cloud.Server.URL+"/v3/" || requests.Load() != 1 {
		t.Fatalf("version=%+v endpoint=%q err=%v requests=%d", version, endpoint, err, requests.Load())
	}
}

func TestCommonOpsChooseVersionUsesEndpointSuffixWithoutRequest(t *testing.T) {
	cloud, requests := commonOpsIdentityCloud(t, `{}`)
	cloud.Provider.IdentityEndpoint = cloud.Server.URL + "/identity/v2.0"
	recognized := []*utils.Version{{ID: "v2", Suffix: "/v2.0/", Priority: 20}, {ID: "v3", Suffix: "/v3/", Priority: 30}}
	version, endpoint, err := utils.New(cloud.Client("identity", "/identity")).ChooseVersion(context.Background(), recognized)
	if err != nil || version.ID != "v2" || endpoint != cloud.Server.URL+"/identity/v2.0/" || requests.Load() != 0 {
		t.Fatalf("version=%+v endpoint=%q err=%v requests=%d", version, endpoint, err, requests.Load())
	}
}

func TestCommonOpsChooseVersionReportsNoSupportedVersion(t *testing.T) {
	cloud, _ := commonOpsIdentityCloud(t, `{"versions":{"values":[{"id":"v3.0","status":"experimental","links":[{"rel":"self","href":"BASE/v3/"}]}]}}`)
	version, endpoint, err := utils.New(cloud.Client("identity", "/identity")).ChooseVersion(context.Background(), []*utils.Version{{ID: "v3", Suffix: "/v3/", Priority: 30}})
	if err == nil || !strings.Contains(err.Error(), "no supported version") || version != nil || endpoint != "" {
		t.Fatalf("version=%+v endpoint=%q err=%v", version, endpoint, err)
	}
}

func TestCommonOpsGetServiceVersionsInfersFromURLOrDiscovers(t *testing.T) {
	cloud := testcloud.New(t)
	var requests atomic.Int32
	cloud.Mux.HandleFunc("GET /compute/", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		testcloud.JSON(w, 200, `{"versions":[
			{"id":"v2.0","status":"DEPRECATED","version":"","min_version":""},
			{"id":"v2.1","status":"CURRENT","version":"2.90","min_version":"2.1"}]}`)
	})
	api := utils.New(cloud.Client("compute", "/compute/"))
	ctx := context.Background()
	inferred, err := api.GetServiceVersions(ctx, cloud.Server.URL+"/compute/v2.1/", false)
	if err != nil || len(inferred) != 1 || inferred[0].Major != 2 || inferred[0].Minor != 1 || requests.Load() != 0 {
		t.Fatalf("inferred=%+v err=%v requests=%d", inferred, err, requests.Load())
	}
	found, err := api.GetServiceVersions(ctx, cloud.Server.URL+"/compute/", true)
	if err != nil || len(found) != 2 || requests.Load() != 1 {
		t.Fatalf("found=%+v err=%v requests=%d", found, err, requests.Load())
	}
	if found[0].Minor != 1 || found[0].Status != utils.StatusCurrent || found[0].MaxMajor != 2 || found[0].MaxMinor != 90 || found[0].MinMajor != 2 || found[0].MinMinor != 1 {
		t.Fatalf("highest version=%+v", found[0])
	}
	if found[1].Minor != 0 || found[1].Status != utils.StatusDeprecated || found[1].MaxMajor != 0 {
		t.Fatalf("lower version=%+v", found[1])
	}
}

func TestCommonOpsGetSupportedMicroversionsRequiresSingleVersionedEndpoint(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /compute/v2.1/", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"version":{"id":"v2.1","status":"CURRENT","version":"2.90","min_version":"2.1"}}`)
	})
	cloud.Mux.HandleFunc("GET /plain/v1/", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"version":{"id":"v1","status":"CURRENT"}}`)
	})
	cloud.Mux.HandleFunc("GET /multi/", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"versions":[{"id":"v1","status":"CURRENT","version":"1.5","min_version":"1.0"},{"id":"v2","status":"CURRENT","version":"2.5","min_version":"2.0"}]}`)
	})
	ctx := context.Background()
	got, err := utils.New(cloud.Client("compute", "/compute/v2.1")).GetSupportedMicroversions(ctx)
	if err != nil || got.MinMajor != 2 || got.MinMinor != 1 || got.MaxMajor != 2 || got.MaxMinor != 90 {
		t.Fatalf("microversions=%+v err=%v", got, err)
	}
	if _, err := utils.New(cloud.Client("plain", "/plain/v1")).GetSupportedMicroversions(ctx); err == nil || !strings.Contains(err.Error(), "microversions not supported") {
		t.Fatalf("no microversions error=%v", err)
	}
	if _, err := utils.New(cloud.Client("multi", "/multi")).GetSupportedMicroversions(ctx); err == nil || !strings.Contains(err.Error(), "multiple alternatives") {
		t.Fatalf("unversioned error=%v", err)
	}
}
