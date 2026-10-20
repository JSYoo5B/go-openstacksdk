package image_test

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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/image"
	sdkimages "github.com/JSYoo5B/go-openstacksdk/image/v2/images"
	sdkmembers "github.com/JSYoo5B/go-openstacksdk/image/v2/members"
	sdktasks "github.com/JSYoo5B/go-openstacksdk/image/v2/tasks"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const schemasPrefix = "/reverse/schema-discovery/glance/v2/"
const schemasBase = "https://glance.invalid" + schemasPrefix

type schemasGetter struct {
	name, path string
	get        func(*image.Service, context.Context, ...image.GetSchemaOption) (*image.Schema, error)
}

func schemasGetters() []schemasGetter {
	return []schemasGetter{
		{"GetImagesSchema", "schemas/images", (*image.Service).GetImagesSchema},
		{"GetImageSchema", "schemas/image", (*image.Service).GetImageSchema},
		{"GetMembersSchema", "schemas/members", (*image.Service).GetMembersSchema},
		{"GetMemberSchema", "schemas/member", (*image.Service).GetMemberSchema},
		{"GetTasksSchema", "schemas/tasks", (*image.Service).GetTasksSchema},
		{"GetTaskSchema", "schemas/task", (*image.Service).GetTaskSchema},
		{"GetMetadefNamespaceSchema", "schemas/metadefs/namespace", (*image.Service).GetMetadefNamespaceSchema},
		{"GetMetadefNamespacesSchema", "schemas/metadefs/namespaces", (*image.Service).GetMetadefNamespacesSchema},
		{"GetMetadefResourceTypeSchema", "schemas/metadefs/resource_type", (*image.Service).GetMetadefResourceTypeSchema},
		{"GetMetadefResourceTypesSchema", "schemas/metadefs/resource_types", (*image.Service).GetMetadefResourceTypesSchema},
		{"GetMetadefObjectSchema", "schemas/metadefs/object", (*image.Service).GetMetadefObjectSchema},
		{"GetMetadefObjectsSchema", "schemas/metadefs/objects", (*image.Service).GetMetadefObjectsSchema},
		{"GetMetadefPropertySchema", "schemas/metadefs/property", (*image.Service).GetMetadefPropertySchema},
		{"GetMetadefPropertiesSchema", "schemas/metadefs/properties", (*image.Service).GetMetadefPropertiesSchema},
		{"GetMetadefTagSchema", "schemas/metadefs/tag", (*image.Service).GetMetadefTagSchema},
		{"GetMetadefTagsSchema", "schemas/metadefs/tags", (*image.Service).GetMetadefTagsSchema},
	}
}

type schemasTransport func(*http.Request) (*http.Response, error)

func (f schemasTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	wire, err := f(r)
	if wire != nil && wire.Request == nil {
		wire.Request = r
	}
	return wire, err
}

type schemasBody struct {
	io.Reader
	closes   atomic.Int32
	closeErr error
	onClose  func()
}

func (b *schemasBody) Close() error {
	b.closes.Add(1)
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

type schemasReader func([]byte) (int, error)

func (f schemasReader) Read(p []byte) (int, error) { return f(p) }

func schemasWire(code int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{
		"Content-Type": {"application/json"}, "X-Request-Id": {"actual-schema"},
	}, Body: body}
}

func schemasClient(transport schemasTransport) *gophercloud.ServiceClient {
	provider := &gophercloud.ProviderClient{HTTPClient: http.Client{Transport: transport}}
	provider.UseTokenLock()
	provider.SetToken("initial")
	return &gophercloud.ServiceClient{ProviderClient: provider, Type: "image", Endpoint: schemasBase}
}

func schemasProof(t *testing.T, err error, raw []byte) *resource.ResponseError {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != 200 || !bytes.Equal(proof.Body, raw) || proof.Header.Get("X-Request-Id") != "actual-schema" {
		t.Fatalf("accepted schema evidence lost: %v %#v", err, proof)
	}
	return proof
}

