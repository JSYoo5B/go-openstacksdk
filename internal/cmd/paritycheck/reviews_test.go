package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func put(t *testing.T, root, path, body string) {
	t.Helper()
	path = filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func putJSON(t *testing.T, root, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	put(t, root, path, string(data))
}

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	put(t, root, "go.mod", "module example\n\ngo 1.25\n")
	put(t, root, "service/api.go", "package service\ntype API struct{}\nfunc (*API) Fetch() {}\nfunc New() *API {return nil}\ntype Collection[T any] struct{}\nfunc (*Collection[T]) Find() {}\nfunc private() {}\n")
	put(t, root, "service/api_test.go", "package service\nimport test \"testing\"\nfunc TestFetch(t *test.T) {}\nfunc TestNotATest() {}\nfunc TestHelper(value string) {}\nfunc Testlowercase(t *test.T) {}\n")
	put(t, root, "service/README.md", "# Usage\n")
	put(t, root, "internal/hidden.go", "package internal\nfunc Public() {}\n")
	putJSON(t, root, "api/gophercloud_inventory.json", map[string]any{
		"gophercloud_version": gophercloudPin,
		"operations":          []map[string]any{{"package": "service/v1/resources", "name": "Fetch", "source": "upstream/service"}},
	})
	putJSON(t, root, "api/openstacksdk/manifest.json", map[string]any{
		"revision": pythonPin, "proxy_methods": 1, "cloud_workflows": 0, "connection_methods": 0,
		"proxies": []map[string]any{{"file": "service/v1.json", "methods": 1}},
	})
	putJSON(t, root, "api/openstacksdk/service/v1.json", map[string]any{
		"revision":   pythonPin,
		"operations": []map[string]any{{"id": "service/v1/get", "kind": "proxy", "parameters": []any{map[string]any{"name": "id", "required": true}}}},
	})
	for _, file := range []string{"cloud", "connection"} {
		putJSON(t, root, "api/openstacksdk/"+file+".json", map[string]any{"revision": pythonPin, "operations": []any{}})
	}
	putJSON(t, root, "api/sdk_reviews.json", reviewLedger{Schema: 1, Pins: pins, Reviews: []supportReview{}})
	return root
}

func validReview(t *testing.T, root string) supportReview {
	t.Helper()
	catalog, err := loadInventory(root)
	if err != nil {
		t.Fatal(err)
	}
	return supportReview{
		ID: "python:service/v1/get", Fingerprint: catalog.Operations["python:service/v1/get"], Status: "go_mapping",
		GoAPI:       []string{"example/service.API.Fetch", "example/service.Collection.Find", "example/service.New"},
		Contracts:   []contractEvidence{{Behavior: "fetch preserves HTTP error", Tests: []string{"service/api_test.go:TestFetch"}}},
		Differences: []string{"context and error return"}, Docs: []string{"service/README.md"},
	}
}

func writeReviews(t *testing.T, root string, reviews ...supportReview) {
	t.Helper()
	putJSON(t, root, "api/sdk_reviews.json", reviewLedger{Schema: 1, Pins: pins, Reviews: reviews})
}

func TestCatalogSyncPreservesReviewsAndAddsUnresolvedOperations(t *testing.T) {
	root := fixture(t)
	review := validReview(t, root)
	writeReviews(t, root, review)
	original, err := os.ReadFile(filepath.Join(root, "api/sdk_reviews.json"))
	if err != nil {
		t.Fatal(err)
	}
	counts, err := check(root, true)
	if err != nil || counts.total != 2 || counts.status["go_mapping"] != 1 || counts.status["unresolved"] != 1 {
		t.Fatalf("counts=%+v err=%v", counts, err)
	}
	putJSON(t, root, "api/gophercloud_inventory.json", map[string]any{
		"gophercloud_version": gophercloudPin,
		"operations": []map[string]any{
			{"package": "service/v1/resources", "name": "Fetch", "source": "upstream/service"},
			{"package": "service/v1/resources", "name": "List", "source": "upstream/service"},
		},
	})
	if _, err := check(root, false); err == nil || !strings.Contains(err.Error(), "stale or incomplete") {
		t.Fatalf("missing new operation was accepted: %v", err)
	}
	counts, err = check(root, true)
	if err != nil || counts.total != 3 || counts.status["go_mapping"] != 1 || counts.status["unresolved"] != 2 {
		t.Fatalf("counts=%+v err=%v", counts, err)
	}
	final, _ := os.ReadFile(filepath.Join(root, "api/sdk_reviews.json"))
	if !reflect.DeepEqual(original, final) {
		t.Fatal("sync changed the manually reviewed evidence")
	}
	if _, err := check(root, false); err != nil {
		t.Fatal(err)
	}
}

