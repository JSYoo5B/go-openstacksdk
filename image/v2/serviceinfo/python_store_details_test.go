package serviceinfo

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestPythonStoreRecordsDetailsSelectsFixedRouteAndSchema(t *testing.T) {
	calls := 0
	const body = `{"stores":[{"id":"fast","name":"fast","default":"false","description":null,"properties":{"x":1},"type":"rbd","read-only":true,"weight":2}]}`
	client := infoClient(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path != "/reverse/glance/v2/info/stores/detail" || r.URL.RawQuery != "limit=0" || r.Header.Get("Accept") != "application/json" {
			t.Fatal(r.URL, r.Header)
		}
		return infoRecordJSON(r, 203, body), nil
	})
	api := New(client)
	rows, err := api.AllStoreRecords(context.Background(), WithStoreRecordListDetails(true), WithStoreRecordListLimit(0), WithStoreRecordListFilter("is_default", true))
	if err != nil || len(rows) != 1 || calls != 1 {
		t.Fatal(rows, err, calls)
	}
	if string(rows[0].Resource.Body["is_default"]) != "true" {
		t.Fatalf("string(rows[0].Resource.Body[\"is_default\"]) = %v, want %v", string(rows[0].Resource.Body["is_default"]), "true")
	}
	for _, field := range []string{"type", "read-only", "weight"} {
		if rows[0].Resource.Body[field] != nil || rows[0].Wire.Body[field] == nil {
			t.Fatal(field, rows[0])
		}
	}
	if string(rows[0].Envelope) != body {
		t.Fatalf("string(rows[0].Envelope) = %v, want %v", string(rows[0].Envelope), body)
	}
}
func TestPythonStoreRecordsDetailsBasicSelectionAndReusableOptions(t *testing.T) {
	paths := []string{}
	client := infoClient(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		return infoRecordJSON(r, 200, `{"stores":[{}]}`), nil
	})
	api := New(client)
	option := WithStoreRecordListOpts(StoreRecordListOpts{Details: true})
	for i := 0; i < 2; i++ {
		rows, err := api.AllStoreRecords(context.Background(), option)
		if err != nil || len(rows) != 1 {
			t.Fatal(rows, err)
		}
	}
	if _, err := api.AllStoreRecords(context.Background(), option, WithStoreRecordListDetails(false)); err != nil {
		t.Fatal(err)
	}
	if paths[0] != "/reverse/glance/v2/info/stores/detail" || paths[1] != paths[0] || paths[2] != "/reverse/glance/v2/info/stores" {
		t.Fatal(paths)
	}
}
func TestPythonStoreRecordsDetailsPagingPartialAndCaps(t *testing.T) {
	calls := 0
	const bad = `{"stores":[{"id":"second","default":true},null]}`
	client := infoClient(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Query().Get("marker") == "next" {
			return infoRecordJSON(r, 202, bad), nil
		}
		return infoRecordJSON(r, 200, `{"stores":[{"id":"first","default":false}],"next":"/v2/info/stores/detail?marker=next"}`), nil
	})
	api := New(client)
	rows, err := api.AllStoreRecords(context.Background(), WithStoreRecordListDetails(true))
	var proof *resource.ResponseError
	if len(rows) != 2 || !errors.As(err, &proof) || string(proof.Body) != bad || proof.StatusCode != 202 || calls != 2 {
		t.Fatal(rows, err, proof, calls)
	}
	rows, err = api.AllStoreRecords(context.Background(), WithStoreRecordListDetails(true), WithStoreRecordListMaxItems(1), WithStoreRecordListFilter("is_default", true))
	if err != nil || len(rows) != 0 || calls != 3 {
		t.Fatal(rows, err, calls)
	}
}
func TestPythonStoreRecordsDetailsHasNoLocalRoleGateAndOwnsFailures(t *testing.T) {
	calls := 0
	client := infoClient(func(r *http.Request) (*http.Response, error) {
		calls++
		return infoRecordJSON(r, 403, `{"forbidden":"policy"}`), nil
	})
	if _, err := New(client).AllStoreRecords(context.Background(), WithStoreRecordListDetails(true)); err == nil {
		t.Fatal("server denial ignored")
	}
	if calls != 1 {
		t.Fatal(calls)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("stop")
	cancel(cause)
	if _, err := New(client).AllStoreRecords(ctx, WithStoreRecordListDetails(true)); !errors.Is(err, cause) {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal(calls)
	}
}
