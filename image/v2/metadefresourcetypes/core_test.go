package metadefresourcetypes

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type rtCoreTransport func(*http.Request) (*http.Response, error)

func (f rtCoreTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type rtCoreBody struct {
	reader   io.Reader
	closes   int
	closeErr error
}

func (b *rtCoreBody) Read(p []byte) (int, error) { return b.reader.Read(p) }
func (b *rtCoreBody) Close() error               { b.closes++; return b.closeErr }

type rtCoreReader struct {
	data  string
	err   error
	after func()
}

func (r *rtCoreReader) Read(p []byte) (int, error) {
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
func rtCoreClient(f rtCoreTransport) *gophercloud.ServiceClient {
	return &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{TokenID: "original", HTTPClient: http.Client{Transport: f}}, Type: "image", Endpoint: "https://example.test/catalog/", ResourceBase: "https://example.test/reverse/glance/v2/"}
}
func rtCoreHTTP(req *http.Request, status int, body io.ReadCloser) *http.Response {
	return &http.Response{Request: req, StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}, "X-Proof": {"original"}}, Body: body}
}
func rtCoreJSON(req *http.Request, status int, body string) *http.Response {
	return rtCoreHTTP(req, status, io.NopCloser(strings.NewReader(body)))
}
func rtCoreScope(t *testing.T, c *gophercloud.ServiceClient, parent string) *NamespaceScope {
	t.Helper()
	s, err := New(c).InNamespace(context.Background(), parent)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func rtCoreProof(t *testing.T, err error, status int, body string) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Proof") != "original" {
		t.Fatalf("accepted proof missing: %v %+v", err, proof)
	}
	return proof
}

func TestMetadefResourceTypesCoreFixedRoutesAndPassiveModels(t *testing.T) {
	parent, name := "OS::空 白", " child::名字 "
	calls := 0
	client := rtCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		global := "/reverse/glance/v2/metadefs/resource_types"
		collection := "/reverse/glance/v2/metadefs/namespaces/" + url.PathEscape(parent) + "/resource_types"
		if req.URL.RawQuery != "" || req.Header.Get("X-Auth-Token") != "original" {
			t.Fatalf("unexpected query/auth %s", req.URL)
		}
		switch calls {
		case 1:
			if req.Method != "GET" || req.URL.EscapedPath() != global || req.Body != nil {
				t.Fatal("global route")
			}
			return rtCoreJSON(req, 200, `{"resource_types":[{"name":"two"},{"name":"one","protected":true},{"name":"two"}],"next":"https://foreign.test/"}`), nil
		case 2, 3:
			var body map[string]string
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			expected := map[string]string{"name": name}
			if calls == 3 {
				expected["prefix"] = ""
				expected["properties_target"] = "line\n\x00"
			}
			if req.Method != "POST" || req.URL.EscapedPath() != collection || !reflect.DeepEqual(body, expected) {
				t.Fatalf("association create %s %+v", req.URL, body)
			}
			return rtCoreJSON(req, 201, `{"name":"server-choice","prefix":null,"created_at":"not a parsed date","unknown":9007199254740993,"links":false}`), nil
		case 4:
			if req.Method != "DELETE" || req.URL.EscapedPath() != collection+"/"+url.PathEscape(name) || req.Body != nil {
				t.Fatal("fixed deletion route")
			}
			return rtCoreJSON(req, 204, "opaque acknowledgement"), nil
		case 5:
			if req.Method != "GET" || req.URL.EscapedPath() != collection || req.Body != nil {
				t.Fatal("scoped list route")
			}
			return rtCoreJSON(req, 200, `{"resource_type_associations":[{"name":"other","prefix":"hw_","properties_target":""}],"links":false}`), nil
		default:
			t.Fatalf("extra HTTP%d", calls)
			return nil, nil
		}
	})
	api := New(client)
	scope, err := api.InNamespace(context.Background(), parent)
	if err != nil || calls != 0 || scope.NamespaceName() != parent || scope.RawClient() != client || api.RawClient() != client {
		t.Fatal("constructor should be HTTP-free", err)
	}
	types, err := api.All(context.Background())
	if err != nil || len(types) != 3 || *types[0].Name != "two" || *types[1].Name != "one" || *types[2].Name != "two" || string(types[1].Body["protected"]) != "true" {
		t.Fatalf("wire order %v %+v", err, types)
	}
	for _, opts := range [][]CreateOption{nil, {WithCreatePrefix(""), WithCreatePropertiesTarget("line\n\x00")}} {
		value, err := scope.Create(context.Background(), name, opts...)
		if err != nil || *value.Name != "server-choice" || value.Prefix != nil || *value.CreatedAt != "not a parsed date" || value.Links != nil || string(value.Body["unknown"]) != "9007199254740993" {
			t.Fatalf("passive create result %v %+v", err, value)
		}
		*value.Name = "changed"
		if string(value.Body["name"]) != `"server-choice"` {
			t.Fatal("typed name aliases raw metadata")
		}
	}
	ack, err := scope.Delete(context.Background(), name)
	if err != nil || ack.StatusCode != 204 || ack.Namespace != parent || *ack.Name != name || string(ack.Body) != "opaque acknowledgement" {
		t.Fatalf("actual ACK %v %+v", err, ack)
	}
	assocs, err := scope.All(context.Background())
	if err != nil || len(assocs) != 1 || *assocs[0].Name != "other" || *assocs[0].Prefix != "hw_" || *assocs[0].PropertiesTarget != "" || calls != 5 {
		t.Fatalf("finite scoped list %v %+v calls%d", err, assocs, calls)
	}
}

