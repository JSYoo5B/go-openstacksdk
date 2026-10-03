package metadefnamespaces_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	namespaces "gophercloudsdk/image/v2/metadefnamespaces"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

const namespacePrefix = "/reverse/namespace/glance/v2/"
const namespaceBase = "https://glance.invalid" + namespacePrefix
const namespaceName = "OS::Compute::Libvirt"
const namespaceCollection = "metadefs/namespaces"
const namespaceObject = `{"namespace":"response::identity","display_name":"display","protected":false,"created_at":"literal-date","self":"https://passive.invalid/entity","schema":"https://passive.invalid/schema"}`
const namespaceList = `{"namespaces":[` + namespaceObject + `],"first":"https://passive.invalid/first","schema":42}`

type namespaceTransport func(*http.Request) (*http.Response, error)

func (f namespaceTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	v, e := f(r)
	if v != nil && v.Request == nil {
		v.Request = r
	}
	return v, e
}

type namespaceBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *namespaceBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type namespaceReader func([]byte) (int, error)

func (f namespaceReader) Read(p []byte) (int, error) { return f(p) }
func namespaceWire(code int, b io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"actual-namespace"}}, Body: b}
}
func namespaceClient(f namespaceTransport) *gophercloud.ServiceClient {
	p := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: f}}
	p.UseTokenLock()
	p.SetToken("initial")
	return &gophercloud.ServiceClient{ProviderClient: p, Type: "image", Endpoint: namespaceBase}
}

type namespaceOptions struct {
	create   []namespaces.CreateOption
	get      []namespaces.GetOption
	update   []namespaces.UpdateOption
	deletion []namespaces.DeleteOption
	list     []namespaces.ListOption
}
type namespaceResult struct {
	value *namespaces.Namespace
	ack   *namespaces.Acknowledgement
	rows  []*namespaces.Namespace
}

func namespaceCall(a *namespaces.API, ctx context.Context, op, name string, o namespaceOptions) (*namespaceResult, error) {
	var v *namespaces.Namespace
	var e error
	switch op {
	case "Create":
		v, e = a.Create(ctx, name, o.create...)
	case "Get":
		v, e = a.Get(ctx, name, o.get...)
	case "Update":
		v, e = a.Update(ctx, name, o.update...)
	case "Delete":
		v, e := a.Delete(ctx, name, o.deletion...)
		if v == nil {
			return nil, e
		}
		return &namespaceResult{ack: v}, e
	case "All":
		v, e := a.All(ctx, o.list...)
		if v == nil {
			return nil, e
		}
		return &namespaceResult{rows: v}, e
	case "List":
		rows := make([]*namespaces.Namespace, 0)
		for v, e := range a.List(ctx, o.list...) {
			if e != nil {
				return nil, e
			}
			rows = append(rows, v)
		}
		return &namespaceResult{rows: rows}, nil
	default:
		panic("unknown namespace operation")
	}
	if v == nil {
		return nil, e
	}
	return &namespaceResult{value: v}, e
}

var namespaceOperations = []struct {
	name, method, body, raw string
	status                  int
	named                   bool
}{
	{"Create", "POST", `{"namespace":"OS::Compute::Libvirt"}`, namespaceObject, 201, false},
	{"Get", "GET", "", namespaceObject, 200, true},
	{"Update", "PUT", `{"namespace":"OS::Compute::Libvirt"}`, namespaceObject, 200, true},
	{"Delete", "DELETE", "", "", 204, true},
	{"List", "GET", "", namespaceList, 200, false},
	{"All", "GET", "", namespaceList, 200, false},
}

func namespaceProof(t *testing.T, e error, code int, raw []byte) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(e, &proof) || proof.StatusCode != code || !bytes.Equal(proof.Body, raw) || proof.Header.Get("X-Request-Id") != "actual-namespace" {
		t.Fatalf("response proof %v %+v", e, proof)
	}
	return proof
}
func namespaceAllHeaders() namespaceOptions {
	return namespaceOptions{
		create:   []namespaces.CreateOption{namespaces.WithCreateHeaders(map[string]string{"X-Option": "owned"}), namespaces.WithCreateHeader("X-Final", "yes")},
		get:      []namespaces.GetOption{namespaces.WithGetHeaders(map[string]string{"X-Option": "owned"}), namespaces.WithGetHeader("X-Final", "yes")},
		update:   []namespaces.UpdateOption{namespaces.WithUpdateHeaders(map[string]string{"X-Option": "owned"}), namespaces.WithUpdateHeader("X-Final", "yes")},
		deletion: []namespaces.DeleteOption{namespaces.WithDeleteHeaders(map[string]string{"X-Option": "owned"}), namespaces.WithDeleteHeader("X-Final", "yes")},
		list:     []namespaces.ListOption{namespaces.WithListHeaders(map[string]string{"X-Option": "owned"}), namespaces.WithListHeader("X-Final", "yes")},
	}
}

func TestMetadefNamespacesFixedRoutesAndPayloads(t *testing.T) {
	for _, op := range namespaceOperations {
		t.Run(op.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", "/catalog/unused/")
			client.ResourceBase = cloud.Server.URL + namespacePrefix
			client.Microversion = "2.2"
			client.MoreHeaders = map[string]string{"X-Source": "before"}
			cloud.Provider.SetToken("live")
			var calls atomic.Int32
			cloud.Mux.HandleFunc(namespacePrefix, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				path := namespacePrefix + namespaceCollection
				if op.named {
					path += "/" + namespaceName
				}
				if r.Method != op.method || r.URL.Path != path || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "live" || r.Header.Get("X-Source") != "before" || r.Header.Get("X-Option") != "owned" || r.Header.Get("X-Final") != "yes" || r.Header.Get("OpenStack-API-Version") != "image 2.2" {
					t.Error(r.Method, r.URL, r.Header)
				}
				raw, e := io.ReadAll(r.Body)
				if e != nil || string(raw) != op.body {
					t.Error(string(raw), e)
				}
				w.Header().Set("X-Request-Id", "actual-namespace")
				w.Header().Set("Location", "https://passive.invalid/created")
				testcloud.JSON(w, op.status, op.raw)
			})
			v, e := namespaceCall(namespaces.New(client), context.Background(), op.name, namespaceName, namespaceAllHeaders())
			if e != nil || v == nil || calls.Load() != 1 {
				t.Fatal(v, e, calls.Load())
			}
			if v.value != nil && (*v.value.Namespace != "response::identity" || v.value.StatusCode != op.status || v.value.Header.Get("Location") != "https://passive.invalid/created") {
				t.Fatal(v.value)
			}
			if v.ack != nil && (v.ack.Namespace != namespaceName || v.ack.StatusCode != 204) {
				t.Fatal(v.ack)
			}
			if v.rows != nil && len(v.rows) != 1 {
				t.Fatal(v.rows)
			}
		})
	}
	for _, name := range []string{"OS::Vendor Name::한글", strings.Repeat("界", 80)} {
		for _, op := range namespaceOperations {
			if !op.named && op.name != "Create" {
				continue
			}
			t.Run(op.name+" identity "+name, func(t *testing.T) {
				var calls atomic.Int32
				client := namespaceClient(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					path := namespacePrefix + namespaceCollection
					if op.named {
						path += "/" + url.PathEscape(name)
					}
					if r.URL.EscapedPath() != path || r.URL.RawQuery != "" {
						t.Error(r.URL, r.URL.EscapedPath(), path)
					}
					if op.name == "Create" || op.name == "Update" {
						raw, _ := io.ReadAll(r.Body)
						want, _ := json.Marshal(map[string]string{"namespace": name})
						if !bytes.Equal(raw, want) {
							t.Error(string(raw), string(want))
						}
					}
					return namespaceWire(op.status, io.NopCloser(strings.NewReader(op.raw))), nil
				})
				v, e := namespaceCall(namespaces.New(client), context.Background(), op.name, name, namespaceOptions{})
				if e != nil || v == nil || calls.Load() != 1 || v.ack != nil && v.ack.Namespace != name {
					t.Fatal(v, e, calls.Load())
				}
			})
		}
	}
	t.Run("explicit scalar create and replacement rename", func(t *testing.T) {
		var calls atomic.Int32
		client := namespaceClient(func(r *http.Request) (*http.Response, error) {
			n := calls.Add(1)
			raw, _ := io.ReadAll(r.Body)
			var body map[string]json.RawMessage
			if json.Unmarshal(raw, &body) != nil {
				t.Error(string(raw))
			}
			want := map[string]string{"namespace": namespaceName, "display_name": "", "description": "line1\nline2", "visibility": "public", "owner": ""}
			if n == 2 {
				want["namespace"] = "Renamed::Namespace"
				want["visibility"] = "private"
			}
			if len(body) != 6 || string(body["protected"]) != "false" {
				t.Error(string(raw))
			}
			for k, v := range want {
				var actual string
				if json.Unmarshal(body[k], &actual) != nil || actual != v {
					t.Error(k, string(body[k]), v)
				}
			}
			if n == 1 && (r.Method != "POST" || r.URL.String() != namespaceBase+namespaceCollection) || n == 2 && (r.Method != "PUT" || r.URL.String() != namespaceBase+namespaceCollection+"/"+url.PathEscape(namespaceName)) {
				t.Error(r.Method, r.URL)
			}
			code := 201
			if n == 2 {
				code = 200
			}
			return namespaceWire(code, io.NopCloser(strings.NewReader(namespaceObject))), nil
		})
		a := namespaces.New(client)
		v, e := a.Create(context.Background(), namespaceName, namespaces.WithCreateDisplayName(""), namespaces.WithCreateDescription("line1\nline2"), namespaces.WithCreateVisibility("public"), namespaces.WithCreateOwner(""), namespaces.WithCreateProtected(false))
		if e != nil || v == nil {
			t.Fatal(v, e)
		}
		v, e = a.Update(context.Background(), namespaceName, namespaces.WithUpdateNamespace("Renamed::Namespace"), namespaces.WithUpdateDisplayName(""), namespaces.WithUpdateDescription("line1\nline2"), namespaces.WithUpdateVisibility("private"), namespaces.WithUpdateOwner(""), namespaces.WithUpdateProtected(false))
		if e != nil || v == nil || calls.Load() != 2 {
			t.Fatal(v, e, calls.Load())
		}
	})
	for _, resourceType := range []string{"", "OS::Glance::Image, OS::Nova::Flavor"} {
		t.Run("resource_type literal "+resourceType, func(t *testing.T) {
			var calls atomic.Int32
			client := namespaceClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				q := r.URL.Query()
				if !q.Has("resource_type") || q.Get("resource_type") != resourceType || len(q) != 1 || r.Body != nil {
					t.Error(r.URL, r.Body)
				}
				return namespaceWire(200, io.NopCloser(strings.NewReader(`{"properties":{"hw:cpu":{"title":"CPU","type":"integer"}},"namespace":"foreign"}`))), nil
			})
			v, e := namespaces.New(client).Get(context.Background(), namespaceName, namespaces.WithGetResourceType(resourceType))
			if e != nil || v == nil || calls.Load() != 1 || !bytes.Contains(v.Body["properties"], []byte("hw:cpu")) {
				t.Fatal(v, e, calls.Load())
			}
		})
	}
}

