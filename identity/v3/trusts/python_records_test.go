package trusts_test

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

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func trustRecordField(t *testing.T, value *trusts.TrustRecord, key, expected string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != expected {
		t.Fatalf("field %s: got %#v want %s", key, value, expected)
	}
}
func trustRecordIDs(t *testing.T, api *trusts.API, options ...trusts.RecordListOption) ([]string, error) {
	t.Helper()
	var ids []string
	for value, err := range api.ListRecords(context.Background(), options...) {
		if err != nil {
			return ids, err
		}
		var id string
		if err := json.Unmarshal(value.Resource.Body["id"], &id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func TestPythonTrustCreateRawAttributesAliasesAndDescriptorViews(t *testing.T) {
	var calls []nativeTrustCall
	api, _ := nativeTrustAPI(t, &calls, func(*http.Request) (int, string) {
		return 201, `{"trust":{"id":"t1","impersonation":"false","remaining_uses":9007199254740993,"extension":[false,null]}}`
	})
	fields := map[string]json.RawMessage{
		"trustor_user_id": json.RawMessage(`"alice"`), "trustee_user_id": json.RawMessage(`null`),
		"is_impersonation": json.RawMessage(`false`), "allow_redelegation": json.RawMessage(`[]`),
		"roles": json.RawMessage(`[{"id":"member","extension":false}]`), "remaining_uses": json.RawMessage(`0`),
		"expires_at": json.RawMessage(`"unparsed-expiry"`), "project_id": json.RawMessage(`""`),
		"unknown": json.RawMessage(`{broken`),
	}
	owned := trusts.WithRecordCreateOpts(trusts.RecordCreateOpts{Fields: fields})
	fields["roles"][8] = 'X'
	fields["trustor_user_id"] = json.RawMessage(`"mutated"`)
	value, err := api.CreateRecord(context.Background(), owned, trusts.WithRecordCreateField("unknown", make(chan int)))
	if err != nil {
		t.Fatal(value, err)
	}
	expected := `{"trust":{"allow_redelegation":[],"expires_at":"unparsed-expiry","impersonation":false,"project_id":"","remaining_uses":0,"roles":[{"id":"member","extension":false}],"trustee_user_id":null,"trustor_user_id":"alice"}}`
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != "/keystone/v3/OS-TRUST/trusts" || calls[0].query != "" || calls[0].body != expected {
		t.Fatal(calls)
	}
	for key, want := range map[string]string{
		"id": `"t1"`, "is_impersonation": `true`, "allow_redelegation": `false`,
		"trustor_user_id": `"alice"`, "trustee_user_id": `null`, "remaining_uses": `9007199254740993`,
		"roles": `[{"id":"member","extension":false}]`, "expires_at": `"unparsed-expiry"`, "project_id": `""`, "location": `null`,
	} {
		trustRecordField(t, value, key, want)
	}
	if len(value.Resource.Body) != 15 || string(value.Wire.Body["impersonation"]) != `"false"` || string(value.Wire.Body["extension"]) != `[false,null]` {
		t.Fatal(value)
	}
	value.Resource.Body["remaining_uses"][0] = 'X'
	if string(value.Wire.Body["remaining_uses"]) != `9007199254740993` || !strings.Contains(string(value.Envelope), "9007199254740993") {
		t.Fatal("raw channels alias")
	}
	calls = nil
	if _, err := api.CreateRecord(context.Background()); err != nil || calls[0].body != `{"trust":{}}` {
		t.Fatal(err, calls)
	}
	calls = nil
	value, err = api.CreateRecord(context.Background(), trusts.WithRecordCreateField("impersonation", true), trusts.WithRecordCreateField("is_impersonation", nil), trusts.WithRecordCreateField("roles", false))
	if err != nil || calls[0].body != `{"trust":{"impersonation":null,"roles":false}}` {
		t.Fatal(value, err, calls)
	}
	trustRecordField(t, value, "roles", `false`)
	calls = nil
	_, err = api.CreateRecord(context.Background(),
		trusts.WithRecordCreateOpts(trusts.RecordCreateOpts{Fields: map[string]json.RawMessage{"is_impersonation": json.RawMessage(`true`)}}),
		trusts.WithRecordCreateField("impersonation", false))
	if err != nil || calls[0].body != `{"trust":{"impersonation":false}}` {
		t.Fatal(err, calls)
	}
}

func TestPythonTrustGetResponseTranslationAndBoolDefaults(t *testing.T) {
	for _, item := range []struct {
		name, body, impersonation, redelegation string
		valid                                   bool
	}{
		{"wire alias", `{"trust":{"impersonation":[],"allow_redelegation":"false","roles":42}}`, `false`, `true`, true},
		{"canonical alias", `{"is_impersonation":{},"allow_redelegation":null}`, `false`, `null`, true},
		{"null", `{"trust":{"impersonation":null}}`, `null`, `null`, true},
		{"missing", `{}`, `null`, `null`, true},
		{"empty", ``, `null`, `null`, true}, {"malformed", `{broken`, `null`, `null`, true},
		{"array", `[]`, ``, ``, false}, {"nonobject", `null`, ``, ``, false}, {"wrapped null", `{"trust":null}`, ``, ``, false},
		{"alias collision", `{"trust":{"is_impersonation":true,"impersonation":false}}`, ``, ``, false},
	} {
		t.Run(item.name, func(t *testing.T) {
			var calls []nativeTrustCall
			api, _ := nativeTrustAPI(t, &calls, func(*http.Request) (int, string) { return 200, item.body })
			value, err := api.GetRecord(context.Background(), "selected")
			if (err == nil) != item.valid || value == nil || len(calls) != 1 || calls[0].path != "/keystone/v3/OS-TRUST/trusts/selected" {
				t.Fatal(value, err, calls)
			}
			if !item.valid {
				var proof *resource.ResponseError
				if !errors.As(err, &proof) || proof.StatusCode != 200 || string(proof.Body) != item.body {
					t.Fatal(err, proof)
				}
				return
			}
			trustRecordField(t, value, "id", `"selected"`)
			trustRecordField(t, value, "is_impersonation", item.impersonation)
			trustRecordField(t, value, "allow_redelegation", item.redelegation)
			if len(value.Resource.Body) != 15 || value.StatusCode != 200 || string(value.Envelope) != item.body {
				t.Fatal(value)
			}
			if item.name == "wire alias" {
				trustRecordField(t, value, "roles", `42`)
			}
			value.Header.Set("X-Owned", "mutated")
			if value.Resource.Header.Get("X-Owned") != "" || value.Wire != nil && value.Wire.Header.Get("X-Owned") != "" {
				t.Fatal("headers alias")
			}
		})
	}
}

func TestPythonTrustListQueryClassificationLocalFiltersAndRawCap(t *testing.T) {
	var calls []nativeTrustCall
	api, _ := nativeTrustAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.Header.Get("Accept") != "application/json" {
			t.Error(req.Header)
		}
		return 200, `{"trusts":[{"id":"skip","impersonation":true},{"id":"found","impersonation":false,"roles":{"nested":{"name":"member"}},"trustor_user_id":"alice"}],"next":"?marker=next"}`
	})
	options := []trusts.RecordListOption{
		trusts.WithRecordListQuery("trustor_user_id", []any{"alice", true, nil}),
		trusts.WithRecordListQuery("trustee_user_id", nil),
		trusts.WithRecordListQuery("is_impersonation", false),
		trusts.WithRecordListQuery("roles", map[string]any{"nested": map[string]any{"name": "member"}}),
		trusts.WithRecordListQuery("impersonation", make(chan int)),
		trusts.WithRecordListQuery("unknown", make(chan int)),
		trusts.WithRecordListMaxItems(2),
	}
	ids, err := trustRecordIDs(t, api, options...)
	if err != nil || !reflect.DeepEqual(ids, []string{"found"}) || len(calls) != 1 || calls[0].query != "limit=2&trustor_user_id=alice&trustor_user_id=True" {
		t.Fatal(ids, err, calls)
	}
	calls = nil
	ids, err = trustRecordIDs(t, api, append(options, trusts.WithRecordListMaxItems(1))...)
	if err != nil || len(ids) != 0 || len(calls) != 1 || !strings.Contains(calls[0].query, "limit=1") {
		t.Fatal("max_items must count raw rows before local filters", ids, err, calls)
	}
	calls = nil
	ids, err = trustRecordIDs(t, api, trusts.WithRecordListPaginated(false), trusts.WithRecordListQuery("impersonation", false))
	if err != nil || !reflect.DeepEqual(ids, []string{"skip", "found"}) || len(calls) != 1 || calls[0].query != "" {
		t.Fatal("wire Body alias is not a Source local-filter name", ids, err, calls)
	}
}

func TestPythonTrustListPagingLazyOwnershipAndLaterErrors(t *testing.T) {
	var calls []nativeTrustCall
	api, _ := nativeTrustAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.Header.Get("X-Owned") != "old" {
			t.Error(req.Header)
		}
		if req.URL.Query().Get("marker") == "a" {
			return 200, `{"trusts":[]}`
		}
		return 200, `{"trusts":[{"id":"a","impersonation":0}]}`
	})
	query := map[string]json.RawMessage{"trustor_user_id": json.RawMessage(`"alice"`)}
	headers := map[string]string{"X-Owned": "old"}
	limit := 2
	option := trusts.WithRecordListOpts(trusts.RecordListOpts{Query: query, Headers: headers, Limit: &limit})
	query["trustor_user_id"] = json.RawMessage(`"mutated"`)
	headers["X-Owned"] = "mutated"
	limit = 7
	seq := api.ListRecords(context.Background(), option)
	if len(calls) != 0 {
		t.Fatal("list is eager")
	}
	for range 2 {
		for value, err := range seq {
			if err != nil {
				t.Fatal(err)
			}
			trustRecordField(t, value, "id", `"a"`)
			value.Wire.Body["id"] = json.RawMessage(`"mutated"`)
			value.Resource.Body["id"] = json.RawMessage(`"mutated"`)
		}
	}
	if len(calls) != 4 || calls[0].query != "limit=2&trustor_user_id=alice" || calls[1].query != "limit=2&marker=a&trustor_user_id=alice" || calls[2] != calls[0] || calls[3] != calls[1] {
		t.Fatal(calls)
	}
	t.Run("later rejection retained", func(t *testing.T) {
		var calls []nativeTrustCall
		api, _ := nativeTrustAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.Query().Get("marker") == "next" {
				return 500, "later proof"
			}
			return 200, `{"trusts":[{"id":"a"}],"next":"?marker=next"}`
		})
		ids, err := trustRecordIDs(t, api)
		var native gophercloud.ErrUnexpectedResponseCode
		if !reflect.DeepEqual(ids, []string{"a"}) || !errors.As(err, &native) || native.Actual != 500 || string(native.Body) != "later proof" || len(calls) != 2 {
			t.Fatal(ids, err, calls)
		}
		calls = nil
		for _, err := range api.ListRecords(context.Background()) {
			if err != nil {
				t.Fatal(err)
			}
			break
		}
		if len(calls) != 1 {
			t.Fatal("consumer stop fetched a later page", calls)
		}
	})
	t.Run("advertised next and singleton", func(t *testing.T) {
		var calls []nativeTrustCall
		api, _ := nativeTrustAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.Query().Get("marker") == "next" {
				return 200, `{"trusts":{"id":"b"}}`
			}
			return 200, `{"trusts":[{"id":"a"}],"links":[{"rel":"next","href":"?marker=next"}]}`
		})
		ids, err := trustRecordIDs(t, api)
		if err != nil || !reflect.DeepEqual(ids, []string{"a", "b"}) || len(calls) != 2 {
			t.Fatal(ids, err, calls)
		}
	})
}

