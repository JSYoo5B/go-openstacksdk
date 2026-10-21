package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/baremetal/v1/runbooks"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const r2IronicBRunbookBody = `{"uuid":"11111111-1111-1111-1111-111111111111","name":"CUSTOM_RB","description":"","public":false,"owner":null,"extra":{},"traits":["CUSTOM_A"],"steps":[{"interface":"deploy","step":"x","args":{"n":10000000000000000001},"order":0,"future":true}],"created_at":"2026-01-02T03:04:05+00:00","updated_at":null,"links":[{"href":"h","rel":"self"}],"newer":1}`

func r2IronicBRunbooks(t *testing.T, status int, body string) (*runbooks.API, *r2IronicBFake) {
	t.Helper()
	cloud := testcloud.New(t)
	fake := r2IronicBFixed(t, cloud, status, body)
	return runbooks.New(r2IronicBClient(cloud)), fake
}

func TestR2IronicBRunbookCreateSendsExactBodyAndKeepsEvidence(t *testing.T) {
	api, fake := r2IronicBRunbooks(t, 201, r2IronicBRunbookBody)
	zero := 0
	got, err := api.Create(context.Background(), runbooks.CreateOpts{
		Name:        "CUSTOM_RB",
		Description: r2IronicBPtr(""),
		Public:      r2IronicBPtr(false),
		Extra:       map[string]json.RawMessage{},
		Steps:       []runbooks.Step{{Interface: "deploy", Step: "x", Args: map[string]json.RawMessage{"n": json.RawMessage("10000000000000000001")}, Order: &zero, Extra: map[string]json.RawMessage{"future": json.RawMessage("true")}}},
	}, runbooks.WithCreateField("vendor", map[string]int{"a": 0}), runbooks.WithCreateHeader("X-Test", "1"))
	if err != nil {
		t.Fatal(err)
	}
	call := fake.only(t)
	if call.Method != "POST" || call.Path != "/v1/runbooks" || call.RawQuery != "" {
		t.Fatalf("call=%+v", call)
	}
	if call.Header.Get("X-OpenStack-Ironic-API-Version") != r2IronicBVersion || call.Header.Get("X-Test") != "1" {
		t.Fatalf("headers=%v", call.Header)
	}
	want := `{"description":"","extra":{},"name":"CUSTOM_RB","public":false,"steps":[{"args":{"n":10000000000000000001},"future":true,"interface":"deploy","order":0,"step":"x"}],"vendor":{"a":0}}`
	if call.Body != want {
		t.Fatalf("body=%s", call.Body)
	}
	if got.UUID != "11111111-1111-1111-1111-111111111111" || got.StatusCode != 201 || got.Description == nil || *got.Description != "" || got.Public == nil || *got.Public || got.Owner != nil {
		t.Fatalf("got=%+v", got)
	}
	if string(got.Body["owner"]) != "null" || string(got.Body["newer"]) != "1" || string(got.Steps[0].Args["n"]) != "10000000000000000001" || string(got.Steps[0].Extra["future"]) != "true" || *got.Steps[0].Order != 0 {
		t.Fatalf("body=%v steps=%+v", got.Body, got.Steps)
	}
	if got.Traits[0] != "CUSTOM_A" || got.CreatedAt == nil || got.UpdatedAt != nil || got.Header == nil {
		t.Fatalf("got=%+v", got)
	}
}

func TestR2IronicBRunbookCreateOmitsUnsetAndRejectsCollisionsBeforeHTTP(t *testing.T) {
	api, fake := r2IronicBRunbooks(t, 201, r2IronicBRunbookBody)
	if _, err := api.Create(context.Background(), runbooks.CreateOpts{Name: "n"}); err != nil {
		t.Fatal(err)
	}
	if body := fake.only(t).Body; body != `{"name":"n"}` {
		t.Fatalf("body=%s", body)
	}
	api, fake = r2IronicBRunbooks(t, 201, r2IronicBRunbookBody)
	for _, call := range []func() error{
		func() error {
			_, err := api.Create(context.Background(), runbooks.CreateOpts{Name: "n"}, runbooks.WithCreateField("name", "x"))
			return err
		},
		func() error {
			_, err := api.Create(context.Background(), runbooks.CreateOpts{Name: "n", Steps: []runbooks.Step{{Extra: map[string]json.RawMessage{"step": json.RawMessage(`"y"`)}}}})
			return err
		},
		func() error {
			_, err := api.Create(r2IronicBCanceled(), runbooks.CreateOpts{Name: "n"})
			return err
		},
	} {
		if err := call(); err == nil {
			t.Fatal("expected error")
		}
	}
	r2IronicBNoHTTP(t, fake)
}

