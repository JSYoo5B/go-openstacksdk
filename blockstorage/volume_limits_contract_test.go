package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	identityV3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

const glIdentityBase = "/proxy/identity/v3/"
const glProjectsPath = glIdentityBase + "projects"
const glLimitsPath = volumeReadContractBase + "limits"

func glIdentity(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("identity", "/catalog/identity/")
	client.ResourceBase = cloud.Server.URL + glIdentityBase
	client.MoreHeaders = map[string]string{"x-identity": "entry"}
	return client
}

func glIdentityWire(t *testing.T, r *http.Request, path, token string) {
	t.Helper()
	if r.Method != http.MethodGet || r.URL.Path != path || r.Header.Get("X-Identity") != "entry" || r.Header.Get("X-Auth-Token") != token || r.Header.Get("OpenStack-API-Version") != "" {
		t.Error("owned Identity request changed", r.Method, r.URL, r.Header)
	}
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) != 0 {
			t.Error("Identity GET has body", string(body), err)
		}
	}
}

func glCall(ctx context.Context, cinder, identity *gophercloud.ServiceClient, name string, options ...blockstorage.GetVolumeLimitsOption) (*blockstorage.GetVolumeLimitsResult, error) {
	return blockstorage.GetVolumeLimits(ctx, cinder, identity, blockstorage.GetVolumeLimitsRequest{NameOrID: name}, options...)
}

func glOperation(t *testing.T, err error) {
	t.Helper()
	var operation *resource.OperationError
	if !errors.As(err, &operation) || operation.Operation != "GetVolumeLimits" || operation.Resource != "volume limits" || operation.Cause == nil {
		t.Fatal("public operation context missing", err)
	}
}

func glObject(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	return volumeReadContractFields(t, raw)
}

func glArray(t *testing.T, raw json.RawMessage) []json.RawMessage {
	t.Helper()
	var rows []json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(string(raw), err)
	}
	return rows
}

func glJSONSame(a, b json.RawMessage) bool {
	var x, y any
	decode := func(raw json.RawMessage, into *any) error {
		d := json.NewDecoder(strings.NewReader(string(raw)))
		d.UseNumber()
		return d.Decode(into)
	}
	return decode(a, &x) == nil && decode(b, &y) == nil && reflect.DeepEqual(x, y)
}

