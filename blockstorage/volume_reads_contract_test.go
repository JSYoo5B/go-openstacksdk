package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const volumeReadContractBase = "/proxy/cinder/v3/read-project/"
const volumeReadContractPath = volumeReadContractBase + "volumes/detail"

func volumeReadContractClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("volumev3", "/catalog/cinder/")
	client.ResourceBase = cloud.Server.URL + volumeReadContractBase
	client.Microversion = "3.60"
	client.MoreHeaders = map[string]string{"x-source": "entry"}
	return client
}

func volumeReadContractWire(t *testing.T, r *http.Request, path, token string) {
	t.Helper()
	if r.Method != http.MethodGet || r.URL.Path != path || r.Header.Get("X-Source") != "entry" || r.Header.Get("X-Auth-Token") != token || r.Header.Get("OpenStack-API-Version") != "volume 3.60" {
		t.Error("fixed bodyless source request changed", r.Method, r.URL, r.Header)
	}
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) != 0 {
			t.Error("unexpected request body", string(body), err)
		}
	}
}

func volumeReadContractPage(rows, next string) string { return getVolumesContractPage(rows, next) }
func volumeReadContractFields(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(string(raw), err)
	}
	return fields
}
func volumeReadContractRows(t *testing.T, raw json.RawMessage) []map[string]json.RawMessage {
	t.Helper()
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(string(raw), err)
	}
	return rows
}
func volumeReadContractOperation(t *testing.T, err error, name string) {
	t.Helper()
	var operation *resource.OperationError
	if !errors.As(err, &operation) || operation.Operation != name || operation.Resource != "volume" || operation.Cause == nil {
		t.Fatal("public operation context lost", err)
	}
}

