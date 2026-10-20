package rest

import (
	"reflect"
	"testing"
)

func TestHeaderNextLinksPreservesQuotedDelimitersAndRelationOrder(t *testing.T) {
	header := `<https://example.test/types?marker=a,b>; rel="next"; title="comma, semicolon; escaped \"quote\"", </types?marker=last>; rel="prev NEXT"`
	got, err := HeaderNextLinks(header)
	want := []string{"https://example.test/types?marker=a,b", "/types?marker=last"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, err=%v", got, err)
	}
	for _, bad := range []string{"not-a-link", `<next>; rel="unterminated`, `<next>; rel="next", broken`} {
		if _, err := HeaderNextLinks(bad); err == nil {
			t.Fatalf("malformed consumed header %q accepted", bad)
		}
	}
}
