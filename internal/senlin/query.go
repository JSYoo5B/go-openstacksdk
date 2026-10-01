package senlin

import (
	"fmt"
	"slices"
	"strings"

	"gophercloudsdk/resource"
)

// SortGrammar validates key[:asc|desc] comma-separated expressions without
// guessing which plugin or deployment-specific fields the server supports.
func SortGrammar(value string) error {
	if value == "" {
		return nil
	}
	for _, item := range strings.Split(value, ",") {
		parts := strings.Split(item, ":")
		if len(parts) > 2 || parts[0] == "" || strings.TrimSpace(parts[0]) != parts[0] || len(parts) == 2 && parts[1] != "asc" && parts[1] != "desc" {
			return fmt.Errorf("%w: invalid Senlin sort expression %q", resource.ErrInvalidOption, item)
		}
	}
	return nil
}

// Sort additionally restricts keys where the service documents an allowlist.
// An omitted direction lets Senlin use its documented ascending default.
func Sort(value string, keys ...string) error {
	if err := SortGrammar(value); err != nil {
		return err
	}
	if value == "" {
		return nil
	}
	for _, item := range strings.Split(value, ",") {
		if !slices.Contains(keys, strings.SplitN(item, ":", 2)[0]) {
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
