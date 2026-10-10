package swauth_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/swauth"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeSwauthCall struct {
	method, path, user, key, token, extra string
}

func nativeSwauthAPI(t *testing.T, calls *[]nativeSwauthCall, code int) (*swauth.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.IdentityBase = cloud.Server.URL + "/"
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		*calls = append(*calls, nativeSwauthCall{r.Method, r.URL.Path, r.Header.Get("X-Auth-User"), r.Header.Get("X-Auth-Key"), r.Header.Get("X-Auth-Token"), r.Header.Get("X-Extra")})
		w.Header().Set("X-Auth-Token", "swift-token")
		w.Header().Set("X-Storage-Url", "https://swift.example/v1/AUTH_acct")
		w.Header().Set("X-CDN-Management-Url", "https://cdn.example")
		w.WriteHeader(code)
	})
	return swauth.New(cloud.Client("object-store", "/unused/")), cloud
}

func nativeSwauthOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "swauth" {
		t.Fatal("generated swauth context", err, wrapped)
	}
}

// Auth is a GET to the provider identity base, not the object-store endpoint,
// and reads the result only from response headers.
func TestNativeSwauthAuthHeadersAndResult(t *testing.T) {
	ctx := context.Background()
	var calls []nativeSwauthCall
	api, _ := nativeSwauthAPI(t, &calls, 200)
	result, err := api.Auth(ctx, swauth.AuthOpts{User: "acct:user", Key: "secret"}, swauth.WithAuthHeader("X-Extra", "1"))
	want := swauth.AuthResult{Token: "swift-token", StorageURL: "https://swift.example/v1/AUTH_acct", CDNURL: "https://cdn.example"}
	if err != nil || result == nil || *result != want {
		t.Fatal(result, err)
	}
	// The provider token is still sent with the swauth credentials.
	if !reflect.DeepEqual(calls, []nativeSwauthCall{{http.MethodGet, "/auth/v1.0", "acct:user", "secret", "test-token", "1"}}) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeSwauthStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, code := range []int{201, 202, 204, 401, 404} {
		var calls []nativeSwauthCall
		api, _ := nativeSwauthAPI(t, &calls, code)
		_, err := api.Auth(ctx, swauth.AuthOpts{User: "u", Key: "k"})
		nativeSwauthOperation(t, err, "Auth")
		var native gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || len(calls) != 1 {
			t.Fatal(code, err)
		}
	}
	var calls []nativeSwauthCall
	api, _ := nativeSwauthAPI(t, &calls, 200)
	for name, check := range map[string]error{
		"missing user": func() error { _, err := api.Auth(ctx, swauth.AuthOpts{Key: "k"}); return err }(),
		"missing key":  func() error { _, err := api.Auth(ctx, swauth.AuthOpts{User: "u"}); return err }(),
		"core header": func() error {
			_, err := api.Auth(ctx, swauth.AuthOpts{User: "u", Key: "k"}, swauth.WithAuthHeader("X-Auth-Key", "other"))
			return err
		}(),
		"nil option": func() error { _, err := api.Auth(ctx, swauth.AuthOpts{User: "u", Key: "k"}, nil); return err }(),
	} {
		if check == nil {
			t.Fatal(name, "accepted")
		}
		nativeSwauthOperation(t, check, "Auth")
	}
	if len(calls) != 0 {
		t.Fatal(calls)
	}
}

// NewObjectStorageV1 writes the swauth token into the shared ProviderClient,
// so every client built on that provider switches to the swauth token.
func TestNativeSwauthNewObjectStorageV1SharesProviderToken(t *testing.T) {
	ctx := context.Background()
	var calls []nativeSwauthCall
	api, cloud := nativeSwauthAPI(t, &calls, 200)
	client, err := api.NewObjectStorageV1(ctx, swauth.AuthOpts{User: "u", Key: "k"})
	if err != nil || client.Endpoint != "https://swift.example/v1/AUTH_acct/" || client.ProviderClient != cloud.Provider || client.Type != "" {
		t.Fatal(client, err)
	}
	if cloud.Provider.Token() != "swift-token" || len(calls) != 1 {
		t.Fatal(cloud.Provider.Token(), calls)
	}
	t.Run("errors are not wrapped", func(t *testing.T) {
		var calls []nativeSwauthCall
		api, cloud := nativeSwauthAPI(t, &calls, 401)
		client, err := api.NewObjectStorageV1(ctx, swauth.AuthOpts{User: "u", Key: "k"})
		var native gophercloud.ErrUnexpectedResponseCode
		var wrapped *resource.OperationError
		if client != nil || !errors.As(err, &native) || native.Actual != 401 || errors.As(err, &wrapped) || cloud.Provider.Token() != "test-token" {
			t.Fatal(client, err)
		}
		// Required headers are checked inside native Auth, again without operation context.
		client, err = api.NewObjectStorageV1(ctx, swauth.AuthOpts{User: "u"})
		if client != nil || err == nil || errors.As(err, &wrapped) || len(calls) != 1 {
			t.Fatal(client, err, calls)
		}
		_, err = api.NewObjectStorageV1(ctx, swauth.AuthOpts{User: "u", Key: "k"}, nil)
		nativeSwauthOperation(t, err, "NewObjectStorageV1")
	})
}
