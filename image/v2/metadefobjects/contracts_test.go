package metadefobjects_test

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

	objects "github.com/JSYoo5B/gophercloudsdk/image/v2/metadefobjects"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const objectPrefix = "/reverse/object/glance/v2/"
const objectBase = "https://glance.invalid" + objectPrefix
const objectParent = "OS::Compute::Libvirt"
const objectName = "CPU Limits"
const objectCollection = "metadefs/namespaces/" + objectParent + "/objects"
const objectJSON = `{"name":"foreign response name","description":"server description","properties":{"quota:cpu":{"type":"integer","maximum":9007199254740993}},"required":["quota:cpu"],"created_at":"literal-date","self":"https://passive.invalid/entity","schema":"https://passive.invalid/schema"}`
const objectListJSON = `{"objects":[` + objectJSON + `],"next":42,"schema":false}`

type objectTransport func(*http.Request) (*http.Response, error)

func (f objectTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	v, e := f(r)
	if v != nil && v.Request == nil {
		v.Request = r
	}
	return v, e
}

type objectBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *objectBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type objectReader func([]byte) (int, error)

func (f objectReader) Read(p []byte) (int, error) { return f(p) }
func objectWire(status int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"actual-object"}}, Body: body}
}
func objectClient(f objectTransport) *gophercloud.ServiceClient {
	p := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: f}}
	p.UseTokenLock()
	p.SetToken("initial")
	return &gophercloud.ServiceClient{ProviderClient: p, Type: "image", Endpoint: objectBase}
}
func objectScope(t *testing.T, client *gophercloud.ServiceClient, parent string) *objects.NamespaceScope {
	t.Helper()
	api := objects.New(client)
	scope, e := api.InNamespace(context.Background(), parent)
	if e != nil || scope == nil || api.RawClient() != client || scope.RawClient() != client || scope.NamespaceName() != parent {
		t.Fatalf("scope %v %v", scope, e)
	}
	return scope
}

type objectOptions struct {
	create    []objects.CreateOption
	get       []objects.GetOption
	update    []objects.UpdateOption
	deletion  []objects.DeleteOption
	deleteAll []objects.DeleteAllOption
	list      []objects.ListOption
}
type objectResult struct {
	value *objects.Object
	ack   *objects.Acknowledgement
	rows  []*objects.Object
}

func objectCall(scope *objects.NamespaceScope, ctx context.Context, op, name string, opts objectOptions) (*objectResult, error) {
	var value *objects.Object
	var e error
	switch op {
	case "Create":
		value, e = scope.Create(ctx, name, opts.create...)
	case "Get":
		value, e = scope.Get(ctx, name, opts.get...)
	case "Update":
		value, e = scope.Update(ctx, name, opts.update...)
	case "Delete":
		v, e := scope.Delete(ctx, name, opts.deletion...)
		if v == nil {
			return nil, e
		}
		return &objectResult{ack: v}, e
	case "DeleteAll":
		v, e := scope.DeleteAll(ctx, opts.deleteAll...)
		if v == nil {
			return nil, e
		}
		return &objectResult{ack: v}, e
	case "All":
		v, e := scope.All(ctx, opts.list...)
		if v == nil {
			return nil, e
		}
		return &objectResult{rows: v}, e
	case "List":
		rows := make([]*objects.Object, 0)
		for v, e := range scope.List(ctx, opts.list...) {
			if e != nil {
				return nil, e
			}
			rows = append(rows, v)
		}
		return &objectResult{rows: rows}, nil
	default:
		panic("unknown operation")
	}
	if value == nil {
		return nil, e
	}
	return &objectResult{value: value}, e
}

var objectOperations = []struct {
	name, method, body, raw string
	status                  int
	named                   bool
}{
	{"Create", "POST", `{"name":"CPU Limits"}`, objectJSON, 201, false},
	{"Get", "GET", "", objectJSON, 200, true},
	{"Update", "PUT", `{"name":"CPU Limits"}`, objectJSON, 200, true},
	{"Delete", "DELETE", "", "", 204, true},
	{"DeleteAll", "DELETE", "", "", 204, false},
	{"List", "GET", "", objectListJSON, 200, false},
	{"All", "GET", "", objectListJSON, 200, false},
}

func objectProof(t *testing.T, e error, status int, raw []byte) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(e, &proof) || proof.StatusCode != status || !bytes.Equal(proof.Body, raw) || proof.Header.Get("X-Request-Id") != "actual-object" {
		t.Fatalf("owned proof %v %+v", e, proof)
	}
	return proof
}
func objectHeaders() objectOptions {
	return objectOptions{create: []objects.CreateOption{objects.WithCreateHeaders(map[string]string{"X-Option": "owned"}), objects.WithCreateHeader("X-Final", "yes")}, get: []objects.GetOption{objects.WithGetHeaders(map[string]string{"X-Option": "owned"}), objects.WithGetHeader("X-Final", "yes")}, update: []objects.UpdateOption{objects.WithUpdateHeaders(map[string]string{"X-Option": "owned"}), objects.WithUpdateHeader("X-Final", "yes")}, deletion: []objects.DeleteOption{objects.WithDeleteHeaders(map[string]string{"X-Option": "owned"}), objects.WithDeleteHeader("X-Final", "yes")}, deleteAll: []objects.DeleteAllOption{objects.WithDeleteAllHeaders(map[string]string{"X-Option": "owned"}), objects.WithDeleteAllHeader("X-Final", "yes")}, list: []objects.ListOption{objects.WithListHeaders(map[string]string{"X-Option": "owned"}), objects.WithListHeader("X-Final", "yes")}}
}

