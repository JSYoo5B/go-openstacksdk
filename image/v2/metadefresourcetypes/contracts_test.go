package metadefresourcetypes_test

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

	rt "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const rtPrefix = "/reverse/resource-type/glance/v2/"
const rtBase = "https://glance.invalid" + rtPrefix
const rtParent = "OS::Compute::Libvirt"
const rtName = "OS::Compute::Server Extra"
const rtCollection = "metadefs/namespaces/" + rtParent + "/resource_types"
const rtGlobal = "metadefs/resource_types"
const rtJSON = `{"name":"server response","prefix":"hw_","properties_target":"image","created_at":"literal-date","updated_at":"","protected":true,"links":42,"self":false,"schema":null,"id":{},"namespace_name":"foreign","huge":9007199254740993,"exponent":1e400}`
const rtGlobalJSON = `{"resource_types":[` + rtJSON + `],"next":"https://foreign.invalid/follow","links":false}`
const rtListJSON = `{"resource_type_associations":[` + rtJSON + `],"next":"https://foreign.invalid/follow","links":false}`

type rtTransport func(*http.Request) (*http.Response, error)

func (f rtTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	v, e := f(r)
	if v != nil && v.Request == nil {
		v.Request = r
	}
	return v, e
}

type rtBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *rtBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type rtReader func([]byte) (int, error)

func (f rtReader) Read(p []byte) (int, error) { return f(p) }
func rtWire(code int, b io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"actual-resource-type"}}, Body: b}
}
func rtClient(f rtTransport) *gophercloud.ServiceClient {
	p := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: f}}
	p.UseTokenLock()
	p.SetToken("initial")
	return &gophercloud.ServiceClient{ProviderClient: p, Type: "image", Endpoint: rtBase}
}
func rtScope(t *testing.T, c *gophercloud.ServiceClient) (*rt.API, *rt.NamespaceScope) {
	t.Helper()
	a := rt.New(c)
	s, e := a.InNamespace(context.Background(), rtParent)
	if e != nil || s == nil || a.RawClient() != c || s.RawClient() != c || s.NamespaceName() != rtParent {
		t.Fatal(s, e)
	}
	return a, s
}

type rtOptions struct {
	create   []rt.CreateOption
	deletion []rt.DeleteOption
	list     []rt.ListOption
}
type rtResult struct {
	value  *rt.Association
	ack    *rt.Acknowledgement
	rows   []*rt.Association
	global []*rt.ResourceType
}

func rtCall(a *rt.API, s *rt.NamespaceScope, ctx context.Context, op string, o rtOptions) (*rtResult, error) {
	switch op {
	case "Create":
		v, e := s.Create(ctx, rtName, o.create...)
		if v == nil {
			return nil, e
		}
		return &rtResult{value: v}, e
	case "Delete":
		v, e := s.Delete(ctx, rtName, o.deletion...)
		if v == nil {
			return nil, e
		}
		return &rtResult{ack: v}, e
	case "All":
		v, e := s.All(ctx, o.list...)
		if v == nil {
			return nil, e
		}
		return &rtResult{rows: v}, e
	case "GlobalAll":
		v, e := a.All(ctx, o.list...)
		if v == nil {
			return nil, e
		}
		return &rtResult{global: v}, e
	case "List":
		v := make([]*rt.Association, 0)
		for row, e := range s.List(ctx, o.list...) {
			if e != nil {
				return nil, e
			}
			v = append(v, row)
		}
		return &rtResult{rows: v}, nil
	case "GlobalList":
		v := make([]*rt.ResourceType, 0)
		for row, e := range a.List(ctx, o.list...) {
			if e != nil {
				return nil, e
			}
			v = append(v, row)
		}
		return &rtResult{global: v}, nil
	default:
		panic(op)
	}
}

var rtOperations = []struct {
	name, method, body, raw string
	status                  int
	global                  bool
}{
	{"Create", "POST", `{"name":"OS::Compute::Server Extra"}`, rtJSON, 201, false},
	{"Delete", "DELETE", "", "", 204, false},
	{"List", "GET", "", rtListJSON, 200, false},
	{"All", "GET", "", rtListJSON, 200, false},
	{"GlobalList", "GET", "", rtGlobalJSON, 200, true},
	{"GlobalAll", "GET", "", rtGlobalJSON, 200, true},
}

func rtTarget(op string) string {
	if strings.HasPrefix(op, "Global") {
		return rtBase + rtGlobal
	}
	u := rtBase + rtCollection
	if op == "Delete" {
		u += "/" + url.PathEscape(rtName)
	}
	return u
}
func rtProof(t *testing.T, e error, code int, raw []byte) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(e, &proof) || proof.StatusCode != code || !bytes.Equal(proof.Body, raw) || proof.Header.Get("X-Request-Id") != "actual-resource-type" {
		t.Fatalf("owned response %v %+v", e, proof)
	}
	return proof
}
func rtHeaders() rtOptions {
	return rtOptions{create: []rt.CreateOption{rt.WithCreateHeaders(map[string]string{"X-Option": "owned"}), rt.WithCreateHeader("X-Final", "yes")}, deletion: []rt.DeleteOption{rt.WithDeleteHeaders(map[string]string{"X-Option": "owned"}), rt.WithDeleteHeader("X-Final", "yes")}, list: []rt.ListOption{rt.WithListHeaders(map[string]string{"X-Option": "owned"}), rt.WithListHeader("X-Final", "yes")}}
}

