package metadefproperties

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

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

type propertyCoreTransport func(*http.Request) (*http.Response, error)

func (f propertyCoreTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type propertyCoreBody struct {
	reader   io.Reader
	closes   int
	closeErr error
}

func (b *propertyCoreBody) Read(p []byte) (int, error) { return b.reader.Read(p) }
func (b *propertyCoreBody) Close() error               { b.closes++; return b.closeErr }

type propertyCoreReader struct {
	data  string
	err   error
	after func()
}

func (r *propertyCoreReader) Read(p []byte) (int, error) {
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
func propertyCoreClient(transport propertyCoreTransport) *gophercloud.ServiceClient {
	return &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{TokenID: "before", HTTPClient: http.Client{Transport: transport}}, Type: "image", Endpoint: "https://example.test/catalog/", ResourceBase: "https://example.test/reverse/glance/v2/"}
}
func propertyCoreHTTP(req *http.Request, status int, body io.ReadCloser) *http.Response {
	return &http.Response{Request: req, StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}, "X-Proof": {"original"}}, Body: body}
}
func propertyCoreJSON(req *http.Request, status int, body string) *http.Response {
	return propertyCoreHTTP(req, status, io.NopCloser(strings.NewReader(body)))
}
func propertyCoreProof(t *testing.T, err error, status int, body string) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Proof") != "original" {
		t.Fatalf("missing exact proof: %v, %+v", err, proof)
	}
	return proof
}

func propertyCoreScope(t *testing.T, client *gophercloud.ServiceClient, parent string) *NamespaceScope {
	t.Helper()
	scope, err := New(client).InNamespace(context.Background(), parent)
	if err != nil {
		t.Fatal(err)
	}
	return scope
}

