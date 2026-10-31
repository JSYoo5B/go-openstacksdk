package subnetpools_test

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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const poolRecordsPath = "/neutron/v2.0/subnetpools"

func poolRecordValue(t *testing.T, value *subnetpools.SubnetPoolRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatalf("%s: %#v want %s", key, value, want)
	}
}
func TestPythonSubnetPoolRecordsUpdateAliasesNoopAndSeedWireProjection(t *testing.T) {
	var calls []nativePoolCall
	var escaped, conditional string
	api, _ := nativePoolAPI(t, &calls, func(req *http.Request) *http.Response {
		escaped, conditional = req.URL.EscapedPath(), req.Header.Get("If-Match")
		reply := nativePoolWire(203, `{"subnetpool":{"id":null,"project_id":null,"tenant_id":"other","prefixes":{"cidr":"raw"},"revision_number":"٥","maximum_prefix_length":"7","max_prefixlen":"8","shared":1e-4000,"x_actual":9007199254740993,"self":"reserved"}}`)
		reply.Header.Set("If-Match", "actual-header")
		reply.Header.Set("X-Request-Id", "actual")
		return reply
	})
	value, err := api.UpdateRecord(context.Background(), "pool")
	if err != nil || len(calls) != 0 || value.Wire != nil || value.Envelope != nil || value.Header != nil || value.StatusCode != 0 || value.Resource.StatusCode != 0 {
		t.Fatal(value, err, calls)
	}
	poolRecordValue(t, value, "id", `"pool"`)
	poolRecordValue(t, value, "tags", `[]`)
	poolRecordValue(t, value, "default_prefix_length", `null`)
	if _, present := value.Resource.Body["if_match"]; present {
		t.Fatal(value)
	}
	value, err = api.UpdateRecord(context.Background(), "folder/pool?%", subnetpools.WithRecordAttribute("default_prefix_length", "20"), subnetpools.WithRecordAttribute("default_prefixlen", "24"), subnetpools.WithRecordAttribute("default_quota", false), subnetpools.WithRecordAttribute("if_revision", 0), subnetpools.WithRecordAttribute("if_match", "body-only"), subnetpools.WithRecordAttribute("x_seed", map[string]any{"null": nil, "false": false, "zero": 0, "empty": []any{}}))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].query != "" || escaped != poolRecordsPath+"/folder%2Fpool%3F%25" || conditional != "" {
		t.Fatal(value, err, calls, escaped, conditional)
	}
	var body map[string]map[string]json.RawMessage
	if err := json.Unmarshal([]byte(calls[0].body), &body); err != nil {
		t.Fatal(err)
	}
	fields := body["subnetpool"]
	if string(fields["default_prefixlen"]) != `"24"` || string(fields["default_quota"]) != `false` || string(fields["if_revision"]) != `0` || string(fields["if_match"]) != `"body-only"` || fields["is_default"] != nil || fields["shared"] != nil || fields["default_prefix_length"] != nil {
		t.Fatal(fields)
	}
	for key, want := range map[string]string{"id": "null", "project_id": "null", "tenant_id": `"other"`, "default_prefix_length": "24", "default_quota": "false", "minimum_prefix_length": "null", "maximum_prefix_length": "8", "prefixes": `[{"cidr":"raw"}]`, "revision_number": "5", "is_shared": "false", "is_default": "null", "tags": "[]", "if_match": `"body-only"`, "if_revision": "0", "x_actual": "9007199254740993", "x_seed": `{"empty":[],"false":false,"null":null,"zero":0}`} {
		poolRecordValue(t, value, key, want)
	}
	if value.Resource.Body["self"] != nil || value.Wire.Body["self"] == nil || value.Wire.Body["x_seed"] != nil || value.Wire.Body["default_prefixlen"] != nil || string(value.Wire.Body["max_prefixlen"]) != `"8"` || value.Header.Get("If-Match") != "actual-header" || value.StatusCode != 203 {
		t.Fatal(value)
	}
	value.Resource.Body["x_actual"][0] = 'X'
	value.Header.Set("X-Request-Id", "changed")
	if string(value.Wire.Body["x_actual"]) != "9007199254740993" || value.Wire.Header.Get("X-Request-Id") != "actual" || value.Resource.Header.Get("X-Request-Id") != "actual" {
		t.Fatal(value)
	}
}