func TestSourceDriftRejectsPreviousReviewEvenDuringSync(t *testing.T) {
	root := fixture(t)
	writeReviews(t, root, validReview(t, root))
	if _, err := check(root, true); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(filepath.Join(root, "api/sdk_support_catalog.json"))
	putJSON(t, root, "api/openstacksdk/service/v1.json", map[string]any{
		"revision":   pythonPin,
		"operations": []map[string]any{{"id": "service/v1/get", "kind": "proxy", "parameters": []any{map[string]any{"name": "id", "required": false, "default": "None"}}}},
	})
	if _, err := check(root, true); err == nil || !strings.Contains(err.Error(), "fingerprint changed") {
		t.Fatalf("review carried through changed default: %v", err)
	}
	after, _ := os.ReadFile(filepath.Join(root, "api/sdk_support_catalog.json"))
	if !reflect.DeepEqual(before, after) {
		t.Fatal("failed sync overwrote the old catalog")
	}
}

func TestReviewRequiresExistingEvidenceAndHonestStatus(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		change     func(*supportReview)
	}{
		{"unknown operation", "does not exist", func(r *supportReview) { r.ID = "python:absent" }},
		{"fingerprint", "fingerprint changed", func(r *supportReview) { r.Fingerprint = "" }},
		{"missing API", "not an exported declaration", func(r *supportReview) { r.GoAPI = []string{"example/service.API.Absent"} }},
		{"private API", "not an exported declaration", func(r *supportReview) { r.GoAPI = []string{"example/service.private"} }},
		{"internal API", "not an exported declaration", func(r *supportReview) { r.GoAPI = []string{"example/internal.Public"} }},
		{"test helper", "not a declared Test", func(r *supportReview) { r.Contracts[0].Tests = []string{"service/api_test.go:TestHelper"} }},
		{"lowercase test", "not a declared Test", func(r *supportReview) { r.Contracts[0].Tests = []string{"service/api_test.go:Testlowercase"} }},
		{"missing test", "not a declared Test", func(r *supportReview) { r.Contracts[0].Tests = []string{"service/api_test.go:TestMissing"} }},
		{"missing document", "missing evidence", func(r *supportReview) { r.Docs = []string{"service/missing.md"} }},
		{"external document", "invalid local evidence", func(r *supportReview) { r.Docs = []string{"../README.md"} }},
		{"no contracts", "requires Go APIs", func(r *supportReview) { r.Contracts = nil }},
		{"no test evidence", "each claimed contract", func(r *supportReview) { r.Contracts[0].Tests = nil }},
		{"no difference", "requires differences", func(r *supportReview) { r.Differences = nil }},
		{"partial mapping", "no remaining", func(r *supportReview) { r.Remaining = []string{"needs retry"} }},
		{"supported difference", "supported cannot", func(r *supportReview) { r.Status = "supported" }},
		{"unsupported without reason", "specific remaining", func(r *supportReview) { r.Status = "unsupported" }},
		{"unresolved without reason", "specific remaining", func(r *supportReview) { r.Status = "unresolved" }},
		{"invalid status", "unknown status", func(r *supportReview) { r.Status = "complete" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := fixture(t)
			review := validReview(t, root)
			tc.change(&review)
			writeReviews(t, root, review)
			if _, err := check(root, true); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err=%v; want %q", err, tc.want)
			}
		})
	}
}

func TestDuplicateReviewAndJSONCatalogIDsAreRejected(t *testing.T) {
	root := fixture(t)
	review := validReview(t, root)
	writeReviews(t, root, review, review)
	if _, err := check(root, true); err == nil || !strings.Contains(err.Error(), "duplicate review") {
		t.Fatal(err)
	}
	put(t, root, "duplicate.json", `{"operations":{"same":"first","same":"second"}}`)
	var target any
	if err := readJSON(filepath.Join(root, "duplicate.json"), &target); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatal(err)
	}
}

func TestInventoryRejectsDuplicateIDsVersionAndManifestMismatch(t *testing.T) {
	root := fixture(t)
	op := map[string]any{"package": "service/v1/resources", "name": "Fetch"}
	putJSON(t, root, "api/gophercloud_inventory.json", map[string]any{"gophercloud_version": gophercloudPin, "operations": []any{op, op}})
	if _, err := loadInventory(root); err == nil || !strings.Contains(err.Error(), "duplicate inventory operation") {
		t.Fatal(err)
	}
	putJSON(t, root, "api/gophercloud_inventory.json", map[string]any{"gophercloud_version": "v2.99.0", "operations": []any{op}})
	if _, err := loadInventory(root); err == nil || !strings.Contains(err.Error(), "must be pinned") {
		t.Fatal(err)
	}
	root = fixture(t)
	putJSON(t, root, "api/openstacksdk/connection.json", map[string]any{"revision": "wrong", "operations": []any{}})
	if _, err := loadInventory(root); err == nil || !strings.Contains(err.Error(), "inconsistent revision") {
		t.Fatal(err)
	}
}

func TestSDKDiscoveryHintsDoNotInvalidateSourceReviews(t *testing.T) {
	root := fixture(t)
	before, err := loadInventory(root)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "api/openstacksdk/service/v1.json")
	var inventory map[string]any
	if err := readJSON(path, &inventory); err != nil {
		t.Fatal(err)
	}
	op := inventory["operations"].([]any)[0].(map[string]any)
	op["candidates"] = []any{"gophercloudsdk/service/new_candidate"}
	op["review"] = "pending"
	putJSON(t, root, "api/openstacksdk/service/v1.json", inventory)
	after, err := loadInventory(root)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("catalog changed for candidate-only update: %v", err)
	}
}
