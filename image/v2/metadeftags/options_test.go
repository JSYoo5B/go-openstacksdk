package metadeftags

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"gophercloudsdk/resource"
)

func tagOptionPointer[T any](value T) *T { return &value }

func TestMetadefTagsOptionsSnapshotsReplacementAndLastWins(t *testing.T) {
	input := SetOpts{Headers: map[string]string{"X-Initial": "snapshot"}, Append: tagOptionPointer(true)}
	helper := WithSetOpts(input)
	input.Headers["X-Initial"] = "mutated"
	*input.Append = false
	calls := 0
	client := tagCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1, 2:
			if req.Header.Get("X-Initial") != "snapshot" || req.Header.Get("X-Openstack-Append") != "True" {
				t.Fatalf("factory snapshot %v", req.Header)
			}
		case 3:
			if req.Header.Get("X-Initial") != "" || req.Header.Get("X-Replaced") != "replacement" || req.Header.Get("X-Openstack-Append") != "False" {
				t.Fatalf("whole replacement %v", req.Header)
			}
		case 4:
			if req.Header.Get("X-Header") != "last" || req.Header.Get("X-Map") != "owned" || req.Header.Get("X-Openstack-Append") != "False" {
				t.Fatalf("last wins %v", req.Header)
			}
		}
		return tagCoreJSON(req, 201, `{"tags":[]}`), nil
	})
	scope := tagCoreScope(t, client, "parent")
	for range 2 {
		if _, err := scope.Set(context.Background(), nil, helper); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := scope.Set(context.Background(), nil, helper, WithSetOpts(SetOpts{Headers: map[string]string{"X-Replaced": "replacement"}})); err != nil {
		t.Fatal(err)
	}
	headerInput := map[string]string{"X-Map": "owned"}
	headers := WithSetHeaders(headerInput)
	headerInput["X-Map"] = "changed"
	if _, err := scope.Set(context.Background(), nil, helper, WithSetHeader("X-header", "first"), WithSetHeader("x-HEADER", "last"), headers, WithSetAppend(false)); err != nil {
		t.Fatal(err)
	}
	if calls != 4 {
		t.Fatalf("reuse callbacks %d", calls)
	}
	name := tagOptionPointer("snapshot-name")
	update := WithUpdateOpts(UpdateOpts{Name: name, Headers: map[string]string{"X-Update": "snapshot"}})
	*name = "caller-change"
	config, err := prepareUpdate([]UpdateOption{update, WithUpdateName("last-name"), WithUpdateHeader("x-update", "last")})
	if err != nil || *config.Name != "last-name" || config.Headers["X-Update"] != "last" {
		t.Fatalf("update last-wins %v %+v", err, config)
	}
	listInput := ListOpts{Headers: map[string]string{"X-List": "owned"}, Limit: tagOptionPointer(2), Marker: tagOptionPointer("original"), SortKey: tagOptionPointer("name"), SortDir: tagOptionPointer("asc"), MaxItems: 3}
	listHelper := WithListOpts(listInput)
	*listInput.Limit = 99
	*listInput.Marker = "changed"
	*listInput.SortKey = "changed"
	*listInput.SortDir = "desc"
	listInput.Headers["X-List"] = "changed"
	policy, query, err := prepareList([]ListOption{listHelper, WithListMarker("last"), WithListMaxItems(1)})
	if err != nil || query.Get("limit") != "2" || query.Get("marker") != "last" || query.Get("sort_key") != "name" || query.Get("sort_dir") != "asc" || policy.MaxItems != 1 || policy.Headers["X-List"] != "owned" {
		t.Fatalf("list snapshot %v %v %+v", err, query, policy)
	}
}