func TestMetadefResourceTypesCoreCanonicalPresenceAndAtomicDecoder(t *testing.T) {
	for _, data := range []string{`{}`, `{"name":null,"created_at":null,"updated_at":null}`} {
		var value ResourceType
		if err := json.Unmarshal([]byte(data), &value); err != nil || value.Name != nil || value.CreatedAt != nil || value.UpdatedAt != nil || value.Body == nil {
			t.Fatalf("nullable global: %s %v %+v", data, err, value)
		}
	}
	var value Association
	raw := []byte(`{"name":"","prefix":"","properties_target":"line\n","created_at":"literal","updated_at":"","links":{"foreign":"https://foreign.test"},"protected":[1],"number":123456789012345678901234567890}`)
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	if value.Name == nil || *value.Name != "" || value.Prefix == nil || *value.Prefix != "" || *value.PropertiesTarget != "line\n" || *value.CreatedAt != "literal" || value.UpdatedAt == nil || value.Links != nil || string(value.Body["number"]) != "123456789012345678901234567890" {
		t.Fatalf("canonical presence %+v", value)
	}
	raw[2] = 'X'
	if string(value.Body["name"]) != `""` {
		t.Fatal("decoder body aliases input")
	}
	for _, bad := range []string{`null`, `[]`, `{"name":1}`, `{"prefix":false}`, `{"properties_target":{}}`, `{"created_at":123}`, `{"updated_at":[]}`, "{\"unknown\":\"\xff\"}"} {
		old := "retained"
		target := Association{Name: &old, Metadata: resource.Metadata{Header: http.Header{"X-Retained": {"yes"}}, StatusCode: 201}}
		if err := json.Unmarshal([]byte(bad), &target); err == nil || target.Name != &old || *target.Name != "retained" || target.Header.Get("X-Retained") != "yes" || target.StatusCode != 201 {
			t.Fatalf("atomic invalid decode %q: %v %+v", bad, err, target)
		}
	}
	for _, bad := range []string{`{"name":false}`, `{"created_at":[]}`, `{"updated_at":3}`} {
		var global ResourceType
		if err := json.Unmarshal([]byte(bad), &global); err == nil {
			t.Fatal("global canonical type accepted", bad)
		}
	}
}

