package neutronfind

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestSourceDescriptorListDictionaryCastPreservesNullAndWireKinds(t *testing.T) {
	e := &Engine{profile: Profile{SourceFindPolicy: true}}
	for _, tc := range []struct{ input, want string }{
		{`null`, `null`}, {`[null,false,0,"x",[],{"href":"x"}]`, `[{},{},{},{},{},{"href":"x"}]`},
		{`{"href":"x"}`, `[{"href":"x"}]`}, {`0`, `[{}]`}, {`[]`, `[]`},
	} {
		got, err := e.descriptorValue(json.RawMessage(tc.input), Descriptor{Kind: List, ListItemKind: Dictionary})
		if err != nil || string(got) != tc.want {
			t.Fatalf("Source list_type=dict %s -> %s, %v", tc.input, got, err)
		}
	}
	got, err := e.descriptorValue(json.RawMessage(`[null,0]`), Descriptor{Kind: List})
	if err != nil || string(got) != `[null,0]` {
		t.Fatal("untyped baseline changed", string(got), err)
	}
}

func sourceProfile() Profile {
	return Profile{Kind: "widgets", Path: "widgets", Singular: "widget", Plural: "widgets", IDKey: "id", NameKey: "name", SourceFindPolicy: true,
		Body: []Descriptor{{Name: "id", Wire: "id", Missing: "null"}, {Name: "name", Wire: "name", Missing: "null"}}}
}

func TestSourceProfileServiceRoleOwnsBothNovaVersionHeaders(t *testing.T) {
	fixture := testcloud.New(t)
	client := fixture.Client("compute", "/v2.1/p/")
	client.Microversion = "2.97"
	client.MoreHeaders = map[string]string{"OpenStack-API-Version": "compute 2.97", "X-OpenStack-Nova-API-Version": "2.97"}
	profile := sourceProfile()
	profile.ServiceRole = "compute"
	fixture.Mux.HandleFunc("GET /v2.1/p/widgets/a", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("OpenStack-API-Version") != "compute 2.110" || r.Header.Get("X-OpenStack-Nova-API-Version") != "2.110" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 203, `{"widget":{"id":"a"}}`)
	})
	e, err := New(context.Background(), client, profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	version := "2.110"
	row, err := e.Get(context.Background(), "a", Options{Microversion: &version})
	if err != nil || row == nil || row.StatusCode != 203 || client.Microversion != "2.97" || client.MoreHeaders["X-OpenStack-Nova-API-Version"] != "2.97" {
		t.Fatal(row, err, client)
	}
}

func TestSourceFindFallbackPathDoesNotChangeSuccessfulGetOrOriginalRoute(t *testing.T) {
	fixture := testcloud.New(t)
	client := fixture.Client("network", "/v2.0/")
	profile := sourceProfile()
	calls := 0
	fixture.Mux.HandleFunc("GET /v2.0/widgets/direct", func(w http.ResponseWriter, r *http.Request) {
		calls++
		testcloud.JSON(w, 200, `{"widget":{"id":"direct"}}`)
	})
	fixture.Mux.HandleFunc("GET /v2.0/widgets/name", func(w http.ResponseWriter, r *http.Request) { calls++; testcloud.JSON(w, 404, `{"error":"missing"}`) })
	fixture.Mux.HandleFunc("GET /v2.0/alternate", func(w http.ResponseWriter, r *http.Request) {
		calls++
		testcloud.JSON(w, 200, `{"widgets":[{"id":"a","name":"name"}]}`)
	})
	e, err := New(context.Background(), client, profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	unsafe := "https://foreign.invalid/widgets"
	row, err := e.Find(context.Background(), "direct", Options{ListBasePath: &unsafe})
	if err != nil || row == nil || calls != 1 {
		t.Fatal(row, err, calls)
	}
	path := "alternate"
	opts := CloneOptions(Options{ListBasePath: &path})
	path = "changed"
	row, err = e.Find(context.Background(), "name", opts)
	if err != nil || row == nil || string(row.Resource.Body["id"]) != `"a"` || calls != 3 || e.profile.Path != "widgets" {
		t.Fatal(row, err, calls, e.profile.Path)
	}
}

func TestSourceDescriptorProjectionAndMatchRunBeforeLocalSelectionWithoutAliasing(t *testing.T) {
	fixture := testcloud.New(t)
	client := fixture.Client("network", "/v2.0/")
	profile := sourceProfile()
	calls := 0
	projected := json.RawMessage(`[{"logical":true}]`)
	profile.Body = append(profile.Body, Descriptor{Name: "events", Wire: "events", Missing: "null", Project: func(raw json.RawMessage) (json.RawMessage, error) {
		if string(raw) != `[{"physical":true}]` {
			t.Error(string(raw))
		}
		raw[0] = '!'
		return projected, nil
	}, Match: func(actual, wire, filter json.RawMessage) (bool, bool, error) {
		calls++
		if string(actual) != `[{"logical":true}]` || string(wire) != `[{"physical":true}]` {
			t.Error(string(actual), string(wire))
		}
		actual[0] = '!'
		wire[0] = '!'
		filter[0] = '!'
		return true, true, nil
	}})
	fixture.Mux.HandleFunc("GET /v2.0/widgets", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"widgets":[{"id":"a","events":[{"physical":true}]}]}`)
	})
	e, err := New(context.Background(), client, profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	filter := json.RawMessage(`[]`)
	for row, err := range e.List(context.Background(), Options{Query: map[string]json.RawMessage{"events": filter}, Order: []string{"events"}}) {
		if err != nil {
			t.Fatal(err)
		}
		projected[0] = '!'
		if string(row.Resource.Body["events"]) != `[{"logical":true}]` || string(row.Wire.Body["events"]) != `[{"physical":true}]` || string(filter) != `[]` {
			t.Fatal(row, string(filter))
		}
	}
	if calls != 1 {
		t.Fatal(calls)
	}
}

func TestSourceDescriptorProjectionFaultRetainsActualPageAndStopsSelection(t *testing.T) {
	fixture := testcloud.New(t)
	client := fixture.Client("network", "/v2.0/")
	profile := sourceProfile()
	cause := errors.New("projection sentinel")
	profile.Body = append(profile.Body, Descriptor{Name: "events", Wire: "events", Missing: "null", Project: func(json.RawMessage) (json.RawMessage, error) { return nil, cause }})
	fixture.Mux.HandleFunc("GET /v2.0/widgets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Proof", "actual")
		testcloud.JSON(w, 203, `{"widgets":[{"id":"a","events":[]}]}`)
	})
	e, err := New(context.Background(), client, profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	for row, err := range e.List(context.Background(), Options{}) {
		if row != nil || !errors.Is(err, cause) {
			t.Fatal(row, err)
		}
	}
	if _, err := e.PostCollection(nil, nil, Options{}); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
