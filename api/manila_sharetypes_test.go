package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/sharetypes"
	"github.com/gophercloud/gophercloud/v2"
)

const manilaShareTypeBody = `{"share_type":{"id":"st-1","name":"gold","os-share-type-access:is_public":false,"required_extra_specs":{"driver_handles_share_servers":"false"},"extra_specs":{"snapshot_support":"true"},"description":null,"is_default":false,"vendor_extra":1}}`

const manilaSupportVersion = "2.60"

// newManilaSupportClient returns a native Manila service client with a fixed microversion.
func newManilaSupportClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("shared-file-system", "/v2")
	client.Microversion = manilaSupportVersion
	return client
}

func manilaSupportCheckVersion(t *testing.T, r *http.Request) {
	t.Helper()
	if got := r.Header.Get("X-Openstack-Manila-Api-Version"); got != manilaSupportVersion {
		t.Errorf("%s %s microversion header=%q", r.Method, r.URL.Path, got)
	}
}

func manilaSupportFalse() *bool { value := false; return &value }

// manilaSupportEnvelope decodes the named request envelope into raw JSON members.
func manilaSupportEnvelope(t *testing.T, r *http.Request, key string) map[string]json.RawMessage {
	t.Helper()
	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("request body: %v", err)
		return nil
	}
	if key == "" {
		return body
	}
	var inner map[string]json.RawMessage
	if err := json.Unmarshal(body[key], &inner); err != nil {
		t.Errorf("envelope %q in %v: %v", key, body, err)
	}
	return inner
}

func manilaSupportStrings(values map[string]json.RawMessage) map[string]string {
	got := map[string]string{}
	for key, value := range values {
		got[key] = string(value)
	}
	return got
}

func manilaSupportUnexpected(t *testing.T, err error, status int) {
	t.Helper()
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &unexpected) || unexpected.Actual != status {
		t.Fatalf("want HTTP %d response error, got %v", status, err)
	}
}

func TestManilaShareTypeCreateKeepsFalseAndExtensionsAndAccepts200Or202(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, true}, {201, false}, {204, false}} {
		cloud := testcloud.New(t)
		var calls atomic.Int32
		cloud.Mux.HandleFunc("POST /v2/types", func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			manilaSupportCheckVersion(t, r)
			body := manilaSupportEnvelope(t, r, "share_type")
			want := map[string]string{
				"name":                           `"gold"`,
				"os-share-type-access:is_public": `false`,
				"extra_specs":                    `{"driver_handles_share_servers":true,"snapshot_support":false}`,
				"vendor":                         `{"enabled":false}`,
			}
			if got := manilaSupportStrings(body); !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, manilaShareTypeBody)
		})
		value, err := sharetypes.New(newManilaSupportClient(cloud)).Create(context.Background(), sharetypes.CreateOpts{
			Name: "gold", IsPublic: false,
			ExtraSpecs: sharetypes.ExtraSpecsOpts{DriverHandlesShareServers: true, SnapshotSupport: manilaSupportFalse()},
		}, sharetypes.WithCreateField("vendor", map[string]bool{"enabled": false}))
		if tc.success != (err == nil) || calls.Load() != 1 {
			t.Fatalf("status %d value=%+v error=%v calls=%d", tc.status, value, err, calls.Load())
		}
		if tc.success && (value.ID != "st-1" || value.IsPublic || value.Description != nil || value.IsDefault == nil || *value.IsDefault) {
			t.Fatalf("status %d value=%+v", tc.status, value)
		}
		if !tc.success {
			manilaSupportUnexpected(t, err, tc.status)
		}
	}
}

func TestManilaShareTypeCreateCannotSendFalseDriverHandlesShareServers(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	// Gophercloud marks driver_handles_share_servers required, so its zero
	// value (false) fails local validation instead of being sent.
	_, err := sharetypes.New(newManilaSupportClient(cloud)).Create(context.Background(), sharetypes.CreateOpts{
		Name: "gold", ExtraSpecs: sharetypes.ExtraSpecsOpts{DriverHandlesShareServers: false, SnapshotSupport: manilaSupportFalse()},
	})
	if err == nil {
		t.Fatal("false driver_handles_share_servers must be rejected before HTTP")
	}
}

