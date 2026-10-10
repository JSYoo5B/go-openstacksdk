package ec2tokens_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/ec2tokens"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/ec2tokens"
)

type nativeEC2Transport func(*http.Request) (*http.Response, error)

func (transport nativeEC2Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type nativeEC2Call struct{ method, path, query, body, authToken, vendor string }

func nativeEC2API(t *testing.T, calls *[]nativeEC2Call, reply func(*http.Request) (int, string)) (*ec2tokens.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeEC2Transport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeEC2Call{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("X-Auth-Token"), req.Header.Get("X-Vendor")})
		code, body := reply(req)
		header := http.Header{"Content-Type": {"application/json"}, "X-Subject-Token": {"issued"}}
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: header}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return ec2tokens.New(client), cloud
}

func nativeEC2Operation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "ec2tokens" {
		t.Fatal("generated ec2tokens context", err, wrapped)
	}
}

func nativeEC2JSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

const nativeEC2Token = `{"token":{"expires_at":"2026-10-11T01:02:03.000000Z","methods":["ec2credential"],"user":{"id":"u1"}}}`

func TestNativeEC2TokenBodiesSignaturesAndSubjectToken(t *testing.T) {
	ctx := context.Background()
	var calls []nativeEC2Call
	api, cloud := nativeEC2API(t, &calls, func(*http.Request) (int, string) { return 200, nativeEC2Token })
	expires := time.Date(2026, 10, 11, 1, 2, 3, 0, time.UTC)

	// An explicit signature skips signing, so every untagged-omitempty field is sent as-is.
	token, err := api.Create(ctx, &ec2tokens.AuthOptions{Access: "a", Signature: "sig", Token: []byte("t")}, ec2tokens.WithCreateField("x_extension", 1))
	if err != nil || token.ID != "issued" || !token.ExpiresAt.Equal(expires) {
		t.Fatal(token, err)
	}
	if cloud.Provider.Token() != "test-token" || cloud.Provider.ReauthFunc != nil {
		t.Fatal("provider token changed", cloud.Provider.Token())
	}
	if _, err := api.ValidateS3Token(ctx, &ec2tokens.AuthOptions{Access: "a", Signature: []byte("sig"), Token: []byte("t"), Host: "h"}, ec2tokens.WithValidateS3TokenField("x_extension", 1)); err != nil {
		t.Fatal(err)
	}

	// Signature V4 is deterministic once the timestamp and body hash are fixed.
	stamp := time.Date(2026, 10, 11, 1, 2, 3, 0, time.UTC)
	bodyHash := strings.Repeat("ab", 32)
	v4 := ec2tokens.AuthOptions{
		Access: "a", Secret: "s", Region: "RegionOne", Service: "ec2", Verb: "GET", Path: "/", Host: "ignored",
		Headers: map[string]string{"Host": "ec2.example", "X-Amz-SignedHeaders": "host"}, Params: map[string]string{"Action": "Describe"},
		BodyHash: &bodyHash, Timestamp: &stamp,
	}
	if _, err := api.Create(ctx, &v4); err != nil {
		t.Fatal(err)
	}
	if _, err := api.ValidateS3Token(ctx, &v4); err != nil {
		t.Fatal(err)
	}
	if v4.Headers["Authorization"] != "" {
		t.Fatal("caller headers were mutated", v4.Headers)
	}
	stringToSign := upstream.EC2CredentialsBuildStringToSignV4(v4, "host", bodyHash, stamp)
	signature := upstream.EC2CredentialsBuildSignatureV4(upstream.EC2CredentialsBuildSignatureKeyV4("s", "RegionOne", "ec2", stamp), stringToSign)
	v4Create := nativeEC2JSON(t, map[string]any{"credentials": map[string]any{
		"access": "a", "body_hash": bodyHash, "host": "ignored", "path": "/", "verb": "GET", "params": v4.Params, "signature": signature,
		"headers": map[string]string{
			"Host": "ec2.example", "X-Amz-SignedHeaders": "host", "X-Amz-Date": "20261011T010203Z",
			"Authorization": "AWS4-HMAC-SHA256 Credential=a/20261011/RegionOne/ec2/aws4_request, SignedHeaders=host, Signature=" + signature,
		},
	}})
	v4Validate := nativeEC2JSON(t, map[string]any{"credentials": map[string]any{"access": "a", "signature": signature, "token": stringToSign}})

	// Signature V2 signs verb, host, path and the sorted params, and drops body_hash and headers.
	v2 := ec2tokens.AuthOptions{
		Access: "a", Secret: "s", Verb: "GET", Host: "ec2.example", Path: "/",
		Params:  map[string]string{"SignatureVersion": "2", "SignatureMethod": ec2tokens.EC2CredentialsHmacSha256V2, "Action": "Describe"},
		Headers: map[string]string{"dropped": "1"},
	}
	if _, err := api.Create(ctx, &v2); err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, []byte("s"))
	mac.Write([]byte("GET\nec2.example\n/\nAction=Describe&SignatureMethod=HmacSHA256&SignatureVersion=2"))
	v2Create := nativeEC2JSON(t, map[string]any{"credentials": map[string]any{
		"access": "a", "host": "ec2.example", "path": "/", "verb": "GET", "params": v2.Params, "signature": mac.Sum(nil),
	}})

	want := []nativeEC2Call{
		// Create drops token; extensions land inside credentials.
		{http.MethodPost, "/keystone/v3/ec2tokens", "", `{"credentials":{"access":"a","body_hash":null,"headers":null,"host":"","params":null,"path":"","signature":"sig","verb":"","x_extension":1}}`, "test-token", ""},
		// ValidateS3Token keeps only access, signature, token and extensions; []byte values are base64.
		{http.MethodPost, "/keystone/v3/s3tokens", "", `{"credentials":{"access":"a","signature":"c2ln","token":"dA==","x_extension":1}}`, "test-token", ""},
		{http.MethodPost, "/keystone/v3/ec2tokens", "", v4Create, "test-token", ""},
		{http.MethodPost, "/keystone/v3/s3tokens", "", v4Validate, "test-token", ""},
		{http.MethodPost, "/keystone/v3/ec2tokens", "", v2Create, "test-token", ""},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v\n%+v", calls, want)
	}

	// Without a body hash or timestamp the native code generates a random 64-byte hash.
	calls = nil
	if _, err := api.Create(ctx, &ec2tokens.AuthOptions{Access: "a", Secret: "s"}); err != nil {
		t.Fatal(err)
	}
	var random struct {
		Credentials map[string]any `json:"credentials"`
	}
	if err := json.Unmarshal([]byte(calls[0].body), &random); err != nil {
		t.Fatal(err)
	}
	if hash, _ := random.Credentials["body_hash"].(string); len(hash) != 128 || random.Credentials["token"] != nil || random.Credentials["signature"] == "" {
		t.Fatal(random)
	}
}

