package metadefproperties

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const propertyRecordParent = "OS::空 白"
const propertyRecordChild = " child::名字 "

func propertyRecordScope(t *testing.T, client *gophercloud.ServiceClient, deps Dependencies) *NamespaceScope {
	t.Helper()
	scope, err := NewWithDependencies(client, deps).InNamespace(context.Background(), propertyRecordParent)
	if err != nil {
		t.Fatal(err)
	}
	return scope
}
func propertyRecordSeed(t *testing.T, body string) *resource.RawResource {
	t.Helper()
	var result resource.RawResource
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		t.Fatal(err)
	}
	return &result
}
func propertyRecordDefaults(id string) map[string]string {
	encodedID, _ := json.Marshal(id)
	encodedParent, _ := json.Marshal(propertyRecordParent)
	return map[string]string{
		"id": string(encodedID), "name": "null", "location": "null", "namespace_name": string(encodedParent),
		"type": "null", "title": "null", "description": "null", "operators": "null", "default": "null", "is_readonly": "null", "minimum": "null", "maximum": "null", "enum": "null", "pattern": "null", "min_length": "0", "max_length": "null", "items": "null", "require_unique_items": "false", "min_items": "0", "max_items": "null", "allow_additional_items": "null",
	}
}
func propertyRecordValues(record *Record) map[string]string {
	result := make(map[string]string)
	if record == nil || record.Resource == nil {
		return result
	}
	for key, raw := range record.Resource.Body {
		result[key] = string(raw)
	}
	return result
}

// Keep byte delivery and close counting in the existing shared leaf fixture.
// This adapter adds only the one missing Close callback used by guard checks.
type propertyRecordCloseBody struct {
	*propertyCoreBody
	after func()
}

func (body *propertyRecordCloseBody) Close() error {
	err := body.propertyCoreBody.Close()
	if body.after != nil {
		body.after()
	}
	return err
}

func TestMetadefPropertyRecordFixedRouteDefaultsAndOwnedReceipts(t *testing.T) {
	calls := 0
	body := `{"future":{"number":900719925474099312345},"namespace_name":"wire parent","location":{"cloud":"wire"},"created_at":"wire date","self":"https://foreign.test/"}`
	client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		want := "/reverse/glance/v2/metadefs/namespaces/" + url.PathEscape(propertyRecordParent) + "/properties/" + url.PathEscape(propertyRecordChild)
		if req.Method != http.MethodGet || req.URL.EscapedPath() != want || req.URL.RawQuery != "" || req.Body != nil || req.Header.Get("X-Auth-Token") != "before" || req.Header.Get("Accept") != "application/json" {
			t.Fatal("fixed bodyless route", req.Method, req.URL, req.Header)
		}
		return propertyCoreJSON(req, 203, body), nil
	})
	scope := propertyRecordScope(t, client, Dependencies{})
	record, err := scope.GetRecord(context.Background(), RecordRequest{ID: propertyRecordChild})
	if err != nil || record == nil || record.Namespace != propertyRecordParent || calls != 1 || !reflect.DeepEqual(propertyRecordValues(record), propertyRecordDefaults(propertyRecordChild)) {
		t.Fatal(record, err, calls, propertyRecordValues(record))
	}
	if record.Wire == nil || string(record.Wire.Body["created_at"]) != `"wire date"` || string(record.Wire.Body["namespace_name"]) != `"wire parent"` || string(record.Wire.Body["future"]) != `{"number":900719925474099312345}` {
		t.Fatal("raw response narrowed", record.Wire)
	}
	if record.StatusCode != 203 || record.Resource.StatusCode != 203 || record.Wire.StatusCode != 203 || record.Header.Get("X-Proof") != "original" || record.Resource.Header.Get("X-Proof") != "original" || record.Wire.Header.Get("X-Proof") != "original" || string(record.Envelope) != body {
		t.Fatal("physical receipt missing", record)
	}
	record.Resource.Body["id"][1] = 'X'
	record.Resource.Header.Set("X-Proof", "view changed")
	record.Header.Set("X-Proof", "record changed")
	record.Envelope[0] = 'X'
	if record.Wire.Header.Get("X-Proof") != "original" || string(record.Wire.Body["namespace_name"]) != `"wire parent"` {
		t.Fatal("record channels alias")
	}
}

