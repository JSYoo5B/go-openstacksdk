package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestSchemaRecordsFixedKindsClassViewsAndOwnedWire(t *testing.T) {
	kinds := []struct {
		kind SchemaKind
		path string
		meta bool
	}{
		{SchemaImage, "image", false}, {SchemaImages, "images", false},
		{SchemaMember, "member", false}, {SchemaMembers, "members", false},
		{SchemaTask, "task", false}, {SchemaTasks, "tasks", false},
		{SchemaMetadefNamespace, "metadefs/namespace", true}, {SchemaMetadefNamespaces, "metadefs/namespaces", true},
		{SchemaMetadefObject, "metadefs/object", true}, {SchemaMetadefObjects, "metadefs/objects", true},
		{SchemaMetadefProperty, "metadefs/property", true}, {SchemaMetadefProperties, "metadefs/properties", true},
		{SchemaMetadefResourceType, "metadefs/resource_type", true}, {SchemaMetadefResourceTypes, "metadefs/resource_types", true},
		{SchemaMetadefTag, "metadefs/tag", true}, {SchemaMetadefTags, "metadefs/tags", true},
	}
	const raw = `{"id":900719925474099312345,"name":["passive",null],"additionalProperties":{"limit":1.00000000000000000001},"properties":{"n":{"default":900719925474099312345}},"definitions":{"nullable":null},"required":[1,null,{}],"location":{"server":"foreign"},"self":"https://foreign.test/self","$ref":"https://foreign.test/ref","links":[{"href":"https://foreign.test/next"}],"unknown":false}`
	for _, tc := range kinds {
		t.Run("route "+tc.path, func(t *testing.T) {
			calls, callbacks := 0, 0
			var actualHeader http.Header
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet || req.URL.String() != "https://glance.example/physical/glance/v2/schemas/"+tc.path || req.Body != nil || req.URL.RawQuery != "" || req.Header.Get("X-Auth-Token") != "after-option" || req.Header.Get("OpenStack-API-Version") != "image 2.10" || req.Header.Get("X-Source") != "shared" || req.Header.Get("X-Option") != "owned" {
					t.Fatal(req.Method, req.URL, req.Header, req.Body)
				}
				reply := taskCoreJSON(req, 200, raw)
				reply.Header.Set("Link", `<https://foreign.test/next>; rel="next"`)
				actualHeader = reply.Header
				return reply, nil
			})
			client.ResourceBase = "https://glance.example/physical/glance/v2/"
			client.Microversion = "2.10"
			client.MoreHeaders = map[string]string{"X-Source": "shared"}
			option := func(config *GetSchemaOpts) error {
				callbacks++
				client.SetToken("after-option")
				return WithGetSchemaHeader("X-Option", "owned")(config)
			}
			got, err := New(client).GetSchemaRecord(context.Background(), tc.kind, option)
			if err != nil || got == nil || got.Kind != tc.kind || got.Resource == nil || got.Wire == nil || got.StatusCode != 200 || got.Header.Get("X-Task-Proof") != "actual" || string(got.Envelope) != raw || calls != 1 || callbacks != 1 {
				t.Fatal(got, err, calls, callbacks)
			}
			wantFields := 5
			if tc.meta {
				wantFields = 7
			}
			th.AssertEquals(t, wantFields, len(got.Resource.Body))
			th.AssertEquals(t, "900719925474099312345", string(got.Resource.Body["id"]))
			th.AssertEquals(t, `["passive",null]`, string(got.Resource.Body["name"]))
			th.AssertEquals(t, `{"n":{"default":900719925474099312345}}`, string(got.Resource.Body["properties"]))
			th.AssertEquals(t, "null", string(got.Resource.Body["location"]))
			if tc.meta {
				th.AssertEquals(t, "true", string(got.Resource.Body["additional_properties"]))
				th.AssertEquals(t, `{"nullable":null}`, string(got.Resource.Body["definitions"]))
				th.AssertEquals(t, `[1,null,{}]`, string(got.Resource.Body["required"]))
			} else {
				th.AssertEquals(t, `{"limit":1.00000000000000000001}`, string(got.Resource.Body["additional_properties"]))
				if _, exists := got.Resource.Body["definitions"]; exists {
					t.Fatal("ordinary view gained definitions")
				}
				if _, exists := got.Resource.Body["required"]; exists {
					t.Fatal("ordinary view gained required")
				}
			}
			th.AssertEquals(t, `{"server":"foreign"}`, string(got.Wire.Body["location"]))
			th.AssertEquals(t, `"https://foreign.test/ref"`, string(got.Wire.Body["$ref"]))
			th.AssertEquals(t, `{"limit":1.00000000000000000001}`, string(got.Wire.Body["additionalProperties"]))
			th.AssertEquals(t, false, client.MoreHeaders["X-Option"] != "")
			if got.Resource.StatusCode != 200 || got.Wire.StatusCode != 200 || got.Resource.Header.Get("X-Task-Proof") != "actual" || got.Wire.Header.Get("X-Task-Proof") != "actual" {
				t.Fatal("resource receipts absent", got)
			}
			got.Resource.Body["properties"][0] = '!'
			got.Resource.Header.Set("X-Task-Proof", "resource changed")
			got.Wire.Body["id"][0] = '8'
			got.Wire.Header.Set("X-Task-Proof", "wire changed")
			got.Header.Set("X-Task-Proof", "record changed")
			th.AssertEquals(t, raw, string(got.Envelope))
			got.Envelope[0] = '!'
			if string(got.Wire.Body["properties"]) != `{"n":{"default":900719925474099312345}}` || string(got.Resource.Body["id"]) != "900719925474099312345" || actualHeader.Get("X-Task-Proof") != "actual" {
				t.Fatal("record, view, wire or transport ownership leaked", got)
			}
		})
	}
	for _, tc := range []struct {
		name                                       string
		kind                                       SchemaKind
		raw, ap, properties, definitions, required string
	}{
		{"ordinary missing", SchemaImage, `{}`, "null", "null", "", ""},
		{"meta missing", SchemaMetadefProperty, `{}`, "null", "null", "null", "null"},
		{"ordinary explicit null", SchemaImage, `{"id":null,"name":null,"additionalProperties":null,"properties":null}`, "null", "null", "", ""},
		{"meta explicit null", SchemaMetadefProperty, `{"additionalProperties":null,"properties":null,"definitions":null,"required":null}`, "null", "null", "null", "null"},
		{"ordinary nonobjects", SchemaTask, `{"additionalProperties":[1],"properties":"opaque","definitions":{},"required":["ignored"]}`, "{}", "{}", "", ""},
		{"ordinary false fields", SchemaMember, `{"additionalProperties":false,"properties":42}`, "{}", "{}", "", ""},
		{"meta empty containers false", SchemaMetadefNamespace, `{"additionalProperties":{},"properties":[],"definitions":false,"required":{}}`, "false", "{}", "{}", `[{}]`},
		{"meta numeric zero false", SchemaMetadefTag, `{"additionalProperties":0,"required":false}`, "false", "null", "null", `[false]`},
		// Exact raw JSON truthiness intentionally avoids Python float underflow.
		{"meta exact tiny decimal nonzero", SchemaMetadefTag, `{"additionalProperties":1e-400}`, "true", "null", "null", "null"},
		{"meta signed decimal zero huge exponent", SchemaMetadefTag, `{"additionalProperties":-0.0e+9999}`, "false", "null", "null", "null"},
		{"meta nonempty text true", SchemaMetadefResourceType, `{"additionalProperties":"false","definitions":12,"required":"literal"}`, "true", "null", "{}", `["literal"]`},
		{"meta array truthy mixed required", SchemaMetadefObject, `{"additionalProperties":[null],"required":["x",1,null,{},false]}`, "true", "null", "null", `["x",1,null,{},false]`},
		{"ordinary wire alias later", SchemaImage, `{"additional_properties":{"first":1},"additionalProperties":{"second":2}}`, `{"second":2}`, "null", "", ""},
		{"ordinary canonical alias later", SchemaImage, `{"additionalProperties":{"first":1},"additional_properties":{"second":2}}`, `{"second":2}`, "null", "", ""},
		{"meta wire null later", SchemaMetadefProperty, `{"additional_properties":true,"additionalProperties":null}`, "null", "null", "null", "null"},
		{"meta canonical false later", SchemaMetadefProperty, `{"additionalProperties":"truthy","additional_properties":[]}`, "false", "null", "null", "null"},
		{"duplicate first position wire value last", SchemaImage, `{"additionalProperties":{"old":1},"additional_properties":{"middle":2},"additionalProperties":{"last":3}}`, `{"middle":2}`, "null", "", ""},
		{"duplicate first position canonical value last", SchemaMetadefProperty, `{"additional_properties":0,"additionalProperties":"middle","additional_properties":false}`, "true", "null", "null", "null"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, 200, tc.raw), nil })
			got, err := New(client).GetSchemaRecord(context.Background(), tc.kind)
			if err != nil || got == nil || got.Resource == nil || got.Wire == nil || calls != 1 {
				t.Fatal(got, err, calls)
			}
			th.AssertEquals(t, tc.ap, string(got.Resource.Body["additional_properties"]))
			th.AssertEquals(t, tc.properties, string(got.Resource.Body["properties"]))
			th.AssertEquals(t, tc.definitions, string(got.Resource.Body["definitions"]))
			th.AssertEquals(t, tc.required, string(got.Resource.Body["required"]))
			th.AssertEquals(t, "null", string(got.Resource.Body["id"]))
			th.AssertEquals(t, "null", string(got.Resource.Body["name"]))
			th.AssertEquals(t, tc.raw, string(got.Envelope))
			if tc.name == "duplicate first position wire value last" {
				th.AssertEquals(t, `{"last":3}`, string(got.Wire.Body["additionalProperties"]))
			}
		})
	}
}

