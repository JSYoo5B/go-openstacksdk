package addressgroups_test

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

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const pythonAddressListPath = "/neutron/v2.0/address-groups"

type addressListMarshaler func() ([]byte, error)

func (m addressListMarshaler) MarshalJSON() ([]byte, error) { return m() }

func TestPythonAddressGroupListSourceProjectionQueryAndEvidence(t *testing.T) {
	body := `{"address_groups":[{"id":"wanted","name":false,"description":0,"tenant_id":true,"addresses":{"cidr":false},"if_match":"unknown-body","x_row":9007199254740993,"self":"drop"},{"id":"discard","project_id":null,"tenant_id":"legacy","addresses":null}]}`
	var calls []nativeGroupCall
	api, _ := nativeGroupAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.Method != http.MethodGet || req.URL.Path != pythonAddressListPath || req.Header.Get("Accept") != "application/json" || req.Header.Get("OpenStack-API-Version") != "network 2.0" {
			t.Error(req.Method, req.URL, req.Header)
		}
		query := req.URL.Query()
		if !reflect.DeepEqual(query["fields"], []string{"id", "tenant_id", "addresses"}) || query.Get("project_id") != "user-project" || query.Get("name") != "False" || query.Get("description") != "0" || query.Get("sort_dir") != "asc" || query.Has("id") || query.Has("addresses") || query.Has("tenant_id") || query.Has("unknown") {
			t.Error(query)
		}
		response := nativeGroupWire(203, body)
		response.Header.Set("If-Match", "actual-header")
		response.Header.Set("X-Request-Id", "list-proof")
		return response
	})
	api.RawClient().Microversion = "2.0"
	api.RawClient().MoreHeaders = map[string]string{"Accept": "application/source"}
	values, err := api.AllRecords(context.Background(), addressgroups.WithRecordListQuery("fields", []any{"id", nil, "tenant_id", "addresses"}), addressgroups.WithRecordListQuery("project_id", "user-project"), addressgroups.WithRecordListQuery("name", false), addressgroups.WithRecordListQuery("description", 0), addressgroups.WithRecordListQuery("sort_dir", "asc"), addressgroups.WithRecordListQuery("id", "wanted"), addressgroups.WithRecordListQuery("tenant_id", 1), addressgroups.WithRecordListQuery("addresses", []any{map[string]any{"cidr": false}}), addressgroups.WithRecordListQuery("unknown", addressListMarshaler(func() ([]byte, error) { panic("unknown list value must not be evaluated") })))
	if err != nil || len(values) != 1 || len(calls) != 1 || calls[0].body != "" {
		t.Fatal(values, err, calls)
	}
	value := values[0]
	for key, want := range map[string]string{"id": `"wanted"`, "name": "false", "description": "0", "project_id": "true", "tenant_id": "true", "addresses": `[{"cidr":false}]`, "if_match": `"unknown-body"`, "x_row": "9007199254740993", "location": "null"} {
		pythonFindField(t, value, key, want)
	}
	if value.Resource.Body["tags"] != nil || value.Resource.Body["self"] != nil || value.Wire.Body["self"] == nil || string(value.Wire.Body["addresses"]) != `{"cidr":false}` || value.Header.Get("If-Match") != "actual-header" || value.StatusCode != 203 || string(value.Envelope) != body {
		t.Fatal(value)
	}
	value.Resource.Body["x_row"][0] = 'X'
	value.Header.Set("X-Request-Id", "changed")
	if string(value.Wire.Body["x_row"]) != "9007199254740993" || value.Wire.Header.Get("X-Request-Id") != "list-proof" || value.Resource.Header.Get("X-Request-Id") != "list-proof" {
		t.Fatal(value)
	}
	var sparseCalls []nativeGroupCall
	sparse, _ := nativeGroupAPI(t, &sparseCalls, func(*http.Request) *http.Response {
		return nativeGroupWire(200, `{"address_groups":[{"id":null,"project_id":null,"tenant_id":"legacy"},{}]}`)
	})
	rows, err := sparse.AllRecords(context.Background())
	if err != nil || len(rows) != 2 || sparseCalls[0].query != "" {
		t.Fatal(rows, err, sparseCalls)
	}
	for _, row := range rows {
		pythonFindField(t, row, "addresses", "null")
		if row.Resource.Body["tags"] != nil || row.Resource.Body["if_match"] != nil {
			t.Fatal(row)
		}
	}
	pythonFindField(t, rows[0], "project_id", "null")
	pythonFindField(t, rows[1], "id", "null")
}