func TestMetadefPropertyRecordSeedAttributesAndResponseOverlay(t *testing.T) {
	for _, check := range []struct {
		name, seed, wire, wantRoute string
		attrs                       []RecordGetOption
		want                        map[string]string
	}{
		{"resource id retarget", `{"id":"seed-id","name":"seed-name","title":"seed-title","minLength":"04"}`, `{"name":"response-name","title":null}`, "attribute-id", []RecordGetOption{WithRecordGetAttribute("id", "attribute-id"), WithRecordGetAttribute("description", "attribute description")}, map[string]string{"id": `"attribute-id"`, "name": `"response-name"`, "title": "null", "description": `"attribute description"`, "min_length": "4"}},
		{"alternate name retarget", `{"name":"seed-name","title":"seed-title"}`, `{"id":false,"name":{"passive":true}}`, "attribute-name", []RecordGetOption{WithRecordGetAttribute("name", "attribute-name")}, map[string]string{"id": "false", "name": `{"passive":true}`, "title": `"seed-title"`}},
		{"empty resource keyword identity", `{}`, `{}`, "attribute-name", []RecordGetOption{WithRecordGetAttribute("name", "attribute-name"), WithRecordGetAttribute("max_items", "03")}, map[string]string{"id": `"attribute-name"`, "name": `"attribute-name"`, "max_items": "3"}},
		{"canonical seed wins alias", `{"id":"seed-id","min_length":null,"minLength":8,"is_readonly":false,"readonly":"false"}`, `{}`, "seed-id", nil, map[string]string{"min_length": "null", "is_readonly": "false"}},
		{"response alias overlays canonical seed", `{"id":"seed-id","min_length":9,"is_readonly":false}`, `{"minLength":"02","readonly":"false"}`, "seed-id", nil, map[string]string{"min_length": "2", "is_readonly": "true"}},
	} {
		t.Run(check.name, func(t *testing.T) {
			calls := 0
			seed := propertyRecordSeed(t, check.seed)
			seed.Header = http.Header{"X-Old": {"seed header"}}
			seed.StatusCode = 201
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.EscapedPath() != "/reverse/glance/v2/metadefs/namespaces/"+url.PathEscape(propertyRecordParent)+"/properties/"+url.PathEscape(check.wantRoute) || req.URL.RawQuery != "" || req.Body != nil {
					t.Fatal("attributes did not produce fixed query-free route", req.URL)
				}
				return propertyCoreJSON(req, 200, check.wire), nil
			})
			scope := propertyRecordScope(t, client, Dependencies{})
			options := append([]RecordGetOption{}, check.attrs...)
			options = append(options, func(*RecordGetOpts) error {
				seed.Body["name"] = json.RawMessage(`"caller changed"`)
				seed.Body["title"] = json.RawMessage(`"caller changed"`)
				return nil
			})
			record, err := scope.GetRecord(context.Background(), RecordRequest{Resource: seed}, options...)
			if err != nil || record == nil || calls != 1 || len(record.Resource.Body) != 21 {
				t.Fatal(record, err, calls)
			}
			for key, want := range check.want {
				if string(record.Resource.Body[key]) != want {
					t.Fatalf("%s wanted %s got %s", key, want, record.Resource.Body[key])
				}
			}
			if record.Resource.Header.Get("X-Old") != "" || record.Resource.StatusCode != 200 || string(record.Envelope) != check.wire {
				t.Fatal("seed metadata replaced actual receipt", record)
			}
			var raw map[string]json.RawMessage
			if err := json.Unmarshal([]byte(check.wire), &raw); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(record.Wire.Body, raw) {
				t.Fatal("seed or keyword attrs were added to Wire", record.Wire.Body, raw)
			}
		})
	}
}

