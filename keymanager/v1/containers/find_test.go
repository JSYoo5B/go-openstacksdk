package containers_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func findContainerAPI(t *testing.T, calls *atomic.Int32, paths *[]string, reply func(n int32, req *http.Request) *http.Response) *containers.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeContainerTransport(func(req *http.Request) (*http.Response, error) {
		n := calls.Add(1)
		*paths = append(*paths, req.URL.Path+"?"+req.URL.RawQuery)
		return reply(n, req), nil
	})
	return containers.New(nativeContainerClient(cloud))
}

func TestContainerFindIdentityFollowsPythonFind(t *testing.T) {
	ctx := context.Background()
	t.Run("direct GET wins", func(t *testing.T) {
		var calls atomic.Int32
		var paths []string
		api := findContainerAPI(t, &calls, &paths, func(int32, *http.Request) *http.Response {
			return nativeContainerWire(200, `{"name":"web","container_ref":"https://kms/v1/containers/c1"}`)
		})
		found, err := api.FindIdentity(ctx, "c1")
		if err != nil || found.RequestID != "c1" || found.ContainerID == nil || *found.ContainerID != "c1" || string(found.Resource.Body["id"]) != `"c1"` || calls.Load() != 1 || paths[0] != "/barbican/v1/containers/c1?" {
			t.Fatal(found, err, paths)
		}
	})
	for _, code := range []int{400, 403, 404} {
		t.Run("fallback list by name after "+http.StatusText(code), func(t *testing.T) {
			var calls atomic.Int32
			var paths []string
			api := findContainerAPI(t, &calls, &paths, func(n int32, req *http.Request) *http.Response {
				switch n {
				case 1:
					return nativeContainerWire(code, `{}`)
				case 2:
					return nativeContainerWire(200, `{"containers":[{"name":"other","container_ref":"https://kms/v1/containers/c0"}],"next":"`+"http://"+req.Host+`/barbican/v1/containers?offset=1&limit=1"}`)
				}
				return nativeContainerWire(200, `{"containers":[{"name":"web","container_ref":"https://kms/v1/containers/c9"}]}`)
			})
			found, err := api.FindIdentity(ctx, "web")
			// List rows have no request seed: id is the full alternate reference.
			if err != nil || found == nil || string(found.Resource.Body["id"]) != `"https://kms/v1/containers/c9"` || found.ContainerID == nil || *found.ContainerID != "c9" || calls.Load() != 3 {
				t.Fatal(found, err, paths)
			}
			if q, _ := url.ParseQuery(paths[1][len("/barbican/v1/containers?"):]); q.Has("name") {
				t.Fatal("containers map no name query", paths)
			}
		})
	}
	t.Run("full reference matches a list row", func(t *testing.T) {
		var calls atomic.Int32
		var paths []string
		// An identity that is not a safe path segment skips the direct GET.
		api := findContainerAPI(t, &calls, &paths, func(n int32, _ *http.Request) *http.Response {
			return nativeContainerWire(200, `{"containers":[{"name":null,"container_ref":"https://kms/v1/containers/c-ref"}]}`)
		})
		found, err := api.FindIdentity(ctx, "https://kms/v1/containers/c-ref")
		if err != nil || found == nil || string(found.Resource.Body["id"]) != `"https://kms/v1/containers/c-ref"` || *found.ContainerID != "c-ref" || calls.Load() != 1 {
			t.Fatal(found, err)
		}
	})
	t.Run("missing, strict missing, duplicates and terminal errors", func(t *testing.T) {
		list := `{"containers":[{"name":"dup","container_ref":"https://kms/v1/containers/r1"},{"name":"dup","container_ref":"https://kms/v1/containers/r2"}]}`
		newAPI := func(direct int) (*containers.API, *atomic.Int32) {
			var calls atomic.Int32
			var paths []string
			return findContainerAPI(t, &calls, &paths, func(n int32, _ *http.Request) *http.Response {
				if n == 1 {
					return nativeContainerWire(direct, `{}`)
				}
				return nativeContainerWire(200, list)
			}), &calls
		}
		api, _ := newAPI(404)
		if found, err := api.FindIdentity(ctx, "absent"); found != nil || err != nil {
			t.Fatal(found, err)
		}
		api, _ = newAPI(404)
		if found, err := api.FindIdentity(ctx, "absent", resource.WithIdentityFindIgnoreMissing(false)); found != nil || !errors.Is(err, resource.ErrNotFound) {
			t.Fatal(found, err)
		}
		api, _ = newAPI(404)
		if found, err := api.FindIdentity(ctx, "dup"); found != nil || !errors.Is(err, resource.ErrAmbiguous) {
			t.Fatal(found, err)
		}
		api, calls := newAPI(500)
		if found, err := api.FindIdentity(ctx, "dup"); found != nil || !gophercloud.ResponseCodeIs(err, http.StatusInternalServerError) || calls.Load() != 1 {
			t.Fatal(found, err)
		}
		api, calls = newAPI(200)
		for _, identity := range []string{""} {
			if _, err := api.FindIdentity(ctx, identity); err == nil || calls.Load() != 0 {
				t.Fatal(identity, err)
			}
		}
		if _, err := api.FindIdentity(ctx, "c1", resource.WithIdentityFindOptions(resource.IdentityFindOpts{Query: url.Values{"name": {"x"}}})); !errors.Is(err, resource.ErrUnsupported) || calls.Load() != 0 {
			t.Fatal(err)
		}
	})
}