func TestMetadefObjectsFixedScopedRoutesAndPayloads(t *testing.T) {
	for _, op := range objectOperations {
		t.Run(op.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := cloud.Client("image", "/catalog/unused/")
			client.ResourceBase = cloud.Server.URL + objectPrefix
			client.Microversion = "2.2"
			var calls atomic.Int32
			scope := objectScope(t, client, objectParent)
			if calls.Load() != 0 {
				t.Fatal("scope HTTP")
			}
			client.MoreHeaders = map[string]string{"X-Source": "latest"}
			cloud.Provider.SetToken("live")
			cloud.Mux.HandleFunc(objectPrefix, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				want := objectPrefix + objectCollection
				if op.named {
					want += "/" + url.PathEscape(objectName)
				}
				if r.Method != op.method || r.URL.EscapedPath() != want || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "live" || r.Header.Get("X-Source") != "latest" || r.Header.Get("X-Option") != "owned" || r.Header.Get("X-Final") != "yes" || r.Header.Get("OpenStack-API-Version") != "image 2.2" {
					t.Error(r.Method, r.URL, r.Header, want)
				}
				raw, _ := io.ReadAll(r.Body)
				if string(raw) != op.body {
					t.Error(string(raw))
				}
				w.Header().Set("X-Request-Id", "actual-object")
				w.Header().Set("Location", "https://passive.invalid/new")
				testcloud.JSON(w, op.status, op.raw)
			})
			v, e := objectCall(scope, context.Background(), op.name, objectName, objectHeaders())
			if e != nil || v == nil || calls.Load() != 1 || scope.NamespaceName() != objectParent {
				t.Fatal(v, e, calls.Load())
			}
			if v.value != nil && (*v.value.Name != "foreign response name" || v.value.StatusCode != op.status || v.value.Header.Get("Location") != "https://passive.invalid/new") {
				t.Fatal(v.value)
			}
			if v.ack != nil && (v.ack.Namespace != objectParent || v.ack.StatusCode != 204 || op.name == "Delete" && (v.ack.Name == nil || *v.ack.Name != objectName) || op.name == "DeleteAll" && v.ack.Name != nil) {
				t.Fatal(v.ack)
			}
			if v.rows != nil && len(v.rows) != 1 {
				t.Fatal(v.rows)
			}
		})
	}
	for _, parent := range []string{"OS::Vendor Name::한글", strings.Repeat("界", 80)} {
		for _, name := range []string{"object::한글 name", strings.Repeat("名", 80)} {
			for _, op := range objectOperations {
				t.Run(op.name+" literal "+parent+" "+name, func(t *testing.T) {
					var calls atomic.Int32
					client := objectClient(func(r *http.Request) (*http.Response, error) {
						calls.Add(1)
						want := objectPrefix + "metadefs/namespaces/" + url.PathEscape(parent) + "/objects"
						if op.named {
							want += "/" + url.PathEscape(name)
						}
						if r.URL.EscapedPath() != want || r.URL.RawQuery != "" {
							t.Error(r.URL, r.URL.EscapedPath(), want)
						}
						if op.name == "Create" || op.name == "Update" {
							raw, _ := io.ReadAll(r.Body)
							var payload map[string]string
							if json.Unmarshal(raw, &payload) != nil || len(payload) != 1 || payload["name"] != name {
								t.Error(string(raw))
							}
						}
						return objectWire(op.status, io.NopCloser(strings.NewReader(op.raw))), nil
					})
					scope := objectScope(t, client, parent)
					v, e := objectCall(scope, context.Background(), op.name, name, objectOptions{})
					if e != nil || v == nil || calls.Load() != 1 || scope.NamespaceName() != parent || v.ack != nil && v.ack.Namespace != parent {
						t.Fatal(v, e, calls.Load())
					}
				})
			}
		}
	}
	for _, mode := range []string{"nil omitted", "explicit empty", "literal schema and Required"} {
		for _, op := range []string{"Create", "Update"} {
			t.Run(op+" "+mode, func(t *testing.T) {
				var calls atomic.Int32
				description := ""
				properties := map[string]json.RawMessage{}
				required := []string{}
				if mode == "literal schema and Required" {
					description = "line1\n" + strings.Repeat("界", 600)
					properties = map[string]json.RawMessage{"": json.RawMessage(`{}`), "field,\nname": json.RawMessage(`{"no_schema_whitelist":true,"maximum":9007199254740993,"extension":[null,1e400]}`)}
					required = []string{"", "field,\nname", "field,\nname"}
				}
				opts := objectOptions{}
				if mode != "nil omitted" {
					opts.create = []objects.CreateOption{objects.WithCreateDescription(description), objects.WithCreateProperties(properties), objects.WithCreateRequired(required)}
					opts.update = []objects.UpdateOption{objects.WithUpdateDescription(description), objects.WithUpdateProperties(properties), objects.WithUpdateRequired(required)}
				}
				if op == "Update" {
					opts.update = append(opts.update, objects.WithUpdateName("renamed::object"))
				}
				client := objectClient(func(r *http.Request) (*http.Response, error) {
					calls.Add(1)
					raw, _ := io.ReadAll(r.Body)
					var payload map[string]json.RawMessage
					if json.Unmarshal(raw, &payload) != nil {
						t.Error(string(raw))
					}
					name := objectName
					if op == "Update" {
						name = "renamed::object"
						if r.Method != "PUT" || r.URL.EscapedPath() != objectPrefix+objectCollection+"/"+url.PathEscape(objectName) {
							t.Error(r.Method, r.URL)
						}
					}
					if string(payload["name"]) != fmt.Sprintf("%q", name) {
						t.Error(string(raw))
					}
					if mode == "nil omitted" {
						if len(payload) != 1 {
							t.Error(string(raw))
						}
					} else {
						var actualDescription string
						var actualRequired []string
						var actualProperties map[string]json.RawMessage
						if len(payload) != 4 || json.Unmarshal(payload["description"], &actualDescription) != nil || actualDescription != description || json.Unmarshal(payload["required"], &actualRequired) != nil || !reflect.DeepEqual(actualRequired, required) || json.Unmarshal(payload["properties"], &actualProperties) != nil || !reflect.DeepEqual(actualProperties, properties) {
							t.Error(string(raw), description, required, properties)
						}
					}
					code := 201
					if op == "Update" {
						code = 200
					}
					return objectWire(code, io.NopCloser(strings.NewReader(objectJSON))), nil
				})
				v, e := objectCall(objectScope(t, client, objectParent), context.Background(), op, objectName, opts)
				if v == nil || e != nil || calls.Load() != 1 {
					t.Fatal(v, e, calls.Load())
				}
			})
		}
	}
	t.Run("creation context not retained", func(t *testing.T) {
		var calls atomic.Int32
		client := objectClient(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return objectWire(200, io.NopCloser(strings.NewReader(objectJSON))), nil
		})
		ctx, cancel := context.WithCancel(context.Background())
		scope, e := objects.New(client).InNamespace(ctx, objectParent)
		cancel()
		if e != nil || scope == nil || calls.Load() != 0 {
			t.Fatal(scope, e, calls.Load())
		}
		v, e := scope.Get(context.Background(), objectName)
		if e != nil || v == nil || calls.Load() != 1 {
			t.Fatal(v, e, calls.Load())
		}
	})
}

