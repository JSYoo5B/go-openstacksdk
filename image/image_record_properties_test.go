package image

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
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestImageRecordPropertiesBooleanUsesCanonicalComparisonAndCachedSeed(t *testing.T) {
	for _, test := range []struct {
		name, seed string
		options    []ImageRecordPropertiesOption
		updated    bool
	}{
		{"empty cached map", `{"id":"fixed"}`, nil, false},
		{"missing custom null is unchanged", `{"id":"fixed"}`, []ImageRecordPropertiesOption{WithImageRecordProperty("foo", nil)}, false},
		{"canonical size compares converted view", `{"id":"fixed","size":"04"}`, []ImageRecordPropertiesOption{WithImageRecordProperty("size", 4)}, false},
		{"canonical numeric bool equivalence", `{"id":"fixed","size":true}`, []ImageRecordPropertiesOption{WithImageRecordProperty("size", 1)}, false},
		{"canonical protected compares truthy descriptor", `{"id":"fixed","protected":"false"}`, []ImageRecordPropertiesOption{WithImageRecordProperty("is_protected", true)}, false},
		{"cached custom map means true without HTTP", `{"id":"fixed","vendor":"keep"}`, nil, true},
		{"custom null retains cached nonnull value", `{"id":"fixed","foo":7}`, []ImageRecordPropertiesOption{WithImageRecordProperty("foo", nil)}, true},
		{"wire alias compares missing but delegated value is unchanged", `{"id":"fixed","protected":false}`, []ImageRecordPropertiesOption{WithImageRecordProperty("protected", false)}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var handler taskCoreTransport
			calls, locations := 0, 0
			cloud := "fetch location"
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return resource.CloudLocation{Cloud: &cloud}, nil }})
			seed := imageRecordUpdateFetched(t, service, test.seed, &handler)
			seed.Resource.Body["id"] = json.RawMessage(`"public target decoy"`)
			seed.Resource.Body["properties"] = json.RawMessage(`{"public":"must not seed"}`)
			before := cloneImageRecord(seed)
			cloud = "helper location"
			handler = func(req *http.Request) (*http.Response, error) {
				t.Fatal("Source helper boolean must not imply PATCH", req.Method, req.URL)
				return nil, nil
			}
			got, err := service.UpdateImagePropertiesRecord(context.Background(), ImageRecordPropertiesRequest{Record: seed}, test.options...)
			if got == nil || got.Record == nil || err != nil || got.Updated != test.updated || calls != 1 || locations != 2 || got.Record == seed || got.Record.Resource == seed.Resource || len(got.Record.Resource.Body) != 65 || len(got.Record.ImportMethods) != 0 || !reflect.DeepEqual(before, seed) {
				t.Fatal(got, err, calls, locations)
			}
			if string(got.Record.Resource.Body["id"]) != `"fixed"` || got.Record.StatusCode != 203 || string(got.Record.Envelope) != test.seed || !reflect.DeepEqual(got.Record.Wire, seed.Wire) || !reflect.DeepEqual(got.Record.bodyState.current, seed.bodyState.current) {
				t.Fatal("helper changed raw seed or fetch receipt", got.Record, seed)
			}
			var location resource.CloudLocation
			if err := json.Unmarshal(got.Record.Resource.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != "helper location" {
				t.Fatal(location, err)
			}
			if test.name == "custom null retains cached nonnull value" {
				th.AssertEquals(t, "7", string(imageRecordProperties(t, got.Record)["foo"]))
			}
			got.Record.Resource.Body["id"][1] = 'X'
			got.Record.Wire.Body["id"][1] = 'X'
			got.Record.Header.Set("X-Task-Proof", "changed")
			if !reflect.DeepEqual(before, seed) {
				t.Fatal("helper result aliases seed", seed)
			}
		})
	}
}

func TestImageRecordPropertiesCachedCopyShapeBoundaries(t *testing.T) {
	for _, test := range []struct {
		name, raw        string
		options          []ImageRecordPropertiesOption
		invalid, missing bool
	}{
		{"missing raw cached properties", "", nil, true, true},
		{"null cached properties", `null`, nil, true, false},
		{"numeric cached properties", `1`, nil, true, false},
		{"string cached properties", `"text"`, nil, true, false},
		{"empty list copy is false", `[]`, nil, false, false},
		{"empty list unchanged null is false", `[]`, []ImageRecordPropertiesOption{WithImageRecordProperty("foo", nil)}, false, false},
		{"empty list string-key insertion fails", `[]`, []ImageRecordPropertiesOption{WithImageRecordProperty("foo", "new")}, true, false},
		{"nonempty list fails full keyword expansion", `[1]`, nil, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var handler taskCoreTransport
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			fetched := `{"id":"fixed"}`
			if test.missing {
				fetched = "opaque GET response"
			}
			seed := imageRecordUpdateFetched(t, service, fetched, &handler)
			wantCalls := 1
			if !test.missing {
				handler = func(req *http.Request) (*http.Response, error) {
					if req.Method != http.MethodPatch {
						t.Fatal("raw property shape setup fetched again", req.Method)
					}
					return taskCoreJSON(req, 299, "pending raw property response"), nil
				}
				var err error
				seed, err = service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: seed, Attributes: map[string]any{"properties": json.RawMessage(test.raw)}})
				if seed == nil || err != nil || string(seed.Resource.Body["properties"]) != test.raw {
					t.Fatal(seed, err)
				}
				wantCalls++
			}
			before := cloneImageRecord(seed)
			handler = func(req *http.Request) (*http.Response, error) {
				t.Fatal("cached-copy shape selected HTTP", req.Method, req.URL)
				return nil, nil
			}
			got, err := service.UpdateImagePropertiesRecord(context.Background(), ImageRecordPropertiesRequest{Record: seed}, test.options...)
			if calls != wantCalls || !reflect.DeepEqual(before, seed) {
				t.Fatal(calls, seed)
			}
			if test.invalid {
				if got != nil || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(got, err)
				}
			} else if got == nil || got.Record == nil || got.Updated || err != nil || !reflect.DeepEqual(got.Record.bodyState, seed.bodyState) {
				t.Fatal(got, err)
			}
		})
	}
}

