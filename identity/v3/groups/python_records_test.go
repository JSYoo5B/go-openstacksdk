package groups_test

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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func groupRecordAPI(t *testing.T, reply func(*http.Request) *http.Response) (*groups.API, *testcloud.Cloud) {
	t.Helper()
	fixture := testcloud.New(t)
	fixture.SetTransport(testcloud.LinkRequest(func(r *http.Request) (*http.Response, error) { return reply(r), nil }))
	client := fixture.Client("identity", "/catalog/")
	client.ResourceBase = fixture.Server.URL + "/keystone/v3/"
	return groups.New(client), fixture
}
func groupRecordWire(code int, body string) *http.Response {
	return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"group-proof"}})
}
func groupRecordField(t *testing.T, r *groups.GroupRecord, key, want string) {
	t.Helper()
	if r == nil || r.Resource == nil || string(r.Resource.Body[key]) != want {
		t.Fatalf("%s want %s: %+v", key, want, r)
	}
}

type groupRecordMarshaler struct{ calls *int }

func (v groupRecordMarshaler) MarshalJSON() ([]byte, error) {
	*v.calls++
	return nil, errors.New("must not marshal unknown argument")
}

func TestPythonGroupRecordsLazyUntypedDeclaredViewAndEvidence(t *testing.T) {
	calls, encoded := 0, 0
	body := `{"groups":[{"id":"one","name":false,"description":0.10000000000000001,"domain_id":3,"links":{"self":"ignored"},"x_extension":{"n":9007199254740993},"location":{"cloud":"untrusted"}},{"id":null}]}`
	api, _ := groupRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.Method != "GET" || r.URL.Path != "/keystone/v3/groups" || r.URL.RawQuery != "domain_id=d&name=n" || r.Body != nil || r.Header.Get("Accept") != "application/json" || r.Header.Get("OpenStack-API-Version") != "" {
			t.Fatal(r.Method, r.URL, r.Header, r.Body)
		}
		return groupRecordWire(203, body)
	})
	seq := api.ListRecords(context.Background(), groups.WithRecordListQuery("domain_id", "d"), groups.WithRecordListQuery("name", "n"), groups.WithRecordListQuery("x_extension", groupRecordMarshaler{&encoded}))
	if calls != 0 || encoded != 0 {
		t.Fatal(calls, encoded)
	}
	var rows []*groups.GroupRecord
	for row, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if len(rows) != 2 || calls != 1 || encoded != 0 {
		t.Fatal(rows, calls, encoded)
	}
	for key, want := range map[string]string{"name": "false", "description": "0.10000000000000001", "domain_id": "3", "location": "null"} {
		groupRecordField(t, rows[0], key, want)
	}
	for _, key := range []string{"links", "x_extension"} {
		if _, present := rows[0].Resource.Body[key]; present {
			t.Fatal(key, rows[0].Resource)
		}
	}
	for _, key := range []string{"id", "name", "description", "domain_id"} {
		groupRecordField(t, rows[1], key, "null")
	}
	if string(rows[0].Wire.Body["x_extension"]) != `{"n":9007199254740993}` || string(rows[0].Wire.Body["location"]) != `{"cloud":"untrusted"}` || string(rows[0].Envelope) != body || rows[0].StatusCode != 203 || rows[0].Header.Get("X-Request-Id") != "group-proof" {
		t.Fatal(rows[0])
	}
	clone := rows[0].Clone()
	clone.Resource.Body["id"][1] = 'X'
	clone.Envelope[0] = '['
	clone.Header.Set("X-Request-Id", "mutated")
	rows[0].Resource.Body["id"][1] = 'Y'
	rows[0].Header.Set("X-Request-Id", "public")
	if string(rows[0].Wire.Body["id"]) != `"one"` || rows[0].Wire.Header.Get("X-Request-Id") != "group-proof" || string(rows[0].Envelope) != body || rows[1].Header.Get("X-Request-Id") != "group-proof" {
		t.Fatal(rows)
	}
}
func TestPythonGroupFindRecordGETSeedAndPassiveNullID(t *testing.T) {
	for _, body := range []string{`{"group":{"name":"server"}}`, `{"group":{"id":null,"description":null}}`, `{"description":false}`, `not-json`} {
		t.Run(body, func(t *testing.T) {
			calls := 0
			api, _ := groupRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				if r.URL.Path != "/keystone/v3/groups/alias" || r.URL.Query().Get("domain_id") != "seed-domain" || r.URL.Query().Get("x_request") != "unknown" || r.Body != nil {
					t.Fatal(r.URL, r.Body)
				}
				return groupRecordWire(203, body)
			})
			row, err := api.FindRecord(context.Background(), "alias", groups.WithRecordFindQuery("domain_id", "seed-domain"), groups.WithRecordFindQuery("description", "seed"), groups.WithRecordFindQuery("x_request", "unknown"))
			if err != nil || calls != 1 || row == nil || row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "group-proof" || string(row.Envelope) != body {
				t.Fatal(row, err, calls)
			}
			groupRecordField(t, row, "domain_id", `"seed-domain"`)
			if strings.Contains(body, `"id":null`) {
				groupRecordField(t, row, "id", "null")
				groupRecordField(t, row, "description", "null")
			} else {
				groupRecordField(t, row, "id", `"alias"`)
			}
			if _, exists := row.Resource.Body["x_request"]; exists {
				t.Fatal(row.Resource)
			}
		})
	}
}
func TestPythonGroupFindRecordCleanFallbackAndExplicitNamePresence(t *testing.T) {
	for _, status := range []int{400, 403, 404} {
		for _, explicit := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/%t", status, explicit), func(t *testing.T) {
				calls := 0
				var actual url.Values
				api, _ := groupRecordAPI(t, func(r *http.Request) *http.Response {
					calls++
					if calls == 1 {
						return groupRecordWire(status, `{"error":"not id"}`)
					}
					actual = r.URL.Query()
					return groupRecordWire(200, `{"groups":[{"id":"alias","name":"different","description":"keep"}]}`)
				})
				opts := []groups.RecordFindOption{groups.WithRecordFindQuery("domain_id", "d"), groups.WithRecordFindQuery("description", "keep"), groups.WithRecordFindQuery("x_unknown", "GET-only")}
				if explicit {
					opts = append(opts, groups.WithRecordFindQuery("name", nil))
				}
				row, err := api.FindRecord(context.Background(), "alias", opts...)
				if err != nil || row == nil || calls != 2 || actual.Get("domain_id") != "d" || actual.Has("description") || actual.Has("x_unknown") {
					t.Fatal(row, err, calls, actual)
				}
				if explicit && actual.Has("name") || !explicit && actual.Get("name") != "alias" {
					t.Fatal(explicit, actual)
				}
				groupRecordField(t, row, "id", `"alias"`)
			})
		}
	}
}
func TestPythonGroupFindRecordMissingRepeatedIDAndLateFailure(t *testing.T) {
	for _, mode := range []string{"missing", "strict", "duplicate", "late403", "late503"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			api, _ := groupRecordAPI(t, func(*http.Request) *http.Response {
				calls++
				if calls == 1 {
					return groupRecordWire(404, `{"error":"fallback"}`)
				}
				if mode == "missing" || mode == "strict" {
					return groupRecordWire(200, `{"groups":[]}`)
				}
				if calls == 2 {
					return groupRecordWire(200, `{"groups":[{"id":"repeat","name":"same"}],"next":"?marker=repeat&name=same"}`)
				}
				if mode == "duplicate" {
					return groupRecordWire(200, `{"groups":[{"id":"repeat","name":"same"}]}`)
				}
				code := 503
				if mode == "late403" {
					code = 403
				}
				return groupRecordWire(code, `{"error":"late actual"}`)
			})
			opts := []groups.RecordFindOption{}
			if mode == "strict" {
				opts = append(opts, groups.WithRecordFindIgnoreMissing(false))
			}
			row, err := api.FindRecord(context.Background(), "same", opts...)
			switch mode {
			case "missing":
				if row != nil || err != nil || calls != 2 {
					t.Fatal(row, err, calls)
				}
			case "strict":
				if row != nil || !errors.Is(err, resource.ErrNotFound) {
					t.Fatal(row, err)
				}
			case "duplicate":
				if row != nil || !errors.Is(err, resource.ErrAmbiguous) || calls != 3 {
					t.Fatal(row, err, calls)
				}
			default:
				var proof gophercloud.ErrUnexpectedResponseCode
				code := 503
				if mode == "late403" {
					code = 403
				}
				if row != nil || !errors.As(err, &proof) || proof.Actual != code || string(proof.Body) != `{"error":"late actual"}` || proof.ResponseHeader.Get("X-Request-Id") != "group-proof" || proof.Method != "GET" || !strings.Contains(proof.URL, "/keystone/v3/groups?marker=repeat") || calls != 3 {
					t.Fatal(row, err, proof, calls)
				}
			}
		})
	}
}
func TestPythonGroupRecordsSourceDictionaryLinksAndPresence(t *testing.T) {
	for _, links := range []string{`,"links":{"next":"https://untrusted.invalid/v3/groups"}`, `,"links":{"next":false},"groups_links":[{"rel":"next","href":"https://untrusted.invalid/groups"}]`, `,"links":null,"groups_links":[{"rel":"next","href":"https://untrusted.invalid/groups"}]`} {
		calls := 0
		api, _ := groupRecordAPI(t, func(*http.Request) *http.Response {
			calls++
			return groupRecordWire(200, `{"groups":[{"id":"a"}]`+links+`}`)
		})
		rows, err := api.AllRecords(context.Background())
		if strings.Contains(links, `"links":null`) {
			var proof *resource.ResponseError
			if err == nil || rows != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-Id") != "group-proof" || string(proof.Body) != `{"groups":[{"id":"a"}]`+links+`}` || calls != 1 {
				t.Fatal(links, rows, err, proof, calls)
			}
		} else if err != nil || len(rows) != 1 || calls != 1 {
			t.Fatal(links, rows, err, calls)
		}
	}
}
func TestPythonGroupRecordsFilteredMarkerRawCapAndBreak(t *testing.T) {
	for _, mode := range []string{"marker", "cap", "break"} {
		t.Run(mode, func(t *testing.T) {
			var markers []string
			api, _ := groupRecordAPI(t, func(r *http.Request) *http.Response {
				markers = append(markers, r.URL.Query().Get("marker"))
				if r.URL.Query().Get("limit") != "5" || r.URL.Query().Has("description") {
					t.Fatal(r.URL)
				}
				if len(markers) == 1 {
					return groupRecordWire(200, `{"groups":[{"id":"a","description":"keep"},{"id":"last-filtered","description":"skip"}],"links":{"next":"ignored.invalid"}}`)
				}
				if len(markers) == 2 {
					return groupRecordWire(200, `{"groups":[{"id":"short","description":"keep"}]}`)
				}
				return groupRecordWire(200, `{"groups":[]}`)
			})
			opts := []groups.RecordListOption{groups.WithRecordListLimit(5), groups.WithRecordListQuery("description", "keep")}
			if mode == "cap" {
				opts = append(opts, groups.WithRecordListMaxItems(2))
			}
			if mode == "break" {
				for row, err := range api.ListRecords(context.Background(), opts...) {
					if err != nil || row == nil {
						t.Fatal(row, err)
					}
					row.Resource.Body["id"] = json.RawMessage(`"caller-mutated"`)
					break
				}
				if !reflect.DeepEqual(markers, []string{""}) {
					t.Fatal(markers)
				}
				return
			}
			rows, err := api.AllRecords(context.Background(), opts...)
			if mode == "cap" {
				if err != nil || len(rows) != 1 || !reflect.DeepEqual(markers, []string{""}) {
					t.Fatal(rows, err, markers)
				}
				return
			}
			if err != nil || len(rows) != 2 || !reflect.DeepEqual(markers, []string{"", "last-filtered", "short"}) {
				t.Fatal(rows, err, markers)
			}
		})
	}
}
func TestPythonGroupRecordsOrderedLoadedLocalEquality(t *testing.T) {
	for _, tc := range []struct {
		actual, filter string
		want           int
	}{{"0.10000000000000001", "0.1", 1}, {"1e-4000", "false", 1}, {"9007199254740993.0", "9007199254740992", 1}, {"9007199254740993.0", "9007199254740993", 0}, {`{"nested":{"a":true,"b":2}}`, `{"nested":{"a":1}}`, 1}, {`"truthy"`, `{}`, 1}, {`0`, `{}`, 0}} {
		api, _ := groupRecordAPI(t, func(*http.Request) *http.Response {
			return groupRecordWire(200, `{"groups":[{"id":"a","description":`+tc.actual+`}]}`)
		})
		rows, err := api.AllRecords(context.Background(), groups.WithRecordListQuery("description", json.RawMessage(tc.filter)))
		if err != nil || len(rows) != tc.want {
			t.Fatal(tc, rows, err)
		}
	}
	api, _ := groupRecordAPI(t, func(*http.Request) *http.Response {
		return groupRecordWire(200, `{"groups":[{"id":"a","description":"skip","domain_id":"d"}]}`)
	})
	rows, err := api.AllRecords(context.Background(), groups.WithRecordListQuery("description", "mismatch"), groups.WithRecordListQuery("id", map[string]any{"x": 1}))
	if err != nil || len(rows) != 0 {
		t.Fatal(rows, err)
	}
	rows, err = api.AllRecords(context.Background(), groups.WithRecordListQuery("id", map[string]any{"x": 1}), groups.WithRecordListQuery("description", "mismatch"))
	var proof *resource.ResponseError
	if err == nil || rows != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-Id") != "group-proof" || string(proof.Body) != `{"groups":[{"id":"a","description":"skip","domain_id":"d"}]}` {
		t.Fatal(rows, err, proof)
	}
}
func TestPythonGroupRecordsProxyExpressionsRemainOpaqueUntilConsumed(t *testing.T) {
	calls := 0
	api, _ := groupRecordAPI(t, func(*http.Request) *http.Response { calls++; return groupRecordWire(200, `{"groups":[{"id":"a"}]}`) })
	for expr, want := range map[string]string{"[0].id": "null", "type(@)": "null", "`[]`": "[]", "`[1,false,null]`": "[1,false,null]"} {
		result, err := api.ProxyListRecords(context.Background(), groups.WithRecordListJMESPath(expr))
		if err != nil || result == nil || string(result.JSON) != want || calls != 0 {
			t.Fatal(expr, result, err, calls)
		}
	}
	result, err := api.ProxyListRecords(context.Background(), groups.WithRecordListJMESPath("@"))
	if err != nil || result.JSON != nil || calls != 0 {
		t.Fatal(result, err, calls)
	}
	generator, ok := result.Value.(*groups.RecordListGenerator)
	if !ok {
		t.Fatal(result)
	}
	for row, err := range generator.Records {
		if err != nil || row == nil {
			t.Fatal(row, err)
		}
	}
	if calls != 1 {
		t.Fatal(calls)
	}
	result, err = api.ProxyListRecords(context.Background(), groups.WithRecordListJMESPath("to_array(@)"))
	if err != nil || result.JSON != nil {
		t.Fatal(result, err)
	}
	array, ok := result.Value.([]any)
	if !ok || len(array) != 1 {
		t.Fatal(result)
	}
	if _, ok := array[0].(*groups.RecordListGenerator); !ok {
		t.Fatal(result)
	}
	before := calls
	for _, expr := range []string{"length(@)", "["} {
		_, err := api.ProxyListRecords(context.Background(), groups.WithRecordListJMESPath(expr))
		if !errors.Is(err, resource.ErrInvalidOption) || calls != before {
			t.Fatal(expr, err, calls)
		}
	}
	for _, raw := range []string{"null", "false", "0", `""`, "true", "[]"} {
		result, err := api.ProxyListRecords(context.Background(), groups.WithRecordListQuery("jmespath_filters", json.RawMessage(raw)))
		if err != nil {
			t.Fatal(raw, err)
		}
		if _, ok := result.Value.(*groups.RecordListGenerator); !ok {
			t.Fatal(raw, result)
		}
	}
	if calls != before {
		t.Fatal(calls)
	}
}
func TestPythonGroupRecordsOwnedOptionsStickySourceAndCancellation(t *testing.T) {
	calls := 0
	api, _ := groupRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.URL.Query().Get("domain_id") != "owned" || r.Header.Get("X-Owned") != "owned" {
			t.Fatal(r.URL, r.Header)
		}
		return groupRecordWire(200, `{"groups":[]}`)
	})
	raw := json.RawMessage(`"owned"`)
	headers := map[string]string{"X-Owned": "owned"}
	opts := groups.WithRecordListOpts(groups.RecordListOpts{Query: map[string]json.RawMessage{"domain_id": raw}, Headers: headers})
	raw[1] = 'X'
	headers["X-Owned"] = "changed"
	if _, err := api.AllRecords(context.Background(), opts); err != nil || calls != 1 {
		t.Fatal(err, calls)
	}
	for _, mode := range []string{"source", "cancel", "nil"} {
		t.Run(mode, func(t *testing.T) {
			before := calls
			callbacks := 0
			client := api.RawClient()
			endpoint := client.Endpoint
			defer func() { client.Endpoint = endpoint }()
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("custom group cancel")
			change := groups.RecordListOption(func(*groups.RecordListOpts) error {
				if mode == "source" {
					client.Endpoint += "changed"
				} else if mode == "cancel" {
					cancel(cause)
				}
				return nil
			})
			if mode == "nil" {
				change = nil
			}
			restore := groups.RecordListOption(func(*groups.RecordListOpts) error { callbacks++; client.Endpoint = endpoint; return nil })
			_, err := api.AllRecords(ctx, change, restore)
			var proof *resource.ResponseError
			if err == nil || errors.As(err, &proof) || calls != before || callbacks != 0 {
				t.Fatal(err, proof, calls, callbacks)
			}
			if mode == "cancel" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
		})
	}
}
func TestPythonGroupRecordsSourceLocationIsExplicitAndOwned(t *testing.T) {
	api, _ := groupRecordAPI(t, func(*http.Request) *http.Response {
		return groupRecordWire(200, `{"group":{"id":"server","location":{"cloud":"foreign"},"project_id":"foreign"}}`)
	})
	cloudName, projectName := "configured", "current"
	location := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &projectName}}
	opt := groups.WithRecordFindLocation(location)
	cloudName = "mutated"
	projectName = "mutated"
	row, err := api.FindRecord(context.Background(), "fixed", opt)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]json.RawMessage
	_ = json.Unmarshal(row.Resource.Body["location"], &value)
	var project map[string]json.RawMessage
	_ = json.Unmarshal(value["project"], &project)
	if string(value["cloud"]) != `"configured"` || string(project["id"]) != `"scope"` || string(project["name"]) != `"current"` || string(row.Wire.Body["location"]) != `{"cloud":"foreign"}` {
		t.Fatal(row)
	}
	row, err = api.FindRecord(context.Background(), "fixed")
	if err != nil {
		t.Fatal(err)
	}
	groupRecordField(t, row, "location", "null")
}
func TestPythonGroupFindRecordsGETOnlyMicroversionAndListBasePath(t *testing.T) {
	calls := 0
	var paths, versions []string
	api, _ := groupRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		paths = append(paths, r.URL.Path)
		versions = append(versions, r.Header.Get("OpenStack-API-Version"))
		if calls == 1 {
			return groupRecordWire(404, `{}`)
		}
		return groupRecordWire(200, `{"groups":[{"id":"a","name":"alias"}]}`)
	})
	api.RawClient().Microversion = "3.5"
	row, err := api.FindRecord(context.Background(), "alias", groups.WithRecordFindMicroversion("3.9"), groups.WithRecordFindListBasePath("other_groups"))
	if err != nil || row == nil || !reflect.DeepEqual(paths, []string{"/keystone/v3/groups/alias", "/keystone/v3/other_groups"}) || !reflect.DeepEqual(versions, []string{"identity 3.9", "identity 3.5"}) {
		t.Fatal(row, err, paths, versions)
	}
	before := calls
	_, err = api.AllRecords(context.Background(), groups.WithRecordListBasePath("https://foreign.invalid/groups"))
	if !errors.Is(err, resource.ErrInvalidOption) || calls != before {
		t.Fatal(err, calls)
	}
}

