package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestImageSchemasCoreSixteenFixedBodylessRoutes(t *testing.T) {
	for _, test := range []struct {
		name, path string
		get        func(*Service, context.Context, ...GetSchemaOption) (*Schema, error)
	}{
		{"images", "schemas/images", (*Service).GetImagesSchema},
		{"image", "schemas/image", (*Service).GetImageSchema},
		{"members", "schemas/members", (*Service).GetMembersSchema},
		{"member", "schemas/member", (*Service).GetMemberSchema},
		{"tasks", "schemas/tasks", (*Service).GetTasksSchema},
		{"task", "schemas/task", (*Service).GetTaskSchema},
		{"namespace", "schemas/metadefs/namespace", (*Service).GetMetadefNamespaceSchema},
		{"namespaces", "schemas/metadefs/namespaces", (*Service).GetMetadefNamespacesSchema},
		{"resource_type", "schemas/metadefs/resource_type", (*Service).GetMetadefResourceTypeSchema},
		{"resource_types", "schemas/metadefs/resource_types", (*Service).GetMetadefResourceTypesSchema},
		{"object", "schemas/metadefs/object", (*Service).GetMetadefObjectSchema},
		{"objects", "schemas/metadefs/objects", (*Service).GetMetadefObjectsSchema},
		{"property", "schemas/metadefs/property", (*Service).GetMetadefPropertySchema},
		{"properties", "schemas/metadefs/properties", (*Service).GetMetadefPropertiesSchema},
		{"tag", "schemas/metadefs/tag", (*Service).GetMetadefTagSchema},
		{"tags", "schemas/metadefs/tags", (*Service).GetMetadefTagsSchema},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls int
			header := http.Header{"X-Actual": {"owned"}, "Link": {`<https://foreign.test/page>; rel="next"`}}
			body := &deleteCoreBody{reader: strings.NewReader(`{"name":"../passive-name","properties":{"nested":{"type":"object","next":"https://foreign.test"}},"links":false,"created_at":{},"updated_at":5,"id":"decoy","$schema":"https://foreign.test/schema"}`)}
			client := deleteCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/"+test.path || req.URL.RawQuery != "" || req.Body != nil || req.Header.Get("X-Extra") != "owned" {
					t.Fatal("schema route/body/header changed", req.Method, req.URL, req.Header, req.Body)
				}
				return deleteCoreHTTP(200, body, header), nil
			})
			value, err := test.get(New(client), context.Background(), WithGetSchemaHeader("X-Extra", "owned"))
			if err != nil || value == nil || value.Name == nil || *value.Name != "../passive-name" || value.StatusCode != 200 || value.Header.Get("X-Actual") != "owned" || value.AdditionalProperties != nil || value.Links != nil || value.CreatedAt != nil || value.UpdatedAt != nil || string(value.Body["links"]) != "false" || calls != 1 || body.closes != 1 {
				t.Fatalf("value=%+v err=%v calls=%d closes=%d", value, err, calls, body.closes)
			}
			header.Set("X-Actual", "wire mutation")
			if value.Header.Get("X-Actual") != "owned" {
				t.Fatal("schema header aliases wire")
			}
		})
	}
}