func TestImageRecordPropertiesResolveOrderedAliasesAfterCompleteLists(t *testing.T) {
	for _, mode := range []string{"ordered JSON aliases", "upsert retains first key position"} {
		t.Run(mode, func(t *testing.T) {
			var handler taskCoreTransport
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordUpdateFetched(t, service, `{"id":"fixed"}`, &handler)
			options := []ImageRecordPropertiesOption{WithImageRecordPropertiesJSON(json.RawMessage(`{"kernel":"kernel*","kernel_id":"direct","ramdisk_id":"old direct","ramdisk":"ram*"}`))}
			wantCalls := 6
			if mode == "upsert retains first key position" {
				options = []ImageRecordPropertiesOption{WithImageRecordProperty("kernel_id", "direct first"), WithImageRecordProperty("kernel", "kernel*"), WithImageRecordProperty("kernel_id", "direct last")}
				wantCalls = 4
			}
			handler = func(req *http.Request) (*http.Response, error) {
				if calls == wantCalls {
					if req.Method != http.MethodPatch || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed" || req.URL.RawQuery != "" || req.Header.Get("Content-Type") != "application/openstack-images-v2.1-json-patch" || req.Header.Get("Accept") != "" {
						t.Fatal(req.Method, req.URL, req.Header)
					}
					want := `[{"op":"add","path":"/kernel_id","value":"direct"},{"op":"add","path":"/ramdisk_id","value":"ram first"}]`
					if mode == "upsert retains first key position" {
						want = `[{"op":"add","path":"/kernel_id","value":"kernel first"}]`
					}
					imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), want)
					return taskCoreJSON(req, 203, `{}`), nil
				}
				if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images" || req.Body != nil || req.URL.Query().Has("name") || req.URL.Query().Has("os_hidden") || req.URL.Query().Has("member_status") {
					t.Fatal("resolver must use query-free complete listing, without Find", req.Method, req.URL)
				}
				if calls == 2 || calls == 4 {
					th.AssertEquals(t, "", req.URL.RawQuery)
					if calls == 2 {
						return taskCoreJSON(req, 200, `{"images":[{"id":"deleted","name":"kernel-early","status":"DeLeTeD"},{"id":"kernel first","name":"kernel-early","status":"ACTIVE"}],"next":"?marker=tail"}`), nil
					}
					return taskCoreJSON(req, 200, `{"images":[{"id":"ram first","name":"ram-early","status":"active"}],"next":"?marker=tail"}`), nil
				}
				th.AssertEquals(t, "marker=tail", req.URL.RawQuery)
				if calls == 3 {
					return taskCoreJSON(req, 200, `{"images":[{"id":"kernel late","name":"kernel*","status":"active"},{"id":"duplicate","name":"kernel-early","status":"queued"}]}`), nil
				}
				if calls == 5 {
					return taskCoreJSON(req, 200, `{"images":[{"id":"ram late","name":"ram*","status":"active"}]}`), nil
				}
				t.Fatal("unexpected resolver request", calls, req.URL)
				return nil, nil
			}
			got, err := service.UpdateImagePropertiesRecord(context.Background(), ImageRecordPropertiesRequest{Record: seed}, options...)
			if got == nil || got.Record == nil || !got.Updated || err != nil || calls != wantCalls {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordPropertiesFalsyAliasesAndRawMetaSkipResolution(t *testing.T) {
	for _, test := range []struct {
		name, key, raw, value string
		meta, updated         bool
	}{
		{"null kernel", "kernel", `null`, `null`, false, false},
		{"false kernel", "kernel", `false`, `"False"`, false, true},
		{"zero kernel", "kernel", `0`, `"0"`, false, true},
		{"empty string kernel", "kernel", `""`, `""`, false, true},
		{"empty array kernel", "kernel", `[]`, `"[]"`, false, true},
		{"empty object kernel", "kernel", `{}`, `"{}"`, false, true},
		{"false ramdisk", "ramdisk", `false`, `"False"`, false, true},
		{"meta kernel is raw and unresolved", "kernel", `{"literal":true}`, `{"literal":true}`, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var handler taskCoreTransport
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordUpdateFetched(t, service, `{"id":"fixed"}`, &handler)
			handler = func(req *http.Request) (*http.Response, error) {
				if !test.updated || req.Method != http.MethodPatch || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed" {
					t.Fatal("falsy or meta alias invoked image discovery", req.Method, req.URL)
				}
				imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"add","path":"/`+test.key+`","value":`+test.value+`}]`)
				return taskCoreJSON(req, 203, `{}`), nil
			}
			option := WithImageRecordProperty(test.key, json.RawMessage(test.raw))
			if test.meta {
				option = WithImageRecordPropertyMeta(test.key, json.RawMessage(test.raw))
			}
			got, err := service.UpdateImagePropertiesRecord(context.Background(), ImageRecordPropertiesRequest{Record: seed}, option)
			wantCalls := 1
			if test.updated {
				wantCalls = 2
			}
			if got == nil || got.Record == nil || got.Updated != test.updated || err != nil || calls != wantCalls {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordPropertiesResolverMissingAndLateFailuresAreTerminal(t *testing.T) {
	for _, mode := range []string{"missing selection", "later native", "later null status", "later missing status", "later numeric status", "later descriptor", "later malformed JSON", "later read", "later Close", "later source", "later cancel"} {
		t.Run(mode, func(t *testing.T) {
			marker := errors.New("resolver handling cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var handler taskCoreTransport
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordUpdateFetched(t, service, `{"id":"fixed"}`, &handler)
			last := `{"images":[]}`
			var body *taskCoreBody
			handler = func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images" {
					t.Fatal("early match bypassed complete resolver", req.Method, req.URL)
				}
				if calls == 2 {
					th.AssertEquals(t, "", req.URL.RawQuery)
					if mode == "missing selection" {
						return taskCoreJSON(req, 203, `{"images":[{"id":"other","name":"other","status":"active"}]}`), nil
					}
					return taskCoreJSON(req, 203, `{"images":[{"id":"early","name":"kernel-match","status":"active"}],"next":"?marker=later"}`), nil
				}
				if calls != 3 || req.URL.Query().Get("marker") != "later" {
					t.Fatal(calls, req.URL)
				}
				switch mode {
				case "later native":
					return taskCoreJSON(req, 500, "later native rejection"), nil
				case "later null status":
					last = `{"images":[{"id":"last","status":null}]}`
				case "later missing status":
					last = `{"images":[{"id":"last"}]}`
				case "later numeric status":
					last = `{"images":[{"id":"last","status":3}]}`
				case "later descriptor":
					last = `{"images":[{"id":"last","status":"active","hw_vif_multiqueue_enabled":" invalid BoolStr "}]}`
				case "later malformed JSON":
					last = "malformed page"
				}
				body = &taskCoreBody{reader: strings.NewReader(last)}
				switch mode {
				case "later read":
					body.reader = &taskCoreReader{body: last, err: marker}
				case "later Close":
					body.closeErr = marker
				case "later source":
					body.reader = &taskCoreReader{body: last, err: io.EOF, action: func() { client.Endpoint = "https://foreign.test/" }}
				case "later cancel":
					body.reader = &taskCoreReader{body: last, err: io.EOF, action: func() { cancel(marker) }}
				}
				return taskCoreHTTP(req, 203, body), nil
			}
			got, err := service.UpdateImagePropertiesRecord(ctx, ImageRecordPropertiesRequest{Record: seed}, WithImageRecordProperty("kernel", "kernel*"))
			if mode == "missing selection" {
				if got == nil || got.Record == nil || got.Updated || err != nil || calls != 2 {
					t.Fatal(got, err, calls)
				}
				return
			}
			if got != nil || err == nil || calls != 3 {
				t.Fatal(got, err, calls)
			}
			if mode == "later native" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 500 || string(native.Body) != "later native rejection" {
					t.Fatal(native, err)
				}
				return
			}
			if body == nil || body.closes != 1 {
				t.Fatal(body)
			}
			taskCoreProof(t, err, 203, last)
			if (mode == "later read" || mode == "later Close" || mode == "later cancel") && !errors.Is(err, marker) {
				t.Fatal("resolver cause lost", err)
			}
			if mode == "later cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordPropertiesRespectInitialProjectionResolverCopyConversionOrder(t *testing.T) {
	for _, mode := range []string{"literal lookup before missing copy", "lookup rejection before missing copy", "initial descriptor before lookup", "conversion before meta override"} {
		t.Run(mode, func(t *testing.T) {
			var handler taskCoreTransport
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			input := ImageRecordPropertiesRequest{ID: "fixed"}
			options := []ImageRecordPropertiesOption{WithImageRecordProperty("kernel", "match")}
			seedCalls := 0
			if mode == "initial descriptor before lookup" || mode == "conversion before meta override" {
				seed := imageRecordUpdateFetched(t, service, `{"id":"fixed"}`, &handler)
				input = ImageRecordPropertiesRequest{Record: seed}
				seedCalls = 1
				if mode == "initial descriptor before lookup" {
					seed.bodyState.current["is_hw_vif_multiqueue_enabled"] = json.RawMessage(`" invalid BoolStr "`)
				}
				if mode == "conversion before meta override" {
					options = append(options, WithImageRecordProperty("size", "not an integer"), WithImageRecordPropertyMeta("size", 2))
				}
			}
			handler = func(req *http.Request) (*http.Response, error) {
				if mode == "initial descriptor before lookup" || req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images" || req.URL.RawQuery != "" {
					t.Fatal("helper phase order or implicit detail GET changed", req.Method, req.URL)
				}
				if mode == "lookup rejection before missing copy" {
					return taskCoreJSON(req, 500, "lookup failed first"), nil
				}
				return taskCoreJSON(req, 203, `{"images":[{"id":"found","name":"match","status":"active"}]}`), nil
			}
			got, err := service.UpdateImagePropertiesRecord(context.Background(), input, options...)
			wantCalls := seedCalls + 1
			if mode == "initial descriptor before lookup" {
				wantCalls = seedCalls
			}
			if got != nil || err == nil || calls != wantCalls {
				t.Fatal(got, err, calls)
			}
			if mode == "lookup rejection before missing copy" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 500 || errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(native, err)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordPropertiesIntegerConversionIsHelperInt(t *testing.T) {
	for _, test := range []struct {
		name, key, raw, converted string
		invalid                   bool
	}{
		{"min disk bool true", "min_disk", `true`, `1`, false},
		{"min ram bool false", "min_ram", `false`, `0`, false},
		{"size signed spaces underscores", "size", `" +1_024 "`, `1024`, false},
		{"virtual size Unicode decimal", "virtual_size", `"-١_٢"`, `-12`, false},
		{"finite float truncates toward zero", "size", `-2.9`, `-2`, false},
		{"arbitrary precision integer token", "virtual_size", `900719925474099312345`, `900719925474099312345`, false},
		{"null fails rather than descriptor zero", "min_disk", `null`, "", true},
		{"array fails", "min_ram", `[]`, "", true},
		{"object fails", "size", `{}`, "", true},
		{"fraction string fails", "virtual_size", `"1.2"`, "", true},
		{"bad underscore fails", "min_disk", `"1__2"`, "", true},
		{"nonfinite float overflow fails", "size", `1e400`, "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var handler taskCoreTransport
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordUpdateFetched(t, service, `{"id":"fixed"}`, &handler)
			handler = func(req *http.Request) (*http.Response, error) {
				if test.invalid || req.Method != http.MethodPatch {
					t.Fatal("invalid helper int reached PATCH", req.Method, req.URL)
				}
				imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"add","path":"/`+test.key+`","value":`+test.converted+`}]`)
				return taskCoreJSON(req, 203, `{}`), nil
			}
			got, err := service.UpdateImagePropertiesRecord(context.Background(), ImageRecordPropertiesRequest{Record: seed}, WithImageRecordProperty(test.key, json.RawMessage(test.raw)))
			if test.invalid {
				if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
					t.Fatal(got, err, calls)
				}
			} else if got == nil || got.Record == nil || !got.Updated || err != nil || calls != 2 || string(got.Record.Resource.Body[test.key]) != test.converted {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestImageRecordPropertiesRawValuesPythonStringsAndMetaLast(t *testing.T) {
	var handler taskCoreTransport
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
	service := New(client)
	seed := imageRecordUpdateFetched(t, service, `{"id":"fixed"}`, &handler)
	kwargs := json.RawMessage(`{"is_protected":false,"protected":false,"tags":[true,1,null],"custom":true,"nested":{"b":[1,"x",false],"a":null},"name":3.0,"nullcustom":null,"scientific":1e-07}`)
	meta := json.RawMessage(`{"custom":false,"metaobject":{"exact":900719925474099312345},"size":"bad descriptor int is allowed as raw meta"}`)
	handler = func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPatch {
			t.Fatal(req.Method, req.URL)
		}
		imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"add","path":"/custom","value":false},{"op":"add","path":"/metaobject","value":{"exact":900719925474099312345}},{"op":"add","path":"/name","value":"3.0"},{"op":"add","path":"/nested","value":"{'b': [1, 'x', False], 'a': None}"},{"op":"add","path":"/protected","value":false},{"op":"add","path":"/scientific","value":"1e-07"},{"op":"add","path":"/size","value":"bad descriptor int is allowed as raw meta"},{"op":"add","path":"/tags","value":[true,1,null]}]`)
		return taskCoreJSON(req, 203, "opaque accepted conversion response"), nil
	}
	got, err := service.UpdateImagePropertiesRecord(context.Background(), ImageRecordPropertiesRequest{Record: seed}, WithImageRecordPropertiesJSON(kwargs), WithImageRecordPropertiesMetaJSON(meta))
	if got == nil || got.Record == nil || !got.Updated || err != nil || calls != 2 || string(got.Record.Resource.Body["size"]) != "0" || string(got.Record.Resource.Body["is_protected"]) != "false" {
		t.Fatal(got, err, calls)
	}
	properties := imageRecordProperties(t, got.Record)
	th.AssertEquals(t, `false`, string(properties["custom"]))
	if _, present := properties["nullcustom"]; present {
		t.Fatal("canonical missing None inserted unknown null", properties)
	}
}

func TestImageRecordPropertiesRetainCachedKeysAndDelegateDeclaredPromotion(t *testing.T) {
	for _, test := range []struct {
		name, seed, patch, reply string
		options                  []ImageRecordPropertiesOption
		field, value             string
	}{
		{"canonical cached name becomes Body with empty wire diff", `{"id":"fixed","properties":{"name":"cached name","keep":7}}`, `[]`, `{"properties":{"keep":7}}`, nil, "name", `"cached name"`},
		{"wire cached protected becomes declared Body", `{"id":"fixed","properties":{"protected":false,"keep":7}}`, `[]`, `{"properties":{"keep":7}}`, nil, "is_protected", `false`},
		{"explicit properties and retained extras use packing precedence", `{"id":"fixed","keep":7}`, `[{"op":"add","path":"/added","value":"raw"},{"op":"add","path":"/replaced","value":1}]`, `{"properties":{"keep":7,"added":"raw","replaced":1}}`, []ImageRecordPropertiesOption{WithImageRecordPropertiesMetaJSON(json.RawMessage(`{"properties":{"replaced":1},"added":"raw"}`))}, "id", `"fixed"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var handler taskCoreTransport
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordUpdateFetched(t, service, test.seed, &handler)
			before := cloneImageRecord(seed)
			handler = func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPatch || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed" {
					t.Fatal(req.Method, req.URL)
				}
				imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), test.patch)
				return taskCoreJSON(req, 203, test.reply), nil
			}
			got, err := service.UpdateImagePropertiesRecord(context.Background(), ImageRecordPropertiesRequest{Record: seed}, test.options...)
			if got == nil || got.Record == nil || !got.Updated || err != nil || calls != 2 || string(got.Record.Resource.Body[test.field]) != test.value || !reflect.DeepEqual(before, seed) || len(got.Record.bodyState.dirty) != 0 {
				t.Fatal(got, err, calls, seed)
			}
			th.AssertEquals(t, "7", string(imageRecordProperties(t, got.Record)["keep"]))
		})
	}
}