func TestListVolumesMaterializesAllPagesAndOwnsNormalizedAndWireRows(t *testing.T) {
	cloud := testcloud.New(t)
	client := volumeReadContractClient(cloud)
	var calls atomic.Int32
	first := volumeReadContractPage(`[{"id":"duplicate","name":"first","bootable":"TrUe","unknown":{"number":9007199254740993}}]`, "?marker=next&limit=1")
	second := volumeReadContractPage(`[{"id":"duplicate","name":"second","attachments":{"device":false},"metadata":[["ignored","pair"]]}]`, "")
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		volumeReadContractWire(t, r, volumeReadContractPath, "test-token")
		switch calls.Add(1) {
		case 1:
			if r.URL.RawQuery != "" {
				t.Error("initial list query is not empty", r.URL)
			}
			w.Header().Set("X-Proof", "first")
			testcloud.JSON(w, 200, first)
		case 2:
			if r.URL.RawQuery != "marker=next&limit=1" {
				t.Error("advertised query changed", r.URL)
			}
			w.Header().Set("X-Proof", "second")
			testcloud.JSON(w, 200, second)
		default:
			t.Error("list refresh/member lookup/restart", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.ListVolumes(context.Background(), client)
	if err != nil || result == nil || calls.Load() != 2 || len(result.Pages) != 2 || len(result.Volumes) != 2 || result.Volumes[0] == result.Volumes[1] {
		t.Fatal(result, err, calls.Load())
	}
	rows := volumeReadContractRows(t, result.Value)
	if len(rows) != 2 || len(rows[0]) != 38 || string(rows[0]["name"]) != `"first"` || string(rows[1]["name"]) != `"second"` || string(rows[0]["is_bootable"]) != "true" || string(rows[1]["attachments"]) != `[{"device":false}]` || string(rows[1]["metadata"]) != `{}` {
		t.Fatal(string(result.Value))
	}
	if _, present := rows[0]["unknown"]; present {
		t.Fatal("unknown wire field entered normalized view")
	}
	if string(result.Volumes[0].Body["unknown"]) != `{"number":9007199254740993}` || result.Volumes[0].StatusCode != 200 || result.Volumes[1].Header.Get("X-Proof") != "second" {
		t.Fatal("raw precision/page evidence lost", result)
	}
	result.Volumes[0].Body["id"][0] = '!'
	result.Volumes[0].Header.Set("X-Proof", "raw mutation")
	result.Pages[0].Body[0] = '!'
	result.Pages[1].Header.Set("X-Proof", "page mutation")
	if !json.Valid(result.Value) || string(rows[0]["id"]) != `"duplicate"` || string(result.Volumes[1].Body["id"]) != `"duplicate"` || result.Volumes[1].Header.Get("X-Proof") != "second" || result.Pages[0].Header.Get("X-Proof") != "first" || string(result.Pages[1].Body) != second {
		t.Fatal("result views, pages, or rows alias one another")
	}
}

func TestListVolumesEmptyPageStopsWithoutConsumingUnusedContinuation(t *testing.T) {
	cloud := testcloud.New(t)
	client := volumeReadContractClient(cloud)
	var calls atomic.Int32
	body := `{"volumes":[],"volumes_links":{"rel":"next","href":false}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		volumeReadContractWire(t, r, volumeReadContractPath, "test-token")
		calls.Add(1)
		testcloud.JSON(w, 200, body)
	})
	result, err := blockstorage.ListVolumes(context.Background(), client)
	if err != nil || result == nil || string(result.Value) != "[]" || result.Volumes == nil || len(result.Volumes) != 0 || len(result.Pages) != 1 || string(result.Pages[0].Body) != body || calls.Load() != 1 {
		t.Fatal(result, err, calls.Load())
	}
}

func TestListVolumesEagerDescriptorFailureStopsBeforeNextAndDiscardsPartialValues(t *testing.T) {
	for _, bad := range []string{`{"id":"bad","bootable":"not-a-bool"}`, `false`, `{"id":"bad","size":"²"}`} {
		t.Run(bad, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls atomic.Int32
			body := volumeReadContractPage(`[{"id":"good"},`+bad+`]`, "?marker=must-not-fetch")
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Proof", "failed-origin")
				testcloud.JSON(w, 200, body)
			})
			result, err := blockstorage.ListVolumes(context.Background(), client)
			var accepted *resource.ResponseError
			if result == nil || result.Value != nil || result.Volumes != nil || len(result.Pages) != 1 || calls.Load() != 1 || !errors.As(err, &accepted) || accepted.StatusCode != 200 || string(accepted.Body) != body || accepted.Header.Get("X-Proof") != "failed-origin" {
				t.Fatal(result, err, calls.Load())
			}
			volumeReadContractOperation(t, err, "ListVolumes")
			accepted.Body[0] = '!'
			accepted.Header.Set("X-Proof", "error mutation")
			if string(result.Pages[0].Body) != body || result.Pages[0].Header.Get("X-Proof") != "failed-origin" {
				t.Fatal("accepted error borrows page memory")
			}
		})
	}
}

func TestGetVolumeByIDUsesOnlyLiteralMemberAndKeepsResponseIdentity(t *testing.T) {
	for _, row := range []string{`{}`, `{"id":"different","name":"wire","unknown":9007199254740993}`, `{"id":false,"name":[1]}`} {
		t.Run(row, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls atomic.Int32
			body := `{"volume":` + row + `}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				volumeReadContractWire(t, r, volumeReadContractBase+"volumes/volume-ID", "test-token")
				if r.URL.RawQuery != "" {
					t.Error("member query leaked", r.URL)
				}
				calls.Add(1)
				w.Header().Set("X-Proof", "member")
				testcloud.JSON(w, 200, body)
			})
			result, err := blockstorage.GetVolumeByID(context.Background(), client, blockstorage.GetVolumeByIDRequest{ID: "volume-ID"})
			if err != nil || result == nil || result.Volume == nil || result.Observed == nil || len(result.Pages) != 0 || calls.Load() != 1 || result.Volume.StatusCode != 200 || string(result.Observed.Body) != body {
				t.Fatal(result, err, calls.Load())
			}
			fields := volumeReadContractFields(t, result.Value)
			if row == `{}` {
				if string(fields["id"]) != "null" {
					t.Fatal("request ID fabricated in normalized body", string(result.Value))
				}
				if _, exists := result.Volume.Body["id"]; exists {
					t.Fatal("request ID fabricated in wire body")
				}
			} else if row == `{"id":false,"name":[1]}` {
				if string(fields["id"]) != "false" || string(fields["name"]) != "[1]" || string(result.Volume.Body["id"]) != "false" {
					t.Fatal("passive untyped response identity entered find validation", result)
				}
			} else if string(fields["id"]) != `"different"` || string(result.Volume.Body["unknown"]) != "9007199254740993" {
				t.Fatal("response identity was replaced", result)
			}
			result.Observed.Body[0] = '!'
			result.Observed.Header.Set("X-Proof", "observation mutation")
			result.Volume.Body["name"] = json.RawMessage(`"caller"`)
			if !json.Valid(result.Value) || result.Volume.Header.Get("X-Proof") != "member" {
				t.Fatal("member result components share mutable bytes")
			}
		})
	}
}

