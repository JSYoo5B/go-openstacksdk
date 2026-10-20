package api_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules"
	"github.com/gophercloud/gophercloud/v2"
)

const manilaAccessRuleBody = `{"access":{"id":"rule-1","share_id":"share-1","access_type":"ip","access_to":"192.0.2.0/24","access_level":"rw","access_key":null,"state":"active","metadata":{"owner":"ops","flag":false},"created_at":"2026-01-02T03:04:05.000000","updated_at":"2026-01-02T04:04:05.000000","vendor_extra":1}}`

func newManilaAccessRules(cloud *testcloud.Cloud, version string) *shareaccessrules.API {
	client := cloud.Client("sharev2", "/v2")
	client.Microversion = version
	return shareaccessrules.New(client)
}

func manilaAccessRuleCode(t *testing.T, err error, want int) {
	t.Helper()
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &unexpected) || unexpected.Actual != want {
		t.Fatalf("want unexpected response code %d, got %v", want, err)
	}
}

func TestManilaAccessRuleGetReadsGlobalEndpointAndDecodesModel(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/share-access-rules/rule-1", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Values("X-Openstack-Manila-Api-Version"); !reflect.DeepEqual(got, []string{"2.82"}) {
			t.Errorf("microversion header=%v", got)
		}
		if r.URL.RawQuery != "" {
			t.Errorf("query=%q", r.URL.RawQuery)
		}
		testcloud.JSON(w, 200, manilaAccessRuleBody)
	})
	value, err := newManilaAccessRules(cloud, "2.82").Get(context.Background(), "rule-1")
	if err != nil || value == nil || value.ID != "rule-1" || value.ShareID != "share-1" || value.AccessType != "ip" || value.AccessTo != "192.0.2.0/24" ||
		value.AccessLevel != "rw" || value.AccessKey != "" || value.State != "active" || value.CreatedAt.IsZero() || value.UpdatedAt.IsZero() {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	if value.Metadata["owner"] != "ops" || value.Metadata["flag"] != false {
		t.Fatalf("metadata=%v", value.Metadata)
	}
}

func TestManilaAccessRuleGetSendsNoHeaderWithoutSelectedMicroversion(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/share-access-rules/rule-1", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Values("X-Openstack-Manila-Api-Version"); len(got) != 0 {
			t.Errorf("unexpected microversion header %v", got)
		}
		testcloud.JSON(w, 200, manilaAccessRuleBody)
	})
	if _, err := newManilaAccessRules(cloud, "").Get(context.Background(), "rule-1"); err != nil {
		t.Fatalf("the generated API does not validate the microversion locally: %v", err)
	}
}

func TestManilaAccessRuleGetAcceptsOnly200AndKeepsNative404(t *testing.T) {
	for _, status := range []int{200, 201, 202, 204} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("GET /v2/share-access-rules/rule-1", func(w http.ResponseWriter, r *http.Request) {
			if status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, status, manilaAccessRuleBody)
		})
		value, err := newManilaAccessRules(cloud, "2.82").Get(context.Background(), "rule-1")
		if (status == 200) != (err == nil) || (status == 200 && value.ID != "rule-1") {
			t.Fatalf("status %d value=%+v error=%v", status, value, err)
		}
		if status != 200 {
			manilaAccessRuleCode(t, err, status)
		}
	}
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/share-access-rules/missing", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 404, `{"itemNotFound":{"code":404}}`)
	})
	_, err := newManilaAccessRules(cloud, "2.82").Get(context.Background(), "missing")
	manilaAccessRuleCode(t, err, 404)
	if errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("the generated Get must not map 404 to ErrNotFound: %v", err)
	}
}

func TestManilaAccessRuleListFiltersByShareIDQueryAndDecodesRows(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("GET /v2/share-access-rules", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if got := r.Header.Values("X-Openstack-Manila-Api-Version"); !reflect.DeepEqual(got, []string{"2.45"}) {
			t.Errorf("microversion header=%v", got)
		}
		if got := map[string][]string(r.URL.Query()); !reflect.DeepEqual(got, map[string][]string{"share_id": {"share-1"}}) {
			t.Errorf("query=%v", got)
		}
		testcloud.JSON(w, 200, `{"access_list":[{"id":"rule-1","share_id":"share-1","access_level":"ro","metadata":{}},{"id":"rule-2","share_id":"share-1","state":"error","links":[{"rel":"next","href":"https://foreign.invalid/next"}]}]}`)
	})
	values, err := newManilaAccessRules(cloud, "2.45").List(context.Background(), "share-1")
	if err != nil || len(values) != 2 || values[0].ID != "rule-1" || values[0].AccessLevel != "ro" || values[0].Metadata == nil || len(values[0].Metadata) != 0 || values[1].State != "error" {
		t.Fatalf("values=%+v error=%v", values, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("List must read one response and never follow advertised links: %d calls", calls.Load())
	}
}

func TestManilaAccessRuleListEmptyAndNullDecodeWithoutRows(t *testing.T) {
	for _, body := range []string{`{"access_list":[]}`, `{"access_list":null}`, `{}`} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("GET /v2/share-access-rules", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, body) })
		values, err := newManilaAccessRules(cloud, "2.45").List(context.Background(), "share-1")
		if err != nil || len(values) != 0 {
			t.Fatalf("body %s values=%v error=%v", body, values, err)
		}
	}
}

func TestManilaAccessRuleListAcceptsOnly200AndKeepsNativeErrors(t *testing.T) {
	for _, status := range []int{200, 202, 204, 404, 500} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("GET /v2/share-access-rules", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, status, `{"access_list":[{"id":"rule-1"}]}`)
		})
		values, err := newManilaAccessRules(cloud, "2.45").List(context.Background(), "share-1")
		if (status == 200) != (err == nil) || (status == 200 && len(values) != 1) || calls.Load() != 1 {
			t.Fatalf("status %d values=%v error=%v calls=%d", status, values, err, calls.Load())
		}
		if status != 200 {
			manilaAccessRuleCode(t, err, status)
		}
	}
}

func TestManilaAccessRuleCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	api := newManilaAccessRules(cloud, "2.82")
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(canceled, "rule-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if _, err := api.List(canceled, "share-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled list: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.List(deadline, "share-1"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}