func TestMetadefObjectsCanonicalModelsAndRawOwnership(t *testing.T) {
	for _, raw := range []string{`{}`, `{"name":null,"description":null,"self":null,"schema":null,"created_at":null,"updated_at":null,"properties":null,"required":null}`, `{"name":"","description":"","self":"","schema":"","created_at":"","updated_at":"","properties":{},"required":[]}`, `{"name":"other object","description":"line\ntext","self":"https://foreign.invalid","schema":"https://foreign.invalid/schema","created_at":"unparsed","updated_at":"unparsed","properties":{"null":null,"bool":false,"number":9007199254740993,"string":"literal","array":[1e400],"object":{"title":false}},"required":["","field,\nname","field,\nname"],"Name":42,"namespace":false,"id":123,"links":false,"extension":9007199254740993}`} {
		t.Run("passive "+raw, func(t *testing.T) {
			b := &objectBody{Reader: strings.NewReader(raw)}
			wire := objectWire(200, b)
			client := objectClient(func(*http.Request) (*http.Response, error) { return wire, nil })
			v, e := objectScope(t, client, objectParent).Get(context.Background(), objectName)
			if e != nil || v == nil || v.Links != nil || v.StatusCode != 200 || b.closes.Load() != 1 {
				t.Fatal(v, e, b.closes.Load())
			}
			for _, field := range []struct {
				key string
				p   *string
			}{{"name", v.Name}, {"description", v.Description}, {"self", v.Self}, {"schema", v.Schema}, {"created_at", v.CreatedAt}, {"updated_at", v.UpdatedAt}} {
				token, exists := v.Body[field.key]
				if !exists || string(token) == "null" {
					if field.p != nil {
						t.Fatal(field.key, field.p)
					}
				} else {
					var want string
					if json.Unmarshal(token, &want) != nil || field.p == nil || *field.p != want {
						t.Fatal(field.key, field.p)
					}
				}
			}
			var props map[string]json.RawMessage
			if json.Unmarshal(v.Body["properties"], &props) == nil && !reflect.DeepEqual(v.Properties, props) {
				t.Fatal(v.Properties, props)
			}
			var req []string
			if json.Unmarshal(v.Body["required"], &req) == nil && !reflect.DeepEqual(v.Required, req) {
				t.Fatal(v.Required, req)
			}
			if strings.Contains(raw, "extension") && string(v.Body["extension"]) != "9007199254740993" {
				t.Fatal(v.Body)
			}
			wire.Header.Set("X-Request-Id", "wire mutation")
			if v.Header.Get("X-Request-Id") != "actual-object" {
				t.Fatal("header alias")
			}
			if v.Name != nil {
				before := *v.Name
				v.Body["name"][1] = '!'
				if *v.Name != before {
					t.Fatal("typed string aliases raw")
				}
			}
			if len(v.Properties) > 0 {
				before := string(v.Properties["number"])
				v.Body["properties"][1] = '!'
				if string(v.Properties["number"]) != before {
					t.Fatal("property bytes alias raw root")
				}
				v.Properties["number"][0] = '1'
				if !bytes.Contains(v.Body["properties"], []byte("9007199254740993")) {
					t.Fatal("raw root aliases typed map", v.Body)
				}
			}
			if len(v.Required) > 0 {
				before := append([]byte(nil), v.Body["required"]...)
				v.Required[0] = "caller"
				if !bytes.Equal(v.Body["required"], before) {
					t.Fatal("required aliases raw")
				}
			}
		})
	}
	for _, field := range []string{"name", "description", "self", "schema", "created_at", "updated_at", "properties", "required"} {
		tokens := []string{"1", "false"}
		if field == "properties" {
			tokens = append(tokens, `[]`, `"text"`)
		} else if field == "required" {
			tokens = append(tokens, `{}`, `[null]`, `[1]`, `[false]`)
		} else {
			tokens = append(tokens, `[]`, `{}`)
		}
		for _, token := range tokens {
			t.Run(field+" wrong "+token, func(t *testing.T) {
				raw := []byte(fmt.Sprintf(`{%q:%s}`, field, token))
				b := &objectBody{Reader: bytes.NewReader(raw)}
				client := objectClient(func(*http.Request) (*http.Response, error) { return objectWire(200, b), nil })
				v, e := objectScope(t, client, objectParent).Get(context.Background(), objectName)
				objectProof(t, e, 200, raw)
				if v != nil || b.closes.Load() != 1 {
					t.Fatal(v, e, b.closes.Load())
				}
			})
		}
	}
	for _, raw := range [][]byte{[]byte(`null`), []byte(`[]`), []byte(`1`), []byte(`{"name":`), []byte("{\"unknown\":\"\xff\"}")} {
		t.Run("strict201 "+string(raw), func(t *testing.T) {
			b := &objectBody{Reader: bytes.NewReader(raw)}
			client := objectClient(func(*http.Request) (*http.Response, error) { return objectWire(201, b), nil })
			v, e := objectScope(t, client, objectParent).Create(context.Background(), objectName)
			objectProof(t, e, 201, raw)
			if v != nil || b.closes.Load() != 1 {
				t.Fatal(v, e, b.closes.Load())
			}
		})
	}
	t.Run("finite rows independent raw header and typed data", func(t *testing.T) {
		raw := `{"objects":[{"name":"one","properties":{"value":9007199254740993},"required":["one"]},{"name":"two","properties":{"value":9007199254740993},"required":["two"]},null],"next":"https://foreign.invalid"}`
		var calls atomic.Int32
		client := objectClient(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return objectWire(200, io.NopCloser(strings.NewReader(raw))), nil
		})
		scope := objectScope(t, client, objectParent)
		rows, e := scope.All(context.Background(), objects.WithListMaxItems(2))
		if e != nil || len(rows) != 2 {
			t.Fatal(rows, e)
		}
		rows[0].Header.Set("X-Request-Id", "caller")
		rows[0].Properties["value"][0] = '1'
		rows[0].Body["name"][1] = '!'
		rows[0].Required[0] = "caller"
		if rows[1].Header.Get("X-Request-Id") != "actual-object" || string(rows[1].Properties["value"]) != "9007199254740993" || *rows[0].Name != "one" || rows[1].Required[0] != "two" || !bytes.Contains(rows[0].Body["properties"], []byte("9007199254740993")) {
			t.Fatal(rows)
		}
		rows, e = scope.All(context.Background())
		objectProof(t, e, 200, []byte(raw))
		if rows != nil || calls.Load() != 2 {
			t.Fatal(rows, e, calls.Load())
		}
	})
}

