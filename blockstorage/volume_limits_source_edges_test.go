package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestGetVolumeLimitsProjectListConstructorCollisionsDifferFromMemberResponseFields(t *testing.T) {
	for _, key := range []string{"connection", "microversion", "_synchronized"} {
		for _, value := range []string{"null", "false", `{"raw":true}`} {
			for _, member := range []bool{false, true} {
				stage := "list"
				if member {
					stage = "member"
				}
				t.Run(stage+"/"+key+"/"+value, func(t *testing.T) {
					cloud := testcloud.New(t)
					cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
					var calls atomic.Int32
					input := "name/only"
					body := `{"projects":[{"name":"unused-other","` + key + `":` + value + `},{"name":"name/only","id":"requested"}]}`
					if member {
						input = "ref"
						body = `{"project":{"id":"requested","` + key + `":` + value + `}}`
					}
					cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
						if calls.Add(1) == 1 {
							path := glProjectsPath
							if member {
								path += "/ref"
							}
							glIdentityWire(t, r, path, "test-token")
							w.Header().Set("X-Proof", "project phase")
							testcloud.JSON(w, 203, body)
							return
						}
						volumeReadContractWire(t, r, glLimitsPath, "test-token")
						if !member || r.URL.RawQuery != "project_id=requested" {
							t.Error("failed list started limits or member query changed", r.URL)
						}
						testcloud.JSON(w, 200, `{"limits":{}}`)
					})
					result, err := glCall(context.Background(), cinder, identity, input)
					var physical *resource.ResponseError
					if member {
						if err != nil || result == nil || result.Project == nil || result.Project.Project == nil || result.Project.Observed == nil || result.Project.SeededID || len(result.Project.Pages) != 0 || result.Limits == nil || calls.Load() != 2 || string(result.Project.Project.Body[key]) != value || string(result.RequestedProjectID) != `"requested"` {
							t.Fatal(result, err, calls.Load())
						}
						return
					}
					if result == nil || result.Project == nil || result.Project.Project != nil || result.Project.Observed != nil || len(result.Project.Pages) != 1 || result.RequestedProjectID != nil || result.Observed != nil || result.Limits != nil || result.Value != nil || calls.Load() != 1 || !errors.As(err, &physical) || physical.StatusCode != 203 || string(physical.Body) != body || physical.Header.Get("X-Proof") != "project phase" || string(result.Project.Pages[0].Body) != body {
						t.Fatal(result, err, calls.Load())
					}
					glOperation(t, err)
				})
			}
		}
	}
}

func TestGetVolumeLimitsProjectPagingRemovesPriorMarkerLimitAndDropsAdvertisedBlankValues(t *testing.T) {
	cloud := testcloud.New(t)
	cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 4 {
			volumeReadContractWire(t, r, glLimitsPath, "test-token")
			if r.URL.RawQuery != "project_id=requested" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, `{"limits":{}}`)
			return
		}
		glIdentityWire(t, r, glProjectsPath, "test-token")
		query := r.URL.Query()
		var want map[string][]string
		var body string
		switch n {
		case 1:
			want = map[string][]string{"name": {"name/only"}}
			body = `{"projects":[{"name":"other-1"}],"links":{"next":"?marker=first&limit=2&keep=one&blank="}}`
		case 2:
			want = map[string][]string{"name": {"name/only"}, "marker": {"first"}, "limit": {"2"}, "keep": {"one"}}
			body = `{"projects":[{"name":"other-2"}],"links":{"next":"?cursor=third&keep=&name=&limit=&blank=&empty"}}`
		case 3:
			want = map[string][]string{"name": {"name/only"}, "keep": {"one"}, "cursor": {"third"}}
			body = `{"projects":[{"name":"name/only","id":"requested"}],"links":{"next":null}}`
		default:
			t.Error("workflow sent an extra physical request", r.URL)
			w.WriteHeader(500)
			return
		}
		if !reflect.DeepEqual(map[string][]string(query), want) {
			t.Error("source query state differs", n, query, want)
		}
		w.Header().Set("X-Proof", string(rune('0'+n)))
		testcloud.JSON(w, 203, body)
	})
	result, err := glCall(context.Background(), cinder, identity, "name/only")
	if err != nil || result == nil || result.Project == nil || result.Project.Project == nil || result.Project.Observed != nil || len(result.Project.Pages) != 3 || result.Project.SeededID || string(result.Project.ID) != `"requested"` || result.Limits == nil || calls.Load() != 4 {
		t.Fatal(result, err, calls.Load())
	}
	for index, page := range result.Project.Pages {
		if page.StatusCode != 203 || page.Header.Get("X-Proof") != string(rune('1'+index)) || !json.Valid(page.Body) {
			t.Fatal(index, page)
		}
	}
}

