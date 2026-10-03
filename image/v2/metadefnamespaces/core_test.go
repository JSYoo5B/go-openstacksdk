package metadefnamespaces

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

type namespaceCoreTransport func(*http.Request) (*http.Response, error)

func (f namespaceCoreTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type namespaceCoreBody struct {
	reader   io.Reader
	closes   int
	closeErr error
}

func (b *namespaceCoreBody) Read(p []byte) (int, error) { return b.reader.Read(p) }
func (b *namespaceCoreBody) Close() error               { b.closes++; return b.closeErr }

type namespaceCoreReader struct {
	data  string
	err   error
	after func()
}

func (r *namespaceCoreReader) Read(p []byte) (int, error) {
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
func namespaceCoreClient(transport namespaceCoreTransport) *gophercloud.ServiceClient {
	return &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{TokenID: "before", HTTPClient: http.Client{Transport: transport}}, Type: "image", Endpoint: "https://example.test/catalog/", ResourceBase: "https://example.test/reverse/glance/v2/"}
}
func namespaceCoreHTTP(req *http.Request, status int, body io.ReadCloser) *http.Response {
	return &http.Response{Request: req, StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}, "X-Proof": {"original"}}, Body: body}
}
func namespaceCoreJSON(req *http.Request, status int, body string) *http.Response {
	return namespaceCoreHTTP(req, status, io.NopCloser(strings.NewReader(body)))
}
func namespaceCoreProof(t *testing.T, err error, status int, body string) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Proof") != "original" {
		t.Fatalf("missing exact proof: %v, %+v", err, proof)
	}
	return proof
}

func TestMetadefNamespacesCoreFixedRoutesAndReplacementEvidence(t *testing.T) {
	namespace := "OS::Compute::空 白"
	var requests int
	client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
		requests++
		path := "/reverse/glance/v2/metadefs/namespaces"
		var body map[string]any
		if req.Body != nil {
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
		}
		switch requests {
		case 1:
			if req.Method != http.MethodPost || req.URL.EscapedPath() != path || !reflect.DeepEqual(body, map[string]any{"namespace": namespace}) {
				t.Fatalf("create defaults: %s %s %#v", req.Method, req.URL, body)
			}
			return namespaceCoreJSON(req, 201, `{"namespace":"server-name","protected":false,"unknown":9007199254740993}`), nil
		case 2:
			expected := map[string]any{"namespace": namespace, "display_name": "", "description": "line1\nline2", "visibility": "public", "owner": "", "protected": false}
			if req.Method != http.MethodPost || !reflect.DeepEqual(body, expected) {
				t.Fatalf("explicit scalar body: %#v", body)
			}
			return namespaceCoreJSON(req, 201, `{}`), nil
		case 3:
			if req.Method != http.MethodGet || req.URL.EscapedPath() != path+"/"+url.PathEscape(namespace) || req.URL.Query().Get("resource_type") != "OS::Thing,+&" || body != nil {
				t.Fatalf("literal GET: %s %#v", req.URL, body)
			}
			return namespaceCoreJSON(req, 200, `{"namespace":"passive","self":"https://foreign.test/a"}`), nil
		case 4:
			if req.Method != http.MethodPut || req.URL.EscapedPath() != path+"/"+url.PathEscape(namespace) || !reflect.DeepEqual(body, map[string]any{"namespace": "renamed::名字"}) {
				t.Fatalf("rename route/body: %s %#v", req.URL, body)
			}
			return namespaceCoreJSON(req, 200, `{"namespace":"renamed::名字"}`), nil
		case 5:
			if req.Method != http.MethodPut || !reflect.DeepEqual(body, map[string]any{"namespace": namespace}) {
				t.Fatalf("replacement default identity: %#v", body)
			}
			return namespaceCoreJSON(req, 200, `{}`), nil
		case 6:
			if req.Method != http.MethodDelete || req.URL.EscapedPath() != path+"/"+url.PathEscape(namespace) || req.URL.RawQuery != "" || req.Body != nil {
				t.Fatalf("DELETE: %s %s", req.Method, req.URL)
			}
			return namespaceCoreJSON(req, 204, ""), nil
		}
		t.Fatalf("unexpected request %d", requests)
		return nil, nil
	})
	api := New(client)
	value, err := api.Create(context.Background(), namespace)
	if err != nil || value.Namespace == nil || *value.Namespace != "server-name" || value.StatusCode != 201 || string(value.Body["unknown"]) != "9007199254740993" {
		t.Fatalf("create result: %+v %v", value, err)
	}
	if _, err = api.Create(context.Background(), namespace, WithCreateDisplayName(""), WithCreateDescription("line1\nline2"), WithCreateVisibility("public"), WithCreateOwner(""), WithCreateProtected(false)); err != nil {
		t.Fatal(err)
	}
	if _, err = api.Get(context.Background(), namespace, WithGetResourceType("OS::Thing,+&")); err != nil {
		t.Fatal(err)
	}
	if _, err = api.Update(context.Background(), namespace, WithUpdateNamespace("renamed::名字")); err != nil {
		t.Fatal(err)
	}
	if _, err = api.Update(context.Background(), namespace); err != nil {
		t.Fatal(err)
	}
	ack, err := api.Delete(context.Background(), namespace)
	if err != nil || ack == nil || ack.Namespace != namespace || ack.StatusCode != 204 || ack.Header.Get("X-Proof") != "original" || requests != 6 {
		t.Fatalf("ack: %+v %v requests%d", ack, err, requests)
	}
}

