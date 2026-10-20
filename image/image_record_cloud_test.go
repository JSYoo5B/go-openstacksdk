package image

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const cloudImageInventoryPage = `{"images":[` +
	`{"id":"u1","name":"ubuntu-22","status":"active","visibility":"public"},` +
	`{"id":"gone","name":"ubuntu-old","status":"DeLeTeD","visibility":"public"},` +
	`{"id":"u2","name":"ubuntu-24","status":"QUEUED","visibility":"private"},` +
	`{"id":"c1","name":"cirros","status":"active","visibility":"public"}],"next":null}`

func cloudImageIDs(t *testing.T, rows []*ImageRecord) []string {
	t.Helper()
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = imageRecordText(row, "id")
	}
	return ids
}

func cloudImageService(t *testing.T, calls *int, pages func(*http.Request) *http.Response) *Service {
	t.Helper()
	cloud := "current"
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		*calls++
		if req.Method != http.MethodGet || req.Body != nil || req.Header.Get("Accept") != "application/json" {
			t.Fatal(req.Method, req.URL, req.Header)
		}
		return pages(req), nil
	})
	return NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
		return resource.CloudLocation{Cloud: &cloud}, nil
	}})
}

func TestCloudImageRecordListDeletedFilterShowAllAndLocation(t *testing.T) {
	for _, test := range []struct {
		name    string
		options []ImageRecordQueryOption
		query   url.Values
		kept    []string
	}{
		{"default filters lowered deleted status", nil, url.Values{}, []string{"u1", "u2", "c1"}},
		{"explicit filter_deleted false keeps every row", []ImageRecordQueryOption{WithImageRecordQueryFilterDeleted(false)}, url.Values{}, []string{"u1", "gone", "u2", "c1"}},
		{"show_all overrides filter_deleted and selects all members", []ImageRecordQueryOption{WithImageRecordQueryFilterDeleted(true), WithImageRecordQueryShowAll(true)}, url.Values{"member_status": {"all"}}, []string{"u1", "gone", "u2", "c1"}},
		{"explicit show_all false keeps default query", []ImageRecordQueryOption{WithImageRecordQueryShowAll(false)}, url.Values{}, []string{"u1", "u2", "c1"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.EscapedPath() != "/reverse/glance/v2/images" || req.URL.Query().Encode() != test.query.Encode() || req.Header.Get("X-Cloud") != "header" {
					t.Fatal(req.URL, req.Header)
				}
				return taskCoreJSON(req, 200, cloudImageInventoryPage)
			})
			options := append([]ImageRecordQueryOption{WithImageRecordQueryHeader("X-Cloud", "header")}, test.options...)
			result, err := service.AllCloudImageRecords(context.Background(), options...)
			if err != nil || calls != 1 || len(result.Inventory) != 4 {
				t.Fatal(result, err, calls)
			}
			if got := cloudImageIDs(t, result.Images); strings.Join(got, ",") != strings.Join(test.kept, ",") {
				t.Fatal(got)
			}
			var values []map[string]json.RawMessage
			if err := json.Unmarshal(result.Value, &values); err != nil || len(values) != len(test.kept) {
				t.Fatal(string(result.Value), err)
			}
			for i, value := range values {
				var location resource.CloudLocation
				if err := json.Unmarshal(value["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != "current" || len(value) != 65 {
					t.Fatal(i, value, err)
				}
				if string(value["id"]) != `"`+test.kept[i]+`"` {
					t.Fatal(i, string(value["id"]))
				}
			}
		})
	}
	t.Run("empty inventory is a completed empty array", func(t *testing.T) {
		calls := 0
		service := cloudImageService(t, &calls, func(req *http.Request) *http.Response { return taskCoreJSON(req, 200, `{"images":[]}`) })
		result, err := service.AllCloudImageRecords(context.Background())
		if err != nil || string(result.Value) != "[]" || result.Images == nil || len(result.Images) != 0 || len(result.Inventory) != 0 {
			t.Fatal(result, err)
		}
	})
}

