package blockstorage_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestVolumeBackupReadGetKeepsActualIDsAndSeparatesSeedFromPhysicalBody(t *testing.T) {
	for _, row := range []string{`{}`, `{"id":null}`, `{"id":false,"name":[1]}`, `{"id":"different","force":"false","links":{"literal":1},"unknown":9007199254740993}`, `{"id":[],"connection":null,"microversion":null,"_synchronized":null,"self":null}`} {
		t.Run(row, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			body := `{"backup":` + row + `}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				snapshotReadContractWire(t, r, backupReadContractBasic+"/literal-ID", "test-token")
				if r.URL.RawQuery != "" {
					t.Error("backup member query leaked", r.URL)
				}
				w.Header().Set("X-Proof", "member")
				testcloud.JSON(w, 203, body)
			})
			result, err := blockstorage.GetVolumeBackup(backupReadContractContext(t), client, blockstorage.GetVolumeBackupRequest{NameOrID: "literal-ID"})
			if err != nil || result == nil || result.Backup == nil || result.Observed == nil || len(result.Pages) != 0 || result.RequestedID != "literal-ID" || len(snapshotReadContractFields(t, result.Value)) != 24 || string(result.Observed.Body) != body || result.Backup.StatusCode != 203 || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			view := snapshotReadContractFields(t, result.Value)
			wire := snapshotReadContractFields(t, json.RawMessage(row))
			id, present := wire["id"]
			if !present {
				if !result.SeededID || string(view["id"]) != `"literal-ID"` {
					t.Fatal(result, string(result.Value))
				}
				if _, ok := result.Backup.Body["id"]; ok {
					t.Fatal("logical seed fabricated raw Backup ID")
				}
			} else if result.SeededID || string(view["id"]) != string(id) || string(result.Backup.Body["id"]) != string(id) {
				t.Fatal("response ID changed", result, string(result.Value))
			}
			result.Observed.Body[0] = '!'
			result.Observed.Header.Set("X-Proof", "changed")
			result.Backup.Body["name"] = json.RawMessage(`"caller"`)
			if !json.Valid(result.Value) || result.Backup.Header.Get("X-Proof") != "member" {
				t.Fatal("backup member phase aliased logical fields")
			}
		})
	}
}

func TestVolumeBackupReadGetAllowsPartialFlatMalformedAndSub400MemberResponses(t *testing.T) {
	for _, tc := range []struct {
		code         int
		body, wantID string
		seed         bool
	}{
		{201, `{"id":"flat","force":"FALSE","links":false,"object_count":3.9}`, `"flat"`, false},
		{203, `{"backup":{"description":"partial"}}`, `"seed"`, true},
		{204, `{"backup":{"id":"discarded by HTTP204"}}`, `"seed"`, true},
		{300, `{"backup":{"id":"wire","is_incremental":"false"}}`, `"wire"`, false},
		{203, "", `"seed"`, true}, {203, `{"backup":`, `"seed"`, true}, {203, "not JSON", `"seed"`, true},
	} {
		t.Run(http.StatusText(tc.code)+tc.body, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Proof", "physical")
				testcloud.JSON(w, tc.code, tc.body)
			})
			result, err := blockstorage.GetVolumeBackup(backupReadContractContext(t), client, blockstorage.GetVolumeBackupRequest{NameOrID: "seed"})
			if err != nil || result == nil || result.Backup == nil || result.Observed == nil || len(result.Pages) != 0 || calls.Load() != 1 || result.Observed.StatusCode != tc.code || result.Backup.StatusCode != tc.code || result.RequestedID != "seed" || result.SeededID != tc.seed || string(snapshotReadContractFields(t, result.Value)["id"]) != tc.wantID {
				t.Fatal(result, err, calls.Load())
			}
			wantBody := tc.body
			if tc.code == 204 {
				wantBody = ""
			}
			if string(result.Observed.Body) != wantBody {
				t.Fatal("physical body changed", string(result.Observed.Body), wantBody)
			}
			if tc.seed {
				if _, exists := result.Backup.Body["id"]; exists {
					t.Fatal("seed was inserted into raw phase", result.Backup)
				}
			}
			if tc.code == 201 {
				view := snapshotReadContractFields(t, result.Value)
				if string(view["force"]) != "true" || string(view["links"]) != "[false]" || string(view["object_count"]) != "3" {
					t.Fatal(string(result.Value))
				}
			}
		})
	}
}

func TestVolumeBackupReadGetValidShapeAndUTF8FailuresRemainTerminalPhysicalErrors(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `false`, `{"backup":null}`, `{"backup":[]}`, `{"backup":false}`, `{"backup":{"object_count":"²"}}`, string([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'})} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Proof", "invalid")
				testcloud.JSON(w, 203, body)
			})
			result, err := blockstorage.GetVolumeBackup(backupReadContractContext(t), client, blockstorage.GetVolumeBackupRequest{NameOrID: "wanted"})
			var physical *resource.ResponseError
			if result == nil || result.Value != nil || result.Backup != nil || result.Observed == nil || len(result.Pages) != 0 || calls.Load() != 1 || !errors.As(err, &physical) || physical.StatusCode != 203 || string(physical.Body) != body || physical.Header.Get("X-Proof") != "invalid" {
				t.Fatal(result, err, calls.Load())
			}
			snapshotReadContractOperation(t, err, "GetVolumeBackup")
		})
	}
}

func TestVolumeBackupReadFindCleanNativeFallbackIsExactAndUsesActualResponseID(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					snapshotReadContractWire(t, r, backupReadContractBasic+"/wanted", "test-token")
					testcloud.JSON(w, code, `{"error":"compatible backup rejection"}`)
				case 2:
					snapshotReadContractWire(t, r, backupReadContractDetail, "test-token")
					if r.URL.Query().Get("name") != "wanted" || len(r.URL.Query()) != 1 {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 203, backupReadContractPage(`[{"id":"wrong","name":"Wanted"},{"id":false,"name":"wanted","force":"false"},{"id":"other","name":"other"}]`, ""))
				default:
					t.Error("backup finder replayed", r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := blockstorage.GetVolumeBackup(backupReadContractContext(t), client, blockstorage.GetVolumeBackupRequest{NameOrID: "wanted"})
			if err != nil || result == nil || result.Backup == nil || result.Observed != nil || len(result.Pages) != 1 || result.SeededID || string(result.Backup.Body["id"]) != "false" || string(snapshotReadContractFields(t, result.Value)["force"]) != "true" || calls.Load() != 2 {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestVolumeBackupReadFindUnsafeLiteralAndWildcardNamesNeverBecomeSearchPatterns(t *testing.T) {
	for _, name := range []string{"wanted/name", "wanted*"} {
		t.Run(name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var members, lists atomic.Int32
			raw, _ := json.Marshal(name)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != backupReadContractDetail {
					members.Add(1)
					testcloud.JSON(w, 404, `{"error":"missing"}`)
					return
				}
				lists.Add(1)
				if r.URL.Query().Get("name") != name || len(r.URL.Query()) != 1 {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, backupReadContractPage(`[{"id":"decoy","name":"wanted-other"},{"id":"exact","name":`+string(raw)+`}]`, ""))
			})
			result, err := blockstorage.GetVolumeBackup(backupReadContractContext(t), client, blockstorage.GetVolumeBackupRequest{NameOrID: name})
			wantMembers := int32(0)
			if name == "wanted*" {
				wantMembers = 1
			}
			if err != nil || result == nil || result.Backup == nil || string(result.Backup.Body["id"]) != `"exact"` || members.Load() != wantMembers || lists.Load() != 1 {
				t.Fatal(result, err, members.Load(), lists.Load())
			}
		})
	}
}

func TestVolumeBackupReadFindAmbiguityPrecedesUnusedBadTailAndUniqueMustExhaust(t *testing.T) {
	for _, ambiguous := range []bool{true, false} {
		t.Run(map[bool]string{true: "duplicate early", false: "unique consumes bad tail"}[ambiguous], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			rows := `[{"id":"first","name":"name/only"},`
			if ambiguous {
				rows += `{"id":"second","name":"name/only"},`
			}
			rows += `{"id":"bad","object_count":"²"}]`
			body := backupReadContractPage(rows, "https://unused.invalid/")
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 203, body) })
			result, err := blockstorage.GetVolumeBackup(backupReadContractContext(t), client, blockstorage.GetVolumeBackupRequest{NameOrID: "name/only"})
			var physical *resource.ResponseError
			if result == nil || result.Value != nil || result.Backup != nil || result.Observed != nil || len(result.Pages) != 1 || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			if ambiguous {
				if !errors.Is(err, resource.ErrAmbiguous) || errors.As(err, &physical) {
					t.Fatal("unused backup descriptor masked earlier ambiguity", err)
				}
			} else if !errors.As(err, &physical) || string(physical.Body) != body {
				t.Fatal("backup unique result escaped unsuccessful exhaustion", err)
			}
		})
	}
}

func TestVolumeBackupReadFallbackLaterNativeFailureNeverBorrowsSuppressedMemberProof(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls atomic.Int32
	first := backupReadContractPage(`[{"id":"match","name":"wanted"}]`, "?marker=late")
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			w.Header().Set("X-Proof", "old member")
			testcloud.JSON(w, 404, `{"error":"old"}`)
		case 2:
			w.Header().Set("X-Proof", "first list")
			testcloud.JSON(w, 203, first)
		case 3:
			w.Header().Set("X-Proof", "current rejection")
			testcloud.JSON(w, 429, `{"error":"current"}`)
		default:
			t.Error("fallback restarted", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.GetVolumeBackup(backupReadContractContext(t), client, blockstorage.GetVolumeBackupRequest{NameOrID: "wanted"})
	var physical *resource.ResponseError
	var native gophercloud.ErrUnexpectedResponseCode
	if result == nil || result.Value != nil || result.Backup != nil || result.Observed != nil || len(result.Pages) != 1 || string(result.Pages[0].Body) != first || result.Pages[0].Header.Get("X-Proof") != "first list" || calls.Load() != 3 || errors.As(err, &physical) || !errors.As(err, &native) || native.Actual != 429 || native.ResponseHeader.Get("X-Proof") != "current rejection" {
		t.Fatal(result, err, calls.Load())
	}
}

func TestVolumeBackupReadGetEveryNonNullFalseyFilterUsesFullSearchWhileNullFinds(t *testing.T) {
	for _, filter := range []string{"null", "{}", "false", "0", "[]", `""`} {
		t.Run(filter, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if filter == "null" {
					snapshotReadContractWire(t, r, backupReadContractBasic+"/wanted", "test-token")
					testcloud.JSON(w, 203, `{"backup":{}}`)
					return
				}
				snapshotReadContractWire(t, r, backupReadContractDetail, "test-token")
				if r.URL.RawQuery != "" {
					t.Error("nonnull falsey filter borrowed find/name query", r.URL)
				}
				testcloud.JSON(w, 203, backupReadContractPage(`[{"id":"found","name":"wanted"},{"id":"ignored","name":"other"}]`, ""))
			})
			result, err := blockstorage.GetVolumeBackup(backupReadContractContext(t), client, blockstorage.GetVolumeBackupRequest{NameOrID: "wanted"}, blockstorage.WithVolumeBackupSearchFilters(json.RawMessage(filter)))
			if err != nil || result == nil || result.Backup == nil || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			if filter == "null" {
				if !result.SeededID || result.Observed == nil || len(result.Pages) != 0 || string(snapshotReadContractFields(t, result.Value)["id"]) != `"wanted"` {
					t.Fatal(result)
				}
			} else if result.SeededID || result.Observed != nil || len(result.Pages) != 1 || string(result.Backup.Body["id"]) != `"found"` {
				t.Fatal(result)
			}
		})
	}
}

func TestVolumeBackupReadSearchCompletesQuerylessPagesBeforeBackupExpression(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		snapshotReadContractWire(t, r, backupReadContractDetail, "test-token")
		switch calls.Add(1) {
		case 1:
			if r.URL.RawQuery != "" {
				t.Error("backup search predicate entered server query", r.URL)
			}
			testcloud.JSON(w, 203, backupReadContractPage(`[{"id":"first","name":"daily-one","is_incremental":"false"}]`, "?marker=next"))
		case 2:
			if r.URL.Query().Get("marker") != "next" || len(r.URL.Query()) != 1 {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, backupReadContractPage(`[{"id":"second","name":"daily-two","is_incremental":false},{"id":"third","name":"other","is_incremental":true}]`, ""))
		default:
			t.Error("backup search replayed", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.SearchVolumeBackups(backupReadContractContext(t), client, blockstorage.SearchVolumeBackupsRequest{NameOrID: "daily-*"}, blockstorage.WithVolumeBackupSearchExpression("{ids:[].id, incrementals:length([?is_incremental])}"))
	if err != nil || result == nil || result.Backups != nil || len(result.Pages) != 2 || calls.Load() != 2 {
		t.Fatal(result, err, calls.Load())
	}
	fields := snapshotReadContractFields(t, result.Value)
	if string(fields["ids"]) != `["first","second"]` || string(fields["incrementals"]) != "1" {
		t.Fatal(string(result.Value))
	}
}

func TestVolumeBackupReadSearchDoesNotHideExcludedLaterDescriptorFailure(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls atomic.Int32
	first := backupReadContractPage(`[{"id":"match","name":"wanted"}]`, "?marker=next")
	second := backupReadContractPage(`[{"id":"excluded","name":"other","object_count":"²"}]`, "")
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			testcloud.JSON(w, 200, first)
		case 2:
			w.Header().Set("X-Proof", "late bad backup")
			testcloud.JSON(w, 203, second)
		default:
			t.Error("search retried", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.SearchVolumeBackups(backupReadContractContext(t), client, blockstorage.SearchVolumeBackupsRequest{NameOrID: "wanted"})
	var physical *resource.ResponseError
	if result == nil || result.Value != nil || result.Backups != nil || len(result.Pages) != 2 || calls.Load() != 2 || !errors.As(err, &physical) || physical.StatusCode != 203 || string(physical.Body) != second || physical.Header.Get("X-Proof") != "late bad backup" {
		t.Fatal(result, err, calls.Load())
	}
	snapshotReadContractOperation(t, err, "SearchVolumeBackups")
}

func TestVolumeBackupReadGetArbitraryExpressionRetainsNoBackupAssociation(t *testing.T) {
	for _, tc := range []struct {
		expression, want string
		ambiguous        bool
	}{
		{"`[false]`", "false", false}, {"`[1]`", "1", false}, {"`false`", "", false}, {"`null`", "", false}, {"`[1,2]`", "", true},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				testcloud.JSON(w, 203, backupReadContractPage(`[{"id":"raw","force":"false"}]`, ""))
			})
			result, err := blockstorage.GetVolumeBackup(backupReadContractContext(t), client, blockstorage.GetVolumeBackupRequest{}, blockstorage.WithVolumeBackupSearchExpression(tc.expression))
			if result == nil || result.Backup != nil || result.Observed != nil || len(result.Pages) != 1 || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			if tc.ambiguous {
				var multiple *blockstorage.VolumeBackupSelectionError
				var physical *resource.ResponseError
				if !errors.As(err, &multiple) || multiple.Length != 2 || !errors.Is(err, resource.ErrAmbiguous) || errors.As(err, &physical) || result.Value != nil {
					t.Fatal(result, err)
				}
			} else if err != nil || string(result.Value) != tc.want {
				t.Fatal(result, err, string(result.Value))
			}
		})
	}
}
