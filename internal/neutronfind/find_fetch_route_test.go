package neutronfind_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestFindWithFetchControlsDefaultRouteKeepsCanonicalContinuationAlias(t *testing.T) {
	for _, rawNull := range []bool{false, true} {
		t.Run(map[bool]string{false: "omitted", true: "explicit null"}[rawNull], func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			cloud.Provider.HTTPClient.Transport = policyTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					if r.URL.Path != "/neutron/v2.0/records/target" {
						t.Fatal(r.URL)
					}
					return policyWire(404, `{}`), nil
				}
				if r.URL.Path != "/neutron/v2.0/records" {
					t.Fatal("canonical continuation escaped reverse proxy route", r.URL)
				}
				if calls == 2 {
					return policyWire(200, `{"records":[{"id":"one","name":"target"}],"next":"/v2.0/records?name=target&marker=one"}`), nil
				}
				if r.URL.Query().Get("marker") != "one" {
					t.Fatal(r.URL)
				}
				return policyWire(200, `{"records":[]}`), nil
			})
			client := cloud.Client("network", "/catalog/")
			client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
			profile := policyProfile(true)
			profile.SourceLinkPolicy = true
			engine, err := neutronfind.New(context.Background(), client, profile, nil)
			if err != nil {
				t.Fatal(err)
			}
			options := neutronfind.Options{}
			if rawNull {
				options.Query = map[string]json.RawMessage{"base_path": json.RawMessage(`null`)}
			}
			row, err := engine.FindWithFetchControls(context.Background(), "target", options)
			if err != nil || row == nil || calls != 3 || string(row.Resource.Body["id"]) != `"one"` {
				t.Fatal(row, err, calls)
			}
		})
	}
}

func TestFindWithFetchControlsExplicitRouteKeepsContinuationInsideSelectedCollection(t *testing.T) {
	for _, tc := range []struct {
		name, next string
		accept     bool
	}{
		{"selected route", "/neutron/v2.0/custom-records?name=target&marker=one", true},
		{"original canonical route", "/v2.0/records?name=target&marker=one", false},
		{"foreign origin", "https://foreign.invalid/neutron/v2.0/custom-records?name=target&marker=one", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			body := `{"records":[{"id":"one","name":"target"}],"next":"` + tc.next + `"}`
			cloud.Provider.HTTPClient.Transport = policyTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					if r.URL.Path != "/neutron/v2.0/custom-records/target" {
						t.Fatal(r.URL)
					}
					return policyWire(404, `{}`), nil
				}
				if r.URL.Path != "/neutron/v2.0/custom-records" {
					t.Fatal(r.URL)
				}
				if calls == 2 {
					response := policyWire(200, body)
					response.Header.Set("X-Request-ID", "custom-route-proof")
					return response, nil
				}
				if r.URL.Query().Get("marker") != "one" {
					t.Fatal(r.URL)
				}
				return policyWire(200, `{"records":[]}`), nil
			})
			client := cloud.Client("network", "/catalog/")
			client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
			profile := policyProfile(true)
			profile.SourceLinkPolicy = true
			engine, err := neutronfind.New(context.Background(), client, profile, nil)
			if err != nil {
				t.Fatal(err)
			}
			row, err := engine.FindWithFetchControls(context.Background(), "target", fetchControlsQuery(map[string]string{"base_path": `"custom-records"`}))
			if tc.accept {
				if err != nil || row == nil || calls != 3 {
					t.Fatal(row, err, calls)
				}
				return
			}
			var proof *resource.ResponseError
			if row != nil || calls != 2 || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || proof.Header.Get("X-Request-ID") != "custom-route-proof" {
				t.Fatal(row, err, calls, proof)
			}
		})
	}
}

func TestFindWithFetchControlsKeepsEscapedParentAsConcreteRoute(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	cloud.Provider.HTTPClient.Transport = policyTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		want := "/v2.0/parents/vpn%2Fparent/records"
		if calls == 1 {
			want += "/target"
		}
		if r.URL.EscapedPath() != want {
			t.Fatal(r.URL)
		}
		if calls == 1 {
			return policyWire(404, `{}`), nil
		}
		return policyWire(200, `{"records":[{"id":"target"}]}`), nil
	})
	profile := neutronfind.Profile{ServiceRole: "network", Kind: "records", Path: "parents/vpn%2Fparent/records", Singular: "record", Plural: "records", IDKey: "id", NameKey: "name", SourceFindPolicy: true, Body: []neutronfind.Descriptor{{Name: "id", Wire: "id", Missing: "null"}, {Name: "name", Wire: "name", Missing: "null"}}}
	engine, err := neutronfind.New(context.Background(), cloud.Client("network", "/v2.0"), profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	row, err := engine.FindWithFetchControls(context.Background(), "target", neutronfind.Options{})
	if err != nil || row == nil || calls != 2 {
		t.Fatal(row, err, calls)
	}
}
