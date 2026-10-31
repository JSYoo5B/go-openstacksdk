package images_test

import (
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

	"github.com/JSYoo5B/go-openstacksdk/image/v2/images"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const imageTasksBase = "/reverse/glance/v2/images/parent/tasks"

type imageTasksCall struct{ method, path, query, body, version string }

func imageTasksAPI(t *testing.T, calls *[]imageTasksCall, reply func(*http.Request) *http.Response) (*images.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	client := nativeUpdateClient(cloud)
	client.Microversion = "2.16"
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, imageTasksCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("OpenStack-API-Version")})
		return reply(req), nil
	}))
	return images.New(client), cloud
}
func imageTasksWire(code int, body string) *http.Response {
	return nativeDeleteWire(code, io.NopCloser(strings.NewReader(body)))
}
func imageTaskValue(t *testing.T, v *images.ImageTaskRecord, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		t.Fatalf("%s want %s got %+v", key, want, v)
	}
}
func TestPythonImageTasksRecordsRawDescriptorsDefaultsAndProof(t *testing.T) {
	var calls []imageTasksCall
	var headers http.Header
	body := `{"tasks":[{"id":null,"image_id":"parent","input":[null,false,0],"result":false,"deleted":"raw","deleted_at":9007199254740993,"created_at":false,"request-id":"wire-only","user":"wire-only","unknown":{"zero":0},"self":"reserved","if_match":"body"}]}`
	api, _ := imageTasksAPI(t, &calls, func(req *http.Request) *http.Response {
		headers = req.Header.Clone()
		r := imageTasksWire(203, body)
		r.Header.Set("If-Match", "actual")
		return r
	})
	records, err := api.AllTaskRecords(context.Background(), "parent")
	if err != nil || len(records) != 1 || len(calls) != 1 || calls[0].path != imageTasksBase || calls[0].query != "" || calls[0].body != "" || headers.Get("Accept") != "application/json" || headers.Get("X-Auth-Token") != "test-token" || headers.Get("OpenStack-API-Version") != "image 2.16" {
		t.Fatal(records, err, calls, headers)
	}
	v := records[0]
	for key, want := range map[string]string{"id": "null", "image_id": `"parent"`, "input": `[null,false,0]`, "result": "false", "deleted": `"raw"`, "deleted_at": "9007199254740993", "created_at": "false", "type": "null", "status": "null", "owner": "null", "message": "null", "name": "null", "request_id": "null", "user_id": "null"} {
		imageTaskValue(t, v, key, want)
	}
	if v.ImageID != "parent" || v.Microversion != "2.16" || v.StatusCode != 203 || string(v.Envelope) != body || v.Wire == nil || v.Resource.Body["unknown"] != nil || v.Resource.Body["self"] != nil || v.Resource.Body["if_match"] != nil || v.Header.Get("If-Match") != "actual" || string(v.Wire.Body["self"]) != `"reserved"` {
		t.Fatal(v)
	}
	v.Resource.Body["input"][0] = 'X'
	v.Header.Set("If-Match", "changed")
	if string(v.Wire.Body["input"]) != `[null,false,0]` || v.Wire.Header.Get("If-Match") != "actual" || v.Resource.Header.Get("If-Match") != "actual" || string(v.Envelope) != body {
		t.Fatal(v)
	}
}
func TestPythonImageTasksRecordsLiteralParentAndSourceBodyFilter(t *testing.T) {
	var calls []imageTasksCall
	var escaped string
	api, _ := imageTasksAPI(t, &calls, func(req *http.Request) *http.Response {
		escaped = req.URL.EscapedPath()
		return imageTasksWire(200, `{"tasks":[{"id":"missing"},{"id":"null","image_id":null},{"id":"foreign","image_id":"other"},{"id":"keep","image_id":"folder/name%"}]}`)
	})
	rows, err := api.AllTaskRecords(context.Background(), "folder/name%")
	if err != nil || len(rows) != 1 || len(calls) != 1 || escaped != "/reverse/glance/v2/images/folder%2Fname%25/tasks" || rows[0].ImageID != "folder/name%" {
		t.Fatal(rows, err, calls, escaped)
	}
	imageTaskValue(t, rows[0], "image_id", `"folder/name%"`)
	imageTaskValue(t, rows[0], "id", `"keep"`)
	calls = nil
	_, err = api.AllTaskRecords(context.Background(), "image-name")
	if err != nil || len(calls) != 1 || calls[0].path != "/reverse/glance/v2/images/image-name/tasks" {
		t.Fatal(err, calls)
	}
}
func TestPythonImageTasksRecordsEnvelopeSingletonAndConstructorProof(t *testing.T) {
	for _, item := range []struct {
		body  string
		valid bool
		count int
	}{{`{"tasks":[]}`, true, 0}, {`{"tasks":{"image_id":"parent"}}`, true, 1}, {`{"tasks":null}`, false, 0}, {`{"tasks":false}`, false, 0}, {`{"tasks":[null]}`, false, 0}, {`{}`, false, 0}, {`[]`, false, 0}, {"", false, 0}, {"{bad", false, 0}, {`{"tasks":[{"image_id":"parent","cls":null}]}`, false, 0}, {`{"tasks":[{"image_id":"parent","connection":null}]}`, false, 0}, {`{"tasks":[{"image_id":"parent","microversion":null}]}`, false, 0}, {`{"tasks":[{"image_id":"parent","_synchronized":null}]}`, false, 0}, {"{\"tasks\":[{\"image_id\":\"parent\",\"name\":\"\xff\"}]}", false, 0}} {
		t.Run(fmt.Sprintf("%q", item.body), func(t *testing.T) {
			var calls []imageTasksCall
			api, _ := imageTasksAPI(t, &calls, func(*http.Request) *http.Response {
				r := imageTasksWire(203, item.body)
				r.Header.Set("X-Request-Id", "envelope")
				return r
			})
			rows, err := api.AllTaskRecords(context.Background(), "parent")
			if (err == nil) != item.valid || len(rows) != item.count || len(calls) != 1 {
				t.Fatal(item, rows, err, calls)
			}
			if item.valid {
				if rows == nil {
					t.Fatal("successful empty must be nonnil")
				}
			} else {
				var proof *resource.ResponseError
				if rows != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != item.body || proof.Header.Get("X-Request-Id") != "envelope" {
					t.Fatal(err, proof)
				}
			}
		})
	}
}
func TestPythonImageTasksRecordsLocationUsesCurrentScopeNotOwner(t *testing.T) {
	var calls []imageTasksCall
	cloudName, region, name := "cloud", "region", "current-name"
	location := resource.CloudLocation{Cloud: &cloudName, RegionName: &region, Zone: json.RawMessage(`"configured-zone"`), Project: resource.CloudProject{ID: json.RawMessage(`"current"`), Name: &name}}
	option := images.WithTaskRecordsLocation(location)
	location.Zone[1] = 'X'
	*location.Project.Name = "changed"
	api, _ := imageTasksAPI(t, &calls, func(*http.Request) *http.Response {
		return imageTasksWire(200, `{"tasks":[{"image_id":"parent","owner":"foreign","location":{"project":{"id":"forged"}}}]}`)
	})
	rows, err := api.AllTaskRecords(context.Background(), "parent", option)
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	var got resource.CloudLocation
	if json.Unmarshal(rows[0].Resource.Body["location"], &got) != nil || string(got.Project.ID) != `"current"` || got.Project.Name == nil || *got.Project.Name != "current-name" || string(got.Zone) != `"configured-zone"` || got.Cloud == nil || *got.Cloud != "cloud" {
		t.Fatal(got, rows)
	}
	imageTaskValue(t, rows[0], "owner", `"foreign"`)
	if string(rows[0].Wire.Body["location"]) != `{"project":{"id":"forged"}}` {
		t.Fatal(rows)
	}
}
func TestPythonImageTasksRecordsLocalFiltersUnknownAndQueryControls(t *testing.T) {
	var calls []imageTasksCall
	api, _ := imageTasksAPI(t, &calls, func(*http.Request) *http.Response {
		return imageTasksWire(200, `{"tasks":[{"id":"one","image_id":"parent","input":{"nested":{"enabled":1,"extra":false}},"deleted":1},{"id":"two","image_id":"parent","input":{},"deleted":false}]}`)
	})
	rows, err := api.AllTaskRecords(context.Background(), "parent", images.WithTaskRecordsQuery("input", map[string]any{"nested": map[string]any{"enabled": true}}), images.WithTaskRecordsQuery("deleted", true), images.WithTaskRecordsQuery("x_unknown", false), images.WithTaskRecordsPaginated(false))
	if err != nil || len(rows) != 1 || len(calls) != 1 || calls[0].query != "" {
		t.Fatal(rows, err, calls)
	}
	imageTaskValue(t, rows[0], "id", `"one"`)
	for _, item := range []struct {
		actual, filter string
		match, valid   bool
	}{{`"truthy"`, `{}`, true, true}, {`"truthy"`, `{"x":true}`, false, false}, {`1e-400`, `{}`, false, true}, {`1e400`, `{}`, true, true}, {`null`, `null`, true, true}, {`1`, `true`, true, true}} {
		calls = nil
		api, _ = imageTasksAPI(t, &calls, func(*http.Request) *http.Response {
			return imageTasksWire(200, `{"tasks":[{"image_id":"parent","result":`+item.actual+`}]}`)
		})
		rows, err = api.AllTaskRecords(context.Background(), "parent", images.WithTaskRecordsQuery("result", json.RawMessage(item.filter)))
		if (len(rows) == 1) != item.match || (err == nil) != item.valid {
			t.Fatal(item, rows, err)
		}
		if !item.valid {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != 200 {
				t.Fatal(err, proof)
			}
		}
	}
}
func TestPythonImageTasksRecordsMarkerCapsAndUnconsumedRows(t *testing.T) {
	var calls []imageTasksCall
	api, _ := imageTasksAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("marker") == "first" {
			return imageTasksWire(200, `{"tasks":[]}`)
		}
		return imageTasksWire(200, `{"tasks":[{"id":"first","image_id":"parent"}]}`)
	})
	rows, err := api.AllTaskRecords(context.Background(), "parent", images.WithTaskRecordsLimit(5))
	if err != nil || len(rows) != 1 || len(calls) != 2 || !strings.Contains(calls[1].query, "marker=first") {
		t.Fatal(rows, err, calls)
	}
	calls = nil
	rows, err = api.AllTaskRecords(context.Background(), "parent", images.WithTaskRecordsLimit(5), images.WithTaskRecordsPaginated(false))
	if err != nil || len(rows) != 1 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	calls = nil
	api, _ = imageTasksAPI(t, &calls, func(*http.Request) *http.Response {
		return imageTasksWire(200, `{"tasks":[{"id":"skip","image_id":"other"},{"id":"keep","image_id":"parent"},{"image_id":"parent","cls":null}],"next":"https://foreign.invalid/tasks"}`)
	})
	rows, err = api.AllTaskRecords(context.Background(), "parent", images.WithTaskRecordsLimit(0), images.WithTaskRecordsMaxItems(2))
	if err != nil || len(rows) != 1 || len(calls) != 1 || calls[0].query != "limit=2" {
		t.Fatal(rows, err, calls)
	}
	imageTaskValue(t, rows[0], "id", `"keep"`)
	calls = nil
	rows, err = api.AllTaskRecords(context.Background(), "parent", images.WithTaskRecordsMaxItems(1))
	if err != nil || rows == nil || len(rows) != 0 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	calls = nil
	for _, err := range api.TaskRecords(context.Background(), "parent") {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if len(calls) != 1 {
		t.Fatal(calls)
	}
}
func TestPythonImageTasksRecordsContinuationAndLateErrors(t *testing.T) {
	for _, lane := range []string{"plural", "next", "http", "dictionary", "versioned", "falsey", "empty", "foreign", "collection", "filter", "cycle", "late"} {
		t.Run(lane, func(t *testing.T) {
			var calls []imageTasksCall
			first := `{"tasks":[{"id":"first","image_id":"parent"}]}`
			api, _ := imageTasksAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Query().Get("marker") != "" {
					if lane == "late" {
						return imageTasksWire(500, `{"late":"actual"}`)
					}
					if lane == "cycle" {
						return imageTasksWire(200, `{"tasks":[{"id":"again","image_id":"parent"}],"next":"?marker=next"}`)
					}
					return imageTasksWire(200, `{"tasks":[{"id":"second","image_id":"parent"}]}`)
				}
				r := imageTasksWire(200, first)
				switch lane {
				case "plural":
					r = imageTasksWire(200, `{"tasks":[{"id":"first","image_id":"parent"}],"tasks_links":[{"rel":"next","href":"?marker=next"}]}`)
				case "next", "cycle", "late":
					r = imageTasksWire(200, `{"tasks":[{"id":"first","image_id":"parent"}],"next":"?marker=next"}`)
				case "http":
					r.Header.Set("Link", "<?marker=next>; rel=\"next\"")
				case "dictionary":
					r = imageTasksWire(200, `{"tasks":[{"id":"first","image_id":"parent"}],"links":{"next":"?marker=next"}}`)
				case "versioned":
					r = imageTasksWire(200, `{"tasks":[{"id":"first","image_id":"parent"}],"next":"/v2/images/parent/tasks?marker=next"}`)
				case "falsey":
					r = imageTasksWire(200, `{"tasks":[{"id":"first","image_id":"parent"}],"next":1e-400}`)
				case "empty":
					r = imageTasksWire(200, `{"tasks":[],"next":"https://bad.invalid/tasks"}`)
				case "foreign":
					r = imageTasksWire(200, `{"tasks":[{"id":"first","image_id":"parent"}],"next":"https://bad.invalid/tasks"}`)
				case "collection":
					r = imageTasksWire(200, `{"tasks":[{"id":"first","image_id":"parent"}],"next":"/v2/tasks?marker=next"}`)
				case "filter":
					r = imageTasksWire(200, `{"tasks":[{"id":"first","image_id":"parent"}],"next":"?marker=next&input=changed"}`)
				}
				return r
			})
			rows, err := api.AllTaskRecords(context.Background(), "parent")
			if lane == "foreign" || lane == "collection" || lane == "filter" || lane == "cycle" {
				var proof *resource.ResponseError
				if rows != nil || err == nil || !errors.As(err, &proof) || proof.StatusCode != 200 {
					t.Fatal(rows, err, proof, calls)
				}
			} else if lane == "late" {
				if rows != nil || !gophercloud.ResponseCodeIs(err, 500) || len(calls) != 2 {
					t.Fatal(rows, err, calls)
				}
			} else {
				want := 2
				if lane == "falsey" {
					want = 1
				}
				if lane == "empty" {
					want = 0
				}
				if err != nil || len(rows) != want {
					t.Fatal(rows, err, calls)
				}
			}
		})
	}
}
func TestPythonImageTasksRecordsLazySnapshotsReusableAndConcurrent(t *testing.T) {
	var calls []imageTasksCall
	var header http.Header
	api, cloud := imageTasksAPI(t, &calls, func(req *http.Request) *http.Response {
		header = req.Header.Clone()
		return imageTasksWire(200, `{"tasks":[{"image_id":"parent","name":"seed","input":{"x":false}}]}`)
	})
	query := map[string]json.RawMessage{"name": json.RawMessage(`"seed"`)}
	version := "2.9"
	option := images.WithTaskRecordsOpts(images.TaskRecordsOpts{Query: query, Headers: map[string]string{"X-Trace": "owned"}, Microversion: &version})
	query["name"][1] = 'X'
	version = "changed"
	var retained *images.TaskRecordsOpts
	applied := 0
	opts := []images.TaskRecordsOption{option, func(p *images.TaskRecordsOpts) error { applied++; retained = p; return nil }, func(*images.TaskRecordsOpts) error {
		retained.Headers["X-Auth-Token"] = "bad"
		retained.Query["name"] = json.RawMessage(`false`)
		*retained.Microversion = "changed"
		return nil
	}}
	iterator := api.TaskRecords(context.Background(), "parent", opts...)
	opts[0] = nil
	if applied != 0 || len(calls) != 0 {
		t.Fatal(applied, calls)
	}
	for _, err := range iterator {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, err := range iterator {
		if err != nil {
			t.Fatal(err)
		}
	}
	if applied != 2 || len(calls) != 2 || header.Get("X-Trace") != "owned" || header.Get("OpenStack-API-Version") != "image 2.9" {
		t.Fatal(applied, calls, header)
	}
	var attempts atomic.Int64
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		attempts.Add(1)
		return imageTasksWire(200, `{"tasks":[{"image_id":"parent","input":{"x":false}}]}`), nil
	}))
	shared := images.WithTaskRecordsHeaders(map[string]string{"X-Trace": "owned"})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rows, err := api.AllTaskRecords(context.Background(), "parent", shared)
			if err != nil || len(rows) != 1 {
				t.Error(rows, err)
				return
			}
			rows[0].Resource.Body["input"][0] = 'X'
			if string(rows[0].Wire.Body["input"]) != `{"x":false}` {
				t.Error(rows)
			}
		}()
	}
	wg.Wait()
	if attempts.Load() != 8 {
		t.Fatal(attempts.Load())
	}
}
func TestPythonImageTasksRecordsPreflightAndContextCauses(t *testing.T) {
	var calls []imageTasksCall
	api, _ := imageTasksAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("invalid options reached HTTP"); return nil })
	opts := [][]images.TaskRecordsOption{{nil}, {images.WithTaskRecordsLimit(-1)}, {images.WithTaskRecordsMaxItems(-1)}, {images.WithTaskRecordsQuery("limit", []int{1, 2})}, {images.WithTaskRecordsQuery("limit", 1.5)}, {images.WithTaskRecordsQuery("marker", "")}, {images.WithTaskRecordsQuery("status", json.RawMessage(`bad`))}, {images.WithTaskRecordsHeaders(map[string]string{"X-Auth-Token": "bad"})}, {images.WithTaskRecordsHeaders(map[string]string{"OpenStack-API-Version": "image 2.0"})}}
	for _, key := range []string{"image_id", "cls", "session", "self", "connection", "_synchronized", "base_path", "allow_unknown_params", "paginated", "microversion", "headers", "max_items", "location"} {
		opts = append(opts, []images.TaskRecordsOption{images.WithTaskRecordsQuery(key, nil)})
	}
	for _, options := range opts {
		rows, err := api.AllTaskRecords(context.Background(), "parent", options...)
		if rows != nil || err == nil || len(calls) != 0 {
			t.Fatal(rows, err, calls)
		}
	}
	for _, id := range []string{"", "..", "a\n", "\xff"} {
		_, err := api.AllTaskRecords(context.Background(), id)
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(id, err, calls)
		}
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause, callback := errors.New("cancel cause"), errors.New("callback")
	_, err := api.AllTaskRecords(ctx, "parent", func(*images.TaskRecordsOpts) error { cancel(cause); return callback })
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, callback) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	original := api.RawClient().ResourceBase
	_, err = api.AllTaskRecords(context.Background(), "parent", func(*images.TaskRecordsOpts) error { api.RawClient().ResourceBase += "changed/"; return callback })
	api.RawClient().ResourceBase = original
	if !errors.Is(err, callback) || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	api.RawClient().Type = "compute"
	_, err = api.AllTaskRecords(context.Background(), "parent")
	if !errors.Is(err, resource.ErrUnsupported) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
}
func TestPythonImageTasksRecordsSelectedVersionAndAcceptOwnership(t *testing.T) {
	for _, version := range []string{"", "2.9", "latest"} {
		var calls []imageTasksCall
		var header http.Header
		api, _ := imageTasksAPI(t, &calls, func(req *http.Request) *http.Response {
			header = req.Header.Clone()
			return imageTasksWire(200, `{"tasks":[{"image_id":"parent"}]}`)
		})
		api.RawClient().MoreHeaders = map[string]string{"Accept": "configured"}
		rows, err := api.AllTaskRecords(context.Background(), "parent", images.WithTaskRecordsMicroversion(version))
		wanted := ""
		if version != "" {
			wanted = "image " + version
		}
		if err != nil || len(rows) != 1 || rows[0].Microversion != version || len(calls) != 1 || header.Get("Accept") != "application/json" || header.Get("OpenStack-API-Version") != wanted || api.RawClient().Microversion != "2.16" {
			t.Fatal(rows, err, calls, header)
		}
		calls = nil
		rows, err = api.AllTaskRecords(context.Background(), "parent", images.WithTaskRecordsHeaders(map[string]string{"aCcEpT": "owned"}))
		if err != nil || len(rows) != 1 || header.Get("Accept") != "owned" || len(calls) != 1 {
			t.Fatal(rows, err, calls, header)
		}
	}
}
func TestPythonImageTasksRecordsDiscoveryCeilingAndAPIMinorIsNotMicroversion(t *testing.T) {
	for _, item := range []struct{ selected, max, min, want string }{{"", "2.99", "2.0", "2.17"}, {"", "2.9", "2.0", "2.9"}, {"", "", "", ""}, {"", "2.20", "2.18", ""}, {"2.99", "2.9", "2.0", "2.99"}, {"latest", "2.9", "2.0", "latest"}} {
		t.Run(item.selected+item.max+item.min, func(t *testing.T) {
			var calls []imageTasksCall
			api, _ := imageTasksAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Path == "/reverse/glance/v2/" {
					if req.Header.Get("OpenStack-API-Version") != "" || req.Header.Get("X-Trace") != "" {
						t.Error("operation headers leaked into discovery", req.Header)
					}
					return imageTasksWire(200, fmt.Sprintf(`{"versions":[{"id":"v2.17","status":"CURRENT","max_version":%q,"min_version":%q,"links":[{"rel":"self","href":"https://advertised.invalid/glance/v2/"}]}]}`, item.max, item.min))
				}
				return imageTasksWire(200, `{"tasks":[{"image_id":"parent","id":"task"}]}`)
			})
			api.RawClient().Endpoint = api.RawClient().ResourceBase
			api.RawClient().Microversion = item.selected
			rows, err := api.AllTaskRecords(context.Background(), "parent", images.WithTaskRecordsHeaders(map[string]string{"X-Trace": "owned"}))
			count, observations := 1, 0
			if item.selected == "" {
				count, observations = 2, 1
			}
			if err != nil || len(rows) != 1 || len(calls) != count || rows[0].Microversion != item.want || len(rows[0].Discovery) != observations || api.RawClient().Microversion != item.selected {
				t.Fatal(item, rows, err, calls)
			}
			if observations > 0 {
				if rows[0].Discovery[0].StatusCode != 200 || !strings.Contains(string(rows[0].Discovery[0].Envelope), "v2.17") {
					t.Fatal(rows)
				}
			}
		})
	}
}
func TestPythonImageTasksRecordsDiscoveryFailuresRetainActualProof(t *testing.T) {
	for _, code := range []int{200, 403} {
		var calls []imageTasksCall
		body := "{bad"
		if code == 403 {
			body = `{"discovery":"forbidden"}`
		}
		api, _ := imageTasksAPI(t, &calls, func(*http.Request) *http.Response { return imageTasksWire(code, body) })
		api.RawClient().Endpoint = api.RawClient().ResourceBase
		api.RawClient().Microversion = ""
		rows, err := api.AllTaskRecords(context.Background(), "parent")
		if rows != nil || err == nil || len(calls) != 1 {
			t.Fatal(rows, err, calls)
		}
		if code == 200 {
			var proof *resource.ResponseError
			if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != body {
				t.Fatal(err, proof)
			}
		} else if !gophercloud.ResponseCodeIs(err, 403) {
			t.Fatal(err)
		}
	}
}
func TestPythonImageTasksRecordsMissingErrorsAndBoundedStatuses(t *testing.T) {
	for _, code := range []int{199, 200, 203, 204, 299, 300, 399, 400, 403, 404, 409, 503} {
		var calls []imageTasksCall
		body := `{"tasks":[{"image_id":"parent"}]}`
		api, _ := imageTasksAPI(t, &calls, func(*http.Request) *http.Response { return imageTasksWire(code, body) })
		rows, err := api.AllTaskRecords(context.Background(), "parent")
		if code >= 200 && code < 400 {
			if err != nil || len(rows) != 1 || rows[0].StatusCode != code {
				t.Fatal(code, rows, err, calls)
			}
		} else {
			var native gophercloud.ErrUnexpectedResponseCode
			if rows != nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != body {
				t.Fatal(code, rows, err, native)
			}
		}
		if len(calls) != 1 {
			t.Fatal(calls)
		}
	}
}
func TestPythonImageTasksRecordsNativeRetryAndReauthentication(t *testing.T) {
	for _, lane := range []string{"clean", "reauth", "body", "query", "codes", "custom"} {
		t.Run(lane, func(t *testing.T) {
			var calls []imageTasksCall
			var tokens []string
			api, cloud := imageTasksAPI(t, &calls, func(req *http.Request) *http.Response {
				tokens = append(tokens, req.Header.Get("X-Auth-Token"))
				if len(calls) == 1 {
					if lane == "reauth" {
						return imageTasksWire(401, `{"expired":true}`)
					}
					return imageTasksWire(503, `{"rejection":"actual"}`)
				}
				if lane == "codes" {
					return imageTasksWire(400, `{"outside":"original"}`)
				}
				return imageTasksWire(200, `{"tasks":[{"image_id":"parent"}]}`)
			})
			cause := errors.New("callback")
			if lane == "reauth" {
				cloud.Provider.ReauthFunc = func(context.Context) error { cloud.Provider.SetToken("renewed"); return nil }
			} else {
				api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
					switch lane {
					case "body":
						opts.JSONBody = map[string]any{"new": true}
					case "query":
						opts.MoreHeaders["OpenStack-API-Version"] = "image changed"
					case "codes":
						opts.OkCodes = append(opts.OkCodes, 400)
					case "custom":
						return cause
					}
					return nil
				}
			}
			rows, err := api.AllTaskRecords(context.Background(), "parent")
			if lane == "clean" || lane == "reauth" {
				if err != nil || len(rows) != 1 || len(calls) != 2 || calls[0] != calls[1] {
					t.Fatal(rows, err, calls)
				}
				if lane == "reauth" && !reflect.DeepEqual(tokens, []string{"test-token", "renewed"}) {
					t.Fatal(tokens)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				wanted, count := 503, 1
				if lane == "codes" {
					wanted, count = 400, 2
				}
				if rows != nil || !errors.As(err, &native) || native.Actual != wanted || len(calls) != count {
					t.Fatal(rows, err, native, calls)
				}
				if lane == "custom" && !errors.Is(err, cause) || lane != "custom" && lane != "codes" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			}
		})
	}
}
func TestPythonImageTasksRecordsRetryForbiddenHeadersAndOmittedVersion(t *testing.T) {
	for _, key := range []string{"aUtHoRiZaTiOn", "x-Service-Token", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "Te", "Upgrade", "X-OpenStack-Nova-API-Version", "X-OpenStack-Volume-API-Version", "X-OpenStack-Manila-API-Version", "X-OpenStack-Ironic-API-Version", "omit-version"} {
		for _, custom := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", key, custom), func(t *testing.T) {
				var calls []imageTasksCall
				body := `{"rejection":"fixed"}`
				api, _ := imageTasksAPI(t, &calls, func(*http.Request) *http.Response {
					r := imageTasksWire(503, body)
					r.Header.Set("X-Request-Id", "retry-fixed")
					return r
				})
				cause := errors.New("callback")
				api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
					if key == "omit-version" {
						opts.OmitHeaders = []string{"oPeNsTaCk-ApI-vErSiOn"}
					} else {
						opts.MoreHeaders[key] = "forbidden"
					}
					if custom {
						return cause
					}
					return nil
				}
				rows, err := api.AllTaskRecords(context.Background(), "parent")
				var native gophercloud.ErrUnexpectedResponseCode
				if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.Is(err, cause) != custom || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "retry-fixed" || len(calls) != 1 {
					t.Fatal(rows, err, native, calls)
				}
			})
		}
	}
}

