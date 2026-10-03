package metadefnamespaces

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

func nestedCoreObject(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var value map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
func nestedCoreText(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
func nestedCorePayload(t *testing.T, req *http.Request) map[string]json.RawMessage {
	t.Helper()
	var value map[string]json.RawMessage
	if err := json.NewDecoder(req.Body).Decode(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestMetadefNamespaceCreateNestedCoreAllDefinitionsOnePOST(t *testing.T) {
	calls := 0
	key := strings.Repeat("界", 81) + "/%\n"
	name := strings.Repeat("界", 81) + "/%?\n"
	response := `{"namespace":"server-choice","properties":{"wire-key":null},"objects":[],"tags":false,"resource_type_associations":[{"name":"server"}],"self":"https://foreign.test/","unknown":9007199254740993}`
	client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != "POST" || req.URL.EscapedPath() != "/reverse/glance/v2/metadefs/namespaces" || req.URL.RawQuery != "" {
			t.Fatal("wrong nested route")
		}
		body := nestedCorePayload(t, req)
		if nestedCoreText(t, body["namespace"]) != "OS::Nested" || len(body) != 5 {
			t.Fatal(body)
		}
		p := nestedCoreObject(t, nestedCoreObject(t, body["properties"])[key])
		if nestedCoreText(t, p["type"]) != "future-type" || nestedCoreText(t, p["title"]) != "" || string(p["default"]) != "123456789012345678901234567890" || string(p["minimum"]) != "1.234567890123456789" || string(p["custom"]) != "null" {
			t.Fatal("property precision/presence", p)
		}
		var objects []json.RawMessage
		if err := json.Unmarshal(body["objects"], &objects); err != nil {
			t.Fatal(err)
		}
		object := nestedCoreObject(t, objects[0])
		if nestedCoreText(t, object["name"]) != name || string(object["required"]) != `["","a,b","line\n","a,b"]` || nestedCoreText(t, object["description"]) != "" {
			t.Fatal("object literals changed")
		}
		if nestedCoreText(t, nestedCoreObject(t, nestedCoreObject(t, object["properties"])[""])["title"]) != "" {
			t.Fatal("empty property key/title")
		}
		var tags []map[string]string
		if err := json.Unmarshal(body["tags"], &tags); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(tags, []map[string]string{{"name": ""}, {"name": name}, {"name": name}}) {
			t.Fatal(tags)
		}
		var assocs []json.RawMessage
		if err := json.Unmarshal(body["resource_type_associations"], &assocs); err != nil {
			t.Fatal(err)
		}
		a := nestedCoreObject(t, assocs[0])
		if nestedCoreText(t, a["name"]) != "" || nestedCoreText(t, a["prefix"]) != name || nestedCoreText(t, a["properties_target"]) != "line\n\x00" {
			t.Fatal("association literal bounds")
		}
		return namespaceCoreJSON(req, 201, response), nil
	})
	p := PropertyDefinition{Type: "future-type", Title: "", Attributes: map[string]json.RawMessage{"default": json.RawMessage(`123456789012345678901234567890`), "minimum": json.RawMessage(`1.234567890123456789`), "custom": json.RawMessage(`null`)}}
	value, err := New(client).Create(context.Background(), "OS::Nested", WithCreateProperties(map[string]PropertyDefinition{key: p}), WithCreateObjects([]ObjectDefinition{{Name: name, Description: namespaceOptionPointer(""), Properties: map[string]PropertyDefinition{"": {Type: "string"}}, Required: []string{"", "a,b", "line\n", "a,b"}}}), WithCreateTags([]TagDefinition{{}, {Name: name}, {Name: name}}), WithCreateResourceTypeAssociations([]ResourceTypeAssociationDefinition{{Prefix: &name, PropertiesTarget: namespaceOptionPointer("line\n\x00")}}))
	if err != nil || calls != 1 || value.StatusCode != 201 || *value.Namespace != "server-choice" || string(value.Body["tags"]) != "false" || string(value.Body["unknown"]) != "9007199254740993" {
		t.Fatal("raw passive result", err, value, calls)
	}
}

func TestMetadefNamespaceCreateNestedCoreNilEmptyAndDefaults(t *testing.T) {
	calls := 0
	client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		body := nestedCorePayload(t, req)
		switch calls {
		case 1, 3:
			if len(body) != 1 || nestedCoreText(t, body["namespace"]) != "parent" {
				t.Fatal("nil/default body", body)
			}
		case 2:
			for key, want := range map[string]string{"properties": "{}", "objects": "[]", "tags": "[]", "resource_type_associations": "[]"} {
				if string(body[key]) != want {
					t.Fatal(key, string(body[key]))
				}
			}
		case 4:
			var o, a []json.RawMessage
			if err := json.Unmarshal(body["objects"], &o); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(body["resource_type_associations"], &a); err != nil {
				t.Fatal(err)
			}
			first, second := nestedCoreObject(t, o[0]), nestedCoreObject(t, o[1])
			if len(first) != 1 || string(second["properties"]) != "{}" || string(second["required"]) != "[]" || nestedCoreText(t, second["description"]) != "" {
				t.Fatal("object inner presence")
			}
			first, second = nestedCoreObject(t, a[0]), nestedCoreObject(t, a[1])
			if len(first) != 1 || nestedCoreText(t, second["prefix"]) != "" || nestedCoreText(t, second["properties_target"]) != "" {
				t.Fatal("association inner presence")
			}
		default:
			t.Fatal("extra HTTP", calls)
		}
		return namespaceCoreJSON(req, 201, `{}`), nil
	})
	for _, opts := range [][]CreateOption{nil, {WithCreateProperties(map[string]PropertyDefinition{}), WithCreateObjects([]ObjectDefinition{}), WithCreateTags([]TagDefinition{}), WithCreateResourceTypeAssociations([]ResourceTypeAssociationDefinition{})}, {WithCreateProperties(map[string]PropertyDefinition{"x": {Type: "string"}}), WithCreateObjects([]ObjectDefinition{{Name: "x"}}), WithCreateTags([]TagDefinition{{Name: "x"}}), WithCreateResourceTypeAssociations([]ResourceTypeAssociationDefinition{{Name: "x"}}), WithCreateProperties(nil), WithCreateObjects(nil), WithCreateTags(nil), WithCreateResourceTypeAssociations(nil)}, {WithCreateObjects([]ObjectDefinition{{}, {Description: namespaceOptionPointer(""), Properties: map[string]PropertyDefinition{}, Required: []string{}}}), WithCreateResourceTypeAssociations([]ResourceTypeAssociationDefinition{{}, {Prefix: namespaceOptionPointer(""), PropertiesTarget: namespaceOptionPointer("")}})}} {
		if _, err := New(client).Create(context.Background(), "parent", opts...); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 4 {
		t.Fatal(calls)
	}
}

func TestMetadefNamespaceCreateNestedCoreCompletePreflight(t *testing.T) {
	calls := 0
	client := namespaceCoreClient(func(*http.Request) (*http.Response, error) { calls++; t.Fatal("preflight HTTP"); return nil, nil })
	api := New(client)
	bad := string([]byte{0xff})
	opts := []CreateOption{WithCreateProperties(map[string]PropertyDefinition{"x": {}}), WithCreateProperties(map[string]PropertyDefinition{bad: {Type: "string"}}), WithCreateProperties(map[string]PropertyDefinition{"x": {Type: bad}}), WithCreateProperties(map[string]PropertyDefinition{"x": {Type: "string", Title: bad}}), WithCreateProperties(map[string]PropertyDefinition{"x": {Type: "string", Description: &bad}}), WithCreateObjects([]ObjectDefinition{{Name: bad}}), WithCreateObjects([]ObjectDefinition{{Description: &bad}}), WithCreateObjects([]ObjectDefinition{{Properties: map[string]PropertyDefinition{"x": {}}}}), WithCreateObjects([]ObjectDefinition{{Required: []string{bad}}}), WithCreateTags([]TagDefinition{{Name: bad}}), WithCreateResourceTypeAssociations([]ResourceTypeAssociationDefinition{{Name: bad}}), WithCreateResourceTypeAssociations([]ResourceTypeAssociationDefinition{{Prefix: &bad}}), WithCreateResourceTypeAssociations([]ResourceTypeAssociationDefinition{{PropertiesTarget: &bad}})}
	for _, raw := range []json.RawMessage{nil, json.RawMessage(`{`), json.RawMessage(`{"x":`), json.RawMessage([]byte{'"', 0xff, '"'}), json.RawMessage(`NaN`)} {
		opts = append(opts, WithCreateProperties(map[string]PropertyDefinition{"x": {Type: "string", Attributes: map[string]json.RawMessage{"custom": raw}}}))
	}
	for _, key := range []string{bad, "name", "type", "title", "description", "self", "schema", "created_at", "updated_at", "namespace_name"} {
		opts = append(opts, WithCreateProperties(map[string]PropertyDefinition{"x": {Type: "string", Attributes: map[string]json.RawMessage{key: json.RawMessage(`null`)}}}))
	}
	for i, option := range opts {
		v, err := api.Create(context.Background(), "parent", option)
		if v != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(i, v, err)
		}
	}
	cause := errors.New("callback")
	if _, err := api.Create(context.Background(), "parent", func(*CreateOpts) error { return cause }); !errors.Is(err, cause) {
		t.Fatal(err)
	}
	callbacks := 0
	if _, err := api.Create(nil, "parent", func(*CreateOpts) error { callbacks++; return nil }); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.Create(context.Background(), strings.Repeat("界", 81), func(*CreateOpts) error { callbacks++; return nil }); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if calls != 0 || callbacks != 0 {
		t.Fatal(calls, callbacks)
	}
}