type groupFaultBody struct {
	first             bool
	closes            *int
	readErr, closeErr error
	read, close       func()
}

func (b *groupFaultBody) Read(p []byte) (int, error) {
	if b.first {
		return 0, io.EOF
	}
	b.first = true
	if b.read != nil {
		b.read()
	}
	n := copy(p, `{"groups":`)
	if b.readErr != nil {
		return n, b.readErr
	}
	return n, io.EOF
}
func (b *groupFaultBody) Close() error {
	*b.closes++
	if b.close != nil {
		b.close()
	}
	return b.closeErr
}
func TestPythonGroupRecordsFaultedActualRejectionsNeverFallbackOrRetry(t *testing.T) {
	for _, code := range []int{200, 400, 403, 404, 401, 429, 503} {
		for _, phase := range []string{"read", "close", "source", "cancel"} {
			t.Run(fmt.Sprintf("%d/%s", code, phase), func(t *testing.T) {
				calls, closes, hooks := 0, 0, 0
				cause := errors.New("group physical fault")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				var api *groups.API
				api, fixture := groupRecordAPI(t, func(*http.Request) *http.Response {
					calls++
					b := &groupFaultBody{closes: &closes}
					switch phase {
					case "read":
						b.readErr = cause
					case "close":
						b.closeErr = cause
					case "source":
						b.read = func() { api.RawClient().Endpoint += "changed" }
						b.close = func() { api.RawClient().Endpoint = strings.TrimSuffix(api.RawClient().Endpoint, "changed") }
					case "cancel":
						b.read = func() { cancel(cause) }
					}
					response := groupRecordWire(code, "")
					response.Body = b
					return response
				})
				fixture.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					hooks++
					return nil
				}
				fixture.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error { hooks++; return nil }
				fixture.Provider.ReauthFunc = func(context.Context) error { hooks++; return nil }
				fixture.Provider.MaxBackoffRetries = 1
				row, err := api.FindRecord(ctx, "alias")
				if err == nil || row != nil || calls != 1 || closes != 1 || hooks != 0 {
					t.Fatal(row, err, calls, closes, hooks)
				}
				if code < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != code || proof.Header.Get("X-Request-Id") != "group-proof" || string(proof.Body) != `{"groups":` {
						t.Fatal(err, proof)
					}
				} else {
					original := err
					if code == 401 {
						var auth *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &auth) || auth.ErrOriginal == nil || auth.ErrReauth == nil {
							t.Fatal(err, auth)
						}
						original = auth.ErrOriginal
					}
					var proof gophercloud.ErrUnexpectedResponseCode
					if !errors.As(original, &proof) || proof.Actual != code || proof.ResponseHeader.Get("X-Request-Id") != "group-proof" || string(proof.Body) != `{"groups":` || proof.Method != "GET" || !strings.Contains(proof.URL, "/keystone/v3/groups/alias") {
						t.Fatal(err, proof)
					}
				}
				if code != 401 && phase != "source" && !errors.Is(err, cause) {
					t.Fatal(err, cause)
				}
				if code == 401 {
					var reauth *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &reauth) || reauth.ErrReauth == nil {
						t.Fatal(err, reauth)
					}
					// The native 401 wrapper has no Unwrap: its original response
					// and failed reauthentication are independently inspectable.
					// Gophercloud stores ErrOriginal as a response-error value.
					var original gophercloud.ErrUnexpectedResponseCode
					if !errors.As(reauth.ErrOriginal, &original) || original.Actual != 401 || original.Method != "GET" || !strings.Contains(original.URL, "/groups/alias") || string(original.Body) != `{"groups":` || original.ResponseHeader.Get("X-Request-Id") != "group-proof" {
						t.Fatal(err, original)
					}
					if phase == "source" {
						if !errors.Is(reauth.ErrReauth, resource.ErrInvalidOption) {
							t.Fatal(err, reauth.ErrReauth)
						}
					} else if !errors.Is(reauth.ErrReauth, cause) {
						t.Fatal(err, reauth.ErrReauth, cause)
					}
					if phase == "cancel" && !errors.Is(reauth.ErrReauth, context.Canceled) {
						t.Fatal(err, reauth.ErrReauth)
					}
				}
			})
		}
	}
}
func TestPythonGroupRecordsCleanNativeRetryUsesLiveAuthAndCapturedHeaders(t *testing.T) {
	for _, code := range []int{401, 503, 429, 498} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls, reauth, retry, backoff := 0, 0, 0, 0
			api, fixture := groupRecordAPI(t, func(r *http.Request) *http.Response {
				calls++
				if r.Method != "GET" || r.Body != nil || r.Header.Get("X-Snapshot") != "before" || r.Header.Get("X-Late") != "" {
					t.Fatal(r.Header, r.Body)
				}
				if calls == 1 {
					if r.Header.Get("X-Auth-Token") != "test-token" {
						t.Fatal(r.Header)
					}
					return groupRecordWire(code, `{"error":"native clean"}`)
				}
				if r.Header.Get("X-Auth-Token") != "refreshed" {
					t.Fatal(r.Header)
				}
				return groupRecordWire(203, `{"groups":[{"id":"a"}]}`)
			})
			api.RawClient().MoreHeaders = map[string]string{"X-Snapshot": "before"}
			update := func() {
				fixture.Provider.SetToken("refreshed")
				api.RawClient().MoreHeaders["X-Snapshot"] = "after"
				api.RawClient().MoreHeaders["X-Late"] = "after"
			}
			fixture.Provider.ReauthFunc = func(context.Context) error { reauth++; update(); return nil }
			fixture.Provider.RetryFunc = func(ctx context.Context, method, u string, opts *gophercloud.RequestOpts, _ error, count uint) error {
				retry++
				update()
				return nil
			}
			fixture.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				backoff++
				update()
				return nil
			}
			fixture.Provider.MaxBackoffRetries = 1
			rows, err := api.AllRecords(context.Background())
			if err != nil || len(rows) != 1 || rows[0].StatusCode != 203 || calls != 2 {
				t.Fatal(rows, err, calls)
			}
			if code == 401 && (reauth != 1 || retry != 0 || backoff != 0) || code == 503 && (retry != 1 || reauth != 0 || backoff != 0) || (code == 429 || code == 498) && (backoff != 1 || reauth != 0 || retry != 0) {
				t.Fatal(reauth, retry, backoff)
			}
		})
	}
}

