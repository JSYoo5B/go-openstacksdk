package image

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestUpdateImageOptionsFactoriesAndNilPresence(t *testing.T) {
	raw := json.RawMessage(`{"snap":"owned"}`)
	changes := []ImagePatch{{Op: "add", Path: "/x", Value: raw}}
	headers := map[string]string{"X-Policy": "owned"}
	full := WithUpdateImageOpts(UpdateImageOpts{Headers: headers, Changes: changes})
	single := WithUpdateImageChange(changes[0])
	many := WithUpdateImageChanges(changes...)
	fields := map[string]json.RawMessage{"x": raw}
	plural := WithUpdateImageFields(fields)
	properties := WithSetImagePropertiesProperties(fields)
	count := 0
	anyOption := WithUpdateImageField("x", taskOptionMarshaler{calls: &count})
	anyProperty := WithSetImagePropertiesProperty("x", taskOptionMarshaler{calls: &count})
	if count != 2 {
		t.Fatal("factory not immediate", count)
	}
	raw[9] = 'X'
	changes[0].Path = "/changed"
	headers["X-Policy"] = "changed"
	fields["x"] = json.RawMessage("null")
	calls := 0
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		changes, _ := updateCorePayload(t, req)
		if len(changes) != 1 || changes[0].Path != "/x" || string(changes[0].Value) != `{"snap":"owned"}` {
			t.Fatal(changes)
		}
		if calls == 1 && req.Header.Get("X-Policy") != "owned" {
			t.Fatal(req.Header)
		}
		return taskCoreJSON(req, 200, "{}"), nil
	}))
	for _, option := range []UpdateImageOption{full, single, many, plural, anyOption} {
		if v, e := service.UpdateImage(context.Background(), resource.ID("id"), option); v == nil || e != nil {
			t.Fatal(v, e)
		}
	}
	for _, option := range []SetImagePropertiesOption{properties, anyProperty} {
		if v, e := service.SetImageProperties(context.Background(), resource.ID("id"), option); v == nil || e != nil {
			t.Fatal(v, e)
		}
	}
	for _, option := range []UpdateImageOption{WithUpdateImageChange(ImagePatch{Op: "remove", Path: "/x", Value: json.RawMessage{}}), WithUpdateImageChanges(ImagePatch{Op: "remove", Path: "/x", Value: json.RawMessage{}}), WithUpdateImageOpts(UpdateImageOpts{Changes: []ImagePatch{{Op: "remove", Path: "/x", Value: json.RawMessage{}}}}), func(o *UpdateImageOpts) error {
		o.Changes = []ImagePatch{{Op: "remove", Path: "/x", Value: json.RawMessage{}}}
		return nil
	}} {
		if v, e := service.UpdateImage(context.Background(), resource.Name("Name"), option); v != nil || !errors.Is(e, resource.ErrInvalidOption) {
			t.Fatal("nonnil empty value collapsed", v, e)
		}
	}
	if calls != 7 || count != 2 {
		t.Fatal(calls, count)
	}
}