func TestImageRecordPropertiesBooleanControlsPendingCommitAndRetargeting(t *testing.T) {
	for _, cached := range []bool{false, true} {
		t.Run(fmt.Sprintf("pending cached=%v", cached), func(t *testing.T) {
			var handler taskCoreTransport
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			fetched := `{"id":"fixed","name":"before"}`
			if cached {
				fetched = `{"id":"fixed","name":"before","vendor":"keep"}`
			}
			seed := imageRecordUpdateFetched(t, service, fetched, &handler)
			handler = func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPatch {
					t.Fatal("pending body performed discovery", req.Method, req.URL)
				}
				imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"replace","path":"/name","value":"after"}]`)
				if calls == 2 {
					return taskCoreJSON(req, 299, "pending name response"), nil
				}
				if calls != 3 {
					t.Fatal("implicit replay", calls)
				}
				return taskCoreJSON(req, 203, `{}`), nil
			}
			pending, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: seed, Attributes: map[string]any{"name": "after"}})
			if pending == nil || err != nil {
				t.Fatal(pending, err)
			}
			got, err := service.UpdateImagePropertiesRecord(context.Background(), ImageRecordPropertiesRequest{Record: pending})
			wantCalls := 2
			if cached {
				wantCalls = 3
			}
			if got == nil || got.Record == nil || got.Updated != cached || err != nil || calls != wantCalls || string(got.Record.Resource.Body["name"]) != `"after"` {
				t.Fatal(got, err, calls)
			}
			committed, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: got.Record})
			if committed == nil || err != nil || calls != 3 {
				t.Fatal("False must preserve pending state and True must commit it", committed, err, calls)
			}
		})
	}
	t.Run("sticky dirty state can submit empty PATCH", func(t *testing.T) {
		var handler taskCoreTransport
		calls := 0
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
		service := New(client)
		seed := imageRecordUpdateFetched(t, service, `{"id":"fixed","name":"before","vendor":"keep"}`, &handler)
		handler = func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPatch {
				t.Fatal(req.Method)
			}
			want := `[]`
			if calls == 2 {
				want = `[{"op":"replace","path":"/name","value":"after"}]`
			}
			imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), want)
			if calls < 4 {
				return taskCoreJSON(req, 299, "pending dirty body"), nil
			}
			if calls != 4 {
				t.Fatal("clean baseline replayed", calls)
			}
			return taskCoreJSON(req, 203, `{}`), nil
		}
		pending, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: seed, Attributes: map[string]any{"name": "after"}})
		if pending == nil || err != nil {
			t.Fatal(pending, err)
		}
		pending, err = service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: pending, Attributes: map[string]any{"name": "before"}})
		if pending == nil || err != nil {
			t.Fatal(pending, err)
		}
		got, err := service.UpdateImagePropertiesRecord(context.Background(), ImageRecordPropertiesRequest{Record: pending})
		if got == nil || got.Record == nil || !got.Updated || err != nil || calls != 4 || len(got.Record.bodyState.dirty) != 0 {
			t.Fatal(got, err, calls)
		}
		clean, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: got.Record})
		if clean == nil || err != nil || calls != 4 {
			t.Fatal(clean, err, calls)
		}
	})
	for _, immediate := range []bool{false, true} {
		t.Run(fmt.Sprintf("meta ID immediate=%v", immediate), func(t *testing.T) {
			const nextID = "new /한:%?\\target"
			var handler taskCoreTransport
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordUpdateFetched(t, service, `{"id":"fixed"}`, &handler)
			handler = func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPatch || req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape(nextID) || req.URL.RawQuery != "" {
					t.Fatal("raw meta ID did not retarget the owned update", req.Method, req.URL)
				}
				imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"replace","path":"/id","value":"new /한:%?\\target"},{"op":"add","path":"/name","value":"after"}]`)
				return taskCoreJSON(req, 203, `{}`), nil
			}
			options := []ImageRecordPropertiesOption{WithImageRecordPropertyMeta("id", nextID)}
			if immediate {
				options = append(options, WithImageRecordPropertyMeta("name", "after"))
			}
			got, err := service.UpdateImagePropertiesRecord(context.Background(), ImageRecordPropertiesRequest{Record: seed}, options...)
			wantCalls := 1
			if immediate {
				wantCalls = 2
			}
			if got == nil || got.Record == nil || !got.Updated || err != nil || calls != wantCalls || taskCoreText(t, got.Record.Resource.Body["id"]) != nextID || string(seed.Resource.Body["id"]) != `"fixed"` {
				t.Fatal(got, err, calls, seed)
			}
			if !immediate {
				committed, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: got.Record, Attributes: map[string]any{"name": "after"}})
				if committed == nil || err != nil || calls != 2 {
					t.Fatal(committed, err, calls)
				}
			}
		})
	}
}