func TestSchemaRecordsAcceptedToleranceAndTerminalReceipts(t *testing.T) {
	for _, tc := range []struct {
		name                string
		code                int
		raw                 string
		kind                SchemaKind
		wantWire, wantError bool
	}{
		{"opaque 200 bare ordinary", 200, "opaque schema", SchemaImage, false, false},
		{"malformed 201 bare meta", 201, `{"name":!}`, SchemaMetadefTags, false, false},
		{"empty 204 bare ordinary", 204, "", SchemaImages, false, false},
		{"whitespace bare meta", 202, " \n\t ", SchemaMetadefObjects, false, false},
		{"text plain object 206", 206, `{"name":false}`, SchemaTasks, true, false},
		{"accepted 300 no links followed", 300, `{"name":42,"links":[{"href":"https://foreign.test/next"}]}`, SchemaMembers, true, false},
		{"accepted 399 object", 399, `{}`, SchemaMetadefProperties, true, false},
		{"parsed null fails", 200, "null", SchemaImage, false, true},
		{"parsed array fails", 201, "[]", SchemaMetadefTag, false, true},
		{"parsed string fails", 300, `"schema"`, SchemaTask, false, true},
		{"parsed number fails", 202, "12", SchemaMetadefNamespace, false, true},
		{"parsed boolean fails", 200, "false", SchemaMember, false, true},
		{"invalid UTF8 fails", 200, `{"name":"` + string([]byte{0xff}) + `"}`, SchemaImage, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls, retries := 0, 0
			body := &taskCoreBody{reader: strings.NewReader(tc.raw)}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				reply := taskCoreHTTP(req, tc.code, body)
				reply.Header.Set("Content-Type", "text/plain")
				reply.Header.Set("Link", `<https://foreign.test/next>; rel="next"`)
				return reply, nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			got, err := New(client).GetSchemaRecord(context.Background(), tc.kind)
			if tc.wantError {
				if got != nil || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(got, err)
				}
				taskCoreProof(t, err, tc.code, tc.raw)
			} else {
				if err != nil || got == nil || got.Resource == nil || (got.Wire != nil) != tc.wantWire || got.Kind != tc.kind || got.StatusCode != tc.code || string(got.Envelope) != tc.raw || got.Header.Get("X-Task-Proof") != "actual" {
					t.Fatal(got, err)
				}
				if !tc.wantWire {
					for key, value := range got.Resource.Body {
						if string(value) != "null" {
							t.Fatal("bare default", key, string(value))
						}
					}
				}
				if tc.wantWire && tc.raw == `{"name":false}` {
					th.AssertEquals(t, "false", string(got.Resource.Body["name"]))
				}
			}
			if calls != 1 || retries != 0 || body.closes != 1 {
				t.Fatal("accepted body replay or follow", calls, retries, body.closes)
			}
		})
	}
	t.Run("rejected HTTP keeps original native proof", func(t *testing.T) {
		body := &taskCoreBody{reader: strings.NewReader("forbidden schema original")}
		calls := 0
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, 403, body), nil })
		got, err := New(client).GetSchemaRecord(context.Background(), SchemaImage)
		var native gophercloud.ErrUnexpectedResponseCode
		var receipt *resource.ResponseError
		if got != nil || !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != "forbidden schema original" || native.Method != http.MethodGet || native.URL != "https://glance.example/reverse/glance/v2/schemas/image" || native.ResponseHeader.Get("X-Task-Proof") != "actual" || errors.As(err, &receipt) || calls != 1 || body.closes != 1 {
			t.Fatal(got, err, native, calls, body.closes)
		}
	})
	for _, mode := range []string{"accepted read failure", "accepted Close nested404", "accepted source drift", "accepted cancellation"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("schema body cause")
			reader := &taskCoreReader{body: `{"name":"actual"}`, err: io.EOF}
			body := &taskCoreBody{reader: reader}
			calls, retries := 0, 0
			var client *gophercloud.ServiceClient
			switch mode {
			case "accepted read failure":
				reader.err = cause
			case "accepted Close nested404":
				body.closeErr = errors.Join(cause, gophercloud.ErrUnexpectedResponseCode{Actual: 404, Body: []byte("nested only")})
			case "accepted source drift":
				reader.action = func() { client.Microversion = "2.99" }
			case "accepted cancellation":
				reader.action = func() { cancel(cause) }
			}
			client = taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, 201, body), nil })
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			got, err := New(client).GetSchemaRecord(ctx, SchemaMetadefNamespace)
			if got != nil || err == nil || calls != 1 || retries != 0 || body.closes != 1 || !reader.read {
				t.Fatal(got, err, calls, retries, body.closes)
			}
			if mode == "accepted source drift" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if mode == "accepted cancellation" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			taskCoreProof(t, err, 201, `{"name":"actual"}`)
		})
	}
}