func TestMetadefResourceTypesFixedRoutesAndPayloads(t *testing.T) {
	for _, op := range rtOperations {
		t.Run(op.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			c := cloud.Client("image", "/unused/")
			c.ResourceBase = cloud.Server.URL + rtPrefix
			c.Microversion = "2.2"
			var calls atomic.Int32
			a, s := rtScope(t, c)
			if calls.Load() != 0 {
				t.Fatal("scope HTTP")
			}
			c.MoreHeaders = map[string]string{"X-Source": "latest"}
			cloud.Provider.SetToken("live")
			cloud.Mux.HandleFunc(rtPrefix, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				want := strings.TrimPrefix(rtTarget(op.name), "https://glance.invalid")
				raw, _ := io.ReadAll(r.Body)
				if r.Method != op.method || r.URL.EscapedPath() != want || r.URL.RawQuery != "" || string(raw) != op.body || r.Header.Get("X-Auth-Token") != "live" || r.Header.Get("X-Source") != "latest" || r.Header.Get("X-Option") != "owned" || r.Header.Get("X-Final") != "yes" || r.Header.Get("OpenStack-API-Version") != "image 2.2" {
					t.Error(r.Method, r.RequestURI, string(raw), r.Header, want)
				}
				w.Header().Set("X-Request-Id", "actual-resource-type")
				w.Header().Set("Location", "https://passive.invalid/different")
				testcloud.JSON(w, op.status, op.raw)
			})
			v, e := rtCall(a, s, context.Background(), op.name, rtHeaders())
			if e != nil || v == nil || calls.Load() != 1 {
				t.Fatal(v, e, calls.Load())
			}
			if v.value != nil && (*v.value.Name != "server response" || v.value.StatusCode != 201 || v.value.Header.Get("Location") != "https://passive.invalid/different") {
				t.Fatal(v.value)
			}
			if v.ack != nil && (v.ack.Namespace != rtParent || v.ack.Name == nil || *v.ack.Name != rtName || v.ack.StatusCode != 204) {
				t.Fatal(v.ack)
			}
			if v.rows != nil && (len(v.rows) != 1 || *v.rows[0].Name != "server response") {
				t.Fatal(v.rows)
			}
			if v.global != nil && (len(v.global) != 1 || *v.global[0].Name != "server response") {
				t.Fatal(v.global)
			}
		})
	}
	for _, test := range []struct {
		name string
		opts []rt.CreateOption
		want string
	}{
		{"omitted", nil, `{"name":"OS::Compute::Server Extra"}`},
		{"explicit empty", []rt.CreateOption{rt.WithCreatePrefix(""), rt.WithCreatePropertiesTarget("")}, `{"name":"OS::Compute::Server Extra","prefix":"","properties_target":""}`},
		{"separatorless literal", []rt.CreateOption{rt.WithCreatePrefix("hw_"), rt.WithCreatePropertiesTarget("image\nmetadata")}, `{"name":"OS::Compute::Server Extra","prefix":"hw_","properties_target":"image\nmetadata"}`},
		{"80 rune UTF8", []rt.CreateOption{rt.WithCreatePrefix(strings.Repeat("界", 80)), rt.WithCreatePropertiesTarget("\t\x00")}, ""},
	} {
		t.Run("POST "+test.name, func(t *testing.T) {
			var calls atomic.Int32
			c := rtClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				raw, _ := io.ReadAll(r.Body)
				if r.Method != "POST" || r.URL.String() != rtBase+rtCollection {
					t.Error(r)
				}
				if test.want != "" && string(raw) != test.want {
					t.Error(string(raw), test.want)
				}
				if test.name == "80 rune UTF8" {
					var body map[string]string
					if e := json.Unmarshal(raw, &body); e != nil || body["prefix"] != strings.Repeat("界", 80) || body["properties_target"] != "\t\x00" {
						t.Error(body, e)
					}
				}
				return rtWire(201, &rtBody{Reader: strings.NewReader(rtJSON)}), nil
			})
			_, s := rtScope(t, c)
			v, e := s.Create(context.Background(), rtName, test.opts...)
			if v == nil || e != nil || calls.Load() != 1 {
				t.Fatal(v, e, calls.Load())
			}
		})
	}
	t.Run("literal identities escaped once", func(t *testing.T) {
		parent, name := " Parent::界 ", " Type::界 "
		var calls atomic.Int32
		c := rtClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			want := rtPrefix + "metadefs/namespaces/" + url.PathEscape(parent) + "/resource_types/" + url.PathEscape(name)
			if r.URL.EscapedPath() != want || r.URL.RawQuery != "" || r.Method != "DELETE" || r.Body != nil {
				t.Error(r.RequestURI, r.URL.EscapedPath(), want)
			}
			return rtWire(204, &rtBody{Reader: strings.NewReader("")}), nil
		})
		s, e := rt.New(c).InNamespace(context.Background(), parent)
		if e != nil {
			t.Fatal(e)
		}
		v, e := s.Delete(context.Background(), name)
		if e != nil || v == nil || v.Namespace != parent || v.Name == nil || *v.Name != name || calls.Load() != 1 {
			t.Fatal(v, e, calls.Load())
		}
	})
	t.Run("cloud errors no lookup fallback", func(t *testing.T) {
		for _, code := range []int{400, 403, 404, 409} {
			var calls atomic.Int32
			c := rtClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Method != "POST" {
					t.Error(r.Method)
				}
				return rtWire(code, &rtBody{Reader: strings.NewReader("server policy")}), nil
			})
			_, s := rtScope(t, c)
			v, e := s.Create(context.Background(), rtName)
			var native gophercloud.ErrUnexpectedResponseCode
			if v != nil || !errors.As(e, &native) || native.Actual != code || calls.Load() != 1 {
				t.Fatal(v, e, native, calls.Load())
			}
		}
	})
}

func TestMetadefResourceTypesCanonicalModelsAndRawOwnership(t *testing.T) {
	t.Run("canonical passive independent models", func(t *testing.T) {
		var g rt.ResourceType
		var a rt.Association
		if e := json.Unmarshal([]byte(rtJSON), &g); e != nil {
			t.Fatal(e)
		}
		if e := json.Unmarshal([]byte(rtJSON), &a); e != nil {
			t.Fatal(e)
		}
		if g.Name == nil || *g.Name != "server response" || g.CreatedAt == nil || *g.CreatedAt != "literal-date" || g.UpdatedAt == nil || *g.UpdatedAt != "" || g.Links != nil || a.Prefix == nil || *a.Prefix != "hw_" || a.PropertiesTarget == nil || *a.PropertiesTarget != "image" || a.Links != nil {
			t.Fatal(g, a)
		}
		for _, m := range []map[string]json.RawMessage{g.Body, a.Body} {
			if string(m["huge"]) != "9007199254740993" || string(m["exponent"]) != "1e400" || string(m["protected"]) != "true" || string(m["schema"]) != "null" || string(m["name"]) != `"server response"` {
				t.Fatal(m)
			}
		}
		*a.Name = "typed change"
		*a.CreatedAt = "typed date"
		a.Body["prefix"][1] = '!'
		a.Body["name"][1] = '!'
		if *a.Prefix != "hw_" || string(g.Body["prefix"]) != `"hw_"` || *g.Name != "server response" || *g.CreatedAt != "literal-date" {
			t.Fatal("model aliases", g, a)
		}
		for _, raw := range []string{`{}`, `{"name":null,"created_at":null,"updated_at":null,"prefix":null,"properties_target":null}`, `{"Name":12,"Created_At":{},"Prefix":true,"Properties_Target":false,"links":null}`} {
			var x rt.Association
			var y rt.ResourceType
			if e := json.Unmarshal([]byte(raw), &x); e != nil {
				t.Fatal(e)
			}
			if e := json.Unmarshal([]byte(raw), &y); e != nil {
				t.Fatal(e)
			}
			if x.Name != nil || x.Prefix != nil || x.PropertiesTarget != nil || x.CreatedAt != nil || x.UpdatedAt != nil || x.Links != nil || y.Name != nil || y.CreatedAt != nil || y.Links != nil {
				t.Fatal(x, y)
			}
		}
	})
	for _, raw := range []string{`null`, `[]`, `1`, `false`, `"text"`, `{"name":5}`, `{"created_at":[]}`, `{"updated_at":true}`, `{"prefix":2}`, `{"properties_target":{}}`, "{\"extension\":\"\xff\"}", `{"name":`} {
		t.Run("invalid POST "+raw, func(t *testing.T) {
			b := &rtBody{Reader: strings.NewReader(raw)}
			c := rtClient(func(*http.Request) (*http.Response, error) { return rtWire(201, b), nil })
			_, s := rtScope(t, c)
			v, e := s.Create(context.Background(), rtName)
			rtProof(t, e, 201, []byte(raw))
			if v != nil || b.closes.Load() != 1 {
				t.Fatal(v, e, b.closes.Load())
			}
		})
	}
	t.Run("failed canonical decode leaves existing value intact", func(t *testing.T) {
		var g rt.ResourceType
		var a rt.Association
		if e := json.Unmarshal([]byte(rtJSON), &g); e != nil {
			t.Fatal(e)
		}
		if e := json.Unmarshal([]byte(rtJSON), &a); e != nil {
			t.Fatal(e)
		}
		g.Header = http.Header{"X-Previous": {"owned"}}
		a.Header = http.Header{"X-Previous": {"owned"}}
		g.StatusCode, a.StatusCode = 200, 201
		var cause *json.UnmarshalTypeError
		e := json.Unmarshal([]byte(`{"name":"new","created_at":5}`), &g)
		if !errors.As(e, &cause) || g.Name == nil || *g.Name != "server response" || g.CreatedAt == nil || *g.CreatedAt != "literal-date" || g.Header.Get("X-Previous") != "owned" || g.StatusCode != 200 {
			t.Fatal(g, e)
		}
		e = json.Unmarshal([]byte(`{"name":"new","properties_target":false}`), &a)
		if !errors.As(e, &cause) || a.Name == nil || *a.Name != "server response" || a.PropertiesTarget == nil || *a.PropertiesTarget != "image" || a.Header.Get("X-Previous") != "owned" || a.StatusCode != 201 {
			t.Fatal(a, e)
		}
	})
	t.Run("global prefix is raw not canonical", func(t *testing.T) {
		raw := `{"name":"","prefix":false,"properties_target":{},"protected":5,"links":false}`
		var g rt.ResourceType
		if e := json.Unmarshal([]byte(raw), &g); e != nil || g.Name == nil || *g.Name != "" || string(g.Body["prefix"]) != "false" || g.Links != nil {
			t.Fatal(g, e)
		}
	})
	t.Run("typed results raw rows and headers independent", func(t *testing.T) {
		raw := []byte(rtListJSON)
		b := &rtBody{Reader: bytes.NewReader(raw)}
		wire := rtWire(200, b)
		c := rtClient(func(*http.Request) (*http.Response, error) { return wire, nil })
		_, s := rtScope(t, c)
		rows, e := s.All(context.Background())
		if e != nil || len(rows) != 1 || rows[0].StatusCode != 200 || rows[0].Header.Get("X-Request-Id") != "actual-resource-type" {
			t.Fatal(rows, e)
		}
		raw[0] = '!'
		wire.Header.Set("X-Request-Id", "borrowed mutated")
		rows[0].Body["name"][1] = '!'
		if *rows[0].Name != "server response" || rows[0].Header.Get("X-Request-Id") != "actual-resource-type" {
			t.Fatal(rows[0])
		}
	})
}

