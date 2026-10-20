package applicationcredentials

import (
	"context"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"iter"
)

func (s *ApplicationCredentialScope) GetRecord(ctx context.Context, id string, options ...RecordReadOption) (*ApplicationCredentialRecord, error) {
	return s.api.GetCredentialRecord(ctx, s.parentID, id, options...)
}
func (s *ApplicationCredentialScope) CreateRecord(ctx context.Context, name string, options ...RecordCreateOption) (*ApplicationCredentialRecord, error) {
	return s.api.CreateCredentialRecord(ctx, s.parentID, name, options...)
}
func (s *ApplicationCredentialScope) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*ApplicationCredentialRecord, error] {
	return s.api.ListCredentialRecords(ctx, s.parentID, options...)
}
func (s *ApplicationCredentialScope) FindRecord(ctx context.Context, nameOrID string, options ...resource.IdentityFindOption) (*ApplicationCredentialRecord, error) {
	return s.api.FindCredentialRecord(ctx, s.parentID, nameOrID, options...)
}
func (s *ApplicationCredentialScope) DeleteRecord(ctx context.Context, id string, options ...RecordDeleteOption) error {
	return s.api.DeleteCredentialRecord(ctx, s.parentID, id, options...)
}
func (s *AccessRuleScope) GetRecord(ctx context.Context, id string, options ...RecordReadOption) (*AccessRuleRecord, error) {
	return s.api.GetAccessRuleRecord(ctx, s.parentID, id, options...)
}
func (s *AccessRuleScope) ListRecords(ctx context.Context, options ...RecordListOption) iter.Seq2[*AccessRuleRecord, error] {
	return s.api.ListAccessRuleRecords(ctx, s.parentID, options...)
}
func (s *AccessRuleScope) DeleteRecord(ctx context.Context, id string, options ...RecordDeleteOption) error {
	return s.api.DeleteAccessRuleRecord(ctx, s.parentID, id, options...)
}
