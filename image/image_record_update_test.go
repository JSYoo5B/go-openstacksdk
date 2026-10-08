package image

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
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

// Decode the actual request using a test-owned shape. Expected paths/values
// below come from the pinned raw component contract, not the SDK planner.
type imageRecordUpdateWirePatch struct {
	Op    string          `json:"op"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value,omitempty"`
}

func imageRecordUpdatePayload(t *testing.T, req *http.Request) []imageRecordUpdateWirePatch {
	t.Helper()
	raw, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	var patches []imageRecordUpdateWirePatch
	if err := json.Unmarshal(raw, &patches); err != nil || patches == nil {
		t.Fatal("PATCH must be a JSON array", string(raw), err)
	}
	return patches
}

func imageRecordUpdateCheck(t *testing.T, got []imageRecordUpdateWirePatch, want string) {
	t.Helper()
	var expected []imageRecordUpdateWirePatch
	if err := json.Unmarshal([]byte(want), &expected); err != nil {
		t.Fatal(err)
	}
	th.CheckDeepEquals(t, expected, got)
}

func imageRecordUpdateFetched(t *testing.T, service *Service, raw string, handler *taskCoreTransport) *ImageRecord {
	t.Helper()
	*handler = func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Fatal("seed must be fetched once", req.Method)
		}
		response := taskCoreJSON(req, 203, raw)
		response.Header.Set("OpenStack-image-import-methods", "web-download, glance-direct")
		return response, nil
	}
	got, err := service.GetImageRecord(context.Background(), ImageRecordRequest{ID: "fixed"})
	if err != nil || got == nil {
		t.Fatal(got, err)
	}
	return got
}

