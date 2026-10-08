package metadefnamespaces_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	namespaces "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func nestedString(value string) *string { return &value }

func nestedJSON(t *testing.T, raw []byte) map[string]json.RawMessage {
	t.Helper()
	var value map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		t.Fatalf("request object %q: %v", raw, err)
	}
	return value
}

func nestedEqualJSON(t *testing.T, raw json.RawMessage, want any) {
	t.Helper()
	expected, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var gotValue, wantValue any
	gotDecoder := json.NewDecoder(bytes.NewReader(raw))
	gotDecoder.UseNumber()
	wantDecoder := json.NewDecoder(bytes.NewReader(expected))
	wantDecoder.UseNumber()
	if gotDecoder.Decode(&gotValue) != nil || wantDecoder.Decode(&wantValue) != nil || !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("JSON got %s want %s", raw, expected)
	}
}

func nestedDefinition() namespaces.PropertyDefinition {
	return namespaces.PropertyDefinition{Type: "number", Title: "", Description: nestedString("line one\nline two"), Attributes: map[string]json.RawMessage{
		"minimum":     json.RawMessage(`9007199254740993`),
		"default":     json.RawMessage(`null`),
		"x-extension": json.RawMessage(`{"literal":[true,"value",9007199254740995]}`),
	}}
}

func nestedOptions() namespaces.CreateOpts {
	return namespaces.CreateOpts{
		Headers: map[string]string{"X-Option": "owned"}, Protected: new(bool),
		Properties:               map[string]namespaces.PropertyDefinition{"root/property\n": nestedDefinition()},
		Objects:                  []namespaces.ObjectDefinition{{Name: "../object?#\n", Description: nestedString(""), Properties: map[string]namespaces.PropertyDefinition{"": nestedDefinition()}, Required: []string{"", "absent", "absent", "comma,value\n"}}},
		Tags:                     []namespaces.TagDefinition{{Name: ""}, {Name: "tag/one\n"}, {Name: "tag/one\n"}},
		ResourceTypeAssociations: []namespaces.ResourceTypeAssociationDefinition{{Name: "OS::Missing::Type", Prefix: nestedString(""), PropertiesTarget: nestedString("target/field\n")}},
	}
}

func nestedCapture(t *testing.T, options []namespaces.CreateOption, check func(map[string]json.RawMessage)) {
	t.Helper()
	var calls atomic.Int32
	client := namespaceClient(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		if r.Method != http.MethodPost || r.URL.String() != namespaceBase+namespaceCollection {
			t.Error("nested request retargeted", r.Method, r.URL)
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		check(nestedJSON(t, raw))
		return namespaceWire(201, &namespaceBody{Reader: strings.NewReader(namespaceObject)}), nil
	})
	value, err := namespaces.New(client).Create(context.Background(), namespaceName, options...)
	if err != nil || value == nil || calls.Load() != 1 {
		t.Fatal(value, err, calls.Load())
	}
}