func TestImageSchemasCoreCanonicalPresenceAndIndependentRawFields(t *testing.T) {
	for _, raw := range []string{`{}`, `{"name":null,"properties":null,"definitions":null,"required":null}`, `{"name":"","properties":{},"definitions":{},"required":[]}`} {
		var value Schema
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			t.Fatal(raw, err)
		}
		if strings.Contains(raw, `"name":""`) {
			if value.Name == nil || value.Properties == nil || value.Definitions == nil || value.Required == nil {
				t.Fatal("explicit empty canonical fields lost", value)
			}
		} else if value.Name != nil || value.Properties != nil || value.Definitions != nil || value.Required != nil {
			t.Fatal("missing/null canonical fields not nil", value)
		}
		if value.Body == nil {
			t.Fatal("root fields not owned")
		}
	}
	for _, literal := range []string{`false`, `true`, `null`, `{}`, `[]`, `"literal"`, `9007199254740993`, `1.00000000000000001`} {
		var value Schema
		if err := json.Unmarshal([]byte(`{"additionalProperties":`+literal+`}`), &value); err != nil || string(value.AdditionalProperties) != literal {
			t.Fatal("raw AP coerced", value, err)
		}
		value.AdditionalProperties[0] = '!'
		if string(value.Body["additionalProperties"]) != literal {
			t.Fatal("AP aliases Body bytes")
		}
	}
	raw := []byte(`{"name":"resource_type_association","properties":{"properties":{"type":"object","additionalProperties":{"required":["type"],"large":9007199254740993}}},"definitions":{"extension":{"fraction":1.00000000000000001,"nullable":null}},"required":["name","name",""],"additionalProperties":{"type":"string"},"Name":"decoy","CreatedAt":false,"links":"not a link list","created_at":42}`)
	var value Schema
	if err := json.Unmarshal(raw, &value); err != nil || value.Name == nil || *value.Name != "resource_type_association" || len(value.Required) != 3 || value.Links != nil || value.CreatedAt != nil || !bytes.Contains(value.Properties["properties"], []byte(`"type":"object"`)) || !bytes.Contains(value.Definitions["extension"], []byte("1.00000000000000001")) {
		t.Fatal("nested/passive schema changed", value, err)
	}
	rootProperties := append([]byte(nil), value.Body["properties"]...)
	rootDefinitions := append([]byte(nil), value.Body["definitions"]...)
	value.Properties["properties"][0] = '!'
	value.Definitions["extension"][0] = '!'
	value.Body["name"][1] = '!'
	if !bytes.Equal(value.Body["properties"], rootProperties) || !bytes.Equal(value.Body["definitions"], rootDefinitions) || *value.Name != "resource_type_association" || !bytes.Contains(raw, []byte(`"name":"resource_type_association"`)) {
		t.Fatal("typed/raw schema fields alias")
	}
	var decoys Schema
	if err := json.Unmarshal([]byte(`{"Name":"decoy","Properties":{},"Definitions":{},"Required":[],"AdditionalProperties":false}`), &decoys); err != nil || decoys.Name != nil || decoys.Properties != nil || decoys.Required != nil || decoys.AdditionalProperties != nil {
		t.Fatal("case decoys projected", decoys, err)
	}
}

