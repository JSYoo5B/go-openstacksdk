package volumes

import (
	"bytes"
	"encoding/json"
	"maps"
	"slices"
)

// SummaryOpts owns singleton query and protocol controls. The default reads
// the current project and preserves the pinned Proxy's two fetches.
type SummaryOpts struct {
	AllProjects  bool
	Query        map[string]any
	Headers      map[string]string
	Microversion *string
	SingleFetch  bool
}
type SummaryOption func(*SummaryOpts) error

func cloneSummary(value SummaryOpts) SummaryOpts {
	value.Query = maps.Clone(value.Query)
	if value.Query == nil {
		value.Query = make(map[string]any)
	}
	for k, v := range value.Query {
		if raw, ok := v.(json.RawMessage); ok {
			value.Query[k] = json.RawMessage(bytes.Clone(raw))
		}
		if text, ok := v.([]string); ok {
			value.Query[k] = slices.Clone(text)
		}
	}
	value.Headers = maps.Clone(value.Headers)
	if value.Headers == nil {
		value.Headers = make(map[string]string)
	}
	if value.Microversion != nil {
		v := *value.Microversion
		value.Microversion = &v
	}
	return value
}
func WithSummaryOptions(value SummaryOpts) SummaryOption {
	owned := cloneSummary(value)
	err := freezeSummaryQuery(&owned)
	return func(c *SummaryOpts) error {
		if err != nil {
			return err
		}
		*c = cloneSummary(owned)
		return nil
	}
}
func WithSummaryAllProjects(value bool) SummaryOption {
	return func(c *SummaryOpts) error { c.AllProjects = value; return nil }
}
func WithSummaryQuery(value map[string]any) SummaryOption {
	cfg := cloneSummary(SummaryOpts{Query: value})
	err := freezeSummaryQuery(&cfg)
	return func(c *SummaryOpts) error {
		if err != nil {
			return err
		}
		c.Query = cloneSummary(cfg).Query
		return nil
	}
}
func WithSummaryHeaders(value map[string]string) SummaryOption {
	owned := maps.Clone(value)
	return func(c *SummaryOpts) error { c.Headers = maps.Clone(owned); return nil }
}
func WithSummaryMicroversion(value string) SummaryOption {
	return func(c *SummaryOpts) error { c.Microversion = &value; return nil }
}

// WithSummarySingleFetch improves the redundant fixed Python request sequence
// explicitly. It sends only the aggregate request carrying all_tenants.
func WithSummarySingleFetch(value bool) SummaryOption {
	return func(c *SummaryOpts) error { c.SingleFetch = value; return nil }
}