func TestImageRecordUpdateLiteralSparseAutomaticPatchAndReceipts(t *testing.T) {
	const id = "a /한:%?\\b"
	const responseBody = `{"name":"server name","owner":"server owner","protected":false,"vendor":900719925474099312345}`
	calls := 0
	var responseHeader http.Header
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Method != http.MethodPatch || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(id) || req.URL.RawQuery != "" || req.Header.Get("Accept") != "" || req.Header.Get("Content-Type") != "application/openstack-images-v2.1-json-patch" {
			t.Fatal(req.Method, req.URL, req.Header)
		}
		imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"add","path":"/name","value":null},{"op":"add","path":"/os_hidden","value":"false"},{"op":"add","path":"/owner","value":"client owner"},{"op":"add","path":"/protected","value":"false"},{"op":"add","path":"/size","value":"04"},{"op":"add","path":"/tags","value":[]}]`)
		reply := taskCoreJSON(req, 203, responseBody)
		reply.Header.Set("OpenStack-image-import-methods", "first,, last")
		responseHeader = reply.Header
		return reply, nil
	})
	attrs := map[string]any{"name": nil, "is_hidden": "false", "owner_id": "client owner", "is_protected": "false", "size": "04", "tags": []string{}}
	got, err := New(client).UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{ID: id, Attributes: attrs})
	if err != nil || got == nil || calls != 1 || got.Wire == nil || got.StatusCode != 203 || string(got.Envelope) != responseBody || got.Header.Get("X-Task-Proof") != "actual" {
		t.Fatal(got, err, calls)
	}
	th.AssertEquals(t, `"a /한:%?\\b"`, string(got.Resource.Body["id"]))
	th.AssertEquals(t, `"server name"`, string(got.Resource.Body["name"]))
	th.AssertEquals(t, `"server owner"`, string(got.Resource.Body["owner_id"]))
	th.AssertEquals(t, "false", string(got.Resource.Body["is_protected"]))
	th.AssertEquals(t, "true", string(got.Resource.Body["is_hidden"]))
	th.AssertEquals(t, "4", string(got.Resource.Body["size"]))
	th.AssertEquals(t, "900719925474099312345", string(imageRecordProperties(t, got)["vendor"]))
	th.CheckDeepEquals(t, []string{"first", "", " last"}, got.ImportMethods)
	if attrs["size"] != "04" || len(got.Resource.Body) != 65 || got.Resource.Header.Get("X-Task-Proof") != "actual" || got.Wire.StatusCode != 203 {
		t.Fatal(got, attrs)
	}
	got.Header.Set("X-Task-Proof", "record changed")
	got.Resource.Header.Set("X-Task-Proof", "view changed")
	got.Wire.Header.Set("X-Task-Proof", "wire changed")
	got.Envelope[0] = '!'
	got.Resource.Body["name"][1] = 'X'
	if responseHeader.Get("X-Task-Proof") != "actual" || string(got.Wire.Body["name"]) != `"server name"` {
		t.Fatal("response channels alias", got)
	}
}

func TestImageRecordUpdateNoopUsesRawPresenceAndPythonEquality(t *testing.T) {
	for _, test := range []struct {
		name, raw            string
		attrs                map[string]any
		viewField, viewValue string
	}{
		{"literal without attrs", "", nil, "tags", `[]`},
		{"missing projected defaults are passive", `{"id":"fixed"}`, nil, "tags", `[]`},
		{"present null equals null", `{"id":"fixed","name":null}`, map[string]any{"name": nil}, "name", `null`},
		{"raw bool string equals original string", `{"id":"fixed","protected":"false"}`, map[string]any{"is_protected": "false"}, "is_protected", `true`},
		{"raw numeric string equals original string", `{"id":"fixed","size":"04"}`, map[string]any{"size": "04"}, "size", `4`},
		{"raw scalar tags equals original scalar", `{"id":"fixed","tags":"one"}`, map[string]any{"tags": "one"}, "tags", `["one"]`},
		{"true equals one", `{"id":"fixed","protected":true}`, map[string]any{"is_protected": 1}, "is_protected", `true`},
		{"false equals zero", `{"id":"fixed","protected":false}`, map[string]any{"is_protected": 0}, "is_protected", `false`},
		{"recursive bool numeric equality", `{"id":"fixed","properties":{"nested":[true,{"n":false}],"decimal":1.0}}`, map[string]any{"properties": json.RawMessage(`{"decimal":1,"nested":[1,{"n":0}]}`)}, "properties", `{"decimal":1.0,"nested":[true,{"n":false}]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var handler taskCoreTransport
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			input := ImageRecordUpdateRequest{ID: "fixed", Attributes: test.attrs}
			var seed *ImageRecord
			if test.raw != "" {
				seed = imageRecordUpdateFetched(t, service, test.raw, &handler)
				input.ID, input.Record = "", seed
			}
			beforeCalls := calls
			handler = func(req *http.Request) (*http.Response, error) {
				t.Fatal("clean raw body issued HTTP", req.Method, req.URL)
				return nil, nil
			}
			got, err := service.UpdateImageRecord(context.Background(), input)
			if got == nil || err != nil || calls != beforeCalls || got == seed || got.ImportMethods == nil || len(got.ImportMethods) != 0 {
				t.Fatal(got, err, calls, beforeCalls)
			}
			if test.viewField == "properties" {
				var expected map[string]json.RawMessage
				if err := json.Unmarshal([]byte(test.viewValue), &expected); err != nil {
					t.Fatal(err)
				}
				th.CheckDeepEquals(t, expected, imageRecordProperties(t, got))
			} else {
				th.AssertEquals(t, test.viewValue, string(got.Resource.Body[test.viewField]))
			}
			if seed != nil {
				if got.Resource == seed.Resource || got.Wire == seed.Wire || got.StatusCode != 203 || string(got.Envelope) != test.raw || len(seed.ImportMethods) != 2 {
					t.Fatal("noop replaced or aliased receipt", got, seed)
				}
				got.Resource.Body["id"][1] = 'X'
				th.AssertEquals(t, `"fixed"`, string(seed.Resource.Body["id"]))
			} else if got.StatusCode != 0 || got.Wire != nil || got.Envelope != nil {
				t.Fatal("literal noop invented a fetched receipt", got)
			}
		})
	}
}

func TestImageRecordUpdateMissingNullAndGetterConversionsCreateRawChanges(t *testing.T) {
	for _, test := range []struct {
		name, seed, expected string
		attrs                map[string]any
	}{
		{"missing null is added", `{"id":"fixed"}`, `[{"op":"add","path":"/name","value":null}]`, map[string]any{"name": nil}},
		{"missing default list is added", `{"id":"fixed"}`, `[{"op":"add","path":"/tags","value":[]}]`, map[string]any{"tags": []any{}}},
		{"bool getter does not replace original", `{"id":"fixed","protected":"false"}`, `[{"op":"replace","path":"/protected","value":true}]`, map[string]any{"is_protected": true}},
		{"integer getter does not replace original", `{"id":"fixed","size":"04"}`, `[{"op":"replace","path":"/size","value":4}]`, map[string]any{"size": 4}},
		{"list getter does not replace original", `{"id":"fixed","tags":"one"}`, `[{"op":"replace","path":"/tags","value":["one"]}]`, map[string]any{"tags": []string{"one"}}},
		{"nullable value replaces present value", `{"id":"fixed","name":"before"}`, `[{"op":"replace","path":"/name","value":null}]`, map[string]any{"name": nil}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var handler taskCoreTransport
			calls := 0
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) }))
			seed := imageRecordUpdateFetched(t, service, test.seed, &handler)
			handler = func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPatch {
					t.Fatal(req.Method)
				}
				imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), test.expected)
				return taskCoreJSON(req, 200, `{}`), nil
			}
			got, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: seed, Attributes: test.attrs})
			if got == nil || err != nil || calls != 2 || string(seed.Envelope) != test.seed {
				t.Fatal(got, err, calls, seed)
			}
		})
	}
}