func TestMetadefPropertiesCoreFixedScopeAndCRUD(t *testing.T) {
	parent, name := "OS::Compute::空 白", " child::名字 "
	calls := 0
	client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		base := "/reverse/glance/v2/metadefs/namespaces/" + url.PathEscape(parent) + "/properties"
		var body map[string]json.RawMessage
		if req.Body != nil {
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
		}
		switch calls {
		case 1:
			if req.Method != "POST" || req.URL.EscapedPath() != base || len(body) != 3 || string(body["name"]) != `" child::名字 "` || string(body["type"]) != `"custom-type"` || string(body["title"]) != `""` {
				t.Fatalf("create %s %s %#v", req.Method, req.URL, body)
			}
			return propertyCoreJSON(req, 201, `{"name":"passive","type":"custom-type","title":"","unknown":9007199254740993}`), nil
		case 2:
			if req.Method != "POST" || string(body["minimum"]) != "1.25" || string(body["maximum"]) != "9007199254740993" || string(body["readonly"]) != "false" || string(body["default"]) != "null" || string(body["description"]) != `""` {
				t.Fatalf("literal schema attrs %#v", body)
			}
			return propertyCoreJSON(req, 201, `{}`), nil
		case 3:
			if req.Method != "GET" || req.URL.EscapedPath() != base+"/"+url.PathEscape(name) || req.URL.Query().Get("resource_type") != "OS::Type,+&" || req.Body != nil {
				t.Fatalf("get %s", req.URL)
			}
			return propertyCoreJSON(req, 200, `{"name":"other","schema":"https://foreign.test"}`), nil
		case 4:
			if req.URL.RawQuery != "resource_type=" {
				t.Fatalf("explicit empty query %s", req.URL)
			}
			return propertyCoreJSON(req, 200, `{}`), nil
		case 5:
			if req.Method != "PUT" || req.URL.EscapedPath() != base+"/"+url.PathEscape(name) || len(body) != 3 || string(body["name"]) != `"new::name"` {
				t.Fatalf("fixed rename %s %#v", req.URL, body)
			}
			return propertyCoreJSON(req, 200, `{"name":"new::name"}`), nil
		case 6:
			if req.Method != "PUT" || len(body) != 3 || string(body["name"]) != `" child::名字 "` {
				t.Fatalf("replacement name default %#v", body)
			}
			return propertyCoreJSON(req, 200, `{}`), nil
		case 7:
			if req.Method != "DELETE" || req.URL.EscapedPath() != base+"/"+url.PathEscape(name) || req.Body != nil || req.URL.RawQuery != "" {
				t.Fatalf("delete %s", req.URL)
			}
			return propertyCoreJSON(req, 204, "opaque"), nil
		case 8:
			if req.Method != "DELETE" || req.URL.EscapedPath() != base || req.Body != nil {
				t.Fatalf("bulk %s", req.URL)
			}
			return propertyCoreJSON(req, 204, "all"), nil
		}
		t.Fatalf("unexpected request%d", calls)
		return nil, nil
	})
	api := New(client)
	scope, err := api.InNamespace(context.Background(), parent)
	if err != nil || calls != 0 || scope.NamespaceName() != parent || scope.RawClient() != client || api.RawClient() != client {
		t.Fatalf("constructor %+v %v calls%d", scope, err, calls)
	}
	value, err := scope.Create(context.Background(), name, WithCreateType("custom-type"), WithCreateTitle(""))
	if err != nil || value.Key != nil || value.Name == nil || *value.Name != "passive" || value.StatusCode != 201 || string(value.Body["unknown"]) != "9007199254740993" {
		t.Fatalf("create %+v %v", value, err)
	}
	if _, err = scope.Create(context.Background(), name, WithCreateType("number"), WithCreateTitle("title"), WithCreateDescription(""), WithCreateAttributes(map[string]any{"minimum": json.Number("1.25"), "maximum": json.Number("9007199254740993"), "readonly": false, "default": nil})); err != nil {
		t.Fatal(err)
	}
	if _, err = scope.Get(context.Background(), name, WithGetResourceType("OS::Type,+&")); err != nil {
		t.Fatal(err)
	}
	if _, err = scope.Get(context.Background(), name, WithGetResourceType("")); err != nil {
		t.Fatal(err)
	}
	if _, err = scope.Update(context.Background(), name, WithUpdateName("new::name"), WithUpdateType("string"), WithUpdateTitle("")); err != nil {
		t.Fatal(err)
	}
	if _, err = scope.Update(context.Background(), name, WithUpdateType("string"), WithUpdateTitle("")); err != nil {
		t.Fatal(err)
	}
	ack, err := scope.Delete(context.Background(), name)
	if err != nil || ack == nil || ack.Namespace != parent || ack.Name == nil || *ack.Name != name || string(ack.Body) != "opaque" {
		t.Fatalf("delete %+v %v", ack, err)
	}
	ack, err = scope.DeleteAll(context.Background())
	if err != nil || ack == nil || ack.Name != nil || ack.Namespace != parent || ack.StatusCode != 204 || calls != 8 {
		t.Fatalf("bulk %+v %v calls%d", ack, err, calls)
	}
}

