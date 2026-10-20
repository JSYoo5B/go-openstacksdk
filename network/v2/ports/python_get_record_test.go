package ports_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestPortGetRecordReusesDescriptorsAndNeverFinds(t *testing.T) {
	for _, status := range []int{203, 404} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			cloud.Mux.HandleFunc("/v2.0/ports/", func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.EscapedPath() != "/v2.0/ports/folder%2Fname" || r.Header.Get("X-Owned") != "original" {
					t.Fatal(r.URL, r.Header)
				}
				testcloud.JSON(w, status, `{"port":{"id":"actual","revision_number":"٩","binding:profile":{"keep":1},"x_unknown":true}}`)
			})
			api := ports.New(cloud.Client("network", "/v2.0"))
			headers := map[string]string{"X-Owned": "original"}
			option := ports.WithRecordFindHeaders(headers)
			headers["X-Owned"] = "changed"
			row, err := api.GetRecord(context.Background(), "folder/name", option)
			if calls != 1 {
				t.Fatal(calls)
			}
			if status == 404 {
				if !gophercloud.ResponseCodeIs(err, 404) {
					t.Fatal(err)
				}
				return
			}
			if err != nil || row == nil || string(row.Resource.Body["revision_number"]) != "9" || string(row.Wire.Body["revision_number"]) != `"٩"` || string(row.Resource.Body["x_unknown"]) != "true" {
				t.Fatal(row, err)
			}
		})
	}
}

func TestPortGetRecordParentProjectionFailureRetainsReceipt(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	cloud.Mux.HandleFunc("/v2.0/ports/p", func(w http.ResponseWriter, r *http.Request) {
		calls++
		testcloud.JSON(w, 203, `{"port":{"revision_number":"²"}}`)
	})
	api := ports.New(cloud.Client("network", "/v2.0"))
	row, err := api.GetRecord(context.Background(), "p")
	var proof *resource.ResponseError
	if err == nil || row == nil || row.Resource != nil || !errors.As(err, &proof) || proof.StatusCode != 203 || calls != 1 {
		t.Fatal(row, err, calls)
	}
}
