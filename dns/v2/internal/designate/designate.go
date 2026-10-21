// Package designate holds the shared transport rules of SDK-owned Designate
// bindings: flat JSON bodies, explicit success codes, links.next paging and
// the documented X-Auth-Sudo-Project-ID / X-Auth-All-Projects headers.
package designate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"net/http"
	"net/url"
	"strconv"

	"github.com/JSYoo5B/go-openstacksdk/internal/fixedrequest"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Evidence retains the exact accepted object and HTTP metadata of a decoded
// model. Body distinguishes omitted, null and extension fields.
type Evidence struct {
	Body       map[string]json.RawMessage `json:"-"`
	Header     http.Header                `json:"-"`
	StatusCode int                        `json:"-"`
}

// Meta lets generic helpers reach the embedded evidence of any model.
func (e *Evidence) Meta() *Evidence { return e }

// Model constrains *T to models embedding Evidence.
type Model[T any] interface {
	*T
	Meta() *Evidence
}

// Headers builds the optional Designate scope headers. A nil allProjects and an
// empty sudo project add nothing; an explicit false is sent as "false".
func Headers(allProjects *bool, sudoProject string) map[string]string {
	headers := map[string]string{}
	if allProjects != nil {
		headers["X-Auth-All-Projects"] = strconv.FormatBool(*allProjects)
	}
	if sudoProject != "" {
		headers["X-Auth-Sudo-Project-ID"] = sudoProject
	}
	if len(headers) == 0 {
		return nil
	}
	return headers
}

// Client validates the service client and context before any request.
func Client(ctx context.Context, client *gophercloud.ServiceClient) error {
	if ctx == nil {
		return fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if client == nil || client.ProviderClient == nil {
		return fmt.Errorf("%w: service client is required", resource.ErrInvalidOption)
	}
	return nil
}

// ValidateID rejects empty and unescaped-path-unsafe identifiers.
func ValidateID(id string) error { return resource.ID(id).Validate() }

// Do sends one JSON request. A nil body sends no body.
func Do(ctx context.Context, client *gophercloud.ServiceClient, method, endpoint string, body any, headers map[string]string, codes ...int) (*rest.Response, error) {
	if err := Client(ctx, client); err != nil {
		return nil, err
	}
	return rest.DoJSON(ctx, client, method, endpoint, body, headers, codes...)
}

// DoRaw sends a non-JSON request body with explicit headers. The body is
// snapshotted, so native reauthentication replays the same bytes.
func DoRaw(ctx context.Context, source *gophercloud.ServiceClient, method, endpoint string, body []byte, headers map[string]string, codes ...int) (*rest.Response, error) {
	if err := Client(ctx, source); err != nil {
		return nil, err
	}
	if len(codes) == 0 {
		return nil, fmt.Errorf("%w: explicit success codes are required", resource.ErrInvalidOption)
	}
	if err := rest.ValidateTarget(source, endpoint); err != nil {
		return nil, err
	}
	client, err := fixedrequest.New(source, method, endpoint)
	if err != nil {
		return nil, err
	}
	opts := &gophercloud.RequestOpts{OkCodes: append([]int(nil), codes...), KeepResponseBody: true, MoreHeaders: headers}
	if body != nil {
		opts.RawBody = bytes.NewReader(append([]byte(nil), body...))
	}
	wire, err := client.Request(ctx, method, endpoint, opts)
	if err != nil {
		return nil, err
	}
	response := &rest.Response{Header: wire.Header.Clone(), StatusCode: wire.StatusCode}
	var readErr error
	response.Body, readErr = io.ReadAll(wire.Body)
	closeErr := wire.Body.Close()
	if readErr == nil {
		readErr = closeErr
	}
	if readErr == nil {
		readErr = ctx.Err()
	}
	if readErr != nil {
		return response, response.Fail(readErr)
	}
	return response, nil
}

// Decode decodes one accepted flat JSON object into T and retains evidence.
func Decode[T any, P Model[T]](response *rest.Response) (P, error) {
	return decodeObject[T, P](response.Body, response)
}

func decodeObject[T any, P Model[T]](data []byte, response *rest.Response) (P, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return nil, response.Fail(fmt.Errorf("Designate response must be a JSON object"))
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, response.Fail(err)
	}
	value := new(T)
	if err := json.Unmarshal(data, value); err != nil {
		return nil, response.Fail(err)
	}
	meta := P(value).Meta()
	meta.Body, meta.Header, meta.StatusCode = fields, response.Header.Clone(), response.StatusCode
	return value, nil
}

// ListSpec names a fixed collection route and its plural envelope key.
type ListSpec struct {
	Path      []string
	PluralKey string
	// IDKey is the row field used as the marker fallback. It defaults to id.
	IDKey string
}