func rtEach(a *rt.API, s *rt.NamespaceScope, ctx context.Context, global bool, opts []rt.ListOption, yield func(*resource.Metadata, *string, error) bool) {
	if global {
		for v, e := range a.List(ctx, opts...) {
			if e != nil {
				yield(nil, nil, e)
				return
			}
			if !yield(&v.Metadata, v.Name, nil) {
				return
			}
		}
	} else {
		for v, e := range s.List(ctx, opts...) {
			if e != nil {
				yield(nil, nil, e)
				return
			}
			if !yield(&v.Metadata, v.Name, nil) {
				return
			}
		}
	}
}

func TestMetadefResourceTypesFiniteIterationAndLocalCaps(t *testing.T) {
	for _, global := range []bool{false, true} {
		key := "resource_type_associations"
		label := "associations"
		if global {
			key = "resource_types"
			label = "global"
		}
		for _, cap := range []int{0, 1, 3, 20} {
			t.Run(fmt.Sprintf("%s cap%d no wire query", label, cap), func(t *testing.T) {
				raw := `{"` + key + `":[{"name":"z"},{"name":"a"}],"next":"https://foreign.invalid/wrong","` + key + `_links":[{"rel":"next","href":"/v2/wrong"}]}`
				var calls atomic.Int32
				c := rtClient(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					want := rtBase + rtCollection
					if global {
						want = rtBase + rtGlobal
					}
					if r.Method != "GET" || r.Body != nil || r.URL.String() != want || r.URL.RawQuery != "" {
						t.Error(r)
					}
					w := rtWire(200, &rtBody{Reader: strings.NewReader(raw)})
					w.Header.Set("Link", `<https://foreign.invalid/follow>; rel="next"`)
					return w, nil
				})
				a, s := rtScope(t, c)
				var names []string
				rtEach(a, s, context.Background(), global, []rt.ListOption{rt.WithListMaxItems(cap)}, func(m *resource.Metadata, n *string, e error) bool {
					if e != nil {
						t.Fatal(e)
					}
					if m.StatusCode != 200 {
						t.Error(m)
					}
					names = append(names, *n)
					return true
				})
				want := []string{"z", "a"}
				if cap == 1 {
					want = want[:1]
				}
				if !reflect.DeepEqual(names, want) || calls.Load() != 1 {
					t.Fatal(names, calls.Load())
				}
			})
		}
		for _, mode := range []string{"cap", "break", "unlimited"} {
			t.Run(label+" consumed rows "+mode, func(t *testing.T) {
				raw := `{"` + key + `":[{"name":"first"},null,{"name":5}]}`
				var calls atomic.Int32
				c := rtClient(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					return rtWire(200, &rtBody{Reader: strings.NewReader(raw)}), nil
				})
				a, s := rtScope(t, c)
				opts := []rt.ListOption{}
				if mode == "cap" {
					opts = append(opts, rt.WithListMaxItems(1))
				}
				n, terminal := 0, 0
				rtEach(a, s, context.Background(), global, opts, func(_ *resource.Metadata, name *string, e error) bool {
					if e != nil {
						terminal++
						rtProof(t, e, 200, []byte(raw))
						return false
					}
					n++
					if *name != "first" {
						t.Error(*name)
					}
					return mode != "break"
				})
				wantTerminal := 0
				if mode == "unlimited" {
					wantTerminal = 1
				}
				if n != 1 || terminal != wantTerminal || calls.Load() != 1 {
					t.Fatal(n, terminal, calls.Load())
				}
				if mode == "unlimited" {
					v, e := rtCall(a, s, context.Background(), map[bool]string{false: "All", true: "GlobalAll"}[global], rtOptions{})
					if v != nil || e == nil {
						t.Fatal("All partial rows", v, e)
					}
					rtProof(t, e, 200, []byte(raw))
				}
			})
		}
		for _, raw := range []string{`null`, `[]`, `{}`, `{"` + key + `":null}`, `{"` + key + `":{}}`, `{"` + key + `":"wrong"}`, `{"` + key + `":[{"name":"good"}],"bad":}`, "{\"" + key + "\":[{}],\"other\":\"\xff\"}"} {
			t.Run(label+" envelope "+raw, func(t *testing.T) {
				var n int
				c := rtClient(func(*http.Request) (*http.Response, error) {
					return rtWire(200, &rtBody{Reader: strings.NewReader(raw)}), nil
				})
				a, s := rtScope(t, c)
				rtEach(a, s, context.Background(), global, nil, func(_ *resource.Metadata, _ *string, e error) bool {
					if e == nil {
						n++
						t.Error("invalid envelope yielded row")
					} else {
						rtProof(t, e, 200, []byte(raw))
					}
					return true
				})
				if n != 0 {
					t.Fatal(n)
				}
			})
		}
		t.Run(label+" empty nonnil All", func(t *testing.T) {
			c := rtClient(func(*http.Request) (*http.Response, error) {
				return rtWire(200, &rtBody{Reader: strings.NewReader(`{"` + key + `":[]}`)}), nil
			})
			a, s := rtScope(t, c)
			v, e := rtCall(a, s, context.Background(), map[bool]string{false: "All", true: "GlobalAll"}[global], rtOptions{})
			if e != nil || v == nil || global && v.global == nil || !global && v.rows == nil {
				t.Fatal(v, e)
			}
		})
		t.Run(label+" repeated wire rows independently owned", func(t *testing.T) {
			raw := `{"` + key + `":[{"name":"same","huge":9007199254740993},{"name":"same","huge":9007199254740993}]}`
			c := rtClient(func(*http.Request) (*http.Response, error) {
				return rtWire(200, &rtBody{Reader: strings.NewReader(raw)}), nil
			})
			a, s := rtScope(t, c)
			var first *resource.Metadata
			var names []string
			rtEach(a, s, context.Background(), global, nil, func(m *resource.Metadata, name *string, e error) bool {
				if e != nil {
					t.Fatal(e)
				}
				names = append(names, *name)
				if first == nil {
					first = m
					*name = "caller"
					m.Body["huge"][0] = '1'
					m.Header.Set("X-Request-Id", "caller")
				} else if string(m.Body["huge"]) != "9007199254740993" || m.Header.Get("X-Request-Id") != "actual-resource-type" {
					t.Fatal("row aliases", m)
				}
				return true
			})
			if !reflect.DeepEqual(names, []string{"same", "same"}) {
				t.Fatal(names)
			}
		})
		for _, mode := range []string{"cancel", "API", "provider", "Endpoint"} {
			t.Run(label+" before next consumed row "+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("between rows")
				raw := `{"` + key + `":[{},{}]}`
				var calls atomic.Int32
				c := rtClient(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					return rtWire(200, &rtBody{Reader: strings.NewReader(raw)}), nil
				})
				a, s := rtScope(t, c)
				n, terminal := 0, 0
				rtEach(a, s, ctx, global, nil, func(_ *resource.Metadata, _ *string, e error) bool {
					if e != nil {
						terminal++
						rtProof(t, e, 200, []byte(raw))
						if mode == "cancel" {
							if !errors.Is(e, cause) || !errors.Is(e, context.Canceled) {
								t.Error(e)
							}
						} else if !errors.Is(e, resource.ErrInvalidOption) {
							t.Error(e)
						}
						return false
					}
					n++
					switch mode {
					case "cancel":
						cancel(cause)
					case "API":
						*a = *rt.New(rtClient(nil))
					case "provider":
						c.ProviderClient = &gophercloud.ProviderClient{}
					case "Endpoint":
						c.Endpoint = "https://glance.invalid/changed/v2/"
					}
					return true
				})
				if n != 1 || terminal != 1 || calls.Load() != 1 {
					t.Fatal(n, terminal, calls.Load())
				}
			})
		}
	}
	t.Run("lazy reusable parallel shared ListOption snapshot", func(t *testing.T) {
		var calls, callbacks atomic.Int32
		c := rtClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Option") != "owned" || r.URL.RawQuery != "" {
				t.Error(r)
			}
			return rtWire(200, &rtBody{Reader: strings.NewReader(rtGlobalJSON)}), nil
		})
		a, _ := rtScope(t, c)
		opts := []rt.ListOption{func(o *rt.ListOpts) error {
			callbacks.Add(1)
			o.Headers = map[string]string{"X-Option": "owned"}
			return nil
		}}
		seq := a.List(context.Background(), opts...)
		opts[0] = nil
		if calls.Load() != 0 || callbacks.Load() != 0 {
			t.Fatal("not lazy")
		}
		var wg sync.WaitGroup
		for range 2 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				n := 0
				for v, e := range seq {
					if e != nil || v == nil {
						t.Error(v, e)
						return
					}
					n++
				}
				if n != 1 {
					t.Error(n)
				}
			}()
		}
		wg.Wait()
		if calls.Load() != 2 || callbacks.Load() != 2 {
			t.Fatal(calls.Load(), callbacks.Load())
		}
	})
	t.Run("global capture is per execution scope target is lifetime", func(t *testing.T) {
		var calls atomic.Int32
		c := rtClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.URL.String() != "https://glance.invalid/new/v2/"+rtGlobal {
				t.Error(r.URL)
			}
			return rtWire(200, &rtBody{Reader: strings.NewReader(rtGlobalJSON)}), nil
		})
		a, s := rtScope(t, c)
		seq := a.List(context.Background())
		c.Endpoint = "https://glance.invalid/new/v2/"
		n := 0
		for v, e := range seq {
			if e != nil || v == nil {
				t.Fatal(v, e)
			}
			n++
		}
		v, e := s.All(context.Background())
		if n != 1 || v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 1 {
			t.Fatal(n, v, e, calls.Load())
		}
	})
}

