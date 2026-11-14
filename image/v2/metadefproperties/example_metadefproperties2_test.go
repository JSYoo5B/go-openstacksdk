package metadefproperties_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func deleteOwnedPropertiesMetadefproperties2(ctx context.Context, scope *metadefproperties.NamespaceScope, id string, clearNamespace bool) (member, collection *metadefproperties.Acknowledgement, err error) {
	member, err = scope.DeleteRecord(ctx, metadefproperties.RecordRequest{ID: id},
		request.IgnoreMissing(false),
		request.Header("X-Request-Source", "owned-delete-example"),
	)
	if err != nil || !clearNamespace {
		return member, nil, err
	}
	collection, err = scope.DeleteAllRecords(ctx,
		request.Header("X-Request-Source", "owned-delete-example"),
	)
	return member, collection, err
}
