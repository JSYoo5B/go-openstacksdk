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

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/quotasets"
	"github.com/gophercloud/gophercloud/v2"
)

func TestManilaScopedQuotaIDsFixEveryOperationAndResponseTarget(t *testing.T) {
	for _, selector := range []string{"user_id", "share_type"} {
		t.Run(selector, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			path := "/manila/v2/catalog-project/quota-sets/quota-project"
			id := "scope+&id"
			cloud.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if len(r.URL.Query()) != 1 || r.URL.Query().Get(selector) != id || r.URL.RawQuery != selector+"=scope%2B%26id" {
					t.Errorf("scope widened: %s", r.URL)
				}
				if r.Header.Get("X-OpenStack-Manila-API-Version") != "2.80" {
					t.Errorf("lost microversion header: %v", r.Header)
				}
				w.Header().Set("X-Quota-Scope", selector)
				switch r.Method {
				case "GET":
					testcloud.JSON(w, 200, `{"quota_set":{"id":"wire-project","shares":10,"vendor":9007199254740993}}`)
				case "PUT":
					var body map[string]map[string]json.RawMessage
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					quota := body["quota_set"]
					if string(quota["shares"]) != "0" || string(quota["force"]) != "false" || quota["user_id"] != nil || quota["share_type"] != nil || quota["project_id"] != nil {
						t.Errorf("bad scoped update: %v", quota)
					}
					testcloud.JSON(w, 200, `{"quota_set":{"id":"wire-project","shares":0}}`)
				case "DELETE":
					w.WriteHeader(202)
				default:
					t.Errorf("unexpected method: %s", r.Method)
				}
			})
			cloud.Mux.HandleFunc(path+"/detail", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "GET" || len(r.URL.Query()) != 1 || r.URL.Query().Get(selector) != id {
					t.Errorf("detail scope widened: %s", r.URL)
				}
				testcloud.JSON(w, 200, `{"quota_set":{"shares":{"limit":10,"in_use":1,"reserved":0}}}`)
			})
			scope := newManilaQuotaScope(t, cloud, "2.80")
			var get func(context.Context) (*quotasets.QuotaResource, error)
			var detail func(context.Context) (*quotasets.QuotaDetailResource, error)
			var update func(context.Context, quotasets.UpdateOpts, ...quotasets.UpdateOption) (*quotasets.QuotaResource, error)
			var reset func(context.Context, ...quotasets.ResetOption) (*quotasets.ResetResponse, error)
			if selector == "user_id" {
				child, err := scope.InUser(context.Background(), resource.ID(id))
				if err != nil || child.ProjectID() != "quota-project" || child.UserID() != id {
					t.Fatalf("child=%v err=%v", child, err)
				}
				get, detail, update, reset = child.Get, child.Detail, child.Update, child.Reset
			} else {
				child, err := scope.InShareType(context.Background(), resource.ID(id))
				if err != nil || child.ProjectID() != "quota-project" || child.ShareTypeID() != id {
					t.Fatalf("child=%v err=%v", child, err)
				}
				get, detail, update, reset = child.Get, child.Detail, child.Update, child.Reset
			}
			if calls.Load() != 0 {
				t.Fatal("explicit IDs issued HTTP")
			}
			value, err := get(context.Background())
			if err != nil || value.ID != "wire-project" || value.ProjectID != "quota-project" || string(value.Body["vendor"]) != "9007199254740993" || value.Header.Get("X-Quota-Scope") != selector || value.StatusCode != 200 || selector == "user_id" && (value.UserID != id || value.ShareTypeID != "") || selector == "share_type" && (value.ShareTypeID != id || value.UserID != "") {
				t.Fatalf("quota=%+v err=%v", value, err)
			}
			usage, err := detail(context.Background())
			if err != nil || *usage.Shares.InUse != 1 || usage.ProjectID != "quota-project" || selector == "user_id" && usage.UserID != id || selector == "share_type" && usage.ShareTypeID != id {
				t.Fatalf("detail=%+v err=%v", usage, err)
			}
			zero := int64(0)
			value, err = update(context.Background(), quotasets.UpdateOpts{Shares: &zero}, quotasets.WithUpdateForce(false))
			if err != nil || *value.Shares != 0 || selector == "user_id" && value.UserID != id || selector == "share_type" && value.ShareTypeID != id {
				t.Fatalf("updated=%+v err=%v", value, err)
			}
			removed, err := reset(context.Background())
			if err != nil || removed.ProjectID != "quota-project" || removed.StatusCode != 202 || removed.Header.Get("X-Quota-Scope") != selector || selector == "user_id" && removed.UserID != id || selector == "share_type" && removed.ShareTypeID != id || calls.Load() != 4 {
				t.Fatalf("reset=%+v calls=%d err=%v", removed, calls.Load(), err)
			}
		})
	}
}

