package credentials_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func credentialRecordField(t *testing.T, v *credentials.CredentialRecord, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		t.Fatalf("%s: got %#v want %s", key, v, want)
	}
}
func credentialRecordIDs(t *testing.T, api *credentials.API, options ...credentials.RecordListOption) ([]string, error) {
	t.Helper()
	var ids []string
	for v, err := range api.ListRecords(context.Background(), options...) {
		if err != nil {
			return ids, err
		}
		var id string
		if err := json.Unmarshal(v.Resource.Body["id"], &id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func TestPythonCredentialCreateRawBodyAndOwnedViews(t *testing.T) {
	var calls []nativeCredCall
	api, _ := nativeCredAPI(t, &calls, func(*http.Request) (int, string) {
		return 201, `{"credential":{"id":"c1","blob":{"count":9007199254740993},"extension":[null,false]}}`
	})
	fields := map[string]json.RawMessage{"user_id": json.RawMessage(`"u1"`), "type": json.RawMessage(`"ec2"`), "blob": json.RawMessage(`{"access":"key"}`), "project_id": json.RawMessage(`null`), "name": json.RawMessage(`""`), "unknown": json.RawMessage(`{bad`)}
	option := credentials.WithRecordCreateOpts(credentials.RecordCreateOpts{Fields: fields})
	fields["user_id"][1] = 'x'
	fields["blob"] = json.RawMessage(`"changed"`)
	v, err := api.CreateRecord(context.Background(), option, credentials.WithRecordCreateField("unused", make(chan int)))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"credential":{"blob":{"access":"key"},"name":"","project_id":null,"type":"ec2","user_id":"u1"}}`
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != "/keystone/v3/credentials" || calls[0].query != "" || calls[0].body != want {
		t.Fatal(calls)
	}
	credentialRecordField(t, v, "id", `"c1"`)
	credentialRecordField(t, v, "blob", `{"count":9007199254740993}`)
	credentialRecordField(t, v, "user_id", `"u1"`)
	credentialRecordField(t, v, "type", `"ec2"`)
	credentialRecordField(t, v, "name", `""`)
	credentialRecordField(t, v, "project_id", `null`)
	credentialRecordField(t, v, "location", `null`)
	if len(v.Resource.Body) != 7 || v.StatusCode != 201 || string(v.Wire.Body["extension"]) != `[null,false]` {
		t.Fatal(v)
	}
	v.Resource.Body["blob"][2] = 'X'
	if string(v.Wire.Body["blob"]) != `{"count":9007199254740993}` || !strings.Contains(string(v.Envelope), "9007199254740993") {
		t.Fatal("body channels alias")
	}
	v.Header.Set("X-Owned", "one")
	if v.Resource.Header.Get("X-Owned") != "" || v.Wire.Header.Get("X-Owned") != "" {
		t.Fatal("header channels alias")
	}
	calls = nil
	if _, err := api.CreateRecord(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls[0].body != `{"credential":{}}` {
		t.Fatal("Source imposes no native required-field precheck", calls)
	}
	calls = nil
	if _, err := api.CreateRecord(context.Background(), credentials.WithRecordCreateField("blob", false), credentials.WithRecordCreateField("project_id", ""), credentials.WithRecordCreateField("user_id", nil), credentials.WithRecordCreateField("type", []string{})); err != nil {
		t.Fatal(err)
	}
	if calls[0].body != `{"credential":{"blob":false,"project_id":"","type":[],"user_id":null}}` {
		t.Fatal(calls)
	}
}

func TestPythonCredentialGetTranslationAndStatuses(t *testing.T) {
	for _, item := range []struct {
		name, body string
		valid      bool
	}{
		{"singular", `{"credential":{"blob":[null,true],"user_id":42}}`, true}, {"root", `{"id":"server","blob":false,"extension":9007199254740993}`, true}, {"blank", "", true}, {"malformed", "{bad", true}, {"empty object", `{}`, true}, {"array", `[]`, false}, {"null", `null`, false}, {"wrapped null", `{"credential":null}`, false},
	} {
		t.Run(item.name, func(t *testing.T) {
			var calls []nativeCredCall
			api, _ := nativeCredAPI(t, &calls, func(*http.Request) (int, string) { return 200, item.body })
			v, err := api.GetRecord(context.Background(), "selected")
			if (err == nil) != item.valid || len(calls) != 1 || calls[0].path != "/keystone/v3/credentials/selected" || calls[0].query != "" {
				t.Fatal(v, err, calls)
			}
			if !item.valid {
				var proof *resource.ResponseError
				if v == nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != item.body {
					t.Fatal(v, err, proof)
				}
				return
			}
			if string(v.Envelope) != item.body || v.StatusCode != 200 {
				t.Fatal(v)
			}
			credentialRecordField(t, v, "location", `null`)
			if item.name != "root" {
				credentialRecordField(t, v, "id", `"selected"`)
			}
			if len(v.Resource.Body) != 7 {
				t.Fatal(v.Resource.Body)
			}
		})
	}
	for _, code := range []int{200, 201, 202, 204, 206, 299, 301, 399, 400, 403, 404, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []nativeCredCall
			api, _ := nativeCredAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
			v, err := api.GetRecord(context.Background(), "id")
			if code < 400 {
				if err != nil || v == nil || v.StatusCode != code {
					t.Fatal(v, err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if v != nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != `{}` {
					t.Fatal(v, err, native)
				}
			}
		})
	}
}

func TestPythonCredentialListServerQueryAndLocalJSONFilters(t *testing.T) {
	var calls []nativeCredCall
	api, _ := nativeCredAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"credentials":[{"id":"a","name":"ci","type":"cert","user_id":"server-value","blob":{"access":"x","count":9007199254740993}},{"id":"b","name":"other","blob":{"access":"y"}}],"links":{"next":null}}`
	})
	ids, err := credentialRecordIDs(t, api, credentials.WithRecordListType("ec2"), credentials.WithRecordListUserID("u1"), credentials.WithRecordListQuery("name", "ci"), credentials.WithRecordListQuery("blob", map[string]any{"access": "x"}), credentials.WithRecordListQuery("project_id", nil), credentials.WithRecordListQuery("unknown", make(chan int)))
	if err != nil || !reflect.DeepEqual(ids, []string{"a"}) || len(calls) != 1 || calls[0].query != "type=ec2&user_id=u1" {
		t.Fatal(ids, err, calls)
	}
	calls = nil
	ids, err = credentialRecordIDs(t, api, credentials.WithRecordListQuery("type", json.RawMessage(`["ec2",null,true,{"b":0,"a":false}]`)), credentials.WithRecordListQuery("user_id", nil))
	if err != nil || !reflect.DeepEqual(ids, []string{"a", "b"}) || calls[0].query != "type=ec2&type=True&type=b&type=a" {
		t.Fatal(ids, err, calls)
	}
	calls = nil
	ids, err = credentialRecordIDs(t, api, credentials.WithRecordListQuery("name", "other"), credentials.WithRecordListMaxItems(1))
	if err != nil || len(ids) != 0 || len(calls) != 1 || calls[0].query != "limit=1" {
		t.Fatal("raw-row cap precedes local filters", ids, err, calls)
	}
	calls = nil
	ids, err = credentialRecordIDs(t, api, credentials.WithRecordListLimit(0), credentials.WithRecordListMaxItems(1))
	if err != nil || !reflect.DeepEqual(ids, []string{"a"}) || calls[0].query != "limit=1" {
		t.Fatal(ids, err, calls)
	}
	calls = nil
	ids, err = credentialRecordIDs(t, api, credentials.WithRecordListLimit(0), credentials.WithRecordListMarker("start"), credentials.WithRecordListPaginated(false))
	if err != nil || !reflect.DeepEqual(ids, []string{"a", "b"}) || calls[0].query != "limit=0&marker=start" {
		t.Fatal(ids, err, calls)
	}
}

