package metadefproperties_test

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
	properties "gophercloudsdk/image/v2/metadefproperties"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

const propertyPrefix = "/reverse/property/glance/v2/"
const propertyBase = "https://glance.invalid" + propertyPrefix
const propertyParent = "OS::Compute::Libvirt"
const propertyName = "CPU Limits"
const propertyCollection = "metadefs/namespaces/" + propertyParent + "/properties"
const propertyPayload = `{"name":"CPU Limits","title":"","type":"number"}`
const propertyJSON = `{"name":"foreign response name","type":"number","title":"server title","description":"server description","minimum":9007199254740993,"maximum":1e400,"default":null,"readonly":false,"items":{"type":"string"},"created_at":"literal-date","self":"https://passive.invalid/entity","schema":"https://passive.invalid/schema"}`
const propertyListJSON = `{"properties":{"quota:cpu":` + propertyJSON + `},"next":42,"schema":false}`

type propertyTransport func(*http.Request) (*http.Response, error)

func (f propertyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	v, e := f(r)
	if v != nil && v.Request == nil {
		v.Request = r
	}
	return v, e
}

type propertyBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *propertyBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type propertyReader func([]byte) (int, error)

func (f propertyReader) Read(p []byte) (int, error) { return f(p) }
func propertyWire(status int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"actual-property"}}, Body: body}
}
func propertyClient(f propertyTransport) *gophercloud.ServiceClient {
	p := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: f}}
	p.UseTokenLock()
	p.SetToken("initial")
	return &gophercloud.ServiceClient{ProviderClient: p, Type: "image", Endpoint: propertyBase}
}
func propertyScope(t *testing.T, client *gophercloud.ServiceClient, parent string) *properties.NamespaceScope {
	t.Helper()
	api := properties.New(client)
	scope, e := api.InNamespace(context.Background(), parent)
	if e != nil || scope == nil || api.RawClient() != client || scope.RawClient() != client || scope.NamespaceName() != parent {
		t.Fatalf("scope %v %v", scope, e)
	}
	return scope
}

type propertyOptions struct {
	create    []properties.CreateOption
	get       []properties.GetOption
	update    []properties.UpdateOption
	deletion  []properties.DeleteOption
	deleteAll []properties.DeleteAllOption
	list      []properties.ListOption
}
type propertyResult struct {
	value *properties.Property
	ack   *properties.Acknowledgement
	rows  []*properties.Property
}

func propertyCall(scope *properties.NamespaceScope, ctx context.Context, op, name string, opts propertyOptions) (*propertyResult, error) {
	var value *properties.Property
	var e error
	switch op {
	case "Create":
		value, e = scope.Create(ctx, name, append([]properties.CreateOption{properties.WithCreateType("number"), properties.WithCreateTitle("")}, opts.create...)...)
	case "Get":
		value, e = scope.Get(ctx, name, opts.get...)
	case "Update":
		value, e = scope.Update(ctx, name, append([]properties.UpdateOption{properties.WithUpdateType("number"), properties.WithUpdateTitle("")}, opts.update...)...)
	case "Delete":
		v, e := scope.Delete(ctx, name, opts.deletion...)
		if v == nil {
			return nil, e
		}
		return &propertyResult{ack: v}, e
	case "DeleteAll":
		v, e := scope.DeleteAll(ctx, opts.deleteAll...)
		if v == nil {
			return nil, e
		}
		return &propertyResult{ack: v}, e
	case "All":
		v, e := scope.All(ctx, opts.list...)
		if v == nil {
			return nil, e
		}
		return &propertyResult{rows: v}, e
	case "List":
		rows := make([]*properties.Property, 0)
		for v, e := range scope.List(ctx, opts.list...) {
			if e != nil {
				return nil, e
			}
			rows = append(rows, v)
		}
		return &propertyResult{rows: rows}, nil
	default:
		panic("unknown operation")
	}
	if value == nil {
		return nil, e
	}
	return &propertyResult{value: value}, e
}

var propertyOperations = []struct {
	name, method, body, raw string
	status                  int
	named                   bool
}{
	{"Create", "POST", propertyPayload, propertyJSON, 201, false},
	{"Get", "GET", "", propertyJSON, 200, true},
	{"Update", "PUT", propertyPayload, propertyJSON, 200, true},
	{"Delete", "DELETE", "", "", 204, true},
	{"DeleteAll", "DELETE", "", "", 204, false},
	{"List", "GET", "", propertyListJSON, 200, false},
	{"All", "GET", "", propertyListJSON, 200, false},
}

func propertyProof(t *testing.T, e error, status int, raw []byte) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(e, &proof) || proof.StatusCode != status || !bytes.Equal(proof.Body, raw) || proof.Header.Get("X-Request-Id") != "actual-property" {
		t.Fatalf("owned proof %v %+v", e, proof)
	}
	return proof
}
func propertyHeaders() propertyOptions {
	return propertyOptions{create: []properties.CreateOption{properties.WithCreateHeaders(map[string]string{"X-Option": "owned"}), properties.WithCreateHeader("X-Final", "yes")}, get: []properties.GetOption{properties.WithGetHeaders(map[string]string{"X-Option": "owned"}), properties.WithGetHeader("X-Final", "yes")}, update: []properties.UpdateOption{properties.WithUpdateHeaders(map[string]string{"X-Option": "owned"}), properties.WithUpdateHeader("X-Final", "yes")}, deletion: []properties.DeleteOption{properties.WithDeleteHeaders(map[string]string{"X-Option": "owned"}), properties.WithDeleteHeader("X-Final", "yes")}, deleteAll: []properties.DeleteAllOption{properties.WithDeleteAllHeaders(map[string]string{"X-Option": "owned"}), properties.WithDeleteAllHeader("X-Final", "yes")}, list: []properties.ListOption{properties.WithListHeaders(map[string]string{"X-Option": "owned"}), properties.WithListHeader("X-Final", "yes")}}
}