func TestGetVolumeLimitsDefaultSkipsIdentityAndDoesNotInferScopeQuery(t *testing.T) {
	cloud := testcloud.New(t)
	cinder := volumeReadContractClient(cloud)
	auth := &identityV3.CreateResult{}
	auth.Header = http.Header{"X-Subject-Token": {"scope-token"}}
	auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": "authenticated-project"}}}
	if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
		t.Fatal(err)
	}
	ignored := &gophercloud.ServiceClient{Type: "not-identity"}
	var calls atomic.Int32
	body := `{"limits":{"unknown":9007199254740993,"location":"wire","project_id":"wire-project","zone":"wire-zone"}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		volumeReadContractWire(t, r, glLimitsPath, "scope-token")
		if r.URL.RawQuery != "" {
			t.Error("current scope became query", r.URL)
		}
		auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": "during-HTTP-project"}}}
		calls.Add(1)
		w.Header().Set("X-Proof", "limits")
		testcloud.JSON(w, 203, body)
	})
	result, err := glCall(context.Background(), cinder, ignored, "")
	if err != nil || result == nil || result.Project != nil || result.RequestedProjectID != nil || result.Observed == nil || result.Limits == nil || calls.Load() != 1 {
		t.Fatal(result, err, calls.Load())
	}
	view := glObject(t, result.Value)
	location := glObject(t, view["location"])
	if len(view) != 5 || string(view["absolute"]) != "null" || string(view["rate"]) != "null" || string(view["id"]) != "null" || string(location["zone"]) != "null" || string(glObject(t, location["project"])["id"]) != `"authenticated-project"` || result.Limits.StatusCode != 203 || string(result.Limits.Body["unknown"]) != "9007199254740993" || string(result.Observed.Body) != body {
		t.Fatal(string(result.Value), result)
	}
	if _, ok := view["unknown"]; ok {
		t.Fatal("unknown field entered normalized Limits")
	}
}

func TestGetVolumeLimitsTruthyInputResolvesIdentityBeforeLimitsWithoutTrimming(t *testing.T) {
	for _, input := range []string{"ref", "123e4567-e89b-12d3-a456-426614174000", "   ", "name/with space"} {
		t.Run(input, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
			var calls atomic.Int32
			listOnly := strings.ContainsAny(input, "/ ")
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					path := glProjectsPath + "/" + input
					if listOnly {
						path = glProjectsPath
					}
					glIdentityWire(t, r, path, "test-token")
					if listOnly {
						if q := r.URL.Query(); q.Get("name") != input || len(q) != 1 {
							t.Error("list hint trimmed or domain inferred", r.URL)
						}
						row, _ := json.Marshal(map[string]any{"id": "actual-ID", "name": input})
						testcloud.JSON(w, 203, `{"projects":[`+string(row)+`]}`)
					} else {
						if r.URL.RawQuery != "" {
							t.Error(r.URL)
						}
						testcloud.JSON(w, 201, `{"project":{"id":"actual-ID"}}`)
					}
					return
				}
				volumeReadContractWire(t, r, glLimitsPath, "test-token")
				if r.URL.RawQuery != "project_id=actual-ID" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"limits":{}}`)
			})
			result, err := glCall(context.Background(), cinder, identity, input)
			if err != nil || result == nil || result.Project == nil || result.Project.Project == nil || result.Project.SeededID || string(result.RequestedProjectID) != `"actual-ID"` || calls.Load() != 2 {
				t.Fatal(result, err, calls.Load())
			}
			if listOnly && (result.Project.Observed != nil || len(result.Project.Pages) != 1) || !listOnly && (result.Project.Observed == nil || len(result.Project.Pages) != 0) {
				t.Fatal("lookup phase evidence changed", result.Project)
			}
		})
	}
}

