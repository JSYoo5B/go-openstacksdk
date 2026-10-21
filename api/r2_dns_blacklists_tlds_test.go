package api_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/dns/v2/blacklists"
	"github.com/JSYoo5B/go-openstacksdk/dns/v2/tlds"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const r2dnsBlacklistBody = `{"id":"bl-1","pattern":"^example\\.org\\.$","description":null,"created_at":"2026-01-02T03:04:05.000000","updated_at":null,"links":{"self":"http://dns.example/v2/blacklists/bl-1"},"extra":1}`

func TestR2DNSBlacklistCreateFlatBodyPreservesOmitNullEmptyAndExtensions(t *testing.T) {
	cases := []struct {
		name string
		opts blacklists.CreateOpts
		more []blacklists.CreateOption
		want map[string]string
	}{
		{"omitted description", blacklists.CreateOpts{Pattern: "p"}, nil, map[string]string{"pattern": `"p"`}},
		{"empty description", blacklists.CreateOpts{Pattern: "p", Description: request.Present("")}, nil, map[string]string{"pattern": `"p"`, "description": `""`}},
		{"null description and false extension", blacklists.CreateOpts{Pattern: "p", Description: request.Null[string]()}, []blacklists.CreateOption{blacklists.WithCreateField("flag", false)}, map[string]string{"pattern": `"p"`, "description": `null`, "flag": `false`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("POST /v2/blacklists", func(w http.ResponseWriter, r *http.Request) {
				if got := r2dnsBody(t, r); !reflect.DeepEqual(got, tc.want) {
					t.Errorf("body=%v", got)
				}
				testcloud.JSON(w, 201, r2dnsBlacklistBody)
			})
			got, err := blacklists.New(r2dnsClient(cloud)).Create(context.Background(), tc.opts, tc.more...)
			if err != nil || got.ID != "bl-1" || got.StatusCode != 201 || string(got.Body["description"]) != "null" || string(got.Body["extra"]) != "1" {
				t.Fatalf("%+v %v", got, err)
			}
		})
	}
}