func TestImageRecordUpdatePropertyReplacementFlatteningAndPointerEscapes(t *testing.T) {
	for _, test := range []struct {
		name, seed, expected, id string
		attrs                    map[string]any
	}{
		{"incoming properties replace old component", `{"id":"fixed","properties":{"keep":1,"drop":2}}`, `[{"op":"remove","path":"/drop"},{"op":"replace","path":"/keep","value":3}]`, "fixed", map[string]any{"properties": map[string]any{"keep": 3}}},
		{"unknown attrs replace property component", `{"id":"fixed","properties":{"old":1}}`, `[{"op":"add","path":"/new","value":true},{"op":"remove","path":"/old"}]`, "fixed", map[string]any{"new": true}},
		{"unknown attrs overlay incoming properties", `{"id":"fixed"}`, `[{"op":"add","path":"/keep","value":null},{"op":"add","path":"/vendor","value":false}]`, "fixed", map[string]any{"properties": map[string]any{"vendor": "before", "keep": nil}, "vendor": false}},
		{"property names use JSON pointer escaping", `{"id":"fixed"}`, `[{"op":"add","path":"/","value":null},{"op":"add","path":"/a~1b~0c","value":900719925474099312345}]`, "fixed", map[string]any{"properties": json.RawMessage(`{"":null,"a/b~c":900719925474099312345}`)}},
		{"property id changes patch not route", `{"id":"fixed"}`, `[{"op":"replace","path":"/id","value":"body decoy"},{"op":"add","path":"/name","value":"nested override"}]`, "fixed", map[string]any{"name": "declared", "properties": map[string]any{"id": "body decoy", "name": "nested override"}}},
		{"falsy properties removes flattened fields", `{"id":"fixed","properties":{"old":1}}`, `[{"op":"remove","path":"/old"}]`, "fixed", map[string]any{"properties": map[string]any{}}},
		{"nested object diff", `{"id":"fixed","properties":{"vendor":{"drop":1,"change":2}}}`, `[{"op":"replace","path":"/vendor/change","value":3},{"op":"remove","path":"/vendor/drop"},{"op":"add","path":"/vendor/new","value":null}]`, "fixed", map[string]any{"properties": json.RawMessage(`{"vendor":{"change":3,"new":null}}`)}},
		{"truthy string properties remain a wire field", `{"id":"fixed"}`, `[{"op":"add","path":"/properties","value":"opaque properties"}]`, "fixed", map[string]any{"properties": "opaque properties"}},
		{"null properties remain dirty with empty diff", `{"id":"fixed"}`, `[]`, "fixed", map[string]any{"properties": nil}},
		{"truthy list properties remain dirty with empty diff", `{"id":"fixed"}`, `[]`, "fixed", map[string]any{"properties": []string{"ignored by flattening"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var handler taskCoreTransport
			calls := 0
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) }))
			seed := imageRecordUpdateFetched(t, service, test.seed, &handler)
			handler = func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPatch || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(test.id) {
					t.Fatal(req.Method, req.URL)
				}
				imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), test.expected)
				return taskCoreJSON(req, 203, `{"name":"returned"}`), nil
			}
			got, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: seed, Attributes: test.attrs})
			if got == nil || err != nil || calls != 2 || string(seed.Envelope) != test.seed {
				t.Fatal(got, err, calls)
			}
			th.AssertEquals(t, `"returned"`, string(got.Resource.Body["name"]))
			th.CheckDeepEquals(t, map[string]json.RawMessage{}, imageRecordProperties(t, got))
		})
	}
}