func TestPythonCredentialListPagingLazinessAndWireMarker(t *testing.T) {
	for _, shape := range []string{"next", "links", "credentials_links"} {
		t.Run(shape, func(t *testing.T) {
			var calls []nativeCredCall
			api, _ := nativeCredAPI(t, &calls, func(req *http.Request) (int, string) {
				if req.URL.Query().Get("marker") == "next" {
					return 200, `{"credentials":[],"links":{"next":null}}`
				}
				link := `"next":"?marker=next"`
				if shape == "links" {
					link = `"links":{"next":"?marker=next"}`
				}
				if shape == "credentials_links" {
					link = `"credentials_links":[{"rel":"next","href":"?marker=next"}]`
				}
				return 200, `{"credentials":[{"id":"a"}],` + link + `}`
			})
			ids, err := credentialRecordIDs(t, api)
			if err != nil || !reflect.DeepEqual(ids, []string{"a"}) || len(calls) != 2 || calls[1].query != "marker=next" {
				t.Fatal(ids, err, calls)
			}
		})
	}
	t.Run("HTTP Link", func(t *testing.T) {
		var calls []nativeCredCall
		api, cloud := nativeCredAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{}` })
		attempts := 0
		cloud.SetTransport(testcloud.LinkRequest(func(*http.Request) (*http.Response, error) {
			attempts++
			body := `{"credentials":[{"id":"a"}]}`
			header := http.Header{"Link": {`<?marker=next>; rel="next"`}}
			if attempts == 2 {
				body = `{"credentials":[]}`
				header = http.Header{}
			}
			return testcloud.Response(200, body, header), nil
		}))
		ids, err := credentialRecordIDs(t, api)
		if err != nil || !reflect.DeepEqual(ids, []string{"a"}) || attempts != 2 {
			t.Fatal(ids, err, attempts)
		}
	})
	t.Run("marker derives from untouched page after consumer mutation", func(t *testing.T) {
		var calls []nativeCredCall
		api, _ := nativeCredAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.Query().Get("marker") == "a" {
				return 200, `{"credentials":[]}`
			}
			return 200, `{"credentials":[{"id":"a"}]}`
		})
		count := 0
		for v, err := range api.ListRecords(context.Background(), credentials.WithRecordListType("ec2"), credentials.WithRecordListLimit(2)) {
			if err != nil {
				t.Fatal(err)
			}
			count++
			v.Resource.Body["id"] = json.RawMessage(`"changed"`)
			v.Wire.Body["id"] = json.RawMessage(`"also changed"`)
		}
		if count != 1 || len(calls) != 2 || calls[1].query != "limit=2&marker=a&type=ec2" {
			t.Fatal(count, calls)
		}
	})
	t.Run("singleton and cap leave malformed tail unread", func(t *testing.T) {
		var calls []nativeCredCall
		api, _ := nativeCredAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.Query().Get("limit") == "1" {
				return 200, `{"credentials":[{"id":"a"},null],"links":{"next":"?marker=next"}}`
			}
			return 200, `{"credentials":{"id":"one"}}`
		})
		ids, err := credentialRecordIDs(t, api)
		if err != nil || !reflect.DeepEqual(ids, []string{"one"}) {
			t.Fatal(ids, err)
		}
		calls = nil
		ids, err = credentialRecordIDs(t, api, credentials.WithRecordListMaxItems(1))
		if err != nil || !reflect.DeepEqual(ids, []string{"a"}) || len(calls) != 1 {
			t.Fatal(ids, err, calls)
		}
	})
	t.Run("late error and early stop", func(t *testing.T) {
		var calls []nativeCredCall
		api, _ := nativeCredAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.Query().Get("marker") == "next" {
				return 500, "later"
			}
			return 200, `{"credentials":[{"id":"a"}],"next":"?marker=next"}`
		})
		ids, err := credentialRecordIDs(t, api)
		var native gophercloud.ErrUnexpectedResponseCode
		if !reflect.DeepEqual(ids, []string{"a"}) || !errors.As(err, &native) || native.Actual != 500 {
			t.Fatal(ids, err)
		}
		calls = nil
		for _, err := range api.ListRecords(context.Background()) {
			if err != nil {
				t.Fatal(err)
			}
			break
		}
		if len(calls) != 1 {
			t.Fatal(calls)
		}
	})
}

func TestPythonCredentialDeleteMissingPolicyAndMixedFailure(t *testing.T) {
	for _, code := range []int{200, 202, 204, 301, 399, 400, 403, 404, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []nativeCredCall
			api, _ := nativeCredAPI(t, &calls, func(*http.Request) (int, string) { return code, "opaque" })
			err := api.DeleteRecord(context.Background(), "selected")
			if (err == nil) != (code < 400 || code == 404) || len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != "/keystone/v3/credentials/selected" || calls[0].query != "" {
				t.Fatal(err, calls)
			}
			if code == 404 {
				err = api.DeleteRecord(context.Background(), "selected", credentials.WithRecordDeleteIgnoreMissing(false))
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "opaque" {
					t.Fatal(err, native)
				}
			}
		})
	}
	t.Run("404 plus retry failure not suppressed", func(t *testing.T) {
		var calls []nativeCredCall
		api, _ := nativeCredAPI(t, &calls, func(*http.Request) (int, string) { return 404, "original" })
		cause := errors.New("callback failed")
		api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return cause }
		err := api.DeleteRecord(context.Background(), "a")
		if !errors.Is(err, cause) || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
}

func TestPythonCredentialFindDefaultFallbackAndAllPages(t *testing.T) {
	for _, status := range []int{400, 403, 404} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls []nativeCredCall
			api, _ := nativeCredAPI(t, &calls, func(req *http.Request) (int, string) {
				if strings.HasSuffix(req.URL.Path, "/ci") {
					return status, "missing"
				}
				if req.URL.Query().Get("marker") == "next" {
					return 200, `{"credentials":[{"id":"found","name":"ci"}]}`
				}
				return 200, `{"credentials":[{"id":"skip","name":"other"}],"next":"?marker=next"}`
			})
			v, err := api.FindRecord(context.Background(), "ci")
			if err != nil {
				t.Fatal(err)
			}
			credentialRecordField(t, v, "id", `"found"`)
			if len(calls) != 3 || calls[0].query != "" || calls[1].query != "" || calls[2].query != "marker=next" {
				t.Fatal(calls)
			}
		})
	}
	t.Run("direct malformed response keeps literal ID seed", func(t *testing.T) {
		var calls []nativeCredCall
		api, _ := nativeCredAPI(t, &calls, func(*http.Request) (int, string) { return 200, "{broken" })
		v, err := api.FindRecord(context.Background(), "selected")
		if err != nil || len(calls) != 1 {
			t.Fatal(v, err, calls)
		}
		credentialRecordField(t, v, "id", `"selected"`)
	})
	for _, item := range []struct {
		name, list, kind string
		strict           bool
	}{
		{"missing", `[]`, "", false}, {"strict missing", `[]`, "missing", true}, {"duplicate", `[{"id":"a","name":"ci"},{"id":"b","name":"ci"}]`, "ambiguous", false}, {"ID match", `[{"id":"ci","name":"other"}]`, "", false},
	} {
		t.Run(item.name, func(t *testing.T) {
			var calls []nativeCredCall
			api, _ := nativeCredAPI(t, &calls, func(req *http.Request) (int, string) {
				if strings.HasSuffix(req.URL.Path, "/ci") {
					return 404, "missing"
				}
				return 200, `{"credentials":` + item.list + `}`
			})
			v, err := api.FindRecord(context.Background(), "ci", credentials.WithRecordFindIgnoreMissing(!item.strict))
			switch item.kind {
			case "missing":
				var target *resource.NotFoundError
				if !errors.As(err, &target) {
					t.Fatal(err)
				}
			case "ambiguous":
				var target *resource.AmbiguousError
				if !errors.As(err, &target) {
					t.Fatal(err)
				}
			default:
				if err != nil || item.name == "missing" && v != nil {
					t.Fatal(v, err)
				}
			}
		})
	}
	t.Run("later failure overrides early match", func(t *testing.T) {
		var calls []nativeCredCall
		api, _ := nativeCredAPI(t, &calls, func(req *http.Request) (int, string) {
			if strings.HasSuffix(req.URL.Path, "/ci") {
				return 404, "missing"
			}
			if req.URL.Query().Get("marker") == "next" {
				return 500, "later"
			}
			return 200, `{"credentials":[{"id":"a","name":"ci"}],"next":"?marker=next"}`
		})
		v, err := api.FindRecord(context.Background(), "ci")
		var native gophercloud.ErrUnexpectedResponseCode
		if v != nil || !errors.As(err, &native) || native.Actual != 500 || len(calls) != 3 {
			t.Fatal(v, err, calls)
		}
	})
	t.Run("accepted nonobject error never falls back", func(t *testing.T) {
		var calls []nativeCredCall
		api, _ := nativeCredAPI(t, &calls, func(*http.Request) (int, string) { return 200, `[]` })
		_, err := api.FindRecord(context.Background(), "ci")
		var proof *resource.ResponseError
		if !errors.As(err, &proof) || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
}

func TestPythonCredentialOptionsOwnInputsAndRejectBeforeHTTP(t *testing.T) {
	var calls []nativeCredCall
	api, _ := nativeCredAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.Header.Get("X-Owned") != "old" {
			t.Error(req.Header)
		}
		return 200, `{"credentials":[{"id":"a","name":"ci"}]}`
	})
	limit := 2
	paginated := false
	query := map[string]json.RawMessage{"name": json.RawMessage(`"ci"`), "type": json.RawMessage(`"ec2"`)}
	headers := map[string]string{"X-Owned": "old"}
	option := credentials.WithRecordListOpts(credentials.RecordListOpts{Limit: &limit, Paginated: &paginated, Query: query, Headers: headers})
	limit = 9
	query["name"] = json.RawMessage(`"changed"`)
	headers["X-Owned"] = "changed"
	seq := api.ListRecords(context.Background(), option)
	if len(calls) != 0 {
		t.Fatal("eager HTTP")
	}
	for range 2 {
		count := 0
		for v, err := range seq {
			if err != nil {
				t.Fatal(err)
			}
			count++
			credentialRecordField(t, v, "id", `"a"`)
		}
		if count != 1 {
			t.Fatal(count)
		}
	}
	if len(calls) != 2 || calls[0].query != "limit=2&type=ec2" {
		t.Fatal(calls)
	}
	t.Run("retained callback pointer", func(t *testing.T) {
		var retained *credentials.RecordListOpts
		ids, err := credentialRecordIDs(t, api, func(c *credentials.RecordListOpts) error {
			retained = c
			c.Headers = map[string]string{"X-Owned": "old"}
			c.Query = map[string]json.RawMessage{"name": json.RawMessage(`"ci"`)}
			return nil
		}, func(*credentials.RecordListOpts) error {
			retained.Headers["X-Owned"] = "bad"
			retained.Query["name"] = json.RawMessage(`"bad"`)
			return nil
		})
		if err != nil || !reflect.DeepEqual(ids, []string{"a"}) {
			t.Fatal(ids, err)
		}
	})
	for _, item := range []struct {
		name string
		call func() error
	}{
		{"nil read", func() error { _, err := api.GetRecord(context.Background(), "a", nil); return err }}, {"nil create", func() error { _, err := api.CreateRecord(context.Background(), nil); return err }}, {"nil delete", func() error { return api.DeleteRecord(context.Background(), "a", nil) }}, {"nil find", func() error { _, err := api.FindRecord(context.Background(), "a", nil); return err }}, {"nil list", func() error { _, err := credentialRecordIDs(t, api, nil); return err }},
		{"invalid JSON create", func() error {
			_, err := api.CreateRecord(context.Background(), credentials.WithRecordCreateOpts(credentials.RecordCreateOpts{Fields: map[string]json.RawMessage{"blob": json.RawMessage(`{bad`)}}))
			return err
		}},
		{"invalid JSON list", func() error {
			_, err := credentialRecordIDs(t, api, credentials.WithRecordListOpts(credentials.RecordListOpts{Query: map[string]json.RawMessage{"type": json.RawMessage(`{bad`)}}))
			return err
		}},
		{"negative limit", func() error { _, err := credentialRecordIDs(t, api, credentials.WithRecordListLimit(-1)); return err }},
		{"protected header", func() error {
			_, err := api.GetRecord(context.Background(), "a", credentials.WithRecordReadHeaders(map[string]string{"X-Auth-Token": "wrong"}))
			return err
		}},
		{"invalid ID", func() error { _, err := api.GetRecord(context.Background(), ".."); return err }},
	} {
		t.Run(item.name, func(t *testing.T) {
			before := len(calls)
			err := item.call()
			if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
				t.Fatal(err, calls)
			}
		})
	}
	t.Run("callback cancellation and error", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		cause, callbackErr := errors.New("caller cause"), errors.New("callback error")
		before := len(calls)
		_, err := api.FindRecord(ctx, "ci", func(*credentials.RecordFindOpts) error { cancel(cause); return callbackErr })
		if !errors.Is(err, cause) || !errors.Is(err, callbackErr) || !errors.Is(err, context.Canceled) || len(calls) != before {
			t.Fatal(err, calls)
		}
	})
	t.Run("callback source mutation and error", func(t *testing.T) {
		client := api.RawClient()
		original := client.Endpoint
		defer func() { client.Endpoint = original }()
		cause := errors.New("callback error")
		before := len(calls)
		_, err := api.CreateRecord(context.Background(), func(*credentials.RecordCreateOpts) error { client.Endpoint = original + "changed/"; return cause })
		if !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
			t.Fatal(err, calls)
		}
	})
}

func TestPythonCredentialAcceptedResponseProofAndEscapedMember(t *testing.T) {
	t.Run("read close causes retained", func(t *testing.T) {
		var calls []nativeCredCall
		api, cloud := nativeCredAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{}` })
		readErr, closeErr := errors.New("read failed"), errors.New("close failed")
		cloud.SetTransport(testcloud.LinkRequest(func(*http.Request) (*http.Response, error) {
			return testcloud.StreamResponse(200, &credentialRecordFaultBody{data: []byte(`{"credential":`), read: readErr, close: closeErr}, http.Header{"X-Request-Id": {"partial"}}), nil
		}))
		v, err := api.GetRecord(context.Background(), "a")
		var proof *resource.ResponseError
		if v == nil || !errors.Is(err, readErr) || !errors.Is(err, closeErr) || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-Id") != "partial" || string(v.Envelope) != `{"credential":` {
			t.Fatal(v, err, proof)
		}
	})
	t.Run("literal member escaped once", func(t *testing.T) {
		var calls []nativeCredCall
		api, _ := nativeCredAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.EscapedPath() != "/keystone/v3/credentials/folder%2Fname%25" {
				t.Error(req.URL.EscapedPath())
			}
			return 200, `{}`
		})
		_, err := api.GetRecord(context.Background(), "folder/name%")
		if err != nil || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
}