func TestMetadefNamespaceNestedCreateWireAndPresence(t *testing.T) {
	t.Run("four containers one fixed POST", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := cloud.Client("image", "/catalog/not-used/")
		client.ResourceBase = cloud.Server.URL + namespacePrefix
		client.MoreHeaders = map[string]string{"X-Source": "source"}
		client.Microversion = "2.2"
		cloud.Provider.SetToken("current-token")
		config := nestedOptions()
		var calls atomic.Int32
		cloud.Mux.HandleFunc(namespacePrefix, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.Method != "POST" || r.URL.Path != namespacePrefix+namespaceCollection || r.URL.RawQuery != "" || r.Header.Get("X-Source") != "source" || r.Header.Get("X-Option") != "owned" || r.Header.Get("X-Auth-Token") != "current-token" || r.Header.Get("OpenStack-API-Version") != "image 2.2" {
				t.Error(r.Method, r.URL, r.Header)
			}
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			body := nestedJSON(t, raw)
			if len(body) != 6 {
				t.Error("extra or missing root fields", string(raw))
			}
			nestedEqualJSON(t, body["namespace"], namespaceName)
			nestedEqualJSON(t, body["protected"], false)
			nestedEqualJSON(t, body["properties"], map[string]any{"root/property\n": map[string]any{"type": "number", "title": "", "description": "line one\nline two", "minimum": json.Number("9007199254740993"), "default": nil, "x-extension": map[string]any{"literal": []any{true, "value", json.Number("9007199254740995")}}}})
			nestedEqualJSON(t, body["objects"], []any{map[string]any{"name": "../object?#\n", "description": "", "required": []string{"", "absent", "absent", "comma,value\n"}, "properties": map[string]any{"": map[string]any{"type": "number", "title": "", "description": "line one\nline two", "minimum": json.Number("9007199254740993"), "default": nil, "x-extension": map[string]any{"literal": []any{true, "value", json.Number("9007199254740995")}}}}}})
			nestedEqualJSON(t, body["tags"], []map[string]string{{"name": ""}, {"name": "tag/one\n"}, {"name": "tag/one\n"}})
			nestedEqualJSON(t, body["resource_type_associations"], []map[string]string{{"name": "OS::Missing::Type", "prefix": "", "properties_target": "target/field\n"}})
			w.Header().Set("Location", "https://passive.invalid/no-follow-up")
			testcloud.JSON(w, 201, namespaceObject)
		})
		value, err := namespaces.New(client).Create(context.Background(), namespaceName, namespaces.WithCreateOpts(config))
		if err != nil || value == nil || calls.Load() != 1 || value.Header.Get("Location") != "https://passive.invalid/no-follow-up" {
			t.Fatal(value, err, calls.Load())
		}
	})
	t.Run("nil versus explicit empty", func(t *testing.T) {
		for _, empty := range []bool{false, true} {
			t.Run(fmt.Sprint(empty), func(t *testing.T) {
				config := namespaces.CreateOpts{}
				if empty {
					config.Properties = map[string]namespaces.PropertyDefinition{}
					config.Objects = []namespaces.ObjectDefinition{}
					config.Tags = []namespaces.TagDefinition{}
					config.ResourceTypeAssociations = []namespaces.ResourceTypeAssociationDefinition{}
				}
				nestedCapture(t, []namespaces.CreateOption{namespaces.WithCreateOpts(config)}, func(body map[string]json.RawMessage) {
					for _, key := range []string{"properties", "objects", "tags", "resource_type_associations"} {
						raw, exists := body[key]
						if exists != empty {
							t.Errorf("%s presence %v", key, exists)
						}
						if empty {
							want := "[]"
							if key == "properties" {
								want = "{}"
							}
							if string(raw) != want {
								t.Errorf("%s got%s", key, raw)
							}
						}
					}
				})
			})
		}
	})
	t.Run("object optional container presence", func(t *testing.T) {
		nestedCapture(t, []namespaces.CreateOption{namespaces.WithCreateObjects([]namespaces.ObjectDefinition{{Name: ""}, {Name: "", Properties: map[string]namespaces.PropertyDefinition{}, Required: []string{}}})}, func(body map[string]json.RawMessage) {
			nestedEqualJSON(t, body["objects"], []any{map[string]any{"name": ""}, map[string]any{"name": "", "properties": map[string]any{}, "required": []any{}}})
		})
	})
	t.Run("all helpers literal body names and unchanged update", func(t *testing.T) {
		literal := "../?#%\\\n" + strings.Repeat("界", 81)
		options := []namespaces.CreateOption{
			namespaces.WithCreateProperties(map[string]namespaces.PropertyDefinition{literal: {Type: "vendor/non-schema-type", Title: ""}}),
			namespaces.WithCreateObjects([]namespaces.ObjectDefinition{{Name: literal, Required: []string{literal, literal}}}),
			namespaces.WithCreateTags([]namespaces.TagDefinition{{Name: literal}}),
			namespaces.WithCreateResourceTypeAssociations([]namespaces.ResourceTypeAssociationDefinition{{Name: literal, Prefix: &literal, PropertiesTarget: &literal}}),
		}
		nestedCapture(t, options, func(body map[string]json.RawMessage) {
			nestedEqualJSON(t, body["objects"], []map[string]any{{"name": literal, "required": []string{literal, literal}}})
			nestedEqualJSON(t, body["tags"], []map[string]string{{"name": literal}})
			nestedEqualJSON(t, body["resource_type_associations"], []map[string]string{{"name": literal, "prefix": literal, "properties_target": literal}})
		})
		var calls atomic.Int32
		client := namespaceClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			raw, _ := io.ReadAll(r.Body)
			if r.Method != "PUT" || r.URL.Path != namespacePrefix+namespaceCollection+"/"+namespaceName || string(raw) != `{"namespace":"OS::Compute::Libvirt"}` {
				t.Error(r.Method, r.URL, string(raw))
			}
			return namespaceWire(200, &namespaceBody{Reader: strings.NewReader(namespaceObject)}), nil
		})
		if value, err := namespaces.New(client).Update(context.Background(), namespaceName); value == nil || err != nil || calls.Load() != 1 {
			t.Fatal(value, err, calls.Load())
		}
	})
}

