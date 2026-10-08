package metadefobjects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type objectCoreTransport func(*http.Request) (*http.Response, error)

func (f objectCoreTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type objectCoreBody struct {
	reader   io.Reader
	closes   int
	closeErr error
}

func (b *objectCoreBody) Read(p []byte) (int, error) { return b.reader.Read(p) }
func (b *objectCoreBody) Close() error               { b.closes++; return b.closeErr }

type objectCoreReader struct {
	data  string
	err   error
	after func()
}

func (r *objectCoreReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	if r.after != nil {
		r.after()
		r.after = nil
	}
	if r.data == "" {
		return n, r.err
	}
	return n, nil
}
func objectCoreClient(transport objectCoreTransport) *gophercloud.ServiceClient {
	return &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{TokenID: "before", HTTPClient: http.Client{Transport: transport}}, Type: "image", Endpoint: "https://example.test/catalog/", ResourceBase: "https://example.test/reverse/glance/v2/"}
}
func objectCoreHTTP(req *http.Request, status int, body io.ReadCloser) *http.Response {
	return &http.Response{Request: req, StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}, "X-Proof": {"original"}}, Body: body}
}
func objectCoreJSON(req *http.Request, status int, body string) *http.Response {
	return objectCoreHTTP(req, status, io.NopCloser(strings.NewReader(body)))
}
func objectCoreProof(t *testing.T, err error, status int, body string) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Proof") != "original" {
		t.Fatalf("missing exact proof: %v, %+v", err, proof)
	}
	return proof
}

func objectCoreScope(t *testing.T, client *gophercloud.ServiceClient, parent string) *NamespaceScope {
	t.Helper()
	scope, err := New(client).InNamespace(context.Background(), parent)
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func TestMetadefObjectsCoreFixedScopeAndCRUD(t *testing.T) {
	parent, name := "OS::Compute::空 白", " child::名字 "
	calls := 0
	client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		base := "/reverse/glance/v2/metadefs/namespaces/" + url.PathEscape(parent) + "/objects"
		var body map[string]any
		if req.Body != nil {
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
		}
		if req.URL.RawQuery != "" {
			t.Fatalf("unexpected query %s", req.URL)
		}
		switch calls {
		case 1:
			if req.Method != "POST" || req.URL.EscapedPath() != base || !reflect.DeepEqual(body, map[string]any{"name": name}) {
				t.Fatalf("create %s %s %#v", req.Method, req.URL, body)
			}
			return objectCoreJSON(req, 201, `{"name":"passive-server-name","self":"https://foreign.test","unknown":9007199254740993}`), nil
		case 2:
			expected := map[string]any{"name": name, "description": "", "properties": map[string]any{}, "required": []any{}}
			if req.Method != "POST" || !reflect.DeepEqual(body, expected) {
				t.Fatalf("explicit empty body %#v", body)
			}
			return objectCoreJSON(req, 201, `{}`), nil
		case 3:
			if req.Method != "GET" || req.URL.EscapedPath() != base+"/"+url.PathEscape(name) || body != nil {
				t.Fatalf("get %s %#v", req.URL, body)
			}
			return objectCoreJSON(req, 200, `{"name":"other","schema":"https://foreign.test/schema"}`), nil
		case 4:
			if req.Method != "PUT" || req.URL.EscapedPath() != base+"/"+url.PathEscape(name) || !reflect.DeepEqual(body, map[string]any{"name": "new::object"}) {
				t.Fatalf("rename fixed route %s %#v", req.URL, body)
			}
			return objectCoreJSON(req, 200, `{"name":"new::object"}`), nil
		case 5:
			if req.Method != "PUT" || !reflect.DeepEqual(body, map[string]any{"name": name}) {
				t.Fatalf("replacement default %#v", body)
			}
			return objectCoreJSON(req, 200, `{}`), nil
		case 6:
			if req.Method != "DELETE" || req.URL.EscapedPath() != base+"/"+url.PathEscape(name) || req.Body != nil {
				t.Fatalf("delete %s", req.URL)
			}
			return objectCoreJSON(req, 204, "opaque"), nil
		case 7:
			if req.Method != "DELETE" || req.URL.EscapedPath() != base || req.Body != nil {
				t.Fatalf("deleteall %s", req.URL)
			}
			return objectCoreJSON(req, 204, "all"), nil
		}
		t.Fatalf("unexpected call%d", calls)
		return nil, nil
	})
	api := New(client)
	scope, err := api.InNamespace(context.Background(), parent)
	if err != nil || calls != 0 || scope.NamespaceName() != parent || scope.RawClient() != client || api.RawClient() != client {
		t.Fatalf("zeroHTTP scope %+v %v calls%d", scope, err, calls)
	}
	value, err := scope.Create(context.Background(), name)
	if err != nil || value.Name == nil || *value.Name != "passive-server-name" || value.StatusCode != 201 || string(value.Body["unknown"]) != "9007199254740993" {
		t.Fatalf("create %+v %v", value, err)
	}
	if _, err = scope.Create(context.Background(), name, WithCreateDescription(""), WithCreateProperties(map[string]json.RawMessage{}), WithCreateRequired([]string{})); err != nil {
		t.Fatal(err)
	}
	if _, err = scope.Get(context.Background(), name); err != nil {
		t.Fatal(err)
	}
	if _, err = scope.Update(context.Background(), name, WithUpdateName("new::object")); err != nil {
		t.Fatal(err)
	}
	if _, err = scope.Update(context.Background(), name); err != nil {
		t.Fatal(err)
	}
	ack, err := scope.Delete(context.Background(), name)
	if err != nil || ack == nil || ack.Namespace != parent || ack.Name == nil || *ack.Name != name || ack.StatusCode != 204 || string(ack.Body) != "opaque" {
		t.Fatalf("delete ack %+v %v", ack, err)
	}
	ack, err = scope.DeleteAll(context.Background())
	if err != nil || ack == nil || ack.Namespace != parent || ack.Name != nil || ack.StatusCode != 204 || string(ack.Body) != "all" || calls != 7 {
		t.Fatalf("deleteall ack %+v %v calls%d", ack, err, calls)
	}
}

