package volumetypes_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonTypeFindRecordGetFirstUsesLowercaseNone(t *testing.T) {
	api, calls := pythonTypeAPI(t, func(*http.Request) *http.Response { return pythonTypeWire(200, `{"volume_type":`+pythonTypeRow+`}`) })
	value, err := api.FindRecord(context.Background(), "vt-1")
	if err != nil || value == nil || string(value.Resource.Body["name"]) != `"fast"` || string(value.Resource.Body["is_public"]) != "false" || calls.Len() != 1 || pythonTypeCalls(calls)[0].query != "is_public=none" {
		t.Fatal(value, err, pythonTypeCalls(calls))
	}
}
func TestPythonTypeFindRecordFallsBackWithoutNameFilterAndMatchesID(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			api, calls := pythonTypeAPI(t, func(req *http.Request) *http.Response {
				if req.URL.Path != pythonTypeBase {
					return pythonTypeWire(code, `{"error":"missing"}`)
				}
				return pythonTypeWire(200, `{"volume_types":[{"id":"wanted","name":"other","os-volume-type-access:is_public":false,"extra_specs":{"large":9007199254740993}}]}`)
			})
			value, err := api.FindRecord(context.Background(), "wanted")
			if err != nil || value == nil || string(value.Resource.Body["id"]) != `"wanted"` || calls.Len() != 2 || pythonTypeCalls(calls)[1].query != "is_public=none" {
				t.Fatal(value, err, pythonTypeCalls(calls))
			}
			if value.Header.Get("Content-Type") == "" || string(value.Resource.Body["extra_specs"]) != `{"large":9007199254740993}` || string(value.Envelope) == "" {
				t.Fatal(value)
			}
		})
	}
}
func TestPythonTypeFindRecordScansContinuationAndRejectsLateErrorOrDuplicate(t *testing.T) {
	for _, duplicate := range []bool{false, true} {
		t.Run(map[bool]string{false: "late error", true: "duplicate"}[duplicate], func(t *testing.T) {
			api, calls := pythonTypeAPI(t, func(req *http.Request) *http.Response {
				if req.URL.Path != pythonTypeBase {
					return pythonTypeWire(404, `{}`)
				}
				if req.URL.Query().Get("marker") == "next" {
					if duplicate {
						return pythonTypeWire(200, `{"volume_types":[{"id":"vt-2","name":"fast"}]}`)
					}
					return pythonTypeWire(500, `{"error":"late list failure"}`)
				}
				return pythonTypeWire(200, `{"volume_types":[{"id":"vt-1","name":"fast"}],"volume_types_links":[{"rel":"next","href":"?is_public=none&marker=next"}]}`)
			})
			value, err := api.FindRecord(context.Background(), "fast")
			if value != nil || calls.Len() != 3 || duplicate && !errors.Is(err, resource.ErrAmbiguous) || !duplicate && !gophercloud.ResponseCodeIs(err, 500) {
				t.Fatal(value, err, pythonTypeCalls(calls))
			}
		})
	}
}
func TestPythonTypeFindRecordMissingPoliciesAndPreflightGuard(t *testing.T) {
	api, calls := pythonTypeAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Path != pythonTypeBase {
			return pythonTypeWire(404, `{}`)
		}
		return pythonTypeWire(200, `{"volume_types":[]}`)
	})
	value, err := api.FindRecord(context.Background(), "none")
	if value != nil || err != nil {
		t.Fatal(value, err)
	}
	_, err = api.FindRecord(context.Background(), "none", volumetypes.WithRecordFindIgnoreMissing(false))
	if !errors.Is(err, resource.ErrNotFound) || calls.Len() != 4 {
		t.Fatal(err, pythonTypeCalls(calls))
	}
	before := calls.Len()
	_, err = api.FindRecord(context.Background(), "none", func(*volumetypes.RecordFindOpts) error {
		api.RawClient().ResourceBase = "http://changed.invalid/v3/project/"
		return nil
	})
	if !errors.Is(err, resource.ErrInvalidOption) || calls.Len() != before {
		t.Fatal(err, pythonTypeCalls(calls))
	}
}
func TestPythonTypeFindRecordPhysicalRejectedBodyDoesNotFallback(t *testing.T) {
	readFailure, closeFailure := errors.New("find read"), errors.New("find close")
	api, calls := pythonTypeAPI(t, func(*http.Request) *http.Response {
		response := pythonTypeWire(404, "")
		response.Header.Set("X-Request-Id", "type-find-proof")
		response.Body = &testcloud.Body{Data: `{"error":"missing"}`, ReadErr: readFailure, CloseErr: closeFailure}
		return response
	})
	value, err := api.FindRecord(context.Background(), "fast")
	var proof gophercloud.ErrUnexpectedResponseCode
	if value != nil || !errors.Is(err, readFailure) || !errors.Is(err, closeFailure) || !gophercloud.ResponseCodeIs(err, 404) || !errors.As(err, &proof) || proof.Actual != 404 || string(proof.Body) != `{"error":"missing"}` || proof.ResponseHeader.Get("X-Request-Id") != "type-find-proof" || proof.Method != http.MethodGet || !strings.HasSuffix(proof.URL, "/types/fast?is_public=none") || calls.Len() != 1 {
		t.Fatal(value, err, proof, pythonTypeCalls(calls))
	}
}
func TestPythonTypeFindRecordQueryExtensionLocalFilterAndSnapshot(t *testing.T) {
	query := map[string][]string{"sort": {"name:asc"}}
	option := volumetypes.WithRecordFindQuery("sort", query["sort"])
	query["sort"][0] = "changed"
	api, calls := pythonTypeAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Path != pythonTypeBase {
			return pythonTypeWire(404, `{}`)
		}
		return pythonTypeWire(200, `{"volume_types":[{"id":"vt-1","name":"fast","extra_specs":{"nested":{"v":true}}},{"id":"vt-2","name":"fast","extra_specs":{"nested":{"v":false}}}]}`)
	})
	value, err := api.FindRecord(context.Background(), "fast", option, volumetypes.WithRecordFindQuery("extra_specs", map[string]any{"nested": map[string]any{"v": 1}}))
	if err != nil || value == nil || string(value.Resource.Body["id"]) != `"vt-1"` || calls.Len() != 2 {
		t.Fatal(value, err, pythonTypeCalls(calls))
	}
	parsed, _ := url.ParseQuery(pythonTypeCalls(calls)[1].query)
	if !reflect.DeepEqual(parsed, url.Values{"is_public": {"none"}, "sort": {"name:asc"}}) {
		t.Fatal(parsed)
	}
}