func TestImageRecordPropertiesSharedAcceptedTranslationAndPendingResponses(t *testing.T) {
	for _, test := range []struct {
		name, body       string
		code             int
		invalid, pending bool
	}{
		{"object overlays and cleans", `{"name":"server name","server":{"exact":900719925474099312345}}`, 200, false, false},
		{"invalid syntax keeps pending", "opaque PATCH response", 299, false, true},
		{"null response", `null`, 201, true, false},
		{"array response", `[]`, 300, true, false},
		{"scalar response", `false`, 399, true, false},
		{"descriptor response error", `{"hw_vif_multiqueue_enabled":" invalid BoolStr "}`, 203, true, false},
		{"nonUTF8 response", "opaque\xff", 203, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var handler taskCoreTransport
			calls, retries := 0, 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordUpdateFetched(t, service, `{"id":"fixed"}`, &handler)
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			handler = func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPatch || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed" || req.Header.Get("Content-Type") != "application/openstack-images-v2.1-json-patch" || req.Header.Get("Accept") != "" {
					t.Fatal(req.Method, req.URL, req.Header)
				}
				imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"add","path":"/name","value":"client name"},{"op":"add","path":"/vendor","value":"True"}]`)
				if calls == 3 {
					return taskCoreJSON(req, 203, `{}`), nil
				}
				reply := taskCoreJSON(req, test.code, test.body)
				reply.Header.Set("OpenStack-image-import-methods", "one,, two")
				return reply, nil
			}
			got, err := service.UpdateImagePropertiesRecord(context.Background(), ImageRecordPropertiesRequest{Record: seed}, WithImageRecordPropertiesJSON(json.RawMessage(`{"name":"client name","vendor":true}`)))
			if got == nil || got.Record == nil || got.Record.StatusCode != test.code || string(got.Record.Envelope) != test.body || got.Record.Header.Get("X-Task-Proof") != "actual" || calls != 2 || retries != 0 {
				t.Fatal(got, err, calls, retries)
			}
			if test.invalid {
				if got.Updated || err == nil {
					t.Fatal("translation failure reported helper success", got, err)
				}
				taskCoreProof(t, err, test.code, test.body)
				return
			}
			if !got.Updated || err != nil {
				t.Fatal(got, err)
			}
			th.CheckDeepEquals(t, []string{"one", "", " two"}, got.Record.ImportMethods)
			if !test.pending {
				if got.Record.Wire == nil || string(got.Record.Resource.Body["name"]) != `"server name"` || len(got.Record.bodyState.dirty) != 0 {
					t.Fatal(got.Record)
				}
				th.AssertEquals(t, `{"exact":900719925474099312345}`, string(imageRecordProperties(t, got.Record)["server"]))
			} else if got.Record.Wire != nil || len(got.Record.bodyState.dirty) == 0 || string(got.Record.Resource.Body["name"]) != `"client name"` {
				t.Fatal(got.Record)
			}
			clean, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: got.Record})
			wantCalls := 2
			if test.pending {
				wantCalls = 3
			}
			if clean == nil || err != nil || calls != wantCalls {
				t.Fatal("response failed clean/pending lifecycle", clean, err, calls)
			}
		})
	}
}

func TestImageRecordPropertiesAcceptedHandlingErrorsReturnFalsePartialReceiptWithoutReplay(t *testing.T) {
	for _, mode := range []string{"read", "Close", "cancel", "source restored on Close", "outer restored on Close"} {
		t.Run(mode, func(t *testing.T) {
			const raw = "already accepted PATCH"
			marker := errors.New("property PATCH handling cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			outerInvalid := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerInvalid {
					return marker
				}
				return nil
			})
			var handler taskCoreTransport
			calls, retries := 0, 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordUpdateFetched(t, service, `{"id":"fixed"}`, &handler)
			before := cloneImageRecord(seed)
			body := &taskCoreBody{reader: strings.NewReader(raw)}
			beforeRead := func() {}
			switch mode {
			case "read":
				body.reader = &taskCoreReader{body: raw, err: marker}
			case "Close":
				body.closeErr = marker
			case "cancel":
				beforeRead = func() { cancel(marker) }
			case "source restored on Close":
				beforeRead = func() { client.Endpoint = "https://foreign.test/" }
			case "outer restored on Close":
				beforeRead = func() { outerInvalid = true }
			}
			if mode != "read" {
				body.reader = &taskCoreReader{body: raw, err: io.EOF, action: beforeRead}
			}
			selected := io.ReadCloser(body)
			if strings.Contains(mode, "restored") {
				selected = &imageRecordCloseBody{taskCoreBody: body, after: func() { client.Endpoint = "https://glance.example/reverse/glance/v2/"; outerInvalid = false }}
			}
			handler = func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPatch {
					t.Fatal(req.Method)
				}
				return taskCoreHTTP(req, 201, selected), nil
			}
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			got, err := service.UpdateImagePropertiesRecord(ctx, ImageRecordPropertiesRequest{Record: seed}, WithImageRecordProperty("name", "after"))
			if got == nil || got.Record == nil || got.Updated || err == nil || got.Record.StatusCode != 201 || string(got.Record.Envelope) != raw || calls != 2 || retries != 0 || body.closes != 1 || !reflect.DeepEqual(before, seed) {
				t.Fatal(got, err, calls, retries, body.closes)
			}
			if strings.HasPrefix(mode, "source") {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, marker) {
				t.Fatal("accepted handling cause lost", err)
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			taskCoreProof(t, err, 201, raw)
		})
	}
}

func TestImageRecordPropertiesNativeRejectionsAndRetryOwnership(t *testing.T) {
	for _, mode := range []string{"400", "403", "404", "599", "ordinary retry", "header ownership", "body ownership", "source retry", "hook cause", "expanded OkCodes"} {
		t.Run(mode, func(t *testing.T) {
			marker := errors.New("property retry cause")
			var handler taskCoreTransport
			calls, retries := 0, 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordUpdateFetched(t, service, `{"id":"fixed"}`, &handler)
			firstCode := 503
			switch mode {
			case "400":
				firstCode = 400
			case "403":
				firstCode = 403
			case "404":
				firstCode = 404
			case "599":
				firstCode = 599
			}
			handler = func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPatch || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed" || req.URL.RawQuery != "" || req.Header.Get("Content-Type") != "application/openstack-images-v2.1-json-patch" || req.Header.Get("Accept") != "" {
					t.Fatal(req.Method, req.URL, req.Header)
				}
				imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"add","path":"/name","value":"after"}]`)
				if calls == 2 {
					return taskCoreJSON(req, firstCode, "actual first rejection"), nil
				}
				if calls != 3 || req.Header.Get("X-Retry") != "ordinary" || req.Header.Get("X-Auth-Token") != "retry token" {
					t.Fatal(calls, req.Header)
				}
				if mode == "expanded OkCodes" {
					return taskCoreJSON(req, 418, "actual expanded rejection"), nil
				}
				return taskCoreJSON(req, 203, `{}`), nil
			}
			client.RetryFunc = func(_ context.Context, _, _ string, config *gophercloud.RequestOpts, original error, count uint) error {
				retries++
				if mode == "400" || mode == "403" || mode == "404" || mode == "599" || count > 1 {
					return original
				}
				client.SetToken("retry token")
				config.MoreHeaders = map[string]string{"X-Retry": "ordinary", "Content-Type": "application/openstack-images-v2.1-json-patch", "Accept": ""}
				switch mode {
				case "header ownership":
					config.MoreHeaders["Content-Type"] = "application/json"
				case "body ownership":
					config.JSONBody = map[string]bool{"foreign": true}
				case "source retry":
					client.Endpoint = "https://foreign.test/"
				case "hook cause":
					return errors.Join(original, marker)
				case "expanded OkCodes":
					config.OkCodes = append(config.OkCodes, 418)
				}
				return nil
			}
			got, err := service.UpdateImagePropertiesRecord(context.Background(), ImageRecordPropertiesRequest{Record: seed}, WithImageRecordProperty("name", "after"))
			wantCalls := 2
			if mode == "ordinary retry" || mode == "expanded OkCodes" {
				wantCalls = 3
			}
			if calls != wantCalls || retries != 1 {
				t.Fatal(calls, retries)
			}
			if mode == "ordinary retry" {
				if got == nil || got.Record == nil || !got.Updated || err != nil || got.Record.StatusCode != 203 {
					t.Fatal(got, err)
				}
				return
			}
			var native gophercloud.ErrUnexpectedResponseCode
			wantCode, wantBody := firstCode, "actual first rejection"
			if mode == "expanded OkCodes" {
				wantCode, wantBody = 418, "actual expanded rejection"
			}
			if got != nil || err == nil || !errors.As(err, &native) || native.Actual != wantCode || string(native.Body) != wantBody || native.ResponseHeader.Get("X-Task-Proof") != "actual" {
				t.Fatal(got, err, native)
			}
			if (mode == "header ownership" || mode == "body ownership" || mode == "source retry") && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if mode == "hook cause" && !errors.Is(err, marker) {
				t.Fatal("native hook cause lost", err)
			}
		})
	}
}