func TestMetadefNamespaceNestedDefinitionsAndRawSchema(t *testing.T) {
	invalidUTF8 := string([]byte{0xff})
	invalids := []struct {
		name   string
		change func(*namespaces.CreateOpts)
	}{
		{"empty property Type", func(v *namespaces.CreateOpts) {
			p := v.Properties["root/property\n"]
			p.Type = ""
			v.Properties["root/property\n"] = p
		}},
		{"property Type UTF8", func(v *namespaces.CreateOpts) {
			p := v.Properties["root/property\n"]
			p.Type = invalidUTF8
			v.Properties["root/property\n"] = p
		}},
		{"property Title UTF8", func(v *namespaces.CreateOpts) {
			p := v.Properties["root/property\n"]
			p.Title = invalidUTF8
			v.Properties["root/property\n"] = p
		}},
		{"property Description UTF8", func(v *namespaces.CreateOpts) {
			p := v.Properties["root/property\n"]
			p.Description = &invalidUTF8
			v.Properties["root/property\n"] = p
		}},
		{"dictionary key UTF8", func(v *namespaces.CreateOpts) {
			v.Properties = map[string]namespaces.PropertyDefinition{invalidUTF8: nestedDefinition()}
		}},
		{"nested object property Type", func(v *namespaces.CreateOpts) {
			v.Objects[0].Properties[""] = namespaces.PropertyDefinition{Title: "title"}
		}},
		{"object Name UTF8", func(v *namespaces.CreateOpts) { v.Objects[0].Name = invalidUTF8 }},
		{"object Description UTF8", func(v *namespaces.CreateOpts) { v.Objects[0].Description = &invalidUTF8 }},
		{"Required UTF8", func(v *namespaces.CreateOpts) { v.Objects[0].Required = []string{invalidUTF8} }},
		{"tag UTF8", func(v *namespaces.CreateOpts) { v.Tags[0].Name = invalidUTF8 }},
		{"association Name UTF8", func(v *namespaces.CreateOpts) { v.ResourceTypeAssociations[0].Name = invalidUTF8 }},
		{"prefix UTF8", func(v *namespaces.CreateOpts) { v.ResourceTypeAssociations[0].Prefix = &invalidUTF8 }},
		{"target UTF8", func(v *namespaces.CreateOpts) { v.ResourceTypeAssociations[0].PropertiesTarget = &invalidUTF8 }},
		{"attribute key UTF8 before marshal", func(v *namespaces.CreateOpts) {
			p := v.Properties["root/property\n"]
			p.Attributes = map[string]json.RawMessage{invalidUTF8: json.RawMessage(`1`)}
			v.Properties["root/property\n"] = p
		}},
		{"attribute raw UTF8", func(v *namespaces.CreateOpts) {
			p := v.Properties["root/property\n"]
			p.Attributes["x-extension"] = json.RawMessage{'"', 0xff, '"'}
			v.Properties["root/property\n"] = p
		}},
		{"attribute invalid JSON", func(v *namespaces.CreateOpts) {
			p := v.Properties["root/property\n"]
			p.Attributes["minimum"] = json.RawMessage(`01`)
			v.Properties["root/property\n"] = p
		}},
		{"attribute nil JSON", func(v *namespaces.CreateOpts) {
			p := v.Properties["root/property\n"]
			p.Attributes["minimum"] = nil
			v.Properties["root/property\n"] = p
		}},
	}
	for _, test := range invalids {
		t.Run(test.name, func(t *testing.T) {
			config := nestedOptions()
			test.change(&config)
			var calls atomic.Int32
			client := namespaceClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
			value, err := namespaces.New(client).Create(context.Background(), namespaceName, namespaces.WithCreateOpts(config))
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(value, err, calls.Load())
			}
		})
	}
	for _, key := range []string{"name", "type", "title", "description", "self", "schema", "created_at", "updated_at", "namespace_name"} {
		t.Run("reserved "+key, func(t *testing.T) {
			config := nestedOptions()
			p := config.Properties["root/property\n"]
			p.Attributes[key] = json.RawMessage(`"shadow"`)
			config.Properties["root/property\n"] = p
			var calls atomic.Int32
			client := namespaceClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
			value, err := namespaces.New(client).Create(context.Background(), namespaceName, namespaces.WithCreateOpts(config))
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(value, err, calls.Load())
			}
		})
	}
	t.Run("raw keyword values no schema validator", func(t *testing.T) {
		attributes := map[string]json.RawMessage{"minimum": json.RawMessage(`1.25`), "enum": json.RawMessage(`[1,null,false,{"x":2}]`), "required": json.RawMessage(`["absent","absent"]`), "default": json.RawMessage(`null`), "Type": json.RawMessage(`"case-decoy"`), "": json.RawMessage(`9007199254740993123456789`)}
		nestedCapture(t, []namespaces.CreateOption{namespaces.WithCreateProperties(map[string]namespaces.PropertyDefinition{"": {Type: "future/type", Title: "", Attributes: attributes}})}, func(body map[string]json.RawMessage) {
			properties := nestedJSON(t, body["properties"])
			p := nestedJSON(t, properties[""])
			if string(p["type"]) != `"future/type"` || string(p["title"]) != `""` {
				t.Fatal("canonical value fields missing", string(properties[""]))
			}
			for key, expected := range attributes {
				if !bytes.Equal(p[key], expected) {
					t.Errorf("%q raw precision changed: %s", key, p[key])
				}
			}
		})
	})
}

