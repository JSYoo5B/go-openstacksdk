package availabilityzones_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/availabilityzones"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeZoneTransport func(*http.Request) (*http.Response, error)

func (transport nativeZoneTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeZoneAPI(t *testing.T, paths *[]string, code int, body string) *availabilityzones.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeZoneTransport(func(req *http.Request) (*http.Response, error) {
		*paths = append(*paths, req.Method+" "+req.URL.Path+"?"+req.URL.RawQuery)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v2/project/"
	return availabilityzones.New(client)
}

func TestNativeVolumeAvailabilityZonesSinglePageObjectEnvelope(t *testing.T) {
	ctx := context.Background()
	var paths []string
	// The page embeds SinglePageBase, whose array-only IsEmpty would reject this object envelope.
	api := nativeZoneAPI(t, &paths, 200, `{"availabilityZoneInfo":[{"zoneName":"nova","zoneState":{"available":true}},{"zoneName":"down","zoneState":{"available":false}}],"availabilityZoneInfo_links":[{"rel":"next","href":"/never"}]}`)
	var zones []availabilityzones.AvailabilityZone
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		zones = append(zones, *value)
	}
	want := []availabilityzones.AvailabilityZone{{ZoneName: "nova", ZoneState: availabilityzones.ZoneState{Available: true}}, {ZoneName: "down"}}
	if !reflect.DeepEqual(zones, want) || !reflect.DeepEqual(paths, []string{"GET /cinder/v2/project/os-availability-zone?"}) {
		t.Fatal(zones, paths)
	}
}

func TestNativeVolumeAvailabilityZonesStatusesAndDecode(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		code int
		body string
	}{{404, `{}`}, {200, `{"availabilityZoneInfo":[]}`}, {200, `{}`}, {200, `{"availabilityZoneInfo":{}}`}, {200, `[]`}} {
		var paths []string
		var errs []error
		count := 0
		for value, err := range nativeZoneAPI(t, &paths, tc.code, tc.body).List(ctx) {
			if err != nil {
				errs = append(errs, err)
			} else if value != nil {
				count++
			}
		}
		var native gophercloud.ErrUnexpectedResponseCode
		switch {
		case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
		case tc.body == `{"availabilityZoneInfo":[]}` && len(errs) == 0 && count == 0:
		case tc.body == `{}` && len(errs) == 0 && count == 0:
		// A non-array zone list and a bare array body fail extraction.
		case (tc.body == `{"availabilityZoneInfo":{}}` || tc.body == `[]`) && len(errs) == 1 && count == 0:
		default:
			t.Fatal(tc.code, tc.body, errs, count)
		}
		if len(paths) != 1 {
			t.Fatal(paths)
		}
	}
}
