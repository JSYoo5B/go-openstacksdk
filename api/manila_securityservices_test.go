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
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/securityservices"
)

const manilaSecurityServiceBody = `{"security_service":{"id":"ss-1","project_id":"p1","domain":"example.test","status":"new","type":"ldap","name":"dir","description":"","dns_ip":"192.0.2.53","ou":null,"user":"svc","password":"synthetic-secret","server":"ldap.example.test","created_at":"2026-01-02T03:04:05.000000","updated_at":null,"vendor_extra":1}}`

func TestManilaSecurityServiceCreateSendsTypeOmitsEmptyAndAcceptsOnly200(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, false}, {201, false}, {204, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /v2/security-services", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			manilaSupportCheckVersion(t, r)
			// Empty optional strings are omitted by upstream; the extension field is kept.
			want := map[string]string{"type": `"ldap"`, "name": `"dir"`, "dns_ip": `"192.0.2.53"`, "vendor": `false`}
			if got := manilaSupportStrings(manilaSupportEnvelope(t, r, "security_service")); !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, manilaSecurityServiceBody)
		})
		value, err := securityservices.New(newManilaSupportClient(cloud)).Create(context.Background(),
			securityservices.CreateOpts{Type: securityservices.LDAP, Name: "dir", Description: "", DNSIP: "192.0.2.53"},
			securityservices.WithCreateField("vendor", false))
		if tc.success != (err == nil) || calls.Load() != 1 {
			t.Fatalf("status %d value=%+v error=%v calls=%d", tc.status, value, err, calls.Load())
		}
		if tc.success && (value.ID != "ss-1" || value.Type != "ldap" || value.OU != "" || value.Password != "synthetic-secret" || value.CreatedAt.IsZero() || !value.UpdatedAt.IsZero()) {
			t.Fatalf("status %d value=%+v", tc.status, value)
		}
		if !tc.success {
			manilaSupportUnexpected(t, err, tc.status)
		}
	}
}

func TestManilaSecurityServiceRejectsMissingTypeProtectedOrNilOptionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := securityservices.New(newManilaSupportClient(cloud))
	ctx := context.Background()
	if _, err := api.Create(ctx, securityservices.CreateOpts{Name: "dir"}); err == nil {
		t.Error("missing required type must fail before HTTP")
	}
	for name, err := range map[string]error{
		"create type": func() error {
			_, err := api.Create(ctx, securityservices.CreateOpts{Type: securityservices.Kerberos}, securityservices.WithCreateField("type", "ldap"))
			return err
		}(),
		"create nil": func() error {
			_, err := api.Create(ctx, securityservices.CreateOpts{Type: securityservices.Kerberos}, nil)
			return err
		}(),
		"update name": func() error {
			_, err := api.Update(ctx, "ss-1", securityservices.UpdateOpts{}, securityservices.WithUpdateField("name", "x"))
			return err
		}(),
		"update nil": func() error { _, err := api.Update(ctx, "ss-1", securityservices.UpdateOpts{}, nil); return err }(),
	} {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for range api.List(ctx, nil) {
		// The sequence yields one invalid-option error and no HTTP request.
	}
}

func TestManilaSecurityServiceUpdateSendsNullNameAndEmptyStringsAndAcceptsOnly200(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, false}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("PUT /v2/security-services/ss-1", func(w http.ResponseWriter, r *http.Request) {
			manilaSupportCheckVersion(t, r)
			// Name has no omitempty, so a nil pointer is sent as JSON null.
			want := map[string]string{"name": `null`, "description": `""`, "ou": `""`, "password": `""`, "vendor": `[]`}
			if got := manilaSupportStrings(manilaSupportEnvelope(t, r, "security_service")); !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, manilaSecurityServiceBody)
		})
		empty := ""
		value, err := securityservices.New(newManilaSupportClient(cloud)).Update(context.Background(), "ss-1",
			securityservices.UpdateOpts{Description: &empty, OU: &empty, Password: &empty},
			securityservices.WithUpdateField("vendor", []string{}))
		if tc.success != (err == nil) || (tc.success && value.ID != "ss-1") {
			t.Fatalf("status %d value=%+v error=%v", tc.status, value, err)
		}
		if !tc.success {
			manilaSupportUnexpected(t, err, tc.status)
		}
	}
}

