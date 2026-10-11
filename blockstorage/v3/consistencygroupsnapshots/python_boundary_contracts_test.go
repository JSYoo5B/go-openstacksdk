package consistencygroupsnapshots_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPythonConsistencyGroupSnapshotListReservedControlsFailBeforeMarshalOrHTTP(t *testing.T) {
	for _, key := range []string{"cls", "self", "session", "base_path", "list_base_path", "microversion", "headers", "__conflicting_attrs", "jmespath_filters", "resource_type", "details", "connection", "_synchronized"} {
		for _, input := range []any{nil, false, 0, snapshotMarshal(func() ([]byte, error) { t.Fatal("reserved list control was marshaled"); return nil, nil })} {
			api, calls := snapshotAPI(t, "", func(r *http.Request) *http.Response {
				t.Fatal("reserved control reached HTTP", r.URL)
				return snapshotWire(200, `{}`)
			})
			option := consistencygroupsnapshots.WithRecordListQuery(key, input)
			if len(*calls) != 0 {
				t.Fatal("option factory executed HTTP")
			}
			values, err := snapshotsCollect(api.ListRecords(context.Background(), option))
			if len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
				t.Fatal(key, input, values, err, *calls)
			}
		}
		for _, raw := range []json.RawMessage{json.RawMessage(`null`), json.RawMessage(`false`), json.RawMessage(`0`), json.RawMessage(`not JSON`)} {
			api, calls := snapshotAPI(t, "", func(*http.Request) *http.Response { t.Fatal("raw reserved control reached HTTP"); return nil })
			values, err := snapshotsCollect(api.ListRecords(context.Background(), consistencygroupsnapshots.WithRecordListOpts(consistencygroupsnapshots.RecordListOpts{Query: map[string]json.RawMessage{key: raw}})))
			if len(values) != 0 || !errors.Is(err, resource.ErrInvalidOption) || len(*calls) != 0 {
				t.Fatal(key, string(raw), values, err, *calls)
			}
		}
	}
	api, calls := snapshotAPI(t, "", func(*http.Request) *http.Response { return snapshotWire(200, `{"cgsnapshots":[]}`) })
	values, err := snapshotsCollect(api.ListRecords(context.Background(), consistencygroupsnapshots.WithRecordListQuery("CLS", snapshotMarshal(func() ([]byte, error) { t.Fatal("ordinary unknown parameter marshaled"); return nil, nil }))))
	if err != nil || len(values) != 0 || len(*calls) != 1 || (*calls)[0].query != "" {
		t.Fatal(values, err, *calls)
	}
}

func TestPythonConsistencyGroupSnapshotListAndFindBinary64NextTruthiness(t *testing.T) {
	cases := []struct {
		next   string
		truthy bool
	}{
		{`null`, false}, {`false`, false}, {`0`, false}, {`-0.0`, false}, {`1e-400`, false}, {`-1e-400`, false}, {`[]`, false}, {`{}`, false}, {`""`, false},
		{`true`, true}, {`1`, true}, {`1e-300`, true}, {`1e400`, true}, {`[0]`, true}, {`{"unused":false}`, true},
	}
	for _, find := range []bool{false, true} {
		for _, test := range cases {
			t.Run(fmt.Sprintf("find=%t/next=%s", find, test.next), func(t *testing.T) {
				body := `{"cgsnapshots":[{"id":"first","name":"data"}],"next":` + test.next + `}`
				api, calls := snapshotAPI(t, "", func(r *http.Request) *http.Response {
					if find && strings.HasSuffix(r.URL.Path, "/data") {
						return snapshotWire(404, "suppressed member")
					}
					if r.URL.Path != snapshotBase+"/detail" {
						t.Fatal("unexpected continuation", r.URL)
					}
					return snapshotWire(203, body)
				})
				var err error
				var value *consistencygroupsnapshots.ConsistencyGroupSnapshotRecord
				if find {
					value, err = api.FindRecord(context.Background(), "data")
				} else {
					values, cause := snapshotsCollect(api.ListRecords(context.Background()))
					err = cause
					if len(values) != 1 {
						t.Fatal(values, cause)
					}
					value = values[0]
				}
				count := 1
				if find {
					count = 2
				}
				if (err != nil) != test.truthy || len(*calls) != count {
					t.Fatal(find, test, value, err, *calls)
				}
				if test.truthy {
					var proof *resource.ResponseError
					if !errors.As(err, &proof) || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Request-Id") != "actual-consistency-group-snapshot" {
						t.Fatal(proof, err)
					}
					if find && (value == nil || value.Resource != nil || value.Wire != nil || value.StatusCode != 203 || string(value.Envelope) != body) {
						t.Fatal("terminal pager error became successful candidate", value, err)
					}
				} else {
					snapshotField(t, value, "id", `"first"`)
					if value.StatusCode != 203 || string(value.Envelope) != body {
						t.Fatal(value)
					}
				}
			})
		}
	}
}