func TestMetadefNamespacesCoreCanonicalPresenceAndAtomicDecoder(t *testing.T) {
	data := []byte(`{"namespace":"","display_name":null,"description":"d","visibility":"unexpected-passive","owner":"","protected":false,"created_at":"unparsed-date","updated_at":"","self":"foreign","schema":"s","links":false,"properties":[1],"objects":null,"tags":{},"resource_type_associations":"raw","unknown":9007199254740993}`)
	var value Namespace
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	if value.Namespace == nil || *value.Namespace != "" || value.DisplayName != nil || value.IsProtected == nil || *value.IsProtected || value.CreatedAt == nil || *value.CreatedAt != "unparsed-date" || value.UpdatedAt == nil || value.Links != nil {
		t.Fatalf("presence/passive metadata: %+v", value)
	}
	if string(value.Body["unknown"]) != "9007199254740993" || string(value.Body["links"]) != "false" {
		t.Fatalf("raw precision: %#v", value.Body)
	}
	*value.Namespace = "changed"
	if string(value.Body["namespace"]) != `""` {
		t.Fatal("typed value aliases raw body")
	}
	value.Body["description"][1] = 'X'
	if *value.Description != "d" {
		t.Fatal("raw body aliases typed value")
	}
	before := value
	for _, body := range []string{`null`, `[]`, `{"namespace":1}`, `{"display_name":false}`, `{"description":{}}`, `{"visibility":[]}`, `{"owner":1}`, `{"self":false}`, `{"schema":1}`, `{"protected":"false"}`, `{"created_at":1}`, `{"updated_at":true}`} {
		if err := json.Unmarshal([]byte(body), &value); err == nil {
			t.Fatalf("accepted malformed canonical field: %s", body)
		}
		if !reflect.DeepEqual(value, before) {
			t.Fatalf("decoder partially changed value on %s", body)
		}
	}
	if err := json.Unmarshal(append([]byte(`{"unknown":"`), append([]byte{0xff}, []byte(`"}`)...)...), &value); err == nil {
		t.Fatal("accepted invalid root UTF8")
	}
	for _, body := range []string{`{}`, `{"namespace":null,"protected":null,"created_at":null}`} {
		var empty Namespace
		if err := json.Unmarshal([]byte(body), &empty); err != nil || empty.Namespace != nil || empty.IsProtected != nil || empty.CreatedAt != nil || empty.Body == nil {
			t.Fatalf("missing/null: %+v %v", empty, err)
		}
	}
}

