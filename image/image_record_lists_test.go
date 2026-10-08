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

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestImageRecordListProjectionPackingAndImportMethods(t *testing.T) {
	for _, test := range []struct {
		name, row, properties string
		methods               []string
	}{
		{"empty constructor", `{}`, "null", []string{}},
		{"null properties stays null", `{"properties":null}`, "null", []string{}},
		{"false properties stays false", `{"properties":false}`, "false", []string{}},
		{"unknown packs null properties", `{"properties":null,"vendor":1e400}`, `{"properties":null,"vendor":1e400}`, []string{}},
		{"only wire location becomes property", `{"location":{"cloud":"wire"}}`, `{"location":{"cloud":"wire"}}`, []string{}},
		{"self removed before constructor packing", `{"self":"https://foreign.test/"}`, "null", []string{}},
		{"plain header attr", `{"OpenStack-image-import-methods":"a,, a,a"}`, "null", []string{"a", "", " a", "a"}},
		{"empty header attr", `{"OpenStack-image-import-methods":""}`, "null", []string{}},
		{"falsey header attr", `{"OpenStack-image-import-methods":false}`, "null", []string{}},
		{"case decoy header attr", `{"openstack-image-import-methods":"a"}`, `{"openstack-image-import-methods":"a"}`, []string{}},
		{"plain field name unknown", `{"image_import_methods":["a"]}`, `{"image_import_methods":["a"]}`, []string{}},
		{"arbitrary name no alternate id", `{"name":false}`, "null", []string{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			raw := `{"images":[` + test.row + `]}`
			calls, locations := 0, 0
			cloud := "current"
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet || req.URL.EscapedPath() != "/reverse/glance/v2/images" || req.URL.RawQuery != "" || req.Header.Get("Accept") != "application/json" || req.Body != nil {
					t.Fatal(req.Method, req.URL, req.Header)
				}
				response := taskCoreJSON(req, 203, raw)
				response.Header.Set("OpenStack-image-import-methods", "page header ignored")
				return response, nil
			})
			service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return resource.CloudLocation{Cloud: &cloud}, nil }})
			rows, err := service.AllImageRecords(context.Background())
			if err != nil || len(rows) != 1 || calls != 1 || locations != 1 || len(rows[0].Resource.Body) != 65 || rows[0].ImportMethods == nil {
				t.Fatal(rows, err, calls, locations)
			}
			row := rows[0]
			want := imageRecordDefaults(test.properties)
			want["location"] = string(row.Resource.Body["location"])
			if test.name == "arbitrary name no alternate id" {
				want["name"] = "false"
			}
			th.CheckDeepEquals(t, want, imageRecordValues(row.Resource))
			th.CheckDeepEquals(t, test.methods, row.ImportMethods)
			var location resource.CloudLocation
			if err := json.Unmarshal(row.Resource.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != "current" {
				t.Fatal(location, err)
			}
			if row.StatusCode != 203 || row.Resource.StatusCode != 203 || row.Wire.StatusCode != 203 || row.Header.Get("X-Task-Proof") != "actual" || row.Resource.Header.Get("X-Task-Proof") != "actual" || row.Wire.Header.Get("X-Task-Proof") != "actual" || string(row.Envelope) != raw {
				t.Fatal("owned page receipt", row)
			}
			var expectedWire map[string]json.RawMessage
			if err := json.Unmarshal([]byte(test.row), &expectedWire); err != nil {
				t.Fatal(err)
			}
			th.CheckDeepEquals(t, expectedWire, row.Wire.Body)
		})
	}
	t.Run("row channels do not alias neighbors", func(t *testing.T) {
		const raw = `{"images":[{"metadata":{"n":1.00000000000000000001},"vendor":[null,false]},{"metadata":{"n":1.00000000000000000001},"vendor":[null,false]}]}`
		var header http.Header
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			response := taskCoreJSON(req, 200, raw)
			header = response.Header
			return response, nil
		})
		rows, err := New(client).AllImageRecords(context.Background())
		if err != nil || len(rows) != 2 {
			t.Fatal(rows, err)
		}
		rows[0].Resource.Body["metadata"][0] = '!'
		rows[0].Resource.Header.Set("X-Task-Proof", "view changed")
		rows[0].Wire.Header.Set("X-Task-Proof", "wire changed")
		rows[0].Header.Set("X-Task-Proof", "record changed")
		rows[0].Envelope[0] = '!'
		if string(rows[0].Wire.Body["metadata"]) != `{"n":1.00000000000000000001}` || string(rows[1].Resource.Body["metadata"]) != `{"n":1.00000000000000000001}` || rows[1].Header.Get("X-Task-Proof") != "actual" || string(rows[1].Envelope) != raw || header.Get("X-Task-Proof") != "actual" {
			t.Fatal("row ownership", rows)
		}
	})
}