func TestGetVolumeLimitsProjectLinksAreConsumedInOrderAndStopAtSelectedHref(t *testing.T) {
	cases := []struct {
		name, links, top string
		success          bool
	}{
		{"consumed scalar prefix", `[false,{"rel":"next","href":"?marker=next"}]`, `"?marker=next"`, false},
		{"consumed null prefix", `[null,{"rel":"next","href":"?marker=next"}]`, `"?marker=next"`, false},
		{"next without href does not stop", `[{"rel":"next"},false]`, `"?marker=next"`, false},
		{"truthy nonstring selected href", `[{"rel":"next","href":true},false]`, `"?marker=next"`, false},
		{"selected next bypasses invalid tail", `[{"rel":"next","href":"?marker=next"},false]`, `"?marker=wrong"`, true},
		{"false href stops before top fallback", `[{"rel":"next","href":false},false]`, `"?marker=next"`, true},
		{"null href stops before top fallback", `[{"rel":"next","href":null},false]`, `"?marker=next"`, true},
		{"empty string href stops before top fallback", `[{"rel":"next","href":""},false]`, `"?marker=next"`, true},
		{"empty list href stops before top fallback", `[{"rel":"next","href":[]},false]`, `"?marker=next"`, true},
		{"Keystone dict repair has declared priority", `{"next":"?marker=next","ignored":false}`, `"?marker=wrong"`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cinder, identity := volumeReadContractClient(cloud), glIdentity(cloud)
			var calls atomic.Int32
			firstBody := `{"projects":[{"name":"other"}],"links":` + tc.links + `,"next":` + tc.top + `}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					glIdentityWire(t, r, glProjectsPath, "test-token")
					w.Header().Set("X-Proof", "consumed first page")
					w.Header().Set("Link", "unused invalid HTTP link")
					testcloud.JSON(w, 203, firstBody)
				case 2:
					glIdentityWire(t, r, glProjectsPath, "test-token")
					if query := r.URL.Query(); query.Get("name") != "name/only" || query.Get("marker") != "next" || len(query) != 2 {
						t.Error("selected continuation changed", r.URL)
					}
					testcloud.JSON(w, 203, `{"projects":[{"name":"name/only","id":"requested"}]}`)
				case 3:
					volumeReadContractWire(t, r, glLimitsPath, "test-token")
					if r.URL.RawQuery != "project_id=requested" {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, `{"limits":{}}`)
				default:
					t.Error("unexpected extra physical request", r.URL)
					w.WriteHeader(500)
				}
			})
			result, err := glCall(context.Background(), cinder, identity, "name/only")
			if tc.success {
				if err != nil || result == nil || result.Project == nil || result.Project.Project == nil || len(result.Project.Pages) != 2 || result.Project.Observed != nil || result.Limits == nil || calls.Load() != 3 {
					t.Fatal(result, err, calls.Load())
				}
				return
			}
			var physical *resource.ResponseError
			if result == nil || result.Project == nil || result.Project.Project != nil || len(result.Project.Pages) != 1 || result.Project.Observed != nil || result.Observed != nil || result.RequestedProjectID != nil || result.Limits != nil || result.Value != nil || calls.Load() != 1 || !errors.As(err, &physical) || physical.StatusCode != 203 || string(physical.Body) != firstBody || physical.Header.Get("X-Proof") != "consumed first page" {
				t.Fatal(result, err, calls.Load())
			}
			glOperation(t, err)
		})
	}
}
