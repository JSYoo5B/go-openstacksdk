package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharenetworks"
)

const manilaShareNetworkBody = `{"share_network":{"id":"sn-1","project_id":"p1","neutron_net_id":"net-1","neutron_subnet_id":"sub-1","nova_net_id":null,"network_type":"vlan","segmentation_id":100,"cidr":"192.0.2.0/24","ip_version":4,"name":"","description":"","created_at":"2026-01-02T03:04:05.000000","updated_at":null,"vendor_extra":1}}`

func TestManilaShareNetworkCreateSendsEmptyNameAndAccepts200Or202(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, true}, {201, false}, {204, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /v2/share-networks", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			manilaSupportCheckVersion(t, r)
			// name and description have no omitempty; empty neutron IDs are omitted.
			want := map[string]string{"name": `""`, "description": `""`, "neutron_net_id": `"net-1"`, "vendor": `{"on":false}`}
			if got := manilaSupportStrings(manilaSupportEnvelope(t, r, "share_network")); !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, manilaShareNetworkBody)
		})
		value, err := sharenetworks.New(newManilaSupportClient(cloud)).Create(context.Background(),
			sharenetworks.CreateOpts{NeutronNetID: "net-1"},
			sharenetworks.WithCreateField("vendor", map[string]bool{"on": false}))
		if tc.success != (err == nil) || calls.Load() != 1 {
			t.Fatalf("status %d value=%+v error=%v calls=%d", tc.status, value, err, calls.Load())
		}
		if tc.success && (value.ID != "sn-1" || value.SegmentationID != 100 || value.NovaNetID != "" || value.CreatedAt.IsZero() || !value.UpdatedAt.IsZero()) {
			t.Fatalf("status %d value=%+v", tc.status, value)
		}
		if !tc.success {
			manilaSupportUnexpected(t, err, tc.status)
		}
	}
}

func TestManilaShareNetworkRejectsProtectedOrNilOptionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := sharenetworks.New(newManilaSupportClient(cloud))
	ctx := context.Background()
	for name, err := range map[string]error{
		"create name": func() error {
			_, err := api.Create(ctx, sharenetworks.CreateOpts{}, sharenetworks.WithCreateField("name", "shadow"))
			return err
		}(),
		"create nil": func() error { _, err := api.Create(ctx, sharenetworks.CreateOpts{}, nil); return err }(),
		"update protected": func() error {
			empty := ""
			_, err := api.Update(ctx, "sn-1", sharenetworks.UpdateOpts{Name: &empty}, sharenetworks.WithUpdateField("name", "x"))
			return err
		}(),
		"update nil": func() error { _, err := api.Update(ctx, "sn-1", sharenetworks.UpdateOpts{}, nil); return err }(),
		"add protected": func() error {
			_, err := api.AddSecurityService(ctx, "sn-1", sharenetworks.AddSecurityServiceOpts{SecurityServiceID: "a"}, sharenetworks.WithAddSecurityServiceField("security_service_id", "b"))
			return err
		}(),
		"add nil": func() error {
			_, err := api.AddSecurityService(ctx, "sn-1", sharenetworks.AddSecurityServiceOpts{}, nil)
			return err
		}(),
		"remove protected": func() error {
			_, err := api.RemoveSecurityService(ctx, "sn-1", sharenetworks.RemoveSecurityServiceOpts{SecurityServiceID: "a"}, sharenetworks.WithRemoveSecurityServiceField("security_service_id", "b"))
			return err
		}(),
		"remove nil": func() error {
			_, err := api.RemoveSecurityService(ctx, "sn-1", sharenetworks.RemoveSecurityServiceOpts{}, nil)
			return err
		}(),
	} {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for range api.ListDetail(ctx, nil) {
		// The sequence yields one invalid-option error and no HTTP request.
	}
}