func TestMetadefNamespacesCanonicalModelsAndRawOwnership(t *testing.T) {
	for _, raw := range []string{`{}`, `{"namespace":null,"display_name":null,"description":null,"owner":null,"visibility":null,"self":null,"schema":null,"protected":null,"created_at":null,"updated_at":null}`, `{"namespace":"","display_name":"","description":"","owner":"","visibility":"","self":"","schema":"","protected":false,"created_at":"","updated_at":""}`, `{"namespace":"other::namespace","display_name":"label","description":"line\ntext","visibility":"future","owner":"foreign","self":"https://passive.invalid","schema":"https://schema.invalid","created_at":"unparsed","updated_at":"not-a-date","protected":true,"properties":42,"objects":false,"tags":null,"resource_type_associations":"opaque","links":false,"Namespace":42,"extension":9007199254740993}`} {
		t.Run("nullable passive "+raw, func(t *testing.T) {
			b := &namespaceBody{Reader: strings.NewReader(raw)}
			wire := namespaceWire(200, b)
			client := namespaceClient(func(*http.Request) (*http.Response, error) { return wire, nil })
			v, e := namespaces.New(client).Get(context.Background(), namespaceName)
			if e != nil || v == nil || v.Links != nil || v.StatusCode != 200 || b.closes.Load() != 1 {
				t.Fatal(v, e, b.closes.Load())
			}
			for _, f := range []struct {
				k string
				p *string
			}{{"namespace", v.Namespace}, {"display_name", v.DisplayName}, {"description", v.Description}, {"visibility", v.Visibility}, {"owner", v.Owner}, {"self", v.Self}, {"schema", v.Schema}, {"created_at", v.CreatedAt}, {"updated_at", v.UpdatedAt}} {
				token, exists := v.Body[f.k]
				if !exists || string(token) == "null" {
					if f.p != nil {
						t.Fatal(f.k, f.p)
					}
				} else {
					var want string
					if json.Unmarshal(token, &want) != nil || f.p == nil || *f.p != want {
						t.Fatal(f.k, f.p, string(token))
					}
				}
			}
			token, exists := v.Body["protected"]
			if !exists || string(token) == "null" {
				if v.IsProtected != nil {
					t.Fatal(v.IsProtected)
				}
			} else {
				var flag bool
				if json.Unmarshal(token, &flag) != nil || v.IsProtected == nil || *v.IsProtected != flag {
					t.Fatal(v.IsProtected, string(token))
				}
			}
			if strings.Contains(raw, "extension") && string(v.Body["extension"]) != "9007199254740993" {
				t.Fatal(v.Body)
			}
			wire.Header.Set("X-Request-Id", "changed")
			if v.Header.Get("X-Request-Id") != "actual-namespace" {
				t.Fatal("header alias")
			}
			if v.Namespace != nil {
				before := *v.Namespace
				v.Body["namespace"][1] = '!'
				if *v.Namespace != before {
					t.Fatal("typed raw alias")
				}
			}
		})
	}
	for _, field := range []string{"namespace", "display_name", "description", "owner", "visibility", "self", "schema", "created_at", "updated_at", "protected"} {
		tokens := []string{"1", "[]", "{}"}
		if field == "protected" {
			tokens = append(tokens, `"false"`)
		} else {
			tokens = append(tokens, "false")
		}
		for _, token := range tokens {
			t.Run(field+" rejects "+token, func(t *testing.T) {
				raw := []byte(fmt.Sprintf(`{%q:%s}`, field, token))
				b := &namespaceBody{Reader: bytes.NewReader(raw)}
				client := namespaceClient(func(*http.Request) (*http.Response, error) { return namespaceWire(200, b), nil })
				v, e := namespaces.New(client).Get(context.Background(), namespaceName)
				namespaceProof(t, e, 200, raw)
				if v != nil || b.closes.Load() != 1 {
					t.Fatal(v, e, b.closes.Load())
				}
			})
		}
	}
	for _, raw := range [][]byte{[]byte(`null`), []byte(`[]`), []byte(`1`), []byte(`{"namespace":`), []byte("{\"extension\":\"\xff\"}")} {
		t.Run("strict root "+string(raw), func(t *testing.T) {
			b := &namespaceBody{Reader: bytes.NewReader(raw)}
			client := namespaceClient(func(*http.Request) (*http.Response, error) { return namespaceWire(201, b), nil })
			v, e := namespaces.New(client).Create(context.Background(), namespaceName)
			namespaceProof(t, e, 201, raw)
			if v != nil || b.closes.Load() != 1 {
				t.Fatal(v, e, b.closes.Load())
			}
		})
	}
	t.Run("rows independently owned and full page retained", func(t *testing.T) {
		raw := `{"namespaces":[{"namespace":"one","extension":9007199254740993},{"namespace":"two"},null],"next":"https://passive.invalid"}`
		var calls atomic.Int32
		client := namespaceClient(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return namespaceWire(200, io.NopCloser(strings.NewReader(raw))), nil
		})
		a := namespaces.New(client)
		rows, e := a.All(context.Background(), namespaces.WithListMaxItems(2))
		if e != nil || len(rows) != 2 {
			t.Fatal(rows, e)
		}
		rows[0].Header.Set("X-Request-Id", "caller")
		rows[0].Body["namespace"][1] = '!'
		if rows[1].Header.Get("X-Request-Id") != "actual-namespace" || *rows[0].Namespace != "one" || string(rows[0].Body["extension"]) != "9007199254740993" {
			t.Fatal(rows)
		}
		rows, e = a.All(context.Background())
		namespaceProof(t, e, 200, []byte(raw))
		if rows != nil || calls.Load() != 2 {
			t.Fatal(rows, e, calls.Load())
		}
	})
}