func TestImageRecordUpdateReusesPendingInvalidJSONGetWithoutSyntheticDefaults(t *testing.T) {
	for _, invalidBody := range []string{"opaque GET body", ""} {
		t.Run(fmt.Sprintf("get=%q", invalidBody), func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed" || req.URL.RawQuery != "" {
					t.Fatal(req.URL)
				}
				if calls == 1 {
					if req.Method != http.MethodGet || req.Body != nil {
						t.Fatal(req.Method, req.Body)
					}
					return taskCoreJSON(req, 203, invalidBody), nil
				}
				if calls != 2 || req.Method != http.MethodPatch {
					t.Fatal("pending reuse performed discovery or repeated HTTP", calls, req.Method)
				}
				imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"add","path":"/name","value":"pending name"},{"op":"add","path":"/vendor","value":900719925474099312345}]`)
				return taskCoreJSON(req, 201, `{}`), nil
			})
			service := New(client)
			pending, err := service.GetImageRecord(context.Background(), ImageRecordRequest{ID: "fixed", Attributes: map[string]any{"name": "pending name", "vendor": json.RawMessage(`900719925474099312345`)}})
			if pending == nil || err != nil || pending.Wire != nil || pending.StatusCode != 203 || calls != 1 {
				t.Fatal(pending, err, calls)
			}
			th.AssertEquals(t, `[]`, string(pending.Resource.Body["tags"]))
			th.AssertEquals(t, `null`, string(pending.Resource.Body["is_hidden"]))
			pending.Resource.Body["name"] = json.RawMessage(`"public view edit"`)
			got, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: pending})
			if got == nil || err != nil || calls != 2 || got.StatusCode != 201 || got.Wire == nil || string(got.Resource.Body["name"]) != `"pending name"` || string(pending.Envelope) != invalidBody {
				t.Fatal(got, err, calls, pending)
			}
			clean, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: got})
			if clean == nil || err != nil || calls != 2 {
				t.Fatal("valid PATCH response did not clean pending GET state", clean, err, calls)
			}
		})
	}
}

func TestImageRecordUpdateConsumesPlainImportMethodsWithoutBodyCommit(t *testing.T) {
	var handler taskCoreTransport
	calls := 0
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) }))
	seed := imageRecordUpdateFetched(t, service, `{"id":"fixed","name":"before"}`, &handler)
	handler = func(req *http.Request) (*http.Response, error) {
		t.Fatal("plain header attribute created Body dirtiness", req.Method, req.URL)
		return nil, nil
	}
	got, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: seed}, WithImageRecordAttribute("OpenStack-image-import-methods", "new,, untrimmed"))
	if got == nil || err != nil || calls != 1 || got.StatusCode != 203 || got.Header.Get("OpenStack-image-import-methods") != seed.Header.Get("OpenStack-image-import-methods") {
		t.Fatal(got, err, calls)
	}
	th.CheckDeepEquals(t, []string{"new", "", " untrimmed"}, got.ImportMethods)
	th.CheckDeepEquals(t, []string{"web-download", " glance-direct"}, seed.ImportMethods)
	cleared, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: got})
	if cleared == nil || err != nil || cleared.ImportMethods == nil || len(cleared.ImportMethods) != 0 || calls != 1 {
		t.Fatal(cleared, err, calls)
	}
	if _, present := got.bodyState.current["OpenStack-image-import-methods"]; present {
		t.Fatal("plain attribute entered raw Body", got.bodyState.current)
	}
}

func TestImageRecordUpdateIDOnlyRetargetAndPrivateOriginal(t *testing.T) {
	var handler taskCoreTransport
	calls := 0
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) }))
	seed := imageRecordUpdateFetched(t, service, `{"id":"fixed","name":"before"}`, &handler)
	seed.Resource.Body["name"] = json.RawMessage(`"public view edit"`)
	seed.Wire.Body["name"] = json.RawMessage(`"wire edit"`)
	handler = func(req *http.Request) (*http.Response, error) {
		t.Fatal("id-only dirty field must be discarded", req.Method)
		return nil, nil
	}
	retargeted, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: seed, Attributes: map[string]any{"id": "after/id"}})
	if err != nil || retargeted == nil || calls != 1 {
		t.Fatal(retargeted, err, calls)
	}
	th.AssertEquals(t, `"after/id"`, string(retargeted.Resource.Body["id"]))
	th.AssertEquals(t, `"before"`, string(retargeted.Resource.Body["name"]))
	th.AssertEquals(t, `"fixed"`, string(seed.Resource.Body["id"]))
	handler = func(req *http.Request) (*http.Response, error) {
		if req.URL.EscapedPath() != "/reverse/glance/v2/images/after%2Fid" || req.Method != http.MethodPatch {
			t.Fatal(req.Method, req.URL)
		}
		imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"replace","path":"/id","value":"after/id"},{"op":"replace","path":"/name","value":"after"}]`)
		return taskCoreJSON(req, 200, `{}`), nil
	}
	got, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: retargeted}, WithImageRecordAttribute("name", "after"))
	if got == nil || err != nil || calls != 2 {
		t.Fatal(got, err, calls)
	}
}

