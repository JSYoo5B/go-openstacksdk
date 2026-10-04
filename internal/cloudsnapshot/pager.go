package cloudsnapshot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"gophercloudsdk/internal/cloudfilter"
	"gophercloudsdk/internal/jsonfilter"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

// snapshotNext follows source body precedence. The caller owns the canonical
// empty-list EOF check, physical response failure and continuation/cycle guards.
func snapshotNext(fields map[string]json.RawMessage, headers http.Header) (json.RawMessage, error) {
	return resourceNext(fields, headers, "snapshots")
}

func resourceNext(fields map[string]json.RawMessage, headers http.Header, plural string) (json.RawMessage, error) {
	links, present := fields["links"]
	if !present {
		links, present = fields[plural+"_links"]
	}
	if present {
		next, err := snapshotSourceLinks(links)
		if err != nil {
			return nil, err
		}
		truthy, err := cloudfilter.PythonTruthy(next)
		if err != nil {
			return nil, err
		}
		if truthy {
			return next, nil
		}
	}
	if next, present := fields["next"]; present {
		truthy, err := cloudfilter.PythonTruthy(next)
		if err != nil {
			return nil, err
		}
		if truthy {
			return bytes.Clone(next), nil
		}
	}
	// Requests exposes URL rather than the pinned source's "uri". The shared
	// quoted Link parser is an explicit usable HTTP repair, with its existing
	// multi-relation/case-insensitive policy. Repeated next relations use last.
	var href string
	for _, header := range headers.Values("Link") {
		if strings.TrimSpace(header) == "" {
			continue
		}
		if !utf8.ValidString(header) {
			return nil, snapshotPagerInvalid("Cinder resource Link header must be UTF-8")
		}
		for _, character := range header {
			if unicode.IsControl(character) && character != '\t' {
				return nil, snapshotPagerInvalid("Cinder resource Link header contains controls")
			}
		}
		links, err := rest.HeaderNextLinks(header)
		if err != nil {
			return nil, err
		}
		if len(links) != 0 {
			href = links[len(links)-1]
		}
	}
	if href == "" {
		return nil, nil
	}
	raw, _ := json.Marshal(href)
	return raw, nil
}

func snapshotSourceLinks(raw json.RawMessage) (json.RawMessage, error) {
	value := bytes.TrimSpace(raw)
	if !utf8.Valid(raw) || !json.Valid(value) {
		return nil, snapshotPagerInvalid("Cinder resource links must be complete UTF-8 JSON")
	}
	switch value[0] {
	case '{':
		// Source turns a dictionary into one-key dictionaries. No such item
		// has both rel==next and href. Do not add Keystone links.next repair.
		return nil, nil
	case '"':
		var text string
		if err := json.Unmarshal(value, &text); err != nil {
			return nil, err
		}
		if text == "" {
			return nil, nil
		}
		return nil, snapshotPagerInvalid("Cinder resource links contains a nonobject row")
	case '[':
		var rows []json.RawMessage
		if err := json.Unmarshal(value, &rows); err != nil {
			return nil, err
		}
		for _, rawRow := range rows {
			var row map[string]json.RawMessage
			if err := json.Unmarshal(rawRow, &row); err != nil {
				return nil, fmt.Errorf("Cinder resource pagination link row: %w", err)
			}
			if row == nil {
				return nil, snapshotPagerInvalid("Cinder resource pagination link row must be an object")
			}
			var rel string
			if err := json.Unmarshal(row["rel"], &rel); err != nil || rel != "next" {
				continue
			}
			if href, present := row["href"]; present {
				// A falsey selected href stops the rows too. The outer helper
				// falls through to top next/header, never to a later body row.
				return bytes.Clone(href), nil
			}
		}
		return nil, nil
	default:
		return nil, snapshotPagerInvalid("Cinder resource links must be iterable rows")
	}
}

