package senlin

import (
	"fmt"
	"slices"
	"strings"

	"gophercloudsdk/resource"
)

// Sort validates the documented key[:asc|desc] comma-separated grammar.
// An omitted direction lets Senlin use its documented ascending default.
func Sort(value string, keys ...string) error {
	if value == "" {
		return nil
	}
	for _, item := range strings.Split(value, ",") {
		parts := strings.Split(item, ":")
		if len(parts) > 2 || !slices.Contains(keys, parts[0]) || len(parts) == 2 && parts[1] != "asc" && parts[1] != "desc" {
			return fmt.Errorf("%w: invalid Senlin sort expression %q", resource.ErrInvalidOption, item)
		}
	}
	return nil
}

func Bool(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