func TestImageRecordUpdateValidResponseCleansAndInvalidSyntaxKeepsPending(t *testing.T) {
	for _, reply := range []string{`{"name":"server"}`, `{}`, `not JSON`, `{"broken":`, "", " \n\t "} {
		t.Run(fmt.Sprintf("response=%q", reply), func(t *testing.T) {
			calls := 0
			patches := [][]imageRecordUpdateWirePatch{}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodPatch {
					t.Fatal(req.Method)
				}
				patches = append(patches, imageRecordUpdatePayload(t, req))
				if calls == 1 {
					return taskCoreJSON(req, 299, reply), nil
				}
				return taskCoreJSON(req, 200, `{}`), nil
			})
			service := New(client)
			pending, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{ID: "fixed", Attributes: map[string]any{"name": "requested"}})
			if pending == nil || err != nil || calls != 1 || pending.StatusCode != 299 || string(pending.Envelope) != reply {
				t.Fatal(pending, err, calls)
			}
			got, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: pending})
			if got == nil || err != nil {
				t.Fatal(got, err)
			}
			if json.Valid([]byte(reply)) {
				if calls != 1 || got.StatusCode != 299 || got.Wire == nil {
					t.Fatal("valid body failed to clean raw baseline", got, calls)
				}
			} else {
				if calls != 2 || pending.Wire != nil || got.StatusCode != 200 {
					t.Fatal("invalid syntax lost pending dirty state", got, calls)
				}
				th.CheckDeepEquals(t, patches[0], patches[1])
				clean, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: got})
				if clean == nil || err != nil || calls != 2 {
					t.Fatal("valid retry did not clean", clean, err, calls)
				}
			}
		})
	}
	t.Run("sticky dirty survives returning to original", func(t *testing.T) {
		var handler taskCoreTransport
		calls := 0
		service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) }))
		seed := imageRecordUpdateFetched(t, service, `{"id":"fixed","name":"before"}`, &handler)
		handler = func(req *http.Request) (*http.Response, error) {
			imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"replace","path":"/name","value":"after"}]`)
			return taskCoreJSON(req, 200, `opaque`), nil
		}
		pending, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: seed, Attributes: map[string]any{"name": "after"}})
		if pending == nil || err != nil || calls != 2 {
			t.Fatal(pending, err, calls)
		}
		handler = func(req *http.Request) (*http.Response, error) {
			imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[]`)
			return taskCoreJSON(req, 200, `{}`), nil
		}
		got, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: pending, Attributes: map[string]any{"name": "before"}})
		if got == nil || err != nil || calls != 3 {
			t.Fatal("empty diff suppressed a required dirty commit", got, err, calls)
		}
	})
}

func TestImageRecordUpdateSourceOptionsAndAttributeSnapshots(t *testing.T) {
	headers := map[string]string{"X-Option": "factory"}
	attrs := map[string]any{"name": "factory name", "size": "04"}
	option := WithImageRecordOpts(ImageRecordOpts{Headers: headers, Attributes: attrs})
	headers["X-Option"], attrs["name"] = "caller changed", "caller changed"
	cloud := "captured"
	facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"token project"`)}}
	calls, callbacks, locations := 0, 0, 0
	var retained *ImageRecordOpts
	var client *gophercloud.ServiceClient
	client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Source") != "captured" || req.Header.Get("X-Option") != "factory" || req.Header.Get("X-Final") != "yes" || req.Header.Get("X-Auth-Token") != "live" || req.Header.Get("OpenStack-API-Version") != "image 2.10" {
			t.Fatal(req.Header)
		}
		imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"add","path":"/name","value":"factory name"},{"op":"add","path":"/size","value":"04"}]`)
		retained.Attributes["name"] = "retained changed"
		retained.Headers["X-Option"] = "retained changed"
		return taskCoreJSON(req, 200, `{}`), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "captured"}
	client.Microversion = "2.10"
	service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
		locations++
		client.MoreHeaders["X-Source"] = "callback changed"
		return facts, nil
	}})
	got, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{ID: "fixed"}, option, func(config *ImageRecordOpts) error {
		callbacks++
		retained = config
		cloud = "after options"
		facts.Project.ID[1] = 'X'
		client.SetToken("live")
		return WithImageRecordHeader("X-Final", "yes")(config)
	})
	if got == nil || err != nil || calls != 1 || callbacks != 1 || locations != 1 {
		t.Fatal(got, err, calls, callbacks, locations)
	}
	th.AssertEquals(t, `"factory name"`, string(got.Resource.Body["name"]))
	th.AssertEquals(t, "4", string(got.Resource.Body["size"]))
	var captured resource.CloudLocation
	if err := json.Unmarshal(got.Resource.Body["location"], &captured); err != nil || captured.Cloud == nil || *captured.Cloud != "captured" || string(captured.Project.ID) != `"token project"` {
		t.Fatal(captured, err)
	}
}