func TestUpdateImageOptionsReplacementAppendAndTypedFields(t *testing.T) {
	calls := 0
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		changes, _ := updateCorePayload(t, req)
		if req.Header.Get("X-Erased") != "" || req.Header.Get("X-Policy") != "last" {
			t.Fatal(req.Header)
		}
		if calls == 1 {
			paths := []string{"/name", "/visibility", "/protected", "/os_hidden", "/owner", "/container_format", "/disk_format", "/min_disk", "/min_ram", "/tags", "/a", "/z", "/a~1b~0", "/name"}
			values := []string{`""`, `"future"`, "false", "false", `""`, `"future"`, `"future"`, "0", "-1", "[]", "null", "[1]", `"literal"`, ""}
			if len(changes) != len(paths) {
				t.Fatal(changes)
			}
			for i, c := range changes {
				if c.Path != paths[i] || string(c.Value) != values[i] || i < len(paths)-1 && c.Op != "add" || i == len(paths)-1 && c.Op != "remove" {
					t.Fatal(i, c, paths, values)
				}
			}
		} else {
			expected := []ImagePatch{{Op: "add", Path: "/a", Value: json.RawMessage("null")}, {Op: "add", Path: "/properties", Value: json.RawMessage("false")}, {Op: "add", Path: "/readonly", Value: json.RawMessage("0")}}
			if !reflect.DeepEqual(changes, expected) {
				t.Fatal(changes)
			}
		}
		return taskCoreJSON(req, 200, "{}"), nil
	}))
	v, e := service.UpdateImage(context.Background(), resource.ID("id"), WithUpdateImageChange(ImagePatch{Op: "invalid"}), WithUpdateImageHeader("X-Erased", "old"), WithUpdateImageOpts(UpdateImageOpts{}), WithUpdateImageHeaders(map[string]string{"x-policy": "first"}), WithUpdateImageHeader("X-Policy", "last"), WithUpdateImageName(""), WithUpdateImageVisibility("future"), WithUpdateImageProtected(false), WithUpdateImageHidden(false), WithUpdateImageOwner(""), WithUpdateImageContainerFormat("future"), WithUpdateImageDiskFormat("future"), WithUpdateImageMinDisk(0), WithUpdateImageMinRAM(-1), WithUpdateImageTags(), WithUpdateImageFields(map[string]json.RawMessage{"z": json.RawMessage("[1]"), "a": json.RawMessage("null")}), WithUpdateImageField("a/b~", "literal"), WithUpdateImageRemoveField("name"))
	if v == nil || e != nil {
		t.Fatal(v, e)
	}
	v, e = service.SetImageProperties(context.Background(), resource.ID("id"), WithSetImagePropertiesHeader("X-Erased", "old"), WithSetImagePropertiesProperty("erased", 1), WithSetImagePropertiesOpts(SetImagePropertiesOpts{}), WithSetImagePropertiesHeaders(map[string]string{"x-policy": "first"}), WithSetImagePropertiesHeader("X-Policy", "last"), WithSetImagePropertiesProperties(map[string]json.RawMessage{"erased": json.RawMessage("1")}), WithSetImagePropertiesProperties(map[string]json.RawMessage{"properties": json.RawMessage("false"), "readonly": json.RawMessage("0")}), WithSetImagePropertiesProperty("a", nil))
	if v == nil || e != nil || calls != 2 {
		t.Fatal(v, e, calls)
	}
	p, err := parseUpdateImageOptions([]UpdateImageOption{WithUpdateImageName("erased"), WithUpdateImageChanges(ImagePatch{Op: "remove", Path: "/kept"}), WithUpdateImageChange(ImagePatch{Op: "add", Path: "/kept", Value: json.RawMessage("null")})})
	if err != nil || len(p.Changes) != 2 || p.Changes[0].Op != "remove" || p.Changes[1].Op != "add" {
		t.Fatal(p, err)
	}
}

func TestUpdateImageOptionsCallbacksOwnRetainedState(t *testing.T) {
	calls := 0
	callbacks := 0
	var retained *UpdateImageOpts
	var retainedSet *SetImagePropertiesOpts
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		changes, _ := updateCorePayload(t, req)
		if req.Header.Get("X-Policy") != "owned" || len(changes) != 1 || changes[0].Path != "/owned" || string(changes[0].Value) != "null" {
			t.Fatal(req.Header, changes)
		}
		return taskCoreJSON(req, 200, "{}"), nil
	}))
	first := func(o *UpdateImageOpts) error {
		callbacks++
		if o.Headers == nil {
			t.Fatal("nil Headers")
		}
		o.Headers["X-Policy"] = "owned"
		o.Changes = []ImagePatch{{Op: "add", Path: "/owned", Value: json.RawMessage("null")}}
		retained = o
		return nil
	}
	second := func(o *UpdateImageOpts) error {
		callbacks++
		retained.Headers["X-Policy"] = "changed"
		retained.Changes[0].Path = "/changed"
		retained.Changes[0].Value[0] = '0'
		if o.Headers["X-Policy"] != "owned" || o.Changes[0].Path != "/owned" || string(o.Changes[0].Value) != "null" {
			t.Fatal(o)
		}
		return nil
	}
	if v, e := service.UpdateImage(context.Background(), resource.ID("id"), first, second); v == nil || e != nil {
		t.Fatal(v, e)
	}
	if v, e := service.SetImageProperties(context.Background(), resource.ID("id"), func(o *SetImagePropertiesOpts) error {
		callbacks++
		if o.Headers == nil || o.Properties == nil {
			t.Fatal("nil callback maps")
		}
		o.Headers["X-Policy"] = "owned"
		o.Properties["owned"] = json.RawMessage("null")
		retainedSet = o
		return nil
	}, func(o *SetImagePropertiesOpts) error {
		callbacks++
		retainedSet.Headers["X-Policy"] = "changed"
		retainedSet.Properties["owned"][0] = '0'
		if string(o.Properties["owned"]) != "null" {
			t.Fatal(o)
		}
		return nil
	}); v == nil || e != nil {
		t.Fatal(v, e)
	}
	retained.Changes = nil
	retainedSet.Properties = nil
	cause := errors.New("callback")
	if v, e := service.UpdateImage(context.Background(), resource.Name("Parent"), func(*UpdateImageOpts) error { return cause }); v != nil || !errors.Is(e, cause) {
		t.Fatal(v, e)
	}
	if callbacks != 4 || calls != 2 {
		t.Fatal(callbacks, calls)
	}
}