func TestMetadefPropertyRecordDescriptorsAliasesAndExplicitNull(t *testing.T) {
	for _, check := range []struct {
		name, body string
		want       map[string]string
	}{
		{"raw strings containers and big numbers", `{"name":false,"type":{"custom":true},"title":[1],"description":42,"default":{"number":900719925474099312345},"pattern":false,"minimum":900719925474099312345,"maximum":-3.9}`, map[string]string{"name": "false", "type": `{"custom":true}`, "title": "[1]", "description": "42", "default": `{"number":900719925474099312345}`, "pattern": "false", "minimum": "900719925474099312345", "maximum": "-3"}},
		{"list scalar and dict fallback", `{"operators":"and","enum":{"entry":1},"items":false}`, map[string]string{"operators": `["and"]`, "enum": `[{"entry":1}]`, "items": "{}"}},
		{"list arrays and dict pass through", `{"operators":[null,true],"enum":[900719925474099312345],"items":{"literal":[1]}}`, map[string]string{"operators": "[null,true]", "enum": "[900719925474099312345]", "items": `{"literal":[1]}`}},
		{"int digit float and bool descriptors", `{"minLength":"٠٣","maxLength":" -4 ","minItems":2.8,"maxItems":true,"minimum":[],"maximum":"²notdigits"}`, map[string]string{"min_length": "3", "max_length": "0", "min_items": "2", "max_items": "true", "minimum": "0", "maximum": "0"}},
		{"bool truthiness", `{"readonly":"false","uniqueItems":[],"additionalItems":{"present":false}}`, map[string]string{"is_readonly": "true", "require_unique_items": "false", "allow_additional_items": "true"}},
		{"all explicit descriptor nulls", `{"id":null,"name":null,"minLength":null,"maxLength":null,"minItems":null,"maxItems":null,"uniqueItems":null,"readonly":null,"additionalItems":null,"operators":null,"enum":null,"items":null}`, map[string]string{"id": "null", "min_length": "null", "max_length": "null", "min_items": "null", "max_items": "null", "require_unique_items": "null", "is_readonly": "null", "allow_additional_items": "null", "operators": "null", "enum": "null", "items": "null"}},
		{"wire alias last", `{"is_readonly":false,"readonly":"false","min_length":9,"minLength":"02","require_unique_items":false,"uniqueItems":"false"}`, map[string]string{"is_readonly": "true", "min_length": "2", "require_unique_items": "true"}},
		{"canonical alias last", `{"readonly":"false","is_readonly":false,"minLength":"02","min_length":9,"uniqueItems":"false","require_unique_items":false}`, map[string]string{"is_readonly": "false", "min_length": "9", "require_unique_items": "false"}},
		{"duplicate keeps first position final value", `{"readonly":false,"is_readonly":true,"readonly":null}`, map[string]string{"is_readonly": "true"}},
	} {
		t.Run(check.name, func(t *testing.T) {
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) { return propertyCoreJSON(req, 200, check.body), nil })
			record, err := propertyRecordScope(t, client, Dependencies{}).GetRecord(context.Background(), RecordRequest{ID: propertyRecordChild})
			if err != nil || record == nil || len(record.Resource.Body) != 21 {
				t.Fatal(record, err)
			}
			for key, want := range check.want {
				if string(record.Resource.Body[key]) != want {
					t.Fatalf("%s wanted %s got %s", key, want, record.Resource.Body[key])
				}
			}
			if string(record.Envelope) != check.body {
				t.Fatal("descriptor projection changed envelope")
			}
		})
	}
	for _, body := range []string{`{"minimum":"²"}`, `{"maxLength":1e400}`} {
		t.Run("descriptor failure "+body, func(t *testing.T) {
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) { return propertyCoreJSON(req, 203, body), nil })
			record, err := propertyRecordScope(t, client, Dependencies{}).GetRecord(context.Background(), RecordRequest{ID: propertyRecordChild})
			if record != nil || err == nil {
				t.Fatal(record, err)
			}
			propertyCoreProof(t, err, 203, body)
		})
	}
}