func TestCloudImageRecordListStatusAndPageFailuresKeepPartialInventory(t *testing.T) {
	for _, status := range []string{`null`, `1`, `["deleted"]`} {
		t.Run("status "+status, func(t *testing.T) {
			page := `{"images":[{"id":"first","status":"active"},{"id":"bad","status":` + status + `},{"id":"later","status":"active"}]}`
			calls := 0
			service := cloudImageService(t, &calls, func(req *http.Request) *http.Response { return taskCoreJSON(req, 203, page) })
			result, err := service.AllCloudImageRecords(context.Background())
			var response *resource.ResponseError
			if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &response) || response.StatusCode != 203 || string(response.Body) != page {
				t.Fatal(err)
			}
			if result == nil || result.Value != nil || result.Images != nil || strings.Join(cloudImageIDs(t, result.Inventory), ",") != "first,bad" {
				t.Fatal(result)
			}
			// filter_deleted false never inspects status.
			result, err = service.AllCloudImageRecords(context.Background(), WithImageRecordQueryFilterDeleted(false))
			if err != nil || len(result.Images) != 3 {
				t.Fatal(result, err)
			}
		})
	}
	t.Run("late page failure", func(t *testing.T) {
		calls := 0
		service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
			if calls == 1 {
				return taskCoreJSON(req, 200, `{"images":[{"id":"first","status":"active"}],"next":"/v2/images?marker=first"}`)
			}
			return taskCoreJSON(req, 403, `{"message":"late"}`)
		})
		result, err := service.AllCloudImageRecords(context.Background())
		if !gophercloud.ResponseCodeIs(err, http.StatusForbidden) || calls != 2 {
			t.Fatal(err, calls)
		}
		if result.Value != nil || result.Images != nil || strings.Join(cloudImageIDs(t, result.Inventory), ",") != "first" {
			t.Fatal(result)
		}
	})
}

func TestCloudImageRecordSearchSelectionAndExpressions(t *testing.T) {
	calls := 0
	service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.RawQuery != "" {
			t.Fatal("search forwards no query", req.URL)
		}
		return taskCoreJSON(req, 200, cloudImageInventoryPage)
	})
	ctx := context.Background()
	for _, test := range []struct {
		name, pattern string
		filters       json.RawMessage
		ids           []string
	}{
		{"all non-deleted rows", "", nil, []string{"u1", "u2", "c1"}},
		{"glob skips deleted rows", "ubuntu-*", nil, []string{"u1", "u2"}},
		{"exact id", "c1", nil, []string{"c1"}},
		{"dictionary after identifier", "ubuntu-*", json.RawMessage(`{"visibility":"public"}`), []string{"u1"}},
		{"falsey filters keep identifier selection", "ubuntu-*", json.RawMessage(`{}`), []string{"u1", "u2"}},
		{"nested location dictionary", "", json.RawMessage(`{"location":{"cloud":"current"},"name":"cirros"}`), []string{"c1"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls = 0
			options := []ImageRecordQueryOption{}
			if test.filters != nil {
				options = append(options, WithImageRecordQueryFilters(test.filters))
			}
			result, err := service.SearchImageRecords(ctx, test.pattern, options...)
			if err != nil || calls != 1 || len(result.Inventory) != 4 || strings.Join(cloudImageIDs(t, result.Images), ",") != strings.Join(test.ids, ",") {
				t.Fatal(result, err, calls)
			}
		})
	}
	t.Run("expression returns arbitrary JSON without Images", func(t *testing.T) {
		result, err := service.SearchImageRecords(ctx, "ubuntu-*", WithImageRecordQueryExpression("[].id"))
		if err != nil || string(result.Value) != `["u1","u2"]` || result.Images != nil || len(result.Inventory) != 4 {
			t.Fatal(result, err)
		}
	})
	t.Run("declared null attribute matches before unknown key fails", func(t *testing.T) {
		calls = 0
		result, err := service.SearchImageRecords(ctx, "", WithImageRecordQueryFilters(json.RawMessage(`{"os_distro":null,"vendor_unknown":"x"}`)))
		if !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || result.Value != nil || result.Images != nil || len(result.Inventory) != 4 {
			t.Fatal(result, err)
		}
	})
}