func TestMetadefNamespacesAdvertisedPagingAndControls(t *testing.T) {
	for _, mode := range []string{"exact escaped prefix", "other escaped alias"} {
		t.Run(mode, func(t *testing.T) {
			const base = "https://glance.invalid/reverse/tenant%2Fsegment/glance/v2/"
			const path = "/reverse/tenant%2Fsegment/glance/v2/metadefs/namespaces"
			var calls atomic.Int32
			var bodies []*namespaceBody
			var firstRaw string
			client := namespaceClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if r.URL.EscapedPath() != path || n == 1 && r.URL.RawQuery != "" || n == 2 && r.URL.Query().Get("marker") != "Next::Namespace" {
					t.Error(r.URL, r.URL.EscapedPath())
				}
				raw := `{"namespaces":[{}]}`
				if n == 1 {
					next := base + namespaceCollection + "?marker=Next%3A%3ANamespace"
					if mode == "other escaped alias" {
						next = strings.Replace(next, "/namespaces?", "/%6eamespaces?", 1)
					}
					raw = fmt.Sprintf(`{"namespaces":[{}],"next":%q}`, next)
					firstRaw = raw
				}
				b := &namespaceBody{Reader: strings.NewReader(raw)}
				bodies = append(bodies, b)
				return namespaceWire(200, b), nil
			})
			client.Endpoint = base
			rows, e := namespaces.New(client).All(context.Background())
			if mode == "exact escaped prefix" {
				if e != nil || len(rows) != 2 || calls.Load() != 2 {
					t.Fatal(rows, e, calls.Load())
				}
			} else {
				namespaceProof(t, e, 200, []byte(firstRaw))
				if rows != nil || calls.Load() != 1 {
					t.Fatal(rows, e, calls.Load())
				}
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
	for _, form := range []string{"version root", "captured absolute", "captured relative", "query relative"} {
		t.Run(form+" next preserves literal filters", func(t *testing.T) {
			initial := url.Values{"limit": {"2"}, "marker": {"Start::Namespace"}, "visibility": {"public"}, "sort_key": {"future_sort"}, "sort_dir": {"asc"}, "resource_types": {"OS::Glance::Image, OS::Nova::Flavor"}}
			nextQuery := url.Values{}
			for k, v := range initial {
				nextQuery[k] = append([]string(nil), v...)
			}
			nextQuery.Set("marker", "Next::Namespace")
			var calls atomic.Int32
			var bodies []*namespaceBody
			client := namespaceClient(nil)
			client.HTTPClient.Transport = namespaceTransport(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				want := initial
				if n == 2 {
					want = nextQuery
				}
				if n > 2 || r.URL.EscapedPath() != namespacePrefix+namespaceCollection || !reflect.DeepEqual(r.URL.Query(), want) || r.Body != nil {
					t.Error(r.URL, want, r.Body)
				}
				raw := `{"namespaces":[{"namespace":"response::one"}]}`
				if n == 1 {
					next := "/v2/" + namespaceCollection + "?" + nextQuery.Encode()
					switch form {
					case "captured absolute":
						next = namespaceBase + namespaceCollection + "?" + nextQuery.Encode()
					case "captured relative":
						next = "namespaces?" + nextQuery.Encode()
					case "query relative":
						next = "?" + nextQuery.Encode()
					}
					raw = fmt.Sprintf(`{"namespaces":[{"namespace":"passive::wrong-marker"}],"next":%q}`, next)
				}
				b := &namespaceBody{Reader: strings.NewReader(raw)}
				if n == 1 {
					b.onClose = func() { client.SetToken("next-token") }
				} else if r.Header.Get("X-Auth-Token") != "next-token" {
					t.Error(r.Header)
				}
				bodies = append(bodies, b)
				return namespaceWire(200, b), nil
			})
			rows, e := namespaces.New(client).All(context.Background(), namespaces.WithListLimit(2), namespaces.WithListMarker("Start::Namespace"), namespaces.WithListVisibility("public"), namespaces.WithListSortKey("future_sort"), namespaces.WithListSortDir("asc"), namespaces.WithListResourceTypes("OS::Glance::Image, OS::Nova::Flavor"))
			if e != nil || len(rows) != 2 || calls.Load() != 2 {
				t.Fatal(rows, e, calls.Load())
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
	for _, mode := range []string{"no hint cap20", "explicit limit no next", "explicit zero", "empty marker omitted", "empty page malformed next", "header and noncanonical links ignored"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			client := namespaceClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				q := r.URL.Query()
				if mode == "explicit limit no next" || mode == "explicit zero" {
					want := "2"
					if mode == "explicit zero" {
						want = "0"
					}
					if q.Get("limit") != want || len(q) != 1 {
						t.Error(r.URL)
					}
				} else if len(q) != 0 {
					t.Error("invented query", r.URL)
				}
				raw := `{"namespaces":[{},{}]}`
				if mode == "explicit zero" {
					raw = `{"namespaces":[]}`
				}
				if mode == "empty page malformed next" {
					raw = `{"namespaces":[],"next":42}`
				}
				if mode == "header and noncanonical links ignored" {
					raw = `{"namespaces":[{}],"links":[{"rel":"next","href":"https://foreign.invalid"}],"Next":"https://foreign.invalid"}`
				}
				wire := namespaceWire(200, io.NopCloser(strings.NewReader(raw)))
				wire.Header.Set("Link", `<https://foreign.invalid/next>; rel="next"`)
				return wire, nil
			})
			opts := []namespaces.ListOption{namespaces.WithListMaxItems(20)}
			switch mode {
			case "explicit limit no next":
				opts = append(opts, namespaces.WithListLimit(2))
			case "explicit zero":
				opts = append(opts, namespaces.WithListLimit(0))
			case "empty marker omitted":
				opts = append(opts, namespaces.WithListMarker(""))
			}
			rows, e := namespaces.New(client).All(context.Background(), opts...)
			if e != nil || rows == nil || calls.Load() != 1 {
				t.Fatal(rows, e, calls.Load())
			}
			if (mode == "explicit zero" || mode == "empty page malformed next") && len(rows) != 0 {
				t.Fatal(rows)
			}
		})
	}
	for _, mode := range []string{"cap skips bad row and next", "break skips bad row and next", "SinglePage skips next", "SinglePage validates bad row"} {
		t.Run(mode, func(t *testing.T) {
			raw := `{"namespaces":[{},null],"next":42}`
			if mode == "SinglePage skips next" {
				raw = `{"namespaces":[{},{}],"next":42}`
			}
			var calls atomic.Int32
			client := namespaceClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return namespaceWire(200, io.NopCloser(strings.NewReader(raw))), nil
			})
			a := namespaces.New(client)
			if mode == "break skips bad row and next" {
				n := 0
				for v, e := range a.List(context.Background()) {
					if v == nil || e != nil {
						t.Fatal(v, e)
					}
					n++
					break
				}
				if n != 1 || calls.Load() != 1 {
					t.Fatal(n, calls.Load())
				}
				return
			}
			opts := []namespaces.ListOption{namespaces.WithListSinglePage(true)}
			if mode == "cap skips bad row and next" {
				opts = []namespaces.ListOption{namespaces.WithListMaxItems(1)}
			}
			rows, e := a.All(context.Background(), opts...)
			if mode == "SinglePage validates bad row" {
				namespaceProof(t, e, 200, []byte(raw))
				if rows != nil {
					t.Fatal(rows)
				}
			} else if e != nil || len(rows) == 0 {
				t.Fatal(rows, e)
			}
			if calls.Load() != 1 {
				t.Fatal(calls.Load())
			}
		})
	}
	for _, next := range []string{
		`https://foreign.invalid/v2/metadefs/namespaces?limit=2&visibility=public&marker=Next`,
		`//glance.invalid/reverse/namespace/glance/v2/metadefs/namespaces?limit=2&visibility=public&marker=Next`,
		`https://user@glance.invalid/reverse/namespace/glance/v2/metadefs/namespaces?limit=2&visibility=public&marker=Next`,
		`/v2/metadefs/namespaces?limit=2&visibility=public&marker=Next#fragment`,
		`/v2/metadefs/other?limit=2&visibility=public&marker=Next`,
		`/v2/metadefs/../metadefs/namespaces?limit=2&visibility=public&marker=Next`,
		`/v2/metadefs/%6eamespaces?limit=2&visibility=public&marker=Next`,
		`/v2/metadefs/namespaces?limit=2&visibility=private&marker=Next`,
		`/v2/metadefs/namespaces?limit=2&marker=Next`,
		`/v2/metadefs/namespaces?limit=1&visibility=public&marker=Next`,
		`/v2/metadefs/namespaces?limit=2&visibility=public&marker=Next&extra=value`,
		`/v2/metadefs/namespaces?limit=2&visibility=public&marker=Next&marker=Other`,
		`/v2/metadefs/namespaces?limit=2&visibility=public&marker=`,
		`/v2/metadefs/namespaces?limit=2&visibility=public`,
		`/v2/metadefs/namespaces?limit=2&visibility=public&marker=bad%2Fsegment`,
		`/v2/metadefs/namespaces?limit=2&visibility=public&marker=%zz`,
	} {
		t.Run("guard "+next, func(t *testing.T) {
			raw := fmt.Sprintf(`{"namespaces":[{}],"next":%q}`, next)
			var calls atomic.Int32
			b := &namespaceBody{Reader: strings.NewReader(raw)}
			client := namespaceClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return namespaceWire(200, b), nil })
			rows, e := namespaces.New(client).All(context.Background(), namespaces.WithListLimit(2), namespaces.WithListVisibility("public"))
			namespaceProof(t, e, 200, []byte(raw))
			if rows != nil || calls.Load() != 1 || b.closes.Load() != 1 {
				t.Fatal(rows, e, calls.Load(), b.closes.Load())
			}
		})
	}
	for _, mode := range []string{"initial marker repeated", "later marker repeated", "later HTTP404", "later read", "later bad row"} {
		t.Run(mode, func(t *testing.T) {
			readCause := errors.New("later page Read")
			var calls atomic.Int32
			var bodies []*namespaceBody
			var lastRaw string
			client := namespaceClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				code := 200
				raw := `{"namespaces":[{}],"next":"/v2/metadefs/namespaces?marker=Next"}`
				if mode == "initial marker repeated" {
					raw = `{"namespaces":[{}],"next":"/v2/metadefs/namespaces?marker=Start"}`
				}
				if n == 2 {
					switch mode {
					case "later marker repeated":
						raw = `{"namespaces":[{}],"next":"/v2/metadefs/namespaces?marker=Next"}`
					case "later HTTP404":
						code = 404
						raw = "actual later404"
					case "later read":
						raw = `{"namespaces":`
					case "later bad row":
						raw = `{"namespaces":[{"protected":"false"}]}`
					}
				}
				lastRaw = raw
				b := &namespaceBody{Reader: strings.NewReader(raw)}
				if mode == "later read" && n == 2 {
					b.Reader = namespaceReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
				}
				bodies = append(bodies, b)
				return namespaceWire(code, b), nil
			})
			opts := []namespaces.ListOption{}
			if mode == "initial marker repeated" {
				opts = append(opts, namespaces.WithListMarker("Start"))
			}
			rows, e := namespaces.New(client).All(context.Background(), opts...)
			wantCalls := 2
			if mode == "initial marker repeated" {
				wantCalls = 1
			}
			if rows != nil || e == nil || calls.Load() != int32(wantCalls) {
				t.Fatal(rows, e, calls.Load())
			}
			if strings.Contains(mode, "marker repeated") {
				namespaceProof(t, e, 200, []byte(lastRaw))
				if !errors.Is(e, resource.ErrPaginationCycle) {
					t.Fatal(e)
				}
			} else if mode == "later HTTP404" {
				if !gophercloud.ResponseCodeIs(e, 404) {
					t.Fatal(e)
				}
			} else {
				namespaceProof(t, e, 200, []byte(lastRaw))
				if mode == "later read" && !errors.Is(e, readCause) {
					t.Fatal(e)
				}
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
	for _, raw := range []string{`{}`, `null`, `[]`, `{"namespaces":null}`, `{"namespaces":{}}`, `{"Namespaces":[]}`, `{"namespaces":[null]}`, `{"namespaces":[[]]}`, `{"namespaces":[{"protected":1}]}`, "{\"namespaces\":[{}],\"unknown\":\"\xff\"}"} {
		t.Run("strict envelope "+raw, func(t *testing.T) {
			client := namespaceClient(func(*http.Request) (*http.Response, error) {
				return namespaceWire(200, io.NopCloser(strings.NewReader(raw))), nil
			})
			rows, e := namespaces.New(client).All(context.Background())
			namespaceProof(t, e, 200, []byte(raw))
			if rows != nil {
				t.Fatal(rows, e)
			}
		})
	}
	t.Run("parallel reusable lazy option slice", func(t *testing.T) {
		var calls, callbacks atomic.Int32
		headers := map[string]string{"X-Option": "snapshot"}
		opts := []namespaces.ListOption{namespaces.WithListOpts(namespaces.ListOpts{Headers: headers, MaxItems: 1}), func(*namespaces.ListOpts) error { callbacks.Add(1); return nil }}
		client := namespaceClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Option") != "snapshot" || r.URL.RawQuery != "" {
				t.Error(r.Header, r.URL)
			}
			return namespaceWire(200, io.NopCloser(strings.NewReader(`{"namespaces":[{},null],"next":42}`))), nil
		})
		seq := namespaces.New(client).List(context.Background(), opts...)
		opts[0] = nil
		headers["X-Option"] = "later"
		if calls.Load() != 0 || callbacks.Load() != 0 {
			t.Fatal("eager work")
		}
		var wg sync.WaitGroup
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				n := 0
				for v, e := range seq {
					if v == nil || e != nil {
						t.Error(v, e)
					}
					n++
				}
				if n != 1 {
					t.Error(n)
				}
			}()
		}
		wg.Wait()
		if calls.Load() != 4 || callbacks.Load() != 4 {
			t.Fatal(calls.Load(), callbacks.Load())
		}
	})
}

