package image

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func taskPythonIDs(t *testing.T, service *Service, options ...ListTasksOption) ([]string, error) {
	t.Helper()
	var ids []string
	for value, err := range service.Tasks(context.Background(), options...) {
		if err != nil {
			return ids, err
		}
		id := "<none>"
		if value.ID != nil {
			id = *value.ID
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func TestTaskPythonBodyFiltersStayLocalAndCountRawRows(t *testing.T) {
	const page = `{"tasks":[{"id":"a","owner":"p1","input":{"x":1,"y":2},"message":""},{"id":"b","owner":"p2","input":{"x":1}},{"id":"c","owner":"p1","input":{"x":2}},{"id":"d","owner":"p1","input":{"x":1}}]}`
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.RawQuery != "" {
			t.Errorf("body filters must not reach the wire: %s", req.URL.RawQuery)
		}
		return taskCoreJSON(req, 200, page), nil
	})
	service := New(client)
	input := map[string]any{"x": 1}
	filter := WithListTasksBodyFilter("input", input)
	input["x"] = 2
	ids, err := taskPythonIDs(t, service, WithListTasksBodyFilter("owner_id", "p2"), WithListTasksBodyFilter("owner_id", "p1"), filter)
	if err != nil || !reflect.DeepEqual(ids, []string{"a", "d"}) {
		t.Fatalf("ids=%v err=%v", ids, err)
	}
	// Python counts excluded rows toward max_items: a and b are consumed, only a matches.
	ids, err = taskPythonIDs(t, service, WithListTasksBodyFilter("owner_id", "p1"), WithListTasksMaxItems(2))
	if err != nil || !reflect.DeepEqual(ids, []string{"a"}) {
		t.Fatalf("raw cap ids=%v err=%v", ids, err)
	}
	// A missing field compares as null and an empty string is not null.
	ids, err = taskPythonIDs(t, service, WithListTasksBodyFilter("message", nil))
	if err != nil || !reflect.DeepEqual(ids, []string{"b", "c", "d"}) {
		t.Fatalf("null ids=%v err=%v", ids, err)
	}
	// Replacing all options clears earlier filters.
	ids, err = taskPythonIDs(t, service, WithListTasksBodyFilter("owner_id", "none"), WithListTasksOpts(ListTasksOpts{}))
	if err != nil || len(ids) != 4 || calls != 4 {
		t.Fatalf("cleared ids=%v err=%v calls=%d", ids, err, calls)
	}
	for name, option := range map[string]ListTasksOption{
		"wire alias":        WithListTasksBodyFilter("owner", "p1"),
		"query attribute":   WithListTasksBodyFilter("type", "import"),
		"unknown attribute": WithListTasksBodyFilter("vendor", 1),
		"unserializable":    WithListTasksBodyFilter("id", func() {}),
		"raw invalid JSON":  WithListTasksOpts(ListTasksOpts{BodyFilters: map[string]json.RawMessage{"id": json.RawMessage("{")}}),
		"raw unknown name":  WithListTasksOpts(ListTasksOpts{BodyFilters: map[string]json.RawMessage{"owner": json.RawMessage(`"p1"`)}}),
	} {
		if _, err := service.AllTasks(context.Background(), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if calls != 4 {
		t.Fatalf("invalid filters made HTTP requests: %d", calls)
	}
}

func TestTaskPythonMarkerFallbackFollowsLimitAndLastRawRow(t *testing.T) {
	pages := map[string]string{
		"":  `{"tasks":[{"id":"a","owner":"p1"},{"id":"b","owner":"p2"}]}`,
		"b": `{"tasks":[{"id":"c","owner":"p1"}]}`,
		"c": `{"tasks":[]}`,
	}
	var requested []string
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		query := req.URL.Query()
		requested = append(requested, req.URL.RawQuery)
		if query.Get("limit") != "2" || query.Get("type") != "import" || len(query["marker"]) > 1 {
			t.Errorf("fallback changed the collection query: %s", req.URL.RawQuery)
		}
		return taskCoreJSON(req, 200, pages[query.Get("marker")]), nil
	})
	service := New(client)
	ids, err := taskPythonIDs(t, service, WithListTasksLimit(2), WithListTasksType("import"), WithListTasksMarkerFallback(true), WithListTasksBodyFilter("owner_id", "p1"))
	want := []string{"limit=2&type=import", "limit=2&marker=b&type=import", "limit=2&marker=c&type=import"}
	if err != nil || !reflect.DeepEqual(ids, []string{"a", "c"}) || !reflect.DeepEqual(requested, want) {
		t.Fatalf("ids=%v err=%v requested=%v", ids, err, requested)
	}
	for name, options := range map[string][]ListTasksOption{
		"disabled by default": {WithListTasksLimit(2), WithListTasksType("import")},
		"last option wins":    {WithListTasksLimit(2), WithListTasksType("import"), WithListTasksMarkerFallback(true), WithListTasksMarkerFallback(false)},
		"single page":         {WithListTasksLimit(2), WithListTasksType("import"), WithListTasksMarkerFallback(true), WithListTasksSinglePage(true)},
		"max items":           {WithListTasksLimit(2), WithListTasksType("import"), WithListTasksMarkerFallback(true), WithListTasksMaxItems(2)},
	} {
		requested = nil
		ids, err := taskPythonIDs(t, service, options...)
		if err != nil || !reflect.DeepEqual(ids, []string{"a", "b"}) || len(requested) != 1 {
			t.Fatalf("%s: ids=%v err=%v requested=%v", name, ids, err, requested)
		}
	}
}

func TestTaskPythonMarkerFallbackGuards(t *testing.T) {
	for _, tc := range []struct {
		name, first, second string
		limit               int
		wantCalls           int
		check               func(error) bool
	}{
		{"no positive limit", `{"tasks":[{"id":"a"}]}`, ``, 0, 1, func(err error) bool { return err == nil }},
		{"body next wins", `{"tasks":[{"id":"a"}],"next":"/v2/tasks?limit=1&marker=from-body"}`, `{"tasks":[]}`, 1, 2, func(err error) bool { return err == nil }},
		{"repeated marker", `{"tasks":[{"id":"a"}]}`, `{"tasks":[{"id":"a"}]}`, 1, 2, func(err error) bool {
			var cycle *resource.PaginationCycleError
			return errors.As(err, &cycle)
		}},
		{"last row without id", `{"tasks":[{"id":"a"},{"status":"pending"}]}`, ``, 2, 1, func(err error) bool { return errors.Is(err, resource.ErrInvalidOption) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return taskCoreJSON(req, 200, tc.first), nil
				}
				if tc.name == "body next wins" && req.URL.Query().Get("marker") != "from-body" {
					t.Errorf("fallback replaced the body next link: %s", req.URL)
				}
				return taskCoreJSON(req, 200, tc.second), nil
			})
			options := []ListTasksOption{WithListTasksMarkerFallback(true)}
			if tc.limit > 0 {
				options = append(options, WithListTasksLimit(tc.limit))
			}
			_, err := taskPythonIDs(t, New(client), options...)
			if !tc.check(err) || calls != tc.wantCalls {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
		})
	}
}