func TestMetadefResourceTypesCoreCompletePreflight(t *testing.T) {
	calls, callbacks := 0, 0
	client := rtCoreClient(func(*http.Request) (*http.Response, error) { calls++; t.Fatal("preflight made HTTP"); return nil, nil })
	api := New(client)
	ctx := context.Background()
	for _, parent := range []string{"", ".", "..", "x/y", "x\\y", "x%y", "x?y", "x#y", "line\n", strings.Repeat("界", 81), string([]byte{0xff})} {
		if _, err := api.InNamespace(ctx, parent); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("invalid parent %q %v", parent, err)
		}
	}
	scope := rtCoreScope(t, client, "parent")
	for _, name := range []string{"", "..", "bad/name", strings.Repeat("界", 81), string([]byte{0xff})} {
		if _, err := scope.Create(ctx, name, func(*CreateOpts) error { callbacks++; return nil }); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("invalid create name %q %v", name, err)
		}
		if _, err := scope.Delete(ctx, name, func(*DeleteOpts) error { callbacks++; return nil }); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("invalid delete name %q %v", name, err)
		}
	}
	for _, opts := range [][]CreateOption{{nil}, {WithCreatePrefix(strings.Repeat("界", 81))}, {WithCreatePropertiesTarget(string([]byte{0xff}))}, {WithCreateHeader("Content-Type", "text/plain")}} {
		if _, err := scope.Create(ctx, "child", opts...); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if _, err := scope.Delete(ctx, "child", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	for _, opts := range [][]ListOption{{nil}, {WithListMaxItems(-1)}, {WithListHeader("X-Auth-Token", "foreign")}} {
		if _, err := api.All(ctx, opts...); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
		if _, err := scope.All(ctx, opts...); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	cause := errors.New("callback")
	if _, err := scope.Create(ctx, "child", func(*CreateOpts) error { return cause }); !errors.Is(err, cause) {
		t.Fatal(err)
	}
	if _, err := api.All(nil, func(*ListOpts) error { callbacks++; return nil }); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancelCause(ctx)
	custom := errors.New("custom cancel")
	cancel(custom)
	if _, err := scope.Delete(canceled, "child", func(*DeleteOpts) error { callbacks++; return nil }); !errors.Is(err, context.Canceled) || !errors.Is(err, custom) {
		t.Fatal(err)
	}
	client.MoreHeaders = map[string]string{"Authorization": "secret"}
	if _, err := api.All(ctx, func(*ListOpts) error { callbacks++; return nil }); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	client.MoreHeaders = nil
	client.Type = "compute"
	if _, err := New(client).InNamespace(ctx, "parent"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	if calls != 0 || callbacks != 0 {
		t.Fatalf("complete preflight calls%d callbacks%d", calls, callbacks)
	}
}

func TestMetadefResourceTypesCoreFiniteConsumptionAndRootEvidence(t *testing.T) {
	t.Run("cap and break skip unused invalid row", func(t *testing.T) {
		calls, callbacks := 0, 0
		client := rtCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.URL.RawQuery != "" {
				t.Fatal("local cap sent query")
			}
			return rtCoreJSON(req, 200, `{"resource_types":[{"name":"one"},{"name":false}],"next":false}`), nil
		})
		api := New(client)
		option := ListOption(func(c *ListOpts) error { callbacks++; c.MaxItems = 1; return nil })
		seq := api.List(context.Background(), option)
		if calls != 0 || callbacks != 0 {
			t.Fatal("iterator eager")
		}
		for range 2 {
			count := 0
			for value, err := range seq {
				if err != nil || *value.Name != "one" {
					t.Fatal(err)
				}
				count++
			}
			if count != 1 {
				t.Fatal(count)
			}
		}
		for _, err := range api.List(context.Background()) {
			if err != nil {
				t.Fatal(err)
			}
			break
		}
		if calls != 3 || callbacks != 2 {
			t.Fatalf("lazy repeat %d/%d", calls, callbacks)
		}
		values, err := api.All(context.Background())
		if values != nil || err == nil {
			t.Fatalf("All late error retained partial %v %+v", err, values)
		}
		rtCoreProof(t, err, 200, `{"resource_types":[{"name":"one"},{"name":false}],"next":false}`)
	})
	t.Run("scoped root validation before consumption", func(t *testing.T) {
		for _, raw := range []string{`null`, `[]`, `{}`, `{"resource_type_associations":null}`, `{"resource_type_associations":{}}`, `{"resource_type_associations":[{}],"bad":`, "{\"resource_type_associations\":[],\"bad\":\"\xff\"}"} {
			client := rtCoreClient(func(req *http.Request) (*http.Response, error) { return rtCoreJSON(req, 200, raw), nil })
			scope := rtCoreScope(t, client, "parent")
			values, err := scope.All(context.Background(), WithListMaxItems(1))
			if values != nil || err == nil {
				t.Fatalf("root %q %v %+v", raw, err, values)
			}
			rtCoreProof(t, err, 200, raw)
		}
	})
	t.Run("empty and raw rows independent", func(t *testing.T) {
		calls := 0
		client := rtCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if strings.Contains(req.URL.Path, "namespaces") {
				return rtCoreJSON(req, 200, `{"resource_type_associations":[]}`), nil
			}
			return rtCoreJSON(req, 200, `{"resource_types":[{"name":"same"},{"name":"same"}]}`), nil
		})
		api := New(client)
		empty, err := rtCoreScope(t, client, "parent").All(context.Background())
		if err != nil || empty == nil || len(empty) != 0 {
			t.Fatalf("empty %v %+v", err, empty)
		}
		values, err := api.All(context.Background(), WithListMaxItems(20))
		if err != nil || len(values) != 2 || calls != 2 {
			t.Fatal("server finite list must not repeat", err, calls)
		}
		values[0].Header.Set("X-Proof", "changed")
		values[0].Body["name"][1] = 'X'
		if values[1].Header.Get("X-Proof") != "original" || *values[1].Name != "same" || string(values[1].Body["name"]) != `"same"` {
			t.Fatal("row evidence aliases sibling")
		}
	})
}

