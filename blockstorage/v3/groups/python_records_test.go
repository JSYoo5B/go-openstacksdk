package groups_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type groupCall struct {
	method, path, escaped, query, body string
	headers                            http.Header
}
type groupTransport func(*http.Request) (*http.Response, error)

func (f groupTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	v, err := f(r)
	if v != nil && v.Request == nil {
		v.Request = r
	}
	return v, err
}
func groupWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"actual"}}, Body: io.NopCloser(strings.NewReader(body))}
}

const groupBase = "/cinder/v3/project/groups"

func groupAPI(t *testing.T, version string, reply func(*http.Request) *http.Response) (*groups.API, *[]groupCall) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := &[]groupCall{}
	cloud.Provider.HTTPClient.Transport = groupTransport(func(r *http.Request) (*http.Response, error) {
		body := ""
		if r.Body != nil {
			raw, _ := io.ReadAll(r.Body)
			body = string(raw)
		}
		*calls = append(*calls, groupCall{r.Method, r.URL.Path, r.URL.EscapedPath(), r.URL.RawQuery, body, r.Header.Clone()})
		return reply(r), nil
	})
	client := cloud.Client("volumev3", "/cinder/v3/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	client.Microversion = version
	return groups.New(client), calls
}
func groupField(t *testing.T, v *groups.GroupRecord, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		t.Fatal(key, want, v)
	}
}
func groupJSON(t *testing.T, got, want string) {
	t.Helper()
	var a, b any
	if json.Unmarshal([]byte(got), &a) != nil || json.Unmarshal([]byte(want), &b) != nil || !reflect.DeepEqual(a, b) {
		t.Fatal(got, want)
	}
}
func collectGroups(seq func(func(*groups.GroupRecord, error) bool)) ([]*groups.GroupRecord, error) {
	var values []*groups.GroupRecord
	for v, err := range seq {
		if err != nil {
			return values, err
		}
		values = append(values, v)
	}
	return values, nil
}