func TestPythonTrustFindGetFirstCleanFallbackFullSearchAndMissing(t *testing.T) {
	for _, status := range []int{400, 403, 404} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls []nativeTrustCall
			api, _ := nativeTrustAPI(t, &calls, func(req *http.Request) (int, string) {
				if strings.HasSuffix(req.URL.Path, "/ci") {
					return status, "member missing"
				}
				if req.URL.Query().Get("marker") == "next" {
					return 200, `{"trusts":[{"id":"found","name":"ci"}]}`
				}
				return 200, `{"trusts":[{"id":"skip","name":"other"}],"next":"?marker=next"}`
			})
			value, err := api.FindRecord(context.Background(), "ci")
			if err != nil || len(calls) != 3 || calls[0].query != "" || calls[1].query != "" || calls[2].query != "marker=next" {
				t.Fatal(value, err, calls)
			}
			trustRecordField(t, value, "id", `"found"`)
		})
	}
	for _, item := range []struct {
		name, list string
		strict     bool
		expected   string
	}{
		{"default missing", `[]`, false, "missing"}, {"strict missing", `[]`, true, "strict"},
		{"duplicate", `[{"id":"a","name":"ci"},{"id":"b","name":"ci"}]`, false, "duplicate"},
		{"ID match", `[{"id":"ci","name":"other"}]`, false, "ci"},
	} {
		t.Run(item.name, func(t *testing.T) {
			var calls []nativeTrustCall
			api, _ := nativeTrustAPI(t, &calls, func(req *http.Request) (int, string) {
				if strings.HasSuffix(req.URL.Path, "/ci") {
					return 404, "member missing"
				}
				return 200, `{"trusts":` + item.list + `}`
			})
			value, err := api.FindRecord(context.Background(), "ci", trusts.WithRecordFindIgnoreMissing(!item.strict))
			switch item.expected {
			case "missing":
				if value != nil || err != nil {
					t.Fatal(value, err)
				}
			case "strict":
				var target *resource.NotFoundError
				if value != nil || !errors.As(err, &target) {
					t.Fatal(value, err)
				}
			case "duplicate":
				var target *resource.AmbiguousError
				if value != nil || !errors.As(err, &target) {
					t.Fatal(value, err)
				}
			default:
				if err != nil {
					t.Fatal(err)
				}
				trustRecordField(t, value, "id", `"ci"`)
			}
		})
	}
	t.Run("direct malformed body retains seed", func(t *testing.T) {
		var calls []nativeTrustCall
		api, _ := nativeTrustAPI(t, &calls, func(*http.Request) (int, string) { return 200, "{broken" })
		value, err := api.FindRecord(context.Background(), "selected")
		if err != nil || len(calls) != 1 {
			t.Fatal(value, err, calls)
		}
		trustRecordField(t, value, "id", `"selected"`)
	})
	t.Run("later error overrides early match", func(t *testing.T) {
		var calls []nativeTrustCall
		api, _ := nativeTrustAPI(t, &calls, func(req *http.Request) (int, string) {
			if strings.HasSuffix(req.URL.Path, "/ci") {
				return 404, "missing"
			}
			if req.URL.Query().Get("marker") == "next" {
				return 500, "later"
			}
			return 200, `{"trusts":[{"id":"found","name":"ci"}],"next":"?marker=next"}`
		})
		value, err := api.FindRecord(context.Background(), "ci")
		var native gophercloud.ErrUnexpectedResponseCode
		if value != nil || !errors.As(err, &native) || native.Actual != 500 || len(calls) != 3 {
			t.Fatal(value, err, calls)
		}
	})
}