func TestPythonAddressGroupListPaginationCapsNextTruthinessAndLateErrors(t *testing.T) {
	for _, nextKind := range []string{"plural", "header", "dictionary", "versioned", "marker"} {
		t.Run(nextKind, func(t *testing.T) {
			var calls []nativeGroupCall
			api, _ := nativeGroupAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Query().Get("marker") != "" {
					return nativeGroupWire(200, `{"address_groups":[]}`)
				}
				body := `{"address_groups":[{"id":9007199254740993,"addresses":false}]`
				switch nextKind {
				case "plural":
					body += `,"address_groups_links":[{"rel":"next","href":"?marker=9007199254740993"}]`
				case "dictionary":
					body += `,"links":{"next":"?marker=9007199254740993"}`
				case "versioned":
					body += `,"next":"/v2.0/address-groups?marker=9007199254740993"`
				}
				response := nativeGroupWire(207, body+`}`)
				if nextKind == "header" {
					response.Header.Set("Link", `<?marker=9007199254740993>; rel="next"`)
				}
				return response
			})
			opts := []addressgroups.RecordListOption{addressgroups.WithRecordListQuery("project_id", "project")}
			if nextKind == "marker" {
				opts = append(opts, addressgroups.WithRecordListLimit(2))
			}
			var values []*addressgroups.AddressGroupListRecord
			for value, err := range api.ListRecords(context.Background(), opts...) {
				if err != nil {
					t.Fatal(err)
				}
				values = append(values, value)
				value.Resource.Body["id"] = json.RawMessage(`"changed"`)
				value.Wire.Body["id"] = json.RawMessage(`"also changed"`)
			}
			if len(values) != 1 || len(calls) != 2 || !strings.Contains(calls[1].query, "marker=9007199254740993") || !strings.Contains(calls[1].query, "project_id=project") {
				t.Fatal(values, calls)
			}
		})
	}
	for _, next := range []string{`1e-400`, `-1e-400`, `1e400`, `true`} {
		var calls []nativeGroupCall
		body := `{"address_groups":[{"id":"one"}],"next":` + next + `}`
		api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response {
			r := nativeGroupWire(203, body)
			r.Header.Set("X-Request-Id", "next-proof")
			return r
		})
		rows, err := api.AllRecords(context.Background())
		if next == `1e-400` || next == `-1e-400` {
			if err != nil || len(rows) != 1 || len(calls) != 1 {
				t.Fatal(rows, err, calls)
			}
		} else {
			var proof *resource.ResponseError
			if rows != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "next-proof" || len(calls) != 1 {
				t.Fatal(rows, err, proof, calls)
			}
		}
	}
	for _, mode := range []string{"cap", "single_page", "break", "late503", "changed_filter", "cycle"} {
		t.Run(mode, func(t *testing.T) {
			var calls []nativeGroupCall
			api, _ := nativeGroupAPI(t, &calls, func(req *http.Request) *http.Response {
				if len(calls) > 1 {
					return nativeGroupWire(503, `{"late":true}`)
				}
				next := "?marker=next"
				if mode == "changed_filter" {
					next += "&project_id=other"
				}
				if mode == "cycle" {
					next = "?marker=start"
				}
				return nativeGroupWire(200, `{"address_groups":[{"id":"discard"},{"id":"wanted"}],"next":"`+next+`"}`)
			})
			options := []addressgroups.RecordListOption{addressgroups.WithRecordListQuery("project_id", "project")}
			switch mode {
			case "cap":
				options = append(options, addressgroups.WithRecordListMaxItems(1), addressgroups.WithRecordListQuery("id", "wanted"))
			case "single_page":
				options = append(options, addressgroups.WithRecordListPaginated(false))
			case "cycle":
				options = append(options, addressgroups.WithRecordListMarker("start"))
			}
			seq := api.ListRecords(context.Background(), options...)
			if len(calls) != 0 {
				t.Fatal("non-lazy", calls)
			}
			if mode == "break" {
				for _, err := range seq {
					if err != nil {
						t.Fatal(err)
					}
					break
				}
				if len(calls) != 1 {
					t.Fatal(calls)
				}
				return
			}
			var rows []*addressgroups.AddressGroupListRecord
			var failure error
			for value, err := range seq {
				if err != nil {
					failure = err
					break
				}
				rows = append(rows, value)
			}
			switch mode {
			case "cap":
				if failure != nil || len(rows) != 0 || len(calls) != 1 {
					t.Fatal(rows, failure, calls)
				}
			case "single_page":
				if failure != nil || len(rows) != 2 || len(calls) != 1 {
					t.Fatal(rows, failure, calls)
				}
			case "late503":
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(failure, &native) || native.Actual != 503 || string(native.Body) != `{"late":true}` || len(calls) != 2 {
					t.Fatal(failure, native, calls)
				}
			default:
				var proof *resource.ResponseError
				if !errors.As(failure, &proof) || proof.StatusCode != 200 || len(calls) != 1 {
					t.Fatal(failure, proof, calls)
				}
			}
		})
	}
}

