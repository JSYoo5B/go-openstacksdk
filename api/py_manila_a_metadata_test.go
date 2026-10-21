package api_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shares"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharetypes"
	"github.com/gophercloud/gophercloud/v2"
)

// delete_share_metadata: one DELETE per key in order; 404 swallowed by default,
// 403 and other failures collected while the loop continues, aggregated at the end.
func TestPyManilaADeleteMetadataContinuesAfterFailuresAndAggregates(t *testing.T) {
	status := map[string]int{"a": 200, "b": 404, "c": 403, "d": 500, "e": 200}
	for _, tc := range []struct {
		name   string
		opts   []resource.LookupOption
		failed []string
	}{
		{"default ignores missing", nil, []string{"c", "d"}},
		{"explicit ignore", []resource.LookupOption{resource.WithIgnoreMissing()}, []string{"c", "d"}},
		{"missing is a failure", []resource.LookupOption{resource.WithMissingError()}, []string{"b", "c", "d"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var mu sync.Mutex
			var order []string
			cloud.Mux.HandleFunc("DELETE /v2/shares/share-1/metadata/{key}", func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				order = append(order, r.PathValue("key"))
				mu.Unlock()
				if status[r.PathValue("key")] == 200 {
					w.WriteHeader(200)
					return
				}
				testcloud.JSON(w, status[r.PathValue("key")], `{}`)
			})
			err := newManilaShares(cloud, "2.82").DeleteMetadata(context.Background(), "share-1", []string{"a", "b", "c", "d", "e"}, tc.opts...)
			var failure *shares.MetadataDeleteError
			if !errors.As(err, &failure) || !reflect.DeepEqual(failure.Failed, tc.failed) || len(failure.Unattempted) != 0 || len(failure.Causes) != len(tc.failed) {
				t.Fatalf("err=%v failure=%+v", err, failure)
			}
			if !reflect.DeepEqual(order, []string{"a", "b", "c", "d", "e"}) {
				t.Fatalf("order=%v", order)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if !errors.As(err, &native) {
				t.Fatalf("native response error is not reachable: %v", err)
			}
		})
	}
}

func TestPyManilaADeleteMetadataSucceedsWithEmptyAndMissingKeys(t *testing.T) {
	cloud := testcloud.New(t)
	var calls int
	cloud.Mux.HandleFunc("DELETE /v2/shares/share-1/metadata/gone", func(w http.ResponseWriter, r *http.Request) {
		calls++
		testcloud.JSON(w, 404, `{}`)
	})
	api := newManilaShares(cloud, "")
	if err := api.DeleteMetadata(context.Background(), "share-1", nil); err != nil {
		t.Fatal(err)
	}
	if err := api.DeleteMetadata(context.Background(), "share-1", []string{"gone"}); err != nil || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestPyManilaADeleteMetadataCancellationStopsAndReportsUnattempted(t *testing.T) {
	cloud := testcloud.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	var order []string
	cloud.Mux.HandleFunc("DELETE /v2/shares/share-1/metadata/{key}", func(w http.ResponseWriter, r *http.Request) {
		order = append(order, r.PathValue("key"))
		if r.PathValue("key") == "b" {
			cancel()
		}
		w.WriteHeader(200)
	})
	err := newManilaShares(cloud, "").DeleteMetadata(ctx, "share-1", []string{"a", "b", "c", "d"})
	var failure *shares.MetadataDeleteError
	if !errors.Is(err, context.Canceled) || !errors.As(err, &failure) || !reflect.DeepEqual(failure.Unattempted, []string{"c", "d"}) {
		t.Fatalf("err=%v failure=%+v", err, failure)
	}
	if len(order) > 2 {
		t.Fatalf("requests after cancellation: %v", order)
	}
}

func TestPyManilaADeleteMetadataRejectsInvalidInputBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("unexpected request %s", r.URL) })
	api := newManilaShares(cloud, "")
	ctx := context.Background()
	for name, err := range map[string]error{
		"empty id":  api.DeleteMetadata(ctx, "", []string{"a"}),
		"slash id":  api.DeleteMetadata(ctx, "a/b", []string{"a"}),
		"empty key": api.DeleteMetadata(ctx, "share-1", []string{"a", ""}),
		"nil opt":   api.DeleteMetadata(ctx, "share-1", []string{"a"}, nil),
	} {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s err=%v", name, err)
		}
	}
}