func TestManilaShareNetworkUpdateKeepsEmptyStringsAndAcceptsOnly200(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, false}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("PUT /v2/share-networks/sn-1", func(w http.ResponseWriter, r *http.Request) {
			manilaSupportCheckVersion(t, r)
			want := map[string]string{"name": `""`, "description": `""`, "vendor": `null`}
			if got := manilaSupportStrings(manilaSupportEnvelope(t, r, "share_network")); !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, manilaShareNetworkBody)
		})
		empty := ""
		value, err := sharenetworks.New(newManilaSupportClient(cloud)).Update(context.Background(), "sn-1",
			sharenetworks.UpdateOpts{Name: &empty, Description: &empty}, sharenetworks.WithUpdateField("vendor", nil))
		if tc.success != (err == nil) || (tc.success && value.ID != "sn-1") {
			t.Fatalf("status %d value=%+v error=%v", tc.status, value, err)
		}
		if !tc.success {
			manilaSupportUnexpected(t, err, tc.status)
		}
	}
}

func TestManilaShareNetworkUpdateOmitsNilPointers(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("PUT /v2/share-networks/sn-1", func(w http.ResponseWriter, r *http.Request) {
		if got := manilaSupportStrings(manilaSupportEnvelope(t, r, "share_network")); !reflect.DeepEqual(got, map[string]string{"neutron_subnet_id": `"sub-2"`}) {
			t.Errorf("body=%v", got)
		}
		testcloud.JSON(w, 200, manilaShareNetworkBody)
	})
	if _, err := sharenetworks.New(newManilaSupportClient(cloud)).Update(context.Background(), "sn-1", sharenetworks.UpdateOpts{NeutronSubnetID: "sub-2"}); err != nil {
		t.Fatal(err)
	}
}

func TestManilaShareNetworkGetDeleteAndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/share-networks/{id}", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		if r.PathValue("id") == "missing" {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		testcloud.JSON(w, 200, manilaShareNetworkBody)
	})
	deleted := map[string]int{"sn-1": 202, "sn-2": 204, "sn-3": 200, "missing": 404}
	cloud.Mux.HandleFunc("DELETE /v2/share-networks/{id}", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		w.WriteHeader(deleted[r.PathValue("id")])
	})
	api := sharenetworks.New(newManilaSupportClient(cloud))
	ctx := context.Background()
	value, err := api.Get(ctx, "sn-1")
	if err != nil || value.CIDR != "192.0.2.0/24" || value.IPVersion != 4 || value.NetworkType != "vlan" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	_, err = api.Get(ctx, "missing")
	manilaSupportUnexpected(t, err, 404)
	for _, id := range []string{"sn-1", "sn-2"} {
		if err := api.Delete(ctx, id); err != nil {
			t.Fatalf("delete %s: %v", id, err)
		}
	}
	manilaSupportUnexpected(t, api.Delete(ctx, "sn-3"), 200)
	manilaSupportUnexpected(t, api.Delete(ctx, "missing"), 404)
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores a missing share network by default: %v", err)
	}
}

func TestManilaShareNetworkSecurityServiceActionsSendEnvelopeAndRequire200(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, false}, {204, false}} {
		cloud := testcloud.New(t)
		var bodies []map[string]string
		var envelopes []string
		cloud.Mux.HandleFunc("POST /v2/share-networks/sn-1/action", func(w http.ResponseWriter, r *http.Request) {
			manilaSupportCheckVersion(t, r)
			raw := manilaSupportEnvelope(t, r, "")
			for key := range raw {
				envelopes = append(envelopes, key)
			}
			for _, key := range []string{"add_security_service", "remove_security_service"} {
				if _, ok := raw[key]; ok {
					bodies = append(bodies, manilaSupportStrings(manilaSupportEnvelopeFrom(t, raw[key])))
				}
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, manilaShareNetworkBody)
		})
		api := sharenetworks.New(newManilaSupportClient(cloud))
		added, addErr := api.AddSecurityService(context.Background(), "sn-1", sharenetworks.AddSecurityServiceOpts{SecurityServiceID: "ss-1"}, sharenetworks.WithAddSecurityServiceField("vendor", false))
		removed, removeErr := api.RemoveSecurityService(context.Background(), "sn-1", sharenetworks.RemoveSecurityServiceOpts{}, sharenetworks.WithRemoveSecurityServiceField("vendor", 0))
		if tc.success != (addErr == nil) || tc.success != (removeErr == nil) {
			t.Fatalf("status %d add=%v remove=%v", tc.status, addErr, removeErr)
		}
		wantBodies := []map[string]string{{"security_service_id": `"ss-1"`, "vendor": `false`}, {"security_service_id": `""`, "vendor": `0`}}
		if !reflect.DeepEqual(bodies, wantBodies) || !reflect.DeepEqual(envelopes, []string{"add_security_service", "remove_security_service"}) {
			t.Fatalf("status %d bodies=%v envelopes=%v", tc.status, bodies, envelopes)
		}
		if tc.success && (added.ID != "sn-1" || removed.ID != "sn-1") {
			t.Fatalf("added=%+v removed=%+v", added, removed)
		}
		if !tc.success {
			manilaSupportUnexpected(t, addErr, tc.status)
			manilaSupportUnexpected(t, removeErr, tc.status)
		}
	}
}