func TestPythonTrustDeleteMissingAndSourceStatusPolicies(t *testing.T) {
	for _, code := range []int{200, 201, 202, 204, 299, 301, 399, 400, 403, 404, 409, 500, 600} {
		for _, operation := range []string{"get", "create", "delete", "list"} {
			t.Run(fmt.Sprintf("%s/%d", operation, code), func(t *testing.T) {
				var calls []nativeTrustCall
				api, _ := nativeTrustAPI(t, &calls, func(*http.Request) (int, string) { return code, `{"trusts":[],"trust":{}}` })
				var err error
				switch operation {
				case "get":
					_, err = api.GetRecord(context.Background(), "selected")
				case "create":
					_, err = api.CreateRecord(context.Background())
				case "delete":
					err = api.DeleteRecord(context.Background(), "selected")
				case "list":
					_, err = trustRecordIDs(t, api)
				}
				accepted := code < 400 || operation == "delete" && code == 404
				if (err == nil) != accepted || len(calls) != 1 {
					t.Fatal(err, calls)
				}
				if !accepted {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != code {
						t.Fatal(err, native)
					}
				}
				if operation == "delete" && code == 404 {
					err = api.DeleteRecord(context.Background(), "selected", trusts.WithRecordDeleteIgnoreMissing(false))
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != 404 {
						t.Fatal(err, native)
					}
				}
			})
		}
	}
}

