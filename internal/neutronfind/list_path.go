package neutronfind

import (
	"context"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

// withListBasePath clones the concrete profile. Per-call routes never change
// the captured engine or other consumers, and paging remains locked to the
// selected collection path rather than the binding's original default.
func (e *Engine) withListBasePath(ctx context.Context, path string) (*Engine, error) {
	if err := e.Check(ctx); err != nil {
		return nil, err
	}
	if !utf8.ValidString(path) || strings.IndexFunc(path, unicode.IsControl) >= 0 {
		return nil, findInvalid("list base path must be valid UTF-8 without controls")
	}
	parsed, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	if parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return nil, findInvalid("list base path must be a service-relative path")
	}
	for _, segment := range strings.Split(parsed.Path, "/") {
		if segment == "." || segment == ".." {
			return nil, findInvalid("list base path contains traversal")
		}
	}
	selected := *e
	selected.selectedGuard = e.Check
	if path != "" {
		selected.profile.Path = strings.TrimPrefix(parsed.EscapedPath(), "/")
		actual, err := url.Parse(e.source.Client.ServiceURL(selected.profile.Path))
		if err != nil {
			return nil, err
		}
		selected.profile.VersionedPath = actual.EscapedPath()
	}
	return &selected, selected.Check(ctx)
}
