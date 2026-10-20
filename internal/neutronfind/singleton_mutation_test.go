package neutronfind_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func singletonMutationEngine(t *testing.T, source, singleton bool, reply func(*http.Request) (*http.Response, error)) *neutronfind.Engine {
	t.Helper()
	cloud := testcloud.New(t)
	client := cloud.Client("network", "/neutron/v2.0/")
	cloud.Provider.HTTPClient.Transport = policyTransport(reply)
	profile := mutationProfile()
	profile.SourceFindPolicy = source
	profile.SourceSingletonMutation = singleton
	engine, err := neutronfind.New(context.Background(), client, profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	return engine
}

func TestSourceSingletonMutationOwnsFixedRouteWrapperAndSeed(t *testing.T) {
	t.Run("singleton GET keeps fixed route", func(t *testing.T) {
		cloud := testcloud.New(t)
		calls := 0
		const raw = `{"record":{"id":"singleton","name":"actual"}}`
		cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Method != http.MethodGet || req.URL.EscapedPath() != "/neutron/v2.0/records" || req.URL.RawQuery != "" || req.Body != nil {
				t.Fatal(req.Method, req.URL, req.Body)
			}
			return mutationResponse(203, raw), nil
		})
		profile := mutationProfile()
		profile.SourceCollectionGET = true
		profile.SourceSingletonMutation = true
		engine, err := neutronfind.New(context.Background(), cloud.Client("network", "/neutron/v2.0/"), profile, nil)
		if err != nil {
			t.Fatal(err)
		}
		row, err := engine.Get(context.Background(), "", neutronfind.Options{})
		if err != nil || row == nil || row.Resource == nil || row.Wire == nil || string(row.Resource.Body["id"]) != `"singleton"` || string(row.Envelope) != raw || row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "mutation-actual" || calls != 1 {
			t.Fatal(row, err, calls)
		}
	})
	for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			calls := 0
			const raw = `{"record":{"name":"actual","shared":false},"physical":"whole"}`
			engine := singletonMutationEngine(t, true, true, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != method || req.URL.EscapedPath() != "/neutron/v2.0/records" || req.URL.RawQuery != "" || mutationRequestBody(t, req) != `{"record":{"name":"owned"}}` {
					t.Fatal(req.Method, req.URL)
				}
				return mutationResponse(203, raw), nil
			})
			row, err := engine.Mutate(context.Background(), method, "", map[string]json.RawMessage{"name": json.RawMessage(`"owned"`)}, map[string]json.RawMessage{"id": json.RawMessage(`"seed"`), "tenant_id": json.RawMessage(`"owner"`)}, neutronfind.Options{})
			if err != nil || row == nil || row.Resource == nil || row.Wire == nil || calls != 1 || row.StatusCode != 203 || string(row.Envelope) != raw || row.Header.Get("X-Request-Id") != "mutation-actual" {
				t.Fatal(row, err, calls)
			}
			for key, want := range map[string]string{"id": `"seed"`, "name": `"actual"`, "project_id": `"owner"`, "is_shared": `false`, "tags": `[]`} {
				if string(row.Resource.Body[key]) != want {
					t.Fatal(key, row.Resource.Body)
				}
			}
			if row.Wire.Body["id"] != nil || row.Wire.Body["tenant_id"] != nil {
				t.Fatal(row.Wire)
			}
		})
	}
}

