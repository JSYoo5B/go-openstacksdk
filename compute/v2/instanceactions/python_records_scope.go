package instanceactions

import (
	"context"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"iter"
)

// ListRecords reuses the server already resolved by InServer, preserving its
// native Get/List ABI while adding the owned Source history view.
func (s *ActionScope) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*ServerActionRecord, error] {
	if s == nil || s.api == nil {
		return func(yield func(*ServerActionRecord, error) bool) {
			yield(nil, request.Wrap("ListRecords", "server actions", findInvalid("action scope is required")))
		}
	}
	return s.api.ListRecords(ctx, s.serverID, options...)
}
func (s *ActionScope) ProxyListRecords(ctx context.Context, options ...RecordListOption) (*RecordProxyListResult, error) {
	if s == nil || s.api == nil {
		return nil, request.Wrap("ProxyListRecords", "server actions", findInvalid("action scope is required"))
	}
	return s.api.ProxyListRecords(ctx, s.serverID, options...)
}
