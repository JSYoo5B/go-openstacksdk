package api_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules"
)

func accessRuleIDs(t *testing.T, scope *shareaccessrules.AccessRuleScope, options ...shareaccessrules.ListOption) ([]string, error) {
	t.Helper()
	values, err := scope.All(context.Background(), options...)
	ids := make([]string, 0, len(values))
	for _, value := range values {
		ids = append(ids, value.ID)
	}
	return ids, err
}

func TestManilaAccessListBodyFiltersStayLocalAndCountRawRows(t *testing.T) {
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc(shareAccessBase+"/share-access-rules", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if !reflect.DeepEqual(map[string][]string(r.URL.Query()), map[string][]string{"share_id": {"share-id"}}) {
			t.Errorf("body filters must not reach the wire: %s", r.URL.RawQuery)
		}
		testcloud.JSON(w, 200, `{"access_list":[`+
			`{"id":"a","access_level":"rw","state":"active","metadata":{"team":"ops","tier":"1"},"lock_deletion":false},`+
			`{"id":"b","access_level":"ro","state":"active","metadata":{"team":"ops"}},`+
			`{"id":"c","access_level":"rw","state":"error","metadata":{}},`+
			`{"id":"d","access_level":"rw","state":"active","metadata":{"team":"dev"},"lock_deletion":null}]}`)
	})
	scope := shareAccessScope(t, cloud)
	metadata := map[string]string{"team": "ops"}
	filter := shareaccessrules.WithListBodyFilter("metadata", metadata)
	metadata["team"] = "dev"
	ids, err := accessRuleIDs(t, scope, shareaccessrules.WithListBodyFilter("access_level", "ro"), shareaccessrules.WithListBodyFilter("access_level", "rw"), filter)
	if err != nil || !reflect.DeepEqual(ids, []string{"a"}) {
		t.Fatalf("ids=%v err=%v", ids, err)
	}
	// Python counts excluded rows toward max_items: a and b are consumed, b is excluded.
	ids, err = accessRuleIDs(t, scope, shareaccessrules.WithListBodyFilter("state", "active"), shareaccessrules.WithListBodyFilter("access_level", "rw"), shareaccessrules.WithListMaxItems(3))
	if err != nil || !reflect.DeepEqual(ids, []string{"a"}) {
		t.Fatalf("raw cap ids=%v err=%v", ids, err)
	}
	// Missing and null compare equal; false is not null.
	ids, err = accessRuleIDs(t, scope, shareaccessrules.WithListBodyFilter("lock_deletion", nil))
	if err != nil || !reflect.DeepEqual(ids, []string{"b", "c", "d"}) {
		t.Fatalf("null ids=%v err=%v", ids, err)
	}
	// An empty object filter needs a nonempty response object.
	ids, err = accessRuleIDs(t, scope, shareaccessrules.WithListBodyFilter("metadata", map[string]string{}))
	if err != nil || !reflect.DeepEqual(ids, []string{"a", "b", "d"}) {
		t.Fatalf("empty object ids=%v err=%v", ids, err)
	}
	before := calls.Load()
	for name, option := range map[string]shareaccessrules.ListOption{
		"scope query attribute": shareaccessrules.WithListBodyFilter("share_id", "share-id"),
		"unknown attribute":     shareaccessrules.WithListBodyFilter("vendor", 1),
		"unserializable":        shareaccessrules.WithListBodyFilter("state", func() {}),
		"forged argument":       request.WithArgument[shareaccessrules.ListOpts]("shareaccessrules.body_filters", "state=active"),
	} {
		if _, err := scope.All(context.Background(), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if calls.Load() != before {
		t.Fatal("invalid body filters made HTTP requests")
	}
}