func TestMetadefNamespaceCreateNestedCoreAcceptedResponseOwnership(t *testing.T) {
	t.Run("read Close and custom context causes", func(t *testing.T) {
		raw := `{"namespace":"server","properties":{"x":{"type":"string","title":""}}}`
		readErr, closeErr, custom := errors.New("read"), errors.New("close"), errors.New("cancel")
		ctx, cancel := context.WithCancelCause(context.Background())
		body := &namespaceCoreBody{reader: &namespaceCoreReader{data: raw, err: readErr, after: func() { cancel(custom) }}, closeErr: closeErr}
		calls := 0
		client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			return namespaceCoreHTTP(req, 201, body), nil
		})
		v, err := New(client).Create(ctx, "parent", WithCreateProperties(map[string]PropertyDefinition{"x": {Type: "string"}}))
		if v != nil || body.closes != 1 || calls != 1 {
			t.Fatal(v, err, body.closes, calls)
		}
		for _, cause := range []error{readErr, closeErr, context.Canceled, custom} {
			if !errors.Is(err, cause) {
				t.Fatal("lost cause", cause, err)
			}
		}
		namespaceCoreProof(t, err, 201, raw)
	})
	t.Run("canonical root failure", func(t *testing.T) {
		for _, raw := range []string{`{"namespace":false,"tags":[]}`, "{\"namespace\":\"x\",\"objects\":[{\"name\":\"\xff\"}]}"} {
			c := namespaceCoreClient(func(req *http.Request) (*http.Response, error) { return namespaceCoreJSON(req, 201, raw), nil })
			v, err := New(c).Create(context.Background(), "parent", WithCreateTags([]TagDefinition{{Name: "x"}}))
			if v != nil || err == nil {
				t.Fatal(v, err)
			}
			namespaceCoreProof(t, err, 201, raw)
		}
	})
	t.Run("raw response has no input seed", func(t *testing.T) {
		raw := `{"namespace":null,"properties":false,"objects":{"unknown":9007199254740993},"tags":[],"resource_type_associations":[]}`
		var wire *http.Response
		c := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
			wire = namespaceCoreJSON(req, 201, raw)
			return wire, nil
		})
		v, err := New(c).Create(context.Background(), "requested", WithCreateObjects([]ObjectDefinition{{Name: "input"}}))
		if err != nil || v.Namespace != nil || string(v.Body["properties"]) != "false" {
			t.Fatal(v, err)
		}
		wire.Header.Set("X-Proof", "changed")
		if v.Header.Get("X-Proof") != "original" || string(v.Body["objects"]) != `{"unknown":9007199254740993}` {
			t.Fatal("proof alias")
		}
	})
}

