package api_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestServerPasswordDefaultAndOptionalDecryption(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := rsa.EncryptPKCS1v15(rand.Reader, &key.PublicKey, []byte("test-password"))
	if err != nil {
		t.Fatal(err)
	}
	encrypted := base64.StdEncoding.EncodeToString(sealed)
	cloud := testcloud.New(t)
	calls := 0
	cloud.Mux.HandleFunc("/compute/servers/server/os-server-password", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want %s", r.Method, http.MethodGet)
		}
		if got := r.Header.Values("X-Auth-Token"); len(got) == 0 || slices.ContainsFunc(got, func(value string) bool { return value != "test-token" }) {
			t.Errorf("header %s = %q, want %q", "X-Auth-Token", got, "test-token")
		}
		if got := r.Header.Values("X-OpenStack-Nova-API-Version"); len(got) == 0 || slices.ContainsFunc(got, func(value string) bool { return value != "2.87" }) {
			t.Errorf("header %s = %q, want %q", "X-OpenStack-Nova-API-Version", got, "2.87")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"password": encrypted})
	})
	client := cloud.Client("compute", "/compute")
	client.Microversion = "2.87"
	api := servers.New(client)
	ctx := context.Background()
	value, err := api.GetPassword(ctx, "server")
	if err != nil || value != encrypted {
		t.Fatalf("default value=%q err=%v", value, err)
	}
	value, err = api.GetPassword(ctx, "server", servers.WithGetPasswordPrivateKey(key))
	if err != nil || value != "test-password" {
		t.Fatalf("decrypted value=%q err=%v", value, err)
	}
	value, err = api.GetPassword(ctx, "server", servers.WithGetPasswordPrivateKey(key), servers.WithGetPasswordPrivateKey(nil))
	if err != nil || value != encrypted {
		t.Fatalf("restored value=%q err=%v", value, err)
	}
	for _, opts := range [][]servers.GetPasswordOption{{nil}, {servers.WithGetPasswordPrivateKey(&rsa.PrivateKey{})}} {
		value, err := api.GetPassword(ctx, "server", opts...)
		if value != "" || !errors.Is(err, resource.ErrInvalidOption) || calls != 3 {
			t.Fatalf("value=%q err=%v calls=%d", value, err, calls)
		}
	}
}

func TestServerPasswordHandlesEmptyResponsesAndErrors(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, body, want string
		status           int
		decrypt, fails   bool
	}{
		{"empty", `{"password":""}`, "", 200, true, false},
		{"missing", `{}`, "", 200, true, false},
		{"null", `{"password":null}`, "", 200, true, false},
		{"opaque-default", `{"password":"not-base64"}`, "not-base64", 200, false, false},
		{"invalid-ciphertext", `{"password":"not-base64"}`, "", 200, true, true},
		{"invalid-type", `{"password":false}`, "", 200, false, true},
		{"forbidden", `{"error":"forbidden"}`, "", 403, false, true},
		{"unexpected-status", `{"password":"opaque"}`, "", 203, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("/compute/servers/server/os-server-password", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, tc.status, tc.body) })
			var opts []servers.GetPasswordOption
			if tc.decrypt {
				opts = append(opts, servers.WithGetPasswordPrivateKey(key))
			}
			value, err := servers.New(cloud.Client("compute", "/compute")).GetPassword(context.Background(), "server", opts...)
			if value != tc.want || (err != nil) != tc.fails {
				t.Fatalf("value=%q err=%v", value, err)
			}
			if tc.fails {
				var operation *resource.OperationError
				if value != "" || !errors.As(err, &operation) {
					t.Fatalf("value=%q err=%v", value, err)
				}
			}
			if tc.status == 403 || tc.status == 203 {
				var response gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &response) || response.Actual != tc.status {
					t.Fatal(err)
				}
			}
		})
	}
}