func TestSchemaRecordsLocationOptionsAndPreflightBindings(t *testing.T) {
	t.Run("location captured once before owned options retaining zone", func(t *testing.T) {
		cloud, region, projectName := "configured", "region-one", "current-project"
		facts := resource.CloudLocation{Cloud: &cloud, RegionName: &region, Zone: json.RawMessage(`{"zone":"retained"}`), Project: resource.CloudProject{ID: json.RawMessage("900719925474099312345"), Name: &projectName}}
		wantLocation, err := json.Marshal(facts)
		th.AssertNoErr(t, err)
		locations, callbacks, calls := 0, 0, 0
		headers := map[string]string{"X-Option": "snapshot"}
		option := WithGetSchemaHeaders(headers)
		headers["X-Option"] = "caller changed"
		var retained *GetSchemaOpts
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			if locations != 1 || callbacks != 1 || req.Header.Get("X-Option") != "snapshot" || req.Header.Get("X-Callback") != "owned" {
				t.Fatal(locations, callbacks, req.Header)
			}
			return taskCoreJSON(req, 201, `{"location":{"server":"ignored"},"project_id":"foreign","availability_zone":"foreign","name":12}`), nil
		})
		service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return facts, nil }})
		got, err := service.GetSchemaRecord(context.Background(), SchemaMetadefProperty, option, func(config *GetSchemaOpts) error {
			callbacks++
			if locations != 1 {
				t.Fatal("options before location")
			}
			cloud = "later"
			facts.Zone[0] = '!'
			facts.Project.ID[0] = '8'
			config.Headers["X-Callback"] = "owned"
			retained = config
			return nil
		})
		if err != nil || got == nil || got.Resource == nil || got.Wire == nil || locations != 1 || callbacks != 1 || calls != 1 {
			t.Fatal(got, err, locations, callbacks, calls)
		}
		th.AssertEquals(t, string(wantLocation), string(got.Resource.Body["location"]))
		th.AssertEquals(t, `{"server":"ignored"}`, string(got.Wire.Body["location"]))
		th.AssertEquals(t, "12", string(got.Resource.Body["name"]))
		retained.Headers["X-Callback"] = "retained changed"
		if client.MoreHeaders["X-Callback"] != "" {
			t.Fatal("options mutated source", client.MoreHeaders)
		}
		got.Resource.Body["location"][0] = '!'
		if string(got.Wire.Body["location"]) != `{"server":"ignored"}` || !strings.Contains(string(got.Envelope), `"server":"ignored"`) {
			t.Fatal("computed location aliases actual wire")
		}
	})
	for _, mode := range []string{"nil context", "nil service", "unknown kind", "invalid source", "dependency failure", "invalid location", "callback source drift", "callback binding drift", "callback cancellation", "outer guard"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("preflight schema cause")
			calls, callbacks, locations := 0, 0, 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, 200, `{}`), nil })
			service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
				locations++
				if mode == "dependency failure" {
					return resource.CloudLocation{}, cause
				}
				if mode == "invalid location" {
					return resource.CloudLocation{Zone: json.RawMessage("{")}, nil
				}
				return resource.CloudLocation{}, nil
			}})
			kind := SchemaImage
			if mode == "nil context" {
				ctx = nil
			}
			if mode == "nil service" {
				service = nil
			}
			if mode == "unknown kind" {
				kind = SchemaKind("https://foreign.test/schema")
			}
			if mode == "invalid source" {
				client.MoreHeaders = map[string]string{"X-Auth-Token": "owned"}
			}
			if mode == "outer guard" {
				ctx = rest.WithOperationGuard(ctx, func(context.Context) error { return cause })
			}
			option := func(config *GetSchemaOpts) error {
				callbacks++
				switch mode {
				case "callback source drift":
					client.Microversion = "2.99"
				case "callback binding drift":
					service.API = New(taskCoreClient(nil)).API
				case "callback cancellation":
					cancel(cause)
				}
				return nil
			}
			got, err := service.GetSchemaRecord(ctx, kind, option, func(config *GetSchemaOpts) error { callbacks++; return nil })
			wantCallbacks, wantLocations := 0, 0
			if strings.HasPrefix(mode, "callback ") {
				wantCallbacks, wantLocations = 1, 1
			}
			if mode == "dependency failure" || mode == "invalid location" {
				wantLocations = 1
			}
			if got != nil || err == nil || calls != 0 || callbacks != wantCallbacks || locations != wantLocations {
				t.Fatal(got, err, calls, callbacks, locations, fmt.Sprint(wantCallbacks, wantLocations))
			}
			if mode == "dependency failure" || mode == "callback cancellation" || mode == "outer guard" {
				if !errors.Is(err, cause) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if mode == "callback cancellation" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}
