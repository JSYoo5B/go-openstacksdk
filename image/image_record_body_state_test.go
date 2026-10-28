package image

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
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
			if !reflect.DeepEqual(map[string]string{"is_protected": string(got.Resource.Body["is_protected"]), "size": string(got.Resource.Body["size"]), "tags": string(got.Resource.Body["tags"]), "is_hidden": string(got.Resource.Body["is_hidden"])}, map[string]string{"is_protected": "true", "size": "4", "tags": `["scalar"]`, `is_hidden`: "null"}) {
				t.Errorf("got %#v, want %#v", map[string]string{"is_protected": string(got.Resource.Body["is_protected"]), "size": string(got.Resource.Body["size"]), "tags": string(got.Resource.Body["tags"]), "is_hidden": string(got.Resource.Body["is_hidden"])}, map[string]string{"is_protected": "true", "size": "4", "tags": `["scalar"]`, `is_hidden`: "null"})
			}
			for _, fields := range []map[string]json.RawMessage{got.bodyState.current, got.bodyState.original} {
				if _, present := fields["is_hidden"]; present {
					t.Fatal("absent raw field became nullable default", fields)
				}
				if _, present := fields["location"]; present {
					t.Fatal("computed location became raw body", fields)
				}
				if string(fields["is_protected"]) != `"false"` {
					t.Fatalf("string(fields[\"is_protected\"]) = %v, want %v", string(fields["is_protected"]), `"false"`)
				}
				if string(fields["size"]) != `"04"` {
					t.Fatalf("string(fields[\"size\"]) = %v, want %v", string(fields["size"]), `"04"`)
				}
				if string(fields["tags"]) != `"scalar"` {
					t.Fatalf("string(fields[\"tags\"]) = %v, want %v", string(fields["tags"]), `"scalar"`)
				}
				if string(fields["name"]) != "null" {
					t.Fatalf("string(fields[\"name\"]) = %v, want %v", string(fields["name"]), "null")
				}
				var props map[string]json.RawMessage
				if err := json.Unmarshal(fields["properties"], &props); err != nil {
					t.Fatal(err)
				}
				if string(props["retained"]) != "true" {
					t.Fatalf("string(props[\"retained\"]) = %v, want %v", string(props["retained"]), "true")
				}
				if string(props["vendor"]) != `{"n":900719925474099312345}` {
					t.Fatalf("string(props[\"vendor\"]) = %v, want %v", string(props["vendor"]), `{"n":900719925474099312345}`)
				}
			}
			// Editing the exposed projected view or actual receipt cannot change the
			// private raw original. Current/original and cloned record are also separate.
			clone := cloneImageRecord(got)
			got.Resource.Body["size"][0] = '9'
			got.Wire.Body["size"][1] = '8'
			got.bodyState.current["size"][1] = '7'
			clone.bodyState.original["size"][1] = '6'
			if string(got.bodyState.original["size"]) != `"04"` {
				t.Fatalf("string(got.bodyState.original[\"size\"]) = %v, want %v", string(got.bodyState.original["size"]), `"04"`)
			}
			if string(clone.bodyState.current["size"]) != `"04"` {
				t.Fatalf("string(clone.bodyState.current[\"size\"]) = %v, want %v", string(clone.bodyState.current["size"]), `"04"`)
			}
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
				if string(got.Record.bodyState.current["tags"]) != `["b"]` {
					t.Fatalf("string(got.Record.bodyState.current[\"tags\"]) = %v, want %v", string(got.Record.bodyState.current["tags"]), `["b"]`)
				}
			} else {
				if calls != 2 {
					t.Fatal(calls)
				}
				if string(got.Record.bodyState.original["tags"]) != `["a"]` {
					t.Fatalf("string(got.Record.bodyState.original[\"tags\"]) = %v, want %v", string(got.Record.bodyState.original["tags"]), `["a"]`)
				}
				if string(got.Record.bodyState.current["tags"]) != `["a","b"]` {
					t.Fatalf("string(got.Record.bodyState.current[\"tags\"]) = %v, want %v", string(got.Record.bodyState.current["tags"]), `["a","b"]`)
				}
				if string(seed.bodyState.current["tags"]) != `["a"]` {
					t.Fatalf("string(seed.bodyState.current[\"tags\"]) = %v, want %v", string(seed.bodyState.current["tags"]), `["a"]`)
				}
			}
			// Local changes do not rewrite the original or manufacture fetch evidence.
			got.Record.bodyState.current["tags"][1] = 'x'
			if literal {
				if got.Record.Wire != nil || got.Record.StatusCode != 0 {
					t.Fatal(got.Record)
				}
			} else {
				if string(got.Record.Wire.Body["tags"]) != `["a"]` {
					t.Fatalf("string(got.Record.Wire.Body[\"tags\"]) = %v, want %v", string(got.Record.Wire.Body["tags"]), `["a"]`)
				}
				if string(got.Record.bodyState.original["tags"]) != `["a"]` {
					t.Fatalf("string(got.Record.bodyState.original[\"tags\"]) = %v, want %v", string(got.Record.bodyState.original["tags"]), `["a"]`)
				}
			}
		})
	}
}
