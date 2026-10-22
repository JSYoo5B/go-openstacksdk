package api_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules"
	"github.com/gophercloud/gophercloud/v2"
)

const legacyAccessListBody = `{"access_list":[{"id":"rule-a","access_type":"ip","access_to":"192.0.2.0/24","access_level":"rw","state":"active","vendor":{"large":9007199254740993}},{"id":"rule-b","access_type":"user","access_to":"alice","access_level":"ro","state":"new","access_key":null}]}`

func TestManilaLegacyAccessListActionSelectsActionByMicroversion(t *testing.T) {
	for _, tc := range []struct{ version, action string }{
		{"", "os-access_list"}, {"2.6", "os-access_list"}, {"2.7", "access_list"}, {"2.44", "access_list"},
	} {
		t.Run("v"+tc.version, func(t *testing.T) {
			cloud := testcloud.New(t)
			var posts atomic.Int32
			cloud.Mux.HandleFunc("POST "+shareAccessBase+"/shares/share-id/action", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				if tc.version != "" {
					checkShareAccessHeaders(t, r, tc.version)
				}
				body, _ := io.ReadAll(r.Body)
				if string(body) != `{"`+tc.action+`":null}` {
					t.Errorf("body=%s", body)
				}
				w.Header().Set("X-Request-Id", "legacy-list")
				testcloud.JSON(w, 200, legacyAccessListBody)
			})
			client := shareAccessClient(cloud, tc.version)
			result, err := shareaccessrules.New(client).ListAccessAction(context.Background(), "share-id")
			if err != nil || result == nil || posts.Load() != 1 {
				t.Fatalf("result=%+v err=%v posts=%d", result, err, posts.Load())
			}
			if result.ShareID != "share-id" || result.Action != tc.action || result.StatusCode != 200 || result.Header.Get("X-Request-Id") != "legacy-list" || string(result.Body) != legacyAccessListBody || len(result.Rules) != 2 {
				t.Fatalf("result=%+v", result)
			}
			first, second := result.Rules[0], result.Rules[1]
			if first.ID != "rule-a" || first.ParentShareID != "share-id" || first.ShareID != "" || first.AccessTo != "192.0.2.0/24" || first.AccessLevel != "rw" || string(first.Body["vendor"]) != `{"large":9007199254740993}` {
				t.Fatalf("first=%+v", first)
			}
			if _, exists := first.Body["share_id"]; exists {
				t.Fatal("a missing parent must not be fabricated")
			}
			if second.ID != "rule-b" || second.State != "new" || string(second.Body["access_key"]) != "null" {
				t.Fatalf("second=%+v", second)
			}
			if client.Microversion != tc.version {
				t.Fatal("client microversion must not be mutated")
			}
		})
	}
}

func TestManilaLegacyAccessListActionRejectsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected HTTP %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	for _, version := range []string{"2.45", "2.82"} {
		_, err := shareaccessrules.New(shareAccessClient(cloud, version)).ListAccessAction(context.Background(), "share-id")
		if !errors.Is(err, resource.ErrUnsupported) {
			t.Fatalf("%s: err=%v", version, err)
		}
	}
	for _, version := range []string{"latest", "3.1", "2.x"} {
		if _, err := shareaccessrules.New(shareAccessClient(cloud, version)).ListAccessAction(context.Background(), "share-id"); err == nil {
			t.Fatalf("%s must fail before HTTP", version)
		}
	}
	wrongType := cloud.Client("compute", shareAccessBase)
	wrongType.Microversion = "2.7"
	if _, err := shareaccessrules.New(wrongType).ListAccessAction(context.Background(), "share-id"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("wrong service type: %v", err)
	}
	api := shareaccessrules.New(shareAccessClient(cloud, "2.7"))
	if _, err := api.ListAccessAction(context.Background(), ""); err == nil {
		t.Fatal("empty share ID must fail")
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.ListAccessAction(canceled, "share-id"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled: %v", err)
	}
}

func TestManilaLegacyAccessListActionAcceptedAndHTTPErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		check      func(error) bool
	}{
		{"missing collection", `{"access":[]}`, 200, isResponseErrorWithBody(`{"access":[]}`)},
		{"null collection", `{"access_list":null}`, 200, isResponseErrorWithBody(`{"access_list":null}`)},
		{"row without id", `{"access_list":[{"access_to":"x"}]}`, 200, isResponseErrorWithBody(`{"access_list":[{"access_to":"x"}]}`)},
		{"foreign parent", `{"access_list":[{"id":"rule-a","share_id":"other-share"}]}`, 200, func(err error) bool { return errors.Is(err, shareaccessrules.ErrParentMismatch) }},
		{"not found", `{"itemNotFound":{}}`, 404, func(err error) bool { return gophercloud.ResponseCodeIs(err, 404) }},
		{"unexpected success", `{"access_list":[]}`, 202, func(err error) bool { return gophercloud.ResponseCodeIs(err, 202) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var posts atomic.Int32
			cloud.Mux.HandleFunc("POST "+shareAccessBase+"/shares/share-id/action", func(w http.ResponseWriter, r *http.Request) {
				posts.Add(1)
				testcloud.JSON(w, tc.status, tc.body)
			})
			result, err := shareaccessrules.New(shareAccessClient(cloud, "2.40")).ListAccessAction(context.Background(), "share-id")
			if result != nil || err == nil || !tc.check(err) || posts.Load() != 1 {
				t.Fatalf("result=%+v err=%v posts=%d", result, err, posts.Load())
			}
		})
	}
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST "+shareAccessBase+"/shares/share-id/action", func(w http.ResponseWriter, r *http.Request) {
		testcloud.JSON(w, 200, `{"access_list":[]}`)
	})
	result, err := shareaccessrules.New(shareAccessClient(cloud, "2.7")).ListAccessAction(context.Background(), "share-id")
	if err != nil || result == nil || result.Rules == nil || len(result.Rules) != 0 {
		t.Fatalf("empty collection: result=%+v err=%v", result, err)
	}
}

func isResponseErrorWithBody(body string) func(error) bool {
	return func(err error) bool {
		var response *resource.ResponseError
		return errors.As(err, &response) && string(response.Body) == body && response.StatusCode == 200
	}
}