func TestImageRecordPropertiesConcreteFactoriesOwnValuesAndReplacementOrder(t *testing.T) {
	for _, mode := range []string{"single encodes once", "sorted bulk merges aliases", "JSON replaces kwargs", "MetaJSON replaces meta", "whole opts replace all"} {
		t.Run(mode, func(t *testing.T) {
			var handler taskCoreTransport
			calls, marshals := 0, 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			service := New(client)
			seed := imageRecordUpdateFetched(t, service, `{"id":"fixed"}`, &handler)
			var options []ImageRecordPropertiesOption
			patch, wantCalls := `[{"op":"add","path":"/vendor","value":"once"}]`, 2
			switch mode {
			case "single encodes once":
				options = []ImageRecordPropertiesOption{WithImageRecordProperty("vendor", imageRecordMarshalCallback(func() ([]byte, error) { marshals++; return []byte(`"once"`), nil }))}
				if marshals != 1 {
					t.Fatal("factory did not snapshot value", marshals)
				}
			case "sorted bulk merges aliases":
				values := map[string]any{"kernel_id": "direct", "kernel": "match"}
				bulk := WithImageRecordProperties(values)
				values["kernel_id"], values["kernel"] = "caller changed", "caller changed"
				options = []ImageRecordPropertiesOption{WithImageRecordProperty("retained", "keep"), bulk}
				patch, wantCalls = `[{"op":"add","path":"/kernel_id","value":"direct"},{"op":"add","path":"/retained","value":"keep"}]`, 3
			case "JSON replaces kwargs":
				raw := json.RawMessage(`{"name":"after"}`)
				factory := WithImageRecordPropertiesJSON(raw)
				raw[9] = 'X'
				options = []ImageRecordPropertiesOption{WithImageRecordProperty("discarded", "old"), factory}
				patch = `[{"op":"add","path":"/name","value":"after"}]`
			case "MetaJSON replaces meta":
				raw := json.RawMessage(`{"vendor":false}`)
				factory := WithImageRecordPropertiesMetaJSON(raw)
				raw[2] = 'X'
				options = []ImageRecordPropertiesOption{WithImageRecordPropertyMeta("discarded", true), factory}
				patch = `[{"op":"add","path":"/vendor","value":false}]`
			case "whole opts replace all":
				headers := map[string]string{"X-Retained": "new"}
				kwargs, meta := json.RawMessage(`{"name":"after"}`), json.RawMessage(`{"vendor":false}`)
				factory := WithImageRecordPropertiesOpts(ImageRecordPropertiesOpts{Headers: headers, Properties: kwargs, Meta: meta})
				headers["X-Retained"], kwargs[9], meta[2] = "caller changed", 'X', 'X'
				options = []ImageRecordPropertiesOption{WithImageRecordPropertiesHeader("X-Discarded", "old"), WithImageRecordProperty("discarded", "old"), WithImageRecordPropertyMeta("discarded meta", true), factory}
				patch = `[{"op":"add","path":"/name","value":"after"},{"op":"add","path":"/vendor","value":false}]`
			}
			handler = func(req *http.Request) (*http.Response, error) {
				if mode == "sorted bulk merges aliases" && calls == 2 {
					if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images" || req.URL.RawQuery != "" {
						t.Fatal(req.Method, req.URL)
					}
					return taskCoreJSON(req, 203, `{"images":[{"id":"resolved","name":"match","status":"active"}]}`), nil
				}
				if req.Method != http.MethodPatch || calls != wantCalls {
					t.Fatal(req.Method, req.URL, calls)
				}
				if mode == "whole opts replace all" && (req.Header.Get("X-Retained") != "new" || req.Header.Get("X-Discarded") != "") {
					t.Fatal(req.Header)
				}
				imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), patch)
				return taskCoreJSON(req, 203, "opaque factory response"), nil
			}
			got, err := service.UpdateImagePropertiesRecord(context.Background(), ImageRecordPropertiesRequest{Record: seed}, options...)
			if got == nil || got.Record == nil || !got.Updated || err != nil || calls != wantCalls || (mode == "single encodes once" && marshals != 1) {
				t.Fatal(got, err, calls, marshals)
			}
		})
	}
	for _, raw := range []string{"null", "false", "0", `""`, "[]", "{}"} {
		t.Run("falsy meta/"+raw, func(t *testing.T) {
			var handler taskCoreTransport
			calls := 0
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) }))
			seed := imageRecordUpdateFetched(t, service, `{"id":"fixed"}`, &handler)
			handler = func(req *http.Request) (*http.Response, error) {
				t.Fatal("Source falsy meta was fabricated as properties", req.URL)
				return nil, nil
			}
			got, err := service.UpdateImagePropertiesRecord(context.Background(), ImageRecordPropertiesRequest{Record: seed}, WithImageRecordPropertiesMetaJSON(json.RawMessage(raw)))
			if got == nil || got.Record == nil || got.Updated || err != nil || calls != 1 {
				t.Fatal(got, err, calls)
			}
		})
	}
	t.Run("failed factory preserves marshal cause", func(t *testing.T) {
		marker := errors.New("property value marshal")
		marshals, calls := 0, 0
		option := WithImageRecordProperty("vendor", imageRecordMarshalCallback(func() ([]byte, error) { marshals++; return nil, marker }))
		if marshals != 1 {
			t.Fatal(marshals)
		}
		var handler taskCoreTransport
		service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) }))
		seed := imageRecordUpdateFetched(t, service, `{"id":"fixed"}`, &handler)
		handler = func(req *http.Request) (*http.Response, error) {
			t.Fatal("failed value reached HTTP", req.URL)
			return nil, nil
		}
		got, err := service.UpdateImagePropertiesRecord(context.Background(), ImageRecordPropertiesRequest{Record: seed}, option)
		if got != nil || !errors.Is(err, marker) || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || marshals != 1 {
			t.Fatal(got, err, calls, marshals)
		}
	})
}