func TestCloudImageRecordArgumentsFailBeforeHTTP(t *testing.T) {
	calls := 0
	service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
		t.Fatal("unexpected request", req.URL)
		return nil
	})
	ctx := context.Background()
	filters := []ImageRecordQueryOption{WithImageRecordQueryFilters(json.RawMessage(`null`))}
	listControls := []ImageRecordQueryOption{WithImageRecordQueryShowAll(false)}
	deleted := []ImageRecordQueryOption{WithImageRecordQueryFilterDeleted(true)}
	invalidJSON := []ImageRecordQueryOption{WithImageRecordQueryFilters(json.RawMessage(`{`))}
	checks := []struct {
		name string
		call func() error
	}{
		{"list rejects explicit null filters", func() error { _, err := service.AllCloudImageRecords(ctx, filters...); return err }},
		{"list rejects malformed filters", func() error { _, err := service.AllCloudImageRecords(ctx, invalidJSON...); return err }},
		{"search rejects show_all", func() error { _, err := service.SearchImageRecords(ctx, "", listControls...); return err }},
		{"search rejects malformed filters", func() error { _, err := service.SearchImageRecords(ctx, "", invalidJSON...); return err }},
		{"get rejects filter_deleted", func() error { _, err := service.GetCloudImageRecord(ctx, "image", deleted...); return err }},
		{"get rejects blank find identity", func() error { _, err := service.GetCloudImageRecord(ctx, " "); return err }},
		{"by ID rejects filters", func() error { _, err := service.GetImageRecordByID(ctx, "image", filters...); return err }},
		{"by ID rejects show_all", func() error { _, err := service.GetImageRecordByID(ctx, "image", listControls...); return err }},
		{"by ID rejects blank ID", func() error { _, err := service.GetImageRecordByID(ctx, ""); return err }},
		{"nil option", func() error { _, err := service.SearchImageRecords(ctx, "", nil); return err }},
		{"invalid header", func() error {
			_, err := service.AllCloudImageRecords(ctx, WithImageRecordQueryHeader("X-Bad", "line\nbreak"))
			return err
		}},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.call(); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(err, calls)
			}
		})
	}
	t.Run("clearing filters with nil restores the list argument set", func(t *testing.T) {
		service := cloudImageService(t, &calls, func(req *http.Request) *http.Response { return taskCoreJSON(req, 200, `{"images":[]}`) })
		if _, err := service.AllCloudImageRecords(ctx, WithImageRecordQueryFilters(json.RawMessage(`{}`)), WithImageRecordQueryFilters(nil)); err != nil || calls != 1 {
			t.Fatal(err, calls)
		}
	})
}

func TestCloudImageRecordOptionsCancelBeforeLaterCallbacks(t *testing.T) {
	calls := 0
	service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
		t.Fatal("unexpected request", req.URL)
		return nil
	})
	cause := errors.New("caller stopped")
	for _, call := range []func(context.Context, ...ImageRecordQueryOption) error{
		func(ctx context.Context, options ...ImageRecordQueryOption) error {
			_, err := service.AllCloudImageRecords(ctx, options...)
			return err
		},
		func(ctx context.Context, options ...ImageRecordQueryOption) error {
			_, err := service.GetImageRecordByID(ctx, "image", options...)
			return err
		},
	} {
		ctx, cancel := context.WithCancelCause(context.Background())
		later := 0
		err := call(ctx, func(*ImageRecordQueryOpts) error { cancel(cause); return nil }, func(*ImageRecordQueryOpts) error { later++; return nil })
		if !errors.Is(err, cause) || later != 0 || calls != 0 {
			t.Fatal(err, later, calls)
		}
	}
}

