package stacks

import (
	"net/url"
	"strconv"

	"github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/orchestration/v1/stacks"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

// The pinned native StackPage is a SinglePageBase. Heat/Python list responses
// can contain top-level next links; the SDK owns a linked page for its resources.
type resourcePage struct{ pagination.LinkedPageBase }

func (page resourcePage) IsEmpty() (bool, error) {
	return (upstream.StackPage{SinglePageBase: pagination.SinglePageBase(page.PageResult)}).IsEmpty()
}

func (page resourcePage) NextPageURL() (string, error) {
	var wire struct {
		Links []gophercloud.Link `json:"links"`
	}
	if err := page.ExtractInto(&wire); err != nil {
		return "", err
	}
	next, err := gophercloud.ExtractNextURL(wire.Links)
	if err != nil {
		return next, err
	}
	if next == "" {
		// Python Resource also advances Heat's declared marker/limit query
		// when no link is supplied. Continue until an empty page; the common
		// guard rejects an unchanged last marker before fetching it twice.
		limit, err := strconv.Atoi(page.URL.Query().Get("limit"))
		if err != nil || limit <= 0 {
			return "", nil
		}
		values, err := extractResources(page)
		if err != nil || len(values) == 0 {
			return "", err
		}
		endpoint := page.URL
		query := endpoint.Query()
		query.Set("marker", values[len(values)-1].ID)
		endpoint.RawQuery = query.Encode()
		return endpoint.String(), nil
	}
	reference, err := url.Parse(next)
	if err != nil {
		return "", err
	}
	return page.URL.ResolveReference(reference).String(), nil
}

func resourcePager(client *gophercloud.ServiceClient, query url.Values) pagination.Pager {
	endpoint := client.ServiceURL("stacks")
	if len(query) != 0 {
		endpoint += "?" + query.Encode()
	}
	return pagination.NewPager(client, endpoint, func(result pagination.PageResult) pagination.Page {
		return resourcePage{pagination.LinkedPageBase{PageResult: result}}
	})
}

func extractResources(page pagination.Page) ([]StackResource, error) {
	native := upstream.StackPage{SinglePageBase: pagination.SinglePageBase(page.(resourcePage).PageResult)}
	listed, err := upstream.ExtractStacks(native)
	if err != nil {
		return nil, err
	}
	values := make([]StackResource, len(listed))
	for i, value := range listed {
		values[i] = StackResource{RetrievedStack: RetrievedStack{
			CreationTime: value.CreationTime, Description: value.Description,
			ID: value.ID, Links: value.Links, Name: value.Name,
			Status: value.Status, StatusReason: value.StatusReason,
			Tags: value.Tags, UpdatedTime: value.UpdatedTime,
		}}
		if _, err := values[i].Identity(); err != nil {
			return nil, err
		}
	}
	return values, nil
}
