package metadefnamespaces

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"gophercloudsdk/resource"
)

func TestMetadefNamespaceCreateNestedOptionsDeepFactorySnapshots(t *testing.T) {
	description, prefix, target := "before", "prefix", "target"
	properties := map[string]PropertyDefinition{"p": {Type: "string", Title: "title", Description: &description, Attributes: map[string]json.RawMessage{"default": json.RawMessage(`9007199254740993`)}}}
	objects := []ObjectDefinition{{Name: "object", Description: &description, Properties: properties, Required: []string{"owned"}}}
	tags := []TagDefinition{{Name: "tag"}}
	associations := []ResourceTypeAssociationDefinition{{Name: "type", Prefix: &prefix, PropertiesTarget: &target}}
	options := []CreateOption{WithCreateProperties(properties), WithCreateObjects(objects), WithCreateTags(tags), WithCreateResourceTypeAssociations(associations)}
	full := WithCreateOpts(CreateOpts{Properties: properties, Objects: objects, Tags: tags, ResourceTypeAssociations: associations})
	description, prefix, target = "changed", "changed", "changed"
	properties["p"].Attributes["default"][0] = '1'
	definition := properties["p"]
	definition.Type = "changed"
	properties["p"] = definition
	objects[0].Name = "changed"
	objects[0].Required[0] = "changed"
	tags[0].Name = "changed"
	associations[0].Name = "changed"
	check := func(v CreateOpts) {
		t.Helper()
		p := v.Properties["p"]
		if p.Type != "string" || *p.Description != "before" || string(p.Attributes["default"]) != "9007199254740993" || v.Objects[0].Name != "object" || v.Objects[0].Required[0] != "owned" || *v.Objects[0].Description != "before" || v.Tags[0].Name != "tag" || v.ResourceTypeAssociations[0].Name != "type" || *v.ResourceTypeAssociations[0].Prefix != "prefix" || *v.ResourceTypeAssociations[0].PropertiesTarget != "target" {
			t.Fatalf("deep snapshot %+v", v)
		}
	}
	for _, opts := range [][]CreateOption{options, {full}} {
		value, err := prepareCreate(opts)
		if err != nil {
			t.Fatal(err)
		}
		check(value)
		value.Properties["p"].Attributes["default"][0] = '2'
		*value.Properties["p"].Description = "consumer"
		value.Objects[0].Properties["p"].Attributes["default"][0] = '3'
		value.Objects[0].Required[0] = "consumer"
		value.Tags[0].Name = "consumer"
		*value.ResourceTypeAssociations[0].Prefix = "consumer"
		again, err := prepareCreate(opts)
		if err != nil {
			t.Fatal(err)
		}
		check(again)
	}
}

func TestMetadefNamespaceCreateNestedOptionsReplacementAndPresence(t *testing.T) {
	initial := WithCreateOpts(CreateOpts{DisplayName: namespaceOptionPointer("scalar"), Properties: map[string]PropertyDefinition{"p": {Type: "string"}}, Objects: []ObjectDefinition{{Name: "one"}}, Tags: []TagDefinition{{Name: "one"}}, ResourceTypeAssociations: []ResourceTypeAssociationDefinition{{Name: "one"}}})
	v, err := prepareCreate([]CreateOption{initial, WithCreateOpts(CreateOpts{})})
	if err != nil || v.DisplayName != nil || v.Properties != nil || v.Objects != nil || v.Tags != nil || v.ResourceTypeAssociations != nil {
		t.Fatal("full replacement failed", err, v)
	}
	v, err = prepareCreate([]CreateOption{initial, WithCreateProperties(nil), WithCreateObjects(nil), WithCreateTags(nil), WithCreateResourceTypeAssociations(nil)})
	if err != nil || v.DisplayName == nil || v.Properties != nil || v.Objects != nil || v.Tags != nil || v.ResourceTypeAssociations != nil {
		t.Fatal("field replacement failed", err, v)
	}
	v, err = prepareCreate([]CreateOption{initial, WithCreateProperties(map[string]PropertyDefinition{}), WithCreateObjects([]ObjectDefinition{}), WithCreateTags([]TagDefinition{}), WithCreateResourceTypeAssociations([]ResourceTypeAssociationDefinition{})})
	if err != nil || v.Properties == nil || v.Objects == nil || v.Tags == nil || v.ResourceTypeAssociations == nil {
		t.Fatal("empty presence lost", err, v)
	}
	body, err := json.Marshal(createBody("parent", v))
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{`"properties":{}`, `"objects":[]`, `"tags":[]`, `"resource_type_associations":[]`} {
		if !strings.Contains(string(body), part) {
			t.Fatal(string(body), part)
		}
	}
	v, err = prepareCreate([]CreateOption{initial, WithCreateProperties(map[string]PropertyDefinition{"last": {Type: "number", Title: ""}}), WithCreateObjects([]ObjectDefinition{{Name: "last"}}), WithCreateTags([]TagDefinition{{Name: "dup"}, {Name: "dup"}}), WithCreateResourceTypeAssociations([]ResourceTypeAssociationDefinition{{Name: "last"}})})
	if err != nil || len(v.Properties) != 1 || v.Properties["last"].Type != "number" || v.Objects[0].Name != "last" || !reflect.DeepEqual(v.Tags, []TagDefinition{{Name: "dup"}, {Name: "dup"}}) || v.ResourceTypeAssociations[0].Name != "last" {
		t.Fatal("last field helper wins", err, v)
	}
}