func TestCloudImageRecordGetFindPathAndMissing(t *testing.T) {
	for _, filters := range []json.RawMessage{nil, json.RawMessage(` null `)} {
		calls := 0
		service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
			switch calls {
			case 1:
				if req.URL.EscapedPath() != "/reverse/glance/v2/images/ubuntu" || req.URL.RawQuery != "" || req.Header.Get("X-Cloud") != "header" {
					t.Fatal(req.URL, req.Header)
				}
				return taskCoreJSON(req, 404, `{"message":"missing"}`)
			case 2:
				if req.URL.Query().Get("name") != "ubuntu" || req.Header.Get("X-Cloud") != "header" {
					t.Fatal(req.URL, req.Header)
				}
				return taskCoreJSON(req, 200, `{"images":[{"id":"found","name":"ubuntu","status":"deleted"}]}`)
			}
			t.Fatal("unexpected request", req.URL)
			return nil
		})
		options := []ImageRecordQueryOption{WithImageRecordQueryHeader("X-Cloud", "header")}
		if filters != nil {
			options = append(options, WithImageRecordQueryFilters(filters))
		}
		result, err := service.GetCloudImageRecord(context.Background(), "ubuntu", options...)
		// Proxy find has no Cloud deleted-status filter.
		if err != nil || calls != 2 || result.Image == nil || imageRecordText(result.Image, "id") != "found" || result.Inventory != nil {
			t.Fatal(result, err, calls)
		}
		var value map[string]json.RawMessage
		if err := json.Unmarshal(result.Value, &value); err != nil || string(value["id"]) != `"found"` || len(value) != 65 {
			t.Fatal(string(result.Value), err)
		}
	}
	t.Run("ignore missing returns an empty result", func(t *testing.T) {
		calls := 0
		service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
			if calls == 1 {
				return taskCoreJSON(req, 404, `{}`)
			}
			return taskCoreJSON(req, 200, `{"images":[]}`)
		})
		result, err := service.GetCloudImageRecord(context.Background(), "absent")
		if err != nil || calls != 3 || result == nil || result.Image != nil || result.Value != nil {
			t.Fatal(result, err, calls)
		}
	})
	t.Run("terminal direct failure", func(t *testing.T) {
		calls := 0
		service := cloudImageService(t, &calls, func(req *http.Request) *http.Response { return taskCoreJSON(req, 500, `{"message":"server"}`) })
		result, err := service.GetCloudImageRecord(context.Background(), "image")
		if result != nil || !gophercloud.ResponseCodeIs(err, http.StatusInternalServerError) || calls != 1 {
			t.Fatal(result, err, calls)
		}
	})
}