func TestMetadefObjectsCoreCanonicalPresenceAndAtomicDecoder(t *testing.T) {
	var value Object
	raw := []byte(`{"name":"","description":null,"properties":{"raw":false,"large":9007199254740993,"nested":{"type":"unrecognized"}},"required":["","x,y","line\nnext"],"created_at":"not-a-date","updated_at":"","self":"foreign","schema":"s","links":false,"Namespace":"decoy","unknown":9007199254740993}`)
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	if value.Name == nil || *value.Name != "" || value.Description != nil || value.CreatedAt == nil || *value.CreatedAt != "not-a-date" || value.UpdatedAt == nil || *value.UpdatedAt != "" || value.Links != nil || value.Required[1] != "x,y" {
		t.Fatalf("presence %+v", value)
	}
	if string(value.Properties["large"]) != "9007199254740993" || string(value.Body["unknown"]) != "9007199254740993" || string(value.Body["links"]) != "false" {
		t.Fatal("lost passive precision")
	}
	value.Properties["raw"][0] = 't'
	value.Required[0] = "mutated"
	*value.Name = "mutated"
	if string(value.Body["properties"]) != `{"raw":false,"large":9007199254740993,"nested":{"type":"unrecognized"}}` || string(value.Body["required"]) != `["","x,y","line\nnext"]` || string(value.Body["name"]) != `""` {
		t.Fatal("typed fields alias raw body")
	}
	before := value
	for _, body := range []string{`null`, `[]`, `{"name":1}`, `{"description":false}`, `{"self":[]}`, `{"schema":true}`, `{"created_at":1}`, `{"updated_at":{}}`, `{"properties":[]}`, `{"required":{}}`, `{"required":[null]}`, `{"required":[1]}`} {
		if err := json.Unmarshal([]byte(body), &value); err == nil {
			t.Fatalf("accepted malformed %s", body)
		}
		if !reflect.DeepEqual(value, before) {
			t.Fatalf("partial decoder assignment on %s", body)
		}
	}
	if err := json.Unmarshal([]byte("{\"unknown\":\"\xff\"}"), &value); err == nil {
		t.Fatal("accepted invalid root UTF8")
	}
	for _, body := range []string{`{}`, `{"properties":null,"required":null,"name":null,"created_at":null}`} {
		var empty Object
		if err := json.Unmarshal([]byte(body), &empty); err != nil || empty.Properties != nil || empty.Required != nil || empty.Name != nil || empty.CreatedAt != nil || empty.Body == nil {
			t.Fatalf("null/missing %+v %v", empty, err)
		}
	}
	var empty Object
	if err := json.Unmarshal([]byte(`{"properties":{},"required":[]}`), &empty); err != nil || empty.Properties == nil || empty.Required == nil {
		t.Fatalf("explicit empty %+v %v", empty, err)
	}
}