func TestMetadefNamespacesCoreCompletePreflight(t *testing.T) {
	calls := 0
	client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		return namespaceCoreJSON(req, 200, `{}`), nil
	})
	api := New(client)
	for _, identity := range []string{"", ".", "..", "a/b", "a\\b", "a%2Fb", "a?b", "a#b", "a\n", string([]byte{0xff}), strings.Repeat("界", 81)} {
		ran := 0
		_, err := api.Get(context.Background(), identity, func(*GetOpts) error { ran++; return nil })
		if !errors.Is(err, resource.ErrInvalidOption) || ran != 0 {
			t.Fatalf("identity preflight %q: %v callbacks%d", identity, err, ran)
		}
	}
	ran := 0
	if _, err := api.Create(nil, "valid", func(*CreateOpts) error { ran++; return nil }); !errors.Is(err, resource.ErrInvalidOption) || ran != 0 {
		t.Fatalf("nilctx %v callbacks%d", err, ran)
	}
	custom := errors.New("cancel-preflight")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(custom)
	if _, err := api.Update(ctx, "valid"); !errors.Is(err, context.Canceled) || !errors.Is(err, custom) {
		t.Fatal(err)
	}
	if _, err := (*API)(nil).Delete(context.Background(), "valid"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	checks := []func() error{
		func() error { _, err := api.Create(context.Background(), "valid", nil); return err },
		func() error {
			_, err := api.Create(context.Background(), "valid", WithCreateVisibility(""))
			return err
		},
		func() error {
			_, err := api.Create(context.Background(), "valid", WithCreateDisplayName(strings.Repeat("界", 81)))
			return err
		},
		func() error {
			_, err := api.Create(context.Background(), "valid", WithCreateDescription(strings.Repeat("x", 501)))
			return err
		},
		func() error {
			_, err := api.Create(context.Background(), "valid", WithCreateOwner(strings.Repeat("界", 256)))
			return err
		},
		func() error {
			_, err := api.Update(context.Background(), "valid", WithUpdateNamespace("a/b"))
			return err
		},
		func() error { _, err := api.Get(context.Background(), "valid", WithGetResourceType("a\t")); return err },
		func() error {
			_, err := api.Delete(context.Background(), "valid", WithDeleteHeader("Authorization", "x"))
			return err
		},
		func() error { _, err := api.All(context.Background(), WithListLimit(-1)); return err },
		func() error { _, err := api.All(context.Background(), WithListMaxItems(-1)); return err },
		func() error { _, err := api.All(context.Background(), WithListMarker("a/b")); return err },
		func() error { _, err := api.All(context.Background(), WithListSortDir("DESC")); return err },
		func() error { _, err := api.All(context.Background(), WithListSortKey("x\n")); return err },
		func() error {
			_, err := api.All(context.Background(), WithListResourceTypes(string([]byte{0xff})))
			return err
		},
	}
	for index, check := range checks {
		if err := check(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("option %d %v", index, err)
		}
	}
	client.Type = "compute"
	if _, err := api.Get(context.Background(), "valid"); !errors.Is(err, resource.ErrUnsupported) {
		t.Fatal(err)
	}
	client.Type = "image"
	client.MoreHeaders = map[string]string{"Host": "foreign"}
	if _, err := api.Get(context.Background(), "valid"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("preflight sent %d requests", calls)
	}
}