func TestCloudImageRecordGetFilteredSelection(t *testing.T) {
	calls := 0
	service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.EscapedPath() != "/reverse/glance/v2/images" || req.URL.RawQuery != "" {
			t.Fatal("filtered get uses search, not find", req.URL)
		}
		return taskCoreJSON(req, 200, cloudImageInventoryPage)
	})
	ctx := context.Background()
	for _, test := range []struct {
		name, pattern string
		filters       json.RawMessage
		value, id     string
		length        int
	}{
		{"falsey dictionary still searches", "cirros", json.RawMessage(`{}`), "", "c1", 0},
		{"false filter still searches", "c1", json.RawMessage(`false`), "", "c1", 0},
		{"dictionary unique", "ubuntu-*", json.RawMessage(`{"status":"QUEUED"}`), "", "u2", 0},
		{"deleted row is not selectable", "ubuntu-old", json.RawMessage(`{}`), "", "", 0},
		{"dictionary ambiguous", "ubuntu-*", json.RawMessage(`{}`), "", "", 2},
		{"expression scalar string has length", "", json.RawMessage(`"[0].name"`), "", "", 9},
		{"expression one character string", "c1", json.RawMessage(`"[0].id"`), "", "", 2},
		{"expression singleton keeps raw value without Image", "c1", json.RawMessage(`"[].visibility"`), `"public"`, "", 0},
		{"expression empty array is missing", "absent", json.RawMessage(`"[].id"`), "", "", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls = 0
			result, err := service.GetCloudImageRecord(ctx, test.pattern, WithImageRecordQueryFilters(test.filters))
			if calls != 1 || result == nil || len(result.Inventory) != 4 {
				t.Fatal(result, err, calls)
			}
			if test.length != 0 {
				var selection *ImageRecordSelectionError
				if !errors.Is(err, resource.ErrAmbiguous) || !errors.As(err, &selection) || selection.Length != test.length || selection.NameOrID != test.pattern || result.Value != nil || result.Image != nil {
					t.Fatal(result, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch {
			case test.id != "":
				if result.Image == nil || imageRecordText(result.Image, "id") != test.id || !strings.Contains(string(result.Value), `"id":"`+test.id+`"`) {
					t.Fatal(result)
				}
			case test.value != "":
				if result.Image != nil || string(result.Value) != test.value {
					t.Fatal(result, string(result.Value))
				}
			default:
				if result.Image != nil || result.Value != nil {
					t.Fatal(result)
				}
			}
		})
	}
	t.Run("expression object has no index zero", func(t *testing.T) {
		result, err := service.GetCloudImageRecord(ctx, "c1", WithImageRecordQueryExpression("{a: [0].id}"))
		if !errors.Is(err, resource.ErrInvalidOption) || errors.Is(err, resource.ErrAmbiguous) || result.Value != nil {
			t.Fatal(result, err)
		}
	})
}

func TestCloudImageRecordGetByIDIsStrictLiteralGet(t *testing.T) {
	calls := 0
	service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.EscapedPath() != "/reverse/glance/v2/images/"+url.PathEscape("literal/空") || req.URL.RawQuery != "" || req.Header.Get("X-Cloud") != "header" {
			t.Fatal(req.URL, req.Header)
		}
		if calls == 1 {
			return taskCoreJSON(req, 200, `{"id":"literal/空","name":"n","status":"deleted"}`)
		}
		return taskCoreJSON(req, 404, `{"message":"missing"}`)
	})
	record, err := service.GetImageRecordByID(context.Background(), "literal/空", WithImageRecordQueryHeaders(map[string]string{"X-Cloud": "header"}))
	if err != nil || calls != 1 || imageRecordText(record, "id") != "literal/空" || imageRecordText(record, "status") != "deleted" {
		t.Fatal(record, err)
	}
	var location resource.CloudLocation
	if err := json.Unmarshal(record.Resource.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != "current" {
		t.Fatal(location, err)
	}
	record, err = service.GetImageRecordByID(context.Background(), "literal/空", WithImageRecordQueryHeader("X-Cloud", "header"))
	var operation *resource.OperationError
	if record != nil || calls != 2 || !gophercloud.ResponseCodeIs(err, http.StatusNotFound) || !errors.As(err, &operation) || operation.Operation != "GetImageRecordByID" {
		t.Fatal(record, err, calls)
	}
	if errors.As(operation.Cause, &operation) {
		t.Fatal("delegated operation is renamed, not nested", err)
	}
}

