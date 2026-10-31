package nodes_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func nodeWire(code int, body string) *http.Response {
	return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}, "X-Request-Id": {"node-proof"}})
}
func nodeAPI(t *testing.T, callback func(*http.Request) *http.Response) (*nodes.API, *testcloud.Cloud, *int) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := new(int)
	cloud.SetTransport(testcloud.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		*calls++
		response := callback(r)
		response.Request = r
		return response, nil
	}))
	client := cloud.Client("baremetal", "/ironic/v1/")
	client.Microversion = "1.109"
	return nodes.New(client), cloud, calls
}
func field(t *testing.T, row *nodes.NodeRecord, key, want string) {
	t.Helper()
	if row == nil || row.Resource == nil || string(row.Resource.Body[key]) != want {
		t.Fatalf("%s want %s: %+v", key, want, row)
	}
}

func TestPythonNodeListDetailsCSVAndDeclaredProjection(t *testing.T) {
	api, _, calls := nodeAPI(t, func(r *http.Request) *http.Response {
		if r.Method != "GET" || r.URL.Path != "/ironic/v1/nodes/detail" || r.URL.Query().Get("fields") != "uuid,instance_uuid" || r.URL.Query().Get("maintenance") != "False" || r.URL.Query().Has("driver_info") || r.URL.Query().Has("details") || r.URL.Query().Has("sort_key") {
			t.Fatal(r.URL)
		}
		return nodeWire(203, `{"nodes":[{"uuid":"n","name":"node","driver_info":{"vendor":"v","extra":1},"properties":false,"traits":"trait","maintenance":0,"x_unknown":9007199254740993}]}`)
	})
	seq := api.ListRecords(context.Background(), nodes.WithRecordListDetails(true), nodes.WithRecordListQuery("fields", []string{"id", "instance_id"}), nodes.WithRecordListQuery("is_maintenance", false), nodes.WithRecordListQuery("driver_info", map[string]any{"vendor": "v"}), nodes.WithRecordListQuery("sort_key", "name"))
	if *calls != 0 {
		t.Fatal(*calls)
	}
	rows := 0
	for row, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		rows++
		field(t, row, "id", `"n"`)
		field(t, row, "is_maintenance", "false")
		field(t, row, "properties", "{}")
		field(t, row, "traits", `["trait"]`)
		field(t, row, "allocation_id", "null")
		if row.Resource.Body["x_unknown"] != nil || row.Wire.Body["x_unknown"] == nil || row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "node-proof" {
			t.Fatal(row)
		}
	}
	if rows != 1 || *calls != 1 {
		t.Fatal(rows, *calls)
	}
}
func TestPythonNodeListExplicitPathAndSourceCSVString(t *testing.T) {
	api, _, calls := nodeAPI(t, func(r *http.Request) *http.Response {
		if r.URL.Path != "/ironic/v1/custom" || r.URL.Query().Get("fields") != "i,d" {
			t.Fatal(r.URL)
		}
		return nodeWire(200, `{"nodes":[]}`)
	})
	rows, err := api.AllRecords(context.Background(), nodes.WithRecordListDetails(true), nodes.WithRecordListBasePath("/custom"), nodes.WithRecordListQuery("fields", "id"))
	if err != nil || rows == nil || len(rows) != 0 || *calls != 1 {
		t.Fatal(rows, err, *calls)
	}
}
func TestPythonNodeGetMapsFieldsAndKeepsPassiveUUID(t *testing.T) {
	const body = `{"uuid":null,"name":"different","driver_info":7,"allocation_uuid":"a","portgroups":{"href":"p"},"x_extension":true}`
	api, _, calls := nodeAPI(t, func(r *http.Request) *http.Response {
		if r.URL.EscapedPath() != "/ironic/v1/nodes/folder%2Fnode%25" || r.URL.Query().Get("fields") != "uuid,allocation_uuid,portgroups,unknown,uuid" {
			t.Fatal(r.URL)
		}
		return nodeWire(202, body)
	})
	row, err := api.GetRecord(context.Background(), "folder/node%", nodes.WithRecordGetFields("id", "allocation_id", "port_groups", "unknown", "id"))
	if err != nil || *calls != 1 || string(row.Envelope) != body {
		t.Fatal(row, err, *calls)
	}
	field(t, row, "id", "null")
	field(t, row, "allocation_id", `"a"`)
	field(t, row, "driver_info", "{}")
	field(t, row, "port_groups", `[{"href":"p"}]`)
	if row.Resource.Body["x_extension"] != nil || string(row.Wire.Body["uuid"]) != "null" {
		t.Fatal(row)
	}
}
func TestPythonNodeFindFallbackDetailsDefaultsAndMissing(t *testing.T) {
	for _, mode := range []string{"detail", "basic", "missing", "strict", "duplicate", "late"} {
		t.Run(mode, func(t *testing.T) {
			api, _, calls := nodeAPI(t, func(r *http.Request) *http.Response {
				if strings.HasSuffix(r.URL.Path, "/wanted") {
					return nodeWire(404, `{}`)
				}
				want := "/ironic/v1/nodes/detail"
				if mode == "basic" {
					want = "/ironic/v1/nodes"
				}
				if r.URL.Path != want || r.URL.Query().Has("name") || r.URL.Query().Has("details") {
					t.Fatal(r.URL)
				}
				if mode == "missing" || mode == "strict" {
					return nodeWire(200, `{"nodes":[]}`)
				}
				if r.URL.Query().Has("marker") {
					if mode == "late" {
						return nodeWire(503, `{"error":"late"}`)
					}
					return nodeWire(200, `{"nodes":[{"uuid":"n","name":"wanted"}]}`)
				}
				if mode == "duplicate" || mode == "late" {
					return nodeWire(200, `{"nodes":[{"uuid":"n","name":"wanted"}],"next":"?marker=n"}`)
				}
				return nodeWire(200, `{"nodes":[{"uuid":"n","name":"wanted"}]}`)
			})
			options := []nodes.RecordFindOption{}
			if mode == "basic" {
				options = append(options, nodes.WithRecordFindDetails(false))
			}
			if mode == "strict" {
				options = append(options, nodes.WithRecordFindIgnoreMissing(false))
			}
			row, err := api.FindRecord(context.Background(), "wanted", options...)
			switch mode {
			case "detail", "basic":
				if err != nil || row == nil || *calls != 2 {
					t.Fatal(row, err, *calls)
				}
				field(t, row, "id", `"n"`)
			case "missing":
				if row != nil || err != nil {
					t.Fatal(row, err)
				}
			case "strict":
				if !errors.Is(err, resource.ErrNotFound) {
					t.Fatal(err)
				}
			case "duplicate":
				if !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(err)
				}
			case "late":
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &proof) || proof.Actual != 503 {
					t.Fatal(err)
				}
			}
		})
	}
}
func TestPythonNodeFindGETDoesNotRematchAndGet404IsStrict(t *testing.T) {
	api, _, calls := nodeAPI(t, func(r *http.Request) *http.Response {
		if strings.HasSuffix(r.URL.Path, "/missing") {
			return nodeWire(404, `{}`)
		}
		return nodeWire(200, `{"uuid":"actual","name":"different"}`)
	})
	row, err := api.FindRecord(context.Background(), "wanted")
	if err != nil || row == nil || *calls != 1 {
		t.Fatal(row, err, *calls)
	}
	field(t, row, "name", `"different"`)
	_, err = api.GetRecord(context.Background(), "missing")
	var proof gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &proof) || proof.Actual != 404 || *calls != 2 {
		t.Fatal(err, *calls)
	}
}
func TestPythonNodeListProxyPreservesOpaqueGeneratorWithoutHTTP(t *testing.T) {
	api, _, calls := nodeAPI(t, func(*http.Request) *http.Response { return nodeWire(200, `{"nodes":[]}`) })
	result, err := api.ProxyListRecords(context.Background(), nodes.WithRecordListJMESPath("[*].id"))
	if err != nil || string(result.JSON) != "null" || *calls != 0 {
		t.Fatal(result, err, *calls)
	}
	opaque, err := api.ProxyListRecords(context.Background(), nodes.WithRecordListJMESPath("@"))
	if err != nil {
		t.Fatal(err)
	}
	generator, ok := opaque.Value.(*nodes.RecordListGenerator)
	if !ok || *calls != 0 {
		t.Fatal(opaque, *calls)
	}
	for _, err := range generator.Records {
		if err != nil {
			t.Fatal(err)
		}
	}
	if *calls != 1 {
		t.Fatal(*calls)
	}
}
func TestPythonNodeOptionsOwnDataAndGuardPreflight(t *testing.T) {
	headers := map[string]string{"X-Owned": "original"}
	fields := []string{"id"}
	option := nodes.WithRecordGetOpts(nodes.RecordGetOpts{Fields: fields, Headers: headers})
	headers["X-Owned"] = "mutated"
	fields[0] = "name"
	api, cloud, calls := nodeAPI(t, func(r *http.Request) *http.Response {
		if r.Header.Get("X-Owned") != "original" || r.URL.Query().Get("fields") != "uuid" {
			t.Fatal(r.Header, r.URL)
		}
		return nodeWire(200, `{"uuid":"n"}`)
	})
	if _, err := api.GetRecord(context.Background(), "n", option); err != nil {
		t.Fatal(err)
	}
	before := *calls
	for _, id := range []string{"", "..", "bad\n"} {
		if _, err := api.GetRecord(context.Background(), id); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(id, err)
		}
	}
	if _, err := api.GetRecord(context.Background(), "n", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	_, err := api.GetRecord(context.Background(), "n", func(*nodes.RecordGetOpts) error {
		*api = *nodes.New(cloud.Client("baremetal", "/different/"))
		return nil
	})
	if err == nil || *calls != before {
		t.Fatal(err, *calls, before)
	}
}
func TestPythonNodeDefaultVersionDiscoveryAndUncappedSelection(t *testing.T) {
	cloud := testcloud.New(t)
	var paths []string
	serverMaximum, expectedSelection := "1.100", "1.100"
	cloud.SetTransport(testcloud.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		if strings.HasSuffix(r.URL.Path, "/nodes/n") {
			if r.Header.Get("X-OpenStack-Ironic-API-Version") != expectedSelection {
				t.Fatal(r.Header)
			}
			return nodeWire(200, `{"uuid":"n"}`), nil
		}
		return nodeWire(200, `{"version":{"id":"v1","min_version":"1.1","version":"`+serverMaximum+`"}}`), nil
	}))
	api := nodes.New(cloud.Client("baremetal", "/ironic/v1/"))
	row, err := api.GetRecord(context.Background(), "n")
	if err != nil || row == nil || len(row.Discovery.Responses) == 0 || len(paths) != 2 {
		t.Fatal(row, err, paths)
	}
	copy := row.Clone()
	copy.Discovery.Responses[0].Body[0] = '!'
	if row.Discovery.Responses[0].Body[0] == '!' {
		t.Fatal("discovery alias")
	}
	serverMaximum, expectedSelection = "1.120", "1.109"
	paths = nil
	row, err = api.GetRecord(context.Background(), "n")
	if err != nil || row == nil || len(paths) != 2 || row.Discovery.Maximum != "1.120" {
		t.Fatal("class ceiling selection", row, err, paths)
	}
	api, _, calls := nodeAPI(t, func(r *http.Request) *http.Response {
		if r.Header.Get("X-OpenStack-Ironic-API-Version") != "1.120" {
			t.Fatal(r.Header)
		}
		return nodeWire(200, `{"uuid":"n"}`)
	})
	row, err = api.GetRecord(context.Background(), "n", nodes.WithRecordGetMicroversion("1.120"))
	if err != nil || row == nil || len(row.Discovery.Responses) != 0 || *calls != 1 {
		t.Fatal(row, err, *calls)
	}
}