func TestPythonTrustOptionsOwnershipPreflightAndJoinedCallbacks(t *testing.T) {
	var calls []nativeTrustCall
	api, _ := nativeTrustAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.Header.Get("X-Owned") != "old" {
			t.Error(req.Header)
		}
		return 200, `{}`
	})
	var retained *trusts.RecordCreateOpts
	value, err := api.CreateRecord(context.Background(), func(c *trusts.RecordCreateOpts) error {
		retained = c
		c.Fields = map[string]json.RawMessage{"roles": json.RawMessage(`[]`)}
		c.Headers = map[string]string{"X-Owned": "old"}
		return nil
	}, func(*trusts.RecordCreateOpts) error {
		retained.Fields["roles"] = json.RawMessage(`true`)
		retained.Headers["X-Owned"] = "changed"
		return nil
	})
	if err != nil || len(calls) != 1 || calls[0].body != `{"trust":{"roles":[]}}` {
		t.Fatal(value, err, calls)
	}
	for _, item := range []struct {
		name string
		call func() error
	}{
		{"nil create", func() error { _, err := api.CreateRecord(context.Background(), nil); return err }},
		{"nil get", func() error { _, err := api.GetRecord(context.Background(), "a", nil); return err }},
		{"nil delete", func() error { return api.DeleteRecord(context.Background(), "a", nil) }},
		{"nil find", func() error { _, err := api.FindRecord(context.Background(), "a", nil); return err }},
		{"nil list", func() error { _, err := trustRecordIDs(t, api, nil); return err }},
		{"raw JSON", func() error {
			_, err := api.CreateRecord(context.Background(), trusts.WithRecordCreateOpts(trusts.RecordCreateOpts{Fields: map[string]json.RawMessage{"roles": json.RawMessage(`{bad`)}}))
			return err
		}},
		{"alias collision", func() error {
			_, err := api.CreateRecord(context.Background(), trusts.WithRecordCreateOpts(trusts.RecordCreateOpts{Fields: map[string]json.RawMessage{"impersonation": json.RawMessage(`true`), "is_impersonation": json.RawMessage(`false`)}}))
			return err
		}},
		{"negative limit", func() error { _, err := trustRecordIDs(t, api, trusts.WithRecordListLimit(-1)); return err }},
		{"protected header", func() error {
			_, err := api.GetRecord(context.Background(), "a", trusts.WithRecordReadHeaders(map[string]string{"X-Auth-Token": "wrong"}))
			return err
		}},
		{"dot ID", func() error { _, err := api.GetRecord(context.Background(), ".."); return err }},
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
		defer cancel(nil)
		cause, callback := errors.New("caller"), errors.New("callback")
		before := len(calls)
		_, err := api.CreateRecord(ctx, func(*trusts.RecordCreateOpts) error { cancel(cause); return callback })
		if !errors.Is(err, cause) || !errors.Is(err, callback) || !errors.Is(err, context.Canceled) || len(calls) != before {
			t.Fatal(err, calls)
		}
	})
	t.Run("callback source mutation and error", func(t *testing.T) {
		endpoint := api.RawClient().Endpoint
		defer func() { api.RawClient().Endpoint = endpoint }()
		cause := errors.New("callback")
		before := len(calls)
		_, err := api.CreateRecord(context.Background(), func(*trusts.RecordCreateOpts) error { api.RawClient().Endpoint += "changed/"; return cause })
		if !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || len(calls) != before {
			t.Fatal(err, calls)
		}
	})
}