func TestManilaShareTypeCreateOmitsUnsetSnapshotSupport(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("POST /v2/types", func(w http.ResponseWriter, r *http.Request) {
		body := manilaSupportEnvelope(t, r, "share_type")
		if got := string(body["extra_specs"]); got != `{"driver_handles_share_servers":true}` {
			t.Errorf("extra_specs=%s", got)
		}
		testcloud.JSON(w, 200, manilaShareTypeBody)
	})
	if _, err := sharetypes.New(newManilaSupportClient(cloud)).Create(context.Background(), sharetypes.CreateOpts{
		Name: "gold", ExtraSpecs: sharetypes.ExtraSpecsOpts{DriverHandlesShareServers: true},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestManilaShareTypeRejectsProtectedOrNilOptionsBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	})
	api := sharetypes.New(newManilaSupportClient(cloud))
	ctx := context.Background()
	for name, err := range map[string]error{
		"create name": func() error {
			_, err := api.Create(ctx, sharetypes.CreateOpts{Name: "x", ExtraSpecs: sharetypes.ExtraSpecsOpts{DriverHandlesShareServers: true}}, sharetypes.WithCreateField("name", "shadow"))
			return err
		}(),
		"create nil": func() error {
			_, err := api.Create(ctx, sharetypes.CreateOpts{Name: "x", ExtraSpecs: sharetypes.ExtraSpecsOpts{DriverHandlesShareServers: true}}, nil)
			return err
		}(),
		"update is_public": func() error {
			_, err := api.Update(ctx, "st-1", sharetypes.UpdateOpts{}, sharetypes.WithUpdateField("description", "x"))
			return err
		}(),
		"update nil": func() error { _, err := api.Update(ctx, "st-1", sharetypes.UpdateOpts{}, nil); return err }(),
		"set extra specs": func() error {
			_, err := api.SetExtraSpecs(ctx, "st-1", sharetypes.SetExtraSpecsOpts{ExtraSpecs: map[string]any{"a": 1}}, sharetypes.WithSetExtraSpecsField("a", 2))
			return err
		}(),
		"set extra specs nil": func() error {
			_, err := api.SetExtraSpecs(ctx, "st-1", sharetypes.SetExtraSpecsOpts{ExtraSpecs: map[string]any{"a": 1}}, nil)
			return err
		}(),
		"add access project": func() error {
			return api.AddAccess(ctx, "st-1", sharetypes.AccessOpts{Project: "p"}, sharetypes.WithAddAccessField("project", "q"))
		}(),
		"add access nil": func() error { return api.AddAccess(ctx, "st-1", sharetypes.AccessOpts{}, nil) }(),
		"remove access": func() error {
			return api.RemoveAccess(ctx, "st-1", sharetypes.AccessOpts{}, sharetypes.WithRemoveAccessField("project", "q"))
		}(),
		"remove access nil": func() error { return api.RemoveAccess(ctx, "st-1", sharetypes.AccessOpts{}, nil) }(),
	} {
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for range api.List(ctx, nil) {
		// The sequence yields one invalid-option error and no HTTP request.
	}
}

func TestManilaShareTypeUpdateSendsEmptyValuesAndAcceptsOnly200(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, false}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("PUT /v2/types/st-1", func(w http.ResponseWriter, r *http.Request) {
			manilaSupportCheckVersion(t, r)
			want := map[string]string{"name": `""`, "description": `""`, "share_type_access:is_public": `false`, "vendor": `null`}
			if got := manilaSupportStrings(manilaSupportEnvelope(t, r, "share_type")); !reflect.DeepEqual(got, want) {
				t.Errorf("body=%v", got)
			}
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, manilaShareTypeBody)
		})
		empty := ""
		value, err := sharetypes.New(newManilaSupportClient(cloud)).Update(context.Background(), "st-1", sharetypes.UpdateOpts{
			Name: &empty, Description: &empty, IsPublic: manilaSupportFalse(),
		}, sharetypes.WithUpdateField("vendor", nil))
		if tc.success != (err == nil) || (tc.success && value.ID != "st-1") {
			t.Fatalf("status %d value=%+v error=%v", tc.status, value, err)
		}
		if !tc.success {
			manilaSupportUnexpected(t, err, tc.status)
		}
	}
}