func TestPythonSubnetPoolRecordsMemberDescriptorsSparseAndMalformedEvidence(t *testing.T) {
	for _, sample := range []struct {
		body  string
		valid bool
	}{
		{"", true}, {"{bad", true}, {`{}`, true}, {`{"subnetpool":{}}`, true}, {`null`, false}, {`[]`, false}, {`false`, false}, {`{"subnetpool":null}`, false}, {`{"subnetpool":[]}`, false}, {"{\"subnetpool\":{\"name\":\"\xff\"}}", false},
	} {
		for _, update := range []bool{false, true} {
			t.Run(fmt.Sprintf("%q/update=%t", sample.body, update), func(t *testing.T) {
				var calls []nativePoolCall
				api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { return nativePoolWire(204, sample.body) })
				var value *subnetpools.SubnetPoolRecord
				var err error
				if update {
					value, err = api.UpdateRecord(context.Background(), "seed", subnetpools.WithRecordAttribute("x_seed", false))
				} else {
					value, err = api.FindRecord(context.Background(), "seed", subnetpools.WithRecordFindQuery("x_seed", false))
				}
				if (err == nil) != sample.valid || value == nil || value.StatusCode != 204 || string(value.Envelope) != sample.body || len(calls) != 1 {
					t.Fatal(sample, value, err, calls)
				}
				if sample.valid {
					poolRecordValue(t, value, "id", `"seed"`)
					poolRecordValue(t, value, "x_seed", `false`)
					poolRecordValue(t, value, "tags", `[]`)
					poolRecordValue(t, value, "minimum_prefix_length", `null`)
				} else {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != 204 || string(proof.Body) != sample.body {
						t.Fatal(err, proof)
					}
				}
			})
		}
	}
	var calls []nativePoolCall
	api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response {
		return nativePoolWire(299, `{"subnetpool":{"id":null,"name":"observed","default_prefixlen":1.9,"default_quota":"-4","ip_version":false,"max_prefixlen":"١٢","min_prefixlen":9007199254740993,"prefixes":false,"tags":null,"shared":"false","is_default":1e-4000,"tenant_id":{"raw":true},"created_at":"raw-time","if_match":"unknown-body"}}`)
	})
	value, err := api.FindRecord(context.Background(), "literal/id?", subnetpools.WithRecordFindQuery("name", "seed-name"))
	if err != nil || len(calls) != 1 || value.StatusCode != 299 {
		t.Fatal(value, err, calls)
	}
	for key, want := range map[string]string{"id": "null", "name": `"observed"`, "default_prefix_length": "1", "default_quota": "0", "ip_version": "false", "maximum_prefix_length": "12", "minimum_prefix_length": "9007199254740993", "prefixes": "[false]", "tags": "null", "is_shared": "true", "is_default": "false", "project_id": `{"raw":true}`, "created_at": `"raw-time"`, "if_match": `"unknown-body"`} {
		poolRecordValue(t, value, key, want)
	}
	if string(value.Wire.Body["default_prefixlen"]) != "1.9" || string(value.Wire.Body["min_prefixlen"]) != "9007199254740993" {
		t.Fatal(value)
	}
}

func TestPythonSubnetPoolRecordsQueryClassificationAliasesAndLocalDescriptors(t *testing.T) {
	var calls []nativePoolCall
	var accepts []string
	api, _ := nativePoolAPI(t, &calls, func(req *http.Request) *http.Response {
		accepts = append(accepts, req.Header.Get("Accept"))
		if req.URL.Path != poolRecordsPath {
			return nativePoolWire(404, `{}`)
		}
		return nativePoolWire(200, `{"subnetpools":[{"id":"wanted","name":"wanted","default_quota":"١","tenant_id":null,"prefixes":{"cidr":true},"shared":false,"x_row":null}]}`)
	})
	api.RawClient().MoreHeaders = map[string]string{"Accept": "application/source"}
	value, err := api.FindRecord(context.Background(), "wanted", subnetpools.WithRecordFindQuery("shared", false), subnetpools.WithRecordFindQuery("is_shared", nil), subnetpools.WithRecordFindQuery("default_quota", true), subnetpools.WithRecordFindQuery("prefixes", []any{map[string]any{"cidr": true}}), subnetpools.WithRecordFindQuery("tenant_id", nil), subnetpools.WithRecordFindQuery("project_id", "fixed"), subnetpools.WithRecordFindQuery("any_tags", []any{"one", nil, "two"}), subnetpools.WithRecordFindQuery("fields", []any{"maximum_prefix_length", nil, "id"}), subnetpools.WithRecordFindQuery("x_unknown", map[string]any{"key": false}), subnetpools.WithRecordFindQuery("all_projects", true))
	if err != nil || value == nil || len(calls) != 2 || !reflect.DeepEqual(accepts, []string{"application/source", "application/json"}) {
		t.Fatal(value, err, calls, accepts)
	}
	member, _ := url.ParseQuery(calls[0].query)
	list, _ := url.ParseQuery(calls[1].query)
	if member.Get("shared") != "False" || member.Has("is_shared") || member.Get("default_quota") != "True" || member.Get("x_unknown") != "key" || member.Has("all_projects") || list.Has("shared") || list.Has("default_quota") || list.Has("prefixes") || list.Has("tenant_id") || list.Has("x_unknown") || list.Has("all_projects") || list.Get("project_id") != "fixed" || list.Get("name") != "wanted" || !reflect.DeepEqual(list["tags-any"], []string{"one", "two"}) || !reflect.DeepEqual(list["fields"], []string{"maximum_prefix_length", "id"}) {
		t.Fatal(member, list)
	}
	poolRecordValue(t, value, "default_quota", "1")
	poolRecordValue(t, value, "minimum_prefix_length", "null")
	calls = nil
	accepts = nil
	values, err := api.AllRecords(context.Background(), subnetpools.WithRecordListQuery("id", "wanted"), subnetpools.WithRecordListQuery("default_quota", true), subnetpools.WithRecordListQuery("default_prefixlen", 999), subnetpools.WithRecordListQuery("x_unknown", poolRecordMarshaler(func() ([]byte, error) { return nil, errors.New("discarded unknown") })), subnetpools.WithRecordListOpts(subnetpools.RecordListOpts{Query: map[string]json.RawMessage{"x_raw": json.RawMessage(`invalid`)}}), subnetpools.WithRecordListHeaders(map[string]string{"accept": "application/explicit"}))
	// WithRecordListOpts replaces preceding options, and unknown list values are
	// discarded without invoking their marshaler or becoming server filters.
	if err != nil || len(values) != 1 || len(calls) != 1 || accepts[0] != "application/explicit" || calls[0].query != "" {
		t.Fatal(values, err, calls, accepts)
	}
	calls = nil
	values, err = api.AllRecords(context.Background(), subnetpools.WithRecordListQuery("id", "wanted"), subnetpools.WithRecordListQuery("default_quota", true), subnetpools.WithRecordListQuery("default_prefixlen", 999))
	if err != nil || len(values) != 1 || calls[0].query != "" {
		t.Fatal(values, err, calls)
	}
}