func TestManilaUserQuotaExactNamesInheritOrOverrideKeystone(t *testing.T) {
	for _, override := range []bool{false, true} {
		t.Run(map[bool]string{false: "inherited", true: "override"}[override], func(t *testing.T) {
			cloud := testcloud.New(t)
			identityPath := "/identity/v3"
			if override {
				identityPath = "/alternate/v3"
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc(identityPath+"/users", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("name") != "same" || r.Header.Get("X-OpenStack-Manila-API-Version") != "" {
					t.Errorf("bad user lookup: %s %v", r.URL, r.Header)
				}
				testcloud.JSON(w, 200, `{"users":[{"id":"wrong","name":"same-other"},{"id":"user-fixed","name":"same"}]}`)
			})
			cloud.Mux.HandleFunc("/manila/v2/catalog-project/quota-sets/quota-project", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("user_id") != "user-fixed" {
					t.Errorf("bad user target: %s", r.URL)
				}
				testcloud.JSON(w, 200, `{"quota_set":{"shares":1}}`)
			})
			client := cloud.Client("shared-file-system", "/manila/v2/catalog-project")
			client.Microversion = "2.80"
			scope, err := quotasets.New(client).InProject(context.Background(), resource.ID("quota-project"), quotasets.WithIdentityClient(cloud.Client("identity", "/identity/v3")))
			if err != nil {
				t.Fatal(err)
			}
			var options []quotasets.ProjectOption
			if override {
				options = append(options, quotasets.WithIdentityClient(cloud.Client("identity", identityPath)))
			}
			user, err := scope.InUser(context.Background(), resource.Name("same"), options...)
			if err != nil || user.UserID() != "user-fixed" || calls.Load() != 1 {
				t.Fatalf("user=%v calls=%d err=%v", user, calls.Load(), err)
			}
			for range 2 {
				if _, err := user.Get(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			if calls.Load() != 1 {
				t.Fatal("user name was re-resolved")
			}
		})
	}
}

func TestManilaShareTypeQuotaNamesUseServiceCollectionAfterVersionGate(t *testing.T) {
	cloud := testcloud.New(t)
	var lookups atomic.Int32
	cloud.Mux.HandleFunc("/manila/v2/catalog-project/types", func(w http.ResponseWriter, r *http.Request) {
		lookups.Add(1)
		if r.URL.RawQuery != "" || r.Header.Get("X-OpenStack-Manila-API-Version") != "2.39" {
			t.Errorf("bad share type lookup: %s %v", r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"share_types":[{"id":"wrong","name":"gold-other"},{"id":"type-fixed","name":"gold"}]}`)
	})
	cloud.Mux.HandleFunc("/manila/v2/catalog-project/quota-sets/quota-project", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("share_type") != "type-fixed" || r.URL.Query().Has("user_id") {
			t.Errorf("bad share-type target: %s", r.URL)
		}
		testcloud.JSON(w, 200, `{"quota_set":{"shares":1}}`)
	})
	client := cloud.Client("shared-file-system", "/manila/v2/catalog-project")
	client.Microversion = "2.38"
	parent, err := quotasets.New(client).InProject(context.Background(), resource.ID("quota-project"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parent.InShareType(context.Background(), resource.Name("gold")); !errors.Is(err, resource.ErrUnsupported) || lookups.Load() != 0 {
		t.Fatalf("lookups=%d err=%v", lookups.Load(), err)
	}
	client.Microversion = "2.39"
	child, err := parent.InShareType(context.Background(), resource.Name("gold"))
	if err != nil || child.ShareTypeID() != "type-fixed" || lookups.Load() != 1 {
		t.Fatalf("child=%v lookups=%d err=%v", child, lookups.Load(), err)
	}
	if _, err := child.Get(context.Background()); err != nil {
		t.Fatal(err)
	}
	client.Microversion = "2.38"
	if _, err := child.Reset(context.Background()); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal("downgraded client reset share-type quotas")
	}
}

func TestManilaScopedQuotaLookupFailuresPreventQuotaRequests(t *testing.T) {
	for _, kind := range []string{"user", "sharetype"} {
		for _, tc := range []struct {
			name, rows string
			status     int
			want       error
		}{{"missing", `[]`, 200, resource.ErrNotFound}, {"ambiguous", `[{"id":"one","name":"target"},{"id":"two","name":"target"}]`, 200, resource.ErrAmbiguous}, {"invalidID", `[{"id":"bad/id","name":"target"}]`, 200, resource.ErrInvalidOption}, {"forbidden", `[]`, 403, nil}} {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				path, key := "/identity/v3/users", "users"
				if kind == "sharetype" {
					path, key = "/manila/v2/catalog-project/types", "share_types"
				}
				cloud.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, tc.status, `{"`+key+`":`+tc.rows+`}`) })
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("lookup failure made quota request: %s", r.URL) })
				client := cloud.Client("shared-file-system", "/manila/v2/catalog-project")
				client.Microversion = "2.80"
				parent, err := quotasets.New(client).InProject(context.Background(), resource.ID("quota-project"), quotasets.WithIdentityClient(cloud.Client("identity", "/identity/v3")))
				if err != nil {
					t.Fatal(err)
				}
				if kind == "user" {
					_, err = parent.InUser(context.Background(), resource.Name("target"))
				} else {
					_, err = parent.InShareType(context.Background(), resource.Name("target"))
				}
				if err == nil || tc.want != nil && !errors.Is(err, tc.want) {
					t.Fatalf("err=%v", err)
				}
				if tc.status == 403 {
					var cause gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &cause) || cause.Actual != 403 {
						t.Fatalf("HTTP cause lost: %v", err)
					}
				}
			})
		}
	}
}

func TestManilaScopedQuotaPreflightAndMethodSetsCannotWidenScope(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("invalid scope made HTTP: %s", r.URL) })
	parent := newManilaQuotaScope(t, cloud, "2.80")
	if _, err := parent.InUser(context.Background(), resource.Name("user")); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	for _, ref := range []resource.Ref{{}, resource.ID("bad/id")} {
		if _, err := parent.InUser(context.Background(), ref); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
		if _, err := parent.InShareType(context.Background(), ref); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := parent.InUser(context.Background(), resource.ID("user"), nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, typ := range []reflect.Type{reflect.TypeOf((*quotasets.UserQuotaScope)(nil)), reflect.TypeOf((*quotasets.ShareTypeQuotaScope)(nil))} {
		for _, method := range []string{"Defaults", "InUser", "InShareType", "List", "Find", "Wait"} {
			if _, present := typ.MethodByName(method); present {
				t.Errorf("%s exposes %s", typ, method)
			}
		}
	}
	var user quotasets.UserQuotaScope
	var shareType quotasets.ShareTypeQuotaScope
	if _, err := user.Reset(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := shareType.Reset(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := user.Reset(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := shareType.Reset(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	child, err := parent.InShareType(context.Background(), resource.ID("type"))
	if err != nil {
		t.Fatal(err)
	}
	zero := int64(0)
	if _, err := child.Update(context.Background(), quotasets.UpdateOpts{ShareNetworks: &zero}); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("share type accepted share_networks: %v", err)
	}
}

func TestManilaScopedQuotaRedirectsCannotDropOrChangeSelector(t *testing.T) {
	for _, selector := range []string{"user_id", "share_type"} {
		for _, location := range []string{"", "?" + selector + "=different", "?" + selector + "=fixed&other=1"} {
			t.Run(selector+"/"+location, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				path := "/manila/v2/catalog-project/quota-sets/quota-project"
				cloud.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Method != "DELETE" || r.URL.Query().Get(selector) != "fixed" {
						t.Errorf("redirect widened operation: %s %s", r.Method, r.URL)
					}
					http.Redirect(w, r, path+location, 307)
				})
				parent := newManilaQuotaScope(t, cloud, "2.80")
				var err error
				if selector == "user_id" {
					child, e := parent.InUser(context.Background(), resource.ID("fixed"))
					if e != nil {
						t.Fatal(e)
					}
					_, err = child.Reset(context.Background())
				} else {
					child, e := parent.InShareType(context.Background(), resource.ID("fixed"))
					if e != nil {
						t.Fatal(e)
					}
					_, err = child.Reset(context.Background())
				}
				if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
					t.Fatalf("calls=%d err=%v", calls.Load(), err)
				}
			})
		}
	}
}

func TestManilaScopedQuotaReauthenticationKeepsBodyAndFixedUser(t *testing.T) {
	cloud := testcloud.New(t)
	var calls, reauth atomic.Int32
	cloud.Provider.ReauthFunc = func(ctx context.Context) error { reauth.Add(1); cloud.Provider.SetToken("fresh-token"); return nil }
	cloud.Mux.HandleFunc("/manila/v2/catalog-project/quota-sets/quota-project", func(w http.ResponseWriter, r *http.Request) {
		call := calls.Add(1)
		var body map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if r.Method != "PUT" || r.URL.Query().Get("user_id") != "user-fixed" || string(body["quota_set"]["shares"]) != "9007199254740993" || !strings.Contains(string(body["quota_set"]["vendor"]), "9007199254740995") {
			t.Errorf("reauth changed request: %s %s body=%v", r.Method, r.URL, body)
		}
		if call == 1 {
			if r.Header.Get("X-Auth-Token") != "test-token" {
				t.Error("wrong initial token")
			}
			testcloud.JSON(w, 401, `{}`)
			return
		}
		if r.Header.Get("X-Auth-Token") != "fresh-token" {
			t.Error("stale token after reauth")
		}
		testcloud.JSON(w, 200, `{"quota_set":{"shares":9007199254740993}}`)
	})
	parent := newManilaQuotaScope(t, cloud, "2.80")
	child, err := parent.InUser(context.Background(), resource.ID("user-fixed"))
	if err != nil {
		t.Fatal(err)
	}
	limit := int64(9007199254740993)
	extra := map[string]any{"vendor": map[string]any{"counter": int64(9007199254740995)}}
	option := quotasets.WithQuotaOptions(quotasets.UpdateOpts{Shares: &limit, Extra: extra})
	limit = 99
	extra["vendor"].(map[string]any)["counter"] = 99
	result, err := child.Update(context.Background(), quotasets.UpdateOpts{}, option)
	if err != nil || *result.Shares != 9007199254740993 || result.UserID != "user-fixed" || calls.Load() != 2 || reauth.Load() != 1 {
		t.Fatalf("result=%v calls=%d reauth=%d err=%v", result, calls.Load(), reauth.Load(), err)
	}
}
