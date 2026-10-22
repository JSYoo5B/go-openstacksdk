package api_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

// Pinned Python QuotaSet/QuotaClassSet._translate_response swallow a
// non-JSON success body (ValueError) and return a Resource that holds only the
// requested identity. Go reports the accepted response as an error instead and
// never retries the read to obtain a parsable body.
func TestPyManilaQuotaReadsRejectNonJSONSuccessBodiesPythonIgnores(t *testing.T) {
	for _, body := range []string{"not json", ""} {
		cloud := testcloud.New(t)
		var project, class atomic.Int32
		cloud.Mux.HandleFunc("GET /manila/v2/catalog-project/quota-sets/quota-project", func(w http.ResponseWriter, r *http.Request) {
			project.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(body))
		})
		cloud.Mux.HandleFunc("GET /manila/v2/catalog-project/quota-class-sets/fixed-class", func(w http.ResponseWriter, r *http.Request) {
			class.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(body))
		})
		if value, err := newManilaQuotaScope(t, cloud, "2.80").Get(context.Background()); value != nil || err == nil || project.Load() != 1 {
			t.Fatalf("project %q: value=%+v err=%v calls=%d", body, value, err, project.Load())
		}
		if value, err := newManilaQuotaClassScope(t, cloud, "2.80").Get(context.Background()); value != nil || err == nil || class.Load() != 1 {
			t.Fatalf("class %q: value=%+v err=%v calls=%d", body, value, err, class.Load())
		}
	}
}

// Python flattens {limit,in_use,reserved} objects into the limit plus usage
// and reservation maps. Plain Get never asks for that shape; a Detail-shaped
// object on the limits route is rejected by the typed int64 model rather than
// being silently flattened, while Detail keeps limit/in_use/reserved apart.
func TestPyManilaQuotaGetDoesNotFlattenDetailShapedLimits(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /manila/v2/catalog-project/quota-sets/quota-project", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("Get must not send query parameters: %s", r.URL.RawQuery)
		}
		testcloud.JSON(w, 200, `{"quota_set":{"shares":{"limit":10,"in_use":2,"reserved":1}}}`)
	})
	cloud.Mux.HandleFunc("GET /manila/v2/catalog-project/quota-sets/quota-project/detail", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"quota_set":{"shares":{"limit":10,"in_use":2,"reserved":1}}}`)
	})
	scope := newManilaQuotaScope(t, cloud, "2.80")
	if value, err := scope.Get(context.Background()); value != nil || err == nil {
		t.Fatalf("value=%+v err=%v", value, err)
	}
	detail, err := scope.Detail(context.Background())
	if err != nil || detail.Shares == nil || *detail.Shares.Limit != 10 || *detail.Shares.InUse != 2 || *detail.Shares.Reserved != 1 {
		t.Fatalf("detail=%+v err=%v", detail, err)
	}
}
