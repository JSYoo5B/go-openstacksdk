package neutronfind_test

import (
	"context"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"net/http"
	"testing"
)

func TestSourceCollectionGETRequiresExplicitSourceProfileAndKeepsTrailingRoute(t *testing.T) {
	for _, c := range []struct{ source, opt, success bool }{{true, true, true}, {true, false, false}, {false, true, false}} {
		cloud := testcloud.New(t)
		calls := 0
		client := cloud.Client("network", "/v2.0/")
		cloud.SetTransport(testcloud.LinkRequest(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method != "GET" || r.URL.EscapedPath() != "/v2.0/records/" {
				t.Fatal(r.Method, r.URL)
			}
			return policyWire(200, `{"record":{"name":"actual"}}`), nil
		}))
		profile := mutationProfile()
		profile.SourceFindPolicy = c.source
		profile.SourceCollectionGET = c.opt
		engine, err := neutronfind.New(context.Background(), client, profile, nil)
		if err != nil {
			t.Fatal(err)
		}
		v, err := engine.Get(context.Background(), "", neutronfind.Options{})
		if c.success {
			if err != nil || calls != 1 || v == nil || v.Resource == nil || string(v.Resource.Body["id"]) != `""` {
				t.Fatal(v, err, calls)
			}
		} else if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || v != nil {
			t.Fatal(v, err, calls)
		}
	}
}