func TestMetadefObjectsFiniteListAndLocalControls(t *testing.T) {
	for _, cap := range []int{0, 1, 3, 20} {
		t.Run(fmt.Sprint("local cap ", cap), func(t *testing.T) {
			var calls atomic.Int32
			raw := `{"objects":[{},{}],"next":"https://foreign.invalid/will-not-follow","first":42,"schema":false}`
			client := objectClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Method != "GET" || r.URL.RawQuery != "" || r.Body != nil || r.URL.EscapedPath() != objectPrefix+objectCollection {
					t.Error(r.Method, r.URL, r.Body)
				}
				wire := objectWire(200, io.NopCloser(strings.NewReader(raw)))
				wire.Header.Set("Link", `<https://foreign.invalid>; rel="next"`)
				return wire, nil
			})
			rows, e := objectScope(t, client, objectParent).All(context.Background(), objects.WithListMaxItems(cap))
			want := 2
			if cap == 1 {
				want = 1
			}
			if e != nil || len(rows) != want || calls.Load() != 1 {
				t.Fatal(rows, e, calls.Load())
			}
		})
	}
	for _, raw := range []string{`{"objects":[],"next":42}`, `{"objects":[{}],"next":{"malformed":true}}`, `{"objects":[{}],"next":"/v2/metadefs/namespaces/other/objects?marker=Next"}`, `{"objects":[{}],"links":[{"rel":"next","href":"https://foreign.invalid"}],"first":false}`} {
		t.Run("passive continuation "+raw, func(t *testing.T) {
			var calls atomic.Int32
			client := objectClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return objectWire(200, io.NopCloser(strings.NewReader(raw))), nil
			})
			rows, e := objectScope(t, client, objectParent).All(context.Background())
			if e != nil || rows == nil || calls.Load() != 1 {
				t.Fatal(rows, e, calls.Load())
			}
		})
	}
	for _, mode := range []string{"cap", "break", "unlimited late error"} {
		t.Run(mode, func(t *testing.T) {
			raw := `{"objects":[{},null],"next":42}`
			var calls atomic.Int32
			b := &objectBody{Reader: strings.NewReader(raw)}
			client := objectClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return objectWire(200, b), nil })
			scope := objectScope(t, client, objectParent)
			if mode == "break" {
				n := 0
				for v, e := range scope.List(context.Background()) {
					if v == nil || e != nil {
						t.Fatal(v, e)
					}
					n++
					break
				}
				if n != 1 {
					t.Fatal(n)
				}
			} else {
				opts := []objects.ListOption{}
				if mode == "cap" {
					opts = append(opts, objects.WithListMaxItems(1))
				}
				rows, e := scope.All(context.Background(), opts...)
				if mode == "cap" {
					if e != nil || len(rows) != 1 {
						t.Fatal(rows, e)
					}
				} else {
					objectProof(t, e, 200, []byte(raw))
					if rows != nil {
						t.Fatal(rows, e)
					}
				}
			}
			if calls.Load() != 1 || b.closes.Load() != 1 {
				t.Fatal(calls.Load(), b.closes.Load())
			}
		})
	}
	for _, raw := range []string{`{}`, `null`, `[]`, `{"objects":null}`, `{"objects":{}}`, `{"Objects":[]}`, `{"objects":[[]]}`, `{"objects":[{"required":[null]}]}`, "{\"objects\":[{}],\"unknown\":\"\xff\"}"} {
		t.Run("strict envelope "+raw, func(t *testing.T) {
			client := objectClient(func(*http.Request) (*http.Response, error) {
				return objectWire(200, io.NopCloser(strings.NewReader(raw))), nil
			})
			rows, e := objectScope(t, client, objectParent).All(context.Background())
			objectProof(t, e, 200, []byte(raw))
			if rows != nil {
				t.Fatal(rows, e)
			}
		})
	}
	t.Run("parallel lazy reused iterator snapshots", func(t *testing.T) {
		var calls, callbacks atomic.Int32
		headers := map[string]string{"X-Option": "snapshot"}
		opts := []objects.ListOption{objects.WithListOpts(objects.ListOpts{Headers: headers, MaxItems: 1}), func(*objects.ListOpts) error { callbacks.Add(1); return nil }}
		client := objectClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Option") != "snapshot" || r.URL.RawQuery != "" {
				t.Error(r.Header, r.URL)
			}
			return objectWire(200, io.NopCloser(strings.NewReader(`{"objects":[{},null],"next":42}`))), nil
		})
		scope := objectScope(t, client, objectParent)
		seq := scope.List(context.Background(), opts...)
		opts[0] = nil
		headers["X-Option"] = "caller"
		if calls.Load() != 0 || callbacks.Load() != 0 {
			t.Fatal("eager iterator")
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
	for _, mode := range []string{"context", "provider", "Endpoint"} {
		t.Run("between consumed rows "+mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("row cause")
			raw := `{"objects":[{},{}]}`
			var calls atomic.Int32
			client := objectClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return objectWire(200, io.NopCloser(strings.NewReader(raw))), nil
			})
			scope := objectScope(t, client, objectParent)
			n, terminal := 0, 0
			for v, e := range scope.List(ctx) {
				if e != nil {
					objectProof(t, e, 200, []byte(raw))
					terminal++
					if v != nil || mode == "context" && (!errors.Is(e, cause) || !errors.Is(e, context.Canceled)) || mode != "context" && !errors.Is(e, resource.ErrInvalidOption) {
						t.Fatal(v, e)
					}
					continue
				}
				n++
				switch mode {
				case "context":
					cancel(cause)
				case "provider":
					client.ProviderClient = &gophercloud.ProviderClient{}
				case "Endpoint":
					client.Endpoint = "https://glance.invalid/replaced/v2/"
				}
			}
			if n != 1 || terminal != 1 || calls.Load() != 1 {
				t.Fatal(n, terminal, calls.Load())
			}
		})
	}
}

