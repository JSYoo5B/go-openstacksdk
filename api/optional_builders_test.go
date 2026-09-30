package api_test

import (
	"context"
	"net/http"
	"testing"

	"gophercloudsdk/dns/v2/recordsets"
	"gophercloudsdk/dns/v2/zones"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/objectstorage/v1/objects"
)

func TestOptionalDNSHeadersSurviveLibraryOwnedBuilders(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/v2/zones", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-All-Projects") != "true" || r.Header.Get("X-Auth-Sudo-Tenant-ID") != "project-id" || r.Header.Get("X-Vendor-Setting") != "false" {
			t.Errorf("zone headers=%v", r.Header)
		}
		testcloud.JSON(w, 200, "{\"zones\":[]}")
	})
	cloud.Mux.HandleFunc("/v2/zones/zone-id/recordsets", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Auth-All-Projects") != "true" || r.Header.Get("X-Auth-Sudo-Tenant-ID") != "project-id" || r.Header.Get("X-Vendor-Setting") != "false" {
			t.Errorf("record set headers=%v", r.Header)
		}
		if r.Method == http.MethodGet {
			testcloud.JSON(w, 200, "{\"recordsets\":[]}")
			return
		}
		testcloud.JSON(w, 201, "{\"id\":\"record-id\"}")
	})
	ctx := context.Background()
	for _, err := range zones.New(cloud.Client("dns", "/v2")).List(ctx, zones.WithListOptions(zones.ListOpts{AllProjects: true, SudoTenantID: "project-id"}), zones.WithListHeader("X-Vendor-Setting", "false")) {
		if err != nil {
			t.Fatal(err)
		}
	}
	api := recordsets.New(cloud.Client("dns", "/v2"))
	if _, err := api.Create(ctx, "zone-id", recordsets.CreateOpts{Name: "www.example.org.", Type: "A", Records: []string{"192.0.2.1"}, AllProjects: true, SudoTenantID: "project-id"}, recordsets.WithCreateHeader("X-Vendor-Setting", "false")); err != nil {
		t.Fatal(err)
	}
	for _, err := range api.ListByZone(ctx, "zone-id", recordsets.WithListByZoneOptions(recordsets.ListOpts{AllProjects: true, SudoTenantID: "project-id"}), recordsets.WithListByZoneHeader("X-Vendor-Setting", "false")) {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestOptionalSwiftCopyQuerySurvivesLibraryOwnedBuilder(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/v1/AUTH_project/source/object", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "COPY" || r.URL.Query().Get("version-id") != "version" || r.URL.Query().Get("vendor") != "a&b" || r.Header.Get("Destination") != "/target/object" {
			t.Errorf("method=%s query=%v headers=%v", r.Method, r.URL.Query(), r.Header)
		}
		w.WriteHeader(201)
	})
	_, err := objects.New(cloud.Client("object-store", "/v1/AUTH_project")).Copy(context.Background(), "source", "object", objects.CopyOpts{Destination: "/target/object", ObjectVersionID: "version"}, objects.WithCopyQuery("vendor", "a&b"))
	if err != nil {
		t.Fatal(err)
	}
}
