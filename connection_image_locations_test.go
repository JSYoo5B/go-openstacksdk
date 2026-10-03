package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gophercloud/gophercloud/v2"

	sdk "gophercloudsdk"
	"gophercloudsdk/image"
	"gophercloudsdk/resource"
)

func TestConnectionImageLocationsShareNativeClientAndPreservePassiveData(t *testing.T) {
	provider := &gophercloud.ProviderClient{}
	provider.UseTokenLock()
	provider.SetToken("location-0")
	const locationURL = "cinder://backend/image%2Fdata?key=value#fragment"
	var calls int
	provider.HTTPClient.Transport = serviceInfoTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://cloud.test/reverse/glance/v2/images/fixed/locations" || r.Header.Get("X-Source") != "shared" || r.Header.Get("X-Call") != "locations" {
			t.Fatalf("request=%s %s headers=%v", r.Method, r.URL, r.Header)
		}
		calls++
		switch calls {
		case 1:
			if r.Method != http.MethodPost || r.Header.Get("X-Auth-Token") != "location-0" || r.Body == nil {
				t.Fatalf("add request=%+v", r)
			}
			var body struct {
				URL            string            `json:"url"`
				ValidationData map[string]string `json:"validation_data"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.URL != locationURL || len(body.ValidationData) != 2 || body.ValidationData["os_hash_algo"] != "sha256" || body.ValidationData["os_hash_value"] != "AB CD" {
				t.Fatalf("submitted body=%+v err=%v", body, err)
			}
			return &http.Response{StatusCode: 202, Header: http.Header{"X-Proof": {"accepted"}}, Body: io.NopCloser(strings.NewReader("opaque acceptance"))}, nil
		case 2:
			if r.Method != http.MethodGet || r.Header.Get("X-Auth-Token") != "location-1" || r.Body != nil {
				t.Fatalf("get request=%+v", r)
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"X-Proof": {"listed"}}, Body: io.NopCloser(strings.NewReader(`[{"url":"cinder://backend/image%2Fdata?key=value#fragment","metadata":{"store":"backend","precise":9007199254740993},"id":"passive-only"}]`))}, nil
		default:
			t.Fatalf("unexpected continuation or metadata request %d", calls)
			return nil, nil
		}
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
		t.Fatal("location methods replaced the cached native image client")
	}
	added, err := service.AddImageLocation(ctx, resource.ID("fixed"), locationURL,
		image.WithImageLocationValidation("sha256", "AB CD"), image.WithAddImageLocationHeader("X-Call", "locations"))
	if err != nil || added == nil || added.ImageID != "fixed" || added.URL != locationURL || added.StatusCode != 202 || added.Header.Get("X-Proof") != "accepted" || string(added.Body) != "opaque acceptance" {
		t.Fatalf("added=%+v err=%v", added, err)
	}
	added.Body[0] = 'X'
	added.Header.Set("X-Proof", "caller mutation")
	provider.SetToken("location-1")
	listed, err := service.GetImageLocations(ctx, resource.ID("fixed"), image.WithGetImageLocationsHeader("X-Call", "locations"))
	if err != nil || listed == nil || listed.ImageID != "fixed" || listed.StatusCode != 200 || listed.Header.Get("X-Proof") != "listed" || len(listed.Locations) != 1 || calls != 2 {
		t.Fatalf("listed=%+v err=%v calls=%d", listed, err, calls)
	}
	row := listed.Locations[0]
	if row == nil || row.URL == nil || *row.URL != locationURL || string(row.Metadata["precise"]) != "9007199254740993" || string(row.Body["id"]) != `"passive-only"` {
		t.Fatalf("passive row=%+v", row)
	}
	row.Metadata["precise"][0] = '0'
	if !strings.Contains(string(listed.Body), "9007199254740993") || !strings.Contains(string(row.Body["metadata"]), "9007199254740993") {
		t.Fatal("typed metadata aliases the raw response or complete raw row")
	}
}
