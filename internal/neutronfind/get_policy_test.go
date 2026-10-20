package neutronfind_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestSourceMemberGetOwnsFetchQueryVersionAndPassiveEvidence(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("network", "/catalog/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	client.Microversion = "2.0"
	client.MoreHeaders = map[string]string{"Openstack-Api-Version": "network 2.0", "X-Source": "original"}
	var seen *http.Request
	cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
		seen = req.Clone(req.Context())
		response := policyWire(203, `{"record":{"id":null,"name":"actual","cls":false}}`)
		response.Header.Set("X-Proof", "member")
		return response, nil
	})
	engine, err := neutronfind.New(context.Background(), client, policyProfile(true), nil)
	if err != nil {
		t.Fatal(err)
	}
	version := "2.9"
	query := map[string]json.RawMessage{"fields": json.RawMessage(`["target_tenant","id","target_tenant"]`)}
	value, err := engine.Get(context.Background(), "folder/name%", neutronfind.Options{Microversion: &version, Query: query})
	if err != nil || value == nil || value.Resource == nil || value.Wire == nil {
		t.Fatal(value, err)
	}
	if seen.URL.EscapedPath() != "/neutron/v2.0/records/folder%2Fname%25" || !reflect.DeepEqual(seen.URL.Query()["fields"], []string{"target_tenant", "id", "target_tenant"}) || seen.Header.Get("OpenStack-API-Version") != "network 2.9" {
		t.Fatal(seen)
	}
	if string(value.Resource.Body["id"]) != "null" || string(value.Resource.Body["cls"]) != "false" || value.Resource.Body["fields"] != nil || value.Wire.Body["fields"] != nil || value.StatusCode != 203 || value.Header.Get("X-Proof") != "member" {
		t.Fatal(value)
	}
	if client.Microversion != "2.0" || client.MoreHeaders["Openstack-Api-Version"] != "network 2.0" {
		t.Fatal(client)
	}
	query["fields"][2] = 'x'
	if !strings.Contains(string(value.Envelope), `"name":"actual"`) {
		t.Fatal(value)
	}
}

func TestSourceMemberGetNeverFallsBackAndRetainsRejectedAndDecodeEvidence(t *testing.T) {
	for _, lane := range []struct {
		code int
		body string
	}{{404, `{"error":"missing"}`}, {200, `{"record":false}`}} {
		t.Run(http.StatusText(lane.code), func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			client := cloud.Client("network", "/neutron/v2.0/")
			cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				res := policyWire(lane.code, lane.body)
				res.Header.Set("X-Proof", "actual")
				return res, nil
			})
			engine, err := neutronfind.New(context.Background(), client, policyProfile(true), nil)
			if err != nil {
				t.Fatal(err)
			}
			value, err := engine.Get(context.Background(), "missing", neutronfind.Options{})
			if err == nil || calls != 1 || value != nil && value.Resource != nil {
				t.Fatal(value, err, calls)
			}
			if lane.code == 404 {
				var rejected gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &rejected) || rejected.Actual != 404 || string(rejected.Body) != lane.body || rejected.ResponseHeader.Get("X-Proof") != "actual" {
					t.Fatal(err)
				}
			} else {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != lane.body || proof.Header.Get("X-Proof") != "actual" {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestSourceConstructorClassCollisionIsOptInAndPreservesActualListPage(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, raw := range []string{"null", "false"} {
			t.Run(strings.Join([]string{map[bool]string{false: "baseline", true: "source"}[enabled], raw}, "/"), func(t *testing.T) {
				cloud := testcloud.New(t)
				calls := 0
				client := cloud.Client("network", "/neutron/v2.0/")
				body := `{"records":[{"id":"one","cls":` + raw + `}]}`
				cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
					calls++
					res := policyWire(200, body)
					res.Header.Set("X-Proof", "page")
					return res, nil
				})
				engine, err := neutronfind.New(context.Background(), client, policyProfile(enabled), nil)
				if err != nil {
					t.Fatal(err)
				}
				yielded := 0
				var got error
				for value, err := range engine.List(context.Background(), neutronfind.Options{}) {
					if err != nil {
						got = err
						break
					}
					if value != nil {
						yielded++
					}
				}
				if enabled {
					var proof *resource.ResponseError
					if yielded != 0 || !errors.As(got, &proof) || string(proof.Body) != body || proof.Header.Get("X-Proof") != "page" {
						t.Fatal(yielded, got)
					}
				} else if got != nil || yielded != 1 {
					t.Fatal(yielded, got)
				}
				if calls != 1 {
					t.Fatal(calls)
				}
				if !enabled {
					return
				}
				opts := neutronfind.Options{Query: map[string]json.RawMessage{"cls": json.RawMessage(raw)}}
				_, err = engine.Get(context.Background(), "one", opts)
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				_, err = engine.Find(context.Background(), "one", opts)
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				for _, err = range engine.List(context.Background(), opts) {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				}
				if calls != 1 {
					t.Fatal("preflight sent HTTP", calls)
				}
			})
		}
	}
}

