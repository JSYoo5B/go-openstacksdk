package gophercloudsdk_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/image"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestConnectionImageMutationsShareClientAndKeepAcknowledgements(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("live-0")
	var calls int
	methods := []string{http.MethodPut, http.MethodDelete, http.MethodPost, http.MethodPost}
	tokens := []string{"live-0", "live-1", "live-2", "live-3"}
	paths := []string{"images/fixed/tags/literal%252F", "images/fixed/tags/literal%252F", "images/fixed/actions/deactivate", "images/fixed/actions/reactivate"}
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		if calls >= len(methods) || r.Method != methods[calls] || r.URL.String() != "https://cloud.test/reverse/glance/v2/"+paths[calls] || r.Body != nil || r.Header.Get("X-Auth-Token") != tokens[calls] || r.Header.Get("X-Source") != "shared" || r.Header.Get("X-Call") != "mutation" {
			t.Fatalf("request=%d %s %s headers=%v body=%v", calls, r.Method, r.URL, r.Header, r.Body)
		}
		calls++
		// A synthetic transport retains opaque acknowledgement bytes that an
		// actual HTTP connection would suppress for a bodyless 204 response.
		return &http.Response{StatusCode: 204, Header: http.Header{"X-Proof": {"actual"}}, Body: io.NopCloser(strings.NewReader("opaque acknowledgement"))}, nil
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
	if service.RawClient() != client || service.API.RawClient() != client || client.ProviderClient != provider {
		t.Fatal("mutations replaced the cached native image client")
	}
	option := image.WithImageMutationHeader("X-Call", "mutation")
	added, err := service.AddTag(ctx, resource.ID("fixed"), "literal%2F", option)
	if err != nil || added == nil || added.ImageID != "fixed" || added.Tag != "literal%2F" || added.StatusCode != 204 || added.Header.Get("X-Proof") != "actual" || string(added.Body) != "opaque acknowledgement" {
		t.Fatalf("added=%+v err=%v", added, err)
	}
	added.Body[0] = 'X'
	added.Header.Set("X-Proof", "caller mutation")
	provider.SetToken("live-1")
	removed, err := service.RemoveTag(ctx, resource.ID("fixed"), "literal%2F", option)
	if err != nil || removed == nil || removed.ImageID != "fixed" || removed.Tag != "literal%2F" || removed.StatusCode != 204 || removed.Header.Get("X-Proof") != "actual" || string(removed.Body) != "opaque acknowledgement" {
		t.Fatalf("removed=%+v err=%v", removed, err)
	}
	provider.SetToken("live-2")
	deactivated, err := service.DeactivateImage(ctx, resource.ID("fixed"), option)
	if err != nil || deactivated == nil || deactivated.ImageID != "fixed" || deactivated.Action != "deactivate" || deactivated.StatusCode != 204 || deactivated.Header.Get("X-Proof") != "actual" || string(deactivated.Body) != "opaque acknowledgement" {
		t.Fatalf("deactivated=%+v err=%v", deactivated, err)
	}
	provider.SetToken("live-3")
	reactivated, err := service.ReactivateImage(ctx, resource.ID("fixed"), option)
	if err != nil || reactivated == nil || reactivated.ImageID != "fixed" || reactivated.Action != "reactivate" || reactivated.StatusCode != 204 || reactivated.Header.Get("X-Proof") != "actual" || string(reactivated.Body) != "opaque acknowledgement" || calls != 4 {
		t.Fatalf("reactivated=%+v err=%v calls=%d", reactivated, err, calls)
	}
}
