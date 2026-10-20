package services_test

import (
	"context"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/compute/v2/services"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"net/http"
	"testing"
)

func TestComputeServiceFindRecordsListOnlyMatchesIDBinaryAndHost(t *testing.T) {
	for _, identity := range []string{"7", "nova-compute"} {
		t.Run(identity, func(t *testing.T) {
			api, calls := pythonComputeServiceAPI(t, "2.69", func(r *http.Request) *http.Response {
				if r.URL.Path != pythonComputeServiceBase || r.URL.RawQuery != "" || r.Method != "GET" {
					t.Fatal(r.Method, r.URL)
				}
				return pythonComputeServiceWire(203, `{"services":[{"id":7,"binary":"nova-compute","host":"wrong"},{"id":7,"binary":"nova-compute","host":"wanted","forced_down":"false","zone":"zone","updated_at":"literal","x_unknown":1}]}`)
			})
			row, err := api.FindRecord(context.Background(), identity, services.WithRecordQuery("host", "wanted"), services.WithRecordQuery("binary", "ignored"), services.WithRecordQuery("status", "ignored"))
			if err != nil || row == nil || len(*calls) != 1 || string(row.Resource.Body["name"]) != `"nova-compute"` || string(row.Resource.Body["availability_zone"]) != `"zone"` || string(row.Resource.Body["is_forced_down"]) != "true" || row.Resource.Body["x_unknown"] != nil || string(row.Wire.Body["x_unknown"]) != "1" {
				t.Fatal(row, err, *calls)
			}
		})
	}
}
func TestComputeServiceFindRecordsMissingDuplicateAndLateErrors(t *testing.T) {
	for _, mode := range []string{"ignored", "strict", "duplicate", "late"} {
		t.Run(mode, func(t *testing.T) {
			n := 0
			api, _ := pythonComputeServiceAPI(t, "2.69", func(r *http.Request) *http.Response {
				n++
				if mode == "ignored" || mode == "strict" {
					return pythonComputeServiceWire(200, `{"services":[]}`)
				}
				if n == 1 {
					return pythonComputeServiceWire(200, `{"services":[{"id":"a","binary":"nova-compute"}],"next":"?marker=a"}`)
				}
				if mode == "late" {
					return pythonComputeServiceWire(500, `{"error":"late"}`)
				}
				return pythonComputeServiceWire(200, `{"services":[{"id":"a","binary":"nova-compute"}]}`)
			})
			row, err := api.FindRecord(context.Background(), "nova-compute", services.WithRecordIgnoreMissing(mode != "strict"))
			if row != nil {
				t.Fatal(row, err)
			}
			switch mode {
			case "ignored":
				if err != nil {
					t.Fatal(err)
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
				if err == nil || n != 2 {
					t.Fatal(err, n)
				}
			}
		})
	}
}
func TestComputeServiceFindRecordOwnedOptionsAndFallbackPath(t *testing.T) {
	headers := map[string]string{"X-Owned": "original"}
	option := services.WithRecordHeaders(headers)
	headers["X-Owned"] = "changed"
	api, calls := pythonComputeServiceAPI(t, "2.69", func(r *http.Request) *http.Response {
		if r.URL.Path != "/nova/v2.1/alternative-services" || r.Header.Get("X-Owned") != "original" {
			t.Fatal(r.URL, r.Header)
		}
		return pythonComputeServiceWire(200, `{"services":[{"id":"a","binary":"nova-compute"}]}`)
	})
	row, err := api.FindRecord(context.Background(), "nova-compute", option, services.WithRecordListBasePath("alternative-services"))
	if err != nil || row == nil || len(*calls) != 1 {
		t.Fatal(row, err, *calls)
	}
}
func TestComputeServiceFindRecordGuardAndAcceptedProjectionFailures(t *testing.T) {
	api, calls := pythonComputeServiceAPI(t, "2.69", func(r *http.Request) *http.Response { return pythonComputeServiceWire(203, `{"services":[7]}`) })
	_, err := api.FindRecord(context.Background(), "x", nil)
	if err == nil || len(*calls) != 0 {
		t.Fatal(err, *calls)
	}
	_, err = api.FindRecord(context.Background(), "x")
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != `{"services":[7]}` || len(*calls) != 1 {
		t.Fatal(err, proof, *calls)
	}
	cause := errors.New("service cancelled")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	_, err = api.FindRecord(ctx, "x")
	if !errors.Is(err, cause) || len(*calls) != 1 {
		t.Fatal(err, *calls)
	}
}

func TestComputeServiceFindNullNameDoesNotBecomeEmptyString(t *testing.T) {
	for _, raw := range []string{"null", "false", "7"} {
		t.Run(raw, func(t *testing.T) {
			api, calls := pythonComputeServiceAPI(t, "2.69", func(r *http.Request) *http.Response {
				return pythonComputeServiceWire(200, `{"services":[{"id":null,"name":`+raw+`,"binary":null}]}`)
			})
			row, err := api.FindRecord(context.Background(), "")
			if err != nil || row != nil || len(*calls) != 1 {
				t.Fatal(row, err, *calls)
			}
		})
	}
}
