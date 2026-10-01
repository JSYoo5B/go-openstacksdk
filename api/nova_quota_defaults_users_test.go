package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/compute/v2/quotasets"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

func TestNovaQuotaDefaultsPreserveProjectAndRawResponse(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/nova/os-quota-sets/project-fixed/defaults", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.URL.RawQuery != "" || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.56" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("X-Client-Extension") != "preserved" {
			t.Errorf("defaults request=%s %s header=%s", r.Method, r.URL, r.Header)
		}
		w.Header().Add("X-Openstack-Request-Id", "req-defaults")
		w.Header().Add("X-Vendor", "one")
		w.Header().Add("X-Vendor", "two")
		testcloud.JSON(w, 200, `{"quota_set":{"id":"wire-project","cores":-1,"instances":0,"networks":8,"counter":9007199254740993,"vendor":{"array":[1,null]},"optional":null}}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("defaults changed the target: %s %s", r.Method, r.URL)
		testcloud.JSON(w, 404, "{}")
	})
	client := cloud.Client("compute", "/nova")
	client.Microversion = "2.56"
	client.MoreHeaders = map[string]string{"X-Client-Extension": "preserved"}
	scope, err := quotasets.New(client).InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil || calls.Load() != 0 {
		t.Fatalf("scope=%v calls=%d err=%v", scope, calls.Load(), err)
	}
	first, err := scope.Defaults(context.Background())
	if err != nil || first == nil || first.ProjectID != "project-fixed" || first.ID != "wire-project" || first.Cores != -1 || first.Instances != 0 || string(first.Body["networks"]) != "8" || string(first.Body["counter"]) != "9007199254740993" || string(first.Body["optional"]) != "null" || string(first.Body["vendor"]) != `{"array":[1,null]}` || first.Header.Get("X-Openstack-Request-Id") != "req-defaults" || len(first.Header.Values("X-Vendor")) != 2 {
		t.Fatalf("defaults=%+v err=%v", first, err)
	}
	first.Header["X-Vendor"][0] = "changed"
	first.Body["counter"][0] = '0'
	second, err := scope.Defaults(context.Background())
	if err != nil || second.Header.Values("X-Vendor")[0] != "one" || string(second.Body["counter"]) != "9007199254740993" || calls.Load() != 2 || scope.ProjectID() != "project-fixed" {
		t.Fatalf("second=%+v calls=%d err=%v", second, calls.Load(), err)
	}
}

func TestNovaQuotaDefaultsPreserveHTTPAndDecodeErrors(t *testing.T) {
	for _, status := range []int{403, 404, 203} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			body := `{"error":{"message":"defaults denied"}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Openstack-Request-Id", "denied-defaults")
				testcloud.JSON(w, status, body)
			})
			value, err := newProjectQuotaScope(t, cloud).Defaults(context.Background())
			var response gophercloud.ErrUnexpectedResponseCode
			var operation *resource.OperationError
			if value != nil || !errors.As(err, &response) || response.Actual != status || string(response.Body) != body || response.ResponseHeader.Get("X-Openstack-Request-Id") != "denied-defaults" || !errors.As(err, &operation) || operation.Operation != "Defaults" || errors.Is(err, resource.ErrNotFound) != (status == 404) {
				t.Fatalf("quota=%v response=%+v operation=%+v err=%v", value, response, operation, err)
			}
		})
	}
	for _, body := range []string{`{}`, `{"quota_set":null}`, `{"quota_set":[]}`, `{"quota_set":"bad"}`, `{"quota_set":{"cores":"bad"}}`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, body) })
			value, err := newProjectQuotaScope(t, cloud).Defaults(context.Background())
			if value != nil || err == nil {
				t.Fatalf("defaults=%v err=%v", value, err)
			}
			if body == `{"quota_set":{"cores":"bad"}}` {
				var decode *json.UnmarshalTypeError
				if !errors.As(err, &decode) {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestNovaQuotaDefaultsRespectContext(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); <-r.Context().Done() })
	scope := newProjectQuotaScope(t, cloud)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if value, err := scope.Defaults(ctx); value != nil || !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatalf("defaults=%v calls=%d err=%v", value, calls.Load(), err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if value, err := scope.Defaults(ctx); value != nil || !errors.Is(err, context.DeadlineExceeded) || calls.Load() != 1 {
		t.Fatalf("defaults=%v calls=%d err=%v", value, calls.Load(), err)
	}
}

