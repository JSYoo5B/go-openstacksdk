package qos_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonQoSFindRecordGetFirstConvertsIDAndKeepsUnknownFields(t *testing.T) {
	api, calls := pythonQoSAPI(t, func(*http.Request) *http.Response {
		return pythonQoSWire(200, `{"qos_specs":{"id":9007199254740993,"name":"gold","consumer":false,"specs":{"n":9007199254740993},"vendor":[null,0]}}`)
	})
	value, err := api.FindRecord(context.Background(), "9007199254740993")
	if err != nil || value == nil || string(value.Resource.Body["id"]) != `"9007199254740993"` || string(value.Wire.Body["id"]) != "9007199254740993" || string(value.Resource.Body["consumer"]) != "false" || string(value.Resource.Body["vendor"]) != `[null,0]` || calls.Len() != 1 {
		t.Fatal(value, err, pythonQoSCalls(calls))
	}
	if value.StatusCode != 200 || value.Header.Get("Content-Type") == "" || len(value.Envelope) == 0 {
		t.Fatal(value)
	}
}
func TestPythonQoSFindRecordSplitsGETKwargsAndListServerBodyFilters(t *testing.T) {
	api, calls := pythonQoSAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Path != pythonQoSBase {
			return pythonQoSWire(403, `{}`)
		}
		return pythonQoSWire(200, `{"qos_specs":[{"id":"q-1","name":"gold","consumer":"back-end","specs":{"n":true,"extra":"ok"}},{"id":"q-2","name":"gold","consumer":"front-end","specs":{"n":true}}]}`)
	})
	value, err := api.FindRecord(context.Background(), "gold", qos.WithRecordFindQuery("project_id", []any{"p1", nil, "p2"}), qos.WithRecordFindQuery("consumer", "back-end"), qos.WithRecordFindQuery("specs", map[string]any{"n": 1}), qos.WithRecordFindQuery("host", "controller"), qos.WithRecordFindQuery("unknown", nil))
	if err != nil || value == nil || string(value.Resource.Body["id"]) != `"q-1"` || calls.Len() != 2 {
		t.Fatal(value, err, pythonQoSCalls(calls))
	}
	getQuery, _ := url.ParseQuery(pythonQoSCalls(calls)[0].query)
	listQuery, _ := url.ParseQuery(pythonQoSCalls(calls)[1].query)
	if getQuery.Get("host") != "controller" || getQuery.Get("consumer") != "back-end" || getQuery.Get("specs") != "n" || getQuery.Has("unknown") || !reflect.DeepEqual(getQuery["project_id"], []string{"p1", "p2"}) || !reflect.DeepEqual(listQuery, url.Values{"project_id": {"p1", "p2"}}) {
		t.Fatal(getQuery, listQuery)
	}
}
func TestPythonQoSFindRecordDefaultsMissingAndFallbackCodes(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			api, calls := pythonQoSAPI(t, func(req *http.Request) *http.Response {
				if req.URL.Path != pythonQoSBase {
					return pythonQoSWire(code, `{}`)
				}
				return pythonQoSWire(200, `{"qos_specs":[]}`)
			})
			value, err := api.FindRecord(context.Background(), "none")
			if value != nil || err != nil {
				t.Fatal(value, err)
			}
			_, err = api.FindRecord(context.Background(), "none", qos.WithRecordFindIgnoreMissing(false))
			if !errors.Is(err, resource.ErrNotFound) || calls.Len() != 4 {
				t.Fatal(err, pythonQoSCalls(calls))
			}
		})
	}
	api, calls := pythonQoSAPI(t, func(*http.Request) *http.Response { return pythonQoSWire(500, `{"error":"unavailable"}`) })
	_, err := api.FindRecord(context.Background(), "gold")
	if !gophercloud.ResponseCodeIs(err, 500) || calls.Len() != 1 {
		t.Fatal(err, pythonQoSCalls(calls))
	}
}
func TestPythonQoSFindRecordFindsIDAndRejectsLateDuplicate(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		t.Run(map[bool]string{false: "ID fallback", true: "late duplicate"}[duplicate], func(t *testing.T) {
			api, calls := pythonQoSAPI(t, func(req *http.Request) *http.Response {
				if req.URL.Path != pythonQoSBase {
					return pythonQoSWire(404, `{}`)
				}
				if req.URL.Query().Get("marker") == "next" {
					if duplicate {
						return pythonQoSWire(200, `{"qos_specs":[{"id":"q-2","name":"gold"}]}`)
					}
					return pythonQoSWire(200, `{"qos_specs":[]}`)
				}
				return pythonQoSWire(200, `{"qos_specs":[{"id":"gold","name":"other"}],"qos_specs_links":[{"rel":"next","href":"?marker=next"}]}`)
			})
			value, err := api.FindRecord(context.Background(), "gold")
			if duplicate {
				if value != nil || !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(value, err)
				}
			} else if err != nil || value == nil || string(value.Resource.Body["id"]) != `"gold"` {
				t.Fatal(value, err)
			}
			if calls.Len() != 3 {
				t.Fatal(pythonQoSCalls(calls))
			}
		})
	}
}
func TestPythonQoSFindRecordOwnsOptionsAndCapsBeforeBodyFilter(t *testing.T) {
	query := map[string]json.RawMessage{"consumer": json.RawMessage(`"front-end"`), "max_items": json.RawMessage("1")}
	option := qos.WithRecordFindOpts(qos.RecordFindOpts{Query: query})
	query["consumer"] = json.RawMessage(`"back-end"`)
	api, calls := pythonQoSAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Path != pythonQoSBase {
			return pythonQoSWire(404, `{}`)
		}
		return pythonQoSWire(200, `{"qos_specs":[{"id":"q-1","name":"gold","consumer":"back-end"},{"id":"q-2","name":"gold","consumer":"front-end"}],"qos_specs_links":[{"rel":"next","href":"?marker=q-2&limit=1"}]}`)
	})
	value, err := api.FindRecord(context.Background(), "gold", option)
	if err != nil || value != nil || calls.Len() != 2 {
		t.Fatal(value, err, pythonQoSCalls(calls))
	}
	getQuery, _ := url.ParseQuery(pythonQoSCalls(calls)[0].query)
	listQuery, _ := url.ParseQuery(pythonQoSCalls(calls)[1].query)
	if getQuery.Get("max_items") != "1" || getQuery.Get("consumer") != "front-end" || listQuery.Get("limit") != "1" || listQuery.Has("consumer") {
		t.Fatal(getQuery, listQuery)
	}
}
func TestPythonQoSFindRecordGuardAndQueryFailuresBeforeHTTP(t *testing.T) {
	for _, options := range [][]qos.RecordFindOption{{nil}, {qos.WithRecordFindOpts(qos.RecordFindOpts{Query: map[string]json.RawMessage{"name": json.RawMessage(`"unterminated`)}})}, {qos.WithRecordFindQuery("id", "collision")}} {
		api, calls := pythonQoSAPI(t, func(*http.Request) *http.Response { t.Fatal("preflight reached HTTP"); return nil })
		_, err := api.FindRecord(context.Background(), "gold", options...)
		if !errors.Is(err, resource.ErrInvalidOption) || calls.Len() != 0 {
			t.Fatal(err, pythonQoSCalls(calls))
		}
	}
	api, calls := pythonQoSAPI(t, func(*http.Request) *http.Response { t.Fatal("cancelled reached HTTP"); return nil })
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("find cancelled by user")
	_, err := api.FindRecord(ctx, "gold", func(*qos.RecordFindOpts) error { cancel(cause); return nil })
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls.Len() != 0 {
		t.Fatal(err, pythonQoSCalls(calls))
	}
}

