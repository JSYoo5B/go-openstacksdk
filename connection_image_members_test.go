package gophercloudsdk_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	sdk "gophercloudsdk"
	"gophercloudsdk/image"
	"gophercloudsdk/resource"
)

func TestConnectionImageMembersShareClientAndConcreteDefaults(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("member-0")
	calls, retries := 0, 0
	paths := []string{"members", "members/fixed", "members/fixed", "members/fixed", "members", "members", "members/fixed", "members/missing", "members/missing"}
	methods := []string{"POST", "GET", "PUT", "GET", "GET", "GET", "DELETE", "GET", "DELETE"}
	codes := []int{200, 200, 200, 200, 200, 200, 204, 404, 404}
	const member = `{"member_id":"passive-other","image_id":"passive-parent","status":"future-status","schema":"https://foreign.test/member","created_at":"2026-10-03T09:01:02.12345678901+09:00","updated_at":null,"links":42,"precision":9007199254740993}`
	bodies := []string{member, member, member, member, `{"members":[` + member + `,null],"next":"https://foreign.test/next","schema":42}`, `{"members":[` + member + `,{"member_id":"","created_at":null,"updated_at":"unparsed","status":null}],"next":"https://foreign.test/next"}`, "opaque deletion", "missing member", "missing member"}
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		if calls >= len(paths) {
			t.Fatalf("unexpected continuation %s", r.URL)
		}
		if r.Method != methods[calls] || r.URL.String() != "https://cloud.test/reverse/glance/v2/images/parent/"+paths[calls] || r.Header.Get("X-Source") != "shared" || r.Header.Get("X-Call") != "members" || r.Header.Get("X-Auth-Token") != fmt.Sprintf("member-%d", calls) {
			t.Fatalf("request %d: %s %s headers=%v", calls, r.Method, r.URL, r.Header)
		}
		if calls < 3 && calls != 1 {
			want := `{"member":"fixed"}`
			if calls == 2 {
				want = `{"status":"accepted"}`
			}
			body, err := io.ReadAll(r.Body)
			if err != nil || string(body) != want {
				t.Fatalf("payload %d = %s err=%v", calls, body, err)
			}
		} else if r.Body != nil {
			t.Fatalf("bodyless operation %d acquired body", calls)
		}
		n := calls
		calls++
		return &http.Response{StatusCode: codes[n], Header: http.Header{"X-Proof": {fmt.Sprintf("member-%d", n)}, "Link": {`<https://foreign.test/next>; rel="next"`}}, Body: io.NopCloser(strings.NewReader(bodies[n]))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", "https://cloud.test/reverse/glance/v2/"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	service, err := conn.Image(ctx)
	if err != nil {
		t.Fatal(err)
	}
	versioned, err := conn.ImageV2(ctx)
	if err != nil {
		t.Fatal(err)
	}
	client := versioned.RawClient()
	client.MoreHeaders = map[string]string{"X-Source": "shared"}
	client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		retries++
		return nil
	}
	if service.RawClient() != client || service.API.RawClient() != client || service.API.Members.RawClient() != client || client.ProviderClient != provider {
		t.Fatal("member workflows replaced shared native client")
	}
	parent := resource.ID("parent")
	options := []image.ImageMemberOption{image.WithImageMemberOpts(image.ImageMemberOpts{Headers: map[string]string{"X-Call": "initial"}}), image.WithImageMemberHeader("X-Call", "intermediate"), image.WithImageMemberHeaders(map[string]string{"x-call": "members"})}
	check := func(value *image.ImageMember, err error) {
		t.Helper()
		if err != nil || value == nil || value.StatusCode != 200 || value.MemberID == nil || *value.MemberID != "passive-other" || value.ImageID == nil || *value.ImageID != "passive-parent" || value.Status == nil || *value.Status != "future-status" || value.Schema == nil || *value.Schema != "https://foreign.test/member" || value.CreatedAt == nil || *value.CreatedAt != "2026-10-03T09:01:02.12345678901+09:00" || value.UpdatedAt != nil || value.Links != nil || string(value.Body["precision"]) != "9007199254740993" {
			t.Fatalf("member=%+v err=%v", value, err)
		}
	}
	advance := func() { provider.SetToken(fmt.Sprintf("member-%d", calls)) }
	created, err := service.AddImageMember(ctx, parent, "fixed", options...)
	check(created, err)
	advance()
	fetched, err := service.GetImageMember(ctx, parent, "fixed", options...)
	check(fetched, err)
	created.Header.Set("X-Proof", "caller")
	created.Body["member_id"][1] = '!'
	if fetched.Header.Get("X-Proof") != "member-1" || *created.MemberID != "passive-other" || string(fetched.Body["member_id"]) != `"passive-other"` {
		t.Fatal("typed fields or separate responses alias raw evidence")
	}
	advance()
	updated, err := service.UpdateImageMember(ctx, parent, "fixed", "accepted", options...)
	check(updated, err)
	advance()
	findOptions := []image.FindImageMemberOption{image.WithFindImageMemberOpts(image.FindImageMemberOpts{Headers: map[string]string{"X-Call": "initial"}}), image.WithFindImageMemberHeader("X-Call", "intermediate"), image.WithFindImageMemberHeaders(map[string]string{"x-call": "members"}), image.WithFindImageMemberIgnoreMissing(false)}
	found, err := service.FindImageMember(ctx, parent, "fixed", findOptions...)
	check(found, err)
	advance()
	listOptions := []image.ListImageMembersOption{image.WithListImageMembersOpts(image.ListImageMembersOpts{Headers: map[string]string{"X-Call": "initial"}}), image.WithListImageMembersHeader("X-Call", "intermediate"), image.WithListImageMembersHeaders(map[string]string{"x-call": "members"}), image.WithListImageMembersMaxItems(1)}
	sequence := service.ListImageMembers(ctx, parent, listOptions...)
	if calls != 4 {
		t.Fatal("member iterator sent an eager request")
	}
	rows := 0
	for value, err := range sequence {
		check(value, err)
		rows++
	}
	if rows != 1 || calls != 5 {
		t.Fatalf("local cap rows=%d calls=%d", rows, calls)
	}
	advance()
	all, err := service.AllImageMembers(ctx, parent, image.WithListImageMembersOpts(image.ListImageMembersOpts{Headers: map[string]string{"X-Call": "members"}}))
	if err != nil || len(all) != 2 || all[1].MemberID == nil || *all[1].MemberID != "" || all[1].CreatedAt != nil || all[1].UpdatedAt == nil || *all[1].UpdatedAt != "unparsed" {
		t.Fatalf("all=%+v err=%v", all, err)
	}
	all[0].Header.Set("X-Proof", "caller row")
	if all[1].Header.Get("X-Proof") != "member-5" {
		t.Fatal("list row headers alias")
	}
	advance()
	deleteOptions := []image.RemoveImageMemberOption{image.WithRemoveImageMemberOpts(image.RemoveImageMemberOpts{Headers: map[string]string{"X-Call": "initial"}}), image.WithRemoveImageMemberHeader("X-Call", "intermediate"), image.WithRemoveImageMemberHeaders(map[string]string{"x-call": "members"}), image.WithRemoveImageMemberIgnoreMissing(false)}
	deleted, err := service.RemoveImageMember(ctx, parent, "fixed", deleteOptions...)
	if err != nil || deleted == nil || deleted.ImageID != "parent" || deleted.MemberID != "fixed" || deleted.StatusCode != 204 || string(deleted.Body) != "opaque deletion" {
		t.Fatalf("delete=%+v err=%v", deleted, err)
	}
	advance()
	missing, err := service.FindImageMember(ctx, parent, "missing", image.WithFindImageMemberOpts(image.FindImageMemberOpts{Headers: map[string]string{"X-Call": "members"}}))
	if err != nil || missing != nil {
		t.Fatalf("default find=%+v err=%v", missing, err)
	}
	advance()
	absent, err := service.RemoveImageMember(ctx, parent, "missing", image.WithRemoveImageMemberOpts(image.RemoveImageMemberOpts{Headers: map[string]string{"X-Call": "members"}}))
	if err != nil || absent != nil || calls != 9 || retries != 0 {
		t.Fatalf("default delete=%+v err=%v calls=%d retries=%d", absent, err, calls, retries)
	}
}