func TestMetadefNamespaceNestedOptionsAndCapturedSource(t *testing.T) {
	t.Run("recursive factory and callback ownership", func(t *testing.T) {
		config := nestedOptions()
		full := namespaces.WithCreateOpts(config)
		properties := namespaces.WithCreateProperties(config.Properties)
		objects := namespaces.WithCreateObjects(config.Objects)
		tags := namespaces.WithCreateTags(config.Tags)
		associations := namespaces.WithCreateResourceTypeAssociations(config.ResourceTypeAssociations)
		mutate := func() {
			p := config.Properties["root/property\n"]
			p.Attributes["minimum"][0] = '1'
			*p.Description = "changed"
			config.Properties["root/property\n"] = namespaces.PropertyDefinition{}
			config.Objects[0].Name = "changed"
			config.Objects[0].Required[0] = "changed"
			config.Objects[0].Properties[""] = namespaces.PropertyDefinition{}
			config.Tags[0].Name = "changed"
			*config.ResourceTypeAssociations[0].Prefix = "changed"
			config.ResourceTypeAssociations[0].Name = "changed"
			config.Headers["X-Option"] = "changed"
			*config.Protected = true
		}
		mutate()
		for _, options := range [][]namespaces.CreateOption{{full}, {properties, objects, tags, associations}} {
			nestedCapture(t, options, func(body map[string]json.RawMessage) {
				p := nestedJSON(t, nestedJSON(t, body["properties"])["root/property\n"])
				if string(p["minimum"]) != `9007199254740993` || string(p["description"]) != `"line one\nline two"` {
					t.Fatal("property input alias", string(body["properties"]))
				}
				nestedEqualJSON(t, body["tags"], []map[string]string{{"name": ""}, {"name": "tag/one\n"}, {"name": "tag/one\n"}})
				nestedEqualJSON(t, body["resource_type_associations"], []map[string]string{{"name": "OS::Missing::Type", "prefix": "", "properties_target": "target/field\n"}})
				var rows []map[string]json.RawMessage
				if json.Unmarshal(body["objects"], &rows) != nil || string(rows[0]["name"]) != `"../object?#\n"` || string(rows[0]["required"]) != `["","absent","absent","comma,value\n"]` {
					t.Fatal("object input alias", string(body["objects"]))
				}
			})
		}
		var retained map[string]namespaces.PropertyDefinition
		var retainedRaw json.RawMessage
		options := []namespaces.CreateOption{
			namespaces.WithCreateOpts(nestedOptions()),
			func(config *namespaces.CreateOpts) error {
				retained = config.Properties
				retainedRaw = config.Properties["root/property\n"].Attributes["minimum"]
				return nil
			},
			func(config *namespaces.CreateOpts) error {
				retainedRaw[0] = '1'
				retained["root/property\n"] = namespaces.PropertyDefinition{}
				return nil
			},
		}
		nestedCapture(t, options, func(body map[string]json.RawMessage) {
			if string(nestedJSON(t, nestedJSON(t, body["properties"])["root/property\n"])["minimum"]) != `9007199254740993` {
				t.Fatal("retained callback alias", string(body["properties"]))
			}
		})
	})
	t.Run("full replacement and last helper wins", func(t *testing.T) {
		options := []namespaces.CreateOption{namespaces.WithCreateOpts(nestedOptions()), namespaces.WithCreateOpts(namespaces.CreateOpts{}), namespaces.WithCreateTags([]namespaces.TagDefinition{{Name: "discard"}}), namespaces.WithCreateTags([]namespaces.TagDefinition{})}
		nestedCapture(t, options, func(body map[string]json.RawMessage) {
			if len(body) != 2 || string(body["tags"]) != "[]" {
				t.Fatal("replacement retained old state", body)
			}
		})
	})
	t.Run("preflight before callbacks and zero HTTP", func(t *testing.T) {
		for _, mode := range []string{"nil context", "nil provider", "wrong service", "protected source header", "protected option header", "callback error", "callback cancel"} {
			t.Run(mode, func(t *testing.T) {
				var calls, callbacks atomic.Int32
				client := namespaceClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected") })
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("callback cause")
				var contextArg context.Context = ctx
				if mode == "nil context" {
					contextArg = nil
				}
				if mode == "nil provider" {
					client.ProviderClient = nil
				}
				if mode == "wrong service" {
					client.Type = "compute"
				}
				if mode == "protected source header" {
					client.MoreHeaders = map[string]string{"X-Auth-Token": "spoof"}
				}
				options := []namespaces.CreateOption{func(config *namespaces.CreateOpts) error {
					callbacks.Add(1)
					if mode == "callback error" {
						return cause
					}
					if mode == "callback cancel" {
						cancel(cause)
					}
					config.Tags = []namespaces.TagDefinition{{Name: ""}}
					return nil
				}}
				if mode == "protected option header" {
					options = append(options, namespaces.WithCreateHeader("Content-Type", "text/plain"))
				}
				value, err := namespaces.New(client).Create(contextArg, namespaceName, options...)
				if value != nil || err == nil || calls.Load() != 0 {
					t.Fatal(value, err, calls.Load())
				}
				if mode == "callback error" && !errors.Is(err, cause) || mode == "callback cancel" && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
					t.Fatal(err)
				}
				before := mode == "nil context" || mode == "nil provider" || mode == "wrong service" || mode == "protected source header"
				if before && callbacks.Load() != 0 || !before && callbacks.Load() != 1 {
					t.Fatal("callback order", callbacks.Load(), err)
				}
			})
		}
	})
	t.Run("captured headers live auth and original target guard", func(t *testing.T) {
		var calls atomic.Int32
		client := namespaceClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Source") != "before" || r.Header.Get("X-Auth-Token") != "new-live" || r.URL.String() != namespaceBase+namespaceCollection {
				t.Error(r.URL, r.Header)
			}
			return namespaceWire(201, &namespaceBody{Reader: strings.NewReader(namespaceObject)}), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "before"}
		provider := client.ProviderClient
		value, err := namespaces.New(client).Create(context.Background(), namespaceName, namespaces.WithCreateTags([]namespaces.TagDefinition{{Name: ""}}), func(config *namespaces.CreateOpts) error {
			client.MoreHeaders["X-Source"] = "after"
			provider.SetToken("new-live")
			return nil
		})
		if err != nil || value == nil || calls.Load() != 1 || client.ProviderClient != provider || client.MoreHeaders["X-Source"] != "after" {
			t.Fatal(value, err, calls.Load())
		}
		for _, mode := range []string{"provider", "endpoint", "resource base", "microversion"} {
			t.Run(mode, func(t *testing.T) {
				calls.Store(0)
				client := namespaceClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("retargeted") })
				value, err := namespaces.New(client).Create(context.Background(), namespaceName, namespaces.WithCreateTags([]namespaces.TagDefinition{{Name: ""}}), func(*namespaces.CreateOpts) error {
					switch mode {
					case "provider":
						client.ProviderClient = &gophercloud.ProviderClient{}
					case "endpoint":
						client.Endpoint += "different/"
					case "resource base":
						client.ResourceBase = namespaceBase + "different/"
					case "microversion":
						client.Microversion = "2.18"
					}
					return nil
				})
				if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
					t.Fatal(value, err, calls.Load())
				}
			})
		}
	})
}