func TestImageSchemasFixedRoutesAndRawDiscovery(t *testing.T) {
	t.Run("all sixteen fixed bodyless getters", func(t *testing.T) {
		cloud := testcloud.New(t)
		client := cloud.Client("image", "/catalog/unused/")
		client.ResourceBase = cloud.Server.URL + schemasPrefix
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		client.Microversion = "2.10"
		getters := schemasGetters()
		var calls atomic.Int32
		cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			index := int(calls.Add(1)) - 1
			raw, err := io.ReadAll(r.Body)
			if index >= len(getters) {
				t.Error("schema discovery, pagination or URL following", r.Method, r.URL)
				w.WriteHeader(500)
				return
			}
			if err != nil || len(raw) != 0 || r.ContentLength != 0 || r.Method != http.MethodGet || r.URL.Path != schemasPrefix+getters[index].path || r.URL.RawQuery != "" || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Option") != "ordinary" || r.Header.Get("X-Auth-Token") != fmt.Sprint("live-", index) || r.Header.Get("OpenStack-API-Version") != "image 2.10" {
				t.Error(getters[index].name, r.Method, r.URL, string(raw), r.Header, err)
			}
			w.Header().Set("X-Request-Id", "actual-schema")
			w.Header().Set("Location", "https://foreign.invalid/schemas/other")
			w.Header().Set("Link", "<https://foreign.invalid/next>; rel=\"next\"")
			testcloud.JSON(w, 200, "{\"name\":\"passive-server-name\",\"properties\":{},\"links\":[{\"href\":\"https://foreign.invalid/{next}\",\"rel\":\"next\"}]}")
		})
		service := image.New(client)
		for index, getter := range getters {
			cloud.Provider.SetToken(fmt.Sprint("live-", index))
			value, err := getter.get(service, context.Background(), image.WithGetSchemaHeader("X-Option", "ordinary"))
			if err != nil || value == nil || value.Name == nil || *value.Name != "passive-server-name" || value.Properties == nil || len(value.Properties) != 0 || value.AdditionalProperties != nil || value.Links != nil || value.StatusCode != 200 || value.Header.Get("X-Request-Id") != "actual-schema" || string(value.Body["links"]) == "" || calls.Load() != int32(index+1) {
				t.Fatal(getter.name, value, err, calls.Load())
			}
		}
		if calls.Load() != 16 {
			t.Fatal(calls.Load())
		}
	})
	fixtures := []struct {
		index int
		raw   string
	}{
		{1, `{"name":"image","properties":{"operator_extension":{"type":"string","is_base":false}},"required":["operator_extension"],"additionalProperties":{"type":"string"},"links":[{"rel":"self","href":"{self}"}]}`},
		{3, `{"name":"member","properties":{"member_id":{"type":"string"},"status":{"type":"string"}}}`},
		{4, `{"name":"tasks","properties":{"tasks":{"type":"array","items":{"name":"task","properties":{"id":{"type":"string"},"status":{"type":"string"}}}},"schema":{"type":"string"}},"links":[{"rel":"describedby","href":"{schema}"}]}`},
		{8, `{"name":"resource_type_association","properties":{"name":{"type":"string"}},"required":["name"],"additionalProperties":false}`},
		{13, `{"name":"properties","properties":{"properties":{"type":"object","additionalProperties":{"name":"property","properties":{"description":{"type":"string"}},"required":["type"],"additionalProperties":false}},"first":{"type":"string"},"next":{"type":"string"},"schema":{"type":"string"}},"definitions":{"stringArray":{"type":"array","items":{"type":"string"}}},"links":[{"rel":"next","href":"{next}"},{"rel":"describedby","href":"{schema}"}]}`},
	}
	for _, fixture := range fixtures {
		getter := schemasGetters()[fixture.index]
		t.Run("raw or minimal "+getter.name, func(t *testing.T) {
			body := &schemasBody{Reader: strings.NewReader(fixture.raw)}
			var calls atomic.Int32
			client := schemasClient(func(r *http.Request) (*http.Response, error) {
				calls.Add(1)
				if r.Method != http.MethodGet || r.URL.String() != schemasBase+getter.path || r.Body != nil {
					t.Error(r.Method, r.URL, r.Body)
				}
				wire := schemasWire(200, body)
				wire.Header.Set("Link", "<https://foreign.invalid/next>; rel=\"next\"")
				return wire, nil
			})
			value, err := getter.get(image.New(client), context.Background())
			var fields map[string]json.RawMessage
			if err != nil || value == nil || json.Unmarshal([]byte(fixture.raw), &fields) != nil || !reflect.DeepEqual(value.Body, fields) || string(value.AdditionalProperties) != string(fields["additionalProperties"]) || value.Links != nil || calls.Load() != 1 || body.closes.Load() != 1 {
				t.Fatal(value, err, calls.Load(), body.closes.Load())
			}
			if fixture.index == 4 {
				if bytes.Contains(value.Properties["tasks"], []byte("\"input\"")) || !bytes.Contains(value.Properties["tasks"], []byte("\"items\"")) {
					t.Fatal("minimal task item changed", value)
				}
			}
			if fixture.index == 13 && (!bytes.Contains(value.Properties["properties"], []byte("\"type\":\"object\"")) || !bytes.Contains(value.Properties["properties"], []byte("\"required\":[\"type\"]")) || string(value.Definitions["stringArray"]) == "") {
				t.Fatal("DictCollection was flattened or converted to array", value)
			}
		})
	}
}

