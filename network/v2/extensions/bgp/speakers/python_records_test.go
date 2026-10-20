package speakers_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"io"
	"net/http"
	"testing"
)

func bgpReadField(t *testing.T, row *speakers.SpeakerReadRecord, key, want string) {
	t.Helper()
	if row == nil || row.Resource == nil || string(row.Resource.Body[key]) != want {
		t.Fatalf("%s want %s: %+v", key, want, row)
	}
}
func TestBGPSpeakerListOwnsAliasesUntypedFieldsAndPhysicalEvidence(t *testing.T) {
	body := `{"bgp_speakers":[{"id":"one","tenant_id":"tenant","ip_version":"4","advertise_tenant_networks":"false","advertise_floating_ip_host_routes":null,"local_as":9007199254740993,"networks":"net","x_extension":{"n":9007199254740993}},{"id":"two","project_id":null,"tenant_id":"fallback"}]}`
	api, _, calls, _ := pythonBGPSpeakerAPI(t, func(r *http.Request) *http.Response {
		if r.URL.Path != "/neutron/v2.0/bgp-speakers" || r.URL.RawQuery != "" {
			t.Fatal(r.URL)
		}
		response := pythonBGPSpeakerWire(203, body)
		response.Header.Set("X-Record-Proof", "actual")
		return response
	})
	seq := api.ListRecords(context.Background())
	if len(*calls) != 0 {
		t.Fatal(*calls)
	}
	var rows []*speakers.SpeakerReadRecord
	for row, err := range seq {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if len(rows) != 2 || len(*calls) != 1 {
		t.Fatal(rows, *calls)
	}
	bgpReadField(t, rows[0], "project_id", `"tenant"`)
	bgpReadField(t, rows[1], "project_id", "null")
	for k, v := range map[string]string{"ip_version": `"4"`, "advertise_tenant_networks": `"false"`, "advertise_floating_ip_host_routes": "null", "local_as": "9007199254740993", "networks": `["net"]`} {
		bgpReadField(t, rows[0], k, v)
	}
	bgpReadField(t, rows[0], "x_extension", `{"n":9007199254740993}`)
	if rows[0].StatusCode != 203 || string(rows[0].Envelope) != body || rows[0].Header.Get("X-Record-Proof") != "actual" || rows[0].Resource.Body["tags"] != nil || rows[0].Resource.Body["revision_number"] != nil {
		t.Fatal(rows)
	}
	rows[0].Resource.Body["x_extension"][0] = '['
	rows[0].Header.Set("X-Record-Proof", "caller")
	if string(rows[0].Wire.Body["x_extension"]) != `{"n":9007199254740993}` || rows[0].Wire.Header.Get("X-Record-Proof") != "actual" {
		t.Fatal(rows)
	}
}
func TestBGPSpeakerListUsesLocalBodyFiltersAndCompleteMarkerPages(t *testing.T) {
	api, _, calls, _ := pythonBGPSpeakerAPI(t, func(r *http.Request) *http.Response {
		if r.URL.Query().Get("limit") != "1" || r.URL.Query().Has("name") || r.URL.Query().Has("project_id") {
			t.Fatal(r.URL)
		}
		switch r.URL.Query().Get("marker") {
		case "":
			return pythonBGPSpeakerWire(200, `{"bgp_speakers":[{"id":"skip","name":"wrong","tenant_id":"tenant"}]}`)
		case "skip":
			return pythonBGPSpeakerWire(200, `{"bgp_speakers":[{"id":"kept","name":"wanted","tenant_id":"tenant"}]}`)
		default:
			return pythonBGPSpeakerWire(200, `{"bgp_speakers":[]}`)
		}
	})
	rows, err := api.AllRecords(context.Background(), speakers.WithRecordListLimit(1), speakers.WithRecordListQuery("name", "wanted"), speakers.WithRecordListQuery("project_id", "tenant"), speakers.WithRecordListQuery("unknown", make(chan int)))
	if err != nil || len(rows) != 1 || len(*calls) != 3 {
		t.Fatal(rows, err, *calls)
	}
	bgpReadField(t, rows[0], "id", `"kept"`)
}
func TestBGPSpeakerFindGETSeedAndFilteredFallbackKeepProfilesSeparate(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		t.Run(map[bool]string{false: "GET", true: "fallback"}[fallback], func(t *testing.T) {
			api, _, calls, _ := pythonBGPSpeakerAPI(t, func(r *http.Request) *http.Response {
				if r.URL.Path == "/neutron/v2.0/bgp-speakers/wanted" || r.URL.Path == "/neutron/v2.0/bgp-speakers/7" {
					if r.URL.Query().Get("name") != "wanted" || r.URL.Query().Get("x_seed") != "seed" {
						t.Fatal(r.URL)
					}
					if fallback {
						return pythonBGPSpeakerWire(404, `{}`)
					}
					return pythonBGPSpeakerWire(200, `{"bgp_speaker":{"tenant_id":"tenant"}}`)
				}
				if r.URL.Query().Has("name") || r.URL.Query().Has("x_seed") {
					t.Fatal(r.URL)
				}
				return pythonBGPSpeakerWire(200, `{"bgp_speakers":[{"id":"actual","name":"wanted"}]}`)
			})
			row, err := api.FindRecord(context.Background(), "wanted", speakers.WithRecordFindQuery("name", "wanted"), speakers.WithRecordFindQuery("x_seed", "seed"))
			expectedCalls := 1
			if fallback {
				expectedCalls = 2
			}
			if err != nil || row == nil || len(*calls) != expectedCalls {
				t.Fatal(row, err, *calls)
			}
			if fallback {
				bgpReadField(t, row, "id", `"actual"`)
				if row.Resource.Body["x_seed"] != nil {
					t.Fatal(row.Resource)
				}
			} else {
				bgpReadField(t, row, "id", `"wanted"`)
				bgpReadField(t, row, "x_seed", `"seed"`)
				bgpReadField(t, row, "project_id", `"tenant"`)
			}
		})
	}
}
func TestBGPSpeakerFindCompletesDuplicateMissingAndLiteralIDSearch(t *testing.T) {
	for _, mode := range []string{"missing", "strict", "duplicate", "numeric id"} {
		t.Run(mode, func(t *testing.T) {
			api, _, calls, _ := pythonBGPSpeakerAPI(t, func(r *http.Request) *http.Response {
				if r.URL.Path == "/neutron/v2.0/bgp-speakers/wanted" || r.URL.Path == "/neutron/v2.0/bgp-speakers/7" {
					return pythonBGPSpeakerWire(403, `{}`)
				}
				if mode == "duplicate" {
					return pythonBGPSpeakerWire(200, `{"bgp_speakers":[{"id":"one","name":"wanted"},{"id":"one","name":"wanted"}]}`)
				}
				if mode == "numeric id" {
					return pythonBGPSpeakerWire(200, `{"bgp_speakers":[{"id":7,"name":"other"}]}`)
				}
				return pythonBGPSpeakerWire(200, `{"bgp_speakers":[]}`)
			})
			identity := "wanted"
			if mode == "numeric id" {
				identity = "7"
			}
			row, err := api.FindRecord(context.Background(), identity, speakers.WithRecordFindIgnoreMissing(mode != "strict"))
			if row != nil || len(*calls) != 2 {
				t.Fatal(row, err, *calls)
			}
			switch mode {
			case "strict":
				if !errors.Is(err, resource.ErrNotFound) {
					t.Fatal(err)
				}
			case "duplicate":
				if !errors.Is(err, resource.ErrAmbiguous) {
					t.Fatal(err)
				}
			default:
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
func TestBGPSpeakerFindFetchControlsKeepUnknownConstructorSeed(t *testing.T) {
	api, _, calls, _ := pythonBGPSpeakerAPI(t, func(r *http.Request) *http.Response {
		if r.URL.Path != "/neutron/v2.0/custom" || r.URL.Query().Get("headers") != "X-Raw" || r.Header.Get("X-Raw") != "" {
			t.Fatal(r.URL, r.Header)
		}
		return pythonBGPSpeakerWire(200, `{"selected":{"id":null,"tenant_id":"tenant"}}`)
	})
	row, err := api.FindRecord(context.Background(), "wanted", speakers.WithRecordFindQuery("requires_id", false), speakers.WithRecordFindQuery("base_path", "/custom"), speakers.WithRecordFindQuery("resource_response_key", "selected"), speakers.WithRecordFindQuery("headers", json.RawMessage(`{"X-Raw":7}`)))
	if err != nil || row == nil || len(*calls) != 1 {
		t.Fatal(row, err, *calls)
	}
	bgpReadField(t, row, "id", "null")
	bgpReadField(t, row, "requires_id", "false")
	bgpReadField(t, row, "headers", `{"X-Raw":7}`)
	bgpReadField(t, row, "project_id", `"tenant"`)
}
func TestBGPSpeakerProxyJMESOwnsUnconsumedGenerator(t *testing.T) {
	for _, expr := range []string{"[*].name", "@", "length(@)"} {
		t.Run(expr, func(t *testing.T) {
			api, _, calls, _ := pythonBGPSpeakerAPI(t, func(r *http.Request) *http.Response {
				return pythonBGPSpeakerWire(200, `{"bgp_speakers":[{"id":"one"}]}`)
			})
			result, err := api.ProxyListRecords(context.Background(), speakers.WithRecordListJMESPath(expr))
			if len(*calls) != 0 {
				t.Fatal(*calls)
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
			if expr == "[*].name" {
				if result.Value != nil || string(result.JSON) != "null" {
					t.Fatal(result)
				}
				return
			}
			generator, ok := result.Value.(*speakers.RecordListGenerator)
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
			if n != 1 || len(*calls) != 1 {
				t.Fatal(n, *calls)
			}
		})
	}
}

type bgpFaultBody struct {
	body        io.ReadCloser
	read, close error
}

func (b *bgpFaultBody) Read(p []byte) (int, error) {
	n, err := b.body.Read(p)
	if n > 0 && b.read != nil {
		return n, b.read
	}
	return n, err
}
func (b *bgpFaultBody) Close() error { return errors.Join(b.body.Close(), b.close) }
func TestBGPSpeakerFindPhysicalFaultsAndPreflightAreTerminal(t *testing.T) {
	for _, mode := range []string{"read", "close", "cancel", "nil option"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New("BGP physical " + mode)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			api, _, calls, _ := pythonBGPSpeakerAPI(t, func(r *http.Request) *http.Response {
				response := pythonBGPSpeakerWire(404, `{"error":"physical"}`)
				fault := &bgpFaultBody{body: response.Body}
				if mode == "read" {
					fault.read = cause
				}
				if mode == "close" {
					fault.close = cause
				}
				response.Body = fault
				return response
			})
			opts := []speakers.RecordFindOption{}
			if mode == "nil option" {
				opts = append(opts, nil)
			}
			if mode == "cancel" {
				cancel(cause)
			}
			_, err := api.FindRecord(ctx, "wanted", opts...)
			if err == nil {
				t.Fatal("fault suppressed")
			}
			if mode == "read" || mode == "close" {
				if !errors.Is(err, cause) || len(*calls) != 1 {
					t.Fatal(err, *calls)
				}
			} else if len(*calls) != 0 {
				t.Fatal(err, *calls)
			}
		})
	}
}