func TestMetadefResourceTypesCoreAcceptedBodiesAndMissingPolicy(t *testing.T) {
	t.Run("accepted204 all causes and independent ACK", func(t *testing.T) {
		readErr, closeErr, custom := errors.New("read"), errors.New("close"), errors.New("cancel")
		ctx, cancel := context.WithCancelCause(context.Background())
		body := &rtCoreBody{reader: &rtCoreReader{data: "opaque", err: readErr, after: func() { cancel(custom) }}, closeErr: closeErr}
		client := rtCoreClient(func(req *http.Request) (*http.Response, error) { return rtCoreHTTP(req, 204, body), nil })
		ack, err := rtCoreScope(t, client, "parent").Delete(ctx, "child")
		if ack == nil || ack.StatusCode != 204 || *ack.Name != "child" || string(ack.Body) != "opaque" || body.closes != 1 {
			t.Fatalf("ACK %v %+v closes%d", err, ack, body.closes)
		}
		for _, cause := range []error{readErr, closeErr, context.Canceled, custom} {
			if !errors.Is(err, cause) {
				t.Fatalf("lost cause %v in%v", cause, err)
			}
		}
		proof := rtCoreProof(t, err, 204, "opaque")
		ack.Body[0] = 'X'
		ack.Header.Set("X-Proof", "changed")
		if string(proof.Body) != "opaque" || proof.Header.Get("X-Proof") != "original" {
			t.Fatal("ACK and error proof alias")
		}
	})
	t.Run("201 failure yields nil typed with proof", func(t *testing.T) {
		for _, bodyFailure := range []bool{false, true} {
			closeErr := errors.New("close")
			body := &rtCoreBody{reader: strings.NewReader(`{"name":false}`)}
			if bodyFailure {
				body.reader = strings.NewReader(`{"name":"child"}`)
				body.closeErr = closeErr
			}
			raw := `{"name":false}`
			if bodyFailure {
				raw = `{"name":"child"}`
			}
			client := rtCoreClient(func(req *http.Request) (*http.Response, error) { return rtCoreHTTP(req, 201, body), nil })
			value, err := rtCoreScope(t, client, "parent").Create(context.Background(), "child")
			if value != nil || err == nil || body.closes != 1 {
				t.Fatalf("accepted create failure %v %+v", err, value)
			}
			rtCoreProof(t, err, 201, raw)
			if bodyFailure && !errors.Is(err, closeErr) {
				t.Fatal(err)
			}
		}
	})
	t.Run("default clean404 only", func(t *testing.T) {
		for _, failure := range []bool{false, true} {
			retries := 0
			closeErr := errors.New("close404")
			body := &rtCoreBody{reader: strings.NewReader(`hidden namespace`)}
			if failure {
				body.closeErr = closeErr
			}
			client := rtCoreClient(func(req *http.Request) (*http.Response, error) { return rtCoreHTTP(req, 404, body), nil })
			client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return errors.New("should bypass for default404")
			}
			ack, err := rtCoreScope(t, client, "parent").Delete(context.Background(), "child")
			if ack != nil || body.closes != 1 || retries != 0 {
				t.Fatal("owned404 fabricatedACK/retry", ack, body.closes, retries)
			}
			if failure {
				if !errors.Is(err, closeErr) {
					t.Fatal(err)
				}
				rtCoreProof(t, err, 404, `hidden namespace`)
			} else if err != nil {
				t.Fatal(err)
			}
		}
	})
	t.Run("strict and nested404 remain errors", func(t *testing.T) {
		client := rtCoreClient(func(req *http.Request) (*http.Response, error) { return rtCoreJSON(req, 404, `missing`), nil })
		ack, err := rtCoreScope(t, client, "parent").Delete(context.Background(), "child", WithDeleteIgnoreMissing(false))
		if ack != nil || !gophercloud.ResponseCodeIs(err, 404) {
			t.Fatalf("strict404 %v %+v", err, ack)
		}
		nested := &gophercloud.ErrUnexpectedResponseCode{Actual: 404}
		client = rtCoreClient(func(*http.Request) (*http.Response, error) { return nil, nested })
		ack, err = rtCoreScope(t, client, "parent").Delete(context.Background(), "child")
		if ack != nil || err == nil || !errors.Is(err, nested) {
			t.Fatalf("transport404 became missing %v", err)
		}
	})
}

