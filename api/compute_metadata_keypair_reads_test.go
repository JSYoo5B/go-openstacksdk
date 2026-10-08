package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestComputeServerMetadataReadNativeContract(t *testing.T) {
	// Python fetch_server_metadata and its deprecated get_server_metadata alias
	// share this one stateless Go string-map projection and physical GET.
	for _, check := range []struct {
		name, body           string
		status               int
		want                 map[string]string
		wantError, typeError bool
	}{
		{"unicode strings", `{"metadata":{"한글":"값🙂","empty":""}}`, 200, map[string]string{"한글": "값🙂", "empty": ""}, false, false},
		{"empty map", `{"metadata":{}}`, 200, map[string]string{}, false, false},
		{"null metadata", `{"metadata":null}`, 200, nil, false, false},
		{"missing metadata", `{}`, 200, nil, false, false},
		{"null member native empty string", `{"metadata":{"blank":null}}`, 200, map[string]string{"blank": ""}, false, false},
		{"malformed JSON", `{`, 200, nil, true, false},
		{"wrong metadata shape", `{"metadata":[]}`, 200, nil, true, true},
		{"nonstring member native partial", `{"metadata":{"good":"kept","wrong":7}}`, 200, map[string]string{"good": "kept", "wrong": ""}, true, true},
		{"forbidden", `{"error":{"message":"metadata denied"}}`, 403, nil, true, false},
		{"missing server", `{"error":{"message":"server missing"}}`, 404, nil, true, false},
		{"unexpected success code", `{"metadata":{"ignored":"203"}}`, 203, nil, true, false},
	} {
		t.Run(check.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, other atomic.Int32
			const path = "/reverse/nova/v2.1/servers/server-id/metadata"
			cloud.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "metadata-live")
				th.TestHeader(t, r, "X-Source", "selected")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.55")
				if r.URL.RawQuery != "" || r.ContentLength != 0 {
					t.Error("metadata GET gained query/body", r.URL, r.ContentLength)
				}
				w.Header().Set("X-Read-Proof", "metadata actual")
				testcloud.JSON(w, check.status, check.body)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				other.Add(1)
				t.Error("metadata read performed unrelated request", r.Method, r.URL)
				w.WriteHeader(500)
			})
			client := cloud.Client("compute", "/unused/catalog")
			client.ResourceBase = cloud.Server.URL + "/reverse/nova/v2.1/"
			client.Microversion = "2.55"
			client.MoreHeaders = map[string]string{"X-Source": "selected"}
			cloud.Provider.SetToken("metadata-live")
			value, err := servers.New(client).Metadata(context.Background(), "server-id")
			if !reflect.DeepEqual(value, check.want) || (err != nil) != check.wantError || gets.Load() != 1 || other.Load() != 0 {
				t.Fatal(value, err, gets.Load(), other.Load(), check.want)
			}
			if check.status != 200 {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != check.status || !reflect.DeepEqual(native.Expected, []int{200}) || native.Method != http.MethodGet || native.URL != cloud.Server.URL+path || string(native.Body) != check.body || native.ResponseHeader.Get("X-Read-Proof") != "metadata actual" {
					t.Fatal("native metadata HTTP evidence lost", err, native)
				}
			} else if check.typeError {
				var typed *json.UnmarshalTypeError
				if !errors.As(err, &typed) {
					t.Fatal("native map type cause lost", err)
				}
			}
			if client.Microversion != "2.55" || client.ProviderClient != cloud.Provider || client.ResourceBase != cloud.Server.URL+"/reverse/nova/v2.1/" {
				t.Fatal("read changed selected client", client)
			}
		})
	}
}