func TestMetadefPropertiesFixedScopedRoutesAndPayloads(t *testing.T) {
	for _, op := range propertyOperations {
		t.Run(op.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", "/unused/")
			client.ResourceBase = cloud.Server.URL + propertyPrefix
			client.Microversion = "2.2"
			var calls atomic.Int32
			scope := propertyScope(t, client, propertyParent)
			if calls.Load() != 0 {
				t.Fatal("scope HTTP")
			}
			client.MoreHeaders = map[string]string{"X-Source": "latest"}
			cloud.Provider.SetToken("live")
			cloud.Mux.HandleFunc(propertyPrefix, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				path := propertyPrefix + propertyCollection
				if op.named {
					path += "/" + url.PathEscape(propertyName)
				}
				raw, _ := io.ReadAll(r.Body)
				if r.Method != op.method || r.URL.EscapedPath() != path || r.URL.RawQuery != "" || string(raw) != op.body || r.Header.Get("X-Auth-Token") != "live" || r.Header.Get("X-Source") != "latest" || r.Header.Get("X-Option") != "owned" || r.Header.Get("X-Final") != "yes" || r.Header.Get("OpenStack-API-Version") != "image 2.2" {
					t.Error(r.Method, r.RequestURI, string(raw), r.Header)
				}
				w.Header().Set("X-Request-Id", "actual-property")
				w.Header().Set("Location", "https://passive.invalid/other")
				w.WriteHeader(op.status)
				io.WriteString(w, op.raw)
			})
			v, e := propertyCall(scope, context.Background(), op.name, propertyName, propertyHeaders())
			if v == nil || e != nil || calls.Load() != 1 {
				t.Fatal(v, e, calls.Load())
			}
			if v.value != nil && (v.value.Key != nil || v.value.StatusCode != op.status || v.value.Header.Get("X-Request-Id") != "actual-property" || v.value.Name == nil || *v.value.Name != "foreign response name") {
				t.Fatal(v.value)
			}
			if v.ack != nil {
				if v.ack.Namespace != propertyParent || v.ack.StatusCode != 204 || (op.name == "Delete" && (v.ack.Name == nil || *v.ack.Name != propertyName)) || (op.name == "DeleteAll" && v.ack.Name != nil) {
					t.Fatal(v.ack)
				}
			}
			if v.rows != nil && (len(v.rows) != 1 || v.rows[0].Key == nil || *v.rows[0].Key != "quota:cpu" || *v.rows[0].Name != "foreign response name") {
				t.Fatal(v.rows)
			}
		})
	}
	for _, parent := range []string{"OS::Compute::Libvirt", strings.Repeat("한", 80), "literal parent"} {
		for _, name := range []string{"quota:cpu", strings.Repeat("界", 80), " literal name "} {
			for _, op := range propertyOperations {
				t.Run(op.name+" literal "+parent+" "+name, func(t *testing.T) {
					var calls atomic.Int32
					client := propertyClient(func(r *http.Request) (*http.Response, error) {
						calls.Add(1)
						want := propertyPrefix + "metadefs/namespaces/" + url.PathEscape(parent) + "/properties"
						if op.named {
							want += "/" + url.PathEscape(name)
						}
						if r.URL.EscapedPath() != want || r.URL.RawQuery != "" {
							t.Error(r.URL)
						}
						if r.Body != nil {
							raw, _ := io.ReadAll(r.Body)
							var fields map[string]json.RawMessage
							if e := json.Unmarshal(raw, &fields); e != nil || string(fields["name"]) != fmt.Sprintf("%q", name) {
								t.Error(string(raw), e)
							}
						}
						return propertyWire(op.status, &propertyBody{Reader: strings.NewReader(op.raw)}), nil
					})
					v, e := propertyCall(propertyScope(t, client, parent), context.Background(), op.name, name, propertyOptions{})
					if v == nil || e != nil || calls.Load() != 1 {
						t.Fatal(v, e, calls.Load())
					}
				})
			}
		}
	}
	for _, op := range []string{"Create", "Update"} {
		for _, mode := range []string{"minimal", "literal definition", "any snapshot"} {
			t.Run(op+" "+mode, func(t *testing.T) {
				opts := propertyOptions{}
				expected := map[string]any{"name": propertyName, "type": "number", "title": ""}
				desc := strings.Repeat("界", 600) + "\nmultiline"
				newName := "renamed: name"
				if mode == "literal definition" {
					attributes := map[string]json.RawMessage{"minimum": json.RawMessage("9007199254740993"), "maximum": json.RawMessage("1e400"), "default": json.RawMessage("null"), "readonly": json.RawMessage("false"), "enum": json.RawMessage("[]"), "items": json.RawMessage("{}"), "future": json.RawMessage(`{"nested":true}`)}
					opts.create = []properties.CreateOption{properties.WithCreateOpts(properties.CreateOpts{Type: propertyString("future-type"), Title: propertyString(""), Description: &desc, Attributes: attributes})}
					opts.update = []properties.UpdateOption{properties.WithUpdateOpts(properties.UpdateOpts{Name: &newName, Type: propertyString("future-type"), Title: propertyString(""), Description: &desc, Attributes: attributes})}
					expected["type"] = "future-type"
					expected["description"] = desc
					for k, v := range attributes {
						expected[k] = v
					}
					if op == "Update" {
						expected["name"] = newName
					}
				}
				if mode == "any snapshot" {
					nested := map[string]any{"amount": json.Number("9007199254740993")}
					data := map[string]any{"default": nil, "readonly": false, "minItems": 0, "enum": []any{}, "items": nested}
					single := json.RawMessage(`{"x":1}`)
					opts.create = []properties.CreateOption{properties.WithCreateAttributes(data), properties.WithCreateAttribute("future", single), properties.WithCreateDescription(desc)}
					opts.update = []properties.UpdateOption{properties.WithUpdateAttributes(data), properties.WithUpdateAttribute("future", single), properties.WithUpdateDescription(desc)}
					expected["description"] = desc
					expected["default"] = nil
					expected["readonly"] = false
					expected["minItems"] = 0
					expected["enum"] = []any{}
					expected["items"] = json.RawMessage(`{"amount":9007199254740993}`)
					expected["future"] = json.RawMessage(`{"x":1}`)
					nested["amount"] = 99
					data["readonly"] = true
					single[len(single)-2] = '2'
				}
				want, _ := json.Marshal(expected)
				var calls atomic.Int32
				code := 201
				if op == "Update" {
					code = 200
				}
				client := propertyClient(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					raw, _ := io.ReadAll(r.Body)
					target := propertyBase + propertyCollection
					if op == "Update" {
						target += "/" + url.PathEscape(propertyName)
					}
					if r.URL.String() != target || !bytes.Equal(raw, want) {
						t.Error(r.URL, string(raw), string(want))
					}
					return propertyWire(code, &propertyBody{Reader: strings.NewReader(propertyJSON)}), nil
				})
				v, e := propertyCall(propertyScope(t, client, propertyParent), context.Background(), op, propertyName, opts)
				if v == nil || e != nil || calls.Load() != 1 {
					t.Fatal(v, e, calls.Load())
				}
			})
		}
	}
	for _, query := range []*string{nil, propertyString(""), propertyString("OS::Compute + &?/资源")} {
		t.Run("resource_type "+fmt.Sprint(query), func(t *testing.T) {
			var calls atomic.Int32
			client := propertyClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				expected := ""
				if query != nil {
					expected = url.Values{"resource_type": {*query}}.Encode()
				}
				if r.Method != "GET" || r.URL.RawQuery != expected || r.URL.EscapedPath() != propertyPrefix+propertyCollection+"/"+url.PathEscape("prefix:quota") || r.Body != nil {
					t.Error(r.Method, r.URL, r.Body)
				}
				return propertyWire(200, &propertyBody{Reader: strings.NewReader(propertyJSON)}), nil
			})
			opts := []properties.GetOption{properties.WithGetOpts(properties.GetOpts{ResourceType: query})}
			if query != nil {
				opts = append(opts, properties.WithGetResourceType(*query))
			}
			v, e := propertyScope(t, client, propertyParent).Get(context.Background(), "prefix:quota", opts...)
			if v == nil || e != nil || v.Key != nil || calls.Load() != 1 {
				t.Fatal(v, e, calls.Load())
			}
		})
	}
	t.Run("scope context not retained", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		client := propertyClient(func(*http.Request) (*http.Response, error) {
			return propertyWire(200, &propertyBody{Reader: strings.NewReader(propertyJSON)}), nil
		})
		scope, e := properties.New(client).InNamespace(ctx, propertyParent)
		cancel()
		if e != nil {
			t.Fatal(e)
		}
		v, e := scope.Get(context.Background(), propertyName)
		if v == nil || e != nil {
			t.Fatal(v, e)
		}
	})
}
func propertyString(s string) *string { return &s }