func TestMetadefPropertiesCoreCanonicalPresenceAndAtomicDecoder(t *testing.T) {
	var value Property
	raw := []byte(`{"name":"","type":"unrecognized","title":"","description":null,"self":"foreign","schema":"s","created_at":"not-date","updated_at":"","links":false,"readonly":"passive","minimum":1.25,"maximum":9007199254740993,"default":null,"items":[1],"required":[null],"Key":"raw"}`)
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	if value.Key != nil || value.Name == nil || *value.Name != "" || value.Type == nil || *value.Type != "unrecognized" || value.Description != nil || value.CreatedAt == nil || *value.CreatedAt != "not-date" || value.Links != nil {
		t.Fatalf("passive model %+v", value)
	}
	if string(value.Body["maximum"]) != "9007199254740993" || string(value.Body["minimum"]) != "1.25" || string(value.Body["default"]) != "null" || string(value.Body["readonly"]) != `"passive"` {
		t.Fatal("raw definition narrowed")
	}
	*value.Name = "changed"
	value.Body["type"][1] = 'X'
	if string(value.Body["name"]) != `""` || *value.Type != "unrecognized" {
		t.Fatal("typed strings alias Body")
	}
	before := value
	for _, body := range []string{`null`, `[]`, `{"name":1}`, `{"type":false}`, `{"title":[]}`, `{"description":{}}`, `{"self":false}`, `{"schema":1}`, `{"created_at":1}`, `{"updated_at":true}`} {
		if err := json.Unmarshal([]byte(body), &value); err == nil {
			t.Fatalf("bad canonical %s", body)
		}
		if !reflect.DeepEqual(value, before) {
			t.Fatalf("partial assignment %s", body)
		}
	}
	if err := json.Unmarshal([]byte("{\"unknown\":\"\xff\"}"), &value); err == nil {
		t.Fatal("invalid UTF8 root accepted")
	}
	for _, body := range []string{`{}`, `{"name":null,"type":null,"title":null,"created_at":null}`} {
		var empty Property
		if err := json.Unmarshal([]byte(body), &empty); err != nil || empty.Name != nil || empty.Type != nil || empty.Title != nil || empty.CreatedAt != nil || empty.Body == nil {
			t.Fatalf("missing/null %+v %v", empty, err)
		}
	}
}