func TestMetadefNamespacesOwnedPreparationAndSource(t *testing.T) {
	for _, op := range namespaceOperations {
		for _, mode := range []string{"nil context", "cancelled context", "nil API", "nil provider", "bad source", "wrong type", "source protected header"} {
			t.Run(op.name+" "+mode, func(t *testing.T) {
				var calls, callbacks atomic.Int32
				client := namespaceClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				a := namespaces.New(client)
				ctx := context.Background()
				cause := errors.New("preflight cancel")
				switch mode {
				case "nil context":
					ctx = nil
				case "cancelled context":
					c, cancel := context.WithCancelCause(ctx)
					cancel(cause)
					ctx = c
				case "nil API":
					a = nil
				case "nil provider":
					client.ProviderClient = nil
				case "bad source":
					client.Endpoint = "https://user:secret@glance.invalid/v2/"
				case "wrong type":
					client.Type = "compute"
				case "source protected header":
					client.MoreHeaders = map[string]string{"x-auth-token": "forged"}
				}
				o := namespaceOptions{create: []namespaces.CreateOption{func(*namespaces.CreateOpts) error { callbacks.Add(1); return nil }}, get: []namespaces.GetOption{func(*namespaces.GetOpts) error { callbacks.Add(1); return nil }}, update: []namespaces.UpdateOption{func(*namespaces.UpdateOpts) error { callbacks.Add(1); return nil }}, deletion: []namespaces.DeleteOption{func(*namespaces.DeleteOpts) error { callbacks.Add(1); return nil }}, list: []namespaces.ListOption{func(*namespaces.ListOpts) error { callbacks.Add(1); return nil }}}
				v, e := namespaceCall(a, ctx, op.name, namespaceName, o)
				if v != nil || e == nil || calls.Load() != 0 || callbacks.Load() != 0 {
					t.Fatal(v, e, calls.Load(), callbacks.Load())
				}
				if mode == "cancelled context" {
					if !errors.Is(e, cause) || !errors.Is(e, context.Canceled) {
						t.Fatal(e)
					}
				} else if mode == "wrong type" {
					if !errors.Is(e, resource.ErrUnsupported) {
						t.Fatal(e)
					}
				} else if !errors.Is(e, resource.ErrInvalidOption) {
					t.Fatal(e)
				}
			})
		}
	}
	for _, op := range namespaceOperations {
		if !op.named && op.name != "Create" {
			continue
		}
		for _, name := range []string{"", ".", "..", "bad/name", "back\\name", "literal%2F", "query?name", "fragment#name", "line\nname", "delete\x7fname", strings.Repeat("界", 81), string([]byte{255})} {
			t.Run(op.name+" invalid identity "+name, func(t *testing.T) {
				var calls, callbacks atomic.Int32
				client := namespaceClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				o := namespaceOptions{create: []namespaces.CreateOption{func(*namespaces.CreateOpts) error { callbacks.Add(1); return nil }}, get: []namespaces.GetOption{func(*namespaces.GetOpts) error { callbacks.Add(1); return nil }}, update: []namespaces.UpdateOption{func(*namespaces.UpdateOpts) error { callbacks.Add(1); return nil }}, deletion: []namespaces.DeleteOption{func(*namespaces.DeleteOpts) error { callbacks.Add(1); return nil }}}
				v, e := namespaceCall(namespaces.New(client), context.Background(), op.name, name, o)
				if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 || callbacks.Load() != 0 {
					t.Fatal(v, e, calls.Load(), callbacks.Load())
				}
			})
		}
	}
	for _, op := range namespaceOperations {
		for _, header := range []string{"X-Auth-Token", "Content-Type", "Accept", "Content-Length", "Transfer-Encoding", "Host", "OpenStack-API-Version", "Bad Header"} {
			t.Run(op.name+" protected option "+header, func(t *testing.T) {
				var calls atomic.Int32
				client := namespaceClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				o := namespaceOptions{create: []namespaces.CreateOption{namespaces.WithCreateHeader(header, "value")}, get: []namespaces.GetOption{namespaces.WithGetHeader(header, "value")}, update: []namespaces.UpdateOption{namespaces.WithUpdateHeader(header, "value")}, deletion: []namespaces.DeleteOption{namespaces.WithDeleteHeader(header, "value")}, list: []namespaces.ListOption{namespaces.WithListHeader(header, "value")}}
				v, e := namespaceCall(namespaces.New(client), context.Background(), op.name, namespaceName, o)
				if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
					t.Fatal(v, e, calls.Load())
				}
			})
		}
	}
	for _, op := range namespaceOperations {
		for _, mode := range []string{"nil callback", "callback cause", "provider", "Endpoint", "base", "type", "Microversion", "callback cancel"} {
			t.Run(op.name+" callback "+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("callback cause")
				var calls atomic.Int32
				client := namespaceClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				mutate := func() error {
					switch mode {
					case "callback cause":
						return cause
					case "provider":
						client.ProviderClient = &gophercloud.ProviderClient{}
					case "Endpoint":
						client.Endpoint = "https://glance.invalid/replacement/v2/"
					case "base":
						client.ResourceBase = "https://glance.invalid/replacement/v2/"
					case "type":
						client.Type = "compute"
					case "Microversion":
						client.Microversion = "2.3"
					case "callback cancel":
						cancel(cause)
					}
					return nil
				}
				o := namespaceOptions{create: []namespaces.CreateOption{func(*namespaces.CreateOpts) error { return mutate() }}, get: []namespaces.GetOption{func(*namespaces.GetOpts) error { return mutate() }}, update: []namespaces.UpdateOption{func(*namespaces.UpdateOpts) error { return mutate() }}, deletion: []namespaces.DeleteOption{func(*namespaces.DeleteOpts) error { return mutate() }}, list: []namespaces.ListOption{func(*namespaces.ListOpts) error { return mutate() }}}
				if mode == "nil callback" {
					o = namespaceOptions{create: []namespaces.CreateOption{nil}, get: []namespaces.GetOption{nil}, update: []namespaces.UpdateOption{nil}, deletion: []namespaces.DeleteOption{nil}, list: []namespaces.ListOption{nil}}
				}
				v, e := namespaceCall(namespaces.New(client), ctx, op.name, namespaceName, o)
				if v != nil || e == nil || calls.Load() != 0 {
					t.Fatal(v, e, calls.Load())
				}
				if mode == "callback cause" || mode == "callback cancel" {
					if !errors.Is(e, cause) {
						t.Fatal(e)
					}
				} else if !errors.Is(e, resource.ErrInvalidOption) {
					t.Fatal(e)
				}
			})
		}
	}
	invalidOptions := []struct {
		name string
		o    namespaceOptions
	}{
		{"create display length", namespaceOptions{create: []namespaces.CreateOption{namespaces.WithCreateDisplayName(strings.Repeat("界", 81))}}},
		{"create description length", namespaceOptions{create: []namespaces.CreateOption{namespaces.WithCreateDescription(strings.Repeat("界", 501))}}},
		{"create owner length", namespaceOptions{create: []namespaces.CreateOption{namespaces.WithCreateOwner(strings.Repeat("界", 256))}}},
		{"create visibility", namespaceOptions{create: []namespaces.CreateOption{namespaces.WithCreateVisibility("")}}},
		{"create UTF8", namespaceOptions{create: []namespaces.CreateOption{namespaces.WithCreateDescription(string([]byte{255}))}}},
		{"update rename", namespaceOptions{update: []namespaces.UpdateOption{namespaces.WithUpdateNamespace("bad/name")}}},
		{"update visibility", namespaceOptions{update: []namespaces.UpdateOption{namespaces.WithUpdateVisibility("Public")}}},
		{"get ResourceType control", namespaceOptions{get: []namespaces.GetOption{namespaces.WithGetResourceType("bad\nvalue")}}},
		{"negative limit", namespaceOptions{list: []namespaces.ListOption{namespaces.WithListLimit(-1)}}},
		{"negative cap", namespaceOptions{list: []namespaces.ListOption{namespaces.WithListMaxItems(-1)}}},
		{"bad marker", namespaceOptions{list: []namespaces.ListOption{namespaces.WithListMarker("bad/name")}}},
		{"bad visibility", namespaceOptions{list: []namespaces.ListOption{namespaces.WithListVisibility("Public")}}},
		{"bad sortdir", namespaceOptions{list: []namespaces.ListOption{namespaces.WithListSortDir("ASC")}}},
		{"sort control", namespaceOptions{list: []namespaces.ListOption{namespaces.WithListSortKey("sort\nkey")}}},
		{"ResourceTypes UTF8", namespaceOptions{list: []namespaces.ListOption{namespaces.WithListResourceTypes(string([]byte{255}))}}},
	}
	for _, test := range invalidOptions {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			client := namespaceClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
			op := "All"
			if test.o.create != nil {
				op = "Create"
			}
			if test.o.update != nil {
				op = "Update"
			}
			if test.o.get != nil {
				op = "Get"
			}
			v, e := namespaceCall(namespaces.New(client), context.Background(), op, namespaceName, test.o)
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(v, e, calls.Load())
			}
		})
	}
	for _, op := range namespaceOperations {
		t.Run(op.name+" helper inputs and callback configs copied", func(t *testing.T) {
			input := map[string]string{"X-Owned": "before"}
			text := "before"
			flag := true
			limit := 1
			var escaped map[string]string
			var escapedText *string
			var escapedFlag *bool
			var escapedLimit *int
			var callbacks atomic.Int32
			mutate := func() {
				callbacks.Add(1)
				escaped["X-Owned"] = "escaped"
				if escapedText != nil {
					*escapedText = "escaped"
				}
				if escapedFlag != nil {
					*escapedFlag = false
				}
				if escapedLimit != nil {
					*escapedLimit = 99
				}
			}
			o := namespaceOptions{
				create: []namespaces.CreateOption{namespaces.WithCreateHeader("X-Dropped", "gone"), namespaces.WithCreateOpts(namespaces.CreateOpts{Headers: input, DisplayName: &text, Protected: &flag}), func(c *namespaces.CreateOpts) error {
					escaped = c.Headers
					escapedText = c.DisplayName
					escapedFlag = c.Protected
					return nil
				}, func(*namespaces.CreateOpts) error { mutate(); return nil }},
				get: []namespaces.GetOption{namespaces.WithGetHeader("X-Dropped", "gone"), namespaces.WithGetOpts(namespaces.GetOpts{Headers: input, ResourceType: &text}), func(c *namespaces.GetOpts) error { escaped = c.Headers; escapedText = c.ResourceType; return nil }, func(*namespaces.GetOpts) error { mutate(); return nil }},
				update: []namespaces.UpdateOption{namespaces.WithUpdateHeader("X-Dropped", "gone"), namespaces.WithUpdateOpts(namespaces.UpdateOpts{Headers: input, DisplayName: &text, Protected: &flag}), func(c *namespaces.UpdateOpts) error {
					escaped = c.Headers
					escapedText = c.DisplayName
					escapedFlag = c.Protected
					return nil
				}, func(*namespaces.UpdateOpts) error { mutate(); return nil }},
				deletion: []namespaces.DeleteOption{namespaces.WithDeleteHeader("X-Dropped", "gone"), namespaces.WithDeleteOpts(namespaces.DeleteOpts{Headers: input, IgnoreMissing: &flag}), func(c *namespaces.DeleteOpts) error { escaped = c.Headers; escapedFlag = c.IgnoreMissing; return nil }, func(*namespaces.DeleteOpts) error { mutate(); return nil }},
				list:     []namespaces.ListOption{namespaces.WithListHeader("X-Dropped", "gone"), namespaces.WithListOpts(namespaces.ListOpts{Headers: input, Limit: &limit, MaxItems: 1}), func(c *namespaces.ListOpts) error { escaped = c.Headers; escapedLimit = c.Limit; return nil }, func(*namespaces.ListOpts) error { mutate(); return nil }},
			}
			input["X-Owned"] = "caller"
			text = "caller"
			flag = false
			limit = 99
			client := namespaceClient(func(r *http.Request) (*http.Response, error) {
				if r.Header.Get("X-Owned") != "before" || r.Header.Get("X-Dropped") != "" {
					t.Error(r.Header)
				}
				if op.name == "Create" || op.name == "Update" {
					raw, _ := io.ReadAll(r.Body)
					var body map[string]json.RawMessage
					_ = json.Unmarshal(raw, &body)
					if string(body["display_name"]) != `"before"` || string(body["protected"]) != "true" {
						t.Error(string(raw))
					}
				}
				if op.name == "Get" && r.URL.Query().Get("resource_type") != "before" {
					t.Error(r.URL)
				}
				if (op.name == "List" || op.name == "All") && r.URL.Query().Get("limit") != "1" {
					t.Error(r.URL)
				}
				code, raw := op.status, op.raw
				if op.name == "Delete" {
					code = 404
					raw = "physical404"
				}
				return namespaceWire(code, io.NopCloser(strings.NewReader(raw))), nil
			})
			v, e := namespaceCall(namespaces.New(client), context.Background(), op.name, namespaceName, o)
			if e != nil || op.name == "Delete" && v != nil || op.name != "Delete" && v == nil || callbacks.Load() != 1 {
				t.Fatal(v, e, callbacks.Load())
			}
		})
	}
	t.Run("live token ordinary source headers snapshot", func(t *testing.T) {
		client := namespaceClient(nil)
		client.MoreHeaders = map[string]string{"X-Source": "before"}
		var calls atomic.Int32
		client.HTTPClient.Transport = namespaceTransport(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Source") != "before" || r.Header.Get("X-Auth-Token") != "live" {
				t.Error(r.Header)
			}
			return namespaceWire(200, io.NopCloser(strings.NewReader(namespaceObject))), nil
		})
		a := namespaces.New(client)
		if a.RawClient() != client || calls.Load() != 0 {
			t.Fatal("constructor/source")
		}
		v, e := a.Get(context.Background(), namespaceName, func(*namespaces.GetOpts) error {
			client.MoreHeaders["X-Source"] = "later"
			client.SetToken("live")
			return nil
		})
		if e != nil || v == nil || calls.Load() != 1 || client.MoreHeaders["X-Source"] != "later" {
			t.Fatal(v, e, calls.Load())
		}
	})
	for _, mode := range []string{"cancel", "provider", "Endpoint"} {
		t.Run("between consumed rows "+mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("row cancellation")
			raw := `{"namespaces":[{},{}]}`
			var calls atomic.Int32
			client := namespaceClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return namespaceWire(200, io.NopCloser(strings.NewReader(raw))), nil
			})
			n, terminal := 0, 0
			for v, e := range namespaces.New(client).List(ctx) {
				if e != nil {
					namespaceProof(t, e, 200, []byte(raw))
					terminal++
					if v != nil || mode == "cancel" && (!errors.Is(e, cause) || !errors.Is(e, context.Canceled)) || mode != "cancel" && !errors.Is(e, resource.ErrInvalidOption) {
						t.Fatal(v, e)
					}
					continue
				}
				n++
				switch mode {
				case "cancel":
					cancel(cause)
				case "provider":
					client.ProviderClient = &gophercloud.ProviderClient{}
				case "Endpoint":
					client.Endpoint = "https://glance.invalid/changed/v2/"
				}
			}
			if n != 1 || terminal != 1 || calls.Load() != 1 {
				t.Fatal(n, terminal, calls.Load())
			}
		})
	}
}

