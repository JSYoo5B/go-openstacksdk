package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules"
	"github.com/gophercloud/gophercloud/v2"
)

const shareAccessBase = "/manila/v2/project"
const shareAccessValue = `{"id":"rule-id","share_id":"share-id","access_type":"ip","access_to":"192.0.2.0/24","access_level":"rw","state":"active","access_key":"credential","metadata":{"owner":"ops"},"created_at":"2026-10-01T00:00:00.123456","updated_at":null,"lock_visibility":false,"lock_deletion":true,"lock_reason":"maintenance","vendor":null}`

func shareAccessClient(cloud *testcloud.Cloud, version string) *gophercloud.ServiceClient {
	client := cloud.Client("shared-file-system", shareAccessBase)
	client.Microversion = version
	return client
}

func shareAccessScope(t *testing.T, cloud *testcloud.Cloud) *shareaccessrules.AccessRuleScope {
	t.Helper()
	scope, err := shareaccessrules.New(shareAccessClient(cloud, "2.82")).InShare(context.Background(), resource.ID("share-id"))
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func checkShareAccessHeaders(t *testing.T, r *http.Request, version string) {
	t.Helper()
	if r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-OpenStack-Manila-API-Version") != version || r.Header.Get("OpenStack-API-Version") != "shared-file-system "+version {
		t.Errorf("incorrect auth/microversion headers: %v", r.Header)
	}
}

func shareAccessBody(t *testing.T, r *http.Request, envelope string) map[string]any {
	t.Helper()
	var body map[string]map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Error(err)
	}
	if len(body) != 1 || body[envelope] == nil {
		t.Errorf("incorrect %s body: %v", envelope, body)
	}
	return body[envelope]
}