func TestManilaShareTypeUpdateWithNoFieldsSendsEmptyEnvelope(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("PUT /v2/types/st-1", func(w http.ResponseWriter, r *http.Request) {
		var raw map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil || string(raw["share_type"]) != `{}` || len(raw) != 1 {
			t.Errorf("body=%v error=%v", raw, err)
		}
		testcloud.JSON(w, 200, `{"share_type":{"id":"st-1","name":"gold","share_type_access:is_public":true}}`)
	})
	value, err := sharetypes.New(newManilaSupportClient(cloud)).Update(context.Background(), "st-1", sharetypes.UpdateOpts{})
	if err != nil || !value.IsPublic {
		t.Fatalf("legacy is_public key must override: value=%+v error=%v", value, err)
	}
}

func TestManilaShareTypeGetGetDefaultAndMissingPolicies(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/types/{id}", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		switch r.PathValue("id") {
		case "default":
			testcloud.JSON(w, 200, `{"share_type":{"id":"st-d","name":"default","os-share-type-access:is_public":true,"is_default":true}}`)
		case "missing":
			testcloud.JSON(w, 404, `{"itemNotFound":{"code":404}}`)
		default:
			testcloud.JSON(w, 200, manilaShareTypeBody)
		}
	})
	api := sharetypes.New(newManilaSupportClient(cloud))
	ctx := context.Background()
	value, err := api.Get(ctx, "st-1")
	if err != nil || value.Name != "gold" || value.Description != nil || value.ExtraSpecs["snapshot_support"] != "true" || value.RequiredExtraSpecs["driver_handles_share_servers"] != "false" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
	def, err := api.GetDefault(ctx)
	if err != nil || def.ID != "st-d" || !def.IsPublic || def.IsDefault == nil || !*def.IsDefault {
		t.Fatalf("default=%+v error=%v", def, err)
	}
	_, err = api.Get(ctx, "missing")
	manilaSupportUnexpected(t, err, 404)
	if errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("native Get must not add ErrNotFound: %v", err)
	}
}

func TestManilaShareTypeDeleteAcceptsOnly202Or204AndRemoveIgnoresMissing(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{202, true}, {204, true}, {200, false}, {404, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("DELETE /v2/types/st-1", func(w http.ResponseWriter, r *http.Request) {
			manilaSupportCheckVersion(t, r)
			w.WriteHeader(tc.status)
		})
		err := sharetypes.New(newManilaSupportClient(cloud)).Delete(context.Background(), "st-1")
		if tc.success != (err == nil) {
			t.Fatalf("status %d error=%v", tc.status, err)
		}
		if !tc.success {
			manilaSupportUnexpected(t, err, tc.status)
		}
	}
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("DELETE /v2/types/missing", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	api := sharetypes.New(newManilaSupportClient(cloud))
	manilaSupportUnexpected(t, api.Delete(context.Background(), "missing"), 404)
	if err := api.Remove(context.Background(), resource.ID("missing")); err != nil {
		t.Fatalf("Remove ignores a missing share type by default: %v", err)
	}
}

func TestManilaShareTypeExtraSpecsReadSetAndUnset(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/types/st-1/extra_specs", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		testcloud.JSON(w, 200, `{"extra_specs":{"snapshot_support":"true","empty":""}}`)
	})
	cloud.Mux.HandleFunc("POST /v2/types/st-1/extra_specs", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		// An extension field is merged into the extra_specs map itself.
		want := map[string]string{"extra_specs": `{"empty":"","flag":false,"none":null,"vendor":[]}`}
		if got := manilaSupportStrings(manilaSupportEnvelope(t, r, "")); !reflect.DeepEqual(got, want) {
			t.Errorf("body=%v", got)
		}
		testcloud.JSON(w, 200, `{"extra_specs":{"flag":false,"empty":"","none":null}}`)
	})
	var unset []string
	cloud.Mux.HandleFunc("DELETE /v2/types/st-1/extra_specs/{key}", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		unset = append(unset, r.PathValue("key"))
		if r.PathValue("key") == "gone" {
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(map[string]int{"a": 202, "b": 204}[r.PathValue("key")])
	})
	api := sharetypes.New(newManilaSupportClient(cloud))
	ctx := context.Background()
	specs, err := api.GetExtraSpecs(ctx, "st-1")
	if err != nil || specs["snapshot_support"] != "true" || specs["empty"] != "" {
		t.Fatalf("specs=%v error=%v", specs, err)
	}
	set, err := api.SetExtraSpecs(ctx, "st-1", sharetypes.SetExtraSpecsOpts{ExtraSpecs: map[string]any{"flag": false, "empty": "", "none": nil}},
		sharetypes.WithSetExtraSpecsField("vendor", []string{}))
	if err != nil || set["flag"] != false || set["empty"] != "" {
		t.Fatalf("set=%v error=%v", set, err)
	}
	for _, key := range []string{"a", "b"} {
		if err := api.UnsetExtraSpecs(ctx, "st-1", key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}
	manilaSupportUnexpected(t, api.UnsetExtraSpecs(ctx, "st-1", "gone"), 200)
	if !reflect.DeepEqual(unset, []string{"a", "b", "gone"}) {
		t.Fatalf("unset=%v", unset)
	}
}