func TestGetVolumeByIDPropagatesNativeClientErrorsWithoutFindFallback(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				volumeReadContractWire(t, r, volumeReadContractBase+"volumes/missing", "test-token")
				calls.Add(1)
				w.Header().Set("X-Proof", "rejected")
				testcloud.JSON(w, code, `{"error":"literal member rejected"}`)
			})
			result, err := blockstorage.GetVolumeByID(context.Background(), client, blockstorage.GetVolumeByIDRequest{ID: "missing"})
			var native gophercloud.ErrUnexpectedResponseCode
			var accepted *resource.ResponseError
			if result == nil || result.Value != nil || result.Volume != nil || result.Observed != nil || len(result.Pages) != 0 || calls.Load() != 1 || !errors.As(err, &native) || native.Actual != code || native.ResponseHeader.Get("X-Proof") != "rejected" || errors.As(err, &accepted) {
				t.Fatal(result, err, calls.Load())
			}
			volumeReadContractOperation(t, err, "GetVolumeByID")
		})
	}
}

func TestGetVolumeByIDAcceptsUnicodeLiteralSingleSegment(t *testing.T) {
	cloud := testcloud.New(t)
	client := volumeReadContractClient(cloud)
	id := "disk-한글"
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		volumeReadContractWire(t, r, volumeReadContractBase+"volumes/"+id, "test-token")
		if r.URL.EscapedPath() != volumeReadContractBase+"volumes/"+url.PathEscape(id) || r.URL.RawQuery != "" {
			t.Error("literal Unicode ID was reinterpreted or double escaped", r.URL)
		}
		calls.Add(1)
		testcloud.JSON(w, 200, `{"volume":{"id":"wire"}}`)
	})
	result, err := blockstorage.GetVolumeByID(context.Background(), client, blockstorage.GetVolumeByIDRequest{ID: id})
	if err != nil || result == nil || result.Volume == nil || len(result.Pages) != 0 || calls.Load() != 1 {
		t.Fatal(result, err, calls.Load())
	}
}