// set_share_metadata(replace=True) is PUT with the whole metadata object; an
// empty/None metadata dict is sent as {}.
func TestPyManilaAReplaceMetadataPutsWholeObject(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   map[string]string
		body string
	}{
		{"values", map[string]string{"empty": "", "k": "v"}, `{"metadata":{"empty":"","k":"v"}}`},
		{"empty", map[string]string{}, `{"metadata":{}}`},
		{"nil", nil, `{"metadata":{}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Mux.HandleFunc("PUT /v2/shares/share-1/metadata", func(w http.ResponseWriter, r *http.Request) {
				manilaShareVersion(t, r, "2.82")
				if got := manilaShareRead(t, r); got != tc.body {
					t.Errorf("body=%s", got)
				}
				testcloud.JSON(w, 200, `{"metadata":{"k":"v"}}`)
			})
			got, err := newManilaShares(cloud, "2.82").ReplaceMetadata(context.Background(), "share-1", tc.in)
			if err != nil || !reflect.DeepEqual(got, map[string]string{"k": "v"}) {
				t.Fatalf("got=%v err=%v", got, err)
			}
		})
	}
}

func TestPyManilaAReplaceMetadataStatusesAndPreflight(t *testing.T) {
	for _, status := range []int{202, 204, 404, 500} {
		cloud := testcloud.New(t)
		var calls int
		cloud.Mux.HandleFunc("PUT /v2/shares/share-1/metadata", func(w http.ResponseWriter, r *http.Request) {
			calls++
			testcloud.JSON(w, status, `{}`)
		})
		_, err := newManilaShares(cloud, "").ReplaceMetadata(context.Background(), "share-1", map[string]string{"k": "v"})
		manilaShareCode(t, err, status)
		if calls != 1 {
			t.Fatalf("status %d replayed: %d", status, calls)
		}
	}
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("unexpected request %s", r.URL) })
	if _, err := newManilaShares(cloud, "").ReplaceMetadata(context.Background(), "a/b", nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := newManilaShares(cloud, "").ReplaceMetadata(ctx, "share-1", nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

// delete_share_type_extra_spec_property defaults to ignore_missing=False.
func TestPyManilaAUnsetExtraSpecPropertyMissingPolicy(t *testing.T) {
	cloud := testcloud.New(t)
	status := 404
	var calls int
	cloud.Mux.HandleFunc("DELETE /v2/types/st-1/extra_specs/snapshot_support", func(w http.ResponseWriter, r *http.Request) {
		calls++
		manilaSupportCheckVersion(t, r)
		if status == 202 {
			w.WriteHeader(202)
			return
		}
		testcloud.JSON(w, status, `{}`)
	})
	api := sharetypes.New(newManilaSupportClient(cloud))
	ctx := context.Background()
	var native gophercloud.ErrUnexpectedResponseCode
	err := api.UnsetExtraSpecProperty(ctx, "st-1", "snapshot_support")
	if !errors.Is(err, resource.ErrNotFound) || !errors.As(err, &native) || native.Actual != 404 {
		t.Fatalf("err=%v", err)
	}
	if err := api.UnsetExtraSpecProperty(ctx, "st-1", "snapshot_support", resource.WithIgnoreMissing()); err != nil {
		t.Fatal(err)
	}
	status = 403
	if err := api.UnsetExtraSpecProperty(ctx, "st-1", "snapshot_support", resource.WithIgnoreMissing()); err == nil || errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("403 must fail: %v", err)
	}
	status = 202
	if err := api.UnsetExtraSpecProperty(ctx, "st-1", "snapshot_support"); err != nil {
		t.Fatal(err)
	}
	if calls != 4 {
		t.Fatalf("calls=%d", calls)
	}
	for _, key := range []string{"", " "} {
		if err := api.UnsetExtraSpecProperty(ctx, "st-1", key); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("key %q err=%v", key, err)
		}
	}
	if calls != 4 {
		t.Fatal("invalid key reached HTTP")
	}
}

// create_share_type(name, extra_specs={...}) can send driver_handles_share_servers
// = false, which the native required-bool CreateOpts cannot express.
func TestPyManilaACreateShareTypeWithExtraSpecsPreservesFalseNullAndOmission(t *testing.T) {
	yes, no := true, false
	empty := ""
	for _, tc := range []struct {
		name string
		opts sharetypes.CreateSpecsOpts
		want map[string]string
	}{
		{"dhss false", sharetypes.CreateSpecsOpts{Name: "gold", ExtraSpecs: map[string]any{"driver_handles_share_servers": false, "snapshot_support": nil, "empty": ""}},
			map[string]string{"name": `"gold"`, "extra_specs": `{"driver_handles_share_servers":false,"empty":"","snapshot_support":null}`}},
		{"all fields", sharetypes.CreateSpecsOpts{Name: "gold", Description: &empty, IsPublic: &no, ExtraSpecs: map[string]any{}},
			map[string]string{"name": `"gold"`, "description": `""`, "os-share-type-access:is_public": `false`, "extra_specs": `{}`}},
		{"name only", sharetypes.CreateSpecsOpts{Name: "gold", IsPublic: &yes},
			map[string]string{"name": `"gold"`, "os-share-type-access:is_public": `true`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls int
			cloud.Mux.HandleFunc("POST /v2/types", func(w http.ResponseWriter, r *http.Request) {
				calls++
				manilaSupportCheckVersion(t, r)
				if got := manilaSupportStrings(manilaSupportEnvelope(t, r, "share_type")); !reflect.DeepEqual(got, tc.want) {
					t.Errorf("body=%v", got)
				}
				testcloud.JSON(w, 202, manilaShareTypeBody)
			})
			value, err := sharetypes.New(newManilaSupportClient(cloud)).CreateWithExtraSpecs(context.Background(), tc.opts)
			if err != nil || value == nil || value.ID != "st-1" || calls != 1 {
				t.Fatalf("value=%+v err=%v calls=%d", value, err, calls)
			}
		})
	}
}

func TestPyManilaACreateShareTypeWithExtraSpecsStatusesAndPreflight(t *testing.T) {
	for _, status := range []int{201, 204, 400, 500} {
		cloud := testcloud.New(t)
		var calls int
		cloud.Mux.HandleFunc("POST /v2/types", func(w http.ResponseWriter, r *http.Request) {
			calls++
			testcloud.JSON(w, status, `{}`)
		})
		_, err := sharetypes.New(newManilaSupportClient(cloud)).CreateWithExtraSpecs(context.Background(), sharetypes.CreateSpecsOpts{Name: "gold"})
		manilaSupportUnexpected(t, err, status)
		if calls != 1 {
			t.Fatalf("status %d replayed: %d", status, calls)
		}
	}
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { t.Errorf("unexpected request %s", r.URL) })
	api := sharetypes.New(newManilaSupportClient(cloud))
	for _, opts := range []sharetypes.CreateSpecsOpts{{}, {Name: " "}, {Name: "x", ExtraSpecs: map[string]any{" ": true}}} {
		if _, err := api.CreateWithExtraSpecs(context.Background(), opts); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("opts %+v err=%v", opts, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.CreateWithExtraSpecs(ctx, sharetypes.CreateSpecsOpts{Name: "x"}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