func TestMetadefPropertyRecordStatusParsingAndMissingPolicy(t *testing.T) {
	for _, check := range []struct {
		code         int
		body         string
		good, parsed bool
	}{
		{200, `{}`, true, true}, {201, `{"name":"response"}`, true, true}, {299, `{}`, true, true}, {300, `{}`, true, true}, {399, `{}`, true, true},
		{204, "", true, false}, {304, "not JSON", true, false}, {200, `{"broken":`, true, false},
		{200, `null`, false, true}, {200, `[]`, false, true}, {200, `false`, false, true}, {200, `1`, false, true}, {200, `"text"`, false, true},
		{200, "{\"unknown\":\"\xff\"}", false, true},
	} {
		t.Run(fmt.Sprintf("%d/%q", check.code, check.body), func(t *testing.T) {
			calls := 0
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return propertyCoreJSON(req, check.code, check.body), nil
			})
			record, err := propertyRecordScope(t, client, Dependencies{}).GetRecord(context.Background(), RecordRequest{ID: propertyRecordChild})
			if calls != 1 {
				t.Fatal(calls)
			}
			if check.good {
				if err != nil || record == nil || record.StatusCode != check.code || record.Resource.StatusCode != check.code || string(record.Envelope) != check.body {
					t.Fatal(record, err)
				}
				if check.parsed && record.Wire == nil || !check.parsed && record.Wire != nil {
					t.Fatal("Wire lost parsed-versus-unparsed evidence", record.Wire)
				}
				if !check.parsed && !reflect.DeepEqual(propertyRecordValues(record), propertyRecordDefaults(propertyRecordChild)) {
					t.Fatal("unparsed success lost seed/default view", record.Resource.Body)
				}
			} else {
				if record != nil || err == nil {
					t.Fatal(record, err)
				}
				propertyCoreProof(t, err, check.code, check.body)
			}
		})
	}
	for _, code := range []int{400, 404, 409, 500} {
		t.Run(fmt.Sprintf("native status %d", code), func(t *testing.T) {
			calls := 0
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return propertyCoreJSON(req, code, "native rejection"), nil
			})
			record, err := propertyRecordScope(t, client, Dependencies{}).GetRecord(context.Background(), RecordRequest{ID: propertyRecordChild})
			var native gophercloud.ErrUnexpectedResponseCode
			if record != nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != "native rejection" || native.ResponseHeader.Get("X-Proof") != "original" || calls != 1 {
				t.Fatal(record, err, native, calls)
			}
		})
	}
}

