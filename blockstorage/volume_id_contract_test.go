package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestGetVolumeIDRetainsFoundResourceAndArbitraryActualResponseID(t *testing.T) {
	for _, rawID := range []string{"missing", "null", "false", "0", `""`, `[]`, `{}`, `[false,9007199254740993]`, `{"opaque":9007199254740993}`, "9007199254740993", "1e-9999", `"actual-response-ID"`} {
		t.Run(rawID, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			row := `{}`
			wantID := rawID
			if rawID == "missing" {
				wantID = "null"
			} else {
				row = `{"id":` + rawID + `}`
			}
			body := `{"volume":` + row + `}`
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				volumeReadContractWire(t, r, volumeReadContractBase+"volumes/requested", "test-token")
				if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				calls.Add(1)
				w.Header().Set("X-Proof", "member")
				testcloud.JSON(w, 200, body)
			})
			result, err := blockstorage.GetVolumeID(context.Background(), client, blockstorage.GetVolumeIDRequest{NameOrID: "requested"})
			if err != nil || result == nil || result.ID == nil || string(result.ID) != wantID || result.Volume == nil || result.Observed == nil || len(result.Pages) != 0 || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			fields := volumeReadContractFields(t, result.Value)
			if string(fields["id"]) != wantID || len(fields) != 38 || result.Volume.StatusCode != 200 || string(result.Observed.Body) != body {
				t.Fatal("normalized/raw/member projection changed", result)
			}
			if rawID == "missing" {
				if _, present := result.Volume.Body["id"]; present {
					t.Fatal("request ID fabricated in raw evidence")
				}
			} else if string(result.Volume.Body["id"]) != rawID {
				t.Fatal("untyped ID coerced in wire evidence", result)
			}
			result.ID[0] = '!'
			if string(volumeReadContractFields(t, result.Value)["id"]) != wantID || string(result.Observed.Body) != body || (rawID != "missing" && string(result.Volume.Body["id"]) != rawID) {
				t.Fatal("ID aliases logical view or wire proof", result)
			}
			result.Observed.Body[0] = '!'
			result.Observed.Header.Set("X-Proof", "caller")
			if result.Volume.Header.Get("X-Proof") != "member" || !json.Valid(result.Value) {
				t.Fatal("member proof aliases logical/raw results", result)
			}
		})
	}
}

func TestGetVolumeIDResolvesActualListIDAfterCompatibleMemberRejection(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls atomic.Int32
			first := volumeReadContractPage(`[{"id":"not-it","name":"other"}]`, "?marker=next")
			second := volumeReadContractPage(`[{"id":false,"name":"wanted","unknown":9007199254740993}]`, "")
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					volumeReadContractWire(t, r, volumeReadContractBase+"volumes/wanted", "test-token")
					testcloud.JSON(w, code, `{"error":"member rejected"}`)
				case 2:
					volumeReadContractWire(t, r, volumeReadContractPath, "test-token")
					if r.URL.Query().Get("name") != "wanted" || len(r.URL.Query()) != 1 {
						t.Error(r.URL)
					}
					w.Header().Set("X-Proof", "first")
					testcloud.JSON(w, 200, first)
				case 3:
					volumeReadContractWire(t, r, volumeReadContractPath, "test-token")
					if r.URL.RawQuery != "marker=next" {
						t.Error(r.URL)
					}
					w.Header().Set("X-Proof", "found")
					testcloud.JSON(w, 200, second)
				default:
					t.Error("shortcut, restart or per-row GET", r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := blockstorage.GetVolumeID(context.Background(), client, blockstorage.GetVolumeIDRequest{NameOrID: "wanted"})
			if err != nil || result == nil || string(result.ID) != "false" || result.Volume == nil || result.Observed != nil || len(result.Pages) != 2 || calls.Load() != 3 || string(result.Volume.Body["unknown"]) != "9007199254740993" || result.Volume.Header.Get("X-Proof") != "found" {
				t.Fatal(result, err, calls.Load())
			}
			result.ID[0] = '!'
			result.Pages[1].Body[0] = '!'
			result.Pages[1].Header.Set("X-Proof", "caller")
			if string(result.Volume.Body["id"]) != "false" || string(volumeReadContractFields(t, result.Value)["id"]) != "false" || result.Volume.Header.Get("X-Proof") != "found" || string(result.Pages[0].Body) != first {
				t.Fatal("fallback result proofs are not owned", result)
			}
		})
	}
}