func TestGetVolumeLimitsMemberSeedIsExplicitAndNeverFabricatesRawProjectID(t *testing.T) {
	for _, body := range []string{`{"project":{}}`, `{"name":"flat","extension":9007199254740993}`, "", `{"unfinished":`} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					glIdentityWire(t, r, glProjectsPath+"/ref", "test-token")
					w.Header().Set("X-Proof", "seed")
					testcloud.JSON(w, 203, body)
					return
				}
				volumeReadContractWire(t, r, glLimitsPath, "test-token")
				if r.URL.RawQuery != "project_id=ref" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{}`)
			})
			result, err := glCall(context.Background(), cinder, identity, "ref")
			if err != nil || result == nil || result.Project == nil || !result.Project.SeededID || string(result.Project.ID) != `"ref"` || string(result.RequestedProjectID) != `"ref"` || result.Project.Project == nil || result.Project.Observed == nil || string(result.Project.Observed.Body) != body || calls.Load() != 2 {
				t.Fatal(result, err, calls.Load())
			}
			if _, present := result.Project.Project.Body["id"]; present {
				t.Fatal("lookup seed entered actual raw project fields", result.Project.Project)
			}
		})
	}
}

func TestGetVolumeLimitsRawProjectIDsUseRequestsTwoLevelQueryExpansion(t *testing.T) {
	cases := []struct{ id, query string }{
		{`null`, ""}, {`false`, "project_id=False"}, {`true`, "project_id=True"}, {`0`, "project_id=0"},
		{`9007199254740993`, "project_id=9007199254740993"}, {`1.0`, "project_id=1.0"}, {`1e400`, "project_id=inf"},
		{`""`, "project_id="}, {`"a /?&= Ω\n"`, "project_id=a+%2F%3F%26%3D+%CE%A9%0A"}, {`[]`, ""}, {`{}`, ""},
		{`["a",null,false,""]`, "project_id=a&project_id=False&project_id="},
		{`{"b":1,"a":false,"b":2}`, "project_id=b&project_id=a"},
		{`[[1,2],[],null]`, "project_id=1&project_id=2"}, {`[{"a":1,"b":2}]`, "project_id=a&project_id=b"},
		{`[[null,false]]`, "project_id=None&project_id=False"},
		{`[[[1,false],{"a":true}]]`, "project_id=%5B1%2C+False%5D&project_id=%7B%27a%27%3A+True%7D"},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					glIdentityWire(t, r, glProjectsPath+"/ref", "test-token")
					testcloud.JSON(w, 200, `{"project":{"id":`+tc.id+`}}`)
					return
				}
				volumeReadContractWire(t, r, glLimitsPath, "test-token")
				if r.URL.RawQuery != tc.query {
					t.Error("query differs from Requests JSON domain", r.URL.RawQuery, tc.query)
				}
				testcloud.JSON(w, 200, `{"limits":{}}`)
			})
			result, err := glCall(context.Background(), cinder, identity, "ref")
			if err != nil || result == nil || result.Project == nil || result.Project.SeededID || string(result.Project.ID) != tc.id || string(result.RequestedProjectID) != tc.id || calls.Load() != 2 {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestGetVolumeLimitsFallbackListMissingIDIsNullAndOwnsMergedPaging(t *testing.T) {
	cloud := testcloud.New(t)
	cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			glIdentityWire(t, r, glProjectsPath+"/ref", "test-token")
			testcloud.JSON(w, 404, `{"error":"missing member"}`)
		case 2:
			glIdentityWire(t, r, glProjectsPath, "test-token")
			if r.URL.Query().Get("name") != "ref" || len(r.URL.Query()) != 1 {
				t.Error(r.URL)
			}
			w.Header().Set("X-Proof", "first")
			testcloud.JSON(w, 201, `{"projects":[{"id":"other"}],"links":{"next":"?marker=next"}}`)
		case 3:
			glIdentityWire(t, r, glProjectsPath, "test-token")
			if q := r.URL.Query(); q.Get("name") != "ref" || q.Get("marker") != "next" || len(q) != 2 {
				t.Error(r.URL)
			}
			w.Header().Set("X-Proof", "second")
			testcloud.JSON(w, 203, `{"projects":{"name":"ref","self":null,"extension":false},"links":{"next":null}}`)
		case 4:
			volumeReadContractWire(t, r, glLimitsPath, "test-token")
			if r.URL.RawQuery != "" {
				t.Error("absent list ID seeded or path validated", r.URL)
			}
			testcloud.JSON(w, 200, `{"limits":{}}`)
		default:
			t.Error("unexpected request", r.URL)
			w.WriteHeader(500)
		}
	})
	result, err := glCall(context.Background(), cinder, identity, "ref")
	if err != nil || result == nil || result.Project == nil || result.Project.Project == nil || result.Project.SeededID || string(result.Project.ID) != "null" || string(result.RequestedProjectID) != "null" || result.Project.Observed != nil || len(result.Project.Pages) != 2 || result.Project.Pages[1].Header.Get("X-Proof") != "second" || calls.Load() != 4 {
		t.Fatal(result, err, calls.Load())
	}
	if _, ok := result.Project.Project.Body["id"]; ok {
		t.Fatal("list absent ID fabricated")
	}
}

func TestGetVolumeLimitsProjectAmbiguityStopsBeforeUnusedInvalidRowAndLink(t *testing.T) {
	cloud := testcloud.New(t)
	cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
	var calls atomic.Int32
	body := `{"projects":[{"name":"name/only","id":"one"},{"name":"name/only","id":"two"},false],"links":{"next":false}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		glIdentityWire(t, r, glProjectsPath, "test-token")
		testcloud.JSON(w, 203, body)
	})
	result, err := glCall(context.Background(), cinder, identity, "name/only")
	if result == nil || result.Project == nil || result.Project.Project != nil || len(result.Project.Pages) != 1 || result.Project.Observed != nil || result.RequestedProjectID != nil || result.Observed != nil || result.Value != nil || !errors.Is(err, resource.ErrAmbiguous) || calls.Load() != 1 {
		t.Fatal(result, err, calls.Load())
	}
	glOperation(t, err)
}