func TestMetadefPropertyRecordInputAndAttributesPreflight(t *testing.T) {
	for _, check := range []struct {
		name    string
		input   RecordRequest
		options []RecordGetOption
	}{
		{"missing input", RecordRequest{}, nil},
		{"two input forms", RecordRequest{ID: "literal", Resource: &resource.RawResource{}}, nil},
		{"unsafe literal", RecordRequest{ID: "bad/name"}, nil},
		{"overlong literal", RecordRequest{ID: strings.Repeat("界", 81)}, nil},
		{"invalid utf8 literal", RecordRequest{ID: string([]byte{0xff})}, nil},
		{"explicit null resource identity", RecordRequest{Resource: propertyRecordSeed(t, `{"id":null,"name":"alias"}`)}, nil},
		{"empty resource identity", RecordRequest{Resource: propertyRecordSeed(t, `{"id":"","name":"alias"}`)}, nil},
		{"nonstring resource identity", RecordRequest{Resource: propertyRecordSeed(t, `{"id":false,"name":"alias"}`)}, nil},
		{"empty resource without attrs", RecordRequest{Resource: propertyRecordSeed(t, `{}`)}, nil},
		{"nil option", RecordRequest{ID: propertyRecordChild}, []RecordGetOption{nil}},
		{"string plus duplicate id attr", RecordRequest{ID: propertyRecordChild}, []RecordGetOption{WithRecordGetAttribute("id", "other")}},
		{"known attr encoding failure", RecordRequest{ID: propertyRecordChild}, []RecordGetOption{WithRecordGetAttribute("minimum", make(chan int))}},
		{"nil semantic option", RecordRequest{ID: propertyRecordChild}, []RecordGetOption{WithRecordGetOpts(RecordGetOpts{Attributes: []resource.ListOption{nil}})}},
		{"nonsemantic option", RecordRequest{ID: propertyRecordChild}, []RecordGetOption{WithRecordGetOpts(RecordGetOpts{Attributes: []resource.ListOption{resource.WithName("other")}})}},
		{"protected auth header", RecordRequest{ID: propertyRecordChild}, []RecordGetOption{WithRecordGetHeader("X-Auth-Token", "foreign")}},
		{"accept header", RecordRequest{ID: propertyRecordChild}, []RecordGetOption{WithRecordGetHeader("Accept", "text/plain")}},
		{"header aliases", RecordRequest{ID: propertyRecordChild}, []RecordGetOption{WithRecordGetHeaders(map[string]string{"X-Alias": "one", "x-alias": "two"})}},
		{"header newline", RecordRequest{ID: propertyRecordChild}, []RecordGetOption{WithRecordGetHeader("X-Option", "bad\nvalue")}},
	} {
		t.Run(check.name, func(t *testing.T) {
			calls := 0
			client := propertyCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })
			record, err := propertyRecordScope(t, client, Dependencies{}).GetRecord(context.Background(), check.input, check.options...)
			if record != nil || err == nil || calls != 0 {
				t.Fatal(record, err, calls)
			}
		})
	}
	for _, key := range []string{"base_path", "namespace_name", "requires_id", "skip_cache", "microversion", "session", "headers", "resource_type", "value"} {
		t.Run("reserved "+key, func(t *testing.T) {
			calls := 0
			client := propertyCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			record, err := propertyRecordScope(t, client, Dependencies{}).GetRecord(context.Background(), RecordRequest{ID: propertyRecordChild}, WithRecordGetAttribute(key, "unsafe"))
			if record != nil || err == nil || calls != 0 {
				t.Fatal(record, err, calls)
			}
		})
	}
}

