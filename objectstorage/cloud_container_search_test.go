package objectstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestSwiftCloudContainerSearchCollectsEveryPageBeforeIdentifierAndMapping(t *testing.T) {
	fixture := testcloud.New(t)
	var queries []string
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		if r.Method != "GET" || r.Header.Get("Accept") != "application/json" {
			t.Error(r)
		}
		w.Header().Set("X-Account-Container-Count", "3")
		if r.URL.Query().Get("marker") == "beta" {
			testcloud.JSON(w, 203, `[{"name":"alpha-final","count":2}]`)
		} else {
			testcloud.JSON(w, 200, `[{"name":"alpha","count":1},{"name":"beta","count":1}]`)
		}
	})
	result, err := objectstorage.New(fixture.Client("object-store", "/account/")).SearchContainers(context.Background(), objectstorage.WithContainerSearchName("alpha*"), objectstorage.WithContainerSearchFiltersJSON(json.RawMessage(`{"count":2}`)))
	if err != nil || len(result.AllRecords) != 3 || len(result.Records) != 1 || string(result.Records[0].Resource.Body["name"]) != `"alpha-final"` || result.LastPage.StatusCode != 203 || !reflect.DeepEqual(queries, []string{"", "marker=beta"}) {
		t.Fatal(result, err, queries)
	}
}

func TestSwiftCloudContainerSearchNameUsesFnmatchAndDeclaredAliases(t *testing.T) {
	fixture := testcloud.New(t)
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `[{"name":"alpha"},{"id":"beta"},{"name":"alpha"},{"name":null},{"name":"literal*"}]`)
	})
	cloud := objectstorage.New(fixture.Client("object-store", "/account/"))
	for _, tc := range []struct {
		name  string
		count int
	}{{"", 5}, {"alpha", 2}, {"a[lp]*", 2}, {"beta", 1}, {"None", 1}, {"literal*", 1}, {"none", 0}} {
		r, e := cloud.SearchContainers(context.Background(), objectstorage.WithContainerSearchName(tc.name))
		if e != nil || len(r.Records) != tc.count {
			t.Fatal(tc.name, r, e)
		}
	}
}

func TestSwiftCloudContainerSearchFalseyAndLoadedMappingFilters(t *testing.T) {
	fixture := testcloud.New(t)
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `[{"name":"alpha","count":true,"bytes":9007199254740993},{"name":"beta","count":0}]`)
	})
	cloud := objectstorage.New(fixture.Client("object-store", "/account/"))
	for _, raw := range []string{"null", "false", "0", "1e-400", `""`, "[]", "{}"} {
		r, e := cloud.SearchContainers(context.Background(), objectstorage.WithContainerSearchFiltersJSON(json.RawMessage(raw)))
		if e != nil || len(r.Records) != 2 {
			t.Fatal(raw, r, e)
		}
	}
	for _, raw := range []string{`{"count":1}`, `{"bytes":9007199254740993}`, `{"bytes":9007199254740992.0}`} {
		r, e := cloud.SearchContainers(context.Background(), objectstorage.WithContainerSearchFiltersJSON(json.RawMessage(raw)))
		want := 1
		if raw == `{"bytes":9007199254740992.0}` {
			want = 0
		}
		if e != nil || len(r.Records) != want {
			t.Fatal(raw, r, e)
		}
	}
}

func TestSwiftCloudContainerSearchExpressionsReturnArbitraryJSONWithoutRowAssociation(t *testing.T) {
	fixture := testcloud.New(t)
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `[{"name":"alpha","count":2},{"name":"beta","count":1}]`)
	})
	cloud := objectstorage.New(fixture.Client("object-store", "/account/"))
	for _, tc := range []struct{ expression, want string }{{"length(@)", "2"}, {"[0].name", `"alpha"`}, {"[5]", "null"}, {"[].name", `["alpha","beta"]`}, {"{total:length(@)}", `{"total":2}`}} {
		r, e := cloud.SearchContainers(context.Background(), objectstorage.WithContainerSearchExpression(tc.expression))
		if e != nil || !r.Expression || r.Records != nil || string(r.Value) != tc.want || len(r.AllRecords) != 2 {
			t.Fatal(tc, r, e)
		}
	}
	r, e := cloud.SearchContainers(context.Background(), objectstorage.WithContainerSearchName("absent"), objectstorage.WithContainerSearchExpression("["))
	if e == nil || r.Value != nil || r.Records != nil || len(r.AllRecords) != 2 {
		t.Fatal(r, e)
	}
}