func TestManilaShareTypeSetExtraSpecsAcceptsOnly200Or202(t *testing.T) {
	for _, tc := range []struct {
		status  int
		success bool
	}{{200, true}, {202, true}, {201, false}, {204, false}} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("POST /v2/types/st-1/extra_specs", func(w http.ResponseWriter, r *http.Request) {
			if tc.status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, tc.status, `{"extra_specs":{"a":"b"}}`)
		})
		_, err := sharetypes.New(newManilaSupportClient(cloud)).SetExtraSpecs(context.Background(), "st-1", sharetypes.SetExtraSpecsOpts{ExtraSpecs: map[string]any{"a": "b"}})
		if tc.success != (err == nil) {
			t.Fatalf("status %d error=%v", tc.status, err)
		}
		if !tc.success {
			manilaSupportUnexpected(t, err, tc.status)
		}
	}
}

func TestManilaShareTypeAccessShowAddRemove(t *testing.T) {
	cloud := testcloud.New(t)
	cloud.Mux.HandleFunc("GET /v2/types/st-1/share_type_access", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		testcloud.JSON(w, 200, `{"share_type_access":[{"share_type_id":"st-1","project_id":"p1"},{"share_type_id":"st-1","project_id":"p2"}]}`)
	})
	var actions []map[string]string
	status := 200
	cloud.Mux.HandleFunc("POST /v2/types/st-1/action", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		var body map[string]map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		for action, fields := range body {
			actions = append(actions, map[string]string{"action": action, "project": string(fields["project"]), "vendor": string(fields["vendor"])})
		}
		if status == 204 {
			w.WriteHeader(204)
			return
		}
		w.WriteHeader(status)
	})
	api := sharetypes.New(newManilaSupportClient(cloud))
	ctx := context.Background()
	access, err := api.ShowAccess(ctx, "st-1")
	if err != nil || !reflect.DeepEqual(access, []sharetypes.ShareTypeAccess{{ShareTypeID: "st-1", ProjectID: "p1"}, {ShareTypeID: "st-1", ProjectID: "p2"}}) {
		t.Fatalf("access=%v error=%v", access, err)
	}
	if err := api.AddAccess(ctx, "st-1", sharetypes.AccessOpts{Project: "p1"}, sharetypes.WithAddAccessField("vendor", 1)); err != nil {
		t.Fatal(err)
	}
	if err := api.RemoveAccess(ctx, "st-1", sharetypes.AccessOpts{}, sharetypes.WithRemoveAccessField("vendor", 2)); err != nil {
		t.Fatal(err)
	}
	want := []map[string]string{
		{"action": "addProjectAccess", "project": `"p1"`, "vendor": `1`},
		{"action": "removeProjectAccess", "project": `""`, "vendor": `2`},
	}
	if !reflect.DeepEqual(actions, want) {
		t.Fatalf("actions=%v", actions)
	}
	status = 202
	if err := api.AddAccess(ctx, "st-1", sharetypes.AccessOpts{Project: "p1"}); err != nil {
		t.Fatalf("202: %v", err)
	}
	for _, code := range []int{201, 204} {
		status = code
		manilaSupportUnexpected(t, api.AddAccess(ctx, "st-1", sharetypes.AccessOpts{Project: "p1"}), code)
		manilaSupportUnexpected(t, api.RemoveAccess(ctx, "st-1", sharetypes.AccessOpts{Project: "p1"}), code)
	}
}