func TestTaskPythonLinkHeaderContinuation(t *testing.T) {
	newClient := func(link string, calls *int, requested *[]url.Values) *Service {
		return New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
			*calls++
			*requested = append(*requested, req.URL.Query())
			if *calls == 1 {
				response := taskCoreJSON(req, 200, `{"tasks":[{"id":"one"}]}`)
				response.Header.Set("Link", link)
				return response, nil
			}
			return taskCoreJSON(req, 200, `{"tasks":[{"id":"two"}]}`), nil
		}))
	}
	const valid = `<https://glance.example/other>; rel="prev", <https://glance.example/reverse/glance/v2/tasks?limit=1&marker=one>; rel="next"`
	calls, requested := 0, []url.Values(nil)
	ids, err := taskPythonIDs(t, newClient(valid, &calls, &requested), WithListTasksLimit(1), WithListTasksLinkHeader(true))
	if err != nil || !reflect.DeepEqual(ids, []string{"one", "two"}) || calls != 2 || requested[1].Get("marker") != "one" || requested[1].Get("limit") != "1" {
		t.Fatalf("ids=%v err=%v calls=%d requested=%v", ids, err, calls, requested)
	}
	calls, requested = 0, nil
	ids, err = taskPythonIDs(t, newClient(valid, &calls, &requested), WithListTasksLimit(1))
	if err != nil || !reflect.DeepEqual(ids, []string{"one"}) || calls != 1 {
		t.Fatalf("disabled: ids=%v err=%v calls=%d", ids, err, calls)
	}
	for name, link := range map[string]string{
		"foreign origin":  `<https://foreign.invalid/reverse/glance/v2/tasks?limit=1&marker=one>; rel="next"`,
		"dropped filter":  `<https://glance.example/reverse/glance/v2/tasks?marker=one>; rel="next"`,
		"malformed value": `https://glance.example/reverse/glance/v2/tasks?marker=one; rel=next`,
	} {
		calls, requested = 0, nil
		_, err := taskPythonIDs(t, newClient(link, &calls, &requested), WithListTasksLimit(1), WithListTasksLinkHeader(true))
		var response *resource.ResponseError
		if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &response) || response.StatusCode != 200 || calls != 1 {
			t.Fatalf("%s: err=%v calls=%d", name, err, calls)
		}
		calls, requested = 0, nil
		if _, err := taskPythonIDs(t, newClient(link, &calls, &requested), WithListTasksLimit(1)); err != nil || calls != 1 {
			t.Fatalf("%s disabled: err=%v calls=%d", name, err, calls)
		}
	}
}
