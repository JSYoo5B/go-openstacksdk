package identity_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func domainGetCloudFixture(t *testing.T, reply func(http.ResponseWriter, *http.Request)) (*identity.DomainReadCloud, *testcloud.Cloud) {
	t.Helper()
	f := testcloud.New(t)
	f.Mux.HandleFunc("GET /identity/v3/domains/", reply)
	f.Mux.HandleFunc("GET /identity/v3/domains", reply)
	return identity.NewDomainReadCloud(domains.New(f.Client("identity", "/identity/v3/"))), f
}

type domainIgnoredFilters struct{ calls *int }

func (v domainIgnoredFilters) MarshalJSON() ([]byte, error) {
	*v.calls++
	return nil, errors.New("ignored filter must not marshal")
}
func TestPythonCloudDomainGetIDPresenceWinsAndFiltersAreNeverConsumed(t *testing.T) {
	calls, encoded := 0, 0
	cloud, _ := domainGetCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) != 0 || r.URL.Path != "/identity/v3/domains/direct" || r.URL.RawQuery != "" {
			t.Fatal(r.URL, string(body), err)
		}
		w.Header().Set("X-Request-Id", "domain-direct")
		testcloud.JSON(w, 203, `{"domain":{"id":null,"name":"wire","options":{"keep":1}}}`)
	})
	for _, ignored := range []any{domainIgnoredFilters{&encoded}, json.RawMessage(`{invalid`), func() {}, false} {
		result, err := cloud.GetDomain(context.Background(), identity.WithDomainGetID("direct"), identity.WithDomainGetNameOrID("ignored"), identity.WithDomainGetFilters(ignored))
		if err != nil || result == nil || !result.DeprecatedFilters || result.Record == nil || result.Record.StatusCode != 203 || result.Record.Header.Get("X-Request-Id") != "domain-direct" {
			t.Fatal(result, err)
		}
		if string(result.Record.Resource.Body["id"]) != "null" || string(result.Record.Resource.Body["options"]) != `{"keep":1}` {
			t.Fatal(result)
		}
	}
	result, err := cloud.GetDomain(context.Background(), identity.WithDomainGetID("direct"), identity.WithDomainGetFilters(nil))
	if err != nil || result.DeprecatedFilters || encoded != 0 || calls != 5 {
		t.Fatal(result, err, encoded, calls)
	}
}
func TestPythonCloudDomainGetDirectMissingVersusClosedFinderMissing(t *testing.T) {
	for _, direct := range []bool{false, true} {
		calls := 0
		cloud, _ := domainGetCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			if calls == 1 {
				testcloud.JSON(w, 404, `{"error":"missing"}`)
			} else {
				if r.URL.Path != "/identity/v3/domains" || r.URL.Query().Get("name") != "alias" {
					t.Fatal(r.URL)
				}
				testcloud.JSON(w, 200, `{"domains":[]}`)
			}
		})
		var option identity.DomainGetOption = identity.WithDomainGetNameOrID("alias")
		if direct {
			option = identity.WithDomainGetID("direct")
		}
		result, err := cloud.GetDomain(context.Background(), option)
		if direct {
			var proof gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &proof) || proof.Actual != 404 || string(proof.Body) != `{"error":"missing"}` || calls != 1 || result == nil || result.Record != nil {
				t.Fatal(result, err, proof, calls)
			}
		} else if err != nil || result == nil || result.Record != nil || calls != 2 {
			t.Fatal(result, err, calls)
		}
	}
}
func TestPythonCloudDomainGetBothOmittedMatchesActualNoIDPreflight(t *testing.T) {
	calls := 0
	cloud, _ := domainGetCloudFixture(t, func(w http.ResponseWriter, r *http.Request) { calls++; testcloud.JSON(w, 200, `{"domain":{}}`) })
	result, err := cloud.GetDomain(context.Background(), identity.WithDomainGetFilters(json.RawMessage(`{invalid`)))
	var proof *resource.ResponseError
	if !errors.Is(err, resource.ErrInvalidOption) || result == nil || !result.DeprecatedFilters || result.Record != nil || errors.As(err, &proof) || calls != 0 {
		t.Fatal(result, err, proof, calls)
	}
	for _, option := range []identity.DomainGetOption{identity.WithDomainGetID(""), identity.WithDomainGetNameOrID("")} {
		result, err := cloud.GetDomain(context.Background(), option)
		if !errors.Is(err, resource.ErrInvalidOption) || result == nil || result.Record != nil || calls != 0 {
			t.Fatal(result, err, calls)
		}
	}
}
func TestPythonCloudDomainGetClosedFallbackDuplicatesAndLateHTTPProof(t *testing.T) {
	for _, late := range []bool{false, true} {
		calls := 0
		cloud, _ := domainGetCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.Header().Set("X-Request-Id", "cloud-domain-late")
			switch calls {
			case 1:
				testcloud.JSON(w, 403, `{"error":"fallback"}`)
			case 2:
				testcloud.JSON(w, 200, `{"domains":[{"id":"one","name":"alias"}],"next":"?marker=one&name=alias"}`)
			default:
				if late {
					testcloud.JSON(w, 503, `{"error":"actual late"}`)
				} else {
					testcloud.JSON(w, 200, `{"domains":[{"id":"one","name":"alias"}]}`)
				}
			}
		})
		result, err := cloud.GetDomain(context.Background(), identity.WithDomainGetNameOrID("alias"), identity.WithDomainGetFilters(json.RawMessage(`"["`)))
		if result == nil || result.Record != nil || !result.DeprecatedFilters || calls != 3 {
			t.Fatal(result, err, calls)
		}
		if late {
			var proof gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &proof) || proof.Actual != 503 || proof.Method != "GET" || !strings.Contains(proof.URL, "/domains?marker=one") || string(proof.Body) != `{"error":"actual late"}` || proof.ResponseHeader.Get("X-Request-Id") != "cloud-domain-late" {
				t.Fatal(err, proof)
			}
		} else if !errors.Is(err, resource.ErrAmbiguous) {
			t.Fatal(err)
		}
	}
}
func TestPythonCloudDomainGetOwnOptionLocationHeadersAndSourceSnapshot(t *testing.T) {
	cloudName, project := "configured", "current"
	location := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &project}}
	id := "direct"
	headers := map[string]string{"X-Owned": "owned"}
	opt := identity.WithDomainGetOpts(identity.DomainGetOpts{DomainID: &id, Headers: headers, Location: &location})
	cloudName = "changed"
	project = "changed"
	id = "changed"
	headers["X-Owned"] = "changed"
	cloud, _ := domainGetCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/identity/v3/domains/direct" || r.Header.Get("X-Owned") != "owned" || r.Header.Get("X-Snapshot") != "before" || r.Header.Get("X-Late") != "" {
			t.Fatal(r.URL, r.Header)
		}
		testcloud.JSON(w, 203, `{"domain":{"id":"wire","project_id":"foreign","location":{"cloud":"forged"}}}`)
	})
	client := cloud.RawAPI().RawClient()
	client.MoreHeaders = map[string]string{"X-Snapshot": "before"}
	result, err := cloud.GetDomain(context.Background(), opt, identity.DomainGetOption(func(*identity.DomainGetOpts) error {
		client.MoreHeaders["X-Snapshot"] = "after"
		client.MoreHeaders["X-Late"] = "after"
		return nil
	}))
	if err != nil || result == nil || result.Record == nil {
		t.Fatal(result, err)
	}
	var value map[string]json.RawMessage
	_ = json.Unmarshal(result.Record.Resource.Body["location"], &value)
	var scope map[string]json.RawMessage
	_ = json.Unmarshal(value["project"], &scope)
	if string(value["cloud"]) != `"configured"` || string(scope["id"]) != `"scope"` || string(scope["name"]) != `"current"` || string(result.Record.Wire.Body["location"]) != `{"cloud":"forged"}` {
		t.Fatal(result)
	}
}
func TestPythonCloudDomainGetOptionsStickySourceCancelAndNil(t *testing.T) {
	calls := 0
	cloud, _ := domainGetCloudFixture(t, func(w http.ResponseWriter, r *http.Request) { calls++; testcloud.JSON(w, 200, `{"domain":{}}`) })
	for _, mode := range []string{"source", "cancel", "nil"} {
		t.Run(mode, func(t *testing.T) {
			client := cloud.RawAPI().RawClient()
			endpoint := client.Endpoint
			defer func() { client.Endpoint = endpoint }()
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("cloud domain custom cancel")
			seen := 0
			first := identity.DomainGetOption(func(*identity.DomainGetOpts) error {
				if mode == "source" {
					client.Endpoint += "changed"
				} else {
					cancel(cause)
				}
				return nil
			})
			if mode == "nil" {
				first = nil
			}
			result, err := cloud.GetDomain(ctx, identity.WithDomainGetID("direct"), first, identity.DomainGetOption(func(*identity.DomainGetOpts) error { seen++; client.Endpoint = endpoint; return nil }))
			var proof *resource.ResponseError
			if result != nil || err == nil || seen != 0 || calls != 0 || errors.As(err, &proof) {
				t.Fatal(result, err, seen, calls, proof)
			}
			if mode == "cancel" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
		})
	}
}