type poolRecordMarshaler func() ([]byte, error)

func (m poolRecordMarshaler) MarshalJSON() ([]byte, error) { return m() }

func TestPythonSubnetPoolRecordsPaginationFullSearchAndRawCaps(t *testing.T) {
	for _, nextKind := range []string{"body", "dictionary", "header", "marker"} {
		t.Run(nextKind, func(t *testing.T) {
			var calls []nativePoolCall
			api, cloud := nativePoolAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Query().Get("marker") != "" {
					return nativePoolWire(200, `{"subnetpools":[{"id":"b","name":"wanted"}]}`)
				}
				next := "/neutron/v2.0/subnetpools?marker=a&name=wanted"
				body := `{"subnetpools":[{"id":"a","name":"wanted","x_actual":false}]`
				switch nextKind {
				case "body":
					body += `,"subnetpools_links":[{"rel":"next","href":"` + next + `"}]`
				case "dictionary":
					body += `,"links":{"next":"` + next + `"}`
				}
				response := nativePoolWire(206, body+`}`)
				if nextKind == "header" {
					response.Header.Set("Link", "<"+next+">; rel=next")
				}
				response.Header.Set("X-Request-Id", "first-page")
				return response
			})
			_ = cloud
			opts := []subnetpools.RecordListOption{subnetpools.WithRecordListQuery("name", "wanted")}
			if nextKind == "marker" {
				opts = append(opts, subnetpools.WithRecordListLimit(2))
			}
			values, err := api.AllRecords(context.Background(), opts...)
			// Marker continuation keeps fetching short pages until an empty page.
			if nextKind == "marker" {
				// The test server deliberately repeats marker b; the pager rejects a
				// cycle rather than looping or silently claiming full enumeration.
				if err == nil || values != nil || len(calls) != 3 {
					t.Fatal(values, err, calls)
				}
			} else {
				if err != nil || len(values) != 2 || len(calls) != 2 || values[0].StatusCode != 206 || values[0].Header.Get("X-Request-Id") != "first-page" || string(values[0].Wire.Body["x_actual"]) != "false" {
					t.Fatal(values, err, calls)
				}
				poolRecordValue(t, values[0], "tags", "[]")
				if !strings.Contains(string(values[0].Envelope), `"subnetpools"`) || urlMustQuery(t, calls[1].query).Get("name") != "wanted" {
					t.Fatal(values, calls)
				}
			}
		})
	}
	for _, late := range []string{"duplicate", "503", "malformed"} {
		t.Run("late_"+late, func(t *testing.T) {
			var calls []nativePoolCall
			api, _ := nativePoolAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Path != poolRecordsPath {
					return nativePoolWire(404, `{"missing":true}`)
				}
				if req.URL.Query().Get("marker") == "" {
					return nativePoolWire(200, `{"subnetpools":[{"id":"a","name":"wanted"}],"subnetpools_links":[{"rel":"next","href":"?marker=a"}]}`)
				}
				switch late {
				case "duplicate":
					return nativePoolWire(200, `{"subnetpools":[{"id":"b","name":"wanted"}]}`)
				case "503":
					return nativePoolWire(503, `{"late":true}`)
				default:
					return nativePoolWire(200, `{"subnetpools":[false]}`)
				}
			})
			value, err := api.FindRecord(context.Background(), "wanted")
			if err == nil || value != nil || len(calls) != 3 {
				t.Fatal(value, err, calls)
			}
			if late == "duplicate" && !errors.Is(err, resource.ErrAmbiguous) {
				t.Fatal(err)
			}
			if late == "503" {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"late":true}` {
					t.Fatal(err, native)
				}
			}
			if late == "malformed" {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != `{"subnetpools":[false]}` {
					t.Fatal(err, proof)
				}
			}
		})
	}
	var calls []nativePoolCall
	api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response {
		return nativePoolWire(200, `{"subnetpools":[{"id":"discard","name":"other"},{"id":"wanted","name":"wanted"}],"subnetpools_links":[{"rel":"next","href":"?marker=wanted"}]}`)
	})
	values, err := api.AllRecords(context.Background(), subnetpools.WithRecordListQuery("id", "wanted"), subnetpools.WithRecordListMaxItems(1))
	if err != nil || len(values) != 0 || len(calls) != 1 || urlMustQuery(t, calls[0].query).Get("limit") != "1" {
		t.Fatal(values, err, calls)
	}
	calls = nil
	values, err = api.AllRecords(context.Background(), subnetpools.WithRecordListPaginated(false))
	if err != nil || len(values) != 2 || len(calls) != 1 {
		t.Fatal(values, err, calls)
	}
	calls = nil
	seq := api.ListRecords(context.Background())
	if len(calls) != 0 {
		t.Fatal("iterator is not lazy", calls)
	}
	for value, err := range seq {
		if value == nil || err != nil {
			t.Fatal(value, err)
		}
		break
	}
	if len(calls) != 1 {
		t.Fatal(calls)
	}
}
func urlMustQuery(t *testing.T, raw string) url.Values {
	t.Helper()
	v, err := url.ParseQuery(raw)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestPythonSubnetPoolRecordsCleanMissingAndConstructorFailures(t *testing.T) {
	for _, status := range []int{400, 403, 404, 401, 409, 429, 500, 503, 599} {
		for _, strict := range []bool{false, true} {
			t.Run(fmt.Sprintf("status=%d/strict=%t", status, strict), func(t *testing.T) {
				var calls []nativePoolCall
				api, _ := nativePoolAPI(t, &calls, func(req *http.Request) *http.Response {
					if req.URL.Path != poolRecordsPath {
						return nativePoolWire(status, `{"rejected":"member"}`)
					}
					return nativePoolWire(200, `{"subnetpools":[]}`)
				})
				value, err := api.FindRecord(context.Background(), "absent", subnetpools.WithRecordFindIgnoreMissing(!strict))
				clean := status == 400 || status == 403 || status == 404
				if value != nil {
					t.Fatal(value)
				}
				if clean {
					if len(calls) != 2 || (err == nil) == strict {
						t.Fatal(value, err, calls)
					}
					if strict {
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.Is(err, resource.ErrNotFound) || errors.As(err, &native) {
							t.Fatal(err, native)
						}
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if len(calls) != 1 || !errors.As(err, &native) || native.Actual != status || string(native.Body) != `{"rejected":"member"}` {
						t.Fatal(value, err, calls, native)
					}
				}
			})
		}
	}
	for _, row := range []string{`{"cls":"x"}`, `{"connection":null}`, `{"microversion":null}`, `{"_synchronized":false}`, `{"default_quota":"²"}`, `false`, `null`, `[]`} {
		t.Run("row_"+row, func(t *testing.T) {
			var calls []nativePoolCall
			body := `{"subnetpools":[` + row + `]}`
			api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response {
				response := nativePoolWire(207, body)
				response.Header.Set("X-Request-Id", "row-proof")
				return response
			})
			values, err := api.AllRecords(context.Background())
			var proof *resource.ResponseError
			if values != nil || !errors.As(err, &proof) || proof.StatusCode != 207 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "row-proof" || len(calls) != 1 {
				t.Fatal(values, err, proof, calls)
			}
		})
	}
	for _, body := range []string{`{"subnetpools":{"id":"one","name":"one"}}`, `{"subnetpools":[{"id":"one","self":"dropped"}]}`} {
		var calls []nativePoolCall
		api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { return nativePoolWire(200, body) })
		values, err := api.AllRecords(context.Background())
		if err != nil || len(values) != 1 || len(calls) != 1 || values[0].Resource.Body["self"] != nil {
			t.Fatal(values, err, calls)
		}
	}
	for _, body := range []string{`{}`, `{"subnetpools":null}`} {
		var calls []nativePoolCall
		api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { return nativePoolWire(200, body) })
		values, err := api.AllRecords(context.Background())
		var proof *resource.ResponseError
		if values != nil || !errors.As(err, &proof) || string(proof.Body) != body || len(calls) != 1 {
			t.Fatal(values, err, calls)
		}
	}
	var calls []nativePoolCall
	api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response {
		return nativePoolWire(200, `{"subnetpools":[{"id":"one","description":"truthy"}]}`)
	})
	_, err := api.AllRecords(context.Background(), subnetpools.WithRecordListQuery("description", map[string]any{"nested": 1}))
	// description is a server query, so this dict is not a local mapping filter.
	if err != nil {
		t.Fatal(err)
	}
	calls = nil
	_, err = api.AllRecords(context.Background(), subnetpools.WithRecordListQuery("tenant_id", map[string]any{"nested": 1}))
	if err != nil {
		t.Fatal("missing/falsy tenant must skip nested mapping", err)
	}
}

type poolRecordPhysicalBody struct {
	body              []byte
	readErr, closeErr error
	onRead, onClose   func()
	read, closed      int
}

func (b *poolRecordPhysicalBody) Read(p []byte) (int, error) {
	if b.read > 0 {
		return 0, io.EOF
	}
	b.read++
	if b.onRead != nil {
		b.onRead()
	}
	n := copy(p, b.body)
	if b.readErr != nil {
		return n, b.readErr
	}
	return n, io.EOF
}
func (b *poolRecordPhysicalBody) Close() error {
	b.closed++
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

func TestPythonSubnetPoolRecordsPhysicalFailuresPreserveAcceptedAndRejectedProof(t *testing.T) {
	for _, operation := range []string{"UpdateRecord", "FindRecord", "ListRecords", "fallback_list"} {
		for _, status := range []int{200, 299, 400, 403, 404, 503, 599} {
			for _, lane := range []string{"read_close", "cancel", "source"} {
				t.Run(fmt.Sprintf("%s/%d/%s", operation, status, lane), func(t *testing.T) {
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					readCause, closeCause, cancelCause := errors.New("physical read"), errors.New("physical close"), errors.New("physical cancel")
					var calls []nativePoolCall
					var physical *poolRecordPhysicalBody
					api, cloud := nativePoolAPI(t, &calls, func(req *http.Request) *http.Response {
						if operation == "fallback_list" && req.URL.Path != poolRecordsPath {
							return nativePoolWire(404, `{"missing":true}`)
						}
						response := nativePoolWire(status, "")
						response.Body = physical
						response.Header.Set("X-Request-Id", "physical-proof")
						return response
					})
					client := api.RawClient()
					original := client.ResourceBase
					physical = &poolRecordPhysicalBody{body: []byte(`{"actual":"physical"}`)}
					switch lane {
					case "read_close":
						physical.readErr, physical.closeErr = readCause, closeCause
					case "cancel":
						physical.onRead = func() { cancel(cancelCause) }
					case "source":
						physical.onRead = func() { client.ResourceBase = cloud.Server.URL + "/changed/" }
						physical.onClose = func() { client.ResourceBase = original }
					}
					var value *subnetpools.SubnetPoolRecord
					var err error
					switch operation {
					case "UpdateRecord":
						value, err = api.UpdateRecord(ctx, "pool", subnetpools.WithRecordAttribute("name", "new"))
					case "FindRecord", "fallback_list":
						value, err = api.FindRecord(ctx, "pool")
					case "ListRecords":
						_, err = api.AllRecords(ctx)
					}
					if err == nil || physical.closed != 1 || len(calls) != 1+boolInt(operation == "fallback_list") {
						t.Fatal(value, err, calls, physical.closed)
					}
					switch lane {
					case "read_close":
						if !errors.Is(err, readCause) || !errors.Is(err, closeCause) {
							t.Fatal(err)
						}
					case "cancel":
						if !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) {
							t.Fatal(err)
						}
					case "source":
						if !errors.Is(err, resource.ErrInvalidOption) || client.ResourceBase != original {
							t.Fatal(err)
						}
					}
					if status < 400 {
						var proof *resource.ResponseError
						if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != string(physical.body) || proof.Header.Get("X-Request-Id") != "physical-proof" {
							t.Fatal(err, proof)
						}
						if operation == "UpdateRecord" || operation == "FindRecord" {
							if value == nil || value.Resource != nil || value.StatusCode != status || string(value.Envelope) != string(physical.body) {
								t.Fatal(value)
							}
						}
					} else {
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(err, &native) || native.Actual != status || string(native.Body) != string(physical.body) || native.ResponseHeader.Get("X-Request-Id") != "physical-proof" || value != nil {
							t.Fatal(value, err, native)
						}
					}
				})
			}
		}
	}
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func TestPythonSubnetPoolRecordsRetriesKeepProtocolBodyAndNativeCauses(t *testing.T) {
	changes := []struct {
		name, version string
		change        func(*gophercloud.RequestOpts)
		bodyOnly      bool
	}{
		{"selected_change", "2.0", func(o *gophercloud.RequestOpts) { o.MoreHeaders["OpenStack-API-Version"] = "network other" }, false},
		{"selected_delete", "2.0", func(o *gophercloud.RequestOpts) { delete(o.MoreHeaders, "OpenStack-API-Version") }, false},
		{"selected_alias", "2.0", func(o *gophercloud.RequestOpts) { o.MoreHeaders["openstack-api-version"] = "network other" }, false},
		{"selected_omit", "2.0", func(o *gophercloud.RequestOpts) { o.OmitHeaders = []string{"oPeNsTaCk-ApI-vErSiOn"} }, false},
		{"absent_inject", "", func(o *gophercloud.RequestOpts) { o.MoreHeaders["OpenStack-API-Version"] = "network 2.0" }, false},
		{"foreign_version", "", func(o *gophercloud.RequestOpts) { o.MoreHeaders["X-OpenStack-Nova-API-Version"] = "2.99" }, false},
		{"authorization", "", func(o *gophercloud.RequestOpts) { o.MoreHeaders["Authorization"] = "foreign" }, false},
		{"framing", "2.0", func(o *gophercloud.RequestOpts) { o.MoreHeaders["Transfer-Encoding"] = "chunked" }, false},
		{"content_type", "", func(o *gophercloud.RequestOpts) { o.MoreHeaders["Content-Type"] = "application/octet-stream" }, true},
		{"content_omit", "2.0", func(o *gophercloud.RequestOpts) { o.OmitHeaders = []string{"content-type"} }, true},
		{"body_mutation", "", func(o *gophercloud.RequestOpts) { o.JSONBody = json.RawMessage(`{"changed":true}`) }, false},
	}
	for _, operation := range []string{"UpdateRecord", "FindRecord", "ListRecords", "fallback_list"} {
		for _, sample := range changes {
			if sample.bodyOnly && operation != "UpdateRecord" {
				continue
			}
			for _, callbackFailure := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/callback=%t", operation, sample.name, callbackFailure), func(t *testing.T) {
					var calls []nativePoolCall
					api, _ := nativePoolAPI(t, &calls, func(req *http.Request) *http.Response {
						if operation == "fallback_list" && req.URL.Path != poolRecordsPath {
							return nativePoolWire(404, `{}`)
						}
						want := ""
						if sample.version != "" {
							want = "network " + sample.version
						}
						if req.Header.Get("OpenStack-API-Version") != want {
							t.Error(req.Header, want)
						}
						response := nativePoolWire(503, `{"actual":"retry rejected"}`)
						response.Header.Set("X-Request-Id", "retry-proof")
						return response
					})
					client := api.RawClient()
					client.Microversion = sample.version
					callbacks := 0
					cause := errors.New("retry callback cause")
					client.RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, original error, _ uint) error {
						var native gophercloud.ErrUnexpectedResponseCode
						// Preserve the clean initial find rejection before testing list retry.
						if operation == "fallback_list" && errors.As(original, &native) && native.Actual == 404 {
							return original
						}
						callbacks++
						sample.change(o)
						if callbackFailure {
							return cause
						}
						return nil
					}
					var err error
					switch operation {
					case "UpdateRecord":
						_, err = api.UpdateRecord(context.Background(), "pool", subnetpools.WithRecordAttribute("name", "new"))
					case "FindRecord", "fallback_list":
						_, err = api.FindRecord(context.Background(), "pool")
					case "ListRecords":
						_, err = api.AllRecords(context.Background())
					}
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"actual":"retry rejected"}` || native.ResponseHeader.Get("X-Request-Id") != "retry-proof" || callbacks != 1 || len(calls) != 1+boolInt(operation == "fallback_list") || client.Microversion != sample.version {
						t.Fatal(err, native, callbacks, calls)
					}
					if callbackFailure && !errors.Is(err, cause) {
						t.Fatal(err)
					}
				})
			}
		}
	}
	for _, operation := range []string{"UpdateRecord", "FindRecord", "ListRecords"} {
		for _, reauth := range []bool{false, true} {
			t.Run(fmt.Sprintf("ordinary/%s/reauth=%t", operation, reauth), func(t *testing.T) {
				var calls []nativePoolCall
				var tokens []string
				api, cloud := nativePoolAPI(t, &calls, func(req *http.Request) *http.Response {
					tokens = append(tokens, req.Header.Get("X-Auth-Token"))
					if len(calls) == 1 {
						if reauth {
							return nativePoolWire(401, `{"expired":true}`)
						}
						return nativePoolWire(503, `{"retry":true}`)
					}
					if !reauth && req.Header.Get("X-Retry") != "ordinary" {
						t.Error(req.Header)
					}
					if operation == "ListRecords" {
						return nativePoolWire(200, `{"subnetpools":[{"id":"pool"}]}`)
					}
					return nativePoolWire(200, `{"subnetpool":{"id":"pool"}}`)
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
				var err error
				switch operation {
				case "UpdateRecord":
					_, err = api.UpdateRecord(context.Background(), "pool", subnetpools.WithRecordAttribute("name", "new"))
				case "FindRecord":
					_, err = api.FindRecord(context.Background(), "pool")
				case "ListRecords":
					_, err = api.AllRecords(context.Background())
				}
				if err != nil || len(calls) != 2 || calls[0] != calls[1] || !reflect.DeepEqual(tokens, []string{"test-token", "refreshed"}) {
					t.Fatal(err, calls, tokens)
				}
			})
		}
	}
}

func TestPythonSubnetPoolRecordsOptionsSourceContextAndConcurrentOwnership(t *testing.T) {
	input := map[string]any{"items": []any{nil, false, 0}}
	headers := map[string]string{"X-Record": "owned"}
	field, header := subnetpools.WithRecordAttribute("x_input", input), subnetpools.WithRecordMutationHeaders(headers)
	input["items"] = nil
	headers["X-Record"] = "changed"
	var calls []nativePoolCall
	var retained *subnetpools.RecordMutationOpts
	api, _ := nativePoolAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.Header.Get("X-Record") != "owned" {
			t.Error(req.Header)
		}
		retained.Fields["x_input"][0] = 'X'
		retained.Headers["X-Record"] = "retained mutation"
		return nativePoolWire(200, `{"subnetpool":{"id":"pool"}}`)
	})
	value, err := api.UpdateRecord(context.Background(), "pool", field, header, func(o *subnetpools.RecordMutationOpts) error { retained = o; return nil })
	if err != nil || len(calls) != 1 || calls[0].body != `{"subnetpool":{"x_input":{"items":[null,false,0]}}}` {
		t.Fatal(value, err, calls)
	}
	poolRecordValue(t, value, "x_input", `{"items":[null,false,0]}`)
	for _, lane := range []string{"nil_context", "cancel_callback", "source_callback", "callback_cause", "nil_option", "bad_id", "protected_header", "bad_descriptor", "marshal_cause", "reserved_body"} {
		t.Run(lane, func(t *testing.T) {
			var calls []nativePoolCall
			api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { return nativePoolWire(200, `{}`) })
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			var actual context.Context = ctx
			id := "pool"
			cause := errors.New("option cause")
			options := []subnetpools.RecordMutationOption{subnetpools.WithRecordAttribute("name", "new")}
			switch lane {
			case "nil_context":
				actual = nil
			case "cancel_callback":
				options = append(options, func(*subnetpools.RecordMutationOpts) error { cancel(cause); return nil })
			case "source_callback":
				options = append(options, func(*subnetpools.RecordMutationOpts) error { api.RawClient().Type = "compute"; return cause })
			case "callback_cause":
				options = append(options, func(*subnetpools.RecordMutationOpts) error { return cause })
			case "nil_option":
				options = append(options, nil)
			case "bad_id":
				id = " \n"
			case "protected_header":
				options = append(options, subnetpools.WithRecordMutationHeaders(map[string]string{"Authorization": "foreign"}))
			case "bad_descriptor":
				options = append(options, subnetpools.WithRecordAttribute("default_quota", "²"))
			case "marshal_cause":
				options = append(options, subnetpools.WithRecordAttribute("name", poolRecordMarshaler(func() ([]byte, error) { return nil, cause })))
			case "reserved_body":
				options = append(options, subnetpools.WithRecordAttribute("id", "changed"))
			}
			value, err := api.UpdateRecord(actual, id, options...)
			if err == nil || value != nil || len(calls) != 0 {
				t.Fatal(value, err, calls)
			}
			if (lane == "cancel_callback" || lane == "source_callback" || lane == "callback_cause" || lane == "marshal_cause") && !errors.Is(err, cause) {
				t.Fatal(err)
			}
		})
	}
	for _, key := range []string{"id", "self", "cls", "session", "name_or_id", "ignore_missing", "connection", "_synchronized", "microversion", "headers", "max_items", "paginated"} {
		t.Run("find_control_"+key, func(t *testing.T) {
			var calls []nativePoolCall
			api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { return nativePoolWire(200, `{}`) })
			value, err := api.FindRecord(context.Background(), "pool", subnetpools.WithRecordFindQuery(key, nil))
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
				t.Fatal(value, err, calls)
			}
		})
	}
	for _, opts := range [][]subnetpools.RecordListOption{{nil}, {subnetpools.WithRecordListLimit(-1)}, {subnetpools.WithRecordListMaxItems(-1)}, {subnetpools.WithRecordListQuery("cls", nil)}, {subnetpools.WithRecordListHeaders(map[string]string{"Cookie": "foreign"})}} {
		var calls []nativePoolCall
		api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response { return nativePoolWire(200, `{}`) })
		_, err := api.AllRecords(context.Background(), opts...)
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(err, calls)
		}
	}
	var concurrentCalls atomic.Int32
	var unused []nativePoolCall
	api, cloud := nativePoolAPI(t, &unused, func(*http.Request) *http.Response { panic("replaced transport") })
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		concurrentCalls.Add(1)
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		return nativePoolWire(200, string(body)), nil
	}))
	option := subnetpools.WithRecordAttribute("x_shared", []any{false, 0, nil})
	values := make([]*subnetpools.SubnetPoolRecord, 8)
	failures := make([]error, 8)
	var workers sync.WaitGroup
	for i := range values {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			values[i], failures[i] = api.UpdateRecord(context.Background(), fmt.Sprint(i), option)
		}(i)
	}
	workers.Wait()
	if concurrentCalls.Load() != 8 {
		t.Fatal(concurrentCalls.Load())
	}
	for i, value := range values {
		if failures[i] != nil {
			t.Fatal(failures[i])
		}
		poolRecordValue(t, value, "id", fmt.Sprintf(`"%d"`, i))
		poolRecordValue(t, value, "x_shared", `[false,0,null]`)
		value.Resource.Body["x_shared"][0] = 'X'
		if string(value.Wire.Body["x_shared"]) != `[false,0,null]` || !strings.Contains(string(value.Envelope), `[false,0,null]`) {
			t.Fatal(value)
		}
	}
}

