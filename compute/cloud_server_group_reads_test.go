package compute_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	computev2 "github.com/JSYoo5B/go-openstacksdk/compute/v2"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const serverGroupCloudPath = "/nova/v2.1/os-server-groups"
const serverGroupCloudRows = `{"server_groups":[{"id":"g1","name":"db01","members":["s1"],"policy":"anti-affinity","metadata":{"nested":{"x":true}},"rules":{"max_server_per_host":2},"project_id":"p","unknown":"discarded"},{"id":"g2","name":"db02","members":[],"policy":"affinity","rules":null}]}`

// The SDK's public Gophercloud testhelper-backed fixture and existing Nova
// transport/body observer are reused, rather than starting another mock server.
func serverGroupCloudFixture(t *testing.T, reply func(*http.Request) *http.Response) (*compute.Service, *gophercloud.ServiceClient, *[]string) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := &[]string{}
	cloud.SetTransport(testcloud.LinkRequest(func(r *http.Request) (*http.Response, error) {
		*calls = append(*calls, r.Method+" "+r.URL.EscapedPath()+"?"+r.URL.RawQuery)
		return reply(r), nil
	}))
	client := cloud.Client("compute", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/nova/v2.1/"
	client.Microversion = "2.64"
	return compute.New(client, compute.Dependencies{}), client, calls
}
func serverGroupCloudIDs(t *testing.T, r *compute.ServerGroupCloudSearchResult) []string {
	t.Helper()
	ids := []string{}
	for _, row := range r.Groups {
		var id string
		if e := json.Unmarshal(row.Resource.Body["id"], &id); e != nil {
			t.Fatal(e)
		}
		ids = append(ids, id)
	}
	return ids
}
func TestCloudServerGroupReadsSearchExactGlobOrderedDictCanonicalAndNoPushdown(t *testing.T) {
	cases := []struct {
		name, filter string
		ids          []string
		bad          bool
	}{
		{"", "", []string{"g1", "g2"}, false}, {"g1", "", []string{"g1"}, false}, {"db0*", "", []string{"g1", "g2"}, false}, {"db0[12]", "", []string{"g1", "g2"}, false}, {"db0?", "", []string{"g1", "g2"}, false},
		{"", `{"policy":"anti-affinity"}`, []string{"g1"}, false}, {"", `{"metadata":{"nested":{"x":1}}}`, []string{"g1"}, false}, {"", `{"member_ids":["s1"]}`, []string{"g1"}, false}, {"", `{"members":["s1"]}`, nil, true}, {"", `{"unknown":"discarded"}`, nil, true},
		{"", `{"policy":"not","unknown":true}`, []string{}, false}, {"", `{"unknown":true,"policy":"not"}`, nil, true}, {"absent", `true`, []string{}, false}, {"", `true`, nil, true}, {"", `[1]`, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/"+tc.filter, func(t *testing.T) {
			service, _, calls := serverGroupCloudFixture(t, func(r *http.Request) *http.Response {
				if r.URL.Path != serverGroupCloudPath || r.URL.RawQuery != "" || r.Method != http.MethodGet {
					t.Error(r.Method, r.URL)
				}
				return backupServerWire(200, serverGroupCloudRows, "groups")
			})
			opts := []compute.ServerGroupCloudReadOption{}
			if tc.filter != "" {
				opts = append(opts, compute.WithServerGroupCloudFilters(json.RawMessage(tc.filter)))
			}
			result, e := service.SearchServerGroups(context.Background(), tc.name, opts...)
			if result == nil || len(result.Inventory) != 2 || len(*calls) != 1 {
				t.Fatal(result, e, *calls)
			}
			if tc.bad {
				if !errors.Is(e, resource.ErrInvalidOption) || result.Value != nil || result.Groups != nil {
					t.Fatal(result, e)
				}
				return
			}
			if e != nil || !reflect.DeepEqual(serverGroupCloudIDs(t, result), tc.ids) || result.Value == nil {
				t.Fatal(result, e, serverGroupCloudIDs(t, result))
			}
			if result.Inventory[0].Resource.Body["unknown"] != nil || result.Inventory[0].Resource.Body["members"] != nil || string(result.Inventory[0].Wire.Body["unknown"]) != `"discarded"` {
				t.Fatal(result)
			}
		})
	}
}
func TestCloudServerGroupReadsGetPresenceDefaultFindVersusDeprecatedSearch(t *testing.T) {
	for _, raw := range []string{"", "null", "{}", "false", "0", `""`, "[]", "1e-400"} {
		t.Run(raw, func(t *testing.T) {
			service, _, calls := serverGroupCloudFixture(t, func(r *http.Request) *http.Response {
				if r.URL.Path == serverGroupCloudPath {
					return backupServerWire(200, serverGroupCloudRows, "list")
				}
				return backupServerWire(200, `{"server_group":{"id":"g1","name":"direct"}}`, "direct")
			})
			opts := []compute.ServerGroupCloudReadOption{}
			if raw != "" {
				opts = append(opts, compute.WithServerGroupCloudFilters(json.RawMessage(raw)))
			}
			result, e := service.GetServerGroup(context.Background(), "g1", opts...)
			filtered := raw != "" && raw != "null"
			want := []string{"GET " + serverGroupCloudPath + "/g1?"}
			name := `"direct"`
			if filtered {
				want = []string{"GET " + serverGroupCloudPath + "?"}
				name = `"db01"`
			}
			if e != nil || result == nil || result.Group == nil || result.Filtered != filtered || string(result.Group.Resource.Body["name"]) != name || !reflect.DeepEqual(*calls, want) {
				t.Fatal(result, e, *calls, want)
			}
		})
	}
}
func TestCloudServerGroupReadsJMESArbitraryJSONAndFilteredGetIndexSemantics(t *testing.T) {
	cases := []struct {
		expression, want   string
		ambiguous, invalid bool
	}{
		{"'x'", `"x"`, false, false}, {"'ab'", "", true, false}, {"length(@)", "", false, true}, {"`0`", "", false, false}, {"`false`", "", false, false}, {"`null`", "", false, false}, {"`[]`", "", false, false}, {"`{}`", "", false, false}, {"`1e-400`", "", false, false}, {"`true`", "", false, true}, {"`1`", "", false, true},
		{"`[false]`", "false", false, false}, {"`[0]`", "0", false, false}, {"`[\"\"]`", `""`, false, false}, {"`[null]`", "", false, false}, {"`[{},{}]`", "", true, false}, {"`{\"a\":1}`", "", false, true}, {"`{\"a\":1,\"b\":2}`", "", true, false}, {"[0].members", "", false, false}, {"[0].member_ids", `"s1"`, false, false}, {"[?", "", false, true}, {"unknown(@)", "", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.expression, func(t *testing.T) {
			service, _, calls := serverGroupCloudFixture(t, func(*http.Request) *http.Response { return backupServerWire(200, serverGroupCloudRows, "expression") })
			result, e := service.GetServerGroup(context.Background(), "", compute.WithServerGroupCloudExpression(tc.expression))
			if result == nil || !result.Filtered || len(result.Inventory) != 2 || result.Group != nil || len(*calls) != 1 {
				t.Fatal(result, e, *calls)
			}
			if tc.ambiguous {
				if !errors.Is(e, resource.ErrAmbiguous) || result.Value != nil {
					t.Fatal(result, e)
				}
			} else if tc.invalid {
				if !errors.Is(e, resource.ErrInvalidOption) || result.Value != nil {
					t.Fatal(result, e)
				}
			} else if e != nil || string(result.Value) != tc.want {
				t.Fatal(result, e, tc.want)
			}
		})
	}
	service, _, _ := serverGroupCloudFixture(t, func(*http.Request) *http.Response { return backupServerWire(200, serverGroupCloudRows, "arbitrary") })
	for expression, want := range map[string]string{"[].id": `["g1","g2"]`, "length(@)": "2", "'x'": `"x"`, "`null`": "null", "[0].members": "null"} {
		r, e := service.SearchServerGroups(context.Background(), "", compute.WithServerGroupCloudExpression(expression))
		if e != nil || r == nil || string(r.Value) != want || !r.Expression || r.Groups != nil || len(r.Inventory) != 2 {
			t.Fatal(expression, r, e, want)
		}
	}
}
func TestCloudServerGroupReadsCompleteSearchAndDefaultFindStopsSecondDuplicate(t *testing.T) {
	for _, filtered := range []bool{false, true} {
		service, _, calls := serverGroupCloudFixture(t, func(r *http.Request) *http.Response {
			if r.URL.Path != serverGroupCloudPath {
				return backupServerWire(404, `{"error":"member absent"}`, "lookup")
			}
			if r.URL.Query().Get("marker") == "next" {
				return backupServerWire(503, `{"error":"late page"}`, "late")
			}
			return backupServerWire(200, `{"server_groups":[{"id":"first","name":"same"},{"id":"second","name":"same"}],"server_groups_links":[{"rel":"next","href":"?marker=next"}]}`, "duplicates")
		})
		opts := []compute.ServerGroupCloudReadOption{}
		if filtered {
			opts = append(opts, compute.WithServerGroupCloudFilters(json.RawMessage("{}")))
		}
		result, e := service.GetServerGroup(context.Background(), "same", opts...)
		if result == nil || result.Group != nil || result.Value != nil || len(result.Inventory) != 2 || len(*calls) != 2 {
			t.Fatal(result, e, *calls)
		}
		if filtered {
			if !gophercloud.ResponseCodeIs(e, 503) || (*calls)[1] != "GET "+serverGroupCloudPath+"?marker=next" {
				t.Fatal(result, e, *calls)
			}
		} else if !errors.Is(e, resource.ErrAmbiguous) || result.Observed == nil || result.Observed.StatusCode != 404 {
			t.Fatal(result, e, *calls)
		}
	}
	service, _, calls := serverGroupCloudFixture(t, func(r *http.Request) *http.Response {
		if r.URL.Query().Get("marker") == "next" {
			return backupServerWire(200, `{"server_groups":[{"id":"g2","name":"excluded"}]}`, "tail")
		}
		return backupServerWire(200, `{"server_groups":[{"id":"g1","name":"wanted"}],"server_groups_links":[{"rel":"next","href":"?marker=next"}]}`, "head")
	})
	result, e := service.SearchServerGroups(context.Background(), "wanted")
	if e != nil || len(result.Inventory) != 2 || len(result.Groups) != 1 || len(*calls) != 2 || result.Groups[0].StatusCode != 200 {
		t.Fatal(result, e, *calls)
	}
}
func TestCloudServerGroupReadsDefaultExactNoGlobMissingEmptyCollectionAndNativeErrors(t *testing.T) {
	for _, name := range []string{"g1", "db01", "db0*", "absent", ""} {
		service, _, calls := serverGroupCloudFixture(t, func(r *http.Request) *http.Response {
			if r.URL.Path == serverGroupCloudPath {
				return backupServerWire(200, serverGroupCloudRows, "fallback")
			}
			return backupServerWire(404, `{"error":"member absent"}`, "member")
		})
		result, e := service.GetServerGroup(context.Background(), name)
		match := name == "g1" || name == "db01"
		literal := name
		if name == "db0*" {
			literal = "db0%2A" // Go single-segment encoding; find still performs exact equality.
		}
		want := "GET " + serverGroupCloudPath + "/" + literal + "?"
		if e != nil || result == nil || (result.Group != nil) != match || len(result.Inventory) != 2 || len(*calls) != 2 || (*calls)[0] != want || (*calls)[1] != "GET "+serverGroupCloudPath+"?" {
			t.Fatal(name, result, e, *calls)
		}
	}
	for _, status := range []int{400, 403, 404} {
		service, _, calls := serverGroupCloudFixture(t, func(r *http.Request) *http.Response {
			if r.URL.Path != serverGroupCloudPath {
				return backupServerWire(status, `{"error":"clean"}`, "member")
			}
			return backupServerWire(200, `{"server_groups":[]}`, "empty")
		})
		result, e := service.GetServerGroup(context.Background(), "missing")
		if e != nil || result == nil || result.Group != nil || result.Value != nil || result.Observed.StatusCode != status || len(*calls) != 2 {
			t.Fatal(result, e, *calls)
		}
	}
	for _, status := range []int{401, 409, 503} {
		service, _, calls := serverGroupCloudFixture(t, func(*http.Request) *http.Response { return backupServerWire(status, `{"error":"failure"}`, "rejected") })
		result, e := service.GetServerGroup(context.Background(), "g1")
		var native gophercloud.ErrUnexpectedResponseCode
		if result == nil || result.Group != nil || result.Observed == nil || result.Observed.StatusCode != status || !errors.As(e, &native) || native.Actual != status || len(*calls) != 1 {
			t.Fatal(result, e, native, *calls)
		}
	}
}
func TestCloudServerGroupReadsOwnsOptionsRecordsHeadersAndCurrentLocation(t *testing.T) {
	var selectedHeader string
	service, client, _ := serverGroupCloudFixture(t, func(r *http.Request) *http.Response {
		selectedHeader = r.Header.Get("X-Owned")
		return backupServerWire(203, serverGroupCloudRows, "owned")
	})
	name, project := "cloud", json.RawMessage(`"current"`)
	location := resource.CloudLocation{Cloud: &name, Project: resource.CloudProject{ID: project, Name: &name}}
	loc := compute.WithServerGroupCloudCurrentLocation(location)
	name = "changed"
	project[1] = 'X'
	raw := json.RawMessage(`{"policy":"anti-affinity"}`)
	filter := compute.WithServerGroupCloudFilters(raw)
	raw[2] = 'X'
	headers := map[string]string{"X-Owned": "before"}
	header := compute.WithServerGroupCloudHeaders(headers)
	headers["X-Owned"] = "wrong"
	client.MoreHeaders = map[string]string{"X-Original": "original"}
	var retained *compute.ServerGroupCloudReadOpts
	result, e := service.SearchServerGroups(context.Background(), "", loc, filter, header, func(c *compute.ServerGroupCloudReadOpts) error { retained = c; return nil }, func(*compute.ServerGroupCloudReadOpts) error {
		retained.Headers["X-Owned"] = "late"
		(*retained.Filters)[2] = 'X'
		return nil
	})
	if e != nil || result == nil || len(result.Groups) != 1 || selectedHeader != "before" {
		t.Fatal(result, e, selectedHeader)
	}
	var facts resource.CloudLocation
	if e := json.Unmarshal(result.Groups[0].Resource.Body["location"], &facts); e != nil || facts.Cloud == nil || *facts.Cloud != "cloud" || string(facts.Project.ID) != `"p"` || facts.Project.Name != nil {
		t.Fatal(facts, e)
	}
	result.Groups[0].Resource.Body["name"][1] = 'X'
	result.Groups[0].Header.Set("X-Backup-Stage", "changed")
	result.Groups[0].Envelope[0] = 'X'
	if string(result.Inventory[0].Resource.Body["name"]) != `"db01"` || result.Inventory[0].Header.Get("X-Backup-Stage") != "owned" || string(result.Groups[0].Wire.Body["name"]) != `"db01"` {
		t.Fatal(result)
	}
	result, e = service.SearchServerGroups(context.Background(), "", compute.WithServerGroupCloudFilters(json.RawMessage(`{"unknown":true}`)), compute.WithServerGroupCloudFilters(nil))
	if e != nil || len(result.Groups) != 2 {
		t.Fatal(result, e)
	}
}
func TestCloudServerGroupReadsPreflightSelectedAPIOptionsAndCustomCancellation(t *testing.T) {
	service, client, calls := serverGroupCloudFixture(t, func(*http.Request) *http.Response { return backupServerWire(200, serverGroupCloudRows, "unexpected") })
	for _, options := range [][]compute.ServerGroupCloudReadOption{{nil}, {compute.WithServerGroupCloudFilters(json.RawMessage(`broken`))}, {compute.WithServerGroupCloudHeaders(map[string]string{"X-Auth-Token": "forged"})}, {compute.WithServerGroupCloudCurrentLocation(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`broken`)}})}} {
		result, e := service.SearchServerGroups(context.Background(), "", options...)
		if result != nil || !errors.Is(e, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(result, e, *calls)
		}
	}
	var nilService *compute.Service
	if r, e := nilService.SearchServerGroups(context.Background(), ""); r != nil || !errors.Is(e, resource.ErrInvalidOption) {
		t.Fatal(r, e)
	}
	if r, e := service.SearchServerGroups(nil, ""); r != nil || !errors.Is(e, resource.ErrInvalidOption) {
		t.Fatal(r, e)
	}
	cause := errors.New("caller cancelled")
	for _, kind := range []string{"API", "group API", "cancel", "route"} {
		originalAPI, originalGroup, originalEndpoint := service.API, service.API.ServerGroups, client.Endpoint
		ctx, cancel := context.WithCancelCause(context.Background())
		option := func(*compute.ServerGroupCloudReadOpts) error {
			switch kind {
			case "API":
				service.API = computev2.New(client)
			case "group API":
				service.API.ServerGroups = computev2.New(client).ServerGroups
			case "cancel":
				cancel(cause)
			default:
				client.Endpoint += "changed/"
			}
			return nil
		}
		r, e := service.SearchServerGroups(ctx, "", option)
		if r != nil || e == nil || len(*calls) != 0 || kind == "cancel" && (!errors.Is(e, context.Canceled) || !errors.Is(e, cause)) || kind != "cancel" && !errors.Is(e, resource.ErrInvalidOption) {
			t.Fatal(r, e, *calls)
		}
		service.API = originalAPI
		service.API.ServerGroups = originalGroup
		client.Endpoint = originalEndpoint
		cancel(nil)
	}
}
func TestCloudServerGroupReadsPhysicalFaultsCannotFallbackRetryOrBecomeMissing(t *testing.T) {
	for _, op := range []string{"search", "get", "discovery"} {
		for _, status := range []int{200, 401, 404, 429, 503} {
			for _, kind := range []string{"read-close", "cancel", "restored-source"} {
				t.Run(fmt.Sprint(op, "/", status, "/", kind), func(t *testing.T) {
					readErr, closeErr, cancelCause := errors.New("group physical read"), errors.New("group physical close"), errors.New("group custom cancel")
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					raw := `{"server_groups":[{"id":"g1"}]}`
					if op == "get" {
						raw = `{"server_group":{"id":"g1"}}`
					}
					body := &testcloud.Body{Data: raw}
					var client *gophercloud.ServiceClient
					service, owned, calls := serverGroupCloudFixture(t, func(*http.Request) *http.Response {
						return testcloud.StreamResponse(status, body, http.Header{"X-Actual": {"group physical"}})
					})
					client = owned
					if op == "discovery" {
						client.Microversion = ""
						client.Endpoint = client.ResourceBase
					}
					original := client.ResourceBase
					switch kind {
					case "read-close":
						body.ReadErr, body.CloseErr = readErr, closeErr
					case "cancel":
						body.OnClose = func() { cancel(cancelCause) }
					default:
						body.OnRead = func() { client.ResourceBase = original + "changed/" }
						body.OnClose = func() { client.ResourceBase = original }
					}
					retries, reauths, backoffs := 0, 0, 0
					client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
						retries++
						return errors.New("unexpected bounded group retry")
					}
					client.ReauthFunc = func(context.Context) error { reauths++; return errors.New("unexpected bounded group reauth") }
					client.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
						backoffs++
						return errors.New("unexpected bounded group backoff")
					}
					var e error
					if op == "get" {
						r, cause := service.GetServerGroup(ctx, "g1")
						e = cause
						if r != nil && (r.Group != nil || r.Value != nil) {
							t.Fatal(r, e)
						}
					} else {
						r, cause := service.SearchServerGroups(ctx, "")
						e = cause
						if r != nil && (r.Groups != nil || r.Value != nil) {
							t.Fatal(r, e)
						}
					}
					if e == nil || len(*calls) != 1 || body.Closes() != 1 || retries != 0 || reauths != 0 || backoffs != 0 {
						t.Fatal(e, *calls, body.Closes(), retries, reauths, backoffs)
					}
					if kind == "read-close" && status != 401 && (!errors.Is(e, readErr) || !errors.Is(e, closeErr)) || kind == "cancel" && (!errors.Is(e, context.Canceled) || !errors.Is(e, cancelCause)) || kind == "restored-source" && !errors.Is(e, resource.ErrInvalidOption) {
						t.Fatal(e)
					}
					if status < 400 {
						var proof *resource.ResponseError
						if !errors.As(e, &proof) || proof.StatusCode != status || string(proof.Body) != raw || proof.Header.Get("X-Actual") != "group physical" {
							t.Fatal(e, proof)
						}
					} else {
						actual := e
						if status == 401 {
							var auth *gophercloud.ErrUnableToReauthenticate
							if !errors.As(e, &auth) || auth.ErrOriginal == nil || auth.ErrReauth == nil {
								t.Fatal(e, auth)
							}
							actual = auth.ErrOriginal
						}
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(actual, &native) || native.Actual != status || string(native.Body) != raw || native.Method != http.MethodGet {
							t.Fatal(e, native)
						}
					}
				})
			}
		}
	}
}
func TestCloudServerGroupReadsCleanRetryReauthStrictStatusAndHeaderCapture(t *testing.T) {
	for _, op := range []string{"search", "get"} {
		for _, kind := range []string{"retry", "reauth", "route", "source199", "expanded-codes"} {
			t.Run(op+"/"+kind, func(t *testing.T) {
				attempts := 0
				tokens := []string{}
				versions := []string{}
				service, client, calls := serverGroupCloudFixture(t, func(r *http.Request) *http.Response {
					attempts++
					tokens = append(tokens, r.Header.Get("X-Auth-Token"))
					versions = append(versions, r.Header.Get("X-OpenStack-Nova-API-Version"))
					if r.Header.Get("X-Owned") != "before" {
						t.Error(r.Header)
					}
					if (attempts == 1 || kind == "expanded-codes") && kind != "source199" {
						status := 503
						if kind == "reauth" {
							status = 401
						}
						return backupServerWire(status, `{"error":"original"}`, "original")
					}
					raw := serverGroupCloudRows
					if op == "get" {
						raw = `{"server_group":{"id":"g1","name":"direct"}}`
					}
					status := 200
					if kind == "source199" {
						status = 199
					}
					return backupServerWire(status, raw, "complete")
				})
				client.MoreHeaders = map[string]string{"X-Owned": "before"}
				callbacks := 0
				if kind == "reauth" {
					client.ReauthFunc = func(context.Context) error { callbacks++; client.SetToken("renewed"); return nil }
				} else if kind != "source199" {
					client.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
						callbacks++
						if kind == "route" {
							client.ResourceBase += "changed/"
						}
						if kind == "expanded-codes" {
							o.OkCodes = append(o.OkCodes, 503)
						}
						return nil
					}
				}
				option := func(*compute.ServerGroupCloudReadOpts) error { client.MoreHeaders["X-Owned"] = "after"; return nil }
				var e error
				if op == "get" {
					r, cause := service.GetServerGroup(context.Background(), "g1", option)
					e = cause
					if kind != "route" && kind != "expanded-codes" && (r == nil || r.Group == nil || r.Group.StatusCode != map[bool]int{true: 199, false: 200}[kind == "source199"]) {
						t.Fatal(r, e)
					}
				} else {
					r, cause := service.SearchServerGroups(context.Background(), "", option)
					e = cause
					if kind != "route" && kind != "expanded-codes" && (r == nil || len(r.Groups) != 2 || r.Inventory[0].StatusCode != map[bool]int{true: 199, false: 200}[kind == "source199"]) {
						t.Fatal(r, e)
					}
				}
				if kind == "route" {
					if !errors.Is(e, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(e, 503) || len(*calls) != 1 || callbacks != 1 {
						t.Fatal(e, *calls, callbacks)
					}
					return
				}
				if kind == "expanded-codes" {
					if !gophercloud.ResponseCodeIs(e, 503) || len(*calls) != 2 || callbacks != 1 {
						t.Fatal(e, *calls, callbacks)
					}
					return
				}
				want := 2
				if kind == "source199" {
					want = 1
				}
				if e != nil || len(*calls) != want {
					t.Fatal(e, *calls)
				}
				for _, version := range versions {
					if version != "2.64" {
						t.Fatal(versions)
					}
				}
				if kind == "reauth" && !reflect.DeepEqual(tokens, []string{"test-token", "renewed"}) {
					t.Fatal(tokens)
				}
			})
		}
	}
}
func TestCloudServerGroupReadsLoadedNumericMappingAndJMESPredicate(t *testing.T) {
	raw := `{"server_groups":[{"id":"float","name":"float","rules":{"max_server_per_host":9007199254740992.0}},{"id":"int","name":"int","rules":{"max_server_per_host":9007199254740993}}]}`
	service, _, _ := serverGroupCloudFixture(t, func(*http.Request) *http.Response { return backupServerWire(200, raw, "numeric") })
	result, e := service.SearchServerGroups(context.Background(), "", compute.WithServerGroupCloudFilters(json.RawMessage(`{"rules":{"max_server_per_host":9007199254740993.0}}`)))
	if e != nil || !reflect.DeepEqual(serverGroupCloudIDs(t, result), []string{"float"}) {
		t.Fatal(result, e)
	}
	expression := "[?rules.max_server_per_host == `9007199254740993.0`]"
	found, e := service.GetServerGroup(context.Background(), "", compute.WithServerGroupCloudExpression(expression))
	if e != nil || found == nil || found.Group != nil || len(found.Inventory) != 2 {
		t.Fatal(found, e)
	}
	var row map[string]json.RawMessage
	if e := json.Unmarshal(found.Value, &row); e != nil || string(row["id"]) != `"float"` {
		t.Fatal(found, e, row)
	}
}