func TestShareAccessScopeResolvesParentOnceAndPreservesModernResponse(t *testing.T) {
	cloud := testcloud.New(t)
	var lookups, lists, actions atomic.Int32
	cloud.Mux.HandleFunc(shareAccessBase+"/shares/detail", func(w http.ResponseWriter, r *http.Request) {
		lookups.Add(1)
		checkShareAccessHeaders(t, r, "2.82")
		if r.URL.Query().Get("name") != "team.prod" {
			t.Errorf("parent name query=%s", r.URL.RawQuery)
		}
		testcloud.JSON(w, 200, `{"shares":[{"id":"other","name":"teamXprod"},{"id":"share-id","name":"team.prod"}]}`)
	})
	cloud.Mux.HandleFunc(shareAccessBase+"/share-access-rules", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		checkShareAccessHeaders(t, r, "2.82")
		if r.Method != http.MethodGet || r.URL.Query().Get("share_id") != "share-id" || r.URL.Query().Get("vendor") != "a&b" || r.Header.Get("X-Vendor") != "list" {
			t.Errorf("list method=%s query=%s headers=%v", r.Method, r.URL.RawQuery, r.Header)
		}
		w.Header().Set("X-Request-Id", "list-request")
		// Native Manila List fixtures omit share_id. The scope preserves that omission.
		testcloud.JSON(w, 200, `{"access_list":[{"id":"rule-id","state":"active","access_to":"192.0.2.0/24"}]}`)
	})
	cloud.Mux.HandleFunc(shareAccessBase+"/share-access-rules/rule-id", func(w http.ResponseWriter, r *http.Request) {
		checkShareAccessHeaders(t, r, "2.82")
		if r.Method != http.MethodGet || r.URL.RawQuery != "" {
			t.Errorf("get method=%s query=%s", r.Method, r.URL.RawQuery)
		}
		w.Header().Set("X-Request-Id", "get-request")
		testcloud.JSON(w, 200, `{"access":`+shareAccessValue+`}`)
	})
	cloud.Mux.HandleFunc(shareAccessBase+"/shares/share-id/action", func(w http.ResponseWriter, r *http.Request) {
		checkShareAccessHeaders(t, r, "2.82")
		if r.Method != http.MethodPost || r.URL.RawQuery != "" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("action method=%s query=%s headers=%v", r.Method, r.URL.RawQuery, r.Header)
		}
		switch actions.Add(1) {
		case 1:
			body := shareAccessBody(t, r, "allow_access")
			want := map[string]any{"access_type": "ip", "access_to": "192.0.2.0/24", "access_level": "ro", "metadata": map[string]any{"owner": "ops"}, "lock_visibility": false, "lock_deletion": true, "lock_reason": "maintenance", "vendor:enabled": false}
			if !reflect.DeepEqual(body, want) || r.Header.Get("Accept") != "application/json" || r.Header.Get("X-Vendor") != "allow" {
				t.Errorf("allow body=%v headers=%v", body, r.Header)
			}
			w.Header().Set("X-Request-Id", "allow-request")
			testcloud.JSON(w, 200, `{"access":`+shareAccessValue+`}`)
		case 2:
			body := shareAccessBody(t, r, "deny_access")
			want := map[string]any{"access_id": "rule-id", "unrestrict": false, "vendor:enabled": false}
			if !reflect.DeepEqual(body, want) || r.Header.Get("Accept") != "" || r.Header.Get("X-Vendor") != "deny" {
				t.Errorf("deny body=%v headers=%v", body, r.Header)
			}
			w.WriteHeader(202)
		default:
			t.Errorf("unexpected repeated action")
		}
	})
	ctx := context.Background()
	client := shareAccessClient(cloud, "2.82")
	scope, err := shareaccessrules.New(client).InShare(ctx, resource.Name("team.prod"))
	if err != nil || scope.ShareID() != "share-id" {
		t.Fatalf("scope=%v err=%v", scope, err)
	}
	for range 2 {
		values, err := scope.All(ctx, shareaccessrules.WithListQuery("vendor", "a&b"), shareaccessrules.WithListHeader("X-Vendor", "list"))
		if err != nil || len(values) != 1 || values[0].ShareID != "" || values[0].ParentShareID != "share-id" || values[0].Header.Get("X-Request-Id") != "list-request" {
			t.Fatalf("list=%v err=%v", values, err)
		}
	}
	value, err := scope.Get(ctx, "rule-id", shareaccessrules.WithGetHeader("X-Vendor", "get"))
	if err != nil || value.State != "active" || value.AccessKey != "credential" || value.Metadata["owner"] != "ops" || value.CreatedAt.Nanosecond() != 123456000 || !value.UpdatedAt.IsZero() || value.LockVisibility == nil || *value.LockVisibility || value.LockDeletion == nil || !*value.LockDeletion || value.LockReason == nil || *value.LockReason != "maintenance" || string(value.Body["vendor"]) != "null" || value.Header.Get("X-Request-Id") != "get-request" {
		t.Fatalf("get=%+v err=%v", value, err)
	}
	metadata := map[string]string{"owner": "ops"}
	option := shareaccessrules.WithMetadata(metadata)
	metadata["owner"] = "changed"
	value, err = scope.Allow(ctx, shareaccessrules.AllowOpts{AccessType: "ip", AccessTo: "192.0.2.0/24"}, option,
		shareaccessrules.WithAccessLevel("rw"), shareaccessrules.WithAccessLevel("ro"), shareaccessrules.WithLockVisibility(false), shareaccessrules.WithLockDeletion(true), shareaccessrules.WithLockReason("maintenance"), shareaccessrules.WithAllowField("vendor:enabled", false), shareaccessrules.WithAllowHeader("X-Vendor", "allow"))
	if err != nil || value.ID != "rule-id" || value.ParentShareID != "share-id" || value.Header.Get("X-Request-Id") != "allow-request" {
		t.Fatalf("allow=%+v err=%v", value, err)
	}
	if err := scope.Deny(ctx, "rule-id", shareaccessrules.WithUnrestrict(false), shareaccessrules.WithDenyField("vendor:enabled", false), shareaccessrules.WithDenyHeader("X-Vendor", "deny")); err != nil {
		t.Fatal(err)
	}
	if lookups.Load() != 1 || lists.Load() != 2 || actions.Load() != 2 || client.Microversion != "2.82" || client.Endpoint != cloud.Server.URL+shareAccessBase+"/" {
		t.Fatalf("lookup=%d list=%d actions=%d version=%s endpoint=%s", lookups.Load(), lists.Load(), actions.Load(), client.Microversion, client.Endpoint)
	}
}

