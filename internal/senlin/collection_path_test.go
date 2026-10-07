package senlin

import (
	"errors"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestCollectionPathOwnsLiteralRelativeSegments(t *testing.T) {
	for input, want := range map[string]string{
		"clusters":                     "clusters",
		"tenants/project-alpha/nodes":  "tenants/project-alpha/nodes",
		"extensions/v2.1/custom_nodes": "extensions/v2.1/custom_nodes",
		"서비스/노드":                       "%EC%84%9C%EB%B9%84%EC%8A%A4/%EB%85%B8%EB%93%9C",
		"vendor/@tenant&value/nodes":   "vendor/@tenant&value/nodes",
	} {
		got, err := CollectionPath(input)
		if err != nil || got != want {
			t.Fatalf("CollectionPath(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	for _, input := range []string{
		"", " ", "/clusters", "clusters/", "a//nodes", ".", "..",
		"a/./nodes", "a/../nodes", "https://other.example/clusters", "//other.example/nodes",
		"https:clusters", "a\\nodes", "a?query=1", "a#fragment", "a/%2e%2e/nodes",
		"a/%(id)s/nodes", "a%25nodes", "a nodes", "a\tnodes", "a\rnodes", "a\nnodes",
		"a\x00nodes", "a\x7fnodes", "a\u00a0nodes", "a\u2028nodes", "a\xffnodes",
	} {
		got, err := CollectionPath(input)
		if got != "" || !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("CollectionPath(%q) = %q, %v; unsafe path was accepted", input, got, err)
		}
	}
}