func TestGetVolumeLimitsProjectCompletedAbsenceStopsBeforeLimitsAndUnusedEmptyPageLink(t *testing.T) {
	cloud := testcloud.New(t)
	cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		glIdentityWire(t, r, glProjectsPath, "test-token")
		if calls.Add(1) == 1 {
			w.Header().Set("Link", "<"+cloud.Server.URL+glProjectsPath+"?marker=next>; rel=\"next\"")
			testcloud.JSON(w, 200, `{"projects":[{"name":"other"}]}`)
			return
		}
		if r.URL.Query().Get("marker") != "next" || r.URL.Query().Get("name") != "name/only" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"projects":[],"links":{"next":false}}`)
	})
	result, err := glCall(context.Background(), cinder, identity, "name/only")
	if result == nil || result.Project == nil || len(result.Project.Pages) != 2 || result.Project.Project != nil || result.RequestedProjectID != nil || result.Observed != nil || result.Value != nil || !errors.Is(err, resource.ErrNotFound) || calls.Load() != 2 {
		t.Fatal(result, err, calls.Load())
	}
}

func TestGetVolumeLimitsNestedDescriptorsPreserveNullableShapesAndUntypedPrecision(t *testing.T) {
	cloud := testcloud.New(t)
	cinder := volumeReadContractClient(cloud)
	body := `{"limits":{"id":false,"name":[1],"absolute":{"maxTotalBackupGigabytes":true,"maxTotalBackups":9007199254740993,"maxTotalSnapshots":3.9,"maxTotalVolumeGigabytes":"１２","maxTotalVolumes":"-3","totalBackupGigabytesUsed":" 4 ","totalBackupsUsed":[],"totalGigabytesUsed":{"x":1},"totalSnapshotsUsed":null,"totalVolumesUsed":"٤","unknown":9},"rate":[null,false,[],{}, {"regex":{"raw":9007199254740993},"uri":false,"limit":[null,0,[],{}, {"next-available":false,"remaining":"7","unit":[1],"value":2.9,"verb":{"v":true},"id":0,"name":"n","extension":1}]}],"extension":1}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		volumeReadContractWire(t, r, glLimitsPath, "test-token")
		testcloud.JSON(w, 200, body)
	})
	result, err := glCall(context.Background(), cinder, nil, "")
	if err != nil || result == nil || result.Limits == nil {
		t.Fatal(result, err)
	}
	view := glObject(t, result.Value)
	absolute := glObject(t, view["absolute"])
	want := map[string]string{"max_total_backup_gigabytes": "true", "max_total_backups": "9007199254740993", "max_total_snapshots": "3", "max_total_volume_gigabytes": "12", "max_total_volumes": "0", "total_backup_gigabytes_used": "0", "total_backups_used": "0", "total_gigabytes_used": "0", "total_snapshots_used": "null", "total_volumes_used": "4"}
	if len(view) != 5 || len(absolute) != 13 || string(view["id"]) != "false" || string(view["name"]) != "[1]" {
		t.Fatal(string(result.Value))
	}
	for key, value := range want {
		if string(absolute[key]) != value {
			t.Error(key, string(absolute[key]), value)
		}
	}
	rate := glArray(t, view["rate"])
	if len(rate) != 5 || string(rate[0]) != "{}" || string(rate[1]) != "{}" || string(rate[2]) != "{}" || len(glObject(t, rate[3])) != 6 {
		t.Fatal(string(view["rate"]))
	}
	group := glObject(t, rate[4])
	limits := glArray(t, group["limits"])
	if len(group) != 6 || len(limits) != 5 || string(limits[0]) != "{}" || string(limits[1]) != "{}" || string(limits[2]) != "{}" || len(glObject(t, limits[3])) != 8 {
		t.Fatal(string(rate[4]))
	}
	limit := glObject(t, limits[4])
	if len(limit) != 8 || string(limit["remaining"]) != "7" || string(limit["value"]) != "2" || string(limit["next_available"]) != "false" || string(limit["unit"]) != "[1]" || string(limit["verb"]) != `{"v":true}` || string(limit["location"]) != "null" || string(group["regex"]) != `{"raw":9007199254740993}` {
		t.Fatal(string(limits[4]), string(rate[4]))
	}
	if _, ok := result.Limits.Body["extension"]; !ok {
		t.Fatal("raw extension discarded")
	}
}