func TestShareAccessScopeCreateOmissionEmptyMetadataAndKnownIdentityOnFailure(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc(shareAccessBase+"/shares/share-id/action", func(w http.ResponseWriter, r *http.Request) {
		body := shareAccessBody(t, r, "allow_access")
		if body["access_type"] != "vendor-backend" || body["access_to"] != "backend-subject" {
			t.Errorf("backend-specific identity was changed: %v", body)
		}
		if _, exists := body["access_level"]; exists {
			t.Errorf("omitted access level was materialized: %v", body)
		}
		if calls.Add(1) == 1 {
			if _, exists := body["metadata"]; exists {
				t.Errorf("omitted metadata was materialized: %v", body)
			}
			testcloud.JSON(w, 200, `{"access":{"id":"created-id","state":"queued_to_apply"}}`)
			return
		}
		if metadata, ok := body["metadata"].(map[string]any); !ok || len(metadata) != 0 {
			t.Errorf("explicit empty metadata must be {}: %v", body)
		}
		w.Header().Set("X-Request-Id", "created-but-undecodable")
		testcloud.JSON(w, 200, `{"access":{"id":"created-id","created_at":"bad-date"}}`)
	})
	scope := shareAccessScope(t, cloud)
	opts := shareaccessrules.AllowOpts{AccessType: "vendor-backend", AccessTo: "backend-subject"}
	value, err := scope.Create(context.Background(), opts)
	if err != nil || value.ID != "created-id" || value.State != "queued_to_apply" {
		t.Fatalf("create=%+v err=%v", value, err)
	}
	value, err = scope.Create(context.Background(), opts, shareaccessrules.WithMetadata(nil))
	if err == nil || value == nil || value.ID != "created-id" || value.Header.Get("X-Request-Id") != "created-but-undecodable" || string(value.Body["created_at"]) != `"bad-date"` {
		t.Fatalf("known created resource must survive decoding failure: value=%+v err=%v", value, err)
	}
}

