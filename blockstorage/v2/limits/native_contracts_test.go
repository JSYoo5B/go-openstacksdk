package limits_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/limits"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeLimitV2Transport func(*http.Request) (*http.Response, error)

func (transport nativeLimitV2Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeLimitV2Wire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeLimitV2Call struct{ method, path, query, body string }

func nativeLimitV2API(t *testing.T, calls *[]nativeLimitV2Call, reply func(*http.Request) *http.Response) *limits.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeLimitV2Transport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeLimitV2Call{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v2/project/"
	return limits.New(client)
}

func nativeLimitV2Operation(t *testing.T, err error) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != "Get" || wrapped.Resource != "limits" {
		t.Fatal("generated limits context", err, wrapped)
	}
}

func TestNativeLimitsV2GetRouteAndDecode(t *testing.T) {
	ctx := context.Background()
	var calls []nativeLimitV2Call
	api := nativeLimitV2API(t, &calls, func(*http.Request) *http.Response {
		return nativeLimitV2Wire(200, `{"limits":{"absolute":{"maxTotalVolumes":10,"maxTotalSnapshots":-1,"maxTotalVolumeGigabytes":1000,"maxTotalBackups":10,"maxTotalBackupGigabytes":1000,
			"totalVolumesUsed":2,"totalGigabytesUsed":20,"totalSnapshotsUsed":0,"totalBackupsUsed":1,"totalBackupGigabytesUsed":5,"maxTotalGroups":10},
			"rate":[{"regex":".*","uri":"*","limit":[{"verb":"POST","next-available":"2017-06-29T05:50:35Z","unit":"MINUTE","value":10,"remaining":9}]}]}}`)
	})
	got, err := api.Get(ctx)
	want := limits.Limits{
		Absolute: limits.Absolute{MaxTotalVolumes: 10, MaxTotalSnapshots: -1, MaxTotalVolumeGigabytes: 1000, MaxTotalBackups: 10, MaxTotalBackupGigabytes: 1000,
			TotalVolumesUsed: 2, TotalGigabytesUsed: 20, TotalBackupsUsed: 1, TotalBackupGigabytesUsed: 5},
		// next-available stays a raw string.
		Rate: []limits.Rate{{Regex: ".*", URI: "*", Limit: []limits.Limit{{Verb: "POST", NextAvailable: "2017-06-29T05:50:35Z", Unit: "MINUTE", Value: 10, Remaining: 9}}}},
	}
	if err != nil || !reflect.DeepEqual(*got, want) {
		t.Fatalf("%+v %v", got, err)
	}
	if !reflect.DeepEqual(calls, []nativeLimitV2Call{{http.MethodGet, "/cinder/v2/project/limits", "", ""}}) {
		t.Fatalf("%+v", calls)
	}
}

func TestNativeLimitsV2GetStatusesAndDecode(t *testing.T) {
	ctx := context.Background()
	for _, code := range []int{201, 202, 204, 404} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []nativeLimitV2Call
			api := nativeLimitV2API(t, &calls, func(*http.Request) *http.Response { return nativeLimitV2Wire(code, `{}`) })
			_, err := api.Get(ctx)
			nativeLimitV2Operation(t, err)
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, []int{200}) || len(calls) != 1 {
				t.Fatal(err, native)
			}
		})
	}
	t.Run("decode", func(t *testing.T) {
		body := `{}`
		var calls []nativeLimitV2Call
		api := nativeLimitV2API(t, &calls, func(*http.Request) *http.Response { return nativeLimitV2Wire(200, body) })
		// A missing limits envelope yields nil without an error.
		if got, err := api.Get(ctx); err != nil || got != nil {
			t.Fatal(got, err)
		}
		body = `{"limits":{"absolute":{"maxTotalVolumes":"ten"}}}`
		_, err := api.Get(ctx)
		nativeLimitV2Operation(t, err)
	})
}
