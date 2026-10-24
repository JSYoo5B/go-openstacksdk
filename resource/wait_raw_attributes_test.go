package resource_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// rawWaitRecord mimics an SDK record that keeps its declared Resource view
// as raw JSON and exposes no exported string attribute of its own.
type rawWaitRecord struct {
	Resource *resource.RawResource
	Header   map[string][]string
}

// rawWaitCollection serves the views in order and repeats the last one.
func rawWaitCollection(status func(*rawWaitRecord) string, views ...string) (*resource.Collection[rawWaitRecord], *int) {
	calls := new(int)
	return resource.NewCollection(resource.Adapter[rawWaitRecord]{
		Kind:           "raw",
		WaitAttributes: func(v *rawWaitRecord) map[string]json.RawMessage { return resource.RawResourceView(v.Resource) },
		Status:         status,
		ID:             func(*rawWaitRecord) string { return "fixed" },
		Get: func(context.Context, string) (*rawWaitRecord, error) {
			view := views[min(*calls, len(views)-1)]
			*calls++
			if view == "" {
				return &rawWaitRecord{}, nil
			}
			var body map[string]json.RawMessage
			if err := json.Unmarshal([]byte(view), &body); err != nil {
				return nil, err
			}
			return &rawWaitRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: body}}}, nil
		},
	}), calls
}

func rawFast() []resource.WaitOption {
	return []resource.WaitOption{resource.WithPollInterval(time.Millisecond), resource.WithTimeout(5 * time.Second)}
}

func TestWaitRawAttributeComparesViewStringsLikeTypedFields(t *testing.T) {
	ctx := context.Background()
	collection, calls := rawWaitCollection(nil, `{"ether_type":"IPv4","ethertype":"wire"}`, `{"ether_type":"ipv6"}`)
	value, err := collection.Wait(ctx, resource.ID("fixed"), "IPV6", append(rawFast(), resource.WithStatusAttribute("ether_type"))...)
	if err != nil || value == nil || *calls != 2 {
		t.Fatalf("value=%v err=%v calls=%d", value, err, *calls)
	}

	// Failure states use the same case-insensitive comparison.
	collection, calls = rawWaitCollection(nil, `{"state":"building"}`, `{"state":"Broken"}`)
	_, err = collection.Wait(ctx, resource.ID("fixed"), "ready", append(rawFast(), resource.WithStatusAttribute("state"), resource.WithFailureStates("BROKEN"))...)
	var failed *resource.FailedStateError
	if !errors.As(err, &failed) || failed.Status != "Broken" || *calls != 2 {
		t.Fatalf("err=%v calls=%d", err, *calls)
	}

	// Progress is still read from the record struct, which has none.
	var progress []int
	collection, _ = rawWaitCollection(nil, `{"state":"a"}`, `{"state":"b"}`)
	if _, err := collection.Wait(ctx, resource.ID("fixed"), "b", append(rawFast(), resource.WithStatusAttribute("state"), resource.WithProgressCallback(func(p int) { progress = append(progress, p) }))...); err != nil || len(progress) != 1 || progress[0] != 0 {
		t.Fatalf("progress=%v err=%v", progress, err)
	}
}

func TestWaitRawAttributeRejectsMissingNullAndNonStringValues(t *testing.T) {
	ctx := context.Background()
	for _, test := range []struct{ name, view, attribute string }{
		{"wire spelling is not an attribute", `{"ether_type":"IPv4"}`, "ethertype"},
		{"absent", `{"id":"fixed"}`, "status"},
		{"null", `{"status":null}`, "status"},
		{"number", `{"status":1}`, "status"},
		{"boolean", `{"status":true}`, "status"},
		{"object", `{"status":{"a":"b"}}`, "status"},
		{"case differs", `{"status":"ready"}`, "Status"},
		{"no view", ``, "status"},
	} {
		t.Run(test.name, func(t *testing.T) {
			collection, calls := rawWaitCollection(nil, test.view)
			if _, err := collection.Wait(ctx, resource.ID("fixed"), "ready", append(rawFast(), resource.WithStatusAttribute(test.attribute))...); !errors.Is(err, resource.ErrUnsupported) || *calls != 1 {
				t.Fatalf("wait err=%v calls=%d", err, *calls)
			}
			collection, calls = rawWaitCollection(nil, test.view)
			if err := collection.WaitDeleted(ctx, resource.ID("fixed"), append(rawFast(), resource.WithStatusAttribute(test.attribute))...); !errors.Is(err, resource.ErrUnsupported) || *calls != 1 {
				t.Fatalf("delete err=%v calls=%d", err, *calls)
			}
		})
	}
}

func TestWaitRawAttributeKeepsBindingStatusDefaultAndDeletion(t *testing.T) {
	ctx := context.Background()
	status := func(v *rawWaitRecord) string {
		var text string
		_ = json.Unmarshal(resource.RawResourceView(v.Resource)["status"], &text)
		return text
	}
	// Without a selection the binding status accessor applies.
	collection, calls := rawWaitCollection(status, `{"status":"creating","replication_status":"enabled"}`, `{"status":"available"}`)
	if _, err := collection.Wait(ctx, resource.ID("fixed"), "AVAILABLE", rawFast()...); err != nil || *calls != 2 {
		t.Fatalf("default err=%v calls=%d", err, *calls)
	}
	// A selection replaces it for both waiting and the deleted check.
	collection, calls = rawWaitCollection(status, `{"status":"creating","replication_status":"enabled"}`)
	if _, err := collection.Wait(ctx, resource.ID("fixed"), "ENABLED", append(rawFast(), resource.WithStatusAttribute("replication_status"))...); err != nil || *calls != 1 {
		t.Fatalf("selected err=%v calls=%d", err, *calls)
	}
	collection, calls = rawWaitCollection(nil, `{"phase":"active"}`, `{"phase":"DELETED"}`)
	if err := collection.WaitDeleted(ctx, resource.ID("fixed"), append(rawFast(), resource.WithStatusAttribute("phase"))...); err != nil || *calls != 2 {
		t.Fatalf("deleted err=%v calls=%d", err, *calls)
	}
	// Without a selection or a binding status, waiting fails before any GET.
	collection, calls = rawWaitCollection(nil, `{"status":"ready"}`)
	if _, err := collection.Wait(ctx, resource.ID("fixed"), "ready", rawFast()...); !errors.Is(err, resource.ErrUnsupported) || *calls != 0 {
		t.Fatalf("unselected err=%v calls=%d", err, *calls)
	}
	// Invalid attribute names still fail before any GET.
	if _, err := collection.Wait(ctx, resource.ID("fixed"), "ready", resource.WithStatusAttribute("a.b")); !errors.Is(err, resource.ErrInvalidOption) || *calls != 0 {
		t.Fatalf("invalid err=%v calls=%d", err, *calls)
	}
}

func TestWaitRawAttributeRespectsFixedWaitStatus(t *testing.T) {
	calls := 0
	collection := resource.NewCollection(resource.Adapter[rawWaitRecord]{
		Kind: "raw", FixedWaitStatus: true,
		WaitAttributes: func(v *rawWaitRecord) map[string]json.RawMessage { return resource.RawResourceView(v.Resource) },
		Status:         func(*rawWaitRecord) string { return "x" },
		ID:             func(*rawWaitRecord) string { return "fixed" },
		Get:            func(context.Context, string) (*rawWaitRecord, error) { calls++; return &rawWaitRecord{}, nil },
	})
	if _, err := collection.Wait(context.Background(), resource.ID("fixed"), "x", resource.WithStatusAttribute("status")); !errors.Is(err, resource.ErrUnsupported) || calls != 0 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}
