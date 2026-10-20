package users

import (
	"context"
	"fmt"
	"iter"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const userGroupRecordKind = "identity.user_groups"

// UserGroupRecord keeps the actual group response and an independent view bound
// to the requested user. It does not add Group CRUD or resource-routing methods.
type UserGroupRecord struct {
	Resource *resource.RawResource
	Wire     *resource.RawResource
}

// UnmarshalJSON retains arbitrary group fields without native model coercion.
func (value *UserGroupRecord) UnmarshalJSON(data []byte) error {
	if value == nil {
		return fmt.Errorf("%w: user-group receiver is required", resource.ErrInvalidOption)
	}
	if err := unmarshalMembershipRecord(data, &value.Wire); err != nil {
		return err
	}
	value.Resource = nil
	return nil
}

func prepareUserGroupRecord(value *UserProjectRecord, userID string) error {
	view, err := cloneMembershipRecordView(value.Wire, userID)
	if err != nil {
		return err
	}
	value.Resource = view
	return nil
}

// ListGroupRecords lazily lists the groups containing the explicit user ID.
// It starts without query/control options, as does Python user_groups(user),
// and consumes advertised continuation links through the owned read policy.
// Native ListGroups remains available with its original native Group result.
func (a *API) ListGroupRecords(ctx context.Context, userID string) iter.Seq2[*UserGroupRecord, error] {
	sequence := a.listMembershipRecords(ctx, userID, membershipRecordRead{
		operation: "ListGroupRecords", kind: userGroupRecordKind, plural: "groups",
		project: prepareUserGroupRecord,
	})
	return func(yield func(*UserGroupRecord, error) bool) {
		for value, err := range sequence {
			if err != nil {
				yield(nil, err)
				return
			}
			// The owned read runner has already created independent resources;
			// wrapping its fields requires no additional decode, copy or request.
			group := &UserGroupRecord{Resource: value.Resource, Wire: value.Wire}
			if !yield(group, nil) {
				return
			}
		}
	}
}