func TestR2IronicBRunbookCreateAcceptsOnly201AndKeepsResponseOnBadBody(t *testing.T) {
	for _, status := range []int{200, 202} {
		api, _ := r2IronicBRunbooks(t, status, r2IronicBRunbookBody)
		_, err := api.Create(context.Background(), runbooks.CreateOpts{Name: "n"})
		var code gophercloud.ErrUnexpectedResponseCode
		if !errors.As(err, &code) || code.Actual != status {
			t.Fatalf("status %d: %v", status, err)
		}
	}
	api, fake := r2IronicBRunbooks(t, 201, `[]`)
	_, err := api.Create(context.Background(), runbooks.CreateOpts{Name: "n"})
	var response *resource.ResponseError
	if !errors.As(err, &response) || response.StatusCode != 201 || string(response.Body) != `[]` {
		t.Fatalf("err=%v", err)
	}
	if n := len(fake.all()); n != 1 {
		t.Fatalf("a decode failure must not resend: %d", n)
	}
}

func TestR2IronicBRunbookGetFieldsAndErrors(t *testing.T) {
	api, fake := r2IronicBRunbooks(t, 200, `{"uuid":"u","name":"n"}`)
	got, err := api.Get(context.Background(), "a/b c", runbooks.WithGetFields("uuid", "name"))
	if err != nil || got.UUID != "u" || got.Description != nil || got.Steps != nil {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	call := fake.only(t)
	if call.Method != "GET" || call.Path != "/v1/runbooks/a%2Fb%20c" || call.RawQuery != "fields=uuid%2Cname" || call.Body != "" {
		t.Fatalf("call=%+v", call)
	}
	api, fake = r2IronicBRunbooks(t, 404, `{"error_message":"nope"}`)
	_, err = api.Get(context.Background(), "u")
	if !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatalf("err=%v", err)
	}
	for _, id := range []string{"", ".", "..", " u"} {
		if _, err := api.Get(context.Background(), id); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("id %q: %v", id, err)
		}
	}
	if _, err := api.Get(context.Background(), "u", runbooks.WithGetFields("a,b")); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	if _, err := api.Get(r2IronicBCanceled(), "u"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if n := len(fake.all()); n != 1 {
		t.Fatalf("only the first Get may reach HTTP: %d", n)
	}
}

func TestR2IronicBRunbookListFollowsNextLinkQueryOnly(t *testing.T) {
	cloud := testcloud.New(t)
	fake := r2IronicBServe(t, cloud, func(c r2IronicBCall) (int, string) {
		switch c.RawQuery {
		case "detail=true&limit=1&sort_key=name":
			return 200, `{"runbooks":[{"uuid":"a","name":"A"}],"next":"http://other.invalid/v1/runbooks?limit=1&marker=a&detail=True&sort_key=name"}`
		case "detail=True&limit=1&marker=a&sort_key=name":
			return 200, `{"runbooks":[{"uuid":"b","name":"B"}]}`
		}
		return 200, `{"runbooks":[]}`
	})
	api := runbooks.New(r2IronicBClient(cloud))
	var ids []string
	for value, err := range api.List(context.Background(), runbooks.WithListDetail(true), runbooks.WithListLimit(1), runbooks.WithListQuery("sort_key", "name")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.UUID)
	}
	calls := fake.all()
	// a next link stops at its page; limit-driven continuation then asks once more.
	if strings.Join(ids, ",") != "a,b" || len(calls) != 3 {
		t.Fatalf("ids=%v calls=%+v", ids, calls)
	}
	for _, c := range calls {
		if c.Method != "GET" || c.Path != "/v1/runbooks" || c.Body != "" {
			t.Fatalf("call=%+v", c)
		}
	}
	if calls[2].RawQuery != "detail=True&limit=1&marker=b&sort_key=name" {
		t.Fatalf("last=%+v", calls[2])
	}
}