func TestMetadefResourceTypesCoreSourceLifetimeAndNativePolicies(t *testing.T) {
	t.Run("scope lifetime before callback", func(t *testing.T) {
		for _, mutate := range []func(*API, *gophercloud.ServiceClient){func(a *API, c *gophercloud.ServiceClient) { a.client = rtCoreClient(nil) }, func(_ *API, c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} }, func(_ *API, c *gophercloud.ServiceClient) { c.Endpoint = "https://other.test/" }, func(_ *API, c *gophercloud.ServiceClient) { c.ResourceBase += "changed/" }, func(_ *API, c *gophercloud.ServiceClient) { c.Microversion = "2.3" }, func(_ *API, c *gophercloud.ServiceClient) { c.Type = "compute" }} {
			calls, callbacks := 0, 0
			client := rtCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			api := New(client)
			scope, err := api.InNamespace(context.Background(), "parent")
			if err != nil {
				t.Fatal(err)
			}
			mutate(api, client)
			if _, err = scope.Create(context.Background(), "child", func(*CreateOpts) error { callbacks++; return nil }); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || callbacks != 0 {
				t.Fatalf("retarget %v %d/%d", err, calls, callbacks)
			}
		}
	})
	t.Run("latest headers captured before callback", func(t *testing.T) {
		calls := 0
		client := rtCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			want := "before"
			if calls == 2 {
				want = "next"
			}
			if req.Header.Get("X-Caller") != want {
				t.Fatalf("owned header %q wanted%q", req.Header.Get("X-Caller"), want)
			}
			return rtCoreJSON(req, 200, `{"resource_types":[]}`), nil
		})
		api := New(client)
		client.MoreHeaders = map[string]string{"X-Caller": "before"}
		seq := api.List(context.Background(), func(*ListOpts) error { client.MoreHeaders["X-Caller"] = "after"; return nil })
		for _, err := range seq {
			if err != nil {
				t.Fatal(err)
			}
		}
		client.MoreHeaders["X-Caller"] = "next"
		if _, err := api.All(context.Background()); err != nil || calls != 2 {
			t.Fatal(err, calls)
		}
	})
	t.Run("source change during accepted read and after row", func(t *testing.T) {
		var client *gophercloud.ServiceClient
		client = rtCoreClient(func(req *http.Request) (*http.Response, error) {
			b := &rtCoreBody{reader: &rtCoreReader{data: `{"name":"server"}`, err: io.EOF, after: func() { client.Endpoint = "https://changed.test/" }}}
			return rtCoreHTTP(req, 201, b), nil
		})
		value, err := rtCoreScope(t, client, "parent").Create(context.Background(), "child")
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err, value)
		}
		rtCoreProof(t, err, 201, `{"name":"server"}`)
		client = rtCoreClient(func(req *http.Request) (*http.Response, error) {
			return rtCoreJSON(req, 200, `{"resource_type_associations":[{"name":"one"},{"name":"two"}]}`), nil
		})
		scope := rtCoreScope(t, client, "parent")
		rows, failures := 0, 0
		for _, err := range scope.List(context.Background()) {
			if err != nil {
				failures++
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				rtCoreProof(t, err, 200, `{"resource_type_associations":[{"name":"one"},{"name":"two"}]}`)
				continue
			}
			rows++
			client.MoreHeaders = map[string]string{"X-Auth-Token": "foreign"}
		}
		if rows != 1 || failures != 1 {
			t.Fatal(rows, failures)
		}
	})
	t.Run("live auth and native prebody retry", func(t *testing.T) {
		calls, retries := 0, 0
		client := rtCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Header.Get("X-Auth-Token") != "new-token" {
				t.Fatal("auth snapshot instead of live")
			}
			if calls == 1 {
				return rtCoreJSON(req, 503, "busy"), nil
			}
			if req.Header.Get("X-Advanced") != "native" {
				t.Fatal("configured advanced native header policy lost")
			}
			return rtCoreJSON(req, 201, `{"name":"server"}`), nil
		})
		scope := rtCoreScope(t, client, "parent")
		client.ProviderClient.TokenID = "new-token"
		client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			retries++
			if o.MoreHeaders == nil {
				o.MoreHeaders = map[string]string{}
			}
			o.MoreHeaders["X-Advanced"] = "native"
			return nil
		}
		value, err := scope.Create(context.Background(), "child")
		if err != nil || *value.Name != "server" || calls != 2 || retries != 1 {
			t.Fatal(err, calls, retries)
		}
	})
	t.Run("native request ownership and actual status", func(t *testing.T) {
		for _, expand := range []bool{false, true} {
			calls := 0
			client := rtCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return rtCoreJSON(req, 503, "busy"), nil
				}
				return rtCoreJSON(req, 202, "opaque202"), nil
			})
			client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				if expand {
					o.OkCodes = append(o.OkCodes, 202)
				} else {
					o.JSONBody = map[string]string{"name": "changed"}
				}
				return nil
			}
			value, err := rtCoreScope(t, client, "parent").Create(context.Background(), "child")
			if value != nil || err == nil {
				t.Fatal("ownership accepted")
			}
			if expand {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, []int{201}) || string(native.Body) != "opaque202" || calls != 2 {
					t.Fatalf("actualstatus %v %+v calls%d", err, native, calls)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || calls != 1 {
				t.Fatal(err, calls)
			}
		}
	})
}
