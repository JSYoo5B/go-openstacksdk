package network

import (
	"net/http"
	"net/url"

	"github.com/gophercloud/gophercloud/v2/pagination"
)

// Selection must inspect later pages even when an intermediate page has no
// rows. Native Pager stops on IsEmpty before reading its next link. Preserve
// its decoding errors and 204 terminal response while following JSON links.
type floatingIPSelectionPage struct {
	pagination.Page
	URL       url.URL
	noContent bool
}

func (p floatingIPSelectionPage) IsEmpty() (bool, error) {
	empty, err := p.Page.IsEmpty()
	return empty && p.noContent, err
}

func floatingIPSelectionPager(pager pagination.Pager, create func(pagination.PageResult) pagination.Page) pagination.Pager {
	wrapped := pager.WithPageCreator(func(r pagination.PageResult) pagination.Page {
		return floatingIPSelectionPage{Page: create(r), URL: r.URL, noContent: r.StatusCode == http.StatusNoContent}
	})
	// WithPageCreator does not copy either field in the pinned native version.
	wrapped.Err, wrapped.Headers = pager.Err, pager.Headers
	return wrapped
}
