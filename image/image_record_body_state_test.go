package image

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

// A sparse wire body and its descriptor view contain different information.
// Commit planning must retain raw values and presence rather than sending
// nullable defaults or getter coercions back to Glance.
func TestImageRecordBodySnapshotsKeepSparseRawValuesBeforeProjection(t *testing.T) {
	const raw = `{"id":"fixed","name":null,"protected":"false","size":"04","tags":"scalar","properties":{"retained":true},"vendor":{"n":900719925474099312345}}`
	for _, listed := range []bool{false, true} {
		name := "get"
		if listed {
			name = "list"
		}
		t.Run(name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodGet {
					t.Fatal(req.Method)
				}
				if listed {
					return taskCoreJSON(req, 203, `{"images":[`+raw+`]}`), nil
				}
				return taskCoreJSON(req, 203, raw), nil
			})
			service := New(client)
			var got *ImageRecord
			if listed {
				records, err := service.AllImageRecords(context.Background())
				if err != nil || len(records) != 1 {
					t.Fatal(records, err)
				}
				got = records[0]
			} else {
				var err error
				got, err = service.GetImageRecord(context.Background(), ImageRecordRequest{ID: "fixed"})
				if err != nil {
					t.Fatal(err)
				}
			}
			if calls != 1 || got == nil || len(got.Resource.Body) != 65 || got.bodyState == nil {
				t.Fatal(got, calls)
			}
			th.CheckDeepEquals(t, map[string]string{"is_protected": "true", "size": "4", "tags": `["scalar"]`, `is_hidden`: "null"}, map[string]string{"is_protected": string(got.Resource.Body["is_protected"]), "size": string(got.Resource.Body["size"]), "tags": string(got.Resource.Body["tags"]), "is_hidden": string(got.Resource.Body["is_hidden"])})
			for _, fields := range []map[string]json.RawMessage{got.bodyState.current, got.bodyState.original} {
				if _, present := fields["is_hidden"]; present {
					t.Fatal("absent raw field became nullable default", fields)
				}
				if _, present := fields["location"]; present {
					t.Fatal("computed location became raw body", fields)
				}
				th.AssertEquals(t, `"false"`, string(fields["is_protected"]))
				th.AssertEquals(t, `"04"`, string(fields["size"]))
				th.AssertEquals(t, `"scalar"`, string(fields["tags"]))
				th.AssertEquals(t, "null", string(fields["name"]))
				var props map[string]json.RawMessage
				if err := json.Unmarshal(fields["properties"], &props); err != nil {
					t.Fatal(err)
				}
				th.AssertEquals(t, "true", string(props["retained"]))
				th.AssertEquals(t, `{"n":900719925474099312345}`, string(props["vendor"]))
			}
			// Editing the exposed projected view or actual receipt cannot change the
			// private raw original. Current/original and cloned record are also separate.
			clone := cloneImageRecord(got)
			got.Resource.Body["size"][0] = '9'
			got.Wire.Body["size"][1] = '8'
			got.bodyState.current["size"][1] = '7'
			clone.bodyState.original["size"][1] = '6'
			th.AssertEquals(t, `"04"`, string(got.bodyState.original["size"]))
			th.AssertEquals(t, `"04"`, string(clone.bodyState.current["size"]))
		})
	}
}

func TestImageRecordBodySnapshotsKeepOriginalAcrossLocalTagUpdates(t *testing.T) {
	for _, literal := range []bool{false, true} {
		name := "fetched"
		if literal {
			name = "literal constructor"
		}
		t.Run(name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method == http.MethodGet {
					return taskCoreJSON(req, 200, `{"id":"fixed","tags":["a"]}`), nil
				}
				if req.Method != http.MethodPut {
					t.Fatal(req.Method)
				}
				return taskCoreJSON(req, 204, ""), nil
			})
			service := New(client)
			input := ImageRecordTagRequest{ID: "fixed"}
			var seed *ImageRecord
			if !literal {
				var err error
				seed, err = service.GetImageRecord(context.Background(), ImageRecordRequest{ID: "fixed"})
				if err != nil {
					t.Fatal(err)
				}
				input = ImageRecordTagRequest{Record: seed}
			}
			got, err := service.AddImageRecordTag(context.Background(), input, "b")
			if err != nil || got == nil || got.Record == nil || got.Record.bodyState == nil {
				t.Fatal(got, err)
			}
			if literal {
				if calls != 1 || len(got.Record.bodyState.original) != 1 || string(got.Record.bodyState.original["id"]) != `"fixed"` {
					t.Fatal(calls, got.Record.bodyState.original)
				}
				if _, present := got.Record.bodyState.original["tags"]; present {
					t.Fatal("constructor default forged original tags")
				}
				th.AssertEquals(t, `["b"]`, string(got.Record.bodyState.current["tags"]))
			} else {
				if calls != 2 {
					t.Fatal(calls)
				}
				th.AssertEquals(t, `["a"]`, string(got.Record.bodyState.original["tags"]))
				th.AssertEquals(t, `["a","b"]`, string(got.Record.bodyState.current["tags"]))
				th.AssertEquals(t, `["a"]`, string(seed.bodyState.current["tags"]))
			}
			// Local changes do not rewrite the original or manufacture fetch evidence.
			got.Record.bodyState.current["tags"][1] = 'x'
			if literal {
				if got.Record.Wire != nil || got.Record.StatusCode != 0 {
					t.Fatal(got.Record)
				}
			} else {
				th.AssertEquals(t, `["a"]`, string(got.Record.Wire.Body["tags"]))
				th.AssertEquals(t, `["a"]`, string(got.Record.bodyState.original["tags"]))
			}
		})
	}
}