func TestR2DNSBlacklistRejectsCoreExtensionCollisionAndCancelledContextBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	calls := r2dnsCounter(t, cloud, "/", 500, "")
	api := blacklists.New(r2dnsClient(cloud))
	_, err := api.Create(context.Background(), blacklists.CreateOpts{Pattern: "p"}, blacklists.WithCreateField("pattern", "x"))
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("collision: %v", err)
	}
	if _, err := api.Update(context.Background(), "bl-1", blacklists.UpdateOpts{}, blacklists.WithUpdateField("description", "x")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("update collision: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(ctx, "bl-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
	if _, err := api.Get(context.Background(), "a/b"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("bad id: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestR2DNSBlacklistGetUpdateDeleteCodesAndMissingPolicy(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/blacklists/bl-1", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, r2dnsBlacklistBody) })
	cloud.Mux.HandleFunc("GET /v2/blacklists/gone", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("PATCH /v2/blacklists/bl-1", func(w http.ResponseWriter, r *http.Request) {
		if got := r2dnsBody(t, r); !reflect.DeepEqual(got, map[string]string{"description": `null`}) {
			t.Errorf("patch body=%v", got)
		}
		testcloud.JSON(w, 200, r2dnsBlacklistBody)
	})
	var deletes atomic.Int32
	cloud.Mux.HandleFunc("DELETE /v2/blacklists/bl-1", func(w http.ResponseWriter, r *http.Request) {
		deletes.Add(1)
		w.WriteHeader(204)
	})
	cloud.Mux.HandleFunc("DELETE /v2/blacklists/bad-code", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	cloud.Mux.HandleFunc("DELETE /v2/blacklists/gone", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api := blacklists.New(r2dnsClient(cloud))
	ctx := context.Background()
	if v, err := api.Get(ctx, "bl-1"); err != nil || v.Pattern != `^example\.org\.$` {
		t.Fatalf("get %+v %v", v, err)
	}
	if _, err := api.Get(ctx, "gone"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("missing get: %v", err)
	}
	if _, err := api.Update(ctx, "bl-1", blacklists.UpdateOpts{Description: request.Null[string]()}); err != nil {
		t.Fatal(err)
	}
	if ack, err := api.Delete(ctx, "bl-1"); err != nil || ack.StatusCode != 204 {
		t.Fatalf("delete %+v %v", ack, err)
	}
	if _, err := api.Delete(ctx, "bad-code"); err == nil {
		t.Fatal("200 must not be accepted for DELETE")
	}
	if _, err := api.Delete(ctx, "gone"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("strict delete: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("gone")); err != nil {
		t.Fatalf("Remove ignores missing by default: %v", err)
	}
	if err := api.Remove(ctx, resource.ID("gone"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("Remove strict: %v", err)
	}
}

func TestR2DNSBlacklistListFiltersHeadersAndLinksNextPaging(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/blacklists", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("pattern") != "x" {
			t.Errorf("query=%s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("marker") == "" {
			testcloud.JSON(w, 200, `{"blacklists":[{"id":"a","pattern":"1"}],"links":{"next":"`+cloud.Server.URL+`/v2/blacklists?pattern=x&marker=a"}}`)
			return
		}
		testcloud.JSON(w, 200, `{"blacklists":[{"id":"b","pattern":"2"}],"links":{}}`)
	})
	var ids []string
	for v, err := range blacklists.New(r2dnsClient(cloud)).List(context.Background(), blacklists.WithListOptions(blacklists.ListOpts{Pattern: "x"})) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, v.ID)
	}
	if !reflect.DeepEqual(ids, []string{"a", "b"}) {
		t.Fatalf("ids=%v", ids)
	}
}

func TestR2DNSTLDCreateGetUpdateDeleteAndFindIdentity(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /v2/tlds", func(w http.ResponseWriter, r *http.Request) {
		if got := r2dnsBody(t, r); !reflect.DeepEqual(got, map[string]string{"name": `"com"`, "description": `""`}) {
			t.Errorf("body=%v", got)
		}
		testcloud.JSON(w, 201, `{"id":"t1","name":"com","description":""}`)
	})
	cloud.Mux.HandleFunc("PATCH /v2/tlds/t1", func(w http.ResponseWriter, r *http.Request) {
		if got := r2dnsBody(t, r); !reflect.DeepEqual(got, map[string]string{"name": `"net"`}) {
			t.Errorf("patch=%v", got)
		}
		testcloud.JSON(w, 200, `{"id":"t1","name":"net"}`)
	})
	cloud.Mux.HandleFunc("DELETE /v2/tlds/t1", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	// find_tld: GET by the identifier first (a name is rejected with 400), then list by name.
	cloud.Mux.HandleFunc("GET /v2/tlds/net", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 400, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/tlds", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("name") {
		case "net":
			testcloud.JSON(w, 200, `{"tlds":[{"id":"t1","name":"net"}]}`)
		default:
			testcloud.JSON(w, 200, `{"tlds":[]}`)
		}
	})
	api := tlds.New(r2dnsClient(cloud))
	ctx := context.Background()
	if v, err := api.Create(ctx, tlds.CreateOpts{Name: "com", Description: request.Present("")}); err != nil || v.ID != "t1" {
		t.Fatalf("create %+v %v", v, err)
	}
	if v, err := api.Update(ctx, "t1", tlds.UpdateOpts{Name: request.Present("net")}); err != nil || v.Name != "net" {
		t.Fatalf("update %+v %v", v, err)
	}
	if v, err := api.FindIdentity(ctx, "net"); err != nil || v == nil || v.ID != "t1" {
		t.Fatalf("find by name %+v %v", v, err)
	}
	if v, err := api.FindIdentity(ctx, "org"); err != nil || v != nil {
		t.Fatalf("missing defaults to nil: %+v %v", v, err)
	}
	if _, err := api.FindIdentity(ctx, "org", resource.WithIdentityFindIgnoreMissing(false)); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("strict: %v", err)
	}
	if ack, err := api.Delete(ctx, "t1"); err != nil || ack.StatusCode != 204 {
		t.Fatalf("delete %+v %v", ack, err)
	}
}
