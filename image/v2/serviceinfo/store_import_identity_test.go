package serviceinfo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func TestStoreRecordImportIdentityRetainsPrivateIDAndReturnsOwnedCopies(t *testing.T) {
	for _, raw := range []string{`"fast /한글"`, `""`, `null`, `false`, `900719925474099312345`, `[1,null]`, `{"id":"nested"}`} {
		t.Run(raw, func(t *testing.T) {
			client := infoClient(func(req *http.Request) (*http.Response, error) {
				return infoRecordJSON(req, 203, `{"stores":[{"id":`+raw+`,"name":"public"}]}`), nil
			})
			var record *StoreRecord
			for row, err := range New(client).ListStoreRecords(context.Background()) {
				th.AssertNoErr(t, err)
				record = row
				break
			}
			if record == nil {
				t.Fatal("missing SDK store")
			}
			record.Resource.Body["id"] = json.RawMessage(`"view decoy"`)
			record.Wire.Body["id"] = json.RawMessage(`"wire decoy"`)
			record.Envelope = json.RawMessage(`{"id":"envelope decoy"}`)
			record.Header.Set("Location", "https://foreign.test/store")
			first, err := record.ImportIdentity()
			th.AssertNoErr(t, err)
			th.AssertEquals(t, raw, string(first))
			first[0] = '!'
			second, err := record.ImportIdentity()
			th.AssertNoErr(t, err)
			th.AssertEquals(t, raw, string(second))
			copied := *record
			copied.Resource = &resource.RawResource{}
			third, err := copied.ImportIdentity()
			th.AssertNoErr(t, err)
			th.AssertEquals(t, raw, string(third))
		})
	}
}

func TestStoreRecordImportIdentityMissingAndForgedInputs(t *testing.T) {
	t.Run("missing ID stays null without name fallback", func(t *testing.T) {
		client := infoClient(func(req *http.Request) (*http.Response, error) {
			return infoRecordJSON(req, 200, `{"stores":[{"name":"not an ID"}]}`), nil
		})
		var record *StoreRecord
		for row, err := range New(client).ListStoreRecords(context.Background()) {
			th.AssertNoErr(t, err)
			record = row
			break
		}
		got, err := record.ImportIdentity()
		th.AssertNoErr(t, err)
		th.AssertEquals(t, "null", string(got))
	})
	for _, test := range []struct {
		name   string
		record *StoreRecord
	}{
		{"nil", nil},
		{"zero", &StoreRecord{}},
		{"public raw view", &StoreRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"forged"`)}}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.record.ImportIdentity()
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(got, err)
			}
		})
	}
}

func TestStoreImportIdentityConstructorValuesAndOwnership(t *testing.T) {
	for _, test := range []struct{ name, attrs, want string }{
		{"empty object", "{}", "null"},
		{"name is not alternate ID", `{"name":"only name"}`, "null"},
		{"null", `{"id":null}`, "null"},
		{"empty string", `{"id":""}`, `""`},
		{"string", `{"id":"fast /한글"}`, `"fast /한글"`},
		{"false", `{"id":false}`, "false"},
		{"exact number", `{"id":900719925474099312345}`, "900719925474099312345"},
		{"huge exponent", `{"id":1e400}`, "1e400"},
		{"array", `{"id":[1,null]}`, `[1,null]`},
		{"object", `{"id":{"store":[true]}}`, `{"store":[true]}`},
		{"duplicate ID last wins", `{"id":"first","id":"second"}`, `"second"`},
		{"unrelated descriptor is lazy", `{"id":"fast","is_default":{"future":true},"properties":false,"weight":1e400}`, `"fast"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := json.RawMessage(test.attrs)
			got, err := StoreImportIdentity(input)
			th.AssertNoErr(t, err)
			th.AssertEquals(t, test.want, string(got))
			input[0] = '!'
			th.AssertEquals(t, test.want, string(got))
			got[0] = '!'
			again, err := StoreImportIdentity(json.RawMessage(test.attrs))
			th.AssertNoErr(t, err)
			th.AssertEquals(t, test.want, string(again))
		})
	}
}

func TestStoreImportIdentityRejectsInvalidConstructorBeforeHTTP(t *testing.T) {
	for _, raw := range []string{"", `{`, "{} {}", `null`, `[]`, `false`, `{"connection":null}`, `{"_synchronized":false}`, `{"microversion":"2.10"}`, "{\"id\":\"bad\xff\"}"} {
		t.Run(raw, func(t *testing.T) {
			got, err := StoreImportIdentity(json.RawMessage(raw))
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(string(got), err)
			}
		})
	}
}