func TestMetadefPropertiesCoreCompletePreflight(t *testing.T) {
	requests, callbacks := 0, 0
	client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
		requests++
		return propertyCoreJSON(req, 200, `{}`), nil
	})
	api := New(client)
	for _, parent := range []string{"", ".", "..", "a/b", "a\\b", "a%b", "a?b", "a#b", "a\nb", string([]byte{0xff}), strings.Repeat("界", 81)} {
		if _, err := api.InNamespace(context.Background(), parent); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("bad parent%q %v", parent, err)
		}
	}
	scope := propertyCoreScope(t, client, "OS::parent")
	for _, name := range []string{"", ".", "..", "a/b", "a\\b", "a%b", "a?b", "a#b", "a\nb", string([]byte{0xff}), strings.Repeat("界", 81)} {
		if _, err := scope.Create(context.Background(), name, func(*CreateOpts) error { callbacks++; return nil }); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("bad child%q %v", name, err)
		}
	}
	if _, err := scope.Get(nil, "child", func(*GetOpts) error { callbacks++; return nil }); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("context-cause")
	cancel(cause)
	if _, err := api.InNamespace(ctx, "parent"); !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal(err)
	}
	if callbacks != 0 || requests != 0 {
		t.Fatalf("beforecallback callbacks%d requests%d", callbacks, requests)
	}
	checks := []func() error{
		func() error { _, e := scope.Create(context.Background(), "child"); return e },
		func() error { _, e := scope.Create(context.Background(), "child", WithCreateType("string")); return e },
		func() error {
			_, e := scope.Create(context.Background(), "child", WithCreateType(""), WithCreateTitle(""))
			return e
		},
		func() error { _, e := scope.Update(context.Background(), "child", WithUpdateTitle("title")); return e },
		func() error {
			_, e := scope.Update(context.Background(), "child", WithUpdateType("string"), WithUpdateTitle(""), WithUpdateName("a/b"))
			return e
		},
		func() error {
			_, e := scope.Get(context.Background(), "child", WithGetResourceType("line\n"))
			return e
		},
		func() error { _, e := scope.Delete(context.Background(), "child", nil); return e },
		func() error {
			_, e := scope.DeleteAll(context.Background(), WithDeleteAllHeader("Host", "foreign"))
			return e
		},
		func() error { _, e := scope.All(context.Background(), WithListMaxItems(-1)); return e },
	}
	for i, check := range checks {
		if err := check(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("check%d %v", i, err)
		}
	}
	badKey := string([]byte{0xff})
	for _, option := range []CreateOption{WithCreateAttributes(map[string]any{badKey: 1}), WithCreateAttribute(badKey, 1), WithCreateOpts(CreateOpts{Type: propertyOptionPointer("string"), Title: propertyOptionPointer(""), Attributes: map[string]json.RawMessage{badKey: json.RawMessage(`1`)}})} {
		if _, err := scope.Create(context.Background(), "child", WithCreateType("string"), WithCreateTitle(""), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("malformed key sent %v", err)
		}
	}
	marker := errors.New("marshal-cause")
	if _, err := scope.Create(context.Background(), "child", WithCreateType("string"), WithCreateTitle(""), WithCreateAttribute("arbitrary", propertyOptionMarshaler{cause: marker})); !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, marker) {
		t.Fatalf("encodingcause %v", err)
	}
	if requests != 0 {
		t.Fatalf("invalid sent%d", requests)
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

func TestMetadefPropertiesCoreOrderedDictionaryConsumption(t *testing.T) {
	body := `{"properties":{"z":{"name":"old"},"a":null,"z":{"name":"passive-last","title":"last","unknown":9007199254740993}},"next":123,"first":"foreign"}`
	calls, callbacks := 0, 0
	scope := propertyCoreScope(t, propertyCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.RawQuery != "" || req.Method != "GET" || req.Body != nil {
			t.Fatalf("finite query/body %s", req.URL)
		}
		response := propertyCoreJSON(req, 200, body)
		response.Header.Set("Link", "<https://foreign.test>; rel=next")
		return response, nil
	}), "parent")
	seq := scope.List(context.Background(), func(*ListOpts) error { callbacks++; return nil }, WithListMaxItems(1))
	if calls != 0 || callbacks != 0 {
		t.Fatal("iterator eager")
	}
	for iteration := 0; iteration < 2; iteration++ {
		count := 0
		for value, err := range seq {
			if err != nil || value.Key == nil || *value.Key != "z" || value.Name == nil || *value.Name != "passive-last" || *value.Title != "last" || string(value.Body["unknown"]) != "9007199254740993" {
				t.Fatalf("orderedduplicate %+v %v", value, err)
			}
			if _, exists := value.Body["Key"]; exists {
				t.Fatal("key injected into raw body")
			}
			*value.Key = "local"
			if *value.Name != "passive-last" {
				t.Fatal("key aliases Name")
			}
			count++
		}
		if count != 1 {
			t.Fatal(count)
		}
	}
	if calls != 2 || callbacks != 2 {
		t.Fatalf("reuse calls%d callbacks%d", calls, callbacks)
	}
	if rows, err := scope.All(context.Background()); rows != nil || err == nil {
		t.Fatalf("Allpartial %+v %v", rows, err)
	} else {
		propertyCoreProof(t, err, 200, body)
	}
	for value, err := range scope.List(context.Background()) {
		if value == nil || err != nil {
			t.Fatal(err)
		}
		break
	}
	for _, sample := range []struct {
		body string
		cap  int
		good bool
	}{
		{`{"properties":{"good":{},"bad":{},"bad":false}}`, 1, true},
		{`{"properties":{"good":{},"bad":{},"bad":false}}`, 2, false},
		{`{"properties":{"good":{},"bad":}}`, 1, false},
		{`{"properties":{},"next":false}`, 0, true},
		{`{}`, 1, false}, {`{"properties":null}`, 1, false}, {`{"properties":[]}`, 1, false},
		{"{\"properties\":{},\"unknown\":\"\xff\"}", 1, false},
	} {
		testScope := propertyCoreScope(t, propertyCoreClient(func(req *http.Request) (*http.Response, error) { return propertyCoreJSON(req, 200, sample.body), nil }), "parent")
		rows, err := testScope.All(context.Background(), WithListMaxItems(sample.cap))
		if sample.good {
			if err != nil || rows == nil {
				t.Fatalf("goodfinite %+v %v", rows, err)
			}
		} else {
			if rows != nil || err == nil {
				t.Fatalf("badfinite%s %+v %v", sample.body, rows, err)
			}
			propertyCoreProof(t, err, 200, sample.body)
		}
	}
	order := propertyCoreScope(t, propertyCoreClient(func(req *http.Request) (*http.Response, error) {
		return propertyCoreJSON(req, 200, `{"properties":{"z":{},"a":{},"m":{},"a":{"title":"last"}}}`), nil
	}), "parent")
	rows, err := order.All(context.Background())
	if err != nil || len(rows) != 3 || *rows[0].Key != "z" || *rows[1].Key != "a" || *rows[2].Key != "m" || *rows[1].Title != "last" {
		t.Fatalf("wireorder %+v %v", rows, err)
	}
}

