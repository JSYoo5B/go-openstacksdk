package manageablevolumes_test

import (
	"context"
	"encoding/json"
	"errors"
	api "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/manageablevolumes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"net/http"
	"testing"
)

func TestPythonManageableVolumesListDefaultsDetailsAndViews(t *testing.T) {
	for _, detail := range []bool{false, true} {
		t.Run(map[bool]string{false: "summary", true: "detail"}[detail], func(t *testing.T) {
			cloud := testcloud.New(t)
			calls := 0
			cloud.Mux.HandleFunc("/v3/project/", func(w http.ResponseWriter, r *http.Request) {
				calls++
				path := "/v3/project/manageable_volumes"
				if detail {
					path += "/detail"
				}
				if r.URL.Path != path || r.URL.Query().Get("cluster") != "c" || r.URL.Query().Has("unknown") {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 202, `{"manageable-volumes":[{"id":"one","reference":7,"safe_to_manage":"False","size":"٢","x_extension":true,"cinder_id":null}]}`)
			})
			a := api.New(cloud.Client("block-storage", "/v3/project/"))
			rows, err := a.AllRecords(context.Background(), api.WithRecordListDetails(detail), api.WithRecordListQuery("cluster", "c"), api.WithRecordListQuery("unknown", 1))
			if err != nil || len(rows) != 1 || calls != 1 {
				t.Fatal(rows, err, calls)
			}
			row := rows[0]
			for key, want := range map[string]string{"reference": "{}", "safe_to_manage": "true", "size": "2", "name": "null", "location": "null"} {
				if string(row.Resource.Body[key]) != want {
					t.Fatal(key, row.Resource.Body)
				}
			}
			if row.Resource.Body["x_extension"] != nil || string(row.Wire.Body["x_extension"]) != "true" || row.StatusCode != 202 {
				t.Fatal(row)
			}
			row.Resource.Body["size"][0] = '9'
			if string(row.Wire.Body["size"]) != `"٢"` {
				t.Fatal("view aliases wire")
			}
		})
	}
}
func TestPythonManageableVolumesFiltersCapsProxyAndGuard(t *testing.T) {
	cloud := testcloud.New(t)
	calls := 0
	cloud.Mux.HandleFunc("/v3/project/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		testcloud.JSON(w, 200, `{"manageable-volumes":[{"id":"one","safe_to_manage":false},{"id":"two","safe_to_manage":true}]}`)
	})
	a := api.New(cloud.Client("volumev3", "/v3/project/"))
	result, err := a.ProxyListRecords(context.Background(), api.WithRecordListJMESPath("type(@)"))
	if err != nil || calls != 0 || string(result.JSON) != "null" {
		t.Fatal(result, err, calls)
	}
	rows, err := a.AllRecords(context.Background(), api.WithRecordListQuery("safe_to_manage", true), api.WithRecordListMaxItems(1))
	if err != nil || len(rows) != 0 || calls != 1 {
		t.Fatal(rows, err, calls)
	}
	rows, err = a.AllRecords(context.Background(), api.WithRecordListQuery("safe_to_manage", true))
	if err != nil || len(rows) != 1 || string(rows[0].Resource.Body["id"]) != `"two"` {
		t.Fatal(rows, err)
	}
	_, err = a.AllRecords(context.Background(), api.WithRecordListQuery("base_path", "/outside"))
	if !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	before := calls
	_, err = a.AllRecords(context.Background(), func(v *api.RecordListOpts) error {
		v.Query = map[string]json.RawMessage{}
		*a = *api.New(cloud.Client("volumev3", "/changed/"))
		return nil
	})
	if !errors.Is(err, resource.ErrInvalidOption) || calls != before {
		t.Fatal(err, calls)
	}
}