func newNovaUserQuotaScope(t *testing.T, cloud *testcloud.Cloud, id string) *quotasets.UserQuotaScope {
	t.Helper()
	scope, err := newProjectQuotaScope(t, cloud).InUser(context.Background(), resource.ID(id))
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func TestNovaQuotaUserOperationsKeepQueryTargetAndRawResponse(t *testing.T) {
	cloud := testcloud.New(t)
	const userID = "user+tag&scope=chosen"
	query := url.Values{"user_id": []string{userID}}.Encode()
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/nova/os-quota-sets/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.RawQuery != query || len(r.URL.Query()) != 1 || len(r.URL.Query()["user_id"]) != 1 || r.URL.Query().Get("user_id") != userID || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.56" || r.Header.Get("X-Quota-Client") != "preserved" || r.Header.Get("X-Auth-Token") != "test-token" {
			t.Errorf("request=%s %s headers=%s", r.Method, r.URL, r.Header)
		}
		w.Header().Set("X-Openstack-Request-Id", "user-"+r.Method)
		switch r.Method {
		case http.MethodGet:
			testcloud.JSON(w, 200, `{"quota_set":{"id":"wire-project-or-user","user_id":"wire-other-user","cores":-1,"instances":0,"networks":8,"counter":9007199254740993,"optional":null}}`)
		case http.MethodPut:
			var body struct {
				Quota map[string]json.RawMessage `json:"quota_set"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if len(body.Quota) != 4 || string(body.Quota["cores"]) != "-1" || string(body.Quota["instances"]) != "0" || string(body.Quota["force"]) != "false" || string(body.Quota["vendor"]) != `{"counter":9007199254740993,"tier":"original"}` {
				t.Errorf("update=%s", body.Quota)
			}
			testcloud.JSON(w, 200, `{"quota_set":{"id":"wire-other","cores":-1,"instances":0,"counter":9007199254740993}}`)
		case http.MethodDelete:
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Error(r.Method)
		}
	})
	cloud.Mux.HandleFunc("/nova/os-quota-sets/project-fixed/detail", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.URL.RawQuery != query || r.Header.Get("X-Quota-Client") != "preserved" {
			t.Errorf("detail=%s %s", r.Method, r.URL)
		}
		w.Header().Set("X-Openstack-Request-Id", "user-detail")
		testcloud.JSON(w, 200, `{"quota_set":{"id":"detail-wire-other","cores":{"limit":-1,"in_use":2,"reserved":3,"unknown":null},"instances":{"limit":0,"in_use":0,"reserved":0},"vendor":{"limit":9007199254740993,"in_use":4,"reserved":5}}}`)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("request escaped scope: %s %s", r.Method, r.URL)
		testcloud.JSON(w, 404, "{}")
	})
	client := cloud.Client("compute", "/nova")
	client.Microversion = "2.56"
	client.MoreHeaders = map[string]string{"X-Quota-Client": "preserved"}
	api := quotasets.New(client)
	project, err := api.InProject(context.Background(), resource.ID("project-fixed"))
	if err != nil {
		t.Fatal(err)
	}
	scope, err := project.InUser(context.Background(), resource.ID(userID))
	if err != nil || calls.Load() != 0 || scope.ProjectID() != "project-fixed" || scope.UserID() != userID {
		t.Fatalf("scope=%v calls=%d err=%v", scope, calls.Load(), err)
	}
	if _, promoted := reflect.TypeOf(scope).MethodByName("Defaults"); promoted {
		t.Fatal("user scope promotes project-only Defaults")
	}
	value, err := scope.Get(context.Background())
	if err != nil || value == nil || value.ProjectID != "project-fixed" || value.UserID != userID || value.ID != "wire-project-or-user" || value.Cores != -1 || value.Instances != 0 || string(value.Body["user_id"]) != `"wire-other-user"` || string(value.Body["counter"]) != "9007199254740993" || string(value.Body["optional"]) != "null" || value.Header.Get("X-Openstack-Request-Id") != "user-GET" {
		t.Fatalf("quota=%+v err=%v", value, err)
	}
	detail, err := scope.Detail(context.Background())
	if err != nil || detail == nil || detail.ProjectID != "project-fixed" || detail.UserID != userID || detail.ID != "detail-wire-other" || detail.Cores.Limit != -1 || detail.Cores.InUse != 2 || detail.Cores.Reserved != 3 || string(detail.Body["vendor"]) != `{"in_use":4,"limit":9007199254740993,"reserved":5}` || detail.Header.Get("X-Openstack-Request-Id") != "user-detail" {
		t.Fatalf("detail=%+v err=%v", detail, err)
	}
	zero, unlimited := 0, -1
	extra := map[string]any{"counter": json.Number("9007199254740993"), "tier": "original"}
	field := quotasets.WithUpdateField("vendor", extra)
	extra["tier"] = "changed"
	updated, err := scope.Update(context.Background(), quotasets.UpdateOpts{}, quotasets.WithQuotaOptions(quotasets.UpdateOpts{Cores: &unlimited, Instances: &zero, Force: true}), quotasets.WithUpdateForce(false), field)
	if err != nil || updated == nil || updated.ProjectID != "project-fixed" || updated.UserID != userID || updated.ID != "wire-other" || updated.Cores != -1 || updated.Instances != 0 || string(updated.Body["counter"]) != "9007199254740993" || updated.Header.Get("X-Openstack-Request-Id") != "user-PUT" {
		t.Fatalf("updated=%+v err=%v", updated, err)
	}
	reset, err := scope.Reset(context.Background())
	if err != nil || reset == nil || reset.ProjectID != "project-fixed" || reset.UserID != userID || reset.Header.Get("X-Openstack-Request-Id") != "user-DELETE" || calls.Load() != 4 {
		t.Fatalf("reset=%+v calls=%d err=%v", reset, calls.Load(), err)
	}
	value.Header.Set("X-Openstack-Request-Id", "changed")
	value.Body["counter"][0] = '0'
	if updated.Header.Get("X-Openstack-Request-Id") != "user-PUT" || string(updated.Body["counter"]) != "9007199254740993" || detail.Header.Get("X-Openstack-Request-Id") != "user-detail" || reset.Header.Get("X-Openstack-Request-Id") != "user-DELETE" || api.RawClient() != client || client.ProviderClient != cloud.Provider || client.MoreHeaders["X-Quota-Client"] != "preserved" {
		t.Fatal("response or guarded client changed shared state")
	}
}

func TestNovaQuotaUserNameUsesInheritedOrOverrideKeystoneClientOnce(t *testing.T) {
	cloud := testcloud.New(t)
	var projects, users, override, nova atomic.Int32
	cloud.Mux.HandleFunc("/identity/v3/projects", func(w http.ResponseWriter, r *http.Request) {
		projects.Add(1)
		if r.URL.Query().Get("name") != "tenant" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"projects":[{"id":"project-fixed","name":"tenant"}]}`)
	})
	cloud.Mux.HandleFunc("/identity/v3/users", func(w http.ResponseWriter, r *http.Request) {
		users.Add(1)
		if r.URL.Query().Get("name") != "alice" || r.Header.Get("X-OpenStack-Nova-API-Version") != "" {
			t.Errorf("user lookup=%s header=%s", r.URL, r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"users": []map[string]string{{"id": "wrong", "name": "alice-extra"}}, "links": map[string]string{"next": cloud.Server.URL + "/identity/v3/users/page2"}})
	})
	cloud.Mux.HandleFunc("/identity/v3/users/page2", func(w http.ResponseWriter, r *http.Request) {
		users.Add(1)
		testcloud.JSON(w, 200, `{"users":[{"id":"resolved-user","name":"alice","domain_id":"one"}]}`)
	})
	cloud.Mux.HandleFunc("/override/v3/users", func(w http.ResponseWriter, r *http.Request) {
		override.Add(1)
		if r.URL.Query().Get("name") != "alice" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"users":[{"id":"override-user","name":"alice"}]}`)
	})
	cloud.Mux.HandleFunc("/nova/os-quota-sets/project-fixed", func(w http.ResponseWriter, r *http.Request) {
		nova.Add(1)
		id := r.URL.Query().Get("user_id")
		if id != "resolved-user" && id != "override-user" && id != "literal-user" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, projectQuotaLimits)
	})
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("wrong user resolver service: %s", r.URL)
		testcloud.JSON(w, 404, "{}")
	})
	client := cloud.Client("compute", "/nova")
	client.Microversion = "2.56"
	project, err := quotasets.New(client).InProject(context.Background(), resource.Name("tenant"), quotasets.WithIdentityClient(cloud.Client("identity", "/identity/v3")))
	if err != nil {
		t.Fatal(err)
	}
	inherited, err := project.InUser(context.Background(), resource.Name("alice"))
	if err != nil || inherited.UserID() != "resolved-user" || inherited.ProjectID() != "project-fixed" || users.Load() != 2 {
		t.Fatalf("scope=%v users=%d err=%v", inherited, users.Load(), err)
	}
	overridden, err := project.InUser(context.Background(), resource.Name("alice"), quotasets.WithIdentityClient(cloud.Client("identity", "/override/v3")))
	if err != nil || overridden.UserID() != "override-user" {
		t.Fatalf("scope=%v err=%v", overridden, err)
	}
	explicit, err := project.InUser(context.Background(), resource.ID("literal-user"))
	if err != nil || projects.Load() != 1 || users.Load() != 2 || override.Load() != 1 || nova.Load() != 0 {
		t.Fatalf("scope=%v lookups=%d/%d/%d nova=%d err=%v", explicit, projects.Load(), users.Load(), override.Load(), nova.Load(), err)
	}
	for _, scope := range []*quotasets.UserQuotaScope{inherited, overridden, explicit, inherited} {
		if value, err := scope.Get(context.Background()); err != nil || value.UserID != scope.UserID() || value.ProjectID != "project-fixed" {
			t.Fatalf("value=%+v err=%v", value, err)
		}
	}
	if projects.Load() != 1 || users.Load() != 2 || override.Load() != 1 || nova.Load() != 4 {
		t.Fatal("quota operation repeated project/user resolution")
	}
}

func TestNovaQuotaUserLookupFailuresDoNotCallNova(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		want       error
	}{
		{"missing", `{"users":[]}`, 200, resource.ErrNotFound},
		{"multiple domains", `{"users":[{"id":"one","name":"alice","domain_id":"one"},{"id":"two","name":"alice","domain_id":"two"}]}`, 200, resource.ErrAmbiguous},
		{"invalid user ID", `{"users":[{"id":"bad/user","name":"alice"}]}`, 200, resource.ErrInvalidOption},
		{"forbidden", `{"error":{"message":"user lookup denied"}}`, 403, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/identity/v3/users", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Openstack-Request-Id", "user-denied")
				testcloud.JSON(w, tc.status, tc.body)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("failed user lookup called Nova: %s", r.URL)
				testcloud.JSON(w, 404, "{}")
			})
			scope, err := newProjectQuotaScope(t, cloud).InUser(context.Background(), resource.Name("alice"), quotasets.WithIdentityClient(cloud.Client("identity", "/identity/v3")))
			if scope != nil || err == nil || calls.Load() != 1 || tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("scope=%v calls=%d err=%v", scope, calls.Load(), err)
			}
			if tc.status == 403 {
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) || response.Actual != 403 || string(response.Body) != tc.body || response.ResponseHeader.Get("X-Openstack-Request-Id") != "user-denied" {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestNovaQuotaUserInvalidInputsAndZeroScopeCannotResetProject(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, projectQuotaLimits) })
	project := newProjectQuotaScope(t, cloud)
	for _, ref := range []resource.Ref{resource.ID(""), resource.ID("bad/user"), resource.ID("user?other"), resource.ID("user%2Fother")} {
		if scope, err := project.InUser(context.Background(), ref); scope != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("scope=%v err=%v", scope, err)
		}
	}
	if scope, err := project.InUser(context.Background(), resource.Name("alice")); scope != nil || !errors.Is(err, resource.ErrUnsupported) {
		t.Fatalf("scope=%v err=%v", scope, err)
	}
	for _, option := range []quotasets.ProjectOption{nil, quotasets.WithIdentityClient(nil), quotasets.WithIdentityClient(cloud.Client("compute", "/nova"))} {
		if scope, err := project.InUser(context.Background(), resource.ID("user"), option); scope != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("scope=%v err=%v", scope, err)
		}
	}
	for _, scope := range []*quotasets.UserQuotaScope{nil, {}} {
		if value, err := scope.Reset(context.Background()); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("zero reset=%v err=%v", value, err)
		}
		if value, err := scope.Get(context.Background()); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("zero get=%v err=%v", value, err)
		}
	}
	scope, err := project.InUser(context.Background(), resource.ID("chosen-user"))
	if err != nil {
		t.Fatal(err)
	}
	negative := -2
	for _, options := range [][]quotasets.UpdateOption{{nil}, {quotasets.WithQuotaOptions(quotasets.UpdateOpts{Cores: &negative})}, {quotasets.WithUpdateField("cores", 0)}, {quotasets.WithUpdateField("force", false)}, {request.WithQuery[quotasets.UpdateOpts]("user_id", "other-user")}, {request.WithHeader[quotasets.UpdateOpts]("X-Vendor", "value")}} {
		if value, err := scope.Update(context.Background(), quotasets.UpdateOpts{}, options...); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("invalid update=%v err=%v", value, err)
		}
	}
	if value, err := scope.Reset(context.Background(), nil); value != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("reset=%v err=%v", value, err)
	}
	if calls.Load() != 0 {
		t.Fatalf("invalid scope widened or called HTTP: %d", calls.Load())
	}
}

func TestNovaQuotaUserErrorsKeepBothIdentitiesAndMissingPolicy(t *testing.T) {
	for _, status := range []int{403, 404} {
		for _, operation := range []string{"Get", "Detail", "Update", "Reset"} {
			t.Run(operation+http.StatusText(status), func(t *testing.T) {
				cloud := testcloud.New(t)
				const body = `{"error":{"message":"user quota denied"}}`
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					if len(r.URL.Query()["user_id"]) != 1 || r.URL.Query().Get("user_id") != "chosen-user" {
						t.Error(r.URL)
					}
					w.Header().Set("X-Openstack-Request-Id", "user-quota-denied")
					testcloud.JSON(w, status, body)
				})
				scope := newNovaUserQuotaScope(t, cloud, "chosen-user")
				err := callNovaUserQuota(scope, operation, context.Background())
				var response gophercloud.ErrUnexpectedResponseCode
				var sdk *resource.OperationError
				if !errors.As(err, &response) || response.Actual != status || string(response.Body) != body || response.ResponseHeader.Get("X-Openstack-Request-Id") != "user-quota-denied" || !errors.As(err, &sdk) || sdk.Operation != operation || errors.Is(err, resource.ErrNotFound) != (status == 404) || scope.ProjectID() != "project-fixed" || scope.UserID() != "chosen-user" {
					t.Fatalf("response=%+v sdk=%+v err=%v", response, sdk, err)
				}
				if status == 404 {
					var missing *resource.NotFoundError
					if !errors.As(err, &missing) || missing.Reference != "project-fixed/chosen-user" {
						t.Fatalf("missing=%+v err=%v", missing, err)
					}
				}
				if operation == "Reset" {
					value, err := scope.Reset(context.Background(), quotasets.WithResetIgnoreMissing(true))
					if value != nil || (err == nil) != (status == 404) {
						t.Fatalf("ignore reset=%v err=%v", value, err)
					}
					_, err = scope.Reset(context.Background(), quotasets.WithResetIgnoreMissing(true), quotasets.WithResetIgnoreMissing(false))
					if !gophercloud.ResponseCodeIs(err, status) {
						t.Fatal(err)
					}
				}
			})
		}
	}
}

func callNovaUserQuota(scope *quotasets.UserQuotaScope, operation string, ctx context.Context) error {
	var err error
	switch operation {
	case "Get":
		_, err = scope.Get(ctx)
	case "Detail":
		_, err = scope.Detail(ctx)
	case "Update":
		_, err = scope.Update(ctx, quotasets.UpdateOpts{})
	case "Reset":
		_, err = scope.Reset(ctx)
	}
	return err
}

func TestNovaQuotaUserResetAccepts202And204WithoutFollowUp(t *testing.T) {
	for _, status := range []int{202, 204, 200} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != http.MethodDelete || r.URL.Query().Get("user_id") != "chosen-user" {
					t.Errorf("reset=%s %s", r.Method, r.URL)
				}
				w.Header().Set("X-Openstack-Request-Id", "reset-user")
				w.WriteHeader(status)
			})
			value, err := newNovaUserQuotaScope(t, cloud, "chosen-user").Reset(context.Background())
			if status == 200 {
				var response gophercloud.ErrUnexpectedResponseCode
				if value != nil || !errors.As(err, &response) || response.Actual != 200 {
					t.Fatalf("reset=%v err=%v", value, err)
				}
			} else if err != nil || value == nil || value.ProjectID != "project-fixed" || value.UserID != "chosen-user" || value.Header.Get("X-Openstack-Request-Id") != "reset-user" {
				t.Fatalf("reset=%+v err=%v", value, err)
			}
			if calls.Load() != 1 {
				t.Fatal("reset made a follow-up request")
			}
		})
	}
}

type novaQuotaTransport func(*http.Request) (*http.Response, error)

func (f novaQuotaTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestNovaQuotaUserRetryAndReauthKeepOneQueryAndExactBody(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("large native int limit needs a 64-bit platform")
	}
	for _, mode := range []string{"reauth", "backoff", "retry"} {
		for _, operation := range []string{"Get", "Detail", "Update", "Reset"} {
			t.Run(mode+operation, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls, callbacks, transportCalls atomic.Int32
				const userID = "user+tag&scope=chosen"
				query := url.Values{"user_id": []string{userID}}.Encode()
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					n := calls.Add(1)
					if r.URL.RawQuery != query || len(r.URL.Query()["user_id"]) != 1 || !strings.HasPrefix(r.URL.Path, "/nova/os-quota-sets/project-fixed") {
						t.Errorf("retry widened target: %s", r.URL)
					}
					if r.Method == http.MethodPut {
						var body struct {
							Quota map[string]json.RawMessage `json:"quota_set"`
						}
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Error(err)
						}
						if string(body.Quota["cores"]) != "9007199254740993" || string(body.Quota["force"]) != "false" {
							t.Errorf("retry body=%s", body.Quota)
						}
					}
					if n == 1 {
						switch mode {
						case "reauth":
							testcloud.JSON(w, 401, `{"error":"refresh"}`)
						case "backoff":
							testcloud.JSON(w, 429, `{"error":"backoff"}`)
						case "retry":
							testcloud.JSON(w, 503, `{"error":"retry"}`)
						}
						return
					}
					if mode == "reauth" && r.Header.Get("X-Auth-Token") != "refreshed-token" {
						t.Errorf("reauth token=%q", r.Header.Get("X-Auth-Token"))
					}
					if r.Method == http.MethodDelete {
						w.WriteHeader(202)
					} else if strings.HasSuffix(r.URL.Path, "/detail") {
						testcloud.JSON(w, 200, `{"quota_set":{"cores":{"limit":-1,"in_use":2,"reserved":3}}}`)
					} else {
						testcloud.JSON(w, 200, projectQuotaLimits)
					}
				})
				originalTransport := cloud.Provider.HTTPClient.Transport
				cloud.Provider.HTTPClient.Transport = novaQuotaTransport(func(req *http.Request) (*http.Response, error) {
					transportCalls.Add(1)
					return originalTransport.RoundTrip(req)
				})
				switch mode {
				case "reauth":
					cloud.Provider.ReauthFunc = func(ctx context.Context) error {
						callbacks.Add(1)
						cloud.Provider.SetToken("refreshed-token")
						return nil
					}
				case "backoff":
					cloud.Provider.RetryBackoffFunc = func(ctx context.Context, response *gophercloud.ErrUnexpectedResponseCode, err error, count uint) error {
						callbacks.Add(1)
						if response.Actual != 429 || !strings.HasSuffix(response.URL, "?"+query) || count != 1 {
							t.Errorf("backoff=%+v count=%d", response, count)
						}
						return nil
					}
				case "retry":
					cloud.Provider.RetryFunc = func(ctx context.Context, method, endpoint string, opts *gophercloud.RequestOpts, err error, count uint) error {
						callbacks.Add(1)
						if !strings.HasSuffix(endpoint, "?"+query) || count != 1 {
							t.Errorf("retry=%s %s count=%d", method, endpoint, count)
						}
						return nil
					}
				}
				scope := newNovaUserQuotaScope(t, cloud, userID)
				var err error
				if operation == "Update" {
					var integer int64 = 9007199254740993
					large := int(integer)
					_, err = scope.Update(context.Background(), quotasets.UpdateOpts{Cores: &large}, quotasets.WithUpdateForce(false))
				} else {
					err = callNovaUserQuota(scope, operation, context.Background())
				}
				if err != nil || calls.Load() != 2 || callbacks.Load() != 1 || transportCalls.Load() != 2 || scope.ProjectID() != "project-fixed" || scope.UserID() != userID {
					t.Fatalf("calls=%d callbacks=%d transport=%d scope=%s/%s err=%v", calls.Load(), callbacks.Load(), transportCalls.Load(), scope.ProjectID(), scope.UserID(), err)
				}
			})
		}
	}
}

func TestNovaQuotaUserRedirectCannotChangeProjectUserOriginOrMethod(t *testing.T) {
	for _, tc := range []struct {
		name, location string
		status         int
	}{
		{"drop user", "/nova/os-quota-sets/project-fixed", 307},
		{"different user", "/nova/os-quota-sets/project-fixed?user_id=other-user", 307},
		{"duplicate user", "/nova/os-quota-sets/project-fixed?user_id=chosen-user&user_id=other-user", 307},
		{"different project", "/nova/os-quota-sets/other-project?user_id=chosen-user", 307},
		{"change method", "/nova/os-quota-sets/project-fixed?user_id=chosen-user", 303},
		{"external origin", "external", 307},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var initial, unsafe, policy, transport atomic.Int32
			other := testcloud.New(t)
			other.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { unsafe.Add(1); w.WriteHeader(202) })
			location := tc.location
			if location == "external" {
				location = other.Server.URL + "/nova/os-quota-sets/project-fixed?user_id=chosen-user"
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if initial.Add(1) == 1 {
					w.Header().Set("Location", location)
					w.WriteHeader(tc.status)
					return
				}
				unsafe.Add(1)
				w.WriteHeader(202)
			})
			originalTransport := cloud.Provider.HTTPClient.Transport
			cloud.Provider.HTTPClient.Transport = novaQuotaTransport(func(req *http.Request) (*http.Response, error) {
				transport.Add(1)
				return originalTransport.RoundTrip(req)
			})
			cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error { policy.Add(1); return nil }
			value, err := newNovaUserQuotaScope(t, cloud, "chosen-user").Reset(context.Background())
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || initial.Load() != 1 || unsafe.Load() != 0 || policy.Load() != 1 || transport.Load() != 1 {
				t.Fatalf("reset=%v initial=%d unsafe=%d policy=%d transport=%d err=%v", value, initial.Load(), unsafe.Load(), policy.Load(), transport.Load(), err)
			}
		})
	}
}

func TestNovaQuotaUserPreservesCallerRedirectRejectionAndExplicitAuthHeaders(t *testing.T) {
	t.Run("caller redirect rejection", func(t *testing.T) {
		cloud := testcloud.New(t)
		sentinel := errors.New("caller denied redirect")
		var policy atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", r.URL.String())
			w.WriteHeader(307)
		})
		cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error { policy.Add(1); return sentinel }
		if _, err := newNovaUserQuotaScope(t, cloud, "chosen-user").Reset(context.Background()); !errors.Is(err, sentinel) || policy.Load() != 1 {
			t.Fatalf("policy=%d err=%v", policy.Load(), err)
		}
	})
	for _, throwaway := range []bool{false, true} {
		t.Run(strconv.FormatBool(throwaway), func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Provider.SetToken("")
			cloud.Provider.SetThrowaway(throwaway)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Auth-Token") != "explicit-token" || !strings.Contains(r.Header.Get("User-Agent"), "quota-test-agent") {
					t.Errorf("headers=%s", r.Header)
				}
				testcloud.JSON(w, 200, projectQuotaLimits)
			})
			cloud.Provider.UserAgent.Prepend("quota-test-agent")
			client := cloud.Client("compute", "/nova")
			client.MoreHeaders = map[string]string{"X-Auth-Token": "explicit-token"}
			project, err := quotasets.New(client).InProject(context.Background(), resource.ID("project-fixed"))
			if err != nil {
				t.Fatal(err)
			}
			scope, err := project.InUser(context.Background(), resource.ID("chosen-user"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := scope.Get(context.Background()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNovaQuotaUserContextAndCurrentProjectKeepExplicitUser(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	})
	setQuotaProjectAuth(t, cloud.Provider, "project-fixed")
	project, err := quotasets.New(cloud.Client("compute", "/nova")).CurrentProject(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if scope, err := project.InUser(context.Background(), resource.ID("")); scope != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("scope=%v err=%v", scope, err)
	}
	scope, err := project.InUser(context.Background(), resource.ID("chosen-user"))
	if err != nil {
		t.Fatal(err)
	}
	setQuotaProjectAuth(t, cloud.Provider, "changed-project")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if value, err := project.InUser(ctx, resource.ID("chosen-user")); value != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("scope=%v err=%v", value, err)
	}
	for _, operation := range []string{"Get", "Detail", "Update", "Reset"} {
		if err := callNovaUserQuota(scope, operation, ctx); !errors.Is(err, context.Canceled) || calls.Load() != 0 {
			t.Fatalf("%s calls=%d err=%v", operation, calls.Load(), err)
		}
	}
	for _, operation := range []string{"Get", "Detail", "Update", "Reset"} {
		deadline, stop := context.WithTimeout(context.Background(), 30*time.Millisecond)
		if err := callNovaUserQuota(scope, operation, deadline); !errors.Is(err, context.DeadlineExceeded) {
			stop()
			t.Fatalf("%s err=%v", operation, err)
		}
		stop()
	}
	if calls.Load() != 4 || scope.ProjectID() != "project-fixed" || scope.UserID() != "chosen-user" {
		t.Fatal("context or auth refresh changed identity")
	}
}

func TestNovaQuotaUserCancellationDoesNotWaitForSharedReauth(t *testing.T) {
	cloud := testcloud.New(t)
	started, release := make(chan struct{}), make(chan struct{})
	reauthDone := make(chan error, 1)
	var requests atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		testcloud.JSON(w, 200, projectQuotaLimits)
	})
	cloud.Provider.ReauthFunc = func(ctx context.Context) error { close(started); <-release; return nil }
	go func() { reauthDone <- cloud.Provider.Reauthenticate(context.Background(), "test-token") }()
	<-started
	scope := newNovaUserQuotaScope(t, cloud, "chosen-user")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := scope.Get(ctx)
	close(release)
	if cause := <-reauthDone; cause != nil {
		t.Fatal(cause)
	}
	if !errors.Is(err, context.DeadlineExceeded) || requests.Load() != 0 {
		t.Fatalf("requests=%d err=%v", requests.Load(), err)
	}
}

func TestNovaQuotaUserMalformedResponsesKeepDecodeCauses(t *testing.T) {
	for _, operation := range []string{"Get", "Detail", "Update"} {
		for _, tc := range []struct {
			name, body string
			typed      bool
		}{
			{"missing", `{}`, false},
			{"null", `{"quota_set":null}`, false},
			{"array", `{"quota_set":[]}`, false},
			{"scalar", `{"quota_set":7}`, false},
			{"known type", `{"quota_set":{"cores":"bad"}}`, true},
		} {
			t.Run(operation+tc.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				var calls atomic.Int32
				body := tc.body
				if operation == "Detail" && tc.typed {
					body = `{"quota_set":{"cores":{"limit":"bad","in_use":0,"reserved":0}}}`
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.URL.Query().Get("user_id") != "chosen-user" {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, body)
				})
				scope := newNovaUserQuotaScope(t, cloud, "chosen-user")
				err := callNovaUserQuota(scope, operation, context.Background())
				var sdk *resource.OperationError
				if err == nil || !errors.As(err, &sdk) || sdk.Operation != operation || sdk.Unwrap() == nil || calls.Load() != 1 {
					t.Fatalf("calls=%d err=%v", calls.Load(), err)
				}
				if tc.typed {
					var decode *json.UnmarshalTypeError
					if !errors.As(err, &decode) {
						t.Fatalf("typed decode cause missing: %v", err)
					}
				}
				if scope.ProjectID() != "project-fixed" || scope.UserID() != "chosen-user" {
					t.Fatal("decode error changed scope identity")
				}
			})
		}
	}
}

func TestNovaQuotaUserRedirectKeepsSamePairAndCallerPolicy(t *testing.T) {
	for _, operation := range []string{"Get", "Detail", "Update", "Reset"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls, policy atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("user_id") != "chosen-user" || len(r.URL.Query()["user_id"]) != 1 {
					t.Error(r.URL)
				}
				if calls.Add(1) == 1 {
					w.Header().Set("Location", r.URL.String())
					w.WriteHeader(http.StatusTemporaryRedirect)
					return
				}
				if operation == "Reset" {
					w.WriteHeader(http.StatusAccepted)
				} else if operation == "Detail" {
					testcloud.JSON(w, 200, `{"quota_set":{"id":"wire-other","cores":{"limit":7,"in_use":1,"reserved":0}}}`)
				} else {
					testcloud.JSON(w, 200, projectQuotaLimits)
				}
			})
			cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
				policy.Add(1)
				return nil
			}
			scope := newNovaUserQuotaScope(t, cloud, "chosen-user")
			if err := callNovaUserQuota(scope, operation, context.Background()); err != nil || calls.Load() != 2 || policy.Load() != 1 {
				t.Fatalf("calls=%d policy=%d err=%v", calls.Load(), policy.Load(), err)
			}
		})
	}
	// Guard after the caller policy too. URL.Opaque can replace EscapedPath in
	// the actual RequestURI; Request.Host can select another virtual host.
	for _, tc := range []struct {
		name   string
		change func(*http.Request)
	}{
		{"policy drops user", func(next *http.Request) { next.URL.RawQuery = "" }},
		{"policy changes opaque target", func(next *http.Request) { next.URL.Opaque = "/nova/os-quota-sets/other-project" }},
		{"policy changes virtual host", func(next *http.Request) { next.Host = "other-cloud.example" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Location", r.URL.String())
				w.WriteHeader(http.StatusTemporaryRedirect)
			})
			cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
				tc.change(next)
				return nil
			}
			if value, err := newNovaUserQuotaScope(t, cloud, "chosen-user").Reset(context.Background()); value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
				t.Fatalf("reset=%v calls=%d err=%v", value, calls.Load(), err)
			}
		})
	}
}

func TestNovaQuotaUserRedirectRetainsCallerAndHTTPTimeoutContexts(t *testing.T) {
	for _, httpTimeout := range []bool{false, true} {
		t.Run(strconv.FormatBool(httpTimeout), func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			arrived, release := make(chan struct{}), make(chan struct{})
			t.Cleanup(func() { close(release) })
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					w.Header().Set("Location", r.URL.String())
					w.WriteHeader(http.StatusTemporaryRedirect)
					return
				case 2:
					close(arrived)
				}
				select {
				case <-r.Context().Done():
				case <-release:
				}
			})
			var redirectRequest *http.Request
			var redirectContext context.Context
			cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, via []*http.Request) error {
				redirectRequest, redirectContext = next, next.Context()
				*next = *next.WithContext(context.Background())
				return nil
			}
			ctx := context.Background()
			cancel := func() {}
			expected := context.DeadlineExceeded
			var sentContexts []context.Context
			if httpTimeout {
				// net/http installs Client.Timeout when sending each request,
				// after CheckRedirect. Observe that deadline at the transport
				// boundary without requiring the second handler to be scheduled.
				cloud.Provider.HTTPClient.Timeout = 500 * time.Millisecond
				transport := cloud.Provider.HTTPClient.Transport
				if transport == nil {
					transport = http.DefaultTransport
				}
				cloud.Provider.HTTPClient.Transport = novaQuotaTransport(func(req *http.Request) (*http.Response, error) {
					sentContexts = append(sentContexts, req.Context())
					return transport.RoundTrip(req)
				})
			} else {
				// A long deadline also checks deadline ownership. Cancellation
				// is triggered by redirect arrival, rather than elapsed time.
				ctx, cancel = context.WithTimeout(ctx, time.Hour)
				expected = context.Canceled
			}
			defer cancel()
			scope := newNovaUserQuotaScope(t, cloud, "chosen-user")
			type result struct {
				value *quotasets.ResetResponse
				err   error
			}
			done := make(chan result, 1)
			go func() {
				value, err := scope.Reset(ctx)
				done <- result{value, err}
			}()
			if !httpTimeout {
				select {
				case <-arrived:
					cancel()
				case got := <-done:
					t.Fatalf("reset finished before redirect arrival: reset=%v calls=%d err=%v", got.value, calls.Load(), got.err)
				case <-time.After(5 * time.Second):
					t.Fatal("redirect request did not arrive")
				}
			}
			var got result
			select {
			case got = <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("redirect lost caller cancellation or HTTP timeout")
			}
			if got.value != nil || !errors.Is(got.err, expected) {
				t.Fatalf("reset=%v calls=%d err=%v", got.value, calls.Load(), got.err)
			}
			if redirectRequest == nil || redirectContext == nil {
				t.Fatal("redirect policy was not entered")
			}
			if httpTimeout {
				if len(sentContexts) == 0 {
					t.Fatal("request did not reach the source transport")
				}
				deadline, present := sentContexts[0].Deadline()
				if !present {
					t.Fatal("HTTP client timeout was absent at the transport boundary")
				}
				for _, sent := range sentContexts[1:] {
					other, present := sent.Deadline()
					if !present || !other.Equal(deadline) {
						t.Fatal("redirect changed the HTTP client timeout deadline")
					}
				}
			} else {
				before, beforeOK := redirectContext.Deadline()
				after, afterOK := redirectRequest.Context().Deadline()
				if !beforeOK || !afterOK || !before.Equal(after) || redirectRequest.Context().Done() != redirectContext.Done() {
					t.Fatal("redirect policy replaced the original deadline or cancellation context")
				}
			}
		})
	}
}