func TestManilaShareTypeListQueriesSinglePageAndCommonLookup(t *testing.T) {
	cloud := testcloud.New(t)
	var queries []url.Values
	cloud.Mux.HandleFunc("GET /v2/types/gold", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("GET /v2/types", func(w http.ResponseWriter, r *http.Request) {
		manilaSupportCheckVersion(t, r)
		queries = append(queries, r.URL.Query())
		// Links are not followed: the upstream pager is single-page.
		testcloud.JSON(w, 200, `{"share_types":[{"id":"st-1","name":"gold"},{"id":"st-2","name":"gold-2"}],"share_types_links":[{"rel":"next","href":"`+cloud.Server.URL+`/v2/types?page=2"}]}`)
	})
	api := sharetypes.New(newManilaSupportClient(cloud))
	ctx := context.Background()
	var ids []string
	for value, err := range api.List(ctx, sharetypes.WithListOptions(sharetypes.ListOpts{IsPublic: "false"}), sharetypes.WithListQuery("vendor", "x")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	if !reflect.DeepEqual(ids, []string{"st-1", "st-2"}) || len(queries) != 1 || queries[0].Get("is_public") != "false" || queries[0].Get("vendor") != "x" {
		t.Fatalf("ids=%v queries=%v", ids, queries)
	}
	queries = nil
	for range api.List(ctx) {
	}
	if len(queries) != 1 || len(queries[0]) != 0 {
		t.Fatalf("unset filter must be omitted: %v", queries)
	}
	value, err := api.Find(ctx, resource.Name("gold"))
	if err != nil || value.ID != "st-1" {
		t.Fatalf("value=%+v error=%v", value, err)
	}
}

func TestManilaShareTypeEmptyAndNoContentListsYieldNothing(t *testing.T) {
	for _, status := range []int{200, 204} {
		cloud := testcloud.New(t)
		cloud.Mux.HandleFunc("GET /v2/types", func(w http.ResponseWriter, r *http.Request) {
			if status == 204 {
				w.WriteHeader(204)
				return
			}
			testcloud.JSON(w, 200, `{"share_types":[]}`)
		})
		for value, err := range sharetypes.New(newManilaSupportClient(cloud)).List(context.Background()) {
			t.Fatalf("status %d value=%+v error=%v", status, value, err)
		}
	}
}

func TestManilaShareTypeCancellationPreservesCauseAndSkipsHTTP(t *testing.T) {
	t.Parallel()
	cloud := testcloud.New(t)
	var calls atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	api := sharetypes.New(newManilaSupportClient(cloud))
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := api.Get(canceled, "st-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled get: %v", err)
	}
	if err := api.Delete(canceled, "st-1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled delete: %v", err)
	}
	if err := api.AddAccess(canceled, "st-1", sharetypes.AccessOpts{Project: "p"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled add access: %v", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("canceled calls reached HTTP: %d", calls.Load())
	}
	deadline, stop := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer stop()
	if _, err := api.Create(deadline, sharetypes.CreateOpts{Name: "gold", ExtraSpecs: sharetypes.ExtraSpecsOpts{DriverHandlesShareServers: true}}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("in-flight deadline: %v", err)
	}
}

// manilaSupportEnvelopeFrom decodes one raw JSON object into raw members.
func manilaSupportEnvelopeFrom(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var inner map[string]json.RawMessage
	if err := json.Unmarshal(raw, &inner); err != nil {
		t.Errorf("object %s: %v", raw, err)
	}
	return inner
}