func TestGetVolumeIDExactNamesAndCompletedAbsenceNeverBecomeIDShortcutsOrGlob(t *testing.T) {
	for _, name := range []string{"00000000-0000-0000-0000-000000000000", "disk*", "unsafe/name", "percent%2Fname", "space\u00a0name"} {
		t.Run(name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var members, lists atomic.Int32
			encoded, _ := json.Marshal(name)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != volumeReadContractPath {
					members.Add(1)
					testcloud.JSON(w, 404, `{"error":"missing"}`)
					return
				}
				lists.Add(1)
				if r.URL.Query().Get("name") != name || len(r.URL.Query()) != 1 {
					t.Error("name became route or another filter", r.URL)
				}
				rows := `[{"id":"different","name":"disk-other"}]`
				if name != "disk*" && name != "00000000-0000-0000-0000-000000000000" {
					rows = `[{"id":"resolved","name":` + string(encoded) + `}]`
				}
				testcloud.JSON(w, 200, volumeReadContractPage(rows, ""))
			})
			result, err := blockstorage.GetVolumeID(context.Background(), client, blockstorage.GetVolumeIDRequest{NameOrID: name})
			wantMember := int32(0)
			absent := name == "disk*" || name == "00000000-0000-0000-0000-000000000000"
			if absent {
				wantMember = 1
			}
			if err != nil || result == nil || result.Observed != nil || len(result.Pages) != 1 || members.Load() != wantMember || lists.Load() != 1 {
				t.Fatal(result, err, members.Load(), lists.Load())
			}
			if absent {
				if result.ID != nil || result.Value != nil || result.Volume != nil {
					t.Fatal("absent UUID/wildcard returned fabricated ID", result)
				}
			} else if string(result.ID) != `"resolved"` || result.Volume == nil {
				t.Fatal(result)
			}
		})
	}
}

func TestGetVolumeIDAmbiguityPrecedesUnusedBadTailAndContinuation(t *testing.T) {
	cloud := testcloud.New(t)
	client := volumeReadContractClient(cloud)
	var calls atomic.Int32
	body := volumeReadContractPage(`[{"id":"one","name":"unsafe/name"},{"id":"two","name":"unsafe/name"},{"id":"bad","bootable":"invalid"},false]`, "https://must-not-follow.invalid/volumes/detail")
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, body) })
	result, err := blockstorage.GetVolumeID(context.Background(), client, blockstorage.GetVolumeIDRequest{NameOrID: "unsafe/name"})
	var accepted *resource.ResponseError
	var ambiguous *resource.AmbiguousError
	if result == nil || result.ID != nil || result.Value != nil || result.Volume != nil || result.Observed != nil || len(result.Pages) != 1 || string(result.Pages[0].Body) != body || calls.Load() != 1 || !errors.Is(err, resource.ErrAmbiguous) || !errors.As(err, &ambiguous) || !reflect.DeepEqual(ambiguous.IDs, []string{"one", "two"}) || errors.As(err, &accepted) {
		t.Fatal(result, err, calls.Load())
	}
	volumeReadContractOperation(t, err, "GetVolumeID")
}

func TestGetVolumeIDDistinguishesPreparationFailureFromExecutedEmptyProofFailure(t *testing.T) {
	for _, kind := range []string{"nil context", "canceled context", "nil client", "wrong service", "nil option", "callback rejection", "callback cancellation", "empty identity", "control identity", "invalid UTF8 identity"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("caller preflight cause")
			input := blockstorage.GetVolumeIDRequest{NameOrID: "target"}
			var calls, originals, later atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			first := blockstorage.VolumeReadOption(func(*blockstorage.VolumeReadOpts) error {
				originals.Add(1)
				if kind == "callback rejection" {
					return cause
				}
				if kind == "callback cancellation" {
					cancel(cause)
				}
				return nil
			})
			switch kind {
			case "nil context":
				ctx = nil
			case "canceled context":
				cancel(cause)
			case "nil client":
				client = nil
			case "wrong service":
				client.Type = "compute"
			case "nil option":
				first = nil
			case "empty identity":
				input.NameOrID = ""
			case "control identity":
				input.NameOrID = "disk\x00name"
			case "invalid UTF8 identity":
				input.NameOrID = string([]byte{'i', 0xff})
			}
			result, err := blockstorage.GetVolumeID(ctx, client, input, first, func(*blockstorage.VolumeReadOpts) error { later.Add(1); return nil })
			executed := kind == "empty identity" || kind == "control identity" || kind == "invalid UTF8 identity"
			if err == nil || calls.Load() != 0 || (result != nil) != executed {
				t.Fatal(result, err, calls.Load())
			}
			if executed {
				if result.ID != nil || result.Value != nil || result.Volume != nil || result.Observed != nil || len(result.Pages) != 0 || originals.Load() != 1 || later.Load() != 1 || !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(result, err, originals.Load(), later.Load())
				}
			} else {
				if later.Load() != 0 {
					t.Fatal("option after terminal preflight executed", later.Load())
				}
				wantOriginal := int32(0)
				if kind == "callback rejection" || kind == "callback cancellation" {
					wantOriginal = 1
				}
				if originals.Load() != wantOriginal {
					t.Fatal(originals.Load(), wantOriginal)
				}
				if kind == "wrong service" {
					if !errors.Is(err, resource.ErrUnsupported) {
						t.Fatal(err)
					}
				} else if kind == "callback rejection" {
					if !errors.Is(err, cause) {
						t.Fatal(err)
					}
				} else if kind == "canceled context" || kind == "callback cancellation" {
					if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			}
			volumeReadContractOperation(t, err, "GetVolumeID")
		})
	}
}

