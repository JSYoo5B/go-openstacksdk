package instanceactions

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/testhelper"
)

func actionRecordsCloud(t *testing.T) (*testcloud.Cloud, *API) {
	t.Helper()
	cloud := testcloud.New(t)
	client := cloud.Client("compute", "/v2.1/p/")
	client.Microversion = "2.84"
	return cloud, New(client)
}
func actionRecordField(t *testing.T, value *ServerActionRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatalf("%s want=%s value=%+v", key, want, value)
	}
}
func TestActionRecordsSourceMROEventsProjectionAndActualWire(t *testing.T) {
	cloud, api := actionRecordsCloud(t)
	calls := 0
	body := `{"instanceActions":[{"request_id":"req","instance_uuid":"foreign","action":"reboot","start_time":"literal","events":[{"event":"compute_reboot","hostId":"hash","details":"provided","unknown":17},null,false,4],"unknown":9007199254740993,"self":"wire"}]}`
	cloud.Mux.HandleFunc("/v2.1/p/servers/server/os-instance-actions", func(w http.ResponseWriter, r *http.Request) {
		calls++
		testhelper.AssertEquals(t, "compute 2.84", r.Header.Get("OpenStack-API-Version"))
		testhelper.AssertEquals(t, "2.84", r.Header.Get("X-OpenStack-Nova-API-Version"))
		w.Header().Set("X-Proof", "actual")
		testcloud.JSON(w, 203, body)
	})
	rows, err := api.AllRecords(context.Background(), "server")
	if err != nil || len(rows) != 1 || calls != 1 {
		t.Fatal(rows, err, calls)
	}
	value := rows[0]
	actionRecordField(t, value, "id", `"req"`)
	actionRecordField(t, value, "start_time", `"literal"`)
	if value.ServerID != "server" || value.StatusCode != 203 || string(value.Envelope) != body || value.Resource.Body["instance_uuid"] != nil || value.Resource.Body["unknown"] != nil || string(value.Wire.Body["unknown"]) != "9007199254740993" {
		t.Fatal(value)
	}
	var events []map[string]json.RawMessage
	if err := json.Unmarshal(value.Resource.Body["events"], &events); err != nil || len(events) != 4 {
		t.Fatal(events, err)
	}
	if len(events[0]) != 11 || string(events[0]["host_id"]) != `"hash"` || string(events[0]["details"]) != `"provided"` || events[0]["unknown"] != nil || len(events[1]) != 0 || len(events[2]) != 0 || len(events[3]) != 0 {
		t.Fatal(events)
	}
	value.Header.Set("X-Proof", "changed")
	if value.Wire.Header.Get("X-Proof") != "actual" {
		t.Fatal("header alias")
	}
}
func TestActionRecordsEventsNullScalarAndOrderedAliases(t *testing.T) {
	for _, test := range []struct{ events, want string }{{`null`, `null`}, {`[]`, `[]`}, {`false`, `[{}]`}, {`"text"`, `[{}]`}, {`{"hostId":"first","host_id":"last"}`, `"last"`}} {
		t.Run(test.events, func(t *testing.T) {
			cloud, api := actionRecordsCloud(t)
			cloud.Mux.HandleFunc("/v2.1/p/servers/server/os-instance-actions", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"instanceActions":[{"request_id":"req","events":`+test.events+`}]}`)
			})
			rows, err := api.AllRecords(context.Background(), "server")
			if err != nil || len(rows) != 1 {
				t.Fatal(rows, err)
			}
			if test.want == `"last"` {
				var events []map[string]json.RawMessage
				if err := json.Unmarshal(rows[0].Resource.Body["events"], &events); err != nil || string(events[0]["host_id"]) != test.want {
					t.Fatal(events, err)
				}
			} else {
				actionRecordField(t, rows[0], "events", test.want)
			}
		})
	}
}
func TestActionRecordsQueryAliasesLocalFiltersRawCapAndParentScope(t *testing.T) {
	cloud, api := actionRecordsCloud(t)
	calls := 0
	cloud.Mux.HandleFunc("/v2.1/p/servers/server/os-instance-actions", func(w http.ResponseWriter, r *http.Request) {
		calls++
		testhelper.AssertEquals(t, "since", r.URL.Query().Get("changes-since"))
		testhelper.AssertEquals(t, "before", r.URL.Query().Get("changes-before"))
		testhelper.AssertEquals(t, "2", r.URL.Query().Get("limit"))
		if r.URL.Query().Has("action") || r.URL.Query().Has("unknown") {
			t.Fatal(r.URL)
		}
		testcloud.JSON(w, 200, `{"instanceActions":[{"request_id":"one","action":"stop"},{"request_id":"two","action":"reboot"},{"request_id":"three","action":"reboot"}],"next":{"unconsumed":true}}`)
	})
	scope, err := api.InServer(context.Background(), resource.ID("server"))
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for value, err := range scope.ListRecords(context.Background(), WithRecordListQuery("changes_since", "since"), WithRecordListQuery("changes-before", "before"), WithRecordListQuery("action", "reboot"), WithRecordListMaxItems(2), WithRecordListQuery("unknown", make(chan int))) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, string(value.Resource.Body["id"]))
	}
	if !reflect.DeepEqual(ids, []string{`"two"`}) || calls != 1 {
		t.Fatal(ids, calls)
	}
}
func TestActionRecordsDiscoveryIsLazyAndCeilingOwnsEveryPage(t *testing.T) {
	for _, selected := range []string{"", "2.58", "2.100"} {
		t.Run(selected, func(t *testing.T) {
			cloud, api := actionRecordsCloud(t)
			api.client.Microversion = selected
			discovery, pages := 0, 0
			cloud.Mux.HandleFunc("/v2.1/", func(w http.ResponseWriter, r *http.Request) {
				discovery++
				for name := range r.Header {
					if actionVersionHeader(name) {
						t.Fatal("discovery has version")
					}
				}
				testcloud.JSON(w, 200, `{"version":{"id":"v2.1","status":"CURRENT","min_version":"2.1","version":"2.100","links":[{"rel":"self","href":"`+cloud.Server.URL+`/v2.1/"}]}}`)
			})
			cloud.Mux.HandleFunc("/v2.1/p/servers/server/os-instance-actions", func(w http.ResponseWriter, r *http.Request) {
				pages++
				want := selected
				if want == "" {
					want = "2.84"
				}
				testhelper.AssertEquals(t, "compute "+want, r.Header.Get("OpenStack-API-Version"))
				testhelper.AssertEquals(t, want, r.Header.Get("X-OpenStack-Nova-API-Version"))
				if pages == 1 {
					testcloud.JSON(w, 200, `{"instanceActions":[{"request_id":"one"}],"links":[{"rel":"next","href":"`+cloud.Server.URL+`/v2.1/p/servers/server/os-instance-actions?marker=one"}]}`)
				} else {
					testcloud.JSON(w, 200, `{"instanceActions":[{"request_id":"two"}]}`)
				}
			})
			sequence := api.ListRecords(context.Background(), "server")
			if discovery != 0 || pages != 0 {
				t.Fatal("eager list")
			}
			rows := 0
			for value, err := range sequence {
				if err != nil {
					t.Fatal(err)
				}
				rows++
				want := 0
				if selected == "" {
					want = 1
				}
				if len(value.Discovery) != want {
					t.Fatal(value.Discovery)
				}
			}
			want := 0
			if selected == "" {
				want = 1
			}
			if rows != 2 || pages != 2 || discovery != want || api.client.Microversion != selected {
				t.Fatal(rows, pages, discovery)
			}
		})
	}
}
func TestActionRecordsOpaqueJMESUsesActualGeneratorWithZeroHTTP(t *testing.T) {
	cloud, api := actionRecordsCloud(t)
	api.client.Microversion = ""
	calls := 0
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(500) })
	for _, expression := range []string{"[].action", "[0]", "type(@)", "`{\"literal\":true}`", "@ == @", "contains(to_array(@), @)"} {
		value, err := api.ProxyListRecords(context.Background(), "server", WithRecordListJMESPath(expression))
		if err != nil || value == nil || calls != 0 {
			t.Fatal(value, err, calls)
		}
	}
	value, err := api.ProxyListRecords(context.Background(), "server", WithRecordListJMESPath("@"))
	if err != nil || value == nil || calls != 0 {
		t.Fatal(value, err, calls)
	}
	if _, ok := value.Value.(*RecordListGenerator); !ok || value.JSON != nil {
		t.Fatal(value)
	}
	if _, err := json.Marshal(value.Value); err == nil || calls != 0 {
		t.Fatal(err, calls)
	}
	for _, expression := range []string{"length(@)", "to_string(@)"} {
		if value, err := api.ProxyListRecords(context.Background(), "server", WithRecordListJMESPath(expression)); err == nil || value != nil || calls != 0 {
			t.Fatal(value, err, calls)
		}
	}
}
func TestActionRecordsCancelAfterYieldDoesNotReadNextPage(t *testing.T) {
	cloud, api := actionRecordsCloud(t)
	calls := 0
	cloud.Mux.HandleFunc("/v2.1/p/servers/server/os-instance-actions", func(w http.ResponseWriter, r *http.Request) {
		calls++
		testcloud.JSON(w, 200, `{"instanceActions":[{"request_id":"one"}],"next":"`+cloud.Server.URL+`/v2.1/p/servers/server/os-instance-actions?marker=one"}`)
	})
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("cancel witness")
	var failure error
	rows := 0
	for _, err := range api.ListRecords(ctx, "server") {
		if err != nil {
			failure = err
			break
		}
		rows++
		cancel(cause)
	}
	if rows != 1 || calls != 1 || !errors.Is(failure, cause) {
		t.Fatal(rows, calls, failure)
	}
}
func TestActionRecordsOptionsCannotReplaceParentAndKeepNilMapOwnership(t *testing.T) {
	cloud, api := actionRecordsCloud(t)
	calls := 0
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(500) })
	for _, err := range api.ListRecords(context.Background(), "server", WithRecordListQuery("server_id", make(chan int))) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	late := 0
	for _, err := range api.ListRecords(context.Background(), "server", func(c *RecordListOpts) error {
		api.client.ResourceBase = cloud.Server.URL + "/changed/"
		c.Headers = nil
		c.Query = nil
		return nil
	}, func(c *RecordListOpts) error { late++; return nil }) {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if calls != 0 || late != 0 {
		t.Fatal(calls, late)
	}
}

type actionRecordFaultTransport func(*http.Request) (*http.Response, error)

func (f actionRecordFaultTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type actionRecordFaultBody struct {
	io.ReadCloser
	fault  error
	closes *int
}

func (b *actionRecordFaultBody) Close() error {
	*b.closes++
	return errors.Join(b.ReadCloser.Close(), b.fault)
}
func TestActionRecordsPhysicalFailuresPreserveActualPageAndStopHooks(t *testing.T) {
	for _, status := range []int{203, 401, 404, 429, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cloud, api := actionRecordsCloud(t)
			calls, closes, hooks := 0, 0, 0
			body := `{"instanceActions":[{"request_id":"one"}]}`
			cloud.Mux.HandleFunc("/v2.1/p/servers/server/os-instance-actions", func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("X-Proof", "actual")
				testcloud.JSON(w, status, body)
			})
			cause := errors.New("Close fault")
			base := cloud.Provider.HTTPClient.Transport
			cloud.Provider.HTTPClient.Transport = actionRecordFaultTransport(func(r *http.Request) (*http.Response, error) {
				resp, err := base.RoundTrip(r)
				if resp != nil {
					resp.Body = &actionRecordFaultBody{ReadCloser: resp.Body, fault: cause, closes: &closes}
				}
				return resp, err
			})
			cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				hooks++
				return nil
			}
			cloud.Provider.ReauthFunc = func(context.Context) error { hooks++; return nil }
			cloud.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error { hooks++; return nil }
			var failure error
			rows := 0
			for _, err := range api.ListRecords(context.Background(), "server") {
				if err != nil {
					failure = err
					break
				}
				rows++
			}
			if failure == nil || rows != 0 || calls != 1 || closes != 1 || hooks != 0 {
				t.Fatal(failure, rows, calls, closes, hooks)
			}
			if status == 203 {
				var proof *resource.ResponseError
				if !errors.As(failure, &proof) || !errors.Is(failure, cause) || string(proof.Body) != body || proof.StatusCode != status {
					t.Fatal(failure)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if status == 401 {
					var renewal *gophercloud.ErrUnableToReauthenticate
					if !errors.As(failure, &renewal) || !errors.As(renewal.ErrOriginal, &native) || !errors.Is(renewal.ErrReauth, cause) {
						t.Fatal(failure)
					}
				} else if !errors.As(failure, &native) || !errors.Is(failure, cause) {
					t.Fatal(failure)
				}
				if native.Actual != status || string(native.Body) != body || native.Method != http.MethodGet || native.URL != cloud.Server.URL+"/v2.1/p/servers/server/os-instance-actions" || native.ResponseHeader.Get("X-Proof") != "actual" {
					t.Fatal(native)
				}
			}
		})
	}
}
func TestActionRecordsClean401And503RetriesKeepScopeQueryAndVersion(t *testing.T) {
	for _, status := range []int{401, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cloud, api := actionRecordsCloud(t)
			calls, hooks := 0, 0
			cloud.Mux.HandleFunc("/v2.1/p/servers/server/os-instance-actions", func(w http.ResponseWriter, r *http.Request) {
				calls++
				testhelper.AssertEquals(t, "compute 2.84", r.Header.Get("OpenStack-API-Version"))
				testhelper.AssertEquals(t, "since", r.URL.Query().Get("changes-since"))
				if calls == 1 {
					testcloud.JSON(w, status, `{}`)
					return
				}
				testcloud.JSON(w, 200, `{"instanceActions":[]}`)
			})
			if status == 401 {
				cloud.Provider.ReauthFunc = func(context.Context) error { hooks++; return nil }
			} else {
				cloud.Provider.RetryFunc = func(_ context.Context, method, target string, _ *gophercloud.RequestOpts, original error, count uint) error {
					hooks++
					var native gophercloud.ErrUnexpectedResponseCode
					if count != 1 || !errors.As(original, &native) || native.Actual != http.StatusServiceUnavailable {
						return original
					}
					testhelper.AssertEquals(t, http.MethodGet, method)
					testhelper.AssertEquals(t, cloud.Server.URL+"/v2.1/p/servers/server/os-instance-actions?changes-since=since", target)
					testhelper.AssertEquals(t, method, native.Method)
					testhelper.AssertEquals(t, target, native.URL)
					testhelper.AssertEquals(t, `{}`, string(native.Body))
					return nil
				}
			}
			rows, err := api.AllRecords(context.Background(), "server", WithRecordListQuery("changes_since", "since"))
			if err != nil || len(rows) != 0 || calls != 2 || hooks != 1 {
				t.Fatal(rows, err, calls, hooks)
			}
		})
	}
}