func TestImageRecordListSourceQueryAndLocalFilterClassification(t *testing.T) {
	calls := 0
	want := url.Values{"id": {"literal id"}, "name": {"literal name"}, "visibility": {"shared"}, "member_status": {"all"}, "owner": {"wire owner"}, "status": {"active"}, "size_min": {"0"}, "size_max": {"900719925474099312345"}, "protected": {"False"}, "os_hidden": {"False"}, "sort_key": {"name", "id"}, "sort_dir": {"asc", "desc"}, "sort": {"name:asc"}, "tag": {"a", "False", ""}, "created_at": {"gte:literal date"}, "updated_at": {"lt:literal date"}, "limit": {"0"}, "marker": {"start"}}
	filters := map[string]any{
		"id": "literal id", "name": "literal name", "visibility": "shared", "member_status": "all", "owner": "wire owner", "status": "active", "size_min": 0, "size_max": json.RawMessage(`900719925474099312345`), "protected": false, "is_hidden": false, "os_hidden": true,
		"sort_key": []any{"name", "id"}, "sort_dir": []string{"asc", "desc"}, "sort": "name:asc", "tag": []any{"a", nil, false, ""}, "created_at": "gte:literal date", "updated_at": "lt:literal date",
		"checksum": "selected", "is_protected": true, "owner_id": "row owner", "properties": map[string]any{"subset": map[string]any{"chosen": true}},
		// Neither these remote-only body aliases nor arbitrary fields are filters;
		// unknown-before-marshal permits values outside the JSON domain.
		"os_hash_algo": make(chan int), "hw_boot_menu": make(chan int), "unknown": make(chan int),
	}
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if !reflect.DeepEqual(req.URL.Query(), want) {
			t.Fatal("Source query transpose", req.URL.Query(), want)
		}
		return taskCoreJSON(req, 200, `{"images":[{"id":"first","checksum":"other","protected":true,"owner":"row owner","properties":{"subset":{"chosen":true}}},{"id":"kept","checksum":"selected","protected":"false","owner":"row owner","properties":{"subset":{"chosen":true,"extra":null},"passive":1}}]}`), nil
	})
	rows, err := New(client).AllImageRecords(context.Background(), WithImageRecordListFilters(filters), WithImageRecordListLimit(0), WithImageRecordListMarker("start"))
	if err != nil || len(rows) != 1 || calls != 1 || string(rows[0].Resource.Body["id"]) != `"kept"` {
		t.Fatal(rows, err, calls)
	}
	t.Run("invalid raw unknown remains passive", func(t *testing.T) {
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			if req.URL.RawQuery != "" {
				t.Fatal(req.URL)
			}
			return taskCoreJSON(req, 200, `{"images":[{}]}`), nil
		})
		rows, err := New(client).AllImageRecords(context.Background(), WithImageRecordListOpts(ImageRecordListOpts{Filters: map[string]json.RawMessage{"unknown": json.RawMessage(`{`), "os_hash_algo": json.RawMessage(`{`)}}))
		if err != nil || len(rows) != 1 {
			t.Fatal(rows, err)
		}
	})
	t.Run("remote bool spelling and nested doseq", func(t *testing.T) {
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			want := url.Values{"os_hidden": {"True"}, "tag": {"1", "2", "None", "False"}}
			if !reflect.DeepEqual(req.URL.Query(), want) {
				t.Fatal(req.URL.Query(), want)
			}
			return taskCoreJSON(req, 200, `{"images":[]}`), nil
		})
		_, err := New(client).AllImageRecords(context.Background(), WithImageRecordListFilter("os_hidden", true), WithImageRecordListFilter("tag", []any{[]any{1, 2}, nil, []any{nil, false}}))
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestImageRecordListLazyRepeatedOptionSnapshots(t *testing.T) {
	calls, callbacks, locations := 0, 0, 0
	cloud := "first"
	headers := map[string]string{"X-Option": "factory snapshot"}
	filters := map[string]json.RawMessage{"checksum": json.RawMessage(`"selected"`)}
	limit := 0
	paginated := false
	option := WithImageRecordListOpts(ImageRecordListOpts{Headers: headers, Filters: filters, Limit: &limit, Marker: "start", Paginated: &paginated})
	var retained *ImageRecordListOpts
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		wantSource := "first"
		if calls == 2 {
			wantSource = "second"
		}
		if req.Header.Get("X-Source") != wantSource || req.Header.Get("X-Option") != "factory snapshot" || req.URL.Query().Get("limit") != "0" || req.URL.Query().Get("marker") != "start" || len(req.URL.Query()) != 2 {
			t.Fatal(req.URL, req.Header)
		}
		retained.Headers["X-Option"] = "retained mutation"
		retained.Filters["checksum"][1] = 'X'
		*retained.Limit = 2
		*retained.Paginated = true
		return taskCoreJSON(req, 200, `{"images":[{"checksum":"selected"}],"next":"https://foreign.test/unused"}`), nil
	})
	client.MoreHeaders = map[string]string{"X-Source": "first"}
	service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
		locations++
		client.MoreHeaders["X-Source"] = "getter mutation"
		return resource.CloudLocation{Cloud: &cloud}, nil
	}})
	options := []ImageRecordListOption{option, func(value *ImageRecordListOpts) error {
		callbacks++
		retained = value
		cloud = "callback mutation"
		client.MoreHeaders["X-Source"] = "callback mutation"
		return nil
	}}
	stream := service.ListImageRecords(context.Background(), options...)
	headers["X-Option"] = "caller mutation"
	filters["checksum"][1] = 'X'
	limit = 3
	paginated = true
	options[0] = nil
	if calls != 0 || callbacks != 0 || locations != 0 {
		t.Fatal("eager stream", calls, callbacks, locations)
	}
	for iteration := 0; iteration < 2; iteration++ {
		wantCloud := "first"
		if iteration == 1 {
			client.MoreHeaders["X-Source"] = "second"
			cloud = "second"
			wantCloud = "second"
		}
		count := 0
		for row, err := range stream {
			if err != nil {
				t.Fatal(err)
			}
			count++
			var location resource.CloudLocation
			if err := json.Unmarshal(row.Resource.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != wantCloud {
				t.Fatal(location, err)
			}
		}
		if count != 1 || calls != iteration+1 || callbacks != iteration+1 || locations != iteration+1 {
			t.Fatal(count, calls, callbacks, locations)
		}
	}
}