// The native Create and ValidateS3Token never call ToTokenV3HeadersMap, so the
// generated header extension is accepted and then silently dropped.
func TestNativeEC2TokenHeaderExtensionIsDropped(t *testing.T) {
	ctx := context.Background()
	var calls []nativeEC2Call
	api, _ := nativeEC2API(t, &calls, func(*http.Request) (int, string) { return 200, nativeEC2Token })
	if _, err := api.Create(ctx, &ec2tokens.AuthOptions{Access: "a", Signature: "sig"}, ec2tokens.WithCreateHeader("X-Vendor", "1")); err != nil {
		t.Fatal(err)
	}
	if _, err := api.ValidateS3Token(ctx, &ec2tokens.AuthOptions{Access: "a", Signature: "sig"}, ec2tokens.WithValidateS3TokenHeader("X-Vendor", "1")); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0].vendor != "" || calls[1].vendor != "" {
		t.Fatal(calls)
	}
}

func TestNativeEC2TokenStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	valid := func() *ec2tokens.AuthOptions { return &ec2tokens.AuthOptions{Access: "a", Signature: "sig"} }
	for _, call := range []struct {
		name     string
		accepted []int
		call     func(*ec2tokens.API) error
	}{
		{"Create", []int{200}, func(api *ec2tokens.API) error { _, err := api.Create(ctx, valid()); return err }},
		{"ValidateS3Token", []int{200}, func(api *ec2tokens.API) error { _, err := api.ValidateS3Token(ctx, valid()); return err }},
	} {
		for _, code := range []int{200, 201, 202, 204, 404} {
			if slices.Contains(call.accepted, code) {
				continue
			}
			t.Run(fmt.Sprintf("%s/%d", call.name, code), func(t *testing.T) {
				var calls []nativeEC2Call
				api, _ := nativeEC2API(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
				err := call.call(api)
				nativeEC2Operation(t, err, call.name)
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
					t.Fatal(err, native)
				}
			})
		}
	}
	t.Run("preflight", func(t *testing.T) {
		var calls []nativeEC2Call
		api, _ := nativeEC2API(t, &calls, func(*http.Request) (int, string) { return 200, nativeEC2Token })
		v2 := func(params map[string]string) *ec2tokens.AuthOptions {
			return &ec2tokens.AuthOptions{Access: "a", Secret: "s", Params: params}
		}
		for name, check := range map[string]struct {
			operation string
			err       error
		}{
			"access":      {"Create", func() error { _, err := api.Create(ctx, &ec2tokens.AuthOptions{Signature: "sig"}); return err }()},
			"nil options": {"Create", func() error { _, err := api.Create(ctx, nil); return err }()},
			"nil option":  {"Create", func() error { _, err := api.Create(ctx, valid(), nil); return err }()},
			"signature version": {"Create", func() error {
				_, err := api.Create(ctx, v2(map[string]string{"SignatureVersion": "1"}))
				return err
			}()},
			"v2 method missing": {"Create", func() error {
				_, err := api.Create(ctx, v2(map[string]string{"SignatureVersion": "2"}))
				return err
			}()},
			"v2 method unsupported": {"Create", func() error {
				_, err := api.Create(ctx, v2(map[string]string{"SignatureVersion": "2", "SignatureMethod": "HmacMD5"}))
				return err
			}()},
			"core extension": {"Create", func() error {
				_, err := api.Create(ctx, valid(), ec2tokens.WithCreateField("signature", "x"))
				return err
			}()},
			// token is reserved by its json tag even though Create deletes it from the body.
			"token extension": {"Create", func() error {
				_, err := api.Create(ctx, valid(), ec2tokens.WithCreateField("token", "x"))
				return err
			}()},
			"invalid header": {"Create", func() error {
				_, err := api.Create(ctx, valid(), ec2tokens.WithCreateHeader("bad header", "x"))
				return err
			}()},
			"validate access":      {"ValidateS3Token", func() error { _, err := api.ValidateS3Token(ctx, &ec2tokens.AuthOptions{}); return err }()},
			"validate nil options": {"ValidateS3Token", func() error { _, err := api.ValidateS3Token(ctx, nil); return err }()},
			"validate core extension": {"ValidateS3Token", func() error {
				_, err := api.ValidateS3Token(ctx, valid(), ec2tokens.WithValidateS3TokenField("access", "x"))
				return err
			}()},
		} {
			if check.err == nil {
				t.Fatal(name, "accepted")
			}
			nativeEC2Operation(t, check.err, check.operation)
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