func TestGetVolumeIDOwnsOriginalOptionSliceLocationAndHeaders(t *testing.T) {
	cloud := testcloud.New(t)
	client := volumeReadContractClient(cloud)
	cloudName := "cloud-entry"
	location := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`)}}
	factory := blockstorage.WithVolumeReadLocation(location)
	cloudName = "caller-mutated"
	location.Project.ID[1] = '!'
	var callbacks, calls atomic.Int32
	var retained *blockstorage.VolumeReadOpts
	var options []blockstorage.VolumeReadOption
	options = []blockstorage.VolumeReadOption{factory, func(next *blockstorage.VolumeReadOpts) error {
		callbacks.Add(1)
		retained = next
		options[2] = func(*blockstorage.VolumeReadOpts) error { return errors.New("replaced callback must not run") }
		client.MoreHeaders["x-source"] = "after-capture"
		cloud.Provider.SetToken("callback-live")
		return nil
	}, func(next *blockstorage.VolumeReadOpts) error {
		callbacks.Add(1)
		*retained.Location.Cloud = "retained-mutated"
		retained.Location.Project.ID[1] = '!'
		if *next.Location.Cloud != "cloud-entry" || string(next.Location.Project.ID) != `"scope"` {
			t.Fatal("retained option changed prepared next", next)
		}
		return nil
	}}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		volumeReadContractWire(t, r, volumeReadContractBase+"volumes/target", "callback-live")
		testcloud.JSON(w, 200, `{"volume":{"id":"wire","availability_zone":"az"}}`)
	})
	result, err := blockstorage.GetVolumeID(context.Background(), client, blockstorage.GetVolumeIDRequest{NameOrID: "target"}, options...)
	if err != nil || result == nil || string(result.ID) != `"wire"` || calls.Load() != 1 || callbacks.Load() != 2 {
		t.Fatal(result, err, calls.Load(), callbacks.Load())
	}
	fields := volumeReadContractFields(t, result.Value)
	computed := volumeReadContractFields(t, fields["location"])
	project := volumeReadContractFields(t, computed["project"])
	if string(computed["cloud"]) != `"cloud-entry"` || string(computed["zone"]) != `"az"` || string(project["id"]) != `"scope"` {
		t.Fatal("location factory or retained config not owned", string(result.Value))
	}
}

func TestGetVolumeIDOriginalSourceGuardStopsRestoringOptionBeforeHTTP(t *testing.T) {
	for _, fact := range []string{"provider", "endpoint", "resource base", "type", "microversion"} {
		t.Run(fact, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			original := *client
			var calls, restore atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			result, err := blockstorage.GetVolumeID(context.Background(), client, blockstorage.GetVolumeIDRequest{NameOrID: "target"}, func(*blockstorage.VolumeReadOpts) error {
				switch fact {
				case "provider":
					client.ProviderClient = &gophercloud.ProviderClient{}
				case "endpoint":
					client.Endpoint = cloud.Server.URL + "/changed/"
				case "resource base":
					client.ResourceBase = cloud.Server.URL + "/changed/"
				case "type":
					client.Type = "volume"
				case "microversion":
					client.Microversion = "3.61"
				}
				return nil
			}, func(*blockstorage.VolumeReadOpts) error { restore.Add(1); *client = original; return nil })
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 || restore.Load() != 0 {
				t.Fatal(result, err, calls.Load(), restore.Load())
			}
			volumeReadContractOperation(t, err, "GetVolumeID")
		})
	}
}