func TestImageSchemasCoreMalformedAcceptedResponsesKeepProof(t *testing.T) {
	for index, raw := range [][]byte{nil, []byte(`null`), []byte(`[]`), []byte(`false`), []byte(`{"name":true}`), []byte(`{"properties":[]}`), []byte(`{"definitions":"bad"}`), []byte(`{"required":{}}`), []byte(`{"required":[null]}`), []byte(`{"required":["valid",null]}`), []byte(`{"required":[false]}`), []byte(`{"required":[1]}`), []byte(`{"required":[[]]}`), []byte(`{"name":`), []byte("{\"unknown\":\"bad\xff\"}")} {
		t.Run(string(rune('A'+index)), func(t *testing.T) {
			body := &deleteCoreBody{reader: bytes.NewReader(raw)}
			client := deleteCoreClient(func(*http.Request) (*http.Response, error) {
				return deleteCoreHTTP(200, body, http.Header{"X-Actual": {"owned"}}), nil
			})
			value, err := New(client).GetImageSchema(context.Background())
			var proof *resource.ResponseError
			if value != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || !bytes.Equal(proof.Body, raw) || proof.Header.Get("X-Actual") != "owned" || body.closes != 1 {
				t.Fatalf("raw=%q value=%+v err=%v proof=%+v closes=%d", raw, value, err, proof, body.closes)
			}
		})
	}
	var value Schema
	if err := json.Unmarshal([]byte(`{"name":"before","properties":{}}`), &value); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"name":"after","required":[null]}`), &value); err == nil || *value.Name != "before" || value.Properties == nil {
		t.Fatal("failed decode partially updated Schema", value, err)
	}
}

func TestImageSchemasCoreCaptureAndCompletePreflight(t *testing.T) {
	var client *gophercloud.ServiceClient
	var retained *GetSchemaOpts
	var calls int
	client = deleteCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		retained.Headers["X-Extra"] = "late"
		if req.URL.Path != "/reverse/glance/v2/schemas/image" || req.Header.Get("X-Snapshot") != "captured" || req.Header.Get("X-Extra") != "owned" || req.Header.Get("X-Auth-Token") != "latest" {
			t.Fatal("source capture/live auth changed", req.URL, req.Header)
		}
		return deleteCoreHTTP(200, io.NopCloser(strings.NewReader(`{}`)), nil), nil
	})
	client.MoreHeaders = map[string]string{"X-Snapshot": "captured"}
	value, err := New(client).GetImageSchema(context.Background(), func(config *GetSchemaOpts) error {
		retained = config
		config.Headers["X-Extra"] = "owned"
		client.ResourceBase = "https://example.test/later/"
		client.MoreHeaders["X-Snapshot"] = "late"
		client.ProviderClient.SetToken("latest")
		return nil
	})
	if value == nil || err != nil || calls != 1 {
		t.Fatal(value, err, calls)
	}
	for _, mutate := range []func(*gophercloud.ServiceClient){func(c *gophercloud.ServiceClient) { c.ProviderClient = &gophercloud.ProviderClient{} }, func(c *gophercloud.ServiceClient) { c.ResourceBase = "https://foreign.test/v2/" }} {
		client := deleteCoreClient(func(*http.Request) (*http.Response, error) { t.Fatal("retargeted schema request"); return nil, nil })
		value, err := New(client).GetImageSchema(context.Background(), func(*GetSchemaOpts) error { mutate(client); return nil })
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal("source retarget accepted", value, err)
		}
	}
	for _, mutate := range []func(*gophercloud.ServiceClient){func(c *gophercloud.ServiceClient) { c.ProviderClient = nil }, func(c *gophercloud.ServiceClient) { c.Type = "compute" }, func(c *gophercloud.ServiceClient) { c.Endpoint = "relative" }, func(c *gophercloud.ServiceClient) { c.ResourceBase += "?query=true" }, func(c *gophercloud.ServiceClient) { c.MoreHeaders = map[string]string{"X-Auth-Token": "owned"} }, func(c *gophercloud.ServiceClient) { c.Microversion = "bad\nvalue" }} {
		client := deleteCoreClient(func(*http.Request) (*http.Response, error) { t.Fatal("bad source requested"); return nil, nil })
		mutate(client)
		var callbacks int
		value, err := New(client).GetImageSchema(context.Background(), func(*GetSchemaOpts) error { callbacks++; return nil })
		if value != nil || err == nil || callbacks != 0 {
			t.Fatal("source not checked before callback", value, err, callbacks)
		}
	}
	custom := errors.New("custom cancellation")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(custom)
	client = deleteCoreClient(func(*http.Request) (*http.Response, error) { t.Fatal("canceled schema request"); return nil, nil })
	value, err = New(client).GetImageSchema(ctx)
	if value != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, custom) {
		t.Fatal("context causes lost", value, err)
	}
	value, err = New(client).GetImageSchema(nil)
	if value != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("nil context accepted", value, err)
	}
	value, err = (*Service)(nil).GetImageSchema(context.Background())
	if value != nil || !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("nil service accepted", value, err)
	}
}

func TestImageSchemasCoreAcceptedBodyFailuresAndCauses(t *testing.T) {
	readCause, closeCause, customCause := errors.New("read failed"), errors.New("close failed"), errors.New("caller stopped")
	for _, mode := range []string{"read", "close", "context", "all"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			header := http.Header{"X-Actual": {"before"}}
			raw := []byte(`{"name":"partial"}`)
			body := &deleteCoreBody{reader: deleteCoreReader(func(buffer []byte) (int, error) {
				header.Set("X-Actual", "during read")
				if mode == "context" || mode == "all" {
					cancel(customCause)
				}
				if mode == "read" || mode == "all" {
					return copy(buffer, raw), readCause
				}
				return copy(buffer, raw), io.EOF
			})}
			if mode == "close" || mode == "all" {
				body.closeErr = closeCause
			}
			var calls int
			client := deleteCoreClient(func(*http.Request) (*http.Response, error) { calls++; return deleteCoreHTTP(200, body, header), nil })
			client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				t.Fatal("accepted schema replay")
				return nil
			}
			value, err := New(client).GetImageSchema(ctx)
			var proof *resource.ResponseError
			if value != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || !bytes.Equal(proof.Body, raw) || proof.Header.Get("X-Actual") != "before" || calls != 1 || body.closes != 1 {
				t.Fatalf("value=%v err=%v proof=%+v calls=%d closes=%d", value, err, proof, calls, body.closes)
			}
			if mode == "read" || mode == "all" {
				if !errors.Is(err, readCause) {
					t.Fatal("read cause lost", err)
				}
			}
			if mode == "close" || mode == "all" {
				if !errors.Is(err, closeCause) {
					t.Fatal("close cause lost", err)
				}
			}
			if mode == "context" || mode == "all" {
				if !errors.Is(err, customCause) || !errors.Is(err, context.Canceled) {
					t.Fatal("context causes lost", err)
				}
			}
		})
	}
}

func TestImageSchemasCoreStrictStatusAndPrebodyOwnership(t *testing.T) {
	for _, status := range []int{201, 202, 204, 404, 503} {
		client := deleteCoreClient(func(*http.Request) (*http.Response, error) { return deleteCoreHTTP(status, http.NoBody, nil), nil })
		value, err := New(client).GetImageSchema(context.Background())
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || !errors.As(err, &native) || native.Actual != status || !reflect.DeepEqual(native.Expected, []int{200}) {
			t.Fatal("unexpected schema status accepted", status, value, err, native)
		}
	}
	for _, change := range []func(*gophercloud.RequestOpts){func(o *gophercloud.RequestOpts) { o.KeepResponseBody = false }, func(o *gophercloud.RequestOpts) { o.JSONResponse = &struct{}{} }, func(o *gophercloud.RequestOpts) { o.RawBody = strings.NewReader("retargeted") }, func(o *gophercloud.RequestOpts) { o.JSONBody = json.RawMessage(`null`) }} {
		var calls int
		client := deleteCoreClient(func(*http.Request) (*http.Response, error) {
			calls++
			return deleteCoreHTTP(503, http.NoBody, nil), nil
		})
		client.ProviderClient.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, _ error, _ uint) error {
			change(options)
			return nil
		}
		value, err := New(client).GetImageSchema(context.Background())
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || calls != 1 {
			t.Fatal("body ownership changed", value, err, calls)
		}
	}
	closeCause := errors.New("unexpected accepted Close failed")
	unexpected := &deleteCoreBody{reader: strings.NewReader("unexpected bytes"), closeErr: closeCause}
	var calls int
	client := deleteCoreClient(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return deleteCoreHTTP(503, http.NoBody, nil), nil
		}
		return deleteCoreHTTP(202, unexpected, http.Header{"X-Actual": {"owned"}}), nil
	})
	client.ProviderClient.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, _ error, _ uint) error {
		options.OkCodes = append(options.OkCodes, 202)
		return nil
	}
	value, err := New(client).GetImageSchema(context.Background())
	var native gophercloud.ErrUnexpectedResponseCode
	var accepted *resource.ResponseError
	if value != nil || !errors.As(err, &native) || native.Actual != 202 || string(native.Body) != "unexpected bytes" || !reflect.DeepEqual(native.Expected, []int{200}) || errors.As(err, &accepted) || !errors.Is(err, closeCause) || unexpected.closes != 1 || calls != 2 {
		t.Fatal("original status policy lost", value, err, native, calls)
	}
}
