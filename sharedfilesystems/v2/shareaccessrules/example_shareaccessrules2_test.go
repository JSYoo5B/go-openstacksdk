package shareaccessrules_test

import (
	"context"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules"
)

func FirstAccessRulesShareaccessrules2(ctx context.Context, conn *sdk.Connection,
	share resource.Ref) ([]*shareaccessrules.AccessRule, error) {
	service, err := conn.SharedFileSystemV2(ctx)
	if err != nil {
		return nil, err
	}
	scope, err := service.ShareAccessRules.InShare(ctx, share)
	if err != nil {
		return nil, err
	}
	return scope.All(ctx,
		request.MaxItems(5),
		request.Paginated(false))
}