func TestGetVolumeLimitsAliasTraversalUsesCollapsedDictionaryInsertionOrder(t *testing.T) {
	cases := []struct {
		row                       string
		absolute, remaining, next string
	}{
		{`{"absolute":{"max_total_volumes":1,"maxTotalVolumes":2,"max_total_volumes":3},"rate":{"limits":[{"remaining":1}],"limit":[{"remaining":2,"next_available":1,"next-available":2,"next_available":3}],"limits":[{"remaining":3}]}}`, "2", "2", "2"},
		{`{"absolute":{"maxTotalVolumes":1,"max_total_volumes":2,"maxTotalVolumes":3},"rate":{"limit":[{"remaining":1}],"limits":[{"remaining":2,"next-available":1,"next_available":2,"next-available":3}],"limit":[{"remaining":3}]}}`, "2", "2", "2"},
	}
	for _, tc := range cases {
		t.Run(tc.row, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder := volumeReadContractClient(cloud)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"limits":`+tc.row+`}`) })
			result, err := glCall(context.Background(), cinder, nil, "")
			if err != nil {
				t.Fatal(result, err)
			}
			view := glObject(t, result.Value)
			absolute := glObject(t, view["absolute"])
			rate := glArray(t, view["rate"])
			group := glObject(t, rate[0])
			limit := glObject(t, glArray(t, group["limits"])[0])
			if string(absolute["max_total_volumes"]) != tc.absolute || string(limit["remaining"]) != tc.remaining || string(limit["next_available"]) != tc.next {
				t.Fatal(string(result.Value))
			}
		})
	}
}

func TestGetVolumeLimitsNestedLocationPresenceAndTopLocationHaveDifferentOwnershipRules(t *testing.T) {
	cloud := testcloud.New(t)
	cinder := volumeReadContractClient(cloud)
	cloudName, region, projectName := "owned-cloud", "owned-region", "current-name"
	location := resource.CloudLocation{Cloud: &cloudName, RegionName: &region, Zone: json.RawMessage(`{"zone":false}`), Project: resource.CloudProject{ID: json.RawMessage(`"current-project"`), Name: &projectName}}
	body := `{"limits":{"location":"wire-top","absolute":{"location":false,"unknown":1},"rate":[{"location":{"raw":true}},{"location":"ignored","regex":null},{"limit":[{"location":[]},{"location":"ignored","remaining":null}]}]}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, body) })
	result, err := glCall(context.Background(), cinder, nil, "", blockstorage.WithGetVolumeLimitsLocation(location))
	if err != nil {
		t.Fatal(result, err)
	}
	view := glObject(t, result.Value)
	top := glObject(t, view["location"])
	absolute := glObject(t, view["absolute"])
	groups := glArray(t, view["rate"])
	if string(top["cloud"]) != `"owned-cloud"` || string(top["zone"]) != `{"zone":false}` || string(glObject(t, top["project"])["id"]) != `"current-project"` || string(absolute["location"]) != "false" || string(glObject(t, groups[0])["location"]) != `{"raw":true}` || string(glObject(t, groups[1])["location"]) != "null" {
		t.Fatal(string(result.Value))
	}
	items := glArray(t, glObject(t, groups[2])["limits"])
	if string(glObject(t, items[0])["location"]) != "[]" || string(glObject(t, items[1])["location"]) != "null" {
		t.Fatal(string(groups[2]))
	}
}