func TestMetadefNamespaceNestedAcceptedResponseEvidence(t *testing.T) {
	t.Run("passive nested response and independent raw ownership", func(t *testing.T) {
		raw := []byte(`{"namespace":"different::response","properties":17,"objects":null,"tags":[null,42],"resource_type_associations":"passive","created_at":"not-a-date","links":17,"x-number":9007199254740993123456789}`)
		body := &namespaceBody{Reader: bytes.NewReader(raw)}
		wire := namespaceWire(201, body)
		wire.Header.Set("Location", "https://other.invalid/never-follow")
		var calls atomic.Int32
		client := namespaceClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return wire, nil })
		value, err := namespaces.New(client).Create(context.Background(), namespaceName, namespaces.WithCreateOpts(nestedOptions()))
		if err != nil || value == nil || calls.Load() != 1 || body.closes.Load() != 1 || *value.Namespace != "different::response" || value.Links != nil || *value.CreatedAt != "not-a-date" || string(value.Body["properties"]) != "17" || string(value.Body["x-number"]) != "9007199254740993123456789" {
			t.Fatal(value, err, calls.Load())
		}
		raw[0] = '!'
		wire.Header.Set("Location", "changed")
		if value.Header.Get("Location") != "https://other.invalid/never-follow" || string(value.Body["namespace"]) != `"different::response"` {
			t.Fatal("response aliases borrowed proof", value)
		}
		*value.Namespace = "typed changed"
		if string(value.Body["namespace"]) != `"different::response"` {
			t.Fatal("typed canonical aliases raw")
		}
	})
	for _, raw := range [][]byte{[]byte(`[]`), []byte(`null`), []byte(`{"namespace":42}`), []byte(`{"namespace":"ok","protected":"false"}`), []byte(`{"namespace":"ok"`), []byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}} {
		t.Run("strict model "+fmt.Sprintf("%x", raw), func(t *testing.T) {
			body := &namespaceBody{Reader: bytes.NewReader(raw)}
			client := namespaceClient(func(*http.Request) (*http.Response, error) { return namespaceWire(201, body), nil })
			value, err := namespaces.New(client).Create(context.Background(), namespaceName, namespaces.WithCreateTags([]namespaces.TagDefinition{{Name: ""}}))
			namespaceProof(t, err, 201, raw)
			if value != nil || body.closes.Load() != 1 {
				t.Fatal(value, err, body.closes.Load())
			}
		})
	}
	for _, mode := range []string{"Read", "Close", "cancel", "Read Close cancel", "source"} {
		t.Run("accepted "+mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("Read"), errors.New("Close"), errors.New("custom cancellation")
			raw := []byte(namespaceObject)
			body := &namespaceBody{Reader: bytes.NewReader(raw)}
			if strings.Contains(mode, "Read") {
				body.Reader = namespaceReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
			}
			if strings.Contains(mode, "Close") {
				body.closeErr = closeCause
			}
			var calls, hooks atomic.Int32
			client := namespaceClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return namespaceWire(201, body), nil })
			if strings.Contains(mode, "cancel") {
				body.onClose = func() { cancel(cancelCause) }
			}
			if mode == "source" {
				body.onClose = func() { client.ProviderClient = &gophercloud.ProviderClient{} }
			}
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				hooks.Add(1)
				return errors.New("unexpected accepted replay")
			}
			value, err := namespaces.New(client).Create(ctx, namespaceName, namespaces.WithCreateOpts(nestedOptions()))
			namespaceProof(t, err, 201, raw)
			if value != nil || calls.Load() != 1 || hooks.Load() != 0 || body.closes.Load() != 1 || strings.Contains(mode, "Read") && !errors.Is(err, readCause) || strings.Contains(mode, "Close") && !errors.Is(err, closeCause) || strings.Contains(mode, "cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) || mode == "source" && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(value, err, calls.Load(), hooks.Load(), body.closes.Load())
			}
		})
	}
}

