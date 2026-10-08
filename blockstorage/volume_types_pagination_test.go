package blockstorage_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestVolumeTypesPaginationRespectsSourcePrecedenceAndLazyFirstMatchingHref(t *testing.T) {
	for _, tc := range []struct{ name, tail, header, want string }{
		{"body links win", `,"links":[{"rel":"next","href":"?marker=body"}],"volume_types_links":false,"next":true,"volume_type_links":false`, `<?marker=http>; rel="next"`, "body"},
		{"plural source links", `,"volume_types_links":[{"rel":"next","href":"?marker=plural"}],"next":true,"volume_type_links":false`, `<?marker=http>; rel="next"`, "plural"},
		{"present empty body suppresses plural", `,"links":[],"volume_types_links":[{"rel":"next","href":"?marker=unused"}],"next":"?marker=top","volume_type_links":false`, `<?marker=http>; rel="next"`, "top"},
		{"first falsey href skips unvisited tail", `,"links":[{"rel":"next","href":null},false,{"rel":"next","href":"?marker=unused"}],"next":"?marker=top"`, "", "top"},
		{"usable HTTP repair last relation", `,"volume_type_links":false`, `<?marker=first>; rel="next", <?marker=http>; rel="NEXT"`, "http"},
		{"native singular extension", `,"volume_type_links":[{"rel":"next","href":"?marker=first"},{"rel":"next","href":"?marker=native"}]`, "", "native"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls atomic.Int32
			first := `{"volume_types":[{"id":"first"}]` + tc.tail + `}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				volumeReadContractWire(t, r, volumeTypeContractPath, "test-token")
				switch calls.Add(1) {
				case 1:
					if r.URL.RawQuery != "" {
						t.Error(r.URL)
					}
					if tc.header != "" {
						w.Header().Set("Link", tc.header)
					}
					testcloud.JSON(w, 200, first)
				case 2:
					if r.URL.Query().Get("marker") != tc.want || len(r.URL.Query()) != 1 {
						t.Error("link precedence changed", r.URL, tc.want)
					}
					testcloud.JSON(w, 200, volumeTypeContractPage(`[{"id":"second"}]`, ""))
				default:
					t.Error("unexpected source branch refresh", r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := blockstorage.ListVolumeTypes(context.Background(), client)
			if err != nil || result == nil || calls.Load() != 2 || len(result.Pages) != 2 || len(result.Types) != 2 {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestGetVolumeTypeContinuationMergesVisibilityDropsPaginationAndBlankAdvertisedValues(t *testing.T) {
	cloud := testcloud.New(t)
	client := volumeReadContractClient(cloud)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		volumeReadContractWire(t, r, volumeTypeContractPath, "test-token")
		query := r.URL.Query()
		switch calls.Add(1) {
		case 1:
			if r.URL.RawQuery != "is_public=none" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, volumeTypeContractPage(`[{"id":"other"}]`, "?marker=a%2Fb&limit=1&is_public=&q=&repeat=first&repeat=&repeat=second"))
		case 2:
			if len(query) != 4 || query.Get("is_public") != "none" || query.Get("marker") != "a/b" || query.Get("limit") != "1" || !reflect.DeepEqual(query["repeat"], []string{"first", "second"}) {
				t.Error("source query merge failed", r.URL)
			}
			testcloud.JSON(w, 200, volumeTypeContractPage(`[{"id":"other2"}]`, "?marker=last&limit=&repeat=third"))
		case 3:
			if len(query) != 3 || query.Get("is_public") != "none" || query.Get("marker") != "last" || query.Has("limit") || !reflect.DeepEqual(query["repeat"], []string{"third"}) {
				t.Error("previous marker/limit or blank values survived", r.URL)
			}
			testcloud.JSON(w, 200, volumeTypeContractPage(`[{"id":"actual","name":"literal / type"}]`, ""))
		default:
			t.Error("unsafe name used direct route or extra search", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.GetVolumeType(context.Background(), client, blockstorage.GetVolumeTypeRequest{NameOrID: "literal / type"})
	if err != nil || result == nil || calls.Load() != 3 || result.Observed != nil || len(result.Pages) != 3 || result.Type == nil || string(result.Type.Body["id"]) != `"actual"` {
		t.Fatal(result, err, calls.Load())
	}
}

func TestVolumeTypesPaginationRejectsConsumedLinksOriginPathFragmentsAndQueryCycles(t *testing.T) {
	for _, kind := range []string{"malformed selected links", "foreign origin", "changed path", "empty fragment", "cycle"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls atomic.Int32
			tail := `,"links":false,"volume_types_links":[{"rel":"next","href":"?marker=unused"}]`
			switch kind {
			case "foreign origin":
				tail = `,"next":"https://foreign.invalid/types?marker=next"`
			case "changed path":
				tail = `,"next":"/other/types?marker=next"`
			case "empty fragment":
				tail = `,"next":"?marker=next#"`
			case "cycle":
				tail = `,"next":"?"`
			}
			body := `{"volume_types":[{"id":"first"}]` + tail + `}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				volumeReadContractWire(t, r, volumeTypeContractPath, "test-token")
				if calls.Add(1) != 1 {
					t.Error("unsafe continuation sent", r.URL)
				}
				w.Header().Set("X-Proof", "links-origin")
				testcloud.JSON(w, 200, body)
			})
			result, err := blockstorage.ListVolumeTypes(context.Background(), client)
			var physical *resource.ResponseError
			if result == nil || result.Value != nil || result.Types != nil || calls.Load() != 1 || len(result.Pages) != 1 || !errors.As(err, &physical) || physical.StatusCode != 200 || string(physical.Body) != body || physical.Header.Get("X-Proof") != "links-origin" {
				t.Fatal(result, err, calls.Load())
			}
			if kind == "cycle" {
				var cycle *resource.PaginationCycleError
				if !errors.As(err, &cycle) {
					t.Fatal(err)
				}
			}
			volumeTypeContractOperation(t, err, "ListVolumeTypes")
		})
	}
}