func TestImageRecordUpdateRetryKeepsOwnedPatchHeadersAndLiveAuthentication(t *testing.T) {
	for _, spoof := range []bool{true, false} {
		t.Run(fmt.Sprintf("spoof=%v", spoof), func(t *testing.T) {
			calls, callbacks := 0, 0
			var client *gophercloud.ServiceClient
			bodies := []*taskCoreBody{{reader: strings.NewReader(`{"message":"retry"}`)}, {reader: strings.NewReader(`{"name":"accepted"}`)}}
			client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls > 2 || req.Method != http.MethodPatch || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed" || req.Header.Get("Content-Type") != "application/openstack-images-v2.1-json-patch" || req.Header.Get("Accept") != "" {
					t.Fatal(req.Method, req.URL, req.Header, calls)
				}
				imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"add","path":"/name","value":"requested"}]`)
				code := 503
				if calls == 2 {
					code = 203
					if spoof || req.Header.Get("X-Auth-Token") != "live retry" || req.Header.Get("X-Retry") != "allowed" {
						t.Fatal("unsafe replay or lost auth/ordinary headers", req.Header, spoof)
					}
				}
				return taskCoreHTTP(req, code, bodies[calls-1]), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, original error, count uint) error {
				callbacks++
				if count > 1 {
					return original
				}
				client.SetToken("live retry")
				options.MoreHeaders = map[string]string{"content-type": "application/openstack-images-v2.1-json-patch", "aCcEpT": "", "X-Retry": "allowed"}
				if spoof {
					options.MoreHeaders["content-type"], options.MoreHeaders["aCcEpT"] = "application/json", "application/json"
				}
				return nil
			}
			originalHook := reflect.ValueOf(client.RetryFunc).Pointer()
			got, err := New(client).UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{ID: "fixed", Attributes: map[string]any{"name": "requested"}})
			if spoof {
				var native gophercloud.ErrUnexpectedResponseCode
				if got != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"message":"retry"}` || native.ResponseHeader.Get("X-Task-Proof") != "actual" || calls != 1 || bodies[1].closes != 0 {
					t.Fatal("owned header mutation was replayed or lost native proof", got, err, calls, native, bodies[1].closes)
				}
			} else if got == nil || err != nil || got.StatusCode != 203 || string(got.Resource.Body["name"]) != `"accepted"` || calls != 2 || bodies[1].closes != 1 {
				t.Fatal(got, err, calls, bodies[1].closes)
			}
			if callbacks != 1 || bodies[0].closes != 1 || reflect.ValueOf(client.RetryFunc).Pointer() != originalHook {
				t.Fatal(callbacks, bodies[0].closes)
			}
		})
	}
}