func TestShareAccessScopeRejectsPreflightInputsWithoutRequests(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 500, "{}")
	})
	ctx := context.Background()
	api := shareaccessrules.New(shareAccessClient(cloud, "2.82"))
	for _, ref := range []resource.Ref{{}, resource.ID(".."), resource.ID("bad/id"), resource.ID("bad%id"), resource.Name(" ")} {
		if _, err := api.InShare(ctx, ref); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("parent=%v err=%v", ref, err)
		}
	}
	if _, err := shareaccessrules.New(nil).InShare(ctx, resource.ID("share-id")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	scope := shareAccessScope(t, cloud)
	for _, id := range []string{"", ".", "..", "a/b", "a%2fb", "a?b", "a#b", "a b"} {
		if _, err := scope.Get(ctx, id); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("get id=%q err=%v", id, err)
		}
		if err := scope.Deny(ctx, id); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("deny id=%q err=%v", id, err)
		}
	}
	for _, ref := range []resource.Ref{resource.Name("subject"), resource.Name("rule-id")} {
		for _, run := range []func() error{
			func() error { _, err := scope.Find(ctx, ref, resource.WithIgnoreMissing()); return err },
			func() error { _, err := scope.ResolveID(ctx, ref); return err },
			func() error { return scope.Delete(ctx, ref) },
			func() error { _, err := scope.Wait(ctx, ref, "active"); return err },
			func() error { return scope.WaitDeleted(ctx, ref) },
		} {
			if err := run(); !errors.Is(err, resource.ErrUnsupported) {
				t.Errorf("rules have IDs and no name: %v", err)
			}
		}
	}
	for _, opts := range []shareaccessrules.AllowOpts{{}, {AccessType: "ip"}, {AccessTo: "192.0.2.1"}, {AccessType: "ip", AccessTo: "192.0.2.1", AccessLevel: "admin"}} {
		if _, err := scope.Allow(ctx, opts); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("allow=%+v err=%v", opts, err)
		}
	}
	valid := shareaccessrules.AllowOpts{AccessType: "ip", AccessTo: "192.0.2.1"}
	for _, option := range []shareaccessrules.AllowOption{
		nil, shareaccessrules.WithAllowField("access_type", "override"), shareaccessrules.WithAllowField("access_level", "ro"), shareaccessrules.WithAllowField("lock_visibility", false),
		shareaccessrules.WithAllowHeader("x-auth-token", "override"), shareaccessrules.WithAllowHeader("openstack-api-version", "share 2.82"), shareaccessrules.WithAllowHeader("x-openstack-manila-api-version", "2.82"), shareaccessrules.WithAllowHeader("content-type", "text/plain"),
		request.WithQuery[shareaccessrules.AllowOpts]("unsupported", "yes"), request.WithArgument[shareaccessrules.AllowOpts]("ignored", true),
	} {
		if _, err := scope.Allow(ctx, valid, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("allow option err=%v", err)
		}
	}
	for _, option := range []shareaccessrules.DenyOption{nil, shareaccessrules.WithDenyField("access_id", "other"), shareaccessrules.WithDenyField("unrestrict", false), shareaccessrules.WithDenyHeader("Accept", "override"), request.WithQuery[shareaccessrules.DenyOpts]("unsupported", "yes"), request.WithArgument[shareaccessrules.DenyOpts]("ignored", true)} {
		if err := scope.Deny(ctx, "rule-id", option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("deny option err=%v", err)
		}
	}
	for _, option := range []shareaccessrules.GetOption{nil, request.WithField[shareaccessrules.GetOpts]("unsupported", true), request.WithQuery[shareaccessrules.GetOpts]("unsupported", "yes"), request.WithArgument[shareaccessrules.GetOpts]("ignored", true), shareaccessrules.WithGetHeader("X-Auth-Token", "other")} {
		if _, err := scope.Get(ctx, "rule-id", option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("get option err=%v", err)
		}
	}
	for _, option := range []shareaccessrules.ListOption{nil, shareaccessrules.WithListQuery("share_id", "share-id"), request.WithField[shareaccessrules.ListOpts]("unsupported", true), request.WithArgument[shareaccessrules.ListOpts]("ignored", true), shareaccessrules.WithListHeader("Accept", "override")} {
		if _, err := scope.All(ctx, option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("list option err=%v", err)
		}
	}
	if _, err := scope.Find(ctx, resource.ID("rule-id"), nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if err := scope.Delete(ctx, resource.ID("rule-id"), nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := scope.Wait(ctx, resource.ID("rule-id"), "active", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if err := scope.WaitDeleted(ctx, resource.ID("rule-id"), resource.WithTimeout(0)); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	for _, run := range []func() error{
		func() error { _, err := api.InShare(canceled, resource.Name("team")); return err },
		func() error { _, err := scope.Get(canceled, "rule-id"); return err },
		func() error { _, err := scope.All(canceled); return err },
		func() error { _, err := scope.Allow(canceled, valid); return err },
		func() error { return scope.Deny(canceled, "rule-id") },
		func() error { _, err := scope.ResolveID(canceled, resource.ID("rule-id")); return err },
		func() error { return scope.Delete(canceled, resource.ID("rule-id")) },
		func() error { _, err := scope.Find(canceled, resource.ID("rule-id")); return err },
		func() error { _, err := scope.Wait(canceled, resource.ID("rule-id"), "active"); return err },
		func() error { return scope.WaitDeleted(canceled, resource.ID("rule-id")) },
	} {
		if err := run(); !errors.Is(err, context.Canceled) {
			t.Errorf("cancellation err=%v", err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("preflight made %d HTTP requests", calls.Load())
	}
}

func TestShareAccessScopeChecksSelectedMicroversionIncludingExplicitFalse(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	var expected string
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		checkShareAccessHeaders(t, r, expected)
		if r.Method == http.MethodPost {
			if expected == "2.100" {
				body := shareAccessBody(t, r, "allow_access")
				if body["lock_visibility"] != false {
					t.Errorf("explicit false lock lost: %v", body)
				}
			}
		}
		testcloud.JSON(w, 200, `{"access":`+shareAccessValue+`}`)
	})
	ctx := context.Background()
	for _, version := range []string{"", "2.44", "2.9", "1.45", "3.45", "latest", "2", "2.-45", "2.45.extra", " 2.45", "2.999999999999999999999999999999"} {
		client := shareAccessClient(cloud, version)
		if _, err := shareaccessrules.New(client).InShare(ctx, resource.Name("team")); err == nil || client.Microversion != version {
			t.Errorf("version=%q selected=%q err=%v", version, client.Microversion, err)
		}
	}
	client := shareAccessClient(cloud, "2.81")
	scope, err := shareaccessrules.New(client).InShare(ctx, resource.ID("share-id"))
	if err != nil {
		t.Fatal(err)
	}
	valid := shareaccessrules.AllowOpts{AccessType: "ip", AccessTo: "192.0.2.1"}
	for _, option := range []shareaccessrules.AllowOption{shareaccessrules.WithLockVisibility(false), shareaccessrules.WithLockDeletion(false), shareaccessrules.WithLockReason("")} {
		if _, err := scope.Allow(ctx, valid, option); !errors.Is(err, resource.ErrUnsupported) {
			t.Fatal(err)
		}
	}
	if err := scope.Deny(ctx, "rule-id", shareaccessrules.WithUnrestrict(false)); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatalf("unsupported versions made %d requests", calls.Load())
	}
	for _, version := range []string{"2.45", "2.82", "2.100"} {
		expected = version
		client := shareAccessClient(cloud, version)
		scope, err := shareaccessrules.New(client).InShare(ctx, resource.ID("share-id"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := scope.Get(ctx, "rule-id"); err != nil || client.Microversion != version {
			t.Fatalf("version=%s err=%v", version, err)
		}
		if version == "2.100" {
			if _, err := scope.Allow(ctx, valid, shareaccessrules.WithLockVisibility(false)); err != nil || client.Microversion != version {
				t.Fatal(err)
			}
		}
	}
}

func TestShareAccessScopeCanBeSharedWithoutMutatingClientOrOptions(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		checkShareAccessHeaders(t, r, "2.82")
		if r.Method == http.MethodPost {
			body := shareAccessBody(t, r, "allow_access")
			if !reflect.DeepEqual(body["metadata"], map[string]any{"owner": "ops"}) {
				t.Errorf("metadata changed: %v", body)
			}
		}
		testcloud.JSON(w, 200, `{"access":`+shareAccessValue+`}`)
	})
	client := shareAccessClient(cloud, "2.82")
	scope, err := shareaccessrules.New(client).InShare(context.Background(), resource.ID("share-id"))
	if err != nil {
		t.Fatal(err)
	}
	option := shareaccessrules.WithMetadata(map[string]string{"owner": "ops"})
	var work sync.WaitGroup
	for range 8 {
		work.Add(1)
		go func() {
			defer work.Done()
			if _, err := scope.Get(context.Background(), "rule-id"); err != nil {
				t.Error(err)
			}
			if _, err := scope.Allow(context.Background(), shareaccessrules.AllowOpts{AccessType: "ip", AccessTo: "192.0.2.1"}, option); err != nil {
				t.Error(err)
			}
		}()
	}
	work.Wait()
	if client.Endpoint != cloud.Server.URL+shareAccessBase+"/" || client.Microversion != "2.82" {
		t.Fatalf("shared client changed: %+v", client)
	}
}

func TestShareAccessScopeListStopsBeforeDecodingLaterEntries(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc(shareAccessBase+"/share-access-rules", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 200, `{"access_list":[{"id":"first"},{"id":"later","created_at":"not-a-date"}],"links":[{"rel":"next","href":"`+cloud.Server.URL+`/must-not-follow"}]}`)
	})
	scope := shareAccessScope(t, cloud)
	count := 0
	for value, err := range scope.List(context.Background()) {
		if err != nil || value.ID != "first" {
			t.Fatalf("value=%+v err=%v", value, err)
		}
		count++
		break
	}
	if count != 1 || calls.Load() != 1 {
		t.Fatalf("items=%d requests=%d", count, calls.Load())
	}
	if values, err := scope.All(context.Background()); err == nil || values != nil {
		t.Fatalf("partial All must fail without claiming completeness: %v %v", values, err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	for value, err := range scope.List(canceled) {
		if value != nil {
			cancel()
			continue
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("between-item cancellation=%v", err)
		}
	}
	cancel()
}

func TestShareAccessScopeListEmptyAndMalformedResponses(t *testing.T) {
	for _, tc := range []struct {
		body string
		ok   bool
	}{
		{`{"access_list":[]}`, true}, {`{}`, false}, {`{"access_list":null}`, false}, {`{"access_list":{}}`, false}, {`{"access_list":[null]}`, false}, {`{"access_list":[42]}`, false}, {`{"access_list":[{}]}`, false}, {`{"access_list":[{"id":"rule-id","share_id":"other"}]}`, false},
	} {
		t.Run(tc.body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, tc.body) })
			values, err := shareAccessScope(t, cloud).All(context.Background())
			if tc.ok && (err != nil || values == nil || len(values) != 0) || !tc.ok && (err == nil || values != nil) {
				t.Fatalf("values=%v err=%v", values, err)
			}
		})
	}
}