func TestPythonAddressGroupListConstructorDecodeAndLocalFilterFailures(t *testing.T) {
	for _, status := range []int{199, 200, 203, 204, 299, 399, 400, 404, 503, 599} {
		var calls []nativeGroupCall
		body := `{"address_groups":[{"id":"one"}]}`
		api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response { return nativeGroupWire(status, body) })
		rows, err := api.AllRecords(context.Background())
		if status >= 200 && status < 400 {
			if err != nil || len(rows) != 1 || rows[0].StatusCode != status || len(calls) != 1 {
				t.Fatal(status, rows, err, calls)
			}
		} else {
			var native gophercloud.ErrUnexpectedResponseCode
			if rows != nil || !errors.As(err, &native) || native.Actual != status || string(native.Body) != body || len(calls) != 1 {
				t.Fatal(status, rows, err, native, calls)
			}
		}
	}
	for _, row := range []string{`{"cls":null}`, `{"cls":false}`, `{"connection":null}`, `{"microversion":false}`, `{"_synchronized":null}`, `null`, `false`, `[]`} {
		var calls []nativeGroupCall
		body := `{"address_groups":[` + row + `]}`
		api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response {
			r := nativeGroupWire(299, body)
			r.Header.Set("X-Request-Id", "constructor-proof")
			return r
		})
		rows, err := api.AllRecords(context.Background())
		var proof *resource.ResponseError
		if rows != nil || !errors.As(err, &proof) || proof.StatusCode != 299 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "constructor-proof" || len(calls) != 1 {
			t.Fatal(row, rows, err, proof, calls)
		}
	}
	for _, body := range []string{`{}`, `{"address_groups":null}`, `{"address_groups":[]`, "{\"address_groups\":[{\"id\":\"bad\",\"name\":\"\xff\"}]}", ``} {
		var calls []nativeGroupCall
		api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response { return nativeGroupWire(204, body) })
		_, err := api.AllRecords(context.Background())
		var proof *resource.ResponseError
		if !errors.As(err, &proof) || proof.StatusCode != 204 || string(proof.Body) != body || len(calls) != 1 {
			t.Fatal(body, err, proof, calls)
		}
	}
	for _, tenant := range []string{`"truthy"`, `{}`, `false`, `1e-400`, `{"nested":{"on":1}}`} {
		var calls []nativeGroupCall
		api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response {
			return nativeGroupWire(200, `{"address_groups":[{"id":"one","tenant_id":`+tenant+`}]}`)
		})
		rows, err := api.AllRecords(context.Background(), addressgroups.WithRecordListQuery("tenant_id", map[string]any{}))
		want := tenant == `"truthy"` || tenant == `{"nested":{"on":1}}`
		if err != nil || (len(rows) == 1) != want {
			t.Fatal(tenant, rows, err)
		}
	}
	var calls []nativeGroupCall
	api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeGroupWire(200, `{"address_groups":[{"id":"one","tenant_id":"truthy"}]}`)
	})
	_, err := api.AllRecords(context.Background(), addressgroups.WithRecordListQuery("tenant_id", map[string]any{"nested": 1}))
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || len(calls) != 1 {
		t.Fatal(err, proof, calls)
	}
	// This shared comparison policy deliberately retains exact JSON decimal
	// equality, while scalar truthiness above follows binary64 JSON decoding.
	calls = nil
	api, _ = nativeGroupAPI(t, &calls, func(*http.Request) *http.Response {
		return nativeGroupWire(200, `{"address_groups":[{"tenant_id":9007199254740992}]}`)
	})
	rows, err := api.AllRecords(context.Background(), addressgroups.WithRecordListQuery("tenant_id", json.RawMessage(`9007199254740993.0`)))
	if err != nil || len(rows) != 0 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
}