func TestImageSchemasCanonicalFieldsAndPassiveKeywords(t *testing.T) {
	for _, mode := range []string{"missing", "null", "empty", "required strings"} {
		t.Run("canonical presence "+mode, func(t *testing.T) {
			raw := `{}`
			switch mode {
			case "null":
				raw = `{"name":null,"properties":null,"definitions":null,"required":null}`
			case "empty":
				raw = `{"name":"","properties":{},"definitions":{},"required":[]}`
			case "required strings":
				raw = `{"required":["","name","name","属性"]}`
			}
			client := schemasClient(func(*http.Request) (*http.Response, error) {
				return schemasWire(200, io.NopCloser(strings.NewReader(raw))), nil
			})
			value, err := image.New(client).GetImageSchema(context.Background())
			if err != nil || value == nil || value.Body == nil {
				t.Fatal(value, err)
			}
			if mode == "empty" {
				if value.Name == nil || *value.Name != "" || value.Properties == nil || len(value.Properties) != 0 || value.Definitions == nil || len(value.Definitions) != 0 || value.Required == nil || len(value.Required) != 0 {
					t.Fatal("explicit empties were lost", value)
				}
			} else if mode == "required strings" {
				if !reflect.DeepEqual(value.Required, []string{"", "name", "name", "属性"}) {
					t.Fatal("required list interpreted as validation policy", value)
				}
			} else if value.Name != nil || value.Properties != nil || value.Definitions != nil || value.Required != nil || value.AdditionalProperties != nil {
				t.Fatal("missing or null acquired typed defaults", value)
			}
			for _, key := range []string{"name", "properties", "definitions", "required"} {
				_, exists := value.Body[key]
				if mode == "missing" && exists || mode == "null" && string(value.Body[key]) != "null" {
					t.Fatal("raw field presence changed", key, value.Body)
				}
			}
		})
	}
	for _, rawAP := range []string{"false", "true", "null", `{}`, `{"type":"string","unknown":{"large":9007199254740993}}`, `[]`, `[false,9007199254740993]`, `"future-policy"`, "-0", "1.000000000000000000001", "1e1000"} {
		t.Run("passive additionalProperties "+rawAP, func(t *testing.T) {
			raw := `{"additionalProperties":` + rawAP + `}`
			client := schemasClient(func(*http.Request) (*http.Response, error) {
				return schemasWire(200, io.NopCloser(strings.NewReader(raw))), nil
			})
			value, err := image.New(client).GetMetadefNamespaceSchema(context.Background())
			if err != nil || value == nil || string(value.AdditionalProperties) != rawAP || string(value.Body["additionalProperties"]) != rawAP {
				t.Fatal("additionalProperties coerced or schema-validated", value, err)
			}
			value.AdditionalProperties[0] = '!'
			if string(value.Body["additionalProperties"]) != rawAP {
				t.Fatal("additionalProperties aliases Body", value)
			}
		})
	}
	t.Run("canonical values own raw precision and passive inherited keywords", func(t *testing.T) {
		raw := `{"name":"other/route?literal","properties":{"large":9007199254740993,"fraction":1.000000000000000000001,"exponent":1e1000,"false":false,"null":null,"nested":{"$ref":"https://foreign.invalid/definition","type":["string","null"]}},"definitions":{"large":9007199254740993,"empty":{},"null":null},"required":["name","name"],"additionalProperties":{"type":"unknown"},"links":false,"created_at":12,"updated_at":{"future":true},"id":"foreign/id","$schema":"unsupported://dialect","$ref":"https://foreign.invalid/schema","next":"https://foreign.invalid/next","self":"https://foreign.invalid/self","Name":false,"Properties":12,"Definitions":false,"Required":{},"AdditionalProperties":false,"extension":{"n":9007199254740993}}`
		body := &schemasBody{Reader: strings.NewReader(raw)}
		wire := schemasWire(200, body)
		var calls atomic.Int32
		client := schemasClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return wire, nil })
		value, err := image.New(client).GetImageSchema(context.Background())
		if err != nil || value == nil || value.Name == nil || *value.Name != "other/route?literal" || string(value.Properties["large"]) != "9007199254740993" || string(value.Properties["fraction"]) != "1.000000000000000000001" || string(value.Properties["exponent"]) != "1e1000" || string(value.Properties["false"]) != "false" || string(value.Properties["null"]) != "null" || string(value.Definitions["large"]) != "9007199254740993" || string(value.Definitions["null"]) != "null" || !reflect.DeepEqual(value.Required, []string{"name", "name"}) || value.Links != nil || value.CreatedAt != nil || value.UpdatedAt != nil || string(value.Body["links"]) != "false" || string(value.Body["created_at"]) != "12" || string(value.Body["Name"]) != "false" || string(value.Body["AdditionalProperties"]) != "false" || calls.Load() != 1 || body.closes.Load() != 1 {
			t.Fatal(value, err, calls.Load(), body.closes.Load())
		}
		originalProperties, originalAP := string(value.Body["properties"]), string(value.Body["additionalProperties"])
		value.Properties["large"][0] = '!'
		value.Body["definitions"][0] = '!'
		value.AdditionalProperties[0] = '!'
		value.Required[0], *value.Name = "caller", "caller"
		wire.Header.Set("X-Request-Id", "changed")
		if string(value.Body["properties"]) != originalProperties || string(value.Definitions["large"]) != "9007199254740993" || string(value.Body["additionalProperties"]) != originalAP || string(value.Body["required"]) != `["name","name"]` || string(value.Body["name"]) != `"other/route?literal"` || value.Header.Get("X-Request-Id") != "actual-schema" || string(value.Body["extension"]) != `{"n":9007199254740993}` {
			t.Fatal("raw and typed projections alias", value)
		}
	})
	for _, raw := range []string{`{"links":"https://foreign.invalid/schema","created_at":false,"updated_at":[]}`, `{"links":[{"href":"{next}","rel":"next"},null,false],"id":12,"$schema":null}`, `{"Name":12,"Properties":false,"Definitions":[],"Required":12,"AdditionalProperties":false}`} {
		t.Run("passive keywords "+raw, func(t *testing.T) {
			var calls atomic.Int32
			client := schemasClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return schemasWire(200, io.NopCloser(strings.NewReader(raw))), nil
			})
			value, err := image.New(client).GetTasksSchema(context.Background())
			var expected map[string]json.RawMessage
			if err != nil || value == nil || json.Unmarshal([]byte(raw), &expected) != nil || !reflect.DeepEqual(value.Body, expected) || value.Links != nil || value.CreatedAt != nil || value.UpdatedAt != nil || value.Name != nil || value.Properties != nil || value.Required != nil || calls.Load() != 1 {
				t.Fatal("passive keywords interpreted by embedded Metadata", value, err, calls.Load())
			}
		})
	}
}

