package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/quotaclasssets"
	"github.com/gophercloud/gophercloud/v2"
)

func newManilaQuotaClassScope(t *testing.T, cloud *testcloud.Cloud, version string) *quotaclasssets.QuotaClassScope {
	t.Helper()
	client := cloud.Client("shared-file-system", "/manila/v2/catalog-project")
	client.Microversion = version
	scope, err := quotaclasssets.New(client).InClass(context.Background(), "fixed-class")
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func TestManilaQuotaClassNamedScopesUseVersionedRoutesWithoutLookups(t *testing.T) {
	for _, tc := range []struct{ version, root string }{{"", "os-quota-class-sets"}, {"2.6", "os-quota-class-sets"}, {"2.7", "quota-class-sets"}, {"2.80", "quota-class-sets"}} {
		t.Run("version-"+tc.version, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/manila/v2/catalog-project/"+tc.root+"/fixed-class", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "GET" || r.URL.RawQuery != "" || r.Header.Get("X-OpenStack-Manila-API-Version") != tc.version {
					t.Errorf("bad class request: %s %s %v", r.Method, r.URL, r.Header)
				}
				testcloud.JSON(w, 200, `{"quota_class_set":{"id":"wire-class","shares":20}}`)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("class name triggered unrelated request: %s", r.URL)
			})
			scope := newManilaQuotaClassScope(t, cloud, tc.version)
			if scope.ClassName() != "fixed-class" || calls.Load() != 0 {
				t.Fatal("class construction performed lookup")
			}
			value, err := scope.Get(context.Background())
			if err != nil || value.ClassName != "fixed-class" || value.ID != "wire-class" || *value.Shares != 20 || value.StatusCode != 200 || calls.Load() != 1 {
				t.Fatalf("value=%+v calls=%d err=%v", value, calls.Load(), err)
			}
		})
	}
}

func TestManilaQuotaClassResponsePreservesNullOmissionAndExtensions(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/manila/v2/catalog-project/quota-class-sets/fixed-class", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Class-Request", "kept")
		testcloud.JSON(w, 200, `{"quota_class_set":{"shares":9007199254740993,"snapshots":null,"gigabytes":-1,"vendor":{"counter":9223372036854775807}}}`)
	})
	value, err := newManilaQuotaClassScope(t, cloud, "2.80").Get(context.Background())
	if err != nil || *value.Shares != 9007199254740993 || value.Snapshots != nil || value.Backups != nil || *value.Gigabytes != -1 || string(value.Body["snapshots"]) != "null" || value.Body["backups"] != nil || !strings.Contains(string(value.Body["vendor"]), "9223372036854775807") || value.Header.Get("X-Class-Request") != "kept" {
		t.Fatalf("value=%+v err=%v", value, err)
	}
}

func TestManilaQuotaClassAllTypedLimitsKeepExactWireValuesAndSnapshots(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	keys := []string{"gigabytes", "snapshots", "shares", "snapshot_gigabytes", "share_groups", "share_group_snapshots", "share_networks", "share_replicas", "replica_gigabytes", "per_share_gigabytes", "backups", "backup_gigabytes"}
	cloud.Mux.HandleFunc("/manila/v2/catalog-project/quota-class-sets/fixed-class", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		quota := body["quota_class_set"]
		if r.Method != "PUT" || r.URL.RawQuery != "" || len(body) != 1 || len(quota) != 14 {
			t.Errorf("bad request: %s %s body=%v", r.Method, r.URL, body)
		}
		for _, key := range keys {
			if string(quota[key]) != "9007199254740993" {
				t.Errorf("quota %s lost precision: %s", key, quota[key])
			}
		}
		if string(quota["vendor"]) != `{"counter":9007199254740995}` || string(quota["other"]) != `{"mode":"old"}` {
			t.Errorf("extensions were not captured: %v", quota)
		}
		testcloud.JSON(w, 200, `{"quota_class_set":{"shares":9007199254740993}}`)
	})
	limit := int64(9007199254740993)
	extra := map[string]any{"vendor": map[string]any{"counter": int64(9007199254740995)}}
	opts := quotaclasssets.UpdateOpts{Gigabytes: &limit, Snapshots: &limit, Shares: &limit, SnapshotGigabytes: &limit, ShareGroups: &limit, ShareGroupSnapshots: &limit, ShareNetworks: &limit, ShareReplicas: &limit, ReplicaGigabytes: &limit, PerShareGigabytes: &limit, Backups: &limit, BackupGigabytes: &limit, Extra: extra}
	option := quotaclasssets.WithQuotaOptions(opts)
	other := map[string]any{"mode": "old"}
	field := quotaclasssets.WithUpdateField("other", other)
	limit = 99
	extra["vendor"].(map[string]any)["counter"] = 99
	other["mode"] = "new"
	scope := newManilaQuotaClassScope(t, cloud, "2.80")
	for range 2 {
		value, err := scope.Update(context.Background(), quotaclasssets.UpdateOpts{}, option, field)
		if err != nil || *value.Shares != 9007199254740993 || value.ClassName != "fixed-class" {
			t.Fatalf("value=%+v err=%v", value, err)
		}
	}
	if calls.Load() != 2 {
		t.Fatal("unexpected PUT count")
	}
}