func TestPythonConsistencyGroupSnapshotPhysical401StopsRetryAndReauthWithNativeFields(t *testing.T) {
	for _, operation := range []string{"get", "find", "list", "create", "delete"} {
		for _, mode := range []string{"read", "close", "both"} {
			t.Run(operation+"/"+mode, func(t *testing.T) {
				body := `{"expired":"actual401"}`
				readCause, closeCause := errors.New("physical401 read"), errors.New("physical401 close")
				var closes atomic.Int32
				faulty := &snapshotFaultBody{data: []byte(body), closes: &closes}
				if mode != "close" {
					faulty.read = readCause
				}
				if mode != "read" {
					faulty.close = closeCause
				}
				api, calls := snapshotAPI(t, "", func(*http.Request) *http.Response {
					response := snapshotWire(401, body)
					response.Body = faulty
					return response
				})
				retries, renewals := 0, 0
				client := api.RawClient()
				client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
					retries++
					return nil
				}
				client.ReauthFunc = func(context.Context) error { renewals++; client.SetToken("renewed"); return nil }
				retryPointer, reauthPointer := reflect.ValueOf(client.RetryFunc).Pointer(), reflect.ValueOf(client.ReauthFunc).Pointer()
				var value *consistencygroupsnapshots.ConsistencyGroupSnapshotRecord
				var deletion *consistencygroupsnapshots.DeleteRecordResult
				var err error
				switch operation {
				case "get":
					value, err = api.GetRecord(context.Background(), "v")
				case "find":
					value, err = api.FindRecord(context.Background(), "v")
				case "list":
					values, cause := snapshotsCollect(api.ListRecords(context.Background()))
					err = cause
					if len(values) != 0 {
						t.Fatal(values, cause)
					}
				case "create":
					value, err = api.CreateRecord(context.Background(), map[string]any{"consistencygroup_id": "source"})
				case "delete":
					deletion, err = api.DeleteRecord(context.Background(), "v")
				}
				var authentication *gophercloud.ErrUnableToReauthenticate
				if !errors.As(err, &authentication) || authentication == nil || len(*calls) != 1 || retries != 0 || renewals != 0 || closes.Load() != 1 || reflect.ValueOf(client.RetryFunc).Pointer() != retryPointer || reflect.ValueOf(client.ReauthFunc).Pointer() != reauthPointer {
					t.Fatal(operation, mode, value, deletion, err, authentication, *calls, retries, renewals, closes.Load())
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(authentication.ErrOriginal, &native) || native.Actual != 401 || string(native.Body) != body || native.ResponseHeader.Get("X-Request-Id") != "actual-consistency-group-snapshot" || native.Method != (*calls)[0].method || !strings.HasSuffix(native.URL, (*calls)[0].escaped) {
					t.Fatal(authentication, native, *calls)
				}
				if mode != "close" && (!errors.Is(err, readCause) || !errors.Is(authentication.ErrReauth, readCause)) || mode != "read" && (!errors.Is(err, closeCause) || !errors.Is(authentication.ErrReauth, closeCause)) {
					t.Fatal(mode, err, authentication.ErrReauth)
				}
				if operation == "delete" {
					if deletion == nil || deletion.IgnoredMissing || deletion.StatusCode != 401 || string(deletion.Envelope) != body || deletion.Header.Get("X-Request-Id") != "actual-consistency-group-snapshot" {
						t.Fatal(deletion, err)
					}
				} else if operation != "list" && (value == nil || value.Resource != nil || value.Wire != nil || value.StatusCode != 401 || string(value.Envelope) != body || value.Header.Get("X-Request-Id") != "actual-consistency-group-snapshot") {
					t.Fatal(value, err)
				}
			})
		}
	}
}

func TestPythonConsistencyGroupSnapshotClean401RenewsTokenAndRetainsRequest(t *testing.T) {
	for _, operation := range []string{"get", "find", "list", "create", "delete"} {
		t.Run(operation, func(t *testing.T) {
			attempts := 0
			api, calls := snapshotAPI(t, "3.71", func(r *http.Request) *http.Response {
				attempts++
				if attempts == 1 {
					return snapshotWire(401, `{"expired":"clean"}`)
				}
				if operation == "list" {
					return snapshotWire(202, `{"cgsnapshots":[{"id":"v"}]}`)
				}
				if operation == "delete" {
					return snapshotWire(202, "actual delete ACK")
				}
				return snapshotWire(202, `{"cgsnapshot":{"id":"v"}}`)
			})
			renewals := 0
			client := api.RawClient()
			client.ReauthFunc = func(context.Context) error { renewals++; client.SetToken("renewed"); return nil }
			pointer := reflect.ValueOf(client.ReauthFunc).Pointer()
			var err error
			switch operation {
			case "get":
				_, err = api.GetRecord(context.Background(), "v")
			case "find":
				_, err = api.FindRecord(context.Background(), "v")
			case "list":
				values, cause := snapshotsCollect(api.ListRecords(context.Background()))
				err = cause
				if len(values) != 1 {
					t.Fatal(values, cause)
				}
			case "create":
				_, err = api.CreateRecord(context.Background(), map[string]any{"consistencygroup_id": "source", "description": false})
			case "delete":
				_, err = api.DeleteRecord(context.Background(), "v")
			}
			if err != nil || len(*calls) != 2 || renewals != 1 || (*calls)[0].headers.Get("X-Auth-Token") != "test-token" || (*calls)[1].headers.Get("X-Auth-Token") != "renewed" || reflect.ValueOf(client.ReauthFunc).Pointer() != pointer {
				t.Fatal(operation, err, *calls, renewals)
			}
			if (*calls)[0].method != (*calls)[1].method || (*calls)[0].escaped != (*calls)[1].escaped || (*calls)[0].query != (*calls)[1].query || (*calls)[0].body != (*calls)[1].body {
				t.Fatal("reauthentication changed the owned request", *calls)
			}
			for _, call := range *calls {
				if call.headers.Get("OpenStack-API-Version") != "volume 3.71" || call.headers.Get("X-OpenStack-Volume-API-Version") != "3.71" {
					t.Fatal(call)
				}
			}
		})
	}
}
