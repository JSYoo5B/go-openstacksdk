package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const volumeTypeAccessContractID = "returned-한글"
const volumeTypeAccessContractPath = volumeTypeContractPath + "/" + volumeTypeAccessContractID + "/os-volume-type-access"
const volumeTypeAccessContractAction = volumeTypeContractPath + "/" + volumeTypeAccessContractID + "/action"

type volumeTypeAccessContractOutcome struct {
	nilResult         bool
	resolved          *blockstorage.GetVolumeTypeResult
	typeID, projectID string
	value             json.RawMessage
	accesses          []*resource.RawResource
	phase             *blockstorage.VolumeTypesPage
	err               error
}

func volumeTypeAccessContractCall(ctx context.Context, client *gophercloud.ServiceClient, operation, name, project string, options ...blockstorage.VolumeTypeReadOption) volumeTypeAccessContractOutcome {
	switch operation {
	case "GetVolumeTypeAccess":
		result, err := blockstorage.GetVolumeTypeAccess(ctx, client, blockstorage.GetVolumeTypeAccessRequest{NameOrID: name}, options...)
		if result == nil {
			return volumeTypeAccessContractOutcome{nilResult: true, err: err}
		}
		return volumeTypeAccessContractOutcome{resolved: result.Resolved, typeID: result.TypeID, value: result.Value, accesses: result.Accesses, phase: result.Observed, err: err}
	case "AddVolumeTypeAccess":
		result, err := blockstorage.AddVolumeTypeAccess(ctx, client, blockstorage.VolumeTypeAccessRequest{NameOrID: name, ProjectID: project}, options...)
		if result == nil {
			return volumeTypeAccessContractOutcome{nilResult: true, err: err}
		}
		return volumeTypeAccessContractOutcome{resolved: result.Resolved, typeID: result.TypeID, projectID: result.ProjectID, phase: result.Applied, err: err}
	case "RemoveVolumeTypeAccess":
		result, err := blockstorage.RemoveVolumeTypeAccess(ctx, client, blockstorage.VolumeTypeAccessRequest{NameOrID: name, ProjectID: project}, options...)
		if result == nil {
			return volumeTypeAccessContractOutcome{nilResult: true, err: err}
		}
		return volumeTypeAccessContractOutcome{resolved: result.Resolved, typeID: result.TypeID, projectID: result.ProjectID, phase: result.Applied, err: err}
	default:
		panic("unknown public volume type access operation")
	}
}

func volumeTypeAccessContractLookup(t *testing.T, r *http.Request, token string) {
	t.Helper()
	volumeReadContractWire(t, r, volumeTypeContractMember, token)
	if r.URL.RawQuery != "is_public=none" {
		t.Error("lookup none query changed", r.URL)
	}
}

func volumeTypeAccessContractSecond(t *testing.T, r *http.Request, operation, project, token string) {
	t.Helper()
	method, path := http.MethodGet, volumeTypeAccessContractPath
	if operation != "GetVolumeTypeAccess" {
		method, path = http.MethodPost, volumeTypeAccessContractAction
	}
	if r.Method != method || r.URL.Path != path || r.URL.RawQuery != "" || r.Header.Get("X-Source") != "entry" || r.Header.Get("X-Auth-Token") != token || r.Header.Get("OpenStack-API-Version") != "volume 3.60" {
		t.Error("fixed access stage request changed", r.Method, r.URL, r.Header)
	}
	if operation == "GetVolumeTypeAccess" {
		if r.Body != nil {
			body, err := io.ReadAll(r.Body)
			if err != nil || len(body) != 0 {
				t.Error(string(body), err)
			}
		}
		return
	}
	var body map[string]map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Error(err)
		return
	}
	key := "addProjectAccess"
	if operation == "RemoveVolumeTypeAccess" {
		key = "removeProjectAccess"
	}
	if len(body) != 1 || len(body[key]) != 1 || body[key]["project"] != project {
		t.Error("exact literal action payload changed", body, project)
	}
}

