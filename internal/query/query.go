// Package query adapts optional filters to Gophercloud's builder contracts.
package query

import "net/url"

type Adapter url.Values

func (q Adapter) encode() (string, error) {
	if len(q) == 0 {
		return "", nil
	}
	return "?" + url.Values(q).Encode(), nil
}
func (q Adapter) ToServerListQuery() (string, error)  { return q.encode() }
func (q Adapter) ToFlavorListQuery() (string, error)  { return q.encode() }
func (q Adapter) ToNetworkListQuery() (string, error) { return q.encode() }
func (q Adapter) ToImageListQuery() (string, error)   { return q.encode() }
func (q Adapter) ToVolumeListQuery() (string, error)  { return q.encode() }