func TestUpdateImageOptionsReusablePoliciesAndFactoryCauses(t *testing.T) {
	marshalCalls := 0
	cause := errors.New("marshal cause")
	bad := WithUpdateImageField("field", taskOptionMarshaler{calls: &marshalCalls, cause: cause})
	badProperty := WithSetImagePropertiesProperty("field", taskOptionMarshaler{calls: &marshalCalls, cause: cause})
	invalidKey := WithUpdateImageField(string([]byte{0xff}), taskOptionMarshaler{calls: &marshalCalls})
	if marshalCalls != 2 {
		t.Fatal("invalid key executed marshal", marshalCalls)
	}
	var calls atomic.Int64
	service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls.Add(1)
		changes, _ := updateCorePayload(t, req)
		if len(changes) != 1 || changes[0].Path != "/a~1b~0" || string(changes[0].Value) != "900719925474099312345" || req.Header.Get("X-Policy") != "fixed" {
			return nil, fmt.Errorf("unexpected owned request: %v %v", changes, req.Header)
		}
		return taskCoreJSON(req, 200, "{}"), nil
	}))
	for _, option := range []UpdateImageOption{bad, invalidKey, WithUpdateImageField("unsupported", make(chan int))} {
		v, e := service.UpdateImage(context.Background(), resource.Name("Name"), option)
		if v != nil || e == nil {
			t.Fatal(v, e)
		}
		if option != nil && calls.Load() != 0 {
			t.Fatal("factory error made HTTP")
		}
	}
	if v, e := service.UpdateImage(context.Background(), resource.Name("Name"), bad); v != nil || !errors.Is(e, cause) {
		t.Fatal(v, e)
	}
	if v, e := service.SetImageProperties(context.Background(), resource.Name("Name"), badProperty); v != nil || !errors.Is(e, cause) {
		t.Fatal(v, e)
	}
	policy := WithUpdateImageOpts(UpdateImageOpts{Headers: map[string]string{"X-Policy": "fixed"}, Changes: []ImagePatch{{Op: "add", Path: "/a~1b~0", Value: json.RawMessage("900719925474099312345")}}})
	setPolicy := WithSetImagePropertiesOpts(SetImagePropertiesOpts{Headers: map[string]string{"X-Policy": "fixed"}, Properties: map[string]json.RawMessage{"a/b~": json.RawMessage("900719925474099312345")}})
	var wg sync.WaitGroup
	errc := make(chan error, 40)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := service.UpdateImage(context.Background(), resource.ID("id"), policy); e != nil {
				errc <- e
			}
			if _, e := service.SetImageProperties(context.Background(), resource.ID("id"), setPolicy); e != nil {
				errc <- e
			}
		}()
	}
	wg.Wait()
	close(errc)
	for e := range errc {
		t.Error(e)
	}
	if calls.Load() != 40 || marshalCalls != 2 {
		t.Fatal(calls.Load(), marshalCalls)
	}
}