func TestActionRecordsEventResourceIdentityDoesNotEqualPlainJSONDictionary(t *testing.T) {
	cloud, api := actionRecordsCloud(t)
	cloud.Mux.HandleFunc("/v2.1/p/servers/server/os-instance-actions", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"instanceActions":[{"request_id":"resource","events":[{"event":"e"}]},{"request_id":"plain","events":[null]},{"request_id":"empty","events":[]},{"request_id":"none","events":null}]}`)
	})
	full := map[string]any{}
	for _, key := range []string{"event", "start_time", "finish_time", "result", "traceback", "host", "host_id", "details", "id", "name", "location"} {
		full[key] = nil
	}
	full["event"] = "e"
	for _, test := range []struct {
		filter any
		want   []string
	}{
		{[]any{full}, nil},
		{[]any{map[string]any{}}, []string{`"plain"`}},
		{[]any{}, []string{`"empty"`}},
		{nil, []string{`"none"`}},
	} {
		var ids []string
		for value, err := range api.ListRecords(context.Background(), "server", WithRecordListQuery("events", test.filter)) {
			if err != nil {
				t.Fatal(err)
			}
			ids = append(ids, string(value.Resource.Body["id"]))
		}
		if !reflect.DeepEqual(ids, test.want) {
			t.Fatalf("filter=%v ids=%v want=%v", test.filter, ids, test.want)
		}
	}
}