func TestVolumeExistsDistinguishesPresentResourceFromSuccessfulAbsence(t *testing.T) {
	for _, kind := range []string{"empty member object", "name found", "absent", "ambiguous"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path == volumeReadContractBase+"volumes/wanted" {
					if kind == "empty member object" {
						testcloud.JSON(w, 200, `{"volume":{}}`)
						return
					}
					testcloud.JSON(w, 404, `{"error":"member missing"}`)
					return
				}
				volumeReadContractWire(t, r, volumeReadContractPath, "test-token")
				if r.URL.Query().Get("name") != "wanted" || len(r.URL.Query()) != 1 {
					t.Error("exact name hint changed", r.URL)
				}
				rows := `[]`
				if kind == "name found" {
					rows = `[{"id":"found","name":"wanted"},{"id":"ignored","name":"Wanted"}]`
				}
				if kind == "ambiguous" {
					rows = `[{"id":"one","name":"wanted"},{"id":"two","name":"wanted"}]`
				}
				testcloud.JSON(w, 200, volumeReadContractPage(rows, ""))
			})
			result, err := blockstorage.VolumeExists(context.Background(), client, blockstorage.VolumeExistsRequest{NameOrID: "wanted"})
			if result == nil {
				t.Fatal(err)
			}
			if kind == "ambiguous" {
				if !errors.Is(err, resource.ErrAmbiguous) || result.Exists != nil || result.Value != nil || result.Volume != nil || calls.Load() != 2 {
					t.Fatal(result, err, calls.Load())
				}
				volumeReadContractOperation(t, err, "VolumeExists")
				return
			}
			want := kind != "absent"
			if err != nil || result.Exists == nil || *result.Exists != want || (result.Volume != nil) != want || (result.Value != nil) != want {
				t.Fatal(result, err)
			}
			if kind == "empty member object" {
				if calls.Load() != 1 || result.Observed == nil || len(result.Pages) != 0 || string(volumeReadContractFields(t, result.Value)["id"]) != "null" {
					t.Fatal("empty resource treated as absence", result, calls.Load())
				}
			} else if calls.Load() != 2 || result.Observed != nil || len(result.Pages) != 1 {
				t.Fatal("fallback dispatch/proof changed", result, calls.Load())
			}
		})
	}
}

func TestVolumeExistsUnsafeNamesAndWildcardTextUseExactNameSemantics(t *testing.T) {
	for _, name := range []string{"disk*", "disk/with/slash", "disk%2Fencoded", "disk\u00a0name", "disk?query", "disk\\path"} {
		t.Run(name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var member, list atomic.Int32
			encoded, _ := json.Marshal(name)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != volumeReadContractPath {
					member.Add(1)
					testcloud.JSON(w, 404, `{"error":"missing"}`)
					return
				}
				list.Add(1)
				if r.URL.Query().Get("name") != name || len(r.URL.Query()) != 1 {
					t.Error("unsafe name escaped/lost or route controls entered query", r.URL)
				}
				testcloud.JSON(w, 200, volumeReadContractPage(`[{"id":"decoy","name":"disk-other"},{"id":"exact","name":`+string(encoded)+`}]`, ""))
			})
			result, err := blockstorage.VolumeExists(context.Background(), client, blockstorage.VolumeExistsRequest{NameOrID: name})
			wantMember := int32(0)
			if name == "disk*" {
				wantMember = 1
			}
			if err != nil || result == nil || result.Exists == nil || !*result.Exists || result.Volume == nil || string(result.Volume.Body["id"]) != `"exact"` || member.Load() != wantMember || list.Load() != 1 {
				t.Fatal(result, err, member.Load(), list.Load())
			}
		})
	}
}

func TestVolumeExistsAmbiguityStopsBeforeUnusedTailThatFullListMustConsume(t *testing.T) {
	for _, tail := range []string{`false`, `{"id":"bad","bootable":"bad"}`} {
		t.Run(tail, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls atomic.Int32
			body := volumeReadContractPage(`[{"id":"first","name":"unsafe/name"},{"id":"second","name":"unsafe/name"},`+tail+`]`, "https://unused.invalid/volumes/detail")
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 200, body) })
			exists, err := blockstorage.VolumeExists(context.Background(), client, blockstorage.VolumeExistsRequest{NameOrID: "unsafe/name"})
			var proof *resource.ResponseError
			if exists == nil || exists.Exists != nil || exists.Value != nil || exists.Volume != nil || !errors.Is(err, resource.ErrAmbiguous) || errors.As(err, &proof) || len(exists.Pages) != 1 || calls.Load() != 1 {
				t.Fatal("ambiguity did not stop before unused tail/continuation", exists, err, calls.Load())
			}
			list, err := blockstorage.ListVolumes(context.Background(), client)
			if list == nil || list.Value != nil || list.Volumes != nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != body || calls.Load() != 2 {
				t.Fatal("full list failed to consume eager tail", list, err, calls.Load())
			}
		})
	}
}