func TestGetVolumeTypeAccessUsesReturnedIDAndOwnsRawAccessRowsWithoutProjectionOrPaging(t *testing.T) {
	cloud := testcloud.New(t)
	client := volumeReadContractClient(cloud)
	var calls atomic.Int32
	lookup := `{"volume_type":{"id":"returned-한글","name":"wire","location":false,"extra_specs":false}}`
	value := `[{"volume_type_id":false,"project_id":null,"metadata":{"number":9007199254740993},"location":false},{"project_id":{"number":9007199254740993},"unknown":[null]}]`
	body := `{"volume_type_access":` + value + `,"links":false,"volume_type_access_links":[{"rel":"next","href":"https://unused.invalid"}]}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			volumeTypeAccessContractLookup(t, r, "test-token")
			w.Header().Set("X-Proof", "lookup")
			testcloud.JSON(w, 200, lookup)
		case 2:
			volumeTypeAccessContractSecond(t, r, "GetVolumeTypeAccess", "", "test-token")
			w.Header().Set("X-Proof", "access")
			testcloud.JSON(w, 200, body)
		default:
			t.Error("access paging/refetch/enrichment", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.GetVolumeTypeAccess(context.Background(), client, blockstorage.GetVolumeTypeAccessRequest{NameOrID: "target"})
	if err != nil || result == nil || result.TypeID != volumeTypeAccessContractID || result.Resolved == nil || result.Resolved.Type == nil || result.Resolved.Observed == nil || len(result.Resolved.Pages) != 0 || result.Observed == nil || calls.Load() != 2 || string(result.Value) != value || len(result.Accesses) != 2 || result.Accesses[0] == result.Accesses[1] {
		t.Fatal(result, err, calls.Load())
	}
	if string(result.Accesses[0].Body["volume_type_id"]) != "false" || string(result.Accesses[0].Body["project_id"]) != "null" || string(result.Accesses[0].Body["location"]) != "false" || string(result.Accesses[0].Body["metadata"]) != `{"number":9007199254740993}` || result.Accesses[1].Header.Get("X-Proof") != "access" || result.Accesses[0].StatusCode != 200 {
		t.Fatal("access dictionaries projected or enriched", result)
	}
	result.Resolved.Observed.Body[0] = '!'
	result.Resolved.Observed.Header.Set("X-Proof", "lookup changed")
	result.Observed.Body[0] = '!'
	result.Accesses[0].Body["metadata"][0] = '!'
	result.Accesses[0].Header.Set("X-Proof", "row changed")
	if string(result.Value) != value || result.Resolved.Type.Header.Get("X-Proof") != "lookup" || result.Accesses[1].Header.Get("X-Proof") != "access" || result.Observed.Header.Get("X-Proof") != "access" || !json.Valid(result.Resolved.Value) {
		t.Fatal("lookup/access/row/view proofs alias", result)
	}
}

func TestGetVolumeTypeAccessPreservesMissingAndEveryArbitraryFieldShape(t *testing.T) {
	for _, tc := range []struct {
		field, want string
		objects     bool
	}{
		{"", "[]", true}, {`,"volume_type_access":[]`, "[]", true}, {`,"volume_type_access":null`, "null", false}, {`,"volume_type_access":false`, "false", false}, {`,"volume_type_access":0`, "0", false}, {`,"volume_type_access":9007199254740993`, "9007199254740993", false}, {`,"volume_type_access":"opaque"`, `"opaque"`, false}, {`,"volume_type_access":{"opaque":false}`, `{"opaque":false}`, false}, {`,"volume_type_access":[{},null,false]`, `[{},null,false]`, false},
	} {
		t.Run(tc.want+tc.field, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					volumeTypeAccessContractLookup(t, r, "test-token")
					testcloud.JSON(w, 200, `{"volume_type":{"id":"returned-한글"}}`)
				case 2:
					volumeTypeAccessContractSecond(t, r, "GetVolumeTypeAccess", "", "test-token")
					testcloud.JSON(w, 200, `{"Volume_Type_Access":["wrongcase"]`+tc.field+`,"next":false}`)
				default:
					t.Error(r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := blockstorage.GetVolumeTypeAccess(context.Background(), client, blockstorage.GetVolumeTypeAccessRequest{NameOrID: "target"})
			if err != nil || result == nil || result.Value == nil || string(result.Value) != tc.want || calls.Load() != 2 || result.Observed == nil || result.Resolved == nil {
				t.Fatal(result, err, calls.Load())
			}
			if tc.objects {
				if result.Accesses == nil || len(result.Accesses) != 0 {
					t.Fatal(result)
				}
			} else if result.Accesses != nil {
				t.Fatal("partial/fabricated dictionaries for arbitrary shape", result)
			}
		})
	}
}

func TestVolumeTypeAccessActionsForwardLiteralProjectAndOpaqueAcknowledgement(t *testing.T) {
	for _, operation := range []string{"AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		for _, project := range []string{"", "nonexistent-project", "literal /?%한글", "control\x00\n\t"} {
			t.Run(operation+"/"+project, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				var calls atomic.Int32
				opaque := []byte{0xff, 0x00, 'n', 'o', 't', 'J', 'S', 'O', 'N'}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					switch calls.Add(1) {
					case 1:
						volumeTypeAccessContractLookup(t, r, "test-token")
						w.Header().Set("X-Proof", "lookup")
						testcloud.JSON(w, 200, `{"volume_type":{"id":"returned-한글"}}`)
					case 2:
						volumeTypeAccessContractSecond(t, r, operation, project, "test-token")
						w.Header().Set("X-Proof", "action")
						w.WriteHeader(202)
						_, _ = w.Write(opaque)
					default:
						t.Error("identity/project/verification/rollback request", r.URL)
						w.WriteHeader(500)
					}
				})
				got := volumeTypeAccessContractCall(context.Background(), client, operation, "target", project)
				if got.err != nil || got.nilResult || got.typeID != volumeTypeAccessContractID || got.projectID != project || got.resolved == nil || got.resolved.Observed == nil || got.phase == nil || got.phase.StatusCode != 202 || string(got.phase.Body) != string(opaque) || calls.Load() != 2 {
					t.Fatal(got, calls.Load())
				}
				got.phase.Body[0] = '!'
				got.phase.Header.Set("X-Proof", "caller")
				if got.resolved.Observed.Header.Get("X-Proof") != "lookup" || got.resolved.Type.Header.Get("X-Proof") != "lookup" {
					t.Fatal("acknowledgement aliases lookup", got)
				}
			})
		}
	}
}

func TestVolumeTypeAccessRejectsInvalidCanonicalResponseIDLocallyAfterLookup(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		for _, id := range []string{"", `null`, `false`, `0`, `""`, `[]`, `{}`, `"."`, `"a/b"`, `"a?b"`, `"a%b"`, `"a b"`, `"a\u2000b"`, `"a\nb"`} {
			t.Run(operation+"/"+id, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				var calls atomic.Int32
				row := `{}`
				if id != "" {
					row = `{"id":` + id + `}`
				}
				body := `{"volume_type":` + row + `}`
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					volumeTypeAccessContractLookup(t, r, "test-token")
					if calls.Add(1) != 1 {
						t.Error("invalid returned id reached access/action", r.URL)
					}
					w.Header().Set("X-Proof", "lookup-id")
					testcloud.JSON(w, 200, body)
				})
				got := volumeTypeAccessContractCall(context.Background(), client, operation, "target", "project")
				var physical *resource.ResponseError
				if got.nilResult || got.resolved == nil || got.resolved.Type == nil || got.resolved.Observed == nil || string(got.resolved.Observed.Body) != body || got.phase != nil || got.value != nil || got.accesses != nil || got.typeID != "" || calls.Load() != 1 || !errors.Is(got.err, resource.ErrInvalidOption) || errors.As(got.err, &physical) {
					t.Fatal("local returned-id error lost lookup or borrowed response failure", got, calls.Load())
				}
				volumeTypeContractOperation(t, got.err, operation)
			})
		}
	}
}

func TestVolumeTypeAccessInvalidUTF8ProjectIsLocalAfterResolvedType(t *testing.T) {
	for _, operation := range []string{"AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				volumeTypeAccessContractLookup(t, r, "test-token")
				if calls.Add(1) != 1 {
					t.Error("invalid project reached mutation", r.URL)
				}
				testcloud.JSON(w, 200, `{"volume_type":{"id":"returned-한글"}}`)
			})
			got := volumeTypeAccessContractCall(context.Background(), client, operation, "target", string([]byte{0xff}))
			var physical *resource.ResponseError
			if got.nilResult || got.resolved == nil || got.resolved.Type == nil || got.resolved.Observed == nil || got.phase != nil || calls.Load() != 1 || !errors.Is(got.err, resource.ErrInvalidOption) || errors.As(got.err, &physical) {
				t.Fatal(got, calls.Load())
			}
		})
	}
}

func TestVolumeTypeAccessCompletedAbsenceAndEarlyAmbiguityStopBeforeSecondStage(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		for _, multiple := range []bool{false, true} {
			t.Run(operation+map[bool]string{false: " absent", true: " ambiguous"}[multiple], func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				var calls atomic.Int32
				name := "unsafe / name"
				rows := `[]`
				tail := ""
				if multiple {
					rows = `[{"id":"one","name":"unsafe / name"},{"id":"two","name":"unsafe / name"},false]`
					tail = `,"links":false`
				}
				body := `{"volume_types":` + rows + tail + `}`
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					volumeReadContractWire(t, r, volumeTypeContractPath, "test-token")
					if r.URL.RawQuery != "is_public=none" || calls.Add(1) != 1 {
						t.Error(r.URL)
					}
					w.Header().Set("X-Proof", "list")
					testcloud.JSON(w, 200, body)
				})
				got := volumeTypeAccessContractCall(context.Background(), client, operation, name, "project")
				var physical *resource.ResponseError
				if got.nilResult || got.resolved == nil || got.resolved.Type != nil || got.resolved.Observed != nil || len(got.resolved.Pages) != 1 || string(got.resolved.Pages[0].Body) != body || got.phase != nil || got.value != nil || calls.Load() != 1 || errors.As(got.err, &physical) {
					t.Fatal(got, calls.Load())
				}
				if multiple {
					if !errors.Is(got.err, resource.ErrAmbiguous) {
						t.Fatal(got.err)
					}
				} else {
					var missing *resource.NotFoundError
					if !errors.As(got.err, &missing) || missing.Resource != "volume type" || missing.Reference != name {
						t.Fatal(got.err)
					}
				}
				volumeTypeContractOperation(t, got.err, operation)
			})
		}
	}
}

func TestVolumeTypeAccessFallbackFindUsesNoneThenReturnedIDWithoutVisibilityQuery(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		for _, code := range []int{400, 403, 404} {
			t.Run(operation+"/"+http.StatusText(code), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				var calls atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					switch calls.Add(1) {
					case 1:
						volumeTypeAccessContractLookup(t, r, "test-token")
						testcloud.JSON(w, code, `{"error":"missing member"}`)
					case 2:
						volumeReadContractWire(t, r, volumeTypeContractPath, "test-token")
						if r.URL.RawQuery != "is_public=none" {
							t.Error("fallback name hint or visibility changed", r.URL)
						}
						w.Header().Set("X-Proof", "lookup-list")
						testcloud.JSON(w, 200, volumeTypeContractPage(`[{"id":"returned-한글","name":"target"}]`, ""))
					case 3:
						volumeTypeAccessContractSecond(t, r, operation, "project", "test-token")
						if operation == "GetVolumeTypeAccess" {
							testcloud.JSON(w, 200, `{}`)
						} else {
							w.WriteHeader(202)
						}
					default:
						t.Error(r.URL)
						w.WriteHeader(500)
					}
				})
				got := volumeTypeAccessContractCall(context.Background(), client, operation, "target", "project")
				if got.err != nil || got.nilResult || got.typeID != volumeTypeAccessContractID || got.resolved == nil || got.resolved.Type == nil || got.resolved.Observed != nil || len(got.resolved.Pages) != 1 || got.phase == nil || calls.Load() != 3 {
					t.Fatal(got, calls.Load())
				}
			})
		}
	}
}

func TestVolumeTypeAccessAdmitsSub400StatusWhileAccessJSONRemainsRequired(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		codes := []int{201, 203, 304, 399, 204}
		if operation != "GetVolumeTypeAccess" {
			codes = []int{200, 204, 302, 399}
		}
		for _, code := range codes {
			t.Run(operation+"/"+http.StatusText(code), func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				var calls atomic.Int32
				body := string([]byte{0xff, 0x00, '!'})
				if operation == "GetVolumeTypeAccess" {
					body = `{"volume_type_access":false}`
					if code == 204 {
						body = ""
					}
				}
				cloud.Provider.HTTPClient.Transport = getVolumesContractTransport(func(r *http.Request) (*http.Response, error) {
					if calls.Add(1) == 1 {
						volumeTypeAccessContractLookup(t, r, "test-token")
						return getVolumesContractResponse(r, 200, io.NopCloser(strings.NewReader(`{"volume_type":{"id":"returned-한글"}}`)), "lookup"), nil
					}
					volumeTypeAccessContractSecond(t, r, operation, "project", "test-token")
					return getVolumesContractResponse(r, code, io.NopCloser(strings.NewReader(body)), "access-status"), nil
				})
				got := volumeTypeAccessContractCall(context.Background(), client, operation, "target", "project")
				if got.nilResult || got.resolved == nil || got.phase == nil || got.phase.StatusCode != code || string(got.phase.Body) != body || calls.Load() != 2 {
					t.Fatal(got, calls.Load())
				}
				if operation == "GetVolumeTypeAccess" && code == 204 {
					var physical *resource.ResponseError
					if !errors.As(got.err, &physical) || physical.StatusCode != 204 || got.value != nil || got.accesses != nil {
						t.Fatal(got)
					}
				} else if got.err != nil {
					t.Fatal(got)
				} else if operation == "GetVolumeTypeAccess" && string(got.value) != "false" {
					t.Fatal(got)
				}
			})
		}
	}
}