func TestSourceSingletonMutationRequiresBothProfileFlagsAndKeepsMembers(t *testing.T) {
	for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete} {
		for _, flags := range []struct{ source, singleton bool }{{false, false}, {false, true}, {true, false}} {
			t.Run(fmt.Sprintf("%s/source=%t/singleton=%t", method, flags.source, flags.singleton), func(t *testing.T) {
				calls := 0
				engine := singletonMutationEngine(t, flags.source, flags.singleton, func(*http.Request) (*http.Response, error) { calls++; return mutationResponse(200, `{}`), nil })
				row, err := engine.Mutate(context.Background(), method, "", nil, nil, neutronfind.Options{})
				if row != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
					t.Fatal(row, err, calls)
				}
			})
		}
	}
	t.Run("ordinary member still escaped", func(t *testing.T) {
		calls := 0
		engine := singletonMutationEngine(t, true, true, func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Method != http.MethodPatch || req.URL.EscapedPath() != "/neutron/v2.0/records/folder%2Fname%25" {
				t.Fatal(req.Method, req.URL)
			}
			return mutationResponse(200, `{"record":{"id":"actual"}}`), nil
		})
		row, err := engine.Mutate(context.Background(), http.MethodPatch, "folder/name%", nil, nil, neutronfind.Options{})
		if err != nil || row == nil || calls != 1 {
			t.Fatal(row, err, calls)
		}
	})
	for _, method := range []string{http.MethodDelete, http.MethodPost} {
		t.Run("preflight/"+method, func(t *testing.T) {
			calls := 0
			engine := singletonMutationEngine(t, true, true, func(*http.Request) (*http.Response, error) { calls++; return mutationResponse(200, `{}`), nil })
			id := ""
			body := map[string]json.RawMessage{"name": json.RawMessage(`"forbidden"`)}
			if method == http.MethodPost {
				id = "member"
			}
			row, err := engine.Mutate(context.Background(), method, id, body, nil, neutronfind.Options{})
			if row != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(row, err, calls)
			}
		})
	}
}

func TestSourceSingletonDeleteKeepsBodylessReceiptAndPhysical404(t *testing.T) {
	t.Run("raw bodyless receipt", func(t *testing.T) {
		const raw = "\xffnot JSON"
		calls := 0
		engine := singletonMutationEngine(t, true, true, func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Method != http.MethodDelete || req.URL.EscapedPath() != "/neutron/v2.0/records" || req.URL.RawQuery != "" || req.Body != nil {
				t.Fatal(req.Method, req.URL, req.Body)
			}
			return mutationResponse(200, raw), nil
		})
		row, err := engine.Mutate(context.Background(), http.MethodDelete, "", nil, map[string]json.RawMessage{"name": json.RawMessage(`"not a receipt"`)}, neutronfind.Options{})
		if err != nil || row == nil || row.Resource != nil || row.Wire != nil || row.StatusCode != 200 || string(row.Envelope) != raw || row.Header.Get("X-Request-Id") != "mutation-actual" || calls != 1 {
			t.Fatal(row, err, calls)
		}
	})
	for _, fault := range []bool{false, true} {
		t.Run(fmt.Sprintf("404/physical=%t", fault), func(t *testing.T) {
			const raw = `{"error":"actual singleton missing"}`
			readCause, closeCause := errors.New("singleton read"), errors.New("singleton close")
			done := false
			physical := &mutationFaultBody{Reader: mutationFaultReader(func(p []byte) (int, error) {
				if done {
					return 0, io.EOF
				}
				done = true
				return copy(p, raw), readCause
			}), closeCause: closeCause}
			calls := 0
			engine := singletonMutationEngine(t, true, true, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodDelete || req.URL.EscapedPath() != "/neutron/v2.0/records" || req.Body != nil {
					t.Fatal(req.Method, req.URL, req.Body)
				}
				response := mutationResponse(404, raw)
				if fault {
					response.Body = physical
				}
				return response, nil
			})
			row, err := engine.Mutate(context.Background(), http.MethodDelete, "", nil, nil, neutronfind.Options{})
			var native gophercloud.ErrUnexpectedResponseCode
			if row != nil || !errors.As(err, &native) || native.Actual != 404 || native.Method != http.MethodDelete || string(native.Body) != raw || native.ResponseHeader.Get("X-Request-Id") != "mutation-actual" || calls != 1 {
				t.Fatal(row, err, native, calls)
			}
			if fault && (!errors.Is(err, readCause) || !errors.Is(err, closeCause) || physical.closed != 1) {
				t.Fatal(err, physical.closed)
			}
		})
	}
}