func TestCloudImageRecordExcludeMembershipOrderAndValues(t *testing.T) {
	const page = `{"images":[` +
		`{"id":"gone","name":"ubuntu-gone","status":"deleted"},` +
		`{"id":"u1","name":"ubuntu-22-test","status":"active"},` +
		`{"id":"u2","name":["ubuntu-24","test"],"status":"active"},` +
		`{"id":"u3","name":{"test":1},"status":"active"},` +
		`{"id":"u4","name":"ubuntu-24","status":"active"},` +
		`{"id":7,"name":null,"status":"active"}]}`
	calls := 0
	service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.RawQuery != "" || req.Header.Get("X-Cloud") != "header" {
			t.Fatal(req.URL, req.Header)
		}
		return taskCoreJSON(req, 200, page)
	})
	ctx := context.Background()
	header := WithImageRecordQueryHeader("X-Cloud", "header")
	for _, test := range []struct {
		name, pattern, exclude, id string
	}{
		{"empty exclude returns first non-deleted search row", "ubuntu*", "", "u1"},
		{"substring skips string name", "ubuntu*", "test", "u4"},
		{"list name uses element equality", "u2", "ubuntu", "u2"},
		{"list name element match is skipped", "u[23]", "test", ""},
		{"dictionary name uses key membership", "u3", "tes", "u3"},
		{"deleted rows are not candidates", "ubuntu-gone", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls = 0
			result, err := service.GetImageRecordExclude(ctx, test.pattern, test.exclude, header)
			if err != nil || calls != 1 || len(result.Inventory) != 6 {
				t.Fatal(result, err, calls)
			}
			if test.id == "" {
				if result.Image != nil || result.Value != nil {
					t.Fatal(result)
				}
				return
			}
			if result.Image == nil || imageRecordText(result.Image, "id") != test.id || !strings.Contains(string(result.Value), `"id":"`+test.id+`"`) {
				t.Fatal(result, string(result.Value))
			}
		})
	}
	t.Run("name and id return raw selected values", func(t *testing.T) {
		name, err := service.GetImageRecordName(ctx, "u2", "", header)
		if err != nil || string(name.Value) != `["ubuntu-24","test"]` || imageRecordText(name.Image, "id") != "u2" {
			t.Fatal(name, err)
		}
		id, err := service.GetImageRecordID(ctx, "ubuntu-24", "", header)
		if err != nil || string(id.Value) != `"u4"` {
			t.Fatal(id, err)
		}
		// The identifier phase matches str(7) and a null name is a present value.
		name, err = service.GetImageRecordName(ctx, "7", "", header)
		if err != nil || string(name.Value) != "null" || name.Image == nil {
			t.Fatal(name, err)
		}
		id, err = service.GetImageRecordID(ctx, "7", "", header)
		if err != nil || string(id.Value) != "7" {
			t.Fatal(id, err)
		}
		missing, err := service.GetImageRecordID(ctx, "absent", "", header)
		if err != nil || missing.Value != nil || missing.Image != nil || len(missing.Inventory) != 6 {
			t.Fatal(missing, err)
		}
	})
	t.Run("reached non-container name is a type error with receipt", func(t *testing.T) {
		result, err := service.GetImageRecordName(ctx, "7", "x", header)
		var response *resource.ResponseError
		if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &response) || string(response.Body) != page || result.Image != nil || result.Value != nil || len(result.Inventory) != 6 {
			t.Fatal(result, err)
		}
	})
	t.Run("unreached non-container name is not inspected", func(t *testing.T) {
		result, err := service.GetImageRecordID(ctx, "*", "nothing", header)
		if err != nil || string(result.Value) != `"u1"` {
			t.Fatal(result, err)
		}
	})
}

func TestCloudImageRecordExcludeArgumentsFailBeforeHTTP(t *testing.T) {
	calls := 0
	service := cloudImageService(t, &calls, func(req *http.Request) *http.Response {
		t.Fatal("unexpected request", req.URL)
		return nil
	})
	ctx := context.Background()
	for name, call := range map[string]func() error{
		"exclude rejects filters": func() error {
			_, err := service.GetImageRecordExclude(ctx, "", "", WithImageRecordQueryFilters(json.RawMessage(`{}`)))
			return err
		},
		"name rejects show_all": func() error {
			_, err := service.GetImageRecordName(ctx, "", "", WithImageRecordQueryShowAll(true))
			return err
		},
		"id rejects filter_deleted": func() error {
			_, err := service.GetImageRecordID(ctx, "", "", WithImageRecordQueryFilterDeleted(false))
			return err
		},
		"invalid UTF-8 exclude": func() error {
			_, err := service.GetImageRecordID(ctx, "", "\xff")
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := call(); !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(err, calls)
			}
		})
	}
}