func TestSwiftCloudContainerSearchPreservesLazyMissingAndNestedMappingFailures(t *testing.T) {
	fixture := testcloud.New(t)
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `[{"name":"alpha","count":1,"foreign":true}]`)
	})
	cloud := objectstorage.New(fixture.Client("object-store", "/account/"))
	for _, tc := range []struct {
		name, filter string
		failure      bool
	}{{"", `{"count":0,"missing":true}`, false}, {"", `{"missing":true,"count":0}`, true}, {"absent", `{"missing":true}`, false}, {"", `{"location":{"project":{"id":null}}}`, false}, {"", `{"name":{}}`, false}, {"", `{"name":{"a":true}}`, true}, {"", `true`, true}, {"absent", `true`, false}, {"", `{"foreign":true}`, true}} {
		r, e := cloud.SearchContainers(context.Background(), objectstorage.WithContainerSearchName(tc.name), objectstorage.WithContainerSearchFiltersJSON(json.RawMessage(tc.filter)))
		if (e != nil) != tc.failure || r == nil || len(r.AllRecords) != 1 {
			t.Fatal(tc, r, e)
		}
		if tc.failure && r.Value != nil {
			t.Fatal(r)
		}
	}
}

func TestSwiftCloudContainerSearchLocationIsOwnedAndRemoteWireCannotReplaceIt(t *testing.T) {
	fixture := testcloud.New(t)
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `[{"name":"alpha","location":{"cloud":"foreign"}}]`)
	})
	cloudName, region, project := "selected", "RegionOne", "project-name"
	location := resource.CloudLocation{Cloud: &cloudName, RegionName: &region, Project: resource.CloudProject{ID: json.RawMessage(`"project-id"`), Name: &project}}
	option := objectstorage.WithContainerSearchLocation(location)
	cloudName = "caller"
	location.Project.ID[1] = 'X'
	result, err := objectstorage.New(fixture.Client("object-store", "/account/")).SearchContainers(context.Background(), option, objectstorage.WithContainerSearchFiltersJSON(json.RawMessage(`{"location":{"cloud":"selected","project":{"id":"project-id"}}}`)))
	if err != nil || len(result.Records) != 1 || string(result.Records[0].Wire.Body["location"]) != `{"cloud":"foreign"}` {
		t.Fatal(result, err)
	}
	result.Records[0].Resource.Body["name"][1] = 'X'
	result.Records[0].Envelope[0] = 'X'
	result.Records[0].Header.Set("X-Probe", "changed")
	if string(result.AllRecords[0].Resource.Body["name"]) != `"alpha"` || result.AllRecords[0].Envelope[0] != '[' || result.AllRecords[0].Header.Get("X-Probe") != "" || string(result.Value) == "" {
		t.Fatal(result)
	}
}

func TestSwiftCloudContainerSearchMalformedLatePagePreventsSuccessfulPartialResult(t *testing.T) {
	fixture := testcloud.New(t)
	calls := 0
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("X-Account-Container-Count", "2")
		w.Header().Set("X-Proof", fmt.Sprint(calls))
		if calls == 1 {
			testcloud.JSON(w, 200, `[{"name":"alpha"}]`)
		} else {
			testcloud.JSON(w, 203, `{"cls":"constructor-collision"}`)
		}
	})
	result, err := objectstorage.New(fixture.Client("object-store", "/account/")).SearchContainers(context.Background())
	if err == nil || result == nil || len(result.AllRecords) != 1 || result.Value != nil || result.Records != nil || result.LastPage.StatusCode != 203 || result.LastPage.Header.Get("X-Proof") != "2" || calls != 2 {
		t.Fatal(result, err, calls)
	}
}

func TestSwiftCloudContainerSearchSourceCapturePrecedesOptionsAndKeepsCancellationCause(t *testing.T) {
	fixture := testcloud.New(t)
	calls, callbacks := 0, 0
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls++; testcloud.JSON(w, 200, `[]`) })
	client := fixture.Client("object-store", "/account/")
	wrong := *client
	wrong.Type = "image"
	option := func(*objectstorage.ContainerSearchOpts) error { callbacks++; return nil }
	if r, e := objectstorage.New(&wrong).SearchContainers(context.Background(), option); r == nil || !errors.Is(e, resource.ErrUnsupported) {
		t.Fatal(r, e)
	}
	if callbacks != 0 || calls != 0 {
		t.Fatal(callbacks, calls)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause, optionCause := errors.New("cancel callback"), errors.New("option failure")
	if r, e := objectstorage.New(client).SearchContainers(ctx, func(*objectstorage.ContainerSearchOpts) error { cancel(cause); return optionCause }); r == nil || !errors.Is(e, context.Canceled) || !errors.Is(e, cause) || !errors.Is(e, optionCause) || calls != 0 {
		t.Fatal(r, e, calls)
	}
	client.ResourceBase = ""
	if _, e := objectstorage.New(client).SearchContainers(context.Background(), func(*objectstorage.ContainerSearchOpts) error { client.ResourceBase = client.Endpoint; return nil }); !errors.Is(e, resource.ErrInvalidOption) || calls != 0 {
		t.Fatal(e, calls)
	}
}