func TestPythonQoSFindRecordPhysicalRejectedBodyAndLateAcceptedCloseKeepEvidence(t *testing.T) {
	for _, rejected := range []bool{true, false} {
		t.Run(map[bool]string{true: "rejected member", false: "accepted list"}[rejected], func(t *testing.T) {
			readFailure, closeFailure := errors.New("qos find read"), errors.New("qos find close")
			api, calls := pythonQoSAPI(t, func(req *http.Request) *http.Response {
				if !rejected && req.URL.Path != pythonQoSBase {
					return pythonQoSWire(404, `{}`)
				}
				status, body := 404, `{"error":"missing"}`
				if !rejected {
					status, body = 200, `{"qos_specs":[{"id":"q-1","name":"gold"}]}`
				}
				response := pythonQoSWire(status, "")
				response.Header.Set("X-Request-Id", "qos-proof")
				response.Body = &testcloud.Body{Data: body, ReadErr: readFailure, CloseErr: closeFailure}
				return response
			})
			value, err := api.FindRecord(context.Background(), "gold")
			wantCalls := 2
			if rejected {
				wantCalls = 1
			}
			if value != nil || !errors.Is(err, readFailure) || !errors.Is(err, closeFailure) || calls.Len() != wantCalls {
				t.Fatal(value, err, pythonQoSCalls(calls))
			}
			if rejected {
				var proof gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &proof) || proof.Actual != 404 || string(proof.Body) != `{"error":"missing"}` || proof.ResponseHeader.Get("X-Request-Id") != "qos-proof" || proof.Method != http.MethodGet || !strings.HasSuffix(proof.URL, "/qos-specs/gold") {
					t.Fatal(err, proof)
				}
			} else {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != `{"qos_specs":[{"id":"q-1","name":"gold"}]}` || proof.Header.Get("X-Request-Id") != "qos-proof" {
					t.Fatal(err, proof)
				}
			}
		})
	}
}