func TestMetadefNamespacesCoreAdvertisedPaginationAndCaps(t *testing.T) {
	t.Run("version route and exact filters", func(t *testing.T) {
		calls := 0
		client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			expected := url.Values{"limit": {"0"}, "visibility": {"public"}, "sort_key": {"literal-column"}, "sort_dir": {"asc"}, "resource_types": {"OS::A, OS::B"}}
			if calls == 2 {
				expected.Set("marker", "OS::first")
			}
			if req.URL.EscapedPath() != "/reverse/glance/v2/metadefs/namespaces" || !reflect.DeepEqual(req.URL.Query(), expected) {
				t.Fatalf("paging query %d %s", calls, req.URL)
			}
			if calls == 1 {
				return namespaceCoreJSON(req, 200, `{"namespaces":[{"namespace":"first"}],"next":"/v2/metadefs/namespaces?limit=0&visibility=public&sort_key=literal-column&sort_dir=asc&resource_types=OS%3A%3AA%2C+OS%3A%3AB&marker=OS%3A%3Afirst","first":"foreign"}`), nil
			}
			return namespaceCoreJSON(req, 200, `{"namespaces":[{"namespace":"last"}],"next":null}`), nil
		})
		values, err := New(client).All(context.Background(), WithListLimit(0), WithListVisibility("public"), WithListSortKey("literal-column"), WithListSortDir("asc"), WithListResourceTypes("OS::A, OS::B"))
		if err != nil || len(values) != 2 || calls != 2 {
			t.Fatalf("values%d calls%d %v", len(values), calls, err)
		}
		values[0].Header.Set("X-Proof", "changed")
		if values[1].Header.Get("X-Proof") != "original" {
			t.Fatal("row headers alias")
		}
	})
	t.Run("exact captured encoded prefix", func(t *testing.T) {
		for _, prefix := range []string{"/reverse%20proxy/glance/v2/", "/代理/glance/v2/"} {
			calls := 0
			var collection string
			client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.EscapedPath() != collection {
					t.Fatalf("encoded route changed: %s", req.URL)
				}
				if calls == 1 {
					return namespaceCoreJSON(req, 200, fmt.Sprintf(`{"namespaces":[{}],"next":%q}`, "https://example.test"+collection+"?marker=next")), nil
				}
				return namespaceCoreJSON(req, 200, `{"namespaces":[]}`), nil
			})
			client.ResourceBase = "https://example.test" + prefix
			parsed, _ := url.Parse(client.ServiceURL("metadefs", "namespaces"))
			collection = parsed.EscapedPath()
			rows, err := New(client).All(context.Background())
			if err != nil || len(rows) != 1 || calls != 2 {
				t.Fatalf("exact captured escaped prefix %q: calls%d %v", prefix, calls, err)
			}
		}
		for _, link := range []string{"https://example.test/reverse%20proxy/glance/v2/metadefs/%6eamespaces?marker=x", "https://example.test/reverse%20proxy/glance/v2/metadefs/%2e%2e/namespaces?marker=x", "https://example.test/reverse%20proxy/glance/v2/metadefs/namespaces?marker=x"} {
			calls := 0
			client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return namespaceCoreJSON(req, 200, fmt.Sprintf(`{"namespaces":[{}],"next":%q}`, link)), nil
			})
			client.ResourceBase = "https://example.test/reverse%20proxy/glance/v2/"
			// The last case uses an alternate encoding of the source prefix.
			if strings.HasSuffix(link, "namespaces?marker=x") && !strings.Contains(link, "%6e") && !strings.Contains(link, "%2e") {
				client.ResourceBase = "https://example.test/reverse%20pro%78y/glance/v2/"
			}
			rows, err := New(client).All(context.Background())
			if rows != nil || err == nil || calls != 1 {
				t.Fatalf("escaped alias %q accepted: %v", link, err)
			}
		}
	})
	t.Run("unused rows and next", func(t *testing.T) {
		body := `{"namespaces":[{"namespace":"good"},{"namespace":1}],"next":false}`
		calls := 0
		api := New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.URL.RawQuery != "" {
				t.Fatalf("cap generated query %s", req.URL)
			}
			return namespaceCoreJSON(req, 200, body), nil
		}))
		rows, err := api.All(context.Background(), WithListMaxItems(1))
		if err != nil || len(rows) != 1 || calls != 1 {
			t.Fatalf("cap %d %v", len(rows), err)
		}
		for value, err := range api.List(context.Background()) {
			if err != nil || value == nil {
				t.Fatal(err)
			}
			break
		}
		rows, err = api.All(context.Background())
		if rows != nil || err == nil {
			t.Fatalf("late row must discard partials %+v %v", rows, err)
		}
		namespaceCoreProof(t, err, 200, body)
		singleBody := `{"namespaces":[{}],"next":false}`
		api = New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) { return namespaceCoreJSON(req, 200, singleBody), nil }))
		if rows, err = api.All(context.Background(), WithListSinglePage(true)); err != nil || len(rows) != 1 {
			t.Fatal(err)
		}
		if rows, err = api.All(context.Background()); rows != nil || err == nil {
			t.Fatalf("malformed consumed next: %+v %v", rows, err)
		}
		namespaceCoreProof(t, err, 200, singleBody)
	})
	t.Run("finite empty and passive links", func(t *testing.T) {
		for _, body := range []string{`{"namespaces":[],"next":false}`, `{"namespaces":[{},{}],"links":[{"rel":"next","href":"https://foreign.test"}],"first":false}`} {
			calls := 0
			api := New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return namespaceCoreJSON(req, 200, body), nil
			}))
			rows, err := api.All(context.Background(), WithListMaxItems(20))
			if err != nil || rows == nil || calls != 1 {
				t.Fatalf("finite: calls%d %v", calls, err)
			}
		}
	})
	t.Run("unsafe and cyclic next retains whole proof", func(t *testing.T) {
		links := []string{"https://foreign.test/reverse/glance/v2/metadefs/namespaces?marker=x", "//example.test/reverse/glance/v2/metadefs/namespaces?marker=x", "/v2/metadefs/objects?marker=x", "/v2/metadefs/../metadefs/namespaces?marker=x", "/v2/metadefs/%6eamespaces?marker=x", "?marker=x&new=1", "?marker=x&marker=y", "?marker=", "?marker=a%2Fb", "?marker=x#", "?marker=start"}
		for _, link := range links {
			calls := 0
			body := fmt.Sprintf(`{"namespaces":[{}],"next":%q}`, link)
			api := New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return namespaceCoreJSON(req, 200, body), nil
			}))
			rows, err := api.All(context.Background(), WithListMarker("start"))
			if err == nil || rows != nil || calls != 1 {
				t.Fatalf("unsafe link %q rows%v calls%d err%v", link, rows, calls, err)
			}
			namespaceCoreProof(t, err, 200, body)
		}
	})
}

