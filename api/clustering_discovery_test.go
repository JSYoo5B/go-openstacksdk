package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/buildinfo"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/policytypes"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestClusteringDiscoveryBuildInfoSingletonAndRawEvidence(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /tenant/senlin/v1/build-info", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "test-token" || r.Header.Get("OpenStack-API-Version") != "clustering 1.5" {
			t.Error(r.URL, r.Header)
		}
		w.Header().Set("X-Openstack-Request-Id", "build-request")
		testcloud.JSON(w, 200, `{"build_info":{"api":{"revision":"rev-api","vendor":9007199254740993},"engine":{"revision":"rev-engine"},"future":null}}`)
	})
	client := cloud.Client("clustering", "/catalog")
	client.ResourceBase = gophercloud.NormalizeURL(cloud.Server.URL + "/tenant/senlin/v1")
	client.Microversion = "1.5"
	value, err := buildinfo.New(client).Get(context.Background())
	if err != nil || value.API == nil || value.API.Revision == nil || *value.API.Revision != "rev-api" || value.Engine == nil || value.Engine.Revision == nil || *value.Engine.Revision != "rev-engine" {
		t.Fatal(value, err)
	}
	if string(value.API.Body["vendor"]) != "9007199254740993" || string(value.Body["future"]) != "null" || value.Header.Get("X-Openstack-Request-Id") != "build-request" || value.StatusCode != 200 {
		t.Fatal(value)
	}
	if _, exists := value.Body["created_at"]; exists {
		t.Fatal("invented an omitted field")
	}
}

func TestClusteringDiscoveryTypeNamePathsAndIndependentVersion(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /senlin/v1/profile-types/os.nova.server-1.0", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "profile-detail")
		testcloud.JSON(w, 200, `{"profile_type":{"name":"returned.profile-2.0","schema":{"flavor":{"default":9007199254740993}},"support_status":{"1.0":[{"status":"SUPPORTED","since":"2016.04"}]},"future":false}}`)
	})
	cloud.Mux.HandleFunc("GET /senlin/v1/policy-types/senlin.policy.scaling-1.0", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"policy_type":{"name":"senlin.policy.scaling-1.0","schema":{"adjustment":{"default":0}},"support_status":null}}`)
	})
	client := cloud.Client("clustering", "/senlin/v1")
	profile, err := profiletypes.New(client).Get(context.Background(), "os.nova.server-1.0")
	if err != nil || profile.Name != "returned.profile-2.0" || profile.Version != nil || string(profile.Schema["flavor"]) != `{"default":9007199254740993}` || profile.Header.Get("X-Request-Id") != "profile-detail" {
		t.Fatal(profile, err)
	}
	policy, err := policytypes.New(client).Resources.Get(context.Background(), "senlin.policy.scaling-1.0")
	if err != nil || policy.Name != "senlin.policy.scaling-1.0" || string(policy.Schema["adjustment"]) != `{"default":0}` || string(policy.SupportStatus) != "null" {
		t.Fatal(policy, err)
	}
}

func TestClusteringDiscoveryCatalogLinksPreserveExtensionsAndNoMarkerFallback(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/profile-types", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "1" || r.URL.Query().Get("vendor") != "original" {
			t.Error(r.URL)
		}
		if calls.Add(1) == 1 {
			w.Header().Set("X-Request-Id", "list-first")
			testcloud.JSON(w, 200, `{"profile_types":[{"name":"os.nova.server","version":"1.0","schema":null,"support_status":{"1.0":[]}}],"profile_types_links":[{"rel":"next","href":"?marker=next"}]}`)
			return
		}
		if r.URL.Query().Get("marker") != "next" {
			t.Error(r.URL)
		}
		w.Header().Set("X-Request-Id", "list-second")
		testcloud.JSON(w, 200, `{"profile_types":[{"name":"os.heat.stack","version":"1.0"}]}`)
	})
	api := profiletypes.New(cloud.Client("clustering", "/senlin/v1"))
	options := []profiletypes.ListOption{profiletypes.WithListOptions(profiletypes.ListOpts{Limit: 1}), profiletypes.WithListQuery("vendor", "original")}
	iterator := api.List(context.Background(), options...)
	options[1] = profiletypes.WithListQuery("vendor", "changed")
	if calls.Load() != 0 {
		t.Fatal("list construction performed HTTP")
	}
	for range 2 {
		var values []*profiletypes.ProfileType
		for value, err := range iterator {
			if err != nil {
				t.Fatal(err)
			}
			values = append(values, value)
		}
		if len(values) != 2 || values[0].Name != "os.nova.server" || values[0].Version == nil || *values[0].Version != "1.0" || values[0].Header.Get("X-Request-Id") != "list-first" || values[1].Header.Get("X-Request-Id") != "list-second" {
			t.Fatal(values)
		}
		values[0].Header.Set("X-Request-Id", "caller-change")
		if values[1].Header.Get("X-Request-Id") != "list-second" {
			t.Fatal("result headers share mutable state")
		}
		calls.Store(0)
	}
}

func TestClusteringDiscoveryHTTPLinksBreakAndTargetGuard(t *testing.T) {
	for _, mode := range []string{"follow", "break", "foreign", "filter-change"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("GET /senlin/v1/policy-types", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					link := "?marker=page2"
					if mode == "foreign" {
						link = "https://foreign.invalid/senlin/v1/policy-types?marker=page2"
					} else if mode == "filter-change" {
						link = "?marker=page2&vendor=changed"
					}
					w.Header().Set("Link", "<"+link+">; rel=\"next\"")
					testcloud.JSON(w, 200, `{"policy_types":[{"name":"first"}]}`)
					return
				}
				if r.URL.Query().Get("vendor") != "original" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"policy_types":[{"name":"second"}]}`)
			})
			api := policytypes.New(cloud.Client("clustering", "/senlin/v1"))
			if mode == "break" {
				for _, err := range api.List(context.Background(), policytypes.WithListQuery("vendor", "original")) {
					if err != nil {
						t.Fatal(err)
					}
					break
				}
				if calls.Load() != 1 {
					t.Fatal(calls.Load())
				}
				return
			}
			values, err := api.All(context.Background(), policytypes.WithListQuery("vendor", "original"))
			if mode == "follow" {
				if err != nil || len(values) != 2 || calls.Load() != 2 {
					t.Fatal(values, err, calls.Load())
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
				t.Fatal(values, err, calls.Load())
			}
		})
	}
}