func TestR2IronicBRunbookListControlsAndFailures(t *testing.T) {
	cloud := testcloud.New(t)
	fake := r2IronicBFixed(t, cloud, 200, `{"runbooks":[{"uuid":"a"},{"uuid":"b"},{"uuid":"c"}]}`)
	api := runbooks.New(r2IronicBClient(cloud))
	count := 0
	for _, err := range api.List(context.Background(), runbooks.WithListMaxItems(2), runbooks.WithListSinglePage(), runbooks.WithListFields("uuid", "name")) {
		if err != nil {
			t.Fatal(err)
		}
		count++
	}
	call := fake.only(t)
	if count != 2 || call.RawQuery != "fields=uuid%2Cname&limit=2" {
		t.Fatalf("count=%d call=%+v", count, call)
	}
	for _, opt := range []runbooks.ListOption{runbooks.WithListQuery("detail", "x"), runbooks.WithListQuery("marker", "x"), runbooks.WithListLimit(0), runbooks.WithListMarker(""), runbooks.WithListMaxItems(-1), runbooks.WithListFields("")} {
		var gotErr error
		for _, err := range api.List(context.Background(), opt) {
			gotErr = err
		}
		if !errors.Is(gotErr, resource.ErrInvalidOption) {
			t.Fatalf("err=%v", gotErr)
		}
	}
	if len(fake.all()) != 1 {
		t.Fatalf("invalid options reached HTTP: %+v", fake.all())
	}
}

func TestR2IronicBRunbookListDetectsMarkerLoopAndBadPages(t *testing.T) {
	cloud := testcloud.New(t)
	r2IronicBFixed(t, cloud, 200, `{"runbooks":[{"uuid":"a"}]}`)
	api := runbooks.New(r2IronicBClient(cloud))
	var err error
	n := 0
	for _, e := range api.List(context.Background(), runbooks.WithListLimit(5)) {
		err, n = e, n+1
	}
	if !errors.Is(err, resource.ErrPaginationCycle) {
		t.Fatalf("err=%v n=%d", err, n)
	}
	for _, body := range []string{`[]`, `{"other":[]}`, `{"runbooks":{}}`, `{"runbooks":[1]}`, `{"runbooks":[],"next":5}`} {
		cloud := testcloud.New(t)
		r2IronicBFixed(t, cloud, 200, body)
		var got error
		for _, e := range runbooks.New(r2IronicBClient(cloud)).List(context.Background()) {
			got = e
		}
		var response *resource.ResponseError
		if !errors.As(got, &response) || response.StatusCode != 200 {
			t.Fatalf("body %s: %v", body, got)
		}
	}
	cloud = testcloud.New(t)
	r2IronicBFixed(t, cloud, 200, `{"runbooks":[{"uuid":"a"}],"next":"http://x/v1/chassis?marker=a"}`)
	var got error
	for _, e := range runbooks.New(r2IronicBClient(cloud)).List(context.Background()) {
		got = e
	}
	var response *resource.ResponseError
	if !errors.As(got, &response) {
		t.Fatalf("foreign next link must fail: %v", got)
	}
}