func TestManilaSecurityServiceUpdateSendsExplicitNameAndType(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("PUT /v2/security-services/ss-1", func(w http.ResponseWriter, r *http.Request) {
		want := map[string]string{"name": `"renamed"`, "type": `"kerberos"`, "server": `"kdc.example.test"`}
		if got := manilaSupportStrings(manilaSupportEnvelope(t, r, "security_service")); !reflect.DeepEqual(got, want) {
			t.Errorf("body=%v", got)
		}
		testcloud.JSON(w, 200, manilaSecurityServiceBody)
	})
	name, server := "renamed", "kdc.example.test"
	if _, err := securityservices.New(newManilaSupportClient(cloud)).Update(context.Background(), "ss-1",
		securityservices.UpdateOpts{Name: &name, Type: "kerberos", Server: &server}); err != nil {
		t.Fatal(err)
	}
}

func TestManilaSecurityServiceGetDeleteAndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/security-services/{id}", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		if r.PathValue("id") == "missing" {
			testcloud.JSON(w, 404, `{}`)
			return
		}
		testcloud.JSON(w, 200, manilaSecurityServiceBody)
	})
	deleted := map[string]int{"ss-1": 202, "ss-2": 204, "ss-3": 200, "missing": 404}
	cloud.Mux.HandleFunc("DELETE /v2/security-services/{id}", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		w.WriteHeader(deleted[r.PathValue("id")])
	})
	api := securityservices.New(newManilaSupportClient(cloud))
	ctx := context.Background()
	value, err := api.Get(ctx, "ss-1")
	if err != nil || value.Domain != "example.test" || value.Status != "new" || value.DNSIP != "192.0.2.53" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	_, err = api.Get(ctx, "missing")
	manilaSupportUnexpected(t, err, 404)
	for _, id := range []string{"ss-1", "ss-2"} {
		if err := api.Delete(ctx, id); err != nil {
			t.Fatalf("delete %s: %v", id, err)
		}
	}
	manilaSupportUnexpected(t, api.Delete(ctx, "ss-3"), 200)
	manilaSupportUnexpected(t, api.Delete(ctx, "missing"), 404)
	if err := api.Remove(ctx, resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores a missing security service by default: %v", err)
	}
}

func TestManilaSecurityServiceListUsesDetailRouteFiltersAndSinglePage(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/security-services/detail", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		queries = append(queries, r.URL.Query())
		testcloud.JSON(w, 200, `{"security_services":[{"id":"ss-1","name":"one","type":"ldap"},{"id":"ss-2","name":"two","type":"kerberos"}],"next":"ignored"}`)
	})
	api := securityservices.New(newManilaSupportClient(cloud))
	ctx := context.Background()
	var ids []string
	for value, err := range api.List(ctx,
		securityservices.WithListOptions(securityservices.ListOpts{AllTenants: true, Type: securityservices.ActiveDirectory, ShareNetworkID: "sn-1", Name: "one"}),
		securityservices.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	want := url.Values{"all_tenants": {"true"}, "type": {"active_directory"}, "share_network_id": {"sn-1"}, "name": {"one"}, "vendor": {"x"}}
	if !reflect.DeepEqual(ids, []string{"ss-1", "ss-2"}) || len(queries) != 1 || !reflect.DeepEqual(queries[0], want) {
		t.Fatalf("ids=%v queries=%v", ids, queries)
	}
	queries = nil
	for range api.List(ctx) {
	}
	if len(queries) != 1 || len(queries[0]) != 0 {
		t.Fatalf("zero filters must be omitted: %v", queries)
	}
}

func TestManilaSecurityServiceNoContentListYieldsNothingAndFindMatchesExactName(t *testing.T) {
	cloud := testcloud.New(t)
	empty := true
	cloud.Mux.HandleFunc("GET /v2/security-services/dir", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/security-services/detail", func(w http.ResponseWriter, r *http.Request) {
		if empty {
			w.WriteHeader(204)
			return
		}
		testcloud.JSON(w, 200, `{"security_services":[{"id":"ss-1","name":"dir"},{"id":"ss-2","name":"dir-2"}]}`)
	})
	api := securityservices.New(newManilaSupportClient(cloud))
	for value, err := range api.List(context.Background()) {
		t.Fatalf("204 list yielded value=%+v error=%v", value, err)
	}
	empty = false
	value, err := api.Find(context.Background(), resource.Name("dir"))
	if err != nil || value.ID != "ss-1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
}

func TestManilaSecurityServiceCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
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
	api := securityservices.New(newManilaSupportClient(cloud))
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(canceled, "ss-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if err := api.Delete(canceled, "ss-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled delete: %v", err)
	}
	for _, err := range api.List(canceled) {
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled list: %v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Create(deadline, securityservices.CreateOpts{Type: securityservices.LDAP}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