type credentialRecordFaultBody struct {
	data            []byte
	read, close     error
	closes          int
	onRead, onClose func()
}

func (b *credentialRecordFaultBody) Read(p []byte) (int, error) {
	if len(b.data) > 0 {
		n := copy(p, b.data)
		b.data = b.data[n:]
		if b.onRead != nil {
			b.onRead()
		}
		return n, nil
	}
	if b.onRead != nil {
		b.onRead()
	}
	return 0, b.read
}
func (b *credentialRecordFaultBody) Close() error {
	b.closes++
	if b.onClose != nil {
		b.onClose()
	}
	return b.close
}

func TestPythonCredentialRejectedResponseFailuresStopDeleteAndFind(t *testing.T) {
	for _, operation := range []string{"delete default", "delete strict", "find"} {
		statuses := []int{404}
		if operation == "find" {
			statuses = []int{400, 403, 404}
		}
		for _, status := range statuses {
			for _, kind := range []string{"read", "close", "read and close", "cancel after read", "cancel after close", "source restored by Close", "retry callback"} {
				t.Run(fmt.Sprintf("%s/%d/%s", operation, status, kind), func(t *testing.T) {
					var serverCalls []nativeCredCall
					api, cloud := nativeCredAPI(t, &serverCalls, func(*http.Request) (int, string) { return 200, `{}` })
					calls := 0
					readCause, closeCause, cancelCause := errors.New("physical rejected Read"), errors.New("physical rejected Close"), errors.New("caller canceled in rejected body")
					retryCause := errors.New("rejected callback failed")
					if kind == "retry callback" {
						api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return retryCause }
					}
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					body := &credentialRecordFaultBody{data: []byte("physical rejection"), read: io.EOF}
					if kind == "read" || kind == "read and close" {
						body.read = readCause
					}
					if kind == "close" || kind == "read and close" {
						body.close = closeCause
					}
					if kind == "cancel after read" {
						body.onRead = func() { cancel(cancelCause) }
					}
					if kind == "cancel after close" {
						body.onClose = func() { cancel(cancelCause) }
					}
					originalEndpoint := api.RawClient().Endpoint
					if kind == "source restored by Close" {
						body.onRead = func() { api.RawClient().Endpoint = originalEndpoint + "changed/" }
						body.onClose = func() { api.RawClient().Endpoint = originalEndpoint }
					}
					cloud.SetTransport(testcloud.LinkRequest(func(*http.Request) (*http.Response, error) {
						calls++
						if calls > 1 {
							return testcloud.Response(200, `{"credentials":[{"id":"found","name":"ci"}]}`, make(http.Header)), nil
						}
						return testcloud.StreamResponse(status, body, http.Header{"X-Request-Id": {"rejected-proof"}}), nil
					}))
					var err error
					if operation == "find" {
						value, findErr := api.FindRecord(ctx, "ci")
						err = findErr
						if value != nil {
							t.Fatal("faulty member rejection produced a lookup result", value, err)
						}
					} else if strings.Contains(operation, "strict") {
						err = api.DeleteRecord(ctx, "ci", credentials.WithRecordDeleteIgnoreMissing(false))
					} else {
						err = api.DeleteRecord(ctx, "ci")
					}
					if err == nil || calls != 1 || body.closes != 1 {
						t.Fatal("faulty rejection established absence or retried", err, calls, body.closes)
					}
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != status || string(native.Body) != "physical rejection" || native.ResponseHeader.Get("X-Request-Id") != "rejected-proof" {
						t.Fatal("native rejection proof lost", err, native)
					}
					if kind == "read" || kind == "read and close" {
						if !errors.Is(err, readCause) {
							t.Fatal(err)
						}
					}
					if kind == "close" || kind == "read and close" {
						if !errors.Is(err, closeCause) {
							t.Fatal(err)
						}
					}
					if kind == "retry callback" && !errors.Is(err, retryCause) {
						t.Fatal(err)
					}
					if strings.HasPrefix(kind, "cancel") {
						if !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) {
							t.Fatal(err)
						}
					}
					if kind == "source restored by Close" {
						if !errors.Is(err, resource.ErrInvalidOption) || api.RawClient().Endpoint != originalEndpoint {
							t.Fatal("source read failure lost after restoration", err)
						}
					}
				})
			}
		}
	}
}

