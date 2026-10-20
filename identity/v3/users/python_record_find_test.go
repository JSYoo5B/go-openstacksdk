package users_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestPythonUserFindRecordGETFirstSeedDescriptorsAndUnknownWire(t *testing.T) {
	var calls []pythonUserCall
	api := recordAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.URL.Path != "/keystone/v3/users/alice" || r.URL.Query().Get("domain_id") != "d" || r.URL.Query().Get("is_enabled") != "True" || r.URL.Query().Get("description") != "seed" || r.URL.Query().Get("x_seed") != "unknown" {
			t.Fatal(r.URL)
		}
		reply := recordUserWire(203, `{"user":{"id":"actual","name":"different","enabled":0,"federated":"scalar","options":false,"x_extension":{"keep":1}}}`)
		reply.Header.Set("X-Request-Id", "member-proof")
		return reply
	})
	row, err := api.FindRecord(context.Background(), "alice", sdk.WithRecordFindQuery("domain_id", "d"), sdk.WithRecordFindQuery("is_enabled", true), sdk.WithRecordFindQuery("description", "seed"), sdk.WithRecordFindQuery("x_seed", "unknown"))
	if err != nil || row == nil || len(calls) != 1 || row.StatusCode != 203 || row.Header.Get("X-Request-Id") != "member-proof" {
		t.Fatal(row, err, calls)
	}
	for key, want := range map[string]string{"id": `"actual"`, "name": `"different"`, "domain_id": `"d"`, "description": `"seed"`, "is_enabled": `false`, "federated": `["scalar"]`, "options": `{}`, "email": `null`} {
		recordField(t, row, key, want)
	}
	for _, key := range []string{"x_seed", "x_extension"} {
		if _, ok := row.Resource.Body[key]; ok {
			t.Fatal(row)
		}
	}
	if string(row.Wire.Body["x_extension"]) != `{"keep":1}` || !strings.Contains(string(row.Envelope), `"x_extension"`) {
		t.Fatal(row)
	}
}
func TestPythonUserFindRecordFallbackClassifiesFiltersAliasesAndAllPages(t *testing.T) {
	var calls []pythonUserCall
	api := recordAPI(t, &calls, func(r *http.Request) *http.Response {
		q := r.URL.Query()
		if strings.HasSuffix(r.URL.Path, "/alice") {
			if q.Get("description") != "keep" || q.Get("is_enabled") != "False" || q.Get("x_ignored") != "sent-get" {
				t.Fatal(r.URL)
			}
			return recordUserWire(403, `{}`)
		}
		if q.Get("name") != "alice" || q.Get("enabled") != "False" || q.Get("domain_id") != "d" || q.Has("description") || q.Has("is_enabled") || q.Has("x_ignored") {
			t.Fatal(r.URL)
		}
		if q.Get("marker") == "one" {
			return recordUserWire(200, `{"users":[{"id":"u","name":"alice","description":"keep","enabled":false}],"links":[]}`)
		}
		return recordUserWire(200, `{"users":[{"id":"filtered","name":"alice","description":"other"}],"links":[{"rel":"next","href":"?domain_id=d&enabled=False&name=alice&marker=one"}]}`)
	})
	row, err := api.FindRecord(context.Background(), "alice", sdk.WithRecordFindQuery("domain_id", "d"), sdk.WithRecordFindQuery("description", "keep"), sdk.WithRecordFindQuery("is_enabled", false), sdk.WithRecordFindQuery("x_ignored", "sent-get"))
	if err != nil || row == nil || len(calls) != 3 {
		t.Fatal(row, err, calls)
	}
	recordField(t, row, "id", `"u"`)
}
func TestPythonUserFindRecordMissingDuplicatesAndLateFailures(t *testing.T) {
	for _, mode := range []string{"missing", "strict", "duplicate", "late-error"} {
		t.Run(mode, func(t *testing.T) {
			var calls []pythonUserCall
			api := recordAPI(t, &calls, func(r *http.Request) *http.Response {
				if strings.HasSuffix(r.URL.Path, "/alice") {
					return recordUserWire(404, `{}`)
				}
				if mode == "missing" || mode == "strict" {
					return recordUserWire(200, `{"users":[]}`)
				}
				if r.URL.Query().Get("marker") == "u" {
					if mode == "late-error" {
						return recordUserWire(500, `{"error":"late"}`)
					}
					return recordUserWire(200, `{"users":[{"id":"v","name":"alice"}]}`)
				}
				return recordUserWire(200, `{"users":[{"id":"u","name":"alice"}],"links":[{"rel":"next","href":"?name=alice&marker=u"}]}`)
			})
			opts := []sdk.RecordFindOption{}
			if mode == "strict" {
				opts = append(opts, sdk.WithRecordFindIgnoreMissing(false))
			}
			row, err := api.FindRecord(context.Background(), "alice", opts...)
			if mode == "missing" {
				if row != nil || err != nil {
					t.Fatal(row, err)
				}
				return
			}
			if err == nil {
				t.Fatal(row)
			}
			if mode == "strict" && !errors.Is(err, resource.ErrNotFound) {
				t.Fatal(err)
			}
			if mode == "duplicate" && !errors.Is(err, resource.ErrAmbiguous) {
				t.Fatal(err)
			}
			if mode == "late-error" && len(calls) != 3 {
				t.Fatal(calls)
			}
		})
	}
}
func TestPythonUserFindRecordOwnsOptionsAndNamesFallbackPath(t *testing.T) {
	var calls []pythonUserCall
	raw := json.RawMessage(`"owned"`)
	opt := sdk.WithRecordFindOpts(sdk.RecordFindOpts{Query: map[string]json.RawMessage{"domain_id": raw}, Headers: map[string]string{"X-Owned": "yes"}})
	raw[1] = 'X'
	path := "/alternate-users"
	api := recordAPI(t, &calls, func(r *http.Request) *http.Response {
		if r.Header.Get("X-Owned") != "yes" {
			t.Fatal(r.Header)
		}
		if strings.HasSuffix(r.URL.Path, "/alice") {
			if r.URL.Query().Get("domain_id") != "owned" {
				t.Fatal(r.URL)
			}
			return recordUserWire(400, `{}`)
		}
		if !strings.HasSuffix(r.URL.Path, path) || r.URL.Query().Get("domain_id") != "owned" || r.URL.Query().Get("name") != "explicit" {
			t.Fatal(r.URL)
		}
		return recordUserWire(200, `{"users":[{"id":"alice","name":"explicit"}]}`)
	})
	row, err := api.FindRecord(context.Background(), "alice", opt, sdk.WithRecordFindListBasePath(path), sdk.WithRecordFindQuery("name", "explicit"))
	if err != nil || row == nil || len(calls) != 2 {
		t.Fatal(row, err, calls)
	}
	recordField(t, row, "id", `"alice"`)
}
func TestPythonUserFindRecordPreflightAndReservedControls(t *testing.T) {
	var calls []pythonUserCall
	api := recordAPI(t, &calls, func(*http.Request) *http.Response { t.Fatal("unexpected HTTP"); return nil })
	for _, key := range []string{"resource_type", "connection", "session", "_synchronized", "self"} {
		t.Run(key, func(t *testing.T) {
			row, err := api.FindRecord(context.Background(), "alice", sdk.WithRecordFindQuery(key, true))
			if row != nil || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
				t.Fatal(row, err, calls)
			}
		})
	}
	for _, opts := range [][]sdk.RecordFindOption{{nil}, {sdk.WithRecordFindQuery("name", make(chan int))}} {
		row, err := api.FindRecord(context.Background(), "alice", opts...)
		if row != nil || err == nil || len(calls) != 0 {
			t.Fatal(row, err, calls)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := api.FindRecord(ctx, "alice")
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

// Source interprets a mapping as one-key dictionaries; a {"next":url}
// shape therefore has no rel/href pair and does not advertise another page.
func TestPythonUserFindRecordIgnoresDictionaryNextLikeSource(t *testing.T) {
	var calls []pythonUserCall
	api := recordAPI(t, &calls, func(r *http.Request) *http.Response {
		if strings.HasSuffix(r.URL.Path, "/alice") {
			return recordUserWire(404, `{}`)
		}
		if r.URL.Query().Has("marker") {
			t.Fatal("Source ignores dictionary next", r.URL)
		}
		return recordUserWire(200, `{"users":[{"id":"u","name":"alice"}],"links":{"next":"?name=alice&marker=unexpected"}}`)
	})
	row, err := api.FindRecord(context.Background(), "alice")
	if err != nil || row == nil || len(calls) != 2 {
		t.Fatal(row, err, calls)
	}
	recordField(t, row, "id", `"u"`)
}
