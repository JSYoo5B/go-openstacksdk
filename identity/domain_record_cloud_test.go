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

func domainCloudFixture(t *testing.T, reply func(http.ResponseWriter, *http.Request)) (*identity.DomainReadCloud, *testcloud.Cloud) {
	t.Helper()
	f := testcloud.New(t)
	f.Mux.HandleFunc("GET /identity/v3/domains", reply)
	return identity.NewDomainReadCloud(domains.New(f.Client("identity", "/identity/v3/"))), f
}
func TestPythonDomainCloudListKeywordQueryAliasAndLocalFilters(t *testing.T) {
	calls := 0
	cloud, _ := domainCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) != 0 || r.ContentLength != 0 || r.URL.RawQuery != "enabled=False&name=alpha" || r.Header.Get("Accept") != "application/json" {
			t.Fatal(r.URL, r.Header, string(body), err)
		}
		w.Header().Set("X-Request-Id", "cloud-domain")
		testcloud.JSON(w, 203, `{"domains":[{"id":"a","name":"alpha","description":"keep","options":{"n":0.10000000000000001}},{"id":"b","description":"skip"}]}`)
	})
	result, err := cloud.ListDomains(context.Background(), identity.WithDomainReadFilters(json.RawMessage(`{"name":"alpha","is_enabled":false,"enabled":true,"description":"keep","options":{"n":0.1},"unknown":true}`)))
	if err != nil || result == nil || len(result.Domains) != 1 || len(result.Inventory) != 1 || calls != 1 || result.JSON == nil || result.Domains[0].StatusCode != 203 || result.Domains[0].Header.Get("X-Request-Id") != "cloud-domain" {
		t.Fatal(result, err, calls)
	}
}
func TestPythonDomainCloudSearchIdentifierPresenceSelectsSourceBranch(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		calls := 0
		cloud, _ := domainCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
			calls++
			want := "name=alpha"
			if explicit {
				want = ""
			}
			if r.URL.RawQuery != want {
				t.Fatal(explicit, r.URL)
			}
			testcloud.JSON(w, 200, `{"domains":[{"id":"a","name":"alpha","description":"keep"},{"id":"b","name":"other","description":"keep"}]}`)
		})
		opts := []identity.DomainReadOption{identity.WithDomainReadFilters(json.RawMessage(`{"name":"alpha","description":"keep"}`))}
		if explicit {
			opts = append(opts, identity.WithDomainReadNameOrID(""))
		}
		result, err := cloud.SearchDomains(context.Background(), opts...)
		want := 2
		if explicit {
			want = 1
		}
		if err != nil || result == nil || len(result.Domains) != want || len(result.Inventory) != 2 || calls != 1 {
			t.Fatal(explicit, result, err, calls)
		}
	}
}
func TestPythonDomainCloudSearchAbsentKeywordMappingVersusPresentJMESPath(t *testing.T) {
	calls := 0
	cloud, _ := domainCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.RawQuery != "" {
			t.Fatal(r.URL)
		}
		testcloud.JSON(w, 200, `{"domains":[{"id":"a","name":"alpha"},{"id":"b","name":"other"}]}`)
	})
	for _, raw := range []string{`"length(@)"`, `false`, `0`, `""`, `[]`} {
		before := calls
		result, err := cloud.SearchDomains(context.Background(), identity.WithDomainReadFilters(json.RawMessage(raw)))
		var proof *resource.ResponseError
		if !errors.Is(err, resource.ErrInvalidOption) || result == nil || len(result.Inventory) != 0 || errors.As(err, &proof) || calls != before {
			t.Fatal(raw, result, err, proof, calls)
		}
	}
	for _, tc := range []struct{ name, filter, want string }{{"", "length(@)", "2"}, {"alpha", "[0].id", `"a"`}, {"missing", "length(@)", "0"}} {
		raw, _ := json.Marshal(tc.filter)
		result, err := cloud.SearchDomains(context.Background(), identity.WithDomainReadNameOrID(tc.name), identity.WithDomainReadFilters(raw))
		if err != nil || result == nil || !result.Expression || string(result.Value) != tc.want || result.Domains != nil || len(result.Inventory) != 2 {
			t.Fatal(tc, result, err)
		}
	}
	for _, explicit := range []bool{false, true} {
		for _, raw := range []string{`null`, `{}`} {
			opts := []identity.DomainReadOption{identity.WithDomainReadFilters(json.RawMessage(raw))}
			if explicit {
				opts = append(opts, identity.WithDomainReadNameOrID(""))
			}
			result, err := cloud.SearchDomains(context.Background(), opts...)
			if err != nil || result == nil || len(result.Domains) != 2 {
				t.Fatal(explicit, raw, result, err)
			}
		}
	}
}
func TestPythonDomainCloudListProxyPolymorphicResultIsUnread(t *testing.T) {
	calls := 0
	cloud, _ := domainCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		testcloud.JSON(w, 200, `{"domains":[{"id":"a"}]}`)
	})
	for _, tc := range []struct{ filter, want string }{{`{"jmespath_filters":"\u0060[1,false,null]\u0060"}`, `[1,false,null]`}, {`{"jmespath_filters":"\u0060\"ab\"\u0060"}`, `["a","b"]`}, {`{"jmespath_filters":"\u0060{\"z\":1,\"a\":2}\u0060"}`, `["a","z"]`}} {
		result, err := cloud.ListDomains(context.Background(), identity.WithDomainReadFilters(json.RawMessage(tc.filter)))
		if err != nil || result == nil || string(result.JSON) != tc.want || result.Domains != nil || len(result.Inventory) != 0 || calls != 0 {
			t.Fatal(tc, result, err, calls)
		}
	}
	result, err := cloud.ListDomains(context.Background(), identity.WithDomainReadFilters(json.RawMessage(`{"jmespath_filters":"to_array(@)"}`)))
	if err != nil || result == nil || result.JSON != nil || len(result.Value) != 1 || calls != 0 {
		t.Fatal(result, err, calls)
	}
	if _, ok := result.Value[0].(*domains.RecordListGenerator); !ok {
		t.Fatal(result)
	}
	resultSearch, err := cloud.SearchDomains(context.Background(), identity.WithDomainReadFilters(json.RawMessage(`{"jmespath_filters":"to_array(@)"}`)))
	if err != nil || resultSearch == nil || resultSearch.Value != nil || len(resultSearch.ValueObjects) != 1 || calls != 0 {
		t.Fatal(resultSearch, err, calls)
	}
	if _, ok := resultSearch.ValueObjects[0].(*domains.RecordListGenerator); !ok {
		t.Fatal(resultSearch)
	}
}
func TestPythonDomainCloudLocalLoadedSubsetGlobAndIndependentRecord(t *testing.T) {
	cloud, _ := domainCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Fatal(r.URL)
		}
		testcloud.JSON(w, 203, `{"domains":[{"id":"repeat","name":"ops-a","options":{"n":9007199254740993.0,"flag":true}},{"id":"repeat","name":"ops-b","options":{"n":9007199254740992,"flag":1}},{"id":"c","name":"other","options":{}}]}`)
	})
	result, err := cloud.SearchDomains(context.Background(), identity.WithDomainReadNameOrID("ops-*"), identity.WithDomainReadFilters(json.RawMessage(`{"options":{"n":9007199254740992,"flag":1}}`)))
	if err != nil || result == nil || len(result.Domains) != 2 || len(result.Inventory) != 3 || result.Expression {
		t.Fatal(result, err)
	}
	result.Domains[0].Resource.Body["id"][1] = 'X'
	if string(result.Inventory[0].Resource.Body["id"]) != `"repeat"` {
		t.Fatal(result)
	}
}
func TestPythonDomainCloudLateHTTPPrecedesLocalFilterError(t *testing.T) {
	calls := 0
	cloud, _ := domainCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("X-Request-Id", "domain-late")
		if calls == 1 {
			testcloud.JSON(w, 203, `{"domains":[{"id":"a"}],"next":"?marker=a"}`)
		} else {
			testcloud.JSON(w, 503, `{"error":"late domain"}`)
		}
	})
	result, err := cloud.SearchDomains(context.Background(), identity.WithDomainReadNameOrID(""), identity.WithDomainReadFilters(json.RawMessage(`"["`)))
	var proof gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &proof) || proof.Actual != 503 || proof.ResponseHeader.Get("X-Request-Id") != "domain-late" || string(proof.Body) != `{"error":"late domain"}` || proof.Method != "GET" || !strings.Contains(proof.URL, "/identity/v3/domains?marker=a") || result == nil || len(result.Inventory) != 1 || calls != 2 {
		t.Fatal(result, err, proof, calls)
	}
}
func TestPythonDomainCloudExplicitLocationOwnedBeforeLocalFiltering(t *testing.T) {
	name, project := "configured", "current"
	location := resource.CloudLocation{Cloud: &name, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &project}}
	raw := json.RawMessage(`{"location":{"cloud":"configured","project":{"id":"scope","name":"current"}}}`)
	opt := identity.WithDomainReadOpts(identity.DomainReadOpts{Location: &location, Filters: &raw})
	name = "changed"
	project = "changed"
	raw[0] = '['
	cloud, _ := domainCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"domains":[{"id":"a","project_id":"foreign","location":{"cloud":"wire"}}]}`)
	})
	result, err := cloud.SearchDomains(context.Background(), identity.WithDomainReadNameOrID(""), opt, identity.WithDomainReadNameOrID(""))
	if err != nil || len(result.Domains) != 1 || len(result.Inventory) != 1 || string(result.Domains[0].Wire.Body["location"]) != `{"cloud":"wire"}` {
		t.Fatal(result, err)
	}
}
func TestPythonDomainCloudOptionsStickySourceCancelAndHeaderSnapshot(t *testing.T) {
	calls := 0
	cloud, _ := domainCloudFixture(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("X-Snapshot") != "before" || r.Header.Get("X-Late") != "" {
			t.Fatal(r.Header)
		}
		testcloud.JSON(w, 200, `{"domains":[]}`)
	})
	client := cloud.RawAPI().RawClient()
	client.MoreHeaders = map[string]string{"X-Snapshot": "before"}
	_, err := cloud.ListDomains(context.Background(), identity.DomainReadOption(func(*identity.DomainReadOpts) error {
		client.MoreHeaders["X-Snapshot"] = "after"
		client.MoreHeaders["X-Late"] = "after"
		return nil
	}))
	if err != nil || calls != 1 {
		t.Fatal(err, calls)
	}
	for _, mode := range []string{"source", "cancel", "nil"} {
		t.Run(mode, func(t *testing.T) {
			before := calls
			seen := 0
			endpoint := client.Endpoint
			defer func() { client.Endpoint = endpoint }()
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("custom domain cancel")
			first := identity.DomainReadOption(func(*identity.DomainReadOpts) error {
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
			_, err := cloud.ListDomains(ctx, first, identity.DomainReadOption(func(*identity.DomainReadOpts) error { seen++; client.Endpoint = endpoint; return nil }))
			var proof *resource.ResponseError
			if err == nil || seen != 0 || calls != before || errors.As(err, &proof) {
				t.Fatal(err, seen, calls, proof)
			}
			if mode == "cancel" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
		})
	}
}