func TestImageRecordListPagingRawCapsAndPartials(t *testing.T) {
	for _, mode := range []string{"next", "plural links", "links", "HTTP Link", "marker fallback", "server limit only", "single page", "raw cap filter", "early break", "zero limit empty stop"} {
		t.Run(mode, func(t *testing.T) {
			calls, count := 0, 0
			next := "/v2/images?marker=second"
			extra := `,"next":"` + next + `"`
			header := http.Header{"X-Task-Proof": {"actual"}}
			options := []ImageRecordListOption{}
			switch mode {
			case "plural links":
				extra = `,"images_links":[{"rel":"next","href":"` + next + `"}]`
			case "links":
				extra = `,"links":[{"rel":"next","href":"` + next + `"}]`
			case "HTTP Link":
				extra = ""
				header.Set("Link", "<"+next+">; rel=\"next\"")
			case "marker fallback":
				extra = ""
				options = append(options, WithImageRecordListLimit(3))
			case "server limit only":
				extra = `,"limit":2`
			case "single page":
				options = append(options, WithImageRecordListPaginated(false))
			case "raw cap filter":
				options = append(options, WithImageRecordListMaxItems(2), WithImageRecordListFilter("checksum", "selected"))
			case "zero limit empty stop":
				options = append(options, WithImageRecordListLimit(0))
			}
			first := `{"images":[{"id":"first","checksum":"other"},{"id":"last","checksum":"selected"}]` + extra + `}`
			if mode == "zero limit empty stop" {
				first = `{"images":[],"next":"https://foreign.test/unused"}`
			}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					if mode == "marker fallback" {
						th.AssertEquals(t, "3", req.URL.Query().Get("limit"))
					} else if mode == "raw cap filter" {
						th.AssertEquals(t, "2", req.URL.Query().Get("limit"))
					} else if mode == "zero limit empty stop" {
						th.AssertEquals(t, "0", req.URL.Query().Get("limit"))
					} else if req.URL.RawQuery != "" {
						t.Fatal(req.URL)
					}
					reply := taskCoreJSON(req, 200, first)
					reply.Header = header
					return reply, nil
				}
				if calls != 2 {
					t.Fatal("unexpected third page", req.URL)
				}
				if mode == "marker fallback" {
					if req.URL.Query().Get("marker") != "last" || req.URL.Query().Get("limit") != "3" {
						t.Fatal("marker must use captured wire id", req.URL)
					}
				} else if req.URL.Query().Get("marker") != "second" {
					t.Fatal(req.URL)
				}
				return taskCoreJSON(req, 200, `{"images":[]}`), nil
			})
			for row, err := range New(client).ListImageRecords(context.Background(), options...) {
				if err != nil {
					t.Fatal(err)
				}
				count++
				if mode == "marker fallback" {
					row.Resource.Body["id"] = json.RawMessage(`"view mutation"`)
					row.Wire.Body["id"] = json.RawMessage(`"wire mutation"`)
				}
				if mode == "early break" {
					break
				}
			}
			wantCalls, wantCount := 2, 2
			switch mode {
			case "server limit only", "single page":
				wantCalls = 1
			case "raw cap filter", "early break":
				wantCalls = 1
				wantCount = 1
			case "zero limit empty stop":
				wantCalls = 1
				wantCount = 0
			}
			if calls != wantCalls || count != wantCount {
				t.Fatal(calls, count, wantCalls, wantCount)
			}
		})
	}
	for _, mode := range []string{"later malformed row", "later native error", "ignored pagination cycle", "missing wire marker"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			options := []ImageRecordListOption{}
			first := `{"images":[{"id":"kept"}],"next":"?marker=second"}`
			if mode == "ignored pagination cycle" || mode == "missing wire marker" {
				options = append(options, WithImageRecordListLimit(1))
				first = `{"images":[{"id":"kept"}]}`
			}
			if mode == "missing wire marker" {
				first = `{"images":[{"name":"name cannot be marker"}]}`
			}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 || mode == "ignored pagination cycle" {
					return taskCoreJSON(req, 200, first), nil
				}
				if mode == "later native error" {
					return taskCoreJSON(req, 500, "native second page"), nil
				}
				return taskCoreJSON(req, 201, `{"images":[null]}`), nil
			})
			rows, err := New(client).AllImageRecords(context.Background(), options...)
			wantRows, wantCalls := 1, 2
			if mode == "ignored pagination cycle" {
				wantRows = 2
			}
			if mode == "missing wire marker" {
				wantCalls = 1
			}
			if err == nil || len(rows) != wantRows || calls != wantCalls {
				t.Fatal(rows, err, calls)
			}
			switch mode {
			case "later malformed row":
				taskCoreProof(t, err, 201, `{"images":[null]}`)
			case "later native error":
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 500 {
					t.Fatal(err)
				}
			case "ignored pagination cycle":
				if !errors.Is(err, resource.ErrPaginationCycle) {
					t.Fatal(err)
				}
			case "missing wire marker":
				taskCoreProof(t, err, 200, first)
			}
		})
	}
	t.Run("cap skips unused bad row and unsafe link", func(t *testing.T) {
		calls := 0
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
			calls++
			return taskCoreJSON(req, 200, `{"images":[{"id":"kept"},{"hw_vif_multiqueue_enabled":1}],"next":"https://foreign.test/unused"}`), nil
		})
		rows, err := New(client).AllImageRecords(context.Background(), WithImageRecordListMaxItems(1))
		if err != nil || len(rows) != 1 || calls != 1 {
			t.Fatal(rows, err, calls)
		}
	})
}

