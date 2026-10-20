package users

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"testing"
)

// Fresh source/AST hashes are checked by sdkgen. This test links that reviewed
// manifest to the independently owned runtime map without enabling a native
// User or Project collection's semantic filters.
func TestUserProjectRecordDescriptorMatchesPinnedOwnedManifest(t *testing.T) {
	data, err := os.ReadFile("../../../api/openstacksdk/resources/identity/v3/user_project.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		SourcePin  string            `json:"source_pin"`
		Resource   string            `json:"resource"`
		SDKPackage string            `json:"sdk_package"`
		BasePath   string            `json:"base_path"`
		Envelope   string            `json:"envelope"`
		Query      map[string]string `json:"query"`
		Body       map[string]struct {
			Field        string  `json:"field"`
			ResponseType *string `json:"response_type"`
		} `json:"body"`
		URI map[string]struct {
			Field string `json:"field"`
		} `json:"uri"`
		Reserved []string `json:"reserved"`
		Counts   struct {
			CanonicalQuery int `json:"canonical_query"`
			AcceptedQuery  int `json:"accepted_query"`
			LocalBody      int `json:"local_body"`
		} `json:"counts"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SourcePin != "ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe" || manifest.Resource != "openstack.identity.v3.project.UserProject" || manifest.SDKPackage != "github.com/JSYoo5B/go-openstacksdk/identity/v3/users" || manifest.BasePath != "/users/%(user_id)s/projects" || manifest.Envelope != "projects" || manifest.Counts.CanonicalQuery != 11 || manifest.Counts.AcceptedQuery != 15 || manifest.Counts.LocalBody != 4 {
		t.Fatalf("unexpected owned UserProject manifest identity: %+v", manifest)
	}
	descriptor := projectRecordFilterDescriptor()
	if !reflect.DeepEqual(descriptor.Query, manifest.Query) {
		t.Fatalf("query differs from pinned descriptor: %v != %v", descriptor.Query, manifest.Query)
	}
	body := make(map[string]string, len(manifest.Body))
	for name, field := range manifest.Body {
		body[name] = field.Field
		if name == "options" {
			if field.ResponseType == nil || *field.ResponseType != "dict" {
				t.Fatalf("options descriptor type = %v", field.ResponseType)
			}
		} else if field.ResponseType != nil {
			t.Fatalf("untyped Body %q acquired a conversion: %v", name, field.ResponseType)
		}
	}
	if !reflect.DeepEqual(descriptor.Body, body) || len(manifest.URI) != 1 || manifest.URI["user_id"].Field != "user_id" {
		t.Fatalf("Body or parent URI differs: %v, %v", descriptor.Body, manifest.URI)
	}
	reserved := append(slices.Clone(manifest.Reserved), "user_id")
	if !reflect.DeepEqual(descriptor.Reserved, reserved) {
		t.Fatalf("controls or parent protection differs: %v != %v", descriptor.Reserved, reserved)
	}

	// Each operation receives fresh policy maps. A caller cannot contaminate a
	// later iterator by mutating a descriptor retained by an earlier operation.
	descriptor.Query["is_enabled"] = "changed"
	descriptor.Body["options"] = "changed"
	descriptor.Reserved[0] = "changed"
	fresh := projectRecordFilterDescriptor()
	if !reflect.DeepEqual(fresh.Query, manifest.Query) || !reflect.DeepEqual(fresh.Body, body) || !reflect.DeepEqual(fresh.Reserved, reserved) {
		t.Fatal("owned descriptor shares mutable state between calls")
	}
}