func TestR2IronicBRunbookListCancellationAndEarlyStop(t *testing.T) {
	cloud := testcloud.New(t)
	fake := r2IronicBFixed(t, cloud, 200, `{"runbooks":[{"uuid":"a"},{"uuid":"b"}]}`)
	api := runbooks.New(r2IronicBClient(cloud))
	for _, err := range api.List(r2IronicBCanceled()) {
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	}
	r2IronicBNoHTTP(t, fake)
	for _, err := range api.List(context.Background()) {
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if len(fake.all()) != 1 {
		t.Fatalf("calls=%+v", fake.all())
	}
}

func TestR2IronicBRunbookUpdateSendsSortedAddOperations(t *testing.T) {
	api, fake := r2IronicBRunbooks(t, 200, r2IronicBRunbookBody)
	got, err := api.Update(context.Background(), "u 1", runbooks.UpdateOpts{
		Public: r2IronicBPtr(false), Description: r2IronicBPtr(""), Name: r2IronicBPtr("N"), Extra: map[string]json.RawMessage{},
	}, runbooks.WithUpdateField("zeta", 1), runbooks.WithUpdateHeader("X-Test", "2"))
	if err != nil || got.StatusCode != 200 {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	call := fake.only(t)
	want := `[{"op":"add","path":"/description","value":""},{"op":"add","path":"/extra","value":{}},{"op":"add","path":"/name","value":"N"},{"op":"add","path":"/public","value":false},{"op":"add","path":"/zeta","value":1}]`
	if call.Method != "PATCH" || call.Path != "/v1/runbooks/u%201" || call.Body != want || call.Header.Get("X-Test") != "2" {
		t.Fatalf("call=%+v", call)
	}
}

func TestR2IronicBRunbookUpdateRejectsEmptyAndCollisionsBeforeHTTP(t *testing.T) {
	api, fake := r2IronicBRunbooks(t, 200, r2IronicBRunbookBody)
	for _, call := range []func() error{
		func() error { _, err := api.Update(context.Background(), "u", runbooks.UpdateOpts{}); return err },
		func() error {
			_, err := api.Update(context.Background(), "u", runbooks.UpdateOpts{}, runbooks.WithUpdateField("name", "x"))
			return err
		},
		func() error {
			_, err := api.Update(context.Background(), "", runbooks.UpdateOpts{Name: r2IronicBPtr("x")})
			return err
		},
		func() error {
			_, err := api.Update(r2IronicBCanceled(), "u", runbooks.UpdateOpts{Name: r2IronicBPtr("x")})
			return err
		},
	} {
		if err := call(); err == nil {
			t.Fatal("expected error")
		}
	}
	r2IronicBNoHTTP(t, fake)
}

func TestR2IronicBRunbookPatchSendsOperationsVerbatimAndValidates(t *testing.T) {
	api, fake := r2IronicBRunbooks(t, 200, r2IronicBRunbookBody)
	from := "/extra/a"
	ops := []runbooks.PatchOperation{
		{Op: "replace", Path: "/description", Value: json.RawMessage(`null`)},
		{Op: "remove", Path: "/extra/k"},
		{Op: "move", Path: "/extra/b", From: &from},
	}
	if _, err := api.Patch(context.Background(), "u", ops); err != nil {
		t.Fatal(err)
	}
	call := fake.only(t)
	if call.Method != "PATCH" || call.Body != `[{"op":"replace","path":"/description","value":null},{"op":"remove","path":"/extra/k"},{"op":"move","path":"/extra/b","from":"/extra/a"}]` {
		t.Fatalf("call=%+v", call)
	}
	for _, bad := range [][]runbooks.PatchOperation{nil, {{Op: "add", Path: "/x"}}, {{Op: "remove", Path: "/x", Value: json.RawMessage(`1`)}}, {{Op: "replace", Path: "x", Value: json.RawMessage(`1`)}}, {{Op: "bogus", Path: "/x"}}, {{Op: "copy", Path: "/x"}}} {
		if _, err := api.Patch(context.Background(), "u", bad); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%+v: %v", bad, err)
		}
	}
	if len(fake.all()) != 1 {
		t.Fatalf("invalid patches reached HTTP: %+v", fake.all())
	}
	api, _ = r2IronicBRunbooks(t, 204, "")
	if _, err := api.Patch(context.Background(), "u", ops[:1]); !gophercloud.ResponseCodeIs(err, 204) {
		t.Fatalf("only 200 succeeds: %v", err)
	}
}

func TestR2IronicBRunbookDeleteDefaultsIgnoreMissing(t *testing.T) {
	api, fake := r2IronicBRunbooks(t, 204, "")
	deleted, err := api.Delete(context.Background(), "u")
	if err != nil || !deleted {
		t.Fatalf("deleted=%v err=%v", deleted, err)
	}
	if call := fake.only(t); call.Method != "DELETE" || call.Path != "/v1/runbooks/u" || call.Body != "" {
		t.Fatalf("call=%+v", call)
	}
	api, fake = r2IronicBRunbooks(t, 404, `{"error_message":"x"}`)
	deleted, err = api.Delete(context.Background(), "u")
	if err != nil || deleted {
		t.Fatalf("deleted=%v err=%v", deleted, err)
	}
	_, err = api.Delete(context.Background(), "u", runbooks.WithDeleteIgnoreMissing(false))
	var missing *resource.NotFoundError
	if !errors.Is(err, resource.ErrNotFound) || !errors.As(err, &missing) || !gophercloud.ResponseCodeIs(err, 404) {
		t.Fatalf("err=%v", err)
	}
	if len(fake.all()) != 2 {
		t.Fatalf("calls=%+v", fake.all())
	}
	for _, status := range []int{200, 202, 409} {
		api, _ = r2IronicBRunbooks(t, status, "")
		if deleted, err := api.Delete(context.Background(), "u"); err == nil || deleted {
			t.Fatalf("status %d must not succeed", status)
		}
	}
}

func TestR2IronicBRunbookTraitsRoutesBodiesAndStatuses(t *testing.T) {
	cloud := testcloud.New(t)
	fake := r2IronicBFixed(t, cloud, 204, "")
	api := runbooks.New(r2IronicBClient(cloud))
	ctx := context.Background()
	if err := api.AddTrait(ctx, "u", "CUSTOM_A/B?"); err != nil {
		t.Fatal(err)
	}
	if err := api.RemoveTrait(ctx, "u", "CUSTOM_A"); err != nil {
		t.Fatal(err)
	}
	if err := api.SetTraits(ctx, "u", []string{"CUSTOM_A", "CUSTOM_B"}); err != nil {
		t.Fatal(err)
	}
	if err := api.SetTraits(ctx, "u", nil); err != nil {
		t.Fatal(err)
	}
	if err := api.SetTraits(ctx, "u", []string{}); err != nil {
		t.Fatal(err)
	}
	calls := fake.all()
	type want struct{ method, path, body string }
	wants := []want{
		{"PUT", "/v1/runbooks/u/traits/CUSTOM_A%2FB%3F", ""},
		{"DELETE", "/v1/runbooks/u/traits/CUSTOM_A", ""},
		{"PUT", "/v1/runbooks/u/traits", `{"traits":["CUSTOM_A","CUSTOM_B"]}`},
		{"PUT", "/v1/runbooks/u/traits", `{"traits":[]}`},
		{"PUT", "/v1/runbooks/u/traits", `{"traits":[]}`},
	}
	if len(calls) != len(wants) {
		t.Fatalf("calls=%+v", calls)
	}
	for i, w := range wants {
		if calls[i].Method != w.method || calls[i].Path != w.path || calls[i].Body != w.body || calls[i].Header.Get("X-OpenStack-Ironic-API-Version") != r2IronicBVersion {
			t.Fatalf("call %d=%+v", i, calls[i])
		}
	}
	for _, c := range calls[:2] {
		if c.Header.Get("Content-Type") != "" {
			t.Fatalf("bodyless call has content type: %+v", c)
		}
	}
	// 200 and 202 are not documented successes
	for _, status := range []int{200, 202} {
		cloud := testcloud.New(t)
		r2IronicBFixed(t, cloud, status, "")
		api := runbooks.New(r2IronicBClient(cloud))
		for name, err := range map[string]error{
			"add": api.AddTrait(ctx, "u", "T"), "remove": api.RemoveTrait(ctx, "u", "T"), "set": api.SetTraits(ctx, "u", []string{"T"}),
		} {
			if !gophercloud.ResponseCodeIs(err, status) {
				t.Fatalf("%s %d: %v", name, status, err)
			}
		}
	}
}

func TestR2IronicBRunbookTraitsRejectBadInputBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	fake := r2IronicBFixed(t, cloud, 204, "")
	api := runbooks.New(r2IronicBClient(cloud))
	ctx := context.Background()
	errs := []error{
		api.AddTrait(ctx, "", "T"), api.AddTrait(ctx, "u", ""), api.RemoveTrait(ctx, "u", ""), api.RemoveTrait(ctx, "", "T"),
		api.SetTraits(ctx, "", nil), api.SetTraits(ctx, "u", []string{"A", ""}),
		api.AddTrait(r2IronicBCanceled(), "u", "T"), api.RemoveTrait(r2IronicBCanceled(), "u", "T"), api.SetTraits(r2IronicBCanceled(), "u", nil),
	}
	for i, err := range errs {
		if err == nil {
			t.Fatalf("case %d: expected error", i)
		}
	}
	r2IronicBNoHTTP(t, fake)
}

func TestR2IronicBRunbookFailedMutationIsNeverReplayed(t *testing.T) {
	cloud := testcloud.New(t)
	fake := r2IronicBServe(t, cloud, func(r2IronicBCall) (int, string) { return http.StatusServiceUnavailable, `{"error_message":"busy"}` })
	api := runbooks.New(r2IronicBClient(cloud))
	ctx := context.Background()
	_, _ = api.Create(ctx, runbooks.CreateOpts{Name: "n"})
	_, _ = api.Update(ctx, "u", runbooks.UpdateOpts{Name: r2IronicBPtr("n")})
	_ = api.AddTrait(ctx, "u", "T")
	_, _ = api.Delete(ctx, "u")
	if n := len(fake.all()); n != 4 {
		t.Fatalf("each mutation must be sent once: %d", n)
	}
}