func TestMetadefNamespacesCoreAcceptedBodyFailuresAndMissingOwnership(t *testing.T) {
	readCause, closeCause := errors.New("read-cause"), errors.New("close-cause")
	t.Run("accepted creation read close context", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		custom := errors.New("custom-context")
		body := &namespaceCoreBody{reader: &namespaceCoreReader{data: `{"namespace":"partial"}`, err: readCause, after: func() { cancel(custom) }}, closeErr: closeCause}
		api := New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) { return namespaceCoreHTTP(req, 201, body), nil }))
		value, err := api.Create(ctx, "literal")
		if value != nil || body.closes != 1 || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, custom) {
			t.Fatalf("ownership %v closes%d", err, body.closes)
		}
		namespaceCoreProof(t, err, 201, `{"namespace":"partial"}`)
	})
	t.Run("opaque204 keeps acknowledgement on close failure", func(t *testing.T) {
		body := &namespaceCoreBody{reader: strings.NewReader("opaque\x00payload"), closeErr: closeCause}
		api := New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) { return namespaceCoreHTTP(req, 204, body), nil }))
		ack, err := api.Delete(context.Background(), "literal")
		if ack == nil || ack.StatusCode != 204 || string(ack.Body) != "opaque\x00payload" || body.closes != 1 || !errors.Is(err, closeCause) {
			t.Fatalf("ack %+v %v closes%d", ack, err, body.closes)
		}
		proof := namespaceCoreProof(t, err, 204, "opaque\x00payload")
		proof.Body[0] = 'X'
		proof.Header.Set("X-Proof", "changed")
		if string(ack.Body) != "opaque\x00payload" || ack.Header.Get("X-Proof") != "original" {
			t.Fatal("ack aliases error proof")
		}
	})
	t.Run("physical404 only clean is ignored", func(t *testing.T) {
		callbacks := 0
		client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) { return namespaceCoreJSON(req, 404, "masked"), nil })
		client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			callbacks++
			return errors.New("retry-cause")
		}
		api := New(client)
		if ack, err := api.Delete(context.Background(), "literal"); ack != nil || err != nil || callbacks != 0 {
			t.Fatalf("default404: %v %v hooks%d", ack, err, callbacks)
		}
		if ack, err := api.Delete(context.Background(), "literal", WithDeleteIgnoreMissing(false)); ack != nil || err == nil || callbacks != 1 {
			t.Fatalf("strict404: %v %v hooks%d", ack, err, callbacks)
		}
		bad := &namespaceCoreBody{reader: strings.NewReader("masked"), closeErr: closeCause}
		api = New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) { return namespaceCoreHTTP(req, 404, bad), nil }))
		if ack, err := api.Delete(context.Background(), "literal"); ack != nil || !errors.Is(err, closeCause) || bad.closes != 1 {
			t.Fatalf("dirty404: %v %v closes%d", ack, err, bad.closes)
		} else {
			namespaceCoreProof(t, err, 404, "masked")
		}
		nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404}
		api = New(namespaceCoreClient(func(*http.Request) (*http.Response, error) { return nil, nested }))
		if ack, err := api.Delete(context.Background(), "literal"); ack != nil || err == nil {
			t.Fatalf("transport404: %v %v", ack, err)
		}
	})
}