type trustRecordFaultBody struct {
	data            []byte
	read, close     error
	closes          int
	onRead, onClose func()
}

func (body *trustRecordFaultBody) Read(data []byte) (int, error) {
	if len(body.data) > 0 {
		n := copy(data, body.data)
		body.data = body.data[n:]
		if body.onRead != nil {
			body.onRead()
		}
		return n, nil
	}
	if body.onRead != nil {
		body.onRead()
	}
	return 0, body.read
}
func (body *trustRecordFaultBody) Close() error {
	body.closes++
	if body.onClose != nil {
		body.onClose()
	}
	return body.close
}

func TestPythonTrustPhysicalAcceptedAndRejectedFailures(t *testing.T) {
	for _, operation := range []string{"get", "create", "delete", "find", "list"} {
		for _, status := range []int{200, 404, 500} {
			for _, kind := range []string{"read", "close", "read and close", "cancel after read", "cancel after close", "source restored by Close"} {
				t.Run(fmt.Sprintf("%s/%d/%s", operation, status, kind), func(t *testing.T) {
					var serverCalls []nativeTrustCall
					api, cloud := nativeTrustAPI(t, &serverCalls, func(*http.Request) (int, string) { return 200, `{}` })
					ctx, cancel := context.WithCancelCause(context.Background())
					defer cancel(nil)
					readCause, closeCause, cancelCause := errors.New("physical Read"), errors.New("physical Close"), errors.New("caller")
					body := &trustRecordFaultBody{data: []byte("physical proof"), read: io.EOF}
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
					endpoint := api.RawClient().Endpoint
					if kind == "source restored by Close" {
						body.onRead = func() { api.RawClient().Endpoint = endpoint + "changed/" }
						body.onClose = func() { api.RawClient().Endpoint = endpoint }
					}
					attempts := 0
					cloud.Provider.HTTPClient.Transport = nativeTrustTransport(func(*http.Request) (*http.Response, error) {
						attempts++
						return &http.Response{StatusCode: status, Header: http.Header{"X-Request-Id": {"physical-proof"}}, Body: body}, nil
					})
					var err error
					switch operation {
					case "get":
						_, err = api.GetRecord(ctx, "selected")
					case "create":
						_, err = api.CreateRecord(ctx)
					case "delete":
						err = api.DeleteRecord(ctx, "selected")
					case "find":
						value, foundErr := api.FindRecord(ctx, "ci")
						err = foundErr
						if value != nil && status >= 400 {
							t.Fatal("bad member rejection fell back", value, err)
						}
					case "list":
						for _, listErr := range api.ListRecords(ctx) {
							if listErr != nil {
								err = listErr
								break
							}
						}
					}
					if err == nil || attempts != 1 || body.closes != 1 {
						t.Fatal(err, attempts, body.closes)
					}
					if status < 400 {
						var proof *resource.ResponseError
						if !errors.As(err, &proof) || proof.StatusCode != status || string(proof.Body) != "physical proof" || proof.Header.Get("X-Request-Id") != "physical-proof" {
							t.Fatal(err, proof)
						}
					} else {
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(err, &native) || native.Actual != status || string(native.Body) != "physical proof" || native.ResponseHeader.Get("X-Request-Id") != "physical-proof" {
							t.Fatal(err, native)
						}
					}
					if (kind == "read" || kind == "read and close") && !errors.Is(err, readCause) {
						t.Fatal(err)
					}
					if (kind == "close" || kind == "read and close") && !errors.Is(err, closeCause) {
						t.Fatal(err)
					}
					if strings.HasPrefix(kind, "cancel") && (!errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled)) {
						t.Fatal(err)
					}
					if kind == "source restored by Close" && (!errors.Is(err, resource.ErrInvalidOption) || api.RawClient().Endpoint != endpoint) {
						t.Fatal(err)
					}
				})
			}
		}
	}
}