func TestImageRecordUpdatePreflightIncludesSourceControlsAndFinalDescriptors(t *testing.T) {
	cases := []struct {
		name    string
		input   ImageRecordUpdateRequest
		options []ImageRecordOption
	}{
		{"missing identity", ImageRecordUpdateRequest{}, nil},
		{"only name", ImageRecordUpdateRequest{Attributes: map[string]any{"name": "name"}}, nil},
		{"both identity forms", ImageRecordUpdateRequest{ID: "fixed", Record: &ImageRecord{}}, nil},
		{"nil resource", ImageRecordUpdateRequest{Record: &ImageRecord{}}, nil},
		{"handcrafted record has no original state", ImageRecordUpdateRequest{Record: &ImageRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"fixed"`)}}}}}, nil},
		{"literal attr id collision", ImageRecordUpdateRequest{ID: "fixed", Attributes: map[string]any{"id": "fixed"}}, nil},
		{"literal option id collision", ImageRecordUpdateRequest{ID: "fixed"}, []ImageRecordOption{WithImageRecordAttribute("id", nil)}},
		{"nil option", ImageRecordUpdateRequest{ID: "fixed"}, []ImageRecordOption{nil}},
		{"owned token header", ImageRecordUpdateRequest{ID: "fixed"}, []ImageRecordOption{WithImageRecordHeader("X-Auth-Token", "foreign")}},
		{"owned patch type header", ImageRecordUpdateRequest{ID: "fixed"}, []ImageRecordOption{WithImageRecordHeader("Content-Type", "other")}},
		{"unsupported attr", ImageRecordUpdateRequest{ID: "fixed", Attributes: map[string]any{"name": make(chan int)}}, nil},
		{"invalid descriptor", ImageRecordUpdateRequest{ID: "fixed", Attributes: map[string]any{"hw_vif_multiqueue_enabled": " false "}}, nil},
	}
	for _, control := range []string{"self", "image", "base_path", "microversion", "connection", "_synchronized"} {
		cases = append(cases, struct {
			name    string
			input   ImageRecordUpdateRequest
			options []ImageRecordOption
		}{"control " + control, ImageRecordUpdateRequest{ID: "fixed", Attributes: map[string]any{control: "value"}}, nil})
	}
	for _, id := range []string{" ", ".", "..", "bad\n", string([]byte{0xff})} {
		cases = append(cases, struct {
			name    string
			input   ImageRecordUpdateRequest
			options []ImageRecordOption
		}{fmt.Sprintf("id %q", id), ImageRecordUpdateRequest{ID: id}, nil})
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, 200, `{}`), nil }))
			got, err := service.UpdateImageRecord(context.Background(), test.input, test.options...)
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(got, err, calls)
			}
		})
	}
	for _, mode := range []string{"nil context", "canceled context", "nil service"} {
		t.Run(mode, func(t *testing.T) {
			calls, callbacks := 0, 0
			ctx := context.Background()
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil }))
			if mode == "nil context" {
				ctx = nil
			}
			if mode == "canceled context" {
				canceled, cancel := context.WithCancel(context.Background())
				cancel()
				ctx = canceled
			}
			if mode == "nil service" {
				service = nil
			}
			got, err := service.UpdateImageRecord(ctx, ImageRecordUpdateRequest{ID: "fixed"}, func(*ImageRecordOpts) error { callbacks++; return nil })
			if got != nil || err == nil || calls != 0 || callbacks != 0 {
				t.Fatal(got, err, calls, callbacks)
			}
		})
	}
}

func TestImageRecordUpdateAccepts200Through399WithoutAutomaticReplay(t *testing.T) {
	for _, code := range []int{200, 201, 204, 299, 300, 304, 399} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls, retries := 0, 0
			const raw = `{"name":"accepted"}`
			body := &taskCoreBody{reader: strings.NewReader(raw)}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, code, body), nil })
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			got, err := New(client).UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{ID: "fixed", Attributes: map[string]any{"name": "requested"}})
			if got == nil || err != nil || got.StatusCode != code || string(got.Envelope) != raw || string(got.Resource.Body["name"]) != `"accepted"` || calls != 1 || retries != 0 || body.closes != 1 {
				t.Fatal(got, err, calls, retries, body.closes)
			}
		})
	}
}

func TestImageRecordUpdateCapturesOptionsAndTopLevelAttributesBeforeMarshal(t *testing.T) {
	calls, marshals := 0, 0
	options := []ImageRecordOption{WithImageRecordHeader("X-Option", "captured")}
	attributes := map[string]any{"zzz_later": "captured scalar"}
	attributes["a_callback"] = imageRecordMarshalCallback(func() ([]byte, error) {
		marshals++
		options[0] = WithImageRecordHeader("X-Option", "changed by marshaler")
		attributes["zzz_later"] = "changed by marshaler"
		return []byte(`{"owned":true}`), nil
	})
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Option") != "captured" || req.Method != http.MethodPatch {
			t.Fatal(req.Header, req.Method)
		}
		imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"add","path":"/a_callback","value":{"owned":true}},{"op":"add","path":"/zzz_later","value":"captured scalar"}]`)
		return taskCoreJSON(req, 200, "opaque response"), nil
	})
	got, err := New(client).UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{ID: "fixed", Attributes: attributes}, options...)
	if got == nil || err != nil || calls != 1 || marshals != 1 || got.Wire != nil {
		t.Fatal(got, err, calls, marshals)
	}
	th.CheckDeepEquals(t, map[string]json.RawMessage{"a_callback": json.RawMessage(`{"owned":true}`), "zzz_later": json.RawMessage(`"captured scalar"`)}, imageRecordProperties(t, got))
	if attributes["zzz_later"] != "changed by marshaler" {
		t.Fatal("mutation callback did not run", attributes)
	}
}

func TestImageRecordUpdateAcceptedMalformedObjectAndProjectionRetainEvidence(t *testing.T) {
	for _, raw := range []string{`null`, `[]`, `false`, `42`, `"text"`, `{"hw_vif_multiqueue_enabled":" false "}`, "{\"x\":\"\xff\"}"} {
		t.Run(fmt.Sprintf("body=%q", raw), func(t *testing.T) {
			calls, retries := 0, 0
			body := &taskCoreBody{reader: strings.NewReader(raw)}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, 201, body), nil })
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			got, err := New(client).UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{ID: "fixed", Attributes: map[string]any{"name": "requested"}})
			if got == nil || err == nil || !errors.Is(err, resource.ErrInvalidOption) || got.StatusCode != 201 || string(got.Envelope) != raw || got.Header.Get("X-Task-Proof") != "actual" || calls != 1 || retries != 0 || body.closes != 1 {
				t.Fatal(got, err, calls, retries, body.closes)
			}
			taskCoreProof(t, err, 201, raw)
		})
	}
}