func TestSourceMemberVersionAttributeUsesSelectedPolicyWhileWireStaysActual(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, version := range []string{"", "2.2"} {
			t.Run(map[bool]string{false: "baseline", true: "Source"}[enabled]+"/"+version, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("network", "/neutron/v2.0/")
				client.Microversion = version
				cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
					return policyWire(200, `{"record":{"id":"actual","microversion":"foreign"}}`), nil
				})
				engine, err := neutronfind.New(context.Background(), client, policyProfile(enabled), nil)
				if err != nil {
					t.Fatal(err)
				}
				value, err := engine.Get(context.Background(), "target", neutronfind.Options{})
				if err != nil || value == nil {
					t.Fatal(value, err)
				}
				want := `"foreign"`
				if enabled {
					want = "null"
					if version != "" {
						want = `"` + version + `"`
					}
				}
				if string(value.Resource.Body["microversion"]) != want || string(value.Wire.Body["microversion"]) != `"foreign"` {
					t.Fatal(value, want)
				}
			})
		}
	}
}
func TestSourceFindControlsNeverBecomeFetchQueryOrConstructorSeed(t *testing.T) {
	for _, key := range []string{"name_or_id", "ignore_missing"} {
		cloud := testcloud.New(t)
		client := cloud.Client("network", "/neutron/v2.0/")
		calls := 0
		cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) { calls++; return policyWire(200, `{"record":{}}`), nil })
		engine, err := neutronfind.New(context.Background(), client, policyProfile(true), nil)
		if err != nil {
			t.Fatal(err)
		}
		_, err = engine.Find(context.Background(), "target", neutronfind.Options{Query: map[string]json.RawMessage{key: json.RawMessage("false")}})
		if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatal(key, err, calls)
		}
	}
}

func TestSourceComputedLocationIgnoresResponseExtensionAndKeepsWire(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, lane := range []string{"get", "find", "list"} {
			t.Run(map[bool]string{false: "baseline", true: "Source"}[enabled]+"/"+lane, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("network", "/neutron/v2.0/")
				calls := 0
				row := `{"id":"one","location":{"project":{"id":"forged"}}}`
				body := `{"record":` + row + `}`
				if lane == "list" {
					body = `{"records":[` + row + `]}`
				}
				cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) { calls++; return policyWire(200, body), nil })
				engine, err := neutronfind.New(context.Background(), client, policyProfile(enabled), nil)
				if err != nil {
					t.Fatal(err)
				}
				var value *neutronfind.Record
				switch lane {
				case "get":
					value, err = engine.Get(context.Background(), "one", neutronfind.Options{})
				case "find":
					value, err = engine.Find(context.Background(), "one", neutronfind.Options{})
				case "list":
					for current, cause := range engine.List(context.Background(), neutronfind.Options{}) {
						value, err = current, cause
						break
					}
				}
				if err != nil || value == nil || calls != 1 {
					t.Fatal(value, err, calls)
				}
				want := `{"project":{"id":"forged"}}`
				if enabled {
					want = "null"
				}
				if string(value.Resource.Body["location"]) != want || string(value.Wire.Body["location"]) != `{"project":{"id":"forged"}}` || string(value.Envelope) != body {
					t.Fatal(value, want)
				}
				value.Resource.Body["location"][0] = 'x'
				if string(value.Wire.Body["location"]) != `{"project":{"id":"forged"}}` || string(value.Envelope) != body {
					t.Fatal("projection aliases actual response", value)
				}
			})
		}
	}
}

func TestAlternatePhysicalMarkerIsOptInAndPrimaryPresenceWins(t *testing.T) {
	cases := []struct {
		name, row, want string
		alternate       bool
	}{
		{"opt-in", `{"type":"alternate"}`, "alternate", true},
		{"baseline", `{"type":"alternate"}`, "", false},
		{"primary", `{"id":"primary","type":"alternate"}`, "primary", true},
		{"null", `{"id":null,"type":"alternate"}`, "", true},
		{"empty", `{"id":"","type":"alternate"}`, "", true},
		{"false", `{"id":false,"type":"alternate"}`, "False", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("network", "/neutron/v2.0/")
			profile := policyProfile(true)
			if tc.alternate {
				profile.AlternateIDKey = "type"
			}
			profile.Body[0].Alias = "type"
			profile.Query = append(profile.Query, neutronfind.QueryPair{Name: "limit", Wire: "limit"})
			calls := 0
			body := `{"records":[` + tc.row + `]}`
			cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					if req.URL.Query().Get("marker") != "" {
						t.Fatal(req.URL)
					}
					return policyWire(200, body), nil
				}
				if calls != 2 || tc.want == "" || req.URL.Query().Get("marker") != tc.want {
					t.Fatal("alternate marker changed primary presence", req.URL, calls, tc.want)
				}
				return policyWire(200, `{"records":[]}`), nil
			})
			engine, err := neutronfind.New(context.Background(), client, profile, nil)
			if err != nil {
				t.Fatal(err)
			}
			rows := 0
			var got error
			for value, cause := range engine.List(context.Background(), neutronfind.Options{Query: map[string]json.RawMessage{"limit": json.RawMessage("1")}}) {
				if cause != nil {
					got = cause
					break
				}
				if value != nil {
					rows++
				}
			}
			if rows != 1 {
				t.Fatal(rows, got)
			}
			if tc.want != "" {
				if got != nil || calls != 2 {
					t.Fatal(got, calls)
				}
				return
			}
			var proof *resource.ResponseError
			if !errors.Is(got, resource.ErrInvalidOption) || !errors.As(got, &proof) || calls != 1 || proof.StatusCode != 200 || string(proof.Body) != body {
				t.Fatal(got, calls, proof)
			}
		})
	}
}

