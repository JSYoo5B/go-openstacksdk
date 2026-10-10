package noauth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/noauth"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/noauth"
)

// The constructors make no HTTP request. They take the project from the
// provider token, which native noauth.NewClient sets to "user:project".
func nativeNoAuthAPI(t *testing.T, token string) (*noauth.API, *gophercloud.ProviderClient) {
	t.Helper()
	provider := &gophercloud.ProviderClient{TokenID: token}
	return noauth.New(&gophercloud.ServiceClient{ProviderClient: provider, Endpoint: "https://unused.example/"}), provider
}

func TestNativeNoAuthClientEndpointAndProvider(t *testing.T) {
	ctx := context.Background()
	provider, err := upstream.NewClient(gophercloud.AuthOptions{})
	if err != nil || provider.TokenID != "admin:admin" {
		t.Fatal(provider, err)
	}
	// Both option aliases name request.Option[EndpointOpts].
	for name, create := range map[string]func(*noauth.API, noauth.EndpointOpts, ...noauth.NewBlockStorageNoAuthV2Option) (*gophercloud.ServiceClient, error){
		"v2": func(api *noauth.API, opts noauth.EndpointOpts, options ...noauth.NewBlockStorageNoAuthV2Option) (*gophercloud.ServiceClient, error) {
			return api.NewBlockStorageNoAuthV2(ctx, opts, options...)
		},
		"v3": func(api *noauth.API, opts noauth.EndpointOpts, options ...noauth.NewBlockStorageNoAuthV2Option) (*gophercloud.ServiceClient, error) {
			return api.NewBlockStorageNoAuthV3(ctx, opts, options...)
		},
	} {
		t.Run(name, func(t *testing.T) {
			api, provider := nativeNoAuthAPI(t, "admin:project-1")
			// The endpoint gets a trailing slash and the project from the token appended.
			client, err := create(api, noauth.EndpointOpts{CinderEndpoint: "http://cinder:8776/v3"})
			if err != nil || client.Endpoint != "http://cinder:8776/v3/project-1/" || client.Type != "block-storage" || client.ProviderClient != provider {
				t.Fatal(client, err)
			}
			for input, token := range map[string]string{"missing endpoint": "admin:project-1", "token without colon": "keystone-token", "token with two colons": "a:b:c"} {
				api, _ := nativeNoAuthAPI(t, token)
				endpoint := "http://cinder:8776/v3"
				if input == "missing endpoint" {
					endpoint = ""
				}
				client, err := create(api, noauth.EndpointOpts{CinderEndpoint: endpoint})
				// Native validation errors are returned without operation context.
				var wrapped *resource.OperationError
				if client != nil || err == nil || errors.As(err, &wrapped) {
					t.Fatal(input, client, err)
				}
			}
			api, _ = nativeNoAuthAPI(t, "admin:project-1")
			_, err = create(api, noauth.EndpointOpts{CinderEndpoint: "http://cinder:8776/v3"}, nil)
			var wrapped *resource.OperationError
			if !errors.As(err, &wrapped) || wrapped.Resource != "noauth" || wrapped.Operation != "NewBlockStorageNoAuth"+map[string]string{"v2": "V2", "v3": "V3"}[name] {
				t.Fatal(err)
			}
		})
	}
}
