package api_test

import (
	"context"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

func TestShareAccessRuleEscapeListKeepsReservedShareIDInOneQueryValue(t *testing.T) {
	for _, shareID := range []string{"x&injected=yes", "a#frag", "a b+c", "p=q;r", "100%", "../x?y=z", "plain-id", "ünï"} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("GET /v2/share-access-rules", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if got := map[string][]string(r.URL.Query()); !reflect.DeepEqual(got, map[string][]string{"share_id": {shareID}}) {
				t.Errorf("share %q query=%v raw=%q", shareID, got, r.URL.RawQuery)
			}
			testcloud.JSON(w, 200, `{"access_list":[{"id":"rule-1","share_id":"s"}]}`)
		})
		values, err := newManilaAccessRules(cloud, "2.45").List(context.Background(), shareID)
		if err != nil || len(values) != 1 || values[0].ID != "rule-1" || calls.Load() != 1 {
			t.Fatalf("share %q values=%+v error=%v calls=%d", shareID, values, err, calls.Load())
		}
	}
}

func TestShareAccessRuleEscapeListKeepsOnly200AndNoRetry(t *testing.T) {
	for _, status := range []int{201, 204, 404, 500} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("GET /v2/share-access-rules", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, status, `{"access_list":[]}`)
		})
		values, err := newManilaAccessRules(cloud, "2.45").List(context.Background(), "x&injected=yes")
		if err == nil || values != nil || calls.Load() != 1 {
			t.Fatalf("status %d values=%+v error=%v calls=%d", status, values, err, calls.Load())
		}
		manilaAccessRuleCode(t, err, status)
	}
}
