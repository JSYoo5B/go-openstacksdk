package hypervisors_test

import (
	"context"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestHypervisorRecordsSummaryDetailAndDeclaredProjection(t *testing.T) {
	for _, detail := range []bool{false, true} {
		t.Run(map[bool]string{false: "summary", true: "detail"}[detail], func(t *testing.T) {
			api, calls := pythonHypervisorAPI(t, "2.88", func(r *http.Request) *http.Response {
				want := pythonHypervisorBase
				if detail {
					want += "/detail"
				}
				if r.URL.Path != want || r.URL.Query().Get("with_servers") != "True" {
					t.Fatal(r.URL)
				}
				return pythonHypervisorWire(203, `{"hypervisors":[{"id":7,"hypervisor_hostname":"node","service":false,"servers":[1,{"uuid":"vm"}],"free_ram_mb":"raw","x_unknown":true}]}`)
			})
			seq := api.ListRecords(context.Background(), hypervisors.WithRecordDetails(detail), hypervisors.WithRecordQuery("with_servers", true))
			if calls.Len() != 0 {
				t.Fatal(pythonHypervisorCalls(calls))
			}
			n := 0
			for row, err := range seq {
				if err != nil || row == nil {
					t.Fatal(row, err)
				}
				n++
				if string(row.Resource.Body["name"]) != `"node"` || string(row.Resource.Body["service_details"]) != "{}" || string(row.Resource.Body["servers"]) != `[{},{"uuid":"vm"}]` || string(row.Resource.Body["memory_free"]) != `"raw"` || row.Resource.Body["x_unknown"] != nil || string(row.Wire.Body["x_unknown"]) != "true" || row.StatusCode != 203 {
					t.Fatal(row)
				}
			}
			if n != 1 || calls.Len() != 1 {
				t.Fatal(n, pythonHypervisorCalls(calls))
			}
		})
	}
}
func TestHypervisorRecordsLegacyHostnameSearchOwnsPatternAndVersion(t *testing.T) {
	for _, version := range []string{"2.52", "2.53"} {
		t.Run(version, func(t *testing.T) {
			api, _ := pythonHypervisorAPI(t, version, func(r *http.Request) *http.Response {
				if !strings.Contains(r.URL.Path, "os-hypervisors") {
					return pythonHypervisorWire(200, `{"version":{"id":"v2.1","status":"CURRENT","min_version":"2.1","version":"2.95","links":[{"rel":"self","href":"http://`+r.Host+`/nova/v2.1/"}]}}`)
				}

				if version == "2.52" {
					if r.URL.EscapedPath() != pythonHypervisorBase+"/node%2Fname/search" || r.URL.Query().Has("hypervisor_hostname_pattern") {
						t.Fatal(r.URL)
					}
				} else if r.URL.Path != pythonHypervisorBase+"/detail" || r.URL.Query().Get("hypervisor_hostname_pattern") != "node/name" {
					t.Fatal(r.URL)
				}
				return pythonHypervisorWire(200, `{"hypervisors":[]}`)
			})
			api.RawClient().Endpoint = api.RawClient().ResourceBase
			rows, err := api.AllRecords(context.Background(), hypervisors.WithRecordDetails(true), hypervisors.WithRecordQuery("hypervisor_hostname_pattern", "node/name"))
			if err != nil || len(rows) != 0 {
				t.Fatal(rows, err)
			}
		})
	}
}
func TestHypervisorFindRecordsGETThenConfiguredFallbackPreservesPassiveID(t *testing.T) {
	for _, detail := range []bool{false, true} {
		t.Run(map[bool]string{false: "summary", true: "detail"}[detail], func(t *testing.T) {
			n := 0
			api, calls := pythonHypervisorAPI(t, "2.88", func(r *http.Request) *http.Response {
				n++
				if n == 1 {
					return pythonHypervisorWire(404, `{}`)
				}
				want := pythonHypervisorBase
				if detail {
					want += "/detail"
				}
				if r.URL.Path != want {
					t.Fatal(r.URL)
				}
				return pythonHypervisorWire(200, `{"hypervisors":[{"id":7,"hypervisor_hostname":"node"}]}`)
			})
			row, err := api.FindRecord(context.Background(), "node", hypervisors.WithRecordDetails(detail))
			if err != nil || row == nil || calls.Len() != 2 || string(row.Resource.Body["id"]) != "7" {
				t.Fatal(row, err, pythonHypervisorCalls(calls))
			}
		})
	}
	for _, rawID := range []string{`7`, `"7"`} {
		t.Run("literal ID "+rawID, func(t *testing.T) {
			n := 0
			api, _ := pythonHypervisorAPI(t, "2.88", func(r *http.Request) *http.Response {
				n++
				if n == 1 {
					return pythonHypervisorWire(404, `{}`)
				}
				return pythonHypervisorWire(200, `{"hypervisors":[{"id":`+rawID+`,"hypervisor_hostname":"other"}]}`)
			})
			row, err := api.FindRecord(context.Background(), "7")
			if err != nil || n != 2 {
				t.Fatal(row, err, n)
			}
			if rawID == `7` {
				if row != nil {
					t.Fatal("numeric ID must not equal string identity", row)
				}
			} else if row == nil {
				t.Fatal("literal string ID lost")
			}
		})
	}

}
func TestHypervisorFindRecordsStrictMissingDuplicatesAndLateError(t *testing.T) {
	for _, mode := range []string{"missing", "duplicate", "late"} {
		t.Run(mode, func(t *testing.T) {
			n := 0
			api, _ := pythonHypervisorAPI(t, "2.88", func(r *http.Request) *http.Response {
				n++
				if n == 1 {
					return pythonHypervisorWire(403, `{}`)
				}
				if mode == "missing" {
					return pythonHypervisorWire(200, `{"hypervisors":[]}`)
				}
				if n == 2 {
					return pythonHypervisorWire(200, `{"hypervisors":[{"id":"a","hypervisor_hostname":"node"}],"next":"?marker=a"}`)
				}
				if mode == "late" {
					return pythonHypervisorWire(500, `{"error":"late"}`)
				}
				return pythonHypervisorWire(200, `{"hypervisors":[{"id":"a","hypervisor_hostname":"node"}]}`)
			})
			row, err := api.FindRecord(context.Background(), "node", hypervisors.WithRecordIgnoreMissing(false))
			if row != nil || err == nil {
				t.Fatal(row, err)
			}
			if mode == "missing" && !errors.Is(err, resource.ErrNotFound) {
				t.Fatal(err)
			}
			if mode == "duplicate" && !errors.Is(err, resource.ErrAmbiguous) {
				t.Fatal(err)
			}
			if mode == "late" && !strings.Contains(err.Error(), "500") {
				t.Fatal(err)
			}
		})
	}
}
func TestHypervisorRecordsDefaultDiscoveryAndOptionSnapshots(t *testing.T) {
	var proof string
	api, calls := pythonHypervisorAPI(t, "", func(r *http.Request) *http.Response {
		if !strings.Contains(r.URL.Path, "os-hypervisors") {
			proof = `{"version":{"id":"v2.1","status":"CURRENT","min_version":"2.1","version":"2.95","links":[{"rel":"self","href":"http://` + r.Host + `/nova/v2.1/"}]}}`
			return pythonHypervisorWire(200, proof)
		}
		if r.Header.Get("OpenStack-API-Version") != "compute 2.88" || r.Header.Get("X-Owned") != "original" {
			t.Fatal(r.Header)
		}
		return pythonHypervisorWire(200, `{"hypervisors":[{"id":"one"}]}`)
	})
	headers := map[string]string{"X-Owned": "original"}
	option := hypervisors.WithRecordHeaders(headers)
	headers["X-Owned"] = "changed"
	api.RawClient().Endpoint = api.RawClient().ResourceBase
	rows, err := api.AllRecords(context.Background(), option)
	if err != nil || len(rows) != 1 || calls.Len() != 2 || len(rows[0].Discovery.Responses) != 1 || string(rows[0].Discovery.Responses[0].Body) != proof {
		t.Fatal(rows, err, pythonHypervisorCalls(calls))
	}
}
func TestHypervisorRecordPreflightAndPhysicalFailureNeverFallback(t *testing.T) {
	api, calls := pythonHypervisorAPI(t, "2.88", func(r *http.Request) *http.Response { return pythonHypervisorWire(200, `{}`) })
	_, err := api.FindRecord(context.Background(), "x", nil)
	if err == nil || calls.Len() != 0 {
		t.Fatal(err, pythonHypervisorCalls(calls))
	}
	cause := errors.New("cancelled hypervisor")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	_, err = api.FindRecord(ctx, "x")
	if !errors.Is(err, cause) || calls.Len() != 0 {
		t.Fatal(err, pythonHypervisorCalls(calls))
	}
}

type hypervisorReadFault struct {
	body        io.ReadCloser
	read, close error
}

func (b *hypervisorReadFault) Read(p []byte) (int, error) {
	n, err := b.body.Read(p)
	if n > 0 && b.read != nil {
		return n, b.read
	}
	return n, err
}
func (b *hypervisorReadFault) Close() error { return errors.Join(b.body.Close(), b.close) }
func TestHypervisorFindPhysicalReadAndCloseFaultsNeverFallback(t *testing.T) {
	for _, mode := range []string{"read", "close"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New("hypervisor physical " + mode)
			api, calls := pythonHypervisorAPI(t, "2.88", func(r *http.Request) *http.Response {
				response := pythonHypervisorWire(404, `{"error":"physical"}`)
				fault := &hypervisorReadFault{body: response.Body}
				if mode == "read" {
					fault.read = cause
				} else {
					fault.close = cause
				}
				response.Body = fault
				return response
			})
			_, err := api.FindRecord(context.Background(), "node")
			if !errors.Is(err, cause) || calls.Len() != 1 {
				t.Fatal(err, pythonHypervisorCalls(calls))
			}
		})
	}
}