func TestPythonSubnetPoolRecordsFindMicroversionIsMemberPhaseOnly(t *testing.T) {
	for _, selected := range []string{"", "2.0"} {
		t.Run("selected_"+selected, func(t *testing.T) {
			var calls []nativePoolCall
			var versions []string
			api, _ := nativePoolAPI(t, &calls, func(req *http.Request) *http.Response {
				versions = append(versions, req.Header.Get("OpenStack-API-Version"))
				if req.Header.Get("X-Record") != "both-phases" {
					t.Error(req.Header)
				}
				if req.URL.Path != poolRecordsPath {
					return nativePoolWire(404, `{"missing":true}`)
				}
				if req.URL.Query().Get("marker") == "" {
					return nativePoolWire(200, `{"subnetpools":[{"id":"other","name":"other"}],"subnetpools_links":[{"rel":"next","href":"?marker=other"}]}`)
				}
				return nativePoolWire(200, `{"subnetpools":[{"id":"found","name":"wanted"}]}`)
			})
			api.RawClient().Microversion = selected
			value, err := api.FindRecord(context.Background(), "wanted", subnetpools.WithRecordFindMicroversion("2.9"), subnetpools.WithRecordFindHeaders(map[string]string{"X-Record": "both-phases"}))
			listVersion := ""
			if selected != "" {
				listVersion = "network " + selected
			}
			if err != nil || value == nil || len(calls) != 3 || !reflect.DeepEqual(versions, []string{"network 2.9", listVersion, listVersion}) || api.RawClient().Microversion != selected {
				t.Fatal(value, err, calls, versions)
			}
			poolRecordValue(t, value, "id", `"found"`)
		})
		t.Run("fallback_retry_"+selected, func(t *testing.T) {
			var calls []nativePoolCall
			api, _ := nativePoolAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Path != poolRecordsPath {
					return nativePoolWire(404, `{}`)
				}
				want := ""
				if selected != "" {
					want = "network " + selected
				}
				if req.Header.Get("OpenStack-API-Version") != want {
					t.Error(req.Header, want)
				}
				response := nativePoolWire(503, `{"phase":"original LIST"}`)
				response.Header.Set("X-Request-Id", "fallback-native")
				return response
			})
			api.RawClient().Microversion = selected
			callbacks := 0
			api.RawClient().RetryFunc = func(_ context.Context, _, _ string, o *gophercloud.RequestOpts, original error, _ uint) error {
				var native gophercloud.ErrUnexpectedResponseCode
				if errors.As(original, &native) && native.Actual == 404 {
					return original
				}
				callbacks++
				o.MoreHeaders["OpenStack-API-Version"] = "network 2.9"
				return nil
			}
			value, err := api.FindRecord(context.Background(), "wanted", subnetpools.WithRecordFindMicroversion("2.9"))
			var native gophercloud.ErrUnexpectedResponseCode
			if value != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &native) || native.Actual != 503 || string(native.Body) != `{"phase":"original LIST"}` || native.ResponseHeader.Get("X-Request-Id") != "fallback-native" || callbacks != 1 || len(calls) != 2 {
				t.Fatal(value, err, native, calls, callbacks)
			}
		})
	}
	// Direct LIST and UPDATE overrides still apply to their own operation.
	for _, operation := range []string{"list", "update"} {
		var calls []nativePoolCall
		api, _ := nativePoolAPI(t, &calls, func(req *http.Request) *http.Response {
			if req.Header.Get("OpenStack-API-Version") != "network 2.9" {
				t.Error(req.Header)
			}
			if operation == "list" {
				return nativePoolWire(200, `{"subnetpools":[]}`)
			}
			return nativePoolWire(200, `{}`)
		})
		api.RawClient().Microversion = "2.0"
		var err error
		if operation == "list" {
			_, err = api.AllRecords(context.Background(), subnetpools.WithRecordListMicroversion("2.9"))
		} else {
			_, err = api.UpdateRecord(context.Background(), "pool", subnetpools.WithRecordMutationMicroversion("2.9"), subnetpools.WithRecordAttribute("name", "new"))
		}
		if err != nil || len(calls) != 1 || api.RawClient().Microversion != "2.0" {
			t.Fatal(err, calls)
		}
	}
}