func TestMetadefPropertyRecordPreflightOrderAndCallbackCauses(t *testing.T) {
	for _, mode := range []string{"nil context", "canceled context", "source retarget", "protected source", "outer failure", "location failure", "invalid location", "callback failure", "callback cancel", "callback retarget"} {
		t.Run(mode, func(t *testing.T) {
			calls, locations, callbacks := 0, 0, 0
			cause := errors.New("preflight cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var use context.Context = ctx
			client := propertyCoreClient(func(*http.Request) (*http.Response, error) { calls++; return nil, nil })
			scope := propertyRecordScope(t, client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
				locations++
				if mode == "location failure" {
					return resource.CloudLocation{}, cause
				}
				if mode == "invalid location" {
					return resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`{`)}}, nil
				}
				return resource.CloudLocation{}, nil
			}})
			switch mode {
			case "nil context":
				use = nil
			case "canceled context":
				cancel(cause)
			case "source retarget":
				client.ResourceBase += "changed/"
			case "protected source":
				client.MoreHeaders = map[string]string{"Authorization": "foreign"}
			case "outer failure":
				use = rest.WithOperationGuard(use, func(context.Context) error { return cause })
			}
			record, err := scope.GetRecord(use, RecordRequest{ID: propertyRecordChild}, func(*RecordGetOpts) error {
				callbacks++
				if mode == "callback failure" {
					return cause
				}
				if mode == "callback cancel" {
					cancel(cause)
				}
				if mode == "callback retarget" {
					client.Endpoint = "https://changed.test/"
				}
				return nil
			})
			if record != nil || err == nil || calls != 0 {
				t.Fatal(record, err, calls)
			}
			if strings.HasPrefix(mode, "callback") {
				if callbacks != 1 || locations != 1 {
					t.Fatal(callbacks, locations)
				}
			} else if callbacks != 0 {
				t.Fatal("invalid preflight invoked options", callbacks)
			}
			if mode == "nil context" || mode == "canceled context" || mode == "source retarget" || mode == "protected source" || mode == "outer failure" {
				if locations != 0 {
					t.Fatal("source/outer guard invoked location", locations)
				}
			}
			if mode == "location failure" || mode == "outer failure" || mode == "callback failure" || mode == "callback cancel" || mode == "canceled context" {
				if !errors.Is(err, cause) {
					t.Fatal("cause lost", err)
				}
			}
			if mode == "callback cancel" || mode == "canceled context" {
				if !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestMetadefPropertyRecordOptionFactoriesAndRetainedStorage(t *testing.T) {
	calls, locationCalls := 0, 0
	cloud := "captured"
	location := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"original project"`)}}
	seed := propertyRecordSeed(t, `{"id":"seed-id","title":"seed-title","location":{"cloud":"seed"}}`)
	client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Full") != "owned" || req.Header.Get("X-Bulk") != "factory" || req.Header.Get("X-Final") != "final" || req.Header.Get("X-Replaced") != "" || req.Header.Get("X-Source") != "before" || req.URL.RawQuery != "" {
			t.Fatal("option ownership", req.URL, req.Header)
		}
		return propertyCoreJSON(req, 200, `{"location":{"cloud":"wire"}}`), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "before"}
	scope := propertyRecordScope(t, client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locationCalls++; return location, nil }})
	headers := map[string]string{"X-Full": "owned"}
	bulk := map[string]string{"X-Bulk": "factory"}
	attributes := map[string]any{"description": "owned description", "unknown": make(chan int)}
	attrSlice := []resource.ListOption{resource.WithFilters(attributes)}
	full := WithRecordGetOpts(RecordGetOpts{Headers: headers, Attributes: attrSlice})
	bulkOption := WithRecordGetHeaders(bulk)
	attributeFactory := WithRecordGetAttributes(map[string]any{"description": "factory description", "unknown": make(chan int)})
	var retained *RecordGetOpts
	options := []RecordGetOption{WithRecordGetHeader("X-Replaced", "old"), full, bulkOption, attributeFactory,
		func(config *RecordGetOpts) error {
			retained = config
			cloud = "option changed"
			location.Project.ID[1] = 'X'
			seed.Body["title"] = json.RawMessage(`"caller changed"`)
			client.MoreHeaders["X-Source"] = "option changed"
			return nil
		},
		func(config *RecordGetOpts) error {
			retained.Headers["X-Full"] = "retained changed"
			retained.Attributes[0] = resource.WithFilter("description", "retained changed")
			return nil
		},
		WithRecordGetHeader("x-final", "final"), WithRecordGetAttribute("description", "last description"),
	}
	headers["X-Full"] = "caller changed"
	bulk["X-Bulk"] = "caller changed"
	attributes["description"] = "caller changed"
	attrSlice[0] = resource.WithFilter("description", "caller changed")
	record, err := scope.GetRecord(context.Background(), RecordRequest{Resource: seed}, options...)
	if err != nil || record == nil || calls != 1 || locationCalls != 1 || string(record.Resource.Body["title"]) != `"seed-title"` || string(record.Resource.Body["description"]) != `"last description"` {
		t.Fatal(record, err, calls, locationCalls)
	}
	var actual resource.CloudLocation
	if err := json.Unmarshal(record.Resource.Body["location"], &actual); err != nil || actual.Cloud == nil || *actual.Cloud != "captured" || string(actual.Project.ID) != `"original project"` {
		t.Fatal("location did not capture before options", actual, err)
	}
	if string(record.Wire.Body["location"]) != `{"cloud":"wire"}` {
		t.Fatal("location normalization changed Wire")
	}
}

func TestMetadefPropertyRecordAcceptedBodyErrorsPreserveAllCauses(t *testing.T) {
	for _, mode := range []string{"read", "close", "read close cancel"} {
		t.Run(mode, func(t *testing.T) {
			calls, retries := 0, 0
			readErr, closeErr, cancelCause := errors.New("read"), errors.New("close"), errors.New("cancel")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			bodyText := `{"title":"response"}`
			body := &propertyCoreBody{reader: strings.NewReader(bodyText)}
			if mode != "close" {
				body.reader = &propertyCoreReader{data: bodyText, err: readErr}
			}
			if mode != "read" {
				body.closeErr = closeErr
			}
			if mode == "read close cancel" {
				body.reader = &propertyCoreReader{data: bodyText, err: readErr, after: func() { cancel(cancelCause) }}
			}
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return propertyCoreHTTP(req, 203, body), nil })
			client.ProviderClient.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			record, err := propertyRecordScope(t, client, Dependencies{}).GetRecord(ctx, RecordRequest{ID: propertyRecordChild})
			if record != nil || err == nil || calls != 1 || retries != 0 || body.closes != 1 {
				t.Fatal(record, err, calls, retries, body.closes)
			}
			propertyCoreProof(t, err, 203, bodyText)
			if mode != "close" && !errors.Is(err, readErr) || mode != "read" && !errors.Is(err, closeErr) {
				t.Fatal("body cause lost", err)
			}
			if mode == "read close cancel" && (!errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled)) {
				t.Fatal("context cause lost", err)
			}
		})
	}
}

func TestMetadefPropertyRecordNativeRetryPoliciesAndOwnership(t *testing.T) {
	for _, mode := range []string{"ordinary retry", "body mutation", "source mutation", "status policy expansion", "reauth"} {
		t.Run(mode, func(t *testing.T) {
			calls, retries, reauths := 0, 0, 0
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.RawQuery != "" || req.Body != nil {
					t.Fatal("native request changed", req.URL, req.Body)
				}
				if calls == 1 {
					if mode == "reauth" {
						return propertyCoreJSON(req, 401, "expired"), nil
					}
					return propertyCoreJSON(req, 503, "busy"), nil
				}
				if mode == "status policy expansion" {
					return propertyCoreJSON(req, 400, "outside original policy"), nil
				}
				if req.Header.Get("X-Auth-Token") != "live" || mode == "ordinary retry" && req.Header.Get("X-Native") != "retry" {
					t.Fatal("native auth/header policy lost", req.Header)
				}
				return propertyCoreJSON(req, 200, `{}`), nil
			})
			scope := propertyRecordScope(t, client, Dependencies{})
			if mode == "reauth" {
				client.ProviderClient.ReauthFunc = func(context.Context) error { reauths++; client.ProviderClient.SetToken("live"); return nil }
			} else {
				client.ProviderClient.SetToken("live")
				client.ProviderClient.RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
					retries++
					if retries > 1 {
						return errors.New("extra retry")
					}
					switch mode {
					case "ordinary retry":
						if opts.MoreHeaders == nil {
							opts.MoreHeaders = map[string]string{}
						}
						opts.MoreHeaders["X-Native"] = "retry"
					case "body mutation":
						opts.JSONBody = map[string]string{"injected": "body"}
					case "source mutation":
						client.Microversion = "2.3"
					case "status policy expansion":
						opts.OkCodes = append(opts.OkCodes, 400)
					}
					return nil
				}
			}
			record, err := scope.GetRecord(context.Background(), RecordRequest{ID: propertyRecordChild})
			switch mode {
			case "ordinary retry":
				if err != nil || record == nil || calls != 2 || retries != 1 {
					t.Fatal(record, err, calls, retries)
				}
			case "reauth":
				if err != nil || record == nil || calls != 2 || reauths != 1 || retries != 0 {
					t.Fatal(record, err, calls, reauths, retries)
				}
			case "status policy expansion":
				var native gophercloud.ErrUnexpectedResponseCode
				if record != nil || !errors.As(err, &native) || native.Actual != 400 || string(native.Body) != "outside original policy" || calls != 2 || retries != 1 {
					t.Fatal(record, err, native, calls, retries)
				}
			default:
				if record != nil || !errors.Is(err, resource.ErrInvalidOption) || !gophercloud.ResponseCodeIs(err, 503) || calls != 1 || retries != 1 {
					t.Fatal(record, err, calls, retries)
				}
			}
		})
	}
}

func TestMetadefPropertyRecordSourceAndOuterGuardAtPhysicalBoundaries(t *testing.T) {
	for _, mode := range []string{"round trip", "read", "close", "outer read", "outer close", "observed source restored", "observed outer restored"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			outerCause := errors.New("outer invariant changed")
			outerFailed := false
			ctx := rest.WithOperationGuard(context.Background(), func(context.Context) error {
				if outerFailed {
					return outerCause
				}
				return nil
			})
			bodyText := `{"name":"response"}`
			var client *gophercloud.ServiceClient
			var body *propertyCoreBody
			client = propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if mode == "round trip" {
					client.Endpoint = "https://changed.test/"
				}
				body = &propertyCoreBody{reader: strings.NewReader(bodyText)}
				if mode == "read" || mode == "outer read" || strings.HasPrefix(mode, "observed") {
					body.reader = &propertyCoreReader{data: bodyText, err: io.EOF, after: func() {
						if strings.Contains(mode, "outer") {
							outerFailed = true
						} else {
							client.ResourceBase += "changed/"
						}
						if strings.HasPrefix(mode, "observed") {
							if observed := rest.CheckOperationGuard(req.Context()); observed == nil {
								t.Error("mutation was not observed by operation guard")
							}
						}
					}}
				}
				if mode == "close" || mode == "outer close" || strings.HasPrefix(mode, "observed") {
					return propertyCoreHTTP(req, 200, &propertyRecordCloseBody{propertyCoreBody: body, after: func() {
						if strings.HasPrefix(mode, "observed") {
							outerFailed = false
							client.ResourceBase = "https://example.test/reverse/glance/v2/"
						} else if mode == "close" {
							client.Type = "compute"
						} else {
							outerFailed = true
						}
					}}), nil
				}
				return propertyCoreHTTP(req, 200, body), nil
			})
			record, err := propertyRecordScope(t, client, Dependencies{}).GetRecord(ctx, RecordRequest{ID: propertyRecordChild})
			if record != nil || err == nil || calls != 1 || body.closes != 1 {
				t.Fatal(record, err, calls, body.closes)
			}
			propertyCoreProof(t, err, 200, bodyText)
			if strings.Contains(mode, "outer") {
				if !errors.Is(err, outerCause) {
					t.Fatal("outer cause lost", err)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("source failure lost", err)
			}
			if strings.HasPrefix(mode, "observed") && (outerFailed || client.ResourceBase != "https://example.test/reverse/glance/v2/") {
				t.Fatal("fixture did not restore observed source")
			}
		})
	}
}

func TestMetadefPropertyRecordLegacyGetRemainsRawAndStrict(t *testing.T) {
	calls := 0
	client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.Query().Get("resource_type") != "legacy extension" {
			t.Fatal("legacy query extension changed", req.URL)
		}
		return propertyCoreJSON(req, 200, `{"minLength":"03","readonly":"false"}`), nil
	})
	value, err := propertyCoreScope(t, client, propertyRecordParent).Get(context.Background(), propertyRecordChild, WithGetResourceType("legacy extension"))
	if err != nil || value == nil || calls != 1 || string(value.Body["minLength"]) != `"03"` || string(value.Body["readonly"]) != `"false"` || value.Name != nil {
		t.Fatal(value, err, calls)
	}
}

func TestMetadefPropertyRecordSupersededAliasDoesNotExposeEncodingFailure(t *testing.T) {
	for _, check := range []struct {
		name  string
		value any
		want  string
	}{
		{"canonical false", false, "false"},
		{"canonical null", nil, "null"},
	} {
		t.Run(check.name, func(t *testing.T) {
			calls := 0
			client := propertyCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return propertyCoreJSON(req, 200, `{}`), nil
			})
			record, err := propertyRecordScope(t, client, Dependencies{}).GetRecord(context.Background(), RecordRequest{ID: propertyRecordChild}, WithRecordGetAttributes(map[string]any{"is_readonly": check.value, "readonly": make(chan int)}))
			if err != nil || record == nil || calls != 1 || string(record.Resource.Body["is_readonly"]) != check.want {
				t.Fatal("superseded alias encoding affected canonical value", record, err, calls)
			}
		})
	}
}