func TestMetadefObjectsOwnedPreparationAndSource(t *testing.T) {
	for _, mode := range []string{"nil context", "cancelled context", "nil API", "nil client", "nil provider", "wrong type", "bad base", "protected source header"} {
		t.Run("scope "+mode, func(t *testing.T) {
			var calls atomic.Int32
			client := objectClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
			api := objects.New(client)
			ctx := context.Background()
			cause := errors.New("scope cancel")
			switch mode {
			case "nil context":
				ctx = nil
			case "cancelled context":
				c, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx = c
			case "nil API":
				api = nil
			case "nil client":
				api = objects.New(nil)
			case "nil provider":
				client.ProviderClient = nil
			case "wrong type":
				client.Type = "compute"
			case "bad base":
				client.ResourceBase = "https://user:secret@glance.invalid/v2/"
			case "protected source header":
				client.MoreHeaders = map[string]string{"X-Auth-Token": "forged"}
			}
			scope, e := api.InNamespace(ctx, objectParent)
			if scope != nil || e == nil || calls.Load() != 0 {
				t.Fatal(scope, e, calls.Load())
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
	invalidNames := []string{"", ".", "..", "bad/name", "bad\\name", "literal%2F", "query?name", "fragment#name", "line\nname", "delete\x7fname", strings.Repeat("界", 81), string([]byte{255})}
	for _, parent := range invalidNames {
		t.Run("parent "+parent, func(t *testing.T) {
			var calls atomic.Int32
			client := objectClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
			scope, e := objects.New(client).InNamespace(context.Background(), parent)
			if scope != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(scope, e, calls.Load())
			}
		})
	}
	for _, op := range objectOperations {
		if !op.named && op.name != "Create" {
			continue
		}
		for _, name := range invalidNames {
			t.Run(op.name+" unsafe child "+name, func(t *testing.T) {
				var calls, callbacks atomic.Int32
				client := objectClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				scope := objectScope(t, client, objectParent)
				o := objectOptions{create: []objects.CreateOption{func(*objects.CreateOpts) error { callbacks.Add(1); return nil }}, get: []objects.GetOption{func(*objects.GetOpts) error { callbacks.Add(1); return nil }}, update: []objects.UpdateOption{func(*objects.UpdateOpts) error { callbacks.Add(1); return nil }}, deletion: []objects.DeleteOption{func(*objects.DeleteOpts) error { callbacks.Add(1); return nil }}}
				v, e := objectCall(scope, context.Background(), op.name, name, o)
				if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 || callbacks.Load() != 0 {
					t.Fatal(v, e, calls.Load(), callbacks.Load())
				}
			})
		}
	}
	for _, op := range objectOperations {
		for _, mode := range []string{"nil context", "cancelled context", "nil scope", "provider", "Endpoint", "base", "type", "Microversion", "protected source"} {
			t.Run(op.name+" lifetime "+mode, func(t *testing.T) {
				var calls, callbacks atomic.Int32
				client := objectClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				scope := objectScope(t, client, objectParent)
				ctx := context.Background()
				cause := errors.New("operation cause")
				switch mode {
				case "nil context":
					ctx = nil
				case "cancelled context":
					c, cancel := context.WithCancelCause(ctx)
					cancel(cause)
					ctx = c
				case "nil scope":
					scope = nil
				case "provider":
					client.ProviderClient = &gophercloud.ProviderClient{}
				case "Endpoint":
					client.Endpoint = "https://glance.invalid/replaced/v2/"
				case "base":
					client.ResourceBase = "https://glance.invalid/replaced/v2/"
				case "type":
					client.Type = "compute"
				case "Microversion":
					client.Microversion = "2.3"
				case "protected source":
					client.MoreHeaders = map[string]string{"X-Auth-Token": "forged"}
				}
				o := objectOptions{create: []objects.CreateOption{func(*objects.CreateOpts) error { callbacks.Add(1); return nil }}, get: []objects.GetOption{func(*objects.GetOpts) error { callbacks.Add(1); return nil }}, update: []objects.UpdateOption{func(*objects.UpdateOpts) error { callbacks.Add(1); return nil }}, deletion: []objects.DeleteOption{func(*objects.DeleteOpts) error { callbacks.Add(1); return nil }}, deleteAll: []objects.DeleteAllOption{func(*objects.DeleteAllOpts) error { callbacks.Add(1); return nil }}, list: []objects.ListOption{func(*objects.ListOpts) error { callbacks.Add(1); return nil }}}
				v, e := objectCall(scope, ctx, op.name, objectName, o)
				if v != nil || e == nil || calls.Load() != 0 || callbacks.Load() != 0 {
					t.Fatal(v, e, calls.Load(), callbacks.Load())
				}
				if mode == "cancelled context" {
					if !errors.Is(e, cause) || !errors.Is(e, context.Canceled) {
						t.Fatal(e)
					}
				} else if !errors.Is(e, resource.ErrInvalidOption) {
					t.Fatal(e)
				}
			})
		}
	}
	for _, op := range objectOperations {
		for _, header := range []string{"X-Auth-Token", "Content-Type", "Accept", "Content-Length", "Transfer-Encoding", "Host", "OpenStack-API-Version", "Bad Header"} {
			t.Run(op.name+" header "+header, func(t *testing.T) {
				var calls atomic.Int32
				client := objectClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				o := objectOptions{create: []objects.CreateOption{objects.WithCreateHeader(header, "value")}, get: []objects.GetOption{objects.WithGetHeader(header, "value")}, update: []objects.UpdateOption{objects.WithUpdateHeader(header, "value")}, deletion: []objects.DeleteOption{objects.WithDeleteHeader(header, "value")}, deleteAll: []objects.DeleteAllOption{objects.WithDeleteAllHeader(header, "value")}, list: []objects.ListOption{objects.WithListHeader(header, "value")}}
				v, e := objectCall(objectScope(t, client, objectParent), context.Background(), op.name, objectName, o)
				if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
					t.Fatal(v, e, calls.Load())
				}
			})
		}
	}
	for _, op := range objectOperations {
		for _, mode := range []string{"nil callback", "callback cause", "callback cancel", "provider", "Endpoint", "base", "type", "Microversion", "protected source"} {
			t.Run(op.name+" callback "+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("callback cause")
				var calls atomic.Int32
				client := objectClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				scope := objectScope(t, client, objectParent)
				mutate := func() error {
					switch mode {
					case "callback cause":
						return cause
					case "callback cancel":
						cancel(cause)
					case "provider":
						client.ProviderClient = &gophercloud.ProviderClient{}
					case "Endpoint":
						client.Endpoint = "https://glance.invalid/replaced/v2/"
					case "base":
						client.ResourceBase = "https://glance.invalid/replaced/v2/"
					case "type":
						client.Type = "compute"
					case "Microversion":
						client.Microversion = "2.3"
					case "protected source":
						client.MoreHeaders = map[string]string{"X-Auth-Token": "forged"}
					}
					return nil
				}
				o := objectOptions{create: []objects.CreateOption{func(*objects.CreateOpts) error { return mutate() }}, get: []objects.GetOption{func(*objects.GetOpts) error { return mutate() }}, update: []objects.UpdateOption{func(*objects.UpdateOpts) error { return mutate() }}, deletion: []objects.DeleteOption{func(*objects.DeleteOpts) error { return mutate() }}, deleteAll: []objects.DeleteAllOption{func(*objects.DeleteAllOpts) error { return mutate() }}, list: []objects.ListOption{func(*objects.ListOpts) error { return mutate() }}}
				if mode == "nil callback" {
					o = objectOptions{create: []objects.CreateOption{nil}, get: []objects.GetOption{nil}, update: []objects.UpdateOption{nil}, deletion: []objects.DeleteOption{nil}, deleteAll: []objects.DeleteAllOption{nil}, list: []objects.ListOption{nil}}
				}
				v, e := objectCall(scope, ctx, op.name, objectName, o)
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
	for _, op := range []string{"Create", "Update"} {
		for _, raw := range []json.RawMessage{nil, json.RawMessage(`null`), json.RawMessage(`[]`), json.RawMessage(`1`), json.RawMessage(`false`), json.RawMessage(`"text"`), json.RawMessage(`{"value":`), json.RawMessage("{\"value\":\"\xff\"}")} {
			t.Run(op+" invalid request property "+string(raw), func(t *testing.T) {
				var calls atomic.Int32
				client := objectClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
				o := objectOptions{create: []objects.CreateOption{objects.WithCreateProperties(map[string]json.RawMessage{"key": raw})}, update: []objects.UpdateOption{objects.WithUpdateProperties(map[string]json.RawMessage{"key": raw})}}
				v, e := objectCall(objectScope(t, client, objectParent), context.Background(), op, objectName, o)
				if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
					t.Fatal(v, e, calls.Load())
				}
			})
		}
	}
	invalidOpts := []struct {
		name, op string
		opts     objectOptions
	}{
		{"Description UTF8", "Create", objectOptions{create: []objects.CreateOption{objects.WithCreateDescription(string([]byte{255}))}}},
		{"Property key UTF8", "Create", objectOptions{create: []objects.CreateOption{objects.WithCreateProperties(map[string]json.RawMessage{string([]byte{255}): json.RawMessage(`{}`)})}}},
		{"Required UTF8", "Update", objectOptions{update: []objects.UpdateOption{objects.WithUpdateRequired([]string{string([]byte{255})})}}},
		{"unsafe rename", "Update", objectOptions{update: []objects.UpdateOption{objects.WithUpdateName("bad/name")}}},
		{"negative local cap", "All", objectOptions{list: []objects.ListOption{objects.WithListMaxItems(-1)}}},
	}
	for _, test := range invalidOpts {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			client := objectClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
			v, e := objectCall(objectScope(t, client, objectParent), context.Background(), test.op, objectName, test.opts)
			if v != nil || !errors.Is(e, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(v, e, calls.Load())
			}
		})
	}
	for _, op := range objectOperations {
		t.Run(op.name+" full replacement deep callback ownership", func(t *testing.T) {
			input := map[string]string{"X-Owned": "before"}
			text := "before"
			flag := true
			properties := map[string]json.RawMessage{"key": json.RawMessage(`{"value":1}`)}
			required := []string{"key"}
			var escapedHeaders map[string]string
			var escapedText *string
			var escapedFlag *bool
			var escapedProperties map[string]json.RawMessage
			var escapedRequired []string
			var callbacks atomic.Int32
			mutate := func() {
				callbacks.Add(1)
				escapedHeaders["X-Owned"] = "escaped"
				if escapedText != nil {
					*escapedText = "escaped"
				}
				if escapedFlag != nil {
					*escapedFlag = false
				}
				if escapedProperties != nil {
					escapedProperties["key"][len(escapedProperties["key"])-2] = '2'
				}
				if escapedRequired != nil {
					escapedRequired[0] = "escaped"
				}
			}
			o := objectOptions{
				create: []objects.CreateOption{objects.WithCreateHeader("X-Dropped", "gone"), objects.WithCreateOpts(objects.CreateOpts{Headers: input, Description: &text, Properties: properties, Required: required}), func(c *objects.CreateOpts) error {
					escapedHeaders = c.Headers
					escapedText = c.Description
					escapedProperties = c.Properties
					escapedRequired = c.Required
					return nil
				}, func(*objects.CreateOpts) error { mutate(); return nil }},
				update: []objects.UpdateOption{objects.WithUpdateHeader("X-Dropped", "gone"), objects.WithUpdateOpts(objects.UpdateOpts{Headers: input, Description: &text, Properties: properties, Required: required}), func(c *objects.UpdateOpts) error {
					escapedHeaders = c.Headers
					escapedText = c.Description
					escapedProperties = c.Properties
					escapedRequired = c.Required
					return nil
				}, func(*objects.UpdateOpts) error { mutate(); return nil }},
				get: []objects.GetOption{objects.WithGetHeader("X-Dropped", "gone"), objects.WithGetOpts(objects.GetOpts{Headers: input}), func(c *objects.GetOpts) error { escapedHeaders = c.Headers; return nil }, func(*objects.GetOpts) error { mutate(); return nil }},
				deletion: []objects.DeleteOption{objects.WithDeleteHeader("X-Dropped", "gone"), objects.WithDeleteOpts(objects.DeleteOpts{Headers: input, IgnoreMissing: &flag}), func(c *objects.DeleteOpts) error {
					escapedHeaders = c.Headers
					escapedFlag = c.IgnoreMissing
					return nil
				}, func(*objects.DeleteOpts) error { mutate(); return nil }},
				deleteAll: []objects.DeleteAllOption{objects.WithDeleteAllHeader("X-Dropped", "gone"), objects.WithDeleteAllOpts(objects.DeleteAllOpts{Headers: input}), func(c *objects.DeleteAllOpts) error { escapedHeaders = c.Headers; return nil }, func(*objects.DeleteAllOpts) error { mutate(); return nil }},
				list:      []objects.ListOption{objects.WithListHeader("X-Dropped", "gone"), objects.WithListOpts(objects.ListOpts{Headers: input, MaxItems: 1}), func(c *objects.ListOpts) error { escapedHeaders = c.Headers; return nil }, func(*objects.ListOpts) error { mutate(); return nil }},
			}
			input["X-Owned"] = "caller"
			text = "caller"
			flag = false
			properties["key"][len(properties["key"])-2] = '3'
			required[0] = "caller"
			var calls atomic.Int32
			client := objectClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Header.Get("X-Owned") != "before" || r.Header.Get("X-Dropped") != "" || r.URL.RawQuery != "" {
					t.Error(r.Header, r.URL)
				}
				if op.name == "Create" || op.name == "Update" {
					raw, _ := io.ReadAll(r.Body)
					var payload map[string]json.RawMessage
					_ = json.Unmarshal(raw, &payload)
					if string(payload["description"]) != `"before"` || string(payload["properties"]) != `{"key":{"value":1}}` || string(payload["required"]) != `["key"]` {
						t.Error(string(raw))
					}
				}
				code, raw := op.status, op.raw
				if op.name == "Delete" {
					code, raw = 404, "physical404"
				}
				return objectWire(code, io.NopCloser(strings.NewReader(raw))), nil
			})
			v, e := objectCall(objectScope(t, client, objectParent), context.Background(), op.name, objectName, o)
			if e != nil || calls.Load() != 1 || callbacks.Load() != 1 || op.name == "Delete" && v != nil || op.name != "Delete" && v == nil {
				t.Fatal(v, e, calls.Load(), callbacks.Load())
			}
		})
	}
	t.Run("latest headers per request captured before callbacks and live auth", func(t *testing.T) {
		var calls atomic.Int32
		client := objectClient(nil)
		client.MoreHeaders = map[string]string{"X-Source": "at-scope"}
		scope := objectScope(t, client, objectParent)
		client.MoreHeaders["X-Source"] = "first"
		client.SetToken("first-token")
		client.HTTPClient.Transport = objectTransport(func(r *http.Request) (*http.Response, error) {
			n := calls.Add(1)
			headers, tokens := []string{"first", "second", "third"}, []string{"first-token", "second-token", "third-token"}
			if n > 3 || r.Header.Get("X-Source") != headers[n-1] || r.Header.Get("X-Auth-Token") != tokens[n-1] || r.URL.String() != objectBase+objectCollection+"/"+url.PathEscape(objectName) {
				t.Error(r.URL, r.Header)
			}
			return objectWire(200, io.NopCloser(strings.NewReader(objectJSON))), nil
		})
		v, e := scope.Get(context.Background(), objectName)
		if v == nil || e != nil {
			t.Fatal(v, e)
		}
		client.MoreHeaders = map[string]string{"X-Source": "second"}
		client.SetToken("second-token")
		v, e = scope.Get(context.Background(), objectName, func(*objects.GetOpts) error { client.MoreHeaders["X-Source"] = "third"; return nil })
		if v == nil || e != nil {
			t.Fatal(v, e)
		}
		client.SetToken("third-token")
		v, e = scope.Get(context.Background(), objectName)
		if v == nil || e != nil || calls.Load() != 3 {
			t.Fatal(v, e, calls.Load())
		}
	})
}

