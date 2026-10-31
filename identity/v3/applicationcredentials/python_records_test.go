package applicationcredentials_test

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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func recordIDs(t *testing.T, api *applicationcredentials.API, rules bool, opts ...applicationcredentials.RecordListOption) ([]string, error) {
	t.Helper()
	seq := api.ListCredentialRecords(context.Background(), "u1", opts...)
	if rules {
		seq = api.ListAccessRuleRecords(context.Background(), "u1", opts...)
	}
	var ids []string
	for v, err := range seq {
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
func recordField(t *testing.T, v *applicationcredentials.ApplicationCredentialRecord, key, want string) {
	t.Helper()
	if v == nil || v.Resource == nil || string(v.Resource.Body[key]) != want {
		t.Fatalf("%s: got %#v, want %s", key, v, want)
	}
}

func TestPythonApplicationCredentialCreateAndOwnedViews(t *testing.T) {
	var calls []nativeAppCredCall
	api, _ := nativeAppCredAPI(t, &calls, func(*http.Request) (int, string) {
		return 201, `{"application_credential":{"id":"ac-1","unrestricted":"yes","user_id":"attacker","expires_at":"arbitrary date","extension":9007199254740993}}`
	})
	fields := map[string]json.RawMessage{"description": json.RawMessage(`""`), "roles": json.RawMessage(`null`), "access_rules": json.RawMessage(`[]`), "unrestricted": json.RawMessage(`false`), "extension": json.RawMessage(`{"bad"`)}
	option := applicationcredentials.WithRecordCreateOpts(applicationcredentials.RecordCreateOpts{Fields: fields})
	fields["description"][1] = 'x'
	fields["roles"] = json.RawMessage(`"changed"`)
	v, err := api.CreateCredentialRecord(context.Background(), "u1", "ci", option, applicationcredentials.WithRecordCreateField("unknown", make(chan int)))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].path != "/keystone/v3/users/u1/application_credentials" || calls[0].method != http.MethodPost || calls[0].query != "" {
		t.Fatal(calls)
	}
	want := `{"application_credential":{"access_rules":[],"description":"","name":"ci","roles":null,"unrestricted":false}}`
	if calls[0].body != want {
		t.Fatal(calls[0].body)
	}
	recordField(t, v, "name", `"ci"`)
	recordField(t, v, "user_id", `"u1"`)
	recordField(t, v, "expires_at", `"arbitrary date"`)
	recordField(t, v, "unrestricted", `true`)
	recordField(t, v, "roles", `null`)
	recordField(t, v, "secret", `null`)
	recordField(t, v, "location", `null`)
	if len(v.Resource.Body) != 13 || string(v.Wire.Body["extension"]) != "9007199254740993" || string(v.Wire.Body["user_id"]) != `"attacker"` || v.StatusCode != 201 {
		t.Fatal(v)
	}
	v.Resource.Body["expires_at"][1] = 'X'
	if string(v.Wire.Body["expires_at"]) != `"arbitrary date"` || !strings.Contains(string(v.Envelope), "arbitrary date") {
		t.Fatal("view aliases wire or envelope")
	}
	v.Header.Set("X-Owned", "one")
	if v.Resource.Header.Get("X-Owned") != "" || v.Wire.Header.Get("X-Owned") != "" {
		t.Fatal("header aliases")
	}
	calls = nil
	if _, err := api.CreateCredentialRecord(context.Background(), "u1", "ci"); err != nil {
		t.Fatal(err)
	}
	if calls[0].body != `{"application_credential":{"name":"ci"}}` {
		t.Fatal("omitted unrestricted injected", calls)
	}
}

func TestPythonApplicationCredentialGetTranslationAndStatuses(t *testing.T) {
	for _, rules := range []bool{false, true} {
		for _, item := range []struct {
			name, body string
			valid      bool
		}{
			{"wrapped", `{"application_credential":{"id":"response","expires_at":42},"access_rule":{"id":"response","method":42}}`, true},
			{"root", `{"id":"response","extension":{"n":9007199254740993}}`, true},
			{"empty", "", true}, {"invalid", "{broken", true}, {"empty object", `{}`, true},
			{"array", `[]`, false}, {"null", `null`, false}, {"wrapped null", `{"application_credential":null,"access_rule":null}`, false},
		} {
			t.Run(fmt.Sprintf("rules=%t/%s", rules, item.name), func(t *testing.T) {
				var calls []nativeAppCredCall
				api, _ := nativeAppCredAPI(t, &calls, func(*http.Request) (int, string) { return 200, item.body })
				var v *applicationcredentials.ApplicationCredentialRecord
				var err error
				if rules {
					v, err = api.GetAccessRuleRecord(context.Background(), "u1", "selected")
				} else {
					v, err = api.GetCredentialRecord(context.Background(), "u1", "selected")
				}
				if (err == nil) != item.valid || len(calls) != 1 || calls[0].query != "" {
					t.Fatal(v, err, calls)
				}
				if !item.valid {
					var proof *resource.ResponseError
					if v == nil || !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != item.body {
						t.Fatal(v, err, proof)
					}
					return
				}
				if v.StatusCode != 200 || string(v.Envelope) != item.body {
					t.Fatal(v)
				}
				recordField(t, v, "user_id", `"u1"`)
				recordField(t, v, "location", `null`)
				if item.name == "empty" || item.name == "invalid" || item.name == "empty object" {
					recordField(t, v, "id", `"selected"`)
				}
				if rules && len(v.Resource.Body) != 8 || !rules && len(v.Resource.Body) != 13 {
					t.Fatal(v.Resource.Body)
				}
			})
		}
	}
	for _, code := range []int{200, 201, 202, 204, 206, 299, 301, 399, 400, 403, 404, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []nativeAppCredCall
			api, _ := nativeAppCredAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
			v, err := api.GetCredentialRecord(context.Background(), "u1", "ac")
			if code < 400 {
				if err != nil || v == nil || v.StatusCode != code {
					t.Fatal(v, err)
				}
			} else {
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(err, &native) || native.Actual != code || string(native.Body) != `{}` || v != nil {
					t.Fatal(v, err, native)
				}
			}
		})
	}
}