func TestComputeKeypairGetNativeContract(t *testing.T) {
	full := &keypairs.KeyPair{Name: "wire-name", Fingerprint: "fingerprint", PublicKey: "ssh-ed25519 public", PrivateKey: "private material", UserID: "response-owner", Type: "x509"}
	for _, check := range []struct {
		name, body, owner    string
		status               int
		explicitOptions      bool
		want                 *keypairs.KeyPair
		wantError, typeError bool
	}{
		{"default current user", `{"keypair":{"id":99,"name":"wire-name","fingerprint":"fingerprint","public_key":"ssh-ed25519 public","private_key":"private material","user_id":"response-owner","type":"x509","vendor":false}}`, "", 200, false, full, false, false},
		{"explicit owner", `{"keypair":{"name":"wire-name","fingerprint":"fingerprint","public_key":"ssh-ed25519 public","private_key":"private material","user_id":"response-owner","type":"x509"}}`, "owner+team", 200, true, full, false, false},
		{"explicit empty owner omitted", `{"keypair":{"name":"wire-name","type":"ssh"}}`, "", 200, true, &keypairs.KeyPair{Name: "wire-name", Type: "ssh"}, false, false},
		{"missing type native empty", `{"keypair":{"name":"wire-name"}}`, "", 200, false, &keypairs.KeyPair{Name: "wire-name"}, false, false},
		{"null type native empty", `{"keypair":{"name":"wire-name","type":null}}`, "", 200, false, &keypairs.KeyPair{Name: "wire-name"}, false, false},
		{"missing name is not seeded", `{"keypair":{"id":42,"type":"ssh"}}`, "", 200, false, &keypairs.KeyPair{Type: "ssh"}, false, false},
		{"null keypair", `{"keypair":null}`, "", 200, false, nil, false, false},
		{"missing keypair", `{}`, "", 200, false, nil, false, false},
		{"malformed JSON", `{`, "", 200, false, nil, true, false},
		{"nonstring field native partial", `{"keypair":{"name":7,"fingerprint":"retained"}}`, "", 200, false, &keypairs.KeyPair{Fingerprint: "retained"}, true, true},
		{"forbidden owner", `{"error":{"message":"owner denied"}}`, "foreign-owner", 403, true, nil, true, false},
		{"missing name", `{"error":{"message":"keypair missing"}}`, "", 404, false, nil, true, false},
		{"unexpected success code", `{"keypair":{"name":"ignored"}}`, "", 203, false, nil, true, false},
	} {
		t.Run(check.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var gets, other atomic.Int32
			const path = "/reverse/nova/v2.1/os-keypairs/requested-name"
			wantQuery := make(url.Values)
			if check.owner != "" {
				wantQuery.Set("user_id", check.owner)
			}
			cloud.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				th.TestMethod(t, r, http.MethodGet)
				th.TestHeader(t, r, "X-Auth-Token", "keypair-live")
				th.TestHeader(t, r, "X-Source", "selected")
				th.TestHeader(t, r, "X-OpenStack-Nova-API-Version", "2.55")
				th.TestHeader(t, r, "OpenStack-API-Version", "compute 2.55")
				if !reflect.DeepEqual(r.URL.Query(), wantQuery) || r.URL.RawQuery != wantQuery.Encode() || r.ContentLength != 0 {
					t.Error("name-based GET or optional owner changed", r.URL, r.ContentLength, wantQuery)
				}
				w.Header().Set("X-Read-Proof", "keypair actual")
				testcloud.JSON(w, check.status, check.body)
			})
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				other.Add(1)
				t.Error("keypair GET performed LIST/fallback/unrelated request", r.Method, r.URL)
				w.WriteHeader(500)
			})
			client := cloud.Client("compute", "/unused/catalog")
			client.ResourceBase = cloud.Server.URL + "/reverse/nova/v2.1/"
			client.Microversion = "2.55"
			client.MoreHeaders = map[string]string{"X-Source": "selected"}
			cloud.Provider.SetToken("keypair-live")
			var options []keypairs.GetOption
			if check.explicitOptions {
				options = append(options, keypairs.WithGetOptions(keypairs.GetOpts{UserID: check.owner}))
			}
			value, err := keypairs.New(client).Get(context.Background(), "requested-name", options...)
			if !reflect.DeepEqual(value, check.want) || (err != nil) != check.wantError || gets.Load() != 1 || other.Load() != 0 {
				t.Fatal(value, err, gets.Load(), other.Load(), check.want)
			}
			if check.status != 200 {
				var native gophercloud.ErrUnexpectedResponseCode
				target := cloud.Server.URL + path
				if len(wantQuery) != 0 {
					target += "?" + wantQuery.Encode()
				}
				if !errors.As(err, &native) || native.Actual != check.status || !reflect.DeepEqual(native.Expected, []int{200}) || native.Method != http.MethodGet || native.URL != target || string(native.Body) != check.body || native.ResponseHeader.Get("X-Read-Proof") != "keypair actual" {
					t.Fatal("native keypair HTTP evidence lost", err, native)
				}
			} else if check.typeError {
				var typed *json.UnmarshalTypeError
				if !errors.As(err, &typed) {
					t.Fatal("native keypair type cause lost", err)
				}
			}
			if client.Microversion != "2.55" || client.ProviderClient != cloud.Provider || client.ResourceBase != cloud.Server.URL+"/reverse/nova/v2.1/" {
				t.Fatal("keypair read changed selected client", client)
			}
		})
	}
}