func TestMetadefNamespacesCoreCapturedSourceAndNativeRetryBoundaries(t *testing.T) {
	for _, mutate := range []func(*gophercloud.ServiceClient){
		func(c *gophercloud.ServiceClient) { c.Endpoint = "https://example.test/changed/" }, func(c *gophercloud.ServiceClient) { c.ResourceBase = "https://example.test/changed/" }, func(c *gophercloud.ServiceClient) { c.Microversion = "2.9" }, func(c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} }, func(c *gophercloud.ServiceClient) { c.Type = "compute" },
	} {
		calls := 0
		client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			return namespaceCoreJSON(req, 200, `{}`), nil
		})
		_, err := New(client).Get(context.Background(), "literal", func(*GetOpts) error { mutate(client); return nil })
		if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatalf("source mutation sent calls%d %v", calls, err)
		}
	}
	t.Run("headers captured auth live", func(t *testing.T) {
		client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("X-Source") != "before" || req.Header.Get("X-Option") != "chosen" || req.Header.Get("X-Auth-Token") != "after" {
				t.Fatalf("headers %#v", req.Header)
			}
			return namespaceCoreJSON(req, 200, `{"self":"https://foreign.test","schema":"foreign","namespace":"different"}`), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "before"}
		_, err := New(client).Get(context.Background(), "literal", func(o *GetOpts) error {
			client.MoreHeaders["X-Source"] = "after"
			client.ProviderClient.TokenID = "after"
			o.Headers["X-Option"] = "chosen"
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("retry request body cannot change", func(t *testing.T) {
		calls := 0
		client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			return namespaceCoreJSON(req, 503, "original503"), nil
		})
		client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			o.JSONBody = map[string]string{"namespace": "retarget"}
			return nil
		}
		value, err := New(client).Create(context.Background(), "literal")
		if value != nil || calls != 1 || !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) {
			t.Fatalf("body ownership calls%d %v", calls, err)
		}
	})
	t.Run("expanded accepted status is not SDK proof", func(t *testing.T) {
		calls := 0
		client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return namespaceCoreJSON(req, 503, "first"), nil
			}
			return namespaceCoreJSON(req, 202, `{"namespace":"unexpected"}`), nil
		})
		client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			o.OkCodes = []int{202}
			return nil
		}
		value, err := New(client).Create(context.Background(), "literal")
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || calls != 2 || !errors.As(err, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, []int{201}) || string(native.Body) != `{"namespace":"unexpected"}` {
			t.Fatalf("actual status: %+v %v", native, err)
		}
	})
	t.Run("source changed during accepted body retains proof", func(t *testing.T) {
		var client *gophercloud.ServiceClient
		body := &namespaceCoreBody{reader: &namespaceCoreReader{data: `{}`, err: io.EOF, after: func() { client.ResourceBase = "https://example.test/changed/" }}}
		client = namespaceCoreClient(func(req *http.Request) (*http.Response, error) { return namespaceCoreHTTP(req, 200, body), nil })
		value, err := New(client).Get(context.Background(), "literal")
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || body.closes != 1 {
			t.Fatalf("accepted source changed: %v", err)
		}
		namespaceCoreProof(t, err, 200, `{}`)
	})
}