func TestMetadefNamespaceCreateNestedCoreNativeErrorsAndRetryBody(t *testing.T) {
	t.Run("native rejection never compensates", func(t *testing.T) {
		for _, code := range []int{400, 403, 404, 409, 202} {
			calls := 0
			c := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return namespaceCoreJSON(req, code, "native error"), nil
			})
			v, err := New(c).Create(context.Background(), "parent", WithCreateTags([]TagDefinition{{Name: "child"}}))
			if v != nil || !gophercloud.ResponseCodeIs(err, code) || calls != 1 {
				t.Fatal(code, err, calls)
			}
		}
	})
	t.Run("in-place retry body ownership", func(t *testing.T) {
		calls := 0
		c := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			return namespaceCoreJSON(req, 503, "busy"), nil
		})
		c.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			raw := o.JSONBody.(json.RawMessage)
			at := strings.Index(string(raw), `"title":"owned"`)
			if at < 0 {
				t.Fatal("title missing")
			}
			copy(raw[at:], []byte(`"title":"rogue"`))
			return nil
		}
		v, err := New(c).Create(context.Background(), "parent", WithCreateProperties(map[string]PropertyDefinition{"x": {Type: "string", Title: "owned"}}))
		if v != nil || !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || calls != 1 {
			t.Fatal(v, err, calls)
		}
	})
	t.Run("safe retry retains full nested JSON", func(t *testing.T) {
		calls := 0
		var first []byte
		c := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			raw, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatal(err)
			}
			if calls == 1 {
				first = append([]byte(nil), raw...)
				return namespaceCoreJSON(req, 503, "busy"), nil
			}
			if string(first) != string(raw) {
				t.Fatal("body replay changed")
			}
			return namespaceCoreJSON(req, 201, `{}`), nil
		})
		c.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
			o.JSONBody = json.RawMessage(append([]byte(nil), o.JSONBody.(json.RawMessage)...))
			return nil
		}
		_, err := New(c).Create(context.Background(), "parent", WithCreateObjects([]ObjectDefinition{{Name: "x", Required: []string{}, Properties: map[string]PropertyDefinition{"p": {Type: "string"}}}}))
		if err != nil || calls != 2 {
			t.Fatal(err, calls)
		}
	})
}