func TestImageRecordPropertiesSnapshotsCoverDiscoveryAndCommitWithoutRecapture(t *testing.T) {
	var handler taskCoreTransport
	calls, locations, callbacks := 0, 0, 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
	seed := imageRecordUpdateFetched(t, New(client), `{"id":"fixed","cached":7}`, &handler)
	headers := map[string]string{"X-Option": "factory"}
	factory := WithImageRecordPropertiesHeaders(headers)
	headers["X-Option"] = "caller changed"
	cloud := "captured"
	facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"captured project"`)}}
	var options []ImageRecordPropertiesOption
	var retained *ImageRecordPropertiesOpts
	client.MoreHeaders, client.Microversion = map[string]string{"X-Source": "captured"}, "2.10"
	handler = func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("X-Source") != "captured" || req.Header.Get("X-Option") != "factory" || req.Header.Get("X-Final") != "yes" || req.Header.Get("X-Auth-Token") != "live token" || req.Header.Get("OpenStack-API-Version") != "image 2.10" {
			t.Fatal(req.Header)
		}
		if calls == 2 {
			if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images" || req.URL.RawQuery != "" || req.Header.Get("Accept") != "application/json" {
				t.Fatal(req.Method, req.URL, req.Header)
			}
			retained.Headers["X-Option"] = "retained changed"
			retained.Properties[2] = 'X'
			retained.Meta[2] = 'X'
			client.MoreHeaders["X-Source"] = "ordinary changed after list"
			return taskCoreJSON(req, 203, `{"images":[{"id":"resolved","name":"match","status":"active"}]}`), nil
		}
		if calls != 3 || req.Method != http.MethodPatch || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed" || req.URL.RawQuery != "" || req.Header.Get("Content-Type") != "application/openstack-images-v2.1-json-patch" || req.Header.Get("Accept") != "" {
			t.Fatal("helper recaptured target/header or HTTP phase", calls, req.Method, req.URL, req.Header)
		}
		imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"add","path":"/kernel_id","value":"resolved"},{"op":"add","path":"/meta","value":false},{"op":"add","path":"/vendor","value":"{'n': 1}"}]`)
		return taskCoreJSON(req, 203, "opaque snapshot response"), nil
	}
	service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
		locations++
		seed.bodyState.current["id"] = json.RawMessage(`"location target decoy"`)
		seed.bodyState.current["properties"] = json.RawMessage(`{"cached":99}`)
		options[0] = WithImageRecordPropertiesHeader("X-Option", "location changed option")
		client.MoreHeaders["X-Source"] = "ordinary changed by location"
		return facts, nil
	}})
	options = []ImageRecordPropertiesOption{factory, WithImageRecordPropertiesJSON(json.RawMessage(`{"kernel":"match","vendor":{"n":1}}`)), WithImageRecordPropertyMeta("meta", false), func(config *ImageRecordPropertiesOpts) error {
		callbacks++
		retained = config
		cloud = "option changed location"
		facts.Project.ID[1] = 'X'
		client.SetToken("live token")
		return WithImageRecordPropertiesHeader("X-Final", "yes")(config)
	}}
	got, err := service.UpdateImagePropertiesRecord(context.Background(), ImageRecordPropertiesRequest{Record: seed}, options...)
	if got == nil || got.Record == nil || !got.Updated || err != nil || calls != 3 || locations != 1 || callbacks != 1 || string(got.Record.Resource.Body["id"]) != `"fixed"` {
		t.Fatal(got, err, calls, locations, callbacks)
	}
	th.AssertEquals(t, "7", string(imageRecordProperties(t, got.Record)["cached"]))
	var location resource.CloudLocation
	if err := json.Unmarshal(got.Record.Resource.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != "captured" || string(location.Project.ID) != `"captured project"` {
		t.Fatal(location, err)
	}
}

