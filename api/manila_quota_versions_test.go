package api_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/JSYoo5B/gophercloudsdk/sharedfilesystems/v2/quotaclasssets"
	"github.com/JSYoo5B/gophercloudsdk/sharedfilesystems/v2/quotasets"
)

func TestManilaQuotaVersionLatestCannotChooseRoutesBeforeLookupOrHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("symbolic latest made quota/lookup HTTP: %s", r.URL)
	})
	ctx := context.Background()
	client := cloud.Client("shared-file-system", "/manila")
	client.Microversion = "2.39"
	api := quotasets.New(client)
	parent, err := api.InProject(ctx, resource.ID("p"), quotasets.WithIdentityClient(cloud.Client("identity", "/identity/v3")))
	if err != nil {
		t.Fatal(err)
	}
	user, err := parent.InUser(ctx, resource.ID("u"))
	if err != nil {
		t.Fatal(err)
	}
	shareType, err := parent.InShareType(ctx, resource.ID("t"))
	if err != nil {
		t.Fatal(err)
	}
	classes := quotaclasssets.New(client)
	class, err := classes.InClass(ctx, "default")
	if err != nil {
		t.Fatal(err)
	}
	client.Microversion = "latest"
	operations := map[string]func() error{
		"project name lookup": func() error { _, e := api.InProject(ctx, resource.Name("tenant")); return e },
		"current project":     func() error { _, e := api.CurrentProject(ctx); return e },
		"class binding":       func() error { _, e := classes.InClass(ctx, "default"); return e },
		"user name lookup":    func() error { _, e := parent.InUser(ctx, resource.Name("user")); return e },
		"type name lookup":    func() error { _, e := parent.InShareType(ctx, resource.Name("type")); return e },
		"project get":         func() error { _, e := parent.Get(ctx); return e },
		"user get":            func() error { _, e := user.Get(ctx); return e },
		"type get":            func() error { _, e := shareType.Get(ctx); return e },
		"class get":           func() error { _, e := class.Get(ctx); return e },
	}
	for name, operation := range operations {
		if err := operation(); !errors.Is(err, resource.ErrUnsupported) || !strings.Contains(err.Error(), "numeric microversion") {
			t.Errorf("%s err=%v", name, err)
		}
	}
	if client.Microversion != "latest" {
		t.Fatal("scope upgraded or mutated source version")
	}
}

func TestManilaQuotaVersionConflictsFailBeforeProjectOrClassLookup(t *testing.T) {
	for _, tc := range []struct {
		name, selected, kind string
		headers              map[string]string
	}{
		{"legacy same key", "2.39", "shared-file-system", map[string]string{"X-OpenStack-Manila-API-Version": "2.25"}},
		{"legacy lower case", "2.39", "sharev2", map[string]string{"x-openstack-manila-api-version": "2.25"}},
		{"legacy duplicate casing", "2.39", "share", map[string]string{"X-OpenStack-Manila-API-Version": "2.39", "x-openstack-manila-api-version": "2.6"}},
		{"generic lower version", "2.39", "shared-file-system", map[string]string{"OpenStack-API-Version": "shared-file-system 2.6"}},
		{"generic duplicate casing", "2.39", "share", map[string]string{"OpenStack-API-Version": "share 2.39", "openstack-api-version": "share 2.25"}},
		{"generic foreign service", "2.39", "shared-file-system", map[string]string{"OpenStack-API-Version": "volume 2.39"}},
		{"empty selected advanced legacy", "", "shared-file-system", map[string]string{"X-OpenStack-Manila-API-Version": "2.39"}},
		{"empty selected advanced generic", "", "share", map[string]string{"OpenStack-API-Version": "share 2.39"}},
		{"empty legacy override", "2.39", "shared-file-system", map[string]string{"X-OpenStack-Manila-API-Version": ""}},
		{"empty type no header", "2.39", "", nil},
		{"empty type generic only", "2.39", "", map[string]string{"OpenStack-API-Version": "shared-file-system 2.39"}},
		{"foreign type", "2.39", "compute", nil},
		{"foreign type with legacy", "2.39", "compute", map[string]string{"X-OpenStack-Manila-API-Version": "2.39"}},
		{"unrecognized type casing", "2.39", "Share", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("conflicting configuration made lookup/HTTP: %s", r.URL)
			})
			client := cloud.Client(tc.kind, "/manila/v2/catalog-project")
			client.Microversion = tc.selected
			client.MoreHeaders = tc.headers
			api := quotasets.New(client)
			if _, err := api.InProject(context.Background(), resource.Name("tenant"), quotasets.WithIdentityClient(cloud.Client("identity", "/identity/v3"))); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if _, err := api.CurrentProject(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if _, err := quotaclasssets.New(client).InClass(context.Background(), "default"); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestManilaQuotaVersionNativeAliasesPreserveMatchingHeadersAndRoutes(t *testing.T) {
	for _, kind := range []string{"shared-file-system", "sharev2", "share"} {
		for _, version := range []string{"", "2.6", "2.7", "2.25", "2.39"} {
			t.Run(kind+"/"+version, func(t *testing.T) {
				cloud := testcloud.New(t)
				root, classRoot := "quota-sets", "quota-class-sets"
				if version == "" || version == "2.6" {
					root, classRoot = "os-quota-sets", "os-quota-class-sets"
				}
				expected := version
				if expected == "" {
					expected = "2.0"
				}
				var calls atomic.Int32
				cloud.Mux.HandleFunc("/manila/v2/catalog-project/"+root+"/p", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Header.Get("X-OpenStack-Manila-API-Version") != expected {
						t.Errorf("legacy header changed: %v", r.Header)
					}
					testcloud.JSON(w, 200, `{"quota_set":{"shares":1}}`)
				})
				cloud.Mux.HandleFunc("/manila/v2/catalog-project/"+classRoot+"/default", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Header.Get("X-OpenStack-Manila-API-Version") != expected {
						t.Errorf("legacy class header changed: %v", r.Header)
					}
					testcloud.JSON(w, 200, `{"quota_class_set":{"shares":1}}`)
				})
				client := cloud.Client(kind, "/manila/v2/catalog-project")
				client.Microversion = version
				client.MoreHeaders = map[string]string{"x-openstack-manila-api-version": expected, "OpenStack-API-Version": kind + " " + expected, "X-Custom": "kept"}
				before := map[string]string{}
				for key, value := range client.MoreHeaders {
					before[key] = value
				}
				project, err := quotasets.New(client).InProject(context.Background(), resource.ID("p"))
				if err != nil {
					t.Fatal(err)
				}
				if _, err := project.Get(context.Background()); err != nil {
					t.Fatal(err)
				}
				class, err := quotaclasssets.New(client).InClass(context.Background(), "default")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := class.Get(context.Background()); err != nil {
					t.Fatal(err)
				}
				if calls.Load() != 2 || client.Microversion != version || !reflect.DeepEqual(client.MoreHeaders, before) {
					t.Fatal("version or source headers were mutated")
				}
			})
		}
	}
}

