package senlin

import (
	"errors"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestSortGrammarLeavesDeploymentKeysToServer(t *testing.T) {
	for _, value := range []string{"", "name", "name:desc,type:asc", "vendor_field:desc"} {
		if err := SortGrammar(value); err != nil {
			t.Fatalf("valid expression %q rejected: %v", value, err)
		}
	}
	for _, value := range []string{",name", "name,", ":asc", "name:", "name:up", "name:asc:desc", " name:asc", "name :desc"} {
		if err := SortGrammar(value); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("invalid expression %q accepted: %v", value, err)
		}
	}
	if err := Sort("vendor_field:desc", "name", "type"); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatalf("a documented allowlist was widened: %v", err)
	}
	if err := Sort("name:desc,type", "name", "type"); err != nil {
		t.Fatal(err)
	}
}