func TestManilaQuotaClassUpdateDistinguishesZeroUnlimitedAndOmitted(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/manila/v2/catalog-project/quota-class-sets/fixed-class", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		quota := body["quota_class_set"]
		if len(quota) != 2 || string(quota["shares"]) != "0" || string(quota["gigabytes"]) != "-1" {
			t.Errorf("bad partial update: %v", quota)
		}
		testcloud.JSON(w, 200, `{"quota_class_set":{"shares":0,"gigabytes":-1}}`)
	})
	zero, unlimited := int64(0), int64(-1)
	value, err := newManilaQuotaClassScope(t, cloud, "2.80").Update(context.Background(), quotaclasssets.UpdateOpts{Shares: &zero, Gigabytes: &unlimited})
	if err != nil || *value.Shares != 0 || *value.Gigabytes != -1 {
		t.Fatalf("value=%+v err=%v", value, err)
	}
}

func TestManilaQuotaClassInvalidOptionsAndMethodSetsPreventUnsupportedRequests(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("invalid class request: %s", r.URL) })
	scope := newManilaQuotaClassScope(t, cloud, "2.80")
	for _, key := range []string{"shares", "force", "id", "class_name", "quota_class_name", "project_id", "user_id", "share_type", "usage", "reservation", "quota_class_set", " "} {
		for _, option := range []quotaclasssets.UpdateOption{quotaclasssets.WithUpdateField(key, 1), quotaclasssets.WithQuotaOptions(quotaclasssets.UpdateOpts{Extra: map[string]any{key: 1}})} {
			if _, err := scope.Update(context.Background(), quotaclasssets.UpdateOpts{}, option); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("key=%s err=%v", key, err)
			}
		}
	}
	bad := int64(-2)
	if _, err := scope.Update(context.Background(), quotaclasssets.UpdateOpts{Shares: &bad}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, option := range []quotaclasssets.UpdateOption{nil, request.WithQuery[quotaclasssets.UpdateOpts]("project_id", "other"), request.WithArgument[quotaclasssets.UpdateOpts]("force", true), quotaclasssets.WithUpdateField("bad", make(chan int))} {
		if _, err := scope.Update(context.Background(), quotaclasssets.UpdateOpts{}, option); err == nil {
			t.Fatal("unsupported class option accepted")
		}
	}
	for _, method := range []string{"Delete", "Reset", "Defaults", "Detail", "List", "Find", "Wait", "InUser", "InProject", "InShareType"} {
		if _, present := reflect.TypeOf(scope).MethodByName(method); present {
			t.Errorf("quota class exposes unsupported %s", method)
		}
	}
	for _, name := range []string{"", " ", "bad/name", "..", "bad?query", "bad%20name"} {
		if _, err := quotaclasssets.New(cloud.Client("shared-file-system", "/manila")).InClass(context.Background(), name); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("name=%q err=%v", name, err)
		}
	}
}

func TestManilaQuotaClassStatusAndDecodeErrorsKeepOriginalCauses(t *testing.T) {
	for _, operation := range []string{"GET", "PUT"} {
		for _, tc := range []struct {
			name   string
			status int
			body   string
			decode bool
		}{{"accepted", 202, `{"quota_class_set":{}}`, false}, {"nonauthoritative", 203, `{"quota_class_set":{}}`, false}, {"missing", 404, `{}`, false}, {"forbidden", 403, `{}`, false}, {"badtype", 200, `{"quota_class_set":{"shares":"bad"}}`, true}, {"overflow", 200, `{"quota_class_set":{"shares":9223372036854775808}}`, true}, {"null", 200, `{"quota_class_set":null}`, false}, {"wrongEnvelope", 200, `{"quota_set":{}}`, false}} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				cloud.Mux.HandleFunc("/manila/v2/catalog-project/quota-class-sets/fixed-class", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, tc.status, tc.body) })
				scope := newManilaQuotaClassScope(t, cloud, "2.80")
				var err error
				if operation == "GET" {
					_, err = scope.Get(context.Background())
				} else {
					_, err = scope.Update(context.Background(), quotaclasssets.UpdateOpts{})
				}
				if err == nil {
					t.Fatal("invalid response accepted")
				}
				if tc.status != 200 {
					var cause gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &cause) || cause.Actual != tc.status {
						t.Fatalf("HTTP cause lost: %v", err)
					}
				}
				if tc.status == 404 && !errors.Is(err, resource.ErrNotFound) {
					t.Fatal(err)
				}
				if tc.decode {
					var cause *json.UnmarshalTypeError
					if !errors.As(err, &cause) {
						t.Fatalf("decode cause lost: %v", err)
					}
				}
			})
		}
	}
}

func TestManilaQuotaClassRedirectCannotChangeFixedClassOrMethod(t *testing.T) {
	for _, status := range []int{302, 307} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/manila/v2/catalog-project/quota-class-sets/fixed-class", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				http.Redirect(w, r, "/manila/v2/catalog-project/quota-class-sets/other", status)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("redirect changed class: %s %s", r.Method, r.URL)
			})
			_, err := newManilaQuotaClassScope(t, cloud, "2.80").Update(context.Background(), quotaclasssets.UpdateOpts{})
			if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
				t.Fatalf("calls=%d err=%v", calls.Load(), err)
			}
		})
	}
}

func TestManilaQuotaClassPreflightAndInFlightContextCancellation(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/manila/v2/catalog-project/quota-class-sets/fixed-class", func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() })
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := newManilaQuotaClassScope(t, cloud, "2.80").Get(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	for _, version := range []string{"3.1", "2.bad", "2.-1", "2.025"} {
		client := cloud.Client("shared-file-system", "/manila")
		client.Microversion = version
		if _, err := quotaclasssets.New(client).InClass(context.Background(), "default"); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("version=%s err=%v", version, err)
		}
	}
	var zero quotaclasssets.QuotaClassScope
	if _, err := zero.Update(context.Background(), quotaclasssets.UpdateOpts{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	if _, err := zero.Get(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := quotaclasssets.New(nil).InClass(context.Background(), "default"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