func TestManilaQuotaVersionBlankTypeNeedsLegacyHeaderEvenWithGenericHeader(t *testing.T) {
	for _, generic := range []string{"shared-file-system", "sharev2", "share"} {
		t.Run(generic, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/manila/quota-sets/p", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Header.Get("X-OpenStack-Manila-API-Version") != "2.39" || r.Header.Get("OpenStack-API-Version") != generic+" 2.39" {
					t.Errorf("matching explicit headers changed: %v", r.Header)
				}
				testcloud.JSON(w, 200, `{"quota_set":{}}`)
			})
			client := cloud.Client("", "/manila")
			client.Microversion = "2.39"
			client.MoreHeaders = map[string]string{"OpenStack-API-Version": generic + " 2.39"}
			if _, err := quotasets.New(client).InProject(context.Background(), resource.Name("tenant"), quotasets.WithIdentityClient(cloud.Client("identity", "/identity/v3"))); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("generic-only manual client accepted: %v", err)
			}
			client.MoreHeaders["X-OpenStack-Manila-API-Version"] = "2.39"
			scope, err := quotasets.New(client).InProject(context.Background(), resource.ID("p"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := scope.Get(context.Background()); err != nil || calls.Load() != 1 {
				t.Fatalf("calls=%d err=%v", calls.Load(), err)
			}
		})
	}
}

func TestManilaQuotaVersionChangedSourceBlocksEveryScopeOperation(t *testing.T) {
	for _, header := range []string{"X-OpenStack-Manila-API-Version", "x-openstack-manila-api-version", "OpenStack-API-Version"} {
		t.Run(header, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("changed version made quota/lookup HTTP: %s", r.URL)
			})
			client := cloud.Client("shared-file-system", "/manila")
			client.Microversion = "2.39"
			api := quotasets.New(client)
			parent, err := api.InProject(context.Background(), resource.ID("p"), quotasets.WithIdentityClient(cloud.Client("identity", "/identity/v3")))
			if err != nil {
				t.Fatal(err)
			}
			user, err := parent.InUser(context.Background(), resource.ID("u"))
			if err != nil {
				t.Fatal(err)
			}
			shareType, err := parent.InShareType(context.Background(), resource.ID("t"))
			if err != nil {
				t.Fatal(err)
			}
			classes := quotaclasssets.New(client)
			class, err := classes.InClass(context.Background(), "default")
			if err != nil {
				t.Fatal(err)
			}
			value := "2.6"
			if header == "OpenStack-API-Version" {
				value = "shared-file-system 2.6"
			}
			client.MoreHeaders = map[string]string{header: value}
			operations := map[string]func() error{
				"project get":      func() error { _, e := parent.Get(context.Background()); return e },
				"project defaults": func() error { _, e := parent.Defaults(context.Background()); return e },
				"project detail":   func() error { _, e := parent.Detail(context.Background()); return e },
				"project update":   func() error { _, e := parent.Update(context.Background(), quotasets.UpdateOpts{}); return e },
				"project reset":    func() error { _, e := parent.Reset(context.Background()); return e },
				"user lookup":      func() error { _, e := parent.InUser(context.Background(), resource.Name("user")); return e },
				"type lookup":      func() error { _, e := parent.InShareType(context.Background(), resource.Name("gold")); return e },
				"user get":         func() error { _, e := user.Get(context.Background()); return e },
				"user detail":      func() error { _, e := user.Detail(context.Background()); return e },
				"user update":      func() error { _, e := user.Update(context.Background(), quotasets.UpdateOpts{}); return e },
				"user reset":       func() error { _, e := user.Reset(context.Background()); return e },
				"type get":         func() error { _, e := shareType.Get(context.Background()); return e },
				"type detail":      func() error { _, e := shareType.Detail(context.Background()); return e },
				"type update":      func() error { _, e := shareType.Update(context.Background(), quotasets.UpdateOpts{}); return e },
				"type reset":       func() error { _, e := shareType.Reset(context.Background()); return e },
				"class get":        func() error { _, e := class.Get(context.Background()); return e },
				"class update":     func() error { _, e := class.Update(context.Background(), quotaclasssets.UpdateOpts{}); return e },
				"class construct":  func() error { _, e := classes.InClass(context.Background(), "other"); return e },
			}
			for name, operation := range operations {
				if err := operation(); !errors.Is(err, resource.ErrInvalidOption) {
					t.Errorf("%s err=%v", name, err)
				}
			}
		})
	}
}
