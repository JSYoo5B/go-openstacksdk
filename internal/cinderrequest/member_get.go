package cinderrequest

import (
	"context"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// MemberGet retains Cinder's version-owned, bodyless member request policy.
func MemberGet(ctx context.Context, source *cloudread.Source, target, version string, codes ...int) (*rest.Response, error) {
	return microversions.MemberGet(ctx, source, target, version, microversions.CinderProfile, codes...)
}