// snapshotContinuation retains the authoritative raw query map rather than
// reparsing its bool/number/container source values from the physical URL.
// Only advertised parse_qs values become JSON string arrays. The caller owns
// whole-URL visited-page tracking and max-items/page-end timing.
func snapshotContinuation(current string, next json.RawMessage, collection string, query map[string]json.RawMessage, initialLimit, lastID json.RawMessage) (string, map[string]json.RawMessage, bool, error) {
	if _, err := snapshotPageKey(current, collection); err != nil {
		return "", nil, false, err
	}
	currentURL, _ := url.Parse(current)
	merged := make(map[string]json.RawMessage, len(query)+2)
	for key, value := range query {
		merged[key] = bytes.Clone(value)
	}
	previousMarker := query["marker"]
	delete(merged, "marker")
	delete(merged, "limit")
	truthy, err := cloudfilter.PythonTruthy(next)
	if err != nil {
		return "", nil, false, err
	}
	var target *url.URL
	var newMarker json.RawMessage
	var markerPresent bool
	if truthy {
		var href string
		if err := json.Unmarshal(next, &href); err != nil {
			return "", nil, false, snapshotPagerInvalid("a truthy Cinder resource next link must be a string")
		}
		reference, err := snapshotPagerURL(href, false)
		if err != nil {
			return "", nil, false, err
		}
		advertised, err := url.ParseQuery(reference.RawQuery)
		if err != nil {
			return "", nil, false, snapshotPagerInvalid("Cinder resource advertised query: %v", err)
		}
		for key, values := range advertised {
			if !utf8.ValidString(key) {
				return "", nil, false, snapshotPagerInvalid("Cinder resource advertised query key must be UTF-8")
			}
			var retained []string
			for _, value := range values {
				if !utf8.ValidString(value) {
					return "", nil, false, snapshotPagerInvalid("Cinder resource advertised query value must be UTF-8")
				}
				if value != "" {
					retained = append(retained, value)
				}
			}
			if len(retained) != 0 {
				raw, _ := json.Marshal(retained)
				merged[key] = raw
				if key == "marker" {
					newMarker, markerPresent = raw, true
				}
			}
		}
		target = currentURL.ResolveReference(reference)
	} else {
		limited, err := cloudfilter.PythonTruthy(initialLimit)
		if err != nil {
			return "", nil, false, err
		}
		if !limited {
			return "", merged, false, nil
		}
		target, _ = url.Parse(collection)
		newMarker = bytes.Clone(lastID)
		if newMarker == nil {
			newMarker = json.RawMessage("null")
		}
		markerPresent = true
		merged["marker"] = bytes.Clone(newMarker)
		merged["limit"] = bytes.Clone(initialLimit)
	}
	if markerPresent {
		if previousMarker == nil {
			previousMarker = json.RawMessage("null")
		}
		equal, err := jsonfilter.EqualPythonJSON(newMarker, previousMarker)
		if err != nil {
			return "", nil, false, err
		}
		if equal {
			return "", nil, false, &resource.PaginationCycleError{URL: current}
		}
	}
	encoded, err := encodeQuery(merged)
	if err != nil {
		return "", nil, false, err
	}
	target.RawQuery = encoded.Encode()
	target.ForceQuery = false
	if _, err := snapshotPageKey(target.String(), collection); err != nil {
		return "", nil, false, err
	}
	return target.String(), merged, true, nil
}

// snapshotPageKey is a canonical physical-page key. It preserves an exact
// escaped collection path and same origin, rejecting URI authority/traversal.
func snapshotPageKey(rawURL, collection string) (string, error) {
	target, err := snapshotPagerURL(rawURL, true)
	if err != nil {
		return "", err
	}
	base, err := snapshotPagerURL(collection, true)
	if err != nil {
		return "", err
	}
	if base.RawQuery != "" || base.ForceQuery {
		return "", snapshotPagerInvalid("Cinder resource collection URL must not have a query")
	}
	if !strings.EqualFold(target.Scheme, base.Scheme) || !strings.EqualFold(target.Host, base.Host) || target.EscapedPath() != base.EscapedPath() {
		return "", snapshotPagerInvalid("Cinder resource pagination changes collection origin or escaped path")
	}
	query, err := url.ParseQuery(target.RawQuery)
	if err != nil {
		return "", snapshotPagerInvalid("Cinder resource pagination query: %v", err)
	}
	for key, values := range query {
		if !utf8.ValidString(key) {
			return "", snapshotPagerInvalid("Cinder resource pagination query key must be UTF-8")
		}
		for _, value := range values {
			if !utf8.ValidString(value) {
				return "", snapshotPagerInvalid("Cinder resource pagination query value must be UTF-8")
			}
		}
	}
	return strings.ToLower(target.Scheme) + "://" + strings.ToLower(target.Host) + target.EscapedPath() + "?" + query.Encode(), nil
}

func snapshotPagerURL(raw string, absolute bool) (*url.URL, error) {
	if !utf8.ValidString(raw) || strings.Contains(raw, "#") {
		return nil, snapshotPagerInvalid("Cinder resource pagination URL must be UTF-8 without a fragment")
	}
	for _, character := range raw {
		if unicode.IsControl(character) {
			return nil, snapshotPagerInvalid("Cinder resource pagination URL contains controls")
		}
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, snapshotPagerInvalid("Cinder resource pagination URL: %v", err)
	}
	if parsed.User != nil || parsed.Opaque != "" || parsed.Fragment != "" {
		return nil, snapshotPagerInvalid("Cinder resource pagination URL changes authority or form")
	}
	if absolute && (parsed.Host == "" || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https")) {
		return nil, snapshotPagerInvalid("Cinder resource pagination URL must be absolute HTTP(S)")
	}
	if !utf8.ValidString(parsed.Path) || strings.Contains(parsed.Path, "\\") {
		return nil, snapshotPagerInvalid("Cinder resource pagination path must be UTF-8 without backslashes")
	}
	for _, character := range parsed.Path {
		if unicode.IsControl(character) {
			return nil, snapshotPagerInvalid("Cinder resource pagination path contains controls")
		}
	}
	for _, segment := range strings.Split(parsed.Path, "/") {
		if segment == "." || segment == ".." {
			return nil, snapshotPagerInvalid("Cinder resource pagination path traverses dot segments")
		}
	}
	return parsed, nil
}

func snapshotPagerInvalid(format string, arguments ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, arguments...))
}
