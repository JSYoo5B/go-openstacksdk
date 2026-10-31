package roles_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func closedRoleFixture(t *testing.T, reply func(*http.Request) *http.Response) (*roles.API, *testcloud.Cloud) {
	t.Helper()
	fixture := testcloud.New(t)
	fixture.SetTransport(testcloud.LinkRequest(func(q *http.Request) (*http.Response, error) { return reply(q), nil }))
	client := fixture.Client("identity", "/unused/catalog/")
	client.ResourceBase = fixture.Server.URL + "/keystone/v3/"
	return roles.New(client), fixture
}
func closedRoleWire(status int, body string) *http.Response {
	r := roleRecordWire(status, body)
	r.Header.Set("X-Request-Id", "closed-role-proof")
	return r
}
func TestPythonRoleClosedLookupDomainSeedRawQueryAndDescriptorEvidence(t *testing.T) {
	for _, tc := range []struct {
		raw   string
		query url.Values
	}{
		{"null", url.Values{}}, {`""`, url.Values{"domain_id": {""}}}, {"false", url.Values{"domain_id": {"False"}}}, {"0", url.Values{"domain_id": {"0"}}}, {`["d",null,false]`, url.Values{"domain_id": {"d", "False"}}}, {`{"b":1,"a":2}`, url.Values{"domain_id": {"b", "a"}}},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			api, _ := closedRoleFixture(t, func(q *http.Request) *http.Response {
				if q.Method != "GET" || q.URL.Path != "/keystone/v3/roles/reader" || !reflect.DeepEqual(q.URL.Query(), tc.query) || q.Body != nil {
					t.Fatal(q.URL, q.Body)
				}
				return closedRoleWire(203, `{"role":{"id":"actual","name":"reader","links":{"self":"member"},"x_unknown":3}}`)
			})
			row, err := api.ResolveRecord(context.Background(), "reader", roles.WithClosedLookupDomainIDJSON(json.RawMessage(tc.raw)))
			if err != nil || row == nil || string(row.Resource.Body["domain_id"]) != tc.raw || string(row.Resource.Body["options"]) != "null" || row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "closed-role-proof" {
				t.Fatal(row, err)
			}
			if _, exists := row.Wire.Body["domain_id"]; exists {
				t.Fatal(row)
			}
			if _, exists := row.Resource.Body["x_unknown"]; exists {
				t.Fatal(row)
			}
			copy := row.Clone()
			copy.Resource.Body["name"][1] = 'X'
			copy.Wire.Body["x_unknown"][0] = '4'
			copy.Header.Set("X-Request-Id", "changed")
			copy.Envelope[0] = '['
			if string(row.Resource.Body["name"]) != `"reader"` || string(row.Wire.Body["x_unknown"]) != "3" || row.Header.Get("X-Request-Id") != "closed-role-proof" {
				t.Fatal(row)
			}
		})
	}
	api, _ := closedRoleFixture(t, func(*http.Request) *http.Response {
		return closedRoleWire(200, `{"role":{"id":"a","domain_id":false}}`)
	})
	row, err := api.ResolveRecord(context.Background(), "a", roles.WithClosedLookupDomainID("seed"))
	if err != nil || string(row.Resource.Body["domain_id"]) != "false" {
		t.Fatal(row, err)
	}
}
func TestPythonRoleClosedLookupCleanStatusFallbackAndLiteralNameHint(t *testing.T) {
	for _, status := range []int{199, 203, 304, 399, 400, 403, 404, 409, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls := 0
			api, _ := closedRoleFixture(t, func(q *http.Request) *http.Response {
				calls++
				if q.URL.Query().Get("domain_id") != "domain" {
					t.Fatal(q.URL)
				}
				if strings.HasSuffix(q.URL.Path, "/reader") {
					if q.URL.Query().Has("name") {
						t.Fatal(q.URL)
					}
					return closedRoleWire(status, `{"role":{"id":"direct","name":"reader"}}`)
				}
				if q.URL.Query().Get("name") != "reader" {
					t.Fatal(q.URL)
				}
				return closedRoleWire(200, `{"roles":[{"id":"other","name":"reader"}]}`)
			})
			row, err := api.ResolveRecord(context.Background(), "reader", roles.WithClosedLookupDomainID("domain"))
			switch {
			case status < 400:
				if err != nil || row == nil || row.StatusCode != status || calls != 1 || string(row.Resource.Body["id"]) != `"direct"` {
					t.Fatal(row, err, calls)
				}
			case status == 400 || status == 403 || status == 404:
				if err != nil || row == nil || calls != 2 || string(row.Resource.Body["id"]) != `"other"` {
					t.Fatal(row, err, calls)
				}
			default:
				var proof gophercloud.ErrUnexpectedResponseCode
				if row != nil || !errors.As(err, &proof) || proof.Actual != status || calls != 1 {
					t.Fatal(row, err, calls)
				}
			}
		})
	}
}
func TestPythonRoleClosedLookupMissingDuplicateLatePageAndGetOnlyMicroversion(t *testing.T) {
	for _, mode := range []string{"missing", "duplicate", "late", "single", "id match"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			api, _ := closedRoleFixture(t, func(q *http.Request) *http.Response {
				calls++
				if q.Header.Get("X-Owned") != "owned" {
					t.Fatal(q.Header)
				}
				if strings.HasSuffix(q.URL.Path, "/reader") {
					if q.Header.Get("OpenStack-API-Version") != "identity 3.8" {
						t.Fatal(q.Header)
					}
					return closedRoleWire(403, `{"error":"denied member"}`)
				}
				if q.Header.Get("OpenStack-API-Version") != "identity 3.1" || q.URL.Query().Get("name") != "reader" || q.URL.Query().Get("domain_id") != "domain" {
					t.Fatal(q.URL, q.Header)
				}
				if calls == 2 {
					if mode == "missing" {
						return closedRoleWire(200, `{"roles":[]}`)
					}
					if mode == "id match" {
						return closedRoleWire(203, `{"roles":[{"id":"reader","name":"different"}]}`)
					}
					return closedRoleWire(203, `{"roles":[{"id":"one","name":"reader"}],"next":"?name=reader&domain_id=domain&marker=one"}`)
				}
				if mode == "late" {
					return closedRoleWire(503, `{"error":"late"}`)
				}
				if mode == "duplicate" {
					return closedRoleWire(200, `{"roles":[{"id":"one","name":"reader"}],"next":"?name=reader&domain_id=domain&marker=another"}`)
				}
				return closedRoleWire(200, `{"roles":[]}`)
			})
			api.RawClient().Microversion = "3.1"
			row, err := api.ResolveRecord(context.Background(), "reader", roles.WithClosedLookupDomainID("domain"), roles.WithClosedLookupHeaders(map[string]string{"X-Owned": "owned"}), roles.WithClosedLookupMicroversion("3.8"))
			switch mode {
			case "missing":
				if err != nil || row != nil || calls != 2 {
					t.Fatal(row, err, calls)
				}
			case "duplicate":
				if row != nil || !errors.Is(err, resource.ErrAmbiguous) || calls != 3 {
					t.Fatal(row, err, calls)
				}
			case "late":
				var proof gophercloud.ErrUnexpectedResponseCode
				if row != nil || !errors.As(err, &proof) || proof.Actual != 503 || calls != 3 {
					t.Fatal(row, err, calls)
				}
			case "single", "id match":
				if err != nil || row == nil || row.StatusCode != 203 {
					t.Fatal(row, err, calls)
				}
			}
		})
	}
}
func TestPythonRoleClosedLookupImmutableOptsSourceCancellationAndExplicitLocation(t *testing.T) {
	domain := json.RawMessage(`"domain"`)
	cloud := "configured"
	location := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`)}}
	headers := map[string]string{"X-Owned": "owned"}
	option := roles.WithClosedLookupOpts(roles.ClosedLookupOpts{DomainID: domain, Headers: headers, Location: &location})
	domain[1] = 'X'
	cloud = "changed"
	headers["X-Owned"] = "changed"
	calls := 0
	api, _ := closedRoleFixture(t, func(q *http.Request) *http.Response {
		calls++
		if q.URL.Query().Get("domain_id") != "domain" || q.Header.Get("X-Owned") != "owned" {
			t.Fatal(q.URL, q.Header)
		}
		return closedRoleWire(200, `{"role":{"id":"a","project_id":"foreign","location":{"cloud":"wire"}}}`)
	})
	row, err := api.ResolveRecord(context.Background(), "a", option)
	if err != nil || row == nil || !strings.Contains(string(row.Resource.Body["location"]), `"cloud":"configured"`) || string(row.Wire.Body["project_id"]) != `"foreign"` {
		t.Fatal(row, err)
	}
	for _, mode := range []string{"nil", "source", "cancel", "domain", "location"} {
		t.Run(mode, func(t *testing.T) {
			before := calls
			seen := 0
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("closed caller cause")
			old := api.RawClient().Endpoint
			defer func() { api.RawClient().Endpoint = old }()
			var first roles.ClosedLookupOption = func(*roles.ClosedLookupOpts) error {
				if mode == "source" {
					api.RawClient().Endpoint = "https://changed.invalid/"
				} else {
					cancel(cause)
				}
				return nil
			}
			if mode == "nil" {
				first = nil
			}
			if mode == "domain" {
				first = roles.WithClosedLookupDomainIDJSON(json.RawMessage(`invalid`))
			}
			if mode == "location" {
				first = roles.WithClosedLookupLocation(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`invalid`)}})
			}
			row, err := api.ResolveRecord(ctx, "a", first, func(*roles.ClosedLookupOpts) error { seen++; return nil })
			var proof *resource.ResponseError
			if row != nil || err == nil || calls != before || errors.As(err, &proof) || ((mode == "nil" || mode == "source" || mode == "cancel") && seen != 0) {
				t.Fatal(row, err, calls, seen, proof)
			}
			if mode == "cancel" && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
				t.Fatal(err)
			}
		})
	}
}
func TestPythonRoleClosedLookupPhysicalFaultStopsFallbackAndNativeCallbacks(t *testing.T) {
	for _, status := range []int{203, 400, 403, 404, 401, 503} {
		for _, mode := range []string{"read", "close", "both"} {
			t.Run(fmt.Sprintf("%d/%s", status, mode), func(t *testing.T) {
				calls, closed, retries, reauth, backoff := 0, 0, 0, 0, 0
				readCause := errors.New("closed read")
				closeCause := errors.New("closed close")
				payload := `{"role":{"id":"one"}}`
				body := &testcloud.Body{Data: payload, OnClose: func() { closed++ }}
				if mode == "read" || mode == "both" {
					body.ReadErr = readCause
				}
				if mode == "close" || mode == "both" {
					body.CloseErr = closeCause
				}
				api, fixture := closedRoleFixture(t, func(*http.Request) *http.Response {
					calls++
					reply := closedRoleWire(status, payload)
					reply.Body = body
					return reply
				})
				fixture.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				fixture.Provider.ReauthFunc = func(context.Context) error { reauth++; return errors.New("unexpected physical-fault reauthentication") }
				fixture.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
					backoff++
					return nil
				}
				row, err := api.ResolveRecord(context.Background(), "one")
				if row != nil || err == nil || calls != 1 || closed != 1 || retries != 0 || reauth != 0 || backoff != 0 {
					t.Fatal(row, err, calls, closed, retries, reauth, backoff)
				}
				if body.ReadErr != nil && !errors.Is(err, readCause) {
					t.Fatal(err)
				}
				if body.CloseErr != nil && !errors.Is(err, closeCause) {
					t.Fatal(err)
				}
				if status < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != payload || proof.Header.Get("X-Request-Id") != "closed-role-proof" {
						t.Fatal(err, proof)
					}
				} else {
					original := err
					if status == 401 {
						var failure *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &failure) || failure.ErrOriginal == nil || failure.ErrReauth == nil {
							t.Fatal(err, failure)
						}
						original = failure.ErrOriginal
					}
					var proof gophercloud.ErrUnexpectedResponseCode
					if !errors.As(original, &proof) || proof.Actual != status || string(proof.Body) != payload || proof.Method != "GET" {
						t.Fatal(err, proof)
					}
				}
			})
		}
	}
}
func TestPythonRoleClosedLookupCleanAuthRetryKeepsFrozenRequest(t *testing.T) {
	for _, status := range []int{401, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls, callbacks := 0, 0
			api, fixture := closedRoleFixture(t, func(q *http.Request) *http.Response {
				calls++
				if q.URL.Query().Get("domain_id") != "domain" || q.Header.Get("X-Selected") != "frozen" || q.Header.Get("If-Match") != "" || q.Header.Get("OpenStack-API-Version") != "identity 3.2" {
					t.Fatal(q.URL, q.Header)
				}
				if calls == 1 {
					return closedRoleWire(status, `{"error":"clean"}`)
				}
				if status == 401 && q.Header.Get("X-Auth-Token") != "renewed" {
					t.Fatal(q.Header)
				}
				return closedRoleWire(200, `{"role":{"id":"one"}}`)
			})
			api.RawClient().MoreHeaders = map[string]string{"X-Selected": "frozen"}
			fixture.Provider.ReauthFunc = func(context.Context) error {
				callbacks++
				fixture.Provider.SetToken("renewed")
				api.RawClient().MoreHeaders["If-Match"] = "injected"
				return nil
			}
			fixture.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				callbacks++
				api.RawClient().MoreHeaders["If-Match"] = "injected"
				return nil
			}
			row, err := api.ResolveRecord(context.Background(), "one", roles.WithClosedLookupDomainID("domain"), roles.WithClosedLookupMicroversion("3.2"))
			if err != nil || row == nil || calls != 2 || callbacks != 1 {
				t.Fatal(row, err, calls, callbacks)
			}
		})
	}
}
func TestPythonRoleClosedLookupPrivateGuardAndMalformedAcceptedReceipt(t *testing.T) {
	cause := errors.New("outer closed guard")
	calls := 0
	api, _ := closedRoleFixture(t, func(*http.Request) *http.Response { calls++; return closedRoleWire(203, `{"role":null}`) })
	ctx := rest.WithOperationGuard(context.Background(), func(context.Context) error { return cause })
	row, err := api.ResolveRecord(ctx, "one")
	var proof *resource.ResponseError
	if row != nil || !errors.Is(err, cause) || calls != 0 || errors.As(err, &proof) {
		t.Fatal(row, err, calls, proof)
	}
	row, err = api.ResolveRecord(context.Background(), "one")
	if row != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != `{"role":null}` || calls != 1 {
		t.Fatal(row, err, calls, proof)
	}
	var empty *roles.API
	row, err = empty.ResolveRecord(context.Background(), "one")
	if row != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(row, err)
	}
}