func TestMetadefObjectsCoreCompletePreflight(t *testing.T) {
	calls, callbacks := 0, 0
	client := objectCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return objectCoreJSON(req, 200, `{}`), nil })
	api := New(client)
	for _, name := range []string{"", ".", "..", "a/b", "a\\b", "a%b", "a?b", "a#b", "a\nb", string([]byte{0xff}), strings.Repeat("界", 81)} {
		if scope, err := api.InNamespace(context.Background(), name); scope != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("bad parent %q %+v %v", name, scope, err)
		}
	}
	if scope, err := api.InNamespace(nil, "parent"); scope != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("nil context %+v %v", scope, err)
	}
	custom := errors.New("cancel-cause")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(custom)
	if _, err := api.InNamespace(ctx, "parent"); !errors.Is(err, context.Canceled) || !errors.Is(err, custom) {
		t.Fatalf("cancel cause %v", err)
	}
	scope := objectCoreScope(t, client, "OS::parent")
	callback := CreateOption(func(*CreateOpts) error { callbacks++; return nil })
	for _, name := range []string{"", ".", "..", "a/b", "a\\b", "a%b", "a?b", "a#b", "a\nb", strings.Repeat("界", 81)} {
		if _, err := scope.Create(context.Background(), name, callback); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("bad child %q %v", name, err)
		}
	}
	if calls != 0 || callbacks != 0 {
		t.Fatalf("preflight calls%d callbacks%d", calls, callbacks)
	}
	checks := []func() error{
		func() error { _, e := scope.Create(context.Background(), "child", nil); return e },
		func() error {
			_, e := scope.Create(context.Background(), "child", WithCreateProperties(map[string]json.RawMessage{"x": json.RawMessage(`null`)}))
			return e
		},
		func() error {
			_, e := scope.Update(context.Background(), "child", WithUpdateName("invalid/path"))
			return e
		},
		func() error {
			_, e := scope.Get(context.Background(), "child", WithGetHeader("Host", "foreign"))
			return e
		},
		func() error { _, e := scope.Delete(context.Background(), "child", nil); return e },
		func() error {
			_, e := scope.DeleteAll(context.Background(), WithDeleteAllHeader("Accept", "custom"))
			return e
		},
		func() error { _, e := scope.All(context.Background(), WithListMaxItems(-1)); return e },
	}
	for i, check := range checks {
		if err := check(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("optioncheck%d %v", i, err)
		}
	}
	cause := errors.New("caller-cause")
	if _, err := scope.Update(context.Background(), "child", func(*UpdateOpts) error { return cause }); !errors.Is(err, cause) {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("invalid options sent calls%d", calls)
	}
	wrong := objectCoreClient(nil)
	wrong.Type = "compute"
	if _, err := New(wrong).InNamespace(context.Background(), "parent"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	var nilAPI *API
	var nilScope *NamespaceScope
	if _, err := nilAPI.InNamespace(context.Background(), "parent"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := nilScope.Get(context.Background(), "child"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestMetadefObjectsCoreFiniteConsumptionAndOwnership(t *testing.T) {
	calls, callbacks := 0, 0
	body := `{"objects":[{"name":"one","unknown":9007199254740993},{"name":1}],"next":123,"first":"https://foreign.test","schema":"foreign"}`
	scope := objectCoreScope(t, objectCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != "GET" || req.URL.RawQuery != "" || req.Body != nil {
			t.Fatalf("list has query/body %s", req.URL)
		}
		response := objectCoreJSON(req, 200, body)
		response.Header.Set("Link", "<https://foreign.test>; rel=next")
		return response, nil
	}), "parent")
	seq := scope.List(context.Background(), func(*ListOpts) error { callbacks++; return nil }, WithListMaxItems(1))
	if calls != 0 || callbacks != 0 {
		t.Fatal("eager iterator")
	}
	for iteration := 0; iteration < 2; iteration++ {
		rows := 0
		for value, err := range seq {
			if err != nil || *value.Name != "one" {
				t.Fatalf("capped row %+v %v", value, err)
			}
			rows++
			value.Body["unknown"][0] = '8'
			value.Header.Set("X-Proof", "mutated")
		}
		if rows != 1 {
			t.Fatal(rows)
		}
	}
	if calls != 2 || callbacks != 2 {
		t.Fatalf("reuse calls%d callbacks%d", calls, callbacks)
	}
	rows, err := scope.All(context.Background())
	if rows != nil || err == nil {
		t.Fatalf("late error retained rows %+v %v", rows, err)
	}
	objectCoreProof(t, err, 200, body)
	for value, err := range scope.List(context.Background()) {
		if err != nil || value == nil {
			t.Fatal(err)
		}
		break
	}
	if calls != 4 {
		t.Fatal(calls)
	}
	for _, bad := range []string{`{}`, `null`, `[]`, `{"objects":null}`, `{"objects":{}}`, "{\"objects\":[],\"unknown\":\"\xff\"}"} {
		badScope := objectCoreScope(t, objectCoreClient(func(req *http.Request) (*http.Response, error) { return objectCoreJSON(req, 200, bad), nil }), "parent")
		rows, err := badScope.All(context.Background())
		if rows != nil || err == nil {
			t.Fatalf("bad envelope %s %+v %v", bad, rows, err)
		}
		objectCoreProof(t, err, 200, bad)
	}
	empty := objectCoreScope(t, objectCoreClient(func(req *http.Request) (*http.Response, error) {
		return objectCoreJSON(req, 200, `{"objects":[],"next":false}`), nil
	}), "parent")
	if rows, err := empty.All(context.Background()); err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("empty %+v %v", rows, err)
	}
	var client *gophercloud.ServiceClient
	client = objectCoreClient(func(req *http.Request) (*http.Response, error) {
		return objectCoreJSON(req, 200, `{"objects":[{},{}]}`), nil
	})
	changed := objectCoreScope(t, client, "parent")
	count := 0
	for value, err := range changed.List(context.Background()) {
		if count == 0 {
			if value == nil || err != nil {
				t.Fatal(err)
			}
			client.ResourceBase = "https://example.test/changed/"
			count++
			continue
		}
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("postyield drift %+v %v", value, err)
		}
		objectCoreProof(t, err, 200, `{"objects":[{},{}]}`)
		count++
	}
	if count != 2 {
		t.Fatal(count)
	}
}