func TestMetadefNamespaceCreateNestedCoreSourceAndUpdatePreservation(t *testing.T) {
	t.Run("Update still scalar replacement", func(t *testing.T) {
		calls := 0
		c := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			b := nestedCorePayload(t, req)
			if req.Method != "PUT" || req.URL.EscapedPath() != "/reverse/glance/v2/metadefs/namespaces/current" || len(b) != 1 || nestedCoreText(t, b["namespace"]) != "current" {
				t.Fatal("Update changed")
			}
			return namespaceCoreJSON(req, 200, `{}`), nil
		})
		if _, err := New(c).Update(context.Background(), "current"); err != nil || calls != 1 {
			t.Fatal(err, calls)
		}
	})
	t.Run("callback source change", func(t *testing.T) {
		calls := 0
		c := namespaceCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
		v, err := New(c).Create(context.Background(), "parent", WithCreateTags([]TagDefinition{{Name: "data"}}), func(*CreateOpts) error { c.ResourceBase = "https://example.test/other/"; return nil })
		if v != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
			t.Fatal(v, err, calls)
		}
	})
	t.Run("accepted body source change", func(t *testing.T) {
		var c *gophercloud.ServiceClient
		raw := `{"namespace":"server","tags":[{"name":"x"}]}`
		c = namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
			b := &namespaceCoreBody{reader: &namespaceCoreReader{data: raw, err: io.EOF, after: func() { c.Endpoint = "https://other.test/" }}}
			return namespaceCoreHTTP(req, 201, b), nil
		})
		v, err := New(c).Create(context.Background(), "parent", WithCreateTags([]TagDefinition{{Name: "data"}}))
		if v != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(v, err)
		}
		namespaceCoreProof(t, err, 201, raw)
	})
}