func TestImageSchemasPreparedOptionsAndLiveSource(t *testing.T) {
	for _, mode := range []string{"nil service", "nil client", "nil provider", "wrong type", "foreign base", "query base", "missing slash", "endpoint UTF8", "base UTF8", "source auth", "source aliases", "source bad value", "source version", "bad microversion", "nil context", "canceled", "nil option", "callback error", "provider replacement", "service replacement", "late unsafe source", "late canceled"} {
		t.Run("preflight "+mode, func(t *testing.T) {
			var calls, callbacks atomic.Int32
			cause := errors.New("preflight cause")
			client := schemasClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, cause })
			ctx := context.Background()
			refCtx, cancel := context.WithCancelCause(ctx)
			defer cancel(nil)
			want, wantCallbacks := error(resource.ErrInvalidOption), int32(0)
			switch mode {
			case "nil client":
				client = nil
			case "nil provider":
				client.ProviderClient = nil
			case "wrong type":
				client.Type, want = "compute", resource.ErrUnsupported
			case "foreign base":
				client.ResourceBase = "https://foreign.invalid/v2/"
			case "query base":
				client.ResourceBase = schemasBase + "?foreign=1"
			case "missing slash":
				client.ResourceBase = strings.TrimSuffix(schemasBase, "/")
			case "endpoint UTF8":
				client.Endpoint = schemasBase + string([]byte{255}) + "/"
			case "base UTF8":
				client.ResourceBase = schemasBase + string([]byte{255}) + "/"
			case "source auth":
				client.MoreHeaders = map[string]string{"x-auth-token": "foreign"}
			case "source aliases":
				client.MoreHeaders = map[string]string{"X-Source": "one", "x-source": "two"}
			case "source bad value":
				client.MoreHeaders = map[string]string{"X-Source": "bad\nvalue"}
			case "source version":
				client.MoreHeaders = map[string]string{"OpenStack-API-Version": "image 2.17"}
			case "bad microversion":
				client.Microversion = "2.17\nforeign"
			case "nil context":
				ctx = nil
			case "canceled":
				cancel(cause)
				ctx, want = refCtx, context.Canceled
			}
			service := image.New(client)
			if mode == "nil service" {
				service = nil
			}
			option := image.GetSchemaOption(func(*image.GetSchemaOpts) error { callbacks.Add(1); return nil })
			switch mode {
			case "nil option":
				option = nil
			case "callback error":
				want, wantCallbacks = cause, 1
				option = func(*image.GetSchemaOpts) error { callbacks.Add(1); return cause }
			case "provider replacement", "service replacement", "late unsafe source", "late canceled":
				wantCallbacks = 1
				option = func(*image.GetSchemaOpts) error {
					callbacks.Add(1)
					if mode == "provider replacement" {
						client.ProviderClient = &gophercloud.ProviderClient{}
					}
					if mode == "service replacement" {
						*service = *image.New(schemasClient(nil))
					}
					if mode == "late unsafe source" {
						client.ResourceBase = "https://foreign.invalid/v2/"
					}
					if mode == "late canceled" {
						cancel(cause)
					}
					return nil
				}
				if mode == "late canceled" {
					ctx, want = refCtx, context.Canceled
				}
			}
			value, err := service.GetImageSchema(ctx, option)
			if value != nil || !errors.Is(err, want) || calls.Load() != 0 || callbacks.Load() != wantCallbacks {
				t.Fatal(value, err, calls.Load(), callbacks.Load())
			}
			if strings.Contains(mode, "canceled") && !errors.Is(err, cause) {
				t.Fatal("custom cancellation cause lost", err)
			}
		})
	}
	for _, header := range []string{"X-Auth-Token", "X-Service-Token", "Authorization", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "Te", "Upgrade", "Accept", "Content-Type", "OpenStack-API-Version", "X-Openstack-Glance-Api-Version", "X-Openstack-Image-Size", "Bad Key"} {
		t.Run("protected option "+header, func(t *testing.T) {
			var calls atomic.Int32
			client := schemasClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
			value, err := image.New(client).GetImageSchema(context.Background(), image.WithGetSchemaHeader(header, "foreign"))
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(value, err, calls.Load())
			}
		})
	}
	for _, headers := range []map[string]string{{"X-Value": "bad\nvalue"}, {"X-Value": string([]byte{255})}, {"X-Alias": "a", "x-alias": "b"}} {
		t.Run("invalid custom header map", func(t *testing.T) {
			var calls atomic.Int32
			client := schemasClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, errors.New("unexpected HTTP") })
			value, err := image.New(client).GetImageSchema(context.Background(), image.WithGetSchemaHeaders(headers))
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(value, err, calls.Load())
			}
		})
	}
	t.Run("full replacement helpers and every callback own snapshots", func(t *testing.T) {
		headers := map[string]string{"X-Kept": "snapshot"}
		full := image.WithGetSchemaOpts(image.GetSchemaOpts{Headers: headers})
		merged := map[string]string{"X-Merged": "snapshot", "x-source": "ordinary"}
		merge := image.WithGetSchemaHeaders(merged)
		headers["X-Kept"], merged["X-Merged"] = "caller", "caller"
		var retained *image.GetSchemaOpts
		var calls, callbacks atomic.Int32
		client := schemasClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Body != nil || r.Header.Get("X-Discarded") != "" || r.Header.Get("X-Kept") != "snapshot" || r.Header.Get("X-Merged") != "snapshot" || r.Header.Get("X-Source") != "ordinary" || r.Header.Get("X-Retained") != "original" || r.Header.Get("X-Last") != "applied" {
				t.Error(r.Header, r.Body)
			}
			return schemasWire(200, io.NopCloser(strings.NewReader(`{}`))), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "source"}
		options := []image.GetSchemaOption{image.WithGetSchemaHeader("X-Discarded", "discarded"), full, merge, nil, nil}
		options[3] = func(c *image.GetSchemaOpts) error {
			callbacks.Add(1)
			retained = c
			c.Headers["X-Retained"] = "original"
			options[4] = nil
			return nil
		}
		options[4] = func(c *image.GetSchemaOpts) error {
			callbacks.Add(1)
			retained.Headers["X-Retained"] = "late"
			return image.WithGetSchemaHeader("X-Last", "applied")(c)
		}
		value, err := image.New(client).GetImageSchema(context.Background(), options...)
		if err != nil || value == nil || calls.Load() != 1 || callbacks.Load() != 2 || client.MoreHeaders["X-Source"] != "source" {
			t.Fatal(value, err, calls.Load(), callbacks.Load(), client.MoreHeaders)
		}
	})
	t.Run("source capture precedes callbacks and token stays live", func(t *testing.T) {
		var calls atomic.Int32
		client := schemasClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.URL.String() != schemasBase+"schemas/image" || r.Method != http.MethodGet || r.Body != nil || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != "after-options" || r.Header.Get("OpenStack-API-Version") != "image 2.10" {
				t.Error(r.URL, r.Method, r.Header, r.Body)
			}
			return schemasWire(200, io.NopCloser(strings.NewReader(`{}`))), nil
		})
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		client.Microversion = "2.10"
		provider := client.ProviderClient
		value, err := image.New(client).GetImageSchema(context.Background(), func(*image.GetSchemaOpts) error {
			client.Endpoint = "https://glance.invalid/later/catalog/"
			client.ResourceBase = "https://glance.invalid/later/v2/"
			client.MoreHeaders = map[string]string{"X-Source": "later"}
			client.Microversion = "2.99"
			client.SetToken("after-options")
			return nil
		})
		if err != nil || value == nil || calls.Load() != 1 || client.ProviderClient != provider || client.MoreHeaders["X-Source"] != "later" {
			t.Fatal(value, err, calls.Load())
		}
	})
	t.Run("standard options reused in parallel across getters", func(t *testing.T) {
		option := image.WithGetSchemaOpts(image.GetSchemaOpts{Headers: map[string]string{"X-Parallel": "owned"}})
		var calls atomic.Int32
		client := schemasClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Header.Get("X-Parallel") != "owned" || r.Body != nil {
				t.Error(r.Header, r.Body)
			}
			return schemasWire(200, io.NopCloser(strings.NewReader(`{}`))), nil
		})
		service := image.New(client)
		var wait sync.WaitGroup
		for _, getter := range schemasGetters()[:6] {
			wait.Add(1)
			go func(g schemasGetter) {
				defer wait.Done()
				value, err := g.get(service, context.Background(), option)
				if err != nil || value == nil {
					t.Error(g.name, value, err)
				}
			}(getter)
		}
		wait.Wait()
		if calls.Load() != 6 {
			t.Fatal(calls.Load())
		}
	})
}