func TestMetadefNamespaceCreateNestedOptionsCallbackAndRawValidation(t *testing.T) {
	t.Run("retained recursive config cannot change next callback", func(t *testing.T) {
		calls, callbacks := 0, 0
		var retained *CreateOpts
		client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			body := nestedCorePayload(t, req)
			p := nestedCoreObject(t, nestedCoreObject(t, body["properties"])["p"])
			if nestedCoreText(t, p["type"]) != "string" || nestedCoreText(t, p["title"]) != "" || string(p["default"]) != "9007199254740993" {
				t.Fatal("owned property changed")
			}
			var objects []json.RawMessage
			if err := json.Unmarshal(body["objects"], &objects); err != nil {
				t.Fatal(err)
			}
			if string(nestedCoreObject(t, objects[0])["required"]) != `["owned"]` {
				t.Fatal("required alias")
			}
			var tags []map[string]string
			if err := json.Unmarshal(body["tags"], &tags); err != nil {
				t.Fatal(err)
			}
			if tags[0]["name"] != "owned" {
				t.Fatal("tag alias")
			}
			return namespaceCoreJSON(req, 201, `{}`), nil
		})
		first := CreateOption(func(c *CreateOpts) error {
			callbacks++
			c.Properties = map[string]PropertyDefinition{"p": {Type: "string", Attributes: map[string]json.RawMessage{"default": json.RawMessage(`9007199254740993`)}}}
			c.Objects = []ObjectDefinition{{Name: "object", Required: []string{"owned"}}}
			c.Tags = []TagDefinition{{Name: "owned"}}
			c.ResourceTypeAssociations = []ResourceTypeAssociationDefinition{{Name: "type", Prefix: namespaceOptionPointer("owned")}}
			retained = c
			return nil
		})
		second := CreateOption(func(c *CreateOpts) error {
			callbacks++
			retained.Properties["p"].Attributes["default"][0] = '1'
			p := retained.Properties["p"]
			p.Type = "changed"
			retained.Properties["p"] = p
			retained.Objects[0].Required[0] = "changed"
			retained.Tags[0].Name = "changed"
			*retained.ResourceTypeAssociations[0].Prefix = "changed"
			if c.Properties["p"].Type != "string" || *c.ResourceTypeAssociations[0].Prefix != "owned" {
				t.Fatal("callbacks shared recursive storage")
			}
			return nil
		})
		if _, err := New(client).Create(context.Background(), "parent", first, second); err != nil || calls != 1 || callbacks != 2 {
			t.Fatal(err, calls, callbacks)
		}
	})
	t.Run("value Type and Title precise validation", func(t *testing.T) {
		good := PropertyDefinition{Type: "custom", Title: "", Attributes: map[string]json.RawMessage{"default": json.RawMessage(`null`), "enum": json.RawMessage(`[1,true,"x"]`), "minimum": json.RawMessage(`1.25`), "unknown": json.RawMessage(`{"n":12345678901234567890}`)}}
		v, err := prepareCreate([]CreateOption{WithCreateProperties(map[string]PropertyDefinition{"": good})})
		if err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(createBody("p", v))
		if err != nil || !strings.Contains(string(raw), `"title":""`) || !strings.Contains(string(raw), `"minimum":1.25`) {
			t.Fatal(string(raw), err)
		}
		bad := string([]byte{0xff})
		for _, p := range []PropertyDefinition{{}, {Type: bad}, {Type: "string", Title: bad}, {Type: "string", Attributes: map[string]json.RawMessage{"x": nil}}, {Type: "string", Attributes: map[string]json.RawMessage{bad: json.RawMessage(`null`)}}} {
			if _, err := prepareCreate([]CreateOption{WithCreateProperties(map[string]PropertyDefinition{"p": p})}); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(p, err)
			}
		}
		cause := errors.New("callback")
		if _, err := prepareCreate([]CreateOption{func(*CreateOpts) error { return cause }}); err != cause {
			t.Fatal("cause identity", err)
		}
	})
}

func TestMetadefNamespaceCreateNestedOptionsParallelReusableHelpers(t *testing.T) {
	var calls atomic.Int64
	client := namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		var body map[string]json.RawMessage
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		var properties map[string]json.RawMessage
		var p map[string]json.RawMessage
		if err := json.Unmarshal(body["properties"], &properties); err != nil {
			t.Error(err)
		}
		if err := json.Unmarshal(properties["p"], &p); err != nil {
			t.Error(err)
		}
		if string(p["default"]) != "9007199254740993" || string(p["title"]) != `""` {
			t.Error("parallel helper payload changed")
		}
		return namespaceCoreJSON(req, 201, `{}`), nil
	})
	raw := json.RawMessage(`9007199254740993`)
	prefix := "owned"
	input := CreateOpts{Properties: map[string]PropertyDefinition{"p": {Type: "string", Attributes: map[string]json.RawMessage{"default": raw}}}, Objects: []ObjectDefinition{{Name: "object", Required: []string{"owned"}}}, Tags: []TagDefinition{{Name: "tag"}}, ResourceTypeAssociations: []ResourceTypeAssociationDefinition{{Name: "type", Prefix: &prefix}}}
	full := WithCreateOpts(input)
	property := WithCreateProperties(input.Properties)
	objects := WithCreateObjects(input.Objects)
	tags := WithCreateTags(input.Tags)
	associations := WithCreateResourceTypeAssociations(input.ResourceTypeAssociations)
	raw[0] = '1'
	prefix = "changed"
	input.Objects[0].Required[0] = "changed"
	input.Tags[0].Name = "changed"
	var wg sync.WaitGroup
	for range 24 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, opts := range [][]CreateOption{{full}, {property, objects, tags, associations}} {
				v, err := New(client).Create(context.Background(), "parent", opts...)
				if err != nil || v == nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 48 {
		t.Fatal(calls.Load())
	}
}