func TestImageRecordListEnvelopeConstructorAndStatusEvidence(t *testing.T) {
	for _, test := range []struct {
		code  int
		raw   string
		good  bool
		count int
	}{
		{200, `{"images":[]}`, true, 0}, {201, `{"images":{}}`, true, 1}, {299, `{"images":[{}]}`, true, 1}, {300, `{"images":[{}]}`, true, 1}, {399, `{"images":[]}`, true, 0},
		{204, "", false, 0}, {200, "not JSON", false, 0}, {200, `{}`, false, 0}, {200, `null`, false, 0}, {200, `[]`, false, 0}, {200, `{"images":null}`, false, 0}, {200, `{"images":false}`, false, 0},
		{200, `{"images":[null]}`, false, 0}, {200, `{"images":[1]}`, false, 0}, {200, `{"images":[{"connection":null}]}`, false, 0}, {200, `{"images":[{"microversion":"2"}]}`, false, 0}, {200, `{"images":[{"_synchronized":true}]}`, false, 0},
		{200, `{"images":[{"OpenStack-image-import-methods":["truthy"]}]}`, false, 0}, {200, `{"images":[{"hw_vif_multiqueue_enabled":1}]}`, false, 0}, {200, `{"images":[{"instance_type_rxtx_factor":"not float"}]}`, false, 0}, {200, "{\"images\":[{\"unknown\":\"\xff\"}]}", false, 0},
	} {
		t.Run(fmt.Sprintf("%d/%q", test.code, test.raw), func(t *testing.T) {
			calls, retries := 0, 0
			body := &taskCoreBody{reader: strings.NewReader(test.raw)}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return taskCoreHTTP(req, test.code, body), nil
			})
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			rows, err := New(client).AllImageRecords(context.Background())
			if test.good {
				if err != nil || rows == nil || len(rows) != test.count {
					t.Fatal(rows, err)
				}
				for _, row := range rows {
					if row.StatusCode != test.code {
						t.Fatal(row)
					}
				}
			} else {
				if err == nil || len(rows) != 0 {
					t.Fatal(rows, err)
				}
				taskCoreProof(t, err, test.code, test.raw)
			}
			if calls != 1 || retries != 0 || body.closes != 1 {
				t.Fatal("accepted body replay", calls, retries, body.closes)
			}
		})
	}
}

