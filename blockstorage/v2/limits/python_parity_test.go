package limits_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/limits"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

func TestPythonLimitsGetIgnoresProject(t *testing.T) {
	var requests []string
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.Method+" "+req.URL.Path+"?"+req.URL.RawQuery)
		body := `{"limits":{"absolute":{"maxTotalVolumes":10,"maxTotalBackupGigabytes":1000,"totalVolumesUsed":2,"totalBackupsUsed":1},"rate":[{"regex":".*","uri":"*","limit":[{"verb":"POST","next-available":"2026-10-11T00:00:00Z","unit":"MINUTE","value":10,"remaining":9}]}]}}`
		return testcloud.Response(200, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v2/project/"
	// get_limits() and get_limits(project) both GET /limits: v2 Limits has no project URI or query.
	got, err := limits.New(client).Get(context.Background())
	if err != nil || got.Absolute.MaxTotalVolumes != 10 || got.Absolute.MaxTotalBackupGigabytes != 1000 || got.Absolute.TotalVolumesUsed != 2 || got.Absolute.TotalBackupsUsed != 1 {
		t.Fatal(got, err)
	}
	if len(got.Rate) != 1 || got.Rate[0].Regex != ".*" || got.Rate[0].Limit[0].Verb != "POST" || got.Rate[0].Limit[0].NextAvailable == "" || got.Rate[0].Limit[0].Remaining != 9 {
		t.Fatal(got.Rate)
	}
	if len(requests) != 1 || requests[0] != "GET /cinder/v2/project/limits?" {
		t.Fatal(requests)
	}
}