func TestHypervisorProxyRecordsEvaluateSharedJMESAgainstLazyGenerator(t *testing.T) {
	for _, expr := range []string{"[*].name", "@", "length(@)", "{lazy: @}"} {
		t.Run(expr, func(t *testing.T) {
			api, calls := pythonHypervisorAPI(t, "2.88", func(r *http.Request) *http.Response {
				return pythonHypervisorWire(200, `{"hypervisors":[{"id":"one"}]}`)
			})
			result, err := api.ProxyListRecords(context.Background(), hypervisors.WithRecordQuery("jmespath_filters", expr))
			if calls.Len() != 0 {
				t.Fatal("JMES consumed generator", pythonHypervisorCalls(calls))
			}
			if expr == "length(@)" {
				if err == nil {
					t.Fatal(result)
				}
				return
			}
			if err != nil || result == nil {
				t.Fatal(result, err)
			}
			switch expr {
			case "[*].name":
				if result.Value != nil || string(result.JSON) != "null" {
					t.Fatal(result)
				}
			case "@":
				generator, ok := result.Value.(*hypervisors.RecordGenerator)
				if !ok {
					t.Fatal(result)
				}
				n := 0
				for row, cause := range generator.Records {
					if cause != nil || row == nil {
						t.Fatal(row, cause)
					}
					n++
				}
				if n != 1 || calls.Len() != 1 {
					t.Fatal(n, pythonHypervisorCalls(calls))
				}
			case "{lazy: @}":
				object, ok := result.Value.(map[string]any)
				if !ok {
					t.Fatal(result)
				}
				if _, ok := object["lazy"].(*hypervisors.RecordGenerator); !ok {
					t.Fatal(result)
				}
			}
		})
	}
}
func TestHypervisorRecordHeadersRejectBeforeUnversionedDiscovery(t *testing.T) {
	api, calls := pythonHypervisorAPI(t, "", func(r *http.Request) *http.Response { return pythonHypervisorWire(500, `{}`) })
	_, err := api.AllRecords(context.Background(), hypervisors.WithRecordHeaders(map[string]string{"X-Auth-Token": "foreign"}))
	if !errors.Is(err, resource.ErrInvalidOption) || calls.Len() != 0 {
		t.Fatal(err, pythonHypervisorCalls(calls))
	}
}