func TestMetadefNamespaceNestedNativeRetriesAndServerFailures(t *testing.T) {
	t.Run("prebody retry unchanged body and advanced headers", func(t *testing.T) {
		var calls, hooks atomic.Int32
		var initial []byte
		var rejected *namespaceBody
		client := namespaceClient(func(r *http.Request) (*http.Response, error) {
			n := calls.Add(1)
			raw, _ := io.ReadAll(r.Body)
			if n == 1 {
				initial = append([]byte(nil), raw...)
				rejected = &namespaceBody{Reader: strings.NewReader("first503")}
				return namespaceWire(503, rejected), nil
			}
			if n != 2 || !bytes.Equal(raw, initial) || r.Header.Get("X-Native") != "policy" || r.URL.String() != namespaceBase+namespaceCollection || r.Method != "POST" {
				t.Error(n, r.URL, r.Header, string(raw), string(initial))
			}
			return namespaceWire(201, &namespaceBody{Reader: strings.NewReader(namespaceObject)}), nil
		})
		client.RetryFunc = func(ctx context.Context, method, target string, options *gophercloud.RequestOpts, original error, retries uint) error {
			hooks.Add(1)
			if !gophercloud.ResponseCodeIs(original, 503) {
				t.Error(original)
			}
			options.JSONBody = json.RawMessage(append([]byte(nil), initial...))
			options.MoreHeaders = map[string]string{"X-Native": "policy"}
			return nil
		}
		value, err := namespaces.New(client).Create(context.Background(), namespaceName, namespaces.WithCreateOpts(nestedOptions()))
		if err != nil || value == nil || calls.Load() != 2 || hooks.Load() != 1 || rejected.closes.Load() != 1 || client.MoreHeaders != nil {
			t.Fatal(value, err, calls.Load(), hooks.Load())
		}
	})
	for _, mode := range []string{"JSON replacement", "in-place JSON", "null JSON", "raw body", "JSON response", "drain body"} {
		t.Run("guard "+mode, func(t *testing.T) {
			var calls, hooks atomic.Int32
			body := &namespaceBody{Reader: strings.NewReader("guard503")}
			client := namespaceClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return namespaceWire(503, body), nil })
			client.RetryFunc = func(ctx context.Context, method, target string, options *gophercloud.RequestOpts, original error, retries uint) error {
				hooks.Add(1)
				switch mode {
				case "JSON replacement":
					options.JSONBody = map[string]any{"namespace": namespaceName, "properties": map[string]any{}}
				case "in-place JSON":
					raw := options.JSONBody.(json.RawMessage)
					i := bytes.Index(raw, []byte("9007199254740993"))
					if i < 0 {
						t.Fatal("nested number missing")
					}
					raw[i] = '1'
				case "null JSON":
					options.JSONBody = json.RawMessage(`null`)
				case "raw body":
					options.RawBody = strings.NewReader("body")
				case "JSON response":
					options.JSONResponse = new(any)
				case "drain body":
					options.KeepResponseBody = false
				}
				return nil
			}
			value, err := namespaces.New(client).Create(context.Background(), namespaceName, namespaces.WithCreateOpts(nestedOptions()))
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || calls.Load() != 1 || hooks.Load() != 1 || body.closes.Load() != 1 {
				t.Fatal(value, err, calls.Load(), hooks.Load(), body.closes.Load())
			}
		})
	}
	t.Run("native-expanded status remains actual200 error", func(t *testing.T) {
		var calls, hooks atomic.Int32
		raw := []byte(`{"namespace":"unexpected"}`)
		body := &namespaceBody{Reader: bytes.NewReader(raw)}
		client := namespaceClient(func(*http.Request) (*http.Response, error) {
			if calls.Add(1) == 1 {
				return namespaceWire(503, &namespaceBody{Reader: strings.NewReader("initial")}), nil
			}
			return namespaceWire(200, body), nil
		})
		client.RetryFunc = func(ctx context.Context, method, target string, options *gophercloud.RequestOpts, original error, retries uint) error {
			hooks.Add(1)
			options.OkCodes = append(options.OkCodes, 200)
			return nil
		}
		value, err := namespaces.New(client).Create(context.Background(), namespaceName, namespaces.WithCreateTags([]namespaces.TagDefinition{{Name: ""}}))
		var native gophercloud.ErrUnexpectedResponseCode
		var sdk *resource.ResponseError
		if value != nil || !errors.As(err, &native) || errors.As(err, &sdk) || native.Actual != 200 || !reflect.DeepEqual(native.Expected, []int{201}) || native.Method != "POST" || native.URL != namespaceBase+namespaceCollection || !bytes.Equal(native.Body, raw) || native.ResponseHeader.Get("X-Request-Id") != "actual-namespace" || calls.Load() != 2 || hooks.Load() != 1 || body.closes.Load() != 1 {
			t.Fatal(value, err, native, calls.Load(), hooks.Load())
		}
	})
	t.Run("transport and callback causes do not become missing success", func(t *testing.T) {
		transportCause, callbackCause := errors.New("transport cause"), errors.New("callback cause")
		nested404 := gophercloud.ErrUnexpectedResponseCode{Actual: 404, Expected: []int{201}, Method: "POST", URL: "https://nested.invalid/decoy"}
		original := errors.Join(transportCause, nested404)
		var calls, hooks atomic.Int32
		client := namespaceClient(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return nil, original
		})
		client.RetryFunc = func(ctx context.Context, method, target string, options *gophercloud.RequestOpts, received error, retries uint) error {
			hooks.Add(1)
			if !errors.Is(received, transportCause) {
				t.Error("hook lost transport cause", received)
			}
			return callbackCause
		}
		value, err := namespaces.New(client).Create(context.Background(), namespaceName, namespaces.WithCreateOpts(nestedOptions()))
		var native gophercloud.ErrUnexpectedResponseCode
		var sdk *resource.ResponseError
		if value != nil || !errors.Is(err, transportCause) || !errors.Is(err, callbackCause) || !errors.As(err, &native) || errors.As(err, &sdk) || native.Actual != 404 || calls.Load() != 1 || hooks.Load() != 1 {
			t.Fatal(value, err, native, calls.Load(), hooks.Load())
		}
	})
	for _, code := range []int{400, 403, 404, 409} {
		t.Run(fmt.Sprintf("server%d no rollback", code), func(t *testing.T) {
			var calls atomic.Int32
			raw := []byte(fmt.Sprintf("partial remote state status%d", code))
			body := &namespaceBody{Reader: bytes.NewReader(raw)}
			client := namespaceClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Method != "POST" || r.URL.String() != namespaceBase+namespaceCollection {
					t.Error("client compensation", r.Method, r.URL)
				}
				return namespaceWire(code, body), nil
			})
			value, err := namespaces.New(client).Create(context.Background(), namespaceName, namespaces.WithCreateOpts(nestedOptions()))
			var native gophercloud.ErrUnexpectedResponseCode
			if value != nil || !errors.As(err, &native) || native.Actual != code || !bytes.Equal(native.Body, raw) || native.ResponseHeader.Get("X-Request-Id") != "actual-namespace" || calls.Load() != 1 || body.closes.Load() != 1 {
				t.Fatal(value, err, native, calls.Load(), body.closes.Load())
			}
		})
	}
}