func TestImageRecordListPhysicalIOAndPostYieldGuards(t *testing.T) {
	const raw = `{"images":[{"id":"first"},{"id":"second"}],"next":"?marker=more"}`
	for _, mode := range []string{"read", "close", "cancel", "source drift after yield", "cancel after yield"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New("image list body failure")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			calls, retries, count := 0, 0, 0
			body := &taskCoreBody{reader: strings.NewReader(raw)}
			switch mode {
			case "read":
				body.reader = &taskCoreReader{body: raw, err: cause}
			case "close":
				body.closeErr = cause
			case "cancel":
				body.reader = &taskCoreReader{body: raw, err: io.EOF, action: func() { cancel(cause) }}
			}
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, 200, body), nil })
			client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
				retries++
				return err
			}
			var failure error
			for _, err := range New(client).ListImageRecords(ctx) {
				if err != nil {
					failure = err
					break
				}
				count++
				if mode == "source drift after yield" {
					client.Endpoint = "https://foreign.test/"
				}
				if mode == "cancel after yield" {
					cancel(cause)
				}
			}
			wantCount := 0
			if strings.Contains(mode, "after yield") {
				wantCount = 1
			}
			if failure == nil || count != wantCount || calls != 1 || retries != 0 || body.closes != 1 {
				t.Fatal(failure, count, calls, retries, body.closes)
			}
			if mode == "source drift after yield" {
				if !errors.Is(failure, resource.ErrInvalidOption) {
					t.Fatal(failure)
				}
			} else if !errors.Is(failure, cause) {
				t.Fatal(failure)
			}
			taskCoreProof(t, failure, 200, raw)
		})
	}
}

