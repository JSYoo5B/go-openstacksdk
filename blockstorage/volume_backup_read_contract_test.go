package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

const backupReadContractBasic = snapshotReadContractBase + "backups"
const backupReadContractDetail = backupReadContractBasic + "/detail"

func backupReadContractContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}
func backupReadContractPage(rows, next string) string {
	if next == "" {
		return `{"backups":` + rows + `}`
	}
	href, _ := json.Marshal(next)
	return `{"backups":` + rows + `,"backups_links":[{"rel":"next","href":` + string(href) + `}]}`
}

func TestVolumeBackupReadListDefaultDetailedAndBasicKeepCanonicalEmptyEOF(t *testing.T) {
	for _, basic := range []bool{false, true} {
		t.Run(map[bool]string{false: "default detailed", true: "explicit basic"}[basic], func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			path := backupReadContractDetail
			var options []blockstorage.VolumeBackupListOption
			if basic {
				path = backupReadContractBasic
				options = append(options, blockstorage.WithVolumeBackupListDetailed(false))
			}
			body := `{"backups":[],"links":[null],"next":"https://unused.invalid/"}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				snapshotReadContractWire(t, r, path, "test-token")
				if r.URL.RawQuery != "" {
					t.Error("default backup list query leaked", r.URL)
				}
				w.Header().Set("X-Proof", "empty")
				testcloud.JSON(w, 203, body)
			})
			result, err := blockstorage.ListVolumeBackups(backupReadContractContext(t), client, options...)
			if err != nil || result == nil || string(result.Value) != "[]" || result.Backups == nil || len(result.Backups) != 0 || len(result.Pages) != 1 || calls.Load() != 1 || result.Pages[0].StatusCode != 203 || string(result.Pages[0].Body) != body || result.Pages[0].Header.Get("X-Proof") != "empty" {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestVolumeBackupReadListOwnsAllTwentyFourFieldsAndRawIndependentProofs(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls atomic.Int32
	currentName, cloudName, region := "current name", "configured cloud", "region-one"
	location := resource.CloudLocation{Cloud: &cloudName, RegionName: &region, Zone: json.RawMessage(`"discard configured zone"`), Project: resource.CloudProject{ID: json.RawMessage(`"current"`), Name: &currentName}}
	body := backupReadContractPage(`[{"id":"same","name":"first","force":"false","has_dependent_backups":{},"is_incremental":[false],"links":{"rel":"self"},"object_count":"003","size":3.9,"metadata":[1],"availability_zone":"az-one","project_id":"first alias","os-backup-project-attr:project_id":"foreign","created_at":"literal date","data_timestamp":false,"encryption_key_id":[1],"snapshot_id":null,"user_id":0,"volume_name":{"raw":true},"unknown":9007199254740993,"location":{"ignored":true}},{"id":"same","name":"second","links":null,"availability_zone":[1],"force":0,"is_incremental":null}]`, "")
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		snapshotReadContractWire(t, r, backupReadContractDetail, "test-token")
		w.Header().Set("X-Proof", "raw")
		testcloud.JSON(w, 203, body)
	})
	result, err := blockstorage.ListVolumeBackups(backupReadContractContext(t), client, blockstorage.WithVolumeBackupListLocation(location))
	if err != nil || result == nil || len(result.Backups) != 2 || result.Backups[0] == result.Backups[1] || len(result.Pages) != 1 || calls.Load() != 1 {
		t.Fatal(result, err, calls.Load())
	}
	rows := snapshotReadContractRows(t, result.Value)
	if len(rows) != 2 || len(rows[0]) != 24 || len(rows[1]) != 24 {
		t.Fatal(string(result.Value))
	}
	expected := map[string]string{"force": "true", "has_dependent_backups": "false", "is_incremental": "true", "links": `[{"rel":"self"}]`, "object_count": "3", "size": "3", "metadata": "{}", "project_id": `"foreign"`, "created_at": `"literal date"`, "data_timestamp": "false", "encryption_key_id": "[1]", "snapshot_id": "null", "user_id": "0", "volume_name": `{"raw":true}`}
	for key, want := range expected {
		if string(rows[0][key]) != want {
			t.Errorf("field %s: %s want %s", key, rows[0][key], want)
		}
	}
	if string(rows[1]["links"]) != "null" || string(rows[1]["force"]) != "false" || string(rows[1]["is_incremental"]) != "null" {
		t.Fatal(string(result.Value))
	}
	if _, ok := rows[0]["unknown"]; ok {
		t.Fatal("wire extension entered normalized Backup")
	}
	if _, ok := rows[0]["is_forced"]; ok {
		t.Fatal("Snapshot descriptor entered Backup")
	}
	firstLocation := snapshotReadContractFields(t, rows[0]["location"])
	project := snapshotReadContractFields(t, firstLocation["project"])
	if string(firstLocation["zone"]) != `"az-one"` || string(firstLocation["cloud"]) != `"configured cloud"` || string(firstLocation["region_name"]) != `"region-one"` || string(project["id"]) != `"foreign"` || string(project["name"]) != "null" {
		t.Fatal(string(rows[0]["location"]))
	}
	secondLocation := snapshotReadContractFields(t, rows[1]["location"])
	if string(secondLocation["zone"]) != "[1]" {
		t.Fatal(string(rows[1]["location"]))
	}
	if string(result.Backups[0].Body["force"]) != `"false"` || string(result.Backups[0].Body["unknown"]) != "9007199254740993" || result.Backups[1].Header.Get("X-Proof") != "raw" {
		t.Fatal(result)
	}
	result.Backups[0].Body["id"][0] = '!'
	result.Backups[0].Header.Set("X-Proof", "changed row")
	result.Pages[0].Body[0] = '!'
	result.Pages[0].Header.Set("X-Proof", "changed page")
	location.Project.ID[0] = '!'
	location.Zone[0] = '!'
	currentName = "changed"
	cloudName = "changed"
	if !json.Valid(result.Value) || string(result.Backups[1].Body["id"]) != `"same"` || result.Backups[1].Header.Get("X-Proof") != "raw" || string(snapshotReadContractFields(t, rows[0]["location"])["cloud"]) != `"configured cloud"` {
		t.Fatal("backup view/raw/proof/location storage aliased")
	}
}

func TestVolumeBackupReadAllProjectsIsLiteralWhileSnapshotProxyBindsTruthiness(t *testing.T) {
	for _, tc := range []struct {
		raw              string
		backup, snapshot []string
	}{
		{"null", nil, []string{"9"}}, {"false", []string{"False"}, []string{"9"}}, {"0", []string{"0"}, []string{"9"}}, {`"false"`, []string{"false"}, []string{"True"}}, {"[]", nil, []string{"9"}}, {"[1]", []string{"1"}, []string{"True"}}, {"{}", nil, []string{"9"}}, {`{"x":1}`, []string{"x"}, []string{"True"}},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				want := tc.backup
				body := `{"backups":[]}`
				if r.URL.Path == snapshotReadContractDetail {
					want = tc.snapshot
					body = `{"snapshots":[]}`
				} else if r.URL.Path != backupReadContractDetail {
					t.Error("resource route changed", r.URL)
				}
				if !reflect.DeepEqual(r.URL.Query()["all_tenants"], want) {
					t.Error("literal vs bound alias", r.URL.Query(), want)
				}
				if len(r.URL.Query()) != map[bool]int{true: 0, false: 1}[want == nil] {
					t.Error("raw all_projects was sent or query expanded", r.URL.Query())
				}
				testcloud.JSON(w, 200, body)
			})
			filters := json.RawMessage(`{"all_projects":` + tc.raw + `,"all_tenants":9}`)
			backup, err := blockstorage.ListVolumeBackups(backupReadContractContext(t), client, blockstorage.WithVolumeBackupListFilters(filters))
			if err != nil || backup == nil || string(backup.Value) != "[]" {
				t.Fatal(backup, err)
			}
			snapshot, err := blockstorage.ListVolumeSnapshots(backupReadContractContext(t), client, blockstorage.WithVolumeSnapshotListFilters(filters))
			if err != nil || snapshot == nil || string(snapshot.Value) != "[]" || calls.Load() != 2 {
				t.Fatal(snapshot, err, calls.Load())
			}
		})
	}
}

func TestVolumeBackupReadLocalAttributesDifferFromIgnoredSnapshotAndCreationNames(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls atomic.Int32
	filters := json.RawMessage(`{"name":"server name","status":"available","volume_id":"parent","force":true,"is_incremental":true,"has_dependent_backups":false,"availability_zone":"az","object_count":3,"size":3,"links":[{"rel":"keep"}],"metadata":{"flag":1},"is_forced":false,"consumes_quota":true,"group_snapshot_id":"ignored","incremental":false,"with_count":true,"name~":"ignored","unknown":1}`)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		snapshotReadContractWire(t, r, backupReadContractDetail, "test-token")
		want := map[string][]string{"name": {"server name"}, "status": {"available"}, "volume_id": {"parent"}}
		if !reflect.DeepEqual(map[string][]string(r.URL.Query()), want) {
			t.Error("backup server/local/unknown separation", r.URL.Query(), want)
		}
		testcloud.JSON(w, 200, backupReadContractPage(`[{"id":"keep","force":"false","is_incremental":"false","has_dependent_backups":0,"availability_zone":"az","object_count":"3","size":"003","links":{"rel":"keep"},"metadata":{"flag":true,"extra":2}},{"id":"exclude","force":false,"is_incremental":true,"has_dependent_backups":false,"availability_zone":"az","object_count":3,"size":3,"links":[{"rel":"keep"}],"metadata":{"flag":1}}]`, ""))
	})
	result, err := blockstorage.ListVolumeBackups(backupReadContractContext(t), client, blockstorage.WithVolumeBackupListFilters(filters))
	if err != nil || result == nil || len(result.Backups) != 1 || string(result.Backups[0].Body["id"]) != `"keep"` || len(result.Pages) != 1 || calls.Load() != 1 {
		t.Fatal(result, err, calls.Load())
	}
}

func TestVolumeBackupReadInvalidIntegersAreEagerBeforeLocalExclusion(t *testing.T) {
	for _, field := range []string{"size", "object_count"} {
		t.Run(field, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			body := backupReadContractPage(`[{"id":"good","force":false},{"id":"excluded","force":"false","`+field+`":"²"}]`, "https://unused.invalid/")
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Proof", "eager")
				testcloud.JSON(w, 203, body)
			})
			result, err := blockstorage.ListVolumeBackups(backupReadContractContext(t), client, blockstorage.WithVolumeBackupListFilters(json.RawMessage(`{"force":false}`)))
			var physical *resource.ResponseError
			if result == nil || result.Value != nil || result.Backups != nil || len(result.Pages) != 1 || calls.Load() != 1 || !errors.As(err, &physical) || physical.StatusCode != 203 || string(physical.Body) != body || physical.Header.Get("X-Proof") != "eager" {
				t.Fatal(result, err, calls.Load())
			}
			snapshotReadContractOperation(t, err, "ListVolumeBackups")
		})
	}
}

func TestVolumeBackupReadCollectionEnvelopeAndListConstructorControlsStayResourceSpecific(t *testing.T) {
	for _, tc := range []struct {
		body    string
		ok      bool
		rawSelf bool
	}{
		{`{"backups":{"id":"one","force":"bad accepted bool"}}`, true, false},
		{`{"backups":[{"id":"one","self":null}]}`, true, true},
		{`{"backups":[{"id":"one","connection":null}]}`, false, false},
		{`{"backups":[{"id":"one","microversion":null}]}`, false, false},
		{`{"backups":[{"id":"one","_synchronized":null}]}`, false, false},
		{`{"snapshots":[]}`, false, false}, {`{"backups":null}`, false, false}, {`{"backups":[false]}`, false, false},
	} {
		t.Run(tc.body, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); testcloud.JSON(w, 203, tc.body) })
			result, err := blockstorage.ListVolumeBackups(backupReadContractContext(t), client)
			if result == nil || len(result.Pages) != 1 || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
			if tc.ok {
				if err != nil || len(result.Backups) != 1 || len(snapshotReadContractRows(t, result.Value)[0]) != 24 {
					t.Fatal(result, err)
				}
				if tc.rawSelf && string(result.Backups[0].Body["self"]) != "null" {
					t.Fatal("list self source no-op changed physical row", result)
				}
			} else {
				var physical *resource.ResponseError
				if result.Value != nil || result.Backups != nil || !errors.As(err, &physical) || string(physical.Body) != tc.body {
					t.Fatal(result, err)
				}
			}
		})
	}
}

func TestVolumeBackupReadShortLimitUsesExcludedArbitraryMarkerThenEmptyEOF(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls atomic.Int32
	first := backupReadContractPage(`[{"id":"keep","force":false},{"id":{"excluded":true},"force":"false"}]`, "")
	last := `{"backups":[],"backups_links":[null],"next":"https://unused.invalid/"}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		snapshotReadContractWire(t, r, backupReadContractDetail, "test-token")
		switch calls.Add(1) {
		case 1:
			want := map[string][]string{"limit": {"3"}, "status": {"only"}}
			if !reflect.DeepEqual(map[string][]string(r.URL.Query()), want) {
				t.Error(r.URL.Query(), want)
			}
			w.Header().Set("X-Proof", "short")
			testcloud.JSON(w, 203, first)
		case 2:
			want := map[string][]string{"limit": {"3"}, "status": {"only"}, "marker": {"excluded"}}
			if !reflect.DeepEqual(map[string][]string(r.URL.Query()), want) {
				t.Error("last excluded arbitrary ID marker lost", r.URL.Query(), want)
			}
			w.Header().Set("X-Proof", "EOF")
			testcloud.JSON(w, 200, last)
		default:
			t.Error("extra backup page", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.ListVolumeBackups(backupReadContractContext(t), client, blockstorage.WithVolumeBackupListFilters(json.RawMessage(`{"limit":3,"status":"only","force":false}`)))
	if err != nil || result == nil || len(result.Backups) != 1 || string(result.Backups[0].Body["id"]) != `"keep"` || len(result.Pages) != 2 || calls.Load() != 2 || string(result.Pages[0].Body) != first || string(result.Pages[1].Body) != last || result.Pages[0].Header.Get("X-Proof") != "short" || result.Pages[1].Header.Get("X-Proof") != "EOF" {
		t.Fatal(result, err, calls.Load())
	}
}

func TestVolumeBackupReadBackupsLinksMaterializeDistinctRowsAcrossPages(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		snapshotReadContractWire(t, r, backupReadContractDetail, "test-token")
		switch calls.Add(1) {
		case 1:
			if r.URL.RawQuery != "" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 203, backupReadContractPage(`[{"id":"duplicate","name":"first","links":false}]`, "?marker=two"))
		case 2:
			if r.URL.Query().Get("marker") != "two" || len(r.URL.Query()) != 1 {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, backupReadContractPage(`[{"id":"duplicate","name":"second","links":"literal","force":"false"}]`, ""))
		default:
			t.Error("backup pager lost plural links or replayed", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := blockstorage.ListVolumeBackups(backupReadContractContext(t), client)
	if err != nil || result == nil || len(result.Backups) != 2 || result.Backups[0] == result.Backups[1] || len(result.Pages) != 2 || calls.Load() != 2 {
		t.Fatal(result, err, calls.Load())
	}
	rows := snapshotReadContractRows(t, result.Value)
	if string(rows[0]["links"]) != "[false]" || string(rows[1]["links"]) != `["literal"]` || string(rows[1]["force"]) != "true" || string(rows[0]["name"]) != `"first"` || string(rows[1]["name"]) != `"second"` {
		t.Fatal(string(result.Value))
	}
}

func TestVolumeBackupReadMaximumCountsExcludedRowsAndDefersUnusedConstruction(t *testing.T) {
	for _, tc := range []struct {
		maximum   int
		body      string
		wantQuery string
	}{
		{1, backupReadContractPage(`[{"id":"excluded","force":"false"},false]`, "https://unused.invalid/"), "1"},
		{-1, `{"backups":[false],"backups_links":[null]}`, "-1"},
	} {
		t.Run(tc.wantQuery, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("limit") != tc.wantQuery {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 203, tc.body)
			})
			result, err := blockstorage.ListVolumeBackups(backupReadContractContext(t), client, blockstorage.WithVolumeBackupListMaxItems(tc.maximum), blockstorage.WithVolumeBackupListFilters(json.RawMessage(`{"force":false}`)))
			if err != nil || result == nil || string(result.Value) != "[]" || result.Backups == nil || len(result.Backups) != 0 || len(result.Pages) != 1 || calls.Load() != 1 {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestVolumeBackupReadMaximumAtPageEndStillFetchesAndDecodesNextResponse(t *testing.T) {
	for _, body := range []string{`{"backups":[false]}`, `{"backups":`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				snapshotReadContractWire(t, r, backupReadContractDetail, "test-token")
				switch calls.Add(1) {
				case 1:
					testcloud.JSON(w, 200, backupReadContractPage(`[{"id":"first"}]`, "?marker=next"))
				case 2:
					if r.URL.Query().Get("marker") != "next" {
						t.Error(r.URL)
					}
					w.Header().Set("X-Proof", "second")
					testcloud.JSON(w, 203, body)
				default:
					t.Error("maximum issued third response", r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := blockstorage.ListVolumeBackups(backupReadContractContext(t), client, blockstorage.WithVolumeBackupListMaxItems(1))
			if result == nil || len(result.Pages) != 2 || calls.Load() != 2 || string(result.Pages[1].Body) != body {
				t.Fatal(result, err, calls.Load())
			}
			if json.Valid([]byte(body)) {
				if err != nil || len(result.Backups) != 1 || string(result.Backups[0].Body["id"]) != `"first"` {
					t.Fatal(result, err)
				}
			} else {
				var physical *resource.ResponseError
				if result.Value != nil || result.Backups != nil || !errors.As(err, &physical) || physical.StatusCode != 203 || string(physical.Body) != body {
					t.Fatal(result, err)
				}
			}
		})
	}
}

func TestVolumeBackupReadListExpressionUsesCompletedBackupValueAndNoRowAssociation(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		testcloud.JSON(w, 203, backupReadContractPage(`[{"id":"truthy","force":"false"},{"id":"falsey","force":0}]`, ""))
	})
	result, err := blockstorage.ListVolumeBackups(backupReadContractContext(t), client, blockstorage.WithVolumeBackupListExpression("{forced:length([?force]), ids:[].id}"))
	if err != nil || result == nil || result.Backups != nil || len(result.Pages) != 1 || calls.Load() != 1 {
		t.Fatal(result, err, calls.Load())
	}
	fields := snapshotReadContractFields(t, result.Value)
	if string(fields["forced"]) != "1" || string(fields["ids"]) != `["truthy","falsey"]` {
		t.Fatal(string(result.Value))
	}
}

func TestVolumeBackupReadLocalFilterOrderIsLazyAfterEagerBoolNormalization(t *testing.T) {
	for _, tc := range []struct {
		filter string
		fail   bool
	}{
		{`{"force":false,"metadata":{"nested":{"one":1}}}`, false},
		{`{"metadata":{"nested":{"one":1}},"force":false}`, true},
	} {
		t.Run(tc.filter, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := snapshotReadContractClient(cloud)
			var calls atomic.Int32
			body := backupReadContractPage(`[{"id":"excluded","force":"false","metadata":{"nested":true}}]`, "")
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("X-Proof", "admitted")
				testcloud.JSON(w, 203, body)
			})
			result, err := blockstorage.ListVolumeBackups(backupReadContractContext(t), client, blockstorage.WithVolumeBackupListFilters(json.RawMessage(tc.filter)))
			var physical *resource.ResponseError
			if result == nil || len(result.Pages) != 1 || string(result.Pages[0].Body) != body || calls.Load() != 1 || errors.As(err, &physical) {
				t.Fatal(result, err, calls.Load())
			}
			if tc.fail {
				if !errors.Is(err, resource.ErrInvalidOption) || result.Value != nil || result.Backups != nil {
					t.Fatal(result, err)
				}
			} else if err != nil || string(result.Value) != "[]" || result.Backups == nil {
				t.Fatal(result, err)
			}
		})
	}
}