func TestMetadefObjectsResponseOwnershipAndNativeHooks(t *testing.T) {
	for _, op := range objectOperations {
		for _, mode := range []string{"Read", "Close", "cancel", "Read Close cancel", "source after Close"} {
			t.Run(op.name+" accepted "+mode, func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("accepted Read"), errors.New("accepted Close"), errors.New("accepted cancel")
				raw := []byte(op.raw)
				if op.name == "Delete" || op.name == "DeleteAll" {
					raw = []byte{'a', 'c', 'k', 255}
				}
				b := &objectBody{Reader: bytes.NewReader(raw)}
				if strings.Contains(mode, "Read") {
					b.Reader = objectReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
				}
				if strings.Contains(mode, "Close") {
					b.closeErr = closeCause
				}
				var calls, hooks atomic.Int32
				client := objectClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return objectWire(op.status, b), nil })
				scope := objectScope(t, client, objectParent)
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
				v, e := objectCall(scope, ctx, op.name, objectName, objectOptions{})
				proof := objectProof(t, e, op.status, raw)
				if calls.Load() != 1 || hooks.Load() != 0 || b.closes.Load() != 1 || strings.Contains(mode, "Read") && !errors.Is(e, readCause) || mode != "source after Close" && strings.Contains(mode, "Close") && !errors.Is(e, closeCause) || strings.Contains(mode, "cancel") && (!errors.Is(e, cancelCause) || !errors.Is(e, context.Canceled)) || mode == "source after Close" && !errors.Is(e, resource.ErrInvalidOption) {
					t.Fatal(v, e, calls.Load(), hooks.Load(), b.closes.Load())
				}
				if op.name == "Delete" || op.name == "DeleteAll" {
					if v == nil || v.ack == nil || v.ack.Namespace != objectParent || v.ack.StatusCode != 204 || !bytes.Equal(v.ack.Body, raw) || op.name == "Delete" && (v.ack.Name == nil || *v.ack.Name != objectName) || op.name == "DeleteAll" && v.ack.Name != nil {
						t.Fatal(v, e)
					}
					proof.Body[0] = '!'
					proof.Header.Set("X-Request-Id", "proof changed")
					if v.ack.Body[0] != 'a' || v.ack.Header.Get("X-Request-Id") != "actual-object" {
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
			b := &objectBody{Reader: bytes.NewReader(raw)}
			wire := objectWire(204, b)
			var calls atomic.Int32
			client := objectClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Method != "DELETE" || r.Body != nil || r.URL.RawQuery != "" {
					t.Error(r.Method, r.URL, r.Body)
				}
				return wire, nil
			})
			v, e := objectCall(objectScope(t, client, objectParent), context.Background(), op, objectName, objectOptions{})
			if e != nil || v == nil || v.ack == nil || !bytes.Equal(v.ack.Body, raw) || v.ack.StatusCode != 204 || b.closes.Load() != 1 || calls.Load() != 1 {
				t.Fatal(v, e, b.closes.Load(), calls.Load())
			}
			wire.Header.Set("X-Request-Id", "late")
			raw[0] = 1
			if v.ack.Header.Get("X-Request-Id") != "actual-object" || v.ack.Body[0] != 255 {
				t.Fatal("borrowed ack alias", v.ack)
			}
		})
	}
	for _, op := range objectOperations {
		for _, code := range []int{202, 404, 409} {
			if op.name == "Delete" && code == 404 {
				continue
			}
			t.Run(fmt.Sprintf("%s strict%d", op.name, code), func(t *testing.T) {
				raw := []byte("server-owned rejection")
				b := &objectBody{Reader: bytes.NewReader(raw)}
				var calls atomic.Int32
				client := objectClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return objectWire(code, b), nil })
				v, e := objectCall(objectScope(t, client, objectParent), context.Background(), op.name, objectName, objectOptions{})
				var native gophercloud.ErrUnexpectedResponseCode
				if v != nil || !errors.As(e, &native) || native.Actual != code || !bytes.Equal(native.Body, raw) || native.ResponseHeader.Get("X-Request-Id") != "actual-object" || calls.Load() != 1 || b.closes.Load() != 1 {
					t.Fatal(v, e, native, calls.Load(), b.closes.Load())
				}
			})
		}
	}
	for _, mode := range []string{"default", "explicit true", "explicit false", "DeleteAll strict"} {
		t.Run("physical404 "+mode, func(t *testing.T) {
			var calls, hooks atomic.Int32
			hookCause := errors.New("strict callback")
			b := &objectBody{Reader: strings.NewReader("actual404")}
			client := objectClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return objectWire(404, b), nil })
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				hooks.Add(1)
				return hookCause
			}
			opts := objectOptions{}
			op := "Delete"
			if mode == "DeleteAll strict" {
				op = "DeleteAll"
			}
			if mode == "explicit true" || mode == "explicit false" {
				opts.deletion = []objects.DeleteOption{objects.WithDeleteIgnoreMissing(mode == "explicit true")}
			}
			v, e := objectCall(objectScope(t, client, objectParent), context.Background(), op, objectName, opts)
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
			b := &objectBody{Reader: bytes.NewReader(raw)}
			if strings.Contains(mode, "Read") {
				b.Reader = objectReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
			}
			if strings.Contains(mode, "Close") {
				b.closeErr = closeCause
			}
			var calls, hooks atomic.Int32
			client := objectClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return objectWire(404, b), nil })
			scope := objectScope(t, client, objectParent)
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
			v, e := scope.Delete(ctx, objectName)
			objectProof(t, e, 404, raw)
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
			b := &objectBody{Reader: strings.NewReader("original failure")}
			client := objectClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				if mode == "transport nested404" {
					return nil, errors.Join(cause, nested)
				}
				code := 503
				if mode == "reauth nested404" {
					code = 401
				}
				return objectWire(code, b), nil
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
			v, e := objectScope(t, client, objectParent).Delete(context.Background(), objectName)
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
		var bodies []*objectBody
		transportCause := errors.New("prebody transport")
		client := objectClient(nil)
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
			if !o.KeepResponseBody || o.JSONResponse != nil || o.RawBody != nil || string(raw) != `{"name":"CPU Limits"}` {
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
		scope := objectScope(t, client, objectParent)
		client.HTTPClient.Transport = objectTransport(func(r *http.Request) (*http.Response, error) {
			i := int(calls.Add(1)) - 1
			codes := []int{401, 429, 503, 0, 201}
			tokens := []string{"initial", "reauth", "backoff", "retry503", "retrytransport"}
			if i >= len(codes) {
				return nil, errors.New("replay")
			}
			raw, _ := io.ReadAll(r.Body)
			if r.Method != "POST" || r.URL.String() != objectBase+objectCollection || string(raw) != `{"name":"CPU Limits"}` || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != tokens[i] {
				t.Error(r.Method, r.URL, string(raw), r.Header)
			}
			if codes[i] == 0 {
				return nil, transportCause
			}
			b := &objectBody{Reader: strings.NewReader(objectJSON)}
			bodies = append(bodies, b)
			return objectWire(codes[i], b), nil
		})
		v, e := scope.Create(context.Background(), objectName)
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
			b := &objectBody{Reader: strings.NewReader("original503")}
			borrowed := &objectBody{Reader: objectReader(func([]byte) (int, error) { borrowedReads.Add(1); return 0, io.EOF })}
			client := objectClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				raw, _ := io.ReadAll(r.Body)
				if string(raw) != `{"name":"CPU Limits"}` {
					t.Error(string(raw))
				}
				return objectWire(503, b), nil
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
			v, e := objectScope(t, client, objectParent).Create(context.Background(), objectName)
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
		b := &objectBody{Reader: strings.NewReader("original503")}
		client := objectClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Body != nil {
				t.Error(r.Body)
			}
			return objectWire(503, b), nil
		})
		client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			o.JSONBody = json.RawMessage("null")
			return nil
		}
		v, e := objectScope(t, client, objectParent).DeleteAll(context.Background())
		if v != nil || !errors.Is(e, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(e, 503) || calls.Load() != 1 || b.closes.Load() != 1 {
			t.Fatal(v, e, calls.Load(), b.closes.Load())
		}
	})
	for _, op := range []string{"Create", "Update"} {
		t.Run(op+" same serialized replacement", func(t *testing.T) {
			var calls, hooks atomic.Int32
			var bodies []*objectBody
			code := 201
			if op == "Update" {
				code = 200
			}
			client := objectClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				raw, _ := io.ReadAll(r.Body)
				if string(raw) != `{"name":"CPU Limits"}` {
					t.Error(string(raw))
				}
				status, reply := 503, "original503"
				if n == 2 {
					status, reply = code, objectJSON
				}
				b := &objectBody{Reader: strings.NewReader(reply)}
				bodies = append(bodies, b)
				return objectWire(status, b), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				hooks.Add(1)
				o.JSONBody = map[string]string{"name": objectName}
				return nil
			}
			v, e := objectCall(objectScope(t, client, objectParent), context.Background(), op, objectName, objectOptions{})
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
	for _, op := range objectOperations {
		t.Run(op.name+" expanded codes reject actual response", func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("unexpected Read"), errors.New("unexpected Close"), errors.New("unexpected cause")
			var calls, hooks atomic.Int32
			var bodies []*objectBody
			client := objectClient(func(*http.Request) (*http.Response, error) {
				n := calls.Add(1)
				code, raw := 503, "original503"
				if n == 2 {
					code, raw = 202, "unexpected actual202"
				}
				b := &objectBody{Reader: strings.NewReader(raw)}
				if n == 2 {
					b.Reader = objectReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
					b.closeErr = closeCause
					b.onClose = func() { cancel(cancelCause) }
				}
				bodies = append(bodies, b)
				return objectWire(code, b), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, e error, _ uint) error {
				hooks.Add(1)
				if !gophercloud.ResponseCodeIs(e, 503) {
					return e
				}
				o.OkCodes = []int{202}
				return nil
			}
			v, e := objectCall(objectScope(t, client, objectParent), ctx, op.name, objectName, objectOptions{})
			var native gophercloud.ErrUnexpectedResponseCode
			var proof *resource.ResponseError
			expected := []int{op.status}
			if op.name == "Delete" {
				expected = append(expected, 404)
			}
			target := objectBase + objectCollection
			if op.named {
				target += "/" + url.PathEscape(objectName)
			}
			if v != nil || !errors.As(e, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, expected) || native.Method != op.method || native.URL != target || string(native.Body) != "unexpected actual202" || native.ResponseHeader.Get("X-Request-Id") != "actual-object" || !errors.Is(e, readCause) || !errors.Is(e, closeCause) || !errors.Is(e, cancelCause) || !errors.Is(e, context.Canceled) || errors.As(e, &proof) || calls.Load() != 2 || hooks.Load() != 1 {
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
			var bodies []*objectBody
			client := objectClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				b := &objectBody{Reader: strings.NewReader(objectJSON)}
				bodies = append(bodies, b)
				if n == 2 {
					raw, _ := io.ReadAll(r.Body)
					if r.Method != "POST" || string(raw) != `{"name":"CPU Limits"}` {
						t.Error(r.Method, string(raw))
					}
					return objectWire(201, b), nil
				}
				code, target := 307, objectBase+objectCollection
				switch redirect {
				case "foreign origin":
					target = "https://foreign.invalid/object"
				case "changed path":
					target = objectBase + "metadefs/namespaces/other/objects"
				case "changed query":
					target += "?marker=other"
				case "changed method":
					code = 303
				}
				wire := objectWire(code, b)
				wire.Header.Set("Location", target)
				return wire, nil
			})
			client.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { redirects.Add(1); return nil }
			v, e := objectScope(t, client, objectParent).Create(context.Background(), objectName)
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