func TestPythonTrustNativeRetryAndFixedRequestOwnership(t *testing.T) {
	for _, operation := range []string{"delete", "find"} {
		t.Run("clean stop "+operation, func(t *testing.T) {
			var calls []nativeTrustCall
			api, _ := nativeTrustAPI(t, &calls, func(req *http.Request) (int, string) {
				if operation == "find" && req.URL.Path == "/keystone/v3/OS-TRUST/trusts" {
					return 200, `{"trusts":[{"id":"found","name":"ci"}]}`
				}
				return 404, "clean"
			})
			retries := 0
			api.RawClient().RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, _ uint) error {
				retries++
				return original
			}
			if operation == "delete" {
				if err := api.DeleteRecord(context.Background(), "selected"); err != nil || len(calls) != 1 {
					t.Fatal(err, calls)
				}
			} else {
				value, err := api.FindRecord(context.Background(), "ci")
				if err != nil || len(calls) != 2 {
					t.Fatal(value, err, calls)
				}
				trustRecordField(t, value, "id", `"found"`)
			}
			if retries != 1 {
				t.Fatal(retries)
			}
		})
	}
	for _, kind := range []string{"success", "callback error", "body changed", "source changed", "cancel"} {
		t.Run("create retry "+kind, func(t *testing.T) {
			var calls []nativeTrustCall
			api, _ := nativeTrustAPI(t, &calls, func(*http.Request) (int, string) {
				if len(calls) == 1 {
					return 503, "retry proof"
				}
				return 201, `{"trust":{"id":"created"}}`
			})
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("retry cause")
			api.RawClient().RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				switch kind {
				case "callback error":
					return cause
				case "body changed":
					opts.JSONBody = json.RawMessage(`{"trust":{}}`)
				case "source changed":
					api.RawClient().Endpoint += "changed/"
				case "cancel":
					cancel(cause)
				}
				return nil
			}
			value, err := api.CreateRecord(ctx, trusts.WithRecordCreateField("roles", []any{}))
			if kind == "success" {
				if err != nil || len(calls) != 2 || calls[0] != calls[1] {
					t.Fatal(value, err, calls)
				}
				trustRecordField(t, value, "id", `"created"`)
				return
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if value != nil || !errors.As(err, &native) || native.Actual != 503 || len(calls) != 1 {
				t.Fatal(value, err, calls, native)
			}
			if kind == "callback error" && !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if (kind == "body changed" || kind == "source changed") && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
			if kind == "cancel" && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
				t.Fatal(err)
			}
		})
	}
}

func TestPythonTrustEscapedMemberAndConfiguredFutureVersion(t *testing.T) {
	var calls []nativeTrustCall
	api, _ := nativeTrustAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.EscapedPath() != "/keystone/v3/OS-TRUST/trusts/folder%2Fname%25" {
			t.Error(req.URL.EscapedPath())
		}
		return 200, `{}`
	})
	api.RawClient().Microversion = "99.42"
	value, err := api.GetRecord(context.Background(), "folder/name%")
	if err != nil || len(calls) != 1 {
		t.Fatal(value, err, calls)
	}
	trustRecordField(t, value, "id", `"folder/name%"`)
}