type nodeFaultBody struct {
	reader     io.Reader
	cause      error
	onClose    func()
	closed     int
	closeFault bool
}

func (b *nodeFaultBody) Read(p []byte) (int, error) {
	if b.reader != nil {
		n, err := b.reader.Read(p)
		if n > 0 {
			return n, nil
		}
		if err != io.EOF {
			return n, err
		}
	}
	if b.closeFault {
		return 0, io.EOF
	}
	return 0, b.cause
}
func (b *nodeFaultBody) Close() error {
	b.closed++
	if b.onClose != nil {
		b.onClose()
	}
	if b.closeFault {
		return b.cause
	}
	return nil
}
func TestPythonNodePhysicalReadCloseAndCancelNeverFallback(t *testing.T) {
	for _, mode := range []string{"read", "close", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New(mode)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			body := &nodeFaultBody{reader: strings.NewReader(`{"error":"unclean404"}`), cause: cause, closeFault: mode != "read"}
			if mode == "cancel" {
				body.onClose = func() { cancel(cause) }
			}
			api, _, calls := nodeAPI(t, func(*http.Request) *http.Response { r := nodeWire(404, `{}`); r.Body = body; return r })
			_, err := api.FindRecord(ctx, "wanted")
			if !errors.Is(err, cause) || *calls != 1 || body.closed != 1 {
				t.Fatal(err, *calls, body.closed)
			}
		})
	}
}
func TestPythonNodeListPaginationStopsAfterBreakAndDropsPartialCollection(t *testing.T) {
	for _, early := range []bool{true, false} {
		t.Run(map[bool]string{true: "early", false: "late"}[early], func(t *testing.T) {
			api, _, calls := nodeAPI(t, func(r *http.Request) *http.Response {
				if r.URL.Query().Get("marker") == "one" {
					return nodeWire(503, `{"error":"late"}`)
				}
				return nodeWire(200, `{"nodes":[{"uuid":"one"}],"next":"?marker=one"}`)
			})
			if early {
				for _, err := range api.ListRecords(context.Background()) {
					if err != nil {
						t.Fatal(err)
					}
					break
				}
				if *calls != 1 {
					t.Fatal(*calls)
				}
			} else {
				rows, err := api.AllRecords(context.Background())
				if err == nil || rows != nil || *calls != 2 {
					t.Fatal(rows, err, *calls)
				}
			}
		})
	}
}
func TestPythonNodeHeadersCapturedBeforeCallbacks(t *testing.T) {
	api, _, calls := nodeAPI(t, func(r *http.Request) *http.Response {
		if r.Header.Get("X-Captured") != "old" {
			t.Fatal(r.Header)
		}
		return nodeWire(200, `{"uuid":"n"}`)
	})
	api.RawClient().MoreHeaders = map[string]string{"X-Captured": "old"}
	_, err := api.GetRecord(context.Background(), "n", func(*nodes.RecordGetOpts) error { api.RawClient().MoreHeaders["X-Captured"] = "new"; return nil })
	if err != nil || *calls != 1 || !reflect.DeepEqual(api.RawClient().MoreHeaders, map[string]string{"X-Captured": "new"}) {
		t.Fatal(err, *calls)
	}
}