func TestPythonApplicationCredentialListFiltersRawCapAndPaging(t *testing.T) {
	for _, rules := range []bool{false, true} {
		t.Run(fmt.Sprintf("rules=%t", rules), func(t *testing.T) {
			plural := "application_credentials"
			if rules {
				plural = "access_rules"
			}
			var calls []nativeAppCredCall
			api, _ := nativeAppCredAPI(t, &calls, func(req *http.Request) (int, string) {
				if req.URL.Query().Get("marker") == "next" {
					return 200, `{"` + plural + `":[{"id":"ac2","name":"ci","roles":{"unexpected":true},"method":"GET"}],"links":{"next":null}}`
				}
				return 200, `{"` + plural + `":[{"id":"skip","name":"other"},{"id":"ac1","name":"ci","unrestricted":false,"method":"GET"}],"links":{"next":"?marker=next"}}`
			})
			ids, err := recordIDs(t, api, rules, applicationcredentials.WithRecordListFilter("name", "ci"), applicationcredentials.WithRecordListFilter("unknown", make(chan int)))
			if err != nil || !reflect.DeepEqual(ids, []string{"ac1", "ac2"}) || len(calls) != 2 || calls[0].query != "" || calls[1].query != "marker=next" {
				t.Fatal(ids, err, calls)
			}
			calls = nil
			ids, err = recordIDs(t, api, rules, applicationcredentials.WithRecordListFilter("name", "ci"), applicationcredentials.WithRecordListMaxItems(1))
			if err != nil || len(ids) != 0 || len(calls) != 1 || calls[0].query != "limit=1" {
				t.Fatal("max_items counts observed rows before filtering", ids, err, calls)
			}
			calls = nil
			ids, err = recordIDs(t, api, rules, applicationcredentials.WithRecordListPaginated(false), applicationcredentials.WithRecordListLimit(0), applicationcredentials.WithRecordListMarker("start"))
			if err != nil || !reflect.DeepEqual(ids, []string{"skip", "ac1"}) || len(calls) != 1 || calls[0].query != "limit=0&marker=start" {
				t.Fatal(ids, err, calls)
			}
			calls = nil
			ids, err = recordIDs(t, api, rules, applicationcredentials.WithRecordListMaxItems(1), applicationcredentials.WithRecordListLimit(0))
			if err != nil || !reflect.DeepEqual(ids, []string{"skip"}) || calls[0].query != "limit=1" {
				t.Fatal(ids, err, calls)
			}
		})
	}
	t.Run("dictionary filter and absent versus null", func(t *testing.T) {
		var calls []nativeAppCredCall
		api, _ := nativeAppCredAPI(t, &calls, func(*http.Request) (int, string) {
			return 200, `{"application_credentials":[{"id":"a","links":{"self":"x","extra":true}},{"id":"b","links":{}},{"id":"c"}],"links":{"next":null}}`
		})
		ids, err := recordIDs(t, api, false, applicationcredentials.WithRecordListFilter("links", map[string]any{"self": "x"}), applicationcredentials.WithRecordListFilter("description", nil))
		if err != nil || !reflect.DeepEqual(ids, []string{"a"}) {
			t.Fatal(ids, err)
		}
	})
	t.Run("marker fallback follows short pages", func(t *testing.T) {
		var calls []nativeAppCredCall
		api, _ := nativeAppCredAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.Query().Get("marker") == "a" {
				return 200, `{"application_credentials":[]}`
			}
			return 200, `{"application_credentials":[{"id":"a"}]}`
		})
		ids, err := recordIDs(t, api, false, applicationcredentials.WithRecordListLimit(2))
		if err != nil || !reflect.DeepEqual(ids, []string{"a"}) || len(calls) != 2 || calls[1].query != "limit=2&marker=a" {
			t.Fatal(ids, err, calls)
		}
	})
	t.Run("singleton and malformed unused row", func(t *testing.T) {
		var calls []nativeAppCredCall
		api, _ := nativeAppCredAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.Query().Get("limit") == "1" {
				return 200, `{"application_credentials":[{"id":"a"},null],"links":{"next":"?marker=next"}}`
			}
			return 200, `{"application_credentials":{"id":"one"},"links":{"next":null}}`
		})
		ids, err := recordIDs(t, api, false)
		if err != nil || !reflect.DeepEqual(ids, []string{"one"}) {
			t.Fatal(ids, err)
		}
		calls = nil
		ids, err = recordIDs(t, api, false, applicationcredentials.WithRecordListMaxItems(1))
		if err != nil || !reflect.DeepEqual(ids, []string{"a"}) || len(calls) != 1 {
			t.Fatal(ids, err, calls)
		}
	})
}