func TestPythonGroupRecordsProxyFormalRestorationAndBoundCollision(t *testing.T) {
	calls := 0
	api, _ := groupRecordAPI(t, func(r *http.Request) *http.Response {
		calls++
		if r.URL.RawQuery != "domain_id=restored" || r.Header.Get("X-Raw") != "restored" || r.Header.Get("OpenStack-API-Version") != "identity 3.7" || r.URL.Path != "/keystone/v3/groups" {
			t.Fatal(r.URL, r.Header)
		}
		return groupRecordWire(200, `{"groups":[{"id":"a"}]}`)
	})
	rows, err := api.AllRecords(context.Background(),
		groups.WithRecordListQuery("domain_id", "original"),
		groups.WithRecordListQuery("headers", map[string]string{"X-Raw": "original"}),
		groups.WithRecordListQuery("microversion", "3.1"),
		groups.WithRecordListQuery("__conflicting_attrs", json.RawMessage(`{"domain_id":"restored","headers":{"X-Raw":"restored"},"microversion":"3.7","x_unknown":{"n":1},"jmespath_filters":"length(@)"}`)),
	)
	if err != nil || len(rows) != 1 || calls != 1 {
		t.Fatal(rows, err, calls)
	}
	before := calls
	for _, raw := range []string{`{"paginated":false}`, `{"base_path":null}`, `{"resource_type":1}`, `true`} {
		_, err := api.AllRecords(context.Background(), groups.WithRecordListQuery("__conflicting_attrs", json.RawMessage(raw)))
		var proof *resource.ResponseError
		if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || calls != before {
			t.Fatal(raw, err, proof, calls)
		}
	}
	result, err := api.ProxyListRecords(context.Background(), groups.WithRecordListJMESPath("`[]`"), groups.WithRecordListQuery("__conflicting_attrs", json.RawMessage(`{"jmespath_filters":"length(@)","x_unknown":true}`)))
	if err != nil || result == nil || string(result.JSON) != "[]" || calls != before {
		t.Fatal(result, err, calls)
	}
}
