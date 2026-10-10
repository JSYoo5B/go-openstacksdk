package secretacls_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretacls"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type aclTransport func(*http.Request) (*http.Response, error)

func (transport aclTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func aclWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}, "X-Acl-Proof": {"actual"}}}
}

type aclCall struct{ method, path, body, extra string }

func aclScope(t *testing.T, calls *[]aclCall, reply func(*http.Request) *http.Response) *secretacls.SecretScope {
	t.Helper()
	cloud := testcloud.New(t)
	client := cloud.Client("key-manager", "/barbican/v1/")
	cloud.Provider.HTTPClient.Transport = aclTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, aclCall{req.Method, req.URL.EscapedPath(), raw, req.Header.Get("X-Extra")})
		return reply(req), nil
	})
	scope, err := secretacls.New(client).InSecret(context.Background(), resource.ID("s 1"))
	if err == nil {
		t.Fatal("whitespace IDs are rejected before HTTP")
	}
	scope, err = secretacls.New(client).InSecret(context.Background(), resource.ID("s1"))
	if err != nil || scope.SecretID() != "s1" || len(*calls) != 0 {
		t.Fatal(scope, err, *calls)
	}
	return scope
}

func TestSecretACLGetFollowsPythonFetch(t *testing.T) {
	for _, test := range []struct {
		name, body, read, ref string
		code                  int
	}{
		{"read object", `{"read":{"users":["u"],"project-access":false,"created":"x"},"extra":1}`, `{"users":["u"],"project-access":false,"created":"x"}`, "null", 200},
		{"accepted below 400 with ref", `{"acl_ref":"https://kms/acl"}`, "null", `"https://kms/acl"`, 203},
		{"empty body keeps seed", ``, "null", "null", 204},
		{"invalid JSON keeps seed", `not json`, "null", "null", 300},
		{"null read", `{"read":null}`, "null", "null", 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls []aclCall
			scope := aclScope(t, &calls, func(*http.Request) *http.Response { return aclWire(test.code, test.body) })
			value, err := scope.Get(context.Background(), secretacls.WithHeader("X-Extra", "1"))
			if err != nil || value.SecretID != "s1" || string(value.Read) != test.read || string(value.ACLRef) != test.ref || value.StatusCode != test.code || value.Header.Get("X-ACL-Proof") != "actual" {
				t.Fatal(value, err)
			}
			if len(calls) != 1 || calls[0] != (aclCall{http.MethodGet, "/barbican/v1/secrets/s1/acl", "", "1"}) {
				t.Fatal(calls)
			}
		})
	}
	for _, body := range []string{`[]`, `"x"`, `{"read":"x"}`, `{"read":[1]}`} {
		t.Run("rejected "+body, func(t *testing.T) {
			var calls []aclCall
			scope := aclScope(t, &calls, func(*http.Request) *http.Response { return aclWire(200, body) })
			value, err := scope.Get(context.Background())
			var response *resource.ResponseError
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &response) || string(response.Body) != body {
				t.Fatal(value, err)
			}
		})
	}
	t.Run("native 404", func(t *testing.T) {
		var calls []aclCall
		scope := aclScope(t, &calls, func(*http.Request) *http.Response { return aclWire(404, `{}`) })
		if value, err := scope.Get(context.Background()); value != nil || !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			t.Fatal(value, err)
		}
	})
}

func TestSecretACLSetAndUpdateCommitWithPut(t *testing.T) {
	for _, operation := range []string{"Set", "Update"} {
		t.Run(operation, func(t *testing.T) {
			var calls []aclCall
			scope := aclScope(t, &calls, func(*http.Request) *http.Response { return aclWire(200, `{"acl_ref":"https://kms/acl"}`) })
			call := scope.Set
			if operation == "Update" {
				call = scope.Update
			}
			read := []byte(`{"users":["u"],"project-access":false}`)
			value, err := call(context.Background(), secretacls.ACLInput{Read: read}, secretacls.WithHeader("X-Extra", "1"))
			if err != nil || string(value.Read) != string(read) || string(value.ACLRef) != `"https://kms/acl"` || value.StatusCode != 200 {
				t.Fatal(value, err)
			}
			// Without a declared attribute Python commit has nothing dirty.
			empty, err := call(context.Background(), secretacls.ACLInput{})
			if err != nil || string(empty.Read) != "null" || string(empty.ACLRef) != "null" || empty.StatusCode != 0 {
				t.Fatal(empty, err)
			}
			if _, err := call(context.Background(), secretacls.ACLInput{ACLRef: []byte(`"sent"`)}); err != nil {
				t.Fatal(err)
			}
			want := []aclCall{
				{http.MethodPut, "/barbican/v1/secrets/s1/acl", `{"read":{"users":["u"],"project-access":false}}`, "1"},
				{http.MethodPut, "/barbican/v1/secrets/s1/acl", `{"acl_ref":"sent"}`, ""},
			}
			if fmt.Sprint(calls) != fmt.Sprint(want) {
				t.Fatal(calls)
			}
			for _, input := range []secretacls.ACLInput{{Read: []byte(`[]`)}, {Read: []byte(`null`)}, {Read: []byte(`{`)}, {ACLRef: []byte(`{`)}} {
				if _, err := call(context.Background(), input); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(input, err)
				}
			}
			if _, err := call(context.Background(), secretacls.ACLInput{Read: read}, secretacls.WithHeader("Content-Type", "x")); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 2 {
				t.Fatal(err, calls)
			}
		})
	}
}

func TestSecretACLDeleteIgnoresMissingByDefault(t *testing.T) {
	var code atomic.Int32
	var calls []aclCall
	scope := aclScope(t, &calls, func(*http.Request) *http.Response { return aclWire(int(code.Load()), "") })
	code.Store(200)
	value, err := scope.Delete(context.Background(), secretacls.WithDeleteHeader("X-Extra", "1"))
	if err != nil || value == nil || value.SecretID != "s1" || value.StatusCode != 200 || string(value.Read) != "null" {
		t.Fatal(value, err)
	}
	code.Store(404)
	if value, err := scope.Delete(context.Background()); value != nil || err != nil {
		t.Fatal(value, err)
	}
	if value, err := scope.Delete(context.Background(), secretacls.WithDeleteIgnoreMissing(false)); value != nil || !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(value, err)
	}
	code.Store(409)
	if value, err := scope.Delete(context.Background()); value != nil || !gophercloud.ResponseCodeIs(err, http.StatusConflict) {
		t.Fatal(value, err)
	}
	if len(calls) != 4 || calls[0] != (aclCall{http.MethodDelete, "/barbican/v1/secrets/s1/acl", "", "1"}) {
		t.Fatal(calls)
	}
}