func TestPythonApplicationCredentialListContinuationsAndLazyErrors(t *testing.T) {
	for _, field := range []string{"next", "links", "application_credentials_links"} {
		t.Run(field, func(t *testing.T) {
			var calls []nativeAppCredCall
			api, _ := nativeAppCredAPI(t, &calls, func(req *http.Request) (int, string) {
				if req.URL.Query().Get("marker") == "next" {
					return 200, `{"application_credentials":[],"links":{"next":null}}`
				}
				next := `"next":"?marker=next"`
				if field == "links" {
					next = `"links":{"next":"?marker=next"}`
				}
				if field == "application_credentials_links" {
					next = `"application_credentials_links":[{"rel":"next","href":"?marker=next"}]`
				}
				return 200, `{"application_credentials":[{"id":"a"}],` + next + `}`
			})
			ids, err := recordIDs(t, api, false)
			if err != nil || !reflect.DeepEqual(ids, []string{"a"}) || len(calls) != 2 {
				t.Fatal(ids, err, calls)
			}
		})
	}
	t.Run("HTTP Link", func(t *testing.T) {
		var calls []nativeAppCredCall
		api, cloud := nativeAppCredAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{}` })
		attempts := 0
		cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
			attempts++
			body := `{"application_credentials":[{"id":"a"}]}`
			h := http.Header{"Link": {`<?marker=next>; rel="next"`}}
			if attempts == 2 {
				body = `{"application_credentials":[]}`
				h = http.Header{}
			}
			return testcloud.Response(200, body, h), nil
		}))
		ids, err := recordIDs(t, api, false)
		if err != nil || !reflect.DeepEqual(ids, []string{"a"}) || attempts != 2 {
			t.Fatal(ids, err, attempts)
		}
	})
	t.Run("late failure and early break", func(t *testing.T) {
		var calls []nativeAppCredCall
		api, _ := nativeAppCredAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.Query().Get("marker") == "next" {
				return 500, `{"error":"later"}`
			}
			return 200, `{"application_credentials":[{"id":"a"}],"links":{"next":"?marker=next"}}`
		})
		ids, err := recordIDs(t, api, false)
		var native gophercloud.ErrUnexpectedResponseCode
		if !reflect.DeepEqual(ids, []string{"a"}) || !errors.As(err, &native) || native.Actual != 500 {
			t.Fatal(ids, err)
		}
		calls = nil
		for _, err := range api.ListCredentialRecords(context.Background(), "u1") {
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

func TestPythonApplicationCredentialDeleteDefaultAndMixedFailures(t *testing.T) {
	for _, rules := range []bool{false, true} {
		for _, code := range []int{200, 202, 204, 301, 399, 400, 403, 404, 500} {
			t.Run(fmt.Sprintf("rules=%t/%d", rules, code), func(t *testing.T) {
				var calls []nativeAppCredCall
				api, _ := nativeAppCredAPI(t, &calls, func(*http.Request) (int, string) { return code, "opaque body" })
				remove := api.DeleteCredentialRecord
				if rules {
					remove = api.DeleteAccessRuleRecord
				}
				err := remove(context.Background(), "u1", "selected")
				if (err == nil) != (code < 400 || code == 404) || len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].query != "" {
					t.Fatal(err, calls)
				}
				if code == 404 {
					err = remove(context.Background(), "u1", "selected", applicationcredentials.WithRecordDeleteIgnoreMissing(false))
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != "opaque body" {
						t.Fatal(err, native)
					}
				}
			})
		}
	}
	t.Run("404 joined retry callback failure stays visible", func(t *testing.T) {
		var calls []nativeAppCredCall
		api, _ := nativeAppCredAPI(t, &calls, func(*http.Request) (int, string) { return 404, "missing" })
		cause := errors.New("retry callback failed")
		api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return cause }
		err := api.DeleteCredentialRecord(context.Background(), "u1", "a")
		if !errors.Is(err, cause) || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
}

func TestPythonApplicationCredentialFindSourceDefaultsAndQueries(t *testing.T) {
	for _, code := range []int{400, 403, 404} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []nativeAppCredCall
			api, _ := nativeAppCredAPI(t, &calls, func(req *http.Request) (int, string) {
				if strings.HasSuffix(req.URL.Path, "/ci") {
					return code, "missing"
				}
				if req.URL.Query().Get("marker") == "next" {
					return 200, `{"application_credentials":[{"id":"match","name":"ci"}],"links":{"next":null}}`
				}
				return 200, `{"application_credentials":[{"id":"other","name":"other"}],"links":{"next":"?marker=next"}}`
			})
			v, err := api.FindCredentialRecord(context.Background(), "u1", "ci", resource.WithIdentityFindQuery("name", "ci"), resource.WithIdentityFindQuery("ignored", "value"))
			if err != nil {
				t.Fatal(err)
			}
			recordField(t, v, "id", `"match"`)
			if len(calls) != 3 || calls[0].query != "ignored=value&name=ci&user_id=u1" || calls[1].query != "" || calls[2].query != "marker=next" {
				t.Fatal(calls)
			}
		})
	}
	t.Run("direct query Body seed survives absent response", func(t *testing.T) {
		var calls []nativeAppCredCall
		api, _ := nativeAppCredAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{}` })
		v, err := api.FindCredentialRecord(context.Background(), "u1", "selected", resource.WithIdentityFindQuery("name", "caller"))
		if err != nil {
			t.Fatal(err)
		}
		recordField(t, v, "id", `"selected"`)
		recordField(t, v, "name", `"caller"`)
		if calls[0].query != "name=caller&user_id=u1" {
			t.Fatal(calls)
		}
	})
	t.Run("direct seeded success suppresses fallback", func(t *testing.T) {
		var calls []nativeAppCredCall
		api, _ := nativeAppCredAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"application_credential":{"name":"ci"}}` })
		v, err := api.FindCredentialRecord(context.Background(), "u1", "selected")
		if err != nil || len(calls) != 1 || calls[0].query != "user_id=u1" {
			t.Fatal(v, err, calls)
		}
		recordField(t, v, "id", `"selected"`)
	})
	for _, item := range []struct {
		name, list string
		strict     bool
		kind       string
	}{
		{"missing", `[]`, false, ""}, {"strict missing", `[]`, true, "missing"}, {"duplicate", `[{"id":"a","name":"ci"},{"id":"b","name":"ci"}]`, false, "ambiguous"}, {"ID match", `[{"id":"ci","name":"other"}]`, false, ""},
	} {
		t.Run(item.name, func(t *testing.T) {
			var calls []nativeAppCredCall
			api, _ := nativeAppCredAPI(t, &calls, func(req *http.Request) (int, string) {
				if strings.HasSuffix(req.URL.Path, "/ci") {
					return 404, "missing"
				}
				return 200, `{"application_credentials":` + item.list + `,"links":{"next":null}}`
			})
			v, err := api.FindCredentialRecord(context.Background(), "u1", "ci", resource.WithIdentityFindIgnoreMissing(!item.strict))
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
				if err != nil {
					t.Fatal(err)
				}
				if item.name == "missing" && v != nil {
					t.Fatal(v)
				}
			}
			if calls[0].query != "user_id=u1" || calls[1].query != "" {
				t.Fatal(calls)
			}
		})
	}
	t.Run("all pages observed before unique match", func(t *testing.T) {
		var calls []nativeAppCredCall
		api, _ := nativeAppCredAPI(t, &calls, func(req *http.Request) (int, string) {
			if strings.HasSuffix(req.URL.Path, "/ci") {
				return 404, "missing"
			}
			if req.URL.Query().Get("marker") == "next" {
				return 500, "later"
			}
			return 200, `{"application_credentials":[{"id":"a","name":"ci"}],"links":{"next":"?marker=next"}}`
		})
		v, err := api.FindCredentialRecord(context.Background(), "u1", "ci")
		var native gophercloud.ErrUnexpectedResponseCode
		if v != nil || !errors.As(err, &native) || native.Actual != 500 || len(calls) != 3 {
			t.Fatal(v, err, calls)
		}
	})
}

func TestPythonApplicationCredentialOptionsOwnershipAndPreflight(t *testing.T) {
	var calls []nativeAppCredCall
	api, _ := nativeAppCredAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.Header.Get("X-Owned") != "old" {
			t.Error(req.Header)
		}
		return 200, `{"application_credentials":[{"id":"a","name":"ci"}],"links":{"next":null}}`
	})
	limit := 2
	singlePage := false
	filters := map[string]json.RawMessage{"name": json.RawMessage(`"ci"`)}
	headers := map[string]string{"X-Owned": "old"}
	option := applicationcredentials.WithRecordListOpts(applicationcredentials.RecordListOpts{Limit: &limit, Paginated: &singlePage, Filters: filters, Headers: headers})
	limit = 9
	filters["name"] = json.RawMessage(`"changed"`)
	headers["X-Owned"] = "changed"
	seq := api.ListCredentialRecords(context.Background(), "u1", option)
	if len(calls) != 0 {
		t.Fatal("list was eager")
	}
	for range 2 {
		var ids []string
		for v, err := range seq {
			if err != nil {
				t.Fatal(err)
			}
			var id string
			if err := json.Unmarshal(v.Resource.Body["id"], &id); err != nil {
				t.Fatal(err)
			}
			ids = append(ids, id)
		}
		if !reflect.DeepEqual(ids, []string{"a"}) {
			t.Fatal(ids)
		}
	}
	if len(calls) != 2 || calls[0].query != "limit=2" {
		t.Fatal(calls)
	}
	t.Run("retained option pointer cannot change applied policy", func(t *testing.T) {
		var retained *applicationcredentials.RecordListOpts
		ids, err := recordIDs(t, api, false, func(c *applicationcredentials.RecordListOpts) error {
			retained = c
			c.Headers = map[string]string{"X-Owned": "old"}
			c.Filters = map[string]json.RawMessage{"name": json.RawMessage(`"ci"`)}
			return nil
		}, func(*applicationcredentials.RecordListOpts) error {
			retained.Headers["X-Owned"] = "wrong"
			retained.Filters["name"] = json.RawMessage(`"wrong"`)
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
		{"nil read option", func() error { _, err := api.GetCredentialRecord(context.Background(), "u1", "a", nil); return err }},
		{"nil list option", func() error { _, err := recordIDs(t, api, false, nil); return err }},
		{"nil delete option", func() error { return api.DeleteCredentialRecord(context.Background(), "u1", "a", nil) }},
		{"reserved name", func() error {
			_, err := api.CreateCredentialRecord(context.Background(), "u1", "ci", applicationcredentials.WithRecordCreateField("name", "other"))
			return err
		}},
		{"fixed create user", func() error {
			_, err := api.CreateCredentialRecord(context.Background(), "u1", "ci", applicationcredentials.WithRecordCreateField("user_id", "u2"))
			return err
		}},
		{"bad JSON", func() error {
			_, err := api.CreateCredentialRecord(context.Background(), "u1", "ci", applicationcredentials.WithRecordCreateOpts(applicationcredentials.RecordCreateOpts{Fields: map[string]json.RawMessage{"roles": json.RawMessage(`{bad`)}}))
			return err
		}},
		{"negative max", func() error {
			_, err := recordIDs(t, api, false, applicationcredentials.WithRecordListMaxItems(-1))
			return err
		}},
		{"protected header", func() error {
			_, err := api.GetCredentialRecord(context.Background(), "u1", "a", applicationcredentials.WithRecordReadHeaders(map[string]string{"X-Auth-Token": "wrong"}))
			return err
		}},
		{"fixed find id", func() error {
			_, err := api.FindCredentialRecord(context.Background(), "u1", "a", resource.WithIdentityFindQuery("id", "other"))
			return err
		}},
		{"fixed find user", func() error {
			_, err := api.FindCredentialRecord(context.Background(), "u1", "a", resource.WithIdentityFindQuery("user_id", "u2"))
			return err
		}},
	} {
		t.Run(item.name, func(t *testing.T) {
			before := len(calls)
			if err := item.call(); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
				t.Fatal(err, calls)
			}
		})
	}
	t.Run("callback cancellation plus cause", func(t *testing.T) {
		ctx, cancel := context.WithCancelCause(context.Background())
		cause := errors.New("caller cause")
		callbackErr := errors.New("callback error")
		before := len(calls)
		_, err := api.GetCredentialRecord(ctx, "u1", "a", func(*applicationcredentials.RecordReadOpts) error { cancel(cause); return callbackErr })
		if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, callbackErr) || len(calls) != before {
			t.Fatal(err, calls)
		}
	})
	t.Run("callback source mutation plus error", func(t *testing.T) {
		client := api.RawClient()
		original := client.Endpoint
		defer func() { client.Endpoint = original }()
		cause := errors.New("callback error")
		before := len(calls)
		_, err := api.CreateCredentialRecord(context.Background(), "u1", "ci", func(*applicationcredentials.RecordCreateOpts) error {
			client.Endpoint = original + "different/"
			return cause
		})
		if !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
			t.Fatal(err, calls)
		}
	})
}

func TestPythonApplicationCredentialLiteralScopesAndResponseProof(t *testing.T) {
	t.Run("literal scopes avoid parent GET", func(t *testing.T) {
		var calls []nativeAppCredCall
		api, _ := nativeAppCredAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.Method == http.MethodDelete {
				return 204, "opaque"
			}
			if strings.Contains(req.URL.Path, "/access_rules") {
				if strings.HasSuffix(req.URL.Path, "/ar") {
					return 200, `{"access_rule":{"method":"GET"}}`
				}
				return 200, `{"access_rules":[],"links":{"next":null}}`
			}
			if req.Method == http.MethodPost {
				return 201, `{"application_credential":{"id":"created"}}`
			}
			if strings.HasSuffix(req.URL.Path, "/application_credentials") {
				return 200, `{"application_credentials":[],"links":{"next":null}}`
			}
			return 200, `{"application_credential":{"id":"ac"}}`
		})
		scope, err := api.InUser(context.Background(), resource.ID("u1"))
		if err != nil {
			t.Fatal(err)
		}
		rules, err := api.AccessRules(context.Background(), resource.ID("u1"))
		if err != nil || len(calls) != 0 {
			t.Fatal(err, calls)
		}
		if _, err := scope.GetRecord(context.Background(), "ac"); err != nil {
			t.Fatal(err)
		}
		if _, err := scope.CreateRecord(context.Background(), "ci"); err != nil {
			t.Fatal(err)
		}
		if _, err := scope.FindRecord(context.Background(), "ac"); err != nil {
			t.Fatal(err)
		}
		for _, err := range scope.ListRecords(context.Background()) {
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := scope.DeleteRecord(context.Background(), "ac"); err != nil {
			t.Fatal(err)
		}
		if _, err := rules.GetRecord(context.Background(), "ar"); err != nil {
			t.Fatal(err)
		}
		for _, err := range rules.ListRecords(context.Background()) {
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := rules.DeleteRecord(context.Background(), "ar"); err != nil {
			t.Fatal(err)
		}
		if len(calls) != 8 {
			t.Fatal(calls)
		}
		for _, call := range calls {
			if !strings.HasPrefix(call.path, "/keystone/v3/users/u1/") {
				t.Fatal(call)
			}
		}
	})
	t.Run("read close and accepted proof", func(t *testing.T) {
		var calls []nativeAppCredCall
		api, cloud := nativeAppCredAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{}` })
		readErr, closeErr := errors.New("read failed"), errors.New("close failed")
		cloud.SetTransport(testcloud.LinkRequest(func(*http.Request) (*http.Response, error) {
			return testcloud.StreamResponse(200, &appRecordFaultBody{data: []byte(`{"application_credential":`), read: readErr, close: closeErr}, http.Header{"X-Request-Id": {"partial"}}), nil
		}))
		v, err := api.GetCredentialRecord(context.Background(), "u1", "a")
		var proof *resource.ResponseError
		if v == nil || !errors.Is(err, readErr) || !errors.Is(err, closeErr) || !errors.As(err, &proof) || proof.StatusCode != 200 || proof.Header.Get("X-Request-Id") != "partial" || string(v.Envelope) != `{"application_credential":` {
			t.Fatal(v, err, proof)
		}
	})
	t.Run("member identity escapes once", func(t *testing.T) {
		var calls []nativeAppCredCall
		api, _ := nativeAppCredAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.EscapedPath() != "/keystone/v3/users/u1/application_credentials/folder%2Fname%25" {
				t.Error(req.URL.EscapedPath())
			}
			return 200, `{}`
		})
		_, err := api.GetCredentialRecord(context.Background(), "u1", "folder/name%")
		if err != nil || len(calls) != 1 {
			t.Fatal(err, calls)
		}
	})
}