func TestMetadefPropertiesCanonicalModelsAndRawOwnership(t *testing.T) {
	for _, raw := range []string{`{}`, `{"name":null,"type":null,"title":null,"description":null,"self":null,"schema":null,"created_at":null,"updated_at":null}`, `{"name":"","type":"","title":"","description":"","self":"","schema":"","created_at":"","updated_at":""}`, `{"name":"foreign","type":"future","title":"","description":"line\ntext","minimum":9007199254740993,"maximum":1e400,"default":null,"readonly":false,"enum":[null,1,true,{}],"items":false,"required":[null],"Name":42,"id":99,"namespace":"other","links":false,"created_at":"literal"}`} {
		t.Run(raw, func(t *testing.T) {
			b := &propertyBody{Reader: strings.NewReader(raw)}
			wire := propertyWire(200, b)
			client := propertyClient(func(*http.Request) (*http.Response, error) { return wire, nil })
			v, e := propertyScope(t, client, propertyParent).Get(context.Background(), propertyName)
			if v == nil || e != nil || v.Key != nil || v.Links != nil || v.StatusCode != 200 || b.closes.Load() != 1 {
				t.Fatal(v, e)
			}
			var fields map[string]json.RawMessage
			json.Unmarshal([]byte(raw), &fields)
			if !reflect.DeepEqual(v.Body, fields) {
				t.Fatal(v.Body, fields)
			}
			for key, ptr := range map[string]*string{"name": v.Name, "type": v.Type, "title": v.Title, "description": v.Description, "self": v.Self, "schema": v.Schema, "created_at": v.CreatedAt, "updated_at": v.UpdatedAt} {
				token, present := fields[key]
				if !present || string(token) == "null" {
					if ptr != nil {
						t.Fatal(key, ptr)
					}
				} else {
					var expected string
					json.Unmarshal(token, &expected)
					if ptr == nil || *ptr != expected {
						t.Fatal(key, ptr, expected)
					}
				}
			}
			v.Header.Set("X-Request-Id", "mutated")
			if wire.Header.Get("X-Request-Id") != "actual-property" {
				t.Fatal(wire.Header)
			}
			if v.Name != nil {
				*v.Name = "changed"
				if string(v.Body["name"]) != string(fields["name"]) {
					t.Fatal(v.Body)
				}
			}
			if precision, ok := v.Body["minimum"]; ok {
				if string(precision) != "9007199254740993" || string(v.Body["maximum"]) != "1e400" {
					t.Fatal(v.Body)
				}
				precision[0] = '1'
				if fields["minimum"][0] != '9' || !strings.Contains(raw, "9007199254740993") {
					t.Fatal("raw alias")
				}
			}
		})
	}
	for _, field := range []string{"name", "type", "title", "description", "self", "schema", "created_at", "updated_at"} {
		for _, token := range []string{"1", "true", "[]", "{}"} {
			t.Run(field+token, func(t *testing.T) {
				raw := []byte(`{"` + field + `":` + token + `}`)
				b := &propertyBody{Reader: bytes.NewReader(raw)}
				client := propertyClient(func(*http.Request) (*http.Response, error) { return propertyWire(200, b), nil })
				v, e := propertyScope(t, client, propertyParent).Get(context.Background(), propertyName)
				if v != nil || e == nil || b.closes.Load() != 1 {
					t.Fatal(v, e)
				}
				propertyProof(t, e, 200, raw)
			})
		}
	}
	for _, raw := range [][]byte{[]byte("null"), []byte("[]"), []byte("42"), []byte(`{"name":`), {'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}} {
		t.Run("invalid201 "+string(raw), func(t *testing.T) {
			b := &propertyBody{Reader: bytes.NewReader(raw)}
			client := propertyClient(func(*http.Request) (*http.Response, error) { return propertyWire(201, b), nil })
			v, e := propertyScope(t, client, propertyParent).Create(context.Background(), propertyName, properties.WithCreateType("number"), properties.WithCreateTitle(""))
			if v != nil || e == nil || b.closes.Load() != 1 {
				t.Fatal(v, e)
			}
			propertyProof(t, e, 201, raw)
		})
	}
	t.Run("key and canonical name independently owned", func(t *testing.T) {
		raw := `{"properties":{"z first":{"name":"passive other","minimum":9007199254740993},"a second":{},"late":null}}`
		b := &propertyBody{Reader: strings.NewReader(raw)}
		wire := propertyWire(200, b)
		client := propertyClient(func(*http.Request) (*http.Response, error) { return wire, nil })
		scope := propertyScope(t, client, propertyParent)
		rows, e := scope.All(context.Background(), properties.WithListMaxItems(2))
		if e != nil || len(rows) != 2 || *rows[0].Key != "z first" || *rows[0].Name != "passive other" || *rows[1].Key != "a second" || rows[1].Name != nil || len(rows[1].Body) != 0 {
			t.Fatal(rows, e)
		}
		*rows[0].Key = "changed"
		*rows[0].Name = "changed"
		rows[0].Body["minimum"][0] = '1'
		rows[0].Header.Set("X-Request-Id", "changed")
		if *rows[1].Key != "a second" || rows[1].Header.Get("X-Request-Id") != "actual-property" || wire.Header.Get("X-Request-Id") != "actual-property" || string(rows[0].Body["name"]) != `"passive other"` {
			t.Fatal(rows, wire.Header)
		}
		encoded, _ := json.Marshal(rows[0])
		var output map[string]json.RawMessage
		json.Unmarshal(encoded, &output)
		if _, ok := output["Key"]; ok {
			t.Fatal(string(encoded))
		}
		if _, ok := output["key"]; ok {
			t.Fatal(string(encoded))
		}
		wire.Body = &propertyBody{Reader: strings.NewReader(raw)}
		rows, e = scope.All(context.Background())
		if rows != nil || e == nil {
			t.Fatal(rows, e)
		}
		propertyProof(t, e, 200, []byte(raw))
	})
}

func TestMetadefPropertiesFiniteDictionaryAndLocalControls(t *testing.T) {
	for _, cap := range []int{0, 1, 3, 20} {
		t.Run(fmt.Sprint("cap", cap), func(t *testing.T) {
			raw := `{"properties":{"z":{"title":"first"},"a":{"title":"second"}},"next":"https://foreign.invalid/x","first":42}`
			var calls atomic.Int32
			client := propertyClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.URL.RawQuery != "" || r.Body != nil || r.URL.String() != propertyBase+propertyCollection {
					t.Error(r.URL, r.Body)
				}
				wire := propertyWire(200, &propertyBody{Reader: strings.NewReader(raw)})
				wire.Header.Set("Link", `<https://foreign.invalid/b>; rel="next"`)
				return wire, nil
			})
			rows, e := propertyScope(t, client, propertyParent).All(context.Background(), properties.WithListMaxItems(cap))
			expected := 2
			if cap == 1 {
				expected = 1
			}
			if e != nil || len(rows) != expected || *rows[0].Key != "z" || calls.Load() != 1 {
				t.Fatal(rows, e, calls.Load())
			}
			if expected == 2 && *rows[1].Key != "a" {
				t.Fatal(rows)
			}
		})
	}
	for _, raw := range []string{`{"properties":{}}`, `{"properties":{},"next":false,"first":[],"schema":null}`, `{"properties":{"z":null,"a":{},"z":{"title":"last"}}}`, `{"properties":{"z":{},"a":null,"z":null}}`} {
		t.Run("ordered "+raw, func(t *testing.T) {
			var calls atomic.Int32
			client := propertyClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				wire := propertyWire(200, &propertyBody{Reader: strings.NewReader(raw)})
				wire.Header.Set("Link", "broken")
				return wire, nil
			})
			rows, e := propertyScope(t, client, propertyParent).All(context.Background())
			if strings.Contains(raw, `"z":null}}`) {
				if rows != nil || e == nil {
					t.Fatal(rows, e)
				}
				propertyProof(t, e, 200, []byte(raw))
			} else if strings.Contains(raw, `"z":null,`) {
				if e != nil || len(rows) != 2 || *rows[0].Key != "z" || *rows[0].Title != "last" || *rows[1].Key != "a" {
					t.Fatal(rows, e)
				}
			} else if e != nil || rows == nil || len(rows) != 0 {
				t.Fatal(rows, e)
			}
			if calls.Load() != 1 {
				t.Fatal(calls.Load())
			}
		})
	}
	for _, mode := range []string{"cap", "break", "all"} {
		t.Run(mode, func(t *testing.T) {
			raw := `{"properties":{"first":{},"unused":null}}`
			b := &propertyBody{Reader: strings.NewReader(raw)}
			client := propertyClient(func(*http.Request) (*http.Response, error) { return propertyWire(200, b), nil })
			scope := propertyScope(t, client, propertyParent)
			if mode == "break" {
				seen := 0
				for row, e := range scope.List(context.Background()) {
					if e != nil || row == nil {
						t.Fatal(row, e)
					}
					seen++
					break
				}
				if seen != 1 {
					t.Fatal(seen)
				}
			} else {
				opts := []properties.ListOption{}
				if mode == "cap" {
					opts = append(opts, properties.WithListMaxItems(1))
				}
				rows, e := scope.All(context.Background(), opts...)
				if mode == "all" {
					if rows != nil || e == nil {
						t.Fatal(rows, e)
					}
					propertyProof(t, e, 200, []byte(raw))
				} else if e != nil || len(rows) != 1 {
					t.Fatal(rows, e)
				}
			}
			if b.closes.Load() != 1 {
				t.Fatal(b.closes.Load())
			}
		})
	}
	for _, raw := range []string{`null`, `[]`, `{}`, `{"properties":null}`, `{"properties":[]}`, `{"properties":true}`, `{"properties":{"one":[]}}`, `{"properties":{"one":1}}`, `{"properties":{"one":{}}} trailing`, "{\"properties\":{\"x\":{\"title\":\"" + string([]byte{0xff}) + "\"}}}"} {
		t.Run("strict "+raw, func(t *testing.T) {
			b := &propertyBody{Reader: strings.NewReader(raw)}
			client := propertyClient(func(*http.Request) (*http.Response, error) { return propertyWire(200, b), nil })
			rows, e := propertyScope(t, client, propertyParent).All(context.Background(), properties.WithListMaxItems(1))
			if rows != nil || e == nil || b.closes.Load() != 1 {
				t.Fatal(rows, e)
			}
			propertyProof(t, e, 200, []byte(raw))
		})
	}
	t.Run("parallel lazy iterator snapshots", func(t *testing.T) {
		var calls, callbacks atomic.Int32
		client := propertyClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Snapshot") != "owned" || r.URL.RawQuery != "" {
				t.Error(r.Header, r.URL)
			}
			return propertyWire(200, &propertyBody{Reader: strings.NewReader(`{"properties":{"first":{},"unused":null}}`)}), nil
		})
		headers := map[string]string{"X-Snapshot": "owned"}
		opts := []properties.ListOption{properties.WithListHeaders(headers), properties.WithListMaxItems(1), func(*properties.ListOpts) error { callbacks.Add(1); return nil }}
		seq := propertyScope(t, client, propertyParent).List(context.Background(), opts...)
		headers["X-Snapshot"] = "mutated"
		opts[0] = nil
		if calls.Load() != 0 || callbacks.Load() != 0 {
			t.Fatal("eager")
		}
		var wg sync.WaitGroup
		for range 4 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				seen := 0
				for row, e := range seq {
					if row == nil || e != nil || *row.Key != "first" {
						t.Error(row, e)
					}
					seen++
				}
				if seen != 1 {
					t.Error(seen)
				}
			}()
		}
		wg.Wait()
		if calls.Load() != 4 || callbacks.Load() != 4 {
			t.Fatal(calls.Load(), callbacks.Load())
		}
	})
	for _, mode := range []string{"context", "provider", "target"} {
		t.Run("between rows "+mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("between rows")
			raw := `{"properties":{"first":{},"second":{}}}`
			client := propertyClient(func(*http.Request) (*http.Response, error) {
				return propertyWire(200, &propertyBody{Reader: strings.NewReader(raw)}), nil
			})
			scope := propertyScope(t, client, propertyParent)
			seen, fail := 0, 0
			for row, e := range scope.List(ctx) {
				if e != nil {
					fail++
					if row != nil {
						t.Fatal(row)
					}
					propertyProof(t, e, 200, []byte(raw))
					if mode == "context" {
						if !errors.Is(e, context.Canceled) || !errors.Is(e, cause) {
							t.Fatal(e)
						}
					} else if !errors.Is(e, resource.ErrInvalidOption) {
						t.Fatal(e)
					}
					continue
				}
				seen++
				switch mode {
				case "context":
					cancel(cause)
				case "provider":
					client.ProviderClient = &gophercloud.ProviderClient{}
				case "target":
					client.Endpoint = propertyBase + "other/"
				}
			}
			if seen != 1 || fail != 1 {
				t.Fatal(seen, fail)
			}
		})
	}
}