func TestImageRecordPropertiesControlsParserAndIdentityFailBeforePatch(t *testing.T) {
	for _, key := range []string{"image", "self", "resource_type", "value", "base_path", "microversion", "connection", "_synchronized"} {
		t.Run("bound control/"+key, func(t *testing.T) {
			var handler taskCoreTransport
			calls := 0
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) }))
			seed := imageRecordUpdateFetched(t, service, `{"id":"fixed"}`, &handler)
			handler = func(req *http.Request) (*http.Response, error) {
				t.Fatal("property control changed SDK route/session", req.URL)
				return nil, nil
			}
			got, err := service.UpdateImagePropertiesRecord(context.Background(), ImageRecordPropertiesRequest{Record: seed}, WithImageRecordPropertyMeta(key, "foreign"))
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
				t.Fatal(got, err, calls)
			}
		})
	}
	t.Run("raw meta ID cannot hide behind replacement equality", func(t *testing.T) {
		var handler taskCoreTransport
		calls := 0
		service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) }))
		seed := imageRecordUpdateFetched(t, service, `{"id":"�"}`, &handler)
		handler = func(req *http.Request) (*http.Response, error) {
			t.Fatal("invalid raw meta ID was discarded by equality", req.URL)
			return nil, nil
		}
		got, err := service.UpdateImagePropertiesRecord(context.Background(), ImageRecordPropertiesRequest{Record: seed}, WithImageRecordPropertiesMetaJSON(json.RawMessage(`{"id":"\uD800","name":"after"}`)))
		if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
			t.Fatal(got, err, calls)
		}
	})
	for _, test := range []struct {
		name    string
		input   ImageRecordPropertiesRequest
		options []ImageRecordPropertiesOption
	}{
		{"missing input", ImageRecordPropertiesRequest{}, nil},
		{"both selectors", ImageRecordPropertiesRequest{ID: "fixed", Record: &ImageRecord{}}, nil},
		{"handcrafted record", ImageRecordPropertiesRequest{Record: &ImageRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"fixed"`)}}}}}, nil},
		{"nil option", ImageRecordPropertiesRequest{ID: "fixed"}, []ImageRecordPropertiesOption{nil}},
		{"owned auth", ImageRecordPropertiesRequest{ID: "fixed"}, []ImageRecordPropertiesOption{WithImageRecordPropertiesHeader("X-Auth-Token", "foreign")}},
		{"invalid kwargs JSON", ImageRecordPropertiesRequest{ID: "fixed"}, []ImageRecordPropertiesOption{WithImageRecordPropertiesJSON(json.RawMessage(`{"broken":`))}},
		{"nonobject kwargs", ImageRecordPropertiesRequest{ID: "fixed"}, []ImageRecordPropertiesOption{WithImageRecordPropertiesJSON(json.RawMessage(`[]`))}},
		{"truthy nonobject meta", ImageRecordPropertiesRequest{ID: "fixed"}, []ImageRecordPropertiesOption{WithImageRecordPropertiesMetaJSON(json.RawMessage(`[1]`))}},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				t.Fatal("invalid helper input reached HTTP", req.URL)
				return nil, nil
			}))
			got, err := service.UpdateImagePropertiesRecord(context.Background(), test.input, test.options...)
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(got, err, calls)
			}
			if test.name == "invalid kwargs JSON" {
				var syntax *json.SyntaxError
				if !errors.As(err, &syntax) {
					t.Fatal("parser cause lost", err)
				}
			}
		})
	}
}

func TestImageRecordPropertiesOptionAndContextGuardsStopLaterCallbacks(t *testing.T) {
	for _, mode := range []string{"nil context", "pre canceled", "callback cancel", "callback source", "callback binding", "callback outer", "callback cause", "location cause"} {
		t.Run(mode, func(t *testing.T) {
			marker := errors.New("property helper guard cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			outerInvalid := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerInvalid {
					return marker
				}
				return nil
			})
			var handler taskCoreTransport
			calls, callbacks, later := 0, 0, 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			seed := imageRecordUpdateFetched(t, New(client), `{"id":"fixed"}`, &handler)
			handler = func(req *http.Request) (*http.Response, error) {
				t.Fatal("failed guard performed discovery/PATCH", req.URL)
				return nil, nil
			}
			service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
				if mode == "location cause" {
					return resource.CloudLocation{}, marker
				}
				return resource.CloudLocation{}, nil
			}})
			if mode == "nil context" {
				ctx = nil
			}
			if mode == "pre canceled" {
				cancel(marker)
			}
			got, err := service.UpdateImagePropertiesRecord(ctx, ImageRecordPropertiesRequest{Record: seed}, func(config *ImageRecordPropertiesOpts) error {
				callbacks++
				switch mode {
				case "callback cancel":
					cancel(marker)
				case "callback source":
					client.Endpoint = "https://foreign.test/"
				case "callback binding":
					service.API = nil
				case "callback outer":
					outerInvalid = true
				case "callback cause":
					return marker
				}
				return WithImageRecordProperty("kernel", "must not list")(config)
			}, func(*ImageRecordPropertiesOpts) error { later++; return nil })
			wantCallbacks := 1
			if mode == "nil context" || mode == "pre canceled" || mode == "location cause" {
				wantCallbacks = 0
			}
			if got != nil || err == nil || calls != 1 || callbacks != wantCallbacks || later != 0 {
				t.Fatal(got, err, calls, callbacks, later)
			}
			if mode == "callback source" || mode == "callback binding" {
				if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			} else if mode != "nil context" && !errors.Is(err, marker) {
				t.Fatal("guard cause lost", err)
			}
			if strings.Contains(mode, "cancel") && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}

func TestImageRecordPropertiesCompleteListFailuresPrecedeNumericAliasSelection(t *testing.T) {
	for _, lateFailure := range []bool{false, true} {
		t.Run(fmt.Sprintf("late failure=%v", lateFailure), func(t *testing.T) {
			var handler taskCoreTransport
			calls := 0
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) }))
			seed := imageRecordUpdateFetched(t, service, `{"id":"fixed"}`, &handler)
			handler = func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images" {
					t.Fatal("numeric alias must consume the complete image list", req.Method, req.URL)
				}
				if calls == 2 {
					th.AssertEquals(t, "", req.URL.RawQuery)
					if lateFailure {
						return taskCoreJSON(req, 203, `{"images":[{"id":"unused","status":"active"}],"next":"?marker=later"}`), nil
					}
					return taskCoreJSON(req, 203, `{"images":[]}`), nil
				}
				if !lateFailure || calls != 3 || req.URL.RawQuery != "marker=later" {
					t.Fatal(calls, req.URL)
				}
				return taskCoreJSON(req, 500, "late list failure precedes alias selection"), nil
			}
			got, err := service.UpdateImagePropertiesRecord(context.Background(), ImageRecordPropertiesRequest{Record: seed}, WithImageRecordProperty("kernel", json.RawMessage(`1e9999`)))
			if lateFailure {
				var native gophercloud.ErrUnexpectedResponseCode
				if got != nil || err == nil || calls != 3 || !errors.As(err, &native) || native.Actual != 500 || string(native.Body) != "late list failure precedes alias selection" {
					t.Fatal(got, err, calls, native)
				}
			} else {
				// The shared Python string policy renders the nonfinite float as
				// "inf". A successfully empty list therefore selects raw null;
				// it leaves the empty cached property map false without PATCH.
				if got == nil || got.Record == nil || got.Updated || err != nil || calls != 2 || string(got.Record.Envelope) != `{"id":"fixed"}` {
					t.Fatal(got, err, calls)
				}
			}
		})
	}
}