func TestSwiftCloudContainerSearchOwnsRawFiltersAndHeaderNilReset(t *testing.T) {
	fixture := testcloud.New(t)
	var kept *objectstorage.ContainerSearchOpts
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		*kept.Filters = json.RawMessage(`true`)
		// This retained callback state deliberately ended with Headers=nil.
		// Replace that caller-owned map; the next option received a separate
		// initialized snapshot and must keep its own request header.
		kept.Headers = map[string]string{"X-Probe": "late"}
		if r.Header.Get("X-Probe") != "owned" {
			t.Error(r.Header)
		}
		testcloud.JSON(w, 200, `[{"name":"alpha","count":1}]`)
	})
	filter := json.RawMessage(`{"count":1}`)
	option := objectstorage.WithContainerSearchFiltersJSON(filter)
	filter[2] = 'X'
	result, err := objectstorage.New(fixture.Client("object-store", "/account/")).SearchContainers(context.Background(), option, func(c *objectstorage.ContainerSearchOpts) error { c.Headers = nil; kept = c; return nil }, objectstorage.WithContainerSearchHeaders(map[string]string{"X-Probe": "owned"}))
	if err != nil || len(result.Records) != 1 {
		t.Fatal(result, err)
	}
}

func TestSwiftCloudContainerSearchLoadedHeaderBoolViewKeepsRawWireAndNull(t *testing.T) {
	fixture := testcloud.New(t)
	fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `[{"name":"underflow","is_newest":1e-400,"is_content_type_detected":null},{"name":"string","is_newest":"false"}]`)
	})
	result, err := objectstorage.New(fixture.Client("object-store", "/account/")).SearchContainers(context.Background(), objectstorage.WithContainerSearchFiltersJSON(json.RawMessage(`{"is_newest":false}`)))
	if err != nil || len(result.Records) != 1 || string(result.Records[0].Resource.Body["name"]) != `"underflow"` || string(result.Records[0].Resource.Body["is_newest"]) != "false" || string(result.Records[0].Resource.Body["is_content_type_detected"]) != "null" || string(result.Records[0].Wire.Body["is_newest"]) != "1e-400" || string(result.AllRecords[1].Resource.Body["is_newest"]) != "true" {
		t.Fatal(result, err)
	}
}

func TestSwiftCloudContainerSearchPhysicalLatePageKeepsNative401AndStopsHooks(t *testing.T) {
	for _, status := range []int{200, 203, 401, 404, 429, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			fixture := testcloud.New(t)
			calls, retry, backoff, renew := 0, 0, 0, 0
			const actual = `[{"name":"broken"}]`
			fixture.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("X-Account-Container-Count", "2")
				if calls == 1 {
					testcloud.JSON(w, 200, `[{"name":"first"}]`)
					return
				}
				w.Header().Set("X-Proof", "last physical")
				w.Header().Set("Content-Length", strconv.Itoa(len(actual)+9))
				w.WriteHeader(status)
				_, _ = io.WriteString(w, actual)
			})
			fixture.Provider.ReauthFunc = func(context.Context) error { renew++; return nil }
			fixture.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retry++
				return nil
			}
			fixture.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
				backoff++
				return nil
			}
			fixture.Provider.MaxBackoffRetries = 3
			result, err := objectstorage.New(fixture.Client("object-store", "/account/")).SearchContainers(context.Background())
			if !errors.Is(err, io.ErrUnexpectedEOF) || result == nil || result.Value != nil || result.Records != nil || len(result.AllRecords) != 1 || result.LastPage == nil || result.LastPage.Resource != nil || result.LastPage.StatusCode != status || string(result.LastPage.Envelope) != actual || result.LastPage.Header.Get("X-Proof") != "last physical" || calls != 2 || retry != 0 || backoff != 0 || renew != 0 {
				t.Fatal(result, err, calls, retry, backoff, renew)
			}
			if status == 401 {
				var native *gophercloud.ErrUnableToReauthenticate
				var original gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || !errors.As(native.ErrOriginal, &original) || original.Actual != 401 || string(original.Body) != actual || !errors.Is(native.ErrReauth, io.ErrUnexpectedEOF) {
					t.Fatal(err, native)
				}
			}
		})
	}
}
