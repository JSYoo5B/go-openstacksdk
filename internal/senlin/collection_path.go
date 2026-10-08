package senlin

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// CollectionPath owns a literal collection route below the service resource
// base. Callers supply unescaped segments; URLs and Python format templates are
// not routes. No normalization silently removes or changes path components.
func CollectionPath(value string) (string, error) {
	invalid := func() (string, error) {
		return "", fmt.Errorf("%w: collection path must contain nonempty, unescaped relative segments without whitespace, controls, URLs or templates", resource.ErrInvalidOption)
	}
	if value == "" || !utf8.ValidString(value) || strings.ContainsAny(value, ":\\?#%") {
		return invalid()
	}
	for _, character := range value {
		if unicode.IsSpace(character) || unicode.IsControl(character) {
			return invalid()
		}
	}
	parts := strings.Split(value, "/")
	for index, part := range parts {
		if err := Identifier(part); err != nil {
			return invalid()
		}
		parts[index] = url.PathEscape(part)
	}
	return strings.Join(parts, "/"), nil
}
