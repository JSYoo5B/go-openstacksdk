package cinderowned

import "maps"

// ReadHeaders sets per-request headers.
func ReadHeaders(values map[string]string) func(*ReadOpts) error {
	owned := maps.Clone(values)
	return func(c *ReadOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// WriteSnapshot replaces the whole write policy with an owned copy.
func WriteSnapshot(v WriteOpts) func(*WriteOpts) error {
	owned := CopyWrite(v)
	return func(c *WriteOpts) error { *c = CopyWrite(owned); return nil }
}

// WriteField records one Python attribute. Unknown names are ignored before
// marshaling, as Resource.new drops them.
func (s *Spec) WriteField(key string, value any) func(*WriteOpts) error {
	if _, known := s.Attrs[key]; !known {
		return func(*WriteOpts) error { return nil }
	}
	raw := Marshal(value)
	return func(c *WriteOpts) error { return SetField(&c.Fields, key, raw) }
}

// WriteHeaders sets per-request headers.
func WriteHeaders(values map[string]string) func(*WriteOpts) error {
	owned := maps.Clone(values)
	return func(c *WriteOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// DeleteIgnoreMissing selects Python ignore_missing.
func DeleteIgnoreMissing(v bool) func(*DeleteOpts) error {
	return func(c *DeleteOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}

// DeleteHeaders sets per-request headers.
func DeleteHeaders(values map[string]string) func(*DeleteOpts) error {
	owned := maps.Clone(values)
	return func(c *DeleteOpts) error { c.Headers = maps.Clone(owned); return nil }
}

// FindIgnoreMissing selects Python ignore_missing.
func FindIgnoreMissing(v bool) func(*FindOpts) error {
	return func(c *FindOpts) error { x := v; c.IgnoreMissing = &x; return nil }
}

// ListSnapshot replaces the whole list policy with an owned copy.
func ListSnapshot(v ListOpts) func(*ListOpts) error {
	owned := CopyList(v)
	return func(c *ListOpts) error { *c = CopyList(owned); return nil }
}

// ListQuery records one Python **query name. Names that are neither server
// query parameters nor local Body filters are ignored before marshaling.
func (s *Spec) ListQuery(key string, value any) func(*ListOpts) error {
	_, query := s.Query[key]
	_, filter := s.Filters[key]
	if !query && !filter {
		return func(*ListOpts) error { return nil }
	}
	raw := Marshal(value)
	return func(c *ListOpts) error { return SetField(&c.Query, key, raw) }
}

func ListLimit(v int) func(*ListOpts) error {
	return func(c *ListOpts) error { x := v; c.Limit = &x; return nil }
}
func ListMarker(v string) func(*ListOpts) error {
	return func(c *ListOpts) error { c.Marker = v; return nil }
}
func ListMaxItems(v int) func(*ListOpts) error {
	return func(c *ListOpts) error { c.MaxItems = v; return nil }
}
func ListPaginated(v bool) func(*ListOpts) error {
	return func(c *ListOpts) error { x := v; c.Paginated = &x; return nil }
}
func ListHeaders(values map[string]string) func(*ListOpts) error {
	owned := maps.Clone(values)
	return func(c *ListOpts) error { c.Headers = maps.Clone(owned); return nil }
}