func TestMetadefObjectsCoreAcceptedBodiesAndMissingPolicy(t *testing.T) {
	readCause, closeCause := errors.New("read-cause"), errors.New("close-cause")
	t.Run("accepted typed model failure proof", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		custom := errors.New("custom-context")
		body := &objectCoreBody{reader: &objectCoreReader{data: `{"name":"partial"}`, err: readCause, after: func() { cancel(custom) }}, closeErr: closeCause}
		scope := objectCoreScope(t, objectCoreClient(func(req *http.Request) (*http.Response, error) { return objectCoreHTTP(req, 201, body), nil }), "parent")
		value, err := scope.Create(ctx, "child")
		if value != nil || body.closes != 1 || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, custom) {
			t.Fatalf("body ownership %+v %v closes%d", value, err, body.closes)
		}
		objectCoreProof(t, err, 201, `{"name":"partial"}`)
	})
	for _, bulk := range []bool{false, true} {
		t.Run(fmt.Sprintf("opaque204 bulk%v", bulk), func(t *testing.T) {
			body := &objectCoreBody{reader: strings.NewReader("opaque\x00body"), closeErr: closeCause}
			scope := objectCoreScope(t, objectCoreClient(func(req *http.Request) (*http.Response, error) { return objectCoreHTTP(req, 204, body), nil }), "parent")
			var ack *Acknowledgement
			var err error
			if bulk {
				ack, err = scope.DeleteAll(context.Background())
			} else {
				ack, err = scope.Delete(context.Background(), "child")
			}
			if ack == nil || ack.Namespace != "parent" || ack.StatusCode != 204 || string(ack.Body) != "opaque\x00body" || body.closes != 1 || !errors.Is(err, closeCause) || bulk != (ack.Name == nil) {
				t.Fatalf("ack %+v %v closes%d", ack, err, body.closes)
			}
			proof := objectCoreProof(t, err, 204, "opaque\x00body")
			proof.Body[0] = 'X'
			proof.Header.Set("X-Proof", "changed")
			if string(ack.Body) != "opaque\x00body" || ack.Header.Get("X-Proof") != "original" {
				t.Fatal("ack aliases error proof")
			}
		})
	}
	t.Run("physical404 only individual clean", func(t *testing.T) {
		hooks := 0
		retryCause := errors.New("retry-cause")
		client := objectCoreClient(func(req *http.Request) (*http.Response, error) { return objectCoreJSON(req, 404, "masked"), nil })
		client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			hooks++
			return retryCause
		}
		scope := objectCoreScope(t, client, "parent")
		if ack, err := scope.Delete(context.Background(), "child"); ack != nil || err != nil || hooks != 0 {
			t.Fatalf("default %+v %v hooks%d", ack, err, hooks)
		}
		if ack, err := scope.Delete(context.Background(), "child", WithDeleteIgnoreMissing(false)); ack != nil || !errors.Is(err, retryCause) || hooks != 1 || !gophercloud.ResponseCodeIs(err, 404) {
			t.Fatalf("strict %+v %v hooks%d", ack, err, hooks)
		}
		if ack, err := scope.DeleteAll(context.Background()); ack != nil || !errors.Is(err, retryCause) || hooks != 2 {
			t.Fatalf("bulk strict %+v %v hooks%d", ack, err, hooks)
		}
		bad := &objectCoreBody{reader: strings.NewReader("masked"), closeErr: closeCause}
		scope = objectCoreScope(t, objectCoreClient(func(req *http.Request) (*http.Response, error) { return objectCoreHTTP(req, 404, bad), nil }), "parent")
		if ack, err := scope.Delete(context.Background(), "child"); ack != nil || !errors.Is(err, closeCause) || bad.closes != 1 {
			t.Fatalf("dirty404 %+v %v", ack, err)
		} else {
			objectCoreProof(t, err, 404, "masked")
		}
		nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404}
		scope = objectCoreScope(t, objectCoreClient(func(*http.Request) (*http.Response, error) { return nil, nested }), "parent")
		if ack, err := scope.Delete(context.Background(), "child"); ack != nil || err == nil {
			t.Fatalf("nested404 %+v %v", ack, err)
		}
	})
}

