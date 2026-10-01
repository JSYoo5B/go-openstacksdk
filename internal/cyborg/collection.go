// Package cyborg implements transport for the SDK-owned Cyborg resource models.
package cyborg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"gophercloudsdk/accelerator/v2/common"
	"gophercloudsdk/resource"
)

func Collection[T any](client *gophercloud.ServiceClient, path, single, plural string,
	id, name, status func(*T) string, metadata func(*T) *common.Metadata,
	deleteURL func(string) string, validateID ...func(string) error) *resource.Collection[T] {
	adapter := resource.Adapter[T]{Kind: plural, ID: id, Name: name, Status: status,
		Get: func(ctx context.Context, id string) (*T, error) {
			return Fetch[T](ctx, client, "GET", client.ServiceURL(path, url.PathEscape(id)), nil, single, metadata, 200)
		},
		List: func(query url.Values) pagination.Pager {
			guarded, err := guardedClient(client)
			if err != nil {
				return pagination.Pager{Err: err}
			}
			endpoint := client.ServiceURL(path)
			if len(query) > 0 {
				endpoint += "?" + query.Encode()
			}
			return pagination.NewPager(guarded, endpoint, func(result pagination.PageResult) pagination.Page {
				return page[T]{LinkedPageBase: pagination.LinkedPageBase{PageResult: result}, plural: plural, metadata: metadata}
			})
		},
		Extract: func(p pagination.Page) ([]T, error) { return p.(page[T]).extract() },
	}
	if len(validateID) > 0 {
		adapter.ValidateID = validateID[0]
	}
	if deleteURL != nil {
		adapter.Delete = func(ctx context.Context, id string) error {
			guarded, err := guardedClient(client)
			if err != nil {
				return err
			}
			_, err = guarded.Delete(ctx, deleteURL(id), &gophercloud.RequestOpts{OkCodes: []int{204}})
			return err
		}
	}
	if status != nil {
		adapter.Failed = func(value string) bool {
			return strings.EqualFold(value, "error") || strings.EqualFold(value, "bindfailed")
		}
	}
	return resource.NewCollection(adapter)
}

func Fetch[T any](ctx context.Context, client *gophercloud.ServiceClient, method, endpoint string, body any, single string, metadata func(*T) *common.Metadata, codes ...int) (*T, error) {
	return fetch[T](ctx, client, method, endpoint, body, nil, single, metadata, codes...)
}

func fetch[T any](ctx context.Context, client *gophercloud.ServiceClient, method, endpoint string, body any, headers map[string]string, single string, metadata func(*T) *common.Metadata, codes ...int) (*T, error) {
	return DecodeSingleMutation(ctx, client, method, endpoint, body, headers, single, single+"s", metadata, codes...)
}

// JSONResponse retains the entire response for operations that return batches.
// A successful HTTP request can still return a decoding error; metadata survives
// so callers can inspect it without retrying an already accepted mutation.
func JSONResponse(ctx context.Context, client *gophercloud.ServiceClient, method, endpoint string, body any, headers map[string]string, codes ...int) (json.RawMessage, *common.Metadata, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	guarded, err := guardedClient(client)
	if err != nil {
		return nil, nil, err
	}
	var raw json.RawMessage
	response, err := guarded.Request(ctx, method, endpoint, &gophercloud.RequestOpts{JSONBody: body, JSONResponse: &raw, MoreHeaders: headers, OkCodes: codes})
	var meta *common.Metadata
	if response != nil {
		value := ResponseMetadata(response)
		meta = &value
	}
	return raw, meta, err
}

// DecodeSingleMutation accepts a flat object, a singular envelope, or exactly
// one resource in a plural envelope. It never silently chooses a batch member.
func DecodeSingleMutation[T any](ctx context.Context, client *gophercloud.ServiceClient, method, endpoint string, body any, headers map[string]string, single, plural string, metadata func(*T) *common.Metadata, codes ...int) (*T, error) {
	raw, response, err := JSONResponse(ctx, client, method, endpoint, body, headers, codes...)
	if err != nil {
		return nil, err
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '{' {
		return nil, fmt.Errorf("Cyborg singleton response must be an object")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	nested, singular := envelope[single]
	batch, multiple := envelope[plural]
	if singular && multiple {
		return nil, fmt.Errorf("Cyborg response contains both %s and %s", single, plural)
	}
	if singular {
		raw = nested
	} else if multiple {
		batch = bytes.TrimSpace(batch)
		if len(batch) == 0 || batch[0] != '[' {
			return nil, fmt.Errorf("Cyborg response must contain %s array", plural)
		}
		var items []json.RawMessage
		if err := json.Unmarshal(batch, &items); err != nil {
			return nil, err
		}
		if len(items) != 1 {
			return nil, fmt.Errorf("Cyborg singleton response contains %d %s", len(items), plural)
		}
		raw = items[0]
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	meta := metadata(&value)
	meta.Header, meta.StatusCode = response.Header.Clone(), response.StatusCode
	return &value, nil
}

type page[T any] struct {
	pagination.LinkedPageBase
	plural   string
	metadata func(*T) *common.Metadata
}

func (p page[T]) extract() ([]T, error) {
	if p.Err != nil {
		return nil, p.Err
	}
	encoded, err := json.Marshal(p.Body)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	raw, ok := fields[p.plural]
	if !ok || len(raw) == 0 || raw[0] != '[' {
		return nil, fmt.Errorf("Cyborg response must contain %s array", p.plural)
	}
	var values []T
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	for i := range values {
		meta := p.metadata(&values[i])
		meta.Header = p.Header.Clone()
		meta.StatusCode = p.StatusCode
	}
	return values, nil
}

func (p page[T]) IsEmpty() (bool, error) {
	values, err := p.extract()
	if err != nil || len(values) > 0 {
		return false, err
	}
	next, err := p.NextPageURL()
	return next == "", err
}

func (p page[T]) NextPageURL() (string, error) {
	encoded, err := json.Marshal(p.Body)
	if err != nil {
		return "", err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return "", err
	}
	for _, key := range []string{"links", p.plural + "_links"} {
		if raw, ok := fields[key]; ok {
			var links []common.Link
			if err := json.Unmarshal(raw, &links); err != nil {
				return "", err
			}
			for _, link := range links {
				if link.Rel == "next" {
					return p.resolve(link.Href)
				}
			}
		}
	}
	if raw, ok := fields["next"]; ok {
		var next string
		if err := json.Unmarshal(raw, &next); err != nil {
			return "", err
		}
		return p.resolve(next)
	}
	return "", nil
}

func (p page[T]) resolve(next string) (string, error) {
	if next == "" {
		return "", nil
	}
	u, err := url.Parse(next)
	if err != nil {
		return "", err
	}
	u = p.URL.ResolveReference(u)
	if u.User != nil || u.Scheme != p.URL.Scheme || !strings.EqualFold(u.Host, p.URL.Host) {
		return "", fmt.Errorf("Cyborg pagination link changes service origin")
	}
	u.Fragment = ""
	return u.String(), nil
}

// ResponseMetadata snapshots mutation responses without inventing resource data.
func ResponseMetadata(response *http.Response) common.Metadata {
	return common.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
