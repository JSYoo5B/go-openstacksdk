// Package octaviarest holds the wire helpers shared by SDK-owned Octavia
// bindings whose routes Gophercloud v2.15.0 does not provide.
package octaviarest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Wire retains one response's raw body, headers and status.
type Wire struct {
	Body   []byte
	Header http.Header
	Status int
}

// Endpoint builds an /lbaas route, respecting a client base already ending in lbaas.
func Endpoint(client *gophercloud.ServiceClient, parts ...string) string {
	if strings.HasSuffix(strings.TrimSuffix(client.ServiceURL(), "/"), "/lbaas") {
		return client.ServiceURL(parts...)
	}
	return client.ServiceURL(append([]string{"lbaas"}, parts...)...)
}

// Do sends one request exactly once. Mutations are never replayed.
func Do(ctx context.Context, client *gophercloud.ServiceClient, method, endpoint string, body any, accepted []int) (Wire, error) {
	if client == nil || ctx == nil {
		return Wire{}, fmt.Errorf("%w: client and context are required", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return Wire{}, err
	}
	options := &gophercloud.RequestOpts{OkCodes: accepted, KeepResponseBody: true}
	if body != nil {
		options.JSONBody = body
	}
	response, err := client.Request(ctx, method, endpoint, options)
	if err != nil {
		return Wire{}, err
	}
	defer response.Body.Close()
	wire := Wire{Header: response.Header.Clone(), Status: response.StatusCode}
	if wire.Body, err = io.ReadAll(response.Body); err != nil {
		return wire, err
	}
	if err := ctx.Err(); err != nil {
		return wire, err
	}
	return wire, nil
}

// Object extracts the JSON object stored under key.
func Object(body []byte, key string) (json.RawMessage, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	raw, ok := envelope[key]
	if !ok || strings.TrimSpace(string(raw)) == "null" {
		return nil, fmt.Errorf("response requires a %q object", key)
	}
	return raw, nil
}

// Row is one list entry with its page's response metadata.
type Row struct {
	Raw    json.RawMessage
	Header http.Header
	Status int
}

// Pages follows <key>_links next links from first, never leaving the origin
// and failing on a repeated page.
func Pages(ctx context.Context, client *gophercloud.ServiceClient, first, key string, singlePage bool) iter.Seq2[Row, error] {
	return func(yield func(Row, error) bool) {
		origin, err := url.Parse(first)
		if err != nil {
			yield(Row{}, err)
			return
		}
		next := first
		visited := map[string]bool{}
		for next != "" {
			if visited[next] {
				yield(Row{}, &resource.PaginationCycleError{URL: next})
				return
			}
			visited[next] = true
			wire, err := Do(ctx, client, http.MethodGet, next, nil, []int{200})
			if err != nil {
				yield(Row{}, err)
				return
			}
			var envelope map[string]json.RawMessage
			if err := json.Unmarshal(wire.Body, &envelope); err != nil {
				yield(Row{}, err)
				return
			}
			array, ok := envelope[key]
			if !ok || strings.TrimSpace(string(array)) == "null" {
				yield(Row{}, fmt.Errorf("list response requires a %q array", key))
				return
			}
			var rows []json.RawMessage
			if err := json.Unmarshal(array, &rows); err != nil {
				yield(Row{}, err)
				return
			}
			for _, raw := range rows {
				if err := ctx.Err(); err != nil {
					yield(Row{}, err)
					return
				}
				if !yield(Row{Raw: raw, Header: wire.Header.Clone(), Status: wire.Status}, nil) {
					return
				}
			}
			if singlePage {
				return
			}
			if next, err = nextLink(envelope[key+"_links"], next, origin); err != nil {
				yield(Row{}, err)
				return
			}
		}
	}
}

func nextLink(raw json.RawMessage, current string, origin *url.URL) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	var links []struct{ Href, Rel string }
	if err := json.Unmarshal(raw, &links); err != nil {
		return "", err
	}
	base, err := url.Parse(current)
	if err != nil {
		return "", err
	}
	next := ""
	for _, link := range links {
		if link.Rel != "next" || link.Href == "" {
			continue
		}
		candidate, err := url.Parse(link.Href)
		if err != nil {
			return "", err
		}
		candidate = base.ResolveReference(candidate)
		if candidate.User != nil || !strings.EqualFold(candidate.Scheme, origin.Scheme) || !strings.EqualFold(candidate.Host, origin.Host) {
			return "", fmt.Errorf("%w: pagination link changes origin", resource.ErrUnsupported)
		}
		candidate.Fragment = ""
		query := candidate.Query()
		for key, values := range origin.Query() {
			if _, present := query[key]; !present {
				query[key] = append([]string(nil), values...)
			}
		}
		candidate.RawQuery = query.Encode()
		if next != "" && next != candidate.String() {
			return "", fmt.Errorf("%w: page has multiple next links", resource.ErrInvalidOption)
		}
		next = candidate.String()
	}
	return next, nil
}
