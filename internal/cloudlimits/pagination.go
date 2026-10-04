package cloudlimits

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

func projectNext(response *rest.Response, fields map[string]json.RawMessage) (string, error) {
	links, present := fields["links"]
	if !present {
		links, present = fields["projects_links"]
	}
	if present {
		var object map[string]json.RawMessage
		if json.Unmarshal(links, &object) == nil && object != nil {
			// Native Keystone links.next is a deliberate compatibility repair.
			if next, err := nextText(object["next"]); err != nil || next != "" {
				return next, err
			}
		} else {
			var items []json.RawMessage
			if bytes.Equal(bytes.TrimSpace(links), []byte("null")) {
				return "", fmt.Errorf("project pagination links must not be null")
			}
			if err := json.Unmarshal(links, &items); err != nil {
				return "", err
			}
			for _, raw := range items {
				var item map[string]json.RawMessage
				if err := json.Unmarshal(raw, &item); err != nil {
					return "", err
				}
				if item == nil {
					return "", fmt.Errorf("project pagination link must be an object")
				}
				var rel string
				_ = json.Unmarshal(item["rel"], &rel)
				if rel == "next" {
					if href, present := item["href"]; present {
						next, err := nextText(href)
						if err != nil || next != "" {
							return next, err
						}
						break
					}
				}
			}
		}
	}
	if next, err := nextText(fields["next"]); err != nil || next != "" {
		return next, err
	}
	for _, header := range response.Header.Values("Link") {
		parts := splitLinkHeader(header)
		for _, part := range parts {
			part = strings.TrimSpace(part)
			end := strings.IndexByte(part, '>')
			if !strings.HasPrefix(part, "<") || end < 1 {
				return "", fmt.Errorf("invalid project Link header")
			}
			_, params, err := mime.ParseMediaType("application/links" + part[end+1:])
			if err != nil {
				return "", err
			}
			for _, rel := range strings.Fields(params["rel"]) {
				if rel == "next" {
					return part[1:end], nil
				}
			}
		}
	}
	return "", nil
}

func nextText(raw json.RawMessage) (string, error) {
	if raw == nil {
		return "", nil
	}
	truthy, err := limitsModelConnectionTruthy(raw)
	if err != nil || !truthy {
		return "", err
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return "", fmt.Errorf("project next link must be a string: %w", err)
	}
	return text, nil
}

func splitLinkHeader(header string) []string {
	var parts []string
	start := 0
	quoted, angle, escaped := false, false, false
	for i, c := range header {
		if escaped {
			escaped = false
			continue
		}
		if quoted && c == '\\' {
			escaped = true
			continue
		}
		if c == '"' {
			quoted = !quoted
		}
		if !quoted {
			if c == '<' {
				angle = true
			}
			if c == '>' {
				angle = false
			}
			if c == ',' && !angle {
				parts = append(parts, header[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, header[start:])
}

func projectContinuation(current, base *url.URL, href string) (*url.URL, error) {
	if !utf8.ValidString(href) {
		return nil, fmt.Errorf("%w: project next link must be UTF-8", resource.ErrInvalidOption)
	}
	for _, c := range href {
		if unicode.IsControl(c) {
			return nil, fmt.Errorf("%w: project next link contains controls", resource.ErrInvalidOption)
		}
	}
	reference, err := url.Parse(href)
	if err != nil {
		return nil, fmt.Errorf("%w: project next link: %w", resource.ErrInvalidOption, err)
	}
	if reference.Fragment != "" || strings.Contains(href, "#") || reference.User != nil || reference.Opaque != "" {
		return nil, fmt.Errorf("%w: invalid project continuation authority/form", resource.ErrInvalidOption)
	}
	if !reference.IsAbs() && reference.Host == "" {
		path := reference.Path
		if strings.HasPrefix(path, "/v") {
			if i := strings.IndexByte(path[1:], '/'); i >= 0 {
				path = path[i+1:]
			}
		}
		if path == "/projects" {
			reference.Path, reference.RawPath = base.Path, base.RawPath
		}
	}
	next := current.ResolveReference(reference)
	query, err := url.ParseQuery(current.RawQuery)
	if err != nil {
		return nil, err
	}
	advertised, err := url.ParseQuery(next.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("%w: project pagination query: %w", resource.ErrInvalidOption, err)
	}
	// Resource.list discards pagination keys after each received page, and
	// parse_qs drops empty advertised values before merging new parameters.
	query.Del("marker")
	query.Del("limit")
	for key, values := range advertised {
		var nonempty []string
		for _, value := range values {
			if value != "" {
				nonempty = append(nonempty, value)
			}
		}
		if len(nonempty) != 0 {
			query[key] = nonempty
		}
	}
	next.RawQuery = query.Encode()
	if _, err := projectPageKey(next, base); err != nil {
		return nil, err
	}
	return next, nil
}

func projectPageKey(target, base *url.URL) (string, error) {
	if target.User != nil || target.Opaque != "" || target.Fragment != "" || !strings.EqualFold(target.Scheme, base.Scheme) || !strings.EqualFold(target.Host, base.Host) || target.EscapedPath() != base.EscapedPath() {
		return "", fmt.Errorf("%w: project pagination changes collection origin or escaped path", resource.ErrInvalidOption)
	}
	query, err := url.ParseQuery(target.RawQuery)
	if err != nil {
		return "", fmt.Errorf("%w: project pagination query: %w", resource.ErrInvalidOption, err)
	}
	return strings.ToLower(target.Scheme) + "://" + strings.ToLower(target.Host) + target.EscapedPath() + "?" + query.Encode(), nil
}