func TestMetadefResourceTypesOwnedPreparationAndSource(t *testing.T) {
	invalidNames := []string{"", ".", "..", "bad/name", "bad\\name", "literal%2F", "bad?query", "bad#fragment", "line\nname", "bad\x7fname", strings.Repeat("界", 81), string([]byte{255})}
	for _, name := range invalidNames {
		t.Run("unsafe literal "+name, func(t *testing.T) {
			var calls, callbacks atomic.Int32
			c := rtClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
			bad, e := rt.New(c).InNamespace(context.Background(), name)
			if bad != nil || !errors.Is(e, resource.ErrInvalidOption) {
				t.Fatal(bad, e)
			}
			_, s := rtScope(t, c)
			for _, op := range []string{"Create", "Delete"} {
				var e error
				if op == "Create" {
					_, e = s.Create(context.Background(), name, func(*rt.CreateOpts) error { callbacks.Add(1); return nil })
				} else {
					_, e = s.Delete(context.Background(), name, func(*rt.DeleteOpts) error { callbacks.Add(1); return nil })
				}
				if !errors.Is(e, resource.ErrInvalidOption) {
					t.Fatal(e)
				}
			}
			if calls.Load() != 0 || callbacks.Load() != 0 {
				t.Fatal(calls.Load(), callbacks.Load())
			}
		})
	}
	for _, mode := range []string{"nil context", "cancel", "nil client", "nil provider", "wrong type", "query base", "relative base", "bad microversion", "protected source"} {
		t.Run("scope source "+mode, func(t *testing.T) {
			ctx := context.Background()
			var calls atomic.Int32
			c := rtClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
			cause := errors.New("scope canceled")
			switch mode {
			case "nil context":
				ctx = nil
			case "cancel":
				x, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx = x
			case "nil client":
				c = nil
			case "nil provider":
				c.ProviderClient = nil
			case "wrong type":
				c.Type = "compute"
			case "query base":
				c.ResourceBase = rtBase + "?query=1"
			case "relative base":
				c.Endpoint = "/relative/v2/"
			case "bad microversion":
				c.Microversion = "2.2\nforged"
			case "protected source":
				c.MoreHeaders = map[string]string{"X-Auth-Token": "forged"}
			}
			a := rt.New(c)
			s, e := a.InNamespace(ctx, rtParent)
			if s != nil || e == nil || calls.Load() != 0 {
				t.Fatal(s, e, calls.Load())
			}
			if mode == "wrong type" {
				if !errors.Is(e, resource.ErrUnsupported) {
					t.Fatal(e)
				}
			} else if mode == "cancel" {
				if !errors.Is(e, context.Canceled) || !errors.Is(e, cause) {
					t.Fatal(e)
				}
			} else if !errors.Is(e, resource.ErrInvalidOption) {
				t.Fatal(e)
			}
			var callbacks atomic.Int32
			rows, e := a.All(ctx, func(*rt.ListOpts) error { callbacks.Add(1); return nil })
			if rows != nil || e == nil || callbacks.Load() != 0 || calls.Load() != 0 {
				t.Fatal(rows, e, callbacks.Load(), calls.Load())
			}
		})
	}
	for _, op := range rtOperations {
		for _, mode := range []string{"nil context", "cancel", "nil receiver", "API", "provider", "Endpoint", "base", "type", "Microversion", "protected source"} {
			t.Run(op.name+" before callback "+mode, func(t *testing.T) {
				var calls, callbacks atomic.Int32
				c := rtClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				a, s := rtScope(t, c)
				ctx := context.Background()
				cause := errors.New("preflight canceled")
				switch mode {
				case "nil context":
					ctx = nil
				case "cancel":
					x, cancel := context.WithCancelCause(ctx)
					cancel(cause)
					ctx = x
				case "nil receiver":
					a = nil
					s = nil
				case "API":
					*a = *rt.New(rtClient(nil))
				case "provider":
					c.ProviderClient = nil
				case "Endpoint":
					c.Endpoint = "https://changed.invalid/v2/"
				case "base":
					c.ResourceBase = "https://changed.invalid/v2/"
				case "type":
					c.Type = "compute"
				case "Microversion":
					c.Microversion = "2.3"
				case "protected source":
					c.MoreHeaders = map[string]string{"Authorization": "forged"}
				}
				// Valid changed global targets are captured anew rather than lifetime errors.
				if op.global && (mode == "API" || mode == "Endpoint" || mode == "base" || mode == "Microversion") {
					return
				}
				o := rtOptions{create: []rt.CreateOption{func(*rt.CreateOpts) error { callbacks.Add(1); return nil }}, deletion: []rt.DeleteOption{func(*rt.DeleteOpts) error { callbacks.Add(1); return nil }}, list: []rt.ListOption{func(*rt.ListOpts) error { callbacks.Add(1); return nil }}}
				v, e := rtCall(a, s, ctx, op.name, o)
				if v != nil || e == nil || calls.Load() != 0 || callbacks.Load() != 0 {
					t.Fatal(v, e, calls.Load(), callbacks.Load())
				}
				if mode == "cancel" && (!errors.Is(e, cause) || !errors.Is(e, context.Canceled)) {
					t.Fatal(e)
				}
			})
		}
	}
	for _, op := range rtOperations {
		for _, mode := range []string{"nil option", "callback error", "callback cancel", "API", "provider", "Endpoint", "base", "type", "Microversion", "protected source"} {
			t.Run(op.name+" callback "+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("callback cause")
				var calls atomic.Int32
				c := rtClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				a, s := rtScope(t, c)
				mutate := func() error {
					switch mode {
					case "callback error":
						return cause
					case "callback cancel":
						cancel(cause)
					case "API":
						*a = *rt.New(rtClient(nil))
					case "provider":
						c.ProviderClient = &gophercloud.ProviderClient{}
					case "Endpoint":
						c.Endpoint = "https://changed.invalid/v2/"
					case "base":
						c.ResourceBase = "https://changed.invalid/v2/"
					case "type":
						c.Type = "compute"
					case "Microversion":
						c.Microversion = "2.3"
					case "protected source":
						c.MoreHeaders = map[string]string{"X-Auth-Token": "forged"}
					}
					return nil
				}
				o := rtOptions{create: []rt.CreateOption{func(*rt.CreateOpts) error { return mutate() }}, deletion: []rt.DeleteOption{func(*rt.DeleteOpts) error { return mutate() }}, list: []rt.ListOption{func(*rt.ListOpts) error { return mutate() }}}
				if mode == "nil option" {
					o = rtOptions{create: []rt.CreateOption{nil}, deletion: []rt.DeleteOption{nil}, list: []rt.ListOption{nil}}
				}
				v, e := rtCall(a, s, ctx, op.name, o)
				if v != nil || e == nil || calls.Load() != 0 {
					t.Fatal(v, e, calls.Load())
				}
				if strings.HasPrefix(mode, "callback") {
					if !errors.Is(e, cause) {
						t.Fatal(e)
					}
				} else if !errors.Is(e, resource.ErrInvalidOption) {
					t.Fatal(e)
				}
			})
		}
	}
	for _, op := range rtOperations {
		for _, header := range []string{"X-Auth-Token", "X-Service-Token", "Authorization", "Host", "Cookie", "Content-Type", "Accept", "Content-Length", "Transfer-Encoding", "Connection", "OpenStack-API-Version", "Bad Header"} {
			t.Run(op.name+" header "+header, func(t *testing.T) {
				var calls atomic.Int32
				c := rtClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				a, s := rtScope(t, c)
				o := rtOptions{create: []rt.CreateOption{rt.WithCreateHeader(header, "bad")}, deletion: []rt.DeleteOption{rt.WithDeleteHeader(header, "bad")}, list: []rt.ListOption{rt.WithListHeader(header, "bad")}}
				v, e := rtCall(a, s, context.Background(), op.name, o)
				if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
					t.Fatal(v, e, calls.Load())
				}
			})
		}
	}
	for _, test := range []struct {
		name string
		o    rtOptions
		op   string
	}{
		{"long prefix", rtOptions{create: []rt.CreateOption{rt.WithCreatePrefix(strings.Repeat("界", 81))}}, "Create"},
		{"bad prefix UTF8", rtOptions{create: []rt.CreateOption{rt.WithCreatePrefix(string([]byte{255}))}}, "Create"},
		{"long target", rtOptions{create: []rt.CreateOption{rt.WithCreatePropertiesTarget(strings.Repeat("界", 81))}}, "Create"},
		{"bad target UTF8", rtOptions{create: []rt.CreateOption{rt.WithCreatePropertiesTarget(string([]byte{255}))}}, "Create"},
		{"negative local cap", rtOptions{list: []rt.ListOption{rt.WithListMaxItems(-1)}}, "GlobalAll"},
		{"alias conflict", rtOptions{create: []rt.CreateOption{rt.WithCreateHeaders(map[string]string{"X-Alias": "one", "x-alias": "two"})}}, "Create"},
		{"header newline", rtOptions{deletion: []rt.DeleteOption{rt.WithDeleteHeader("X-Value", "one\ntwo")}}, "Delete"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			c := rtClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
			a, s := rtScope(t, c)
			v, e := rtCall(a, s, context.Background(), test.op, test.o)
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(v, e, calls.Load())
			}
		})
	}
	t.Run("full create replacement factory and callback deep ownership", func(t *testing.T) {
		headers := map[string]string{"X-Owned": "before"}
		prefix, target := "before", "target"
		owned := rt.WithCreateOpts(rt.CreateOpts{Headers: headers, Prefix: &prefix, PropertiesTarget: &target})
		headers["X-Owned"] = "caller"
		prefix = "caller"
		target = "caller"
		var escapedHeaders map[string]string
		var escapedPrefix, escapedTarget *string
		var callbacks, calls atomic.Int32
		c := rtClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			raw, _ := io.ReadAll(r.Body)
			if string(raw) != `{"name":"OS::Compute::Server Extra","prefix":"before","properties_target":"target"}` || r.Header.Get("X-Owned") != "before" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != "live" {
				t.Error(string(raw), r.Header)
			}
			return rtWire(201, &rtBody{Reader: strings.NewReader(rtJSON)}), nil
		})
		c.MoreHeaders = map[string]string{"X-Source": "captured"}
		_, s := rtScope(t, c)
		v, e := s.Create(context.Background(), rtName, rt.WithCreatePrefix("discard"), owned, func(o *rt.CreateOpts) error {
			callbacks.Add(1)
			escapedHeaders = o.Headers
			escapedPrefix = o.Prefix
			escapedTarget = o.PropertiesTarget
			return nil
		}, func(*rt.CreateOpts) error {
			callbacks.Add(1)
			escapedHeaders["X-Owned"] = "escaped"
			*escapedPrefix = "escaped"
			*escapedTarget = "escaped"
			c.MoreHeaders["X-Source"] = "late"
			c.SetToken("live")
			return nil
		})
		if e != nil || v == nil || calls.Load() != 1 || callbacks.Load() != 2 || c.MoreHeaders["X-Owned"] != "" || c.MoreHeaders["X-Source"] != "late" {
			t.Fatal(v, e, calls.Load(), callbacks.Load(), c.MoreHeaders)
		}
	})
	t.Run("full create replacement restores omission", func(t *testing.T) {
		c := rtClient(func(r *http.Request) (*http.Response, error) {
			raw, _ := io.ReadAll(r.Body)
			if string(raw) != `{"name":"OS::Compute::Server Extra"}` || r.Header.Get("X-Old") != "" {
				t.Error(string(raw), r.Header)
			}
			return rtWire(201, &rtBody{Reader: strings.NewReader(rtJSON)}), nil
		})
		_, s := rtScope(t, c)
		v, e := s.Create(context.Background(), rtName, rt.WithCreatePrefix("discard"), rt.WithCreateHeader("X-Old", "discard"), rt.WithCreateOpts(rt.CreateOpts{}))
		if e != nil || v == nil {
			t.Fatal(v, e)
		}
	})
	t.Run("delete pointer snapshots retain explicit false", func(t *testing.T) {
		flag := false
		input := map[string]string{"X-Owned": "before"}
		owned := rt.WithDeleteOpts(rt.DeleteOpts{Headers: input, IgnoreMissing: &flag})
		flag = true
		input["X-Owned"] = "caller"
		var escaped *bool
		var hooks atomic.Int32
		cause := errors.New("native strict404 hook")
		c := rtClient(func(r *http.Request) (*http.Response, error) {
			if r.Header.Get("X-Owned") != "before" {
				t.Error(r.Header)
			}
			return rtWire(404, &rtBody{Reader: strings.NewReader("strict404")}), nil
		})
		c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
			hooks.Add(1)
			return cause
		}
		_, s := rtScope(t, c)
		v, e := s.Delete(context.Background(), rtName, rt.WithDeleteIgnoreMissing(true), owned, func(o *rt.DeleteOpts) error { escaped = o.IgnoreMissing; return nil }, func(*rt.DeleteOpts) error { *escaped = true; return nil })
		if v != nil || !errors.Is(e, cause) || !gophercloud.ResponseCodeIs(e, 404) || hooks.Load() != 1 {
			t.Fatal(v, e, hooks.Load())
		}
	})
	t.Run("shared ListOpts full replacement snapshots maps and cap", func(t *testing.T) {
		input := map[string]string{"X-Owned": "before"}
		owned := rt.WithListOpts(rt.ListOpts{Headers: input, MaxItems: 1})
		input["X-Owned"] = "caller"
		var escaped map[string]string
		var calls atomic.Int32
		c := rtClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Owned") != "before" || r.Header.Get("X-Discard") != "" || r.URL.RawQuery != "" {
				t.Error(r.Header, r.URL)
			}
			return rtWire(200, &rtBody{Reader: strings.NewReader(`{"resource_types":[{},null]}`)}), nil
		})
		a, _ := rtScope(t, c)
		rows, e := a.All(context.Background(), rt.WithListMaxItems(9), rt.WithListHeader("X-Discard", "old"), owned, func(o *rt.ListOpts) error { escaped = o.Headers; return nil }, func(*rt.ListOpts) error { escaped["X-Owned"] = "escaped"; return nil })
		if e != nil || len(rows) != 1 || calls.Load() != 1 {
			t.Fatal(rows, e, calls.Load())
		}
	})
}