func TestPythonCinderGroupGetProjectionListVolumeAndOwnedProof(t *testing.T) {
	body := `{"group":{"id":null,"name":false,"group_type":9007199254740993,"volumes":"one","volume_types":false,"replication_targets":{"target":true},"project_id":"project","unknown":9007199254740993,"self":"ignored"}}`
	for _, choice := range []string{"absent", "false", "true"} {
		t.Run(choice, func(t *testing.T) {
			api, calls := groupAPI(t, "3.90", func(*http.Request) *http.Response { return groupWire(200, body) })
			options := []groups.RecordGetOption{}
			if choice != "absent" {
				options = append(options, groups.WithRecordGetListVolume(choice == "true"))
			}
			v, err := api.GetRecord(context.Background(), "literal/id %2F", options...)
			if err != nil || len(*calls) != 1 || (*calls)[0].escaped != groupBase+"/literal%2Fid%20%252F" || (*calls)[0].body != "" || (*calls)[0].headers.Get("OpenStack-API-Version") != "volume 3.90" || len(v.Discovery) != 0 {
				t.Fatal(v, err, *calls)
			}
			want := ""
			if choice == "false" {
				want = "list_volume=False"
			}
			if choice == "true" {
				want = "list_volume=True"
			}
			if (*calls)[0].query != want {
				t.Fatal(*calls)
			}
			for key, want := range map[string]string{"id": `null`, "name": `false`, "group_type": `9007199254740993`, "volumes": `["one"]`, "volume_types": `[false]`, "replication_targets": `[{"target":true}]`, "description": `null`, "location": `null`} {
				groupField(t, v, key, want)
			}
			if len(v.Resource.Body) != 17 || v.Resource.Body["unknown"] != nil || string(v.Wire.Body["unknown"]) != "9007199254740993" || v.Resource.Body["self"] != nil || v.StatusCode != 200 || string(v.Envelope) != body {
				t.Fatal(v)
			}
			v.Resource.Body["project_id"][1] = 'X'
			v.Header.Set("X-Request-Id", "changed")
			if string(v.Wire.Body["project_id"]) != `"project"` || v.Wire.Header.Get("X-Request-Id") != "actual" || v.Resource.Header.Get("X-Request-Id") != "actual" {
				t.Fatal("aliased proof", v)
			}
			if _, err := api.GetRecord(context.Background(), "literal/id %2F", options...); err != nil || len(*calls) != 2 {
				t.Fatal("leaf must not invent a shared response cache", err, *calls)
			}
		})
	}
}
func TestPythonCinderGroupGetEnvelopeSeedsAndActualDecodeErrors(t *testing.T) {
	for _, item := range []struct {
		body  string
		valid bool
	}{{`{"group":{}}`, true}, {`{"name":0,"unknown":false}`, true}, {"", true}, {"{bad", true}, {`null`, false}, {`[]`, false}, {`{"group":null}`, false}, {`{"group":[]}`, false}, {string([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}), false}} {
		api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(202, item.body) })
		v, err := api.GetRecord(context.Background(), "seed")
		if v == nil || (err == nil) != item.valid || len(*calls) != 1 || v.StatusCode != 202 || string(v.Envelope) != item.body {
			t.Fatal(item, v, err, *calls)
		}
		if item.valid {
			groupField(t, v, "id", `"seed"`)
		} else {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != item.body || proof.Header.Get("X-Request-Id") != "actual" {
				t.Fatal(err, proof)
			}
		}
	}
	api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(404, `{"missing":true}`) })
	v, err := api.GetRecord(context.Background(), "id")
	if v != nil || !gophercloud.ResponseCodeIs(err, 404) || len(*calls) != 1 {
		t.Fatal(v, err, *calls)
	}
}
func TestPythonCinderGroupCreatePartialNullUnknownAndOptionSnapshots(t *testing.T) {
	fields := map[string]json.RawMessage{"group_type": json.RawMessage(`"type"`), "volume_types": json.RawMessage(`false`), "description": json.RawMessage(`null`), "name": json.RawMessage(`""`), "availability_zone": json.RawMessage(`0`), "unknown": json.RawMessage(`{bad`)}
	headers := map[string]string{"X-Trace": "original"}
	version := "3.38"
	option := groups.WithRecordCreateOpts(groups.RecordCreateOpts{Fields: fields, Headers: headers, Microversion: &version})
	fields["group_type"][1] = 'X'
	headers["X-Trace"] = "changed"
	version = "3.1"
	api, calls := groupAPI(t, "3.20", func(*http.Request) *http.Response {
		return groupWire(202, `{"group":{"id":"created","name":"server","unknown":false}}`)
	})
	v, err := api.CreateRecord(context.Background(), option)
	if err != nil || len(*calls) != 1 || (*calls)[0].method != http.MethodPost || (*calls)[0].path != groupBase || (*calls)[0].headers.Get("X-Trace") != "original" || (*calls)[0].headers.Get("OpenStack-API-Version") != "volume 3.38" {
		t.Fatal(v, err, *calls)
	}
	groupJSON(t, (*calls)[0].body, `{"group":{"group_type":"type","volume_types":false,"description":null,"name":"","availability_zone":0}}`)
	groupField(t, v, "id", `"created"`)
	groupField(t, v, "name", `"server"`)
	groupField(t, v, "volume_types", `[false]`)
	groupField(t, v, "group_type", `"type"`)
	if v.Resource.Body["unknown"] != nil || string(v.Wire.Body["unknown"]) != "false" {
		t.Fatal(v)
	}
}
func TestPythonCinderGroupCreateFromSourceRequiredNullableIDsAndFreshResponse(t *testing.T) {
	api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(202, "") })
	if v, err := api.CreateFromSourceRecord(context.Background()); v != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
		t.Fatal(v, err, *calls)
	}
	if v, err := api.CreateFromSourceRecord(context.Background(), groups.WithRecordSourceSourceGroupID("parent")); v != nil || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
		t.Fatal(v, err, *calls)
	}
	v, err := api.CreateFromSourceRecord(context.Background(), groups.WithRecordSourceGroupSnapshotID(nil), groups.WithRecordSourceSourceGroupID("parent"))
	if err != nil || len(*calls) != 1 || (*calls)[0].path != groupBase+"/action" || (*calls)[0].method != http.MethodPost {
		t.Fatal(v, err, *calls)
	}
	groupJSON(t, (*calls)[0].body, `{"create-from-src":{"group_snapshot_id":null,"source_group_id":"parent","name":null,"description":null}}`)
	for _, key := range []string{"id", "source_group_id", "name"} {
		groupField(t, v, key, `null`)
	}
	api, calls = groupAPI(t, "3.38", func(*http.Request) *http.Response {
		return groupWire(202, `{"group":{"id":"new","group_snapshot_id":"wire"}}`)
	})
	raw := json.RawMessage(`false`)
	option := groups.WithRecordSourceOpts(groups.RecordSourceOpts{GroupSnapshotID: json.RawMessage(`"snapshot"`), SourceGroupID: json.RawMessage(`null`), Name: raw, Description: json.RawMessage(`0`)})
	raw[0] = 'X'
	v, err = api.CreateFromSourceRecord(context.Background(), option)
	if err != nil || len(*calls) != 1 {
		t.Fatal(v, err, *calls)
	}
	groupJSON(t, (*calls)[0].body, `{"create-from-src":{"group_snapshot_id":"snapshot","source_group_id":null,"name":false,"description":0}}`)
	groupField(t, v, "id", `"new"`)
	groupField(t, v, "group_snapshot_id", `"wire"`)
	groupField(t, v, "name", `null`)
}
func TestPythonCinderGroupUpdateDirtyNoopAndResponseMerge(t *testing.T) {
	for _, options := range [][]groups.RecordUpdateOption{nil, {groups.WithRecordUpdateOpts(groups.RecordUpdateOpts{Fields: map[string]json.RawMessage{"unknown": json.RawMessage(`{bad`)}})}, {groups.WithRecordUpdateField("id", "ignored")}} {
		api, calls := groupAPI(t, "", func(*http.Request) *http.Response { t.Fatal("no-op sent HTTP"); return nil })
		v, err := api.UpdateRecord(context.Background(), "seed", options...)
		if err != nil || len(*calls) != 0 || v.StatusCode != 0 || v.Wire != nil || v.Envelope != nil || len(v.Discovery) != 0 {
			t.Fatal(v, err, *calls)
		}
		groupField(t, v, "id", `"seed"`)
		groupField(t, v, "name", `null`)
	}
	api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response {
		return groupWire(202, `{"group":{"description":"server","unknown":0}}`)
	})
	v, err := api.UpdateRecord(context.Background(), "id", groups.WithRecordUpdateField("name", nil), groups.WithRecordUpdateField("description", false), groups.WithRecordUpdateField("add_volumes", "a,b"), groups.WithRecordUpdateField("remove_volumes", ""))
	if err != nil || len(*calls) != 1 || (*calls)[0].method != http.MethodPut || (*calls)[0].path != groupBase+"/id" {
		t.Fatal(v, err, *calls)
	}
	groupJSON(t, (*calls)[0].body, `{"group":{"name":null,"description":false,"add_volumes":"a,b","remove_volumes":""}}`)
	groupField(t, v, "id", `"id"`)
	groupField(t, v, "name", `null`)
	groupField(t, v, "description", `"server"`)
	groupField(t, v, "add_volumes", `"a,b"`)
}
func TestPythonCinderGroupDeleteActionDefaultsStrictMissingAndStatuses(t *testing.T) {
	for _, code := range []int{200, 202, 204, 299, 300, 304, 399} {
		api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(code, "not JSON") })
		if err := api.DeleteRecord(context.Background(), "id"); err != nil || len(*calls) != 1 || (*calls)[0].method != http.MethodPost || (*calls)[0].path != groupBase+"/id/action" {
			t.Fatal(code, err, *calls)
		}
		groupJSON(t, (*calls)[0].body, `{"delete":{"delete-volumes":false}}`)
	}
	api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(404, `{"missing":true}`) })
	if err := api.DeleteRecord(context.Background(), "id"); err != nil || len(*calls) != 1 {
		t.Fatal(err, *calls)
	}
	if err := api.DeleteRecord(context.Background(), "id", groups.WithRecordDeleteIgnoreMissing(false), groups.WithRecordDeleteVolumes(true)); !gophercloud.ResponseCodeIs(err, 404) || len(*calls) != 2 {
		t.Fatal(err, *calls)
	}
	groupJSON(t, (*calls)[1].body, `{"delete":{"delete-volumes":true}}`)
	api, calls = groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(403, `{"policy":false}`) })
	if err := api.DeleteRecord(context.Background(), "id"); !gophercloud.ResponseCodeIs(err, 403) || len(*calls) != 1 {
		t.Fatal(err, *calls)
	}
}
func TestPythonCinderGroupListDeclaredAliasesLocalFiltersAndRawCap(t *testing.T) {
	query := map[string]json.RawMessage{"all_projects": json.RawMessage(`false`), "all_tenants": json.RawMessage(`true`), "name": json.RawMessage(`"target"`), "sort": json.RawMessage(`"created_at:desc"`), "list_volume": json.RawMessage(`{bad`), "unknown": json.RawMessage(`{bad`)}
	option := groups.WithRecordListOpts(groups.RecordListOpts{Query: query})
	query["name"][1] = 'X'
	api, calls := groupAPI(t, "3.38", func(r *http.Request) *http.Response {
		if r.Header.Get("Accept") != "application/json" {
			t.Fatal(r.Header)
		}
		return groupWire(200, `{"groups":[{"id":"a","name":"other"},{"id":"b","name":"target","unknown":9007199254740993},null],"next":"https://wrong.example/groups"}`)
	})
	seq := api.ListRecords(context.Background(), option, groups.WithRecordListLimit(0), groups.WithRecordListMaxItems(2))
	if len(*calls) != 0 {
		t.Fatal("eager list", *calls)
	}
	values, err := collectGroups(seq)
	if err != nil || len(values) != 1 || len(*calls) != 1 || (*calls)[0].path != groupBase+"/detail" {
		t.Fatal(values, err, *calls)
	}
	groupField(t, values[0], "id", `"b"`)
	q, _ := url.ParseQuery((*calls)[0].query)
	if !reflect.DeepEqual(q, url.Values{"all_tenants": {"False"}, "sort": {"created_at:desc"}, "limit": {"2"}}) || string(values[0].Wire.Body["unknown"]) != "9007199254740993" || values[0].Resource.Body["unknown"] != nil {
		t.Fatal(q, values)
	}
}
func TestPythonCinderGroupListLocalDictionaryFailuresKeepActualPage(t *testing.T) {
	for _, item := range []struct {
		value, filter string
		valid, match  bool
	}{{`"truthy"`, `{}`, true, true}, {`"truthy"`, `{"k":true}`, false, false}, {`""`, `{}`, true, false}, {`null`, `null`, true, true}, {`1`, `true`, true, true}} {
		body := `{"groups":[{"id":` + item.value + `}]}`
		api, _ := groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(200, body) })
		values, err := collectGroups(api.ListRecords(context.Background(), groups.WithRecordListOpts(groups.RecordListOpts{Query: map[string]json.RawMessage{"id": json.RawMessage(item.filter)}})))
		if (err == nil) != item.valid || (len(values) == 1) != item.match {
			t.Fatal(item, values, err)
		}
		if !item.valid {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body {
				t.Fatal(err, proof)
			}
		}
	}
	api, _ := groupAPI(t, "3.38", func(*http.Request) *http.Response {
		return groupWire(200, `{"groups":[{"id":"other","name":"scalar"}]}`)
	})
	values, err := collectGroups(api.ListRecords(context.Background(), groups.WithRecordListQuery("id", "target"), groups.WithRecordListQuery("name", map[string]bool{"k": true})))
	if err != nil || len(values) != 0 {
		t.Fatal("first false must short-circuit later local errors", values, err)
	}
}
func TestPythonCinderGroupListDetailsLinksMarkersOffsetsAndConsumerBreak(t *testing.T) {
	for _, kind := range []string{"groups_links", "next", "Link", "dictionary"} {
		t.Run(kind, func(t *testing.T) {
			api, calls := groupAPI(t, "3.38", func(r *http.Request) *http.Response {
				if r.URL.Query().Get("marker") == "last" {
					return groupWire(200, `{"groups":[{"id":"b"}]}`)
				}
				extra := ""
				switch kind {
				case "groups_links":
					extra = `,"groups_links":[{"rel":"next","href":"?marker=last"}]`
				case "next":
					extra = `,"next":"?marker=last"`
				case "dictionary":
					extra = `,"links":{"next":"?marker=last"}`
				}
				reply := groupWire(200, `{"groups":[{"id":"a"}]`+extra+`}`)
				if kind == "Link" {
					reply.Header.Set("Link", `<?marker=last>; rel="next"`)
				}
				return reply
			})
			values, err := collectGroups(api.ListRecords(context.Background(), groups.WithRecordListDetails(false), groups.WithRecordListQuery("sort", "id:asc")))
			if err != nil || len(values) != 2 || len(*calls) != 2 || (*calls)[0].path != groupBase || (*calls)[1].path != groupBase {
				t.Fatal(values, err, *calls)
			}
			for _, call := range *calls {
				q, _ := url.ParseQuery(call.query)
				if q.Get("sort") != "id:asc" {
					t.Fatal(q)
				}
			}
		})
	}
	api, calls := groupAPI(t, "3.38", func(r *http.Request) *http.Response {
		if r.URL.Query().Get("marker") == "9007199254740993" {
			return groupWire(200, `{"groups":[]}`)
		}
		return groupWire(200, `{"groups":{"id":9007199254740993}}`)
	})
	values, err := collectGroups(api.ListRecords(context.Background(), groups.WithRecordListLimit(5)))
	if err != nil || len(values) != 1 || len(*calls) != 2 || !strings.Contains((*calls)[1].query, "marker=9007199254740993") {
		t.Fatal(values, err, *calls)
	}
	api, calls = groupAPI(t, "3.38", func(r *http.Request) *http.Response {
		if r.URL.Query().Get("offset") == "2" {
			return groupWire(200, `{"groups":[]}`)
		}
		return groupWire(200, `{"groups":[{"id":"a"}],"next":"?offset=2&limit=2"}`)
	})
	values, err = collectGroups(api.ListRecords(context.Background(), groups.WithRecordListQuery("offset", 0), groups.WithRecordListLimit(2)))
	if err != nil || len(values) != 1 || len(*calls) != 2 {
		t.Fatal(values, err, *calls)
	}
	api, calls = groupAPI(t, "3.38", func(*http.Request) *http.Response {
		return groupWire(200, `{"groups":[{"id":"a"},null],"next":"https://wrong.example/groups"}`)
	})
	for v, err := range api.ListRecords(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		groupField(t, v, "id", `"a"`)
		break
	}
	if len(*calls) != 1 {
		t.Fatal(*calls)
	}
}
func TestPythonCinderGroupListShapesConstructorCollisionsAndActualLateError(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `{"groups":null}`, `{"groups":false}`, `{"groups":[null]}`, `{"groups":[{"connection":null}]}`, `{"groups":[{"microversion":"3.1"}]}`, `{"groups":[{"_synchronized":true}]}`} {
		api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(200, body) })
		_, err := collectGroups(api.ListRecords(context.Background()))
		var proof *resource.ResponseError
		if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || len(*calls) != 1 {
			t.Fatal(body, err, proof, *calls)
		}
	}
	api, calls := groupAPI(t, "3.38", func(r *http.Request) *http.Response {
		if r.URL.Query().Has("marker") {
			return groupWire(503, `{"second":"actual"}`)
		}
		return groupWire(200, `{"groups":[{"id":"first"}],"next":"?marker=next"}`)
	})
	values, err := collectGroups(api.ListRecords(context.Background()))
	var native gophercloud.ErrUnexpectedResponseCode
	if len(values) != 1 || len(*calls) != 2 || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"second":"actual"}` || native.ResponseHeader.Get("X-Request-Id") != "actual" {
		t.Fatal(values, err, *calls)
	}
}
func TestPythonCinderGroupDiscoveryCeilingSelectedOverridesAndOwnedEvidence(t *testing.T) {
	api, calls := groupAPI(t, "", func(r *http.Request) *http.Response {
		if r.URL.Path == "/cinder/v3/" {
			return groupWire(200, `{"version":{"id":"v3.0","version":"3.90","min_version":"3.0"}}`)
		}
		return groupWire(200, `{"group":{"id":"actual"}}`)
	})
	v, err := api.GetRecord(context.Background(), "id")
	if err != nil || len(*calls) != 2 || (*calls)[0].path != "/cinder/v3/" || (*calls)[0].headers.Get("OpenStack-API-Version") != "" || (*calls)[1].headers.Get("OpenStack-API-Version") != "volume 3.38" || len(v.Discovery) != 1 || v.Discovery[0].StatusCode != 200 {
		t.Fatal(v, err, *calls)
	}
	v.Discovery[0].Envelope[0] = 'X'
	if string(v.Envelope) != `{"group":{"id":"actual"}}` || v.StatusCode != 200 {
		t.Fatal(v)
	}
	api, calls = groupAPI(t, "3.80", func(*http.Request) *http.Response { return groupWire(200, `{"group":{}}`) })
	v, err = api.GetRecord(context.Background(), "id", groups.WithRecordGetMicroversion("3.13"))
	if err != nil || len(*calls) != 1 || (*calls)[0].headers.Get("OpenStack-API-Version") != "volume 3.13" || api.RawClient().Microversion != "3.80" || len(v.Discovery) != 0 {
		t.Fatal(v, err, *calls)
	}
	api, calls = groupAPI(t, "", func(*http.Request) *http.Response { return groupWire(403, `{"discovery":"denied"}`) })
	v, err = api.GetRecord(context.Background(), "id")
	var native gophercloud.ErrUnexpectedResponseCode
	if v != nil || len(*calls) != 1 || !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != `{"discovery":"denied"}` {
		t.Fatal(v, err, *calls)
	}
}
func TestPythonCinderGroupOptionsPreflightUnknownDiscardAndStickySource(t *testing.T) {
	api, calls := groupAPI(t, "", func(*http.Request) *http.Response { t.Fatal("unexpected preflight HTTP"); return nil })
	for _, invoke := range []func() error{
		func() error { _, err := api.GetRecord(nil, "id"); return err },
		func() error { _, err := (*groups.API)(nil).GetRecord(context.Background(), "id"); return err },
		func() error { _, err := api.GetRecord(context.Background(), " "); return err },
		func() error { _, err := api.GetRecord(context.Background(), "id", nil); return err },
		func() error {
			_, err := api.GetRecord(context.Background(), "id", groups.WithRecordGetMicroversion("bad"))
			return err
		},
		func() error {
			_, err := api.CreateRecord(context.Background(), groups.WithRecordCreateField("connection", nil))
			return err
		},
		func() error {
			_, err := api.CreateRecord(context.Background(), groups.WithRecordCreateOpts(groups.RecordCreateOpts{Fields: map[string]json.RawMessage{"name": json.RawMessage(`{bad`)}}))
			return err
		},
		func() error {
			_, err := collectGroups(api.ListRecords(context.Background(), groups.WithRecordListLimit(-1)))
			return err
		},
		func() error {
			_, err := collectGroups(api.ListRecords(context.Background(), groups.WithRecordListQuery("base_path", "/other")))
			return err
		},
		func() error {
			_, err := collectGroups(api.ListRecords(context.Background(), groups.WithRecordListQuery("limit", false)))
			return err
		},
		func() error {
			return api.DeleteRecord(context.Background(), "id", groups.WithRecordDeleteHeaders(map[string]string{"Authorization": "forged"}))
		},
	} {
		if err := invoke(); !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
			t.Fatal(err, *calls)
		}
	}
	cause := errors.New("option failed")
	ctx, cancel := context.WithCancelCause(context.Background())
	_, err := api.CreateRecord(ctx, func(*groups.RecordCreateOpts) error { cancel(cause); return cause })
	if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || len(*calls) != 0 {
		t.Fatal(err, *calls)
	}
	api, calls = groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(202, `{"group":{}}`) })
	_, err = api.CreateRecord(context.Background(), groups.WithRecordCreateField("unknown", make(chan int)))
	if err != nil || len(*calls) != 1 {
		t.Fatal("Source discards unknown before JSON encoding", err, *calls)
	}
	groupJSON(t, (*calls)[0].body, `{"group":{}}`)
	api, calls = groupAPI(t, "3.38", func(*http.Request) *http.Response { t.Fatal("mutated source sent HTTP"); return nil })
	_, err = api.GetRecord(context.Background(), "id", func(*groups.RecordGetOpts) error {
		api.RawClient().ResourceBase = "https://different.example/v3/project/"
		return nil
	})
	if !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
		t.Fatal(err, *calls)
	}
}

func TestPythonCinderGroupGetDeclaredDeprecatedAttrsSeedWithoutWireQuery(t *testing.T) {
	fields := map[string]json.RawMessage{"description": json.RawMessage(`false`), "volumes": json.RawMessage(`"one"`), "unknown": json.RawMessage(`{bad`)}
	option := groups.WithRecordGetOpts(groups.RecordGetOpts{Fields: fields})
	fields["description"][0] = 'X'
	api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(200, `{"group":{"name":"wire"}}`) })
	v, err := api.GetRecord(context.Background(), "id", option, groups.WithRecordGetField("group_type", 9007199254740993))
	if err != nil || len(*calls) != 1 || (*calls)[0].query != "" || (*calls)[0].body != "" {
		t.Fatal(v, err, *calls)
	}
	groupField(t, v, "description", `false`)
	groupField(t, v, "volumes", `["one"]`)
	groupField(t, v, "name", `"wire"`)
	groupField(t, v, "group_type", `9007199254740993`)
	if _, err := api.GetRecord(context.Background(), "id", groups.WithRecordGetField("id", "forged")); !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 1 {
		t.Fatal(err, *calls)
	}
}

func TestPythonCinderGroupListClassCollisionKeepsAcceptedPageAndDirectUnknown(t *testing.T) {
	for _, raw := range []string{`null`, `false`} {
		t.Run(raw, func(t *testing.T) {
			page := `{"groups":[{"id":"wire","cls":` + raw + `}],"extra":"accepted-page"}`
			member := `{"group":{"id":"wire","cls":` + raw + `},"extra":"accepted-member"}`
			api, calls := groupAPI(t, "3.38", func(r *http.Request) *http.Response {
				if r.URL.Path == groupBase+"/wire" {
					return groupWire(200, member)
				}
				return groupWire(202, page)
			})
			values, err := collectGroups(api.ListRecords(context.Background()))
			var proof *resource.ResponseError
			if len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &proof) || proof.StatusCode != 202 || string(proof.Body) != page || proof.Header.Get("X-Request-Id") != "actual" || len(*calls) != 1 {
				t.Fatal(values, err, proof, *calls)
			}
			value, err := api.GetRecord(context.Background(), "wire")
			if err != nil || value == nil || value.Wire == nil || string(value.Wire.Body["cls"]) != raw || value.Resource.Body["cls"] != nil || string(value.Envelope) != member || value.StatusCode != 200 || value.Header.Get("X-Request-Id") != "actual" || len(*calls) != 2 {
				t.Fatal(value, err, *calls)
			}
		})
	}
}

func TestPythonCinderGroupClassInputsAreBoundControlsBeforeDiscovery(t *testing.T) {
	for _, raw := range []string{`null`, `false`} {
		t.Run(raw, func(t *testing.T) {
			var decoded any
			if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
				t.Fatal(err)
			}
			fields := map[string]json.RawMessage{"cls": json.RawMessage(raw)}
			for _, item := range []struct {
				name   string
				invoke func(*groups.API) error
			}{
				{"get_field", func(api *groups.API) error {
					_, err := api.GetRecord(context.Background(), "id", groups.WithRecordGetField("cls", decoded))
					return err
				}},
				{"get_opts", func(api *groups.API) error {
					_, err := api.GetRecord(context.Background(), "id", groups.WithRecordGetOpts(groups.RecordGetOpts{Fields: fields}))
					return err
				}},
				{"create_field", func(api *groups.API) error {
					_, err := api.CreateRecord(context.Background(), groups.WithRecordCreateField("cls", decoded))
					return err
				}},
				{"create_opts", func(api *groups.API) error {
					_, err := api.CreateRecord(context.Background(), groups.WithRecordCreateOpts(groups.RecordCreateOpts{Fields: fields}))
					return err
				}},
				{"update_field", func(api *groups.API) error {
					_, err := api.UpdateRecord(context.Background(), "id", groups.WithRecordUpdateField("cls", decoded))
					return err
				}},
				{"update_opts", func(api *groups.API) error {
					_, err := api.UpdateRecord(context.Background(), "id", groups.WithRecordUpdateOpts(groups.RecordUpdateOpts{Fields: fields}))
					return err
				}},
				{"list_query", func(api *groups.API) error {
					_, err := collectGroups(api.ListRecords(context.Background(), groups.WithRecordListQuery("cls", decoded)))
					return err
				}},
				{"list_opts", func(api *groups.API) error {
					_, err := collectGroups(api.ListRecords(context.Background(), groups.WithRecordListOpts(groups.RecordListOpts{Query: fields})))
					return err
				}},
			} {
				t.Run(item.name, func(t *testing.T) {
					api, calls := groupAPI(t, "", func(*http.Request) *http.Response {
						t.Fatal("bound cls input reached discovery or mutation HTTP")
						return nil
					})
					err := item.invoke(api)
					var proof *resource.ResponseError
					if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &proof) || len(*calls) != 0 {
						t.Fatal(err, proof, *calls)
					}
				})
			}
		})
	}
}

func TestPythonCinderGroupListAdvertisedOffsetTransitionsKeepScopeAndFallback(t *testing.T) {
	for _, initial := range []string{"absent", "marker", "offset"} {
		t.Run(initial, func(t *testing.T) {
			api, calls := groupAPI(t, "3.38", func(r *http.Request) *http.Response {
				query := r.URL.Query()
				if query.Has("marker") && query.Has("offset") {
					t.Fatal("mixed cursor families", query)
				}
				if r.URL.Path != groupBase+"/detail" || query.Get("sort") != "id:asc" || query.Get("all_tenants") != "False" || query.Get("limit") != "2" || query.Has("name") || r.Header.Get("OpenStack-API-Version") != "volume 3.38" {
					t.Fatal(r.URL, r.Header)
				}
				if query.Get("marker") == "b" {
					return groupWire(200, `{"groups":[]}`)
				}
				if query.Get("offset") == "2" {
					return groupWire(200, `{"groups":[{"id":"b","name":"other"}]}`)
				}
				return groupWire(200, `{"groups":[{"id":"a","name":"wanted"}],"next":"?offset=2&limit=2"}`)
			})
			options := []groups.RecordListOption{groups.WithRecordListLimit(2), groups.WithRecordListQuery("sort", "id:asc"), groups.WithRecordListQuery("all_projects", false), groups.WithRecordListQuery("name", "wanted")}
			switch initial {
			case "marker":
				options = append(options, groups.WithRecordListMarker("initial"))
			case "offset":
				options = append(options, groups.WithRecordListQuery("offset", 0))
			}
			values, err := collectGroups(api.ListRecords(context.Background(), options...))
			if err != nil || len(values) != 1 || len(*calls) != 3 {
				t.Fatal(values, err, *calls)
			}
			groupField(t, values[0], "id", `"a"`)
			first, _ := url.ParseQuery((*calls)[0].query)
			switch initial {
			case "absent":
				if first.Has("offset") || first.Has("marker") {
					t.Fatal(first)
				}
			case "marker":
				if first.Get("marker") != "initial" || first.Has("offset") {
					t.Fatal(first)
				}
			case "offset":
				if first.Get("offset") != "0" || first.Has("marker") {
					t.Fatal(first)
				}
			}
			second, _ := url.ParseQuery((*calls)[1].query)
			third, _ := url.ParseQuery((*calls)[2].query)
			if second.Get("offset") != "2" || second.Has("marker") || third.Get("marker") != "b" || third.Has("offset") {
				t.Fatal("advertised offset then raw last-row marker", second, third)
			}
		})
	}
}

func TestPythonCinderGroupListOffsetContinuationRejectsScopeAndCursorRegressions(t *testing.T) {
	for _, item := range []struct {
		name, next  string
		late, cycle bool
	}{
		{"changed_filter", "?offset=2&sort=id:desc", false, false},
		{"changed_project_collection", "/cinder/v3/other/groups/detail?offset=2", false, false},
		{"changed_origin", "https://wrong.example/cinder/v3/project/groups/detail?offset=2", false, false},
		{"mixed_cursor_families", "?offset=2&marker=last", false, false},
		{"repeat_offset", "?offset=2", true, true},
		{"backwards_offset", "?offset=1", true, false},
	} {
		t.Run(item.name, func(t *testing.T) {
			var actualBody string
			var actualStatus int
			api, calls := groupAPI(t, "3.38", func(r *http.Request) *http.Response {
				next, id := item.next, "first"
				actualStatus = 200
				if item.late && r.URL.Query().Get("offset") != "2" {
					next = "?offset=2"
				} else if item.late {
					id, actualStatus = "second", 202
				}
				actualBody = `{"groups":[{"id":"` + id + `"}],"next":"` + next + `","proof":"` + item.name + `"}`
				return groupWire(actualStatus, actualBody)
			})
			values, err := collectGroups(api.ListRecords(context.Background(), groups.WithRecordListQuery("sort", "id:asc"), groups.WithRecordListQuery("all_projects", false)))
			wantCalls := 1
			if item.late {
				wantCalls = 2
			}
			var proof *resource.ResponseError
			var cycle *resource.PaginationCycleError
			if len(*calls) != wantCalls || len(values) != wantCalls || !errors.As(err, &proof) || proof.StatusCode != actualStatus || string(proof.Body) != actualBody || proof.Header.Get("X-Request-Id") != "actual" || errors.As(err, &cycle) != item.cycle || !item.cycle && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(values, err, proof, *calls)
			}
			for _, call := range *calls {
				query, _ := url.ParseQuery(call.query)
				if call.path != groupBase+"/detail" || query.Get("sort") != "id:asc" || query.Get("all_tenants") != "False" || call.headers.Get("OpenStack-API-Version") != "volume 3.38" {
					t.Fatal("scope or request policy escaped before rejection", call)
				}
			}
		})
	}
}

func TestPythonCinderGroupListDeclaredNumericFiltersUseExactDecimalPolicy(t *testing.T) {
	for _, item := range []struct {
		name, actual, filter string
		match                bool
	}{
		{"binary64_rounding_not_applied", `9007199254740992`, `9007199254740993.0`, false},
		{"exact_integer_decimal_equality", `9007199254740993`, `9007199254740993.0`, true},
		{"binary64_underflow_not_applied", `1e-400`, `{}`, true},
	} {
		t.Run(item.name, func(t *testing.T) {
			body := `{"groups":[{"id":` + item.actual + `,"name":"wire"}]}`
			api, calls := groupAPI(t, "3.38", func(*http.Request) *http.Response { return groupWire(200, body) })
			values, err := collectGroups(api.ListRecords(context.Background(), groups.WithRecordListOpts(groups.RecordListOpts{Query: map[string]json.RawMessage{"id": json.RawMessage(item.filter)}})))
			if err != nil || (len(values) == 1) != item.match || len(*calls) != 1 || (*calls)[0].query != "" {
				t.Fatal(item, values, err, *calls)
			}
			if item.match {
				groupField(t, values[0], "id", item.actual)
				if string(values[0].Wire.Body["id"]) != item.actual || string(values[0].Envelope) != body || values[0].StatusCode != 200 || values[0].Header.Get("X-Request-Id") != "actual" {
					t.Fatal("numeric filtering altered wire proof", values[0])
				}
			}
		})
	}
}