func TestSourceRetryCannotAddUnselectedConditionsWhileBaselineRetainsPolicy(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, key := range []string{"If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since"} {
			t.Run(map[bool]string{false: "baseline", true: "Source"}[enabled]+"/"+key, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := cloud.Client("network", "/neutron/v2.0/")
				calls, hooks := 0, 0
				cloud.Provider.RetryFunc = func(ctx context.Context, method, target string, options *gophercloud.RequestOpts, cause error, count uint) error {
					hooks++
					options.MoreHeaders[key] = "foreign"
					return nil
				}
				cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
					calls++
					if calls == 1 {
						return policyWire(503, `{"error":"actual first"}`), nil
					}
					if calls != 2 || req.Header.Get(key) != "foreign" {
						t.Fatal(req.Header, calls)
					}
					return policyWire(200, `{"record":{"id":"one"}}`), nil
				})
				engine, err := neutronfind.New(context.Background(), client, policyProfile(enabled), nil)
				if err != nil {
					t.Fatal(err)
				}
				row, err := engine.Get(context.Background(), "one", neutronfind.Options{})
				if !enabled {
					if err != nil || row == nil || calls != 2 || hooks != 1 {
						t.Fatal(row, err, calls, hooks)
					}
					return
				}
				var actual gophercloud.ErrUnexpectedResponseCode
				if row != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &actual) || actual.Actual != 503 || string(actual.Body) != `{"error":"actual first"}` || calls != 1 || hooks != 1 {
					t.Fatal(row, err, calls, hooks, actual)
				}
			})
		}
	}
}

func TestSourceMemberHeaderCoalescesActualValuesWithoutChangingBaseline(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "baseline", true: "Source"}[enabled], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("network", "/neutron/v2.0/")
			cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
				r := policyWire(200, `{"record":{"id":"one"}}`)
				r.Header["If-Match"] = []string{"a", "b"}
				return r, nil
			})
			profile := policyProfile(enabled)
			profile.Headers[0].Kind = neutronfind.List
			engine, err := neutronfind.New(context.Background(), client, profile, nil)
			if err != nil {
				t.Fatal(err)
			}
			value, err := engine.Get(context.Background(), "one", neutronfind.Options{})
			if err != nil || value == nil {
				t.Fatal(value, err)
			}
			want := `["a"]`
			if enabled {
				want = `["a, b"]`
			}
			if string(value.Resource.Body["if_match"]) != want || !reflect.DeepEqual(value.Wire.Header["If-Match"], []string{"a", "b"}) {
				t.Fatal(value, want)
			}
			value.Resource.Body["if_match"][2] = 'x'
			value.Header["If-Match"][0] = "changed"
			if !reflect.DeepEqual(value.Wire.Header["If-Match"], []string{"a", "b"}) {
				t.Fatal("projected header aliases actual response", value)
			}
		})
	}
}
func TestSourceHeaderCanonicalCaseIsLiteralAndWireAliasIsInsensitive(t *testing.T) {
	cloud := testcloud.New(t)
	client := cloud.Client("network", "/neutron/v2.0/")
	cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.Query().Get("IF_MATCH") != "extension" || req.URL.Query().Get("IF-MATCH") != "selected" {
			t.Fatal(req.URL)
		}
		return policyWire(200, `{"record":{"id":"one"}}`), nil
	})
	profile := policyProfile(true)
	profile.Headers[0].Kind = neutronfind.List
	engine, err := neutronfind.New(context.Background(), client, profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	value, err := engine.Find(context.Background(), "one", neutronfind.Options{Query: map[string]json.RawMessage{"IF_MATCH": json.RawMessage(`"extension"`), "IF-MATCH": json.RawMessage(`"selected"`)}})
	if err != nil || value == nil || string(value.Resource.Body["IF_MATCH"]) != `"extension"` || string(value.Resource.Body["if_match"]) != `["selected"]` || value.Wire.Body["IF_MATCH"] != nil || value.Wire.Body["if_match"] != nil {
		t.Fatal(value, err)
	}
}