func TestMetadefObjectsCoreLifetimeSourceAndNativeRetryBoundaries(t *testing.T) {
	mutations := []func(*gophercloud.ServiceClient){func(c *gophercloud.ServiceClient) { c.Endpoint = "https://example.test/changed/" }, func(c *gophercloud.ServiceClient) { c.ResourceBase = "https://example.test/changed/" }, func(c *gophercloud.ServiceClient) { c.Microversion = "2.9" }, func(c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} }, func(c *gophercloud.ServiceClient) { c.Type = "compute" }}
	for i, mutate := range mutations {
		t.Run(fmt.Sprintf("lifetime%d", i), func(t *testing.T) {
			calls, callbacks := 0, 0
			client := objectCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return objectCoreJSON(req, 200, `{}`), nil })
			scope := objectCoreScope(t, client, "parent")
			mutate(client)
			if _, err := scope.Get(context.Background(), "child", func(*GetOpts) error { callbacks++; return nil }); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 {
				t.Fatalf("drift beforecallback %v calls%d callbacks%d", err, calls, callbacks)
			}
		})
	}
	t.Run("latest headers before each call auth live", func(t *testing.T) {
		calls := 0
		client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Header.Get("X-Source") != "latest" || req.Header.Get("X-Option") != "chosen" || req.Header.Get("X-Auth-Token") != "after" {
				t.Fatalf("headers %#v", req.Header)
			}
			return objectCoreJSON(req, 200, `{}`), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "initial"}
		scope := objectCoreScope(t, client, "parent")
		client.MoreHeaders["X-Source"] = "latest"
		if _, err := scope.Get(context.Background(), "child", func(o *GetOpts) error {
			client.MoreHeaders["X-Source"] = "callback-change"
			client.ProviderClient.TokenID = "after"
			o.Headers["X-Option"] = "chosen"
			return nil
		}); err != nil || calls != 1 {
			t.Fatal(err)
		}
	})
	t.Run("option callback source drift", func(t *testing.T) {
		calls := 0
		client := objectCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return objectCoreJSON(req, 200, `{}`), nil })
		scope := objectCoreScope(t, client, "parent")
		if _, err := scope.Get(context.Background(), "child", func(*GetOpts) error { client.ResourceBase = "https://example.test/changed/"; return nil }); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatalf("callback drift %v calls%d", err, calls)
		}
	})
	t.Run("retry encoded body remains owned", func(t *testing.T) {
		calls := 0
		client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			return objectCoreJSON(req, 503, "first"), nil
		})
		client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			o.JSONBody = map[string]string{"name": "changed"}
			return nil
		}
		value, err := objectCoreScope(t, client, "parent").Create(context.Background(), "child")
		if value != nil || calls != 1 || !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) {
			t.Fatalf("retry ownership %+v %v calls%d", value, err, calls)
		}
	})
	t.Run("expanded accepted status fails original gate", func(t *testing.T) {
		calls := 0
		client := objectCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return objectCoreJSON(req, 503, "first"), nil
			}
			return objectCoreJSON(req, 202, `{"name":"unexpected"}`), nil
		})
		client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			o.OkCodes = []int{202}
			return nil
		}
		value, err := objectCoreScope(t, client, "parent").Create(context.Background(), "child")
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || calls != 2 || !errors.As(err, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, []int{201}) || string(native.Body) != `{"name":"unexpected"}` {
			t.Fatalf("actual code gate %+v %v", native, err)
		}
	})
	t.Run("drift during accepted body keeps raw proof", func(t *testing.T) {
		var client *gophercloud.ServiceClient
		body := &objectCoreBody{reader: &objectCoreReader{data: `{}`, err: io.EOF, after: func() { client.ResourceBase = "https://example.test/changed/" }}}
		client = objectCoreClient(func(req *http.Request) (*http.Response, error) { return objectCoreHTTP(req, 200, body), nil })
		value, err := objectCoreScope(t, client, "parent").Get(context.Background(), "child")
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || body.closes != 1 {
			t.Fatalf("body drift %+v %v", value, err)
		}
		objectCoreProof(t, err, 200, `{}`)
	})
}