func TestPythonAddressGroupListPhysicalFaultsRetainActualProofWithoutReplay(t *testing.T) {
	for _, status := range []int{200, 299, 400, 403, 404, 503, 599} {
		for _, mode := range []string{"read_close", "cancel", "source"} {
			t.Run(fmt.Sprintf("%d/%s", status, mode), func(t *testing.T) {
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readCause, closeCause, cancelCause := errors.New("physical read"), errors.New("physical close"), errors.New("physical cancel")
				body := `{"address_groups":[{"id":"one"}],"x_actual":false}`
				physical := &actionFaultBody{data: []byte(body), read: io.EOF}
				closes := 0
				var calls []nativeGroupCall
				api, cloud := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response {
					response := nativeGroupWire(status, "")
					response.Body = physical
					response.Header.Set("X-Request-Id", "physical-proof")
					return response
				})
				original := api.RawClient().ResourceBase
				switch mode {
				case "read_close":
					physical.read, physical.close = readCause, closeCause
				case "cancel":
					physical.onRead = func() { cancel(cancelCause) }
				case "source":
					physical.onRead = func() { api.RawClient().ResourceBase = cloud.Server.URL + "/changed/" }
				}
				physical.onClose = func() {
					closes++
					if mode == "source" {
						api.RawClient().ResourceBase = original
					}
				}
				rows, err := api.AllRecords(ctx)
				if rows != nil || err == nil || len(calls) != 1 || closes != 1 {
					t.Fatal(rows, err, calls, closes)
				}
				switch mode {
				case "read_close":
					if !errors.Is(err, readCause) || !errors.Is(err, closeCause) {
						t.Fatal(err)
					}
				case "cancel":
					if !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) {
						t.Fatal(err)
					}
				case "source":
					if !errors.Is(err, resource.ErrInvalidOption) || api.RawClient().ResourceBase != original {
						t.Fatal(err)
					}
				}
				if status < 400 {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "physical-proof" {
						t.Fatal(err, proof)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "physical-proof" {
						t.Fatal(err, native)
					}
				}
			})
		}
	}
}