// List streams every page. It follows an advertised links.next that stays on
// the service origin; without one, an explicit positive wire limit continues
// with marker=<last row id> until an empty page, like the pinned Python
// Resource.list. metadata.total_count stops once that many rows were read.
// MaxItems caps raw rows; SinglePage stops after the first page.
func List[T any, P Model[T]](ctx context.Context, client *gophercloud.ServiceClient, spec ListSpec, query url.Values, headers map[string]string, control resource.ListControl) iter.Seq2[P, error] {
	return func(yield func(P, error) bool) {
		fail := func(err error) { yield(nil, err) }
		if control.MaxItems < 0 {
			fail(fmt.Errorf("%w: maximum items must be non-negative", resource.ErrInvalidOption))
			return
		}
		if err := Client(ctx, client); err != nil {
			fail(err)
			return
		}
		pageQuery := url.Values{}
		for key, values := range query {
			pageQuery[key] = append([]string(nil), values...)
		}
		limit := 0
		if raw := pageQuery.Get("limit"); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n <= 0 {
				fail(fmt.Errorf("%w: limit must be a positive integer", resource.ErrInvalidOption))
				return
			}
			limit = n
		}
		endpoint := client.ServiceURL(spec.Path...)
		seen := map[string]bool{}
		yielded, rows := 0, 0
		lastMarker := pageQuery.Get("marker")
		for {
			target := endpoint
			if len(pageQuery) > 0 {
				target += "?" + pageQuery.Encode()
			}
			if seen[target] {
				fail(fmt.Errorf("Designate pagination loop at %s", target))
				return
			}
			seen[target] = true
			response, err := Do(ctx, client, http.MethodGet, target, nil, headers, http.StatusOK)
			if err != nil {
				fail(err)
				return
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(response.Body, &fields); err != nil || fields == nil {
				if err == nil {
					err = fmt.Errorf("list response must be a JSON object")
				}
				fail(response.Fail(err))
				return
			}
			var items []json.RawMessage
			raw := bytes.TrimSpace(fields[spec.PluralKey])
			if len(raw) == 0 || raw[0] != '[' || json.Unmarshal(raw, &items) != nil {
				fail(response.Fail(fmt.Errorf("list response requires %q array", spec.PluralKey)))
				return
			}
			marker := ""
			for _, item := range items {
				if control.MaxItems > 0 && yielded >= control.MaxItems {
					return
				}
				value, err := decodeObject[T, P](item, response)
				if err != nil {
					fail(err)
					return
				}
				yielded++
				rows++
				if id := rowID(P(value).Meta().Body, spec.IDKey); id != "" {
					marker = id
				}
				if !yield(value, nil) {
					return
				}
			}
			if len(items) == 0 || control.SinglePage || (control.MaxItems > 0 && yielded >= control.MaxItems) {
				return
			}
			var meta struct {
				TotalCount json.Number `json:"total_count"`
			}
			if raw := fields["metadata"]; len(raw) > 0 {
				_ = json.Unmarshal(raw, &meta)
			}
			if total, err := strconv.Atoi(meta.TotalCount.String()); err == nil && total > 0 && total <= rows {
				return
			}
			next, err := nextLink(fields["links"])
			if err != nil {
				fail(response.Fail(err))
				return
			}
			switch {
			case next != "":
				if err := rest.ValidateTarget(client, next); err != nil {
					fail(response.Fail(err))
					return
				}
				parsed, _ := url.Parse(next)
				pageQuery = parsed.Query()
				parsed.RawQuery = ""
				endpoint = parsed.String()
			case limit > 0 && marker != "":
				if marker == lastMarker {
					fail(response.Fail(fmt.Errorf("Endless pagination loop detected, aborting")))
					return
				}
				lastMarker = marker
				pageQuery.Set("marker", marker)
			default:
				return
			}
		}
	}
}

func rowID(body map[string]json.RawMessage, key string) string {
	if key == "" {
		key = "id"
	}
	var id string
	if json.Unmarshal(body[key], &id) != nil {
		return ""
	}
	return id
}

func nextLink(raw json.RawMessage) (string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return "", nil
	}
	var links map[string]json.RawMessage
	if json.Unmarshal(raw, &links) != nil {
		return "", fmt.Errorf("links must be a JSON object")
	}
	next := bytes.TrimSpace(links["next"])
	if len(next) == 0 || bytes.Equal(next, []byte("null")) {
		return "", nil
	}
	var value string
	if json.Unmarshal(next, &value) != nil {
		return "", fmt.Errorf("links.next must be a string")
	}
	return value, nil
}
