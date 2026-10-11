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