func TestMetadefNamespacesResponseOwnershipAndNativeHooks(t *testing.T) {
	for _, op := range namespaceOperations {
		for _, mode := range []string{"Read", "Close", "cancel", "Read Close cancel", "provider after Close"} {
			t.Run(op.name+" accepted "+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("accepted Read"), errors.New("accepted Close"), errors.New("accepted cause")
				raw := []byte(op.raw)
				if op.name == "Delete" {
					raw = []byte{'a', 'c', 'k', 0xff}
				}
				b := &namespaceBody{Reader: bytes.NewReader(raw)}
				if strings.Contains(mode, "Read") {
					b.Reader = namespaceReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
				}
				if strings.Contains(mode, "Close") {
					b.closeErr = closeCause
				}
				var calls, hooks atomic.Int32
				client := namespaceClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return namespaceWire(op.status, b), nil })
				if mode == "provider after Close" {
					b.closeErr = nil
					b.onClose = func() { client.ProviderClient = &gophercloud.ProviderClient{} }
				} else if strings.Contains(mode, "cancel") {
					b.onClose = func() { cancel(cancelCause) }
				}
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks.Add(1)
					return errors.New("accepted replay")
				}
				v, e := namespaceCall(namespaces.New(client), ctx, op.name, namespaceName, namespaceOptions{})
				proof := namespaceProof(t, e, op.status, raw)
				if calls.Load() != 1 || hooks.Load() != 0 || b.closes.Load() != 1 || strings.Contains(mode, "Read") && !errors.Is(e, readCause) || mode != "provider after Close" && strings.Contains(mode, "Close") && !errors.Is(e, closeCause) || strings.Contains(mode, "cancel") && (!errors.Is(e, cancelCause) || !errors.Is(e, context.Canceled)) || mode == "provider after Close" && !errors.Is(e, resource.ErrInvalidOption) {
					t.Fatal(v, e, calls.Load(), hooks.Load(), b.closes.Load())
				}
				if op.name == "Delete" {
					if v == nil || v.ack == nil || v.ack.Namespace != namespaceName || v.ack.StatusCode != 204 || !bytes.Equal(v.ack.Body, raw) {
						t.Fatal(v, e)
					}
					proof.Body[0] = '!'
					proof.Header.Set("X-Request-Id", "proof changed")
					if v.ack.Body[0] != 'a' || v.ack.Header.Get("X-Request-Id") != "actual-namespace" {
						t.Fatal("ack aliases error evidence", v.ack)
					}
				} else if v != nil {
					t.Fatal("partial typed result", v, e)
				}
			})
		}
	}
	t.Run("opaque204 successful acknowledgement", func(t *testing.T) {
		raw := []byte{0xff, 0, 1}
		b := &namespaceBody{Reader: bytes.NewReader(raw)}
		wire := namespaceWire(204, b)
		client := namespaceClient(func(r *http.Request) (*http.Response, error) {
			if r.Body != nil || r.URL.RawQuery != "" || r.Method != "DELETE" {
				t.Error(r.Method, r.URL, r.Body)
			}
			return wire, nil
		})
		v, e := namespaces.New(client).Delete(context.Background(), namespaceName)
		if e != nil || v == nil || v.Namespace != namespaceName || !bytes.Equal(v.Body, raw) || v.StatusCode != 204 || b.closes.Load() != 1 {
			t.Fatal(v, e, b.closes.Load())
		}
		wire.Header.Set("X-Request-Id", "late")
		raw[0] = 1
		if v.Header.Get("X-Request-Id") != "actual-namespace" || v.Body[0] != 255 {
			t.Fatal("borrowed response alias", v)
		}
	})
	for _, op := range namespaceOperations {
		for _, code := range []int{202, 404, 409} {
			if op.name == "Delete" && code == 404 {
				continue
			}
			t.Run(fmt.Sprintf("%s strict actual%d", op.name, code), func(t *testing.T) {
				raw := []byte("server-owned rejection")
				b := &namespaceBody{Reader: bytes.NewReader(raw)}
				var calls atomic.Int32
				client := namespaceClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return namespaceWire(code, b), nil })
				v, e := namespaceCall(namespaces.New(client), context.Background(), op.name, namespaceName, namespaceOptions{})
				var native gophercloud.ErrUnexpectedResponseCode
				if v != nil || !errors.As(e, &native) || native.Actual != code || !bytes.Equal(native.Body, raw) || native.ResponseHeader.Get("X-Request-Id") != "actual-namespace" || calls.Load() != 1 || b.closes.Load() != 1 {
					t.Fatal(v, e, native, calls.Load(), b.closes.Load())
				}
			})
		}
	}
	for _, mode := range []string{"default", "explicit true", "explicit false"} {
		t.Run("physical404 "+mode, func(t *testing.T) {
			var calls, hooks atomic.Int32
			hookCause := errors.New("strict native callback")
			b := &namespaceBody{Reader: strings.NewReader("actual404")}
			client := namespaceClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return namespaceWire(404, b), nil })
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				hooks.Add(1)
				return hookCause
			}
			opts := []namespaces.DeleteOption{}
			if mode != "default" {
				opts = append(opts, namespaces.WithDeleteIgnoreMissing(mode == "explicit true"))
			}
			v, e := namespaces.New(client).Delete(context.Background(), namespaceName, opts...)
			if v != nil || calls.Load() != 1 || b.closes.Load() != 1 {
				t.Fatal(v, e, calls.Load(), b.closes.Load())
			}
			if mode == "explicit false" {
				if !errors.Is(e, hookCause) || !gophercloud.ResponseCodeIs(e, 404) || hooks.Load() != 1 {
					t.Fatal(e, hooks.Load())
				}
			} else if e != nil || hooks.Load() != 0 {
				t.Fatal(e, hooks.Load())
			}
		})
	}
	for _, mode := range []string{"Read", "Close", "cancel", "Read Close cancel", "source"} {
		t.Run("owned404 failure "+mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			raw := []byte("physical404 partial")
			readCause, closeCause, cancelCause := errors.New("404 Read"), errors.New("404 Close"), errors.New("404 cancel")
			b := &namespaceBody{Reader: bytes.NewReader(raw)}
			if strings.Contains(mode, "Read") {
				b.Reader = namespaceReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
			}
			if strings.Contains(mode, "Close") {
				b.closeErr = closeCause
			}
			var calls, hooks atomic.Int32
			client := namespaceClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return namespaceWire(404, b), nil })
			if strings.Contains(mode, "cancel") {
				b.onClose = func() { cancel(cancelCause) }
			}
			if mode == "source" {
				b.onClose = func() { client.ResourceBase = "https://glance.invalid/replaced/v2/" }
			}
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				hooks.Add(1)
				return errors.New("404 retry")
			}
			v, e := namespaces.New(client).Delete(ctx, namespaceName)
			namespaceProof(t, e, 404, raw)
			if v != nil || calls.Load() != 1 || hooks.Load() != 0 || b.closes.Load() != 1 || strings.Contains(mode, "Read") && !errors.Is(e, readCause) || strings.Contains(mode, "Close") && !errors.Is(e, closeCause) || strings.Contains(mode, "cancel") && (!errors.Is(e, cancelCause) || !errors.Is(e, context.Canceled)) || mode == "source" && !errors.Is(e, resource.ErrInvalidOption) {
				t.Fatal(v, e, calls.Load(), hooks.Load(), b.closes.Load())
			}
		})
	}
	for _, mode := range []string{"transport nested404", "callback nested404", "reauth nested404"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New("unrelated404 cause")
			nested := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{204}, Method: "DELETE", URL: "https://unrelated.invalid", Body: []byte("unrelated404")}
			var calls, hooks atomic.Int32
			b := &namespaceBody{Reader: strings.NewReader("original failure")}
			client := namespaceClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				if mode == "transport nested404" {
					return nil, errors.Join(cause, nested)
				}
				code := 503
				if mode == "reauth nested404" {
					code = 401
				}
				return namespaceWire(code, b), nil
			})
			if mode == "callback nested404" {
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks.Add(1)
					return errors.Join(cause, nested)
				}
			}
			if mode == "reauth nested404" {
				client.ReauthFunc = func(context.Context) error { hooks.Add(1); return errors.Join(cause, nested) }
			}
			v, e := namespaces.New(client).Delete(context.Background(), namespaceName)
			if v != nil || e == nil || calls.Load() != 1 {
				t.Fatal(v, e, calls.Load())
			}
			if mode == "reauth nested404" {
				var reauth *gophercloud.ErrUnableToReauthenticate
				if !errors.As(e, &reauth) || !gophercloud.ResponseCodeIs(reauth.ErrOriginal, 401) || !errors.Is(reauth.ErrReauth, cause) || !gophercloud.ResponseCodeIs(reauth.ErrReauth, 404) || hooks.Load() != 1 {
					t.Fatal(e, reauth, hooks.Load())
				}
			} else if !errors.Is(e, cause) {
				t.Fatal(e)
			}
			if mode != "transport nested404" && b.closes.Load() != 1 {
				t.Fatal(b.closes.Load())
			}
		})
	}
	t.Run("configured prebody hooks and original provider", func(t *testing.T) {
		var calls, reauth, backoff, retries atomic.Int32
		var bodies []*namespaceBody
		transportCause := errors.New("prebody transport")
		client := namespaceClient(nil)
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		client.ReauthFunc = func(context.Context) error { reauth.Add(1); client.SetToken("reauth"); return nil }
		client.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
			backoff.Add(1)
			client.SetToken("backoff")
			return nil
		}
		client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, e error, _ uint) error {
			retries.Add(1)
			raw, _ := json.Marshal(o.JSONBody)
			if !o.KeepResponseBody || o.JSONResponse != nil || o.RawBody != nil || string(raw) != `{"namespace":"OS::Compute::Libvirt"}` {
				t.Error(o, string(raw))
			}
			if gophercloud.ResponseCodeIs(e, 503) {
				client.SetToken("retry503")
				return nil
			}
			if errors.Is(e, transportCause) {
				client.SetToken("retrytransport")
				return nil
			}
			return e
		}
		originalProvider, originalRetry := client.ProviderClient, reflect.ValueOf(client.RetryFunc).Pointer()
		client.HTTPClient.Transport = namespaceTransport(func(r *http.Request) (*http.Response, error) {
			i := int(calls.Add(1)) - 1
			codes := []int{401, 429, 503, 0, 201}
			tokens := []string{"initial", "reauth", "backoff", "retry503", "retrytransport"}
			if i >= len(codes) {
				return nil, errors.New("replay")
			}
			raw, _ := io.ReadAll(r.Body)
			if r.Method != "POST" || r.URL.String() != namespaceBase+namespaceCollection || string(raw) != `{"namespace":"OS::Compute::Libvirt"}` || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != tokens[i] {
				t.Error(r.Method, r.URL, string(raw), r.Header)
			}
			if codes[i] == 0 {
				return nil, transportCause
			}
			b := &namespaceBody{Reader: strings.NewReader(namespaceObject)}
			bodies = append(bodies, b)
			return namespaceWire(codes[i], b), nil
		})
		v, e := namespaces.New(client).Create(context.Background(), namespaceName)
		if e != nil || v == nil || calls.Load() != 5 || reauth.Load() != 1 || backoff.Load() != 1 || retries.Load() != 2 || client.ProviderClient != originalProvider || reflect.ValueOf(client.RetryFunc).Pointer() != originalRetry {
			t.Fatal(v, e, calls.Load(), reauth.Load(), backoff.Load(), retries.Load())
		}
		for _, b := range bodies {
			if b.closes.Load() != 1 {
				t.Fatal(b.closes.Load())
			}
		}
	})
	for _, op := range []string{"Create", "Update"} {
		for _, change := range []string{"in-place RawMessage", "changed JSON", "JSON nil", "JSON null", "KeepResponseBody", "JSONResponse", "RawBody", "unsupported JSONBody"} {
			t.Run(op+" retry guard "+change, func(t *testing.T) {
				callbackCause := errors.New("retry callback")
				var calls, hooks, borrowedReads atomic.Int32
				b := &namespaceBody{Reader: strings.NewReader("original503")}
				borrowed := &namespaceBody{Reader: namespaceReader(func([]byte) (int, error) { borrowedReads.Add(1); return 0, io.EOF })}
				client := namespaceClient(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					raw, _ := io.ReadAll(r.Body)
					if string(raw) != `{"namespace":"OS::Compute::Libvirt"}` {
						t.Error(string(raw))
					}
					return namespaceWire(503, b), nil
				})
				client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, e error, _ uint) error {
					hooks.Add(1)
					if !gophercloud.ResponseCodeIs(e, 503) {
						t.Error(e)
					}
					switch change {
					case "in-place RawMessage":
						raw, ok := o.JSONBody.(json.RawMessage)
						if !ok {
							t.Error(o.JSONBody)
						} else {
							raw[len(raw)-3] = 'x'
						}
					case "changed JSON":
						o.JSONBody = map[string]string{"namespace": "other"}
					case "JSON nil":
						o.JSONBody = nil
					case "JSON null":
						o.JSONBody = json.RawMessage("null")
					case "KeepResponseBody":
						o.KeepResponseBody = false
					case "JSONResponse":
						o.JSONResponse = new(any)
					case "RawBody":
						o.RawBody = borrowed
					case "unsupported JSONBody":
						o.JSONBody = make(chan int)
					}
					return callbackCause
				}
				v, e := namespaceCall(namespaces.New(client), context.Background(), op, namespaceName, namespaceOptions{})
				var native gophercloud.ErrUnexpectedResponseCode
				if v != nil || !errors.Is(e, resource.ErrInvalidOption) || !errors.Is(e, callbackCause) || !errors.As(e, &native) || native.Actual != 503 || string(native.Body) != "original503" || calls.Load() != 1 || hooks.Load() != 1 || b.closes.Load() != 1 || borrowedReads.Load() != 0 || borrowed.closes.Load() != 0 {
					t.Fatal(v, e, native, calls.Load(), hooks.Load())
				}
				if change == "unsupported JSONBody" {
					var encoding *json.UnsupportedTypeError
					if !errors.As(e, &encoding) {
						t.Fatal("encoding cause lost", e)
					}
				}
			})
		}
	}
	t.Run("bodyless null differs from absent", func(t *testing.T) {
		var calls atomic.Int32
		b := &namespaceBody{Reader: strings.NewReader("original503")}
		client := namespaceClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Body != nil {
				t.Error(r.Body)
			}
			return namespaceWire(503, b), nil
		})
		client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			o.JSONBody = json.RawMessage("null")
			return nil
		}
		v, e := namespaces.New(client).Delete(context.Background(), namespaceName)
		if v != nil || !errors.Is(e, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(e, 503) || calls.Load() != 1 || b.closes.Load() != 1 {
			t.Fatal(v, e, calls.Load(), b.closes.Load())
		}
	})
	for _, op := range []string{"Create", "Update"} {
		t.Run(op+" same serialized replacement", func(t *testing.T) {
			var calls, hooks atomic.Int32
			var bodies []*namespaceBody
			code := 201
			if op == "Update" {
				code = 200
			}
			client := namespaceClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				raw, _ := io.ReadAll(r.Body)
				if string(raw) != `{"namespace":"OS::Compute::Libvirt"}` {
					t.Error(string(raw))
				}
				status, reply := 503, "original503"
				if n == 2 {
					status, reply = code, namespaceObject
				}
				b := &namespaceBody{Reader: strings.NewReader(reply)}
				bodies = append(bodies, b)
				return namespaceWire(status, b), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				hooks.Add(1)
				o.JSONBody = map[string]string{"namespace": namespaceName}
				return nil
			}
			v, e := namespaceCall(namespaces.New(client), context.Background(), op, namespaceName, namespaceOptions{})
			if v == nil || e != nil || calls.Load() != 2 || hooks.Load() != 1 {
				t.Fatal(v, e, calls.Load(), hooks.Load())
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
	for _, op := range namespaceOperations {
		t.Run(op.name+" expanded codes retain rejection evidence", func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("unexpected Read"), errors.New("unexpected Close"), errors.New("unexpected cause")
			var calls, hooks atomic.Int32
			var bodies []*namespaceBody
			client := namespaceClient(func(*http.Request) (*http.Response, error) {
				n := calls.Add(1)
				code, raw := 503, "original503"
				if n == 2 {
					code, raw = 202, "unexpected actual202"
				}
				b := &namespaceBody{Reader: strings.NewReader(raw)}
				if n == 2 {
					b.Reader = namespaceReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
					b.closeErr = closeCause
					b.onClose = func() { cancel(cancelCause) }
				}
				bodies = append(bodies, b)
				return namespaceWire(code, b), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, e error, _ uint) error {
				hooks.Add(1)
				if !gophercloud.ResponseCodeIs(e, 503) {
					return e
				}
				o.OkCodes = []int{202}
				return nil
			}
			v, e := namespaceCall(namespaces.New(client), ctx, op.name, namespaceName, namespaceOptions{})
			var native gophercloud.ErrUnexpectedResponseCode
			var owned *resource.ResponseError
			expected := []int{op.status}
			if op.name == "Delete" {
				expected = append(expected, 404)
			}
			target := namespaceBase + namespaceCollection
			if op.named {
				target += "/" + url.PathEscape(namespaceName)
			}
			if v != nil || !errors.As(e, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, expected) || native.Method != op.method || native.URL != target || string(native.Body) != "unexpected actual202" || native.ResponseHeader.Get("X-Request-Id") != "actual-namespace" || !errors.Is(e, readCause) || !errors.Is(e, closeCause) || !errors.Is(e, cancelCause) || !errors.Is(e, context.Canceled) || errors.As(e, &owned) || calls.Load() != 2 || hooks.Load() != 1 {
				t.Fatal(v, e, native, calls.Load(), hooks.Load())
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
	for _, redirect := range []string{"same target", "foreign origin", "changed path", "changed query", "changed method"} {
		t.Run("configured redirect "+redirect, func(t *testing.T) {
			var calls, redirects atomic.Int32
			var bodies []*namespaceBody
			client := namespaceClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				b := &namespaceBody{Reader: strings.NewReader(namespaceObject)}
				bodies = append(bodies, b)
				if n == 2 {
					raw, _ := io.ReadAll(r.Body)
					if r.Method != "POST" || string(raw) != `{"namespace":"OS::Compute::Libvirt"}` {
						t.Error(r.Method, string(raw))
					}
					return namespaceWire(201, b), nil
				}
				code, target := 307, namespaceBase+namespaceCollection
				switch redirect {
				case "foreign origin":
					target = "https://foreign.invalid/namespace"
				case "changed path":
					target = namespaceBase + "metadefs/other"
				case "changed query":
					target += "?resource_type=other"
				case "changed method":
					code = 303
				}
				wire := namespaceWire(code, b)
				wire.Header.Set("Location", target)
				return wire, nil
			})
			client.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects.Add(1); return nil }
			v, e := namespaces.New(client).Create(context.Background(), namespaceName)
			if redirect == "same target" {
				if e != nil || v == nil || calls.Load() != 2 || redirects.Load() != 1 {
					t.Fatal(v, e, calls.Load(), redirects.Load())
				}
			} else if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 1 {
				t.Fatal(v, e, calls.Load(), redirects.Load())
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
}
