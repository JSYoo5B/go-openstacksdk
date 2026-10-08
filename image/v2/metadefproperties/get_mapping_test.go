package metadefproperties_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
)

// These assertions protect the stable raw DTO mapping while a Source-shaped
// Resource layer is developed separately. Python descriptor defaults and
// coercion must not silently change the existing leaf Get result.
func TestMetadefPropertyGetMappingDescriptorKeywordsRemainRaw(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{"omitted descriptor defaults", `{}`},
		{"supplied descriptor values and aliases", `{"operators":"and","default":{"amount":9007199254740993},"readonly":"false","is_readonly":false,"minimum":1.75,"maximum":9007199254740993,"enum":"one","pattern":null,"minLength":"03","min_length":9,"maxLength":1e400,"items":false,"uniqueItems":"false","require_unique_items":false,"minItems":null,"maxItems":-1.5,"additionalItems":[],"allow_additional_items":true,"future":{"items":[null,true]}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			body := &propertyBody{Reader: strings.NewReader(tc.raw)}
			client := propertyClient(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != http.MethodGet || r.URL.EscapedPath() != propertyPrefix+propertyCollection+"/"+url.PathEscape(propertyName) || r.URL.RawQuery != "" || r.Body != nil {
					t.Fatalf("unexpected Get request: %s %s", r.Method, r.URL)
				}
				return propertyWire(http.StatusOK, body), nil
			})
			value, err := propertyScope(t, client, propertyParent).Get(context.Background(), propertyName)
			if err != nil || value == nil || calls != 1 || body.closes.Load() != 1 {
				t.Fatalf("Get value=%+v err=%v calls=%d closes=%d", value, err, calls, body.closes.Load())
			}
			var expected map[string]json.RawMessage
			if err := json.Unmarshal([]byte(tc.raw), &expected); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(value.Body, expected) || value.Name != nil || value.Type != nil || value.Title != nil || value.Description != nil || value.Key != nil || value.CreatedAt != nil || value.UpdatedAt != nil || value.Links != nil {
				t.Fatalf("response was seeded or descriptor-normalized: %+v", value)
			}
			if value.StatusCode != http.StatusOK || value.Header.Get("X-Request-Id") != "actual-property" {
				t.Fatalf("missing original response receipt: %+v", value)
			}
		})
	}
}

func TestMetadefPropertyGetMappingDoesNotReuseCachedOrSeededResults(t *testing.T) {
	const firstRaw = `{"name":"passive other","title":"first","self":"https://foreign.invalid/other","schema":"https://foreign.invalid/schema","minimum":1.75}`
	replies := []struct {
		status int
		raw    string
	}{
		{http.StatusOK, firstRaw},
		{http.StatusNotModified, ""},
		{http.StatusOK, `{}`},
	}
	bodies := make([]*propertyBody, 0, len(replies))
	calls := 0
	client := propertyClient(func(r *http.Request) (*http.Response, error) {
		if calls >= len(replies) {
			t.Fatalf("unexpected follow-up request: %s", r.URL)
		}
		if r.Method != http.MethodGet || r.URL.EscapedPath() != propertyPrefix+propertyCollection+"/"+url.PathEscape(propertyName) || r.URL.RawQuery != "" || r.Body != nil || r.Header.Get("If-None-Match") != "" || r.Header.Get("If-Modified-Since") != "" {
			t.Fatalf("request used response routing or cache state: %s %s headers=%v", r.Method, r.URL, r.Header)
		}
		reply := replies[calls]
		calls++
		body := &propertyBody{Reader: strings.NewReader(reply.raw)}
		bodies = append(bodies, body)
		response := propertyWire(reply.status, body)
		response.Header.Set("ETag", "passive-etag")
		response.Header.Set("Location", "https://foreign.invalid/redirect")
		return response, nil
	})
	scope := propertyScope(t, client, propertyParent)
	first, err := scope.Get(context.Background(), propertyName)
	if err != nil || first == nil || first.Name == nil || *first.Name != "passive other" || calls != 1 {
		t.Fatalf("first=%+v err=%v calls=%d", first, err, calls)
	}
	*first.Name = "caller changed name"
	first.Body["minimum"][0] = '9'
	first.Header.Set("ETag", "caller changed etag")
	cached, err := scope.Get(context.Background(), propertyName)
	var native gophercloud.ErrUnexpectedResponseCode
	if cached != nil || !errors.As(err, &native) || native.Actual != http.StatusNotModified || !reflect.DeepEqual(native.Expected, []int{http.StatusOK}) || native.Method != http.MethodGet || native.URL != propertyBase+propertyCollection+"/"+url.PathEscape(propertyName) || len(native.Body) != 0 || native.ResponseHeader.Get("X-Request-Id") != "actual-property" || calls != 2 {
		t.Fatalf("304 reused or lost evidence: value=%+v err=%v native=%+v calls=%d", cached, err, native, calls)
	}
	fresh, err := scope.Get(context.Background(), propertyName)
	if err != nil || fresh == nil || fresh == first || fresh.Name != nil || fresh.Title != nil || fresh.Key != nil || len(fresh.Body) != 0 || fresh.StatusCode != http.StatusOK || calls != 3 {
		t.Fatalf("new response inherited prior fields: value=%+v err=%v calls=%d", fresh, err, calls)
	}
	for index, body := range bodies {
		if body.closes.Load() != 1 {
			t.Fatalf("response%d closes=%d", index, body.closes.Load())
		}
	}
}