func TestPythonSubnetPoolRecordsNextUsesBinary64TruthinessAndWholePageProof(t *testing.T) {
	for _, next := range []string{`1e-400`, `-1e-400`, `0.0`, `1e400`, `1.0`, `false`, `null`} {
		t.Run(next, func(t *testing.T) {
			body := `{"subnetpools":[{"id":"one","x_actual":false}],"next":` + next + `}`
			var calls []nativePoolCall
			api, _ := nativePoolAPI(t, &calls, func(*http.Request) *http.Response {
				reply := nativePoolWire(207, body)
				reply.Header.Set("X-Request-Id", "next-physical")
				return reply
			})
			var rows []*subnetpools.SubnetPoolRecord
			var failure error
			for value, err := range api.ListRecords(context.Background()) {
				if err != nil {
					failure = err
					break
				}
				rows = append(rows, value)
			}
			wantError := next == `1e400` || next == `1.0`
			if (failure != nil) != wantError || len(rows) != 1 || len(calls) != 1 || string(rows[0].Envelope) != body || rows[0].StatusCode != 207 || rows[0].Header.Get("X-Request-Id") != "next-physical" {
				t.Fatal(rows, failure, calls)
			}
			if wantError {
				var proof *resource.ResponseError
				if !errors.As(failure, &proof) || proof.StatusCode != 207 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "next-physical" {
					t.Fatal(failure, proof)
				}
			}
		})
	}
}