type appRecordFaultBody struct {
	data            []byte
	read, close     error
	closes          int
	onRead, onClose func()
}

func (b *appRecordFaultBody) Read(p []byte) (int, error) {
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
func (b *appRecordFaultBody) Close() error {
	b.closes++
	if b.onClose != nil {
		b.onClose()
	}
	return b.close
}

func TestPythonApplicationCredentialRejectedResponseFailuresStopDeleteAndFind(t *testing.T) {
	for _, operation := range []string{"delete credential default", "delete credential strict", "delete access default", "delete access strict", "find"} {
		statuses := []int{404}
		if operation == "find" {
			statuses = []int{400, 403, 404}
		}
		for _, status := range statuses {
			for _, kind := range []string{"read", "close", "read and close", "cancel after read", "cancel after close", "source restored by Close", "retry callback"} {
				t.Run(fmt.Sprintf("%s/%d/%s", operation, status, kind), func(t *testing.T) {
					var serverCalls []nativeAppCredCall
					api, cloud := nativeAppCredAPI(t, &serverCalls, func(*http.Request) (int, string) { return 200, `{}` })
					remove := api.DeleteCredentialRecord
					if strings.Contains(operation, "access") {
						remove = api.DeleteAccessRuleRecord
					}
					calls := 0
					readCause, closeCause, cancelCause := errors.New("physical rejected Read"), errors.New("physical rejected Close"), errors.New("caller canceled in rejected body")
					retryCause := errors.New("rejected callback failed")
					if kind == "retry callback" {
						api.RawClient().RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error { return retryCause }
					}
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					body := &appRecordFaultBody{data: []byte("physical rejection"), read: io.EOF}
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
							return testcloud.Response(200, `{"application_credentials":[{"id":"found","name":"ci"}]}`, make(http.Header)), nil
						}
						return testcloud.StreamResponse(status, body, http.Header{"X-Request-Id": {"rejected-proof"}}), nil
					}))
					var err error
					if operation == "find" {
						value, findErr := api.FindCredentialRecord(ctx, "u1", "ci")
						err = findErr
						if value != nil {
							t.Fatal("faulty member rejection produced a lookup result", value, err)
						}
					} else if strings.Contains(operation, "strict") {
						err = remove(ctx, "u1", "ci", applicationcredentials.WithRecordDeleteIgnoreMissing(false))
					} else {
						err = remove(ctx, "u1", "ci")
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

func TestPythonApplicationCredentialCleanRejectionsRemainEligibleAfterNativeRetryHook(t *testing.T) {
	for _, operation := range []string{"delete credential default", "delete credential strict", "delete access default", "delete access strict", "find"} {
		statuses := []int{404}
		if operation == "find" {
			statuses = []int{400, 403, 404}
		}
		for _, status := range statuses {
			t.Run(fmt.Sprintf("%s/%d", operation, status), func(t *testing.T) {
				var serverCalls []nativeAppCredCall
				api, cloud := nativeAppCredAPI(t, &serverCalls, func(*http.Request) (int, string) { return 200, `{}` })
				remove := api.DeleteCredentialRecord
				if strings.Contains(operation, "access") {
					remove = api.DeleteAccessRuleRecord
				}
				ctx := context.Background()
				calls, retries := 0, 0
				body := &appRecordFaultBody{data: []byte("clean rejection"), read: io.EOF}
				api.RawClient().RetryFunc = func(_ context.Context, _ string, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
					retries++
					return original
				}
				cloud.SetTransport(testcloud.LinkRequest(func(*http.Request) (*http.Response, error) {
					calls++
					if calls == 1 {
						return testcloud.StreamResponse(status, body, http.Header{"X-Request-Id": {"clean-proof"}}), nil
					}
					return testcloud.Response(200, `{"application_credentials":[{"id":"found","name":"ci"}]}`, make(http.Header)), nil
				}))
				if operation == "find" {
					value, err := api.FindCredentialRecord(ctx, "u1", "ci")
					if err != nil || calls != 2 {
						t.Fatal(value, err, calls)
					}
					recordField(t, value, "id", `"found"`)
				} else {
					var err error
					if strings.Contains(operation, "strict") {
						err = remove(ctx, "u1", "ci", applicationcredentials.WithRecordDeleteIgnoreMissing(false))
					} else {
						err = remove(ctx, "u1", "ci")
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