func TestPythonCredentialCleanRejectionsRemainEligibleAfterNativeRetryHook(t *testing.T) {
	for _, operation := range []string{"delete default", "delete strict", "find"} {
		statuses := []int{404}
		if operation == "find" {
			statuses = []int{400, 403, 404}
		}
		for _, status := range statuses {
			t.Run(fmt.Sprintf("%s/%d", operation, status), func(t *testing.T) {
				var serverCalls []nativeCredCall
				api, cloud := nativeCredAPI(t, &serverCalls, func(*http.Request) (int, string) { return 200, `{}` })
				ctx := context.Background()
				calls, retries := 0, 0
				body := &credentialRecordFaultBody{data: []byte("clean rejection"), read: io.EOF}
				api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
					retries++
					return original
				}
				cloud.SetTransport(testcloud.LinkRequest(func(*http.Request) (*http.Response, error) {
					calls++
					if calls == 1 {
						return testcloud.StreamResponse(status, body, http.Header{"X-Request-Id": {"clean-proof"}}), nil
					}
					return testcloud.Response(200, `{"credentials":[{"id":"found","name":"ci"}]}`, make(http.Header)), nil
				}))
				if operation == "find" {
					value, err := api.FindRecord(ctx, "ci")
					if err != nil || calls != 2 {
						t.Fatal(value, err, calls)
					}
					credentialRecordField(t, value, "id", `"found"`)
				} else {
					var err error
					if strings.Contains(operation, "strict") {
						err = api.DeleteRecord(ctx, "ci", credentials.WithRecordDeleteIgnoreMissing(false))
					} else {
						err = api.DeleteRecord(ctx, "ci")
					}
					if !strings.Contains(operation, "strict") {
						if err != nil {
							t.Fatal("identical native retry rejection was marked mixed", err)
						}
					} else {
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "clean rejection" {
							t.Fatal(err, native)
						}
					}
					if calls != 1 {
						t.Fatal("stop hook replayed DELETE", calls)
					}
				}
				if retries != 1 || body.closes != 1 {
					t.Fatal(retries, body.closes)
				}
			})
		}
	}
}