func TestMetadefResourceTypesResponseOwnershipAndNativeHooks(t *testing.T) {
	for _, op := range rtOperations {
		for _, mode := range []string{"Read", "Close", "cancel", "Read Close cancel", "source after Close"} {
			t.Run(op.name+" accepted "+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("accepted Read"), errors.New("accepted Close"), errors.New("accepted cancel")
				raw := []byte(op.raw)
				if op.name == "Delete" {
					raw = []byte{'a', 'c', 'k', 255}
				}
				b := &rtBody{Reader: bytes.NewReader(raw)}
				if strings.Contains(mode, "Read") {
					b.Reader = rtReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
				}
				if strings.Contains(mode, "Close") {
					b.closeErr = closeCause
				}
				var calls, hooks atomic.Int32
				c := rtClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return rtWire(op.status, b), nil })
				a, s := rtScope(t, c)
				if mode == "source after Close" {
					b.closeErr = nil
					b.onClose = func() { c.ProviderClient = &gophercloud.ProviderClient{} }
				} else if strings.Contains(mode, "cancel") {
					b.onClose = func() { cancel(cancelCause) }
				}
				c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks.Add(1)
					return errors.New("accepted replay")
				}
				v, e := rtCall(a, s, ctx, op.name, rtOptions{})
				proof := rtProof(t, e, op.status, raw)
				if calls.Load() != 1 || hooks.Load() != 0 || b.closes.Load() != 1 || strings.Contains(mode, "Read") && !errors.Is(e, readCause) || mode != "source after Close" && strings.Contains(mode, "Close") && !errors.Is(e, closeCause) || strings.Contains(mode, "cancel") && (!errors.Is(e, cancelCause) || !errors.Is(e, context.Canceled)) || mode == "source after Close" && !errors.Is(e, resource.ErrInvalidOption) {
					t.Fatal(v, e, calls.Load(), hooks.Load(), b.closes.Load())
				}
				if op.name == "Delete" {
					if v == nil || v.ack == nil || v.ack.Namespace != rtParent || v.ack.Name == nil || *v.ack.Name != rtName || v.ack.StatusCode != 204 || !bytes.Equal(v.ack.Body, raw) {
						t.Fatal(v, e)
					}
					proof.Body[0] = '!'
					proof.Header.Set("X-Request-Id", "proof changed")
					if v.ack.Body[0] != 'a' || v.ack.Header.Get("X-Request-Id") != "actual-resource-type" {
						t.Fatal("ack aliases proof", v.ack)
					}
				} else if v != nil {
					t.Fatal("partial accepted typed result", v, e)
				}
			})
		}
	}
	t.Run("opaque204 copied acknowledgement", func(t *testing.T) {
		raw := []byte{255, 0, 1}
		b := &rtBody{Reader: bytes.NewReader(raw)}
		wire := rtWire(204, b)
		c := rtClient(func(r *http.Request) (*http.Response, error) {
			if r.Method != "DELETE" || r.Body != nil {
				t.Error(r)
			}
			return wire, nil
		})
		_, s := rtScope(t, c)
		v, e := s.Delete(context.Background(), rtName)
		if e != nil || v == nil || !bytes.Equal(v.Body, raw) || b.closes.Load() != 1 {
			t.Fatal(v, e, b.closes.Load())
		}
		wire.Header.Set("X-Request-Id", "late")
		raw[0] = 1
		if v.Header.Get("X-Request-Id") != "actual-resource-type" || v.Body[0] != 255 {
			t.Fatal(v)
		}
	})
	for _, op := range rtOperations {
		for _, code := range []int{202, 404, 409} {
			if op.name == "Delete" && code == 404 {
				continue
			}
			t.Run(fmt.Sprintf("%s strict%d", op.name, code), func(t *testing.T) {
				b := &rtBody{Reader: strings.NewReader("native rejection")}
				c := rtClient(func(*http.Request) (*http.Response, error) { return rtWire(code, b), nil })
				a, s := rtScope(t, c)
				v, e := rtCall(a, s, context.Background(), op.name, rtOptions{})
				var native gophercloud.ErrUnexpectedResponseCode
				expected := []int{op.status}
				if op.name == "Delete" {
					expected = append(expected, 404)
				}
				if v != nil || !errors.As(e, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, expected) || string(native.Body) != "native rejection" || native.ResponseHeader.Get("X-Request-Id") != "actual-resource-type" || b.closes.Load() != 1 {
					t.Fatal(v, e, native, b.closes.Load())
				}
			})
		}
	}
	for _, mode := range []string{"default", "explicit true", "explicit false"} {
		t.Run("physical404 "+mode, func(t *testing.T) {
			var calls, hooks atomic.Int32
			cause := errors.New("strict404 hook")
			b := &rtBody{Reader: strings.NewReader("actual404")}
			c := rtClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return rtWire(404, b), nil })
			c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				hooks.Add(1)
				return cause
			}
			_, s := rtScope(t, c)
			opts := []rt.DeleteOption{}
			if mode != "default" {
				opts = append(opts, rt.WithDeleteIgnoreMissing(mode == "explicit true"))
			}
			v, e := s.Delete(context.Background(), rtName, opts...)
			if v != nil || calls.Load() != 1 || b.closes.Load() != 1 {
				t.Fatal(v, e, calls.Load(), b.closes.Load())
			}
			if mode == "explicit false" {
				if !errors.Is(e, cause) || !gophercloud.ResponseCodeIs(e, 404) || hooks.Load() != 1 {
					t.Fatal(e, hooks.Load())
				}
			} else if e != nil || hooks.Load() != 0 {
				t.Fatal(e, hooks.Load())
			}
		})
	}
	for _, mode := range []string{"Read", "Close", "cancel", "Read Close cancel", "source"} {
		t.Run("owned404 "+mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("404 Read"), errors.New("404 Close"), errors.New("404 cause")
			raw := []byte("physical404 evidence")
			b := &rtBody{Reader: bytes.NewReader(raw)}
			if strings.Contains(mode, "Read") {
				b.Reader = rtReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
			}
			if strings.Contains(mode, "Close") {
				b.closeErr = closeCause
			}
			var calls, hooks atomic.Int32
			c := rtClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return rtWire(404, b), nil })
			_, s := rtScope(t, c)
			if strings.Contains(mode, "cancel") {
				b.onClose = func() { cancel(cancelCause) }
			}
			if mode == "source" {
				b.onClose = func() { c.Endpoint = "https://changed.invalid/v2/" }
			}
			c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				hooks.Add(1)
				return errors.New("404 replay")
			}
			v, e := s.Delete(ctx, rtName)
			rtProof(t, e, 404, raw)
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
			b := &rtBody{Reader: strings.NewReader("original failure")}
			c := rtClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				if mode == "transport nested404" {
					return nil, errors.Join(cause, nested)
				}
				code := 503
				if mode == "reauth nested404" {
					code = 401
				}
				return rtWire(code, b), nil
			})
			if mode == "callback nested404" {
				c.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks.Add(1)
					return errors.Join(cause, nested)
				}
			}
			if mode == "reauth nested404" {
				c.ReauthFunc = func(context.Context) error { hooks.Add(1); return errors.Join(cause, nested) }
			}
			_, s := rtScope(t, c)
			v, e := s.Delete(context.Background(), rtName)
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
	t.Run("configured prebody policy uses original provider live auth", func(t *testing.T) {
		var calls, reauth, backoff, retries atomic.Int32
		var bodies []*rtBody
		transportCause := errors.New("prebody transport")
		c := rtClient(nil)
		c.MoreHeaders = map[string]string{"X-Source": "captured"}
		c.ReauthFunc = func(context.Context) error { reauth.Add(1); c.SetToken("reauth"); return nil }
		c.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
			backoff.Add(1)
			c.SetToken("backoff")
			return nil
		}
		c.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, e error, _ uint) error {
			retries.Add(1)
			raw, _ := json.Marshal(o.JSONBody)
			if !o.KeepResponseBody || o.JSONResponse != nil || o.RawBody != nil || string(raw) != `{"name":"OS::Compute::Server Extra"}` {
				t.Error(o, string(raw))
			}
			if gophercloud.ResponseCodeIs(e, 503) {
				c.SetToken("retry503")
				return nil
			}
			if errors.Is(e, transportCause) {
				c.SetToken("retrytransport")
				return nil
			}
			return e
		}
		provider, hook := c.ProviderClient, reflect.ValueOf(c.RetryFunc).Pointer()
		_, s := rtScope(t, c)
		c.HTTPClient.Transport = rtTransport(func(r *http.Request) (*http.Response, error) {
			i := int(calls.Add(1)) - 1
			codes := []int{401, 429, 503, 0, 201}
			tokens := []string{"initial", "reauth", "backoff", "retry503", "retrytransport"}
			if i >= len(codes) {
				return nil, errors.New("replay")
			}
			raw, _ := io.ReadAll(r.Body)
			if r.Method != "POST" || r.URL.String() != rtBase+rtCollection || string(raw) != `{"name":"OS::Compute::Server Extra"}` || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != tokens[i] {
				t.Error(r.Method, r.URL, string(raw), r.Header)
			}
			if codes[i] == 0 {
				return nil, transportCause
			}
			b := &rtBody{Reader: strings.NewReader(rtJSON)}
			bodies = append(bodies, b)
			return rtWire(codes[i], b), nil
		})
		v, e := s.Create(context.Background(), rtName)
		if v == nil || e != nil || calls.Load() != 5 || reauth.Load() != 1 || backoff.Load() != 1 || retries.Load() != 2 || c.ProviderClient != provider || reflect.ValueOf(c.RetryFunc).Pointer() != hook {
			t.Fatal(v, e, calls.Load(), reauth.Load(), backoff.Load(), retries.Load())
		}
		for _, b := range bodies {
			if b.closes.Load() != 1 {
				t.Fatal(b.closes.Load())
			}
		}
	})
	for _, change := range []string{"in-place RawMessage", "changed JSON", "JSON nil", "JSON null", "KeepResponseBody", "JSONResponse", "RawBody", "unsupported JSONBody"} {
		t.Run("POST retry guard "+change, func(t *testing.T) {
			cause := errors.New("retry callback cause")
			var calls, hooks, borrowedReads atomic.Int32
			b := &rtBody{Reader: strings.NewReader("original503")}
			borrowed := &rtBody{Reader: rtReader(func([]byte) (int, error) { borrowedReads.Add(1); return 0, io.EOF })}
			c := rtClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				raw, _ := io.ReadAll(r.Body)
				if string(raw) != `{"name":"OS::Compute::Server Extra"}` {
					t.Error(string(raw))
				}
				return rtWire(503, b), nil
			})
			c.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, e error, _ uint) error {
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
						index := bytes.Index(raw, []byte("Server"))
						if index < 0 {
							t.Error(string(raw))
						} else {
							raw[index] = 'T'
						}
					}
				case "changed JSON":
					o.JSONBody = map[string]string{"name": "other"}
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
				return cause
			}
			_, s := rtScope(t, c)
			v, e := s.Create(context.Background(), rtName)
			var native gophercloud.ErrUnexpectedResponseCode
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) || !errors.Is(e, cause) || !errors.As(e, &native) || native.Actual != 503 || string(native.Body) != "original503" || calls.Load() != 1 || hooks.Load() != 1 || b.closes.Load() != 1 || borrowedReads.Load() != 0 || borrowed.closes.Load() != 0 {
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
	t.Run("bodyless DELETE rejects explicit JSON null", func(t *testing.T) {
		var calls atomic.Int32
		b := &rtBody{Reader: strings.NewReader("original503")}
		c := rtClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Body != nil {
				t.Error(r.Body)
			}
			return rtWire(503, b), nil
		})
		c.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			o.JSONBody = json.RawMessage("null")
			return nil
		}
		_, s := rtScope(t, c)
		v, e := s.Delete(context.Background(), rtName)
		if v != nil || !errors.Is(e, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(e, 503) || calls.Load() != 1 || b.closes.Load() != 1 {
			t.Fatal(v, e, calls.Load(), b.closes.Load())
		}
	})
	t.Run("same serialized replacement native header mutation", func(t *testing.T) {
		var calls, hooks atomic.Int32
		var bodies []*rtBody
		input := map[string]string{"X-Source": "original"}
		c := rtClient(func(r *http.Request) (*http.Response, error) {
			n := calls.Add(1)
			raw, _ := io.ReadAll(r.Body)
			if string(raw) != `{"name":"OS::Compute::Server Extra"}` || n == 1 && r.Header.Get("X-Source") != "original" || n == 2 && r.Header.Get("X-Native") != "allowed" {
				t.Error(string(raw), r.Header)
			}
			code, reply := 503, "original503"
			if n == 2 {
				code, reply = 201, rtJSON
			}
			b := &rtBody{Reader: strings.NewReader(reply)}
			bodies = append(bodies, b)
			return rtWire(code, b), nil
		})
		c.MoreHeaders = input
		c.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			hooks.Add(1)
			o.JSONBody = map[string]string{"name": rtName}
			o.MoreHeaders = map[string]string{"X-Native": "allowed"}
			return nil
		}
		provider, hook := c.ProviderClient, reflect.ValueOf(c.RetryFunc).Pointer()
		_, s := rtScope(t, c)
		v, e := s.Create(context.Background(), rtName)
		if v == nil || e != nil || calls.Load() != 2 || hooks.Load() != 1 || c.ProviderClient != provider || reflect.ValueOf(c.RetryFunc).Pointer() != hook || c.MoreHeaders["X-Source"] != "original" || c.MoreHeaders["X-Native"] != "" {
			t.Fatal(v, e, calls.Load(), hooks.Load(), c.MoreHeaders)
		}
		for _, b := range bodies {
			if b.closes.Load() != 1 {
				t.Fatal(b.closes.Load())
			}
		}
	})
	for _, op := range rtOperations {
		t.Run(op.name+" expanded codes actual proof", func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("unexpected Read"), errors.New("unexpected Close"), errors.New("unexpected cause")
			var calls, hooks atomic.Int32
			var bodies []*rtBody
			c := rtClient(func(*http.Request) (*http.Response, error) {
				n := calls.Add(1)
				code, raw := 503, "original503"
				if n == 2 {
					code, raw = 202, "unexpected actual202"
				}
				b := &rtBody{Reader: strings.NewReader(raw)}
				if n == 2 {
					b.Reader = rtReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
					b.closeErr = closeCause
					b.onClose = func() { cancel(cancelCause) }
				}
				bodies = append(bodies, b)
				return rtWire(code, b), nil
			})
			c.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, e error, _ uint) error {
				hooks.Add(1)
				if !gophercloud.ResponseCodeIs(e, 503) {
					return e
				}
				o.OkCodes = []int{202}
				return nil
			}
			a, s := rtScope(t, c)
			v, e := rtCall(a, s, ctx, op.name, rtOptions{})
			var native gophercloud.ErrUnexpectedResponseCode
			var proof *resource.ResponseError
			expected := []int{op.status}
			if op.name == "Delete" {
				expected = append(expected, 404)
			}
			if v != nil || !errors.As(e, &native) || errors.As(e, &proof) || native.Actual != 202 || native.Method != op.method || native.URL != rtTarget(op.name) || !reflect.DeepEqual(native.Expected, expected) || string(native.Body) != "unexpected actual202" || native.ResponseHeader.Get("X-Request-Id") != "actual-resource-type" || !errors.Is(e, readCause) || !errors.Is(e, closeCause) || !errors.Is(e, cancelCause) || !errors.Is(e, context.Canceled) || calls.Load() != 2 || hooks.Load() != 1 {
				t.Fatal(v, e, native, calls.Load(), hooks.Load())
			}
			for _, b := range bodies {
				if b.closes.Load() != 1 {
					t.Fatal(b.closes.Load())
				}
			}
		})
	}
	for _, mode := range []string{"same target", "path changed", "method changed"} {
		t.Run("native redirect "+mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			c := cloud.Client("image", rtPrefix)
			var first, follow atomic.Int32
			path := rtPrefix + rtCollection
			cloud.Mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				if first.Add(1) == 1 {
					target := path
					code := 307
					if mode == "path changed" {
						target = rtPrefix + "different"
					}
					if mode == "method changed" {
						code = 303
					}
					http.Redirect(w, r, target, code)
					return
				}
				follow.Add(1)
				if r.Method != "POST" {
					t.Error(r.Method)
				}
				testcloud.JSON(w, 201, rtJSON)
			})
			cloud.Mux.HandleFunc(rtPrefix+"different", func(w http.ResponseWriter, r *http.Request) { follow.Add(1); testcloud.JSON(w, 201, rtJSON) })
			_, s := rtScope(t, c)
			v, e := s.Create(context.Background(), rtName)
			if mode == "same target" {
				if e != nil || v == nil || first.Load() != 2 || follow.Load() != 1 {
					t.Fatal(v, e, first.Load(), follow.Load())
				}
			} else if v != nil || !errors.Is(e, resource.ErrInvalidOption) || first.Load() != 1 || follow.Load() != 0 {
				t.Fatal(v, e, first.Load(), follow.Load())
			}
		})
	}
}
