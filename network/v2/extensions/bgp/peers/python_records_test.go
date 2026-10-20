package peers_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"io"
	"net/http"
	"testing"
)

func bgpReadField(t *testing.T, row *peers.PeerReadRecord, key, want string) {
	t.Helper()
	if row == nil || row.Resource == nil || string(row.Resource.Body[key]) != want {
		t.Fatalf("%s want %s: %+v", key, want, row)
	}
}
func TestBGPPeerListOwnsAliasesUntypedFieldsAndPhysicalEvidence(t *testing.T) {
	body := `{"bgp_peers":[{"id":"one","tenant_id":"tenant","auth_type":false,"remote_as":"65001","peer_ip":7,"x_extension":{"n":9007199254740993}},{"id":"two","project_id":null,"tenant_id":"fallback"}]}`
	api, calls, _ := pythonBGPPeerAPI(t, func(r *http.Request) *http.Response {
		if r.URL.Path != "/neutron/v2.0/bgp-peers" || r.URL.RawQuery != "" {
			t.Fatal(r.URL)
		}
		response := pythonBGPPeerWire(203, body)
		response.Header.Set("X-Record-Proof", "actual")
		return response
	})
	seq := api.ListRecords(context.Background())
	if len(*calls) != 0 {
		t.Fatal(*calls)
	}
	var rows []*peers.PeerReadRecord
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
	for k, v := range map[string]string{"auth_type": "false", "remote_as": `"65001"`, "peer_ip": "7"} {
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
func TestBGPPeerListUsesLocalBodyFiltersAndCompleteMarkerPages(t *testing.T) {
	api, calls, _ := pythonBGPPeerAPI(t, func(r *http.Request) *http.Response {
		if r.URL.Query().Get("limit") != "1" || r.URL.Query().Has("name") || r.URL.Query().Has("project_id") {
			t.Fatal(r.URL)
		}
		switch r.URL.Query().Get("marker") {
		case "":
			return pythonBGPPeerWire(200, `{"bgp_peers":[{"id":"skip","name":"wrong","tenant_id":"tenant"}]}`)
		case "skip":
			return pythonBGPPeerWire(200, `{"bgp_peers":[{"id":"kept","name":"wanted","tenant_id":"tenant"}]}`)
		default:
			return pythonBGPPeerWire(200, `{"bgp_peers":[]}`)
		}
	})
	rows, err := api.AllRecords(context.Background(), peers.WithRecordListLimit(1), peers.WithRecordListQuery("name", "wanted"), peers.WithRecordListQuery("project_id", "tenant"), peers.WithRecordListQuery("unknown", make(chan int)))
	if err != nil || len(rows) != 1 || len(*calls) != 3 {
		t.Fatal(rows, err, *calls)
	}
	bgpReadField(t, rows[0], "id", `"kept"`)
}
func TestBGPPeerFindGETSeedAndFilteredFallbackKeepProfilesSeparate(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		t.Run(map[bool]string{false: "GET", true: "fallback"}[fallback], func(t *testing.T) {
			api, calls, _ := pythonBGPPeerAPI(t, func(r *http.Request) *http.Response {
				if r.URL.Path == "/neutron/v2.0/bgp-peers/wanted" || r.URL.Path == "/neutron/v2.0/bgp-peers/7" {
					if r.URL.Query().Get("name") != "wanted" || r.URL.Query().Get("x_seed") != "seed" {
						t.Fatal(r.URL)
					}
					if fallback {
						return pythonBGPPeerWire(404, `{}`)
					}
					return pythonBGPPeerWire(200, `{"bgp_peer":{"tenant_id":"tenant"}}`)
				}
				if r.URL.Query().Has("name") || r.URL.Query().Has("x_seed") {
					t.Fatal(r.URL)
				}
				return pythonBGPPeerWire(200, `{"bgp_peers":[{"id":"actual","name":"wanted"}]}`)
			})
			row, err := api.FindRecord(context.Background(), "wanted", peers.WithRecordFindQuery("name", "wanted"), peers.WithRecordFindQuery("x_seed", "seed"))
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
func TestBGPPeerFindCompletesDuplicateMissingAndLiteralIDSearch(t *testing.T) {
	for _, mode := range []string{"missing", "strict", "duplicate", "numeric id"} {
		t.Run(mode, func(t *testing.T) {
			api, calls, _ := pythonBGPPeerAPI(t, func(r *http.Request) *http.Response {
				if r.URL.Path == "/neutron/v2.0/bgp-peers/wanted" || r.URL.Path == "/neutron/v2.0/bgp-peers/7" {
					return pythonBGPPeerWire(403, `{}`)
				}
				if mode == "duplicate" {
					return pythonBGPPeerWire(200, `{"bgp_peers":[{"id":"one","name":"wanted"},{"id":"one","name":"wanted"}]}`)
				}
				if mode == "numeric id" {
					return pythonBGPPeerWire(200, `{"bgp_peers":[{"id":7,"name":"other"}]}`)
				}
				return pythonBGPPeerWire(200, `{"bgp_peers":[]}`)
			})
			identity := "wanted"
			if mode == "numeric id" {
				identity = "7"
			}
			row, err := api.FindRecord(context.Background(), identity, peers.WithRecordFindIgnoreMissing(mode != "strict"))
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
func TestBGPPeerFindFetchControlsKeepUnknownConstructorSeed(t *testing.T) {
	api, calls, _ := pythonBGPPeerAPI(t, func(r *http.Request) *http.Response {
		if r.URL.Path != "/neutron/v2.0/custom" || r.URL.Query().Get("headers") != "X-Raw" || r.Header.Get("X-Raw") != "" {
			t.Fatal(r.URL, r.Header)
		}
		return pythonBGPPeerWire(200, `{"selected":{"id":null,"tenant_id":"tenant"}}`)
	})
	row, err := api.FindRecord(context.Background(), "wanted", peers.WithRecordFindQuery("requires_id", false), peers.WithRecordFindQuery("base_path", "/custom"), peers.WithRecordFindQuery("resource_response_key", "selected"), peers.WithRecordFindQuery("headers", json.RawMessage(`{"X-Raw":7}`)))
	if err != nil || row == nil || len(*calls) != 1 {
		t.Fatal(row, err, *calls)
	}
	bgpReadField(t, row, "id", "null")
	bgpReadField(t, row, "requires_id", "false")
	bgpReadField(t, row, "headers", `{"X-Raw":7}`)
	bgpReadField(t, row, "project_id", `"tenant"`)
}
func TestBGPPeerProxyJMESOwnsUnconsumedGenerator(t *testing.T) {
	for _, expr := range []string{"[*].name", "@", "length(@)"} {
		t.Run(expr, func(t *testing.T) {
			api, calls, _ := pythonBGPPeerAPI(t, func(r *http.Request) *http.Response { return pythonBGPPeerWire(200, `{"bgp_peers":[{"id":"one"}]}`) })
			result, err := api.ProxyListRecords(context.Background(), peers.WithRecordListJMESPath(expr))
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
			generator, ok := result.Value.(*peers.RecordListGenerator)
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
func TestBGPPeerFindPhysicalFaultsAndPreflightAreTerminal(t *testing.T) {
	for _, mode := range []string{"read", "close", "cancel", "nil option"} {
		t.Run(mode, func(t *testing.T) {
			cause := errors.New("BGP physical " + mode)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			api, calls, _ := pythonBGPPeerAPI(t, func(r *http.Request) *http.Response {
				response := pythonBGPPeerWire(404, `{"error":"physical"}`)
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
			opts := []peers.RecordFindOption{}
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