func TestMetadefPropertiesCoreAcceptedBodiesAndMissingPolicy(t *testing.T) {
	readCause, closeCause := errors.New("read-cause"), errors.New("close-cause")
	t.Run("accepted typed model failure proof", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		custom := errors.New("custom-context")
		body := &propertyCoreBody{reader: &propertyCoreReader{data: `{"name":"partial"}`, err: readCause, after: func() { cancel(custom) }}, closeErr: closeCause}
		scope := propertyCoreScope(t, propertyCoreClient(func(req *http.Request) (*http.Response, error) { return propertyCoreHTTP(req, 201, body), nil }), "parent")
		value, err := scope.Create(ctx, "child", WithCreateType("string"), WithCreateTitle(""))
		if value != nil || body.closes != 1 || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, custom) {
			t.Fatalf("body ownership %+v %v closes%d", value, err, body.closes)
		}
		propertyCoreProof(t, err, 201, `{"name":"partial"}`)
	})
	for _, bulk := range []bool{false, true} {
		t.Run(fmt.Sprintf("opaque204 bulk%v", bulk), func(t *testing.T) {
			body := &propertyCoreBody{reader: strings.NewReader("opaque\x00body"), closeErr: closeCause}
			scope := propertyCoreScope(t, propertyCoreClient(func(req *http.Request) (*http.Response, error) { return propertyCoreHTTP(req, 204, body), nil }), "parent")
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
			proof := propertyCoreProof(t, err, 204, "opaque\x00body")
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
		client := propertyCoreClient(func(req *http.Request) (*http.Response, error) { return propertyCoreJSON(req, 404, "masked"), nil })
		client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			hooks++
			return retryCause
		}
		scope := propertyCoreScope(t, client, "parent")
		if ack, err := scope.Delete(context.Background(), "child"); ack != nil || err != nil || hooks != 0 {
			t.Fatalf("default %+v %v hooks%d", ack, err, hooks)
		}
		if ack, err := scope.Delete(context.Background(), "child", WithDeleteIgnoreMissing(false)); ack != nil || !errors.Is(err, retryCause) || hooks != 1 || !gophercloud.ResponseCodeIs(err, 404) {
			t.Fatalf("strict %+v %v hooks%d", ack, err, hooks)
		}
		if ack, err := scope.DeleteAll(context.Background()); ack != nil || !errors.Is(err, retryCause) || hooks != 2 {
			t.Fatalf("bulk strict %+v %v hooks%d", ack, err, hooks)
		}
		bad := &propertyCoreBody{reader: strings.NewReader("masked"), closeErr: closeCause}
		scope = propertyCoreScope(t, propertyCoreClient(func(req *http.Request) (*http.Response, error) { return propertyCoreHTTP(req, 404, bad), nil }), "parent")
		if ack, err := scope.Delete(context.Background(), "child"); ack != nil || !errors.Is(err, closeCause) || bad.closes != 1 {
			t.Fatalf("dirty404 %+v %v", ack, err)
		} else {
			propertyCoreProof(t, err, 404, "masked")
		}
		nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404}
		scope = propertyCoreScope(t, propertyCoreClient(func(*http.Request) (*http.Response, error) { return nil, nested }), "parent")
		if ack, err := scope.Delete(context.Background(), "child"); ack != nil || err == nil {
			t.Fatalf("nested404 %+v %v", ack, err)
		}
	})
}