func TestShareAccessScopeWaitsForStateAndDeletionUsingOriginalAccessID(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc(shareAccessBase+"/share-access-rules/rule-id", func(w http.ResponseWriter, r *http.Request) {
		state := "queued_to_apply"
		switch calls.Add(1) {
		case 2:
			state = "applying"
		case 3, 4:
			state = "active"
		case 5:
			state = "denying"
		case 6:
			testcloud.JSON(w, 404, `{"itemNotFound":{"message":"gone"}}`)
			return
		}
		// A changed body ID must not change the URL used by subsequent polls.
		testcloud.JSON(w, 200, `{"access":{"id":"wire-id","share_id":"share-id","state":"`+state+`"}}`)
	})
	cloud.Mux.HandleFunc(shareAccessBase+"/shares/share-id", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"share":{"id":"share-id"}}`)
	})
	scope := shareAccessScope(t, cloud)
	value, err := scope.Wait(context.Background(), resource.ID("rule-id"), "ACTIVE", resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second))
	if err != nil || value.State != "active" || calls.Load() != 3 {
		t.Fatalf("value=%+v err=%v calls=%d", value, err, calls.Load())
	}
	if err := scope.WaitDeleted(context.Background(), resource.ID("rule-id"), resource.WithPollInterval(time.Millisecond), resource.WithTimeout(time.Second)); err != nil || calls.Load() != 6 {
		t.Fatalf("delete waiter=%v calls=%d", err, calls.Load())
	}
}
