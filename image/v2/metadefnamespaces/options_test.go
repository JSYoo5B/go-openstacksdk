package metadefnamespaces

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func namespaceOptionPointer[T any](value T) *T { return &value }

func TestMetadefNamespacesOptionsSnapshotsReplacementAndLastWins(t *testing.T) {
	display, description, visibility, owner, protected := "display", "description", "private", "owner", false
	headers := map[string]string{"X-Original": "before"}
	create := WithCreateOpts(CreateOpts{Headers: headers, DisplayName: &display, Description: &description, Visibility: &visibility, Owner: &owner, Protected: &protected})
	display, description, visibility, owner, protected = "mutated", "mutated", "public", "mutated", true
	headers["X-Original"] = "after"
	value, err := prepareCreate([]CreateOption{WithCreateDisplayName("superseded"), WithCreateHeader("X-Removed", "x"), create, WithCreateHeader("x-original", "last"), WithCreateProtected(false)})
	if err != nil || *value.DisplayName != "display" || *value.Description != "description" || *value.Visibility != "private" || *value.Owner != "owner" || *value.Protected || value.Headers["X-Original"] != "last" || value.Headers["X-Removed"] != "" {
		t.Fatalf("create snapshot/lastwins %+v %v", value, err)
	}
	*value.DisplayName = "local"
	value.Headers["X-Original"] = "local"
	again, err := prepareCreate([]CreateOption{create})
	if err != nil || *again.DisplayName != "display" || again.Headers["X-Original"] != "before" {
		t.Fatalf("helper reuse aliases %+v %v", again, err)
	}
	updateInput := UpdateOpts{Headers: map[string]string{"X-Replace": "before"}, Namespace: namespaceOptionPointer("new::name"), DisplayName: namespaceOptionPointer("display"), Description: namespaceOptionPointer("description"), Visibility: namespaceOptionPointer("public"), Owner: namespaceOptionPointer("owner"), Protected: namespaceOptionPointer(false)}
	update := WithUpdateOpts(updateInput)
	*updateInput.Namespace = "changed"
	*updateInput.Protected = true
	updateInput.Headers["X-Replace"] = "after"
	updated, err := prepareUpdate([]UpdateOption{WithUpdateHeader("X-Removed", "x"), update})
	if err != nil || *updated.Namespace != "new::name" || *updated.Protected || updated.Headers["X-Replace"] != "before" || updated.Headers["X-Removed"] != "" {
		t.Fatalf("update snapshot %+v %v", updated, err)
	}
	resourceType := "OS::thing"
	get := WithGetOpts(GetOpts{ResourceType: &resourceType, Headers: map[string]string{"X-Get": "before"}})
	resourceType = "changed"
	fetched, err := prepareGet([]GetOption{get, WithGetResourceType("")})
	if err != nil || fetched.ResourceType == nil || *fetched.ResourceType != "" {
		t.Fatalf("explicit empty %+v %v", fetched, err)
	}
	ignore := false
	deletion := WithDeleteOpts(DeleteOpts{IgnoreMissing: &ignore, Headers: map[string]string{"X-Delete": "before"}})
	ignore = true
	deleted, err := prepareDelete([]DeleteOption{deletion})
	if err != nil || *deleted.IgnoreMissing {
		t.Fatalf("delete snapshot %+v %v", deleted, err)
	}
	limit := 0
	listInput := ListOpts{Headers: map[string]string{"X-List": "before"}, Limit: &limit, Marker: "OS::start", Visibility: "public", SortKey: "arbitrary_db_column", SortDir: "asc", ResourceTypes: "OS::A, OS::B", MaxItems: 3, SinglePage: true}
	list := WithListOpts(listInput)
	limit = 2
	listInput.Headers["X-List"] = "after"
	listed, query, err := prepareList([]ListOption{WithListMarker("superseded"), list})
	if err != nil || *listed.Limit != 0 || query.Get("limit") != "0" || query.Get("marker") != "OS::start" || query.Get("resource_types") != "OS::A, OS::B" || listed.Headers["X-List"] != "before" || listed.MaxItems != 3 || !listed.SinglePage {
		t.Fatalf("list snapshot %+v %s %v", listed, query, err)
	}
	mapInput := map[string]string{"x-snapshot": "before"}
	mapOption := WithGetHeaders(mapInput)
	mapInput["x-snapshot"] = "after"
	if got, err := prepareGet([]GetOption{mapOption, WithGetHeader("X-Snapshot", "last")}); err != nil || got.Headers["X-Snapshot"] != "last" {
		t.Fatalf("header helper snapshot %+v %v", got, err)
	}
	// Public header helpers also work when applied directly to a zero options value.
	var zero GetOpts
	if err := WithGetHeader("X-Direct", "yes")(&zero); err != nil || zero.Headers["X-Direct"] != "yes" {
		t.Fatalf("direct helper %+v %v", zero, err)
	}
}