func TestManilaShareNetworkListDetailFiltersAndOffsetPaging(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v2/share-networks/detail", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		requests = append(requests, r.URL.Query())
		switch r.URL.Query().Get("offset") {
		case "":
			testcloud.JSON(w, 200, `{"share_networks":[{"id":"sn-1","name":"one"}]}`)
		case "1":
			testcloud.JSON(w, 200, `{"share_networks":[{"id":"sn-2","name":"two"}]}`)
		default:
			testcloud.JSON(w, 200, `{"share_networks":[]}`)
		}
	})
	api := sharenetworks.New(newManilaSupportClient(cloud))
	var ids []string
	for value, err := range api.ListDetail(context.Background(),
		sharenetworks.WithListDetailOptions(sharenetworks.ListOpts{AllTenants: true, Name: "n", IPVersion: 4, SegmentationID: 100, Limit: 1}),
		sharenetworks.WithListDetailQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"sn-1", "sn-2"}) || len(requests) != 3 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	first := requests[0]
	if first.Get("all_tenants") != "true" || first.Get("name") != "n" || first.Get("ip_version") != "4" || first.Get("segmentation_id") != "100" || first.Get("limit") != "1" || first.Get("vendor") != "x" {
		t.Fatalf("first query=%v", first)
	}
	if requests[1].Get("offset") != "1" || requests[2].Get("offset") != "2" || requests[2].Get("vendor") != "x" {
		t.Fatalf("offset paging: %v", requests)
	}
}

func TestManilaShareNetworkListDetailOmitsZeroFiltersAndProbesWithoutLimit(t *testing.T) {
	cloud := testcloud.New(t)
	var requests []url.Values
	cloud.Mux.HandleFunc("GET /v2/share-networks/detail", func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Query())
		if r.URL.Query().Has("offset") {
			testcloud.JSON(w, 200, `{"share_networks":[]}`)
			return
		}
		testcloud.JSON(w, 200, `{"share_networks":[{"id":"sn-1","name":"one"}]}`)
	})
	var ids []string
	for value, err := range sharenetworks.New(newManilaSupportClient(cloud)).ListDetail(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"sn-1"}) || len(requests) != 2 || len(requests[0]) != 0 {
		t.Fatalf("ids=%v requests=%v", ids, requests)
	}
	// Without a limit the upstream pager asks for one more page at the maximum offset.
	if got := requests[1].Get("offset"); got != "9223372036854775807" {
		t.Fatalf("probe offset=%q", got)
	}
}

func TestManilaShareNetworkCommonLookupFindsExactName(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/share-networks/web", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/share-networks/detail", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		if r.URL.Query().Has("offset") {
			testcloud.JSON(w, 200, `{"share_networks":[]}`)
			return
		}
		testcloud.JSON(w, 200, `{"share_networks":[{"id":"sn-1","name":"web"},{"id":"sn-2","name":"web-2"}]}`)
	})
	value, err := sharenetworks.New(newManilaSupportClient(cloud)).Find(context.Background(), resource.Name("web"))
	if err != nil || value.ID != "sn-1" || len(queries) == 0 || queries[0].Get("name") != "web" {
		t.Fatalf("value=%+v error=%v queries=%v", value, err, queries)
	}
}

func TestManilaShareNetworkCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	api := sharenetworks.New(newManilaSupportClient(cloud))
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(canceled, "sn-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if err := api.Delete(canceled, "sn-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled delete: %v", err)
	}
	if _, err := api.AddSecurityService(canceled, "sn-1", sharenetworks.AddSecurityServiceOpts{SecurityServiceID: "ss"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled add: %v", err)
	}
	for _, err := range api.ListDetail(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled list: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Create(deadline, sharenetworks.CreateOpts{Name: "n"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