func TestMetadefPropertiesCoreLifetimeSourceAndNativeRetryBoundaries(t *testing.T) {
	mutations := []func(*gophercloud.ServiceClient){func(c *gophercloud.ServiceClient) { c.Endpoint = "https://example.test/changed/" }, func(c *gophercloud.ServiceClient) { c.ResourceBase = "https://example.test/changed/" }, func(c *gophercloud.ServiceClient) { c.Microversion = "2.9" }, func(c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} }, func(c *gophercloud.ServiceClient) { c.Type = "compute" }}
	for i, mutate := range mutations {
		t.Run(fmt.Sprintf("lifetime%d", i), func(t *testing.T) {
			calls, callbacks := 0, 0
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return propertyCoreJSON(req, 200, `{}`), nil })
			scope := propertyCoreScope(t, client, "parent")
			mutate(client)
			if _, err := scope.Get(context.Background(), "child", func(*GetOpts) error { callbacks++; return nil }); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 {
				t.Fatalf("drift beforecallback %v calls%d callbacks%d", err, calls, callbacks)
			}
		})
	}
	t.Run("latest headers before each call auth live", func(t *testing.T) {
		calls := 0
		client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Header.Get("X-Source") != "latest" || req.Header.Get("X-Option") != "chosen" || req.Header.Get("X-Auth-Token") != "after" {
				t.Fatalf("headers %#v", req.Header)
			}
			return propertyCoreJSON(req, 200, `{}`), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "initial"}
		scope := propertyCoreScope(t, client, "parent")
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
		client := propertyCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return propertyCoreJSON(req, 200, `{}`), nil })
		scope := propertyCoreScope(t, client, "parent")
		if _, err := scope.Get(context.Background(), "child", func(*GetOpts) error { client.ResourceBase = "https://example.test/changed/"; return nil }); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatalf("callback drift %v calls%d", err, calls)
		}
	})
	t.Run("retry encoded body remains owned", func(t *testing.T) {
		calls := 0
		client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			return propertyCoreJSON(req, 503, "first"), nil
		})
		client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			o.JSONBody = map[string]string{"name": "changed"}
			return nil
		}
		value, err := propertyCoreScope(t, client, "parent").Create(context.Background(), "child", WithCreateType("string"), WithCreateTitle(""))
		if value != nil || calls != 1 || !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) {
			t.Fatalf("retry ownership %+v %v calls%d", value, err, calls)
		}
	})
	t.Run("expanded accepted status fails original gate", func(t *testing.T) {
		calls := 0
		client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return propertyCoreJSON(req, 503, "first"), nil
			}
			return propertyCoreJSON(req, 202, `{"name":"unexpected"}`), nil
		})
		client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			o.OkCodes = []int{202}
			return nil
		}
		value, err := propertyCoreScope(t, client, "parent").Create(context.Background(), "child", WithCreateType("string"), WithCreateTitle(""))
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || calls != 2 || !errors.As(err, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, []int{201}) || string(native.Body) != `{"name":"unexpected"}` {
			t.Fatalf("actual code gate %+v %v", native, err)
		}
	})
	t.Run("drift during accepted body keeps raw proof", func(t *testing.T) {
		var client *gophercloud.ServiceClient
		body := &propertyCoreBody{reader: &propertyCoreReader{data: `{}`, err: io.EOF, after: func() { client.ResourceBase = "https://example.test/changed/" }}}
		client = propertyCoreClient(func(req *http.Request) (*http.Response, error) { return propertyCoreHTTP(req, 200, body), nil })
		value, err := propertyCoreScope(t, client, "parent").Get(context.Background(), "child")
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || body.closes != 1 {
			t.Fatalf("body drift %+v %v", value, err)
		}
		propertyCoreProof(t, err, 200, `{}`)
	})
}