func TestGetVolumeLimitsAcceptedFlatAndMalformedJSONProduceOwnedNullableResource(t *testing.T) {
	cases := []struct {
		code int
		body string
	}{{399, ""}, {399, `{"unfinished":`}, {399, `{"absolute":false,"rate":0,"self":"ignored","extension":9007199254740993}`}, {204, ""}, {304, ""}}
	for _, tc := range cases {
		t.Run(http.StatusText(tc.code)+tc.body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder := volumeReadContractClient(cloud)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				volumeReadContractWire(t, r, glLimitsPath, "test-token")
				w.Header().Set("X-Proof", "accepted")
				testcloud.JSON(w, tc.code, tc.body)
			})
			result, err := glCall(context.Background(), cinder, nil, "")
			if err != nil || result == nil || result.Limits == nil || result.Observed == nil || result.Limits.StatusCode != tc.code || result.Observed.StatusCode != tc.code || string(result.Observed.Body) != tc.body || result.Limits.Header.Get("X-Proof") != "accepted" {
				t.Fatal(result, err)
			}
			view := glObject(t, result.Value)
			if len(view) != 5 {
				t.Fatal(string(result.Value))
			}
			if strings.Contains(tc.body, `"absolute"`) {
				if string(view["absolute"]) != "{}" || string(view["rate"]) != "[{}]" {
					t.Fatal(string(result.Value))
				}
			} else if string(view["absolute"]) != "null" || string(view["rate"]) != "null" || len(result.Limits.Body) != 0 {
				t.Fatal(string(result.Value), result.Limits.Body)
			}
		})
	}
}

func TestGetVolumeLimitsInvalidAcceptedShapesAndDescriptorsFailAtomicallyWithCurrentProof(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `false`, `{"limits":null}`, `{"limits":[]}`, `{"limits":{"absolute":{"maxTotalVolumes":"²"}}}`, `{"limits":{"absolute":{"maxTotalVolumes":1e400}}}`, `{"limits":{"absolute":{"self":null}}}`, `{"limits":{"rate":[{"connection":true}]}}`, string([]byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'})} {
		t.Run(body, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder := volumeReadContractClient(cloud)
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Proof", "current")
				testcloud.JSON(w, 203, body)
			})
			result, err := glCall(context.Background(), cinder, nil, "")
			var physical *resource.ResponseError
			if result == nil || result.Limits != nil || result.Value != nil || result.Observed == nil || !errors.As(err, &physical) || physical.StatusCode != 203 || string(physical.Body) != body || physical.Header.Get("X-Proof") != "current" || string(result.Observed.Body) != body {
				t.Fatal(result, err)
			}
			glOperation(t, err)
			physical.Body[0] = '!'
			physical.Header.Set("X-Proof", "changed")
			if string(result.Observed.Body) != body || result.Observed.Header.Get("X-Proof") != "current" {
				t.Fatal("error proof aliases observed")
			}
		})
	}
}