type propertyMarshal struct {
	count *atomic.Int32
	cause error
}

func (m propertyMarshal) MarshalJSON() ([]byte, error) {
	m.count.Add(1)
	if m.cause != nil {
		return nil, m.cause
	}
	return []byte(`{"at_factory":1}`), nil
}
func TestMetadefPropertiesOwnedPreparationAndSource(t *testing.T) {
	for _, mode := range []string{"nil context", "cancel", "nil API", "nil client", "nil provider", "wrong type", "bad base", "protected source"} {
		t.Run("scope "+mode, func(t *testing.T) {
			var calls atomic.Int32
			client := propertyClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpectedHTTP") })
			api := properties.New(client)
			ctx := context.Background()
			expected := resource.ErrInvalidOption
			switch mode {
			case "nil context":
				ctx = nil
			case "cancel":
				c, cancel := context.WithCancel(ctx)
				cancel()
				ctx = c
				expected = context.Canceled
			case "nil API":
				api = nil
			case "nil client":
				api = properties.New(nil)
			case "nil provider":
				client.ProviderClient = nil
			case "wrong type":
				client.Type = "compute"
				expected = resource.ErrUnsupported
			case "bad base":
				client.ResourceBase = "relative"
			case "protected source":
				client.MoreHeaders = map[string]string{"X-Auth-Token": "override"}
			}
			scope, e := api.InNamespace(ctx, propertyParent)
			if scope != nil || !errors.Is(e, expected) || calls.Load() != 0 {
				t.Fatal(scope, e, calls.Load())
			}
		})
	}
	unsafe := []string{"", ".", "..", "a/b", "a\\b", "%2F", "x?y", "x#y", "line\nname", string([]byte{127}), string([]byte{0xff}), strings.Repeat("界", 81)}
	for _, parent := range unsafe {
		t.Run("parent "+parent, func(t *testing.T) {
			var calls atomic.Int32
			client := propertyClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpectedHTTP") })
			scope, e := properties.New(client).InNamespace(context.Background(), parent)
			if scope != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(scope, e, calls.Load())
			}
		})
	}
	for _, name := range unsafe {
		for _, op := range []string{"Create", "Get", "Update", "Delete"} {
			t.Run(op+" unsafechild "+name, func(t *testing.T) {
				var calls, callbacks atomic.Int32
				client := propertyClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpectedHTTP") })
				opts := propertyCallbacks(func() { callbacks.Add(1) }, nil)
				v, e := propertyCall(propertyScope(t, client, propertyParent), context.Background(), op, name, opts)
				if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 || callbacks.Load() != 0 {
					t.Fatal(v, e, calls.Load(), callbacks.Load())
				}
			})
		}
	}
	for _, op := range propertyOperations {
		for _, mode := range []string{"nil context", "cancel", "nil scope", "provider", "type", "endpoint", "base", "version", "protected source"} {
			t.Run(op.name+" lifetime "+mode, func(t *testing.T) {
				var calls, callbacks atomic.Int32
				client := propertyClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpectedHTTP") })
				scope := propertyScope(t, client, propertyParent)
				ctx := context.Background()
				expected := resource.ErrInvalidOption
				switch mode {
				case "nil context":
					ctx = nil
				case "cancel":
					c, cancel := context.WithCancel(ctx)
					cancel()
					ctx = c
					expected = context.Canceled
				case "nil scope":
					scope = nil
				case "provider":
					client.ProviderClient = &gophercloud.ProviderClient{}
				case "type":
					client.Type = "compute"
				case "endpoint":
					client.Endpoint = propertyBase + "other/"
				case "base":
					client.ResourceBase = propertyBase + "other/"
				case "version":
					client.Microversion = "2.3"
				case "protected source":
					client.MoreHeaders = map[string]string{"Authorization": "no"}
				}
				v, e := propertyCall(scope, ctx, op.name, propertyName, propertyCallbacks(func() { callbacks.Add(1) }, nil))
				if v != nil || !errors.Is(e, expected) || calls.Load() != 0 || callbacks.Load() != 0 {
					t.Fatal(v, e, calls.Load(), callbacks.Load())
				}
			})
		}
		for _, header := range []string{"X-Auth-Token", "Authorization", "Content-Type", "Accept", "Content-Length", "Transfer-Encoding", "OpenStack-API-Version", "X-OpenStack-Glance-API-Version", "bad header"} {
			t.Run(op.name+" header "+header, func(t *testing.T) {
				var calls atomic.Int32
				client := propertyClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpectedHTTP") })
				opts := propertyOptions{create: []properties.CreateOption{properties.WithCreateHeader(header, "bad")}, get: []properties.GetOption{properties.WithGetHeader(header, "bad")}, update: []properties.UpdateOption{properties.WithUpdateHeader(header, "bad")}, deletion: []properties.DeleteOption{properties.WithDeleteHeader(header, "bad")}, deleteAll: []properties.DeleteAllOption{properties.WithDeleteAllHeader(header, "bad")}, list: []properties.ListOption{properties.WithListHeader(header, "bad")}}
				v, e := propertyCall(propertyScope(t, client, propertyParent), context.Background(), op.name, propertyName, opts)
				if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
					t.Fatal(v, e, calls.Load())
				}
			})
		}
		for _, mode := range []string{"nil", "error", "cancel", "source"} {
			t.Run(op.name+" callback "+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("callback cause")
				var calls, callbacks atomic.Int32
				client := propertyClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpectedHTTP") })
				scope := propertyScope(t, client, propertyParent)
				callback := func() {
					callbacks.Add(1)
					if mode == "cancel" {
						cancel(cause)
					}
					if mode == "source" {
						client.Endpoint = propertyBase + "other/"
					}
				}
				callbackErr := error(nil)
				if mode == "error" {
					callbackErr = cause
				}
				opts := propertyCallbacks(callback, callbackErr)
				if mode == "nil" {
					opts = propertyOptions{create: []properties.CreateOption{nil}, get: []properties.GetOption{nil}, update: []properties.UpdateOption{nil}, deletion: []properties.DeleteOption{nil}, deleteAll: []properties.DeleteAllOption{nil}, list: []properties.ListOption{nil}}
				}
				v, e := propertyCall(scope, ctx, op.name, propertyName, opts)
				if v != nil || e == nil || calls.Load() != 0 {
					t.Fatal(v, e, calls.Load())
				}
				expected := int32(1)
				if mode == "nil" {
					expected = 0
				}
				if callbacks.Load() != expected {
					t.Fatal(callbacks.Load())
				}
				if mode == "error" || mode == "cancel" {
					if !errors.Is(e, cause) {
						t.Fatal(e)
					}
				} else if !errors.Is(e, resource.ErrInvalidOption) {
					t.Fatal(e)
				}
			})
		}
	}
	for _, op := range []string{"Create", "Update"} {
		for _, mode := range []string{"missing type", "missing title", "empty type", "invalid type", "invalid title", "invalid description", "invalid raw", "invalid key", "rename"} {
			t.Run(op+" required "+mode, func(t *testing.T) {
				var calls atomic.Int32
				client := propertyClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpectedHTTP") })
				scope := propertyScope(t, client, propertyParent)
				kind, title, desc := "future-type", "", ""
				attrs := map[string]json.RawMessage{}
				create := properties.CreateOpts{Type: &kind, Title: &title, Description: &desc, Attributes: attrs}
				update := properties.UpdateOpts{Type: &kind, Title: &title, Description: &desc, Attributes: attrs}
				switch mode {
				case "missing type":
					create.Type = nil
					update.Type = nil
				case "missing title":
					create.Title = nil
					update.Title = nil
				case "empty type":
					kind = ""
				case "invalid type":
					kind = string([]byte{0xff})
				case "invalid title":
					title = string([]byte{0xff})
				case "invalid description":
					desc = string([]byte{0xff})
				case "invalid raw":
					attrs["default"] = json.RawMessage(`{"x":`)
				case "invalid key":
					attrs[string([]byte{0xff})] = json.RawMessage("null")
				case "rename":
					if op == "Create" {
						return
					}
					bad := "../unsafe"
					update.Name = &bad
				}
				var v *properties.Property
				var e error
				if op == "Create" {
					v, e = scope.Create(context.Background(), propertyName, properties.WithCreateOpts(create))
				} else {
					v, e = scope.Update(context.Background(), propertyName, properties.WithUpdateOpts(update))
				}
				if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
					t.Fatal(v, e, calls.Load())
				}
			})
		}
		for _, key := range []string{"name", "type", "title", "description", "self", "schema", "created_at", "updated_at", "namespace_name"} {
			t.Run(op+" reserved "+key, func(t *testing.T) {
				var calls atomic.Int32
				client := propertyClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpectedHTTP") })
				opts := propertyOptions{create: []properties.CreateOption{properties.WithCreateAttribute(key, nil)}, update: []properties.UpdateOption{properties.WithUpdateAttribute(key, nil)}}
				v, e := propertyCall(propertyScope(t, client, propertyParent), context.Background(), op, propertyName, opts)
				if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
					t.Fatal(v, e, calls.Load())
				}
			})
		}
		for _, mode := range []string{"custom failure", "unsupported", "invalid RawMessage"} {
			t.Run(op+" factory error "+mode, func(t *testing.T) {
				var calls, marshals atomic.Int32
				cause := errors.New("factory marshal cause")
				var value any = propertyMarshal{&marshals, cause}
				if mode == "unsupported" {
					value = make(chan int)
				}
				if mode == "invalid RawMessage" {
					value = json.RawMessage(`{"x":`)
				}
				opts := propertyOptions{create: []properties.CreateOption{properties.WithCreateAttribute("future", value)}, update: []properties.UpdateOption{properties.WithUpdateAttributes(map[string]any{"future": value})}}
				client := propertyClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpectedHTTP") })
				v, e := propertyCall(propertyScope(t, client, propertyParent), context.Background(), op, propertyName, opts)
				if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
					t.Fatal(v, e, calls.Load())
				}
				if mode == "custom failure" && (!errors.Is(e, cause) || marshals.Load() != 2) {
					t.Fatal(e, marshals.Load())
				}
				if mode == "unsupported" {
					var typed *json.UnsupportedTypeError
					if !errors.As(e, &typed) {
						t.Fatal(e)
					}
				}
			})
		}
	}
	for _, op := range []string{"Create", "Update"} {
		t.Run(op+" original invalid UTF8 map key before serialization", func(t *testing.T) {
			var calls, marshals atomic.Int32
			data := map[string]any{string([]byte{0xff}): propertyMarshal{count: &marshals}}
			opts := propertyOptions{create: []properties.CreateOption{properties.WithCreateAttributes(data)}, update: []properties.UpdateOption{properties.WithUpdateAttributes(data)}}
			if marshals.Load() != 0 {
				t.Fatal("invalid key reached marshaler", marshals.Load())
			}
			client := propertyClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
			v, e := propertyCall(propertyScope(t, client, propertyParent), context.Background(), op, propertyName, opts)
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 || marshals.Load() != 0 {
				t.Fatal(v, e, calls.Load(), marshals.Load())
			}
		})
	}
	for _, query := range []string{"bad\nquery", string([]byte{0xff})} {
		t.Run("invalid query "+query, func(t *testing.T) {
			var calls atomic.Int32
			client := propertyClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpectedHTTP") })
			v, e := propertyScope(t, client, propertyParent).Get(context.Background(), propertyName, properties.WithGetResourceType(query))
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(v, e, calls.Load())
			}
		})
	}
	t.Run("negative local cap", func(t *testing.T) {
		var calls atomic.Int32
		client := propertyClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpectedHTTP") })
		v, e := propertyScope(t, client, propertyParent).All(context.Background(), properties.WithListMaxItems(-1))
		if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
			t.Fatal(v, e, calls.Load())
		}
	})
	for _, op := range propertyOperations {
		t.Run(op.name+" full replacement callback ownership", func(t *testing.T) {
			kind, title, desc := "number", "", "owned"
			ignore := true
			query := ""
			headers := map[string]string{"X-Owned": "yes"}
			attrs := map[string]json.RawMessage{"future": json.RawMessage(`{"x":1}`)}
			var escapedHeaders map[string]string
			var escapedRaw json.RawMessage
			var escapedType *string
			var escapedIgnore *bool
			var calls, callbacks atomic.Int32
			opts := propertyOptions{}
			opts.create = []properties.CreateOption{properties.WithCreateHeader("X-Dropped", "no"), properties.WithCreateOpts(properties.CreateOpts{Headers: headers, Type: &kind, Title: &title, Description: &desc, Attributes: attrs}), func(o *properties.CreateOpts) error {
				callbacks.Add(1)
				escapedHeaders = o.Headers
				escapedRaw = o.Attributes["future"]
				escapedType = o.Type
				return nil
			}, func(*properties.CreateOpts) error {
				escapedHeaders["X-Owned"] = "mutated"
				escapedRaw[len(escapedRaw)-2] = '2'
				*escapedType = "mutated"
				return nil
			}}
			opts.update = []properties.UpdateOption{properties.WithUpdateHeader("X-Dropped", "no"), properties.WithUpdateOpts(properties.UpdateOpts{Headers: headers, Type: &kind, Title: &title, Description: &desc, Attributes: attrs}), func(o *properties.UpdateOpts) error {
				callbacks.Add(1)
				escapedHeaders = o.Headers
				escapedRaw = o.Attributes["future"]
				escapedType = o.Type
				return nil
			}, func(*properties.UpdateOpts) error {
				escapedHeaders["X-Owned"] = "mutated"
				escapedRaw[len(escapedRaw)-2] = '2'
				*escapedType = "mutated"
				return nil
			}}
			opts.get = []properties.GetOption{properties.WithGetHeader("X-Dropped", "no"), properties.WithGetOpts(properties.GetOpts{Headers: headers, ResourceType: &query}), func(o *properties.GetOpts) error {
				callbacks.Add(1)
				escapedHeaders = o.Headers
				escapedType = o.ResourceType
				return nil
			}, func(*properties.GetOpts) error {
				escapedHeaders["X-Owned"] = "mutated"
				*escapedType = "mutated"
				return nil
			}}
			opts.deletion = []properties.DeleteOption{properties.WithDeleteHeader("X-Dropped", "no"), properties.WithDeleteOpts(properties.DeleteOpts{Headers: headers, IgnoreMissing: &ignore}), func(o *properties.DeleteOpts) error {
				callbacks.Add(1)
				escapedHeaders = o.Headers
				escapedIgnore = o.IgnoreMissing
				return nil
			}, func(*properties.DeleteOpts) error {
				escapedHeaders["X-Owned"] = "mutated"
				*escapedIgnore = false
				return nil
			}}
			opts.deleteAll = []properties.DeleteAllOption{properties.WithDeleteAllHeader("X-Dropped", "no"), properties.WithDeleteAllOpts(properties.DeleteAllOpts{Headers: headers}), func(o *properties.DeleteAllOpts) error { callbacks.Add(1); escapedHeaders = o.Headers; return nil }, func(*properties.DeleteAllOpts) error { escapedHeaders["X-Owned"] = "mutated"; return nil }}
			opts.list = []properties.ListOption{properties.WithListHeader("X-Dropped", "no"), properties.WithListOpts(properties.ListOpts{Headers: headers, MaxItems: 1}), func(o *properties.ListOpts) error { callbacks.Add(1); escapedHeaders = o.Headers; return nil }, func(*properties.ListOpts) error { escapedHeaders["X-Owned"] = "mutated"; return nil }}
			kind = "caller changed"
			desc = "caller changed"
			headers["X-Owned"] = "caller changed"
			attrs["future"][len(attrs["future"])-2] = '9'
			ignore = false
			query = "caller changed"
			client := propertyClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Header.Get("X-Owned") != "yes" || r.Header.Get("X-Dropped") != "" {
					t.Error(r.Header)
				}
				if op.name == "Get" && r.URL.RawQuery != "resource_type=" {
					t.Error(r.URL)
				}
				if op.name == "Create" || op.name == "Update" {
					raw, _ := io.ReadAll(r.Body)
					if string(raw) != `{"description":"owned","future":{"x":1},"name":"CPU Limits","title":"","type":"number"}` {
						t.Error(string(raw))
					}
				}
				code, raw := op.status, op.raw
				if op.name == "Delete" {
					code, raw = 404, "physical404"
				}
				if op.name == "List" || op.name == "All" {
					raw = `{"properties":{"first":{},"unused":null}}`
				}
				return propertyWire(code, &propertyBody{Reader: strings.NewReader(raw)}), nil
			})
			v, e := propertyCall(propertyScope(t, client, propertyParent), context.Background(), op.name, propertyName, opts)
			if e != nil || calls.Load() != 1 || callbacks.Load() != 1 || (op.name == "Delete" && v != nil) || (op.name != "Delete" && v == nil) {
				t.Fatal(v, e, calls.Load(), callbacks.Load())
			}
		})
	}
	t.Run("factory serializes once", func(t *testing.T) {
		var marshals, calls atomic.Int32
		option := properties.WithCreateAttribute("future", propertyMarshal{count: &marshals})
		if marshals.Load() != 1 {
			t.Fatal(marshals.Load())
		}
		client := propertyClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			raw, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(raw), `"future":{"at_factory":1}`) {
				t.Error(string(raw))
			}
			return propertyWire(201, &propertyBody{Reader: strings.NewReader(propertyJSON)}), nil
		})
		scope := propertyScope(t, client, propertyParent)
		for range 2 {
			v, e := scope.Create(context.Background(), propertyName, properties.WithCreateType("number"), properties.WithCreateTitle(""), option)
			if v == nil || e != nil {
				t.Fatal(v, e)
			}
		}
		if calls.Load() != 2 || marshals.Load() != 1 {
			t.Fatal(calls.Load(), marshals.Load())
		}
	})
	t.Run("latest source headers before callbacks and live token", func(t *testing.T) {
		var calls atomic.Int32
		client := propertyClient(nil)
		client.MoreHeaders = map[string]string{"X-Source": "at scope"}
		scope := propertyScope(t, client, propertyParent)
		client.HTTPClient.Transport = propertyTransport(func(r *http.Request) (*http.Response, error) {
			n := calls.Add(1)
			expected := []string{"first", "second", "third"}[n-1]
			if r.Header.Get("X-Source") != expected || r.Header.Get("X-Auth-Token") != expected {
				t.Error(r.Header, expected)
			}
			return propertyWire(200, &propertyBody{Reader: strings.NewReader(propertyJSON)}), nil
		})
		client.MoreHeaders["X-Source"] = "first"
		client.SetToken("first")
		v, e := scope.Get(context.Background(), propertyName)
		if v == nil || e != nil {
			t.Fatal(v, e)
		}
		client.MoreHeaders["X-Source"] = "second"
		client.SetToken("second")
		v, e = scope.Get(context.Background(), propertyName, func(*properties.GetOpts) error { client.MoreHeaders["X-Source"] = "third"; return nil })
		if v == nil || e != nil {
			t.Fatal(v, e)
		}
		client.SetToken("third")
		v, e = scope.Get(context.Background(), propertyName)
		if v == nil || e != nil || calls.Load() != 3 {
			t.Fatal(v, e, calls.Load())
		}
	})
}
func propertyCallbacks(f func(), cause error) propertyOptions {
	return propertyOptions{create: []properties.CreateOption{func(*properties.CreateOpts) error { f(); return cause }}, get: []properties.GetOption{func(*properties.GetOpts) error { f(); return cause }}, update: []properties.UpdateOption{func(*properties.UpdateOpts) error { f(); return cause }}, deletion: []properties.DeleteOption{func(*properties.DeleteOpts) error { f(); return cause }}, deleteAll: []properties.DeleteAllOption{func(*properties.DeleteAllOpts) error { f(); return cause }}, list: []properties.ListOption{func(*properties.ListOpts) error { f(); return cause }}}
}
func TestMetadefPropertiesResponseOwnershipAndNativeHooks(t *testing.T) {
	for _, op := range propertyOperations {
		for _, mode := range []string{"Read", "Close", "cancel", "Read Close cancel", "source after Close"} {
			t.Run(op.name+" accepted "+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("accepted Read"), errors.New("accepted Close"), errors.New("accepted cancel")
				raw := []byte(op.raw)
				if op.name == "Delete" || op.name == "DeleteAll" {
					raw = []byte{'a', 'c', 'k', 255}
				}
				b := &propertyBody{Reader: bytes.NewReader(raw)}
				if strings.Contains(mode, "Read") {
					b.Reader = propertyReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
				}
				if strings.Contains(mode, "Close") {
					b.closeErr = closeCause
				}
				var calls, hooks atomic.Int32
				client := propertyClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return propertyWire(op.status, b), nil })
				scope := propertyScope(t, client, propertyParent)
				if mode == "source after Close" {
					b.closeErr = nil
					b.onClose = func() { client.ResourceBase = "https://glance.invalid/replaced/v2/" }
				} else if strings.Contains(mode, "cancel") {
					b.onClose = func() { cancel(cancelCause) }
				}
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks.Add(1)
					return errors.New("accepted replay")
				}
				v, e := propertyCall(scope, ctx, op.name, propertyName, propertyOptions{})
				proof := propertyProof(t, e, op.status, raw)
				if calls.Load() != 1 || hooks.Load() != 0 || b.closes.Load() != 1 || strings.Contains(mode, "Read") && !errors.Is(e, readCause) || mode != "source after Close" && strings.Contains(mode, "Close") && !errors.Is(e, closeCause) || strings.Contains(mode, "cancel") && (!errors.Is(e, cancelCause) || !errors.Is(e, context.Canceled)) || mode == "source after Close" && !errors.Is(e, resource.ErrInvalidOption) {
					t.Fatal(v, e, calls.Load(), hooks.Load(), b.closes.Load())
				}
				if op.name == "Delete" || op.name == "DeleteAll" {
					if v == nil || v.ack == nil || v.ack.Namespace != propertyParent || v.ack.StatusCode != 204 || !bytes.Equal(v.ack.Body, raw) || op.name == "Delete" && (v.ack.Name == nil || *v.ack.Name != propertyName) || op.name == "DeleteAll" && v.ack.Name != nil {
						t.Fatal(v, e)
					}
					proof.Body[0] = '!'
					proof.Header.Set("X-Request-Id", "proof changed")
					if v.ack.Body[0] != 'a' || v.ack.Header.Get("X-Request-Id") != "actual-property" {
						t.Fatal("ack aliases error proof", v.ack)
					}
				} else if v != nil {
					t.Fatal("partial typed result", v, e)
				}
			})
		}
	}
	for _, op := range []string{"Delete", "DeleteAll"} {
		t.Run(op+" opaque204", func(t *testing.T) {
			raw := []byte{255, 0, 1}
			b := &propertyBody{Reader: bytes.NewReader(raw)}
			wire := propertyWire(204, b)
			var calls atomic.Int32
			client := propertyClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Method != "DELETE" || r.Body != nil || r.URL.RawQuery != "" {
					t.Error(r.Method, r.URL, r.Body)
				}
				return wire, nil
			})
			v, e := propertyCall(propertyScope(t, client, propertyParent), context.Background(), op, propertyName, propertyOptions{})
			if e != nil || v == nil || v.ack == nil || !bytes.Equal(v.ack.Body, raw) || v.ack.StatusCode != 204 || b.closes.Load() != 1 || calls.Load() != 1 {
				t.Fatal(v, e, b.closes.Load(), calls.Load())
			}
			wire.Header.Set("X-Request-Id", "late")
			raw[0] = 1
			if v.ack.Header.Get("X-Request-Id") != "actual-property" || v.ack.Body[0] != 255 {
				t.Fatal("borrowed ack alias", v.ack)
			}
		})
	}
	for _, op := range propertyOperations {
		for _, code := range []int{202, 404, 409} {
			if op.name == "Delete" && code == 404 {
				continue
			}
			t.Run(fmt.Sprintf("%s strict%d", op.name, code), func(t *testing.T) {
				raw := []byte("server-owned rejection")
				b := &propertyBody{Reader: bytes.NewReader(raw)}
				var calls atomic.Int32
				client := propertyClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return propertyWire(code, b), nil })
				v, e := propertyCall(propertyScope(t, client, propertyParent), context.Background(), op.name, propertyName, propertyOptions{})
				var native gophercloud.ErrUnexpectedResponseCode
				if v != nil || !errors.As(e, &native) || native.Actual != code || !bytes.Equal(native.Body, raw) || native.ResponseHeader.Get("X-Request-Id") != "actual-property" || calls.Load() != 1 || b.closes.Load() != 1 {
					t.Fatal(v, e, native, calls.Load(), b.closes.Load())
				}
			})
		}
	}
	for _, mode := range []string{"default", "explicit true", "explicit false", "DeleteAll strict"} {
		t.Run("physical404 "+mode, func(t *testing.T) {
			var calls, hooks atomic.Int32
			hookCause := errors.New("strict callback")
			b := &propertyBody{Reader: strings.NewReader("actual404")}
			client := propertyClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return propertyWire(404, b), nil })
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				hooks.Add(1)
				return hookCause
			}
			opts := propertyOptions{}
			op := "Delete"
			if mode == "DeleteAll strict" {
				op = "DeleteAll"
			}
			if mode == "explicit true" || mode == "explicit false" {
				opts.deletion = []properties.DeleteOption{properties.WithDeleteIgnoreMissing(mode == "explicit true")}
			}
			v, e := propertyCall(propertyScope(t, client, propertyParent), context.Background(), op, propertyName, opts)
			if v != nil || calls.Load() != 1 || b.closes.Load() != 1 {
				t.Fatal(v, e, calls.Load(), b.closes.Load())
			}
			if mode == "explicit false" || mode == "DeleteAll strict" {
				if !errors.Is(e, hookCause) || !gophercloud.ResponseCodeIs(e, 404) || hooks.Load() != 1 {
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
			raw := []byte("physical404 partial")
			b := &propertyBody{Reader: bytes.NewReader(raw)}
			if strings.Contains(mode, "Read") {
				b.Reader = propertyReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
			}
			if strings.Contains(mode, "Close") {
				b.closeErr = closeCause
			}
			var calls, hooks atomic.Int32
			client := propertyClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return propertyWire(404, b), nil })
			scope := propertyScope(t, client, propertyParent)
			if strings.Contains(mode, "cancel") {
				b.onClose = func() { cancel(cancelCause) }
			}
			if mode == "source" {
				b.onClose = func() { client.ProviderClient = &gophercloud.ProviderClient{} }
			}
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				hooks.Add(1)
				return errors.New("404 replay")
			}
			v, e := scope.Delete(ctx, propertyName)
			propertyProof(t, e, 404, raw)
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
			b := &propertyBody{Reader: strings.NewReader("original failure")}
			client := propertyClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				if mode == "transport nested404" {
					return nil, errors.Join(cause, nested)
				}
				code := 503
				if mode == "reauth nested404" {
					code = 401
				}
				return propertyWire(code, b), nil
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
			v, e := propertyScope(t, client, propertyParent).Delete(context.Background(), propertyName)
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
	t.Run("configured prebody hooks original provider and live auth", func(t *testing.T) {
		var calls, reauth, backoff, retries atomic.Int32
		var bodies []*propertyBody
		transportCause := errors.New("prebody transport")
		client := propertyClient(nil)
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
			if !o.KeepResponseBody || o.JSONResponse != nil || o.RawBody != nil || string(raw) != propertyPayload {
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
		scope := propertyScope(t, client, propertyParent)
		client.HTTPClient.Transport = propertyTransport(func(r *http.Request) (*http.Response, error) {
			i := int(calls.Add(1)) - 1
			codes := []int{401, 429, 503, 0, 201}
			tokens := []string{"initial", "reauth", "backoff", "retry503", "retrytransport"}
			if i >= len(codes) {
				return nil, errors.New("replay")
			}
			raw, _ := io.ReadAll(r.Body)
			if r.Method != "POST" || r.URL.String() != propertyBase+propertyCollection || string(raw) != propertyPayload || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != tokens[i] {
				t.Error(r.Method, r.URL, string(raw), r.Header)
			}
			if codes[i] == 0 {
				return nil, transportCause
			}
			b := &propertyBody{Reader: strings.NewReader(propertyJSON)}
			bodies = append(bodies, b)
			return propertyWire(codes[i], b), nil
		})
		v, e := scope.Create(context.Background(), propertyName, properties.WithCreateType("number"), properties.WithCreateTitle(""))
		if v == nil || e != nil || calls.Load() != 5 || reauth.Load() != 1 || backoff.Load() != 1 || retries.Load() != 2 || client.ProviderClient != originalProvider || reflect.ValueOf(client.RetryFunc).Pointer() != originalRetry {
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
			callbackCause := errors.New("retry callback")
			var calls, hooks, borrowedReads atomic.Int32
			b := &propertyBody{Reader: strings.NewReader("original503")}
			borrowed := &propertyBody{Reader: propertyReader(func([]byte) (int, error) { borrowedReads.Add(1); return 0, io.EOF })}
			client := propertyClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				raw, _ := io.ReadAll(r.Body)
				if string(raw) != propertyPayload {
					t.Error(string(raw))
				}
				return propertyWire(503, b), nil
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
				return callbackCause
			}
			v, e := propertyScope(t, client, propertyParent).Create(context.Background(), propertyName, properties.WithCreateType("number"), properties.WithCreateTitle(""))
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
	t.Run("bodyless null differs from absent", func(t *testing.T) {
		var calls atomic.Int32
		b := &propertyBody{Reader: strings.NewReader("original503")}
		client := propertyClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Body != nil {
				t.Error(r.Body)
			}
			return propertyWire(503, b), nil
		})
		client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			o.JSONBody = json.RawMessage("null")
			return nil
		}
		v, e := propertyScope(t, client, propertyParent).DeleteAll(context.Background())
		if v != nil || !errors.Is(e, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(e, 503) || calls.Load() != 1 || b.closes.Load() != 1 {
			t.Fatal(v, e, calls.Load(), b.closes.Load())
		}
	})
	for _, op := range []string{"Create", "Update"} {
		t.Run(op+" same serialized replacement", func(t *testing.T) {
			var calls, hooks atomic.Int32
			var bodies []*propertyBody
			code := 201
			if op == "Update" {
				code = 200
			}
			client := propertyClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				raw, _ := io.ReadAll(r.Body)
				if string(raw) != propertyPayload {
					t.Error(string(raw))
				}
				status, reply := 503, "original503"
				if n == 2 {
					status, reply = code, propertyJSON
				}
				b := &propertyBody{Reader: strings.NewReader(reply)}
				bodies = append(bodies, b)
				return propertyWire(status, b), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				hooks.Add(1)
				o.JSONBody = map[string]string{"name": propertyName, "type": "number", "title": ""}
				return nil
			}
			v, e := propertyCall(propertyScope(t, client, propertyParent), context.Background(), op, propertyName, propertyOptions{})
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
	for _, op := range propertyOperations {
		t.Run(op.name+" expanded codes reject actual response", func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("unexpected Read"), errors.New("unexpected Close"), errors.New("unexpected cause")
			var calls, hooks atomic.Int32
			var bodies []*propertyBody
			client := propertyClient(func(*http.Request) (*http.Response, error) {
				n := calls.Add(1)
				code, raw := 503, "original503"
				if n == 2 {
					code, raw = 202, "unexpected actual202"
				}
				b := &propertyBody{Reader: strings.NewReader(raw)}
				if n == 2 {
					b.Reader = propertyReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
					b.closeErr = closeCause
					b.onClose = func() { cancel(cancelCause) }
				}
				bodies = append(bodies, b)
				return propertyWire(code, b), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, e error, _ uint) error {
				hooks.Add(1)
				if !gophercloud.ResponseCodeIs(e, 503) {
					return e
				}
				o.OkCodes = []int{202}
				return nil
			}
			v, e := propertyCall(propertyScope(t, client, propertyParent), ctx, op.name, propertyName, propertyOptions{})
			var native gophercloud.ErrUnexpectedResponseCode
			var proof *resource.ResponseError
			expected := []int{op.status}
			if op.name == "Delete" {
				expected = append(expected, 404)
			}
			target := propertyBase + propertyCollection
			if op.named {
				target += "/" + url.PathEscape(propertyName)
			}
			if v != nil || !errors.As(e, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, expected) || native.Method != op.method || native.URL != target || string(native.Body) != "unexpected actual202" || native.ResponseHeader.Get("X-Request-Id") != "actual-property" || !errors.Is(e, readCause) || !errors.Is(e, closeCause) || !errors.Is(e, cancelCause) || !errors.Is(e, context.Canceled) || errors.As(e, &proof) || calls.Load() != 2 || hooks.Load() != 1 {
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
			var bodies []*propertyBody
			client := propertyClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				b := &propertyBody{Reader: strings.NewReader(propertyJSON)}
				bodies = append(bodies, b)
				if n == 2 {
					raw, _ := io.ReadAll(r.Body)
					if r.Method != "POST" || string(raw) != propertyPayload {
						t.Error(r.Method, string(raw))
					}
					return propertyWire(201, b), nil
				}
				code, target := 307, propertyBase+propertyCollection
				switch redirect {
				case "foreign origin":
					target = "https://foreign.invalid/property"
				case "changed path":
					target = propertyBase + "metadefs/namespaces/other/properties"
				case "changed query":
					target += "?marker=other"
				case "changed method":
					code = 303
				}
				wire := propertyWire(code, b)
				wire.Header.Set("Location", target)
				return wire, nil
			})
			client.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects.Add(1); return nil }
			v, e := propertyScope(t, client, propertyParent).Create(context.Background(), propertyName, properties.WithCreateType("number"), properties.WithCreateTitle(""))
			if redirect == "same target" {
				if v == nil || e != nil || calls.Load() != 2 || redirects.Load() != 1 {
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
