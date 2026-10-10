package secrets_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
	"github.com/gophercloud/gophercloud/v2"
)

func TestNativeSecretMetadataRoutesBodiesAndDecode(t *testing.T) {
	cloud := testcloud.New(t)
	type exchange struct{ method, path, body string }
	var seen []exchange
	cloud.Provider.HTTPClient.Transport = nativeSecretTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		seen = append(seen, exchange{req.Method, req.URL.Path, raw})
		switch {
		case req.Method == http.MethodPut && req.URL.Path == "/barbican/v1/secrets/s1/metadata":
			return nativeSecretWire(201, `{"metadata_ref":"https://kms/v1/secrets/s1/metadata"}`), nil
		case req.Method == http.MethodPost:
			return nativeSecretWire(201, `{"key":"ignored"}`), nil
		case req.Method == http.MethodDelete:
			return nativeSecretWire(204, ""), nil
		case req.Method == http.MethodGet && req.URL.Path == "/barbican/v1/secrets/s1/metadata":
			return nativeSecretWire(200, `{"metadata":{"a":"1","b":"2"}}`), nil
		}
		return nativeSecretWire(200, `{"key":"a","value":"1"}`), nil
	})
	api := secrets.New(nativeSecretClient(cloud))
	ctx := context.Background()
	metadata, err := api.GetMetadata(ctx, "s1")
	if err != nil || !reflect.DeepEqual(metadata, map[string]string{"a": "1", "b": "2"}) {
		t.Fatal(metadata, err)
	}
	// CreateMetadata replaces all metadata and returns the whole response map.
	created, err := api.CreateMetadata(ctx, "s1", secrets.MetadataOpts{"a": "1"})
	if err != nil || !reflect.DeepEqual(created, map[string]string{"metadata_ref": "https://kms/v1/secrets/s1/metadata"}) {
		t.Fatal(created, err)
	}
	if _, err := api.CreateMetadata(ctx, "s1", secrets.MetadataOpts{"a": "1"}, secrets.WithCreateMetadataField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}
	datum, err := api.GetMetadatum(ctx, "s1", "a")
	if err != nil || datum.Key != "a" || datum.Value != "1" {
		t.Fatal(datum, err)
	}
	if err := api.CreateMetadatum(ctx, "s1", secrets.MetadatumOpts{Key: "c", Value: "3"}); err != nil {
		t.Fatal(err)
	}
	updated, err := api.UpdateMetadatum(ctx, "s1", secrets.MetadatumOpts{Key: "a", Value: "9"}, secrets.WithUpdateMetadatumField("x_extension", true))
	if err != nil || updated.Key != "a" || updated.Value != "1" {
		t.Fatal("native update returns the decoded response", updated, err)
	}
	if err := api.DeleteMetadatum(ctx, "s1", "a/b"); err != nil {
		t.Fatal(err)
	}
	want := []exchange{
		{http.MethodGet, "/barbican/v1/secrets/s1/metadata", ""},
		{http.MethodPut, "/barbican/v1/secrets/s1/metadata", `{"metadata":{"a":"1"}}`},
		// The typed metadata map is not a generic object envelope, so the
		// shared extension merge adds the field beside "metadata".
		{http.MethodPut, "/barbican/v1/secrets/s1/metadata", `{"metadata":{"a":"1"},"x_extension":1}`},
		{http.MethodGet, "/barbican/v1/secrets/s1/metadata/a", ""},
		{http.MethodPost, "/barbican/v1/secrets/s1/metadata", `{"key":"c","value":"3"}`},
		{http.MethodPut, "/barbican/v1/secrets/s1/metadata/a", `{"key":"a","value":"9","x_extension":true}`},
		{http.MethodDelete, "/barbican/v1/secrets/s1/metadata/a/b", ""},
	}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("%+v", seen)
	}
}

func TestNativeSecretMetadataStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	datum := secrets.MetadatumOpts{Key: "a", Value: "1"}
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*secrets.API) error
	}{
		{"GetMetadata", []int{200}, func(api *secrets.API) error { _, err := api.GetMetadata(ctx, "s1"); return err }},
		{"CreateMetadata", []int{201}, func(api *secrets.API) error {
			_, err := api.CreateMetadata(ctx, "s1", secrets.MetadataOpts{})
			return err
		}},
		{"GetMetadatum", []int{200}, func(api *secrets.API) error { _, err := api.GetMetadatum(ctx, "s1", "a"); return err }},
		{"CreateMetadatum", []int{201}, func(api *secrets.API) error { return api.CreateMetadatum(ctx, "s1", datum) }},
		{"UpdateMetadatum", []int{200}, func(api *secrets.API) error { _, err := api.UpdateMetadatum(ctx, "s1", datum); return err }},
		{"DeleteMetadatum", []int{202, 204}, func(api *secrets.API) error { return api.DeleteMetadatum(ctx, "s1", "a") }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404, 409} {
			if contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				cloud := testcloud.New(t)
				var requests atomic.Int32
				cloud.Provider.HTTPClient.Transport = nativeSecretTransport(func(req *http.Request) (*http.Response, error) {
					requests.Add(1)
					return nativeSecretWire(code, `{"key":"a","value":"1"}`), nil
				})
				err := call.call(secrets.New(nativeSecretClient(cloud)))
				nativeSecretOperation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || requests.Load() != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("required key and value, collisions and nil options", func(t *testing.T) {
		cloud := testcloud.New(t)
		var requests atomic.Int32
		cloud.Provider.HTTPClient.Transport = nativeSecretTransport(func(req *http.Request) (*http.Response, error) {
			requests.Add(1)
			return nativeSecretWire(201, `{}`), nil
		})
		api := secrets.New(nativeSecretClient(cloud))
		checks := map[string]func() error{
			"create missing value": func() error { return api.CreateMetadatum(ctx, "s1", secrets.MetadatumOpts{Key: "a"}) },
			"update missing key": func() error {
				_, err := api.UpdateMetadatum(ctx, "s1", secrets.MetadatumOpts{Value: "1"})
				return err
			},
			"create datum key collision": func() error {
				return api.CreateMetadatum(ctx, "s1", datum, secrets.WithCreateMetadatumField("key", "x"))
			},
			"update datum value collision": func() error {
				_, err := api.UpdateMetadatum(ctx, "s1", datum, secrets.WithUpdateMetadatumField("value", "x"))
				return err
			},
			"metadata envelope collision": func() error {
				_, err := api.CreateMetadata(ctx, "s1", secrets.MetadataOpts{}, secrets.WithCreateMetadataField("metadata", map[string]string{}))
				return err
			},
			"nil option": func() error { return api.CreateMetadatum(ctx, "s1", datum, nil) },
		}
		for name, check := range checks {
			if err := check(); err == nil {
				t.Fatal(name, "accepted")
			}
		}
		if requests.Load() != 0 {
			t.Fatal(requests.Load())
		}
	})
}
