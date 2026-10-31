package identity_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonCloudGetUserFiltersNoneVersusAllFalseyPresence(t *testing.T) {
	for _, filter := range []string{"omitted", "null", "{}", "[]", "false", "0", `""`} {
		t.Run(filter, func(t *testing.T) {
			calls := []string{}
			cloud, _ := userCloudFixture(t, func(q *http.Request) *http.Response {
				calls = append(calls, q.URL.RequestURI())
				if q.URL.Path == "/keystone/v3/users" {
					if q.URL.RawQuery != "name=reader" {
						t.Fatal(q.URL)
					}
					return userCloudWire(203, `{"users":[{"id":"search","name":"reader"}]}`)
				}
				if q.URL.Path != "/keystone/v3/users/reader" || q.URL.RawQuery != "" {
					t.Fatal(q.URL)
				}
				return userCloudWire(200, `{"user":{"id":"lookup","name":"reader"}}`)
			})
			opts := []identity.UserGetOption{}
			if filter != "omitted" {
				opts = append(opts, identity.WithUserGetFilters(json.RawMessage(filter)))
			}
			result, err := cloud.GetUser(context.Background(), "reader", opts...)
			deprecated := filter != "omitted" && filter != "null"
			want := 1
			if deprecated {
				want = 2
			}
			if err != nil || result == nil || result.User == nil || result.Lookup == nil || result.DeprecatedFilters != deprecated || len(calls) != want || string(result.User.Resource.Body["id"]) != `"lookup"` {
				t.Fatal(result, err, calls)
			}
			if deprecated && len(result.Inventory) != 1 {
				t.Fatal(result)
			}
			if !deprecated && len(result.Inventory) != 0 {
				t.Fatal(result)
			}
		})
	}
}
func TestPythonCloudGetUserPreservesSuccessfulSearchThenFindBugAndRealReceipts(t *testing.T) {
	calls := 0
	cloud, _ := userCloudFixture(t, func(q *http.Request) *http.Response {
		calls++
		if q.URL.Path == "/keystone/v3/users" {
			if q.URL.RawQuery != "domain_id=domain&name=reader" {
				t.Fatal(q.URL)
			}
			return userCloudWire(203, `{"users":[{"id":"selected","name":"reader","email":"wanted","description":{"n":0.10000000000000001}}]}`)
		}
		if q.URL.RawQuery != "domain_id=domain" || q.URL.Path != "/keystone/v3/users/reader" {
			t.Fatal(q.URL)
		}
		return userCloudWire(200, `{"user":{"id":"different","name":"reader","email":"NOT WANTED","description":"finder"}}`)
	})
	result, err := cloud.GetUser(context.Background(), "reader", identity.WithUserGetDomainID("domain"), identity.WithUserGetFilters(json.RawMessage(`{"email":"wanted","description":{"n":0.1}}`)))
	if err != nil || result == nil || result.User == nil || result.User.StatusCode != 200 || result.Lookup.StatusCode != 200 || len(result.Inventory) != 1 || result.Inventory[0].StatusCode != 203 || calls != 2 || string(result.User.Resource.Body["id"]) != `"different"` || string(result.User.Resource.Body["email"]) != `"NOT WANTED"` {
		t.Fatal(result, err, calls)
	}
	if !strings.Contains(string(result.SearchValue), `"id":"selected"`) {
		t.Fatal(result)
	}
	result.User.Resource.Body["id"][1] = 'X'
	result.User.Header.Set("X-Request-Id", "caller")
	if string(result.Lookup.Resource.Body["id"]) != `"different"` || string(result.Inventory[0].Resource.Body["id"]) != `"selected"` || result.Lookup.Header.Get("X-Request-Id") != "actual-user-cloud" {
		t.Fatal(result)
	}
}
func TestPythonCloudGetUserSearchStopsOnMissingDuplicateLocalErrorAndLateHTTP(t *testing.T) {
	for _, mode := range []string{"missing", "duplicate", "local", "late", "early mismatch"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			cloud, _ := userCloudFixture(t, func(q *http.Request) *http.Response {
				calls++
				if q.URL.Path != "/keystone/v3/users" {
					t.Fatal("find must not run", q.URL)
				}
				if mode == "missing" {
					return userCloudWire(200, `{"users":[]}`)
				}
				if mode == "early mismatch" {
					return userCloudWire(200, `{"users":[{"id":"a","name":"reader","description":"scalar"}]}`)
				}
				if calls == 1 {
					return userCloudWire(203, `{"users":[{"id":"a","name":"reader"}],"next":"?name=reader&marker=a"}`)
				}
				if mode == "late" {
					return userCloudWire(503, `{"error":"late"}`)
				}
				return userCloudWire(200, `{"users":[{"id":"a","name":"reader"}]}`)
			})
			filter := json.RawMessage(`{}`)
			if mode == "local" || mode == "late" {
				filter = json.RawMessage(`"["`)
			}
			if mode == "early mismatch" {
				filter = json.RawMessage(`{"name":"mismatch","description":{"key":1}}`)
			}
			result, err := cloud.GetUser(context.Background(), "reader", identity.WithUserGetFilters(filter))
			if result == nil || result.User != nil || result.Lookup != nil || !result.DeprecatedFilters {
				t.Fatal(result, err)
			}
			switch mode {
			case "missing", "early mismatch":
				if err != nil || string(result.Value) != "null" || calls != 1 {
					t.Fatal(result, err, calls)
				}
			case "duplicate":
				if !errors.Is(err, resource.ErrAmbiguous) || len(result.Inventory) != 2 || calls != 2 {
					t.Fatal(result, err, calls)
				}
			case "local":
				var proof *resource.ResponseError
				if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || len(result.Inventory) != 2 || calls != 2 {
					t.Fatal(result, err, calls)
				}
			case "late":
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &proof) || proof.Actual != 503 || len(result.Inventory) != 1 || calls != 2 {
					t.Fatal(result, err, calls)
				}
			}
		})
	}
}
func TestPythonCloudGetUserPolymorphicSearchUsesLenOnlyNeverIndexZero(t *testing.T) {
	for _, tc := range []struct{ expression, kind string }{
		{"`null`", "none"}, {"`false`", "none"}, {"`0`", "none"}, {"`[]`", "none"}, {"`{}`", "none"}, {"`\"\"`", "none"},
		{"`[null]`", "find"}, {"`[false]`", "find"}, {"`[0]`", "find"}, {"`{\"a\":1}`", "find"}, {"`\"界\"`", "find"}, {"[0:1]", "find"},
		{"`[1,2]`", "ambiguous"}, {"`{\"a\":1,\"b\":2}`", "ambiguous"}, {"`\"ab\"`", "ambiguous"}, {"`true`", "invalid"}, {"`1`", "invalid"}, {"`1.5`", "invalid"},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			calls := 0
			cloud, _ := userCloudFixture(t, func(q *http.Request) *http.Response {
				calls++
				if q.URL.Path == "/keystone/v3/users" {
					return userCloudWire(203, `{"users":[{"id":"one","name":"reader"}]}`)
				}
				return userCloudWire(200, `{"user":{"id":"finder","name":"reader"}}`)
			})
			result, err := cloud.GetUser(context.Background(), "reader", identity.WithUserGetExpression(tc.expression))
			if result == nil || !result.Expression || len(result.Inventory) != 1 || !result.DeprecatedFilters {
				t.Fatal(result, err)
			}
			switch tc.kind {
			case "none":
				if err != nil || result.User != nil || calls != 1 || string(result.Value) != "null" {
					t.Fatal(result, err, calls)
				}
			case "find":
				if err != nil || result.User == nil || calls != 2 || string(result.User.Resource.Body["id"]) != `"finder"` {
					t.Fatal(result, err, calls)
				}
			case "ambiguous":
				if !errors.Is(err, resource.ErrAmbiguous) || result.User != nil || calls != 1 {
					t.Fatal(result, err, calls)
				}
			case "invalid":
				var proof *resource.ResponseError
				if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || result.User != nil || calls != 1 {
					t.Fatal(result, err, calls)
				}
			}
		})
	}
}
func TestPythonCloudGetUserRawDomainTruthinessSearchVersusFixedFinder(t *testing.T) {
	for _, tc := range []struct {
		raw          string
		finder       url.Values
		searchDomain bool
	}{
		{"null", url.Values{}, false}, {`""`, url.Values{"domain_id": {""}}, false}, {"false", url.Values{"domain_id": {"False"}}, false}, {"0", url.Values{"domain_id": {"0"}}, false}, {`"domain"`, url.Values{"domain_id": {"domain"}}, true}, {`["domain",null,false]`, url.Values{"domain_id": {"domain", "False"}}, true},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			calls := 0
			cloud, _ := userCloudFixture(t, func(q *http.Request) *http.Response {
				calls++
				got := q.URL.Query()
				if q.URL.Path == "/keystone/v3/users" {
					want := url.Values{"name": {"reader"}}
					if tc.searchDomain {
						want["domain_id"] = tc.finder["domain_id"]
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatal(got, want)
					}
					return userCloudWire(200, `{"users":[{"id":"one","name":"reader"}]}`)
				}
				if !reflect.DeepEqual(got, tc.finder) {
					t.Fatal(got, tc.finder)
				}
				return userCloudWire(200, `{"user":{"id":"resolved"}}`)
			})
			result, err := cloud.GetUser(context.Background(), "reader", identity.WithUserGetFilters(json.RawMessage(`{}`)), identity.WithUserGetDomainIDJSON(json.RawMessage(tc.raw)))
			if err != nil || result.User == nil || calls != 2 || string(result.User.Resource.Body["domain_id"]) != tc.raw {
				t.Fatal(result, err, calls)
			}
		})
	}
	for _, name := range []string{"0123456789ABCDEF0123456789ABCDEF", "{0123456789abcdef0123456789abcdef}"} {
		calls := 0
		cloud, _ := userCloudFixture(t, func(q *http.Request) *http.Response {
			calls++
			if q.URL.Path == "/keystone/v3/users" {
				if q.URL.Query().Has("name") {
					t.Fatal(q.URL)
				}
				return userCloudWire(200, `{"users":[{"id":"`+name+`"}]}`)
			}
			return userCloudWire(200, `{"user":{"id":"resolved"}}`)
		})
		result, err := cloud.GetUser(context.Background(), name, identity.WithUserGetFilters(json.RawMessage(`{}`)))
		if err != nil || result.User == nil || calls != 2 {
			t.Fatal(result, err, calls)
		}
	}
}
func TestPythonCloudGetUserOptionsOnceOwnedFilterDomainLocationAndFrozenSource(t *testing.T) {
	raw := json.RawMessage(`{"email":"wanted"}`)
	domain := json.RawMessage(`"domain"`)
	headers := map[string]string{"X-Owned": "owned"}
	cloudName := "configured"
	location := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`)}}
	option := identity.WithUserGetOpts(identity.UserGetOpts{Filters: &raw, DomainID: domain, Headers: headers, Location: &location})
	raw[1] = 'X'
	domain[1] = 'X'
	headers["X-Owned"] = "mutated"
	cloudName = "mutated"
	calls, callbacks := 0, 0
	cloud, _ := userCloudFixture(t, func(q *http.Request) *http.Response {
		calls++
		if q.Header.Get("X-Selected") != "before" || q.Header.Get("X-Late") != "" || q.Header.Get("X-Owned") != "owned" || q.URL.Query().Get("domain_id") != "domain" {
			t.Fatal(q.Header, q.URL)
		}
		if q.URL.Path == "/keystone/v3/users" {
			return userCloudWire(203, `{"users":[{"id":"one","name":"reader","email":"wanted"}]}`)
		}
		return userCloudWire(200, `{"user":{"id":"lookup","project_id":"foreign","location":{"cloud":"forged"}}}`)
	})
	client := cloud.RawAPI().RawClient()
	client.MoreHeaders = map[string]string{"X-Selected": "before"}
	result, err := cloud.GetUser(context.Background(), "reader", option, func(*identity.UserGetOpts) error {
		callbacks++
		client.MoreHeaders["X-Selected"] = "after"
		client.MoreHeaders["X-Late"] = "after"
		return nil
	})
	if err != nil || result.User == nil || calls != 2 || callbacks != 1 || !strings.Contains(string(result.User.Resource.Body["location"]), `"cloud":"configured"`) || string(result.User.Wire.Body["location"]) != `{"cloud":"forged"}` {
		t.Fatal(result, err, calls, callbacks)
	}
}
func TestPythonCloudGetUserOptionCancellationOwnerAndPrivateGuardStopWork(t *testing.T) {
	calls := 0
	cloud, _ := userCloudFixture(t, func(*http.Request) *http.Response { calls++; return userCloudWire(200, `{"user":{"id":"one"}}`) })
	for _, mode := range []string{"nil", "cancel", "source", "private"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("user getter guard")
			seen := 0
			client := cloud.RawAPI().RawClient()
			old := client.Endpoint
			defer func() { client.Endpoint = old }()
			var first identity.UserGetOption = func(*identity.UserGetOpts) error {
				if mode == "source" {
					client.Endpoint = "https://changed.invalid/"
				} else if mode == "cancel" {
					cancel(cause)
				}
				return nil
			}
			if mode == "nil" {
				first = nil
			}
			if mode == "private" {
				ctx = rest.WithOperationGuard(ctx, func(context.Context) error { return cause })
			}
			result, err := cloud.GetUser(ctx, "one", first, func(*identity.UserGetOpts) error { seen++; client.Endpoint = old; return nil })
			var proof *resource.ResponseError
			if result != nil || err == nil || seen != 0 || calls != 0 || errors.As(err, &proof) {
				t.Fatal(result, err, seen, calls, proof)
			}
			if mode == "cancel" && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
				t.Fatal(err)
			}
			if mode == "private" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
		})
	}
	for _, empty := range []*identity.UserReadCloud{nil, identity.NewUserReadCloud(nil)} {
		result, err := empty.GetUser(context.Background(), "one")
		if result != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(result, err)
		}
	}
}
func TestPythonCloudGetUserSearchGateThenFinalFinderMissingFallbackAndLateFailure(t *testing.T) {
	for _, mode := range []string{"missing", "fallback", "duplicate", "late"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			cloud, _ := userCloudFixture(t, func(q *http.Request) *http.Response {
				calls++
				if calls == 1 {
					return userCloudWire(203, `{"users":[{"id":"gate","name":"reader"}]}`)
				}
				if strings.HasSuffix(q.URL.Path, "/reader") {
					return userCloudWire(403, `{"error":"find denied"}`)
				}
				if q.URL.Query().Get("name") != "reader" {
					t.Fatal(q.URL)
				}
				if mode == "missing" {
					return userCloudWire(200, `{"users":[]}`)
				}
				if calls == 3 {
					return userCloudWire(200, `{"users":[{"id":"final","name":"reader"}],"next":"?name=reader&marker=final"}`)
				}
				if mode == "late" {
					return userCloudWire(503, `{"error":"final late"}`)
				}
				if mode == "duplicate" {
					return userCloudWire(200, `{"users":[{"id":"final","name":"reader"}]}`)
				}
				return userCloudWire(200, `{"users":[]}`)
			})
			result, err := cloud.GetUser(context.Background(), "reader", identity.WithUserGetFilters(json.RawMessage(`{}`)))
			if result == nil || len(result.Inventory) != 1 || result.Inventory[0].StatusCode != 203 {
				t.Fatal(result, err)
			}
			switch mode {
			case "missing":
				if err != nil || result.User != nil || result.Lookup != nil || string(result.Value) != "null" || calls != 3 {
					t.Fatal(result, err, calls)
				}
			case "fallback":
				if err != nil || result.User == nil || string(result.User.Resource.Body["id"]) != `"final"` || calls != 4 {
					t.Fatal(result, err, calls)
				}
			case "duplicate":
				if !errors.Is(err, resource.ErrAmbiguous) || result.User != nil || result.Value != nil || calls != 4 {
					t.Fatal(result, err, calls)
				}
			case "late":
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &proof) || proof.Actual != 503 || result.User != nil || result.Value != nil || calls != 4 {
					t.Fatal(result, err, calls)
				}
			}
		})
	}
}
func TestPythonCloudGetUserPhysicalForbiddenNeverFindFallsThroughOrRetries(t *testing.T) {
	for _, deprecated := range []bool{false, true} {
		t.Run(fmt.Sprint(deprecated), func(t *testing.T) {
			calls, retries, reauth := 0, 0, 0
			cloud, fixture := userCloudFixture(t, func(q *http.Request) *http.Response {
				calls++
				if q.URL.Path == "/keystone/v3/users" {
					return userCloudWire(203, `{"users":[{"id":"one","name":"reader"}]}`)
				}
				r := userCloudWire(403, `{"error":"physical"}`)
				r.ContentLength = 128
				r.Body = &testcloud.Body{Data: `{"error":"physical"}`, ReadErr: io.ErrUnexpectedEOF}
				return r
			})
			fixture.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			fixture.Provider.ReauthFunc = func(context.Context) error { reauth++; return errors.New("unexpected physical-fault reauthentication") }
			opts := []identity.UserGetOption{}
			if deprecated {
				opts = append(opts, identity.WithUserGetFilters(json.RawMessage(`{}`)))
			}
			result, err := cloud.GetUser(context.Background(), "reader", opts...)
			var proof gophercloud.ErrUnexpectedResponseCode
			want := 1
			if deprecated {
				want = 2
			}
			if result == nil || result.User != nil || result.Value != nil || !errors.Is(err, io.ErrUnexpectedEOF) || !errors.As(err, &proof) || proof.Actual != 403 || proof.ResponseHeader.Get("X-Request-Id") != "actual-user-cloud" || calls != want || retries != 0 || reauth != 0 {
				t.Fatal(result, err, proof, calls, retries, reauth)
			}
		})
	}
}
