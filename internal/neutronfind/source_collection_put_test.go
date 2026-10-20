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

func TestSourceCollectionPUTRequiresExplicitProfileAndOwnsOnlyFixedRoute(t *testing.T) {
	for _, scenario := range []struct {
		name        string
		source, opt bool
		method, id  string
		accepted    bool
	}{
		{"source opted in", true, true, http.MethodPut, "", true},
		{"baseline", true, false, http.MethodPut, "", false},
		{"non source profile", false, true, http.MethodPut, "", false},
		{"delete still needs member", true, true, http.MethodDelete, "", false},
		{"nonempty member", true, true, http.MethodPut, "member", true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("network", "/neutron/v2.0/")
			calls := 0
			cloud.Provider.HTTPClient.Transport = policyTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				want := "/neutron/v2.0/records/"
				if scenario.id != "" {
					want += scenario.id
				}
				if req.Method != http.MethodPut || req.URL.EscapedPath() != want || req.URL.RawQuery != "" || mutationRequestBody(t, req) != `{"record":{"name":"owned"}}` {
					t.Fatal(req.Method, req.URL)
				}
				return mutationResponse(200, `{"record":{"id":"actual","name":"owned"}}`), nil
			})
			profile := mutationProfile()
			profile.SourceFindPolicy = scenario.source
			profile.SourceCollectionPUT = scenario.opt
			engine, err := neutronfind.New(context.Background(), client, profile, nil)
			if err != nil {
				t.Fatal(err)
			}
			value, err := engine.Mutate(context.Background(), scenario.method, scenario.id, map[string]json.RawMessage{"name": json.RawMessage(`"owned"`)}, nil, neutronfind.Options{})
			if scenario.accepted {
				if err != nil || calls != 1 || value == nil || value.StatusCode != 200 {
					t.Fatal(value, err, calls)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || value != nil {
				t.Fatal(value, err, calls)
			}
		})
	}
}