func TestGetVolumeLimitsResultProofProjectAndNormalizedValueAreIndependent(t *testing.T) {
	cloud := testcloud.New(t)
	cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
	var calls atomic.Int32
	projectBody := `{"project":{"id":{"b":9007199254740993},"unknown":false}}`
	limitsBody := `{"limits":{"id":0,"absolute":{"maxTotalVolumes":5},"unknown":{"x":1}}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("X-Proof", "project")
			testcloud.JSON(w, 200, projectBody)
			return
		}
		w.Header().Set("X-Proof", "limits")
		testcloud.JSON(w, 203, limitsBody)
	})
	result, err := glCall(context.Background(), cinder, identity, "ref")
	if err != nil {
		t.Fatal(result, err)
	}
	value := string(result.Value)
	requested := string(result.RequestedProjectID)
	result.Project.ID[0] = '!'
	if string(result.Project.Project.Body["id"]) != requested || string(result.RequestedProjectID) != requested {
		t.Fatal("project ID evidence aliases another field")
	}
	result.Project.Project.Body["id"][0] = '!'
	result.Project.Observed.Body[0] = '!'
	result.Project.Observed.Header.Set("X-Proof", "mutated project")
	result.Limits.Body["absolute"][0] = '!'
	result.Limits.Header.Set("X-Proof", "mutated limits")
	result.Observed.Body[0] = '!'
	if string(result.Value) != value || string(result.RequestedProjectID) != requested || result.Observed.Header.Get("X-Proof") != "limits" || result.Limits.Header.Get("X-Proof") != "mutated limits" || result.Project.Project.Header.Get("X-Proof") != "project" {
		t.Fatal("result fields share ownership", result)
	}
	if !json.Valid(result.Value) || !json.Valid(result.RequestedProjectID) {
		t.Fatal("published JSON changed")
	}
}

func TestGetVolumeLimitsPreservesSelectedMicroversionWithoutProjectUpgradeOrCap(t *testing.T) {
	for _, mv := range []string{"", "3.0", "3.38", "3.39", "3.60", "latest"} {
		t.Run(mv, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
			cinder.Microversion = mv
			if mv != "" {
				cinder.MoreHeaders["openstack-api-version"] = "volume " + mv
				cinder.MoreHeaders["X-OpenStack-Volume-API-Version"] = mv
			}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					glIdentityWire(t, r, glProjectsPath+"/ref", "test-token")
					testcloud.JSON(w, 200, `{"project":{"id":"actual"}}`)
					return
				}
				want := ""
				if mv != "" {
					want = "volume " + mv
				}
				if r.URL.Path != glLimitsPath || r.URL.Query().Get("project_id") != "actual" || r.Header.Get("OpenStack-API-Version") != want || r.Header.Get("X-OpenStack-Volume-API-Version") != mv {
					t.Error(r.URL, r.Header)
				}
				testcloud.JSON(w, 200, `{"limits":{}}`)
			})
			result, err := glCall(context.Background(), cinder, identity, "ref")
			if err != nil || result == nil || result.Limits == nil || calls.Load() != 2 {
				t.Fatal(result, err, calls.Load())
			}
		})
	}
}

func TestGetVolumeLimitsFalseyNestedConstructorControlsDoNotBecomeBodyFields(t *testing.T) {
	cloud := testcloud.New(t)
	cinder := volumeReadContractClient(cloud)
	body := `{"limits":{"self":"ignored","absolute":{"connection":false,"_synchronized":{},"microversion":[],"location":"raw-location"},"rate":[{"connection":0.0,"location":false},{"limit":[{"connection":null,"location":[]},{"connection":[],"remaining":null,"location":"ignored"}]}]}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, body) })
	result, err := glCall(context.Background(), cinder, nil, "")
	if err != nil || result == nil {
		t.Fatal(result, err)
	}
	view := glObject(t, result.Value)
	absolute := glObject(t, view["absolute"])
	groups := glArray(t, view["rate"])
	items := glArray(t, glObject(t, groups[1])["limits"])
	if len(absolute) != 13 || string(absolute["location"]) != `"raw-location"` || string(glObject(t, groups[0])["location"]) != "false" || string(glObject(t, items[0])["location"]) != "[]" || string(glObject(t, items[1])["location"]) != "null" {
		t.Fatal(string(result.Value))
	}
}