func TestPythonAddressGroupListRetryHeadersVersionAndReauthOwnership(t *testing.T) {
	changes := []struct {
		name, version string
		change        func(*gophercloud.RequestOpts)
	}{
		{"selected_change", "2.0", func(o *gophercloud.RequestOpts) { o.MoreHeaders["OpenStack-API-Version"] = "network other" }},
		{"selected_delete", "2.0", func(o *gophercloud.RequestOpts) { delete(o.MoreHeaders, "OpenStack-API-Version") }},
		{"selected_omit", "2.0", func(o *gophercloud.RequestOpts) { o.OmitHeaders = []string{"OPENSTACK-API-VERSION"} }},
		{"unset_inject", "", func(o *gophercloud.RequestOpts) { o.MoreHeaders["OpenStack-API-Version"] = "network 2.0" }},
		{"foreign", "", func(o *gophercloud.RequestOpts) { o.MoreHeaders["X-OpenStack-Volume-API-Version"] = "3.99" }},
		{"authorization", "", func(o *gophercloud.RequestOpts) { o.MoreHeaders["Authorization"] = "foreign" }},
		{"framing", "2.0", func(o *gophercloud.RequestOpts) { o.MoreHeaders["Transfer-Encoding"] = "chunked" }},
		{"body", "", func(o *gophercloud.RequestOpts) { o.JSONBody = json.RawMessage(`null`) }},
	}
	for _, sample := range changes {
		for _, callbackFailure := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/callback=%t", sample.name, callbackFailure), func(t *testing.T) {
				var calls []nativeGroupCall
				api, _ := nativeGroupAPI(t, &calls, func(req *http.Request) *http.Response {
					want := ""
					if sample.version != "" {
						want = "network " + sample.version
					}
					if req.Header.Get("OpenStack-API-Version") != want {
						t.Error(req.Header, want)
					}
					r := nativeGroupWire(503, `{"error":"original list"}`)
					r.Header.Set("X-Request-Id", "retry-proof")
					return r
				})
				api.RawClient().Microversion = sample.version
				callbacks := 0
				cause := errors.New("callback cause")
				api.RawClient().RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
					callbacks++
					sample.change(o)
					if callbackFailure {
						return cause
					}
					return nil
				}
				rows, err := api.AllRecords(context.Background())
				var native gophercloud.ErrUnexpectedResponseCode
				if rows != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"error":"original list"}` || native.ResponseHeader.Get("X-Request-Id") != "retry-proof" || len(calls) != 1 || callbacks != 1 {
					t.Fatal(rows, err, native, calls, callbacks)
				}
				if callbackFailure && !errors.Is(err, cause) {
					t.Fatal(err)
				}
			})
		}
	}
	for _, reauth := range []bool{false, true} {
		var calls []nativeGroupCall
		var tokens []string
		api, cloud := nativeGroupAPI(t, &calls, func(req *http.Request) *http.Response {
			tokens = append(tokens, req.Header.Get("X-Auth-Token"))
			if len(calls) == 1 {
				if reauth {
					return nativeGroupWire(401, `{}`)
				}
				return nativeGroupWire(503, `{}`)
			}
			if !reauth && req.Header.Get("X-Retry") != "ordinary" {
				t.Error(req.Header)
			}
			return nativeGroupWire(200, `{"address_groups":[{"id":"one"}]}`)
		})
		if reauth {
			cloud.Provider.ReauthFunc = func(context.Context) error { cloud.Provider.SetToken("refreshed"); return nil }
		} else {
			cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, _ error, _ uint) error {
				o.MoreHeaders["X-Retry"] = "ordinary"
				cloud.Provider.SetToken("refreshed")
				return nil
			}
		}
		rows, err := api.AllRecords(context.Background())
		if err != nil || len(rows) != 1 || len(calls) != 2 || calls[0] != calls[1] || !reflect.DeepEqual(tokens, []string{"test-token", "refreshed"}) {
			t.Fatal(rows, err, calls, tokens)
		}
	}
}

func TestPythonAddressGroupListOptionsPreflightSnapshotsAndConcurrentReuse(t *testing.T) {
	headers := map[string]string{"Accept": "application/explicit"}
	query := map[string]json.RawMessage{"fields": json.RawMessage(`["id"]`), "unknown": json.RawMessage(`invalid`)}
	option := addressgroups.WithRecordListOpts(addressgroups.RecordListOpts{Query: query, Headers: headers})
	query["fields"][2] = 'X'
	headers["Accept"] = "changed"
	var retained *addressgroups.RecordListOpts
	var calls []nativeGroupCall
	api, _ := nativeGroupAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.Header.Get("Accept") != "application/explicit" || req.URL.Query().Get("fields") != "id" || req.URL.Query().Has("unknown") {
			t.Error(req.Header, req.URL)
		}
		retained.Query["fields"][2] = 'Y'
		retained.Headers["Accept"] = "retained"
		return nativeGroupWire(200, `{"address_groups":[{"id":"one"}]}`)
	})
	rows, err := api.AllRecords(context.Background(), option, func(c *addressgroups.RecordListOpts) error { retained = c; return nil })
	if err != nil || len(rows) != 1 || len(calls) != 1 {
		t.Fatal(rows, err, calls)
	}
	for _, mode := range []string{"nil_ctx", "nil_option", "cancel", "source", "cause", "protected_header", "bad_json", "negative_limit", "negative_cap", "reserved", "marshal"} {
		var calls []nativeGroupCall
		api, _ := nativeGroupAPI(t, &calls, func(*http.Request) *http.Response { return nativeGroupWire(200, `{}`) })
		ctx, cancel := context.WithCancelCause(context.Background())
		defer cancel(nil)
		var actual context.Context = ctx
		cause := errors.New("option cause")
		var options []addressgroups.RecordListOption
		switch mode {
		case "nil_ctx":
			actual = nil
		case "nil_option":
			options = []addressgroups.RecordListOption{nil}
		case "cancel":
			options = append(options, func(*addressgroups.RecordListOpts) error { cancel(cause); return nil })
		case "source":
			options = append(options, func(*addressgroups.RecordListOpts) error { api.RawClient().Type = "compute"; return cause })
		case "cause":
			options = append(options, func(*addressgroups.RecordListOpts) error { return cause })
		case "protected_header":
			options = append(options, addressgroups.WithRecordListHeaders(map[string]string{"Authorization": "foreign"}))
		case "bad_json":
			options = append(options, addressgroups.WithRecordListOpts(addressgroups.RecordListOpts{Query: map[string]json.RawMessage{"id": json.RawMessage(`invalid`)}}))
		case "negative_limit":
			options = append(options, addressgroups.WithRecordListLimit(-1))
		case "negative_cap":
			options = append(options, addressgroups.WithRecordListMaxItems(-1))
		case "reserved":
			options = append(options, addressgroups.WithRecordListQuery("cls", nil))
		case "marshal":
			options = append(options, addressgroups.WithRecordListQuery("name", addressListMarshaler(func() ([]byte, error) { return nil, cause })))
		}
		rows, err := api.AllRecords(actual, options...)
		if err == nil || rows != nil || len(calls) != 0 {
			t.Fatal(mode, rows, err, calls)
		}
		if (mode == "cancel" || mode == "source" || mode == "cause" || mode == "marshal") && !errors.Is(err, cause) {
			t.Fatal(mode, err)
		}
	}
	var unused []nativeGroupCall
	api, cloud := nativeGroupAPI(t, &unused, func(*http.Request) *http.Response { panic("replaced transport") })
	var observed atomic.Int32
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		observed.Add(1)
		if req.URL.Query().Get("name") != "owned" {
			return nil, fmt.Errorf("unowned query: %s", req.URL)
		}
		return nativeGroupWire(200, `{"address_groups":[{"id":"one","x_raw":[false,0,null]}]}`), nil
	}))
	shared := addressgroups.WithRecordListQuery("name", "owned")
	values := make([][]*addressgroups.AddressGroupListRecord, 8)
	failures := make([]error, 8)
	var workers sync.WaitGroup
	for i := range values {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			values[i], failures[i] = api.AllRecords(context.Background(), shared)
		}(i)
	}
	workers.Wait()
	if observed.Load() != 8 {
		t.Fatal(observed.Load())
	}
	for i, rows := range values {
		if failures[i] != nil || len(rows) != 1 {
			t.Fatal(rows, failures[i])
		}
		rows[0].Resource.Body["x_raw"][0] = 'X'
		if string(rows[0].Wire.Body["x_raw"]) != `[false,0,null]` || !strings.Contains(string(rows[0].Envelope), `[false,0,null]`) {
			t.Fatal(rows)
		}
	}
}

func TestPythonAddressGroupListExplicitVersionOwnsEveryPageAndOriginalHeaders(t *testing.T) {
	for _, version := range []string{"2.9", ""} {
		var calls []nativeGroupCall
		api, _ := nativeGroupAPI(t, &calls, func(req *http.Request) *http.Response {
			want := ""
			if version != "" {
				want = "network " + version
			}
			if req.Header.Get("OpenStack-API-Version") != want {
				t.Error(req.Header, want)
			}
			if req.URL.Query().Get("marker") == "next" {
				return nativeGroupWire(200, `{"address_groups":[]}`)
			}
			return nativeGroupWire(200, `{"address_groups":[{"id":"one"}],"next":"?marker=next"}`)
		})
		api.RawClient().Microversion = "2.0"
		api.RawClient().MoreHeaders = map[string]string{"Openstack-Api-Version": "network 2.0"}
		rows, err := api.AllRecords(context.Background(), addressgroups.WithRecordListMicroversion(version))
		if err != nil || len(rows) != 1 || len(calls) != 2 || api.RawClient().Microversion != "2.0" || api.RawClient().MoreHeaders["Openstack-Api-Version"] != "network 2.0" {
			t.Fatal(rows, err, calls, api.RawClient())
		}
	}
}
