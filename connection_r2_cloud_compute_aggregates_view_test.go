package openstack

import (
	"encoding/json"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// cloudAggregateView applies the openstack/fields.py conversions of the
// Aggregate descriptors: a non-list hosts value becomes a one-element list, a
// non-dict metadata value an empty dict, deleted uses Python truthiness, and
// null or absent members stay null.
func TestCloudAggregateViewFollowsSourceTypeConversions(t *testing.T) {
	for raw, want := range map[string]string{
		`{"hosts":"h1","metadata":"x","deleted":0}`:            `{"hosts":["h1"],"is_deleted":false,"metadata":{}}`,
		`{"hosts":null,"metadata":null,"deleted":null}`:        `{"hosts":null,"is_deleted":null,"metadata":null}`,
		`{"hosts":["a"],"metadata":{"k":"v"},"deleted":"yes"}`: `{"hosts":["a"],"is_deleted":true,"metadata":{"k":"v"}}`,
		`{}`: `{"hosts":null,"is_deleted":null,"metadata":null}`,
	} {
		view, err := cloudAggregateView(json.RawMessage(raw), resource.CloudLocation{})
		if err != nil {
			t.Fatal(raw, err)
		}
		var members map[string]json.RawMessage
		if err := json.Unmarshal(view, &members); err != nil {
			t.Fatal(err)
		}
		got, _ := json.Marshal(map[string]json.RawMessage{"hosts": members["hosts"], "metadata": members["metadata"], "is_deleted": members["is_deleted"]})
		if string(got) != want {
			t.Fatalf("%s: %s, want %s", raw, got, want)
		}
		for _, name := range []string{"id", "name", "uuid", "availability_zone", "created_at", "updated_at", "deleted_at", "location"} {
			if _, ok := members[name]; !ok {
				t.Fatalf("%s: missing %s in %s", raw, name, view)
			}
		}
		if _, ok := members["deleted"]; ok {
			t.Fatalf("%s: wire name deleted kept in %s", raw, view)
		}
	}
}