func TestMetadefNamespacesOptionsCallbackPointersAndSliceOwnership(t *testing.T) {
	var retained *CreateOpts
	callbacks := 0
	first := CreateOption(func(value *CreateOpts) error {
		callbacks++
		if value.Headers == nil {
			t.Fatal("callback headers uninitialized")
		}
		value.DisplayName = namespaceOptionPointer("before")
		value.Headers["X-Handle"] = "before"
		retained = value
		return nil
	})
	second := CreateOption(func(value *CreateOpts) error {
		callbacks++
		*retained.DisplayName = "old-handle"
		retained.Headers["X-Handle"] = "old-handle"
		if *value.DisplayName != "before" || value.Headers["X-Handle"] != "before" {
			t.Fatal("earlier callback handle aliases next config")
		}
		value.Owner = namespaceOptionPointer("owner")
		return nil
	})
	config, err := prepareCreate([]CreateOption{first, second})
	if err != nil || callbacks != 2 || *config.DisplayName != "before" || config.Headers["X-Handle"] != "before" {
		t.Fatalf("retained callback %+v %v", config, err)
	}
	*retained.DisplayName = "later"
	retained.Headers["X-Handle"] = "later"
	if *config.DisplayName != "before" || config.Headers["X-Handle"] != "before" {
		t.Fatal("closed retained handle affects prepared config")
	}
	var options []GetOption
	options = []GetOption{func(value *GetOpts) error {
		options[1] = func(*GetOpts) error { return errors.New("unowned replacement") }
		return nil
	}, WithGetHeader("X-Chosen", "yes")}
	calls := 0
	api := New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		if req.Header.Get("X-Chosen") != "yes" {
			t.Fatalf("option slice changed %#v", req.Header)
		}
		return namespaceCoreJSON(req, 200, `{}`), nil
	}))
	if _, err := api.Get(context.Background(), "literal", options...); err != nil || calls != 1 {
		t.Fatalf("method option slice %v calls%d", err, calls)
	}
	lazyCallbacks, lazyRequests := 0, 0
	lazy := New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
		lazyRequests++
		if req.URL.RawQuery != "" {
			t.Fatalf("local cap leaked query: %s", req.URL)
		}
		return namespaceCoreJSON(req, 200, `{"namespaces":[{}]}`), nil
	}))
	listOptions := []ListOption{func(*ListOpts) error { lazyCallbacks++; return nil }, WithListMaxItems(3)}
	seq := lazy.List(context.Background(), listOptions...)
	listOptions[1] = func(*ListOpts) error { return errors.New("changed source slice") }
	if lazyCallbacks != 0 || lazyRequests != 0 {
		t.Fatal("iterator is not lazy")
	}
	for iteration := 0; iteration < 2; iteration++ {
		for value, err := range seq {
			if err != nil || value == nil {
				t.Fatal(err)
			}
		}
	}
	if lazyCallbacks != 2 || lazyRequests != 2 {
		t.Fatalf("iterator reuse callbacks%d requests%d", lazyCallbacks, lazyRequests)
	}
}