func TestImageSchemasResponseOwnershipAndStatusPolicy(t *testing.T) {
	for _, mode := range []string{"read", "Close", "read Close cancellation", "cancel only"} {
		t.Run("accepted "+mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("schema Read"), errors.New("schema Close"), errors.New("schema cancellation")
			raw := []byte(`{"name":"actual","properties":{}}`)
			body := &schemasBody{Reader: bytes.NewReader(raw)}
			if strings.Contains(mode, "read") {
				body.Reader = schemasReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
			}
			if strings.Contains(mode, "Close") {
				body.closeErr = closeCause
			}
			wire := schemasWire(200, body)
			body.onClose = func() {
				wire.Header.Set("X-Request-Id", "changed during Close")
				if strings.Contains(mode, "cancel") {
					cancel(cancelCause)
				}
			}
			var calls, retries atomic.Int32
			client := schemasClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return wire, nil })
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries.Add(1)
				return errors.New("accepted replay")
			}
			value, err := image.New(client).GetImageSchema(ctx)
			proof := schemasProof(t, err, raw)
			if value != nil || calls.Load() != 1 || retries.Load() != 0 || body.closes.Load() != 1 {
				t.Fatal(value, err, calls.Load(), retries.Load(), body.closes.Load())
			}
			if strings.Contains(mode, "read") && !errors.Is(err, readCause) || strings.Contains(mode, "Close") && !errors.Is(err, closeCause) || strings.Contains(mode, "cancel") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
				t.Fatal("accepted cause lost", err)
			}
			proof.Body[0] = '!'
			proof.Header.Set("X-Request-Id", "caller")
			if raw[0] != '{' || wire.Header.Get("X-Request-Id") != "changed during Close" {
				t.Fatal("response evidence aliases borrowed bytes/header", proof)
			}
		})
	}
	for _, raw := range []string{"", "null", "[]", "true", "12", `"schema"`, "{", `{"name":12}`, `{"name":false}`, `{"name":{}}`, `{"properties":false}`, `{"properties":[]}`, `{"properties":12}`, `{"definitions":[]}`, `{"definitions":"value"}`, `{"required":{}}`, `{"required":"name"}`, `{"required":[null]}`, `{"required":["name",null]}`, `{"required":[12]}`, `{"required":[false]}`, `{"required":[{}]}`, `{"unknown":"` + string([]byte{255}) + `"}`, `{"properties":{"future":"` + string([]byte{255}) + `"}}`, `{"additionalProperties":{"future":"` + string([]byte{255}) + `"}}`} {
		t.Run(fmt.Sprintf("strict model %q", raw), func(t *testing.T) {
			body := &schemasBody{Reader: strings.NewReader(raw)}
			wire := schemasWire(200, body)
			body.onClose = func() { wire.Header.Set("X-Request-Id", "changed during Close") }
			var calls, retries atomic.Int32
			client := schemasClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return wire, nil })
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries.Add(1)
				return errors.New("model replay")
			}
			value, err := image.New(client).GetImageSchema(context.Background())
			schemasProof(t, err, []byte(raw))
			if value != nil || calls.Load() != 1 || retries.Load() != 0 || body.closes.Load() != 1 {
				t.Fatal(value, err, calls.Load(), retries.Load(), body.closes.Load())
			}
		})
	}
	for _, code := range []int{201, 202, 204, 206, 300, 304, 400, 403, 404, 409, 429, 500, 503} {
		t.Run(fmt.Sprint("strict actual status ", code), func(t *testing.T) {
			body := &schemasBody{Reader: strings.NewReader("native raw status proof")}
			var calls atomic.Int32
			client := schemasClient(func(*http.Request) (*http.Response, error) { calls.Add(1); return schemasWire(code, body), nil })
			value, err := image.New(client).GetImageSchema(context.Background())
			var native gophercloud.ErrUnexpectedResponseCode
			var proof *resource.ResponseError
			if value != nil || !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || native.Method != http.MethodGet || native.URL != schemasBase+"schemas/image" || string(native.Body) != "native raw status proof" || native.ResponseHeader.Get("X-Request-Id") != "actual-schema" || errors.As(err, &proof) || calls.Load() != 1 || body.closes.Load() != 1 {
				t.Fatal(value, err, native, calls.Load(), body.closes.Load())
			}
		})
	}
}