func TestClusteringDiscoveryOperationsVersionSchemaAndRawEnvelope(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/profile-types/os.nova.server-1.0/ops", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("OpenStack-API-Version") != "clustering 1.4" {
			t.Error(r.Header)
		}
		w.Header().Set("X-Request-Id", "operations-request")
		testcloud.JSON(w, 200, `{"operations":{"reboot":{"parameters":{"timeout":{"default":9007199254740993}},"description":"reboot"}},"vendor":null}`)
	})
	client := cloud.Client("clustering", "/senlin/v1")
	api := profiletypes.New(client)
	for _, version := range []string{"", "1.0", "1.3", "latest", "2.4", "1.04"} {
		client.Microversion = version
		if _, err := api.Operations(context.Background(), "os.nova.server-1.0"); err == nil {
			t.Fatal("accepted unsupported version", version)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("version validation performed HTTP")
	}
	client.Microversion = "1.4"
	value, err := api.Operations(context.Background(), "os.nova.server-1.0")
	if err != nil || string(value.Operations["reboot"]) != `{"parameters":{"timeout":{"default":9007199254740993}},"description":"reboot"}` || value.StatusCode != 200 || value.Header.Get("X-Request-Id") != "operations-request" || string(value.Body["vendor"]) != "null" || calls.Load() != 1 || client.Microversion != "1.4" {
		t.Fatal(value, err, calls.Load())
	}
}

func TestClusteringDiscoveryHTTPAndDecodeErrorsRetainEvidence(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /senlin/v1/profile-types/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/senlin/v1/profile-types/")
		w.Header().Set("X-Request-Id", "error-request")
		if name == "denied" {
			testcloud.JSON(w, 403, `{"error":{"message":"permission denied"}}`)
		} else if name == "missing" {
			testcloud.JSON(w, 404, `{"error":{"message":"missing"}}`)
		} else {
			testcloud.JSON(w, 200, `{"profile_type":[]}`)
		}
	})
	api := profiletypes.New(cloud.Client("clustering", "/senlin/v1"))
	_, err := api.Get(context.Background(), "denied")
	var statusError gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &statusError) || statusError.Actual != 403 || statusError.ResponseHeader.Get("X-Request-Id") != "error-request" || !strings.Contains(string(statusError.Body), "permission denied") {
		t.Fatal(err)
	}
	if _, err := api.Get(context.Background(), "missing"); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	_, err = api.Get(context.Background(), "bad")
	var responseError *resource.ResponseError
	if !errors.As(err, &responseError) || responseError.StatusCode != 200 || responseError.Header.Get("X-Request-Id") != "error-request" || string(responseError.Body) != `{"profile_type":[]}` {
		t.Fatal(err)
	}
}

func TestClusteringDiscoveryInvalidInputsCancelAndRedirectBeforeSecondRequest(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /senlin/v1/profile-types/redirect", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Redirect(w, r, "/senlin/v1/policy-types", http.StatusFound)
	})
	cloud.Mux.HandleFunc("GET /senlin/v1/policy-types", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(100)
		testcloud.JSON(w, 200, `{"policy_types":[]}`)
	})
	api := profiletypes.New(cloud.Client("clustering", "/senlin/v1"))
	for _, name := range []string{"", "..", "type/other", "type?query", "type%2fother"} {
		if _, err := api.Get(context.Background(), name); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(name, err)
		}
	}
	for _, options := range [][]profiletypes.ListOption{
		{nil},
		{profiletypes.WithListOptions(profiletypes.ListOpts{Limit: -1})},
		{profiletypes.WithListQuery("limit", "1")},
	} {
		if _, err := api.All(context.Background(), options...); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(ctx, "valid"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := api.All(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	var nilBuild *buildinfo.API
	if _, err := nilBuild.Get(context.Background()); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("validation performed HTTP", calls.Load())
	}
	if _, err := api.Get(context.Background(), "redirect"); !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
		t.Fatal(err, calls.Load())
	}
}

func TestClusteringDiscoveryOperationsMalformedBodyEvidence(t *testing.T) {
	for _, body := range []string{`{}`, `{"operations":null}`, `{"operations":[]}`} {
		t.Run(fmt.Sprintf("%x", body), func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("GET /senlin/v1/profile-types/type/ops", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, body)
			})
			client := cloud.Client("clustering", "/senlin/v1")
			client.Microversion = "1.4"
			_, err := profiletypes.New(client).Operations(context.Background(), "type")
			var responseError *resource.ResponseError
			if !errors.As(err, &responseError) || string(responseError.Body) != body || responseError.StatusCode != 200 {
				t.Fatal(err)
			}
		})
	}
}