func TestImageRecordUpdateReadCloseStickyGuardAndRejectionProof(t *testing.T) {
	const raw = `{"name":"accepted"}`
	for _, mode := range []string{"read", "close", "cancel", "source drift", "read drift restored on Close", "outer drift restored on Close"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New("owned update body failure")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			calls, retries := 0, 0
			var client *gophercloud.ServiceClient
			body := &taskCoreBody{reader: strings.NewReader(raw)}
			outerInvalid := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerInvalid {
					return cause
				}
				return nil
			})
			action := func() {}
			switch mode {
			case "read":
				body.reader = &taskCoreReader{body: raw, err: cause}
			case "close":
				body.closeErr = errors.Join(cause, gophercloud.ErrUnexpectedResponseCode{Actual: 404})
			case "cancel":
				action = func() { cancel(cause) }
			case "source drift", "read drift restored on Close":
				action = func() { client.Endpoint = "https://foreign.test/" }
			case "outer drift restored on Close":
				action = func() { outerInvalid = true }
			}
			if mode != "read" {
				body.reader = &taskCoreReader{body: raw, err: io.EOF, action: action}
			}
			selected := io.ReadCloser(body)
			if strings.Contains(mode, "restored") {
				selected = &imageRecordCloseBody{taskCoreBody: body, after: func() { client.Endpoint = "https://glance.example/reverse/glance/v2/"; outerInvalid = false }}
			}
			client = taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, 201, selected), nil })
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			got, err := New(client).UpdateImageRecord(ctx, ImageRecordUpdateRequest{ID: "fixed", Attributes: map[string]any{"name": "requested"}})
			if got == nil || err == nil || calls != 1 || retries != 0 || body.closes != 1 {
				t.Fatal(got, err, calls, retries, body.closes)
			}
			if strings.Contains(mode, "source") || mode == "read drift restored on Close" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, cause) {
				t.Fatal("cause lost", err)
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if errors.Is(err, resource.ErrNotFound) {
				t.Fatal("accepted close failure treated as missing", err)
			}
			taskCoreProof(t, err, 201, raw)
		})
	}
	for _, code := range []int{400, 403, 404, 409, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls := 0
			const reply = `{"message":"rejected"}`
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreJSON(req, code, reply), nil })
			got, err := New(client).UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{ID: "fixed", Attributes: map[string]any{"name": "requested"}})
			if got != nil || err == nil || calls != 1 {
				t.Fatal(got, err, calls)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			var accepted *resource.ResponseError
			if !errors.As(err, &native) || native.Actual != code || string(native.Body) != reply || native.ResponseHeader.Get("X-Task-Proof") != "actual" || errors.As(err, &accepted) {
				t.Fatal("native rejection type or evidence lost", err)
			}
		})
	}
}

// Equal maps with distinct byte slices establish independence of the retained
// pending body, original snapshot and public projected view across retries.
func TestImageRecordUpdatePendingRecordsDoNotAliasInputsOrReceipts(t *testing.T) {
	var handler taskCoreTransport
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { return handler(req) }))
	seed := imageRecordUpdateFetched(t, service, `{"id":"fixed","name":"before","properties":{"nested":{"exact":900719925474099312345}}}`, &handler)
	before := cloneImageRecord(seed)
	attrs := map[string]any{"name": "after", "properties": json.RawMessage(`{"nested":{"exact":900719925474099312346}}`)}
	handler = func(req *http.Request) (*http.Response, error) { return taskCoreJSON(req, 201, `opaque accepted`), nil }
	got, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: seed, Attributes: attrs})
	if got == nil || err != nil || got == seed || got.Resource == seed.Resource || got.bodyState == seed.bodyState || !reflect.DeepEqual(before, seed) {
		t.Fatal(got, err, seed)
	}
	got.Resource.Body["name"][1] = 'X'
	got.bodyState.current["name"][1] = 'X'
	got.bodyState.original["name"][1] = 'X'
	got.Envelope[0] = '!'
	got.Header.Set("X-Task-Proof", "changed")
	if !reflect.DeepEqual(before, seed) || !bytes.Equal(attrs["properties"].(json.RawMessage), []byte(`{"nested":{"exact":900719925474099312346}}`)) {
		t.Fatal("pending state aliases seed or attributes", seed, attrs)
	}
}