func TestImageSchemasProviderHooksAndNativeCompatibility(t *testing.T) {
	t.Run("configured prebody HTTP and transport hooks preserve live auth", func(t *testing.T) {
		var calls, reauth, backoff, retries atomic.Int32
		var bodies []*schemasBody
		transportCause := errors.New("prebody transport cause")
		client := schemasClient(nil)
		client.MoreHeaders = map[string]string{"X-Source": "captured"}
		client.ReauthFunc = func(context.Context) error { reauth.Add(1); client.SetToken("reauth"); return nil }
		client.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
			backoff.Add(1)
			client.SetToken("backoff")
			return nil
		}
		client.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, original error, _ uint) error {
			retries.Add(1)
			if !opts.KeepResponseBody || opts.JSONResponse != nil || opts.JSONBody != nil || opts.RawBody != nil {
				t.Error(opts, original)
			}
			if gophercloud.ResponseCodeIs(original, 503) {
				client.SetToken("retry503")
				return nil
			}
			if errors.Is(original, transportCause) {
				client.SetToken("retrytransport")
				return nil
			}
			return original
		}
		originalRetry, originalProvider := reflect.ValueOf(client.RetryFunc).Pointer(), client.ProviderClient
		client.HTTPClient.Transport = schemasTransport(func(r *http.Request) (*http.Response, error) {
			n := int(calls.Add(1)) - 1
			codes, tokens := []int{401, 429, 503, 0, 200}, []string{"initial", "reauth", "backoff", "retry503", "retrytransport"}
			if n >= len(codes) {
				return nil, errors.New("unexpected replay")
			}
			if r.Method != http.MethodGet || r.URL.String() != schemasBase+"schemas/image" || r.Body != nil || r.Header.Get("X-Source") != "captured" || r.Header.Get("X-Auth-Token") != tokens[n] {
				t.Error(r.Method, r.URL, r.Header, r.Body)
			}
			if codes[n] == 0 {
				return nil, transportCause
			}
			body := &schemasBody{Reader: strings.NewReader(`{"name":"actual"}`)}
			bodies = append(bodies, body)
			return schemasWire(codes[n], body), nil
		})
		value, err := image.New(client).GetImageSchema(context.Background())
		if err != nil || value == nil || value.Name == nil || *value.Name != "actual" || calls.Load() != 5 || reauth.Load() != 1 || backoff.Load() != 1 || retries.Load() != 2 || client.ProviderClient != originalProvider || reflect.ValueOf(client.RetryFunc).Pointer() != originalRetry {
			t.Fatal(value, err, calls.Load(), reauth.Load(), backoff.Load(), retries.Load())
		}
		for _, body := range bodies {
			if body.closes.Load() != 1 {
				t.Fatal(body.closes.Load())
			}
		}
	})
	for _, change := range []string{"JSON null", "JSON object", "KeepResponseBody", "JSONResponse", "RawBody", "unsupported JSONBody", "expanded OkCodes"} {
		t.Run("shared request ownership "+change, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			callbackCause, readCause, closeCause, cancelCause := errors.New("callback"), errors.New("private Read"), errors.New("private Close"), errors.New("private cancellation")
			var calls, retries, borrowedReads atomic.Int32
			borrowed := &schemasBody{Reader: schemasReader(func([]byte) (int, error) { borrowedReads.Add(1); return 0, io.EOF })}
			var bodies []*schemasBody
			client := schemasClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if n > 2 {
					return nil, errors.New("extra schema replay")
				}
				if r.Method != http.MethodGet || r.URL.String() != schemasBase+"schemas/image" || r.Body != nil {
					t.Error(r.Method, r.URL, r.Body)
				}
				code, raw := 503, "original503"
				if n == 2 {
					code, raw = 202, "private202"
				}
				body := &schemasBody{Reader: strings.NewReader(raw)}
				if n == 2 {
					body.Reader = schemasReader(func(p []byte) (int, error) { return copy(p, raw), readCause })
					body.closeErr = closeCause
					body.onClose = func() { cancel(cancelCause) }
				}
				bodies = append(bodies, body)
				return schemasWire(code, body), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, original error, _ uint) error {
				retries.Add(1)
				if !gophercloud.ResponseCodeIs(original, 503) || opts.JSONBody != nil || opts.RawBody != nil || opts.JSONResponse != nil || !opts.KeepResponseBody {
					t.Error(original, opts)
				}
				switch change {
				case "JSON null":
					opts.JSONBody = json.RawMessage("null")
				case "JSON object":
					opts.JSONBody = map[string]any{}
				case "KeepResponseBody":
					opts.KeepResponseBody = false
				case "JSONResponse":
					opts.JSONResponse = new(any)
				case "RawBody":
					opts.RawBody = borrowed
				case "unsupported JSONBody":
					opts.JSONBody = make(chan int)
				case "expanded OkCodes":
					opts.OkCodes = []int{200, 202}
					return nil
				}
				return callbackCause
			}
			original := reflect.ValueOf(client.RetryFunc).Pointer()
			value, err := image.New(client).GetImageSchema(ctx)
			if change == "expanded OkCodes" {
				var native gophercloud.ErrUnexpectedResponseCode
				var proof *resource.ResponseError
				if value != nil || !errors.As(err, &native) || native.Actual != 202 || !reflect.DeepEqual(native.Expected, []int{200}) || native.Method != http.MethodGet || native.URL != schemasBase+"schemas/image" || string(native.Body) != "private202" || native.ResponseHeader.Get("X-Request-Id") != "actual-schema" || errors.As(err, &proof) || calls.Load() != 2 || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) {
					t.Fatal(value, err, native, calls.Load())
				}
			} else if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.Is(err, callbackCause) || !gophercloud.ResponseCodeIs(err, 503) || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
			if change == "unsupported JSONBody" {
				var encoding *json.UnsupportedTypeError
				if !errors.As(err, &encoding) {
					t.Fatal("encoding cause lost", err)
				}
			}
			if retries.Load() != 1 || borrowedReads.Load() != 0 || borrowed.closes.Load() != 0 || reflect.ValueOf(client.RetryFunc).Pointer() != original {
				t.Fatal("provider/borrowed body ownership changed", retries.Load(), borrowedReads.Load(), borrowed.closes.Load())
			}
			for _, body := range bodies {
				if body.closes.Load() != 1 {
					t.Fatal(body.closes.Load())
				}
			}
		})
	}
	for _, mode := range []string{"transport", "callback", "reauth"} {
		t.Run("nested missing "+mode, func(t *testing.T) {
			cause := errors.New("nested missing cause")
			nested := errors.Join(gophercloud.ErrUnexpectedResponseCode{Method: http.MethodGet, URL: schemasBase, Expected: []int{200}, Actual: 404})
			var calls atomic.Int32
			client := schemasClient(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				if mode == "transport" {
					return nil, errors.Join(cause, nested)
				}
				code := 503
				if mode == "reauth" {
					code = 401
				}
				return schemasWire(code, io.NopCloser(strings.NewReader("original failure"))), nil
			})
			if mode == "callback" {
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					return errors.Join(cause, nested)
				}
			}
			if mode == "reauth" {
				client.ReauthFunc = func(context.Context) error { return errors.Join(cause, nested) }
			}
			value, err := image.New(client).GetImageSchema(context.Background())
			if value != nil || err == nil || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
			if mode == "reauth" {
				var native *gophercloud.ErrUnableToReauthenticate
				if !errors.As(err, &native) || !errors.Is(native.ErrReauth, cause) || !errors.Is(native.ErrReauth, nested) || !gophercloud.ResponseCodeIs(native.ErrOriginal, 401) {
					t.Fatal("native direct reauth fields lost", err)
				}
			} else if !errors.Is(err, cause) || !errors.Is(err, nested) || mode == "callback" && !gophercloud.ResponseCodeIs(err, 503) {
				t.Fatal("original/nested causes lost", err)
			}
		})
	}
	for _, mode := range []string{"foreign origin", "other path", "other query", "changed method", "identical target"} {
		t.Run("redirect "+mode, func(t *testing.T) {
			var calls, redirects atomic.Int32
			var bodies []*schemasBody
			target := schemasBase + "schemas/image"
			client := schemasClient(func(r *http.Request) (*http.Response, error) {
				n := calls.Add(1)
				if r.Method != http.MethodGet || r.URL.String() != target || r.Body != nil {
					t.Error("redirect escaped fixed scope", r.Method, r.URL, r.Body)
				}
				body := &schemasBody{Reader: strings.NewReader(`{}`)}
				bodies = append(bodies, body)
				if n == 2 {
					return schemasWire(200, body), nil
				}
				location := target
				if mode == "foreign origin" {
					location = "https://foreign.invalid/schemas/image"
				}
				if mode == "other path" {
					location = schemasBase + "schemas/tasks"
				}
				if mode == "other query" {
					location = target + "?future=1"
				}
				wire := schemasWire(307, body)
				wire.Header.Set("Location", location)
				return wire, nil
			})
			client.HTTPClient.CheckRedirect = func(r *http.Request, _ []*http.Request) error {
				redirects.Add(1)
				if mode == "changed method" {
					r.Method = http.MethodPost
				}
				return nil
			}
			original := reflect.ValueOf(client.HTTPClient.CheckRedirect).Pointer()
			value, err := image.New(client).GetImageSchema(context.Background())
			if mode == "identical target" {
				if err != nil || value == nil || calls.Load() != 2 {
					t.Fatal(value, err, calls.Load())
				}
			} else if value != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 1 {
				t.Fatal(value, err, calls.Load())
			}
			if redirects.Load() != 1 || reflect.ValueOf(client.HTTPClient.CheckRedirect).Pointer() != original {
				t.Fatal("original redirect policy changed", redirects.Load())
			}
			for _, body := range bodies {
				if body.closes.Load() != 1 {
					t.Fatal(body.closes.Load())
				}
			}
		})
	}
	t.Run("existing native schema URI projections stay separate", func(t *testing.T) {
		var calls atomic.Int32
		client := schemasClient(func(r *http.Request) (*http.Response, error) {
			calls.Add(1)
			if r.Method != http.MethodGet || r.Body != nil {
				t.Error(r.Method, r.Body)
			}
			raw := ""
			switch r.URL.String() {
			case schemasBase + "images/fixed":
				raw = `{"id":"fixed","schema":"https://foreign.invalid/image"}`
			case schemasBase + "images/fixed/members/member":
				raw = `{"member_id":"member","schema":"https://foreign.invalid/member"}`
			case schemasBase + "tasks/fixed":
				raw = `{"id":"fixed","schema":"https://foreign.invalid/task"}`
			default:
				t.Error("native metadata auto-fetched schema", r.URL)
			}
			return schemasWire(200, io.NopCloser(strings.NewReader(raw))), nil
		})
		img, err := sdkimages.New(client).Get(context.Background(), "fixed")
		if err != nil || img == nil || img.Schema != "https://foreign.invalid/image" {
			t.Fatal(img, err)
		}
		member, err := sdkmembers.New(client).Get(context.Background(), "fixed", "member")
		if err != nil || member == nil || member.Schema != "https://foreign.invalid/member" {
			t.Fatal(member, err)
		}
		task, err := sdktasks.New(client).Get(context.Background(), "fixed")
		if err != nil || task == nil || task.Schema != "https://foreign.invalid/task" || calls.Load() != 3 {
			t.Fatal(task, err, calls.Load())
		}
	})
}
