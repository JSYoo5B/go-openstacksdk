package neutronfind_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

type policyTransport func(*http.Request) (*http.Response, error)

func (f policyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := f(req)
	if response != nil {
		response.Request = req
	}
	return response, err
}
func policyWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}
func policyProfile(source bool) neutronfind.Profile {
	return neutronfind.Profile{Kind: "records", Path: "records", Singular: "record", Plural: "records", IDKey: "id", NameKey: "name", PreserveUnknownBody: true, NameQueryHint: true, VersionedPath: "/v2.0/records", SourceFindPolicy: source,
		Headers: []neutronfind.Descriptor{{Name: "if_match", Wire: "if-match", Missing: "null"}},
		Body:    []neutronfind.Descriptor{{Name: "id", Wire: "id", Missing: "null"}, {Name: "name", Wire: "name", Missing: "null"}, {Name: "is_shared", Wire: "shared", Kind: neutronfind.Boolean, Missing: "null"}, {Name: "tenant_id", Wire: "tenant_id", Missing: "null"}},
		Query:   []neutronfind.QueryPair{{Name: "name", Wire: "name"}, {Name: "is_shared", Wire: "shared"}},
	}
}
func TestSourceProfileOwnsOrderedHeaderAndBodySeedWithoutChangingBaseline(t *testing.T) {
	for _, source := range []bool{false, true} {
		t.Run(map[bool]string{false: "baseline", true: "Source"}[source], func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) { return policyWire(200, `{"record":{}}`), nil })
			client := cloud.Client("network", "/catalog/")
			client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
			engine, err := neutronfind.New(context.Background(), client, policyProfile(source), nil)
			if err != nil {
				t.Fatal(err)
			}
			value, err := engine.Find(context.Background(), "target", neutronfind.Options{Order: []string{"if_match", "if-match", "is_shared", "shared"}, Query: map[string]json.RawMessage{"if_match": json.RawMessage(`"first"`), "if-match": json.RawMessage(`"last"`), "is_shared": json.RawMessage(`false`), "shared": json.RawMessage(`true`)}})
			header, body := `"first"`, `false`
			if source {
				header, body = `"last"`, `true`
			}
			if err != nil || value == nil || string(value.Resource.Body["if_match"]) != header || string(value.Resource.Body["is_shared"]) != body {
				t.Fatal(value, err, header, body)
			}
			if value.Wire == nil || len(value.Wire.Body) != 0 {
				t.Fatal(value)
			}
		})
	}
}
func TestSourceProfileOwnsFloatTruthinessAndFindVersionPhasesWithoutChangingBaseline(t *testing.T) {
	for _, source := range []bool{false, true} {
		t.Run(map[bool]string{false: "baseline", true: "Source"}[source], func(t *testing.T) {
			cloud := testcloud.New(t)
			var versions []string
			calls := 0
			cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				versions = append(versions, req.Header.Get("OpenStack-API-Version"))
				if calls == 1 {
					return policyWire(404, `{"error":"fallback"}`), nil
				}
				return policyWire(200, `{"records":[{"id":"match","name":"target","shared":1e-4000}]}`), nil
			})
			client := cloud.Client("network", "/catalog/")
			client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
			client.Microversion = "2.0"
			engine, err := neutronfind.New(context.Background(), client, policyProfile(source), nil)
			if err != nil {
				t.Fatal(err)
			}
			version := "2.1"
			value, err := engine.Find(context.Background(), "target", neutronfind.Options{Microversion: &version})
			want := []string{"network 2.1", "network 2.1"}
			boolean := `true`
			if source {
				want[1] = "network 2.0"
				boolean = `false`
			}
			if err != nil || value == nil || !reflect.DeepEqual(versions, want) || string(value.Resource.Body["is_shared"]) != boolean || client.Microversion != "2.0" {
				t.Fatal(value, err, versions, want, client)
			}
		})
	}
}

func TestSourceProfileNumericNextTruthinessDoesNotChangeBaselinePager(t *testing.T) {
	for _, source := range []bool{false, true} {
		t.Run(map[bool]string{false: "baseline", true: "Source"}[source], func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			body := `{"records":[{"id":"one"}],"next":1e-4000}`
			cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) { calls++; return policyWire(200, body), nil })
			client := cloud.Client("network", "/catalog/")
			client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
			engine, err := neutronfind.New(context.Background(), client, policyProfile(source), nil)
			if err != nil {
				t.Fatal(err)
			}
			rows := 0
			var final error
			for value, cause := range engine.List(context.Background(), neutronfind.Options{}) {
				if cause != nil {
					final = cause
					break
				}
				if value == nil || string(value.Envelope) != body {
					t.Fatal(value)
				}
				rows++
			}
			if calls != 1 || rows != 1 || (final == nil) != source {
				t.Fatal(calls, rows, source, final)
			}
		})
	}
}