func TestMetadefNamespacesOptionsValidationAndDefaults(t *testing.T) {
	create, err := prepareCreate(nil)
	if err != nil || create.Headers == nil || create.DisplayName != nil || create.Protected != nil {
		t.Fatalf("create defaults %+v %v", create, err)
	}
	update, err := prepareUpdate(nil)
	if err != nil || update.Namespace != nil || update.Protected != nil {
		t.Fatalf("update defaults %+v %v", update, err)
	}
	get, err := prepareGet(nil)
	if err != nil || get.ResourceType != nil {
		t.Fatalf("get defaults %+v %v", get, err)
	}
	deletion, err := prepareDelete(nil)
	if err != nil || deletion.IgnoreMissing != nil {
		t.Fatalf("delete default %+v %v", deletion, err)
	}
	list, query, err := prepareList(nil)
	if err != nil || list.Limit != nil || list.MaxItems != 0 || list.SinglePage || len(query) != 0 {
		t.Fatalf("list defaults %+v %s %v", list, query, err)
	}
	if _, _, err := prepareList([]ListOption{WithListLimit(0), WithListMarker(""), WithListSortKey("unknown_column"), WithListResourceTypes("OS::A, OS::A")}); err != nil {
		t.Fatal(err)
	}
	if err := literal(strings.Repeat("界", 80)); err != nil {
		t.Fatal(err)
	}
	for _, identity := range []string{"OS::Compute::Libvirt", " leading space ", "名 空間"} {
		if err := literal(identity); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := prepareCreate([]CreateOption{WithCreateDisplayName(strings.Repeat("界", 80)), WithCreateDescription(strings.Repeat("界", 500)), WithCreateOwner(strings.Repeat("界", 255))}); err != nil {
		t.Fatal(err)
	}
	checks := []func() error{
		func() error { _, e := prepareCreate([]CreateOption{nil}); return e }, func() error { _, e := prepareUpdate([]UpdateOption{nil}); return e }, func() error { _, e := prepareGet([]GetOption{nil}); return e }, func() error { _, e := prepareDelete([]DeleteOption{nil}); return e }, func() error { _, _, e := prepareList([]ListOption{nil}); return e },
	}
	for index, check := range checks {
		if err := check(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("nil callback family%d: %v", index, err)
		}
	}
	for _, key := range []string{"X-Auth-Token", "X-Service-Token", "Authorization", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "Te", "Upgrade", "Accept", "Content-Type", "OpenStack-API-Version", "X-OpenStack-Glance-Api-Version", "X-OpenStack-Image-Size"} {
		if _, err := prepareGet([]GetOption{WithGetHeader(key, "value")}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("protected header %q: %v", key, err)
		}
	}
	for _, headers := range []map[string]string{{"X-Test": "one", "x-test": "two"}, {"Bad Key": "x"}, {"X-Test": "a\r\nb"}, {"X-Test": string([]byte{0xff})}} {
		if _, err := prepareDelete([]DeleteOption{WithDeleteHeaders(headers)}); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("bad headers %#v: %v", headers, err)
		}
	}
	sourceHeaders := map[string]string{"Accept": "application/custom", "OpenStack-API-Version": "image 2.9"}
	if _, err := validateHeaders(sourceHeaders, true, "2.9"); err != nil {
		t.Fatal(err)
	}
	if _, err := validateHeaders(sourceHeaders, true, ""); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
	cause := errors.New("callback cause")
	if _, err := prepareUpdate([]UpdateOption{func(*UpdateOpts) error { return cause }}); err != cause {
		t.Fatalf("callback identity lost: %v", err)
	}
	if body := scalarBody("literal", namespaceOptionPointer(""), nil, nil, nil, namespaceOptionPointer(false)); !reflect.DeepEqual(body, map[string]any{"namespace": "literal", "display_name": "", "protected": false}) {
		t.Fatalf("false/empty presence %#v", body)
	}
}

func TestMetadefNamespacesOptionsParallelReusableHelpers(t *testing.T) {
	var calls atomic.Int32
	api := New(namespaceCoreClient(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		if req.Header.Get("X-Reuse") != "owned" || req.URL.Query().Get("resource_type") != "OS::A" {
			return nil, errors.New("shared helper snapshot changed")
		}
		return namespaceCoreJSON(req, 200, `{"namespace":"literal","unknown":9007199254740993}`), nil
	}))
	input := GetOpts{Headers: map[string]string{"X-Reuse": "owned"}, ResourceType: namespaceOptionPointer("OS::A")}
	option := WithGetOpts(input)
	input.Headers["X-Reuse"] = "mutated"
	*input.ResourceType = "mutated"
	var wg sync.WaitGroup
	failures := make(chan error, 24)
	for index := 0; index < 24; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, err := api.Get(context.Background(), "literal", option)
			if err != nil {
				failures <- err
				return
			}
			value.Body["unknown"][0] = '8'
			value.Header.Set("X-Proof", "local")
			*value.Namespace = "local"
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	if calls.Load() != 24 {
		t.Fatalf("calls%d", calls.Load())
	}
	// Every operation handle is closed before the race receipt is captured.
}
