package identity_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/testhelper"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
)

func groupGetFixture(t *testing.T, reply func(http.ResponseWriter, *http.Request)) (*identity.GroupReadCloud, *testcloud.Cloud, *atomic.Int32) {
	t.Helper()
	f := testcloud.New(t)
	calls := &atomic.Int32{}
	f.Mux.HandleFunc("/identity/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testhelper.TestMethod(t, r, http.MethodGet)
		w.Header().Set("X-Request-Id", "group-get")
		reply(w, r)
	})
	return identity.NewGroupReadCloud(groups.New(f.Client("identity", "/identity/v3/"))), f, calls
}
func TestCloudGetGroupNoneVersusPresentFalseySearch(t *testing.T) {
	for _, raw := range []string{"omitted", "null", "false", "0", "1e-9999", `""`, "[]", "{}"} {
		t.Run(raw, func(t *testing.T) {
			member, list := 0, 0
			cloud, _, calls := groupGetFixture(t, func(w http.ResponseWriter, r *http.Request) {
				testhelper.AssertEquals(t, "domain", r.URL.Query().Get("domain_id"))
				data, err := io.ReadAll(r.Body)
				if err != nil || len(data) != 0 || r.ContentLength != 0 {
					t.Error(err, string(data), r.ContentLength)
				}
				if r.URL.Path == "/identity/v3/groups/one" {
					member++
					testcloud.JSON(w, 203, `{"group":{"id":"one","name":"member"}}`)
				} else {
					list++
					testhelper.AssertEquals(t, "/identity/v3/groups", r.URL.Path)
					testhelper.AssertEquals(t, "", r.URL.Query().Get("name"))
					testcloud.JSON(w, 200, `{"groups":[{"id":"one","name":"listed"}]}`)
				}
			})
			options := []identity.GroupGetOption{identity.WithGroupGetDomainID("domain")}
			if raw != "omitted" {
				options = append(options, identity.WithGroupGetFilters(json.RawMessage(raw)))
			}
			result, err := cloud.GetGroup(context.Background(), "one", options...)
			deprecated := raw != "omitted" && raw != "null"
			if err != nil || result == nil || result.Group == nil || calls.Load() != 1 || result.DeprecatedFilters != deprecated {
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
			if string(result.Group.Resource.Body["name"]) != want {
				t.Fatal(result)
			}
			result.Group.Resource.Body["name"][1] = 'X'
			if string(result.Inventory[0].Resource.Body["name"]) != want {
				t.Fatal("owned Inventory", result)
			}
		})
	}
}
func TestCloudGetGroupLateNativeFailureBeforeLocalFilter(t *testing.T) {
	cloud, _, calls := groupGetFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("marker") == "" {
			testcloud.JSON(w, 203, `{"groups":[{"id":"one","description":"first"}],"next":"?marker=one"}`)
		} else {
			testcloud.JSON(w, 503, `{"error":"late"}`)
		}
	})
	result, err := cloud.GetGroup(context.Background(), "one", identity.WithGroupGetFilters(json.RawMessage(`{}`)))
	var native gophercloud.ErrUnexpectedResponseCode
	if result == nil || !result.DeprecatedFilters || len(result.Inventory) != 1 || result.Value != nil || result.Group != nil || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"error":"late"}` || native.ResponseHeader.Get("X-Request-Id") != "group-get" || calls.Load() != 2 {
		t.Fatal(result, err, native, calls.Load())
	}
}
func TestCloudGetGroupStickyOptionsCancelAndOuterGuard(t *testing.T) {
	for _, lane := range []string{"nil", "source", "cancel", "outer", "role", "headers"} {
		t.Run(lane, func(t *testing.T) {
			cloud, _, calls := groupGetFixture(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, 200, `{"group":{"id":"one"}}`) })
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("caller witness")
			seen := 0
			var options []identity.GroupGetOption
			switch lane {
			case "nil":
				options = append(options, nil)
			case "source":
				options = append(options, func(*identity.GroupGetOpts) error { cloud.RawAPI().RawClient().ResourceBase += "changed/"; return nil })
			case "cancel":
				cancel(cause)
			case "outer":
				ctx = rest.WithOperationGuard(ctx, func(context.Context) error { return cause })
			case "role":
				cloud.RawAPI().RawClient().Type = "network"
			case "headers":
				options = append(options, identity.WithGroupGetHeaders(map[string]string{"X-Auth-Token": "foreign"}))
			}
			if lane != "headers" {
				options = append(options, func(*identity.GroupGetOpts) error { seen++; return nil })
			}
			result, err := cloud.GetGroup(ctx, "one", options...)
			if err == nil || result != nil && result.Group != nil || calls.Load() != 0 || seen != 0 {
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
func TestCloudGetGroupPhysicalCloseSourceCancelKeepActualNativeEvidence(t *testing.T) {
	for _, status := range []int{203, 401, 404, 429, 503} {
		for _, lane := range []string{"close", "source", "cancel"} {
			t.Run(fmt.Sprintf("%d/%s", status, lane), func(t *testing.T) {
				body := `{"group":{"id":"one"}}`
				cloud, f, calls := groupGetFixture(t, func(w http.ResponseWriter, _ *http.Request) { testcloud.JSON(w, status, body) })
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New(lane + " witness")
				closes := &atomic.Int32{}
				base := f.Provider.HTTPClient.Transport
				f.Provider.HTTPClient.Transport = userCloudHTTP(func(r *http.Request) (*http.Response, error) {
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
				})
				hooks := 0
				f.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks++
					return nil
				}
				f.Provider.ReauthFunc = func(context.Context) error { hooks++; return nil }
				f.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error { hooks++; return nil }
				result, err := cloud.GetGroup(ctx, "one")
				if err == nil || result != nil && (result.Group != nil || result.Value != nil) || calls.Load() != 1 || closes.Load() != 1 || hooks != 0 {
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
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "group-get" {
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
				if !errors.As(receipt, &native) || native.Actual != status || native.Method != http.MethodGet || native.URL != f.Server.URL+"/identity/v3/groups/one" || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "group-get" {
					t.Fatal(err, native)
				}
			})
		}
	}
}

func TestCloudGetGroupJMESFirstTruthLenAndIndexKeepsActualInventory(t *testing.T) {
	cloud, _, calls := groupGetFixture(t, func(w http.ResponseWriter, r *http.Request) {
		testhelper.AssertEquals(t, "False", r.URL.Query().Get("domain_id"))
		testcloud.JSON(w, 200, `{"groups":[{"id":"one","name":"ops-one"}]}`)
	})
	for _, tc := range []struct {
		expression, want       string
		absent, duplicate, bad bool
	}{{expression: "`null`", absent: true}, {expression: "`false`", absent: true}, {expression: "`0`", absent: true}, {expression: "`1e-9999`", absent: true}, {expression: "`[]`", absent: true}, {expression: "`[false]`", want: "false"}, {expression: "`[0]`", want: "0"}, {expression: "`[\"\"]`", want: `""`}, {expression: "`[null]`", absent: true}, {expression: "`\"중\"`", want: `"중"`}, {expression: "`[1,2]`", duplicate: true}, {expression: "`\"중a\"`", duplicate: true}, {expression: "`{\"a\":1,\"b\":2}`", duplicate: true}, {expression: "`true`", bad: true}, {expression: "`1`", bad: true}, {expression: "`{\"0\":1}`", bad: true}} {
		before := calls.Load()
		result, err := cloud.GetGroup(context.Background(), "ops-*", identity.WithGroupGetRawDomainID(json.RawMessage("false")), identity.WithGroupGetFilters(endpointCloudExpression(tc.expression)))
		if result == nil || !result.DeprecatedFilters || !result.Expression || result.Group != nil || len(result.Inventory) != 1 || calls.Load() != before+1 {
			t.Fatal(tc, result, err, calls.Load())
		}
		if tc.duplicate {
			if !errors.Is(err, resource.ErrAmbiguous) {
				t.Fatal(tc, err)
			}
			continue
		}
		if tc.bad {
			if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(tc, err)
			}
			continue
		}
		if err != nil || (result.Value == nil) != tc.absent || (!tc.absent && string(result.Value) != tc.want) {
			t.Fatal(tc, result, err)
		}
	}
}
func TestCloudGetGroupOwnedRawDomainLocationAndLoadedNestedFilter(t *testing.T) {
	cloud, _, calls := groupGetFixture(t, func(w http.ResponseWriter, r *http.Request) {
		testhelper.TestHeader(t, r, "X-Owned", "before")
		testhelper.AssertEquals(t, "0", r.URL.Query().Get("domain_id"))
		testcloud.JSON(w, 203, `{"groups":[{"id":"one","name":"ops-one","description":{"n":9007199254740992,"flag":1}},{"id":"skip","description":{}}]}`)
	})
	raw := json.RawMessage("0")
	header := map[string]string{"X-Owned": "before"}
	domainOption := identity.WithGroupGetRawDomainID(raw)
	headerOption := identity.WithGroupGetHeaders(header)
	header["X-Owned"] = "after"
	raw[0] = '9'
	result, err := cloud.GetGroup(context.Background(), "ops-*", domainOption, headerOption, identity.WithGroupGetFilters(json.RawMessage(`{"description":{"n":9007199254740993.0,"flag":true}}`)))
	if err != nil || result == nil || result.Group == nil || len(result.Inventory) != 2 || calls.Load() != 1 {
		t.Fatal(result, err, calls.Load())
	}
	if string(result.Group.Resource.Body["id"]) != `"one"` {
		t.Fatal(result)
	}
}
