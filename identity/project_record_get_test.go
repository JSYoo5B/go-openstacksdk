package identity_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/projects"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
)

func projectGetFixture(t *testing.T, reply func(http.ResponseWriter, *http.Request)) (*identity.ProjectReadCloud, *testcloud.Cloud, *atomic.Int32) {
	t.Helper()
	f := testcloud.New(t)
	calls := &atomic.Int32{}
	f.Mux.HandleFunc("/identity/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want %s", r.Method, http.MethodGet)
		}
		w.Header().Set("X-Request-Id", "project-get")
		reply(w, r)
	})
	return identity.NewProjectReadCloud(projects.New(f.Client("identity", "/identity/v3/"))), f, calls
}
func TestCloudGetProjectNoneVersusPresentFalseySearch(t *testing.T) {
	for _, raw := range []string{"omitted", "null", "false", "0", "1e-9999", `""`, "[]", "{}"} {
		t.Run(raw, func(t *testing.T) {
			member, list := 0, 0
			cloud, _, calls := projectGetFixture(t, func(w http.ResponseWriter, r *http.Request) {
				if got := r.URL.Query().Get("domain_id"); got != "domain" {
					t.Errorf("r.URL.Query().Get(\"domain_id\") = %v, want %v", got, "domain")
					return
				}
				data, err := io.ReadAll(r.Body)
				if err != nil || len(data) != 0 || r.ContentLength != 0 {
					t.Error(err, string(data), r.ContentLength)
				}
				if r.URL.Path == "/identity/v3/projects/one" {
					member++
					testcloud.JSON(w, 203, `{"project":{"id":"one","name":"member"}}`)
				} else {
					list++
					if r.URL.Path != "/identity/v3/projects" {
						t.Errorf("r.URL.Path = %v, want %v", r.URL.Path, "/identity/v3/projects")
						return
					}
					if got := r.URL.Query().Get("name"); got != "" {
						t.Errorf("r.URL.Query().Get(\"name\") = %v, want %v", got, "")
						return
					}
					testcloud.JSON(w, 200, `{"projects":[{"id":"one","name":"listed"}]}`)
				}
			})
			options := []identity.ProjectGetOption{identity.WithProjectGetDomainID("domain")}
			if raw != "omitted" {
				options = append(options, identity.WithProjectGetFilters(json.RawMessage(raw)))
			}
			result, err := cloud.GetProject(context.Background(), "one", options...)
			deprecated := raw != "omitted" && raw != "null"
			if err != nil || result == nil || result.Project == nil || calls.Load() != 1 || result.DeprecatedFilters != deprecated {
				t.Fatal(result, err, calls.Load())
			}
			want := `"member"`
			if deprecated {
				want = `"listed"`
				if member != 0 || list != 1 {
					t.Fatal(member, list)
				}
			} else if member != 1 || list != 0 {
				t.Fatal(member, list)
			}
			if string(result.Project.Resource.Body["name"]) != want {
				t.Fatal(result)
			}
			result.Project.Resource.Body["name"][1] = 'X'
			if string(result.Inventory[0].Resource.Body["name"]) != want {
				t.Fatal("owned Inventory", result)
			}
		})
	}
}
func TestCloudGetProjectLateNativeFailureBeforeLocalFilter(t *testing.T) {
	cloud, _, calls := projectGetFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("marker") == "" {
			testcloud.JSON(w, 203, `{"projects":[{"id":"one","description":"first"}],"next":"?marker=one"}`)
		} else {
			testcloud.JSON(w, 503, `{"error":"late"}`)
		}
	})
	result, err := cloud.GetProject(context.Background(), "one", identity.WithProjectGetFilters(json.RawMessage(`{}`)))
	var native gophercloud.ErrUnexpectedResponseCode
	if result == nil || !result.DeprecatedFilters || len(result.Inventory) != 1 || result.Value != nil || result.Project != nil || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"error":"late"}` || native.ResponseHeader.Get("X-Request-Id") != "project-get" || calls.Load() != 2 {
		t.Fatal(result, err, native, calls.Load())
	}
}
func TestCloudGetProjectStickyOptionsCancelAndOuterGuard(t *testing.T) {
	for _, lane := range []string{"nil", "source", "cancel", "outer", "role", "headers"} {
		t.Run(lane, func(t *testing.T) {
			cloud, _, calls := projectGetFixture(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, 200, `{"project":{"id":"one"}}`) })
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("caller witness")
			seen := 0
			var options []identity.ProjectGetOption
			switch lane {
			case "nil":
				options = append(options, nil)
			case "source":
				options = append(options, func(*identity.ProjectGetOpts) error {
					cloud.RawAPI().RawClient().ResourceBase += "changed/"
					return nil
				})
			case "cancel":
				cancel(cause)
			case "outer":
				ctx = rest.WithOperationGuard(ctx, func(context.Context) error { return cause })
			case "role":
				cloud.RawAPI().RawClient().Type = "network"
			case "headers":
				options = append(options, identity.WithProjectGetHeaders(map[string]string{"X-Auth-Token": "foreign"}))
			}
			if lane != "headers" {
				options = append(options, func(*identity.ProjectGetOpts) error { seen++; return nil })
			}
			result, err := cloud.GetProject(ctx, "one", options...)
			if err == nil || result != nil && result.Project != nil || calls.Load() != 0 || seen != 0 {
				t.Fatal(result, err, calls.Load(), seen)
			}
			if (lane == "cancel" || lane == "outer") && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if lane == "role" && !errors.Is(err, resource.ErrUnsupported) {
				t.Fatal(err)
			}
		})
	}
}
func TestCloudGetProjectPhysicalCloseSourceCancelKeepActualNativeEvidence(t *testing.T) {
	for _, status := range []int{203, 401, 404, 429, 503} {
		for _, lane := range []string{"close", "source", "cancel"} {
			t.Run(fmt.Sprintf("%d/%s", status, lane), func(t *testing.T) {
				body := `{"project":{"id":"one"}}`
				cloud, f, calls := projectGetFixture(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, status, body) })
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New(lane + " witness")
				closes := &atomic.Int32{}
				base := f.Provider.HTTPClient.Transport
				f.SetTransport(testcloud.LinkRequest(func(r *http.Request) (*http.Response, error) {
					response, err := base.RoundTrip(r)
					if response != nil {
						b := &endpointCloudFaultBody{ReadCloser: response.Body, closes: closes}
						switch lane {
						case "close":
							b.cause = cause
						case "source":
							b.onClose = func() { cloud.RawAPI().RawClient().ResourceBase += "changed/" }
						case "cancel":
							b.onClose = func() { cancel(cause) }
						}
						response.Body = b
					}
					return response, err
				}))
				hooks := 0
				f.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks++
					return nil
				}
				f.Provider.ReauthFunc = func(context.Context) error { hooks++; return nil }
				f.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error { hooks++; return nil }
				result, err := cloud.GetProject(ctx, "one")
				if err == nil || result != nil && (result.Project != nil || result.Value != nil) || calls.Load() != 1 || closes.Load() != 1 || hooks != 0 {
					t.Fatal(result, err, calls.Load(), closes.Load(), hooks)
				}
				if lane != "source" && !errors.Is(err, cause) {
					t.Fatal(err)
				}
				if lane == "source" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				if status < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "project-get" {
						t.Fatal(err, proof)
					}
					return
				}
				receipt := err
				if status == 401 {
					var auth *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &auth) || auth.ErrOriginal == nil || auth.ErrReauth == nil {
						t.Fatal(err)
					}
					receipt = auth.ErrOriginal
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(receipt, &native) || native.Actual != status || native.Method != http.MethodGet || native.URL != f.Server.URL+"/identity/v3/projects/one" || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "project-get" {
					t.Fatal(err, native)
				}
			})
		}
	}
}

func TestCloudGetProjectTruthyFilterKwargsExpansionFailsBeforeHTTP(t *testing.T) {
	cloud, _, calls := projectGetFixture(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, 200, `{"projects":[]}`) })
	for _, raw := range []string{`"length(@)"`, `true`, `1`, `[1]`} {
		result, err := cloud.GetProject(context.Background(), "one", identity.WithProjectGetFilters(json.RawMessage(raw)))
		if result == nil || !result.DeprecatedFilters || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal(raw, result, err, calls.Load())
		}
	}
}
func TestCloudGetProjectProxyLiteralFalseyRowsFollowDictFilterBeforeShape(t *testing.T) {
	cloud, _, calls := projectGetFixture(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, 200, `{"projects":[]}`) })
	result, err := cloud.GetProject(context.Background(), "", identity.WithProjectGetFilters(json.RawMessage(`{"jmespath_filters":"\u0060[false,0,null,\"\"]\u0060"}`)))
	if err != nil || result == nil || !result.DeprecatedFilters || result.Value != nil || result.Project != nil || len(result.Inventory) != 0 || calls.Load() != 0 {
		t.Fatal(result, err, calls.Load())
	}
	result, err = cloud.GetProject(context.Background(), "one", identity.WithProjectGetFilters(json.RawMessage(`{"jmespath_filters":"\u0060[false]\u0060"}`)))
	if !errors.Is(err, resource.ErrInvalidOption) || result == nil || calls.Load() != 0 {
		t.Fatal(result, err, calls.Load())
	}
}
func TestCloudGetProjectRawDomainTruthyOverrideKeepsOriginalLocalFilters(t *testing.T) {
	for _, domain := range []string{"null", "false", "0", `""`, `"bound"`, `1`} {
		t.Run(domain, func(t *testing.T) {
			cloud, _, calls := projectGetFixture(t, func(w http.ResponseWriter, r *http.Request) {
				want := "filter-domain"
				if domain == `"bound"` {
					want = "bound"
				}
				if domain == "1" {
					want = "1"
				}
				if got := r.URL.Query().Get("domain_id"); got != want {
					t.Errorf("r.URL.Query().Get(\"domain_id\") = %v, want %v", got, want)
					return
				}
				testcloud.JSON(w, 203, `{"projects":[{"id":"one","name":"ops-one","domain_id":"filter-domain","options":{"flag":1}}]}`)
			})
			result, err := cloud.GetProject(context.Background(), "ops-*", identity.WithProjectGetRawDomainID(json.RawMessage(domain)), identity.WithProjectGetFilters(json.RawMessage(`{"domain_id":"filter-domain","options":{"flag":true}}`)))
			if err != nil || result == nil || result.Project == nil || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			if string(result.Project.Resource.Body["domain_id"]) != `"filter-domain"` {
				t.Fatal(result)
			}
		})
	}
}