func TestMetadefTagsOptionsCallbackAndBorrowedOwnership(t *testing.T) {
	names := []string{"original", "original"}
	callbacks := 0
	var retained *SetOpts
	var retainedMap map[string]string
	client := tagCoreClient(func(req *http.Request) (*http.Response, error) {
		var body struct {
			Tags []map[string]string `json:"tags"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(body.Tags, []map[string]string{{"name": "original"}, {"name": "original"}}) || req.Header.Get("X-Option") != "owned" || req.Header.Get("X-Openstack-Append") != "True" {
			t.Fatalf("owned callback arguments %v %v", body.Tags, req.Header)
		}
		return tagCoreJSON(req, 201, `{"tags":[]}`), nil
	})
	scope := tagCoreScope(t, client, "parent")
	first := SetOption(func(c *SetOpts) error {
		callbacks++
		if c.Headers == nil {
			t.Fatal("headers not initialized")
		}
		c.Headers["X-Option"] = "owned"
		c.Append = tagOptionPointer(true)
		retained = c
		retainedMap = c.Headers
		names[0] = string([]byte{255})
		return nil
	})
	second := SetOption(func(c *SetOpts) error {
		callbacks++
		retained.Headers["X-Option"] = "mutated"
		*retained.Append = false
		if c.Headers["X-Option"] != "owned" || !*c.Append {
			t.Fatal("callback copies share borrowed pointers")
		}
		return nil
	})
	if _, err := scope.Set(context.Background(), names, first, second); err != nil || callbacks != 2 {
		t.Fatalf("callbacks once %v %d", err, callbacks)
	}
	// Handles are no longer being used by the completed operation.
	retainedMap["X-Option"] = "after-completion"
	*retained.Append = false
	var prior *ListOpts
	count := 0
	policy, query, err := prepareList([]ListOption{func(c *ListOpts) error {
		count++
		c.Limit = tagOptionPointer(2)
		c.Marker = tagOptionPointer("owned")
		c.SortKey = tagOptionPointer("name")
		c.SortDir = tagOptionPointer("asc")
		c.Headers["X-List"] = "owned"
		prior = c
		return nil
	}, func(c *ListOpts) error {
		count++
		*prior.Limit = 0
		*prior.Marker = "bad\n"
		*prior.SortKey = "bad\n"
		*prior.SortDir = "bad"
		prior.Headers["X-List"] = "bad"
		return nil
	}})
	if err != nil || count != 2 || query.Get("limit") != "2" || query.Get("marker") != "owned" || query.Get("sort_key") != "name" || query.Get("sort_dir") != "asc" || policy.Headers["X-List"] != "owned" {
		t.Fatalf("owned list callbacks %v %v", err, query)
	}
	sentinel := errors.New("callback error")
	_, err = scope.Set(context.Background(), nil, func(*SetOpts) error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("callback error identity %v", err)
	}
}

func TestMetadefTagsOptionsValidationAndDefaults(t *testing.T) {
	cases := []struct {
		name    string
		options []ListOption
	}{
		{"negative limit", []ListOption{WithListLimit(-1)}},
		{"negative cap", []ListOption{WithListMaxItems(-1)}},
		{"marker control", []ListOption{WithListMarker("x\n")}},
		{"marker utf8", []ListOption{WithListMarker(string([]byte{255}))}},
		{"key control", []ListOption{WithListSortKey("x\x00")}},
		{"key utf8", []ListOption{WithListSortKey(string([]byte{255}))}},
		{"direction", []ListOption{WithListSortDir("ASC")}},
		{"empty direction", []ListOption{WithListSortDir("")}},
		{"nil", []ListOption{nil}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := prepareList(tc.options); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("invalid list %v", err)
			}
		})
	}
	defaults, query, err := prepareList(nil)
	if err != nil || defaults.Limit != nil || defaults.Marker != nil || defaults.SortKey != nil || defaults.SortDir != nil || defaults.MaxItems != 0 || len(query) != 0 || defaults.Headers == nil {
		t.Fatalf("nil defaults %v %+v %v", err, defaults, query)
	}
	policy, query, err := prepareList([]ListOption{WithListLimit(0), WithListMarker(""), WithListSortKey(""), WithListMaxItems(20)})
	if err != nil || query.Encode() != "limit=0&marker=&sort_key=" || policy.MaxItems != 20 {
		t.Fatalf("explicit values %v %v", err, query)
	}
	marker := strings.Repeat("界", 81) + "/?%# &"
	_, query, err = prepareList([]ListOption{WithListMarker(marker), WithListSortKey("custom-db-field")})
	if err != nil || query.Get("marker") != marker || query.Get("sort_key") != "custom-db-field" {
		t.Fatalf("query literals %v %v", err, query)
	}
	set, err := prepareSet(nil)
	if err != nil || set.Append != nil || set.Headers == nil {
		t.Fatalf("Set defaults %v %+v", err, set)
	}
	update, err := prepareUpdate(nil)
	if err != nil || update.Name != nil {
		t.Fatalf("Update nil name %v", err)
	}
	rejectedHeaders := []map[string]string{{"X-Openstack-Append": "True"}, {"x-OPENSTACK-APPEND": "False"}, {"X-A": "one", "x-a": "two"}, {"Accept": "application/json"}, {"Content-Type": "application/json"}, {"Authorization": "Bearer token"}, {"X-Auth-Token": "other"}, {"Host": "foreign.test"}, {"Content-Length": "0"}, {"OpenStack-API-Version": "image 2.9"}, {"Bad Key": "value"}, {"X-Good": "value\n"}}
	for _, headers := range rejectedHeaders {
		checks := []func() error{
			func() error {
				_, e := prepareCreate([]CreateOption{WithCreateOpts(CreateOpts{Headers: headers})})
				return e
			},
			func() error { _, e := prepareGet([]GetOption{WithGetHeaders(headers)}); return e },
			func() error {
				_, e := prepareUpdate([]UpdateOption{WithUpdateOpts(UpdateOpts{Headers: headers})})
				return e
			},
			func() error { _, e := prepareDelete([]DeleteOption{WithDeleteHeaders(headers)}); return e },
			func() error {
				_, e := prepareDeleteAll([]DeleteAllOption{WithDeleteAllOpts(DeleteAllOpts{Headers: headers})})
				return e
			},
			func() error { _, e := prepareSet([]SetOption{WithSetHeaders(headers)}); return e },
			func() error { _, _, e := prepareList([]ListOption{WithListOpts(ListOpts{Headers: headers})}); return e },
		}
		for index, check := range checks {
			if err := check(); !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatalf("headers family%d %#v %v", index, headers, err)
			}
		}
	}
	// Source representation and matching selected version remain valid ordinary policy.
	headers, err := validateHeaders(map[string]string{"Accept": "application/custom", "Content-Type": "application/custom", "OpenStack-API-Version": "image 2.9"}, true, "2.9")
	if err != nil || headers["Openstack-Api-Version"] != "image 2.9" {
		t.Fatalf("source version %v %v", err, headers)
	}
}

func TestMetadefTagsOptionsParallelReusableHelpers(t *testing.T) {
	var calls, callbacks atomic.Int64
	headers := map[string]string{"X-Reusable": "original"}
	appendValue := true
	helper := WithSetOpts(SetOpts{Headers: headers, Append: &appendValue})
	headers["X-Reusable"] = "caller-change"
	appendValue = false
	client := tagCoreClient(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		if req.Header.Get("X-Reusable") != "original" || req.Header.Get("X-Openstack-Append") != "True" {
			t.Errorf("parallel owned fields %v", req.Header)
		}
		return tagCoreJSON(req, 201, `{"tags":[{"name":"literal"}]}`), nil
	})
	scope := tagCoreScope(t, client, "parent")
	option := SetOption(func(c *SetOpts) error { callbacks.Add(1); c.Headers["X-Per-Call"] = "owned"; return nil })
	var wait sync.WaitGroup
	for range 24 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := scope.Set(context.Background(), []string{"literal"}, helper, option)
			if err != nil || result == nil || len(result.Tags) != 1 || *result.Tags[0].Name != "literal" {
				t.Errorf("parallel result %v %+v", err, result)
				return
			}
			*result.Tags[0].Name = "local"
			result.Tags[0].Body["name"][1] = 'x'
			result.Header.Set("X-Local", "owned")
		}()
	}
	wait.Wait()
	if calls.Load() != 24 || callbacks.Load() != 24 {
		t.Fatalf("parallel calls/callbacks %d/%d", calls.Load(), callbacks.Load())
	}
}
