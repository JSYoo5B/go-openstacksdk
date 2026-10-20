package cloudread

import (
	"context"
	"maps"
)

// WithPolicy owns per-operation headers and an explicit microversion override.
// The original selected service stays unchanged and continues to be guarded.
// Auth, route and protocol headers retain the same library-owned restrictions.
func (s *Source) WithPolicy(ctx context.Context, microversion *string, headers map[string]string) error {
	if s == nil {
		return invalid("read source is required")
	}
	if err := s.Guard(ctx); err != nil {
		return err
	}
	copy := s.Client
	copy.MoreHeaders = maps.Clone(s.Client.MoreHeaders)
	if microversion != nil {
		copy.Microversion = *microversion
	}
	overrides, err := canonicalHeaders(headers, s.role, copy.Microversion)
	if err != nil {
		return err
	}
	if len(overrides) != 0 {
		if copy.MoreHeaders == nil {
			copy.MoreHeaders = make(map[string]string, len(overrides))
		}
		maps.Copy(copy.MoreHeaders, overrides)
	}
	canonical, err := validateSource(ctx, &copy, s.role)
	if err != nil {
		return err
	}
	if err := s.Guard(ctx); err != nil {
		return err
	}
	copy.MoreHeaders = canonical
	s.Client = copy
	return nil
}