// The existing nativeDeleteBody carries the physical error and close count;
// this thin wrapper only schedules this operation's cancel/Source hook.
type imageTasksHookBody struct {
	io.ReadCloser
	onRead, onClose func()
}

func (b *imageTasksHookBody) Read(p []byte) (int, error) {
	if b.onRead != nil {
		f := b.onRead
		b.onRead = nil
		f()
	}
	return b.ReadCloser.Read(p)
}
func (b *imageTasksHookBody) Close() error {
	if b.onClose != nil {
		b.onClose()
	}
	return b.ReadCloser.Close()
}
func TestPythonImageTasksRecordsPhysicalProofAndStickySource(t *testing.T) {
	for _, code := range []int{200, 400, 403, 404, 409, 503} {
		for _, lane := range []string{"read", "close", "read+close", "cancel", "source"} {
			t.Run(fmt.Sprintf("%d/%s", code, lane), func(t *testing.T) {
				var calls []imageTasksCall
				body := `{"tasks":[{"image_id":"parent"}],"next":1e-400}`
				readCause, closeCause, cancelCause := errors.New("read"), errors.New("close"), errors.New("cancel")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				physical := &testcloud.Body{Data: body}
				switch lane {
				case "read":
					physical.ReadErr = readCause
				case "close":
					physical.CloseErr = closeCause
				case "read+close":
					physical.ReadErr = readCause
					physical.CloseErr = closeCause
				}
				hook := &imageTasksHookBody{ReadCloser: physical}
				api, _ := imageTasksAPI(t, &calls, func(*http.Request) *http.Response {
					r := nativeDeleteWire(code, hook)
					r.Header.Set("X-Request-Id", "physical")
					return r
				})
				original := api.RawClient().ResourceBase
				if lane == "cancel" {
					hook.onRead = func() { cancel(cancelCause) }
				}
				if lane == "source" {
					hook.onRead = func() { api.RawClient().ResourceBase += "changed/" }
					hook.onClose = func() { api.RawClient().ResourceBase = original }
				}
				rows, err := api.AllTaskRecords(ctx, "parent")
				api.RawClient().ResourceBase = original
				if rows != nil || err == nil || len(calls) != 1 || physical.Closes() != 1 {
					t.Fatal(rows, err, calls, physical.Closes())
				}
				if code < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "physical" {
						t.Fatal(err, proof)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != code || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "physical" {
						t.Fatal(err, native)
					}
				}
				if (lane == "read" || lane == "read+close") && !errors.Is(err, readCause) || (lane == "close" || lane == "read+close") && !errors.Is(err, closeCause) || lane == "cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) || lane == "source" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			})
		}
	}
}

// Conditional headers are safe caller choices; a native retry cannot add an
// unselected condition, replace a selected value, or omit it before replay.
func TestPythonImageTasksRecordsConditionalHeadersAreOwned(t *testing.T) {
	for _, key := range []string{"If-Match", "If-None-Match", "If-Modified-Since", "If-Unmodified-Since"} {
		for _, lane := range []string{"absent", "absent-custom", "captured", "explicit", "changed", "omitted"} {
			t.Run(key+"/"+lane, func(t *testing.T) {
				var calls []imageTasksCall
				var observed []http.Header
				body := `{"rejection":"conditional-original"}`
				value := `"selected-etag"`
				if strings.HasSuffix(key, "-Since") {
					value = "Wed, 21 Oct 2015 07:28:00 GMT"
				}
				api, _ := imageTasksAPI(t, &calls, func(req *http.Request) *http.Response {
					observed = append(observed, req.Header.Clone())
					if len(calls) == 1 {
						r := imageTasksWire(503, body)
						r.Header.Set("X-Request-Id", "conditional-proof")
						return r
					}
					return imageTasksWire(200, `{"tasks":[{"image_id":"parent"}]}`)
				})
				var options []images.TaskRecordsOption
				if lane == "captured" {
					api.RawClient().MoreHeaders = map[string]string{strings.ToLower(key): value}
				}
				if lane == "explicit" || lane == "changed" || lane == "omitted" {
					options = append(options, images.WithTaskRecordsHeaders(map[string]string{strings.ToLower(key): value}))
				}
				cause := errors.New("conditional callback")
				api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
					switch lane {
					case "absent", "absent-custom":
						opts.MoreHeaders[strings.ToLower(key)] = "injected"
					case "changed":
						opts.MoreHeaders[key] = "changed"
					case "omitted":
						opts.OmitHeaders = []string{strings.ToLower(key)}
					}
					if lane == "absent-custom" {
						return cause
					}
					return nil
				}
				rows, err := api.AllTaskRecords(context.Background(), "parent", options...)
				if lane == "captured" || lane == "explicit" {
					if err != nil || len(rows) != 1 || len(calls) != 2 || len(observed) != 2 || observed[0].Get(key) != value || observed[1].Get(key) != value {
						t.Fatal(rows, err, calls, observed)
					}
					return
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.Is(err, cause) != (lane == "absent-custom") || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "conditional-proof" || len(calls) != 1 {
					t.Fatal(rows, err, native, calls)
				}
				want := ""
				if lane == "changed" || lane == "omitted" {
					want = value
				}
				if len(observed) != 1 || observed[0].Get(key) != want {
					t.Fatal(observed, want)
				}
			})
		}
	}
}