func TestImageRecordListCompletePreflightAndLegacyBoundary(t *testing.T) {
	for _, test := range []struct {
		name    string
		options []ImageRecordListOption
	}{
		{"nil option", []ImageRecordListOption{nil}},
		{"negative limit", []ImageRecordListOption{WithImageRecordListLimit(-1)}}, {"negative cap", []ImageRecordListOption{WithImageRecordListMaxItems(-1)}}, {"control marker", []ImageRecordListOption{WithImageRecordListMarker("bad\n")}},
		{"owned token", []ImageRecordListOption{WithImageRecordListHeader("X-Auth-Token", "foreign")}}, {"header newline", []ImageRecordListOption{WithImageRecordListHeaders(map[string]string{"X-Extra": "\n"})}},
		{"broken known server filter", []ImageRecordListOption{WithImageRecordListOpts(ImageRecordListOpts{Filters: map[string]json.RawMessage{"status": json.RawMessage(`{`)}})}},
		{"broken known local filter", []ImageRecordListOption{WithImageRecordListOpts(ImageRecordListOpts{Filters: map[string]json.RawMessage{"checksum": json.RawMessage(`{`)}})}},
		{"known nonJSON filter", []ImageRecordListOption{WithImageRecordListFilter("metadata", make(chan int))}},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				return taskCoreJSON(req, 200, `{"images":[]}`), nil
			})
			rows, err := New(client).AllImageRecords(context.Background(), test.options...)
			if len(rows) != 0 || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(rows, err, calls)
			}
		})
	}
	for _, ctx := range []context.Context{nil, func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }()} {
		calls, callbacks := 0, 0
		client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil })
		rows, err := New(client).AllImageRecords(ctx, func(*ImageRecordListOpts) error { callbacks++; return nil })
		if len(rows) != 0 || err == nil || calls != 0 || callbacks != 0 {
			t.Fatal(rows, err, calls, callbacks)
		}
	}
	var nilService *Service
	if rows, err := nilService.AllImageRecords(context.Background()); len(rows) != 0 || err == nil {
		t.Fatal(rows, err)
	}
	t.Run("legacy array and status contracts remain strict", func(t *testing.T) {
		for _, test := range []struct {
			code int
			raw  string
		}{{200, `{"images":{}}`}, {201, `{"images":[]}`}, {200, `{"images":[{"size":"3"}]}`}} {
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { return taskCoreJSON(req, test.code, test.raw), nil })
			rows, err := New(client).AllImages(context.Background())
			if len(rows) != 0 || err == nil {
				t.Fatal("legacy decoder changed", rows, err, test)
			}
		}
	})
}
